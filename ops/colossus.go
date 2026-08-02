package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsonval"
)

func init() {
	core.Register(Colossus{})
}

// Colossus / Lorenz shared argument option lists. The Greek letters Χ (U+03A7)
// and Ψ (U+03A8) are used exactly as in CyberChef; note the Limitation list
// mixes Greek "Χ2" with Latin "X2 + Ψ1" verbatim, and the limitation test below
// keys on the Greek forms — so "X2 + Ψ1" (Latin X) does NOT enable the Χ2
// limitation. This quirk is preserved.
var (
	colPatterns   = []string{"KH Pattern", "ZMUG Pattern", "BREAM Pattern"}
	colQBusZ      = []string{"", "Z", "ΔZ"}
	colQBusChi    = []string{"", "Χ", "ΔΧ"}
	colQBusPsi    = []string{"", "Ψ", "ΔΨ"}
	colLimitation = []string{"None", "Χ2", "Χ2 + P5", "X2 + Ψ1", "X2 + Ψ1 + P5"}
	colKRackOpt   = []string{"Select Program", "Top Section - Conditional", "Bottom Section - Addition", "Advanced"}
	colProgram    = []string{"", "Letter Count", "1+2=. (1+2 Break In, Find X1,X2)", "4=5=/1=2 (Given X1,X2 find X4,X5)", "/,5,U (Count chars to find X3)"}
	colCounter    = []string{"", "1", "2", "3", "4", "5"}
	colStep       = []string{"", "X1", "X2", "X3", "X4", "X5", "M37", "M61", "S1", "S2", "S3", "S4", "S5"}
)

// Colossus emulates the WW2 Colossus codebreaking computer.
type Colossus struct{}

// Meta returns the operation metadata.
func (Colossus) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Colossus",
		Module:      "Bletchley",
		Description: "Colossus is the name of the world's first electronic computer. Ten Colossi were designed by Tommy Flowers and built at the Post Office Research Labs at Dollis Hill in 1943 during World War 2. They assisted with the breaking of the German Lorenz cipher attachment, a machine created to encipher communications between Hitler and his generals on the front lines.",
		InfoURL:     "https://wikipedia.org/wiki/Colossus_computer",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// switchDef is an editableOptionShort switch (blank/./x), defaulting to blank.
func switchDef(name string) core.ArgDef {
	return core.ArgDef{Name: name, Type: core.ArgEditableOption, Value: ""}
}

// Args returns the argument definitions. The 57-slot layout matches CyberChef's
// args array index-for-index; the four "label" slots are display-only empty
// strings so downstream indices line up with the original operation.
func (Colossus) Args() []core.ArgDef {
	defs := []core.ArgDef{
		{Name: "Input", Type: core.ArgString, Value: ""}, // 0 label
		{Name: "Pattern", Type: core.ArgOption, Value: colPatterns},
		{Name: "QBusZ", Type: core.ArgOption, Value: colQBusZ},
		{Name: "QBusΧ", Type: core.ArgOption, Value: colQBusChi},
		{Name: "QBusΨ", Type: core.ArgOption, Value: colQBusPsi},
		{Name: "Limitation", Type: core.ArgOption, Value: colLimitation},
		{Name: "K Rack Option", Type: core.ArgOption, Value: colKRackOpt},
		{Name: "Program to run", Type: core.ArgOption, Value: colProgram},
		{Name: "K Rack: Conditional", Type: core.ArgString, Value: ""}, // 8 label
	}
	// Conditional rows R1-R3: Q1-Q5, Negate, Counter.
	for r := 1; r <= 3; r++ {
		for q := 1; q <= 5; q++ {
			defs = append(defs, switchDef(fmt.Sprintf("R%d-Q%d", r, q)))
		}
		defs = append(defs,
			core.ArgDef{Name: fmt.Sprintf("R%d-Negate", r), Type: core.ArgBoolean, Value: false},
			core.ArgDef{Name: fmt.Sprintf("R%d-Counter", r), Type: core.ArgOption, Value: colCounter})
	}
	defs = append(defs,
		core.ArgDef{Name: "Negate All", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "K Rack: Addition", Type: core.ArgString, Value: ""}, // 31 label
		core.ArgDef{Name: "Add-Q1", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "Add-Q2", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "Add-Q3", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "Add-Q4", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "Add-Q5", Type: core.ArgBoolean, Value: false},
		switchDef("Add-Equals"),
		core.ArgDef{Name: "Add-Counter1", Type: core.ArgBoolean, Value: false},
		core.ArgDef{Name: "Add Negate All", Type: core.ArgBoolean, Value: false},
		switchDef("Total Motor"),
		core.ArgDef{Name: "Master Control Panel", Type: core.ArgString, Value: ""}, // 41 label
		core.ArgDef{Name: "Set Total", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		core.ArgDef{Name: "Fast Step", Type: core.ArgOption, Value: colStep},
		core.ArgDef{Name: "Slow Step", Type: core.ArgOption, Value: colStep})
	// Start positions Χ1-5, M61, M37, Ψ1-5.
	for _, name := range []string{
		"Start Χ1", "Start Χ2", "Start Χ3", "Start Χ4", "Start Χ5",
		"Start M61", "Start M37", "Start Ψ1", "Start Ψ2", "Start Ψ3", "Start Ψ4", "Start Ψ5",
	} {
		defs = append(defs, core.ArgDef{Name: name, Type: core.ArgNumber, Value: float64(1)})
	}
	return defs
}

// Run emulates a Colossus run.
func (Colossus) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := strings.ToUpper(in.String())
	if err := colossusValidateInput(input); err != nil {
		return nil, err
	}

	// Applying a selected program rewrites the switch args, mirroring the .mjs.
	if args[6].(string) == "Select Program" && args[7].(string) != "" {
		args = colossusSelectProgram(args[7].(string), append([]any(nil), args...))
	}
	if err := colossusValidateArgs(args); err != nil {
		return nil, err
	}

	c := newColossusComputer(input, args)
	printout, counters, runcount := c.run()

	out := jsonval.NewOMap()
	out.Set("printout", printout)
	out.Set("counters", counters[:])
	out.Set("runcount", runcount)
	b, err := jsonval.MarshalOMap(out)
	if err != nil {
		return nil, err
	}
	return core.NewDish(b, core.TypeJSON), nil
}

