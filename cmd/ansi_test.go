package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

// newANSICmd returns a command carrying the IO flags with output captured, so
// stdout is never a terminal unless a case says otherwise.
func newANSICmd(t *testing.T) *cobra.Command {
	t.Helper()
	c := newIOCmd()
	c.SetOut(&bytes.Buffer{})
	return c
}

func TestWantANSIModes(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{ansiAlways, true},
		{ansiNever, false},
		{ansiAuto, false}, // captured output is not a terminal
	}
	for _, tc := range cases {
		c := newANSICmd(t)
		flagANSI = tc.mode
		got, err := wantANSI(c)
		if err != nil {
			t.Fatalf("%s: %v", tc.mode, err)
		}
		if got != tc.want {
			t.Errorf("wantANSI with --ansi=%s = %v, want %v", tc.mode, got, tc.want)
		}
	}
	flagANSI = ansiAuto
}

func TestWantANSIRejectsUnknownMode(t *testing.T) {
	c := newANSICmd(t)
	flagANSI = "sometimes"
	defer func() { flagANSI = ansiAuto }()
	if _, err := wantANSI(c); err == nil {
		t.Fatal("expected an error for an unrecognized --ansi value")
	}
}

func TestWantANSIHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	c := newANSICmd(t)

	flagANSI = ansiAuto
	if got, err := wantANSI(c); err != nil || got {
		t.Errorf("NO_COLOR should turn auto off (got %v, %v)", got, err)
	}
	// An explicit --ansi=always is the user overriding the environment.
	flagANSI = ansiAlways
	if got, err := wantANSI(c); err != nil || !got {
		t.Errorf("--ansi=always should override NO_COLOR (got %v, %v)", got, err)
	}
	flagANSI = ansiAuto
}

// TestAutoANSI covers the "auto" decision directly, since neither an
// interactive stdout nor -o redirection can be staged through wantANSI in a
// test process whose output is a pipe.
func TestAutoANSI(t *testing.T) {
	cases := []struct {
		name             string
		toFile, terminal bool
		want             bool
	}{
		{"interactive stdout", false, true, true},
		{"piped stdout", false, false, false},
		{"redirected with -o", true, true, false},
	}
	for _, tc := range cases {
		if got := autoANSI(tc.toFile, tc.terminal); got != tc.want {
			t.Errorf("%s: autoANSI(%v, %v) = %v, want %v", tc.name, tc.toFile, tc.terminal, got, tc.want)
		}
	}
	t.Setenv("NO_COLOR", "1")
	if autoANSI(false, true) {
		t.Error("NO_COLOR should turn auto off even for an interactive stdout")
	}
}

// TestANSIFlagLeavesColourArgumentsAlone checks that --ansi takes no name an
// operation argument uses: View Bit Plane's Colour argument keeps both its
// spellings, and --ansi is a separate flag alongside them.
func TestANSIFlagLeavesColourArgumentsAlone(t *testing.T) {
	for _, spelling := range []string{"--colour=Green", "--color=Green"} {
		cmd := opCommandByName(t, "View Bit Plane")
		if err := cmd.ParseFlags([]string{spelling, "--ansi=never"}); err != nil {
			t.Fatalf("%s: %v", spelling, err)
		}
		if got, _ := cmd.Flags().GetString("colour"); got != "Green" {
			t.Errorf("%s: colour = %q, want %q", spelling, got, "Green")
		}
		if got, _ := cmd.Flags().GetString("ansi"); got != ansiNever {
			t.Errorf("%s: ansi = %q, want %q", spelling, got, ansiNever)
		}
	}
	flagANSI = ansiAuto
}

