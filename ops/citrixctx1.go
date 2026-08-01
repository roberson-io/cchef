package ops

import (
	"errors"

	"github.com/roberson-io/cchef/core"
)

// citrixCTX1CP is the codepage the Citrix CTX1 format runs over: 1200 = UTF-16LE.
const citrixCTX1CP = 1200

func init() {
	core.Register(CitrixCTX1Encode{})
	core.Register(CitrixCTX1Decode{})
}

// CitrixCTX1Encode encodes a string to the Citrix CTX1 password format.
type CitrixCTX1Encode struct{}

// Meta returns the operation metadata.
func (CitrixCTX1Encode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Citrix CTX1 Encode",
		Module:      "Encodings",
		Description: "Encodes strings to Citrix CTX1 password format.",
		InfoURL:     "https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (CitrixCTX1Encode) Args() []core.ArgDef { return nil }

// Run encodes the input. Ported from CyberChef CitrixCTX1Encode.mjs: the input
// is UTF-16LE encoded, each byte is folded into a running XOR chain against
// 0xa5, and the two nibbles of each result byte are emitted as 'A'..'P'.
func (CitrixCTX1Encode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	utf16pass, err := cptableEncode(citrixCTX1CP, in.String())
	if err != nil {
		return nil, err
	}
	result := make([]byte, 0, len(utf16pass)*2)
	temp := 0
	for _, b := range utf16pass {
		temp = int(b) ^ 0xa5 ^ temp
		result = append(result, byte(((temp>>4)&0xf)+0x41))
		result = append(result, byte((temp&0xf)+0x41))
	}
	return core.NewDish(result, core.TypeByteArray), nil
}

// CitrixCTX1Decode decodes a Citrix CTX1 password back to plaintext.
type CitrixCTX1Decode struct{}

// Meta returns the operation metadata.
func (CitrixCTX1Decode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Citrix CTX1 Decode",
		Module:      "Encodings",
		Description: "Decodes strings in a Citrix CTX1 password format to plaintext.",
		InfoURL:     "https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CitrixCTX1Decode) Args() []core.ArgDef { return nil }

// Run decodes the input. Ported from CyberChef CitrixCTX1Decode.mjs: the
// hex-nibble pairs are folded back through the XOR chain (over the reversed
// input) and the resulting bytes are decoded as UTF-16LE.
func (CitrixCTX1Decode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.Bytes()
	if len(input)%4 != 0 {
		return nil, errors.New("Incorrect hash length") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	revinput := make([]byte, len(input))
	for i := range input {
		revinput[i] = input[len(input)-1-i]
	}

	result := make([]byte, 0, len(revinput)/2)
	temp := 0
	for i := 0; i < len(revinput); i += 2 {
		if i+2 >= len(revinput) {
			temp = 0
		} else {
			temp = ((int(revinput[i+2]) - 0x41) & 0xf) ^ (((int(revinput[i+3]) - 0x41) << 4) & 0xf0)
		}
		temp = (((int(revinput[i]) - 0x41) & 0xf) ^ (((int(revinput[i+1]) - 0x41) << 4) & 0xf0)) ^ 0xa5 ^ temp
		result = append(result, byte(temp)) // #nosec G115 -- temp is an XOR of byte-sized values, so it is in [0,255]
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	out, err := cptableDecode(citrixCTX1CP, result)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
