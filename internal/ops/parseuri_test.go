package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseURIOracle checks Parse URI against CyberChef-server output (v11.2.0);
// there is no upstream fixture. cchef uses Go's net/url, approximating Node's
// url.parse: the protocol keeps its ":", the hash its "#", "Arguments:" is always
// emitted, query args preserve order, and a host with no path yields "/".
func TestParseURIOracle(t *testing.T) {
	uri := core.Recipe{{Op: "Parse URI", Args: []any{}}}
	runCases(t, []opCase{
		{"full URI", "https://user:pass@example.com:8080/path/to/page?q=1&r=2#frag",
			"Protocol:\thttps:\nAuth:\t\tuser:pass\nHostname:\texample.com\nPort:\t\t8080\nPath name:\t/path/to/page\nArguments:\n\tq = 1\n\tr = 2\nHash:\t\t#frag\n",
			uri},
		{"simple", "http://example.com/index.html",
			"Protocol:\thttp:\nHostname:\texample.com\nPath name:\t/index.html\nArguments:\n", uri},
		{"empty query value + padding", "http://host/p?a=1&bb=",
			"Protocol:\thttp:\nHostname:\thost\nPath name:\t/p\nArguments:\n\ta  = 1\n\tbb\n", uri},
		{"no path, with hash", "ftp://files.example.org#top",
			"Protocol:\tftp:\nHostname:\tfiles.example.org\nPath name:\t/\nArguments:\nHash:\t\t#top\n", uri},
	})
}
