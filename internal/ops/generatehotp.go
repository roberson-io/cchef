package ops

import "github.com/roberson-io/cchef/internal/core"

// GenerateHOTP works out a counter-based one-time password.
type GenerateHOTP struct{}

// Meta returns the operation metadata.
func (GenerateHOTP) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Generate HOTP",
		Module: "Default",
		Description: "The HMAC-based One-Time Password algorithm (HOTP) works a " +
			"password out from a shared secret and a counter that climbs with each " +
			"use. It was adopted as RFC 4226, is the cornerstone of the Initiative " +
			"For Open Authentication (OAUTH), and is used by a number of two-factor " +
			"authentication systems.<br><br>Give the secret as the input, or leave " +
			"it empty for one to be drawn at random. The secret must be a valid " +
			"base32 string (characters A–Z and 2–7).",
		InfoURL:    "https://wikipedia.org/wiki/HMAC-based_One-time_Password_algorithm",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateHOTP) Args() []core.ArgDef {
	minDigits, maxDigits, minCounter := float64(otpMinDigits), float64(otpMaxDigits), 0.0
	return []core.ArgDef{
		{Name: "Name", Type: core.ArgString, Value: "Account", NonEmpty: true},
		{
			Name: "Code length", Type: core.ArgNumber, Value: float64(otpMinDigits),
			Min: &minDigits, Max: &maxDigits, Integer: true,
		},
		{Name: "Counter", Type: core.ArgNumber, Value: 0.0, Min: &minCounter, Integer: true},
	}
}

// Run generates the password.
func (GenerateHOTP) Run(in *core.Dish, args []any) (*core.Dish, error) {
	label, _ := args[0].(string)
	digits, _ := args[1].(float64)
	counter, _ := args[2].(float64)

	secret, err := otpReadSecret(in.Bytes())
	if err != nil {
		return nil, err
	}

	uri := otpURI("hotp", label, secret, int(digits), "counter", counter)
	return core.NewDish(
		otpOutput(uri, otpPassword(secret, counter, int(digits))),
		core.TypeString,
	), nil
}

func init() { core.Register(GenerateHOTP{}) }
