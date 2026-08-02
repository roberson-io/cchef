package ops

import (
	"crypto/md5"  // #nosec G501 -- MD5 fingerprint is part of the certificate report, not security
	"crypto/sha1" // #nosec G505 -- SHA1 fingerprint is part of the certificate report, not security
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(ParseX509Certificate{})
}

// ParseX509Certificate renders an X.509 certificate in an openssl-like human
// readable form. The certificate is walked directly over its DER hex.
type ParseX509Certificate struct{}

// Meta returns the operation metadata.
func (ParseX509Certificate) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse X.509 certificate",
		Module:      "PublicKey",
		Description: "X.509 is an ITU-T standard for a public key infrastructure (PKI) and Privilege Management Infrastructure (PMI). It is commonly involved with SSL/TLS security.<br><br>This operation displays the contents of a certificate in a human readable format, similar to the openssl command line tool.<br><br>Tags: X509, server hello, handshake",
		InfoURL:     "https://wikipedia.org/wiki/X.509",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseX509Certificate) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"PEM", "DER Hex", "Base64", "Raw"}},
	}
}

// Run parses the certificate and renders its description.
func (ParseX509Certificate) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if len(input) == 0 {
		return core.NewDish([]byte("No input"), core.TypeString), nil
	}
	hexStr, err := x509InputToHex(input, args[0].(string))
	if err != nil {
		return nil, err
	}
	out, err := formatCertificate(hexStr)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// certView holds the parsed positions/values of a Certificate needed for
// formatting.
type certView struct {
	version                int
	serialHex, sigAlgField string
	issuer, subject        x509Name
	notBefore, notAfter    string
	spkiIdx, extSeqIdx     int
	sigAlgName, signatureV string
}

// parseCertificate walks the Certificate DER hex into a certView.
func parseCertificate(h string) (certView, error) {
	var v certView
	root, err := asn1GetChildIdx(h, 0)
	if err != nil || len(root) < 3 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	tbs, err := asn1GetChildIdx(h, root[0])
	if err != nil || len(tbs) < 6 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // verbatim CyberChef text
	}

	off := 0
	if v.version, off, err = certVersion(h, tbs); err != nil {
		return v, err
	}
	if len(tbs) < off+7 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // verbatim CyberChef text
	}

	v.serialHex = asn1GetV(h, tbs[off])
	if v.sigAlgField, err = algIDName(h, tbs[off+1]); err != nil {
		return v, err
	}
	if v.issuer, err = parseX500Name(h, tbs[off+2]); err != nil {
		return v, err
	}
	if v.notBefore, v.notAfter, err = parseValidity(h, tbs[off+3]); err != nil {
		return v, err
	}
	if v.subject, err = parseX500Name(h, tbs[off+4]); err != nil {
		return v, err
	}
	v.spkiIdx = tbs[off+5]
	v.extSeqIdx = findCertExtSeq(h, tbs[off+6:])

	if v.sigAlgName, err = algIDName(h, root[1]); err != nil {
		return v, err
	}
	v.signatureV = jsSubstrFrom(asn1GetV(h, root[2]), 2)
	return v, nil
}

// certVersion reads the optional [0] version, returning the version number and
// the index offset (1 if present, else 0) for the fields that follow.
func certVersion(h string, tbs []int) (version, off int, err error) {
	if jsSubstr(h, tbs[0], 2) != "a0" {
		return 1, 0, nil
	}
	vkids, err := asn1GetChildIdx(h, tbs[0])
	if err != nil || len(vkids) == 0 {
		return 0, 0, errors.New("malformed version")
	}
	n, _ := new(big.Int).SetString(asn1GetV(h, vkids[0]), 16)
	return int(n.Int64()) + 1, 1, nil
}

// findCertExtSeq returns the index of the extensions SEQUENCE inside the [3]
// EXPLICIT wrapper among the given TBS tail indices, or -1 if absent.
func findCertExtSeq(h string, idxs []int) int {
	ext := -1
	for _, idx := range idxs {
		if jsSubstr(h, idx, 2) == "a3" {
			kids, err := asn1GetChildIdx(h, idx)
			if err == nil && len(kids) > 0 {
				ext = kids[0]
			}
		}
	}
	return ext
}

