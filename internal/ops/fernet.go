package ops

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(FernetEncrypt{})
	core.Register(FernetDecrypt{})
}

// Fernet token layout: version(1) || timestamp(8, big-endian) || IV(16) ||
// AES-128-CBC ciphertext(PKCS#7) || HMAC-SHA256(32), URL-safe base64 encoded.
const (
	fernetVersion   = 0x80
	fernetKeyLen    = 32 // signing key = key[:16], encryption key = key[16:]
	fernetSignLen   = 16
	fernetTimeLen   = 8
	fernetHeaderLen = 1 + fernetTimeLen + aes.BlockSize // version + time + IV = 25
	fernetHMACLen   = sha256.Size                       // 32
	fernetMinLen    = fernetHeaderLen + fernetHMACLen   // 57 (header + HMAC, no ciphertext)
)

// Verbatim error messages from the fernet npm library CyberChef wraps.
var (
	//nolint:staticcheck,revive // verbatim fernet-library message
	errFernetSecret = errors.New("Secret must be 32 url-safe base64-encoded bytes.")
	//nolint:staticcheck,revive // verbatim fernet-library message
	errFernetVersion = errors.New("Invalid version")
	//nolint:staticcheck,revive // verbatim fernet-library message
	errFernetHMAC = errors.New("Invalid Token: HMAC")
)

// fernetDecodeBase64 decodes a Fernet key or token, matching the npm library's
// lenient handling: trailing padding is stripped and URL-safe characters are
// accepted (as are standard-alphabet ones).
func fernetDecodeBase64(s string) ([]byte, bool) {
	s = strings.TrimRight(s, "=")
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	b, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return b, true
}

// fernetEncodeBase64 encodes a token: standard base64 (padding kept) with the
// two URL-safe character substitutions, matching the npm library's output.
func fernetEncodeBase64(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	s = strings.ReplaceAll(s, "+", "-")
	s = strings.ReplaceAll(s, "/", "_")
	return s
}

// fernetSecret decodes a 32-byte Fernet key into its signing key and an
// AES-128 cipher built from the encryption key.
func fernetSecret(key string) (signKey []byte, block cipher.Block, err error) {
	raw, ok := fernetDecodeBase64(key)
	if !ok || len(raw) != fernetKeyLen {
		return nil, nil, errFernetSecret
	}
	block, err = aes.NewCipher(raw[fernetSignLen:])
	if err != nil { // unreachable: the encryption key is always 16 bytes
		return nil, nil, err
	}
	return raw[:fernetSignLen], block, nil
}

// fernetUnpad reverses PKCS#7 the way crypto-js does — it removes the number of
// bytes named by the final byte without validating them (a lenient quirk of the
// fernet library; identical to strict unpadding for any well-formed token).
func fernetUnpad(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	n := min(int(b[len(b)-1]), len(b))
	return b[:len(b)-n]
}

// fernetEncrypt builds a Fernet token from the given signing key, AES cipher,
// IV, timestamp and plaintext. Split out so the byte layout can be tested with a
// fixed IV/time; the public op supplies a random IV and the current time.
func fernetEncrypt(signKey []byte, block cipher.Block, iv []byte, timeSec uint64, plaintext []byte) string {
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ct := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, padded)

	body := make([]byte, 0, fernetHeaderLen+len(ct))
	body = append(body, fernetVersion)
	body = binary.BigEndian.AppendUint64(body, timeSec)
	body = append(body, iv...)
	body = append(body, ct...)

	mac := hmac.New(sha256.New, signKey)
	mac.Write(body)
	return fernetEncodeBase64(mac.Sum(body))
}

// fernetDecrypt verifies and decrypts a Fernet token. Timestamp/TTL are not
// checked (the op uses ttl=0), matching CyberChef.
func fernetDecrypt(signKey []byte, block cipher.Block, token string) ([]byte, error) {
	data, ok := fernetDecodeBase64(token)
	if !ok || len(data) < 1 || data[0] != fernetVersion {
		return nil, errFernetVersion
	}
	if len(data) < fernetMinLen || (len(data)-fernetMinLen)%aes.BlockSize != 0 {
		return nil, errFernetHMAC
	}
	body, given := data[:len(data)-fernetHMACLen], data[len(data)-fernetHMACLen:]
	mac := hmac.New(sha256.New, signKey)
	mac.Write(body)
	if !hmac.Equal(mac.Sum(nil), given) {
		return nil, errFernetHMAC
	}
	iv := data[1+fernetTimeLen : fernetHeaderLen]
	ct := data[fernetHeaderLen : len(data)-fernetHMACLen]
	if len(ct) == 0 {
		return nil, errFernetHMAC
	}
	pt := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(pt, ct)
	return fernetUnpad(pt), nil
}

// fernetDescription is shared by both operations.
const fernetDescription = "Fernet is a symmetric encryption method which makes sure that the message encrypted cannot be manipulated/read without the key. It uses URL safe encoding for the keys. Fernet uses 128-bit AES in CBC mode and PKCS7 padding, with HMAC using SHA256 for authentication. The IV is created from os.random().<br><br><b>Key:</b> The key must be 32 bytes (256 bits) encoded with Base64."

// FernetEncrypt encrypts input into a Fernet token.
type FernetEncrypt struct{}

// Meta returns the operation metadata.
func (FernetEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Fernet Encrypt",
		Module:      "Default",
		Description: fernetDescription,
		InfoURL:     "https://asecuritysite.com/encryption/fer",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FernetEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgString, Value: ""}}
}

// Run encrypts the input. Ported from CyberChef FernetEncrypt.mjs (a wrapper
// over the fernet npm library). A fresh random IV and the current time are used.
func (FernetEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	signKey, block, err := fernetSecret(args[0].(string))
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	// #nosec G115 -- a current Unix timestamp is always non-negative
	token := fernetEncrypt(signKey, block, iv, uint64(time.Now().Unix()), in.Bytes())
	return core.NewDish([]byte(token), core.TypeString), nil
}

// FernetDecrypt decrypts a Fernet token.
type FernetDecrypt struct{}

// Meta returns the operation metadata.
func (FernetDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Fernet Decrypt",
		Module:      "Default",
		Description: fernetDescription,
		InfoURL:     "https://asecuritysite.com/encryption/fer",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FernetDecrypt) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Key", Type: core.ArgString, Value: ""}}
}

// Run decrypts the token. Ported from CyberChef FernetDecrypt.mjs.
func (FernetDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	signKey, block, err := fernetSecret(args[0].(string))
	if err != nil {
		return nil, err
	}
	plaintext, err := fernetDecrypt(signKey, block, in.String())
	if err != nil {
		return nil, err
	}
	return core.NewDish(plaintext, core.TypeString), nil
}
