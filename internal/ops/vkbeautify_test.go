package ops

import "testing"

// createShiftArr backs XML/CSS Beautify indentation. It builds ["\n", "\n"+unit,
// "\n"+unit+unit, ...]. The indent unit is the given string, EXCEPT when the
// string parses as a leading integer (JS isNaN(parseInt(step)) is false): the
// original library then falls through a numeric switch that never matches a
// string, leaving the 4-space default. An empty unit is handled by the callers.
func TestCreateShiftArr(t *testing.T) {
	cases := []struct {
		step string
		want string // expected shift[2]
	}{
		{"  ", "\n    "},       // two spaces
		{"\t", "\n\t\t"},       // tab
		{"....", "\n........"}, // non-numeric string used verbatim
		{"2", "\n        "},    // digit-leading -> 4-space default (quirk)
		{"    ", "\n        "}, // four spaces
	}
	for _, c := range cases {
		got := createShiftArr(c.step)
		if len(got) != 101 {
			t.Errorf("createShiftArr(%q): len %d, want 101", c.step, len(got))
		}
		if got[0] != "\n" {
			t.Errorf("createShiftArr(%q)[0] = %q, want %q", c.step, got[0], "\n")
		}
		if got[2] != c.want {
			t.Errorf("createShiftArr(%q)[2] = %q, want %q", c.step, got[2], c.want)
		}
	}
}
