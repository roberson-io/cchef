package ops

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(AESEncrypt{})
	core.Register(AESDecrypt{})
}

// aesToggleValues are the key/IV/AAD encoding modes shared by the AES operations.
var aesToggleValues = []string{"Hex", "UTF8", "Latin1", "Base64"}

// aesModes are the cipher modes accepted by AES Encrypt and AES Decrypt.
var aesModes = []string{"CBC", "CFB", "OFB", "CTR", "GCM", "ECB", "CBC/NoPadding", "ECB/NoPadding"}

// --- shared helpers ---

// aesKeyLenError returns CyberChef's verbatim invalid-key-length message.
func aesKeyLenError(n int) error {
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return fmt.Errorf(`Invalid key length: %d bytes

The following algorithms will be used based on the size of the key:
  16 bytes = AES-128
  24 bytes = AES-192
  32 bytes = AES-256`, n)
}

// parseAESMode splits a mode string such as "CBC/NoPadding" into its base mode
// ("CBC") and whether the NoPadding variant was selected.
func parseAESMode(m string) (base string, noPadding bool) {
	base = m
	if before, _, ok := strings.Cut(m, "/"); ok {
		base = before
	}
	return base, strings.HasSuffix(m, "NoPadding")
}

// decodeAESInput reads the operation input as raw bytes ("Raw") or by decoding
// the hex text ("Hex"), mirroring Utils.convertToByteString(input, inputType).
func decodeAESInput(in *core.Dish, inputType string) []byte {
	if inputType == "Hex" {
		return splitHexToBytes(in.String())
	}
	return in.Bytes()
}

// aesIV16 normalises an IV to exactly the 16-byte AES block size for the block
// and stream modes: an empty IV defaults to 16 null bytes (as documented), a
// short IV is zero-padded and a long one is truncated.
func aesIV16(iv []byte) []byte {
	out := make([]byte, aes.BlockSize)
	copy(out, iv)
	return out
}

// pkcs7Pad appends PKCS#7 padding to make the data a multiple of blockSize.
func pkcs7Pad(data []byte, blockSize int) []byte {
	n := blockSize - len(data)%blockSize
	pad := make([]byte, n)
	for i := range pad {
		pad[i] = byte(n) // #nosec G115 -- n is in [1, blockSize], always a valid byte
	}
	return append(data, pad...)
}

// pkcs7Unpad removes PKCS#7 padding, reporting whether the padding was valid.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	n := int(data[len(data)-1])
	if n == 0 || n > blockSize {
		return nil, false
	}
	for _, c := range data[len(data)-n:] {
		if int(c) != n {
			return nil, false
		}
	}
	return data[:len(data)-n], true
}

// ecbEncrypt encrypts each block independently (AES-ECB). data must be a
// multiple of the block size.
func ecbEncrypt(block cipher.Block, data []byte) []byte {
	bs := block.BlockSize()
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += bs {
		block.Encrypt(out[i:i+bs], data[i:i+bs])
	}
	return out
}

// ecbDecrypt decrypts each block independently (AES-ECB). data must be a
// multiple of the block size.
func ecbDecrypt(block cipher.Block, data []byte) []byte {
	bs := block.BlockSize()
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += bs {
		block.Decrypt(out[i:i+bs], data[i:i+bs])
	}
	return out
}

// aesCFB applies full-block (128-bit segment) CFB, matching node-forge. The
// decrypt flag selects whether the feedback comes from the ciphertext (decrypt)
// or the freshly produced output (encrypt).
func aesCFB(block cipher.Block, iv, data []byte, decrypt bool) []byte {
	bs := block.BlockSize()
	out := make([]byte, len(data))
	feedback := make([]byte, bs)
	copy(feedback, iv)
	ks := make([]byte, bs)
	for i := 0; i < len(data); i += bs {
		block.Encrypt(ks, feedback)
		n := min(bs, len(data)-i)
		for j := range n {
			out[i+j] = data[i+j] ^ ks[j]
		}
		if decrypt {
			copy(feedback, data[i:i+n])
		} else {
			copy(feedback, out[i:i+n])
		}
	}
	return out
}

