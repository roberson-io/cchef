package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestFiletimeFixtures transcribes CyberChef's DateTime.mjs Filetime cases.
// Windows Filetime counts 100 ns intervals since 1601-01-01; the ops convert to
// and from a UNIX timestamp in the chosen unit, offsetting by the epoch delta
// 116444736000000000.
func TestFiletimeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Filetime to Unix (ns, decimal)", "129207366395297693", "1276263039529769300",
			core.Recipe{{Op: "Windows Filetime to UNIX Timestamp", Args: []any{"Nanoseconds (ns)", "Decimal"}}},
		},
		{
			"Unix to Filetime (ns, decimal)", "1276263039529769300", "129207366395297693",
			core.Recipe{{Op: "UNIX Timestamp to Windows Filetime", Args: []any{"Nanoseconds (ns)", "Decimal"}}},
		},
	})
}

// TestFiletimeOracle covers the radix and unit paths CyberChef's fixtures omit,
// using output captured from the CyberChef-server oracle (v11.2.0). The seconds
// and microseconds conversions produce fractional results, exercising the 20-dp
// rounding shared with the arithmetic operations.
func TestFiletimeOracle(t *testing.T) {
	u2f := func(u, f string) core.Recipe {
		return core.Recipe{{Op: "UNIX Timestamp to Windows Filetime", Args: []any{u, f}}}
	}
	f2u := func(u, f string) core.Recipe {
		return core.Recipe{{Op: "Windows Filetime to UNIX Timestamp", Args: []any{u, f}}}
	}
	runCases(t, []opCase{
		{"Unix to Filetime (s, hex BE)", "1276263039", "1cb096a480a2980", u2f("Seconds (s)", "Hex (big endian)")},
		{"Unix to Filetime (s, hex LE)", "1276263039", "80290a486a09cb01", u2f("Seconds (s)", "Hex (little endian)")},
		{"Unix to Filetime (s, decimal)", "1276263039", "129207366390000000", u2f("Seconds (s)", "Decimal")},
		{"Unix to Filetime (empty input)", "", "", u2f("Seconds (s)", "Decimal")},

		{"Filetime to Unix (s, hex BE)", "1a01b62d45f9ce0", "67896559.5413728", f2u("Seconds (s)", "Hex (big endian)")},
		{"Filetime to Unix (ms, hex LE)", "e09c5f452db6010a", "60461298490966.7552", f2u("Milliseconds (ms)", "Hex (little endian)")},
		{"Filetime to Unix (μs, decimal)", "129207366395297693", "1276263039529769.3", f2u("Microseconds (μs)", "Decimal")},
		{"Filetime to Unix (empty input)", "", "", f2u("Seconds (s)", "Decimal")},
	})
}

// TestFiletimeBranches covers the NaN/odd-length paths: non-hex input, a
// NaN passed to the hex encoder, and an odd-length little-endian hex string.
func TestFiletimeBranches(t *testing.T) {
	if _, err := runOp(t, "Windows Filetime to UNIX Timestamp", "zz", "Nanoseconds (ns)", "Hex (big endian)"); err != nil {
		t.Fatalf("non-hex filetime: %v", err)
	}
	if _, err := runOp(t, "UNIX Timestamp to Windows Filetime", "abc", "Nanoseconds (ns)", "Hex (big endian)"); err != nil {
		t.Fatalf("non-numeric to filetime: %v", err)
	}
	if _, err := runOp(t, "Windows Filetime to UNIX Timestamp", "abc", "Nanoseconds (ns)", "Hex (little endian)"); err != nil {
		t.Fatalf("odd-length LE filetime: %v", err)
	}
}
