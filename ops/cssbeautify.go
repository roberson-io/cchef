package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(CSSBeautify{})
}

// CSSBeautify struct.
type CSSBeautify struct{}

// Meta returns the operation metadata.
func (CSSBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CSS Beautify",
		Module:      "Code",
		Description: "Indents and prettifies Cascading Style Sheets (CSS) code.",
		InfoURL:     "https://wikipedia.org/wiki/CSS",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CSSBeautify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Indent string", Type: core.ArgString, Value: `\t`},
	}
}

// Run beautifies the CSS input.
func (CSSBeautify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	indentStr := opsutil.ParseEscapedChars(args[0].(string))
	return core.NewDish([]byte(vkCSSBeautify(in.String(), indentStr)), core.TypeString), nil
}

// cssEmptyMarkerRe collapses "~::~<whitespace>~::~" to a single marker
// (vkbeautify's /~::~\s{0,}~::~/g).
var cssEmptyMarkerRe = regexp.MustCompile("~::~[" + jsWSChars + "]*~::~")

// vkShiftAt returns shift[idx], or the literal "undefined" when idx is out of
// range — replicating JS array access on a negative/overflowing index, which
// vkbeautify.css relies on for unbalanced braces.
func vkShiftAt(shift []string, idx int) string {
	if idx < 0 || idx >= len(shift) {
		return "undefined"
	}
	return shift[idx]
}

// vkCSSBeautify ports vkbeautify.css: mark boundaries at {, }, ; and comments,
// split, then indent each segment by nesting depth.
func vkCSSBeautify(text, indentStr string) string {
	shift := createShiftArr("    ") // this.shift default (4 spaces) when step is falsy
	if indentStr != "" {
		shift = createShiftArr(indentStr)
	}
	s := jsWSRun.ReplaceAllString(text, " ")
	s = strings.ReplaceAll(s, "{", "{~::~")
	s = strings.ReplaceAll(s, "}", "~::~}~::~")
	s = strings.ReplaceAll(s, ";", ";~::~")
	s = strings.ReplaceAll(s, "/*", "~::~/*")
	s = strings.ReplaceAll(s, "*/", "*/~::~")
	s = cssEmptyMarkerRe.ReplaceAllString(s, "~::~")
	ar := strings.Split(s, "~::~")

	deep := 0
	var sb strings.Builder
	for _, seg := range ar {
		switch {
		case strings.Contains(seg, "{"):
			sb.WriteString(vkShiftAt(shift, deep) + seg)
			deep++
		case strings.Contains(seg, "}"):
			deep--
			sb.WriteString(vkShiftAt(shift, deep) + seg)
		default:
			// vkbeautify's redundant /\*\\/ branch and the else branch both indent
			// at the current depth, so they are one case here.
			sb.WriteString(vkShiftAt(shift, deep) + seg)
		}
	}
	return strings.TrimLeft(sb.String(), "\n")
}
