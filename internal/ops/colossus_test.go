package ops

// Colossus tests. Ported from CyberChef Colossus.mjs (which wraps
// lib/Colossus.mjs + lib/Lorenz.mjs). CyberChef ships only one active fixture
// (Letter Count); the others there are commented out as too slow. Every case
// below was verified against the CyberChef-server oracle using no-stepping
// configurations (fast/slow step left blank) so each run terminates in a single
// tape pass. Stepping behaviour is covered by the source port itself.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// colArgs returns the 57-slot Colossus argument list with neutral defaults,
// matching the index layout of CyberChef's args array exactly. Label slots
// (0, 8, 31, 41) are empty strings; starts default to 1; Set Total to 0.
func colArgs() []any {
	return []any{
		"",           // 0  Input (label)
		"KH Pattern", // 1  Pattern
		"", "", "",   // 2-4  QBusZ / QBusΧ / QBusΨ
		"None",                        // 5  Limitation
		"Advanced",                    // 6  K Rack Option
		"",                            // 7  Program to run
		"",                            // 8  K Rack: Conditional (label)
		"", "", "", "", "", false, "", // 9-15   R1 Q1-5, Negate, Counter
		"", "", "", "", "", false, "", // 16-22  R2
		"", "", "", "", "", false, "", // 23-29  R3
		false,                             // 30 Negate All
		"",                                // 31 K Rack: Addition (label)
		false, false, false, false, false, // 32-36 Add Q1-5
		"", false, false, "", // 37-40 Add-Equals, Add-Counter1, Add Negate All, Total Motor
		"",     // 41 Master Control Panel (label)
		0,      // 42 Set Total
		"", "", // 43-44 Fast Step / Slow Step
		1, 1, 1, 1, 1, // 45-49 Start Χ1-5
		1, 1, // 50-51 Start M61 / M37
		1, 1, 1, 1, 1, // 52-56 Start Ψ1-5
	}
}

// with clones the base args and applies index→value overrides.
func with(base []any, over map[int]any) []any {
	out := append([]any(nil), base...)
	for i, v := range over {
		out[i] = v
	}
	return out
}

func colRecipe(over map[int]any) core.Recipe {
	return core.Recipe{{Op: "Colossus", Args: with(colArgs(), over)}}
}

