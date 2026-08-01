package ops

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// The operation reads a property list and writes it out in the shorthand
// CyberChef shows: `plist => ` and the root container, braces round a
// dictionary and brackets round an array, one entry to a line indented by a tab
// per level, and `/plist` at the end.
//
// CyberChef reaches that shorthand by rewriting the raw XML with a chain of
// regular-expression replacements and then walking the resulting lines
// (src/core/operations/PLISTViewer.mjs). That approach loses information a
// listing needs — whitespace inside values, and which container each position
// belongs to — so this reads the document instead. The output format is
// CyberChef's; the reading of it is not.

// plistIndent is what one level of nesting is written with.
const plistIndent = "\t"

// plistArrow joins a name, or a position in an array, to its value.
const plistArrow = " => "

// The elements a property list is built from.
const (
	plistRoot       = "plist"
	plistDict       = "dict"
	plistArray      = "array"
	plistKey        = "key"
	plistString     = "string"
	plistInteger    = "integer"
	plistReal       = "real"
	plistDate       = "date"
	plistData       = "data"
	plistTrue       = "true"
	plistFalse      = "false"
	plistUnexpected = "unexpected element"
)

// plistValue is one thing a property list holds.
type plistValue interface {
	// write appends the value to out, indented for a value sitting at depth.
	write(out *strings.Builder, depth int)
}

// plistScalar is a value written as it stands, in quotes for a string.
type plistScalar struct {
	text   string
	quoted bool
}

func (s plistScalar) write(out *strings.Builder, _ int) {
	if s.quoted {
		out.WriteString(`"` + s.text + `"`)
		return
	}
	out.WriteString(s.text)
}

// plistEntry is one member of a dictionary.
type plistEntry struct {
	name  string
	value plistValue
}

// plistDictionary is a set of named values, kept in the order they were given.
type plistDictionary []plistEntry

func (d plistDictionary) write(out *strings.Builder, depth int) {
	out.WriteString("{\n")
	for _, entry := range d {
		out.WriteString(strings.Repeat(plistIndent, depth+1) + entry.name + plistArrow)
		entry.value.write(out, depth+1)
		out.WriteString("\n")
	}
	out.WriteString(strings.Repeat(plistIndent, depth) + "}")
}

// plistList is a run of values, written with the position of each.
type plistList []plistValue

func (l plistList) write(out *strings.Builder, depth int) {
	out.WriteString("[\n")
	for at, value := range l {
		out.WriteString(strings.Repeat(plistIndent, depth+1) + strconv.Itoa(at) + plistArrow)
		value.write(out, depth+1)
		out.WriteString("\n")
	}
	out.WriteString(strings.Repeat(plistIndent, depth) + "]")
}

// PlistViewer lays a property list out in a form that can be read.
type PlistViewer struct{}

// Meta returns the operation metadata.
func (PlistViewer) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "P-list Viewer",
		Module: "Default",
		Description: "In the macOS, iOS, NeXTSTEP, and GNUstep programming " +
			"frameworks, property list files are files that store serialized " +
			"objects. Property list files use the filename extension .plist, and " +
			"thus are often referred to as p-list files.<br><br>This operation " +
			"displays plist files in a human readable format.",
		InfoURL:    "https://wikipedia.org/wiki/Property_list",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (PlistViewer) Args() []core.ArgDef { return nil }

