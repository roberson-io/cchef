package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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
