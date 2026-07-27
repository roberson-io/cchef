package ops

import (
	"math"

	"github.com/roberson-io/cchef/internal/core"
)

// otpDefaultPeriod is how long a time-based password stands for, in seconds.
const otpDefaultPeriod = 30

// GenerateTOTP works out a time-based one-time password.
type GenerateTOTP struct{}

// Meta returns the operation metadata.
func (GenerateTOTP) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Generate TOTP",
		Module: "Default",
		Description: "The Time-based One-Time Password algorithm (TOTP) works a " +
			"password out from a shared secret and the current time. It is the " +
			"counter-based algorithm of RFC 4226 with the counter taken from the " +
			"clock, and was adopted as RFC 6238.<br><br>Give the secret as the " +
			"input, or leave it empty for one to be drawn at random. The secret " +
			"must be a valid base32 string (characters A–Z and 2–7).<br><br>The " +
			"epoch offset (T0) is accepted but has no effect, which is what " +
			"CyberChef does: it hands the offset to a version of its library that " +
			"has no such setting, and the setting is quietly dropped.",
		InfoURL:    "https://wikipedia.org/wiki/Time-based_One-time_Password_algorithm",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateTOTP) Args() []core.ArgDef {
	minDigits, maxDigits := float64(otpMinDigits), float64(otpMaxDigits)
	minOffset, minPeriod := 0.0, 1.0
	return []core.ArgDef{
		{Name: "Name", Type: core.ArgString, Value: "Account", NonEmpty: true},
		{
			Name: "Code length", Type: core.ArgNumber, Value: float64(otpMinDigits),
			Min: &minDigits, Max: &maxDigits, Integer: true,
		},
		{
			Name: "Epoch offset (T0)", Type: core.ArgNumber, Value: 0.0,
			Min: &minOffset, Integer: true,
		},
		{
			Name: "Interval (T1)", Type: core.ArgNumber, Value: float64(otpDefaultPeriod),
			Min: &minPeriod, Integer: true,
		},
	}
}

// Run generates the password.
func (GenerateTOTP) Run(in *core.Dish, args []any) (*core.Dish, error) {
	label, _ := args[0].(string)
	digits, _ := args[1].(float64)
	period, _ := args[3].(float64)

	secret, err := otpReadSecret(in.Bytes())
	if err != nil {
		return nil, err
	}

	// The epoch offset at args[2] is deliberately left unread: see Meta.
	counter := math.Floor(float64(otpNow()) / 1000 / period)

	uri := otpURI("totp", label, secret, int(digits), "period", period)
	return core.NewDish(
		otpOutput(uri, otpPassword(secret, counter, int(digits))),
		core.TypeString,
	), nil
}

func init() { core.Register(GenerateTOTP{}) }
