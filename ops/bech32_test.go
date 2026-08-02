package ops

import (
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestToBech32Fixtures transcribes the To Bech32 cases from
// CyberChef's tests/operations/tests/Bech32.mjs (official BIP-0173/BIP-0350
// vectors).
func TestToBech32Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"empty input", "", "bc1gmk9yu",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Raw bytes", "Generic", 0}}},
		},
		{
			"single byte", "A", "bc1gyufle22",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Raw bytes", "Generic", 0}}},
		},
		{
			"Hello", "Hello", "bc1fpjkcmr0gzsgcg",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Raw bytes", "Generic", 0}}},
		},
		{
			"custom HRP", "test", "custom1w3jhxaq593qur",
			core.Recipe{{Op: "To Bech32", Args: []any{"custom", "Bech32", "Raw bytes", "Generic", 0}}},
		},
		{
			"testnet HRP", "data", "tb1v3shgcg3x07jr",
			core.Recipe{{Op: "To Bech32", Args: []any{"tb", "Bech32", "Raw bytes", "Generic", 0}}},
		},
		{
			"Bech32m empty", "", "bc1a8xfp7",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32m", "Raw bytes", "Generic", 0}}},
		},
		{
			"Bech32m single byte", "A", "bc1gyf4040g",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32m", "Raw bytes", "Generic", 0}}},
		},
		{
			"Bech32m Hello", "Hello", "bc1fpjkcmr0a7qya2",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32m", "Raw bytes", "Generic", 0}}},
		},

		// Bitcoin SegWit encoding (Hex input).
		{
			"SegWit v0 P2WPKH", "751e76e8199196d454941c45d1b3a323f1433bd6",
			"bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Hex", "Bitcoin SegWit", 0}}},
		},
		{
			"SegWit v0 P2WSH testnet", "1863143c14c5166804bd19203356da136c985678cd4d27a1b8c6329604903262",
			"tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
			core.Recipe{{Op: "To Bech32", Args: []any{"tb", "Bech32", "Hex", "Bitcoin SegWit", 0}}},
		},
		{
			"Taproot v1", "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
			"bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj0",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32m", "Hex", "Bitcoin SegWit", 1}}},
		},
		{
			"SegWit v16", "751e", "bc1sw50qgdz25j",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32m", "Hex", "Bitcoin SegWit", 16}}},
		},
	})
}

