package ops

import (
	"errors"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(PHPDeserialize{})
}

// PHPDeserialize struct.
type PHPDeserialize struct{}

// Meta returns the operation metadata.
func (PHPDeserialize) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PHP Deserialize",
		Module:      "Default",
		Description: "Deserializes PHP serialized data, outputting keyed arrays as JSON.",
		InfoURL:     "https://wikipedia.org/wiki/Serialization#Programming_language_support",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PHPDeserialize) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Output valid JSON", Type: core.ArgBoolean, Value: true},
	}
}

// Run deserialises PHP serialized input.
func (PHPDeserialize) Run(in *core.Dish, args []any) (*core.Dish, error) {
	p := &phpDeserializer{runes: []rune(in.String()), validJSON: args[0].(bool)}
	v, err := p.handleInput()
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(phpStringify(v)), core.TypeString), nil
}

// phpDeserializer walks the serialized input rune by rune (JS iterates UTF-16
// units; runes match for BMP text, which serialized PHP strings are).
type phpDeserializer struct {
	runes     []rune
	pos       int
	validJSON bool
}

func (p *phpDeserializer) read(length int) (string, error) {
	if p.pos+length > len(p.runes) {
		return "", errors.New("End of input reached before end of script") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	s := string(p.runes[p.pos : p.pos+length])
	p.pos += length
	return s, nil
}

func (p *phpDeserializer) readUntil(until string) (string, error) {
	var b strings.Builder
	for {
		c, err := p.read(1)
		if err != nil {
			return "", err
		}
		if c == until {
			return b.String(), nil
		}
		b.WriteString(c)
	}
}

func (p *phpDeserializer) expect(expect string) error {
	got, err := p.read(len(expect))
	if err != nil {
		return err
	}
	if got != expect {
		return errors.New("Unexpected input found") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return nil
}

// handleInput deserialises one value, returning a string or a bool (for "b").
func (p *phpDeserializer) handleInput() (any, error) {
	kind, err := p.read(1)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(kind) {
	case "n":
		return "null", p.expect(";")
	case "i", "d", "b":
		if err := p.expect(":"); err != nil {
			return nil, err
		}
		data, err := p.readUntil(";")
		if err != nil {
			return nil, err
		}
		if kind == "b" {
			n, _ := strconv.Atoi(data)
			return n != 0, nil
		}
		return data, nil
	case "a":
		if err := p.expect(":"); err != nil {
			return nil, err
		}
		items, err := p.handleArray()
		if err != nil {
			return nil, err
		}
		return "{" + strings.Join(items, ",") + "}", nil
	case "s":
		return p.handleString()
	default:
		return nil, errors.New("Unknown type: " + kind) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
}

// handleArray reads count*2 alternating key/value entries into "key: value"
// strings, quoting integer keys when outputting valid JSON.
func (p *phpDeserializer) handleArray() ([]string, error) {
	countStr, err := p.readUntil(":")
	if err != nil {
		return nil, err
	}
	count, _ := strconv.Atoi(countStr)
	if err := p.expect("{"); err != nil {
		return nil, err
	}
	var result []string
	var lastItem string
	isKey := true
	for idx := 0; idx < count*2; idx++ {
		item, err := p.handleInput()
		if err != nil {
			return nil, err
		}
		if isKey {
			lastItem = phpStringify(item)
			isKey = false
			continue
		}
		if p.validJSON && phpAllDigits(lastItem) {
			result = append(result, `"`+lastItem+`": `+phpStringify(item))
		} else {
			result = append(result, lastItem+": "+phpStringify(item))
		}
		isKey = true
	}
	if err := p.expect("}"); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *phpDeserializer) handleString() (any, error) {
	if err := p.expect(":"); err != nil {
		return nil, err
	}
	lengthStr, err := p.readUntil(":")
	if err != nil {
		return nil, err
	}
	length, _ := strconv.Atoi(lengthStr)
	if err := p.expect(`"`); err != nil {
		return nil, err
	}
	value, err := p.read(length)
	if err != nil {
		return nil, err
	}
	if err := p.expect(`";`); err != nil {
		return nil, err
	}
	if p.validJSON {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`, nil
	}
	return `"` + value + `"`, nil
}

// phpStringify coerces a deserialised value (string or bool) to its output text,
// matching JS string coercion of booleans.
func phpStringify(v any) string {
	if b, ok := v.(bool); ok {
		if b {
			return "true"
		}
		return "false"
	}
	return v.(string)
}

// phpAllDigits reports whether s is non-empty and entirely ASCII digits (matching
// the original's /[0-9]+/ full-length check).
func phpAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
