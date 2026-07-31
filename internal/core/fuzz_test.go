package core

import (
	"strings"
	"testing"
)

// The recipe parser is the one part of core that reads text cchef did not
// write: a recipe from -e, a recipe file, a shared URL, or the staged recipe
// on disk. It must reject anything malformed rather than panic.

func FuzzParseRecipeConfig(f *testing.F) {
	for _, seed := range []string{
		"",
		"[]",
		`[{"op":"To Base64","args":["A-Za-z0-9+/="]}]`,
		`[{"op":"To Hex","args":["Space"],"disabled":true}]`,
		`[{"op":"Fork","args":["\\n","\\n",false]}]`,
		"To_Base64('A-Za-z0-9+/=')",
		"To_Base64()To_Hex('Space')",
		"ROT13(true,true,false,13)",
		"To_Hex('Space'/disabled)",
		"To_Hex('Space'/breakpoint)",
		`Find_/_Replace({'option':'Regex','string':'a'},'b',true,false,true,false)`,
		"A(",
		")A(",
		"A('unterminated",
		`[{"op":"No Such Operation","args":[]}]`,
		`[{"op":123}]`,
		"[{",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, recipe string) {
		r, err := ParseRecipeConfig(recipe)
		if err != nil {
			return
		}
		// The Chef format writes an operation name unquoted, so a name
		// containing the format's own punctuation has no written form. No real
		// operation is named that way, and both cchef and CyberChef reject such
		// a name when the recipe runs, so those recipes are out of scope here.
		for _, step := range r {
			if strings.ContainsAny(step.Op, "()',/") {
				return
			}
		}
		// Everything else must survive being written back out and read again:
		// the Chef format is what a shared URL carries, so a recipe that
		// cannot round-trip would be silently altered by sharing it.
		pretty := GeneratePrettyRecipe(r, false)
		again, err := ParseRecipeConfig(pretty)
		if err != nil {
			t.Fatalf("re-parsing generated recipe %q: %v", pretty, err)
		}
		if len(again) != len(r) {
			t.Fatalf("round trip changed step count: %d -> %d via %q", len(r), len(again), pretty)
		}
		for i := range r {
			if again[i].Op != r[i].Op {
				t.Fatalf("round trip changed step %d: %q -> %q via %q", i, r[i].Op, again[i].Op, pretty)
			}
			if again[i].Disabled != r[i].Disabled || again[i].Breakpoint != r[i].Breakpoint {
				t.Fatalf("round trip changed the flags on step %d via %q", i, pretty)
			}
		}
	})
}

// FuzzEncodeURIFragment checks the encoder that puts a recipe into a URL
// fragment: it must terminate on any input and emit only characters that are
// legal there.
func FuzzEncodeURIFragment(f *testing.F) {
	for _, seed := range []string{"", "hello", "a b", "To_Base64('A-Za-z0-9+/=')", "\x00\x01", "é", "🙂", "%%%"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := EncodeURIFragment(s)
		if strings.ContainsAny(got, " \"<>\\^`{|}") {
			t.Fatalf("EncodeURIFragment(%q) = %q, which contains a character illegal in a fragment", s, got)
		}
	})
}

// FuzzCoerceArg runs the argument coercion every recipe value passes through.
// It must reject a value it cannot use rather than panic on it.
func FuzzCoerceArg(f *testing.F) {
	f.Add(0, "text")
	f.Add(1, "12")
	f.Add(2, "true")
	f.Add(3, "Space")
	f.Add(4, "")
	f.Fuzz(func(t *testing.T, which int, value string) {
		defs := []ArgDef{
			{Name: "s", Type: ArgString},
			{Name: "n", Type: ArgNumber},
			{Name: "b", Type: ArgBoolean},
			{Name: "o", Type: ArgOption, Value: []string{"Space", "Comma"}},
			{Name: "t", Type: ArgToggleString, ToggleValues: []string{"Hex", "UTF8"}},
			{Name: "e", Type: ArgEditableOption, Value: ""},
		}
		def := defs[((which%len(defs))+len(defs))%len(defs)]
		// Every value form a recipe can carry, so the type switches inside
		// coercion are all reachable from the fuzzer.
		for _, v := range []any{
			value,
			float64(len(value)),
			len(value)%2 == 0,
			ToggleString{Value: value, Option: "Hex"},
			map[string]any{"string": value, "option": "Hex"},
			[]any{value},
			nil,
		} {
			_, _ = CoerceArg(def, v)
		}
	})
}
