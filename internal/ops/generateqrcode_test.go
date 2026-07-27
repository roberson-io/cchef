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

// The vector fixtures are long enough to read better as their own constants.
const (
	qrSVGFixture = `<svg xmlns="http://www.w3.org/2000/svg" width="145" height="145" viewBox="0 0 29 29"><path d="M4 4h7v7h-7zM12 4h5v2h-1v-1h-1v1h1v1h1v1h-2v3h-1v-1h-1v-1h1v-4h-1v1h-1zM18 4h7v7h-7zM5 5v5h5v-5zM19 5v5h5v-5zM6 6h3v3h-3zM20 6h3v3h-3zM12 7h1v1h-1zM12 10h1v1h1v1h-2zM16 10h1v1h-1zM4 12h1v1h-1zM6 12h2v2h-1v1h1v-1h3v1h-1v1h-2v1h-2v-1h-1v-1h-1v-1h2zM9 12h3v2h-1v-1h-2zM14 12h1v2h-2v-1h1zM18 12h1v1h2v-1h1v1h1v1h-3v2h-1v-2h-1v1h-1v1h-1v-3h2zM23 12h2v3h-1v1h-1v-2h1v-1h-1zM12 14h1v1h-1zM11 15h1v2h-2v-1h1zM14 15h1v1h-1zM21 15h1v1h-1zM15 16h1v2h-1v1h2v-1h1v-1h-1v-1h2v1h2v2h-1v1h-1v-1h-1v1h-2v1h-1v-1h-1v1h-1v-1h-1v-3h3zM24 16h1v2h-1zM22 17h1v1h-1zM4 18h7v7h-7zM13 18v1h1v-1zM5 19v5h5v-5zM21 19h2v1h1v2h-1v1h-1v-1h-1zM6 20h3v3h-3zM12 21h1v2h-1zM14 21h1v1h-1zM16 21h1v2h-1zM18 21h1v1h-1zM19 22h1v1h1v1h-1v1h-3v-2h2zM13 23h1v1h-1zM15 23h1v2h-2v-1h1zM24 23h1v1h-1zM12 24h1v1h-1zM22 24h1v1h-1z"/></svg>`

	qrEPSFixture = `%!PS-Adobe-3.0 EPSF-3.0%%BoundingBox: 0 0 315 315/h { 0 rlineto } bind def/v { 0 exch neg rlineto } bind def/M { neg 30 add moveto } bind def/z { closepath } bind def9 9 scale5 0 M 7 h 7 v -7 h z13 0 M 1 h 1 v -1 h z16 0 M 2 h 1 v -1 h 1 v 1 h 1 v -2 h -1 v -1 h 2 v -1 h -3 v 2 h z20 0 M 2 h 1 v -2 h z23 0 M 7 h 7 v -7 h z6 1 M 5 v 5 h -5 v z24 1 M 5 v 5 h -5 v z7 2 M 3 h 3 v -3 h z19 2 M 1 h 1 v -1 h z21 2 M 1 h 2 v -1 h z25 2 M 3 h 3 v -3 h z13 4 M 1 h 3 v -1 h z17 4 M 4 h 1 v 1 h 2 v -1 h -1 v -1 h 1 v -1 h -1 v -1 h -1 v -1 h z15 6 M 1 h 1 v -1 h z17 6 M 1 h 1 v -1 h z16 7 M 1 h 1 v -1 h z18 7 M 1 h 1 v -1 h z20 7 M 1 h 2 v 1 h 1 v -2 h -1 v -1 h -1 v 1 h z6 8 M 7 h 1 v 2 h 1 v -2 h 1 v -2 h -1 v 1 h -1 v -4 h 1 v -1 h -1 v -1 h z17 8 M 1 h 1 v -1 h z24 8 M 2 h 1 v -1 h 1 v -1 h z29 8 M 1 h 1 v -1 h z27 9 M 1 h 1 v -1 h z6 10 M 1 h 1 v 1 h 1 v -1 h 1 v -1 h z8 10 M 2 h 5 v 1 h 1 v -3 h 1 v -1 h -4 v 1 h 1 v 1 h -1 v -1 h -1 v 1 h -1 v -1 h z16 10 M 2 h 4 v 1 h -1 v 2 h 1 v 3 h -1 v -3 h -2 v 1 h 1 v 1 h -1 v 1 h 1 v 4 h -2 v 2 h 3 v -2 h 1 v -1 h -1 v -2 h 1 v 2 h 1 v -1 h 1 v 2 h 2 v 1 h -1 v 1 h 3 v -1 h -1 v -2 h -2 v -1 h 2 v 1 h 1 v 2 h 1 v -2 h 2 v -1 h -1 v -1 h -1 v -1 h 1 v -1 h -1 v -1 h 2 v -1 h -1 v -1 h -1 v 1 h -2 v -1 h -1 v -1 h 1 v 1 h 2 v -1 h -1 v -1 h 1 v -1 h -3 v 1 h -1 v -1 h 1 v -1 h -1 v -1 h -1 v 4 h 1 v 1 h -2 v -3 h -1 v -1 h 1 v -1 h -1 v -1 h 2 v -1 h 1 v -2 h -1 v 1 h -1 v -1 h -1 v 1 h -1 v 1 h -1 v 1 h 1 v 1 h -1 v 1 h 1 v 1 h -1 v -1 h z22 10 M 1 h 1 v -1 h z25 10 M 2 h 1 v -2 h z19 11 M 1 h 1 v -1 h z11 12 M 1 h 1 v -1 h z5 13 M 1 h 4 v -1 h z28 14 M 2 h 2 v -1 h -1 v -1 h z21 15 M 1 v 2 h -1 v z13 17 M 1 h 1 v 2 h 1 v -2 h 1 v 1 h 1 v 1 h 1 v -2 h 1 v 2 h 2 v -1 h -1 v -2 h z22 17 M 3 v 3 h -3 v z5 18 M 7 h 7 v -7 h z23 18 M 1 h 1 v -1 h z6 19 M 5 v 5 h -5 v z7 20 M 3 h 3 v -3 h z29 21 M 1 h 4 v -3 h -1 v 2 h z17 22 M 2 h 3 v -2 h -1 v 1 h -1 v -1 h z24 22 M 1 h 1 v 1 h 1 v -1 h 1 v -3 h -1 v 1 h -1 v 1 h z20 23 M 1 h 2 v -1 h zfill%%EOF`

	qrPDFFixture = `%PDF-1.01 0 obj << /Type /Catalog /Pages 2 0 R >> endobj2 0 obj << /Type /Pages /Count 1 /Kids [ 3 0 R ] >> endobj3 0 obj << /Type /Page /Parent 2 0 R /Resources <<>> /Contents 4 0 R /MediaBox [ 0 0 261 261 ] >> endobj4 0 obj << /Length 1837 >> stream9 0 0 9 0 0 cm4 25 m 11 25 l 11 18 l 4 18 l h12 25 m 14 25 l 14 23 l 13 23 l 13 24 l 12 24 l h16 25 m 17 25 l 17 22 l 16 22 l h18 25 m 25 25 l 25 18 l 18 18 l h5 24 m 5 19 l 10 19 l 10 24 l h19 24 m 19 19 l 24 19 l 24 24 l h6 23 m 9 23 l 9 20 l 6 20 l h12 23 m 13 23 l 13 21 l 15 21 l 15 20 l 12 20 l h14 23 m 15 23 l 15 22 l 14 22 l h20 23 m 23 23 l 23 20 l 20 20 l h15 22 m 16 22 l 16 21 l 15 21 l h12 19 m 13 19 l 13 18 l 12 18 l h14 19 m 15 19 l 15 13 l 13 13 l 13 11 l 12 11 l 12 14 l 14 14 l 14 15 l 11 15 l 11 16 l 12 16 l 12 17 l 13 17 l 13 16 l 14 16 l 14 17 l 13 17 l 13 18 l 14 18 l h16 19 m 17 19 l 17 18 l 16 18 l h4 17 m 8 17 l 8 16 l 10 16 l 10 15 l 11 15 l 11 14 l 10 14 l 10 13 l 11 13 l 11 12 l 9 12 l 9 15 l 8 15 l 8 13 l 6 13 l 6 15 l 7 15 l 7 16 l 4 16 l h10 17 m 11 17 l 11 16 l 10 16 l h17 17 m 18 17 l 18 16 l 20 16 l 20 17 l 23 17 l 23 15 l 20 15 l 20 13 l 19 13 l 19 15 l 18 15 l 18 14 l 17 14 l 17 13 l 16 13 l 16 16 l 17 16 l h24 17 m 25 17 l 25 14 l 24 14 l 24 13 l 23 13 l 23 15 l 24 15 l h21 14 m 22 14 l 22 13 l 21 13 l h15 13 m 16 13 l 16 11 l 15 11 l h17 13 m 19 13 l 19 12 l 21 12 l 21 10 l 20 10 l 20 9 l 19 9 l 19 10 l 18 10 l 18 9 l 16 9 l 16 8 l 15 8 l 15 10 l 17 10 l 17 11 l 18 11 l 18 12 l 17 12 l h24 13 m 25 13 l 25 11 l 24 11 l h22 12 m 23 12 l 23 11 l 22 11 l h4 11 m 11 11 l 11 4 l 4 4 l h14 11 m 15 11 l 15 10 l 14 10 l h5 10 m 5 5 l 10 5 l 10 10 l h13 10 m 14 10 l 14 9 l 13 9 l h21 10 m 23 10 l 23 9 l 24 9 l 24 7 l 23 7 l 23 6 l 22 6 l 22 7 l 21 7 l h6 9 m 9 9 l 9 6 l 6 6 l h12 8 m 15 8 l 15 7 l 13 7 l 13 6 l 16 6 l 16 4 l 15 4 l 15 5 l 14 5 l 14 4 l 12 4 l h16 8 m 17 8 l 17 6 l 16 6 l h18 8 m 19 8 l 19 7 l 18 7 l h19 7 m 20 7 l 20 6 l 21 6 l 21 5 l 20 5 l 20 4 l 17 4 l 17 6 l 19 6 l h24 6 m 25 6 l 25 5 l 24 5 l h22 5 m 23 5 l 23 4 l 22 4 l hfendstreamendobjxref0 50000000000 65535 f 0000000010 00000 n 0000000059 00000 n 0000000118 00000 n 0000000223 00000 n trailer << /Root 1 0 R /Size 5 >>startxref2111%%EOF`
)

