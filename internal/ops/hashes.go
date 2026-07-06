package ops

import (
	"crypto/md5"  // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"crypto/sha1" // #nosec G505 -- crypto/sha1 required by the ported CyberChef operation, not a security control
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(hashOp{name: "MD5", infoURL: "https://wikipedia.org/wiki/MD5", new: func() hash.Hash { return md5.New() }})     // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control
	core.Register(hashOp{name: "SHA1", infoURL: "https://wikipedia.org/wiki/SHA-1", new: func() hash.Hash { return sha1.New() }}) // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control
	core.Register(hashOp{name: "SHA224", infoURL: "https://wikipedia.org/wiki/SHA-2", new: func() hash.Hash { return sha256.New224() }})
	core.Register(hashOp{name: "SHA256", infoURL: "https://wikipedia.org/wiki/SHA-2", new: func() hash.Hash { return sha256.New() }})
	core.Register(hashOp{name: "SHA384", infoURL: "https://wikipedia.org/wiki/SHA-2", new: func() hash.Hash { return sha512.New384() }})
	core.Register(hashOp{name: "SHA512", infoURL: "https://wikipedia.org/wiki/SHA-2", new: func() hash.Hash { return sha512.New() }})
}

// hashOp implements a hashing operation backed by a stdlib hash constructor.
// Its output is the lower-case hex digest of the input.
type hashOp struct {
	name    string
	infoURL string
	new     func() hash.Hash
}

// Meta returns the operation metadata.
func (h hashOp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        h.name,
		Module:      "Crypto",
		Description: fmt.Sprintf("Computes the %s hash digest of the input, output as a lower-case hex string.", h.name),
		InfoURL:     h.infoURL,
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (hashOp) Args() []core.ArgDef { return nil }

// Run computes the digest.
func (h hashOp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	digest := h.new()
	digest.Write(in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(digest.Sum(nil))), core.TypeString), nil
}
