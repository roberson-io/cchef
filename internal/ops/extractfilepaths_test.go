package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// filePathRecipe builds a recipe for the operation with the arguments given.
func filePathRecipe(windows, unix, total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract file paths",
		Args: []any{windows, unix, total, sorted, unique},
	}}
}

// TestExtractFilePaths covers the switches and the shape of a path, each
// expectation taken from the CyberChef-server oracle.
func TestExtractFilePaths(t *testing.T) {
	runCases(t, []opCase{
		{
			"both",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar\n/usr/local/bin/python3\n/etc/passwd\n/path/no/slash\n/a/b.c/d-e.f",
			filePathRecipe(true, true, false, false, false),
		},
		{
			"windows only",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar",
			filePathRecipe(true, false, false, false, false),
		},
		{
			"unix only",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"/usr/local/bin/python3\n/etc/passwd\n/path/no/slash\n/a/b.c/d-e.f",
			filePathRecipe(false, true, false, false, false),
		},
		{
			"neither",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"",
			filePathRecipe(false, false, false, false, false),
		},
		{
			"neither, and the total is not shown either",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"",
			filePathRecipe(false, false, true, false, false),
		},
		{
			"everything on",
			"C:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar.gz\n/usr/local/bin/python3 and /etc/passwd\nrelative/path/no/slash\n\\\\server\\share\\file.txt\n/a/b.c/d-e.f\n",
			"Total found: 6\n\n/a/b.c/d-e.f\n/etc/passwd\n/path/no/slash\n/usr/local/bin/python3\nC:\\Windows\\System32\\drivers\\etc\\hosts\nD:\\Program Files (x86)\\My App~1\\data.tar",
			filePathRecipe(true, true, true, true, true),
		},
		{
			"unique",
			"/etc/passwd /etc/passwd /Etc/passwd C:\\a\\b",
			"/etc/passwd\n/Etc/passwd\nC:\\a\\b",
			filePathRecipe(true, true, false, false, true),
		},
		{
			"sorted",
			"/etc/passwd /etc/passwd /Etc/passwd C:\\a\\b",
			"/etc/passwd\n/etc/passwd\n/Etc/passwd\nC:\\a\\b",
			filePathRecipe(true, true, false, true, false),
		},
	})
}
