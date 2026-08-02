package ops

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(ParseX509CRL{})
}

// ParseX509CRL parses a Certificate Revocation List into a human-readable dump.
// The container is walked directly over its DER hex using the shared asn1*
// helpers.
type ParseX509CRL struct{}

// Meta returns the operation metadata.
func (ParseX509CRL) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse X.509 CRL",
		Module:      "PublicKey",
		Description: "Parse Certificate Revocation List (CRL)",
		InfoURL:     "https://wikipedia.org/wiki/Certificate_revocation_list",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseX509CRL) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"PEM", "DER Hex", "Base64", "Raw"}},
	}
}

// Run parses the CRL and renders the openssl-like description.
func (ParseX509CRL) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if len(input) == 0 {
		return core.NewDish([]byte("No input"), core.TypeString), nil
	}
	hexStr, err := x509InputToHex(input, args[0].(string))
	if err != nil {
		return nil, err
	}
	out, err := formatCRL(hexStr)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// crlView holds the parsed positions/values of a CertificateList needed for
// formatting.
type crlView struct {
	hex        string
	version2   bool
	sigAlgName string
	issuer     x509Name
	thisUpdate string
	nextUpdate string
	extSeqIdx  int // index of the crlExtensions SEQUENCE, or -1
	revSeqIdx  int // index of the revokedCertificates SEQUENCE, or -1
	signatureV string
}

// parseCRL walks the CertificateList DER hex into a crlView.
func parseCRL(h string) (crlView, error) {
	v := crlView{hex: h, extSeqIdx: -1, revSeqIdx: -1}
	root, err := asn1GetChildIdx(h, 0)
	if err != nil || len(root) < 3 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // CyberChef-style text
	}
	tbs, err := asn1GetChildIdx(h, root[0])
	if err != nil || len(tbs) < 3 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // CyberChef-style text
	}

	// Optional version INTEGER precedes the signature AlgorithmIdentifier.
	off := 0
	if jsSubstr(h, tbs[0], 2) == "02" {
		v.version2 = true
		off = 1
	}
	if len(tbs) < off+3 {
		return v, errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // CyberChef-style text
	}
	if v.sigAlgName, err = algIDName(h, tbs[off]); err != nil {
		return v, err
	}
	if v.issuer, err = parseX500Name(h, tbs[off+1]); err != nil {
		return v, err
	}
	v.thisUpdate = hextoutf8(asn1GetV(h, tbs[off+2]))

	// Remaining optional fields: nextUpdate (Time), revokedCertificates
	// (SEQUENCE), crlExtensions ([0]).
	for _, idx := range tbs[off+3:] {
		switch jsSubstr(h, idx, 2) {
		case "17", "18": // UTCTime / GeneralizedTime
			v.nextUpdate = hextoutf8(asn1GetV(h, idx))
		case "30": // revokedCertificates
			v.revSeqIdx = idx
		case "a0": // [0] crlExtensions
			kids, err := asn1GetChildIdx(h, idx)
			if err == nil && len(kids) > 0 {
				v.extSeqIdx = kids[0]
			}
		}
	}

	// signatureValue BIT STRING: drop the leading "unused bits" octet.
	v.signatureV = jsSubstrFrom(asn1GetV(h, root[2]), 2)
	return v, nil
}

// formatCRL renders a crlView as the CyberChef CRL description.
func formatCRL(h string) (string, error) {
	v, err := parseCRL(h)
	if err != nil {
		return "", err
	}

	version := "1 (0x0)"
	if v.version2 {
		version = "2 (0x1)"
	}
	thisUpdate, err := generalizedDateTimeToUTC(v.thisUpdate)
	if err != nil {
		return "", err
	}
	nextUpdate, err := generalizedDateTimeToUTC(v.nextUpdate)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	fmt.Fprintf(&out, "Certificate Revocation List (CRL):\n    Version: %s\n    Signature Algorithm: %s\n    Issuer:\n%s\n    Last Update: %s\n    Next Update: %s\n",
		version, v.sigAlgName, formatDnObj(v.issuer, 8), thisUpdate, nextUpdate)

	if v.extSeqIdx >= 0 {
		exts, err := formatCRLExtensions(h, v.extSeqIdx, 8)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, "\tCRL extensions:\n%s\n", exts)
	}

	revoked, err := formatRevokedCertificates(h, v.revSeqIdx, 4)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(&out, "Revoked Certificates:\n%s\nSignature Value:\n%s", revoked, formatCRLSignature(v.signatureV, 8))
	return out.String(), nil
}

