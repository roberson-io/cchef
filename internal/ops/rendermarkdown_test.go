package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func mdRecipe(args ...any) core.Recipe {
	return core.Recipe{{Op: "Render Markdown", Args: args}}
}

// Fixtures transcribed from ../CyberChef/tests/operations/tests/RenderMarkdown.mjs.
// CyberChef renders Markdown to HTML with markdown-it (html disabled) and wraps
// it in a div; cchef reproduces this over goldmark. Args are
// [convertLinks, enableHighlighting, openLinksInNewTab]; an empty Args list uses
// the CyberChef defaults [false, true, false].
func TestRenderMarkdownFixtures(t *testing.T) {
	const div = `<div style="font-family: var(--primary-font-family)">`
	const url = "https://gchq.github.io/CyberChef/"
	runCases(t, []opCase{
		{"Render Markdown: Nothing", "", div + "</div>", mdRecipe()},
		{
			"Render Markdown: Basic Text", "Hello World!",
			div + "<p>Hello World!</p>\n</div>", mdRecipe(),
		},
		{
			"Render Markdown: Simple Markdown", "# Hello World!",
			div + "<h1>Hello World!</h1>\n</div>", mdRecipe(),
		},
		{
			"Render Markdown: URL (not expanded)", url,
			div + "<p>" + url + "</p>\n</div>", mdRecipe(false, false, false),
		},
		{
			"Render Markdown: URL (expanded)", url,
			div + `<p><a href="` + url + `">` + url + "</a></p>\n</div>",
			mdRecipe(true, false, false),
		},
		{
			"Render Markdown: Link (not expanded)", "[CyberChef](" + url + ")",
			div + `<p><a href="` + url + `">CyberChef</a></p>` + "\n</div>",
			mdRecipe(false, false, false),
		},
		{
			"Render Markdown: Link (expanded)", "[CyberChef](" + url + ")",
			div + `<p><a href="` + url + `">CyberChef</a></p>` + "\n</div>",
			mdRecipe(true, false, false),
		},
		{
			"Render Markdown: Link (open in new window)", "[CyberChef](" + url + ")",
			div + `<p><a href="` + url + `" target="_blank">CyberChef</a></p>` + "\n</div>",
			mdRecipe(true, false, true),
		},
		{
			"Render Markdown: URL (open in new window)", url,
			div + `<p><a href="` + url + `" target="_blank">` + url + "</a></p>\n</div>",
			mdRecipe(true, false, true),
		},
	})
}

// Additional oracle-verified cases exercising the extensions (tables,
// strikethrough), fenced code and inline raw-HTML escaping.
func TestRenderMarkdownMore(t *testing.T) {
	const div = `<div style="font-family: var(--primary-font-family)">`
	runCases(t, []opCase{
		{
			"strikethrough", "~~gone~~", div + "<p><s>gone</s></p>\n</div>",
			mdRecipe(false, false, false),
		},
		{
			"table", "| a |\n|---|\n| 1 |",
			div + "<table>\n<thead>\n<tr>\n<th>a</th>\n</tr>\n</thead>\n<tbody>\n<tr>\n<td>1</td>\n</tr>\n</tbody>\n</table>\n</div>",
			mdRecipe(false, false, false),
		},
		{
			"fenced code", "```\nx\n```", div + "<pre><code>x\n</code></pre>\n</div>",
			mdRecipe(false, false, false),
		},
		{
			"inline raw HTML is escaped", "a <b>x</b> c",
			div + "<p>a &lt;b&gt;x&lt;/b&gt; c</p>\n</div>",
			mdRecipe(false, false, false),
		},
	})
}

// TestRenderMarkdownBlockHTML documents cchef's block-level raw-HTML handling: it
// escapes the HTML (unlike goldmark's default omission) but, being a documented
// divergence, does not reproduce markdown-it's surrounding <p> wrapping.
func TestRenderMarkdownBlockHTML(t *testing.T) {
	runCases(t, []opCase{
		{
			"block raw HTML escaped without paragraph", "<div>raw</div>",
			`<div style="font-family: var(--primary-font-family)">&lt;div&gt;raw&lt;/div&gt;</div>`,
			mdRecipe(false, false, false),
		},
	})
}