// formatCertificate walks a Certificate DER hex and renders the report.
func formatCertificate(h string) (string, error) {
	v, err := parseCertificate(h)
	if err != nil {
		return "", err
	}
	pkStr, err := formatCertPublicKey(h, v.spkiIdx)
	if err != nil {
		return "", err
	}
	extensions, err := formatCertExtensions(h, v.extSeqIdx)
	if err != nil {
		return "", err
	}

	der, _ := hex.DecodeString(h)
	serialDec, _ := new(big.Int).SetString(v.serialHex, 16)

	var out strings.Builder
	fmt.Fprintf(&out, "Version:          %d (0x%s)\n", v.version, utilsHex(v.version-1, 2))
	fmt.Fprintf(&out, "Serial number:    %s (0x%s)\n", serialDec.String(), v.serialHex)
	fmt.Fprintf(&out, "Algorithm ID:     %s\n", v.sigAlgField)
	fmt.Fprintf(&out, "Validity\n  Not Before:     %s (dd-mm-yyyy hh:mm:ss) (%s)\n", formatCertDate(v.notBefore), v.notBefore)
	fmt.Fprintf(&out, "  Not After:      %s (dd-mm-yyyy hh:mm:ss) (%s)\n", formatCertDate(v.notAfter), v.notAfter)
	fmt.Fprintf(&out, "Issuer\n%s\n", formatDnObj(v.issuer, 2))
	fmt.Fprintf(&out, "Subject\n%s\n", formatDnObj(v.subject, 2))
	fmt.Fprintf(&out, "Fingerprints\n  MD5:            %x\n  SHA1:           %x\n  SHA256:         %x\n",
		md5.Sum(der), sha1.Sum(der), sha256.Sum256(der)) // #nosec G401 -- see import note
	fmt.Fprintf(&out, "Public Key\n%s\n", pkStr)
	fmt.Fprintf(&out, "Certificate Signature\n  Algorithm:      %s\n%s\n\nExtensions\n%s",
		v.sigAlgName, formatCertSignature(v.signatureV), extensions)
	return out.String(), nil
}

// parseValidity returns the raw notBefore/notAfter time strings.
func parseValidity(h string, seqIdx int) (string, string, error) {
	kids, err := asn1GetChildIdx(h, seqIdx)
	if err != nil || len(kids) < 2 {
		return "", "", errors.New("malformed validity")
	}
	return hextoutf8(asn1GetV(h, kids[0])), hextoutf8(asn1GetV(h, kids[1])), nil
}

// formatCertDate ports ParseX509Certificate.mjs formatDate.
func formatCertDate(d string) string {
	if len(d) == 13 { // UTCTime YYMMDDHHMMSSZ: prepend century
		if d[0] < '5' {
			d = "20" + d
		} else {
			d = "19" + d
		}
	}
	if len(d) < 14 {
		return d
	}
	return string(d[6]) + string(d[7]) + "/" + string(d[4]) + string(d[5]) + "/" +
		d[0:4] + " " + string(d[8]) + string(d[9]) + ":" + string(d[10]) + string(d[11]) + ":" +
		string(d[12]) + string(d[13])
}

// formatCertPublicKey renders the Public Key section (ports the pkFields logic).
func formatCertPublicKey(h string, spkiIdx int) (string, error) {
	kids, err := asn1GetChildIdx(h, spkiIdx)
	if err != nil || len(kids) < 2 {
		return "", errors.New("malformed SubjectPublicKeyInfo")
	}
	algKids, err := asn1GetChildIdx(h, kids[0])
	if err != nil || len(algKids) == 0 {
		return "", errors.New("malformed public key algorithm")
	}
	algOID := asn1OidHexToInt(asn1GetV(h, algKids[0]))
	bitContent := asn1GetVidx(h, kids[1]) + 2

	var fields [][2]string
	switch algOID {
	case "1.2.840.113549.1.1.1": // RSA
		fields = append(fields, [2]string{"Algorithm", "RSA"})
		nk, err := asn1GetChildIdx(h, bitContent)
		if err != nil || len(nk) < 2 {
			return "", errors.New("malformed RSA public key")
		}
		nHex := trimBigHex(asn1GetV(h, nk[0]))
		eVal, _ := new(big.Int).SetString(asn1GetV(h, nk[1]), 16)
		fields = append(fields,
			[2]string{"Length", fmt.Sprintf("%d bits", bitLenOfHex(nHex))},
			[2]string{"Modulus", certByteStr(nHex)},
			[2]string{"Exponent", fmt.Sprintf("%s (0x%s)", eVal.String(), eVal.Text(16))})
	case "1.2.840.10045.2.1": // EC
		point := jsSubstrFrom(asn1GetV(h, kids[1]), 2)
		curveOID := asn1OidHexToInt(asn1GetV(h, algKids[1]))
		_, asn1oid, _, _ := ecCurveInfo(curveOID)
		pn, _ := new(big.Int).SetString(point, 16)
		fields = append(fields,
			[2]string{"Algorithm", "EC"},
			[2]string{"Curve Name", asn1oid},
			[2]string{"Length", fmt.Sprintf("%d bits", (pn.BitLen()-3)/2)},
			[2]string{"pub", certByteStr(point)})
	default:
		fields = append(fields, [2]string{"Error", "Unknown Public Key type"})
	}

	var out strings.Builder
	for _, f := range fields {
		out.WriteString("  " + f[0] + ":" + strings.Repeat(" ", 15-len(f[0])) + f[1] + "\n")
	}
	return chopLast(out.String()), nil
}

