package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execRoot runs the root command with the given args and returns stdout. It
// resets shared flag globals first. Tests use positional input (never -i) so
// cobra's persistent per-flag "changed" state between Execute calls is not
// consulted by resolveInput.
func execRoot(t *testing.T, args ...string) string {
	t.Helper()
	resetIOFlags()
	flagRecipeExpr, flagRecipeFile, flagConvertTo = "", "", ""

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute(%v): %v\noutput: %s", args, err, buf.String())
	}
	return buf.String()
}

// execRootErr runs the root command expecting an error, returning it. Mirrors
// execRoot's flag reset so cases stay independent.
func execRootErr(t *testing.T, args ...string) error {
	t.Helper()
	resetIOFlags()
	flagRecipeExpr, flagRecipeFile, flagConvertTo = "", "", ""

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// TestExecuteErrorPaths covers the user-facing CLI failure modes: missing or
// malformed recipes, unknown operations/formats, bad argument types, and a
// missing input file. Each is surfaced as a non-nil error (root sets
// SilenceErrors, so Execute returns it rather than printing).
func TestExecuteErrorPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.json")
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		// Positional input (never -i): -i leaves cobra's per-flag "changed" state
		// set across Execute calls, which would poison later tests' resolveInput.
		{"no recipe given", []string{"bake", "hello"}, "no recipe given"},
		{"missing recipe file", []string{"bake", "-r", missing, "hello"}, "no such file"},
		{"broken JSON recipe", []string{"bake", "-e", `[{"op":`, "hello"}, "parse JSON recipe"},
		{"unknown operation", []string{"bake", "-e", `[{"op":"Nonexistent Op","args":[]}]`, "hello"}, "unknown operation"},
		{"bad arg type", []string{"bake", "-e", `[{"op":"To Base64","args":[123]}]`, "hello"}, "Alphabet must be text."},
		{"convert unknown target", []string{"recipe", "convert", "-e", "To_Hex()", "--to", "xml"}, "unknown target format"},
		{"missing input file", []string{"reverse", "--in-file", missing}, "no such file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := execRootErr(t, c.args...)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got err %v, want containing %q", err, c.wantErr)
			}
		})
	}
}

func TestExecuteOperations(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"to-base64", []string{"to-base64", "hello"}, "aGVsbG8="},
		{"to-base64 custom alphabet", []string{"to-base64", "--alphabet", "A-Za-z0-9-_", "hello"}, "aGVsbG8"},
		{"rot13 bool+number flags", []string{"rot13", "Hello, World!"}, "Uryyb, Jbeyq!"},
		{"to-hex option flag", []string{"to-hex", "--delimiter", "Colon", "Hello"}, "48:65:6c:6c:6f"},
		{"md5", []string{"md5", "Hello, World!"}, "65a8e27d8879283831b664bd8b7f0ad4"},
		{"reverse", []string{"reverse", "abc"}, "cba"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := execRoot(t, c.args...); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestExecuteXORToggleStringFlag(t *testing.T) {
	// Exercises the toggleString flag getter (--key / --key-type). Output is raw
	// bytes; we only assert the command runs without error.
	_ = execRoot(t, "xor", "--key", "42", "--key-type", "Hex", "hello")
}

func TestExecuteBakeChef(t *testing.T) {
	if got := execRoot(t, "bake", "-e", "To_Base64()", "hello"); got != "aGVsbG8=" {
		t.Fatalf("got %q", got)
	}
}

func TestExecuteURL(t *testing.T) {
	got := execRoot(t, "url", "-e", "To_Hex()", "-i", "hello")
	want := "https://gchq.github.io/CyberChef/#recipe=To_Hex()&input=aGVsbG8\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	// url uses -i, which leaves the flag in a "changed" state; reset for safety.
	resetIOFlags()
}

func TestExecuteRecipeConvertToJSON(t *testing.T) {
	got := execRoot(t, "recipe", "convert", "-e", "To_Base64('A-Za-z0-9+/=')", "--to", "json")
	if !strings.Contains(got, `"op": "To Base64"`) {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestExecuteRecipeConvertToChef(t *testing.T) {
	got := execRoot(t, "recipe", "convert", "-e", `[{"op":"To Hex","args":["Space"]}]`, "--to", "chef")
	if strings.TrimSpace(got) != "To_Hex('Space')" {
		t.Fatalf("got %q", got)
	}
}

func TestExecuteBakeFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recipe.json")
	if err := os.WriteFile(path, []byte(`[{"op":"To Base64","args":["A-Za-z0-9+/="]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := execRoot(t, "bake", "-r", path, "hello"); got != "aGVsbG8=" {
		t.Fatalf("got %q", got)
	}
}

func TestExecuteRecipeConvertAutoDetect(t *testing.T) {
	// No --to: a Chef-format input should default to JSON output.
	got := execRoot(t, "recipe", "convert", "-e", "To_Hex('Space')")
	if !strings.Contains(got, `"op": "To Hex"`) {
		t.Fatalf("expected JSON output, got: %s", got)
	}
}

func TestExecuteList(t *testing.T) {
	got := execRoot(t, "list")
	// Grouped by category now, showing the kebab subcommand + a short summary.
	for _, want := range []string{"to-base64", "Data format", "Networking", "Utils", "Hashing"} {
		if !strings.Contains(got, want) {
			t.Fatalf("list output missing %q:\n%s", want, got)
		}
	}
}

// TestExecuteMoreErrorPaths covers command error/branch paths not in the primary
// error test: convert's recipe-load failure, bake's runtime op error, url's
// recipe-load and input-resolution failures, and convert's format auto-detection.
func TestExecuteMoreErrorPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.json")
	// convert with an unparseable recipe.
	if err := execRootErr(t, "recipe", "convert", "-e", `[{"op":`); err == nil {
		t.Error("convert bad recipe: expected an error")
	}
	// bake with a missing input file (resolveInput error).
	if err := execRootErr(t, "bake", "-e", "To_Hex()", "--in-file", missing); err == nil {
		t.Error("bake missing input: expected an error")
	}
	// A generated operation subcommand with a missing input file (resolveInput error).
	if err := execRootErr(t, "to-hex", "--in-file", missing); err == nil {
		t.Error("op missing input: expected an error")
	}
	// A generated operation subcommand whose operation fails at runtime.
	if err := execRootErr(t, "parse-ipv6-address", "not-an-address"); err == nil {
		t.Error("op runtime error: expected an error")
	}
	// url with an unparseable recipe.
	if err := execRootErr(t, "url", "-e", `[{"op":`); err == nil {
		t.Error("url bad recipe: expected an error")
	}
	// url with a missing input file.
	if err := execRootErr(t, "url", "-e", "To_Hex()", "--in-file", missing); err == nil {
		t.Error("url missing input: expected an error")
	}
	resetIOFlags()
}

// TestExecuteRecipeConvertJSONAutoDetect covers convert's target detection for a
// JSON-format recipe (leading '['): with no --to it defaults to Chef output.
func TestExecuteRecipeConvertJSONAutoDetect(t *testing.T) {
	got := execRoot(t, "recipe", "convert", "-e", `[{"op":"To Hex","args":["Space"]}]`)
	if !strings.Contains(got, "To_Hex(") {
		t.Fatalf("expected Chef output, got: %s", got)
	}
}
