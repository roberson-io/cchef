package ops

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(FromUNIXTimestamp{})
	core.Register(ToUNIXTimestamp{})
	core.Register(GetTime{})
}

// Fixed Go layouts for the "ddd D MMMM YYYY HH:mm:ss" moment format the UNIX
// timestamp operations always render in (never a user-supplied format).
const (
	unixLayout   = "Mon 2 January 2006 15:04:05"
	unixLayoutMs = "Mon 2 January 2006 15:04:05.000"
)

// unixAutoParseLayouts are tried in order to reproduce moment's lenient default
// string parsing (ISO 8601 and close relatives).
var unixAutoParseLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05.999999999",
	"2006/01/02 15:04:05",
	"2006-01-02",
}

// momentAutoParse parses input the way moment(input)/moment.utc(input) does,
// interpreting a zone-less value as being in loc. Returns false if no layout matches.
func momentAutoParse(input string, loc *time.Location) (time.Time, bool) {
	s := strings.TrimSpace(input)
	for _, layout := range unixAutoParseLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// jsIntString formats an integer-valued float64 the way JavaScript's
// Number.prototype.toString would (plain digits below 1e21).
func jsIntString(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// FromUNIXTimestamp renders a UNIX timestamp as a human-readable UTC datetime.
type FromUNIXTimestamp struct{}

// Meta returns the operation metadata.
func (FromUNIXTimestamp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From UNIX Timestamp",
		Module:      "Default",
		Description: "Converts a UNIX timestamp to a datetime string.<br><br>e.g. <code>978346800</code> becomes <code>Mon 1 January 2001 11:00:00 UTC</code><br><br>A UNIX timestamp is a 32-bit value representing the number of seconds since January 1, 1970 UTC (the UNIX epoch).",
		InfoURL:     "https://wikipedia.org/wiki/Unix_time",
		InputType:   core.TypeNumber,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FromUNIXTimestamp) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Units", Type: core.ArgOption, Value: dateTimeUnits}}
}

// Run renders the timestamp. Ported from CyberChef FromUNIXTimestamp.mjs.
func (FromUNIXTimestamp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	units := args[0].(string)
	f, err := strconv.ParseFloat(strings.TrimSpace(in.String()), 64)
	if err != nil {
		//nolint:nilerr // returns CyberChef's "Invalid date" text as output, not an error
		return core.NewDish([]byte("Invalid date UTC"), core.TypeString), nil
	}

	var out string
	if units == "Seconds (s)" {
		sec := int64(f)
		nsec := int64((f - float64(sec)) * 1e9)
		out = time.Unix(sec, nsec).UTC().Format(unixLayout) + " UTC"
	} else {
		var ms float64
		switch units {
		case "Milliseconds (ms)":
			ms = f
		case "Microseconds (μs)":
			ms = f / 1000
		default: // Nanoseconds (ns)
			ms = f / 1000000
		}
		out = time.UnixMilli(int64(ms)).UTC().Format(unixLayoutMs) + " UTC"
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// ToUNIXTimestamp parses a datetime string into a UNIX timestamp.
type ToUNIXTimestamp struct{}

// Meta returns the operation metadata.
func (ToUNIXTimestamp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To UNIX Timestamp",
		Module:      "Default",
		Description: "Parses a datetime string in UTC and returns the corresponding UNIX timestamp.<br><br>e.g. <code>Mon 1 January 2001 11:00:00 UTC</code> becomes <code>978346800</code><br><br>A UNIX timestamp is a 32-bit value representing the number of seconds since January 1, 1970 UTC (the UNIX epoch).",
		InfoURL:     "https://wikipedia.org/wiki/Unix_time",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToUNIXTimestamp) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Units", Type: core.ArgOption, Value: dateTimeUnits},
		{Name: "Treat as UTC", Type: core.ArgBoolean, Value: true},
		{Name: "Show parsed datetime", Type: core.ArgBoolean, Value: true},
	}
}

// Run parses the datetime. Ported from CyberChef ToUNIXTimestamp.mjs.
func (ToUNIXTimestamp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	units := args[0].(string)
	treatAsUTC := args[1].(bool)
	showDateTime := args[2].(bool)

	loc := time.Local
	if treatAsUTC {
		loc = time.UTC
	}
	t, ok := momentAutoParse(in.String(), loc)
	if !ok {
		if showDateTime {
			return core.NewDish([]byte("NaN (Invalid date UTC)"), core.TypeString), nil
		}
		return core.NewDish([]byte("NaN"), core.TypeString), nil
	}

	var result string
	switch units {
	case "Seconds (s)":
		result = strconv.FormatInt(t.Unix(), 10)
	case "Milliseconds (ms)":
		result = strconv.FormatInt(t.UnixMilli(), 10)
	case "Microseconds (μs)":
		result = jsIntString(float64(t.UnixMilli()) * 1000)
	default: // Nanoseconds (ns)
		result = jsIntString(float64(t.UnixMilli()) * 1000000)
	}

	if showDateTime {
		result += " (" + t.UTC().Format(unixLayout) + " UTC)"
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}

// GetTime returns the current time in the chosen unit.
type GetTime struct{}

// Meta returns the operation metadata.
func (GetTime) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Get Time",
		Module:      "Default",
		Description: "Generates a timestamp showing the amount of time since the UNIX epoch (1970-01-01 00:00:00 UTC).",
		InfoURL:     "https://wikipedia.org/wiki/Unix_time",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (GetTime) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Granularity", Type: core.ArgOption, Value: dateTimeUnits}}
}

// Run returns the current time. Ported from CyberChef GetTime.mjs.
func (GetTime) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ns := float64(time.Now().UnixNano())
	var v float64
	switch args[0].(string) {
	case "Seconds (s)":
		v = math.Round(ns / 1e9)
	case "Milliseconds (ms)":
		v = math.Round(ns / 1e6)
	case "Microseconds (μs)":
		v = math.Round(ns / 1e3)
	default: // Nanoseconds (ns)
		v = ns
	}
	return core.NewDish([]byte(jsIntString(v)), core.TypeNumber), nil
}
