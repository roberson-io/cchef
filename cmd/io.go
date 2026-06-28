package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Global input/output flags shared by operation subcommands and bake/url.
var (
	flagInput  string // -i/--input: literal input string
	flagInFile string // --in-file: read input from a file
	flagOutput string // -o/--output: write output to a file
)

func addIOFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVarP(&flagInput, "input", "i", "", "input data as a literal string")
	f.StringVar(&flagInFile, "in-file", "", "read input from a file")
	f.StringVarP(&flagOutput, "output", "o", "", "write output to a file (default stdout)")
}

// resolveInput returns the input bytes, in priority order: --in-file, then
// -i/--input, then any positional arguments (joined with spaces), then stdin.
// Positional args make `cchef rot13 "Have a nice day."` work directly, while the
// stdin fallback lets operations chain through Unix pipes.
func resolveInput(cmd *cobra.Command, args []string) ([]byte, error) {
	switch {
	case flagInFile != "":
		return os.ReadFile(flagInFile)
	case cmd.Flags().Changed("input"):
		return []byte(flagInput), nil
	case len(args) > 0:
		return []byte(strings.Join(args, " ")), nil
	default:
		return io.ReadAll(cmd.InOrStdin())
	}
}

// writeOutput writes result bytes to the -o file or to stdout. When stdout is a
// terminal and the data does not already end in a newline, a trailing newline is
// added for readability. Piped/redirected output stays byte-exact so operations
// chain cleanly.
func writeOutput(cmd *cobra.Command, data []byte) error {
	if flagOutput != "" {
		return os.WriteFile(flagOutput, data, 0o644)
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

// isTerminal reports whether w is a character device (an interactive terminal).
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
