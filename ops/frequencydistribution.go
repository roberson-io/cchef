package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(FrequencyDistribution{})
}

// FrequencyDistribution reports how often each byte value occurs.
type FrequencyDistribution struct{}

// Meta returns the operation metadata.
func (FrequencyDistribution) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Frequency distribution",
		Module:      "Default",
		Description: "Displays the distribution of bytes in the data as a graph.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FrequencyDistribution) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Show 0%s", Type: core.ArgBoolean, Value: true},
		{Name: "Show ASCII", Type: core.ArgBoolean, Value: true},
	}
}

// The characters standing in for the ones that do not print: the control
// pictures block covers the first thirty-three, and delete has its own.
const (
	freqControlPictures = 0x2400
	freqLastControl     = 32
	freqDelete          = 0x7F
	freqDeletePicture   = 0x2421
	freqPercentWidth    = 8
)

// Run reports the distribution. CyberChef draws a bar chart into a canvas
// alongside the table, which needs a browser to size and render; the table and
// the counts above it carry the same figures.
func (FrequencyDistribution) Run(in *core.Dish, args []any) (*core.Dish, error) {
	showZeroes := args[0].(bool)
	showASCII := args[1].(bool)

	data := in.Bytes()
	counts := byteCounts(data)
	represented := 0
	for _, count := range counts {
		if count > 0 {
			represented++
		}
	}

	var out strings.Builder
	fmt.Fprintf(&out, "Total data length: %d\n", len(data))
	fmt.Fprintf(&out, "Number of bytes represented: %d\n", represented)
	fmt.Fprintf(&out, "Number of bytes not represented: %d\n\n", 256-represented)

	out.WriteString(`<table class="table table-hover table-sm">` + "\n")
	out.WriteString("    <tr><th>Byte</th>")
	if showASCII {
		out.WriteString("<th>ASCII</th>")
	}
	out.WriteString("<th>Percentage</th><th></th></tr>")

	for value, count := range counts {
		if count == 0 && !showZeroes {
			continue
		}
		percentage := 0.0
		if len(data) > 0 {
			percentage = float64(count) / float64(len(data)) * 100
		}

		fmt.Fprintf(&out, "<tr><td>%02x</td>", value)
		if showASCII {
			fmt.Fprintf(&out, "<td>%c</td>", freqPrintable(value))
		}
		fmt.Fprintf(&out, "<td>%s</td>", freqPercentage(percentage))
		fmt.Fprintf(&out, "<td>%s</td></tr>", strings.Repeat("|", freqBarLength(percentage)))
	}

	out.WriteString("</table>")
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// freqPrintable gives the character shown for a byte value, standing in for the
// ones that would not print.
func freqPrintable(value int) rune {
	// #nosec G115 -- the value indexes a table of byte counts, so it is one byte
	switch {
	case value <= freqLastControl:
		return rune(freqControlPictures + value)
	case value == freqDelete:
		return freqDeletePicture
	}
	return rune(value) // #nosec G115 -- as above
}

// freqPercentage renders a percentage to two decimal places, dropping them
// where they are both zero, and pads the result to a fixed width.
func freqPercentage(percentage float64) string {
	text := strconv.FormatFloat(percentage, 'f', 2, 64)
	text = strings.Replace(text, ".00", "", 1) + "%"
	for len(text) < freqPercentWidth {
		text += " "
	}
	return text
}

// freqBarLength is how many marks stand for a percentage, one for each whole
// percent it reaches or passes.
func freqBarLength(percentage float64) int {
	length := int(percentage)
	if float64(length) < percentage {
		length++
	}
	return length
}
