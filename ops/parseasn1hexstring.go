package ops

import (
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseASN1HexString{})
}

// ParseASN1HexString parses ASN.1 data given as a hex string into a text tree.
// It is a faithful port of jsrsasign's ASN1HEX.dump (the library CyberChef's
// "Parse ASN.1 hex string" operation wraps): a recursive walk over BER/DER
// Type-Length-Value structures producing an indented, human-readable dump.
type ParseASN1HexString struct{}

// Meta returns the operation metadata.
func (ParseASN1HexString) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse ASN.1 hex string",
		Module:      "PublicKey",
		Description: "Abstract Syntax Notation One (ASN.1) is a standard and notation that describes rules and structures for representing, encoding, transmitting, and decoding data in telecommunications and computer networking.",
		InfoURL:     "https://wikipedia.org/wiki/Abstract_Syntax_Notation_One",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseASN1HexString) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Starting index", Type: core.ArgNumber, Integer: true, Value: 0},
		{Name: "Truncate octet strings longer than", Flag: "truncate-octet-strings", Type: core.ArgNumber, Integer: true, Value: 32},
	}
}

// Run parses the ASN.1 hex string starting at the given (hex-character) index,
// truncating long primitive values to the given length.
func (ParseASN1HexString) Run(in *core.Dish, args []any) (*core.Dish, error) {
	index := int(args[0].(float64))
	truncateLen := int(args[1].(float64))

	// CyberChef: input.replace(/\s/g, "").toLowerCase()
	hexStr := strings.ToLower(stripWhitespace(in.String()))

	out, err := asn1Dump(hexStr, truncateLen, index, "")
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// stripWhitespace removes all Unicode whitespace (JS \s) from s.
func stripWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if mimeIsJSSpace(r) {
			return -1
		}
		return r
	}, s)
}

// --- ASN1HEX TLV helpers (ports of jsrsasign ASN1HEX.*) ------------------------
//
// Positions ("index") are offsets into the hex string measured in hex
// characters, matching jsrsasign, which indexes the hex string directly.

// asn1GetLblen returns the number of hex chars (÷2 = bytes) the length field
// occupies at position a, or a negative sentinel for indefinite (-1) / reserved
// (-2) forms.
func asn1GetLblen(s string, a int) int {
	if jsSubstr(s, a+2, 1) != "8" {
		return 1
	}
	b, ok := jsParseInt(jsSubstr(s, a+3, 1), 10)
	if !ok {
		return -2
	}
	if b == 0 {
		return -1
	}
	// b is a single decimal digit here, so b is in 1..9.
	return b + 1
}

// asn1GetL returns the hex of the length field at position b.
func asn1GetL(s string, b int) string {
	a := asn1GetLblen(s, b)
	if a < 1 {
		return ""
	}
	return jsSubstr(s, b+2, a*2)
}

// asn1GetVblen returns the byte length of the value at position a (-1 if none).
func asn1GetVblen(s string, a int) int {
	c := asn1GetL(s, a)
	if c == "" {
		return -1
	}
	var b *big.Int
	if jsSubstr(c, 0, 1) == "8" {
		b = bigFromHex(jsSubstrFrom(c, 2))
	} else {
		b = bigFromHex(c)
	}
	return bigIntValue(b)
}

// asn1GetVidx returns the hex-char offset of the value at position b.
func asn1GetVidx(s string, b int) int {
	a := asn1GetLblen(s, b)
	if a < 0 {
		return a
	}
	return b + (a+1)*2
}

// asn1GetV returns the hex of the value at position a.
func asn1GetV(s string, a int) string {
	c := asn1GetVidx(s, a)
	b := asn1GetVblen(s, a)
	return jsSubstr(s, c, b*2)
}

// asn1GetTLVblen returns the total byte length of the TLV at position a.
func asn1GetTLVblen(s string, a int) int {
	return 2 + asn1GetLblen(s, a)*2 + asn1GetVblen(s, a)*2
}

// asn1GetChildIdx returns the hex-char offsets of the child TLVs of the
// constructed value at position k.
func asn1GetChildIdx(e string, k int) ([]int, error) {
	var j []int
	c := asn1GetVidx(e, k)
	f := asn1GetVblen(e, k) * 2
	if c+f > len(e) {
		return nil, errors.New("too short ASN.1 value")
	}
	// BITSTRING: skip the leading "unused bits" octet.
	if jsSubstr(e, k, 2) == "03" {
		c += 2
		f -= 2
	}
	g := 0
	d := c
	for g <= f {
		b := asn1GetTLVblen(e, d)
		if b <= 0 {
			return nil, errors.New("malformed ASN.1: invalid TLV length")
		}
		g += b
		if g <= f {
			j = append(j, d)
		}
		d += b
		if g >= f {
			break
		}
	}
	return j, nil
}

