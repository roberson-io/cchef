package ops

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Argon2{})
	core.Register(Argon2Compare{})
}

// argon2PHCName maps the operation's type option to the PHC identifier used in
// the encoded hash.
var argon2PHCName = map[string]string{
	"Argon2i":  "argon2i",
	"Argon2d":  "argon2d",
	"Argon2id": "argon2id",
}

// Argon2 derives a hash from a password with the Argon2 KDF. A faithful port of
// CyberChef's Argon2 (backed by the argon2-browser WASM): Argon2i and Argon2id
// use golang.org/x/crypto/argon2; Argon2d uses the from-scratch implementation
// in argon2core.go. Output can be the PHC-encoded hash, hex or raw bytes.
type Argon2 struct{}

// Meta returns the operation metadata.
func (Argon2) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Argon2",
		Module:      "Crypto",
		Description: "Argon2 is a key derivation function that was selected as the winner of the Password Hashing Competition in July 2015. It was designed by Alex Biryukov, Daniel Dinu, and Dmitry Khovratovich from the University of Luxembourg. Enter the password in the input to generate its hash.",
		InfoURL:     "https://wikipedia.org/wiki/Argon2",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Argon2) Args() []core.ArgDef {
	maxIter, maxMem := float64(argon2MaxIterations), float64(argon2MaxMemory)
	maxLanes, maxLen := float64(argon2MaxParallelism), float64(argon2MaxHashLen)
	return []core.ArgDef{
		{Name: "Salt", Type: core.ArgToggleString, Value: "somesalt", ToggleValues: []string{"UTF8", "Hex", "Base64", "Latin1"}},
		{Name: "Iterations", Type: core.ArgNumber, Integer: true, Value: 3, Max: &maxIter},
		{Name: "Memory (KiB)", Type: core.ArgNumber, Integer: true, Value: 4096, Max: &maxMem},
		{Name: "Parallelism", Type: core.ArgNumber, Integer: true, Value: 1, Max: &maxLanes},
		{Name: "Hash length (bytes)", Type: core.ArgNumber, Integer: true, Value: 32, Max: &maxLen},
		{Name: "Type", Type: core.ArgOption, Value: []string{"Argon2i", "Argon2d", "Argon2id"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Encoded hash", "Hex hash", "Raw hash"}},
	}
}

// Upper limits on the parameters that size the work. CyberChef leaves all four
// open. The lane count is the one the algorithm forces: the backend takes it as
// a uint8, so a larger value used to wrap around and hash at a different cost
// than the one reported. The rest are set far above any published
// recommendation — RFC 9106 suggests 1 to 3 iterations, and libsodium's most
// expensive preset asks for 1 GiB.
const (
	argon2MaxIterations  = 4096
	argon2MaxMemory      = 2 * 1024 * 1024 // KiB, so 2 GiB
	argon2MaxParallelism = 255
	argon2MaxHashLen     = 4096
)

// Run derives the Argon2 hash in the requested output format.
func (Argon2) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ts := args[0].(core.ToggleString)
	salt, err := convertToByteArray(ts.Value, ts.Option)
	if err != nil {
		return nil, err
	}
	time := int(args[1].(float64))
	memory := int(args[2].(float64))
	parallelism := int(args[3].(float64))
	hashLen := int(args[4].(float64))
	typ := args[5].(string)
	format := args[6].(string)

	if err := argon2Validate(hashLen, len(salt), memory, parallelism, time); err != nil {
		return nil, err
	}

	// #nosec G115 -- parameters are validated positive and within Argon2's small bounds
	raw := argon2Compute(typ, []byte(in.String()), salt, uint32(time), uint32(memory), uint32(parallelism), uint32(hashLen))

	var out string
	switch format {
	case "Hex hash":
		out = hex.EncodeToString(raw)
	case "Raw hash":
		out = byteArrayToUtf8(raw)
	default: // Encoded hash
		out = argon2Encode(typ, memory, time, parallelism, salt, raw)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// argon2Compute produces the raw Argon2 tag for the given type, using x/crypto
// for Argon2i/Argon2id and the from-scratch core for Argon2d.
func argon2Compute(typ string, password, salt []byte, time, memory, parallelism, hashLen uint32) []byte {
	switch typ {
	case "Argon2i":
		return argon2.Key(password, salt, time, memory, uint8(parallelism), hashLen) // #nosec G115 -- the argument declares a maximum of argon2MaxParallelism (255)
	case "Argon2id":
		return argon2.IDKey(password, salt, time, memory, uint8(parallelism), hashLen) // #nosec G115 -- the argument declares a maximum of argon2MaxParallelism (255)
	default: // Argon2d
		return argon2dRaw(password, salt, time, memory, parallelism, hashLen)
	}
}

// argon2Validate reproduces the reference argon2 library's parameter checks and
// their order (output length, salt, memory, time, parallelism), returning the
// same "Error: <message>" text CyberChef surfaces from the WASM.
func argon2Validate(hashLen, saltLen, memory, parallelism, time int) error {
	switch {
	case hashLen < 4:
		return errors.New("Error: Output is too short") //nolint:staticcheck,revive // verbatim CyberChef text
	case saltLen < 8:
		return errors.New("Error: Salt is too short") //nolint:staticcheck,revive // verbatim CyberChef text
	case memory < 8*parallelism:
		return errors.New("Error: Memory cost is too small") //nolint:staticcheck,revive // verbatim CyberChef text
	case time < 1:
		return errors.New("Error: Time cost is too small") //nolint:staticcheck,revive // verbatim CyberChef text
	case parallelism < 1:
		return errors.New("Error: Too few lanes") //nolint:staticcheck,revive // verbatim CyberChef text
	}
	return nil
}

// argon2Encode builds the PHC-format encoded hash
// ($argon2<type>$v=19$m=,t=,p=$<b64salt>$<b64hash>), base64 without padding.
func argon2Encode(typ string, memory, time, parallelism int, salt, hash []byte) string {
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2PHCName[typ], argon2Version, memory, time, parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(hash))
}

// Argon2Compare tests whether the input password matches a given Argon2 encoded
// hash. Ported from CyberChef's Argon2 compare.
type Argon2Compare struct{}

// Meta returns the operation metadata.
func (Argon2Compare) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Argon2 compare",
		Module:      "Crypto",
		Description: "Tests whether the input matches the given Argon2 hash. To test multiple possible passwords, use the 'Fork' operation.",
		InfoURL:     "https://wikipedia.org/wiki/Argon2",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Argon2Compare) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Encoded hash", Type: core.ArgString, Value: ""}}
}

