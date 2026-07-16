package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// asn1 wraps a Parse ASN.1 hex string recipe with the given args.
func asn1(index, trunc int) core.Recipe {
	return core.Recipe{{Op: "Parse ASN.1 hex string", Args: []any{index, trunc}}}
}

// TestParseASN1HexStringFixtures checks output against the authoritative
// jsrsasign ASN1HEX.dump oracle (CyberChef bundles jsrsasign; outputs were
// generated from the same library version).
func TestParseASN1HexStringFixtures(t *testing.T) {
	runCases(t, []opCase{
		// primitives
		{"bool true", "0101ff", "BOOLEAN TRUE\n", asn1(0, 32)},
		{"bool false", "010100", "BOOLEAN FALSE\n", asn1(0, 32)},
		{"integer", "02012a", "INTEGER 2a\n", asn1(0, 32)},
		{"integer negative", "0201ff", "INTEGER ff\n", asn1(0, 32)},
		{"null", "0500", "NULL\n", asn1(0, 32)},
		{"enumerated", "0a0105", "ENUMERATED 5\n", asn1(0, 32)},
		{"octetstring", "0404deadbeef", "OCTETSTRING deadbeef\n", asn1(0, 32)},
		{"utf8string", "0c0668c3a96c6c6f", "UTF8String 'héllo'\n", asn1(0, 32)},
		{"printablestring", "130548656c6c6f", "PrintableString 'Hello'\n", asn1(0, 32)},
		{"ia5string", "16076140622e636f6d", "IA5String 'a@b.com'\n", asn1(0, 32)},
		{"utctime", "170d3230303130313030303030305a", "UTCTime 200101000000Z\n", asn1(0, 32)},
		{"generalizedtime", "180f32303230303130313030303030305a", "GeneralizedTime 20200101000000Z\n", asn1(0, 32)},
		{"visualstring", "1a0548656c6c6f", "VisualString 'Hello'\n", asn1(0, 32)},
		{"teletexstring", "140548656c6c6f", "TeletexString 'Hello'\n", asn1(0, 32)},
		{"bmpstring", "1e0400480049", "BMPString 'HI'\n", asn1(0, 32)},

		// object identifiers (named, first-match-wins, and unknown)
		{"oid sha256", "0609608648016503040201", "ObjectIdentifier sha256 (2 16 840 1 101 3 4 2 1)\n", asn1(0, 32)},
		{"oid commonName", "0603550403", "ObjectIdentifier commonName (2 5 4 3)\n", asn1(0, 32)},
		{"oid P-256 wins over secp256r1", "06082a8648ce3d030107", "ObjectIdentifier P-256 (1 2 840 10045 3 1 7)\n", asn1(0, 32)},
		{"oid unknown", "06042a030405", "ObjectIdentifier (1 2 3 4 5)\n", asn1(0, 32)},

		// constructed
		{"sequence", "3006020101020102", "SEQUENCE\n  INTEGER 01\n  INTEGER 02\n", asn1(0, 32)},
		{"sequence empty", "3000", "SEQUENCE {}\n", asn1(0, 32)},
		{"set", "3106020101020102", "SET\n  INTEGER 01\n  INTEGER 02\n", asn1(0, 32)},
		{
			"algorithmidentifier", "300d06096086480165030402010500",
			"SEQUENCE\n  ObjectIdentifier sha256 (2 16 840 1 101 3 4 2 1)\n  NULL\n", asn1(0, 32),
		},
		{
			"octetstring encapsulates", "04053003020101",
			"OCTETSTRING, encapsulates\n  SEQUENCE\n    INTEGER 01\n", asn1(0, 32),
		},
		{
			"bitstring encapsulates", "030400020101",
			"BITSTRING, encapsulates\n  INTEGER 01\n", asn1(0, 32),
		},
		{"bitstring raw", "03020012", "BITSTRING 0012\n", asn1(0, 32)},

		// context tags
		{"context constructed", "a003020101", "[0]\n  INTEGER 01\n", asn1(0, 32)},
		{
			"context primitive http decode", "8613687474703a2f2f6578616d706c652e636f6d",
			"[6] http://example.com\n", asn1(0, 32),
		},
		{"context primitive raw", "80020102", "[0] 0102\n", asn1(0, 32)},

		// x509 extension SEQUENCEs (OID + OCTETSTRING): subjectAltName [2] decodes
		{
			"subjectAltName ext", "30100603551d11040930078205612e636f6d",
			"SEQUENCE\n  ObjectIdentifier subjectAltName (2 5 29 17)\n" +
				"  OCTETSTRING, encapsulates\n    SEQUENCE\n      [2] a.com\n", asn1(0, 32),
		},
		{
			"basicConstraints ext", "300c0603551d13040530030101ff",
			"SEQUENCE\n  ObjectIdentifier basicConstraints (2 5 29 19)\n" +
				"  OCTETSTRING, encapsulates\n    SEQUENCE\n      BOOLEAN TRUE\n", asn1(0, 32),
		},

		// truncation of long values (first/last N HEX CHARS, not bytes)
		{
			"octet long default trunc", "0428" + strings.Repeat("ab", 40),
			"OCTETSTRING " + strings.Repeat("ab", 16) + "..(total 40bytes).." + strings.Repeat("ab", 16) + "\n", asn1(0, 32),
		},
		{
			"octet custom trunc 4", "0428" + strings.Repeat("ab", 40),
			"OCTETSTRING abab..(total 40bytes)..abab\n", asn1(0, 4),
		},
		{
			"octet at boundary not truncated", "0420" + strings.Repeat("ab", 32),
			"OCTETSTRING " + strings.Repeat("ab", 32) + "\n", asn1(0, 32),
		},
		{
			"integer long custom trunc 4", "0229" + "00" + strings.Repeat("ff", 40),
			"INTEGER 00ff..(total 41bytes)..ffff\n", asn1(0, 4),
		},

		// starting index is a HEX-CHAR offset into the (normalised) input
		{"starting index", "020101020102", "INTEGER 02\n", asn1(6, 32)},

		// input normalisation: whitespace stripped, hex lower-cased
		{
			"whitespace and uppercase normalised", "30 06 02 01 01\n02 01 02",
			"SEQUENCE\n  INTEGER 01\n  INTEGER 02\n", asn1(0, 32),
		},
		{"uppercase context tag", "A003020101", "[0]\n  INTEGER 01\n", asn1(0, 32)},

		// quirky edges reproduced faithfully from jsrsasign
		{"empty input", "", "UNKNOWN(NaN) \n", asn1(0, 32)},
		{"odd length integer", "020", "INTEGER \n", asn1(0, 32)},
		{"indefinite length sequence", "3080020101020102", "SEQUENCE\n", asn1(0, 32)},
		{"unknown tag 1f", "1f020102", "UNKNOWN(31) 0102\n", asn1(0, 32)},
		{"unknown tag 09", "09022a31", "UNKNOWN(9) 2a31\n", asn1(0, 32)},
		{"unknown tag 07 with value", "07022a31", "UNKNOWN(7) 2a31\n", asn1(0, 32)},

		// long-form (multi-byte) definite length: 0x81 0xc8 = 200 bytes
		{
			"long-form length", "0481c8" + strings.Repeat("ab", 200),
			"OCTETSTRING " + strings.Repeat("ab", 16) + "..(total 200bytes).." + strings.Repeat("ab", 16) + "\n", asn1(0, 32),
		},
		{
			"long-form length custom trunc", "0481c8" + strings.Repeat("ab", 200),
			"OCTETSTRING abab..(total 200bytes)..abab\n", asn1(0, 4),
		},
		// reserved long-form length (length-of-length nibble >= 10) → empty value
		{"reserved length form", "048a", "OCTETSTRING \n", asn1(0, 32)},
		// ENUMERATED whose value hex is not decimal digits (jsrsasign base-10 quirk)
		{"enumerated non-decimal", "0a01fa", "ENUMERATED NaN\n", asn1(0, 32)},
		// empty OID value → jsrsasign yields NaN arcs
		{"empty oid value", "0600", "ObjectIdentifier (NaN NaN)\n", asn1(0, 32)},
		// invalid UTF-8 in a UTF8String → jsrsasign returns null
		{"invalid utf8", "0c04f0288cbc", "UTF8String 'null'\n", asn1(0, 32)},
		// extension SEQUENCE with an unknown OID keeps the dotted form
		{
			"unknown-oid extension", "300806022a0304020500",
			"SEQUENCE\n  ObjectIdentifier (1 2 3)\n  OCTETSTRING, encapsulates\n    NULL\n", asn1(0, 32),
		},
		// context-primitive tag whose value is itself ASN.1 → encapsulated
		{"context primitive encapsulates", "8003020101", "[0]\n  INTEGER 01\n", asn1(0, 32)},
		// critical extension: SEQ { OID, BOOLEAN, OCTETSTRING } (3 children)
		{
			"critical extension", "300f0603551d130101ff040530030101ff",
			"SEQUENCE\n  ObjectIdentifier basicConstraints (2 5 29 19)\n  BOOLEAN TRUE\n" +
				"  OCTETSTRING, encapsulates\n    SEQUENCE\n      BOOLEAN TRUE\n", asn1(0, 32),
		},
	})
}

