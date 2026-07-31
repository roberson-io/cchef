package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// rc6Single builds a single-op RC6 recipe (Hex key/IV).
func rc6Single(op, key, iv, mode, inFmt, outFmt, padding string, w, r float64) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		mode, inFmt, outFmt, padding, w, r,
	}}}
}

// RC6 fixtures transcribed from
// ../CyberChef/tests/operations/tests/RC6.mjs (IETF draft-krovetz test vectors).
func TestRC6IETFVectors(t *testing.T) {
	runCases(t, []opCase{
		{
			"RC6-8/12/4 encrypt", "00010203", "aefc4612",
			rc6Single("RC6 Encrypt", "00010203", "", "ECB", "Hex", "Hex", "NO", 8, 12),
		},
		{
			"RC6-8/12/4 decrypt", "aefc4612", "00010203",
			rc6Single("RC6 Decrypt", "00010203", "", "ECB", "Hex", "Hex", "NO", 8, 12),
		},
		{
			"RC6-16/16/8 encrypt", "0001020304050607", "2ff0b68eaeffad5b",
			rc6Single("RC6 Encrypt", "0001020304050607", "", "ECB", "Hex", "Hex", "NO", 16, 16),
		},
		{
			"RC6-16/16/8 decrypt", "2ff0b68eaeffad5b", "0001020304050607",
			rc6Single("RC6 Decrypt", "0001020304050607", "", "ECB", "Hex", "Hex", "NO", 16, 16),
		},
		{
			"RC6-32/20/16 encrypt", "000102030405060708090a0b0c0d0e0f", "3a96f9c7f6755cfe46f00e3dcd5d2a3c",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b0c0d0e0f", "", "ECB", "Hex", "Hex", "NO", 32, 20),
		},
		{
			"RC6-32/20/16 decrypt", "3a96f9c7f6755cfe46f00e3dcd5d2a3c", "000102030405060708090a0b0c0d0e0f",
			rc6Single("RC6 Decrypt", "000102030405060708090a0b0c0d0e0f", "", "ECB", "Hex", "Hex", "NO", 32, 20),
		},
		{
			"RC6-64/24/24 encrypt",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			"c002de050bd55e5d36864ab9853338e6dc4a1326c6bdaaeb1bc9e4fd67886617",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b0c0d0e0f1011121314151617", "", "ECB", "Hex", "Hex", "NO", 64, 24),
		},
		{
			"RC6-64/24/24 decrypt",
			"c002de050bd55e5d36864ab9853338e6dc4a1326c6bdaaeb1bc9e4fd67886617",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
			rc6Single("RC6 Decrypt", "000102030405060708090a0b0c0d0e0f1011121314151617", "", "ECB", "Hex", "Hex", "NO", 64, 24),
		},
		{
			"RC6-128/28/32 encrypt",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
			"4ed87c64baffecd4303ee6a79aafaef575b351c024272be70a70b4a392cfc157dba52d529a79e83845bf43d67545383aed3dbf4f0d23640e44cbf6cdaa034dcb",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "", "ECB", "Hex", "Hex", "NO", 128, 28),
		},
		{
			"RC6-128/28/32 decrypt",
			"4ed87c64baffecd4303ee6a79aafaef575b351c024272be70a70b4a392cfc157dba52d529a79e83845bf43d67545383aed3dbf4f0d23640e44cbf6cdaa034dcb",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f",
			rc6Single("RC6 Decrypt", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "", "ECB", "Hex", "Hex", "NO", 128, 28),
		},
		// Non-standard word sizes.
		{
			"RC6-24/4/0 encrypt (empty key)", "000102030405060708090a0b", "0177982579be2ee3303269b9",
			rc6Single("RC6 Encrypt", "", "", "ECB", "Hex", "Hex", "NO", 24, 4),
		},
		{
			"RC6-24/4/0 decrypt (empty key)", "0177982579be2ee3303269b9", "000102030405060708090a0b",
			rc6Single("RC6 Decrypt", "", "", "ECB", "Hex", "Hex", "NO", 24, 4),
		},
		{
			"RC6-80/4/12 encrypt",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f2021222324252627",
			"26d9d6128601d06dec3817d401f1c0ff715473543875da417c2116d1e87c919a49311b00b4e17962",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b", "", "ECB", "Hex", "Hex", "NO", 80, 4),
		},
		{
			"RC6-80/4/12 decrypt",
			"26d9d6128601d06dec3817d401f1c0ff715473543875da417c2116d1e87c919a49311b00b4e17962",
			"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f2021222324252627",
			rc6Single("RC6 Decrypt", "000102030405060708090a0b", "", "ECB", "Hex", "Hex", "NO", 80, 4),
		},
		// Larger keys with w=32.
		{
			"RC6-32/20/24 (192-bit key)", "000102030405060708090a0b0c0d0e0f", "a68a14ff1342262a2bbd21f7966615eb",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b0c0d0e0f1011121314151617", "", "ECB", "Hex", "Hex", "NO", 32, 20),
		},
		{
			"RC6-32/20/32 (256-bit key)", "000102030405060708090a0b0c0d0e0f", "921c3ecd43d9426a90089334d67aea2e",
			rc6Single("RC6 Encrypt", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "", "ECB", "Hex", "Hex", "NO", 32, 20),
		},
	})
}

