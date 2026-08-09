package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	got := execRoot(t, "--version")
	if !strings.Contains(got, buildVersion) {
		t.Fatalf("--version output %q does not contain version %q", got, buildVersion)
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

// TestResolveVersion covers where the reported version comes from. A release
// binary is stamped by the linker; a `go install` binary has no stamp but the
// toolchain records the module version it was built from; a local build has
// neither and says so.
func TestResolveVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		ldflag, module, want string
	}{
		"release stamp wins":     {"1.2.3", "v9.9.9", "1.2.3"},
		"stamp without a module": {"1.2.3", "", "1.2.3"},
		"module version":         {"", "v1.0.0", "1.0.0"},
		"module pseudo-version":  {"", "v1.0.1-0.20260809100557-a68f68f08e55", "1.0.1-0.20260809100557-a68f68f08e55"},
		"local build":            {"", "(devel)", devVersion},
		"no information at all":  {"", "", devVersion},
	} {
		if got := resolveVersion(tc.ldflag, tc.module); got != tc.want {
			t.Errorf("%s: resolveVersion(%q, %q) = %q, want %q", name, tc.ldflag, tc.module, got, tc.want)
		}
	}
}

// TestBuildVersionIsReported checks that whatever resolveVersion decided is what
// --version actually prints, so the two cannot drift apart.
func TestBuildVersionIsReported(t *testing.T) {
	got := execRoot(t, "--version")
	if !strings.Contains(got, buildVersion) {
		t.Errorf("--version output %q does not contain %q", got, buildVersion)
	}
	if strings.Contains(got, "0.1.0-dev") {
		t.Errorf("--version still reports the old placeholder: %q", got)
	}
}
