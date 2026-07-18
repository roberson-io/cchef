package ops

import (
	"crypto/elliptic"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JWTDecode{})
	core.Register(JWTSign{})
	core.Register(JWTVerify{})
}

// jwtAlgorithms is the list of signing algorithms CyberChef offers, in order.
var jwtAlgorithms = []string{
	"HS256", "HS384", "HS512",
	"RS256", "RS384", "RS512",
	"ES256", "ES384", "ES512",
	"None",
}

// jwtSegEnc is the base64url (no padding) encoding used for every JWT segment.
var jwtSegEnc = base64.RawURLEncoding

// jwtRSAMinBits is the minimum RSA key size jsonwebtoken enforces for RS/PS algs.
const jwtRSAMinBits = 2048

// jwtSignKeyHint is the prefix jsonwebtoken adds to every JWT Sign failure.
const jwtSignKeyHint = "Error: Have you entered the key correctly? The key should be either the secret for HMAC algorithms or the PEM-encoded private key for RSA and ECDSA."

// jwtNow returns the current Unix time in seconds, matching jsonwebtoken's iat.
func jwtNow() int64 { return time.Now().Unix() }

// JWTDecode decodes a JSON Web Token without verifying its signature.
type JWTDecode struct{}

// Meta returns the operation metadata.
func (JWTDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JWT Decode",
		Module:      "Crypto",
		Description: "Decodes a JSON Web Token without checking whether the provided secret / private key is valid. Use 'JWT Verify' to check if the signature is valid as well.",
		InfoURL:     "https://wikipedia.org/wiki/JSON_Web_Token",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (JWTDecode) Args() []core.ArgDef { return nil }

// Run decodes the token's payload without checking its signature.
func (JWTDecode) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	payload, err := jwtSegment(in.String(), 1, "JWT Decode")
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(jsStringify(payload, 4)), core.TypeJSON), nil
}

// JWTSign signs a JSON object as a JSON Web Token.
type JWTSign struct{}

// Meta returns the operation metadata.
func (JWTSign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JWT Sign",
		Module:      "Crypto",
		Description: "Signs a JSON object as a JSON Web Token using a provided secret / private key. The key should be either the secret for HMAC algorithms or the PEM-encoded private key for RSA and ECDSA.",
		InfoURL:     "https://wikipedia.org/wiki/JSON_Web_Token",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JWTSign) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Private/Secret Key", Type: core.ArgString, Value: "secret"},
		{Name: "Signing algorithm", Type: core.ArgOption, Value: jwtAlgorithms},
		{Name: "Header", Type: core.ArgString, Value: "{}"},
	}
}

// Run signs the input payload as a JWT.
func (JWTSign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, alg, header := args[0].(string), args[1].(string), args[2].(string)
	token, err := jwtSign(in.Bytes(), key, alg, header)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(token), core.TypeString), nil
}

// JWTVerify verifies a JSON Web Token's signature and returns its payload.
type JWTVerify struct{}

// Meta returns the operation metadata.
func (JWTVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JWT Verify",
		Module:      "Crypto",
		Description: "Verifies that a JSON Web Token is valid and has been signed with the provided secret / private key. The key should be either the secret for HMAC algorithms or the PEM-encoded public key for RSA and ECDSA.",
		InfoURL:     "https://wikipedia.org/wiki/JSON_Web_Token",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (JWTVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Public/Secret Key", Type: core.ArgString, Value: "secret"},
	}
}

// Run verifies the token's signature and returns its payload.
func (JWTVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	payload, err := jwtVerify(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(jsStringify(payload, 4)), core.TypeJSON), nil
}

// jwtSegment splits a token and returns its base64url-decoded, JSON-parsed
// segment n (0 header, 1 payload), preserving key order. who names the op for
// error messages.
func jwtSegment(token string, n int, who string) (any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%s: invalid token: expected 3 segments", who)
	}
	raw, err := jwtSegEnc.DecodeString(parts[n])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", who, err)
	}
	v, err := jsonParseOrdered(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", who, err)
	}
	return v, nil
}

