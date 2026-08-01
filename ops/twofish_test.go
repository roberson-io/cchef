package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// twofishKey builds a Key/IV toggle string.
func twofishKey(v, opt string) core.ToggleString { return core.ToggleString{Value: v, Option: opt} }

// twoEnc builds a Twofish Encrypt recipe step.
func twoEnc(key, iv core.ToggleString, mode, in, out, pad string) core.RecipeOp {
	return core.RecipeOp{Op: "Twofish Encrypt", Args: []any{key, iv, mode, in, out, pad}}
}

// twoDec builds a Twofish Decrypt recipe step.
func twoDec(key, iv core.ToggleString, mode, in, out, pad string) core.RecipeOp {
	return core.RecipeOp{Op: "Twofish Decrypt", Args: []any{key, iv, mode, in, out, pad}}
}

// TestTwofish covers the operations against CyberChef's own fixtures
// (../CyberChef/tests/operations/tests/Twofish.mjs): the official Twofish-paper
// ECB vectors and round trips across every mode and key size.
func TestTwofish(t *testing.T) {
	z128 := twofishKey("00000000000000000000000000000000", "Hex")
	z192 := twofishKey("000000000000000000000000000000000000000000000000", "Hex")
	z256 := twofishKey("0000000000000000000000000000000000000000000000000000000000000000", "Hex")
	k128 := twofishKey("00112233445566778899aabbccddeeff", "Hex")
	k192 := twofishKey("000102030405060708090a0b0c0d0e0f1011121314151617", "Hex")
	k256 := twofishKey("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "Hex")
	noIV := twofishKey("", "Hex")
	iv0 := twofishKey("ffeeddccbbaa99887766554433221100", "Hex")

	runCases(t, []opCase{
		// Official Twofish-paper vectors (ECB, no padding).
		{
			"official 128-bit", "00000000000000000000000000000000",
			"9f589f5cf6122c32b6bfec2f2ae8c35a",
			core.Recipe{twoEnc(z128, noIV, "ECB", "Hex", "Hex", "NO")},
		},
		{
			"official 192-bit", "00000000000000000000000000000000",
			"efa71f788965bd4453f860178fc19101",
			core.Recipe{twoEnc(z192, noIV, "ECB", "Hex", "Hex", "NO")},
		},
		{
			"official 256-bit", "00000000000000000000000000000000",
			"57ff739d4dc92c1bd7fc01700cc8216f",
			core.Recipe{twoEnc(z256, noIV, "ECB", "Hex", "Hex", "NO")},
		},
		{
			"official 128-bit decrypt", "9f589f5cf6122c32b6bfec2f2ae8c35a",
			"00000000000000000000000000000000",
			core.Recipe{twoDec(z128, noIV, "ECB", "Hex", "Hex", "NO")},
		},
		// Consistency vector (Raw input, PKCS5, hex out).
		{
			"consistency vector", "TestData12345678",
			"8aed2d3a85dc3e0b663ba1fe1fdaf056771d591428af301d69fa1e227d083527",
			core.Recipe{twoEnc(z128, noIV, "ECB", "Raw", "Hex", "PKCS5")},
		},

		// Round trips: ECB across all key sizes.
		{
			"round trip ECB 128", "Hello, World!!!", "Hello, World!!!",
			core.Recipe{
				twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"),
				twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip ECB 192", "Testing Twofish with 192-bit key", "Testing Twofish with 192-bit key",
			core.Recipe{
				twoEnc(k192, noIV, "ECB", "Raw", "Hex", "PKCS5"),
				twoDec(k192, noIV, "ECB", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip ECB 256", "Testing Twofish with 256-bit key encryption", "Testing Twofish with 256-bit key encryption",
			core.Recipe{
				twoEnc(k256, noIV, "ECB", "Raw", "Hex", "PKCS5"),
				twoDec(k256, noIV, "ECB", "Hex", "Raw", "PKCS5"),
			},
		},

		// Round trips: CBC across all key sizes.
		{
			"round trip CBC 128", "The quick brown fox jumps over the lazy dog", "The quick brown fox jumps over the lazy dog",
			core.Recipe{
				twoEnc(k128, iv0, "CBC", "Raw", "Hex", "PKCS5"),
				twoDec(k128, iv0, "CBC", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip CBC 192", "Testing Twofish with 192-bit key in CBC mode", "Testing Twofish with 192-bit key in CBC mode",
			core.Recipe{
				twoEnc(k192, iv0, "CBC", "Raw", "Hex", "PKCS5"),
				twoDec(k192, iv0, "CBC", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip CBC 256", "Testing Twofish with 256-bit key in CBC mode", "Testing Twofish with 256-bit key in CBC mode",
			core.Recipe{
				twoEnc(k256, iv0, "CBC", "Raw", "Hex", "PKCS5"),
				twoDec(k256, iv0, "CBC", "Hex", "Raw", "PKCS5"),
			},
		},

		// Round trips: stream modes.
		{
			"round trip CFB", "Testing Twofish CFB mode encryption", "Testing Twofish CFB mode encryption",
			core.Recipe{
				twoEnc(twofishKey("deadbeefcafebabe0123456789abcdef", "Hex"), twofishKey("0102030405060708090a0b0c0d0e0f10", "Hex"), "CFB", "Raw", "Hex", "PKCS5"),
				twoDec(twofishKey("deadbeefcafebabe0123456789abcdef", "Hex"), twofishKey("0102030405060708090a0b0c0d0e0f10", "Hex"), "CFB", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip OFB", "Testing Twofish OFB mode encryption", "Testing Twofish OFB mode encryption",
			core.Recipe{
				twoEnc(k128, iv0, "OFB", "Raw", "Hex", "PKCS5"),
				twoDec(k128, iv0, "OFB", "Hex", "Raw", "PKCS5"),
			},
		},
		{
			"round trip CTR", "Testing Twofish CTR mode encryption", "Testing Twofish CTR mode encryption",
			core.Recipe{
				twoEnc(k128, twofishKey("00000000000000000000000000000001", "Hex"), "CTR", "Raw", "Hex", "PKCS5"),
				twoDec(k128, twofishKey("00000000000000000000000000000001", "Hex"), "CTR", "Hex", "Raw", "PKCS5"),
			},
		},

		// UTF8 key/IV round trip.
		{
			"round trip UTF8 key", "Secret message!", "Secret message!",
			core.Recipe{
				twoEnc(twofishKey("MySecretPassword", "UTF8"), twofishKey("InitVectorHere!!", "UTF8"), "CBC", "Raw", "Hex", "PKCS5"),
				twoDec(twofishKey("MySecretPassword", "UTF8"), twofishKey("InitVectorHere!!", "UTF8"), "CBC", "Hex", "Raw", "PKCS5"),
			},
		},

		// Varied input lengths (ECB, PKCS5).
		{"round trip 1 byte", "A", "A", core.Recipe{
			twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"), twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
		}},
		{"round trip 15 bytes", "123456789012345", "123456789012345", core.Recipe{
			twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"), twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
		}},
		{"round trip 16 bytes", "1234567890123456", "1234567890123456", core.Recipe{
			twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"), twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
		}},
		{"round trip 17 bytes", "12345678901234567", "12345678901234567", core.Recipe{
			twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"), twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
		}},
		{"round trip 32 bytes", "12345678901234567890123456789012", "12345678901234567890123456789012", core.Recipe{
			twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5"), twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5"),
		}},

		// Empty input passes through unchanged.
		{"encrypt empty", "", "", core.Recipe{twoEnc(k128, noIV, "ECB", "Raw", "Hex", "PKCS5")}},
		{"decrypt empty", "", "", core.Recipe{twoDec(k128, noIV, "ECB", "Hex", "Raw", "PKCS5")}},
	})
}

// TestTwofishErrors covers the key-length, IV-length, ciphertext-length, and
// padding validation paths.
func TestTwofishErrors(t *testing.T) {
	k128 := twofishKey("00112233445566778899aabbccddeeff", "Hex")
	iv0 := twofishKey("ffeeddccbbaa99887766554433221100", "Hex")

	cases := []struct {
		name string
		step core.RecipeOp
		in   string
		want string
	}{
		{
			"short key", twoEnc(twofishKey("0011", "Hex"), iv0, "CBC", "Raw", "Hex", "PKCS5"),
			"hi", "Invalid key length: 2 bytes",
		},
		{
			"short IV", twoEnc(k128, twofishKey("00", "Hex"), "CBC", "Raw", "Hex", "PKCS5"),
			"hi", "Invalid IV length: 1 bytes",
		},
		{
			"non-block ciphertext", twoDec(k128, noIVHex(), "ECB", "Hex", "Raw", "PKCS5"),
			"00112233", "Invalid ciphertext length: 4 bytes",
		},
		{
			"no-padding partial", twoEnc(k128, noIVHex(), "ECB", "Raw", "Hex", "NO"),
			"short", "No padding requested but input is not a 16-byte multiple.",
		},
		{
			"bad key encoding (decrypt)", twoDec(twofishKey("!!!", "Base64"), noIVHex(), "ECB", "Hex", "Raw", "PKCS5"),
			"00", "illegal base64 data",
		},
		{
			"bad IV encoding (encrypt)", twoEnc(k128, twofishKey("!!!", "Base64"), "CBC", "Raw", "Hex", "PKCS5"),
			"hi", "illegal base64 data",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := core.Recipe{c.step}.Execute(core.NewDish([]byte(c.in), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want error containing %q", err, c.want)
			}
		})
	}
}

// noIVHex is the empty Hex IV used by ECB cases.
func noIVHex() core.ToggleString { return twofishKey("", "Hex") }
