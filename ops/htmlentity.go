package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/htmlent"
)

//go:generate go run ../../tools/htmlentgen/gen.go

func init() {
	core.Register(ToHTMLEntity{})
	core.Register(FromHTMLEntity{})
}

// htmlEntityModes are the "Convert to" options for To HTML Entity.
var htmlEntityModes = []string{"Named entities", "Numeric entities", "Hex entities"}

// ToHTMLEntity converts characters to HTML entities.
type ToHTMLEntity struct{}

// Meta returns the operation metadata.
func (ToHTMLEntity) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To HTML Entity",
		Module:      "Encodings",
		Description: "Converts characters to HTML entities, e.g. & becomes &amp;.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToHTMLEntity) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Convert all characters", Type: core.ArgBoolean, Value: false},
		{Name: "Convert to", Type: core.ArgOption, Value: htmlEntityModes},
	}
}

// Run encodes the input. Ported from ToHTMLEntity.mjs. Iterates code points
// (like Utils.strToCharcode, which merges surrogate pairs).
func (ToHTMLEntity) Run(in *core.Dish, args []any) (*core.Dish, error) {
	convertAll := args[0].(bool)
	mode := args[1].(string)
	numeric := mode == "Numeric entities"
	hexa := mode == "Hex entities"

	var sb strings.Builder
	for _, r := range dishText(in) {
		sb.WriteString(htmlEncodeRune(r, convertAll, numeric, hexa))
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// latin1Max is the highest code point representable in a single Latin-1 byte;
// above it a character is always escaped rather than emitted verbatim.
const latin1Max = 255

// htmlEncodeRune returns the encoding of one code point for To HTML Entity,
// honouring the convert-all flag and the numeric/hex mode selection.
func htmlEncodeRune(r rune, convertAll, numeric, hexa bool) string {
	c := int(r)
	name, named := htmlent.ByteToEntity[c]
	// The table holds bare names, so the delimiters are added here rather than
	// stored; a name cannot then carry stray punctuation of its own.
	entity := "&" + name + ";"
	switch {
	case convertAll && numeric:
		return fmt.Sprintf("&#%d;", c)
	case convertAll && hexa:
		return fmt.Sprintf("&#x%02x;", c)
	case convertAll:
		if named {
			return entity
		}
		return fmt.Sprintf("&#%d;", c)
	case numeric:
		if c > latin1Max || named {
			return fmt.Sprintf("&#%d;", c)
		}
		return string(r)
	case hexa:
		if c > latin1Max || named {
			return fmt.Sprintf("&#x%02x;", c)
		}
		return string(r)
	default:
		switch {
		case named:
			return entity
		case c > latin1Max:
			return fmt.Sprintf("&#%d;", c)
		default:
			return string(r)
		}
	}
}

// FromHTMLEntity converts HTML entities back into raw characters.
type FromHTMLEntity struct{}

// Meta returns the operation metadata.
func (FromHTMLEntity) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From HTML Entity",
		Module:      "Encodings",
		Description: "Converts HTML entities back into characters, e.g. &amp; becomes &.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromHTMLEntity) Args() []core.ArgDef { return nil }

// HTML entity matchers ported from FromHTMLEntity.mjs.
var (
	htmlEntityRE  = regexp.MustCompile(`&(#?x?[a-zA-Z0-9]{1,20});`)
	htmlNumericRE = regexp.MustCompile(`^#\d{1,6}$`)
	htmlHexRE     = regexp.MustCompile(`(?i)^#x[\da-f]{2,8}$`)
)

// Run decodes the input. Ported from FromHTMLEntity.mjs. Named entities resolve
// via the table, then numeric (&#d;) and hex (&#xH;) forms; anything else is left
// verbatim.
func (FromHTMLEntity) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var sb strings.Builder
	last := 0
	for _, loc := range htmlEntityRE.FindAllStringSubmatchIndex(input, -1) {
		sb.WriteString(input[last:loc[0]])
		name := input[loc[2]:loc[3]]
		switch {
		case htmlent.EntityToByte[name] != 0:
			sb.WriteRune(rune(htmlent.EntityToByte[name])) // #nosec G115 -- entity table code points are within the Unicode range
		case len(name) > 1 && name[0] == '#' && htmlNumericRE.MatchString(name):
			n, _ := strconv.Atoi(name[1:])
			sb.WriteRune(rune(n)) // #nosec G115 -- numeric entity bounded to 6 digits by the regex
		case len(name) > 3 && name[0] == '#' && htmlHexRE.MatchString(name):
			n, _ := strconv.ParseInt(name[2:], 16, 32)
			sb.WriteRune(rune(n))
		default:
			sb.WriteString(input[loc[0]:loc[1]])
		}
		last = loc[1]
	}
	sb.WriteString(input[last:])
	return core.NewDish(textAsBytes(sb.String()), core.TypeString), nil
}
