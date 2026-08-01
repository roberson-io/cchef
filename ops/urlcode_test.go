package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// URL Encode/Decode have no upstream fixture file; these use hand-verified
// values derived from CyberChef's encodeBytes logic.
func TestURLEncodeDecode(t *testing.T) {
	runCases(t, []opCase{
		{
			"URL Encode default", "Hello World!", "Hello%20World!",
			core.Recipe{{Op: "URL Encode", Args: []any{false}}},
		},
		{
			"URL Encode all", "Hello World!", "Hello%20World%21",
			core.Recipe{{Op: "URL Encode", Args: []any{true}}},
		},
		{
			"URL Encode = default keeps", "a=b", "a=b",
			core.Recipe{{Op: "URL Encode", Args: []any{false}}},
		},
		{
			"URL Encode = all", "a=b", "a%3Db",
			core.Recipe{{Op: "URL Encode", Args: []any{true}}},
		},

		{
			"URL Decode percent", "Hello%20World%21", "Hello World!",
			core.Recipe{{Op: "URL Decode", Args: []any{true}}},
		},
		{
			"URL Decode plus as space", "a+b", "a b",
			core.Recipe{{Op: "URL Decode", Args: []any{true}}},
		},
		{
			"URL Decode plus literal", "a+b", "a+b",
			core.Recipe{{Op: "URL Decode", Args: []any{false}}},
		},

		{
			"URL round trip", "key=value & more!", "key=value & more!",
			core.Recipe{
				{Op: "URL Encode", Args: []any{true}},
				{Op: "URL Decode", Args: []any{true}},
			},
		},
	})
}
