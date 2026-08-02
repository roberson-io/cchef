package ops

import (
	"bytes"
	"embed"
	"encoding/xml"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"strings"
	"sync"

	"github.com/roberson-io/cchef/internal/jimp"
)

// BMFont (AngelCode bitmap font) parsing and text rendering. The glyphs come
// from the same 72px Roboto atlases CyberChef ships, so rendered text is
// pixel-identical.

//go:embed bmfonts/*.fnt bmfonts/*.png
var bmFontFS embed.FS

// bmFontFiles maps CyberChef's font-face option to the vendored asset stem.
var bmFontFiles = map[string]string{
	"Roboto":       "Roboto72White",
	"Roboto Black": "RobotoBlack72White",
	"Roboto Mono":  "RobotoMono72White",
	"Roboto Slab":  "RobotoSlab72White",
}

// bmChar is one glyph's placement in the atlas and its advance metrics.
type bmChar struct {
	X, Y, Width, Height  int
	XOffset, YOffset     int
	XAdvance, PageNumber int
}

// bmFont is a parsed bitmap font: glyph metrics, kerning pairs and the atlas
// pages the glyphs are cut from.
type bmFont struct {
	LineHeight int
	Chars      map[rune]bmChar
	Kernings   map[rune]map[rune]int
	Pages      []*image.NRGBA
	// DefaultCharWidth is the advance used for characters with none of their
	// own: the first glyph in file order with a non-zero xadvance, matching
	// Jimp's `Object.entries(font.chars).find(c => c[1].xadvance)`.
	DefaultCharWidth int
}

// bmFontXML mirrors the subset of the AngelCode XML format the atlases use.
type bmFontXML struct {
	Common struct {
		LineHeight int `xml:"lineHeight,attr"`
	} `xml:"common"`
	Pages []struct {
		ID   int    `xml:"id,attr"`
		File string `xml:"file,attr"`
	} `xml:"pages>page"`
	Chars []struct {
		ID       rune `xml:"id,attr"`
		X        int  `xml:"x,attr"`
		Y        int  `xml:"y,attr"`
		Width    int  `xml:"width,attr"`
		Height   int  `xml:"height,attr"`
		XOffset  int  `xml:"xoffset,attr"`
		YOffset  int  `xml:"yoffset,attr"`
		XAdvance int  `xml:"xadvance,attr"`
		Page     int  `xml:"page,attr"`
	} `xml:"chars>char"`
	Kernings []struct {
		First  rune `xml:"first,attr"`
		Second rune `xml:"second,attr"`
		Amount int  `xml:"amount,attr"`
	} `xml:"kernings>kerning"`
}

// bmFontCache holds fonts already parsed; the atlases are immutable once loaded.
var bmFontCache sync.Map

// loadBMFont parses the vendored atlas for a font face. Parsed fonts are cached
// and must not be mutated by callers.
func loadBMFont(face string) (*bmFont, error) {
	if f, ok := bmFontCache.Load(face); ok {
		return f.(*bmFont), nil
	}
	font, err := loadBMFontFrom(bmFontFS, face)
	if err != nil {
		return nil, err
	}
	bmFontCache.Store(face, font)
	return font, nil
}

// loadBMFontFrom reads a font face's descriptor and atlas pages from fsys.
func loadBMFontFrom(fsys fs.FS, face string) (*bmFont, error) {
	stem, ok := bmFontFiles[face]
	if !ok {
		return nil, fmt.Errorf("unknown font face %q", face)
	}
	data, err := fs.ReadFile(fsys, "bmfonts/"+stem+".fnt")
	if err != nil {
		return nil, err
	}
	return parseBMFont(data, bmFontPageLoader(fsys))
}

