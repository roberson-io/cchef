package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

// TestToggleStringGetterFlagErrors covers the flag-lookup error paths in the
// toggleString getter by invoking it against commands missing the expected flags
// (the getter takes the *cobra.Command as a parameter, which is the injection
// seam). In normal operation the getter is always called against the command its
// flags were registered on, so these never fire.
func TestToggleStringGetterFlagErrors(t *testing.T) {
	registrar := &cobra.Command{}
	getter := addArgFlag(registrar,
		core.ArgDef{Name: "Key", Type: core.ArgToggleString, ToggleValues: []string{"Hex"}}, "key")

	// A command missing the value flag: the first GetString fails.
	if _, err := getter(&cobra.Command{}); err == nil {
		t.Error("expected an error when the value flag is absent")
	}

	// A command with the value flag but not the "-type" mode flag: the second fails.
	valueOnly := &cobra.Command{}
	valueOnly.Flags().String("key", "", "")
	if _, err := getter(valueOnly); err == nil {
		t.Error("expected an error when the mode flag is absent")
	}
}
