package ops

import (
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(SM4Encrypt{})
	core.Register(SM4Decrypt{})
}

// sm4Modes are the block cipher mode choices shared by both operations.
var sm4Modes = []string{"CBC", "CFB", "OFB", "CTR", "ECB", "CBC/NoPadding", "ECB/NoPadding"}

// sm4Args builds the shared argument list for both SM4 operations.
func sm4Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: sm4Modes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// sm4Inputs parses and validates the shared key/IV/input arguments.
func sm4Inputs(in *core.Dish, args []any) (key, iv, input []byte, mode string, noPad bool, err error) {
	ks := args[0].(core.ToggleString)
	ivs := args[1].(core.ToggleString)
	mode = args[2].(string)
	if key, err = convertToByteArray(ks.Value, ks.Option); err != nil {
		return
	}
	if iv, err = convertToByteArray(ivs.Value, ivs.Option); err != nil {
		return
	}
	if len(key) != 16 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid key length: %d bytes\n\nSM4 uses a key length of 16 bytes (128 bits).", len(key))
		return
	}
	if len(iv) != 16 && mode[:3] != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid IV length: %d bytes\n\nSM4 uses an IV length of 16 bytes (128 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv))
		return
	}
	input = decodeAESInput(in, args[3].(string))
	noPad = len(mode) > 3 // "CBC/NoPadding" / "ECB/NoPadding"
	return
}

// sm4Output formats the result as space-delimited hex or a raw string.
func sm4Output(out []byte, outType string) *core.Dish {
	if outType == "Hex" {
		return core.NewDish([]byte(toHexSpace(out)), core.TypeString)
	}
	return core.NewDish([]byte(opsutil.BytesAsText(out)), core.TypeString)
}

// SM4Encrypt encrypts with the SM4 block cipher.
type SM4Encrypt struct{}

