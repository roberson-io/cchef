package charts

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

// lineVector is one recorded case: a set of points and the path d3 draws
// through them.
//
// The vectors are d3's own output, produced by running d3.line() with
// curveMonotoneX under Node, which is what CyberChef's entropy line and curve
// views draw with. Where d3 reports no path at all it returns null, which is
// recorded here as an empty string: setting an attribute to either leaves the
// element without one.
type lineVector struct {
	Name   string       `json:"name"`
	Points [][2]float64 `json:"points"`
	Want   string       `json:"want"`
}

// TestD3LineMonotone covers the curve against those vectors, over the cases its
// slope rule turns on: a straight run, a peak, a valley, a flat stretch, points
// sharing a position, and values that need rounding.
func TestD3LineMonotone(t *testing.T) {
	file, err := os.Open("testdata/d3_line_monotone.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var v lineVector
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("parse vector: %v", err)
		}

		t.Run(v.Name, func(t *testing.T) {
			points := make([]Point, len(v.Points))
			for i, p := range v.Points {
				points[i] = Point{p[0], p[1]}
			}
			if got := LineMonotoneX(points); got != v.Want {
				t.Errorf("path differs\n got %q\nwant %q", got, v.Want)
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
}