// aesOFB applies OFB mode: the key stream is the repeated encryption of the IV,
// independent of the data (so encryption and decryption are identical).
func aesOFB(block cipher.Block, iv, data []byte) []byte {
	bs := block.BlockSize()
	out := make([]byte, len(data))
	o := make([]byte, bs)
	copy(o, iv)
	for i := 0; i < len(data); i += bs {
		block.Encrypt(o, o)
		n := min(bs, len(data)-i)
		for j := range n {
			out[i+j] = data[i+j] ^ o[j]
		}
	}
	return out
}

// --- GCM (implemented directly so any IV length, including empty, is
// supported; Go's crypto/cipher rejects nonce lengths other than a positive
// size). Follows NIST SP 800-38D. ---

// gcmGFMul multiplies two 128-bit blocks in GF(2^128) using the GCM reduction
// polynomial.
func gcmGFMul(x, y [16]byte) [16]byte {
	var z [16]byte
	v := y
	for i := range 128 {
		if (x[i/8]>>(7-uint(i%8)))&1 == 1 {
			for j := range z {
				z[j] ^= v[j]
			}
		}
		lsb := v[15] & 1
		for j := 15; j > 0; j-- {
			v[j] = (v[j] >> 1) | (v[j-1] << 7)
		}
		v[0] >>= 1
		if lsb == 1 {
			v[0] ^= 0xe1
		}
	}
	return z
}

// gcmGHASH computes GHASH_H over data, which must already be block-aligned.
func gcmGHASH(h [16]byte, data []byte) [16]byte {
	var y [16]byte
	for i := 0; i < len(data); i += 16 {
		var blk [16]byte
		copy(blk[:], data[i:min(i+16, len(data))])
		for j := range y {
			y[j] ^= blk[j]
		}
		y = gcmGFMul(y, h)
	}
	return y
}

// gcmPadBlock zero-pads b up to a multiple of 16 bytes.
func gcmPadBlock(b []byte) []byte {
	if len(b)%16 == 0 {
		return b
	}
	return append(b, make([]byte, 16-len(b)%16)...)
}

// gcmInc32 increments the rightmost 32 bits of a counter block modulo 2^32.
func gcmInc32(b [16]byte) [16]byte {
	binary.BigEndian.PutUint32(b[12:16], binary.BigEndian.Uint32(b[12:16])+1)
	return b
}

