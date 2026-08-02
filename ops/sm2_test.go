package ops

import (
	"io"
	"math/big"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/roberson-io/cchef/core"
)

// Fixtures transcribed verbatim from CyberChef's tests/operations/tests/SM2.mjs.
const (
	sm2SmallPlain = "I am a small plaintext"
	sm2LargePlain = "I am a larger plaintext, that will require the encryption KDF to generate a much larger key to properly encrypt me"

	sm2PublicX = "f7d903cab7925066c31150a92b31e548e63f954f92d01eaa0271fb2a336baef8"
	sm2PublicY = "fb0c45e410ef7a6cdae724e6a78dbff52562e97ede009e762b667d9b14adea6c"
	sm2Private = "e74a72505084c3269aa9b696d603e3e08c74c6740212c11a31e26cdfe08bdf6a"
	sm2Curve   = "sm2p256v1"

	sm2Cipher1 = "9a31bc0adb4677cdc4141479e3949572a55c3e6fb52094721f741c2bd2e179aaa87be6263bc1be602e473be3d5de5dce97f8248948b3a7e15f9f67f64aef21575e0c05e6171870a10ff9ab778dbef24267ad90e1a9d47d68f757d57c4816612e9829f804025dea05a511cda39371c22a2828f976f72e"
	sm2Cipher2 = "d3647d68568a2e7a4f8e843286be7bf2b4d80256697d19a73df306ae1a7e6d0364d942e23d2340606e7a2502a838b132f9242587b2ea7e4c207e87242eea8cae68f5ff4da2a95a7f6d350608ae5b6777e1d925bf9c560087af84aba7befba713130106ddb4082d803811bca3864594722f3198d58257fe4ba37f4aa540adf4cb0568bddd2d8140ad3030deea0a87e3198655cc4d22bfc3d73b1c4afec2ff15d68c8d1298d97132cace922ee8a4e41ca288a7e748b77ca94aa81dc283439923ae7939e00898e16fe5111fbe1d928d152b216a"
	sm2Cipher3 = "5f340eeb4398fa8950ee3408d0e3fe34bf7728c9fdb060c94b916891b5c693610274160b52a7132a2bf16ad5cdb57d1e00da2f3ddbd55350729aa9c268b53e40c05ccce9912daa14406e8c132e389484e69757350be25351755dcc6c25c94b3c1a448b2cf8c2017582125eb6cf782055b199a875e966"
	sm2Cipher4 = "0649bac46c3f9fd7fb3b2be4bff27414d634651efd02ca67d8c802bbc5468e77d035c39b581d6b56227f5d87c0b4efbea5032c0761139295ae194b9f1fce698f2f4b51d89fa5554171a1aad2e61fe9de89831aec472ecc5ab178ebf4d2230c1fb94fca03e536b87b9eba6db71ba9939260a08ffd230ca86cb45cf754854222364231bdb8b873791d63ad57a4b3fa5b6375388dc879373f5f1be9051bc5072a8afbec5b7b034e4907aa5bb4b6b1f50e725d09cb6a02e07ce20263005f6c9157ce05d3ea739d231d4f09396fb72aa680884d78"
)

func sm2DecryptRecipe(inputFormat string) core.Recipe {
	return core.Recipe{{Op: "SM2 Decrypt", Args: []any{sm2Private, inputFormat, sm2Curve}}}
}

func TestSM2DecryptFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"SM2 Decrypt: Small Input; Format One", sm2Cipher1, sm2SmallPlain, sm2DecryptRecipe("C1C3C2")},
		{"SM2 Decrypt: Large Input; Format One", sm2Cipher2, sm2LargePlain, sm2DecryptRecipe("C1C3C2")},
		{"SM2 Decrypt: Small Input; Format Two", sm2Cipher3, sm2SmallPlain, sm2DecryptRecipe("C1C2C3")},
		{"SM2 Decrypt: Large Input; Format Two", sm2Cipher4, sm2LargePlain, sm2DecryptRecipe("C1C2C3")},
	})
}

