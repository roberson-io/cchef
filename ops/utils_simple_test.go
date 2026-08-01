package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Hand-verified cases for the simple Utils operations (no upstream fixtures).
func TestSwapCase(t *testing.T) {
	runCases(t, []opCase{
		{
			"Swap case", "Hello, World! 123", "hELLO, wORLD! 123",
			core.Recipe{{Op: "Swap case"}},
		},
	})
}

func TestRemoveWhitespace(t *testing.T) {
	// All flags on except full stops (defaults).
	allButStops := []any{true, true, true, true, true, false}
	allOn := []any{true, true, true, true, true, true}
	runCases(t, []opCase{
		{
			"Remove whitespace (default)", "a b\tc\r\nd.e", "abcd.e",
			core.Recipe{{Op: "Remove whitespace", Args: allButStops}},
		},
		{
			"Remove whitespace + full stops", "a b.c", "abc",
			core.Recipe{{Op: "Remove whitespace", Args: allOn}},
		},
	})
}

func TestRemoveNullBytes(t *testing.T) {
	runCases(t, []opCase{
		{
			"Remove null bytes", "a\x00b\x00\x00c", "abc",
			core.Recipe{{Op: "Remove null bytes"}},
		},
	})
}

func TestPadLines(t *testing.T) {
	runCases(t, []opCase{
		{
			"Pad lines start", "ab\ncd", "***ab\n***cd",
			core.Recipe{{Op: "Pad lines", Args: []any{"Start", 3, "*"}}},
		},
		{
			"Pad lines end", "ab\ncd", "ab***\ncd***",
			core.Recipe{{Op: "Pad lines", Args: []any{"End", 3, "*"}}},
		},
		// Multi-character pad pattern repeats then truncates to the length.
		{
			"Pad lines multi-char", "x", "ababax",
			core.Recipe{{Op: "Pad lines", Args: []any{"Start", 5, "ab"}}},
		},
		{
			// A non-positive length produces an empty pad (input unchanged).
			"Pad lines zero length", "ab\ncd", "ab\ncd",
			core.Recipe{{Op: "Pad lines", Args: []any{"Start", 0, "*"}}},
		},
	})
}
