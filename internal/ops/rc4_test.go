package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// rc4Recipe builds a single-op RC4 recipe.
func rc4Recipe(key, keyOpt, inFmt, outFmt string) core.Recipe {
	return core.Recipe{{Op: "RC4", Args: []any{
		core.ToggleString{Value: key, Option: keyOpt}, inFmt, outFmt,
	}}}
}

// rc4DropRecipe builds a single-op RC4 Drop recipe.
func rc4DropRecipe(key, keyOpt, inFmt, outFmt string, drop float64) core.Recipe {
	return core.Recipe{{Op: "RC4 Drop", Args: []any{
		core.ToggleString{Value: key, Option: keyOpt}, inFmt, outFmt, drop,
	}}}
}

// RC4 has no upstream fixture file. The Hex-output vectors are the classic RC4
// test vectors (Wikipedia) and CyberChef-server oracle values (CryptoJS).
func TestRC4Vectors(t *testing.T) {
	runCases(t, []opCase{
		// Classic RC4 test vectors.
		{
			"RC4: Key/Plaintext", "Plaintext", "bbf316e8d940af0ad3",
			rc4Recipe("Key", "Latin1", "Latin1", "Hex"),
		},
		{
			"RC4: Wiki/pedia", "pedia", "1021bf0420",
			rc4Recipe("Wiki", "Latin1", "Latin1", "Hex"),
		},
		{
			"RC4: Secret/Attack", "Attack at dawn", "45a01f645fc35b383552544b9bf5",
			rc4Recipe("Secret", "Latin1", "Latin1", "Hex"),
		},
		// Input formats (all encrypt "Plaintext" under key "Key").
		{
			"RC4: Hex input", "506c61696e74657874", "bbf316e8d940af0ad3",
			rc4Recipe("Key", "Latin1", "Hex", "Hex"),
		},
		{
			"RC4: Base64 input", "UGxhaW50ZXh0", "bbf316e8d940af0ad3",
			rc4Recipe("Key", "Latin1", "Base64", "Hex"),
		},
		{
			"RC4: UTF16 input", "AB", "ebde77c3",
			rc4Recipe("Key", "Latin1", "UTF16", "Hex"),
		},
		{
			"RC4: UTF16LE input", "AB", "aa9f3581",
			rc4Recipe("Key", "Latin1", "UTF16LE", "Hex"),
		},
		{
			"RC4: UTF8 astral input", "A\U0001F600B", "aa6fe8193776",
			rc4Recipe("Key", "Latin1", "UTF8", "Hex"),
		},
		{
			"RC4: empty key", "Plaintext", "8e74e828cd433842fe",
			rc4Recipe("", "Latin1", "Latin1", "Hex"),
		},
		// Output formats.
		{
			"RC4: Base64 output", "Plaintext", "u/MW6NlArwrT",
			rc4Recipe("Key", "Latin1", "Latin1", "Base64"),
		},
	})
}

