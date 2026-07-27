package ops

import "github.com/roberson-io/cchef/internal/core"

// xkcdRandomNumber is the number, chosen by a fair dice roll and guaranteed to
// be random. See https://xkcd.com/221/.
const xkcdRandomNumber = "4"

// XKCDRandomNumber gives a random number.
type XKCDRandomNumber struct{}

// Meta returns the operation metadata.
func (XKCDRandomNumber) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "XKCD Random Number",
		Module: "Default",
		Description: "RFC 1149.5 specifies 4 as the standard IEEE-vetted random " +
			"number.",
		InfoURL:    "https://xkcd.com/221/",
		InputType:  core.TypeString,
		OutputType: core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (XKCDRandomNumber) Args() []core.ArgDef { return nil }

// Run gives the number.
func (XKCDRandomNumber) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish([]byte(xkcdRandomNumber), core.TypeNumber), nil
}

func init() { core.Register(XKCDRandomNumber{}) }
