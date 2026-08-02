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
	Example: "  echo -n hello | cchef to-base64\n" +
		"  cchef to-hex --delimiter Colon hello\n" +
		"  echo -n hello | cchef bake -e \"To_Base64()To_Hex()\"\n" +
		"  cchef list                 # show every operation by category",
	Version:       version + " (operations aligned with CyberChef " + alignedCyberChef + ")",
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
