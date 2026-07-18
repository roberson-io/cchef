package ops

import (
	"errors"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(XXTEAEncrypt{})
	core.Register(XXTEADecrypt{})
}

// xxteaDelta is the golden-ratio key-schedule constant.
const xxteaDelta = 0x9E3779B9

// xxteaDescription is shared by both operations.
const xxteaDescription = "Corrected Block TEA (often referred to as XXTEA) is a block cipher designed to correct weaknesses in the original Block TEA. XXTEA operates on variable-length blocks that are some arbitrary multiple of 32 bits in size (minimum 64 bits). The number of full cycles depends on the block size, but there are at least six (rising to 32 for small block sizes). The original Block TEA applies the XTEA round function to each word in the block and combines it additively with its leftmost neighbour. Slow diffusion rate of the decryption process was immediately exploited to break the cipher. Corrected Block TEA uses a more involved round function which makes use of both immediate neighbours in processing each word in the block."

// xxteaMX is the data-randomisation round function.
func xxteaMX(sum, y, z uint32, p, e int, k [4]uint32) uint32 {
	return ((z>>5 ^ y<<2) + (y>>3 ^ z<<4)) ^ ((sum ^ y) + (k[(p&3)^e] ^ z))
}

// xxteaToUint32Array packs little-endian bytes into words; includeLength appends
// the byte length as a trailing word (used when encrypting).
func xxteaToUint32Array(bs []byte, includeLength bool) []uint32 {
	n := len(bs) >> 2
	if len(bs)&3 != 0 {
		n++
	}
	var v []uint32
	if includeLength {
		v = make([]uint32, n+1)
		v[n] = uint32(len(bs)) // #nosec G115 -- input length, bounded well below 2^32
	} else {
		v = make([]uint32, n)
	}
	for i, b := range bs {
		v[i>>2] |= uint32(b) << ((i & 3) << 3)
	}
	return v
}

// xxteaToByteArray unpacks words into little-endian bytes. When includeLength is
// set, the trailing word gives the original byte length; an inconsistent length
// returns ok=false (the ciphertext could not be decrypted with this key).
func xxteaToByteArray(v []uint32, includeLength bool) ([]byte, bool) {
	n := len(v) << 2
	if includeLength {
		m := int(v[len(v)-1])
		n -= 4
		if m < n-3 || m > n {
			return nil, false
		}
		n = m
	}
	bytes := make([]byte, n)
	for i := range bytes {
		// #nosec G115 -- deliberate low-byte extraction from a 32-bit word
		bytes[i] = byte(v[i>>2] >> ((i & 3) << 3))
	}
	return bytes, true
}

// xxteaKeyWords returns the 4-word key: the key is truncated or zero-padded to 16
// bytes (CyberChef's fixk pads short keys; only the first four words are used).
func xxteaKeyWords(key []byte) [4]uint32 {
	var kb [16]byte
	copy(kb[:], key)
	var k [4]uint32
	for i := range 4 {
		k[i] = uint32(kb[i*4]) | uint32(kb[i*4+1])<<8 | uint32(kb[i*4+2])<<16 | uint32(kb[i*4+3])<<24
	}
	return k
}

// xxteaEncryptWords performs XXTEA encryption in place on the word block.
func xxteaEncryptWords(v []uint32, k [4]uint32) {
	n := len(v) - 1
	z := v[n]
	var sum uint32
	for q := 6 + 52/len(v); q > 0; q-- {
		sum += xxteaDelta
		e := int(sum >> 2 & 3)
		var y uint32
		for p := range n {
			y = v[p+1]
			v[p] += xxteaMX(sum, y, z, p, e, k)
			z = v[p]
		}
		y = v[0]
		v[n] += xxteaMX(sum, y, z, n, e, k)
		z = v[n]
	}
}

// xxteaDecryptWords performs XXTEA decryption in place on the word block.
func xxteaDecryptWords(v []uint32, k [4]uint32) {
	n := len(v) - 1
	y := v[0]
	q := 6 + 52/len(v)
	// #nosec G115 -- q is a small positive cycle count; the product wraps mod 2^32 as in the source
	for sum := uint32(q) * xxteaDelta; sum != 0; sum -= xxteaDelta {
		e := int(sum >> 2 & 3)
		var z uint32
		for p := n; p > 0; p-- {
			z = v[p-1]
			v[p] -= xxteaMX(sum, y, z, p, e, k)
			y = v[p]
		}
		z = v[n]
		v[0] -= xxteaMX(sum, y, z, 0, e, k)
		y = v[0]
	}
}

// xxteaEncrypt enciphers data with the given key.
func xxteaEncrypt(data, key []byte) []byte {
	if len(data) == 0 {
		return data
	}
	v := xxteaToUint32Array(data, true)
	xxteaEncryptWords(v, xxteaKeyWords(key))
	out, _ := xxteaToByteArray(v, false)
	return out
}

// xxteaDecrypt deciphers data with the given key, returning an error when the
// data is not a valid XXTEA ciphertext for this key.
func xxteaDecrypt(data, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	v := xxteaToUint32Array(data, false)
	xxteaDecryptWords(v, xxteaKeyWords(key))
	out, ok := xxteaToByteArray(v, true)
	if !ok {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Unable to decrypt using this key")
	}
	return out, nil
}

// XXTEAEncrypt encrypts with the XXTEA (Corrected Block TEA) cipher.
type XXTEAEncrypt struct{}

// Meta returns the operation metadata.
func (XXTEAEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XXTEA Encrypt",
		Module:      "Ciphers",
		Description: xxteaDescription,
		InfoURL:     "https://wikipedia.org/wiki/XXTEA",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (XXTEAEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues}}
}

// Run encrypts with XXTEA. Ported from CyberChef XXTEAEncrypt.mjs.
func (XXTEAEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ks := args[0].(core.ToggleString)
	key, err := convertToByteArray(ks.Value, ks.Option)
	if err != nil {
		return nil, err
	}
	return core.NewDish(xxteaEncrypt(in.Bytes(), key), core.TypeArrayBuffer), nil
}

// XXTEADecrypt decrypts with the XXTEA (Corrected Block TEA) cipher.
type XXTEADecrypt struct{}

// Meta returns the operation metadata.
func (XXTEADecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "XXTEA Decrypt",
		Module:      "Ciphers",
		Description: xxteaDescription,
		InfoURL:     "https://wikipedia.org/wiki/XXTEA",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (XXTEADecrypt) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues}}
}

// Run decrypts with XXTEA. Ported from CyberChef XXTEADecrypt.mjs.
func (XXTEADecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ks := args[0].(core.ToggleString)
	key, err := convertToByteArray(ks.Value, ks.Option)
	if err != nil {
		return nil, err
	}
	out, err := xxteaDecrypt(in.Bytes(), key)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}