// jwtSign builds and signs a JWT, wrapping every failure with jsonwebtoken's key
// hint (matching CyberChef's JWT Sign error output).
func jwtSign(payloadJSON []byte, key, alg, headerArg string) (string, error) {
	payload, err := jwtBuildPayload(payloadJSON)
	if err != nil {
		return "", jwtSignError(err)
	}
	header, err := jwtBuildHeader(alg, headerArg)
	if err != nil {
		return "", jwtSignError(err)
	}
	input := jwtSegEnc.EncodeToString([]byte(jsStringify(header, 0))) + "." +
		jwtSegEnc.EncodeToString([]byte(jsStringify(payload, 0)))
	sig, err := jwtSignSegment(input, key, alg)
	if err != nil {
		return "", jwtSignError(err)
	}
	return input + "." + sig, nil
}

// jwtSignError wraps err with jsonwebtoken's key hint, matching CyberChef's JWT
// Sign error output (the "Error: " prefix mirrors a JavaScript Error's toString).
func jwtSignError(err error) error {
	return fmt.Errorf("%s\n\nError: %w", jwtSignKeyHint, err)
}

// jwtBuildPayload parses the input JSON object preserving key order and appends
// iat with the current timestamp when absent, as jsonwebtoken does.
func jwtBuildPayload(data []byte) (jsObject, error) {
	v, err := jsonParseOrdered(data)
	if err != nil {
		return nil, err
	}
	obj, ok := v.(jsObject)
	if !ok {
		return nil, errors.New("payload must be a JSON object")
	}
	if jsIndex(obj, "iat") < 0 {
		obj = append(obj, jsPair{k: "iat", v: float64(jwtNow())})
	}
	return obj, nil
}

// jwtBuildHeader constructs the JWT header the way jsonwebtoken does: a base of
// {alg, typ:"JWT", kid:undefined} with the user's header object merged over it
// (existing keys updated in place, new keys appended).
func jwtBuildHeader(alg, headerArg string) (jsObject, error) {
	algField := alg
	if alg == "None" {
		algField = "none"
	}
	header := jsObject{
		{k: "alg", v: algField},
		{k: "typ", v: "JWT"},
		{k: "kid", v: jsUndefined{}},
	}
	if strings.TrimSpace(headerArg) == "" {
		headerArg = "{}"
	}
	custom, err := jsonParseOrdered([]byte(headerArg))
	if err != nil {
		return nil, err
	}
	customObj, ok := custom.(jsObject)
	if !ok {
		return nil, errors.New("header must be a JSON object")
	}
	for _, p := range customObj {
		if i := jsIndex(header, p.k); i >= 0 {
			header[i].v = p.v
		} else {
			header = append(header, p)
		}
	}
	return header, nil
}

// jwtSignSegment computes the base64url signature over input for the given alg.
func jwtSignSegment(input, key, alg string) (string, error) {
	if alg == "None" {
		return "", nil
	}
	method := jwt.GetSigningMethod(alg)
	if method == nil {
		return "", fmt.Errorf("unsupported algorithm %q", alg)
	}
	signKey, err := jwtSigningKey(alg, key)
	if err != nil {
		return "", err
	}
	return jwtSignWith(method, input, signKey)
}

// jwtSignWith signs input with method and returns the base64url signature.
func jwtSignWith(method jwt.SigningMethod, input string, signKey any) (string, error) {
	sig, err := method.Sign(input, signKey)
	if err != nil {
		return "", err
	}
	return jwtSegEnc.EncodeToString(sig), nil
}

// jwtSigningKey parses the private/secret key for alg and enforces the same
// preconditions jsonwebtoken does (RSA minimum size, ECDSA curve match).
func jwtSigningKey(alg, key string) (any, error) {
	switch alg[:2] {
	case "HS":
		return []byte(key), nil
	case "RS", "PS":
		priv, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key))
		if err != nil {
			return nil, err
		}
		if priv.N.BitLen() < jwtRSAMinBits {
			//nolint:staticcheck,revive // jsonwebtoken's verbatim OperationError text
			return nil, fmt.Errorf("secretOrPrivateKey has a minimum key size of %d bits for %s", jwtRSAMinBits, alg)
		}
		return priv, nil
	default: // ES
		priv, err := jwt.ParseECPrivateKeyFromPEM([]byte(key))
		if err != nil {
			return nil, err
		}
		if err := jwtCheckCurve(alg, priv.Curve); err != nil {
			return nil, err
		}
		return priv, nil
	}
}

