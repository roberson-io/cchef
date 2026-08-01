package ops

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(JWKToPEM{})
	core.Register(PEMToJWK{})
}

// CyberChef OperationError text, reproduced verbatim (hence the capitalisation
// staticcheck/revive would otherwise flag).
var (
	errNotJWK     = errors.New("Input is not a JSON Web Key") //nolint:staticcheck,revive // verbatim CyberChef text
	errInvalidJWK = errors.New("Invalid JWK format")          //nolint:staticcheck,revive // verbatim CyberChef text
)

// JWKToPEM converts JSON Web Keys (a single key, an array, or a JWK Set) into
// PEM: PKCS#8 for private keys, SPKI for public keys. Ported from CyberChef's
// JWKToPem.mjs (jsrsasign KEYUTIL.getPEM); only RSA and EC keys are supported,
// as upstream. Go's crypto/x509 marshaling is byte-identical to jsrsasign for
// these key types.
type JWKToPEM struct{}

// Meta returns the operation metadata.
func (JWKToPEM) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JWK to PEM",
		Module:      "PublicKey",
		Description: "Converts Keys in JSON Web Key format to PEM format (PKCS#8).",
		InfoURL:     "https://datatracker.ietf.org/doc/html/rfc7517",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JWKToPEM) Args() []core.ArgDef { return nil }

// Run converts each JWK in the input to PEM and concatenates the results.
func (JWKToPEM) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var parsed any
	if err := json.Unmarshal([]byte(in.String()), &parsed); err != nil {
		return nil, errNotJWK
	}

	var keys []any
	switch v := parsed.(type) {
	case []any:
		// list of keys => transform all keys
		keys = v
	case map[string]any:
		if ks, ok := v["keys"].([]any); ok {
			// JSON Web Key Set => transform all keys
			keys = ks
		} else {
			// single key
			keys = []any{v}
		}
	default:
		return nil, errNotJWK
	}

	var out strings.Builder
	for _, k := range keys {
		p, err := jwkToPEM(k)
		if err != nil {
			return nil, err
		}
		out.WriteString(p)
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// jwkToPEM converts one parsed JWK into its PEM encoding.
func jwkToPEM(k any) (string, error) {
	jwk, _ := k.(map[string]any)
	kty, ok := jwk["kty"].(string)
	if !ok {
		return "", errInvalidJWK
	}
	switch kty {
	case "RSA":
		return jwkRSAToPEM(jwk)
	case "EC":
		return jwkECToPEM(jwk)
	default:
		return "", fmt.Errorf("Unsupported JWK key type '%s'", kty) //nolint:staticcheck,revive // verbatim CyberChef text
	}
}

// jwkRSAToPEM builds an RSA key from a JWK and marshals it (PKCS#8 if the
// private exponent is present, else SPKI).
func jwkRSAToPEM(jwk map[string]any) (string, error) {
	n, err := jwkBigInt(jwk, "n")
	if err != nil {
		return "", err
	}
	e, err := jwkBigInt(jwk, "e")
	if err != nil {
		return "", err
	}
	pub := &rsa.PublicKey{N: n, E: int(e.Int64())}

	if _, hasD := jwk["d"]; !hasD {
		return pkixPEM(pub)
	}

	d, err := jwkBigInt(jwk, "d")
	if err != nil {
		return "", err
	}
	p, err := jwkBigInt(jwk, "p")
	if err != nil {
		return "", err
	}
	q, err := jwkBigInt(jwk, "q")
	if err != nil {
		return "", err
	}
	priv := &rsa.PrivateKey{PublicKey: *pub, D: d, Primes: []*big.Int{p, q}}
	priv.Precompute()
	return pkcs8PEM(priv)
}

// jwkECToPEM builds an EC key from a JWK and marshals it (PKCS#8 if the private
// scalar is present, else SPKI).
func jwkECToPEM(jwk map[string]any) (string, error) {
	crv, _ := jwk["crv"].(string)
	curve, err := ecCurve(crv)
	if err != nil {
		return "", err
	}
	x, err := jwkBigInt(jwk, "x")
	if err != nil {
		return "", err
	}
	y, err := jwkBigInt(jwk, "y")
	if err != nil {
		return "", err
	}
	// JWK carries the raw affine coordinates; an off-curve point is rejected by
	// the marshaler. (crypto/ecdsa deprecates raw coordinate use, but reproducing
	// JWK requires exactly that.)
	pub := &ecdsa.PublicKey{Curve: curve, X: x, Y: y} //nolint:staticcheck // JWK is defined in terms of raw EC coordinates

	if _, hasD := jwk["d"]; !hasD {
		return pkixPEM(pub)
	}

	d, err := jwkBigInt(jwk, "d")
	if err != nil {
		return "", err
	}
	priv := &ecdsa.PrivateKey{PublicKey: *pub, D: d} //nolint:staticcheck // JWK is defined in terms of the raw EC scalar
	return pkcs8PEM(priv)
}

// pkixPEM marshals a public key as an SPKI PEM block.
func pkixPEM(pub any) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	return keyPEM("PUBLIC KEY", der), nil
}

