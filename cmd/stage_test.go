package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

// stageIn points the staged recipe at a scratch file for the duration of a
// test, so tests never touch a real working recipe.
func stageIn(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), stageFileName)
	t.Setenv(stageEnvVar, path)
	return path
}

func TestStageAddAppends(t *testing.T) {
	stageIn(t)
	// A bare operation name stages the operation with its default arguments,
	// which is what makes the command usable without quoting a whole recipe.
	for _, spec := range []string{"to-base64", "To_Hex('Colon')"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatalf("add %q: %v", spec, err)
		}
	}
	r, err := loadStagedRecipe()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 2 || r[0].Op != "To Base64" || r[1].Op != "To Hex" {
		t.Fatalf("staged recipe = %+v", r)
	}
	if got := r[1].Args[0]; got != "Colon" {
		t.Errorf("To Hex delimiter = %v, want Colon", got)
	}
	// The default arguments are filled in, not left empty.
	if len(r[0].Args) == 0 {
		t.Error("To Base64 staged without its default alphabet")
	}
}

func TestStageAddAt(t *testing.T) {
	stageIn(t)
	for _, spec := range []string{"to-base64", "to-hex"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatal(err)
		}
	}
	if err := stageAdd("rot13", 0); err != nil {
		t.Fatal(err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 3 || r[0].Op != "ROT13" {
		t.Fatalf("after insert at 0: %+v", r)
	}
	if err := stageAdd("to-hex", 99); err == nil {
		t.Error("expected an out-of-range position to be refused")
	}
}

func TestStageAddRejectsUnknown(t *testing.T) {
	stageIn(t)
	if err := stageAdd("no-such-operation", -1); err == nil {
		t.Error("expected an unknown operation to be refused")
	}
	if err := stageAdd("To_Base64(", -1); err == nil {
		t.Error("expected a malformed recipe expression to be refused")
	}
}

func TestStageRemove(t *testing.T) {
	stageIn(t)
	for _, spec := range []string{"to-base64", "to-hex", "rot13"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatal(err)
		}
	}
	// Removing several at once must not shift the indices out from under the
	// user mid-command.
	if err := stageRemove([]int{0, 2}); err != nil {
		t.Fatal(err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "To Hex" {
		t.Fatalf("after removing 0 and 2: %+v", r)
	}
	if err := stageRemove([]int{5}); err == nil {
		t.Error("expected an out-of-range index to be refused")
	}
}

func TestStageMove(t *testing.T) {
	stageIn(t)
	for _, spec := range []string{"to-base64", "to-hex", "rot13"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatal(err)
		}
	}
	if err := stageMove(2, 0); err != nil {
		t.Fatal(err)
	}
	r, _ := loadStagedRecipe()
	if r[0].Op != "ROT13" || r[1].Op != "To Base64" || r[2].Op != "To Hex" {
		t.Fatalf("after move 2 -> 0: %+v", r)
	}
	if err := stageMove(0, 9); err == nil {
		t.Error("expected an out-of-range destination to be refused")
	}
}

func TestStageToggle(t *testing.T) {
	stageIn(t)
	if err := stageAdd("to-base64", -1); err != nil {
		t.Fatal(err)
	}
	if err := stageToggle([]int{0}); err != nil {
		t.Fatal(err)
	}
	r, _ := loadStagedRecipe()
	if !r[0].Disabled {
		t.Error("step not disabled")
	}
	if err := stageToggle([]int{0}); err != nil {
		t.Fatal(err)
	}
	r, _ = loadStagedRecipe()
	if r[0].Disabled {
		t.Error("step not re-enabled")
	}
}

func TestStageClearAndEmpty(t *testing.T) {
	path := stageIn(t)
	// Nothing staged is not an error; it is an empty recipe.
	if r, err := loadStagedRecipe(); err != nil || len(r) != 0 {
		t.Fatalf("empty stage = %+v, %v", r, err)
	}
	if err := stageAdd("to-base64", -1); err != nil {
		t.Fatal(err)
	}
	if err := stageClear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("clear left the stage file behind")
	}
	if err := stageClear(); err != nil {
		t.Errorf("clearing an empty stage: %v", err)
	}
}

func TestStageFormat(t *testing.T) {
	stageIn(t)
	for _, spec := range []string{"to-base64", "to-hex"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatal(err)
		}
	}
	if err := stageToggle([]int{1}); err != nil {
		t.Fatal(err)
	}
	r, _ := loadStagedRecipe()
	out := formatStage(r)
	if !strings.Contains(out, "0  To Base64") {
		t.Errorf("listing lacks an indexed first step:\n%s", out)
	}
	if !strings.Contains(out, "disabled") {
		t.Errorf("listing does not mark the disabled step:\n%s", out)
	}
	if got := formatStage(nil); !strings.Contains(got, "empty") {
		t.Errorf("empty listing = %q", got)
	}
}

