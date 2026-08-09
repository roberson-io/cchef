package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

var flagConvertTo string

// The formats a recipe is read and written in.
const (
	formatJSON = "json"
	formatChef = "chef"
)

// formatRecipe renders a recipe in the named target format.
func formatRecipe(r core.Recipe, format string) (string, error) {
	switch format {
	case formatChef:
		return core.GeneratePrettyRecipe(r, false), nil
	case formatJSON:
		return core.MarshalRecipeJSON(r)
	}
	return "", fmt.Errorf("unknown target format %q (want %s or %s)", format, formatJSON, formatChef)
}

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
	addRecipeSourceFlags(convertCmd)
	convertCmd.Flags().StringVar(&flagConvertTo, "to", "", "target format: json or chef (default: the other format)")

	recipeCmd.AddCommand(convertCmd)
	addStageCommands(recipeCmd)
	rootCmd.AddCommand(recipeCmd)
}

func runConvert(cmd *cobra.Command, _ []string) error {
	src, err := loadRecipeSource(cmd)
	if err != nil {
		return err
	}
	recipe, raw := src.Recipe, src.Text

	target := flagConvertTo
	if target == "" {
		// Default to the opposite of the detected input format.
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" && trimmed[0] == '[' {
			target = formatChef
		} else {
			target = formatJSON
		}
	}

	out, err := formatRecipe(recipe, target)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), out)
	return err
}
