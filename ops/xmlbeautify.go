package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(XMLBeautify{})
}

// XMLBeautify struct.
type XMLBeautify struct{}

// Meta returns the operation metadata.
func (XMLBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XML Beautify",
		Module:      "Code",
		Description: "Indents and prettifies eXtensible Markup Language (XML) code.",
		InfoURL:     "https://wikipedia.org/wiki/XML",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XMLBeautify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Indent string", Type: core.ArgString, Value: `\t`},
	}
}

// Run beautifies the XML input.
func (XMLBeautify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	indentStr := opsutil.ParseEscapedChars(args[0].(string))
	return core.NewDish([]byte(vkXMLBeautify(in.String(), indentStr)), core.TypeString), nil
}

// XML Beautify split/classification regexes, ported from vkbeautify.xml.
var (
	xmlnsColonSplitRe = regexp.MustCompile("[" + jsWSChars + "]*xmlns:")
	xmlnsEqSplitRe    = regexp.MustCompile("[" + jsWSChars + "]*xmlns=")
	reStartTagWord    = regexp.MustCompile(`^<\w`)
	reCloseTagWord    = regexp.MustCompile(`^</\w`)
	reOpenTagName     = regexp.MustCompile(`^<[\w:.,-]+`)
	reCloseTagName    = regexp.MustCompile(`^</[\w:.,-]+`)
	reHasTagWord      = regexp.MustCompile(`<\w`)
)

// vkXMLBeautify ports vkbeautify.xml: it marks segment boundaries with "~::~",
// splits, then walks the segments tracking nesting depth and comment/CDATA state.
func vkXMLBeautify(text, indentStr string) string {
	shift := createShiftArr("    ") // this.shift default (4 spaces) when step is falsy
	if indentStr != "" {
		shift = createShiftArr(indentStr)
	}
	s := xmlTagWSRe.ReplaceAllString(text, "><")
	s = strings.ReplaceAll(s, "<", "~::~<")
	s = xmlnsColonSplitRe.ReplaceAllString(s, "~::~xmlns:")
	s = xmlnsEqSplitRe.ReplaceAllString(s, "~::~xmlns=")
	ar := strings.Split(s, "~::~")

	f := &xmlFormatter{shift: shift}
	for ix := range ar {
		prev := ""
		if ix > 0 {
			prev = ar[ix-1]
		}
		f.step(prev, ar[ix])
	}
	// The library slices a single leading newline (str[0] == '\n' ? str.slice(1)).
	return strings.TrimPrefix(f.sb.String(), "\n")
}

// xmlFormatter accumulates the beautified output while tracking nesting depth and
// whether we are inside a comment / CDATA / DOCTYPE run (where content stays on one
// line).
type xmlFormatter struct {
	shift     []string
	deep      int
	inComment bool
	sb        strings.Builder
}

// step appends one segment, replicating vkbeautify.xml's branch cascade.
func (f *xmlFormatter) step(prev, cur string) {
	switch {
	case strings.Contains(cur, "<!"):
		f.sb.WriteString(f.shift[f.deep] + cur)
		f.inComment = true
		if strings.Contains(cur, "-->") || strings.Contains(cur, "]>") || strings.Contains(cur, "!DOCTYPE") {
			f.inComment = false
		}
	case strings.Contains(cur, "-->") || strings.Contains(cur, "]>"):
		f.sb.WriteString(cur)
		f.inComment = false
	case f.isMatchingClose(prev, cur):
		f.sb.WriteString(cur)
		if !f.inComment {
			f.deep--
		}
	default:
		f.stepTag(cur)
	}
}

// isMatchingClose reports whether cur is the closing tag for the element opened in
// prev (e.g. prev "<div ..." followed by cur "</div>").
func (f *xmlFormatter) isMatchingClose(prev, cur string) bool {
	if !reStartTagWord.MatchString(prev) || !reCloseTagWord.MatchString(cur) {
		return false
	}
	return reOpenTagName.FindString(prev) == strings.Replace(reCloseTagName.FindString(cur), "/", "", 1)
}

// stepTag handles the tag-shaped segments (open, self-contained, close, empty,
// processing instruction, xmlns) and the plain-text fallthrough.
func (f *xmlFormatter) stepTag(cur string) {
	hasTagWord := reHasTagWord.MatchString(cur)
	hasClose := strings.Contains(cur, "</")
	// Note: the library also has a `<elm>...</elm>` branch (a segment matching both
	// <\w and </). It is unreachable here because the "~::~" markers are inserted
	// before every "<", so each segment holds exactly one "<" — it can carry a
	// tag-word or a "</", never both — so that branch is omitted.
	switch {
	case hasTagWord && !strings.Contains(cur, "/>"): // <elm>
		f.write(f.deep, cur)
		if !f.inComment {
			f.deep++
		}
	case hasClose: // </elm>
		if !f.inComment && f.deep > 0 {
			// A closing tag with nothing open leaves the depth at zero rather
			// than going negative, so unbalanced input is laid out flat.
			f.deep--
		}
		f.write(f.deep, cur)
	case strings.Contains(cur, "/>"): // <elm/>
		f.write(f.deep, cur)
	case strings.Contains(cur, "<?"), strings.Contains(cur, "xmlns:"), strings.Contains(cur, "xmlns="):
		f.sb.WriteString(f.shift[f.deep] + cur)
	default:
		f.sb.WriteString(cur)
	}
}

// write appends shift[idx]+cur when not inside a comment/CDATA run, else just cur
// (matching the library's `!inComment ? shift+seg : seg`).
func (f *xmlFormatter) write(idx int, cur string) {
	if f.inComment {
		f.sb.WriteString(cur)
		return
	}
	f.sb.WriteString(f.shift[idx] + cur)
}
