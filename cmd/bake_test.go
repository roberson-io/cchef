package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newSourceCmd returns a command carrying the recipe-source flags, with stdin
// set to the given text so `-r -` can be exercised.
func newSourceCmd(t *testing.T, stdin string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	addRecipeSourceFlags(c)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&bytes.Buffer{})
	return c
}

func TestLoadRecipeSourceExpr(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	// The command is built first: registering the flags assigns their defaults,
	// which would wipe a value set beforehand.
	c := newSourceCmd(t, "")
	flagRecipeExpr = "To_Base64()To_Hex('Space')"

	src, err := loadRecipeSource(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Recipe) != 2 || src.Recipe[0].Op != "To Base64" {
		t.Errorf("recipe = %+v", src.Recipe)
	}
	if src.Text != flagRecipeExpr {
		t.Errorf("text = %q, want the expression as written", src.Text)
	}
	if src.Input != nil {
		t.Errorf("an expression carries no input, got %q", src.Input)
	}
}

func TestLoadRecipeSourceFile(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	path := filepath.Join(t.TempDir(), "recipe.json")
	if err := os.WriteFile(path, []byte(`[{"op":"ROT13"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	c := newSourceCmd(t, "")
	flagRecipeFile = path

	src, err := loadRecipeSource(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Recipe) != 1 || src.Recipe[0].Op != "ROT13" {
		t.Errorf("recipe = %+v", src.Recipe)
	}
}

// TestLoadRecipeSourceStdin covers `-r -`, which lets a recipe be piped in the
// way every other file argument in the CLI accepts "-".
func TestLoadRecipeSourceStdin(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	c := newSourceCmd(t, "To_Hex('Colon')")
	flagRecipeFile = "-"

	src, err := loadRecipeSource(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Recipe) != 1 || src.Recipe[0].Op != "To Hex" {
		t.Errorf("recipe = %+v", src.Recipe)
	}
}

// TestLoadRecipeSourceStage covers the fallback that makes `cchef bake` run a
// recipe built up with `cchef recipe add`.
func TestLoadRecipeSourceStage(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	if err := stageAdd("rot13", -1); err != nil {
		t.Fatal(err)
	}
	src, err := loadRecipeSource(newSourceCmd(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Recipe) != 1 || src.Recipe[0].Op != "ROT13" {
		t.Errorf("recipe = %+v", src.Recipe)
	}
}

// TestLoadRecipeSourceNone reports that no recipe was given, naming every way
// one can be.
func TestLoadRecipeSourceNone(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	_, err := loadRecipeSource(newSourceCmd(t, ""))
	if err == nil {
		t.Fatal("expected an error when no recipe is given")
	}
	for _, want := range []string{"-e", "-r", "recipe add"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// TestLoadRecipeSourceConflict refuses two sources rather than silently
// preferring one.
func TestLoadRecipeSourceConflict(t *testing.T) {
	stageIn(t)
	defer resetRecipeSourceFlags()
	c := newSourceCmd(t, "")
	flagRecipeExpr = "ROT13()"
	flagRecipeFile = "recipe.json"
	_, err := loadRecipeSource(c)
	if err == nil {
		t.Fatal("expected two recipe sources to be refused")
	}
	for _, want := range []string{"-e", "-r"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

// TestLoadRecipeSourceCorruptStage reports a stage file that has been edited
// into something that is not a recipe, rather than treating it as no recipe.
func TestLoadRecipeSourceCorruptStage(t *testing.T) {
	path := stageIn(t)
	defer resetRecipeSourceFlags()
	if err := os.WriteFile(path, []byte("}not a recipe{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRecipeSource(newSourceCmd(t, "")); err == nil {
		t.Error("expected a corrupt stage file to be refused")
	}
}
