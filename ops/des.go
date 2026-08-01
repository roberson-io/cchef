package ops

import (
	"crypto/cipher"
	"crypto/des" // #nosec G502 -- DES is the cipher these operations port, not a security choice
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// errDESDecrypt is CyberChef's verbatim decrypt-failure message.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errDESDecrypt = errors.New("Unable to decrypt input with these parameters.")

func init() {
	core.Register(DESEncrypt{})
	core.Register(DESDecrypt{})
	core.Register(TripleDESEncrypt{})
	core.Register(TripleDESDecrypt{})
}

// desBlockSize is the 64-bit block shared by DES and Triple DES.
const desBlockSize = 8

// desEncModes / desDecModes match CyberChef's Mode option lists. CBC/ECB use
// PKCS#7 padding; CFB/OFB/CTR are streaming. Decryption additionally offers the
// NoPadding variants, which skip unpadding.
var (
	desEncModes = []string{"CBC", "CFB", "OFB", "CTR", "ECB"}
	desDecModes = []string{"CBC", "CFB", "OFB", "CTR", "ECB", "CBC/NoPadding", "ECB/NoPadding"}
)

// desBuildBlock validates a DES key (exactly 8 bytes) and returns the cipher.
func desBuildBlock(keyArg core.ToggleString) (cipher.Block, error) {
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != desBlockSize {
		return nil, fmt.Errorf("Invalid key length: %d bytes\n\nDES uses a key length of 8 bytes (64 bits).", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return des.NewCipher(key) // #nosec G405 -- DES is the cipher this operation ports, not a security choice
}

// tripleDESBuildBlock validates a Triple DES key (16 or 24 bytes) and returns
// the cipher. A 16-byte key is expanded to K1‖K2‖K1, matching node-forge.
func tripleDESBuildBlock(keyArg core.ToggleString) (cipher.Block, error) {
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 24 {
		return nil, fmt.Errorf("Invalid key length: %d bytes\n\nTriple DES uses a key length of 24 bytes (192 bits).", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if len(key) == 16 {
		key = append(append([]byte{}, key...), key[:desBlockSize]...)
	}
	return des.NewTripleDESCipher(key) // #nosec G405 -- Triple DES is the cipher this operation ports, not a security choice
}

// desBaseMode strips the "/NoPadding" suffix, returning the underlying mode and
// whether padding should be skipped (matching CyberChef's substring(0,3) parse).
func desBaseMode(mode string) (base string, noPadding bool) {
	if before, ok := strings.CutSuffix(mode, "/NoPadding"); ok {
		return before, true
	}
	return mode, false
}

// desDecodeIV decodes the IV and validates its length (8 bytes unless ECB).
// cipherName is "DES" or "Triple DES" for the verbatim error text.
func desDecodeIV(ivArg core.ToggleString, baseMode, cipherName string) ([]byte, error) {
	iv, err := convertToByteArray(ivArg.Value, ivArg.Option)
	if err != nil {
		return nil, err
	}
	if baseMode != "ECB" && len(iv) != desBlockSize {
		return nil, fmt.Errorf("Invalid IV length: %d bytes\n\n%s uses an IV length of 8 bytes (64 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv), cipherName) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return iv, nil
}

// desEncryptBlocks applies the block cipher over the selected mode (encrypt).
func desEncryptBlocks(block cipher.Block, mode string, iv, input []byte) []byte {
	switch mode {
	case "CBC":
		data := pkcs7Pad(input, desBlockSize)
		out := make([]byte, len(data))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
		return out
	case "ECB":
		return ecbEncrypt(block, pkcs7Pad(input, desBlockSize))
	case "CFB":
		return aesCFB(block, iv, input, false)
	case "OFB":
		return aesOFB(block, iv, input)
	default: // CTR
		out := make([]byte, len(input))
		cipher.NewCTR(block, iv).XORKeyStream(out, input)
		return out
	}
}

// desDecryptBlocks reverses desEncryptBlocks, returning false when the input is
// not a whole number of blocks or the PKCS#7 padding is invalid.
func desDecryptBlocks(block cipher.Block, mode string, noPadding bool, iv, input []byte) ([]byte, bool) {
	switch mode {
	case "CBC":
		if len(input)%desBlockSize != 0 {
			return nil, false
		}
		buf := make([]byte, len(input))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(buf, input)
		if noPadding {
			return buf, true
		}
		return pkcs7Unpad(buf, desBlockSize)
	case "ECB":
		if len(input)%desBlockSize != 0 {
			return nil, false
		}
		dec := ecbDecrypt(block, input)
		if noPadding {
			return dec, true
		}
		return pkcs7Unpad(dec, desBlockSize)
	case "CFB":
		return aesCFB(block, iv, input, true), true
	case "OFB":
		return aesOFB(block, iv, input), true
	default: // CTR
		out := make([]byte, len(input))
		cipher.NewCTR(block, iv).XORKeyStream(out, input)
		return out, true
	}
}

// runDESEncrypt is the shared encrypt driver for DES and Triple DES.
func runDESEncrypt(block cipher.Block, cipherName string, in *core.Dish, args []any) (*core.Dish, error) {
	mode := args[2].(string)
	iv, err := desDecodeIV(args[1].(core.ToggleString), mode, cipherName)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))
	out := desEncryptBlocks(block, mode, iv, input)
	return blowfishOutput(out, args[4].(string)), nil
}

// runDESDecrypt is the shared decrypt driver for DES and Triple DES.
func runDESDecrypt(block cipher.Block, cipherName string, in *core.Dish, args []any) (*core.Dish, error) {
	baseMode, noPadding := desBaseMode(args[2].(string))
	iv, err := desDecodeIV(args[1].(core.ToggleString), baseMode, cipherName)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))
	out, ok := desDecryptBlocks(block, baseMode, noPadding, iv, input)
	if !ok {
		return nil, errDESDecrypt
	}
	return blowfishOutput(out, args[4].(string)), nil
}

// desEncArgs / desDecArgs build the shared argument definitions.
func desEncArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: desEncModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

func desDecArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: desDecModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// DESEncrypt encrypts input with the DES block cipher.
type DESEncrypt struct{}

// Meta returns the operation metadata.
func (DESEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "DES Encrypt",
		Module:      "Ciphers",
		Description: "DES is a previously dominant algorithm for encryption, and was published as an official U.S. Federal Information Processing Standard (FIPS). It is now considered to be insecure due to its small key size.<br><br><b>Key:</b> DES uses a key length of 8 bytes (64 bits).<br><br>You can generate a password-based key using one of the KDF operations.<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, PKCS#7 padding will be used.",
		InfoURL:     "https://wikipedia.org/wiki/Data_Encryption_Standard",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DESEncrypt) Args() []core.ArgDef { return desEncArgs() }

// Run performs the encryption. Ported from CyberChef DESEncrypt.mjs.
func (DESEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	block, err := desBuildBlock(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	return runDESEncrypt(block, "DES", in, args)
}

// DESDecrypt decrypts DES ciphertext.
type DESDecrypt struct{}

// Meta returns the operation metadata.
func (DESDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "DES Decrypt",
		Module:      "Ciphers",
		Description: "DES is a previously dominant algorithm for encryption, and was published as an official U.S. Federal Information Processing Standard (FIPS). It is now considered to be insecure due to its small key size.<br><br><b>Key:</b> DES uses a key length of 8 bytes (64 bits).<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, PKCS#7 padding will be used as a default.",
		InfoURL:     "https://wikipedia.org/wiki/Data_Encryption_Standard",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DESDecrypt) Args() []core.ArgDef { return desDecArgs() }

// Run performs the decryption. Ported from CyberChef DESDecrypt.mjs.
func (DESDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	block, err := desBuildBlock(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	return runDESDecrypt(block, "DES", in, args)
}

// TripleDESEncrypt encrypts input with the Triple DES block cipher.
type TripleDESEncrypt struct{}

// Meta returns the operation metadata.
func (TripleDESEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Triple DES Encrypt",
		Module:      "Ciphers",
		Description: "Triple DES applies DES three times to each block to increase key size.<br><br><b>Key:</b> Triple DES uses a key length of 24 bytes (192 bits).<br><br>You can generate a password-based key using one of the KDF operations.<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, PKCS#7 padding will be used.",
		InfoURL:     "https://wikipedia.org/wiki/Triple_DES",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TripleDESEncrypt) Args() []core.ArgDef { return desEncArgs() }

// Run performs the encryption. Ported from CyberChef TripleDESEncrypt.mjs.
func (TripleDESEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	block, err := tripleDESBuildBlock(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	return runDESEncrypt(block, "Triple DES", in, args)
}

// TripleDESDecrypt decrypts Triple DES ciphertext.
type TripleDESDecrypt struct{}

// Meta returns the operation metadata.
func (TripleDESDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Triple DES Decrypt",
		Module:      "Ciphers",
		Description: "Triple DES applies DES three times to each block to increase key size.<br><br><b>Key:</b> Triple DES uses a key length of 24 bytes (192 bits).<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, PKCS#7 padding will be used as a default.",
		InfoURL:     "https://wikipedia.org/wiki/Triple_DES",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TripleDESDecrypt) Args() []core.ArgDef { return desDecArgs() }

// Run performs the decryption. Ported from CyberChef TripleDESDecrypt.mjs.
func (TripleDESDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	block, err := tripleDESBuildBlock(args[0].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	return runDESDecrypt(block, "Triple DES", in, args)
}
