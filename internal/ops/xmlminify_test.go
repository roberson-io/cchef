package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func xmmRecipe(preserve bool) core.Recipe {
	return core.Recipe{{Op: "XML Minify", Args: []any{preserve}}}
}

// XML Minify has no CyberChef fixtures; these vectors are verified against the
// CyberChef-server oracle. vkbeautify.xmlmin optionally strips comments and
// collapses whitespace before xmlns, then removes whitespace between tags
// (>\s*< -> ><, using JS \s). The comment/xmlns regexes use an ASCII-only
// [ \r\n\t] class, matching the library.
func TestXMLMinifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", xmmRecipe(false)},
		{"collapse tag whitespace", "<a>  <b>1</b>  <c/> </a>", "<a><b>1</b><c/></a>", xmmRecipe(false)},
		{"strip comment", "<a> <!-- comment --> <b>1</b> </a>", "<a><b>1</b></a>", xmmRecipe(false)},
		{"preserve comment", "<a> <!-- comment --> <b>1</b> </a>", "<a><!-- comment --><b>1</b></a>", xmmRecipe(true)},
		{
			"collapse xmlns whitespace",
			"<root\n  xmlns=\"http://x\"\n  xmlns:y=\"http://y\">\n  <y:e/>\n</root>",
			`<root xmlns="http://x" xmlns:y="http://y"><y:e/></root>`,
			xmmRecipe(false),
		},
		{"multiline comment stripped", "<a><!-- multi\nline\ncomment --></a>", "<a></a>", xmmRecipe(false)},
		{"whitespace-only element body", "<x>   </x>", "<x></x>", xmmRecipe(false)},
		{"plain text unchanged", "plain text no tags", "plain text no tags", xmmRecipe(false)},
	})
}