// removeWhitespace is the recipe step CyberChef's EPS and PDF fixtures apply
// before comparing, which strips line feeds and carriage returns.
var removeWhitespace = core.RecipeOp{
	Op:   "Remove whitespace",
	Args: []any{false, true, true, false, false, false},
}

// TestGenerateQRCodeVectorFixtures covers CyberChef's own cases for the three
// vector formats.
func TestGenerateQRCodeVectorFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Generate QR Code: SVG", "Hello world!", qrSVGFixture,
			core.Recipe{{Op: "Generate QR Code", Args: []any{"SVG", float64(5), float64(4), "Medium"}}},
		},
		{
			"Generate QR Code: EPS", "Hello world!", qrEPSFixture,
			core.Recipe{{Op: "Generate QR Code", Args: []any{"EPS", float64(6), float64(5), "Quartile"}}, removeWhitespace},
		},
		{
			"Generate QR Code: PDF", "Hello world!", qrPDFFixture,
			core.Recipe{{Op: "Generate QR Code", Args: []any{"PDF", float64(5), float64(4), "Low"}}, removeWhitespace},
		},
	})
}

// qrGoldenCase is one recorded case of the corpus: the arguments, and the bytes
// CyberChef produces for them.
type qrGoldenCase struct {
	Input  string `json:"input"`
	Format string `json:"format"`
	Size   int    `json:"size"`
	Margin int    `json:"margin"`
	Level  string `json:"level"`
	Want   string `json:"want"` // the output in hexadecimal
}

