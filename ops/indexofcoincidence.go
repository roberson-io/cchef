package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(IndexOfCoincidence{})
}

// IndexOfCoincidence measures how often two letters drawn from the text are the
// same. Ported from CyberChef's Index of Coincidence.
type IndexOfCoincidence struct{}

// Meta returns the operation metadata.
func (IndexOfCoincidence) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Index of Coincidence",
		Module:      "Default",
		Description: "Index of Coincidence (IC) is the probability of two randomly selected characters being the same.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (IndexOfCoincidence) Args() []core.ArgDef { return nil }

// The alphabet the coincidence is counted over, and what a coincidence of one
// letter in twenty-six comes to, which the normalised figure is measured in.
const (
	iocAlphabet   = "abcdefghijklmnopqrstuvwxyz"
	iocLeastPairs = 2
)

// Run measures the coincidence: how often two letters drawn from the text at
// random turn out to be the same.
func (IndexOfCoincidence) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	var counts [len(iocAlphabet)]int
	for _, r := range strings.ToLower(in.String()) {
		if r >= 'a' && r <= 'z' {
			counts[r-'a']++
		}
	}

	coincidence, letters := 0, 0
	for _, count := range counts {
		coincidence += count * (count - 1)
		letters += count
	}
	// Fewer than two letters leave no pair to draw, so the divisor is floored
	// rather than allowed to reach zero.
	density := float64(max(letters, iocLeastPairs))

	result := float64(coincidence) / (density * (density - 1))
	return core.NewDish([]byte(
		"Index of Coincidence: "+jsNumberString(result)+
			"\nNormalized: "+jsNumberString(result*float64(len(iocAlphabet))),
	), core.TypeString), nil
}
