package ops

import (
	"math"
	"strconv"
	"strings"
)

// Ports of the parts of d3-scale and d3-array that CyberChef's chart operations
// rely on, plus JavaScript's number formatting. The charts emit these values
// straight into SVG attributes, so they have to match d3 exactly.

// e10, e5 and e2 are d3-array's tick-step thresholds: sqrt(50), sqrt(10) and
// sqrt(2). A candidate step is rounded up to 10, 5 or 2 times a power of ten
// according to which of these it exceeds.
var (
	e10 = math.Sqrt(50)
	e5  = math.Sqrt(10)
	e2  = math.Sqrt(2)
)

// tickIncrement returns d3's step between ticks over [start, stop] for roughly
// count ticks. A negative result means "divide by -result" rather than multiply,
// which keeps fractional steps exact.
func tickIncrement(start, stop float64, count int) float64 {
	step := (stop - start) / math.Max(0, float64(count))
	power := math.Floor(math.Log10(step))
	errorRatio := step / math.Pow(10, power)

	var factor float64
	switch {
	case errorRatio >= e10:
		factor = 10
	case errorRatio >= e5:
		factor = 5
	case errorRatio >= e2:
		factor = 2
	default:
		factor = 1
	}
	if power >= 0 {
		return factor * math.Pow(10, power)
	}
	return -math.Pow(10, -power) / factor
}

// d3Ticks returns the tick values d3 would place over [start, stop]. Fractional
// steps are computed as i/step rather than i*step so the values render without
// floating-point noise, exactly as d3 does.
func d3Ticks(start, stop float64, count int) []float64 {
	if count <= 0 {
		return nil
	}
	if start == stop {
		return []float64{start}
	}
	reverse := stop < start
	if reverse {
		start, stop = stop, start
	}

	step := tickIncrement(start, stop, count)
	if step == 0 || math.IsInf(step, 0) {
		return nil
	}

	var ticks []float64
	if step > 0 {
		r0 := math.Round(start / step)
		r1 := math.Round(stop / step)
		if r0*step < start {
			r0++
		}
		if r1*step > stop {
			r1--
		}
		for i := 0.0; i <= r1-r0; i++ {
			ticks = append(ticks, (r0+i)*step)
		}
	} else {
		step = -step
		r0 := math.Round(start * step)
		r1 := math.Round(stop * step)
		if r0/step < start {
			r0++
		}
		if r1/step > stop {
			r1--
		}
		for i := 0.0; i <= r1-r0; i++ {
			ticks = append(ticks, (r0+i)/step)
		}
	}

	if reverse {
		for i, j := 0, len(ticks)-1; i < j; i, j = i+1, j-1 {
			ticks[i], ticks[j] = ticks[j], ticks[i]
		}
	}
	return ticks
}

// linearScale maps a numeric domain onto a numeric range.
type linearScale struct {
	domain, rng [2]float64
}

// scaleLinear builds a linear scale over the given domain and range.
func scaleLinear(domain, rng [2]float64) linearScale {
	return linearScale{domain: domain, rng: rng}
}

// scale maps a domain value onto the range. A zero-width domain maps everything
// to the middle of the range, as d3's normalise step does.
func (s linearScale) scale(v float64) float64 {
	d0, d1 := s.domain[0], s.domain[1]
	t := 0.5
	if d1 != d0 {
		t = (v - d0) / (d1 - d0)
	}
	// d3-interpolate computes a*(1-t) + b*t. The algebraically equivalent
	// a + t*(b-a) rounds differently, and the result is printed into the SVG.
	// The conversions stop Go fusing a multiply and add into an FMA, which
	// rounds once where JavaScript rounds twice.
	return float64(s.rng[0]*(1-t)) + float64(s.rng[1]*t)
}

// ticks returns the scale's tick values.
func (s linearScale) ticks(count int) []float64 {
	return d3Ticks(s.domain[0], s.domain[1], count)
}

// tickStep is d3-array's tickStep: the positive spacing between ticks.
func tickStep(start, stop float64, count int) float64 {
	step := tickIncrement(start, stop, count)
	if step < 0 {
		return 1 / -step
	}
	return step
}

// fixedPrecision is d3-format's precisionFixed: how many decimal places are
// needed to show a value stepping by step. A zero or non-finite step has no
// answer, which the caller turns into the specifier's default precision.
func fixedPrecision(step float64) (int, bool) {
	if step == 0 || math.IsNaN(step) || math.IsInf(step, 0) {
		return 0, false
	}
	return max(0, -int(math.Floor(math.Log10(math.Abs(step))))), true
}

// defaultFixedPrecision is the precision of a ",f" specifier that does not name
// one, matching d3-format.
const defaultFixedPrecision = 6

// tickFormat returns d3-scale's default axis label formatter for a domain: a
// ",f" specifier — grouped thousands, fixed decimals — whose precision is
// derived from the tick step.
func tickFormat(d0, d1 float64, count int) func(float64) string {
	precision := defaultFixedPrecision
	if p, ok := fixedPrecision(tickStep(d0, d1, count)); ok {
		precision = p
	}
	return func(v float64) string {
		return formatGrouped(v, precision)
	}
}

// unicodeMinus is the sign d3-format's default locale uses for negatives; it is
// U+2212 MINUS SIGN rather than an ASCII hyphen.
const unicodeMinus = "−"

// groupSize is the number of integer digits between thousands separators.
const groupSize = 3

// formatGrouped renders v with the given number of decimals and comma-grouped
// thousands, as d3-format's ",f" does.
func formatGrouped(v float64, precision int) string {
	sign := ""
	if math.Signbit(v) && v != 0 {
		sign, v = unicodeMinus, -v
	}
	s := strconv.FormatFloat(v, 'f', precision, 64)
	intPart, frac, hasFrac := strings.Cut(s, ".")

	var b strings.Builder
	for i, digit := range intPart {
		if i > 0 && (len(intPart)-i)%groupSize == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(digit)
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return sign + b.String()
}

// pointScale places discrete domain values evenly along a range, as d3's
// scalePoint does. It backs the series chart's x axis.
type pointScale struct {
	positions map[string]float64
}

// scalePoint spreads domain evenly across rng. A single value sits in the
// middle; otherwise the first and last sit on the range's ends.
func scalePoint(domain []string, rng [2]float64) pointScale {
	s := pointScale{positions: make(map[string]float64, len(domain))}
	if len(domain) == 0 {
		return s
	}
	span := rng[1] - rng[0]
	if len(domain) == 1 {
		s.positions[domain[0]] = rng[0] + span/2
		return s
	}
	step := span / float64(len(domain)-1)
	for i, v := range domain {
		if _, seen := s.positions[v]; !seen {
			s.positions[v] = rng[0] + float64(i)*step
		}
	}
	return s
}

// scale returns the position of a domain value, or 0 if it is not in the domain.
func (s pointScale) scale(v string) float64 {
	pos, _ := s.lookup(v)
	return pos
}

// lookup returns a domain value's position and whether it is in the domain.
func (s pointScale) lookup(v string) (float64, bool) {
	pos, ok := s.positions[v]
	return pos, ok
}
