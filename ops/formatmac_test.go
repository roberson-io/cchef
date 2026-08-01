package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestFormatMACAddressesOracle checks Format MAC addresses against
// CyberChef-server output (v11.2.0); there is no upstream fixture. Args are
// [Output case, No delimiter, Dash, Colon, Cisco style, IPv6 interface ID].
func TestFormatMACAddressesOracle(t *testing.T) {
	runCases(t, []opCase{
		{
			"default (both, no/dash/colon)", "01:02:03:04:05:06",
			"010203040506\n010203040506\n01-02-03-04-05-06\n01-02-03-04-05-06\n01:02:03:04:05:06\n01:02:03:04:05:06\n",
			core.Recipe{{Op: "Format MAC addresses", Args: []any{"Both", true, true, true, false, false}}},
		},
		{
			"cisco + ipv6, upper only", "01:02:03:04:05:06",
			"0102.0304.0506\n0302:03FF:FE04:0506\n",
			core.Recipe{{Op: "Format MAC addresses", Args: []any{"Upper only", false, false, false, true, true}}},
		},
		{
			"lower only, from dashed uppercase", "AA-BB-CC-DD-EE-FF",
			"aabbccddeeff\naa:bb:cc:dd:ee:ff\n",
			core.Recipe{{Op: "Format MAC addresses", Args: []any{"Lower only", true, false, true, false, false}}},
		},
		// A short (sub-3-byte) input still gets the fffe interface-ID insertion,
		// matching CyberChef's unguarded slice; exercises macInsertEvery's
		// no-op early return for groups shorter than the split width.
		{
			"short input keeps fffe insertion", "ab",
			"ab\nAB\nab\nAB\nab\nAB\nab\nAB\na9ff:fe\nA9FF:FE\n",
			core.Recipe{{Op: "Format MAC addresses", Args: []any{"Both", true, true, true, true, true}}},
		},
		// Empty input returns an empty string.
		{
			"empty input", "", "",
			core.Recipe{{Op: "Format MAC addresses", Args: []any{"Both", true, true, true, false, false}}},
		},
	})
}
