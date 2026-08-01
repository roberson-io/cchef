package ops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(CSVToJSON{})
	core.Register(JSONToCSV{})
}

// csvFormats are the CSV to JSON output shapes.
var csvFormats = []string{"Array of dictionaries", "Array of arrays"}

// CSVToJSON converts CSV text into JSON. It ports CyberChef's operation: parse
// the CSV with Utils.parseCSV (shared with To Table), then shape the rows as
// either an array of arrays or an array of header-keyed dictionaries.
type CSVToJSON struct{}

// Meta returns the operation metadata.
func (CSVToJSON) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CSV to JSON",
		Module:      "Default",
		Description: "Converts a CSV file to JSON format.",
		InfoURL:     "https://wikipedia.org/wiki/Comma-separated_values",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (CSVToJSON) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Cell delimiters", Type: core.ArgString, Value: ","},
		{Name: "Row delimiters", Type: core.ArgString, Value: `\r\n`},
		{Name: "Format", Type: core.ArgOption, Value: csvFormats},
	}
}

// Run converts the CSV input into JSON.
func (CSVToJSON) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cellDelims := []rune(parseEscapedChars(args[0].(string)))
	rowDelims := []rune(parseEscapedChars(args[1].(string)))
	dict := args[2].(string) == "Array of dictionaries"

	rows := parseCSV(in.String(), cellDelims, rowDelims)
	v := csvRowsToValue(rows, dict)
	return core.NewDish([]byte(jsStringify(v, 4)), core.TypeJSON), nil
}

// csvRowsToValue shapes parsed CSV rows into the JSON value tree. For "Array of
// dictionaries" the first row is the header and each later row becomes an
// object; a missing cell leaves the key undefined (omitted by JSON.stringify),
// and a duplicate header name keeps its first position with the last value.
func csvRowsToValue(rows [][]string, dict bool) any {
	if !dict {
		out := make([]any, len(rows))
		for i, row := range rows {
			cells := make([]any, len(row))
			for j, c := range row {
				cells[j] = c
			}
			out[i] = cells
		}
		return out
	}

	out := []any{}
	if len(rows) == 0 {
		return out
	}
	header := rows[0]
	for _, row := range rows[1:] {
		obj := jsObject{}
		for i, h := range header {
			var v any = jsUndefined{}
			if i < len(row) {
				v = row[i]
			}
			if idx := jsIndex(obj, h); idx >= 0 {
				obj[idx].v = v
			} else {
				obj = append(obj, jsPair{k: h, v: v})
			}
		}
		out = append(out, obj)
	}
	return out
}

// JSONToCSV converts JSON data into CSV (RFC 4180). It reproduces CyberChef's
// operation: a direct pass over an array of arrays or array of dictionaries,
// falling back to flattening nested structures into dotted keys (the `flat` npm
// library) and forcing remaining containers through JSON.stringify.
type JSONToCSV struct{}

// Meta returns the operation metadata.
func (JSONToCSV) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JSON to CSV",
		Module:      "Default",
		Description: "Converts JSON data to a CSV based on the definition in RFC 4180.",
		InfoURL:     "https://wikipedia.org/wiki/Comma-separated_values",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JSONToCSV) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Cell delimiter", Type: core.ArgString, Value: ","},
		{Name: "Row delimiter", Type: core.ArgString, Value: `\r\n`},
	}
}

// Run converts the JSON input into CSV.
func (JSONToCSV) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cellDelim := parseEscapedChars(args[0].(string))
	rowDelim := parseEscapedChars(args[1].(string))

	v, err := jsonParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("JSON to CSV: parse JSON input: %w", err)
	}
	out, err := jsonToCSV(v, cellDelim, rowDelim)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// jsonToCSV converts a parsed JSON value into CSV. It first tries the direct
// path; if a cell holds a nested container (or a header cannot be derived) that
// path "throws", and it retries against the flattened input with forced cell
// conversion, matching CyberChef's try/catch.
func jsonToCSV(val any, cellDelim, rowDelim string) (string, error) {
	if out, ok := csvBuild(csvWrap(val), cellDelim, rowDelim, false); ok {
		return out, nil
	}
	flat, err := csvFlatten(val)
	if err != nil {
		return "", errors.New("Unable to parse JSON to CSV") //nolint:staticcheck,revive // CyberChef's verbatim OperationError prefix
	}
	// After flattening, every cell is a primitive or an empty container, so the
	// forced build always succeeds.
	out, _ := csvBuild(csvWrap(flat), cellDelim, rowDelim, true)
	return out, nil
}

// csvWrap mirrors CyberChef's `if (!(input instanceof Array)) input = [input]`.
func csvWrap(val any) []any {
	if arr, ok := val.([]any); ok {
		return arr
	}
	return []any{val}
}

