package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// rc2Recipe builds a single-op RC2 recipe.
func rc2Recipe(op, key, keyOpt, iv, ivOpt, inFmt, outFmt string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: keyOpt},
		core.ToggleString{Value: iv, Option: ivOpt},
		inFmt, outFmt,
	}}}
}

// RC2 has no upstream fixture file; these vectors were derived from the
// CyberChef-server oracle (node-forge RC2, 128 effective key bits, PKCS#7).
func TestRC2EncryptVectors(t *testing.T) {
	runCases(t, []opCase{
		{
			"RC2 Encrypt: ECB aligned", "Hello, RC2 World",
			"d410725739e237732998b2bdcb5f790f28033affcb6f7673",
			rc2Recipe("RC2 Encrypt", "0123456789abcdef", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: ECB unaligned 3", "RC2", "21c8fa6ec752de1c",
			rc2Recipe("RC2 Encrypt", "0123456789abcdef", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: ECB unaligned 7", "1234567", "f47a2e696ca2eef5",
			rc2Recipe("RC2 Encrypt", "00112233445566778899aabbccddeeff", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: UTF8 key", "The quick brown fox",
			"5528c3524c4eca2f0c67cd7632c76b4e47b02eeb2f7d4f6e",
			rc2Recipe("RC2 Encrypt", "secret!!", "UTF8", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: 1-byte key", "abcdefgh", "d44b1f07b98a6045a48e0e15f55e1e78",
			rc2Recipe("RC2 Encrypt", "ff", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: 16-byte key", "abcdefgh", "f1a4ac4e1b3fc88fabc56efbc40e6334",
			rc2Recipe("RC2 Encrypt", "000102030405060708090a0b0c0d0e0f", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: empty key", "abcdefgh", "4620ae59fcb50422f2bd0373ce496aa5",
			rc2Recipe("RC2 Encrypt", "", "Hex", "", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: hex input", "48656c6c6f", "01d2569f85a50a44",
			rc2Recipe("RC2 Encrypt", "0123456789abcdef", "Hex", "", "Hex", "Hex", "Hex"),
		},
		{
			"RC2 Encrypt: CBC", "Hello, RC2 World",
			"511dc1507de26ead4c943032df190f9c3ea1365d2a758498",
			rc2Recipe("RC2 Encrypt", "0123456789abcdef", "Hex", "0011223344556677", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: CBC unaligned", "RC2 cipher test",
			"6f5601c732e71a7697d73d6aa5c4bad6",
			rc2Recipe("RC2 Encrypt", "0123456789abcdef", "Hex", "8877665544332211", "Hex", "Raw", "Hex"),
		},
		{
			"RC2 Encrypt: CBC long key", "Testing RC2 with a longer message here",
			"346ece61dac4acfd70e46093691984cb1d9a92ff9ec564af3eb2cf07733c3c95499e1ddfa7504fc3",
			rc2Recipe("RC2 Encrypt", "0011223344556677889900112233445566778899", "Hex", "0123456789abcdef", "Hex", "Raw", "Hex"),
		},
	})
}

// TestRC2DecryptVectors covers decryption, including forge's lenient unpadding
// (bad padding is returned un-stripped; a non-block-multiple input decrypts only
// its whole blocks and skips unpadding — no error either way).
func TestRC2DecryptVectors(t *testing.T) {
	runCases(t, []opCase{
		{
			"RC2 Decrypt: ECB", "d410725739e237732998b2bdcb5f790f28033affcb6f7673",
			"Hello, RC2 World",
			rc2Recipe("RC2 Decrypt", "0123456789abcdef", "Hex", "", "Hex", "Hex", "Raw"),
		},
		{
			"RC2 Decrypt: CBC", "511dc1507de26ead4c943032df190f9c3ea1365d2a758498",
			"Hello, RC2 World",
			rc2Recipe("RC2 Decrypt", "0123456789abcdef", "Hex", "0011223344556677", "Hex", "Hex", "Raw"),
		},
		{
			"RC2 Decrypt: UTF8 key", "5528c3524c4eca2f0c67cd7632c76b4e47b02eeb2f7d4f6e",
			"The quick brown fox",
			rc2Recipe("RC2 Decrypt", "secret!!", "UTF8", "", "Hex", "Hex", "Raw"),
		},
		{
			"RC2 Decrypt: invalid padding kept", "1122334455667788", "18f17b18c42524ef",
			rc2Recipe("RC2 Decrypt", "0123456789abcdef", "Hex", "", "Hex", "Hex", "Hex"),
		},
		{
			"RC2 Decrypt: non-block-multiple", "1122334455667788aabbcc", "18f17b18c42524ef",
			rc2Recipe("RC2 Decrypt", "0123456789abcdef", "Hex", "", "Hex", "Hex", "Hex"),
		},
		{
			"RC2 Decrypt: too short", "112233", "",
			rc2Recipe("RC2 Decrypt", "0123456789abcdef", "Hex", "", "Hex", "Hex", "Hex"),
		},
	})
}

// TestRC2RoundTrip covers round-tripping (including empty input, which the oracle
// cannot bake but forge pads to a full block).
func TestRC2RoundTrip(t *testing.T) {
	for _, in := range []string{"", "x", "12345678", "The quick brown fox jumps"} {
		enc := core.Recipe{{Op: "RC2 Encrypt", Args: []any{
			core.ToggleString{Value: "cafebabe", Option: "Hex"},
			core.ToggleString{Value: "1122334455667788", Option: "Hex"},
			"Raw", "Hex",
		}}}
		ct, err := enc.Execute(core.NewDish([]byte(in), core.TypeString))
		if err != nil {
			t.Fatalf("encrypt %q: %v", in, err)
		}
		dec := core.Recipe{{Op: "RC2 Decrypt", Args: []any{
			core.ToggleString{Value: "cafebabe", Option: "Hex"},
			core.ToggleString{Value: "1122334455667788", Option: "Hex"},
			"Hex", "Raw",
		}}}
		out, err := dec.Execute(ct)
		if err != nil {
			t.Fatalf("decrypt %q: %v", in, err)
		}
		if out.String() != in {
			t.Fatalf("round-trip %q: got %q", in, out.String())
		}
	}
}

// TestRC2IVError covers the IV-length validation (0 for ECB or 8 for CBC).
func TestRC2IVError(t *testing.T) {
	for _, op := range []string{"RC2 Encrypt", "RC2 Decrypt"} {
		_, err := runOp(t, op, "abcdefgh",
			core.ToggleString{Value: "0123456789abcdef", Option: "Hex"},
			core.ToggleString{Value: "00112233", Option: "Hex"},
			"Hex", "Hex")
		if err == nil || !strings.Contains(err.Error(), "invalid IV length") {
			t.Fatalf("%s: got %v", op, err)
		}
	}
}

// TestRC2LongKey covers keys longer than 128 bytes (node-forge uses the first
// 128 bytes of the expanded register). Vector from the oracle.
func TestRC2LongKey(t *testing.T) {
	const key130 = "d1a5ac9a015fac2ef7b341673635512a1511f41fe37d111b267f039eec5d4f586ab9f1eb8f7d3388f4f9d586f66e99fd54080df2c446f0e58668b09c08a16dd0015f7e6bc5aeaf483724089e9252cc13b50951a6b69412522765cff4d780306e2f5052c9fd15b19a18c584d01363568198613f0c34e84409ef7938709a159ec29409"
	runCases(t, []opCase{
		{
			"RC2 Encrypt: >128-byte key", "abcdefgh", "dd32e418286e2d880ff81d65631ba58b",
			rc2Recipe("RC2 Encrypt", key130, "Hex", "", "Hex", "Raw", "Hex"),
		},
	})
}

// TestRC2DecodeErrors covers the key/IV Base64 decode error paths for both ops.
func TestRC2DecodeErrors(t *testing.T) {
	badB64 := core.ToggleString{Value: "!!!not base64!!!", Option: "Base64"}
	hexKey := core.ToggleString{Value: "0123456789abcdef", Option: "Hex"}
	for _, op := range []string{"RC2 Encrypt", "RC2 Decrypt"} {
		if _, err := runOp(t, op, "abcdefgh", badB64,
			core.ToggleString{Value: "", Option: "Hex"}, "Hex", "Hex"); err == nil {
			t.Fatalf("%s: bad base64 key should error", op)
		}
		if _, err := runOp(t, op, "abcdefgh", hexKey, badB64, "Hex", "Hex"); err == nil {
			t.Fatalf("%s: bad base64 IV should error", op)
		}
	}
}
