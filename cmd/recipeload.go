package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

// newLoadCmd builds `cchef recipe load`, which replaces the staged recipe with
// one read from somewhere else.
func newLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Replace the staged recipe with one read from a file, string or URL",
		Long: "Read a whole recipe and stage it, replacing whatever was staged before.\n" +
			"The recipe comes from -e, -r (a file, or - for stdin), or --from-url;\n" +
			"exactly one of them. Use `cchef recipe add` to append instead.\n\n" +
			"Export the other way with `cchef recipe convert`, which prints the\n" +
			"staged recipe when given no recipe of its own.",
		Example: "  cchef recipe load -e \"To_Base64()To_Hex()\"\n" +
			"  cchef recipe load -r recipe.json\n" +
			"  cchef recipe load --from-url \"https://gchq.github.io/CyberChef/#recipe=ROT13()\"",
		Args: cobra.NoArgs,
		RunE: runLoad,
	}
	addRecipeSourceFlags(cmd)
	return cmd
}

// runLoad stages the recipe the named source holds. The recipe is read and
// checked before the stage file is written, so a load that fails leaves the
// recipe that was already staged where it was.
func runLoad(cmd *cobra.Command, _ []string) error {
	src, err := loadRecipeSourceNoStage(cmd)
	if err != nil {
		return err
	}
	if len(src.Recipe) == 0 {
		return fmt.Errorf("that recipe has no operations in it")
	}
	if err := validateRecipeOps(src.Recipe); err != nil {
		return err
	}
	if err := saveStagedRecipe(src.Recipe); err != nil {
		return err
	}
	// The recipe is staged, but an input a share URL carried has nowhere to go:
	// say so rather than dropping it silently.
	if len(src.Input) > 0 {
		_, err = fmt.Fprintln(cmd.ErrOrStderr(),
			"note: the URL also carries input; run it directly with `cchef bake --from-url ...`")
	}
	return err
}

// loadRecipeSourceNoStage resolves a recipe from the command line only. Falling
// back to the staged recipe would make `cchef recipe load` with no arguments
// stage what is already staged, which says nothing and hides the mistake.
func loadRecipeSourceNoStage(cmd *cobra.Command) (recipeSource, error) {
	if flagRecipeExpr == "" && flagRecipeFile == "" && flagRecipeURL == "" {
		return recipeSource{}, fmt.Errorf("no recipe given: use -e <expr>, -r <file> (or -r -), or --from-url <url>")
	}
	return loadRecipeSource(cmd)
}

// validateRecipeOps rejects a recipe naming an operation cchef does not have,
// so an imported recipe fails where it is read rather than where it is run.
func validateRecipeOps(r core.Recipe) error {
	for i, step := range r {
		if _, ok := core.Default.Get(step.Op); !ok {
			return fmt.Errorf("step %d names an operation cchef does not have: %q", i+1, step.Op)
		}
	}
	return nil
}
