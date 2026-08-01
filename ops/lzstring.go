package ops

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/roberson-io/cchef/core"
)

// LZString compression, the form Pieroxy's lz-string library writes.
//
// It is LZW worked in UTF-16 code units: the dictionary starts empty, takes an
// entry for each character the first time it is seen, and gains another every
// time a code is written, so a reader following along builds the same one. The
// codes are packed into characters of a fixed width, and the three formats
// differ only in that width and in where they put the characters that result.
//
// The plain format uses all sixteen bits of a character, which means about a
// quarter of its output holds a surrogate with no partner — a code unit with no
// character behind it. Those are written here as three bytes holding the number
// itself, so that the text comes back; strict UTF-8 would put a replacement
// character in their place and the stream would no longer read.

// The three values that are not dictionary codes, and how a stream starts.
const (
	lzsNewByteChar = 0 // a character below 256, written as eight bits
	lzsNewWideChar = 1 // any other character, written as sixteen
	lzsEndOfStream = 2
	// lzsFirstCode is the first code the dictionary gives out, the three marks
	// above having taken the rest.
	lzsFirstCode = 3
	// lzsMarkerBits is how wide a code is to begin with, which is just enough
	// for those three marks.
	lzsMarkerBits = 2
	// lzsCodeBits is how wide a code is once the stream is under way: the three
	// marks and the character it opens with have filled two bits between them.
	lzsCodeBits = 3

	lzsByteCharBits = 8
	lzsWideCharBits = 16
	// lzsByteCharLimit is the point above which a character needs the wider form.
	lzsByteCharLimit = 256
	// lzsNoPrefix stands for the empty string, which no code is given to.
	lzsNoPrefix = 0
)

// How each format packs its bits and where it puts the characters that result.
const (
	lzsFormatUTF16  = "UTF16"
	lzsFormatBase64 = "Base64"

	lzsPlainBits  = 16
	lzsUTF16Bits  = 15
	lzsBase64Bits = 6
	// lzsUTF16Offset lifts every character clear of the control range.
	lzsUTF16Offset = 32
	// lzsBase64Group is the length Base64 output is padded out to.
	lzsBase64Group = 4
)

// lzsBase64Alphabet is the one lz-string uses, the padding character included:
// it reads that character as a value like any other.
const lzsBase64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

// lzsOutputFormats are the three shapes the compressed form is offered in.
var lzsOutputFormats = []string{"default", lzsFormatUTF16, lzsFormatBase64}

// lzsBitsFor is how many bits of each output character a format fills. Anything
// other than the two named formats is the plain one, which is what the option
// offers as its default and all it can otherwise be.
func lzsBitsFor(format string) int {
	switch format {
	case lzsFormatUTF16:
		return lzsUTF16Bits
	case lzsFormatBase64:
		return lzsBase64Bits
	}
	return lzsPlainBits
}

// LZStringCompress compresses text with lz-string.
type LZStringCompress struct{}

// Meta returns the operation metadata.
func (LZStringCompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZString Compress",
		Module:      "Compression",
		Description: "Compress the input with lz-string.",
		InfoURL:     "https://pieroxy.net/blog/pages/lz-string/index.html",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LZStringCompress) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression Format", Type: core.ArgOption, Value: lzsOutputFormats},
	}
}

// Run compresses the input.
func (LZStringCompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format, _ := args[0].(string)
	values := lzsCompress(lzsTextToUnits(dishText(in)), lzsBitsFor(format))
	return core.NewDish([]byte(lzsWriteOutput(values, format)), core.TypeString), nil
}

// lzsPair names a dictionary entry by the entry it extends and the character it
// adds. That is enough to tell every entry apart, since each one is an earlier
// entry plus a character and a single character extends the empty one.
type lzsPair struct {
	prefix int
	unit   uint16
}

// lzsEncoder holds what a run of compression needs: the dictionary, the
// characters given a code but not yet spelled out, and the packer.
type lzsEncoder struct {
	out       lzsWriter
	dict      map[lzsPair]int
	pending   map[int]uint16
	numBits   int
	enlargeIn int
}