// TestFromBech32Fixtures transcribes the From Bech32 cases.
func TestFromBech32Fixtures(t *testing.T) {
	jsonHello := "{\n  \"hrp\": \"bc\",\n  \"encoding\": \"Bech32\",\n  \"data\": \"48656c6c6f\"\n}"
	jsonAge := "{\n  \"hrp\": \"age\",\n  \"encoding\": \"Bech32\",\n  \"data\": \"b58fa5d7e3ac0bc732609082acc6834fd980ce4b8c4052d2bec2d5130acf8421\"\n}"

	runCases(t, []opCase{
		// Raw output.
		{
			"decode single byte", "bc1gyufle22", "A",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Raw"}}},
		},
		{
			"decode Hello", "bc1fpjkcmr0gzsgcg", "Hello",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Raw"}}},
		},
		{
			"auto-detect Bech32", "bc1fpjkcmr0gzsgcg", "Hello",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Raw"}}},
		},
		{
			"decode Bech32m Hello", "bc1fpjkcmr0a7qya2", "Hello",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Raw"}}},
		},
		{
			"auto-detect Bech32m", "bc1fpjkcmr0a7qya2", "Hello",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Raw"}}},
		},
		{
			"uppercase input", "BC1FPJKCMR0GZSGCG", "Hello",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Raw"}}},
		},
		{
			"custom HRP", "custom1w3jhxaq593qur", "test",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Raw"}}},
		},
		{
			"empty input", "", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
		},
		{
			"empty data part", "bc1gmk9yu", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},

		// HRP / JSON output.
		{
			"HRP: Hex output", "bc1fpjkcmr0gzsgcg", "bc: 48656c6c6f",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "HRP: Hex"}}},
		},
		{
			"JSON output", "bc1fpjkcmr0gzsgcg", jsonHello,
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "JSON"}}},
		},
		{
			"Hex output", "bc1fpjkcmr0gzsgcg", "48656c6c6f",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},

		// AGE key vectors.
		{
			"AGE public key 1", "age1kk86t4lr4s9uwvnqjzp2e35rflvcpnjt33q99547ct23xzk0ssss3ma49j",
			"age: b58fa5d7e3ac0bc732609082acc6834fd980ce4b8c4052d2bec2d5130acf8421",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "HRP: Hex"}}},
		},
		{
			"AGE private key 1", "AGE-SECRET-KEY-1Z5N23X54Y4E9NLMPNH6EZDQQX9V883TMKJ3ZJF5QXXMKNZ2RPFXQUQF74G",
			"age-secret-key-: 1526a89a95257259ff619df5913400315873c57bb4a229268031b76989430a4c",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "HRP: Hex"}}},
		},
		{
			"AGE public key 2", "age1nwt7gkq7udvalagqn7l8a4jgju7wtenkg925pvuqvn7cfcry6u2qkae4ad",
			"age: 9b97e4581ee359dff5009fbe7ed648973ce5e676415540b38064fd84e064d714",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "HRP: Hex"}}},
		},
		{
			"AGE private key 2", "AGE-SECRET-KEY-137M0YVE3CL6M8C4ET9L2KU67FPQHJZTW547QD5CK0R5A5T09ZGJSQGR9LX",
			"age-secret-key-: 8fb6f23331c7f5b3e2b9597eab735e484179096ea57c06d31678e9da2de51225",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "HRP: Hex"}}},
		},
		{
			"AGE public key 1 JSON", "age1kk86t4lr4s9uwvnqjzp2e35rflvcpnjt33q99547ct23xzk0ssss3ma49j", jsonAge,
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "JSON"}}},
		},

		// BIP-0173 vectors.
		{
			"BIP-0173 A12UEL5L", "A12UEL5L", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},
		{
			"BIP-0173 lowercase", "a12uel5l", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},
		{
			"BIP-0173 long HRP bio",
			"an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1tt5tgs", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},
		{
			"BIP-0173 abcdef data", "abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw",
			"abcdef: 00443214c74254b635cf84653a56d7c675be77df",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "HRP: Hex"}}},
		},
		{
			"BIP-0173 split HRP", "split1checkupstagehandshakeupstreamerranterredcaperred2y9e3w",
			"split: c5f38b70305f519bf66d85fb6cf03058f3dde463ecd7918f2dc743918f2d",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "HRP: Hex"}}},
		},
		{
			"BIP-0173 question mark HRP", "?1ezyfcl", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
		},

		// BIP-0350 (Bech32m) vectors.
		{
			"BIP-0350 A1LQFN3A", "A1LQFN3A", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Hex"}}},
		},
		{
			"BIP-0350 lowercase", "a1lqfn3a", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Hex"}}},
		},
		{
			"BIP-0350 long HRP",
			"an83characterlonghumanreadablepartthatcontainsthetheexcludedcharactersbioandnumber11sg7hg6", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Hex"}}},
		},
		{
			"BIP-0350 abcdef data", "abcdef1l7aum6echk45nj3s0wdvt2fg8x9yrzpqzd3ryx",
			"abcdef: ffbbcdeb38bdab49ca307b9ac5a928398a418820",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "HRP: Hex"}}},
		},
		{
			"BIP-0350 split HRP", "split1checkupstagehandshakeupstreamerranterredcaperredlc445v",
			"split: c5f38b70305f519bf66d85fb6cf03058f3dde463ecd7918f2dc743918f2d",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "HRP: Hex"}}},
		},
		{
			"BIP-0350 question mark HRP", "?1v759aa", "",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Hex"}}},
		},

		// Bitcoin scriptPubKey output.
		{
			"scriptPubKey v0 P2WPKH", "BC1QW508D6QEJXTDG4Y5R3ZARVARY0C5XW7KV8F3T4",
			"0014751e76e8199196d454941c45d1b3a323f1433bd6",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Bitcoin scriptPubKey"}}},
		},
		{
			"scriptPubKey v0 P2WSH", "tb1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sl5k7",
			"00201863143c14c5166804bd19203356da136c985678cd4d27a1b8c6329604903262",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Bitcoin scriptPubKey"}}},
		},
		{
			"scriptPubKey v1 Taproot", "bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj0",
			"512079be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Bitcoin scriptPubKey"}}},
		},
		{
			"scriptPubKey v16", "BC1SW50QGDZ25J", "6002751e",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Bitcoin scriptPubKey"}}},
		},
		{
			"scriptPubKey v2", "bc1zw508d6qejxtdg4y5r3zarvaryvaxxpcs",
			"5210751e76e8199196d454941c45d1b3a323",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Bitcoin scriptPubKey"}}},
		},
	})
}

