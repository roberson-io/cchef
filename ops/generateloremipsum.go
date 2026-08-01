package ops

import (
	"errors"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"unicode"

	"github.com/roberson-io/cchef/core"
)

// The bounds on how much text will be made at once.
const (
	maxLoremWords = 100000
	maxLoremBytes = 1000000
)

// The shape of the text: how long a sentence and a paragraph run on average and
// how widely each varies, and how often a comma is dropped into a sentence.
const (
	loremSentenceMean   = 15
	loremSentenceSpread = 9
	loremParagraphMean  = 5
	loremParagraphRange = 2
	loremCommaChance    = 0.35
)

// loremDraws is how many values are added together to draw a length. Adding
// several evens the draw out, so lengths gather about the mean instead of lying
// flat across the range.
const loremDraws = 3

// loremOpening is what the text always begins with, however the rest falls out.
var loremOpening = []string{"Lorem", "ipsum", "dolor", "sit", "amet"}

// The units the length may be given in.
const (
	loremParagraphs = "Paragraphs"
	loremSentences  = "Sentences"
	loremWords      = "Words"
	loremBytes      = "Bytes"
)

// loremBytesPerWord is how many characters a word is reckoned to be worth when
// working out how many to draw for a length given in bytes. It is a deliberate
// underestimate, so that more than enough text is made and then cut back.
const loremBytesPerWord = 3

// loremWordList is the vocabulary, which is the passage the text is named for.
var loremWordList = []string{
	"ad", "adipisicing", "aliqua", "aliquip", "amet", "anim",
	"aute", "cillum", "commodo", "consectetur", "consequat", "culpa",
	"cupidatat", "deserunt", "do", "dolor", "dolore", "duis",
	"ea", "eiusmod", "elit", "enim", "esse", "est",
	"et", "eu", "ex", "excepteur", "exercitation", "fugiat",
	"id", "in", "incididunt", "ipsum", "irure", "labore",
	"laboris", "laborum", "Lorem", "magna", "minim", "mollit",
	"nisi", "non", "nostrud", "nulla", "occaecat", "officia",
	"pariatur", "proident", "qui", "quis", "reprehenderit", "sint",
	"sit", "sunt", "tempor", "ullamco", "ut", "velit",
	"veniam", "voluptate",
}

// loremDraft is the text as it is put together: paragraphs, each holding
// sentences, each holding the words of that sentence. Punctuation is left until
// the end so that the opening can be put in place without disturbing it.
type loremDraft [][][]string

// GenerateLoremIpsum makes placeholder text.
type GenerateLoremIpsum struct{}

// Meta returns the operation metadata.
func (GenerateLoremIpsum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate Lorem Ipsum",
		Module:      "Default",
		Description: "Generate varying length lorem ipsum placeholder text.",
		InfoURL:     "https://wikipedia.org/wiki/Lorem_ipsum",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateLoremIpsum) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Length", Type: core.ArgNumber, Integer: true, Value: 3.0},
		{
			Name: "Length in", Type: core.ArgOption,
			Value: []string{loremParagraphs, loremSentences, loremWords, loremBytes},
		},
	}
}

// Run makes the text.
func (GenerateLoremIpsum) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length, _ := args[0].(float64)
	unit, _ := args[1].(string)

	if err := checkLoremLength(unit, length); err != nil {
		return nil, err
	}

	var text string
	switch unit {
	case loremParagraphs:
		text = loremDraftParagraphs(int(length)).render()
	case loremSentences:
		text = loremDraftSentences(int(length)).render()
	case loremWords:
		text = loremDraftWords(int(length)).render()
	case loremBytes:
		text = loremMakeBytes(int(length))
	default:
		return nil, errors.New("Length in must be one of: " + strings.Join(
			[]string{loremParagraphs, loremSentences, loremWords, loremBytes}, ", "))
	}
	return core.NewDish([]byte(text), core.TypeString), nil
}

// checkLoremLength reports whether that much text will be made.
func checkLoremLength(unit string, length float64) error {
	if length < 1 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return errors.New("Length must be greater than 0")
	}
	limit := maxLoremWords
	if unit == loremBytes {
		limit = maxLoremBytes
	}
	if length > float64(limit) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return errors.New("Length must be less than " + strconv.Itoa(limit))
	}
	return nil
}

// loremDraftParagraphs drafts that many paragraphs, each of a few sentences.
func loremDraftParagraphs(length int) loremDraft {
	draft := make(loremDraft, length)
	for i := range draft {
		sentences := make([][]string, loremDrawLength(loremParagraphMean, loremParagraphRange))
		for j := range sentences {
			sentences[j] = loremPickWords(loremDrawSentenceLength(i == 0 && j == 0))
		}
		draft[i] = sentences
	}
	return draft
}

