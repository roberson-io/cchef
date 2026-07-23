package ops

import (
	"errors"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JavaScriptMinify{})
}

// JavaScriptMinify compresses JavaScript code. CyberChef wraps the npm `terser`
// library; there is no logic to port, so this uses esbuild's minifier (pure Go,
// via github.com/evanw/esbuild/pkg/api) instead. The output is therefore NOT
// byte-identical to CyberChef's — esbuild and terser use different identifier
// manglers and compression passes — but is equivalently minified.
type JavaScriptMinify struct{}

// Meta returns the operation metadata.
func (JavaScriptMinify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JavaScript Minify",
		Module:      "Code",
		Description: "Compresses JavaScript code.",
		InfoURL:     "https://github.com/evanw/esbuild",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions (none, matching CyberChef).
func (JavaScriptMinify) Args() []core.ArgDef { return nil }

// Run minifies the input with esbuild (whitespace + identifiers + syntax).
func (JavaScriptMinify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	result := api.Transform(in.String(), api.TransformOptions{
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(result.Errors) > 0 {
		return nil, errors.New("Error minifying JavaScript. (" + result.Errors[0].Text + ")")
	}
	// esbuild appends a trailing newline; terser (and thus CyberChef) does not.
	out := strings.TrimSuffix(string(result.Code), "\n")
	return core.NewDish([]byte(out), core.TypeString), nil
}
