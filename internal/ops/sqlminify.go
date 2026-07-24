package ops

import (
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SQLMinify{})
}

// SQLMinify struct.
type SQLMinify struct{}

// Meta returns the operation metadata.
func (SQLMinify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SQL Minify",
		Module:      "Code",
		Description: "Compresses Structured Query Language (SQL) code.",
		InfoURL:     "https://wikipedia.org/wiki/SQL",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SQLMinify) Args() []core.ArgDef { return nil }

// Run minifies the SQL input. Ported from vkbeautify.sqlmin: collapse every
// whitespace run to a single space, then drop the space before the first "(" and
// the first ")" (the library's replaces for those two are not global).
func (SQLMinify) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	str := jsWSRun.ReplaceAllString(in.String(), " ")
	str = strings.Replace(str, " (", "(", 1)
	str = strings.Replace(str, " )", ")", 1)
	return core.NewDish([]byte(str), core.TypeString), nil
}
