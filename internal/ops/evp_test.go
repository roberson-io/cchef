package ops

// Derive EVP key tests. CyberChef ships no fixture file for this operation, so
// every expected value below was produced by the CyberChef-server oracle (which
// wraps crypto-js's EvpKDF). The operation is deterministic: the salt is the
// parsed argument value, not a random salt.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// evpRecipe builds a Derive EVP key recipe from its five arguments.
func evpRecipe(pw, pwOpt string, keySize, iters int, hasher, salt, saltOpt string) core.Recipe {
	return core.Recipe{{Op: "Derive EVP key", Args: []any{
		core.ToggleString{Value: pw, Option: pwOpt},
		keySize, iters, hasher,
		core.ToggleString{Value: salt, Option: saltOpt},
	}}}
}

func TestDeriveEVPKey(t *testing.T) {
	runCases(t, []opCase{
		{
			"EVP: SHA1 128 i1", "",
			"c88e9c67041a74e0357befdff93f87dd",
			evpRecipe("password", "UTF8", 128, 1, "SHA1", "73616c74", "Hex"),
		},
		{
			"EVP: MD5 128 i1", "",
			"b305cadbb3bce54f3aa59c64fec00dea",
			evpRecipe("password", "UTF8", 128, 1, "MD5", "73616c74", "Hex"),
		},
		{
			"EVP: SHA256 256 i1", "",
			"7a37b85c8918eac19a9089c0fa5a2ab4dce3f90528dcdeec108b23ddf3607b99",
			evpRecipe("password", "UTF8", 256, 1, "SHA256", "73616c74", "Hex"),
		},
		{
			"EVP: SHA256 256 i3", "",
			"cc19a87959d70ba1d9d2979b5fc2323e0d62a40fb2545492e9ec4d57ce79956d",
			evpRecipe("password", "UTF8", 256, 3, "SHA256", "73616c74", "Hex"),
		},
		{
			"EVP: SHA512 512 i1", "",
			"fa6a2185b3e0a9a85ef41ffb67ef3c1fb6f74980f8ebf970e4e72e353ed9537d593083c201dfd6e43e1c8a7aac2bc8dbb119c7dfb7d4b8f131111395bd70e97f",
			evpRecipe("password", "UTF8", 512, 1, "SHA512", "73616c74", "Hex"),
		},
		{
			"EVP: SHA384 384 i2", "",
			"9f491e14c1c201ac0eb48928370b6efb9371d0a026b82becafa547399072d5ef2917fb46b3d9ee049ddc5fc53d8f1c43",
			evpRecipe("password", "UTF8", 384, 2, "SHA384", "73616c74", "Hex"),
		},
		// Empty salt: EVP with MD5, 128-bit, 1 iteration reduces to MD5(password).
		{
			"EVP: MD5 128 empty salt", "",
			"5f4dcc3b5aa765d61d8327deb882cf99",
			evpRecipe("password", "UTF8", 128, 1, "MD5", "", "Hex"),
		},
		// Multi-block: SHA1 (20-byte block) needs two blocks for a 256-bit key.
		{
			"EVP: SHA1 256 i1 (two blocks)", "",
			"c88e9c67041a74e0357befdff93f87dde09042141736f45af10453231815d26b",
			evpRecipe("password", "UTF8", 256, 1, "SHA1", "73616c74", "Hex"),
		},
		// Latin1 passphrase differs from the same bytes read as UTF8.
		{
			"EVP: Latin1 passphrase", "",
			"9432909609bab49023aa61263061547e",
			evpRecipe("pÃ¤ss", "Latin1", 128, 1, "SHA1", "73616c74", "Hex"),
		},
		{
			"EVP: UTF8 passphrase (contrast)", "",
			"9e63452b6255c6d45d2454d6c157ab4f",
			evpRecipe("pÃ¤ss", "UTF8", 128, 1, "SHA1", "73616c74", "Hex"),
		},
		// Base64 passphrase ("cGFzcw==" = "pass").
		{
			"EVP: Base64 passphrase", "",
			"9d4e1e23bd5b727046a9e3b4b7db57bd",
			evpRecipe("cGFzcw==", "Base64", 128, 1, "SHA1", "", "Hex"),
		},
		// UTF8 salt "salt" == Hex salt "73616c74" (same bytes).
		{
			"EVP: UTF8 salt", "",
			"7a37b85c8918eac19a9089c0fa5a2ab4dce3f90528dcdeec108b23ddf3607b99",
			evpRecipe("password", "UTF8", 256, 1, "SHA256", "salt", "UTF8"),
		},
		// Non-byte-aligned key size: 100 bits → ceil(100/8) = 13 bytes.
		{
			"EVP: 100-bit key", "",
			"b305cadbb3bce54f3aa59c64fe",
			evpRecipe("password", "UTF8", 100, 1, "MD5", "73616c74", "Hex"),
		},
	})
}

// TestDeriveEVPKeyErrors covers the passphrase/salt decode error paths. cchef's
// shared convertToByteArray uses Go's strict base64 decoder, so malformed Base64
// errors here — a minor divergence from CyberChef's lenient fromBase64 (which
// strips non-alphabet characters). This matches the other crypto ops that share
// convertToByteArray (XOR, Ascon, Blowfish).
func TestDeriveEVPKeyErrors(t *testing.T) {
	if _, err := runOp(t, "Derive EVP key",
		"", core.ToggleString{Value: "!!!not base64", Option: "Base64"},
		float64(128), float64(1), "SHA1", core.ToggleString{Value: "", Option: "Hex"}); err == nil {
		t.Fatal("expected error for invalid Base64 passphrase")
	}
	if _, err := runOp(t, "Derive EVP key",
		"", core.ToggleString{Value: "password", Option: "UTF8"},
		float64(128), float64(1), "SHA1", core.ToggleString{Value: "!!!bad", Option: "Base64"}); err == nil {
		t.Fatal("expected error for invalid Base64 salt")
	}
}
