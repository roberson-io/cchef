package ops

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// makeIntegers draws integers and hands back what the operation wrote.
func makeIntegers(t *testing.T, count, low, high int64, delimiter, format string) string {
	t.Helper()
	out, err := runOp(t, "Pseudo-Random Integer Generator", "",
		float64(count), float64(low), float64(high), delimiter, format)
	if err != nil {
		t.Fatalf("draw %d in %d..%d: %v", count, low, high, err)
	}
	return out
}

// drawnIntegers draws integers and reads them back as numbers.
func drawnIntegers(t *testing.T, count, low, high int64) []int64 {
	t.Helper()
	out := makeIntegers(t, count, low, high, "Space", "Decimal")
	if out == "" {
		return nil
	}
	fields := strings.Split(out, " ")
	values := make([]int64, len(fields))
	for i, f := range fields {
		v, err := strconv.ParseInt(f, 10, 64)
		if err != nil {
			t.Fatalf("%q is not a number: %v", f, err)
		}
		values[i] = v
	}
	return values
}

// TestPRIGCount covers how many integers come back.
func TestPRIGCount(t *testing.T) {
	for _, count := range []int64{1, 2, 5, 100, 2000} {
		if got := int64(len(drawnIntegers(t, count, 0, 99))); got != count {
			t.Errorf("asked for %d integers and got %d", count, got)
		}
	}
}

// TestPRIGStaysInRange covers the bounds: every integer drawn falls between the
// two given, and both ends are reachable.
func TestPRIGStaysInRange(t *testing.T) {
	for _, r := range []struct{ low, high int64 }{
		{0, 9}, {1, 6}, {-5, -1}, {-3, 3}, {100, 110}, {0, 1},
	} {
		t.Run(strconv.FormatInt(r.low, 10)+".."+strconv.FormatInt(r.high, 10), func(t *testing.T) {
			seen := map[int64]bool{}
			for _, v := range drawnIntegers(t, 4000, r.low, r.high) {
				if v < r.low || v > r.high {
					t.Fatalf("drew %d, which is outside %d..%d", v, r.low, r.high)
				}
				seen[v] = true
			}
			if want := int(r.high-r.low) + 1; len(seen) != want {
				t.Errorf("saw %d of the %d values in range", len(seen), want)
			}
		})
	}
}

// TestPRIGIsEven covers the spread: over enough draws no value in a small range
// is favoured over another. The bound allows for ordinary variation while
// catching a range that is skewed, which is what a rejection threshold worked
// out wrongly would give.
func TestPRIGIsEven(t *testing.T) {
	const draws = 60000
	const values = 10

	counts := map[int64]int{}
	for _, v := range drawnIntegers(t, draws, 0, values-1) {
		counts[v]++
	}

	expected := float64(draws) / values
	// Five standard deviations of a binomial with this many draws.
	tolerance := 5 * math.Sqrt(expected*(1-1.0/values))
	for v := range int64(values) {
		if math.Abs(float64(counts[v])-expected) > tolerance {
			t.Errorf("%d came up %d times, want about %.0f", v, counts[v], expected)
		}
	}
}

// TestPRIGSingleValue covers a range holding one value, which leaves nothing to
// choose.
func TestPRIGSingleValue(t *testing.T) {
	for _, v := range drawnIntegers(t, 50, 7, 7) {
		if v != 7 {
			t.Fatalf("drew %d from a range of one", v)
		}
	}
}

// TestPRIGWholeNumbers covers a bound that is not a whole number: the low bound
// rounds up and the high bound rounds down, so both stay inside what was asked
// for.
func TestPRIGWholeNumbers(t *testing.T) {
	out, err := runOp(t, "Pseudo-Random Integer Generator", "",
		200.0, 0.5, 3.5, "Space", "Decimal")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	seen := map[string]bool{}
	for f := range strings.SplitSeq(out, " ") {
		seen[f] = true
	}
	for _, want := range []string{"1", "2", "3"} {
		if !seen[want] {
			t.Errorf("%s never came up", want)
		}
	}
	for _, unwanted := range []string{"0", "4"} {
		if seen[unwanted] {
			t.Errorf("%s came up, which is outside 0.5..3.5", unwanted)
		}
	}
}

// TestPRIGWidestRange covers the largest span the operation will draw from.
func TestPRIGWidestRange(t *testing.T) {
	const widest = int64(1)<<53 - 2 // the high bound giving a span of 2^53 - 1

	values := drawnIntegers(t, 20, 0, widest)
	if len(values) != 20 {
		t.Fatalf("got %d integers, want 20", len(values))
	}
	for _, v := range values {
		if v < 0 || v > widest {
			t.Errorf("drew %d, which is outside 0..%d", v, widest)
		}
	}
}

// TestPRIGDelimiters covers what goes between the integers.
func TestPRIGDelimiters(t *testing.T) {
	for delimiter, between := range map[string]string{
		"Space": " ", "Comma": ",", "Semi-colon": ";",
		"Colon": ":", "Line feed": "\n", "CRLF": "\r\n",
	} {
		t.Run(delimiter, func(t *testing.T) {
			out := makeIntegers(t, 5, 0, 9, delimiter, "Decimal")
			if got := strings.Count(out, between); got != 4 {
				t.Errorf("%q holds %d separators, want 4", out, got)
			}
		})
	}
}

