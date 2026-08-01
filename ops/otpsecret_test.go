package ops

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// otpVector is one recorded call into the otpauth package: a secret read back
// out, a secret refused, or a password for a given counter or moment. The
// input is hexadecimal so any bytes may appear.
type otpVector struct {
	Kind      string  `json:"kind"`
	Input     string  `json:"input"`
	Label     string  `json:"label"`
	Digits    int     `json:"digits"`
	Counter   float64 `json:"counter"`
	Period    int64   `json:"period"`
	Timestamp int64   `json:"timestamp"`
	URI       string  `json:"uri"`
	Want      string  `json:"want"`
}

// otpVectors reads the recorded calls of one kind.
func otpVectors(t *testing.T, kind string) []otpVector {
	t.Helper()
	file, err := os.Open("testdata/otp.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	var out []otpVector
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var v otpVector
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("parse vector: %v", err)
		}
		if v.Kind == kind {
			out = append(out, v)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("no %s vectors were read", kind)
	}
	return out
}

// TestOTPSecretCanonical covers what the secret becomes once it has been read.
// It is decoded and written out again, so the base32 that ends up in the URI is
// not always the base32 that went in: padding goes, case is levelled, spacing
// is dropped, and a partial group at the end loses the bits that do not make up
// a whole byte.
func TestOTPSecretCanonical(t *testing.T) {
	for _, v := range otpVectors(t, "secret") {
		t.Run(v.Input, func(t *testing.T) {
			secret, err := otpReadSecret(mustHex(t, v.Input))
			if err != nil {
				t.Fatalf("read secret: %v", err)
			}
			if got := otpBase32(secret); got != v.Want {
				t.Errorf("got %q, want %q", got, v.Want)
			}
		})
	}
}

// TestOTPSecretRejected covers the input the secret parser will not read. Only
// the base32 alphabet is allowed, and padding counts only at the very end.
func TestOTPSecretRejected(t *testing.T) {
	for _, v := range otpVectors(t, "secretError") {
		t.Run(v.Input, func(t *testing.T) {
			if _, err := otpReadSecret(mustHex(t, v.Input)); err == nil {
				t.Errorf("read %q as a secret", v.Input)
			}
		})
	}
}

// TestOTPSecretNoRandomness covers what happens when the system will not give
// out random bytes and no secret was supplied to fall back on.
func TestOTPSecretNoRandomness(t *testing.T) {
	original := otpRandomBytes
	otpRandomBytes = func([]byte) error { return errors.New("no entropy") }
	defer func() { otpRandomBytes = original }()

	for _, tc := range []struct {
		op   string
		args []any
	}{
		{"Generate HOTP", []any{"Account", 6, 0}},
		{"Generate TOTP", []any{"Account", 6, 0, 30}},
	} {
		t.Run(tc.op, func(t *testing.T) {
			if _, err := runOp(t, tc.op, "", tc.args...); err == nil {
				t.Error("made a secret with no randomness to draw on")
			}
		})
	}
}

// TestOTPSecretDrawnAtRandom covers the empty input, which stands for "make me
// one". The secret is twenty bytes, which is what the base32 length says.
func TestOTPSecretDrawnAtRandom(t *testing.T) {
	const wantLength = 32 // twenty bytes written in base32

	seen := map[string]bool{}
	for _, input := range []string{"", "   ", "\t\n"} {
		for range 10 {
			secret, err := otpReadSecret([]byte(input))
			if err != nil {
				t.Fatalf("read secret from %q: %v", input, err)
			}
			written := otpBase32(secret)
			if len(written) != wantLength {
				t.Fatalf("secret %q is %d characters, want %d", written, len(written), wantLength)
			}
			if strings.Trim(written, "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567") != "" {
				t.Fatalf("secret %q is not base32", written)
			}
			if seen[written] {
				t.Fatalf("secret %q came up twice", written)
			}
			seen[written] = true
		}
	}
}
