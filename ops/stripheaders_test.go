package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// stripRecipe brackets a header-strip op with From Hex / To Hex so byte-array
// input and output are expressed as hex, mirroring CyberChef's fixtures.
func stripRecipe(op string) core.Recipe {
	return core.Recipe{
		{Op: "From Hex", Args: []any{"None"}},
		{Op: op, Args: []any{}},
		{Op: "To Hex", Args: []any{"None"}},
	}
}

// TestStripHeaderFixtures transcribes the success cases from CyberChef's
// StripIPv4Header/StripTCPHeader/StripUDPHeader fixtures.
func TestStripHeaderFixtures(t *testing.T) {
	runCases(t, []opCase{
		// Strip IPv4 header
		{
			"IPv4: no options, no payload", "450000140005400080060000c0a80001c0a80002", "",
			stripRecipe("Strip IPv4 header"),
		},
		{
			"IPv4: no options, payload", "450000140005400080060000c0a80001c0a80002ffffffffffffffff", "ffffffffffffffff",
			stripRecipe("Strip IPv4 header"),
		},
		{
			"IPv4: options, no payload", "460000140005400080060000c0a80001c0a8000207000000", "",
			stripRecipe("Strip IPv4 header"),
		},
		{
			"IPv4: options, payload", "460000140005400080060000c0a80001c0a8000207000000ffffffffffffffff", "ffffffffffffffff",
			stripRecipe("Strip IPv4 header"),
		},

		// Strip TCP header
		{
			"TCP: no options, no payload", "7f900050000fa4b2000cb2a45010bff100000000", "",
			stripRecipe("Strip TCP header"),
		},
		{
			"TCP: no options, payload", "7f900050000fa4b2000cb2a45010bff100000000ffffffffffffffff", "ffffffffffffffff",
			stripRecipe("Strip TCP header"),
		},
		{
			"TCP: options, no payload", "7f900050000fa4b2000cb2a47010bff100000000020405b404020000", "",
			stripRecipe("Strip TCP header"),
		},
		{
			"TCP: options, payload", "7f900050000fa4b2000cb2a47010bff100000000020405b404020000ffffffffffffffff", "ffffffffffffffff",
			stripRecipe("Strip TCP header"),
		},

		// Strip UDP header
		{"UDP: no payload", "8111003500000000", "", stripRecipe("Strip UDP header")},
		{"UDP: payload", "8111003500080000ffffffffffffffff", "ffffffffffffffff", stripRecipe("Strip UDP header")},
	})
}

// TestStripHeaderErrors covers the length-validation error cases; CyberChef
// surfaces these as OperationErrors, which cchef returns from Execute.
func TestStripHeaderErrors(t *testing.T) {
	cases := []struct {
		name, op, input, wantErr string
	}{
		{"IPv4 < min header", "Strip IPv4 header", "450000140005400080060000c0a80001c0a800", "input length is less than minimum IPv4 header length"},
		{"IPv4 < IHL", "Strip IPv4 header", "460000140005400080060000c0a80001c0a80000", "input length is less than IHL"},
		{"TCP < min header", "Strip TCP header", "7f900050000fa4b2000cb2a45010bff1000000", "need at least 20 bytes for a TCP header"},
		{"TCP < data offset", "Strip TCP header", "7f900050000fa4b2000cb2a47010bff100000000", "input length is less than data offset"},
		{"UDP < header", "Strip UDP header", "81110035000000", "need 8 bytes for a UDP header"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := stripRecipe(c.op).Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got err %v, want containing %q", err, c.wantErr)
			}
		})
	}
}
