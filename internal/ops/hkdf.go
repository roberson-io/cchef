package ops

import (
	"crypto/hmac"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(DeriveHKDFKey{})
}

// hkdfHashOptions lists the hashing functions in CyberChef's argument order.
var hkdfHashOptions = []string{
	"MD2", "MD4", "MD5", "SHA0", "SHA1", "SHA224", "SHA256", "SHA384", "SHA512",
	"SHA512/224", "SHA512/256", "RIPEMD128", "RIPEMD160", "RIPEMD256", "RIPEMD320",
	"HAS160", "Whirlpool", "Whirlpool-0", "Whirlpool-T", "Snefru",
}

// hkdfExtractModes are the extract-mode options.
var hkdfExtractModes = []string{"with salt", "no salt", "skip"}

// hkdfMaxBlocks is the RFC 5869 cap on HKDF-Expand output blocks (T(1)..T(255)).
const hkdfMaxBlocks = 255

// DeriveHKDFKey derives a key with the HMAC-based KDF from RFC 5869.
type DeriveHKDFKey struct{}

// Meta returns the operation metadata.
func (DeriveHKDFKey) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Derive HKDF key",
		Module:      "Crypto",
		Description: "A simple Hashed Message Authenticaton Code (HMAC)-based key derivation function (HKDF), defined in RFC5869.", //nolint:misspell // verbatim CyberChef description
		InfoURL:     "https://wikipedia.org/wiki/HKDF",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DeriveHKDFKey) Args() []core.ArgDef {
	toggle := []string{"Hex", "Decimal", "Base64", "UTF8", "Latin1"}
	return []core.ArgDef{
		{Name: "Salt", Type: core.ArgToggleString, Value: "", ToggleValues: toggle},
		{Name: "Info", Type: core.ArgToggleString, Value: "", ToggleValues: toggle},
		{Name: "Hashing function", Type: core.ArgOption, Value: hkdfHashOptions, DefaultIndex: 6},
		{Name: "Extract mode", Type: core.ArgOption, Value: hkdfExtractModes},
		{Name: "L (number of output octets)", Type: core.ArgNumber, Integer: true, Value: float64(16)},
	}
}

// Run derives the key. Ported from CyberChef DeriveHKDFKey.mjs.
func (DeriveHKDFKey) Run(in *core.Dish, args []any) (*core.Dish, error) {
	saltArg := args[0].(core.ToggleString)
	infoArg := args[1].(core.ToggleString)
	hashName := args[2].(string)
	extractMode := args[3].(string)
	l := int(args[4].(float64))

	newHash, ok := cryptoapiHashers[hashName]
	if !ok {
		return nil, fmt.Errorf("hashing function %q is not supported", hashName)
	}

	argSalt, err := convertToByteArray(saltArg.Value, saltArg.Option)
	if err != nil {
		return nil, err
	}
	info, err := convertToByteArray(infoArg.Value, infoArg.Option)
	if err != nil {
		return nil, err
	}

	hashLen := newHash().Size()
	if l < 0 {
		return nil, fmt.Errorf("L must be non-negative") //nolint:staticcheck,revive // verbatim CyberChef message
	}
	if l > hkdfMaxBlocks*hashLen {
		return nil, fmt.Errorf("L too large (maximum length for %s is %d)", hashName, hkdfMaxBlocks*hashLen) //nolint:staticcheck,revive // verbatim CyberChef message
	}

	key := hkdf(newHash, in.Bytes(), argSalt, info, extractMode, l, hashLen)
	return core.NewDish([]byte(hex.EncodeToString(key)), core.TypeString), nil
}

// hkdf runs HKDF-Extract (unless skipped) followed by HKDF-Expand.
func hkdf(newHash func() hash.Hash, ikm, argSalt, info []byte, extractMode string, l, hashLen int) []byte {
	hmacHash := func(key, data []byte) []byte {
		mac := hmac.New(newHash, key)
		mac.Write(data)
		return mac.Sum(nil)
	}

	// Extract: derive the pseudorandom key (PRK).
	salt := argSalt
	if extractMode != "with salt" {
		salt = make([]byte, hashLen) // HashLen zero bytes
	}
	prk := ikm
	if extractMode != "skip" {
		prk = hmacHash(salt, ikm)
	}

	// Expand: T(i) = HMAC(PRK, T(i-1) || info || i), concatenated to length L.
	var t, result []byte
	for i := 1; i <= hkdfMaxBlocks && len(result) < l; i++ {
		data := append(append(append([]byte{}, t...), info...), byte(i)) // #nosec G115 -- i is in [1,255]
		t = hmacHash(prk, data)
		result = append(result, t...)
	}
	return result[:l]
}
