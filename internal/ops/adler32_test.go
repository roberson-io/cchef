package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Adler-32 has no dedicated CyberChef fixture; these are spec-authoritative
// vectors (the canonical "Wikipedia" example, and the empty-input value of 1).
func TestAdler32(t *testing.T) {
	runCases(t, []opCase{
		{
			"Adler-32: Wikipedia", "Wikipedia", "11e60398",
			core.Recipe{{Op: "Adler-32 Checksum"}},
		},
		{
			"Adler-32: empty", "", "00000001",
			core.Recipe{{Op: "Adler-32 Checksum"}},
		},
	})
}
