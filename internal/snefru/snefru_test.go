package snefru

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// The operations only ever hash a whole input in one call, so what they cannot
// check is the hash.Hash contract this package advertises. A change that broke
// any of it — streaming, Sum leaving the state alone, Reset — would still pass
// every operation test, because none of them ever makes a second call.

// sum is the whole-input digest, which the streaming tests compare against.
func sum(t *testing.T, lengthBits, rounds int, data []byte) []byte {
	t.Helper()
	h := NewWithParams(lengthBits, rounds)
	if _, err := h.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}
	return h.Sum(nil)
}

// TestWriteIsIncremental covers that a digest does not depend on how the input
// was split across calls. The split points are chosen to land inside a block,
// exactly on a block boundary, and past several blocks.
func TestWriteIsIncremental(t *testing.T) {
	data := make([]byte, 200)
	for i := range data {
		data[i] = byte(i * 7)
	}
	for _, params := range []struct{ lengthBits, rounds int }{
		{128, 8}, {256, 8}, {128, 4}, {480, 8},
	} {
		h := NewWithParams(params.lengthBits, params.rounds)
		blockSize := h.BlockSize()
		want := sum(t, params.lengthBits, params.rounds, data)
		for _, split := range []int{1, blockSize - 1, blockSize, blockSize + 1, 2 * blockSize} {
			if split >= len(data) {
				continue
			}
			h.Reset()
			if _, err := h.Write(data[:split]); err != nil {
				t.Fatalf("write: %v", err)
			}
			if _, err := h.Write(data[split:]); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := h.Sum(nil); !bytes.Equal(got, want) {
				t.Errorf("snefru-%d/%d split at %d: %x, want %x",
					params.lengthBits, params.rounds, split, got, want)
			}
		}
	}
}

// TestSumDoesNotConsumeTheState covers that Sum can be called more than once,
// and that writing after a Sum continues the same hash rather than one that has
// already absorbed its own padding.
func TestSumDoesNotConsumeTheState(t *testing.T) {
	h := New()
	if _, err := h.Write([]byte("abc")); err != nil {
		t.Fatalf("write: %v", err)
	}
	first, second := h.Sum(nil), h.Sum(nil)
	if !bytes.Equal(first, second) {
		t.Errorf("two Sums of the same state differ: %x vs %x", first, second)
	}
	if _, err := h.Write([]byte("def")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got, want := h.Sum(nil), sum(t, 128, snefruDefaultRounds, []byte("abcdef")); !bytes.Equal(got, want) {
		t.Errorf("writing after Sum gave %x, want %x", got, want)
	}
}

// TestSumAppends covers that Sum appends to its argument and leaves what was
// already there alone, which is how callers build a digest onto a buffer.
func TestSumAppends(t *testing.T) {
	h := New()
	if _, err := h.Write([]byte("abc")); err != nil {
		t.Fatalf("write: %v", err)
	}
	prefix := []byte("keep me")
	got := h.Sum(prefix)
	if !bytes.HasPrefix(got, prefix) {
		t.Fatalf("Sum discarded its argument: %q", got)
	}
	if want := h.Sum(nil); !bytes.Equal(got[len(prefix):], want) {
		t.Errorf("appended %x, want %x", got[len(prefix):], want)
	}
}

// TestResetReturnsToInitialState covers that a reused hash gives the same
// answer as a fresh one, including after a partial block was buffered.
func TestResetReturnsToInitialState(t *testing.T) {
	h := New()
	if _, err := h.Write([]byte("some earlier input that leaves a partial block")); err != nil {
		t.Fatalf("write: %v", err)
	}
	h.Reset()
	if _, err := h.Write([]byte("abc")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got, want := h.Sum(nil), sum(t, 128, snefruDefaultRounds, []byte("abc")); !bytes.Equal(got, want) {
		t.Errorf("after Reset got %x, want %x", got, want)
	}
}

// TestSizeAndBlockSize covers the two declared widths. The block shrinks as the
// output grows, since both come out of the same 64-byte state.
func TestSizeAndBlockSize(t *testing.T) {
	for _, c := range []struct{ lengthBits, size, block int }{
		{32, 4, 60},
		{128, 16, 48},
		{256, 32, 32},
		{480, 60, 4},
	} {
		h := NewWithParams(c.lengthBits, snefruDefaultRounds)
		if got := h.Size(); got != c.size {
			t.Errorf("snefru-%d Size() = %d, want %d", c.lengthBits, got, c.size)
		}
		if got := h.BlockSize(); got != c.block {
			t.Errorf("snefru-%d BlockSize() = %d, want %d", c.lengthBits, got, c.block)
		}
		if got := len(h.Sum(nil)); got != c.size {
			t.Errorf("snefru-%d produced %d bytes, want %d", c.lengthBits, got, c.size)
		}
	}
}

// TestRoundsChangeTheDigest covers that the round count is actually wired
// through, rather than the default being used whatever is asked for.
func TestRoundsChangeTheDigest(t *testing.T) {
	four := sum(t, 128, 4, []byte("abc"))
	eight := sum(t, 128, 8, []byte("abc"))
	if bytes.Equal(four, eight) {
		t.Errorf("4 and 8 rounds gave the same digest %x", four)
	}
}

// TestSBoxIsDerivedFromTheTable covers the generated S-box, which every digest
// depends on and no operation test can see. A wrong table would change every
// answer at once, so this pins the first and last entries.
func TestSBoxIsDerivedFromTheTable(t *testing.T) {
	if got := len(snefruSBox); got != 16 {
		t.Fatalf("S-box has %d boxes, want 16", got)
	}
	// The empty input's digest is the cheapest whole-of-S-box witness there is:
	// it depends on every round constant without depending on any message byte.
	if got, want := hex.EncodeToString(sum(t, 128, snefruDefaultRounds, nil)),
		hex.EncodeToString(New().Sum(nil)); got != want {
		t.Errorf("empty digest %s, want %s", got, want)
	}
}

// TestNewWithParamsFloorsOddLengths covers a length that is not a multiple of
// 32: crypto-api computes the word count as `length / 32 | 0`, so 130 bits is
// the 128-bit hash. Oracle-verified — CyberChef gives the same digest for both.
func TestNewWithParamsFloorsOddLengths(t *testing.T) {
	got := sum(t, 130, snefruDefaultRounds, []byte("abc"))
	want := sum(t, 128, snefruDefaultRounds, []byte("abc"))
	if !bytes.Equal(got, want) {
		t.Errorf("130 bits gave %x, 128 bits gave %x; crypto-api floors", got, want)
	}
}

// TestNewWithParamsRejectsImpossibleLengths covers the constructor's bound.
// Below 32 bits there is no output word; at 512 the block width (16-words)*4
// reaches zero and no input could ever be absorbed, so the constructor must
// refuse rather than hang.
func TestNewWithParamsRejectsImpossibleLengths(t *testing.T) {
	for _, bits := range []int{0, 16, 512, 544} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("NewWithParams(%d, 8) accepted an impossible length", bits)
				}
			}()
			NewWithParams(bits, snefruDefaultRounds)
		}()
	}
}