// lzsCompress packs text into values of the given width.
func lzsCompress(units []uint16, width int) []int {
	e := &lzsEncoder{
		out:     lzsWriter{width: width},
		dict:    map[lzsPair]int{},
		pending: map[int]uint16{},
		numBits: lzsMarkerBits,
		// Two, rather than one, to make up for the first entry not counting.
		enlargeIn: 2,
	}

	held := lzsNoPrefix
	for _, c := range units {
		single := lzsPair{lzsNoPrefix, c}
		if _, ok := e.dict[single]; !ok {
			e.dict[single] = len(e.dict) + lzsFirstCode
			e.pending[e.dict[single]] = c
		}
		// Keep going while what is held plus this character is something the
		// dictionary already knows.
		if code, ok := e.dict[lzsPair{held, c}]; ok {
			held = code
			continue
		}
		e.emit(held)
		e.dict[lzsPair{held, c}] = len(e.dict) + lzsFirstCode
		held = e.dict[single]
	}
	if held != lzsNoPrefix {
		e.emit(held)
	}

	e.out.write(lzsEndOfStream, e.numBits)
	e.out.flush()
	return e.out.values
}

// emit writes the code for an entry. A character that has been given a code but
// never spelled out goes out in full the first time, so that a reader can put
// it in its own dictionary.
func (e *lzsEncoder) emit(code int) {
	if unit, ok := e.pending[code]; ok {
		if unit < lzsByteCharLimit {
			e.out.write(lzsNewByteChar, e.numBits)
			e.out.write(int(unit), lzsByteCharBits)
		} else {
			e.out.write(lzsNewWideChar, e.numBits)
			e.out.write(int(unit), lzsWideCharBits)
		}
		e.grow()
		delete(e.pending, code)
	} else {
		e.out.write(code, e.numBits)
	}
	e.grow()
}

// grow counts an entry off, widening the codes once the dictionary has filled
// the width it has.
func (e *lzsEncoder) grow() {
	e.enlargeIn--
	if e.enlargeIn == 0 {
		e.enlargeIn = 1 << e.numBits
		e.numBits++
	}
}

// lzsWriter packs values into characters of a fixed width, filling each from its
// top bit down while taking the value from its bottom bit up.
type lzsWriter struct {
	width  int
	val    int
	pos    int
	values []int
}

// write puts the low bits of value into the output, least significant first.
func (w *lzsWriter) write(value, bits int) {
	for range bits {
		w.val = w.val<<1 | value&1
		if w.pos == w.width-1 {
			w.pos = 0
			w.values = append(w.values, w.val)
			w.val = 0
		} else {
			w.pos++
		}
		value >>= 1
	}
}

// flush shifts whatever is left of the last character up until it is full.
func (w *lzsWriter) flush() {
	for {
		w.val <<= 1
		if w.pos == w.width-1 {
			w.values = append(w.values, w.val)
			return
		}
		w.pos++
	}
}

// lzsWriteOutput turns packed values into the characters a format writes.
func lzsWriteOutput(values []int, format string) string {
	switch format {
	case lzsFormatUTF16:
		// Every character is lifted clear of the control range, and a space
		// closes the run.
		units := make([]uint16, 0, len(values)+1)
		for _, v := range values {
			units = append(units, uint16(v+lzsUTF16Offset)) // #nosec G115 -- fifteen bits and an offset of 32
		}
		return lzsUnitsToString(append(units, ' '))

	case lzsFormatBase64:
		var b strings.Builder
		for _, v := range values {
			b.WriteByte(lzsBase64Alphabet[v])
		}
		for b.Len()%lzsBase64Group != 0 {
			b.WriteByte('=')
		}
		return b.String()
	}

	units := make([]uint16, len(values))
	for i, v := range values {
		units[i] = uint16(v) // #nosec G115 -- a value is as wide as a character
	}
	return lzsUnitsToString(units)
}

// LZStringDecompress reads text back out of an lz-string stream.
type LZStringDecompress struct{}

// Meta returns the operation metadata.
func (LZStringDecompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LZString Decompress",
		Module:      "Compression",
		Description: "Decompresses data that was compressed with lz-string.",
		InfoURL:     "https://pieroxy.net/blog/pages/lz-string/index.html",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LZStringDecompress) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression Format", Type: core.ArgOption, Value: lzsOutputFormats},
	}
}

