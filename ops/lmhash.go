package ops

import (
	"crypto/des" // #nosec G502 -- LM Hash is defined in terms of DES; not a security choice
	"encoding/hex"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(LMHash{})
}

const (
	lmMaxLen = 14 // LM passwords are truncated/padded to 14 bytes (two 7-byte halves)
	lmHalf   = 7  // bytes per half before DES key expansion
)

// lmMagic is the fixed plaintext "KGS!@#$%" that each half-key DES-encrypts.
var lmMagic = []byte{0x4b, 0x47, 0x53, 0x21, 0x40, 0x23, 0x24, 0x25}

// lmExpandKey turns a 7-byte (56-bit) key into the 8-byte (64-bit) DES key by
// spreading each 7 bits across a byte. The low parity bit of each byte is left
// zero; DES ignores it, so no odd-parity adjustment is needed.
func lmExpandKey(k []byte) []byte {
	return []byte{
		k[0] & 0xFE,
		(k[0] << 7) | (k[1] >> 1),
		(k[1] << 6) | (k[2] >> 2),
		(k[2] << 5) | (k[3] >> 3),
		(k[3] << 4) | (k[4] >> 4),
		(k[4] << 3) | (k[5] >> 5),
		(k[5] << 2) | (k[6] >> 6),
		k[6] << 1,
	}
}

// lmDESHalf DES-encrypts the magic constant under the key expanded from a 7-byte
// half. The expanded key is always 8 bytes, so des.NewCipher never errors.
func lmDESHalf(half []byte) []byte {
	block, _ := des.NewCipher(lmExpandKey(half)) // #nosec G405 -- LM Hash is defined in terms of DES
	out := make([]byte, desBlockSize)
	block.Encrypt(out, lmMagic)
	return out
}

// LMHash computes an LM (LAN Manager) hash. Ported from CyberChef LMHash.mjs
// (which wraps the ntlm npm package); the algorithm is implemented in-repo
// using the standard crypto/des primitive.
type LMHash struct{}

// Meta returns the operation metadata.
func (LMHash) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LM Hash",
		Module:      "Crypto",
		Description: "An LM Hash, or LAN Manager Hash, is a deprecated way of storing passwords on old Microsoft operating systems. It is particularly weak and can be cracked in seconds on modern hardware using rainbow tables.",
		InfoURL:     "https://wikipedia.org/wiki/LAN_Manager#Password_hashing_algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LMHash) Args() []core.ArgDef { return nil }

// Run computes the LM hash.
func (LMHash) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// First 14 UTF-16 code units, uppercased (JS: substring(0,14).toUpperCase()).
	units := utf16.Encode([]rune(in.String()))
	if len(units) > lmMaxLen {
		units = units[:lmMaxLen]
	}
	upper := utf16.Encode([]rune(strings.ToUpper(string(utf16.Decode(units)))))

	// Null-pad to 14 bytes, taking the low byte of each code unit (ASCII).
	y := make([]byte, lmMaxLen)
	for i := 0; i < len(upper) && i < lmMaxLen; i++ {
		y[i] = byte(upper[i]) // #nosec G115 -- ASCII low byte of the code unit, matching Node's ascii encoding
	}

	digest := append(lmDESHalf(y[:lmHalf]), lmDESHalf(y[lmHalf:])...)
	return core.NewDish([]byte(strings.ToUpper(hex.EncodeToString(digest))), core.TypeString), nil
}
