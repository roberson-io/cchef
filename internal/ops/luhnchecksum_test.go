package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// luhnTuple is a fixture from ../CyberChef/tests/operations/tests/LuhnChecksum.mjs:
// radix, input, and the expected checksum / check digit (rendered in that radix).
type luhnTuple struct {
	radix         int
	input, cs, cd string
}

// All 72 generated Luhn fixtures (one set per even radix 2..36).
var luhnTuples = []luhnTuple{
	{2, "01", "1", "1"},
	{2, "001111", "0", "0"},
	{2, "00011101", "0", "0"},
	{2, "0100101101", "1", "1"},
	{4, "0123", "1", "1"},
	{4, "130100", "2", "2"},
	{4, "32020313", "3", "0"},
	{4, "302233210112", "3", "0"},
	{6, "012345", "4", "4"},
	{6, "134255", "2", "4"},
	{6, "15021453", "5", "4"},
	{6, "211450230513", "3", "1"},
	{8, "01234567", "2", "2"},
	{8, "340624", "0", "4"},
	{8, "07260247", "3", "3"},
	{8, "026742114675", "7", "1"},
	{10, "0123456789", "7", "7"},
	{10, "468543", "7", "4"},
	{10, "59377601", "5", "6"},
	{10, "013909981254", "1", "3"},
	{12, "0123456789ab", "3", "3"},
	{12, "284685", "0", "6"},
	{12, "951a2661", "0", "8"},
	{12, "898202676387", "b", "9"},
	{14, "0123456789abcd", "a", "a"},
	{14, "33db25", "0", "d"},
	{14, "0b4ac128", "b", "3"},
	{14, "3d1c6d16160d", "3", "c"},
	{16, "0123456789abcdef", "4", "4"},
	{16, "e1fe64", "b", "6"},
	{16, "241a5dcd", "1", "9"},
	{16, "1fea740e0e1f", "7", "4"},
	{18, "0123456789abcdefgh", "d", "d"},
	{18, "995dgf", "9", "1"},
	{18, "9f80h32h", "1", "0"},
	{18, "5f9428e493g4", "8", "c"},
	{20, "0123456789abcdefghij", "5", "5"},
	{20, "918jci", "h", "d"},
	{20, "jab7j50d", "g", "j"},
	{20, "c56fe85eb6gg", "g", "5"},
	{22, "0123456789abcdefghijkl", "g", "g"},
	{22, "de57le", "5", "l"},
	{22, "e3fg6dfc", "f", "d"},
	{22, "1f8l80ai4kbg", "l", "f"},
	{24, "0123456789abcdefghijklmn", "6", "6"},
	{24, "agne7d", "4", "f"},
	{24, "1l4d9cf4", "d", "c"},
	{24, "blc1j09i3296", "8", "7"},
	{26, "0123456789abcdefghijklmnop", "j", "j"},
	{26, "82n9op", "i", "2"},
	{26, "e9cddn70", "9", "i"},
	{26, "ck0ep419knom", "p", "g"},
	{28, "0123456789abcdefghijklmnopqr", "7", "7"},
	{28, "a6hnoo", "h", "9"},
	{28, "lblc7kh0", "a", "f"},
	{28, "64k5piod3lmf", "0", "p"},
	{30, "0123456789abcdefghijklmnopqrst", "m", "m"},
	{30, "t69j7d", "9", "s"},
	{30, "p54o9ig3", "a", "o"},
	{30, "gc1njrt55030", "6", "1"},
	{32, "0123456789abcdefghijklmnopqrstuv", "8", "8"},
	{32, "rdou19", "u", "3"},
	{32, "ighj0pc7", "3", "8"},
	{32, "op4nn5fvjsrs", "g", "j"},
	{34, "0123456789abcdefghijklmnopqrstuvwx", "p", "p"},
	{34, "nvftj5", "b", "f"},
	{34, "u9v9g162", "j", "b"},
	{34, "o5gqg5d7gjh9", "5", "q"},
	{36, "0123456789abcdefghijklmnopqrstuvwxyz", "9", "9"},
	{36, "29zehu", "i", "j"},
	{36, "1snmikbu", "s", "v"},
	{36, "jpkar545q7gb", "3", "d"},
}

func TestLuhnChecksumFixtures(t *testing.T) {
	var cases []opCase
	for _, tc := range luhnTuples {
		want := "Checksum: " + tc.cs + "\nCheckdigit: " + tc.cd +
			"\nLuhn Validated String: " + tc.input + tc.cd
		cases = append(cases, opCase{
			"Luhn Mod " + tc.input, tc.input, want,
			core.Recipe{{Op: "Luhn Checksum", Args: []any{tc.radix}}},
		})
	}
	// The four hand-written fixtures (standard mod-10 data and empty input).
	cases = append(
		cases,
		opCase{"Luhn standard", "35641709012469", "Checksum: 7\nCheckdigit: 0\nLuhn Validated String: 356417090124690", core.Recipe{{Op: "Luhn Checksum", Args: []any{10}}}},
		opCase{"Luhn standard 2", "896101950123440000", "Checksum: 5\nCheckdigit: 1\nLuhn Validated String: 8961019501234400001", core.Recipe{{Op: "Luhn Checksum", Args: []any{10}}}},
		opCase{"Luhn standard 3", "35726908971331", "Checksum: 6\nCheckdigit: 7\nLuhn Validated String: 357269089713317", core.Recipe{{Op: "Luhn Checksum", Args: []any{10}}}},
		opCase{"Luhn empty", "", "", core.Recipe{{Op: "Luhn Checksum", Args: []any{10}}}},
	)
	runCases(t, cases)
}

func TestLuhnChecksumErrors(t *testing.T) {
	cases := []struct {
		name, in, wantErr string
		radix             int
	}{
		{"radix too small", "123", "Error: Radix argument must be between 2 and 36", 1},
		{"radix too big", "123", "Error: Radix argument must be between 2 and 36", 37},
		{"radix odd", "123", "Error: Radix argument must be divisible by 2", 11},
		{"invalid character", "12x", "Character: x is not valid in radix 10.", 10},
		{"uppercase digit out of range", "1G", "Character: G is not valid in radix 16.", 16},
		{"non-alphanumeric character", "1!", "Character: ! is not valid in radix 10.", 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Luhn Checksum", c.in, c.radix)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}