// crlExtension is a parsed CRL extension: its jsrsasign extname and value hex.
type crlExtension struct {
	extname  string
	valueHex string
}

// parseExtensions parses the Extensions SEQUENCE at seqIdx into extname/value
// pairs, resolving OIDs to jsrsasign extension names.
func parseExtensions(h string, seqIdx int) ([]crlExtension, error) {
	idxs, err := asn1GetChildIdx(h, seqIdx)
	if err != nil {
		return nil, err
	}
	var out []crlExtension
	for _, idx := range idxs {
		kids, err := asn1GetChildIdx(h, idx)
		if err != nil || len(kids) < 2 {
			return nil, errors.New("malformed extension")
		}
		oid := asn1OidHexToInt(asn1GetV(h, kids[0]))
		// extnValue is the last child (OCTET STRING); a BOOLEAN critical flag
		// may sit between the OID and it.
		valueOctet := kids[len(kids)-1]
		out = append(out, crlExtension{extname: extName(oid), valueHex: asn1GetV(h, valueOctet)})
	}
	return out, nil
}

// extName maps an extension OID to jsrsasign's extension name, or the dotted OID.
func extName(oid string) string {
	switch oid {
	case "2.5.29.35":
		return "authorityKeyIdentifier"
	case "2.5.29.31":
		return "cRLDistributionPoints"
	case "2.5.29.20":
		return "cRLNumber"
	case "2.5.29.18":
		return "issuerAltName"
	}
	return oid
}

// formatCRLExtensions ports ParseX509CRL.mjs formatCRLExtensions.
func formatCRLExtensions(h string, seqIdx, indent int) (string, error) {
	exts, err := parseExtensions(h, seqIdx)
	if err != nil {
		return "", err
	}
	if len(exts) == 0 {
		return indentLines("No CRL extensions.", indent), nil
	}
	sort.SliceStable(exts, func(i, j int) bool { return exts[i].extname < exts[j].extname })

	var out strings.Builder
	for _, ext := range exts {
		s, err := formatCRLExtension(h, ext)
		if err != nil {
			return "", err
		}
		out.WriteString(s)
	}
	return indentLines(chopLast(out.String()), indent), nil
}

// formatCRLExtension formats one CRL extension.
func formatCRLExtension(h string, ext crlExtension) (string, error) {
	switch ext.extname {
	case "authorityKeyIdentifier":
		return formatAuthorityKeyID(h, ext.valueHex)
	case "cRLDistributionPoints":
		return formatCRLDistributionPoints(h, ext.valueHex)
	case "cRLNumber":
		return "X509v3 CRL Number:\n\t" + crlNumberHex(ext.valueHex) + "\n", nil
	case "issuerAltName":
		names, err := parseGeneralNames(ext.valueHex, 0)
		if err != nil {
			return "", err
		}
		return "X509v3 Issuer Alternative Name:\n" + formatGeneralNames(names, 4) + "\n", nil
	default:
		return ext.extname + ":\n\tUnsupported CRL extension. Try openssl CLI.\n", nil
	}
}

// crlNumberHex returns the uppercase hex of the cRLNumber INTEGER value.
func crlNumberHex(valueHex string) string {
	return strings.ToUpper(asn1GetV(valueHex, 0))
}

// formatAuthorityKeyID formats a 2.5.29.35 extension value.
func formatAuthorityKeyID(_ string, valueHex string) (string, error) {
	kids, err := asn1GetChildIdx(valueHex, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("X509v3 Authority Key Identifier:\n")
	for _, idx := range kids {
		switch jsSubstr(valueHex, idx, 2) {
		case "80": // [0] keyIdentifier
			fmt.Fprintf(&out, "\tkeyid:%s\n", colonHex(strings.ToUpper(asn1GetV(valueHex, idx))))
		case "a1": // [1] authorityCertIssuer (GeneralNames)
			gnames, err := asn1GetChildIdx(valueHex, idx)
			if err != nil || len(gnames) == 0 {
				continue
			}
			gn, err := parseGeneralName(valueHex, gnames[0])
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&out, "\tDirName:%s\n", gn.dn.str())
		case "82": // [2] authorityCertSerialNumber
			fmt.Fprintf(&out, "\tserial:%s\n", colonHex(strings.ToUpper(asn1GetV(valueHex, idx))))
		}
	}
	return out.String(), nil
}

