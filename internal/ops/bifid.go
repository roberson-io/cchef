package ops

import (
	"errors"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(BifidCipherEncode{})
	core.Register(BifidCipherDecode{})
}

// bifidAlpha is the 25-letter Polybius alphabet (J is folded onto I).
const bifidAlpha = "ABCDEFGHIKLMNOPQRSTUVWXYZ"

// bifidDesc is shared by both operations.
const bifidDesc = "The Bifid cipher is a cipher which uses a Polybius square in conjunction with transposition, which can be fairly difficult to decipher without knowing the alphabet keyword."

// bifidSquare builds the 5x5 Polybius square (row-major) from the keyword,
// mirroring CyberChef genPolybiusSquare: unique(keyword + alphabet) truncated to
// 25 letters. The keyword is already upper-cased and J-folded by the caller.
func bifidSquare(keyword string) [25]byte {
	var seen [128]bool
	var sq [25]byte
	n := 0
	add := func(s string) {
		for i := 0; i < len(s) && n < 25; i++ {
			if c := s[i]; c < 128 && !seen[c] {
				seen[c] = true
				sq[n] = c
				n++
			}
		}
	}
	add(keyword)
	add(bifidAlpha)
	return sq
}

// bifidIndex returns the row-major position of c in the square, or -1.
func bifidIndex(sq [25]byte, c byte) int {
	for i := range 25 {
		if sq[i] == c {
			return i
		}
	}
	return -1
}

// asciiUpper upper-cases an ASCII letter, leaving other runes unchanged (matches
// the effective behaviour of JS toUpperCase for the English alphabet).
func asciiUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 32
	}
	return r
}

// bifidToken is either a cipher letter (with its square coordinates and case) or
// a passthrough rune (spaces, punctuation, non-alphabet characters).
type bifidToken struct {
	letter bool
	upper  bool
	lit    rune
	row    int
	col    int
}

// bifidPrepare validates the keyword and tokenises the input. Ported from the
// shared preamble of BifidCipherEncode/Decode.mjs.
func bifidPrepare(input, keyword string) ([25]byte, []bifidToken, error) {
	// keywordStr = args[0].toUpperCase().replace("J", "I") — replace only the
	// first "J", matching JS String.replace with a non-global pattern.
	kw := strings.Replace(strings.ToUpper(keyword), "J", "I", 1)
	if len(kw) > 0 && !isAllUpperLetters(kw) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return [25]byte{}, nil, errors.New("The key must consist only of letters in the English alphabet")
	}
	sq := bifidSquare(kw)

	// input.replace("J", "I") — again only the first "J".
	in := strings.Replace(input, "J", "I", 1)
	var toks []bifidToken
	for _, r := range in {
		up := asciiUpper(r)
		if up >= 0 && up < 128 {
			if pos := bifidIndex(sq, byte(up)); pos >= 0 {
				toks = append(toks, bifidToken{letter: true, upper: r == up, row: pos / 5, col: pos % 5})
				continue
			}
		}
		toks = append(toks, bifidToken{lit: r})
	}
	return sq, toks, nil
}

// isAllUpperLetters reports whether s is non-empty and every byte is A-Z
// (mirrors /^[A-Z]+$/ on the already-upper-cased keyword).
func isAllUpperLetters(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return len(s) > 0
}

// bifidRender emits the output: each cipher letter uses its recomputed
// coordinates (row[k], col[k]); passthrough runes are copied verbatim.
func bifidRender(sq [25]byte, toks []bifidToken, row, col []int) string {
	var b strings.Builder
	k := 0
	for _, t := range toks {
		if !t.letter {
			b.WriteRune(t.lit)
			continue
		}
		c := sq[row[k]*5+col[k]]
		if !t.upper {
			c += 32
		}
		b.WriteByte(c)
		k++
	}
	return b.String()
}

// BifidCipherEncode enciphers text with the Bifid cipher.
type BifidCipherEncode struct{}

// Meta returns the operation metadata.
func (BifidCipherEncode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bifid Cipher Encode",
		Module:      "Ciphers",
		Description: bifidDesc,
		InfoURL:     "https://wikipedia.org/wiki/Bifid_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BifidCipherEncode) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Keyword", Type: core.ArgString, Value: ""}}
}

// Run encodes the input. Ported from CyberChef BifidCipherEncode.mjs.
func (BifidCipherEncode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sq, toks, err := bifidPrepare(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	// Collect every letter's row then every letter's column, then re-pair each
	// consecutive (row, col) as new coordinates.
	var rows, cols []int
	for _, t := range toks {
		if t.letter {
			rows = append(rows, t.row)
			cols = append(cols, t.col)
		}
	}
	seq := make([]int, 0, len(rows)+len(cols))
	seq = append(seq, rows...)
	seq = append(seq, cols...)
	newRow := make([]int, len(rows))
	newCol := make([]int, len(rows))
	for k := range rows {
		newRow[k] = seq[2*k]
		newCol[k] = seq[2*k+1]
	}
	return core.NewDish([]byte(bifidRender(sq, toks, newRow, newCol)), core.TypeString), nil
}

// BifidCipherDecode deciphers Bifid-encoded text.
type BifidCipherDecode struct{}

// Meta returns the operation metadata.
func (BifidCipherDecode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bifid Cipher Decode",
		Module:      "Ciphers",
		Description: bifidDesc,
		InfoURL:     "https://wikipedia.org/wiki/Bifid_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (BifidCipherDecode) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Keyword", Type: core.ArgString, Value: ""}}
}

// Run decodes the input. Ported from CyberChef BifidCipherDecode.mjs.
func (BifidCipherDecode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sq, toks, err := bifidPrepare(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	// Interleave each letter's (row, col), then split the sequence in half: the
	// new row comes from the first half, the new column from the second.
	var seq []int
	for _, t := range toks {
		if t.letter {
			seq = append(seq, t.row, t.col)
		}
	}
	n := len(seq) / 2
	newRow := make([]int, n)
	newCol := make([]int, n)
	for k := range n {
		newRow[k] = seq[k]
		newCol[k] = seq[k+n]
	}
	return core.NewDish([]byte(bifidRender(sq, toks, newRow, newCol)), core.TypeString), nil
}
