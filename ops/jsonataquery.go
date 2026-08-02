package ops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/recolabs/gnata"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

// JsonataQuery reshapes a JSON document with a JSONata expression.
type JsonataQuery struct{}

// Meta returns the operation metadata.
func (JsonataQuery) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Jsonata Query",
		Module:      "Code",
		Description: "Query and transform JSON data with a jsonata query.",
		InfoURL:     "https://docs.jsonata.org/overview.html",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JsonataQuery) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Query", Type: core.ArgString, Value: "string"},
	}
}

// Run evaluates the expression against the input document.
func (JsonataQuery) Run(in *core.Dish, args []any) (*core.Dish, error) {
	query, _ := args[0].(string)

	var document any
	if err := json.Unmarshal(in.Bytes(), &document); err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid input JSON: %w", err)
	}

	result, err := jsonataEvaluate(query, document)
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Invalid Jsonata Expression: %w", err)
	}

	// An expression that selects nothing has no result at all, which is reported
	// as the empty string rather than as nothing.
	if result == nil {
		result = ""
	}
	encoded, err := jsonval.MarshalNoEscape(result)
	if err != nil {
		return nil, err
	}
	return core.NewDish(encoded, core.TypeString), nil
}

// jsonataEvaluate compiles and runs the expression, turning a failure in either
// into an error rather than letting it unwind.
func jsonataEvaluate(query string, document any) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	expression, err := gnata.Compile(query)
	if err != nil {
		return nil, err
	}
	return expression.Eval(context.Background(), document)
}

func init() { core.Register(JsonataQuery{}) }
