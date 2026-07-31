package ops

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(XORBruteForce{})
}

// XORBruteForce enumerates every XOR key up to the configured length, optionally
// filtering results by a crib (known plaintext).
type XORBruteForce struct{}

// Meta returns the operation metadata.
func (XORBruteForce) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XOR Brute Force",
		Module:      "Default",
		Description: "Enumerate all possible XOR solutions. Current maximum key length is 2 due to browser performance. Optionally enter a string that you expect to find in the plaintext to filter results (crib).",
		InfoURL:     "https://wikipedia.org/wiki/Exclusive_or",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XORBruteForce) Args() []core.ArgDef {
	keyMin, keyMax := 1.0, 2.0
	return []core.ArgDef{
		{Name: "Key length", Type: core.ArgNumber, Integer: true, Value: 1, Min: &keyMin, Max: &keyMax},
		{Name: "Sample length", Type: core.ArgNumber, Integer: true, Value: 100},
		{Name: "Sample offset", Type: core.ArgNumber, Integer: true, Value: 0},
		{Name: "Scheme", Type: core.ArgOption, Value: []string{"Standard", "Input differential", "Output differential"}},
		{Name: "Null preserving", Type: core.ArgBoolean, Value: false},
		{Name: "Print key", Type: core.ArgBoolean, Value: true},
		{Name: "Output as hex", Type: core.ArgBoolean, Value: false},
		{Name: "Crib (known plaintext string)", Flag: "crib", Type: core.ArgString, Value: ""},
	}
}

// Run enumerates the keys. Ported from CyberChef XORBruteForce.mjs.
func (XORBruteForce) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyLength := int(args[0].(float64))
	sampleLength := int(args[1].(float64))
	sampleOffset := int(args[2].(float64))
	scheme := args[3].(string)
	nullPreserving := args[4].(bool)
	printKey := args[5].(bool)
	outputHex := args[6].(bool)
	crib := strings.ToLower(args[7].(string))

	input := sampleSlice(in.Bytes(), sampleOffset, sampleLength)

	var output []string
	limit := 1
	for range keyLength {
		limit *= 256
	}
	for key := 1; key < limit; key++ {
		result := bitOp(input, intToByteArray(key, keyLength), xorByte, nullPreserving, scheme)
		resultUtf8 := byteArrayToUtf8(result)
		if crib != "" && !strings.Contains(strings.ToLower(resultUtf8), crib) {
			continue
		}
		var record strings.Builder
		if printKey {
			record.WriteString("Key = " + fmt.Sprintf("%0*x", 2*keyLength, key) + ": ")
		}
		if outputHex {
			record.WriteString(toHexSpace(result))
		} else {
			record.WriteString(escapeWhitespace(resultUtf8))
		}
		output = append(output, record.String())
	}
	return core.NewDish([]byte(strings.Join(output, "\n")), core.TypeString), nil
}

// sampleSlice returns input[offset : offset+length], clamped to the bounds of
// input (mirroring Array.prototype.slice for non-negative arguments).
func sampleSlice(input []byte, offset, length int) []byte {
	if offset < 0 {
		offset = 0
	}
	if offset > len(input) {
		offset = len(input)
	}
	end := min(max(offset+length, offset), len(input))
	return input[offset:end]
}

// intToByteArray expresses int as a big-endian byte array of the given length
// (CyberChef's local intToByteArray helper).
func intToByteArray(n, length int) []byte {
	res := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		res[i] = byte(n & 0xff)
		n >>= 8
	}
	return res
}

// byteArrayToUtf8 decodes bytes as UTF-8, falling back to a per-byte character
// mapping (Latin1) when the bytes are not valid UTF-8. Ported from
// Utils.byteArrayToUtf8 / Utils.byteArrayToChars.
func byteArrayToUtf8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	for _, by := range b {
		sb.WriteRune(rune(by))
	}
	return sb.String()
}

// escapeWhitespace maps control characters 0x09–0x10 into the Private Use Area
// (0xE000 + code) so they render, matching Utils.escapeWhitespace.
func escapeWhitespace(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= 0x09 && r <= 0x10 {
			sb.WriteRune(0xe000 + r)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// toHexSpace renders bytes as space-delimited two-digit lowercase hex, matching
// lib/Hex.mjs toHex with its default delimiter.
func toHexSpace(b []byte) string {
	parts := make([]string, len(b))
	for i, by := range b {
		parts[i] = fmt.Sprintf("%02x", by)
	}
	return strings.Join(parts, " ")
}
