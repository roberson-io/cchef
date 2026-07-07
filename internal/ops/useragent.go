package ops

import (
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/internal/core"
)

// useragent_rules.go is generated from ua-parser-js 2.0.10; see tools/uagen.
//go:generate go run ../../tools/uagen/gen.go -in ../../tools/uagen/uarules.json -out useragent_rules.go

func init() {
	core.Register(ParseUserAgent{})
}

// uaRule is one [regexes → props] entry in a ua-parser-js detection table.
type uaRule struct {
	regexes []*regexp2.Regexp
	props   []uaProp
}

// uaProp describes how a matched capture group maps onto a result property,
// mirroring ua-parser-js's prop-spec forms (capture / static / fn / replace).
type uaProp struct {
	prop   string
	kind   string // "cap", "static", "fn", "replace"
	static string
	// fn forms
	fn    string // "strMapper", "strTest", "lowerize", "trim"
	fnMap *uaLookup
	// strTest form
	testRe          *regexp2.Regexp
	ifTrue, ifFalse string
	// replace form
	replRe  *regexp2.Regexp
	repl    string
	replFn  string
	replMap *uaLookup
}

type uaLookupEntry struct {
	key   string
	vals  []string
	undef bool
}

// uaLookup is an ordered value→key map for strMapper.
type uaLookup struct {
	entries   []uaLookupEntry
	hasStar   bool
	star      string
	starUndef bool
}

// uaTrim strips leading whitespace (ua-parser-js trim, no length limit here).
func uaTrim(s string) string { return strings.TrimLeft(s, " \t\n\r\f\v") }

// uaStrMapper reverse-maps str to a key whose value equals str (case-insensitive),
// falling back to the "*" entry, then to str. ok=false means "undefined". Ported
// from ua-parser-js strMapper (has() is case-insensitive equality). The current
// generated ruleset happens not to use undef entries, a "?" key or starUndef, so
// those arms (and the "strMapper not found -> delete" arms in uaApplyProps) are
// not reached by real user agents; they are exercised directly in the tests.
func uaStrMapper(str string, m *uaLookup) (string, bool) {
	for _, e := range m.entries {
		if e.undef {
			continue
		}
		for _, v := range e.vals {
			if strings.EqualFold(str, v) {
				if e.key == "?" { // UNKNOWN
					return "", false
				}
				return e.key, true
			}
		}
	}
	if m.hasStar {
		if m.starUndef {
			return "", false
		}
		return m.star, true
	}
	return str, true
}

// uaStrTest returns ifTrue when testRe matches str, else ifFalse.
func uaStrTest(str string, p uaProp) string {
	if ok, _ := p.testRe.MatchString(str); ok {
		return p.ifTrue
	}
	return p.ifFalse
}

// uaApplyProps assigns the matched capture groups to result properties per the
// rule's prop specs. Ported from ua-parser-js rgxMapper's inner loop.
func uaApplyProps(res map[string]string, props []uaProp, matched []string) {
	k := 0
	for _, p := range props {
		k++
		match := ""
		if k < len(matched) {
			match = matched[k]
		}
		switch p.kind {
		case "cap":
			set(res, p.prop, match)
		case "static":
			res[p.prop] = p.static
		case "fn":
			switch p.fn {
			case "lowerize":
				res[p.prop] = strings.ToLower(match)
			case "trim":
				res[p.prop] = uaTrim(match)
			case "strTest":
				if match != "" {
					res[p.prop] = uaStrTest(match, p)
				} else {
					delete(res, p.prop)
				}
			case "strMapper":
				if match != "" {
					if v, ok := uaStrMapper(match, p.fnMap); ok {
						res[p.prop] = v
					} else {
						delete(res, p.prop)
					}
				} else {
					delete(res, p.prop)
				}
			}
		case "replace":
			if match != "" {
				v, _ := p.replRe.Replace(match, p.repl, -1, -1)
				switch p.replFn {
				case "lowerize":
					v = strings.ToLower(v)
				case "trim":
					v = uaTrim(v)
				case "strMapper":
					if mv, ok := uaStrMapper(v, p.replMap); ok {
						res[p.prop] = mv
					} else {
						delete(res, p.prop)
					}
					continue
				}
				res[p.prop] = v
			} else {
				delete(res, p.prop)
			}
		}
	}
}

// set assigns match to prop when truthy, otherwise clears it (JS `x ? x : undefined`).
func set(res map[string]string, prop, match string) {
	if match != "" {
		res[prop] = match
	} else {
		delete(res, prop)
	}
}

// uaParseCategory runs a detection table against ua and returns the matched
// properties (first matching rule wins). Ported from ua-parser-js rgxMapper.
func uaParseCategory(ua string, rules []uaRule) map[string]string {
	res := map[string]string{}
	for _, rule := range rules {
		for _, re := range rule.regexes {
			m, _ := re.FindStringMatch(ua)
			if m == nil {
				continue
			}
			groups := m.Groups()
			matched := make([]string, len(groups))
			for gi := range groups {
				matched[gi] = groups[gi].String()
			}
			uaApplyProps(res, rule.props, matched)
			return res
		}
	}
	return res
}

// ParseUserAgent parses a User-Agent string into browser/device/engine/os/cpu
// details, ported from ua-parser-js 2.0.10's default detection tables.
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

// field returns res[key] or "unknown".
func uaField(res map[string]string, key string) string {
	if v, ok := res[key]; ok && v != "" {
		return v
	}
	return "unknown"
}

// Run parses the user agent. Ported from CyberChef ParseUserAgent.mjs.
func (ParseUserAgent) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ua := in.String()
	browser := uaParseCategory(ua, uaTables["browser"])
	cpu := uaParseCategory(ua, uaTables["cpu"])
	device := uaParseCategory(ua, uaTables["device"])
	engine := uaParseCategory(ua, uaTables["engine"])
	os := uaParseCategory(ua, uaTables["os"])

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
