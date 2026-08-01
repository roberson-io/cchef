package ops

// Flask Session Decode/Sign/Verify tests. Fixtures transcribed from
// ../CyberChef/tests/operations/tests/FlaskSession.mjs. Signing is normally
// non-deterministic (current timestamp); the byte-exact vectors reproduce the
// fixture tokens by feeding their embedded timestamp to the internal seam, and
// the public op is checked by Sign -> Verify round trips.

import (
	"crypto/sha1"
	"crypto/sha256"
	"testing"

	"github.com/roberson-io/cchef/core"
)

const (
	flaskTokenSha1   = "eyJyb2xlIjoic3VwZXJ1c2VyIiwidXNlciI6ImFkbWluIn0.aZ-KEw.E_x6bOhA4GU9t72pMinJUjN-O3I"
	flaskTokenSha256 = "eyJyb2xlIjoic3VwZXJ1c2VyIiwidXNlciI6ImFkbWluIn0.aab3Ew.Jsx2DOx_H9anZg0YcvhsASxQ11897EFHeQfS2oja4y8"
	flaskKey         = "mysecretkey"
	flaskSalt        = "cookie-session"
	flaskPayloadJSON = "{\n    \"role\": \"superuser\",\n    \"user\": \"admin\"\n}"
)

// flaskOutputObject / flaskOutputVerify mirror the fixture expected outputs.
const (
	flaskOutputObject = "{\n    \"role\": \"superuser\",\n    \"user\": \"admin\"\n}"
	flaskOutputVerify = "{\n    \"valid\": true,\n    \"payload\": {\n        \"role\": \"superuser\",\n        \"user\": \"admin\"\n    }\n}"
)

// flaskKeyArg / flaskSaltArg build the toggle-string arguments.
func flaskKeyArg(s string) core.ToggleString  { return core.ToggleString{Value: s, Option: "UTF8"} }
func flaskSaltArg(s string) core.ToggleString { return core.ToggleString{Value: s, Option: "UTF8"} }

func TestFlaskSessionDecode(t *testing.T) {
	runCases(t, []opCase{
		{
			"decode (no timestamp)", flaskTokenSha1, flaskOutputObject,
			core.Recipe{{Op: "Flask Session Decode", Args: []any{false}}},
		},
		{
			"decode (with timestamp)", flaskTokenSha1,
			"{\n    \"payload\": {\n        \"role\": \"superuser\",\n        \"user\": \"admin\"\n    },\n    \"timestamp\": 1772063251\n}",
			core.Recipe{{Op: "Flask Session Decode", Args: []any{true}}},
		},
	})
}

func TestFlaskSessionVerify(t *testing.T) {
	verify := func(key, salt, algo string, viewTS bool) core.Recipe {
		return core.Recipe{{Op: "Flask Session Verify", Args: []any{
			flaskKeyArg(key), flaskSaltArg(salt), algo, viewTS,
		}}}
	}
	runCases(t, []opCase{
		{"verify sha1", flaskTokenSha1, flaskOutputVerify, verify(flaskKey, flaskSalt, "sha1", false)},
		{"verify sha256", flaskTokenSha256, flaskOutputVerify, verify(flaskKey, flaskSalt, "sha256", false)},
		// Empty salt falls back to the "cookie-session" default, so this verifies.
		{"verify empty salt (default)", flaskTokenSha1, flaskOutputVerify, verify(flaskKey, "", "sha1", false)},
		{
			"verify sha1 with timestamp", flaskTokenSha1,
			"{\n    \"valid\": true,\n    \"payload\": {\n        \"role\": \"superuser\",\n        \"user\": \"admin\"\n    },\n    \"timestamp\": 1772063251\n}",
			verify(flaskKey, flaskSalt, "sha1", true),
		},
	})
}

