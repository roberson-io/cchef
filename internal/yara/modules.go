package yara

import (
	"crypto/md5"  // #nosec G501 -- the hash module offers it and rules depend on it
	"crypto/sha1" // #nosec G505 -- likewise
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"math"
	"strings"
	"time"
)

// The modules a rule may reach into.
//
// Each one says what it offers in moduleSchemas, which is what a rule is
// checked against while it compiles, and builds those same names into values in
// moduleBuilders once there is data to read.

// byteValues is how many values a byte can take, which the counts below are
// spread over.
const byteValues = 256

// meanBytes is the average byte of data spread evenly over every value, which
// the math module offers as a yardstick to compare real data against.
const meanBytes = 127.5

// noCorrelation is what libyara gives when a stretch of data offers nothing to
// correlate, in place of a division by zero.
const noCorrelation = -100000.0

// moduleSchemas is what each module declares. A rule that names anything not
// here is refused before it runs.
var moduleSchemas = map[string]*modDecl{
	"hash":    hashSchema(),
	"math":    mathSchema(),
	"console": consoleSchema(),
	"time":    timeSchema(),
	"elf":     elfSchema(),
	"pe":      peSchema(),
	"dotnet":  dotnetSchema(),
}

// moduleBuilders builds what each module offers over the data being scanned.
var moduleBuilders = map[string]func(*evaluator) modValue{
	"hash":    hashModule,
	"math":    mathModule,
	"console": consoleModule,
	"time":    timeModule,
	"elf":     elfModule,
	"pe":      peModule,
	"dotnet":  dotnetModule,
}

func hashSchema() *modDecl {
	digest := decFunc(modString, "ii", "s")
	return decStruct(map[string]*modDecl{
		"md5": digest, "sha1": digest, "sha256": digest,
		"checksum32": decFunc(modInt, "ii", "s"),
		"crc32":      decFunc(modInt, "ii", "s"),
	})
}

func mathSchema() *modDecl {
	return decStruct(map[string]*modDecl{
		"MEAN_BYTES":         decFloat(),
		"in_range":           decFunc(modInt, "fff"),
		"deviation":          decFunc(modFloat, "iif", "sf"),
		"mean":               decFunc(modFloat, "ii", "s"),
		"serial_correlation": decFunc(modFloat, "ii", "s"),
		"monte_carlo_pi":     decFunc(modFloat, "ii", "s"),
		"entropy":            decFunc(modFloat, "ii", "s"),
		"min":                decFunc(modInt, "ii"),
		"max":                decFunc(modInt, "ii"),
		"to_number":          decFunc(modInt, "b"),
		"abs":                decFunc(modInt, "i"),
		"count":              decFunc(modInt, "iii", "i"),
		"percentage":         decFunc(modFloat, "iii", "i"),
		"mode":               decFunc(modInt, "ii", ""),
	})
}

func consoleSchema() *modDecl {
	return decStruct(map[string]*modDecl{
		"log": decFunc(modInt, "s", "ss", "i", "si", "f", "sf"),
		"hex": decFunc(modInt, "i", "si"),
	})
}

func timeSchema() *modDecl {
	return decStruct(map[string]*modDecl{"now": decFunc(modInt, "")})
}

