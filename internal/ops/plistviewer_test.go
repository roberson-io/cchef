package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// The output format is CyberChef's, taken from what it produces for well-formed
// plists: `plist => ` and the root container, braces for dictionaries and
// brackets for arrays, one entry to a line indented by a tab per level, names
// and array positions joined to their values by ` => `, strings in quotes, and
// `/plist` at the end.
//
// The cases are in two tables. The first holds those CyberChef also gets right,
// so they double as a check that the format has not drifted. The second holds
// those it does not, each noting what it gives instead.

// plistCase is one property list and what the operation should make of it.
type plistCase struct {
	name  string
	input string
	want  string
}

// plistFixtures are the cases CyberChef agrees on.
var plistFixtures = []plistCase{
	{
		"a flat dictionary",
		"<plist version=\"1.0\">\n<dict>\n\t<key>Name</key>\n\t<string>Widget</string>\n" +
			"\t<key>Count</key>\n\t<integer>42</integer>\n</dict>\n</plist>",
		"plist => {\n\tName => \"Widget\"\n\tCount => 42\n}\n/plist\n",
	},
	{
		"every value type",
		"<plist version=\"1.0\">\n<dict>\n\t<key>Name</key>\n\t<string>Widget</string>\n" +
			"\t<key>Count</key>\n\t<integer>42</integer>\n\t<key>Price</key>\n\t<real>19.99</real>\n" +
			"\t<key>Enabled</key>\n\t<true/>\n\t<key>Hidden</key>\n\t<false/>\n" +
			"\t<key>Made</key>\n\t<date>2026-01-01T00:00:00Z</date>\n" +
			"\t<key>Blob</key>\n\t<data>SGVsbG8=</data>\n</dict>\n</plist>",
		"plist => {\n\tName => \"Widget\"\n\tCount => 42\n\tPrice => 19.99\n" +
			"\tEnabled => true\n\tHidden => false\n\tMade => 2026-01-01T00:00:00Z\n" +
			"\tBlob => SGVsbG8=\n}\n/plist\n",
	},
	{
		"an array of strings",
		"<plist version=\"1.0\">\n<array>\n\t<string>alpha</string>\n\t<string>beta</string>\n" +
			"\t<string>gamma</string>\n</array>\n</plist>",
		"plist => [\n\t0 => \"alpha\"\n\t1 => \"beta\"\n\t2 => \"gamma\"\n]\n/plist\n",
	},
	{
		"an array inside a dictionary",
		"<plist version=\"1.0\">\n<dict>\n\t<key>Tags</key>\n\t<array>\n" +
			"\t\t<string>alpha</string>\n\t\t<string>beta</string>\n\t</array>\n" +
			"\t<key>After</key>\n\t<integer>7</integer>\n</dict>\n</plist>",
		"plist => {\n\tTags => [\n\t\t0 => \"alpha\"\n\t\t1 => \"beta\"\n\t]\n" +
			"\tAfter => 7\n}\n/plist\n",
	},
	{
		"a dictionary inside a dictionary",
		"<plist version=\"1.0\">\n<dict>\n\t<key>Outer</key>\n\t<dict>\n" +
			"\t\t<key>Inner</key>\n\t\t<string>value</string>\n\t</dict>\n" +
			"\t<key>Last</key>\n\t<true/>\n</dict>\n</plist>",
		"plist => {\n\tOuter => {\n\t\tInner => \"value\"\n\t}\n\tLast => true\n}\n/plist\n",
	},
	{
		"an empty dictionary",
		"<plist version=\"1.0\">\n<dict>\n</dict>\n</plist>",
		"plist => {\n}\n/plist\n",
	},
	{
		"an empty array",
		"<plist version=\"1.0\">\n<array>\n</array>\n</plist>",
		"plist => [\n]\n/plist\n",
	},
	{
		"negative and fractional numbers",
		"<plist version=\"1.0\">\n<dict>\n\t<key>Below</key>\n\t<integer>-17</integer>\n" +
			"\t<key>Tiny</key>\n\t<real>-0.5</real>\n</dict>\n</plist>",
		"plist => {\n\tBelow => -17\n\tTiny => -0.5\n}\n/plist\n",
	},
	{
		"a declaration and a doctype before the plist",
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" " +
			"\"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\">\n<dict>\n" +
			"\t<key>a</key>\n\t<integer>1</integer>\n</dict>\n</plist>",
		"plist => {\n\ta => 1\n}\n/plist\n",
	},
	{
		"deeply nested dictionaries",
		"<plist version=\"1.0\">\n<dict>\n\t<key>one</key>\n\t<dict>\n\t\t<key>two</key>\n" +
			"\t\t<dict>\n\t\t\t<key>three</key>\n\t\t\t<string>deep</string>\n\t\t</dict>\n" +
			"\t</dict>\n</dict>\n</plist>",
		"plist => {\n\tone => {\n\t\ttwo => {\n\t\t\tthree => \"deep\"\n\t\t}\n\t}\n}\n/plist\n",
	},
	{
		"booleans on their own in an array",
		"<plist version=\"1.0\">\n<array>\n\t<true/>\n\t<false/>\n\t<true/>\n</array>\n</plist>",
		"plist => [\n\t0 => true\n\t1 => false\n\t2 => true\n]\n/plist\n",
	},
	{
		"carriage returns between the elements",
		"<plist version=\"1.0\">\r\n<dict>\r\n<key>a</key>\r\n<integer>1</integer>\r\n</dict>\r\n</plist>",
		"plist => {\n\ta => 1\n}\n/plist\n",
	},
	{
		"a string at the root",
		"<plist version=\"1.0\">\n<string>alone</string>\n</plist>",
		"plist => \"alone\"\n/plist\n",
	},
}