// gcmBE64 encodes v as a big-endian 64-bit value.
func gcmBE64(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// gcmCTR runs the GCTR key stream from the initial counter block over x.
func gcmCTR(block cipher.Block, icb [16]byte, x []byte) []byte {
	out := make([]byte, len(x))
	cb := icb
	var ks [16]byte
	for i := 0; i < len(x); i += 16 {
		block.Encrypt(ks[:], cb[:])
		n := min(16, len(x)-i)
		for j := range n {
			out[i+j] = x[i+j] ^ ks[j]
		}
		cb = gcmInc32(cb)
	}
	return out
}

// gcmJ0 derives the pre-counter block J0 from the IV.
func gcmJ0(h [16]byte, iv []byte) [16]byte {
	var j0 [16]byte
	if len(iv) == 12 {
		copy(j0[:], iv)
		j0[15] = 1
		return j0
	}
	data := append([]byte{}, gcmPadBlock(append([]byte{}, iv...))...)
	data = append(data, gcmBE64(0)...)
	data = append(data, gcmBE64(uint64(len(iv))*8)...)
	return gcmGHASH(h, data)
}

// gcmTag computes the 16-byte authentication tag for ciphertext and AAD.
func gcmTag(block cipher.Block, h, j0 [16]byte, ciphertext, aad []byte) []byte {
	var s []byte
	s = append(s, gcmPadBlock(append([]byte{}, aad...))...)
	s = append(s, gcmPadBlock(append([]byte{}, ciphertext...))...)
	s = append(s, gcmBE64(uint64(len(aad))*8)...)
	s = append(s, gcmBE64(uint64(len(ciphertext))*8)...)
	sh := gcmGHASH(h, s)
	return gcmCTR(block, j0, sh[:])
}

// aesGCMEncrypt returns the ciphertext and 16-byte tag for the given plaintext.
func aesGCMEncrypt(block cipher.Block, iv, plaintext, aad []byte) (ciphertext, tag []byte) {
	var zero [16]byte
	var h [16]byte
	block.Encrypt(h[:], zero[:])
	j0 := gcmJ0(h, iv)
	ciphertext = gcmCTR(block, gcmInc32(j0), plaintext)
	tag = gcmTag(block, h, j0, ciphertext, aad)
	return ciphertext, tag
}

// aesGCMDecrypt returns the plaintext for the given ciphertext, or ok=false if
// the supplied tag does not authenticate.
func aesGCMDecrypt(block cipher.Block, iv, ciphertext, aad, wantTag []byte) (plaintext []byte, ok bool) {
	var zero [16]byte
	var h [16]byte
	block.Encrypt(h[:], zero[:])
	j0 := gcmJ0(h, iv)
	tag := gcmTag(block, h, j0, ciphertext, aad)
	if subtle.ConstantTimeCompare(tag, wantTag) != 1 {
		return nil, false
	}
	return gcmCTR(block, gcmInc32(j0), ciphertext), true
}

// --- AES Encrypt ---

// AESEncrypt encrypts input with AES in a selectable mode. Ported from
// CyberChef AESEncrypt.mjs.
type AESEncrypt struct{}

// Meta returns the operation metadata.
func (AESEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AES Encrypt",
		Module:      "Ciphers",
		Description: "Advanced Encryption Standard (AES) is a U.S. Federal Information Processing Standard (FIPS). It was selected after a 5-year process where 15 competing designs were evaluated. Key sizes of 16, 24 and 32 bytes select AES-128, AES-192 and AES-256. In CBC and ECB mode, PKCS#7 padding is used.",
		InfoURL:     "https://wikipedia.org/wiki/Advanced_Encryption_Standard",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AESEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: aesModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Additional Authenticated Data", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Include IV in output", Type: core.ArgOption, Value: []string{"Off", "Prepend", "Append"}},
	}
}

// Run performs the encryption.
func (AESEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	ivArg := args[1].(core.ToggleString)
	mode, noPadding := parseAESMode(args[2].(string))
	inputType := args[3].(string)
	outputType := args[4].(string)
	aadArg := args[5].(core.ToggleString)
	includeIV := args[6].(string)

	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	iv, err := convertToByteArray(ivArg.Value, ivArg.Option)
	if err != nil {
		return nil, err
	}
	aad, err := convertToByteArray(aadArg.Value, aadArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, aesKeyLenError(len(key))
	}

	input := decodeAESInput(in, inputType)
	if noPadding && len(input)%aes.BlockSize != 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Input length must be a multiple of 16 bytes for NoPadding modes.")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	var out, tag []byte
	switch mode {
	case "CBC":
		data := input
		if !noPadding {
			data = pkcs7Pad(input, aes.BlockSize)
		}
		out = make([]byte, len(data))
		cipher.NewCBCEncrypter(block, aesIV16(iv)).CryptBlocks(out, data)
	case "ECB":
		data := input
		if !noPadding {
			data = pkcs7Pad(input, aes.BlockSize)
		}
		out = ecbEncrypt(block, data)
	case "CFB":
		out = aesCFB(block, aesIV16(iv), input, false)
	case "OFB":
		out = aesOFB(block, aesIV16(iv), input)
	case "CTR":
		out = make([]byte, len(input))
		cipher.NewCTR(block, aesIV16(iv)).XORKeyStream(out, input)
	case "GCM":
		out, tag = aesGCMEncrypt(block, iv, input, aad)
	}

	switch includeIV {
	case "Prepend":
		out = append(append([]byte{}, iv...), out...)
	case "Append":
		out = append(out, iv...)
	}

	if outputType == "Hex" {
		s := hex.EncodeToString(out)
		if mode == "GCM" {
			s += "\n\nTag: " + hex.EncodeToString(tag)
		}
		return core.NewDish([]byte(s), core.TypeString), nil
	}
	if mode == "GCM" {
		return core.NewDish([]byte(string(out)+"\n\nTag: "+string(tag)), core.TypeString), nil
	}
	return core.NewDish(out, core.TypeString), nil
}