// Run lays the property list out.
func (PlistViewer) Run(in *core.Dish, args []any) (*core.Dish, error) {
	root, err := parsePlist(in.String())
	if err != nil {
		return nil, err
	}

	var out strings.Builder
	out.WriteString(plistRoot + plistArrow)
	root.write(&out, 0)
	out.WriteString("\n/" + plistRoot + "\n")

	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// parsePlist reads the one value a property list wraps.
func parsePlist(input string) (plistValue, error) {
	decoder := xml.NewDecoder(strings.NewReader(input))
	decoder.Strict = true

	if err := plistFindRoot(decoder); err != nil {
		return nil, err
	}

	start, err := plistNextElement(decoder)
	if err != nil {
		return nil, err
	}
	if start == nil {
		return nil, errors.New("the plist holds nothing")
	}

	root, err := plistReadValue(decoder, *start)
	if err != nil {
		return nil, err
	}

	// Anything beyond the one value the plist wraps is not a property list.
	extra, err := plistNextElement(decoder)
	if err != nil {
		return nil, err
	}
	if extra != nil {
		return nil, fmt.Errorf("%s: <%s> after the value the plist holds",
			plistUnexpected, extra.Name.Local)
	}
	return root, nil
}

// plistFindRoot advances to just inside the plist element.
func plistFindRoot(decoder *xml.Decoder) error {
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return errors.New("no <plist> element in the input")
		}
		if err != nil {
			return fmt.Errorf("not valid XML: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok {
			if start.Name.Local != plistRoot {
				return fmt.Errorf("%s: <%s> where <%s> was expected",
					plistUnexpected, start.Name.Local, plistRoot)
			}
			return nil
		}
	}
}

// plistNextElement advances to the next element, reporting none where the
// enclosing element ends first. Text between elements is skipped, which is how
// the indentation of the document itself is passed over.
func plistNextElement(decoder *xml.Decoder) (*xml.StartElement, error) {
	for {
		// Reading always happens inside the plist element, so the end of the
		// input arrives as a syntax error rather than as a clean finish.
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("not valid XML: %w", err)
		}
		switch t := token.(type) {
		case xml.StartElement:
			return &t, nil
		case xml.EndElement:
			return nil, nil
		}
	}
}

// plistReadValue reads the value the element just begun holds.
func plistReadValue(decoder *xml.Decoder, start xml.StartElement) (plistValue, error) {
	switch start.Name.Local {
	case plistDict:
		return plistReadDictionary(decoder)
	case plistArray:
		return plistReadList(decoder)
	case plistTrue, plistFalse:
		if err := decoder.Skip(); err != nil {
			return nil, fmt.Errorf("not valid XML: %w", err)
		}
		return plistScalar{text: start.Name.Local}, nil
	case plistString, plistInteger, plistReal, plistDate, plistData:
		text, err := plistReadText(decoder, start)
		if err != nil {
			return nil, err
		}
		return plistScalar{text: text, quoted: start.Name.Local == plistString}, nil
	default:
		return nil, fmt.Errorf("%s: <%s>", plistUnexpected, start.Name.Local)
	}
}

// plistReadText reads the characters an element holds, with the whitespace of
// the document trimmed from each end but anything inside kept.
func plistReadText(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var text string
	if err := decoder.DecodeElement(&text, &start); err != nil {
		return "", fmt.Errorf("not valid XML: %w", err)
	}
	return strings.TrimSpace(text), nil
}

// plistReadDictionary reads the named values of a dictionary, which are written
// as a key followed by the value it names.
func plistReadDictionary(decoder *xml.Decoder) (plistValue, error) {
	dictionary := plistDictionary{}
	for {
		start, err := plistNextElement(decoder)
		if err != nil {
			return nil, err
		}
		if start == nil {
			return dictionary, nil
		}
		if start.Name.Local != plistKey {
			return nil, fmt.Errorf("%s: <%s> where <%s> was expected in a <%s>",
				plistUnexpected, start.Name.Local, plistKey, plistDict)
		}

		name, err := plistReadText(decoder, *start)
		if err != nil {
			return nil, err
		}

		valueStart, err := plistNextElement(decoder)
		if err != nil {
			return nil, err
		}
		if valueStart == nil {
			return nil, fmt.Errorf("<%s>%s</%s> has no value after it", plistKey, name, plistKey)
		}
		value, err := plistReadValue(decoder, *valueStart)
		if err != nil {
			return nil, err
		}
		dictionary = append(dictionary, plistEntry{name: name, value: value})
	}
}

// plistReadList reads the values of an array.
func plistReadList(decoder *xml.Decoder) (plistValue, error) {
	list := plistList{}
	for {
		start, err := plistNextElement(decoder)
		if err != nil {
			return nil, err
		}
		if start == nil {
			return list, nil
		}
		value, err := plistReadValue(decoder, *start)
		if err != nil {
			return nil, err
		}
		list = append(list, value)
	}
}

func init() { core.Register(PlistViewer{}) }
