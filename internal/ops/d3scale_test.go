package ops

import (
	"math"
	"testing"
)

// jsNum must render doubles the way JavaScript's Number#toString does, since
// those strings land verbatim in the chart SVG. Expected values were taken from
// Node.
func TestJSNum(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{0.5, "0.5"},
		{39.166666666666664, "39.166666666666664"},
		{300.50000000000006, "300.50000000000006"},
		{100, "100"},
		{470.5, "470.5"},
		{-225, "-225"},
		{0.32, "0.32"},
		{1.0 / 3.0, "0.3333333333333333"},
		{2.0 / 3.0, "0.6666666666666666"},
		// JavaScript switches to exponential notation only below 1e-6 and at
		// or above 1e21; Go's %g would switch far sooner.
		{0.000001, "0.000001"},
		{0.0000001, "1e-7"},
		{1.5e-7, "1.5e-7"},
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},
		{1e300, "1e+300"},
		{5e-324, "5e-324"},
		{123456789012345680000, "123456789012345680000"},
	} {
		if got := jsNum(tc.in); got != tc.want {
			t.Errorf("jsNum(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// d3Ticks is d3-array's ticks(): the tick values an axis draws. Expected values
// were taken from d3 under Node.
func TestD3Ticks(t *testing.T) {
	for _, tc := range []struct {
		start, stop float64
		count       int
		want        []float64
	}{
		{80, 320, 10, []float64{80, 100, 120, 140, 160, 180, 200, 220, 240, 260, 280, 300, 320}},
		{0, 1, 10, []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}},
		{0, 1, 5, []float64{0, 0.2, 0.4, 0.6, 0.8, 1}},
		{100, 500, 10, []float64{100, 150, 200, 250, 300, 350, 400, 450, 500}},
		{1, 9, 10, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{-5, 5, 10, []float64{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5}},
		{0, 100, 3, []float64{0, 50, 100}},
		{1, 1, 10, []float64{1}},
	} {
		got := d3Ticks(tc.start, tc.stop, tc.count)
		if len(got) != len(tc.want) {
			t.Errorf("ticks(%v,%v,%d) = %v, want %v", tc.start, tc.stop, tc.count, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("ticks(%v,%v,%d)[%d] = %v, want %v", tc.start, tc.stop, tc.count, i, got[i], tc.want[i])
			}
		}
	}
}

// Ticks are rendered through jsNum, so fractional steps must not pick up
// floating-point noise.
func TestD3TicksRenderCleanly(t *testing.T) {
	var got []string
	for _, v := range d3Ticks(2, 3, 10) {
		got = append(got, jsNum(v))
	}
	want := []string{"2", "2.1", "2.2", "2.3", "2.4", "2.5", "2.6", "2.7", "2.8", "2.9", "3"}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("rendered ticks = %v, want %v", got, want)
		}
	}
}

// scaleLinear maps a domain onto a range, and is what turns data values into
// pixel coordinates.
func TestScaleLinear(t *testing.T) {
	s := scaleLinear([2]float64{80, 320}, [2]float64{0, 470})
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{100, "39.166666666666664"},
		{200, "235"},
		{320, "470"},
		{80, "0"},
	} {
		if got := jsNum(s.scale(tc.in)); got != tc.want {
			t.Errorf("scale(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// An inverted range (as the y axis uses) maps the domain minimum to the
	// bottom of the chart.
	y := scaleLinear([2]float64{80, 320}, [2]float64{450, 0})
	if got := jsNum(y.scale(100)); got != "412.5" {
		t.Errorf("inverted scale(100) = %q, want 412.5", got)
	}
}

// A zero-width domain (a single data point, or a column of identical values)
// must not divide by zero. d3 maps every input to the middle of the range.
func TestScaleLinearDegenerate(t *testing.T) {
	s := scaleLinear([2]float64{5, 5}, [2]float64{0, 100})
	for _, in := range []float64{5, 7, -3} {
		if got := s.scale(in); got != 50 {
			t.Errorf("degenerate scale(%v) = %v, want 50 (range midpoint)", in, got)
		}
	}
	if got := d3Ticks(5, 5, 10); len(got) != 1 || got[0] != 5 {
		t.Errorf("degenerate ticks = %v, want [5]", got)
	}
}

// tickFormat is d3-scale's default axis label formatter: a ",f" specifier whose
// precision comes from the tick step. Expected values were taken from d3.
func TestTickFormat(t *testing.T) {
	for _, tc := range []struct {
		d0, d1 float64
		count  int
		want   []string
	}{
		{80, 320, 10, []string{"80", "100", "120"}},
		{0, 1, 5, []string{"0.0", "0.2", "0.4", "0.6", "0.8", "1.0"}},
		{0, 0.1, 10, []string{"0.00", "0.01", "0.02"}},
		{1, 3, 5, []string{"1.0", "1.5", "2.0", "2.5", "3.0"}},
		{0.001, 0.002, 5, []string{"0.0010", "0.0012", "0.0014"}},
		// Thousands are grouped with commas.
		{0, 1e6, 5, []string{"0", "200,000", "400,000"}},
		// A degenerate domain has a zero step, so the precision falls back to
		// the specifier default of six decimals.
		{1, 1, 5, []string{"1.000000"}},
	} {
		format := tickFormat(tc.d0, tc.d1, tc.count)
		ticks := d3Ticks(tc.d0, tc.d1, tc.count)
		for i, want := range tc.want {
			if i >= len(ticks) {
				t.Fatalf("domain [%v,%v]: only %d ticks, want at least %d", tc.d0, tc.d1, len(ticks), len(tc.want))
			}
			if got := format(ticks[i]); got != want {
				t.Errorf("domain [%v,%v] tick %d = %q, want %q", tc.d0, tc.d1, i, got, want)
			}
		}
	}
}

// d3-format renders a negative number with a Unicode minus (U+2212), not an
// ASCII hyphen.
func TestTickFormatUnicodeMinus(t *testing.T) {
	format := tickFormat(-2000, 2000, 5)
	if got := format(-1000); got != "−1,000" {
		t.Errorf("format(-1000) = %q, want %q", got, "−1,000")
	}
}

// scalePoint spreads discrete values evenly across the range, placing a single
// value in the middle.
func TestScalePoint(t *testing.T) {
	for _, tc := range []struct {
		domain []string
		want   []float64
	}{
		{[]string{"1", "2", "3"}, []float64{0, 215, 430}},
		{[]string{"a"}, []float64{215}},
		{[]string{"a", "b"}, []float64{0, 430}},
		{[]string{"x", "y", "z", "w"}, []float64{0, 143.33333333333334, 286.6666666666667, 430}},
	} {
		s := scalePoint(tc.domain, [2]float64{0, 430})
		for i, v := range tc.domain {
			if got := s.scale(v); got != tc.want[i] {
				t.Errorf("domain %v: scale(%q) = %v, want %v", tc.domain, v, got, tc.want[i])
			}
		}
	}
}

// A value outside the domain has no position.
func TestScalePointUnknown(t *testing.T) {
	s := scalePoint([]string{"a", "b"}, [2]float64{0, 100})
	if _, ok := s.lookup("zzz"); ok {
		t.Error("expected no position for a value outside the domain")
	}
}

// Degenerate and reversed inputs to the tick algorithm.
func TestD3TicksEdgeCases(t *testing.T) {
	if got := d3Ticks(0, 1, 0); got != nil {
		t.Errorf("zero count = %v, want none", got)
	}
	// A reversed range yields the same ticks in reverse.
	got := d3Ticks(320, 80, 10)
	if len(got) == 0 || got[0] != 320 || got[len(got)-1] != 80 {
		t.Errorf("reversed ticks = %v, want 320 down to 80", got)
	}
	if got := d3Ticks(0, math.Inf(1), 10); got != nil {
		t.Errorf("infinite range = %v, want none", got)
	}
	// A fractional step must not overshoot the stop value.
	for _, v := range d3Ticks(0.05, 0.95, 10) {
		if v < 0.05 || v > 0.95 {
			t.Errorf("tick %v outside [0.05, 0.95]", v)
		}
	}
	// An integral step must not overshoot either.
	for _, v := range d3Ticks(1, 99, 10) {
		if v < 1 || v > 99 {
			t.Errorf("tick %v outside [1, 99]", v)
		}
	}
}

// An empty domain has no positions at all.
func TestScalePointEmpty(t *testing.T) {
	s := scalePoint(nil, [2]float64{0, 100})
	if _, ok := s.lookup("a"); ok {
		t.Error("expected no position from an empty domain")
	}
}

// Repeated domain values keep their first position.
func TestScalePointDuplicates(t *testing.T) {
	s := scalePoint([]string{"a", "b", "a"}, [2]float64{0, 100})
	if got := s.scale("a"); got != 0 {
		t.Errorf("duplicate value moved to %v, want its first position 0", got)
	}
}
