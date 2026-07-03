package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestTranslateDateTimeFormatFixtures transcribes CyberChef's
// TranslateDateTimeFormat.mjs cases (timezone conversion and the automatic input
// format included). The first arg is the ignored "Built in formats" populateOption.
func TestTranslateDateTimeFormatFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Translate DateTime Format", "01/04/1999 22:33:01",
			"Thursday 1st April 1999 22:33:01 +00:00 UTC",
			core.Recipe{{Op: "Translate DateTime Format", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "UTC", "dddd Do MMMM YYYY HH:mm:ss Z z", "UTC"}}}},
		{"Translate DateTime Format: timezone conversion", "01/04/1999 22:33:01",
			"Thursday 1st April 1999 17:33:01 -05:00 EST",
			core.Recipe{{Op: "Translate DateTime Format", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "UTC", "dddd Do MMMM YYYY HH:mm:ss Z z", "US/Eastern"}}}},
		{"Translate DateTime Format: automatic input format", "1999-04-01 22:33:01",
			"Thursday 1st April 1999 22:33:01 +00:00 UTC",
			core.Recipe{{Op: "Translate DateTime Format", Args: []any{
				"Automatic", "", "UTC", "dddd Do MMMM YYYY HH:mm:ss Z z", "UTC"}}}},
		// Timezone conversion to a DST-free +09:00 zone (JST).
		{"Translate DateTime Format: to Tokyo", "01/07/2024 00:00:00",
			"2024-07-01 09:00:00 +09:00 JST",
			core.Recipe{{Op: "Translate DateTime Format", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "UTC", "YYYY-MM-DD HH:mm:ss Z z", "Asia/Tokyo"}}}},
		// A format exercising most of the token set (note: moment renders a bare
		// "W" as a literal, only "WW"/"Wo" produce week numbers).
		{"Translate DateTime Format: token coverage", "2024-03-09 07:04:02",
			"Mar|Sat|Sa|6th|6|6|6|09|03|3|9|3rd|24|07|am|1709967842|1709967842000|W|10|10th|069|69th|10th|10|1",
			core.Recipe{{Op: "Translate DateTime Format", Args: []any{
				"Automatic", "", "UTC",
				"MMM|ddd|dd|do|d|e|E|DD|MM|M|D|Mo|YY|hh|a|X|x|W|WW|Wo|DDDD|DDDo|wo|ww|Q", "UTC"}}}},
	})
}

// TestTranslateDateTimeFormatInvalid checks the invalid-format path (upstream
// uses expectedMatch /Invalid format./).
func TestTranslateDateTimeFormatInvalid(t *testing.T) {
	out, err := core.Recipe{{Op: "Translate DateTime Format", Args: []any{
		"Standard date and time", "DD/MM/YYYY HH:mm:ss", "UTC", "dddd Do MMMM YYYY HH:mm:ss Z z", "UTC"}}}.
		Execute(core.NewDish([]byte("1234567890"), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out.String(), "Invalid format.") {
		t.Fatalf("got %q, want prefix %q", out.String(), "Invalid format.")
	}
}

// TestParseDateTimeOracle checks Parse DateTime against CyberChef-server output
// (v11.2.0); there is no upstream fixture. The success path emits plain text.
func TestParseDateTimeOracle(t *testing.T) {
	runCases(t, []opCase{
		{"Parse DateTime", "01/04/1999 22:33:01",
			"Date: Thursday 1st April 1999\nTime: 22:33:01\nPeriod: PM\nTimezone: UTC\nUTC offset: +0000\n\n" +
				"Daylight Saving Time: false\nLeap year: false\nDays in this month: 30\n\n" +
				"Day of year: 91\nWeek number: 14\nQuarter: 2",
			core.Recipe{{Op: "Parse DateTime", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "UTC"}}}},
		// A summer US/Eastern datetime exercises the DST-true path (EDT, -0400).
		{"Parse DateTime: US/Eastern DST", "01/07/2024 12:00:00",
			"Date: Monday 1st July 2024\nTime: 12:00:00\nPeriod: PM\nTimezone: EDT\nUTC offset: -0400\n\n" +
				"Daylight Saving Time: true\nLeap year: true\nDays in this month: 31\n\n" +
				"Day of year: 183\nWeek number: 27\nQuarter: 3",
			core.Recipe{{Op: "Parse DateTime", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "US/Eastern"}}}},
	})
}

// TestDateTimeDeltaFixtures transcribes CyberChef's DateTime.mjs Delta cases.
func TestDateTimeDeltaFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"DateTime Delta Positive", "20/02/2024 13:36:00", "20/02/2024 13:37:00",
			core.Recipe{{Op: "DateTime Delta", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "Add", 0, 0, 1, 0}}}},
		{"DateTime Delta Negative", "20/02/2024 14:37:00", "20/02/2024 13:37:00",
			core.Recipe{{Op: "DateTime Delta", Args: []any{
				"Standard date and time", "DD/MM/YYYY HH:mm:ss", "Subtract", 0, 1, 0, 0}}}},
	})
}
