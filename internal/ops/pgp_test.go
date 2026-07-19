package ops

import (
	"io"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"

	"github.com/roberson-io/cchef/internal/core"
)

const (
	pgpASCII = "A common mistake that people make when trying to design something completely foolproof is to underestimate the ingenuity of complete fools."
	pgpUTF8  = "Шанцы на высвятленне таго, што адбываецца на самай справе ў сусвеце настолькі выдаленыя, адзінае, што трэба зрабіць, гэта павесіць пачуццё яго і трымаць сябе занятымі."
)

// TestPGPDecryptFixed decrypts a fixed kbpgp-produced message (interop with
// Keybase's OpenPGP). From ../CyberChef/tests/operations/tests/PGP.mjs.
func TestPGPDecryptFixed(t *testing.T) {
	got, err := runOp(t, "PGP Decrypt", pgpFixedDecMsg, pgpAlicePriv, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != pgpASCII {
		t.Errorf("decrypt fixed: got %q", got)
	}
}

// TestPGPVerifyFixed verifies a fixed kbpgp-signed message, including the exact
// signer metadata (key ID, fingerprint, signing time).
func TestPGPVerifyFixed(t *testing.T) {
	want := `Signed by PGP key ID: 2ADF8D8C
PGP fingerprint: 7afe93ff7614167c3fe831fe1b75204b2adf8d8c
Signed on Wed, 27 May 2026 19:03:45 GMT
----------------------------------
` + pgpASCII
	got, err := runOp(t, "PGP Verify", pgpFixedVerifyMsg, pgpAlicePub)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("verify fixed:\n got %q\nwant %q", got, want)
	}
}

// TestPGPRoundTrips covers encrypt/decrypt, sign/verify, and
// encrypt+sign/decrypt+verify across RSA (Alice) and ECC (Bob) fixture keys.
func TestPGPRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		name, priv, pub, signerPub, msg string
	}{
		{"RSA empty", pgpAlicePriv, pgpAlicePub, pgpAlicePub, ""},
		{"RSA text", pgpAlicePriv, pgpAlicePub, pgpAlicePub, pgpASCII},
		{"ECC text", pgpBobPriv, pgpBobPub, pgpBobPub, pgpUTF8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt -> Decrypt
			out, err := core.Recipe{
				{Op: "PGP Encrypt", Args: []any{tc.pub}},
				{Op: "PGP Decrypt", Args: []any{tc.priv, ""}},
			}.Execute(core.NewDish([]byte(tc.msg), core.TypeString))
			if err != nil || out.String() != tc.msg {
				t.Fatalf("encrypt/decrypt: got %q err %v", out.String(), err)
			}

			// Sign -> Verify (verify output ends with the message)
			out, err = core.Recipe{
				{Op: "PGP Sign", Args: []any{tc.priv, ""}},
				{Op: "PGP Verify", Args: []any{tc.signerPub}},
			}.Execute(core.NewDish([]byte(tc.msg), core.TypeString))
			if err != nil || !strings.HasSuffix(out.String(), tc.msg) || !strings.HasPrefix(out.String(), "Signed by") {
				t.Fatalf("sign/verify: got %q err %v", out.String(), err)
			}

			// Encrypt and Sign -> Decrypt and Verify
			out, err = core.Recipe{
				{Op: "PGP Encrypt and Sign", Args: []any{tc.priv, "", tc.pub}},
				{Op: "PGP Decrypt and Verify", Args: []any{tc.signerPub, tc.priv, ""}},
			}.Execute(core.NewDish([]byte(tc.msg), core.TypeString))
			if err != nil || !strings.HasSuffix(out.String(), tc.msg) || !strings.HasPrefix(out.String(), "Signed by") {
				t.Fatalf("encrypt+sign/decrypt+verify: got %q err %v", out.String(), err)
			}
		})
	}
}

