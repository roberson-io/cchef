package ops

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"
	"testing/fstest"
)

// The vendored atlases parse into a usable font: the metrics come straight from
// Roboto72White.fnt's <common> and <char> elements.
func TestLoadBMFont(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	if f.LineHeight != 85 {
		t.Errorf("lineHeight = %d, want 85", f.LineHeight)
	}
	if len(f.Pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(f.Pages))
	}
	if w := f.Pages[0].Rect.Dx(); w != 512 {
		t.Errorf("page width = %d, want 512", w)
	}
	a, ok := f.Chars['A']
	if !ok {
		t.Fatal("no glyph for 'A'")
	}
	if a.Width == 0 || a.Height == 0 || a.XAdvance == 0 {
		t.Errorf("glyph A = %+v, want non-zero metrics", a)
	}
}

func TestLoadBMFontUnknownFace(t *testing.T) {
	if _, err := loadBMFont("Comic Sans"); err == nil {
		t.Error("expected an error for an unknown font face")
	}
}

// Every face CyberChef offers must load.
func TestLoadBMFontAllFaces(t *testing.T) {
	for _, face := range []string{"Roboto", "Roboto Black", "Roboto Mono", "Roboto Slab"} {
		if _, err := loadBMFont(face); err != nil {
			t.Errorf("%s: %v", face, err)
		}
	}
}

// measureText sums xadvance plus kerning, skipping characters the font lacks.
func TestBMMeasureText(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	a := f.Chars['A'].XAdvance
	if got := bmMeasureText(f, "A"); got != a {
		t.Errorf("measure(\"A\") = %d, want %d", got, a)
	}
	// Two As, plus any A->A kerning pair.
	want := 2*a + f.Kernings['A']['A']
	if got := bmMeasureText(f, "AA"); got != want {
		t.Errorf("measure(\"AA\") = %d, want %d", got, want)
	}
}

// measureTextHeight is called by CyberChef without a max width. Jimp's
// splitLines compares against undefined, so every comparison is false and each
// word ends up on its own line after an initial empty one: the height is
// (words + 1) * lineHeight.
func TestBMMeasureTextHeight(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		text  string
		lines int
	}{
		{"Hi", 2},
		{"Hello world", 3},
		{"a b c", 4},
	} {
		if got, want := bmMeasureTextHeight(f, tc.text), tc.lines*f.LineHeight; got != want {
			t.Errorf("height(%q) = %d, want %d", tc.text, got, want)
		}
	}
}

// A malformed atlas descriptor is reported rather than yielding a broken font.
func TestParseBMFontMalformed(t *testing.T) {
	if _, err := parseBMFont([]byte("<font><chars"), nil); err == nil {
		t.Error("expected an error for malformed XML")
	}
}

// A page that cannot be read fails the whole font load.
func TestParseBMFontPageError(t *testing.T) {
	const doc = `<font><common lineHeight="10"/><pages><page id="0" file="missing.png"/></pages></font>`
	_, err := parseBMFont([]byte(doc), func(string) (*image.NRGBA, error) {
		return nil, errors.New("no such page")
	})
	if err == nil || !strings.Contains(err.Error(), "no such page") {
		t.Errorf("error = %v, want the page error", err)
	}
}

// Page ids outside the declared page list are rejected instead of panicking.
func TestParseBMFontBadPageID(t *testing.T) {
	const doc = `<font><common lineHeight="10"/><pages><page id="3" file="a.png"/></pages></font>`
	_, err := parseBMFont([]byte(doc), func(string) (*image.NRGBA, error) {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	})
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error = %v, want an out-of-range error", err)
	}
}

