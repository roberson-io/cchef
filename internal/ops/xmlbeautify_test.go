package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func xmbRecipe(indent string) core.Recipe {
	return core.Recipe{{Op: "XML Beautify", Args: []any{indent}}}
}

// XML Beautify has no CyberChef fixtures; these vectors are verified against the
// vkbeautify library directly (node -e "require('vkbeautify').xml(...)"), which is
// what CyberChef wraps. The library indents each nested element on its own line
// (tracking depth), keeps comment/CDATA/DOCTYPE content on one line, and puts each
// xmlns declaration on its own line. cchef ports the algorithm from scratch.
func TestXMLBeautifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"empty", "", "", xmbRecipe("  ")},
		{"tab indent", "<a><b>1</b><c/></a>", "<a>\n\t<b>1</b>\n\t<c/>\n</a>", xmbRecipe("\t")},
		{"two-space indent", "<a><b>1</b><c/></a>", "<a>\n  <b>1</b>\n  <c/>\n</a>", xmbRecipe("  ")},
		{"numeric-string indent -> 4 spaces", "<a><b>1</b></a>", "<a>\n    <b>1</b>\n</a>", xmbRecipe("2")},
		{"nested depth", "<a><b><c>deep</c></b></a>", "<a>\n  <b>\n    <c>deep</c>\n  </b>\n</a>", xmbRecipe("  ")},
		{"collapses inter-tag whitespace", "<a>  <b>1</b>  <c/> </a>", "<a>\n  <b>1</b>\n  <c/>\n</a>", xmbRecipe("  ")},
		{
			"processing instruction",
			"<?xml version=\"1.0\"?><root><child attr=\"x\">text</child></root>",
			"<?xml version=\"1.0\"?>\n<root>\n  <child attr=\"x\">text</child>\n</root>",
			xmbRecipe("  "),
		},
		{"comment on its own line", "<a><!-- comment --><b/></a>", "<a>\n  <!-- comment -->\n  <b/>\n</a>", xmbRecipe("  ")},
		{
			"xmlns declarations split",
			`<root xmlns="u" xmlns:y="v"><y:e/></root>`,
			"<root\n  xmlns=\"u\"\n  xmlns:y=\"v\">\n  <y:e/>\n</root>",
			xmbRecipe("  "),
		},
		{"CDATA kept on one line", "<a><![CDATA[ raw <x> ]]></a>", "<a>\n  <![CDATA[ raw <x> ]]>\n</a>", xmbRecipe("  ")},
		{"mixed content", "<a>text<b>inner</b>more</a>", "<a>text\n  <b>inner</b>more\n</a>", xmbRecipe("  ")},
		{"plain text unchanged", "plain text no tags", "plain text no tags", xmbRecipe("  ")},
		{"tag inside comment stays inline", "<a><!-- <b>x</b> --></a>", "<a>\n  <!-- <b>x</b> -->\n</a>", xmbRecipe("  ")},
		{"DOCTYPE", "<!DOCTYPE html><html><body>hi</body></html>", "<!DOCTYPE html>\n<html>\n  <body>hi</body>\n</html>", xmbRecipe("  ")},
	})
}