// rc6Round runs an encrypt→decrypt round-trip and checks it recovers the input.
func rc6Round(t *testing.T, input, key, keyOpt, iv, ivOpt, mode, padding string, w, r float64) {
	t.Helper()
	recipe := core.Recipe{
		{Op: "RC6 Encrypt", Args: []any{
			core.ToggleString{Value: key, Option: keyOpt},
			core.ToggleString{Value: iv, Option: ivOpt},
			mode, "Raw", "Hex", padding, w, r,
		}},
		{Op: "RC6 Decrypt", Args: []any{
			core.ToggleString{Value: key, Option: keyOpt},
			core.ToggleString{Value: iv, Option: ivOpt},
			mode, "Hex", "Raw", padding, w, r,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte(input), core.TypeString))
	if err != nil {
		t.Fatalf("%s w=%v: %v", mode, w, err)
	}
	if out.String() != input {
		t.Fatalf("%s w=%v round-trip: got %q want %q", mode, w, out.String(), input)
	}
}

// TestRC6RoundTrips covers the round-trip fixtures across modes, word sizes,
// keys and rounds.
func TestRC6RoundTrips(t *testing.T) {
	rc6Round(t, "Hello World!", "mysecret", "UTF8", "abcd", "UTF8", "CBC", "PKCS5", 8, 12)
	rc6Round(t, "The quick brown fox", "secretkey1234567", "UTF8", "initvec!", "UTF8", "CBC", "PKCS5", 16, 16)
	rc6Round(t, "The quick brown fox jumps over the lazy dog", "aabbccddeeff00112233445566778899", "Hex", "00112233445566778899aabbccddeeff", "Hex", "CBC", "PKCS5", 32, 20)
	rc6Round(t, "RC6 with 64-bit words is powerful!", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "Hex", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "Hex", "CBC", "PKCS5", 64, 24)
	rc6Round(t, "RC6 with 128-bit words provides massive block size for testing purposes!", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "Hex", "", "Hex", "ECB", "PKCS5", 128, 28)
	rc6Round(t, "CTR mode test message", "00112233445566778899aabbccddeeff", "Hex", "00000000000000000000000000000001", "Hex", "CTR", "PKCS5", 32, 20)
	rc6Round(t, "Testing custom rounds", "00112233445566778899aabbccddeeff", "Hex", "", "Hex", "ECB", "PKCS5", 32, 8)
	rc6Round(t, "1234567890123456", "00112233445566778899aabbccddeeff", "Hex", "", "Hex", "ECB", "PKCS5", 32, 20)
}

// TestRC6ValidationErrors covers the word-size, rounds and IV-length checks.
func TestRC6ValidationErrors(t *testing.T) {
	key := core.ToggleString{Value: "00112233445566778899aabbccddeeff", Option: "Hex"}
	noIV := core.ToggleString{Value: "", Option: "Hex"}
	cases := []struct {
		name string
		iv   core.ToggleString
		mode string
		w, r float64
		sub  string
	}{
		// Values outside the declared bounds are refused during coercion; these
		// are the in-range ones the operation itself has to catch.
		{"bad word size (not mult 8)", noIV, "ECB", 20, 20, "Invalid word size"},
		{"bad word size (fractional)", noIV, "ECB", 32.5, 20, "Invalid word size"},
		{"bad rounds (fractional)", noIV, "ECB", 32, 20.5, "Invalid number of rounds"},
		{"bad IV length", core.ToggleString{Value: "0011", Option: "Hex"}, "CBC", 32, 20, "Invalid IV length"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "RC6 Encrypt", "test", key, c.iv, c.mode, "Raw", "Hex", "PKCS5", c.w, c.r)
			if err == nil || !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("got %v, want %q", err, c.sub)
			}
		})
	}
}

// TestRC6StreamModes covers CFB/OFB/CTR with byte-exact oracle vectors (a
// 3-block message so the modes diverge after the first block).
func TestRC6StreamModes(t *testing.T) {
	const msg = "RC6 CFB and OFB differ on block 2!"
	const key = "00112233445566778899aabbccddeeff"
	const iv = "000102030405060708090a0b0c0d0e0f"
	runCases(t, []opCase{
		{
			"RC6 CFB", msg, "6e29beeff97557a6934e9d430c2d1c169cc66d650ece56cb015499ac7ad73c0e736f",
			core.Recipe{{Op: "RC6 Encrypt", Args: []any{
				core.ToggleString{Value: key, Option: "Hex"},
				core.ToggleString{Value: iv, Option: "Hex"},
				"CFB", "Raw", "Hex", "PKCS5", float64(32), float64(20),
			}}},
		},
		{
			"RC6 OFB", msg, "6e29beeff97557a6934e9d430c2d1c163488ccf4d101956ecbd1abca20245aecac10",
			core.Recipe{{Op: "RC6 Encrypt", Args: []any{
				core.ToggleString{Value: key, Option: "Hex"},
				core.ToggleString{Value: iv, Option: "Hex"},
				"OFB", "Raw", "Hex", "PKCS5", float64(32), float64(20),
			}}},
		},
		{
			"RC6 CTR", msg, "6e29beeff97557a6934e9d430c2d1c169e4c7678cbf8b10e9f38f9a1fbec0b7d5006",
			core.Recipe{{Op: "RC6 Encrypt", Args: []any{
				core.ToggleString{Value: key, Option: "Hex"},
				core.ToggleString{Value: iv, Option: "Hex"},
				"CTR", "Raw", "Hex", "PKCS5", float64(32), float64(20),
			}}},
		},
	})
	// Round-trip each stream mode.
	for _, mode := range []string{"CFB", "OFB"} {
		rc6Round(t, msg, key, "Hex", iv, "Hex", mode, "PKCS5", 32, 20)
	}
}

