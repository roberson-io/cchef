package ops

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(unitConverter{
		kind: "distance", name: "Convert distance",
		desc: "Converts a distance between units.", infoURL: "https://wikipedia.org/wiki/Orders_of_magnitude_(length)",
	})
	core.Register(unitConverter{
		kind: "mass", name: "Convert mass",
		desc: "Converts a mass between units.", infoURL: "https://wikipedia.org/wiki/Orders_of_magnitude_(mass)",
	})
	core.Register(unitConverter{
		kind: "speed", name: "Convert speed",
		desc: "Converts a speed between units.", infoURL: "https://wikipedia.org/wiki/Orders_of_magnitude_(speed)",
	})
	core.Register(unitConverter{
		kind: "area", name: "Convert area",
		desc: "Converts an area between units.", infoURL: "https://wikipedia.org/wiki/Orders_of_magnitude_(area)",
	})
	core.Register(unitConverter{
		kind: "data", name: "Convert data units",
		desc: "Converts a quantity of data between units (e.g. bits, bytes, kibibytes).", infoURL: "https://wikipedia.org/wiki/Orders_of_magnitude_(data)",
	})
}

// unitConverter converts a value between units of the same kind by scaling
// through a common base unit. The unit tables live in convert_data.go.
type unitConverter struct {
	kind, name, desc, infoURL string
}

// Meta returns the operation metadata.
func (c unitConverter) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        c.name,
		Module:      "Default",
		Description: c.desc,
		InfoURL:     c.infoURL,
		InputType:   core.TypeBigNumber,
		OutputType:  core.TypeBigNumber,
	}
}

// Args returns the argument definitions.
func (c unitConverter) Args() []core.ArgDef {
	units := unitTables[c.kind].units
	return []core.ArgDef{
		{Name: "Input units", Type: core.ArgOption, Value: units},
		{Name: "Output units", Type: core.ArgOption, Value: units},
	}
}

// Run performs the conversion. Ported from CyberChef Convert*.mjs:
// value * factor[in] / factor[out].
func (c unitConverter) Run(in *core.Dish, args []any) (*core.Dish, error) {
	table := unitTables[c.kind]
	value, ok := new(big.Rat).SetString(strings.TrimSpace(in.String()))
	if !ok {
		return nil, fmt.Errorf("input %q is not a valid number", in.String())
	}
	fin, _ := new(big.Rat).SetString(table.factors[args[0].(string)])
	fout, _ := new(big.Rat).SetString(table.factors[args[1].(string)])

	result := new(big.Rat).Mul(value, fin)
	result.Quo(result, fout)
	return core.NewDish([]byte(ratToDecimal(result)), core.TypeBigNumber), nil
}

// ratToDecimal formats a rational as a decimal string: exact when the value
// terminates, otherwise rounded to 20 decimal places. Trailing zeros (and a
// trailing point) are trimmed.
func ratToDecimal(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	s := r.FloatString(20)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}
