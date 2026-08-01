package ops

import "testing"

// TestXKCDRandomNumber covers the whole of what the operation does. The joke is
// that a number chosen once by a fair dice roll is thereafter returned every
// time, so the test that it does not vary is the point rather than an oversight.
func TestXKCDRandomNumber(t *testing.T) {
	for _, input := range []string{"", "anything", "1234", "\x00\xff"} {
		out, err := runOp(t, "XKCD Random Number", input)
		if err != nil {
			t.Fatalf("Run(%q): %v", input, err)
		}
		if out != "4" {
			t.Errorf("for %q got %q, want %q", input, out, "4")
		}
	}
}
