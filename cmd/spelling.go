package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Operation, flag and option-value names keep CyberChef's original spellings,
// many of which are UK English. So a US-English user never has to remember
// which variant a name uses, every UK-spelled name also answers to its US
// spelling: subcommands get an alias, flags get a hidden alternative name,
// and option values are translated to the canonical choice before the
// operation sees them.

// spellingPairs are the UK/US word pairs that appear in CyberChef names, in
// both capitalisations they occur with.
var spellingPairs = [][2]string{
	{"analyse", "analyze"},
	{"Analyse", "Analyze"},
	{"colour", "color"},
	{"Colour", "Color"},
	{"centre", "center"},
	{"Centre", "Center"},
	{"co-ordinate", "coordinate"},
	{"Co-ordinate", "Coordinate"},
	{"grey", "gray"},
	{"Grey", "Gray"},
	{"metre", "meter"},
	{"Metre", "Meter"},
	{"normalise", "normalize"},
	{"Normalise", "Normalize"},
	{"randomise", "randomize"},
	{"Randomise", "Randomize"},
	{"serialise", "serialize"},
	{"Serialise", "Serialize"},
}

var (
	toUS = func() *strings.Replacer {
		var args []string
		for _, p := range spellingPairs {
			args = append(args, p[0], p[1])
		}
		return strings.NewReplacer(args...)
	}()
	toUK = func() *strings.Replacer {
		var args []string
		for _, p := range spellingPairs {
			args = append(args, p[1], p[0])
		}
		return strings.NewReplacer(args...)
	}()
)

// usSpelling and ukSpelling rewrite a name into the other spelling; a name
// containing none of the words comes back unchanged.
func usSpelling(s string) string { return toUS.Replace(s) }
func ukSpelling(s string) string { return toUK.Replace(s) }

// canonicalOption returns the defined choice that value names, accepting the
// other spelling of a choice and any casing of it; a value that names none of
// them, or several of them at once, is returned as given, for coercion to
// reject with the full list.
//
// A choice matched exactly wins over one matched loosely, so an operation whose
// choices differ only by case — To Morse Code's Dash/Dot, DASH/DOT and dash/dot
// — still selects the one asked for.
func canonicalOption(value string, opts []string) string {
	for _, o := range opts {
		if value == o || value == usSpelling(o) || value == ukSpelling(o) {
			return o
		}
	}
	found, n := "", 0
	for _, o := range opts {
		if strings.EqualFold(value, o) ||
			strings.EqualFold(value, usSpelling(o)) ||
			strings.EqualFold(value, ukSpelling(o)) {
			found, n = o, n+1
		}
	}
	if n == 1 {
		return found
	}
	return value
}

// addSpellingAliases lets every UK-spelled flag on cmd parse under its US
// spelling too; the alias does not appear in help.
func addSpellingAliases(cmd *cobra.Command) {
	aliases := map[string]string{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if us := usSpelling(f.Name); us != f.Name {
			aliases[us] = f.Name
		}
	})
	if len(aliases) == 0 {
		return
	}
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if canonical, ok := aliases[name]; ok {
			name = canonical
		}
		return pflag.NormalizedName(name)
	})
}
