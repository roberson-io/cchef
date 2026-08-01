package ops

import (
	"strconv"
	"strings"
	"time"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseDateTime{})
	core.Register(TranslateDateTimeFormat{})
	core.Register(DateTimeDelta{})
}

// stripAngleBrackets mirrors CyberChef's outputFormat.replace(/[<>]/g, "").
func stripAngleBrackets(s string) string {
	return strings.NewReplacer("<", "", ">", "").Replace(s)
}

// isLeapYear reports whether year is a Gregorian leap year.
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// dateTimeTimezones is the timezone option list. CyberChef offers "UTC" plus the
// full moment-timezone name list; cchef resolves names via the system tz database
// at run time, so the option is left editable rather than an enumerated list.
func dateTimeTimezoneArg(name string) core.ArgDef {
	return core.ArgDef{Name: name, Type: core.ArgEditableOption, Value: "UTC"}
}

// ParseDateTime parses a datetime string and reports its components.
type ParseDateTime struct{}

// Meta returns the operation metadata.
func (ParseDateTime) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse DateTime",
		Module:      "Default",
		Description: "Parses a DateTime string in your specified format and displays it in whichever timezone you choose with the following information:<ul><li>Date</li><li>Time</li><li>Period (AM/PM)</li><li>Timezone</li><li>UTC offset</li><li>Daylight Saving Time</li><li>Leap year</li><li>Days in this month</li><li>Day of year</li><li>Week number</li><li>Quarter</li></ul>Run with no input to see format string examples if required.",
		InfoURL:     "https://momentjs.com/docs/#/parsing/string-format/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseDateTime) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Built in formats", Type: core.ArgOption, Value: dateTimeBuiltinFormats},
		{Name: "Input format string", Type: core.ArgString, Value: "DD/MM/YYYY HH:mm:ss"},
		dateTimeTimezoneArg("Input timezone"),
	}
}

// Run parses and describes the datetime. Ported from CyberChef ParseDateTime.mjs.
func (ParseDateTime) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[1].(string)
	loc := loadMomentZone(args[2].(string))

	t, ok := momentParse(in.String(), inputFormat, loc)
	if !ok {
		return core.NewDish([]byte("Invalid format.\n\n"+formatExamples), core.TypeString), nil
	}

	var b strings.Builder
	b.WriteString("Date: " + momentFormat(t, "dddd Do MMMM YYYY"))
	b.WriteString("\nTime: " + momentFormat(t, "HH:mm:ss"))
	b.WriteString("\nPeriod: " + momentFormat(t, "A"))
	b.WriteString("\nTimezone: " + momentFormat(t, "z"))
	b.WriteString("\nUTC offset: " + momentFormat(t, "ZZ"))
	b.WriteString("\n\nDaylight Saving Time: " + strconv.FormatBool(isDST(t)))
	b.WriteString("\nLeap year: " + strconv.FormatBool(isLeapYear(t.Year())))
	b.WriteString("\nDays in this month: " + strconv.Itoa(daysInMonth(t)))
	b.WriteString("\n\nDay of year: " + strconv.Itoa(t.YearDay()))
	b.WriteString("\nWeek number: " + strconv.Itoa(usWeekNumber(t)))
	b.WriteString("\nQuarter: " + strconv.Itoa((int(t.Month())-1)/3+1))
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// TranslateDateTimeFormat reformats a datetime string between formats/timezones.
type TranslateDateTimeFormat struct{}

// Meta returns the operation metadata.
func (TranslateDateTimeFormat) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Translate DateTime Format",
		Module:      "Default",
		Description: "Parses a datetime string in one format and re-writes it in another.<br><br>Run with no input to see the relevant format string examples.",
		InfoURL:     "https://momentjs.com/docs/#/parsing/string-format/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (TranslateDateTimeFormat) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Built in formats", Type: core.ArgOption, Value: dateTimeBuiltinFormats},
		{Name: "Input format string", Type: core.ArgString, Value: "DD/MM/YYYY HH:mm:ss"},
		dateTimeTimezoneArg("Input timezone"),
		{Name: "Output format string", Type: core.ArgString, Value: "dddd Do MMMM YYYY HH:mm:ss Z z"},
		dateTimeTimezoneArg("Output timezone"),
	}
}

// Run reformats the datetime. Ported from CyberChef TranslateDateTimeFormat.mjs.
func (TranslateDateTimeFormat) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[1].(string)
	inputTZ := args[2].(string)
	outputFormat := args[3].(string)
	outputTZ := args[4].(string)

	t, ok := momentParse(in.String(), inputFormat, loadMomentZone(inputTZ))
	if !ok {
		return core.NewDish([]byte("Invalid format."), core.TypeString), nil
	}
	out := momentFormat(t.In(loadMomentZone(outputTZ)), stripAngleBrackets(outputFormat))
	return core.NewDish([]byte(out), core.TypeString), nil
}

// DateTimeDelta adds or subtracts a time delta from a datetime string.
type DateTimeDelta struct{}

// Meta returns the operation metadata.
func (DateTimeDelta) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "DateTime Delta",
		Module:      "Default",
		Description: "Calculates a new DateTime value given an input DateTime value and a time difference (delta) from the input DateTime value.",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DateTimeDelta) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Built in formats", Type: core.ArgOption, Value: dateTimeBuiltinFormats},
		{Name: "Input format string", Type: core.ArgString, Value: "DD/MM/YYYY HH:mm:ss"},
		{Name: "Time Operation", Type: core.ArgOption, Value: []string{"Add", "Subtract"}},
		{Name: "Days", Type: core.ArgNumber, Value: 0},
		{Name: "Hours", Type: core.ArgNumber, Value: 0},
		{Name: "Minutes", Type: core.ArgNumber, Value: 0},
		{Name: "Seconds", Type: core.ArgNumber, Value: 0},
	}
}

// Run applies the delta. Ported from CyberChef DateTimeDelta.mjs.
func (DateTimeDelta) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[1].(string)
	operationType := args[2].(string)
	days := int(args[3].(float64))
	hours := int(args[4].(float64))
	minutes := int(args[5].(float64))
	seconds := int(args[6].(float64))

	// CyberChef fixes the input timezone to UTC.
	t, ok := momentParse(in.String(), inputFormat, time.UTC)
	if !ok {
		return core.NewDish([]byte("Invalid format.\n\n"+formatExamples), core.TypeString), nil
	}

	sign := 1
	if operationType != "Add" {
		sign = -1
	}
	newT := t.AddDate(0, 0, sign*days).Add(time.Duration(sign) *
		(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second))
	out := momentFormat(newT, stripAngleBrackets(inputFormat))
	return core.NewDish([]byte(out), core.TypeString), nil
}
