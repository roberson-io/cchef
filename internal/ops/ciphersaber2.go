package ops

import (
	"crypto/rand"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CipherSaber2Encrypt{})
	core.Register(CipherSaber2Decrypt{})
}

// cipherSaberRandIV fills b with random bytes for the CipherSaber2 encrypt IV.
// It is a package variable so tests can pin the IV for byte-exact assertions.
var cipherSaberRandIV = func(b []byte) error {
	_, err := rand.Read(b)
	return err
}

// cipherSaber2Encode is the CipherSaber (RC4) keystream cipher shared by both
// operations. The state is keyed with key||iv, mixed over the given number of
// rounds, then used to XOR the input. Ported from lib/CipherSaber2.mjs.
func cipherSaber2Encode(iv, key []byte, rounds int, input []byte) []byte {
	ivp := make([]byte, 0, len(key)+len(iv))
	ivp = append(ivp, key...)
	ivp = append(ivp, iv...)
	if len(ivp) == 0 {
		// With no key material the key-scheduling modulo is undefined; the only
		// way to reach this is empty key and empty IV, where the input is empty
		// too, so the result is empty.
		return nil
	}

	state := make([]int, 256)
	for i := range 256 {
		state[i] = i
	}

	j := 0
	for range rounds {
		for k := range 256 {
			j = (j + state[k] + int(ivp[k%len(ivp)])) % 256
			state[k], state[j] = state[j], state[k]
		}
	}

	j = 0
	i := 0
	result := make([]byte, 0, len(input))
	for x := range input {
		i = (i + 1) % 256
		j = (j + state[i]) % 256
		state[i], state[j] = state[j], state[i]
		n := (state[i] + state[j]) % 256
		// #nosec G115 -- state is a permutation of 0..255, so byte(state[n]) never overflows
		result = append(result, byte(state[n])^input[x])
	}
	return result
}

const cipherSaber2Description = "CipherSaber is a simple symmetric encryption protocol based on the RC4 stream cipher. It gives reasonably strong protection of message confidentiality, yet it's designed to be simple enough that even novice programmers can memorize the algorithm and implement it from scratch."

// CipherSaber2Encrypt encrypts with the CipherSaber-2 protocol (RC4 keyed with
// the key followed by a random 10-byte IV, mixed over N rounds).
type CipherSaber2Encrypt struct{}

// Meta returns the operation metadata.
func (CipherSaber2Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CipherSaber2 Encrypt",
		Module:      "Crypto",
		Description: cipherSaber2Description,
		InfoURL:     "https://wikipedia.org/wiki/CipherSaber",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (CipherSaber2Encrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
		{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: 20},
	}
}

// Run encrypts the input. Ported from CyberChef CipherSaber2Encrypt.mjs: a
// random 10-byte IV is generated, prepended to the output, and used to key the
// cipher over the input.
func (CipherSaber2Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	rounds := int(args[1].(float64))
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, 10)
	if err := cipherSaberRandIV(iv); err != nil {
		return nil, err
	}

	out := make([]byte, 0, 10+len(in.Bytes()))
	out = append(out, iv...)
	out = append(out, cipherSaber2Encode(iv, key, rounds, in.Bytes())...)
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// CipherSaber2Decrypt reverses CipherSaber2Encrypt: the first 10 input bytes are
// the IV and the remainder is the ciphertext.
type CipherSaber2Decrypt struct{}

// Meta returns the operation metadata.
func (CipherSaber2Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CipherSaber2 Decrypt",
		Module:      "Crypto",
		Description: cipherSaber2Description,
		InfoURL:     "https://wikipedia.org/wiki/CipherSaber",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (CipherSaber2Decrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
		{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: 20},
	}
}

// Run decrypts the input. Ported from CyberChef CipherSaber2Decrypt.mjs: the
// first 10 input bytes are the IV, the remainder is the ciphertext.
func (CipherSaber2Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	rounds := int(args[1].(float64))
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}

	data := in.Bytes()
	n := min(10, len(data))
	iv := data[:n]
	cipher := data[n:]
	out := cipherSaber2Encode(iv, key, rounds, cipher)
	return core.NewDish(out, core.TypeArrayBuffer), nil
}