// TestGenerateQRCodeGolden replays a corpus of CyberChef's own answers, taken
// from the running server across every encoding mode, error correction level,
// version and vector format. CyberChef ships only three vector fixtures, all of
// them byte mode at the smallest versions.
func TestGenerateQRCodeGolden(t *testing.T) {
	file, err := os.Open("testdata/generate_qr_code.jsonl")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 32*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var c qrGoldenCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatalf("line %d: %v", line, err)
		}

		recipe := core.Recipe{{
			Op:   "Generate QR Code",
			Args: []any{c.Format, float64(c.Size), float64(c.Margin), c.Level},
		}}
		out, err := recipe.Execute(core.NewDish([]byte(c.Input), core.TypeString))
		if err != nil {
			t.Errorf("line %d (%s, size %d, margin %d, %s): %v",
				line, c.Format, c.Size, c.Margin, c.Level, err)
			continue
		}
		if got := hex.EncodeToString(out.Bytes()); got != c.Want {
			t.Errorf("line %d (%s, size %d, margin %d, %s): output differs",
				line, c.Format, c.Size, c.Margin, c.Level)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read corpus: %v", err)
	}
}

// TestGenerateQRCodeCapacity covers the largest input each mode carries and the
// first one it refuses, and the versions a high correction level puts out of
// reach. The bounds are those of the largest version, so the mode limit and the
// version search each refuse some inputs and the two boundaries differ by level.
func TestGenerateQRCodeCapacity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		level   string
		refused bool
	}{
		{"the longest numeric input", strings.Repeat("1", 7089), "Low", false},
		{"one digit more", strings.Repeat("1", 7090), "Low", true},
		{"the longest alphanumeric input", strings.Repeat("A", 4296), "Low", false},
		{"one character more", strings.Repeat("A", 4297), "Low", true},
		{"the longest byte input", strings.Repeat("é", 1476) + "x", "Low", false},
		{"one byte more", strings.Repeat("é", 1477), "Low", true},
		{"an input no high-correction version holds", strings.Repeat("x", 2900), "High", true},
		{"nor at the longest numeric input", strings.Repeat("1", 7089), "High", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recipe := core.Recipe{{
				Op:   "Generate QR Code",
				Args: []any{"SVG", float64(5), float64(4), tc.level},
			}}
			out, err := recipe.Execute(core.NewDish([]byte(tc.input), core.TypeString))
			switch {
			case tc.refused && err == nil:
				t.Errorf("accepted %d bytes, want a refusal", len(tc.input))
			case !tc.refused && err != nil:
				t.Errorf("refused %d bytes: %v", len(tc.input), err)
			case !tc.refused && !strings.HasPrefix(out.String(), "<svg"):
				t.Errorf("output is not an SVG: %.40q", out.String())
			}
		})
	}
}

