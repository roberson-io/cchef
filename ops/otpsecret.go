package ops

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 4226 defines the password in terms of HMAC-SHA1
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/roberson-io/cchef/internal/jsnum"
)

// The machinery the two one-time password operations share, ported from
// CyberChef's node_modules/otpauth/dist/otpauth.node.mjs.

// otpAlphabet is the base32 alphabet of RFC 4648, which secrets are written in.
const otpAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// otpSecretSize is how many bytes are drawn when no secret is given. Twenty is
// what otpauth settles on, and comes out as thirty-two characters.
const otpSecretSize = 20

// The bounds CyberChef puts on the length of a password.
const (
	otpMinDigits = 6
	otpMaxDigits = 8
)

// otpCounterSize is the width of the message a password is worked out over: a
// counter written big-endian across eight bytes.
const otpCounterSize = 8

// errInvalidOTPSecret is what both operations report for input that is not
// base32, in place of the message the package would give naming the character.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errInvalidOTPSecret = errors.New(
	"Invalid secret. The input must be a valid base32 string (characters A–Z and 2–7).")

// otpNow is the moment a time-based password is worked out for. It is a package
// variable so tests can hold the clock still.
var otpNow = func() int64 { return time.Now().UnixMilli() }

// otpRandomBytes fills b with randomness. It is a package variable so tests can
// stand in for the system's source of it.
var otpRandomBytes = func(b []byte) error {
	_, err := rand.Read(b)
	return err
}

// otpReadSecret takes the secret from the input. An input that is nothing but
// space stands for "make me one", which is how both operations offer a secret
// to start from.
func otpReadSecret(in []byte) ([]byte, error) {
	text := jsTrimSpace(string(in))
	if text == "" {
		secret := make([]byte, otpSecretSize)
		if err := otpRandomBytes(secret); err != nil {
			return nil, err
		}
		return secret, nil
	}

	// Case is levelled and spacing dropped before the secret is read, since
	// issuers write secrets in groups for legibility.
	upper := strings.Map(func(r rune) rune {
		if jsnum.IsSpace(r) {
			return -1
		}
		return r
	}, strings.ToUpper(text))

	return otpDecodeBase32(upper)
}

// otpDecodeBase32 reads a base32 secret. Padding is dropped from the end,
// and a group at the end too short to make a whole byte is dropped with it, so
// the secret that comes back may be shorter than the text suggests.
func otpDecodeBase32(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")

	out := make([]byte, len(s)*5/8)
	var bits, value, index int
	for _, c := range s {
		digit := strings.IndexRune(otpAlphabet, c)
		if digit < 0 {
			return nil, errInvalidOTPSecret
		}
		value = value<<5 | digit
		bits += 5
		if bits >= 8 {
			bits -= 8
			if index < len(out) {
				out[index] = byte(value >> bits) // #nosec G115 -- eight bits have accumulated, which is a byte
				index++
			}
		}
	}
	return out, nil
}

// otpBase32 writes a secret out again. This is what goes into the URI, and it
// need not match what was read: padding is gone, case is levelled, and the bits
// of a partial group that carried no byte are gone with them.
func otpBase32(secret []byte) string {
	var out strings.Builder
	var bits, value int
	for _, b := range secret {
		value = value<<8 | int(b)
		bits += 8
		for bits >= 5 {
			out.WriteByte(otpAlphabet[value>>(bits-5)&31])
			bits -= 5
		}
	}
	if bits > 0 {
		out.WriteByte(otpAlphabet[value<<(5-bits)&31])
	}
	return out.String()
}

// otpCounterMessage writes the counter across eight bytes, most significant
// first. It is transliterated from otpauth's uintDecode, which works in floating
// point and so behaves in its own way once the counter runs past what a number
// counts exactly.
func otpCounterMessage(counter float64) []byte {
	out := make([]byte, otpCounterSize)
	remaining := counter
	for i := otpCounterSize - 1; i >= 0; i-- {
		if remaining == 0 {
			break
		}
		out[i] = byte(jsToInt32(remaining) & 255)
		remaining -= float64(out[i])
		remaining /= 256
	}
	return out
}

// otpPassword works the password out from the secret and the counter, by the
// dynamic truncation RFC 4226 describes: four bytes are taken from a place the
// digest itself picks, and what is left after the leading bit is cut is reduced
// to the number of digits wanted.
func otpPassword(secret []byte, counter float64, digits int) string {
	mac := hmac.New(sha1.New, secret) // #nosec G401 -- RFC 4226 defines the password this way
	mac.Write(otpCounterMessage(counter))
	digest := mac.Sum(nil)

	offset := digest[len(digest)-1] & 0x0f
	value := int64(digest[offset]&0x7f)<<24 |
		int64(digest[offset+1])<<16 |
		int64(digest[offset+2])<<8 |
		int64(digest[offset+3])

	return leftPad(strconv.FormatInt(value%int64(math.Pow10(digits)), 10), digits)
}

// otpURI writes the key URI a password app reads to set an account up. An
// issuer is never given, so the label stands alone.
func otpURI(kind, label string, secret []byte, digits int, field string, value float64) string {
	e := jsEncodeURIComponent
	return "otpauth://" + kind + "/" + e(label) +
		"?secret=" + e(otpBase32(secret)) +
		"&algorithm=SHA1" +
		"&digits=" + e(strconv.Itoa(digits)) +
		"&" + field + "=" + e(jsnum.Format(value))
}

// otpOutput is what both operations report: the URI to set the account up with,
// and the password as it stands.
func otpOutput(uri, password string) []byte {
	return []byte("URI: " + uri + "\n\nPassword: " + password)
}
