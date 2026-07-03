package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestStripHTTPHeadersOracle checks Strip HTTP headers against CyberChef-server
// output (v11.2.0); there is no upstream fixture. Everything up to the first
// blank line (CRLFCRLF or LFLF) is removed.
func TestStripHTTPHeadersOracle(t *testing.T) {
	strip := core.Recipe{{Op: "Strip HTTP headers", Args: []any{}}}
	runCases(t, []opCase{
		{"CRLF headers", "HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n<h1>Hi</h1>", "<h1>Hi</h1>", strip},
		{"LF headers", "HTTP/1.1 200 OK\nX-Test: 1\n\nbody here", "body here", strip},
	})
}
