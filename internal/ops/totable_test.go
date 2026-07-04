package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// To Table outputs verified byte-for-byte against the CyberChef-server oracle.
func TestToTable(t *testing.T) {
	asciiHdr := "+---+---+---+\n" +
		"| a | b | c |\n" +
		"+---+---+---+\n" +
		"| d | e | f |\n" +
		"+---+---+---+\n"
	markdown := "| name  | age |\n" +
		"| ----- | --- |\n" +
		"| Alice | 30  |\n" +
		"| Bob   | 25  |\n"
	html := "<table class='table table-hover table-sm table-bordered table-nonfluid'>" +
		"<tbody><tr><td>a</td><td>b</td></tr><tr><td>c</td><td>d</td></tr></tbody></table>"

	runCases(t, []opCase{
		{
			"ASCII with header", "a,b,c\nd,e,f", asciiHdr,
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, true, "ASCII"}}},
		},
		{
			"Markdown", "name,age\nAlice,30\nBob,25", markdown,
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, true, "Markdown"}}},
		},
		{
			"HTML", "a,b\nc,d", html,
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, false, "HTML"}}},
		},
		{
			"HTML with header", "a,b\n1,2",
			"<table class='table table-hover table-sm table-bordered table-nonfluid'>" +
				"<thead class='thead-light'><tr><th>a</th><th>b</th></tr></thead>" +
				"<tbody><tr><td>1</td><td>2</td></tr></tbody></table>",
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, true, "HTML"}}},
		},
		// escapeHtml is applied before parsing (a CyberChef quirk).
		{
			"HTML-escaped cells", "a&b,c<d", "+---------+--------+\n| a&amp;b | c&lt;d |\n+---------+--------+\n",
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, false, "ASCII"}}},
		},
	})
}
