package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Quoted Printable has no upstream fixture file; these cases are verified
// against the CyberChef-server oracle. To QP is driven through From Hex so the
// input bytes are exact; From QP output is shown as hex.
func TestToQuotedPrintable(t *testing.T) {
	enc := core.Recipe{
		{Op: "From Hex", Args: []any{"Auto"}},
		{Op: "To Quoted Printable", Args: []any{}},
	}
	runCases(t, []opCase{
		{"ascii unchanged", "48656c6c6f2c20576f726c6421", "Hello, World!", enc},
		{"equals escaped", "613d6220262063", "a=3Db & c", enc},
		{"utf-8 escaped", "636166c3a9", "caf=C3=A9", enc},
		{"trailing space escaped", "68656c6c6f20", "hello=20", enc},
		{"tab kept, trailing space escaped", "68656c6c6f0968656c6c6f20", "hello\thello=20", enc},
		{"CRLF bytes", "0d0a", "\r\n", enc},
		{
			"soft break at 76 (80 a bytes)", strings.Repeat("61", 80),
			strings.Repeat("a", 75) + "=\r\n" + strings.Repeat("a", 5), enc,
		},
		{
			"soft break keeps =XX intact (40 0xFF)", strings.Repeat("ff", 40),
			strings.Repeat("=FF", 24) + "=\r\n" + strings.Repeat("=FF", 16), enc,
		},
	})

	// Direct (ASCII) inputs exercising the soft-break branches.
	direct := core.Recipe{{Op: "To Quoted Printable", Args: []any{}}}
	runCases(t, []opCase{
		{"printable range bytes kept", "a > b ~ !", "a > b ~ !", direct},
		{
			"soft break at nearest space",
			"The quick brown fox jumps over the lazy dog and then runs away quickly today.",
			"The quick brown fox jumps over the lazy dog and then runs away quickly =\r\ntoday.",
			direct,
		},
		{
			"embedded newline normalized to CRLF",
			strings.Repeat("x", 40) + "\n" + strings.Repeat("y", 40),
			strings.Repeat("x", 40) + "\r\n" + strings.Repeat("y", 40),
			direct,
		},
		{"trailing space before newline escaped", "line1 \nline2", "line1=20\r\nline2", direct},
		{
			"soft break backs off an ASCII =XX escape",
			strings.Repeat("=", 30),
			strings.Repeat("=3D", 25) + "=\r\n" + strings.Repeat("=3D", 5), direct,
		},
		{
			"CRLF straddling the 76-char boundary",
			strings.Repeat("a", 75) + "\r\nzzz",
			strings.Repeat("a", 75) + "=\r\n\r\nzzz", direct,
		},
	})
}

// TestQPSoftBreaksDirect exercises qpSoftBreaks directly. To Quoted Printable
// normalises newlines to CRLF before calling it, so its bare-LF and some
// boundary-backoff branches are unreachable through Run — but the helper is a
// general utility and a future caller might not normalise first. Expected values
// are ground truth from mimelib's own _addQPSoftLinebreaks (run in node).
func TestQPSoftBreaksDirect(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"line ending in bare LF", "abc\n", "abc\n"},
		{"LF within the trailing margin", "abc\ndef", "abc\ndef"},
		{
			"76-char line ending in a complete =XX",
			strings.Repeat("x", 74) + "=FF" + strings.Repeat("y", 5),
			strings.Repeat("x", 74) + "=\r\n=FFyyyyy",
		},
		{"continuation-byte backoff to non-hex", "abcd=80", "abcd=80"},
		{
			"exactly 76 chars ending in a complete =XX",
			strings.Repeat("z", 73) + "=3D" + "abc",
			strings.Repeat("z", 73) + "=\r\n=3Dabc",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := qpSoftBreaks(c.in); got != c.want {
				t.Fatalf("qpSoftBreaks(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// From QP: decode QP text to bytes (shown as hex).
func TestFromQuotedPrintable(t *testing.T) {
	dec := func() core.Recipe {
		return core.Recipe{
			{Op: "From Quoted Printable", Args: []any{}},
			{Op: "To Hex", Args: []any{"None"}},
		}
	}
	runCases(t, []opCase{
		{"decode = escape", "a=3Db & c", "613d6220262063", dec()},
		{"decode utf-8", "caf=C3=A9", "636166c3a9", dec()},
		{"decode trailing space", "hello=20", "68656c6c6f20", dec()},
		{"decode soft break removed", "line1=\r\nline2", "6c696e65316c696e6532", dec()},
		{"trailing = removed", "incomplete=", "696e636f6d706c657465", dec()},
		{"invalid hex kept literal", "keep=ZZliteral", "6b6565703d5a5a6c69746572616c", dec()},
	})
}
