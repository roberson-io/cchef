package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/termimage"
)

// resetIOFlags clears the shared input/output flag globals between cases.
func resetIOFlags() {
	flagInput, flagInFile, flagOutput = "", "", ""
	flagInDir, flagOutDir = "", ""
	flagRecursive = false
	flagANSI = ansiAuto
}

func newIOCmd() *cobra.Command {
	resetIOFlags()
	c := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	addIOFlags(c)
	return c
}

func TestFlagName(t *testing.T) {
	cases := map[string]string{
		"Alphabet":                  "alphabet",
		"Remove non-alphabet chars": "remove-non-alphabet-chars",
		`Treat "+" as space`:        "treat-as-space",
		"Encode all special chars":  "encode-all-special-chars",
		"Bytes per line":            "bytes-per-line",
		"0x with comma":             "0x-with-comma",
	}
	for in, want := range cases {
		if got := flagName(in); got != want {
			t.Errorf("flagName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveInputPositional(t *testing.T) {
	c := newIOCmd()
	got, err := resolveInput(c, []string{"Have", "a", "nice", "day."})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Have a nice day." {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInputFlag(t *testing.T) {
	c := newIOCmd()
	if err := c.Flags().Set("input", "hello"); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInput(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	// -i takes priority over positional args.
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInputFile(t *testing.T) {
	c := newIOCmd()
	path := filepath.Join(t.TempDir(), "in.txt")
	if err := os.WriteFile(path, []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("in-file", path); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInput(c, []string{"ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "from file" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInputStdin(t *testing.T) {
	c := newIOCmd()
	c.SetIn(strings.NewReader("piped input"))
	got, err := resolveInput(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "piped input" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInputFileDashIsStdin(t *testing.T) {
	c := newIOCmd()
	c.SetIn(strings.NewReader("via dash"))
	if err := c.Flags().Set("in-file", "-"); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInput(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	// --in-file - reads stdin (clig.dev: support - as a filename).
	if string(got) != "via dash" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteOutputDashIsStdout(t *testing.T) {
	c := newIOCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	flagOutput = "-"
	defer resetIOFlags()
	if err := writeOutput(c, []byte("result")); err != nil {
		t.Fatal(err)
	}
	// --output - writes stdout, byte-exact.
	if buf.String() != "result" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestWriteOutputToBuffer(t *testing.T) {
	c := newIOCmd()
	var buf bytes.Buffer
	c.SetOut(&buf)
	if err := writeOutput(c, []byte("result")); err != nil {
		t.Fatal(err)
	}
	// A non-terminal writer must stay byte-exact (no trailing newline).
	if buf.String() != "result" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestWriteOutputToFile(t *testing.T) {
	c := newIOCmd()
	path := filepath.Join(t.TempDir(), "out.txt")
	flagOutput = path
	defer resetIOFlags()
	if err := writeOutput(c, []byte("saved")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "saved" {
		t.Fatalf("got %q", b)
	}
}

func TestIsTerminal(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Fatal("bytes.Buffer should not be a terminal")
	}
	// A regular file is not a character device either.
	f, err := os.CreateTemp(t.TempDir(), "f")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if isTerminal(f) {
		t.Fatal("regular file should not be a terminal")
	}
}

// failWriter always fails, to exercise writeOutput's write-error path.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

// TestWriteOutputError covers writeOutput's error when the destination writer
// fails, and isTerminal's Stat-error path via a closed file.
func TestWriteOutputError(t *testing.T) {
	resetIOFlags()
	cmd := &cobra.Command{}
	cmd.SetOut(failWriter{})
	if err := writeOutput(cmd, []byte("data")); err == nil {
		t.Fatal("writeOutput to a failing writer: expected an error")
	}

	// isTerminal on a closed file: Stat fails, so it reports false.
	f, err := os.CreateTemp(t.TempDir(), "term")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if isTerminal(f) {
		t.Fatal("isTerminal(closed file) should be false")
	}
}

// TestPresentOutputDataURI checks --data-uri wraps output with its sniffed
// media type, for any operation's bytes rather than only the three whose
// CyberChef counterparts render in the browser.
func TestPresentOutputDataURI(t *testing.T) {
	t.Cleanup(func() { flagDataURI = false })
	flagDataURI = true
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16))
	got, err := presentOutput(png)
	if err != nil {
		t.Fatal(err)
	}
	want := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	if string(got) != want {
		t.Errorf("data URI = %q, want %q", got, want)
	}

	got, err = presentOutput([]byte("plain text"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "data:text/plain;base64,") {
		t.Errorf("text data URI = %q", got)
	}
}

// TestPresentOutputPreview checks --preview against each terminal protocol,
// and that it refuses politely when it cannot help.
func TestPresentOutputPreview(t *testing.T) {
	t.Cleanup(func() { flagPreview = false })
	flagPreview = true
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16))

	t.Run("iterm", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "iTerm.app")
		t.Setenv("KITTY_WINDOW_ID", "")
		got, err := presentOutput(png)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := termimage.Encode(termimage.ITerm, "image/png", png)
		if string(got) != string(want) {
			t.Error("iterm preview mismatch")
		}
	})

	t.Run("kitty", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("TERM", "xterm-kitty")
		got, err := presentOutput(png)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := termimage.Encode(termimage.Kitty, "image/png", png)
		if string(got) != string(want) {
			t.Error("kitty preview mismatch")
		}
	})

	t.Run("unsupported terminal", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "")
		t.Setenv("TERM", "dumb")
		t.Setenv("KITTY_WINDOW_ID", "")
		if _, err := presentOutput(png); err == nil {
			t.Error("expected an error on a terminal with no image support")
		}
	})

	t.Run("not an image", func(t *testing.T) {
		t.Setenv("TERM_PROGRAM", "iTerm.app")
		if _, err := presentOutput([]byte("just text")); err == nil {
			t.Error("expected an error previewing non-image output")
		}
	})
}

// TestPositionalFileGuard covers the guard on a positional argument that names
// an existing file: cchef treats positionals as literal text, so silently
// encoding the path instead of the file's contents would be a wrong answer
// rather than an error.
func TestPositionalFileGuard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.txt")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &cobra.Command{}
	addIOFlags(c)

	_, err := resolveInput(c, []string{path})
	if err == nil {
		t.Fatal("expected an error for a positional naming an existing file")
	}
	for _, want := range []string{path, "--in-file", "-i", "--"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}

	// Text that merely looks path-like but names nothing is still text.
	if got, err := resolveInput(c, []string{filepath.Join(dir, "absent.txt")}); err != nil {
		t.Errorf("non-existent path: %v", err)
	} else if string(got) != filepath.Join(dir, "absent.txt") {
		t.Errorf("non-existent path = %q", got)
	}

	// A directory is not readable as input either, so it gets the same guard.
	if _, err := resolveInput(c, []string{dir}); err == nil {
		t.Error("expected an error for a positional naming a directory")
	}

	// Several positionals join into one string; the guard checks each.
	if _, err := resolveInput(c, []string{"hello", path}); err == nil {
		t.Error("expected an error when any positional names a file")
	}
	if got, err := resolveInput(c, []string{"hello", "world"}); err != nil || string(got) != "hello world" {
		t.Errorf("plain positionals = %q, %v", got, err)
	}
}

// TestPositionalFileGuardAfterDash checks that `--` opts out of the guard: the
// user has said explicitly that what follows is text.
func TestPositionalFileGuardAfterDash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "msg.txt")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
	addIOFlags(c)
	c.SetArgs([]string{"--", path})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := resolveInput(c, []string{path})
	if err != nil {
		t.Fatalf("after --: %v", err)
	}
	if string(got) != path {
		t.Errorf("after -- = %q, want the literal path", got)
	}
}
