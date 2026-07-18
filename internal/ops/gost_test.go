package ops

// Tests for the GOST block-cipher operations (Encrypt, Decrypt, Key Wrap, Key
// Unwrap, Sign, Verify). Cases marked "fixture" are transcribed from
// ../CyberChef/tests/operations/tests/GOST.mjs; the rest were produced by the
// CyberChef-server oracle (which wraps the same @wavesenterprise crypto-gost-js
// engine this file ports). ASCII input + Hex output are used where possible so
// there is no Latin1/UTF-8 ambiguity.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const (
	gostMagma   = "GOST 28147 (1989)"
	gostMagma15 = "GOST R 34.12 (Magma, 2015)"
	gostKuz     = "GOST R 34.12 (Kuznyechik, 2015)"
	// k32 is a 32-byte hex key used for the oracle-derived vectors.
	k32 = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
)

// gostEncRecipe builds a GOST Encrypt/Decrypt recipe from its nine arguments.
func gostEncRecipe(op, key, keyOpt, iv, ivOpt, inputType, outputType, algo, sBox, block, meshing, padding string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: iv, Option: ivOpt},
		inputType, outputType, algo, sBox, block, meshing, padding,
	}}}
}

func TestGOSTEncryptFixtures(t *testing.T) {
	runCases(t, []opCase{
		// --- upstream fixtures ---
		{
			"GOST Encrypt: 1989 (fixture)", "Hello, World!", "f124ac5c0853870906dbaf9b56",
			gostEncRecipe("GOST Encrypt", "00112233", "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-SC", "OFB", "CP", "ZERO"),
		},
		{
			"GOST Encrypt: Kuznyechik (fixture)", "Hello, World!", "8673d490dfa4a66d5e3ff00ba316724f",
			gostEncRecipe("GOST Encrypt", "00112233", "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-SC", "CBC", "CP", "PKCS5"),
		},

		// --- Magma 1989, oracle-derived ---
		{
			"Magma1989 ECB NO NO", "Hello, World!", "ade5f4241b16656354fe7c0bad3e61ed",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "", "Hex", "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "NO"),
		},
		{
			"Magma1989 ECB NO ZERO", "Hello, World!", "ade5f4241b16656354fe7c0bad3e61ed",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "", "Hex", "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "ZERO"),
		},
		{
			"Magma1989 ECB NO BIT", "Hello, World!", "00621a54d611d199b0edc77af0b8b9f9",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "", "Hex", "Raw", "Hex", gostMagma, "E-Z", "ECB", "NO", "BIT"),
		},
		{
			"Magma1989 CFB", "Hello, World!", "9d329bae8b7f9b966571d8bc86",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Magma1989 OFB", "Hello, World!", "9d329bae8b7f9b96ff9bfc088d",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "OFB", "NO", "NO"),
		},
		{
			"Magma1989 CTR", "Hello, World!", "7ce52b26811965b274c9806a35",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "CTR", "NO", "NO"),
		},
		{
			"Magma1989 CBC ZERO", "Hello, World!", "3a3bc75dea5a2a5002c6461e4cf98837",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "CBC", "NO", "ZERO"),
		},

		// --- Magma 2015 ---
		{
			"Magma2015 ECB ZERO", "Hello, World!", "17e66a5ce5c7a5dd0b47cff89601ba9d",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "", "Hex", "Raw", "Hex", gostMagma15, "E-A", "ECB", "NO", "ZERO"),
		},
		{
			"Magma2015 CFB", "Hello, World!", "3cc8c562a48a85cc9f52c2b621",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma15, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Magma2015 CTR", "Hello, World!", "b3e066fb2071de9f60ff19c003",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233", "Hex", "Raw", "Hex", gostMagma15, "E-A", "CTR", "NO", "NO"),
		},

		// --- Kuznyechik ---
		{
			"Kuznyechik ECB ZERO", "Hello, World!", "c2b342fea105dc2ac37fa6f520ae14fc",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "", "Hex", "Raw", "Hex", gostKuz, "E-A", "ECB", "NO", "ZERO"),
		},
		{
			"Kuznyechik CBC PKCS5", "Hello, World!", "d065d4097747b4c1a6b9a0fe73bda984",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", "CBC", "NO", "PKCS5"),
		},
		{
			"Kuznyechik CFB", "Hello, World!", "7ff3d743babaa6901d70e1e2b3",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Kuznyechik OFB", "Hello, World!", "7ff3d743babaa6901d70e1e2b3",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", "OFB", "NO", "NO"),
		},
		{
			"Kuznyechik CTR", "Hello, World!", "06133a3190c05186992402c358",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostKuz, "E-A", "CTR", "NO", "NO"),
		},

		// Long IV (a multiple of the block) exercises the m>n feedback-register
		// path in CFB/OFB/CBC (Magma 2015, 16-byte IV over an 8-byte block).
		{
			"Magma2015 CFB long IV", "Hello, World! Longer text here", "3cc8c562a48a85cc86a304bbc5f9c5079e47cba0201727de1ffe926c4d42",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostMagma15, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Magma2015 OFB long IV", "Hello, World! Longer text here", "3cc8c562a48a85cc86a304bbc5f9c507894db3786f398bff2471a67086b0",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostMagma15, "E-A", "OFB", "NO", "NO"),
		},
		{
			"Magma2015 CBC long IV", "Hello, World! Longer text here", "37b553ed3ca6c0806e40ce5423dcc3b535d50649e0bb5ca11dba81edec31bb69",
			gostEncRecipe("GOST Encrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostMagma15, "E-A", "CBC", "NO", "ZERO"),
		},
	})
}

