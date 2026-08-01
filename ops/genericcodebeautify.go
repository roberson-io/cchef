package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(GenericCodeBeautify{})
}

// GenericCodeBeautify struct.
type GenericCodeBeautify struct{}

// Meta returns the operation metadata.
func (GenericCodeBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generic Code Beautify",
		Module:      "Code",
		Description: "Attempts to pretty print C-style languages such as C, Java, PHP, JavaScript etc.",
		InfoURL:     "https://wikipedia.org/wiki/Prettyprint",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenericCodeBeautify) Args() []core.ArgDef { return nil }

// Token-preservation regexes (strings, comments, regex literals). Applied in the
// same order as GenericCodeBeautify.mjs.
var gcbPreserveRes = []*regexp.Regexp{
	regexp.MustCompile(`'([^'\\]|\\.)*'`),          // single-quoted strings
	regexp.MustCompile(`"([^"\\]|\\.)*"`),          // double-quoted strings
	regexp.MustCompile(`//[^\n\r]*`),               // line comments
	regexp.MustCompile(`/\*[\s\S]*?\*/`),           // block comments
	regexp.MustCompile(`(^|\n)#[^\n\r#]+`),         // hash comments
	regexp.MustCompile(`(?i)/.*?[^\\]/[gim]{0,3}`), // regex literals
}

var gcbPreservedToken = regexp.MustCompile(`###preservedToken(\d+)###`)

// gcbReplace pairs a regex with its replacement for the formatting chains.
type gcbReplace struct {
	re   *regexp.Regexp
	with string
}

var gcbPhase1 = []gcbReplace{
	{regexp.MustCompile(`;`), ";\n"},
	{regexp.MustCompile(`{`), "{\n"},
	{regexp.MustCompile(`}`), "\n}\n"},
	{regexp.MustCompile(`\r`), ""},
	{regexp.MustCompile(`^\s+`), ""},
	{regexp.MustCompile(`\n\s+`), "\n"},
	{regexp.MustCompile(`\s*$`), ""},
	{regexp.MustCompile(`\n{`), "{"},
}

var gcbPhase2 = []gcbReplace{
	{regexp.MustCompile(`\s*([!<>=+-/*]?)=\s*`), " ${1}= "},
	{regexp.MustCompile(`\s*<([=]?)\s*`), " <${1} "},
	{regexp.MustCompile(`\s*>([=]?)\s*`), " >${1} "},
	{regexp.MustCompile(`([^+])\+([^+=])`), "${1} + ${2}"},
	{regexp.MustCompile(`([^-])-([^-=])`), "${1} - ${2}"},
	{regexp.MustCompile(`([^*])\*([^*=])`), "${1} * ${2}"},
	{regexp.MustCompile(`([^/])/([^/=])`), "${1} / ${2}"},
	{regexp.MustCompile(`\s*,\s*`), ", "},
	{regexp.MustCompile(`\s*{`), " {"},
	{regexp.MustCompile(`}\n`), "}\n\n"},
	{regexp.MustCompile(`(?im)(if|for|while|with|elif|elseif)\s*\(([^\n]*)\)\s*\n([^{])`), "${1} (${2})\n    ${3}"},
	{regexp.MustCompile(`(?im)(if|for|while|with|elif|elseif)\s*\(([^\n]*)\)([^{])`), "${1} (${2}) ${3}"},
	{regexp.MustCompile(`(?im)else\s*\n([^{])`), "else\n    ${1}"},
	{regexp.MustCompile(`(?im)else\s+([^{])`), "else ${1}"},
	{regexp.MustCompile(`\s+;`), ";"},
	{regexp.MustCompile(`\{\s+\}`), "{}"},
	{regexp.MustCompile(`\[\s+\]`), "[]"},
	{regexp.MustCompile(`(?i)}\s*(else|catch|except|finally|elif|elseif|else if)`), "} ${1}"},
}

// Run beautifies C-style code. Ported from GenericCodeBeautify.mjs.
func (GenericCodeBeautify) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	code := in.String()
	var tokens []string
	for _, re := range gcbPreserveRes {
		code = gcbPreserve(re, code, &tokens)
	}

	for _, r := range gcbPhase1 {
		code = r.re.ReplaceAllString(code, r.with)
	}
	code = gcbIndent(code)
	for _, r := range gcbPhase2 {
		code = r.re.ReplaceAllString(code, r.with)
	}

	code = gcbPreservedToken.ReplaceAllStringFunc(code, func(m string) string {
		i, _ := strconv.Atoi(gcbPreservedToken.FindStringSubmatch(m)[1])
		return tokens[i]
	})
	return core.NewDish([]byte(code), core.TypeString), nil
}

// gcbPreserve replaces each match of re with a "###preservedTokenN###" placeholder,
// re-scanning from the match position (the .mjs's exec/lastIndex loop).
func gcbPreserve(re *regexp.Regexp, code string, tokens *[]string) string {
	pos := 0
	for {
		loc := re.FindStringIndex(code[pos:])
		if loc == nil {
			break
		}
		start, end := pos+loc[0], pos+loc[1]
		token := fmt.Sprintf("###preservedToken%d###", len(*tokens))
		*tokens = append(*tokens, code[start:end])
		code = code[:start] + token + code[end:]
		pos = start
	}
	return code
}

// gcbIndent inserts depth-based indentation after newlines. It ports the .mjs's
// index-based loop; iterating bytes matches its UTF-16 iteration because it only
// slices at ASCII '{', '}' and '\n' boundaries.
func gcbIndent(code string) string {
	i, level := 0, 0
	for i < len(code) {
		switch code[i] {
		case '{':
			level++
		case '\n':
			if i+1 >= len(code) {
				break
			}
			if code[i+1] == '}' {
				level--
			}
			indent := ""
			if level >= 0 {
				indent = strings.Repeat(" ", level*4)
			}
			code = code[:i+1] + indent + code[i+1:]
			if level > 0 {
				i += level * 4
			}
		}
		i++
	}
	return code
}