// csvBuild renders the rows as CSV, returning ok=false where CyberChef's toCSV
// would throw (an empty input, a non-array row in the array path, a header that
// cannot be read, or a nested cell without force) so the caller can fall back.
func csvBuild(rows []any, cellDelim, rowDelim string, force bool) (string, bool) {
	if len(rows) == 0 {
		return "", false // flattened[0] is undefined: Object.keys throws
	}

	// Array of arrays.
	if _, ok := rows[0].([]any); ok {
		lines := make([]string, 0, len(rows))
		for _, r := range rows {
			arr, ok := r.([]any)
			if !ok {
				return "", false // row.map on a non-array throws
			}
			cells := make([]string, 0, len(arr))
			for _, d := range arr {
				s, ok := csvEscapeCell(d, force, cellDelim, rowDelim)
				if !ok {
					return "", false
				}
				cells = append(cells, s)
			}
			lines = append(lines, strings.Join(cells, cellDelim))
		}
		return strings.Join(lines, rowDelim) + rowDelim, true
	}

	// Array of dictionaries.
	header, ok := csvKeys(rows[0])
	if !ok {
		return "", false // Object.keys(null) throws
	}
	headerCells := make([]string, 0, len(header))
	for _, h := range header {
		s, _ := csvEscapeCell(h, force, cellDelim, rowDelim) // header keys are strings, always ok
		headerCells = append(headerCells, s)
	}
	out := strings.Join(headerCells, cellDelim) + rowDelim

	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		cells := make([]string, 0, len(header))
		for _, h := range header {
			v, ok := csvGet(r, h)
			if !ok {
				return "", false // row[h] on null throws
			}
			s, ok := csvEscapeCell(v, force, cellDelim, rowDelim)
			if !ok {
				return "", false
			}
			cells = append(cells, s)
		}
		lines = append(lines, strings.Join(cells, cellDelim))
	}
	return out + strings.Join(lines, rowDelim) + rowDelim, true
}

// csvKeys reproduces Object.keys for the header row: object own keys, a string's
// character indices, [] for other primitives, and a throw (ok=false) for null.
func csvKeys(v any) ([]string, bool) {
	switch x := v.(type) {
	case jsObject:
		ordered := jsESOrder(x)
		keys := make([]string, len(ordered))
		for i, p := range ordered {
			keys[i] = p.k
		}
		return keys, true
	case string:
		keys := make([]string, len([]rune(x)))
		for i := range keys {
			keys[i] = strconv.Itoa(i)
		}
		return keys, true
	case nil:
		return nil, false
	default: // number, bool
		return []string{}, true
	}
}

// csvGet reproduces JS property access row[key]: object lookup (undefined if
// missing), a string/array element by numeric index, undefined for other
// primitives, and a throw (ok=false) for null.
func csvGet(r any, key string) (any, bool) {
	switch x := r.(type) {
	case jsObject:
		if i := jsIndex(x, key); i >= 0 {
			return x[i].v, true
		}
		return jsUndefined{}, true
	case string:
		runes := []rune(x)
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && i < len(runes) {
			return string(runes[i]), true
		}
		return jsUndefined{}, true
	case []any:
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && i < len(x) {
			return x[i], true
		}
		return jsUndefined{}, true
	case nil:
		return nil, false
	default: // number, bool
		return jsUndefined{}, true
	}
}

// csvEscapeCell stringifies and RFC 4180-escapes a cell value, matching
// CyberChef's escapeCellContents. A container value is stringified only with
// force; without it, ok=false signals the caller to fall back to flattening.
func csvEscapeCell(v any, force bool, cellDelim, rowDelim string) (string, bool) {
	var s string
	switch x := v.(type) {
	case string:
		s = x
	case float64:
		s = jsFormatNumber(x)
	case bool:
		if x {
			s = "true"
		} else {
			s = "false"
		}
	case nil:
		s = "null"
	case jsUndefined:
		s = "undefined"
	case jsObject, []any:
		if !force {
			return "", false
		}
		s = jsStringify(v, 0)
	default:
		return "", false
	}

	s = strings.ReplaceAll(s, `"`, `""`)
	if strings.Contains(s, cellDelim) || strings.Contains(s, rowDelim) ||
		strings.Contains(s, "\n") || strings.Contains(s, "\r") || strings.Contains(s, `"`) {
		s = `"` + s + `"`
	}
	return s, true
}

// csvFlatten flattens nested objects and arrays into a single object with
// dot-separated keys, matching the `flat` library's defaults: it recurses into
// non-empty containers and keeps empty ones and primitives as leaf values. It
// errors on a non-container top-level value (e.g. null).
func csvFlatten(val any) (any, error) {
	if _, ok := csvEntries(val); !ok {
		return nil, errors.New("cannot flatten non-container")
	}
	result := jsObject{}
	csvFlattenStep(val, "", &result)
	return result, nil
}

// csvFlattenStep recurses over a container (guaranteed by csvFlatten and the
// non-empty check below), appending leaf values under dotted keys.
func csvFlattenStep(cur any, prefix string, result *jsObject) {
	entries, _ := csvEntries(cur)
	for _, e := range entries {
		key := e.k
		if prefix != "" {
			key = prefix + "." + e.k
		}
		if csvNonEmptyContainer(e.v) {
			csvFlattenStep(e.v, key, result)
		} else {
			*result = append(*result, jsPair{k: key, v: e.v})
		}
	}
}

// csvEntries returns a container's key/value entries (object keys, or array
// indices as string keys), or ok=false for a non-container.
func csvEntries(cur any) ([]jsPair, bool) {
	switch x := cur.(type) {
	case jsObject:
		return []jsPair(jsESOrder(x)), true
	case []any:
		entries := make([]jsPair, len(x))
		for i, v := range x {
			entries[i] = jsPair{k: strconv.Itoa(i), v: v}
		}
		return entries, true
	default:
		return nil, false
	}
}

// csvNonEmptyContainer reports whether v is a non-empty object or array.
func csvNonEmptyContainer(v any) bool {
	switch x := v.(type) {
	case jsObject:
		return len(x) > 0
	case []any:
		return len(x) > 0
	default:
		return false
	}
}