// jwtESCurve pairs each ECDSA algorithm with its required curve and the OpenSSL
// curve name jsonwebtoken reports in its error message.
type jwtESCurve struct {
	curve elliptic.Curve
	name  string
}

var jwtESCurves = map[string]jwtESCurve{
	"ES256": {elliptic.P256(), "prime256v1"},
	"ES384": {elliptic.P384(), "secp384r1"},
	"ES512": {elliptic.P521(), "secp521r1"},
}

// jwtCheckCurve verifies the key's curve matches the one the algorithm requires.
func jwtCheckCurve(alg string, curve elliptic.Curve) error {
	want := jwtESCurves[alg]
	if curve != want.curve {
		//nolint:staticcheck,revive // jsonwebtoken's verbatim OperationError text
		return fmt.Errorf("\"alg\" parameter %q requires curve %q.", alg, want.name)
	}
	return nil
}

// jwtVerify verifies the token's signature against key and returns its payload.
func jwtVerify(token, key string) (any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("JWT Verify: invalid token: expected 3 segments")
	}
	header, err := jwtSegment(token, 0, "JWT Verify")
	if err != nil {
		return nil, err
	}
	alg, err := jwtHeaderAlg(header)
	if err != nil {
		return nil, err
	}
	sig, err := jwtSegEnc.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("JWT Verify: %w", err)
	}
	if err := jwtVerifySignature(parts[0]+"."+parts[1], sig, alg, key); err != nil {
		return nil, err
	}
	payload, err := jwtSegment(token, 1, "JWT Verify")
	if err != nil {
		return nil, err
	}
	if err := jwtCheckClaims(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// jwtHeaderAlg extracts the "alg" string from a decoded JWT header.
func jwtHeaderAlg(header any) (string, error) {
	obj, ok := header.(jsObject)
	if !ok {
		return "", errors.New("JWT Verify: invalid header")
	}
	i := jsIndex(obj, "alg")
	if i < 0 {
		return "", errors.New("JWT Verify: invalid algorithm")
	}
	alg, ok := obj[i].v.(string)
	if !ok {
		return "", errors.New("JWT Verify: invalid algorithm")
	}
	return alg, nil
}

// jwtVerifySignature checks sig over input using the algorithm named in the token.
func jwtVerifySignature(input string, sig []byte, alg, key string) error {
	if alg == "none" {
		if len(sig) != 0 {
			return errors.New("JWT Verify: invalid signature")
		}
		return nil
	}
	method := jwt.GetSigningMethod(alg)
	if method == nil {
		return errors.New("JWT Verify: invalid algorithm")
	}
	verifyKey, err := jwtVerifyKey(alg, key)
	if err != nil {
		return err
	}
	if err := method.Verify(input, sig, verifyKey); err != nil {
		return fmt.Errorf("JWT Verify: %w", err)
	}
	return nil
}

// jwtVerifyKey parses the public/secret key used to verify a token of type alg.
func jwtVerifyKey(alg, key string) (any, error) {
	switch alg[:2] {
	case "HS":
		return []byte(key), nil
	case "RS", "PS":
		pub, err := jwt.ParseRSAPublicKeyFromPEM([]byte(key))
		if err != nil {
			return nil, fmt.Errorf("JWT Verify: %w", err)
		}
		return pub, nil
	default: // ES
		pub, err := jwt.ParseECPublicKeyFromPEM([]byte(key))
		if err != nil {
			return nil, fmt.Errorf("JWT Verify: %w", err)
		}
		return pub, nil
	}
}

// jwtCheckClaims enforces the nbf/exp time claims jsonwebtoken validates.
func jwtCheckClaims(payload any) error {
	obj, ok := payload.(jsObject)
	if !ok {
		return nil
	}
	now := float64(jwtNow())
	if i := jsIndex(obj, "nbf"); i >= 0 {
		if nbf, ok := obj[i].v.(float64); ok && now < nbf {
			return errors.New("JWT Verify: jwt not active")
		}
	}
	if i := jsIndex(obj, "exp"); i >= 0 {
		if exp, ok := obj[i].v.(float64); ok && now >= exp {
			return errors.New("JWT Verify: jwt expired")
		}
	}
	return nil
}
