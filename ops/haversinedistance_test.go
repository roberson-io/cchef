package ops

import (
	"math"
	"strconv"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestHaversineFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/HaversineDistance.mjs).
func TestHaversineFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Haversine distance",
			"51.487263,-0.124323, 38.9517,-77.1467",
			"5902542.836307819",
			core.Recipe{{Op: "Haversine distance"}},
		},
		{
			"Haversine distance, zero distance",
			"51.487263,-0.124323, 51.487263,-0.124323",
			"0",
			core.Recipe{{Op: "Haversine distance"}},
		},
	})
}

// haversineTolerance is how far the answer may sit from CyberChef's.
//
// Go's math.Cos and the one V8 runs disagree by a unit or two in the last place
// for some arguments — both are correctly rounded to within an accepted
// tolerance, but they are not the same implementation, so the last digit or two
// of a seventeen-digit answer can differ. Over two hundred positions taken at
// random, 136 agreed to the digit and the rest differed by no more than 6.3e-16
// relative, which on a distance right across the globe is about ten nanometres.
//
// Anything looser than this would let a real mistake through: the next thing
// that could go wrong, a radius or a conversion out by a place, is many orders
// larger.
const haversineTolerance = 1e-15

// closeEnough reports whether two written distances agree to within that.
func closeEnough(t *testing.T, got, want string) bool {
	t.Helper()
	if got == want {
		return true
	}
	a, err := strconv.ParseFloat(got, 64)
	if err != nil {
		t.Fatalf("%q is not a number: %v", got, err)
	}
	b, err := strconv.ParseFloat(want, 64)
	if err != nil {
		t.Fatalf("%q is not a number: %v", want, err)
	}
	if b == 0 {
		return a == 0
	}
	return math.Abs(a-b)/math.Abs(b) <= haversineTolerance
}

// TestHaversineValues covers the formula across the globe, against the oracle.
// The two antipodal cases do not come out at exactly half the circumference:
// the arithmetic loses the last few places, and the answer is reported as it
// falls out rather than being tidied up.
func TestHaversineValues(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"a point against itself", "0,0,0,0", "0"},
		{"a quarter of the way round the equator", "0,0, 0, 1", "111194.92664455874"},
		{"halfway round the equator", "0,0,0,180", "20015086.79602057"},
		{"pole to equator", "0,0,90,0", "10007543.398010286"},
		{"pole to pole", "-90,0,90,0", "20015086.79602057"},
		{"one degree of each", "1,1,2,2", "157225.4320380729"},
		{"Sydney to New York", "-33.8688,151.2093,40.7128,-74.0060", "15988755.50703963"},
		{"a very short hop", "51.5,0,51.5,0.0001", "6.922046935607885"},
		{"written with no spaces", "51.487263,-0.124323,38.9517,-77.1467", "5902542.836307819"},
		{"written with a space after each comma", "51.487263, -0.124323, 38.9517, -77.1467", "5902542.836307819"},
		{"written with trailing zeroes", "0.0,0.0,0.0,0.0", "0"},
		{"a negative zero", "-0,0,0,0", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Haversine distance", tc.input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if !closeEnough(t, out, tc.want) {
				t.Errorf("got %s, want %s", out, tc.want)
			}
		})
	}
}

// TestHaversineNearlyTouching covers two ways of naming the same place. The
// formula is at its weakest here: the two points are half a turn apart in
// longitude, which drives the intermediate value to one, and taking it from one
// again leaves almost no significant figures behind. Both CyberChef and cchef
// report a few nanometres of arithmetic noise rather than nothing at all, and
// which nanometres they report is not something either can be held to. What can
// be held to is that the answer rounds to nothing on any scale a distance is
// measured at.
func TestHaversineNearlyTouching(t *testing.T) {
	// A micrometre, far below what the formula could resolve even in principle
	// and far above the noise either implementation produces.
	const negligible = 1e-6

	for _, tc := range []struct{ name, input string }{
		{"the two ways of writing the date line", "0,-180,0,180"},
		{"the north pole reached the long way round", "90,0,90,180"},
		{"the south pole reached the long way round", "-90,0,-90,180"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Haversine distance", tc.input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			metres, err := strconv.ParseFloat(out, 64)
			if err != nil {
				t.Fatalf("%q is not a number: %v", out, err)
			}
			if math.Abs(metres) > negligible {
				t.Errorf("got %s metres between two names for one place", out)
			}
		})
	}
}

// TestHaversineRejects covers input it cannot read a pair of positions from.
// Only a space after a comma is allowed, so anything else in the spacing is
// turned away.
func TestHaversineRejects(t *testing.T) {
	const want = "Input must in the format lat1, lng1, lat2, lng2"

	for _, tc := range []struct{ name, input string }{
		{"nothing at all", ""},
		{"three numbers", "0,0,0"},
		{"five numbers", "0,0,0,0,0"},
		{"letters", "a,b,c,d"},
		{"a space before a comma", "51.487263 , -0.124323, 38.9517, -77.1467"},
		{"space around the whole thing", "  0,0,0,0  "},
		{"a number written with an exponent", "1e2,0,0,0"},
		{"a number written with a plus", "+1,0,0,0"},
		{"two positions on separate lines", "0,0\n0,0"},
		{"semicolons instead of commas", "0;0;0;0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Haversine distance", tc.input)
			if err == nil {
				t.Fatalf("read %q as a pair of positions, giving %s", tc.input, out)
			}
			if err.Error() != want {
				t.Errorf("got %q, want %q", err.Error(), want)
			}
		})
	}
}
