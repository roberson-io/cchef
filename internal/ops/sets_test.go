package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestSetOpsFixtures transcribes CyberChef's Set* / CartesianProduct / PowerSet
// fixtures. Set Difference and Set Intersection dedup the first sample per
// CyberChef PR #2286. A few Set Union cases (numeric-key ordering) come from the
// CyberChef-server oracle, which matches upstream for the unchanged Union op.
func TestSetOpsFixtures(t *testing.T) {
	runCases(t, []opCase{
		// Set Union
		{
			"Union: nothing", "\n\n", "",
			core.Recipe{{Op: "Set Union", Args: []any{"\n\n", " "}}},
		},
		{
			"Union: space", "1 2 3 4 5\n\n3 4 5 6 7", "1 2 3 4 5 6 7",
			core.Recipe{{Op: "Set Union", Args: []any{"\n\n", " "}}},
		},
		{
			"Union: item delimiter", "1,2,3,4,5\n\n3,4,5,6,7", "1,2,3,4,5,6,7",
			core.Recipe{{Op: "Set Union", Args: []any{"\n\n", ","}}},
		},
		{
			"Union: sample delimiter", "1 2 3 4 5whatever3 4 5 6 7", "1 2 3 4 5 6 7",
			core.Recipe{{Op: "Set Union", Args: []any{"whatever", " "}}},
		},
		// JS object-key ordering: integer-index keys sort ascending, then strings.
		{
			"Union: numeric key ordering", "3,1,2\n\n5,4", "1,2,3,4,5",
			core.Recipe{{Op: "Set Union", Args: []any{"\n\n", ","}}},
		},
		{
			"Union: integers before strings", "b,3,a,1\n\n2", "1,2,3,b,a",
			core.Recipe{{Op: "Set Union", Args: []any{"\n\n", ","}}},
		},

		// Set Intersection (dedups first sample; PR #2286)
		{
			"Intersection: space", "1 2 3 4 5\n\n3 4 5 6 7", "3 4 5",
			core.Recipe{{Op: "Set Intersection", Args: []any{"\n\n", " "}}},
		},
		{
			"Intersection: item delimiter", "1-2-3-4-5\n\n3-4-5-6-7", "3-4-5",
			core.Recipe{{Op: "Set Intersection", Args: []any{"\n\n", "-"}}},
		},
		{
			"Intersection: sample delimiter", "1-2-3-4-5z3-4-5-6-7", "3-4-5",
			core.Recipe{{Op: "Set Intersection", Args: []any{"z", "-"}}},
		},
		{
			"Intersection: dupes in first set removed", "red,red,blue\n\nred,blue", "red,blue",
			core.Recipe{{Op: "Set Intersection", Args: []any{"\n\n", ","}}},
		},
		{
			"Intersection: dupes in both sets", "1 1 2 2 3\n\n2 2 3 3 4", "2 3",
			core.Recipe{{Op: "Set Intersection", Args: []any{"\n\n", " "}}},
		},

		// Set Difference (dedups first sample; PR #2286)
		{
			"Difference: space", "1 2 3 4 5\n\n3 4 5 6 7", "1 2",
			core.Recipe{{Op: "Set Difference", Args: []any{"\n\n", " "}}},
		},
		{
			"Difference: item delimiter", "1;2;3;4;5\n\n3;4;5;6;7", "1;2",
			core.Recipe{{Op: "Set Difference", Args: []any{"\n\n", ";"}}},
		},
		{
			"Difference: sample delimiter", "1;2;3;4;5===3;4;5;6;7", "1;2",
			core.Recipe{{Op: "Set Difference", Args: []any{"===", ";"}}},
		},
		{
			"Difference: dupes in first set removed", "red,red,blue\n\nblue", "red",
			core.Recipe{{Op: "Set Difference", Args: []any{"\n\n", ","}}},
		},
		{
			"Difference: dupes in both sets", "1 1 2 2 3\n\n2 2 3 3", "1",
			core.Recipe{{Op: "Set Difference", Args: []any{"\n\n", " "}}},
		},

		// Symmetric Difference (preserves duplicates)
		{
			"Symmetric Difference: space", "1 2 3 4 5\n\n3 4 5 6 7", "1 2 6 7",
			core.Recipe{{Op: "Symmetric Difference", Args: []any{"\n\n", " "}}},
		},
		{
			"Symmetric Difference: item delimiter", "a_b_c_d_e\n\nc_d_e_f_g", "a_b_f_g",
			core.Recipe{{Op: "Symmetric Difference", Args: []any{"\n\n", "_"}}},
		},
		{
			"Symmetric Difference: sample delimiter", "a_b_c_d_eAAAAAc_d_e_f_g", "a_b_f_g",
			core.Recipe{{Op: "Symmetric Difference", Args: []any{"AAAAA", "_"}}},
		},
		{
			"Symmetric Difference: preserves dupes", "1,1,2\n\n2,3", "1,1,3",
			core.Recipe{{Op: "Symmetric Difference", Args: []any{"\n\n", ","}}},
		},

		// Cartesian Product
		{
			"Cartesian: space", "1 2 3\n\na b",
			"(1,a) (1,b) (2,a) (2,b) (3,a) (3,b)",
			core.Recipe{{Op: "Cartesian Product", Args: []any{"\n\n", " "}}},
		},
		{
			"Cartesian: item delimiter", "1-2-3-4-5\n\na-b-c-d-e",
			"(1,a)-(1,b)-(1,c)-(1,d)-(1,e)-(2,a)-(2,b)-(2,c)-(2,d)-(2,e)-(3,a)-(3,b)-(3,c)-(3,d)-(3,e)-(4,a)-(4,b)-(4,c)-(4,d)-(4,e)-(5,a)-(5,b)-(5,c)-(5,d)-(5,e)",
			core.Recipe{{Op: "Cartesian Product", Args: []any{"\n\n", "-"}}},
		},
		{
			"Cartesian: three sets", "1,2\n\na,b\n\nx,y",
			"(1,a,x),(1,a,y),(1,b,x),(1,b,y),(2,a,x),(2,a,y),(2,b,x),(2,b,y)",
			core.Recipe{{Op: "Cartesian Product", Args: []any{"\n\n", ","}}},
		},

		// Power Set
		{
			"Power Set: nothing", "", "",
			core.Recipe{{Op: "Power Set", Args: []any{","}}},
		},
		{
			"Power Set: space delimiter", "1 2 4", "\n4\n2\n1\n2 4\n1 4\n1 2\n1 2 4\n",
			core.Recipe{{Op: "Power Set", Args: []any{" "}}},
		},
		{
			"Power Set: comma delimiter", "a,b,c", "\nc\nb\na\nb,c\na,c\na,b\na,b,c\n",
			core.Recipe{{Op: "Power Set", Args: []any{","}}},
		},
	})
}

