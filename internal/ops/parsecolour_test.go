package ops

import (
	"strings"
	"testing"
)

// Parse colour code conversions verified against the CyberChef-server oracle.
func TestParseColourCode(t *testing.T) {
	out, err := runOp(t, "Parse colour code", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Hex:  #ff0000",
		"RGB:  rgb(255, 0, 0)",
		"RGBA: rgba(255, 0, 0, 1)",
		"HSL:  hsl(0, 100%, 50%)",
		"CMYK: cmyk(0.00, 1.00, 1.00, 0.00)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestParseColourCodeInputs exercises the rgb/hsl/cmyk input parsers (and the
// hsl<->rgb conversion), verified against the CyberChef-server oracle.
func TestParseColourCodeInputs(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"rgb(217, 237, 247)", []string{"Hex:  #d9edf7", "HSL:  hsl(200, 65%, 91%)", "CMYK: cmyk(0.12, 0.04, 0.00, 0.03)"}},
		{"hsl(200, 65%, 91%)", []string{"Hex:  #d9edf7", "RGB:  rgb(217, 237, 247)"}},
		{"cmyk(0.12, 0.04, 0.00, 0.03)", []string{"Hex:  #daedf7", "RGB:  rgb(218, 237, 247)", "HSL:  hsl(201, 64%, 91%)"}},
		{"rgba(255, 0, 0, 0.5)", []string{"RGBA: rgba(255, 0, 0, 0.5)", "HSLA: hsla(0, 100%, 50%, 0.5)"}},
	}
	for _, c := range cases {
		out, err := runOp(t, "Parse colour code", c.input)
		if err != nil {
			t.Fatalf("%s: %v", c.input, err)
		}
		for _, want := range c.want {
			if !strings.Contains(out, want) {
				t.Errorf("%s: missing %q in:\n%s", c.input, want, out)
			}
		}
	}
}
