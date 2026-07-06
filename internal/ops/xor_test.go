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
		{
			"XOR single-byte hex key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("42"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// "ABCD" ^ 0x0102 (repeating two-byte key).
		{
			"XOR multi-byte hex key", "ABCD", "40404246",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("0102"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// "Hello" ^ "K" (0x4b) with a UTF8 key.
		{
			"XOR utf8 key", "Hello", "032e272724",
			core.Recipe{
				{Op: "XOR", Args: []any{utf8Key("K"), "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// XOR is symmetric: applying the same key twice restores the input.
		{
			"XOR round trip", "Hello, World!", "Hello, World!",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("3f"), "Standard", false}},
				{Op: "XOR", Args: []any{hexKey("3f"), "Standard", false}},
			},
		},

		// The same 0x42 key expressed in every supported encoding yields the
		// same result, exercising convertToByteArray's decode paths.
		{
			"XOR decimal key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{core.ToggleString{Value: "66", Option: "Decimal"}, "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"XOR binary key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{core.ToggleString{Value: "01000010", Option: "Binary"}, "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"XOR base64 key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{core.ToggleString{Value: "Qg==", Option: "Base64"}, "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"XOR latin1 key", "Hello", "0a272e2e2d",
			core.Recipe{
				{Op: "XOR", Args: []any{core.ToggleString{Value: "B", Option: "Latin1"}, "Standard", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},

		// Schemes and null-preserving option.
		{
			"XOR cascade scheme", "Hello", "2d0900036f",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("00"), "Cascade", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// Empty key defaults to 0x00, so null-preserving leaves input unchanged.
		{
			"XOR null preserving", "Hello", "Hello",
			core.Recipe{{Op: "XOR", Args: []any{hexKey(""), "Standard", true}}},
		},
		// Null preserving with a key that matches the first byte ('H' == 0x48):
		// that byte is left untouched (skip), the rest XOR normally.
		{
			"XOR null preserving skips matching byte", "Hello", "482d242427",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("48"), "Standard", true}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// Input differential rewrites each key byte to the plaintext byte just read,
		// so the key ratchets down the input.
		{
			"XOR input differential", "Hello", "0a2d090003",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("42"), "Input differential", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// Output differential rewrites each key byte to the ciphertext byte emitted.
		{
			"XOR output differential", "Hello", "0a6f036f00",
			core.Recipe{
				{Op: "XOR", Args: []any{hexKey("42"), "Output differential", false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}
