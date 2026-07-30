package ops

import (
	"strings"
	"testing"
)

// yaraGolden is one case from testdata/yara.jsonl: a rule set, an input, the
// six display flags, and exactly what CyberChef printed — or the fault it
// reported instead.
type yaraGolden struct {
	Name     string `json:"name"`
	Rules    string `json:"rules"`
	InputHex string `json:"inputHex"`
	Flags    []bool `json:"flags"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// args builds the operation's arguments for a golden.
func (g yaraGolden) args() []any {
	out := make([]any, 0, len(g.Flags)+1)
	out = append(out, g.Rules)
	for _, f := range g.Flags {
		out = append(out, f)
	}
	return out
}

// TestYARAGoldens runs every case against what CyberChef's own libyara gave.
func TestYARAGoldens(t *testing.T) {
	for _, g := range readJSONL[yaraGolden](t, "testdata/yara.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			got, err := runOp(t, "YARA Rules", string(unhex(t, g.InputHex)), g.args()...)
			if g.Error != "" {
				if err == nil {
					t.Fatalf("accepted rules CyberChef refused with %q", g.Error)
				}
				if err.Error() != g.Error {
					t.Errorf("refused with  %q\nCyberChef says %q", err.Error(), g.Error)
				}
				return
			}
			if err != nil {
				t.Fatalf("refused rules CyberChef accepted: %v", err)
			}
			if got != g.Output {
				t.Errorf("got  %q\nwant %q", got, g.Output)
			}
		})
	}
}

// TestYARAFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/YARA.mjs).
func TestYARAFixtures(t *testing.T) {
	const consoleRule = `import "console"
rule a
{
  strings:
    $s=" "
  condition:
    $s and console.log("log rule a")
}
rule b
{
  strings:
    $s=" "
  condition:
    $s and console.hex("log rule b: int8(0)=", int8(0))
}`

	cases := []struct {
		name  string
		rules string
		input string
		args  []any
		want  string
	}{
		{
			"simple foobar",
			`rule foo {strings: $re1 = /foo/ condition: $re1} ` +
				`rule bar {strings: $re1 = /bar/ condition: $re1}`,
			"foobar foobar bar foo foobar",
			[]any{true, true, true, true},
			"Rule \"foo\" matches (4 times):\n" +
				"Pos 0, length 3, identifier $re1, data: \"foo\"\n" +
				"Pos 7, length 3, identifier $re1, data: \"foo\"\n" +
				"Pos 18, length 3, identifier $re1, data: \"foo\"\n" +
				"Pos 22, length 3, identifier $re1, data: \"foo\"\n" +
				"Rule \"bar\" matches (4 times):\n" +
				"Pos 3, length 3, identifier $re1, data: \"bar\"\n" +
				"Pos 10, length 3, identifier $re1, data: \"bar\"\n" +
				"Pos 14, length 3, identifier $re1, data: \"bar\"\n" +
				"Pos 25, length 3, identifier $re1, data: \"bar\"\n",
		},
		{
			"hashing rules",
			`import "hash"
			rule HelloWorldMD5 {
				condition:
					hash.md5(0,filesize) == "ed076287532e86365e841e92bfc50d8c"
			}

			rule HelloWorldSHA256 {
				condition:
					hash.sha256(0,filesize) == "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"
			}`,
			"Hello World!",
			[]any{true, true, true, true, false, false},
			"Input matches rule \"HelloWorldMD5\".\nInput matches rule \"HelloWorldSHA256\".\n",
		},
		{
			"compile warnings",
			consoleRule, "CyberChef Yara",
			[]any{false, false, false, false, true, false},
			"Warning on line 5: string \"$s\" may slow down scanning\n" +
				"Warning on line 12: string \"$s\" may slow down scanning\n" +
				"Input matches rule \"a\".\nInput matches rule \"b\".\n",
		},
		{
			"console messages",
			consoleRule, "CyberChef Yara",
			[]any{false, false, false, false, false, true},
			"log rule a\nlog rule b: int8(0)=0x43\n" +
				"Input matches rule \"a\".\nInput matches rule \"b\".\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runOp(t, "YARA Rules", c.input, append([]any{c.rules}, c.args...)...)
			if err != nil {
				t.Fatalf("YARA Rules: %v", err)
			}
			if got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestYARARefusesUnsupportedModifiers covers what this engine does not do yet,
// which is said outright rather than left to match nothing.
func TestYARARefusesUnsupportedModifiers(t *testing.T) {
	for _, rules := range []string{
		`rule R { strings: $a = /x/ xor condition: $a }`,
		`rule R { strings: $a = { 68 } wide condition: $a }`,
	} {
		t.Run(rules, func(t *testing.T) {
			_, err := runOp(t, "YARA Rules", "hello", rules, false, false, false, true, true, true)
			if err == nil {
				t.Fatal("accepted rules this engine cannot run")
			}
			if strings.TrimSpace(err.Error()) == "" {
				t.Error("refused without saying why")
			}
		})
	}
}
