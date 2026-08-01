package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Fixtures transcribed verbatim from
// ../CyberChef/tests/operations/tests/PubKeyFromPrivKey.mjs. The RSA/EC/DSA
// PUBLIC KEY expected outputs are identical to the PubKeyFromCert fixtures and
// are reused here (pkcRSAPubKey/pkcECPubKey/pkcDSAPubKey).

const pkpRSAPrivPKCS1 = `-----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelMYKtboGLrk6ihtqFPZFRL
NcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQJAOJUpM0lv36MAQR3WAwsF
F7DOy+LnigteCvaNWiNVxZ6jByB5Qb7sall/Qlu9sFI0ZwrlVcKS0kldee7JTYlL
WQIhAP3UKEfOtpTgT1tYmdhaqjxqMfxBom0Ri+rt9ajlzs6vAiEA9L85B8/Gnb7p
6Af7/wpmafL277OV4X4xBfzMR+TUzHUCIBq+VLQkInaTH6lXL3ZtLwyIf9W9MJjf
RWeuRLjT5bM/AiBF7Kw6kx5Hy1fAtydEApCoDIaIjWJw/kC7WTJ0B+jUUQIgV6dw
NSyj0feakeD890gmId+lvl/w/3oUXiczqvl/N9o=
-----END RSA PRIVATE KEY-----`

const pkpRSAPrivPKCS8 = `-----BEGIN PRIVATE KEY-----
MIIBUwIBADANBgkqhkiG9w0BAQEFAASCAT0wggE5AgEAAkEA8qvQOnph0i3M5+Tp
ruZrsvgEXgud6Uxgq1ugYuuTqKG2oU9kVEs1wmLrwe+e3yy0ys/nS3qOrBZDYSMx
2SPp+wIDAQABAkA4lSkzSW/fowBBHdYDCwUXsM7L4ueKC14K9o1aI1XFnqMHIHlB
vuxqWX9CW72wUjRnCuVVwpLSSV157slNiUtZAiEA/dQoR862lOBPW1iZ2FqqPGox
/EGibRGL6u31qOXOzq8CIQD0vzkHz8advunoB/v/CmZp8vbvs5XhfjEF/MxH5NTM
dQIgGr5UtCQidpMfqVcvdm0vDIh/1b0wmN9FZ65EuNPlsz8CIEXsrDqTHkfLV8C3
J0QCkKgMhoiNYnD+QLtZMnQH6NRRAiBXp3A1LKPR95qR4Pz3SCYh36W+X/D/ehRe
JzOq+X832g==
-----END PRIVATE KEY-----`

const pkpECPrivSEC1 = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINtTjwUkgfAiSwqgcGAXWyE0ueIW6n2k395dmQZ3vGr4oAoGCCqGSM49
AwEHoUQDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJgusgcAE8H6810fkJ8ZmTNiCC
a6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END EC PRIVATE KEY-----`

const pkpECPrivPKCS8 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg21OPBSSB8CJLCqBw
YBdbITS54hbqfaTf3l2ZBne8avihRANCAAQNRzwDQQM0qgJgg9YwfPXJTOoTmYmC
6yBwATwfrzXR+QnxmZM2IIJrqwuBHa8PVU2HZ2KKtaAo8fg9Uwpq/l7p
-----END PRIVATE KEY-----`

