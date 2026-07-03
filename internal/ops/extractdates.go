package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ExtractDates{})
}

// extractDatesRe matches yyyy-mm-dd, dd/mm/yyyy and mm/dd/yyyy, where each
// separator is one of - / . or space. Ported from CyberChef ExtractDates.mjs.
var extractDatesRe = regexp.MustCompile(`(?i)` +
	`(?:19|20)\d\d[- /.](?:0[1-9]|1[012])[- /.](?:0[1-9]|[12][0-9]|3[01])` + `|` +
	`(?:0[1-9]|[12][0-9]|3[01])[- /.](?:0[1-9]|1[012])[- /.](?:19|20)\d\d` + `|` +
	`(?:0[1-9]|1[012])[- /.](?:0[1-9]|[12][0-9]|3[01])[- /.](?:19|20)\d\d`)

// ExtractDates extracts dates in yyyy-mm-dd, dd/mm/yyyy and mm/dd/yyyy shapes.
type ExtractDates struct{}

// Meta returns the operation metadata.
func (ExtractDates) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Extract dates",
		Module:      "Regex",
		Description: "Extracts dates in the following formats<ul><li>yyyy-mm-dd</li><li>dd/mm/yyyy</li><li>mm/dd/yyyy</li></ul>Dividers can be any of /, -, . or space.",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractDates) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Display total", Type: core.ArgBoolean, Value: false}}
}

// Run extracts the dates. Ported from CyberChef ExtractDates.mjs.
func (ExtractDates) Run(in *core.Dish, args []any) (*core.Dish, error) {
	displayTotal := args[0].(bool)
	results := extractDatesRe.FindAllString(in.String(), -1)
	joined := strings.Join(results, "\n")
	if displayTotal {
		joined = fmt.Sprintf("Total found: %d\n\n%s", len(results), joined)
	}
	return core.NewDish([]byte(joined), core.TypeString), nil
}
