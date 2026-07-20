package ops

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	encasn1 "encoding/asn1"
	"errors"
	"math/big"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(PubKeyFromPrivKey{})
}

// privKeyHeaderRe matches a private-key PEM header, capturing the full label
// (e.g. "RSA PRIVATE KEY", "PRIVATE KEY") so the corresponding footer can be
// built, mirroring CyberChef's regex in PubKeyFromPrivKey.mjs.
var privKeyHeaderRe = regexp.MustCompile(`-----BEGIN ((RSA |EC |DSA )?PRIVATE KEY)-----`)

// PubKeyFromPrivKey extracts the Public Key from one or more Private Keys.
// Faithful port of CyberChef's PubKeyFromPrivKey.mjs (jsrsasign KEYUTIL.getKey):
// each private-key PEM block is located, parsed, and its public key emitted as a
// PUBLIC KEY PEM block (CRLF line endings). RSA, EC and DSA keys are supported;
// DSA in PKCS#8 and EdDSA are rejected, as upstream.
type PubKeyFromPrivKey struct{}

// Meta returns the operation metadata.
func (PubKeyFromPrivKey) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Public Key from Private Key",
		Module:      "PublicKey",
		Description: "Extracts the Public Key from a Private Key.",
		InfoURL:     "https://en.wikipedia.org/wiki/PKCS_8",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PubKeyFromPrivKey) Args() []core.ArgDef { return nil }

// Run extracts each private key's public key, concatenating the PEM blocks.
func (PubKeyFromPrivKey) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	input := in.String()
	var out strings.Builder
	for _, loc := range privKeyHeaderRe.FindAllStringSubmatchIndex(input, -1) {
		label := input[loc[2]:loc[3]]
		footer := "-----END " + label + "-----"
		fi := strings.Index(input[loc[1]:], footer)
		if fi == -1 {
			return nil, errors.New("PEM footer '" + footer + "' not found")
		}
		privPEM := input[loc[0] : loc[1]+fi+len(footer)]
		pubPEM, err := privKeyPublicKeyPEM(label, privPEM)
		if err != nil {
			return nil, err
		}
		out.WriteString(pubPEM)
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// privKeyPublicKeyPEM parses one private-key PEM block (identified by its label)
// and returns the derived public key as a PUBLIC KEY PEM block.
func privKeyPublicKeyPEM(label, privPEM string) (string, error) {
	der, err := pemDER(privPEM)
	if err != nil {
		return "", err
	}
	switch label {
	case "RSA PRIVATE KEY":
		return rsaPKCS1PublicKey(der)
	case "EC PRIVATE KEY":
		return ecSEC1PublicKey(der)
	case "DSA PRIVATE KEY":
		return dsaTraditionalPublicKey(der)
	default: // "PRIVATE KEY" (PKCS#8)
		return pkcs8PublicKey(der)
	}
}

// rsaPKCS1PublicKey derives the public key from a PKCS#1 RSA private key.
func rsaPKCS1PublicKey(der []byte) (string, error) {
	key, err := x509.ParsePKCS1PrivateKey(der)
	if err != nil {
		return "", errUnsupportedKeyType(err)
	}
	return pkixPEM(&key.PublicKey)
}

// ecSEC1PublicKey derives the public key from a SEC1 EC private key.
func ecSEC1PublicKey(der []byte) (string, error) {
	key, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return "", errUnsupportedKeyType(err)
	}
	return pkixPEM(&key.PublicKey)
}

// pkcs8PublicKey derives the public key from a PKCS#8 private key, dispatching on
// the algorithm OID to reproduce jsrsasign KEYUTIL.getKey: it supports RSA and
// EC, rejects DSA with a dedicated message, and treats every other algorithm
// (EdDSA, X25519, …) as jsrsasign's unsupported-PKCS#8 error.
func pkcs8PublicKey(der []byte) (string, error) {
	switch pkcs8AlgOID(der) {
	case pkAlgDSA:
		return "", errors.New("DSA Private Key in PKCS#8 is not supported")
	case pkAlgRSA, pkAlgEC:
		return pkcs8RSAorECPublicKey(der)
	default:
		return "", errMalformedPKCS8
	}
}

// pkcs8RSAorECPublicKey extracts the public key from a PKCS#8 block already known
// (by OID) to hold an RSA or EC key.
func pkcs8RSAorECPublicKey(der []byte) (string, error) {
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return "", errUnsupportedKeyType(err)
	}
	if rsaKey, ok := key.(*rsa.PrivateKey); ok {
		return pkixPEM(&rsaKey.PublicKey)
	}
	return pkixPEM(&key.(*ecdsa.PrivateKey).PublicKey)
}

// pkcs8AlgOID returns the dotted algorithm OID of a PKCS#8 PrivateKeyInfo, or ""
// if it cannot be read. Only version + AlgorithmIdentifier are parsed; the
// AlgorithmIdentifier's optional parameters and the private-key octets are left
// as raw values.
func pkcs8AlgOID(der []byte) string {
	var info struct {
		Version   int
		Algorithm struct {
			Algorithm encasn1.ObjectIdentifier
			Params    encasn1.RawValue `asn1:"optional"`
		}
		PrivateKey encasn1.RawValue
	}
	if _, err := encasn1.Unmarshal(der, &info); err != nil {
		return ""
	}
	return info.Algorithm.Algorithm.String()
}

// dsaAlgOID is the id-dsa AlgorithmIdentifier (1.2.840.10040.4.1) used in a DSA
// SubjectPublicKeyInfo.
var dsaAlgOID = encasn1.ObjectIdentifier{1, 2, 840, 10040, 4, 1}

// dsaTraditionalPublicKey builds the SubjectPublicKeyInfo for an OpenSSL
// "traditional" DSA private key (SEQUENCE { version, p, q, g, y, x }), which
// carries the public value y directly. Marshalling positive big.Int values into
// this fixed structure cannot fail, so those errors are not checked.
func dsaTraditionalPublicKey(der []byte) (string, error) {
	var key struct {
		Version       int
		P, Q, G, Y, X *big.Int
	}
	if _, err := encasn1.Unmarshal(der, &key); err != nil {
		return "", errUnsupportedKeyType(err)
	}
	yDER, _ := encasn1.Marshal(key.Y)
	spki := struct {
		Algorithm struct {
			Algorithm encasn1.ObjectIdentifier
			Params    struct{ P, Q, G *big.Int }
		}
		PublicKey encasn1.BitString
	}{}
	spki.Algorithm.Algorithm = dsaAlgOID
	spki.Algorithm.Params.P = key.P
	spki.Algorithm.Params.Q = key.Q
	spki.Algorithm.Params.G = key.G
	spki.PublicKey = encasn1.BitString{Bytes: yDER, BitLength: len(yDER) * 8}
	out, _ := encasn1.Marshal(spki)
	return keyPEM("PUBLIC KEY", out), nil
}

// errMalformedPKCS8 reproduces jsrsasign's error for a PKCS#8 key whose
// algorithm it does not support (EdDSA), surfaced by the op as "Unsupported key
// type: <err>".
var errMalformedPKCS8 = errors.New("Unsupported key type: Error: malformed PKCS8 private key(code:004)") //nolint:staticcheck,revive // verbatim CyberChef text

// errUnsupportedKeyType wraps a parse failure as CyberChef's "Unsupported key
// type: <err>" message.
func errUnsupportedKeyType(err error) error {
	return errors.New("Unsupported key type: " + err.Error()) //nolint:staticcheck,revive // verbatim CyberChef text
}