// TestBech32RoundTrip covers the multi-op encode/decode round-trip fixtures.
func TestBech32RoundTrip(t *testing.T) {
	runCases(t, []opCase{
		{
			"Bech32 round-trip", "The quick brown fox jumps over the lazy dog",
			"The quick brown fox jumps over the lazy dog",
			core.Recipe{
				{Op: "To Bech32", Args: []any{"test", "Bech32", "Raw bytes", "Generic", 0}},
				{Op: "From Bech32", Args: []any{"Bech32", "Raw"}},
			},
		},
		{
			"Bech32m round-trip", "The quick brown fox jumps over the lazy dog",
			"The quick brown fox jumps over the lazy dog",
			core.Recipe{
				{Op: "To Bech32", Args: []any{"test", "Bech32m", "Raw bytes", "Generic", 0}},
				{Op: "From Bech32", Args: []any{"Bech32m", "Raw"}},
			},
		},
		{
			"binary data round-trip", "0001020304050607", "0001020304050607",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Bech32", Args: []any{"bc", "Bech32", "Raw bytes", "Generic", 0}},
				{Op: "From Bech32", Args: []any{"Bech32", "Hex"}},
			},
		},
		{
			"auto-detect round-trip", "CyberChef Bech32 Test", "CyberChef Bech32 Test",
			core.Recipe{
				{Op: "To Bech32", Args: []any{"cyberchef", "Bech32", "Raw bytes", "Generic", 0}},
				{Op: "From Bech32", Args: []any{"Auto-detect", "Raw"}},
			},
		},
		{
			"Bech32m auto-detect round-trip", "CyberChef Bech32m Test", "CyberChef Bech32m Test",
			core.Recipe{
				{Op: "To Bech32", Args: []any{"cyberchef", "Bech32m", "Raw bytes", "Generic", 0}},
				{Op: "From Bech32", Args: []any{"Auto-detect", "Raw"}},
			},
		},
	})
}

// TestBech32SegWitFallback covers decoding SegWit-HRP strings that are not
// valid SegWit addresses and fall back to a generic decode, plus the
// scriptPubKey output falling back to plain hex for a non-SegWit address.
// The SegWit vectors were crafted with an independent bech32 reference and
// confirmed against the CyberChef-server oracle.
func TestBech32SegWitFallback(t *testing.T) {
	runCases(t, []opCase{
		// words[0]<=16 but the witness program cannot be byte-decoded -> generic.
		{
			"segwit fallback (program decode throws)", "bc1qylhukqn", "01",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
		},
		// program decodes but has an invalid length for its version -> generic.
		{
			"segwit fallback (invalid program length)", "bc1qypqxpq9qcrssk7jqt9", "0102030405060708",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
		},
		// scriptPubKey output on a non-SegWit HRP falls back to plain hex.
		{
			"scriptPubKey non-segwit fallback", "abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw",
			"00443214c74254b635cf84653a56d7c675be77df",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Bitcoin scriptPubKey"}}},
		},
	})
}

// TestBech32Helpers directly exercises two defensive branches that no valid
// input reaches through the operations: the empty-input guard in bech32Decode
// (the op short-circuits empty input before calling it) and the invalid
// witness-version fallback in bech32ScriptPubKey (a decoded SegWit version is
// always 0-16).
func TestBech32Helpers(t *testing.T) {
	if _, err := bech32Decode("", "Auto-detect"); err == nil ||
		!strings.Contains(err.Error(), "Input cannot be empty.") {
		t.Fatalf("bech32Decode(\"\") err = %v, want empty-input error", err)
	}

	d := &bech32Decoded{hrp: "bc", data: []byte{17, 1, 2, 3}, encoding: "Bech32", witnessVersion: 0}
	if got := bech32ScriptPubKey(d); got != "11010203" {
		t.Fatalf("bech32ScriptPubKey out-of-range version = %q, want %q", got, "11010203")
	}
}

