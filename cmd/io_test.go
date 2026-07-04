package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resetIOFlags clears the shared input/output flag globals between cases.
func resetIOFlags() {
	flagInput, flagInFile, flagOutput = "", "", ""
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
