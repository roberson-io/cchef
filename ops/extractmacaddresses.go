package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The shape of a MAC address: six pairs of hexadecimal digits, separated
// throughout by colons or by hyphens.
const macPattern = `[A-F\d]{2}(?:[:-][A-F\d]{2}){5}`

var macRegexp = jsRegex(macPattern, regexp2.IgnoreCase)

// ExtractMACAddresses pulls the MAC addresses out of the input.
type ExtractMACAddresses struct{}

// Meta returns the operation metadata.
func (ExtractMACAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract MAC addresses",
		Module:      "Regex",
		Description: "Extracts all Media Access Control (MAC) addresses from the input.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractMACAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the addresses.
func (ExtractMACAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	displayTotal, _ := args[0].(bool)
	sortResults, _ := args[1].(bool)
	unique, _ := args[2].(bool)

	// The ordering reads each pair as the number it stands for rather than as
	// text, so that 0a: comes before ff: and an address written with hyphens
	// falls next to the same address written with colons.
	var less func(a, b string) bool
	if sortResults {
		less = func(a, b string) bool { return naturalCompare(a, b, true) < 0 }
	}

	found := extractSearch(in.String(), macRegexp, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

func init() { core.Register(ExtractMACAddresses{}) }