// TestBech32Errors covers the error fixtures, which CyberChef surfaces as the
// operation's error message.
func TestBech32Errors(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		recipe core.Recipe
		errSub string
	}{
		{
			"To Bech32 empty HRP", "test",
			core.Recipe{{Op: "To Bech32", Args: []any{"", "Bech32", "Raw bytes", "Generic", 0}}},
			"Human-Readable Part (HRP) cannot be empty.",
		},
		{
			"mixed case", "bc1FpjKcmr0gzsgcg",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"mixed case is not allowed",
		},
		{
			"no separator", "noseparator",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"no separator '1' found",
		},
		{
			"from empty HRP", "1qqqqqqqqqqqqqqqq",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"Human-Readable Part (HRP) cannot be empty",
		},
		{
			"invalid checksum", "bc1fpjkcmr0gzsgcx",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"checksum verification failed",
		},
		{
			"data too short", "bc1abc",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"data part is too short",
		},
		{
			"wrong encoding", "bc1fpjkcmr0gzsgcg",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32m", "Hex"}}},
			"Invalid Bech32m checksum.",
		},

		// Encode error paths.
		{
			"encode invalid HRP char", "test",
			core.Recipe{{Op: "To Bech32", Args: []any{"a b", "Bech32", "Raw bytes", "Generic", 0}}},
			"HRP contains invalid character at position 1",
		},
		{
			"encode segwit witness version > 16", "751e76e8199196d454941c45d1b3a323f1433bd6",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Hex", "Bitcoin SegWit", 17}}},
			"Invalid witness version: 17",
		},
		{
			"encode segwit program too short", "ab",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Hex", "Bitcoin SegWit", 0}}},
			"Invalid witness program length: 1",
		},
		{
			"encode segwit v0 wrong length", "010203",
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Hex", "Bitcoin SegWit", 0}}},
			"Invalid witness program length for v0: 3",
		},
		{
			"encode exceeds max length", strings.Repeat("A", 60),
			core.Recipe{{Op: "To Bech32", Args: []any{"bc", "Bech32", "Raw bytes", "Generic", 0}}},
			"exceeds maximum length of 90 characters",
		},

		// Decode error paths.
		{
			"decode exceeds max length", "bc1" + strings.Repeat("q", 88),
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"exceeds maximum length of 90 characters",
		},
		{
			"decode invalid HRP char", "a\x7f1qqqqqq",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"HRP contains invalid character at position 1",
		},
		{
			"decode invalid data char", "abc1bqqqqq",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"Invalid character 'b'",
		},
		{
			"decode explicit Bech32 checksum fail", "bc1fpjkcmr0gzsgcx",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
			"Invalid Bech32 checksum.",
		},

		// Padding failures: vectors crafted with an independent bech32 reference
		// and confirmed against the CyberChef-server oracle.
		{
			"decode padding too many bits", "abc1pzrlkcat6",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
			"Failed to decode data: Invalid padding: too many bits remaining",
		},
		{
			"decode padding non-zero bits", "abc1qrynsfqe",
			core.Recipe{{Op: "From Bech32", Args: []any{"Bech32", "Hex"}}},
			"Failed to decode data: Invalid padding: non-zero bits in padding",
		},
		// SegWit HRP where both the witness-program and the generic fallback
		// decode fail (crafted with an independent reference, oracle-confirmed).
		{
			"segwit both decodes fail", "bc1yh74eczee",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"Failed to decode data",
		},
		{
			"segwit invalid length then fallback fails", "bc12hst767gw",
			core.Recipe{{Op: "From Bech32", Args: []any{"Auto-detect", "Hex"}}},
			"Failed to decode data",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.recipe.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Fatalf("got err %v, want substring %q", err, c.errSub)
			}
		})
	}
}

// --- direct tests for the helpers extracted from bech32Decode ---

