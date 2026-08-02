package ops

import (
	"math"
	"testing"

	"github.com/roberson-io/cchef/internal/jsnum"
)

// Non-finite values render with JavaScript's names.
func TestJSNumNonFinite(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	} {
		if got := jsnum.Format(tc.in); got != tc.want {
			t.Errorf("jsnum.Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
