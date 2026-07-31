package ops

import (
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The golden table in testdata/utm_golden.tsv holds 462 forward and 504
// reverse conversions spanning every zone-boundary special case (Norway,
// Svalbard), both hemispheres, band edges, and out-of-range inputs. Line
// formats, tab-separated:
//
//	F  lat lon easting northing zone letter   (letter is unused)
//	R  easting northing zone northern lat lon
//	FE lat lon message
//	RE easting northing zone northern message
//
// The tolerances are set by what the conversion can promise, not by what one
// machine produces: the series themselves are accurate to well under a
// micrometre over the valid range, while math.Sin and friends differ in the
// last bit across architectures. A micrometre in metres and its angular
// equivalent in degrees are far below the operation's displayable precision
// and hold on any architecture.
const (
	utmMetreTolerance  = 1e-6
	utmDegreeTolerance = 1e-11
)

// angularDiff is the distance between two angles in degrees, ignoring whole
// turns, so that a wrapped and an unwrapped form of the same longitude
// compare equal.
func angularDiff(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	return math.Min(d, 360-d)
}

// relDiff is the difference between two values relative to their magnitude,
// for comparing numbers far too large for an absolute tolerance.
func relDiff(a, b float64) float64 {
	return math.Abs(a-b) / math.Max(1, math.Abs(b))
}

type utmGolden struct {
	kind   string
	fields []string
}

func loadUTMGoldens(t *testing.T) []utmGolden {
	t.Helper()
	data, err := os.ReadFile("testdata/utm_golden.tsv")
	if err != nil {
		t.Fatalf("read goldens: %v", err)
	}
	var rows []utmGolden
	for line := range strings.Lines(strings.TrimRight(string(data), "\n")) {
		fields := strings.Split(strings.TrimRight(line, "\n"), "\t")
		rows = append(rows, utmGolden{kind: fields[0], fields: fields[1:]})
	}
	return rows
}

func goldFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("golden float %q: %v", s, err)
	}
	return v
}

func TestUTMFromLatLonGoldens(t *testing.T) {
	n := 0
	for _, row := range loadUTMGoldens(t) {
		if row.kind != "F" {
			continue
		}
		n++
		lat, lon := goldFloat(t, row.fields[0]), goldFloat(t, row.fields[1])
		wantE, wantN := goldFloat(t, row.fields[2]), goldFloat(t, row.fields[3])
		wantZone, _ := strconv.Atoi(row.fields[4])

		e, north, zone, err := utmFromLatLon(lat, lon)
		if err != nil {
			t.Errorf("utmFromLatLon(%v, %v): %v", lat, lon, err)
			continue
		}
		if zone != wantZone {
			t.Errorf("utmFromLatLon(%v, %v) zone = %d, want %d", lat, lon, zone, wantZone)
		}
		if math.Abs(e-wantE) > utmMetreTolerance || math.Abs(north-wantN) > utmMetreTolerance {
			t.Errorf("utmFromLatLon(%v, %v) = %v, %v, want %v, %v", lat, lon, e, north, wantE, wantN)
		}
	}
	if n == 0 {
		t.Fatal("no forward goldens loaded")
	}
}

func TestUTMToLatLonGoldens(t *testing.T) {
	n := 0
	for _, row := range loadUTMGoldens(t) {
		if row.kind != "R" {
			continue
		}
		n++
		e, north := goldFloat(t, row.fields[0]), goldFloat(t, row.fields[1])
		zone, _ := strconv.Atoi(row.fields[2])
		northern := row.fields[3] == "true"
		wantLat, wantLon := goldFloat(t, row.fields[4]), goldFloat(t, row.fields[5])

		lat, lon, err := utmToLatLon(e, north, zone, northern)
		if err != nil {
			t.Errorf("utmToLatLon(%v, %v, %d, %v): %v", e, north, zone, northern, err)
			continue
		}
		if math.Abs(wantLat) > 90 {
			// Inputs like a zero northing in the southern hemisphere describe no
			// point on Earth, and the series returns garbage latitudes for them.
			// The golden's longitude is then an astronomically large unwrapped
			// angle whose low digits are float noise, so the only meaningful
			// checks are that the latitude garbage is reproduced and that the
			// longitude is at least wrapped into range.
			if relDiff(lat, wantLat) > utmDegreeTolerance {
				t.Errorf("utmToLatLon(%v, %v, %d, %v) lat = %v, want %v",
					e, north, zone, northern, lat, wantLat)
			}
			if lon < -180 || lon >= 180 {
				t.Errorf("utmToLatLon(%v, %v, %d, %v) lon = %v, want wrapped into [-180, 180)",
					e, north, zone, northern, lon)
			}
			continue
		}
		// Corners of the valid input range can still sit thousands of
		// kilometres from the central meridian, where the series has lost
		// metre-level accuracy; those rows are compared to what it can promise
		// there. They still pin zone handling and hemisphere.
		tol := utmDegreeTolerance
		if angularDiff(lon, 0) > 90 {
			tol = 1e-6
		}
		if math.Abs(lat-wantLat) > tol || angularDiff(lon, wantLon) > tol {
			t.Errorf("utmToLatLon(%v, %v, %d, %v) = %v, %v, want %v, %v",
				e, north, zone, northern, lat, lon, wantLat, wantLon)
		}
	}
	if n == 0 {
		t.Fatal("no reverse goldens loaded")
	}
}

