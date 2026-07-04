//go:build ignore

// Command uagen generates internal/ops/useragent_rules.go from the ua-parser-js
// rule tables previously extracted to uarules.json by dump.mjs.
//
// Run from the repository root:
//
//	go run tools/uagen/gen.go
//
// or via `go generate ./internal/ops/`. See tools/uagen/README.md for how to
// refresh uarules.json when bumping the pinned ua-parser-js version.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	in := flag.String("in", "tools/uagen/uarules.json", "path to the extracted ua-parser-js rules JSON")
	out := flag.String("out", "internal/ops/useragent_rules.go", "path to the generated Go file")
	flag.Parse()

	raw, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalf("read %s: %v", *in, err)
	}
	var tables map[string][]any
	if err := json.Unmarshal(raw, &tables); err != nil {
		log.Fatalf("parse %s: %v", *in, err)
	}

	src := render(tables)
	formatted, err := format.Source([]byte(src))
	if err != nil {
		log.Fatalf("gofmt generated source: %v", err)
	}
	if err := os.WriteFile(*out, formatted, 0o644); err != nil {
		log.Fatalf("write %s: %v", *out, err)
	}
	fmt.Printf("wrote %s\n", *out)
}

// goq renders s as a Go double-quoted string literal. The extracted rules are
// pure ASCII, so this matches the json.dumps escaping the original generator used.
func goq(s string) string { return strconv.Quote(s) }

// pyStr mirrors Python str(): strings pass through; anything else is formatted
// with the default representation (defensive — the rule values are all strings).
func pyStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// flagsOpts maps a JS regex flag string to the equivalent regexp2 options.
func flagsOpts(fl string) string {
	var opts []string
	if strings.Contains(fl, "i") {
		opts = append(opts, "regexp2.IgnoreCase")
	}
	if strings.Contains(fl, "m") {
		opts = append(opts, "regexp2.Multiline")
	}
	if strings.Contains(fl, "s") {
		opts = append(opts, "regexp2.Singleline")
	}
	if len(opts) == 0 {
		return "regexp2.None"
	}
	return strings.Join(opts, " | ")
}

// rx renders a compiled-regex expression from a {re, flags} spec.
func rx(re, fl string) string {
	return fmt.Sprintf("rx(%s, %s)", goq(re), flagsOpts(fl))
}

// m is a shorthand for the untyped-map shape decoded from JSON.
func asMap(v any) map[string]any { return v.(map[string]any) }
func asSlice(v any) []any        { return v.([]any) }

// gomap renders a strMapper value→key lookup as a *uaLookup literal.
// entries is a list of [key, valspec] pairs.
func gomap(entries []any) string {
	var parts []string
	star, hasStar, starUndef := `""`, "false", "false"
	for _, e := range asSlice2(entries) {
		pair := asSlice(e)
		key := pair[0].(string)
		vs := asMap(pair[1])
		undef, vals := "false", "nil"
		switch vs["k"] {
		case "undef":
			undef = "true"
		case "val":
			vals = fmt.Sprintf("[]string{%s}", goq(pyStr(vs["v"])))
		case "arr":
			var qs []string
			for _, x := range asSlice(vs["arr"]) {
				qs = append(qs, goq(pyStr(x)))
			}
			vals = fmt.Sprintf("[]string{%s}", strings.Join(qs, ", "))
		}
		parts = append(parts, fmt.Sprintf("{key: %s, vals: %s, undef: %s}", goq(key), vals, undef))
		if key == "*" {
			hasStar = "true"
			switch vs["k"] {
			case "undef":
				starUndef = "true"
			case "val":
				star = goq(pyStr(vs["v"]))
			case "arr":
				arr := asSlice(vs["arr"])
				if len(arr) > 0 {
					star = goq(pyStr(arr[0]))
				}
			}
		}
	}
	return fmt.Sprintf("&uaLookup{entries: []uaLookupEntry{%s}, hasStar: %s, star: %s, starUndef: %s}",
		strings.Join(parts, ", "), hasStar, star, starUndef)
}

