package ops

import (
	"crypto/md5"  // #nosec G501 -- MD5 is a user-selectable KDF hash, matching CyberChef
	"crypto/sha1" // #nosec G505 -- SHA1 is a user-selectable KDF hash, matching CyberChef
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"math"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(DeriveEVPKey{})
}

// evpHashOptions lists the hashing functions in CyberChef's argument order.
var evpHashOptions = []string{"SHA1", "SHA256", "SHA384", "SHA512", "MD5"}

// evpHashers maps each hashing option to its Go hash constructor.
var evpHashers = map[string]func() hash.Hash{
	"SHA1":   sha1.New,
	"SHA256": sha256.New,
	"SHA384": sha512.New384,
	"SHA512": sha512.New,
	"MD5":    md5.New,
}

// CryptoJS's EvpKDF measures the key size in 32-bit words.
const (
	evpWordBits  = 32
	evpWordBytes = 4
	bitsPerByte  = 8
)

// DeriveEVPKey performs the OpenSSL EVP_BytesToKey password-based key derivation
// function (as implemented by crypto-js's EvpKDF).
type DeriveEVPKey struct{}

// Meta returns the operation metadata.
func (DeriveEVPKey) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Derive EVP key",
		Module:      "Ciphers",
		Description: "This operation performs a password-based key derivation function (PBKDF) used extensively in OpenSSL. In many applications of cryptography, user security is ultimately dependent on a password, and because a password usually can't be used directly as a cryptographic key, some processing is required.<br><br>A salt provides a large set of keys for any given password, and an iteration count increases the cost of producing keys from a password, thereby also increasing the difficulty of attack.<br><br>If you leave the salt argument empty, a random salt will be generated.",
		InfoURL:     "https://wikipedia.org/wiki/Key_derivation_function",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DeriveEVPKey) Args() []core.ArgDef {
	maxKeySize, maxIterations := float64(kdfMaxKeySize), float64(kdfMaxIterations)
	return []core.ArgDef{
		{Name: "Passphrase", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"UTF8", "Latin1", "Hex", "Base64"}},
		{Name: "Key size", Type: core.ArgNumber, Integer: true, Value: float64(128), Max: &maxKeySize},
		{Name: "Iterations", Type: core.ArgNumber, Integer: true, Value: float64(1), Max: &maxIterations},
		{Name: "Hashing function", Type: core.ArgOption, Value: evpHashOptions},
		{Name: "Salt", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
	}
}

// Run derives the key. Ported from CyberChef DeriveEVPKey.mjs. Note the salt is
// the parsed argument value (empty → no salt); despite the description, this
// operation does not generate a random salt.
func (DeriveEVPKey) Run(in *core.Dish, args []any) (*core.Dish, error) {
	pass := args[0].(core.ToggleString)
	keySizeBits := args[1].(float64)
	iterations := int(args[2].(float64))
	newHash := evpHashers[args[3].(string)]
	saltArg := args[4].(core.ToggleString)

	passphrase, err := convertToByteArray(pass.Value, pass.Option)
	if err != nil {
		return nil, err
	}
	salt, err := convertToByteArray(saltArg.Value, saltArg.Option)
	if err != nil {
		return nil, err
	}

	key := evpKDF(passphrase, salt, keySizeBits, iterations, newHash)
	return core.NewDish([]byte(hex.EncodeToString(key)), core.TypeString), nil
}

// evpKDF implements crypto-js's EvpKDF: it repeatedly hashes
// (previousBlock || passphrase || salt), re-hashing each block `iterations`
// times, concatenating blocks until enough key material is produced. The result
// is truncated to ceil(keySizeBits/8) bytes.
func evpKDF(passphrase, salt []byte, keySizeBits float64, iterations int, newHash func() hash.Hash) []byte {
	keySizeWords := keySizeBits / evpWordBits
	var block, derived []byte
	for float64(len(derived))/evpWordBytes < keySizeWords {
		h := newHash()
		h.Write(block) // empty on the first round
		h.Write(passphrase)
		h.Write(salt)
		block = h.Sum(nil)
		for range iterations - 1 {
			h.Reset()
			h.Write(block)
			block = h.Sum(nil)
		}
		derived = append(derived, block...)
	}
	outBytes := int(math.Ceil(keySizeBits / bitsPerByte))
	return derived[:outBytes]
}
