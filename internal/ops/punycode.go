package ops

import (
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToPunycode{})
	core.Register(FromPunycode{})
}

// RFC 3492 Bootstring parameters for Punycode, mirroring punycode.js.
const (
	punyBase          = 36
	punyTMin          = 1
	punyTMax          = 26
	punySkew          = 38
	punyDamp          = 700
	punyInitialBias   = 72
	punyInitialN      = 128
	punyDelimiter     = '-'
	punyBaseMinusTMin = punyBase - punyTMin
	punyMaxInt        = 2147483647 // 2^31 - 1
	punyMaxCodePoint  = 0x10FFFF
)

// punycode.js surfaces these RangeError messages verbatim; kept as-is for
// fidelity.
var (
	errPunyOverflow     = errors.New("Overflow: input needs wider integers to process") //nolint:staticcheck // verbatim punycode.js message
	errPunyNotBasic     = errors.New("Illegal input >= 0x80 (not a basic code point)")  //nolint:staticcheck // verbatim punycode.js message
	errPunyInvalidInput = errors.New("Invalid input")                                   //nolint:staticcheck // verbatim punycode.js message
	// errPunyCodePoint corresponds to the String.fromCodePoint RangeError thrown
	// when a decoded value lies outside the Unicode range.
	errPunyCodePoint = errors.New("decoded code point out of Unicode range")
)

const punyDesc = "Punycode is a way to represent Unicode with the limited character subset of ASCII supported by the Domain Name System."

// ToPunycode converts Unicode to Punycode.
//
// CyberChef wraps punycode.js's encode (raw) and toASCII (IDN); this is a
// from-scratch RFC 3492 port of both.
type ToPunycode struct{}

// Meta returns the operation metadata.
func (ToPunycode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Punycode",
		Module:      "Encodings",
		Description: punyDesc + "<br><br>e.g. <code>münchen</code> encodes to <code>mnchen-3ya</code>",
		InfoURL:     "https://wikipedia.org/wiki/Punycode",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToPunycode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Internationalised domain name", Type: core.ArgBoolean, Value: false},
	}
}