// certByteStr colon-formats hex into 16-byte (48-char) lines with an 18-space
// continuation indent (CyberChef lib/PublicKey.mjs formatByteStr).
func certByteStr(h string) string {
	return splitEvery(colonHex(h), 48, "\n"+strings.Repeat(" ", 18))
}

// formatCertSignature ports the signature breakout logic. RSA signatures are
// dumped as one byte string; SEQUENCE (ECDSA/DSA) signatures break into r and s.
func formatCertSignature(sigHex string) string {
	if jsSubstr(sigHex, 0, 2) == "30" { // SEQUENCE => ECDSA/DSA
		return "  r:              " + certByteStr(asn1GetV(sigHex, 4)) +
			"\n  s:              " + certByteStr(asn1GetV(sigHex, 48))
	}
	return "  Signature:      " + certByteStr(sigHex)
}

// formatCertExtensions ports jsrsasign X509.getInfo's extension section (the
// text ParseX509Certificate.mjs slices out between "X509v3 Extensions:" and
// "signature").
func formatCertExtensions(h string, seqIdx int) (string, error) {
	if seqIdx < 0 {
		return "", nil
	}
	idxs, err := asn1GetChildIdx(h, seqIdx)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, idx := range idxs {
		kids, err := asn1GetChildIdx(h, idx)
		if err != nil || len(kids) < 2 {
			return "", errors.New("malformed extension")
		}
		oid := asn1OidHexToInt(asn1GetV(h, kids[0]))
		critical := len(kids) >= 3 && jsSubstr(h, kids[1], 2) == "01" && asn1GetV(h, kids[1]) == "ff"
		value := asn1GetV(h, kids[len(kids)-1])
		name := oid2name(oid)
		if name == "" {
			name = oid
		}
		crit := ""
		if critical {
			crit = "CRITICAL"
		}
		fmt.Fprintf(&out, "  %s %s:\n", name, crit)
		body, err := formatCertExtensionBody(name, value)
		if err != nil {
			return "", err
		}
		out.WriteString(body)
	}
	return out.String(), nil
}

// formatCertExtensionBody renders one extension's indented content, matching
// jsrsasign getInfo. Extension types getInfo does not special-case emit no body.
func formatCertExtensionBody(name, value string) (string, error) {
	switch name {
	case "basicConstraints":
		return certExtBasicConstraints(value), nil
	case "keyUsage":
		return "    " + strings.Join(keyUsageIdentifiers(value), ",") + "\n", nil
	case "subjectKeyIdentifier":
		return "    " + asn1GetV(value, 0) + "\n", nil
	case "authorityKeyIdentifier":
		return certExtAuthorityKeyID(value), nil
	case "extKeyUsage":
		ekus, err := extKeyUsageOIDs(value)
		if err != nil {
			return "", err
		}
		return "    " + strings.Join(ekus, ", ") + "\n", nil
	case "subjectAltName":
		return certExtSubjectAltName(value)
	case "cRLDistributionPoints":
		return certExtCRLDistPoints(value)
	case "authorityInfoAccess":
		return certExtAuthorityInfoAccess(value)
	case "certificatePolicies":
		return certExtCertificatePolicies(value)
	default:
		return "", nil
	}
}

