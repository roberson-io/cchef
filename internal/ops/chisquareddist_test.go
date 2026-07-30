package ops

import (
	"math"
	"testing"
)

// The expected values are the distribution's true ones, computed to forty
// significant figures and rounded to a float64. They are deliberately not the
// values CyberChef gets: the chi-squared package it uses is inaccurate from
// about the seventh decimal place (it puts the closed-form 1-exp(-1) at
// 0.63212044740307971 rather than 0.63212055882855767). What the number is
// used for — whether a probability is above zero — is unaffected.
func TestChiSquaredCDF(t *testing.T) {
	cases := []struct {
		x, k, want float64
	}{
		{0, 1, 0},
		{0.5, 1, 0.52049987781304652},
		{1, 1, 0.68268949213708585},
		{2.5, 1, 0.88615370199334198},
		{3.84, 1, 0.94995647875129485},
		{1, 2, 0.39346934028736658},
		{2, 2, 0.63212055882855767},
		{5, 2, 0.91791500137610116},
		{10, 3, 0.9814338645369568},
		{255, 255, 0.51177747822959363},
		{300, 255, 0.97227247794609517},
		{200, 255, 0.0045745554580481048},
		{0.001, 255, 0},
		{1000, 255, 1},
		{12.5, 10, 0.74701467669070176},
		{0.5, 0.5, 0.74367794473146109},
	}
	// A hundred-odd terms of a series in float64 loses the last digit or two at
	// the larger degrees of freedom, so the comparison allows for that and no
	// more.
	for _, c := range cases {
		got := chiSquaredCDF(c.x, c.k)
		if math.Abs(got-c.want) > 1e-14 {
			t.Errorf("cdf(%v, %v) = %.17g, want %.17g", c.x, c.k, got, c.want)
		}
	}
}

// TestChiSquaredCDFAgainstClosedForms checks the two degrees of freedom whose
// distribution can be written down exactly, so the port is anchored to the
// mathematics rather than to another implementation of it.
func TestChiSquaredCDFAgainstClosedForms(t *testing.T) {
	for _, x := range []float64{0.25, 1, 2, 5, 20} {
		// Two degrees of freedom is the exponential distribution.
		if got, want := chiSquaredCDF(x, 2), 1-math.Exp(-x/2); math.Abs(got-want) > 1e-15 {
			t.Errorf("cdf(%v, 2) = %.17g, want %.17g", x, got, want)
		}
		// One degree of freedom is the error function.
		if got, want := chiSquaredCDF(x, 1), math.Erf(math.Sqrt(x/2)); math.Abs(got-want) > 1e-15 {
			t.Errorf("cdf(%v, 1) = %.17g, want %.17g", x, got, want)
		}
	}
}

// TestChiSquaredCDFEdges covers the values outside the distribution's domain,
// where the probability is nothing rather than a number.
func TestChiSquaredCDFEdges(t *testing.T) {
	for _, x := range []float64{-1, -0.0001} {
		if got := chiSquaredCDF(x, 3); got != 0 {
			t.Errorf("cdf(%v, 3) = %v, want 0", x, got)
		}
	}
	// A very large score is certain, and must not run away or return NaN.
	if got := chiSquaredCDF(1e9, 255); got != 1 {
		t.Errorf("cdf(1e9, 255) = %v, want 1", got)
	}
	if got := chiSquaredCDF(math.Inf(1), 10); got != 1 {
		t.Errorf("cdf(+Inf, 10) = %v, want 1", got)
	}
}

// TestChiSquaredCDFIsMonotonic checks the shape of the curve: never falling,
// and always a probability.
func TestChiSquaredCDFIsMonotonic(t *testing.T) {
	for _, k := range []float64{1, 2, 10, 255} {
		prev := 0.0
		for x := 0.0; x < 3*k+50; x += 0.25 {
			got := chiSquaredCDF(x, k)
			if got < prev-1e-12 {
				t.Fatalf("k=%v: cdf fell from %v to %v at x=%v", k, prev, got, x)
			}
			if got < 0 || got > 1 {
				t.Fatalf("k=%v: cdf(%v) = %v is not a probability", k, x, got)
			}
			prev = got
		}
	}
}

// TestAwayFromZero checks the guard that keeps the continued fraction's
// denominators divisible, which the arguments this operation produces never
// reach but the method requires.
func TestAwayFromZero(t *testing.T) {
	if got := awayFromZero(0); got != chiSquaredTiny {
		t.Errorf("awayFromZero(0) = %v, want %v", got, chiSquaredTiny)
	}
	if got := awayFromZero(math.Copysign(0, -1)); got != chiSquaredTiny {
		t.Errorf("awayFromZero(negative zero) = %v, want %v", got, chiSquaredTiny)
	}
	if got := awayFromZero(1e-320); got != chiSquaredTiny {
		t.Errorf("a number below the floor was kept: %v", got)
	}
	for _, v := range []float64{1, -1, 1e-200, -1e-200} {
		if got := awayFromZero(v); got != v {
			t.Errorf("awayFromZero(%v) = %v, want it unchanged", v, got)
		}
	}
}