// Run decompresses the input.
func (LZStringDecompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format, _ := args[0].(string)
	values, err := lzsReadInput(in.String(), format)
	if err != nil {
		return nil, err
	}
	units, err := lzsDecompress(values, lzsBitsFor(format))
	if err != nil {
		return nil, err
	}
	return core.NewDish(textAsBytes(lzsUnitsToString(units)), core.TypeString), nil
}

// lzsReadInput turns the characters a format writes back into packed values.
func lzsReadInput(s, format string) ([]int, error) {
	switch format {
	case lzsFormatUTF16:
		units := lzsStringToUnits(s)
		values := make([]int, len(units))
		for i, u := range units {
			if u < lzsUTF16Offset {
				return nil, fmt.Errorf("the UTF16 form has no character below %#04x, and this holds %#04x",
					lzsUTF16Offset, u)
			}
			values[i] = int(u) - lzsUTF16Offset
		}
		return values, nil

	case lzsFormatBase64:
		values := make([]int, 0, len(s))
		for _, r := range s {
			at := strings.IndexRune(lzsBase64Alphabet, r)
			if at < 0 {
				return nil, fmt.Errorf("%q is not a character the Base64 form uses", r)
			}
			values = append(values, at)
		}
		return values, nil
	}

	units := lzsStringToUnits(s)
	values := make([]int, len(units))
	for i, u := range units {
		values[i] = int(u)
	}
	return values, nil
}

// lzsReader takes bits out of packed values, from the top of each one down.
type lzsReader struct {
	values []int
	val    int
	mask   int
	reset  int
	// index counts the values taken so far, and going past the end is only a
	// fault if bits are then read from what was not there.
	index int
}

// read takes the next bits, assembling them least significant first.
func (r *lzsReader) read(bits int) int {
	value, place := 0, 1
	for range bits {
		bit := r.val & r.mask
		r.mask >>= 1
		if r.mask == 0 {
			r.mask = r.reset
			r.val = r.next()
		}
		if bit > 0 {
			value |= place
		}
		place <<= 1
	}
	return value
}

// next takes the following value, or nothing once they run out.
func (r *lzsReader) next() int {
	at := r.index
	r.index++
	if at >= len(r.values) {
		return 0
	}
	return r.values[at]
}

// spent reports whether more values have been asked for than there were.
func (r *lzsReader) spent() bool { return r.index > len(r.values) }

// lzsDecompress reads packed values of the given width back into text.
func lzsDecompress(values []int, width int) ([]uint16, error) {
	if len(values) == 0 {
		return nil, errors.New("there is no LZString stream to read")
	}
	r := &lzsReader{values: values, reset: 1 << (width - 1)}
	r.mask = r.reset
	r.val = r.next()

	first, err := lzsOpen(r)
	if err != nil {
		return nil, err
	}
	if first == nil {
		return nil, nil // a stream holding nothing but its end mark
	}

	// The first three entries stand for the marks, and are never looked up.
	dict := [][]uint16{nil, nil, nil, first}
	numBits := lzsCodeBits
	enlargeIn := 4
	held := first
	out := append([]uint16{}, first...)

	for {
		if r.spent() {
			return nil, errors.New("the LZString stream ends before its end mark")
		}
		code := r.read(numBits)
		if code == lzsEndOfStream {
			return out, nil
		}
		if code == lzsNewByteChar || code == lzsNewWideChar {
			// #nosec G115 -- eight bits or sixteen, either way a code unit
			dict = append(dict, []uint16{uint16(r.read(lzsCharBits(code)))})
			code = len(dict) - 1
			enlargeIn--
		}
		if enlargeIn == 0 {
			enlargeIn = 1 << numBits
			numBits++
		}

		entry, err := lzsEntry(dict, code, held)
		if err != nil {
			return nil, err
		}
		out = append(out, entry...)

		// What was held, followed by the first character of what just arrived,
		// is the entry both sides add next.
		dict = append(dict, append(append([]uint16{}, held...), entry[0]))
		held = entry
		enlargeIn--
		if enlargeIn == 0 {
			enlargeIn = 1 << numBits
			numBits++
		}
	}
}

