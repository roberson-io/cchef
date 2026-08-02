package ops

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(RC2Encrypt{})
	core.Register(RC2Decrypt{})
}

// rc2BlockSize is the RC2 block size in bytes (64 bits).
const rc2BlockSize = 8

// rc2PiTable is the PITABLE from RFC 2268 used during key expansion.
var rc2PiTable = [256]byte{
	0xd9, 0x78, 0xf9, 0xc4, 0x19, 0xdd, 0xb5, 0xed, 0x28, 0xe9, 0xfd, 0x79, 0x4a, 0xa0, 0xd8, 0x9d,
	0xc6, 0x7e, 0x37, 0x83, 0x2b, 0x76, 0x53, 0x8e, 0x62, 0x4c, 0x64, 0x88, 0x44, 0x8b, 0xfb, 0xa2,
	0x17, 0x9a, 0x59, 0xf5, 0x87, 0xb3, 0x4f, 0x13, 0x61, 0x45, 0x6d, 0x8d, 0x09, 0x81, 0x7d, 0x32,
	0xbd, 0x8f, 0x40, 0xeb, 0x86, 0xb7, 0x7b, 0x0b, 0xf0, 0x95, 0x21, 0x22, 0x5c, 0x6b, 0x4e, 0x82,
	0x54, 0xd6, 0x65, 0x93, 0xce, 0x60, 0xb2, 0x1c, 0x73, 0x56, 0xc0, 0x14, 0xa7, 0x8c, 0xf1, 0xdc,
	0x12, 0x75, 0xca, 0x1f, 0x3b, 0xbe, 0xe4, 0xd1, 0x42, 0x3d, 0xd4, 0x30, 0xa3, 0x3c, 0xb6, 0x26,
	0x6f, 0xbf, 0x0e, 0xda, 0x46, 0x69, 0x07, 0x57, 0x27, 0xf2, 0x1d, 0x9b, 0xbc, 0x94, 0x43, 0x03,
	0xf8, 0x11, 0xc7, 0xf6, 0x90, 0xef, 0x3e, 0xe7, 0x06, 0xc3, 0xd5, 0x2f, 0xc8, 0x66, 0x1e, 0xd7,
	0x08, 0xe8, 0xea, 0xde, 0x80, 0x52, 0xee, 0xf7, 0x84, 0xaa, 0x72, 0xac, 0x35, 0x4d, 0x6a, 0x2a,
	0x96, 0x1a, 0xd2, 0x71, 0x5a, 0x15, 0x49, 0x74, 0x4b, 0x9f, 0xd0, 0x5e, 0x04, 0x18, 0xa4, 0xec,
	0xc2, 0xe0, 0x41, 0x6e, 0x0f, 0x51, 0xcb, 0xcc, 0x24, 0x91, 0xaf, 0x50, 0xa1, 0xf4, 0x70, 0x39,
	0x99, 0x7c, 0x3a, 0x85, 0x23, 0xb8, 0xb4, 0x7a, 0xfc, 0x02, 0x36, 0x5b, 0x25, 0x55, 0x97, 0x31,
	0x2d, 0x5d, 0xfa, 0x98, 0xe3, 0x8a, 0x92, 0xae, 0x05, 0xdf, 0x29, 0x10, 0x67, 0x6c, 0xba, 0xc9,
	0xd3, 0x00, 0xe6, 0xcf, 0xe1, 0x9e, 0xa8, 0x2c, 0x63, 0x16, 0x01, 0x3f, 0x58, 0xe2, 0x89, 0xa9,
	0x0d, 0x38, 0x34, 0x1b, 0xab, 0x33, 0xff, 0xb0, 0xbb, 0x48, 0x0c, 0x5f, 0xb9, 0xb1, 0xcd, 0x2e,
	0xc5, 0xf3, 0xdb, 0x47, 0xe5, 0xa5, 0x9c, 0x77, 0x0a, 0xa6, 0x20, 0x68, 0xfe, 0x7f, 0xc1, 0xad,
}

