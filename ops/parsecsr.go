package ops

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseCSR{})
}

// ParseCSR renders a Certificate Signing Request in a human-readable form,
// ported from CyberChef's ParseCSR.mjs (jsrsasign CSRUtil.getParam). The request
// is walked directly over its DER hex using the shared asn1* helpers.
type ParseCSR struct{}

// Meta returns the operation metadata.
func (ParseCSR) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse CSR",
		Module:      "PublicKey",
		Description: "Parse Certificate Signing Request (CSR) for an X.509 certificate",
		InfoURL:     "https://wikipedia.org/wiki/Certificate_signing_request",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseCSR) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"PEM"}},
	}
}

// Run parses the CSR and renders its description.
func (ParseCSR) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if len(input) == 0 {
		return core.NewDish([]byte("No input"), core.TypeString), nil
	}
	der, err := pemDER(input)
	if err != nil {
		return nil, err
	}
	out, err := formatCSR(hex.EncodeToString(der))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// formatCSR walks a CertificationRequest DER hex and renders the CSR dump.
func formatCSR(h string) (string, error) {
	root, err := asn1GetChildIdx(h, 0)
	if err != nil || len(root) < 3 {
		return "", errors.New("malformed CSR")
	}
	cri, err := asn1GetChildIdx(h, root[0])
	if err != nil || len(cri) < 3 {
		return "", errors.New("malformed CSR")
	}

	subject, err := parseX500Name(h, cri[1])
	if err != nil {
		return "", err
	}
	pubKey, err := formatCSRPublicKey(h, cri[2])
	if err != nil {
		return "", err
	}
	sigAlg, err := algIDName(h, root[1])
	if err != nil {
		return "", err
	}
	sigHex := jsSubstrFrom(asn1GetV(h, root[2]), 2)
	sig, err := formatCSRSignature(sigAlg, sigHex)
	if err != nil {
		return "", err
	}

	// Attributes [0] may carry the extensionRequest.
	extSeq := -1
	for _, idx := range cri[3:] {
		if jsSubstr(h, idx, 2) == "a0" {
			extSeq = idx
		}
	}
	exts, err := formatRequestedExtensions(h, extSeq)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Subject\n%s\nPublic Key%s\nSignature%s\nRequested Extensions%s",
		formatDnObj(subject, 2), pubKey, sig, exts), nil
}

// formatCSRPublicKey ports ParseCSR.mjs formatSubjectPublicKey.
func formatCSRPublicKey(h string, spkiIdx int) (string, error) {
	kids, err := asn1GetChildIdx(h, spkiIdx)
	if err != nil || len(kids) < 2 {
		return "", errors.New("malformed SubjectPublicKeyInfo")
	}
	algKids, err := asn1GetChildIdx(h, kids[0])
	if err != nil || len(algKids) == 0 {
		return "", errors.New("malformed public key algorithm")
	}
	algOID := asn1OidHexToInt(asn1GetV(h, algKids[0]))
	bitContent := asn1GetVidx(h, kids[1]) + 2 // skip the BIT STRING unused-bits octet

	var out strings.Builder
	out.WriteString("\n")
	switch algOID {
	case "1.2.840.113549.1.1.1": // RSA
		nk, err := asn1GetChildIdx(h, bitContent)
		if err != nil || len(nk) < 2 {
			return "", errors.New("malformed RSA public key")
		}
		nHex := asn1GetV(h, nk[0])
		eVal, _ := new(big.Int).SetString(asn1GetV(h, nk[1]), 16)
		fmt.Fprintf(&out, "  Algorithm:      RSA\n  Length:         %d bits\n  Modulus:        %s\n  Exponent:       %s (0x%s)\n",
			bitLenOfHex(nHex), formatHexMulti(nHex), eVal.String(), utilsHex(int(eVal.Int64()), 2))
	case "1.2.840.10045.2.1": // EC
		curveOID := asn1OidHexToInt(asn1GetV(h, algKids[1]))
		short, asn1oid, keylen, ok := ecCurveInfo(curveOID)
		if !ok {
			return "", fmt.Errorf("unsupported curve %s", curveOID)
		}
		point := jsSubstrFrom(asn1GetV(h, kids[1]), 2)
		fmt.Fprintf(&out, "  Algorithm:      ECDSA\n  Length:         %d bits\n  Pub:            %s\n  ASN1 OID:       %s\n  NIST CURVE:     %s\n",
			keylen, formatHexMulti(point), asn1oid, short)
	case "1.2.840.10040.4.1": // DSA
		params, err := asn1GetChildIdx(h, algKids[1])
		if err != nil || len(params) < 3 {
			return "", errors.New("malformed DSA parameters")
		}
		p := asn1GetV(h, params[0])
		q := asn1GetV(h, params[1])
		g := asn1GetV(h, params[2])
		y := asn1GetV(h, bitContent)
		fmt.Fprintf(&out, "  Algorithm:      DSA\n  Length:         %d bits\n  Pub:            %s\n  P:              %s\n  Q:              %s\n  G:              %s\n",
			len(trimBigHex(p))*4, formatHexMulti(y), formatHexMulti(p), formatHexMulti(q), formatHexMulti(g))
	default:
		out.WriteString("unsupported public key algorithm\n")
	}
	return chopLast(out.String()), nil
}

