package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestScanOptionCaseCollisions(t *testing.T) {
	n := 0
	for _, op := range core.Default.All() {
		for _, a := range op.Args() {
			if a.Type != core.ArgOption && a.Type != core.ArgToggleString {
				continue
			}
			var vals []string
			if a.Type == core.ArgOption {
				v, ok := a.Value.([]string)
				if !ok {
					continue
				}
				vals = v
			} else {
				vals = a.ToggleValues
			}
			seen := map[string]string{}
			for _, v := range vals {
				k := strings.ToLower(v)
				if prev, dup := seen[k]; dup && prev != v {
					t.Errorf("%s / %s: %q and %q differ only by case", op.Meta().Name, a.Name, prev, v)
					n++
				}
				seen[k] = v
			}
		}
	}
	t.Logf("case collisions: %d", n)
}
