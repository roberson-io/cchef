package cmd

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestFormatRecipe covers the two formats a recipe is written in, and the error
// naming both when the target is neither.
func TestFormatRecipe(t *testing.T) {
	r := core.Recipe{{Op: "To Hex", Args: []any{"Space"}}}

	got, err := formatRecipe(r, formatJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "[\n") || !strings.Contains(got, `"op": "To Hex"`) {
		t.Errorf("json = %q", got)
	}

	got, err = formatRecipe(r, formatChef)
	if err != nil {
		t.Fatal(err)
	}
	if got != "To_Hex('Space')" {
		t.Errorf("chef = %q, want To_Hex('Space')", got)
	}

	_, err = formatRecipe(r, "yaml")
	if err == nil {
		t.Fatal("expected an unknown format to be refused")
	}
	for _, want := range []string{formatJSON, formatChef} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}