// formatCSRSignature ports ParseCSR.mjs formatSignature.
func formatCSRSignature(sigAlg, sigHex string) (string, error) {
	var out strings.Builder
	out.WriteString("\n")
	fmt.Fprintf(&out, "  Algorithm:      %s\n", sigAlg)
	switch {
	case strings.Contains(strings.ToLower(sigAlg), "withdsa"):
		rs, err := asn1GetChildIdx(sigHex, 0)
		if err != nil || len(rs) < 2 {
			return "", errors.New("malformed DSA signature")
		}
		fmt.Fprintf(&out, "  Signature:\n      R:          %s\n      S:          %s\n",
			formatHexMulti(asn1GetV(sigHex, rs[0])), formatHexMulti(asn1GetV(sigHex, rs[1])))
	case strings.Contains(strings.ToLower(sigAlg), "withrsa"):
		fmt.Fprintf(&out, "  Signature:      %s\n", formatHexMulti(sigHex))
	default:
		fmt.Fprintf(&out, "  Signature:      %s\n", formatHexMulti(ensureHexPositive(sigHex)))
	}
	return chopLast(out.String()), nil
}

// formatRequestedExtensions ports ParseCSR.mjs formatRequestedExtensions: known
// extensions occupy fixed slots, unknown ones are appended.
func formatRequestedExtensions(h string, attrSeqIdx int) (string, error) {
	slots := make([]string, 4)
	var extra []string

	extIdxs, err := csrExtensionList(h, attrSeqIdx)
	if err != nil {
		return "", err
	}
	for _, extIdx := range extIdxs {
		kids, err := asn1GetChildIdx(h, extIdx)
		if err != nil || len(kids) < 2 {
			return "", errors.New("malformed extension")
		}
		oid := asn1OidHexToInt(asn1GetV(h, kids[0]))
		critical := len(kids) >= 3 && jsSubstr(h, kids[1], 2) == "01" && asn1GetV(h, kids[1]) == "ff"
		inner := asn1GetV(h, kids[len(kids)-1])
		crit := ""
		if critical {
			crit = " critical"
		}
		slot, text, err := describeRequestedExtension(oid, crit, inner)
		if err != nil {
			return "", err
		}
		if slot < 0 {
			extra = append(extra, text)
		} else {
			slots[slot] = text
		}
	}

	var out strings.Builder
	out.WriteString("\n")
	for _, s := range append(slots, extra...) {
		if s != "" {
			out.WriteString(s)
		}
	}
	return chopLast(out.String()), nil
}

// describeRequestedExtension renders one requested extension, returning its
// fixed slot index (0-3) or -1 to append it as an extra/unknown extension.
func describeRequestedExtension(oid, crit, inner string) (int, string, error) {
	switch oid {
	case "2.5.29.19": // basicConstraints
		return 0, "  Basic Constraints:" + crit + "\n" + indentParts(4, describeBasicConstraints(inner)), nil
	case "2.5.29.15": // keyUsage
		return 1, "  Key Usage:" + crit + "\n" + indentParts(4, describeKeyUsage(inner)), nil
	case "2.5.29.37": // extKeyUsage
		p, err := describeExtendedKeyUsage(inner)
		if err != nil {
			return 0, "", err
		}
		return 2, "  Extended Key Usage:" + crit + "\n" + indentParts(4, p), nil
	case "2.5.29.17": // subjectAltName
		p, err := describeSubjectAltName(inner)
		if err != nil {
			return 0, "", err
		}
		return 3, "  Subject Alternative Name:" + crit + "\n" + indentParts(4, p), nil
	default:
		return -1, "  " + extNameOrOID(oid) + ":" + crit + "\n" + indentParts(4, []string{"(unsuported extension)"}), nil
	}
}

