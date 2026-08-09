package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// These regexes are direct ports of CyberChef's Utils.mjs recipe handling.
var (
	reDoubleQuoted = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	reRecipeOp     = regexp.MustCompile(`([^(]+)\(((?:'[^'\\]*(?:\\.[^'\\]*)*'|[^)/'])*)(/[^)]+)?\)`)
	reOpenQuote    = regexp.MustCompile(`(^|,|\{|:)'`)
	reCloseQuote   = regexp.MustCompile(`([^\\]|(?:\\\\)+)'(,|:|\}|$)`)
)

// marshalNoEscapeHTML marshals v to JSON without escaping <, > and & (matching
// JavaScript's JSON.stringify behaviour).
func marshalNoEscapeHTML(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// MarshalRecipeJSON renders a recipe as CyberChef's indented JSON form. It is
// the inverse of ParseRecipeConfig for a recipe written as a JSON array, and
// carries no trailing newline. <, > and & are left as themselves, matching
// JavaScript's JSON.stringify, so a saved recipe stays readable to edit.
func MarshalRecipeJSON(r Recipe) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// chefArgs renders a recipe step's arguments in Chef format: JSON values with
// the enclosing [] removed and double-quoted strings rewritten as single-quoted.
func chefArgs(args []any) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	s, err := marshalNoEscapeHTML(args)
	if err != nil {
		return "", err
	}
	s = s[1 : len(s)-1] // strip surrounding [ ]
	s = strings.ReplaceAll(s, "'", `\'`)
	s = reDoubleQuoted.ReplaceAllString(s, "'$1'")
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s, nil
}

// GeneratePrettyRecipe serialises a recipe to CyberChef's compact "Chef" text
// format, e.g. To_Base64('A-Za-z0-9+/='). Ported from Utils.generatePrettyRecipe.
func GeneratePrettyRecipe(r Recipe, newline bool) string {
	var sb strings.Builder
	for _, op := range r {
		name := strings.ReplaceAll(op.Op, " ", "_")
		args, err := chefArgs(op.Args)
		if err != nil {
			args = ""
		}
		sb.WriteString(name)
		sb.WriteString("(")
		sb.WriteString(args)
		if op.Disabled {
			sb.WriteString("/disabled")
		}
		if op.Breakpoint {
			sb.WriteString("/breakpoint")
		}
		sb.WriteString(")")
		if newline {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// ParseRecipeConfig parses a recipe given as either a JSON array or the Chef
// text format, auto-detecting by a leading "[". Ported from Utils.parseRecipeConfig.
func ParseRecipeConfig(recipe string) (Recipe, error) {
	recipe = strings.TrimSpace(recipe)
	if recipe == "" {
		return Recipe{}, nil
	}
	if recipe[0] == '[' {
		var r Recipe
		if err := json.Unmarshal([]byte(recipe), &r); err != nil {
			return nil, fmt.Errorf("parse JSON recipe: %w", err)
		}
		for i := range r {
			// A step has to name an operation. Without this the recipe would
			// be accepted here and fail only when run, and it has no valid
			// written form, so sharing it would silently change it. A name of
			// nothing but whitespace is the same case: the Chef form writes it
			// as an empty name.
			//
			// The name is trimmed for the same reason the Chef form trims it:
			// no operation name carries surrounding whitespace, and leaving it
			// would mean a recipe that changes when written out and read back.
			r[i].Op = strings.TrimSpace(r[i].Op)
			if r[i].Op == "" {
				return nil, fmt.Errorf("step %d names no operation", i+1)
			}
		}
		return r, nil
	}

	recipe = strings.ReplaceAll(recipe, "\n", "")
	if err := validatePrettyRecipe(recipe); err != nil {
		return nil, err
	}
	out := Recipe{}
	for _, m := range reRecipeOp.FindAllStringSubmatch(recipe, -1) {
		args := m[2]
		args = strings.ReplaceAll(args, `"`, `\"`)          // escape double quotes
		args = reOpenQuote.ReplaceAllString(args, `$1"`)    // opening ' -> "
		args = reCloseQuote.ReplaceAllString(args, `$1"$2`) // closing ' -> "
		args = strings.ReplaceAll(args, `\'`, "'")          // unescape single quotes
		args = "[" + args + "]"

		var parsed []any
		if err := json.Unmarshal([]byte(args), &parsed); err != nil {
			return nil, fmt.Errorf("parse args for %q: %w", m[1], err)
		}

		// The name is trimmed so a recipe written across indented lines names the
		// operation it means. No operation name begins or ends with whitespace,
		// and CyberChef keeps it only because its own recipes arrive on one line.
		name := strings.TrimSpace(strings.ReplaceAll(m[1], "_", " "))
		if name == "" {
			return nil, fmt.Errorf("step %d names no operation", len(out)+1)
		}
		step := RecipeOp{Op: name, Args: parsed}
		if strings.Contains(m[3], "disabled") {
			step.Disabled = true
		}
		if strings.Contains(m[3], "breakpoint") {
			step.Breakpoint = true
		}
		out = append(out, step)
	}
	return out, nil
}

// validatePrettyRecipe walks a recipe in the compact "Chef" format once,
// checking that it is built of `Name(arguments)` and nothing else, so that a
// mistyped recipe is refused rather than quietly matching nothing and doing
// nothing. CyberChef gained the same pass in 11.3.0, where it also guards its
// parser against a hostile recipe; Go's regexp cannot be made to backtrack that
// way, so here it is only about saying no clearly.
func validatePrettyRecipe(recipe string) error {
	for i := 0; i < len(recipe); {
		open := strings.IndexByte(recipe[i:], '(')
		if open <= 0 {
			// Either there is no bracket left, or one turned up with no
			// operation name in front of it.
			return errInvalidRecipe
		}
		end, err := endOfArguments(recipe, i+open+1)
		if err != nil {
			return err
		}
		i = end
	}
	return nil
}

// errInvalidRecipe is what a recipe that does not parse is refused with.
var errInvalidRecipe = errors.New("invalid recipe")

// endOfArguments finds where the argument list starting at from closes,
// stepping over quoted text and the escapes inside it. It returns the index
// just past the closing bracket.
func endOfArguments(recipe string, from int) (int, error) {
	inString, escaped := false, false
	for i := from; i < len(recipe); i++ {
		switch c := recipe[i]; {
		case escaped:
			escaped = false
		case inString && c == '\\':
			escaped = true
		case c == '\'':
			inString = !inString
		case c == ')' && !inString:
			return i + 1, nil
		}
	}
	// The arguments never closed, or ended inside a quote or an escape.
	return 0, errInvalidRecipe
}
