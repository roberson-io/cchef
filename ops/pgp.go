package ops

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(PGPEncrypt{})
	core.Register(PGPDecrypt{})
	core.Register(PGPSign{})
	core.Register(PGPVerify{})
	core.Register(PGPEncryptAndSign{})
	core.Register(PGPDecryptAndVerify{})
	core.Register(GeneratePGPKeyPair{})
}

// pgpMessageType is the armor block type for an OpenPGP message.
const pgpMessageType = "PGP MESSAGE"

// pgpImportPublic reads an ASCII-armoured public key.
func pgpImportPublic(armored string) (*openpgp.Entity, error) {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil || len(el) == 0 {
		return nil, fmt.Errorf("Could not import public key: %w", err) //nolint:staticcheck,revive // CyberChef-style text
	}
	return el[0], nil
}

// pgpImportPrivate reads an ASCII-armoured private key, unlocking it with the
// passphrase if it is encrypted.
func pgpImportPrivate(armored, passphrase string) (*openpgp.Entity, error) {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil || len(el) == 0 {
		return nil, fmt.Errorf("Could not import private key: %w", err) //nolint:staticcheck,revive // CyberChef-style text
	}
	key := el[0]
	if key.PrivateKey != nil && key.PrivateKey.Encrypted {
		if passphrase == "" {
			return nil, errors.New("Did not provide passphrase with locked private key.") //nolint:staticcheck,revive // verbatim CyberChef text
		}
		if err := key.DecryptPrivateKeys([]byte(passphrase)); err != nil {
			return nil, fmt.Errorf("Could not import private key: %w", err) //nolint:staticcheck,revive // CyberChef-style text
		}
	}
	return key, nil
}

