package ops

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// bech32Charset is the Bech32 character set (32 chars, excludes 1, b, i, o).
const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

// bech32Err builds an error carrying CyberChef's verbatim OperationError text.
// Routing the message through this wrapper (rather than a direct fmt.Errorf)
// preserves the upstream capitalised, punctuated wording without tripping the
// lowercase/no-trailing-punctuation error-string linters.
func bech32Err(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

const (
	bech32Const  = 1          // BIP-0173 checksum constant
	bech32mConst = 0x2bc830a3 // BIP-0350 checksum constant
)

// bech32Generator holds the generator polynomial coefficients for the checksum.
var bech32Generator = [5]uint32{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

// bech32CharsetRev maps a Bech32 character to its 5-bit value, or -1.
var bech32CharsetRev = func() [256]int {
	var t [256]int
	for i := range t {
		t[i] = -1
	}
	for i := range len(bech32Charset) {
		t[bech32Charset[i]] = i
	}
	return t
}()

// bech32Polymod computes the polymod checksum over 5-bit values.
func bech32Polymod(values []int) uint32 {
	var chk uint32 = 1
	for _, v := range values {
		top := chk >> 25
		chk = ((chk & 0x1ffffff) << 5) ^ uint32(v) // #nosec G115 -- v is a 5-bit value (0-31)
		for i := range 5 {
			if (top>>i)&1 == 1 {
				chk ^= bech32Generator[i]
			}
		}
	}
	return chk
}

// bech32HrpExpand expands the HRP for checksum computation.
func bech32HrpExpand(hrp string) []int {
	result := make([]int, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		result = append(result, int(hrp[i])>>5)
	}
	result = append(result, 0)
	for i := 0; i < len(hrp); i++ {
		result = append(result, int(hrp[i])&31)
	}
	return result
}

// bech32VerifyChecksum reports whether data (including checksum) is valid.
func bech32VerifyChecksum(hrp string, data []int, encoding string) bool {
	constant := uint32(bech32Const)
	if encoding == "Bech32m" {
		constant = bech32mConst
	}
	return bech32Polymod(append(bech32HrpExpand(hrp), data...)) == constant
}

// bech32CreateChecksum returns the 6 checksum values for the encoding.
func bech32CreateChecksum(hrp string, data []int, encoding string) []int {
	constant := uint32(bech32Const)
	if encoding == "Bech32m" {
		constant = bech32mConst
	}
	values := append(bech32HrpExpand(hrp), data...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	mod := bech32Polymod(values) ^ constant
	result := make([]int, 6)
	for i := range 6 {
		result[i] = int((mod >> (5 * (5 - uint(i)))) & 31)
	}
	return result
}

// bech32ToWords converts 8-bit bytes to 5-bit words.
func bech32ToWords(data []byte) []int {
	var value uint32
	var bits uint
	result := []int{}
	for _, d := range data {
		value = (value << 8) | uint32(d)
		bits += 8
		for bits >= 5 {
			bits -= 5
			result = append(result, int((value>>bits)&31))
		}
	}
	if bits > 0 {
		result = append(result, int((value<<(5-bits))&31))
	}
	return result
}

// bech32FromWords converts 5-bit words to 8-bit bytes, validating the padding
// per BIP-0173.
func bech32FromWords(words []int) ([]byte, error) {
	var value uint32
	var bits uint
	result := []byte{}
	for _, w := range words {
		value = (value << 5) | uint32(w) // #nosec G115 -- w is a 5-bit word (0-31)
		bits += 5
		for bits >= 8 {
			bits -= 8
			result = append(result, byte((value>>bits)&255))
		}
	}
	if bits >= 5 {
		return nil, bech32Err("Invalid padding: too many bits remaining")
	}
	if bits > 0 {
		if (value<<(8-bits))&255 != 0 {
			return nil, bech32Err("Invalid padding: non-zero bits in padding")
		}
	}
	return result, nil
}

// bech32Encode encodes data to a Bech32/Bech32m string. When segwit is true and
// data has at least two bytes, the first byte is the witness version.
func bech32Encode(hrp string, data []byte, encoding string, segwit bool) (string, error) {
	if len(hrp) == 0 {
		return "", bech32Err("Human-Readable Part (HRP) cannot be empty.")
	}
	for i := 0; i < len(hrp); i++ {
		if hrp[i] < 33 || hrp[i] > 126 {
			return "", bech32Err("HRP contains invalid character at position %d. Only printable ASCII characters (33-126) are allowed.", i)
		}
	}

	hrpLower := strings.ToLower(hrp)

	var words []int
	if segwit && len(data) >= 2 {
		witnessVersion := data[0]
		if witnessVersion > 16 {
			return "", bech32Err("Invalid witness version: %d. Must be 0-16.", witnessVersion)
		}
		witnessProgram := data[1:]
		if len(witnessProgram) < 2 || len(witnessProgram) > 40 {
			return "", bech32Err("Invalid witness program length: %d. Must be 2-40 bytes.", len(witnessProgram))
		}
		if witnessVersion == 0 && len(witnessProgram) != 20 && len(witnessProgram) != 32 {
			return "", bech32Err("Invalid witness program length for v0: %d. Must be 20 or 32 bytes.", len(witnessProgram))
		}
		words = append([]int{int(witnessVersion)}, bech32ToWords(witnessProgram)...)
	} else {
		words = bech32ToWords(data)
	}

	checksum := bech32CreateChecksum(hrpLower, words, encoding)

	var sb strings.Builder
	sb.WriteString(hrpLower)
	sb.WriteByte('1')
	for _, w := range append(words, checksum...) {
		sb.WriteByte(bech32Charset[w])
	}
	result := sb.String()

	if len(result) > 90 {
		return "", bech32Err("Encoded string exceeds maximum length of 90 characters (got %d). Consider using smaller input data.", len(result))
	}
	return result, nil
}

// bech32Decoded is the result of decoding a Bech32/Bech32m string.
type bech32Decoded struct {
	hrp            string
	data           []byte
	encoding       string
	witnessVersion int // -1 when not a SegWit address
}

// bech32SegwitHrps lists the HRPs treated as Bitcoin SegWit during decoding.
var bech32SegwitHrps = map[string]bool{"bc": true, "tb": true, "ltc": true, "tltc": true, "bcrt": true}

// bech32Decode decodes a Bech32/Bech32m string.
func bech32Decode(str, encoding string) (*bech32Decoded, error) {
	if len(str) == 0 {
		return nil, bech32Err("Input cannot be empty.")
	}
	if len(str) > 90 {
		return nil, bech32Err("Invalid Bech32 string: exceeds maximum length of 90 characters (got %d).", len(str))
	}

	hasUpper := strings.ContainsFunc(str, func(r rune) bool { return r >= 'A' && r <= 'Z' })
	hasLower := strings.ContainsFunc(str, func(r rune) bool { return r >= 'a' && r <= 'z' })
	if hasUpper && hasLower {
		return nil, bech32Err("Invalid Bech32 string: mixed case is not allowed. Use all uppercase or all lowercase.")
	}

	str = strings.ToLower(str)

	sepIndex := strings.LastIndex(str, "1")
	if sepIndex == -1 {
		return nil, bech32Err("Invalid Bech32 string: no separator '1' found.")
	}
	if sepIndex == 0 {
		return nil, bech32Err("Invalid Bech32 string: Human-Readable Part (HRP) cannot be empty.")
	}
	if sepIndex+7 > len(str) {
		return nil, bech32Err("Invalid Bech32 string: data part is too short (minimum 6 characters for checksum).")
	}

	hrp := str[:sepIndex]
	dataPart := str[sepIndex+1:]

	for i := 0; i < len(hrp); i++ {
		if hrp[i] < 33 || hrp[i] > 126 {
			return nil, bech32Err("HRP contains invalid character at position %d.", i)
		}
	}

	data := make([]int, 0, len(dataPart))
	for i := 0; i < len(dataPart); i++ {
		v := bech32CharsetRev[dataPart[i]]
		if v == -1 {
			return nil, bech32Err("Invalid character '%c' at position %d.", dataPart[i], sepIndex+1+i)
		}
		data = append(data, v)
	}

	var usedEncoding string
	switch encoding {
	case "Bech32":
		if !bech32VerifyChecksum(hrp, data, "Bech32") {
			return nil, bech32Err("Invalid Bech32 checksum.")
		}
		usedEncoding = "Bech32"
	case "Bech32m":
		if !bech32VerifyChecksum(hrp, data, "Bech32m") {
			return nil, bech32Err("Invalid Bech32m checksum.")
		}
		usedEncoding = "Bech32m"
	default:
		switch {
		case bech32VerifyChecksum(hrp, data, "Bech32"):
			usedEncoding = "Bech32"
		case bech32VerifyChecksum(hrp, data, "Bech32m"):
			usedEncoding = "Bech32m"
		default:
			return nil, bech32Err("Invalid Bech32/Bech32m string: checksum verification failed.")
		}
	}

	words := data[:len(data)-6]

	couldBeSegWit := bech32SegwitHrps[hrp] && len(words) > 0 && words[0] <= 16

	var bytes []byte
	witnessVersion := -1

	if couldBeSegWit {
		witnessVersion = words[0]
		programBytes, err := bech32FromWords(words[1:])
		if err != nil {
			witnessVersion = -1
			bytes, err = bech32FromWords(words)
			if err != nil {
				return nil, bech32Err("Failed to decode data: %w", err)
			}
		} else {
			validV0 := witnessVersion == 0 && (len(programBytes) == 20 || len(programBytes) == 32)
			validOther := witnessVersion != 0 && len(programBytes) >= 2 && len(programBytes) <= 40
			if validV0 || validOther {
				bytes = append([]byte{byte(witnessVersion)}, programBytes...) // #nosec G115 -- witnessVersion is a decoded 5-bit word, guarded to 0-16
			} else {
				witnessVersion = -1
				bytes, err = bech32FromWords(words)
				if err != nil {
					return nil, bech32Err("Failed to decode data: %w", err)
				}
			}
		}
	} else {
		var err error
		bytes, err = bech32FromWords(words)
		if err != nil {
			return nil, bech32Err("Failed to decode data: %w", err)
		}
	}

	return &bech32Decoded{hrp: hrp, data: bytes, encoding: usedEncoding, witnessVersion: witnessVersion}, nil
}

// ToBech32 encodes data as a Bech32/Bech32m string.
type ToBech32 struct{}

// Meta returns the operation metadata.
func (ToBech32) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Bech32",
		Module:      "Default",
		Description: "Bech32 is an encoding scheme primarily used for Bitcoin SegWit addresses (BIP-0173). It uses a 32-character alphabet that excludes easily confused characters (1, b, i, o) and includes a checksum for error detection. Bech32m (BIP-0350) is an updated version used for Bitcoin Taproot addresses.",
		InfoURL:     "https://wikipedia.org/wiki/Bech32",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToBech32) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Human-Readable Part (HRP)", Type: core.ArgString, Value: "bc"},
		{Name: "Encoding", Type: core.ArgOption, Value: []string{"Bech32", "Bech32m"}},
		{Name: "Input Format", Type: core.ArgOption, Value: []string{"Raw bytes", "Hex"}},
		{Name: "Mode", Type: core.ArgOption, Value: []string{"Generic", "Bitcoin SegWit"}},
		{Name: "Witness Version", Type: core.ArgNumber, Value: 0},
	}
}

