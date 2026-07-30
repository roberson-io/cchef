package yara

import "testing"

// TestCompileErrorText pins how a fault in the rules is written, which is the
// form CyberChef reports and so the form the operation has to produce.
func TestCompileErrorText(t *testing.T) {
	err := &compileError{line: 7, msg: `undefined string "$nope"`}
	if got, want := err.Error(), `Error on line 7: undefined string "$nope"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if err.Line() != 7 {
		t.Errorf("line %d, want 7", err.Line())
	}
	if got, want := err.Message(), `undefined string "$nope"`; got != want {
		t.Errorf("message %q, want %q", got, want)
	}
}

// TestWarningText pins the other half of the same output.
func TestWarningText(t *testing.T) {
	w := Warning{Line: 5, Message: `string "$s" may slow down scanning`}
	want := `Warning on line 5: string "$s" may slow down scanning`
	if got := w.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
