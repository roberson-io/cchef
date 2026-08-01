package ops

import (
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

// JWT fixtures transcribed from
// ../CyberChef/tests/operations/tests/JWT{Decode,Sign,Verify}.mjs.

// jwtPayloadObject is JSON.stringify({String, Number, iat}, null, 4) — the shared
// input/output object used across the upstream JWT fixtures.
const jwtPayloadObject = `{
    "String": "SomeString",
    "Number": 42,
    "iat": 1
}`

const jwtHSKey = "secret_cat"

const jwtRSKey1024 = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQDdlatRjRjogo3WojgGHFHYLugdUWAY9iR3fy4arWNA1KoS8kVw
33cJibXr8bvwUAUparCwlvdbH6dvEOfou0/gCFQsHUfQrSDv+MuSUMAe8jzKE4qW
+jK+xQU9a03GUnKHkkle+Q0pX/g6jXZ7r1/xAK5Do2kQ+X5xK9cipRgEKwIDAQAB
AoGAD+onAtVye4ic7VR7V50DF9bOnwRwNXrARcDhq9LWNRrRGElESYYTQ6EbatXS
3MCyjjX2eMhu/aF5YhXBwkppwxg+EOmXeh+MzL7Zh284OuPbkglAaGhV9bb6/5Cp
uGb1esyPbYW+Ty2PC0GSZfIXkXs76jXAu9TOBvD0ybc2YlkCQQDywg2R/7t3Q2OE
2+yo382CLJdrlSLVROWKwb4tb2PjhY4XAwV8d1vy0RenxTB+K5Mu57uVSTHtrMK0
GAtFr833AkEA6avx20OHo61Yela/4k5kQDtjEf1N0LfI+BcWZtxsS3jDM3i1Hp0K
Su5rsCPb8acJo5RO26gGVrfAsDcIXKC+bQJAZZ2XIpsitLyPpuiMOvBbzPavd4gY
6Z8KWrfYzJoI/Q9FuBo6rKwl4BFoToD7WIUS+hpkagwWiz+6zLoX1dbOZwJACmH5
fSSjAkLRi54PKJ8TFUeOP15h9sQzydI8zJU+upvDEKZsZc/UhT/SySDOxQ4G/523
Y0sz/OZtSWcol/UMgQJALesy++GdvoIDLfJX5GBQpuFgFenRiRDabxrE9MNUZ2aP
FaFp+DyAe+b4nDwuJaW2LURbr8AEZga7oQj0uYxcYw==
-----END RSA PRIVATE KEY-----`

const jwtRSKey2048 = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAk0VOoksAblwP82DALTG6xGC86Hfho3nChbcPGWyqn+ScfHBF
cg3SeKyy6aWCyLcKfNwE5cPYzuYvVBsZyIrdfFOuV90D/aRYbuw6UkKR3cmmy9qE
qvu05dogvc0BcmkwbC37Q8JnsZBRcosoLGgTFxcK+LXdsG7DukajpsGesxQjOLb2
1jnx+ypzx74xvj7grqlXkxeDKr22q7QkO3A1ApoOuJRAU+SjEEZmqdXzRery2RWx
hkWbCXuQw4PnW5Lh3Wwabnu7XKVIa6wJa1pqL2IAxmlZ0bvGTfjtO5ggNfgJk5V4
bGSOXnsplpG71AWMrK2q6NqHjFIE1szEycUKrwIDAQABAoIBAAivyt6Zy/G2g8kC
852hfvcRubLV92eRdAmNGFqTOqaUcS00i3QZyp4MRGqxtOV/88y/nEOtP1RHkZJw
HXTjHq4JsDvwhnQR8JbCX6z1zkLQdS01u3jrwJTaPpooxdATfPlfO6CYjqM+SapB
o7dS1ZAZb4U8vPx+MWoDEVNxvO7/xyqho1Oc4H9MwqQUiyG2WfIoqxLSrBYcambv
RmySwTIpgQZTr61EeWf/0eWpV0iEYbSnkB/VaKW+5tg4gCjPgy5v6/LQ0u/pzlYz
ayCL3xN2rp0tigXsiiWz3cM5gDsnatK4nVNRs9y3JSZpWpI236ZfZjs8Lts+WBUw
hAEoE9kCgYEAyEIGD1A7R/t5EYk5HhHDH5tGdyxejAcQL5AIz0YnTZU8Iixyc7FR
uDmAMiuKIcJY/nUlxZjSxNc3MkOfZNggQvf9ONrt+ftQ1yyTjv+019NfU4w4d0Ep
LNaiAHgaPKimBUZjYXbLgiMXj/1pBaQmgUYTK/VlO3PVdowxxzxMYlMCgYEAvEOG
GrhVaQV1nAYx86BgZ3wn90hBFXZWGaN+eXUmyrast93Ih3TCSgQDKPuN3pdv/TIe
cpQv/BxEMpW+6d5Z1NP3GbrLpaZUiUNk8fqw1S3pmD5aWZrYIUaNukAyOxnZVgjv
EWD9QTpI663gODaeZZTkDYiRNzTzGOg5HtzporUCgYBBOphEtqqImNXnq13qeHip
O+eo+8/UJpzUEUN9WGmG8NxEeVvSaWin7DrgnKQCuQ5J3Biwk0XcDgoRmks6Ctf/
WE2oDk/DxGOhowhxZMMgJd6AFUVzOstRqpvcMULCjWB+iV3nqk1Bl3KeWTmzN7O/
Gfc2s1kFE4btdV7lebObtwKBgE3rkLS8eLVYCh6Cvef9CAms7Im/wRhV+zrvXWh9
4YljZEdRpy7RV5z03i33N/faLALa3JlF1jp9pIhfTD5Vxk59ULe4hZNRLYoGd+Bj
hw8kyps1q4WMvkm/fueIrIGjqD2gwvopb4iwy/+n3rbFfHfE0UL8tEXqR3eWnhW1
D4pFAoGAccR4eMJD43hJWaUQLtsj0RoW9lFKVXj7aqkIIeupXwt7Ic2z/FhCAJi+
V0MWpd3K6+kPl+ifdt8U4kcYfubPMfJhd7IkMcgQS+yZK1+5xWdRISvI8GpNwIHE
LUkVkCCadXNNZ7b1nmUKjse95u4IaE6hwAqjSTNb05gPmCfoEjg=
-----END RSA PRIVATE KEY-----`

const jwtESKeyP256 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgevZzL1gdAFr88hb2
OF/2NxApJCzGCEDdfSp6VQO30hyhRANCAAQRWz+jn65BtOMvdyHKcvjBeBSDZH2r
1RTwjmYSi9R/zpBnuQ4EiMnCqfMPWiZqB4QdbAd0E7oH50VpuZ1P087G
-----END PRIVATE KEY-----`

const jwtESKeyP384 = `-----BEGIN PRIVATE KEY-----
MIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDDpgCvB2frnLKd7TuWe
JM1ejXXmr9y/5gskxKuuylLvpQTiDdtLtuhJnvw1/zWKWO6hZANiAAQ5Crhsi5FD
t55i53dCtdzG9OzCnbDFf/6136ZfEiakDTDeWCdUvNnB3WQEcVBr97BfSWLI9mO+
T5yzm0RfhgvWIq/tBou+sIDeGp6NQfJwhDhf+JsdeF174gtfNMZGj/s=
-----END PRIVATE KEY-----`

const jwtESKeyP521 = `-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIA0dBErrZ5ovKq4Xf/
iTlRkYxuOfgBZ6+tWIfG13YwthB1XrH06YmteZGNjHHLZEeycwUt0jM4kUb+tOsJ
3ckhj1ihgYkDgYYABACYgsa8JWKH46CQagwNw14v/L+DIs1WAjJdMXZySjKlRkD9
LtLMxkbX2H4H4Zl2KzCMJkwTSETzSKNlXvAUJqKbRwHezCp4y5XZN9MOBYdmyylZ
NOVxwwTouimNkJ0K6A8+/Im5S3PWB8Ra1D6t+bT1WHHhEePZcltSLLFlbIIyot5m
2w==
-----END PRIVATE KEY-----`

const jwtRSPub = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDdlatRjRjogo3WojgGHFHYLugd
UWAY9iR3fy4arWNA1KoS8kVw33cJibXr8bvwUAUparCwlvdbH6dvEOfou0/gCFQs
HUfQrSDv+MuSUMAe8jzKE4qW+jK+xQU9a03GUnKHkkle+Q0pX/g6jXZ7r1/xAK5D
o2kQ+X5xK9cipRgEKwIDAQAB
-----END PUBLIC KEY-----`

const jwtESPub = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEEVs/o5+uQbTjL3chynL4wXgUg2R9
q9UU8I5mEovUf86QZ7kOBIjJwqnzD1omageEHWwHdBO6B+dFabmdT9POxg==
-----END PUBLIC KEY-----`

func TestJWTDecodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"JWT Decode: HS",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.0ha6-j4FwvEIKPVZ-hf3S_R9Hy_UtXzq4dnedXcUrXk",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Decode", Args: []any{}}},
		},
		{
			"JWT Decode: RS",
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.MjEJhtZk2nXzigi24piMzANmrj3mILHJcDl0xOjl5a8EgdKVL1oaMEjTkMQp5RA8YrqeRBFaX-BGGCKOXn5zPY1DJwWsBUyN9C-wGR2Qye0eogH_3b4M9EW00TPCUPXm2rx8URFj7Wg9VlsmrGzLV2oKkPgkVxuFSxnpO3yjn1Y",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Decode", Args: []any{}}},
		},
		{
			"JWT Decode: ES",
			"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.WkECT51jSfpRkcpQ4x0h5Dwe7CFBI6u6Et2gWp91HC7mpN_qCFadRpsvJLtKubm6cJTLa68xtei0YrDD8fxIUA",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Decode", Args: []any{}}},
		},
	})
}

func TestJWTSignFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"JWT Sign: HS256", jwtPayloadObject,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.0ha6-j4FwvEIKPVZ-hf3S_R9Hy_UtXzq4dnedXcUrXk",
			core.Recipe{{Op: "JWT Sign", Args: []any{jwtHSKey, "HS256", "{}"}}},
		},
		{
			"JWT Sign: HS256 with custom header", jwtPayloadObject,
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImN1c3RvbS5rZXkifQ.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.kXln8btJburfRlND8IDZAQ8NZGFFZhvHyooHa6N9za8",
			core.Recipe{{Op: "JWT Sign", Args: []any{jwtHSKey, "HS256", `{"kid":"custom.key"}`}}},
		},
		{
			"JWT Sign: HS384", jwtPayloadObject,
			"eyJhbGciOiJIUzM4NCIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ._bPK-Y3mIACConbJqkGFMQ_L3vbxgKXy9gSxtL9hA5XTganozTSXxD0vX0N1yT5s",
			core.Recipe{{Op: "JWT Sign", Args: []any{jwtHSKey, "HS384", "{}"}}},
		},
		{
			"JWT Sign: HS512", jwtPayloadObject,
			"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.vZIJU4XYMFt3FLE1V_RZOxEetmV4RvxtPZQGzJthK_d47pjwlEb6pQE23YxHFmOj8H5RLEdqqLPw4jNsOyHRzA",
			core.Recipe{{Op: "JWT Sign", Args: []any{jwtHSKey, "HS512", "{}"}}},
		},
		// RS/ES signatures are round-tripped through Decode (ES is non-deterministic;
		// this also matches the upstream fixtures' approach).
		{
			"JWT Sign: ES256", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtESKeyP256, "ES256", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
		{
			"JWT Sign: ES384", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtESKeyP384, "ES384", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
		{
			"JWT Sign: ES512", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtESKeyP521, "ES512", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
		{
			"JWT Sign: RS256", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtRSKey2048, "RS256", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
		{
			"JWT Sign: RS384", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtRSKey2048, "RS384", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
		{
			"JWT Sign: RS512", jwtPayloadObject, jwtPayloadObject,
			core.Recipe{
				{Op: "JWT Sign", Args: []any{jwtRSKey2048, "RS512", "{}"}},
				{Op: "JWT Decode", Args: []any{}},
			},
		},
	})
}

func TestJWTVerifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"JWT Verify: HS",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.0ha6-j4FwvEIKPVZ-hf3S_R9Hy_UtXzq4dnedXcUrXk",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Verify", Args: []any{jwtHSKey}}},
		},
		{
			"JWT Verify: RS",
			"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.MjEJhtZk2nXzigi24piMzANmrj3mILHJcDl0xOjl5a8EgdKVL1oaMEjTkMQp5RA8YrqeRBFaX-BGGCKOXn5zPY1DJwWsBUyN9C-wGR2Qye0eogH_3b4M9EW00TPCUPXm2rx8URFj7Wg9VlsmrGzLV2oKkPgkVxuFSxnpO3yjn1Y",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Verify", Args: []any{jwtRSPub}}},
		},
		{
			"JWT Verify: ES",
			"eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.WkECT51jSfpRkcpQ4x0h5Dwe7CFBI6u6Et2gWp91HC7mpN_qCFadRpsvJLtKubm6cJTLa68xtei0YrDD8fxIUA",
			jwtPayloadObject,
			core.Recipe{{Op: "JWT Verify", Args: []any{jwtESPub}}},
		},
	})
}

// TestJWTSignKeyErrors covers the two upstream error fixtures: a weak RSA key and
// an ECDSA curve mismatch, both of which jsonwebtoken rejects before signing.
func TestJWTSignKeyErrors(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		alg     string
		wantSub string
	}{
		{
			"RS256 weak key", jwtRSKey1024, "RS256",
			"secretOrPrivateKey has a minimum key size of 2048 bits for RS256",
		},
		{
			"ES384 with P256 key", jwtESKeyP256, "ES384",
			`"alg" parameter "ES384" requires curve "secp384r1".`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWT Sign", jwtPayloadObject, c.key, c.alg, "{}")
			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if !strings.Contains(err.Error(), c.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), c.wantSub)
			}
		})
	}
}

// jwtToken builds a raw token from three already-encoded-or-literal parts,
// base64url-encoding the header/payload JSON and using sig verbatim.
func jwtToken(headerJSON, payloadJSON, sig string) string {
	return jwtSegEnc.EncodeToString([]byte(headerJSON)) + "." +
		jwtSegEnc.EncodeToString([]byte(payloadJSON)) + "." + sig
}

// TestJWTSignIatInjection verifies iat is appended with the current timestamp
// when the payload omits it (matching jsonwebtoken).
func TestJWTSignIatInjection(t *testing.T) {
	before := jwtNow()
	tok, err := runOp(t, "JWT Sign", `{"a":1}`, "secret", "HS256", "{}")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	after := jwtNow()
	out, err := runOp(t, "JWT Decode", tok)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	v, err := jsonval.ParseOrdered([]byte(out))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	obj := v.(jsonval.Object)
	i := jsonval.Index(obj, "iat")
	if i < 0 {
		t.Fatalf("iat not injected: %s", out)
	}
	iat := int64(obj[i].V.(float64))
	if iat < before || iat > after {
		t.Fatalf("iat %d not in [%d, %d]", iat, before, after)
	}
}

// TestJWTNoneRoundTrip covers signing and verifying an unsecured (alg=none) token.
func TestJWTNoneRoundTrip(t *testing.T) {
	tok, err := runOp(t, "JWT Sign", `{"a":1,"iat":1}`, "ignored", "None", "{}")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !strings.HasSuffix(tok, ".") {
		t.Fatalf("none token should have empty signature: %q", tok)
	}
	out, err := runOp(t, "JWT Verify", tok, "ignored")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	want := "{\n    \"a\": 1,\n    \"iat\": 1\n}"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

// TestJWTSignErrors covers the input/header validation error paths of JWT Sign.
func TestJWTSignErrors(t *testing.T) {
	cases := []struct {
		name, input, key, alg, header, sub string
	}{
		{"bad payload JSON", "{", "s", "HS256", "{}", jwtSignKeyHint},
		{"non-object payload", "[1,2]", "s", "HS256", "{}", "payload must be a JSON object"},
		{"bad header JSON", `{"a":1}`, "s", "HS256", "{", jwtSignKeyHint},
		{"non-object header", `{"a":1}`, "s", "HS256", "[1]", "header must be a JSON object"},
		{"bad RSA PEM", `{"a":1}`, "not a pem", "RS256", "{}", jwtSignKeyHint},
		{"bad ES PEM", `{"a":1}`, "not a pem", "ES256", "{}", jwtSignKeyHint},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWT Sign", c.input, c.key, c.alg, c.header)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("error %q missing %q", err.Error(), c.sub)
			}
		})
	}
}

// TestJWTSignSegmentUnsupportedAlg directly exercises the defensive guard for an
// algorithm that passed no option check (unreachable via Run, which restricts to
// the registered algorithms).
func TestJWTSignSegmentUnsupportedAlg(t *testing.T) {
	if _, err := jwtSignSegment("a.b", "k", "ZZ999"); err == nil {
		t.Fatal("expected error for unsupported algorithm")
	}
}

// TestJWTDecodeErrors covers malformed tokens.
func TestJWTDecodeErrors(t *testing.T) {
	cases := []struct{ name, token, sub string }{
		{"two segments", "abc.def", "expected 3 segments"},
		{"bad base64 payload", "eyJhIjoxfQ.!!!.sig", "JWT Decode"},
		{"non-JSON payload", jwtToken(`{"alg":"none"}`, "not json", ""), "JWT Decode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWT Decode", c.token)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("error %q missing %q", err.Error(), c.sub)
			}
		})
	}
}

// TestJWTVerifyErrors covers the signature, structure, and claim failure paths.
func TestJWTVerifyErrors(t *testing.T) {
	valid, err := runOp(t, "JWT Sign", `{"a":1,"iat":1}`, "right", "HS256", "{}")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	expired, err := runOp(t, "JWT Sign", `{"a":1,"exp":1}`, "k", "HS256", "{}")
	if err != nil {
		t.Fatalf("sign expired: %v", err)
	}
	notYet, err := runOp(t, "JWT Sign", `{"a":1,"nbf":9999999999}`, "k", "HS256", "{}")
	if err != nil {
		t.Fatalf("sign nbf: %v", err)
	}
	cases := []struct{ name, token, key, sub string }{
		{"wrong key", valid, "wrong", "JWT Verify"},
		{"tampered", valid[:len(valid)-1] + "X", "right", "JWT Verify"},
		{"two segments", "a.b", "k", "expected 3 segments"},
		{"bad base64 header", "!!!.eyJhIjoxfQ.sig", "k", "JWT Verify"},
		{"header not object", jwtToken(`[1]`, `{"a":1}`, "sig"), "k", "invalid header"},
		{"missing alg", jwtToken(`{"typ":"JWT"}`, `{"a":1}`, "sig"), "k", "invalid algorithm"},
		{"alg not string", jwtToken(`{"alg":1}`, `{"a":1}`, "sig"), "k", "invalid algorithm"},
		{"unknown alg", jwtToken(`{"alg":"FOO"}`, `{"a":1}`, "c2ln"), "k", "invalid algorithm"},
		{"none with signature", jwtToken(`{"alg":"none"}`, `{"a":1}`, "c2ln"), "k", "invalid signature"},
		{"bad sig base64", jwtToken(`{"alg":"HS256"}`, `{"a":1}`, "!!!"), "k", "JWT Verify"},
		{"expired", expired, "k", "jwt expired"},
		{"not active", notYet, "k", "jwt not active"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "JWT Verify", c.token, c.key)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("error %q missing %q", err.Error(), c.sub)
			}
		})
	}
}

// TestJWTVerifyKeyParseErrors covers RSA/ECDSA public-key parse failures during
// verification (bad PEM supplied for an RS/ES token).
func TestJWTVerifyKeyParseErrors(t *testing.T) {
	rsTok := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.MjEJhtZk2nXzigi24piMzANmrj3mILHJcDl0xOjl5a8EgdKVL1oaMEjTkMQp5RA8YrqeRBFaX-BGGCKOXn5zPY1DJwWsBUyN9C-wGR2Qye0eogH_3b4M9EW00TPCUPXm2rx8URFj7Wg9VlsmrGzLV2oKkPgkVxuFSxnpO3yjn1Y"
	esTok := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJTdHJpbmciOiJTb21lU3RyaW5nIiwiTnVtYmVyIjo0MiwiaWF0IjoxfQ.WkECT51jSfpRkcpQ4x0h5Dwe7CFBI6u6Et2gWp91HC7mpN_qCFadRpsvJLtKubm6cJTLa68xtei0YrDD8fxIUA"
	for _, tok := range []string{rsTok, esTok} {
		if _, err := runOp(t, "JWT Verify", tok, "not a pem"); err == nil {
			t.Fatalf("expected key parse error for %.20s...", tok)
		}
	}
}

// TestJWTSignHeaderBranches covers the empty-header default and appending a novel
// custom header key (not one of alg/typ/kid).
func TestJWTSignHeaderBranches(t *testing.T) {
	// Empty header arg defaults to "{}".
	tok, err := runOp(t, "JWT Sign", `{"a":1,"iat":1}`, "k", "HS256", "")
	if err != nil {
		t.Fatalf("empty header: %v", err)
	}
	if hdr, _ := jwtSegment(tok, 0, "x"); jsonval.Stringify(hdr, 0) != `{"alg":"HS256","typ":"JWT"}` {
		t.Fatalf("unexpected header: %s", jsonval.Stringify(hdr, 0))
	}
	// A novel custom key is appended after alg/typ.
	tok, err = runOp(t, "JWT Sign", `{"a":1,"iat":1}`, "k", "HS256", `{"foo":"bar"}`)
	if err != nil {
		t.Fatalf("custom header: %v", err)
	}
	hdr, _ := jwtSegment(tok, 0, "x")
	if got := jsonval.Stringify(hdr, 0); got != `{"alg":"HS256","typ":"JWT","foo":"bar"}` {
		t.Fatalf("unexpected header: %s", got)
	}
}

// TestJWTSignWithError directly exercises the signing-error path with a key of the
// wrong type for the method (unreachable via Run, which resolves the right key).
func TestJWTSignWithError(t *testing.T) {
	if _, err := jwtSignWith(jwt.SigningMethodES256, "a.b", []byte("not an ec key")); err == nil {
		t.Fatal("expected signing error for wrong key type")
	}
}

// TestJWTVerifyNonObjectAndBadPayload covers verifying unsecured tokens whose
// payload is a non-object (skips claim checks) or invalid JSON (decode error
// after a successful signature check).
func TestJWTVerifyNonObjectAndBadPayload(t *testing.T) {
	arrTok := jwtToken(`{"alg":"none"}`, `[1,2]`, "")
	out, err := runOp(t, "JWT Verify", arrTok, "k")
	if err != nil {
		t.Fatalf("array payload: %v", err)
	}
	if out != "[\n    1,\n    2\n]" {
		t.Fatalf("unexpected array output: %q", out)
	}
	badTok := jwtToken(`{"alg":"none"}`, `not json`, "")
	if _, err := runOp(t, "JWT Verify", badTok, "k"); err == nil {
		t.Fatal("expected payload decode error")
	}
}
