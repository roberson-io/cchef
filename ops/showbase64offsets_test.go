package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestShowBase64OffsetsPlain covers the plain (Show variable chars = false) output:
// the three static Base64 sections, one per line. Oracle-verified (no upstream
// fixtures).
func TestShowBase64OffsetsPlain(t *testing.T) {
	plain := func(format string) core.Recipe {
		return core.Recipe{{Op: "Show Base64 offsets", Args: []any{"A-Za-z0-9+/=", false, format}}}
	}
	runCases(t, []opCase{
		{"one byte", "a", "Y\n\nh", plain("Raw")},
		{"two bytes", "ab", "YW\nFi\nhY", plain("Raw")},
		{"three bytes no padding", "abc", "YWJj\nFiY\nhYm", plain("Raw")},
		{"four bytes", "test", "dGVzd\nRlc3\n0ZXN0", plain("Raw")},
		{"six bytes", "hello!", "aGVsbG8h\nhlbGxvI\noZWxsby", plain("Raw")},
		// Base64 input format decodes first: "dGVzdA==" -> "test".
		{"base64 input", "dGVzdA==", "dGVzd\nRlc3\n0ZXN0", plain("Base64")},
	})
}

// TestShowBase64OffsetsHTML covers the full annotated HTML output (Show variable
// chars = true), reproduced verbatim from CyberChef.
func TestShowBase64OffsetsHTML(t *testing.T) {
	abcHTML := "Characters highlighted in <span class='hl5'>green</span> could change if the input is surrounded by more data.\nCharacters highlighted in <span class='hl3'>red</span> are for padding purposes only.\nUnhighlighted characters are <span data-toggle='tooltip' data-placement='top' title='Tooltip on left'>static</span>.\nHover over the static sections to see what they decode to on their own.\n\nOffset 0: <span data-toggle='tooltip' data-placement='top' title='abc'>YWJj</span>\nOffset 1: <span class='hl3'>A</span><span class='hl5'>G</span><span data-toggle='tooltip' data-placement='top' title=''>FiY</span><span class='hl5'>w</span><span class='hl3'>==</span>\nOffset 2: <span class='hl3'>AA</span><span class='hl5'>B</span><span data-toggle='tooltip' data-placement='top' title=''>hYm</span><span class='hl5'>M</span><span class='hl3'>=</span><script type='application/javascript'>$('[data-toggle=\"tooltip\"]').tooltip()</script>"
	runCases(t, []opCase{
		{
			"abc full html", "abc", abcHTML,
			core.Recipe{{Op: "Show Base64 offsets", Args: []any{"A-Za-z0-9+/=", true, "Raw"}}},
		},
	})
}

// TestShowBase64OffsetsEmpty covers the empty-input error.
func TestShowBase64OffsetsEmpty(t *testing.T) {
	if _, err := runOp(t, "Show Base64 offsets", "", "A-Za-z0-9+/=", true, "Raw"); err == nil {
		t.Fatal("expected an error for empty input")
	}
}

// TestSliceUTF16 exercises the JS-slice helper directly, including the
// out-of-range clamps (start past the end, dropEnd past the start).
func TestSliceUTF16(t *testing.T) {
	cases := []struct {
		s              string
		start, dropEnd int
		want           string
	}{
		{"abcd", 0, 0, "abcd"},
		{"abcd", 1, 1, "bc"},
		{"abcd", 0, 4, ""},
		{"abcd", 5, 0, ""}, // start past the end -> clamped
		{"abcd", 2, 5, ""}, // dropEnd past the start -> empty
		{"abcd", 3, 0, "d"},
	}
	for _, c := range cases {
		if got := sliceUTF16(c.s, c.start, c.dropEnd); got != c.want {
			t.Errorf("sliceUTF16(%q, %d, %d) = %q; want %q", c.s, c.start, c.dropEnd, got, c.want)
		}
	}
}
