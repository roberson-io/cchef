package ops

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RC4{})
	core.Register(RC4Drop{})
}

// rc4Process runs RC4 over data, discarding dropWords 32-bit keystream words
// (4 keystream bytes each) first. Encryption and decryption are identical.
func rc4Process(key, data []byte, dropWords int) []byte {
	var s [256]byte
	for i := range s {
		s[i] = byte(i)
	}
	j := 0
	for i := range 256 {
		kb := 0
		if len(key) > 0 {
			kb = int(key[i%len(key)])
		}
		j = (j + int(s[i]) + kb) & 0xff
		s[i], s[j] = s[j], s[i]
	}
	i, j := 0, 0
	next := func() byte {
		i = (i + 1) & 0xff
		j = (j + int(s[i])) & 0xff
		s[i], s[j] = s[j], s[i]
		return s[(int(s[i])+int(s[j]))&0xff]
	}
	for range dropWords * 4 {
		next()
	}
	out := make([]byte, len(data))
	for k := range data {
		out[k] = data[k] ^ next()
	}
	return out
}

// rc4ErrMalformedUTF8 mirrors CryptoJS's error for UTF8-stringifying non-UTF-8
// bytes.
//
//nolint:staticcheck,revive // CryptoJS's verbatim error text
var rc4ErrMalformedUTF8 = errors.New("Malformed UTF-8 data")

// rc4Parse decodes s from the given CryptoJS format into bytes.
func rc4Parse(s, format string) ([]byte, error) {
	switch format {
	case "UTF16", "UTF16BE":
		return rc4EncodeUTF16(s, true), nil
	case "UTF16LE":
		return rc4EncodeUTF16(s, false), nil
	default: // Hex / Base64 / UTF8 / Latin1
		return convertToByteArray(s, format)
	}
}

// rc4Stringify encodes b into a string in the given CryptoJS format.
func rc4Stringify(b []byte, format string) (string, error) {
	switch format {
	case "Hex":
		return hex.EncodeToString(b), nil
	case "Base64":
		return base64.StdEncoding.EncodeToString(b), nil
	case "UTF8":
		if !utf8.Valid(b) {
			return "", rc4ErrMalformedUTF8
		}
		return string(b), nil
	case "UTF16", "UTF16BE":
		return rc4DecodeUTF16(b, true), nil
	case "UTF16LE":
		return rc4DecodeUTF16(b, false), nil
	default: // Latin1
		var sb strings.Builder
		for _, c := range b {
			sb.WriteRune(rune(c))
		}
		return sb.String(), nil
	}
}

// rc4EncodeUTF16 encodes s as UTF-16 (big- or little-endian) bytes, matching
// CryptoJS's Utf16/Utf16LE encoders.
func rc4EncodeUTF16(s string, bigEndian bool) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		if bigEndian {
			out = append(out, byte(u>>8), byte(u)) // #nosec G115 -- splitting a 16-bit code unit into bytes
		} else {
			out = append(out, byte(u), byte(u>>8)) // #nosec G115 -- splitting a 16-bit code unit into bytes
		}
	}
	return out
}

// rc4DecodeUTF16 decodes bytes as UTF-16 code units into a string. It is a
// best-effort inverse of rc4EncodeUTF16: a trailing odd byte is treated as a
// high byte, and lone surrogates decode to U+FFFD (Go strings cannot hold the
// lone surrogates CryptoJS keeps).
func rc4DecodeUTF16(b []byte, bigEndian bool) string {
	units := make([]uint16, 0, (len(b)+1)/2)
	for i := 0; i < len(b); i += 2 {
		var lo byte
		if i+1 < len(b) {
			lo = b[i+1]
		}
		if bigEndian {
			units = append(units, uint16(b[i])<<8|uint16(lo))
		} else {
			units = append(units, uint16(lo)<<8|uint16(b[i]))
		}
	}
	return string(utf16.Decode(units))
}

// rc4KeyFormats / rc4DataFormats are the CryptoJS encoders the RC4 operations
// offer for the passphrase and for the input/output respectively.
var (
	rc4KeyFormats  = []string{"UTF8", "UTF16", "UTF16LE", "UTF16BE", "Latin1", "Hex", "Base64"}
	rc4DataFormats = []string{"Latin1", "UTF8", "UTF16", "UTF16LE", "UTF16BE", "Hex", "Base64"}
)

// RC4 is the RC4 stream cipher.
type RC4 struct{}

// Meta returns the operation metadata.
func (RC4) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC4",
		Module:      "Ciphers",
		Description: "RC4 (also known as ARC4) is a widely-used stream cipher designed by Ron Rivest. It is used in popular protocols such as SSL and WEP. Although remarkable for its simplicity and speed, the algorithm's history doesn't inspire confidence in its security.",
		InfoURL:     "https://wikipedia.org/wiki/RC4",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC4) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Passphrase", Type: core.ArgToggleString, Value: "", ToggleValues: rc4KeyFormats},
		{Name: "Input format", Type: core.ArgOption, Value: rc4DataFormats},
		{Name: "Output format", Type: core.ArgOption, Value: rc4DataFormats},
	}
}

// rc4Run parses the key and input, runs RC4 (dropping dropWords keystream
// words), and formats the output.
func rc4Run(in *core.Dish, keyArg core.ToggleString, inFmt, outFmt string, dropWords int) (*core.Dish, error) {
	key, err := rc4Parse(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	data, err := rc4Parse(in.String(), inFmt)
	if err != nil {
		return nil, err
	}
	out, err := rc4Stringify(rc4Process(key, data, dropWords), outFmt)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// Run performs the RC4 stream cipher (encryption and decryption are identical).
func (RC4) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return rc4Run(in, args[0].(core.ToggleString), args[1].(string), args[2].(string), 0)
}

// RC4Drop is RC4 with an initial portion of the keystream discarded.
type RC4Drop struct{}

// Meta returns the operation metadata.
func (RC4Drop) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC4 Drop",
		Module:      "Ciphers",
		Description: "It was discovered that the first few bytes of the RC4 keystream are strongly non-random and leak information about the key. We can defend against this attack by discarding the initial portion of the keystream. This modified algorithm is traditionally called RC4-drop.",
		InfoURL:     "https://wikipedia.org/wiki/RC4#Fluhrer,_Mantin_and_Shamir_attack",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC4Drop) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Passphrase", Type: core.ArgToggleString, Value: "", ToggleValues: rc4KeyFormats},
		{Name: "Input format", Type: core.ArgOption, Value: rc4DataFormats},
		{Name: "Output format", Type: core.ArgOption, Value: rc4DataFormats},
		{Name: "Number of dwords to drop", Type: core.ArgNumber, Value: float64(192)},
	}
}

// Run performs RC4-drop.
func (RC4Drop) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return rc4Run(in, args[0].(core.ToggleString), args[1].(string), args[2].(string), int(args[3].(float64)))
}
