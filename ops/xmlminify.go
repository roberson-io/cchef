package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(XMLMinify{})
}

// XML minify regexes, ported verbatim from vkbeautify.xmlmin. The comment and
// xmlns patterns use the library's ASCII-only [ \r\n\t] class; the between-tags
// pattern (xmlTagWSRe, shared with XML Beautify) uses JS \s (jsWSChars).
var (
	xmlCommentRe = regexp.MustCompile(`<![ \r\n\t]*(--([^-]|[\r\n]|-[^-])*--[ \r\n\t]*)>`)
	xmlnsWSRe    = regexp.MustCompile(`[ \r\n\t]{1,}xmlns`)
)

// XMLMinify struct.
type XMLMinify struct{}

// Meta returns the operation metadata.
func (XMLMinify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XML Minify",
		Module:      "Code",
		Description: "Compresses eXtensible Markup Language (XML) code.",
		InfoURL:     "https://wikipedia.org/wiki/XML",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XMLMinify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Preserve comments", Type: core.ArgBoolean, Value: false},
	}
}

// Run minifies the XML input. Unless comments are preserved, HTML/XML comments are
// stripped and whitespace before xmlns is collapsed; then whitespace between tags
// is removed.
func (XMLMinify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	preserveComments := args[0].(bool)
	str := in.String()
	if !preserveComments {
		str = xmlCommentRe.ReplaceAllString(str, "")
		str = xmlnsWSRe.ReplaceAllString(str, " xmlns")
	}
	str = xmlTagWSRe.ReplaceAllString(str, "><")
	return core.NewDish([]byte(str), core.TypeString), nil
}
