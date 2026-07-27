package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// freqGolden is one recorded case: the input, the two flags, and what CyberChef
// presents for them.
//
// CyberChef ships no case for this operation, so the goldens are its own output
// produced by running the real operation under Node, with the two browser-only
// parts removed: the empty <canvas> and the script that draws into it after
// measuring the page it sits on.
type freqGolden struct {
	Name       string `json:"name"`
	Input      string `json:"input"` // hexadecimal, so any byte may appear
	ShowZeroes bool   `json:"showZeroes"`
	ShowASCII  bool   `json:"showAscii"`
	Want       string `json:"want"`
}

// TestFrequencyDistributionGolden replays those cases.
func TestFrequencyDistributionGolden(t *testing.T) {
	file, err := os.Open("testdata/frequency_distribution.jsonl")
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c freqGolden
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("parse golden: %v", err)
		}

		t.Run(c.Name, func(t *testing.T) {
			input, err := hex.DecodeString(c.Input)
			if err != nil {
				t.Fatalf("decode input: %v", err)
			}
			out, err := core.Recipe{{
				Op:   "Frequency distribution",
				Args: []any{c.ShowZeroes, c.ShowASCII},
			}}.Execute(core.NewDish(input, core.TypeArrayBuffer))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != c.Want {
				t.Errorf("output differs\n got %.200q\nwant %.200q", out.String(), c.Want)
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read goldens: %v", err)
	}
}

// TestFrequencyDistributionEmptyInput covers input with no bytes, where every
// percentage would divide by zero.
func TestFrequencyDistributionEmptyInput(t *testing.T) {
	out, err := core.Recipe{{
		Op:   "Frequency distribution",
		Args: []any{false, true},
	}}.Execute(core.NewDish(nil, core.TypeArrayBuffer))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{
		"Total data length: 0",
		"Number of bytes represented: 0",
		"Number of bytes not represented: 256",
	} {
		if !containsLine(out.String(), want) {
			t.Errorf("output does not report %q", want)
		}
	}
}

// containsLine reports whether the text holds the given line.
func containsLine(text, line string) bool {
	for _, l := range strings.Split(text, "\n") {
		if l == line {
			return true
		}
	}
	return false
}
