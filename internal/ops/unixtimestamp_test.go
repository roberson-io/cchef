package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestFromUNIXTimestampOracle checks From UNIX Timestamp against CyberChef-server
// output (v11.2.0); there is no upstream fixture. The op renders the timestamp in
// UTC using a fixed format ("ddd D MMMM YYYY HH:mm:ss", plus ".SSS" for sub-second
// units).
func TestFromUNIXTimestampOracle(t *testing.T) {
	runCases(t, []opCase{
		{
			"From UNIX: seconds", "1276263039", "Fri 11 June 2010 13:30:39 UTC",
			core.Recipe{{Op: "From UNIX Timestamp", Args: []any{"Seconds (s)"}}},
		},
		{
			"From UNIX: milliseconds", "1276263039529", "Fri 11 June 2010 13:30:39.529 UTC",
			core.Recipe{{Op: "From UNIX Timestamp", Args: []any{"Milliseconds (ms)"}}},
		},
		{
			"From UNIX: microseconds", "1276263039529769", "Fri 11 June 2010 13:30:39.529 UTC",
			core.Recipe{{Op: "From UNIX Timestamp", Args: []any{"Microseconds (μs)"}}},
		},
		{
			"From UNIX: nanoseconds", "1276263039529769300", "Fri 11 June 2010 13:30:39.529 UTC",
			core.Recipe{{Op: "From UNIX Timestamp", Args: []any{"Nanoseconds (ns)"}}},
		},
	})
}

// TestToUNIXTimestampOracle checks To UNIX Timestamp against CyberChef-server
// output (v11.2.0). Treat-as-UTC is used so the result is deterministic.
func TestToUNIXTimestampOracle(t *testing.T) {
	runCases(t, []opCase{
		{
			"To UNIX: seconds, show datetime", "2013-02-04 22:33:01",
			"1360017181 (Mon 4 February 2013 22:33:01 UTC)",
			core.Recipe{{Op: "To UNIX Timestamp", Args: []any{"Seconds (s)", true, true}}},
		},
		{
			"To UNIX: seconds, no datetime", "2013-02-04 22:33:01", "1360017181",
			core.Recipe{{Op: "To UNIX Timestamp", Args: []any{"Seconds (s)", true, false}}},
		},
		{
			"To UNIX: milliseconds", "2013-02-04 22:33:01", "1360017181000",
			core.Recipe{{Op: "To UNIX Timestamp", Args: []any{"Milliseconds (ms)", true, false}}},
		},
		{
			"To UNIX: microseconds", "2013-02-04 22:33:01", "1360017181000000",
			core.Recipe{{Op: "To UNIX Timestamp", Args: []any{"Microseconds (μs)", true, false}}},
		},
		{
			"To UNIX: nanoseconds", "2013-02-04 22:33:01", "1360017181000000000",
			core.Recipe{{Op: "To UNIX Timestamp", Args: []any{"Nanoseconds (ns)", true, false}}},
		},
	})
}
