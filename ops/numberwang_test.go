package ops

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// numberwangFact separates the verdict from the fact appended to it.
const numberwangFact = "\n\nDid you know: "

// numberwangCase is one recorded verdict. CyberChef ships no fixtures for this
// operation, so the verdicts come from the oracle. Only the part before the
// fact is recorded, since the fact is picked at random. The input is
// hexadecimal so anything outside ASCII survives.
type numberwangCase struct {
	Input string `json:"input"`
	Want  string `json:"want"`
}

// splitNumberwang divides the output into the verdict and the fact.
func splitNumberwang(t *testing.T, out string) (verdict, fact string) {
	t.Helper()
	verdict, fact, found := strings.Cut(out, numberwangFact)
	if !found {
		t.Fatalf("no fact in %q", out)
	}
	return verdict, fact
}

// TestNumberwangVerdicts covers what the operation makes of its input: whether
// it finds a number, and whether a letter follows one.
func TestNumberwangVerdicts(t *testing.T) {
	file, err := os.Open("testdata/numberwang.jsonl")
	if err != nil {
		t.Fatalf("open cases: %v", err)
	}
	defer func() { _ = file.Close() }()

	seen := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c numberwangCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("parse case: %v", err)
		}
		input := string(mustHex(t, c.Input))
		seen++

		t.Run(c.Input, func(t *testing.T) {
			out, err := runOp(t, "Numberwang", input)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if verdict, _ := splitNumberwang(t, out); verdict != c.Want {
				t.Errorf("for %q got %q, want %q", input, verdict, c.Want)
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read cases: %v", err)
	}
	if seen == 0 {
		t.Fatal("no cases were read")
	}
}

// TestNumberwangEmptyInput covers input with nothing in it, which gets the
// other game entirely. The oracle will not take an empty request body, so this
// one comes from the operation's own source.
func TestNumberwangEmptyInput(t *testing.T) {
	out, err := runOp(t, "Numberwang", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if verdict, _ := splitNumberwang(t, out); verdict != "Let's play Wangernumb!" {
		t.Errorf("got %q, want %q", verdict, "Let's play Wangernumb!")
	}
}

// TestNumberwangFactsAreFromTheList covers the fact appended to every verdict:
// it is one of those the operation knows, and over enough runs it is not always
// the same one.
func TestNumberwangFactsAreFromTheList(t *testing.T) {
	known := map[string]bool{}
	for _, fact := range numberwangFacts {
		known[fact] = true
	}
	if len(known) != len(numberwangFacts) {
		t.Errorf("the list holds %d facts but only %d are distinct",
			len(numberwangFacts), len(known))
	}

	seen := map[string]bool{}
	for range 500 {
		out, err := runOp(t, "Numberwang", "42")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		_, fact := splitNumberwang(t, out)
		if !known[fact] {
			t.Fatalf("fact %q is not one the operation knows", fact)
		}
		seen[fact] = true
	}
	// Five hundred draws from thirty-eight facts leaving any unseen would be
	// remarkable; picking only one would mean no choice is being made.
	if len(seen) != len(numberwangFacts) {
		t.Errorf("saw %d of %d facts over 500 runs", len(seen), len(numberwangFacts))
	}
}
