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
| Regular expression | `regular-expression` | [Utils](utils.md#regular-expression) |

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
