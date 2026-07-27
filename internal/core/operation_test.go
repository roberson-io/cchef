package core

import "testing"

func TestCoerceArgNumber(t *testing.T) {
	def := ArgDef{Name: "Amount", Type: ArgNumber, Value: 13}
	// JSON numbers decode to float64; CLI may pass int. Both should coerce.
	got, err := CoerceArg(def, float64(7))
	if err != nil {
		t.Fatalf("CoerceArg float64: %v", err)
	}
	if got.(float64) != 7 {
		t.Fatalf("got %v, want 7", got)
	}
	if _, err := CoerceArg(def, "not-a-number"); err == nil {
		t.Fatal("expected error coercing non-numeric string to number")
	}
}

func TestCoerceArgBoolean(t *testing.T) {
	def := ArgDef{Name: "Flag", Type: ArgBoolean, Value: true}
	got, err := CoerceArg(def, false)
	if err != nil {
		t.Fatalf("CoerceArg bool: %v", err)
	}
	if got.(bool) != false {
		t.Fatalf("got %v, want false", got)
	}
}

func TestCoerceArgOptionValidates(t *testing.T) {
	def := ArgDef{Name: "Delimiter", Type: ArgOption, Value: []string{"Space", "Comma"}}
	if _, err := CoerceArg(def, "Space"); err != nil {
		t.Fatalf("valid option rejected: %v", err)
	}
	if _, err := CoerceArg(def, "Nope"); err == nil {
		t.Fatal("expected error for option value outside the allowed set")
	}
}

func TestCoerceArgToggleString(t *testing.T) {
	def := ArgDef{Name: "Key", Type: ArgToggleString, ToggleValues: []string{"Hex", "UTF8"}}
	got, err := CoerceArg(def, ToggleString{Value: "ff", Option: "Hex"})
	if err != nil {
		t.Fatalf("CoerceArg toggleString: %v", err)
	}
	ts := got.(ToggleString)
	if ts.Value != "ff" || ts.Option != "Hex" {
		t.Fatalf("got %+v", ts)
	}
	if _, err := CoerceArg(def, ToggleString{Value: "ff", Option: "Bad"}); err == nil {
		t.Fatal("expected error for toggle option outside ToggleValues")
	}
}

func TestCoerceArgToggleStringFromMap(t *testing.T) {
	def := ArgDef{Name: "Key", Type: ArgToggleString, ToggleValues: []string{"Hex", "UTF8"}}
	// JSON recipes decode a toggleString into a map[string]any.
	got, err := CoerceArg(def, map[string]any{"string": "ff", "option": "Hex"})
	if err != nil {
		t.Fatalf("CoerceArg map toggleString: %v", err)
	}
	ts := got.(ToggleString)
	if ts.Value != "ff" || ts.Option != "Hex" {
		t.Fatalf("got %+v", ts)
	}
	if _, err := CoerceArg(def, 123); err == nil {
		t.Fatal("expected error for non-toggleString value")
	}
}

func TestDefaultArgs(t *testing.T) {
	defs := []ArgDef{
		{Name: "Alphabet", Type: ArgEditableOption, Value: "A-Za-z0-9+/="},
		{Name: "Amount", Type: ArgNumber, Value: 13},
	}
	got := DefaultArgs(defs)
	if len(got) != 2 || got[0] != "A-Za-z0-9+/=" || got[1] != 13 {
		t.Fatalf("DefaultArgs = %v", got)
	}
}

// TestCoerceArgNumberConversions covers every numeric input type toFloat accepts
// (recipes built in Go may pass int/int64/float32, not just JSON's float64).
func TestCoerceArgNumberConversions(t *testing.T) {
	def := ArgDef{Name: "Amount", Type: ArgNumber, Value: 0}
	for _, v := range []any{int(7), int64(7), float32(7), float64(7)} {
		got, err := CoerceArg(def, v)
		if err != nil {
			t.Fatalf("CoerceArg(%T): %v", v, err)
		}
		if got.(float64) != 7 {
			t.Fatalf("CoerceArg(%T) = %v, want 7", v, got)
		}
	}
}

