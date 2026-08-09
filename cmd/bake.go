package cmd

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

var (
	flagRecipeFile string
	flagRecipeExpr string
	flagRecipeURL  string
)

// recipeSource is a resolved recipe: the recipe itself, the text it was written
// as (so `recipe convert` can pick the other format), and any input a share URL
// carried alongside it.
type recipeSource struct {
	Recipe core.Recipe
	Text   string
	Input  []byte
}

// addRecipeSourceFlags registers the ways a command can be given a recipe.
func addRecipeSourceFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVarP(&flagRecipeExpr, "expr", "e", "", "recipe as an inline JSON or Chef string")
	f.StringVarP(&flagRecipeFile, "recipe", "r", "", "path to a recipe file (JSON or Chef), or - for stdin")
	f.StringVar(&flagRecipeURL, "from-url", "", "read the recipe from a CyberChef share URL")
}

// resetRecipeSourceFlags clears the recipe-source flags. The flags are package
// globals, so a process running several commands has to put them back.
func resetRecipeSourceFlags() {
	flagRecipeExpr, flagRecipeFile, flagRecipeURL = "", "", ""
}

// loadRecipeSource resolves the recipe named by -e, -r or --from-url, falling
// back to the staged recipe when the command line names none. Naming more than
// one is refused rather than resolved by precedence, so a command line that
// says two things does not quietly do one of them.
func loadRecipeSource(cmd *cobra.Command) (recipeSource, error) {
	named := map[string]string{}
	if flagRecipeExpr != "" {
		named["-e"] = flagRecipeExpr
	}
	if flagRecipeFile != "" {
		named["-r"] = flagRecipeFile
	}
	if flagRecipeURL != "" {
		named["--from-url"] = flagRecipeURL
	}
	if len(named) > 1 {
		return recipeSource{}, fmt.Errorf("give only one of %s", strings.Join(slices.Sorted(maps.Keys(named)), ", "))
	}

	switch {
	case flagRecipeURL != "":
		r, in, err := core.ParseURL(flagRecipeURL)
		if err != nil {
			return recipeSource{}, err
		}
		return recipeSource{Recipe: r, Text: core.GeneratePrettyRecipe(r, false), Input: in}, nil
	case flagRecipeExpr != "":
		return parseRecipeText(flagRecipeExpr)
	case flagRecipeFile != "":
		text, err := readRecipeFile(cmd, flagRecipeFile)
		if err != nil {
			return recipeSource{}, err
		}
		return parseRecipeText(text)
	}

	// Fall back to the staged recipe, so a recipe built up with
	// `cchef recipe add` runs without naming it again.
	staged, err := loadStagedRecipe()
	if err != nil {
		return recipeSource{}, err
	}
	if len(staged) == 0 {
		return recipeSource{}, fmt.Errorf("no recipe given: use -e <expr>, -r <file>, --from-url <url>, or stage one with `cchef recipe add`")
	}
	text, err := core.MarshalRecipeJSON(staged)
	if err != nil {
		return recipeSource{}, err
	}
	return recipeSource{Recipe: staged, Text: text}, nil
}

// readRecipeFile reads a recipe file, or stdin when the path is "-".
func readRecipeFile(cmd *cobra.Command, path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		return string(b), err
	}
	b, err := os.ReadFile(path) // #nosec G304 -- reads a user-specified recipe path by design (CLI file argument)
	return string(b), err
}

// parseRecipeText parses recipe text, keeping the text so a caller can tell
// which format it was written in.
func parseRecipeText(text string) (recipeSource, error) {
	r, err := core.ParseRecipeConfig(text)
	if err != nil {
		return recipeSource{}, err
	}
	return recipeSource{Recipe: r, Text: text}, nil
}

func init() {
	bakeCmd := &cobra.Command{
		Use:   "bake",
		Short: "Run a full recipe (JSON or Chef format) against the input",
		Long: "Execute a multi-operation recipe. The recipe may be supplied inline\n" +
			"with -e (JSON array or Chef format, auto-detected), read from a file\n" +
			"with -r (or - for stdin), or taken from a CyberChef share URL with\n" +
			"--from-url. Input is read from stdin, -i, or --in-file; a share URL's\n" +
			"own input is used when the command line names none.",
		Example: "  echo hello | cchef bake -e \"To_Base64()\"\n" +
			"  cchef bake -r recipe.json -i hello\n" +
			"  cchef bake --from-url \"https://gchq.github.io/CyberChef/#recipe=ROT13()&input=SGVsbG8\"",
		RunE: runBake,
	}
	addRecipeSourceFlags(bakeCmd)
	addIOFlags(bakeCmd)
	rootCmd.AddCommand(bakeCmd)
}

func runBake(cmd *cobra.Command, posArgs []string) error {
	src, err := loadRecipeSource(cmd)
	if err != nil {
		return err
	}
	// A share URL can carry the input it was built with, which is used when the
	// command line names none of its own.
	return runRecipeIOWithInput(cmd, posArgs, src.Recipe, src.Input)
}
