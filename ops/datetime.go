package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// dateTimeUnits mirrors CyberChef lib/DateTime.mjs UNITS.
var dateTimeUnits = []string{"Seconds (s)", "Milliseconds (ms)", "Microseconds (μs)", "Nanoseconds (ns)"}

// dateTimeBuiltinFormats are the names of CyberChef's DATETIME_FORMATS. In
// CyberChef this is a populateOption that fills the format field in the UI; in a
// recipe it is passed but ignored by the operation (which reads the resolved
// format string from the next argument).
var dateTimeBuiltinFormats = []string{
	"Standard date and time",
	"American-style date and time",
	"International date and time",
	"Verbose date and time",
	"UNIX timestamp (seconds)",
	"UNIX timestamp offset (milliseconds)",
	"Automatic",
}

// loadMomentZone resolves a moment-timezone name to a Go *time.Location. Unknown
// names fall back to UTC (moment likewise degrades rather than erroring).
func loadMomentZone(name string) *time.Location {
	if name == "" || name == "UTC" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.UTC
}

// momentOrdinal renders n with its English ordinal suffix (1st, 2nd, 3rd, 4th…),
// matching moment's Do/Mo/DDDo tokens.
func momentOrdinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

// usWeekNumber is moment's default (en-US locale) week-of-year: weeks start on
// Sunday and week 1 is the week containing 1 January.
func usWeekNumber(t time.Time) int {
	return (t.YearDay()+6-int(t.Weekday()))/7 + 1
}

// daysInMonth returns the number of days in t's month.
func daysInMonth(t time.Time) int {
	return time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, t.Location()).Day()
}

// isDST reports whether t is in daylight saving time, by comparing its UTC offset
// to the standard (smaller) offset seen in January and July of the same year.
func isDST(t time.Time) bool {
	_, cur := t.Zone()
	_, jan := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()).Zone()
	_, jul := time.Date(t.Year(), 7, 1, 0, 0, 0, 0, t.Location()).Zone()
	std := min(jul, jan)
	return cur != std
}

// momentTwoLetterDays are moment's "dd" token outputs, indexed by Go weekday
// (0 = Sunday).
var momentTwoLetterDays = [...]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

// momentFormatTokens lists the multi-character formatting tokens longest-first so
// the walker matches greedily (e.g. "MMMM" before "MM" before "M").
var momentFormatTokens = []string{
	"dddd", "DDDo", "DDDD", "MMMM", "GGGG", "gggg",
	"ddd", "DDD", "MMM", "Do", "Mo", "Wo", "wo", "do",
	"YYYY", "YY", "MM", "DD", "dd", "HH", "hh", "mm", "ss",
	"ZZ", "zz", "WW", "ww", "GG", "gg",
	"M", "D", "d", "e", "E", "H", "h", "m", "s", "A", "a",
	"Z", "z", "X", "x", "Q", "w", "Y",
	// NB: the bare "W" token is intentionally omitted — moment renders it as a
	// literal "W" (only "WW"/"Wo" produce ISO week numbers), so leaving it out
	// makes it fall through to a literal, matching CyberChef.
}

