package ops

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"regexp"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Key material and signature vectors transcribed from
// ../CyberChef/tests/operations/tests/ECDSA.mjs.
const (
	p256PrivPkcs1 = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINtTjwUkgfAiSwqgcGAXWyE0ueIW6n2k395dmQZ3vGr4oAoGCCqGSM49
AwEHoUQDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJgusgcAE8H6810fkJ8ZmTNiCC
a6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END EC PRIVATE KEY-----`

	p256PrivPkcs8 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg21OPBSSB8CJLCqBw
YBdbITS54hbqfaTf3l2ZBne8avihRANCAAQNRzwDQQM0qgJgg9YwfPXJTOoTmYmC
6yBwATwfrzXR+QnxmZM2IIJrqwuBHa8PVU2HZ2KKtaAo8fg9Uwpq/l7p
-----END PRIVATE KEY-----`

	p256Pub = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJ
gusgcAE8H6810fkJ8ZmTNiCCa6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END PUBLIC KEY-----`

	p384PrivPkcs8 = `-----BEGIN PRIVATE KEY-----
MIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDAYo22xn2kZjN8MInom
NDsgD/zhpUwnCYch634jUgO59fN9m2lR5ekaI1XABHz39rihZANiAAQwXoCsPOLv
Nn2STUs/hpL41CQveSL3WUmJ4QdtD7UFCl1mBO6ME0xSUgIQTUNkHt5k9CpOq3x9
r+LG5+GcisoLn7R54R+bRoGp/p1ZBeuBXoCgthvs+RFoT3OewUmA8oQ=
-----END PRIVATE KEY-----`

	p384Pub = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEMF6ArDzi7zZ9kk1LP4aS+NQkL3ki91lJ
ieEHbQ+1BQpdZgTujBNMUlICEE1DZB7eZPQqTqt8fa/ixufhnIrKC5+0eeEfm0aB
qf6dWQXrgV6AoLYb7PkRaE9znsFJgPKE
-----END PUBLIC KEY-----`

	p521PrivPkcs8 = `-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIAifBaJDqNwOtKgThc
FU34GzPQ73ubOQg9dnighpVGwA3b/KwCifimCNKDmKnXJaE04mEcxg8yzcFKausF
5I8o206hgYkDgYYABAGwpkwrBBlZOdx4u9mxqYxJvtzAHaFFAzl21WQVbAjyrqXe
nFPMkhbFpEEWr1ualPYKQkHe14AX33iU3fQ9MlBkgAAripsPbiKggAaog74cUERo
qbrUFZwMbptGgovpE6pU93h7A1wb3Vtw9DZQCgiNbwzMbdsft+p2RJ8iSxWEC6Gd
mw==
-----END PRIVATE KEY-----`

	p521Pub = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBsKZMKwQZWTnceLvZsamMSb7cwB2h
RQM5dtVkFWwI8q6l3pxTzJIWxaRBFq9bmpT2CkJB3teAF994lN30PTJQZIAAK4qb
D24ioIAGqIO+HFBEaKm61BWcDG6bRoKL6ROqVPd4ewNcG91bcPQ2UAoIjW8MzG3b
H7fqdkSfIksVhAuhnZs=
-----END PUBLIC KEY-----`

	rsaPub = `-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----`

	asciiText = "A common mistake that people make when trying to design something completely foolproof is to underestimate the ingenuity of complete fools."

	sigASN1  = "3046022100e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127022100b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d"
	sigP1363 = "e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d"
	sigJWS   = "4GkFYIovp9vanihMKnlZ37aPtSel8AOy15df8TUUUSe2uqJTeTM0-Lk-od1iK8YAEk2AkLq9gH7-P3e4syQ4jQ"
	sigJSON  = `{"r":"00e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127","s":"00b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d"}`
)

// TestECDSASignatureConversion covers the deterministic format conversions.
func TestECDSASignatureConversion(t *testing.T) {
	conv := func(in, out string) core.Recipe {
		return core.Recipe{{Op: "ECDSA Signature Conversion", Args: []any{in, out}}}
	}
	runCases(t, []opCase{
		{"asn1->asn1", sigASN1, sigASN1, conv("Auto", "ASN.1 HEX")},
		{"asn1->p1363", sigASN1, sigP1363, conv("Auto", "P1363 HEX")},
		{"asn1->jws", sigASN1, sigJWS, conv("Auto", "JSON Web Signature")},
		{"asn1->json", sigASN1, sigJSON, conv("Auto", "Raw JSON")},
		{"p1363->asn1", sigP1363, sigASN1, conv("Auto", "ASN.1 HEX")},
		{"p1363->p1363", sigP1363, sigP1363, conv("Auto", "P1363 HEX")},
		{"p1363->jws", sigP1363, sigJWS, conv("Auto", "JSON Web Signature")},
		{"p1363->json", sigP1363, sigJSON, conv("Auto", "Raw JSON")},
		{"json->asn1", sigJSON, sigASN1, conv("Auto", "ASN.1 HEX")},
		{"json->p1363", sigJSON, sigP1363, conv("Auto", "P1363 HEX")},
		{"json->jws", sigJSON, sigJWS, conv("Auto", "JSON Web Signature")},
		{"json->json", sigJSON, sigJSON, conv("Auto", "Raw JSON")},
		{"jws->asn1", sigJWS, sigASN1, conv("Auto", "ASN.1 HEX")},
		{"jws->jws", sigJWS, sigJWS, conv("Auto", "JSON Web Signature")},
		// Explicit input format (not Auto).
		{"explicit p1363->asn1", sigP1363, sigASN1, conv("P1363 HEX", "ASN.1 HEX")},
		{"explicit jws->asn1", sigJWS, sigASN1, conv("JSON Web Signature", "ASN.1 HEX")},
		{"explicit json->asn1", sigJSON, sigASN1, conv("Raw JSON", "ASN.1 HEX")},
	})
}

// P-521 signature vectors (from the CyberChef-server oracle) exercise the
// long-form DER length and the P-521 branch of asn1SigToConcatSig.
const (
	sig521ASN1  = "30818702412e6c534b94d89065f9a076d902a6a40f40b39f757b8a8f8d264d0f65e903f33dd5aa5c95a066def8eea5a95e6870a65d3776840d226d429f4b3954cc6f9899e337024201a8407ddaf317ebffbe90567ed577f8154f0399d773b7bb95c5f5475043742ee5ec23bf2ba7ed3445ec1d3dc8e8b7edb3cfd3d64823bfe120e747d5860c6908c34d"
	sig521P1363 = "002e6c534b94d89065f9a076d902a6a40f40b39f757b8a8f8d264d0f65e903f33dd5aa5c95a066def8eea5a95e6870a65d3776840d226d429f4b3954cc6f9899e33701a8407ddaf317ebffbe90567ed577f8154f0399d773b7bb95c5f5475043742ee5ec23bf2ba7ed3445ec1d3dc8e8b7edb3cfd3d64823bfe120e747d5860c6908c34d"
	sig521JSON  = `{"r":"2e6c534b94d89065f9a076d902a6a40f40b39f757b8a8f8d264d0f65e903f33dd5aa5c95a066def8eea5a95e6870a65d3776840d226d429f4b3954cc6f9899e337","s":"01a8407ddaf317ebffbe90567ed577f8154f0399d773b7bb95c5f5475043742ee5ec23bf2ba7ed3445ec1d3dc8e8b7edb3cfd3d64823bfe120e747d5860c6908c34d"}`
)

const rsaPrivPkcs1 = `-----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelMYKtboGLrk6ihtqFPZFRL
NcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQJAOJUpM0lv36MAQR3WAwsF
F7DOy+LnigteCvaNWiNVxZ6jByB5Qb7sall/Qlu9sFI0ZwrlVcKS0kldee7JTYlL
WQIhAP3UKEfOtpTgT1tYmdhaqjxqMfxBom0Ri+rt9ajlzs6vAiEA9L85B8/Gnb7p
6Af7/wpmafL277OV4X4xBfzMR+TUzHUCIBq+VLQkInaTH6lXL3ZtLwyIf9W9MJjf
RWeuRLjT5bM/AiBF7Kw6kx5Hy1fAtydEApCoDIaIjWJw/kC7WTJ0B+jUUQIgV6dw
NSyj0feakeD890gmId+lvl/w/3oUXiczqvl/N9o=
-----END RSA PRIVATE KEY-----`

// TestECDSAConversionP521 covers the P-521 conversion branch (long-form length,
// max-width padding).
func TestECDSAConversionP521(t *testing.T) {
	conv := func(in, out string) core.Recipe {
		return core.Recipe{{Op: "ECDSA Signature Conversion", Args: []any{in, out}}}
	}
	runCases(t, []opCase{
		{"521 asn1->p1363", sig521ASN1, sig521P1363, conv("Auto", "P1363 HEX")},
		{"521 asn1->json", sig521ASN1, sig521JSON, conv("Auto", "Raw JSON")},
		{"521 p1363->asn1", sig521P1363, sig521ASN1, conv("Auto", "ASN.1 HEX")},
		{"521 json->p1363", sig521JSON, sig521P1363, conv("Auto", "P1363 HEX")},
	})
}

// TestECDSAConversionErrors covers the malformed-input paths of the conversion.
func TestECDSAConversionErrors(t *testing.T) {
	conv := func(in, out string) core.Recipe {
		return core.Recipe{{Op: "ECDSA Signature Conversion", Args: []any{in, out}}}
	}
	cases := []struct {
		name, input string
		recipe      core.Recipe
		want        string
	}{
		{"p1363 odd length", "abc", conv("P1363 HEX", "ASN.1 HEX"), "r-s sig length error"},
		{"json non-hex r", `{"r":"zz","s":"11"}`, conv("Auto", "ASN.1 HEX"), "invalid r value"},
		{"json non-hex s", `{"r":"11","s":"zz"}`, conv("Auto", "ASN.1 HEX"), "invalid s value"},
		{"bad width to p1363", "3006020105020103", conv("ASN.1 HEX", "P1363 HEX"), "unknown ECDSA sig r length error"},
		{"bad width to jws", "3006020105020103", conv("ASN.1 HEX", "JSON Web Signature"), "unknown ECDSA sig r length error"},
		{"non-seq to json", "020101", conv("ASN.1 HEX", "Raw JSON"), "signature is not a ASN.1 sequence"},
		{"undetectable", "@@@ not a sig", conv("Auto", "ASN.1 HEX"), "Signature format could not be detected"},
		{"explicit json non-json", "not json", conv("Raw JSON", "ASN.1 HEX"), "Signature is not valid JSON"},
		{"explicit jws bad base64", "@@@@", conv("JSON Web Signature", "ASN.1 HEX"), "illegal base64"},
		{"not asn1 sequence", "020101", conv("ASN.1 HEX", "P1363 HEX"), "signature is not a ASN.1 sequence"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.recipe.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want %q", err, c.want)
			}
		})
	}
}

// TestECDSAInternals directly exercises the low-level parsing/conversion helper
// branches that are awkward to reach through the operations.
func TestECDSAInternals(t *testing.T) {
	// ecdsaDerLen: offset past the buffer, and an over-long length field.
	if _, _, ok := ecdsaDerLen([]byte{0x30, 0x01}, 5); ok {
		t.Error("derLen out of range should fail")
	}
	if _, _, ok := ecdsaDerLen([]byte{0x30, 0x88, 1, 2, 3, 4, 5, 6, 7, 8}, 1); ok {
		t.Error("derLen n>4 should fail")
	}

	// ecdsaParseSigRS error paths.
	parseErrs := map[string]string{
		"zz":               "signature is not a ASN.1 sequence", // bad hex
		"020101":           "signature is not a ASN.1 sequence", // not a SEQUENCE
		"300501":           "signature is not a ASN.1 sequence", // declared length overruns
		"300402050101":     "signature shall have two elements", // r length overruns
		"3006020101030101": "signature shall have two elements", // second element not INTEGER
	}
	for in, want := range parseErrs {
		if _, _, err := ecdsaParseSigRS(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseSigRS(%q): got %v, want %q", in, err, want)
		}
	}

	// ecdsaIsASN1 false cases.
	for _, bad := range []string{"zz", "30ff", "3005010203"} {
		if ecdsaIsASN1(bad) {
			t.Errorf("isASN1(%q) should be false", bad)
		}
	}

	// asn1SigToConcatSig: r/s that are 15 bytes each get zero-padded (%32==30).
	r15 := "020f" + strings.Repeat("22", 15)
	s15 := "020f" + strings.Repeat("33", 15)
	seq15 := "30" + toHexByte(len(r15+s15)/2) + r15 + s15
	if got, err := ecdsaAsn1ToConcat(seq15); err != nil || len(got) != 64 {
		t.Errorf("asn1ToConcat pad: got %q (len %d) err %v", got, len(got), err)
	}
	// asn1SigToConcatSig: valid-width r but a 1-byte s triggers the s-length error.
	r32 := "0220" + strings.Repeat("11", 32)
	s1 := "020105"
	badS := "30" + toHexByte(len(r32+s1)/2) + r32 + s1
	if _, err := ecdsaAsn1ToConcat(badS); err == nil || !strings.Contains(err.Error(), "unknown ECDSA sig s length error") {
		t.Errorf("asn1ToConcat s-length: got %v", err)
	}

	// ecdsaGetKey: unsupported PEM block, and malformed key bodies per block type.
	if _, err := ecdsaGetKey("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"); err == nil {
		t.Error("unsupported block should error")
	}
	for _, blk := range []string{"EC PRIVATE KEY", "PRIVATE KEY", "PUBLIC KEY"} {
		pemStr := "-----BEGIN " + blk + "-----\nAAAA\n-----END " + blk + "-----"
		if _, err := ecdsaGetKey(pemStr); err == nil {
			t.Errorf("malformed %q should error", blk)
		}
	}

	// ecdsaGetKey classifies RSA (PKCS8) and non-EC/RSA (Ed25519) keys.
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	if k := pkcs8Pem(t, rsaKey); mustGetKey(t, k).algo != "RSA" {
		t.Error("RSA PKCS8 should classify as RSA")
	}
	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if k := mustGetKey(t, pkcs8Pem(t, edPriv)); k.algo != "" || !k.isPrivate {
		t.Errorf("Ed25519 private should be non-EC private, got %+v", k)
	}
	edPubDer, err := x509.MarshalPKIXPublicKey(edPriv.Public())
	if err != nil {
		t.Fatal(err)
	}
	edPubPem := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: edPubDer}))
	if k := mustGetKey(t, edPubPem); k.algo != "" || !k.isPublic {
		t.Errorf("Ed25519 public should be non-EC public, got %+v", k)
	}
}

// pkcs8Pem marshals a private key to a PKCS#8 PEM string.
func pkcs8Pem(t *testing.T, key any) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

// mustGetKey parses a PEM key or fails the test.
func mustGetKey(t *testing.T, pemStr string) ecdsaKey {
	t.Helper()
	k, err := ecdsaGetKey(pemStr)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// toHexByte formats n as a two-char hex byte (n < 256), for building DER in tests.
func toHexByte(n int) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[n>>4], digits[n&0xf]})
}

// TestECDSAGetKeyErrors covers key-parsing error paths shared by Sign/Verify.
func TestECDSAGetKeyErrors(t *testing.T) {
	// RSA PKCS1 private key -> "not an EC key" on Sign.
	if _, err := runOp(t, "ECDSA Sign", asciiText, rsaPrivPkcs1, "SHA-256", "ASN.1 HEX"); err == nil || err.Error() != "Provided key is not an EC key." {
		t.Errorf("RSA priv: got %v", err)
	}
	// Malformed PEM.
	if _, err := runOp(t, "ECDSA Sign", asciiText, "-----BEGIN EC PRIVATE KEY-----\nnot base64\n-----END EC PRIVATE KEY-----", "SHA-256", "ASN.1 HEX"); err == nil {
		t.Errorf("malformed key: expected error")
	}
}

// TestECDSAVerify covers verification of a known-good signature in every format.
func TestECDSAVerify(t *testing.T) {
	vfy := func(inFmt string) core.Recipe {
		return core.Recipe{{Op: "ECDSA Verify", Args: []any{inFmt, "SHA-256", p256Pub, asciiText, "Raw"}}}
	}
	runCases(t, []opCase{
		{"asn1 auto", sigASN1, "Verified OK", vfy("Auto")},
		{"p1363 auto", sigP1363, "Verified OK", vfy("Auto")},
		{"jws auto", sigJWS, "Verified OK", vfy("Auto")},
		{"json auto", sigJSON, "Verified OK", vfy("Auto")},
		{"asn1 explicit", sigASN1, "Verified OK", vfy("ASN.1 HEX")},
		// Wrong message -> verification failure (not an error).
		{
			"wrong message", sigASN1, "Verification Failure",
			core.Recipe{{Op: "ECDSA Verify", Args: []any{"Auto", "SHA-256", p256Pub, "different message", "Raw"}}},
		},
	})
}

// TestECDSASignVerify round-trips Sign then Verify across curves and digests.
func TestECDSASignVerify(t *testing.T) {
	rt := func(priv, pub, md string) core.Recipe {
		return core.Recipe{
			{Op: "ECDSA Sign", Args: []any{priv, md, "ASN.1 HEX"}},
			{Op: "ECDSA Verify", Args: []any{"ASN.1 HEX", md, pub, asciiText, "Raw"}},
		}
	}
	runCases(t, []opCase{
		{"P-256 MD5", asciiText, "Verified OK", rt(p256PrivPkcs1, p256Pub, "MD5")},
		{"P-256 SHA-1", asciiText, "Verified OK", rt(p256PrivPkcs1, p256Pub, "SHA-1")},
		{"P-256 SHA-256", asciiText, "Verified OK", rt(p256PrivPkcs1, p256Pub, "SHA-256")},
		{"P-256 SHA-384", asciiText, "Verified OK", rt(p256PrivPkcs1, p256Pub, "SHA-384")},
		{"P-256 SHA-512", asciiText, "Verified OK", rt(p256PrivPkcs1, p256Pub, "SHA-512")},
		{"P-256 PKCS8", asciiText, "Verified OK", rt(p256PrivPkcs8, p256Pub, "SHA-256")},
		{"P-384 SHA-384", asciiText, "Verified OK", rt(p384PrivPkcs8, p384Pub, "SHA-384")},
		{"P-521 SHA-512", asciiText, "Verified OK", rt(p521PrivPkcs8, p521Pub, "SHA-512")},
	})

	// Round-trip through the other output formats too.
	for _, outFmt := range []string{"P1363 HEX", "JSON Web Signature", "Raw JSON"} {
		got, err := core.Recipe{
			{Op: "ECDSA Sign", Args: []any{p256PrivPkcs1, "SHA-256", outFmt}},
			{Op: "ECDSA Verify", Args: []any{"Auto", "SHA-256", p256Pub, asciiText, "Raw"}},
		}.Execute(core.NewDish([]byte(asciiText), core.TypeString))
		if err != nil || got.String() != "Verified OK" {
			t.Errorf("round trip via %s: got %q, err %v", outFmt, got.String(), err)
		}
	}

	// Binary message via Hex/Base64 message formats: sign the raw bytes, then
	// verify with the message given in that format.
	binMsg := []byte{0x00, 0x01, 0x80, 0xff, 0x2a, 0x7f, 0xcd, 0xb2}
	for _, mf := range []struct{ format, msg string }{
		{"Hex", "000180ff2a7fcdb2"},
		{"Base64", base64.StdEncoding.EncodeToString(binMsg)},
	} {
		got, err := core.Recipe{
			{Op: "ECDSA Sign", Args: []any{p256PrivPkcs1, "SHA-256", "ASN.1 HEX"}},
			{Op: "ECDSA Verify", Args: []any{"ASN.1 HEX", "SHA-256", p256Pub, mf.msg, mf.format}},
		}.Execute(core.NewDish(binMsg, core.TypeString))
		if err != nil || got.String() != "Verified OK" {
			t.Errorf("binary round trip via %s: got %q, err %v", mf.format, got.String(), err)
		}
	}
}

// TestECDSASignErrors covers Sign's key-validation paths.
func TestECDSASignErrors(t *testing.T) {
	cases := []struct{ name, key, want string }{
		{"public key", p256Pub, "Provided key is not a private key."},
		{"RSA key", rsaPub, "Provided key is not an EC key."},
		{"empty key", "-----BEGIN EC PRIVATE KEY-----", "Please enter a private key."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "ECDSA Sign", asciiText, c.key, "SHA-256", "ASN.1 HEX")
			if err == nil || err.Error() != c.want {
				t.Fatalf("got %v, want %q", err, c.want)
			}
		})
	}
}

// TestECDSAVerifyErrors covers Verify's validation paths.
func TestECDSAVerifyErrors(t *testing.T) {
	missingR := `{"s":"00b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d"}`
	missingS := `{"r":"00e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127"}`
	cases := []struct {
		name, input, inFmt, key, want string
	}{
		{"private key", sigASN1, "ASN.1 HEX", p256PrivPkcs1, "Provided key is not a public key."},
		{"RSA key", sigASN1, "ASN.1 HEX", rsaPub, "Provided key is not an EC key."},
		{"missing r", missingR, "Auto", p256Pub, `No "r" value in the signature JSON`},
		{"missing s", missingS, "Auto", p256Pub, `No "s" value in the signature JSON`},
		{"empty key", sigASN1, "ASN.1 HEX", "-----BEGIN PUBLIC KEY-----", "Please enter a public key."},
		{"undetectable", "!!! not a signature !!!", "Auto", p256Pub, "Signature format could not be detected"},
		{"malformed key", sigASN1, "ASN.1 HEX", "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----", "asn1: "},
		{"non-hex ASN.1 sig", "nothexdigits", "ASN.1 HEX", p256Pub, "encoding/hex"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "ECDSA Verify", c.input, c.inFmt, "SHA-256", c.key, asciiText, "Raw")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want substring %q", err, c.want)
			}
		})
	}

	// A bad Base64 message triggers the message-decode error path.
	if _, err := runOp(t, "ECDSA Verify", sigASN1, "ASN.1 HEX", "SHA-256", p256Pub, "@@@bad", "Base64"); err == nil {
		t.Error("bad Base64 message: expected error")
	}
}

// TestECDSAGenerate checks each output format is structurally valid and that the
// generated key pair round-trips through Sign/Verify.
func TestECDSAGenerate(t *testing.T) {
	for _, curve := range []string{"P-256", "P-384", "P-521"} {
		pem, err := runOp(t, "Generate ECDSA Key Pair", "", curve, "PEM")
		if err != nil {
			t.Fatalf("%s PEM: %v", curve, err)
		}
		out := pem
		if !strings.Contains(out, "-----BEGIN PUBLIC KEY-----") || !strings.Contains(out, "-----BEGIN PRIVATE KEY-----") {
			t.Fatalf("%s PEM missing key blocks:\n%s", curve, out)
		}
		pub := out[strings.Index(out, "-----BEGIN PUBLIC KEY-----") : strings.Index(out, "-----END PUBLIC KEY-----")+len("-----END PUBLIC KEY-----")]
		priv := out[strings.Index(out, "-----BEGIN PRIVATE KEY-----") : strings.Index(out, "-----END PRIVATE KEY-----")+len("-----END PRIVATE KEY-----")]
		got, err := core.Recipe{
			{Op: "ECDSA Sign", Args: []any{priv, "SHA-256", "ASN.1 HEX"}},
			{Op: "ECDSA Verify", Args: []any{"ASN.1 HEX", "SHA-256", pub, asciiText, "Raw"}},
		}.Execute(core.NewDish([]byte(asciiText), core.TypeString))
		if err != nil || got.String() != "Verified OK" {
			t.Fatalf("%s generated key round trip: got %q err %v", curve, got.String(), err)
		}
	}

	// DER output is the raw private scalar hex (curve key length).
	der, err := runOp(t, "Generate ECDSA Key Pair", "", "P-256", "DER")
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := regexp.MatchString(`^[0-9a-f]{64}$`, der); !m {
		t.Errorf("P-256 DER not 64 hex chars: %q", der)
	}

	// JWK output parses and has the expected shape.
	jwk, err := runOp(t, "Generate ECDSA Key Pair", "", "P-256", "JWK")
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal([]byte(jwk), &parsed); err != nil {
		t.Fatalf("JWK not valid JSON: %v", err)
	}
	if len(parsed.Keys) != 2 || parsed.Keys[0]["kid"] != "PrivateKey" || parsed.Keys[1]["kid"] != "PublicKey" {
		t.Errorf("unexpected JWK shape: %s", jwk)
	}
	if parsed.Keys[0]["d"] == nil || parsed.Keys[1]["d"] != nil {
		t.Errorf("JWK d-field placement wrong: %s", jwk)
	}
}
