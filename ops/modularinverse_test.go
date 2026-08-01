package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestModularInverseFixtures runs CyberChef's ModularInverse.mjs cases.
func TestModularInverseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Modular Inverse: basic example (3 mod 11)",
			"",
			"4",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"3", "11"}}},
		},
		{
			"Modular Inverse: another coprime pair (7 mod 26)",
			"",
			"15",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"7", "26"}}},
		},
		{
			"Modular Inverse: hexadecimal input (0x10 mod 0x11)",
			"",
			"16",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"0x10", "0x11"}}},
		},
		{
			"Modular Inverse: using input field for value",
			"5",
			"21",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"", "26"}}},
		},
		{
			"Modular Inverse: using input field for modulus",
			"17",
			"7",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"5", ""}}},
		},
		{
			"Modular Inverse: large number (RSA-like)",
			"",
			"934281398294",
			core.Recipe{{Op: "Modular Inverse", Args: []any{"65537", "9999999999999"}}},
		},
	})
}

// TestModularInverseRefusals covers the moduli and values it will not work
// with, and taking a value from the input.
func TestModularInverseRefusals(t *testing.T) {
	got, err := runOp(t, "Modular Inverse", "3", "", "11")
	if err != nil {
		t.Fatalf("value from the input: %v", err)
	}
	if got != "4" {
		t.Errorf("3 inverse mod 11 = %s, want 4", got)
	}

	for _, c := range []struct{ name, a, m, want string }{
		{"a modulus of zero", "3", "0", "Modulus must be greater than zero"},
		{"a negative modulus", "3", "-7", "Modulus must be greater than zero"},
		{"no inverse exists", "4", "8", "Inverse does not exist"},
		{"the value will not read", "three", "11", "Value (a) must be decimal or hex"},
		{"the modulus will not read", "3", "eleven", "Modulus (m) must be decimal or hex"},
		{"neither given", "", "", "Value (a) and Modulus (m) must be defined"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Modular Inverse", "", c.a, c.m)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want it to mention %q", err, c.want)
			}
		})
	}

	// A modulus given through the input, and a value larger than it.
	if got, err := runOp(t, "Modular Inverse", "11", "14", ""); err != nil || got != "4" {
		t.Errorf("14 inverse mod 11 = %q (%v), want 4", got, err)
	}
	// A negative value is brought into range first.
	if got, err := runOp(t, "Modular Inverse", "", "-8", "11"); err != nil || got != "4" {
		t.Errorf("-8 inverse mod 11 = %q (%v), want 4", got, err)
	}
}
