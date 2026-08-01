package ops

import (
	"errors"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// errCipherSaberTestIV is returned by the pinned IV source to exercise the
// encrypt op's IV-generation error path.
var errCipherSaberTestIV = errors.New("iv failure")

// csFixtureCipher is the CipherSaber2 decrypt fixture's ciphertext (10-byte IV
// followed by 30 bytes of ciphertext), taken from https://ciphersaber.gurus.org/.
const csFixtureCipher = "\x6f\x6d\x0b\xab\xf3\xaa\x67\x19\x03\x15\x30\xed\xb6\x77" +
	"\xca\x74\xe0\x08\x9d\xd0\xe7\xb8\x85\x43\x56\xbb\x14\x48\xe3" +
	"\x7c\xdb\xef\xe7\xf3\xa8\x4f\x4f\x5f\xb3\xfd"

// TestCipherSaber2Decrypt transcribes the authoritative CyberChef decrypt
// fixture.
func TestCipherSaber2Decrypt(t *testing.T) {
	runCases(t, []opCase{
		{
			"CipherSaber2 Decrypt", csFixtureCipher, "This is a test of CipherSaber.",
			core.Recipe{{Op: "CipherSaber2 Decrypt", Args: []any{aesTS("Latin1", "asdfg"), 1}}},
		},
	})
}

// TestCipherSaber2Encrypt is the decrypt fixture in reverse: encrypting the
// plaintext with the same key/rounds and the fixture's IV must reproduce the
// exact ciphertext. The random IV source is pinned for the duration of the test.
func TestCipherSaber2Encrypt(t *testing.T) {
	old := cipherSaberRandIV
	cipherSaberRandIV = func(b []byte) error {
		copy(b, []byte{0x6f, 0x6d, 0x0b, 0xab, 0xf3, 0xaa, 0x67, 0x19, 0x03, 0x15})
		return nil
	}
	t.Cleanup(func() { cipherSaberRandIV = old })

	runCases(t, []opCase{
		{
			"CipherSaber2 Encrypt", "This is a test of CipherSaber.", csFixtureCipher,
			core.Recipe{{Op: "CipherSaber2 Encrypt", Args: []any{aesTS("Latin1", "asdfg"), 1}}},
		},
	})
}

// TestCipherSaber2EncryptLength covers the fixture length checks: the output is
// the input plus a 10-byte IV, using the real random source.
func TestCipherSaber2EncryptLength(t *testing.T) {
	cases := []struct {
		name  string
		input string
		key   core.ToggleString
		want  int
	}{
		{"Hello World", "Hello World", aesTS("Latin1", "test"), 21},
		{"empty", "", aesTS("Latin1", ""), 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runOp(t, "CipherSaber2 Encrypt", c.input, c.key, 20)
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			if len(out) != c.want {
				t.Fatalf("output length = %d, want %d", len(out), c.want)
			}
		})
	}
}

// TestCipherSaber2DecryptEmpty covers the empty-key/empty-IV guard: decrypting
// empty input with an empty key yields empty output rather than a modulo panic.
func TestCipherSaber2DecryptEmpty(t *testing.T) {
	out, err := runOp(t, "CipherSaber2 Decrypt", "", aesTS("Latin1", ""), 20)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got %q", out)
	}
}

// TestCipherSaber2KeyErrors covers the key byte-conversion error path on both
// operations, reached with malformed Base64.
func TestCipherSaber2KeyErrors(t *testing.T) {
	if _, err := runOp(t, "CipherSaber2 Encrypt", "x", aesTS("Base64", "!!!"), 20); err == nil {
		t.Fatal("expected encrypt key error, got none")
	}
	if _, err := runOp(t, "CipherSaber2 Decrypt", "x", aesTS("Base64", "!!!"), 20); err == nil {
		t.Fatal("expected decrypt key error, got none")
	}
}

// TestCipherSaber2IVError covers the random-IV failure path via the seam.
func TestCipherSaber2IVError(t *testing.T) {
	old := cipherSaberRandIV
	cipherSaberRandIV = func(b []byte) error { return errCipherSaberTestIV }
	t.Cleanup(func() { cipherSaberRandIV = old })

	if _, err := runOp(t, "CipherSaber2 Encrypt", "x", aesTS("Latin1", "k"), 20); err == nil {
		t.Fatal("expected IV-generation error, got none")
	}
}

// TestCipherSaber2RoundTrip confirms encrypt then decrypt recovers the input
// with a real random IV, across several keys and round counts.
func TestCipherSaber2RoundTrip(t *testing.T) {
	cases := []struct {
		key    core.ToggleString
		rounds int
		text   string
	}{
		{aesTS("UTF8", "secret"), 20, "The quick brown fox."},
		{aesTS("Hex", "00ff10"), 1, "rounds=1 is classic CipherSaber"},
		{aesTS("Latin1", "k"), 5, ""},
	}
	for _, c := range cases {
		enc, err := runOp(t, "CipherSaber2 Encrypt", c.text, c.key, c.rounds)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		dec, err := runOp(t, "CipherSaber2 Decrypt", enc, c.key, c.rounds)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if dec != c.text {
			t.Fatalf("round trip: got %q, want %q", dec, c.text)
		}
	}
}
