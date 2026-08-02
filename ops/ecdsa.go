package ops

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5" // #nosec G501 -- MD5 is offered as a user-selectable digest, matching CyberChef
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" // #nosec G505 -- SHA-1 is offered as a user-selectable digest, matching CyberChef
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	encasn1 "encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ECDSASign{})
	core.Register(ECDSAVerify{})
	core.Register(ECDSASignatureConversion{})
	core.Register(GenerateECDSAKeyPair{})
}

// ecdsaMDs are the user-selectable message digest algorithms.
var ecdsaMDs = []string{"SHA-256", "SHA-384", "SHA-512", "SHA-1", "MD5"}

// ecdsaSigFormats are the signature output formats; ecdsaInputFormats prepends
// "Auto" for the input side.
var (
	ecdsaSigFormats   = []string{"ASN.1 HEX", "P1363 HEX", "JSON Web Signature", "Raw JSON"}
	ecdsaInputFormats = append([]string{"Auto"}, ecdsaSigFormats...)
)

// ecdsaHexRe matches an all-hexadecimal string (CyberChef's /^[a-f\d]{2,}$/gi).
var ecdsaHexRe = regexp.MustCompile(`(?i)^[a-f0-9]{2,}$`)

// --- Signature format conversion (ported from jsrsasign's KJUR.crypto.ECDSA) ---

// ecdsaDerLen reads a DER length at b[i], returning the content length and the
// number of header bytes the length field occupies.
func ecdsaDerLen(b []byte, i int) (length, header int, ok bool) {
	if i >= len(b) {
		return 0, 0, false
	}
	first := b[i]
	if first < 0x80 {
		return int(first), 1, true
	}
	n := int(first & 0x7f)
	if n == 0 || n > 4 || i+1+n > len(b) {
		return 0, 0, false
	}
	for k := range n {
		length = length<<8 | int(b[i+1+k])
	}
	return length, 1 + n, true
}

