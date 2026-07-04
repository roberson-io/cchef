package cmd

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestOpAliasesValid checks that every alias names a registered operation, that
// aliases are globally unique, and that none collides with a canonical
// subcommand name (which would make the alias ambiguous).
func TestOpAliasesValid(t *testing.T) {
	registered := map[string]bool{}
	subcommands := map[string]bool{}
	for _, op := range core.Default.All() {
		registered[op.Meta().Name] = true
		subcommands[core.Kebab(op.Meta().Name)] = true
	}

	seen := map[string]string{} // alias -> owning op
	for name, aliases := range opAliases {
		if !registered[name] {
			t.Errorf("opAliases has entry for unregistered operation %q", name)
		}
		for _, a := range aliases {
			if subcommands[a] {
				t.Errorf("alias %q (for %q) collides with a canonical subcommand name", a, name)
			}
			if prev, dup := seen[a]; dup {
				t.Errorf("alias %q is used by both %q and %q", a, prev, name)
			}
			seen[a] = name
		}
	}
}

func TestAliasResolvesToOperation(t *testing.T) {
	// b64e is the alias for To Base64: invoking it must be equivalent to
	// invoking the canonical subcommand. (Compared directly rather than against
	// a literal to stay robust to cobra's cross-Execute flag state in-process.)
	viaAlias := execRoot(t, "b64e", "hello")
	viaName := execRoot(t, "to-base64", "hello")
	if viaAlias != viaName {
		t.Fatalf("alias b64e = %q, canonical to-base64 = %q", viaAlias, viaName)
	}
}
