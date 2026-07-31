package ops

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(TEAEncrypt{})
	core.Register(TEADecrypt{})
	core.Register(XTEAEncrypt{})
	core.Register(XTEADecrypt{})
}

// teaModes are the block cipher mode choices shared by all four operations.
var teaModes = []string{"CBC", "CFB", "OFB", "CTR", "ECB"}

// teaPaddings are the padding choices for the ECB/CBC modes.
var teaPaddings = []string{"PKCS5", "NO", "ZERO", "RANDOM", "BIT"}

// teaArgs builds the common TEA argument list; XTEA appends a Rounds arg.
func teaArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: teaModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Padding", Type: core.ArgOption, Value: teaPaddings},
	}
}

// teaInputs parses and validates the shared key/IV/input arguments. name is the
// cipher name used in the verbatim error messages ("TEA" or "XTEA").
func teaInputs(in *core.Dish, args []any, name string) (key, iv, input []byte, mode, padding, outType string, err error) {
	ks, ivs := args[0].(core.ToggleString), args[1].(core.ToggleString)
	mode, padding, outType = args[2].(string), args[5].(string), args[4].(string)
	if key, err = convertToByteArray(ks.Value, ks.Option); err != nil {
		return
	}
	if iv, err = convertToByteArray(ivs.Value, ivs.Option); err != nil {
		return
	}
	if len(key) != 16 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid key length: %d bytes\n\n%s requires a key length of 16 bytes (128 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(key), name)
		return
	}
	if len(iv) != teaBlockSize && len(iv) != 0 && mode != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid IV length: %d bytes\n\n%s uses an IV length of %d bytes (%d bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv), name, teaBlockSize, teaBlockSize*8)
		return
	}
	if len(iv) == 0 {
		iv = make([]byte, teaBlockSize)
	}
	input = decodeAESInput(in, args[3].(string))
	return
}

// teaOutput formats the result as concatenated hex or a raw string.
func teaOutput(out []byte, outType string) *core.Dish {
	if outType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(out)), core.TypeString)
	}
	return core.NewDish([]byte(byteArrayToUtf8(out)), core.TypeString)
}

// XTEA round-count limits, declared on the Rounds argument so a value outside
// them is refused during coercion, and re-checked here.
const (
	xteaMinRounds = 1
	xteaMaxRounds = 255
)

// teaValidateRounds reproduces XTEA's rounds check (an integer in range).
func teaValidateRounds(roundsF float64) (int, error) {
	rounds := int(roundsF)
	if float64(rounds) != roundsF || rounds < xteaMinRounds || rounds > xteaMaxRounds {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return 0, fmt.Errorf("Invalid number of rounds: %s\n\nRounds must be an integer between %d and %d. Standard XTEA uses 32 rounds.", strconv.FormatFloat(roundsF, 'g', -1, 64), xteaMinRounds, xteaMaxRounds)
	}
	return rounds, nil
}

// TEAEncrypt encrypts with the TEA block cipher.
type TEAEncrypt struct{}