// momentFormat renders t using a moment.js format string. It supports the token
// set in CyberChef's FORMAT_EXAMPLES for the en-US locale. Fidelity gaps (noted
// in the docs): week-year tokens (gg/GG) approximate to the calendar year, and
// the zone abbreviation (z) is whatever the Go tz database reports.
func momentFormat(t time.Time, format string) string {
	zoneName, offsetSec := t.Zone()
	var b strings.Builder
	for i := 0; i < len(format); {
		c := format[i]
		// Literal text in [brackets].
		if c == '[' {
			if end := strings.IndexByte(format[i:], ']'); end >= 0 {
				b.WriteString(format[i+1 : i+end])
				i += end + 1
				continue
			}
		}
		// A run of S is fractional seconds of that width.
		if c == 'S' {
			n := 0
			for i+n < len(format) && format[i+n] == 'S' {
				n++
			}
			frac := fmt.Sprintf("%09d", t.Nanosecond())
			if n <= 9 {
				b.WriteString(frac[:n])
			} else {
				b.WriteString(frac + strings.Repeat("0", n-9))
			}
			i += n
			continue
		}
		if tok := matchMomentToken(format, i); tok != "" {
			b.WriteString(momentTokenValue(tok, t, zoneName, offsetSec))
			i += len(tok)
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// matchMomentToken returns the longest formatting token at format[i], or "".
func matchMomentToken(format string, i int) string {
	for _, tok := range momentFormatTokens {
		if strings.HasPrefix(format[i:], tok) {
			return tok
		}
	}
	return ""
}

// momentCtx carries the values a moment.js format token may reference.
type momentCtx struct {
	t         time.Time
	zoneName  string
	offsetSec int
}

// The moment-token handlers mostly share three shapes: take an int (or string)
// extracted from the time, then zero-pad it, base-10 it, or ordinal it. These
// builders turn a field extractor into a token handler so the table below reads
// declaratively, e.g. `momentPad(2, time.Time.Hour)` = "hour, zero-padded to 2".

// momentPad renders an int field zero-padded to width.
func momentPad(width int, get func(time.Time) int) func(momentCtx) string {
	return func(c momentCtx) string { return fmt.Sprintf("%0*d", width, get(c.t)) }
}

// momentNum renders an int field in base 10.
func momentNum(get func(time.Time) int) func(momentCtx) string {
	return func(c momentCtx) string { return strconv.Itoa(get(c.t)) }
}

// momentOrd renders an int field as an English ordinal (1st, 2nd, ...).
func momentOrd(get func(time.Time) int) func(momentCtx) string {
	return func(c momentCtx) string { return momentOrdinal(get(c.t)) }
}

// momentText renders a string field.
func momentText(get func(time.Time) string) func(momentCtx) string {
	return func(c momentCtx) string { return get(c.t) }
}

// momentInt64 renders an int64 field (epoch tokens) in base 10.
func momentInt64(get func(time.Time) int64) func(momentCtx) string {
	return func(c momentCtx) string { return strconv.FormatInt(get(c.t), 10) }
}

// momentMeridiem renders AM/PM (upper) or am/pm (lower).
func momentMeridiem(upper bool) func(momentCtx) string {
	return func(c momentCtx) string { return meridiem(c.t, upper) }
}

// momentOffset renders the UTC offset with (colon) or without a colon. It reads
// momentCtx.offsetSec rather than the time, so it can't use momentText.
func momentOffset(colon bool) func(momentCtx) string {
	return func(c momentCtx) string { return formatOffset(c.offsetSec, colon) }
}

// momentZone renders the zone abbreviation from momentCtx.
func momentZone(c momentCtx) string { return c.zoneName }

// Field extractors for values a method expression can't express directly.
func mMonth(t time.Time) int           { return int(t.Month()) }
func mWeekday(t time.Time) int         { return int(t.Weekday()) }
func mYear2(t time.Time) int           { return t.Year() % 100 }
func mQuarter(t time.Time) int         { return (int(t.Month())-1)/3 + 1 }
func mMonthName(t time.Time) string    { return t.Month().String() }
func mMonthAbbr(t time.Time) string    { return t.Month().String()[:3] }
func mDayName(t time.Time) string      { return t.Weekday().String() }
func mDayAbbr(t time.Time) string      { return t.Weekday().String()[:3] }
func mDayTwoLetter(t time.Time) string { return momentTwoLetterDays[int(t.Weekday())] }
func mISOWeek(t time.Time) int         { _, w := t.ISOWeek(); return w }

// mISOWeekday is the day of week numbered Monday=1..Sunday=7 (moment's "E").
func mISOWeekday(t time.Time) int {
	if wd := int(t.Weekday()); wd != 0 {
		return wd
	}
	return 7
}

// momentTokenFns maps each supported moment.js format token to the function that
// renders it. Each entry declares its field and formatting; tokens that share a
// rendering appear once per key.
var momentTokenFns = map[string]func(momentCtx) string{
	// Year.
	"YYYY": momentPad(4, time.Time.Year),
	"Y":    momentPad(4, time.Time.Year),
	"YY":   momentPad(2, mYear2),
	"gggg": momentPad(4, time.Time.Year),
	"GGGG": momentPad(4, time.Time.Year),
	"gg":   momentPad(2, mYear2),
	"GG":   momentPad(2, mYear2),

	// Month.
	"MMMM": momentText(mMonthName),
	"MMM":  momentText(mMonthAbbr),
	"MM":   momentPad(2, mMonth),
	"Mo":   momentOrd(mMonth),
	"M":    momentNum(mMonth),
	"Q":    momentNum(mQuarter),

	// Day of year / month.
	"DDDD": momentPad(3, time.Time.YearDay),
	"DDDo": momentOrd(time.Time.YearDay),
	"DDD":  momentNum(time.Time.YearDay),
	"DD":   momentPad(2, time.Time.Day),
	"Do":   momentOrd(time.Time.Day),
	"D":    momentNum(time.Time.Day),

	// Day of week.
	"dddd": momentText(mDayName),
	"ddd":  momentText(mDayAbbr),
	"dd":   momentText(mDayTwoLetter),
	"do":   momentOrd(mWeekday),
	"d":    momentNum(mWeekday),
	"e":    momentNum(mWeekday),
	"E":    momentNum(mISOWeekday),

	// Time.
	"HH": momentPad(2, time.Time.Hour),
	"H":  momentNum(time.Time.Hour),
	"hh": momentPad(2, hour12),
	"h":  momentNum(hour12),
	"mm": momentPad(2, time.Time.Minute),
	"m":  momentNum(time.Time.Minute),
	"ss": momentPad(2, time.Time.Second),
	"s":  momentNum(time.Time.Second),
	"A":  momentMeridiem(true),
	"a":  momentMeridiem(false),

	// Zone / epoch.
	"Z":  momentOffset(true),
	"ZZ": momentOffset(false),
	"z":  momentZone,
	"zz": momentZone,
	"X":  momentInt64(time.Time.Unix),
	"x":  momentInt64(time.Time.UnixMilli),

	// Week of year.
	"w":  momentNum(usWeekNumber),
	"ww": momentPad(2, usWeekNumber),
	"wo": momentOrd(usWeekNumber),
	"WW": momentPad(2, mISOWeek),
	"Wo": momentOrd(mISOWeek),
}

// momentTokenValue renders one moment.js format token, or returns the token
// unchanged when it is not a recognised token (literal text).
func momentTokenValue(tok string, t time.Time, zoneName string, offsetSec int) string {
	if fn, ok := momentTokenFns[tok]; ok {
		return fn(momentCtx{t: t, zoneName: zoneName, offsetSec: offsetSec})
	}
	return tok
}

func hour12(t time.Time) int {
	h := t.Hour() % 12
	if h == 0 {
		h = 12
	}
	return h
}

func meridiem(t time.Time, upper bool) string {
	if t.Hour() < 12 {
		if upper {
			return "AM"
		}
		return "am"
	}
	if upper {
		return "PM"
	}
	return "pm"
}

// formatOffset renders a UTC offset in seconds as ±HH:MM (colon) or ±HHMM.
func formatOffset(offsetSec int, colon bool) string {
	sign := "+"
	if offsetSec < 0 {
		sign = "-"
		offsetSec = -offsetSec
	}
	hh := offsetSec / 3600
	mm := (offsetSec % 3600) / 60
	if colon {
		return fmt.Sprintf("%s%02d:%02d", sign, hh, mm)
	}
	return fmt.Sprintf("%s%02d%02d", sign, hh, mm)
}

// momentParseLayoutTokens maps moment parsing tokens to Go reference-time layout
// fragments, longest-first.
var momentParseLayoutTokens = []struct{ moment, golang string }{
	{"YYYY", "2006"},
	{"MMMM", "January"},
	{"dddd", "Monday"},
	{"MMM", "Jan"},
	{"ddd", "Mon"},
	{"SSS", "000"},
	{"YY", "06"},
	{"MM", "01"},
	{"DD", "02"},
	{"HH", "15"},
	{"hh", "03"},
	{"mm", "04"},
	{"ss", "05"},
	{"ZZ", "-0700"},
	{"M", "1"},
	{"D", "2"},
	{"H", "15"},
	{"h", "3"},
	{"m", "4"},
	{"s", "5"},
	{"A", "PM"},
	{"a", "pm"},
	{"Z", "-07:00"},
}

// momentToGoLayout converts a moment parse format into a Go time layout. Tokens
// it does not recognise pass through as literals.
func momentToGoLayout(format string) string {
	var b strings.Builder
	for i := 0; i < len(format); {
		if format[i] == '[' {
			if end := strings.IndexByte(format[i:], ']'); end >= 0 {
				b.WriteString(format[i+1 : i+end])
				i += end + 1
				continue
			}
		}
		matched := false
		for _, m := range momentParseLayoutTokens {
			if strings.HasPrefix(format[i:], m.moment) {
				b.WriteString(m.golang)
				i += len(m.moment)
				matched = true
				break
			}
		}
		if !matched {
			b.WriteByte(format[i])
			i++
		}
	}
	return b.String()
}

// momentParse parses input per a moment format string in loc. An empty format
// triggers moment's lenient auto-parsing. Returns false when the value is invalid.
func momentParse(input, format string, loc *time.Location) (time.Time, bool) {
	f := strings.TrimSpace(format)
	switch {
	case f == "":
		return momentAutoParse(input, loc)
	// moment's "X"/"x" are single-character tokens whose regex greedily consumes
	// the leading timestamp, so any trailing format tokens (e.g. the ".SSS" of
	// the Squid log format "X.SSS") match nothing and a format beginning with
	// "X"/"x" behaves as the bare token.
	case strings.HasPrefix(f, "X"): // UNIX timestamp in seconds
		return unixParse(input, false)
	case strings.HasPrefix(f, "x"): // UNIX timestamp in milliseconds
		return unixParse(input, true)
	}
	t, err := time.ParseInLocation(momentToGoLayout(f), strings.TrimSpace(input), loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// unixSecondsToken and unixMillisToken mirror moment's parse regexes: "X" reads
// matchTimestamp (up to three fractional digits, i.e. milliseconds) and "x"
// reads matchSigned (a signed integer).
var (
	unixSecondsToken = regexp.MustCompile(`^[+-]?\d+(\.\d{1,3})?`)
	unixMillisToken  = regexp.MustCompile(`^[+-]?\d+`)
)

// unixParse reads a leading UNIX timestamp token. It reproduces moment exactly:
// "X" is new Date(parseFloat(input)*1000) and "x" is new Date(toInt(input)), so
// both resolve to whole milliseconds truncated toward zero the way ECMAScript's
// TimeClip does. Working in milliseconds (not nanoseconds) is what keeps a
// fractional value like "1609459200.123" from losing a millisecond to float error.
func unixParse(input string, millis bool) (time.Time, bool) {
	s := strings.TrimSpace(input)
	if millis {
		tok := unixMillisToken.FindString(s)
		v, err := strconv.ParseInt(tok, 10, 64)
		if err != nil {
			return time.Time{}, false
		}
		return time.UnixMilli(v).UTC(), true
	}
	tok := unixSecondsToken.FindString(s)
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(f * 1000)).UTC(), true
}
