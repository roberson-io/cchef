package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// urlRecipe builds a recipe for the operation with the arguments given.
func urlRecipe(total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract URLs",
		Args: []any{total, sorted, unique},
	}}
}

// TestExtractURLs covers the switches and the shape of a URL, each expectation
// taken from the CyberChef-server oracle.
func TestExtractURLs(t *testing.T) {
	runCases(t, []opCase{
		{
			"plain",
			"See https://www.example.com/path/to/page?query=1&x=2#frag for details.\nftp://files.example.org:2121/pub/file.txt\nBare www.example.com is not a URL. HTTP://UPPER.EXAMPLE.COM/A\n(https://example.com/a,b) and https://example.com/end.\nweird://a.b/c!d?e\n",
			"https://www.example.com/path/to/page?query=1&x=2#frag\nftp://files.example.org:2121/pub/file.txt\nHTTP://UPPER.EXAMPLE.COM/A\nhttps://example.com/a,b)\nhttps://example.com/end\nweird://a.b/c!d?e",
			urlRecipe(false, false, false),
		},
		{
			"display total",
			"See https://www.example.com/path/to/page?query=1&x=2#frag for details.\nftp://files.example.org:2121/pub/file.txt\nBare www.example.com is not a URL. HTTP://UPPER.EXAMPLE.COM/A\n(https://example.com/a,b) and https://example.com/end.\nweird://a.b/c!d?e\n",
			"Total found: 6\n\nhttps://www.example.com/path/to/page?query=1&x=2#frag\nftp://files.example.org:2121/pub/file.txt\nHTTP://UPPER.EXAMPLE.COM/A\nhttps://example.com/a,b)\nhttps://example.com/end\nweird://a.b/c!d?e",
			urlRecipe(true, false, false),
		},
		{
			"sorted",
			"See https://www.example.com/path/to/page?query=1&x=2#frag for details.\nftp://files.example.org:2121/pub/file.txt\nBare www.example.com is not a URL. HTTP://UPPER.EXAMPLE.COM/A\n(https://example.com/a,b) and https://example.com/end.\nweird://a.b/c!d?e\n",
			"ftp://files.example.org:2121/pub/file.txt\nHTTP://UPPER.EXAMPLE.COM/A\nhttps://example.com/a,b)\nhttps://example.com/end\nhttps://www.example.com/path/to/page?query=1&x=2#frag\nweird://a.b/c!d?e",
			urlRecipe(false, true, false),
		},
		{
			"unique",
			"http://a.com/x http://A.com/x http://a.com/x http://b.org",
			"http://a.com/x\nhttp://A.com/x\nhttp://b.org",
			urlRecipe(false, false, true),
		},
		{
			"sorted and unique with total",
			"http://a.com/x http://A.com/x http://a.com/x http://b.org",
			"Total found: 3\n\nhttp://a.com/x\nhttp://A.com/x\nhttp://b.org",
			urlRecipe(true, true, true),
		},
		{
			"protocol required",
			"no links here, just www.example.com",
			"",
			urlRecipe(false, false, false),
		},
		{
			"nothing found with total",
			"no links here, just www.example.com",
			"Total found: 0\n\n",
			urlRecipe(true, false, false),
		},
	})
}