func TestGOSTDecryptFixtures(t *testing.T) {
	runCases(t, []opCase{
		// --- upstream fixtures (Raw output) ---
		{
			"GOST Decrypt: 1989 (fixture)", "f124ac5c0853870906dbaf9b56", "Hello, World!",
			gostEncRecipe("GOST Decrypt", "00112233", "Hex", "0011223344556677", "Hex", "Hex", "Raw", gostMagma, "E-SC", "OFB", "CP", "ZERO"),
		},
		{
			"GOST Decrypt: Kuznyechik (fixture)", "8673d490dfa4a66d5e3ff00ba316724f", "Hello, World!\x00\x00\x00",
			gostEncRecipe("GOST Decrypt", "00112233", "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Raw", gostKuz, "E-TEST", "CBC", "CP", "PKCS5"),
		},

		// --- round-trips of the encrypt vectors (Hex output for byte clarity) ---
		{
			"Magma1989 ECB ZERO (keeps padding)", "ade5f4241b16656354fe7c0bad3e61ed", "48656c6c6f2c20576f726c6421000000",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "", "Hex", "Hex", "Hex", gostMagma, "E-A", "ECB", "NO", "ZERO"),
		},
		{
			"Magma1989 ECB BIT (strips padding)", "00621a54d611d199b0edc77af0b8b9f9", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "", "Hex", "Hex", "Hex", gostMagma, "E-Z", "ECB", "NO", "BIT"),
		},
		{
			"Magma1989 CFB", "9d329bae8b7f9b966571d8bc86", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "0011223344556677", "Hex", "Hex", "Hex", gostMagma, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Magma1989 OFB", "9d329bae8b7f9b96ff9bfc088d", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "0011223344556677", "Hex", "Hex", "Hex", gostMagma, "E-A", "OFB", "NO", "NO"),
		},
		{
			"Magma1989 CTR", "7ce52b26811965b274c9806a35", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "0011223344556677", "Hex", "Hex", "Hex", gostMagma, "E-A", "CTR", "NO", "NO"),
		},
		{
			"Magma1989 CBC ZERO", "3a3bc75dea5a2a5002c6461e4cf98837", "48656c6c6f2c20576f726c6421000000",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "0011223344556677", "Hex", "Hex", "Hex", gostMagma, "E-A", "CBC", "NO", "ZERO"),
		},
		{
			"Magma2015 CTR", "b3e066fb2071de9f60ff19c003", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "00112233", "Hex", "Hex", "Hex", gostMagma15, "E-A", "CTR", "NO", "NO"),
		},
		{
			"Kuznyechik CBC PKCS5", "d065d4097747b4c1a6b9a0fe73bda984", "48656c6c6f2c20576f726c6421000000",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Hex", gostKuz, "E-A", "CBC", "NO", "PKCS5"),
		},
		{
			"Kuznyechik CFB", "7ff3d743babaa6901d70e1e2b3", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Hex", gostKuz, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Kuznyechik CTR", "06133a3190c05186992402c358", "48656c6c6f2c20576f726c6421",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "0011223344556677", "Hex", "Hex", "Hex", gostKuz, "E-A", "CTR", "NO", "NO"),
		},
		// Long IV (m>n) round-trips.
		{
			"Magma2015 CFB long IV", "3cc8c562a48a85cc86a304bbc5f9c5079e47cba0201727de1ffe926c4d42", "48656c6c6f2c20576f726c6421204c6f6e67657220746578742068657265",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Hex", gostMagma15, "E-A", "CFB", "NO", "NO"),
		},
		{
			"Magma2015 CBC long IV", "37b553ed3ca6c0806e40ce5423dcc3b535d50649e0bb5ca11dba81edec31bb69", "48656c6c6f2c20576f726c6421204c6f6e676572207465787420686572650000",
			gostEncRecipe("GOST Decrypt", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Hex", gostMagma15, "E-A", "CBC", "NO", "ZERO"),
		},
	})
}