// rc2Shifts are the per-word rotation amounts used in the mixing rounds.
var rc2Shifts = [4]int{1, 2, 3, 5}

// rc2ExpandKey performs RFC 2268 key expansion with 128 effective key bits (the
// value node-forge uses), returning the 64 little-endian key words. An empty key
// yields an all-0xd9 register, matching node-forge's out-of-bounds behaviour.
func rc2ExpandKey(key []byte) [64]uint16 {
	t := len(key)
	size := max(t, 128)
	l := make([]byte, size)
	copy(l, key)
	if t == 0 {
		for i := range l {
			l[i] = 0xd9
		}
	} else {
		for i := t; i < 128; i++ {
			l[i] = rc2PiTable[(int(l[i-1])+int(l[i-t]))&0xff]
		}
	}
	// With 128 effective key bits, T8 = 16 and the top-byte mask is 0xff.
	l[128-16] = rc2PiTable[l[128-16]]
	for i := 127 - 16; i >= 0; i-- {
		l[i] = rc2PiTable[l[i+1]^l[i+16]]
	}
	var k [64]uint16
	for i := range k {
		k[i] = binary.LittleEndian.Uint16(l[2*i:])
	}
	return k
}

// rc2Cipher is a cipher.Block implementation for RC2, so the standard library's
// CBC mode (and the shared ECB helpers) can drive it.
type rc2Cipher struct{ k [64]uint16 }

// newRC2Cipher expands key into the RC2 round-key schedule.
func newRC2Cipher(key []byte) rc2Cipher { return rc2Cipher{rc2ExpandKey(key)} }

// BlockSize returns the RC2 block size.
func (rc2Cipher) BlockSize() int { return rc2BlockSize }

// Encrypt encrypts one block from src into dst.
func (c rc2Cipher) Encrypt(dst, src []byte) {
	var r [4]uint16
	for i := range r {
		r[i] = binary.LittleEndian.Uint16(src[2*i:])
	}
	j := 0
	mix := func() {
		for i := range 4 {
			r[i] += c.k[j] + (r[(i+3)%4] & r[(i+2)%4]) + (^r[(i+3)%4] & r[(i+1)%4])
			r[i] = bits.RotateLeft16(r[i], rc2Shifts[i])
			j++
		}
	}
	mash := func() {
		for i := range 4 {
			r[i] += c.k[r[(i+3)%4]&63]
		}
	}
	for range 5 {
		mix()
	}
	mash()
	for range 6 {
		mix()
	}
	mash()
	for range 5 {
		mix()
	}
	for i := range r {
		binary.LittleEndian.PutUint16(dst[2*i:], r[i])
	}
}

// Decrypt decrypts one block from src into dst.
func (c rc2Cipher) Decrypt(dst, src []byte) {
	var r [4]uint16
	for i := range r {
		r[i] = binary.LittleEndian.Uint16(src[2*i:])
	}
	j := 63
	rmix := func() {
		for i := 3; i >= 0; i-- {
			r[i] = bits.RotateLeft16(r[i], -rc2Shifts[i])
			r[i] -= c.k[j] + (r[(i+3)%4] & r[(i+2)%4]) + (^r[(i+3)%4] & r[(i+1)%4])
			j--
		}
	}
	rmash := func() {
		for i := 3; i >= 0; i-- {
			r[i] -= c.k[r[(i+3)%4]&63]
		}
	}
	for range 5 {
		rmix()
	}
	rmash()
	for range 6 {
		rmix()
	}
	rmash()
	for range 5 {
		rmix()
	}
	for i := range r {
		binary.LittleEndian.PutUint16(dst[2*i:], r[i])
	}
}

