package ops

import "github.com/roberson-io/cchef/core"

// The bounds on the two numeric arguments, and on the length of the string.
const (
	validationMinInteger = 5
	validationMaxInteger = 10
	validationMinReal    = 1.5
	validationMaxReal    = 5.5
	validationMaxLength  = 5
)

// validationOptions are the choices the option argument offers. CyberChef also
// lists `[Group 1]` and `[Group 2]` headings among them, which its interface
// shows as headings and never lets anyone pick; cchef leaves them out, as its
// other operations with grouped choices do.
var validationOptions = []string{"Option 1", "Option 2", "Option 3"}

// validationToggles are the modes the toggle-string argument offers.
var validationToggles = []string{"Option A", "Option B"}

// AutomatedValidationTestOp does nothing but report that its arguments were
// acceptable. It exists so that the checks on arguments can be exercised
// end to end: every kind of limit an argument may carry is declared here, so a
// recipe that names this operation is really a test of the checker.
type AutomatedValidationTestOp struct{}

// Meta returns the operation metadata.
func (AutomatedValidationTestOp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Automated Validation Test Op",
		Module:      "Default",
		Description: "Operation used specifically to test automated parameter validation.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions, one for each kind of limit.
func (AutomatedValidationTestOp) Args() []core.ArgDef {
	minInteger, maxInteger := float64(validationMinInteger), float64(validationMaxInteger)
	minReal, maxReal := validationMinReal, validationMaxReal
	maxLength := validationMaxLength

	return []core.ArgDef{
		{
			Name: "Integer Number", Type: core.ArgNumber, Value: float64(validationMinInteger),
			Min: &minInteger, Max: &maxInteger, Integer: true,
		},
		{
			Name: "Real Number", Type: core.ArgNumber, Value: validationMinReal,
			Min: &minReal, Max: &maxReal,
		},
		{
			Name: "Non Empty String", Type: core.ArgString, Value: "hello",
			MaxLength: &maxLength, NonEmpty: true,
		},
		{Name: "Empty Allowed String", Type: core.ArgString, Value: ""},
		{
			Name: "Non Empty Toggle String", Type: core.ArgToggleString, Value: "test",
			ToggleValues: validationToggles, NonEmpty: true,
		},
		{Name: "Option Ingredient", Type: core.ArgOption, Value: validationOptions},
	}
}

// Run reports that the arguments were acceptable, which is all it has to say:
// anything wrong with them is caught before it is reached.
func (AutomatedValidationTestOp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish([]byte("Success"), core.TypeString), nil
}

func init() { core.Register(AutomatedValidationTestOp{}) }
