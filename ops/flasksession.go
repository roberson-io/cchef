package ops

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- SHA1 is a user-selectable Flask/itsdangerous algorithm, matching CyberChef
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"strings"
	"time"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(FlaskSessionDecode{})
	core.Register(FlaskSessionSign{})
	core.Register(FlaskSessionVerify{})
}

// flaskAlgos is the signing-algorithm option list.
var flaskAlgos = []string{"sha1", "sha256"}

// flaskHashers maps each algorithm option to its Go hash constructor.
var flaskHashers = map[string]func() hash.Hash{
	"sha1":   sha1.New,
	"sha256": sha256.New,
}

// flaskKeyToggles / flaskSaltToggles list the toggle-string encodings in
// CyberChef's order (the salt defaults to UTF8).
var (
	flaskKeyToggles  = []string{"Hex", "Decimal", "Binary", "Base64", "UTF8", "Latin1"}
	flaskSaltToggles = []string{"UTF8", "Hex", "Decimal", "Binary", "Base64", "Latin1"}
)

// flaskDefaultSalt is itsdangerous's Flask session salt.
const flaskDefaultSalt = "cookie-session"

// Verbatim error messages from the CyberChef operations.
var (
	//nolint:staticcheck,revive // verbatim CyberChef message
	errFlaskFormat = errors.New("Invalid Flask token format. Expected payload.timestamp.signature")
	//nolint:staticcheck,revive // verbatim CyberChef message
	errFlaskKey = errors.New("Secret key required")
	//nolint:staticcheck,revive // verbatim CyberChef message
	errFlaskSignature = errors.New("Invalid signature!")
	//nolint:staticcheck,revive // verbatim CyberChef message
	errFlaskBase64 = errors.New("Invalid Base64 payload")
)

// flaskB64Encode is itsdangerous's URL-safe base64 with no padding.
func flaskB64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// flaskB64Decode decodes a URL-safe (or standard) base64 segment, padding it to
// a multiple of four as CyberChef does before calling fromBase64.
func flaskB64Decode(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return base64.StdEncoding.DecodeString(s)
}

// flaskTimestamp reads the 4-byte big-endian timestamp from a token's time
// segment (best-effort, matching CyberChef which reads the first four bytes).
func flaskTimestamp(seg string) int64 {
	b, _ := flaskB64Decode(seg)
	var buf [4]byte
	copy(buf[:], b) // zero-padded when fewer than four bytes decode
	// #nosec G115 -- reinterpret the 4 bytes as a signed int32, matching DataView.getInt32
	return int64(int32(binary.BigEndian.Uint32(buf[:])))
}

// flaskHMAC computes HMAC(key, data) with the given hash.
func flaskHMAC(newHash func() hash.Hash, key, data []byte) []byte {
	m := hmac.New(newHash, key)
	m.Write(data)
	return m.Sum(nil)
}

// flaskSignToken builds a signed Flask session cookie. The itsdangerous scheme
// derives a per-salt key as HMAC(secret, salt), then signs "payload.time" with
// HMAC(derivedKey, ...). Split out so the byte layout can be tested with a fixed
// timestamp; the public op supplies the current time.
func flaskSignToken(newHash func() hash.Hash, key, salt []byte, payloadVal any, timestamp int32) string {
	payload := flaskB64Encode([]byte(jsonval.Stringify(payloadVal, 0)))
	var tb [4]byte
	binary.BigEndian.PutUint32(tb[:], uint32(timestamp)) // #nosec G115 -- int32 -> uint32 bit reinterpretation is intended
	timeSeg := flaskB64Encode(tb[:])

	derived := flaskHMAC(newHash, key, salt)
	sig := flaskB64Encode(flaskHMAC(newHash, derived, []byte(payload+"."+timeSeg)))
	return payload + "." + timeSeg + "." + sig
}

// flaskDecodePayload decodes and JSON-parses a token's payload segment.
func flaskDecodePayload(seg string) (any, error) {
	raw, err := flaskB64Decode(seg)
	if err != nil {
		return nil, errFlaskBase64
	}
	val, err := jsonval.ParseOrdered(raw)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode JSON payload: %w", err) //nolint:staticcheck,revive // verbatim CyberChef message prefix
	}
	return val, nil
}

// flaskParseKeySalt decodes the key (required) and salt arguments.
func flaskParseKeySalt(keyArg, saltArg core.ToggleString) (key, salt []byte, err error) {
	if keyArg.Value == "" {
		return nil, nil, errFlaskKey
	}
	if key, err = convertToByteArray(keyArg.Value, keyArg.Option); err != nil {
		return nil, nil, err
	}
	saltStr := saltArg.Value
	if saltStr == "" {
		saltStr = flaskDefaultSalt
	}
	if salt, err = convertToByteArray(saltStr, saltArg.Option); err != nil {
		return nil, nil, err
	}
	return key, salt, nil
}