// Meta returns the operation metadata.
func (SM4Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM4 Encrypt",
		Module:      "Ciphers",
		Description: "SM4 is a 128-bit block cipher, currently established as a national standard (GB/T 32907-2016) of China. Multiple block cipher modes are supported. When using CBC or ECB mode, the PKCS#7 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/SM4_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SM4Encrypt) Args() []core.ArgDef { return sm4Args() }

// Run encrypts with SM4. Ported from CyberChef SM4Encrypt.mjs.
func (SM4Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, noPad, err := sm4Inputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := sm4Encrypt(input, key, iv, mode[:3], noPad)
	if err != nil {
		return nil, err
	}
	return sm4Output(out, args[4].(string)), nil
}

// SM4Decrypt decrypts with the SM4 block cipher.
type SM4Decrypt struct{}

// Meta returns the operation metadata.
func (SM4Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM4 Decrypt",
		Module:      "Ciphers",
		Description: "SM4 is a 128-bit block cipher, currently established as a national standard (GB/T 32907-2016) of China.",
		InfoURL:     "https://wikipedia.org/wiki/SM4_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SM4Decrypt) Args() []core.ArgDef { return sm4Args() }

// Run decrypts with SM4. Ported from CyberChef SM4Decrypt.mjs.
func (SM4Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, noPad, err := sm4Inputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := sm4Decrypt(input, key, iv, mode[:3], noPad)
	if err != nil {
		return nil, err
	}
	return sm4Output(out, args[4].(string)), nil
}

// SM4 (GB/T 32907-2016) block cipher, ported from CyberChef lib/SM4.mjs. Modes
// follow IETF draft-ribose-cfrg-sm4-09.

const sm4Rounds = 32

// sm4BlockSize is the SM4 block size in bytes.
const sm4BlockSize = 16

// sm4Sbox is the SM4 substitution box (256 8-bit values).
var sm4Sbox = [256]uint32{
	0xd6, 0x90, 0xe9, 0xfe, 0xcc, 0xe1, 0x3d, 0xb7, 0x16, 0xb6, 0x14, 0xc2, 0x28, 0xfb, 0x2c, 0x05,
	0x2b, 0x67, 0x9a, 0x76, 0x2a, 0xbe, 0x04, 0xc3, 0xaa, 0x44, 0x13, 0x26, 0x49, 0x86, 0x06, 0x99,
	0x9c, 0x42, 0x50, 0xf4, 0x91, 0xef, 0x98, 0x7a, 0x33, 0x54, 0x0b, 0x43, 0xed, 0xcf, 0xac, 0x62,
	0xe4, 0xb3, 0x1c, 0xa9, 0xc9, 0x08, 0xe8, 0x95, 0x80, 0xdf, 0x94, 0xfa, 0x75, 0x8f, 0x3f, 0xa6,
	0x47, 0x07, 0xa7, 0xfc, 0xf3, 0x73, 0x17, 0xba, 0x83, 0x59, 0x3c, 0x19, 0xe6, 0x85, 0x4f, 0xa8,
	0x68, 0x6b, 0x81, 0xb2, 0x71, 0x64, 0xda, 0x8b, 0xf8, 0xeb, 0x0f, 0x4b, 0x70, 0x56, 0x9d, 0x35,
	0x1e, 0x24, 0x0e, 0x5e, 0x63, 0x58, 0xd1, 0xa2, 0x25, 0x22, 0x7c, 0x3b, 0x01, 0x21, 0x78, 0x87,
	0xd4, 0x00, 0x46, 0x57, 0x9f, 0xd3, 0x27, 0x52, 0x4c, 0x36, 0x02, 0xe7, 0xa0, 0xc4, 0xc8, 0x9e,
	0xea, 0xbf, 0x8a, 0xd2, 0x40, 0xc7, 0x38, 0xb5, 0xa3, 0xf7, 0xf2, 0xce, 0xf9, 0x61, 0x15, 0xa1,
	0xe0, 0xae, 0x5d, 0xa4, 0x9b, 0x34, 0x1a, 0x55, 0xad, 0x93, 0x32, 0x30, 0xf5, 0x8c, 0xb1, 0xe3,
	0x1d, 0xf6, 0xe2, 0x2e, 0x82, 0x66, 0xca, 0x60, 0xc0, 0x29, 0x23, 0xab, 0x0d, 0x53, 0x4e, 0x6f,
	0xd5, 0xdb, 0x37, 0x45, 0xde, 0xfd, 0x8e, 0x2f, 0x03, 0xff, 0x6a, 0x72, 0x6d, 0x6c, 0x5b, 0x51,
	0x8d, 0x1b, 0xaf, 0x92, 0xbb, 0xdd, 0xbc, 0x7f, 0x11, 0xd9, 0x5c, 0x41, 0x1f, 0x10, 0x5a, 0xd8,
	0x0a, 0xc1, 0x31, 0x88, 0xa5, 0xcd, 0x7b, 0xbd, 0x2d, 0x74, 0xd0, 0x12, 0xb8, 0xe5, 0xb4, 0xb0,
	0x89, 0x69, 0x97, 0x4a, 0x0c, 0x96, 0x77, 0x7e, 0x65, 0xb9, 0xf1, 0x09, 0xc5, 0x6e, 0xc6, 0x84,
	0x18, 0xf0, 0x7d, 0xec, 0x3a, 0xdc, 0x4d, 0x20, 0x79, 0xee, 0x5f, 0x3e, 0xd7, 0xcb, 0x39, 0x48,
}

// sm4CK is the fixed parameter CK used in key expansion.
var sm4CK = [32]uint32{
	0x00070e15, 0x1c232a31, 0x383f464d, 0x545b6269,
	0x70777e85, 0x8c939aa1, 0xa8afb6bd, 0xc4cbd2d9,
	0xe0e7eef5, 0xfc030a11, 0x181f262d, 0x343b4249,
	0x50575e65, 0x6c737a81, 0x888f969d, 0xa4abb2b9,
	0xc0c7ced5, 0xdce3eaf1, 0xf8ff060d, 0x141b2229,
	0x30373e45, 0x4c535a61, 0x686f767d, 0x848b9299,
	0xa0a7aeb5, 0xbcc3cad1, 0xd8dfe6ed, 0xf4fb0209,
	0x10171e25, 0x2c333a41, 0x484f565d, 0x646b7279,
}

// sm4FK is the system parameter FK.
var sm4FK = [4]uint32{0xa3b1bac6, 0x56aa3350, 0x677d9197, 0xb27022dc}

// sm4Substitute replaces each byte of b with its S-box value.
func sm4Substitute(b uint32) uint32 {
	return sm4Sbox[(b>>24)&0xFF]<<24 | sm4Sbox[(b>>16)&0xFF]<<16 |
		sm4Sbox[(b>>8)&0xFF]<<8 | sm4Sbox[b&0xFF]
}

// sm4TransformL is the linear transformation L (used in the round function).
func sm4TransformL(b uint32) uint32 {
	b = sm4Substitute(b)
	return b ^ bits.RotateLeft32(b, 2) ^ bits.RotateLeft32(b, 10) ^ bits.RotateLeft32(b, 18) ^ bits.RotateLeft32(b, 24)
}

// sm4TransformLprime is the linear transformation L' (used in key expansion).
func sm4TransformLprime(b uint32) uint32 {
	b = sm4Substitute(b)
	return b ^ bits.RotateLeft32(b, 13) ^ bits.RotateLeft32(b, 23)
}

// sm4RoundKey expands a 4-word raw key into the 32 round keys.
func sm4RoundKey(rawKey [4]uint32) [32]uint32 {
	var k [36]uint32
	for i := range 4 {
		k[i] = rawKey[i] ^ sm4FK[i]
	}
	var rk [32]uint32
	for i := range 32 {
		k[i+4] = k[i] ^ sm4TransformLprime(k[i+1]^k[i+2]^k[i+3]^sm4CK[i])
		rk[i] = k[i+4]
	}
	return rk
}

// sm4ReverseKey returns the round keys in reverse order (for decryption).
func sm4ReverseKey(rk [32]uint32) [32]uint32 {
	var out [32]uint32
	for i := range 32 {
		out[i] = rk[31-i]
	}
	return out
}

// sm4EncryptBlock runs the 32-round SM4 permutation on one block (4 words).
func sm4EncryptBlock(x [4]uint32, rk [32]uint32) [4]uint32 {
	var X [36]uint32
	copy(X[:4], x[:])
	for i := range sm4Rounds {
		X[i+4] = X[i] ^ sm4TransformL(X[i+1]^X[i+2]^X[i+3]^rk[i])
	}
	return [4]uint32{X[35], X[34], X[33], X[32]}
}

// sm4BytesToInts reads 16 bytes at offset as 4 big-endian words.
func sm4BytesToInts(b []byte, offset int) [4]uint32 {
	var v [4]uint32
	for i := range 4 {
		o := offset + i*4
		v[i] = uint32(b[o])<<24 | uint32(b[o+1])<<16 | uint32(b[o+2])<<8 | uint32(b[o+3])
	}
	return v
}

// sm4IntsToBytes writes words as big-endian bytes.
func sm4IntsToBytes(ints [4]uint32) []byte {
	out := make([]byte, 0, 16)
	for _, v := range ints {
		// #nosec G115 -- deliberate big-endian byte extraction from a 32-bit word
		out = append(out, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
	return out
}

// sm4XorInto xors keystream words into a block in place.
func sm4XorInto(block *[4]uint32, ks [4]uint32) {
	for i := range 4 {
		block[i] ^= ks[i]
	}
}

// sm4ECBBlocks applies the block permutation (with the given round keys) to every
// block; encryption passes forward keys, decryption passes reversed keys.
func sm4ECBBlocks(data []byte, rk [32]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		out = append(out, sm4IntsToBytes(sm4EncryptBlock(sm4BytesToInts(data, i), rk))...)
	}
	return out
}

// sm4CBCEncrypt encrypts in CBC mode.
func sm4CBCEncrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		block := sm4BytesToInts(data, i)
		sm4XorInto(&block, iv)
		iv = sm4EncryptBlock(block, rk)
		out = append(out, sm4IntsToBytes(iv)...)
	}
	return out
}

// sm4CBCDecrypt decrypts in CBC mode (rk must be the reversed round keys).
func sm4CBCDecrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		ctBlock := sm4BytesToInts(data, i)
		block := sm4EncryptBlock(ctBlock, rk)
		sm4XorInto(&block, iv)
		out = append(out, sm4IntsToBytes(block)...)
		iv = ctBlock
	}
	return out
}

