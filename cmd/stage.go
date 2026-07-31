package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

// A staged recipe is built up a step at a time and then run, in the way a
// commit is built up with `git add`. It lives in a file in the working
// directory so that different projects keep different recipes, and `bake`,
// `url` and `recipe convert` all fall back to it when given no recipe of their
// own.

const (
	// stageFileName is the working recipe in the current directory.
	stageFileName = ".cchef-recipe.json"
	// stageEnvVar overrides that location.
	stageEnvVar = "CCHEF_RECIPE"
)

// stagePath is where the staged recipe is kept.
func stagePath() string {
	if p := os.Getenv(stageEnvVar); p != "" {
		return p
	}
	return stageFileName
}

// loadStagedRecipe reads the staged recipe; nothing staged is an empty recipe
// rather than an error.
func loadStagedRecipe() (core.Recipe, error) {
	b, err := os.ReadFile(stagePath()) // #nosec G304 -- reads the staged recipe path, which the user chooses
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return core.ParseRecipeConfig(string(b))
}

// saveStagedRecipe writes the staged recipe back, in JSON so that a
// hand-editing user sees the arguments spelled out.
func saveStagedRecipe(r core.Recipe) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stagePath(), append(b, '\n'), 0o600)
}

// parseStageSpec turns what the user typed into recipe steps. A bare operation
// name (either spelling) stages that operation with its default arguments;
// anything else is parsed as a recipe expression, so a whole pipeline can be
// staged at once.
func parseStageSpec(spec string) (core.Recipe, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, errors.New("nothing to add")
	}
	if op, ok := lookupOperation(trimmed); ok {
		return core.Recipe{{Op: op.Meta().Name, Args: core.DefaultArgs(op.Args())}}, nil
	}
	r, err := core.ParseRecipeConfig(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%q is neither an operation name nor a valid recipe: %w", spec, err)
	}
	if len(r) == 0 {
		return nil, fmt.Errorf("%q adds no operations", spec)
	}
	return r, nil
}

// lookupOperation finds an operation by its CyberChef name or by the
// subcommand name, in either spelling.
func lookupOperation(name string) (core.Operation, bool) {
	if op, ok := core.Default.Get(name); ok {
		return op, true
	}
	for _, candidate := range []string{name, usSpelling(name), ukSpelling(name)} {
		for _, op := range core.Default.All() {
			if core.Kebab(op.Meta().Name) == candidate {
				return op, true
			}
		}
	}
	return nil, false
}

// stageAdd stages the operations named by spec, at position at (-1 appends).
func stageAdd(spec string, at int) error {
	steps, err := parseStageSpec(spec)
	if err != nil {
		return err
	}
	r, err := loadStagedRecipe()
	if err != nil {
		return err
	}
	if at < 0 {
		r = append(r, steps...)
	} else {
		if at > len(r) {
			return fmt.Errorf("position %d is past the end of a %d-step recipe", at, len(r))
		}
		r = slices.Insert(r, at, steps...)
	}
	return saveStagedRecipe(r)
}

// checkIndexes rejects any index outside the recipe, naming the first bad one.
func checkIndexes(r core.Recipe, idx []int) error {
	for _, i := range idx {
		if i < 0 || i >= len(r) {
			return fmt.Errorf("no step %d in a %d-step recipe", i, len(r))
		}
	}
	return nil
}

// stageRemove drops the numbered steps. Every index refers to the recipe as it
// was before the command, so removing several at once behaves predictably.
func stageRemove(idx []int) error {
	r, err := loadStagedRecipe()
	if err != nil {
		return err
	}
	if err := checkIndexes(r, idx); err != nil {
		return err
	}
	kept := make(core.Recipe, 0, len(r))
	for i, step := range r {
		if !slices.Contains(idx, i) {
			kept = append(kept, step)
		}
	}
	return saveStagedRecipe(kept)
}

// stageMove moves one step to another position.
func stageMove(from, to int) error {
	r, err := loadStagedRecipe()
	if err != nil {
		return err
	}
	if err := checkIndexes(r, []int{from, to}); err != nil {
		return err
	}
	step := r[from]
	r = slices.Delete(r, from, from+1)
	r = slices.Insert(r, to, step)
	return saveStagedRecipe(r)
}