// TestParseASN1HexStringErrors covers the thrown-error paths (verbatim jsrsasign
// messages), which cannot flow through runCases.
func TestParseASN1HexStringErrors(t *testing.T) {
	const badLen = "malformed ASN.1: invalid TLV length"
	cases := []struct{ name, input, want string }{
		{"too short value", "3006020101", "too short ASN.1 value"},
		{"malformed child TLV length", "30020480", badLen},
		// error propagation up through each nesting construct:
		{"octetstring encaps propagates", "040430020480", badLen},
		{"sequence child propagates", "3006040430020480", badLen},
		{"bitstring encaps propagates", "03050030020480", badLen},
		{"set malformed", "31020480", badLen},
		{"set child propagates", "3106040430020480", badLen},
		{"context constructed malformed", "a0020480", badLen},
		{"context constructed child propagates", "a006040430020480", badLen},
		{"context primitive encaps propagates", "800430020480", badLen},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Parse ASN.1 hex string", c.input, 0, 32)
			if err == nil {
				t.Fatalf("expected error %q", c.want)
			}
			if err.Error() != c.want {
				t.Fatalf("got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

// TestParseASN1HexStringHelpers directly exercises the pure helper functions,
// including branches not reachable through Run (input is lower-cased before it
// reaches the parser, so the upper-case hex paths need direct coverage). These
// mirror the case-insensitive JS semantics of the originals.
func TestParseASN1HexStringHelpers(t *testing.T) {
	// jsSubstr: negative start, over-long start, negative length, end<start.
	if got := jsSubstr("abcdef", -2, 2); got != "ef" {
		t.Errorf("jsSubstr negative start: %q", got)
	}
	if got := jsSubstr("abcdef", -100, 2); got != "ab" {
		t.Errorf("jsSubstr clamped negative start: %q", got)
	}
	if got := jsSubstr("abc", 10, 2); got != "" {
		t.Errorf("jsSubstr start past end: %q", got)
	}
	if got := jsSubstr("abc", 1, -1); got != "" {
		t.Errorf("jsSubstr negative length: %q", got)
	}
	// jsSubstrFrom: negative and over-long start.
	if got := jsSubstrFrom("abcdef", -2); got != "ef" {
		t.Errorf("jsSubstrFrom negative start: %q", got)
	}
	if got := jsSubstrFrom("abc", 10); got != "" {
		t.Errorf("jsSubstrFrom start past end: %q", got)
	}
	// jsParseInt: leading whitespace, sign, upper-case hex, and no-digit (NaN).
	if v, ok := jsParseInt("  -1f", 16); !ok || v != -31 {
		t.Errorf("jsParseInt signed hex: %d %v", v, ok)
	}
	if v, ok := jsParseInt("FF", 16); !ok || v != 255 {
		t.Errorf("jsParseInt upper-case hex: %d %v", v, ok)
	}
	if _, ok := jsParseInt("xyz", 16); ok {
		t.Error("jsParseInt no digits should be NaN")
	}
	// digitVal: upper-case and invalid.
	if digitVal('A') != 10 || digitVal('z') != -1 {
		t.Error("digitVal upper/invalid")
	}
	// isHexString: valid, odd length, non-hex.
	if !isHexString("00ab") || isHexString("abc") || isHexString("zz") {
		t.Error("isHexString")
	}
	// bigFromHex: empty → 0.
	if bigFromHex("").Sign() != 0 {
		t.Error("bigFromHex empty")
	}
	// hextoutf8: odd-length hex is undecodable → "null".
	if got := hextoutf8("f"); got != "null" {
		t.Errorf("hextoutf8 bad hex: %q", got)
	}
	// asn1OidName: dotted (non-hex) input passes straight through oid2name.
	if got := asn1OidName("2.5.4.3"); got != "commonName" {
		t.Errorf("asn1OidName dotted known: %q", got)
	}
	if got := asn1OidName("1.2.3.4.5"); got != "1.2.3.4.5" {
		t.Errorf("asn1OidName dotted unknown: %q", got)
	}
	// asn1IsASN1HEX: odd-length input is not a valid TLV.
	if asn1IsASN1HEX("abc") {
		t.Error("asn1IsASN1HEX odd length should be false")
	}
	// asn1GetChildIdx on a BITSTRING skips the leading unused-bits octet; the
	// "03" branch is not reachable through Run (dump handles BITSTRING directly).
	if idx, err := asn1GetChildIdx("030700020101020102", 0); err != nil || len(idx) != 2 || idx[0] != 6 || idx[1] != 12 {
		t.Errorf("asn1GetChildIdx bitstring: %v %v", idx, err)
	}
}

// --- direct tests for the helpers extracted from asn1Dump ---

// TestASN1DumpChildren documents the shared "header + recursively-dumped
// children" rendering used by SEQUENCE/SET/constructed context tags.
func TestASN1DumpChildren(t *testing.T) {
	// SEQUENCE { INTEGER 1 } = 3003020101; its one child (the INTEGER) is at
	// hex offset 4.
	const e = "3003020101"
	d, err := asn1GetChildIdx(e, 0)
	if err != nil {
		t.Fatalf("asn1GetChildIdx: %v", err)
	}
	got, err := asn1DumpChildren(e, 32, d, "", "SEQUENCE", "")
	if err != nil {
		t.Fatalf("asn1DumpChildren: %v", err)
	}
	if got != "SEQUENCE\n  INTEGER 01\n" {
		t.Fatalf("got %q", got)
	}
}

// TestASN1DumpContextTag documents context/application/private tag handling and
// the UNKNOWN fallback.
func TestASN1DumpContextTag(t *testing.T) {
	cases := []struct{ name, e, z, want string }{
		{"primitive [0]", "800141", "80", "[0] 41\n"},
		{"constructed [0]", "a003020101", "a0", "[0]\n  INTEGER 01\n"},
		{"unknown tag", "0f0100", "0f", "UNKNOWN(15) 00\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := asn1DumpContextTag(c.e, 32, 0, "", "", c.z)
			if err != nil {
				t.Fatalf("asn1DumpContextTag: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestASN1TruncValue documents the long-value truncation helper.
func TestASN1TruncValue(t *testing.T) {
	if got := asn1TruncValue("0011", 2); got != "0011" { // short: passthrough
		t.Fatalf("short: %q", got)
	}
	if got := asn1TruncValue("00112233445566778899", 2); got != "00..(total 10bytes)..99" {
		t.Fatalf("long: %q", got)
	}
}

// TestASN1DumpEncapsulated documents the shared BITSTRING/OCTETSTRING rendering:
// recurse when the inner hex is valid ASN.1, otherwise show the raw value.
func TestASN1DumpEncapsulated(t *testing.T) {
	// Not encapsulated: "41" is not valid ASN.1 hex.
	got, err := asn1DumpEncapsulated("OCTETSTRING", "41", "41", 32, "", "")
	if err != nil || got != "OCTETSTRING 41\n" {
		t.Fatalf("plain: %q, %v", got, err)
	}
	// Encapsulated: a nested SEQUENCE { INTEGER 1 }.
	got, err = asn1DumpEncapsulated("OCTETSTRING", "3003020101", "3003020101", 32, "", "")
	if err != nil || got != "OCTETSTRING, encapsulates\n  SEQUENCE\n    INTEGER 01\n" {
		t.Fatalf("encapsulated: %q, %v", got, err)
	}
}

// TestASN1DumpPrimitive documents the primitive value-tag dispatch and its
// "handled" signal.
func TestASN1DumpPrimitive(t *testing.T) {
	cases := []struct{ name, e, z, want string }{
		{"null", "0500", "05", "NULL\n"},
		{"integer", "020101", "02", "INTEGER 01\n"},
		{"boolean true", "0101ff", "01", "BOOLEAN TRUE\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, handled, err := asn1DumpPrimitive(c.e, 32, 0, "", "", c.z)
			if err != nil || !handled || got != c.want {
				t.Fatalf("got (%q, %v, %v), want %q", got, handled, err, c.want)
			}
		})
	}
	// A constructed tag is not a primitive.
	if _, handled, _ := asn1DumpPrimitive("3003020101", 32, 0, "", "", "30"); handled {
		t.Fatal("SEQUENCE should not be handled by asn1DumpPrimitive")
	}
}
