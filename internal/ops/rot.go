package ops

import "github.com/roberson-io/cchef/internal/core"

func init() {
	core.Register(ROT13{})
	core.Register(ROT47{})
}

// ROT13 is a Caesar substitution cipher rotating alphabet characters (and
// optionally digits) by a configurable amount.
type ROT13 struct{}

// Meta returns the operation metadata.
func (ROT13) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT13",
		Module:      "Default",
		Description: "A simple Caesar substitution cipher which rotates alphabet characters by the specified amount (default 13).",
		InfoURL:     "https://wikipedia.org/wiki/ROT13",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ROT13) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Rotate lower case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate upper case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate numbers", Type: core.ArgBoolean, Value: false},
		{Name: "Amount", Type: core.ArgNumber, Value: 13},
	}
}

// Run applies the rotation. Ported from CyberChef ROT13.mjs.
func (ROT13) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rotLower := args[0].(bool)
	rotUpper := args[1].(bool)
	rotNumbers := args[2].(bool)
	amount := int(args[3].(float64))
	amountNumbers := amount

	out := append([]byte(nil), in.Bytes()...)
	if amount == 0 {
		return core.NewDish(out, core.TypeByteArray), nil
	}
	if amount < 0 {
		amount = 26 - (abs(amount) % 26)
		amountNumbers = 10 - (abs(amountNumbers) % 10)
	}

	for i, chr := range out {
		switch {
		case rotUpper && chr >= 'A' && chr <= 'Z':
			out[i] = byte('A' + (int(chr)-'A'+amount)%26)
		case rotLower && chr >= 'a' && chr <= 'z':
			out[i] = byte('a' + (int(chr)-'a'+amount)%26)
		case rotNumbers && chr >= '0' && chr <= '9':
			out[i] = byte('0' + (int(chr)-'0'+amountNumbers)%10)
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// ROT47 rotates printable ASCII characters from 33 ('!') to 126 ('~').
type ROT47 struct{}

// Meta returns the operation metadata.
func (ROT47) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT47",
		Module:      "Default",
		Description: "A variant of the Caesar cipher covering ASCII characters 33 ('!') to 126 ('~'). Default rotation: 47.",
		InfoURL:     "https://wikipedia.org/wiki/ROT13#Variants",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ROT47) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Value: 47},
	}
}

// Run applies the rotation. Ported from CyberChef ROT47.mjs.
func (ROT47) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := int(args[0].(float64))
	out := append([]byte(nil), in.Bytes()...)
	if amount == 0 {
		return core.NewDish(out, core.TypeByteArray), nil
	}
	if amount < 0 {
		amount = 94 - (abs(amount) % 94)
	}
	for i, chr := range out {
		if chr >= 33 && chr <= 126 {
			out[i] = byte(33 + (int(chr)-33+amount)%94)
		}
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
