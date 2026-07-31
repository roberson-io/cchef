package ops

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ChaCha{})
}

// chachaCounterMin is the lower bound for the ChaCha "Counter" argument.
var chachaCounterMin = 0.0

// ChaCha is Daniel J. Bernstein's ChaCha stream cipher, in the parameterised
// form CyberChef ships: 16- or 32-byte key, 8- or 12-byte nonce (or an integer
// nonce), a counter, and 8/12/20 rounds.
type ChaCha struct{}

// Meta returns the operation metadata.
func (ChaCha) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ChaCha",
		Module:      "Ciphers",
		Description: "ChaCha is a stream cipher designed by Daniel J. Bernstein. It is a variant of the Salsa stream cipher. Several parameterizations exist; 'ChaCha' may refer to the original construction, or to the variant as described in RFC-8439. ChaCha is often used with Poly1305, in the ChaCha20-Poly1305 AEAD construction.<br><br><b>Key:</b> ChaCha uses a key of 16 or 32 bytes (128 or 256 bits).<br><br><b>Nonce:</b> ChaCha uses a nonce of 8 or 12 bytes (64 or 96 bits).<br><br><b>Counter:</b> ChaCha uses a counter of 4 or 8 bytes (32 or 64 bits); together, the nonce and counter must add up to 16 bytes. The counter starts at zero at the start of the keystream, and is incremented at every 64 bytes.",
		InfoURL:     "https://wikipedia.org/wiki/Salsa20#ChaCha_variant",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ChaCha) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
		{Name: "Nonce", Type: core.ArgToggleString, ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64", "Integer"}},
		{Name: "Counter", Type: core.ArgNumber, Integer: true, Value: 0, Min: &chachaCounterMin},
		{Name: "Rounds", Type: core.ArgOption, Value: []string{"20", "12", "8"}},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// chachaIntToLE encodes value as length little-endian bytes. It mirrors
// CyberChef's Utils.intToByteArray with byteorder "little", where the JS `>>> 8`
// and `& 0xFF` coerce the value to 32 bits, so only the low four bytes carry
// information (higher bytes are zero).
func chachaIntToLE(value uint64, length int) []byte {
	arr := make([]byte, length)
	v := uint32(value) // #nosec G115 -- mirrors JS >>>/& 32-bit coercion; truncation to low 32 bits is intentional
	for i := 0; i < length && i < 4; i++ {
		arr[i] = byte(v & 0xff)
		v >>= 8
	}
	return arr
}

// chachaLEToInt reads a little-endian byte array as an integer, mirroring
// CyberChef's Utils.byteArrayToInt with byteorder "little".
func chachaLEToInt(b []byte) uint64 {
	var v uint64
	for i := len(b) - 1; i >= 0; i-- {
		v = v*256 + uint64(b[i])
	}
	return v
}

// chachaBlock computes one 64-byte ChaCha keystream block for the given key,
// nonce, counter and round count. Ported from the chacha() helper in ChaCha.mjs.
func chachaBlock(key, nonce, counter []byte, rounds int) []byte {
	state := make([]byte, 0, 64)
	if len(key) == 16 {
		state = append(state, "expand 16-byte k"...)
		state = append(state, key...)
		state = append(state, key...)
	} else {
		state = append(state, "expand 32-byte k"...)
		state = append(state, key...)
	}
	state = append(state, counter...)
	state = append(state, nonce...)

	var x [16]uint32
	for i := range 16 {
		x[i] = binary.LittleEndian.Uint32(state[i*4 : i*4+4])
	}
	a := x

	quarterround := func(ai, bi, ci, di int) {
		x[ai] += x[bi]
		x[di] = bits.RotateLeft32(x[di]^x[ai], 16)
		x[ci] += x[di]
		x[bi] = bits.RotateLeft32(x[bi]^x[ci], 12)
		x[ai] += x[bi]
		x[di] = bits.RotateLeft32(x[di]^x[ai], 8)
		x[ci] += x[di]
		x[bi] = bits.RotateLeft32(x[bi]^x[ci], 7)
	}

	for i := 0; i < rounds/2; i++ {
		quarterround(0, 4, 8, 12)
		quarterround(1, 5, 9, 13)
		quarterround(2, 6, 10, 14)
		quarterround(3, 7, 11, 15)
		quarterround(0, 5, 10, 15)
		quarterround(1, 6, 11, 12)
		quarterround(2, 7, 8, 13)
		quarterround(3, 4, 9, 14)
	}

	out := make([]byte, 64)
	for i := range 16 {
		binary.LittleEndian.PutUint32(out[i*4:i*4+4], x[i]+a[i])
	}
	return out
}

// Run applies the cipher. Ported from CyberChef ChaCha.mjs.
func (ChaCha) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	nonceArg := args[1].(core.ToggleString)
	counterArg := int(args[2].(float64))
	rounds, _ := strconv.Atoi(args[3].(string))
	inputType := args[4].(string)
	outputType := args[5].(string)

	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 32 {
		return nil, fmt.Errorf("Invalid key length: %d bytes.\n\nChaCha uses a key of 16 or 32 bytes (128 or 256 bits).", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	var nonce []byte
	var counterLength int
	if nonceArg.Option == "Integer" {
		n, _ := strconv.ParseInt(nonceArg.Value, 10, 64)
		nonce = chachaIntToLE(uint64(n), 12) // #nosec G115 -- JS parseInt/intToByteArray coerce to 32 bits; wrap is intentional
		counterLength = 4
	} else {
		nonce, err = convertToByteArray(nonceArg.Value, nonceArg.Option)
		if err != nil {
			return nil, err
		}
		if len(nonce) != 12 && len(nonce) != 8 {
			return nil, fmt.Errorf("Invalid nonce length: %d bytes.\n\nChaCha uses a nonce of 8 or 12 bytes (64 or 96 bits).", len(nonce)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		counterLength = 16 - len(nonce)
	}
	counter := chachaIntToLE(uint64(counterArg), counterLength) // #nosec G115 -- counter is non-negative (Min 0)

	var data []byte
	if inputType == "Hex" {
		data = splitHexToBytes(in.String())
	} else {
		data = in.Bytes()
	}

	output := make([]byte, len(data))
	counterAsInt := chachaLEToInt(counter)
	for i := 0; i < len(data); i += 64 {
		counter = chachaIntToLE(counterAsInt, counterLength)
		stream := chachaBlock(key, nonce, counter, rounds)
		for j := 0; j < 64 && i+j < len(data); j++ {
			output[i+j] = data[i+j] ^ stream[j]
		}
		counterAsInt++
	}

	if outputType == "Hex" {
		return core.NewDish([]byte(toHex(output, " ", "")), core.TypeString), nil
	}
	return core.NewDish(output, core.TypeString), nil
}
