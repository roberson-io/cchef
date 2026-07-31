package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestListJSON checks the machine-readable form: one object per subcommand,
// carrying what a completion script or a wrapper needs.
func TestListJSON(t *testing.T) {
	out := execRoot(t, "list", "--json")
	var got []listEntry
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json did not produce JSON: %v\n%s", err, out)
	}
	if len(got) != len(core.Default.All()) {
		t.Errorf("listed %d operations, registry has %d", len(got), len(core.Default.All()))
	}
	byCommand := map[string]listEntry{}
	for _, e := range got {
		byCommand[e.Command] = e
	}
	e, ok := byCommand["to-base64"]
	if !ok {
		t.Fatal("to-base64 is missing")
	}
	if e.Name != "To Base64" {
		t.Errorf("Name = %q, want %q", e.Name, "To Base64")
	}
	if e.Summary == "" {
		t.Error("Summary is empty")
	}
	if len(e.Categories) == 0 {
		t.Error("Categories is empty")
	}
	// An operation filed under two categories keeps both.
	if u, ok := byCommand["url-decode"]; ok && len(u.Categories) < 2 {
		t.Errorf("url-decode has categories %v, want more than one", u.Categories)
	}
}

// TestListDefaultIsHuman checks the default stays the grouped, readable form.
func TestListDefaultIsHuman(t *testing.T) {
	out := execRoot(t, "list")
	if strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Error("the default listing should not be JSON")
	}
	if !strings.Contains(out, "to-base64") {
		t.Error("the default listing should name subcommands")
	}
}
