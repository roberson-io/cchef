package ops

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Salsa20{})
	core.Register(XSalsa20{})
}

// salsa20Permute applies the Salsa20 permutation (rounds/2 double-rounds) to the
// 16-word state in place.
func salsa20Permute(x *[16]uint32, rounds int) {
	qr := func(a, b, c, d int) {
		x[b] ^= bits.RotateLeft32(x[a]+x[d], 7)
		x[c] ^= bits.RotateLeft32(x[b]+x[a], 9)
		x[d] ^= bits.RotateLeft32(x[c]+x[b], 13)
		x[a] ^= bits.RotateLeft32(x[d]+x[c], 18)
	}
	for range rounds / 2 {
		qr(0, 4, 8, 12)
		qr(5, 9, 13, 1)
		qr(10, 14, 2, 6)
		qr(15, 3, 7, 11)
		qr(0, 1, 2, 3)
		qr(5, 6, 7, 4)
		qr(10, 11, 8, 9)
		qr(15, 12, 13, 14)
	}
}

// salsaSetup returns the 16-byte sigma/tau constant and the (expanded) key: a
// 16-byte key is doubled to 32 bytes and uses the "expand 16-byte k" constant.
func salsaSetup(key []byte) (constant, expandedKey []byte) {
	if len(key) == 16 {
		return []byte("expand 16-byte k"), append(append([]byte{}, key...), key...)
	}
	return []byte("expand 32-byte k"), key
}

// salsaWords reads a 64-byte state buffer as 16 little-endian words.
func salsaWords(state [64]byte) [16]uint32 {
	var x [16]uint32
	for i := range 16 {
		x[i] = binary.LittleEndian.Uint32(state[i*4:])
	}
	return x
}

// salsa20Block computes one 64-byte Salsa20 keystream block. The nonce is 8 bytes
// and the counter 8 bytes; a shorter nonce leaves the trailing state bytes zero.
func salsa20Block(key, nonce, counter []byte, rounds int) []byte {
	c, k := salsaSetup(key)
	var state [64]byte
	copy(state[0:4], c[0:4])
	copy(state[4:20], k[0:16])
	copy(state[20:24], c[4:8])
	copy(state[24:32], nonce)
	copy(state[32:40], counter)
	copy(state[40:44], c[8:12])
	copy(state[44:60], k[16:32])
	copy(state[60:64], c[12:16])

	x := salsaWords(state)
	a := x
	salsa20Permute(&x, rounds)
	out := make([]byte, 64)
	for i := range 16 {
		binary.LittleEndian.PutUint32(out[i*4:], x[i]+a[i])
	}
	return out
}

// hsalsa20 derives a 32-byte subkey from a key and a 16-byte nonce (used by
// XSalsa20). Unlike salsa20Block there is no feed-forward addition.
func hsalsa20(key, nonce []byte, rounds int) []byte {
	c, k := salsaSetup(key)
	var state [64]byte
	copy(state[0:4], c[0:4])
	copy(state[4:20], k[0:16])
	copy(state[20:24], c[4:8])
	copy(state[24:40], nonce)
	copy(state[40:44], c[8:12])
	copy(state[44:60], k[16:32])
	copy(state[60:64], c[12:16])

	x := salsaWords(state)
	salsa20Permute(&x, rounds)
	idx := []int{0, 5, 10, 15, 6, 7, 8, 9}
	out := make([]byte, 32)
	for i, j := range idx {
		binary.LittleEndian.PutUint32(out[i*4:], x[j])
	}
	return out
}

// salsa20XOR runs the keystream over input, one 64-byte block per counter value.
func salsa20XOR(input, key, nonce []byte, counter uint64, rounds int) []byte {
	out := make([]byte, len(input))
	var ctr [8]byte
	for i := 0; i < len(input); i += 64 {
		binary.LittleEndian.PutUint64(ctr[:], counter)
		stream := salsa20Block(key, nonce, ctr[:], rounds)
		for j := 0; j < 64 && i+j < len(input); j++ {
			out[i+j] = input[i+j] ^ stream[j]
		}
		counter++
	}
	return out
}

// salsaKey converts and validates a Salsa key (16 or 32 bytes); name is the
// cipher name for the verbatim error message.
func salsaKey(arg core.ToggleString, name string) ([]byte, error) {
	key, err := convertToByteArray(arg.Value, arg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 32 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid key length: %d bytes.\n\n%s uses a key of 16 or 32 bytes (128 or 256 bits).", len(key), name)
	}
	return key, nil
}

// salsaNonce converts the nonce. The "Integer" type parses a number into 8
// little-endian bytes; otherwise the byte length must equal expectedLen.
func salsaNonce(arg core.ToggleString, expectedLen int, name string) ([]byte, error) {
	if arg.Option == "Integer" {
		v, _ := leadingInt(arg.Value)
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(v)) // #nosec G115 -- reproduces JS intToByteArray of the counter integer
		return b, nil
	}
	nonce, err := convertToByteArray(arg.Value, arg.Option)
	if err != nil {
		return nil, err
	}
	if len(nonce) != expectedLen {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid nonce length: %d bytes.\n\n%s uses a nonce of %d bytes (%d bits).", len(nonce), name, expectedLen, expectedLen*8)
	}
	return nonce, nil
}

