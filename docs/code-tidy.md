# Code tidy

Operations for working with source code and serialised data formats. Some of
these operations belong to another category too, where their detailed
description, options and examples live:
[CSS selector](extractors.md#css-selector),
[JPath expression](extractors.md#jpath-expression) and
[XPath expression](extractors.md#xpath-expression) are documented under
[Extractors](extractors.md), [Diff](utils.md#diff) is documented under
[Utils](utils.md), and the MessagePack operations under
[Data format](data-format.md). Operations whose only category is
Code tidy (such as JavaScript Beautify and JavaScript Parser) are documented in
full below.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| CSS selector | `css-selector` | [CSS selectors](https://wikipedia.org/wiki/Cascading_Style_Sheets#Selector) |
| Diff | `diff` | |
| From MessagePack | `from-messagepack` | [MessagePack](https://wikipedia.org/wiki/MessagePack) |
| JavaScript Beautify | `javascript-beautify` | [escodegen](https://github.com/estools/escodegen) |
| JavaScript Minify | `javascript-minify` | [esbuild](https://github.com/evanw/esbuild) |
| JavaScript Parser | `javascript-parser` | [Abstract syntax tree](https://wikipedia.org/wiki/Abstract_syntax_tree) |
| JPath expression | `jpath-expression` | [JSONPath](extractors.md#jpath-expression) |
| Jq | `jq` | [jq](https://github.com/jqlang/jq) |
| JSON Beautify | `json-beautify` | [JSON](https://wikipedia.org/wiki/JSON) |
| Render Markdown | `render-markdown` | [Markdown](https://wikipedia.org/wiki/Markdown) |
| SQL Beautify | `sql-beautify` | [SQL](https://wikipedia.org/wiki/SQL) |
| Syntax highlighter | `syntax-highlighter` | [Syntax highlighting](https://wikipedia.org/wiki/Syntax_highlighting) |
| To MessagePack | `to-messagepack` | [MessagePack](https://wikipedia.org/wiki/MessagePack) |
| XPath expression | `xpath-expression` | [XPath](extractors.md#xpath-expression) |

## JavaScript Beautify

Parses valid JavaScript (or JSON that is a valid JS expression, such as an array
or a parenthesised object) and pretty-prints it — byte-for-byte identical to
CyberChef's output.

The input is parsed with the [JavaScript Parser](#javascript-parser) port and
regenerated with a from-scratch Go transliteration of
[escodegen](https://github.com/estools/escodegen), using CyberChef's fixed
formatting options. The full script grammar is supported.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Indent string | string | `\t` (tab) | The indentation unit. Backslash escapes (`\t`, `\n`, …) are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. |
| Quotes | option | `Auto` | String quote style: `Auto` (whichever needs fewer escapes), `Single`, or `Double`. |
| Semicolons before closing braces | boolean | `true` | When `false`, the optional semicolon before a closing brace is omitted. |
| Include comments | boolean | `true` | When `true`, comments are collected with source ranges, attached to the nearest node (a port of estraverse's `attachComments`), and re-emitted; when `false`, comments are dropped. |

A syntax error surfaces as `Unable to parse JavaScript.` followed by the parser's
message.

### Simple example

```bash
cchef javascript-beautify -i "const o={a:1,b:[2,3],f:function(){return 1}}"
```

Output:

```
const o = {
	a: 1,
	b: [
		2,
		3
	],
	f: function () {
		return 1;
	}
};
```

### Complex example

Two-space indentation with double quotes:

```bash
cchef javascript-beautify -i "class A extends B{constructor(){super()}m(){return this.x}}" --indent-string "  " --quotes Double
```

Output:

```
class A extends B {
  constructor() {
    super();
  }
  m() {
    return this.x;
  }
}
```

## JavaScript Minify

Compresses JavaScript code. Takes no options.

> **Reduced fidelity.** CyberChef minifies with the npm
> [`terser`](https://github.com/terser/terser) library (which has no logic to
> port). cchef uses [esbuild](https://github.com/evanw/esbuild)'s minifier
> instead — a pure-Go library that keeps cchef a single static binary. The output
> is equivalently minified but **not byte-identical** to CyberChef's: esbuild and
> terser use different identifier manglers and compression passes, so mangled
> names and some rewrites differ (though they often coincide).

Invalid input surfaces an `Error minifying JavaScript. (...)` error.

### Example

```bash
cchef javascript-minify -i "function greet(name) {
    const message = 'Hello, ' + name + '!';
    return message;
}"
```

Output:

```
function greet(e){return"Hello, "+e+"!"}
```

## JavaScript Parser

Returns an [Abstract Syntax Tree](https://wikipedia.org/wiki/Abstract_syntax_tree)
(ESTree/[esprima](https://esprima.org/) shape) for valid JavaScript source code,
serialised as pretty-printed JSON — byte-for-byte identical to CyberChef's
output.

The parser is a from-scratch Go transliteration of esprima and supports the full
script grammar exactly — classes, `async`/`await`, generators and `yield`,
destructuring (including assignment targets), object accessor/generator/async
methods, `new.target`, and Unicode (non-ASCII and `\u`-escaped) identifiers. As
with `esprima.parseScript`, ES modules (`import`/`export`) are syntax errors. The
output-shaping options below are the only unported feature and are rejected if
enabled.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Location info | boolean | `false` | Not yet supported; rejected if `true`. |
| Range info | boolean | `false` | Not yet supported; rejected if `true`. |
| Include tokens array | boolean | `false` | Not yet supported; rejected if `true`. |
| Include comments array | boolean | `false` | Not yet supported; rejected if `true`. |
| Report errors and try to continue | boolean | `false` | Not yet supported; rejected if `true`. |

A syntax error surfaces esprima's message in V8 form (`Line N: <reason>`).

### Simple example

```bash
cchef javascript-parser -i "1 + 2"
```

Output:

```
{
  "type": "Program",
  "body": [
    {
      "type": "ExpressionStatement",
      "expression": {
        "type": "BinaryExpression",
        "operator": "+",
        "left": {
          "type": "Literal",
          "value": 1,
          "raw": "1"
        },
        "right": {
          "type": "Literal",
          "value": 2,
          "raw": "2"
        }
      }
    }
  ],
  "sourceType": "script"
}
```

### Complex example

Modern syntax such as arrow functions, template literals and destructuring is
parsed faithfully:

```bash
cchef javascript-parser -i "const add = (a, b) => a + b;"
```

Output:

```
{
  "type": "Program",
  "body": [
    {
      "type": "VariableDeclaration",
      "declarations": [
        {
          "type": "VariableDeclarator",
          "id": {
            "type": "Identifier",
            "name": "add"
          },
          "init": {
            "type": "ArrowFunctionExpression",
            "id": null,
            "params": [
              {
                "type": "Identifier",
                "name": "a"
              },
              {
                "type": "Identifier",
                "name": "b"
              }
            ],
            "body": {
              "type": "BinaryExpression",
              "operator": "+",
              "left": {
                "type": "Identifier",
                "name": "a"
              },
              "right": {
                "type": "Identifier",
                "name": "b"
              }
            },
            "generator": false,
            "expression": true,
            "async": false
          }
        }
      ],
      "kind": "const"
    }
  ],
  "sourceType": "script"
}
```

## Jq

Processes the JSON input with a [jq](https://github.com/jqlang/jq) query. CyberChef
wraps jq-web (jq compiled to WebAssembly); cchef reimplements the operation over
[gojq](https://github.com/itchyny/gojq), a pure-Go jq, so it stays a single static
binary with no cgo.

The input must be valid JSON. jq's output is a stream of values, which is
collapsed the way jq-web's `jq.json()` does: **zero** results is an error, a
**single** result is returned directly, and **multiple** results become a JSON
array. The result is then printed raw (only when *Raw* is set and the result is a
string) or serialised like JavaScript's `JSON.stringify` (compact, UTF-8
preserved).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Query | string | (empty) | The jq program to run. |
| Raw | boolean | `false` | When set, a string result is printed without surrounding quotes; non-string results are still JSON-encoded. |

An invalid query, or one that raises a runtime error, is reported as
`Invalid jq expression: <message>`. A query that produces no output is reported as
`Invalid jq expression: Unexpected end of JSON input`, matching jq-web.

> As gojq is an independent reimplementation of jq, error message wording can
> differ from jq-web, and rare numeric edge cases may vary; `NaN` serialises to
> `null` (as `JSON.stringify` does).

### Simple example

```bash
cchef jq -i '{"name":"cchef","tags":["cli","json"]}' --query ".tags"
```

Output:

```
["cli","json"]
```

### Complex example

Pull a field out of each element, then reduce — and print a bare string with
`--raw`:

```bash
cchef jq -i '[{"n":1},{"n":2},{"n":3}]' --query "map(.n) | add"
```

Output:

```
6
```

```bash
cchef jq -i '{"msg":"hello world"}' --query ".msg" --raw
```

Output:

```
hello world
```

## JSON Beautify

Indents and pretty-prints JSON. CyberChef parses the input leniently with
[JSON5](https://json5.org/) (allowing comments, trailing commas, unquoted and
single-quoted keys, hexadecimal and non-finite numbers, and more) and re-emits it
with `JSON.stringify(value, null, indent)`; cchef reproduces this over a
from-scratch JSON5 parser feeding the shared JSON serialiser — no dependency is
added.

Object keys are enumerated in ECMAScript order (integer-like keys first, in
ascending numeric order, then the rest in insertion order). Non-finite numbers
(`Infinity`, `NaN`) become `null`, and large integers lose precision exactly as JS
Numbers do.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Indent string | string | 4 spaces | The indentation unit. Backslash escapes are interpreted, so `\t` indents with tabs. An empty string produces compact (single-line) output. |
| Sort Object Keys | boolean | `false` | Recursively sort object keys alphabetically before emitting. |
| Formatted | boolean | `true` | Inert in cchef: it only controls CyberChef's browser tree view. Kept so recipes round-trip. |

### Simple example

```bash
cchef json-beautify -i '{"name":"cchef","tags":["json","cli"],"version":2}'
```

Output:

```
{
    "name": "cchef",
    "tags": [
        "json",
        "cli"
    ],
    "version": 2
}
```

### Complex example

Sorting keys and indenting with tabs, over lenient JSON5 input (a comment, a
hexadecimal number, a leading-dot number and trailing commas):

```bash
cchef json-beautify -i '{b:2,a:{d:0xF,/* c */ c:.5,},}' --indent-string '\t' --sort-object-keys
```

Output:

```
{
	"a": {
		"c": 0.5,
		"d": 15
	},
	"b": 2
}
```

## Render Markdown

Renders Markdown input as HTML, wrapped in a `<div>`. CyberChef uses the
[markdown-it](https://github.com/markdown-it/markdown-it) library (with raw HTML
disabled) plus [highlight.js](https://highlightjs.org/) to colour fenced code
blocks; cchef reimplements it over the pure-Go
[goldmark](https://github.com/yuin/goldmark) library.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Autoconvert URLs to links | boolean | `false` | When set, bare URLs in the text become links (markdown-it's `linkify`). |
| Enable syntax highlighting | boolean | `true` | Accepted for compatibility but has no effect (see below). |
| Open links in new tab. | boolean | `false` | When set, `target="_blank"` is added to every link. |

The common Markdown surface — headings, emphasis, strikethrough, lists (nested),
links, images, inline code, fenced code, blockquotes, tables and linkify —
matches CyberChef byte-for-byte.

> **Reduced fidelity.** goldmark is not markdown-it, and two areas differ and are
> not ported: **syntax highlighting** of fenced code blocks (CyberChef colours
> them with highlight.js; cchef emits the plain escaped code, so the *Enable
> syntax highlighting* option has no effect) and **block-level raw HTML**
> (markdown-it escapes it inside a paragraph; cchef escapes it without the
> surrounding `<p>`). Inline raw HTML is escaped identically.

### Simple example

```bash
cchef render-markdown -i "# Title

Some **bold** text and \`code\`."
```

Output:

```
<div style="font-family: var(--primary-font-family)"><h1>Title</h1>
<p>Some <strong>bold</strong> text and <code>code</code>.</p>
</div>
```

### Complex example

Autoconvert URLs to links and open every link in a new tab:

```bash
cchef render-markdown -i "Visit https://example.com and see [docs](https://docs.example.com)." --autoconvert-urls-to-links --open-links-in-new-tab
```

Output:

```
<div style="font-family: var(--primary-font-family)"><p>Visit <a href="https://example.com" target="_blank">https://example.com</a> and see <a href="https://docs.example.com" target="_blank">docs</a>.</p>
</div>
```

## SQL Beautify

Indents and prettifies SQL. CyberChef wraps the
[sql-formatter](https://github.com/sql-formatter-org/sql-formatter) npm library
(MySQL dialect, standard indent style, keyword case preserved); cchef reimplements
that formatter from scratch — a tokenizer, a small clause/expression parser and a
port of sql-formatter's whitespace-layout engine — so no dependency is added and
the output matches byte-for-byte across the common SQL surface.

Each top-level clause (`SELECT`, `FROM`, `WHERE`, `GROUP BY`, `JOIN`, …) starts a
new line with its contents indented; comma-separated items each go on their own
line; `AND`/`OR` begin new lines; and parenthesised sub-queries expand when they
do not fit inline. Keyword case is preserved. Bind variables (`:name`) are left
untouched.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Indent string | string | `\t` (tab) | The indentation unit. A tab indents with tabs; anything else indents with that many spaces (`  ` → 2 spaces). Backslash escapes are interpreted. |

> **Fidelity.** This reproduces sql-formatter's MySQL/standard output for the
> common SQL surface (verified byte-for-byte against the CyberChef-server oracle
> over a broad corpus). Exotic dialect-specific constructs may differ.

### Simple example

```bash
cchef sql-beautify -i "SELECT id, name FROM users WHERE active = 1"
```

Output:

```
SELECT
	id,
	name
FROM
	users
WHERE
	active = 1
```

### Complex example

Joins, aggregation and clauses, indented with two spaces:

```bash
cchef sql-beautify -i "select o.id, c.name, sum(o.total) from orders o join customers c on o.cust_id=c.id where o.total>100 group by c.name having sum(o.total)>500 order by 2" --indent-string "  "
```

Output:

```
select
  o.id,
  c.name,
  sum(o.total)
from
  orders o
  join customers c on o.cust_id = c.id
where
  o.total > 100
group by
  c.name
having
  sum(o.total) > 500
order by
  2
```

## Syntax highlighter

Adds syntax highlighting to source code. CyberChef highlights with
[highlight.js](https://highlightjs.org/), emitting HTML `<span>`s that carry
`hljs-*` CSS classes (or auto-detecting the language); cchef reimplements this
over [chroma](https://github.com/alecthomas/chroma), mapping chroma's token types
onto the same `hljs-*` class vocabulary so the HTML can be styled with a
highlight.js theme.

As a CLI-native addition beyond CyberChef (which only produces HTML), the
**Output format** option can instead render the highlighting straight to the
terminal with ANSI colours.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Language | string | `auto detect` | A chroma language name or alias (case-insensitive, e.g. `go`, `javascript`, `python`), or `auto detect` to infer it from the input. An unrecognised name is an error. |
| Output format | option | `HTML` | `HTML` emits `hljs-*` spans (matching CyberChef); `Terminal` emits ANSI-coloured text for direct display in a terminal. |

> **Fidelity.** chroma is not highlight.js, so token boundaries and, especially,
> language auto-detection differ — the highlighted regions are not byte-identical
> to CyberChef's. The `hljs-*` class vocabulary and output shape do match. This
> operation is excluded from CyberChef's own Node build, so there are no upstream
> test fixtures or oracle output to compare against.

### Simple example

```bash
cchef syntax-highlighter -i "let x = 42; // answer" --language javascript
```

Output:

```
<span class="hljs-keyword">let</span> x = <span class="hljs-number">42</span>; <span class="hljs-comment">// answer
</span>
```

### Complex example

Highlighting Python — note the function name picks up `hljs-title`:

```bash
cchef syntax-highlighter -i "def add(a, b):
    return a + b" --language python
```

Output:

```
<span class="hljs-keyword">def</span> <span class="hljs-title">add</span>(a, b):
    <span class="hljs-keyword">return</span> a + b
```

To view the highlighting in a terminal instead of as HTML, select the terminal
output format (the result is ANSI-coloured text, so it is not reproduced here):

```bash
cchef syntax-highlighter -i "func add(a, b int) int { return a + b }" --language go --output-format Terminal
```
