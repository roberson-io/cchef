package ops

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// prigMaxRange is the widest span the operation will draw from. A draw is
// fifty-three bits wide, which is as much as a number counts exactly, and one is
// held back so that the span itself can be counted too.
const prigMaxRange = int64(1)<<53 - 1

// prigDrawBits is how many bits each draw is taken from.
const prigDrawBits = 53

// prigBufferBytes is how much randomness is fetched at a time. Drawing eight
// bytes at a time from the system for every integer would dominate the work
// when thousands are asked for.
const prigBufferBytes = 4096

// The ways an integer can be written.
const (
	prigRaw     = "Raw"
	prigHex     = "Hex"
	prigDecimal = "Decimal"
)

// prigDelimiters are what may go between the integers.
var prigDelimiters = []string{"Space", "Comma", "Semi-colon", "Colon", "Line feed", "CRLF"}

// The complaints about a range that cannot be drawn from.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var (
	errPRIGBackwards = errors.New("Min cannot be larger than Max.")
	errPRIGTooWide   = errors.New("Range between Min and Max cannot be larger than `2^53`")
)

// PseudoRandomIntegerGenerator draws integers from a range.
type PseudoRandomIntegerGenerator struct{}

// Meta returns the operation metadata.
func (PseudoRandomIntegerGenerator) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Pseudo-Random Integer Generator",
		Module: "Ciphers",
		Description: "A cryptographically-secure pseudo-random number generator " +
			"(PRNG).<br><br>Generates random integers within a specified range." +
			"<br><br>The supported range of integers is from <code>-(2^53 - 1)</code> " +
			"to <code>(2^53 - 1)</code>, and the span between the two may be as wide " +
			"as <code>2^53 - 1</code>.",
		InfoURL:    "https://wikipedia.org/wiki/Pseudorandom_number_generator",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (PseudoRandomIntegerGenerator) Args() []core.ArgDef {
	one := 1.0
	lowest, highest := -float64(prigMaxRange), float64(prigMaxRange)
	return []core.ArgDef{
		{Name: "Number of Integers", Type: core.ArgNumber, Value: 1.0, Min: &one},
		{Name: "Min Value", Type: core.ArgNumber, Value: 0.0, Min: &lowest, Max: &highest},
		{Name: "Max Value", Type: core.ArgNumber, Value: 99.0, Min: &lowest, Max: &highest},
		{Name: "Delimiter", Type: core.ArgOption, Value: prigDelimiters},
		{Name: "Output", Type: core.ArgOption, Value: []string{prigRaw, prigHex, prigDecimal}},
	}
}

// Run draws the integers.
func (PseudoRandomIntegerGenerator) Run(in *core.Dish, args []any) (*core.Dish, error) {
	count, _ := args[0].(float64)
	lowest, _ := args[1].(float64)
	highest, _ := args[2].(float64)
	delimiter, _ := args[3].(string)
	format, _ := args[4].(string)

	// A bound that is not a whole number is drawn in, so every integer given
	// out lies within what was asked for.
	low := int64(math.Ceil(lowest))
	high := int64(math.Floor(highest))

	if low > high {
		return nil, errPRIGBackwards
	}
	span := high - low + 1
	if span > prigMaxRange {
		return nil, errPRIGTooWide
	}

	source := &prigSource{}
	written := make([]string, int(count))
	for i := range written {
		draw := source.below(prigMaxRange - prigMaxRange%span)
		written[i] = prigWrite(low+draw%span, format)
	}

	// Raw output is a stretch of text rather than a list, so nothing goes
	// between one character and the next.
	between := charRep(delimiter)
	if format == prigRaw {
		between = ""
	}
	return core.NewDish([]byte(strings.Join(written, between)), core.TypeString), nil
}

// prigWrite writes one integer. Anything but the two numeric spellings is given
// as the character the integer stands for.
func prigWrite(value int64, format string) string {
	switch format {
	case prigHex:
		return strconv.FormatInt(value, 16)
	case prigDecimal:
		return strconv.FormatInt(value, 10)
	default:
		return jsChr(float64(value))
	}
}

// prigSource hands out fifty-three-bit draws, fetching randomness in blocks
// rather than one draw at a time.
type prigSource struct {
	block []byte
	at    int
}

// below draws a value under the limit given. Values at or above it are thrown
// away rather than folded in, since folding would favour the low end of the
// range: the limit is the largest whole number of spans that fits, so what is
// left is spread evenly.
func (s *prigSource) below(limit int64) int64 {
	for {
		if draw := s.next(); draw < limit {
			return draw
		}
	}
}

// next draws the next fifty-three-bit value.
func (s *prigSource) next() int64 {
	const width = 8 // bytes read per draw

	if s.at+width > len(s.block) {
		s.block = make([]byte, prigBufferBytes)
		// Since Go 1.24 crypto/rand.Read fills the whole slice and never
		// returns an error, an entropy failure being fatal instead.
		_, _ = rand.Read(s.block)
		s.at = 0
	}

	drawn := binary.BigEndian.Uint64(s.block[s.at:])
	s.at += width
	return int64(drawn >> (64 - prigDrawBits)) // #nosec G115 -- fifty-three bits fit an int64
}

func init() { core.Register(PseudoRandomIntegerGenerator{}) }
