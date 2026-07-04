package cmd

import (
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	got := execRoot(t, "--version")
	if !strings.Contains(got, version) {
		t.Fatalf("--version output %q does not contain version %q", got, version)
	}
}
