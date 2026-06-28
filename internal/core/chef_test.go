package core

import (
	"reflect"
	"testing"
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
