package geocoord

import (
	"math"
	"strings"
	"testing"
)

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
	_, _ = Convert("51.5, -0.1", "Decimal Degrees", "Comma", "Bogus Format", "Comma", "None", 3)
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

// Geohash results verified against the CyberChef-server oracle. These pin the
// cases where a geohash library that rounds a coordinate sitting exactly on a
// cell boundary the other way gives a different, equally valid, answer.
func TestGeohashEncode(t *testing.T) {
	cases := []struct {
		lat, lon  float64
		precision int
		want      string
	}{
		{0, 0, 8, "7zzzzzzz"},
		{90, 180, 8, "zzzzzzzz"},
		{-90, -180, 8, "00000000"},
		{-45, 45, 8, "hzzzzzzz"},
		{51.5074, -0.1278, 9, "gcpvj0duq"},
		{37.7749, -122.4194, 12, "9q8yyk8ytpxr"},
		{0, 0, 1, "7"},
		{0.0001, -0.0001, 5, "ebpbp"},
		{51.5074, -0.1278, 0, ""},
	}
	for _, c := range cases {
		if got := geohashEncode(c.lat, c.lon, c.precision); got != c.want {
			t.Errorf("encode(%v, %v, %d) = %q, want %q", c.lat, c.lon, c.precision, got, c.want)
		}
	}
}

// Decoding transcribed from ngeohash. A character outside the geohash alphabet
// contributes five zero bits, exactly as a "0" would.
func TestGeohashDecodeCenter(t *testing.T) {
	cases := []struct {
		hash     string
		lat, lon float64
	}{
		{"ezs42", 42.60498046875, -5.60302734375},
		{"EZS42", 42.60498046875, -5.60302734375},
		{"s", 22.5, 22.5},
		{"sv", 30.9375, 39.375},
		{"0", -67.5, -157.5},
		{"z", 67.5, 157.5},
		{"zzzzzzzzzzzz", 89.99999991618097, 179.99999983236194},
		{"00000000", -89.99991416931152, -179.99982833862305},
		{"ezs4a2", 42.54180908203125, -5.6085205078125},
		{"aaaa", -89.912109375, -179.82421875},
		{"ail", -89.296875, -179.296875},
		{"u4pruydqqvj", 57.64911063015461, 10.407439693808556},
		{"dr5ru7c02wh", 40.757979825139046, -73.99151913821697},
		{"bbe", 48.515625, -141.328125},
		{"e3m", 7.734375, -26.015625},
		{"", 0, 0},
	}
	for _, c := range cases {
		lat, lon := geohashDecodeCenter(c.hash)
		if lat != c.lat || lon != c.lon {
			t.Errorf("decode(%q) = %v, %v, want %v, %v", c.hash, lat, lon, c.lat, c.lon)
		}
	}
}

// Ordnance Survey National Grid conversions transcribed from geodesy, the
// library CyberChef uses. Both directions pass through a Helmert transform
// between the OSGB36 and WGS84 datums, so a coordinate is not merely
// reprojected but moved onto a different reference ellipsoid.
func TestOSGBGridToLatLon(t *testing.T) {
	const tol = 1e-9
	cases := []struct {
		e, n     float64
		lat, lon float64
	}{
		{651409.903, 313177.270, 52.65797859822991, 1.7160519457128236},
		{216650, 771250, 56.796557296939675, -5.003930468291132},
		{530000, 180000, 51.503990826218306, -0.12835397852590702},
		{400000, -100000, 49.00077077964187, -2.001307500682245},
		{100000, 0, 49.825069446807106, -6.172738505443476},
		{500000, 1000000, 58.87574487611452, -0.2673278941823114},
		{438700, 114800, 50.93135805700044, -1.4506774292622975},
	}
	for _, c := range cases {
		lat, lon := osgbGridToLatLon(c.e, c.n)
		if math.Abs(lat-c.lat) > tol || math.Abs(lon-c.lon) > tol {
			t.Errorf("grid (%v, %v) = %v, %v; want %v, %v", c.e, c.n, lat, lon, c.lat, c.lon)
		}
	}
}

// The reverse direction, likewise from geodesy. Easting and northing are
// rounded to the millimetre, as the reference implementation does.
func TestOSGBLatLonToGrid(t *testing.T) {
	const tol = 1e-6
	cases := []struct {
		lat, lon float64
		e, n     float64
	}{
		{52.65798, 1.71605, 651409.760, 313177.419},
		{51.5074, -0.1278, 530028.746, 180380.095},
		{56.796089, -5.004712, 216599.999, 771200.087},
		{58.0, -3.0, 340989.675, 901635.930},
		{50.0, -5.5, 149280.979, 16965.082},
		{49.0, -2.0, 400095.632, -100085.687},
	}
	for _, c := range cases {
		e, n := osgbLatLonToGrid(c.lat, c.lon)
		if math.Abs(e-c.e) > tol || math.Abs(n-c.n) > tol {
			t.Errorf("lat/lon (%v, %v) = %v, %v; want %v, %v", c.lat, c.lon, e, n, c.e, c.n)
		}
	}
}