// Run encodes the input to Punycode.
func (ToPunycode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var out string
	var err error
	if args[0].(bool) {
		out, err = punyToASCII(in.String())
	} else {
		out, err = punyEncode(in.String())
	}
	if err != nil {
		return nil, fmt.Errorf("to Punycode: %w", err)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// FromPunycode converts Punycode to Unicode.
//
// CyberChef wraps punycode.js's decode (raw) and toUnicode (IDN); this is a
// from-scratch RFC 3492 port of both.
type FromPunycode struct{}

// Meta returns the operation metadata.
func (FromPunycode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Punycode",
		Module:      "Encodings",
		Description: punyDesc + "<br><br>e.g. <code>mnchen-3ya</code> decodes to <code>münchen</code>",
		InfoURL:     "https://wikipedia.org/wiki/Punycode",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromPunycode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Internationalised domain name", Type: core.ArgBoolean, Value: false},
	}
}

// Run decodes the Punycode input to Unicode.
func (FromPunycode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var out string
	var err error
	if args[0].(bool) {
		out, err = punyToUnicode(in.String())
	} else {
		out, err = punyDecode(in.String())
	}
	if err != nil {
		return nil, fmt.Errorf("from Punycode: %w", err)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// --- RFC 3492 core ---

// punyBasicToDigit converts a basic code point to its digit value (0..base-1),
// or base if it is not a digit character.
func punyBasicToDigit(cp int) int {
	switch {
	case cp >= 0x30 && cp < 0x3A:
		return 26 + (cp - 0x30)
	case cp >= 0x41 && cp < 0x5B:
		return cp - 0x41
	case cp >= 0x61 && cp < 0x7B:
		return cp - 0x61
	default:
		return punyBase
	}
}

// punyDigitToBasic converts a digit value (0..base-1) to its lowercase basic
// code point (punycode.js always encodes with flag 0).
func punyDigitToBasic(digit int) byte {
	if digit < 26 {
		return byte(digit + 'a') // #nosec G115 -- digit < 26, so digit+'a' is a lowercase ASCII byte
	}
	return byte(digit - 26 + '0') // #nosec G115 -- digit is 26..35, so digit-26+'0' is an ASCII digit byte
}

// punyAdapt is the bias adaptation function from RFC 3492 section 3.4.
func punyAdapt(delta, numPoints int, firstTime bool) int {
	if firstTime {
		delta /= punyDamp
	} else {
		delta >>= 1
	}
	delta += delta / numPoints
	k := 0
	for delta > (punyBaseMinusTMin*punyTMax)>>1 {
		delta /= punyBaseMinusTMin
		k += punyBase
	}
	return k + (punyBaseMinusTMin+1)*delta/(delta+punySkew)
}

// punyThreshold returns the RFC 3492 threshold t for the given k and bias.
func punyThreshold(k, bias int) int {
	switch {
	case k <= bias:
		return punyTMin
	case k >= bias+punyTMax:
		return punyTMax
	default:
		return k - bias
	}
}

// punyEncode encodes a Unicode string to a raw Punycode string (no xn-- prefix).
func punyEncode(input string) (string, error) {
	codePoints := []rune(input)
	var output []byte

	n := punyInitialN
	delta := 0
	bias := punyInitialBias

	for _, c := range codePoints {
		if c < 0x80 {
			output = append(output, byte(c))
		}
	}
	basicLength := len(output)
	handledCPCount := basicLength
	if basicLength > 0 {
		output = append(output, punyDelimiter)
	}

	for handledCPCount < len(codePoints) {
		m := punyMaxInt
		for _, c := range codePoints {
			if int(c) >= n && int(c) < m {
				m = int(c)
			}
		}

		handledCPCountPlusOne := handledCPCount + 1
		if m-n > (punyMaxInt-delta)/handledCPCountPlusOne {
			return "", errPunyOverflow
		}
		delta += (m - n) * handledCPCountPlusOne
		n = m

		for _, c := range codePoints {
			cv := int(c)
			if cv < n {
				delta++
				if delta > punyMaxInt {
					return "", errPunyOverflow
				}
			}
			if cv == n {
				q := delta
				for k := punyBase; ; k += punyBase {
					t := punyThreshold(k, bias)
					if q < t {
						break
					}
					qMinusT := q - t
					baseMinusT := punyBase - t
					output = append(output, punyDigitToBasic(t+qMinusT%baseMinusT))
					q = qMinusT / baseMinusT
				}
				output = append(output, punyDigitToBasic(q))
				bias = punyAdapt(delta, handledCPCountPlusOne, handledCPCount == basicLength)
				delta = 0
				handledCPCount++
			}
		}
		delta++
		n++
	}
	return string(output), nil
}

// punyDecode decodes a raw Punycode string (no xn-- prefix) to Unicode.
func punyDecode(input string) (string, error) {
	r := []rune(input)
	var output []int

	basic := 0
	for idx, c := range r {
		if c == punyDelimiter {
			basic = idx
		}
	}
	for j := 0; j < basic; j++ {
		if r[j] >= 0x80 {
			return "", errPunyNotBasic
		}
		output = append(output, int(r[j]))
	}

	i := 0
	n := punyInitialN
	bias := punyInitialBias
	index := 0
	if basic > 0 {
		index = basic + 1
	}

	for index < len(r) {
		oldi := i
		w := 1
		for k := punyBase; ; k += punyBase {
			if index >= len(r) {
				return "", errPunyInvalidInput
			}
			digit := punyBasicToDigit(int(r[index]))
			index++
			if digit >= punyBase {
				return "", errPunyInvalidInput
			}
			if digit > (punyMaxInt-i)/w {
				return "", errPunyOverflow
			}
			i += digit * w
			t := punyThreshold(k, bias)
			if digit < t {
				break
			}
			baseMinusT := punyBase - t
			if w > punyMaxInt/baseMinusT {
				return "", errPunyOverflow
			}
			w *= baseMinusT
		}

		out := len(output) + 1
		bias = punyAdapt(i-oldi, out, oldi == 0)
		if i/out > punyMaxInt-n {
			return "", errPunyOverflow
		}
		n += i / out
		i %= out

		if n > punyMaxCodePoint {
			return "", errPunyCodePoint
		}
		output = append(output, 0)
		copy(output[i+1:], output[i:])
		output[i] = n
		i++
	}

	var sb strings.Builder
	for _, cp := range output {
		sb.WriteRune(rune(cp)) // #nosec G115 -- basic code points are < 0x80 and inserted ones are bounded to <= 0x10FFFF above
	}
	return sb.String(), nil
}

// --- IDN layer ---

// punyIsSeparator reports whether r is one of the RFC 3490 label separators.
func punyIsSeparator(r rune) bool {
	switch r {
	case '.', '。', '．', '｡':
		return true
	default:
		return false
	}
}

// punyMapDomain applies cb to each dot-separated label of a domain name or
// email address, mirroring punycode.js's mapDomain: only the domain part of an
// email is processed (and, matching the library, only the first two @-separated
// parts are kept), and the RFC 3490 separators are first normalised to '.'.
func punyMapDomain(input string, cb func(string) (string, error)) (string, error) {
	result := ""
	domain := input
	if parts := strings.Split(input, "@"); len(parts) > 1 {
		result = parts[0] + "@"
		domain = parts[1]
	}
	domain = strings.Map(func(r rune) rune {
		if punyIsSeparator(r) {
			return '.'
		}
		return r
	}, domain)

	labels := strings.Split(domain, ".")
	for idx, label := range labels {
		enc, err := cb(label)
		if err != nil {
			return "", err
		}
		labels[idx] = enc
	}
	return result + strings.Join(labels, "."), nil
}

// punyHasNonASCII reports whether s contains a code point above U+007F.
func punyHasNonASCII(s string) bool {
	for _, r := range s {
		if r > 0x7F {
			return true
		}
	}
	return false
}

// punyToASCII converts a Unicode domain name to its ASCII (xn--) form.
func punyToASCII(input string) (string, error) {
	return punyMapDomain(input, func(label string) (string, error) {
		if !punyHasNonASCII(label) {
			return label, nil
		}
		enc, err := punyEncode(label)
		if err != nil {
			return "", err
		}
		return "xn--" + enc, nil
	})
}

// punyToUnicode converts an ASCII (xn--) domain name back to Unicode. The ACE
// prefix match is case-sensitive, matching punycode.js's /^xn--/ regex.
func punyToUnicode(input string) (string, error) {
	return punyMapDomain(input, func(label string) (string, error) {
		if !strings.HasPrefix(label, "xn--") {
			return label, nil
		}
		return punyDecode(strings.ToLower(label[4:]))
	})
}
