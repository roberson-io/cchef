package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Transcribed from CyberChef's tests/operations/tests/Float.mjs.
func TestFloatFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Float: nothing", "", "",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Float", Args: []any{"Big Endian", "Float (4 bytes)", "Space"}},
			},
		},
		{
			"To Float (Big Endian, 4 bytes): 0.5", "3f0000003f000000", "0.5 0.5",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Float", Args: []any{"Big Endian", "Float (4 bytes)", "Space"}},
			},
		},
		{
			"To Float (Little Endian, 4 bytes): 0.5", "0000003f0000003f", "0.5 0.5",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Float", Args: []any{"Little Endian", "Float (4 bytes)", "Space"}},
			},
		},
		{
			"To Float (Big Endian, 8 bytes): 0.5", "3fe00000000000003fe0000000000000", "0.5 0.5",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Float", Args: []any{"Big Endian", "Double (8 bytes)", "Space"}},
			},
		},
		{
			"To Float (Little Endian, 8 bytes): 0.5", "000000000000e03f000000000000e03f", "0.5 0.5",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Float", Args: []any{"Little Endian", "Double (8 bytes)", "Space"}},
			},
		},
		{
			"From Float: nothing", "", "",
			core.Recipe{
				{Op: "From Float", Args: []any{"Big Endian", "Float (4 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"From Float (Big Endian, 4 bytes): 0.5", "0.5 0.5", "3f0000003f000000",
			core.Recipe{
				{Op: "From Float", Args: []any{"Big Endian", "Float (4 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"From Float (Little Endian, 4 bytes): 0.5", "0.5 0.5", "0000003f0000003f",
			core.Recipe{
				{Op: "From Float", Args: []any{"Little Endian", "Float (4 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"From Float (Big Endian, 8 bytes): 0.5", "0.5 0.5", "3fe00000000000003fe0000000000000",
			core.Recipe{
				{Op: "From Float", Args: []any{"Big Endian", "Double (8 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"From Float (Little Endian, 8 bytes): 0.5", "0.5 0.5", "000000000000e03f000000000000e03f",
			core.Recipe{
				{Op: "From Float", Args: []any{"Little Endian", "Double (8 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}

// TestFloatParity covers formatting/encoding behaviour beyond the upstream
// fixtures, verified against the CyberChef-server oracle: JavaScript number
// formatting (fixed, exponential, negatives, NaN/Infinity) for To Float and
// lenient parseFloat plus the ieee754 NaN/Infinity byte encodings for From
// Float.
func TestFloatParity(t *testing.T) {
	toFloat := func(size string) core.Recipe {
		return core.Recipe{
			{Op: "From Hex", Args: []any{"Auto"}},
			{Op: "To Float", Args: []any{"Big Endian", size, "Space"}},
		}
	}
	fromFloat := func() core.Recipe {
		return core.Recipe{
			{Op: "From Float", Args: []any{"Big Endian", "Float (4 bytes)", "Space"}},
			{Op: "To Hex", Args: []any{"None"}},
		}
	}
	f4, d8 := "Float (4 bytes)", "Double (8 bytes)"
	runCases(t, []opCase{
		// To Float — JavaScript number formatting.
		{"To Float: negative", "c0200000", "-2.5", toFloat(f4)},
		{"To Float: widened float32 precision", "3dcccccd", "0.10000000149011612", toFloat(f4)},
		{"To Float: exponential large", "444b1ae4d6e2ef50", "1e+21", toFloat(d8)},
		{"To Float: exponential small", "3e7ad7f29abcaf48", "1e-7", toFloat(d8)},
		{"To Float: exponential with mantissa", "44dfe154f457ea13", "6.022e+23", toFloat(d8)},
		{"To Float: zero", "00000000", "0", toFloat(f4)},
		{"To Float: integer with trailing zeros", "42c80000", "100", toFloat(f4)},
		{"To Float: NaN", "7f800001", "NaN", toFloat(f4)},
		{"To Float: Infinity", "7f800000", "Infinity", toFloat(f4)},
		{"To Float: -Infinity", "ff800000", "-Infinity", toFloat(f4)},
		// From Float — lenient parseFloat and ieee754 special encodings.
		{"From Float: lenient prefix parse", "1.5abc", "3fc00000", fromFloat()},
		{"From Float: NaN token", "abc", "7f800001", fromFloat()},
		{"From Float: Infinity token", "Infinity", "7f800000", fromFloat()},
		{
			"From Float: double NaN token", "abc", "7ff0000000000001",
			core.Recipe{
				{Op: "From Float", Args: []any{"Big Endian", "Double (8 bytes)", "Space"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}

// TestFloatErrors covers To Float's length-validation error path.
func TestFloatErrors(t *testing.T) {
	if _, err := runOp(t, "To Float", "abc", "Big Endian", "Float (4 bytes)", "Space"); err == nil {
		t.Fatal("expected an error for input that is not a multiple of the byte size")
	}
}