// plistCorrected are the cases where CyberChef's answer is wrong. Each notes
// what CyberChef gives; cchef gives what the plist actually says.
var plistCorrected = []plistCase{
	{
		// CyberChef numbers the second inner array 3, because one counter is
		// shared across every level of nesting.
		"arrays inside arrays, each numbered from zero",
		"<plist version=\"1.0\">\n<array>\n\t<array>\n\t\t<integer>1</integer>\n" +
			"\t\t<integer>2</integer>\n\t</array>\n\t<array>\n\t\t<integer>3</integer>\n" +
			"\t</array>\n</array>\n</plist>",
		"plist => [\n\t0 => [\n\t\t0 => 1\n\t\t1 => 2\n\t]\n\t1 => [\n\t\t0 => 3\n\t]\n]\n/plist\n",
	},
	{
		// CyberChef gives "0 => {\n\t\t1 => a => 1\n\t2 => }" — the entries of
		// the dictionary are numbered as though they were array positions.
		"dictionaries inside an array",
		"<plist version=\"1.0\">\n<array>\n\t<dict>\n\t\t<key>a</key>\n\t\t<integer>1</integer>\n" +
			"\t</dict>\n\t<dict>\n\t\t<key>b</key>\n\t\t<integer>2</integer>\n\t</dict>\n" +
			"</array>\n</plist>",
		"plist => [\n\t0 => {\n\t\ta => 1\n\t}\n\t1 => {\n\t\tb => 2\n\t}\n]\n/plist\n",
	},
	{
		// CyberChef strips every space and tab from the whole document, so this
		// comes out as `ab => "helloworld"`.
		"spaces inside a name and a value, which are kept",
		"<plist version=\"1.0\">\n<dict>\n\t<key>a b</key>\n\t<string>hello world</string>\n" +
			"</dict>\n</plist>",
		"plist => {\n\ta b => \"hello world\"\n}\n/plist\n",
	},
	{
		// CyberChef's opening-tag pattern requires an attribute, so a bare
		// <plist> is left in the output as it stands.
		"a plist tag with no attributes",
		"<plist>\n<dict>\n\t<key>a</key>\n\t<integer>1</integer>\n</dict>\n</plist>",
		"plist => {\n\ta => 1\n}\n/plist\n",
	},
	{
		// CyberChef matches the opening tag greedily as far as the last angle
		// bracket on the line, swallowing the whole document, and then fails.
		"a plist all on one line",
		"<plist version=\"1.0\"><dict><key>a</key><integer>1</integer></dict></plist>",
		"plist => {\n\ta => 1\n}\n/plist\n",
	},
	{
		// CyberChef keeps only the last character of anything without a plist
		// tag, so this comes out as "t".
		"text with no plist in it",
		"just some text\nmore text",
		"",
	},
}

// TestPlistViewerFixtures covers the cases CyberChef and cchef agree on.
func TestPlistViewerFixtures(t *testing.T) { runPlistCases(t, plistFixtures) }

