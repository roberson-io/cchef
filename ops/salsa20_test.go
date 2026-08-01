package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Salsa20/XSalsa20 fixtures transcribed from
// ../CyberChef/tests/operations/tests/{Salsa20,XSalsa20}.mjs (ECRYPT vectors).
func TestSalsa20Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Salsa20: ECRYPT Set 1 vector# 0", "00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00", "e3 be 8f dd 8b ec a2 e3 ea 8e f9 47 5b 29 a6 e7 00 39 51 e1 09 7a 5c 38 d2 3b 7a 5f ad 9f 68 44 b2 2c 97 55 9e 27 23 c7 cb bd 3f e4 fc 8d 9a 07 44 65 2a 83 e7 2a 9c 46 18 76 af 4d 7e f1 a1 17 8d a2 b7 4e ef 1b 62 83 e7 e2 01 66 ab ca e5 38 e9 71 6e 46 69 e2 81 6b 6b 20 c5 c3 56 80 20 01 cc 14 03 a9 a1 17 d1 2a 26 69 f4 56 36 6d 6e bb 0f 12 46 f1 26 51 50 f7 93 cd b4 b2 53 e3 48 ae 20 3d 89 bc 02 5e 80 2a 7e 0e 00 62 1d 70 aa 36 b7 e0 7c b1 e7 d5 b3 8d 5e 22 2b 8b 0e 4b 84 07 01 42 b1 e2 95 04 76 7d 76 82 48 50 32 0b 53 68 12 9f dd 74 e8 61 b4 98 e3 be 8d 16 f2 d7 d1 69 57 be 81 f4 7b 17 d9 ae 7c 4f f1 54 29 a7 3e 10 ac f2 50 ed 3a 90 a9 3c 71 13 08 a7 4c 62 16 a9 ed 84 cd 12 6d a7 f2 8e 8a bf 8b b6 35 17 e1 ca 98 e7 12 f4 fb 2e 1a 6a ed 9f dc 73 29 1f aa 17 95 82 11 c4 ba 2e bd 58 38 c6 35 ed b8 1f 51 3a 91 a2 94 e1 94 f1 c0 39 ae ec 65 7d ce 40 aa 7e 7c 0a f5 7c ac ef a4 0c 9f 14 b7 1a 4b 34 56 a6 3e 16 2e c7 d8 d1 0b 8f fb 18 10 d7 10 01 b6 18 2f 9f 73 da 53 b8 54 05 c1 1f 7b 2d 89 0f a8 ae 0c 7f 2e 92 6d 8a 98 c7 ec 4e 91 b6 51 20 e9 88 34 96 31 a7 00 c6 fa ce c3 47 1c b0 41 36 56 e7 5e 30 94 56 58 40 84 d7 e1 2c 5b 43 a4 1c 43 ed 9a 04 8a bd 9b 88 0d a6 5f 6a 66 5a 20 fe 7b 77 cd 29 2f e6 2c ae 64 4b 7f 7d f6 9f 32 bd b3 31 90 3e 65 05 ce 44 fd c2 93 92 0c 6a 9e c7 05 7e 23 df 7d ad 29 8f 82 dd f4 ef b7 fd c7 bf c6 22 69 6a fc fd 0c dd cc 83 c7 e7 7f 11 a6 49 d7 9a cd c3 35 4e 96 35 ff 13 7e 92 99 33 a0 bd 6f 53 77 ef a1 05 a3 a4 26 6b 7c 0d 08 9d 08 f1 e8 55 cc 32 b1 5b 93 78 4a 36 e5 6a 76 cc 64 bc 84 77",
			core.Recipe{{Op: "Salsa20", Args: []any{
				core.ToggleString{Value: "80:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00", Option: "Hex"},
				core.ToggleString{Value: "00:00:00:00:00:00:00:00", Option: "Hex"},
				0.0, "20", "Hex", "Hex",
			}}},
		},
		{
			"Salsa20: ECRYPT Set 6 vector# 3", "00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00", "71 da ee 51 42 d0 72 8b 41 b6 59 79 33 eb f4 67 e4 32 79 e3 09 78 67 70 78 94 16 02 62 9c bf 68 b7 3d 6b d2 c9 5f 11 8d 2b 3e 6e c9 55 da bb 6d c6 1c 41 43 bc 9a 9b 32 b9 9d be 68 66 16 6d c0 86 31 b7 d6 55 30 50 30 3d 72 52 c2 64 d3 a9 0d 26 c8 53 63 48 13 e0 9a d7 54 5a 6c e7 e8 4a 5d fc 75 ec 43 43 12 07 d5 31 99 70 b0 fa ad b0 e1 51 06 25 bb 54 37 2c 85 15 e2 8e 2a cc f0 a9 93 0a d1 5f 43 18 74 92 3d 2a 59 e2 0d 9f 2a 53 67 db a6 05 15 64 f1 50 28 7d eb b1 db 53 6f f9 b0 9a d9 81 f2 5e 50 10 d8 5d 76 ee 0c 30 5f 75 5b 25 e6 f0 93 41 e0 81 2f 95 c9 4f 42 ee ad 34 6e 81 f3 9c 58 c5 fa a2 c8 89 53 dc 0c ac 90 46 9d b2 06 3c b5 cd b2 2c 9e ae 22 af bf 05 06 fc a4 1d c7 10 b8 46 fb df e3 c4 68 83 dd 11 8f 3a 5e 8b 11 b6 af d9 e7 16 80 d8 66 65 57 30 1a 2d aa fb 94 96 c5 59 78 4d 35 a0 35 36 08 85 f9 b1 7b d7 19 19 77 de ea 93 2b 98 1e bd b2 90 57 ae 3c 92 cf ef f5 e6 c5 d0 cb 62 f2 09 ce 34 2d 4e 35 c6 96 46 cc d1 4e 53 35 0e 48 8b b3 10 a3 2f 8b 02 48 e7 0a cc 5b 47 3d f5 37 ce d3 f8 1a 01 4d 40 83 93 2b ed d6 2e d0 e4 47 b6 76 6c d2 60 4b 70 6e 9b 34 6c 44 68 be b4 6a 34 ec f1 61 0e bd 38 33 1d 52 bf 33 34 6a fe c1 5e ef b2 a7 69 9e 87 59 db 5a 1f 63 6a 48 a0 39 68 8e 39 de 34 d9 95 df 9f 27 ed 9e dc 8d d7 95 e3 9e 53 d9 d9 25 b2 78 01 05 65 ff 66 52 69 04 2f 05 09 6d 94 da 34 33 d9 57 ec 13 d2 fd 82 a0 06 62 83 d0 d1 ee b8 1b f0 ef 13 3b 7f d9 02 48 b8 ff b4 99 b2 41 4c d4 fa 00 30 93 ff 08 64 57 5a 43 74 9b f5 96 02 f2 6c 71 7f a9 6b 1d 05 76 97 db 08 eb c3 fa 66 4a 01 6a 67 dc ef 88 07 57 7c c3 a0 93 85 d3",
			core.Recipe{{Op: "Salsa20", Args: []any{
				core.ToggleString{Value: "0F:62:B5:08:5B:AE:01:54:A7:FA:4D:A0:F3:46:99:EC", Option: "Hex"},
				core.ToggleString{Value: "28:8F:F6:5D:C4:2B:92:F9", Option: "Hex"},
				0.0, "20", "Hex", "Hex",
			}}},
		},
		{
			"XSalsa20 custom vector", "00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00", "7c b6 60 af dd 9e c6 46 8f 57 dd 6d 24 33 f9 34 28 fd 82 cd 73 86 c5 47 1a 24 d8 ad 2a 52 5b 6e 5e ff 38 4f c7 ca a2 10 bb 3c 8f 3e 68 8f 4a 97 52 a5 46 df 8c 25 3f ef 17 a2 67 94 55 c7 a1 e1 83 db f5 d5 45 b0 f5 02 b9 8d e0 99 7a 66 ab 43 23 41 68 9f f3 97 dc 4f bc 1f 27 bd 1a 61 97 f5 dc 80 ff 19 05 16 c9 ed 14 f2 81 d1 ca 73 88 82 f6 d3 d2 fb 92 1e 2e f8 99 38 9e 0a 22 3b e7 ae 81 5a 04 86 5f 82 52 68 2f 6a d1 4f 98 ff 5f 08 23 cc 22 9d d2 22 9e 69 9d c2 1a 81 7d c9 54 bb 9b c9 0d ec 3b 9b d3 bf 20 9b 82 da f7 89 34 8a 5e 14 ec 54 2b 6d ee 8b 60 1e 7e 6d e3 c2 8a 2d 57 b6 25 e7 ea b3 43 d8 eb 20 85 b6 f6 82 09 58 99 35 20 44 22 60 60 61 d2 8d e9 8b ea 58 af bf ba ad 70 03 98 19 a0 c3 9a a8 63 94 47 5c d0 61 94 b0 17 ab c4 bb 28 b7 56 6d 3c 66 1c 76 f4 8a d3 a3 a2 9e d3 36 df 1f c6 8b 4f 44 2f 06 a3 58 0b ae c8 06 e2 e6 5d 39 ab 18 28 fe 80 18 12 69 2c 60 34 b5 0b f5 f3 3c 51 fc 0c fb 43 82 1e 3e 92 d6 b8 06 cf 00 16 e3 49 a0 34 83 20 f9 b0 53 7e ad ac 4a c1 36 5f cc fb be e2 ba 5a ad 1d 29 74 07 19 34 61 0e 9d ce 84 60 24 6a e6 8d ed 50 e0 20 44 26 d8 76 6d f2 da 4b 12 72 5a 85 c2 b1 07 04 f5 10 2e 3c 67 1c 5a fc 5b 46 0e 4d fb 39 b6 10 73 22 47 84 10 93 df 5f c8 92 7e 87 c3 0d 24 3a 48 b2 ad c2 56 3d a2 22 e9 02 9c 58 64 c6 d5 a5 f8 c6 54 99 1c 0f 6b f3 db ed 81 16 85 28 17 b0 eb 11 c7 05 9f f9 d8 fc 4a 1c 36 db 16 fd 38 d8 32 34 5b 8c 80 c6 51 21 1d 91 01 c5 8a 60 ad a4 39 33 d5 32 9a c1 f5 b2 ab 20 46 75 db 63 e0 bd d2 97 c0 e9 fc 1c d9 17 4a d1 3a db ea c2 8c 46 22 21 c3 5a bf 6c 1e cf 28 9c 8c 2f b2 0f",
			core.Recipe{{Op: "XSalsa20", Args: []any{
				core.ToggleString{Value: "00:01:02:03:04:05:06:07:08:09:0A:0B:0C:0D:0E:0F:10:11:12:13:14:15:16:17:18:19:1A:1B:1C:1D:1E:1F", Option: "Hex"},
				core.ToggleString{Value: "00:01:02:03:04:05:06:07:08:09:0A:0B:0C:0D:0E:0F:10:11:12:13:14:15:16:17", Option: "Hex"},
				0.0, "20", "Hex", "Hex",
			}}},
		},
	})
}

