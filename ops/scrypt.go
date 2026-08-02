package ops

import (
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/scrypt"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Scrypt{})
}

// scryptMaxValue is scryptsy's 0x7fffffff bound used in its N/r size checks.
const scryptMaxValue = 0x7fffffff

// scryptToggleValues are the salt encodings, in CyberChef's order.
var scryptToggleValues = []string{"Hex", "Base64", "UTF8", "Latin1"}

// Scrypt derives a key from the input password using the scrypt PBKDF.
type Scrypt struct{}

// Meta returns the operation metadata.
func (Scrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Scrypt",
		Module:      "Crypto",
		Description: "scrypt is a password-based key derivation function (PBKDF) created by Colin Percival. The algorithm was specifically designed to make it costly to perform large-scale custom hardware attacks by requiring large amounts of memory. In 2016, the scrypt algorithm was published by IETF as RFC 7914.<br><br>Enter the password in the input to generate its hash.",
		InfoURL:     "https://wikipedia.org/wiki/Scrypt",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Scrypt) Args() []core.ArgDef {
	maxKeyLen := float64(scryptMaxKeyLen)
	return []core.ArgDef{
		{Name: "Salt", Type: core.ArgToggleString, Value: "", ToggleValues: scryptToggleValues},
		{Name: "Iterations (N)", Type: core.ArgNumber, Integer: true, Value: float64(16384)},
		{Name: "Memory factor (r)", Type: core.ArgNumber, Integer: true, Value: float64(8)},
		{Name: "Parallelization factor (p)", Type: core.ArgNumber, Integer: true, Value: float64(1)},
		{Name: "Key length", Type: core.ArgNumber, Integer: true, Value: float64(64), Max: &maxKeyLen},
	}
}

// scryptValidate reproduces scryptsy's checkAndInit parameter checks so cchef
// surfaces the same "Error: <msg>" strings CyberChef does (it wraps scryptsy's
// already-"Error:"-prefixed message, hence the doubled prefix).
func scryptValidate(n, r, p int) error {
	if n == 0 || n&(n-1) != 0 {
		return errors.New("Error: Error: N must be > 0 and a power of 2") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	// scryptsy compares with JavaScript floating-point division, so match that
	// rather than integer division (which would round the bounds differently).
	if float64(n) > scryptMaxValue/128.0/float64(r) {
		return errors.New("Error: Error: Parameter N is too large") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if float64(r) > scryptMaxValue/128.0/float64(p) {
		return errors.New("Error: Error: Parameter r is too large") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return nil
}

// Run derives the scrypt key. The parameter validation mirrors scryptsy; the
// derivation itself uses the canonical golang.org/x/crypto/scrypt (an RFC 7914
// implementation), so outputs match byte-for-byte on all standard parameters.
// cchef diverges from scryptsy only at RFC-forbidden degenerate parameters
// (e.g. N=1 or p=0, which scryptsy still computes but the standard scrypt
// rejects).
func (Scrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	salt, err := convertToByteArray(args[0].(core.ToggleString).Value, args[0].(core.ToggleString).Option)
	if err != nil {
		return nil, err
	}
	n := int(args[1].(float64))
	r := int(args[2].(float64))
	p := int(args[3].(float64))
	keyLen := int(args[4].(float64))

	if err := scryptValidate(n, r, p); err != nil {
		return nil, err
	}
	// scryptsy's final PBKDF2 with a zero-length key yields an empty buffer
	// ("" once hex-encoded); the canonical scrypt panics on keyLen 0, so short-
	// circuit to match.
	if keyLen <= 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	data, err := scrypt.Key(in.Bytes(), salt, n, r, p, keyLen)
	if err != nil {
		//nolint:staticcheck,revive // matches CyberChef's `"Error: " + err.toString()` wrapping
		return nil, errors.New("Error: " + err.Error())
	}
	return core.NewDish([]byte(hex.EncodeToString(data)), core.TypeString), nil
}

// scryptMaxKeyLen caps the derived key. N and r are already bounded by the
// parameter check ported from scryptsy; the output length was not, and it sizes
// an allocation directly.
const scryptMaxKeyLen = 4096