// TestCoerceArgNumberBounds covers the Min/Max validation branches.
func TestCoerceArgNumberBounds(t *testing.T) {
	lo, hi := 0.0, 10.0
	def := ArgDef{Name: "Amount", Type: ArgNumber, Value: 5, Min: &lo, Max: &hi}
	if _, err := CoerceArg(def, float64(5)); err != nil {
		t.Fatalf("in-range value rejected: %v", err)
	}
	if _, err := CoerceArg(def, float64(-1)); err == nil {
		t.Fatal("expected below-minimum error")
	}
	if _, err := CoerceArg(def, float64(11)); err == nil {
		t.Fatal("expected above-maximum error")
	}
}

// TestCoerceArgStringTypes covers the ArgString/ArgEditableOption arms, including
// the wrong-type error.
func TestCoerceArgStringTypes(t *testing.T) {
	for _, ty := range []ArgType{ArgString, ArgEditableOption} {
		def := ArgDef{Name: "S", Type: ty, Value: ""}
		if got, err := CoerceArg(def, "ok"); err != nil || got.(string) != "ok" {
			t.Fatalf("%s: got %v, err %v", ty, got, err)
		}
		if _, err := CoerceArg(def, 123); err == nil {
			t.Fatalf("%s: expected error for non-string", ty)
		}
	}
}

// TestCoerceArgInteger covers the whole-number constraint, which CyberChef
// declares as `integer: true` on a number ingredient.
func TestCoerceArgInteger(t *testing.T) {
	def := ArgDef{Name: "Code length", Type: ArgNumber, Value: 6, Integer: true}
	if _, err := CoerceArg(def, float64(6)); err != nil {
		t.Fatalf("whole number rejected: %v", err)
	}
	if _, err := CoerceArg(def, float64(-3)); err != nil {
		t.Fatalf("negative whole number rejected: %v", err)
	}
	if _, err := CoerceArg(def, 6.5); err == nil {
		t.Fatal("expected a fractional value to be rejected")
	}
	if _, err := CoerceArg(ArgDef{Name: "n", Type: ArgNumber}, 6.5); err != nil {
		t.Fatalf("fractional value rejected without the constraint: %v", err)
	}
}

// TestCoerceArgNonEmpty covers the constraint CyberChef writes as
// `allowEmpty: false`. Leaving it off keeps a string argument permissive, which
// is what every other operation expects.
func TestCoerceArgNonEmpty(t *testing.T) {
	for _, ty := range []ArgType{ArgString, ArgEditableOption} {
		def := ArgDef{Name: "Name", Type: ty, Value: "Account", NonEmpty: true}
		if _, err := CoerceArg(def, "Account"); err != nil {
			t.Fatalf("%s: filled value rejected: %v", ty, err)
		}
		if _, err := CoerceArg(def, ""); err == nil {
			t.Fatalf("%s: expected an empty value to be rejected", ty)
		}
		if _, err := CoerceArg(ArgDef{Name: "S", Type: ty, Value: ""}, ""); err != nil {
			t.Fatalf("%s: empty value rejected without the constraint: %v", ty, err)
		}
	}
}

// TestCoerceArgMaxLength covers the length limit CyberChef declares as
// `maxLength`, which caps a string argument.
func TestCoerceArgMaxLength(t *testing.T) {
	five := 5
	def := ArgDef{Name: "Non Empty String", Type: ArgString, Value: "hello", MaxLength: &five}

	if _, err := CoerceArg(def, "hello"); err != nil {
		t.Fatalf("a value of exactly the limit was rejected: %v", err)
	}
	if _, err := CoerceArg(def, "hi"); err != nil {
		t.Fatalf("a shorter value was rejected: %v", err)
	}
	if _, err := CoerceArg(def, "helloooo"); err == nil {
		t.Fatal("expected a value past the limit to be rejected")
	}
	if _, err := CoerceArg(ArgDef{Name: "S", Type: ArgString}, "helloooo"); err != nil {
		t.Fatalf("a long value was rejected without a limit: %v", err)
	}
}