// TestPRIGFormats covers how each integer is written.
func TestPRIGFormats(t *testing.T) {
	t.Run("Decimal", func(t *testing.T) {
		for f := range strings.SplitSeq(makeIntegers(t, 200, -50, 50, "Space", "Decimal"), " ") {
			if _, err := strconv.ParseInt(f, 10, 64); err != nil {
				t.Fatalf("%q is not a decimal number", f)
			}
		}
	})

	t.Run("Hex", func(t *testing.T) {
		for f := range strings.SplitSeq(makeIntegers(t, 200, -255, 255, "Space", "Hex"), " ") {
			v, err := strconv.ParseInt(f, 16, 64)
			if err != nil {
				t.Fatalf("%q is not a hexadecimal number", f)
			}
			if v < -255 || v > 255 {
				t.Fatalf("%q is outside the range asked for", f)
			}
			if f != strings.ToLower(f) {
				t.Errorf("%q is not in lower case", f)
			}
			if strings.HasPrefix(strings.TrimPrefix(f, "-"), "0") && len(strings.TrimPrefix(f, "-")) > 1 {
				t.Errorf("%q is padded, which it should not be", f)
			}
		}
	})

	t.Run("Raw", func(t *testing.T) {
		// Letters, so the characters drawn can be read back.
		out := makeIntegers(t, 500, 'A', 'Z', "Space", "Raw")
		if len(out) != 500 {
			t.Fatalf("500 integers gave %d characters", len(out))
		}
		for _, r := range out {
			if r < 'A' || r > 'Z' {
				t.Fatalf("drew %q, which is outside A..Z", r)
			}
		}
	})
}

// TestPRIGRawIgnoresTheDelimiter covers the one format that runs its characters
// together: there is nothing to separate when the point is a stretch of text.
func TestPRIGRawIgnoresTheDelimiter(t *testing.T) {
	for _, delimiter := range []string{"Space", "Comma", "Line feed"} {
		out := makeIntegers(t, 40, 'a', 'z', delimiter, "Raw")
		if len(out) != 40 {
			t.Errorf("with %s the output ran to %d characters, want 40", delimiter, len(out))
		}
	}
}

// TestPRIGRawBeyondTheBasicPlane covers a character that takes a surrogate pair
// to write down, which is more than one byte of output per integer.
func TestPRIGRawBeyondTheBasicPlane(t *testing.T) {
	const grinning = 0x1F600

	out := makeIntegers(t, 3, grinning, grinning, "Space", "Raw")
	if want := strings.Repeat("😀", 3); out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestPRIGRejects covers the ranges the operation will not draw from.
func TestPRIGRejects(t *testing.T) {
	const safe = int64(1)<<53 - 1

	for _, tc := range []struct {
		name             string
		count, low, high float64
		want             string
	}{
		{
			"a low bound above the high one", 1, 10, 5,
			"Min cannot be larger than Max.",
		},
		{
			"a span wider than the largest number that can be counted",
			1, 0, float64(safe),
			"Range between Min and Max cannot be larger than `2^53`",
		},
		{
			"the widest span there could be",
			1, float64(-safe), float64(safe),
			"Range between Min and Max cannot be larger than `2^53`",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Pseudo-Random Integer Generator", "",
				tc.count, tc.low, tc.high, "Space", "Decimal")
			if err == nil {
				t.Fatalf("drew from it anyway, giving %q", out)
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestPRIGArgumentsRefused covers the bounds declared on the arguments, which
// the engine checks before the operation runs.
func TestPRIGArgumentsRefused(t *testing.T) {
	const safe = float64(int64(1)<<53 - 1)

	op, ok := core.Default.Get("Pseudo-Random Integer Generator")
	if !ok {
		t.Fatal("Pseudo-Random Integer Generator is not registered")
	}

	for _, tc := range []struct {
		name string
		args []any
	}{
		{"no integers at all", []any{0.0, 0.0, 9.0, "Space", "Decimal"}},
		{"fewer than none", []any{-1.0, 0.0, 9.0, "Space", "Decimal"}},
		{"a low bound below what can be counted", []any{1.0, -safe - 1, 9.0, "Space", "Decimal"}},
		{"a high bound above it", []any{1.0, 0.0, safe + 1, "Space", "Decimal"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := core.CoerceArgs(op.Args(), tc.args); err == nil {
				t.Errorf("accepted %v", tc.args)
			}
		})
	}
}

// TestPRIGWriting covers how one integer is written in each format. The recipe
// engine checks the option before the operation runs, so the fall-through for a
// format the operation does not offer is only reachable by a direct call.
func TestPRIGWriting(t *testing.T) {
	for _, tc := range []struct {
		format string
		value  int64
		want   string
	}{
		{"Decimal", 42, "42"},
		{"Decimal", -42, "-42"},
		{"Decimal", 1<<53 - 1, "9007199254740991"},
		{"Hex", 42, "2a"},
		{"Hex", 255, "ff"},
		{"Hex", 15, "f"},
		{"Hex", -42, "-2a"},
		{"Raw", 42, "*"},
		{"Raw", 0x1F600, "😀"},
		{"Octal", 42, "*"}, // anything unrecognised is written as a character
	} {
		t.Run(tc.format+"/"+strconv.FormatInt(tc.value, 10), func(t *testing.T) {
			if got := prigWrite(tc.value, tc.format); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
