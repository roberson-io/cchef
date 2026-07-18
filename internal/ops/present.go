package ops

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(PRESENTEncrypt{})
	core.Register(PRESENTDecrypt{})
}

// presentDescription is shared by both PRESENT operations.
const presentDescription = "PRESENT is an ultra-lightweight block cipher designed for constrained environments such as RFID tags and sensor networks. It operates on 64-bit blocks and supports 80-bit or 128-bit keys with 31 rounds. Standardised in ISO/IEC 29192-2:2019.<br><br>When using CBC mode, the PKCS#7 padding scheme is used."

// presentPaddings lists the padding schemes both operations offer.
var presentPaddings = []string{"PKCS5", "NO", "ZERO", "RANDOM", "BIT"}

// presentRounds is the number of cipher rounds.
const presentRounds = 31

// presentBlockSize is the block size in bytes (64 bits).
const presentBlockSize = 8

// presentSBox is the 4-bit substitution box; presentSBoxInv is its inverse.
var (
	presentSBox    = [16]byte{0xC, 0x5, 0x6, 0xB, 0x9, 0x0, 0xA, 0xD, 0x3, 0xE, 0xF, 0x8, 0x4, 0x7, 0x1, 0x2}
	presentSBoxInv = [16]byte{0x5, 0xE, 0xF, 0x8, 0xC, 0x1, 0x2, 0xD, 0xB, 0x4, 0x6, 0x3, 0x0, 0x7, 0x9, 0xA}
)

// presentPBox is the bit permutation (bit i moves to position presentPBox[i]);
// presentPBoxInv is its inverse, built at init.
var (
	presentPBox = [64]int{
		0, 16, 32, 48, 1, 17, 33, 49, 2, 18, 34, 50, 3, 19, 35, 51,
		4, 20, 36, 52, 5, 21, 37, 53, 6, 22, 38, 54, 7, 23, 39, 55,
		8, 24, 40, 56, 9, 25, 41, 57, 10, 26, 42, 58, 11, 27, 43, 59,
		12, 28, 44, 60, 13, 29, 45, 61, 14, 30, 46, 62, 15, 31, 47, 63,
	}
	presentPBoxInv [64]int
)

func init() {
	for i, p := range presentPBox {
		presentPBoxInv[p] = i
	}
}

// presentSBoxLayer applies the S-box to each of the 16 nibbles of state.
func presentSBoxLayer(state uint64, sbox [16]byte) uint64 {
	var result uint64
	for i := range 16 {
		nibble := (state >> (uint(i) * 4)) & 0xF
		result |= uint64(sbox[nibble]) << (uint(i) * 4)
	}
	return result
}

// presentPLayer applies the bit permutation to state.
func presentPLayer(state uint64, pbox [64]int) uint64 {
	var result uint64
	for i := range 64 {
		if (state>>uint(i))&1 == 1 {
			result |= 1 << uint(pbox[i])
		}
	}
	return result
}

// presentEncryptBlock encrypts one 64-bit block.
func presentEncryptBlock(block uint64, roundKeys []uint64) uint64 {
	state := block
	for i := range presentRounds {
		state ^= roundKeys[i]
		state = presentSBoxLayer(state, presentSBox)
		state = presentPLayer(state, presentPBox)
	}
	return state ^ roundKeys[presentRounds]
}

// presentDecryptBlock decrypts one 64-bit block.
func presentDecryptBlock(block uint64, roundKeys []uint64) uint64 {
	state := block ^ roundKeys[presentRounds]
	for i := presentRounds - 1; i >= 0; i-- {
		state = presentPLayer(state, presentPBoxInv)
		state = presentSBoxLayer(state, presentSBoxInv)
		state ^= roundKeys[i]
	}
	return state
}

// presentMask returns a big.Int bit-mask of n low bits.
func presentMask(n uint) *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), n), big.NewInt(1))
}

// presentRoundKeys80 derives the 32 round keys from an 80-bit key.
func presentRoundKeys80(key []byte) []uint64 {
	reg := new(big.Int).SetBytes(key)
	mask := presentMask(80)
	low76 := presentMask(76)
	rk := make([]uint64, 0, presentRounds+1)
	for i := 1; i <= presentRounds+1; i++ {
		rk = append(rk, new(big.Int).Rsh(reg, 16).Uint64())
		// Rotate left by 61 within 80 bits.
		reg.And(new(big.Int).Or(new(big.Int).Lsh(reg, 61), new(big.Int).Rsh(reg, 19)), mask)
		// S-box the top nibble (bits 79-76).
		nib := presentSBox[new(big.Int).Rsh(reg, 76).Uint64()]
		reg.Or(new(big.Int).And(reg, low76), new(big.Int).Lsh(big.NewInt(int64(nib)), 76))
		// XOR the round counter into bits 19-15.
		reg.Xor(reg, new(big.Int).Lsh(big.NewInt(int64(i)), 15))
	}
	return rk
}

