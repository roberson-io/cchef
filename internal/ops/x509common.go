package ops

import (
	"encoding/hex"
	"errors"
	"net"
	"strings"
)

// Shared helpers for the X.509 parse operations (Parse X.509 certificate,
// Parse CSR, Parse X.509 CRL). These reproduce the jsrsasign parsing/formatting
// those CyberChef operations rely on, built on the ASN.1 primitives in
// parseasn1hexstring.go.

// dnAttrShortName maps a DN attribute OID to jsrsasign's short type name
// (KJUR.asn1.x509.OID short-name table). Unknown OIDs fall back to dotted form.
var dnAttrShortName = map[string]string{
	"2.5.4.3":                    "CN",
	"2.5.4.7":                    "L",
	"2.5.4.8":                    "ST",
	"2.5.4.10":                   "O",
	"2.5.4.11":                   "OU",
	"2.5.4.6":                    "C",
	"2.5.4.9":                    "STREET",
	"0.9.2342.19200300.100.1.25": "DC",
	"0.9.2342.19200300.100.1.1":  "UID",
	"2.5.4.4":                    "SN",
	"2.5.4.12":                   "T",
	"2.5.4.42":                   "GN",
	"2.5.4.49":                   "DN",
	"1.2.840.113549.1.9.1":       "E",
	"2.5.4.13":                   "description",
	"2.5.4.15":                   "businessCategory",
	"2.5.4.17":                   "postalCode",
	"2.5.4.5":                    "serialNumber",
	"2.5.4.45":                   "uniqueIdentifier",
	"2.5.4.97":                   "organizationIdentifier",
	"1.3.6.1.4.1.311.60.2.1.1":   "jurisdictionOfIncorporationL",
	"1.3.6.1.4.1.311.60.2.1.2":   "jurisdictionOfIncorporationSP",
	"1.3.6.1.4.1.311.60.2.1.3":   "jurisdictionOfIncorporationC",
}

// x509ATV is one AttributeTypeAndValue in a distinguished name.
type x509ATV struct{ typ, value string }

// x509Name is a parsed distinguished name: an ordered list of RDNs, each of
// which is an ordered list of attributes (RDNs are usually single-valued).
type x509Name struct{ rdns [][]x509ATV }

// parseX500Name parses the RDNSequence at hex position seqIdx into an x509Name.
func parseX500Name(hex string, seqIdx int) (x509Name, error) {
	var name x509Name
	rdnIdxs, err := asn1GetChildIdx(hex, seqIdx)
	if err != nil {
		return name, err
	}
	for _, rdnIdx := range rdnIdxs {
		atvIdxs, err := asn1GetChildIdx(hex, rdnIdx)
		if err != nil {
			return name, err
		}
		var rdn []x509ATV
		for _, atvIdx := range atvIdxs {
			kids, err := asn1GetChildIdx(hex, atvIdx)
			if err != nil || len(kids) < 2 {
				return name, errors.New("malformed distinguished name")
			}
			oid := asn1OidHexToInt(asn1GetV(hex, kids[0]))
			typ, ok := dnAttrShortName[oid]
			if !ok {
				typ = oid
			}
			rdn = append(rdn, x509ATV{typ, asn1StringValue(hex, kids[1])})
		}
		name.rdns = append(name.rdns, rdn)
	}
	return name, nil
}

// str renders the DN in jsrsasign's "/C=UK/ST=London/O=BB/CN=Test Root CA" form.
func (n x509Name) str() string {
	var sb strings.Builder
	for _, rdn := range n.rdns {
		parts := make([]string, len(rdn))
		for i, atv := range rdn {
			parts[i] = atv.typ + "=" + atv.value
		}
		sb.WriteString("/" + strings.Join(parts, "+"))
	}
	return sb.String()
}

// formatDnObj ports CyberChef's lib/PublicKey.mjs formatDnObj: aligns attribute
// names, "type = value" per line, each line prefixed with `indent` spaces.
func formatDnObj(n x509Name, indent int) string {
	maxKeyLen := 0
	for _, rdn := range n.rdns {
		if len(rdn) == 0 {
			continue
		}
		if l := len(rdn[0].typ); l > maxKeyLen {
			maxKeyLen = l
		}
	}
	var out strings.Builder
	for _, rdn := range n.rdns {
		if len(rdn) == 0 {
			continue
		}
		key := rdn[0].typ
		value := rdn[0].value
		str := padEndSpace(key, maxKeyLen) + " = " + value + "\n"
		out.WriteString(strings.Repeat(" ", indent) + str)
	}
	return chopLast(out.String())
}

// asn1StringValue decodes the value of the (string-typed) ASN.1 element at idx
// into a Go string, handling the tags used in DNs and general names.
func asn1StringValue(hex string, idx int) string {
	tag := jsSubstr(hex, idx, 2)
	v := asn1GetV(hex, idx)
	switch tag {
	case "1e": // BMPString (UCS-2)
		return ucs2hextoutf8(v)
	default: // UTF8String / PrintableString / IA5String / T61String / etc.
		return hextoutf8(v)
	}
}

