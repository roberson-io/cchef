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
