package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

func init() {
	urlCmd := &cobra.Command{
		Use:   "url",
		Short: "Print a CyberChef share URL for a recipe (and optional input)",
		Long: "Build a CyberChef URL that opens the given recipe, with the input\n" +
			"pre-loaded if one is supplied. The recipe is given with -e or -r,\n" +
			"exactly like `cchef bake`.\n\n" +
			"Links point at the public instance unless --base-url, $CCHEF_BASE_URL\n" +
			"or base-url in the config file names another.",
		Example: "  cchef url -e \"To_Hex()\" -i hello",
		RunE: func(cmd *cobra.Command, posArgs []string) error {
			base, err := resolveBaseURL(cmd)
			if err != nil {
				return err
			}
			src, err := loadRecipeSource(cmd)
			if err != nil {
				return err
			}
			recipe := src.Recipe
			// Input is optional for url: read it only when it was named, or take
			// the one a share URL carried so re-pointing a link keeps it.
			input := src.Input
			if inputNamed(cmd, posArgs) {
				input, err = resolveInput(cmd, posArgs)
				if err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), core.BuildURL(base, recipe, input))
			return err
		},
	}
	addRecipeSourceFlags(urlCmd)
	addBaseURLFlag(urlCmd)
	addIOFlags(urlCmd)
	rootCmd.AddCommand(urlCmd)
}