// salsaOutput formats the ciphertext as space-delimited hex or a raw string
// (matching CyberChef's toHex / arrayBufferToStr).
func salsaOutput(out []byte, outType string) *core.Dish {
	if outType == "Hex" {
		return core.NewDish([]byte(toHexSpace(out)), core.TypeString)
	}
	return core.NewDish([]byte(byteArrayToUtf8(out)), core.TypeString)
}

// salsaNonceToggles are the nonce input formats (Salsa's toggle adds "Integer").
var salsaNonceToggles = []string{"Hex", "UTF8", "Latin1", "Base64", "Integer"}

// salsaArgs builds the shared Salsa20/XSalsa20 argument list.
func salsaArgs() []core.ArgDef {
	minCtr := 0.0
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Nonce", Type: core.ArgToggleString, Value: "", ToggleValues: salsaNonceToggles},
		{Name: "Counter", Type: core.ArgNumber, Integer: true, Value: float64(0), Min: &minCtr},
		{Name: "Rounds", Type: core.ArgOption, Value: []string{"20", "12", "8"}},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// Salsa20 is the Salsa20 stream cipher (8-byte nonce).
type Salsa20 struct{}

// Meta returns the operation metadata.
func (Salsa20) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Salsa20",
		Module:      "Ciphers",
		Description: "Salsa20 is a stream cipher designed by Daniel J. Bernstein and submitted to the eSTREAM project; Salsa20/8 and Salsa20/12 are round-reduced variants. It is closely related to the ChaCha stream cipher.<br><br><b>Key:</b> Salsa20 uses a key of 16 or 32 bytes (128 or 256 bits).<br><br><b>Nonce:</b> Salsa20 uses a nonce of 8 bytes (64 bits).<br><br><b>Counter:</b> Salsa uses a counter of 8 bytes (64 bits). The counter starts at zero at the start of the keystream, and is incremented at every 64 bytes.",
		InfoURL:     "https://wikipedia.org/wiki/Salsa20",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Salsa20) Args() []core.ArgDef { return salsaArgs() }

// Run performs the Salsa20 stream cipher. Ported from CyberChef Salsa20.mjs.
func (Salsa20) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, err := salsaKey(args[0].(core.ToggleString), "Salsa20")
	if err != nil {
		return nil, err
	}
	nonce, err := salsaNonce(args[1].(core.ToggleString), 8, "Salsa20")
	if err != nil {
		return nil, err
	}
	counter := uint64(args[2].(float64))
	rounds, _ := strconv.Atoi(args[3].(string))
	input := decodeAESInput(in, args[4].(string))
	return salsaOutput(salsa20XOR(input, key, nonce, counter, rounds), args[5].(string)), nil
}

// XSalsa20 is the XSalsa20 stream cipher (24-byte nonce).
type XSalsa20 struct{}

// Meta returns the operation metadata.
func (XSalsa20) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XSalsa20",
		Module:      "Ciphers",
		Description: "XSalsa20 is a variant of the Salsa20 stream cipher designed by Daniel J. Bernstein; XSalsa uses longer nonces.<br><br><b>Key:</b> XSalsa20 uses a key of 16 or 32 bytes (128 or 256 bits).<br><br><b>Nonce:</b> XSalsa20 uses a nonce of 24 bytes (192 bits).<br><br><b>Counter:</b> XSalsa uses a counter of 8 bytes (64 bits). The counter starts at zero at the start of the keystream, and is incremented at every 64 bytes.",
		InfoURL:     "https://en.wikipedia.org/wiki/Salsa20#XSalsa20_with_192-bit_nonce",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XSalsa20) Args() []core.ArgDef { return salsaArgs() }

// Run performs the XSalsa20 stream cipher. Ported from CyberChef XSalsa20.mjs.
func (XSalsa20) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, err := salsaKey(args[0].(core.ToggleString), "XSalsa20")
	if err != nil {
		return nil, err
	}
	nonce, err := salsaNonce(args[1].(core.ToggleString), 24, "XSalsa20")
	if err != nil {
		return nil, err
	}
	counter := uint64(args[2].(float64))
	rounds, _ := strconv.Atoi(args[3].(string))
	input := decodeAESInput(in, args[4].(string))
	// A non-Integer nonce is validated to 24 bytes; an Integer nonce yields only 8
	// bytes (a degenerate input), so the sub-nonce slices are clamped like JS slice.
	xKey := hsalsa20(key, jsSliceBytes(nonce, 0, 16), rounds)
	return salsaOutput(salsa20XOR(input, xKey, jsSliceBytes(nonce, 16, 24), counter, rounds), args[5].(string)), nil
}

// jsSliceBytes returns b[start:end] clamped to the bounds of b, matching JavaScript's
// Array.prototype.slice for non-negative indices.
func jsSliceBytes(b []byte, start, end int) []byte {
	if start > len(b) {
		start = len(b)
	}
	if end > len(b) {
		end = len(b)
	}
	if end < start {
		end = start
	}
	return b[start:end]
}
