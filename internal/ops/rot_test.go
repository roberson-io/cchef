package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Rotate.mjs.
func TestRotFixtures(t *testing.T) {
	const sample = "The Quick Brown Fox Jumped Over The Lazy Dog. 0123456789"
	runCases(t, []opCase{
		{"ROT13: nothing", "", "",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 13}}}},
		{"ROT13: no shift", sample, sample,
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 0}}}},
		{"ROT13: normal", sample, "Gur Dhvpx Oebja Sbk Whzcrq Bire Gur Ynml Qbt. 3456789012",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 13}}}},
		{"ROT13: negative", sample, "Gur Dhvpx Oebja Sbk Whzcrq Bire Gur Ynml Qbt. 7890123456",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, -13}}}},

		{"ROT47: nothing", "", "",
			core.Recipe{{Op: "ROT47", Args: []any{47}}}},
		{"ROT47: normal", "The Quick Brown Fox Jumped Over The Lazy Dog.",
			"%96 \"F:4< qC@H? u@I yF>A65 ~G6C %96 {2KJ s@8]",
			core.Recipe{{Op: "ROT47", Args: []any{47}}}},
	})
}
