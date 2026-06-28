package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(URLEncode{})
	core.Register(URLDecode{})
}

// urlSafePartial is the set of characters left unencoded when "Encode all
// special chars" is false. Ported from CyberChef URLEncode.encodeBytes.
const urlSafePartial = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789" +
	":/?#[]@!$&'()*+,;=%"

func isURLSafe(b byte, encodeAll bool) bool {
	if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
		return true
	}
	if encodeAll {
		return false
	}
	return strings.IndexByte(urlSafePartial, b) >= 0
}

// URLEncode percent-encodes problematic characters into a URL-safe form.
type URLEncode struct{}

// Meta returns the operation metadata.
func (URLEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "URL Encode",
		Module:      "URL",
		Description: "Encodes problematic characters into percent-encoding, a format supported by URIs/URLs.",
		InfoURL:     "https://wikipedia.org/wiki/Percent-encoding",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (URLEncode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encode all special chars", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input.
func (URLEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	encodeAll := args[0].(bool)
	var sb strings.Builder
	for _, b := range in.Bytes() {
		if isURLSafe(b, encodeAll) {
			sb.WriteByte(b)
		} else {
			fmt.Fprintf(&sb, "%%%02X", b)
		}
	}
	return core.NewDish([]byte(sb.String()), core.TypeString), nil
}

// URLDecode converts percent-encoded characters back to their raw values.
type URLDecode struct{}

// Meta returns the operation metadata.
func (URLDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "URL Decode",
		Module:      "URL",
		Description: "Converts URI/URL percent-encoded characters back to their raw values.",
		InfoURL:     "https://wikipedia.org/wiki/Percent-encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (URLDecode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Treat \"+\" as space", Type: core.ArgBoolean, Value: true},
	}
}

// Run decodes the input. Ported from CyberChef URLDecode.mjs.
func (URLDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	plusIsSpace := args[0].(bool)
	data := in.String()
	if plusIsSpace {
		data = strings.ReplaceAll(data, "+", " ")
	}

	var out []byte
	for i := 0; i < len(data); i++ {
		if data[i] == '%' && i+2 < len(data) {
			if v, err := strconv.ParseUint(data[i+1:i+3], 16, 8); err == nil {
				out = append(out, byte(v))
				i += 2
				continue
			}
		}
		out = append(out, data[i])
	}
	return core.NewDish(out, core.TypeString), nil
}
