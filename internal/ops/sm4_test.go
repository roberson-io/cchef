package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// SM4 test vectors from IETF draft-ribose-cfrg-sm4-09 (as used by CyberChef's
// SM4.mjs fixtures).
const (
	sm4TwoBlockPlain  = "aa aa aa aa bb bb bb bb cc cc cc cc dd dd dd dd ee ee ee ee ff ff ff ff aa aa aa aa bb bb bb bb"
	sm4FourBlockPlain = "aa aa aa aa aa aa aa aa bb bb bb bb bb bb bb bb cc cc cc cc cc cc cc cc dd dd dd dd dd dd dd dd ee ee ee ee ee ee ee ee ff ff ff ff ff ff ff ff aa aa aa aa aa aa aa aa bb bb bb bb bb bb bb bb"
	sm4Key1           = "01 23 45 67 89 ab cd ef fe dc ba 98 76 54 32 10"
	sm4Key2           = "fe dc ba 98 76 54 32 10 01 23 45 67 89 ab cd ef"
	sm4IV             = "00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f"

	sm4ECB1  = "5e c8 14 3d e5 09 cf f7 b5 17 9f 8f 47 4b 86 19 2f 1d 30 5a 7f b1 7d f9 85 f8 1c 84 82 19 23 04"
	sm4ECB2  = "c5 87 68 97 e4 a5 9b bb a7 2a 10 c8 38 72 24 5b 12 dd 90 bc 2d 20 06 92 b5 29 a4 15 5a c9 e6 00"
	sm4ECB1P = "5e c8 14 3d e5 09 cf f7 b5 17 9f 8f 47 4b 86 19 2f 1d 30 5a 7f b1 7d f9 85 f8 1c 84 82 19 23 04 00 2a 8a 4e fa 86 3c ca d0 24 ac 03 00 bb 40 d2"
	sm4ECB2P = "c5 87 68 97 e4 a5 9b bb a7 2a 10 c8 38 72 24 5b 12 dd 90 bc 2d 20 06 92 b5 29 a4 15 5a c9 e6 00 a2 51 49 20 93 f8 f6 42 89 b7 8d 6e 8a 28 b1 c6"
	sm4CBC1  = "78 eb b1 1c c4 0b 0a 48 31 2a ae b2 04 02 44 cb 4c b7 01 69 51 90 92 26 97 9b 0d 15 dc 6a 8f 6d"
	sm4CBC2  = "0d 3a 6d dc 2d 21 c6 98 85 72 15 58 7b 7b b5 9a 91 f2 c1 47 91 1a 41 44 66 5e 1f a1 d4 0b ae 38"
	sm4OFB1  = "ac 32 36 cb 86 1d d3 16 e6 41 3b 4e 3c 75 24 b7 1d 01 ac a2 48 7c a5 82 cb f5 46 3e 66 98 53 9b"
	sm4OFB2  = "5d cc cd 25 a8 4b a1 65 60 d7 f2 65 88 70 68 49 33 fa 16 bd 5c d9 c8 56 ca ca a1 e1 01 89 7a 97"
	sm4CFB1  = "ac 32 36 cb 86 1d d3 16 e6 41 3b 4e 3c 75 24 b7 69 d4 c5 4e d4 33 b9 a0 34 60 09 be b3 7b 2b 3f"
	sm4CFB2  = "5d cc cd 25 a8 4b a1 65 60 d7 f2 65 88 70 68 49 0d 9b 86 ff 20 c3 bf e1 15 ff a0 2c a6 19 2c c5"
	sm4CTR1  = "ac 32 36 cb 97 0c c2 07 91 36 4c 39 5a 13 42 d1 a3 cb c1 87 8c 6f 30 cd 07 4c ce 38 5c dd 70 c7 f2 34 bc 0e 24 c1 19 80 fd 12 86 31 0c e3 7b 92 6e 02 fc d0 fa a0 ba f3 8b 29 33 85 1d 82 45 14"
	sm4CTR2  = "5d cc cd 25 b9 5a b0 74 17 a0 85 12 ee 16 0e 2f 8f 66 15 21 cb ba b4 4c c8 71 38 44 5b c2 9e 5c 0a e0 29 72 05 d6 27 04 17 3b 21 23 9b 88 7f 6c 8c b5 b8 00 91 7a 24 88 28 4b de 9e 16 ea 29 06"
)

// sm4Recipe builds a single-op SM4 recipe with Hex key/IV/input/output.
func sm4Recipe(op, key, iv, mode string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		mode, "Hex", "Hex",
	}}}
}

