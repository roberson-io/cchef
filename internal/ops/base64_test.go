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
		{"To Base64: nothing", "", "",
			core.Recipe{{Op: "To Base64", Args: std}}},
		{"To Base64: Hello, World!", "Hello, World!", "SGVsbG8sIFdvcmxkIQ==",
			core.Recipe{{Op: "To Base64", Args: std}}},
		{"To Base64: UTF-8", "ნუ პანიკას", "4YOc4YOjIOGDnuGDkOGDnOGDmOGDmeGDkOGDoQ==",
			core.Recipe{{Op: "To Base64", Args: std}}},
		{"To Base64: All bytes", allBytes(), allBytesB64,
			core.Recipe{{Op: "To Base64", Args: std}}},

		{"From Base64: nothing", "", "",
			core.Recipe{{Op: "From Base64", Args: stdDecode}}},
		{"From Base64: Hello, World!", "SGVsbG8sIFdvcmxkIQ==", "Hello, World!",
			core.Recipe{{Op: "From Base64", Args: stdDecode}}},
		{"From Base64: UTF-8", "4YOc4YOjIOGDnuGDkOGDnOGDmOGDmeGDkOGDoQ==", "ნუ პანიკას",
			core.Recipe{{Op: "From Base64", Args: stdDecode}}},
		{"From Base64: All bytes", allBytesB64, allBytes(),
			core.Recipe{{Op: "From Base64", Args: stdDecode}}},

		// Round-trip through both operations.
		{"Base64 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base64", Args: std},
				{Op: "From Base64", Args: stdDecode},
			}},
	})
}

func TestExpandAlphRange(t *testing.T) {
	got := expandAlphRange("A-Za-z0-9+/=")
	want := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	if got != want {
		t.Fatalf("expandAlphRange = %q\nwant %q", got, want)
	}
}
