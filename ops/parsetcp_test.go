package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestParseTCPFixtures transcribes CyberChef's ParseTCP.mjs cases (expected
// values captured minified from the CyberChef-server oracle). Exercises the
// no-options, options (MSS/NOP/Window Scale/SACK) and Timestamps paths.
func TestParseTCPFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"no options", "c2eb0050a138132e70dc9fb9501804025ea70000",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":1893507001,"Data offset":"5 (20 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":1,"PSH":1,"RST":0,"SYN":0,"FIN":0},"Window size":"1026 (Scaled: 1026)","Checksum":"0x5ea7","Urgent pointer":"0x0000"}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"options", "c2eb0050a1380c1f000000008002faf080950000020405b40103030801010402",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704804895","Acknowledgement number":0,"Data offset":"8 (32 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"64240 (Scaled: 16445440)","Checksum":"0x8095","Urgent pointer":"0x0000","Options":{"Maximum Segment Size":{"Kind":2,"Length":4,"Value":1460},"No-Operation":{"Kind":1},"Window Scale":{"Kind":3,"Length":3,"Value":{"Shift count":8,"Multiplier":256}},"SACK Permitted":{"Kind":4,"Length":2}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"alternate checksum option", "c2eb0050a138132e000000006002faf0000000000e030100",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":0,"Data offset":"6 (24 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"64240 (Scaled: 64240)","Checksum":"0x0000","Urgent pointer":"0x0000","Options":{"TCP Alternate Checksum Request (obsolete)":{"Kind":14,"Length":3,"Value":"8-bit Fletchers's algorithm (0x01)"},"End of Option List":{"Kind":0}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"timestamps", "9e90e11574d57b2c00000000a002ffffe5740000020405b40402080aa4e8c8f50000000001030308",
			`{"Source port":40592,"Destination port":57621,"Sequence number":"1960147756","Acknowledgement number":0,"Data offset":"10 (40 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"65535 (Scaled: 16776960)","Checksum":"0xe574","Urgent pointer":"0x0000","Options":{"Maximum Segment Size":{"Kind":2,"Length":4,"Value":1460},"SACK Permitted":{"Kind":4,"Length":2},"Timestamps":{"Kind":8,"Length":10,"Value":{"Current Timestamp":"2766719221","Echo Reply":"0"}},"No-Operation":{"Kind":1},"Window Scale":{"Kind":3,"Length":3,"Value":{"Shift count":8,"Multiplier":256}}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
	})
}

// TestParseTCPBranches covers the option-parsing edge cases not in the happy-path
// fixture: unknown option kinds, mis-sized option parsers, long hex values,
// trailing data and a too-short header. Values are oracle-verified except the
// window-scale error (CyberChef 500s on it; cchef degrades gracefully via its
// value type guard).
func TestParseTCPBranches(t *testing.T) {
	if _, err := runOp(t, "Parse TCP", "0000", "Hex"); err == nil {
		t.Fatal("expected error for a short TCP header")
	}
	cases := []struct{ name, input, want string }{
		{"unknown kind", "005001bb0000000000000000600000000000000063040000", `"Reserved":{"Kind":99,"Length":4,"Value":0}`},
		{"timestamp bad length", "005001bb0000000000000000600000000000000008040000", "Timestamp field should be 8 bytes long (received 0x0000)"},
		{"long option hex value", "005001bb000000000000000070000000000000000208000000000000", `"Value":"0x000000000000"`},
		{"trailing data", "005001bb00000000000000005000000000000000deadbeef", `"Data":"0xdeadbeef"`},
		{"window scale bad length", "005001bb0000000000000000600000000000000003040000", "Window Scale should be one byte long (received 0x0000)"},
	}
	for _, c := range cases {
		out, err := runOp(t, "Parse TCP", c.input, "Hex")
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: missing %q in:\n%s", c.name, c.want, out)
		}
	}
}

// --- direct tests for the helpers extracted from ParseTCP.Run ---

// TestParseTCPOptionValue documents the three option-value forms: a custom
// parser, a short integer (<= 6 bytes), and a long hex dump.
func TestParseTCPOptionValue(t *testing.T) {
	// Custom parser: receives optLength-2 bytes.
	parsed := parseTCPOptionValue(newByteStream([]byte{0xaa, 0xbb, 0xcc}),
		tcpOpt{parser: func(b []byte) any { return len(b) }}, 5)
	if parsed != 3 {
		t.Errorf("parser: got %v want 3", parsed)
	}
	// Short integer: optLength 4 -> readInt(2), big-endian 0x0005.
	if got := parseTCPOptionValue(newByteStream([]byte{0x00, 0x05}), tcpOpt{}, 4); got != 5 {
		t.Errorf("int: got %v want 5", got)
	}
	// Long value: optLength 9 -> "0x" + hex of 7 bytes.
	got := parseTCPOptionValue(newByteStream([]byte{1, 2, 3, 4, 5, 6, 7}), tcpOpt{}, 9)
	if got != "0x01020304050607" {
		t.Errorf("hex: got %v", got)
	}
}

// TestTCPWindowScale documents extracting the window-scale shift count from a
// parsed option value (a *omap with a "Shift count" entry).
func TestTCPWindowScale(t *testing.T) {
	om := newOMap()
	om.set("Shift count", 7)
	if sc, ok := tcpWindowScale(om); !ok || sc != 7 {
		t.Errorf("omap: got %d %v", sc, ok)
	}
	if _, ok := tcpWindowScale("not an omap"); ok {
		t.Error("non-omap should not yield a shift count")
	}
	if _, ok := tcpWindowScale(newOMap()); ok {
		t.Error("omap without Shift count should not yield one")
	}
}
