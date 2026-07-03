package ops

import (
	"strconv"
	"testing"
	"time"

	"github.com/roberson-io/cchef/internal/core"
)

// TestGetTime checks Get Time returns the current time in the chosen unit. It is
// inherently non-deterministic (like Sleep), so the result is bounded against
// time.Now rather than compared to a fixture.
func TestGetTime(t *testing.T) {
	cases := []struct {
		unit    string
		perSec  int64
		tolTens int64 // allowed slack, in the op's unit, for clock drift during the test
	}{
		{"Seconds (s)", 1, 5},
		{"Milliseconds (ms)", 1000, 5000},
		{"Microseconds (μs)", 1000000, 5000000},
		{"Nanoseconds (ns)", 1000000000, 5000000000},
	}
	for _, c := range cases {
		t.Run(c.unit, func(t *testing.T) {
			before := time.Now().UnixNano()
			out, err := core.Recipe{{Op: "Get Time", Args: []any{c.unit}}}.
				Execute(core.NewDish(nil, core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			got, err := strconv.ParseInt(out.String(), 10, 64)
			if err != nil {
				t.Fatalf("result %q not an integer: %v", out.String(), err)
			}
			lo := before/1000000000*c.perSec - c.tolTens
			hi := time.Now().UnixNano()/1000000000*c.perSec + c.perSec + c.tolTens
			if got < lo || got > hi {
				t.Fatalf("%s: got %d, want within [%d, %d]", c.unit, got, lo, hi)
			}
		})
	}
}
