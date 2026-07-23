package ops

import (
	"errors"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JavaScriptBeautify{})
}

// JavaScriptBeautify parses JavaScript (or JSON) and pretty-prints it. Ported
// from CyberChef JavaScriptBeautify.mjs, which parses with esprima.parseScript
// and regenerates with escodegen; jsParse/jsParseFull and jsGenerate
// (jsbeautify_*.go) are from-scratch transliterations of those. With "Include
// comments" enabled, comments are collected with source ranges, attached to the
// AST (a port of estraverse.attachComments), and re-emitted.
type JavaScriptBeautify struct{}

// Meta returns the operation metadata.
func (JavaScriptBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JavaScript Beautify",
		Module:      "Code",
		Description: "Parses and pretty prints valid JavaScript code. Also works with JavaScript Object Notation (JSON).",
		InfoURL:     "https://github.com/estools/escodegen",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JavaScriptBeautify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Indent string", Type: core.ArgString, Value: "\t"},
		{Name: "Quotes", Type: core.ArgOption, Value: []string{"Auto", "Single", "Double"}},
		{Name: "Semicolons before closing braces", Type: core.ArgBoolean, Value: true},
		{Name: "Include comments", Type: core.ArgBoolean, Value: true},
	}
}

// Run parses the input and regenerates it in beautified form.
func (JavaScriptBeautify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	indent := jsUnescapeIndent(args[0].(string))
	if indent == "" {
		indent = "\t"
	}
	quotes := strings.ToLower(args[1].(string))
	semicolons := args[2].(bool)
	comment := args[3].(bool)

	if comment {
		ast, comments, tokens, err := jsParseFull(in.String())
		if err != nil {
			return nil, errors.New("Unable to parse JavaScript.\n" + err.Error())
		}
		cs := jsAttachComments(ast, comments, tokens)
		out := jsGenerateComments(ast, indent, quotes, semicolons, cs)
		return core.NewDish([]byte(out), core.TypeString), nil
	}

	ast, err := jsParse(in.String())
	if err != nil {
		return nil, errors.New("Unable to parse JavaScript.\n" + err.Error())
	}
	out := jsGenerate(ast, indent, quotes, semicolons)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// jsUnescapeIndent interprets the common backslash escapes CyberChef's
// binaryShortString accepts in the indent argument (so a literal "\t" from the
// CLI becomes a tab).
func jsUnescapeIndent(s string) string {
	if !strings.Contains(s, "\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 't':
				b.WriteByte('\t')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 'f':
				b.WriteByte('\f')
			case 'v':
				b.WriteByte('\v')
			case '0':
				b.WriteByte(0)
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