// Encryption is randomised, so it is verified by decrypting the result and
// checking the original plaintext is recovered (mirroring CyberChef's own
// round-trip fixtures).
func TestSM2RoundTrip(t *testing.T) {
	runCases(t, []opCase{
		{"SM2 Encrypt And Decrypt: Small Input; Format One", sm2SmallPlain, sm2SmallPlain, sm2RoundTripRecipe("C1C3C2")},
		{"SM2 Encrypt And Decrypt: Large Input; Format One", sm2LargePlain, sm2LargePlain, sm2RoundTripRecipe("C1C3C2")},
		{"SM2 Encrypt And Decrypt: Small Input; Format Two", sm2SmallPlain, sm2SmallPlain, sm2RoundTripRecipe("C1C2C3")},
		{"SM2 Encrypt And Decrypt: Large Input; Format Two", sm2LargePlain, sm2LargePlain, sm2RoundTripRecipe("C1C2C3")},
	})
}

func sm2RoundTripRecipe(format string) core.Recipe {
	return core.Recipe{
		{Op: "SM2 Encrypt", Args: []any{sm2PublicX, sm2PublicY, format, sm2Curve}},
		{Op: "SM2 Decrypt", Args: []any{sm2Private, format, sm2Curve}},
	}
}

func TestSM2Errors(t *testing.T) {
	t.Run("encrypt bad public key X", func(t *testing.T) {
		_, err := runOp(t, "SM2 Encrypt", "hello", "DEADBEEF", sm2PublicY, "C1C3C2", sm2Curve)
		want := "Invalid Public Key - Ensure each component is 32 bytes in size and in hex"
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
	t.Run("encrypt bad public key Y", func(t *testing.T) {
		_, err := runOp(t, "SM2 Encrypt", "hello", sm2PublicX, "DEADBEEF", "C1C3C2", sm2Curve)
		want := "Invalid Public Key - Ensure each component is 32 bytes in size and in hex"
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
	t.Run("decrypt bad private key", func(t *testing.T) {
		_, err := runOp(t, "SM2 Decrypt", sm2Cipher1, "DEADBEEF", "C1C3C2", sm2Curve)
		want := "Input private key must be in hex; and should be 32 bytes"
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
	t.Run("decrypt hash mismatch", func(t *testing.T) {
		// Corrupt the final byte of the C2 section so the recomputed C3 differs.
		bad := sm2Cipher1[:len(sm2Cipher1)-2] + "00"
		if bad == sm2Cipher1 {
			bad = sm2Cipher1[:len(sm2Cipher1)-2] + "11"
		}
		_, err := runOp(t, "SM2 Decrypt", bad, sm2Private, "C1C3C2", sm2Curve)
		want := "Decryption Error -- Computed Hashes Do Not Match"
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
}

// The malformed-input guards of both operations, exercised directly.
func TestSM2Guards(t *testing.T) {
	invalidPub := "Invalid Public Key - Ensure each component is 32 bytes in size and in hex"
	invalidPriv := "Input private key must be in hex; and should be 32 bytes"
	hashErr := "Decryption Error -- Computed Hashes Do Not Match"
	nonHex64 := strings.Repeat("zz", 32) // length 64, not hex
	zeros64 := strings.Repeat("0", 64)

	t.Run("encrypt non-hex public key", func(t *testing.T) {
		if _, err := runOp(t, "SM2 Encrypt", "hi", nonHex64, sm2PublicY, "C1C3C2", sm2Curve); err == nil || err.Error() != invalidPub {
			t.Fatalf("got %v, want %q", err, invalidPub)
		}
	})
	t.Run("encrypt entropy failure", func(t *testing.T) {
		orig := sm2RandReader
		sm2RandReader = iotest.ErrReader(io.ErrUnexpectedEOF)
		defer func() { sm2RandReader = orig }()
		if _, err := runOp(t, "SM2 Encrypt", "hi", sm2PublicX, sm2PublicY, "C1C3C2", sm2Curve); err == nil {
			t.Fatal("want error from failing entropy source")
		}
	})
	t.Run("decrypt non-hex private key", func(t *testing.T) {
		if _, err := runOp(t, "SM2 Decrypt", sm2Cipher1, nonHex64, "C1C3C2", sm2Curve); err == nil || err.Error() != invalidPriv {
			t.Fatalf("got %v, want %q", err, invalidPriv)
		}
	})
	t.Run("decrypt non-hex C1 point", func(t *testing.T) {
		bad := strings.Repeat("zz", 96) + sm2Cipher1[192:]
		if _, err := runOp(t, "SM2 Decrypt", bad, sm2Private, "C1C3C2", sm2Curve); err == nil || err.Error() != hashErr {
			t.Fatalf("got %v, want %q", err, hashErr)
		}
	})
	t.Run("decrypt odd-length C2", func(t *testing.T) {
		bad := sm2Cipher1 + "a" // makes the C2 hex odd-length
		if _, err := runOp(t, "SM2 Decrypt", bad, sm2Private, "C1C3C2", sm2Curve); err == nil || err.Error() != hashErr {
			t.Fatalf("got %v, want %q", err, hashErr)
		}
	})
	t.Run("decrypt zero private key gives infinity", func(t *testing.T) {
		if _, err := runOp(t, "SM2 Decrypt", sm2Cipher1, zeros64, "C1C3C2", sm2Curve); err == nil || err.Error() != hashErr {
			t.Fatalf("got %v, want %q", err, hashErr)
		}
	})
	t.Run("decrypt short input", func(t *testing.T) {
		if _, err := runOp(t, "SM2 Decrypt", "1234", sm2Private, "C1C2C3", sm2Curve); err == nil {
			t.Fatal("want error for short input")
		}
	})
}

// Elliptic-curve arithmetic edge cases (adding a point to itself, to its
// negation, and doubling degenerate points), plus the curve-constant guard.
func TestSM2PointMath(t *testing.T) {
	g := sm2Point{sm2Gx, sm2Gy}
	dbl := sm2Double(g)

	if got := sm2Add(g, g); got.x.Cmp(dbl.x) != 0 || got.y.Cmp(dbl.y) != 0 {
		t.Fatal("P + P should equal 2P")
	}
	negG := sm2Point{new(big.Int).Set(sm2Gx), new(big.Int).Sub(sm2P, sm2Gy)}
	if got := sm2Add(g, negG); !sm2IsInf(got) {
		t.Fatal("P + (-P) should be the point at infinity")
	}
	if got := sm2Add(g, dbl); sm2IsInf(got) {
		t.Fatal("P + 2P should be a finite point")
	}
	if got := sm2Add(g, sm2Point{}); got.x.Cmp(g.x) != 0 || got.y.Cmp(g.y) != 0 {
		t.Fatal("P + infinity should be P")
	}
	if got := sm2Add(sm2Point{}, g); got.x.Cmp(g.x) != 0 || got.y.Cmp(g.y) != 0 {
		t.Fatal("infinity + P should be P")
	}
	if !sm2IsInf(sm2Double(sm2Point{})) {
		t.Fatal("doubling infinity should be infinity")
	}
	if !sm2IsInf(sm2Double(sm2Point{x: big.NewInt(1), y: big.NewInt(0)})) {
		t.Fatal("doubling a point with y=0 should be infinity")
	}

	// A small coordinate exercises sm2Pad's left-padding branch.
	if padded := sm2Pad(big.NewInt(1)); len(padded) != sm2FieldBytes || padded[sm2FieldBytes-1] != 1 {
		t.Fatalf("sm2Pad(1) = %x, want 32 bytes ending in 01", padded)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("sm2HexInt should panic on a malformed constant")
		}
	}()
	sm2HexInt("nothex")
}
