package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// asconEnc / asconDec build recipes with the CyberChef argument order:
// Key, Nonce, Associated Data, Input, Output.
func asconEnc(key, nonce, ad core.ToggleString, in, out string) core.Recipe {
	return core.Recipe{{Op: "Ascon Encrypt", Args: []any{key, nonce, ad, in, out}}}
}

func asconDec(key, nonce, ad core.ToggleString, in, out string) core.Recipe {
	return core.Recipe{{Op: "Ascon Decrypt", Args: []any{key, nonce, ad, in, out}}}
}

// Cases transcribed from ../CyberChef/tests/operations/tests/Ascon.mjs (only the
// Encrypt/Decrypt cases; the Hash/MAC ops are separate). These include official
// NIST SP 800-232 / ascon-c KAT vectors.
func TestAsconFixtures(t *testing.T) {
	k := func(s string) core.ToggleString { return core.ToggleString{Value: s, Option: "Hex"} }
	key := k("000102030405060708090a0b0c0d0e0f")
	nonceA := k("101112131415161718191a1b1c1d1e1f")
	nonceB := k("000102030405060708090a0b0c0d0e0f")
	zero := k("00000000000000000000000000000000")
	none := k("")
	meta := core.ToggleString{Value: "metadata", Option: "UTF8"}

	runCases(t, []opCase{
		// --- Encrypt: NIST ascon-c KAT vectors ---
		{
			"Ascon Encrypt: KAT Count=1 (empty PT, empty AD)", "",
			"4f9c278211bec9316bf68f46ee8b2ec6",
			asconEnc(key, nonceA, none, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: KAT Count=2 (empty PT, AD=0x30)", "",
			"cccb674fe18a09a285d6ab11b35675c0",
			asconEnc(key, nonceA, k("30"), "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: KAT Count=34 (PT=0x20, empty AD)", "\x20",
			"e8dd576aba1cd3e6fc704de02aedb79588",
			asconEnc(key, nonceA, none, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: KAT Count=341 (PT=10, AD=10)", "\x20\x21\x22\x23\x24\x25\x26\x27\x28\x29",
			"12042996da42b4536e5a0e64692cf6041ff8c367e1423253c84c",
			asconEnc(key, nonceA, k("30313233343536373839"), "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: KAT (PT=16, AD=16)", "\x20\x21\x22\x23\x24\x25\x26\x27\x28\x29\x2a\x2b\x2c\x2d\x2e\x2f",
			"6373ebb28be97c9bac090cf399c13ef13abfc0d209e8f4844c90814d13f32c59",
			asconEnc(key, nonceA, k("303132333435363738393a3b3c3d3e3f"), "Raw", "Hex"),
		},

		// --- Encrypt: general vectors ---
		{
			"Ascon Encrypt: basic encryption", "Hello",
			"af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0",
			asconEnc(key, nonceB, none, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: with associated data", "Hello",
			"351880c09f9dee12c20c4ba973066bc10dd26000b6",
			asconEnc(key, nonceB, meta, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: longer message", "test message",
			"9314a3fef6cc299a07b8c9e0f9e479ca0d1187e87345cf590adc572b",
			asconEnc(key, nonceB, none, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: empty plaintext", "",
			"4427d64b8e1e1451fc445960f0839bb0",
			asconEnc(key, nonceB, none, "Raw", "Hex"),
		},
		{
			"Ascon Encrypt: zero key and nonce", "Hello",
			"403281e117ebb087e2d9196552b2d123bccb7b5500",
			asconEnc(zero, zero, none, "Raw", "Hex"),
		},

		// --- Decrypt ---
		{
			"Ascon Decrypt: basic decryption", "af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0",
			"Hello",
			asconDec(key, nonceB, none, "Hex", "Raw"),
		},
		{
			"Ascon Decrypt: with associated data", "351880c09f9dee12c20c4ba973066bc10dd26000b6",
			"Hello",
			asconDec(key, nonceB, meta, "Hex", "Raw"),
		},
		{
			"Ascon Decrypt: longer message", "9314a3fef6cc299a07b8c9e0f9e479ca0d1187e87345cf590adc572b",
			"test message",
			asconDec(key, nonceB, none, "Hex", "Raw"),
		},

		// --- Round trip ---
		{
			"Ascon: encrypt then decrypt round-trip",
			"This is a test message for Ascon AEAD encryption!",
			"This is a test message for Ascon AEAD encryption!",
			core.Recipe{
				{Op: "Ascon Encrypt", Args: []any{key, nonceA, core.ToggleString{Value: "additional data", Option: "UTF8"}, "Raw", "Hex"}},
				{Op: "Ascon Decrypt", Args: []any{key, nonceA, core.ToggleString{Value: "additional data", Option: "UTF8"}, "Hex", "Raw"}},
			},
		},
	})
}

func TestAsconErrors(t *testing.T) {
	k := func(s string) core.ToggleString { return core.ToggleString{Value: s, Option: "Hex"} }
	key := k("000102030405060708090a0b0c0d0e0f")

	const (
		keyLen0  = "Invalid key length: 0 bytes.\n\nAscon-AEAD128 requires a key of exactly 16 bytes (128 bits)."
		keyLen8  = "Invalid key length: 8 bytes.\n\nAscon-AEAD128 requires a key of exactly 16 bytes (128 bits)."
		nonce0   = "Invalid nonce length: 0 bytes.\n\nAscon-AEAD128 requires a nonce of exactly 16 bytes (128 bits)."
		nonce12  = "Invalid nonce length: 12 bytes.\n\nAscon-AEAD128 requires a nonce of exactly 16 bytes (128 bits)."
		authFail = "Unable to decrypt: authentication failed. The ciphertext, key, nonce, or associated data may be incorrect or tampered with."
	)

	cases := []struct {
		name    string
		op      string
		key     core.ToggleString
		nonce   core.ToggleString
		ad      core.ToggleString
		in, out string
		input   string
		wantErr string
	}{
		{"encrypt no key", "Ascon Encrypt", k(""), key, k(""), "Raw", "Hex", "test message", keyLen0},
		{"encrypt short key", "Ascon Encrypt", k("0001020304050607"), key, k(""), "Raw", "Hex", "test message", keyLen8},
		{"encrypt no nonce", "Ascon Encrypt", key, k(""), k(""), "Raw", "Hex", "test message", nonce0},
		{"encrypt short nonce", "Ascon Encrypt", key, k("000102030405060708090a0b"), k(""), "Raw", "Hex", "test message", nonce12},
		{"decrypt no key", "Ascon Decrypt", k(""), key, k(""), "Hex", "Raw", "af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0", keyLen0},
		{"decrypt tampered ciphertext", "Ascon Decrypt", key, key, k(""), "Hex", "Raw", "bf14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0", authFail},
		{"decrypt wrong key", "Ascon Decrypt", k("ff0102030405060708090a0b0c0d0e0f"), key, k(""), "Hex", "Raw", "af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0", authFail},
		{"decrypt wrong AD", "Ascon Decrypt", key, key, core.ToggleString{Value: "wrong data", Option: "UTF8"}, "Hex", "Raw", "351880c09f9dee12c20c4ba973066bc10dd26000b6", authFail},
		// Short input (< 16 bytes): the tag can never match, so authentication fails.
		{"decrypt short input", "Ascon Decrypt", key, key, k(""), "Hex", "Raw", "00", authFail},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.key, c.nonce, c.ad, c.in, c.out)
			if err == nil {
				t.Fatalf("%s: expected error %q, got nil", c.op, c.wantErr)
			}
			if err.Error() != c.wantErr {
				t.Fatalf("%s: got error %q\nwant %q", c.op, err.Error(), c.wantErr)
			}
		})
	}
}

// asconHash / asconMAC build Hash and MAC recipes. Both take ArrayBuffer input.
func asconHash() core.Recipe {
	return core.Recipe{{Op: "Ascon Hash", Args: []any{}}}
}

func asconMAC(key core.ToggleString) core.Recipe {
	return core.Recipe{{Op: "Ascon MAC", Args: []any{key}}}
}

// Ascon Hash fixtures transcribed from Ascon.mjs (NIST SP 800-232 / ACVP).
func TestAsconHash(t *testing.T) {
	runCases(t, []opCase{
		{
			"Ascon Hash: empty", "",
			"0b3be5850f2f6b98caf29f8fdea89b64a1fa70aa249b8f839bd53baa304d92b2", asconHash(),
		},
		{
			"Ascon Hash: 0x50", "P",
			"b96da347d720272533a87f5a94a356155f49cdf7c0c10a3e6f346d8a2293e480", asconHash(),
		},
		{
			"Ascon Hash: Hello", "Hello",
			"c1beebe1251d562c4526d6b947cefb932998499424f6cd186e764aa0a36cddb7", asconHash(),
		},
		{
			"Ascon Hash: Hello, World!", "Hello, World!",
			"f40e1ce8d4272e628e9535193f196f4ff2a720b00f6380c5d6f16b975f3a7777", asconHash(),
		},
	})
}

// Ascon MAC fixtures transcribed from Ascon.mjs (NIST LWC_MAC_KAT_128_128).
func TestAsconMAC(t *testing.T) {
	key := core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}
	runCases(t, []opCase{
		{
			"Ascon MAC: KAT Count=1 (empty)", "",
			"eac9d74bbedf8bf1eba2862b26aa6d39", asconMAC(key),
		},
		{
			"Ascon MAC: KAT Count=2 (0x10)", "\x10",
			"e5be5b6dfb7b0e3eae00a070791947a8", asconMAC(key),
		},
		{
			"Ascon MAC: KAT Count=5 (0x10111213)", "\x10\x11\x12\x13",
			"727f6386405a52ad7ca0669a6a885294", asconMAC(key),
		},
	})
}

// Multi-block messages exercise the 8-byte-word absorb loop and the cross-lane
// permutation after four words. Vectors computed from the same vendored ascon.mjs
// CyberChef's Ascon MAC wraps (key = 000102…0f).
func TestAsconMACMultiBlock(t *testing.T) {
	key := core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}
	seq := func(start, n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(start + i)
		}
		return string(b)
	}
	cases := []struct{ name, input, want string }{
		{"one 8-byte word", seq(0x10, 8), "ae64d5de267ad795e29bfe13da776a53"},
		{"40 bytes (5 words)", seq(0x20, 40), "6627c28ee05ec0136ba9d9722171e756"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Ascon MAC", c.input, key)
			if err != nil || got != c.want {
				t.Fatalf("got %q, %v\nwant %q", got, err, c.want)
			}
		})
	}
}

