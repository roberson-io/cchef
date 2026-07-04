package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseIPv4HeaderFixtures transcribes CyberChef's ParseIPv4Header.mjs
// "Data (raw)" fixtures and, per full-coverage discipline, adds oracle-derived
// cases for the other two output modes ("Table" HTML and "Data (hex)").
func TestParseIPv4HeaderFixtures(t *testing.T) {
	const hdr = "45 00 00 3c 1c 46 40 00 40 06 b1 e6 c0 a8 00 01 c0 a8 00 02 3c 73 63 72 69 70 74 3e"
	const tableOut = `<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Field</th><th>Value</th></tr>
<tr><td>Version</td><td>4</td></tr>
<tr><td>Internet Header Length (IHL)</td><td>5 (20 bytes)</td></tr>
<tr><td>Differentiated Services Code Point (DSCP)</td><td>0</td></tr>
<tr><td>Explicit Congestion Notification (ECN)</td><td>0</td></tr>
<tr><td>Total length</td><td>60 bytes
  IP header: 20 bytes
  Data: 40 bytes</td></tr>
<tr><td>Identification</td><td>0x1c46 (7238)</td></tr>
<tr><td>Flags</td><td>0x02
  Reserved bit:0 (must be 0)
  Don't fragment:1
  More fragments:0</td></tr>
<tr><td>Fragment offset</td><td>0</td></tr>
<tr><td>Time-To-Live</td><td>64</td></tr>
<tr><td>Protocol</td><td>6, Transmission Control (TCP)</td></tr>
<tr><td>Header checksum</td><td>b1e6 (incorrect, should be 9d22)</td></tr>
<tr><td>Source IP address</td><td>192.168.0.1</td></tr>
<tr><td>Destination IP address</td><td>192.168.0.2</td></tr>
<tr><td>Data (hex)</td><td>3c 73 63 72 69 70 74 3e</td></tr></table>`

	runCases(t, []opCase{
		{
			"Data (raw)", "45 00 00 3c 1c 46 40 00 40 06 b1 e6 c0 a8 00 01 c0 a8 00 02 3c 73 63 72 69 70 74 3e 61 6c 65 72 74 28 31 33 33 37 29 3c 2f 73 63 72 69 70 74 3e",
			"&lt;script&gt;alert(1337)&lt;/script&gt;",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Data (raw)"}}},
		},
		{
			"truncated raw", "\x45\x00\x00\x14\x00\x00\x00\x00\x40\x06\x00\x00", "",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Raw", "Data (raw)"}}},
		},
		{
			"Data (hex)", hdr, "3c 73 63 72 69 70 74 3e",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Data (hex)"}}},
		},
		{
			"Table", hdr, tableOut,
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Table"}}},
		},
	})
}
