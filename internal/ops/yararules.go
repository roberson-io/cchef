package ops

import (
	"github.com/roberson-io/cchef/internal/core"
	"github.com/roberson-io/cchef/internal/yara"
)

// YARARules matches the input against a set of YARA rules.
type YARARules struct{}

// Meta returns the operation metadata.
func (YARARules) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "YARA Rules",
		Module: "Yara",
		Description: "YARA is a tool developed at VirusTotal, primarily aimed at helping malware " +
			"researchers to identify and classify malware samples. It matches based on rules " +
			"specified by the user containing textual or binary patterns and a boolean expression.",
		InfoURL:    "https://wikipedia.org/wiki/YARA",
		InputType:  core.TypeByteArray,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (YARARules) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Rules", Type: core.ArgString, Value: ""},
		{Name: "Show strings", Type: core.ArgBoolean, Value: false},
		{Name: "Show string lengths", Type: core.ArgBoolean, Value: false},
		{Name: "Show metadata", Type: core.ArgBoolean, Value: false},
		{Name: "Show counts", Type: core.ArgBoolean, Value: true},
		{Name: "Show rule warnings", Type: core.ArgBoolean, Value: true},
		{Name: "Show console module messages", Type: core.ArgBoolean, Value: true},
	}
}

// Run matches the input against the rules.
func (YARARules) Run(in *core.Dish, args []any) (*core.Dish, error) {
	source, _ := args[0].(string)
	show := yara.Display{
		Strings:  boolArg(args[1]),
		Lengths:  boolArg(args[2]),
		Meta:     boolArg(args[3]),
		Counts:   boolArg(args[4]),
		Warnings: boolArg(args[5]),
		Console:  boolArg(args[6]),
	}

	set, err := yara.Parse(source)
	if err != nil {
		return nil, err
	}
	rules, err := yara.Compile(set)
	if err != nil {
		return nil, err
	}
	results, logs, err := rules.Scan(in.Bytes())
	if err != nil {
		return nil, err
	}
	out := yara.Report(rules.Warnings, logs, results, show)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// boolArg reads one of the flags that say how much to write out.
func boolArg(arg any) bool {
	b, _ := arg.(bool)
	return b
}

func init() {
	core.Register(YARARules{})
}
