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
		// A leading UTF-8 BOM is stripped rather than becoming part of the first cell.
		{
			"leading BOM stripped", "\uFEFFname,age\nAlice,30",
			"+-------+-----+\n| name  | age |\n+-------+-----+\n| Alice | 30  |\n+-------+-----+\n",
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, true, "ASCII"}}},
		},
		// A CRLF row delimiter counts as a single break, not two empty rows: the
		// parser skips the \n after a matching \r (parseCSV's double-delim guard).
		{
			"CRLF row delimiter", "a,b\r\nc,d", "+---+---+\n| a | b |\n| c | d |\n+---+---+\n",
			core.Recipe{{Op: "To Table", Args: []any{",", `\r\n`, false, "ASCII"}}},
		},
		// Empty input yields empty output (parseCSV returns no rows). The oracle
		// server rejects empty input, so this is verified against ToTable.mjs.
		{
			"empty input", "", "",
			core.Recipe{{Op: "To Table", Args: []any{",", `\n`, false, "ASCII"}}},
		},
	})
}

// TestParseCSVQuotedFields exercises parseCSV's quote handling directly. These
// arms are unreachable through the To Table op (Run escapeHTML-escapes the input
// first, so a raw " never reaches parseCSV), but parseCSV is a faithful port of
// CyberChef's shared Utils.parseCSV, whose quoting contract is worth pinning:
// quoted fields protect the delimiter/newline, and "" is an escaped quote.
func TestParseCSVQuotedFields(t *testing.T) {
	comma, nl := []rune{','}, []rune{'\n'}
	eq := func(t *testing.T, got [][]string, want [][]string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("rows: got %v, want %v", got, want)
		}
		for i := range want {
			if len(got[i]) != len(want[i]) {
				t.Fatalf("row %d: got %v, want %v", i, got[i], want[i])
			}
			for j := range want[i] {
				if got[i][j] != want[i][j] {
					t.Fatalf("cell [%d][%d]: got %q, want %q", i, j, got[i][j], want[i][j])
				}
			}
		}
	}
	// A quoted field protects an embedded delimiter.
	eq(t, parseCSV(`"a,b",c`, comma, nl), [][]string{{"a,b", "c"}})
	// "" inside a quoted field is a single literal quote. (A trailing field is
	// needed because parseCSV only flushes the last cell once a row has started.)
	eq(t, parseCSV(`"a""b",z`, comma, nl), [][]string{{`a"b`, "z"}})
	// A quoted field protects an embedded newline (stays one cell, one row).
	eq(t, parseCSV("\"line1\nline2\",x", comma, nl), [][]string{{"line1\nline2", "x"}})
}

// TestCSVParser documents the CSV state machine extracted from parseCSV: a
// quoted field may contain the cell delimiter literally, and "" is an escaped
// quote.
func TestCSVParser(t *testing.T) {
	feedAll := func(s string) [][]string {
		p := &csvParser{cellDelims: []rune{','}, lineDelims: []rune{'\n'}}
		r := []rune(s)
		for i := 0; i < len(r); i++ {
			var next rune
			if i+1 < len(r) {
				next = r[i+1]
			}
			if p.feed(r[i], next) {
				i++
			}
		}
		return p.finish()
	}

	got := feedAll(`"a,b",c`)
	if len(got) != 1 || len(got[0]) != 2 || got[0][0] != "a,b" || got[0][1] != "c" {
		t.Fatalf("quoted delimiter: %v", got)
	}
	got = feedAll(`"x""y",z`)
	if len(got) != 1 || got[0][0] != `x"y` || got[0][1] != "z" {
		t.Fatalf("escaped quote: %v", got)
	}
}
