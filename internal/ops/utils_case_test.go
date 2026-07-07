package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Get All Casings cases transcribed from CyberChef GetAllCasings.mjs.
func TestGetAllCasings(t *testing.T) {
	runCases(t, []opCase{
		{
			"All casings of test", "test",
			"test\nTest\ntEst\nTEst\nteSt\nTeSt\ntESt\nTESt\ntesT\nTesT\ntEsT\nTEsT\nteST\nTeST\ntEST\nTEST",
			core.Recipe{{Op: "Get All Casings"}},
		},
		{
			"All casings of t", "t", "t\nT",
			core.Recipe{{Op: "Get All Casings"}},
		},
		{
			"All casings of nothing", "", "",
			core.Recipe{{Op: "Get All Casings"}},
		},
	})
}

// Unescape string cases transcribed from CyberChef UnescapeString.mjs.
func TestUnescapeString(t *testing.T) {
	runCases(t, []opCase{
		{
			"Escape sequences", `\a\b\f\n\r\t\v\'\"`,
			"\x07\x08\x0c\n\r\t\x0b'\"",
			core.Recipe{{Op: "Unescape string"}},
		},
		{
			"Octals", `\0\01\012\1\12`, "\x00\x01\x0a\x01\x0a",
			core.Recipe{{Op: "Unescape string"}},
		},
		{
			"Hexadecimals", `\x00\xAA\xaa`, "\x00ªª",
			core.Recipe{{Op: "Unescape string"}},
		},
		{
			"Unicode", `A\u{1F600}`, "A\U0001F600",
			core.Recipe{{Op: "Unescape string"}},
		},
	})
}

func TestCaseInsensitiveRegex(t *testing.T) {
	runCases(t, []opCase{
		{
			"To CI Regex letters", "test", "[tT][eE][sS][tT]",
			core.Recipe{{Op: "To Case Insensitive Regex"}},
		},
		{
			"To CI Regex range", "[A-Z]", "[A-Za-z]",
			core.Recipe{{Op: "To Case Insensitive Regex"}},
		},
		{
			"To CI Regex lowercase range", "[a-z]", "[A-Za-z]",
			core.Recipe{{Op: "To Case Insensitive Regex"}},
		},
		// Mixed range (verified against the CyberChef-server oracle).
		{
			"To CI Regex mixed range", "[H-d]", "[A-DH-dh-z]",
			core.Recipe{{Op: "To Case Insensitive Regex"}},
		},
		// Each cross-boundary range rule (all oracle-verified).
		{"CI punct-upper", "!-Z", "!-Za-z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI punct-bracket", "!-_", "!-_a-z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI upper-bracket", "A-_", "A-_a-z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI bracket-brace", "_-~", "_-~A-Z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI lower-brace", "a-~", "a-~A-Z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI punct-lower", "!-z", "!-z", core.Recipe{{Op: "To Case Insensitive Regex"}}},
		{"CI bracket-lower", "_-z", "A-Z_-z", core.Recipe{{Op: "To Case Insensitive Regex"}}},

		{
			"From CI Regex", "[tT][eE][sS][tT]", "test",
			core.Recipe{{Op: "From Case Insensitive Regex"}},
		},
		{
			"From CI Regex keeps distinct", "[ab][cD]", "[ab][cD]",
			core.Recipe{{Op: "From Case Insensitive Regex"}},
		},

		// Round-trip through both.
		{
			"CI Regex round trip", "test", "test",
			core.Recipe{
				{Op: "To Case Insensitive Regex"},
				{Op: "From Case Insensitive Regex"},
			},
		},
	})
}
