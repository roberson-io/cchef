package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	got := execRoot(t, "--version")
	if !strings.Contains(got, version) {
		t.Fatalf("--version output %q does not contain version %q", got, version)
	}
}

// TestVersionFlagNamesAlignedCyberChef covers that --version reports which
// CyberChef release the operations are aligned with, so a user can tell what
// parity to expect without reading the docs.
func TestVersionFlagNamesAlignedCyberChef(t *testing.T) {
	got := execRoot(t, "--version")
	if !strings.Contains(got, "CyberChef "+alignedCyberChef) {
		t.Fatalf("--version output %q does not name CyberChef %s", got, alignedCyberChef)
	}
}

// TestAlignedCyberChefQuotedConsistently pins every place that spells out the
// aligned CyberChef version to the constant, so the docs cannot drift from
// what the build reports.
func TestAlignedCyberChefQuotedConsistently(t *testing.T) {
	for _, tc := range []struct{ file, pattern string }{
		{"AGENTS.md", `tracked\s+against CyberChef (\d+\.\d+\.\d+)`},
		{"CHANGELOG.md", `aligned with CyberChef v(\d+\.\d+\.\d+)`},
	} {
		matches := regexp.MustCompile(tc.pattern).FindAllStringSubmatch(readRepoFile(t, tc.file), -1)
		if len(matches) == 0 {
			t.Errorf("%s: no match for %q", tc.file, tc.pattern)
			continue
		}
		for _, m := range matches {
			if m[1] != alignedCyberChef {
				t.Errorf("%s quotes CyberChef %s; cmd/version.go says %s", tc.file, m[1], alignedCyberChef)
			}
		}
	}
}
