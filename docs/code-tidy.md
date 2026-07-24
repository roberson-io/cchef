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
| BSON deserialise | `bson-deserialise` | [BSON](https://wikipedia.org/wiki/BSON) |
| BSON serialise | `bson-serialise` | [BSON](https://wikipedia.org/wiki/BSON) |
| CSS Beautify | `css-beautify` | [CSS](https://wikipedia.org/wiki/CSS) |
| CSS Minify | `css-minify` | [CSS](https://wikipedia.org/wiki/CSS) |
| CSS selector | `css-selector` | [CSS selectors](https://wikipedia.org/wiki/Cascading_Style_Sheets#Selector) |
| Diff | `diff` | |
| From MessagePack | `from-messagepack` | [MessagePack](https://wikipedia.org/wiki/MessagePack) |
| JavaScript Beautify | `javascript-beautify` | [escodegen](https://github.com/estools/escodegen) |
| JavaScript Minify | `javascript-minify` | [esbuild](https://github.com/evanw/esbuild) |
| JavaScript Parser | `javascript-parser` | [Abstract syntax tree](https://wikipedia.org/wiki/Abstract_syntax_tree) |
| JPath expression | `jpath-expression` | [JSONPath](extractors.md#jpath-expression) |
| Jq | `jq` | [jq](https://github.com/jqlang/jq) |
| JSON Beautify | `json-beautify` | [JSON](https://wikipedia.org/wiki/JSON) |
| JSON Minify | `json-minify` | [JSON](https://wikipedia.org/wiki/JSON) |
| Render Markdown | `render-markdown` | [Markdown](https://wikipedia.org/wiki/Markdown) |
| SQL Beautify | `sql-beautify` | [SQL](https://wikipedia.org/wiki/SQL) |
| SQL Minify | `sql-minify` | [SQL](https://wikipedia.org/wiki/SQL) |
| Syntax highlighter | `syntax-highlighter` | [Syntax highlighting](https://wikipedia.org/wiki/Syntax_highlighting) |
| To MessagePack | `to-messagepack` | [MessagePack](https://wikipedia.org/wiki/MessagePack) |
| XML Beautify | `xml-beautify` | [XML](https://wikipedia.org/wiki/XML) |
| XML Minify | `xml-minify` | [XML](https://wikipedia.org/wiki/XML) |
| XPath expression | `xpath-expression` | [XPath](extractors.md#xpath-expression) |

## BSON deserialise

Deserialises [BSON](https://bsonspec.org/) (Binary JSON, MongoDB's binary
interchange format) bytes back into JSON. Input is raw BSON bytes; output is
pretty-printed JSON (2-space indent), matching how CyberChef renders
`JSON.stringify(deserialize(input), null, 2)`.

The richer BSON element types are rendered as js-bson's `JSON.stringify` does: an
ObjectId as its hex string, a UTC datetime as an ISO-8601 string, a Binary as
base64, a Timestamp as `{"$timestamp": "<n>"}`, and RegExp / MinKey / MaxKey as an
empty object. cchef reimplements the codec from scratch (no dependency).

> **Fidelity.** The common types (double, string, document, array, boolean, null,
> int32, int64) and the types above are reproduced exactly. Decimal128 and other
> rare types (JavaScript code, DBPointer, Symbol) are not decoded and produce an
> error.

### Example

Round-tripping a document through serialise and back:

```bash
cchef bson-serialise -i '{"hello":"world","n":42}' | cchef bson-deserialise
```

Output:

```
{
  "hello": "world",
  "n": 42
}
```

Deserialising raw bytes supplied as hex:

```bash
echo -n '160000000268656c6c6f0006000000776f726c640000' | cchef from-hex --delimiter None | cchef bson-deserialise
```

Output:

```
{
  "hello": "world"
}
```

## BSON serialise

Serialises JSON into [BSON](https://bsonspec.org/) (Binary JSON, MongoDB's binary
interchange format) bytes. Input must be valid JSON; output is raw BSON bytes.
cchef reimplements js-bson's `serialize()` from scratch (no dependency), matching
it byte-for-byte, including its number-type rule — an integer in int32 range is
written as an int32, everything else (larger integers, all fractional numbers, and
negative zero) as a 64-bit double — and its ECMAScript key ordering (integer-like
keys first, ascending).

The top-level value must be an object (a JSON `null` serialises to an empty
document); an array or scalar root is rejected, as in js-bson.

### Example

Serialising to bytes (shown here piped through `to-hex` for display):

```bash
cchef bson-serialise -i '{"hello":"world","n":42}' | cchef to-hex --delimiter Space
```

Output:

```
1d 00 00 00 02 68 65 6c 6c 6f 00 06 00 00 00 77 6f 72 6c 64 00 10 6e 00 2a 00 00 00 00
```

## CSS Beautify

Indents and prettifies CSS. Ported from the [vkbeautify](https://github.com/vkiryukhin/vkBeautify)
library CyberChef wraps — reimplemented from scratch in Go (no dependency),
matching it byte-for-byte. Each declaration and block is placed on its own line,
indented by nesting depth.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Indent string | string | `\t` (tab) | The indentation unit. Backslash escapes are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. A step that begins with a digit falls back to four spaces (a vkbeautify quirk). |

### Example

```bash
cchef css-beautify -i 'body{margin:0;padding:0}' --indent-string '  '
```

Output:

```
body{
  margin:0;
  padding:0
}
```

## CSS Minify

Compresses CSS. Ported from vkbeautify (from scratch, no dependency): comments are
stripped (unless preserved), whitespace runs are collapsed to a single space, and
whitespace after `{`, `}`, `;`, `/*` and `*/` is removed.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Preserve comments | boolean | `false` | When `true`, comments are kept instead of stripped. |

### Example

```bash
cchef css-minify -i 'body {
  margin: 0;
  /* reset */
  padding: 0;
}'
```

Output:

```
body {margin: 0;padding: 0;}
```

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

## JSON Minify

Compresses JSON by removing insignificant whitespace. Equivalent to
`JSON.stringify(JSON.parse(text), null, 0)`; cchef reuses the shared
order-preserving JSON serialiser, so output matches byte-for-byte. Strict JSON only
(unlike JSON Beautify, which accepts JSON5). Empty input yields an empty string.

### Example

```bash
cchef json-minify -i '{
  "name": "cchef",
  "tags": ["a", "b"]
}'
```

Output:

```
{"name":"cchef","tags":["a","b"]}
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

## SQL Minify

Compresses SQL. Ported from vkbeautify (from scratch, no dependency): whitespace
runs are collapsed to a single space, then the space before the first `(` and the
first `)` is removed (the library's replaces for those two are deliberately not
global, and cchef preserves that).

### Example

```bash
cchef sql-minify -i 'SELECT  id,  name
FROM   users
WHERE  active = 1'
```

Output:

```
SELECT id, name FROM users WHERE active = 1
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

## XML Beautify

Indents and prettifies XML. Ported from the [vkbeautify](https://github.com/vkiryukhin/vkBeautify)
library CyberChef wraps — reimplemented from scratch in Go (no dependency),
matching it byte-for-byte. Each nested element is placed on its own line indented
by depth; comments, CDATA and DOCTYPE content stay on one line, and each `xmlns`
declaration is put on its own line.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Indent string | string | `\t` (tab) | The indentation unit. Backslash escapes are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. A step that begins with a digit falls back to four spaces (a vkbeautify quirk). |

### Example

```bash
cchef xml-beautify -i '<note><to>Alice</to><body>Hi</body></note>' --indent-string '  '
```

Output:

```
<note>
  <to>Alice</to>
  <body>Hi</body>
</note>
```

## XML Minify

Compresses XML. Ported from vkbeautify (from scratch, no dependency): comments are
stripped (unless preserved) and whitespace before `xmlns` is collapsed, then
whitespace between tags is removed.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Preserve comments | boolean | `false` | When `true`, comments are kept instead of stripped. |

### Example

```bash
cchef xml-minify -i '<note>
  <to>Alice</to>
  <!-- greeting -->
  <body>Hi</body>
</note>'
```

Output:

```
<note><to>Alice</to><body>Hi</body></note>
```
