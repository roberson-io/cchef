package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// aesTS builds a ToggleString argument value for the AES operation tests.
func aesTS(option, s string) core.ToggleString {
	return core.ToggleString{Value: s, Option: option}
}

// TestAESKeyWrapFixtures transcribes the RFC3394 vectors from
// CyberChef's tests/operations/tests/AESKeyWrap.mjs.
func TestAESKeyWrapFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"AES Key Wrap: 128-bit data, 128-bit KEK",
			"00112233445566778899aabbccddeeff",
			"1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Wrap: 128-bit data, 192-bit KEK",
			"00112233445566778899aabbccddeeff",
			"96778b25ae6ca435f92b5b97c050aed2468ab8a17ad84e5d",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f1011121314151617"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Wrap: 128-bit data, 256-bit KEK",
			"00112233445566778899aabbccddeeff",
			"64e8c3f9ce0f5ba263e9777905818a2a93c8191e7d6e8ae7",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Wrap: 192-bit data, 192-bit KEK",
			"00112233445566778899aabbccddeeff0001020304050607",
			"031d33264e15d33268f24ec260743edce1c6c7ddee725a936ba814915c6762d2",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f1011121314151617"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Wrap: 192-bit data, 256-bit KEK",
			"00112233445566778899aabbccddeeff0001020304050607",
			"a8f9bc1612c68b3ff6e6f4fbe30e71e4769c8b80a32cb8958cd5d17d6b254da1",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Wrap: 256-bit data, 256-bit KEK",
			"00112233445566778899aabbccddeeff000102030405060708090a0b0c0d0e0f",
			"28c9f404c4b810f4cbccb35cfb87f8263f5786e2d80ed326cbc7f0e71a99f43bfb988b9b7a02dd21",
			core.Recipe{{Op: "AES Key Wrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
	})
}

// TestAESKeyWrapErrors covers the validation error paths.
func TestAESKeyWrapErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
		kek   string
		iv    string
		want  string
	}{
		{
			"invalid KEK length", "00112233445566778899aabbccddeeff", "00010203040506070809", "a6a6a6a6a6a6a6a6",
			"KEK must be either 16, 24, or 32 bytes (currently 10 bytes)",
		},
		{
			"invalid IV length", "00112233445566778899aabbccddeeff", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6",
			"IV must be 8 bytes (currently 6 bytes)",
		},
		{
			"input not multiple of 8", "00112233445566778899aabbccddeeff0102", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6a6a6",
			"input must be 8n (n>=2) bytes (currently 18 bytes)",
		},
		{
			"input too short", "0011223344556677", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6a6a6",
			"input must be 8n (n>=2) bytes (currently 8 bytes)",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "AES Key Wrap", c.input, aesTS("Hex", c.kek), aesTS("Hex", c.iv), "Hex", "Hex")
			if err == nil || err.Error() != c.want {
				t.Fatalf("got err %v, want %q", err, c.want)
			}
		})
	}
}

// TestAESKeyWrapBadBase64 covers the KEK/IV Base64 decode error paths.
func TestAESKeyWrapBadBase64(t *testing.T) {
	bad := aesTS("Base64", "!!!not base64!!!")
	goodKEK := aesTS("Hex", "000102030405060708090a0b0c0d0e0f")
	goodIV := aesTS("Hex", "a6a6a6a6a6a6a6a6")
	input := "00112233445566778899aabbccddeeff"

	t.Run("bad KEK", func(t *testing.T) {
		if _, err := runOp(t, "AES Key Wrap", input, bad, goodIV, "Hex", "Hex"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("bad IV", func(t *testing.T) {
		if _, err := runOp(t, "AES Key Wrap", input, goodKEK, bad, "Hex", "Hex"); err == nil {
			t.Fatal("want error")
		}
	})
}

// TestAESKeyWrapRaw exercises Raw input/output round-tripping.
func TestAESKeyWrapRaw(t *testing.T) {
	// 16 raw bytes of key data, Raw output.
	out, err := runOp(t, "AES Key Wrap", strings.Repeat("A", 16),
		aesTS("Hex", "000102030405060708090a0b0c0d0e0f"), aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Raw", "Hex")
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	// Round-trip: unwrap the hex output back to the raw input.
	back, err := runOp(t, "AES Key Unwrap", out,
		aesTS("Hex", "000102030405060708090a0b0c0d0e0f"), aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Raw")
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	if back != strings.Repeat("A", 16) {
		t.Fatalf("round-trip got %q", back)
	}
}

// TestAESKeyUnwrapFixtures transcribes the RFC3394 vectors from
// CyberChef's tests/operations/tests/AESKeyWrap.mjs.
func TestAESKeyUnwrapFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"AES Key Unwrap: 128-bit data, 128-bit KEK",
			"1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5",
			"00112233445566778899aabbccddeeff",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Unwrap: 128-bit data, 192-bit KEK",
			"96778b25ae6ca435f92b5b97c050aed2468ab8a17ad84e5d",
			"00112233445566778899aabbccddeeff",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f1011121314151617"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Unwrap: 128-bit data, 256-bit KEK",
			"64e8c3f9ce0f5ba263e9777905818a2a93c8191e7d6e8ae7",
			"00112233445566778899aabbccddeeff",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Unwrap: 192-bit data, 192-bit KEK",
			"031d33264e15d33268f24ec260743edce1c6c7ddee725a936ba814915c6762d2",
			"00112233445566778899aabbccddeeff0001020304050607",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f1011121314151617"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Unwrap: 192-bit data, 256-bit KEK",
			"a8f9bc1612c68b3ff6e6f4fbe30e71e4769c8b80a32cb8958cd5d17d6b254da1",
			"00112233445566778899aabbccddeeff0001020304050607",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
		{
			"AES Key Unwrap: 256-bit data, 256-bit KEK",
			"28c9f404c4b810f4cbccb35cfb87f8263f5786e2d80ed326cbc7f0e71a99f43bfb988b9b7a02dd21",
			"00112233445566778899aabbccddeeff000102030405060708090a0b0c0d0e0f",
			core.Recipe{{Op: "AES Key Unwrap", Args: []any{
				aesTS("Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),
				aesTS("Hex", "a6a6a6a6a6a6a6a6"), "Hex", "Hex",
			}}},
		},
	})
}

// TestAESKeyUnwrapErrors covers the validation and integrity error paths.
func TestAESKeyUnwrapErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
		kek   string
		iv    string
		want  string
	}{
		{
			"invalid KEK length", "1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5", "00010203040506070809", "a6a6a6a6a6a6a6a6",
			"KEK must be either 16, 24, or 32 bytes (currently 10 bytes)",
		},
		{
			"invalid IV length", "1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6",
			"IV must be 8 bytes (currently 6 bytes)",
		},
		{
			"input not multiple of 8", "1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5e621", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6a6a6",
			"input must be 8n (n>=3) bytes (currently 26 bytes)",
		},
		{
			"input too short", "1fa68b0a8112b447aef34bd8fb5a7b82", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6a6a6",
			"input must be 8n (n>=3) bytes (currently 16 bytes)",
		},
		{
			"corrupted input", "1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe6", "000102030405060708090a0b0c0d0e0f", "a6a6a6a6a6a6a6a6",
			"IV mismatch",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "AES Key Unwrap", c.input, aesTS("Hex", c.kek), aesTS("Hex", c.iv), "Hex", "Hex")
			if err == nil || err.Error() != c.want {
				t.Fatalf("got err %v, want %q", err, c.want)
			}
		})
	}
}
