package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TCP/IP Checksum has no upstream operation fixtures; these vectors come from the
// CyberChef-server oracle (driven through From Hex so the input bytes match).
func TestTCPIPChecksumFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"TCP/IP Checksum: IPv4 header",
			"45 00 00 3c 1c 46 40 00 40 06 00 00 ac 10 0a 63 ac 10 0a 0c", "b1e6",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "TCP/IP Checksum", Args: []any{}}},
		},
		{
			"TCP/IP Checksum: 0102030405",
			"0102030405", "f6f9",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "TCP/IP Checksum", Args: []any{}}},
		},
	})
}

func TestTCPIPChecksumBytes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "ffff"},
		{"\x00", "ffff"},
		{"\x01\x02\x03\x04\x05", "f6f9"},
	}
	for _, c := range cases {
		got, err := runOp(t, "TCP/IP Checksum", c.in)
		if err != nil || got != c.want {
			t.Fatalf("TCP/IP Checksum(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}