// pkcs8PEM marshals a private key as a PKCS#8 PEM block.
func pkcs8PEM(priv any) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}
	return keyPEM("PRIVATE KEY", der), nil
}

// PEMToJWK converts PEM keys and certificates to JSON Web Key format. Ported
// from CyberChef's PEMToJWK.mjs (jsrsasign KEYUTIL.getJWKFromKey): each PEM
// block is scanned out, parsed, and emitted as a compact JWK; multiple keys are
// newline-separated. Only RSA and EC keys are supported, as upstream.
type PEMToJWK struct{}

// Meta returns the operation metadata.
func (PEMToJWK) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PEM to JWK",
		Module:      "PublicKey",
		Description: "Converts Keys in PEM format to a JSON Web Key format.",
		InfoURL:     "https://datatracker.ietf.org/doc/html/rfc7517",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PEMToJWK) Args() []core.ArgDef { return nil }

// Run scans each PEM block from the input and converts it to a JWK.
func (PEMToJWK) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var out strings.Builder
	for _, loc := range pemBeginRe.FindAllStringSubmatchIndex(input, -1) {
		typ := input[loc[2]:loc[3]]
		indexBase64 := loc[1]
		footer := "-----END " + typ + "-----"
		rel := strings.Index(input[indexBase64:], footer)
		if rel == -1 {
			return nil, fmt.Errorf("PEM footer '%s' not found", footer) //nolint:staticcheck,revive // verbatim CyberChef text
		}
		block := input[loc[0] : indexBase64+rel+len(footer)]

		jwk, err := pemBlockToJWK(typ, block)
		if err != nil {
			return nil, err
		}
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString(jwk)
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// pemBlockToJWK converts a single PEM block (identified by its BEGIN type) to a
// compact JWK JSON string.
func pemBlockToJWK(typ, block string) (string, error) {
	switch {
	case typ == "CERTIFICATE":
		der, err := pemDER(block)
		if err != nil {
			return "", err
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return "", err
		}
		return keyToJWK(cert.PublicKey)
	case strings.Contains(typ, "KEY"):
		if typ == "RSA PUBLIC KEY" {
			return "", errors.New("Unsupported RSA public key format. Only PKCS#8 is supported.") //nolint:staticcheck,revive // verbatim CyberChef text
		}
		if strings.Contains(typ, "DSA") {
			return "", errors.New("DSA keys are not supported for JWK") //nolint:staticcheck,revive // verbatim CyberChef text
		}
		key, err := parsePEMKey(typ, block)
		if err != nil {
			return "", err
		}
		return keyToJWK(key)
	default:
		return "", fmt.Errorf("Unsupported PEM type '%s'", typ) //nolint:staticcheck,revive // verbatim CyberChef text
	}
}

// parsePEMKey parses a key PEM block into a crypto key, dispatching on the
// BEGIN type.
func parsePEMKey(typ, block string) (any, error) {
	der, err := pemDER(block)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(der)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(der)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(der)
	case "PUBLIC KEY":
		return x509.ParsePKIXPublicKey(der)
	default:
		return nil, fmt.Errorf("Unsupported PEM type '%s'", typ) //nolint:staticcheck,revive // verbatim CyberChef text
	}
}

