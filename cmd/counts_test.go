package cmd

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// The operation counts quoted in AGENTS.md and docs/README.md are maintained by
// hand and drift silently — nothing else fails when they go stale. These tests
// pin every one of them to the registry.

// readRepoFile reads a file from the repository root.
func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

// countIn returns the single number matched by pattern's first group in the
// named file, failing if the pattern does not match exactly once.
func countIn(t *testing.T, file, pattern string) int {
	t.Helper()
	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(readRepoFile(t, file), -1)
	if len(matches) != 1 {
		t.Fatalf("%s: pattern %q matched %d times, want exactly 1", file, pattern, len(matches))
	}
	n, err := strconv.Atoi(matches[0][1])
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	return n
}

// TestSubcommandCountsMatchRegistry checks every quoted subcommand count.
func TestSubcommandCountsMatchRegistry(t *testing.T) {
	want := len(core.Default.All())
	// README.md deliberately quotes no count: every operation is ported, so a
	// running total there would only go stale again.
	for _, tc := range []struct{ file, pattern string }{
		{"docs/README.md", `\*\*Scope:\*\* (\d+) operations, covering every CyberChef operation`},
		{"AGENTS.md", `curated set of (\d+)\s+operations`},
		{"AGENTS.md", `- \*\*(\d+) operations\*\* \(` + "`ops/`" + `\)`},
	} {
		if got := countIn(t, tc.file, tc.pattern); got != want {
			t.Errorf("%s: quoted %d subcommands, registry has %d", tc.file, got, want)
		}
	}
}

// TestUniqueOpCountMatchesRegistry checks AGENTS.md's unique-operation count. It
// is a smaller number than the subcommand count because CyberChef's SHA2 backs
// five subcommands, and the two have to move together.
func TestUniqueOpCountMatchesRegistry(t *testing.T) {
	// sha2 and sha224/sha256/sha384/sha512 are one CyberChef operation, so four
	// of the five subcommands are not a further unique operation.
	const sha2Extras = 4
	want := len(core.Default.All()) - sha2Extras
	if got := countIn(t, "AGENTS.md", `cover\s+(\d+)\s+unique\s+CyberChef\s+operations`); got != want {
		t.Errorf("AGENTS.md quotes %d unique operations, registry implies %d", got, want)
	}
}
