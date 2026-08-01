package ops

import (
	"crypto/aes"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(AESKeyWrap{})
	core.Register(AESKeyUnwrap{})
}

// aesKeyWrapArgs are the argument definitions shared by AES Key Wrap and Unwrap.
func aesKeyWrapArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key (KEK)", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "a6a6a6a6a6a6a6a6", ToggleValues: aesToggleValues},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// aesKeyWrapInputs decodes and validates the KEK, IV and input shared by both
// key-wrap operations. minBlocks is the minimum number of 64-bit input blocks.
func aesKeyWrapInputs(in *core.Dish, args []any, minBlocks int) (kek, iv, input []byte, outputType string, err error) {
	kekArg := args[0].(core.ToggleString)
	ivArg := args[1].(core.ToggleString)
	inputType := args[2].(string)
	outputType = args[3].(string)

	if kek, err = convertToByteArray(kekArg.Value, kekArg.Option); err != nil {
		return
	}
	if iv, err = convertToByteArray(ivArg.Value, ivArg.Option); err != nil {
		return
	}
	if len(kek) != 16 && len(kek) != 24 && len(kek) != 32 {
		err = fmt.Errorf("KEK must be either 16, 24, or 32 bytes (currently %d bytes)", len(kek))
		return
	}
	if len(iv) != 8 {
		err = fmt.Errorf("IV must be 8 bytes (currently %d bytes)", len(iv))
		return
	}
	input = decodeAESInput(in, inputType)
	if len(input)%8 != 0 || len(input) < minBlocks*8 {
		err = fmt.Errorf("input must be 8n (n>=%d) bytes (currently %d bytes)", minBlocks, len(input))
		return
	}
	return kek, iv, input, outputType, nil
}

// aesKeyWrapOutput encodes the result as hex or raw bytes.
func aesKeyWrapOutput(data []byte, outputType string) *core.Dish {
	if outputType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(data)), core.TypeString)
	}
	return core.NewDish(data, core.TypeString)
}

// --- AES Key Wrap ---

// AESKeyWrap wraps key material using the RFC3394 AES key-wrap algorithm.
// Ported from CyberChef AESKeyWrap.mjs.
type AESKeyWrap struct{}

// Meta returns the operation metadata.
func (AESKeyWrap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AES Key Wrap",
		Module:      "Ciphers",
		Description: "A key wrapping algorithm defined in RFC3394, which is used to protect keys in untrusted storage or communications, using AES. This algorithm uses an AES key (KEK: key-encryption key) and a 64-bit IV to encrypt 64-bit blocks.",
		InfoURL:     "https://wikipedia.org/wiki/Key_wrap",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AESKeyWrap) Args() []core.ArgDef { return aesKeyWrapArgs() }

// Run performs the key wrap.
func (AESKeyWrap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	kek, iv, input, outputType, err := aesKeyWrapInputs(in, args, 2)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	// RFC3394 wrap: A is the 64-bit integrity register, R the array of blocks.
	a := make([]byte, 8)
	copy(a, iv)
	n := len(input) / 8
	r := make([][]byte, n)
	for i := range r {
		r[i] = append([]byte{}, input[i*8:i*8+8]...)
	}
	buf := make([]byte, 16)
	var cnt uint64 = 1
	for range 6 {
		for i := range n {
			copy(buf[:8], a)
			copy(buf[8:], r[i])
			block.Encrypt(buf, buf)
			binary.BigEndian.PutUint64(a, binary.BigEndian.Uint64(buf[:8])^cnt)
			copy(r[i], buf[8:])
			cnt++
		}
	}

	out := append([]byte{}, a...)
	for _, blk := range r {
		out = append(out, blk...)
	}
	return aesKeyWrapOutput(out, outputType), nil
}

// --- AES Key Unwrap ---

// AESKeyUnwrap unwraps key material using the RFC3394 AES key-wrap algorithm.
// Ported from CyberChef AESKeyUnwrap.mjs.
type AESKeyUnwrap struct{}

// Meta returns the operation metadata.
func (AESKeyUnwrap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AES Key Unwrap",
		Module:      "Ciphers",
		Description: "Decryptor for a key wrapping algorithm defined in RFC3394, which is used to protect keys in untrusted storage or communications, using AES. This algorithm uses an AES key (KEK: key-encryption key) and a 64-bit IV to decrypt 64-bit blocks.",
		InfoURL:     "https://wikipedia.org/wiki/Key_wrap",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AESKeyUnwrap) Args() []core.ArgDef { return aesKeyWrapArgs() }

// Run performs the key unwrap.
func (AESKeyUnwrap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	kek, iv, input, outputType, err := aesKeyWrapInputs(in, args, 3)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	// RFC3394 unwrap: A starts from the first block, R holds the remaining ones.
	a := append([]byte{}, input[:8]...)
	n := len(input)/8 - 1
	r := make([][]byte, n)
	for i := range r {
		r[i] = append([]byte{}, input[(i+1)*8:(i+2)*8]...)
	}
	buf := make([]byte, 16)
	cnt := uint64(n) * 6 // #nosec G115 -- n is a positive 64-bit block count (input validated >= 24 bytes)
	for range 6 {
		for i := n - 1; i >= 0; i-- {
			binary.BigEndian.PutUint64(a, binary.BigEndian.Uint64(a)^cnt)
			copy(buf[:8], a)
			copy(buf[8:], r[i])
			block.Decrypt(buf, buf)
			copy(a, buf[:8])
			copy(r[i], buf[8:])
			cnt--
		}
	}
	if subtle.ConstantTimeCompare(a, iv) != 1 {
		return nil, errors.New("IV mismatch") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	var out []byte
	for _, blk := range r {
		out = append(out, blk...)
	}
	return aesKeyWrapOutput(out, outputType), nil
}
