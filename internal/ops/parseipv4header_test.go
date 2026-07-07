package ops

import (
	"bytes"
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
		// Version nibble != 4 appends the version-error annotation (oracle-verified;
		// the checksum differs because byte 0 changed).
		{
			"Table version error", "55 00 00 3c 1c 46 40 00 40 06 b1 e6 c0 a8 00 01 c0 a8 00 02",
			"<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Field</th><th>Value</th></tr>\n" +
				"<tr><td>Version</td><td>5 (Error: for IPv4 headers, this should always be set to 4)</td></tr>\n" +
				"<tr><td>Internet Header Length (IHL)</td><td>5 (20 bytes)</td></tr>\n" +
				"<tr><td>Differentiated Services Code Point (DSCP)</td><td>0</td></tr>\n" +
				"<tr><td>Explicit Congestion Notification (ECN)</td><td>0</td></tr>\n" +
				"<tr><td>Total length</td><td>60 bytes\n  IP header: 20 bytes\n  Data: 40 bytes</td></tr>\n" +
				"<tr><td>Identification</td><td>0x1c46 (7238)</td></tr>\n" +
				"<tr><td>Flags</td><td>0x02\n  Reserved bit:0 (must be 0)\n  Don't fragment:1\n  More fragments:0</td></tr>\n" +
				"<tr><td>Fragment offset</td><td>0</td></tr>\n" +
				"<tr><td>Time-To-Live</td><td>64</td></tr>\n" +
				"<tr><td>Protocol</td><td>6, Transmission Control (TCP)</td></tr>\n" +
				"<tr><td>Header checksum</td><td>b1e6 (incorrect, should be 8d22)</td></tr>\n" +
				"<tr><td>Source IP address</td><td>192.168.0.1</td></tr>\n" +
				"<tr><td>Destination IP address</td><td>192.168.0.2</td></tr>\n" +
				"<tr><td>Data (hex)</td><td></td></tr></table>",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Table"}}},
		},
		// IHL < 5 appends the IHL-error annotation. cchef keeps ihl numeric, so the
		// byte counts stay correct (16 bytes); CyberChef reassigns ihl to a string
		// and its later `ihl * 4` yields "NaN bytes" — a JS coercion bug this port
		// deliberately does not reproduce. Only the annotation itself is oracle-derived.
		{
			"Table IHL error", "44 00 00 3c 1c 46 40 00 40 06 b1 e6 c0 a8 00 01 c0 a8 00 02",
			"<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Field</th><th>Value</th></tr>\n" +
				"<tr><td>Version</td><td>4</td></tr>\n" +
				"<tr><td>Internet Header Length (IHL)</td><td>4 (Error: this should always be at least 5) (16 bytes)</td></tr>\n" +
				"<tr><td>Differentiated Services Code Point (DSCP)</td><td>0</td></tr>\n" +
				"<tr><td>Explicit Congestion Notification (ECN)</td><td>0</td></tr>\n" +
				"<tr><td>Total length</td><td>60 bytes\n  IP header: 16 bytes\n  Data: 44 bytes</td></tr>\n" +
				"<tr><td>Identification</td><td>0x1c46 (7238)</td></tr>\n" +
				"<tr><td>Flags</td><td>0x02\n  Reserved bit:0 (must be 0)\n  Don't fragment:1\n  More fragments:0</td></tr>\n" +
				"<tr><td>Fragment offset</td><td>0</td></tr>\n" +
				"<tr><td>Time-To-Live</td><td>64</td></tr>\n" +
				"<tr><td>Protocol</td><td>6, Transmission Control (TCP)</td></tr>\n" +
				"<tr><td>Header checksum</td><td>b1e6 (incorrect, should be 9e22)</td></tr>\n" +
				"<tr><td>Source IP address</td><td>192.168.0.1</td></tr>\n" +
				"<tr><td>Destination IP address</td><td>192.168.0.2</td></tr>\n" +
				// Data starts at ihl*4 = 16, so the last 4 bytes. (CyberChef's NaN slice
				// returns the whole header here — the same coercion bug as above.)
				"<tr><td>Data (hex)</td><td>c0 a8 00 02</td></tr></table>",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Table"}}},
		},
		// IHL > 5 renders the trailing Options row from the extra header words.
		{
			"Table with options", "46 00 00 3c 1c 46 40 00 40 06 b1 e6 c0 a8 00 01 c0 a8 00 02 0a 0b 0c 0d",
			"<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Field</th><th>Value</th></tr>\n" +
				"<tr><td>Version</td><td>4</td></tr>\n" +
				"<tr><td>Internet Header Length (IHL)</td><td>6 (24 bytes)</td></tr>\n" +
				"<tr><td>Differentiated Services Code Point (DSCP)</td><td>0</td></tr>\n" +
				"<tr><td>Explicit Congestion Notification (ECN)</td><td>0</td></tr>\n" +
				"<tr><td>Total length</td><td>60 bytes\n  IP header: 24 bytes\n  Data: 36 bytes</td></tr>\n" +
				"<tr><td>Identification</td><td>0x1c46 (7238)</td></tr>\n" +
				"<tr><td>Flags</td><td>0x02\n  Reserved bit:0 (must be 0)\n  Don't fragment:1\n  More fragments:0</td></tr>\n" +
				"<tr><td>Fragment offset</td><td>0</td></tr>\n" +
				"<tr><td>Time-To-Live</td><td>64</td></tr>\n" +
				"<tr><td>Protocol</td><td>6, Transmission Control (TCP)</td></tr>\n" +
				"<tr><td>Header checksum</td><td>b1e6 (incorrect, should be 9c22)</td></tr>\n" +
				"<tr><td>Source IP address</td><td>192.168.0.1</td></tr>\n" +
				"<tr><td>Destination IP address</td><td>192.168.0.2</td></tr>\n" +
				"<tr><td>Data (hex)</td><td></td></tr><tr><td>Options</td><td>0a 0b 0c 0d</td></tr></table>",
			core.Recipe{{Op: "Parse IPv4 header", Args: []any{"Hex", "Table"}}},
		},
	})
}

