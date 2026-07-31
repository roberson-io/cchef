package ops

import "testing"

// TestCompareSSDEEPMalformed covers hashes that are not hashes. CyberChef
// reports a TypeError for these; cchef reports an error of its own rather
// than indexing off the end of an empty string.
func TestCompareSSDEEPMalformed(t *testing.T) {
	// A hash without the field the comparison needs cannot be scored.
	for _, in := range []string{"\n", "a\nb", "\n\n"} {
		if _, err := runOp(t, "Compare SSDEEP hashes", in, "Line feed"); err == nil {
			t.Errorf("expected an error for %q", in)
		}
	}
	// A hash with the field but nothing in it scores NaN, which is what
	// CyberChef reports too (as null, JSON having no NaN).
	for _, in := range []string{":\n:", "3:\n3:"} {
		got, err := runOp(t, "Compare SSDEEP hashes", in, "Line feed")
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got != "NaN" {
			t.Errorf("%q = %q, want NaN", in, got)
		}
	}
	// Hashes far enough apart in block size score zero without needing their
	// remaining parts, which is what CyberChef answers here too.
	got, err := runOp(t, "Compare SSDEEP hashes", "x\n", "Line feed")
	if err != nil {
		t.Fatalf("x vs empty: %v", err)
	}
	if got != "0" {
		t.Errorf("x vs empty = %q, want 0", got)
	}
	// A well-formed pair still works.
	got, err = runOp(t, "Compare SSDEEP hashes", "3:abc:def\n3:abc:def", "Line feed")
	if err != nil {
		t.Fatalf("identical hashes: %v", err)
	}
	if got != "100" {
		t.Errorf("identical hashes = %q, want 100", got)
	}
}
