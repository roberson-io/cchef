# Extractors

Operations that pull structured information out of text or markup. Some of these
operations belong to another category too, where their detailed description,
options and examples live: [Extract dates](date-time.md#extract-dates) is
documented under [Date / Time](date-time.md), and
[Regular expression](utils.md#regular-expression) under [Utils](utils.md).
Operations documented in full below are grouped here.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| CSS selector | `css-selector` | [CSS selectors](https://wikipedia.org/wiki/Cascading_Style_Sheets#Selector) |
| Extract dates | `extract-dates` | [Date / Time](date-time.md#extract-dates) |
| JPath expression | `jpath-expression` | [JSONPath](http://goessner.net/articles/JsonPath/) |
| Regular expression | `regular-expression` | [Utils](utils.md#regular-expression) |
| XPath expression | `xpath-expression` | [XPath](https://wikipedia.org/wiki/XPath) |

## CSS selector

Extracts elements from an HTML/XML document using a CSS selector, serialising
each matched node and joining the results with a delimiter. This is a
from-scratch port of CyberChef's operation, which wraps
[`@xmldom/xmldom`](https://github.com/xmldom/xmldom) (a lenient XML DOM parser)
and [`nwmatcher`](https://github.com/dperini/nwmatcher) (a CSS3 selector engine).
cchef reimplements the parser and serialiser and evaluates selection by
translating the CSS selector to XPath, reproducing the original's exact output
byte-for-byte.

The input is parsed as **XML** (as CyberChef does), so it has a single root
element — content after the root element closes is ignored — and the five XML
entities plus numeric character references are decoded. Matched nodes are
serialised the way xmldom does: empty elements self-close (`<br/>`), attribute
values are normalised to double quotes, and text/attribute special characters are
escaped.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| CSS selector | string | (empty) | The selector. Supports type, `*`, `.class`, `#id`, `[attr]`/`[attr=val]` (with `~= \| ^= $= *= \|=`), the `>` `+` `~` combinators, comma groups, and structural pseudo-classes (`:first-child`, `:last-child`, `:nth-child(an+b)`, `:first-of-type`, `:not(...)`, `:empty`, `:root`, …). An empty selector yields empty output. |
| Delimiter | string | `\n` | Joins the serialised matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

Element type and attribute names are matched case-insensitively (HTML semantics);
class names, ids and attribute values are case-sensitive, except the HTML
enumerated attributes (`type`, `dir`, `lang`, …) whose values are also
case-insensitive. As in CyberChef, the state/rendering-dependent pseudo-classes
(`:checked`, `:disabled`, `:enabled`, `:hover`, …) and the `checked`/`selected`
attribute selectors never match.

### Simple example

```bash
cchef css-selector -i "<ul><li>Home</li><li>About</li></ul>" --css-selector "li"
```

Output:

```
<li>Home</li>
<li>About</li>
```

### Complex example

Select `<a>` elements that carry both the `nav` class and an `href` attribute,
joined with ` | ` (note the single XML root — the elements are wrapped in a
`<div>` so all three siblings are queryable):

```bash
cchef css-selector -i '<div><a href="/x" class="nav">1</a><a>2</a><a href="/y" class="nav">3</a></div>' --css-selector "a.nav[href]" --delimiter " | "
```

Output:

```
<a href="/x" class="nav">1</a> | <a href="/y" class="nav">3</a>
```

## JPath expression

Extracts values from a JSON document using a [JSONPath](http://goessner.net/articles/JsonPath/)
query, serialising each matched value and joining them with a delimiter. CyberChef
wraps the [`jsonpath-plus`](https://github.com/JSONPath-Plus/JSONPath) npm library;
cchef reimplements the evaluator from scratch over an order-preserving JSON
representation (no new dependency), so matched values serialise byte-for-byte like
`jsonpath-plus`, including ECMAScript object key ordering.

Supported syntax: root `$`, child `.name` / `['name']`, wildcard `*` / `[*]`,
recursive descent `..`, array index and index-union `[0,2]`, slices `[start:end:step]`,
filters `[?(@.price < 10 && @.name == "x")]`, and script expressions `[(@.length-1)]`.
The magic `.length` property yields the length of an array or string.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Query | string | (empty) | The JSONPath query. |
| Result delimiter | string | `\n` | Joins the serialised matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

Invalid input is reported as `Invalid input JSON: <message>`; a malformed query as
`Invalid JPath expression: <message>`.

### Simple example

```bash
cchef jpath-expression -i '{"store":{"books":[{"title":"Go"},{"title":"Rust"}]}}' --query "$.store.books[*].title"
```

Output:

```
"Go"
"Rust"
```

### Complex example

Filter by a predicate and join with `, `:

```bash
cchef jpath-expression -i '{"books":[{"title":"Cheap","price":5},{"title":"Pricey","price":25},{"title":"Mid","price":9}]}' --query '$..books[?(@.price<10)].title' --result-delimiter ", "
```

Output:

```
"Cheap", "Mid"
```

## XPath expression

Extracts nodes from an XML document using an XPath 1.0 query, serialising each
selected node and joining the results with a delimiter. This is a from-scratch
port of CyberChef's operation, which wraps [`@xmldom/xmldom`](https://github.com/xmldom/xmldom)
and the npm [`xpath`](https://github.com/goto100/xpath) library. cchef reuses the
same from-scratch XML parser and serialiser as [CSS selector](#css-selector) and
evaluates the query with [`antchfx/xpath`](https://github.com/antchfx/xpath) over
the parsed tree.

Only **node-set** queries are supported, matching the original: a query that
evaluates to a number, string or boolean (e.g. `count(//a)`, `string(//a)`,
`1+2`) is rejected with `Invalid XPath. Details:\nCannot convert <type> to
nodeset.`. Selected nodes are serialised with the same `node.toString()` rules —
elements as their markup, attributes as ` name="value"` (with a leading space),
text as its escaped content, comments as `<!--…-->`, and CDATA as
`<![CDATA[…]]>`. As with CSS selector, the document is parsed as XML with a single
root element.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| XPath | string | (empty) | The XPath 1.0 query. Must evaluate to a node-set. |
| Result delimiter | string | `\n` | Joins the serialised matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

> The rarely-used `processing-instruction()` node test is not filtered by the
> underlying engine (it matches every node); every other node test, including
> `comment()`, behaves as CyberChef does.

### Simple example

```bash
cchef xpath-expression -i "<r><a>one</a><a>two</a></r>" --xpath "//a"
```

Output:

```
<a>one</a>
<a>two</a>
```

### Complex example

Select the `<title>` of the `<book>` whose `id` attribute is `2`, using an
attribute predicate:

```bash
cchef xpath-expression -i '<books><book id="1"><title>Go</title></book><book id="2"><title>Rust</title></book></books>' --xpath '//book[@id="2"]/title' --result-delimiter " | "
```

Output:

```
<title>Rust</title>
```