// formatCRLDistributionPoints formats a 2.5.29.31 extension value.
func formatCRLDistributionPoints(_ string, valueHex string) (string, error) {
	dps, err := asn1GetChildIdx(valueHex, 0)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString("X509v3 CRL Distribution Points:\n")
	for _, dp := range dps {
		full, err := distributionPointFullNames(valueHex, dp)
		if err != nil {
			return "", err
		}
		out.WriteString(indentLines("Full Name:\n"+formatGeneralNames(full, 4), 4) + "\n")
	}
	return out.String(), nil
}

// distributionPointFullNames extracts the fullName GeneralNames of a
// DistributionPoint (distributionPoint [0] → fullName [0]).
func distributionPointFullNames(h string, dpIdx int) ([]x509GeneralName, error) {
	kids, err := asn1GetChildIdx(h, dpIdx)
	if err != nil || len(kids) == 0 {
		return nil, errors.New("malformed distribution point")
	}
	// distributionPoint [0]
	dpName, err := asn1GetChildIdx(h, kids[0])
	if err != nil || len(dpName) == 0 {
		return nil, errors.New("malformed distribution point name")
	}
	// fullName [0] holds the GeneralNames directly.
	return parseGeneralNamesTagged(h, dpName[0])
}

// formatGeneralNames ports ParseX509CRL.mjs formatGeneralNames.
func formatGeneralNames(names []x509GeneralName, indent int) string {
	var out strings.Builder
	for _, n := range names {
		switch n.kind {
		case "ip":
			fmt.Fprintf(&out, "IP:%s\n", n.value)
		case "dns":
			fmt.Fprintf(&out, "DNS:%s\n", n.value)
		case "uri":
			fmt.Fprintf(&out, "URI:%s\n", n.value)
		case "rfc822":
			fmt.Fprintf(&out, "EMAIL:%s\n", n.value)
		case "dn":
			fmt.Fprintf(&out, "DIR:%s\n", n.dn.str())
		case "other":
			fmt.Fprintf(&out, "OtherName:%s::%s\n", n.oid, n.value)
		default:
			fmt.Fprintf(&out, "%s: unsupported general name type", n.kind)
		}
	}
	return indentLines(chopLast(out.String()), indent)
}

// formatRevokedCertificates ports ParseX509CRL.mjs formatRevokedCertificates.
func formatRevokedCertificates(h string, seqIdx, indent int) (string, error) {
	if seqIdx < 0 {
		return indentLines("No Revoked Certificates.", indent), nil
	}
	entries, err := asn1GetChildIdx(h, seqIdx)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return indentLines("No Revoked Certificates.", indent), nil
	}
	var out strings.Builder
	for _, e := range entries {
		s, err := formatRevokedEntry(h, e, indent)
		if err != nil {
			return "", err
		}
		out.WriteString(s)
	}
	return indentLines(chopLast(out.String()), indent), nil
}

// formatRevokedEntry formats one revokedCertificate entry.
func formatRevokedEntry(h string, entryIdx, indent int) (string, error) {
	kids, err := asn1GetChildIdx(h, entryIdx)
	if err != nil || len(kids) < 2 {
		return "", errors.New("invalid revoked certificate object, missing either serial number or date") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	sn := strings.ToUpper(asn1GetV(h, kids[0]))
	date, err := generalizedDateTimeToUTC(hextoutf8(asn1GetV(h, kids[1])))
	if err != nil {
		return "", err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "Serial Number: %s\n    Revocation Date: %s\n", sn, date)
	if len(kids) > 2 && jsSubstr(h, kids[2], 2) == "30" {
		entryExts, err := formatCRLEntryExtensions(h, kids[2])
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, "\tCRL entry extensions:\n%s\n", indentLines(entryExts, 2*indent))
	}
	return out.String(), nil
}

// crlReasonNames maps CRL reason codes to their messages.
var crlReasonNames = map[int]string{
	0: "Unspecified", 1: "Key Compromise", 2: "CA Compromise", 3: "Affiliation Changed",
	4: "Superseded", 5: "Cessation Of Operation", 6: "Certificate Hold",
	8: "Remove From CRL", 9: "Privilege Withdrawn", 10: "AA Compromise",
}

// holdInstructionNames maps hold-instruction OIDs to their names.
var holdInstructionNames = map[string]string{
	"1.2.840.10040.2.1": "Hold Instruction None",
	"1.2.840.10040.2.2": "Hold Instruction Call Issuer",
	"1.2.840.10040.2.3": "Hold Instruction Reject",
}

