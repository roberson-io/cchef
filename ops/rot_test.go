package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Cases transcribed from CyberChef tests/operations/tests/Rotate.mjs.
func TestRotFixtures(t *testing.T) {
	const sample = "The Quick Brown Fox Jumped Over The Lazy Dog. 0123456789"
	runCases(t, []opCase{
		{
			"ROT13: nothing", "", "",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 13}}},
		},
		{
			"ROT13: no shift", sample, sample,
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 0}}},
		},
		{
			"ROT13: normal", sample, "Gur Dhvpx Oebja Sbk Whzcrq Bire Gur Ynml Qbt. 3456789012",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, 13}}},
		},
		{
			"ROT13: negative", sample, "Gur Dhvpx Oebja Sbk Whzcrq Bire Gur Ynml Qbt. 7890123456",
			core.Recipe{{Op: "ROT13", Args: []any{true, true, true, -13}}},
		},

		{
			"ROT47: nothing", "", "",
			core.Recipe{{Op: "ROT47", Args: []any{47}}},
		},
		{
			"ROT47: normal", "The Quick Brown Fox Jumped Over The Lazy Dog.",
			"%96 \"F:4< qC@H? u@I yF>A65 ~G6C %96 {2KJ s@8]",
			core.Recipe{{Op: "ROT47", Args: []any{47}}},
		},
	})
}

func TestROT47AmountBranches(t *testing.T) {
	if _, err := runOp(t, "ROT47", "abc", 0.0); err != nil {
		t.Fatalf("ROT47 amount 0: %v", err)
	}
	if _, err := runOp(t, "ROT47", "abc", -5.0); err != nil {
		t.Fatalf("ROT47 negative amount: %v", err)
	}
}

func TestROT13BruteForceVectors(t *testing.T) {
	// args: rotateLower, rotateUpper, rotateNum, sampleLength, sampleOffset,
	// printAmount, crib.
	runCases(t, []opCase{
		{
			"ROT13 Brute: crib", "Uryyb Jbeyq", "Amount = 13: Hello World",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, true, false, 100.0, 0.0, true, "hello"}}},
		},
		{
			"ROT13 Brute: print off", "Uryyb", "Hello",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, true, false, 100.0, 0.0, false, "hello"}}},
		},
		{
			"ROT13 Brute: rotate numbers", "Nby5", "Amount = 13: Aol8",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, true, true, 100.0, 0.0, true, "aol8"}}},
		},
		{
			"ROT13 Brute: numbers only", "012",
			"Amount =  3: 345\nAmount = 13: 345\nAmount = 23: 345",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{false, false, true, 100.0, 0.0, true, "345"}}},
		},
		{
			"ROT13 Brute: upper only", "URYYB", "Amount = 13: HELLO",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{false, true, false, 100.0, 0.0, true, "hello"}}},
		},
		{
			"ROT13 Brute: lower only", "uryyb", "Amount = 13: hello",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, false, false, 100.0, 0.0, true, "hello"}}},
		},
		{
			"ROT13 Brute: sample offset/length", "XXUryybYY", "Amount = 13: Hel",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, true, false, 3.0, 2.0, true, "hel"}}},
		},
		{
			"ROT13 Brute: whitespace escaped", "a\tb", "Amount = 13: no",
			core.Recipe{{Op: "ROT13 Brute Force", Args: []any{true, true, false, 100.0, 0.0, true, "n\to"}}},
		},
	})
}

func TestROT47BruteForceVectors(t *testing.T) {
	// args: sampleLength, sampleOffset, printAmount, crib.
	runCases(t, []opCase{
		{
			"ROT47 Brute: crib", "E6DE", "Amount = 15: TEST\nAmount = 47: test",
			core.Recipe{{Op: "ROT47 Brute Force", Args: []any{100.0, 0.0, true, "test"}}},
		},
		{
			"ROT47 Brute: print off", "E6DE", "TEST\ntest",
			core.Recipe{{Op: "ROT47 Brute Force", Args: []any{100.0, 0.0, false, "test"}}},
		},
		{
			"ROT47 Brute: sample offset/length", "zzE6DEzz", "Amount = 15: TEST\nAmount = 47: test",
			core.Recipe{{Op: "ROT47 Brute Force", Args: []any{4.0, 2.0, true, "test"}}},
		},
	})
}
