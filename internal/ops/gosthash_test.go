package ops

// GOST Hash tests. CyberChef ships no fixture file for GOST Hash, so every
// expected value was produced by the CyberChef-server oracle (which wraps the
// same gostDigest engine this file ports), except the empty-input Streebog
// vectors, which are the published GOST R 34.11-2012 test vectors.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// gostHashRecipe builds a GOST Hash recipe. Input is an ArrayBuffer.
func gostHashRecipe(algo, length, sBox string) core.Recipe {
	return core.Recipe{{Op: "GOST Hash", Args: []any{algo, length, sBox}}}
}

func gostHashRun(t *testing.T, input string, r core.Recipe) string {
	t.Helper()
	out, err := r.Execute(core.NewDish([]byte(input), core.TypeArrayBuffer))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out.String()
}

func TestGOSTHash(t *testing.T) {
	const streebog = "GOST R 34.11 (Streebog, 2012)"
	const gost94 = "GOST 28147 (1994)"
	cases := []struct {
		name, input, want string
		recipe            core.Recipe
	}{
		// Streebog-256/512 (oracle-verified).
		{
			"Streebog256 Hello", "Hello, World!", "eb4672c915b0e4f19ce949b9a8fff8ba6b36172ed168458d6a75e752e66faaf3",
			gostHashRecipe(streebog, "256", "E-TEST"),
		},
		{
			"Streebog512 Hello", "Hello, World!", "1fd64d8727c5155293239cf53837704e776997b6ec54e923bcf1849a90c0f4c9155254d4a4dba67a19c380be3c3c12f9badd055dd7b7e1f7e6072f83f5bd15f7",
			gostHashRecipe(streebog, "512", "E-TEST"),
		},
		{
			"Streebog256 abc", "abc", "4e2919cf137ed41ec4fb6270c61826cc4fffb660341e0af3688cd0626d23b481",
			gostHashRecipe(streebog, "256", "E-TEST"),
		},
		{
			"Streebog256 63 bytes", "012345678901234567890123456789012345678901234567890123456789012", "9d151eefd8590b89daa6ba6cb74af9275dd051026bb149a452fd84e5e57b5500",
			gostHashRecipe(streebog, "256", "E-TEST"),
		},
		{
			"Streebog256 64 bytes", "0123456789012345678901234567890123456789012345678901234567890123", "a976cb1524ea234e060d38c439ac83c2dc154f6d6adfd92365b8f88a29d8e666",
			gostHashRecipe(streebog, "256", "E-TEST"),
		},
		// Empty input: published GOST R 34.11-2012 vectors.
		{
			"Streebog256 empty", "", "3f539a213e97c802cc229d474c6aa32a825a360b2a933a949fd925208d9ce1bb",
			gostHashRecipe(streebog, "256", "E-TEST"),
		},
		{
			"Streebog512 empty", "", "8e945da209aa869f0455928529bcae4679e9873ab707b55315f56ceb98bef0a7362f715528356ee83cda5f2aac4c6ad2ba3a715c1bcd81cb8e9f90bf4c1c1a8a",
			gostHashRecipe(streebog, "512", "E-TEST"),
		},

		// GOST R 34.11-94 (oracle-verified).
		{
			"94 E-A Hello", "Hello, World!", "870870f9accd2d875e66070994c9d9bfde2cdaa2237d3e38f6478764736ec2a5",
			gostHashRecipe(gost94, "256", "E-A"),
		},
		{
			"94 D-A Hello", "Hello, World!", "b8570078f02d37ead2f6676dd0d531519bdbd8f30f012ce9f934c92f4961357d",
			gostHashRecipe(gost94, "256", "D-A"),
		},
		// 32-byte (full-block) input exercises the digest94 q>0 loop.
		{
			"94 E-A 32 bytes", "0123456789abcdef0123456789abcdef", "bc01bda115f5cbba12885b774617f282e1627a7066ec7b754831dcadc0cc1a86",
			gostHashRecipe(gost94, "256", "E-A"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := gostHashRun(t, c.input, c.recipe); got != c.want {
				t.Fatalf("got %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestGOSTB64decode covers the base64 decoder's non-alphabet branch (the
// Streebog constants are pure base64, so the default case is otherwise unhit).
func TestGOSTB64decode(t *testing.T) {
	// "TWFu" decodes to "Man"; a stray space is mapped to 0 like the JS decoder.
	if got := string(gostB64decode("TWFu")); got != "Man" {
		t.Fatalf("gostB64decode = %q, want Man", got)
	}
	if got := gostB64decode("T WFu"); len(got) == 0 {
		t.Fatalf("gostB64decode with non-alphabet char returned empty")
	}
}

// TestGOSTHashBadSBox exercises the 1994 cipher-construction error path, which
// is unreachable through the option-validated argument but is reached here by
// calling Run directly with an invalid S-box name.
func TestGOSTHashBadSBox(t *testing.T) {
	_, err := GOSTHash{}.Run(core.NewDish([]byte("hi"), core.TypeArrayBuffer),
		[]any{"GOST 28147 (1994)", "256", "BOGUS"})
	if err == nil {
		t.Fatal("expected error for unknown sBox")
	}
}
