package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// CyberChef's Magic fixtures assert against the HTML table it renders in a
// browser, matching things like `#recipe=From_Hex('Space')"`. cchef reports the
// same findings as text, so each case is transcribed as the behaviour the
// fixture is really checking: which recipe Magic suggests, and where it ranks.

// runMagic runs Magic and returns its report.
func runMagic(t *testing.T, input string, depth float64, intensive, extensive bool, crib string) string {
	t.Helper()
	out, err := runOp(t, "Magic", input, depth, intensive, extensive, crib)
	if err != nil {
		t.Fatalf("Magic: %v", err)
	}
	return out
}

// magicRecipes lists the suggested recipes in the order they were reported.
// A recipe is the run of unindented lines following a "Recipe:" heading.
func magicRecipes(report string) []string {
	var out []string
	var current []string
	collecting := false
	for line := range strings.SplitSeq(report, "\n") {
		switch {
		case line == "Recipe:":
			if collecting {
				out = append(out, strings.Join(current, "\n"))
			}
			collecting, current = true, nil
		case collecting && strings.HasPrefix(line, "  "):
			out = append(out, strings.Join(current, "\n"))
			collecting, current = false, nil
		case collecting && line != "":
			current = append(current, line)
		}
	}
	if collecting {
		out = append(out, strings.Join(current, "\n"))
	}
	return out
}

// TestMagicNothing covers "Magic: nothing": empty input has nothing to report.
func TestMagicNothing(t *testing.T) {
	got := runMagic(t, "", 3, false, false, "")
	if got != magicNothingFound {
		t.Errorf("got %q, want the nothing-found message", got)
	}
}

// TestMagicHexRanking covers "Magic: hex, correct rank": From Hex is suggested,
// and ranked first.
func TestMagicHexRanking(t *testing.T) {
	recipes := magicRecipes(runMagic(t, "41 42 43 44 45", 3, false, false, ""))
	if len(recipes) == 0 {
		t.Fatal("no recipes suggested")
	}
	if recipes[0] != "From_Hex('Space')" {
		t.Errorf("first suggestion is %q, want From_Hex('Space')", recipes[0])
	}
}

// TestMagicJPEG covers "Magic: jpeg render": a JPEG is recognised, and the
// operation that displays it is suggested.
func TestMagicJPEG(t *testing.T) {
	raw, err := os.ReadFile("testdata/magic_jpeg.bin")
	if err != nil {
		t.Skipf("sample missing: %v", err)
	}
	report := runMagic(t, string(raw), 3, false, false, "")
	if !strings.Contains(report, "Render_Image('Raw')") {
		t.Errorf("Render Image was not suggested:\n%s", firstLines(report, 12))
	}
	if !strings.Contains(report, "image/jpeg") {
		t.Errorf("the file type was not reported:\n%s", firstLines(report, 12))
	}
}

// TestMagicChainBase64 covers "Magic Chain: Base64": three rounds of Base64
// are seen through, and the text underneath is reported.
func TestMagicChainBase64(t *testing.T) {
	report := runMagic(t, "WkVkV2VtUkRRbnBrU0Vwd1ltMWpQUT09", 3, false, false, "")
	want := "From_Base64('A-Za-z0-9+/=',true,false)\n" +
		"From_Base64('A-Za-z0-9+/=',true,false)\n" +
		"From_Base64('A-Za-z0-9+/=',true,false)"
	if !strings.Contains(report, want) {
		t.Errorf("the three-round recipe was not suggested:\n%s", firstLines(report, 20))
	}
	// "Magic Chain: Base64 output" checks the decoded text is shown.
	if !strings.Contains(report, "test string") {
		t.Errorf("the decoded text was not reported:\n%s", firstLines(report, 20))
	}
}