// Run encodes the input as Bech32/Bech32m.
func (ToBech32) Run(in *core.Dish, args []any) (*core.Dish, error) {
	hrp := args[0].(string)
	encoding := args[1].(string)
	inputFormat := args[2].(string)
	mode := args[3].(string)
	witnessVersion := int(args[4].(float64))

	var inputArray []byte
	if inputFormat == "Hex" {
		inputArray = fromHexAuto(string(in.Bytes()))
	} else {
		inputArray = in.Bytes()
	}

	if mode == "Bitcoin SegWit" {
		withVersion := make([]byte, len(inputArray)+1)
		withVersion[0] = byte(witnessVersion) // #nosec G115 -- matches CyberChef's Uint8Array truncation; out-of-range versions are rejected by bech32Encode
		copy(withVersion[1:], inputArray)
		out, err := bech32Encode(hrp, withVersion, encoding, true)
		if err != nil {
			return nil, err
		}
		return core.NewDish([]byte(out), core.TypeString), nil
	}

	out, err := bech32Encode(hrp, inputArray, encoding, false)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// FromBech32 decodes a Bech32/Bech32m string.
type FromBech32 struct{}

// Meta returns the operation metadata.
func (FromBech32) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Bech32",
		Module:      "Default",
		Description: "Bech32 is an encoding scheme primarily used for Bitcoin SegWit addresses (BIP-0173). It uses a 32-character alphabet that excludes easily confused characters (1, b, i, o) and includes a checksum for error detection. Auto-detect will attempt Bech32 first, then Bech32m if the checksum fails.",
		InfoURL:     "https://wikipedia.org/wiki/Bech32",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromBech32) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Encoding", Type: core.ArgOption, Value: []string{"Auto-detect", "Bech32", "Bech32m"}},
		{Name: "Output Format", Type: core.ArgOption, Value: []string{"Raw", "Hex", "Bitcoin scriptPubKey", "HRP: Hex", "JSON"}},
	}
}

