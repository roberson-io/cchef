package ops

import (
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JavaScriptParser{})
}

// JavaScriptParser returns the ESTree AST (as pretty JSON) for JavaScript code.
// Ported from CyberChef JavaScriptParser.mjs, which wraps esprima.parseScript;
// the parser is a from-scratch transliteration of esprima (see jsparser_*.go).
// The full script grammar is supported byte-for-byte — including classes,
// async/await, generators/yield, destructuring, and Unicode identifiers. Only the
// output-shaping options (loc/range/tokens/comment/tolerant) are not yet ported
// and raise an error rather than diverging. (import/export are syntax errors in
// script mode, matching esprima.)
type JavaScriptParser struct{}

// Meta returns the operation metadata.
func (JavaScriptParser) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JavaScript Parser",
		Module:      "Code",
		Description: "Returns an Abstract Syntax Tree for valid JavaScript code.",
		InfoURL:     "https://wikipedia.org/wiki/Abstract_syntax_tree",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JavaScriptParser) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Location info", Type: core.ArgBoolean, Value: false},
		{Name: "Range info", Type: core.ArgBoolean, Value: false},
		{Name: "Include tokens array", Type: core.ArgBoolean, Value: false},
		{Name: "Include comments array", Type: core.ArgBoolean, Value: false},
		{Name: "Report errors and try to continue", Type: core.ArgBoolean, Value: false},
	}
}

// Run parses the input and returns the AST as JSON.stringify(ast, null, 2).
func (JavaScriptParser) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// The output-shaping options (loc/range/tokens/comment/tolerant) are not yet
	// ported; reject them rather than silently ignoring.
	for i, name := range []string{"Location info", "Range info", "Include tokens array", "Include comments array", "Report errors and try to continue"} {
		if args[i].(bool) {
			return nil, errors.New("the " + name + " option is not yet supported by this JavaScript Parser port")
		}
	}
	ast, err := jsParse(in.String())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(jsStringify(ast, 2)), core.TypeString), nil
}
