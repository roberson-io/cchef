package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Syntax highlighter has no CyberChef test fixtures and is excluded from
// CyberChef's Node build, so the CyberChef-server oracle cannot bake it.
// CyberChef highlights with highlight.js, emitting hljs-* span classes; cchef
// reimplements this over chroma (github.com/alecthomas/chroma). chroma is a
// different engine, so token boundaries are not byte-identical to highlight.js —
// these tests assert the structural contract (the hljs-* class vocabulary, HTML
// escaping and the error path) rather than an exact highlight.js match.

// TestSyntaxHighlighterArgs pins the argument list to CyberChef's single
// Language argument. An argument CyberChef does not define would travel in a
// generated share URL as a step it cannot read back.
func TestSyntaxHighlighterArgs(t *testing.T) {
	args := SyntaxHighlighter{}.Args()
	if len(args) != 1 {
		t.Fatalf("Args() has %d entries, want 1: %+v", len(args), args)
	}
	if args[0].Name != "Language" {
		t.Errorf("argument 0 is %q, want %q", args[0].Name, "Language")
	}
}

func TestSyntaxHighlighterHTML(t *testing.T) {
	out, err := runOp(t, "Syntax highlighter", "const x = 42;", "javascript")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, want := range []string{
		`<span class="hljs-keyword">const</span>`,
		`<span class="hljs-number">42</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML output missing %q\ngot: %s", want, out)
		}
	}
	// CyberChef emits bare hljs spans, not chroma's <pre class="chroma"> wrapper.
	if strings.Contains(out, "chroma") || strings.Contains(out, "<pre") {
		t.Errorf("HTML output should not contain a chroma/pre wrapper: %s", out)
	}
}

func TestSyntaxHighlighterEscapesHTML(t *testing.T) {
	out, err := runOp(t, "Syntax highlighter", "x = a < b && c > d;", "javascript")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, esc := range []string{"&lt;", "&gt;", "&amp;"} {
		if !strings.Contains(out, esc) {
			t.Errorf("HTML output should escape to include %q\ngot: %s", esc, out)
		}
	}
	if strings.Contains(out, "<b ") || strings.Contains(out, "a < b") {
		t.Errorf("HTML output left raw markup unescaped: %s", out)
	}
}

func TestSyntaxHighlighterAutoDetect(t *testing.T) {
	// chroma's language analysis is weaker than highlight.js's (a documented
	// reduced-fidelity trade-off); Go is reliably detected from its structure.
	src := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n"
	out, err := runOp(t, "Syntax highlighter", src, "auto detect")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "hljs-keyword") {
		t.Errorf("auto-detected Go should highlight keywords\ngot: %s", out)
	}
}

func TestSyntaxHighlighterExactClassMap(t *testing.T) {
	// Go's `int` is a KeywordType, which maps through the exact-override table to
	// hljs-type (distinct from the plain hljs-keyword category default).
	out, err := runOp(t, "Syntax highlighter", "var n int = 5\n", "go")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, `<span class="hljs-type">int</span>`) {
		t.Errorf("expected hljs-type for Go int keyword\ngot: %s", out)
	}
}

func TestSyntaxHighlighterAutoDetectFallback(t *testing.T) {
	// Prose that no lexer recognises exercises the plaintext fallback: it runs
	// without error and emits the (escaped) text with no highlighting spans.
	out, err := runOp(t, "Syntax highlighter", "the quick brown fox\n", "auto detect")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.Contains(out, "<span") {
		t.Errorf("unrecognised prose should not be highlighted\ngot: %s", out)
	}
}

func TestSyntaxHighlighterUnknownLanguage(t *testing.T) {
	if _, err := runOp(t, "Syntax highlighter", "x", "not-a-real-language"); err == nil {
		t.Fatal("expected an error for an unknown language")
	}
}

func TestSyntaxHighlighterDefaults(t *testing.T) {
	// Empty Args → CoerceArgs supplies the default "auto detect" language.
	out, err := core.Recipe{{Op: "Syntax highlighter"}}.Execute(sdish("const x = 1;"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("default output should be HTML, not terminal ANSI: %q", out.String())
	}
}