// algIDName resolves the AlgorithmIdentifier SEQUENCE at algSeqIdx to jsrsasign's
// name (e.g. "SHA256withRSA"), falling back to the dotted OID.
func algIDName(hex string, algSeqIdx int) (string, error) {
	kids, err := asn1GetChildIdx(hex, algSeqIdx)
	if err != nil || len(kids) == 0 {
		return "", errors.New("malformed algorithm identifier")
	}
	oid := asn1OidHexToInt(asn1GetV(hex, kids[0]))
	if name := oid2name(oid); name != "" {
		return name, nil
	}
	return oid, nil
}

// x509GeneralName is one parsed GeneralName CHOICE value.
type x509GeneralName struct {
	kind  string // rfc822 | dns | uri | ip | dn | other
	value string // for rfc822/dns/uri/ip
	oid   string // for other
	dn    x509Name
}

// parseGeneralNames parses the GeneralNames SEQUENCE at seqIdx.
func parseGeneralNames(hex string, seqIdx int) ([]x509GeneralName, error) {
	idxs, err := asn1GetChildIdx(hex, seqIdx)
	if err != nil {
		return nil, err
	}
	var names []x509GeneralName
	for _, idx := range idxs {
		gn, err := parseGeneralName(hex, idx)
		if err != nil {
			return nil, err
		}
		names = append(names, gn)
	}
	return names, nil
}

// parseGeneralName parses a single context-tagged GeneralName at idx.
func parseGeneralName(hex string, idx int) (x509GeneralName, error) {
	tag := jsSubstr(hex, idx, 2)
	switch tag {
	case "a0": // [0] otherName SEQUENCE { type-id OID, [0] value }
		kids, err := asn1GetChildIdx(hex, idx)
		if err != nil || len(kids) < 2 {
			return x509GeneralName{}, errors.New("malformed otherName")
		}
		oid := asn1OidHexToInt(asn1GetV(hex, kids[0]))
		inner, err := asn1GetChildIdx(hex, kids[1])
		if err != nil || len(inner) == 0 {
			return x509GeneralName{}, errors.New("malformed otherName value")
		}
		return x509GeneralName{kind: "other", oid: oid, value: asn1StringValue(hex, inner[0])}, nil
	case "81": // [1] rfc822Name IA5String
		return x509GeneralName{kind: "rfc822", value: hextoutf8(asn1GetV(hex, idx))}, nil
	case "82": // [2] dNSName IA5String
		return x509GeneralName{kind: "dns", value: hextoutf8(asn1GetV(hex, idx))}, nil
	case "86": // [6] uniformResourceIdentifier IA5String
		return x509GeneralName{kind: "uri", value: hextoutf8(asn1GetV(hex, idx))}, nil
	case "87": // [7] iPAddress OCTET STRING
		return x509GeneralName{kind: "ip", value: ipFromHex(asn1GetV(hex, idx))}, nil
	case "a4": // [4] directoryName (EXPLICIT Name)
		kids, err := asn1GetChildIdx(hex, idx)
		if err != nil || len(kids) == 0 {
			return x509GeneralName{}, errors.New("malformed directoryName")
		}
		dn, err := parseX500Name(hex, kids[0])
		if err != nil {
			return x509GeneralName{}, err
		}
		return x509GeneralName{kind: "dn", dn: dn}, nil
	default:
		return x509GeneralName{kind: tag}, nil
	}
}

// ipFromHex formats an IP address octet string (4 or 16 bytes) as text, using
// the canonical (RFC 5952) form for IPv6, matching jsrsasign.
func ipFromHex(h string) string {
	if b, err := hex.DecodeString(h); err == nil && (len(b) == 4 || len(b) == 16) {
		return net.IP(b).String()
	}
	return h
}

// --- string helpers (ports of the CyberChef .mjs formatting utilities) --------

// chopLast removes the final character (JS chop); empty input is returned as-is.
func chopLast(s string) string {
	if len(s) < 1 {
		return s
	}
	return s[:len(s)-1]
}

// indentLines prefixes every line of s with n spaces (JS replace(/^/gm, indent)).
// An empty string yields n spaces, matching the JS regex applying once at pos 0.
func indentLines(s string, n int) string {
	pad := strings.Repeat(" ", n)
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

// colonHex inserts ':' between hex byte pairs, left-padding to even length.
func colonHex(hex string) string {
	if len(hex)%2 != 0 {
		hex = "0" + hex
	}
	var sb strings.Builder
	for i := 0; i < len(hex); i += 2 {
		if i > 0 {
			sb.WriteByte(':')
		}
		sb.WriteString(hex[i : i+2])
	}
	return sb.String()
}
