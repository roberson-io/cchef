package ops

import (
	"sort"
	"strings"

	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

// rakeDefaultStopWords is the list CyberChef ships, taken from the NLTK package.
const rakeDefaultStopWords = "i,me,my,myself,we,our,ours,ourselves,you,you're,you've," +
	"you'll,you'd,your,yours,yourself,yourselves,he,him,his,himself,she,she's,her,hers," +
	"herself,it,it's,its,itsef,they,them,their,theirs,themselves,what,which,who,whom," +
	"this,that,that'll,these,those,am,is,are,was,were,be,been,being,have,has,had,having," +
	"do,does',did,doing,a,an,the,and,but,if,or,because,as,until,while,of,at,by,for,with," +
	"about,against,between,into,through,during,before,after,above,below,to,from,up,down," +
	"in,out,on,off,over,under,again,further,then,once,here,there,when,where,why,how,all," +
	"any,both,each,few,more,most,other,some,such,no,nor,not,only,own,same,so,than,too," +
	"very,s,t,can,will,just,don,don't,should,should've,now,d,ll,m,o,re,ve,y,ain,aren," +
	"aren't,couldn,couldn't,didn,didn't,doesn,doesn't,hadn,hadn't,hasn,hasn't,haven," +
	"haven't,isn,isn't,ma,mightn,mightn't,mustn,mustn't,needn,needn't,shan,shan't," +
	"shouldn,shouldn't,wasn,wasn't,weren,weren't,won,won't,wouldn,wouldn't"

// RAKE picks the keywords out of a piece of text.
type RAKE struct{}

// Meta returns the operation metadata.
func (RAKE) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "RAKE",
		Module: "Default",
		Description: "Rapid Keyword Extraction (RAKE)\n<br><br>\nRAKE is a " +
			"domain-independent keyword extraction algorithm in Natural Language " +
			"Processing.\n<br><br>\nThe list of stop words are from the NLTK python package",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (RAKE) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Word Delimiter (Regex)", Type: core.ArgString, Value: `\s`},
		{Name: "Sentence Delimiter (Regex)", Type: core.ArgString, Value: `\.\s|\n`},
		{Name: "Stop Words", Type: core.ArgString, Value: rakeDefaultStopWords},
	}
}

// Run scores the phrases of the input and lists them, highest first.
func (RAKE) Run(in *core.Dish, args []any) (*core.Dish, error) {
	wordPattern, _ := args[0].(string)
	sentencePattern, _ := args[1].(string)
	stopList, _ := args[2].(string)

	wordDelim, err := regexp2.Compile(wordPattern, regexp2.None)
	if err != nil {
		return nil, err
	}
	sentenceDelim, err := regexp2.Compile(sentencePattern, regexp2.None)
	if err != nil {
		return nil, err
	}

	stop := rakeStopWords(stopList)
	tokens, frequencies, phrases := rakeReadPhrases(
		strings.TrimSpace(strings.ToLower(in.String())), wordDelim, sentenceDelim, stop)

	scored := rakeScorePhrases(tokens, frequencies, phrases)
	return core.NewDish([]byte(rakeRender(scored)), core.TypeString), nil
}

// rakeStopWords reads the words that break a sentence into phrases. The empty
// string counts as one, so that a run of delimiters does not make a phrase.
func rakeStopWords(list string) map[string]bool {
	stop := map[string]bool{"": true}
	for w := range strings.SplitSeq(strings.ReplaceAll(strings.ToLower(list), " ", ""), ",") {
		stop[w] = true
	}
	return stop
}

// rakeReadPhrases splits the text into sentences and each sentence into the runs
// of words between its stop words, counting how often each other word occurs.
func rakeReadPhrases(
	input string,
	wordDelim, sentenceDelim *regexp2.Regexp,
	stop map[string]bool,
) (tokens []string, frequencies []int, phrases [][]string) {
	index := map[string]int{}

	for _, sentence := range regexp2Split(sentenceDelim, input) {
		words := regexp2Split(wordDelim, sentence)
		start := 0
		for i, word := range words {
			if stop[word] {
				phrases = append(phrases, words[start:i])
				start = i + 1
				continue
			}
			if at, seen := index[word]; seen {
				frequencies[at]++
				continue
			}
			index[word] = len(tokens)
			tokens = append(tokens, word)
			frequencies = append(frequencies, 1)
		}
		phrases = append(phrases, words[start:])
	}

	return tokens, frequencies, rakeTidyPhrases(phrases)
}

// rakeTidyPhrases drops the empty phrases and the repeated ones.
func rakeTidyPhrases(phrases [][]string) [][]string {
	seen := map[string]bool{}
	out := make([][]string, 0, len(phrases))
	for _, p := range phrases {
		if len(p) == 0 {
			continue
		}
		key := strings.Join(p, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

// rakeScoredPhrase is one phrase and the score it was given.
type rakeScoredPhrase struct {
	score  float64
	phrase string
}

// rakeScorePhrases scores each phrase by how well connected its words are. A
// word's degree is how many words it shares a phrase with, counting itself and
// counting repeats; dividing that by how often the word occurs favours words
// that turn up in longer phrases rather than merely often.
func rakeScorePhrases(tokens []string, frequencies []int, phrases [][]string) []rakeScoredPhrase {
	index := make(map[string]int, len(tokens))
	for i, tok := range tokens {
		index[tok] = i
	}

	degrees := make([]int, len(tokens))
	for _, phrase := range phrases {
		for _, word := range phrase {
			at, ok := index[word]
			if !ok {
				continue
			}
			// Each word of the phrase contributes to this one's degree, itself
			// included.
			degrees[at] += len(phrase)
		}
	}

	scores := make([]float64, len(tokens))
	for i := range tokens {
		if frequencies[i] != 0 {
			scores[i] = float64(degrees[i]) / float64(frequencies[i])
		}
	}

	out := make([]rakeScoredPhrase, 0, len(phrases))
	for _, phrase := range phrases {
		total := 0.0
		for _, word := range phrase {
			if at, ok := index[word]; ok {
				total += scores[at]
			}
		}
		out = append(out, rakeScoredPhrase{score: total, phrase: strings.Join(phrase, " ")})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// rakeRender lists the phrases under a heading, highest score first. The shape
// is two columns so that the result can be fed to To Table.
func rakeRender(scored []rakeScoredPhrase) string {
	lines := make([]string, 0, len(scored)+1)
	lines = append(lines, "Scores: , Keywords: ")
	for _, s := range scored {
		lines = append(lines, jsnum.Format(s.score)+", "+s.phrase)
	}
	return strings.Join(lines, "\n")
}

// regexp2Split splits s on every match of re, keeping the pieces between them.
func regexp2Split(re *regexp2.Regexp, s string) []string {
	var out []string
	last := 0

	match, err := re.FindStringMatch(s)
	for err == nil && match != nil {
		if match.Length > 0 {
			out = append(out, s[last:match.Index])
			last = match.Index + match.Length
		}
		match, err = re.FindNextMatch(match)
	}
	return append(out, s[last:])
}

func init() { core.Register(RAKE{}) }
