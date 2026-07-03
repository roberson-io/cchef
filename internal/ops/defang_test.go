package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestDefangIPFixtures transcribes CyberChef's DefangIP.mjs cases (IPv4, IPv6 and
// shorthand IPv6). The IPv6 matcher relies on lookahead/backreferences, ported
// verbatim via regexp2.
func TestDefangIPFixtures(t *testing.T) {
	defang := core.Recipe{{Op: "Defang IP Addresses", Args: []any{}}}
	runCases(t, []opCase{
		{"Valid IPV4", "192.168.1.1", "192[.]168[.]1[.]1", defang},
		{"Valid IPV6", "2001:0db8:85a3:0000:0000:8a2e:0370:7343",
			"2001[:]0db8[:]85a3[:]0000[:]0000[:]8a2e[:]0370[:]7343", defang},
		{"Valid IPV6 Shorthand", "2001:db8:3c4d:15::1a2f:1a2b",
			"2001[:]db8[:]3c4d[:]15[:][:]1a2f[:]1a2b", defang},
	})
}

// TestDefangURLOracle checks Defang URL against CyberChef-server output
// (v11.2.0); no upstream fixture. The default process mode defangs bare domains
// via DOMAIN_REGEX (regexp2), which "Only full URLs" leaves alone.
func TestDefangURLOracle(t *testing.T) {
	const in = "Visit http://example.com/path and also example.org today"
	runCases(t, []opCase{
		{"Valid domains and full URLs", in,
			"Visit hxxp[://]example[.]com/path and also example[.]org today",
			core.Recipe{{Op: "Defang URL", Args: []any{true, true, true, "Valid domains and full URLs"}}}},
		{"Only full URLs", in,
			"Visit hxxp[://]example[.]com/path and also example.org today",
			core.Recipe{{Op: "Defang URL", Args: []any{true, true, true, "Only full URLs"}}}},
		{"Everything", "http://example.com", "hxxp[://]example[.]com",
			core.Recipe{{Op: "Defang URL", Args: []any{true, true, true, "Everything"}}}},
	})
}

// TestFangURLOracle checks Fang URL against CyberChef-server output (v11.2.0).
func TestFangURLOracle(t *testing.T) {
	runCases(t, []opCase{
		{"restore all", "hxxp[://]example[.]com/a[.]b", "http://example.com/a.b",
			core.Recipe{{Op: "Fang URL", Args: []any{true, true, true}}}},
	})
}