// qrPNGFixture is CyberChef's own PNG case, whose bytes include the compressed
// bitmap and so pin the deflate encoder as well as the container.
const qrPNGFixture = `89 50 4e 47 0d 0a 1a 0a 00 00 00 0d 49 48 44 52 00 00 00 91 00 00 00 91 08 00 00 00 00 e6 b3 05 ff 00 00 01 1a 49 44 41 54 78 da ed da 41 12 83 20 0c 05 50 ef 7f e9 76 dd 05 f4 47 6c c4 ce 63 e5 8c 0c be 4d 24 24 1c af dd c6 41 44 44 44 44 44 44 44 44 44 f4 9f a2 e3 fb 98 cf 2b ad 42 44 d4 2a 1a 07 c3 e7 37 83 a7 d2 37 88 88 1a 44 c3 18 1a 46 e7 ca 2a 44 44 7b 88 4a f3 88 88 1e 23 9a ef 09 44 44 fb 8a 82 b7 c3 3c fe 8e 8c 8d 88 e8 b2 33 6d ba b3 f4 9d b2 89 88 16 eb 90 f3 a5 ef a8 8c 12 11 55 f3 a3 61 a2 93 e6 4c c3 45 89 88 ba 44 a5 e4 e7 64 5d 9d 88 a8 5f 14 74 82 d2 8a 64 5a b4 24 22 6a 10 a5 2d cf 79 3f e9 6c 53 89 88 e8 a7 a2 79 4f a8 b4 b3 ac 57 6b 88 88 ae 15 a5 de b9 bc 14 70 44 44 3f 13 ad d4 21 03 ea d9 6a 0d 11 d1 15 a2 e0 ff 5f 47 07 36 22 a2 06 d1 4a d2 1f 1c 94 89 88 b6 14 95 ee d5 12 11 3d 50 14 74 8c 82 70 24 22 ea 12 05 6f d3 4b 2c 4b d5 1a 22 a2 cb 44 2b 69 7d e9 5e 00 11 51 97 e8 d6 41 44 44 44 44 44 44 44 44 44 f4 7c d1 1b 1c 52 72 cb 26 c8 c7 0b 00 00 00 00 49 45 4e 44 ae 42 60 82`

// TestGenerateQRCodePNGFixture covers that case.
func TestGenerateQRCodePNGFixture(t *testing.T) {
	runCases(t, []opCase{
		{"Generate QR Code: PNG", "Hello world!", qrPNGFixture, core.Recipe{
			{Op: "Generate QR Code", Args: []any{"PNG", float64(5), float64(4), "Medium"}},
			{Op: "To Hex", Args: []any{"Space"}},
		}},
	})
}

// TestGenerateQRCodeUnsupportedFormat covers the guard on the image format.
// The recipe engine checks the option before the operation runs, so the guard
// only answers a direct call.
func TestGenerateQRCodeUnsupportedFormat(t *testing.T) {
	_, err := GenerateQRCode{}.Run(
		core.NewDish([]byte("Hello world!"), core.TypeString),
		[]any{"TIFF", float64(5), float64(4), "Medium"},
	)
	if err == nil {
		t.Error("accepted an image format the operation does not offer")
	}
}

// qrTestMatrix builds a matrix from rows of characters, where a hash is a dark
// module and anything else is light.
func qrTestMatrix(rows ...string) [][]byte {
	matrix := make([][]byte, len(rows))
	for i, row := range rows {
		matrix[i] = make([]byte, len(row))
		for j := range row {
			if row[j] == '#' {
				matrix[i][j] = 1
			}
		}
	}
	return matrix
}

