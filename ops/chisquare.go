package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(ChiSquare{})
}

// ChiSquare measures how far the byte distribution strays from a flat one.
// Ported from CyberChef's Chi Square.
type ChiSquare struct{}

// Meta returns the operation metadata.
func (ChiSquare) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Chi Square",
		Module:      "Default",
		Description: "Calculates the Chi Square distribution of values.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (ChiSquare) Args() []core.ArgDef { return nil }

// Run measures the distribution. Each byte value contributes the square of how
// far its count strays from an even share, over that share; values that do not
// occur contribute nothing.
func (ChiSquare) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	expected := float64(len(data)) / 256

	total := 0.0
	for _, count := range byteCounts(data) {
		if count > 0 {
			difference := float64(count) - expected
			total += difference * difference / expected
		}
	}
	return core.NewDish([]byte(jsnum.String(total)), core.TypeNumber), nil
}