// TestPGPErrors covers the missing-key validation paths.
func TestPGPErrors(t *testing.T) {
	cases := []struct {
		op   string
		args []any
		want string
	}{
		{"PGP Encrypt", []any{""}, "Enter the public key of the recipient."},
		{"PGP Decrypt", []any{"", ""}, "Enter the private key of the recipient."},
		{"PGP Sign", []any{"", ""}, "Enter the private key of the signer."},
		{"PGP Verify", []any{""}, "Enter the public key of the signer."},
		{"PGP Encrypt and Sign", []any{"", "", pgpAlicePub}, "Enter the private key of the signer."},
		{"PGP Encrypt and Sign", []any{pgpAlicePriv, "", ""}, "Enter the public key of the recipient."},
		{"PGP Decrypt and Verify", []any{"", pgpAlicePriv, ""}, "Enter the public key of the signer."},
		{"PGP Decrypt and Verify", []any{pgpAlicePub, "", ""}, "Enter the private key of the recipient."},
	}
	for _, c := range cases {
		_, err := runOp(t, c.op, "data", c.args...)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: got %v, want %q", c.op, err, c.want)
		}
	}

	// Verifying unsigned data reports it is not signed.
	enc, err := runOp(t, "PGP Encrypt", "hi", pgpAlicePub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runOp(t, "PGP Verify", enc, pgpAlicePub); err == nil {
		t.Error("verify of unsigned message: expected error")
	}
}

// TestPGPGenerate generates each supported key type and round-trips it through
// encrypt/decrypt and sign/verify.
func TestPGPGenerate(t *testing.T) {
	for _, kt := range []string{"RSA-1024", "ECC-256"} {
		t.Run(kt, func(t *testing.T) {
			pair, err := runOp(t, "Generate PGP Key Pair", "", kt, "", "Alice", "alice@example.com")
			if err != nil {
				t.Fatal(err)
			}
			priv := pgpBlock(t, pair, "PRIVATE")
			pub := pgpBlock(t, pair, "PUBLIC")

			out, err := core.Recipe{
				{Op: "PGP Encrypt", Args: []any{pub}},
				{Op: "PGP Decrypt", Args: []any{priv, ""}},
			}.Execute(core.NewDish([]byte("hello"), core.TypeString))
			if err != nil || out.String() != "hello" {
				t.Fatalf("%s encrypt/decrypt: got %q err %v", kt, out.String(), err)
			}

			out, err = core.Recipe{
				{Op: "PGP Sign", Args: []any{priv, ""}},
				{Op: "PGP Verify", Args: []any{pub}},
			}.Execute(core.NewDish([]byte("hello"), core.TypeString))
			if err != nil || !strings.Contains(out.String(), "Alice <alice@example.com>") {
				t.Fatalf("%s sign/verify: got %q err %v", kt, out.String(), err)
			}
		})
	}
}

// pgpBlock extracts a single armored PGP block (PRIVATE or PUBLIC) from text.
func pgpBlock(t *testing.T, text, kind string) string {
	t.Helper()
	begin := "-----BEGIN PGP " + kind + " KEY BLOCK-----"
	end := "-----END PGP " + kind + " KEY BLOCK-----"
	i := strings.Index(text, begin)
	j := strings.Index(text, end)
	if i < 0 || j < 0 {
		t.Fatalf("no %s block in generated key", kind)
	}
	return text[i : j+len(end)]
}

// TestPGPGenConfig covers the key-type to config mapping for every curve/size.
func TestPGPGenConfig(t *testing.T) {
	cases := map[string]string{"ECC-256": "P256", "ECC-384": "P384", "ECC-521": "P521"}
	for kt, wantCurve := range cases {
		if cfg := pgpGenConfig(kt); string(cfg.Curve) != wantCurve {
			t.Errorf("%s: curve %q", kt, cfg.Curve)
		}
	}
	if cfg := pgpGenConfig("RSA-2048"); cfg.RSABits != 2048 {
		t.Errorf("RSA-2048: RSABits %d", cfg.RSABits)
	}
}