// TestMagicChainHexdump covers "Magic Chain: Hex -> Hexdump -> Base64".
func TestMagicChainHexdump(t *testing.T) {
	const input = "MDAwMDAwMDAgIDM3IDM0IDIwIDM2IDM1IDIwIDM3IDMzIDIwIDM3IDM0IDIwIDMyIDMwIDIwIDM3ICB8NzQgNjUgNzMgNzQgMjAgN3wKMDAwMDAwMTAgIDMzIDIwIDM3IDM0IDIwIDM3IDMyIDIwIDM2IDM5IDIwIDM2IDY1IDIwIDM2IDM3ICB8MyA3NCA3MiA2OSA2ZSA2N3w="
	report := runMagic(t, input, 3, false, false, "")
	want := "From_Base64('A-Za-z0-9+/=',true,false)\nFrom_Hexdump()\nFrom_Hex('Space')"
	if !strings.Contains(report, want) {
		t.Errorf("the hexdump chain was not suggested:\n%s", firstLines(report, 20))
	}
}

// TestMagicChainBase32 covers "Magic Chain: Charcode -> Octal -> Base32".
func TestMagicChainBase32(t *testing.T) {
	const input = "GY3SANRUEA2DAIBWGYQDMNJAGQYCANRXEA3DGIBUGAQDMNZAGY2CANBQEA3DEIBWGAQDIMBAGY3SANRTEA2DAIBWG4QDMNBAGQYCANRXEA3DEIBUGAQDMNRAG4YSANBQEA3DMIBRGQ2SANBQEA3DMIBWG4======"
	report := runMagic(t, input, 3, false, false, "")
	want := "From_Base32('A-Z2-7=',false)\nFrom_Octal('Space')\nFrom_Hex('Space')"
	if !strings.Contains(report, want) {
		t.Errorf("the base32 chain was not suggested:\n%s", firstLines(report, 20))
	}
}

// TestMagicChainDecimal covers "Magic Chain: Decimal -> Base32 -> Base32",
// which asserts only that the text underneath is found.
func TestMagicChainDecimal(t *testing.T) {
	const input = "I5CVSVCNJFBFER2BLFJUCTKKKJDVKUKEINGUUV2FIFNFIRKJIJJEORJSKNAU2SSSI5MVCRCDJVFFKRKBLFKECTSKIFDUKWKUIFEUEUSHIFNFCPJ5HU6Q===="
	report := runMagic(t, input, 3, false, false, "")
	if !strings.Contains(report, "test string") {
		t.Errorf("the decoded text was not reported:\n%s", firstLines(report, 20))
	}
}

// TestMagicDefangIP covers the two Defang IP cases: a valid address is
// recognised, and something that only looks like one is not.
func TestMagicDefangIP(t *testing.T) {
	valid := runMagic(t, "192.168.0.1", 1, false, false, "")
	if !strings.Contains(valid, "Defang_IP_Addresses()") {
		t.Errorf("a valid address was not recognised:\n%s", firstLines(valid, 12))
	}
	invalid := runMagic(t, "192.168.0.1.0", 1, false, false, "")
	if strings.Contains(invalid, "Defang_IP_Addresses") {
		t.Errorf("an invalid address was recognised:\n%s", firstLines(invalid, 12))
	}
}

// TestMagicExtensiveLanguage covers "Magic: extensive language support,
// Yiddish": the wider language set is consulted only when asked for.
func TestMagicExtensiveLanguage(t *testing.T) {
	const input = "די שנעל ברוין פאָקס דזשאַמפּס איבער די פויל הונט."
	if got := runMagic(t, input, 1, false, true, ""); !strings.Contains(got, "Yiddish") {
		t.Errorf("Yiddish was not among the languages:\n%s", firstLines(got, 12))
	}
}

// TestMagicCrib checks that a crib keeps only the candidates containing it.
func TestMagicCrib(t *testing.T) {
	const input = "WkVkV2VtUkRRbnBrU0Vwd1ltMWpQUT09"
	report := runMagic(t, input, 3, false, false, "test string")
	if report == magicNothingFound {
		t.Fatal("the crib matched nothing")
	}
	for line := range strings.SplitSeq(report, "\n") {
		if after, found := strings.CutPrefix(line, "  Data:     "); found &&
			!strings.Contains(strings.ToLower(after), "test string") {
			t.Errorf("a candidate without the crib was kept: %q", after)
		}
	}
	// A crib that appears nowhere leaves nothing.
	if got := runMagic(t, input, 3, false, false, "absent phrase"); got != magicNothingFound {
		t.Errorf("an unmatched crib still reported candidates:\n%s", firstLines(got, 8))
	}
}