// asn1IsASN1HEX reports whether e is exactly one self-contained ASN.1 TLV, used
// to decide whether an OCTET/BIT STRING or context tag encapsulates ASN.1.
func asn1IsASN1HEX(e string) bool {
	if len(e)%2 == 1 {
		return false
	}
	c := asn1GetVblen(e, 0)
	b := jsSubstr(e, 0, 2)
	f := asn1GetL(e, 0)
	a := len(e) - len(b) - len(f)
	return a == c*2
}

// asn1OidHexToInt converts an OID's value hex to dotted-decimal notation.
func asn1OidHexToInt(a string) string {
	var j string
	if k, ok := jsParseInt(jsSubstr(a, 0, 2), 16); ok {
		j = strconv.Itoa(k/40) + "." + strconv.Itoa(k%40)
	} else {
		j = "NaN.NaN"
	}
	e := ""
	for f := 2; f < len(a); f += 2 {
		g, _ := jsParseInt(jsSubstr(a, f, 2), 16)
		h := byteToBin(g)  // 8-bit binary, zero-padded
		e += h[1:8]        // drop the continuation bit, keep 7 value bits
		if h[0:1] == "0" { // final byte of this arc
			b := new(big.Int)
			b.SetString(e, 2)
			j += "." + b.String()
			e = ""
		}
	}
	return j
}

// asn1OidName resolves an OID value hex to its name, falling back to the dotted
// form. Mirrors jsrsasign ASN1HEX.oidname.
func asn1OidName(a string) string {
	if isHexString(a) {
		a = asn1OidHexToInt(a)
	}
	b := oid2name(a)
	if b == "" {
		b = a
	}
	return b
}

// oid2name returns the first table name whose OID matches the dotted string, or
// "" if unknown. Order matters: e.g. P-256 precedes secp256r1 for the same OID.
func oid2name(dotted string) string {
	for _, p := range asn1OIDTable {
		if p.oid == dotted {
			return p.name
		}
	}
	return ""
}

// --- string / number helpers mirroring the relevant JS semantics --------------

// jsSubstr mirrors JavaScript String.prototype.substr(start, length).
func jsSubstr(s string, start, length int) string {
	n := len(s)
	if start < 0 {
		start = max(n+start, 0)
	}
	if start > n {
		start = n
	}
	if length < 0 {
		length = 0
	}
	end := max(min(start+length, n), start)
	return s[start:end]
}

// jsSubstrFrom mirrors String.prototype.substr(start) (to end of string).
func jsSubstrFrom(s string, start int) string {
	n := len(s)
	if start < 0 {
		start = max(n+start, 0)
	}
	if start > n {
		start = n
	}
	return s[start:]
}

// jsParseInt mimics JS parseInt(s, base) for base 10/16: skips leading
// whitespace and an optional sign, then consumes valid leading digits. ok is
// false when no digits are consumed (JS returns NaN).
func jsParseInt(s string, base int) (int, bool) {
	i := 0
	for i < len(s) && mimeIsJSSpace(rune(s[i])) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	val := 0
	for i < len(s) {
		d := digitVal(s[i])
		if d < 0 || d >= base {
			break
		}
		val = val*base + d
		i++
	}
	if i == start {
		return 0, false
	}
	if neg {
		val = -val
	}
	return val, true
}

// digitVal returns the value of an ASCII hex digit, or -1 if not one.
func digitVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// byteToBin returns the 8-bit, zero-padded binary string of a byte value.
func byteToBin(v int) string {
	s := strconv.FormatInt(int64(v&0xff), 2)
	if len(s) < 8 {
		s = strings.Repeat("0", 8-len(s)) + s
	}
	return s
}

// isHexString reports whether s is an even-length run of hex digits.
func isHexString(s string) bool {
	if len(s)%2 != 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if v := digitVal(s[i]); v < 0 {
			return false
		}
	}
	return true
}

// bigFromHex parses a hex string as a non-negative big integer (0 if empty/bad).
func bigFromHex(s string) *big.Int {
	b := new(big.Int)
	if s == "" {
		return b
	}
	b.SetString(s, 16)
	return b
}

// bigIntValue mirrors jsrsasign BigInteger.intValue(): the low 32 bits as a
// signed int.
func bigIntValue(b *big.Int) int {
	mask := big.NewInt(0xffffffff)
	low := new(big.Int).And(b, mask)
	// #nosec G115 -- low is masked to 32 bits; the int32 truncation deliberately mirrors jsrsasign BigInteger.intValue()
	return int(int32(low.Uint64()))
}

