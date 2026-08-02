package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Parse UNIX file permissions output verified byte-for-byte against the
// CyberChef-server oracle.
func TestParseUNIXFilePermissions(t *testing.T) {
	want755 := "Textual representation: -rwxr-xr-x\n" +
		"Octal representation:   0755\n" +
		"\n" +
		" +---------+-------+-------+-------+\n" +
		" |         | User  | Group | Other |\n" +
		" +---------+-------+-------+-------+\n" +
		" |    Read |   X   |   X   |   X   |\n" +
		" +---------+-------+-------+-------+\n" +
		" |   Write |   X   |       |       |\n" +
		" +---------+-------+-------+-------+\n" +
		" | Execute |   X   |   X   |   X   |\n" +
		" +---------+-------+-------+-------+"

	runCases(t, []opCase{
		{
			"octal 755", "755", want755,
			core.Recipe{{Op: "Parse UNIX file permissions"}},
		},
	})
}

// --- direct tests for the helpers extracted from ParseUNIXFilePermissions.Run ---

// TestParseOctalPerms documents octal permission parsing.
func TestParseOctalPerms(t *testing.T) {
	if got := parseOctalPerms("755"); got != (unixPerms{ru: true, wu: true, eu: true, rg: true, eg: true, ro: true, eo: true}) {
		t.Fatalf("755: %+v", got)
	}
	// A leading 4th digit carries the setuid bit.
	if !parseOctalPerms("4755").su {
		t.Fatal("4755 should set setuid")
	}
}

// TestParseTextualPerms documents textual (rwx) permission parsing, including
// the setuid 's' marker in the user-execute slot.
func TestParseTextualPerms(t *testing.T) {
	if got := parseTextualPerms("drwxr-xr-x"); got != (unixPerms{d: true, ru: true, wu: true, eu: true, rg: true, eg: true, ro: true, eo: true}) {
		t.Fatalf("drwxr-xr-x: %+v", got)
	}
	if p := parseTextualPerms("-rwsr-xr-x"); !p.su || !p.eu {
		t.Fatal("'s' in user-execute should set setuid and execute")
	}
}

// TestDescribePerms documents the report: octal/textual lines always, file type
// only for textual input, and flag notes when set.
func TestDescribePerms(t *testing.T) {
	out := describePerms(parseOctalPerms("4755"), false)
	if !strings.Contains(out, "Octal representation:") || !strings.Contains(out, "The setuid flag is set") {
		t.Fatalf("octal report missing pieces: %q", out)
	}
	if strings.Contains(out, "File type:") {
		t.Fatal("file type should be absent for octal input")
	}
	if !strings.Contains(describePerms(parseTextualPerms("drwxr-xr-x"), true), "File type:") {
		t.Fatal("file type missing for textual input")
	}
}

// TestApplyPermTypeChar documents the file-type character mapping.
func TestApplyPermTypeChar(t *testing.T) {
	cases := map[byte]func(unixPerms) bool{
		'd': func(p unixPerms) bool { return p.d },
		'l': func(p unixPerms) bool { return p.sl },
		'p': func(p unixPerms) bool { return p.np },
		'b': func(p unixPerms) bool { return p.bd },
		'D': func(p unixPerms) bool { return p.dr },
	}
	for c, isSet := range cases {
		var p unixPerms
		applyPermTypeChar(&p, c)
		if !isSet(p) {
			t.Fatalf("type char %q did not set its field", c)
		}
	}
	// A regular-file '-' sets no type bit.
	var p unixPerms
	applyPermTypeChar(&p, '-')
	if p != (unixPerms{}) {
		t.Fatalf("'-' set a type bit: %+v", p)
	}
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
