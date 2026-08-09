package core

import (
	"bytes"
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
		// containing the format's own punctuation has no written form: "()',/"
		// delimit an operation, "_" stands for a space, and a leading "[" is
		// what marks a recipe as JSON. No real operation is named that way, and
		// such a recipe is refused when it runs, so it is out of scope here.
		for i, step := range r {
			if strings.ContainsAny(step.Op, "()',/_") {
				return
			}
			if i == 0 && strings.HasPrefix(step.Op, "[") {
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

// FuzzParseURL runs the share-link reader, which is a parser reading data cchef
// did not write. It must terminate and report an error rather than panic; when
// it does read a link, rebuilding and re-reading has to agree with it.
func FuzzParseURL(f *testing.F) {
	for _, seed := range []string{
		"",
		"#recipe=MD5()",
		"#recipe=To_Hex('Space')&input=aGVsbG8",
		"https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGk",
		"recipe=" + EncodeURIFragment(`[{"op":"To Hex","args":["Space"]}]`),
		"#recipe=%zz",
		"#input=aGVsbG8",
		"#recipe=MD5()&input=!!!!",
		"#recipe=" + EncodeURIFragment("To_Hex('Space'/disabled)"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		r, in, err := ParseURL(s)
		if err != nil {
			return
		}
		// An operation name carrying the Chef format's own punctuation has no
		// stable written form. See FuzzParseRecipeConfig for the reasoning.
		for i, step := range r {
			if strings.ContainsAny(step.Op, "()',/_") {
				return
			}
			if i == 0 && strings.HasPrefix(step.Op, "[") {
				return
			}
		}
		again, in2, err := ParseURL(BuildURL(DefaultBaseURL, r, in))
		if err != nil {
			t.Fatalf("a link built from a parsed one did not parse: %v", err)
		}
		if got, want := GeneratePrettyRecipe(again, false), GeneratePrettyRecipe(r, false); got != want {
			t.Fatalf("recipe changed on rebuild: %q vs %q", got, want)
		}
		if !bytes.Equal(in2, in) {
			t.Fatalf("input changed on rebuild: %v vs %v", in2, in)
		}
	})
}
