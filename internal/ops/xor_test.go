package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// XOR has no upstream fixture file; these cases use hand-verified values,
// wrapping the byteArray output in To Hex (None) for readable comparison.
func TestXOROp(t *testing.T) {
	hexKey := func(k string) core.ToggleString { return core.ToggleString{Value: k, Option: "Hex"} }
	utf8Key := func(k string) core.ToggleString { return core.ToggleString{Value: k, Option: "UTF8"} }

	runCases(t, []opCase{
		// "Hello" ^ 0x42 (repeating single-byte key).
		{"XOR single-byte hex key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("42"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			}},
		// "ABCD" ^ 0x0102 (repeating two-byte key).
		{"XOR multi-byte hex key", "ABCD", "40404246",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("0102"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			}},
		// "Hello" ^ "K" (0x4b) with a UTF8 key.
		{"XOR utf8 key", "Hello", "032e272724",
			core.Recipe{
				{Op: "XOR", Args: []any{utf8Key("K"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			}},
		// XOR is symmetric: applying the same key twice restores the input.
		{"XOR round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("3f"), "Standard", false}},
				{Op: "XOR", Args: []any{hexKey("3f"), "Standard", false}},
			}},
	})
}