// TestByteSliceClamps directly exercises the JS-slice clamping helpers; the
// negative-start branches are unreachable through the parser operations (which
// only pass non-negative offsets) but must still clamp correctly for callers.
func TestByteSliceClamps(t *testing.T) {
	b := []byte{1, 2, 3}
	if got := byteSliceFrom(b, -1); !bytes.Equal(got, b) {
		t.Fatalf("byteSliceFrom(-1) = %v, want %v", got, b)
	}
	if got := byteSliceFrom(b, 5); len(got) != 0 {
		t.Fatalf("byteSliceFrom(past end) = %v, want empty", got)
	}
	if got := byteSliceRange(b, -1, 2); !bytes.Equal(got, []byte{1, 2}) {
		t.Fatalf("byteSliceRange(-1,2) = %v", got)
	}
	if got := byteSliceRange(b, 5, 6); len(got) != 0 {
		t.Fatalf("byteSliceRange(past end) = %v, want empty", got)
	}
	if got := byteSliceRange(b, 2, 1); len(got) != 0 {
		t.Fatalf("byteSliceRange(end<start) = %v, want empty", got)
	}
	if got := byteSliceRange(b, 1, 10); !bytes.Equal(got, []byte{2, 3}) {
		t.Fatalf("byteSliceRange(end>len) = %v", got)
	}
}

// TestParseIPv4HeaderShort feeds a truncated header so out-of-bounds byte reads
// return 0 and the payload slice clamps to empty.
func TestParseIPv4HeaderShort(t *testing.T) {
	if _, err := runOp(t, "Parse IPv4 header", "45", "Hex", "Table"); err != nil {
		t.Fatalf("short header: %v", err)
	}
}
