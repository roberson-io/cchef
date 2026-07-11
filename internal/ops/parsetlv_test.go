package ops

import (
	"math"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseTLVFixtures transcribes the CyberChef fixtures
// (../CyberChef/tests/operations/tests/ParseTLV.mjs). cchef emits compact JSON.
func TestParseTLVFixtures(t *testing.T) {
	lv := `[{"length":5,"value":[72,111,117,115,101]},{"length":4,"value":[114,111,111,109]},{"length":4,"value":[100,111,111,114]}]`
	klv := `[{"key":[4],"length":5,"value":[72,111,117,115,101]},{"key":[5],"length":4,"value":[114,111,111,109]},{"key":[66],"length":4,"value":[100,111,111,114]}]`

	// BER long-form value arrays.
	a256 := "[" + strings.Repeat("65,", 255) + "65]"
	b128 := "[" + strings.Repeat("66,", 127) + "66]"

	runCases(t, []opCase{
		{
			"LengthValue", "\x05\x48\x6f\x75\x73\x65\x04\x72\x6f\x6f\x6d\x04\x64\x6f\x6f\x72", lv,
			core.Recipe{{Op: "Parse TLV", Args: []any{0, 1, false}}},
		},
		{
			"LengthValue with BER", "\x05\x48\x6f\x75\x73\x65\x04\x72\x6f\x6f\x6d\x04\x64\x6f\x6f\x72", lv,
			core.Recipe{{Op: "Parse TLV", Args: []any{0, 4, true}}},
		},
		{
			"KeyLengthValue", "\x04\x05\x48\x6f\x75\x73\x65\x05\x04\x72\x6f\x6f\x6d\x42\x04\x64\x6f\x6f\x72", klv,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, false}}},
		},
		{
			"KeyLengthValue with BER", "\x04\x05\x48\x6f\x75\x73\x65\x05\x04\x72\x6f\x6f\x6d\x42\x04\x64\x6f\x6f\x72", klv,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 4, true}}},
		},
		{
			"BER long-form two-byte length",
			"\x01\x82\x01\x00" + strings.Repeat("A", 256) + "\x02\x03\x41\x42\x43",
			`[{"key":[1],"length":256,"value":` + a256 + `},{"key":[2],"length":3,"value":[65,66,67]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, true}}},
		},
		{
			"BER long-form one-byte length",
			"\x01\x81\x80" + strings.Repeat("B", 128),
			`[{"key":[1],"length":128,"value":` + b128 + `}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, true}}},
		},
		{
			"BER mixed short and long-form",
			"\x01\x05\x48\x65\x6c\x6c\x6f\x02\x81\x05\x57\x6f\x72\x6c\x64",
			`[{"key":[1],"length":5,"value":[72,101,108,108,111]},{"key":[2],"length":5,"value":[87,111,114,108,100]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, true}}},
		},
	})
}

// TestParseTLVEdges covers the JS parser's overrun quirks, verified against the
// CyberChef-server oracle (non-BER, where the oracle image is faithful):
//   - getValue reads one byte past the end (undefined -> JSON null) when the
//     declared length overruns the buffer;
//   - getLength yields NaN (-> JSON null) when a length byte is missing;
//   - a truthy-but-negative key size reads zero bytes -> empty key array.
func TestParseTLVEdges(t *testing.T) {
	runCases(t, []opCase{
		{
			"empty input", "", `[]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, false}}},
		},
		{
			"value overruns buffer", "\x05\x41\x42", `[{"length":5,"value":[65,66,null]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{0, 1, false}}},
		},
		{
			"length byte missing", "\x04", `[{"key":[4],"length":null,"value":[]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{1, 1, false}}},
		},
		{
			"key overruns buffer", "\x04", `[{"key":[4,null],"length":null,"value":[]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{2, 1, false}}},
		},
		{
			"negative key size", "\x04\x41", `[{"key":[],"length":4,"value":[65,null]}]`,
			core.Recipe{{Op: "Parse TLV", Args: []any{-1, 1, false}}},
		},
	})
}

// TestToInt32 directly exercises the JS ToInt32 helper, including the NaN/Inf
// guard and the modulo wrap-around branches that BER big-endian length decoding
// relies on but that no ordinary TLV input reaches.
func TestToInt32(t *testing.T) {
	cases := []struct {
		in   float64
		want int32
	}{
		{0, 0},
		{5, 5},
		{math.NaN(), 0},
		{math.Inf(1), 0},
		{math.Inf(-1), 0},
		{-1, -1},                  // negative modulo, then high-bit subtraction
		{2147483648, -2147483648}, // 2^31 wraps to INT32_MIN
		{4294967296, 0},           // 2^32 wraps to 0
		{4294967297, 1},           // 2^32 + 1
		{3.9, 3},                  // truncation toward zero
		{-2147483649, 2147483647}, // -(2^31)-1 wraps to INT32_MAX
	}
	for _, c := range cases {
		if got := toInt32(c.in); got != c.want {
			t.Errorf("toInt32(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestParseTLVSizeError covers the guard rejecting non-positive key and length
// sizes together.
func TestParseTLVSizeError(t *testing.T) {
	r := core.Recipe{{Op: "Parse TLV", Args: []any{0, 0, false}}}
	if _, err := r.Execute(core.NewDish([]byte("\x04\x05"), core.TypeString)); err == nil ||
		!strings.Contains(err.Error(), "Type or Length size must be greater than 0") {
		t.Fatalf("got err %v, want size error", err)
	}
}
