package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Fork{})
}

// forkSubRecipe collects the steps a Fork or Subsection at the current position
// governs: everything below it up to the Merge that closes it. A nested Fork or
// Subsection opens another level, so that level's Merge does not close this one,
// and a Merge with "merge all" ticked closes every level at once.
func forkSubRecipe(state *core.FlowState) []core.RecipeOp {
	var sub []core.RecipeOp
	depth := 1
	for i := state.Progress + 1; i < len(state.Steps); i++ {
		step := state.Steps[i]
		if step.Op == "Merge" && !step.Disabled {
			depth--
			if depth == 0 || mergesAll(step) {
				break
			}
			sub = append(sub, step)
			continue
		}
		if step.Op == "Fork" || step.Op == "Subsection" {
			depth++
		}
		sub = append(sub, step)
	}
	return sub
}

// mergesAll reads a Merge step's "Merge All" argument, which defaults to true
// when the step is written without one. Anything but a boolean there is refused
// when the Merge itself runs, so the scan need not account for it.
func mergesAll(step core.RecipeOp) bool {
	if len(step.Args) == 0 {
		return true
	}
	merge, _ := step.Args[0].(bool)
	return merge
}

// runTranche runs the sub-recipe over one piece of the data, returning what it
// became and how many steps ran. A failure is reported unless it is to be
// ignored, in which case the piece is left as it was.
func runTranche(state *core.FlowState, sub []core.RecipeOp, piece string, ignoreErrors bool) (string, int, error) {
	out, ran, err := state.RunSteps(sub, core.NewDish([]byte(piece), core.TypeString))
	if err != nil {
		if !ignoreErrors {
			return "", 0, err
		}
		return piece, len(sub), nil
	}
	return out.String(), ran, nil
}

// parseFlowDelim reads a delimiter argument, which carries escape sequences as
// text (`\n` for a newline) the way a recipe writes them.
func parseFlowDelim(s string) string {
	return strings.NewReplacer(
		`\n`, "\n", `\r`, "\r", `\t`, "\t", `\0`, "\x00", `\\`, `\`,
	).Replace(s)
}

// Fork splits the data and runs the rest of the recipe over each piece. Ported
// from CyberChef Fork.mjs.
type Fork struct{}

// Meta returns the operation metadata.
func (Fork) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Fork",
		Module:      "Default",
		Description: "Split the input data up based on the specified delimiter and run all subsequent operations on each branch separately.\n\nFor example, to decode multiple Base64 strings, enter them all on separate lines then add the 'Fork' and 'From Base64' operations to the recipe. Each string will be decoded separately.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the delimiters and whether a failing branch stops the recipe.
func (Fork) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Split delimiter", Type: core.ArgString, Value: `\n`},
		{Name: "Merge delimiter", Type: core.ArgString, Value: `\n`},
		{Name: "Ignore errors", Type: core.ArgBoolean, Value: false},
	}
}

// Run splits and rejoins the data; outside a recipe there are no steps to run
// over the pieces, so this replaces the split delimiter with the merge one.
func (Fork) Run(in *core.Dish, args []any) (*core.Dish, error) {
	split := parseFlowDelim(args[0].(string))
	merge := parseFlowDelim(args[1].(string))
	joined := strings.Join(strings.Split(in.String(), split), merge)
	return core.NewDish([]byte(joined), core.TypeString), nil
}

// RunFlow runs the steps below over each piece and rejoins the results.
func (Fork) RunFlow(state *core.FlowState) error {
	split := parseFlowDelim(state.Args[0].(string))
	merge := parseFlowDelim(state.Args[1].(string))
	ignoreErrors := state.Args[2].(bool)

	sub := forkSubRecipe(state)

	var pieces []string
	if input := state.Dish.String(); input != "" {
		pieces = strings.Split(input, split)
	}

	outputs := make([]string, 0, len(pieces))
	ran := 0
	for _, piece := range pieces {
		out, n, err := runTranche(state, sub, piece, ignoreErrors)
		if err != nil {
			return err
		}
		outputs = append(outputs, out)
		ran = n
	}

	state.Dish = core.NewDish([]byte(strings.Join(outputs, merge)), core.TypeString)
	// The steps just run are consumed; execution continues after them, which is
	// the Merge that closed this Fork if there was one.
	state.Progress += ran
	return nil
}
