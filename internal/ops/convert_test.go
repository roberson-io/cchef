package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Hand-verified unit-conversion cases (no upstream fixtures). Values use exact
// rational arithmetic, so results are exact terminating decimals.
func TestConverters(t *testing.T) {
	runCases(t, []opCase{
		{
			"distance km->m", "1", "1000",
			core.Recipe{{Op: "Convert distance", Args: []any{"Kilometers (km)", "Metres (m)"}}},
		},
		{
			"distance mi->km", "1", "1.609344",
			core.Recipe{{Op: "Convert distance", Args: []any{"Miles (mi)", "Kilometers (km)"}}},
		},
		{
			"distance in->cm", "1", "2.54",
			core.Recipe{{Op: "Convert distance", Args: []any{"Inches (in)", "Centimetres (cm)"}}},
		},
		{
			"distance fractional", "2.5", "2500",
			core.Recipe{{Op: "Convert distance", Args: []any{"Kilometers (km)", "Metres (m)"}}},
		},

		{
			"mass kg->g", "1", "1000",
			core.Recipe{{Op: "Convert mass", Args: []any{"Kilogram (kg)", "Gram (g)"}}},
		},
		{
			"mass lb->g", "1", "453.59237",
			core.Recipe{{Op: "Convert mass", Args: []any{"Pound (lb)", "Gram (g)"}}},
		},

		{
			"speed c->m/s", "1", "299792458",
			core.Recipe{{Op: "Convert speed", Args: []any{"Light (c)", "Metres per second (m/s)"}}},
		},

		{
			"area ha->sqm", "1", "10000",
			core.Recipe{{Op: "Convert area", Args: []any{"Hectare (ha)", "Square metre (sq m)"}}},
		},

		{
			"data Byte->bits", "1", "8",
			core.Recipe{{Op: "Convert data units", Args: []any{"Bytes (B)", "Bits (b)"}}},
		},
		{
			"data KiB->Bytes", "1", "1024",
			core.Recipe{{Op: "Convert data units", Args: []any{"Kibibytes (KiB)", "Bytes (B)"}}},
		},

		// Round-trip through a unit whose intermediate value terminates exactly.
		{
			"round trip distance", "42", "42",
			core.Recipe{
				{Op: "Convert distance", Args: []any{"Metres (m)", "Kilometers (km)"}},
				{Op: "Convert distance", Args: []any{"Kilometers (km)", "Metres (m)"}},
			},
		},
	})
}
