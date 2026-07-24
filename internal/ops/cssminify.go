package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CSSMinify{})
}

// CSS minify regexes, ported verbatim from vkbeautify.cssmin. The comment pattern
// is ASCII; the whitespace-tightening patterns use JS \s (jsWSChars).
var (
	cssCommentRe      = regexp.MustCompile(`/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/`)
	cssBraceOpenWSRe  = regexp.MustCompile(`\{[` + jsWSChars + `]+`)
	cssBraceCloseWSRe = regexp.MustCompile(`\}[` + jsWSChars + `]+`)
	cssSemiWSRe       = regexp.MustCompile(`;[` + jsWSChars + `]+`)
	cssCommentOpenWS  = regexp.MustCompile(`/\*[` + jsWSChars + `]+`)
	cssCommentCloseWS = regexp.MustCompile(`\*/[` + jsWSChars + `]+`)
)

// CSSMinify struct.
type CSSMinify struct{}

// Meta returns the operation metadata.
func (CSSMinify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CSS Minify",
		Module:      "Code",
		Description: "Compresses Cascading Style Sheets (CSS) code.",
		InfoURL:     "https://wikipedia.org/wiki/CSS",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CSSMinify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Preserve comments", Type: core.ArgBoolean, Value: false},
	}
}

// Run minifies the CSS input. Unless comments are preserved they are stripped
// first; then whitespace runs are collapsed to a single space and whitespace after
// {, }, ;, /* and */ is removed.
func (CSSMinify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	preserveComments := args[0].(bool)
	str := in.String()
	if !preserveComments {
		str = cssCommentRe.ReplaceAllString(str, "")
	}
	str = jsWSRun.ReplaceAllString(str, " ")
	str = cssBraceOpenWSRe.ReplaceAllString(str, "{")
	str = cssBraceCloseWSRe.ReplaceAllString(str, "}")
	str = cssSemiWSRe.ReplaceAllString(str, ";")
	str = cssCommentOpenWS.ReplaceAllString(str, "/*")
	str = cssCommentCloseWS.ReplaceAllString(str, "*/")
	return core.NewDish([]byte(str), core.TypeString), nil
}
