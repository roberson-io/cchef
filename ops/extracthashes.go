package ops

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// bitsPerHexDigit is how much of a hash one hexadecimal character carries.
const bitsPerHexDigit = 4

// hashAllBitLengths are the digest sizes, in bits, that "All hashes" looks for,
// shortest first — which is also the order the results come out in.
var hashAllBitLengths = []int{4, 8, 16, 32, 64, 128, 160, 192, 224, 256, 320, 384, 512, 1024}

// ExtractHashes pulls runs of hexadecimal the length of a hash out of the input.
type ExtractHashes struct{}

// Meta returns the operation metadata.
func (ExtractHashes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract hashes",
		Module:      "Regex",
		Description: "Extracts potential hashes based on hash character length",
		InfoURL:     "https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractHashes) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Hash character length", Type: core.ArgNumber, Value: 40},
		{Name: "All hashes", Type: core.ArgBoolean, Value: false},
		{Name: "Display Total", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the runs of hexadecimal.
func (ExtractHashes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length, _ := args[0].(float64)
	all, _ := args[1].(bool)
	displayTotal, _ := args[2].(bool)

	input := in.String()
	var found []string
	for _, n := range hashLengths(length, all, len(input)) {
		found = append(found, extractSearch(input, hashRegexp(n), nil, nil, false)...)
	}

	var out strings.Builder
	if displayTotal {
		fmt.Fprintf(&out, "Total Results: %d\n\n", len(found))
	}
	out.WriteString(strings.Join(found, "\n"))
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// hashLengths returns the run lengths to look for, in characters. "All hashes"
// walks every size in common use and pays no attention to the length asked for.
// Otherwise that one length is used — unless it describes nothing findable: a
// length below one is not a hash at all, a fractional one is not a count of
// characters, and a run longer than the input cannot be in it.
func hashLengths(length float64, all bool, inputLen int) []int {
	if all {
		lengths := make([]int, len(hashAllBitLengths))
		for i, bits := range hashAllBitLengths {
			lengths[i] = bits / bitsPerHexDigit
		}
		return lengths
	}
	if length < 1 || length > float64(inputLen) || length != math.Trunc(length) {
		return nil
	}
	return []int{int(length)}
}

// hashRegexp matches a run of exactly n lowercase hexadecimal characters. The
// boundaries keep part of a longer run from being reported as a shorter hash.
func hashRegexp(n int) *regexp2.Regexp {
	return jsRegex(jsWordBefore+`[a-f0-9]{`+strconv.Itoa(n)+`}`+jsWordAfter, regexp2.None)
}

func init() { core.Register(ExtractHashes{}) }
