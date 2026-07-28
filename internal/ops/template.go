package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

// Template renders a Handlebars template against JSON input.
type Template struct{}

// Meta returns the operation metadata.
func (Template) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Template",
		Module: "Handlebars",
		Description: "Render a template with Handlebars/Mustache substituting variables " +
			"using JSON input. Templates will be rendered to plain-text only, to prevent XSS.",
		InfoURL:    "https://handlebarsjs.com/",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (Template) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Template definition (.handlebars)", Type: core.ArgString, Value: ""},
	}
}

// Run renders the template.
func (Template) Run(in *core.Dish, args []any) (*core.Dish, error) {
	source, _ := args[0].(string)

	// The document is read keeping the order its fields were written in, since a
	// template can walk them and the order is then visible in the output.
	document, err := jsonParseOrdered(in.Bytes())
	if err != nil {
		return nil, fmt.Errorf("Error translating from ArrayBuffer to JSON: %w", err)
	}

	template, err := hbCompile(source)
	if err != nil {
		return nil, err
	}
	rendered, err := template.render(document)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(rendered), core.TypeString), nil
}

func init() { core.Register(Template{}) }
