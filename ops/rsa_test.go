package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// signVec builds a "RSA Sign then To Hex" case for a digest over "test message".
func rsaSignCase(name, key, pass, md, want string) opCase {
	return opCase{
		name, "test message", want,
		core.Recipe{
			{Op: "RSA Sign", Args: []any{key, pass, md}},
			{Op: "To Hex", Args: []any{"None"}},
		},
	}
}

// TestRSAFixtures round-trips RSA-OAEP encrypt/decrypt for every digest, mirroring
// CyberChef's tests/operations/tests/RSA.mjs (OAEP ciphertext is randomised, so
// only the round-trip is checked).
func TestRSAFixtures(t *testing.T) {
	rt := func(md, input string) opCase {
		return opCase{
			"RSA-OAEP/" + md, input, input,
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048, "RSA-OAEP", md}},
				{Op: "RSA Decrypt", Args: []any{rsaOpPriv2048, "", "RSA-OAEP", md}},
			},
		}
	}
	runCases(t, []opCase{
		rt("SHA-1", ""), rt("SHA-1", asciiText),
		rt("MD5", ""), rt("MD5", asciiText),
		rt("SHA-256", ""), rt("SHA-256", asciiText),
		rt("SHA-384", ""), rt("SHA-384", asciiText),
		rt("SHA-512", ""), rt("SHA-512", asciiText[:100]),
	})
}

// TestRSASchemes covers the PKCS#1 v1.5 and RAW encryption schemes. RAW is
// deterministic (textbook RSA), so its ciphertext is compared byte-for-byte.
func TestRSASchemes(t *testing.T) {
	runCases(t, []opCase{
		{
			"PKCS1v15 round-trip", asciiText[:100], asciiText[:100],
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048, "RSAES-PKCS1-V1_5", "SHA-1"}},
				{Op: "RSA Decrypt", Args: []any{rsaOpPriv2048, "", "RSAES-PKCS1-V1_5", "SHA-1"}},
			},
		},
		{
			"RAW encrypt (deterministic)", "test message", rsaRawEncTestMsg,
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048, "RAW", "SHA-1"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// RAW decrypt returns the full modulus-width block (zero-padded); the
		// message trails at the end.
		{
			"RAW round-trip tail", "test message",
			strings.Repeat("00", 244) + "74657374206d657373616765",
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048, "RAW", "SHA-1"}},
				{Op: "RSA Decrypt", Args: []any{rsaOpPriv2048, "", "RAW", "SHA-1"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
	})
}

// TestRSASignVectors checks the deterministic PKCS#1 v1.5 signatures for every
// digest against oracle-captured vectors.
func TestRSASignVectors(t *testing.T) {
	runCases(t, []opCase{
		rsaSignCase("Sign SHA-1", rsaOpPriv2048, "", "SHA-1", rsaSigSHA1),
		rsaSignCase("Sign MD5", rsaOpPriv2048, "", "MD5", rsaSigMD5),
		rsaSignCase("Sign SHA-256", rsaOpPriv2048, "", "SHA-256", rsaSigSHA256),
		rsaSignCase("Sign SHA-384", rsaOpPriv2048, "", "SHA-384", rsaSigSHA384),
		rsaSignCase("Sign SHA-512", rsaOpPriv2048, "", "SHA-512", rsaSigSHA512),
	})
}

// TestRSAVerify checks signature verification, message formats, and failure.
func TestRSAVerify(t *testing.T) {
	verify := func(name, msg, format, md, want string) opCase {
		return opCase{
			name, rsaSigSHA256, want,
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "RSA Verify", Args: []any{rsaOpPub2048, msg, format, md}},
			},
		}
	}
	runCases(t, []opCase{
		verify("Verify raw", "test message", "Raw", "SHA-256", "Verified OK"),
		verify("Verify hex", "74657374206d657373616765", "Hex", "SHA-256", "Verified OK"),
		verify("Verify base64", "dGVzdCBtZXNzYWdl", "Base64", "SHA-256", "Verified OK"),
		verify("Verify wrong message", "other message", "Raw", "SHA-256", "Verification Failure"),
		verify("Verify wrong digest", "test message", "Raw", "SHA-512", "Verification Failure"),
		// Verify with the PKCS#1-form public key.
		{
			"Verify PKCS#1 pub key", rsaSigSHA256, "Verified OK",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "RSA Verify", Args: []any{rsaOpPub2048PKCS1, "test message", "Raw", "SHA-256"}},
			},
		},
	})
}

