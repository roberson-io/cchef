package ops

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

//go:generate go run ../../tools/magicgen/gen.go

func init() {
	core.Register(Magic{})
}

// magicGuess is one brute-forced reading of the data, with the operation that
// would produce it.
type magicGuess struct {
	data []byte
	step core.RecipeOp
}

// bruteForce tries simple reversible manglings of the data — every single-byte
// exclusive-or, every bit rotation, and every character encoding — so that data
// hidden under one of them can still be recognised. Only the first hundred
// bytes are tried, as upstream does, since that is enough to recognise a shape
// and trying everything over a large input is slow.
func (m *magicRun) bruteForce(data []byte) []magicGuess {
	sample := data
	if len(sample) > magicBruteForceLen {
		sample = sample[:magicBruteForceLen]
	}

	guesses := make([]magicGuess, 0, 262)
	for key := 1; key < 256; key++ {
		out := make([]byte, len(sample))
		for i, b := range sample {
			out[i] = b ^ byte(key)
		}
		guesses = append(guesses, magicGuess{
			data: out,
			step: core.RecipeOp{Op: "XOR", Args: []any{
				core.ToggleString{Value: strconv.FormatInt(int64(key), 16), Option: "Hex"},
				"Standard", false,
			}},
		})
	}

	for by := 1; by < 8; by++ {
		out := make([]byte, len(sample))
		for i, b := range sample {
			out[i] = b>>by | b<<(8-by)
		}
		guesses = append(guesses, magicGuess{
			data: out,
			step: core.RecipeOp{Op: "Rotate right", Args: []any{float64(by), false}},
		})
	}

	return append(guesses, m.encodingGuesses(sample)...)
}

// encodingGuesses reads the sample through every character encoding, keeping
// the ones that actually change it.
func (m *magicRun) encodingGuesses(sample []byte) []magicGuess {
	op, ok := m.registry.Get("Encode text")
	if !ok {
		return nil
	}
	encodings, ok := op.Args()[0].Value.([]string)
	if !ok {
		return nil
	}

	var guesses []magicGuess
	for _, name := range []string{"Encode text", "Decode text"} {
		for _, encoding := range encodings {
			step := core.RecipeOp{Op: name, Args: []any{encoding}}
			out := m.runRecipe(core.Recipe{step}, sample)
			if len(out) == 0 || slices.Equal(out, sample) {
				continue
			}
			guesses = append(guesses, magicGuess{data: out, step: step})
		}
	}
	return guesses
}

// Magic detects what data might be and suggests recipes to make sense of it.
// Ported from CyberChef Magic.mjs and lib/Magic.mjs.
type Magic struct{}

// Meta returns the operation metadata.
func (Magic) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Magic",
		Module:      "Default",
		Description: "The Magic operation attempts to detect various properties of the input data and suggests which operations could help to make more sense of it.\n\nOptions\nDepth: If an operation appears to match the data, it will be run and the result will be analysed further. This argument controls the maximum number of levels of recursion.\n\nIntensive mode: When this is turned on, various operations like XOR, bit rotates, and character encodings are brute-forced to attempt to detect valid data underneath. To improve performance, only the first 100 bytes of the data is brute-forced.\n\nExtensive language support: At each stage, the relative byte frequencies of the data will be compared to average frequencies for a number of languages. The default set consists of ~40 of the most commonly used languages on the Internet. The extensive list consists of 284 languages and can result in many languages matching the data if their byte frequencies are similar.\n\nOptionally enter a regular expression to match a string you expect to find to filter results (crib).",
		InfoURL:     "https://github.com/gchq/CyberChef/wiki/Automatic-detection-of-encoded-data-using-CyberChef-Magic",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns how deep to look, whether to brute-force, whether to weigh the
// wider set of languages, and a string the answer is expected to contain.
func (Magic) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Depth", Type: core.ArgNumber, Integer: true, Value: 3},
		{Name: "Intensive mode", Type: core.ArgBoolean, Value: false},
		{Name: "Extensive language support", Type: core.ArgBoolean, Value: false},
		{Name: "Crib (known plaintext string or regex)", Flag: "crib", Type: core.ArgString, Value: ""},
	}
}

// magicNothingFound is what the operation says when it has nothing to offer.
const magicNothingFound = "Nothing of interest could be detected about the input data.\n" +
	"Have you tried modifying the operation arguments?"

// Run analyses the input and reports the recipes worth trying.
func (Magic) Run(in *core.Dish, args []any) (*core.Dish, error) {
	depth := int(args[0].(float64))
	run := &magicRun{
		extensive: args[2].(bool),
		intensive: args[1].(bool),
		registry:  core.Default,
	}

	if crib := args[3].(string); crib != "" {
		re, err := regexp.Compile("(?i)" + crib)
		if err != nil {
			return nil, err
		}
		run.crib = re
	}

	options := run.speculate(in.Bytes(), depth, nil, false)
	if run.crib != nil {
		kept := options[:0]
		for _, o := range options {
			if o.MatchesCrib {
				kept = append(kept, o)
			}
		}
		options = kept
	}

	return core.NewDish([]byte(magicReport(options)), core.TypeString), nil
}

// magicReport lays the candidates out as text. CyberChef shows a table of
// clickable recipe links in the browser; here each candidate is a block headed
// by the recipe in the form `cchef bake` accepts, so a promising one can be run
// by copying it.
func magicReport(options []magicOption) string {
	if len(options) == 0 {
		return magicNothingFound
	}

	var sb strings.Builder
	for i, o := range options {
		if i > 0 {
			sb.WriteString("\n")
		}
		// One operation per line, exactly as `cchef bake -e` reads a recipe, so
		// a promising suggestion can be copied straight out of the report.
		recipe := strings.TrimRight(core.GeneratePrettyRecipe(o.Recipe, true), "\n")
		if recipe == "" {
			recipe = "(the data as it is)"
		}
		fmt.Fprintf(&sb, "Recipe:\n%s\n", recipe)
		fmt.Fprintf(&sb, "  Data:     %s\n", magicEscape(o.Data))
		for _, line := range magicProperties(o) {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// magicProperties is what the report says about one candidate, in the order
// CyberChef's table shows it.
func magicProperties(o magicOption) []string {
	var lines []string

	if o.LangScores[0].Probability > 0 {
		var likely []string
		for _, l := range o.LangScores {
			if l.Probability > 0 {
				likely = append(likely, magicLanguageName(l.Lang))
			}
		}
		lines = append(lines, "Possible languages: "+strings.Join(likely, ", "))
	}
	if o.FileType != nil {
		lines = append(lines, fmt.Sprintf("File type: %s (%s)", o.FileType.mime, o.FileType.extension))
	}
	if len(o.MatchingOps) > 0 {
		var names []string
		for _, c := range o.MatchingOps {
			if !slices.Contains(names, c.Op) {
				names = append(names, c.Op)
			}
		}
		lines = append(lines, "Matching ops: "+strings.Join(names, ", "))
	}
	if o.Useful {
		lines = append(lines, "Useful op detected")
	}
	if o.IsUTF8 {
		lines = append(lines, "Valid UTF8")
	}
	return append(lines, fmt.Sprintf("Entropy: %.2f", o.Entropy))
}

// magicEscape makes a snippet safe to print on one line, showing whitespace and
// unprintable bytes rather than letting them break the report up.
func magicEscape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r == '\n':
			sb.WriteString("\\n")
		case r == '\r':
			sb.WriteString("\\r")
		case r == '\t':
			sb.WriteString("\\t")
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&sb, "\\x%02x", r)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
