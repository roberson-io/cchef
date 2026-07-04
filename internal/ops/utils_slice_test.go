package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestHeadTail(t *testing.T) {
	const in = "a\nb\nc\nd\ne"
	runCases(t, []opCase{
		{
			"Head 2", in, "a\nb",
			core.Recipe{{Op: "Head", Args: []any{"Line feed", 2}}},
		},
		{
			"Head -1 (all but last)", in, "a\nb\nc\nd",
			core.Recipe{{Op: "Head", Args: []any{"Line feed", -1}}},
		},
		{
			"Tail 2", in, "d\ne",
			core.Recipe{{Op: "Tail", Args: []any{"Line feed", 2}}},
		},
		{
			"Tail -1 (all but first)", in, "b\nc\nd\ne",
			core.Recipe{{Op: "Tail", Args: []any{"Line feed", -1}}},
		},
	})
}

func TestDropTakeBytes(t *testing.T) {
	runCases(t, []opCase{
		{
			"Take bytes", "0123456789", "234",
			core.Recipe{{Op: "Take bytes", Args: []any{2, 3, false}}},
		},
		{
			"Take bytes negative start", "0123456789", "89",
			core.Recipe{{Op: "Take bytes", Args: []any{-2, 2, false}}},
		},
		{
			"Drop bytes", "0123456789", "0156789",
			core.Recipe{{Op: "Drop bytes", Args: []any{2, 3, false}}},
		},
		{
			"Take bytes each line", "abcdef\nghijkl", "ab\ngh",
			core.Recipe{{Op: "Take bytes", Args: []any{0, 2, true}}},
		},
		// Negative length and out-of-range clamping (adjustRange/clampInt
		// branches), verified against the CyberChef-server oracle.
		{
			"Take bytes negative length", "0123456789", "34",
			core.Recipe{{Op: "Take bytes", Args: []any{5, -2, false}}},
		},
		{
			"Take bytes negative length flips past start", "0123456789", "89",
			core.Recipe{{Op: "Take bytes", Args: []any{1, -3, false}}},
		},
		{
			"Take bytes start past end", "0123456789", "",
			core.Recipe{{Op: "Take bytes", Args: []any{100, 5, false}}},
		},
		{
			// Start more negative than the buffer clamps up to 0 (clampInt v<lo).
			"Take bytes start before start", "0123456789", "",
			core.Recipe{{Op: "Take bytes", Args: []any{-100, 3, false}}},
		},
		{
			"Take bytes length past end", "0123456789", "23456789",
			core.Recipe{{Op: "Take bytes", Args: []any{2, 100, false}}},
		},
		{
			"Drop bytes negative length", "0123456789", "01256789",
			core.Recipe{{Op: "Drop bytes", Args: []any{5, -2, false}}},
		},
		{
			"Drop bytes negative length flips past start", "0123456789", "01234567",
			core.Recipe{{Op: "Drop bytes", Args: []any{1, -3, false}}},
		},
		{
			"Drop bytes start past end (unchanged)", "0123456789", "0123456789",
			core.Recipe{{Op: "Drop bytes", Args: []any{100, 5, false}}},
		},
		{
			"Drop bytes length past end", "0123456789", "01",
			core.Recipe{{Op: "Drop bytes", Args: []any{2, 100, false}}},
		},
	})
}

// Cases transcribed from CyberChef DropNthBytes.mjs / TakeNthBytes.mjs.
func TestDropTakeNthBytes(t *testing.T) {
	runCases(t, []opCase{
		{
			"Drop nth basic", "0123456789", "1235679",
			core.Recipe{{Op: "Drop nth bytes", Args: []any{4, 0, false}}},
		},
		{
			"Drop nth complex", "0123456789", "01234678",
			core.Recipe{{Op: "Drop nth bytes", Args: []any{4, 5, false}}},
		},
		{
			"Take nth basic", "0123456789", "048",
			core.Recipe{{Op: "Take nth bytes", Args: []any{4, 0, false}}},
		},
		{
			"Take nth complex", "0123456789", "59",
			core.Recipe{{Op: "Take nth bytes", Args: []any{4, 5, false}}},
		},
		// Apply-to-each-line resets the offset at every newline.
		{
			"Drop nth each line", "abcdef\nghijkl", "bdf\nhjl",
			core.Recipe{{Op: "Drop nth bytes", Args: []any{2, 0, true}}},
		},
		{
			"Take nth each line", "abcdef\nghijkl", "ace\ngik",
			core.Recipe{{Op: "Take nth bytes", Args: []any{2, 0, true}}},
		},
	})
}