const pkpDSAPrivTrad = `-----BEGIN DSA PRIVATE KEY-----
MIIDTQIBAAKCAQEAugtX67Pu8xhkIi7Bc2SBlrC8OBkAbVPsiIbTfRaCp6xE2dy6
hRA6YAkKK43OSNU1JjTytpt7H9EiRYgu/E/3VKXOED23352nzunbfx5zEZrWwPat
77LflE/sjE1Pffrfc23AYi1ncGqoCGgya7DDDFq7QTzPBih1NOXBNGnhBU+kwfV4
kJQF4TNxY/llnWVE2FHkzZOrNJTfJrn/hzdr2sjzqH1UhWDDzJW/DfLrJyXOMAub
ae0PmuKyR0xnqYyR9hDARsgZFMQtfoDstnSVfUKKyV6cX0BJvYTiDacZwxDS5IbM
Dhb2Z/rDCY9LqLBfn5C7+LqMKbmLft9DUfkcdQIdAMD5Sc8czo9kSB8V6fx2NmmX
9yXTkylaT52eKQUCggEANc5OkHDRYD5SKHEMb7qqozTmTFUxibPohFn/Tp99lIZU
U3oHHWLSze+TMzlXwawqmQqRKTtXx9NCcAgiUya3Vjt9/6eDv5N9ii0yu+8RiWiV
grTonZZahQDyIqnz6fny92iTL7Fma4aZy8ICiNUNsR5TbASpgTU/OscJI6g6IoHY
/h86HK4X6rETyeTNREiDZGZJKSpY6ReJbIrJ+lRoH7hwSKynhOEQ0Q+Z2e0TW8uH
avsQ4jGkTkPQ0DLbAUqp7eWN+ATpew9kqBUCC5ENWnN2YvpYNjsMz8FxLvbq/H6R
fnlAOSzO95yto2KueKvZCsH9rQlCMXpjrbIaYbsYXwKCAQBpN+w0N0b5IIAspXnl
J9yuB6ORk3j/5rZ+DUtTzW1YAJI6xjTcFQvN7FpVLkmLtXKUXF04R+sdGJ7VFwOb
0rbaL5vQzrqNkBrbgSzuzeloiG+7OLA6VeQtNbQh6OurrZFi9gY+qA5ciT9kQXyr
HudVXu956NDrooRxmv6JIVFvToaNiwe2vcgdkALw8HUbLFYof4SAE9jgU8EpxTp0
2e8HzvVSVa6yj1nnGhpzLPlEqF8TZvs9pTg2kIk3/zvWojMJoPyTALfbTjbAeiFM
MeKNK/CKOOJj23AVAZxpMSR6cUbrIcRdKDnhCTVkkxXUecAIUs6Mk10kSfkuiGl9
LjKjAhwpK4MOpkKEu+y308fZ+yZXypZW2m9Y/wOT0L4g
-----END DSA PRIVATE KEY-----`

const pkpDSAPrivPKCS8 = `-----BEGIN PRIVATE KEY-----
MIICXAIBADCCAjUGByqGSM44BAEwggIoAoIBAQC6C1frs+7zGGQiLsFzZIGWsLw4
GQBtU+yIhtN9FoKnrETZ3LqFEDpgCQorjc5I1TUmNPK2m3sf0SJFiC78T/dUpc4Q
PbffnafO6dt/HnMRmtbA9q3vst+UT+yMTU99+t9zbcBiLWdwaqgIaDJrsMMMWrtB
PM8GKHU05cE0aeEFT6TB9XiQlAXhM3Fj+WWdZUTYUeTNk6s0lN8muf+HN2vayPOo
fVSFYMPMlb8N8usnJc4wC5tp7Q+a4rJHTGepjJH2EMBGyBkUxC1+gOy2dJV9QorJ
XpxfQEm9hOINpxnDENLkhswOFvZn+sMJj0uosF+fkLv4uowpuYt+30NR+Rx1Ah0A
wPlJzxzOj2RIHxXp/HY2aZf3JdOTKVpPnZ4pBQKCAQA1zk6QcNFgPlIocQxvuqqj
NOZMVTGJs+iEWf9On32UhlRTegcdYtLN75MzOVfBrCqZCpEpO1fH00JwCCJTJrdW
O33/p4O/k32KLTK77xGJaJWCtOidllqFAPIiqfPp+fL3aJMvsWZrhpnLwgKI1Q2x
HlNsBKmBNT86xwkjqDoigdj+HzocrhfqsRPJ5M1ESINkZkkpKljpF4lsisn6VGgf
uHBIrKeE4RDRD5nZ7RNby4dq+xDiMaROQ9DQMtsBSqnt5Y34BOl7D2SoFQILkQ1a
c3Zi+lg2OwzPwXEu9ur8fpF+eUA5LM73nK2jYq54q9kKwf2tCUIxemOtshphuxhf
BB4CHCkrgw6mQoS77LfTx9n7JlfKllbab1j/A5PQviA=
-----END PRIVATE KEY-----`

