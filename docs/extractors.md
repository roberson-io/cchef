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
| Extract IP addresses | `extract-ip-addresses` | [IP address](https://wikipedia.org/wiki/IP_address) |
| Extract MAC addresses | `extract-mac-addresses` | [MAC address](https://wikipedia.org/wiki/MAC_address) |
| Extract URLs | `extract-urls` | [URL](https://wikipedia.org/wiki/URL) |
| Extract dates | `extract-dates` | [Date / Time](date-time.md#extract-dates) |
| Extract domains | `extract-domains` | [Domain name](https://wikipedia.org/wiki/Domain_name) |
| Extract email addresses | `extract-email-addresses` | [Email address](https://wikipedia.org/wiki/Email_address) |
| Extract file paths | `extract-file-paths` | [Path (computing)](https://wikipedia.org/wiki/Path_(computing)) |
| Extract hashes | `extract-hashes` | [Cryptographic hash functions](https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions) |
| JPath expression | `jpath-expression` | [JSONPath](http://goessner.net/articles/JsonPath/) |
| Jsonata Query | `jsonata-query` | [JSONata](https://docs.jsonata.org/overview.html) |
| RAKE | `rake` | [Keyword extraction](https://wikipedia.org/wiki/Keyword_extraction) |
| Regular expression | `regular-expression` | [Utils](utils.md#regular-expression) |
| Strings | `strings` | [strings (Unix)](https://wikipedia.org/wiki/Strings_(Unix)) |
| Template | `template` | [Handlebars](https://handlebarsjs.com/) |
| XPath expression | `xpath-expression` | [XPath](https://wikipedia.org/wiki/XPath) |

## CSS selector

Extracts elements from an HTML/XML document using a CSS selector, serializing
each matched node and joining the results with a delimiter. Parsing is lenient
([`@xmldom/xmldom`](https://github.com/xmldom/xmldom) semantics) and selection
follows CSS3 ([`nwmatcher`](https://github.com/dperini/nwmatcher) semantics).

The input is parsed as **XML** (as CyberChef does), so it has a single root
element — content after the root element closes is ignored — and the five XML
entities plus numeric character references are decoded. Matched nodes are
serialized the way xmldom does: empty elements self-close (`<br/>`), attribute
values are normalized to double quotes, and text/attribute special characters are
escaped.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--css-selector` | string | (empty) | The selector. Supports type, `*`, `.class`, `#id`, `[attr]`/`[attr=val]` (with `~= \| ^= $= *= \|=`), the `>` `+` `~` combinators, comma groups, and structural pseudo-classes (`:first-child`, `:last-child`, `:nth-child(an+b)`, `:first-of-type`, `:not(...)`, `:empty`, `:root`, …). An empty selector yields empty output. |
| `--delimiter` | string | `\n` | Joins the serialized matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

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
whatever the container. Ten formats are recognized from their opening bytes —
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
| `--filename-optional` | string | (empty) | Recorded in the report. It plays no part in reading the file — the container is detected from the bytes. Surrounding spaces are trimmed, and an empty name is reported as `null`. |
| `--max-embedded-text-bytes` | number | `524288` | How much of an embedded text payload is kept. Payloads longer than this are cut and marked `truncated`. Values below 1024 are raised to it. |

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

Signatures are recognized for around 140 formats, but only some can be cut out:
carving needs an algorithm that knows where that format ends. A recognized
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
| `--images` | boolean | `true` | Search for image signatures. |
| `--video` | boolean | `true` | Search for video signatures. |
| `--audio` | boolean | `true` | Search for audio signatures. |
| `--documents` | boolean | `true` | Search for document signatures. |
| `--applications` | boolean | `true` | Search for executable signatures. |
| `--archives` | boolean | `true` | Search for archive and compressed-stream signatures. |
| `--miscellaneous` | boolean | `false` | Off by default: these signatures are short and match often by chance. |
| `--ignore-failed-extractions` | boolean | `true` | When off, a signature that matches but cannot be carved is reported as an error instead of being dropped. |
| `--minimum-file-size` | number | `100` | Carved files smaller than this are discarded, which prunes small false positives. |

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

## Extract IP addresses

Finds the IPv4 and IPv6 addresses in the input, one per line.

IPv4 is matched in decimal (four groups of 0–255) and in octal (four groups
written with a leading zero); an address is one form or the other, not a mixture.
Digits either side of a match are excluded, so `1.2.3.4.5.6.7.8` gives two
addresses rather than several overlapping ones — but as the operation's own
warning says, that means the reading may not be the one you intended, so check
the original.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--ipv4` | boolean | `true` | Match IPv4 addresses. |
| `--ipv6` | boolean | `false` | Match IPv6 addresses. |
| `--remove-local-ipv4-addresses` | boolean | `false` | Leave out the private ranges (`10.`, `172.16–31.`, `192.168.`) and the loopback range (`127.`). |
| `--display-total` | boolean | `false` | Put a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Order by the number the four parts make, so `9.0.0.1` comes before `10.0.0.1`. |
| `--unique` | boolean | `false` | Keep one of each. |

With neither version selected the output is empty.

Two things to know about the IPv6 pattern, both inherited from CyberChef and
confirmed against it. The check that an address shortens its run of zeros only
once looks ahead through the rest of the input rather than just the address, so
in `fe80::1 and ::1` only the second is found. And matching is done with a
back-tracking engine rather than Go's default one, because the pattern needs
look-behind and back-references that the default engine cannot express.

### Simple example

```bash
cchef extract-ip-addresses -i "Server 8.8.8.8 talked to 10.0.0.5 and 2001:db8::1"
```

Output:

```
8.8.8.8
10.0.0.5
```

IPv6 is off by default, which is why `2001:db8::1` is not listed.

### Complex example

Drop the private addresses, keep one of each, and count what is left:

```bash
cchef extract-ip-addresses -i "8.8.8.8 10.0.0.5 192.168.1.1 8.8.8.8" --remove-local-ipv4-addresses --unique --display-total
```

Output:

```
Total found: 1

8.8.8.8
```

## Extract MAC addresses

Pulls [MAC addresses](https://wikipedia.org/wiki/MAC_address) out of the input:
six pairs of hexadecimal digits separated throughout by colons or by hyphens.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--display-total` | boolean | `false` | Puts a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Orders the addresses by value rather than as text, so `0a:` comes before `ff:` and an address written with hyphens falls next to the same address written with colons. |
| `--unique` | boolean | `false` | Keeps one of each. The comparison is exact, so `AA:BB:…` and `aa:bb:…` are two different results. |

Separators may not be mixed within one address, and a run of more than six pairs
is matched from its start, so `01:23:45:67:89:ab:cd` yields `01:23:45:67:89:ab`.

### Simple example

```bash
cchef extract-mac-addresses -i "iface eth0 00:1B:44:11:3A:B7, iface eth1 00-1b-44-11-3a-b8"
```

Output:

```
00:1B:44:11:3A:B7
00-1b-44-11-3a-b8
```

### Complex example

```bash
cchef extract-mac-addresses -i "ff:ff:ff:ff:ff:ff 0a:0b:0c:0d:0e:0f ff:ff:ff:ff:ff:ff" --sort --unique --display-total
```

Output:

```
Total found: 2

0a:0b:0c:0d:0e:0f
ff:ff:ff:ff:ff:ff
```

## Extract URLs

Pulls [URLs](https://wikipedia.org/wiki/URL) out of the input. The protocol is
required — without it almost any dotted word would qualify, and the results would
be mostly noise. Use [Extract domains](#extract-domains) to find host names on
their own.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--display-total` | boolean | `false` | Puts a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Orders the URLs ignoring case. |
| `--unique` | boolean | `false` | Keeps one of each, comparing exactly. |

A path may contain a full stop, comma, exclamation mark or question mark, but not
as its last character, so a URL written at the end of a sentence does not take the
sentence's punctuation with it. A closing bracket, on the other hand, is an
ordinary path character: `(https://example.com/a)` yields `https://example.com/a)`.

### Simple example

```bash
cchef extract-urls -i "Docs at https://example.com/guide?v=2, mirror ftp://files.example.org:2121/pub."
```

Output:

```
https://example.com/guide?v=2
ftp://files.example.org:2121/pub
```

### Complex example

```bash
cchef extract-urls -i "http://b.example.com http://a.example.com/x http://b.example.com" --sort --unique --display-total
```

Output:

```
Total found: 2

http://a.example.com/x
http://b.example.com
```

## Extract domains

Pulls fully qualified [domain names](https://wikipedia.org/wiki/Domain_name) out
of the input. Paths are not included — use [Extract URLs](#extract-urls) for
those.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--display-total` | boolean | `false` | Puts a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Orders the names ignoring case. |
| `--unique` | boolean | `false` | Keeps one of each, comparing exactly, so `example.com` and `Example.com` both survive. |
| `--underscore-dmarc-dkim-etc` | boolean | `false` | Allows `_` in a label, which is how the records DMARC and DKIM publish are named. |

A name needs at least two labels, each no longer than 63 characters, and a
top-level label of two or more letters — so `example.co.uk` is found and `foo.c`
and `localhost` are not. Internationalised names are matched in their `xn--`
form; a name written in its own script is not, since the pattern is limited to
letters, digits and hyphens.

### Simple example

```bash
cchef extract-domains -i "Visit www.example.com or example.co.uk; not foo.c or localhost."
```

Output:

```
www.example.com
example.co.uk
```

### Complex example

Without `--underscore-dmarc-dkim-etc`, a name such as `_dmarc.example.com` is
reported as `example.com`, since the underscore ends the match:

```bash
cchef extract-domains -i "_dmarc.example.com and sel._domainkey.example.com" --underscore-dmarc-dkim-etc --sort --display-total
```

Output:

```
Total found: 2

_dmarc.example.com
sel._domainkey.example.com
```

## Extract email addresses

Pulls [email addresses](https://wikipedia.org/wiki/Email_address) out of the
input.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--display-total` | boolean | `false` | Puts a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Orders the addresses ignoring case. |
| `--unique` | boolean | `false` | Keeps one of each, comparing exactly. |

Both halves accept the whole range of characters above U+00A0, so an
internationalised address such as `用户@例子.广告` is found as readily as an ASCII
one. The part before the `@` may also be a quoted string, which is how an address
holding a character that is otherwise not allowed — including a second `@` — is
written. The part after may be a bracketed IPv4 address, checked to be in range:
`example@[127.0.0.1]` is an address, `example@[1.2.3.]` is not.

### Simple example

```bash
cchef extract-email-addresses -i 'Contact bob@example.com or "very.unusual@strange"@example.org; not a@ or @b.com.'
```

Output:

```
bob@example.com
"very.unusual@strange"@example.org
```

### Complex example

Sorting ignores case, so two spellings of one address sort together; uniquing does
not, so both are kept:

```bash
cchef extract-email-addresses -i "Zoe@Example.com, adam@example.com, zoe@example.com" --sort --unique --display-total
```

Output:

```
Total found: 3

adam@example.com
Zoe@Example.com
zoe@example.com
```

## Extract file paths

Pulls anything shaped like a Windows or UNIX
[path](https://wikipedia.org/wiki/Path_(computing)) out of the input.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--windows` | boolean | `true` | Look for paths starting at a drive letter. |
| `--unix` | boolean | `true` | Look for slash-separated paths. |
| `--display-total` | boolean | `false` | Puts a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Orders the paths ignoring case. |
| `--unique` | boolean | `false` | Keeps one of each, comparing exactly. |

With neither shape asked for the output is empty, and the total is not shown
either.

Both shapes are deliberately loose, and the UNIX one especially so: any run of
slash-separated words qualifies, and a Windows name may contain spaces, so a path
mentioned mid-sentence tends to take some of the sentence with it. This is
CyberChef's behavior and the reason the operation warns about false positives —
read the results against the original input rather than as a list of real paths.

A Windows path ends in at most one extension of up to six characters, so
`data.tar.gz` is reported as `data.tar`.

### Simple example

```bash
cchef extract-file-paths -i "Logs in C:\Users\me\logs\app.log and /var/log/syslog"
```

Output:

```
C:\Users\me\logs\app.log
/var/log/syslog
```

### Complex example

Either shape can be looked for on its own:

```bash
cchef extract-file-paths -i "C:\Users\me\notes.txt and /var/log/syslog" --windows=false
```

Output:

```
/var/log/syslog
```

And this is what the looseness looks like — the Windows match runs on past the
path into the following words, and the UNIX one keeps the full stop that ends the
sentence:

```bash
cchef extract-file-paths -i "Open C:\Windows\System32\drivers\etc\hosts now, or /etc/passwd."
```

Output:

```
C:\Windows\System32\drivers\etc\hosts now
/etc/passwd.
```

## Extract hashes

Pulls runs of lowercase hexadecimal of a given length out of the input — the
shape a [hash](https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions)
is usually written in. Nothing is verified: a run of the right length and the
right characters is reported whatever it actually is.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--hash-character-length` | number | `40` | How many characters a run must have. Ignored when `All hashes` is on. |
| `--all-hashes` | boolean | `false` | Look for every length in common use instead: 1, 2, 4, 8, 16, 32, 40, 48, 56, 64, 80, 96, 128 and 256 characters, shortest first. |
| `--display-total` | boolean | `false` | Puts a `Total Results: N` line before the results. |

Only lowercase hexadecimal is matched, so an uppercase digest is not found. Each
run must stand on its own — part of a longer run is not reported as a shorter
hash — which is also why a length that finds nothing under `All hashes` costs
nothing: the results simply come out grouped by length.

**Fidelity.** A hash character length below 1 finds nothing here. CyberChef
accepts 0, which turns the pattern into one matching the empty string and fills
the output with blank lines; every other unusable length already finds nothing
there, so this brings 0 into line with them. Negative and fractional lengths find
nothing in both.

### Simple example

```bash
cchef extract-hashes -i "MD5: 9e107d9d372bb6826bd81d3542a419d6" --hash-character-length 32
```

Output:

```
9e107d9d372bb6826bd81d3542a419d6
```

### Complex example

```bash
cchef extract-hashes -i "crc 1a2b md5 9e107d9d372bb6826bd81d3542a419d6 sha1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12" --all-hashes --display-total
```

Output:

```
Total Results: 3

1a2b
9e107d9d372bb6826bd81d3542a419d6
2fd4e1c67a2d28fced849ee1bb76e7391b93eb12
```

## JPath expression

Extracts values from a JSON document using a [JSONPath](http://goessner.net/articles/JsonPath/)
query, serializing each matched value and joining them with a delimiter. Matched
values are serialized like [`jsonpath-plus`](https://github.com/JSONPath-Plus/JSONPath),
including ECMAScript object key ordering.

Supported syntax: root `$`, child `.name` / `['name']`, wildcard `*` / `[*]`,
recursive descent `..`, array index and index-union `[0,2]`, slices `[start:end:step]`,
filters `[?(@.price < 10 && @.name == "x")]`, and script expressions `[(@.length-1)]`.
The magic `.length` property yields the length of an array or string.

> **Alternative to** [`jq`](https://jqlang.github.io/jq/) for path-style extraction. The query language is JSONPath, not jq's; see [Jq](code-tidy.md#jq) for the jq language itself.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--query` | string | (empty) | The JSONPath query. |
| `--result-delimiter` | string | `\n` | Joins the serialized matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

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

## Jsonata Query

Queries and reshapes a JSON document with a [JSONata](https://docs.jsonata.org)
expression — a language for selecting, filtering, computing over and rebuilding
JSON, in the spirit of XPath or `jq`.

The result is written back out as JSON, so a selected string comes back quoted.
An expression that selects nothing gives an empty string.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--query` | string | `string` | The JSONata expression. |

Input that is not a JSON document is reported as `Invalid input JSON`, and an
expression that cannot be read or run as `Invalid Jsonata Expression`.

**Fidelity.** CyberChef calls the reference JavaScript implementation; cchef uses
[gnata](https://github.com/RecoLabs/gnata), a Go implementation of JSONata 2.x.
Of CyberChef's 45 test cases, 37 match exactly and 6 more agree in every value
and type but write an object's keys in a different order — gnata keeps the order
internally but does not expose it. Two do not yet come out right: an expression
dividing one indexed value by another (`Numbers[0] / Numbers[4]`), which gnata
misreads the `/` in, and one case where the values of an object come back in a
different order.

### Simple example

```bash
cchef jsonata-query -i '{"FirstName":"Fred","Surname":"Smith","Age":28}' --query "Surname"
```

Output:

```
"Smith"
```

### Complex example

Filter a list and select a field from what is left:

```bash
cchef jsonata-query -i '{"Phone":[{"type":"home","number":"0203 544 1234"},{"type":"office","number":"01962 001234"},{"type":"mobile","number":"077 7700 1234"}]}' --query 'Phone[type="mobile"].number'
```

Output:

```
"077 7700 1234"
```

## RAKE

Rapid Automatic Keyword Extraction: scores the phrases of a piece of text and
lists them, highest first.

The text is split into sentences, and each sentence into the runs of words
between its stop words — those runs are the candidate phrases. A word scores by
how many words it shares a phrase with, divided by how often it occurs, which
favors words appearing in longer phrases over words that merely appear often. A
phrase scores as the sum of its words.

The output has two columns, a score and a phrase, under a heading row, so it can
be fed straight to [To Table](utils.md#to-table).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--word-delimiter-regex` | string | `\s` | Splits a sentence into words. |
| `--sentence-delimiter-regex` | string | `\.\s\|\n` | Splits the text into sentences. |
| `--stop-words` | string | the NLTK list | Comma-separated. Spaces are ignored, and matching is case-insensitive. A stop word ends the phrase it appears in. |

Note that the words are not stripped of punctuation, so a phrase at the end of a
sentence keeps the full stop that ended it.

### Simple example

```bash
cchef rake -i "Compatibility of systems of linear constraints over the set of natural numbers."
```

Output (first rows):

```
Scores: , Keywords:
4, linear constraints
4, natural numbers.
1, compatibility
1, systems
1, set
```

### Complex example

Split on commas rather than spaces, with a stop-word list of your own:

The heading line ends with a space, which CyberChef puts there and cchef keeps;
take care not to strip it when editing this page.

```bash
cchef rake -i "alpha,beta,the,gamma" --word-delimiter-regex "," --sentence-delimiter-regex "\n" --stop-words "the"
```

Output:

```
Scores: , Keywords: 
4, alpha beta
1, gamma
```

## Strings

Finds the runs of readable characters in binary data — the same idea as the Unix
`strings` command, with more control over what counts as readable.

A run has to be at least **Minimum length** characters long to be reported. What
counts as a character is set by **Match**, and how those characters are laid out
in the bytes is set by **Encoding**.

> **Alternative to** [`strings`](https://man7.org/linux/man-pages/man1/strings.1.html). cchef adds encoding and regex options `strings` lacks; the default minimum length and character set follow CyberChef, not `strings`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--encoding` | option | `Single byte` | `Single byte`, `16-bit littleendian`, `16-bit bigendian`, or `All`. The wide encodings look for text stored two bytes to the character by allowing a null byte on the appropriate side of each one; `All` allows one on either side, so it finds both kinds — and will run a one-byte and a two-byte region together into a single result. |
| `--minimum-length` | number | `4` | Shorter runs are passed over. |
| `--match` | option | `Alphanumeric + punctuation (A)` | See below. |
| `--display-total` | boolean | `false` | Put a `Total found: N` line before the results. |
| `--sort` | boolean | `false` | Order the results, ignoring case. |
| `--unique` | boolean | `false` | Keep one of each. |

The six kinds of run, three defined over ASCII and three over Unicode:

| Match | Takes |
| --- | --- |
| `Alphanumeric + punctuation (A)` | Letters, digits and the common punctuation. |
| `All printable chars (A)` | Everything from space to `~`. |
| `Null-terminated strings (A)` | As above, and the run must end with a null byte, which is included. |
| `Alphanumeric + punctuation (U)` | Any Unicode letter, number, punctuation or separator. |
| `All printable chars (U)` | Also marks and symbols, so currency signs and the like are kept. |
| `Null-terminated strings (U)` | As above, ending with a null byte. |

CyberChef's Match list also carries `[ASCII]` and `[Unicode]` headings naming the
two groups. They are not choices — picking one leaves the pattern with no
characters to repeat — so cchef offers only the six, as it does elsewhere for
grouped options.

One thing to know about how the input is read. The bytes are taken as UTF-8 when
the whole input is valid UTF-8, and as one character per byte when it is not.
Which of the two applies changes what counts as a letter, so adding a single
stray byte anywhere can change the reading of accented characters throughout.
This is CyberChef's behavior and cchef matches it.

### Simple example

Given `demo.bin`, which holds `Hello`, a null, `wor`, a null, then `Testing123!`:

```bash
cchef strings --in-file demo.bin
```

Output:

```
Hello
Testing123!
```

`wor` is left out because it is shorter than the default minimum of four.

### Complex example

The Unicode kinds differ over what is punctuation and what is a symbol. A
currency sign is a symbol, so it breaks a run under one and not the other:

```bash
cchef strings -i "Grüße€from€Köln" --match "Alphanumeric + punctuation (U)" --minimum-length 4
```

Output:

```
Grüße
from
Köln
```

```bash
cchef strings -i "Grüße€from€Köln" --match "All printable chars (U)" --minimum-length 4
```

Output:

```
Grüße€from€Köln
```

## Template

Renders a [Handlebars](https://handlebarsjs.com/) template against JSON input.

Values are written with `{{name}}`, which escapes them for HTML, or `{{{name}}}`,
which does not. Nested values are reached with dots (`{{a.b}}`), and a list is
indexed the same way (`{{xs.0}}`).

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--template-definition-handlebars` | string | (empty) | The template. |

The block helpers, and the values each makes available:

| Written | Does |
| --- | --- |
| `{{#if x}}…{{else}}…{{/if}}` | Renders one side or the other. Absent, `false`, `0`, `""` and an empty list all count as not present. |
| `{{#unless x}}…{{/unless}}` | The other way round. |
| `{{#each xs}}…{{else}}…{{/each}}` | Once per item of a list or field of an object, with `{{this}}`, `{{@index}}`, `{{@key}}`, `{{@first}}` and `{{@last}}`. The alternative renders when there is nothing to walk. |
| `{{#with o}}…{{/with}}` | Renders the body against `o`. |
| `{{#*inline "name"}}…{{/inline}}` then `{{> name}}` | Defines a piece of template and uses it by name. A partial written on an indented line keeps that indentation on every line it renders. |

Inside a block, `{{../x}}` reads the enclosing context and `{{@root.x}}` the
document as a whole; a value of the enclosing block is written `{{@../index}}`.
`{{! … }}` is a comment. A block tag, comment or partial alone on its line takes
that line with it, so a template written over several lines does not fill the
output with blank ones.

**Fidelity.** This covers the part of the Handlebars language a template can use
against JSON input. What it does not cover is everything needing a host language —
custom helpers, subexpressions, block
parameters and partial parameters — none of which CyberChef gives a way to
supply. It is also more forgiving than Handlebars about a few malformed
templates: `{{../@index}}`, for instance, renders nothing here where Handlebars
reports a parse error.

### Simple example

```bash
cchef template -i '{"name":"world"}' --template-definition-handlebars 'Hello, {{name}}!'
```

Output:

```
Hello, world!
```

### Complex example

Walk a list, and mark the last item:

```bash
cchef template --in-file users.json --template-definition-handlebars '{{#each users}}
- {{name}} ({{age}}){{#if @last}} — last{{/if}}
{{/each}}'
```

with `users.json`:

```json
{"users":[{"name":"Ada","age":36},{"name":"Grace","age":45}]}
```

Output:

```
- Ada (36)
- Grace (45) — last
```

Note the escaping, which is what keeps a rendered template plain text:

```bash
cchef template -i '{"c":"<b>&</b>"}' --template-definition-handlebars '{{c}} vs {{{c}}}'
```

Output:

```
&lt;b&gt;&amp;&lt;/b&gt; vs <b>&</b>
```

## XPath expression

Extracts nodes from an XML document using an XPath 1.0 query, serializing each
selected node and joining the results with a delimiter. It parses the document the
same way as [CSS selector](#css-selector) and evaluates the query with XPath 1.0
semantics.

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
| `--xpath` | string | (empty) | The XPath 1.0 query. Must evaluate to a node-set. |
| `--result-delimiter` | string | `\n` | Joins the serialized matches. Backslash escapes are interpreted (`\n` → newline, `\t` → tab). |

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
