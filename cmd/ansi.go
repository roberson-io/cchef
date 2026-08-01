package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

// Syntax highlighter emits HTML spans carrying hljs-* classes, which is what
// CyberChef renders in the page. A terminal cannot show that, so when the
// result is going to one the spans are rendered as ANSI color instead. The
// choice belongs to the output stream rather than to the recipe: an argument
// selecting it would travel in a generated share URL as a step CyberChef does
// not define.
//
// The flag is --ansi rather than the conventional --color because operation
// arguments keep CyberChef's spellings, two of which are named Colour. A global
// --color would take that name from them and leave one command with --colour and
// --color meaning unrelated things.

// The values --ansi accepts.
const (
	ansiAuto   = "auto"
	ansiAlways = "always"
	ansiNever  = "never"
)

// wantANSI reports whether colored output is wanted, erroring on an
// unrecognised --ansi value. An explicit "always" overrides everything, which
// is what makes `| less -R` work.
func wantANSI(cmd *cobra.Command) (bool, error) {
	switch flagANSI {
	case ansiNever:
		return false, nil
	case ansiAlways:
		return true, nil
	case ansiAuto:
		return autoANSI(flagOutput != "" && flagOutput != "-", isTerminal(cmd.OutOrStdout())), nil
	}
	return false, fmt.Errorf("--ansi must be %s, %s or %s, not %q", ansiAuto, ansiAlways, ansiNever, flagANSI)
}

// autoANSI decides the "auto" mode: color only where the result is going to an
// interactive stdout and the environment has not opted out. NO_COLOR is the
// explicit opt-out (https://no-color.org); TERM=dumb is a terminal saying it
// renders nothing but plain text.
func autoANSI(toFile, terminal bool) bool {
	return !toFile && terminal &&
		os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
}

// recipeHighlights reports whether the recipe's last enabled step produces
// highlighted HTML, which is the only output color applies to.
func recipeHighlights(recipe core.Recipe) bool {
	for i := len(recipe) - 1; i >= 0; i-- {
		if recipe[i].Disabled {
			continue
		}
		return recipe[i].Op == "Syntax highlighter"
	}
	return false
}

// hljsANSI gives each hljs class an ANSI color. The palette is the basic 16,
// which every terminal renders, and follows the conventions the highlight.js
// themes share: keywords magenta, strings green, comments dim, names blue.
var hljsANSI = map[string]string{
	"hljs-keyword":  "35",
	"hljs-built_in": "36",
	"hljs-type":     "33",
	"hljs-literal":  "36",
	"hljs-number":   "36",
	"hljs-string":   "32",
	"hljs-regexp":   "31",
	"hljs-symbol":   "36",
	"hljs-comment":  "90",
	"hljs-meta":     "90",
	"hljs-variable": "31",
	"hljs-title":    "34",
	"hljs-name":     "34",
	"hljs-attr":     "36",
	"hljs-section":  "1;34",
	"hljs-addition": "32",
	"hljs-deletion": "31",
	"hljs-strong":   "1",
	"hljs-emphasis": "3",
}

// hljsSpanRE matches one highlighted span. The spans do not nest and their text
// is HTML-escaped, so no "</span>" can occur inside one.
var hljsSpanRE = regexp.MustCompile(`(?s)<span class="([^"]*)">(.*?)</span>`)

// hljsUnescaper reverses the escaping the operation applies.
var hljsUnescaper = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&amp;", "&")

// ansiFromHLJS renders highlighted HTML as ANSI-colored text. A span whose
// class has no color is emitted as its text, so nothing is lost.
func ansiFromHLJS(html string) string {
	var b strings.Builder
	last := 0
	for _, m := range hljsSpanRE.FindAllStringSubmatchIndex(html, -1) {
		b.WriteString(hljsUnescaper.Replace(html[last:m[0]]))
		text := hljsUnescaper.Replace(html[m[4]:m[5]])
		if code, ok := hljsANSI[html[m[2]:m[3]]]; ok {
			b.WriteString("\x1b[" + code + "m")
			b.WriteString(text)
			b.WriteString("\x1b[0m")
		} else {
			b.WriteString(text)
		}
		last = m[1]
	}
	b.WriteString(hljsUnescaper.Replace(html[last:]))
	return b.String()
}
