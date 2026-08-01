package ops

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des" // #nosec G502 -- Triple DES is an algorithm this operation offers, not a security choice
	"encoding/hex"
	"fmt"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(CMAC{})
}

// cmacInfo holds the configured block cipher and the CMAC parameters for the
// chosen algorithm.
type cmacInfo struct {
	block     cipher.Block
	blockSize int
	rb        []byte // the reduction polynomial constant (Rb)
}

// cmacBuildInfo validates the key length and builds the cipher for the algorithm.
func cmacBuildInfo(key []byte, algo string) (*cmacInfo, error) {
	if algo == "AES" {
		if len(key) != 16 && len(key) != 24 && len(key) != 32 {
			return nil, fmt.Errorf("The key for AES must be either 16, 24, or 32 bytes (currently %d bytes)", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		block, _ := aes.NewCipher(key)
		rb := make([]byte, aes.BlockSize)
		rb[aes.BlockSize-1] = 0x87
		return &cmacInfo{block, aes.BlockSize, rb}, nil
	}
	// Triple DES: a 16-byte key is expanded to 24 bytes as K1‖K2‖K1.
	if len(key) != 16 && len(key) != 24 {
		return nil, fmt.Errorf("The key for Triple DES must be 16 or 24 bytes (currently %d bytes)", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if len(key) == 16 {
		key = append(append([]byte{}, key...), key[:desBlockSize]...)
	}
	block, _ := des.NewTripleDESCipher(key) // #nosec G405 -- Triple DES is an algorithm this operation offers
	return &cmacInfo{block, desBlockSize, []byte{0, 0, 0, 0, 0, 0, 0, 0x1b}}, nil
}

// cmacLeftShift1 returns a left-shifted by one bit (big-endian across the slice).
func cmacLeftShift1(a []byte) []byte {
	out := make([]byte, len(a))
	var carry byte
	for i := len(a) - 1; i >= 0; i-- {
		out[i] = (a[i] << 1) | carry
		carry = a[i] >> 7
	}
	return out
}

// cmacXor returns a ^ b (over the shorter length is never needed: callers pass
// equal-length slices).
func cmacXor(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// cmacSubkeys derives the K1 and K2 subkeys (RFC 4493 §2.3).
func cmacSubkeys(info *cmacInfo) (k1, k2 []byte) {
	l := make([]byte, info.blockSize)
	info.block.Encrypt(l, l)
	k1 = cmacLeftShift1(l)
	if l[0]&0x80 != 0 {
		k1 = cmacXor(k1, info.rb)
	}
	k2 = cmacLeftShift1(k1)
	if k1[0]&0x80 != 0 {
		k2 = cmacXor(k2, info.rb)
	}
	return k1, k2
}

// cmacLastBlock builds the final message block, applying K1 (complete block) or
// K2 with 0x80 padding (partial/empty block).
func cmacLastBlock(info *cmacInfo, input, k1, k2 []byte, n int) []byte {
	bs := info.blockSize
	if n == 0 {
		last := append([]byte{}, k2...)
		last[0] ^= 0x80
		return last
	}
	inputLast := input[bs*(n-1):]
	if len(inputLast) == bs {
		return cmacXor(inputLast, k1)
	}
	data := make([]byte, bs)
	copy(data, inputLast)
	data[len(inputLast)] = 0x80
	return cmacXor(data, k2)
}

// cmacCompute runs the CBC-MAC over the message and returns the tag.
func cmacCompute(info *cmacInfo, input []byte) []byte {
	bs := info.blockSize
	k1, k2 := cmacSubkeys(info)
	n := (len(input) + bs - 1) / bs // number of blocks (ceil)
	last := cmacLastBlock(info, input, k1, k2, n)

	x := make([]byte, bs)
	for i := 0; i < n-1; i++ {
		info.block.Encrypt(x, cmacXor(x, input[bs*i:bs*i+bs]))
	}
	tag := make([]byte, bs)
	info.block.Encrypt(tag, cmacXor(last, x))
	return tag
}

// CMAC computes a block-cipher-based message authentication code (RFC 4493 /
// NIST SP 800-38B) using AES or Triple DES. Ported from CyberChef CMAC.mjs
// (which wraps node-forge for the cipher); the CMAC construction is the .mjs
// logic, over the standard crypto/aes and crypto/des primitives.
type CMAC struct{}

// Meta returns the operation metadata.
func (CMAC) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CMAC",
		Module:      "Crypto",
		Description: "CMAC is a block-cipher based message authentication code algorithm.<br><br>RFC4493 defines AES-CMAC that uses AES encryption with a 128-bit key.<br>NIST SP 800-38B suggests usages of AES with other key lengths and Triple DES.",
		InfoURL:     "https://wikipedia.org/wiki/CMAC",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CMAC) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
		{Name: "Encryption algorithm", Type: core.ArgOption, Value: []string{"AES", "Triple DES"}},
	}
}

// Run computes the CMAC.
func (CMAC) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	info, err := cmacBuildInfo(key, args[1].(string))
	if err != nil {
		return nil, err
	}
	tag := cmacCompute(info, in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(tag)), core.TypeString), nil
}
