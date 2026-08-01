package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestEntropyShannonScale covers the Shannon entropy against the oracle. The
// figure runs from zero, where every byte is the same, to eight, where they are
// spread evenly.
//
// CyberChef draws a scale bar beside the figure, which needs a browser: the
// canvas is sized from the page it sits on and drawn by a script. The figure it
// annotates is the whole of the portable output.
func TestEntropyShannonScale(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"a single byte, which carries no information", "a", "0"},
		{"the same byte repeated", "aaaa", "0"},
		{"two bytes in equal measure", "ab", "1"},
		{"a short phrase", "Hello world!", "3.0220552088742005"},
		{"a pangram", "The quick brown fox jumps over the lazy dog", "4.431965045349459"},
		{"the ten digits", "0123456789", "3.321928094887362"},
		{"two byte values in equal measure", "AAAAAAAAAABBBBBBBBBB", "1"},
		{"every printable character once", printableRange(), "6.569855608330948"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := core.Recipe{{Op: "Entropy", Args: []any{"Shannon scale"}}}.
				Execute(core.NewDish([]byte(tc.input), core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if want := "Shannon entropy: " + tc.want; out.String() != want {
				t.Errorf("got %q, want %q", out.String(), want)
			}
		})
	}
}

// TestEntropyEmptyInput covers input with no bytes, where there is nothing to
// measure the spread of.
func TestEntropyEmptyInput(t *testing.T) {
	out, err := core.Recipe{{Op: "Entropy", Args: []any{"Shannon scale"}}}.
		Execute(core.NewDish(nil, core.TypeArrayBuffer))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "Shannon entropy: 0" {
		t.Errorf("got %q, want %q", out.String(), "Shannon entropy: 0")
	}
}

// printableRange is every printable character, once each.
func printableRange() string {
	out := make([]byte, 0, 95)
	for b := byte(32); b < 127; b++ {
		out = append(out, b)
	}
	return string(out)
}

// entropyChartGolden is one recorded chart: the input, which visualisation, and
// the SVG CyberChef draws for it.
//
// The goldens are CyberChef's own output, produced by running the real
// operation under Node with the three substitutions cchef's SVG writer applies
// deliberately (see svgbuild.go): D3's `__data__` bindings dropped, since nodom
// leaks them carrying unescaped input; the trailing "; " removed from style
// attributes; and the SVG namespace added, without which the result is not
// recognised as SVG on its own.
type entropyChartGolden struct {
	Name  string `json:"name"`
	Input string `json:"input"` // hexadecimal, so any byte may appear
	View  string `json:"view"`
	Want  string `json:"want"`
}

// TestEntropyChartsGolden replays those cases.
func TestEntropyChartsGolden(t *testing.T) {
	file, err := os.Open("testdata/entropy_charts.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c entropyChartGolden
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("parse golden: %v", err)
		}

		t.Run(c.Name, func(t *testing.T) {
			input, err := hex.DecodeString(c.Input)
			if err != nil {
				t.Fatalf("decode input: %v", err)
			}
			out, err := core.Recipe{{Op: "Entropy", Args: []any{c.View}}}.
				Execute(core.NewDish(input, core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if got := out.String(); got != c.Want {
				t.Errorf("chart differs at %s", firstDifference(got, c.Want))
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read goldens: %v", err)
	}
}

// firstDifference locates where two charts part company, which is more use than
// either in full.
func firstDifference(got, want string) string {
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			lo := max(i-50, 0)
			return fmt.Sprintf("byte %d:\n ...%s\n ...%s", i,
				got[lo:min(i+50, len(got))], want[lo:min(i+50, len(want))])
		}
	}
	return fmt.Sprintf("length: got %d, want %d", len(got), len(want))
}

// TestEntropyUnsupportedVisualisation covers the guard on the visualisation
// name. The recipe engine checks the option before the operation runs, so the
// guard only answers a direct call.
func TestEntropyUnsupportedVisualisation(t *testing.T) {
	if _, err := (Entropy{}).Run(
		core.NewDish([]byte("Hello"), core.TypeArrayBuffer),
		[]any{"Sunburst"},
	); err == nil {
		t.Error("accepted a visualisation the operation does not offer")
	}
}

// TestEntropyExtentOfNothing covers the spread of no values at all, which the
// bar histogram asks for when the input holds no bytes.
func TestEntropyExtentOfNothing(t *testing.T) {
	lowest, highest := entropyExtent(nil)
	if !math.IsNaN(lowest) || !math.IsNaN(highest) {
		t.Errorf("got %v and %v, want both to be no number at all", lowest, highest)
	}
}
