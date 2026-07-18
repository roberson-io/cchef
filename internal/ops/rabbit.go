package ops

import (
	"fmt"
	"math/bits"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Rabbit{})
}

// rabbitA holds the counter-system constants from RFC 4503.
var rabbitA = [8]uint32{
	0x4d34d34d, 0xd34d34d3, 0x34d34d34, 0x4d34d34d,
	0xd34d34d3, 0x34d34d34, 0x4d34d34d, 0xd34d34d3,
}

// rabbitState is the Rabbit cipher's inner state (RFC 4503).
type rabbitState struct {
	x, c [8]uint32
	b    uint32
	le   bool
}

// counterUpdate advances the counter system, propagating the carry.
func (s *rabbitState) counterUpdate() {
	for j := range 8 {
		temp := uint64(s.c[j]) + uint64(rabbitA[j]) + uint64(s.b)
		s.b = uint32(temp >> 32)
		s.c[j] = uint32(temp) // #nosec G115 -- Rabbit's counter state is 32-bit modular arithmetic
	}
}

// rabbitG squares (u+v) mod 2^32 and XORs the high and low 32-bit halves.
func rabbitG(u, v uint32) uint32 {
	uv := u + v
	p := uint64(uv) * uint64(uv)
	return uint32(p>>32) ^ uint32(p) // #nosec G115 -- extracting the two 32-bit halves of the 64-bit square
}

// nextState applies the next-state function to the eight state variables.
func (s *rabbitState) nextState() {
	var g [8]uint32
	for j := range 8 {
		g[j] = rabbitG(s.x[j], s.c[j])
	}
	s.x[0] = g[0] + bits.RotateLeft32(g[7], 16) + bits.RotateLeft32(g[6], 16)
	s.x[1] = g[1] + bits.RotateLeft32(g[0], 8) + g[7]
	s.x[2] = g[2] + bits.RotateLeft32(g[1], 16) + bits.RotateLeft32(g[0], 16)
	s.x[3] = g[3] + bits.RotateLeft32(g[2], 8) + g[1]
	s.x[4] = g[4] + bits.RotateLeft32(g[3], 16) + bits.RotateLeft32(g[2], 16)
	s.x[5] = g[5] + bits.RotateLeft32(g[4], 8) + g[3]
	s.x[6] = g[6] + bits.RotateLeft32(g[5], 16) + bits.RotateLeft32(g[4], 16)
	s.x[7] = g[7] + bits.RotateLeft32(g[6], 8) + g[5]
}

// newRabbitState runs the key and (optional) IV setup schemes. The key is
// exactly 16 bytes and the IV is 0 or 8 bytes (validated by the caller).
func newRabbitState(key, iv []byte, le bool) *rabbitState {
	s := &rabbitState{le: le}
	var k [8]uint32
	for i := range 8 {
		if le {
			k[i] = uint32(key[1+2*i])<<8 | uint32(key[2*i])
		} else {
			k[i] = uint32(key[14-2*i])<<8 | uint32(key[15-2*i])
		}
	}
	for j := range 8 {
		if j%2 == 0 {
			s.x[j] = k[(j+1)%8]<<16 | k[j]
			s.c[j] = k[(j+4)%8]<<16 | k[(j+5)%8]
		} else {
			s.x[j] = k[(j+5)%8]<<16 | k[(j+4)%8]
			s.c[j] = k[j]<<16 | k[(j+1)%8]
		}
	}
	for range 4 {
		s.counterUpdate()
		s.nextState()
	}
	for j := range 8 {
		s.c[j] ^= s.x[(j+4)%8]
	}
	if len(iv) == 8 {
		s.ivSetup(iv)
	}
	return s
}