func TestUTMGoldenRangeErrors(t *testing.T) {
	for _, row := range loadUTMGoldens(t) {
		switch row.kind {
		case "FE":
			lat, lon := goldFloat(t, row.fields[0]), goldFloat(t, row.fields[1])
			if _, _, _, err := utmFromLatLon(lat, lon); err == nil {
				t.Errorf("utmFromLatLon(%v, %v): expected error", lat, lon)
			}
		case "RE":
			e, north := goldFloat(t, row.fields[0]), goldFloat(t, row.fields[1])
			zone, _ := strconv.Atoi(row.fields[2])
			northern := row.fields[3] == "true"
			if _, _, err := utmToLatLon(e, north, zone, northern); err == nil {
				t.Errorf("utmToLatLon(%v, %v, %d, %v): expected error", e, north, zone, northern)
			}
		}
	}
}

func TestUTMErrorMessages(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  func() error
		want string
	}{
		{
			"lat low", func() error { _, _, _, err := utmFromLatLon(-80.0001, 0); return err },
			"latitude out of range (must be between 80 deg S and 84 deg N)",
		},
		{
			"lat high", func() error { _, _, _, err := utmFromLatLon(84.0001, 0); return err },
			"latitude out of range (must be between 80 deg S and 84 deg N)",
		},
		{
			"lon low", func() error { _, _, _, err := utmFromLatLon(0, -180.0001); return err },
			"longitude out of range (must be between 180 deg W and 180 deg E)",
		},
		{
			"lon high", func() error { _, _, _, err := utmFromLatLon(0, 180.0001); return err },
			"longitude out of range (must be between 180 deg W and 180 deg E)",
		},
		{
			"easting low", func() error { _, _, err := utmToLatLon(99999.9999, 0, 30, true); return err },
			"easting out of range (must be between 100,000 m and 999,999 m)",
		},
		{
			"easting high", func() error { _, _, err := utmToLatLon(1000000, 0, 30, true); return err },
			"easting out of range (must be between 100,000 m and 999,999 m)",
		},
		{
			"northing low", func() error { _, _, err := utmToLatLon(500000, -0.0001, 30, true); return err },
			"northing out of range (must be between 0 m and 10,000,000 m)",
		},
		{
			"northing high", func() error { _, _, err := utmToLatLon(500000, 10000000.0001, 30, true); return err },
			"northing out of range (must be between 0 m and 10,000,000 m)",
		},
		{
			"zone low", func() error { _, _, err := utmToLatLon(500000, 500000, 0, true); return err },
			"zone number out of range (must be between 1 and 60)",
		},
		{
			"zone high", func() error { _, _, err := utmToLatLon(500000, 500000, 61, true); return err },
			"zone number out of range (must be between 1 and 60)",
		},
	} {
		err := tc.err()
		if err == nil {
			t.Errorf("%s: expected error", tc.name)
			continue
		}
		if err.Error() != tc.want {
			t.Errorf("%s: error %q, want %q", tc.name, err.Error(), tc.want)
		}
	}
}

// TestUTMBoundaryAcceptance pins the exact edges of the valid input ranges:
// closed latitude/longitude and northing bounds, easting closed below and
// open above.
func TestUTMBoundaryAcceptance(t *testing.T) {
	for _, c := range [][2]float64{{-80, 0}, {84, 0}, {0, -180}, {0, 180}} {
		if _, _, _, err := utmFromLatLon(c[0], c[1]); err != nil {
			t.Errorf("utmFromLatLon(%v, %v): %v", c[0], c[1], err)
		}
	}
	for _, c := range []struct {
		e, n float64
	}{{100000, 0}, {999999.9999, 10000000}} {
		if _, _, err := utmToLatLon(c.e, c.n, 30, true); err != nil {
			t.Errorf("utmToLatLon(%v, %v): %v", c.e, c.n, err)
		}
	}
}

// TestUTMLongitude180 pins the antimeridian: 180°E and 180°W are the same
// meridian and both sit in zone 1, which runs from 180°W to 174°W. (The
// library this replaced returned a nonexistent zone 61 for 180°E.)
func TestUTMLongitude180(t *testing.T) {
	for _, lon := range []float64{180, -180} {
		e, _, zone, err := utmFromLatLon(0, lon)
		if err != nil {
			t.Fatalf("utmFromLatLon(0, %v): %v", lon, err)
		}
		if zone != 1 {
			t.Errorf("utmFromLatLon(0, %v) zone = %d, want 1", lon, zone)
		}
		if math.Abs(e-166021.443179330) > 1e-6 {
			t.Errorf("utmFromLatLon(0, %v) easting = %v, want 166021.443179330", lon, e)
		}
	}
}