const pkpEd25519Priv = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIC18vtoHINC8Mo9dTIqOrBs3J28ZvHrwzRq57g2kpV98
-----END PRIVATE KEY-----`

const pkpEd448Priv = `-----BEGIN PRIVATE KEY-----
MEcCAQAwBQYDK2VxBDsEOWdGJ06bDcWznJhBoQqPeTfsCe+AvBv1n7KfIGYzR4tv
1kcwHnbxlemnCMgqvbrRXaLuFUBysUZThA==
-----END PRIVATE KEY-----`

func pkpRecipe() core.Recipe {
	return core.Recipe{{Op: "Public Key from Private Key", Args: []any{}}}
}

func TestPubKeyFromPrivKeyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Public Key from Private Key: RSA PKCS#1", pkpRSAPrivPKCS1, crlf(pkcRSAPubKey + "\n"), pkpRecipe()},
		{"Public Key from Private Key: RSA PKCS#8", pkpRSAPrivPKCS8, crlf(pkcRSAPubKey + "\n"), pkpRecipe()},
		{"Public Key from Private Key: EC SEC1", pkpECPrivSEC1, crlf(pkcECPubKey + "\n"), pkpRecipe()},
		{"Public Key from Private Key: EC PKCS#8", pkpECPrivPKCS8, crlf(pkcECPubKey + "\n"), pkpRecipe()},
		{"Public Key from Private Key: DSA Traditional", pkpDSAPrivTrad, crlf(pkcDSAPubKey + "\n"), pkpRecipe()},
		{
			"Public Key from Private Key: Multiple keys",
			pkpRSAPrivPKCS8 + "\n" + pkpECPrivPKCS8,
			crlf(pkcRSAPubKey + "\n" + pkcECPubKey + "\n"), pkpRecipe(),
		},
	})
}

func TestPubKeyFromPrivKeyErrors(t *testing.T) {
	cases := []struct{ name, in, wantErr string }{
		{"Missing footer", pkpRSAPrivPKCS1[:len(pkpRSAPrivPKCS1)/2], "PEM footer '-----END RSA PRIVATE KEY-----' not found"},
		{"DSA PKCS#8", pkpDSAPrivPKCS8, "DSA Private Key in PKCS#8 is not supported"},
		{"Ed25519", pkpEd25519Priv, "Unsupported key type: Error: malformed PKCS8 private key(code:004)"},
		{"Ed448", pkpEd448Priv, "Unsupported key type: Error: malformed PKCS8 private key(code:004)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Public Key from Private Key", c.in)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}

// No matching PEM block yields empty output.
func TestPubKeyFromPrivKeyEmpty(t *testing.T) {
	out, err := runOp(t, "Public Key from Private Key", "no keys here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("got %q, want empty", out)
	}
}

// The malformed-input guards of the private-key helpers. Each parse failure is
// surfaced as CyberChef's "Unsupported key type: <err>" (or, for a bad PEM body,
// the PEM-decode error).
func TestPubKeyFromPrivKeyGuards(t *testing.T) {
	garbage := []byte{0xff, 0xff, 0xff}

	t.Run("bad PEM body", func(t *testing.T) {
		_, err := privKeyPublicKeyPEM("PRIVATE KEY", "-----BEGIN PRIVATE KEY-----\n@@@\n-----END PRIVATE KEY-----")
		if err == nil {
			t.Fatal("want error for bad PEM body")
		}
	})
	t.Run("bad RSA PKCS#1", func(t *testing.T) {
		if _, err := rsaPKCS1PublicKey(garbage); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("bad EC SEC1", func(t *testing.T) {
		if _, err := ecSEC1PublicKey(garbage); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("bad RSA/EC PKCS#8 body", func(t *testing.T) {
		if _, err := pkcs8RSAorECPublicKey(garbage); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("bad DSA traditional", func(t *testing.T) {
		if _, err := dsaTraditionalPublicKey(garbage); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("unreadable PKCS#8 alg OID", func(t *testing.T) {
		if oid := pkcs8AlgOID(garbage); oid != "" {
			t.Fatalf("got %q, want empty", oid)
		}
	})
}
