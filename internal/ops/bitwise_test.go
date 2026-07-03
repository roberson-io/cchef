package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// hexKey builds a Hex-encoded toggleString key argument.
func hexKey(s string) core.ToggleString { return core.ToggleString{Value: s, Option: "Hex"} }

// TestBitwiseFixtures covers the byte-array bitwise/logic operations. The Bit
// shift cases are transcribed from CyberChef's BitwiseOp.mjs; the ADD/SUB/AND/OR/
// NOT cases use expected outputs from the CyberChef-server oracle, wrapped in
// To Hex for a readable comparison (mirroring CyberChef's own test style).
func TestBitwiseFixtures(t *testing.T) {
	toHex := core.RecipeOp{Op: "To Hex", Args: []any{"None"}}
	runCases(t, []opCase{
		// ADD (mod 256)
		{"ADD: hex key ff", "hello", "67646b6b6e",
			core.Recipe{{Op: "ADD", Args: []any{hexKey("ff")}}, toHex}},
		{"ADD: repeating hex key", "hello", "69676d6e70",
			core.Recipe{{Op: "ADD", Args: []any{hexKey("01 02")}}, toHex}},
		{"ADD: empty key is identity", "hello", "68656c6c6f",
			core.Recipe{{Op: "ADD", Args: []any{hexKey("")}}, toHex}},
		{"ADD: UTF8 key", "hello", "d3d0d7d7da",
			core.Recipe{{Op: "ADD", Args: []any{core.ToggleString{Value: "k", Option: "UTF8"}}}, toHex}},

		// SUB (mod 256; -1 == +255)
		{"SUB: hex key 01", "hello", "67646b6b6e",
			core.Recipe{{Op: "SUB", Args: []any{hexKey("01")}}, toHex}},

		// AND
		{"AND: hex key 0f", "hello", "08050c0c0f",
			core.Recipe{{Op: "AND", Args: []any{hexKey("0f")}}, toHex}},
		{"AND: decimal key 15", "hello", "08050c0c0f",
			core.Recipe{{Op: "AND", Args: []any{core.ToggleString{Value: "15", Option: "Decimal"}}}, toHex}},

		// OR
		{"OR: hex key 80", "hello", "e8e5ececef",
			core.Recipe{{Op: "OR", Args: []any{hexKey("80")}}, toHex}},

		// NOT
		{"NOT: inverse of each byte", "hello", "979a939390",
			core.Recipe{{Op: "NOT", Args: []any{}}, toHex}},

		// Bit shift left (BitwiseOp.mjs)
		{"Bit shift left", "01010101 10101010 11111111 00000000 11110000 00001111 00110011 11001100",
			"10101010 01010100 11111110 00000000 11100000 00011110 01100110 10011000",
			core.Recipe{
				{Op: "From Binary", Args: []any{"Space"}},
				{Op: "Bit shift left", Args: []any{float64(1)}},
				{Op: "To Binary", Args: []any{"Space"}},
			}},

		// Bit shift right: Logical (BitwiseOp.mjs)
		{"Bit shift right: Logical shift", "01010101 10101010 11111111 00000000 11110000 00001111 00110011 11001100",
			"00101010 01010101 01111111 00000000 01111000 00000111 00011001 01100110",
			core.Recipe{
				{Op: "From Binary", Args: []any{"Space"}},
				{Op: "Bit shift right", Args: []any{float64(1), "Logical shift"}},
				{Op: "To Binary", Args: []any{"Space"}},
			}},

		// Bit shift right: Arithmetic (BitwiseOp.mjs)
		{"Bit shift right: Arithmetic shift", "01010101 10101010 11111111 00000000 11110000 00001111 00110011 11001100",
			"00101010 11010101 11111111 00000000 11111000 00000111 00011001 11100110",
			core.Recipe{
				{Op: "From Binary", Args: []any{"Space"}},
				{Op: "Bit shift right", Args: []any{float64(1), "Arithmetic shift"}},
				{Op: "To Binary", Args: []any{"Space"}},
			}},
	})
}