// hashModule works out digests over a stretch of the data, or over text a rule
// wrote out itself.
func hashModule(*evaluator) modValue {
	digest := func(sum func([]byte) string) modValue {
		return funcOf(func(e *evaluator, args []value) (value, error) {
			data, ok := e.dataArgument(args)
			if !ok {
				return undefined, nil
			}
			return stringValue(sum(data)), nil
		})
	}
	checksum := funcOf(func(e *evaluator, args []value) (value, error) {
		data, ok := e.dataArgument(args)
		if !ok {
			return undefined, nil
		}
		var total int64
		for _, b := range data {
			total += int64(b)
		}
		return intValue(total), nil
	})

	return structOf(map[string]modValue{
		"md5": digest(func(b []byte) string {
			sum := md5.Sum(b) // #nosec G401 -- the module offers md5 and rules depend on it
			return hex.EncodeToString(sum[:])
		}),
		"sha1": digest(func(b []byte) string {
			sum := sha1.Sum(b) // #nosec G401 -- likewise
			return hex.EncodeToString(sum[:])
		}),
		"sha256": digest(func(b []byte) string {
			sum := sha256.Sum256(b)
			return hex.EncodeToString(sum[:])
		}),
		"checksum32": checksum,
		"crc32": funcOf(func(e *evaluator, args []value) (value, error) {
			data, ok := e.dataArgument(args)
			if !ok {
				return undefined, nil
			}
			return intValue(int64(crc32.ChecksumIEEE(data))), nil
		}),
	})
}

// dataArgument reads the bytes a module was pointed at: either a stretch of the
// data being scanned, or text the rule wrote out itself.
func (e *evaluator) dataArgument(args []value) ([]byte, bool) {
	if len(args) == 1 && args[0].kind == valueString {
		return []byte(args[0].s), true
	}
	if len(args) != 2 || args[0].kind != valueInt || args[1].kind != valueInt {
		return nil, false
	}
	at, size := args[0].i, args[1].i
	if at < 0 || size < 0 || at > int64(len(e.buf.data)) {
		return nil, false
	}
	if at+size > int64(len(e.buf.data)) {
		size = int64(len(e.buf.data)) - at
	}
	return e.buf.data[at : at+size], true
}

// mathModule works out numbers about the data, or about numbers it was handed.
func mathModule(*evaluator) modValue {
	spread := func(of func([]byte) float64) modValue {
		return funcOf(func(e *evaluator, args []value) (value, error) {
			data, ok := e.dataArgument(args)
			if !ok {
				return undefined, nil
			}
			return floatOrNothing(of(data)), nil
		})
	}

	return structOf(map[string]modValue{
		"MEAN_BYTES":         valueOf(floatValue(meanBytes)),
		"entropy":            spread(entropyOf),
		"mean":               spread(meanOf),
		"serial_correlation": spread(serialCorrelationOf),
		"monte_carlo_pi":     spread(monteCarloPiOf),
		"deviation":          funcOf(deviation),
		"in_range":           funcOf(inRange),
		"min":                funcOf(smaller),
		"max":                funcOf(larger),
		"to_number":          funcOf(toNumber),
		"abs":                funcOf(absolute),
		"count":              funcOf(countByte),
		"percentage":         funcOf(percentageOfByte),
		"mode":               funcOf(commonestByte),
	})
}

// deviation is how far the bytes of a stretch wander from a given middle, which
// the rule names rather than the data deciding it.
func deviation(e *evaluator, args []value) (value, error) {
	if len(args) == 0 {
		return undefined, nil
	}
	mean, ok := args[len(args)-1].number()
	if !ok {
		return undefined, nil
	}
	data, ok := e.dataArgument(args[:len(args)-1])
	if !ok {
		return undefined, nil
	}
	sum := 0.0
	for _, c := range data {
		sum += math.Abs(float64(c) - mean)
	}
	return floatOrNothing(sum / float64(len(data))), nil
}

// floatOrNothing reads a sum that could not be done — a division by no data at
// all — as no answer rather than as a number.
func floatOrNothing(f float64) value {
	if math.IsNaN(f) {
		return undefined
	}
	return floatValue(f)
}

// inRange says whether a number falls between two others.
func inRange(_ *evaluator, args []value) (value, error) {
	if len(args) != 3 {
		return undefined, nil
	}
	x, xok := args[0].number()
	low, lok := args[1].number()
	high, hok := args[2].number()
	if !xok || !lok || !hok {
		return undefined, nil
	}
	return boolValue(low <= x && x <= high), nil
}

// smaller and larger pick between two whole numbers. libyara compares them as
// if they had no sign, so a negative number counts as a very large one.
func smaller(_ *evaluator, args []value) (value, error) { return pick(args, true) }
func larger(_ *evaluator, args []value) (value, error)  { return pick(args, false) }

