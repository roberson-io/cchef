package ops

import (
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(StripHTTPHeaders{})
}

// StripHTTPHeaders removes the header block from an HTTP request or response.
type StripHTTPHeaders struct{}

// Meta returns the operation metadata.
func (StripHTTPHeaders) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Strip HTTP headers",
		Module:      "Default",
		Description: "Removes HTTP headers from a request or response by looking for the first instance of a double newline.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_HTTP_header_fields",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (StripHTTPHeaders) Args() []core.ArgDef { return nil }

// Run strips the headers. Ported from CyberChef StripHTTPHeaders.mjs.
func (StripHTTPHeaders) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	headerEnd := strings.Index(input, "\r\n\r\n")
	if headerEnd < 0 {
		headerEnd = strings.Index(input, "\n\n") + 2
	} else {
		headerEnd += 4
	}
	if headerEnd < 2 {
		return core.NewDish([]byte(input), core.TypeString), nil
	}
	return core.NewDish([]byte(input[headerEnd:]), core.TypeString), nil
}
