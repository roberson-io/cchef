package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ConvertLeetSpeak{})
}

// ConvertLeetSpeak converts text to and from leet speak. Ported from CyberChef
// ConvertLeetSpeak.mjs.
type ConvertLeetSpeak struct{}

// Meta returns the operation metadata.
func (ConvertLeetSpeak) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Convert Leet Speak",
		Module:      "Default",
		Description: "Converts to and from Leet Speak.",
		InfoURL:     "https://wikipedia.org/wiki/Leet",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the direction.
func (ConvertLeetSpeak) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Direction", Type: core.ArgOption, Value: []string{"To Leet Speak", "From Leet Speak"}},
	}
}

// Run converts the input. Upstream's tables map every letter, but only six
// pairs differ from identity: a/e/i/o/s/t against 4/3/1/0/5/7. Going to leet
// either case of the letter becomes the digit; coming back the digit becomes
// the lowercase letter (leet digits carry no case to restore), and everything
// else — other digits, punctuation, anything beyond ASCII — passes through.
func (ConvertLeetSpeak) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var mapping func(rune) rune
	if args[0].(string) == "To Leet Speak" {
		mapping = func(r rune) rune {
			switch r {
			case 'a', 'A':
				return '4'
			case 'e', 'E':
				return '3'
			case 'i', 'I':
				return '1'
			case 'o', 'O':
				return '0'
			case 's', 'S':
				return '5'
			case 't', 'T':
				return '7'
			}
			return r
		}
	} else {
		mapping = func(r rune) rune {
			switch r {
			case '4':
				return 'a'
			case '3':
				return 'e'
			case '1':
				return 'i'
			case '0':
				return 'o'
			case '5':
				return 's'
			case '7':
				return 't'
			}
			return r
		}
	}
	return core.NewDish([]byte(strings.Map(mapping, in.String())), core.TypeString), nil
}
