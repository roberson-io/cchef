package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

var (
	flagRecipeFile string
	flagRecipeExpr string
)

func init() {
	bakeCmd := &cobra.Command{
		Use:   "bake",
		Short: "Run a full recipe (JSON or Chef format) against the input",
		Long: "Execute a multi-operation recipe. The recipe may be supplied inline\n" +
			"with -e (JSON array or Chef format, auto-detected) or read from a file\n" +
			"with -r. Input is read from stdin, -i, or --in-file.",
		Example: "  echo hello | cchef bake -e \"To_Base64()\"\n" +
			"  cchef bake -r recipe.json -i hello",
		RunE: runBake,
	}
	bakeCmd.Flags().StringVarP(&flagRecipeExpr, "expr", "e", "", "recipe as an inline JSON or Chef string")
	bakeCmd.Flags().StringVarP(&flagRecipeFile, "recipe", "r", "", "path to a recipe file (JSON or Chef)")
	addIOFlags(bakeCmd)
	rootCmd.AddCommand(bakeCmd)
}

// rawRecipeText returns the unparsed recipe text from -e or -r.
func rawRecipeText() (string, error) {
	switch {
	case flagRecipeExpr != "":
		return flagRecipeExpr, nil
	case flagRecipeFile != "":
		b, err := os.ReadFile(flagRecipeFile) // #nosec G304 -- reads a user-specified recipe path by design (CLI file argument)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("no recipe given: use -e <expr> or -r <file>")
	}
}

// loadRecipeWithText resolves the recipe text from -e or -r, parses it, and
// returns both the parsed recipe and the raw text (so callers that also need the
// original format don't have to read it a second time).
func loadRecipeWithText() (core.Recipe, string, error) {
	text, err := rawRecipeText()
	if err != nil {
		return nil, "", err
	}
	r, err := core.ParseRecipeConfig(text)
	return r, text, err
}

// loadRecipe resolves the recipe text from -e or -r and parses it.
func loadRecipe() (core.Recipe, error) {
	r, _, err := loadRecipeWithText()
	return r, err
}

func runBake(cmd *cobra.Command, posArgs []string) error {
	recipe, err := loadRecipe()
	if err != nil {
		return err
	}
	in, err := resolveInput(cmd, posArgs)
	if err != nil {
		return err
	}
	out, err := recipe.Execute(core.NewDish(in, core.TypeString))
	if err != nil {
		return err
	}
	return writeOutput(cmd, out.Bytes())
}