// Run decodes the input from Bech32/Bech32m.
func (FromBech32) Run(in *core.Dish, args []any) (*core.Dish, error) {
	encoding := args[0].(string)
	outputFormat := args[1].(string)

	input := strings.TrimSpace(string(in.Bytes()))
	if len(input) == 0 {
		return core.NewDish([]byte(""), core.TypeString), nil
	}

	decoded, err := bech32Decode(input, encoding)
	if err != nil {
		return nil, err
	}

	switch outputFormat {
	case "Raw":
		var sb strings.Builder
		for _, b := range decoded.data {
			sb.WriteRune(rune(b))
		}
		return core.NewDish([]byte(sb.String()), core.TypeString), nil

	case "Bitcoin scriptPubKey":
		out := bech32ScriptPubKey(decoded)
		return core.NewDish([]byte(out), core.TypeString), nil

	case "HRP: Hex":
		out := fmt.Sprintf("%s: %s", decoded.hrp, toHex(decoded.data, "", ""))
		return core.NewDish([]byte(out), core.TypeString), nil

	case "JSON":
		out := bech32JSON(decoded.hrp, decoded.encoding, toHex(decoded.data, "", ""))
		return core.NewDish([]byte(out), core.TypeString), nil

	default: // "Hex" and any fallback
		return core.NewDish([]byte(toHex(decoded.data, "", "")), core.TypeString), nil
	}
}

