package ops

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"unicode/utf16"

	"github.com/roberson-io/cchef/core"
)

// utf16LE encodes s as UTF-16 little-endian bytes, matching CyberChef's
// charCodeAt-per-code-unit iteration.
func utf16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	buf := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(buf[i*2:], u)
	}
	return buf
}

func init() {
	core.Register(NTHash{})
}

// NTHash computes an NT (NTLM) hash: MD4 over the UTF-16LE-encoded input. Ported
// from CyberChef NTHash.mjs.
type NTHash struct{}

// Meta returns the operation metadata.
func (NTHash) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "NT Hash",
		Module:      "Crypto",
		Description: "An NT Hash, sometimes referred to as an NTLM hash, is a method of storing passwords on Windows systems. It works by running MD4 on UTF-16LE encoded input. NTLM hashes are considered weak because they can be brute-forced very easily with modern hardware.",
		InfoURL:     "https://wikipedia.org/wiki/NT_LAN_Manager",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (NTHash) Args() []core.ArgDef { return nil }

// Run computes the NT hash.
func (NTHash) Run(in *core.Dish, args []any) (*core.Dish, error) {
	h := newMD4()
	h.Write(utf16LE(in.String()))
	digest := hex.EncodeToString(h.Sum(nil))
	return core.NewDish([]byte(strings.ToUpper(digest)), core.TypeString), nil
}