func TestColossusFixtures(t *testing.T) {
	const input = "CTBKJUVXHZ-H3L4QV+YEZUK+SXOZ/N"
	runCases(t, []opCase{
		// CyberChef's sole active fixture: Letter Count (expectedMatch /00 00 : a30/).
		{
			"Colossus: Letter Count", input,
			`{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{6: "Select Program", 7: "Letter Count"}),
		},

		// Manual R1 conditional: Z direct, match ITA2 "....." (the '/' char) into C1.
		{
			"Colossus: R1 dots, Z direct", input,
			`{"printout":" \n00 00 : a1 \n","counters":[1,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: ".", 10: ".", 11: ".", 12: ".", 13: ".", 15: "1"}),
		},

		// Same, negated → counts everything except the single match.
		{
			"Colossus: R1 dots negated", input,
			`{"printout":" \n00 00 : a29 \n","counters":[29,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: ".", 10: ".", 11: ".", 12: ".", 13: ".", 14: true, 15: "1"}),
		},

		// ΔZ + Χ direct, Χ2 limitation, count-all (blank switches) into C1.
		{
			"Colossus: dZ + Chi, lim X2", input,
			`{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "ΔZ", 3: "Χ", 5: "Χ2", 15: "1"}),
		},

		// Ψ direct, "X2 + Ψ1" limitation (Latin X → only the Ψ1 limitation applies),
		// count-all into C2.
		{
			"Colossus: Psi, lim X2+Psi1, C2", input,
			`{"printout":" \n00 00 : b30 \n","counters":[0,30,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{4: "Ψ", 5: "X2 + Ψ1", 22: "2"}),
		},

		// Addition section: ΔZ+ΔΧ, Add Q1+Q2 = "." into C1.
		{
			"Colossus: addition Q1+Q2=.", input,
			`{"printout":" \n00 00 : a13 \n","counters":[13,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "ΔZ", 3: "ΔΧ", 32: true, 33: true, 37: ".", 38: true}),
		},

		// ZMUG pattern, Letter Count (counts every char regardless of pattern).
		{
			"Colossus: ZMUG Letter Count", input,
			`{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{1: "ZMUG Pattern", 6: "Select Program", 7: "Letter Count"}),
		},

		// BREAM pattern, Set Total 100 → counter line suppressed, header only.
		{
			"Colossus: BREAM Set Total 100", input,
			`{"printout":" \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{1: "BREAM Pattern", 6: "Select Program", 7: "Letter Count", 42: 100}),
		},

		// Preset programs (exercise each colossusSelectProgram branch).
		{
			"Colossus: program /,5,U", input,
			`{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{6: "Select Program", 7: "/,5,U (Count chars to find X3)"}),
		},
		{
			"Colossus: program 4=5=/1=2", input,
			`{"printout":" \n00 00 : a3 \n","counters":[3,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "ΔZ", 3: "ΔΧ", 6: "Select Program", 7: "4=5=/1=2 (Given X1,X2 find X4,X5)"}),
		},
		{
			"Colossus: program 1+2=.", input,
			`{"printout":" \n00 00 : a13 \n","counters":[13,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "ΔZ", 3: "ΔΧ", 6: "Select Program", 7: "1+2=. (1+2 Break In, Find X1,X2)"}),
		},

		// ΔΨ input with the P5 limitation (covers the Ψ-delta and P5 paths).
		{
			"Colossus: dPsi + lim Χ2+P5", input,
			`{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{4: "ΔΨ", 5: "Χ2 + P5", 15: "1"}),
		},

		// Addition with Equals = "x" (Add-Equals comparison value 1).
		{
			"Colossus: addition Q1+Q2=x", input,
			`{"printout":" \n00 00 : a17 \n","counters":[17,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "ΔZ", 3: "ΔΧ", 32: true, 33: true, 37: "x", 38: true}),
		},

		// Conditional Negate-All, Addition Negate-All, and Total-Motor gating.
		{
			"Colossus: conditional Negate All", input,
			`{"printout":" \n00 00 : a29 \n","counters":[29,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: ".", 10: ".", 11: ".", 12: ".", 13: ".", 15: "1", 30: true}),
		},
		{
			"Colossus: addition Negate All", input,
			`{"printout":" \n00 00 : a12 \n","counters":[12,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: "x", 15: "1", 39: true}),
		},
		{
			"Colossus: Total Motor x, lim Χ2", input,
			`{"printout":" \n00 00 : a12 \n","counters":[12,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 5: "Χ2", 15: "1", 40: "x"}),
		},
		{
			"Colossus: Total Motor ., lim Χ2", input,
			`{"printout":" \n00 00 : a18 \n","counters":[18,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 5: "Χ2", 15: "1", 40: "."}),
		},

		// Three counters populated at once (C1/C2/C3, "x" and "." switches).
		{
			"Colossus: three counters", input,
			`{"printout":" \n00 00 : a12 b18 c5 \n","counters":[12,18,5,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: ".", 15: "1", 16: "x", 22: "2", 23: ".", 24: "x", 29: "3"}),
		},

		// Fast + slow stepping (X5 fast, X4 slow), Set Total suppresses all counter
		// lines so only the header prints; exercises the full 598-iteration rotor
		// cycle and the slow-step branch. (Verified against the oracle with numeric
		// rotor starts — CyberChef's own string-typed starts infinite-loop here due
		// to a JSON.stringify(number) != JSON.stringify(string) termination bug.)
		{
			"Colossus: fast X5 + slow X4", input,
			`{"printout":"X5 X4\n","counters":[30,0,0,0,0],"runcount":599}`,
			colRecipe(map[int]any{6: "Select Program", 7: "Letter Count", 43: "X5", 44: "X4", 42: 9999}),
		},

		// A 150-char tape (> the 61/37 motor and 43-59 Psi wheel sizes) so those
		// rotor pointers wrap within a single tape pass.
		{
			"Colossus: long tape Letter Count", strings.Repeat(input, 5),
			`{"printout":" \n00 00 : a150 \n","counters":[150,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{6: "Select Program", 7: "Letter Count"}),
		},

		// Two conditional rows feeding the same counter (C1): the AND of a "." row
		// and an "x" row on impulse 1 can never both hold, so C1 stays 0.
		{
			"Colossus: two rows, one counter", input,
			`{"printout":" \n","counters":[0,0,0,0,0],"runcount":2}`,
			colRecipe(map[int]any{2: "Z", 9: ".", 15: "1", 16: "x", 22: "1"}),
		},
	})
}

// TestColEquals documents the Add-Equals switch → comparison-value mapping.
func TestColEquals(t *testing.T) {
	if colEquals("") != -1 || colEquals(".") != 0 || colEquals("x") != 1 {
		t.Fatal("colEquals mapping")
	}
}

// TestColNum documents colNum's numeric coercion: float64 (as CoerceArgs
// delivers), a bare int, and the defensive zero fallback.
func TestColNum(t *testing.T) {
	if colNum(float64(42)) != 42 || colNum(7) != 7 || colNum("x") != 0 {
		t.Fatal("colNum coercion")
	}
}

func TestColossusErrors(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		over    map[int]any
		wantErr string
	}{
		{"invalid ITA2 char", "ABC1", nil, "Invalid ITA2 character : 1"},
		{"invalid ITA2 space", "AB C", nil, "Invalid ITA2 character : Space"},
		{"invalid ITA2 newline", "AB\nC", nil, "Invalid ITA2 character : Carriage Return"},
		{"bad R1-Q1 switch", "ABC", map[int]any{9: "z"}, "Switch R1-Q1 can only be set to blank, . or x"},
		{"bad Add-Equals switch", "ABC", map[int]any{37: "z"}, "Switch Add-Equals can only be set to blank, . or x"},
		{"bad Total Motor switch", "ABC", map[int]any{40: "z"}, "Switch Total Motor can only be set to blank, . or x"},
		{"set total too high", "ABC", map[int]any{42: 10000}, "Set Total must be between 0000 and 9999"},
		{"X1 start out of range", "ABC", map[int]any{45: 50}, "Χ1 start must be between 1 and 41"},
		{"Psi1 start out of range", "ABC", map[int]any{52: 44}, "Ψ1 start must be between 1 and 43"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := with(colArgs(), c.over)
			_, err := runOp(t, "Colossus", c.input, args...)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got err %v, want containing %q", err, c.wantErr)
			}
		})
	}
}
