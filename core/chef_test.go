package core

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGeneratePrettyRecipe(t *testing.T) {
	r := Recipe{
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
		{Op: "To Hex", Args: []any{"Space"}},
	}
	got := GeneratePrettyRecipe(r, false)
	want := "To_Base64('A-Za-z0-9+/=')To_Hex('Space')"
	if got != want {
		t.Fatalf("GeneratePrettyRecipe = %q\nwant %q", got, want)
	}
}

func TestGeneratePrettyRecipeNoArgs(t *testing.T) {
	r := Recipe{{Op: "MD5"}}
	if got := GeneratePrettyRecipe(r, false); got != "MD5()" {
		t.Fatalf("got %q, want MD5()", got)
	}
}

func TestGeneratePrettyRecipeFlags(t *testing.T) {
	r := Recipe{{Op: "To Hex", Args: []any{"Space"}, Disabled: true, Breakpoint: true}}
	got := GeneratePrettyRecipe(r, false)
	want := "To_Hex('Space'/disabled/breakpoint)"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestParseRecipeConfigChef(t *testing.T) {
	got, err := ParseRecipeConfig("To_Base64('A-Za-z0-9+/=')From_Base64('A-Za-z0-9+/=',true,false)")
	if err != nil {
		t.Fatalf("ParseRecipeConfig: %v", err)
	}
	want := Recipe{
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
		{Op: "From Base64", Args: []any{"A-Za-z0-9+/=", true, false}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseRecipeConfig = %#v\nwant %#v", got, want)
	}
}

func TestParseRecipeConfigJSON(t *testing.T) {
	got, err := ParseRecipeConfig(`[{"op":"To Base64","args":["A-Za-z0-9+/="]}]`)
	if err != nil {
		t.Fatalf("ParseRecipeConfig: %v", err)
	}
	if len(got) != 1 || got[0].Op != "To Base64" {
		t.Fatalf("got %#v", got)
	}
}

func TestRecipeRoundTripChef(t *testing.T) {
	orig := "To_Base64('A-Za-z0-9+/=')To_Hex('Space')"
	parsed, err := ParseRecipeConfig(orig)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := GeneratePrettyRecipe(parsed, false); got != orig {
		t.Fatalf("round trip = %q\nwant %q", got, orig)
	}
}

// TestChefEdgeCases covers the remaining chef serialization/parsing branches:
// newline output, a non-marshalable argument (chefArgs error -> empty args),
// empty and JSON-error recipe parsing, and Chef-format disabled/breakpoint flags.
func TestChefEdgeCases(t *testing.T) {
	// newline=true appends a trailing newline per step.
	if got := GeneratePrettyRecipe(Recipe{{Op: "To Hex", Args: []any{"Space"}}}, true); got != "To_Hex('Space')\n" {
		t.Fatalf("newline output = %q", got)
	}
	// An argument that cannot be JSON-marshalled makes chefArgs error, so the
	// generator emits empty args rather than failing.
	if got := GeneratePrettyRecipe(Recipe{{Op: "X", Args: []any{make(chan int)}}}, false); got != "X()" {
		t.Fatalf("unmarshalable arg = %q, want X()", got)
	}
	// An empty recipe string parses to an empty recipe.
	if r, err := ParseRecipeConfig(""); err != nil || len(r) != 0 {
		t.Fatalf("empty parse = %#v, %v", r, err)
	}
	// Chef-format args that don't form valid JSON error.
	if _, err := ParseRecipeConfig("Op(abc)"); err == nil {
		t.Fatal("expected an error for non-JSON Chef args")
	}
	// The /disabled and /breakpoint suffixes set the corresponding flags.
	r, err := ParseRecipeConfig("To_Hex('Space'/disabled)From_Hex('Auto'/breakpoint)")
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 2 || !r[0].Disabled || !r[1].Breakpoint {
		t.Fatalf("flags not parsed: %#v", r)
	}
}

// TestParseRecipeConfigBadJSON covers the invalid-JSON error branch for a
// bracket-prefixed (JSON) recipe.
func TestParseRecipeConfigBadJSON(t *testing.T) {
	if _, err := ParseRecipeConfig("[{"); err == nil {
		t.Fatal("expected an error for an invalid JSON recipe")
	}
}

// TestParseRecipeConfigRejectsMalformed covers the structural check CyberChef
// added in 11.3.0. A recipe that does not parse is refused rather than quietly
// doing nothing, so a mistyped one is not mistaken for a working one.
func TestParseRecipeConfigRejectsMalformed(t *testing.T) {
	cases := []struct{ name, recipe string }{
		{"no arguments at all", "A("},
		{"unmatched quotes", "A(" + strings.Repeat("'", 100)},
		{"a trailing escape", "A('" + strings.Repeat("\\", 100)},
		{"no closing bracket", "To_Upper_case('All'"},
		{"no bracket anywhere", "garbage"},
		{"an operation with no name", "()"},
		{"text after the last operation", "To_Upper_case('All')rubbish"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseRecipeConfig(c.recipe); err == nil {
				t.Errorf("%q was accepted", c.recipe)
			}
		})
	}
}

// TestParseRecipeConfigAcceptsValid checks the shapes that must keep working,
// including the ones CyberChef's own regression tests cover.
func TestParseRecipeConfigAcceptsValid(t *testing.T) {
	cases := []struct {
		name, recipe string
		want         int
	}{
		{"two operations", "From_Base64('A-Za-z0-9+/=',true)To_Hex('Space')", 2},
		{"no arguments", "To_Base64()", 1},
		{"the disabled and breakpoint flags", "A(/disabled/breakpoint)", 1},
		{"escaped quotes and backslashes", `A('\'\\')`, 1},
		{"a long quoted argument", "A('" + strings.Repeat("x", 10000) + "')", 1},
		{"newlines between operations", "To_Upper_case('All')\nTo_Base64('A-Za-z0-9+/=')", 2},
		{"a bracket inside a quoted argument", "Find_/_Replace('a)b','c')", 1},
		// Structurally sound, so it is accepted here and refused later when
		// no operation of that name is found — which is what upstream does.
		{"a stray closing bracket in the name", ")A()", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseRecipeConfig(c.recipe)
			if err != nil {
				t.Fatalf("%q was refused: %v", c.recipe, err)
			}
			if len(got) != c.want {
				t.Errorf("got %d operations, want %d", len(got), c.want)
			}
		})
	}
}

// TestParseRecipeConfigIsLinear checks that a hostile recipe is refused quickly.
// The equivalent parser in CyberChef needed a structural pre-pass to avoid
// catastrophic backtracking; Go's regexp cannot backtrack that way, and this
// keeps it honest.
func TestParseRecipeConfigIsLinear(t *testing.T) {
	for _, n := range []int{1000, 100000} {
		start := time.Now()
		if _, err := ParseRecipeConfig("A('" + strings.Repeat("\\", n)); err == nil {
			t.Errorf("n=%d: a malformed recipe was accepted", n)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("n=%d took %v", n, elapsed)
		}
	}
}