// Meta returns the operation metadata.
func (TEAEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "TEA Encrypt",
		Module:      "Ciphers",
		Description: "TEA (Tiny Encryption Algorithm) is a block cipher designed by David Wheeler and Roger Needham in 1994. It operates on 64-bit blocks using a 128-bit key and performs 32 cycles (64 Feistel rounds) with the DELTA constant 0x9E3779B9 derived from the golden ratio.<br><br>TEA is notable for its simplicity and compact implementation, making it frequently encountered in malware analysis and CTF challenges. Despite its elegance, TEA has known weaknesses including equivalent keys and susceptibility to related-key attacks, leading to successors XTEA and XXTEA.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>IV:</b> The Initialisation Vector should be 8 bytes (64 bits). If not entered, it will default to null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, the PKCS#5 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/Tiny_Encryption_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TEAEncrypt) Args() []core.ArgDef { return teaArgs() }

// Run encrypts with TEA. Ported from CyberChef TEAEncrypt.mjs.
func (TEAEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, padding, outType, err := teaInputs(in, args, "TEA")
	if err != nil {
		return nil, err
	}
	out, err := teaEncrypt(input, iv, mode, padding, func(b []byte) []byte { return teaEncryptBlock(b, key) })
	if err != nil {
		return nil, err
	}
	return teaOutput(out, outType), nil
}

// TEADecrypt decrypts with the TEA block cipher.
type TEADecrypt struct{}

// Meta returns the operation metadata.
func (TEADecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "TEA Decrypt",
		Module:      "Ciphers",
		Description: "TEA (Tiny Encryption Algorithm) is a block cipher designed by David Wheeler and Roger Needham in 1994. It operates on 64-bit blocks using a 128-bit key and performs 32 cycles (64 Feistel rounds) with the DELTA constant 0x9E3779B9 derived from the golden ratio.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>IV:</b> The Initialisation Vector should be 8 bytes (64 bits). If not entered, it will default to null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, the PKCS#5 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/Tiny_Encryption_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TEADecrypt) Args() []core.ArgDef { return teaArgs() }

// Run decrypts with TEA. Ported from CyberChef TEADecrypt.mjs.
func (TEADecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, padding, outType, err := teaInputs(in, args, "TEA")
	if err != nil {
		return nil, err
	}
	out, err := teaDecrypt(input, iv, mode, padding,
		func(b []byte) []byte { return teaEncryptBlock(b, key) },
		func(b []byte) []byte { return teaDecryptBlock(b, key) })
	if err != nil {
		return nil, err
	}
	return teaOutput(out, outType), nil
}

// xteaArgs is the TEA argument list plus the XTEA Rounds count.
func xteaArgs() []core.ArgDef {
	rMin, rMax := float64(xteaMinRounds), float64(xteaMaxRounds)
	return append(teaArgs(), core.ArgDef{Name: "Rounds", Type: core.ArgNumber, Value: float64(32), Min: &rMin, Max: &rMax})
}

// XTEAEncrypt encrypts with the XTEA block cipher.
type XTEAEncrypt struct{}

// Meta returns the operation metadata.
func (XTEAEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XTEA Encrypt",
		Module:      "Ciphers",
		Description: "XTEA (eXtended Tiny Encryption Algorithm) is a block cipher designed by David Wheeler and Roger Needham in 1997 as a successor to TEA, correcting several weaknesses identified in the original algorithm. It operates on 64-bit blocks using a 128-bit key with an improved key schedule that uses sum-dependent key word selection to resist related-key attacks.<br><br>XTEA retains the simplicity and compact implementation of TEA whilst providing significantly improved security. It is frequently encountered in malware analysis and CTF challenges due to its straightforward implementation.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>IV:</b> The Initialisation Vector should be 8 bytes (64 bits). If not entered, it will default to null bytes.<br><br><b>Rounds:</b> The recommended number of rounds is 32 (default). The reference implementation by Wheeler &amp; Needham accepts a configurable round count.<br><br><b>Padding:</b> In CBC and ECB mode, the PKCS#5 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/XTEA",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XTEAEncrypt) Args() []core.ArgDef { return xteaArgs() }

// Run encrypts with XTEA. Ported from CyberChef XTEAEncrypt.mjs.
func (XTEAEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, padding, outType, err := teaInputs(in, args, "XTEA")
	if err != nil {
		return nil, err
	}
	rounds, err := teaValidateRounds(args[6].(float64))
	if err != nil {
		return nil, err
	}
	out, err := teaEncrypt(input, iv, mode, padding, func(b []byte) []byte { return xteaEncryptBlock(b, key, rounds) })
	if err != nil {
		return nil, err
	}
	return teaOutput(out, outType), nil
}

// XTEADecrypt decrypts with the XTEA block cipher.
type XTEADecrypt struct{}

// Meta returns the operation metadata.
func (XTEADecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XTEA Decrypt",
		Module:      "Ciphers",
		Description: "XTEA (eXtended Tiny Encryption Algorithm) is a block cipher designed by David Wheeler and Roger Needham in 1997 as a successor to TEA. It operates on 64-bit blocks using a 128-bit key with an improved key schedule that uses sum-dependent key word selection.<br><br><b>Key:</b> Must be exactly 16 bytes (128 bits).<br><br><b>IV:</b> The Initialisation Vector should be 8 bytes (64 bits). If not entered, it will default to null bytes.<br><br><b>Rounds:</b> The recommended number of rounds is 32 (default).<br><br><b>Padding:</b> In CBC and ECB mode, the PKCS#5 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/XTEA",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (XTEADecrypt) Args() []core.ArgDef { return xteaArgs() }

// Run decrypts with XTEA. Ported from CyberChef XTEADecrypt.mjs.
func (XTEADecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, padding, outType, err := teaInputs(in, args, "XTEA")
	if err != nil {
		return nil, err
	}
	rounds, err := teaValidateRounds(args[6].(float64))
	if err != nil {
		return nil, err
	}
	out, err := teaDecrypt(input, iv, mode, padding,
		func(b []byte) []byte { return xteaEncryptBlock(b, key, rounds) },
		func(b []byte) []byte { return xteaDecryptBlock(b, key, rounds) })
	if err != nil {
		return nil, err
	}
	return teaOutput(out, outType), nil
}

// TEA / XTEA block ciphers, ported from CyberChef lib/TEA.mjs. Both operate on
// 64-bit blocks with a 128-bit key; the block-cipher modes (ECB/CBC/CFB/OFB/CTR)
// and padding follow the same file.

// teaDelta is the golden-ratio key-schedule constant.
const teaDelta = 0x9E3779B9

// teaBlockSize is the 64-bit block size in bytes.
const teaBlockSize = 8

// teaRounds is the standard number of cycles for TEA.
const teaRounds = 32

// teaBlockFunc enciphers or deciphers a single 8-byte block.
type teaBlockFunc func(block []byte) []byte

// teaEncryptBlock enciphers one 64-bit block with TEA.
func teaEncryptBlock(block, key []byte) []byte {
	v0, v1 := binary.BigEndian.Uint32(block), binary.BigEndian.Uint32(block[4:])
	k0 := binary.BigEndian.Uint32(key)
	k1 := binary.BigEndian.Uint32(key[4:])
	k2 := binary.BigEndian.Uint32(key[8:])
	k3 := binary.BigEndian.Uint32(key[12:])
	var sum uint32
	for range teaRounds {
		sum += teaDelta
		v0 += ((v1 << 4) + k0) ^ (v1 + sum) ^ ((v1 >> 5) + k1)
		v1 += ((v0 << 4) + k2) ^ (v0 + sum) ^ ((v0 >> 5) + k3)
	}
	return teaWords(v0, v1)
}

// teaDecryptBlock deciphers one 64-bit block with TEA.
func teaDecryptBlock(block, key []byte) []byte {
	v0, v1 := binary.BigEndian.Uint32(block), binary.BigEndian.Uint32(block[4:])
	k0 := binary.BigEndian.Uint32(key)
	k1 := binary.BigEndian.Uint32(key[4:])
	k2 := binary.BigEndian.Uint32(key[8:])
	k3 := binary.BigEndian.Uint32(key[12:])
	sum := uint32(teaDelta)
	sum *= teaRounds
	for range teaRounds {
		v1 -= ((v0 << 4) + k2) ^ (v0 + sum) ^ ((v0 >> 5) + k3)
		v0 -= ((v1 << 4) + k0) ^ (v1 + sum) ^ ((v1 >> 5) + k1)
		sum -= teaDelta
	}
	return teaWords(v0, v1)
}

// xteaEncryptBlock enciphers one 64-bit block with XTEA using the given rounds.
func xteaEncryptBlock(block, key []byte, rounds int) []byte {
	v0, v1 := binary.BigEndian.Uint32(block), binary.BigEndian.Uint32(block[4:])
	k := xteaKeyWords(key)
	var sum uint32
	for range rounds {
		v0 += (((v1 << 4) ^ (v1 >> 5)) + v1) ^ (sum + k[sum&3])
		sum += teaDelta
		v1 += (((v0 << 4) ^ (v0 >> 5)) + v0) ^ (sum + k[(sum>>11)&3])
	}
	return teaWords(v0, v1)
}

// xteaDecryptBlock deciphers one 64-bit block with XTEA using the given rounds.
func xteaDecryptBlock(block, key []byte, rounds int) []byte {
	v0, v1 := binary.BigEndian.Uint32(block), binary.BigEndian.Uint32(block[4:])
	k := xteaKeyWords(key)
	sum := uint32(teaDelta) * uint32(rounds) // #nosec G115 -- rounds is validated to [1,255]
	for range rounds {
		v1 -= (((v0 << 4) ^ (v0 >> 5)) + v0) ^ (sum + k[(sum>>11)&3])
		sum -= teaDelta
		v0 -= (((v1 << 4) ^ (v1 >> 5)) + v1) ^ (sum + k[sum&3])
	}
	return teaWords(v0, v1)
}

// xteaKeyWords reads the 16-byte key as 4 big-endian words.
func xteaKeyWords(key []byte) [4]uint32 {
	return [4]uint32{
		binary.BigEndian.Uint32(key),
		binary.BigEndian.Uint32(key[4:]),
		binary.BigEndian.Uint32(key[8:]),
		binary.BigEndian.Uint32(key[12:]),
	}
}

// teaWords serialises two words as an 8-byte big-endian block.
func teaWords(v0, v1 uint32) []byte {
	out := make([]byte, 8)
	binary.BigEndian.PutUint32(out, v0)
	binary.BigEndian.PutUint32(out[4:], v1)
	return out
}

// teaApplyPadding pads message to a block boundary, matching TEA.mjs (which uses
// a longer "NO" error message than the shared blockApplyPadding helper).
func teaApplyPadding(message []byte, padding string) ([]byte, error) {
	if padding == "NO" && len(message)%teaBlockSize != 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("No padding requested but input length (%d bytes) is not a multiple of %d bytes.", len(message), teaBlockSize)
	}
	return blockApplyPadding(message, padding, teaBlockSize)
}

// teaIncrementCounter increments a big-endian byte counter in place.
func teaIncrementCounter(counter []byte) {
	for i := len(counter) - 1; i >= 0; i-- {
		counter[i]++
		if counter[i] != 0 {
			break
		}
	}
}

// teaXorInto xors src into dst for the first len(dst) bytes.
func teaXorInto(dst, src []byte) {
	for i := range dst {
		dst[i] ^= src[i]
	}
}

// teaZeroPad returns a copy of data zero-extended to a whole number of blocks.
func teaZeroPad(data []byte) []byte {
	out := append([]byte{}, data...)
	for len(out)%teaBlockSize != 0 {
		out = append(out, 0)
	}
	return out
}

// teaEncryptModes runs the block/stream mode over already-block-aligned data.
func teaEncryptModes(data, iv []byte, mode string, encBlock teaBlockFunc) []byte {
	out := make([]byte, 0, len(data))
	ivBlock := append([]byte{}, iv...)
	for i := 0; i < len(data); i += teaBlockSize {
		block := data[i : i+teaBlockSize]
		switch mode {
		case "ECB":
			out = append(out, encBlock(block)...)
		case "CBC":
			xored := append([]byte{}, block...)
			teaXorInto(xored, ivBlock)
			ivBlock = encBlock(xored)
			out = append(out, ivBlock...)
		case "CFB":
			ks := encBlock(ivBlock)
			teaXorInto(ks, block)
			ivBlock = ks
			out = append(out, ks...)
		case "OFB":
			ivBlock = encBlock(ivBlock)
			ks := append([]byte{}, ivBlock...)
			teaXorInto(ks, block)
			out = append(out, ks...)
		case "CTR":
			ks := encBlock(ivBlock)
			teaXorInto(ks, block)
			out = append(out, ks...)
			teaIncrementCounter(ivBlock)
		}
	}
	return out
}

// teaDecryptModes inverts the block/stream mode over block-aligned ciphertext.
func teaDecryptModes(data, iv []byte, mode string, encBlock, decBlock teaBlockFunc) []byte {
	out := make([]byte, 0, len(data))
	ivBlock := append([]byte{}, iv...)
	for i := 0; i < len(data); i += teaBlockSize {
		block := data[i : i+teaBlockSize]
		switch mode {
		case "ECB":
			out = append(out, decBlock(block)...)
		case "CBC":
			dec := decBlock(block)
			teaXorInto(dec, ivBlock)
			out = append(out, dec...)
			ivBlock = append([]byte{}, block...)
		case "CFB":
			ks := encBlock(ivBlock)
			teaXorInto(ks, block)
			out = append(out, ks...)
			ivBlock = append([]byte{}, block...)
		case "OFB":
			ivBlock = encBlock(ivBlock)
			ks := append([]byte{}, ivBlock...)
			teaXorInto(ks, block)
			out = append(out, ks...)
		case "CTR":
			ks := encBlock(ivBlock)
			teaXorInto(ks, block)
			out = append(out, ks...)
			teaIncrementCounter(ivBlock)
		}
	}
	return out
}

// teaEncrypt encrypts message with the given block function and mode/padding.
func teaEncrypt(message, iv []byte, mode, padding string, encBlock teaBlockFunc) ([]byte, error) {
	if len(message) == 0 {
		return []byte{}, nil
	}
	if mode == "ECB" || mode == "CBC" {
		data, err := teaApplyPadding(message, padding)
		if err != nil {
			return nil, err
		}
		return teaEncryptModes(data, iv, mode, encBlock), nil
	}
	// Stream modes: process zero-padded blocks, then trim to the input length.
	out := teaEncryptModes(teaZeroPad(message), iv, mode, encBlock)
	return out[:len(message)], nil
}

// teaDecrypt decrypts cipherText with the given block functions and mode/padding.
func teaDecrypt(cipherText, iv []byte, mode, padding string, encBlock, decBlock teaBlockFunc) ([]byte, error) {
	originalLength := len(cipherText)
	if originalLength == 0 {
		return []byte{}, nil
	}
	if mode == "ECB" || mode == "CBC" {
		if originalLength%teaBlockSize != 0 {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Invalid ciphertext length: %d bytes. Must be a multiple of %d.", originalLength, teaBlockSize)
		}
		plain := teaDecryptModes(cipherText, iv, mode, encBlock, decBlock)
		return blockRemovePadding(plain, padding, teaBlockSize)
	}
	// Stream modes: zero-pad to a block boundary, then trim to the input length.
	out := teaDecryptModes(teaZeroPad(cipherText), iv, mode, encBlock, decBlock)
	return out[:originalLength], nil
}
