package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Convert co-ordinate format outputs verified against the CyberChef-server oracle.
func TestConvertCoordinateFormat(t *testing.T) {
	dd2 := func(out string, args ...any) opCase {
		return opCase{
			out, "51.5074, -0.1278", out,
			core.Recipe{{Op: "Convert co-ordinate format", Args: args}},
		}
	}
	runCases(t, []opCase{
		dd2("51° 30' 26.64\",-0° 7' 40.08\",",
			"Decimal Degrees", "Auto", "Degrees Minutes Seconds", "Comma", "None", 3.0),
		dd2("51° 30.444',-0° 7.668',",
			"Decimal Degrees", "Auto", "Degrees Decimal Minutes", "Comma", "None", 3.0),
		dd2("gcpvj0duq,",
			"Decimal Degrees", "Auto", "Geohash", "Comma", "None", 9.0),
		dd2("30U XC 99316 10163,",
			"Decimal Degrees", "Auto", "Military Grid Reference System", "Comma", "None", 10.0),
		dd2("TQ 30028 80380,",
			"Decimal Degrees", "Auto", "Ordnance Survey National Grid", "Comma", "None", 10.0),
		dd2("51.5074° N,0.1278° W,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "After", 4.0),
		{
			"NY UTM", "40.7128, -74.0060", "18 N 583959.372 4507350.998,",
			core.Recipe{{Op: "Convert co-ordinate format", Args: []any{
				"Decimal Degrees", "Auto", "Universal Transverse Mercator", "Comma", "None", 3.0,
			}}},
		},
	})
}

// TestConvertCoordinateFormatInputs exercises each input format (parsing to lat/lon),
// verified against the CyberChef-server oracle.
func TestConvertCoordinateFormatInputs(t *testing.T) {
	toDD := func(in, inFmt, inDelim string, prec float64) core.Recipe {
		return core.Recipe{{Op: "Convert co-ordinate format", Args: []any{
			inFmt, inDelim, "Decimal Degrees", "Comma", "None", prec,
		}}}
	}
	runCases(t, []opCase{
		{
			"dms input", "51° 30' 26.64\", -0° 7' 40.08\"", "51.5074°,-0.1278°,",
			toDD("51° 30' 26.64\", -0° 7' 40.08\"", "Degrees Minutes Seconds", "Comma", 5.0),
		},
		{
			"ddm input", "51° 30.444', -0° 7.668'", "51.5074°,-0.1278°,",
			toDD("51° 30.444', -0° 7.668'", "Degrees Decimal Minutes", "Comma", 5.0),
		},
		{
			"mgrs input", "30UXC9931610163", "51.50739°,-0.1278°,",
			toDD("30UXC9931610163", "Military Grid Reference System", "Comma", 5.0),
		},
		{
			"utm input", "30 N 699316.234 5710163.758", "51.5074°,-0.1278°,",
			toDD("30 N 699316.234 5710163.758", "Universal Transverse Mercator", "Comma", 5.0),
		},
		{
			"geohash input", "gcpvj0duq", "51.5074°,-0.12778°,",
			toDD("gcpvj0duq", "Geohash", "Comma", 5.0),
		},
		{
			"direction preceding", "N51.5074 W0.1278", "51.5074°,-0.1278°,",
			toDD("N51.5074 W0.1278", "Decimal Degrees", "Direction Preceding", 4.0),
		},
	})
}

// TestConvertCoordinateFormatOSGBInput covers Ordnance Survey National Grid as input.
// The inverse Helmert transform of the Go geodesy library differs from CyberChef by a
// few metres (here the latitude: 51.50737 vs CyberChef's 51.5074), so this asserts
// cchef's own output rather than the oracle's.
func TestConvertCoordinateFormatOSGBInput(t *testing.T) {
	out, err := runOp(t, "Convert co-ordinate format",
		"TQ 30028 80380", "Ordnance Survey National Grid", "Comma", "Decimal Degrees", "Comma", "None", 5.0)
	if err != nil {
		t.Fatal(err)
	}
	if want := "51.50737°,-0.12781°,"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestConvertCoordinateFormatErrors covers the error branches.
func TestConvertCoordinateFormatErrors(t *testing.T) {
	// Undetectable delimiter (single token, Auto delimiter).
	if _, err := runOp(t, "Convert co-ordinate format",
		"30UXC9931610163", "Geohash", "Auto", "Decimal Degrees", "Comma", "None", 5.0); err == nil {
		t.Error("expected error for undetectable delimiter")
	}
	// Blank input passes through unchanged.
	if out, err := runOp(t, "Convert co-ordinate format",
		"   ", "Auto", "Auto", "Decimal Degrees", "Comma", "None", 5.0); err != nil || out != "   " {
		t.Errorf("blank input: got %q, err %v", out, err)
	}
	// Malformed UTM inputs exercise utmParse's error branches.
	utmErrs := []string{
		"56",                      // too few fields
		"5x S 334368.6 6250948.3", // non-numeric zone
		"56 Q 334368.6 6250948.3", // invalid hemisphere (not N/S)
		"56 S east 6250948.3",     // non-numeric easting
	}
	for _, in := range utmErrs {
		if _, err := runOp(t, "Convert co-ordinate format",
			in, "Universal Transverse Mercator", "Comma", "Decimal Degrees", "Comma", "None", 3.0); err == nil {
			t.Errorf("expected error for malformed UTM %q", in)
		}
	}
	// Auto input format that cannot be classified.
	if _, err := runOp(t, "Convert co-ordinate format",
		"???", "Auto", "Comma", "Decimal Degrees", "Comma", "None", 3.0); err == nil {
		t.Error("expected error for undetectable input format")
	}
	// Each explicit degree-based format rejects inputs with the wrong component
	// count, in both the paired and single-value branches (splitInput arity guards).
	fmtErrs := []struct{ name, input, inFmt string }{
		{"DMS pair too few", "51 30, 0 7", "Degrees Minutes Seconds"},
		{"DMS single too few", "51 30", "Degrees Minutes Seconds"},
		{"DDM pair wrong arity", "51, 0", "Degrees Decimal Minutes"},
		{"DDM single wrong arity", "51", "Degrees Decimal Minutes"},
		{"DD pair wrong arity", "51 30, 0 7", "Decimal Degrees"},
		{"DD single wrong arity", "51 30", "Decimal Degrees"},
	}
	for _, c := range fmtErrs {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, "Convert co-ordinate format",
				c.input, c.inFmt, "Comma", "Decimal Degrees", "Comma", "None", 3.0); err == nil {
				t.Errorf("expected error for %s %q", c.inFmt, c.input)
			}
		})
	}
	// Ordnance Survey National Grid input rejects references that are too short
	// (osgbParse len<2) or have an odd number of easting/northing digits.
	for _, in := range []string{"A", "TQ123"} {
		if _, err := runOp(t, "Convert co-ordinate format",
			in, "Ordnance Survey National Grid", "Comma", "Decimal Degrees", "Comma", "None", 5.0); err == nil {
			t.Errorf("expected error for malformed OSNG %q", in)
		}
	}
}

