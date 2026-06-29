package ops

import (
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
		{"octal 755", "755", want755,
			core.Recipe{{Op: "Parse UNIX file permissions"}}},
	})
}
