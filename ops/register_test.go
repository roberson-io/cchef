package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestRegisterIntoCipherKey covers what CyberChef's Register.mjs cases test —
// a key lifted out of the data and handed to a cipher as a toggle-string
// argument — using XOR, whose result can be checked independently.
//
// Upstream's own two cases drive RC4 and AES Decrypt over data that is not
// actually hexadecimal while asking for it to be read as Hex. cchef drops the
// characters that are not hex digits, the way its own From Hex does; CryptoJS
// instead walks the string two characters at a time and reads anything it
// cannot parse as a zero byte. Both give the same answer with the key written
// in literally as with the register, so the extraction is what is being tested
// here and the reading of malformed hex is not.
func TestRegisterIntoCipherKey(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`k=(\w+)`, true, false, false}},
		{Op: "XOR", Args: []any{
			core.ToggleString{Value: "$R0", Option: "Hex"}, "Standard", false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("k=41;AAAA"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "*|upz\x00\x00\x00\x00" {
		t.Errorf("got %q, want every byte exclusive-ored with 0x41", out.String())
	}
}

// TestRegisterSubstitutesIntoLaterSteps checks the plain case: a capture group
// reaches a later step's argument.
func TestRegisterSubstitutesIntoLaterSteps(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`name=(\w+)`, true, false, false}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "GREETING", Option: "Simple string"},
			"Hello $R0", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("GREETING name=World"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "Hello World name=World" {
		t.Errorf("got %q", out.String())
	}
}

// TestRegisterLookbehind checks that an extractor using lookbehind — which RE2
// cannot compile — runs via the JavaScript-compatible fallback and fills the
// register from the capture group. Verified against the CyberChef engine.
func TestRegisterLookbehind(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`(?<=key=)(\w+)`, false, false, false}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "rest", Option: "Regex"},
			"[$R0]", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("key=SECRET rest"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "key=SECRET [SECRET]" {
		t.Errorf("got %q, want %q", out.String(), "key=SECRET [SECRET]")
	}
}

// TestRegisterSeveralGroups checks that each capture group gets its own number.
func TestRegisterSeveralGroups(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`(\w+)=(\w+)`, true, false, false}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "^.*$", Option: "Regex"},
			"$R1 is the $R0", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("key=secret"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "secret is the key" {
		t.Errorf("got %q", out.String())
	}
}

// TestRegisterTwiceContinuesNumbering checks that a second Register carries on
// where the first left off, so its group is $R1 rather than $R0 again.
func TestRegisterTwiceContinuesNumbering(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`a=(\w+)`, true, false, false}},
		{Op: "Register", Args: []any{`b=(\w+)`, true, false, false}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "^.*$", Option: "Regex"},
			"$R0/$R1", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("a=one b=two"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "one/two" {
		t.Errorf("got %q, want %q", out.String(), "one/two")
	}
}