// TestRC4LatinOutput checks the Latin1 output encodes each ciphertext byte as a
// code point (bytes bb f3 16 e8 d9 40 af 0a d3).
func TestRC4LatinOutput(t *testing.T) {
	want := string([]rune{0xbb, 0xf3, 0x16, 0xe8, 0xd9, 0x40, 0xaf, 0x0a, 0xd3})
	out, err := runOp(t, "RC4", "Plaintext",
		core.ToggleString{Value: "Key", Option: "Latin1"}, "Latin1", "Latin1")
	if err != nil {
		t.Fatal(err)
	}
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

// TestRC4DropVectors covers the drop parameter (dwords, i.e. 4 keystream bytes
// each; drop 0 equals plain RC4).
func TestRC4DropVectors(t *testing.T) {
	runCases(t, []opCase{
		{
			"RC4 Drop: drop 0", "Plaintext", "bbf316e8d940af0ad3",
			rc4DropRecipe("Key", "Latin1", "Latin1", "Hex", 0),
		},
		{
			"RC4 Drop: drop 1", "Plaintext", "e758ab1bc96d2f5013",
			rc4DropRecipe("Key", "Latin1", "Latin1", "Hex", 1),
		},
		{
			"RC4 Drop: drop 2", "Plaintext", "f7752b4109c227ed79",
			rc4DropRecipe("Key", "Latin1", "Latin1", "Hex", 2),
		},
		{
			"RC4 Drop: drop 192", "Plaintext", "857047028b192029fd",
			rc4DropRecipe("Key", "Latin1", "Latin1", "Hex", 192),
		},
	})
}

// TestRC4RoundTrip verifies RC4 is its own inverse across input/output formats.
func TestRC4RoundTrip(t *testing.T) {
	// Hex/Base64/Latin1 losslessly represent any ciphertext; UTF8/UTF16 do not.
	formats := []string{"Latin1", "Hex", "Base64"}
	for _, f := range formats {
		enc, err := runOp(t, "RC4", "The quick brown fox!",
			core.ToggleString{Value: "s3cr3t", Option: "UTF8"}, "Latin1", f)
		if err != nil {
			t.Fatalf("encrypt %s: %v", f, err)
		}
		dec, err := runOp(t, "RC4", enc,
			core.ToggleString{Value: "s3cr3t", Option: "UTF8"}, f, "Latin1")
		if err != nil {
			t.Fatalf("decrypt %s: %v", f, err)
		}
		if dec != "The quick brown fox!" {
			t.Fatalf("round-trip %s: got %q", f, dec)
		}
	}
}

// TestRC4UTF8OutputError covers CryptoJS's "Malformed UTF-8 data" error when the
// ciphertext is not valid UTF-8.
func TestRC4UTF8OutputError(t *testing.T) {
	if _, err := runOp(t, "RC4", "Plaintext",
		core.ToggleString{Value: "Key", Option: "Latin1"}, "Latin1", "UTF8"); err == nil ||
		!strings.Contains(err.Error(), "Malformed UTF-8 data") {
		t.Fatalf("expected malformed UTF-8 error, got %v", err)
	}
}

// TestRC4UTF16Output covers the UTF16 output path with an oracle-verified
// representable case (no lone surrogate). Lone-surrogate ciphertext cannot be
// represented as a Go UTF-8 string, so it is not byte-exact (see rc4DecodeUTF16).
func TestRC4UTF16Output(t *testing.T) {
	out, err := runOp(t, "RC4", "ab",
		core.ToggleString{Value: "0", Option: "Latin1"}, "Latin1", "UTF16")
	if err != nil {
		t.Fatal(err)
	}
	if want := string([]rune{0xe91d}); out != want {
		t.Fatalf("got %q want %q", out, want)
	}
	// Also exercise the little-endian output branch.
	if _, err := runOp(t, "RC4", "ab",
		core.ToggleString{Value: "0", Option: "Latin1"}, "Latin1", "UTF16LE"); err != nil {
		t.Fatal(err)
	}
}

// TestRC4UTF16Codec directly round-trips the UTF-16 encoders for representable
// text and exercises the odd-length / lone-surrogate branches.
func TestRC4UTF16Codec(t *testing.T) {
	for _, be := range []bool{true, false} {
		const s = "Aé中"
		if got := rc4DecodeUTF16(rc4EncodeUTF16(s, be), be); got != s {
			t.Fatalf("round-trip be=%v: got %q", be, got)
		}
	}
	// Valid UTF-8 bytes stringify unchanged.
	if got, err := rc4Stringify([]byte("hi ☺"), "UTF8"); err != nil || got != "hi ☺" {
		t.Fatalf("utf8 stringify: %q %v", got, err)
	}
	// A lone high surrogate plus a trailing odd byte decodes to replacement chars.
	if got := rc4DecodeUTF16([]byte{0xd8, 0x00, 0x41}, true); got == "" {
		t.Fatal("expected non-empty best-effort decode")
	}
}

// TestRC4ParseErrors covers the key- and input-parse error paths (invalid Base64).
func TestRC4ParseErrors(t *testing.T) {
	if _, err := runOp(t, "RC4", "data",
		core.ToggleString{Value: "!!!bad!!!", Option: "Base64"}, "Latin1", "Hex"); err == nil {
		t.Fatal("bad base64 key should error")
	}
	if _, err := runOp(t, "RC4", "!!!bad!!!",
		core.ToggleString{Value: "k", Option: "Latin1"}, "Base64", "Hex"); err == nil {
		t.Fatal("bad base64 input should error")
	}
}
