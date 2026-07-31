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
		Short: "Build up a recipe step by step, and convert between formats",
		Long: "Stage a recipe one operation at a time, then run it with `cchef bake`.\n" +
			"The staged recipe lives in " + stageFileName + " in the working directory\n" +
			"(override with the " + stageEnvVar + " environment variable), so bake, url and\n" +
			"recipe convert all use it when given no recipe of their own.",
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
	addStageCommands(recipeCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runConvert(cmd *cobra.Command, _ []string) error {
	recipe, raw, err := loadRecipeWithText()
	if err != nil {
		return err
	}

	target := flagConvertTo
	if target == "" {
		// Default to the opposite of the detected input format.
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
