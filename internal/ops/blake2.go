package ops

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/bits"
	"strconv"

	"golang.org/x/crypto/blake2b"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(BLAKE2b{})
	core.Register(BLAKE2s{})
}

// blake2Args builds the shared argument list for the BLAKE2 operations, given
// the digest sizes (in bits) that flavour offers.
func blake2Args(sizes []string) []core.ArgDef {
	return []core.ArgDef{
		{Name: "Size", Type: core.ArgOption, Value: sizes},
		{Name: "Output Encoding", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"UTF8", "Decimal", "Base64", "Hex", "Latin1"}},
	}
}

// blake2Key decodes the optional key argument, returning nil for no key and an
// error if it exceeds the flavour's maximum key length.
func blake2Key(v any, maxLen int) ([]byte, error) {
	ts := v.(core.ToggleString)
	key, err := convertToByteArray(ts.Value, ts.Option)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, nil
	}
	if len(key) > maxLen {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, fmt.Errorf("Key cannot be greater than %d bytes\nIt is currently %d bytes.", maxLen, len(key))
	}
	return key, nil
}

// blake2Output encodes a digest as hex, Base64 or raw bytes, matching the
// operation's Output Encoding argument.
func blake2Output(digest []byte, format string) string {
	switch format {
	case "Base64":
		return toBase64(digest, stdBase64Alphabet)
	case "Raw":
		return byteArrayToUtf8(digest)
	default: // Hex
		return hex.EncodeToString(digest)
	}
}

// BLAKE2b computes the BLAKE2b hash (64-bit optimised) with an optional key,
// backed by golang.org/x/crypto/blake2b.
type BLAKE2b struct{}

// Meta returns the operation metadata.
func (BLAKE2b) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "BLAKE2b",
		Module:      "Hashing",
		Description: "Performs BLAKE2b hashing on the input. BLAKE2b is a flavour of the BLAKE cryptographic hash function that is optimized for 64-bit platforms and produces digests of any size between 1 and 64 bytes. Supports the use of an optional key.",
		InfoURL:     "https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2b_algorithm",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BLAKE2b) Args() []core.ArgDef { return blake2Args([]string{"512", "384", "256", "160", "128"}) }

// Run computes the BLAKE2b digest.
func (BLAKE2b) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size, _ := strconv.Atoi(args[0].(string))
	key, err := blake2Key(args[2], 64)
	if err != nil {
		return nil, err
	}
	// The size (16..64 bytes) and key are within bounds, so New never errors.
	h, _ := blake2b.New(size/8, key)
	h.Write(in.Bytes())
	return core.NewDish([]byte(blake2Output(h.Sum(nil), args[1].(string))), core.TypeString), nil
}

// BLAKE2s computes the BLAKE2s hash (8- to 32-bit optimised) with an optional
// key. A from-scratch port (RFC 7693): x/crypto/blake2s only offers 256-bit and
// keyed 128-bit digests, so it cannot produce the 160-bit / unkeyed 128-bit
// variants this operation needs.
type BLAKE2s struct{}

// Meta returns the operation metadata.
func (BLAKE2s) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "BLAKE2s",
		Module:      "Hashing",
		Description: "Performs BLAKE2s hashing on the input. BLAKE2s is a flavour of the BLAKE cryptographic hash function that is optimized for 8- to 32-bit platforms and produces digests of any size between 1 and 32 bytes. Supports the use of an optional key.",
		InfoURL:     "https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BLAKE2s) Args() []core.ArgDef { return blake2Args([]string{"256", "160", "128"}) }

// Run computes the BLAKE2s digest.
func (BLAKE2s) Run(in *core.Dish, args []any) (*core.Dish, error) {
	size, _ := strconv.Atoi(args[0].(string))
	key, err := blake2Key(args[2], 32)
	if err != nil {
		return nil, err
	}
	digest := blake2sSum(in.Bytes(), key, size/8)
	return core.NewDish([]byte(blake2Output(digest, args[1].(string))), core.TypeString), nil
}

