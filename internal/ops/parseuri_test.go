package ops

import (
	"strings"
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
		{
			"full URI", "https://user:pass@example.com:8080/path/to/page?q=1&r=2#frag",
			"Protocol:\thttps:\nAuth:\t\tuser:pass\nHostname:\texample.com\nPort:\t\t8080\nPath name:\t/path/to/page\nArguments:\n\tq = 1\n\tr = 2\nHash:\t\t#frag\n",
			uri,
		},
		{
			"simple", "http://example.com/index.html",
			"Protocol:\thttp:\nHostname:\texample.com\nPath name:\t/index.html\nArguments:\n", uri,
		},
		{
			"empty query value + padding", "http://host/p?a=1&bb=",
			"Protocol:\thttp:\nHostname:\thost\nPath name:\t/p\nArguments:\n\ta  = 1\n\tbb\n", uri,
		},
		{
			"no path, with hash", "ftp://files.example.org#top",
			"Protocol:\tftp:\nHostname:\tfiles.example.org\nPath name:\t/\nArguments:\nHash:\t\t#top\n", uri,
		},
	})
}

func TestParseURIError(t *testing.T) {
	// A URI with no scheme before "://" fails url.Parse.
	if _, err := runOp(t, "Parse URI", "://x"); err == nil {
		t.Fatal("Parse URI (missing scheme): expected an error")
	}
}

// --- direct tests for the helpers extracted from ParseURI.Run ---

// TestParseOrderedQuery documents insertion-ordered query parsing with repeated
// keys (multiple values collect in order).
func TestParseOrderedQuery(t *testing.T) {
	order, values := parseOrderedQuery("a=1&b=2&a=3")
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("order: %v", order)
	}
	if len(values["a"]) != 2 || values["a"][0] != "1" || values["a"][1] != "3" {
		t.Fatalf("values[a]: %v", values["a"])
	}
	if o, _ := parseOrderedQuery(""); len(o) != 0 {
		t.Fatalf("empty query -> %v", o)
	}
}

// TestWriteURIQuery documents the "Arguments:" section rendering.
func TestWriteURIQuery(t *testing.T) {
	var b strings.Builder
	writeURIQuery(&b, "x=1")
	if b.String() != "Arguments:\n\tx = 1\n" {
		t.Fatalf("got %q", b.String())
	}
}
