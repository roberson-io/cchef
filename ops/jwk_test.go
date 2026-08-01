package ops

import (
	"bytes"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// JWK/PEM test vectors transcribed from CyberChef's
// tests/operations/tests/JWK.mjs (RSA-512 and EC P-256 key pairs).

const (
	jwkRSAN  = "8qvQOnph0i3M5-TpruZrsvgEXgud6Uxgq1ugYuuTqKG2oU9kVEs1wmLrwe-e3yy0ys_nS3qOrBZDYSMx2SPp-w"
	jwkRSAE  = "AQAB"
	jwkRSAD  = "OJUpM0lv36MAQR3WAwsFF7DOy-LnigteCvaNWiNVxZ6jByB5Qb7sall_Qlu9sFI0ZwrlVcKS0kldee7JTYlLWQ"
	jwkRSAP  = "_dQoR862lOBPW1iZ2FqqPGox_EGibRGL6u31qOXOzq8"
	jwkRSAQ  = "9L85B8_Gnb7p6Af7_wpmafL277OV4X4xBfzMR-TUzHU"
	jwkRSADP = "Gr5UtCQidpMfqVcvdm0vDIh_1b0wmN9FZ65EuNPlsz8"
	jwkRSADQ = "ReysOpMeR8tXwLcnRAKQqAyGiI1icP5Au1kydAfo1FE"
	jwkRSAQI = "V6dwNSyj0feakeD890gmId-lvl_w_3oUXiczqvl_N9o"

	jwkECX = "DUc8A0EDNKoCYIPWMHz1yUzqE5mJgusgcAE8H6810fk"
	jwkECY = "CfGZkzYggmurC4Edrw9VTYdnYoq1oCjx-D1TCmr-Xuk"
	jwkECD = "21OPBSSB8CJLCqBwYBdbITS54hbqfaTf3l2ZBne8avg"
)

// Expected JWK JSON strings (JSON.stringify field order).
const (
	jwkRSAPrivJSON = `{"kty":"RSA","n":"` + jwkRSAN + `","e":"` + jwkRSAE + `","d":"` + jwkRSAD +
		`","p":"` + jwkRSAP + `","q":"` + jwkRSAQ + `","dp":"` + jwkRSADP +
		`","dq":"` + jwkRSADQ + `","qi":"` + jwkRSAQI + `"}`
	jwkRSAPubJSON = `{"kty":"RSA","n":"` + jwkRSAN + `","e":"` + jwkRSAE + `"}`
	jwkECPrivJSON = `{"kty":"EC","crv":"P-256","x":"` + jwkECX + `","y":"` + jwkECY + `","d":"` + jwkECD + `"}`
	jwkECPubJSON  = `{"kty":"EC","crv":"P-256","x":"` + jwkECX + `","y":"` + jwkECY + `"}`
)

const (
	rsaPrivPEM1 = `-----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelMYKtboGLrk6ihtqFPZFRL
NcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQJAOJUpM0lv36MAQR3WAwsF
F7DOy+LnigteCvaNWiNVxZ6jByB5Qb7sall/Qlu9sFI0ZwrlVcKS0kldee7JTYlL
WQIhAP3UKEfOtpTgT1tYmdhaqjxqMfxBom0Ri+rt9ajlzs6vAiEA9L85B8/Gnb7p
6Af7/wpmafL277OV4X4xBfzMR+TUzHUCIBq+VLQkInaTH6lXL3ZtLwyIf9W9MJjf
RWeuRLjT5bM/AiBF7Kw6kx5Hy1fAtydEApCoDIaIjWJw/kC7WTJ0B+jUUQIgV6dw
NSyj0feakeD890gmId+lvl/w/3oUXiczqvl/N9o=
-----END RSA PRIVATE KEY-----`

	rsaPrivPEM8 = `-----BEGIN PRIVATE KEY-----
MIIBUwIBADANBgkqhkiG9w0BAQEFAASCAT0wggE5AgEAAkEA8qvQOnph0i3M5+Tp
ruZrsvgEXgud6Uxgq1ugYuuTqKG2oU9kVEs1wmLrwe+e3yy0ys/nS3qOrBZDYSMx
2SPp+wIDAQABAkA4lSkzSW/fowBBHdYDCwUXsM7L4ueKC14K9o1aI1XFnqMHIHlB
vuxqWX9CW72wUjRnCuVVwpLSSV157slNiUtZAiEA/dQoR862lOBPW1iZ2FqqPGox
/EGibRGL6u31qOXOzq8CIQD0vzkHz8advunoB/v/CmZp8vbvs5XhfjEF/MxH5NTM
dQIgGr5UtCQidpMfqVcvdm0vDIh/1b0wmN9FZ65EuNPlsz8CIEXsrDqTHkfLV8C3
J0QCkKgMhoiNYnD+QLtZMnQH6NRRAiBXp3A1LKPR95qR4Pz3SCYh36W+X/D/ehRe
JzOq+X832g==
-----END PRIVATE KEY-----`

	rsaPubPEM1 = `-----BEGIN RSA PUBLIC KEY-----
MEgCQQDyq9A6emHSLczn5Omu5muy+AReC53pTGCrW6Bi65OoobahT2RUSzXCYuvB
757fLLTKz+dLeo6sFkNhIzHZI+n7AgMBAAE=
-----END RSA PUBLIC KEY-----`

	rsaPubPEM8 = `-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----`

	rsaCertPEM = `-----BEGIN CERTIFICATE-----
MIIBfTCCASegAwIBAgIUeisK5Nwss2DGg5PCs4uSxxXyyNkwDQYJKoZIhvcNAQEL
BQAwEzERMA8GA1UEAwwIUlNBIHRlc3QwHhcNMjExMTE5MTcyMDI2WhcNMzExMTE3
MTcyMDI2WjATMREwDwYDVQQDDAhSU0EgdGVzdDBcMA0GCSqGSIb3DQEBAQUAA0sA
MEgCQQDyq9A6emHSLczn5Omu5muy+AReC53pTGCrW6Bi65OoobahT2RUSzXCYuvB
757fLLTKz+dLeo6sFkNhIzHZI+n7AgMBAAGjUzBRMB0GA1UdDgQWBBRO+jvkqq5p
pnQgwMMnRoun6e7eiTAfBgNVHSMEGDAWgBRO+jvkqq5ppnQgwMMnRoun6e7eiTAP
BgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA0EAR/5HAZM5qBhU/ezDUIFx
gmUGoFbIb5kJD41YCnaSdrgWglh4He4melSs42G/oxBBjuCJ0bUpqWnLl+lJkv1z
IA==
-----END CERTIFICATE-----`

	ecPrivPEM1 = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINtTjwUkgfAiSwqgcGAXWyE0ueIW6n2k395dmQZ3vGr4oAoGCCqGSM49
AwEHoUQDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJgusgcAE8H6810fkJ8ZmTNiCC
a6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END EC PRIVATE KEY-----`

	ecPrivPEM8 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg21OPBSSB8CJLCqBw
YBdbITS54hbqfaTf3l2ZBne8avihRANCAAQNRzwDQQM0qgJgg9YwfPXJTOoTmYmC
6yBwATwfrzXR+QnxmZM2IIJrqwuBHa8PVU2HZ2KKtaAo8fg9Uwpq/l7p
-----END PRIVATE KEY-----`

	ecPubPEM8 = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJ
gusgcAE8H6810fkJ8ZmTNiCCa6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END PUBLIC KEY-----`

	ecCertPEM = `-----BEGIN CERTIFICATE-----
MIIBfzCCASWgAwIBAgIUK4H8J3Hr7NpRLPrACj8Pje4JJJ0wCgYIKoZIzj0EAwIw
FTETMBEGA1UEAwwKUC0yNTYgdGVzdDAeFw0yMTExMTkxNzE5NDVaFw0zMTExMTcx
NzE5NDVaMBUxEzARBgNVBAMMClAtMjU2IHRlc3QwWTATBgcqhkjOPQIBBggqhkjO
PQMBBwNCAAQNRzwDQQM0qgJgg9YwfPXJTOoTmYmC6yBwATwfrzXR+QnxmZM2IIJr
qwuBHa8PVU2HZ2KKtaAo8fg9Uwpq/l7po1MwUTAdBgNVHQ4EFgQU/SxodXrpkybM
gcIgkxnRKd7HMzowHwYDVR0jBBgwFoAU/SxodXrpkybMgcIgkxnRKd7HMzowDwYD
VR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgNIADBFAiBU9PrOa/kXCpTTBInRf/sN
ac2iDHmbdpWzcXI+xLKNYAIhAIRR1LRSHVwOTLQ/iBXd+8LCkm5aTB27RW46LN80
ylxt
-----END CERTIFICATE-----`

	dsaPrivPEM = `-----BEGIN DSA PRIVATE KEY-----
MIIBuwIBAAKBgQCkFEttBrPHEJRgcvaT8HbZs9h1pVQLHhn2F452izusRox1czMM
IC8Z7YQiM1pt6bgEmf0h8ldx6UFT0YL9JWSbyBy1U5pHKfnz/xjeg7ZMReL4F0/T
Gwmu4ercqfM//TmEg9nL3nDxb4WmF2al/SmHN3qlzYmYaIDEFfEuu8vWbwIVAMOq
7pqQiMGUu6uJY/nQTWW0c3IfAoGARWryStp2AElj538qN9tWRuyobRA93Q1ujrdM
EqsqVpMZd1a8qtRyMaZVVdB7N3EweNUuFOoSAp10s/SQEH9qhVo6NwvzhB7lEtm4
5FjWW9+9WCuuFOGZpTy8PSFAvQcfUqunP/DeaDliNmgKci+n0nfIBakuQn10Zmqk
vGu8NZICgYBUsoQeXSJ19e6XZenk6G8wVI3yXFqnRAwb6s7sAVoPwfDCsOXTxC7W
Mlfz0HcYMiifFKEd28NnuAZ2e0ngyPHsb9s5phzTgRfO3GFzOjsjwgx3DmQI2Ck2
yOWHSAtaNhH4DoBZEyNsb1akiB50vx9b09EHN4weqbgAu743NMDHRQIVAIG5uiiO
OnWUYieHAiVIPkBCrYUd
-----END DSA PRIVATE KEY-----`

	// Ed25519 public JWK — an unsupported key type (RFC 8037 A.2).
	jwkEd25519JSON = `{"kty":"OKP","crv":"Ed25519","x":"11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"}`
)

// pemCRLF mirrors the fixture's expected-output normalisation:
// CyberChef emits PEM with CRLF line endings and a trailing CRLF, whereas the
// fixture PEM constants use bare LF and no trailing newline.
func pemCRLF(pems ...string) string {
	var joined strings.Builder
	for _, p := range pems {
		joined.WriteString(p + "\n")
	}
	return strings.ReplaceAll(joined.String(), "\n", "\r\n")
}

func TestPEMToJWK(t *testing.T) {
	one := func(name, in, want string) opCase {
		return opCase{name, in, want, core.Recipe{{Op: "PEM to JWK"}}}
	}
	runCases(t, []opCase{
		one("RSA Private Key PKCS1", rsaPrivPEM1, jwkRSAPrivJSON),
		one("RSA Private Key PKCS8", rsaPrivPEM8, jwkRSAPrivJSON),
		one("RSA Public Key PKCS8", rsaPubPEM8, jwkRSAPubJSON),
		one("Certificate with RSA Public Key", rsaCertPEM, jwkRSAPubJSON),
		one("EC Private Key PKCS1", ecPrivPEM1, jwkECPrivJSON),
		one("EC Private Key PKCS8", ecPrivPEM8, jwkECPrivJSON),
		one("EC Public Key", ecPubPEM8, jwkECPubJSON),
		one("Certificate with EC Public Key", ecCertPEM, jwkECPubJSON),
	})
}

func TestJWKToPEM(t *testing.T) {
	one := func(name, in, want string) opCase {
		return opCase{name, in, want, core.Recipe{{Op: "JWK to PEM"}}}
	}
	runCases(t, []opCase{
		one("RSA Private Key", jwkRSAPrivJSON, pemCRLF(rsaPrivPEM8)),
		one("RSA Public Key", jwkRSAPubJSON, pemCRLF(rsaPubPEM8)),
		one("EC Private Key", jwkECPrivJSON, pemCRLF(ecPrivPEM8)),
		one("EC Public Key", jwkECPubJSON, pemCRLF(ecPubPEM8)),
		one("Array of keys", `[`+jwkRSAPubJSON+`,`+jwkECPubJSON+`]`, pemCRLF(rsaPubPEM8, ecPubPEM8)),
		one("JSON Web Key Set", `{"keys":[`+jwkRSAPubJSON+`,`+jwkECPubJSON+`]}`, pemCRLF(rsaPubPEM8, ecPubPEM8)),
	})
}

func TestPEMToJWKErrors(t *testing.T) {
	for _, c := range []struct{ name, in, wantErr string }{
		{"Missing footer", rsaPrivPEM1[:len(rsaPrivPEM1)/2], "PEM footer '-----END RSA PRIVATE KEY-----' not found"},
		{"DSA not supported", dsaPrivPEM, "DSA keys are not supported for JWK"},
		{"RSA Public Key PKCS1", rsaPubPEM1, "Unsupported RSA public key format. Only PKCS#8 is supported."},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "PEM to JWK", c.in)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}

func TestJWKToPEMErrors(t *testing.T) {
	for _, c := range []struct{ name, in, wantErr string }{
		{"not a JWK", `"foobar"`, "Input is not a JSON Web Key"},
		{"unsupported key type", jwkEd25519JSON, "Unsupported JWK key type 'OKP'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWK to PEM", c.in)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}

// EC keys on non-P-256 curves, used to exercise the curve helpers.
const (
	ecP384Priv = `-----BEGIN PRIVATE KEY-----
MIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDDzT0lHW4U4FCyn/Sxp
ayVCo1f7im/rJGjtycQ1aVTUhD2br48Uk/Vw69IRQ8ixe6WhZANiAARsu9wYLeeG
UlFW9p9HKS8xIPvJxrU9+FoeQoaeHKEphPV9LePwHeu2GBU5xMax5WTuX114dZXm
F+I42u4AmKbpo6DMO8ACnnLddeGEXo5lIBvEd5ICk3UjPt/vTXIMY8A=
-----END PRIVATE KEY-----`

	ecP521Pub = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBGflnfApqt6Xm2Py2XUtFeFQjstiu
RBoZ+t6hOSIrbCOltAJeRLd70HEt3Cac2WvijNqU/ShY07FNtG3SdAkgfQ4BFTo8
XOt1VLcArT5/EmAhsqkAN6ijxVwi2T2oSWYcY98eQXEObeH0/YDTUFnI1O5BZbq/
8hKEBIBUVTpBnSTTLC8=
-----END PUBLIC KEY-----`

	// P-224 is a valid NIST curve Go can parse but jsrsasign/JWK does not name,
	// so it hits the curve-name error path.
	ecP224Pub = `-----BEGIN PUBLIC KEY-----
ME4wEAYHKoZIzj0CAQYFK4EEACEDOgAEP9ude7G3a7s+C2aNPWAA+SJU/zMpSnyM
vn6fQ48zt8W3JwveQP/487ZAbVcgoZG2Za4gmFsk8ME=
-----END PUBLIC KEY-----`

	ed25519Pub = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAPaIJO1f9JJQH2rH//b28gTgJ1OFL/JTlkZUHjiXIoJ4=
-----END PUBLIC KEY-----`

	ecP224Priv = `-----BEGIN PRIVATE KEY-----
MHgCAQAwEAYHKoZIzj0CAQYFK4EEACEEYTBfAgEBBBwEojOPFcdPKUK1y2LG6CAO
wQi6LTvsNQO48akcoTwDOgAEx7dSNx/M8zqBHfN9q9wdK0V6m0BtZBXR63/0kj0B
krhnqTOYfrRISLA2WynLRqJCPSBpbaQdEeA=
-----END PRIVATE KEY-----`
)

// TestPEMToJWKMultipleBlocks converts an input with two PEM blocks, exercising
// the newline separator between emitted JWKs.
func TestPEMToJWKMultipleBlocks(t *testing.T) {
	runCases(t, []opCase{
		{
			"two public keys", rsaPubPEM8 + "\n" + ecPubPEM8, jwkRSAPubJSON + "\n" + jwkECPubJSON,
			core.Recipe{{Op: "PEM to JWK"}},
		},
	})
}

// TestJWKRoundTripCurves round-trips P-384 and P-521 keys through
// PEM -> JWK -> PEM, exercising the P-384/P-521 curve branches in both
// directions and the coordinate left-padding.
func TestJWKRoundTripCurves(t *testing.T) {
	one := func(name, pemIn string) opCase {
		return opCase{
			name, pemIn, pemCRLF(pemIn),
			core.Recipe{{Op: "PEM to JWK"}, {Op: "JWK to PEM"}},
		}
	}
	runCases(t, []opCase{
		one("P-384 private", ecP384Priv),
		one("P-521 public", ecP521Pub),
	})
}

// TestPEMToJWKBranchErrors covers the remaining error/defensive paths of
// PEM to JWK. Messages for CyberChef-defined errors are asserted exactly; the
// Go-internal parse/decode failures only assert a non-nil error.
func TestPEMToJWKBranchErrors(t *testing.T) {
	corruptRSA := `-----BEGIN RSA PRIVATE KEY-----
bm90IHZhbGlkIGRlcg==
-----END RSA PRIVATE KEY-----`
	corruptCert := `-----BEGIN CERTIFICATE-----
bm90IHZhbGlkIGRlcg==
-----END CERTIFICATE-----`
	badBody := "-----BEGIN PUBLIC KEY-----\n!!!not base64!!!\n-----END PUBLIC KEY-----"

	for _, c := range []struct {
		name, in, wantErr string // wantErr "" => just assert an error occurred
	}{
		{"unsupported PEM type", "-----BEGIN FOO-----\nAAAA\n-----END FOO-----", "Unsupported PEM type 'FOO'"},
		{"unsupported KEY type", "-----BEGIN ENCRYPTED PRIVATE KEY-----\nAAAA\n-----END ENCRYPTED PRIVATE KEY-----", "Unsupported PEM type 'ENCRYPTED PRIVATE KEY'"},
		{"unsupported public curve", ecP224Pub, "unsupported curve name for JWT"},
		{"unsupported private curve", ecP224Priv, "unsupported curve name for JWT"},
		{"unsupported key type", ed25519Pub, "Unsupported key type for JWK"},
		{"corrupt key DER", corruptRSA, ""},
		{"corrupt certificate DER", corruptCert, ""},
		{"non-base64 cert body", "-----BEGIN CERTIFICATE-----\n!!!\n-----END CERTIFICATE-----", ""},
		{"non-base64 body", badBody, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "PEM to JWK", c.in)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if c.wantErr != "" && err.Error() != c.wantErr {
				t.Fatalf("got %q, want %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestJWKToPEMBranchErrors covers the remaining error paths of JWK to PEM.
func TestJWKToPEMBranchErrors(t *testing.T) {
	for _, c := range []struct{ name, in, wantErr string }{
		{"kty not a string", `{"kty":123}`, "Invalid JWK format"},
		{"array element not object", `["foo"]`, "Invalid JWK format"},
		{"RSA missing modulus", `{"kty":"RSA","e":"AQAB"}`, "Invalid JWK format"},
		{"RSA invalid modulus base64", `{"kty":"RSA","n":"!!!","e":"AQAB"}`, "Invalid JWK format"},
		{"RSA invalid exponent base64", `{"kty":"RSA","n":"AQAB","e":"!!!"}`, "Invalid JWK format"},
		{"RSA private invalid d", `{"kty":"RSA","n":"AQAB","e":"AQAB","d":"!!!"}`, "Invalid JWK format"},
		{"RSA private invalid p", `{"kty":"RSA","n":"AQAB","e":"AQAB","d":"AQAB","p":"!!!","q":"AQAB"}`, "Invalid JWK format"},
		{"RSA private invalid q", `{"kty":"RSA","n":"AQAB","e":"AQAB","d":"AQAB","p":"AQAB","q":"!!!"}`, "Invalid JWK format"},
		{"RSA private missing prime", `{"kty":"RSA","n":"AQAB","e":"AQAB","d":"AQAB"}`, "Invalid JWK format"},
		{"EC unsupported curve", `{"kty":"EC","crv":"P-999","x":"AA","y":"AA"}`, "unsupported curve name for JWT: P-999"},
		{"EC invalid x base64", `{"kty":"EC","crv":"P-256","x":"!!!","y":"AA"}`, "Invalid JWK format"},
		{"EC private invalid d", `{"kty":"EC","crv":"P-256","x":"` + jwkECX + `","y":"` + jwkECY + `","d":"!!!"}`, "Invalid JWK format"},
		{"EC missing coordinate", `{"kty":"EC","crv":"P-256","x":"AA"}`, "Invalid JWK format"},
		// An off-curve point fails at ParseUncompressedPublicKey; an on-curve
		// point with a zero scalar fails at ParseRawPrivateKey. Both are
		// Go-internal errors whose text varies by version, so only non-nil is
		// asserted.
		{"EC off-curve public", `{"kty":"EC","crv":"P-256","x":"AQ","y":"AQ"}`, ""},
		{"EC zero private scalar", `{"kty":"EC","crv":"P-256","x":"` + jwkECX + `","y":"` + jwkECY + `","d":"AA"}`, ""},
		{"invalid JSON", `{not json`, "Input is not a JSON Web Key"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWK to PEM", c.in)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if c.wantErr != "" && err.Error() != c.wantErr {
				t.Fatalf("got %q, want %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestLeftPadBytes directly exercises both branches of the coordinate padder.
func TestLeftPadBytes(t *testing.T) {
	if got := leftPadBytes([]byte{0x01, 0x02, 0x03}, 3); !bytes.Equal(got, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("already-full: got %x", got)
	}
	if got := leftPadBytes([]byte{0x01}, 4); !bytes.Equal(got, []byte{0, 0, 0, 0x01}) {
		t.Fatalf("padded: got %x", got)
	}
}
