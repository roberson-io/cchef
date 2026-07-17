package ops

// Fernet Encrypt/Decrypt tests. CyberChef wraps the `fernet` npm library; these
// vectors were produced by that exact library (in ../CyberChef/node_modules).
// Encryption is normally non-deterministic (random IV + current time), so the
// byte-exact encrypt vectors were generated with a fixed IV (00..0f) and time
// (1622505600) via the internal fernetEncrypt seam; the public op is checked by
// round-trip. The decrypt fixture is from ../CyberChef/tests/operations/tests/Fernet.mjs.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const (
	// fernetKey1 is url-safe base64 (contains - and _).
	fernetKey1 = "cw_0x689RpI-jtRR7oE8h_eQsKImvJapLeSbXpwF4e4="
	// fernetKey2 is standard base64 ("ThisIsThirtyTwoCharactersLongKey").
	fernetKey2 = "VGhpc0lzVGhpcnR5VHdvQ2hhcmFjdGVyc0xvbmdLZXk="
	// fernetFixedTime / fernetFixedIVHex are the injected encrypt parameters.
	fernetFixedTime  = uint64(1622505600)
	fernetFixedIVHex = "000102030405060708090a0b0c0d0e0f"
)

// fernetVectors pairs a plaintext with its byte-exact token under the fixed
// IV/time above.
var fernetVectors = []struct {
	name, key, plain, token string
}{
	{"K1 hello world", fernetKey1, "hello world", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD-jYASuvlbTXbiWa16JfsvJuCE8tafNo-BC-ggSZs3OV8eWfNJRd7iMjML1Hq4ju6A=="},
	{"K1 empty", fernetKey1, "", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD3HkMATM5lFqGaerZ-fWPAmhVRwqDl1Mbk0CSYmgg7iguYLUJRd61WatJDCwntbFLw=="},
	{"K1 hi", fernetKey1, "hi", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD5iuEcXlBpYU0nH0AhAhIn-jfN-Wg3yL4xlvn-hJy6r6MENcSPtpNcAvsswcO5LNzg=="},
	{"K1 block16", fernetKey1, "0123456789abcdef", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD1xYxWk-FnTVkOLqDQyotg1f_5l_C34LZuLmvvtMaTZBl8GqAzZVyo4-j17YokUQDoQLbxB66vHNIyWE_a7-Tn8="},
	{"K1 message\\n", fernetKey1, "This is a secret message.\n", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0ODycWl1U0nmC5iXwYeYLbkrS0APf714gvVSxPOPVcYW5PEXGwmfbrhEjkKbbJAJaHn9zj-YJkJUsmWZZIn_1NKLg="},
	{"K2 hello world", fernetKey2, "hello world", "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD8eg3qJt3yUfZxWaNuGzCYU6JUzpkDTQDMmM92o_Iuze9Yao0Q91ctgnqOIEiWBIuA=="},
}

// TestFernetEncryptExact checks the internal encrypt seam against byte-exact
// npm-library tokens (fixed IV + time).
func TestFernetEncryptExact(t *testing.T) {
	iv, _ := hex.DecodeString(fernetFixedIVHex)
	for _, v := range fernetVectors {
		t.Run(v.name, func(t *testing.T) {
			signKey, block, err := fernetSecret(v.key)
			if err != nil {
				t.Fatalf("fernetSecret: %v", err)
			}
			got := fernetEncrypt(signKey, block, iv, fernetFixedTime, []byte(v.plain))
			if got != v.token {
				t.Fatalf("token mismatch\n got %q\nwant %q", got, v.token)
			}
		})
	}
}

// TestFernetDecrypt covers the CyberChef fixture and decrypting every encrypt
// vector back to its plaintext (deterministic: the op uses ttl=0).
func TestFernetDecrypt(t *testing.T) {
	cases := []opCase{
		{
			"Fernet fixture", "gAAAAABce-Tycae8klRxhDX2uenJ-uwV8-A1XZ2HRnfOXlNzkKKfRxviNLlgtemhT_fd1Fw5P_zFUAjd69zaJBQyWppAxVV00SExe77ql8c5n62HYJOnoIU=",
			"This is a secret message.\n",
			core.Recipe{{Op: "Fernet Decrypt", Args: []any{fernetKey2}}},
		},
	}
	for _, v := range fernetVectors {
		cases = append(cases, opCase{
			"decrypt " + v.name, v.token, v.plain,
			core.Recipe{{Op: "Fernet Decrypt", Args: []any{v.key}}},
		})
	}
	runCases(t, cases)
}

// TestFernetDecryptLenientPadding replicates crypto-js's non-validating PKCS7
// unpad: a crafted token (valid HMAC) whose final plaintext byte is 0x00 strips
// zero bytes, yielding the raw 16-byte block — matching the npm library.
func TestFernetDecryptLenientPadding(t *testing.T) {
	const token = "gAAAAABgtXiAAAECAwQFBgcICQoLDA0OD_Q50EsFpVbS9Zxdx2xWxW4bEQWisfQ5EjypGBs0D9xxEo-3NnkGXQjy70v0E91pIA=="
	out, err := runOp(t, "Fernet Decrypt", token, fernetKey1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := string(make([]byte, 16)); out != want {
		t.Fatalf("got %q, want 16 NUL bytes", out)
	}
}

// TestFernetRoundTrip exercises the public (random-IV) encrypt op: its output
// matches the Fernet token shape and decrypts back to the input.
func TestFernetRoundTrip(t *testing.T) {
	shape := regexp.MustCompile(`^gAAA[\w-]+={0,2}$`)
	for _, plain := range []string{"", "hello", "This is a secret message.\n", "0123456789abcdef"} {
		tok, err := runOp(t, "Fernet Encrypt", plain, fernetKey1)
		if err != nil {
			t.Fatalf("encrypt %q: %v", plain, err)
		}
		if !shape.MatchString(tok) {
			t.Fatalf("token %q does not match Fernet shape", tok)
		}
		back, err := runOp(t, "Fernet Decrypt", tok, fernetKey1)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if back != plain {
			t.Fatalf("round trip: got %q, want %q", back, plain)
		}
	}
	// Two encryptions differ (fresh random IV each time).
	a, _ := runOp(t, "Fernet Encrypt", "same", fernetKey1)
	b, _ := runOp(t, "Fernet Encrypt", "same", fernetKey1)
	if a == b {
		t.Fatal("expected differing tokens from fresh random IVs")
	}
}

// TestFernetUnpad directly exercises the lenient-unpad edge cases (empty input
// and an over-large final byte, which is clamped).
func TestFernetUnpad(t *testing.T) {
	if got := fernetUnpad(nil); len(got) != 0 {
		t.Fatalf("empty: got %v", got)
	}
	// Final byte 5 exceeds the 4-byte length → clamped, everything removed.
	if got := fernetUnpad([]byte{1, 2, 3, 5}); len(got) != 0 {
		t.Fatalf("clamp: got %v", got)
	}
}

// TestFernetDecryptEmptyCiphertext crafts a structurally-valid 57-byte token
// (header + correct HMAC, no ciphertext) to reach the empty-ciphertext guard.
func TestFernetDecryptEmptyCiphertext(t *testing.T) {
	signKey, _, err := fernetSecret(fernetKey1)
	if err != nil {
		t.Fatal(err)
	}
	header := make([]byte, fernetHeaderLen) // version + time + IV, no ciphertext
	header[0] = fernetVersion
	mac := hmac.New(sha256.New, signKey)
	mac.Write(header)
	token := fernetEncodeBase64(mac.Sum(header)) // header || HMAC(header), 57 bytes
	if _, err := runOp(t, "Fernet Decrypt", token, fernetKey1); err == nil {
		t.Fatal("expected error for empty ciphertext")
	}
}

// TestFernetErrors covers key and token validation on both operations.
func TestFernetErrors(t *testing.T) {
	validToken := fernetVectors[0].token
	// Malformed tokens with a valid version byte but bad length.
	shortTok := fernetEncodeBase64(append([]byte{fernetVersion}, make([]byte, 39)...))      // len 40 < 57
	misalignedTok := fernetEncodeBase64(append([]byte{fernetVersion}, make([]byte, 63)...)) // len 64 -> (64-57)%16 != 0

	cases := []struct {
		name, op, input, key string
	}{
		{"decrypt empty (invalid version)", "Fernet Decrypt", "", fernetKey1},
		{"decrypt invalid base64", "Fernet Decrypt", "!!!!", fernetKey1},
		{"decrypt bad version byte", "Fernet Decrypt", "AAAA", fernetKey1},
		{"decrypt short token", "Fernet Decrypt", shortTok, fernetKey1},
		{"decrypt misaligned token", "Fernet Decrypt", misalignedTok, fernetKey1},
		{"decrypt empty key", "Fernet Decrypt", validToken, ""},
		{"decrypt wrong key (HMAC)", "Fernet Decrypt", validToken, fernetKey2},
		{"encrypt empty key", "Fernet Encrypt", "hello", ""},
		{"encrypt short key", "Fernet Encrypt", "hello", "YWJj"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, c.op, c.input, c.key); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}
