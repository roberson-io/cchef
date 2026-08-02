package ops

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestGenerateHOTPFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/OTP.mjs). The three that assert an
// argument being turned away are checked separately, because cchef reports
// those from the engine rather than the operation.
func TestGenerateHOTPFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Generate HOTP",
			"JBSWY3DPEHPK3PXP",
			"URI: otpauth://hotp/Account?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=6&counter=0" +
				"\n\nPassword: 282760",
			core.Recipe{{Op: "Generate HOTP", Args: []any{"Account", 6, 0}}},
		},
		{
			"special characters in name are URI-encoded",
			"JBSWY3DPEHPK3PXP",
			"URI: otpauth://hotp/user%40example.com?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=6&counter=0" +
				"\n\nPassword: 282760",
			core.Recipe{{Op: "Generate HOTP", Args: []any{"user@example.com", 6, 0}}},
		},
	})
}

// TestGenerateHOTPVectors covers the URI and the password across the whole
// range of the arguments, against the otpauth package CyberChef calls.
func TestGenerateHOTPVectors(t *testing.T) {
	for _, v := range otpVectors(t, "hotp") {
		name := fmt.Sprintf("%s/%s/%d/%v", v.Input, v.Label, v.Digits, v.Counter)
		t.Run(name, func(t *testing.T) {
			secret, err := hex.DecodeString(v.Input)
			if err != nil {
				t.Fatalf("decode secret: %v", err)
			}
			out, err := core.Recipe{{
				Op:   "Generate HOTP",
				Args: []any{v.Label, v.Digits, v.Counter},
			}}.Execute(core.NewDish(secret, core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			want := "URI: " + v.URI + "\n\nPassword: " + v.Want
			if out.String() != want {
				t.Errorf("got  %q\nwant %q", out.String(), want)
			}
		})
	}
}

// TestGenerateHOTPPasswordLength covers the padding: a password shorter than
// the length asked for is filled out with leading zeroes, so every one is the
// same width.
func TestGenerateHOTPPasswordLength(t *testing.T) {
	for digits := otpMinDigits; digits <= otpMaxDigits; digits++ {
		t.Run(strconv.Itoa(digits), func(t *testing.T) {
			for counter := range 200 {
				out, err := runOp(t, "Generate HOTP", "JBSWY3DPEHPK3PXP", "Account", digits, counter)
				if err != nil {
					t.Fatalf("counter %d: %v", counter, err)
				}
				password, ok := strings.CutPrefix(out[strings.LastIndex(out, "\n")+1:], "Password: ")
				if !ok {
					t.Fatalf("counter %d gave no password: %q", counter, out)
				}
				if len(password) != digits {
					t.Fatalf("counter %d gave %q, which is %d digits not %d",
						counter, password, len(password), digits)
				}
				if _, err := strconv.ParseUint(password, 10, 64); err != nil {
					t.Fatalf("counter %d gave %q, which is not all digits", counter, password)
				}
			}
		})
	}
}

// TestGenerateHOTPArgumentsRefused covers the bounds CyberChef declares on the
// arguments, which its fixtures assert through the engine: the name may not be
// empty, the code length is a whole number between six and eight, and the
// counter does not go below zero.
func TestGenerateHOTPArgumentsRefused(t *testing.T) {
	op, ok := core.Default.Get("Generate HOTP")
	if !ok {
		t.Fatal("Generate HOTP is not registered")
	}
	for _, tc := range []struct {
		name string
		args []any
	}{
		{"an empty name", []any{"", 6, 0}},
		{"a code length below the minimum", []any{"Account", -6, 0}},
		{"a code length above the maximum", []any{"Account", 9, 0}},
		{"a code length that is not whole", []any{"Account", 6.5, 0}},
		{"a counter below zero", []any{"Account", 6, -1}},
		{"a counter that is not whole", []any{"Account", 6, 1.5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := core.CoerceArgs(op.Args(), tc.args); err == nil {
				t.Errorf("accepted %v", tc.args)
			}
		})
	}
}

// TestGenerateHOTPSecretRefused covers input that is not base32, which the
// operation reports in its own words rather than the package's.
func TestGenerateHOTPSecretRefused(t *testing.T) {
	const want = "Invalid secret. The input must be a valid base32 string (characters A–Z and 2–7)."

	_, err := runOp(t, "Generate HOTP", "not,valid|base32;input", "Account", 6, 0)
	if err == nil {
		t.Fatal("accepted a secret that is not base32")
	}
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

// TestGenerateHOTPRandomSecret covers the empty input, which stands for a
// secret drawn at random and put into the URI so it can be kept.
func TestGenerateHOTPRandomSecret(t *testing.T) {
	first, err := runOp(t, "Generate HOTP", "", "Account", 6, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	second, err := runOp(t, "Generate HOTP", "", "Account", 6, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if first == second {
		t.Error("two calls with no secret gave the same one")
	}
}
