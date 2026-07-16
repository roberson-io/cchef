package ops

import (
	"strings"
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
		// Direction-following auto-detection (findDelim/findFormat direction paths).
		cc("auto delim direction-following", "51.5074N 0.1278W", "51.5074°,-0.1278°,",
			"Auto", "Auto", "Decimal Degrees", "Comma", "None", 4.0),
		// Auto format with a leading direction exercises findFormat's split[0]=="" arm.
		cc("auto format direction-preceding", "N51.5074 W0.1278", "51.5074°,-0.1278°,",
			"Auto", "Auto", "Decimal Degrees", "Comma", "None", 4.0),

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

// TestConvertCoordinateOutputBranches covers output-format precision handling,
// hemisphere/direction rendering, and the input normalisation/single-value paths.
// Values are oracle-verified except where noted (CyberChef errors on single-value
// coordinate inputs, so those assert cchef's own lenient behaviour).
func TestConvertCoordinateOutputBranches(t *testing.T) {
	cc := func(name, in, out string, args ...any) opCase {
		return opCase{name, in, out, core.Recipe{{Op: "Convert co-ordinate format", Args: args}}}
	}
	runCases(t, []opCase{
		// MGRS output with odd and >10 precision (rounded to even, capped at 10).
		cc("mgrs out odd precision", "51.5074, -0.1278", "30U XC 99 10,",
			"Decimal Degrees", "Auto", "Military Grid Reference System", "Comma", "None", 3.0),
		cc("mgrs out precision over 10", "51.5074, -0.1278", "30U XC 99316 10163,",
			"Decimal Degrees", "Auto", "Military Grid Reference System", "Comma", "None", 11.0),
		// OSNG output with odd and >10 precision (rounded to even, capped at 10).
		cc("osng out odd precision", "51.5074, -0.1278", "TQ 30 80,",
			"Decimal Degrees", "Auto", "Ordnance Survey National Grid", "Comma", "None", 3.0),
		cc("osng out precision over 10", "51.5074, -0.1278", "TQ 30028 80380,",
			"Decimal Degrees", "Auto", "Ordnance Survey National Grid", "Comma", "None", 11.0),
		// UTM output for a southern-hemisphere coordinate (hemi = "S").
		cc("utm out southern hemisphere", "-33.8688, 151.2093", "56 S 334368.634 6250948.345,",
			"Decimal Degrees", "Auto", "Universal Transverse Mercator", "Comma", "None", 3.0),
		// A leading S/W direction negates the value (Degrees-family input).
		cc("direction negates value", "S51.5074 W0.1278", "-51.5074°,-0.1278°,",
			"Decimal Degrees", "Direction Preceding", "Decimal Degrees", "Comma", "None", 4.0),
		// Negative latitude renders as an S direction (the "-" is stripped).
		cc("southern latitude output direction", "-33.8688, 151.2093", "S 33.869°,E 151.209°,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "Before", 3.0),
		// UTM input without a space between zone and hemisphere is normalised.
		cc("utm input no space", "30N 699316.234 5710163.758", "51.5074°,-0.1278°,",
			"Universal Transverse Mercator", "Comma", "Decimal Degrees", "Comma", "None", 4.0),
		// A negative precision is clamped to zero.
		cc("negative precision clamps to zero", "51.5074, -0.1278", "52°,0°,",
			"Decimal Degrees", "Auto", "Decimal Degrees", "Comma", "None", -1.0),
		// Single-value inputs (CyberChef errors on these; cchef emits a single value).
		cc("dms single value", "51 30 26.64", "51.5074°,",
			"Degrees Minutes Seconds", "Comma", "Decimal Degrees", "Comma", "None", 5.0),
		cc("ddm single value", "51 30.444", "51.5074°,",
			"Degrees Decimal Minutes", "Comma", "Decimal Degrees", "Comma", "None", 5.0),
	})
}

// TestConvertCoordinateMoreErrors covers additional error paths: an invalid MGRS
// reference, out-of-range OSNG output, and a UTM zone the library rejects.
func TestConvertCoordinateMoreErrors(t *testing.T) {
	errs := []struct {
		name, in, inFmt, outFmt string
		prec                    float64
	}{
		{"invalid MGRS", "ZZZZ", "Military Grid Reference System", "Decimal Degrees", 5.0},
		{"OSNG output out of range", "0, 0", "Decimal Degrees", "Ordnance Survey National Grid", 5.0},
		{"UTM zone rejected", "99 N 699316.234 5710163.758", "Universal Transverse Mercator", "Decimal Degrees", 5.0},
		{"MGRS output polar", "89, 0", "Decimal Degrees", "Military Grid Reference System", 5.0},
		{"UTM output polar", "89, 0", "Decimal Degrees", "Universal Transverse Mercator", 5.0},
		{"geohash zero precision", "51.5074, -0.1278", "Decimal Degrees", "Geohash", 0.0},
	}
	for _, c := range errs {
		if _, err := runOp(t, "Convert co-ordinate format",
			c.in, c.inFmt, "Comma", c.outFmt, "Comma", "None", c.prec); err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
	}
}

// TestCoordinateHelpers directly exercises the direction-detection helpers and
// the short-input guard in fmtMGRS; findDirs is only ever called with numeric
// data in convertCoordinates, so its explicit-direction branches are unreachable
// through the operation.
func TestCoordinateHelpers(t *testing.T) {
	if la, lo := findDirs("N51.5 W0.1", "Auto"); la != "N" || lo != "W" {
		t.Fatalf("findDirs(explicit) = %q,%q; want N,W", la, lo)
	}
	// A single explicit direction returns an empty second direction.
	if la, lo := findDirs("N51.5", "Auto"); la != "N" || lo != "" {
		t.Fatalf("findDirs(single) = %q,%q; want N,\"\"", la, lo)
	}
	// Three directions bypass the <=2 shortcut and use the direction-split path;
	// leading direction (split[0]=="") and following direction (split[0]!="").
	if la, lo := findDirs("N51 E0 W1", "Direction Preceding"); la == "" || lo == "" {
		t.Fatalf("findDirs(leading direction) = %q,%q; want non-empty", la, lo)
	}
	if la, lo := findDirs("51N 0E 1W", "Direction Following"); la == "" || lo == "" {
		t.Fatalf("findDirs(following direction) = %q,%q; want non-empty", la, lo)
	}
	if got := findDelim("51.5N 0.1W"); got != "Direction Following" {
		t.Fatalf("findDelim = %q, want Direction Following", got)
	}
	if got := fmtMGRS("30U"); got != "30U" {
		t.Fatalf("fmtMGRS(short) = %q, want 30U", got)
	}
	// A non-direction delimiter with a leading empty field takes findFormat's
	// split[0]=="" arm.
	if got := findFormat(",51.5074", ","); got == "" {
		t.Fatalf("findFormat(leading delimiter) = %q, want a detected format", got)
	}
}

// TestConvertCoordinatesUnknownOutputFormatPanics verifies the defensive panic
// for an output format the arg layer would never allow through.
func TestConvertCoordinatesUnknownOutputFormatPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an unhandled output format")
		}
	}()
	_, _ = convertCoordinates("51.5, -0.1", "Decimal Degrees", "Comma", "Bogus Format", "Comma", "None", 3)
}