// TestRC6EdgeCases covers empty input, the ciphertext-length error, NO padding on
// a partial block, the default-rounds message branches, and key/IV decode errors.
func TestRC6EdgeCases(t *testing.T) {
	key := core.ToggleString{Value: "00112233445566778899aabbccddeeff", Option: "Hex"}
	noIV := core.ToggleString{Value: "", Option: "Hex"}

	// Empty input round-trips to empty.
	if out, err := runOp(t, "RC6 Encrypt", "", key, noIV, "ECB", "Raw", "Hex", "PKCS5", 32.0, 20.0); err != nil || out != "" {
		t.Fatalf("empty encrypt: %q %v", out, err)
	}
	if out, err := runOp(t, "RC6 Decrypt", "", key, noIV, "ECB", "Hex", "Raw", "PKCS5", 32.0, 20.0); err != nil || out != "" {
		t.Fatalf("empty decrypt: %q %v", out, err)
	}
	// Non-block-multiple ciphertext (ECB/CBC).
	if _, err := runOp(t, "RC6 Decrypt", "0011223344", key, noIV, "ECB", "Hex", "Raw", "NO", 32.0, 20.0); err == nil ||
		!strings.Contains(err.Error(), "Invalid ciphertext length") {
		t.Fatalf("ciphertext length: %v", err)
	}
	// NO padding on a partial block.
	if _, err := runOp(t, "RC6 Encrypt", "abc", key, noIV, "ECB", "Raw", "Hex", "NO", 32.0, 20.0); err == nil ||
		!strings.Contains(err.Error(), "No padding requested") {
		t.Fatalf("no padding: %v", err)
	}
	// A fractional round count at several word sizes exercises every
	// default-rounds branch, which the message quotes.
	for _, w := range []float64{8, 64, 256} {
		if _, err := runOp(t, "RC6 Encrypt", "x", key, noIV, "ECB", "Raw", "Hex", "PKCS5", w, 20.5); err == nil ||
			!strings.Contains(err.Error(), "Invalid number of rounds") {
			t.Fatalf("rounds w=%v: %v", w, err)
		}
	}
	// Base64 key / IV decode errors.
	badB64 := core.ToggleString{Value: "!!!bad!!!", Option: "Base64"}
	if _, err := runOp(t, "RC6 Encrypt", "x", badB64, noIV, "ECB", "Raw", "Hex", "PKCS5", 32.0, 20.0); err == nil {
		t.Fatal("bad base64 key should error")
	}
	if _, err := runOp(t, "RC6 Encrypt", "x", key, badB64, "CBC", "Raw", "Hex", "PKCS5", 32.0, 20.0); err == nil {
		t.Fatal("bad base64 IV should error")
	}
}

// TestRC6DeclaredBounds pins the Word Size and Rounds bounds CyberChef declares
// on the ingredient. They are checked during argument coercion, so an
// out-of-range value is reported with the bound's message rather than the
// operation's own; the operation's message is what an in-range but otherwise
// invalid value gets.
func TestRC6DeclaredBounds(t *testing.T) {
	for _, opName := range []string{"RC6 Encrypt", "RC6 Decrypt"} {
		op, _ := core.Default.Get(opName)
		const wordSize, rounds = 6, 7
		for _, tc := range []struct {
			name  string
			index int
			value float64
			want  string
		}{
			{"word size below 8", wordSize, 7, "Word Size must be greater than or equal to 8."},
			{"word size above 256", wordSize, 257, "Word Size must be less than or equal to 256."},
			{"rounds below 1", rounds, 0, "Rounds must be greater than or equal to 1."},
			{"rounds above 255", rounds, 256, "Rounds must be less than or equal to 255."},
		} {
			t.Run(opName+"/"+tc.name, func(t *testing.T) {
				args := core.DefaultArgs(op.Args())
				args[tc.index] = tc.value
				_, err := core.CoerceArgs(op.Args(), args)
				if err == nil {
					t.Fatalf("%v was accepted", tc.value)
				}
				if err.Error() != tc.want {
					t.Errorf("got %q, want %q", err.Error(), tc.want)
				}
			})
		}
	}
}
