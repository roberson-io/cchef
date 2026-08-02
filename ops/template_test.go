package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// templateFixture is one of CyberChef's own cases
// (CyberChef's tests/operations/tests/Template.mjs).
type templateFixture struct {
	Name     string `json:"name"`
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Template string `json:"template"`
}

// TestTemplateFixtures covers CyberChef's own cases.
func TestTemplateFixtures(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "template.json"))
	if err != nil {
		t.Fatalf("reading the fixtures: %v", err)
	}
	var fixtures []templateFixture
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("reading the fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures were read")
	}

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			got, err := runOp(t, "Template", f.Input, f.Template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != f.Expected {
				t.Errorf("got  %q\nwant %q", got, f.Expected)
			}
		})
	}
}

// TestTemplateCases covers the language against the CyberChef-server oracle.
func TestTemplateCases(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{"a value is escaped", `{"t":"&<>\"'` + "`" + `= x"}`, "{{t}}", "&amp;&lt;&gt;&quot;&#x27;&#x60;&#x3D; x"},
		{"three braces write it raw", `{"t":"&<>\"'` + "`" + `= x"}`, "{{{t}}}", "&<>\"'`= x"},
		{"a dotted path", `{"a":{"b":"deep"}}`, "{{a.b}}", "deep"},
		{"each over a list", `{"xs":[1,2,3]}`, "{{#each xs}}{{this}}-{{@index}} {{/each}}", "1-0 2-1 3-2 "},
		{"each over an object", `{"o":{"p":1,"q":2}}`, "{{#each o}}{{@key}}={{this}} {{/each}}", "p=1 q=2 "},
		{"if", `{"f":true}`, "{{#if f}}yes{{else}}no{{/if}}", "yes"},
		{"if else", `{"f":false}`, "{{#if f}}yes{{else}}no{{/if}}", "no"},
		{"unless", `{"f":false}`, "{{#unless f}}yes{{/unless}}", "yes"},
		{"with", `{"o":{"n":"x"}}`, "{{#with o}}{{n}}{{/with}}", "x"},
		{"a comment writes nothing", `{"a":1}`, "{{! a comment }}kept", "kept"},
		{"a missing value writes nothing", `{"a":1}`, "{{missing}}", ""},
		{"the enclosing context", `{"xs":[{"n":1}]}`, "{{#each xs}}{{n}}/{{../top}}{{/each}}", "1/"},
		{"the root context", `{"top":"T","xs":[1]}`, "{{#each xs}}{{@root.top}}{{/each}}", "T"},
		{"first and last", `{"xs":[1,2]}`, "{{#each xs}}{{#if @first}}F{{/if}}{{#if @last}}L{{/if}}{{/each}}", "FL"},
		{"zero is not present", `{"n":0}`, "{{#if n}}yes{{else}}no{{/if}}", "no"},
		{"each over nothing", `{"xs":[]}`, "{{#each xs}}x{{else}}empty{{/each}}", "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

// TestTemplateRejectsBadInput covers input that is not a JSON document.
func TestTemplateRejectsBadInput(t *testing.T) {
	for _, input := range []string{"", "not json", "{"} {
		if _, err := runOp(t, "Template", input, "x"); err == nil {
			t.Errorf("%q was read as JSON", input)
		}
	}
}

// TestTemplateRejectsBadTemplates covers templates that cannot be read.
func TestTemplateRejectsBadTemplates(t *testing.T) {
	for _, tc := range []struct{ name, template string }{
		{"an unclosed tag", "{{ a"},
		{"an empty tag", "{{}}"},
		{"a block that is never closed", "{{#if a}}yes"},
		{"a block closed by the wrong tag", "{{#if a}}yes{{/each}}"},
		{"a close with nothing open", "text{{/if}}"},
		{"an inline partial with no name", `{{#*inline ""}}x{{/inline}}`},
		{"a helper that does not exist", "{{#nope a}}x{{/nope}}"},
		{"a partial that was never defined", "{{> missing}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runOp(t, "Template", `{"a":1}`, tc.template); err == nil {
				t.Error("the template was accepted")
			}
		})
	}
}

// TestTemplateDefaults covers the argument the operation starts from.
func TestTemplateDefaults(t *testing.T) {
	op, ok := core.Default.Get("Template")
	if !ok {
		t.Fatal("Template is not registered")
	}
	args := op.Args()
	if len(args) != 1 || args[0].Name != "Template definition (.handlebars)" || args[0].Value != "" {
		t.Errorf("arguments are %+v", args)
	}
}

