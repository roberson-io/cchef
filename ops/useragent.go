package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseUserAgent{})
}

// uaRule is one detection rule: a pattern and an action that records what a
// match reveals. The rule tables live in useragent_rules.go.
type uaRule struct {
	re *regexp.Regexp
	// unless vetoes the rule when it also matches, for names that appear
	// inside unrelated products.
	unless *regexp.Regexp
	// apply records what the match reveals; returning false declines the
	// match so later rules are still tried.
	apply func(ua string, m []string, res map[string]string) bool
}

// uaParseCategory runs a category's rules in order; the first accepted match
// decides and later rules are not consulted.
func uaParseCategory(ua string, rules []uaRule) map[string]string {
	res := map[string]string{}
	for _, r := range rules {
		if r.unless != nil && r.unless.MatchString(ua) {
			continue
		}
		if m := r.re.FindStringSubmatch(ua); m != nil && r.apply(ua, m, res) {
			return res
		}
	}
	return res
}

// uaCap returns the first non-empty capture group, so alternations can bind
// the same meaning to different groups.
func uaCap(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return ""
}

// uaSetVersion records a version if there is one; an empty capture stays
// unset so it renders as unknown.
func uaSetVersion(res map[string]string, v string) {
	if v != "" {
		res["version"] = v
	}
}

// uaDots turns the underscore-separated versions used inside parenthesised
// system strings ("10_15_7") into dotted form.
func uaDots(v string) string { return strings.ReplaceAll(v, "_", ".") }

// uaNamed builds the common browser rule: a static name with the version in
// the first non-empty capture group.
func uaNamed(name, pattern string) uaRule {
	return uaRule{re: regexp.MustCompile("(?i)" + pattern), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = name
		uaSetVersion(res, uaCap(m))
		return true
	}}
}

// uaCapNamed builds the rule form that keeps the input's own spelling of the
// name: group 1 is the name, group 2 the version.
func uaCapNamed(pattern string) uaRule {
	return uaRule{re: regexp.MustCompile("(?i)" + pattern), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = m[1]
		uaSetVersion(res, m[2])
		return true
	}}
}

// uaOSNamed builds the common OS rule: a static name with the version in the
// first non-empty capture group, underscores dotted.
func uaOSNamed(name, pattern string) uaRule {
	return uaRule{re: regexp.MustCompile("(?i)" + pattern), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = name
		uaSetVersion(res, uaDots(uaCap(m)))
		return true
	}}
}

// uaDevice builds a device rule; the model is the first non-empty capture
// group unless a static model is given, and empty fields stay unset.
func uaDevice(pattern, model, vendor, devType string) uaRule {
	return uaRule{re: regexp.MustCompile("(?i)" + pattern), apply: func(_ string, m []string, res map[string]string) bool {
		mdl := model
		if mdl == "" {
			mdl = uaCap(m)
		}
		if mdl != "" {
			res["model"] = mdl
		}
		if vendor != "" {
			res["vendor"] = vendor
		}
		if devType != "" {
			res["type"] = devType
		}
		return true
	}}
}

// uaField returns res[key] or "unknown".
func uaField(res map[string]string, key string) string {
	if v, ok := res[key]; ok && v != "" {
		return v
	}
	return "unknown"
}

// ParseUserAgent identifies the browser, device, engine, OS and CPU described
// by a user-agent string.
type ParseUserAgent struct{}

// Meta returns the operation metadata.
func (ParseUserAgent) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse User Agent",
		Module:      "Default",
		Description: "Attempts to identify and categorise information contained in a user-agent string.",
		InfoURL:     "https://wikipedia.org/wiki/User_agent",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseUserAgent) Args() []core.ArgDef { return nil }

// Run parses the user agent.
func (ParseUserAgent) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ua := in.String()
	browser := uaParseCategory(ua, uaBrowserRules)
	cpu := uaParseCategory(ua, uaCPURules)
	device := uaParseCategory(ua, uaDeviceRules)
	engine := uaParseCategory(ua, uaEngineRules)
	os := uaParseCategory(ua, uaOSRules)

	out := fmt.Sprintf(`Browser
    Name: %s
    Version: %s
Device
    Model: %s
    Type: %s
    Vendor: %s
Engine
    Name: %s
    Version: %s
OS
    Name: %s
    Version: %s
CPU
    Architecture: %s`,
		uaField(browser, "name"), uaField(browser, "version"),
		uaField(device, "model"), uaField(device, "type"), uaField(device, "vendor"),
		uaField(engine, "name"), uaField(engine, "version"),
		uaField(os, "name"), uaField(os, "version"),
		uaField(cpu, "architecture"))
	return core.NewDish([]byte(out), core.TypeString), nil
}
