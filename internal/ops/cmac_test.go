package ops

// CMAC fixtures transcribed from ../CyberChef/tests/operations/tests/CMAC.mjs
// (NIST CSRC example values). Non-empty inputs are hex, decoded via From Hex as
// in the upstream recipes.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// cmacHex decodes a hex input then computes CMAC; cmacRaw runs CMAC directly
// (used for the empty-input vectors, which have no From Hex step upstream).
func cmacHex(key, algo string) core.Recipe {
	return core.Recipe{
		{Op: "From Hex", Args: []any{"None"}},
		{Op: "CMAC", Args: []any{core.ToggleString{Value: key, Option: "Hex"}, algo}},
	}
}

func cmacRaw(key, algo string) core.Recipe {
	return core.Recipe{{Op: "CMAC", Args: []any{core.ToggleString{Value: key, Option: "Hex"}, algo}}}
}

func TestCMAC(t *testing.T) {
	const (
		k128 = "2b7e151628aed2a6abf7158809cf4f3c"
		k192 = "8e73b0f7da0e6452c810f32b809079e562f8ead2522c6b7b"
		k256 = "603deb1015ca71be2b73aef0857d77811f352c073b6108d72d9810a30914dff4"
		tk1  = "0123456789abcdef23456789abcdef01456789abcdef0123"
		tk2  = "0123456789abcdef23456789abcdef010123456789abcdef"
		msg1 = "6bc1bee22e409f96e93d7e117393172a"
		msg2 = "6bc1bee22e409f96e93d7e117393172aae2d8a57"
		msg3 = "6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e5130c81c46a35ce411e5fbc1191a0a52eff69f2445df4f9b17ad2b417be66c3710"
		tmsg = "6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e51"
	)
	runCases(t, []opCase{
		// AES-128
		{"AES128 #1 (empty)", "", "bb1d6929e95937287fa37d129b756746", cmacRaw(k128, "AES")},
		{"AES128 #2", msg1, "070a16b46b4d4144f79bdd9dd04a287c", cmacHex(k128, "AES")},
		{"AES128 #3", msg2, "7d85449ea6ea19c823a7bf78837dfade", cmacHex(k128, "AES")},
		{"AES128 #4", msg3, "51f0bebf7e3b9d92fc49741779363cfe", cmacHex(k128, "AES")},
		// AES-192
		{"AES192 #1 (empty)", "", "d17ddf46adaacde531cac483de7a9367", cmacRaw(k192, "AES")},
		{"AES192 #2", msg1, "9e99a7bf31e710900662f65e617c5184", cmacHex(k192, "AES")},
		{"AES192 #3", msg2, "3d75c194ed96070444a9fa7ec740ecf8", cmacHex(k192, "AES")},
		{"AES192 #4", msg3, "a1d5df0eed790f794d77589659f39a11", cmacHex(k192, "AES")},
		// AES-256
		{"AES256 #1 (empty)", "", "028962f61b7bf89efc6b551f4667d983", cmacRaw(k256, "AES")},
		{"AES256 #2", msg1, "28a7023f452e8f82bd4bf28d8c37c35c", cmacHex(k256, "AES")},
		{"AES256 #3", msg2, "156727dc0878944a023c1fe03bad6d93", cmacHex(k256, "AES")},
		{"AES256 #4", msg3, "e1992190549f6ed5696a2c056c315410", cmacHex(k256, "AES")},
		// Triple DES (24-byte key, K1 K2 K3)
		{"TDES1 #1 (empty)", "", "7db0d37df936c550", cmacRaw(tk1, "Triple DES")},
		{"TDES1 #2", msg1, "30239cf1f52e6609", cmacHex(tk1, "Triple DES")},
		{"TDES1 #3", msg2, "6c9f3ee4923f6be2", cmacHex(tk1, "Triple DES")},
		{"TDES1 #4", tmsg, "99429bd0bf7904e5", cmacHex(tk1, "Triple DES")},
		// Triple DES (K1 == K3)
		{"TDES2 #1 (empty)", "", "79ce52a7f786a960", cmacRaw(tk2, "Triple DES")},
		{"TDES2 #2", msg1, "cc18a0b79af2413b", cmacHex(tk2, "Triple DES")},
		{"TDES2 #3", msg2, "c06d377ecd101969", cmacHex(tk2, "Triple DES")},
		{"TDES2 #4", tmsg, "9cd33580f9b64dfb", cmacHex(tk2, "Triple DES")},
	})
}

// A 16-byte Triple DES key is expanded to 24 bytes as K1‖K2‖K1, so it yields the
// same tag as the equivalent 24-byte key (here tk2 = 0123…ef 2345…01 0123…ef).
func TestCMACTripleDES16ByteKey(t *testing.T) {
	got, err := runOp(t, "CMAC", "\x6b\xc1\xbe\xe2\x2e\x40\x9f\x96\xe9\x3d\x7e\x11\x73\x93\x17\x2a",
		core.ToggleString{Value: "0123456789abcdef23456789abcdef01", Option: "Hex"}, "Triple DES")
	if err != nil || got != "cc18a0b79af2413b" {
		t.Fatalf("got %q, %v want cc18a0b79af2413b", got, err)
	}
}

// A key whose declared encoding cannot be decoded surfaces the decode error.
func TestCMACKeyDecodeError(t *testing.T) {
	_, err := runOp(t, "CMAC", "", core.ToggleString{Value: "@@@", Option: "Base64"}, "AES")
	if err == nil {
		t.Fatal("want error for undecodable key")
	}
}

// Key-length validation reproduces CyberChef's error text per algorithm.
func TestCMACKeyErrors(t *testing.T) {
	cases := []struct{ algo, key, wantErr string }{
		{"AES", "00112233445566778899aabbccddeeff01234567", "The key for AES must be either 16, 24, or 32 bytes (currently 20 bytes)"},
		{"Triple DES", "00112233445566778899aabbccddeeff01234567", "The key for Triple DES must be 16 or 24 bytes (currently 20 bytes)"},
	}
	for _, c := range cases {
		t.Run(c.algo, func(t *testing.T) {
			_, err := runOp(t, "CMAC", "", core.ToggleString{Value: c.key, Option: "Hex"}, c.algo)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v\nwant %q", err, c.wantErr)
			}
		})
	}
}