// --- direct tests for the helpers extracted from convertCoordinates ---

// TestClampGridPrecision documents the grid-precision rounding: odd values round
// up, and the maximum is 10.
func TestClampGridPrecision(t *testing.T) {
	cases := map[int]int{0: 0, 1: 2, 2: 2, 5: 6, 10: 10, 11: 10, 20: 10}
	for in, want := range cases {
		if got := clampGridPrecision(in); got != want {
			t.Fatalf("clampGridPrecision(%d) = %d, want %d", in, got, want)
		}
	}
}

// TestApplyInputDirections documents the S/W sign negation for degree formats,
// including CyberChef's quirky "W only negates a positive value" precedence.
func TestApplyInputDirections(t *testing.T) {
	// Non-degree formats are untouched.
	if lat, lon := applyInputDirections("Geohash", "S W", 1, 2); lat != 1 || lon != 2 {
		t.Fatalf("non-degree changed values: %v %v", lat, lon)
	}
	// "S ... W" negates latitude (S) and longitude (W, since lon is positive).
	if lat, lon := applyInputDirections("Degrees Minutes Seconds", "51 S 0 W", 51, 0.5); lat != -51 || lon != -0.5 {
		t.Fatalf("got %v %v, want -51 -0.5", lat, lon)
	}
	// A single N marker leaves a positive latitude positive.
	if lat, _ := applyInputDirections("Decimal Degrees", "51 N", 51, 0); lat != 51 {
		t.Fatalf("N negated latitude: %v", lat)
	}
}

// TestAssembleDirectionalOutput documents direction placement and sign stripping.
func TestAssembleDirectionalOutput(t *testing.T) {
	// Before: markers precede each coordinate; the S/W signs are stripped.
	got := assembleDirectionalOutput("-51.5", "-0.1", "S", "W", "Before", ",", true)
	if got != "S 51.5,W 0.1," {
		t.Fatalf("Before pair: %q", got)
	}
	// None: no markers, signs retained, single value (not a pair).
	got = assembleDirectionalOutput("51.5", "", "N", "E", "None", ",", false)
	if got != "51.5," {
		t.Fatalf("None single: %q", got)
	}
}