// FlaskSessionDecode decodes a Flask session cookie payload into JSON.
type FlaskSessionDecode struct{}

// Meta returns the operation metadata.
func (FlaskSessionDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Flask Session Decode",
		Module:      "Crypto",
		Description: "Decodes the payload of a Flask session cookie (itsdangerous) into JSON.",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (FlaskSessionDecode) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "View TimeStamp", Type: core.ArgBoolean, Value: false}}
}

// Run decodes the token. Ported from CyberChef FlaskSessionDecode.mjs.
func (FlaskSessionDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	parts := strings.Split(strings.TrimSpace(in.String()), ".")
	if len(parts) != 3 {
		return nil, errFlaskFormat
	}
	val, err := flaskDecodePayload(parts[0])
	if err != nil {
		return nil, err
	}
	out := val
	if args[0].(bool) {
		out = jsonval.Object{{K: "payload", V: val}, {K: "timestamp", V: flaskTimestamp(parts[1])}}
	}
	return core.NewDish([]byte(jsonval.Stringify(out, 4)), core.TypeJSON), nil
}

// FlaskSessionSign signs a JSON payload into a Flask session cookie.
type FlaskSessionSign struct{}

// Meta returns the operation metadata.
func (FlaskSessionSign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Flask Session Sign",
		Module:      "Crypto",
		Description: "Signs a JSON payload to produce a Flask session cookie (itsdangerous HMAC).",
		InputType:   core.TypeJSON,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FlaskSessionSign) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: flaskKeyToggles},
		{Name: "Salt", Type: core.ArgToggleString, Value: flaskDefaultSalt, ToggleValues: flaskSaltToggles},
		{Name: "Algorithm", Type: core.ArgOption, Value: flaskAlgos},
	}
}

// Run signs the payload. Ported from CyberChef FlaskSessionSign.mjs.
func (FlaskSessionSign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, salt, err := flaskParseKeySalt(args[0].(core.ToggleString), args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	payload, err := jsonval.ParseOrdered(in.Bytes())
	if err != nil {
		return nil, err
	}
	// Math.ceil(Date.now() / 1000).
	ts := int32((time.Now().UnixMilli() + 999) / 1000) // #nosec G115 -- matches JS setInt32 wraparound
	token := flaskSignToken(flaskHashers[args[2].(string)], key, salt, payload, ts)
	return core.NewDish([]byte(token), core.TypeString), nil
}

// FlaskSessionVerify verifies a Flask session cookie's HMAC signature.
type FlaskSessionVerify struct{}

// Meta returns the operation metadata.
func (FlaskSessionVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Flask Session Verify",
		Module:      "Crypto",
		Description: "Verifies the HMAC signature of a Flask session cookie (itsdangerous) generated.",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (FlaskSessionVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: flaskKeyToggles},
		{Name: "Salt", Type: core.ArgToggleString, Value: flaskDefaultSalt, ToggleValues: flaskSaltToggles},
		{Name: "Algorithm", Type: core.ArgOption, Value: flaskAlgos},
		{Name: "View TimeStamp", Type: core.ArgBoolean, Value: true},
	}
}

// Run verifies the token. Ported from CyberChef FlaskSessionVerify.mjs.
func (FlaskSessionVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, salt, err := flaskParseKeySalt(args[0].(core.ToggleString), args[1].(core.ToggleString))
	if err != nil {
		return nil, err
	}
	newHash := flaskHashers[args[2].(string)]

	parts := strings.Split(strings.TrimSpace(in.String()), ".")
	if len(parts) != 3 {
		return nil, errFlaskFormat
	}
	val, err := flaskDecodePayload(parts[0])
	if err != nil {
		return nil, err
	}

	derived := flaskHMAC(newHash, key, salt)
	sig := flaskB64Encode(flaskHMAC(newHash, derived, []byte(parts[0]+"."+parts[1])))
	if sig != parts[2] {
		return nil, errFlaskSignature
	}

	out := jsonval.Object{{K: "valid", V: true}, {K: "payload", V: val}}
	if args[3].(bool) {
		out = append(out, jsonval.Pair{K: "timestamp", V: flaskTimestamp(parts[1])})
	}
	return core.NewDish([]byte(jsonval.Stringify(out, 4)), core.TypeJSON), nil
}