// formatCRLEntryExtensions ports ParseX509CRL.mjs formatCRLEntryExtensions.
func formatCRLEntryExtensions(h string, seqIdx int) (string, error) {
	exts, err := asn1GetChildIdx(h, seqIdx)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, e := range exts {
		kids, err := asn1GetChildIdx(h, e)
		if err != nil || len(kids) < 2 {
			return "", errors.New("malformed CRL entry extension")
		}
		oid := asn1OidHexToInt(asn1GetV(h, kids[0]))
		value := asn1GetV(h, kids[len(kids)-1])
		switch oid {
		case "2.5.29.21": // cRLReason
			code, _ := jsnum.ParseInt(asn1GetV(value, 0), 16)
			msg, ok := crlReasonNames[code]
			if !ok {
				msg = fmt.Sprintf("invalid reason code: %d", code)
			}
			fmt.Fprintf(&out, "X509v3 CRL Reason Code:\n    %s\n", msg)
		case "2.5.29.23": // Hold instruction
			hoid := asn1OidHexToInt(asn1GetV(value, 0))
			name, ok := holdInstructionNames[hoid]
			if !ok {
				name = hoid + ": unknown hold instruction OID"
			}
			fmt.Fprintf(&out, "Hold Instruction Code:\n\t%s\n", name)
		case "2.5.29.24": // Invalidity Date
			d, err := generalizedDateTimeToUTC(hextoutf8(asn1GetV(value, 0)))
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&out, "Invalidity Date:\n\t%s\n", d)
		default:
			fmt.Fprintf(&out, "%s:\n\tUnsupported CRL entry extension. Try openssl CLI.\n", extName(oid))
		}
	}
	return chopLast(out.String()), nil
}

// formatCRLSignature ports ParseX509CRL.mjs formatCRLSignature (54-wide lines).
func formatCRLSignature(sigHex string, indent int) string {
	return indentLines(splitEvery(colonHex(sigHex), 54, "\n"), indent)
}

// splitEvery splits s into width-character chunks joined by sep.
func splitEvery(s string, width int, sep string) string {
	var lines []string
	for len(s) > 0 {
		if len(s) > width {
			lines = append(lines, s[:width])
			s = s[width:]
		} else {
			lines = append(lines, s)
			s = ""
		}
	}
	return strings.Join(lines, sep)
}

// parseGeneralNamesTagged parses a context-tagged GeneralNames container (e.g. a
// DistributionPoint fullName [0]) whose children are GeneralName CHOICEs.
func parseGeneralNamesTagged(h string, idx int) ([]x509GeneralName, error) {
	kids, err := asn1GetChildIdx(h, idx)
	if err != nil {
		return nil, err
	}
	var names []x509GeneralName
	for _, k := range kids {
		gn, err := parseGeneralName(h, k)
		if err != nil {
			return nil, err
		}
		names = append(names, gn)
	}
	return names, nil
}

// generalizedDateTimeToUTC ports ParseX509CRL.mjs generalizedDateTimeToUTC:
// a 12- or 14-digit ASN.1 time (with trailing Z) → an RFC 1123 (UTC) string.
func generalizedDateTimeToUTC(datetime string) (string, error) {
	digits := strings.TrimSuffix(datetime, "Z")
	if len(datetime) == len(digits) || (len(digits) != 12 && len(digits) != 14) || !allDigits(digits) {
		return "", fmt.Errorf("failed to format datetime string %s", datetime)
	}
	century := "20"
	if len(digits) == 14 {
		century = digits[0:2]
		digits = digits[2:]
	}
	iso := fmt.Sprintf("%s%s-%s-%sT%s:%s:%sZ", century, digits[0:2], digits[2:4], digits[4:6], digits[6:8], digits[8:10], digits[10:12])
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return "", fmt.Errorf("failed to format datetime string %s", datetime)
	}
	return t.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"), nil
}

// allDigits reports whether s consists solely of ASCII digits.
func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// x509InputToHex converts the operation input to lowercase DER hex per the
// selected format.
func x509InputToHex(input, format string) (string, error) {
	switch format {
	case "DER Hex":
		return strings.ToLower(stripWhitespace(input)), nil
	case "PEM":
		der, err := pemDER(input)
		if err != nil {
			return "", err
		}
		return hex.EncodeToString(der), nil
	case "Base64":
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input))
		if err != nil {
			return "", errors.New("Certificate load error (non-certificate input?)") //nolint:staticcheck,revive // CyberChef-style text
		}
		return hex.EncodeToString(raw), nil
	default: // "Raw" — the option set is validated upstream, so this is the only remaining case.
		return hex.EncodeToString([]byte(input)), nil
	}
}