// TestParseDegreeInput documents the shared degree parser, including the quirks
// it must preserve: DMS accepts >= 3 fields, and the others need an exact count.
func TestParseDegreeInput(t *testing.T) {
	id := func(f []float64) float64 { return f[0] }

	// Decimal Degrees pair: one field per coordinate.
	lat, lon, err := parseDegreeInput("Decimal Degrees", []string{"51.5", "0.1"}, "", true, 1, id)
	if err != nil || lat != 51.5 || lon != 0.1 {
		t.Fatalf("DD pair: %v %v %v", lat, lon, err)
	}
	// Too few fields errors, and the message names the format.
	if _, _, err := parseDegreeInput("Decimal Degrees", []string{"51.5", ""}, "", true, 1, id); err == nil ||
		!strings.Contains(err.Error(), "Decimal Degrees") {
		t.Fatalf("expected a Decimal Degrees field-count error, got %v", err)
	}
}

// TestTokeniseCoordInput documents the tokenising phase for degree vs grid input.
func TestTokeniseCoordInput(t *testing.T) {
	// Degree formats split into per-coordinate tokens; a pair is detected.
	in, split, isPair := tokeniseCoordInput("51.5,0.1", "Decimal Degrees", ",")
	if in != "51.5,0.1" || len(split) != 2 || !isPair {
		t.Fatalf("degree: in=%q split=%v pair=%v", in, split, isPair)
	}
	// Grid formats (coordNoChange) strip the delimiter and are treated as a pair.
	in, split, isPair = tokeniseCoordInput("ST 1234 5678", "Ordnance Survey National Grid", " ")
	if split != nil || !isPair {
		t.Fatalf("grid: split=%v pair=%v", split, isPair)
	}
	_ = in
}

// --- direct tests for the helpers extracted from findDirs ---

// TestDirFromValue documents mapping a signed coordinate string to its hemisphere
// letter (empty stays empty; non-numeric is treated as positive).
func TestDirFromValue(t *testing.T) {
	cases := []struct{ in, neg, pos, want string }{
		{"", "S", "N", ""},
		{"-5", "S", "N", "S"},
		{"5", "S", "N", "N"},
		{"-0.1", "W", "E", "W"},
		{"abc", "W", "E", "E"}, // unparseable -> positive
	}
	for _, c := range cases {
		if got := dirFromValue(c.in, c.neg, c.pos); got != c.want {
			t.Fatalf("dirFromValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSplitLatLong documents splitting the input into lat/long substrings.
func TestSplitLatLong(t *testing.T) {
	lat, long := splitLatLong("51.5,0.1", ",")
	if lat != "51.5" || long != "0.1" {
		t.Fatalf("pair: %q, %q", lat, long)
	}
	// No delimiter present: the whole string is the latitude.
	lat, long = splitLatLong("51.5", ",")
	if lat != "51.5" || long != "" {
		t.Fatalf("single: %q, %q", lat, long)
	}
}

// --- direct tests for the helpers extracted from findFormat ---

// TestFirstNonEmpty documents choosing the first non-empty split token.
func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty([]string{"a", "b"}); got != "a" {
		t.Fatalf("got %q", got)
	}
	if got := firstNonEmpty([]string{"", "b"}); got != "b" {
		t.Fatalf("got %q", got)
	}
}

// TestFindTestData documents extracting the first coordinate token to classify.
func TestFindTestData(t *testing.T) {
	if td, ok := findTestData("51.5,0.1", ","); !ok || td != "51.5" {
		t.Fatalf("pair: %q, %v", td, ok)
	}
	if td, ok := findTestData("51.5", ","); !ok || td != "51.5" { // no delim in input
		t.Fatalf("single: %q, %v", td, ok)
	}
	if _, ok := findTestData("51.5", ""); ok { // no delimiter -> no test data
		t.Fatal("empty delim should give no test data")
	}
}

// TestDetectGridFormat documents grid-format detection (returns "" for
// degree-style input, which contains °/'/").
func TestDetectGridFormat(t *testing.T) {
	if got := detectGridFormat("gcpvj0duq", ","); got != "Geohash" {
		t.Fatalf("geohash: %q", got)
	}
	if got := detectGridFormat("51° 30'", ","); got != "" {
		t.Fatalf("degree input should not be a grid: %q", got)
	}
}

// TestDetectDegreeFormat documents degree-format classification by field count.
func TestDetectDegreeFormat(t *testing.T) {
	cases := map[string]string{
		"51 30 26": "Degrees Minutes Seconds",
		"51 30":    "Degrees Decimal Minutes",
		"51.5":     "Decimal Degrees",
		"":         "",
	}
	for in, want := range cases {
		if got := detectDegreeFormat(in); got != want {
			t.Fatalf("detectDegreeFormat(%q) = %q, want %q", in, got, want)
		}
	}
}
