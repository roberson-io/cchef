package ops

import (
	"errors"
	"math"

	"github.com/roberson-io/cchef/internal/core"
)

// ParseTLV converts a Type-Length-Value encoded string into JSON.
type ParseTLV struct{}

// Meta returns the operation metadata.
func (ParseTLV) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse TLV",
		Module:      "Default",
		Description: "Converts a Type-Length-Value (TLV) encoded string into a JSON object.  Can optionally include a Key / Type entry.",
		InfoURL:     "https://wikipedia.org/wiki/Type-length-value",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (ParseTLV) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Type/Key size", Type: core.ArgNumber, Integer: true, Value: 1},
		{Name: "Length size", Type: core.ArgNumber, Integer: true, Value: 1},
		{Name: "Use BER", Type: core.ArgBoolean, Value: false},
	}
}

// Run parses the input as TLV data.
func (ParseTLV) Run(in *core.Dish, args []any) (*core.Dish, error) {
	bytesInKey := int(args[0].(float64))
	bytesInLength := int(args[1].(float64))
	basicEncodingRules := args[2].(bool)

	if bytesInKey <= 0 && bytesInLength <= 0 {
		//nolint:staticcheck // CyberChef's verbatim OperationError text
		return nil, errors.New("Type or Length size must be greater than 0")
	}

	p := &tlvParser{input: in.Bytes(), bytesInLength: bytesInLength, ber: basicEncodingRules}

	data := []tlvEntry{}
	for !p.atEnd() {
		var keyPtr *[]any
		if bytesInKey != 0 {
			k := p.getValue(bytesInKey)
			keyPtr = &k
		}

		lengthF := p.getLength()
		var lengthOut any
		var value []any
		if math.IsNaN(lengthF) {
			value = []any{}
		} else {
			lengthOut = int(lengthF)
			value = p.getValue(int(lengthF))
		}

		data = append(data, tlvEntry{Key: keyPtr, Length: lengthOut, Value: value})
	}

	// tlvEntry holds only ints, nil, and slices thereof, so marshalling cannot
	// fail; the error is provably nil.
	out, _ := jsonNoEscape(data)
	return core.NewDish(out, core.TypeJSON), nil
}

// tlvEntry is one parsed TLV record. Key is omitted entirely when the key size
// is zero (mirroring CyberChef's undefined). Length is nil (JSON null) when the
// length bytes overrun the buffer, matching JS NaN.
type tlvEntry struct {
	Key    *[]any `json:"key,omitempty"`
	Length any    `json:"length"`
	Value  []any  `json:"value"`
}

// tlvParser is a faithful port of CyberChef's lib/TLVParser.mjs.
type tlvParser struct {
	input         []byte
	location      int
	bytesInLength int
	ber           bool
}

// byteAt returns the byte at i, or ok=false when i is past the end (JS
// undefined).
func (p *tlvParser) byteAt(i int) (int, bool) {
	if i < 0 || i >= len(p.input) {
		return 0, false
	}
	return int(p.input[i]), true
}

// atEnd reports whether the cursor has reached the end of the input.
func (p *tlvParser) atEnd() bool {
	return len(p.input) <= p.location
}

// getLength reads a length field. In BER mode the top bit of the first byte
// flags a big-endian long form whose remaining low bits give the byte count.
// A missing byte yields NaN, exactly as JS number arithmetic does.
func (p *tlvParser) getLength() float64 {
	bytesInLength := p.bytesInLength
	bigEndian := false

	if p.ber {
		first, ok := p.byteAt(p.location)
		p.location++
		firstLengthByte := 0
		if ok {
			firstLengthByte = first
		}
		if firstLengthByte&0x80 != 0 {
			bytesInLength = firstLengthByte &^ 0x80
			bigEndian = true
		} else {
			return float64(firstLengthByte &^ 0x80)
		}
	}

	length := 0.0
	for i := 0; i < bytesInLength; i++ {
		b, ok := p.byteAt(p.location)
		bv := math.NaN()
		if ok {
			bv = float64(b)
		}
		if bigEndian {
			length = float64(toInt32(length)<<8) + bv
		} else {
			length += bv * math.Pow(math.Pow(2, 8), float64(i))
		}
		p.location++
	}

	return length
}

// getValue reads length bytes from the cursor. Mirroring the JS parser's `>`
// (not `>=`) bound, it appends one trailing null when the length overruns.
func (p *tlvParser) getValue(length int) []any {
	value := []any{}
	for range length {
		if p.location > len(p.input) {
			return value
		}
		if b, ok := p.byteAt(p.location); ok {
			value = append(value, b)
		} else {
			value = append(value, nil)
		}
		p.location++
	}
	return value
}

// toInt32 applies JavaScript's ToInt32 abstract operation, used by the `<<`
// operator in BER big-endian length decoding.
func toInt32(f float64) int32 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	n := math.Trunc(f)
	m := math.Mod(n, 4294967296)
	if m < 0 {
		m += 4294967296
	}
	if m >= 2147483648 {
		m -= 4294967296
	}
	return int32(m)
}

func init() {
	core.Register(ParseTLV{})
}