// TestSalsa20IntegerNonce covers the Integer nonce type (oracle-verified).
func TestSalsa20IntegerNonce(t *testing.T) {
	out, err := runOp(t, "Salsa20", "Hello, Salsa!",
		core.ToggleString{Value: "00112233445566778899aabbccddeeff", Option: "Hex"},
		core.ToggleString{Value: "305419896", Option: "Integer"},
		0.0, "20", "Raw", "Hex")
	if err != nil || out != "4d 16 51 59 e9 f8 8d 5d cb 3b 30 fb f2" {
		t.Fatalf("got %q %v", out, err)
	}
}

// TestSalsaRawRoundTrip covers the Raw output path: Salsa20 is symmetric, so
// encrypting then decrypting with the same parameters recovers the input.
func TestSalsaRawRoundTrip(t *testing.T) {
	key := core.ToggleString{Value: "00112233445566778899aabbccddeeff", Option: "Hex"}
	nonce := core.ToggleString{Value: "0011223344556677", Option: "Hex"}
	ct, err := runOp(t, "Salsa20", "The quick brown fox jumps over the lazy dog!!",
		key, nonce, 0.0, "20", "Raw", "Hex")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := runOp(t, "Salsa20", ct, key, nonce, 0.0, "20", "Hex", "Raw")
	if err != nil {
		t.Fatal(err)
	}
	if pt != "The quick brown fox jumps over the lazy dog!!" {
		t.Fatalf("round-trip: got %q", pt)
	}
}

