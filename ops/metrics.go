package ops

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(LevenshteinDistance{})
	core.Register(Wrap{})
	core.Register(HammingDistance{})
}

// LevenshteinDistance computes the weighted edit distance between two samples.
type LevenshteinDistance struct{}

// Meta returns the operation metadata.
func (LevenshteinDistance) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Levenshtein Distance",
		Module:      "Default",
		Description: "Computes the Levenshtein (edit) distance between two samples, with configurable insertion, deletion and substitution costs.",
		InfoURL:     "https://wikipedia.org/wiki/Levenshtein_distance",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (LevenshteinDistance) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Sample delimiter", Type: core.ArgString, Value: `\n`},
		{Name: "Insertion cost", Type: core.ArgNumber, Value: 1},
		{Name: "Deletion cost", Type: core.ArgNumber, Value: 1},
		{Name: "Substitution cost", Type: core.ArgNumber, Value: 1},
	}
}

// Run computes the distance.
func (LevenshteinDistance) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := opsutil.ParseEscapedChars(args[0].(string))
	insCost := int(args[1].(float64))
	delCost := int(args[2].(float64))
	subCost := int(args[3].(float64))

	samples := strings.Split(in.String(), delim)
	if len(samples) != 2 {
		return nil, fmt.Errorf("incorrect number of samples; check your input and/or delimiter")
	}
	if insCost < 0 || delCost < 0 || subCost < 0 {
		return nil, fmt.Errorf("negative costs are not allowed")
	}

	src := []rune(samples[0])
	dest := []rune(samples[1])
	current := make([]int, len(src)+1)
	next := make([]int, len(src)+1)
	for i := range current {
		current[i] = delCost * i
	}
	for i := range dest {
		next[0] = current[0] + insCost
		for j := range src {
			opt := current[j+1] + insCost // insertion
			if c := next[j] + delCost; c < opt {
				opt = c // deletion
			}
			c := current[j] // substitution / match
			if src[j] != dest[i] {
				c += subCost
			}
			if c < opt {
				opt = c
			}
			next[j+1] = opt
		}
		current, next = next, current
	}
	return core.NewDish([]byte(strconv.Itoa(current[len(src)])), core.TypeNumber), nil
}

// Wrap breaks the input into fixed-width lines.
type Wrap struct{}

// Meta returns the operation metadata.
func (Wrap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Wrap",
		Module:      "Default",
		Description: "Wraps text to a specified line width, breaking it into lines of at most that many characters.",
		InfoURL:     "https://wikipedia.org/wiki/Line_wrap_and_word_wrap",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Wrap) Args() []core.ArgDef {
	minWidth, maxWidth := 1.0, float64(maxWrapWidth)
	return []core.ArgDef{
		{Name: "Line Width", Type: core.ArgNumber, Integer: true, Value: 64, Min: &minWidth, Max: &maxWidth},
	}
}

// maxWrapWidth caps the Wrap line width (matches CyberChef's MAX_LINE_WIDTH,
// added in gchq/CyberChef#2606).
const maxWrapWidth = 65536

// Run wraps the input. Ported from CyberChef Wrap.mjs, which matches
// /.{1,width}/g and joins with newlines — "." excludes line terminators, so
// existing line breaks split runs and empty runs are dropped. Reimplemented with
// rune chunking rather than a regexp, since Go's regexp caps repeat counts at
// 1000 (below the 65536 max width) and would otherwise panic.
func (Wrap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish(nil, core.TypeString), nil
	}
	// The ArgDef bounds the width to a whole number in [1, maxWrapWidth].
	w := int(args[0].(float64))

	var pieces []string
	for _, line := range strings.FieldsFunc(input, isLineTerminator) {
		runes := []rune(line)
		for i := 0; i < len(runes); i += w {
			pieces = append(pieces, string(runes[i:min(i+w, len(runes))]))
		}
	}
	return core.NewDish([]byte(strings.Join(pieces, "\n")), core.TypeString), nil
}

// isLineTerminator reports whether r is one of the characters JavaScript's "."
// does not match (used to split runs before wrapping).
func isLineTerminator(r rune) bool {
	return r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029'
}

// HammingDistance computes the Hamming distance between two equal-length samples.
type HammingDistance struct{}

// Meta returns the operation metadata.
func (HammingDistance) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Hamming Distance",
		Module:      "Default",
		Description: "Computes the Hamming distance between two equal-length samples, by byte or by bit.",
		InfoURL:     "https://wikipedia.org/wiki/Hamming_distance",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HammingDistance) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgString, Value: `\n\n`},
		{Name: "Unit", Type: core.ArgOption, Value: []string{"Byte", "Bit"}},
		{Name: "Input type", Type: core.ArgOption, Value: []string{"Raw string", "Hex"}},
	}
}

// Run computes the distance.
func (HammingDistance) Run(in *core.Dish, args []any) (*core.Dish, error) {
	delim := opsutil.ParseEscapedChars(args[0].(string))
	byByte := args[1].(string) == "Byte"
	samples := strings.Split(in.String(), delim)
	if len(samples) != 2 {
		return nil, fmt.Errorf("you can only calculate the distance between 2 strings; provide exactly two inputs separated by the delimiter")
	}

	var a, b []byte
	if args[2].(string) == "Hex" {
		a, b = splitHexToBytes(samples[0]), splitHexToBytes(samples[1])
	} else {
		a, b = []byte(samples[0]), []byte(samples[1])
	}
	if len(a) != len(b) {
		return nil, fmt.Errorf("both inputs must be of the same length")
	}

	dist := 0
	for i := range a {
		if byByte {
			if a[i] != b[i] {
				dist++
			}
		} else {
			dist += bits.OnesCount8(a[i] ^ b[i])
		}
	}
	return core.NewDish([]byte(strconv.Itoa(dist)), core.TypeString), nil
}
