package ops

import (
	"crypto/md5" // #nosec G501 -- MD5 is a user-selectable PBKDF2 PRF, matching CyberChef
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- SHA1 is a user-selectable PBKDF2 PRF, matching CyberChef
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(DerivePBKDF2Key{})
}

// pbkdf2HashOptions lists the hashing functions in CyberChef's argument order.
var pbkdf2HashOptions = []string{"SHA1", "SHA256", "SHA384", "SHA512", "MD5"}

// pbkdf2Hashers maps each hashing option to its Go hash constructor.
var pbkdf2Hashers = map[string]func() hash.Hash{
	"SHA1":   sha1.New,
	"SHA256": sha256.New,
	"SHA384": sha512.New384,
	"SHA512": sha512.New,
	"MD5":    md5.New,
}

// pbkdf2KeyBits is the number of bits in a byte of derived key; CyberChef's key
// size argument is in bits, so the byte length is keySizeBits / 8.
const pbkdf2KeyBits = 8

// DerivePBKDF2Key performs the PBKDF2 password-based key derivation function
// (PKCS #5 v2.0 / RFC 2898).
type DerivePBKDF2Key struct{}

// Meta returns the operation metadata.
func (DerivePBKDF2Key) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Derive PBKDF2 key",
		Module:      "Ciphers",
		Description: "PBKDF2 is a password-based key derivation function. It is part of RSA Laboratories' Public-Key Cryptography Standards (PKCS) series, specifically PKCS #5 v2.0, also published as Internet Engineering Task Force's RFC 2898.<br><br>In many applications of cryptography, user security is ultimately dependent on a password, and because a password usually can't be used directly as a cryptographic key, some processing is required.<br><br>A salt provides a large set of keys for any given password, and an iteration count increases the cost of producing keys from a password, thereby also increasing the difficulty of attack.<br><br>If you leave the salt argument empty, a random salt will be generated.",
		InfoURL:     "https://wikipedia.org/wiki/PBKDF2",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DerivePBKDF2Key) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Passphrase", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"UTF8", "Latin1", "Hex", "Base64"}},
		{Name: "Key size", Type: core.ArgNumber, Value: float64(128)},
		{Name: "Iterations", Type: core.ArgNumber, Value: float64(1)},
		{Name: "Hashing function", Type: core.ArgOption, Value: pbkdf2HashOptions},
		{Name: "Salt", Type: core.ArgToggleString, Value: "", ToggleValues: []string{"Hex", "UTF8", "Latin1", "Base64"}},
	}
}

// Run derives the key. Ported from CyberChef DerivePBKDF2Key.mjs (a wrapper over
// forge.pkcs5.pbkdf2). The operation ignores its input — the passphrase is the
// first argument.
//
// forge in the browser uses pure-JS PBKDF2, which truncates the derived-key
// length toward zero: a key size that is not a multiple of 8 floors to
// keySizeBits/8 bytes (Go integer division floors identically), and a key size
// of zero or below yields an empty key.
func (DerivePBKDF2Key) Run(_ *core.Dish, args []any) (*core.Dish, error) {
	pass := args[0].(core.ToggleString)
	keySizeBits := int(args[1].(float64))
	iterations := int(args[2].(float64))
	newHash := pbkdf2Hashers[args[3].(string)]
	saltArg := args[4].(core.ToggleString)

	passphrase, err := convertToByteArray(pass.Value, pass.Option)
	if err != nil {
		return nil, err
	}

	keyLen := keySizeBits / pbkdf2KeyBits
	if keyLen <= 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	salt, err := convertToByteArray(saltArg.Value, saltArg.Option)
	if err != nil {
		return nil, err
	}
	if len(salt) == 0 {
		// forge.random.getBytesSync(keySize): the bits value is used as a byte count.
		salt = make([]byte, keySizeBits)
		if _, err := rand.Read(salt); err != nil {
			return nil, err
		}
	}

	key, err := pbkdf2.Key(newHash, string(passphrase), salt, iterations, keyLen)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(hex.EncodeToString(key)), core.TypeString), nil
}