// TestConvertCoordinateFormatAutoDetect exercises the format/delimiter/direction
// auto-detection (findFormat/findDelim/findDirs). Every format branch of
// findFormat is reached via an "Auto" input format. Outputs are verified against
// the CyberChef-server oracle, except OSNG input, whose OSGB Helmert transform
// differs from CyberChef by a few metres (see TestConvertCoordinateFormatOSGBInput).
func TestConvertCoordinateFormatAutoDetect(t *testing.T) {
	cc := func(name, in, out string, args ...any) opCase {
		return opCase{name, in, out, core.Recipe{{Op: "Convert co-ordinate format", Args: args}}}
	}
	runCases(t, []opCase{
		// Auto input-format detection, one per findFormat branch.
		cc("auto dms", "51° 30' 26.64\", -0° 7' 40.08\"", "51.5074°,-0.1278°,",
			"Auto", "Auto", "Decimal Degrees", "Comma", "None", 5.0),
		cc("auto ddm", "51° 30.444', -0° 7.668'", "51.5074°,-0.1278°,",
			"Auto", "Auto", "Decimal Degrees", "Comma", "None", 5.0),
		cc("auto dd", "51.5074, -0.1278", "51.5074°,-0.1278°,",
			"Auto", "Auto", "Decimal Degrees", "Comma", "None", 5.0),
		cc("auto mgrs", "30UXC9931610163,", "51.50739°,-0.1278°,",
			"Auto", "Comma", "Decimal Degrees", "Comma", "None", 5.0),
		cc("auto utm", "30 N 699316.234 5710163.758,", "51.5074°,-0.1278°,",
			"Auto", "Comma", "Decimal Degrees", "Comma", "None", 5.0),
		cc("auto geohash", "gcpvj0duq,", "51.5074°,-0.12778°,",
			"Auto", "Comma", "Decimal Degrees", "Comma", "None", 5.0),
		// Southern-hemisphere UTM exercises utmParse's hemi=="S" branch (Sydney).
		cc("auto utm south", "56 S 334368.634 6250948.345,", "-33.8688°,151.2093°,",
			"Auto", "Comma", "Decimal Degrees", "Comma", "None", 4.0),
		cc("auto osng (cchef-own; OSGB metre diff)", "TQ 30028 80380,", "51.50737°,-0.12781°,",
			"Auto", "Comma", "Decimal Degrees", "Comma", "None", 5.0),

		// Auto delimiter detection, one per findDelim branch.
		cc("auto delim semicolon", "51.5074; -0.1278", "51.5074°,-0.1278°,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "None", 4.0),
		cc("auto delim colon", "51.5074: -0.1278", "51.5074°,-0.1278°,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "None", 4.0),
		cc("auto delim direction-preceding", "N51.5074 W0.1278", "51.5074°,-0.1278°,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "None", 4.0),

		// Direction handling in input and output (findDirs); CyberChef fixtures.
		cc("dirs in input, not output", "N51.504°,W0.126°,", "51.504°,-0.126°,",
			"Decimal Degrees", "Comma", "Decimal Degrees", "Comma", "None", 3.0),
		cc("dirs in input and output", "N51.504°,W0.126°,", "N 51.504°,W 0.126°,",
			"Decimal Degrees", "Comma", "Decimal Degrees", "Comma", "Before", 3.0),
		cc("dirs not in input, in output", "51.504°,-0.126°,", "N 51.504°,W 0.126°,",
			"Decimal Degrees", "Comma", "Decimal Degrees", "Comma", "Before", 3.0),
		cc("dirs not in input, in converted output", "51.504°,-0.126°,", "N 51° 30' 14.4\",W 0° 7' 33.6\",",
			"Decimal Degrees", "Comma", "Degrees Minutes Seconds", "Comma", "Before", 3.0),
	})
}