// The fallback advance is the first glyph in file order with a non-zero
// xadvance, as Jimp's `Object.entries(font.chars).find(...)` gives.
func TestParseBMFontDefaultCharWidth(t *testing.T) {
	const doc = `<font><common lineHeight="10"/><pages></pages><chars>` +
		`<char id="65" xadvance="0"/><char id="66" xadvance="7"/><char id="67" xadvance="9"/>` +
		`</chars></font>`
	f, err := parseBMFont([]byte(doc), nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.DefaultCharWidth != 7 {
		t.Errorf("defaultCharWidth = %d, want 7", f.DefaultCharWidth)
	}
}

// A font with no advancing glyph at all has no fallback width.
func TestParseBMFontNoDefaultCharWidth(t *testing.T) {
	f, err := parseBMFont([]byte(`<font><common lineHeight="10"/><chars><char id="65" xadvance="0"/></chars></font>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.DefaultCharWidth != 0 {
		t.Errorf("defaultCharWidth = %d, want 0", f.DefaultCharWidth)
	}
}

// Characters the font has no glyph for contribute nothing to the measured width.
func TestBMMeasureTextUnknownRune(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := bmMeasureText(f, "A中"), bmMeasureText(f, "A"); got != want {
		t.Errorf("measure with an unmapped rune = %d, want %d", got, want)
	}
}

// Newline runs collapse to a single line break, whatever the line ending.
func TestBMNormaliseNewlines(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"a\nb", "a \nb"},
		{"a\r\nb", "a \nb"},
		{"a\n\n\nb", "a \nb"},
		{"plain", "plain"},
	} {
		if got := bmNormaliseNewlines(tc.in); got != tc.want {
			t.Errorf("normalise(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// Newlines start a new line when printing, so the text occupies two rows.
func TestBMPrintNewline(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 200, 200))
	bmPrint(img, f, 0, 0, "A\nA")
	var rowOne, rowTwo bool
	for y := range 200 {
		for x := range 200 {
			if img.Pix[img.PixOffset(x, y)+3] == 0 {
				continue
			}
			if y < f.LineHeight {
				rowOne = true
			} else {
				rowTwo = true
			}
		}
	}
	if !rowOne || !rowTwo {
		t.Errorf("rows drawn: first=%v second=%v, want both", rowOne, rowTwo)
	}
}

// Whitespace with no glyph is skipped rather than drawn as "?", and a rune the
// font lacks falls back to "?".
func TestBMPrintFallbacks(t *testing.T) {
	f, err := loadBMFont("Roboto")
	if err != nil {
		t.Fatal(err)
	}
	tab := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	bmPrint(tab, f, 0, 0, "\t")
	for i := 3; i < len(tab.Pix); i += 4 {
		if tab.Pix[i] != 0 {
			t.Fatal("a tab drew a glyph; whitespace must be skipped")
		}
	}
	unknown := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	bmPrint(unknown, f, 0, 0, "中")
	question := image.NewNRGBA(image.Rect(0, 0, 300, 200))
	bmPrint(question, f, 0, 0, "?")
	if !bytes.Equal(unknown.Pix, question.Pix) {
		t.Error("an unmapped rune must render as \"?\"")
	}
}

// The atlas assets are embedded, so these failures cannot happen in production;
// they are exercised through the filesystem seam to keep the error handling
// honest.
func TestLoadBMFontFromBrokenAssets(t *testing.T) {
	const good = `<font><common lineHeight="10"/><pages><page id="0" file="p.png"/></pages></font>`
	var png bytes.Buffer
	if err := pngEncodeForTest(&png); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		files fstest.MapFS
	}{
		{"missing descriptor", fstest.MapFS{}},
		{"corrupt descriptor", fstest.MapFS{"bmfonts/Roboto72White.fnt": {Data: []byte("<font><chars")}}},
		{"missing page", fstest.MapFS{"bmfonts/Roboto72White.fnt": {Data: []byte(good)}}},
		{"corrupt page", fstest.MapFS{
			"bmfonts/Roboto72White.fnt": {Data: []byte(good)},
			"bmfonts/p.png":             {Data: []byte("not a png")},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := loadBMFontFrom(tc.files, "Roboto"); err == nil {
				t.Error("expected an error")
			}
		})
	}
	// The same seam with valid assets succeeds.
	ok := fstest.MapFS{
		"bmfonts/Roboto72White.fnt": {Data: []byte(good)},
		"bmfonts/p.png":             {Data: png.Bytes()},
	}
	if _, err := loadBMFontFrom(ok, "Roboto"); err != nil {
		t.Errorf("valid assets: %v", err)
	}
}

// Repeated loads come from the cache and return the same font.
func TestLoadBMFontCached(t *testing.T) {
	first, err := loadBMFont("Roboto Mono")
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadBMFont("Roboto Mono")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("expected the cached font instance")
	}
}

// pngEncodeForTest writes a minimal valid PNG.
func pngEncodeForTest(w io.Writer) error {
	return png.Encode(w, image.NewNRGBA(image.Rect(0, 0, 1, 1)))
}
