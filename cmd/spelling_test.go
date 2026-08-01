package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

func opCommandByName(t *testing.T, opName string) *cobra.Command {
	t.Helper()
	op, ok := core.Default.Get(opName)
	if !ok {
		t.Fatalf("operation %q not registered", opName)
	}
	return buildOpCommand(op)
}

// TestUSSpellingCommandAliases checks that UK-spelled subcommands answer to
// their US spellings, with the CyberChef-derived name staying canonical.
func TestUSSpellingCommandAliases(t *testing.T) {
	for _, tc := range []struct{ op, alias string }{
		{"Analyse hash", "analyze-hash"},
		{"Split Colour Channels", "split-color-channels"},
		{"BSON serialise", "bson-serialize"},
		{"Normalise Unicode", "normalize-unicode"},
		{"Randomize Colour Palette", "randomize-color-palette"},
		{"Convert co-ordinate format", "convert-coordinate-format"},
	} {
		cmd := opCommandByName(t, tc.op)
		if !cmd.HasAlias(tc.alias) {
			t.Errorf("%s: missing alias %q (has %v)", tc.op, tc.alias, cmd.Aliases)
		}
	}
}

// TestUSSpellingFlagAliases checks that a flag keeping CyberChef's UK
// spelling also parses under the US spelling.
func TestUSSpellingFlagAliases(t *testing.T) {
	cmd := opCommandByName(t, "View Bit Plane")
	if err := cmd.ParseFlags([]string{"--color=Green"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := cmd.Flags().GetString("colour"); got != "Green" {
		t.Errorf("colour = %q", got)
	}
	cmd = opCommandByName(t, "Parse QR Code")
	if err := cmd.ParseFlags([]string{"--normalize-image"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := cmd.Flags().GetBool("normalise-image"); !got {
		t.Error("normalise-image not set")
	}
}

// TestUSSpellingOptionValues checks an option value typed in the other
// spelling reaches the operation as the canonical choice, both directions.
func TestUSSpellingOptionValues(t *testing.T) {
	cmd := &cobra.Command{}
	def := core.ArgDef{
		Name: "Mode", Type: core.ArgOption,
		Value: []string{"Greyscale", "RGB", "Metres (m)", "Randomize"},
	}
	get := addArgFlag(cmd, def, "mode")
	for in, want := range map[string]string{
		"Grayscale":  "Greyscale",
		"Meters (m)": "Metres (m)",
		"Randomise":  "Randomize",
		"RGB":        "RGB",
		"bogus":      "bogus", // not a choice either way: left for coercion to reject
		// The American spelling and a different case, together.
		"grayscale":  "Greyscale",
		"GRAYSCALE":  "Greyscale",
		"meters (m)": "Metres (m)",
		"randomise":  "Randomize",
		"rgb":        "RGB",
	} {
		if err := cmd.Flags().Set("mode", in); err != nil {
			t.Fatal(err)
		}
		v, err := get(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if v != want {
			t.Errorf("option %q = %q, want %q", in, v, want)
		}
	}
}

// TestSpellingVariants pins the word-level conversion used for the aliases.
func TestSpellingVariants(t *testing.T) {
	for in, want := range map[string]string{
		"analyse-hash":     "analyze-hash",
		"colour-pattern-1": "color-pattern-1",
		"greyscale":        "grayscale",
		"centre-x":         "center-x",
		"plain-name":       "plain-name",
	} {
		if got := usSpelling(in); got != want {
			t.Errorf("usSpelling(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ukSpelling("serialize-all"); got != "serialise-all" {
		t.Errorf("ukSpelling(serialize-all) = %q", got)
	}
}

// Where the casing of a choice is the setting, an exact match wins and an
// unmatched casing is left for coercion to reject rather than guessed at.
func TestOptionValueCaseIsTheSetting(t *testing.T) {
	cmd := &cobra.Command{}
	def := core.ArgDef{
		Name: "Format options", Type: core.ArgOption,
		Value: []string{"Dash/Dot", "DASH/DOT", "dash/dot"},
	}
	get := addArgFlag(cmd, def, "format-options")
	for in, want := range map[string]string{
		"Dash/Dot": "Dash/Dot",
		"DASH/DOT": "DASH/DOT",
		"dash/dot": "dash/dot",
		"Dash/DOT": "Dash/DOT", // ambiguous: passed through for coercion to reject
	} {
		if err := cmd.Flags().Set("format-options", in); err != nil {
			t.Fatal(err)
		}
		v, err := get(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if v != want {
			t.Errorf("option %q = %q, want %q", in, v, want)
		}
	}
}
