package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Fixtures transcribed from ../CyberChef/tests/operations/tests/MurmurHash3.mjs.
func TestMurmurHash3Fixtures(t *testing.T) {
	mmh := func(args ...any) core.Recipe {
		return core.Recipe{{Op: "MurmurHash3", Args: args}}
	}
	runCases(t, []opCase{
		{"MurmurHash3: nothing", "", "0", mmh(0)},
		{"MurmurHash3: 1", "1", "2484513939", mmh(0)},
		{"MurmurHash3: Hello World!", "Hello World!", "3691591037", mmh(0)},
		{"MurmurHash3: Hello World! with seed", "Hello World!", "1148600031", mmh(1337)},
		{"MurmurHash3: foo", "foo", "4138058784", mmh(0)},
		{"MurmurHash3: foo signed", "foo", "-156908512", mmh(0, true)},
		// A hash below 2^31 stays positive when converted to signed.
		{"MurmurHash3: seed signed positive", "Hello World!", "1148600031", mmh(1337, true)},
	})
}
