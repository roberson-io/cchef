package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

const b62Alph = "0-9A-Za-z"

// Cases transcribed from CyberChef tests/operations/tests/Base62.mjs.
func TestBase62Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base62: nothing", "", "",
			core.Recipe{{Op: "To Base62", Args: []any{b62Alph}}},
		},
		{
			"To Base62: Hello, World!", "Hello, World!", "1wJfrzvdbtXUOlUjUf",
			core.Recipe{{Op: "To Base62", Args: []any{b62Alph}}},
		},
		{
			"To Base62: UTF-8", "ნუ პანიკას", "BPDNbjoGvDCDzHbKT77eWg0vGQrJuWRXltuRVZ",
			core.Recipe{{Op: "To Base62", Args: []any{b62Alph}}},
		},

		{
			"From Base62: nothing", "", "",
			core.Recipe{{Op: "From Base62", Args: []any{b62Alph}}},
		},
		{
			"From Base62: Hello, World!", "1wJfrzvdbtXUOlUjUf", "Hello, World!",
			core.Recipe{{Op: "From Base62", Args: []any{b62Alph}}},
		},
		{
			"From Base62: UTF-8", "BPDNbjoGvDCDzHbKT77eWg0vGQrJuWRXltuRVZ", "ნუ პანიკას",
			core.Recipe{{Op: "From Base62", Args: []any{b62Alph}}},
		},

		{
			"Base62 round trip", "The quick brown fox", "The quick brown fox",
			core.Recipe{
				{Op: "To Base62", Args: []any{b62Alph}},
				{Op: "From Base62", Args: []any{b62Alph}},
			},
		},
	})
}

func TestBase62Branches(t *testing.T) {
	if out, err := runOp(t, "To Base62", "\x00", base62Alphabet); err != nil || out != "0" {
		t.Fatalf("To Base62(0x00) = %q, %v; want %q", out, err, "0")
	}
	if _, err := runOp(t, "From Base62", "!!!abc", base62Alphabet); err != nil {
		t.Fatalf("From Base62(strip): %v", err)
	}
}
