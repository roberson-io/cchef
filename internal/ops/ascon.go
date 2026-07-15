package ops

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(AsconEncrypt{})
	core.Register(AsconDecrypt{})
}

// asconToggleValues are the encodings offered for the Key, Nonce and Associated
// Data toggleString arguments.
var asconToggleValues = []string{"Hex", "UTF8", "Latin1", "Base64"}

// --- Ascon-AEAD128 core (NIST SP 800-232) ---
//
// Reimplemented from scratch to match the js-ascon package CyberChef wraps
// (github.com/brainfoolong/js-ascon, v1.3.0). The 320-bit state is five 64-bit
// words in little-endian byte order; AEAD128 uses 12 initialisation/finalisation
// rounds, 8 intermediate rounds and a 16-byte rate.

const (
	asconRoundsA = 12
	asconRoundsB = 8
	asconRate    = 16
)

// asconPermutation applies the Ascon-p round function `rounds` times.
func asconPermutation(s *[5]uint64, rounds int) {
	for round := 12 - rounds; round < 12; round++ {
		// round constant
		s[2] ^= uint64(0xf0 - round*0x10 + round) // #nosec G115 -- constant in [0x4b, 0xf0]
		// substitution layer
		s[0] ^= s[4]
		s[4] ^= s[3]
		s[2] ^= s[1]
		var t [5]uint64
		for i := range 5 {
			t[i] = (^s[i]) & s[(i+1)%5]
		}
		for i := range 5 {
			s[i] ^= t[(i+1)%5]
		}
		s[1] ^= s[0]
		s[0] ^= s[4]
		s[3] ^= s[2]
		s[2] ^= ^uint64(0)
		// linear diffusion layer
		s[0] ^= rotr64(s[0], 19) ^ rotr64(s[0], 28)
		s[1] ^= rotr64(s[1], 61) ^ rotr64(s[1], 39)
		s[2] ^= rotr64(s[2], 1) ^ rotr64(s[2], 6)
		s[3] ^= rotr64(s[3], 10) ^ rotr64(s[3], 17)
		s[4] ^= rotr64(s[4], 7) ^ rotr64(s[4], 41)
	}
}

// rotr64 rotates x right by k bits.
func rotr64(x uint64, k int) uint64 { return bits.RotateLeft64(x, -k) }

// asconInitialize seeds the state from the IV, key and nonce.
func asconInitialize(key, nonce []byte) [5]uint64 {
	iv := []byte{1, 0, (asconRoundsB << 4) + asconRoundsA, 0x80, 0x00, asconRate, 0, 0}
	s := [5]uint64{
		binary.LittleEndian.Uint64(iv),
		binary.LittleEndian.Uint64(key[0:8]),
		binary.LittleEndian.Uint64(key[8:16]),
		binary.LittleEndian.Uint64(nonce[0:8]),
		binary.LittleEndian.Uint64(nonce[8:16]),
	}
	asconPermutation(&s, asconRoundsA)
	s[3] ^= binary.LittleEndian.Uint64(key[0:8])
	s[4] ^= binary.LittleEndian.Uint64(key[8:16])
	return s
}

// asconPad appends the 0x01 padding byte followed by zeroes so the data length
// becomes a multiple of the 16-byte rate (a full extra block when already
// aligned).
func asconPad(data []byte) []byte {
	padLen := asconRate - len(data)%asconRate
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	out[len(data)] = 0x01
	return out
}

// asconAbsorbAD mixes the associated data into the state and applies the domain
// separation constant.
func asconAbsorbAD(s *[5]uint64, ad []byte) {
	if len(ad) > 0 {
		padded := asconPad(ad)
		for block := 0; block < len(padded); block += asconRate {
			s[0] ^= binary.LittleEndian.Uint64(padded[block : block+8])
			s[1] ^= binary.LittleEndian.Uint64(padded[block+8 : block+16])
			asconPermutation(s, asconRoundsB)
		}
	}
	s[4] ^= 1 << 63
}

// asconFinalize mixes the key back in and returns the 16-byte tag.
func asconFinalize(s *[5]uint64, key []byte) []byte {
	s[2] ^= binary.LittleEndian.Uint64(key[0:8])
	s[3] ^= binary.LittleEndian.Uint64(key[8:16])
	asconPermutation(s, asconRoundsA)
	s[3] ^= binary.LittleEndian.Uint64(key[0:8])
	s[4] ^= binary.LittleEndian.Uint64(key[8:16])
	tag := make([]byte, 16)
	binary.LittleEndian.PutUint64(tag[0:8], s[3])
	binary.LittleEndian.PutUint64(tag[8:16], s[4])
	return tag
}