// TestQRPenaltyRules covers each of the four rules on its own, against counts
// worked out by hand rather than taken from the implementation.
func TestQRPenaltyRules(t *testing.T) {
	t.Run("runs of five or more score their length less two", func(t *testing.T) {
		// Five rows and five columns of five, each scoring three.
		dark := qrTestMatrix("#####", "#####", "#####", "#####", "#####")
		if got := qrRunPenalty(dark); got != 30 {
			t.Errorf("a wholly dark code scored %d, want 30", got)
		}
		// Light runs score the same as dark ones.
		light := qrTestMatrix(".....", ".....", ".....", ".....", ".....")
		if got := qrRunPenalty(light); got != 30 {
			t.Errorf("a wholly light code scored %d, want 30", got)
		}
		// Runs of four score nothing at all.
		short := qrTestMatrix("....", "....", "....", "....")
		if got := qrRunPenalty(short); got != 0 {
			t.Errorf("runs of four scored %d, want 0", got)
		}
	})

	t.Run("blocks of four alike score three", func(t *testing.T) {
		// A two by two square holds one block.
		one := qrTestMatrix("##", "##")
		if got := qrBlockPenaltyOf(one); got != 3 {
			t.Errorf("one dark block scored %d, want 3", got)
		}
		// A three by three square holds four overlapping blocks.
		four := qrTestMatrix("###", "###", "###")
		if got := qrBlockPenaltyOf(four); got != 12 {
			t.Errorf("four dark blocks scored %d, want 12", got)
		}
		// A light square counts the same as a dark one.
		light := qrTestMatrix("..", "..")
		if got := qrBlockPenaltyOf(light); got != 3 {
			t.Errorf("one light block scored %d, want 3", got)
		}
		// A block of mixed modules scores nothing.
		mixed := qrTestMatrix("#.", ".#")
		if got := qrBlockPenaltyOf(mixed); got != 0 {
			t.Errorf("a mixed block scored %d, want 0", got)
		}
	})

	t.Run("patterns resembling a finder score forty", func(t *testing.T) {
		// The sequence with four light modules to its left, in one row of a
		// code light everywhere else.
		blank := []string{
			"............", "............", "............", "............",
			"............", "............", "............", "............",
			"............", "............", "............",
		}
		row := qrTestMatrix(append([]string{"....#.###.#."}, blank...)...)
		if got := qrFinderLikePenalty(row); got != 40 {
			t.Errorf("one finder-like row scored %d, want 40", got)
		}
		// The same sequence read downwards.
		down := make([]string, 12)
		for i, module := range "....#.###.#." {
			down[i] = string(module) + "..........."
		}
		if got := qrFinderLikePenalty(qrTestMatrix(down...)); got != 40 {
			t.Errorf("one finder-like column scored %d, want 40", got)
		}
		// The sequence itself, read out of a line of modules.
		for _, tc := range []struct {
			line string
			want bool
		}{
			{"#.###.#", true},
			{"#.##..#", false},
			{"..###.#", false},
			{"#.###..", false},
			{"##.##.#", false},
		} {
			line := tc.line
			if got := qrIsFinderSequence(func(k int) bool { return line[k] == '#' }); got != tc.want {
				t.Errorf("%q read as a finder sequence: %v, want %v", line, got, tc.want)
			}
		}

		// Without the light modules beside it, it scores nothing.
		bare := qrTestMatrix("#.###.#", ".......", ".......", ".......",
			".......", ".......", ".......")
		if got := qrFinderLikePenalty(bare); got != 0 {
			t.Errorf("a bare sequence scored %d, want 0", got)
		}
	})

	t.Run("an unbalanced code scores ten for each twentieth", func(t *testing.T) {
		// Half dark is perfectly balanced.
		balanced := qrTestMatrix("#.", ".#")
		if got := qrBalancePenaltyOf(balanced); got != 0 {
			t.Errorf("a balanced code scored %d, want 0", got)
		}
		// Wholly dark is as far from balanced as it goes.
		dark := qrTestMatrix("##", "##")
		if got := qrBalancePenaltyOf(dark); got != 100 {
			t.Errorf("a wholly dark code scored %d, want 100", got)
		}
		// Wholly light scores the same.
		light := qrTestMatrix("..", "..")
		if got := qrBalancePenaltyOf(light); got != 100 {
			t.Errorf("a wholly light code scored %d, want 100", got)
		}
		// A quarter dark is half way.
		quarter := qrTestMatrix("#...", "....", "....", "....")
		if got := qrBalancePenaltyOf(quarter); got != 80 {
			t.Errorf("a code a sixteenth dark scored %d, want 80", got)
		}
	})
}
