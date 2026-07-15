package ops

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// dolphinHash is the bcrypt hash of "dolphin" used in CyberChef's Hash.mjs
// fixtures.
const dolphinHash = "$2a$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK"

// Bcrypt compare success paths ("Match: ..."/"No match" are outputs, not errors).
// The dolphin case is transcribed from ../CyberChef/tests/operations/tests/Hash.mjs;
// the rest are oracle-verified.
func TestBcryptCompareFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Bcrypt compare: dolphin", "dolphin", "Match: dolphin",
			core.Recipe{{Op: "Bcrypt compare", Args: []any{dolphinHash}}},
		},
		{
			"Bcrypt compare: wrong password", "wrongpw", "No match",
			core.Recipe{{Op: "Bcrypt compare", Args: []any{dolphinHash}}},
		},
		{
			// A hash whose length isn't 60 never matches (returns false, not an error).
			"Bcrypt compare: wrong-length hash", "x", "No match",
			core.Recipe{{Op: "Bcrypt compare", Args: []any{"$2a$10$short"}}},
		},
	})
}

// Bcrypt compare error paths. The "invalid salt version" case is transcribed
// from Hash.mjs; the others are derived from the bcryptjs source (the oracle
// returns HTTP 500 on these rather than the message text).
func TestBcryptCompareErrors(t *testing.T) {
	cases := []struct{ name, hash, wantErr string }{
		{"invalid salt version", "$ab$04$K.H1WlFDQ/iIo/PiprT/puwluJ5rzuSE5q8D/Fk3NuLgU2aXiGR9m", "Error: Invalid salt version: $a"},
		{"invalid salt revision", "$2c$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK", "Error: Invalid salt revision: c$"},
		{"illegal rounds high", "$2a$99$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK", "Error: Illegal number of rounds (4-31): 99"},
		{"illegal rounds low", "$2a$03$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK", "Error: Illegal number of rounds (4-31): 3"},
		{"missing salt rounds", "$2a$100yon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK1", "Error: Missing salt rounds"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Bcrypt compare", "password", c.hash)
			if err == nil {
				t.Fatalf("expected error %q, got nil", c.wantErr)
			}
			if err.Error() != c.wantErr {
				t.Fatalf("got error %q\nwant %q", err.Error(), c.wantErr)
			}
		})
	}
}

// Bcrypt parse. No CyberChef fixture file exists; the success cases are
// oracle-verified and the error case matches the oracle's message.
func TestBcryptParseFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Bcrypt parse: dolphin", dolphinHash,
			"Rounds: 10\n" +
				"Salt: $2a$10$qyon0LQCmMxpFFjwWH6Qh.\n" +
				"Password hash: dDdhqntQh./IN0RXCc3XIMILuOYZKgK\n" +
				"Full hash: " + dolphinHash,
			core.Recipe{{Op: "Bcrypt parse", Args: []any{}}},
		},
		{
			// A 60-char non-hash: getRounds yields NaN, salt/hash split at char 29.
			"Bcrypt parse: 60-char non-hash (NaN rounds)",
			"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567",
			"Rounds: NaN\n" +
				"Salt: abcdefghijklmnopqrstuvwxyzABC\n" +
				"Password hash: DEFGHIJKLMNOPQRSTUVWXYZ01234567\n" +
				"Full hash: abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567",
			core.Recipe{{Op: "Bcrypt parse", Args: []any{}}},
		},
	})
}

func TestBcryptParseError(t *testing.T) {
	_, err := runOp(t, "Bcrypt parse", "tooshort")
	if err == nil {
		t.Fatal("expected error for wrong-length hash")
	}
	if want := "Error: Error: Illegal hash length: 8 != 60"; err.Error() != want {
		t.Fatalf("got %q\nwant %q", err.Error(), want)
	}
}

// bcryptHashRE matches a $2b$ hash with a two-digit cost and 53 base64 chars.
var bcryptHashRE = regexp.MustCompile(`^\$2b\$\d{2}\$[./A-Za-z0-9]{53}$`)

// The Bcrypt hash op is non-deterministic (random salt), so verify the output
// format and that it round-trips through Bcrypt compare.
func TestBcryptHash(t *testing.T) {
	out, err := runOp(t, "Bcrypt", "password", 10)
	if err != nil {
		t.Fatalf("Bcrypt: %v", err)
	}
	if !bcryptHashRE.MatchString(out) {
		t.Fatalf("hash %q does not match %v", out, bcryptHashRE)
	}
	// Round-trip: the generated hash must verify the same password.
	cmp, err := runOp(t, "Bcrypt compare", "password", out)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if cmp != "Match: password" {
		t.Fatalf("round-trip got %q, want %q", cmp, "Match: password")
	}
}

// Rounds outside [4, 31] are clamped (matching bcryptjs genSalt), not rejected.
// A real cost of 31 is infeasible to run, so the generation call is stubbed to
// capture the (clamped) cost and confirm the $2b$ prefix rewrite.
func TestBcryptHashRoundsClamp(t *testing.T) {
	orig := bcryptGenerate
	defer func() { bcryptGenerate = orig }()

	var gotCost int
	bcryptGenerate = func(_ []byte, cost int) ([]byte, error) {
		gotCost = cost
		return fmt.Appendf(nil, "$2a$%02d$0123456789012345678901ABCDEFGHIJKLMNOPQRSTUVWXYZ012", cost), nil
	}

	for _, c := range []struct {
		rounds   int
		wantCost int
		prefix   string
	}{
		{3, 4, "$2b$04$"},
		{40, 31, "$2b$31$"},
		{10, 10, "$2b$10$"},
	} {
		out, err := runOp(t, "Bcrypt", "pw", c.rounds)
		if err != nil {
			t.Fatalf("Bcrypt rounds=%d: %v", c.rounds, err)
		}
		if gotCost != c.wantCost {
			t.Fatalf("rounds=%d: clamped cost %d, want %d", c.rounds, gotCost, c.wantCost)
		}
		if out[:7] != c.prefix {
			t.Fatalf("rounds=%d: got prefix %q, want %q", c.rounds, out[:7], c.prefix)
		}
	}
}

// Passwords over 72 bytes are truncated (matching bcryptjs) rather than
// rejected as Go's x/crypto would; a long password still round-trips.
func TestBcryptHashLongPassword(t *testing.T) {
	long := strings.Repeat("a", 100)
	hash, err := runOp(t, "Bcrypt", long, 6)
	if err != nil {
		t.Fatalf("Bcrypt long: %v", err)
	}
	got, err := runOp(t, "Bcrypt compare", long, hash)
	if err != nil {
		t.Fatalf("compare long: %v", err)
	}
	if got != "Match: "+long {
		t.Fatalf("long round-trip got %q", got)
	}
	// A password differing only past byte 72 still matches (truncation).
	got2, err := runOp(t, "Bcrypt compare", long[:72]+"DIFFERENT", hash)
	if err != nil {
		t.Fatalf("compare trunc: %v", err)
	}
	if got2 != "Match: "+long[:72]+"DIFFERENT" {
		t.Fatalf("truncation mismatch: got %q", got2)
	}
}

// A generation error is surfaced (the real path can only fail on RNG failure).
func TestBcryptHashGenerateError(t *testing.T) {
	orig := bcryptGenerate
	defer func() { bcryptGenerate = orig }()
	bcryptGenerate = func(_ []byte, _ int) ([]byte, error) { return nil, errors.New("rng failure") }

	if _, err := runOp(t, "Bcrypt", "pw", 10); err == nil {
		t.Fatal("expected generation error to be surfaced")
	}
}
