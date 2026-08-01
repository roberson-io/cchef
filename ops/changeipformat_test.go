package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestChangeIPFormatFixtures transcribes CyberChef's ChangeIPFormat.mjs cases.
func TestChangeIPFormatFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Dotted Decimal to Hex", "192.168.1.1", "c0a80101",
			core.Recipe{{Op: "Change IP format", Args: []any{"Dotted Decimal", "Hex"}}},
		},
		{
			"Decimal to Dotted Decimal", "3232235777", "192.168.1.1",
			core.Recipe{{Op: "Change IP format", Args: []any{"Decimal", "Dotted Decimal"}}},
		},
		{
			"Hex to Octal", "c0a80101", "030052000401",
			core.Recipe{{Op: "Change IP format", Args: []any{"Hex", "Octal"}}},
		},
		{
			"Octal to Decimal", "030052000401", "3232235777",
			core.Recipe{{Op: "Change IP format", Args: []any{"Octal", "Decimal"}}},
		},
	})
}

// TestChangeIPFormatBranches covers the blank-line skip and the identity
// (input format == output format) passthrough.
func TestChangeIPFormatBranches(t *testing.T) {
	out, err := runOp(t, "Change IP format", "1.2.3.4\n\n5.6.7.8", "Dotted Decimal", "Dotted Decimal")
	if err != nil {
		t.Fatal(err)
	}
	if out != "1.2.3.4\n5.6.7.8" {
		t.Fatalf("identity passthrough = %q", out)
	}
}

// --- direct tests for the parse/format helpers extracted from Run ---

// TestIPParseInput documents each input format decoding to IP bytes.
func TestIPParseInput(t *testing.T) {
	cases := []struct {
		format, line string
		want         []byte
	}{
		{"Dotted Decimal", "172.20.23.54", []byte{172, 20, 23, 54}},
		{"Decimal", "10", []byte{0, 0, 0, 10}},
		{"Octal", "012", []byte{0, 0, 0, 10}},
		{"Hex", "ac141736", []byte{0xac, 0x14, 0x17, 0x36}},
	}
	for _, c := range cases {
		got, err := ipParseInput(c.format, c.line)
		if err != nil || len(got) != len(c.want) {
			t.Fatalf("%s %q: %v, %v", c.format, c.line, got, err)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s %q: got %v want %v", c.format, c.line, got, c.want)
			}
		}
	}
	if _, err := ipParseInput("Bogus", "1.2.3.4"); err == nil {
		t.Fatal("expected error for unsupported input format")
	}
}

// TestIPFormatOutput documents each output format rendering of IP bytes.
func TestIPFormatOutput(t *testing.T) {
	ba := []byte{172, 20, 23, 54}
	cases := []struct{ format, want string }{
		{"Dotted Decimal", "172.20.23.54"},
		{"Decimal", "2886997814"},
		{"Octal", "025405013466"},
		{"Hex", "ac141736"},
	}
	for _, c := range cases {
		got, err := ipFormatOutput(c.format, ba)
		if err != nil || got != c.want {
			t.Fatalf("%s: got %q (%v) want %q", c.format, got, err, c.want)
		}
	}
	if _, err := ipFormatOutput("Bogus", ba); err == nil {
		t.Fatal("expected error for unsupported output format")
	}
}
