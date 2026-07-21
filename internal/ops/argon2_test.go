package ops

import (
	"encoding/hex"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Argon2 has no upstream operation fixtures and the CyberChef-server oracle
// cannot run its WASM, so these vectors come from argon2-cffi 25.1.0 — the
// reference phc-winner-argon2 C library that CyberChef's argon2-browser WASM is
// compiled from (verified: x/crypto reproduces the Argon2i/Argon2id values
// exactly). Args: Salt, Iterations, Memory (KiB), Parallelism, Hash length, Type,
// Output format.
func argon2Recipe(salt, saltOpt string, t, m, p, l int, typ, format string) core.Recipe {
	return core.Recipe{{Op: "Argon2", Args: []any{
		core.ToggleString{Value: salt, Option: saltOpt}, t, m, p, l, typ, format,
	}}}
}

func TestArgon2Encoded(t *testing.T) {
	runCases(t, []opCase{
		{
			"Argon2i encoded", "password",
			"$argon2i$v=19$m=256,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2i", "Encoded hash"),
		},
		{
			"Argon2d encoded", "password",
			"$argon2d$v=19$m=256,t=2,p=1$c29tZXNhbHQ$JcTui6RIBUtJ78gE5Hi52CO+H5vS6Z9R1uxAB6OhUB8",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2d", "Encoded hash"),
		},
		{
			"Argon2id encoded", "password",
			"$argon2id$v=19$m=256,t=2,p=1$c29tZXNhbHQ$nf65EOgLrQMR/uIPnA4rEsF5h7TKyQwu9U1bMCHGi/4",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2id", "Encoded hash"),
		},
		{
			"Argon2id encoded (params 2)", "cchef",
			"$argon2id$v=19$m=512,t=3,p=2$TmFDbDEyMzQ$mj4YPfhoWTnG2KQZzXtC8A",
			argon2Recipe("NaCl1234", "UTF8", 3, 512, 2, 16, "Argon2id", "Encoded hash"),
		},
		{
			"Argon2d encoded (params 2)", "abc",
			"$argon2d$v=19$m=128,t=1,p=1$MDAxMTIyMzM0NDU1NjY3Nw$4YDMe+z04Nyz9XXDMI4t4JVvBgC0f/0Z",
			argon2Recipe("0011223344556677", "UTF8", 1, 128, 1, 24, "Argon2d", "Encoded hash"),
		},
	})
}

func TestArgon2Hex(t *testing.T) {
	runCases(t, []opCase{
		{
			"Argon2i hex", "password",
			"89e9029f4637b295beb027056a7336c414fadd43f6b208645281cb214a56452f",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2i", "Hex hash"),
		},
		{
			"Argon2d hex", "password",
			"25c4ee8ba448054b49efc804e478b9d823be1f9bd2e99f51d6ec4007a3a1501f",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2d", "Hex hash"),
		},
		{
			"Argon2id hex", "password",
			"9dfeb910e80bad0311fee20f9c0e2b12c17987b4cac90c2ef54d5b3021c68bfe",
			argon2Recipe("somesalt", "UTF8", 2, 256, 1, 32, "Argon2id", "Hex hash"),
		},
		// Multi-lane Argon2d exercises the from-scratch core's cross-lane
		// addressing and final-block XOR.
		{
			"Argon2d hex (parallelism 2)", "password",
			"fd152795f02375f12cacabef2071743144928e86a7e08ea5fef20e3841c46235",
			argon2Recipe("somesalt", "UTF8", 2, 512, 2, 32, "Argon2d", "Hex hash"),
		},
	})
}

// A salt whose declared encoding cannot be decoded surfaces the decode error.
func TestArgon2SaltDecodeError(t *testing.T) {
	_, err := runOp(t, "Argon2", "password",
		core.ToggleString{Value: "!!! not base64", Option: "Base64"}, 2, 256, 1, 32, "Argon2i", "Hex hash")
	if err == nil {
		t.Fatal("want error for undecodable salt")
	}
}

// Raw output is the digest bytes decoded (UTF-8 with a Latin-1 fallback).
func TestArgon2Raw(t *testing.T) {
	digest, _ := hex.DecodeString("89e9029f4637b295beb027056a7336c414fadd43f6b208645281cb214a56452f")
	runes := make([]rune, len(digest))
	for i, b := range digest {
		runes[i] = rune(b)
	}
	got, err := runOp(t, "Argon2", "password",
		core.ToggleString{Value: "somesalt", Option: "UTF8"}, 2, 256, 1, 32, "Argon2i", "Raw hash")
	if err != nil || got != string(runes) {
		t.Fatalf("Raw output mismatch: got %q, %v", got, err)
	}
}

// Parameter validation reproduces the reference argon2 C library's messages and
// their check order (output length, then salt, memory, time, parallelism).
func TestArgon2Errors(t *testing.T) {
	cases := []struct {
		name          string
		t, m, p, l    int
		salt, wantErr string
	}{
		{"hash length too short", 2, 256, 1, 3, "somesalt", "Error: Output is too short"},
		{"salt too short", 2, 256, 1, 32, "short", "Error: Salt is too short"},
		{"memory too small", 2, 7, 1, 32, "somesalt", "Error: Memory cost is too small"},
		{"time too small", 0, 256, 1, 32, "somesalt", "Error: Time cost is too small"},
		{"too few lanes", 2, 256, 0, 32, "somesalt", "Error: Too few lanes"},
		{"output checked first", 0, 1, 1, 3, "x", "Error: Output is too short"},
		{"salt before memory", 2, 1, 1, 32, "x", "Error: Salt is too short"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Argon2", "password",
				core.ToggleString{Value: c.salt, Option: "UTF8"}, c.t, c.m, c.p, c.l, "Argon2i", "Hex hash")
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}

// Argon2 compare verifies a password against an encoded hash of any type.
func TestArgon2Compare(t *testing.T) {
	cmp := func(encoded string) core.Recipe {
		return core.Recipe{{Op: "Argon2 compare", Args: []any{encoded}}}
	}
	runCases(t, []opCase{
		{
			"compare i match", "password", "Match: password",
			cmp("$argon2i$v=19$m=256,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
		{
			"compare d match", "password", "Match: password",
			cmp("$argon2d$v=19$m=256,t=2,p=1$c29tZXNhbHQ$JcTui6RIBUtJ78gE5Hi52CO+H5vS6Z9R1uxAB6OhUB8"),
		},
		{
			"compare id match", "password", "Match: password",
			cmp("$argon2id$v=19$m=256,t=2,p=1$c29tZXNhbHQ$nf65EOgLrQMR/uIPnA4rEsF5h7TKyQwu9U1bMCHGi/4"),
		},
		{
			"compare wrong password", "wrongpass", "No match",
			cmp("$argon2i$v=19$m=256,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
		// Each malformed encoded-hash field falls through to "No match".
		{"compare wrong field count", "password", "No match", cmp("not-a-valid-hash")},
		{
			"compare unknown type", "password", "No match",
			cmp("$argon2x$v=19$m=256,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
		{
			"compare wrong version", "password", "No match",
			cmp("$argon2i$v=16$m=256,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
		{
			"compare bad params", "password", "No match",
			cmp("$argon2i$v=19$m=x,t=2,p=1$c29tZXNhbHQ$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
		{
			"compare bad base64 salt", "password", "No match",
			cmp("$argon2i$v=19$m=256,t=2,p=1$@@@@$iekCn0Y3spW+sCcFanM2xBT63UP2sghkUoHLIUpWRS8"),
		},
	})
}
