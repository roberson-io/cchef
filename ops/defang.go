package ops

import (
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(DefangIPAddresses{})
	core.Register(DefangURL{})
	core.Register(FangURL{})
}

// Regexes ported verbatim from CyberChef. IPv4 / URL are RE2-compatible; the IPv6
// and domain matchers use lookahead/backreferences, so they run under regexp2.
var (
	defangIPv4Re = regexp.MustCompile(`(?:(?:\d|[01]?\d\d|2[0-4]\d|25[0-5])\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d|\d)(?:/\d{1,2})?`)
	defangIPv6Re = regexp2.MustCompile(`((?=.*::)(?!.*::.+::)(::)?([\dA-Fa-f]{1,4}:(:|\b)|){5}|([\dA-Fa-f]{1,4}:){6})((([\dA-Fa-f]{1,4}((?!\3)::|:\b|(?![\dA-Fa-f])))|(?!\2\3)){2}|(((2[0-4]|1\d|[1-9])?\d|25[0-5])\.?\b){4})`, regexp2.None)

	defangURLRe    = regexp.MustCompile(`(?i)[A-Z]+://[-\w]+(?:\.\w[-\w]*)+(?::\d+)?(?:/[^.!,?"<>\[\]{}\s\x{7f}-\x{ff}]*(?:[.!,?]+[^.!,?"<>\[\]{}\s\x{7f}-\x{ff}]+)*)?`)
	defangDomainRe = regexp2.MustCompile(`\b((?=[a-z0-9-]{1,63}\.)(xn--)?[a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,63}\b`, regexp2.IgnoreCase)

	defangHTTPRe = regexp.MustCompile(`(?i)http`)
)

// defangURLStr neutralises a single URL/domain: dots, the scheme and the "://"
// separator each optionally escaped.
func defangURLStr(url string, dots, http, slashes bool) string {
	if dots {
		url = strings.ReplaceAll(url, ".", "[.]")
	}
	if http {
		url = defangHTTPRe.ReplaceAllString(url, "hxxp")
	}
	if slashes {
		url = strings.ReplaceAll(url, "://", "[://]")
	}
	return url
}

// regexp2ReplaceFunc replaces every match of re in input using repl(match text).
func regexp2ReplaceFunc(re *regexp2.Regexp, input string, repl func(string) string) string {
	out, err := re.ReplaceFunc(input, func(m regexp2.Match) string { return repl(m.String()) }, -1, -1)
	if err != nil {
		return input
	}
	return out
}

// DefangIPAddresses makes IP addresses safe by wrapping their separators.
type DefangIPAddresses struct{}

// Meta returns the operation metadata.
func (DefangIPAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Defang IP Addresses",
		Module:      "Default",
		Description: "Takes a defangable input and 'Defangs' it; meaning it is safe to share.",
		InfoURL:     "https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DefangIPAddresses) Args() []core.ArgDef { return nil }

// Run defangs IP addresses.
func (DefangIPAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := defangIPv4Re.ReplaceAllStringFunc(in.String(), func(m string) string {
		return strings.ReplaceAll(m, ".", "[.]")
	})
	input = regexp2ReplaceFunc(defangIPv6Re, input, func(m string) string {
		return strings.ReplaceAll(m, ":", "[:]")
	})
	return core.NewDish([]byte(input), core.TypeString), nil
}

// DefangURL makes URLs safe to share by neutralising them.
type DefangURL struct{}

// Meta returns the operation metadata.
func (DefangURL) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Defang URL",
		Module:      "Default",
		Description: "Takes a Universal Resource Locator (URL) and 'Defangs' it; meaning the URL becomes invalid, neutralising the risk of accidentally clicking on a malicious link.<br><br>This is often used when sharing malicious links with colleagues.",
		InfoURL:     "https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DefangURL) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Escape dots", Type: core.ArgBoolean, Value: true},
		{Name: "Escape http", Type: core.ArgBoolean, Value: true},
		{Name: "Escape ://", Type: core.ArgBoolean, Value: true},
		{Name: "Process", Type: core.ArgOption, Value: []string{"Valid domains and full URLs", "Only full URLs", "Everything"}},
	}
}

// Run defangs the URL.
func (DefangURL) Run(in *core.Dish, args []any) (*core.Dish, error) {
	dots := args[0].(bool)
	http := args[1].(bool)
	slashes := args[2].(bool)
	process := args[3].(string)
	input := in.String()

	defang := func(m string) string { return defangURLStr(m, dots, http, slashes) }
	switch process {
	case "Valid domains and full URLs":
		input = defangURLRe.ReplaceAllStringFunc(input, defang)
		input = regexp2ReplaceFunc(defangDomainRe, input, defang)
	case "Only full URLs":
		input = defangURLRe.ReplaceAllStringFunc(input, defang)
	case "Everything":
		input = defangURLStr(input, dots, http, slashes)
	}
	return core.NewDish([]byte(input), core.TypeString), nil
}

// FangURL reverses Defang URL, restoring a usable URL.
type FangURL struct{}

// Meta returns the operation metadata.
func (FangURL) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Fang URL",
		Module:      "Default",
		Description: "Takes a 'Defanged' Universal Resource Locator (URL) and 'Fangs' it. Meaning, it removes the alterations (defanging) that render it invalid.",
		InfoURL:     "https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FangURL) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Restore [.]", Type: core.ArgBoolean, Value: true},
		{Name: "Restore hxxp", Type: core.ArgBoolean, Value: true},
		{Name: "Restore ://", Type: core.ArgBoolean, Value: true},
	}
}

// Run fangs the URL.
func (FangURL) Run(in *core.Dish, args []any) (*core.Dish, error) {
	dots := args[0].(bool)
	http := args[1].(bool)
	slashes := args[2].(bool)
	url := in.String()
	if dots {
		url = strings.ReplaceAll(url, "[.]", ".")
	}
	if http {
		url = strings.ReplaceAll(url, "hxxp", "http")
	}
	if slashes {
		url = strings.ReplaceAll(url, "[://]", "://")
	}
	return core.NewDish([]byte(url), core.TypeString), nil
}
