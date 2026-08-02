# Code tidy

Operations for working with source code and serialized data formats. Some of
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
| Generic Code Beautify | `generic-code-beautify` | [Prettyprint](https://wikipedia.org/wiki/Prettyprint) |
| JavaScript Beautify | `javascript-beautify` | [escodegen](https://github.com/estools/escodegen) |
| JavaScript Minify | `javascript-minify` | [esbuild](https://github.com/evanw/esbuild) |
| JavaScript Parser | `javascript-parser` | [Abstract syntax tree](https://wikipedia.org/wiki/Abstract_syntax_tree) |
| JPath expression | `jpath-expression` | [JSONPath](extractors.md#jpath-expression) |
| Jq | `jq` | [jq](https://github.com/jqlang/jq) |
| JSON Beautify | `json-beautify` | [JSON](https://wikipedia.org/wiki/JSON) |
| JSON Minify | `json-minify` | [JSON](https://wikipedia.org/wiki/JSON) |
| Microsoft Script Decoder | `microsoft-script-decoder` | [JScript.Encode](https://wikipedia.org/wiki/JScript.Encode) |
| PHP Deserialize | `php-deserialize` | [Serialization](https://wikipedia.org/wiki/Serialization) |
| PHP Serialize | `php-serialize` | [Serialization](https://wikipedia.org/wiki/Serialization) |
| Render Markdown | `render-markdown` | [Markdown](https://wikipedia.org/wiki/Markdown) |
| SQL Beautify | `sql-beautify` | [SQL](https://wikipedia.org/wiki/SQL) |
| SQL Minify | `sql-minify` | [SQL](https://wikipedia.org/wiki/SQL) |
| Strip HTML tags | `strip-html-tags` | [HTML element](https://wikipedia.org/wiki/HTML_element) |
| Syntax highlighter | `syntax-highlighter` | [Syntax highlighting](https://wikipedia.org/wiki/Syntax_highlighting) |
| To Camel case | `to-camel-case` | [Camel case](https://wikipedia.org/wiki/Camel_case) |
| To Kebab case | `to-kebab-case` | [Kebab case](https://wikipedia.org/wiki/Letter_case#Special_case_styles) |
| To MessagePack | `to-messagepack` | [MessagePack](https://wikipedia.org/wiki/MessagePack) |
| To Snake case | `to-snake-case` | [Snake case](https://wikipedia.org/wiki/Snake_case) |
| XML Beautify | `xml-beautify` | [XML](https://wikipedia.org/wiki/XML) |
| XML Minify | `xml-minify` | [XML](https://wikipedia.org/wiki/XML) |
| XPath expression | `xpath-expression` | [XPath](extractors.md#xpath-expression) |

## BSON deserialise

Deserializes [BSON](https://bsonspec.org/) (Binary JSON, MongoDB's binary
interchange format) bytes back into JSON. Input is raw BSON bytes; output is
pretty-printed JSON (2-space indent), matching how CyberChef renders
`JSON.stringify(deserialize(input), null, 2)`.

The richer BSON element types are rendered as js-bson's `JSON.stringify` does: an
ObjectId as its hex string, a UTC datetime as an ISO-8601 string, a Binary as
base64, a Timestamp as `{"$timestamp": "<n>"}`, and RegExp / MinKey / MaxKey as an
empty object.

> **Fidelity.** The common types (double, string, document, array, boolean, null,
> int32, int64) and the types above are reproduced exactly. Decimal128 and other
> rare types (JavaScript code, DBPointer, Symbol) are not decoded and produce an
> error.

### Example

Round-tripping a document through serialize and back:

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

Deserializing raw bytes supplied as hex:

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

Serializes JSON into [BSON](https://bsonspec.org/) (Binary JSON, MongoDB's binary
interchange format) bytes. Input must be valid JSON; output is raw BSON bytes.
The number-type rule follows js-bson: an integer in int32 range is written as an
int32, everything else (larger integers, all fractional numbers, and negative
zero) as a 64-bit double. Keys are written in ECMAScript order (integer-like keys
first, ascending).

The top-level value must be an object (a JSON `null` serializes to an empty
document); an array or scalar root is rejected, as in js-bson.

### Example

Serializing to bytes (shown here piped through `to-hex` for display):

```bash
cchef bson-serialise -i '{"hello":"world","n":42}' | cchef to-hex --delimiter Space
```

Output:

```
1d 00 00 00 02 68 65 6c 6c 6f 00 06 00 00 00 77 6f 72 6c 64 00 10 6e 00 2a 00 00 00 00
```

## CSS Beautify

Indents and prettifies CSS. Each declaration and block is placed on its own line,
indented by nesting depth.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--indent-string` | string | `\t` (tab) | The indentation unit. Backslash escapes are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. A step that begins with a digit falls back to four spaces (a vkbeautify quirk). |

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

Compresses CSS: comments are
stripped (unless preserved), whitespace runs are collapsed to a single space, and
whitespace after `{`, `}`, `;`, `/*` and `*/` is removed.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--preserve-comments` | boolean | `false` | When `true`, comments are kept instead of stripped. |

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

## Generic Code Beautify

Attempts to pretty-print C-style languages (C, Java, PHP, JavaScript, …) with a
generic, language-agnostic set of rules: strings, comments and regex literals are
preserved verbatim while whitespace, braces and indentation are normalized.

This is a best-effort formatter, not a parser — it will not always produce ideal
output, but it is deterministic.

### Example

```bash
cchef generic-code-beautify -i 'if(x){y=1;}'
```

Output:

```
if (x)  {
    y = 1;
}
```

## JavaScript Beautify

Parses valid JavaScript (or JSON that is a valid JS expression, such as an array
or a parenthesised object) and pretty-prints it with a fixed formatting style.

The input is parsed with the [JavaScript Parser](#javascript-parser); the full
script grammar is supported.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--indent-string` | string | `\t` (tab) | The indentation unit. Backslash escapes (`\t`, `\n`, …) are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. |
| `--quotes` | option | `Auto` | String quote style: `Auto` (whichever needs fewer escapes), `Single`, or `Double`. |
| `--semicolons-before-closing-braces` | boolean | `true` | When `false`, the optional semicolon before a closing brace is omitted. |
| `--include-comments` | boolean | `true` | When `true`, comments are collected with source ranges, attached to the nearest node (a port of estraverse's `attachComments`), and re-emitted; when `false`, comments are dropped. |

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
serialized as pretty-printed JSON.

The parser supports the full script grammar — classes, `async`/`await`, generators
and `yield`,
destructuring (including assignment targets), object accessor/generator/async
methods, `new.target`, and Unicode (non-ASCII and `\u`-escaped) identifiers. As
with `esprima.parseScript`, ES modules (`import`/`export`) are syntax errors. The
output-shaping options below are the only unported feature and are rejected if
enabled.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--location-info` | boolean | `false` | Not yet supported; rejected if `true`. |
| `--range-info` | boolean | `false` | Not yet supported; rejected if `true`. |
| `--include-tokens-array` | boolean | `false` | Not yet supported; rejected if `true`. |
| `--include-comments-array` | boolean | `false` | Not yet supported; rejected if `true`. |
| `--report-errors` | boolean | `false` | Not yet supported; rejected if `true`. |

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

Processes the JSON input with a [jq](https://github.com/jqlang/jq) query.

The input must be valid JSON. jq's output is a stream of values, which is
collapsed the way jq-web's `jq.json()` does: **zero** results is an error, a

> **Alternative to** [`jq`](https://jqlang.github.io/jq/). Built on gojq, so a few corner cases differ from jq proper; parity is with CyberChef's jq-web.

**single** result is returned directly, and **multiple** results become a JSON
array. The result is then printed raw (only when *Raw* is set and the result is a
string) or serialized like JavaScript's `JSON.stringify` (compact, UTF-8
preserved).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--query` | string | (empty) | The jq program to run. |
| `--raw` | boolean | `false` | When set, a string result is printed without surrounding quotes; non-string results are still JSON-encoded. |

An invalid query, or one that raises a runtime error, is reported as
`Invalid jq expression: <message>`. A query that produces no output is reported as
`Invalid jq expression: Unexpected end of JSON input`, matching jq-web.

> Error message wording can differ from jq-web, and rare numeric edge cases may
> vary; `NaN` serializes to `null` (as `JSON.stringify` does).

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
with `JSON.stringify(value, null, indent)`.

Object keys are enumerated in ECMAScript order (integer-like keys first, in
ascending numeric order, then the rest in insertion order). Non-finite numbers
(`Infinity`, `NaN`) become `null`, and large integers lose precision exactly as JS
Numbers do.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--indent-string` | string | 4 spaces | The indentation unit. Backslash escapes are interpreted, so `\t` indents with tabs. An empty string produces compact (single-line) output. |
| `--sort-object-keys` | boolean | `false` | Recursively sort object keys alphabetically before emitting. |
| `--formatted` | boolean | `true` | Inert in cchef: it only controls CyberChef's browser tree view. Kept so recipes round-trip. |

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
shared JSON serializer. Strict JSON only
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

## Microsoft Script Decoder

Decodes Microsoft Encoded Script files (`JScript.Encode` / `VBScript.Encode`,
recognizable by the `#@~^…^#~@` markers); input that does not contain the encoded
markers yields an empty string.

### Example

```bash
cchef microsoft-script-decoder -i '#@~^AAAAAA==,ACfs$BBBBBB==^#~@'
```

Output:

```
[EH3m[
```

## PHP Deserialize

Deserializes [PHP serialized](https://www.php.net/manual/en/function.serialize.php)
data, rendering keyed arrays as JSON-like objects.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--output-valid-json` | boolean | `true` | When `true`, integer keys are quoted (`"0":`) and `"` in strings is escaped; when `false`, integer keys are left bare (`0:`). |

### Example

```bash
cchef php-deserialize -i 'a:2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}'
```

Output:

```
{"name": "Bob","age": 30}
```

## PHP Serialize

Performs [PHP serialization](https://www.php.net/manual/en/function.serialize.php)
on JSON input: arrays and objects both become `a:N:{…}`, and string lengths use
JavaScript UTF-16 code-unit counts
(matching CyberChef; this differs from PHP's own byte-length semantics for
multi-byte strings).

### Example

```bash
cchef php-serialize -i '{"name":"Bob","age":30}'
```

Output:

```
a:2:{s:4:"name";s:3:"Bob";s:3:"age";i:30;}
```

## Render Markdown

Renders Markdown input as HTML, wrapped in a `<div>`. CyberChef uses the
[markdown-it](https://github.com/markdown-it/markdown-it) library (with raw HTML
disabled). cchef renders with [goldmark](https://github.com/yuin/goldmark), which
differs from markdown-it's fenced-code highlighting (see below).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--autoconvert-urls-to-links` | boolean | `false` | When set, bare URLs in the text become links (markdown-it's `linkify`). |
| `--enable-syntax-highlighting` | boolean | `true` | Accepted for compatibility but has no effect (see below). |
| `--open-links-in-new-tab` | boolean | `false` | When set, `target="_blank"` is added to every link. |

The common Markdown surface — headings, emphasis, strikethrough, lists (nested),
links, images, inline code, fenced code, blockquotes, tables and linkify —
behaves as CyberChef does.

> **Reduced fidelity.** goldmark is not markdown-it, and two areas differ and are
> not ported: **syntax highlighting** of fenced code blocks (CyberChef colors
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

Indents and prettifies SQL in the sql-formatter MySQL dialect (standard indent
style, keyword case preserved).

Each top-level clause (`SELECT`, `FROM`, `WHERE`, `GROUP BY`, `JOIN`, …) starts a
new line with its contents indented; comma-separated items each go on their own
line; `AND`/`OR` begin new lines; and parenthesised sub-queries expand when they
do not fit inline. Keyword case is preserved. Bind variables (`:name`) are left
untouched.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--indent-string` | string | `\t` (tab) | The indentation unit. A tab indents with tabs; anything else indents with that many spaces (`  ` → 2 spaces). Backslash escapes are interpreted. |

> **Fidelity.** This targets sql-formatter's MySQL/standard output; exotic
> dialect-specific constructs may differ.

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

Compresses SQL: whitespace
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

## Strip HTML tags

Removes all HTML tags (`<…>`) from a string, repeating until none remain (avoiding
incomplete sanitisation). Optionally tidies whitespace afterwards.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--remove-indentation` | boolean | `true` | Removes leading indentation on each line. |
| `--remove-excess-line-breaks` | boolean | `true` | Drops a leading blank line and collapses runs of blank lines. |

### Example

```bash
cchef strip-html-tags -i '<p>Hello <b>World</b></p>'
```

Output:

```
Hello World
```

## Syntax highlighter

Adds syntax highlighting to source code, emitting HTML `<span>`s that carry
highlight.js `hljs-*` CSS classes so the output can be styled with a highlight.js
theme. cchef highlights with [chroma](https://github.com/alecthomas/chroma),
auto-detecting the language unless one is given.

HTML is not much use on a terminal, so when the output is going to one the spans
are rendered as ANSI color instead. This is a property of where the output is
going rather than of the recipe, so it is the global `--ansi` flag rather than an
operation option:

- `auto` (the default) colors an interactive terminal and leaves the HTML alone
  whenever the output is piped, redirected, or written with `-o`. It also stands
  down for [`NO_COLOR`](https://no-color.org) and for `TERM=dumb`.
- `always` forces the ANSI rendering, which is what `| less -R` needs.
- `never` always gives the HTML, even at a terminal.

The flag is `--ansi` rather than the usual `--color` because operation arguments
keep CyberChef's spellings, and two operations have a *Colour* argument of their
own; a global `--color` would take that name away from them.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--language` | string | `auto detect` | A chroma language name or alias (case-insensitive, e.g. `go`, `javascript`, `python`), or `auto detect` to infer it from the input. An unrecognized name is an error. |

> **Fidelity.** chroma is not highlight.js, so token boundaries and, especially,
> language auto-detection differ — the highlighted regions are not byte-identical
> to CyberChef's. The `hljs-*` class vocabulary and output shape do match.

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

Run at a terminal, the same command shows the highlighting in color rather than
as HTML. Piping it into a pager needs `--ansi always`, since the output is no
longer going to the terminal directly (the result is ANSI-colored text, so it is
not reproduced here):

```bash
cchef syntax-highlighter -i "func add(a, b int) int { return a + b }" --language go --ansi always | less -R
```

Going the other way, `--ansi never` gives the HTML at a terminal too — useful
when reading it directly or copying it out:

```bash
cchef syntax-highlighter -i "let x = 42;" --language javascript --ansi never
```

Output:

```
<span class="hljs-keyword">let</span> x = <span class="hljs-number">42</span>;
```

## To Camel case

Converts the input to camel case (all lower case except letters after word
boundaries, which are upper case, e.g. `thisIsCamelCase`).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--attempt-to-be-context-aware` | boolean | `false` | When set, only identifier-like tokens are converted; quoted strings are left untouched. |

> **Fidelity.** Byte-for-byte identical to lodash across BMP text. lodash's word
> regex is UTF-16-oriented, so astral characters (emoji, surrogate pairs) may split
> into words differently — a documented reduced-fidelity edge.

### Example

```bash
cchef to-camel-case -i 'XMLHttpRequest handler'
```

Output:

```
xmlHttpRequestHandler
```

## To Kebab case

Converts the input to kebab case (all lower case with dashes as word boundaries,
e.g. `this-is-kebab-case`), wrapping lodash's `kebabCase`. See
[To Camel case](#to-camel-case) for the shared options and fidelity notes.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--attempt-to-be-context-aware` | boolean | `false` | When set, only identifier-like tokens are converted; quoted strings are left untouched. |

### Example

```bash
cchef to-kebab-case -i 'XMLHttpRequest handler'
```

Output:

```
xml-http-request-handler
```

## To Snake case

Converts the input to snake case (all lower case with underscores as word
boundaries, e.g. `this_is_snake_case`), wrapping lodash's `snakeCase`. See
[To Camel case](#to-camel-case) for the shared options and fidelity notes.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--attempt-to-be-context-aware` | boolean | `false` | When set, only identifier-like tokens are converted; quoted strings are left untouched. |

### Simple example

```bash
cchef to-snake-case -i 'XMLHttpRequest handler'
```

Output:

```
xml_http_request_handler
```

### Complex example

With "context aware" enabled, only the variable name is transformed — the string
literal is preserved:

```bash
cchef to-snake-case -i 'var fooBar = "leave This";' --attempt-to-be-context-aware
```

Output:

```
var foo_bar = "leave This";
```

## XML Beautify

Indents and prettifies XML. Each nested element is placed on its own line indented
by depth; comments, CDATA and DOCTYPE content stay on one line, and each `xmlns`
declaration is put on its own line.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--indent-string` | string | `\t` (tab) | The indentation unit. Backslash escapes are interpreted, so `\t` yields a tab and `  ` (two spaces) indents with spaces. A step that begins with a digit falls back to four spaces (a vkbeautify quirk). |

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

Compresses XML: comments are
stripped (unless preserved) and whitespace before `xmlns` is collapsed, then
whitespace between tags is removed.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--preserve-comments` | boolean | `false` | When `true`, comments are kept instead of stripped. |

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
