package ops

// Derive PBKDF2 key tests. CyberChef ships no fixture file for this operation,
// so every deterministic expected value below was produced by the CyberChef-
// server oracle (which wraps forge.pkcs5.pbkdf2). The operation ignores its
// input entirely — the passphrase comes from the first argument.
//
// Fidelity note: forge in the browser (what CyberChef users run) uses the
// pure-JS PBKDF2, which truncates the derived-key length toward zero, so a
// key size that is not a multiple of 8 floors to keySize/8 bytes rather than
// erroring, and keySize <= 0 yields an empty string. The oracle's Node-native
// crypto.pbkdf2 path is stricter (it errors on both), so those two cases are
// asserted against the browser semantics, not the oracle.

import (
	"encoding/hex"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// pbkdf2Recipe builds a Derive PBKDF2 key recipe from its five arguments.
func pbkdf2Recipe(pw, pwOpt string, keySize, iters int, hasher, salt, saltOpt string) core.Recipe {
	return core.Recipe{{Op: "Derive PBKDF2 key", Args: []any{
		core.ToggleString{Value: pw, Option: pwOpt},
		keySize, iters, hasher,
		core.ToggleString{Value: salt, Option: saltOpt},
	}}}
}

func TestDerivePBKDF2Key(t *testing.T) {
	// A fixed salt of "salt" (hex 73616c74) makes every case deterministic.
	const salt = "73616c74"
	runCases(t, []opCase{
		// Each hashing function at the defaults (128-bit key, 1 iteration).
		{
			"PBKDF2: SHA1 128 i1", "ignored",
			"0c60c80f961f0e71f3a9b524af601206",
			pbkdf2Recipe("password", "UTF8", 128, 1, "SHA1", salt, "Hex"),
		},
		{
			"PBKDF2: SHA256 128 i1", "ignored",
			"120fb6cffcf8b32c43e7225256c4f837",
			pbkdf2Recipe("password", "UTF8", 128, 1, "SHA256", salt, "Hex"),
		},
		{
			"PBKDF2: SHA384 128 i1", "ignored",
			"c0e14f06e49e32d73f9f52ddf1d0c5c7",
			pbkdf2Recipe("password", "UTF8", 128, 1, "SHA384", salt, "Hex"),
		},
		{
			"PBKDF2: SHA512 128 i1", "ignored",
			"867f70cf1ade02cff3752599a3a53dc4",
			pbkdf2Recipe("password", "UTF8", 128, 1, "SHA512", salt, "Hex"),
		},
		{
			"PBKDF2: MD5 128 i1", "ignored",
			"f31afb6d931392daa5e3130f47f9a9b6",
			pbkdf2Recipe("password", "UTF8", 128, 1, "MD5", salt, "Hex"),
		},

		// Larger key sizes and iteration counts.
		{
			"PBKDF2: SHA256 256 i4", "ignored",
			"cd7b203e3aef28a773613de46901d9a5d621228b3b3de8de24cea5b788459c8a",
			pbkdf2Recipe("password", "UTF8", 256, 4, "SHA256", salt, "Hex"),
		},
		{
			"PBKDF2: SHA512 512 i3", "ignored",
			"a97fe0ebca7f87298179562a255882589e93e4603ef7b8ce6563785021198040" +
				"de86e3d776f8c65b255d2f55c5a07219e15dc3ebd3e55ab7a4e262a41f8c68b0",
			pbkdf2Recipe("password", "UTF8", 512, 3, "SHA512", "00", "Hex"),
		},
		{
			"PBKDF2: SHA256 128 i2", "ignored",
			"ae4d0c95af6b46d32d0adff928f06dd0",
			pbkdf2Recipe("password", "UTF8", 128, 2, "SHA256", salt, "Hex"),
		},

		// Small and off-boundary key sizes. 8 bits → 1 byte; 136 bits → 17 bytes.
		{
			"PBKDF2: 8-bit key", "ignored", "12",
			pbkdf2Recipe("password", "UTF8", 8, 1, "SHA256", salt, "Hex"),
		},
		{
			"PBKDF2: 136-bit key", "ignored",
			"120fb6cffcf8b32c43e7225256c4f837a8",
			pbkdf2Recipe("password", "UTF8", 136, 1, "SHA256", salt, "Hex"),
		},
		// Browser pure-JS semantics: 132 bits floors to 16 bytes (== 128-bit key).
		// The oracle's Node-native path errors here, so this is not oracle-derived.
		{
			"PBKDF2: 132-bit key floors to 16 bytes", "ignored",
			"120fb6cffcf8b32c43e7225256c4f837",
			pbkdf2Recipe("password", "UTF8", 132, 1, "SHA256", salt, "Hex"),
		},
		// Browser pure-JS semantics: keySize 0 → empty derived key.
		{
			"PBKDF2: 0-bit key is empty", "ignored", "",
			pbkdf2Recipe("password", "UTF8", 0, 1, "SHA256", salt, "Hex"),
		},
		// Browser pure-JS semantics: iterations 0 behaves like 1 (the c>=2 loop is
		// skipped). The oracle's Node-native path errors, so compare to the i1 value.
		{
			"PBKDF2: iterations 0 == iterations 1", "ignored",
			"120fb6cffcf8b32c43e7225256c4f837",
			pbkdf2Recipe("password", "UTF8", 128, 0, "SHA256", salt, "Hex"),
		},

		// Empty passphrase.
		{
			"PBKDF2: empty passphrase", "ignored",
			"f135c27993baf98773c5cdb40a5706ce",
			pbkdf2Recipe("", "UTF8", 128, 1, "SHA256", salt, "Hex"),
		},

		// Passphrase/salt toggles.
		{
			"PBKDF2: Hex passphrase ('pass')", "ignored",
			"65acafe9655d154ebe7ca04e8b7ebdbc",
			pbkdf2Recipe("70617373", "Hex", 128, 1, "SHA256", salt, "Hex"),
		},
		{
			"PBKDF2: Base64 salt ('salt')", "ignored",
			"120fb6cffcf8b32c43e7225256c4f837",
			pbkdf2Recipe("password", "UTF8", 128, 1, "SHA256", "c2FsdA==", "Base64"),
		},
		// Non-ASCII passphrase: UTF8 (5 bytes) differs from Latin1 (4 bytes).
		{
			"PBKDF2: UTF8 passphrase (café)", "ignored",
			"62ee13a6686da4d692a02fc9c69fd4f7",
			pbkdf2Recipe("café", "UTF8", 128, 1, "SHA256", salt, "Hex"),
		},
		{
			"PBKDF2: Latin1 passphrase (café)", "ignored",
			"e0935e0c99e022790786a5d5c522602f",
			pbkdf2Recipe("café", "Latin1", 128, 1, "SHA256", salt, "Hex"),
		},
	})
}

// TestDerivePBKDF2RandomSalt covers the empty-salt branch: forge generates a
// random salt of keySize bytes, so the output is non-deterministic but its
// length is fixed at keySize/8 bytes. Two runs must differ.
func TestDerivePBKDF2RandomSalt(t *testing.T) {
	run := func() string {
		out, err := runOp(t, "Derive PBKDF2 key",
			"", core.ToggleString{Value: "password", Option: "UTF8"},
			float64(128), float64(1), "SHA256", core.ToggleString{Value: "", Option: "Hex"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return out
	}
	out := run()
	if len(out) != 32 { // 128-bit key → 16 bytes → 32 hex chars
		t.Fatalf("expected 32 hex chars, got %d (%q)", len(out), out)
	}
	if _, err := hex.DecodeString(out); err != nil {
		t.Fatalf("output is not valid hex: %v", err)
	}
	if out == run() {
		t.Fatal("expected a fresh random salt to produce a different key")
	}
}

// TestDerivePBKDF2Errors covers the passphrase/salt decode error paths (shared
// convertToByteArray uses Go's strict Base64 decoder).
func TestDerivePBKDF2Errors(t *testing.T) {
	if _, err := runOp(t, "Derive PBKDF2 key",
		"", core.ToggleString{Value: "!!!not base64", Option: "Base64"},
		float64(128), float64(1), "SHA256", core.ToggleString{Value: "73616c74", Option: "Hex"}); err == nil {
		t.Fatal("expected error for invalid Base64 passphrase")
	}
	if _, err := runOp(t, "Derive PBKDF2 key",
		"", core.ToggleString{Value: "password", Option: "UTF8"},
		float64(128), float64(1), "SHA256", core.ToggleString{Value: "!!!bad", Option: "Base64"}); err == nil {
		t.Fatal("expected error for invalid Base64 salt")
	}
}

// TestPBKDF2ResourceBounds covers the caps on the two parameters that size the
// work. CyberChef leaves both open, so a mistyped iteration count runs until
// the process is killed.
func TestPBKDF2ResourceBounds(t *testing.T) {
	op, _ := core.Default.Get("Derive PBKDF2 key")
	for _, tc := range []struct {
		name  string
		index int
		value float64
		want  string
	}{
		{"key size", 1, kdfMaxKeySize + 1, "Key size must be less than or equal to 8192."},
		{"iterations", 2, kdfMaxIterations + 1, "Iterations must be less than or equal to 10000000."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := core.DefaultArgs(op.Args())
			args[tc.index] = tc.value
			_, err := core.CoerceArgs(op.Args(), args)
			if err == nil {
				t.Fatalf("%v was accepted", tc.value)
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}