// csrExtensionList returns the Extension SEQUENCE indices from the attributes
// [0] block's extensionRequest attribute (empty if none).
func csrExtensionList(h string, attrSeqIdx int) ([]int, error) {
	if attrSeqIdx < 0 {
		return nil, nil
	}
	attrs, err := asn1GetChildIdx(h, attrSeqIdx)
	if err != nil {
		return nil, err
	}
	for _, attr := range attrs {
		kids, err := asn1GetChildIdx(h, attr)
		if err != nil || len(kids) < 2 {
			continue
		}
		if asn1OidHexToInt(asn1GetV(h, kids[0])) != "1.2.840.113549.1.9.14" {
			continue
		}
		set, err := asn1GetChildIdx(h, kids[1]) // SET
		if err != nil || len(set) == 0 {
			return nil, errors.New("malformed extensionRequest")
		}
		return asn1GetChildIdx(h, set[0]) // SEQUENCE OF Extension
	}
	return nil, nil
}

// describeBasicConstraints ports ParseCSR.mjs describeBasicConstraints.
func describeBasicConstraints(inner string) []string {
	out := []string{"CA = false"}
	kids, err := asn1GetChildIdx(inner, 0)
	if err != nil {
		return out
	}
	for _, k := range kids {
		switch jsSubstr(inner, k, 2) {
		case "01": // cA BOOLEAN
			if asn1GetV(inner, k) == "ff" {
				out[0] = "CA = true"
			}
		case "02": // pathLenConstraint INTEGER
			n, _ := new(big.Int).SetString(asn1GetV(inner, k), 16)
			out = append(out, "PathLenConstraint = "+n.String())
		}
	}
	return out
}

// keyUsageNames lists the KeyUsage bit names in bit order.
var keyUsageNames = []string{
	"digitalSignature", "nonRepudiation", "keyEncipherment", "dataEncipherment",
	"keyAgreement", "keyCertSign", "cRLSign", "encipherOnly", "decipherOnly",
}

// keyUsageDisplay maps a KeyUsage bit name to its display string.
var keyUsageDisplay = map[string]string{
	"digitalSignature": "Digital Signature", "nonRepudiation": "Non-repudiation",
	"keyEncipherment": "Key encipherment", "dataEncipherment": "Data encipherment",
	"keyAgreement": "Key agreement", "keyCertSign": "Key certificate signing",
	"cRLSign": "CRL signing", "encipherOnly": "Encipher Only", "decipherOnly": "Decipher Only",
}

