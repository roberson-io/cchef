package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestDechunkHTTPResponseFixtures transcribes CyberChef's DechunkHTTPResponse.mjs
// cases: chunked transfer-encoding bodies are reassembled and trailing headers
// discarded.
func TestDechunkHTTPResponseFixtures(t *testing.T) {
	dechunk := core.Recipe{{Op: "Dechunk HTTP response", Args: []any{}}}
	runCases(t, []opCase{
		{"CRLF line endings", "7\r\nMozilla\r\n9\r\nDeveloper\r\n7\r\nNetwork\r\n0\r\n\r\n", "MozillaDeveloperNetwork", dechunk},
		{"LF line endings", "7\nMozilla\n9\nDeveloper\n7\nNetwork\n0\n\n", "MozillaDeveloperNetwork", dechunk},
		{"single chunk", "5\r\nHello\r\n0\r\n\r\n", "Hello", dechunk},
		{"trailing headers discarded", "7\nMozilla\n9\nDeveloper\n7\nNetwork\n0\nExpires: Wed, 21 Oct 2015 07:28:00 GMT\n", "MozillaDeveloperNetwork", dechunk},
		{"hex chunk sizes", "a\r\n0123456789\r\n0\r\n\r\n", "0123456789", dechunk},
	})
}

func TestDechunkHTTPResponseBranches(t *testing.T) {
	// A non-hex chunk-size line yields no chunks (leadingHex reports NaN).
	if out, err := runOp(t, "Dechunk HTTP response", "xyz\nbody"); err != nil || out != "" {
		t.Fatalf("dechunk(non-hex) = %q, %v", out, err)
	}
	// A chunk size that overflows int64 also terminates parsing.
	if out, err := runOp(t, "Dechunk HTTP response", "ffffffffffffffff\nbody"); err != nil || out != "" {
		t.Fatalf("dechunk(overflow) = %q, %v", out, err)
	}
}