// TestCoerceArgMaxLengthCountsAsJavaScriptDoes covers how the limit counts a
// character beyond the basic plane. JavaScript stores one as two units and
// reports a length of two, so a limit of two admits one such character and
// turns away a pair.
func TestCoerceArgMaxLengthCountsAsJavaScriptDoes(t *testing.T) {
	two := 2
	def := ArgDef{Name: "Marker", Type: ArgString, MaxLength: &two}

	for _, tc := range []struct {
		name  string
		value string
		allow bool
	}{
		{"two plain characters", "ab", true},
		{"three plain characters", "abc", false},
		{"one character from beyond the basic plane", "😀", true},
		{"two of them", "😀😀", false},
		{"one of them and a plain one", "😀a", false},
		{"a character that takes three bytes but one unit", "あ", true},
		{"two of those", "ああ", true},
		{"three of those", "あああ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CoerceArg(def, tc.value)
			if tc.allow && err != nil {
				t.Errorf("%q was turned away: %v", tc.value, err)
			}
			if !tc.allow && err == nil {
				t.Errorf("%q was accepted", tc.value)
			}
		})
	}
}

// TestCoerceArgToggleStringNonEmpty covers `allowEmpty: false` on a toggle
// string, where it is the text rather than the mode that may not be empty.
func TestCoerceArgToggleStringNonEmpty(t *testing.T) {
	def := ArgDef{
		Name: "Non Empty Toggle String", Type: ArgToggleString,
		ToggleValues: []string{"Option A", "Option B"}, NonEmpty: true,
	}

	if _, err := CoerceArg(def, ToggleString{Value: "test", Option: "Option A"}); err != nil {
		t.Fatalf("a filled value was rejected: %v", err)
	}
	if _, err := CoerceArg(def, ToggleString{Value: "", Option: "Option A"}); err == nil {
		t.Fatal("expected an empty value to be rejected")
	}

	permissive := ArgDef{Name: "Key", Type: ArgToggleString, ToggleValues: []string{"Hex"}}
	if _, err := CoerceArg(permissive, ToggleString{Value: "", Option: "Hex"}); err != nil {
		t.Fatalf("an empty value was rejected without the constraint: %v", err)
	}
}