// asSlice2 returns entries as []any (entries is already []any but kept explicit).
func asSlice2(v []any) []any { return v }

// prop renders one prop spec as a uaProp literal.
func prop(p map[string]any) string {
	t := p["t"].(string)
	pr := goq(p["prop"].(string))
	switch t {
	case "cap":
		return fmt.Sprintf("{prop: %s, kind: \"cap\"}", pr)
	case "static":
		return fmt.Sprintf("{prop: %s, kind: \"static\", static: %s}", pr, goq(pyStr(p["val"])))
	case "fn":
		fn := p["fn"].(string)
		if fn == "strTest" {
			ent := entriesToMap(asMap(asSlice(p["args"])[0])["entries"])
			tr := asMap(ent["test"])
			ift := asMap(ent["ifTrue"])
			iff := asMap(ent["ifFalse"])
			return fmt.Sprintf("{prop: %s, kind: \"fn\", fn: \"strTest\", testRe: %s, ifTrue: %s, ifFalse: %s}",
				pr, rx(tr["re"].(string), tr["flags"].(string)),
				goq(pyStr(getOr(ift, "v", ""))), goq(pyStr(getOr(iff, "v", ""))))
		}
		m := "nil"
		if fn == "strMapper" {
			if args, ok := p["args"].([]any); ok && len(args) > 0 {
				m = gomap(asSlice(asMap(args[0])["entries"]))
			}
		}
		return fmt.Sprintf("{prop: %s, kind: \"fn\", fn: %s, fnMap: %s}", pr, goq(fn), m)
	case "replace":
		var b strings.Builder
		fmt.Fprintf(&b, "{prop: %s, kind: \"replace\", replRe: %s, repl: %s",
			pr, rx(p["re"].(string), p["flags"].(string)), goq(p["repl"].(string)))
		if fn, ok := p["fn"].(string); ok && fn != "" {
			fmt.Fprintf(&b, ", replFn: %s", goq(fn))
			if args, ok := p["args"].([]any); ok && len(args) > 0 {
				if a0 := asMap(args[0]); a0["k"] == "map" {
					fmt.Fprintf(&b, ", replMap: %s", gomap(asSlice(a0["entries"])))
				}
			}
		}
		b.WriteString("}")
		return b.String()
	}
	panic("unknown prop type: " + t)
}

// entriesToMap turns a [[key, valspec], ...] list into a keyed map.
func entriesToMap(v any) map[string]any {
	out := map[string]any{}
	for _, e := range asSlice(v) {
		pair := asSlice(e)
		out[pair[0].(string)] = pair[1]
	}
	return out
}

// getOr returns m[key] or fallback when absent.
func getOr(m map[string]any, key string, fallback any) any {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}

// render assembles the full Go source for useragent_rules.go.
func render(tables map[string][]any) string {
	var lines []string
	lines = append(lines,
		"// Code generated by tools/uagen from ua-parser-js 2.0.10; DO NOT EDIT.",
		"",
		"package ops",
		"",
		`import "github.com/dlclark/regexp2"`,
		"",
		"func rx(pattern string, opt regexp2.RegexOptions) *regexp2.Regexp { return regexp2.MustCompile(pattern, opt) }",
		"",
		"var uaTables = map[string][]uaRule{")
	for _, cat := range []string{"browser", "cpu", "device", "engine", "os"} {
		lines = append(lines, fmt.Sprintf("\t%s: {", goq(cat)))
		for _, r := range tables[cat] {
			rule := asMap(r)
			var res []string
			for _, x := range asSlice(rule["regexes"]) {
				xm := asMap(x)
				res = append(res, rx(xm["re"].(string), xm["flags"].(string)))
			}
			var props []string
			for _, p := range asSlice(rule["props"]) {
				props = append(props, prop(asMap(p)))
			}
			lines = append(lines, fmt.Sprintf("\t\t{regexes: []*regexp2.Regexp{%s}, props: []uaProp{%s}},",
				strings.Join(res, ", "), strings.Join(props, ", ")))
		}
		lines = append(lines, "\t},")
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n") + "\n"
}