// TestRSAKeyFormats exercises every accepted key encoding: PKCS#1 & SPKI public
// keys, and PKCS#1, PKCS#8 and legacy-PEM-encrypted private keys.
func TestRSAKeyFormats(t *testing.T) {
	runCases(t, []opCase{
		{
			"PKCS#1 pub encrypt + PKCS#8 priv decrypt", "hello", "hello",
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048PKCS1, "RSA-OAEP", "SHA-256"}},
				{Op: "RSA Decrypt", Args: []any{rsaOpPriv2048PKCS8, "", "RSA-OAEP", "SHA-256"}},
			},
		},
		{
			"encrypted priv decrypt (password)", "hello", "hello",
			core.Recipe{
				{Op: "RSA Encrypt", Args: []any{rsaOpPub2048, "RSA-OAEP", "SHA-256"}},
				{Op: "RSA Decrypt", Args: []any{rsaOpPriv2048Enc, "password", "RSA-OAEP", "SHA-256"}},
			},
		},
		rsaSignCase("Sign with PKCS#8 priv", rsaOpPriv2048PKCS8, "", "SHA-256", rsaSigSHA256),
		rsaSignCase("Sign with encrypted priv", rsaOpPriv2048Enc, "password", "SHA-256", rsaSigSHA256),
	})
}

