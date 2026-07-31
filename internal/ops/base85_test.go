package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

const b85Std = "!-u"

// Base85 cases. "Man " -> "9jqo^" is the canonical Wikipedia/spec example
// (hand-verified); the "z" all-zero compression is spec-authoritative for the
// Standard alphabet. Round-trips cover the rest.
func TestBase85Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Base85: Man", "Man ", "9jqo^",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, false}}},
		},
		{
			"To Base85: all-zero block -> z", "\x00\x00\x00\x00", "z",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, false}}},
		},
		{
			"To Base85: include delimiter", "Man ", "<~9jqo^~>",
			core.Recipe{{Op: "To Base85", Args: []any{b85Std, true}}},
		},

		{
			"From Base85: Man", "9jqo^", "Man ",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},
		{
			"From Base85: z -> all-zero block", "z", "\x00\x00\x00\x00",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},
		{
			"From Base85: with delimiters", "<~9jqo^~>", "Man ",
			core.Recipe{{Op: "From Base85", Args: []any{b85Std, true, "z"}}},
		},

		{
			"Base85 round trip", "The quick brown fox jumps", "The quick brown fox jumps",
			core.Recipe{
				{Op: "To Base85", Args: []any{b85Std, false}},
				{Op: "From Base85", Args: []any{b85Std, true, "z"}},
			},
		},
	})
}

func TestBase85Errors(t *testing.T) {
	cases := []struct {
		name, op, input string
		args            []any
	}{
		{"To Base85 rejects wrong-length alphabet", "To Base85", "x", []any{"short", false}},
		{"From Base85 rejects wrong-length alphabet", "From Base85", "abc", []any{"short", true, "z"}},
		{"From Base85 rejects all-zero char in alphabet", "From Base85", "abc", []any{base85Standard, true, "!"}},
		{"From Base85 rejects char not in alphabet", "From Base85", "~~~~~", []any{base85Standard, false, "z"}},
	}
	for _, c := range cases {
		if _, err := runOp(t, c.op, c.input, c.args...); err == nil {
			t.Fatalf("%s: expected an error", c.name)
		}
	}
}

// TestFromBase85AllZeroCharIsOneCharacter covers the length of the all-zero
// group character, which stands for a single character and cannot be more.
// CyberChef declares the same limit.
func TestFromBase85AllZeroCharIsOneCharacter(t *testing.T) {
	op, ok := core.Default.Get("From Base85")
	if !ok {
		t.Fatal("From Base85 is not registered")
	}

	if _, err := core.CoerceArgs(op.Args(), []any{base85Standard, true, "z"}); err != nil {
		t.Errorf("one character was rejected: %v", err)
	}
	if _, err := core.CoerceArgs(op.Args(), []any{base85Standard, true, ""}); err != nil {
		t.Errorf("no character at all was rejected: %v", err)
	}

	_, err := core.CoerceArgs(op.Args(), []any{base85Standard, true, "zz"})
	if err == nil {
		t.Fatal("two characters were accepted")
	}
	if want := "All-zero group char length cannot exceed 1."; err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestBase85ValueBranches(t *testing.T) {
	if out, err := runOp(t, "To Base85", "", base85Standard, false); err != nil || out != "" {
		t.Fatalf("To Base85(\"\") = %q, %v; want empty", out, err)
	}
	if out, err := runOp(t, "From Base85", "", base85Standard, false, "z"); err != nil || out != "" {
		t.Fatalf("From Base85(\"\") = %q, %v; want empty", out, err)
	}
	// base85AlphabetName returns "" for an unrecognised alphabet.
	if got := base85AlphabetName("not a known 85-char alphabet"); got != "" {
		t.Fatalf("base85AlphabetName(unknown) = %q, want \"\"", got)
	}
}

// --- direct tests for the helpers extracted from FromBase85.Run ---

// stdB85Idx builds the index map for the standard Ascii85 alphabet ('!'..'u').
func stdB85Idx() map[rune]int {
	idx := map[rune]int{}
	for i, c := range []rune("!\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstu") {
		idx[c] = i
	}
	return idx
}

// TestDecodeBase85Group documents decoding one 5-digit group (and a partial
// group) plus the invalid-character error.
func TestDecodeBase85Group(t *testing.T) {
	idx := stdB85Idx()
	// "9jqo^" is the Ascii85 encoding of "Man " (0x4D616E20).
	got, err := decodeBase85Group([]rune("9jqo^"), 0, idx)
	if err != nil || string(got) != "Man " {
		t.Fatalf("full group: %q, %v", got, err)
	}
	// A partial 2-char group yields 1 byte.
	got, err = decodeBase85Group([]rune("9j"), 0, idx)
	if err != nil || len(got) != 1 {
		t.Fatalf("partial group: %q (%d bytes), %v", got, len(got), err)
	}
	// A character outside the alphabet errors.
	if _, err := decodeBase85Group([]rune("~~~~~"), 0, idx); err == nil {
		t.Fatal("expected invalid-character error")
	}
}

// TestFilterBase85 documents non-alphabet filtering (whitespace dropped, alphabet
// kept).
func TestFilterBase85(t *testing.T) {
	got := filterBase85([]rune("9j qo^"), stdB85Idx(), -1)
	if string(got) != "9jqo^" {
		t.Fatalf("filterBase85 = %q, want %q", string(got), "9jqo^")
	}
}

// TestToBase85PartialZeroGroup covers zero bytes at the end of the input. The
// "z" shorthand stands for a whole four-byte zero group, so a shorter run of
// zeros must be written out in full; using "z" for it loses the length, and
// decoding gives back four zeros however many there were. Expected values are
// the Ascii85 standard, which Python's base64.a85encode also produces.
func TestToBase85PartialZeroGroup(t *testing.T) {
	for _, tc := range []struct {
		zeros int
		want  string
	}{
		{1, "!!"},
		{2, "!!!"},
		{3, "!!!!"},
		{4, "z"},
		{5, "z!!"},
		{6, "z!!!"},
		{7, "z!!!!"},
		{8, "zz"},
	} {
		in := strings.Repeat("\x00", tc.zeros)
		got, err := runOp(t, "To Base85", in, base85Standard, false)
		if err != nil {
			t.Errorf("%d zeros: %v", tc.zeros, err)
			continue
		}
		if got != tc.want {
			t.Errorf("To Base85(%d zero bytes) = %q, want %q", tc.zeros, got, tc.want)
		}
		// And the round trip must return exactly what went in.
		back, err := runOp(t, "From Base85", got, base85Standard, false)
		if err != nil {
			t.Errorf("%d zeros: decoding %q: %v", tc.zeros, got, err)
			continue
		}
		if back != in {
			t.Errorf("%d zero bytes round-tripped to %d", tc.zeros, len(back))
		}
	}
}
