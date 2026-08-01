package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The shape of a domain name: a run of labels, each of letters and digits with
// hyphens allowed between them, ending in a top-level label of letters. The
// look-ahead measures the label — 63 characters is as long as one may be —
// before any of it is consumed. The underscore form allows `_` as well, which is
// how the records DMARC and DKIM publish are named.
const (
	domainPattern = jsWordBefore +
		`((?=[a-z0-9-]{1,63}\.)(xn--)?[a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,63}` +
		jsWordAfter

	dmarcDomainPattern = jsWordBefore +
		`((?=[a-z0-9_-]{1,63}\.)(xn--)?[a-z0-9_]+(-[a-z0-9_]+)*\.)+[a-z]{2,63}` +
		jsWordAfter
)

var (
	domainRegexp      = jsRegex(domainPattern, regexp2.IgnoreCase)
	dmarcDomainRegexp = jsRegex(dmarcDomainPattern, regexp2.IgnoreCase)
)

// ExtractDomains pulls the fully qualified domain names out of the input.
type ExtractDomains struct{}

// Meta returns the operation metadata.
func (ExtractDomains) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract domains",
		Module: "Regex",
		Description: "Extracts fully qualified domain names.<br>Note that this will not " +
			"include paths. Use <strong>Extract URLs</strong> to find entire URLs.",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractDomains) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
		{Name: "Underscore (DMARC, DKIM, etc)", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the domain names.
func (ExtractDomains) Run(in *core.Dish, args []any) (*core.Dish, error) {
	displayTotal, _ := args[0].(bool)
	sortResults, _ := args[1].(bool)
	unique, _ := args[2].(bool)
	underscore, _ := args[3].(bool)

	re := domainRegexp
	if underscore {
		re = dmarcDomainRegexp
	}
	var less func(a, b string) bool
	if sortResults {
		less = caseInsensitiveLess
	}

	found := extractSearch(in.String(), re, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

func init() { core.Register(ExtractDomains{}) }
