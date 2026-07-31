package ops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Bcrypt{})
	core.Register(BcryptCompare{})
	core.Register(BcryptParse{})
}

// The three operations wrap the npm `bcryptjs` package in CyberChef. cchef backs
// them with golang.org/x/crypto/bcrypt (already a dependency) plus a thin
// bcryptjs-compatibility layer: bcryptjs clamps the cost to [4,31] and truncates
// the password to 72 bytes (Go rejects both), emits a `$2b$` version prefix
// (Go emits `$2a$`), and produces specific error strings for malformed hashes.

// bcryptGenerate is a seam so tests can exercise the (otherwise unreachable)
// generation-error path.
var bcryptGenerate = bcrypt.GenerateFromPassword

// bcryptTruncate limits the password to 72 bytes, matching bcrypt's key length
// (bcryptjs truncates silently; Go's GenerateFromPassword errors instead).
func bcryptTruncate(pw []byte) []byte {
	if len(pw) > 72 {
		return pw[:72]
	}
	return pw
}

// Bcrypt hashes the input password with a random salt.
type Bcrypt struct{}

// Meta returns the operation metadata.
func (Bcrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bcrypt",
		Module:      "Crypto",
		Description: "bcrypt is a password hashing function designed by Niels Provos and David Mazières, based on the Blowfish cipher, and presented at USENIX in 1999. Besides incorporating a salt to protect against rainbow table attacks, bcrypt is an adaptive function: over time, the iteration count (rounds) can be increased to make it slower, so it remains resistant to brute-force search attacks even with increasing computation power.<br><br>Enter the password in the input to generate its hash.",
		InfoURL:     "https://wikipedia.org/wiki/Bcrypt",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Bcrypt) Args() []core.ArgDef {
	minRounds, maxRounds := float64(bcryptMinRounds), float64(bcryptMaxRounds)
	return []core.ArgDef{
		{Name: "Rounds", Type: core.ArgNumber, Integer: true, Value: 10, Min: &minRounds, Max: &maxRounds},
	}
}

// Run generates a bcrypt hash. Ported from CyberChef Bcrypt.mjs (bcryptjs).
func (Bcrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rounds := int(args[0].(float64))

	hash, err := bcryptGenerate(bcryptTruncate(in.Bytes()), rounds)
	if err != nil {
		return nil, err
	}
	// bcryptjs v3 emits the `$2b$` version; x/crypto emits `$2a$`. For passwords
	// up to 72 bytes the two produce identical bytes, so rewriting the tag yields
	// exactly what bcryptjs would output for the same salt.
	out := strings.Replace(string(hash), "$2a$", "$2b$", 1)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// BcryptCompare tests whether the input matches the given bcrypt hash.
type BcryptCompare struct{}

// Meta returns the operation metadata.
func (BcryptCompare) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bcrypt compare",
		Module:      "Crypto",
		Description: "Tests whether the input matches the given bcrypt hash. To test multiple possible passwords, use the 'Fork' operation.",
		InfoURL:     "https://wikipedia.org/wiki/Bcrypt",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BcryptCompare) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Hash", Type: core.ArgString, Value: ""}}
}

// Run compares the password against the hash. Ported from CyberChef
// BcryptCompare.mjs (bcryptjs).
func (BcryptCompare) Run(in *core.Dish, args []any) (*core.Dish, error) {
	hash := args[0].(string)
	input := in.String()

	// bcryptjs compare returns false (No match) for a wrong-length hash before
	// validating anything else.
	if len(hash) != 60 {
		return core.NewDish([]byte("No match"), core.TypeString), nil
	}
	if err := bcryptValidateSalt(hash); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), bcryptTruncate([]byte(input))); err == nil {
		return core.NewDish([]byte("Match: "+input), core.TypeString), nil
	}
	// Any non-match (mismatch or a malformed-but-well-formed-prefix hash) is "No
	// match" in bcryptjs, which recomputes and compares rather than erroring.
	return core.NewDish([]byte("No match"), core.TypeString), nil
}

// bcryptValidateSalt reproduces bcryptjs's salt-format validation (on a 60-byte
// hash) so cchef surfaces the same error strings. Each message is prefixed with
// "Error: " to match CyberChef's `err.toString()` of the thrown OperationError.
func bcryptValidateSalt(hash string) error {
	if hash[0] != '$' || hash[1] != '2' {
		return errors.New("Error: Invalid salt version: " + hash[0:2]) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	offset := 3
	if hash[2] != '$' {
		if minor := hash[2]; (minor != 'a' && minor != 'b' && minor != 'y') || hash[3] != '$' {
			return errors.New("Error: Invalid salt revision: " + hash[2:4]) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		offset = 4
	}
	if hash[offset+2] > '$' {
		return errors.New("Error: Missing salt rounds") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	// The rounds field is two digits; a numeric value outside [4,31] is rejected
	// by bcryptjs's _crypt. Non-digit fields yield NaN there, which slips past
	// the range check, so only validate when both characters are digits.
	if isDigit(hash[offset]) && isDigit(hash[offset+1]) {
		rounds := int(hash[offset]-'0')*10 + int(hash[offset+1]-'0')
		if rounds < 4 || rounds > 31 {
			return fmt.Errorf("Error: Illegal number of rounds (4-31): %d", rounds) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
	}
	return nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// BcryptParse parses a bcrypt hash into its rounds, salt, and hash components.
type BcryptParse struct{}

// Meta returns the operation metadata.
func (BcryptParse) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bcrypt parse",
		Module:      "Crypto",
		Description: "Parses a bcrypt hash to determine the number of rounds used, the salt, and the password hash.",
		InfoURL:     "https://wikipedia.org/wiki/Bcrypt",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BcryptParse) Args() []core.ArgDef { return nil }

// Run parses the hash. Ported from CyberChef BcryptParse.mjs (bcryptjs
// getRounds/getSalt). The doubled "Error: " prefix matches CyberChef wrapping
// bcryptjs's already-"Error:"-prefixed message.
func (BcryptParse) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	// bcryptjs getSalt requires a 60-character hash.
	if len(input) != 60 {
		return nil, fmt.Errorf("Error: Error: Illegal hash length: %d != 60", len(input)) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	salt := input[:29]
	out := fmt.Sprintf("Rounds: %s\nSalt: %s\nPassword hash: %s\nFull hash: %s",
		bcryptGetRounds(input), salt, input[29:], input)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// bcryptGetRounds mirrors bcryptjs getRounds: parseInt(hash.split("$")[2], 10),
// rendering NaN when the field has no leading integer.
func bcryptGetRounds(hash string) string {
	parts := strings.Split(hash, "$")
	field := ""
	if len(parts) > 2 {
		field = parts[2]
	}
	if n, ok := jsParseInt(field, 10); ok {
		return strconv.Itoa(n)
	}
	return "NaN"
}

// bcrypt's cost range. bcryptjs, and so CyberChef, silently clamps a value
// outside it and hashes at the clamped cost; declaring the range means a
// mistyped cost is refused rather than quietly becoming a different one.
const (
	bcryptMinRounds = 4
	bcryptMaxRounds = 31
)
