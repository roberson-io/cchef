package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Expected digests transcribed from CyberChef tests/operations/tests/Hash.mjs
// (input "Hello, World!").
func TestSHA3Fixtures(t *testing.T) {
	const in = "Hello, World!"
	runCases(t, []opCase{
		{
			"SHA3 224", in, "853048fb8b11462b6100385633c0cc8dcdc6e2b8e376c28102bc84f2",
			core.Recipe{{Op: "SHA3", Args: []any{"224"}}},
		},
		{
			"SHA3 256", in, "1af17a664e3fa8e419b8ba05c2a173169df76162a5a286e0c405b460d478f7ef",
			core.Recipe{{Op: "SHA3", Args: []any{"256"}}},
		},
		{
			"SHA3 384", in, "aa9ad8a49f31d2ddcabbb7010a1566417cff803fef50eba239558826f872e468c5743e7f026b0a8e5b2d7a1cc465cdbe",
			core.Recipe{{Op: "SHA3", Args: []any{"384"}}},
		},
		{
			"SHA3 512", in, "38e05c33d7b067127f217d8c856e554fcff09c9320b8a5979ce2ff5d95dd27ba35d1fba50c562dfd1d6cc48bc9c5baa4390894418cc942d968f97bcb659419ed",
			core.Recipe{{Op: "SHA3", Args: []any{"512"}}},
		},
		// Default size is 512.
		{
			"SHA3 default", in, "38e05c33d7b067127f217d8c856e554fcff09c9320b8a5979ce2ff5d95dd27ba35d1fba50c562dfd1d6cc48bc9c5baa4390894418cc942d968f97bcb659419ed",
			core.Recipe{{Op: "SHA3"}},
		},
	})
}