func TestRecipeHighlights(t *testing.T) {
	cases := []struct {
		name   string
		recipe core.Recipe
		want   bool
	}{
		{"only step", core.Recipe{{Op: "Syntax highlighter"}}, true},
		{"last step", core.Recipe{{Op: "To Upper case"}, {Op: "Syntax highlighter"}}, true},
		{"not last", core.Recipe{{Op: "Syntax highlighter"}, {Op: "To Base64"}}, false},
		{"disabled", core.Recipe{{Op: "Syntax highlighter", Disabled: true}}, false},
		{"trailing step disabled", core.Recipe{
			{Op: "Syntax highlighter"}, {Op: "To Base64", Disabled: true},
		}, true},
		{"other op", core.Recipe{{Op: "To Base64"}}, false},
		{"empty", nil, false},
	}
	for _, tc := range cases {
		if got := recipeHighlights(tc.recipe); got != tc.want {
			t.Errorf("%s: recipeHighlights = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestANSIFromHLJSPlainText(t *testing.T) {
	// Text outside a span is HTML-escaped by the operation and comes back as it
	// was written, with no escape sequences added.
	got := ansiFromHLJS("a &lt; b &amp;&amp; c &gt; d &quot;e&quot;")
	if want := `a < b && c > d "e"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestANSIFromHLJSColorsSpans(t *testing.T) {
	got := ansiFromHLJS(`<span class="hljs-keyword">const</span> x = <span class="hljs-number">42</span>;`)
	want := "\x1b[35mconst\x1b[0m x = \x1b[36m42\x1b[0m;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestANSIFromHLJSUnescapesInsideSpans(t *testing.T) {
	got := ansiFromHLJS(`<span class="hljs-string">&quot;a &amp; b&quot;</span>`)
	if want := "\x1b[32m\"a & b\"\x1b[0m"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestANSIFromHLJSUnmappedClassIsPlain(t *testing.T) {
	// Every class the operation emits has a color, so an unknown one can only
	// come from elsewhere; it is rendered as its text rather than dropped.
	got := ansiFromHLJS(`<span class="hljs-nonesuch">text</span>`)
	if got != "text" {
		t.Errorf("got %q, want %q", got, "text")
	}
}

// TestSyntaxHighlighterANSIEndToEnd runs the subcommand both ways. Test output
// is a buffer rather than a terminal, so the default is the HTML CyberChef
// produces and --ansi=always is what asks for ANSI.
func TestSyntaxHighlighterANSIEndToEnd(t *testing.T) {
	const src = "const x = 42;"

	plain := execRoot(t, "syntax-highlighter", "--language", "javascript", src)
	if !strings.Contains(plain, `<span class="hljs-keyword">const</span>`) {
		t.Errorf("piped output should be HTML: %q", plain)
	}
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("piped output should not be colored: %q", plain)
	}

	colored := execRoot(t, "syntax-highlighter", "--language", "javascript", "--ansi", ansiAlways, src)
	if !strings.Contains(colored, "\x1b[35mconst\x1b[0m") {
		t.Errorf("--ansi=always should color the keyword: %q", colored)
	}
	if strings.Contains(colored, "hljs-") {
		t.Errorf("colored output should carry no HTML classes: %q", colored)
	}
}

// TestANSILeavesOtherOperationsAlone checks that --ansi=always does not
// touch output that is not highlighted HTML, even when it contains spans.
func TestANSILeavesOtherOperationsAlone(t *testing.T) {
	const html = `<span class="hljs-keyword">const</span>`
	got := execRoot(t, "to-lower-case", "--ansi", ansiAlways, html)
	if !strings.Contains(got, "hljs-keyword") || strings.Contains(got, "\x1b[") {
		t.Errorf("unhighlighted output should pass through: %q", got)
	}
}

// TestANSIOverDirInput checks that color reaches each file's result under
// --in-dir, and that --out-dir keeps the HTML since those results are files.
func TestANSIOverDirInput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte("const x = 42;"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := execRoot(t, "syntax-highlighter", "--in-dir", dir, "--language", "javascript", "--ansi", ansiAlways)
	if !strings.Contains(got, "\x1b[35mconst\x1b[0m") {
		t.Errorf("--in-dir output should be colored: %q", got)
	}

	outDir := t.TempDir()
	execRoot(t, "syntax-highlighter", "--in-dir", dir, "--out-dir", outDir,
		"--language", "javascript", "--ansi", ansiAlways)
	written, err := os.ReadFile(filepath.Join(outDir, "a.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "hljs-keyword") || strings.Contains(string(written), "\x1b[") {
		t.Errorf("files written under --out-dir should keep the HTML: %q", written)
	}
}

// TestANSIRejectedEndToEnd checks an unusable value fails the command rather
// than being ignored, whatever the operation.
func TestANSIRejectedEndToEnd(t *testing.T) {
	err := execRootErr(t, "to-upper-case", "--ansi", "maybe", "x")
	if err == nil || !strings.Contains(err.Error(), "--ansi must be") {
		t.Fatalf("expected an unusable --ansi to fail, got %v", err)
	}
	flagANSI = ansiAuto
}

// TestANSIFromHLJSCoversEveryClass checks that every hljs class the operation
// can emit has a color, so no highlighted token silently loses its color.
func TestANSIFromHLJSCoversEveryClass(t *testing.T) {
	out, err := core.Recipe{{Op: "Syntax highlighter", Args: []any{"go"}}}.Execute(
		core.NewDish([]byte("// c\npackage main\n\ntype T int\n\nfunc f() string { return \"s\" }\n"), core.TypeString))
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, m := range hljsSpanRE.FindAllStringSubmatch(html, -1) {
		if _, ok := hljsANSI[m[1]]; !ok {
			t.Errorf("class %q has no ANSI color", m[1])
		}
	}
	if !strings.Contains(ansiFromHLJS(html), "\x1b[") {
		t.Error("highlighted Go source produced no color")
	}
}
