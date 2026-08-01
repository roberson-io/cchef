package ops

import (
	"math"

	"github.com/roberson-io/cchef/internal/jsnum"
)

// jsNum writes a float the way JavaScript's Number#toString does.
func jsNum(f float64) string { return jsnum.Format(f) }

// jsRound rounds half towards positive infinity, as JavaScript's Math.round
// does. Go's math.Round rounds halves away from zero, which differs for
// negatives such as -0.5.
func jsRound(f float64) int {
	return int(jsRoundFloat(f))
}

// jsRoundFloat is jsRound without the narrowing to int.
func jsRoundFloat(f float64) float64 {
	return math.Floor(f + 0.5)
}