// TestPGPImportErrors covers malformed key and message error paths.
func TestPGPImportErrors(t *testing.T) {
	bad := "not a pgp key"
	checks := []struct {
		op   string
		args []any
		in   string
	}{
		{"PGP Encrypt", []any{bad}, "hi"},
		{"PGP Decrypt", []any{bad, ""}, "hi"},
		{"PGP Sign", []any{bad, ""}, "hi"},
		{"PGP Verify", []any{bad}, "hi"},
		{"PGP Encrypt and Sign", []any{bad, "", pgpAlicePub}, "hi"},
		{"PGP Encrypt and Sign", []any{pgpAlicePriv, "", bad}, "hi"},
		{"PGP Decrypt and Verify", []any{bad, pgpAlicePriv, ""}, "hi"},
		{"PGP Decrypt and Verify", []any{pgpAlicePub, bad, ""}, "hi"},
		// Valid key but malformed message.
		{"PGP Decrypt", []any{pgpAlicePriv, ""}, "not a pgp message"},
		{"PGP Verify", []any{pgpAlicePub}, "not a pgp message"},
		{"PGP Decrypt and Verify", []any{pgpAlicePub, pgpAlicePriv, ""}, "not a pgp message"},
	}
	for _, c := range checks {
		if _, err := runOp(t, c.op, c.in, c.args...); err == nil {
			t.Errorf("%s with %q: expected error", c.op, c.args)
		}
	}
}

// TestPGPPassphrase covers passphrase-protected private keys (generation and the
// locked/unlock paths).
func TestPGPPassphrase(t *testing.T) {
	pair, err := runOp(t, "Generate PGP Key Pair", "", "ECC-256", "s3cret", "", "")
	if err != nil {
		t.Fatal(err)
	}
	priv := pgpBlock(t, pair, "PRIVATE")
	pub := pgpBlock(t, pair, "PUBLIC")
	ct, err := runOp(t, "PGP Encrypt", "hi", pub)
	if err != nil {
		t.Fatal(err)
	}
	// Correct passphrase unlocks and decrypts.
	if got, err := runOp(t, "PGP Decrypt", ct, priv, "s3cret"); err != nil || got != "hi" {
		t.Errorf("correct passphrase: got %q err %v", got, err)
	}
	// No passphrase on a locked key.
	if _, err := runOp(t, "PGP Decrypt", ct, priv, ""); err == nil || err.Error() != "Did not provide passphrase with locked private key." {
		t.Errorf("no passphrase: got %v", err)
	}
	// Wrong passphrase.
	if _, err := runOp(t, "PGP Decrypt", ct, priv, "wrong"); err == nil {
		t.Error("wrong passphrase: expected error")
	}
}

// TestPGPNotSigned covers the "not signed" path via Decrypt and Verify of an
// encrypted-but-unsigned message.
func TestPGPNotSigned(t *testing.T) {
	enc, err := runOp(t, "PGP Encrypt", "hi", pgpAlicePub)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runOp(t, "PGP Decrypt and Verify", enc, pgpAlicePub, pgpAlicePriv, "")
	if err == nil || err.Error() != "The data does not appear to be signed." {
		t.Errorf("got %v, want not-signed error", err)
	}
}

// TestPGPVerifyComment covers the signer-comment branch of the verify output,
// using an in-test key whose user ID carries a comment.
func TestPGPVerifyComment(t *testing.T) {
	e, err := openpgp.NewEntity("Bob", "the builder", "bob@example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := pgpArmor(openpgp.PrivateKeyType, func(w io.Writer) error { return e.SerializePrivateWithoutSigning(w, nil) })
	if err != nil {
		t.Fatal(err)
	}
	pub, err := pgpArmor(openpgp.PublicKeyType, e.Serialize)
	if err != nil {
		t.Fatal(err)
	}
	out, err := core.Recipe{
		{Op: "PGP Sign", Args: []any{priv, ""}},
		{Op: "PGP Verify", Args: []any{pub}},
	}.Execute(core.NewDish([]byte("hi"), core.TypeString))
	if err != nil || !strings.Contains(out.String(), "Bob (the builder) <bob@example.com>") {
		t.Errorf("comment verify: got %q err %v", out.String(), err)
	}
}