// sm4CFBEncrypt encrypts in CFB mode.
func sm4CFBEncrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		iv = sm4EncryptBlock(iv, rk)
		block := sm4BytesToInts(data, i)
		sm4XorInto(&block, iv)
		out = append(out, sm4IntsToBytes(block)...)
		iv = block
	}
	return out
}

// sm4CFBDecrypt decrypts in CFB mode (uses the forward round keys).
func sm4CFBDecrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		iv = sm4EncryptBlock(iv, rk)
		ctBlock := sm4BytesToInts(data, i)
		block := ctBlock
		sm4XorInto(&block, iv)
		out = append(out, sm4IntsToBytes(block)...)
		iv = ctBlock
	}
	return out
}

// sm4OFBCrypt applies OFB mode (identical for encryption and decryption).
func sm4OFBCrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		iv = sm4EncryptBlock(iv, rk)
		block := sm4BytesToInts(data, i)
		sm4XorInto(&block, iv)
		out = append(out, sm4IntsToBytes(block)...)
	}
	return out
}

// sm4CTRCrypt applies CTR mode (identical for encryption and decryption). The
// 32-bit counter is added to the low word of the IV, matching SM4.mjs.
func sm4CTRCrypt(data []byte, rk [32]uint32, iv [4]uint32) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i+sm4BlockSize <= len(data); i += sm4BlockSize {
		ctr := iv
		ctr[3] += uint32(i >> 4) // #nosec G115 -- 32-bit block counter, wraps like the JS source
		ks := sm4EncryptBlock(ctr, rk)
		block := sm4BytesToInts(data, i)
		sm4XorInto(&block, ks)
		out = append(out, sm4IntsToBytes(block)...)
	}
	return out
}

