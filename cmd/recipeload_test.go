package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecipeLoadExpr(t *testing.T) {
	stageIn(t)
	if _, err := execRootCapture(t, "recipe", "load", "-e", "To_Base64()To_Hex('Space')"); err != nil {
		t.Fatalf("recipe load: %v", err)
	}
	r, err := loadStagedRecipe()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 2 || r[0].Op != "To Base64" || r[1].Op != "To Hex" {
		t.Errorf("staged = %+v", r)
	}
}

// TestRecipeLoadReplaces is the difference from `recipe add`: load is the whole
// recipe, not another step on the end of it.
func TestRecipeLoadReplaces(t *testing.T) {
	stageIn(t)
	if err := stageAdd("rot13", -1); err != nil {
		t.Fatal(err)
	}
	if _, err := execRootCapture(t, "recipe", "load", "-e", "To_Hex('Colon')"); err != nil {
		t.Fatalf("recipe load: %v", err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "To Hex" {
		t.Errorf("load should replace the staged recipe, got %+v", r)
	}
}

func TestRecipeLoadFile(t *testing.T) {
	stageIn(t)
	path := filepath.Join(t.TempDir(), "recipe.json")
	if err := os.WriteFile(path, []byte(`[{"op":"ROT13"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := execRootCapture(t, "recipe", "load", "-r", path); err != nil {
		t.Fatalf("recipe load -r: %v", err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "ROT13" {
		t.Errorf("staged = %+v", r)
	}
}

func TestRecipeLoadStdin(t *testing.T) {
	stageIn(t)
	if _, err := execRootStdin(t, "To_Hex('Colon')", "recipe", "load", "-r", "-"); err != nil {
		t.Fatalf("recipe load -r -: %v", err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "To Hex" {
		t.Errorf("staged = %+v", r)
	}
}

// TestRecipeLoadFromURL stages the recipe a share link names, and says what
// happened to the input the link carried rather than dropping it silently.
func TestRecipeLoadFromURL(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "recipe", "load", "--from-url", shareURL)
	if err != nil {
		t.Fatalf("recipe load --from-url: %v", err)
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 || r[0].Op != "ROT13" {
		t.Errorf("staged = %+v", r)
	}
	if !strings.Contains(out, "input") {
		t.Errorf("a URL carrying input should say so, got %q", out)
	}

	// A link with no input has nothing to report.
	stageIn(t)
	out, err = execRootCapture(t, "recipe", "load", "--from-url",
		"https://gchq.github.io/CyberChef/#recipe=ROT13()")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "input") {
		t.Errorf("a URL with no input should say nothing, got %q", out)
	}
}

// TestRecipeLoadSourceErrors covers naming no source, and naming two.
func TestRecipeLoadSourceErrors(t *testing.T) {
	stageIn(t)
	if err := execRootErr(t, "recipe", "load"); err == nil {
		t.Error("expected an error when no source is given")
	}
	if err := execRootErr(t, "recipe", "load", "-e", "ROT13()", "-r", "x.json"); err == nil {
		t.Error("expected two sources to be refused")
	}
	if err := execRootErr(t, "recipe", "load", "-e", ""); err == nil {
		t.Error("expected an empty recipe to be refused")
	}
}

// TestRecipeLoadLeavesStageIntactOnError is what forces the recipe to be parsed
// and checked before the stage file is touched: a load that fails must not
// destroy the recipe someone already had.
func TestRecipeLoadLeavesStageIntactOnError(t *testing.T) {
	stageIn(t)
	if err := stageAdd("rot13", -1); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"malformed":         {"recipe", "load", "-e", "}not a recipe{"},
		"unknown operation": {"recipe", "load", "-e", "No_Such_Op()"},
		"missing file":      {"recipe", "load", "-r", filepath.Join(t.TempDir(), "nope.json")},
	} {
		if err := execRootErr(t, args...); err == nil {
			t.Errorf("%s: expected an error", name)
		}
		r, err := loadStagedRecipe()
		if err != nil {
			t.Fatalf("%s: stage unreadable after a failed load: %v", name, err)
		}
		if len(r) != 1 || r[0].Op != "ROT13" {
			t.Fatalf("%s: a failed load changed the stage: %+v", name, r)
		}
	}
}

// TestRecipeLoadRejectsUnknownOperation reports the bad step where the recipe is
// read, rather than when it is eventually run.
func TestRecipeLoadRejectsUnknownOperation(t *testing.T) {
	stageIn(t)
	err := execRootErr(t, "recipe", "load", "-e", "To_Hex()No_Such_Op()")
	if err == nil {
		t.Fatal("expected an unknown operation to be refused")
	}
	for _, want := range []string{"2", "No Such Op"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// TestRecipeConvertExportsStage pins the documented export path: convert with no
// recipe of its own reads the staged one.
func TestRecipeConvertExportsStage(t *testing.T) {
	stageIn(t)
	for _, spec := range []string{"rot13", "To_Hex('Colon')"} {
		if err := stageAdd(spec, -1); err != nil {
			t.Fatal(err)
		}
	}
	out, err := execRootCapture(t, "recipe", "convert", "--to", "chef")
	if err != nil {
		t.Fatalf("recipe convert: %v", err)
	}
	if got := strings.TrimSpace(out); got != "ROT13(true,true,false,13)To_Hex('Colon')" {
		t.Errorf("exported chef = %q", got)
	}

	out, err = execRootCapture(t, "recipe", "convert", "--to", "json")
	if err != nil {
		t.Fatalf("recipe convert --to json: %v", err)
	}
	if !strings.Contains(out, `"op": "ROT13"`) {
		t.Errorf("exported json = %q", out)
	}
}

// TestRecipeLoadEmptyRecipe covers a source that parses but names nothing.
// Clearing the recipe is `recipe clear`'s job, so this is an error rather than
// a silent way to empty the stage.
func TestRecipeLoadEmptyRecipe(t *testing.T) {
	stageIn(t)
	if err := stageAdd("rot13", -1); err != nil {
		t.Fatal(err)
	}
	if err := execRootErr(t, "recipe", "load", "-e", "   "); err == nil {
		t.Error("expected a recipe with no operations to be refused")
	}
	r, _ := loadStagedRecipe()
	if len(r) != 1 {
		t.Errorf("the staged recipe should be untouched, got %+v", r)
	}
}

// TestRecipeLoadUnwritableStage reports a stage file that cannot be written,
// rather than reporting success and staging nothing.
func TestRecipeLoadUnwritableStage(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("write-permission checks do not apply to root")
	}
	dir := t.TempDir()
	t.Setenv(stageEnvVar, filepath.Join(dir, "sub", stageFileName))
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o500); err != nil {
		t.Fatal(err)
	}
	if err := execRootErr(t, "recipe", "load", "-e", "ROT13()"); err == nil {
		t.Error("expected an unwritable stage file to be reported")
	}
}