// gostSignRecipe builds a GOST Sign recipe (macLength in bits).
func gostSignRecipe(key, keyOpt, iv, ivOpt, inputType, outputType, algo, sBox string, macLength int) core.Recipe {
	return core.Recipe{{Op: "GOST Sign", Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: iv, Option: ivOpt},
		inputType, outputType, algo, sBox, float64(macLength),
	}}}
}

func TestGOSTSignFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"GOST Sign (fixture)", "Hello, World!", "810d0c40e965",
			gostSignRecipe("00112233", "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-C", 48),
		},
		{
			"Sign Magma1989 mac32", "Hello, World!", "8fff5f28",
			gostSignRecipe(k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", 32),
		},
		{
			"Sign Magma1989 mac64", "Hello, World!", "8fff5f283d1ffe90",
			gostSignRecipe(k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", 64),
		},
		{
			"Sign Magma2015 mac32", "Hello, World!", "ea9fe2fb",
			gostSignRecipe(k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma15, "E-A", 32),
		},
		{
			"Sign Kuznyechik mac64", "Hello, World!", "a08041b830117c33",
			gostSignRecipe(k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", 64),
		},
	})
}

// gostVerifyRecipe builds a GOST Verify recipe.
func gostVerifyRecipe(key, keyOpt, iv, ivOpt, mac, macOpt, inputType, algo, sBox string) core.Recipe {
	return core.Recipe{{Op: "GOST Verify", Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: iv, Option: ivOpt},
		core.ToggleString{Value: mac, Option: macOpt},
		inputType, algo, sBox,
	}}}
}

func TestGOSTVerifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"GOST Verify (fixture)", "Hello, World!", "The signature matches",
			gostVerifyRecipe("00112233", "Hex", "00112233445566778899aabbccddeeff", "Hex", "42b77fb3d6f6bf04", "Hex", "Raw", gostKuz, "E-TEST"),
		},
		{
			"Verify matches (Magma1989)", "Hello, World!", "The signature matches",
			gostVerifyRecipe(k32, "Hex", "0011223344556677", "Hex", "8fff5f28", "Hex", "Raw", gostMagma, "E-A"),
		},
		{
			"Verify does not match", "Hello, World!", "The signature does not match",
			gostVerifyRecipe(k32, "Hex", "0011223344556677", "Hex", "deadbeef", "Hex", "Raw", gostMagma, "E-A"),
		},
	})
}

// gostKeyWrapRecipe builds a GOST Key Wrap/Unwrap recipe.
func gostKeyWrapRecipe(op, key, keyOpt, ukm, ukmOpt, inputType, outputType, algo, sBox, wrapping string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: ukm, Option: ukmOpt},
		inputType, outputType, algo, sBox, wrapping,
	}}}
}

func TestGOSTKeyWrapFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"GOST Key Wrap (fixture)", "Hello, World!123", "0bb706e92487fceef97589911faeb28200000000000000000000000000000000\r\n6b7bfd16",
			gostKeyWrapRecipe("GOST Key Wrap", "00112233", "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma15, "E-TEST", "CP"),
		},
		{
			"Key Wrap Magma1989 NO", "Hello, World!123", "ade5f4241b166563e2de46834f7209c100000000000000000000000000000000\r\n48667130",
			gostKeyWrapRecipe("GOST Key Wrap", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "NO"),
		},
		{
			"Key Wrap Magma1989 CP", "Hello, World!123", "8d1ece6cd0b3a3eb16a1da6b320a2be100000000000000000000000000000000\r\nc20b3db1",
			gostKeyWrapRecipe("GOST Key Wrap", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "CP"),
		},
		{
			"Key Wrap Magma1989 SC", "Hello, World!123", "ade5f4241b166563e2de46834f7209c100000000000000000000000000000000\r\n45a9bb30",
			gostKeyWrapRecipe("GOST Key Wrap", k32, "Hex", "0011223344556677", "Hex", "Raw", "Hex", gostMagma, "E-A", "SC"),
		},
		{
			"Key Wrap Kuznyechik NO", "Hello, World!123", "9d71559c6f2b44f3e21435fda953e67600000000000000000000000000000000\r\nbbc9f93923ef5177",
			gostKeyWrapRecipe("GOST Key Wrap", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", "NO"),
		},
		{
			"Key Wrap Kuznyechik SC", "Hello, World!123", "9d71559c6f2b44f3e21435fda953e67600000000000000000000000000000000\r\nc74b010ea85af287",
			gostKeyWrapRecipe("GOST Key Wrap", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Raw", "Hex", gostKuz, "E-A", "SC"),
		},
	})
}

func TestGOSTKeyUnwrapFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"GOST Key Unwrap (fixture)", "c8e58458a42d21974d50103d59b469f2c8e58458a42d21974d50103d59b469f2\r\na32a1575", "0123456789abcdef0123456789abcdef",
			gostKeyWrapRecipe("GOST Key Unwrap", "", "Hex", "00112233", "Latin1", "Hex", "Raw", gostMagma, "E-Z", "CP"),
		},
		// round-trips of the Key Wrap vectors with a proper 32-byte CEK.
		{
			"Key Unwrap Magma1989 NO", "a211ad0a933f617ff779418311e30982a211ad0a933f617ff779418311e3098285f90da6", "0123456789abcdef0123456789abcdef",
			gostKeyWrapRecipe("GOST Key Unwrap", k32, "Hex", "0011223344556677", "Hex", "Hex", "Raw", gostMagma, "E-A", "NO"),
		},
		{
			"Key Unwrap Magma1989 CP", "e0241d25cac43b42867d22580e9c01cbe0241d25cac43b42867d22580e9c01cbb8ed2bf5", "0123456789abcdef0123456789abcdef",
			gostKeyWrapRecipe("GOST Key Unwrap", k32, "Hex", "0011223344556677", "Hex", "Hex", "Raw", gostMagma, "E-A", "CP"),
		},
		{
			"Key Unwrap Kuznyechik NO", "cac72810a42a7b63406923509d4da5e4cac72810a42a7b63406923509d4da5e44822c66aa4fa042e", "0123456789abcdef0123456789abcdef",
			gostKeyWrapRecipe("GOST Key Unwrap", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Raw", gostKuz, "E-A", "NO"),
		},
		{
			"Key Unwrap Kuznyechik SC", "cac72810a42a7b63406923509d4da5e4cac72810a42a7b63406923509d4da5e4ed2219ed9e070b29", "0123456789abcdef0123456789abcdef",
			gostKeyWrapRecipe("GOST Key Unwrap", k32, "Hex", "00112233445566778899aabbccddeeff", "Hex", "Hex", "Raw", gostKuz, "E-A", "SC"),
		},
	})
}

func gts(v, o string) core.ToggleString { return core.ToggleString{Value: v, Option: o} }

