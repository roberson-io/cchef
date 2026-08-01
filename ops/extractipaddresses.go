package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The shapes an address can take. A decimal address is four groups of 0–255; an
// octal one is four groups written with a leading zero. The two forms are not
// mixed within an address. The look-behind and look-ahead keep a match from
// starting or ending in the middle of a longer run of digits, so that
// 1.2.3.4.5.6.7.8 gives two addresses rather than several overlapping ones.
const (
	ipv4DecimalByte = `(?:25[0-5]|2[0-4]\d|1?[0-9]\d|\d)`
	ipv4OctalByte   = `(?:0[1-3]?[0-7]{1,2})`
	ipLookBehind    = `(?<!\d)`
	ipLookAhead     = `(?!\d)`

	ipv4Decimal = `(?:` + ipLookBehind + ipv4DecimalByte + `\.){3}` +
		`(?:` + ipv4DecimalByte + ipLookAhead + `)`
	ipv4Octal = `(?:` + ipLookBehind + ipv4OctalByte + `\.){3}` +
		`(?:` + ipv4OctalByte + ipLookAhead + `)`
	ipv4Pattern = `(?:` + ipv4Decimal + `|` + ipv4Octal + `)`

	// The IPv6 shape allows the run of zero groups to be written once, which is
	// what the look-ahead pair checks, and uses back-references to hold the
	// separator it settled on.
	//
	// Two of those back-references are written differently here than upstream.
	// A reference to a group that did not take part in the match means opposite
	// things in the two engines: JavaScript reads it as the empty string, so
	// `(?!\3)` becomes `(?!)` and can never succeed, while this engine reads it
	// as unmatchable, so the same `(?!\3)` always succeeds. Left as written, the
	// pattern would match nothing at all in the middle of ordinary words and
	// report the letters around it as addresses. Each reference is therefore
	// guarded by a test for whether its group took part, which is what the
	// JavaScript reading amounts to.
	ipv6Pattern = `((?=.*::)(?!.*::.+::)(::)?([\dA-F]{1,4}:(:|\b)|){5}|([\dA-F]{1,4}:){6})` +
		`(([\dA-F]{1,4}(` + ipv6NotSeparator + `::|:\b|(?![\dA-F])))|` + ipv6NotBoth + `){2}`

	// ipv6NotSeparator is `(?!\3)` as JavaScript reads it: when group 3 did not
	// take part, the reference is the empty string and the look-ahead fails.
	ipv6NotSeparator = `(?(3)(?!\3)|(?!))`

	// ipv6NotBoth is `(?!\2\3)` as JavaScript reads it, for each combination of
	// the two groups having taken part or not.
	ipv6NotBoth = `(?(2)(?(3)(?!\2\3)|(?!\2))|(?(3)(?!\3)|(?!)))`
)

// ipLocalRanges matches the addresses set aside for private networks and for the
// host itself, which can be left out.
var ipLocalRanges = regexp2.MustCompile(
	`^(?:10\..+|192\.168\..+|172\.(?:1[6-9]|2\d|3[01])\..+|127\..+)`, regexp2.None)

// The compiled forms of each combination of the two versions.
var (
	ipv4Only = regexp2.MustCompile(ipv4Pattern, regexp2.IgnoreCase)
	ipv6Only = regexp2.MustCompile(ipv6Pattern, regexp2.IgnoreCase)
	ipBoth   = regexp2.MustCompile(ipv4Pattern+`|`+ipv6Pattern, regexp2.IgnoreCase)
)

// ExtractIPAddresses pulls the IP addresses out of the input.
type ExtractIPAddresses struct{}

// Meta returns the operation metadata.
func (ExtractIPAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract IP addresses",
		Module: "Regex",
		Description: "Extracts all IPv4 and IPv6 addresses.<br><br>Warning: Given a string " +
			"<code>1.2.3.4.5.6.7.8</code>, this will match <code>1.2.3.4 and 5.6.7.8</code> " +
			"so always check the original input!",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractIPAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "IPv4", Type: core.ArgBoolean, Value: true},
		{Name: "IPv6", Type: core.ArgBoolean, Value: false},
		{Name: "Remove local IPv4 addresses", Type: core.ArgBoolean, Value: false},
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the addresses.
func (ExtractIPAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	includeV4, _ := args[0].(bool)
	includeV6, _ := args[1].(bool)
	removeLocal, _ := args[2].(bool)
	displayTotal, _ := args[3].(bool)
	sortResults, _ := args[4].(bool)
	unique, _ := args[5].(bool)

	re := ipPattern(includeV4, includeV6)
	if re == nil {
		return core.NewDish(nil, core.TypeString), nil
	}

	var remove *regexp2.Regexp
	if removeLocal {
		remove = ipLocalRanges
	}
	var less func(a, b string) bool
	if sortResults {
		less = extractIPLess
	}

	found := extractSearch(in.String(), re, remove, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

// ipPattern returns the pattern for the versions asked for, or nil when neither
// was.
func ipPattern(includeV4, includeV6 bool) *regexp2.Regexp {
	switch {
	case includeV4 && includeV6:
		return ipBoth
	case includeV4:
		return ipv4Only
	case includeV6:
		return ipv6Only
	}
	return nil
}

func init() { core.Register(ExtractIPAddresses{}) }
