package ops

import (
	"crypto/hmac"
	"crypto/md5"  // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"crypto/sha1" // #nosec G505 -- crypto/sha1 required by the ported CyberChef operation, not a security control
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(HMAC{})
}

// hmacHashes maps a hashing-function name to its constructor. Limited to the
// functions available in the Go standard library.
var hmacHashes = map[string]func() hash.Hash{
	"MD5":    md5.New,
	"SHA1":   sha1.New,
	"SHA224": sha256.New224,
	"SHA256": sha256.New,
	"SHA384": sha512.New384,
	"SHA512": sha512.New,
}

// HMAC computes a keyed-hash message authentication code.
type HMAC struct{}

// Meta returns the operation metadata.
func (HMAC) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "HMAC",
		Module:      "Crypto",
		Description: "Computes a Hash-based Message Authentication Code (HMAC) over the input using the given key and hashing function.",
		InfoURL:     "https://wikipedia.org/wiki/HMAC",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HMAC) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Hex", "Decimal", "Base64", "UTF8", "Latin1"}},
		{Name: "Hashing function", Type: core.ArgOption, Value: []string{"SHA256", "MD5", "SHA1", "SHA224", "SHA384", "SHA512"}},
	}
}

// Run computes the HMAC. Ported from CyberChef HMAC.mjs.
func (HMAC) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyArg := args[0].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	newHash, ok := hmacHashes[args[1].(string)]
	if !ok {
		// The hashing function is an arg-validated option whose values match
		// hmacHashes exactly, so a miss here is a programming error.
		panic(fmt.Sprintf("HMAC: unhandled hashing function %q", args[1]))
	}
	mac := hmac.New(newHash, key)
	mac.Write(in.Bytes())
	return core.NewDish([]byte(hex.EncodeToString(mac.Sum(nil))), core.TypeString), nil
}
