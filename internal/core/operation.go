package core

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/roberson-io/cchef/internal/jsnum"
)

// ArgType mirrors CyberChef's ingredient types (src/core/Ingredient.mjs). The
// curated operation set only needs this subset.
type ArgType string

const (
	// ArgString is a free-form string argument.
	ArgString ArgType = "string"
	// ArgNumber is a numeric argument (stored as float64).
	ArgNumber ArgType = "number"
	// ArgBoolean is a true/false argument.
	ArgBoolean ArgType = "boolean"
	// ArgOption is a single choice from a fixed list (Value is []string).
	ArgOption ArgType = "option"
	// ArgEditableOption is a string with suggested values (Value is the default string).
	ArgEditableOption ArgType = "editableOption"
	// ArgToggleString is a string paired with a mode selected from ToggleValues.
	ArgToggleString ArgType = "toggleString"
)

// ToggleString is the value form of an ArgToggleString argument: a string plus
// the selected mode (e.g. {Value:"ff", Option:"Hex"}).
type ToggleString struct {
	Value  string `json:"string"`
	Option string `json:"option"`
}

// ArgDef describes a single operation argument.
type ArgDef struct {
	Name         string
	Type         ArgType
	Value        any      // default value; for ArgOption this is the []string of choices
	DefaultIndex int      // for ArgOption: index of the default choice
	Min          *float64 // optional numeric lower bound
	Max          *float64 // optional numeric upper bound
	Integer      bool     // for ArgNumber: the value must be a whole number
	NonEmpty     bool     // for string arguments: the value may not be empty
	MaxLength    *int     // optional cap on the length of a string argument
	ToggleValues []string // modes for ArgToggleString
}

// OpMeta is the static metadata describing an operation.
type OpMeta struct {
	Name        string
	Module      string
	Description string
	InfoURL     string
	InputType   DishType
	OutputType  DishType
}

// Operation is the interface every cchef operation implements.
type Operation interface {
	Meta() OpMeta
	Args() []ArgDef
	Run(in *Dish, args []any) (*Dish, error)
}

// DefaultArgs returns the default value for each argument definition. For
// ArgOption the default is the choice at DefaultIndex (0 unless set).
func DefaultArgs(defs []ArgDef) []any {
	out := make([]any, len(defs))
	for i, def := range defs {
		if def.Type == ArgOption {
			if opts, ok := def.Value.([]string); ok && def.DefaultIndex < len(opts) {
				out[i] = opts[def.DefaultIndex]
				continue
			}
		}
		if def.Type == ArgToggleString {
			// The declared Value is the string part; pair it with the first mode
			// so an omitted toggle-string argument has a valid ToggleString default.
			s, _ := def.Value.(string)
			opt := ""
			if len(def.ToggleValues) > 0 {
				opt = def.ToggleValues[0]
			}
			out[i] = ToggleString{Value: s, Option: opt}
			continue
		}
		out[i] = def.Value
	}
	return out
}

// CoerceArg validates and normalises a single argument value against its
// definition, returning the canonical Go type for that ArgType.
func CoerceArg(def ArgDef, value any) (any, error) {
	switch def.Type {
	case ArgString, ArgEditableOption:
		return coerceString(def, value)

	case ArgNumber:
		return coerceNumber(def, value)

	case ArgBoolean:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%s must be true or false.", def.Name) //nolint:staticcheck,revive // reported to the user as a sentence
		}
		return b, nil

	case ArgOption:
		return coerceOption(def, value)

	case ArgToggleString:
		return coerceToggleString(def, value)

	default:
		return nil, fmt.Errorf("arg %q: unknown type %q", def.Name, def.Type)
	}
}

// coerceString checks a string argument against the limits on its length.
func coerceString(def ArgDef, value any) (any, error) {
	s, ok := value.(string)
	if !ok {
		return nil, errMustBeText(def.Name)
	}
	if def.NonEmpty && s == "" {
		return nil, errCannotBeEmpty(def.Name)
	}
	// The limit counts characters as JavaScript does, which is UTF-16 units.
	if def.MaxLength != nil && utf16Len(s) > *def.MaxLength {
		//nolint:staticcheck,revive // reported to the user as a sentence
		return nil, fmt.Errorf("%s length cannot exceed %d.", def.Name, *def.MaxLength)
	}
	return s, nil
}

