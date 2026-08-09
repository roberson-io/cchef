package core

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncodeURIFragment(t *testing.T) {
	cases := map[string]string{
		"aGVsbG8":   "aGVsbG8",   // base64-ish, all safe
		"To_Hex()":  "To_Hex()",  // parens and underscore are safe
		"a+b=c":     "a%2Bb%3Dc", // + and = stay escaped
		"x&y":       "x%26y",     // & stays escaped
		"a b":       "a%20b",     // space encoded
		"'/?:@,;!$": "'/?:@,;!$", // safe set kept literal
	}
	for in, want := range cases {
		if got := EncodeURIFragment(in); got != want {
			t.Errorf("EncodeURIFragment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildURL(t *testing.T) {
	r := Recipe{{Op: "To Hex", Args: []any{"Space"}}}
	got := BuildURL(DefaultBaseURL, r, []byte("hello"))
	want := "https://gchq.github.io/CyberChef/#recipe=To_Hex('Space')&input=aGVsbG8"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}

func TestBuildURLNoInput(t *testing.T) {
	r := Recipe{{Op: "MD5"}}
	got := BuildURL(DefaultBaseURL, r, nil)
	want := "https://gchq.github.io/CyberChef/#recipe=MD5()"
	if got != want {
		t.Fatalf("BuildURL = %q\nwant %q", got, want)
	}
}

// TestBuildURLBase checks that a self-hosted instance replaces the public one
// and that the fragment is unchanged by it.
func TestBuildURLBase(t *testing.T) {
	r := Recipe{{Op: "To Hex", Args: []any{"Space"}}}
	for _, base := range []string{
		"https://cyberchef.internal.example/",
		"https://example.test/tools/cyberchef/",
		"http://localhost:8080/",
	} {
		got := BuildURL(base, r, []byte("hello"))
		if !strings.HasPrefix(got, base+"#recipe=") {
			t.Errorf("BuildURL(%q) = %q, want it to start with %q", base, got, base+"#recipe=")
		}
		if want := BuildURL(DefaultBaseURL, r, []byte("hello")); strings.TrimPrefix(got, base) != strings.TrimPrefix(want, DefaultBaseURL) {
			t.Errorf("the fragment differs between bases:\n %q\n %q", got, want)
		}
	}
}

// TestDecodeURIFragment mirrors TestEncodeURIFragment: what one writes the
// other has to read back.
func TestDecodeURIFragment(t *testing.T) {
	for want, in := range map[string]string{
		"aGVsbG8":   "aGVsbG8",
		"To_Hex()":  "To_Hex()",
		"a+b=c":     "a%2Bb%3Dc",
		"x&y":       "x%26y",
		"a b":       "a%20b",
		"'/?:@,;!$": "'/?:@,;!$",
	} {
		got, err := DecodeURIFragment(in)
		if err != nil {
			t.Errorf("DecodeURIFragment(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("DecodeURIFragment(%q) = %q, want %q", in, got, want)
		}
	}
	// A "+" is data, not a space: EncodeURIFragment writes a space as %20 and a
	// plus as %2B, so a literal + can only have come from the data itself.
	if got, _ := DecodeURIFragment("a+b"); got != "a+b" {
		t.Errorf("DecodeURIFragment(%q) = %q, want a+b", "a+b", got)
	}
	for _, bad := range []string{"%2", "%zz", "%"} {
		if _, err := DecodeURIFragment(bad); err == nil {
			t.Errorf("DecodeURIFragment(%q) should fail", bad)
		}
	}
}

// TestParseURL covers the shapes a share link arrives in.
func TestParseURL(t *testing.T) {
	const want = "To_Hex('Space')"
	for name, u := range map[string]string{
		"full URL":     "https://gchq.github.io/CyberChef/#recipe=To_Hex('Space')&input=aGVsbG8",
		"fragment":     "#recipe=To_Hex('Space')&input=aGVsbG8",
		"bare params":  "recipe=To_Hex('Space')&input=aGVsbG8",
		"self-hosted":  "https://cyberchef.internal.example/#recipe=To_Hex('Space')&input=aGVsbG8",
		"query string": "https://gchq.github.io/CyberChef/?recipe=To_Hex('Space')&input=aGVsbG8",
	} {
		r, in, err := ParseURL(u)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := GeneratePrettyRecipe(r, false); got != want {
			t.Errorf("%s: recipe = %q, want %q", name, got, want)
		}
		if string(in) != "hello" {
			t.Errorf("%s: input = %q, want hello", name, in)
		}
	}
}

// TestParseURLRecipeOnly covers a link that shares only the recipe.
func TestParseURLRecipeOnly(t *testing.T) {
	r, in, err := ParseURL("#recipe=MD5()")
	if err != nil {
		t.Fatal(err)
	}
	if got := GeneratePrettyRecipe(r, false); got != "MD5()" {
		t.Errorf("recipe = %q", got)
	}
	if in != nil {
		t.Errorf("input = %q, want none", in)
	}
}

// TestParseURLJSONRecipe covers a link whose recipe is a JSON array, which
// CyberChef also accepts, and one carrying disabled and breakpoint flags.
func TestParseURLJSONRecipe(t *testing.T) {
	r, _, err := ParseURL("#recipe=" + EncodeURIFragment(`[{"op":"To Hex","args":["Space"]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if got := GeneratePrettyRecipe(r, false); got != "To_Hex('Space')" {
		t.Errorf("recipe = %q", got)
	}

	r, _, err = ParseURL("#recipe=" + EncodeURIFragment("To_Hex('Space'/disabled)ROT13()"))
	if err != nil {
		t.Fatal(err)
	}
	if !r[0].Disabled {
		t.Errorf("disabled flag lost: %+v", r)
	}
}

// TestParseURLIgnoresDisplayParams checks that the settings only a browser can
// act on do not stop a link being read.
func TestParseURLIgnoresDisplayParams(t *testing.T) {
	r, in, err := ParseURL("#recipe=MD5()&input=aGVsbG8&theme=dark&ienc=65001&oenc=0&ieol=CRLF")
	if err != nil {
		t.Fatal(err)
	}
	if got := GeneratePrettyRecipe(r, false); got != "MD5()" {
		t.Errorf("recipe = %q", got)
	}
	if string(in) != "hello" {
		t.Errorf("input = %q", in)
	}
}

// TestParseURLInputVariants checks the base64 spellings a link may carry.
// BuildURL writes unpadded standard base64, but a link written by hand or by
// another tool may be padded or use the URL-safe alphabet. The bytes chosen
// encode to a value using both "+" and "/" in the standard alphabet, so the
// two alphabets genuinely differ.
func TestParseURLInputVariants(t *testing.T) {
	want := []byte{0xfb, 0xef, 0xbe, 0xff}
	for name, enc := range map[string]string{
		"raw standard":    base64.RawStdEncoding.EncodeToString(want),
		"padded":          base64.StdEncoding.EncodeToString(want),
		"raw url-safe":    base64.RawURLEncoding.EncodeToString(want),
		"padded url-safe": base64.URLEncoding.EncodeToString(want),
	} {
		u := "#recipe=MD5()&input=" + EncodeURIFragment(enc)
		_, got, err := ParseURL(u)
		if err != nil {
			t.Errorf("%s (%q): %v", name, enc, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s (%q): input = %v, want %v", name, enc, got, want)
		}
	}
}

// TestParseURLErrors covers links that cannot be read.
func TestParseURLErrors(t *testing.T) {
	for name, u := range map[string]string{
		"no recipe":        "https://gchq.github.io/CyberChef/#input=aGVsbG8",
		"empty recipe":     "#recipe=",
		"unparseable":      "#recipe=" + EncodeURIFragment("To_Hex("),
		"bad input base64": "#recipe=MD5()&input=" + EncodeURIFragment("not base64!!"),
		"bad escape":       "#recipe=%zz",
		"nothing at all":   "",
	} {
		if _, _, err := ParseURL(u); err == nil {
			t.Errorf("%s: expected an error for %q", name, u)
		}
	}
}

// TestParseURLRoundTrip is the property that matters for sharing: a link built
// from a recipe and input reads back as the same recipe and the same bytes.
func TestParseURLRoundTrip(t *testing.T) {
	cases := map[string]struct {
		recipe Recipe
		input  []byte
	}{
		"simple": {Recipe{{Op: "To Hex", Args: []any{"Space"}}}, []byte("hello")},
		"toggle string with awkward characters": {
			Recipe{{Op: "XOR", Args: []any{ToggleString{Value: "& + = ' %", Option: "UTF8"}, "Standard", false}}},
			[]byte("data"),
		},
		"binary input": {Recipe{{Op: "To Base64"}}, []byte{0x00, 0xff, 0x10, 0x80}},
		"no input":     {Recipe{{Op: "MD5"}}, nil},
		"disabled step": {
			Recipe{{Op: "To Hex", Disabled: true}, {Op: "ROT13"}},
			[]byte("x"),
		},
	}
	for name, tc := range cases {
		u := BuildURL(DefaultBaseURL, tc.recipe, tc.input)
		gotRecipe, gotInput, err := ParseURL(u)
		if err != nil {
			t.Errorf("%s: ParseURL(%q): %v", name, u, err)
			continue
		}
		// Both sides are normalised before comparing: a round trip turns a
		// ToggleString into a map and a number into a float64, so the recipe
		// that went in is put through the same conversion rather than compared
		// against its Go-struct spelling.
		if got, want := normaliseRecipe(t, gotRecipe), normaliseRecipe(t, tc.recipe); got != want {
			t.Errorf("%s: recipe = %q, want %q", name, got, want)
		}
		if !bytes.Equal(gotInput, tc.input) {
			t.Errorf("%s: input = %v, want %v", name, gotInput, tc.input)
		}
	}
}

// normaliseRecipe renders a recipe the way it looks once it has been through a
// text form and back, so a recipe built in Go compares equal to the same recipe
// read from a link: a ToggleString becomes a map and a number a float64.
func normaliseRecipe(t *testing.T, r Recipe) string {
	t.Helper()
	back, err := ParseRecipeConfig(GeneratePrettyRecipe(r, false))
	if err != nil {
		t.Fatalf("normalising %v: %v", r, err)
	}
	return GeneratePrettyRecipe(back, false)
}

// TestParseURLBareFlag covers a parameter written with no value, which
// CyberChef's own links can carry. It has nothing to read, so it is skipped
// rather than treated as an empty value.
func TestParseURLBareFlag(t *testing.T) {
	r, in, err := ParseURL("#recipe=MD5()&input=aGVsbG8&nosplash")
	if err != nil {
		t.Fatal(err)
	}
	if got := GeneratePrettyRecipe(r, false); got != "MD5()" {
		t.Errorf("recipe = %q", got)
	}
	if string(in) != "hello" {
		t.Errorf("input = %q", in)
	}
}