// presentRoundKeys128 derives the 32 round keys from a 128-bit key.
func presentRoundKeys128(key []byte) []uint64 {
	reg := new(big.Int).SetBytes(key)
	mask := presentMask(128)
	low120 := presentMask(120)
	rk := make([]uint64, 0, presentRounds+1)
	for i := 1; i <= presentRounds+1; i++ {
		rk = append(rk, new(big.Int).Rsh(reg, 64).Uint64())
		// Rotate left by 61 within 128 bits.
		reg.And(new(big.Int).Or(new(big.Int).Lsh(reg, 61), new(big.Int).Rsh(reg, 67)), mask)
		// S-box the top byte's two nibbles (bits 127-120).
		top := new(big.Int).Rsh(reg, 120).Uint64() & 0xFF
		sub := uint64(presentSBox[(top>>4)&0xF])<<4 | uint64(presentSBox[top&0xF])
		reg.Or(new(big.Int).And(reg, low120), new(big.Int).Lsh(new(big.Int).SetUint64(sub), 120))
		// XOR the round counter into bits 66-62.
		reg.Xor(reg, new(big.Int).Lsh(big.NewInt(int64(i)), 62))
	}
	return rk
}

// presentCipher is a cipher.Block implementation for PRESENT, so the standard
// library's CBC mode (and the shared ECB helpers) can drive it.
type presentCipher struct{ roundKeys []uint64 }

// newPresentCipher builds the cipher for an already-validated 10- or 16-byte key.
func newPresentCipher(key []byte) presentCipher {
	if len(key) == 10 {
		return presentCipher{presentRoundKeys80(key)}
	}
	return presentCipher{presentRoundKeys128(key)}
}

// BlockSize returns the PRESENT block size.
func (presentCipher) BlockSize() int { return presentBlockSize }

// Encrypt encrypts one block from src into dst.
func (c presentCipher) Encrypt(dst, src []byte) {
	binary.BigEndian.PutUint64(dst, presentEncryptBlock(binary.BigEndian.Uint64(src), c.roundKeys))
}

// Decrypt decrypts one block from src into dst.
func (c presentCipher) Decrypt(dst, src []byte) {
	binary.BigEndian.PutUint64(dst, presentDecryptBlock(binary.BigEndian.Uint64(src), c.roundKeys))
}

// presentApplyPadding pads message to a whole number of blocks for the chosen
// scheme. PKCS5 always adds a block when already aligned.
func presentApplyPadding(message []byte, padding string) ([]byte, error) {
	remainder := len(message) % presentBlockSize
	nPadding := 0
	if remainder != 0 {
		nPadding = presentBlockSize - remainder
	}
	if padding == "PKCS5" && remainder == 0 {
		nPadding = presentBlockSize
	}
	if nPadding == 0 {
		return append([]byte{}, message...), nil
	}
	padded := append([]byte{}, message...)
	switch padding {
	case "PKCS5":
		for range nPadding {
			padded = append(padded, byte(nPadding))
		}
		return padded, nil
	case "ZERO":
		return append(padded, make([]byte, nPadding)...), nil
	case "RANDOM":
		for range nPadding {
			padded = append(padded, byte(randInt(256))) // #nosec G115 -- randInt(256) is always in [0,255]
		}
		return padded, nil
	case "BIT":
		padded = append(padded, 0x80)
		return append(padded, make([]byte, nPadding-1)...), nil
	}
	// The only remaining option is "NO", which cannot pad a partial block.
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return nil, fmt.Errorf("No padding requested but input is not a %d-byte multiple.", presentBlockSize)
}

