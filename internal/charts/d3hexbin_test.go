package charts

import (
	"math"
	"strconv"
	"testing"
)

// Points with no numeric position are skipped rather than binned.
func TestHexbinSkipsNonNumeric(t *testing.T) {
	h := NewHexbin(25)
	bins := h.Bin([]ScatterPoint{{X: math.NaN(), Y: 1}, {X: 1, Y: math.NaN()}, {X: 10, Y: 10}})
	if len(bins) != 1 {
		t.Fatalf("got %d bins, want 1", len(bins))
	}
	if len(bins[0].Points) != 1 {
		t.Errorf("bin holds %d points, want 1", len(bins[0].Points))
	}
}

// A grid of points binned against d3's own output. This exercises the
// boundary case where a point is nearer a neighbouring centre than the one it
// first rounds to, which is where d3's binning is least obvious.
func TestHexbinAgainstD3(t *testing.T) {
	var points []ScatterPoint
	for x := 0.0; x <= 20; x += 1.7 {
		for y := 0.0; y <= 20; y += 1.3 {
			points = append(points, ScatterPoint{X: x, Y: y})
		}
	}
	// Centre and count of every bin d3-hexbin produces for this grid at radius 3.
	want := map[string]int{
		"0.000000,0.000000": 5, "2.598076,4.500000": 12, "0.000000,9.000000": 8,
		"2.598076,13.500000": 12, "0.000000,18.000000": 7, "5.196152,0.000000": 7,
		"5.196152,9.000000": 11, "5.196152,18.000000": 10, "7.794229,4.500000": 10,
		"7.794229,13.500000": 10, "10.392305,0.000000": 7, "10.392305,9.000000": 11,
		"10.392305,18.000000": 10, "12.990381,4.500000": 10, "12.990381,13.500000": 11,
		"15.588457,0.000000": 7, "15.588457,9.000000": 10, "15.588457,18.000000": 10,
		"18.186533,4.500000": 8, "18.186533,13.500000": 8, "20.784610,0.000000": 2,
		"20.784610,9.000000": 3, "20.784610,18.000000": 3,
	}

	bins := NewHexbin(3).Bin(points)
	if len(bins) != len(want) {
		t.Fatalf("got %d bins, want %d", len(bins), len(want))
	}
	total := 0
	for _, b := range bins {
		key := strconv.FormatFloat(b.X, 'f', 6, 64) + "," + strconv.FormatFloat(b.Y, 'f', 6, 64)
		count, ok := want[key]
		if !ok {
			t.Errorf("unexpected bin at %s", key)
			continue
		}
		if len(b.Points) != count {
			t.Errorf("bin %s holds %d points, want %d", key, len(b.Points), count)
		}
		total += len(b.Points)
	}
	if total != len(points) {
		t.Errorf("binned %d of %d points", total, len(points))
	}
}

// The hexagon path is a closed six-segment outline.
func TestHexagonPath(t *testing.T) {
	got := NewHexbin(25).HexagonPath(15)
	if got[:1] != "m" || got[len(got)-1:] != "z" {
		t.Errorf("path = %q, want it to start with m and end with z", got)
	}
	segments := 0
	for _, c := range got {
		if c == 'l' {
			segments++
		}
	}
	if segments != 5 {
		t.Errorf("path has %d line segments, want 5 after the initial move", segments)
	}
}