// TestTemplateValueFormatting covers how each kind of value is written.
func TestTemplateValueFormatting(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{"a whole number", `{"n":42}`, "{{n}}", "42"},
		{"a fraction", `{"n":1.5}`, "{{n}}", "1.5"},
		{"true", `{"b":true}`, "{{b}}", "true"},
		{"null", `{"b":null}`, "{{b}}", ""},
		{"a list", `{"xs":[1,2,3]}`, "{{xs}}", "1,2,3"},
		{"an object", `{"o":{"a":1}}`, "{{o}}", "[object Object]"},
		{"an index into a list", `{"xs":["a","b"]}`, "{{xs.1}}", "b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTemplateStandaloneWhitespace covers the rule that decides which newlines
// around a tag survive: a block tag alone on its line takes the line with it,
// while one sharing a line with text does not.
func TestTemplateStandaloneWhitespace(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{
			"a block on its own lines",
			`{"xs":[1,2]}`,
			"start\n{{#each xs}}\n{{this}}\n{{/each}}\nend",
			"start\n1\n2\nend",
		},
		{
			"a block sharing its lines",
			`{"xs":[1,2]}`,
			"start {{#each xs}}{{this}}{{/each}} end",
			"start 12 end",
		},
		{
			"a comment on its own line",
			`{"a":1}`,
			"one\n{{! note }}\ntwo",
			"one\ntwo",
		},
		{
			"an indented partial keeps its indentation",
			`{"a":1}`,
			"{{#*inline \"p\"}}\nx\ny\n{{/inline}}\n  {{> p}}\n",
			"  x\n  y\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

// TestTemplateMoreLanguage covers the corners the fixtures and the first round
// of cases leave out. Every expected value was taken from the CyberChef-server
// oracle before this was written.
func TestTemplateMoreLanguage(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{"a raw tag with no value", `{"a":1}`, "{{{missing}}}", ""},
		{"this on its own", `{"a":1}`, "{{#each xs}}{{/each}}ok", "ok"},
		{"a list walked with this", `{"xs":["a","b"]}`, "{{#each xs}}[{{this}}]{{/each}}", "[a][b]"},
		{"a dot standing for the context", `{"xs":["a"]}`, "{{#each xs}}{{.}}{{/each}}", "a"},
		{"this with a path after it", `{"o":{"n":5}}`, "{{#with o}}{{this.n}}{{/with}}", "5"},
		{"climbing two levels", `{"t":"T","a":[{"b":[1]}]}`, "{{#each a}}{{#each b}}{{../../t}}{{/each}}{{/each}}", "T"},
		{"climbing past the top", `{"a":1}`, "{{../nope}}", ""},
		{"with over nothing", `{"o":null}`, "{{#with o}}x{{else}}none{{/with}}", "none"},
		{"each over a value that is neither", `{"n":5}`, "{{#each n}}x{{else}}none{{/each}}", "none"},
		{"each over an empty object", `{"o":{}}`, "{{#each o}}x{{else}}none{{/each}}", "none"},
		{"an object's key order is kept", `{"o":{"z":1,"a":2}}`, "{{#each o}}{{@key}}{{/each}}", "za"},
		{"whole-number keys come first", `{"o":{"b":1,"2":2,"1":3}}`, "{{#each o}}{{@key}}{{/each}}", "12b"},
		{"a path into a list", `{"o":{"xs":[9,8]}}`, "{{o.xs.0}}", "9"},
		{"a path past the end of a list", `{"xs":[1]}`, "{{xs.5}}", ""},
		{"a path into something that is not a container", `{"n":5}`, "{{n.deep}}", ""},
		{"an unknown @ value", `{"a":1}`, "{{@nope}}", ""},
		{"an empty string is not present", `{"s":""}`, "{{#if s}}y{{else}}n{{/if}}", "n"},
		{"an empty list is not present", `{"xs":[]}`, "{{#if xs}}y{{else}}n{{/if}}", "n"},
		{"an object is present", `{"o":{}}`, "{{#if o}}y{{else}}n{{/if}}", "y"},
		{"a partial used twice", `{"a":1}`, `{{#*inline "p"}}x{{/inline}}{{> p}}{{> p}}`, "xx"},
		{"an inline partial reads the surrounding context", `{"n":7}`, `{{#*inline "p"}}{{n}}{{/inline}}{{> p}}`, "7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTemplateLongErrorText covers an unclosed tag long enough that the message
// shortens it.
func TestTemplateLongErrorText(t *testing.T) {
	_, err := runOp(t, "Template", `{"a":1}`, "{{ a very long tag that never closes at all")
	if err == nil {
		t.Fatal("the template was accepted")
	}
	if !strings.Contains(err.Error(), "…") {
		t.Errorf("got %q, want the tag shortened", err.Error())
	}
}

// TestTemplateHelperWithNoArgument covers a block helper written without the one
// value it works on, which each says its own thing about. The wording is
// Handlebars', taken from the oracle.
func TestTemplateHelperWithNoArgument(t *testing.T) {
	for _, tc := range []struct{ template, want string }{
		{"{{#if}}y{{else}}n{{/if}}", "#if requires exactly one argument"},
		{"{{#unless}}y{{/unless}}", "#unless requires exactly one argument"},
		{"{{#with}}y{{/with}}", "#with requires exactly one argument"},
		{"{{#each}}y{{else}}n{{/each}}", "Must pass iterator to #each"},
	} {
		t.Run(tc.template, func(t *testing.T) {
			_, err := runOp(t, "Template", `{"a":1}`, tc.template)
			if err == nil {
				t.Fatal("the template was accepted")
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestTemplateParseFailuresInside covers a template that is malformed inside a
// block or an inline partial, where the error has to travel back out.
func TestTemplateParseFailuresInside(t *testing.T) {
	for _, tc := range []struct{ name, template string }{
		{"a bad tag inside a block", "{{#if a}}{{}}{{/if}}"},
		{"a bad tag inside an inline partial", `{{#*inline "p"}}{{}}{{/inline}}`},
		{"a bad tag after an else", "{{#if a}}x{{else}}{{}}{{/if}}"},
		{"an inline partial that is never closed", `{{#*inline "p"}}x`},
		{"an inline partial closed by the wrong tag", `{{#*inline "p"}}x{{/if}}`},
		{"an unclosed tag inside a block", "{{#if a}}{{ b {{/if}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runOp(t, "Template", `{"a":1}`, tc.template); err == nil {
				t.Error("the template was accepted")
			}
		})
	}
}

// TestTemplateRootValues covers a document that is not an object, and reading
// the whole of it.
func TestTemplateRootValues(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{"a list at the root", `[1,2]`, "{{#each this}}{{this}}{{/each}}", "12"},
		{"a number at the root", `5`, "{{this}}", "5"},
		{"a string at the root", `"hi"`, "{{.}}", "hi"},
		{"the root read from inside a block", `{"t":"T","xs":[1]}`, "{{#each xs}}{{@root.t}}{{/each}}", "T"},
		{"the whole root", `{"a":1}`, "{{#each this}}{{@key}}{{/each}}", "a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTemplateNestedLocals covers a block reading a value of the block around
// it, which is written with the climb inside the @ rather than before it.
func TestTemplateNestedLocals(t *testing.T) {
	got, err := runOp(t, "Template", `{"xs":[["a"],["b"]]}`,
		"{{#each xs}}{{#each this}}{{@../index}}{{this}}{{/each}}{{/each}}")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got != "0a1b" {
		t.Errorf("got %q, want %q", got, "0a1b")
	}
}

// TestTemplateRemainingCorners covers the last few paths through the renderer.
func TestTemplateRemainingCorners(t *testing.T) {
	for _, tc := range []struct{ name, input, template, want string }{
		{
			"a partial defined inside a block", `{"xs":[1]}`,
			`{{#each xs}}{{#*inline "p"}}p{{/inline}}{{> p}}{{/each}}`, "p",
		},
		{"a value that is a nested list", `{"xs":[[1,2]]}`, "{{#each xs}}{{this}}{{/each}}", "1,2"},
		{"a list holding an object", `{"xs":[{"a":1}]}`, "{{xs}}", "[object Object]"},
		{"a raw list", `{"xs":["<a>"]}`, "{{{xs}}}", "<a>"},
		{"climbing from the root", `{"a":1}`, "{{#each this}}{{../nothing}}{{/each}}", ""},
		{"@ climbing past the top", `{"xs":[1]}`, "{{#each xs}}{{@../../index}}{{/each}}", ""},
		{"@root walked into", `{"o":{"n":3},"xs":[1]}`, "{{#each xs}}{{@root.o.n}}{{/each}}", "3"},
		{"a block with no else", `{"f":false}`, "{{#if f}}y{{/if}}done", "done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := runOp(t, "Template", tc.input, tc.template)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
