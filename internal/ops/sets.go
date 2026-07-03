package ops

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SetUnion{})
	core.Register(SetIntersection{})
	core.Register(SetDifference{})
	core.Register(SymmetricDifference{})
	core.Register(CartesianProduct{})
	core.Register(PowerSet{})
}

// setDelimArgs are the shared Sample/Item delimiter arguments (binaryString).
func setDelimArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Sample delimiter", Type: core.ArgString, Value: "\\n\\n"},
		{Name: "Item delimiter", Type: core.ArgString, Value: ","},
	}
}

// errWrongSampleCount is CyberChef's OperationError message for an unexpected
// number of samples.
const errWrongSampleCount = "Incorrect number of sets, perhaps you need to modify the sample delimiter or add more samples?"

// splitSets parses the binaryString delimiters, splits the input into samples on
// the sample delimiter, validates the sample count, and splits each sample on the
// item delimiter. Mirrors the shared run() preamble of the two-argument set ops.
func splitSets(input, sampleDelimArg, itemDelimArg string, exactlyTwo bool) (sets [][]string, itemDelim string, err error) {
	sampleDelim := parseEscapedChars(sampleDelimArg)
	itemDelim = parseEscapedChars(itemDelimArg)
	samples := strings.Split(input, sampleDelim)
	if (exactlyTwo && len(samples) != 2) || (!exactlyTwo && len(samples) < 2) {
		return nil, itemDelim, fmt.Errorf("%s", errWrongSampleCount)
	}
	sets = make([][]string, len(samples))
	for i, s := range samples {
		sets[i] = strings.Split(s, itemDelim)
	}
	return sets, itemDelim, nil
}

// jsObjectKeys reproduces the enumeration order of a JavaScript object used as a
// hash set: canonical array-index keys (non-negative integers below 2^32-1) come
// first in ascending numeric order, then the remaining keys in insertion order.
func jsObjectKeys(order []string) []string {
	type idxKey struct {
		v uint64
		s string
	}
	var ints []idxKey
	var strs []string
	for _, k := range order {
		if v, ok := arrayIndex(k); ok {
			ints = append(ints, idxKey{v, k})
		} else {
			strs = append(strs, k)
		}
	}
	sort.SliceStable(ints, func(i, j int) bool { return ints[i].v < ints[j].v })
	out := make([]string, 0, len(order))
	for _, k := range ints {
		out = append(out, k.s)
	}
	return append(out, strs...)
}

// arrayIndex reports whether s is a canonical JavaScript array index (a
// non-negative integer, no leading zeros, less than 2^32-1).
func arrayIndex(s string) (uint64, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil || v >= 4294967295 {
		return 0, false
	}
	return v, true
}

// SetUnion calculates the union of two sets.
type SetUnion struct{}

// Meta returns the operation metadata.
func (SetUnion) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Set Union",
		Module:      "Default",
		Description: "Calculates the union of two sets.",
		InfoURL:     "https://wikipedia.org/wiki/Union_(set_theory)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SetUnion) Args() []core.ArgDef { return setDelimArgs() }

// Run computes the union. Ported from CyberChef SetUnion.mjs (uses a JS object
// as the hash set, so its key-ordering quirk is reproduced).
func (SetUnion) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sets, itemDelim, err := splitSets(in.String(), args[0].(string), args[1].(string), true)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var order []string
	for _, set := range sets {
		for _, item := range set {
			if !seen[item] {
				seen[item] = true
				order = append(order, item)
			}
		}
	}
	return core.NewDish([]byte(strings.Join(jsObjectKeys(order), itemDelim)), core.TypeString), nil
}

// SetIntersection calculates the intersection of two sets.
type SetIntersection struct{}

// Meta returns the operation metadata.
func (SetIntersection) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Set Intersection",
		Module:      "Default",
		Description: "Calculates the intersection of two sets.",
		InfoURL:     "https://wikipedia.org/wiki/Intersection_(set_theory)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SetIntersection) Args() []core.ArgDef { return setDelimArgs() }

// Run computes the intersection: items of the first set that are also in the
// second, deduplicated (CyberChef PR #2286). Ported from SetIntersection.mjs.
func (SetIntersection) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sets, itemDelim, err := splitSets(in.String(), args[0].(string), args[1].(string), true)
	if err != nil {
		return nil, err
	}
	included := sliceToSet(sets[1])
	seen := map[string]bool{}
	var out []string
	for _, item := range sets[0] {
		if included[item] && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return core.NewDish([]byte(strings.Join(out, itemDelim)), core.TypeString), nil
}

// SetDifference calculates the relative complement of two sets.
type SetDifference struct{}

