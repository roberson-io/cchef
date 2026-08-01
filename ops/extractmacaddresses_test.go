package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// macRecipe builds a recipe for the operation with the arguments given.
func macRecipe(total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract MAC addresses",
		Args: []any{total, sorted, unique},
	}}
}

// TestExtractMACAddresses covers the switches and the shape of an address, each
// expectation taken from the CyberChef-server oracle.
func TestExtractMACAddresses(t *testing.T) {
	runCases(t, []opCase{
		{
			"plain",
			"00:1B:44:11:3A:B7 and 00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f\nzz:zz:zz:zz:zz:zz not a mac 01:23:45:67:89\ndeadbeefcafe 00:1B:44:11:3A:B7\n",
			"00:1B:44:11:3A:B7\n00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff\n0a:0b:0c:0d:0e:0f\n00:1B:44:11:3A:B7",
			macRecipe(false, false, false),
		},
		{
			"display total",
			"00:1B:44:11:3A:B7 and 00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f\nzz:zz:zz:zz:zz:zz not a mac 01:23:45:67:89\ndeadbeefcafe 00:1B:44:11:3A:B7\n",
			"Total found: 5\n\n00:1B:44:11:3A:B7\n00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff\n0a:0b:0c:0d:0e:0f\n00:1B:44:11:3A:B7",
			macRecipe(true, false, false),
		},
		{
			"sorted by value, not by text",
			"00:1B:44:11:3A:B7 and 00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f\nzz:zz:zz:zz:zz:zz not a mac 01:23:45:67:89\ndeadbeefcafe 00:1B:44:11:3A:B7\n",
			"00-1b-44-11-3a-b8\n00:1B:44:11:3A:B7\n00:1B:44:11:3A:B7\n0a:0b:0c:0d:0e:0f\nff:ff:ff:ff:ff:ff",
			macRecipe(false, true, false),
		},
		{
			"unique",
			"00:1B:44:11:3A:B7 and 00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f\nzz:zz:zz:zz:zz:zz not a mac 01:23:45:67:89\ndeadbeefcafe 00:1B:44:11:3A:B7\n",
			"00:1B:44:11:3A:B7\n00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff\n0a:0b:0c:0d:0e:0f",
			macRecipe(false, false, true),
		},
		{
			"everything on",
			"00:1B:44:11:3A:B7 and 00-1b-44-11-3a-b8\nff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f\nzz:zz:zz:zz:zz:zz not a mac 01:23:45:67:89\ndeadbeefcafe 00:1B:44:11:3A:B7\n",
			"Total found: 4\n\n00-1b-44-11-3a-b8\n00:1B:44:11:3A:B7\n0a:0b:0c:0d:0e:0f\nff:ff:ff:ff:ff:ff",
			macRecipe(true, true, true),
		},
		{
			"nothing found",
			"no addresses 01:23:45:67:89 gg:hh:ii:jj:kk:ll",
			"",
			macRecipe(false, false, false),
		},
		{
			"nothing found with total",
			"no addresses 01:23:45:67:89 gg:hh:ii:jj:kk:ll",
			"Total found: 0\n\n",
			macRecipe(true, false, false),
		},
	})
}