// loremDraftSentences drafts that many sentences, gathered into paragraphs.
func loremDraftSentences(length int) loremDraft {
	sentences := make([][]string, length)
	for i := range sentences {
		sentences[i] = loremPickWords(loremDrawSentenceLength(i == 0))
	}
	return loremGather(sentences)
}

// loremDraftWords drafts that many words, divided into sentences and gathered
// into paragraphs.
func loremDraftWords(length int) loremDraft {
	words := loremPickWords(length)

	var sentences [][]string
	for len(words) > 0 {
		count := min(loremDrawSentenceLength(len(sentences) == 0), len(words))
		sentences = append(sentences, words[:count])
		words = words[count:]
	}
	return loremGather(sentences)
}

// loremMakeBytes makes text and cuts it to that many characters. The shortest
// word is two characters and takes a third for the space or full stop after it,
// so drawing a word for every three characters asked for always gives enough to
// cut back from.
func loremMakeBytes(length int) string {
	return loremDraftWords(length/loremBytesPerWord + 1).render()[:length]
}

// loremGather divides sentences into paragraphs of a few sentences each.
func loremGather(sentences [][]string) loremDraft {
	var draft loremDraft
	for len(sentences) > 0 {
		count := min(loremDrawLength(loremParagraphMean, loremParagraphRange), len(sentences))
		draft = append(draft, sentences[:count])
		sentences = sentences[count:]
	}
	return draft
}

// render punctuates the draft and writes it out, with a blank line between
// paragraphs.
func (d loremDraft) render() string {
	opening := d.openWithLorem()

	paragraphs := make([]string, len(d))
	for i, sentences := range d {
		written := make([]string, len(sentences))
		for j, words := range sentences {
			// Only the opening words are held back from taking a comma, and
			// only in the sentence they open.
			held := 0
			if i == 0 && j == 0 {
				held = opening
			}
			written[j] = loremWriteSentence(words, held)
		}
		paragraphs[i] = strings.Join(written, " ")
	}
	return strings.Join(paragraphs, "\n\n")
}

// openWithLorem puts the opening words at the front of the first sentence and
// reports how many went in. They go in as words, before any punctuation is
// added, so that a sentence ending is never written over. Where there are fewer
// words to be had than the opening holds, as much of it as fits is all there is
// to say.
func (d loremDraft) openWithLorem() int {
	first := d[0][0]
	if len(first) < len(loremOpening) {
		d[0][0] = append([]string(nil), loremOpening[:len(first)]...)
		return len(first)
	}
	copy(first, loremOpening)
	return len(loremOpening)
}

// loremPickWords draws that many words, never the same one twice running.
func loremPickWords(length int) []string {
	words := make([]string, 0, max(length, 0))
	previous := ""
	for len(words) < length {
		var word string
		for {
			// #nosec G404 -- placeholder text, with nothing riding on the choice
			word = loremWordList[rand.IntN(len(loremWordList))]
			if word != previous {
				break
			}
		}
		words = append(words, word)
		previous = word
	}
	return words
}

// loremWriteSentence punctuates a run of words: a capital at the front, a full
// stop at the end, and now and then a comma somewhere in the middle. The first
// held words are left alone, which is how the opening reads as it should.
func loremWriteSentence(words []string, held int) string {
	written := append([]string(nil), words...)
	// The comma goes after any words being held back, and never on the last,
	// where it would sit against the full stop.
	// #nosec G404 -- placeholder text, with nothing riding on the choice
	if last := len(written) - 1; held < last && rand.Float64() < loremCommaChance {
		written[held+rand.IntN(last-held)] += "," // #nosec G404 -- as above
	}

	sentence := []rune(strings.Join(written, " "))
	sentence[0] = unicode.ToUpper(sentence[0])
	return string(sentence) + "."
}

// loremDrawSentenceLength draws how many words a sentence runs to. The first
// sentence of the text is drawn at least as long as the opening, so that the
// opening always has room and never displaces a whole sentence.
func loremDrawSentenceLength(isFirst bool) int {
	length := loremDrawLength(loremSentenceMean, loremSentenceSpread)
	if isFirst {
		return max(length, len(loremOpening))
	}
	return length
}

// loremDrawLength draws a length about the mean given. Several draws are added
// together so the result gathers about the middle, and anything that comes out
// at nothing or less is drawn again.
func loremDrawLength(mean, spread float64) int {
	for {
		var sum float64
		for range loremDraws {
			sum += rand.Float64()*2 - 1 // #nosec G404 -- placeholder text
		}
		if length := int(math.Round(sum*spread + mean)); length > 0 {
			return length
		}
	}
}

func init() { core.Register(GenerateLoremIpsum{}) }