// TestFlaskSessionSignExact reproduces the fixture tokens byte-for-byte by
// injecting their embedded timestamp into the internal signing seam.
func TestFlaskSessionSignExact(t *testing.T) {
	payload, err := jsonParseOrdered([]byte(flaskPayloadJSON))
	if err != nil {
		t.Fatal(err)
	}
	got1 := flaskSignToken(sha1.New, []byte(flaskKey), []byte(flaskSalt), payload, 1772063251)
	if got1 != flaskTokenSha1 {
		t.Fatalf("sha1 token mismatch\n got %q\nwant %q", got1, flaskTokenSha1)
	}
	got256 := flaskSignToken(sha256.New, []byte(flaskKey), []byte(flaskSalt), payload, 1772549907)
	if got256 != flaskTokenSha256 {
		t.Fatalf("sha256 token mismatch\n got %q\nwant %q", got256, flaskTokenSha256)
	}
}

// TestFlaskSessionRoundTrip signs a payload (random timestamp) then verifies it.
func TestFlaskSessionSignRoundTrip(t *testing.T) {
	for _, algo := range []string{"sha1", "sha256"} {
		token, err := runOp(t, "Flask Session Sign", flaskPayloadJSON,
			flaskKeyArg(flaskKey), flaskSaltArg(flaskSalt), algo)
		if err != nil {
			t.Fatalf("sign %s: %v", algo, err)
		}
		out, err := runOp(t, "Flask Session Verify", token,
			flaskKeyArg(flaskKey), flaskSaltArg(flaskSalt), algo, false)
		if err != nil {
			t.Fatalf("verify %s: %v", algo, err)
		}
		if out != flaskOutputVerify {
			t.Fatalf("%s round trip: got %q", algo, out)
		}
	}
}

// TestFlaskSessionErrors covers signature, key, and format validation.
func TestFlaskSessionErrors(t *testing.T) {
	cases := []struct {
		name    string
		op      string
		input   string
		args    []any
		wantErr string
	}{
		{
			"verify wrong key", "Flask Session Verify", flaskTokenSha1,
			[]any{flaskKeyArg("notTheKey"), flaskSaltArg(flaskSalt), "sha1", false},
			"Invalid signature!",
		},
		{
			"verify wrong salt", "Flask Session Verify", flaskTokenSha1,
			[]any{flaskKeyArg(flaskKey), flaskSaltArg("notTheSalt"), "sha1", false},
			"Invalid signature!",
		},
		{
			"verify no key", "Flask Session Verify", flaskTokenSha1,
			[]any{flaskKeyArg(""), flaskSaltArg(flaskSalt), "sha1", false},
			"Secret key required",
		},
		{
			"verify bad format", "Flask Session Verify", "onlyonepart",
			[]any{flaskKeyArg(flaskKey), flaskSaltArg(flaskSalt), "sha1", false},
			"Invalid Flask token format",
		},
		{
			"sign no key", "Flask Session Sign", flaskPayloadJSON,
			[]any{flaskKeyArg(""), flaskSaltArg(flaskSalt), "sha1"},
			"Secret key required",
		},
		{"decode bad format", "Flask Session Decode", "a.b", []any{false}, "Invalid Flask token format"},
		{
			"decode invalid base64 payload", "Flask Session Decode", "!!!!.aZ-KEw.x",
			[]any{false},
			"Invalid Base64 payload",
		},
		{
			"decode non-JSON payload", "Flask Session Decode", "bm90IGpzb24.aZ-KEw.x",
			[]any{false},
			"Unable to decode JSON payload",
		},
		{
			"verify invalid base64 payload", "Flask Session Verify", "!!!!.aZ-KEw.x",
			[]any{flaskKeyArg(flaskKey), flaskSaltArg(flaskSalt), "sha1", false},
			"Invalid Base64 payload",
		},
		{
			"sign bad base64 key", "Flask Session Sign", flaskPayloadJSON,
			[]any{core.ToggleString{Value: "!!!", Option: "Base64"}, flaskSaltArg(flaskSalt), "sha1"},
			"base64",
		},
		{
			"verify bad base64 salt", "Flask Session Verify", flaskTokenSha1,
			[]any{flaskKeyArg(flaskKey), core.ToggleString{Value: "!!!", Option: "Base64"}, "sha1", false},
			"base64",
		},
		{
			"sign non-JSON input", "Flask Session Sign", "not json",
			[]any{flaskKeyArg(flaskKey), flaskSaltArg(flaskSalt), "sha1"},
			"JSON",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.args...)
			if err == nil {
				t.Fatalf("expected error %q", c.wantErr)
			}
		})
	}
}
