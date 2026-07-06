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
