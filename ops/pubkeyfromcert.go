package ops

import (
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(PubKeyFromCert{})
}

// Public-key algorithm OIDs jsrsasign supports for extraction (RSA / EC / DSA).
// Any other algorithm (e.g. Ed25519, Ed448) is rejected as unsupported.
const (
	pkAlgRSA = "1.2.840.113549.1.1.1"
	pkAlgEC  = "1.2.840.10045.2.1"
	pkAlgDSA = "1.2.840.10040.4.1"
)

// errUnsupportedPubKeyType mirrors the OperationError jsrsasign's getPublicKey
// failure produces in CyberChef.
var errUnsupportedPubKeyType = errors.New("Unsupported public key type") //nolint:staticcheck,revive // verbatim CyberChef text

const (
	certHeader = "-----BEGIN CERTIFICATE-----"
	certFooter = "-----END CERTIFICATE-----"
)

var certHeaderRe = regexp.MustCompile(certHeader)

// PubKeyFromCert extracts the Public Key from one or more X.509 certificates.
// Faithful port of CyberChef's PubKeyFromCert.mjs: each PEM certificate block is
// located, its SubjectPublicKeyInfo extracted, and emitted as a PUBLIC KEY PEM
// block (CRLF line endings). RSA, EC and DSA keys are supported, as upstream.
type PubKeyFromCert struct{}

// Meta returns the operation metadata.
func (PubKeyFromCert) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Public Key from Certificate",
		Module:      "PublicKey",
		Description: "Extracts the Public Key from a Certificate.",
		InfoURL:     "https://en.wikipedia.org/wiki/X.509",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PubKeyFromCert) Args() []core.ArgDef { return nil }

// Run extracts each certificate's public key, concatenating the PEM blocks.
func (PubKeyFromCert) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	input := in.String()
	var out strings.Builder
	for _, loc := range certHeaderRe.FindAllStringIndex(input, -1) {
		fi := strings.Index(input[loc[1]:], certFooter)
		if fi == -1 {
			return nil, errors.New("PEM footer '" + certFooter + "' not found")
		}
		certPEM := input[loc[0] : loc[1]+fi+len(certFooter)]
		pubPEM, err := certPublicKeyPEM(certPEM)
		if err != nil {
			return nil, err
		}
		out.WriteString(pubPEM)
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// certPublicKeyPEM extracts the SubjectPublicKeyInfo from a certificate PEM and
// re-emits it as a PUBLIC KEY PEM block, rejecting unsupported key algorithms.
func certPublicKeyPEM(certPEM string) (string, error) {
	der, err := pemDER(certPEM)
	if err != nil {
		return "", errUnsupportedPubKeyType
	}
	h := hex.EncodeToString(der)
	spki, ok := certSPKI(h)
	if !ok || !supportedPubKeyOID(spki) {
		return "", errUnsupportedPubKeyType
	}
	// spki is an even-length substring of lowercase hex, so it always decodes.
	spkiDER, _ := hex.DecodeString(spki)
	return keyPEM("PUBLIC KEY", spkiDER), nil
}

// certSPKI returns the SubjectPublicKeyInfo TLV (hex) of the certificate whose
// DER hex is h, matching jsrsasign X509.getPublicKeyHex (TBS field index 6,
// after the optional [0] version).
func certSPKI(h string) (string, bool) {
	root, err := asn1GetChildIdx(h, 0)
	if err != nil || len(root) < 1 {
		return "", false
	}
	tbs, err := asn1GetChildIdx(h, root[0])
	if err != nil {
		return "", false
	}
	off := 0
	if len(tbs) > 0 && jsSubstr(h, tbs[0], 2) == "a0" {
		off = 1
	}
	if len(tbs) < off+6 {
		return "", false
	}
	// asn1GetChildIdx only returns a child whose TLV length is valid and within
	// bounds, so the SPKI field's length (in hex chars) is always usable here.
	idx := tbs[off+5]
	return jsSubstr(h, idx, asn1GetTLVblen(h, idx)), true
}

// supportedPubKeyOID reports whether the SPKI's algorithm OID is one jsrsasign
// can load (RSA, EC or DSA).
func supportedPubKeyOID(spki string) bool {
	kids, err := asn1GetChildIdx(spki, 0)
	if err != nil || len(kids) < 1 {
		return false
	}
	algKids, err := asn1GetChildIdx(spki, kids[0])
	if err != nil || len(algKids) < 1 {
		return false
	}
	switch asn1OidHexToInt(asn1GetV(spki, algKids[0])) {
	case pkAlgRSA, pkAlgEC, pkAlgDSA:
		return true
	}
	return false
}