// TestBech32DecodeDataChars documents the data-character mapping: each Bech32
// charset character maps to its 5-bit value, and any other character errors.
func TestBech32DecodeDataChars(t *testing.T) {
	// Charset "qpzry9x8gf2tvdw0s3jn54khce6mua7l": q=0, p=1, z=2, r=3.
	got, err := bech32DecodeDataChars("qpzr", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Fatalf("got %v, want [0 1 2 3]", got)
	}
	// 'b', 'i', 'o' and '1' are excluded from the Bech32 charset.
	for _, bad := range []string{"qpb", "i", "o", "1"} {
		if _, err := bech32DecodeDataChars(bad, 0); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

// TestBech32DetectEncoding documents checksum verification and encoding
// selection, anchored on a known-good Bech32 string (decodes to "Hello").
func TestBech32DetectEncoding(t *testing.T) {
	data, err := bech32DecodeDataChars("fpjkcmr0gzsgcg", 2) // data part of bc1fpjkcmr0gzsgcg
	if err != nil {
		t.Fatalf("decoding data part: %v", err)
	}
	// Explicit and Auto both accept the valid Bech32 checksum.
	for _, enc := range []string{"Bech32", "Auto"} {
		if used, err := bech32DetectEncoding("bc", data, enc); err != nil || used != "Bech32" {
			t.Fatalf("%s: got (%q, %v), want (\"Bech32\", nil)", enc, used, err)
		}
	}
	// Demanding Bech32m rejects a Bech32 checksum.
	if _, err := bech32DetectEncoding("bc", data, "Bech32m"); err == nil {
		t.Fatal("expected checksum error when forcing Bech32m")
	}
}

// TestBech32DecodeWords documents word→byte conversion for both the plain path
// and the SegWit path (witness version split off, program validated).
func TestBech32DecodeWords(t *testing.T) {
	// Non-SegWit: bc1fpjkcmr0gzsgcg decodes to "Hello".
	data, _ := bech32DecodeDataChars("fpjkcmr0gzsgcg", 2)
	bytes, wv, err := bech32DecodeWords(data[:len(data)-6], false)
	if err != nil || wv != -1 || string(bytes) != "Hello" {
		t.Fatalf("plain: got (%q, %d, %v)", bytes, wv, err)
	}

	// SegWit v0: the BIP-0173 example address, program is a 20-byte hash.
	seg, _ := bech32DecodeDataChars("qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4", 2)
	sbytes, swv, err := bech32DecodeWords(seg[:len(seg)-6], true)
	if err != nil || swv != 0 || sbytes[0] != 0 {
		t.Fatalf("segwit: got version %d, first byte %d, err %v", swv, sbytes[0], err)
	}
	if hex.EncodeToString(sbytes[1:]) != "751e76e8199196d454941c45d1b3a323f1433bd6" {
		t.Fatalf("segwit program = %x", sbytes[1:])
	}
}

// TestBech32ParseParts documents the input validation and HRP/data split.
func TestBech32ParseParts(t *testing.T) {
	// Valid: all-lowercase and all-uppercase both parse (case is normalised).
	for _, s := range []string{"bc1fpjkcmr0gzsgcg", "BC1FPJKCMR0GZSGCG"} {
		hrp, dataPart, sep, err := bech32ParseParts(s)
		if err != nil || hrp != "bc" || dataPart != "fpjkcmr0gzsgcg" || sep != 2 {
			t.Fatalf("%s: got (%q, %q, %d, %v)", s, hrp, dataPart, sep, err)
		}
	}

	bad := []struct{ name, in, sub string }{
		{"empty", "", "cannot be empty"},
		{"too long", strings.Repeat("q", 91), "maximum length"},
		{"mixed case", "Bc1fpjkcmr0gzsgcg", "mixed case"},
		{"no separator", "bcfpjkcmr", "no separator"},
		{"empty hrp", "1fpjkcmr0gzsgcg", "cannot be empty"},
		{"data too short", "bc1qqqq", "too short"},
		{"bad hrp char", "b\x01c1fpjkcmr0gzsgcg", "invalid character"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			if _, _, _, err := bech32ParseParts(c.in); err == nil || !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("got err %v, want substring %q", err, c.sub)
			}
		})
	}
}