// asconEncryptCore encrypts plaintext, returning ciphertext concatenated with
// the 16-byte authentication tag.
func asconEncryptCore(key, nonce, ad, plaintext []byte) []byte {
	s := asconInitialize(key, nonce)
	asconAbsorbAD(&s, ad)

	lastLen := len(plaintext) % asconRate
	padded := asconPad(plaintext)
	ct := make([]byte, 0, len(plaintext)+16)
	var w [8]byte
	// all blocks but the last
	for block := 0; block < len(padded)-asconRate; block += asconRate {
		s[0] ^= binary.LittleEndian.Uint64(padded[block : block+8])
		binary.LittleEndian.PutUint64(w[:], s[0])
		ct = append(ct, w[:]...)
		s[1] ^= binary.LittleEndian.Uint64(padded[block+8 : block+16])
		binary.LittleEndian.PutUint64(w[:], s[1])
		ct = append(ct, w[:]...)
		asconPermutation(&s, asconRoundsB)
	}
	// last (partial) block: emit only lastLen bytes of keystream-XORed output
	block := len(padded) - asconRate
	s[0] ^= binary.LittleEndian.Uint64(padded[block : block+8])
	s[1] ^= binary.LittleEndian.Uint64(padded[block+8 : block+16])
	var w0, w1 [8]byte
	binary.LittleEndian.PutUint64(w0[:], s[0])
	binary.LittleEndian.PutUint64(w1[:], s[1])
	ct = append(ct, w0[:min(8, lastLen)]...)
	ct = append(ct, w1[:max(0, lastLen-8)]...)

	return append(ct, asconFinalize(&s, key)...)
}

// asconDecryptCore decrypts ciphertext (without the tag) and returns the
// plaintext along with the recomputed tag for the caller to verify.
func asconDecryptCore(key, nonce, ad, ciphertext []byte) (plaintext, tag []byte) {
	s := asconInitialize(key, nonce)
	asconAbsorbAD(&s, ad)

	lastLen := len(ciphertext) % asconRate
	padded := make([]byte, len(ciphertext)+asconRate-lastLen)
	copy(padded, ciphertext)
	pt := make([]byte, 0, len(ciphertext))
	var w [8]byte
	// all blocks but the last
	for block := 0; block < len(padded)-asconRate; block += asconRate {
		ci := binary.LittleEndian.Uint64(padded[block : block+8])
		binary.LittleEndian.PutUint64(w[:], s[0]^ci)
		pt = append(pt, w[:]...)
		s[0] = ci
		ci = binary.LittleEndian.Uint64(padded[block+8 : block+16])
		binary.LittleEndian.PutUint64(w[:], s[1]^ci)
		pt = append(pt, w[:]...)
		s[1] = ci
		asconPermutation(&s, asconRoundsB)
	}
	// last (partial) block
	block := len(padded) - asconRate
	var paddingB, maskB [16]byte
	paddingB[lastLen] = 0x01
	for i := lastLen; i < asconRate; i++ {
		maskB[i] = 0xFF
	}

	ci0 := binary.LittleEndian.Uint64(padded[block : block+8])
	var lastPart [16]byte
	binary.LittleEndian.PutUint64(lastPart[0:8], s[0]^ci0)
	s[0] = (s[0] & binary.LittleEndian.Uint64(maskB[0:8])) ^ ci0 ^ binary.LittleEndian.Uint64(paddingB[0:8])

	ci1 := binary.LittleEndian.Uint64(padded[block+8 : block+16])
	var w1 [8]byte
	binary.LittleEndian.PutUint64(w1[:], s[1]^ci1)
	copy(lastPart[8:], w1[:min(8, lastLen)])
	s[1] = (s[1] & binary.LittleEndian.Uint64(maskB[8:16])) ^ ci1 ^ binary.LittleEndian.Uint64(paddingB[8:16])

	pt = append(pt, lastPart[:lastLen]...)

	return pt, asconFinalize(&s, key)
}

// --- CyberChef operation wrappers ---