// ecdsaParseSigRS parses an ASN.1 SEQUENCE{INTEGER r, INTEGER s} and returns the
// hex of each integer's content octets (leading zero bytes preserved), matching
// jsrsasign's parseSigHexInHexRS.
func ecdsaParseSigRS(asn1Hex string) (rHex, sHex string, err error) {
	b, err := hex.DecodeString(asn1Hex)
	if err != nil || len(b) < 2 || b[0] != 0x30 {
		return "", "", errors.New("signature is not a ASN.1 sequence") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	seqLen, seqHdr, ok := ecdsaDerLen(b, 1)
	if !ok || 1+seqHdr+seqLen != len(b) {
		return "", "", errors.New("signature is not a ASN.1 sequence") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	p := 1 + seqHdr
	readInt := func() (string, bool) {
		if p >= len(b) || b[p] != 0x02 {
			return "", false
		}
		n, hdr, ok := ecdsaDerLen(b, p+1)
		if !ok || p+1+hdr+n > len(b) {
			return "", false
		}
		v := b[p+1+hdr : p+1+hdr+n]
		p += 1 + hdr + n
		return hex.EncodeToString(v), true
	}
	if rHex, ok = readInt(); !ok {
		return "", "", errors.New("signature shall have two elements") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	if sHex, ok = readInt(); !ok {
		return "", "", errors.New("signature shall have two elements") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	return rHex, sHex, nil
}

// ecdsaIsASN1 reports whether hexStr is a self-consistent ASN.1 SEQUENCE whose
// declared length spans the whole input (jsrsasign's ASN1HEX.isASN1HEX).
func ecdsaIsASN1(hexStr string) bool {
	b, err := hex.DecodeString(hexStr)
	if err != nil || len(b) < 2 || b[0] != 0x30 {
		return false
	}
	length, hdr, ok := ecdsaDerLen(b, 1)
	return ok && 1+hdr+length == len(b)
}

// ecdsaPadHexLeft left-pads hex string s with '0' to length n.
func ecdsaPadHexLeft(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}

// ecdsaAsn1ToConcat converts an ASN.1 signature to fixed-width P1363, replicating
// jsrsasign's asn1SigToConcatSig (including its curve-length heuristics).
func ecdsaAsn1ToConcat(asn1Hex string) (string, error) {
	r, s, err := ecdsaParseSigRS(asn1Hex)
	if err != nil {
		return "", err
	}
	if len(r) >= 130 && len(r) <= 134 { // P-521: r/s content is always whole bytes
		r = strings.TrimPrefix(r, "00")
		s = strings.TrimPrefix(s, "00")
		c := max(len(r), len(s))
		return ecdsaPadHexLeft(r, c) + ecdsaPadHexLeft(s, c), nil
	}
	if strings.HasPrefix(r, "00") && len(r)%32 == 2 {
		r = r[2:]
	}
	if strings.HasPrefix(s, "00") && len(s)%32 == 2 {
		s = s[2:]
	}
	if len(r)%32 == 30 {
		r = "00" + r
	}
	if len(s)%32 == 30 {
		s = "00" + s
	}
	if len(r)%32 != 0 {
		return "", errors.New("unknown ECDSA sig r length error") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	if len(s)%32 != 0 {
		return "", errors.New("unknown ECDSA sig s length error") //nolint:staticcheck,revive // verbatim jsrsasign text
	}
	return r + s, nil
}

// ecdsaHexRSToAsn1 encodes hex integers r and s as a canonical DER
// SEQUENCE{INTEGER, INTEGER} (jsrsasign's hexRSSigToASN1Sig).
func ecdsaHexRSToAsn1(rHex, sHex string) (string, error) {
	r, ok := new(big.Int).SetString(rHex, 16)
	if !ok {
		return "", errors.New("invalid r value") //nolint:staticcheck,revive // CyberChef-style text
	}
	s, ok := new(big.Int).SetString(sHex, 16)
	if !ok {
		return "", errors.New("invalid s value") //nolint:staticcheck,revive // CyberChef-style text
	}
	der, err := encasn1.Marshal(struct{ R, S *big.Int }{r, s})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(der), nil
}

// ecdsaConcatToAsn1 converts a fixed-width P1363 signature to ASN.1
// (jsrsasign's concatSigToASN1Sig).
func ecdsaConcatToAsn1(concatHex string) (string, error) {
	if len(concatHex)%4 != 0 {
		return "", errors.New("unknown ECDSA concatinated r-s sig length error") //nolint:staticcheck,revive,misspell // verbatim jsrsasign text ("concatinated")
	}
	half := len(concatHex) / 2
	return ecdsaHexRSToAsn1(concatHex[:half], concatHex[half:])
}

// ecdsaParseJSONObject parses input as a JSON object, returning false if it is
// not valid JSON or not an object.
func ecdsaParseJSONObject(input string) (map[string]any, bool) {
	var v any
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

// ecdsaDetectFormat resolves the signature format when inputFormat is "Auto",
// trying JSON, then hex (ASN.1 vs P1363), then base64url (JWS). It returns any
// data parsed along the way so the caller need not re-parse.
func ecdsaDetectFormat(input, inputFormat string) (format string, jsonMap map[string]any, jwsBytes []byte) {
	if inputFormat != "Auto" {
		return inputFormat, nil, nil
	}
	if m, ok := ecdsaParseJSONObject(input); ok {
		return "Raw JSON", m, nil
	}
	if ecdsaHexRe.MatchString(input) {
		if strings.HasPrefix(input, "30") && ecdsaIsASN1(input) {
			return "ASN.1 HEX", nil, nil
		}
		return "P1363 HEX", nil, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(input); err == nil {
		return "JSON Web Signature", nil, b
	}
	return "Auto", nil, nil
}

// ecdsaInputToASN1 detects (when inputFormat is "Auto") and converts a signature
// in any supported format to ASN.1 hex.
func ecdsaInputToASN1(input, inputFormat string) (string, error) {
	format, jsonMap, jwsBytes := ecdsaDetectFormat(input, inputFormat)
	switch format {
	case "ASN.1 HEX":
		return input, nil
	case "P1363 HEX":
		return ecdsaConcatToAsn1(input)
	case "JSON Web Signature":
		if jwsBytes == nil {
			b, err := base64.RawURLEncoding.DecodeString(input)
			if err != nil {
				return "", err
			}
			jwsBytes = b
		}
		return ecdsaConcatToAsn1(hex.EncodeToString(jwsBytes))
	case "Raw JSON":
		if jsonMap == nil {
			m, ok := ecdsaParseJSONObject(input)
			if !ok {
				return "", errors.New("Signature is not valid JSON") //nolint:staticcheck,revive // CyberChef-style text
			}
			jsonMap = m
		}
		r, _ := jsonMap["r"].(string)
		s, _ := jsonMap["s"].(string)
		if r == "" {
			return "", errors.New(`No "r" value in the signature JSON`) //nolint:staticcheck,revive // verbatim CyberChef text
		}
		if s == "" {
			return "", errors.New(`No "s" value in the signature JSON`) //nolint:staticcheck,revive // verbatim CyberChef text
		}
		return ecdsaHexRSToAsn1(r, s)
	default:
		return "", errors.New("Signature format could not be detected") //nolint:staticcheck,revive // verbatim CyberChef text
	}
}

// ecdsaFromASN1 converts an ASN.1 hex signature to the requested output format.
func ecdsaFromASN1(asn1Hex, outputFormat string) (string, error) {
	switch outputFormat {
	case "ASN.1 HEX":
		return asn1Hex, nil
	case "P1363 HEX":
		return ecdsaAsn1ToConcat(asn1Hex)
	case "JSON Web Signature":
		p1363, err := ecdsaAsn1ToConcat(asn1Hex)
		if err != nil {
			return "", err
		}
		raw, err := hex.DecodeString(p1363)
		if err != nil {
			return "", err
		}
		return base64.RawURLEncoding.EncodeToString(raw), nil
	default: // Raw JSON
		r, s, err := ecdsaParseSigRS(asn1Hex)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(struct {
			R string `json:"r"`
			S string `json:"s"`
		}{r, s})
		return string(b), err
	}
}

// --- Keys and hashing ---

// ecdsaKey is the result of parsing a PEM key: its algorithm and whether it is a
// private or public key, plus whichever ECDSA key was recovered.
type ecdsaKey struct {
	algo      string // "EC", "RSA", or "" for others
	isPrivate bool
	isPublic  bool
	priv      *ecdsa.PrivateKey
	pub       *ecdsa.PublicKey
}

// ecdsaGetKey parses a PEM key, mirroring jsrsasign KEYUTIL.getKey's type and
// public/private classification.
func ecdsaGetKey(pemStr string) (ecdsaKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return ecdsaKey{}, errors.New("failed to parse the key") //nolint:staticcheck,revive // CyberChef-style text
	}
	switch block.Type {
	case "EC PRIVATE KEY":
		k, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return ecdsaKey{}, err
		}
		return ecdsaKey{algo: "EC", isPrivate: true, priv: k}, nil
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return ecdsaKey{}, err
		}
		if ec, ok := k.(*ecdsa.PrivateKey); ok {
			return ecdsaKey{algo: "EC", isPrivate: true, priv: ec}, nil
		}
		if _, ok := k.(*rsa.PrivateKey); ok {
			return ecdsaKey{algo: "RSA", isPrivate: true}, nil
		}
		return ecdsaKey{isPrivate: true}, nil
	case "RSA PRIVATE KEY":
		return ecdsaKey{algo: "RSA", isPrivate: true}, nil
	case "PUBLIC KEY":
		k, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return ecdsaKey{}, err
		}
		if ec, ok := k.(*ecdsa.PublicKey); ok {
			return ecdsaKey{algo: "EC", isPublic: true, pub: ec}, nil
		}
		if _, ok := k.(*rsa.PublicKey); ok {
			return ecdsaKey{algo: "RSA", isPublic: true}, nil
		}
		return ecdsaKey{isPublic: true}, nil
	default:
		return ecdsaKey{}, fmt.Errorf("unsupported key type %q", block.Type)
	}
}

// ecdsaHash hashes msg with the named digest (SHA-1 and MD5 are offered because
// CyberChef offers them).
func ecdsaHash(mdAlgo string, msg []byte) []byte {
	switch mdAlgo {
	case "SHA-384":
		h := sha512.Sum384(msg)
		return h[:]
	case "SHA-512":
		h := sha512.Sum512(msg)
		return h[:]
	case "SHA-1":
		h := sha1.Sum(msg) // #nosec G401 -- user-selectable digest, matching CyberChef
		return h[:]
	case "MD5":
		h := md5.Sum(msg) // #nosec G401 -- user-selectable digest, matching CyberChef
		return h[:]
	default: // SHA-256
		h := sha256.Sum256(msg)
		return h[:]
	}
}

// ecdsaDecodeMessage decodes the verification message per the chosen format.
func ecdsaDecodeMessage(msg, format string) ([]byte, error) {
	switch format {
	case "Hex":
		return splitHexToBytes(msg), nil
	case "Base64":
		return base64.StdEncoding.DecodeString(msg)
	default: // Raw
		return []byte(msg), nil
	}
}

// --- Operations ---

// ECDSASign signs a message with a PEM EC private key.
type ECDSASign struct{}

// Meta returns the operation metadata.
func (ECDSASign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ECDSA Sign",
		Module:      "Ciphers",
		Description: "Sign a plaintext message with a PEM encoded EC key.",
		InfoURL:     "https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ECDSASign) Args() []core.ArgDef {
	return []core.ArgDef{
		// #nosec G101 -- default is an empty PEM header placeholder, not a credential
		{Name: "ECDSA Private Key (PEM)", Type: core.ArgString, Value: "-----BEGIN EC PRIVATE KEY-----"},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: ecdsaMDs},
		{Name: "Output Format", Type: core.ArgOption, Value: ecdsaSigFormats},
	}
}

// Run signs the input.
func (ECDSASign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[0].(string)
	if len(strings.Replace(keyPem, "-----BEGIN EC PRIVATE KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a private key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	key, err := ecdsaGetKey(keyPem)
	if err != nil {
		return nil, err
	}
	if key.algo != "EC" {
		return nil, errors.New("Provided key is not an EC key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if !key.isPrivate {
		return nil, errors.New("Provided key is not a private key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	der, err := ecdsa.SignASN1(rand.Reader, key.priv, ecdsaHash(args[1].(string), in.Bytes()))
	if err != nil {
		return nil, err
	}
	out, err := ecdsaFromASN1(hex.EncodeToString(der), args[2].(string))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// ECDSAVerify verifies a signature against a message and a PEM EC public key.
type ECDSAVerify struct{}

// Meta returns the operation metadata.
func (ECDSAVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ECDSA Verify",
		Module:      "Ciphers",
		Description: "Verify a message against a signature and a public PEM encoded EC key.",
		InfoURL:     "https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ECDSAVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input Format", Type: core.ArgOption, Value: ecdsaInputFormats},
		{Name: "Message Digest Algorithm", Type: core.ArgOption, Value: ecdsaMDs},
		{Name: "ECDSA Public Key (PEM)", Type: core.ArgString, Value: "-----BEGIN PUBLIC KEY-----"},
		{Name: "Message", Type: core.ArgString, Value: ""},
		{Name: "Message format", Type: core.ArgOption, Value: []string{"Raw", "Hex", "Base64"}},
	}
}

// Run verifies the signature.
func (ECDSAVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	keyPem := args[2].(string)
	if len(strings.Replace(keyPem, "-----BEGIN PUBLIC KEY-----", "", 1)) == 0 {
		return nil, errors.New("Please enter a public key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	asn1Hex, err := ecdsaInputToASN1(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	key, err := ecdsaGetKey(keyPem)
	if err != nil {
		return nil, err
	}
	if key.algo != "EC" {
		return nil, errors.New("Provided key is not an EC key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if !key.isPublic {
		return nil, errors.New("Provided key is not a public key.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	msg, err := ecdsaDecodeMessage(args[3].(string), args[4].(string))
	if err != nil {
		return nil, err
	}
	der, err := hex.DecodeString(asn1Hex)
	if err != nil {
		return nil, err
	}
	result := "Verification Failure"
	if ecdsa.VerifyASN1(key.pub, ecdsaHash(args[1].(string), msg), der) {
		result = "Verified OK"
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}

// ECDSASignatureConversion converts a signature between formats.
type ECDSASignatureConversion struct{}

// Meta returns the operation metadata.
func (ECDSASignatureConversion) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ECDSA Signature Conversion",
		Module:      "Ciphers",
		Description: "Convert an ECDSA signature between hex, asn1 and json.",
		InfoURL:     "https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ECDSASignatureConversion) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input Format", Type: core.ArgOption, Value: ecdsaInputFormats},
		{Name: "Output Format", Type: core.ArgOption, Value: ecdsaSigFormats},
	}
}

// Run converts the signature.
func (ECDSASignatureConversion) Run(in *core.Dish, args []any) (*core.Dish, error) {
	asn1Hex, err := ecdsaInputToASN1(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	out, err := ecdsaFromASN1(asn1Hex, args[1].(string))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// ecdsaJWK is a JSON Web Key, with fields in jsrsasign's emission order.
type ecdsaJWK struct {
	Kty    string   `json:"kty"`
	Crv    string   `json:"crv"`
	X      string   `json:"x"`
	Y      string   `json:"y"`
	D      string   `json:"d,omitempty"`
	KeyOps []string `json:"key_ops"`
	Kid    string   `json:"kid"`
}

// GenerateECDSAKeyPair generates an EC key pair.
type GenerateECDSAKeyPair struct{}

// Meta returns the operation metadata.
func (GenerateECDSAKeyPair) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate ECDSA Key Pair",
		Module:      "Ciphers",
		Description: "Generate an ECDSA key pair with a given Curve.",
		InfoURL:     "https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateECDSAKeyPair) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Elliptic Curve", Type: core.ArgOption, Value: []string{"P-256", "P-384", "P-521"}},
		{Name: "Output Format", Type: core.ArgOption, Value: []string{"PEM", "DER", "JWK"}},
	}
}

// ecdsaCurveByName returns the curve, its short name and its coordinate byte size.
func ecdsaCurveByName(name string) (elliptic.Curve, int) {
	switch name {
	case "P-384":
		return elliptic.P384(), 48
	case "P-521":
		return elliptic.P521(), 66
	default: // P-256
		return elliptic.P256(), 32
	}
}

// Run generates the key pair.
func (GenerateECDSAKeyPair) Run(_ *core.Dish, args []any) (*core.Dish, error) {
	curveName := args[0].(string)
	curve, byteLen := ecdsaCurveByName(curveName)
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	// Use the crypto/ecdh view to read the raw scalar and point without the
	// deprecated *big.Int fields of ecdsa.{Private,Public}Key.
	ecdhPriv, err := priv.ECDH()
	if err != nil {
		return nil, err
	}
	scalar := ecdhPriv.Bytes()            // byteLen bytes
	point := ecdhPriv.PublicKey().Bytes() // 0x04 || X || Y
	xB, yB := point[1:1+byteLen], point[1+byteLen:]
	b64 := base64.RawURLEncoding.EncodeToString

	var result string
	switch args[1].(string) {
	case "DER":
		result = hex.EncodeToString(scalar)
	case "JWK":
		x, y := b64(xB), b64(yB)
		privJWK := ecdsaJWK{Kty: "EC", Crv: curveName, X: x, Y: y, D: b64(scalar), KeyOps: []string{"sign"}, Kid: "PrivateKey"}
		pubJWK := ecdsaJWK{Kty: "EC", Crv: curveName, X: x, Y: y, KeyOps: []string{"verify"}, Kid: "PublicKey"}
		b, err := json.MarshalIndent(struct {
			Keys []ecdsaJWK `json:"keys"`
		}{[]ecdsaJWK{privJWK, pubJWK}}, "", "    ")
		if err != nil {
			return nil, err
		}
		result = string(b)
	default: // PEM
		pubDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		if err != nil {
			return nil, err
		}
		privDer, err := x509.MarshalPKCS8PrivateKey(priv)
		if err != nil {
			return nil, err
		}
		pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})
		privPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDer})
		result = string(pubPem) + "\n" + string(privPem)
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}
