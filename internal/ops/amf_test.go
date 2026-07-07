package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// AMF has no CyberChef fixture file. AMF0 primitive encodings below are
// spec-authoritative (computed from the AMF0 specification); the round-trip
// cases assert the stronger property that decode(encode(x)) == x.
func TestAMFFixtures(t *testing.T) {
	runCases(t, []opCase{
		// AMF0 primitive byte vectors, viewed as hex.
		// string "hi": marker 0x02, uint16 len 0x0002, "hi".
		{
			"AMF0 encode string", `"hi"`, "0200026869",
			core.Recipe{
				{Op: "AMF Encode", Args: []any{"AMF0"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// number 5: marker 0x00, IEEE-754 double 5.0 big-endian.
		{
			"AMF0 encode number", `5`, "004014000000000000",
			core.Recipe{
				{Op: "AMF Encode", Args: []any{"AMF0"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		// boolean true: marker 0x01, value 0x01.
		{
			"AMF0 encode boolean", `true`, "0101",
			core.Recipe{
				{Op: "AMF Encode", Args: []any{"AMF0"}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},

		// AMF0 decode of the same string vector back to JSON.
		{
			"AMF0 decode string", "0200026869", `"hi"`,
			core.Recipe{
				{Op: "From Hex", Args: []any{"None"}},
				{Op: "AMF Decode", Args: []any{"AMF0"}},
			},
		},
	})
}

// TestAMFRoundTrip encodes JSON to AMF and decodes it back for both formats.
func TestAMFRoundTrip(t *testing.T) {
	inputs := []string{
		`5`,
		`"hello"`,
		`true`,
		`null`,
		`[1,2,3]`,
		`{"a":1,"b":"x","c":true}`,
		`{"arr":["a","b"],"nested":{"k":1}}`,
	}
	var cases []opCase
	for _, format := range []string{"AMF0", "AMF3"} {
		for _, in := range inputs {
			cases = append(cases, opCase{
				name:  format + " " + in,
				input: in,
				want:  in,
				recipe: core.Recipe{
					{Op: "AMF Encode", Args: []any{format}},
					{Op: "AMF Decode", Args: []any{format}},
				},
			})
		}
	}
	runCases(t, cases)
}

// TestAMFErrors covers the decode error (invalid marker byte) and the encode
// error (unparseable JSON input).
func TestAMFErrors(t *testing.T) {
	if _, err := runOp(t, "AMF Decode", "\xff", "AMF0"); err == nil {
		t.Fatal("expected an error decoding an invalid AMF marker")
	}
	if _, err := runOp(t, "AMF Encode", "not json", "AMF0"); err == nil {
		t.Fatal("expected an error encoding invalid JSON")
	}
}
