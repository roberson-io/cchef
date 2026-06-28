package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

var flagConvertTo string

func init() {
	recipeCmd := &cobra.Command{
		Use:   "recipe",
		Short: "Work with recipes (convert between formats)",
	}

	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert a recipe between JSON and Chef formats",
		Long: "Read a recipe (with -e or -r, JSON or Chef format auto-detected) and\n" +
			"print it in the other format. Use --to to force the target format.",
		Example: "  cchef recipe convert -e \"To_Base64()\" --to json\n" +
			"  cchef recipe convert -r recipe.json --to chef",
		RunE: runConvert,
	}
	convertCmd.Flags().StringVarP(&flagRecipeExpr, "expr", "e", "", "recipe as an inline JSON or Chef string")
	convertCmd.Flags().StringVarP(&flagRecipeFile, "recipe", "r", "", "path to a recipe file (JSON or Chef)")
	convertCmd.Flags().StringVar(&flagConvertTo, "to", "", "target format: json or chef (default: the other format)")

	recipeCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runConvert(cmd *cobra.Command, _ []string) error {
	recipe, err := loadRecipe()
	if err != nil {
		return err
	}

	target := flagConvertTo
	if target == "" {
		// Default to the opposite of the detected input format.
		raw, rErr := rawRecipeText()
		if rErr != nil {
			return rErr
		}
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" && trimmed[0] == '[' {
			target = "chef"
		} else {
			target = "json"
		}
	}

	switch target {
	case "chef":
		_, err = fmt.Fprintln(cmd.OutOrStdout(), core.GeneratePrettyRecipe(recipe, false))
	case "json":
		b, mErr := json.MarshalIndent(recipe, "", "  ")
		if mErr != nil {
			return mErr
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
	default:
		return fmt.Errorf("unknown target format %q (want json or chef)", target)
	}
	return err
}
