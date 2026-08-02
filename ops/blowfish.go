package ops

import (
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // Blowfish is the cipher this operation ports; the deprecation only advises AES for new designs.

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(BlowfishEncrypt{})
	core.Register(BlowfishDecrypt{})
}

// blowfishModes matches CyberChef's Mode option list. CBC/ECB use PKCS#7 padding
// (via node-forge); CFB/OFB/CTR are streaming and leave the length unchanged.
var blowfishModes = []string{"CBC", "CFB", "OFB", "CTR", "ECB"}

// blowfishBlockSize is Blowfish's 64-bit block.
const blowfishBlockSize = 8

// blowfishNewCipher is a seam so tests can exercise the cipher-construction
// error branch (unreachable once the 4-56 byte key length is validated).
var blowfishNewCipher = func(key []byte) (cipher.Block, error) { return blowfish.NewCipher(key) }

// blowfishArgs decodes and validates the key and IV shared by both operations.
func blowfishArgs(keyArg, ivArg core.ToggleString, mode string) (key, iv []byte, err error) {
	if key, err = convertToByteArray(keyArg.Value, keyArg.Option); err != nil {
		return nil, nil, err
	}
	if iv, err = convertToByteArray(ivArg.Value, ivArg.Option); err != nil {
		return nil, nil, err
	}
	if len(key) < 4 || len(key) > 56 {
		return nil, nil, fmt.Errorf("Invalid key length: %d bytes\n\nBlowfish's key length needs to be between 4 and 56 bytes (32-448 bits).", len(key)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if mode != "ECB" && len(iv) != 8 {
		return nil, nil, fmt.Errorf("Invalid IV length: %d bytes. Expected 8 bytes.", len(iv)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return key, iv, nil
}

// blowfishOutput renders the result as hex or raw bytes per the Output option.
func blowfishOutput(data []byte, outputType string) *core.Dish {
	if outputType == "Hex" {
		return core.NewDish([]byte(hex.EncodeToString(data)), core.TypeString)
	}
	return core.NewDish(data, core.TypeString)
}

// BlowfishEncrypt encrypts input with the Blowfish block cipher.
type BlowfishEncrypt struct{}

// Meta returns the operation metadata.
func (BlowfishEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Blowfish Encrypt",
		Module:      "Ciphers",
		Description: "Blowfish is a symmetric-key block cipher designed in 1993 by Bruce Schneier and included in a large number of cipher suites and encryption products. AES now receives more attention.<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.",
		InfoURL:     "https://wikipedia.org/wiki/Blowfish_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BlowfishEncrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: blowfishModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// Run performs the encryption.
func (BlowfishEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	mode := args[2].(string)
	key, iv, err := blowfishArgs(args[0].(core.ToggleString), args[1].(core.ToggleString), mode)
	if err != nil {
		return nil, err
	}
	block, err := blowfishNewCipher(key)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))

	var out []byte
	switch mode {
	case "CBC":
		data := pkcs7Pad(input, blowfishBlockSize)
		out = make([]byte, len(data))
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
	case "ECB":
		out = ecbEncrypt(block, pkcs7Pad(input, blowfishBlockSize))
	case "CFB":
		out = aesCFB(block, iv, input, false)
	case "OFB":
		out = aesOFB(block, iv, input)
	case "CTR":
		out = make([]byte, len(input))
		cipher.NewCTR(block, iv).XORKeyStream(out, input)
	}
	return blowfishOutput(out, args[4].(string)), nil
}

// BlowfishDecrypt decrypts Blowfish ciphertext.
type BlowfishDecrypt struct{}

// Meta returns the operation metadata.
func (BlowfishDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Blowfish Decrypt",
		Module:      "Ciphers",
		Description: "Blowfish is a symmetric-key block cipher designed in 1993 by Bruce Schneier and included in a large number of cipher suites and encryption products. AES now receives more attention.<br><br><b>IV:</b> The Initialization Vector should be 8 bytes long. If not entered, it will default to 8 null bytes.",
		InfoURL:     "https://wikipedia.org/wiki/Blowfish_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BlowfishDecrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: blowfishModes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
	}
}

// Run performs the decryption.
func (BlowfishDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	mode := args[2].(string)
	key, iv, err := blowfishArgs(args[0].(core.ToggleString), args[1].(core.ToggleString), mode)
	if err != nil {
		return nil, err
	}
	block, err := blowfishNewCipher(key)
	if err != nil {
		return nil, err
	}
	input := decodeAESInput(in, args[3].(string))

	var out []byte
	ok := true
	switch mode {
	case "CBC":
		if len(input)%blowfishBlockSize != 0 {
			ok = false
			break
		}
		buf := make([]byte, len(input))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(buf, input)
		out, ok = pkcs7Unpad(buf, blowfishBlockSize)
	case "ECB":
		if len(input)%blowfishBlockSize != 0 {
			ok = false
			break
		}
		out, ok = pkcs7Unpad(ecbDecrypt(block, input), blowfishBlockSize)
	case "CFB":
		out = aesCFB(block, iv, input, true)
	case "OFB":
		out = aesOFB(block, iv, input)
	case "CTR":
		out = make([]byte, len(input))
		cipher.NewCTR(block, iv).XORKeyStream(out, input)
	}
	if !ok {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Unable to decrypt input with these parameters.")
	}
	return blowfishOutput(out, args[4].(string)), nil
}