// --- AES Decrypt ---

// AESDecrypt decrypts AES ciphertext in a selectable mode. Ported from
// CyberChef AESDecrypt.mjs.
type AESDecrypt struct{}

// Meta returns the operation metadata.
func (AESDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "AES Decrypt",
		Module:      "Ciphers",
		Description: "Advanced Encryption Standard (AES) is a U.S. Federal Information Processing Standard (FIPS). It was selected after a 5-year process where 15 competing designs were evaluated. Key sizes of 16, 24 and 32 bytes select AES-128, AES-192 and AES-256. The GCM Tag field is ignored unless GCM mode is used.",
		InfoURL:     "https://wikipedia.org/wiki/Advanced_Encryption_Standard",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (AESDecrypt) Args() []core.ArgDef {
	ivLenMin := 0.0
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV Length", Type: core.ArgNumber, Value: 16.0, Min: &ivLenMin},
		{Name: "Mode", Type: core.ArgOption, Value: aesModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "GCM Tag", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Additional Authenticated Data", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV from input", Type: core.ArgOption, Value: []string{"Off", "From start", "From end"}},
	}
}

// Run performs the decryption.
func (AESDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	ivArg := args[1].(core.ToggleString)
	ivLength := int(args[2].(float64))
	mode, noPadding := parseAESMode(args[3].(string))
	inputType := args[4].(string)
	outputType := args[5].(string)
	tagArg := args[6].(core.ToggleString)
	aadArg := args[7].(core.ToggleString)
	ivFromInput := args[8].(string)

	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, aesKeyLenError(len(key))
	}
	gcmTagBytes, err := convertToByteArray(tagArg.Value, tagArg.Option)
	if err != nil {
		return nil, err
	}
	aad, err := convertToByteArray(aadArg.Value, aadArg.Option)
	if err != nil {
		return nil, err
	}

	input := decodeAESInput(in, inputType)

	var iv []byte
	if ivFromInput != "Off" {
		if len(input) <= ivLength {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Input is too short to contain an IV of %d bytes.", ivLength)
		}
		if ivFromInput == "From start" {
			iv = input[:ivLength]
			input = input[ivLength:]
		} else {
			iv = input[len(input)-ivLength:]
			input = input[:len(input)-ivLength]
		}
	} else {
		iv, err = convertToByteArray(ivArg.Value, ivArg.Option)
		if err != nil {
			return nil, err
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	out, ok := aesDecryptBytes(block, mode, noPadding, iv, input, aad, gcmTagBytes)
	if !ok {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Unable to decrypt input with these parameters.")
	}

	if outputType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(out)), core.TypeString), nil
	}
	return core.NewDish(out, core.TypeString), nil
}

// aesDecryptBytes decrypts input for the given mode, returning ok=false when the
// ciphertext cannot be authenticated or unpadded.
func aesDecryptBytes(block cipher.Block, mode string, noPadding bool, iv, input, aad, tag []byte) ([]byte, bool) {
	switch mode {
	case "CBC":
		if len(input)%aes.BlockSize != 0 {
			return nil, false
		}
		out := make([]byte, len(input))
		cipher.NewCBCDecrypter(block, aesIV16(iv)).CryptBlocks(out, input)
		return aesMaybeUnpad(out, noPadding)
	case "ECB":
		if len(input)%aes.BlockSize != 0 {
			return nil, false
		}
		return aesMaybeUnpad(ecbDecrypt(block, input), noPadding)
	case "CFB":
		return aesCFB(block, aesIV16(iv), input, true), true
	case "OFB":
		return aesOFB(block, aesIV16(iv), input), true
	case "CTR":
		out := make([]byte, len(input))
		cipher.NewCTR(block, aesIV16(iv)).XORKeyStream(out, input)
		return out, true
	case "GCM":
		return aesGCMDecrypt(block, iv, input, aad, tag)
	}
	return nil, false
}

// aesMaybeUnpad removes PKCS#7 padding unless the NoPadding variant is in use.
func aesMaybeUnpad(data []byte, noPadding bool) ([]byte, bool) {
	if noPadding {
		return data, true
	}
	return pkcs7Unpad(data, aes.BlockSize)
}