// TestRSAErrors covers the operation error paths with their verbatim messages.
func TestRSAErrors(t *testing.T) {
	longOAEP := strings.Repeat("x", 215)   // > 256-2*20-2 = 214 for SHA-1
	longPKCS1 := strings.Repeat("x", 246)  // > 256-11 = 245
	shortCipher := strings.Repeat("x", 10) // != 256
	allFF := strings.Repeat("\xff", 256)   // 256 bytes, value >= modulus

	cases := []struct {
		name    string
		op      string
		input   string
		args    []any
		wantErr string
	}{
		{
			"empty pub encrypt", "RSA Encrypt", "hi",
			[]any{"-----BEGIN RSA PUBLIC KEY-----", "RSA-OAEP", "SHA-1"},
			"Please enter a public key.",
		},
		{
			"empty priv decrypt", "RSA Decrypt", "hi",
			[]any{"-----BEGIN RSA PRIVATE KEY-----", "", "RSA-OAEP", "SHA-1"},
			"Please enter a private key.",
		},
		{
			"empty priv sign", "RSA Sign", "hi",
			[]any{"-----BEGIN RSA PRIVATE KEY-----", "", "SHA-1"},
			"Please enter a private key.",
		},
		{
			"empty pub verify", "RSA Verify", "hi",
			[]any{"-----BEGIN RSA PUBLIC KEY-----", "msg", "Raw", "SHA-1"},
			"Please enter a public key.",
		},
		{
			"OAEP too long", "RSA Encrypt", longOAEP,
			[]any{rsaOpPub2048, "RSA-OAEP", "SHA-1"},
			"RSAES-OAEP input message length (215) is longer than the maximum allowed length (214).",
		},
		{
			"PKCS1 too long", "RSA Encrypt", longPKCS1,
			[]any{rsaOpPub2048, "RSAES-PKCS1-V1_5", "SHA-1"},
			"Message is too long for PKCS#1 v1.5 padding.",
		},
		{
			"decrypt bad length", "RSA Decrypt", shortCipher,
			[]any{rsaOpPriv2048, "", "RSA-OAEP", "SHA-1"},
			"Encrypted message length is invalid.",
		},
		{
			"RAW decrypt >= modulus", "RSA Decrypt", allFF,
			[]any{rsaOpPriv2048, "", "RAW", "SHA-1"},
			"Encrypted message is invalid.",
		},
		{
			"verify bad sig length", "RSA Verify", shortCipher,
			[]any{rsaOpPub2048, "msg", "Raw", "SHA-1"},
			"Signature length (10) does not match expected length based on key (256).",
		},
		{
			"wrong password", "RSA Decrypt", strings.Repeat("x", 256),
			[]any{rsaOpPriv2048Enc, "wrongpass", "RSA-OAEP", "SHA-1"},
			"",
		}, // any error
		{
			"bad key material", "RSA Sign", "hi",
			[]any{"-----BEGIN RSA PRIVATE KEY-----\nnotbase64!!!\n-----END RSA PRIVATE KEY-----", "", "SHA-1"},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, c.op, c.input, c.args...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if c.wantErr != "" && err.Error() != c.wantErr {
				t.Fatalf("got error %q\nwant %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestRSAParsing exercises every branch of the key-parsing helpers directly,
// including malformed and wrong-type keys.
func TestRSAParsing(t *testing.T) {
	pubCases := []struct{ name, key, wantErr string }{
		{"pub not PEM", "this is not a pem key", "could not parse the public key"},
		{"pub bad DER", "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----", ""},
		{"pub is EC key", p256Pub, "provided key is not an RSA public key"},
	}
	for _, c := range pubCases {
		t.Run("pub/"+c.name, func(t *testing.T) {
			_, err := rsaParsePublicKey(c.key)
			if err == nil {
				t.Fatal("expected error")
			}
			if c.wantErr != "" && err.Error() != c.wantErr {
				t.Fatalf("got %q want %q", err, c.wantErr)
			}
		})
	}

	privCases := []struct{ name, key, pass, wantErr string }{
		{"priv not PEM", "this is not a pem key", "", "could not parse the private key"},
		{"priv bad PKCS8 DER", "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----", "", ""},
		{"priv is EC PKCS8", p256PrivPkcs8, "", "provided key is not an RSA private key"},
		{"priv is EC PKCS1", p256PrivPkcs1, "", `unsupported key type "EC PRIVATE KEY"`},
		{"priv PKCS8 encrypted", rsaOpPriv2048EncPKCS8, "pw", "PKCS#8 encrypted private keys are not supported"},
		{"priv wrong password", rsaOpPriv2048Enc, "wrong", ""},
	}
	for _, c := range privCases {
		t.Run("priv/"+c.name, func(t *testing.T) {
			_, err := rsaParsePrivateKey(c.key, c.pass)
			if err == nil {
				t.Fatal("expected error")
			}
			if c.wantErr != "" && err.Error() != c.wantErr {
				t.Fatalf("got %q want %q", err, c.wantErr)
			}
		})
	}
}

// TestRSARunErrorGuards exercises the parse-error and message-decode guards in the
// operation Run methods.
func TestRSARunErrorGuards(t *testing.T) {
	badPub := "-----BEGIN PUBLIC KEY-----\nAAAA\n-----END PUBLIC KEY-----"
	// Encrypt with an unparseable (but non-empty) public key.
	if _, err := runOp(t, "RSA Encrypt", "hi", badPub, "RSA-OAEP", "SHA-1"); err == nil {
		t.Fatal("RSA Encrypt: expected parse error")
	}
	// Verify with an unparseable public key (256-byte sig to pass the length gate).
	if _, err := runOp(t, "RSA Verify", strings.Repeat("x", 256), badPub, "msg", "Raw", "SHA-1"); err == nil {
		t.Fatal("RSA Verify: expected parse error")
	}
	// Verify with an invalid Base64 message.
	if _, err := runOp(t, "RSA Verify", strings.Repeat("x", 256), rsaOpPub2048, "!!!not-base64!!!", "Base64", "SHA-1"); err == nil {
		t.Fatal("RSA Verify: expected base64 decode error")
	}
	// Decrypt with an OAEP-undecryptable 256-byte ciphertext.
	if _, err := runOp(t, "RSA Decrypt", strings.Repeat("\x01", 256), rsaOpPriv2048, "", "RSA-OAEP", "SHA-1"); err == nil {
		t.Fatal("RSA Decrypt: expected OAEP decrypt error")
	}
}

// TestRSAKeyBits covers the key-length mapping directly (generating 2048/4096-bit
// keys in every test run would be needlessly slow).
func TestRSAKeyBits(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int
	}{{"1024", 1024}, {"2048", 2048}, {"4096", 4096}, {"", 1024}} {
		if got := rsaKeyBits(c.in); got != c.want {
			t.Fatalf("rsaKeyBits(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestRSAGenerate checks that generated key pairs are well-formed and usable.
func TestRSAGenerate(t *testing.T) {
	// PEM: an SPKI public key and a PKCS#1 private key that round-trip.
	pemOut, err := runOp(t, "Generate RSA Key Pair", "", "1024", "PEM")
	if err != nil {
		t.Fatal(err)
	}
	// Output is "<public PEM>\n\n<private PEM>".
	pub, priv, ok := strings.Cut(pemOut, "\n\n")
	if !ok || !strings.HasPrefix(pub, "-----BEGIN PUBLIC KEY-----") ||
		!strings.HasPrefix(priv, "-----BEGIN RSA PRIVATE KEY-----") {
		t.Fatalf("PEM output not in expected form:\n%s", pemOut)
	}
	rt, err := core.Recipe{
		{Op: "RSA Encrypt", Args: []any{pub, "RSA-OAEP", "SHA-256"}},
		{Op: "RSA Decrypt", Args: []any{priv, "", "RSA-OAEP", "SHA-256"}},
	}.Execute(sdish("hi"))
	if err != nil || rt.String() != "hi" {
		t.Fatalf("generated PEM keypair did not round-trip: out=%q err=%v", rt.String(), err)
	}

	// DER: raw PKCS#1 DER bytes (starts with the SEQUENCE tag 0x30).
	derOut, err := runOp(t, "Generate RSA Key Pair", "", "1024", "DER")
	if err != nil {
		t.Fatal(err)
	}
	if len(derOut) == 0 || derOut[0] != 0x30 {
		t.Fatalf("DER output is not a DER SEQUENCE: % x", derOut[:min(4, len(derOut))])
	}

	// JSON: a documented cchef shape carrying the key parameters.
	jsonOut, err := runOp(t, "Generate RSA Key Pair", "", "1024", "JSON")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut, `"n"`) || !strings.Contains(jsonOut, `"d"`) {
		t.Fatalf("JSON output missing key parameters:\n%s", jsonOut)
	}
}
