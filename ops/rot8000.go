package ops

import (
	"strings"
	"sync"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ROT8000{})
}

// rot8000Once guards lazy construction of the character mapping.
var (
	rot8000Once sync.Once
	rot8000Map  map[rune]rune
)

// rot8000Table builds the char->char rotation map from the valid-code-point
// transitions used by rot8000 (valid-code-point-transitions.json). Each entry
// flips validity for all code points at or above its key; the ordered list of
// valid BMP code points is then rotated by half its length. Because that shift
// is exactly half, the mapping is its own inverse, so one operation both
// encrypts and decrypts.
func rot8000Table() map[rune]rune {
	rot8000Once.Do(func() {
		transitions := []struct {
			cp    int
			valid bool
		}{
			{33, true},
			{127, false},
			{161, true},
			{5760, false},
			{5761, true},
			{8192, false},
			{8203, true},
			{8232, false},
			{8234, true},
			{8239, false},
			{8240, true},
			{8287, false},
			{8288, true},
			{12288, false},
			{12289, true},
			{55296, false},
			{57344, true},
		}
		valid := make(map[int]bool, len(transitions))
		for _, t := range transitions {
			valid[t.cp] = t.valid
		}
		const bmpSize = 0x10000
		validInts := make([]rune, 0, 1<<15)
		curr := false
		for i := range bmpSize {
			if v, ok := valid[i]; ok {
				curr = v
			}
			if curr {
				validInts = append(validInts, rune(i))
			}
		}
		rotateNum := len(validInts) / 2
		rot8000Map = make(map[rune]rune, len(validInts))
		for i, r := range validInts {
			rot8000Map[r] = validInts[(i+rotateNum)%(rotateNum*2)]
		}
	})
	return rot8000Map
}

// ROT8000 is a Caesar cipher over the valid BMP code points, replacing each
// character with the one 0x8000 positions along the (compacted) alphabet.
type ROT8000 struct{}

// Meta returns the operation metadata.
func (ROT8000) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT8000",
		Module:      "Default",
		Description: "The simple Caesar-cypher encryption that replaces each Unicode character with the one 0x8000 places forward or back along the alphabet.",
		InfoURL:     "https://rot8000.com/info",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ROT8000) Args() []core.ArgDef { return nil }

// Run applies the rotation.
func (ROT8000) Run(in *core.Dish, args []any) (*core.Dish, error) {
	table := rot8000Table()
	var b strings.Builder
	for _, r := range dishText(in) {
		if to, ok := table[r]; ok {
			b.WriteRune(to)
		} else {
			b.WriteRune(r)
		}
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}