// bech32ScriptPubKey renders a decoded SegWit address as a Bitcoin scriptPubKey
// hex string, falling back to plain hex when the input is not SegWit.
func bech32ScriptPubKey(decoded *bech32Decoded) string {
	if decoded.witnessVersion == -1 || len(decoded.data) < 2 {
		return toHex(decoded.data, "", "")
	}
	witnessVersion := decoded.data[0]
	witnessProgram := decoded.data[1:]

	var opCode byte
	switch {
	case witnessVersion == 0:
		opCode = 0x00
	case witnessVersion >= 1 && witnessVersion <= 16:
		opCode = 0x50 + witnessVersion
	default:
		return toHex(decoded.data, "", "")
	}

	scriptPubKey := append([]byte{opCode, byte(len(witnessProgram))}, witnessProgram...) // #nosec G115 -- a decoded SegWit witness program is 2-40 bytes
	return toHex(scriptPubKey, "", "")
}

// bech32JSON reproduces CyberChef's JSON.stringify(..., null, 2) output: a
// 2-space-indented object with keys in hrp/encoding/data order and no HTML
// escaping. Encoding a struct of strings cannot fail, so the error is discarded.
func bech32JSON(hrp, encoding, data string) string {
	obj := struct {
		HRP      string `json:"hrp"`
		Encoding string `json:"encoding"`
		Data     string `json:"data"`
	}{hrp, encoding, data}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(obj)
	return strings.TrimRight(b.String(), "\n")
}

func init() {
	core.Register(ToBech32{})
	core.Register(FromBech32{})
}
