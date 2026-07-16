package core

import (
	"fmt"
	"slices"
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
		out[i] = def.Value
	}
	return out
}

// CoerceArg validates and normalises a single argument value against its
// definition, returning the canonical Go type for that ArgType.
func CoerceArg(def ArgDef, value any) (any, error) {
	switch def.Type {
	case ArgString, ArgEditableOption:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("arg %q: expected string, got %T", def.Name, value)
		}
		return s, nil

	case ArgNumber:
		return coerceNumber(def, value)

	case ArgBoolean:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("arg %q: expected boolean, got %T", def.Name, value)
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

// coerceNumber coerces value to a float64 and enforces the optional Min/Max.
func coerceNumber(def ArgDef, value any) (any, error) {
	n, err := toFloat(value)
	if err != nil {
		return nil, fmt.Errorf("arg %q: %w", def.Name, err)
	}
	if def.Min != nil && n < *def.Min {
		return nil, fmt.Errorf("arg %q: %v below minimum %v", def.Name, n, *def.Min)
	}
	if def.Max != nil && n > *def.Max {
		return nil, fmt.Errorf("arg %q: %v above maximum %v", def.Name, n, *def.Max)
	}
	return n, nil
}

// coerceOption checks value is one of the option's allowed strings.
func coerceOption(def ArgDef, value any) (any, error) {
	s, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("arg %q: expected option string, got %T", def.Name, value)
	}
	opts, _ := def.Value.([]string)
	if slices.Contains(opts, s) {
		return s, nil
	}
	return nil, fmt.Errorf("arg %q: %q is not one of %v", def.Name, s, opts)
}

// coerceToggleString coerces value to a ToggleString and validates its mode
// against the declared ToggleValues.
func coerceToggleString(def ArgDef, value any) (any, error) {
	ts, err := toToggleString(value)
	if err != nil {
		return nil, fmt.Errorf("arg %q: %w", def.Name, err)
	}
	if len(def.ToggleValues) > 0 && !slices.Contains(def.ToggleValues, ts.Option) {
		return nil, fmt.Errorf("arg %q: mode %q is not one of %v", def.Name, ts.Option, def.ToggleValues)
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
