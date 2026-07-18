package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SM4Encrypt{})
	core.Register(SM4Decrypt{})
}

// sm4Modes are the block cipher mode choices shared by both operations.
var sm4Modes = []string{"CBC", "CFB", "OFB", "CTR", "ECB", "CBC/NoPadding", "ECB/NoPadding"}

// sm4Args builds the shared argument list for both SM4 operations.
func sm4Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: sm4Modes},
		{Name: "Input", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
	}
}

// sm4Inputs parses and validates the shared key/IV/input arguments.
func sm4Inputs(in *core.Dish, args []any) (key, iv, input []byte, mode string, noPad bool, err error) {
	ks := args[0].(core.ToggleString)
	ivs := args[1].(core.ToggleString)
	mode = args[2].(string)
	if key, err = convertToByteArray(ks.Value, ks.Option); err != nil {
		return
	}
	if iv, err = convertToByteArray(ivs.Value, ivs.Option); err != nil {
		return
	}
	if len(key) != 16 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid key length: %d bytes\n\nSM4 uses a key length of 16 bytes (128 bits).", len(key))
		return
	}
	if len(iv) != 16 && mode[:3] != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		err = fmt.Errorf("Invalid IV length: %d bytes\n\nSM4 uses an IV length of 16 bytes (128 bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", len(iv))
		return
	}
	input = decodeAESInput(in, args[3].(string))
	noPad = len(mode) > 3 // "CBC/NoPadding" / "ECB/NoPadding"
	return
}

// sm4Output formats the result as space-delimited hex or a raw string.
func sm4Output(out []byte, outType string) *core.Dish {
	if outType == "Hex" {
		return core.NewDish([]byte(toHexSpace(out)), core.TypeString)
	}
	return core.NewDish([]byte(byteArrayToUtf8(out)), core.TypeString)
}

// SM4Encrypt encrypts with the SM4 block cipher.
type SM4Encrypt struct{}

// Meta returns the operation metadata.
func (SM4Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM4 Encrypt",
		Module:      "Ciphers",
		Description: "SM4 is a 128-bit block cipher, currently established as a national standard (GB/T 32907-2016) of China. Multiple block cipher modes are supported. When using CBC or ECB mode, the PKCS#7 padding scheme is used.",
		InfoURL:     "https://wikipedia.org/wiki/SM4_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SM4Encrypt) Args() []core.ArgDef { return sm4Args() }

// Run encrypts with SM4. Ported from CyberChef SM4Encrypt.mjs.
func (SM4Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, noPad, err := sm4Inputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := sm4Encrypt(input, key, iv, mode[:3], noPad)
	if err != nil {
		return nil, err
	}
	return sm4Output(out, args[4].(string)), nil
}

// SM4Decrypt decrypts with the SM4 block cipher.
type SM4Decrypt struct{}

// Meta returns the operation metadata.
func (SM4Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM4 Decrypt",
		Module:      "Ciphers",
		Description: "SM4 is a 128-bit block cipher, currently established as a national standard (GB/T 32907-2016) of China.",
		InfoURL:     "https://wikipedia.org/wiki/SM4_(cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SM4Decrypt) Args() []core.ArgDef { return sm4Args() }

// Run decrypts with SM4. Ported from CyberChef SM4Decrypt.mjs.
func (SM4Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, iv, input, mode, noPad, err := sm4Inputs(in, args)
	if err != nil {
		return nil, err
	}
	out, err := sm4Decrypt(input, key, iv, mode[:3], noPad)
	if err != nil {
		return nil, err
	}
	return sm4Output(out, args[4].(string)), nil
}