// TestMagicCribInvalid checks that a crib which is not a regular expression is
// reported rather than ignored.
func TestMagicCribInvalid(t *testing.T) {
	if _, err := runOp(t, "Magic", "data", 1.0, false, false, "(["); err == nil {
		t.Error("want an error for an invalid crib")
	}
}

// TestMagicDepth checks that the depth bounds how many operations are chained.
func TestMagicDepth(t *testing.T) {
	const input = "WkVkV2VtUkRRbnBrU0Vwd1ltMWpQUT09"
	for _, depth := range []float64{0, 1, 2} {
		for _, recipe := range magicRecipes(runMagic(t, input, depth, false, false, "")) {
			if steps := strings.Count(recipe, "\n") + 1; recipe != "(the data as it is)" &&
				float64(steps) > depth+1 {
				t.Errorf("depth %v produced a recipe of %d steps: %q", depth, steps, recipe)
			}
		}
	}
}

// TestMagicIntensive checks that intensive mode finds data hidden under a
// single-byte exclusive-or, which the ordinary pass cannot see.
func TestMagicIntensive(t *testing.T) {
	hidden := make([]byte, 0, 64)
	for _, b := range []byte("The quick brown fox jumps over the lazy dog. Hello there!") {
		hidden = append(hidden, b^0x2a)
	}
	report := runMagic(t, string(hidden), 1, true, false, "")
	if !strings.Contains(report, "XOR(") {
		t.Errorf("the exclusive-or was not found:\n%s", firstLines(report, 12))
	}
}

// firstLines trims a report for a readable failure message.
func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		return strings.Join(lines[:n], "\n") + "\n..."
	}
	return s
}

// TestMagicOutputCheckRejects covers the branch where an operation claims the
// data but its result does not look like what it promised. "BM" is one of the
// signatures Render Image looks for, so it is suggested for anything starting
// that way — but the result is only offered if it really is an image.
func TestMagicOutputCheckRejects(t *testing.T) {
	report := runMagic(t, "BM this is not really a bitmap file at all, just text", 2, false, false, "")
	if !strings.Contains(report, "Matching ops:") || !strings.Contains(report, "Render Image") {
		t.Fatalf("Render Image was not among the matching ops:\n%s", firstLines(report, 10))
	}
	// It is suggested, but never as a recipe that was actually run.
	for _, recipe := range magicRecipes(report) {
		if strings.Contains(recipe, "Render_Image") {
			t.Errorf("Render Image was offered as a recipe: %q", recipe)
		}
	}
}

// TestMagicEncodingGuessesNeedsAnOptionList covers the brute-force encodings
// stopping cleanly when the operation it relies on does not offer a list of
// encodings to work through.
func TestMagicEncodingGuessesNeedsAnOptionList(t *testing.T) {
	reg := core.NewRegistry()
	reg.Register(oddEncodeText{})
	run := &magicRun{registry: reg}
	if got := run.encodingGuesses([]byte("hello")); got != nil {
		t.Errorf("got %d guesses where there is no list of encodings", len(got))
	}
}

// oddEncodeText stands in for an Encode text whose first argument is not a list
// of encodings.
type oddEncodeText struct{}

func (oddEncodeText) Meta() core.OpMeta {
	return core.OpMeta{Name: "Encode text", InputType: core.TypeString, OutputType: core.TypeString}
}

func (oddEncodeText) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Encoding", Type: core.ArgString, Value: "not a list"}}
}

func (oddEncodeText) Run(in *core.Dish, args []any) (*core.Dish, error) { return in, nil }
