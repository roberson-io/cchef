package ops

import (
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The sets of characters a run can be made of. The first three are ASCII; the
// rest are the Unicode categories that stand for the same ideas.
const (
	stringsAlphanumeric    = `A-Z\d`
	stringsPunctuation     = `/\-:.,_$%'"()<>= !\[\]{}@`
	stringsPrintable       = `\x20-\x7e`
	stringsUniAlphanumeric = `\p{L}\p{N}`
	stringsUniPunctuation  = `\p{P}\p{Z}`
	stringsUniPrintable    = `\p{L}\p{M}\p{Z}\p{S}\p{N}\p{P}`
)

// The kinds of run that can be looked for. CyberChef's list also carries two
// headings, "[ASCII]" and "[Unicode]", which name the groups the others fall
// into rather than being choices; cchef leaves them out, as it does for the
// other operations whose options are grouped.
const (
	stringsAlnumASCII   = "Alphanumeric + punctuation (A)"
	stringsPrintASCII   = "All printable chars (A)"
	stringsNullASCII    = "Null-terminated strings (A)"
	stringsAlnumUnicode = "Alphanumeric + punctuation (U)"
	stringsPrintUnicode = "All printable chars (U)"
	stringsNullUnicode  = "Null-terminated strings (U)"
)

// The ways the characters of a run can be laid out in the bytes.
const (
	stringsSingleByte = "Single byte"
	stringsUTF16LE    = "16-bit littleendian"
	stringsUTF16BE    = "16-bit bigendian"
	stringsAnyWidth   = "All"
)

// Strings pulls the runs of readable characters out of the input.
type Strings struct{}

// Meta returns the operation metadata.
func (Strings) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strings",
		Module:      "Regex",
		Description: "Extracts all strings from the input.",
		InfoURL:     "https://wikipedia.org/wiki/Strings_(Unix)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Strings) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: []string{
			stringsSingleByte, stringsUTF16LE, stringsUTF16BE, stringsAnyWidth,
		}},
		{Name: "Minimum length", Type: core.ArgNumber, Integer: true, Value: 4.0},
		{Name: "Match", Type: core.ArgOption, Value: []string{
			stringsAlnumASCII, stringsPrintASCII, stringsNullASCII,
			stringsAlnumUnicode, stringsPrintUnicode, stringsNullUnicode,
		}},
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the runs.
func (Strings) Run(in *core.Dish, args []any) (*core.Dish, error) {
	encoding, _ := args[0].(string)
	minLen, _ := args[1].(float64)
	matchType, _ := args[2].(string)
	displayTotal, _ := args[3].(bool)
	sortResults, _ := args[4].(bool)
	unique, _ := args[5].(bool)

	re, err := stringsPattern(encoding, matchType, int(minLen))
	if err != nil {
		return nil, err
	}

	var less func(a, b string) bool
	if sortResults {
		less = caseInsensitiveLess
	}

	// The input reaches the operation as bytes. CyberChef reads them as text the
	// way it reads any byte array: as UTF-8 when the whole run is valid UTF-8,
	// and a character per byte otherwise. Which of the two it lands on changes
	// what counts as a letter, so it has to be the same reading here.
	found := extractSearch(byteArrayToUtf8(in.Bytes()), re, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

// stringsPattern builds the pattern for one combination of the options: a set of
// characters, laid out as the encoding says, repeated at least minLen times, and
// for the null-terminated kinds followed by the byte that ends them.
func stringsPattern(encoding, matchType string, minLen int) (*regexp2.Regexp, error) {
	set := stringsCharacterSet(matchType)
	pattern := stringsLayout(encoding, set)
	pattern += fmt.Sprintf("{%d,}", minLen)
	if strings.Contains(matchType, "Null-terminated") {
		pattern += "\x00"
	}
	return regexp2.Compile(pattern, regexp2.IgnoreCase)
}

// stringsCharacterSet returns the set of characters a run of the given kind is
// made of.
func stringsCharacterSet(matchType string) string {
	switch matchType {
	case stringsAlnumASCII:
		return "[" + stringsAlphanumeric + stringsPunctuation + "]"
	case stringsPrintASCII, stringsNullASCII:
		return "[" + stringsPrintable + "]"
	case stringsAlnumUnicode:
		return "[" + stringsUniAlphanumeric + stringsUniPunctuation + "]"
	case stringsPrintUnicode, stringsNullUnicode:
		return "[" + stringsUniPrintable + "]"
	}
	return ""
}

// stringsLayout wraps the character set in the null bytes that a wide character
// is padded with, so that text stored two bytes to the character is found as
// well as text stored one.
func stringsLayout(encoding, set string) string {
	switch encoding {
	case stringsAnyWidth:
		return "(\x00?" + set + "\x00?)"
	case stringsUTF16LE:
		return "(" + set + "\x00)"
	case stringsUTF16BE:
		return "(\x00" + set + ")"
	}
	return set
}

func init() { core.Register(Strings{}) }