// TestSetOpsErrors covers the "incorrect number of sets" validation.
func TestSetOpsErrors(t *testing.T) {
	cases := []struct {
		name   string
		recipe core.Recipe
	}{
		{"Union: three samples", core.Recipe{{Op: "Set Union", Args: []any{"\n\n", " "}}}},
		{"Intersection: one sample", core.Recipe{{Op: "Set Intersection", Args: []any{"\n\n", " "}}}},
		{"Cartesian: one sample", core.Recipe{{Op: "Cartesian Product", Args: []any{"\n\n", " "}}}},
	}
	inputs := map[string]string{
		"Union: three samples":     "1 2\n\n3 4\n\n5 6",
		"Intersection: one sample": "1 2 3",
		"Cartesian: one sample":    "1 2 3",
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.recipe.Execute(core.NewDish([]byte(inputs[c.name]), core.TypeString))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "Incorrect number of sets") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetsBranches(t *testing.T) {
	if _, ok := arrayIndex("4294967295"); ok {
		t.Fatal("arrayIndex(2^32-1) should be false")
	}
	if _, ok := arrayIndex("99999999999999999999"); ok {
		t.Fatal("arrayIndex(overflow) should be false")
	}
	if _, err := runOp(t, "Set Difference", "onlyoneset", "\\n\\n", ","); err == nil {
		t.Fatal("Set Difference with a single set: expected an error")
	}
}

// TestSymmetricDifferenceError covers the splitSets error path (a single set).
func TestSymmetricDifferenceError(t *testing.T) {
	if _, err := runOp(t, "Symmetric Difference", "only one set", "\\n\\n", " "); err == nil {
		t.Fatal("expected an error for a single set")
	}
}