// stageToggle flips whether the numbered steps run.
func stageToggle(idx []int) error {
	r, err := loadStagedRecipe()
	if err != nil {
		return err
	}
	if err := checkIndexes(r, idx); err != nil {
		return err
	}
	for _, i := range idx {
		r[i].Disabled = !r[i].Disabled
	}
	return saveStagedRecipe(r)
}

// stageClear discards the staged recipe. Clearing nothing is not an error.
func stageClear() error {
	err := os.Remove(stagePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// formatStage renders the staged recipe as a numbered list, so the indexes the
// other commands take are the ones on screen.
func formatStage(r core.Recipe) string {
	if len(r) == 0 {
		return "No recipe staged (empty). Add a step with `cchef recipe add <operation>`."
	}
	var sb strings.Builder
	for i, step := range r {
		fmt.Fprintf(&sb, "%d  %s", i, step.Op)
		if args := formatStageArgs(step.Args); args != "" {
			fmt.Fprintf(&sb, " %s", args)
		}
		if step.Disabled {
			sb.WriteString("   (disabled)")
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// formatStageArgs renders a step's arguments compactly for the listing.
func formatStageArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		switch v := a.(type) {
		case string:
			parts = append(parts, strconv.Quote(v))
		case core.ToggleString:
			parts = append(parts, fmt.Sprintf("%s:%s", v.Option, strconv.Quote(v.Value)))
		default:
			parts = append(parts, fmt.Sprint(v))
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// parseIndexes reads the step numbers a command was given.
func parseIndexes(args []string) ([]int, error) {
	idx := make([]int, len(args))
	for i, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil {
			return nil, fmt.Errorf("%q is not a step number", a)
		}
		idx[i] = n
	}
	return idx, nil
}

// showStage prints the staged recipe.
func showStage(cmd *cobra.Command) error {
	r, err := loadStagedRecipe()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), formatStage(r))
	return err
}

// flagStageAt is the position `recipe add --at` inserts at.
var flagStageAt int

// addStageCommands registers the staging subcommands under `cchef recipe`.
func addStageCommands(recipeCmd *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add <operation|recipe>",
		Short: "Stage an operation (or a whole recipe) as the next step",
		Long: "Name an operation to stage it with its default arguments, or give a\n" +
			"recipe expression in JSON or Chef format to stage it exactly.",
		Example: "  cchef recipe add to-base64\n" +
			"  cchef recipe add \"To_Hex('Colon')\"\n" +
			"  cchef recipe add rot13 --at 0",
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return stageAdd(args[0], flagStageAt)
		},
	}
	addCmd.Flags().IntVar(&flagStageAt, "at", -1, "insert at this position instead of appending")

	rmCmd := &cobra.Command{
		Use:     "rm <step>...",
		Short:   "Remove staged steps by number",
		Example: "  cchef recipe rm 2\n  cchef recipe rm 0 3",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			idx, err := parseIndexes(args)
			if err != nil {
				return err
			}
			return stageRemove(idx)
		},
	}

	moveCmd := &cobra.Command{
		Use:     "move <from> <to>",
		Short:   "Move a staged step to another position",
		Example: "  cchef recipe move 2 0",
		Args:    cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			idx, err := parseIndexes(args)
			if err != nil {
				return err
			}
			return stageMove(idx[0], idx[1])
		},
	}

	toggleCmd := &cobra.Command{
		Use:     "toggle <step>...",
		Short:   "Turn staged steps off or back on",
		Example: "  cchef recipe toggle 1",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			idx, err := parseIndexes(args)
			if err != nil {
				return err
			}
			return stageToggle(idx)
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the staged recipe, numbered",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return showStage(cmd) },
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Discard the staged recipe",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return stageClear() },
	}

	recipeCmd.AddCommand(addCmd, rmCmd, moveCmd, toggleCmd, showCmd, clearCmd)
}