// pgpArmor runs write against an armor encoder of the given block type and
// returns the resulting ASCII-armoured string.
func pgpArmor(blockType string, write func(io.Writer) error) (string, error) {
	var buf bytes.Buffer
	w, err := armor.Encode(&buf, blockType, nil)
	if err != nil {
		return "", err
	}
	if err := write(w); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// pgpReadMessage armor-decodes input and reads the OpenPGP message, returning the
// message details and the (decrypted) body.
func pgpReadMessage(input string, keyring openpgp.EntityList) (*openpgp.MessageDetails, []byte, error) {
	block, err := armor.Decode(strings.NewReader(input))
	if err != nil {
		return nil, nil, err
	}
	md, err := openpgp.ReadMessage(block.Body, keyring, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	body, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, nil, err
	}
	return md, body, nil
}

// pgpSignerText builds the "Signed by …" header describing the verified signer,
// matching CyberChef's PGP Verify output. It must be called after the message
// body has been fully read (so the signature is available).
func pgpSignerText(md *openpgp.MessageDetails) (string, error) {
	if !md.IsSigned || md.SignedBy == nil {
		return "", errors.New("The data does not appear to be signed.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if md.SignatureError != nil {
		return "", fmt.Errorf("Couldn't verify message: %w", md.SignatureError) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	entity := md.SignedBy.Entity
	var b strings.Builder
	b.WriteString("Signed by ")
	if id := entity.PrimaryIdentity(); id != nil && id.UserId != nil {
		u := id.UserId
		if u.Email != "" || u.Name != "" || u.Comment != "" {
			if u.Name != "" {
				b.WriteString(u.Name + " ")
			}
			if u.Comment != "" {
				b.WriteString("(" + u.Comment + ") ")
			}
			if u.Email != "" {
				b.WriteString("<" + u.Email + ">")
			}
			b.WriteString("\n")
		}
	}
	fmt.Fprintf(&b, "PGP key ID: %s\n", entity.PrimaryKey.KeyIdShortString())
	fmt.Fprintf(&b, "PGP fingerprint: %s\n", hex.EncodeToString(entity.PrimaryKey.Fingerprint))
	fmt.Fprintf(&b, "Signed on %s\n", md.Signature.CreationTime.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	b.WriteString("----------------------------------\n")
	return b.String(), nil
}

// pgpArgString returns a text argument definition.
func pgpArgString(name string) core.ArgDef {
	return core.ArgDef{Name: name, Type: core.ArgString, Value: ""}
}

// PGPEncrypt encrypts a message with a recipient's public key.
type PGPEncrypt struct{}

// Meta returns the operation metadata.
func (PGPEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Encrypt",
		Module:      "PGP",
		Description: "Input: the message you want to encrypt.\n\nArguments: the ASCII-armoured PGP public key of the recipient.\n\nPretty Good Privacy is an encryption standard (OpenPGP) used for encrypting, decrypting, and signing messages.",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{pgpArgString("Public key of recipient")}
}

// Run encrypts the input.
func (PGPEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	pubKey := args[0].(string)
	if pubKey == "" {
		return nil, errors.New("Enter the public key of the recipient.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	key, err := pgpImportPublic(pubKey)
	if err != nil {
		return nil, err
	}
	out, err := pgpArmor(pgpMessageType, func(w io.Writer) error {
		pt, err := openpgp.Encrypt(w, []*openpgp.Entity{key}, nil, nil, nil)
		if err != nil {
			return err
		}
		if _, err := pt.Write(in.Bytes()); err != nil {
			return err
		}
		return pt.Close()
	})
	if err != nil {
		return nil, fmt.Errorf("Couldn't encrypt message with provided public key: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// PGPDecrypt decrypts a message with the recipient's private key.
type PGPDecrypt struct{}

// Meta returns the operation metadata.
func (PGPDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Decrypt",
		Module:      "PGP",
		Description: "Input: the ASCII-armoured PGP message you want to decrypt.\n\nArguments: the ASCII-armoured PGP private key of the recipient, (and the private key password if necessary).",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPDecrypt) Args() []core.ArgDef {
	return []core.ArgDef{pgpArgString("Private key of recipient"), pgpArgString("Private key passphrase")}
}

// Run decrypts the input.
func (PGPDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	privKey := args[0].(string)
	if privKey == "" {
		return nil, errors.New("Enter the private key of the recipient.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	key, err := pgpImportPrivate(privKey, args[1].(string))
	if err != nil {
		return nil, err
	}
	_, body, err := pgpReadMessage(in.String(), openpgp.EntityList{key})
	if err != nil {
		return nil, fmt.Errorf("Couldn't decrypt message with provided private key: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return core.NewDish(body, core.TypeString), nil
}

// PGPSign signs a message with the signer's private key.
type PGPSign struct{}

// Meta returns the operation metadata.
func (PGPSign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Sign",
		Module:      "PGP",
		Description: "Input: the cleartext you want to sign.\n\nArguments: the ASCII-armoured private key of the signer (plus the private key password if necessary).",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPSign) Args() []core.ArgDef {
	return []core.ArgDef{pgpArgString("Private key of signer"), pgpArgString("Private key passphrase (optional)")}
}

// Run signs the input.
func (PGPSign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	privKey := args[0].(string)
	if privKey == "" {
		return nil, errors.New("Enter the private key of the signer.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	key, err := pgpImportPrivate(privKey, args[1].(string))
	if err != nil {
		return nil, err
	}
	out, err := pgpArmor(pgpMessageType, func(w io.Writer) error {
		sw, err := openpgp.Sign(w, key, nil, nil)
		if err != nil {
			return err
		}
		if _, err := sw.Write(in.Bytes()); err != nil {
			return err
		}
		return sw.Close()
	})
	if err != nil {
		return nil, fmt.Errorf("Couldn't sign message: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// PGPVerify verifies a signed message with the signer's public key.
type PGPVerify struct{}

// Meta returns the operation metadata.
func (PGPVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Verify",
		Module:      "PGP",
		Description: "Input: the ASCII-armoured PGP signed message you want to verify.\n\nArgument: the ASCII-armoured PGP public key of the signer.",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPVerify) Args() []core.ArgDef {
	return []core.ArgDef{pgpArgString("Public key of signer")}
}

// Run verifies the input.
func (PGPVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	pubKey := args[0].(string)
	if pubKey == "" {
		return nil, errors.New("Enter the public key of the signer.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	key, err := pgpImportPublic(pubKey)
	if err != nil {
		return nil, err
	}
	md, body, err := pgpReadMessage(in.String(), openpgp.EntityList{key})
	if err != nil {
		return nil, fmt.Errorf("Couldn't verify message: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	header, err := pgpSignerText(md)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(strings.TrimSpace(header+string(body))), core.TypeString), nil
}

// PGPEncryptAndSign encrypts and signs a message.
type PGPEncryptAndSign struct{}

// Meta returns the operation metadata.
func (PGPEncryptAndSign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Encrypt and Sign",
		Module:      "PGP",
		Description: "Input: the message you want to sign.\n\nArguments: the ASCII-armoured private key of the signer (plus the private key password if necessary), and the ASCII-armoured PGP public key of the recipient.\n\nThis operation uses PGP to produce an encrypted digital signature.",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPEncryptAndSign) Args() []core.ArgDef {
	return []core.ArgDef{
		pgpArgString("Private key of signer"),
		pgpArgString("Private key passphrase"),
		pgpArgString("Public key of recipient"),
	}
}

// Run encrypts and signs the input.
func (PGPEncryptAndSign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	privKey := args[0].(string)
	pubKey := args[2].(string)
	if privKey == "" {
		return nil, errors.New("Enter the private key of the signer.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if pubKey == "" {
		return nil, errors.New("Enter the public key of the recipient.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	signer, err := pgpImportPrivate(privKey, args[1].(string))
	if err != nil {
		return nil, err
	}
	recipient, err := pgpImportPublic(pubKey)
	if err != nil {
		return nil, err
	}
	out, err := pgpArmor(pgpMessageType, func(w io.Writer) error {
		pt, err := openpgp.Encrypt(w, []*openpgp.Entity{recipient}, signer, nil, nil)
		if err != nil {
			return err
		}
		if _, err := pt.Write(in.Bytes()); err != nil {
			return err
		}
		return pt.Close()
	})
	if err != nil {
		return nil, fmt.Errorf("Couldn't sign message: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// PGPDecryptAndVerify decrypts and verifies a message.
type PGPDecryptAndVerify struct{}

// Meta returns the operation metadata.
func (PGPDecryptAndVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "PGP Decrypt and Verify",
		Module:      "PGP",
		Description: "Input: the ASCII-armoured encrypted PGP message you want to verify.\n\nArguments: the ASCII-armoured PGP public key of the signer, the ASCII-armoured private key of the recipient (and the private key password if necessary).\n\nThis operation uses PGP to decrypt and verify an encrypted digital signature.",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PGPDecryptAndVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		pgpArgString("Public key of signer"),
		pgpArgString("Private key of recipient"),
		pgpArgString("Private key password"),
	}
}

// Run decrypts and verifies the input.
func (PGPDecryptAndVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	pubKey := args[0].(string)
	privKey := args[1].(string)
	if pubKey == "" {
		return nil, errors.New("Enter the public key of the signer.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if privKey == "" {
		return nil, errors.New("Enter the private key of the recipient.") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	recipient, err := pgpImportPrivate(privKey, args[2].(string))
	if err != nil {
		return nil, err
	}
	signer, err := pgpImportPublic(pubKey)
	if err != nil {
		return nil, err
	}
	md, body, err := pgpReadMessage(in.String(), openpgp.EntityList{recipient, signer})
	if err != nil {
		return nil, fmt.Errorf("Couldn't verify message: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	header, err := pgpSignerText(md)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(strings.TrimSpace(header+string(body))), core.TypeString), nil
}

// GeneratePGPKeyPair generates a PGP key pair.
type GeneratePGPKeyPair struct{}

// Meta returns the operation metadata.
func (GeneratePGPKeyPair) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate PGP Key Pair",
		Module:      "PGP",
		Description: "Generates a new public/private PGP key pair. Supports RSA and Elliptic Curve (EC) keys.",
		InfoURL:     "https://wikipedia.org/wiki/Pretty_Good_Privacy",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GeneratePGPKeyPair) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key type", Type: core.ArgOption, Value: []string{"RSA-1024", "RSA-2048", "RSA-4096", "ECC-256", "ECC-384", "ECC-521"}},
		pgpArgString("Password (optional)"),
		pgpArgString("Name (optional)"),
		pgpArgString("Email (optional)"),
	}
}

// pgpGenConfig maps a "TYPE-SIZE" key selector (constrained by the option arg)
// onto a go-crypto key config.
func pgpGenConfig(keyType string) *packet.Config {
	algo, sizeStr, _ := strings.Cut(keyType, "-")
	size, _ := strconv.Atoi(sizeStr)
	cfg := &packet.Config{}
	if strings.EqualFold(algo, "ecc") {
		cfg.Algorithm = packet.PubKeyAlgoECDSA
		switch size {
		case 256:
			cfg.Curve = packet.CurveNistP256
		case 384:
			cfg.Curve = packet.CurveNistP384
		case 521:
			cfg.Curve = packet.CurveNistP521
		}
	} else {
		cfg.Algorithm = packet.PubKeyAlgoRSA
		cfg.RSABits = size
	}
	return cfg
}

// Run generates the key pair.
func (GeneratePGPKeyPair) Run(_ *core.Dish, args []any) (*core.Dish, error) {
	cfg := pgpGenConfig(args[0].(string))
	password := args[1].(string)
	entity, err := openpgp.NewEntity(args[2].(string), "", args[3].(string), cfg)
	if err != nil {
		return nil, fmt.Errorf("Error whilst generating key pair: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	if password != "" {
		if err := entity.EncryptPrivateKeys([]byte(password), cfg); err != nil {
			return nil, fmt.Errorf("Error whilst generating key pair: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
		}
	}
	privateKey, err := pgpArmor(openpgp.PrivateKeyType, func(w io.Writer) error {
		return entity.SerializePrivateWithoutSigning(w, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("Error whilst generating key pair: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	publicKey, err := pgpArmor(openpgp.PublicKeyType, entity.Serialize)
	if err != nil {
		return nil, fmt.Errorf("Error whilst generating key pair: %w", err) //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return core.NewDish([]byte(privateKey+"\n"+strings.TrimSpace(publicKey)), core.TypeString), nil
}
