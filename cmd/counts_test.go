package cmd

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// The operation counts quoted in PLAN.md and both READMEs are maintained by
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
	for _, tc := range []struct{ file, pattern string }{
		{"README.md", `\*\*(\d+) operations\*\* so far`},
		{"README.md", `The (\d+) operations are grouped`},
		{"docs/README.md", `\*\*Scope:\*\* (\d+) operations are currently ported`},
		{"PLAN.md", `curated set of (\d+)\noperations`},
		{"PLAN.md", `- \*\*(\d+) operations\*\* \(` + "`internal/ops/`" + `\)`},
	} {
		if got := countIn(t, tc.file, tc.pattern); got != want {
			t.Errorf("%s: quoted %d subcommands, registry has %d", tc.file, got, want)
		}
	}
}

// TestUniqueOpCountMatchesChecklist checks PLAN.md's unique-operation count
// against its own implementation checklist. It is a different number from the
// subcommand count because one CyberChef operation can back several
// subcommands.
func TestUniqueOpCountMatchesChecklist(t *testing.T) {
	plan := readRepoFile(t, "PLAN.md")
	done := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^- \[x\] (.+?)\s*$`).FindAllStringSubmatch(plan, -1) {
		done[m[1]] = true
	}
	if got := countIn(t, "PLAN.md", `\*\*(\d+) unique\*\* CyberChef operations`); got != len(done) {
		t.Errorf("PLAN.md quotes %d unique operations, its checklist has %d ticked", got, len(done))
	}
}

// TestCategoryCountsMatchChecklist checks each `### Category (done/total)`
// heading against the checkboxes beneath it.
func TestCategoryCountsMatchChecklist(t *testing.T) {
	plan := readRepoFile(t, "PLAN.md")
	heading := regexp.MustCompile(`(?m)^### (.+?) \((\d+)/(\d+)\)$`)
	sections := heading.FindAllStringSubmatchIndex(plan, -1)
	for i, loc := range sections {
		name := plan[loc[2]:loc[3]]
		quotedDone, _ := strconv.Atoi(plan[loc[4]:loc[5]])
		quotedTotal, _ := strconv.Atoi(plan[loc[6]:loc[7]])

		end := len(plan)
		if i+1 < len(sections) {
			end = sections[i+1][0]
		}
		body := plan[loc[1]:end]
		done := len(regexp.MustCompile(`(?m)^- \[x\] `).FindAllString(body, -1))
		total := done + len(regexp.MustCompile(`(?m)^- \[ \] `).FindAllString(body, -1))

		if done != quotedDone || total != quotedTotal {
			t.Errorf("PLAN.md %q heading says %d/%d, checkboxes give %d/%d",
				name, quotedDone, quotedTotal, done, total)
		}
	}
}

// TestChecklistTicksRegisteredOps ties PLAN.md's checklist to the registry
// rather than to itself. The other two checks compare PLAN.md against its own
// numbers, so a revert that undoes a heading and its checkboxes together stays
// self-consistent and slips through; this one does not, because it asks whether
// an operation cchef actually registers is recorded as done.
func TestChecklistTicksRegisteredOps(t *testing.T) {
	plan := readRepoFile(t, "PLAN.md")

	// An operation may be listed under more than one category, so collect every
	// occurrence and require them all to be ticked.
	type entry struct{ ticked, listed bool }
	checklist := map[string]*entry{}
	for _, m := range regexp.MustCompile(`(?m)^- \[([ x])\] (.+?)\s*$`).FindAllStringSubmatch(plan, -1) {
		e, ok := checklist[m[2]]
		if !ok {
			e = &entry{ticked: true}
			checklist[m[2]] = e
		}
		e.listed = true
		if m[1] != "x" {
			e.ticked = false
		}
	}

	for _, op := range core.Default.All() {
		name := op.Meta().Name
		e, ok := checklist[name]
		if !ok {
			// Some subcommands are one CyberChef operation under another name
			// (sha256 and friends are all SHA2), so an absent entry is fine.
			continue
		}
		if !e.listed || !e.ticked {
			t.Errorf("PLAN.md lists %q unticked, but cchef registers it", name)
		}
	}
}
