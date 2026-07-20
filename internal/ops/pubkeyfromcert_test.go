package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// crlf reproduces CyberChef's expectedOutput transform for these ops:
// (PEM + "\n").replace(/\r/g, "").replace(/\n/g, "\r\n").
func crlf(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", "\r\n")
}

// Fixtures transcribed verbatim from
// ../CyberChef/tests/operations/tests/PubKeyFromCert.mjs.

const pkcRSACert = `-----BEGIN CERTIFICATE-----
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

const pkcRSAPubKey = `-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----`

const pkcECCert = `-----BEGIN CERTIFICATE-----
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

const pkcECPubKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJ
gusgcAE8H6810fkJ8ZmTNiCCa6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END PUBLIC KEY-----`

const pkcDSACert = `-----BEGIN CERTIFICATE-----
MIIEXzCCBA2gAwIBAgIUYYcPJB8UQLzUnqkGJvs3J4RI0OgwCwYJYIZIAWUDBAMC
MBMxETAPBgNVBAMMCERTQSBUZXN0MB4XDTIzMTAxNTAwMjEzNVoXDTMzMTAxMjAw
MjEzNVowEzERMA8GA1UEAwwIRFNBIFRlc3QwggNCMIICNQYHKoZIzjgEATCCAigC
ggEBALoLV+uz7vMYZCIuwXNkgZawvDgZAG1T7IiG030WgqesRNncuoUQOmAJCiuN
zkjVNSY08rabex/RIkWILvxP91SlzhA9t9+dp87p238ecxGa1sD2re+y35RP7IxN
T33633NtwGItZ3BqqAhoMmuwwwxau0E8zwYodTTlwTRp4QVPpMH1eJCUBeEzcWP5
ZZ1lRNhR5M2TqzSU3ya5/4c3a9rI86h9VIVgw8yVvw3y6yclzjALm2ntD5riskdM
Z6mMkfYQwEbIGRTELX6A7LZ0lX1CislenF9ASb2E4g2nGcMQ0uSGzA4W9mf6wwmP
S6iwX5+Qu/i6jCm5i37fQ1H5HHUCHQDA+UnPHM6PZEgfFen8djZpl/cl05MpWk+d
nikFAoIBADXOTpBw0WA+UihxDG+6qqM05kxVMYmz6IRZ/06ffZSGVFN6Bx1i0s3v
kzM5V8GsKpkKkSk7V8fTQnAIIlMmt1Y7ff+ng7+TfYotMrvvEYlolYK06J2WWoUA
8iKp8+n58vdoky+xZmuGmcvCAojVDbEeU2wEqYE1PzrHCSOoOiKB2P4fOhyuF+qx
E8nkzURIg2RmSSkqWOkXiWyKyfpUaB+4cEisp4ThENEPmdntE1vLh2r7EOIxpE5D
0NAy2wFKqe3ljfgE6XsPZKgVAguRDVpzdmL6WDY7DM/BcS726vx+kX55QDkszvec
raNirnir2QrB/a0JQjF6Y62yGmG7GF8DggEFAAKCAQBpN+w0N0b5IIAspXnlJ9yu
B6ORk3j/5rZ+DUtTzW1YAJI6xjTcFQvN7FpVLkmLtXKUXF04R+sdGJ7VFwOb0rba
L5vQzrqNkBrbgSzuzeloiG+7OLA6VeQtNbQh6OurrZFi9gY+qA5ciT9kQXyrHudV
Xu956NDrooRxmv6JIVFvToaNiwe2vcgdkALw8HUbLFYof4SAE9jgU8EpxTp02e8H
zvVSVa6yj1nnGhpzLPlEqF8TZvs9pTg2kIk3/zvWojMJoPyTALfbTjbAeiFMMeKN
K/CKOOJj23AVAZxpMSR6cUbrIcRdKDnhCTVkkxXUecAIUs6Mk10kSfkuiGl9LjKj
o1MwUTAdBgNVHQ4EFgQUE+xZdvgiDIFWKQskMYnNaZ3iPHAwHwYDVR0jBBgwFoAU
E+xZdvgiDIFWKQskMYnNaZ3iPHAwDwYDVR0TAQH/BAUwAwEB/zALBglghkgBZQME
AwIDPwAwPAIcZbtf4+bjXEGQqNs6IglLrOgIjYF46q7qCNfXmQIcMKUtH3S6sDJE
3ds9eL+oC+HPFlfUNfUiU30aDA==
-----END CERTIFICATE-----`

const pkcDSAPubKey = `-----BEGIN PUBLIC KEY-----
MIIDQjCCAjUGByqGSM44BAEwggIoAoIBAQC6C1frs+7zGGQiLsFzZIGWsLw4GQBt
U+yIhtN9FoKnrETZ3LqFEDpgCQorjc5I1TUmNPK2m3sf0SJFiC78T/dUpc4QPbff
nafO6dt/HnMRmtbA9q3vst+UT+yMTU99+t9zbcBiLWdwaqgIaDJrsMMMWrtBPM8G
KHU05cE0aeEFT6TB9XiQlAXhM3Fj+WWdZUTYUeTNk6s0lN8muf+HN2vayPOofVSF
YMPMlb8N8usnJc4wC5tp7Q+a4rJHTGepjJH2EMBGyBkUxC1+gOy2dJV9QorJXpxf
QEm9hOINpxnDENLkhswOFvZn+sMJj0uosF+fkLv4uowpuYt+30NR+Rx1Ah0AwPlJ
zxzOj2RIHxXp/HY2aZf3JdOTKVpPnZ4pBQKCAQA1zk6QcNFgPlIocQxvuqqjNOZM
VTGJs+iEWf9On32UhlRTegcdYtLN75MzOVfBrCqZCpEpO1fH00JwCCJTJrdWO33/
p4O/k32KLTK77xGJaJWCtOidllqFAPIiqfPp+fL3aJMvsWZrhpnLwgKI1Q2xHlNs
BKmBNT86xwkjqDoigdj+HzocrhfqsRPJ5M1ESINkZkkpKljpF4lsisn6VGgfuHBI
rKeE4RDRD5nZ7RNby4dq+xDiMaROQ9DQMtsBSqnt5Y34BOl7D2SoFQILkQ1ac3Zi
+lg2OwzPwXEu9ur8fpF+eUA5LM73nK2jYq54q9kKwf2tCUIxemOtshphuxhfA4IB
BQACggEAaTfsNDdG+SCALKV55SfcrgejkZN4/+a2fg1LU81tWACSOsY03BULzexa
VS5Ji7VylFxdOEfrHRie1RcDm9K22i+b0M66jZAa24Es7s3paIhvuziwOlXkLTW0
Iejrq62RYvYGPqgOXIk/ZEF8qx7nVV7veejQ66KEcZr+iSFRb06GjYsHtr3IHZAC
8PB1GyxWKH+EgBPY4FPBKcU6dNnvB871UlWuso9Z5xoacyz5RKhfE2b7PaU4NpCJ
N/871qIzCaD8kwC32042wHohTDHijSvwijjiY9twFQGcaTEkenFG6yHEXSg54Qk1
ZJMV1HnACFLOjJNdJEn5LohpfS4yow==
-----END PUBLIC KEY-----`

const pkcEd25519Cert = `-----BEGIN CERTIFICATE-----
MIIBQjCB9aADAgECAhRjPJhrdNco5LzpsIs0vSLLaZaZ0DAFBgMrZXAwFzEVMBMG
A1UEAwwMRWQyNTUxOSBUZXN0MB4XDTIzMTAxNTAwMjMwOFoXDTMzMTAxMjAwMjMw
OFowFzEVMBMGA1UEAwwMRWQyNTUxOSBUZXN0MCowBQYDK2VwAyEAELP6AflXwsuZ
5q4NDIO0LP2iCdKRvds4nwsUmRhOw3ijUzBRMB0GA1UdDgQWBBRfxS9q0IemWxkH
4mwAwzr9dQx2xzAfBgNVHSMEGDAWgBRfxS9q0IemWxkH4mwAwzr9dQx2xzAPBgNV
HRMBAf8EBTADAQH/MAUGAytlcANBAI/+03iVq4yJ+DaLVs61w41cVX2UxKvquSzv
lllkpkclM9LH5dLrw4ArdTjS9zAjzY/02WkphHhICHXt3KqZTwI=
-----END CERTIFICATE-----`

const pkcEd448Cert = `-----BEGIN CERTIFICATE-----
MIIBijCCAQqgAwIBAgIUZaCS7zEjOnQ7O4KUFym6fJF5vl8wBQYDK2VxMBUxEzAR
BgNVBAMMCkVkNDQ4IFRlc3QwHhcNMjMxMDE1MDAyMzI1WhcNMzMxMDEyMDAyMzI1
WjAVMRMwEQYDVQQDDApFZDQ0OCBUZXN0MEMwBQYDK2VxAzoAVN8kG0TMVyGOu/Ov
BTe8H0Wi4HJrQAlSv4XLwJbkuoi4EeRlEHQwXsNYLZTtY2Jra6AWhbVYYaEAo1Mw
UTAdBgNVHQ4EFgQUJFrepAf9YXrmDMSAzrMeYQmosd0wHwYDVR0jBBgwFoAUJFre
pAf9YXrmDMSAzrMeYQmosd0wDwYDVR0TAQH/BAUwAwEB/zAFBgMrZXEDcwA+YiZj
puFr2aogfV1qg/ixk7qLi25BbKVNR6+7PEUjo7+4yBn9qnLbAHUGnHn7E96pSey9
VkLqpoDNMRcM3Eb6h3AJpQM0oxGj8q9arjDXqJkXgaO2e0tVn8KKVfy7S8qO72Kd
rWzZowcOjnWKhXm7JgA=
-----END CERTIFICATE-----`

func pkcRecipe() core.Recipe {
	return core.Recipe{{Op: "Public Key from Certificate", Args: []any{}}}
}

func TestPubKeyFromCertFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Public Key from Certificate: RSA", pkcRSACert, crlf(pkcRSAPubKey + "\n"), pkcRecipe()},
		{"Public Key from Certificate: EC", pkcECCert, crlf(pkcECPubKey + "\n"), pkcRecipe()},
		{"Public Key from Certificate: DSA", pkcDSACert, crlf(pkcDSAPubKey + "\n"), pkcRecipe()},
		{
			"Public Key from Certificate: Multiple certificates",
			pkcRSACert + "\n" + pkcECCert,
			crlf(pkcRSAPubKey + "\n" + pkcECPubKey + "\n"), pkcRecipe(),
		},
	})
}

func TestPubKeyFromCertErrors(t *testing.T) {
	cases := []struct{ name, in, wantErr string }{
		{"Missing footer", pkcRSACert[:len(pkcRSACert)/2], "PEM footer '-----END CERTIFICATE-----' not found"},
		{"Ed25519", pkcEd25519Cert, "Unsupported public key type"},
		{"Ed448", pkcEd448Cert, "Unsupported public key type"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Public Key from Certificate", c.in)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}

// Empty input yields empty output (the regex simply never matches).
func TestPubKeyFromCertEmpty(t *testing.T) {
	out, err := runOp(t, "Public Key from Certificate", "no certificates here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("got %q, want empty", out)
	}
}

// A certificate block whose body is not valid base64 fails PEM decoding, which
// the op reports as an unsupported public key.
func TestPubKeyFromCertBadPEM(t *testing.T) {
	in := certHeader + "\n@@@not base64@@@\n" + certFooter
	if _, err := runOp(t, "Public Key from Certificate", in); err == nil || err.Error() != errUnsupportedPubKeyType.Error() {
		t.Fatalf("got %v, want %q", err, errUnsupportedPubKeyType)
	}
}

// certSPKI's malformed-DER guards, exercised with crafted hex.
func TestCertSPKIGuards(t *testing.T) {
	cases := []struct {
		name string
		h    string
	}{
		{"empty (root parse fails)", ""},
		{"tbs parse fails", derTLV("30", derTLV("30", "3080"))},
		{"too few TBS fields", derTLV("30", derTLV("30", derTLV("02", "01")+derTLV("02", "02")))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := certSPKI(c.h); ok {
				t.Fatalf("certSPKI(%q) = ok, want not ok", c.h)
			}
		})
	}
}

// supportedPubKeyOID's malformed-SPKI guards.
func TestSupportedPubKeyOIDGuards(t *testing.T) {
	cases := []struct {
		name string
		spki string
	}{
		{"not a sequence", "0500"},
		{"empty algorithm sequence", derTLV("30", "0500")},
		{"unsupported algorithm OID", derTLV("30", derTLV("30", derTLV("06", "2b6570")))}, // Ed25519 OID
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if supportedPubKeyOID(c.spki) {
				t.Fatalf("supportedPubKeyOID(%q) = true, want false", c.spki)
			}
		})
	}
}
