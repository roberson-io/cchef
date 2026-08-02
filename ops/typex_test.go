package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// typexArgs builds the 24-arg list for a Typex step from the shared fixture
// rotor/reflector/plugboard settings, varying only the keyboard mode and strict
// flag. The five rotors, reflectors, ring/initial settings and reversals match
// CyberChef's tests/operations/tests/Typex.mjs.
func typexArgs(keyboard string, strict bool) []any {
	return []any{
		"MCYLPQUVRXGSAOWNBJEZDTFKHI<BFHNQUW", false, "B", "C",
		"KHWENRCBISXJQGOFMAPVYZDLTU<BFHNQUW", false, "D", "E",
		"BYPDZMGIKQCUSATREHOJNLFWXV<BFHNQUW", false, "F", "G",
		"ZANJCGDLVHIXOBRPMSWQUKFYET<BFHNQUW", true, "H", "I",
		"QXBGUTOVFCZPJIHSWERYNDAMLK<BFHNQUW", true, "J", "K",
		"AN BC FG IE KD LU MH OR TS VZ WQ XJ YP",
		"EHZTLCVKFRPQSYANBUIWOJXGMD",
		keyboard, strict,
	}
}

// TestTypex covers the operation against CyberChef's own fixtures
// (CyberChef's tests/operations/tests/Typex.mjs).
func TestTypex(t *testing.T) {
	msg := "hello world, this is a test message."
	runCases(t, []opCase{
		{
			"basic", msg, "VIXQQ VHLPN UCVLA QDZNZ EAYAT HWC",
			core.Recipe{{Op: "Typex", Args: typexArgs("None", true)}},
		},
		{
			"keyboard", msg, "VIXQQ FDJXT WKLDQ DFQOD CNCSK NULBG JKQDD MVGQ",
			core.Recipe{{Op: "Typex", Args: typexArgs("Encrypt", true)}},
		},
		{
			"self-decrypt", msg, "HELLO WORLD, THIS IS A TEST MESSAGE.",
			core.Recipe{
				{Op: "Typex", Args: typexArgs("Encrypt", true)},
				{Op: "Typex", Args: typexArgs("Decrypt", true)},
			},
		},
		{
			// Non-strict output passes non-alphabet characters through unchanged.
			"non-strict passthrough", "AB.CD-EF", "YR.RK-UX",
			core.Recipe{{Op: "Typex", Args: func() []any { a := typexArgs("None", false); return a }()}},
		},
		{
			// An empty plugboard defaults to the identity map.
			"empty plugboard", msg, "UMRCL TGAXX XGLPX LOLOO RWODU XIO",
			core.Recipe{{Op: "Typex", Args: func() []any {
				a := typexArgs("None", true)
				a[21] = ""
				return a
			}()}},
		},
		{
			// A rotor spec with no "<" stepping list is valid (never steps).
			"rotor without stepping", "HELLO", "VIXQQ",
			core.Recipe{{Op: "Typex", Args: func() []any {
				a := typexArgs("None", true)
				a[0] = "MCYLPQUVRXGSAOWNBJEZDTFKHI"
				return a
			}()}},
		},
	})
}

// TestTypexKeyboard directly exercises the keyboard encode/decode helpers,
// including the shift-mode transitions and the "undefined" branch CyberChef
// produces for a letter with no keyboard symbol.
func TestTypexKeyboard(t *testing.T) {
	if got := typexKeyboardEncode("A1 2B"); got != "AZQXWVB" {
		t.Errorf("encode: got %q, want %q", got, "AZQXWVB")
	}
	if got := typexKeyboardDecode("AZQXWVBZJ"); got != "A1 2Bundefined" {
		t.Errorf("decode: got %q, want %q", got, "A1 2Bundefined")
	}
}

// TestTypexErrors covers the validation paths.
func TestTypexErrors(t *testing.T) {
	cases := []struct {
		name string
		mut  func([]any) []any
		want string
	}{
		{
			"empty rotor", func(a []any) []any { a[0] = ""; return a },
			"Rotor undefined must be provided.",
		},
		{
			"bad rotor wiring", func(a []any) []any { a[0] = "ABCDEF"; return a },
			"Rotor wiring must be 26 unique uppercase letters",
		},
		{
			"bad reflector", func(a []any) []any { a[20] = "AN BC"; return a },
			"Reflector must have exactly 13 pairs covering every letter",
		},
		{
			"bad plugboard", func(a []any) []any { a[21] = "ABC"; return a },
			"Plugboard wiring must be 26 unique uppercase letters",
		},
		{
			"plugboard duplicate letter", func(a []any) []any { a[21] = "AACDEFGHIJKLMNOPQRSTUVWXYZ"; return a },
			"Plugboard wiring must have each letter exactly once",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := c.mut(typexArgs("None", true))
			_, err := core.Recipe{{Op: "Typex", Args: args}}.Execute(core.NewDish([]byte("hello"), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("got %v, want error containing %q", err, c.want)
			}
		})
	}
}
