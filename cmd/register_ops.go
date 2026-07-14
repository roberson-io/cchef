package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
	_ "github.com/roberson-io/cchef/internal/ops" // register operations
)

func init() {
	for _, op := range core.Default.All() {
		rootCmd.AddCommand(buildOpCommand(op))
	}
}

// reservedFlagRenames gives descriptive flag names to operation arguments whose
// derived name would otherwise collide with a global IO flag. The cipher ops'
// "Input"/"Output" Raw-vs-Hex selectors become --input-format/--output-format.
var reservedFlagRenames = map[string]string{
	"input":  "input-format",
	"output": "output-format",
}

// flagGetter reads the current value of a flag and returns it as the canonical
// argument value for its ArgDef.
type flagGetter func(*cobra.Command) (any, error)

// buildOpCommand turns a registered operation into a cobra subcommand. Each of
// the operation's arguments becomes a flag; the command reads input (stdin /
// -i / --in-file), runs a one-operation recipe, and writes the result.
func buildOpCommand(op core.Operation) *cobra.Command {
	meta := op.Meta()
	cmd := &cobra.Command{
		Use:     core.Kebab(meta.Name),
		Aliases: opAliases[meta.Name],
		Short:   summaryOf(meta),
		Long:    fmt.Sprintf("%s\n\nCyberChef operation: %q (module %s)", meta.Description, meta.Name, meta.Module),
	}

	// Reserve the global IO flag names so an operation argument that collapses to
	// one of them is renamed rather than colliding with the --input/--output/
	// --in-file flags added by addIOFlags below. The cipher ops have Raw/Hex
	// selector arguments literally named "Input"/"Output"; give those the
	// descriptive --input-format/--output-format names instead of a bare suffix.
	used := map[string]bool{"input": true, "in-file": true, "output": true}
	getters := make([]flagGetter, len(op.Args()))
	for i, def := range op.Args() {
		name := flagName(def.Name)
		if alt, ok := reservedFlagRenames[name]; ok {
			name = alt
		}
		getters[i] = addArgFlag(cmd, def, uniqueFlagName(name, used))
	}

	addIOFlags(cmd)

	cmd.RunE = func(c *cobra.Command, posArgs []string) error {
		args := make([]any, len(getters))
		for i, get := range getters {
			v, err := get(c)
			if err != nil {
				return err
			}
			args[i] = v
		}

		in, err := resolveInput(c, posArgs)
		if err != nil {
			return err
		}

		recipe := core.Recipe{{Op: meta.Name, Args: args}}
		out, err := recipe.Execute(core.NewDish(in, core.TypeString))
		if err != nil {
			return err
		}
		return writeOutput(c, out.Bytes())
	}

	return cmd
}

// flagName converts an argument name into a clean CLI flag name: lower-cased,
// spaces to hyphens, with any other punctuation dropped and runs of hyphens
// collapsed (e.g. `Treat "+" as space` -> `treat-as-space`).
func flagName(argName string) string {
	var sb strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(argName) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen && sb.Len() > 0 {
				sb.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.TrimRight(sb.String(), "-")
}

// uniqueFlagName returns base if unused, otherwise appends the smallest numeric
// suffix that makes it unique, recording the result in used. This disambiguates
// argument names that collapse to the same flag (e.g. "Restore [.]" and
// "Restore ://" both reduce to "restore").
func uniqueFlagName(base string, used map[string]bool) string {
	name := base
	for n := 2; used[name]; n++ {
		name = fmt.Sprintf("%s-%d", base, n)
	}
	used[name] = true
	return name
}

// addArgFlag registers the flag(s) for a single argument definition under the
// given (already de-duplicated) flag name and returns a getter that reads back
// the value(s) as the canonical argument type.
func addArgFlag(cmd *cobra.Command, def core.ArgDef, name string) flagGetter {
	f := cmd.Flags()

	switch def.Type {
	case core.ArgBoolean:
		dflt, _ := def.Value.(bool)
		f.Bool(name, dflt, def.Name)
		return func(c *cobra.Command) (any, error) { return c.Flags().GetBool(name) }

	case core.ArgNumber:
		var dflt float64
		switch n := def.Value.(type) {
		case int:
			dflt = float64(n)
		case float64:
			dflt = n
		}
		f.Float64(name, dflt, def.Name)
		return func(c *cobra.Command) (any, error) { return c.Flags().GetFloat64(name) }

	case core.ArgOption:
		opts, _ := def.Value.([]string)
		dflt := ""
		if def.DefaultIndex < len(opts) {
			dflt = opts[def.DefaultIndex]
		}
		f.String(name, dflt, fmt.Sprintf("%s (one of %v)", def.Name, opts))
		return func(c *cobra.Command) (any, error) { return c.Flags().GetString(name) }

	case core.ArgToggleString:
		modeName := name + "-type"
		dfltMode := ""
		if len(def.ToggleValues) > 0 {
			dfltMode = def.ToggleValues[0]
		}
		dfltVal, _ := def.Value.(string)
		f.String(name, dfltVal, def.Name)
		f.String(modeName, dfltMode, fmt.Sprintf("type of %s (one of %v)", def.Name, def.ToggleValues))
		return func(c *cobra.Command) (any, error) {
			val, err := c.Flags().GetString(name)
			if err != nil {
				return nil, err
			}
			mode, err := c.Flags().GetString(modeName)
			if err != nil {
				return nil, err
			}
			return core.ToggleString{Value: val, Option: mode}, nil
		}

	default: // ArgString, ArgEditableOption
		dflt, _ := def.Value.(string)
		f.String(name, dflt, def.Name)
		return func(c *cobra.Command) (any, error) { return c.Flags().GetString(name) }
	}
}
