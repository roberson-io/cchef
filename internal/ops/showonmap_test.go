package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Show on map output (the parsed lat,lon pair) verified against the CyberChef-server oracle.
func TestShowOnMap(t *testing.T) {
	runCases(t, []opCase{
		{
			"dd pair", "51.5074, -0.1278", "51.5074,-0.1278",
			core.Recipe{{Op: "Show on map", Args: []any{13.0, "Auto", "Auto"}}},
		},
		{
			"dms", "51° 30' 26.64\", -0° 7' 40.08\"", "51.5074,-0.1278",
			core.Recipe{{Op: "Show on map", Args: []any{13.0, "Degrees Minutes Seconds", "Comma"}}},
		},
		{
			"mgrs", "30UXC9931610163", "51.50739,-0.1278",
			core.Recipe{{Op: "Show on map", Args: []any{13.0, "Military Grid Reference System", "Comma"}}},
		},
		{
			"geohash", "gcpvj0duq", "51.5074,-0.12778",
			core.Recipe{{Op: "Show on map", Args: []any{13.0, "Geohash", "Comma"}}},
		},
	})
}

// Show on map errors when the delimiter cannot be auto-detected (matches CyberChef).
func TestShowOnMapErrors(t *testing.T) {
	if _, err := runOp(t, "Show on map", "30UXC9931610163", 13.0, "Military Grid Reference System", "Auto"); err == nil {
		t.Error("expected error for undetectable delimiter")
	}
	// A single (non-pair) coordinate yields one value, which is not a lat/lon pair.
	if _, err := runOp(t, "Show on map", "51.5", 13.0, "Decimal Degrees", "Comma"); err == nil {
		t.Error("expected error for a non-pair coordinate")
	}
	// Blank input passes through unchanged.
	if out, err := runOp(t, "Show on map", "  ", 13.0, "Auto", "Auto"); err != nil || out != "  " {
		t.Errorf("blank input: got %q, err %v", out, err)
	}
}
