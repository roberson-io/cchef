package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// presentRecipe builds a single-op PRESENT recipe with Hex-encoded key/IV.
func presentRecipe(op, key, iv, mode, inType, outType, padding string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		mode, inType, outType, padding,
	}}}
}

// PRESENT fixtures transcribed from
// CyberChef's tests/operations/tests/PRESENT.mjs. The official vectors come
// from the PRESENT paper (Bogdanov et al., CHES 2007), Table 3.
func TestPRESENTOfficialVectors(t *testing.T) {
	const zeroKey80 = "00000000000000000000"
	const onesKey80 = "ffffffffffffffffffff"
	const zeroKey128 = "00000000000000000000000000000000"
	runCases(t, []opCase{
		{
			"PRESENT Official Vector 1", "0000000000000000", "5579c1387b228445",
			presentRecipe("PRESENT Encrypt", zeroKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 2", "0000000000000000", "e72c46c0f5945049",
			presentRecipe("PRESENT Encrypt", onesKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 3", "ffffffffffffffff", "a112ffc72f68417b",
			presentRecipe("PRESENT Encrypt", zeroKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 4", "ffffffffffffffff", "3333dcd3213210d2",
			presentRecipe("PRESENT Encrypt", onesKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 5", "0000000000000000", "96db702a2e6900af",
			presentRecipe("PRESENT Encrypt", zeroKey128, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 6", "0123456789abcdef", "0e9d28685e671dd6",
			presentRecipe("PRESENT Encrypt", "0123456789abcdef0123456789abcdef", "", "ECB", "Hex", "Hex", "NO"),
		},
		// Decrypt verification of the official vectors.
		{
			"PRESENT Official Vector 1 Decrypt", "5579c1387b228445", "0000000000000000",
			presentRecipe("PRESENT Decrypt", zeroKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 4 Decrypt", "3333dcd3213210d2", "ffffffffffffffff",
			presentRecipe("PRESENT Decrypt", onesKey80, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 5 Decrypt", "96db702a2e6900af", "0000000000000000",
			presentRecipe("PRESENT Decrypt", zeroKey128, "", "ECB", "Hex", "Hex", "NO"),
		},
		{
			"PRESENT Official Vector 6 Decrypt", "0e9d28685e671dd6", "0123456789abcdef",
			presentRecipe("PRESENT Decrypt", "0123456789abcdef0123456789abcdef", "", "ECB", "Hex", "Hex", "NO"),
		},
	})
}

// TestPRESENTConsistency covers the fixed-output PKCS5 encrypt vectors.
func TestPRESENTConsistency(t *testing.T) {
	runCases(t, []opCase{
		{
			"PRESENT Encrypt: 80-bit zero key consistency", "TestData", "b78cfea5ffcd89f265585a6ce7312131",
			presentRecipe("PRESENT Encrypt", "00000000000000000000", "", "ECB", "Raw", "Hex", "PKCS5"),
		},
		{
			"PRESENT Encrypt: 128-bit zero key consistency", "TestData", "e127a24e38de2c36407e794ef5dffefd",
			presentRecipe("PRESENT Encrypt", "00000000000000000000000000000000", "", "ECB", "Raw", "Hex", "PKCS5"),
		},
	})
}

// TestPRESENTRoundTrips covers the encrypt→decrypt round-trip fixtures across
// modes, key sizes and input lengths.
func TestPRESENTRoundTrips(t *testing.T) {
	cases := []struct {
		name, input, key, iv, keyOpt, ivOpt, mode string
	}{
		{"ECB 80-bit short", "Hello!!!", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"CBC 80-bit long", "The quick brown fox jumps over the lazy dog", "aabbccddeeff00112233", "0011223344556677", "Hex", "Hex", "CBC"},
		{"ECB 128-bit", "Testing PRESENT cipher with 128-bit key", "00112233445566778899aabbccddeeff", "", "Hex", "Hex", "ECB"},
		{"CBC 128-bit", "PRESENT is an ultra-lightweight block cipher!", "ffeeddccbbaa99887766554433221100", "8877665544332211", "Hex", "Hex", "CBC"},
		{"UTF8 key", "Secret message", "mypassword", "initvect", "UTF8", "UTF8", "CBC"},
		{"length 1", "A", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"length 7", "1234567", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"length 8 exact block", "12345678", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"length 9", "123456789", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"length 16 two blocks", "1234567890ABCDEF", "00112233445566778899", "", "Hex", "Hex", "ECB"},
		{"binary CBC", "\x00\x01\x02\x03\x04\x05\x06\x07", "ffeeddccbbaa99887766", "0011223344556677", "Hex", "Hex", "CBC"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recipe := core.Recipe{
				{Op: "PRESENT Encrypt", Args: []any{
					core.ToggleString{Value: c.key, Option: c.keyOpt},
					core.ToggleString{Value: c.iv, Option: c.ivOpt},
					c.mode, "Raw", "Hex", "PKCS5",
				}},
				{Op: "PRESENT Decrypt", Args: []any{
					core.ToggleString{Value: c.key, Option: c.keyOpt},
					core.ToggleString{Value: c.iv, Option: c.ivOpt},
					c.mode, "Hex", "Raw", "PKCS5",
				}},
			}
			out, err := recipe.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if out.String() != c.input {
				t.Fatalf("round-trip: got %q want %q", out.String(), c.input)
			}
		})
	}
}

// TestPRESENTKeyIVErrors covers the key- and IV-length validation messages.
func TestPRESENTKeyIVErrors(t *testing.T) {
	// Wrong key length (8 bytes).
	if _, err := runOp(t, "PRESENT Encrypt",
		"0000000000000000",
		core.ToggleString{Value: "0011223344556677", Option: "Hex"},
		core.ToggleString{Value: "", Option: "Hex"},
		"ECB", "Hex", "Hex", "NO"); err == nil ||
		!strings.Contains(err.Error(), "Invalid key length: 8 bytes") {
		t.Fatalf("key length: %v", err)
	}
	// Missing IV in CBC mode.
	if _, err := runOp(t, "PRESENT Encrypt",
		"0000000000000000",
		core.ToggleString{Value: "00000000000000000000", Option: "Hex"},
		core.ToggleString{Value: "", Option: "Hex"},
		"CBC", "Hex", "Hex", "NO"); err == nil ||
		!strings.Contains(err.Error(), "Invalid IV length: 0 bytes") {
		t.Fatalf("iv length: %v", err)
	}
}

// hexKey80 / hexIV are shared helpers for the coverage tests.
func hexKey80() core.ToggleString {
	return core.ToggleString{Value: "00000000000000000000", Option: "Hex"}
}
func hexEmpty() core.ToggleString { return core.ToggleString{Value: "", Option: "Hex"} }

// TestPRESENTEmptyInput covers the empty-message short-circuit in both directions.
func TestPRESENTEmptyInput(t *testing.T) {
	if out, err := runOp(t, "PRESENT Encrypt", "", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "PKCS5"); err != nil || out != "" {
		t.Fatalf("encrypt empty: %q %v", out, err)
	}
	if out, err := runOp(t, "PRESENT Decrypt", "", hexKey80(), hexEmpty(), "ECB", "Hex", "Raw", "PKCS5"); err != nil || out != "" {
		t.Fatalf("decrypt empty: %q %v", out, err)
	}
}

// TestPRESENTPaddingSchemes covers the ZERO, RANDOM, BIT and NO padding paths.
func TestPRESENTPaddingSchemes(t *testing.T) {
	// ZERO: pads with zero bytes; the padding survives decryption.
	ct, err := runOp(t, "PRESENT Encrypt", "Test", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "ZERO")
	if err != nil || ct != "beb61b260e562676" {
		t.Fatalf("zero encrypt: %q %v", ct, err)
	}
	if out, err := runOp(t, "PRESENT Decrypt", ct, hexKey80(), hexEmpty(), "ECB", "Hex", "Hex", "ZERO"); err != nil || out != "5465737400000000" {
		t.Fatalf("zero decrypt: %q %v", out, err)
	}
	// RANDOM: non-deterministic padding, so just check it produces one block.
	if out, err := runOp(t, "PRESENT Encrypt", "Test", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "RANDOM"); err != nil || len(out) != 16 {
		t.Fatalf("random encrypt: %q %v", out, err)
	}
	// BIT: removable padding, round-trips.
	ct, err = runOp(t, "PRESENT Encrypt", "Test", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "BIT")
	if err != nil {
		t.Fatalf("bit encrypt: %v", err)
	}
	if out, err := runOp(t, "PRESENT Decrypt", ct, hexKey80(), hexEmpty(), "ECB", "Hex", "Raw", "BIT"); err != nil || out != "Test" {
		t.Fatalf("bit decrypt: %q %v", out, err)
	}
	// NO padding on a non-block-multiple input errors.
	if _, err := runOp(t, "PRESENT Encrypt", "abc", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "NO"); err == nil ||
		!strings.Contains(err.Error(), "No padding requested but input is not a 8-byte multiple.") {
		t.Fatalf("no-pad unaligned: %v", err)
	}
}

// TestPRESENTDecryptErrors covers ciphertext-length and PKCS5 padding failures.
func TestPRESENTDecryptErrors(t *testing.T) {
	// Ciphertext length not a multiple of the block size.
	if _, err := runOp(t, "PRESENT Decrypt", "abcdef", hexKey80(), hexEmpty(), "ECB", "Hex", "Raw", "PKCS5"); err == nil ||
		!strings.Contains(err.Error(), "Invalid ciphertext length: 3 bytes. Must be a multiple of 8.") {
		t.Fatalf("length: %v", err)
	}
	// Encrypt aligned data with NO padding, then decrypt as PKCS5 -> invalid padding.
	ct, err := runOp(t, "PRESENT Encrypt", "12345678", hexKey80(), hexEmpty(), "ECB", "Raw", "Hex", "NO")
	if err != nil {
		t.Fatalf("setup encrypt: %v", err)
	}
	if _, err := runOp(t, "PRESENT Decrypt", ct, hexKey80(), hexEmpty(), "ECB", "Hex", "Raw", "PKCS5"); err == nil ||
		!strings.Contains(err.Error(), "Invalid PKCS#5 padding.") {
		t.Fatalf("pkcs5: %v", err)
	}
}

// TestPRESENTRemovePaddingBranches directly exercises the padding-removal error
// branches that are awkward to reach through a full decrypt.
func TestPRESENTRemovePaddingBranches(t *testing.T) {
	// Empty message is returned unchanged (guard mirrors the upstream check;
	// unreachable via decrypt, which short-circuits empty ciphertext).
	if out, err := blockRemovePadding([]byte{}, "PKCS5", presentBlockSize); err != nil || len(out) != 0 {
		t.Fatalf("empty: %v %v", out, err)
	}
	// PKCS5: pad byte in range but the bytes do not all match.
	if _, err := blockRemovePadding([]byte{1, 2, 3, 4, 5, 6, 0, 2}, "PKCS5", presentBlockSize); err == nil ||
		!strings.Contains(err.Error(), "Invalid PKCS#5 padding.") {
		t.Fatalf("pkcs5 mismatch: %v", err)
	}
	// PKCS5: pad byte of zero is invalid.
	if _, err := blockRemovePadding([]byte{1, 2, 3, 4, 5, 6, 7, 0}, "PKCS5", presentBlockSize); err == nil {
		t.Fatal("pkcs5 zero pad byte should error")
	}
	// BIT: a non-zero byte before any 0x80 is invalid.
	if _, err := blockRemovePadding([]byte{1, 2, 3, 4, 5, 6, 7, 9}, "BIT", presentBlockSize); err == nil ||
		!strings.Contains(err.Error(), "Invalid BIT padding.") {
		t.Fatalf("bit non-zero: %v", err)
	}
	// BIT: all-zero (no 0x80 marker) is invalid.
	if _, err := blockRemovePadding([]byte{0, 0, 0, 0, 0, 0, 0, 0}, "BIT", presentBlockSize); err == nil {
		t.Fatal("bit all-zero should error")
	}
}

// TestPRESENTKeyIVDecodeErrors covers invalid Base64 key/IV and decrypt-side
// key/IV validation.
func TestPRESENTKeyIVDecodeErrors(t *testing.T) {
	badB64 := core.ToggleString{Value: "!!!not base64!!!", Option: "Base64"}
	if _, err := runOp(t, "PRESENT Encrypt", "0000000000000000", badB64, hexEmpty(), "ECB", "Hex", "Hex", "NO"); err == nil {
		t.Fatal("bad base64 key should error")
	}
	// Decrypt-side key-length and IV-length validation.
	if _, err := runOp(t, "PRESENT Decrypt", "0000000000000000",
		core.ToggleString{Value: "0011", Option: "Hex"}, hexEmpty(), "ECB", "Hex", "Raw", "NO"); err == nil ||
		!strings.Contains(err.Error(), "Invalid key length: 2 bytes") {
		t.Fatalf("decrypt key length: %v", err)
	}
	if _, err := runOp(t, "PRESENT Decrypt", "0000000000000000", hexKey80(),
		core.ToggleString{Value: "00", Option: "Hex"}, "CBC", "Hex", "Raw", "NO"); err == nil ||
		!strings.Contains(err.Error(), "Invalid IV length: 1 bytes") {
		t.Fatalf("decrypt iv length: %v", err)
	}
	// Invalid Base64 IV in CBC mode.
	if _, err := runOp(t, "PRESENT Encrypt", "0000000000000000", hexKey80(),
		core.ToggleString{Value: "!!bad!!", Option: "Base64"}, "CBC", "Hex", "Hex", "NO"); err == nil {
		t.Fatal("bad base64 IV should error")
	}
}