// TestCoerceArgMessages covers the wording of every complaint the argument
// checker makes. CyberChef reports these to the user directly
// (../CyberChef/src/core/Ingredient.mjs), and its own test fixtures assert the
// text, so cchef says the same thing.
func TestCoerceArgMessages(t *testing.T) {
	five, ten := 5.0, 10.0
	oneAndAHalf, fiveAndAHalf := 1.5, 5.5
	maxFive := 5

	for _, tc := range []struct {
		name  string
		def   ArgDef
		value any
		want  string
	}{
		{
			"a number below the lowest allowed",
			ArgDef{Name: "Integer Number", Type: ArgNumber, Min: &five, Max: &ten, Integer: true},
			4.0,
			"Integer Number must be greater than or equal to 5.",
		},
		{
			"a number above the highest allowed",
			ArgDef{Name: "Integer Number", Type: ArgNumber, Min: &five, Max: &ten, Integer: true},
			11.0,
			"Integer Number must be less than or equal to 10.",
		},
		{
			"a number that is not whole",
			ArgDef{Name: "Integer Number", Type: ArgNumber, Min: &five, Max: &ten, Integer: true},
			5.5,
			"Integer Number must be an integer.",
		},
		{
			"a fractional bound below the lowest allowed",
			ArgDef{Name: "Real Number", Type: ArgNumber, Min: &oneAndAHalf, Max: &fiveAndAHalf},
			1.4,
			"Real Number must be greater than or equal to 1.5.",
		},
		{
			"a fractional bound above the highest allowed",
			ArgDef{Name: "Real Number", Type: ArgNumber, Min: &oneAndAHalf, Max: &fiveAndAHalf},
			5.6,
			"Real Number must be less than or equal to 5.5.",
		},
		{
			"something that is not a number at all",
			ArgDef{Name: "Real Number", Type: ArgNumber},
			"nope",
			"Real Number must be a number.",
		},
		{
			"a string longer than allowed",
			ArgDef{Name: "Non Empty String", Type: ArgString, MaxLength: &maxFive},
			"helloooo",
			"Non Empty String length cannot exceed 5.",
		},
		{
			"an empty string where one is needed",
			ArgDef{Name: "Non Empty String", Type: ArgString, NonEmpty: true},
			"",
			"Non Empty String cannot be empty.",
		},
		{
			"an empty toggle string where one is needed",
			ArgDef{
				Name: "Non Empty Toggle String", Type: ArgToggleString,
				ToggleValues: []string{"Option A"}, NonEmpty: true,
			},
			ToggleString{Value: "", Option: "Option A"},
			"Non Empty Toggle String cannot be empty.",
		},
		{
			"a choice that is not on offer",
			ArgDef{
				Name: "Option Ingredient", Type: ArgOption,
				Value: []string{"Option 1", "Option 2", "Option 3"},
			},
			"Option 4",
			"Option Ingredient must be one of the following: Option 1, Option 2, Option 3.",
		},
		{
			"no choice at all",
			ArgDef{
				Name: "Option Ingredient", Type: ArgOption,
				Value: []string{"Option 1", "Option 2", "Option 3"},
			},
			"",
			"Option Ingredient cannot be empty.",
		},
		{
			"a mode the toggle does not offer",
			ArgDef{Name: "Key", Type: ArgToggleString, ToggleValues: []string{"Hex", "UTF8"}},
			ToggleString{Value: "ff", Option: "Base64"},
			"Key must be one of the following: Hex, UTF8.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CoerceArg(tc.def, tc.value)
			if err == nil {
				t.Fatalf("accepted %v", tc.value)
			}
			if err.Error() != tc.want {
				t.Errorf("got  %q\nwant %q", err.Error(), tc.want)
			}
		})
	}
}

// TestCoerceArgOptionAllowsAnEmptyChoice covers an option that offers an empty
// string among its choices, where choosing it is not a mistake.
func TestCoerceArgOptionAllowsAnEmptyChoice(t *testing.T) {
	def := ArgDef{Name: "Suffix", Type: ArgOption, Value: []string{"", "kb", "mb"}}
	if _, err := CoerceArg(def, ""); err != nil {
		t.Errorf("an offered empty choice was rejected: %v", err)
	}
}

// TestCoerceArgUnknownType covers the default arm for an unrecognised ArgType.
func TestCoerceArgUnknownType(t *testing.T) {
	def := ArgDef{Name: "X", Type: ArgType("bogus")}
	if _, err := CoerceArg(def, "whatever"); err == nil {
		t.Fatal("expected error for unknown arg type")
	}
}

// TestCoerceArgsArityAndDefaults covers CoerceArgs's too-many-args guard and its
// filling of defaults for omitted trailing arguments.
func TestCoerceArgsArityAndDefaults(t *testing.T) {
	defs := []ArgDef{
		{Name: "Alphabet", Type: ArgString, Value: "abc"},
		{Name: "Amount", Type: ArgNumber, Value: 13},
	}
	// Omitting the trailing arg fills its default.
	out, err := CoerceArgs(defs, []any{"xyz"})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	if len(out) != 2 || out[0].(string) != "xyz" || out[1].(float64) != 13 {
		t.Fatalf("CoerceArgs = %v", out)
	}
	// More args than the operation defines is an error.
	if _, err := CoerceArgs(defs, []any{"a", 1.0, "extra"}); err == nil {
		t.Fatal("expected too-many-arguments error")
	}
}