// TestPlistViewerCorrected covers the cases where cchef reads the plist and
// CyberChef does not.
func TestPlistViewerCorrected(t *testing.T) { runPlistCases(t, plistCorrected) }

// runPlistCases runs each case through the operation. A case wanting nothing is
// one the operation should refuse.
func runPlistCases(t *testing.T, cases []plistCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := core.Recipe{{Op: "P-list Viewer"}}.
				Execute(core.NewDish([]byte(c.input), core.TypeString))
			if c.want == "" {
				if err == nil {
					t.Fatalf("read it without complaint, giving %q", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.want {
				t.Errorf("got  %q\nwant %q", out.String(), c.want)
			}
		})
	}
}

// TestPlistViewerRejects covers input the operation cannot make a listing of.
// CyberChef gives a stray character or an internal error for most of these.
func TestPlistViewerRejects(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"nothing at all", ""},
		{"a single character", "x"},
		{"a plist tag and nothing else", "<plist version=\"1.0\">"},
		{"a plist holding nothing", "<plist version=\"1.0\">\n</plist>"},
		{"a key with no value", "<plist version=\"1.0\">\n<dict>\n\t<key>orphan</key>\n</dict>\n</plist>"},
		{"a value with no key", "<plist version=\"1.0\">\n<dict>\n\t<integer>1</integer>\n</dict>\n</plist>"},
		{"a closing tag that does not match", "<plist version=\"1.0\">\n<dict>\n</array>\n</plist>"},
		{
			"an element the format does not define",
			"<plist version=\"1.0\">\n<dict>\n\t<key>a</key>\n\t<colour>red</colour>\n</dict>\n</plist>",
		},
		{"a key outside a dictionary", "<plist version=\"1.0\">\n<array>\n\t<key>a</key>\n</array>\n</plist>"},
		{
			"a second value after the one the plist holds",
			"<plist version=\"1.0\">\n<dict>\n</dict>\n<dict>\n</dict>\n</plist>",
		},
		{"a document rooted at something else", "<other>\n<dict>\n</dict>\n</other>"},
		{"an attribute with no quotes round it", "<plist version=1.0>\n<dict>\n</dict>\n</plist>"},
		{
			"a tag left open in a value",
			"<plist version=\"1.0\">\n<dict>\n\t<key>a</key>\n\t<string>x</other>\n</dict>\n</plist>",
		},
		{
			"a tag left open after the value the plist holds",
			"<plist version=\"1.0\">\n<dict>\n</dict>\n<bad</plist>",
		},
		{
			"a tag left open inside an array",
			"<plist version=\"1.0\">\n<array>\n\t<string>x</bad>\n</array>\n</plist>",
		},
		{
			"a boolean left open",
			"<plist version=\"1.0\">\n<dict>\n\t<key>a</key>\n\t<true>\n</dict>\n</plist>",
		},
		{
			"a name left open",
			"<plist version=\"1.0\">\n<dict>\n\t<key>a</bad>\n\t<string>x</string>\n</dict>\n</plist>",
		},
		{
			"a tag left open where a value should follow a name",
			"<plist version=\"1.0\">\n<dict>\n\t<key>a</key>\n\t<bad<\n</dict>\n</plist>",
		},
		{
			"a tag left open where an array should hold a value",
			"<plist version=\"1.0\">\n<array>\n\t<bad<\n</array>\n</plist>",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "P-list Viewer", tc.input)
			if err == nil {
				t.Errorf("read %q as a property list, giving %q", tc.input, out)
			}
		})
	}
}

// TestPlistViewerNestingIsUnbounded covers a listing deep enough that a walk
// which recursed per level would be a worry.
func TestPlistViewerNestingIsUnbounded(t *testing.T) {
	const depth = 5000

	var b strings.Builder
	b.WriteString("<plist version=\"1.0\">\n")
	b.WriteString(strings.Repeat("<array>", depth))
	b.WriteString("<integer>1</integer>")
	b.WriteString(strings.Repeat("</array>", depth))
	b.WriteString("\n</plist>")

	out, err := runOp(t, "P-list Viewer", b.String())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := strings.Repeat("\t", depth) + "0 => 1\n"; !strings.Contains(out, want) {
		t.Errorf("the innermost value is not at depth %d", depth)
	}
}