// describeKeyUsage ports ParseCSR.mjs describeKeyUsage.
func describeKeyUsage(inner string) []string {
	var out []string
	value := asn1GetV(inner, 0) // BIT STRING value: unused-bits octet + bits
	if len(value) >= 2 {
		unused, _ := jsParseInt(jsSubstr(value, 0, 2), 16)
		var bits strings.Builder
		for i := 2; i < len(value); i += 2 {
			b, _ := jsParseInt(jsSubstr(value, i, 2), 16)
			bits.WriteString(byteToBin(b))
		}
		bs := bits.String()
		meaningful := len(bs) - unused
		for i := 0; i < meaningful && i < len(keyUsageNames); i++ {
			if bs[i] == '1' {
				out = append(out, keyUsageDisplay[keyUsageNames[i]])
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "(none)")
	}
	return out
}

// ekuDisplay maps an EKU identifier (jsrsasign short name or OID) to a name.
var ekuDisplay = map[string]string{
	"serverAuth": "TLS Web Server Authentication", "clientAuth": "TLS Web Client Authentication",
	"codeSigning": "Code signing", "emailProtection": "E-mail Protection (S/MIME)",
	"timeStamping":           "Trusted Timestamping",
	"1.3.6.1.4.1.311.2.1.21": "Microsoft Individual Code Signing",
	"1.3.6.1.4.1.311.2.1.22": "Microsoft Commercial Code Signing",
	"1.3.6.1.4.1.311.10.3.1": "Microsoft Trust List Signing",
	"1.3.6.1.4.1.311.10.3.3": "Microsoft Server Gated Crypto",
	"1.3.6.1.4.1.311.10.3.4": "Microsoft Encrypted File System",
	"1.3.6.1.4.1.311.20.2.2": "Microsoft Smartcard Login",
	"2.16.840.1.113730.4.1":  "Netscape Server Gated Crypto",
}

// describeExtendedKeyUsage ports ParseCSR.mjs describeExtendedKeyUsage.
func describeExtendedKeyUsage(inner string) ([]string, error) {
	kids, err := asn1GetChildIdx(inner, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, k := range kids {
		eku := extNameOrOID(asn1OidHexToInt(asn1GetV(inner, k)))
		if name, ok := ekuDisplay[eku]; ok {
			out = append(out, name)
		} else {
			out = append(out, eku)
		}
	}
	if len(out) == 0 {
		out = append(out, "(none)")
	}
	return out, nil
}

// describeSubjectAltName ports ParseCSR.mjs describeSubjectAlternativeName.
func describeSubjectAltName(inner string) ([]string, error) {
	names, err := parseGeneralNames(inner, 0)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range names {
		switch n.kind {
		case "rfc822":
			out = append(out, "EMAIL: "+n.value)
		case "dns":
			out = append(out, "DNS: "+n.value)
		case "uri":
			out = append(out, "URI: "+n.value)
		case "ip":
			out = append(out, "IP: "+n.value)
		case "dn":
			out = append(out, "DIR: "+n.dn.str())
		case "other":
			out = append(out, "Other: "+n.oid+"::"+n.value)
		}
	}
	return out, nil
}

// --- CSR formatting helpers ---------------------------------------------------

// formatHexMulti colon-formats hex across 48-char lines with an 18-space
// continuation indent (ParseCSR.mjs formatHexOntoMultiLine).
func formatHexMulti(h string) string {
	return splitEvery(colonHex(h), 48, "\n"+strings.Repeat(" ", 18))
}

// indentParts joins parts, prefixing each with n spaces and a trailing newline
// (ParseCSR.mjs indent()).
func indentParts(n int, parts []string) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.Join(parts, "\n"+pad) + "\n"
}

// ensureHexPositive pads to even length and prepends 00 when the high bit is set.
func ensureHexPositive(h string) string {
	if len(h)%2 != 0 {
		return "0" + h
	}
	if len(h) >= 2 {
		if b, _ := jsParseInt(jsSubstr(h, 0, 2), 16); b&128 != 0 {
			return "00" + h
		}
	}
	return h
}

// bitLenOfHex returns the bit length of the non-negative integer whose value
// octets are h.
func bitLenOfHex(h string) int {
	n, _ := new(big.Int).SetString(h, 16)
	if n == nil {
		return 0
	}
	return n.BitLen()
}

// trimBigHex returns the minimal hex of the integer whose octets are h (dropping
// the DER sign-padding octet), used for DSA key-length computation.
func trimBigHex(h string) string {
	n, _ := new(big.Int).SetString(h, 16)
	if n == nil {
		return ""
	}
	return n.Text(16)
}

// extNameOrOID resolves an OID to its jsrsasign name, or returns the dotted OID.
func extNameOrOID(oid string) string {
	if name := oid2name(oid); name != "" {
		return name
	}
	return oid
}

// ecCurveInfo returns the NIST short name, ASN.1 OID display name, and key length
// (bits) for an EC named-curve OID.
func ecCurveInfo(oid string) (short, asn1oid string, keylen int, ok bool) {
	switch oid {
	case "1.2.840.10045.3.1.7":
		return "P-256", "secp256r1", 256, true
	case "1.3.132.0.34":
		return "P-384", "secp384r1", 384, true
	case "1.3.132.0.35":
		return "P-521", "secp521r1", 521, true
	}
	return "", "", 0, false
}
