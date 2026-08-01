package ops

import "github.com/roberson-io/cchef/core"

// HTMLToText hands markup on as plain text.
//
// In CyberChef this changes how a result is displayed rather than what it
// holds: the interface renders a value it has been told is HTML, and this
// operation passes the same value on as text so the markup is shown as it
// stands. Nothing renders anything at a command line, so here the input comes
// straight back.
type HTMLToText struct{}

// Meta returns the operation metadata.
func (HTMLToText) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "HTML To Text",
		Module: "Default",
		Description: "Displays HTML as raw text rather than rendered markup." +
			"<br><br>At a command line nothing is rendered in the first place, so " +
			"the input is passed through unchanged.",
		InfoURL:    "https://wikipedia.org/wiki/HTML",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (HTMLToText) Args() []core.ArgDef { return nil }

// Run hands the input back.
func (HTMLToText) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish(in.Bytes(), core.TypeString), nil
}

func init() { core.Register(HTMLToText{}) }