// hextoutf8 decodes a hex string as UTF-8, returning the literal "null" for
// malformed sequences (jsrsasign returns JS null, which stringifies to "null").
func hextoutf8(h string) string {
	b, err := hex.DecodeString(h)
	if err != nil || !utf8.Valid(b) {
		return "null"
	}
	return string(b)
}

// ucs2hextoutf8 decodes a hex string as big-endian UCS-2 (UTF-16 code units,
// no surrogate pairing), used for BMPString. Trailing hex shorter than one code
// unit is ignored (jsrsasign crashes on it; cchef tolerates it).
func ucs2hextoutf8(h string) string {
	var sb strings.Builder
	for i := 0; i+4 <= len(h); i += 4 {
		hi, _ := jsParseInt(h[i:i+2], 16)
		lo, _ := jsParseInt(h[i+2:i+4], 16)
		// #nosec G115 -- hi and lo are single bytes, so the value is a 16-bit BMP code point
		sb.WriteRune(rune(hi<<8 | lo))
	}
	return sb.String()
}

// --- the recursive dumper (port of ASN1HEX.dump) ------------------------------

// asn1Dump renders the TLV at hex-char offset l of e, indented by indent, with
// x509ExtName carrying the enclosing extension's OID name (for subjectAltName
// decoding). ommitLongOctet is the value-truncation threshold in hex chars.
func asn1Dump(e string, ommitLongOctet, l int, indent string, x509ExtName ...string) (string, error) {
	extName := ""
	if len(x509ExtName) > 0 {
		extName = x509ExtName[0]
	}

	z := jsSubstr(e, l, 2)

	// The text-string tags all render as "<Name> '<utf8>'" (unquoted for the
	// time types).
	if t, ok := asn1TextTags[z]; ok {
		v := hextoutf8(asn1GetV(e, l))
		if t.quoted {
			return indent + t.name + " '" + v + "'\n", nil
		}
		return indent + t.name + " " + v + "\n", nil
	}

	if out, ok, err := asn1DumpPrimitive(e, ommitLongOctet, l, indent, extName, z); ok {
		return out, err
	}

	switch z {
	case "30":
		if jsSubstr(e, l, 4) == "3000" {
			return indent + "SEQUENCE {}\n", nil
		}
		d, err := asn1GetChildIdx(e, l)
		if err != nil {
			return "", err
		}
		childExt := extName
		// An X.509 extension is SEQ { OID, [BOOLEAN,] OCTETSTRING }; carry the
		// OID name down so subjectAltName's [2] entries decode as text.
		if (len(d) == 2 || len(d) == 3) && jsSubstr(e, d[0], 2) == "06" && jsSubstr(e, d[len(d)-1], 2) == "04" {
			childExt = asn1OidName(asn1GetV(e, d[0]))
		}
		return asn1DumpChildren(e, ommitLongOctet, d, indent, "SEQUENCE", childExt)
	case "31":
		d, err := asn1GetChildIdx(e, l)
		if err != nil {
			return "", err
		}
		return asn1DumpChildren(e, ommitLongOctet, d, indent, "SET", extName)
	}

	return asn1DumpContextTag(e, ommitLongOctet, l, indent, extName, z)
}

// asn1TruncValue truncates a long value to its first/last ommitLongOctet hex
// characters, noting the total byte count.
func asn1TruncValue(a string, ommitLongOctet int) string {
	if len(a) <= ommitLongOctet*2 {
		return a
	}
	return jsSubstr(a, 0, ommitLongOctet) + "..(total " + strconv.Itoa(len(a)/2) +
		"bytes).." + jsSubstr(a, len(a)-ommitLongOctet, ommitLongOctet)
}

// asn1DumpEncapsulated renders a BITSTRING/OCTETSTRING: when innerHex is itself
// valid ASN.1 it is decoded one level deeper ("encapsulates"), otherwise the
// (possibly truncated) displayValue is shown.
func asn1DumpEncapsulated(label, innerHex, displayValue string, ommitLongOctet int, indent, extName string) (string, error) {
	if asn1IsASN1HEX(innerHex) {
		inner, err := asn1Dump(innerHex, ommitLongOctet, 0, indent+"  ", extName)
		if err != nil {
			return "", err
		}
		return indent + label + ", encapsulates\n" + inner, nil
	}
	return indent + label + " " + displayValue + "\n", nil
}