// ivSetup mixes an 8-byte IV into the counter state.
func (s *rabbitState) ivSetup(iv []byte) {
	ivVal := func(a, b, c, d int) uint32 {
		if s.le {
			return uint32(iv[a])<<24 | uint32(iv[b])<<16 | uint32(iv[c])<<8 | uint32(iv[d])
		}
		return uint32(iv[7-a])<<24 | uint32(iv[7-b])<<16 | uint32(iv[7-c])<<8 | uint32(iv[7-d])
	}
	s.c[0] ^= ivVal(3, 2, 1, 0)
	s.c[1] ^= ivVal(7, 6, 3, 2)
	s.c[2] ^= ivVal(7, 6, 5, 4)
	s.c[3] ^= ivVal(5, 4, 1, 0)
	s.c[4] ^= ivVal(3, 2, 1, 0)
	s.c[5] ^= ivVal(7, 6, 3, 2)
	s.c[6] ^= ivVal(7, 6, 5, 4)
	s.c[7] ^= ivVal(5, 4, 1, 0)
	for range 4 {
		s.counterUpdate()
		s.nextState()
	}
}

// extract advances the cipher and returns the next 16-byte keystream block.
func (s *rabbitState) extract() [16]byte {
	s.counterUpdate()
	s.nextState()
	var out [16]byte
	pos := 0
	add := func(v uint32) {
		out[pos] = byte(v >> 8) // #nosec G115 -- v is a 16-bit keystream word
		out[pos+1] = byte(v)    // #nosec G115 -- low byte of a 16-bit keystream word
		pos += 2
	}
	add((s.x[6] >> 16) ^ (s.x[1] & 0xffff))
	add((s.x[6] & 0xffff) ^ (s.x[3] >> 16))
	add((s.x[4] >> 16) ^ (s.x[7] & 0xffff))
	add((s.x[4] & 0xffff) ^ (s.x[1] >> 16))
	add((s.x[2] >> 16) ^ (s.x[5] & 0xffff))
	add((s.x[2] & 0xffff) ^ (s.x[7] >> 16))
	add((s.x[0] >> 16) ^ (s.x[3] & 0xffff))
	add((s.x[0] & 0xffff) ^ (s.x[5] >> 16))
	if s.le {
		for i, j := 0, 15; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out
}

// rabbitProcess XORs data with the Rabbit keystream. The final partial block
// uses the low bytes of the keystream in big-endian mode and the (already
// byte-reversed) leading bytes in little-endian mode.
func rabbitProcess(data, key, iv []byte, le bool) []byte {
	s := newRabbitState(key, iv, le)
	result := make([]byte, len(data))
	i := 0
	for ; i <= len(data)-16; i += 16 {
		ks := s.extract()
		for j := range 16 {
			result[i+j] = data[i+j] ^ ks[j]
		}
	}
	if rem := len(data) % 16; rem != 0 {
		offset := len(data) - rem
		ks := s.extract()
		for j := range rem {
			if le {
				result[offset+j] = data[offset+j] ^ ks[j]
			} else {
				result[offset+j] = data[offset+j] ^ ks[16-rem+j]
			}
		}
	}
	return result
}

// Rabbit is the RFC 4503 stream cipher.
type Rabbit struct{}

// Meta returns the operation metadata.
func (Rabbit) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Rabbit",
		Module:      "Ciphers",
		Description: "Rabbit is a high-speed stream cipher introduced in 2003 and defined in RFC 4503.<br><br>The cipher uses a 128-bit key and an optional 64-bit initialization vector (IV).<br><br>big-endian: based on RFC4503 and RFC3447<br>little-endian: compatible with Crypto++",
		InfoURL:     "https://wikipedia.org/wiki/Rabbit_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Rabbit) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Endianness", Type: core.ArgOption, Value: []string{"Big", "Little"}},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// Run performs the encryption/decryption (Rabbit is symmetric).
func (Rabbit) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	ivArg := args[1].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	iv, err := convertToByteArray(ivArg.Value, ivArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid key length: %d bytes (expected: 16)", len(key))
	}
	if len(iv) != 0 && len(iv) != 8 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid IV length: %d bytes (expected: 0 or 8)", len(iv))
	}
	le := args[2].(string) == "Little"
	data := decodeAESInput(in, args[3].(string))
	out := rabbitProcess(data, key, iv, le)
	return blowfishOutput(out, args[4].(string)), nil
}
