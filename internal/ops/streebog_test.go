package ops

// Streebog fixtures transcribed from ../CyberChef/tests/operations/tests/Hash.mjs
// (Streebog-256/512 Test Cases 1 & 2). Streebog is GOST R 34.11-2012; the digest
// engine (gostDigest2012) is shared with GOST Hash and covered in gosthash_test.go.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// streebogRecipe builds a Streebog recipe. Input is an ArrayBuffer.
func streebogRecipe(length string) core.Recipe {
	return core.Recipe{{Op: "Streebog", Args: []any{length}}}
}

func streebogRun(t *testing.T, input string, r core.Recipe) string {
	t.Helper()
	out, err := r.Execute(core.NewDish([]byte(input), core.TypeArrayBuffer))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out.String()
}

func TestStreebog(t *testing.T) {
	cases := []struct {
		name, input, want string
		recipe            core.Recipe
	}{
		{
			"256 empty", "",
			"3f539a213e97c802cc229d474c6aa32a825a360b2a933a949fd925208d9ce1bb",
			streebogRecipe("256"),
		},
		{
			"256 fox", "The quick brown fox jumps over the lazy dog",
			"3e7dea7f2384b6c5a3d0e24aaa29c05e89ddd762145030ec22c71a6db8b2c1f4",
			streebogRecipe("256"),
		},
		{
			"512 empty", "",
			"8e945da209aa869f0455928529bcae4679e9873ab707b55315f56ceb98bef0a7362f715528356ee83cda5f2aac4c6ad2ba3a715c1bcd81cb8e9f90bf4c1c1a8a",
			streebogRecipe("512"),
		},
		{
			"512 fox", "The quick brown fox jumps over the lazy dog",
			"d2b793a0bb6cb5904828b5b6dcfb443bb8f33efc06ad09368878ae4cdc8245b97e60802469bed1e7c21a64ff0b179a6a1e0bb74d92965450a0adab69162c00fe",
			streebogRecipe("512"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := streebogRun(t, c.input, c.recipe); got != c.want {
				t.Fatalf("got %q\nwant %q", got, c.want)
			}
		})
	}
}
