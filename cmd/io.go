package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/roberson-io/cchef/internal/termimage"
)

// Global input/output flags shared by operation subcommands and bake/url.
var (
	flagInput     string // -i/--input: literal input string
	flagInFile    string // --in-file: read input from a file
	flagOutput    string // -o/--output: write output to a file
	flagInDir     string // --in-dir: run the recipe over every file in a directory
	flagOutDir    string // --out-dir: mirror per-file output into a directory
	flagRecursive bool   // --recursive: with --in-dir, walk subdirectories
	flagPreview   bool   // --preview: show image output inline in the terminal
	flagDataURI   bool   // --data-uri: write byte output as a data: URI
	flagANSI      string // --ansi: auto/always/never for highlighted output
)

func addIOFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVarP(&flagInput, "input", "i", "", "input data as a literal string")
	f.StringVar(&flagInFile, "in-file", "", "read input from a file")
	f.StringVarP(&flagOutput, "output", "o", "", "write output to a file (default stdout)")
	f.StringVar(&flagInDir, "in-dir", "", "run the recipe over every file in a directory")
	f.StringVar(&flagOutDir, "out-dir", "", "write output into a directory (with --in-dir, or for operations producing several files)")
	f.BoolVar(&flagRecursive, "recursive", false, "with --in-dir, recurse into subdirectories")
	f.BoolVar(&flagPreview, "preview", false, "display image output inline (iTerm2 or kitty terminals)")
	f.BoolVar(&flagDataURI, "data-uri", false, "write output as a data: URI")
	addANSIFlag(cmd)
}

// resolveInput returns the input bytes, in priority order: --in-file, then
// -i/--input, then any positional arguments (joined with spaces), then stdin.
// Positional args make `cchef rot13 "Have a nice day."` work directly, while the
// stdin fallback lets operations chain through Unix pipes.
func resolveInput(cmd *cobra.Command, args []string) ([]byte, error) {
	switch {
	case flagInFile == "-":
		return io.ReadAll(cmd.InOrStdin())
	case flagInFile != "":
		return os.ReadFile(flagInFile) // #nosec G304 -- reads a user-specified input path by design (CLI file argument)
	case cmd.Flags().Changed("input"):
		return []byte(flagInput), nil
	case len(args) > 0:
		if err := guardPositionalFile(cmd, args); err != nil {
			return nil, err
		}
		return []byte(strings.Join(args, " ")), nil
	case inputIsInteractive(cmd.InOrStdin()):
		// Reading stdin here would wait for someone to type and press Ctrl-D,
		// with nothing on screen to say so. Say what to do instead.
		return nil, errNoInput(cmd)
	default:
		return io.ReadAll(cmd.InOrStdin())
	}
}

// inputNamed reports whether the command line named an input source. It does
// not count stdin, which is always there whether or not anyone meant it.
func inputNamed(cmd *cobra.Command, args []string) bool {
	return flagInFile != "" || cmd.Flags().Changed("input") || len(args) > 0
}

// resolveInputOr returns the input, falling back to def when the command line
// named none. A def of nil means the usual stdin handling applies.
func resolveInputOr(cmd *cobra.Command, args []string, def []byte) ([]byte, error) {
	if def != nil && !inputNamed(cmd, args) {
		return def, nil
	}
	return resolveInput(cmd, args)
}

// inputIsInteractive reports whether reading r would wait on a person. The test
// is a real terminal rather than a character device, because /dev/null is a
// character device and redirecting from it means empty input.
func inputIsInteractive(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// errNoInput reports that no input was given and none can be read.
func errNoInput(cmd *cobra.Command) error {
	return fmt.Errorf("no input given, and stdin is a terminal.\n"+
		"Give text with -i, a file with --in-file, a directory with --in-dir, or pipe data in:\n"+
		"  echo -n hello | cchef %s", cmd.Name())
}

// writeOutput writes result bytes to the -o file or to stdout. When stdout is a
// terminal and the data does not already end in a newline, a trailing newline is
// added for readability. Piped/redirected output stays byte-exact so operations
// chain cleanly.
func writeOutput(cmd *cobra.Command, data []byte) error {
	data, err := presentOutput(data)
	if err != nil {
		return err
	}
	if flagOutput != "" && flagOutput != "-" {
		return os.WriteFile(flagOutput, data, 0o644) // #nosec G306 -- 0644 is conventional for CLI output files written on the user's behalf
	}
	out := cmd.OutOrStdout()
	if _, err := out.Write(data); err != nil {
		return err
	}
	if isTerminal(out) && (len(data) == 0 || data[len(data)-1] != '\n') {
		_, err := fmt.Fprintln(out)
		return err
	}
	return nil
}

// isTerminal reports whether w is an interactive terminal.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// presentOutput applies the presentation flags to an operation's output. Both
// are cchef's answer to the previews CyberChef draws in the browser, and they
// work for any operation that emits bytes rather than only the ones whose
// CyberChef counterpart renders in the page.
func presentOutput(data []byte) ([]byte, error) {
	if flagDataURI {
		return []byte("data:" + contentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)), nil
	}
	if !flagPreview {
		return data, nil
	}
	mime := contentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return nil, fmt.Errorf("--preview needs image output, but this output is %s", mime)
	}
	proto := termimage.Detect()
	if proto == termimage.None {
		return nil, errors.New("--preview needs a terminal with inline-image support (iTerm2, WezTerm or kitty)")
	}
	return termimage.Encode(proto, mime, data)
}

// contentType sniffs the media type of output bytes.
func contentType(data []byte) string {
	return strings.SplitN(http.DetectContentType(data), ";", 2)[0]
}

// guardPositionalFile refuses a positional argument that names something on
// disk. Positionals are literal text, so a path given as one would otherwise be
// encoded as the path itself — a plausible-looking wrong answer rather than a
// failure. Anything after a `--` has been declared to be text and is left
// alone.
func guardPositionalFile(cmd *cobra.Command, args []string) error {
	dash := cmd.ArgsLenAtDash()
	for i, a := range args {
		if dash >= 0 && i >= dash {
			break
		}
		if _, err := os.Stat(a); err != nil {
			continue
		}
		return fmt.Errorf("%q names a file, but positional input is used as literal text.\n"+
			"Read the file with --in-file %s, keep the text with -i %q, or pass -- first",
			a, a, a)
	}
	return nil
}