// A key whose declared encoding cannot be decoded surfaces the decode error.
func TestAsconMACKeyDecodeError(t *testing.T) {
	_, err := runOp(t, "Ascon MAC", "msg", core.ToggleString{Value: "@@@", Option: "Base64"})
	if err == nil {
		t.Fatal("want error for undecodable key")
	}
}

// The MAC key must be exactly 16 bytes, matching CyberChef's error text.
func TestAsconMACKeyLength(t *testing.T) {
	_, err := runOp(t, "Ascon MAC", "test",
		core.ToggleString{Value: "0001020304050607", Option: "Hex"})
	want := "Invalid key length: 8 bytes.\n\nAscon-Mac requires a key of exactly 16 bytes (128 bits)."
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %q", err, want)
	}
}

// TestAsconArgDecodeErrors covers the argument-decoding error paths: an
// undecodable Key, Nonce or Associated Data (here invalid Base64) is surfaced as
// an error, as with the other cipher operations.
func TestAsconArgDecodeErrors(t *testing.T) {
	hexKey := core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}
	badB64 := core.ToggleString{Value: "@@@", Option: "Base64"}

	for _, op := range []string{"Ascon Encrypt", "Ascon Decrypt"} {
		t.Run(op+" bad key", func(t *testing.T) {
			if _, err := runOp(t, op, "", badB64, hexKey, hexKey, "Hex", "Hex"); err == nil {
				t.Fatal("expected error for undecodable key")
			}
		})
		t.Run(op+" bad nonce", func(t *testing.T) {
			if _, err := runOp(t, op, "", hexKey, badB64, hexKey, "Hex", "Hex"); err == nil {
				t.Fatal("expected error for undecodable nonce")
			}
		})
		t.Run(op+" bad AD", func(t *testing.T) {
			if _, err := runOp(t, op, "", hexKey, hexKey, badB64, "Hex", "Hex"); err == nil {
				t.Fatal("expected error for undecodable associated data")
			}
		})
	}
}

