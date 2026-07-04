package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestArithmeticFixtures transcribes CyberChef's Arithmetic.mjs fixtures and
// additional cases whose expected outputs were taken from the CyberChef-server
// oracle (authoritative bignumber.js output).
func TestArithmeticFixtures(t *testing.T) {
	runCases(t, []opCase{
		// Sum
		{
			"Sum: mixed radices", "0x0a 8 .5", "18.5",
			core.Recipe{{Op: "Sum", Args: []any{"Space"}}},
		},
		{
			"Sum: line feed delimiter", "10\n8\n.5", "18.5",
			core.Recipe{{Op: "Sum", Args: []any{"Line feed"}}},
		},
		{
			"Sum: no valid input", "test", "NaN",
			core.Recipe{{Op: "Sum", Args: []any{"Space"}}},
		},

		// Subtract (upstream fixtures)
		{
			"Subtract", "321,123,test", "198",
			core.Recipe{{Op: "Subtract", Args: []any{"Comma"}}},
		},
		{
			"Subtract: no valid input", "test", "NaN",
			core.Recipe{{Op: "Subtract", Args: []any{"Comma"}}},
		},

		// Multiply
		{
			"Multiply: mixed radices", "0x0a 8 .5", "40",
			core.Recipe{{Op: "Multiply", Args: []any{"Space"}}},
		},

		// Divide
		{
			"Divide: terminating", "0x0a 8 .5", "2.5",
			core.Recipe{{Op: "Divide", Args: []any{"Space"}}},
		},
		{
			"Divide: 20 decimal places, half-up", "1 3", "0.33333333333333333333",
			core.Recipe{{Op: "Divide", Args: []any{"Space"}}},
		},

		// Mean
		{
			"Mean: terminating", "0x0a 8 .5 .5", "4.75",
			core.Recipe{{Op: "Mean", Args: []any{"Space"}}},
		},
		{
			"Mean: non-terminating rounds to 20dp", "1 2 2", "1.66666666666666666667",
			core.Recipe{{Op: "Mean", Args: []any{"Space"}}},
		},
		{
			// No parseable numbers -> empty list -> meanNums returns NaN.
			"Mean: no numbers", "a b c", "NaN",
			core.Recipe{{Op: "Mean", Args: []any{"Space"}}},
		},

		// Median (upstream Median.mjs fixtures; the list is sorted first — see
		// CyberChef PR #2284, which fixed odd-length inputs not being sorted).
		{
			"Median: odd-length input", "10 1 2", "2",
			core.Recipe{{Op: "Median", Args: []any{"Space"}}},
		},
		{
			"Median: even-length input", "10 1 2 5", "3.5",
			core.Recipe{{Op: "Median", Args: []any{"Space"}}},
		},
		{
			"Median: even count averages middle pair", "0x0a 8 1 .5", "4.5",
			core.Recipe{{Op: "Median", Args: []any{"Space"}}},
		},
		{
			"Median: odd count", "0x0a 8 1", "8",
			core.Recipe{{Op: "Median", Args: []any{"Space"}}},
		},

		// Standard Deviation
		{
			"Standard Deviation", "0x0a 8 .5", "4.08928138212843238213",
			core.Recipe{{Op: "Standard Deviation", Args: []any{"Space"}}},
		},
	})
}
