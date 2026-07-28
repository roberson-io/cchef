# Extractors

Operations that pull structured information out of text or markup. Some of these
operations belong to another category too, where their detailed description,
options and examples live: [Extract dates](date-time.md#extract-dates) is
documented under [Date / Time](date-time.md),
[Extract EXIF](multimedia.md#extract-exif) under [Multimedia](multimedia.md), and
[Regular expression](utils.md#regular-expression) under [Utils](utils.md).
Operations documented in full below are grouped here.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| CSS selector | `css-selector` | [CSS selectors](https://wikipedia.org/wiki/Cascading_Style_Sheets#Selector) |
| Extract Audio Metadata | `extract-audio-metadata` | [Audio file format](https://wikipedia.org/wiki/Audio_file_format) |
| Extract EXIF | `extract-exif` | [Multimedia](multimedia.md#extract-exif) |
| Extract Files | `extract-files` | [File carving](https://forensics.wiki/file_carving) |
| Extract ID3 | `extract-id3` | [ID3](https://wikipedia.org/wiki/ID3) |
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

## Extract Audio Metadata

Reads the metadata out of an audio file and reports it as one JSON document,
whatever the container. Ten formats are recognised from their opening bytes —
MP3, WAV (including BWF and BW64), FLAC, OGG, Opus, AAC, AC3, WMA, MP4/M4A and
AIFF — and each is read with the systems that format carries.

The report always has the same shape, so the same fields can be read whatever
the file was:

| Section | Holds |
| --- | --- |
| `artifact` | The filename given, the length in bytes, and the container detected. |
| `detections` | Which metadata systems were found, such as `id3v2`, `vorbis_comments` or `asf_content_desc`. |
| `tags.common` | Ten tags every format is boiled down to: title, artist, album, date, track, genre, comment, composer, copyright, language. |
| `tags.raw` | Everything as the format itself records it, under a key per system. |
| `embedded` | Payloads carried inside the file — cover art, encapsulated objects, XML chunks. |
| `provenance` | Content credentials (C2PA), when a carrier for them is present. |
| `errors` | Anything that could not be read. |

A tag in `tags.common` is filled in by the first system that names it, so a file
carrying both an ID3v2 and an ID3v1 tag takes the ID3v2 value and keeps it.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Filename (optional) | string | (empty) | Recorded in the report. It plays no part in reading the file — the container is detected from the bytes. Surrounding spaces are trimmed, and an empty name is reported as `null`. |
| Max embedded text bytes (iXML/axml/etc) | number | `524288` | How much of an embedded text payload is kept. Payloads longer than this are cut and marked `truncated`. Values below 1024 are raised to it. |

Input that is not audio is not an error: the container is reported as `unknown`
and the reason is recorded under `errors`.

### Simple example

```bash
cchef extract-audio-metadata --in-file galway.flac --filename-optional "galway.flac" | cchef json-beautify
```

Output (abridged):

```
{
    "schema_version": "audio-meta-1.0",
    "artifact": {
        "filename": "galway.flac",
        "byte_length": 132,
        "container": {
            "type": "flac",
            "brand": null,
            "mime": "audio/flac"
        }
    },
    "detections": {
        "metadata_systems": [
            "flac_metablocks",
            "vorbis_comments"
        ],
        "provenance_systems": []
    },
    "tags": {
        "common": {
            "title": "Galway",
            "artist": "Kevin MacLeod",
            ...
        }
    }
}
```

### Complex example

The common tags are the quickest way to read a file, whatever it is. Pull them
out with [JPath expression](#jpath-expression):

```bash
cchef bake --in-file galway.flac -r recipe.json
```

where `recipe.json` chains the two operations:

```json
[
  { "op": "Extract Audio Metadata", "args": ["galway.flac", 524288] },
  { "op": "JPath expression", "args": ["$.tags.common", "\n"] }
]
```

Output:

```
{"title":"Galway","artist":"Kevin MacLeod","album":null,"date":null,"track":null,"genre":null,"comment":null,"composer":null,"copyright":null,"language":null}
```

## Extract Files

Scans the input for file signatures and cuts out every embedded file it finds —
file carving. Unlike [Detect File Type](forensics.md#detect-file-type), which
only asks what the input *starts* with, this searches the whole buffer, so it
recovers files appended to one another, embedded in a document, or left in slack
space.

Because it produces several files rather than one output stream, it must be the
last step in a recipe and needs `--out-dir` to write them into. Each file is
named `extracted_at_0x<offset>.<extension>`, where the offset is where its
signature matched.

Signatures are recognised for around 140 formats, but only some can be cut out:
carving needs an algorithm that knows where that format ends. A recognised
format with no carving algorithm is passed over silently.

The 34 that can be carved are the same set CyberChef advertises, each listed
under every extension it goes by:

| | | |
| --- | --- | --- |
| `JPG,JPEG,JPE,THM,MPO` | `ZIP` | `EVT` |
| `GIF` | `TAR` | `EVTX` |
| `PNG` | `GZ` | `DMP` |
| `WEBP` | `BZ2` | `PF` |
| `BMP` | `ZLIB` | `PLIST` |
| `ICO` | `XZ` | `KEYCHAIN` |
| `TGA` | `JAR` | `LNK` |
| `FLV` | `LZOP,LZO` | `DOCX,XLSX,PPTX` |
| `WAV` | `DEB` | `EPUB` |
| `MP3` | `SQLITE` | `DYLIB` |
| `PDF` | `EXE,DLL,DRV,VXD,SYS,OCX,VBX,COM,FON,SCR` | |
| `RTF` | `ELF,BIN,AXF,O,PRX,SO` | |

Several of these share one algorithm — a `DOCX`, `EPUB` and `JAR` are all ZIP
archives, and every name in the `EXE` row is a Windows portable executable — so
the 34 entries are carved by 32 distinct algorithms.

One caveat. `PF` covers two formats: the pre-Windows 10 prefetch file, which
records its length and is carved exactly, and the Windows 10 one, which is
compressed and records only the size its contents take once expanded. Nothing in
the latter says how long the compressed data is, so its end cannot be found
without decompressing it; cchef reports that rather than guessing. CyberChef
advertises it too, but reads the file's `MAM` signature as a big-endian length
and fails out of bounds.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Images | boolean | `true` | Search for image signatures. |
| Video | boolean | `true` | Search for video signatures. |
| Audio | boolean | `true` | Search for audio signatures. |
| Documents | boolean | `true` | Search for document signatures. |
| Applications | boolean | `true` | Search for executable signatures. |
| Archives | boolean | `true` | Search for archive and compressed-stream signatures. |
| Miscellaneous | boolean | `false` | Off by default: these signatures are short and match often by chance. |
| Ignore failed extractions | boolean | `true` | When off, a signature that matches but cannot be carved is reported as an error instead of being dropped. |
| Minimum File Size | number | `100` | Carved files smaller than this are discarded, which prunes small false positives. |

Note that the same bytes can match more than one signature, and a signature can
match inside another file's data — an archive's members are often found as
separate streams. Both are expected: carving reports candidates, and the
`Minimum File Size` floor is the first line of defence against noise.

### Simple example

Given `report.bin`, a PNG with a ZIP archive appended to it:

```bash
cchef extract-files --in-file report.bin --out-dir carved
```

Output:

```
carved/extracted_at_0x0.png
carved/extracted_at_0x159.zip
carved/extracted_at_0x1d5.zip
```

Both the archive and the second member's local header inside it are reported;
the first is the whole archive.

### Complex example

Restrict the scan to images, so the archive is passed over:

```bash
cchef extract-files --in-file report.bin --out-dir carved --archives=false --documents=false --applications=false --audio=false --video=false
```

Output:

```
carved/extracted_at_0x0.png
```

## Extract ID3

Reads the ID3 metadata tag an MP3 file can carry — title, artist, album, track
number and so on — and reports it as JSON.

The output names the tag version, its flags and its length, then each frame it
holds under the four-character (or, in ID3v2.2, three-character) identifier the
format uses, with the identifier's meaning and the frame's contents.

A frame's `Data` is the frame's bytes with the first left out, since that byte
says how the rest is encoded rather than being part of it, and the remaining
bytes are reported as written rather than decoded. Text frames are terminated
with a null byte, which is why most values end in `\u0000`.

All three tag versions are read. Note that the lengths are stored differently
between them: the tag's own length is always written as seven-bit groups, and so
are frame lengths from ID3v2.4, but ID3v2.2 and ID3v2.3 write frame lengths as
ordinary integers. CyberChef reads every length as seven-bit groups, so it
cannot read an ID3v2.3 frame of 128 bytes or more, nor any tag over 16 KB;
cchef reads each as the format specifies.

This operation takes no options.

### Simple example

```bash
cchef extract-id3 --in-file tagged.mp3
```

Output:

```
{"Type":"ID3","Version":"4.0","Flags":"0","Size":"130","Tags":{"TIT2":{"Size":"12","Description":"Title/songname/content description","Data":"Test Title\u0000"},"TPE1":{"Size":"13","Description":"Lead performer(s)/Soloist(s)","Data":"Test Artist\u0000"},"TALB":{"Size":"12","Description":"Album/Movie/Show title","Data":"Test Album\u0000"},"TDRC":{"Size":"6","Description":"Recording time","Data":"2026\u0000"},"TRCK":{"Size":"3","Description":"Track number/Position in set","Data":"3\u0000"},"TSSE":{"Size":"14","Description":"Software/Hardware and settings used for encoding","Data":"Lavf61.1.100\u0000"}}}
```

### Complex example

The output is compact JSON, so pipe it through
[JSON Beautify](code-tidy.md#json-beautify) to read it:

```bash
cchef extract-id3 --in-file tagged.mp3 | cchef json-beautify
```

Output:

```
{
    "Type": "ID3",
    "Version": "3.0",
    "Flags": "0",
    "Size": "66",
    "Tags": {
        "TIT2": {
            "Size": "7",
            "Description": "Title/songname/content description",
            "Data": "Small\u0000"
        },
        "TPE1": {
            "Size": "5",
            "Description": "Lead performer(s)/Soloist(s)",
            "Data": "Ann\u0000"
        },
        "TSSE": {
            "Size": "14",
            "Description": "Software/Hardware and settings used for encoding",
            "Data": "Lavf61.1.100\u0000"
        }
    }
}
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