// lzsCharBits is how many bits a character spelled out in full takes.
func lzsCharBits(mark int) int {
	if mark == lzsNewWideChar {
		return lzsWideCharBits
	}
	return lzsByteCharBits
}

// lzsOpen reads the character a stream opens with, returning nothing at all if
// the stream is over before it starts.
func lzsOpen(r *lzsReader) ([]uint16, error) {
	switch mark := r.read(lzsMarkerBits); mark {
	case lzsNewByteChar, lzsNewWideChar:
		// #nosec G115 -- eight bits or sixteen, either way a code unit
		return []uint16{uint16(r.read(lzsCharBits(mark)))}, nil
	case lzsEndOfStream:
		return nil, nil
	default:
		return nil, fmt.Errorf("the LZString stream opens with %d, which means nothing here", mark)
	}
}

// lzsEntry looks a code up. A code one past the end of the dictionary is the
// one case where a reader has not caught up yet: it stands for what was held
// with its own first character on the end.
func lzsEntry(dict [][]uint16, code int, held []uint16) ([]uint16, error) {
	if code >= lzsFirstCode && code < len(dict) {
		return dict[code], nil
	}
	if code == len(dict) {
		return append(append([]uint16{}, held...), held[0]), nil
	}
	return nil, fmt.Errorf("the LZString stream names entry %d, which is not there", code)
}

// lzsUnitsToString writes UTF-16 code units out as bytes.
//
// A surrogate that has no partner stands for no character at all, so it is
// written as though it did: three bytes holding its own number. That is not
// strictly UTF-8, but it is reversible, and the alternative loses the stream.
func lzsUnitsToString(units []uint16) string {
	var b []byte
	for i := 0; i < len(units); i++ {
		u := rune(units[i])
		if utf16.IsSurrogate(u) && i+1 < len(units) {
			if r := utf16.DecodeRune(u, rune(units[i+1])); r != utf8.RuneError {
				b = utf8.AppendRune(b, r)
				i++
				continue
			}
		}
		if !utf16.IsSurrogate(u) {
			b = utf8.AppendRune(b, u)
			continue
		}
		// #nosec G115 -- a surrogate is sixteen bits, split across three of eight
		b = append(b, 0xE0|byte(u>>12), 0x80|byte(u>>6&0x3F), 0x80|byte(u&0x3F))
	}
	return string(b)
}

// lzsStringToUnits reads bytes as the UTF-16 code units lz-string works in,
// taking back the lone surrogates lzsUnitsToString writes.
func lzsStringToUnits(s string) []uint16 {
	var units []uint16
	for i := 0; i < len(s); {
		if u, ok := lzsLoneSurrogateAt(s, i); ok {
			units = append(units, u)
			i += 3
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		units = append(units, utf16.Encode([]rune{r})...)
		i += size
	}
	return units
}

// lzsTextToUnits turns the input text into code units. Unlike
// lzsStringToUnits, which reads a compressed stream and has to recover the
// partnerless surrogates written into one, this reads ordinary text: the
// characters are already whatever dishText made of the input bytes.
func lzsTextToUnits(s string) []uint16 {
	return utf16.Encode([]rune(s))
}

// lzsLoneSurrogateAt reads the three bytes lzsUnitsToString writes a partnerless
// surrogate as, which the standard decoder will not.
func lzsLoneSurrogateAt(s string, i int) (uint16, bool) {
	if i+3 > len(s) || s[i]&0xF0 != 0xE0 || s[i+1]&0xC0 != 0x80 || s[i+2]&0xC0 != 0x80 {
		return 0, false
	}
	u := rune(s[i]&0x0F)<<12 | rune(s[i+1]&0x3F)<<6 | rune(s[i+2]&0x3F)
	if !utf16.IsSurrogate(u) {
		return 0, false
	}
	return uint16(u), true // #nosec G115 -- a surrogate is sixteen bits
}

func init() {
	core.Register(LZStringCompress{})
	core.Register(LZStringDecompress{})
}