// TestBakeUsesStagedRecipe checks the payoff: with nothing given on the command
// line, bake runs whatever is staged.
func TestBakeUsesStagedRecipe(t *testing.T) {
	stageIn(t)
	if err := stageAdd("to-base64", -1); err != nil {
		t.Fatal(err)
	}
	text, err := rawRecipeText()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "To Base64") && !strings.Contains(text, "To_Base64") {
		t.Errorf("bake did not fall back to the staged recipe: %q", text)
	}
}

// TestRawRecipeTextEmptyStage keeps the original error when there is no recipe
// anywhere, rather than silently baking nothing.
func TestRawRecipeTextEmptyStage(t *testing.T) {
	stageIn(t)
	if _, err := rawRecipeText(); err == nil {
		t.Error("expected an error when no recipe is given or staged")
	}
}

func TestParseIndexes(t *testing.T) {
	got, err := parseIndexes([]string{"0", "12"})
	if err != nil || len(got) != 2 || got[0] != 0 || got[1] != 12 {
		t.Fatalf("parseIndexes = %v, %v", got, err)
	}
	if _, err := parseIndexes([]string{"two"}); err == nil {
		t.Error("expected a non-numeric step to be refused")
	}
}

// TestStageErrorPaths covers the failure branches: an unreadable stage file, a
// stage file holding something that is not a recipe, and an empty add.
func TestStageErrorPaths(t *testing.T) {
	path := stageIn(t)
	if err := os.WriteFile(path, []byte("}not a recipe{"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, fn := range map[string]func() error{
		"load":   func() error { _, err := loadStagedRecipe(); return err },
		"add":    func() error { return stageAdd("rot13", -1) },
		"remove": func() error { return stageRemove([]int{0}) },
		"move":   func() error { return stageMove(0, 0) },
		"toggle": func() error { return stageToggle([]int{0}) },
	} {
		if err := fn(); err == nil {
			t.Errorf("%s: expected an error on an unparseable stage file", name)
		}
	}
	if err := stageClear(); err != nil {
		t.Fatal(err)
	}
	if err := stageAdd("   ", -1); err == nil {
		t.Error("expected an empty spec to be refused")
	}
	// An operation named the American way stages the same operation.
	if err := stageAdd("analyze-hash", -1); err != nil {
		t.Fatalf("US-spelled operation: %v", err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "Analyse hash" {
		t.Errorf("staged = %+v", r)
	}
}

// TestFormatStageArgTypes covers the argument shapes the listing renders.
func TestFormatStageArgTypes(t *testing.T) {
	got := formatStageArgs([]any{"text", 4.0, true, core.ToggleString{Value: "ff", Option: "Hex"}})
	for _, want := range []string{`"text"`, "4", "true", `Hex:"ff"`} {
		if !strings.Contains(got, want) {
			t.Errorf("formatStageArgs = %q, missing %q", got, want)
		}
	}
	if formatStageArgs(nil) != "" {
		t.Error("no arguments should render as nothing")
	}
}

// TestStageCommands drives the subcommands as a user would, so their wiring
// (argument parsing, the --at flag, and what show prints) is covered too.
func TestStageCommands(t *testing.T) {
	stageIn(t)
	root := &cobra.Command{Use: "recipe"}
	addStageCommands(root)

	run := func(args ...string) (string, error) {
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		err := root.Execute()
		return out.String(), err
	}

	for _, args := range [][]string{
		{"add", "to-base64"},
		{"add", "to-hex"},
		{"add", "rot13", "--at", "0"},
		{"toggle", "1"},
	} {
		if _, err := run(args...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	out, err := run("show")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "0  ROT13") || !strings.Contains(out, "(disabled)") {
		t.Errorf("show printed:\n%s", out)
	}

	if _, err := run("move", "2", "0"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("rm", "0"); err != nil {
		t.Fatal(err)
	}
	if _, err := run("rm", "nope"); err == nil {
		t.Error("expected a non-numeric step to be refused")
	}
	if _, err := run("move", "x", "0"); err == nil {
		t.Error("expected a non-numeric move to be refused")
	}
	if _, err := run("toggle", "x"); err == nil {
		t.Error("expected a non-numeric toggle to be refused")
	}
	if _, err := run("clear"); err != nil {
		t.Fatal(err)
	}
	out, err = run("show")
	if err != nil || !strings.Contains(out, "empty") {
		t.Errorf("after clear: %q, %v", out, err)
	}
}