// TestRegisterEscapedReferenceIsLiteral checks that a reference behind a
// backslash keeps its text and loses the backslash. Fork's merge delimiter is
// the observation point because it is used exactly as written.
func TestRegisterEscapedReferenceIsLiteral(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`(\w+)`, true, false, false}},
		{Op: "Fork", Args: []any{",", `\$R0/$R0`, false}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("word,word"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "word$R0/wordword" {
		t.Errorf("got %q, want the escaped reference kept and the live one filled", out.String())
	}
}

// TestRegisterNoMatchLeavesArgumentsAlone checks that when the extractor finds
// nothing, later steps keep the arguments they were given.
func TestRegisterNoMatchLeavesArgumentsAlone(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`absent=(\w+)`, true, false, false}},
		{Op: "Fork", Args: []any{",", "$R0", false}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("a,b"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "a$R0b" {
		t.Errorf("got %q, want the reference left as it was", out.String())
	}
}

// TestRegisterCaseInsensitiveFlag checks the flags reach the pattern.
func TestRegisterCaseInsensitiveFlag(t *testing.T) {
	for _, c := range []struct {
		name        string
		insensitive bool
		want        string
	}{
		{"insensitive matches", true, "KEY=VALUEVALUEx"},
		{"sensitive does not", false, "KEY=VALUE$R0x"},
	} {
		t.Run(c.name, func(t *testing.T) {
			recipe := core.Recipe{
				{Op: "Register", Args: []any{`key=(\w+)`, c.insensitive, false, false}},
				{Op: "Fork", Args: []any{",", "$R0", false}},
			}
			out, err := recipe.Execute(core.NewDish([]byte("KEY=VALUE,x"), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Errorf("got %q, want %q", out.String(), c.want)
			}
		})
	}
}

// TestRegisterBadPattern checks an invalid extractor is reported.
func TestRegisterBadPattern(t *testing.T) {
	recipe := core.Recipe{{Op: "Register", Args: []any{"([", true, false, false}}}
	if _, err := recipe.Execute(core.NewDish([]byte("x"), core.TypeString)); err == nil {
		t.Error("want an error for an invalid extractor")
	}
}

// TestRegisterResetsPerForkBranch checks that a register set inside one branch
// of a Fork does not leak into the next, which is why each tranche starts from
// the arguments as written.
func TestRegisterResetsPerForkBranch(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Fork", Args: []any{",", ",", false}},
		{Op: "Register", Args: []any{`(\w+)`, true, false, false}},
		{Op: "Find / Replace", Args: []any{
			core.ToggleString{Value: "^.*$", Option: "Regex"},
			"[$R0]", true, false, true, false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("one,two"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "[one],[two]" {
		t.Errorf("got %q, want %q", out.String(), "[one],[two]")
	}
}

// TestRegisterSkipsDisabledSteps checks that a disabled step is passed over
// when registers are written into later steps' arguments.
func TestRegisterSkipsDisabledSteps(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`(\w+)`, true, false, false}},
		{Op: "Fork", Args: []any{",", "$R0", false}, Disabled: true},
		{Op: "Fork", Args: []any{",", "$R0", false}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("ab,cd"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "abab cd" && out.String() != "ababcd" {
		t.Errorf("got %q, want the live step's delimiter filled", out.String())
	}
}

// TestRegisterIntoJSONToggleString checks a toggle-string argument in the shape
// a recipe read from JSON carries — an object rather than a Go value.
func TestRegisterIntoJSONToggleString(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`k=(\w+)`, true, false, false}},
		{Op: "XOR", Args: []any{
			map[string]any{"option": "Hex", "string": "$R0"}, "Standard", false,
		}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("k=41;AAAA"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "*|upz\x00\x00\x00\x00" {
		t.Errorf("got %q, want the register filled into the object's string", out.String())
	}
}

// TestRegisterLeavesOtherArgumentTypesAlone checks that arguments which are not
// text — numbers, booleans — pass through the substitution untouched.
func TestRegisterLeavesOtherArgumentTypesAlone(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Register", Args: []any{`(\w+)`, true, false, false}},
		{Op: "To Base", Args: []any{16.0}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("255"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "ff" {
		t.Errorf("got %q, want the numeric argument used as it was", out.String())
	}
}

// TestRegisterMultilineAndDotAllFlags checks the remaining two flags reach the
// pattern: multiline makes ^ and $ match at line breaks, and dot-matches-all
// makes . cover one. Fork's merge delimiter is the observation point because it
// is used exactly as written.
func TestRegisterMultilineAndDotAllFlags(t *testing.T) {
	for _, c := range []struct {
		name      string
		pattern   string
		multiline bool
		dotAll    bool
		want      string
	}{
		{"multiline on", `^(b)$`, true, false, "zba\nb"},
		{"multiline off", `^(b)$`, false, false, "z$R0a\nb"},
		{"dot matches all on", `(a.b)`, false, true, "za\nba\nb"},
		{"dot matches all off", `(a.b)`, false, false, "z$R0a\nb"},
	} {
		t.Run(c.name, func(t *testing.T) {
			recipe := core.Recipe{
				{Op: "Register", Args: []any{c.pattern, false, c.multiline, c.dotAll}},
				{Op: "Fork", Args: []any{",", "$R0", false}},
			}
			out, err := recipe.Execute(core.NewDish([]byte("z,a\nb"), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Errorf("got %q, want %q", out.String(), c.want)
			}
		})
	}
}