// TestSM4Fixtures transcribes ../CyberChef/tests/operations/tests/SM4.mjs.
func TestSM4Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Encrypt ECB1 no padding", sm4TwoBlockPlain, sm4ECB1, sm4Recipe("SM4 Encrypt", sm4Key1, "", "ECB/NoPadding")},
		{"Encrypt ECB2 no padding", sm4TwoBlockPlain, sm4ECB2, sm4Recipe("SM4 Encrypt", sm4Key2, "", "ECB/NoPadding")},
		{"Encrypt ECB1 padding", sm4TwoBlockPlain, sm4ECB1P, sm4Recipe("SM4 Encrypt", sm4Key1, "", "ECB")},
		{"Encrypt ECB2 padding", sm4TwoBlockPlain, sm4ECB2P, sm4Recipe("SM4 Encrypt", sm4Key2, "", "ECB")},
		{"Encrypt CBC1", sm4TwoBlockPlain, sm4CBC1, sm4Recipe("SM4 Encrypt", sm4Key1, sm4IV, "CBC/NoPadding")},
		{"Encrypt CBC2", sm4TwoBlockPlain, sm4CBC2, sm4Recipe("SM4 Encrypt", sm4Key2, sm4IV, "CBC/NoPadding")},
		{"Encrypt OFB1", sm4TwoBlockPlain, sm4OFB1, sm4Recipe("SM4 Encrypt", sm4Key1, sm4IV, "OFB")},
		{"Encrypt OFB2", sm4TwoBlockPlain, sm4OFB2, sm4Recipe("SM4 Encrypt", sm4Key2, sm4IV, "OFB")},
		{"Encrypt CFB1", sm4TwoBlockPlain, sm4CFB1, sm4Recipe("SM4 Encrypt", sm4Key1, sm4IV, "CFB")},
		{"Encrypt CFB2", sm4TwoBlockPlain, sm4CFB2, sm4Recipe("SM4 Encrypt", sm4Key2, sm4IV, "CFB")},
		{"Encrypt CTR1", sm4FourBlockPlain, sm4CTR1, sm4Recipe("SM4 Encrypt", sm4Key1, sm4IV, "CTR")},
		{"Encrypt CTR2", sm4FourBlockPlain, sm4CTR2, sm4Recipe("SM4 Encrypt", sm4Key2, sm4IV, "CTR")},

		{"Decrypt ECB1", sm4ECB1, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key1, "", "ECB/NoPadding")},
		{"Decrypt ECB2", sm4ECB2, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key2, "", "ECB/NoPadding")},
		{"Decrypt CBC1", sm4CBC1, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key1, sm4IV, "CBC/NoPadding")},
		{"Decrypt CBC2", sm4CBC2, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key2, sm4IV, "CBC/NoPadding")},
		{"Decrypt OFB1 (symmetric)", sm4TwoBlockPlain, sm4OFB1, sm4Recipe("SM4 Decrypt", sm4Key1, sm4IV, "OFB")},
		{"Decrypt OFB2", sm4OFB2, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key2, sm4IV, "OFB")},
		{"Decrypt CFB1", sm4CFB1, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key1, sm4IV, "CFB")},
		{"Decrypt CFB2", sm4CFB2, sm4TwoBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key2, sm4IV, "CFB")},
		{"Decrypt CTR1", sm4CTR1, sm4FourBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key1, sm4IV, "CTR")},
		{"Decrypt CTR2", sm4CTR2, sm4FourBlockPlain, sm4Recipe("SM4 Decrypt", sm4Key2, sm4IV, "CTR")},
	})
}

