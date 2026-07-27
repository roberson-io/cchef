package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// validArgs are the arguments every case starts from; each case changes one.
func validArgs() []any {
	return []any{
		5.0, 1.5, "hello", "",
		core.ToggleString{Value: "test", Option: "Option A"},
		"Option 1",
	}
}

// withArg returns the valid arguments with one replaced.
func withArg(at int, value any) []any {
	args := validArgs()
	args[at] = value
	return args
}

// TestAutomatedValidationFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/AutomatedValidation.mjs). The operation
// exists to exercise the argument checker, so each case is really a check on
// what the engine says about an argument it will not take.
func TestAutomatedValidationFixtures(t *testing.T) {
	op, ok := core.Default.Get("Automated Validation Test Op")
	if !ok {
		t.Fatal("Automated Validation Test Op is not registered")
	}

	for _, tc := range []struct {
		name string
		args []any
		want string // empty means the arguments should be taken
	}{
		{"valid values", validArgs(), ""},
		{"an empty string where one is allowed", withArg(3, ""), ""},

		{
			"Integer Number under min limit", withArg(0, 4.0),
			"Integer Number must be greater than or equal to 5.",
		},
		{
			"Integer Number over max limit", withArg(0, 11.0),
			"Integer Number must be less than or equal to 10.",
		},
		{
			"Integer Number not an integer", withArg(0, 5.5),
			"Integer Number must be an integer.",
		},

		{
			"Real Number under min limit", withArg(1, 1.4),
			"Real Number must be greater than or equal to 1.5.",
		},
		{
			"Real Number over max limit", withArg(1, 5.6),
			"Real Number must be less than or equal to 5.5.",
		},

		{
			"Non Empty String over maxLength limit", withArg(2, "helloooo"),
			"Non Empty String length cannot exceed 5.",
		},
		{
			"Non Empty String is empty", withArg(2, ""),
			"Non Empty String cannot be empty.",
		},

		{
			"Non Empty Toggle String is empty",
			withArg(4, core.ToggleString{Value: "", Option: "Option A"}),
			"Non Empty Toggle String cannot be empty.",
		},

		{
			"Invalid Option value", withArg(5, "Option 4"),
			"Option Ingredient must be one of the following: Option 1, Option 2, Option 3.",
		},
		{
			"Option value as an optgroup heading", withArg(5, "[Group 1]"),
			"Option Ingredient must be one of the following: Option 1, Option 2, Option 3.",
		},
		{
			"Option value empty", withArg(5, ""),
			"Option Ingredient cannot be empty.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			coerced, err := core.CoerceArgs(op.Args(), tc.args)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("valid arguments were turned away: %v", err)
				}
				out, err := op.Run(core.NewDish([]byte("test"), core.TypeString), coerced)
				if err != nil {
					t.Fatalf("Run: %v", err)
				}
				if out.String() != "Success" {
					t.Errorf("got %q, want %q", out.String(), "Success")
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted %v", tc.args)
			}
			if err.Error() != tc.want {
				t.Errorf("got  %q\nwant %q", err.Error(), tc.want)
			}
		})
	}
}

// TestAutomatedValidationIgnoresItsInput covers the operation itself, which
// reports the same thing whatever it is given once the arguments are in order.
func TestAutomatedValidationIgnoresItsInput(t *testing.T) {
	for _, input := range []string{"", "test", "something else entirely"} {
		out, err := runOp(t, "Automated Validation Test Op", input, validArgs()...)
		if err != nil {
			t.Fatalf("Run(%q): %v", input, err)
		}
		if out != "Success" {
			t.Errorf("got %q, want %q", out, "Success")
		}
	}
}

// TestAutomatedValidationDefaults covers the arguments the operation starts
// from, which are the ones CyberChef declares and are all acceptable.
func TestAutomatedValidationDefaults(t *testing.T) {
	op, ok := core.Default.Get("Automated Validation Test Op")
	if !ok {
		t.Fatal("Automated Validation Test Op is not registered")
	}
	if _, err := core.CoerceArgs(op.Args(), nil); err != nil {
		t.Errorf("the operation's own defaults were turned away: %v", err)
	}
}
