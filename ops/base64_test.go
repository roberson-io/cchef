package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

const allBytesB64 = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8gISIjJCUmJygpKissLS4vMDEyMzQ1Njc4OTo7PD0+P0BBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWltcXV5fYGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6e3x9fn+AgYKDhIWGh4iJiouMjY6PkJGSk5SVlpeYmZqbnJ2en6ChoqOkpaanqKmqq6ytrq+wsbKztLW2t7i5uru8vb6/wMHCw8TFxsfIycrLzM3Oz9DR0tPU1dbX2Nna29zd3t/g4eLj5OXm5+jp6uvs7e7v8PHy8/T19vf4+fr7/P3+/w=="

// Cases transcribed from CyberChef tests/operations/tests/Base64.mjs.
func TestBase64Fixtures(t *testing.T) {
	std := []any{"A-Za-z0-9+/="}
	stdDecode := []any{"A-Za-z0-9+/=", true}
	runCases(t, []opCase{
		{
			"To Base64: nothing", "", "",
			core.Recipe{{Op: "To Base64", Args: std}},
		},
		{
			"To Base64: Hello, World!", "Hello, World!", "SGVsbG8sIFdvcmxkIQ==",
			core.Recipe{{Op: "To Base64", Args: std}},
		},
		{
			"To Base64: UTF-8", "ნუ პანიკას", "4YOc4YOjIOGDnuGDkOGDnOGDmOGDmeGDkOGDoQ==",
			core.Recipe{{Op: "To Base64", Args: std}},
		},
		{
			"To Base64: All bytes", allBytes(), allBytesB64,
			core.Recipe{{Op: "To Base64", Args: std}},
		},

		{
			"From Base64: nothing", "", "",
			core.Recipe{{Op: "From Base64", Args: stdDecode}},
		},
		{
			"From Base64: Hello, World!", "SGVsbG8sIFdvcmxkIQ==", "Hello, World!",
			core.Recipe{{Op: "From Base64", Args: stdDecode}},
		},
		{
			"From Base64: UTF-8", "4YOc4YOjIOGDnuGDkOGDnOGDmOGDmeGDkOGDoQ==", "ნუ პანიკას",
			core.Recipe{{Op: "From Base64", Args: stdDecode}},
		},
		{
			"From Base64: All bytes", allBytesB64, allBytes(),
			core.Recipe{{Op: "From Base64", Args: stdDecode}},
		},

		// Round-trip through both operations.
		{
			"Base64 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base64", Args: std},
				{Op: "From Base64", Args: stdDecode},
			},
		},
	})
}

func TestBase64Branches(t *testing.T) {
	// Non-strict (the default) is lenient: invalid or partial input decodes the
	// bytes it can and never errors, matching CyberChef's fromBase64.
	if out, err := runOp(t, "From Base64", "@@@@", stdBase64Alphabet, false, false); err != nil || out != "" {
		t.Fatalf("From Base64(@@@@, lenient) = %q, %v; want \"\", nil", out, err)
	}
	if out, err := runOp(t, "From Base64", "R", stdBase64Alphabet, true, false); err != nil || out != "" {
		t.Fatalf("From Base64(R, lenient) = %q, %v; want \"\", nil", out, err)
	}
	// A partial (non-multiple-of-4) group still yields its complete bytes.
	if out, err := runOp(t, "From Base64", "YQ", stdBase64Alphabet, false, false); err != nil || out != "a" {
		t.Fatalf("From Base64(YQ) = %q, %v; want \"a\", nil", out, err)
	}
	// Strict mode rejects non-alphabet characters and 4n+1 lengths.
	if _, err := runOp(t, "From Base64", "@@@@", stdBase64Alphabet, false, true); err == nil {
		t.Fatal("From Base64(@@@@, strict): expected a non-alphabet error")
	}
	if _, err := runOp(t, "From Base64", "R", stdBase64Alphabet, true, true); err == nil {
		t.Fatal("From Base64(R, strict): expected a 4n+1 length error")
	}
	// Strict mode also rejects misplaced padding and padding to a non-multiple of 4.
	if _, err := runOp(t, "From Base64", "A=AA", stdBase64Alphabet, false, true); err == nil {
		t.Fatal("From Base64(A=AA, strict): expected a misplaced-padding error")
	}
	if _, err := runOp(t, "From Base64", "AA=", stdBase64Alphabet, false, true); err == nil {
		t.Fatal("From Base64(AA=, strict): expected a padding-length error")
	}
	// An alphabet that is not 64 (or 65 with padding) characters is rejected.
	if _, err := runOp(t, "From Base64", "AAAA", "tooshort", false, false); err == nil {
		t.Fatal("From Base64(bad alphabet): expected an alphabet-length error")
	}
}

// --- direct tests for the helpers extracted from fromBase64 ---

// TestBuildBase64Alphabet documents alphabet expansion and padding detection.
func TestBuildBase64Alphabet(t *testing.T) {
	alphabet, idx, padIndex, err := buildBase64Alphabet("A-Za-z0-9+/=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alphabet) != 65 || padIndex != 64 {
		t.Fatalf("len=%d padIndex=%d, want 65 and 64", len(alphabet), padIndex)
	}
	if idx['A'] != 0 || idx['='] != 64 {
		t.Fatalf("idx['A']=%d idx['=']=%d", idx['A'], idx['='])
	}

	// A 64-character alphabet has no padding character.
	_, _, padIndex, err = buildBase64Alphabet("A-Za-z0-9+/")
	if err != nil || padIndex != -1 {
		t.Fatalf("unpadded: padIndex=%d, err=%v", padIndex, err)
	}

	// The wrong length is rejected.
	if _, _, _, err := buildBase64Alphabet("ABC"); err == nil {
		t.Fatal("expected error for a 3-character alphabet")
	}
}

// TestCheckStrictBase64Padding documents the strict length/padding rules.
func TestCheckStrictBase64Padding(t *testing.T) {
	alphabet, _, padIndex, _ := buildBase64Alphabet("A-Za-z0-9+/=")
	ok := func(s string) error { return checkStrictBase64Padding([]rune(s), alphabet, padIndex) }

	if err := ok("TWFu"); err != nil { // "Man", no padding, length 4
		t.Fatalf("valid unpadded: %v", err)
	}
	if err := ok("TQ=="); err != nil { // "M", two pad chars at the end
		t.Fatalf("valid padded: %v", err)
	}
	if err := ok("TWFuX"); err == nil { // length 4n+1
		t.Fatal("expected 4n+1 length error")
	}
	if err := ok("TW=u"); err == nil { // padding not at the end
		t.Fatal("expected misplaced-padding error")
	}
}

// TestBase64EmitQuad documents decoding one quad, including padding-skipped bytes.
func TestBase64EmitQuad(t *testing.T) {
	// "TWFu" -> T=19 W=22 F=5 u=46 -> "Man".
	if got := base64EmitQuad(nil, 19, 22, 5, 46, -1); string(got) != "Man" {
		t.Fatalf("got %q, want \"Man\"", got)
	}
	// A trailing pad in the fourth position drops the third output byte.
	got := base64EmitQuad(nil, 19, 22, 5, 64, 64)
	if len(got) != 2 {
		t.Fatalf("padded quad emitted %d bytes, want 2", len(got))
	}
}
