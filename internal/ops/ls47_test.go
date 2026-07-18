package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// LS47 fixtures transcribed from ../CyberChef/tests/operations/tests/LS47.mjs.
func TestLS47Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"LS47 Encrypt", "thequickbrownfoxjumped",
			"(,t74ci78cp/8trx*yesu:alp1wqy",
			core.Recipe{{Op: "LS47 Encrypt", Args: []any{"helloworld", 0, "test"}}},
		},
		{
			"LS47 Decrypt", "(,t74ci78cp/8trx*yesu:alp1wqy",
			"thequickbrownfoxjumped---test",
			core.Recipe{{Op: "LS47 Decrypt", Args: []any{"helloworld", 0}}},
		},
	})
}

// TestLS47InvalidPassword covers the fixture where a password character is not in
// the LS47 alphabet (capital H).
func TestLS47InvalidPassword(t *testing.T) {
	_, err := runOp(t, "LS47 Encrypt", "thequickbrownfoxjumped", "Helloworld", float64(0), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Letter H is not included in LS47") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestLS47PaddingRoundTrip covers encryption with random padding (padding > 0),
// which decryption strips to recover the original plaintext plus signature.
func TestLS47PaddingRoundTrip(t *testing.T) {
	ct, err := runOp(t, "LS47 Encrypt", "secretmessage.42", "mypassword", float64(10), "sig")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := runOp(t, "LS47 Decrypt", ct, "mypassword", float64(10))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if pt != "secretmessage.42---sig" {
		t.Fatalf("round-trip mismatch: %q", pt)
	}
}

// TestLS47InvalidCharacters covers the "not in the key" error for plaintext
// (encrypt) and ciphertext (decrypt) characters outside the LS47 alphabet.
func TestLS47InvalidCharacters(t *testing.T) {
	if _, err := runOp(t, "LS47 Encrypt", "helloH", "helloworld", float64(0), ""); err == nil ||
		!strings.Contains(err.Error(), "Letter H is not in the key") {
		t.Fatalf("encrypt invalid plaintext: %v", err)
	}
	if _, err := runOp(t, "LS47 Decrypt", "helloH", "helloworld", float64(0)); err == nil ||
		!strings.Contains(err.Error(), "Letter H is not in the key") {
		t.Fatalf("decrypt invalid ciphertext: %v", err)
	}
}

// TestLS47DecryptInvalidPassword covers the deriveKey error path from Decrypt.
func TestLS47DecryptInvalidPassword(t *testing.T) {
	if _, err := runOp(t, "LS47 Decrypt", "abc", "Bad", float64(0)); err == nil ||
		!strings.Contains(err.Error(), "not included in LS47") {
		t.Fatalf("expected deriveKey error, got %v", err)
	}
}

// TestLS47PaddingSlice covers the JS slice() clamp branches in decryptPad: a
// count past the end, and negative counts (from the end, clamped to zero).
func TestLS47PaddingSlice(t *testing.T) {
	const ct = "(,t74ci78cp/8trx*yesu:alp1wqy" // decrypts to a 29-char string
	cases := []struct {
		padding int
		want    string
	}{
		{100, ""},                               // past the end -> empty
		{-3, "est"},                             // from the end -> last 3
		{-100, "thequickbrownfoxjumped---test"}, // beyond the start -> whole string
	}
	for _, c := range cases {
		out, err := runOp(t, "LS47 Decrypt", ct, "helloworld", float64(c.padding))
		if err != nil {
			t.Fatalf("padding %d: %v", c.padding, err)
		}
		if out != c.want {
			t.Fatalf("padding %d: got %q want %q", c.padding, out, c.want)
		}
	}
}