// Meta returns the operation metadata.
func (SetDifference) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Set Difference",
		Module:      "Default",
		Description: "Calculates the difference, or relative complement, of two sets.",
		InfoURL:     "https://wikipedia.org/wiki/Complement_(set_theory)#Relative_complement",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SetDifference) Args() []core.ArgDef { return setDelimArgs() }

// Run computes the relative complement: items of the first set not in the second,
// deduplicated (CyberChef PR #2286). Ported from SetDifference.mjs.
func (SetDifference) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sets, itemDelim, err := splitSets(in.String(), args[0].(string), args[1].(string), true)
	if err != nil {
		return nil, err
	}
	excluded := sliceToSet(sets[1])
	seen := map[string]bool{}
	var out []string
	for _, item := range sets[0] {
		if !excluded[item] && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return core.NewDish([]byte(strings.Join(out, itemDelim)), core.TypeString), nil
}

// SymmetricDifference calculates the symmetric difference of two sets.
type SymmetricDifference struct{}

// Meta returns the operation metadata.
func (SymmetricDifference) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Symmetric Difference",
		Module:      "Default",
		Description: "Calculates the symmetric difference of two sets.",
		InfoURL:     "https://wikipedia.org/wiki/Symmetric_difference",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SymmetricDifference) Args() []core.ArgDef { return setDelimArgs() }

// Run computes the symmetric difference (items in exactly one set). It preserves
// duplicates within each side. Ported from SymmetricDifference.mjs.
func (SymmetricDifference) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sets, itemDelim, err := splitSets(in.String(), args[0].(string), args[1].(string), true)
	if err != nil {
		return nil, err
	}
	out := append(itemsNotIn(sets[0], sets[1]), itemsNotIn(sets[1], sets[0])...)
	return core.NewDish([]byte(strings.Join(out, itemDelim)), core.TypeString), nil
}

// CartesianProduct calculates the cartesian product of multiple sets.
type CartesianProduct struct{}

// Meta returns the operation metadata.
func (CartesianProduct) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Cartesian Product",
		Module:      "Default",
		Description: "Calculates the cartesian product of multiple sets of data, returning all possible combinations.",
		InfoURL:     "https://wikipedia.org/wiki/Cartesian_product",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CartesianProduct) Args() []core.ArgDef { return setDelimArgs() }

// Run computes the cartesian product of two or more sets, formatting each
// combination as "(a,b,...)". Ported from CartesianProduct.mjs.
func (CartesianProduct) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sets, itemDelim, err := splitSets(in.String(), args[0].(string), args[1].(string), false)
	if err != nil {
		return nil, err
	}
	product := [][]string{{}}
	for _, set := range sets {
		var next [][]string
		for _, combo := range product {
			for _, item := range set {
				next = append(next, append(append([]string{}, combo...), item))
			}
		}
		product = next
	}
	formatted := make([]string, len(product))
	for i, combo := range product {
		formatted[i] = "(" + strings.Join(combo, ",") + ")"
	}
	return core.NewDish([]byte(strings.Join(formatted, itemDelim)), core.TypeString), nil
}

// PowerSet calculates all the subsets of a set.
type PowerSet struct{}

// Meta returns the operation metadata.
func (PowerSet) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Power Set",
		Module:      "Default",
		Description: "Calculates all the subsets of a set.",
		InfoURL:     "https://wikipedia.org/wiki/Power_set",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PowerSet) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Item delimiter", Type: core.ArgString, Value: ","}}
}

// Run computes the power set of the input. Each subset is joined by the item
// delimiter, subsets are ordered by their joined-string length, and each is
// followed by a newline. Ported from PowerSet.mjs.
func (PowerSet) Run(in *core.Dish, args []any) (*core.Dish, error) {
	itemDelim := parseEscapedChars(args[0].(string))
	var items []string
	for _, item := range strings.Split(in.String(), itemDelim) {
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}
	subsets := make([]string, 0, 1<<len(items))
	for mask := 0; mask < (1 << len(items)); mask++ {
		var subset []string
		for i, item := range items {
			// Bit i-from-the-left selects item i (mask's most significant bit first).
			if mask&(1<<(len(items)-1-i)) != 0 {
				subset = append(subset, item)
			}
		}
		subsets = append(subsets, strings.Join(subset, itemDelim))
	}
	sort.SliceStable(subsets, func(i, j int) bool { return len(subsets[i]) < len(subsets[j]) })
	var b strings.Builder
	for _, sub := range subsets {
		b.WriteString(sub)
		b.WriteByte('\n')
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// sliceToSet builds a set of the slice's elements.
func sliceToSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

// itemsNotIn returns the elements of a that do not appear in b, preserving a's
// order and duplicates.
func itemsNotIn(a, b []string) []string {
	exclude := sliceToSet(b)
	out := make([]string, 0, len(a))
	for _, item := range a {
		if !exclude[item] {
			out = append(out, item)
		}
	}
	return out
}
