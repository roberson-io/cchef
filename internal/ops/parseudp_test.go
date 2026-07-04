package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseUDPFixtures transcribes CyberChef's ParseUDP.mjs cases. The op returns
// a JSON object; cchef emits it as compact JSON (the form CyberChef's fixtures
// obtain via a trailing JSON Minify).
func TestParseUDPFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"no data", "04 89 00 35 00 2c 01 01",
			`{"Source port":1161,"Destination port":53,"Length":44,"Checksum":"0x0101"}`,
			core.Recipe{{Op: "Parse UDP", Args: []any{"Hex"}}},
		},
		{
			"with data", "04 89 00 35 00 2c 01 01 02 02",
			`{"Source port":1161,"Destination port":53,"Length":44,"Checksum":"0x0101","Data":"0x0202"}`,
			core.Recipe{{Op: "Parse UDP", Args: []any{"Hex"}}},
		},
	})
}

// TestParseUDPRaw covers the "Raw" input-format path (bytes passed through
// directly rather than decoded from hex).
func TestParseUDPRaw(t *testing.T) {
	runCases(t, []opCase{
		{
			"raw bytes", "\x04\x89\x00\x35\x00\x2c\x01\x01",
			`{"Source port":1161,"Destination port":53,"Length":44,"Checksum":"0x0101"}`,
			core.Recipe{{Op: "Parse UDP", Args: []any{"Raw"}}},
		},
	})
}

// TestParseUDPError covers the too-short input case.
func TestParseUDPError(t *testing.T) {
	short := core.Recipe{{Op: "Parse UDP", Args: []any{"Hex"}}}
	if _, err := short.Execute(core.NewDish([]byte("04 89 00"), core.TypeString)); err == nil ||
		!strings.Contains(err.Error(), "need 8 bytes for a UDP header") {
		t.Fatalf("got err %v, want UDP length error", err)
	}
}
