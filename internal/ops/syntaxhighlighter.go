package ops

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SyntaxHighlighter{})
}

// SyntaxHighlighter adds syntax highlighting to a range of source-code
// languages. Ported from CyberChef SyntaxHighlighter.mjs, which highlights with
// highlight.js and emits HTML spans carrying hljs-* CSS classes (or auto-detects
// the language). cchef reimplements it over chroma
// (github.com/alecthomas/chroma), mapping chroma's token types onto the same
// hljs-* class vocabulary so the HTML output can be styled with a highlight.js
// theme.
//
// Reduced fidelity, by design: chroma is not highlight.js, so token boundaries
// and language auto-detection are not byte-identical. As a CLI-native addition
// beyond CyberChef (which only outputs HTML), the "Output format" option can also
// render the highlighting directly to the terminal with ANSI colours.
type SyntaxHighlighter struct{}

const (
	shAutoDetect = "auto detect"
	shHTML       = "HTML"
	shTerminal   = "Terminal"
	// shTermStyle is the chroma style used for terminal (ANSI) output.
	shTermStyle = "monokai"
	// shTermFormatter is chroma's 256-colour terminal formatter, the widest-
	// compatible ANSI target.
	shTermFormatter = "terminal256"
)

// Meta returns the operation metadata.
func (SyntaxHighlighter) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Syntax highlighter",
		Module:      "Code",
		Description: "Adds syntax highlighting to a range of source code languages.",
		InfoURL:     "https://wikipedia.org/wiki/Syntax_highlighting",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions. "Language" is a free-form language name
// (chroma lexer name or alias, matched case-insensitively) or "auto detect";
// "Output format" selects HTML (default, matching CyberChef) or Terminal ANSI.
func (SyntaxHighlighter) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Language", Type: core.ArgString, Value: shAutoDetect},
		{Name: "Output format", Type: core.ArgOption, Value: []string{shHTML, shTerminal}},
	}
}

// Run highlights the input in the requested language and output format.
func (SyntaxHighlighter) Run(in *core.Dish, args []any) (*core.Dish, error) {
	lang := strings.TrimSpace(args[0].(string))
	format := args[1].(string)
	src := in.String()

	lexer, err := shPickLexer(lang, src)
	if err != nil {
		return nil, err
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, src)
	if err != nil {
		return nil, err
	}
	tokens := it.Tokens()

	if format == shTerminal {
		var b strings.Builder
		f := formatters.Get(shTermFormatter)
		if err := f.Format(&b, styles.Get(shTermStyle), chroma.Literator(tokens...)); err != nil {
			return nil, err
		}
		return core.NewDish([]byte(b.String()), core.TypeString), nil
	}
	return core.NewDish([]byte(formatHLJS(tokens)), core.TypeString), nil
}

// shPickLexer resolves the language name to a chroma lexer. "auto detect" (or an
// empty language) analyses the source, falling back to the plaintext lexer; any
// other name is looked up by name/alias and errors if unknown.
func shPickLexer(lang, src string) (chroma.Lexer, error) {
	if lang == "" || strings.EqualFold(lang, shAutoDetect) {
		if l := lexers.Analyse(src); l != nil {
			return l, nil
		}
		return lexers.Fallback, nil
	}
	if l := lexers.Get(lang); l != nil {
		return l, nil
	}
	return nil, fmt.Errorf("unknown language %q", lang)
}

// shHTMLEscaper escapes the four characters highlight.js escapes in its HTML
// output, so cchef's spans match CyberChef's escaping.
var shHTMLEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// formatHLJS renders the token stream as bare HTML spans carrying hljs-* classes
// (no surrounding <pre>/wrapper), matching CyberChef's highlight.js output shape.
// Tokens with no hljs class are emitted as escaped text.
func formatHLJS(tokens []chroma.Token) string {
	var b strings.Builder
	for _, tk := range tokens {
		esc := shHTMLEscaper.Replace(tk.Value)
		if cls := hljsClass(tk.Type); cls != "" {
			b.WriteString(`<span class="`)
			b.WriteString(cls)
			b.WriteString(`">`)
			b.WriteString(esc)
			b.WriteString(`</span>`)
			continue
		}
		b.WriteString(esc)
	}
	return b.String()
}

// hljsClass maps a chroma token type onto a highlight.js CSS class. The lookup is
// most-specific-first: an exact token override, then the 100-level sub-category,
// then the 1000-level category; unmapped types (plain identifiers, punctuation,
// operators, whitespace) get no class and are emitted as text.
func hljsClass(t chroma.TokenType) string {
	if c, ok := hljsExact[t]; ok {
		return c
	}
	if c, ok := hljsBySub[t.SubCategory()]; ok {
		return c
	}
	return hljsByCat[t.Category()]
}

// hljsByCat maps a chroma token category (1000-level) to an hljs class.
var hljsByCat = map[chroma.TokenType]string{
	chroma.Keyword: "hljs-keyword",
	chroma.Literal: "hljs-literal",
	chroma.Comment: "hljs-comment",
}

// hljsBySub maps a chroma token sub-category (100-level) to an hljs class.
var hljsBySub = map[chroma.TokenType]string{
	chroma.NameBuiltin:    "hljs-built_in",
	chroma.NameVariable:   "hljs-variable",
	chroma.NameFunction:   "hljs-title",
	chroma.LiteralString:  "hljs-string",
	chroma.LiteralNumber:  "hljs-number",
	chroma.CommentPreproc: "hljs-meta",
}

// hljsExact maps individual chroma token types that need a class distinct from
// their sub-category/category default.
var hljsExact = map[chroma.TokenType]string{
	chroma.KeywordType:         "hljs-type",
	chroma.KeywordConstant:     "hljs-literal",
	chroma.OperatorWord:        "hljs-keyword",
	chroma.NameClass:           "hljs-title",
	chroma.NameNamespace:       "hljs-title",
	chroma.NameTag:             "hljs-name",
	chroma.NameAttribute:       "hljs-attr",
	chroma.NameDecorator:       "hljs-meta",
	chroma.NameConstant:        "hljs-variable",
	chroma.NameLabel:           "hljs-symbol",
	chroma.NameException:       "hljs-built_in",
	chroma.LiteralStringRegex:  "hljs-regexp",
	chroma.LiteralStringSymbol: "hljs-symbol",
	chroma.GenericHeading:      "hljs-section",
	chroma.GenericSubheading:   "hljs-section",
	chroma.GenericDeleted:      "hljs-deletion",
	chroma.GenericInserted:     "hljs-addition",
	chroma.GenericStrong:       "hljs-strong",
	chroma.GenericEmph:         "hljs-emphasis",
}
