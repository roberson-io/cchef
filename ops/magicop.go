package ops

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/magic"
)

//go:generate go run ../../tools/magicgen/gen.go

func init() {
	core.Register(Magic{})
}

// magicGuess is one brute-forced reading of the data, with the operation that
// would produce it.

// Magic detects what data might be and suggests recipes to make sense of it.
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
	run := &magic.Runner{
		Extensive: args[2].(bool),
		Intensive: args[1].(bool),
		Registry:  core.Default,
	}

	if crib := args[3].(string); crib != "" {
		re, err := regexp.Compile("(?i)" + crib)
		if err != nil {
			return nil, err
		}
		run.Crib = re
	}

	options := run.Speculate(in.Bytes(), depth, nil, false)
	if run.Crib != nil {
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
func magicReport(options []magic.Option) string {
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
func magicProperties(o magic.Option) []string {
	var lines []string

	if o.LangScores[0].Probability > 0 {
		var likely []string
		for _, l := range o.LangScores {
			if l.Probability > 0 {
				likely = append(likely, magic.LanguageName(l.Lang))
			}
		}
		lines = append(lines, "Possible languages: "+strings.Join(likely, ", "))
	}
	if o.FileType != nil {
		lines = append(lines, fmt.Sprintf("File type: %s (%s)", o.FileType.MIME, o.FileType.Extension))
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