// TestSM4RawAndPadding covers Raw input/output, PKCS#7 padding removal on
// decrypt, and empty input (all oracle-verified). The default key is the SM4.mjs
// KEY_1.
func TestSM4RawAndPadding(t *testing.T) {
	key := core.ToggleString{Value: sm4Key1, Option: "Hex"}
	noIV := core.ToggleString{Value: "", Option: "Hex"}

	// Raw input, Hex output, ECB with PKCS#7 padding.
	enc, err := runOp(t, "SM4 Encrypt", "The quick brown fox", key, noIV, "ECB", "Raw", "Hex")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	const wantCT = "08 8c 41 be ac 31 61 5d 33 c9 4f ac e6 24 04 a6 98 22 35 8e 25 22 aa 27 4d c7 2e 4e 92 63 7c 6b"
	if enc != wantCT {
		t.Fatalf("encrypt: got %q, want %q", enc, wantCT)
	}
	// Hex input, Raw output: padding is removed and the text recovered.
	dec, err := runOp(t, "SM4 Decrypt", wantCT, key, noIV, "ECB", "Hex", "Raw")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != "The quick brown fox" {
		t.Fatalf("decrypt: got %q", dec)
	}
	// A non-block-aligned CTR ciphertext exercises the stream-mode zero-padding
	// path on decrypt; the 5-byte plaintext round-trips.
	ctr, err := runOp(t, "SM4 Encrypt", "hello", key, hexTS(sm4IV), "CTR", "Raw", "Hex")
	if err != nil {
		t.Fatalf("CTR encrypt: %v", err)
	}
	back, err := runOp(t, "SM4 Decrypt", ctr, key, hexTS(sm4IV), "CTR", "Hex", "Raw")
	if err != nil || back != "hello" {
		t.Fatalf("CTR round trip: got %q, err %v", back, err)
	}

	// Empty input yields empty output for both operations.
	for _, op := range []string{"SM4 Encrypt", "SM4 Decrypt"} {
		empty, err := runOp(t, op, "", key, noIV, "ECB", "Raw", "Hex")
		if err != nil || empty != "" {
			t.Fatalf("%s empty: got %q, err %v", op, empty, err)
		}
	}
}

// TestSM4Unpad directly exercises the PKCS#7 padding checks that the fixtures
// (which all use NoPadding) don't reach: a too-large pad byte and an
// inconsistent pad run both error, and a valid pad is stripped.
func TestSM4Unpad(t *testing.T) {
	if _, err := sm4Unpad([]byte{1, 2, 3, 0xff}, false); err == nil {
		t.Fatalf("pad byte > 16: expected error")
	}
	if _, err := sm4Unpad([]byte{1, 2, 0x03, 0x03}, false); err == nil {
		t.Fatalf("inconsistent padding: expected error")
	}
	out, err := sm4Unpad([]byte{9, 9, 0x02, 0x02}, false)
	if err != nil || string(out) != string([]byte{9, 9}) {
		t.Fatalf("valid padding: got %v, err %v", out, err)
	}
}

// TestSM4Errors covers the operations' validation and padding error paths.
func TestSM4Errors(t *testing.T) {
	notMult := "aa bb cc" // 3 bytes, not a block multiple
	zeros := "00000000000000000000000000000000"
	cases := []struct {
		name, op string
		key, iv  core.ToggleString
		input    string
		mode     string
		want     string
	}{
		{"bad key length", "SM4 Encrypt", hexTS("00 11 22"), hexTS(sm4IV), sm4TwoBlockPlain, "CBC", "Invalid key length: 3 bytes"},
		{"decrypt bad key length", "SM4 Decrypt", hexTS("00 11 22"), hexTS(sm4IV), sm4CBC1, "CBC", "Invalid key length: 3 bytes"},
		{"bad IV length", "SM4 Encrypt", hexTS(sm4Key1), hexTS("00 11 22"), sm4TwoBlockPlain, "CBC", "Invalid IV length: 3 bytes"},
		{"encrypt ECB/NoPadding not a multiple", "SM4 Encrypt", hexTS(sm4Key1), hexTS(""), notMult, "ECB/NoPadding", "No padding requested in ECB mode but input is not a 16-byte multiple."},
		{"decrypt CBC not divisible", "SM4 Decrypt", hexTS(sm4Key1), hexTS(sm4IV), notMult, "CBC", "must be divisible into 16 byte blocks"},
		{"invalid PKCS#7 padding", "SM4 Decrypt", hexTS(sm4Key1), hexTS(""), zeros, "ECB", "Invalid PKCS#7 padding."},
		{"bad base64 key", "SM4 Encrypt", core.ToggleString{Value: "!!!", Option: "Base64"}, hexTS(sm4IV), sm4TwoBlockPlain, "CBC", ""},
		{"bad base64 IV", "SM4 Encrypt", hexTS(sm4Key1), core.ToggleString{Value: "!!!", Option: "Base64"}, sm4TwoBlockPlain, "CBC", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.key, c.iv, c.mode, "Hex", "Hex")
			if err == nil {
				t.Fatalf("expected error")
			}
			if c.want != "" && !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %q, want substring %q", err.Error(), c.want)
			}
		})
	}
}

// hexTS is a Hex-encoded toggle string.
func hexTS(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Hex"} }
