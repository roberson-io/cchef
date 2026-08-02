package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// xxteaKey is a Key toggle string.
func xxteaKey(v, opt string) core.ToggleString { return core.ToggleString{Value: v, Option: opt} }

// TestXXTEA covers XXTEA encrypt/decrypt. The first two cases are the upstream
// fixtures (CyberChef's tests/operations/tests/XXTEA.mjs); the rest are verified
// against the CyberChef-server oracle.
func TestXXTEA(t *testing.T) {
	k := xxteaKey("1234567890", "UTF8")
	kHex := xxteaKey("00112233445566778899aabbccddeeff", "Hex")

	runCases(t, []opCase{
		// Fixture: full encrypt/decrypt round trip preserves multibyte UTF-8.
		{
			"round trip", "Hello World! 你好，中国！", "Hello World! 你好，中国！",
			core.Recipe{
				{Op: "XXTEA Encrypt", Args: []any{k}},
				{Op: "XXTEA Decrypt", Args: []any{k}},
			},
		},
		// Fixture: encrypt then hex.
		{
			"encrypt fixture", "ნუ პანიკას",
			"3db5a39db1663fc029bb630a38635b8de5bfef62192e52cc4bf83cda8ccbc701",
			core.Recipe{{Op: "XXTEA Encrypt", Args: []any{k}}, {Op: "To Hex", Args: []any{"None"}}},
		},
		// Oracle-verified encrypt vectors.
		{
			"encrypt test (hex key)", "test", "e2cf35747b2d0fe3",
			core.Recipe{{Op: "XXTEA Encrypt", Args: []any{kHex}}, {Op: "To Hex", Args: []any{"None"}}},
		},
		{
			"encrypt single byte", "A", "1c5965e9b7e2032f",
			core.Recipe{{Op: "XXTEA Encrypt", Args: []any{k}}, {Op: "To Hex", Args: []any{"None"}}},
		},
		{
			"encrypt 8 bytes", "12345678", "5daa8dc8ba1caaa876137e82",
			core.Recipe{{Op: "XXTEA Encrypt", Args: []any{k}}, {Op: "To Hex", Args: []any{"None"}}},
		},
		{
			"encrypt short key padded", "hello", "97b74b19c42b357665d55d27",
			core.Recipe{{Op: "XXTEA Encrypt", Args: []any{xxteaKey("ab", "UTF8")}}, {Op: "To Hex", Args: []any{"None"}}},
		},

		// Standalone decrypt (inverse of the encrypt vectors above).
		{
			"decrypt test", "e2cf35747b2d0fe3", "test",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "XXTEA Decrypt", Args: []any{kHex}}},
		},
		{
			"decrypt multibyte", "3db5a39db1663fc029bb630a38635b8de5bfef62192e52cc4bf83cda8ccbc701", "ნუ პანიკას",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "XXTEA Decrypt", Args: []any{k}}},
		},

		// Empty input passes through unchanged for both directions.
		{"encrypt empty", "", "", core.Recipe{{Op: "XXTEA Encrypt", Args: []any{k}}}},
		{"decrypt empty", "", "", core.Recipe{{Op: "XXTEA Decrypt", Args: []any{k}}}},
	})
}

// TestXXTEADecryptError covers the "Unable to decrypt using this key" path, when
// the length word of the decrypted block is inconsistent (invalid ciphertext).
func TestXXTEADecryptError(t *testing.T) {
	for _, in := range []string{"\x00\x00\x00\x00\x00\x00\x00\x00", "\x00\x00\x00\x00"} {
		_, err := core.Recipe{{Op: "XXTEA Decrypt", Args: []any{xxteaKey("1234567890", "UTF8")}}}.
			Execute(core.NewDish([]byte(in), core.TypeArrayBuffer))
		if err == nil || !strings.Contains(err.Error(), "Unable to decrypt using this key") {
			t.Fatalf("input %q: got %v, want decrypt error", in, err)
		}
	}
}

// TestXXTEAKeyError covers the invalid-key (bad Base64) path in both operations.
func TestXXTEAKeyError(t *testing.T) {
	badKey := xxteaKey("!!!", "Base64")
	for _, op := range []string{"XXTEA Encrypt", "XXTEA Decrypt"} {
		_, err := core.Recipe{{Op: op, Args: []any{badKey}}}.
			Execute(core.NewDish([]byte("data"), core.TypeArrayBuffer))
		if err == nil {
			t.Fatalf("%s: expected key error", op)
		}
	}
}
