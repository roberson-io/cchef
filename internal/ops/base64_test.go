package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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

func TestExpandAlphRange(t *testing.T) {
	got := expandAlphRange("A-Za-z0-9+/=")
	want := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	if got != want {
		t.Fatalf("expandAlphRange = %q\nwant %q", got, want)
	}
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