// TestGOSTOpErrors covers the error paths reachable through the operations.
func TestGOSTOpErrors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
		wantErr         string
	}{
		{
			"iv wrong length (1989)", "GOST Encrypt", "hi",
			[]any{gts(k32, "Hex"), gts("00112233", "Hex"), "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "NO"},
			"Length of iv must be 64 bits",
		},
		{
			"iv wrong length (CTR 2015)", "GOST Encrypt", "hi",
			[]any{gts(k32, "Hex"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma15, "E-A", "CTR", "NO", "NO"},
			"Length of iv must be 32 bits",
		},
		{
			"iv not multiple (CBC 2015)", "GOST Encrypt", "hi",
			[]any{gts(k32, "Hex"), gts("001122334455667788990011", "Hex"), "Raw", "Hex", gostMagma15, "E-A", "CBC", "NO", "ZERO"},
			"Length of iv must be a multiple of 64 bits",
		},
		{
			"key not 4-aligned", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("", "Hex"), "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad base64 key", "GOST Encrypt", "hi",
			[]any{gts("!!!!", "Base64"), gts("", "Hex"), "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "NO"},
			"",
		},
		{
			"empty ukm", "GOST Key Wrap", "0123456789abcdef",
			[]any{gts(k32, "Hex"), gts("", "Hex"), "Raw", "Hex", gostMagma, "E-A", "NO"},
			"Length of ukm must be 64 bits",
		},
		{
			"unwrap wrong length", "GOST Key Unwrap", "abcd",
			[]any{gts(k32, "Hex"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "NO"},
			"Wrapping key size must be 36 bytes",
		},
		{
			"unwrap MAC mismatch", "GOST Key Unwrap", "000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts(k32, "Hex"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "NO"},
			"Error verify MAC of wrapping key",
		},
		{
			"Kuznyechik CP unsupported", "GOST Key Wrap", "0123456789abcdef0123456789abcdef",
			[]any{gts(k32, "Hex"), gts("00112233445566778899aabbccddeeff", "Hex"), "Raw", "Hex", gostKuz, "E-A", "CP"},
			"Incorrect input length. Must be a multiple of the block size.",
		},
		{
			"SC invalid magic", "GOST Key Wrap", "0123456789abcdef0123456789abcdef",
			[]any{gts("000000000000000000000000000000000000", "Hex"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "SC"},
			"Invalid magic number",
		},

		// Malformed key rejected in every block mode / op (keySchedule error).
		{
			"bad key ECB dec", "GOST Decrypt", "0011223344556677",
			[]any{gts("abc", "Latin1"), gts("", "Hex"), "Hex", "Raw", gostMagma, "E-A", "ECB", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key CFB enc", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "CFB", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key CFB dec", "GOST Decrypt", "0011223344556677",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "CFB", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key OFB", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "OFB", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key CTR 1989", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "CTR", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key CTR 2015", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("00112233", "Hex"), "Raw", "Hex", gostMagma15, "E-A", "CTR", "NO", "NO"},
			"multiple of 4",
		},
		{
			"bad key CBC enc", "GOST Encrypt", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "CBC", "NO", "ZERO"},
			"multiple of 4",
		},
		{
			"bad key CBC dec", "GOST Decrypt", "00112233445566770011223344556677",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "CBC", "NO", "ZERO"},
			"multiple of 4",
		},
		{
			"bad key Sign", "GOST Sign", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", float64(32)},
			"multiple of 4",
		},
		{
			"bad key Verify", "GOST Verify", "hi",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), gts("8fff5f28", "Hex"), "Raw", gostMagma, "E-A"},
			"multiple of 4",
		},
		{
			"bad key Wrap NO", "GOST Key Wrap", "0123456789abcdef",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "NO"},
			"multiple of 4",
		},
		{
			"bad key Wrap CP", "GOST Key Wrap", "0123456789abcdef",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "CP"},
			"multiple of 4",
		},
		{
			"bad key Unwrap NO", "GOST Key Unwrap", "000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "NO"},
			"multiple of 4",
		},

		// Malformed argument encodings (per-op decode error branches).
		{
			"bad base64 iv (enc)", "GOST Encrypt", "hi",
			[]any{gts(k32, "Hex"), gts("!!!!", "Base64"), "Raw", "Hex", gostMagma, "E-A", "CFB", "NO", "NO"},
			"",
		},
		{
			"bad base64 iv (sign)", "GOST Sign", "hi",
			[]any{gts(k32, "Hex"), gts("!!!!", "Base64"), "Raw", "Hex", gostMagma, "E-A", float64(32)},
			"",
		},
		{
			"bad base64 ukm (wrap)", "GOST Key Wrap", "0123456789abcdef",
			[]any{gts(k32, "Hex"), gts("!!!!", "Base64"), "Raw", "Hex", gostMagma, "E-A", "NO"},
			"",
		},
		{
			"bad base64 mac (verify)", "GOST Verify", "hi",
			[]any{gts(k32, "Hex"), gts("0011223344556677", "Hex"), gts("!!!!", "Base64"), "Raw", gostMagma, "E-A"},
			"",
		},
		{
			"bad base64 key (wrap)", "GOST Key Wrap", "0123456789abcdef",
			[]any{gts("!!!!", "Base64"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "NO"},
			"",
		},
		{
			"bad base64 key (sign)", "GOST Sign", "hi",
			[]any{gts("!!!!", "Base64"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", float64(32)},
			"",
		},
		{
			"bad base64 mac key (verify)", "GOST Verify", "hi",
			[]any{gts("!!!!", "Base64"), gts("0011223344556677", "Hex"), gts("8fff5f28", "Hex"), "Raw", gostMagma, "E-A"},
			"",
		},
		{
			"bad algo (unwrap)", "GOST Key Unwrap", "00",
			[]any{gts(k32, "Hex"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "NO"},
			"Wrapping key size",
		},
		{
			"bad key CP unwrap", "GOST Key Unwrap", "000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts("abc", "Latin1"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "CP"},
			"multiple of 4",
		},
		{
			"bad base64 key (dec)", "GOST Decrypt", "00112233445566770011223344556677",
			[]any{gts("!!!!", "Base64"), gts("", "Hex"), "Hex", "Raw", gostMagma, "E-A", "ECB", "NO", "NO"},
			"",
		},
		{
			"Sign bad iv length", "GOST Sign", "hi",
			[]any{gts(k32, "Hex"), gts("00112233", "Hex"), "Raw", "Hex", gostMagma, "E-A", float64(32)},
			"Length of iv must be 64 bits",
		},
		{
			"SC short packed kek", "GOST Key Wrap", "0123456789abcdef0123456789abcdef",
			[]any{gts("2201", "Hex"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "SC"},
			"Incorrect input length. Must be a multiple of the block size.",
		},
		{
			"SC unwrap bad packed kek", "GOST Key Unwrap", "00000000000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts("000000000000000000000000000000000000", "Hex"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "SC"},
			"Invalid magic number",
		},
		{
			"Kuznyechik CP unwrap unsupported", "GOST Key Unwrap", "00000000000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts(k32, "Hex"), gts("00112233445566778899aabbccddeeff", "Hex"), "Hex", "Raw", gostKuz, "E-A", "CP"},
			"Incorrect input length. Must be a multiple of the block size.",
		},
		{
			"bad base64 key (unwrap)", "GOST Key Unwrap", "000000000000000000000000000000000000000000000000000000000000000000000000",
			[]any{gts("!!!!", "Base64"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "NO"},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.args...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got error %q, want containing %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestGOSTRandomPadding round-trips RANDOM padding (whose output is
// non-deterministic): the decrypted prefix must equal the plaintext.
func TestGOSTRandomPadding(t *testing.T) {
	enc, err := runOp(t, "GOST Encrypt", "Hello, World!",
		gts(k32, "Hex"), gts("", "Hex"), "Raw", "Hex", gostMagma, "E-A", "ECB", "NO", "RANDOM")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, err := runOp(t, "GOST Decrypt", enc,
		gts(k32, "Hex"), gts("", "Hex"), "Hex", "Raw", gostMagma, "E-A", "ECB", "NO", "RANDOM")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !strings.HasPrefix(dec, "Hello, World!") {
		t.Fatalf("decrypted prefix = %q", dec)
	}
}

// TestGOSTLongInputRoundTrip exercises code paths that only fire on large
// inputs: CryptoPro key meshing (every 1024 bytes, 64- and 128-bit) and the CTR
// counter-carry branches (~256 blocks in). Fidelity to CyberChef was confirmed
// against the oracle for each configuration; the round-trip proves correctness.
func TestGOSTLongInputRoundTrip(t *testing.T) {
	cases := []struct {
		name, algo, iv, block, meshing string
		size                           int
	}{
		{"Magma1989 CFB CP meshing", gostMagma, "0011223344556677", "CFB", "CP", 1040},
		{"Kuznyechik CFB CP meshing", gostKuz, "00112233445566778899aabbccddeeff", "CFB", "CP", 1040},
		{"Magma1989 CTR carry", gostMagma, "0011223344556677", "CTR", "NO", 2600},
		{"Magma2015 CTR carry", gostMagma15, "00112233", "CTR", "NO", 2600},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plain := strings.Repeat("A", c.size)
			enc, err := runOp(t, "GOST Encrypt", plain,
				gts(k32, "Hex"), gts(c.iv, "Hex"), "Raw", "Hex", c.algo, "E-A", c.block, c.meshing, "NO")
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}
			dec, err := runOp(t, "GOST Decrypt", enc,
				gts(k32, "Hex"), gts(c.iv, "Hex"), "Hex", "Raw", c.algo, "E-A", c.block, c.meshing, "NO")
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if dec != plain {
				t.Fatalf("round-trip mismatch")
			}
		})
	}
}

// TestGOSTSignalComPackedKey builds a valid SignalCom packed master key and
// round-trips it through SC key wrapping, exercising unpackKeySC.
func TestGOSTSignalComPackedKey(t *testing.T) {
	c, err := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-A", keyWrapping: "SC"})
	if err != nil {
		t.Fatal(err)
	}
	clearKey := make([]byte, 32)
	for i := range clearKey {
		clearKey[i] = byte(i + 1)
	}
	mac := c.signMAC(clearKey, make([]byte, 32), nil) // 4-byte MAC of zeros under clearKey
	blob := append([]byte{0x22, 1}, mac...)           // magic, mask count, MAC
	blob = append(blob, clearKey...)                  // single mask == the key
	blobHex := fmt.Sprintf("%x", blob)

	cek := "0123456789abcdef0123456789abcdef"
	wrapped, err := runOp(t, "GOST Key Wrap", cek,
		gts(blobHex, "Hex"), gts("0011223344556677", "Hex"), "Raw", "Hex", gostMagma, "E-A", "SC")
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}
	unwrapped, err := runOp(t, "GOST Key Unwrap", wrapped,
		gts(blobHex, "Hex"), gts("0011223344556677", "Hex"), "Hex", "Raw", gostMagma, "E-A", "SC")
	if err != nil {
		t.Fatalf("unwrap: %v", err)
	}
	if unwrapped != cek {
		t.Fatalf("got %q, want %q", unwrapped, cek)
	}
}

// TestGOSTEngineBranches directly exercises defensive engine branches not
// reachable through the (option-validated) operation arguments.
func TestGOSTEngineBranches(t *testing.T) {
	if _, _, err := gostVersionBlock("bogus"); err == nil {
		t.Error("gostVersionBlock: expected error for unknown algorithm")
	}
	if _, err := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "ES", sBoxName: "BOGUS", block: "ECB"}); err == nil {
		t.Error("newGostCipher: expected error for unknown sBox")
	}
	if _, err := bitUnpad([]byte{0, 0, 0}); err == nil {
		t.Error("bitUnpad: expected error for missing marker")
	}
	// randomPad fills to a block multiple, preserving the data prefix.
	got := randomPad([]byte("hello"), 8)
	if len(got) != 8 || string(got[:5]) != "hello" {
		t.Errorf("randomPad = %v", got)
	}
	// verifyMAC returns false when the supplied MAC has the wrong length.
	mc, err := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "MAC", sBoxName: "E-A", macLength: 32})
	if err != nil {
		t.Fatal(err)
	}
	if mc.verifyMAC(splitHexToBytes(k32), []byte{0, 0}, []byte("hi"), nil) {
		t.Error("verifyMAC: expected false for length mismatch")
	}
	// unpackKeySC rejects a packed key whose MAC matches no S-box.
	sc, err := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-A", keyWrapping: "SC"})
	if err != nil {
		t.Fatal(err)
	}
	badBlob := append([]byte{0x22, 1, 0xff, 0xff, 0xff, 0xff}, make([]byte, 32)...)
	if _, err := sc.unpackKeySC(badBlob); err == nil || !strings.Contains(err.Error(), "Invalid main key MAC") {
		t.Errorf("unpackKeySC: got %v, want Invalid main key MAC", err)
	}
	// unpackKeySC: a mask count larger than the blob can hold.
	if _, err := sc.unpackKeySC([]byte{0x22, 9, 0, 0, 0, 0}); err == nil {
		t.Error("unpackKeySC: expected error for oversized mask count")
	}
	// unpackKeySC: MAC that matches an alternate S-box (E-B), not the primary (E-Z).
	ebMaker, _ := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-B", keyWrapping: "SC"})
	ck := make([]byte, 32)
	for i := range ck {
		ck[i] = byte(i)
	}
	altBlob := append(append([]byte{0x22, 1}, ebMaker.signMAC(ck, make([]byte, 32), nil)...), ck...)
	ezUnpacker, _ := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-Z", keyWrapping: "SC"})
	if got, err := ezUnpacker.unpackKeySC(altBlob); err != nil || string(got) != string(ck) {
		t.Errorf("unpackKeySC alt sBox: got %x err %v", got, err)
	}

	// Version-error propagation in the arg builders (unreachable via the
	// option-validated ops; exercised here through a bogus algorithm string).
	badAlgo := "BOGUS"
	esArgs := []any{gts("", "Hex"), gts("", "Hex"), "Raw", "Hex", badAlgo, "E-A", "ECB", "NO", "NO"}
	if _, _, err := buildGostES(esArgs); err == nil {
		t.Error("buildGostES: expected error for bogus algorithm")
	}
	kwArgs := []any{gts("", "Hex"), gts("0011223344556677", "Hex"), "Raw", "Hex", badAlgo, "E-A", "NO"}
	if _, _, err := buildGostKW(kwArgs); err == nil {
		t.Error("buildGostKW: expected error for bogus algorithm")
	}
	if _, _, err := gostMACCipher(gts("", "Hex"), gts("", "Hex"), badAlgo, "E-A", 32); err == nil {
		t.Error("gostMACCipher: expected error for bogus algorithm")
	}

	kek := splitHexToBytes(k32)
	cek := make([]byte, 32)
	// "UKM must be defined" — reachable only when the constructor never set a UKM.
	noUKM, _ := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-A", keyWrapping: "NO"})
	for _, tc := range []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"wrapKeyGOST nil ukm", func() ([]byte, error) { return noUKM.wrapKeyGOST(kek, cek) }},
		{"wrapKeyCP nil ukm", func() ([]byte, error) { return noUKM.wrapKeyCP(kek, cek) }},
		{"unwrapKeyGOST nil ukm", func() ([]byte, error) { return noUKM.unwrapKeyGOST(kek, make([]byte, 36)) }},
		{"unwrapKeyCP nil ukm", func() ([]byte, error) { return noUKM.unwrapKeyCP(kek, make([]byte, 36)) }},
	} {
		if _, err := tc.fn(); err == nil || !strings.Contains(err.Error(), "UKM must be defined") {
			t.Errorf("%s: got %v, want UKM must be defined", tc.name, err)
		}
	}

	// CryptoPro unwrap: wrong length, then a valid-length MAC mismatch.
	cp, _ := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-A", keyWrapping: "CP", ukm: splitHexToBytes("0011223344556677")})
	if _, err := cp.unwrapKeyCP(kek, make([]byte, 10)); err == nil || !strings.Contains(err.Error(), "Wrapping key size") {
		t.Errorf("unwrapKeyCP short: got %v", err)
	}
	if _, err := cp.unwrapKeyCP(kek, make([]byte, 36)); err == nil || !strings.Contains(err.Error(), "Error verify MAC") {
		t.Errorf("unwrapKeyCP mac: got %v", err)
	}
	// SignalCom unwrap: short input, then a MAC mismatch.
	scw, _ := newGostCipher(gostAlgo{version: 1989, length: 64, mode: "KW", sBoxName: "E-A", keyWrapping: "SC"})
	if _, err := scw.unwrapKeySC(kek, make([]byte, 4)); err == nil || !strings.Contains(err.Error(), "Invalid typed array length") {
		t.Errorf("unwrapKeySC short: got %v", err)
	}
	if _, err := scw.unwrapKeySC(kek, make([]byte, 36)); err == nil || !strings.Contains(err.Error(), "Invalid key MAC") {
		t.Errorf("unwrapKeySC mac: got %v", err)
	}
}