// TestSalsaErrors covers key/nonce length and decode validation for both ops.
func TestSalsaErrors(t *testing.T) {
	badB64 := core.ToggleString{Value: "!!!bad!!!", Option: "Base64"}
	hexNonce8 := core.ToggleString{Value: "0011223344556677", Option: "Hex"}
	hexKey16 := core.ToggleString{Value: "00112233445566778899aabbccddeeff", Option: "Hex"}
	cases := []struct {
		op, sub string
		key     core.ToggleString
		nonce   core.ToggleString
	}{
		{"Salsa20", "Invalid key length", core.ToggleString{Value: "0011", Option: "Hex"}, hexNonce8},
		{"Salsa20", "Invalid nonce length", hexKey16, core.ToggleString{Value: "0011", Option: "Hex"}},
		{"Salsa20", "", badB64, hexNonce8},
		{"Salsa20", "", hexKey16, badB64},
		{"XSalsa20", "Invalid key length", core.ToggleString{Value: "0011", Option: "Hex"}, core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f1011121314151617", Option: "Hex"}},
		{"XSalsa20", "Invalid nonce length", core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}, hexNonce8},
	}
	for _, c := range cases {
		_, err := runOp(t, c.op, "test", c.key, c.nonce, 0.0, "20", "Raw", "Hex")
		if err == nil {
			t.Fatalf("%s %s: expected error", c.op, c.sub)
		}
		if c.sub != "" && !strings.Contains(err.Error(), c.sub) {
			t.Fatalf("%s: got %v, want %q", c.op, err, c.sub)
		}
	}
}

// TestJSSliceBytes covers the JS-slice clamping used for the XSalsa20 sub-nonces.
func TestJSSliceBytes(t *testing.T) {
	b := []byte{1, 2, 3, 4}
	for _, c := range []struct {
		start, end int
		want       []byte
	}{
		{0, 2, []byte{1, 2}},
		{2, 10, []byte{3, 4}}, // end clamped
		{10, 20, []byte{}},    // start clamped past end
		{3, 1, []byte{}},      // end < start
	} {
		if got := jsSliceBytes(b, c.start, c.end); string(got) != string(c.want) {
			t.Fatalf("jsSliceBytes(%d,%d)=%v want %v", c.start, c.end, got, c.want)
		}
	}
}