// --- BLAKE2s core (RFC 7693) ---

// blake2sIV is the BLAKE2s initialisation vector.
var blake2sIV = [8]uint32{
	0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
	0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19,
}

// blake2sSigma is the BLAKE2s message-word permutation schedule (10 rounds).
var blake2sSigma = [10][16]byte{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
	{11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4},
	{7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8},
	{9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13},
	{2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9},
	{12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11},
	{13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10},
	{6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5},
	{10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0},
}

// blake2sG is the BLAKE2s mixing function.
func blake2sG(v *[16]uint32, a, b, c, d int, x, y uint32) {
	v[a] = v[a] + v[b] + x
	v[d] = bits.RotateLeft32(v[d]^v[a], -16)
	v[c] += v[d]
	v[b] = bits.RotateLeft32(v[b]^v[c], -12)
	v[a] = v[a] + v[b] + y
	v[d] = bits.RotateLeft32(v[d]^v[a], -8)
	v[c] += v[d]
	v[b] = bits.RotateLeft32(v[b]^v[c], -7)
}

// blake2sCompress applies the compression function to one 64-byte block, with
// byte counter t and the final-block flag.
func blake2sCompress(h *[8]uint32, block []byte, t uint64, last bool) {
	var m [16]uint32
	for i := range m {
		m[i] = binary.LittleEndian.Uint32(block[i*4:])
	}
	var v [16]uint32
	copy(v[:8], h[:])
	copy(v[8:], blake2sIV[:])
	v[12] ^= uint32(t)       // #nosec G115 -- low 32 bits of the block counter
	v[13] ^= uint32(t >> 32) // #nosec G115 -- high 32 bits of the block counter
	if last {
		v[14] = ^v[14]
	}
	for r := range 10 {
		s := blake2sSigma[r]
		blake2sG(&v, 0, 4, 8, 12, m[s[0]], m[s[1]])
		blake2sG(&v, 1, 5, 9, 13, m[s[2]], m[s[3]])
		blake2sG(&v, 2, 6, 10, 14, m[s[4]], m[s[5]])
		blake2sG(&v, 3, 7, 11, 15, m[s[6]], m[s[7]])
		blake2sG(&v, 0, 5, 10, 15, m[s[8]], m[s[9]])
		blake2sG(&v, 1, 6, 11, 12, m[s[10]], m[s[11]])
		blake2sG(&v, 2, 7, 8, 13, m[s[12]], m[s[13]])
		blake2sG(&v, 3, 4, 9, 14, m[s[14]], m[s[15]])
	}
	for i := range h {
		h[i] ^= v[i] ^ v[i+8]
	}
}

// blake2sSum computes the BLAKE2s digest of msg (outlen bytes) with an optional
// key, following the RFC 7693 reference structure.
func blake2sSum(msg, key []byte, outlen int) []byte {
	h := blake2sIV
	// #nosec G115 -- key length (<=32) and outlen (<=32) are small parameter-block fields
	h[0] ^= 0x01010000 ^ (uint32(len(key)) << 8) ^ uint32(outlen)

	var data []byte
	if len(key) > 0 {
		block := make([]byte, 64)
		copy(block, key)
		data = append(data, block...)
	}
	data = append(data, msg...)

	// The counter t must count the real input bytes, not the final block's zero
	// padding, so an empty (unkeyed) input compresses one zero block with t = 0.
	var t uint64
	off := 0
	for len(data)-off > 64 {
		t += 64
		blake2sCompress(&h, data[off:off+64], t, false)
		off += 64
	}
	final := make([]byte, 64)
	copy(final, data[off:])
	t += uint64(len(data) - off) // #nosec G115 -- len(data)-off is a non-negative block length
	blake2sCompress(&h, final, t, true)

	out := make([]byte, 32)
	for i := range h {
		binary.LittleEndian.PutUint32(out[i*4:], h[i])
	}
	return out[:outlen]
}