// bmFontPageLoader returns a page loader that decodes atlas PNGs from fsys.
func bmFontPageLoader(fsys fs.FS) func(string) (*image.NRGBA, error) {
	return func(file string) (*image.NRGBA, error) {
		data, err := fs.ReadFile(fsys, "bmfonts/"+file)
		if err != nil {
			return nil, err
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		return jimp.ToNRGBA(img), nil
	}
}

// parseBMFont parses an AngelCode XML font descriptor, resolving its atlas pages
// through loadPage.
func parseBMFont(data []byte, loadPage func(string) (*image.NRGBA, error)) (*bmFont, error) {
	var doc bmFontXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	font := &bmFont{
		LineHeight: doc.Common.LineHeight,
		Chars:      make(map[rune]bmChar, len(doc.Chars)),
		Kernings:   make(map[rune]map[rune]int),
		Pages:      make([]*image.NRGBA, len(doc.Pages)),
	}
	for _, c := range doc.Chars {
		font.Chars[c.ID] = bmChar{
			X: c.X, Y: c.Y, Width: c.Width, Height: c.Height,
			XOffset: c.XOffset, YOffset: c.YOffset,
			XAdvance: c.XAdvance, PageNumber: c.Page,
		}
		if font.DefaultCharWidth == 0 {
			font.DefaultCharWidth = c.XAdvance
		}
	}
	for _, k := range doc.Kernings {
		if font.Kernings[k.First] == nil {
			font.Kernings[k.First] = make(map[rune]int)
		}
		font.Kernings[k.First][k.Second] = k.Amount
	}
	for _, p := range doc.Pages {
		img, err := loadPage(p.File)
		if err != nil {
			return nil, err
		}
		if p.ID < 0 || p.ID >= len(font.Pages) {
			return nil, fmt.Errorf("font page id %d out of range", p.ID)
		}
		font.Pages[p.ID] = img
	}
	return font, nil
}

// bmKerning returns the kerning between two adjacent runes.
func bmKerning(f *bmFont, a, b rune) int {
	if k, ok := f.Kernings[a]; ok {
		return k[b]
	}
	return 0
}

// bmMeasureText returns the pixel width of a single line, skipping runes the
// font has no glyph for (Jimp's measureText).
func bmMeasureText(f *bmFont, text string) int {
	runes := []rune(text)
	x := 0
	for i, r := range runes {
		c, ok := f.Chars[r]
		if !ok {
			continue
		}
		x += c.XAdvance
		if i+1 < len(runes) {
			x += bmKerning(f, r, runes[i+1])
		}
	}
	return x
}

// bmPrintLines splits text the way Jimp's splitLines does with the default
// unbounded max width: nothing word-wraps, so a line ends only at a newline.
func bmPrintLines(text string) []string {
	return strings.Split(bmNormaliseNewlines(text), " \n")
}

// bmHeightLines is Jimp's splitLines as CyberChef reaches it from
// measureTextHeight, which passes no max width at all. Every width comparison in
// that function is then against `undefined` and so false, which pushes each word
// onto its own line after an initial empty one. The count is what the height is
// derived from, so only that matters here.
func bmHeightLines(text string) int {
	return len(strings.Split(bmNormaliseNewlines(text), " ")) + 1
}

// bmNormaliseNewlines turns newline runs into " \n", as Jimp does before
// splitting on spaces.
func bmNormaliseNewlines(text string) string {
	var b strings.Builder
	inRun := false
	for _, r := range text {
		if r == '\r' || r == '\n' {
			if !inRun {
				b.WriteString(" \n")
				inRun = true
			}
			continue
		}
		inRun = false
		b.WriteRune(r)
	}
	return b.String()
}

// bmMeasureTextHeight returns the height CyberChef reserves for the text: it
// calls Jimp's measureTextHeight with no max width. See bmSplitLines.
func bmMeasureTextHeight(f *bmFont, text string) int {
	return bmHeightLines(text) * f.LineHeight
}

// bmPrint draws text onto img at (x, y), one line per newline-separated line,
// exactly as Jimp's print with the default unbounded width.
func bmPrint(img *image.NRGBA, f *bmFont, x, y int, text string) {
	for _, line := range bmPrintLines(text) {
		bmPrintLine(img, f, x, y, line, f.DefaultCharWidth)
		y += f.LineHeight
	}
}

// bmPrintLine draws one line of text, substituting "?" for runes the font lacks
// and skipping whitespace, as Jimp's printText does.
func bmPrintLine(img *image.NRGBA, f *bmFont, x, y int, line string, defaultWidth int) {
	runes := []rune(line)
	for i, r := range runes {
		glyph, ok := f.Chars[r]
		key := r
		switch {
		case ok:
		case isSpaceRune(r):
			glyph, key = bmChar{}, 0
		default:
			glyph, key = f.Chars['?'], '?'
		}
		if glyph.Width > 0 && glyph.Height > 0 && glyph.PageNumber < len(f.Pages) {
			jimp.BlitRect(img, f.Pages[glyph.PageNumber], x+glyph.XOffset, y+glyph.YOffset,
				glyph.X, glyph.Y, glyph.Width, glyph.Height)
		}
		advance := glyph.XAdvance
		if advance == 0 {
			advance = defaultWidth
		}
		if i+1 < len(runes) && key != 0 {
			x += bmKerning(f, key, runes[i+1])
		}
		x += advance
	}
}

// isSpaceRune matches JavaScript's \s character class closely enough for the
// glyph lookup: any whitespace is skipped rather than drawn as "?".
func isSpaceRune(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x00a0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}