// certExtBasicConstraints ports getExtBasicConstraints formatting.
func certExtBasicConstraints(value string) string {
	kids, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return "    {}\n"
	}
	ca := false
	pathLen := ""
	for _, k := range kids {
		switch jsSubstr(value, k, 2) {
		case "01":
			if asn1GetV(value, k) == "ff" {
				ca = true
			}
		case "02":
			n, _ := new(big.Int).SetString(asn1GetV(value, k), 16)
			pathLen = n.String()
		}
	}
	if !ca {
		return "    {}\n"
	}
	if pathLen != "" {
		return "    cA=true, pathLen=" + pathLen + "\n"
	}
	return "    cA=true\n"
}

// keyUsageIdentifiers returns the jsrsasign KeyUsage bit identifier names set in
// the extension value.
func keyUsageIdentifiers(value string) []string {
	var out []string
	v := asn1GetV(value, 0)
	if len(v) < 2 {
		return out
	}
	unused, _ := jsnum.ParseInt(jsSubstr(v, 0, 2), 16)
	var bits strings.Builder
	for i := 2; i < len(v); i += 2 {
		b, _ := jsnum.ParseInt(jsSubstr(v, i, 2), 16)
		bits.WriteString(byteToBin(b))
	}
	bs := bits.String()
	meaningful := len(bs) - unused
	for i := 0; i < meaningful && i < len(keyUsageNames); i++ {
		if bs[i] == '1' {
			out = append(out, keyUsageNames[i])
		}
	}
	return out
}

// certExtAuthorityKeyID ports getExtAuthorityKeyIdentifier's getInfo output
// (only the key identifier is shown).
func certExtAuthorityKeyID(value string) string {
	kids, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return ""
	}
	for _, k := range kids {
		if jsSubstr(value, k, 2) == "80" {
			return "    kid=" + asn1GetV(value, k) + "\n"
		}
	}
	return ""
}

// extKeyUsageOIDs resolves the EKU OIDs to jsrsasign names or dotted OIDs.
func extKeyUsageOIDs(value string) ([]string, error) {
	kids, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(kids))
	for _, k := range kids {
		out = append(out, extNameOrOID(asn1OidHexToInt(asn1GetV(value, k))))
	}
	return out, nil
}

// certExtSubjectAltName ports the getInfo A() helper.
func certExtSubjectAltName(value string) (string, error) {
	names, err := parseGeneralNames(value, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, n := range names {
		switch n.kind {
		case "dn":
			out.WriteString("    dn: " + n.dn.str() + "\n")
		case "ip":
			out.WriteString("    ip: " + n.value + "\n")
		case "rfc822":
			out.WriteString("    rfc822: " + n.value + "\n")
		case "dns":
			out.WriteString("    dns: " + n.value + "\n")
		case "uri":
			out.WriteString("    uri: " + n.value + "\n")
		case "other":
			out.WriteString("    other: " + n.oid + "=" + n.value + "\n")
		}
	}
	return out.String(), nil
}

// certExtCRLDistPoints ports the getInfo K() helper (fullName URIs only).
func certExtCRLDistPoints(value string) (string, error) {
	dps, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, dp := range dps {
		full, err := distributionPointFullNames(value, dp)
		if err != nil {
			continue
		}
		for _, gn := range full {
			if gn.kind == "uri" {
				out.WriteString("    " + gn.value + "\n")
				break
			}
		}
	}
	return out.String(), nil
}

// certExtAuthorityInfoAccess ports the getInfo I() helper.
func certExtAuthorityInfoAccess(value string) (string, error) {
	entries, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, e := range entries {
		kids, err := asn1GetChildIdx(value, e)
		if err != nil || len(kids) < 2 {
			continue
		}
		method := asn1OidHexToInt(asn1GetV(value, kids[0]))
		loc := hextoutf8(asn1GetV(value, kids[1]))
		switch method {
		case "1.3.6.1.5.5.7.48.2": // caIssuers
			out.WriteString("    caissuer: " + loc + "\n")
		case "1.3.6.1.5.5.7.48.1": // ocsp
			out.WriteString("    ocsp: " + loc + "\n")
		}
	}
	return out.String(), nil
}

// certExtCertificatePolicies ports the getInfo H() helper.
func certExtCertificatePolicies(value string) (string, error) {
	policies, err := asn1GetChildIdx(value, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, p := range policies {
		kids, err := asn1GetChildIdx(value, p)
		if err != nil || len(kids) == 0 {
			continue
		}
		out.WriteString("    policy oid: " + asn1OidHexToInt(asn1GetV(value, kids[0])) + "\n")
	}
	return out.String(), nil
}