// presentRemovePadding strips padding after decryption. NO/ZERO/RANDOM leave the
// message unchanged (they cannot be reliably removed).
func presentRemovePadding(message []byte, padding string) ([]byte, error) {
	if len(message) == 0 {
		return message, nil
	}
	switch padding {
	case "PKCS5":
		padByte := int(message[len(message)-1])
		if padByte > 0 && padByte <= presentBlockSize {
			for i := range padByte {
				if message[len(message)-1-i] != byte(padByte) {
					//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
					return nil, errors.New("Invalid PKCS#5 padding.")
				}
			}
			return message[:len(message)-padByte], nil
		}
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid PKCS#5 padding.")
	case "BIT":
		for i := len(message) - 1; i >= 0; i-- {
			if message[i] == 0x80 {
				return message[:i], nil
			} else if message[i] != 0 {
				//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
				return nil, errors.New("Invalid BIT padding.")
			}
		}
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid BIT padding.")
	}
	return message, nil // NO / ZERO / RANDOM
}

// presentEncrypt pads and encrypts message under the given mode.
func presentEncrypt(message, key, iv []byte, mode, padding string) ([]byte, error) {
	if len(message) == 0 {
		return []byte{}, nil
	}
	padded, err := presentApplyPadding(message, padding)
	if err != nil {
		return nil, err
	}
	c := newPresentCipher(key)
	if mode == "ECB" {
		return ecbEncrypt(c, padded), nil
	}
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(c, iv).CryptBlocks(out, padded)
	return out, nil
}

// presentDecrypt decrypts ciphertext under the given mode and strips padding.
func presentDecrypt(ciphertext, key, iv []byte, mode, padding string) ([]byte, error) {
	if len(ciphertext) == 0 {
		return []byte{}, nil
	}
	if len(ciphertext)%presentBlockSize != 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid ciphertext length: %d bytes. Must be a multiple of 8.", len(ciphertext))
	}
	c := newPresentCipher(key)
	var plain []byte
	if mode == "ECB" {
		plain = ecbDecrypt(c, ciphertext)
	} else {
		plain = make([]byte, len(ciphertext))
		cipher.NewCBCDecrypter(c, iv).CryptBlocks(plain, ciphertext)
	}
	return presentRemovePadding(plain, padding)
}

// presentDecodeKey converts and validates a PRESENT key (10 or 16 bytes).
func presentDecodeKey(arg core.ToggleString) ([]byte, error) {
	key, err := convertToByteArray(arg.Value, arg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 10 && len(key) != 16 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid key length: %d bytes\n\nPRESENT uses a key length of 10 bytes (80 bits) or 16 bytes (128 bits).", len(key))
	}
	return key, nil
}

// presentDecodeIV converts and validates the IV (8 bytes unless ECB).
func presentDecodeIV(arg core.ToggleString, mode string) ([]byte, error) {
	iv, err := convertToByteArray(arg.Value, arg.Option)
	if err != nil {
		return nil, err
	}
	if len(iv) != presentBlockSize && mode != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid IV length: %d bytes\n\nPRESENT uses an IV length of 8 bytes (64 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv))
	}
	return iv, nil
}

// PRESENTEncrypt encrypts input with the PRESENT block cipher.
type PRESENTEncrypt struct{}

// Meta returns the operation metadata.
func (PRESENTEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PRESENT Encrypt",
		Module:      "Ciphers",
		Description: presentDescription,
		InfoURL:     "https://wikipedia.org/wiki/PRESENT_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PRESENTEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: []string{"CBC", "ECB"}},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Padding", Type: core.ArgOption, Value: presentPaddings},
	}
}

// Run performs the encryption. Ported from CyberChef PRESENTEncrypt.mjs.
func (PRESENTEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, err := presentDecodeKey(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	mode := args[2].(string)
	iv, err := presentDecodeIV(args[1].(core.ToggleString), mode)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))
	out, err := presentEncrypt(input, key, iv, mode, args[5].(string))
	if err != nil {
		return nil, err
	}
	return blowfishOutput(out, args[4].(string)), nil
}

// PRESENTDecrypt decrypts PRESENT ciphertext.
type PRESENTDecrypt struct{}

// Meta returns the operation metadata.
func (PRESENTDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PRESENT Decrypt",
		Module:      "Ciphers",
		Description: presentDescription,
		InfoURL:     "https://wikipedia.org/wiki/PRESENT_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PRESENTDecrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: []string{"CBC", "ECB"}},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Padding", Type: core.ArgOption, Value: presentPaddings},
	}
}

// Run performs the decryption. Ported from CyberChef PRESENTDecrypt.mjs.
func (PRESENTDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, err := presentDecodeKey(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	mode := args[2].(string)
	iv, err := presentDecodeIV(args[1].(core.ToggleString), mode)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))
	out, err := presentDecrypt(input, key, iv, mode, args[5].(string))
	if err != nil {
		return nil, err
	}
	return blowfishOutput(out, args[4].(string)), nil
}
