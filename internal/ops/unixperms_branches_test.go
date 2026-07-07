package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// runOp runs a single registered operation against input, returning output/error.
func runOp(t *testing.T, name, input string, args ...any) (string, error) {
	t.Helper()
	op, ok := core.Default.Get(name)
	if !ok {
		t.Fatalf("op %q not registered", name)
	}
	coerced, err := core.CoerceArgs(op.Args(), args)
	if err != nil {
		t.Fatalf("coerce args: %v", err)
	}
	out, err := op.Run(core.NewDish([]byte(input), op.Meta().InputType), coerced)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// TestParseUNIXPermsBranches exercises the textual path, every file type, and the
// special flag lines (verified against the CyberChef-server oracle).
func TestParseUNIXPermsBranches(t *testing.T) {
	cases := []struct {
		in       string
		textual  string // expected "Textual representation:" value
		octal    string // expected "Octal representation:" value
		fileType string // expected "File type:" value ("" = not present)
		flags    []string
	}{
		{"drwxr-xr-x", "drwxr-xr-x", "0755", "Directory", nil},
		{"lrwxrwxrwx", "lrwxrwxrwx", "0777", "Symbolic link", nil},
		{"prw-r--r--", "prw-r--r--", "0644", "Named pipe", nil},
		{"srwxr-xr-x", "srwxr-xr-x", "0755", "Socket", nil},
		{"crw-rw-rw-", "crw-rw-rw-", "0666", "Character device", nil},
		{"brw-rw----", "brw-rw----", "0660", "Block device", nil},
		{"Drwxr-xr-x", "Drwxr-xr-x", "0755", "Door", nil},
		{"-rw-r--r--", "-rw-r--r--", "0644", "Regular file", nil},
		// A textual perm shorter than 10 chars exercises the out-of-range byte
		// accessor (missing positions read as 0).
		{"d", "d---------", "0000", "Directory", nil},
		{
			"-rwsr-sr-t", "-rwsr-sr-t", "7755", "Regular file",
			[]string{"The setuid flag is set", "The setgid flag is set", "The sticky bit is set"},
		},
		{
			"-rwSr-Sr-T", "-rwSr-Sr-T", "7644", "Regular file",
			[]string{"The setuid flag is set", "The setgid flag is set", "The sticky bit is set"},
		},
		// Octal forms (no file-type line).
		{"1777", "-rwxrwxrwt", "1777", "", []string{"The sticky bit is set"}},
		{"4755", "-rwsr-xr-x", "4755", "", []string{"The setuid flag is set"}},
	}
	for _, c := range cases {
		out, err := runOp(t, "Parse UNIX file permissions", c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if !strings.Contains(out, "Textual representation: "+c.textual) {
			t.Errorf("%q: missing textual %q in:\n%s", c.in, c.textual, out)
		}
		if !strings.Contains(out, "Octal representation:   "+c.octal) {
			t.Errorf("%q: missing octal %q in:\n%s", c.in, c.octal, out)
		}
		if c.fileType != "" && !strings.Contains(out, "File type: "+c.fileType) {
			t.Errorf("%q: missing file type %q", c.in, c.fileType)
		}
		if c.fileType == "" && strings.Contains(out, "File type:") {
			t.Errorf("%q: unexpected file type line", c.in)
		}
		for _, f := range c.flags {
			if !strings.Contains(out, f) {
				t.Errorf("%q: missing flag line %q", c.in, f)
			}
		}
	}

	if _, err := runOp(t, "Parse UNIX file permissions", "not a perm"); err == nil {
		t.Error("expected error for invalid input")
	}
}

func TestMetricsErrors(t *testing.T) {
	// Hamming: wrong sample count, length mismatch.
	if _, err := runOp(t, "Hamming Distance", "onlyone", `\n\n`, "Byte", "Raw string"); err == nil {
		t.Error("expected error for != 2 samples")
	}
	if _, err := runOp(t, "Hamming Distance", "abc\n\nde", `\n\n`, "Byte", "Raw string"); err == nil {
		t.Error("expected error for length mismatch")
	}
	// Levenshtein: wrong count, negative cost.
	if _, err := runOp(t, "Levenshtein Distance", "one-sample", `\n`, 1, 1, 1); err == nil {
		t.Error("expected error for != 2 samples")
	}
	if _, err := runOp(t, "Levenshtein Distance", "a\nb", `\n`, -1, 1, 1); err == nil {
		t.Error("expected error for negative cost")
	}
}