func pick(args []value, wantSmaller bool) (value, error) {
	if len(args) != 2 || args[0].kind != valueInt || args[1].kind != valueInt {
		return undefined, nil
	}
	first, second := uint64(args[0].i), uint64(args[1].i) // #nosec G115 -- libyara compares without sign
	if (first < second) == wantSmaller {
		return args[0], nil
	}
	return args[1], nil
}

// toNumber turns a yes or no into one or nought.
func toNumber(_ *evaluator, args []value) (value, error) {
	if len(args) != 1 || args[0].kind == valueUndefined {
		return undefined, nil
	}
	return boolValue(args[0].truth()), nil
}

// absolute is the size of a number without its sign.
func absolute(_ *evaluator, args []value) (value, error) {
	if len(args) != 1 || args[0].kind != valueInt {
		return undefined, nil
	}
	if args[0].i < 0 {
		return intValue(-args[0].i), nil
	}
	return args[0], nil
}

// countByte is how often one byte turns up, over a stretch of the data or over
// all of it.
func countByte(e *evaluator, args []value) (value, error) {
	counts, at, ok := e.distributionArgument(args)
	if !ok {
		return undefined, nil
	}
	return intValue(int64(counts[at])), nil
}

// percentageOfByte is how much of the data one byte accounts for. libyara works
// this out to less precision than the rest of the module, and a rule comparing
// against a written-out number can tell.
func percentageOfByte(e *evaluator, args []value) (value, error) {
	counts, at, ok := e.distributionArgument(args)
	if !ok {
		return undefined, nil
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	return floatOrNothing(float64(float32(counts[at]) / float32(total))), nil
}

// commonestByte is the byte that turns up most often, the earliest of them if
// several are level.
func commonestByte(e *evaluator, args []value) (value, error) {
	data, ok := e.dataArgument(args)
	if len(args) == 0 {
		data, ok = e.buf.data, true
	}
	if !ok {
		return undefined, nil
	}
	counts := distributionOf(data)
	best := 0
	for b, n := range counts[:byteValues] {
		if n > counts[best] {
			best = b
		}
	}
	return intValue(int64(best)), nil
}

// distributionArgument reads the arguments the counting functions share: a byte
// to look for, and then either a stretch of the data or nothing, meaning all of
// it.
func (e *evaluator) distributionArgument(args []value) ([]int, int, bool) {
	if len(args) == 0 || args[0].kind != valueInt {
		return nil, 0, false
	}
	data := e.buf.data
	if len(args) > 1 {
		var ok bool
		if data, ok = e.dataArgument(args[1:]); !ok {
			return nil, 0, false
		}
	}
	counts := distributionOf(data)
	at := args[0].i
	// A byte outside the range one can hold was never there to be counted, so
	// it is counted no times rather than leaving no answer at all.
	if at < 0 || at >= byteValues {
		return counts, byteValues, true
	}
	return counts, int(at), true
}

// distributionOf counts how often each byte turns up.
// The extra place at the end holds the count of a byte that cannot be, which is
// always none.
func distributionOf(data []byte) []int {
	counts := make([]int, byteValues+1)
	for _, c := range data {
		counts[c]++
	}
	return counts
}

// entropyOf is how many bits each byte of a stretch carries, which is high for
// data that has been compressed or encrypted and low for plain text.
func entropyOf(data []byte) float64 {
	entropy := 0.0
	for _, n := range distributionOf(data)[:byteValues] {
		if n == 0 {
			continue
		}
		p := float64(n) / float64(len(data))
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// meanOf is the average byte of a stretch.
func meanOf(data []byte) float64 {
	total := 0.0
	for _, c := range data {
		total += float64(c)
	}
	return total / float64(len(data))
}

// serialCorrelationOf is how much each byte depends on the one before it, which
// is near zero for data with no pattern to it. The last byte is taken to be
// followed by the first, so the stretch closes on itself.
func serialCorrelationOf(data []byte) float64 {
	var t1, t2, t3, last float64
	for _, c := range data {
		this := float64(c)
		t1 += last * this
		t2 += this
		t3 += this * this
		last = this
	}
	if len(data) > 0 {
		t1 += last * float64(data[0])
	}
	n := float64(len(data))
	t2 *= t2
	bottom := n*t3 - t2
	if bottom == 0 {
		return noCorrelation
	}
	return (n*t1 - t2) / bottom
}

// monteCarloPiOf reads the data as points on a square and asks how many land
// inside the circle it holds, which comes to pi for data with no pattern to it.
// The answer is how far off pi that estimate is.
func monteCarloPiOf(data []byte) float64 {
	// Each point takes six bytes: three for how far across, three for how far
	// up. inCircle is the radius of the circle those points fall in, squared.
	const pointBytes, halfPoint = 6, 3
	inCircle := math.Pow(math.Pow(byteValues, halfPoint)-1, 2)

	var monte [pointBytes]float64
	points, inside := 0, 0
	for i, c := range data {
		monte[i%pointBytes] = float64(c)
		if i%pointBytes != pointBytes-1 {
			continue
		}
		points++
		var across, up float64
		for j := range halfPoint {
			across = across*byteValues + monte[j]
			up = up*byteValues + monte[j+halfPoint]
		}
		if across*across+up*up <= inCircle {
			inside++
		}
	}
	if points == 0 {
		return math.NaN()
	}
	return math.Abs((4*float64(inside)/float64(points) - math.Pi) / math.Pi)
}

// consoleModule notes something for the scan to report, and is always true so
// that it can be written into a condition without changing the answer.
func consoleModule(*evaluator) modValue {
	note := func(asHex bool) modValue {
		return funcOf(func(e *evaluator, args []value) (value, error) {
			// Asked to say something that has no value, there is nothing to
			// say, so nothing is noted and the note itself has no answer.
			for _, arg := range args {
				if arg.kind == valueUndefined {
					return undefined, nil
				}
			}
			*e.logs = append(*e.logs, logMessage(args, asHex))
			return yes, nil
		})
	}
	return structOf(map[string]modValue{"log": note(false), "hex": note(true)})
}

// logMessage renders what the console module was handed: text on its own, or
// text and a number, written plainly or in hex.
//
// Where a note is given two things, the first names the second and is written
// as it stands. What is actually being noted, when it is text, has every byte
// that cannot be printed written as its value instead, so that a note stays
// readable however odd the text it was handed.
func logMessage(args []value, asHex bool) string {
	var b strings.Builder
	for at, arg := range args {
		named := len(args) == 2 && at == 0
		switch {
		case arg.kind == valueString && named:
			b.WriteString(arg.s)
		case arg.kind == valueString:
			b.WriteString(printableText(arg.s))
		case arg.kind == valueInt && asHex:
			fmt.Fprintf(&b, "0x%x", arg.i)
		case arg.kind == valueInt:
			fmt.Fprintf(&b, "%d", arg.i)
		case arg.kind == valueFloat:
			fmt.Fprintf(&b, "%f", arg.f)
		}
	}
	return b.String()
}

// printableText writes text with every byte that cannot be printed given as its
// value instead. Only the plain printable characters are kept as they are.
func printableText(s string) string {
	const firstPrintable, lastPrintable = ' ', '~'
	var b strings.Builder
	for i := range len(s) {
		if s[i] >= firstPrintable && s[i] <= lastPrintable {
			b.WriteByte(s[i])
			continue
		}
		fmt.Fprintf(&b, `\x%02x`, s[i])
	}
	return b.String()
}

// timeModule reads the clock.
func timeModule(*evaluator) modValue {
	return structOf(map[string]modValue{
		"now": funcOf(func(*evaluator, []value) (value, error) {
			return intValue(time.Now().Unix()), nil
		}),
	})
}
