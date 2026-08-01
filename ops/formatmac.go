package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(FormatMACAddresses{})
}

var (
	macSplitRe = regexp.MustCompile(`[,\s]+`)
	macCleanRe = regexp.MustCompile(`[:.-]+`)
)

// macInsertEvery inserts delim after each group of n characters, except the final
// (possibly short) group — equivalent to CyberChef's /(.{n}(?=.))/g replacement.
func macInsertEvery(s string, n int, delim string) string {
	if len(s) <= n {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i += n {
		if i > 0 {
			b.WriteString(delim)
		}
		end := min(i+n, len(s))
		b.WriteString(s[i:end])
	}
	return b.String()
}

// FormatMACAddresses reformats MAC addresses into a variety of styles.
type FormatMACAddresses struct{}

// Meta returns the operation metadata.
func (FormatMACAddresses) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Format MAC addresses",
		Module:      "Default",
		Description: "Displays given MAC addresses in multiple different formats.<br><br>Expects addresses in a list separated by newlines, spaces or commas.<br><br>WARNING: There are no validity checks.",
		InfoURL:     "https://wikipedia.org/wiki/MAC_address",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FormatMACAddresses) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Output case", Type: core.ArgOption, Value: []string{"Both", "Upper only", "Lower only"}},
		{Name: "No delimiter", Type: core.ArgBoolean, Value: true},
		{Name: "Dash delimiter", Type: core.ArgBoolean, Value: true},
		{Name: "Colon delimiter", Type: core.ArgBoolean, Value: true},
		{Name: "Cisco style", Type: core.ArgBoolean, Value: false},
		{Name: "IPv6 interface ID", Type: core.ArgBoolean, Value: false},
	}
}

// Run reformats the addresses. Ported from CyberChef FormatMACAddresses.mjs.
func (FormatMACAddresses) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	if input == "" {
		return core.NewDish([]byte(""), core.TypeString), nil
	}
	outputCase := args[0].(string)
	noDelim := args[1].(bool)
	dashDelim := args[2].(bool)
	colonDelim := args[3].(bool)
	ciscoStyle := args[4].(bool)
	ipv6IntID := args[5].(bool)

	var outputList []string
	for _, mac := range macSplitRe.Split(strings.ToLower(input), -1) {
		cleanMac := macCleanRe.ReplaceAllString(mac, "")
		macHyphen := macInsertEvery(cleanMac, 2, "-")
		macColon := macInsertEvery(cleanMac, 2, ":")
		macCisco := macInsertEvery(cleanMac, 4, ".")

		// EUI-64 IPv6 interface ID: insert fffe in the middle, colon-group, then
		// flip the universal/local bit of the first octet. jsSlice mirrors JS
		// slice's clamping so inputs shorter than 6 chars still get fffe (the
		// "fffe" guarantees at least 4 chars for the octet flip below).
		macIPv6 := jsSlice(cleanMac, 0, 6) + "fffe" + jsSlice(cleanMac, 6, len(cleanMac))
		macIPv6 = macInsertEvery(macIPv6, 4, ":")
		bite, _ := strconv.ParseUint(macIPv6[:2], 16, 16)
		macIPv6 = fmt.Sprintf("%02x", bite^2) + macIPv6[2:]

		variants := []string{cleanMac, macHyphen, macColon, macCisco, macIPv6}
		enabled := []bool{noDelim, dashDelim, colonDelim, ciscoStyle, ipv6IntID}
		for i, v := range variants {
			if !enabled[i] {
				continue
			}
			switch outputCase {
			case "Lower only":
				outputList = append(outputList, v)
			case "Upper only":
				outputList = append(outputList, strings.ToUpper(v))
			default: // Both
				outputList = append(outputList, v, strings.ToUpper(v))
			}
		}
		outputList = append(outputList, "") // empty line to delimit groups
	}
	return core.NewDish([]byte(strings.Join(outputList, "\n")), core.TypeString), nil
}
