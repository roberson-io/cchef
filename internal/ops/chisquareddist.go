package ops

import "math"

// The chi-squared distribution, needed to turn a goodness-of-fit score into a
// probability. This is the distribution itself, not the Chi Square operation in
// chisquare.go, which measures a score and does not need one. CyberChef reaches for the chi-squared npm package; this is the
// same mathematics written out, so no dependency is added.
//
// The cumulative distribution function of chi-squared with k degrees of freedom
// is the regularized lower incomplete gamma function P(k/2, x/2), computed
// either from its series or from the continued fraction of the upper function,
// whichever converges quickly for the arguments given.

// chiSquaredIterations bounds both expansions; each converges in far fewer
// steps than this for every argument the operation produces.
const chiSquaredIterations = 300

// chiSquaredEpsilon is the relative size at which a term stops mattering.
const chiSquaredEpsilon = 1e-16

// chiSquaredTiny stands in for zero where a zero would be divided by.
const chiSquaredTiny = 1e-300

// awayFromZero keeps a denominator far enough from zero to divide by, which is
// what Lentz's method needs to stay well behaved.
func awayFromZero(v float64) float64 {
	if math.Abs(v) < chiSquaredTiny {
		return chiSquaredTiny
	}
	return v
}

// chiSquaredCDF returns the probability that a chi-squared variable with k
// degrees of freedom is at most x.
func chiSquaredCDF(x, k float64) float64 {
	if math.IsNaN(x) || math.IsNaN(k) || x <= 0 || k <= 0 {
		return 0
	}
	if math.IsInf(x, 1) {
		return 1
	}
	return lowerGammaP(k/2, x/2)
}

// lowerGammaP is the regularized lower incomplete gamma function P(a, x), for
// positive a and x; chiSquaredCDF has already dealt with the rest.
func lowerGammaP(a, x float64) float64 {
	// The series converges fast below the peak, the continued fraction above it.
	if x < a+1 {
		return gammaSeries(a, x)
	}
	return 1 - gammaContinuedFraction(a, x)
}

// gammaSeries evaluates P(a, x) by its series expansion, which converges
// quickly while x is below a+1.
func gammaSeries(a, x float64) float64 {
	term := 1 / a
	sum := term
	for n := 1; n <= chiSquaredIterations; n++ {
		term *= x / (a + float64(n))
		sum += term
		if math.Abs(term) < math.Abs(sum)*chiSquaredEpsilon {
			break
		}
	}
	return sum * math.Exp(-x+a*math.Log(x)-lgamma(a))
}

// gammaContinuedFraction evaluates Q(a, x), the upper function, by its
// continued fraction in the modified Lentz form, which converges quickly while
// x is at or above a+1.
func gammaContinuedFraction(a, x float64) float64 {
	b := x + 1 - a
	c := 1 / chiSquaredTiny
	d := 1 / b
	h := d
	for n := 1; n <= chiSquaredIterations; n++ {
		an := -float64(n) * (float64(n) - a)
		b += 2
		d = awayFromZero(an*d + b)
		c = awayFromZero(b + an/c)
		d = 1 / d
		delta := d * c
		h *= delta
		if math.Abs(delta-1) < chiSquaredEpsilon {
			break
		}
	}
	return h * math.Exp(-x+a*math.Log(x)-lgamma(a))
}

// lgamma is the natural logarithm of the gamma function.
func lgamma(x float64) float64 {
	v, _ := math.Lgamma(x)
	return v
}
