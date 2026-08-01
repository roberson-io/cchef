package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// ipRecipe builds a recipe for the operation with the arguments given.
func ipRecipe(v4, v6, removeLocal, total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract IP addresses",
		Args: []any{v4, v6, removeLocal, total, sorted, unique},
	}}
}

// TestExtractIPAddressesFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/ExtractIPAddresses.mjs). Each runs with
// IPv4 alone, which is what the fixtures use.
func TestExtractIPAddressesFixtures(t *testing.T) {
	v4 := ipRecipe(true, false, false, false, false, false)
	runCases(t, []opCase{
		{"ExtractIPAddress All Zeros", "0.0.0.0", "0.0.0.0", v4},
		{"ExtractIPAddress All 10s", "10.10.10.10", "10.10.10.10", v4},
		{"ExtractIPAddress All 100s", "100.100.100.100", "100.100.100.100", v4},
		{"ExtractIPAddress 255s", "255.255.255.255", "255.255.255.255", v4},
		{
			"ExtractIPAddress double digits",
			"10.10.10.10 25.25.25.25 99.99.99.99",
			"10.10.10.10\n25.25.25.25\n99.99.99.99", v4,
		},
		{"ExtractIPAddress 256 in middle", "255.256.255.255 255.255.256.255", "", v4},
		{"ExtractIPAddress 256 at each end", "256.255.255.255 255.255.255.256", "", v4},
		{"ExtractIPAddress silly example", "710.65.0.456", "", v4},
		{"ExtractIPAddress longer dotted decimal", "1.2.3.4.5.6.7.8", "1.2.3.4\n5.6.7.8", v4},
		{
			"ExtractIPAddress octal valid",
			"01.01.01.01 0123.0123.0123.0123 0377.0377.0377.0377",
			"01.01.01.01\n0123.0123.0123.0123\n0377.0377.0377.0377", v4,
		},
		{"ExtractIPAddress octal invalid", "0378.01.01.01 03.0377.2.3", "", v4},
	})
}

// TestExtractIPAddressesOptions covers the switches, each against the
// CyberChef-server oracle.
func TestExtractIPAddressesOptions(t *testing.T) {
	const mixed = "8.8.8.8 10.0.0.1 192.168.1.1 172.16.0.1 127.0.0.1 1.1.1.1 8.8.8.8"

	for _, tc := range []struct {
		name   string
		input  string
		recipe core.Recipe
		want   string
	}{
		{
			"local addresses removed", mixed,
			ipRecipe(true, false, true, false, false, false),
			"8.8.8.8\n1.1.1.1\n8.8.8.8",
		},
		{
			"a total before them", "1.1.1.1 2.2.2.2",
			ipRecipe(true, false, false, true, false, false),
			"Total found: 2\n\n1.1.1.1\n2.2.2.2",
		},
		{
			"sorted by value rather than as text", "10.0.0.1 9.0.0.1 100.0.0.1",
			ipRecipe(true, false, false, false, true, false),
			"9.0.0.1\n10.0.0.1\n100.0.0.1",
		},
		{
			"one of each", "1.1.1.1 2.2.2.2 1.1.1.1",
			ipRecipe(true, false, false, false, false, true),
			"1.1.1.1\n2.2.2.2",
		},
		{
			"sorted and uniqued with a total", mixed,
			ipRecipe(true, false, false, true, true, true),
			"Total found: 6\n\n1.1.1.1\n8.8.8.8\n10.0.0.1\n127.0.0.1\n172.16.0.1\n192.168.1.1",
		},
		{
			"neither version asked for", "1.1.1.1",
			ipRecipe(false, false, false, false, false, false),
			"",
		},
		{
			"a total when nothing was found", "no addresses here",
			ipRecipe(true, false, false, true, false, false),
			"Total found: 0\n\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCases(t, []opCase{{tc.name, tc.input, tc.want, tc.recipe}})
		})
	}
}

// TestExtractIPv6 covers the other version, whose shape allows the run of zero
// groups to be written once in several places.
func TestExtractIPv6(t *testing.T) {
	v6 := ipRecipe(false, true, false, false, false, false)
	for _, tc := range []struct{ name, input, want string }{
		{"a full address", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"shortened in the middle", "2001:db8:85a3::8a2e:370:7334", "2001:db8:85a3::8a2e:370:7334"},
		{"the loopback address", "::1", "::1"},
		{"all zeros", "::", "::"},
		{"one on its own", "fe80::1", "fe80::1"},
		// The check that an address shortens its run of zeros only once looks
		// ahead through the whole of the rest of the input rather than just the
		// address, so a second shortened address further along stops the first
		// from matching. This is CyberChef's behaviour, confirmed against the
		// oracle, and the reason only the last of the two is found.
		{"one shortened address after another", "fe80::1 and ::1", "::1"},
		{"nothing to find", "no addresses here", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runCases(t, []opCase{{tc.name, tc.input, tc.want, v6}})
		})
	}
}

// TestExtractIPBothVersions covers asking for both at once, which is one pattern
// rather than two passes.
func TestExtractIPBothVersions(t *testing.T) {
	runCases(t, []opCase{{
		"both versions",
		"8.8.8.8 and 2001:db8::1 and 1.1.1.1",
		"8.8.8.8\n2001:db8::1\n1.1.1.1",
		ipRecipe(true, true, false, false, false, false),
	}})
}
