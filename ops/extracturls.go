package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The shape of a URL: a protocol, a host of two or more labels, an optional
// port, and an optional path. Requiring the protocol is what keeps ordinary
// prose out of the results. The path may hold a full stop or a comma, but not as
// its last character, so that a URL written at the end of a sentence does not
// take the sentence's punctuation with it.
const (
	urlProtocol = `[A-Z]+://`
	urlHostname = `[-\w]+(?:\.\w[-\w]*)+`
	urlPort     = `:\d+`
	urlPathChar = `[^.!,?"<>\[\]{}\s\x7F-\xFF]`
	urlPath     = `/` + urlPathChar + `*(?:[.!,?]+` + urlPathChar + `+)*`

	urlPattern = urlProtocol + urlHostname + `(?:` + urlPort + `)?(?:` + urlPath + `)?`
)

var urlRegexp = jsRegex(urlPattern, regexp2.IgnoreCase)

// ExtractURLs pulls the URLs out of the input.
type ExtractURLs struct{}

// Meta returns the operation metadata.
func (ExtractURLs) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract URLs",
		Module: "Regex",
		Description: "Extracts Uniform Resource Locators (URLs) from the input. The " +
			"protocol (http, ftp etc.) is required otherwise there will be far too many " +
			"false positives.",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractURLs) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the URLs.
func (ExtractURLs) Run(in *core.Dish, args []any) (*core.Dish, error) {
	displayTotal, _ := args[0].(bool)
	sortResults, _ := args[1].(bool)
	unique, _ := args[2].(bool)

	var less func(a, b string) bool
	if sortResults {
		less = caseInsensitiveLess
	}

	found := extractSearch(in.String(), urlRegexp, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

func init() { core.Register(ExtractURLs{}) }