// coerceNumber coerces value to a float64 and enforces the optional bounds.
func coerceNumber(def ArgDef, value any) (any, error) {
	n, err := toFloat(value)
	if err != nil {
		//nolint:staticcheck,revive // reported to the user as a sentence
		return nil, fmt.Errorf("%s must be a number.", def.Name)
	}
	if def.Integer && n != math.Trunc(n) {
		//nolint:staticcheck,revive // reported to the user as a sentence
		return nil, fmt.Errorf("%s must be an integer.", def.Name)
	}
	if def.Min != nil && n < *def.Min {
		//nolint:staticcheck,revive // reported to the user as a sentence
		return nil, fmt.Errorf("%s must be greater than or equal to %s.", def.Name, jsnum.Format(*def.Min))
	}
	if def.Max != nil && n > *def.Max {
		//nolint:staticcheck,revive // reported to the user as a sentence
		return nil, fmt.Errorf("%s must be less than or equal to %s.", def.Name, jsnum.Format(*def.Max))
	}
	return n, nil
}

// coerceOption checks value is one of the option's allowed strings.
func coerceOption(def ArgDef, value any) (any, error) {
	s, ok := value.(string)
	if !ok {
		return nil, errMustBeText(def.Name)
	}
	opts, _ := def.Value.([]string)
	if slices.Contains(opts, s) {
		return s, nil
	}
	// An empty choice is only a mistake where none is on offer; some operations
	// offer one as a meaningful selection.
	if s == "" {
		return nil, errCannotBeEmpty(def.Name)
	}
	return nil, errMustBeOneOf(def.Name, opts)
}

// errMustBeText is what a value that is not text at all gets.
//
//nolint:staticcheck,revive // reported to the user as a sentence
func errMustBeText(name string) error {
	return fmt.Errorf("%s must be text.", name)
}

// errCannotBeEmpty is what an empty value gets where one is needed.
//
//nolint:staticcheck,revive // reported to the user as a sentence
func errCannotBeEmpty(name string) error {
	return fmt.Errorf("%s cannot be empty.", name)
}

// errMustBeOneOf lists what was on offer.
//
//nolint:staticcheck,revive // reported to the user as a sentence
func errMustBeOneOf(name string, allowed []string) error {
	return fmt.Errorf("%s must be one of the following: %s.", name, strings.Join(allowed, ", "))
}

// utf16Len counts the units JavaScript measures a string's length in, which is
// two for anything beyond the basic plane.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

// coerceToggleString coerces value to a ToggleString and validates its mode
// against the declared ToggleValues.
func coerceToggleString(def ArgDef, value any) (any, error) {
	ts, err := toToggleString(value)
	if err != nil {
		return nil, errMustBeText(def.Name)
	}
	// It is the text that may not be empty; the mode always has a value.
	if def.NonEmpty && ts.Value == "" {
		return nil, errCannotBeEmpty(def.Name)
	}
	if len(def.ToggleValues) > 0 && !slices.Contains(def.ToggleValues, ts.Option) {
		return nil, errMustBeOneOf(def.Name, def.ToggleValues)
	}
	return ts, nil
}

// CoerceArgs coerces a full argument list against an operation's definitions,
// filling in defaults for any trailing arguments the caller omitted.
func CoerceArgs(defs []ArgDef, args []any) ([]any, error) {
	if len(args) > len(defs) {
		return nil, fmt.Errorf("too many arguments: got %d, operation takes %d", len(args), len(defs))
	}
	defaults := DefaultArgs(defs)
	out := make([]any, len(defs))
	for i, def := range defs {
		raw := defaults[i]
		if i < len(args) {
			raw = args[i]
		}
		v, err := CoerceArg(def, raw)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func toFloat(value any) (float64, error) {
	switch n := value.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("expected number, got %T", value)
	}
}

func toToggleString(value any) (ToggleString, error) {
	switch v := value.(type) {
	case ToggleString:
		return v, nil
	case map[string]any:
		ts := ToggleString{}
		if s, ok := v["string"].(string); ok {
			ts.Value = s
		}
		if o, ok := v["option"].(string); ok {
			ts.Option = o
		}
		return ts, nil
	default:
		return ToggleString{}, fmt.Errorf("expected toggleString, got %T", value)
	}
}