// sm4Encrypt encrypts message with SM4 in the given mode. noPadding suppresses
// PKCS#7 padding for ECB/CBC (requiring a block-aligned input).
func sm4Encrypt(message, key, iv []byte, mode string, noPadding bool) ([]byte, error) {
	msgLen := len(message)
	if msgLen == 0 {
		return []byte{}, nil
	}
	rk := sm4RoundKey(sm4BytesToInts(key, 0))

	padByte := byte(0)
	nPadding := sm4BlockSize - (len(message) & 0xF)
	if mode == "ECB" || mode == "CBC" {
		if noPadding {
			if nPadding != sm4BlockSize {
				//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
				return nil, fmt.Errorf("No padding requested in %s mode but input is not a 16-byte multiple.", mode)
			}
			nPadding = 0
		} else {
			padByte = byte(nPadding) // #nosec G115 -- nPadding is in [1,16]
		}
	}
	msg := append([]byte{}, message...)
	for range nPadding {
		msg = append(msg, padByte)
	}

	var cipherText []byte
	switch mode {
	case "ECB":
		cipherText = sm4ECBBlocks(msg, rk)
	case "CBC":
		cipherText = sm4CBCEncrypt(msg, rk, sm4BytesToInts(iv, 0))
	case "CFB":
		cipherText = sm4CFBEncrypt(msg, rk, sm4BytesToInts(iv, 0))
	case "OFB":
		cipherText = sm4OFBCrypt(msg, rk, sm4BytesToInts(iv, 0))
	case "CTR":
		cipherText = sm4CTRCrypt(msg, rk, sm4BytesToInts(iv, 0))
	}
	if mode != "ECB" && mode != "CBC" {
		return cipherText[:msgLen], nil
	}
	return cipherText, nil
}

// sm4Decrypt decrypts cipherText with SM4 in the given mode. ignorePadding skips
// the ECB/CBC block-alignment and PKCS#7 checks.
func sm4Decrypt(cipherText, key, iv []byte, mode string, ignorePadding bool) ([]byte, error) {
	originalLength := len(cipherText)
	if originalLength == 0 {
		return []byte{}, nil
	}
	rk := sm4RoundKey(sm4BytesToInts(key, 0))

	data := cipherText
	if mode == "ECB" || mode == "CBC" {
		rk = sm4ReverseKey(rk)
		if originalLength&0xF != 0 && !ignorePadding {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("With ECB or CBC modes, the input must be divisible into 16 byte blocks. (%d bytes extra)", originalLength&0xF)
		}
	} else {
		data = append([]byte{}, cipherText...)
		for len(data)&0xF != 0 {
			data = append(data, 0)
		}
	}

	var clearText []byte
	switch mode {
	case "ECB":
		clearText = sm4ECBBlocks(data, rk)
	case "CBC":
		clearText = sm4CBCDecrypt(data, rk, sm4BytesToInts(iv, 0))
	case "CFB":
		clearText = sm4CFBDecrypt(data, rk, sm4BytesToInts(iv, 0))
	case "OFB":
		clearText = sm4OFBCrypt(data, rk, sm4BytesToInts(iv, 0))
	case "CTR":
		clearText = sm4CTRCrypt(data, rk, sm4BytesToInts(iv, 0))
	}

	if mode == "ECB" || mode == "CBC" {
		return sm4Unpad(clearText, ignorePadding)
	}
	return clearText[:originalLength], nil
}

// sm4Unpad validates and removes PKCS#7 padding from ECB/CBC output, matching
// SM4.mjs (a trailing byte > 16 is rejected; a byte of 0 removes nothing).
func sm4Unpad(clearText []byte, ignorePadding bool) ([]byte, error) {
	if ignorePadding {
		return clearText, nil
	}
	padByte := int(clearText[len(clearText)-1])
	if padByte > sm4BlockSize {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid PKCS#7 padding.")
	}
	for i := range padByte {
		if int(clearText[len(clearText)-i-1]) != padByte {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Invalid PKCS#7 padding.")
		}
	}
	return clearText[:len(clearText)-padByte], nil
}
