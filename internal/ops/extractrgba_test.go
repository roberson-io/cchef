package ops

import (
	"strings"
	"testing"
)

// Expected outputs recorded from CyberChef's Node API over the same pixels.

// TestExtractRGBA covers alpha on and off, and the delimiter used verbatim.
func TestExtractRGBA(t *testing.T) {
	one := pngBytes(t, lsbOnePixel())
	solid := pngBytes(t, lsbSolid3x3())
	cases := []struct {
		name  string
		input string
		args  []any
		want  string
	}{
		{"comma with alpha", one, []any{",", true}, "1,2,3,4"},
		{"comma without alpha", one, []any{",", false}, "1,2,3"},
		{"space", one, []any{" ", true}, "1 2 3 4"},
		{"any string as delimiter", one, []any{"::", true}, "1::2::3::4"},
		{"empty delimiter", one, []any{"", true}, "1234"},
		{
			"several pixels without alpha", solid,
			[]any{",", false},
			strings.TrimSuffix(strings.Repeat("200,100,50,", 9), ","),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "Extract RGBA", c.input, c.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestExtractRGBAInvalid covers the refusal of input that is not an image.
func TestExtractRGBAInvalid(t *testing.T) {
	if _, err := runOp(t, "Extract RGBA", "not an image", ",", true); err == nil ||
		!strings.Contains(err.Error(), "Please enter a valid image file.") {
		t.Errorf("got %v", err)
	}
}
