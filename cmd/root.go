// Package cmd wires the cchef cobra command tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base `cchef` command. Subcommands (one per operation, plus
// bake/url/recipe/list) are attached during package init.
var rootCmd = &cobra.Command{
	Use:   "cchef",
	Short: "CyberChef on the command line",
	Long: "cchef is a CLI port of the CyberChef data-transformation engine.\n" +
		"Each operation is a subcommand that reads stdin and writes stdout, so\n" +
		"operations chain through Unix pipes. Use `cchef bake` to run a full\n" +
		"recipe (JSON or Chef format) and `cchef url` to share it.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "cchef:", err)
		os.Exit(1)
	}
}