// rc2DecodeIV converts the IV and requires it to be empty (ECB) or 8 bytes (CBC).
// CyberChef feeds any length to node-forge, which produces buggy output for other
// lengths; cchef rejects them instead.
func rc2DecodeIV(arg core.ToggleString) ([]byte, error) {
	iv, err := convertToByteArray(arg.Value, arg.Option)
	if err != nil {
		return nil, err
	}
	if len(iv) != 0 && len(iv) != rc2BlockSize {
		return nil, fmt.Errorf("invalid IV length: %d bytes (RC2 CBC mode requires an 8-byte IV)", len(iv))
	}
	return iv, nil
}

// rc2Encrypt pads with PKCS#7 and encrypts under ECB (empty IV) or CBC.
func rc2Encrypt(input, key, iv []byte) []byte {
	c := newRC2Cipher(key)
	padded := pkcs7Pad(input, rc2BlockSize)
	if len(iv) == 0 {
		return ecbEncrypt(c, padded)
	}
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(c, iv).CryptBlocks(out, padded)
	return out
}

// rc2Decrypt decrypts the whole blocks of input and applies node-forge's lenient
// unpadding: only a block-aligned input is unpadded, and then only by trimming
// the final byte's count (never validating the pad bytes, never erroring).
func rc2Decrypt(input, key, iv []byte) []byte {
	nblocks := len(input) / rc2BlockSize
	if nblocks == 0 {
		return []byte{}
	}
	data := input[:nblocks*rc2BlockSize]
	c := newRC2Cipher(key)
	out := make([]byte, len(data))
	if len(iv) == 0 {
		out = ecbDecrypt(c, data)
	} else {
		cipher.NewCBCDecrypter(c, iv).CryptBlocks(out, data)
	}
	if len(input)%rc2BlockSize == 0 {
		if count := int(out[len(out)-1]); count <= len(out) {
			out = out[:len(out)-count]
		}
	}
	return out
}

// rc2EncDescription / rc2DecDescription are the CyberChef op descriptions.
const (
	rc2EncDescription = "RC2 (also known as ARC2) is a symmetric-key block cipher designed by Ron Rivest in 1987. 'RC' stands for 'Rivest Cipher'.<br><br><b>Key:</b> RC2 uses a variable size key.<br><br>You can generate a password-based key using one of the KDF operations.<br><br><b>IV:</b> To run the cipher in CBC mode, the Initialization Vector should be 8 bytes long. If the IV is left blank, the cipher will run in ECB mode.<br><br><b>Padding:</b> In both CBC and ECB mode, PKCS#7 padding will be used."
	rc2DecDescription = "RC2 (also known as ARC2) is a symmetric-key block cipher designed by Ron Rivest in 1987. 'RC' stands for 'Rivest Cipher'.<br><br><b>Key:</b> RC2 uses a variable size key.<br><br><b>IV:</b> To run the cipher in CBC mode, the Initialization Vector should be 8 bytes long. If the IV is left blank, the cipher will run in ECB mode.<br><br><b>Padding:</b> In both CBC and ECB mode, PKCS#7 padding will be used."
)

// RC2Encrypt encrypts input with the RC2 block cipher.
type RC2Encrypt struct{}

// Meta returns the operation metadata.
func (RC2Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC2 Encrypt",
		Module:      "Ciphers",
		Description: rc2EncDescription,
		InfoURL:     "https://wikipedia.org/wiki/RC2",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC2Encrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// Run performs the encryption.
func (RC2Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	iv, err := rc2DecodeIV(args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[2].(string))
	return blowfishOutput(rc2Encrypt(input, key, iv), args[3].(string)), nil
}

// RC2Decrypt decrypts RC2 ciphertext.
type RC2Decrypt struct{}

// Meta returns the operation metadata.
func (RC2Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC2 Decrypt",
		Module:      "Ciphers",
		Description: rc2DecDescription,
		InfoURL:     "https://wikipedia.org/wiki/RC2",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC2Decrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// Run performs the decryption.
func (RC2Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	iv, err := rc2DecodeIV(args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[2].(string))
	return blowfishOutput(rc2Decrypt(input, key, iv), args[3].(string)), nil
}