// asn1DumpPrimitive renders the primitive value tags (boolean, integer, bit /
// octet string, null, object identifier, enumerated, BMP string). The bool
// return reports whether the tag was handled here.
func asn1DumpPrimitive(e string, ommitLongOctet, l int, indent, extName, z string) (string, bool, error) {
	switch z {
	case "01":
		if asn1GetV(e, l) == "00" {
			return indent + "BOOLEAN FALSE\n", true, nil
		}
		return indent + "BOOLEAN TRUE\n", true, nil
	case "02":
		return indent + "INTEGER " + asn1TruncValue(asn1GetV(e, l), ommitLongOctet) + "\n", true, nil
	case "03":
		h := asn1GetV(e, l)
		out, err := asn1DumpEncapsulated("BITSTRING", jsSubstrFrom(h, 2), asn1TruncValue(h, ommitLongOctet), ommitLongOctet, indent, extName)
		return out, true, err
	case "04":
		h := asn1GetV(e, l)
		out, err := asn1DumpEncapsulated("OCTETSTRING", h, asn1TruncValue(h, ommitLongOctet), ommitLongOctet, indent, extName)
		return out, true, err
	case "05":
		return indent + "NULL\n", true, nil
	case "06":
		b := asn1OidHexToInt(asn1GetV(e, l))
		o := oid2name(b)
		a := strings.ReplaceAll(b, ".", " ")
		if o != "" {
			return indent + "ObjectIdentifier " + o + " (" + a + ")\n", true, nil
		}
		return indent + "ObjectIdentifier (" + a + ")\n", true, nil
	case "0a":
		n, ok := jsParseInt(asn1GetV(e, l), 10)
		if !ok {
			return indent + "ENUMERATED NaN\n", true, nil
		}
		return indent + "ENUMERATED " + strconv.Itoa(n) + "\n", true, nil
	case "1e":
		return indent + "BMPString '" + ucs2hextoutf8(asn1GetV(e, l)) + "'\n", true, nil
	default:
		return "", false, nil
	}
}

// asn1TextTags maps the ASN.1 text-string tag bytes to their display name and
// whether the value is quoted (the time types are not).
var asn1TextTags = map[string]struct {
	name   string
	quoted bool
}{
	"0c": {"UTF8String", true},
	"13": {"PrintableString", true},
	"14": {"TeletexString", true},
	"16": {"IA5String", true},
	"17": {"UTCTime", false},
	"18": {"GeneralizedTime", false},
	"1a": {"VisualString", true},
}

// asn1DumpChildren renders header followed by each child at the given indices,
// recursively dumped one level deeper. Shared by SEQUENCE, SET and constructed
// context tags.
func asn1DumpChildren(e string, ommitLongOctet int, indices []int, indent, header, childExt string) (string, error) {
	var k strings.Builder
	k.WriteString(indent + header + "\n")
	for _, di := range indices {
		s, err := asn1Dump(e, ommitLongOctet, di, indent+"  ", childExt)
		if err != nil {
			return "", err
		}
		k.WriteString(s)
	}
	return k.String(), nil
}

// asn1DumpContextTag renders a non-universal tag (context/application/private
// class, high bit set) as a constructed "[n]" group or a primitive "[n] value",
// falling back to UNKNOWN for anything else.
func asn1DumpContextTag(e string, ommitLongOctet, l int, indent, extName, z string) (string, error) {
	zi, ok := jsParseInt(z, 16)
	if ok && zi&128 != 0 {
		n := zi & 31
		if zi&32 != 0 { // constructed
			d, err := asn1GetChildIdx(e, l)
			if err != nil {
				return "", err
			}
			return asn1DumpChildren(e, ommitLongOctet, d, indent, "["+strconv.Itoa(n)+"]", extName)
		}
		// primitive
		h := asn1GetV(e, l)
		if asn1IsASN1HEX(h) {
			inner, err := asn1Dump(h, ommitLongOctet, 0, indent+"  ", extName)
			if err != nil {
				return "", err
			}
			return indent + "[" + strconv.Itoa(n) + "]\n" + inner, nil
		}
		if jsSubstr(h, 0, 8) == "68747470" { // "http" → decode as URL text
			h = hextoutf8(h)
		} else if extName == "subjectAltName" && n == 2 {
			h = hextoutf8(h)
		}
		return indent + "[" + strconv.Itoa(n) + "] " + h + "\n", nil
	}

	if !ok {
		return indent + "UNKNOWN(NaN) " + asn1GetV(e, l) + "\n", nil
	}
	return indent + "UNKNOWN(" + strconv.Itoa(zi) + ") " + asn1GetV(e, l) + "\n", nil
}
