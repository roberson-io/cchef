package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func scryptRecipe(salt core.ToggleString, n, r, p, keyLen int) core.Recipe {
	return core.Recipe{{Op: "Scrypt", Args: []any{salt, n, r, p, keyLen}}}
}

// TestScryptFixtures uses the RFC 7914 known-answer vectors (also confirmed
// byte-for-byte against the CyberChef-server oracle, since the op has no
// upstream fixture file).
func TestScryptFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"RFC 7914 password/NaCl N=1024", "password",
			"fdbabe1c9d3472007856e7190d01e9fe7c6ad7cbc8237830e77376634b3731622eaf30d92e22a3886ff109279d9830dac727afb94a83ee6d8360cbdfa2cc0640",
			scryptRecipe(core.ToggleString{Value: "NaCl", Option: "UTF8"}, 1024, 8, 16, 64),
		},
		{
			"RFC 7914 pleaseletmein/SodiumChloride N=16384", "pleaseletmein",
			"7023bdcb3afd7348461c06cd81fd38ebfda8fbba904f8e3ea9b543f6545da1f2d5432955613f0fcf62d49705242a9af9e61e85dc0d651e40dfcf017b45575887",
			scryptRecipe(core.ToggleString{Value: "SodiumChloride", Option: "UTF8"}, 16384, 8, 1, 64),
		},
	})
}

// TestScryptSaltOptions exercises each salt encoding (oracle-verified).
func TestScryptSaltOptions(t *testing.T) {
	runCases(t, []opCase{
		{
			"Hex salt", "secret",
			"9275ebc83fd0835a40c5665e68e54dee7f125896630e54d3dfc909ae2f616b9b",
			scryptRecipe(core.ToggleString{Value: "00010203", Option: "Hex"}, 16, 8, 1, 32),
		},
		{
			"Base64 salt", "pw",
			"c5d06c368e4a07b8a98e9a06773fcf4a639bb03b909af99f",
			scryptRecipe(core.ToggleString{Value: "c2FsdA==", Option: "Base64"}, 16, 8, 1, 24),
		},
		{
			"Latin1 salt (high bytes)", "pw",
			"c4a870561c8ba9deb392e25b7ce91a7fcae7d059bc3407b0",
			scryptRecipe(core.ToggleString{Value: "ÿþ", Option: "Latin1"}, 16, 8, 1, 24),
		},
		{
			"Empty salt, key length 16", "hello",
			"c82bef90acf838b1123fce6a062ad7a6",
			scryptRecipe(core.ToggleString{Value: "", Option: "UTF8"}, 16, 8, 1, 16),
		},
		{
			"Key length 0 yields empty output", "x",
			"",
			scryptRecipe(core.ToggleString{Value: "s", Option: "UTF8"}, 16, 8, 1, 0),
		},
	})
}

// TestScryptErrors covers the scryptsy validation error strings that cchef
// reproduces, plus a bad salt encoding.
func TestScryptErrors(t *testing.T) {
	cases := []struct {
		name string
		salt core.ToggleString
		n    int
		r    int
		p    int
		want string
	}{
		{"N zero", core.ToggleString{Value: "s", Option: "UTF8"}, 0, 8, 1, "Error: Error: N must be > 0 and a power of 2"},
		{"N not power of 2", core.ToggleString{Value: "s", Option: "UTF8"}, 3, 8, 1, "Error: Error: N must be > 0 and a power of 2"},
		{"N negative", core.ToggleString{Value: "s", Option: "UTF8"}, -1, 8, 1, "Error: Error: N must be > 0 and a power of 2"},
		{"N too large", core.ToggleString{Value: "s", Option: "UTF8"}, 1 << 25, 8, 1, "Error: Error: Parameter N is too large"},
		// N<p keeps the N-size check passing so the r-size check is the one that fires.
		{"r too large", core.ToggleString{Value: "s", Option: "UTF8"}, 2, 6000000, 3, "Error: Error: Parameter r is too large"},
		{"bad base64 salt", core.ToggleString{Value: "!!!bad!!!", Option: "Base64"}, 16, 8, 1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Scrypt", "pw", c.salt, c.n, c.r, c.p, 16)
			if err == nil {
				t.Fatalf("expected error")
			}
			if c.want != "" && err.Error() != c.want {
				t.Fatalf("got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

// TestScryptPowerOfTwoBoundary confirms N=1 (a power of two accepted by
// scryptsy) is rejected by cchef's canonical scrypt backend — a documented
// divergence at an RFC-forbidden parameter.
func TestScryptPowerOfTwoBoundary(t *testing.T) {
	_, err := runOp(t, "Scrypt", "pw", core.ToggleString{Value: "s", Option: "UTF8"}, 1, 8, 1, 16)
	if err == nil || !strings.Contains(err.Error(), "N must be > 1") {
		t.Fatalf("N=1: got %v, want error mentioning N must be > 1", err)
	}
}