// TestCoerceArgWrongTypes covers the type-mismatch errors for boolean and option
// arguments.
func TestCoerceArgWrongTypes(t *testing.T) {
	if _, err := CoerceArg(ArgDef{Name: "b", Type: ArgBoolean}, "not a bool"); err == nil {
		t.Fatal("expected an error for a non-boolean value")
	}
	if _, err := CoerceArg(ArgDef{Name: "o", Type: ArgOption, Value: []string{"x"}}, 123); err == nil {
		t.Fatal("expected an error for a non-string option value")
	}
}

// TestDefaultArgsOption covers the ArgOption branch of DefaultArgs: the default
// is the choice at DefaultIndex.
func TestDefaultArgsOption(t *testing.T) {
	got := DefaultArgs([]ArgDef{{Type: ArgOption, Value: []string{"a", "b", "c"}, DefaultIndex: 2}})
	if len(got) != 1 || got[0] != "c" {
		t.Fatalf("DefaultArgs option = %v, want [c]", got)
	}
}

// TestDefaultArgsToggleString checks that an omitted toggle-string argument
// defaults to a ToggleString pairing the declared value with the first mode, so
// coercion of the default succeeds.
func TestDefaultArgsToggleString(t *testing.T) {
	got := DefaultArgs([]ArgDef{{Type: ArgToggleString, Value: "0", ToggleValues: []string{"Hex", "Decimal"}}})
	ts, ok := got[0].(ToggleString)
	if !ok || ts.Value != "0" || ts.Option != "Hex" {
		t.Fatalf("DefaultArgs toggleString = %#v, want {Value:0 Option:Hex}", got[0])
	}
	// With no modes declared the option is empty but still a valid ToggleString.
	got = DefaultArgs([]ArgDef{{Type: ArgToggleString, Value: "x"}})
	if ts, ok := got[0].(ToggleString); !ok || ts.Value != "x" || ts.Option != "" {
		t.Fatalf("DefaultArgs toggleString (no modes) = %#v", got[0])
	}
}

// TestCoerceArgsCoerceError covers the per-argument coercion error branch of
// CoerceArgs (distinct from the too-many-arguments guard).
func TestCoerceArgsCoerceError(t *testing.T) {
	if _, err := CoerceArgs([]ArgDef{{Name: "n", Type: ArgNumber}}, []any{"not a number"}); err == nil {
		t.Fatal("expected a coercion error")
	}
}

// --- direct tests for the per-type coercers extracted from CoerceArg ---

// TestCoerceNumber documents numeric coercion and the min-bound check.
func TestCoerceNumber(t *testing.T) {
	if got, err := coerceNumber(ArgDef{Name: "n"}, 5); err != nil || got.(float64) != 5 {
		t.Fatalf("int->float: %v, %v", got, err)
	}
	lo := 10.0
	if _, err := coerceNumber(ArgDef{Name: "n", Min: &lo}, 5); err == nil {
		t.Fatal("expected below-minimum error")
	}
	if _, err := coerceNumber(ArgDef{Name: "n"}, "abc"); err == nil {
		t.Fatal("expected non-numeric error")
	}
}

// TestCoerceOption documents option validation against the allowed list.
func TestCoerceOption(t *testing.T) {
	def := ArgDef{Name: "o", Value: []string{"a", "b"}}
	if got, err := coerceOption(def, "a"); err != nil || got.(string) != "a" {
		t.Fatalf("valid: %v, %v", got, err)
	}
	if _, err := coerceOption(def, "c"); err == nil {
		t.Fatal("expected not-in-list error")
	}
	if _, err := coerceOption(def, 5); err == nil {
		t.Fatal("expected non-string error")
	}
}

// TestCoerceToggleString documents toggle-string coercion and mode validation.
func TestCoerceToggleString(t *testing.T) {
	def := ArgDef{Name: "t", ToggleValues: []string{"Hex"}}
	if _, err := coerceToggleString(def, ToggleString{Value: "x", Option: "Hex"}); err != nil {
		t.Fatalf("valid mode: %v", err)
	}
	if _, err := coerceToggleString(def, ToggleString{Value: "x", Option: "Bad"}); err == nil {
		t.Fatal("expected invalid-mode error")
	}
}
