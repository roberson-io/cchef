package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseEthernetFrameFixtures transcribes CyberChef's ParseEthernetFrame.mjs
// cases (plain, one/two VLAN tags, HTML-escaped packet data) and, for full
// coverage, adds oracle-derived cases for the "Packet data" and "Packet data
// (hex)" return types.
func TestParseEthernetFrameFixtures(t *testing.T) {
	text := func() []any { return []any{"Hex", "Text output"} }
	runCases(t, []opCase{
		{"plain", "000000000000ffffffffffff08004500",
			"Source MAC: ff:ff:ff:ff:ff:ff\nDestination MAC: 00:00:00:00:00:00\nData:\n45 00",
			core.Recipe{{Op: "Parse Ethernet frame", Args: text()}}},
		{"one VLAN tag", "01000ccdcdd00013c3dfae188100a0760165aaaa",
			"Source MAC: 00:13:c3:df:ae:18\nDestination MAC: 01:00:0c:cd:cd:d0\nVLAN: 118\nData:\naa aa",
			core.Recipe{{Op: "Parse Ethernet frame", Args: text()}}},
		{"two VLAN tags", "0019aa7de688002155c8f13c810000d18100001408004500",
			"Source MAC: 00:21:55:c8:f1:3c\nDestination MAC: 00:19:aa:7d:e6:88\nVLAN: 209, 20\nData:\n45 00",
			core.Recipe{{Op: "Parse Ethernet frame", Args: text()}}},
		{"packet data escapes HTML", "000000000000ffffffffffff08003c696d67207372633d78206f6e6572726f723d616c6572742831293e3c7363726970743e616c6572742832293c2f7363726970743e",
			"&lt;img src=x onerror=alert(1)&gt;&lt;script&gt;alert(2)&lt;/script&gt;",
			core.Recipe{{Op: "Parse Ethernet frame", Args: []any{"Hex", "Packet data"}}}},
		{"packet data (hex)", "000000000000ffffffffffff08004500", "45 00",
			core.Recipe{{Op: "Parse Ethernet frame", Args: []any{"Hex", "Packet data (hex)"}}}},
	})
}
