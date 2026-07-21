package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func shakeRecipe(capacity string, size int) core.Recipe {
	return core.Recipe{{Op: "Shake", Args: []any{capacity, size}}}
}

// Shake has no upstream operation fixtures; these vectors come from the
// CyberChef-server oracle. The Size argument is in bits and the output is
// floor(size/8) bytes of SHAKE output as hex.
func TestShakeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Shake: 256 cap, 512 bits", "Hello World",
			"840d1ce81a4327840b54cb1d419907fd1f62359bad33656e058653d2e4172a43acc958dbec0cf0d473db458ce1c007aa6eb40eac92aa0e65202edb4d7feed378",
			shakeRecipe("256", 512),
		},
		{
			"Shake: 128 cap, 256 bits", "Hello World",
			"1227c5f882f9c57bf2e3e48d2c87eb20f382a4b639b54d26f6d595ff3db9064d",
			shakeRecipe("128", 256),
		},
		{"Shake: 128 cap, 12 bits (1 byte)", "Hello World", "12", shakeRecipe("128", 12)},
		{"Shake: 128 cap, 20 bits (2 bytes)", "Hello World", "1227", shakeRecipe("128", 20)},
		{"Shake: 128 cap, 4 bits (0 bytes)", "Hello World", "", shakeRecipe("128", 4)},
		{"Shake: 256 cap, size 0", "Hello World", "", shakeRecipe("256", 0)},
	})
}

func TestShakeNegativeSize(t *testing.T) {
	_, err := runOp(t, "Shake", "Hello World", "256", -8)
	if err == nil || err.Error() != "Size must be greater than 0" {
		t.Fatalf("got %v, want size error", err)
	}
}