// keyToJWK renders a parsed crypto key as a compact JWK JSON string.
func keyToJWK(key any) (string, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		k.Precompute()
		return jwkJSON([][2]string{
			{"kty", "RSA"},
			{"n", b64u(k.N.Bytes())},
			{"e", b64u(big.NewInt(int64(k.E)).Bytes())},
			{"d", b64u(k.D.Bytes())},
			{"p", b64u(k.Primes[0].Bytes())},
			{"q", b64u(k.Primes[1].Bytes())},
			{"dp", b64u(k.Precomputed.Dp.Bytes())},
			{"dq", b64u(k.Precomputed.Dq.Bytes())},
			{"qi", b64u(k.Precomputed.Qinv.Bytes())},
		}), nil
	case *rsa.PublicKey:
		return jwkJSON([][2]string{
			{"kty", "RSA"},
			{"n", b64u(k.N.Bytes())},
			{"e", b64u(big.NewInt(int64(k.E)).Bytes())},
		}), nil
	case *ecdsa.PrivateKey:
		name, err := ecCurveName(k.Curve)
		if err != nil {
			return "", err
		}
		size := ecCoordLen(k.Curve)
		return jwkJSON([][2]string{
			{"kty", "EC"},
			{"crv", name},
			{"x", b64u(leftPadBytes(k.X.Bytes(), size))}, //nolint:staticcheck // JWK is defined in terms of raw EC coordinates
			{"y", b64u(leftPadBytes(k.Y.Bytes(), size))}, //nolint:staticcheck // JWK is defined in terms of raw EC coordinates
			{"d", b64u(leftPadBytes(k.D.Bytes(), size))}, //nolint:staticcheck // JWK is defined in terms of the raw EC scalar
		}), nil
	case *ecdsa.PublicKey:
		name, err := ecCurveName(k.Curve)
		if err != nil {
			return "", err
		}
		size := ecCoordLen(k.Curve)
		return jwkJSON([][2]string{
			{"kty", "EC"},
			{"crv", name},
			{"x", b64u(leftPadBytes(k.X.Bytes(), size))}, //nolint:staticcheck // JWK is defined in terms of raw EC coordinates
			{"y", b64u(leftPadBytes(k.Y.Bytes(), size))}, //nolint:staticcheck // JWK is defined in terms of raw EC coordinates
		}), nil
	default:
		return "", errors.New("Unsupported key type for JWK") //nolint:staticcheck,revive // verbatim CyberChef-style text
	}
}

// --- helpers ------------------------------------------------------------------

// b64u base64url-encodes bytes without padding, matching jsrsasign's hextob64u.
func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// jwkRawBytes decodes a base64url JWK member to its raw bytes.
func jwkRawBytes(jwk map[string]any, field string) ([]byte, error) {
	s, ok := jwk[field].(string)
	if !ok {
		return nil, errInvalidJWK
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errInvalidJWK
	}
	return raw, nil
}

// jwkBigInt reads a base64url JWK member as a big integer.
func jwkBigInt(jwk map[string]any, field string) (*big.Int, error) {
	raw, err := jwkRawBytes(jwk, field)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(raw), nil
}

// pemDER decodes the DER bytes from a single PEM block.
func pemDER(block string) ([]byte, error) {
	b, _ := pem.Decode([]byte(block))
	if b == nil {
		return nil, errors.New("Invalid PEM data") //nolint:staticcheck,revive // CyberChef-style text
	}
	return b.Bytes, nil
}

// jwkJSON serialises ordered key/value string pairs as a compact JSON object,
// matching JavaScript JSON.stringify (insertion order, standard escaping).
func jwkJSON(pairs [][2]string) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i, p := range pairs {
		if i > 0 {
			sb.WriteByte(',')
		}
		k, _ := json.Marshal(p[0])
		v, _ := json.Marshal(p[1])
		sb.Write(k)
		sb.WriteByte(':')
		sb.Write(v)
	}
	sb.WriteByte('}')
	return sb.String()
}

// keyPEM encodes DER as a PEM block with the CRLF line endings CyberChef emits.
func keyPEM(typ string, der []byte) string {
	b := pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der})
	return strings.ReplaceAll(string(b), "\n", "\r\n")
}

// ecCurve maps a JWK curve name to its elliptic.Curve.
func ecCurve(name string) (elliptic.Curve, error) {
	switch name {
	case "P-256":
		return elliptic.P256(), nil
	case "P-384":
		return elliptic.P384(), nil
	case "P-521":
		return elliptic.P521(), nil
	default:
		return nil, fmt.Errorf("unsupported curve name for JWT: %s", name)
	}
}

// ecCurveName maps an elliptic.Curve to its JWK curve name.
func ecCurveName(c elliptic.Curve) (string, error) {
	switch c {
	case elliptic.P256():
		return "P-256", nil
	case elliptic.P384():
		return "P-384", nil
	case elliptic.P521():
		return "P-521", nil
	default:
		return "", errors.New("unsupported curve name for JWT")
	}
}

// ecCoordLen returns the fixed byte length of a coordinate on the curve.
func ecCoordLen(c elliptic.Curve) int {
	return (c.Params().BitSize + 7) / 8
}

// leftPad zero-pads b on the left to exactly n bytes (b is assumed <= n).
func leftPadBytes(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}
