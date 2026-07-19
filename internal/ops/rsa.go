package ops

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RSAEncrypt{})
	core.Register(RSADecrypt{})
	core.Register(RSASign{})
	core.Register(RSAVerify{})
	core.Register(GenerateRSAKeyPair{})
}

// rsaSchemes are the encryption schemes; rsaMDs are the message digests in
// node-forge's MD_ALGORITHMS order (SHA-1 is the default).
var (
	rsaSchemes = []string{"RSA-OAEP", "RSAES-PKCS1-V1_5", "RAW"}
	rsaMDs     = []string{"SHA-1", "MD5", "SHA-256", "SHA-384", "SHA-512"}
)

// rsaHash maps a digest name to its crypto.Hash. SHA-1 (forge's default) also
// serves as the fallback. SHA-1 and MD5 are offered because CyberChef offers
// them; the hash implementations are registered by the crypto/* imports in
// ecdsa.go (same package).
func rsaHash(md string) crypto.Hash {
	switch md {
	case "MD5":
		return crypto.MD5
	case "SHA-256":
		return crypto.SHA256
	case "SHA-384":
		return crypto.SHA384
	case "SHA-512":
		return crypto.SHA512
	default: // "SHA-1"
		return crypto.SHA1
	}
}

// rsaDigest hashes msg with the named digest, returning the algorithm and digest
// for PKCS#1 v1.5 signing/verification.
func rsaDigest(md string, msg []byte) (crypto.Hash, []byte) {
	h := rsaHash(md)
	d := h.New()
	d.Write(msg)
	return h, d.Sum(nil)
}

// rsaParsePublicKey parses a PEM RSA public key in either PKCS#1 (RSA PUBLIC KEY)
// or SPKI (PUBLIC KEY) form, mirroring forge.pki.publicKeyFromPem.
func rsaParsePublicKey(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("could not parse the public key") //nolint:staticcheck,revive // CyberChef-style text
	}
	if block.Type == "RSA PUBLIC KEY" {
		return x509.ParsePKCS1PublicKey(block.Bytes)
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("provided key is not an RSA public key") //nolint:staticcheck,revive // CyberChef-style text
	}
	return pub, nil
}

// rsaParsePrivateKey parses a PEM RSA private key, mirroring
// forge.pki.decryptRsaPrivateKey: it accepts PKCS#1 (RSA PRIVATE KEY, optionally
// legacy PEM-encrypted) and PKCS#8 (PRIVATE KEY) encodings. PKCS#8-encrypted keys
// (ENCRYPTED PRIVATE KEY) are not supported by the Go standard library.
func rsaParsePrivateKey(pemStr, password string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("could not parse the private key") //nolint:staticcheck,revive // CyberChef-style text
	}
	der := block.Bytes
	if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck // legacy PEM encryption, matching forge
		d, err := x509.DecryptPEMBlock(block, []byte(password)) //nolint:staticcheck // legacy PEM encryption, matching forge
		if err != nil {
			return nil, err
		}
		der = d
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(der)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return nil, err
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("provided key is not an RSA private key") //nolint:staticcheck,revive // CyberChef-style text
		}
		return rk, nil
	case "ENCRYPTED PRIVATE KEY":
		return nil, errors.New("PKCS#8 encrypted private keys are not supported") //nolint:staticcheck,revive // documented limitation
	default:
		return nil, fmt.Errorf("unsupported key type %q", block.Type)
	}
}

// rsaLeftPad left-pads b with zero bytes to the given size (the modulus width),
// matching forge's fixed-width block output.
func rsaLeftPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

// rsaRawEncrypt performs textbook RSA (m^e mod n) with the input bytes taken as a
// big-endian integer, returning a modulus-width block (forge's RAW scheme).
func rsaRawEncrypt(pub *rsa.PublicKey, msg []byte) []byte {
	m := new(big.Int).SetBytes(msg)
	c := new(big.Int).Exp(m, big.NewInt(int64(pub.E)), pub.N)
	return rsaLeftPad(c.Bytes(), pub.Size())
}

// rsaRawDecrypt performs textbook RSA (c^d mod n), returning a modulus-width
// block. The ciphertext integer must be less than the modulus (forge's check).
func rsaRawDecrypt(priv *rsa.PrivateKey, ct []byte) ([]byte, error) {
	y := new(big.Int).SetBytes(ct)
	if y.Cmp(priv.N) >= 0 {
		return nil, errors.New("Encrypted message is invalid.") //nolint:staticcheck,revive // verbatim forge text
	}
	x := new(big.Int).Exp(y, priv.D, priv.N)
	return rsaLeftPad(x.Bytes(), priv.Size()), nil
}

// RSAEncrypt encrypts a message with a PEM RSA public key.
type RSAEncrypt struct{}