// asconKeyNonce decodes and validates the key and nonce arguments shared by both
// operations.
func asconKeyNonce(keyArg, nonceArg core.ToggleString) (key, nonce []byte, err error) {
	if key, err = convertToByteArray(keyArg.Value, keyArg.Option); err != nil {
		return nil, nil, err
	}
	if nonce, err = convertToByteArray(nonceArg.Value, nonceArg.Option); err != nil {
		return nil, nil, err
	}
	if len(key) != 16 {
		return nil, nil, fmt.Errorf("Invalid key length: %d bytes.\n\nAscon-AEAD128 requires a key of exactly 16 bytes (128 bits).", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if len(nonce) != 16 {
		return nil, nil, fmt.Errorf("Invalid nonce length: %d bytes.\n\nAscon-AEAD128 requires a nonce of exactly 16 bytes (128 bits).", len(nonce)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return key, nonce, nil
}

// asconOutput renders the result bytes as hex or raw per the Output option.
func asconOutput(data []byte, outputType string) *core.Dish {
	if outputType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(data)), core.TypeString)
	}
	return core.NewDish(data, core.TypeString)
}

// AsconEncrypt performs Ascon-AEAD128 authenticated encryption. Ported from
// CyberChef AsconEncrypt.mjs (which wraps the js-ascon package).
type AsconEncrypt struct{}

// Meta returns the operation metadata.
func (AsconEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Ascon Encrypt",
		Module:      "Ciphers",
		Description: "Ascon-AEAD128 authenticated encryption as standardised in NIST SP 800-232. Ascon is a family of lightweight authenticated encryption algorithms designed for constrained devices such as IoT sensors and embedded systems.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>Nonce:</b> Must be exactly 16 bytes (128 bits). Should be unique for each encryption with the same key. Never reuse a nonce with the same key.<br><br><b>Associated Data:</b> Optional additional data that is authenticated but not encrypted. Useful for including metadata like headers or timestamps.<br><br>The output includes both the ciphertext and a 128-bit authentication tag.",
		InfoURL:     "https://wikipedia.org/wiki/Ascon_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AsconEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Nonce", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Associated Data", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// Run performs the encryption.
func (AsconEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, nonce, err := asconKeyNonce(args[0].(core.ToggleString), args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	adArg := args[2].(core.ToggleString)
	ad, err := convertToByteArray(adArg.Value, adArg.Option)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))
	ciphertext := asconEncryptCore(key, nonce, ad, input)
	return asconOutput(ciphertext, args[4].(string)), nil
}

// AsconDecrypt performs Ascon-AEAD128 authenticated decryption. Ported from
// CyberChef AsconDecrypt.mjs (which wraps the js-ascon package).
type AsconDecrypt struct{}

// Meta returns the operation metadata.
func (AsconDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Ascon Decrypt",
		Module:      "Ciphers",
		Description: "Ascon-AEAD128 authenticated decryption as standardised in NIST SP 800-232. Decrypts ciphertext and verifies the authentication tag. Decryption will fail if the ciphertext or associated data has been tampered with.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>Nonce:</b> Must be exactly 16 bytes (128 bits). Must match the nonce used during encryption.<br><br><b>Associated Data:</b> Must match the associated data used during encryption. Any mismatch will cause authentication failure.",
		InfoURL:     "https://wikipedia.org/wiki/Ascon_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AsconDecrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Nonce", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Associated Data", Type: core.ArgToggleString, Value: "", ToggleValues: asconToggleValues},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// Run performs the decryption, verifying the authentication tag.
func (AsconDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, nonce, err := asconKeyNonce(args[0].(core.ToggleString), args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	adArg := args[2].(core.ToggleString)
	ad, err := convertToByteArray(adArg.Value, adArg.Option)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))

	// Split ciphertext from the trailing 16-byte tag (js-ascon slice semantics:
	// a short input leaves an empty ciphertext and a truncated tag, which never
	// authenticates).
	var ciphertext, tagGiven []byte
	if len(input) >= 16 {
		ciphertext, tagGiven = input[:len(input)-16], input[len(input)-16:]
	} else {
		tagGiven = input
	}

	plaintext, tag := asconDecryptCore(key, nonce, ad, ciphertext)
	if !bytes.Equal(tag, tagGiven) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Unable to decrypt: authentication failed. The ciphertext, key, nonce, or associated data may be incorrect or tampered with.")
	}
	return asconOutput(plaintext, args[4].(string)), nil
}