// colossusValidateInput ensures every character is a valid ITA2 letter.
func colossusValidateInput(input string) error {
	for _, ch := range input {
		if !strings.ContainsRune(validITA2, ch) {
			errltr := string(ch)
			switch ch {
			case '\n':
				errltr = "Carriage Return"
			case ' ':
				errltr = "Space"
			}
			return fmt.Errorf("Invalid ITA2 character : %s", errltr) //nolint:staticcheck,revive // verbatim CyberChef message
		}
	}
	return nil
}

// colSwitchOK reports whether a bus switch value is blank, "." or "x".
func colSwitchOK(v string) bool { return v == "" || v == "." || v == "x" }

// colossusValidateArgs checks the switch values, Set Total and rotor starts,
// producing CyberChef's verbatim error messages.
func colossusValidateArgs(args []any) error {
	for qr := range 3 {
		for a := range 5 {
			if !colSwitchOK(args[(qr*7)+(a+9)].(string)) {
				return fmt.Errorf("Switch R%d-Q%d can only be set to blank, . or x", qr+1, a+1) //nolint:staticcheck,revive // verbatim CyberChef message
			}
		}
	}
	if !colSwitchOK(args[37].(string)) {
		return fmt.Errorf("Switch Add-Equals can only be set to blank, . or x") //nolint:staticcheck,revive // verbatim CyberChef message
	}
	if !colSwitchOK(args[40].(string)) {
		return fmt.Errorf("Switch Total Motor can only be set to blank, . or x") //nolint:staticcheck,revive // verbatim CyberChef message
	}
	if st := colNum(args[42]); st < 0 || st > 9999 {
		return fmt.Errorf("Set Total must be between 0000 and 9999") //nolint:staticcheck,revive // verbatim CyberChef message
	}
	return colossusValidateStarts(args)
}

// startBound pairs a start-position arg index with its rotor's valid range and
// the (Greek-lettered) label CyberChef uses in the error message.
type startBound struct {
	idx   int
	label string
	max   int
}

// colStartBounds lists the rotor-start range checks in the same order (and with
// the same Χ/Μ/Ψ labels) as CyberChef's run().
var colStartBounds = []startBound{
	{52, "Ψ1", 43},
	{53, "Ψ2", 47},
	{54, "Ψ3", 51},
	{55, "Ψ4", 53},
	{56, "Ψ5", 59},
	{51, "Μ37", 37},
	{50, "Μ61", 61},
	{45, "Χ1", 41},
	{46, "Χ2", 31},
	{47, "Χ3", 29},
	{48, "Χ4", 26},
	{49, "Χ5", 23},
}

// colossusValidateStarts range-checks each rotor start position.
func colossusValidateStarts(args []any) error {
	for _, b := range colStartBounds {
		if v := colNum(args[b.idx]); v < 1 || v > b.max {
			return fmt.Errorf("%s start must be between 1 and %d", b.label, b.max) //nolint:staticcheck,revive // verbatim CyberChef message
		}
	}
	return nil
}

// colNum coerces a numeric argument (float64 from CoerceArgs) to an int.
func colNum(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

// colossusSelectProgram rewrites the switch args for a named preset program.
func colossusSelectProgram(prog string, args []any) []any {
	switch prog {
	case "Letter Count":
		args[9], args[10], args[11], args[12], args[13] = "", "", "", "", ""
		args[14], args[15] = false, "1"
		args[22], args[29] = "", ""
		args[30] = false
		args[38] = false
	case "1+2=. (1+2 Break In, Find X1,X2)":
		args[15], args[22], args[29] = "", "", ""
		args[32], args[33] = true, true
		args[34], args[35], args[36] = false, false, false
		args[37], args[38] = ".", true
	case "4=5=/1=2 (Given X1,X2 find X4,X5)":
		args[9], args[10], args[11], args[12], args[13] = ".", ".", "", ".", "."
		args[14], args[15] = true, "1"
		args[16], args[17], args[18], args[19], args[20] = "x", "x", "", "x", "x"
		args[21], args[22] = true, "1"
		args[29] = ""
		args[30] = true
		args[38] = false
	case "/,5,U (Count chars to find X3)":
		args[9], args[10], args[11], args[12], args[13] = ".", ".", ".", ".", "."
		args[14], args[15] = false, "1"
		args[16], args[17], args[18], args[19], args[20] = "x", "x", ".", "x", "x"
		args[21], args[22] = false, "2"
		args[23], args[24], args[25], args[26], args[27] = "x", "x", "x", ".", "."
		args[28], args[29] = false, "3"
		args[30] = false
		args[38] = false
	}
	return args
}
