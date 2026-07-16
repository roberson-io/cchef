package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Offset checker output verified against the CyberChef-server oracle.
func TestOffsetChecker(t *testing.T) {
	runCases(t, []opCase{
		{
			"common prefix", "hello world\nhello there",
			"<span class='hl5'>hello </span>world\n<span class='hl5'>hello </span>there",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		// Varied-length samples exercise the span-close paths when a sample runs
		// out mid-match or the match reaches a sample's end. The doubled </span>
		// is a CyberChef quirk this port reproduces (oracle-verified).
		{
			"second sample shorter", "ABCDEF\nABC",
			"<span class='hl5'>ABC</span>DEF\n<span class='hl5'>ABC</span></span>",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		{
			"identical samples match through end", "match\nmatch",
			"<span class='hl5'>match</span></span>\n<span class='hl5'>match</span></span>",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
		{
			"first sample shorter", "ABC\nABCDEF",
			"<span class='hl5'>ABC</span></span>\n<span class='hl5'>ABC</span>DEF",
			core.Recipe{{Op: "Offset checker", Args: []any{`\n`}}},
		},
	})

	// Fewer than two samples cannot be compared.
	if _, err := runOp(t, "Offset checker", "onlyone", `\n`); err == nil {
		t.Error("single sample: expected error")
	}
}

// TestOffsetCheckerTrailingMatch covers the span-close when a match reaches the
// end of a shorter sample. The CyberChef-server oracle rejects these inputs, but
// the output matches CyberChef's OffsetChecker.mjs algorithm exactly (including
// its quirky trailing "</span>").
func TestOffsetCheckerTrailingMatch(t *testing.T) {
	out, err := runOp(t, "Offset checker", "abc\nxb", `\n`)
	if err != nil {
		t.Fatal(err)
	}
	want := "a<span class='hl5'>b</span>c</span>\nx<span class='hl5'>b</span></span>"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

// --- direct tests for the helpers extracted from OffsetChecker.Run ---

// TestOffsetAllMatch documents whether every sample shares s0's char at offset i.
func TestOffsetAllMatch(t *testing.T) {
	samples := [][]rune{[]rune("abc"), []rune("axc"), []rune("ayc")}
	if !offsetAllMatch(samples, 0, 'a') { // all start with 'a'
		t.Fatal("offset 0 should match")
	}
	if offsetAllMatch(samples, 1, 'b') { // 'b' vs 'x'/'y'
		t.Fatal("offset 1 should not match")
	}
	// A sample shorter than i does not match.
	short := [][]rune{[]rune("ab"), []rune("a")}
	if offsetAllMatch(short, 1, 'b') {
		t.Fatal("a too-short sample should fail the match")
	}
}

// TestWriteOffsetSample documents the span-writing state transitions: opening a
// highlight, and closing it, updating inMatch only on the last sample.
func TestWriteOffsetSample(t *testing.T) {
	samples := [][]rune{[]rune("ab"), []rune("ax")}
	s0 := samples[0]
	n := 2

	// Opening a match on the last sample sets inMatch=true and opens a span.
	var out strings.Builder
	got := writeOffsetSample(&out, samples, s0, 1, 0, n, true, false)
	if !got || out.String() != "<span class='hl5'>a" {
		t.Fatalf("open: inMatch=%v out=%q", got, out.String())
	}
	// A non-last sample opens its span but leaves inMatch unchanged (false).
	var out2 strings.Builder
	got = writeOffsetSample(&out2, samples, s0, 0, 0, n, true, false)
	if got || out2.String() != "<span class='hl5'>a" {
		t.Fatalf("open non-last: inMatch=%v out=%q", got, out2.String())
	}
	// Closing (match=false while inMatch) on the last sample resets inMatch.
	var out3 strings.Builder
	got = writeOffsetSample(&out3, samples, s0, 1, 1, n, false, true)
	if got || !strings.HasPrefix(out3.String(), "</span>") {
		t.Fatalf("close: inMatch=%v out=%q", got, out3.String())
	}
}

// TestWriteOffsetTail documents the final-offset tail: close any open span and
// append the remainder of a longer sample.
func TestWriteOffsetTail(t *testing.T) {
	var out strings.Builder
	writeOffsetTail(&out, []rune("abc"), 1, true)
	if out.String() != "</span>c" {
		t.Fatalf("open tail: %q", out.String())
	}
	var out2 strings.Builder
	writeOffsetTail(&out2, []rune("ab"), 1, false)
	if out2.String() != "" {
		t.Fatalf("empty tail: %q", out2.String())
	}
}
