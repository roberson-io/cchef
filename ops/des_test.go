package ops

// DES Encrypt/Decrypt tests. CyberChef ships no fixture file for these, so every
// expected value was produced by the CyberChef-server oracle (which wraps
// node-forge's DES cipher). Key/IV are hex "0123456789abcdef" / "0011223344556677"
// unless noted.

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// desRecipe builds a DES Encrypt/Decrypt recipe from its five arguments.
func desRecipe(op, key, keyOpt, iv, ivOpt, mode, inputType, outputType string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: iv, Option: ivOpt},
		mode, inputType, outputType,
	}}}
}

func TestDESEncrypt(t *testing.T) {
	const key, iv = "0123456789abcdef", "0011223344556677"
	runCases(t, []opCase{
		{
			"DES enc CBC", "hello world", "fc3e773af001de7dcbdb6e6810f57406",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "CBC", "Raw", "Hex"),
		},
		{
			"DES enc CFB", "hello world", "a2be0bee810b3f4c89f9e6",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "CFB", "Raw", "Hex"),
		},
		{
			"DES enc OFB", "hello world", "a2be0bee810b3f4c88216f",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "OFB", "Raw", "Hex"),
		},
		{
			"DES enc CTR", "hello world", "a2be0bee810b3f4cbe200f",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "CTR", "Raw", "Hex"),
		},
		{
			"DES enc ECB", "hello world", "1f797e16614dab0a6acd31ea6fbcdc6b",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "ECB", "Raw", "Hex"),
		},
		{
			"DES enc CBC hex input", "deadbeefcafe0011", "791dfc285d5bf3179d6da17b722ff5ad",
			desRecipe("DES Encrypt", key, "Hex", iv, "Hex", "CBC", "Hex", "Hex"),
		},
		// UTF8-decoded key/IV (both exactly 8 bytes).
		{
			"DES enc CBC UTF8 key", "hello world", "78e8f4397dd39e6f2ec84a550eac6ae0",
			desRecipe("DES Encrypt", "8bytekey", "UTF8", "8byteivv", "UTF8", "CBC", "Raw", "Hex"),
		},
		// Raw output (byte string) rendered as hex for comparison.
		{
			"DES enc CBC raw output → To Hex", "hello world", "fc3e773af001de7dcbdb6e6810f57406",
			core.Recipe{
				{Op: "DES Encrypt", Args: []any{
					core.ToggleString{Value: key, Option: "Hex"},
					core.ToggleString{Value: iv, Option: "Hex"},
					"CBC", "Raw", "Raw",
				}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}

func TestDESDecrypt(t *testing.T) {
	const key, iv = "0123456789abcdef", "0011223344556677"
	runCases(t, []opCase{
		{
			"DES dec CBC", "fc3e773af001de7dcbdb6e6810f57406", "hello world",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "CBC", "Hex", "Raw"),
		},
		{
			"DES dec CFB", "a2be0bee810b3f4c89f9e6", "hello world",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "CFB", "Hex", "Raw"),
		},
		{
			"DES dec OFB", "a2be0bee810b3f4c88216f", "hello world",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "OFB", "Hex", "Raw"),
		},
		{
			"DES dec CTR", "a2be0bee810b3f4cbe200f", "hello world",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "CTR", "Hex", "Raw"),
		},
		{
			"DES dec ECB", "1f797e16614dab0a6acd31ea6fbcdc6b", "hello world",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "ECB", "Hex", "Raw"),
		},
		// NoPadding modes keep the PKCS#7 padding bytes in the output.
		{
			"DES dec CBC/NoPadding", "fc3e773af001de7dcbdb6e6810f57406",
			"68656c6c6f20776f726c640505050505",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "CBC/NoPadding", "Hex", "Hex"),
		},
		{
			"DES dec ECB/NoPadding", "1f797e16614dab0a6acd31ea6fbcdc6b",
			"68656c6c6f20776f726c640505050505",
			desRecipe("DES Decrypt", key, "Hex", iv, "Hex", "ECB/NoPadding", "Hex", "Hex"),
		},
	})
}

// TestDESErrors covers key/IV validation, decode errors, and the decrypt-failure
// branches, on both the encrypt and decrypt operations.
func TestDESErrors(t *testing.T) {
	const iv = "0011223344556677"
	cases := []struct {
		name                  string
		op, key, keyOpt, mode string
		ivStr, ivOpt, input   string
		inputType, outTy      string
	}{
		{"short key", "DES Encrypt", "00112233", "Hex", "CBC", iv, "Hex", "hello", "Raw", "Hex"},
		{"long key", "DES Encrypt", "0123456789abcdef00", "Hex", "CBC", iv, "Hex", "hello", "Raw", "Hex"},
		{"short IV", "DES Encrypt", "0123456789abcdef", "Hex", "CBC", "0011", "Hex", "hello", "Raw", "Hex"},
		{"bad Base64 key", "DES Encrypt", "!!!bad", "Base64", "CBC", iv, "Hex", "hello", "Raw", "Hex"},
		{"bad Base64 IV", "DES Encrypt", "0123456789abcdef", "Hex", "CBC", "!!!bad", "Base64", "hello", "Raw", "Hex"},
		// Decrypt-side key/IV validation (distinct Run error paths).
		{"dec short key", "DES Decrypt", "00112233", "Hex", "CBC", iv, "Hex", "0000000000000000", "Hex", "Raw"},
		{"dec bad Base64 IV", "DES Decrypt", "0123456789abcdef", "Hex", "CBC", "!!!bad", "Base64", "0000000000000000", "Hex", "Raw"},
		// Ciphertext not a multiple of the block size (CBC and ECB paths).
		{"dec short block CBC", "DES Decrypt", "0123456789abcdef", "Hex", "CBC", iv, "Hex", "deadbeef", "Hex", "Raw"},
		{"dec short block ECB", "DES Decrypt", "0123456789abcdef", "Hex", "ECB", iv, "Hex", "deadbeef", "Hex", "Raw"},
		// Full block that decrypts to invalid PKCS#7 padding.
		{"dec bad padding", "DES Decrypt", "0123456789abcdef", "Hex", "CBC", iv, "Hex", "0000000000000000", "Hex", "Raw"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, c.op, c.input,
				core.ToggleString{Value: c.key, Option: c.keyOpt},
				core.ToggleString{Value: c.ivStr, Option: c.ivOpt},
				c.mode, c.inputType, c.outTy); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}
