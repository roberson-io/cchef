package ops

// Triple DES Encrypt/Decrypt tests. Values from the CyberChef-server oracle
// (node-forge 3DES). The distinct-key vectors use three independent 8-byte keys
// so 3DES does not collapse to single DES.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestTripleDESEncrypt(t *testing.T) {
	const iv = "0011223344556677"
	// Three distinct 8-byte subkeys (K1, K2, K3).
	const k24 = "0123456789abcdeffedcba9876543210a1b2c3d4e5f60718"
	// 16-byte key exercises the K1‖K2‖K1 expansion.
	const k16 = "0123456789abcdef23456789abcdef01"
	runCases(t, []opCase{
		{
			"3DES enc CBC", "hello world", "ee933ae049848db0fef9cc23009555f5",
			desRecipe("Triple DES Encrypt", k24, "Hex", iv, "Hex", "CBC", "Raw", "Hex"),
		},
		{
			"3DES enc ECB", "hello world", "14048be5d1279b497c0353a7223b368a",
			desRecipe("Triple DES Encrypt", k24, "Hex", iv, "Hex", "ECB", "Raw", "Hex"),
		},
		{
			"3DES enc CTR", "hello world", "43e7b1d2a4f63e4ce9ec2a",
			desRecipe("Triple DES Encrypt", k24, "Hex", iv, "Hex", "CTR", "Raw", "Hex"),
		},
		{
			"3DES enc CBC 16-byte key (K1K2K1)", "hello world",
			"2a980242d389e9defb8cd73a274a33ba",
			desRecipe("Triple DES Encrypt", k16, "Hex", iv, "Hex", "CBC", "Raw", "Hex"),
		},
		{
			"3DES enc ECB 16-byte key (K1K2K1)", "hello world",
			"520ed4255b1a4d655ab03a6eb8e44136",
			desRecipe("Triple DES Encrypt", k16, "Hex", iv, "Hex", "ECB", "Raw", "Hex"),
		},
	})
}

func TestTripleDESDecrypt(t *testing.T) {
	const iv = "0011223344556677"
	const k24 = "0123456789abcdeffedcba9876543210a1b2c3d4e5f60718"
	runCases(t, []opCase{
		{
			"3DES dec CBC", "ee933ae049848db0fef9cc23009555f5", "hello world",
			desRecipe("Triple DES Decrypt", k24, "Hex", iv, "Hex", "CBC", "Hex", "Raw"),
		},
		{
			"3DES dec ECB", "14048be5d1279b497c0353a7223b368a", "hello world",
			desRecipe("Triple DES Decrypt", k24, "Hex", iv, "Hex", "ECB", "Hex", "Raw"),
		},
		{
			"3DES dec CTR", "43e7b1d2a4f63e4ce9ec2a", "hello world",
			desRecipe("Triple DES Decrypt", k24, "Hex", iv, "Hex", "CTR", "Hex", "Raw"),
		},
	})
}

// TestTripleDESErrors covers 16/24-byte key-length validation, a key decode
// error, and the decrypt-side key-error path.
func TestTripleDESErrors(t *testing.T) {
	const iv = "0011223344556677"
	// Encrypt: wrong key lengths and a malformed Base64 key.
	for _, key := range []string{"0123456789abcdef", "0123456789abcdef23456789abcdef0100"} {
		if _, err := runOp(t, "Triple DES Encrypt", "hello",
			core.ToggleString{Value: key, Option: "Hex"},
			core.ToggleString{Value: iv, Option: "Hex"},
			"CBC", "Raw", "Hex"); err == nil {
			t.Fatalf("expected key-length error for %d-hex-char key", len(key))
		}
	}
	if _, err := runOp(t, "Triple DES Encrypt", "hello",
		core.ToggleString{Value: "!!!bad", Option: "Base64"},
		core.ToggleString{Value: iv, Option: "Hex"},
		"CBC", "Raw", "Hex"); err == nil {
		t.Fatal("expected key decode error")
	}
	// Decrypt: wrong key length (distinct Run error path).
	if _, err := runOp(t, "Triple DES Decrypt", "0000000000000000",
		core.ToggleString{Value: "0123456789abcdef", Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		"CBC", "Hex", "Raw"); err == nil {
		t.Fatal("expected decrypt key-length error")
	}
}