// Run reports whether the input password matches the encoded hash.
func (Argon2Compare) Run(in *core.Dish, args []any) (*core.Dish, error) {
	if argon2Verify(in.String(), args[0].(string)) {
		return core.NewDish([]byte("Match: "+in.String()), core.TypeString), nil
	}
	return core.NewDish([]byte("No match"), core.TypeString), nil
}

// argon2Verify parses a PHC-encoded Argon2 hash, recomputes it for the given
// password, and reports whether it matches. A malformed hash returns false.
func argon2Verify(password, encoded string) bool {
	typ, memory, time, parallelism, salt, hash, ok := argon2ParsePHC(encoded)
	if !ok {
		return false
	}
	raw := argon2Compute(typ, []byte(password), salt, time, memory, parallelism, uint32(len(hash))) // #nosec G115 -- hash length fits a uint32
	return subtle.ConstantTimeCompare(raw, hash) == 1
}

// argon2ParsePHC parses a PHC-format Argon2 string into its type, parameters,
// salt and hash. ok is false for any malformed field.
func argon2ParsePHC(encoded string) (typ string, memory, time, parallelism uint32, salt, hash []byte, ok bool) {
	parts := strings.Split(encoded, "$")
	// "", <type>, v=<ver>, m=,t=,p=, <salt>, <hash>
	if len(parts) != 6 || parts[0] != "" {
		return
	}
	typ, ok = map[string]string{"argon2i": "Argon2i", "argon2d": "Argon2d", "argon2id": "Argon2id"}[parts[1]]
	if !ok {
		return "", 0, 0, 0, nil, nil, false
	}
	if parts[2] != fmt.Sprintf("v=%d", argon2Version) {
		return "", 0, 0, 0, nil, nil, false
	}
	var m, t, p int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return "", 0, 0, 0, nil, nil, false
	}
	b64 := base64.RawStdEncoding
	salt, errS := b64.DecodeString(parts[4])
	hash, errH := b64.DecodeString(parts[5])
	if errS != nil || errH != nil {
		return "", 0, 0, 0, nil, nil, false
	}
	return typ, uint32(m), uint32(t), uint32(p), salt, hash, true // #nosec G115 -- parsed PHC parameters
}