// Meta returns the operation metadata.
func (RSAEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RSA Encrypt",
		Module:      "Ciphers",
		Description: "Encrypt a message with a PEM encoded RSA public key.",
		InfoURL:     "https://wikipedia.org/wiki/RSA_(cryptosystem)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RSAEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		// #nosec G101 -- default is an empty PEM header placeholder, not a credential
		{Name: "RSA Public Key (PEM)", Type: core.ArgString, Value: "-----BEGIN RSA PUBLIC KEY-----"},
		{Name: "Encryption Scheme", Type: core.ArgOption, Value: rsaSchemes},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: rsaMDs},
	}
}

// Run encrypts the input. Ported from CyberChef RSAEncrypt.mjs (node-forge).
func (RSAEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[0].(string)
	if len(strings.Replace(keyPem, "-----BEGIN RSA PUBLIC KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a public key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	pub, err := rsaParsePublicKey(keyPem)
	if err != nil {
		return nil, err
	}
	msg := in.Bytes()
	var out []byte
	switch args[1].(string) {
	case "RAW":
		out = rsaRawEncrypt(pub, msg)
	case "RSAES-PKCS1-V1_5":
		if len(msg) > pub.Size()-rsaPKCS1Overhead {
			return nil, errors.New("Message is too long for PKCS#1 v1.5 padding.") //nolint:staticcheck,revive // verbatim forge text
		}
		out, err = rsa.EncryptPKCS1v15(rand.Reader, pub, msg) //nolint:staticcheck // CyberChef offers the RSAES-PKCS1-V1_5 scheme
	default: // RSA-OAEP
		h := rsaHash(args[2].(string))
		maxLen := pub.Size() - 2*h.Size() - 2
		if len(msg) > maxLen {
			//nolint:staticcheck,revive // verbatim CyberChef text
			return nil, fmt.Errorf("RSAES-OAEP input message length (%d) is longer than the maximum allowed length (%d).", len(msg), maxLen)
		}
		out, err = rsa.EncryptOAEP(h.New(), rand.Reader, pub, msg, nil)
	}
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeString), nil
}

// rsaPKCS1Overhead is the minimum PKCS#1 v1.5 padding overhead in bytes.
const rsaPKCS1Overhead = 11

// RSADecrypt decrypts an RSA message with a PEM private key.
type RSADecrypt struct{}

// Meta returns the operation metadata.
func (RSADecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RSA Decrypt",
		Module:      "Ciphers",
		Description: "Decrypt an RSA encrypted message with a PEM encoded private key.",
		InfoURL:     "https://wikipedia.org/wiki/RSA_(cryptosystem)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RSADecrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		// #nosec G101 -- default is an empty PEM header placeholder, not a credential
		{Name: "RSA Private Key (PEM)", Type: core.ArgString, Value: "-----BEGIN RSA PRIVATE KEY-----"},
		{Name: "Key Password", Type: core.ArgString, Value: ""},
		{Name: "Encryption Scheme", Type: core.ArgOption, Value: rsaSchemes},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: rsaMDs},
	}
}

// Run decrypts the input. Ported from CyberChef RSADecrypt.mjs (node-forge).
func (RSADecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[0].(string)
	if len(strings.Replace(keyPem, "-----BEGIN RSA PRIVATE KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a private key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	priv, err := rsaParsePrivateKey(keyPem, args[1].(string))
	if err != nil {
		return nil, err
	}
	ct := in.Bytes()
	if len(ct) != priv.Size() {
		return nil, errors.New("Encrypted message length is invalid.") //nolint:staticcheck,revive // verbatim forge text
	}
	var out []byte
	switch args[2].(string) {
	case "RAW":
		out, err = rsaRawDecrypt(priv, ct)
	case "RSAES-PKCS1-V1_5":
		out, err = rsa.DecryptPKCS1v15(rand.Reader, priv, ct) //nolint:staticcheck // CyberChef offers the RSAES-PKCS1-V1_5 scheme
	default: // RSA-OAEP
		out, err = rsa.DecryptOAEP(rsaHash(args[3].(string)).New(), rand.Reader, priv, ct, nil)
	}
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeString), nil
}

// RSASign signs a message with a PEM RSA private key.
type RSASign struct{}

// Meta returns the operation metadata.
func (RSASign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RSA Sign",
		Module:      "Ciphers",
		Description: "Sign a plaintext message with a PEM encoded RSA key.",
		InfoURL:     "https://wikipedia.org/wiki/RSA_(cryptosystem)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RSASign) Args() []core.ArgDef {
	return []core.ArgDef{
		// #nosec G101 -- default is an empty PEM header placeholder, not a credential
		{Name: "RSA Private Key (PEM)", Type: core.ArgString, Value: "-----BEGIN RSA PRIVATE KEY-----"},
		{Name: "Key Password", Type: core.ArgString, Value: ""},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: rsaMDs},
	}
}

