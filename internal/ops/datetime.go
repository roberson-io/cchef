package ops

import (
	"fmt"
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
	std := jan
	if jul < std {
		std = jul
	}
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

// momentTokenValue renders a single matched token from t.
func momentTokenValue(tok string, t time.Time, zoneName string, offsetSec int) string {
	switch tok {
	case "YYYY", "Y":
		return fmt.Sprintf("%04d", t.Year())
	case "YY":
		return fmt.Sprintf("%02d", t.Year()%100)
	case "MMMM":
		return t.Month().String()
	case "MMM":
		return t.Month().String()[:3]
	case "MM":
		return fmt.Sprintf("%02d", int(t.Month()))
	case "Mo":
		return momentOrdinal(int(t.Month()))
	case "M":
		return strconv.Itoa(int(t.Month()))
	case "DDDD":
		return fmt.Sprintf("%03d", t.YearDay())
	case "DDDo":
		return momentOrdinal(t.YearDay())
	case "DDD":
		return strconv.Itoa(t.YearDay())
	case "DD":
		return fmt.Sprintf("%02d", t.Day())
	case "Do":
		return momentOrdinal(t.Day())
	case "D":
		return strconv.Itoa(t.Day())
	case "dddd":
		return t.Weekday().String()
	case "ddd":
		return t.Weekday().String()[:3]
	case "dd":
		return momentTwoLetterDays[int(t.Weekday())]
	case "do":
		return momentOrdinal(int(t.Weekday()))
	case "d", "e":
		return strconv.Itoa(int(t.Weekday()))
	case "E":
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		return strconv.Itoa(wd)
	case "HH":
		return fmt.Sprintf("%02d", t.Hour())
	case "H":
		return strconv.Itoa(t.Hour())
	case "hh":
		return fmt.Sprintf("%02d", hour12(t))
	case "h":
		return strconv.Itoa(hour12(t))
	case "mm":
		return fmt.Sprintf("%02d", t.Minute())
	case "m":
		return strconv.Itoa(t.Minute())
	case "ss":
		return fmt.Sprintf("%02d", t.Second())
	case "s":
		return strconv.Itoa(t.Second())
	case "A":
		return meridiem(t, true)
	case "a":
		return meridiem(t, false)
	case "Z":
		return formatOffset(offsetSec, true)
	case "ZZ":
		return formatOffset(offsetSec, false)
	case "z", "zz":
		return zoneName
	case "X":
		return strconv.FormatInt(t.Unix(), 10)
	case "x":
		return strconv.FormatInt(t.UnixMilli(), 10)
	case "Q":
		return strconv.Itoa((int(t.Month())-1)/3 + 1)
	case "w":
		return strconv.Itoa(usWeekNumber(t))
	case "ww":
		return fmt.Sprintf("%02d", usWeekNumber(t))
	case "wo":
		return momentOrdinal(usWeekNumber(t))
	case "WW":
		_, wk := t.ISOWeek()
		return fmt.Sprintf("%02d", wk)
	case "Wo":
		_, wk := t.ISOWeek()
		return momentOrdinal(wk)
	case "gggg", "GGGG":
		return fmt.Sprintf("%04d", t.Year())
	case "gg", "GG":
		return fmt.Sprintf("%02d", t.Year()%100)
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
	{"YYYY", "2006"}, {"MMMM", "January"}, {"dddd", "Monday"},
	{"MMM", "Jan"}, {"ddd", "Mon"}, {"SSS", "000"},
	{"YY", "06"}, {"MM", "01"}, {"DD", "02"}, {"HH", "15"},
	{"hh", "03"}, {"mm", "04"}, {"ss", "05"}, {"ZZ", "-0700"},
	{"M", "1"}, {"D", "2"}, {"H", "15"}, {"h", "3"},
	{"m", "4"}, {"s", "5"}, {"A", "PM"}, {"a", "pm"}, {"Z", "-07:00"},
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
	if strings.TrimSpace(format) == "" {
		return momentAutoParse(input, loc)
	}
	t, err := time.ParseInLocation(momentToGoLayout(format), strings.TrimSpace(input), loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
