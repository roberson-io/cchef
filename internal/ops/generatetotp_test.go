package ops

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// atMoment runs a body with the clock the one-time password operations read
// held at a fixed moment.
func atMoment(t *testing.T, milliseconds int64, body func()) {
	t.Helper()
	original := otpNow
	otpNow = func() int64 { return milliseconds }
	defer func() { otpNow = original }()
	body()
}

// TestGenerateTOTPFixture covers CyberChef's own case
// (../CyberChef/tests/operations/tests/OTP.mjs), which can only assert the
// shape of the output because the password moves with the clock. Holding the
// clock still pins the password too.
func TestGenerateTOTPFixture(t *testing.T) {
	shape := regexp.MustCompile(
		`^URI: otpauth://totp/Account\?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=6&period=30` +
			`\n\nPassword: \d{6}$`)

	out, err := runOp(t, "Generate TOTP", "JBSWY3DPEHPK3PXP", "Account", 6, 0, 30)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !shape.MatchString(out) {
		t.Errorf("got %q", out)
	}
}

// TestGenerateTOTPVectors covers the URI and the password at fixed moments,
// against the otpauth package CyberChef calls.
func TestGenerateTOTPVectors(t *testing.T) {
	for _, v := range otpVectors(t, "totp") {
		name := fmt.Sprintf("%s/%d/%d/%d", v.Input, v.Digits, v.Period, v.Timestamp)
		t.Run(name, func(t *testing.T) {
			secret, err := hex.DecodeString(v.Input)
			if err != nil {
				t.Fatalf("decode secret: %v", err)
			}
			atMoment(t, v.Timestamp, func() {
				out, err := core.Recipe{{
					Op:   "Generate TOTP",
					Args: []any{v.Label, v.Digits, 0, v.Period},
				}}.Execute(core.NewDish(secret, core.TypeArrayBuffer))
				if err != nil {
					t.Fatalf("Execute: %v", err)
				}
				want := "URI: " + v.URI + "\n\nPassword: " + v.Want
				if out.String() != want {
					t.Errorf("got  %q\nwant %q", out.String(), want)
				}
			})
		})
	}
}

// TestGenerateTOTPEpochOffsetIsIgnored covers the "Epoch offset (T0)" argument,
// which does nothing. CyberChef passes it to otpauth as an `epoch` option, and
// the version it depends on has no such option: the constructor keeps only the
// issuer, label, algorithm, digits, period, secret and hmac, and quietly drops
// the rest. The argument is still checked for being a whole number that is not
// negative, so it is not simply decoration, but no value of it changes the
// answer. cchef reproduces that.
func TestGenerateTOTPEpochOffsetIsIgnored(t *testing.T) {
	atMoment(t, 1774514156502, func() {
		var first string
		for _, offset := range []any{0, 1, 30, 59, 1000, 1774514156} {
			out, err := runOp(t, "Generate TOTP", "JBSWY3DPEHPK3PXP", "Account", 6, offset, 30)
			if err != nil {
				t.Fatalf("offset %v: %v", offset, err)
			}
			if first == "" {
				first = out
				continue
			}
			if out != first {
				t.Errorf("offset %v changed the answer:\n got %q\nwant %q", offset, out, first)
			}
		}
	})
}

// TestGenerateTOTPPasswordFollowsTheClock covers the period: the password holds
// steady through one interval and changes at the boundary.
func TestGenerateTOTPPasswordFollowsTheClock(t *testing.T) {
	const period = 30

	password := func(milliseconds int64) string {
		var out string
		atMoment(t, milliseconds, func() {
			var err error
			out, err = runOp(t, "Generate TOTP", "JBSWY3DPEHPK3PXP", "Account", 6, 0, period)
			if err != nil {
				t.Fatalf("at %d: %v", milliseconds, err)
			}
		})
		return out
	}

	start := int64(1774514160000) // the beginning of an interval
	if password(start) != password(start+period*1000-1) {
		t.Error("the password changed within one interval")
	}
	if password(start) == password(start+period*1000) {
		t.Error("the password held over the boundary into the next interval")
	}
}

// TestGenerateTOTPArgumentsRefused covers the bounds CyberChef declares on the
// arguments, which its fixtures assert through the engine.
func TestGenerateTOTPArgumentsRefused(t *testing.T) {
	op, ok := core.Default.Get("Generate TOTP")
	if !ok {
		t.Fatal("Generate TOTP is not registered")
	}
	for _, tc := range []struct {
		name string
		args []any
	}{
		{"an empty name", []any{"", 6, 0, 30}},
		{"a code length below the minimum", []any{"Account", -6, 0, 30}},
		{"a code length above the maximum", []any{"Account", 9, 0, 30}},
		{"a code length that is not whole", []any{"Account", 6.5, 0, 30}},
		{"an interval below one", []any{"Account", 6, 0, -1}},
		{"an interval of zero", []any{"Account", 6, 0, 0}},
		{"an interval that is not whole", []any{"Account", 6, 0, 30.5}},
		{"an epoch offset below zero", []any{"Account", 6, -1, 30}},
		{"an epoch offset that is not whole", []any{"Account", 6, 0.5, 30}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := core.CoerceArgs(op.Args(), tc.args); err == nil {
				t.Errorf("accepted %v", tc.args)
			}
		})
	}
}

// TestGenerateTOTPSecretRefused covers input that is not base32.
func TestGenerateTOTPSecretRefused(t *testing.T) {
	const want = "Invalid secret. The input must be a valid base32 string (characters A–Z and 2–7)."

	_, err := runOp(t, "Generate TOTP", "not,valid|base32;input", "Account", 6, 0, 30)
	if err == nil {
		t.Fatal("accepted a secret that is not base32")
	}
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

// TestGenerateTOTPRandomSecret covers the empty input, which stands for a
// secret drawn at random.
func TestGenerateTOTPRandomSecret(t *testing.T) {
	first, err := runOp(t, "Generate TOTP", "", "Account", 6, 0, 30)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	second, err := runOp(t, "Generate TOTP", "", "Account", 6, 0, 30)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if first == second {
		t.Error("two calls with no secret gave the same one")
	}
}