// Run signs the input. Ported from CyberChef RSASign.mjs (RSASSA-PKCS1-v1_5).
func (RSASign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[0].(string)
	if len(strings.Replace(keyPem, "-----BEGIN RSA PRIVATE KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a private key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	priv, err := rsaParsePrivateKey(keyPem, args[1].(string))
	if err != nil {
		return nil, err
	}
	h, hashed := rsaDigest(args[2].(string), in.Bytes())
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, h, hashed)
	if err != nil {
		return nil, err
	}
	return core.NewDish(sig, core.TypeString), nil
}

// RSAVerify verifies a message against a signature and a PEM RSA public key.
type RSAVerify struct{}

// Meta returns the operation metadata.
func (RSAVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RSA Verify",
		Module:      "Ciphers",
		Description: "Verify a message against a signature and a public PEM encoded RSA key.",
		InfoURL:     "https://wikipedia.org/wiki/RSA_(cryptosystem)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RSAVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		// #nosec G101 -- default is an empty PEM header placeholder, not a credential
		{Name: "RSA Public Key (PEM)", Type: core.ArgString, Value: "-----BEGIN RSA PUBLIC KEY-----"},
		{Name: "Message", Type: core.ArgString, Value: ""},
		{Name: "Message format", Type: core.ArgOption, Value: []string{"Raw", "Hex", "Base64"}},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: rsaMDs},
	}
}

// Run verifies the input signature. Ported from CyberChef RSAVerify.mjs.
func (RSAVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[0].(string)
	if len(strings.Replace(keyPem, "-----BEGIN RSA PUBLIC KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a public key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	pub, err := rsaParsePublicKey(keyPem)
	if err != nil {
		return nil, err
	}
	sig := in.Bytes()
	if len(sig) != pub.Size() {
		//nolint:staticcheck,revive // verbatim CyberChef text
		return nil, fmt.Errorf("Signature length (%d) does not match expected length based on key (%d).", len(sig), pub.Size())
	}
	msg, err := ecdsaDecodeMessage(args[1].(string), args[2].(string))
	if err != nil {
		return nil, err
	}
	h, hashed := rsaDigest(args[3].(string), msg)
	result := "Verification Failure"
	if rsa.VerifyPKCS1v15(pub, h, hashed, sig) == nil {
		result = "Verified OK"
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}

// GenerateRSAKeyPair generates an RSA key pair.
type GenerateRSAKeyPair struct{}

// Meta returns the operation metadata.
func (GenerateRSAKeyPair) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate RSA Key Pair",
		Module:      "Ciphers",
		Description: "Generate an RSA key pair with a given number of bits.",
		InfoURL:     "https://wikipedia.org/wiki/RSA_(cryptosystem)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateRSAKeyPair) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "RSA Key Length", Type: core.ArgOption, Value: []string{"1024", "2048", "4096"}},
		{Name: "Output Format", Type: core.ArgOption, Value: []string{"PEM", "JSON", "DER"}},
	}
}

// rsaKeyBits maps the key-length option to its bit count (default 1024).
func rsaKeyBits(s string) int {
	switch s {
	case "2048":
		return 2048
	case "4096":
		return 4096
	default:
		return 1024
	}
}

// rsaKeyJSON carries the generated key's parameters. This is cchef's own JSON
// shape (hex-encoded integers); it deliberately differs from node-forge's
// internal BigInteger serialization, which is not portable.
type rsaKeyJSON struct {
	N    string `json:"n"`
	E    int    `json:"e"`
	D    string `json:"d"`
	P    string `json:"p"`
	Q    string `json:"q"`
	DP   string `json:"dp"`
	DQ   string `json:"dq"`
	QInv string `json:"qInv"`
}

// Run generates the key pair. Ported from CyberChef GenerateRSAKeyPair.mjs.
func (GenerateRSAKeyPair) Run(_ *core.Dish, args []any) (*core.Dish, error) {
	priv, err := rsa.GenerateKey(rand.Reader, rsaKeyBits(args[0].(string)))
	if err != nil {
		return nil, err
	}
	var result []byte
	switch args[1].(string) {
	case "DER":
		result = x509.MarshalPKCS1PrivateKey(priv)
	case "JSON":
		priv.Precompute()
		hexOf := func(i *big.Int) string { return hex.EncodeToString(i.Bytes()) }
		result, err = json.Marshal(rsaKeyJSON{
			N: hexOf(priv.N), E: priv.E, D: hexOf(priv.D),
			P: hexOf(priv.Primes[0]), Q: hexOf(priv.Primes[1]),
			DP: hexOf(priv.Precomputed.Dp), DQ: hexOf(priv.Precomputed.Dq),
			QInv: hexOf(priv.Precomputed.Qinv),
		})
		if err != nil {
			return nil, err
		}
	default: // PEM
		pubDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			return nil, err
		}
		pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})
		privPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
		result = append(append(pubPem, '\n'), privPem...)
	}
	return core.NewDish(result, core.TypeString), nil
}
