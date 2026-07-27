package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// analyseUUIDRecipe reads a UUID, with or without the metadata section.
func analyseUUIDRecipe(metadata bool) core.Recipe {
	return core.Recipe{{Op: "Analyse UUID", Args: []any{metadata}}}
}

// TestAnalyseUUIDFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/AnalyseUUID.mjs).
func TestAnalyseUUIDFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"v1 UUID extracts timestamp, clock, and node",
			"cefa1760-28ee-11f1-9f95-1fb76af3e239",
			"Version:\n1\n\nTimestamp:\n1774514156502\n\n" +
				"Timestamp (ISO):\n2026-03-26T08:35:56.502Z\n\n" +
				"Node:\n1F:B7:6A:F3:E2:39\n\nClock:\n8085\n\n" +
				"UUID Integer:\n275119515460318071558429785403790975545",
			analyseUUIDRecipe(true),
		},
		{
			"v7 UUID extracts timestamp, randA, and randB",
			"019d294a-af64-7728-9524-26da08f50708",
			"Version:\n7\n\nTimestamp:\n1774514253668\n\n" +
				"Timestamp (ISO):\n2026-03-26T08:37:33.668Z\n\n" +
				"Rand A:\n1832\n\nRand B:\n952426DA08F50708\n\n" +
				"UUID Integer:\n2145256098533991595556290452700595976",
			analyseUUIDRecipe(true),
		},
		{
			"v4 UUID should show no metadata - not possible",
			"f47ac10b-58cc-4372-a567-0e02b2c3d479",
			"Version:\n4\n\nNo metadata available. Only versions 1, 6, 7 are supported.\n\n" +
				"UUID Integer:\n324969006592305634633390616021200786553",
			analyseUUIDRecipe(true),
		},
		{
			"metadata left out on request",
			"cefa1760-28ee-11f1-9f95-1fb76af3e239",
			"Version:\n1\n\nUUID Integer:\n275119515460318071558429785403790975545",
			analyseUUIDRecipe(false),
		},
	})
}

// TestAnalyseUUIDEdges covers the corners of the format against the oracle: the
// two UUIDs the standard reserves, a version 6, the extremes of each
// timestamped layout, and input that arrives in a different shape than expected.
func TestAnalyseUUIDEdges(t *testing.T) {
	runCases(t, []opCase{
		{
			"the nil UUID, whose version reads as zero",
			"00000000-0000-0000-0000-000000000000",
			"Version:\n0\n\nNo metadata available. Only versions 1, 6, 7 are supported.\n\n" +
				"UUID Integer:\n0",
			analyseUUIDRecipe(true),
		},
		{
			"the max UUID, whose version reads as fifteen",
			"ffffffff-ffff-ffff-ffff-ffffffffffff",
			"Version:\n15\n\nNo metadata available. Only versions 1, 6, 7 are supported.\n\n" +
				"UUID Integer:\n340282366920938463463374607431768211455",
			analyseUUIDRecipe(true),
		},
		{
			"a v6 UUID, which is a v1 with its timestamp reordered",
			"1ef128ee-cefa-6760-9f95-1fb76af3e239",
			"Version:\n6\n\nTimestamp:\n1715759054155\n\n" +
				"Timestamp (ISO):\n2024-05-15T07:44:14.155Z\n\n" +
				"Node:\n1F:B7:6A:F3:E2:39\n\nClock:\n8085\n\n" +
				"UUID Integer:\n41129013633197825771554406774013682233",
			analyseUUIDRecipe(true),
		},
		{
			"a version the metadata parsers do not cover",
			"01234567-89ab-8cde-b012-3456789abcde",
			"Version:\n8\n\nNo metadata available. Only versions 1, 6, 7 are supported.\n\n" +
				"UUID Integer:\n1512366075203863674238821691440151774",
			analyseUUIDRecipe(true),
		},
		{
			"a v1 timestamp of zero, which predates the epoch",
			"00000000-0000-1000-8000-000000000000",
			"Version:\n1\n\nTimestamp:\n-12219292800000\n\n" +
				"Timestamp (ISO):\n1582-10-15T00:00:00.000Z\n\n" +
				"Node:\n00:00:00:00:00:00\n\nClock:\n0\n\n" +
				"UUID Integer:\n75567087097951178194944",
			analyseUUIDRecipe(true),
		},
		{
			"the largest v7 timestamp, which falls in a five-figure year",
			"ffffffff-ffff-7fff-bfff-ffffffffffff",
			"Version:\n7\n\nTimestamp:\n281474976710655\n\n" +
				"Timestamp (ISO):\n+010889-08-02T05:31:50.655Z\n\n" +
				"Rand A:\n4095\n\nRand B:\nBFFFFFFFFFFFFFFF\n\n" +
				"UUID Integer:\n340282366920937858995853114098753470463",
			analyseUUIDRecipe(true),
		},
		{
			"upper case, which the format treats the same as lower",
			"CEFA1760-28EE-11F1-9F95-1FB76AF3E239",
			"Version:\n1\n\nTimestamp:\n1774514156502\n\n" +
				"Timestamp (ISO):\n2026-03-26T08:35:56.502Z\n\n" +
				"Node:\n1F:B7:6A:F3:E2:39\n\nClock:\n8085\n\n" +
				"UUID Integer:\n275119515460318071558429785403790975545",
			analyseUUIDRecipe(true),
		},
		{
			"surrounding space, which is trimmed away",
			"  cefa1760-28ee-11f1-9f95-1fb76af3e239\n",
			"Version:\n1\n\nTimestamp:\n1774514156502\n\n" +
				"Timestamp (ISO):\n2026-03-26T08:35:56.502Z\n\n" +
				"Node:\n1F:B7:6A:F3:E2:39\n\nClock:\n8085\n\n" +
				"UUID Integer:\n275119515460318071558429785403790975545",
			analyseUUIDRecipe(true),
		},
	})
}

// TestAnalyseUUIDRejects covers what the operation will not read. The check is
// narrower than it looks: only versions one to eight and the four variant
// digits are accepted, with the nil and max UUIDs allowed by name.
func TestAnalyseUUIDRejects(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"text that is not a UUID at all", "not-a-uuid"},
		{"a version outside the range the format defines", "f47ac10b-58cc-9372-a567-0e02b2c3d479"},
		{"a variant outside the range the format defines", "f47ac10b-58cc-4372-c567-0e02b2c3d479"},
		{"braces around it", "{cefa1760-28ee-11f1-9f95-1fb76af3e239}"},
		{"the URN form", "urn:uuid:cefa1760-28ee-11f1-9f95-1fb76af3e239"},
		{"the hyphens left out", "cefa176028ee11f19f951fb76af3e239"},
		{"a digit short", "cefa1760-28ee-11f1-9f95-1fb76af3e23"},
		{"nothing at all", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Analyse UUID", tc.input, true)
			if err == nil {
				t.Fatalf("read %q as a UUID, giving %q", tc.input, out)
			}
			if err.Error() != "Invalid UUID" {
				t.Errorf("got %q, want %q", err.Error(), "Invalid UUID")
			}
		})
	}
}
