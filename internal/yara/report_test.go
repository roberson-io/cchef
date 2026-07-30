package yara

import "testing"

// reportOf compiles and runs a rule set and renders what it found.
func reportOf(t *testing.T, src, data string, show Display) string {
	t.Helper()
	set, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rules, err := Compile(set)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	results, logs, err := rules.Scan([]byte(data))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return Report(rules.Warnings, logs, results, show)
}

// TestReport covers the words CyberChef writes for each combination of what to
// show, checked against what it actually printed for the same rules.
func TestReport(t *testing.T) {
	const rich = `rule R { meta: author = "me" strings: $a = "hello" $b = "l" ` +
		`condition: $a and #b == 3 }`

	cases := []struct {
		name string
		show Display
		want string
	}{
		{
			"nothing at all",
			Display{},
			"Input matches rule \"R\".\n",
		},
		{
			"counts only",
			Display{Counts: true},
			// The doubled space is CyberChef's own.
			"Input matches rule \"R\"  (4 times).\n",
		},
		{
			"strings only",
			Display{Strings: true},
			"Rule \"R\" matches:\n" +
				"Pos 0, identifier $a, data: \"hello\"\n" +
				"Pos 2, identifier $b, data: \"l\"\n" +
				"Pos 3, identifier $b, data: \"l\"\n" +
				"Pos 9, identifier $b, data: \"l\"\n",
		},
		{
			"strings and lengths",
			Display{Strings: true, Lengths: true},
			"Rule \"R\" matches:\n" +
				"Pos 0, length 5, identifier $a, data: \"hello\"\n" +
				"Pos 2, length 1, identifier $b, data: \"l\"\n" +
				"Pos 3, length 1, identifier $b, data: \"l\"\n" +
				"Pos 9, length 1, identifier $b, data: \"l\"\n",
		},
		{
			"lengths without the data",
			Display{Lengths: true},
			"Rule \"R\" matches:\n" +
				"Pos 0, length 5, identifier $a\n" +
				"Pos 2, length 1, identifier $b\n" +
				"Pos 3, length 1, identifier $b\n" +
				"Pos 9, length 1, identifier $b\n",
		},
		{
			"metadata only",
			Display{Meta: true},
			"Input matches rule \"R\" [author: me].\n",
		},
		{
			"everything",
			Display{Strings: true, Lengths: true, Meta: true, Counts: true},
			"Rule \"R\" [author: me] matches (4 times):\n" +
				"Pos 0, length 5, identifier $a, data: \"hello\"\n" +
				"Pos 2, length 1, identifier $b, data: \"l\"\n" +
				"Pos 3, length 1, identifier $b, data: \"l\"\n" +
				"Pos 9, length 1, identifier $b, data: \"l\"\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := reportOf(t, rich, "hello world", c.show); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestReportCounts covers how a count is worded, which changes for one match.
func TestReportCounts(t *testing.T) {
	one := reportOf(t, `rule R { strings: $a = "hello" condition: $a }`,
		"hello world", Display{Counts: true})
	if one != "Input matches rule \"R\"  (1 time).\n" {
		t.Errorf("got %q", one)
	}
	// A rule with no strings has nothing to count, so no count is written.
	none := reportOf(t, `rule R { condition: true }`, "hello", Display{Counts: true})
	if none != "Input matches rule \"R\".\n" {
		t.Errorf("got %q", none)
	}
}

// TestReportWarningsAndLogs covers the two things written before any rule: the
// warnings about the rules themselves, and whatever they had to say.
func TestReportWarningsAndLogs(t *testing.T) {
	const src = `import "console" rule R { strings: $a = "l" ` +
		`condition: $a and console.log("saying something") }`

	both := reportOf(t, src, "hello", Display{Warnings: true, Console: true})
	want := "Warning on line 1: string \"$a\" may slow down scanning\n" +
		"saying something\nInput matches rule \"R\".\n"
	if both != want {
		t.Errorf("got  %q\nwant %q", both, want)
	}

	neither := reportOf(t, src, "hello", Display{})
	if neither != "Input matches rule \"R\".\n" {
		t.Errorf("got %q", neither)
	}
}

// TestReportMetadataValues covers the three sorts of value metadata can hold.
func TestReportMetadataValues(t *testing.T) {
	got := reportOf(t, `rule R { meta: s = "text" i = 42 n = -7 b = true `+
		`strings: $a = "hello" condition: $a }`, "hello", Display{Meta: true})
	want := "Input matches rule \"R\" [s: text, i: 42, n: -7, b: true].\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestMetaValueTextOfSomethingElse covers a metadata value of a sort the parser
// never produces.
func TestMetaValueTextOfSomethingElse(t *testing.T) {
	if got := metaValueText(1.5); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}

// TestReportOfNothing covers a scan that found nothing at all.
func TestReportOfNothing(t *testing.T) {
	if got := Report(nil, nil, nil, Display{Warnings: true, Console: true}); got != "" {
		t.Errorf("got %q, want nothing", got)
	}
}
