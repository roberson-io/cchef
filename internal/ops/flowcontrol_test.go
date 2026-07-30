package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// allBytesJoined is the ALL_BYTES fixture from CyberChef's flow control tests.
func allBytesJoined() string { return allBytes() }

// TestFlowControlFixtures runs the cases from CyberChef's Comment.mjs (which
// covers Comment and Label) verbatim.
func TestFlowControlFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Comment: nothing", "", "",
			core.Recipe{{Op: "Comment", Args: []any{""}}},
		},
		{
			"Fork, Comment, Base64", "cat\nsat\nmat", "Y2F0\nc2F0\nbWF0",
			core.Recipe{
				{Op: "Fork", Args: []any{"\\n", "\\n", false}},
				{Op: "Comment", Args: []any{"Testing 123"}},
				{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
			},
		},
		{
			"Label, Comment: Complex content", allBytesJoined(), allBytesJoined(),
			core.Recipe{
				{Op: "Label", Args: []any{""}},
				{Op: "Comment", Args: []any{""}},
			},
		},
	})
}

// TestFlowControlNoOps checks that the four steering operations which do not
// touch the data leave it exactly as it was, whatever their arguments.
func TestFlowControlNoOps(t *testing.T) {
	cases := []struct {
		op   string
		args []any
	}{
		{"Comment", []any{"some note"}},
		{"Label", []any{"a label"}},
		{"Merge", []any{true}},
		{"Merge", []any{false}},
		{"Return", nil},
	}
	for _, c := range cases {
		t.Run(c.op, func(t *testing.T) {
			got, err := runOp(t, c.op, allBytesJoined(), c.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != allBytesJoined() {
				t.Errorf("%s changed the data", c.op)
			}
		})
	}
}

// TestReturnStopsTheRecipe checks that Return ends execution, so the steps
// after it never run.
func TestReturnStopsTheRecipe(t *testing.T) {
	recipe := core.Recipe{
		{Op: "To Upper case", Args: []any{"All"}},
		{Op: "Return", Args: []any{}},
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("hello"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "HELLO" {
		t.Errorf("got %q, want %q", out.String(), "HELLO")
	}
}

// TestMergeWithoutForkIsHarmless checks that a Merge with no Fork above it just
// passes the data along, as it does upstream.
func TestMergeWithoutForkIsHarmless(t *testing.T) {
	recipe := core.Recipe{
		{Op: "Merge", Args: []any{true}},
		{Op: "To Upper case", Args: []any{"All"}},
	}
	out, err := recipe.Execute(core.NewDish([]byte("hi"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "HI" {
		t.Errorf("got %q, want %q", out.String(), "HI")
	}
}

// TestLabelIsFoundAnywhere checks that a label can be jumped to whether it sits
// before or after the jump.
func TestLabelIsFoundAnywhere(t *testing.T) {
	forwards := core.Recipe{
		{Op: "Jump", Args: []any{"end", 10.0}},
		{Op: "To Base64", Args: []any{"A-Za-z0-9+/="}},
		{Op: "Label", Args: []any{"end"}},
	}
	out, err := forwards.Execute(core.NewDish([]byte("keep"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "keep" {
		t.Errorf("forwards: got %q, want %q", out.String(), "keep")
	}
}

// TestFlowControlStandalone checks the degenerate behaviour of a flow operation
// invoked on its own rather than inside a recipe: it passes the data through,
// since there is no recipe to steer.
func TestFlowControlStandalone(t *testing.T) {
	for _, op := range []struct {
		name string
		args []any
	}{
		{"Jump", []any{"nowhere", 10.0}},
		{"Conditional Jump", []any{"x", false, "nowhere", 10.0}},
		{"Register", []any{"(.*)", true, false, false}},
		{"Subsection", []any{"", true, true, false}},
	} {
		t.Run(op.name, func(t *testing.T) {
			got, err := runOp(t, op.name, "unchanged", op.args...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if got != "unchanged" {
				t.Errorf("got %q, want %q", got, "unchanged")
			}
		})
	}
}

// TestForkStandalone checks Fork on its own: with no sub-recipe to run, each
// piece passes through and they are rejoined with the merge delimiter.
func TestForkStandalone(t *testing.T) {
	got, err := runOp(t, "Fork", "a,b,c", ",", "-", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got != "a-b-c" {
		t.Errorf("got %q, want %q", got, "a-b-c")
	}
}

// TestFlowControlOpsAreFlowOperations checks every steering operation is wired
// into the engine as a flow operation, not run as an ordinary transform.
func TestFlowControlOpsAreFlowOperations(t *testing.T) {
	for _, name := range []string{
		"Comment", "Conditional Jump", "Fork", "Jump", "Label", "Merge",
		"Register", "Return", "Subsection",
	} {
		op, ok := core.Default.Get(name)
		if !ok {
			t.Fatalf("%s is not registered", name)
		}
		if _, isFlow := op.(core.FlowOperation); !isFlow {
			t.Errorf("%s does not implement core.FlowOperation", name)
		}
	}
}

// TestFlowControlDescriptions checks each operation carries CyberChef's own
// module name, so the recipe format round-trips.
func TestFlowControlDescriptions(t *testing.T) {
	for _, name := range []string{
		"Comment", "Conditional Jump", "Fork", "Jump", "Label", "Merge",
		"Register", "Return", "Subsection",
	} {
		op, _ := core.Default.Get(name)
		if got := op.Meta().Module; got != "FlowControl" && got != "Default" {
			t.Errorf("%s: module %q", name, got)
		}
		if !strings.HasPrefix(op.Meta().InfoURL, "https://") && op.Meta().InfoURL != "" {
			t.Errorf("%s: infoURL %q", name, op.Meta().InfoURL)
		}
	}
}