// TestAsconFormatCombinations exercises the Raw/Hex Input and Output options in
// the combinations the fixtures don't cover: encrypt with Raw output feeding a
// Raw-input decrypt, and a decrypt with Hex output.
func TestAsconFormatCombinations(t *testing.T) {
	key := core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}
	nonce := core.ToggleString{Value: "101112131415161718191a1b1c1d1e1f", Option: "Hex"}
	none := core.ToggleString{Value: "", Option: "Hex"}
	plaintext := "Ascon raw round-trip"

	// Encrypt with Raw output, then Raw-input decrypt back to the plaintext.
	ct, err := runOp(t, "Ascon Encrypt", plaintext, key, nonce, none, "Raw", "Raw")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := runOp(t, "Ascon Decrypt", ct, key, nonce, none, "Raw", "Raw")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("raw round-trip: got %q want %q", got, plaintext)
	}

	// Decrypt with Hex output returns the plaintext hex-encoded.
	hexOut, err := runOp(t, "Ascon Decrypt", "af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0",
		key, core.ToggleString{Value: "000102030405060708090a0b0c0d0e0f", Option: "Hex"}, none, "Hex", "Hex")
	if err != nil {
		t.Fatalf("decrypt hex out: %v", err)
	}
	if hexOut != "48656c6c6f" { // "Hello"
		t.Fatalf("decrypt hex out: got %q want %q", hexOut, "48656c6c6f")
	}
}
