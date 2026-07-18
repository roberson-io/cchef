# Utils

General-purpose text utilities.

> Operations are listed alphabetically.

| Operation | Subcommand |
| --- | --- |
| Add line numbers | `add-line-numbers` |
| Alternating Caps | `alternating-caps` |
| Convert area | `convert-area` |
| Convert co-ordinate format | `convert-co-ordinate-format` |
| Convert data units | `convert-data-units` |
| Convert distance | `convert-distance` |
| Convert mass | `convert-mass` |
| Convert speed | `convert-speed` |
| Count occurrences | `count-occurrences` |
| Diff | `diff` |
| Drop bytes | `drop-bytes` |
| Drop nth bytes | `drop-nth-bytes` |
| Escape string | `escape-string` |
| Expand alphabet range | `expand-alphabet-range` |
| File Tree | `file-tree` |
| Filter | `filter` |
| Find / Replace | `find-replace` |
| From Case Insensitive Regex | `from-case-insensitive-regex` |
| Fuzzy Match | `fuzzy-match` |
| Get All Casings | `get-all-casings` |
| Hamming Distance | `hamming-distance` |
| Head | `head` |
| Levenshtein Distance | `levenshtein-distance` |
| Offset checker | `offset-checker` |
| Pad lines | `pad-lines` |
| Parse colour code | `parse-colour-code` |
| Parse ObjectID timestamp | `parse-objectid-timestamp` |
| Parse UNIX file permissions | `parse-unix-file-permissions` |
| Pseudo-Random Number Generator | `pseudo-random-number-generator` |
| Regular expression | `regular-expression` |
| Remove ANSI Escape Codes | `remove-ansi-escape-codes` |
| Remove line numbers | `remove-line-numbers` |
| Remove null bytes | `remove-null-bytes` |
| Remove whitespace | `remove-whitespace` |
| Reverse | `reverse` |
| Show on map | `show-on-map` |
| Shuffle | `shuffle` |
| Sleep | `sleep` |
| Sort | `sort` |
| Split | `split` |
| Swap case | `swap-case` |
| Tail | `tail` |
| Take bytes | `take-bytes` |
| Take nth bytes | `take-nth-bytes` |
| To Case Insensitive Regex | `to-case-insensitive-regex` |
| To Lower case | `to-lower-case` |
| To Table | `to-table` |
| To Upper case | `to-upper-case` |
| Unescape string | `unescape-string` |
| Unique | `unique` |
| Wrap | `wrap` |

---

## Add line numbers

Adds a right-aligned line number to the start of each line.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--offset` | number | `0` | Added to each line number. |

**Simple example**

```bash
printf 'a\nb\nc' | cchef add-line-numbers
```

Output:

```
1 a
2 b
3 c
```

## Alternating Caps

Applies aLtErNaTiNg capitalisation, starting with lower case (non-letters are
left unchanged). Takes no options.

**Simple example**

```bash
cchef alternating-caps -i 'hello world'
```

Output:

```
hElLo WoRlD
```

## Convert area

Converts an area between units (square metres, hectares, acres, barns, …).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | (first unit) | The unit of the input value. |
| `--output-units` | option | (first unit) | The unit to convert to. |

Run `cchef convert-area --help` for the full list of units.

**Simple example**

```bash
cchef convert-area --input-units 'Square foot (sq ft)' --output-units 'Square metre (sq m)' -i 100
```

Output:

```
9.290304
```

## Convert co-ordinate format

Converts geographic co-ordinates between Decimal Degrees, Degrees Decimal Minutes,
Degrees Minutes Seconds, Geohash, MGRS, Ordnance Survey National Grid and UTM.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Auto` | Input format (`Auto` detects it). |
| `--input-delimiter` | option | `Auto` | How the latitude/longitude are separated. |
| `--output-format` | option | `Degrees Minutes Seconds` | Format to convert to. |
| `--output-delimiter` | option | `Space` | Separator for the output pair. |
| `--include-compass-directions` | option | `None` | `None`, `Before`, or `After` the value. |
| `--precision` | number | `3` | Number of decimal places. |

> UTM easting/northing are computed with a different projection library than
> CyberChef, so the final (sub-millimetre) digit may occasionally differ at high
> precision. MGRS, OSNG, Geohash and the lat/lon formats match exactly.

**Simple example**

```bash
cchef convert-co-ordinate-format -i '51.5074, -0.1278' --output-format 'Degrees Minutes Seconds'
```

Output:

```
51° 30' 26.64" -0° 7' 40.08" 
```

## Convert data units

Converts a quantity of data between units (bits, bytes, and their binary/decimal
multiples such as kibibytes and megabytes).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | (first unit) | The unit of the input value. |
| `--output-units` | option | (first unit) | The unit to convert to. |

Run `cchef convert-data-units --help` for the full list of units.

**Simple example**

```bash
cchef convert-data-units --input-units 'Gibibytes (GiB)' --output-units 'Mebibytes (MiB)' -i 1
```

Output:

```
1024
```

## Convert distance

Converts a distance between units (metres, miles, light-years, …).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | (first unit) | The unit of the input value. |
| `--output-units` | option | (first unit) | The unit to convert to. |

Run `cchef convert-distance --help` for the full list of units.

**Simple example**

```bash
cchef convert-distance --input-units 'Miles (mi)' --output-units 'Kilometers (km)' -i 1
```

Output:

```
1.609344
```

## Convert mass

Converts a mass between units (grams, pounds, tonnes, solar masses, …).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | (first unit) | The unit of the input value. |
| `--output-units` | option | (first unit) | The unit to convert to. |

Run `cchef convert-mass --help` for the full list of units.

**Simple example**

```bash
cchef convert-mass --input-units 'Pound (lb)' --output-units 'Gram (g)' -i 1
```

Output:

```
453.59237
```

## Convert speed

Converts a speed between units (m/s, mph, knots, the speed of light, …).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | (first unit) | The unit of the input value. |
| `--output-units` | option | (first unit) | The unit to convert to. |

Run `cchef convert-speed --help` for the full list of units.

**Simple example**

```bash
cchef convert-speed --input-units 'Kilometres per hour (km/h)' --output-units 'Metres per second (m/s)' -i 100
```

Output:

```
27.78
```

## Count occurrences

Counts how many times a search term appears. Output is the count.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--search-string` | string | (empty) | The term to count. |
| `--search-string-type` | option | `Regex` | `Regex` (case-insensitive), `Extended (\n, \t, \x...)`, or `Simple string`. |

**Simple example**

```bash
cchef count-occurrences --search-string foo --search-string-type 'Simple string' -i 'foofoofoo'
```

Output:

```
3
```

## Diff

Compares two samples (separated by a delimiter) and highlights the differences
with `<ins>` and `<del>` tags.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separates the two samples (escape sequences are expanded). |
| `--diff-by` | option | `Character` | `Character`, `Word`, `Line`, `Sentence`, `CSS`, or `JSON`. |
| `--show-added` | bool | `true` | Wrap added text in `<ins>`. |
| `--show-removed` | bool | `true` | Wrap removed text in `<del>`. |
| `--show-subtraction` | bool | `false` | Show only the differences (omit unchanged text). |
| `--ignore-whitespace` | bool | `false` | Ignore whitespace (Word and Line modes). |

> Diffs use a Go diff library rather than CyberChef's jsdiff. All modes match
> CyberChef except Word with `--ignore-whitespace`, which may attach trailing
> whitespace to a change differently.

**Simple example**

```bash
cchef diff -i 'the quick brown fox|the quick red fox' --sample-delimiter '|' --diff-by Word
```

Output:

```
the quick <del>brown</del><ins>red</ins> fox
```

## Drop bytes

Deletes a range of bytes from the input.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--start` | number | `0` | Start offset (negative counts from the end). |
| `--length` | number | `5` | Number of bytes to drop. |
| `--apply-to-each-line` | bool | `false` | Apply per line. |

**Simple example**

```bash
cchef drop-bytes --start 0 --length 6 -i 'Hello World'
```

Output:

```
World
```

## Drop nth bytes

Drops every nth byte, starting at a given offset.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--drop-every` | number | `4` | Drop every nth byte. |
| `--starting-at` | number | `0` | Offset to start from. |
| `--apply-to-each-line` | bool | `false` | Reset the count per line. |

**Simple example**

```bash
cchef drop-nth-bytes --drop-every 2 -i '0123456789'
```

Output:

```
13579
```

## Escape string

Escapes special characters in a string (e.g. quotes, backslashes, control and
non-ASCII characters).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--escape-level` | option | `Special chars` | `Special chars` (escape control/quote/non-ASCII), `Everything` (also printable ASCII), or `Minimal` (only quote, backslash and control chars; leaves non-ASCII raw). |
| `--escape-quote` | option | `Single` | Which quote character to escape: `Single`, `Double`, or `Backtick`. |
| `--json-compatible` | bool | `false` | Produce a JSON-style string (wrapped in double quotes, `\uNNNN` escapes). |
| `--es6-compatible` | bool | `true` | Use `\u{…}` for astral characters (otherwise UTF-16 surrogate pairs). |
| `--uppercase-hex` | bool | `false` | Use uppercase hex digits. |

> This is a from-scratch implementation of CyberChef's jsesc-backed behaviour,
> validated against CyberChef but not guaranteed identical in every edge case.

**Simple example**

```bash
cchef escape-string -i "it's a café"
```

Output:

```
it\'s a caf\xe9
```

## Expand alphabet range

Expands an alphabet range specification into its characters.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | string | (empty) | Delimiter to join the expanded characters with. |

**Simple example**

```bash
cchef expand-alphabet-range -i 'a-j'
```

Output:

```
abcdefghij
```

## File Tree

Renders a list of file paths as a directory tree.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--file-path-delimiter` | string | `/` | Separator between path components. |
| `--delimiter` | option | `Line feed` | Separator between the input paths. |

**Simple example**

```bash
printf 'home/a.txt\nhome/b/c.txt\nhome/b/d.txt' | cchef file-tree
```

Output:

```
home
|---a.txt
|---b
|   |---c.txt
|   |---d.txt
```

## Filter

Splits the input on a delimiter and keeps only the sections matching a regular
expression (grep-like).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |
| `--regex` | string | (empty) | The regular expression to match against each section. |
| `--invert-condition` | bool | `false` | Keep the sections that do *not* match. |

**Simple example**

```bash
printf 'apple\nbanana\ncherry' | cchef filter --regex a
```

Output:

```
apple
banana
```

## Find / Replace

Replaces matches of a pattern with a replacement string. The pattern can be a
simple string, an extended string (with `\n`, `\t`, `\xNN` escapes), or a regex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--find` | string | (empty) | The pattern to search for. |
| `--find-type` | option | `Regex` | `Regex`, `Extended (\n, \t, \x...)`, or `Simple string`. |
| `--replace` | string | (empty) | The replacement (supports `$1`, `$2`, … group references). |
| `--global-match` | bool | `true` | Replace all matches (vs. just the first). |
| `--case-insensitive` | bool | `false` | Case-insensitive matching. |
| `--multiline-matching` | bool | `true` | `^`/`$` match at line boundaries. |
| `--dot-matches-all` | bool | `false` | `.` matches newlines. |

**Simple string**

```bash
cchef find-replace --find 'foo' --find-type 'Simple string' --replace 'bar' -i 'foofoo'
```

Output:

```
barbar
```

**Regex with capture groups**

```bash
cchef find-replace --find '(\w+) (\w+)' --find-type Regex --replace '$2 $1' -i 'John Smith'
```

Output:

```
Smith John
```

## From Case Insensitive Regex

Converts a case-insensitive regex of the form `[aA][bB]` back to a case-sensitive
one (`ab`). Character classes with distinct letters are left unchanged. Takes no
options.

**Simple example**

```bash
cchef from-case-insensitive-regex -i '[tT][eE][sS][tT]'
```

Output:

```
test
```

## Fuzzy Match

Conducts a fuzzy search to find a pattern within the input, wrapping each match in
`<b>` tags inside an alternating `hl1`/`hl2` `<span>`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--search` | string | (empty) | The pattern to search for (escape sequences are expanded). |
| `--sequential-bonus` | number | `15` | Bonus for adjacent matches. |
| `--separator-bonus` | number | `30` | Bonus if a match follows a separator. |
| `--camel-bonus` | number | `30` | Bonus for an uppercase match after a lowercase letter. |
| `--first-letter-bonus` | number | `15` | Bonus if the first letter is matched. |
| `--leading-letter-penalty` | number | `-5` | Penalty per letter before the first match. |
| `--max-leading-letter-penalty` | number | `-15` | Cap on the leading-letter penalty. |
| `--unmatched-letter-penalty` | number | `-1` | Penalty per unmatched letter. |

**Simple example**

```bash
cchef fuzzy-match -i 'test input' --search tein
```

Output:

```
<span class="hl1"><b>te</b>st <b>in</b></span>put
```

## Get All Casings

Outputs every combination of upper- and lower-case for the input, one per line.
Takes no options.

> The number of results doubles with each character, so keep the input short.

**Simple example**

```bash
cchef get-all-casings -i ab
```

Output:

```
ab
Ab
aB
AB
```

## Hamming Distance

Computes the Hamming distance between two equal-length samples (the number of
positions at which they differ), by byte or by bit.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | string | `\n\n` | Separates the two samples (escape sequences are expanded). |
| `--unit` | option | `Byte` | Count differing `Byte`s or `Bit`s. |
| `--input-type` | option | `Raw string` | `Raw string` or `Hex`. |

**Simple example**

```bash
printf 'karolin\n\nkathrin' | cchef hamming-distance --unit Bit
```

Output:

```
9
```

## Head

Keeps only the first N sections (lines) of the input. A negative N drops the
last |N| sections.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |
| `--number` | number | `10` | Number of sections to keep. |

**Simple example**

```bash
printf 'a\nb\nc\nd\ne' | cchef head --number 2
```

Output:

```
a
b
```

## Levenshtein Distance

Computes the Levenshtein (edit) distance between two samples — the minimum number
of single-character edits to turn one into the other. Output is the distance.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n` | Separates the two samples (escape sequences are expanded). |
| `--insertion-cost` | number | `1` | Cost of an insertion. |
| `--deletion-cost` | number | `1` | Cost of a deletion. |
| `--substitution-cost` | number | `1` | Cost of a substitution. |

**Simple example**

```bash
printf 'kitten\nsitting' | cchef levenshtein-distance
```

Output:

```
3
```

## Offset checker

Compares multiple samples and highlights (with `<span>` tags) the byte offsets that
are identical across all of them.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separates the samples (escape sequences are expanded). |

**Simple example**

```bash
printf 'hello world\nhello there' | cchef offset-checker --sample-delimiter '\n'
```

Output:

```
<span class='hl5'>hello </span>world
<span class='hl5'>hello </span>there
```

## Pad lines

Adds padding characters to the start or end of each line.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--position` | option | `Start` | `Start` or `End`. |
| `--length` | number | `5` | Number of padding characters to add. |
| `--character` | string | (space) | The padding character. |

**Simple example**

```bash
printf 'ab\ncd' | cchef pad-lines --position Start --length 2 --character '*'
```

Output:

```
**ab
**cd
```

## Parse colour code

Parses a colour code in a standard format (hex, RGB, RGBA, HSL, HSLA, CMYK) and
prints all the other representations (plus an HTML colour-picker for the GUI).
Takes no options.

**Simple example**

```bash
cchef parse-colour-code -i '#ff0000'
```

Output:

```
<div id="colorpicker" style="white-space: normal;"></div>
Hex:  #ff0000
RGB:  rgb(255, 0, 0)
RGBA: rgba(255, 0, 0, 1)
HSL:  hsl(0, 100%, 50%)
HSLA: hsla(0, 100%, 50%, 1)
CMYK: cmyk(0.00, 1.00, 1.00, 0.00)
...
```

## Parse ObjectID timestamp

Extracts the embedded creation timestamp from a MongoDB ObjectID (the first 4
bytes), as an ISO 8601 string. Takes no options.

**Simple example**

```bash
cchef parse-objectid-timestamp -i '507f1f77bcf86cd799439011'
```

Output:

```
2012-10-17T21:13:27.000Z
```

## Parse UNIX file permissions

Parses a UNIX file permission string — octal (e.g. `755`) or textual (e.g.
`drwxr-xr-x`) — and prints the textual/octal representations, any special flags,
and a read/write/execute matrix. Takes no options.

**Simple example**

```bash
cchef parse-unix-file-permissions -i 755
```

Output:

```
Textual representation: -rwxr-xr-x
Octal representation:   0755

 +---------+-------+-------+-------+
 |         | User  | Group | Other |
 +---------+-------+-------+-------+
 |    Read |   X   |   X   |   X   |
 +---------+-------+-------+-------+
 |   Write |   X   |       |       |
 +---------+-------+-------+-------+
 | Execute |   X   |   X   |   X   |
 +---------+-------+-------+-------+
```

## Pseudo-Random Number Generator

Generates a number of cryptographically-secure random bytes (using Go's
`crypto/rand`) and outputs them in the chosen representation.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--number-of-bytes` | number | `32` | How many random bytes to generate. |
| `--output-as` | option | `Hex` | `Hex`, `Integer`, `Byte array`, or `Raw`. |

> Output is non-deterministic by design.

**Simple example**

```bash
cchef pseudo-random-number-generator --number-of-bytes 4 --output-as Hex
```

Output:

```
1ed9ec81
```

## Regular expression

Searches the input with a user-supplied regular expression (Go's RE2 syntax),
highlighting or listing the matches.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--built-in-regexes` | string | (empty) | Ignored at run time (mirrors CyberChef's arg); supply the pattern via `--regex`. |
| `--regex` | string | (empty) | The regular expression. |
| `--case-insensitive` | bool | `true` | Case-insensitive matching. |
| `--and-match-at-newlines` | bool | `true` | `^` and `$` match at line boundaries. |
| `--dot-matches-all` | bool | `false` | `.` matches newlines. |
| `--unicode-support` | bool | `false` | (Accepted for compatibility.) |
| `--astral-support` | bool | `false` | (Accepted for compatibility.) |
| `--display-total` | bool | `false` | Prefix the output with the match count. |
| `--output-format` | option | `Highlight matches` | `Highlight matches`, `List matches`, `List capture groups`, or `List matches with capture groups`. |

> RE2 does not support lookaround or backreferences, so some XRegExp-only patterns
> (including a few of CyberChef's built-in regexes) will not compile.

**Simple example**

```bash
cchef regular-expression -i 'a1b22c333' --regex '\d+' --output-format 'List matches'
```

Output:

```
1
22
333
```

## Remove ANSI Escape Codes

Removes ANSI escape codes (e.g. terminal colour codes) from the input. Takes no
options.

**Simple example**

```bash
printf '\x1b[31mred\x1b[0m text' | cchef remove-ansi-escape-codes
```

Output:

```
red text
```

## Remove line numbers

Removes line numbers from the beginning of each line, where they can be found.
Takes no options.

**Simple example**

```bash
printf '1 a\n2 b\n3 c' | cchef remove-line-numbers
```

Output:

```
a
b
c
```

## Remove null bytes

Removes all null bytes (`0x00`) from the input. Takes no options.

**Simple example**

```bash
printf 'a\x00b\x00c' | cchef remove-null-bytes
```

Output:

```
abc
```

## Remove whitespace

Optionally removes spaces, carriage returns, line feeds, tabs, form feeds and
full stops.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--spaces` | bool | `true` | Remove spaces. |
| `--carriage-returns-r` | bool | `true` | Remove `\r`. |
| `--line-feeds-n` | bool | `true` | Remove `\n`. |
| `--tabs` | bool | `true` | Remove tabs. |
| `--form-feeds-f` | bool | `true` | Remove `\f`. |
| `--full-stops` | bool | `false` | Remove `.`. |

**Simple example**

```bash
printf 'a b\tc' | cchef remove-whitespace
```

Output:

```
abc
```

## Reverse

Reverses the input by byte, character (UTF-8 rune), or line.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--by` | option | `Character` | `Byte` (raw bytes), `Character` (UTF-8 runes, keeps multi-byte sequences intact), or `Line` (reverses line order). |

**Simple example**

```bash
cchef reverse -i 'Hello, World!'
```

Output:

```
!dlroW ,olleH
```

**Reverse line order**

```bash
printf 'one\ntwo\nthree' | cchef reverse --by Line
```

Output:

```
three
two
one
```

## Show on map

Parses co-ordinates (in any supported format) and returns the `latitude,longitude`
pair. In CyberChef the GUI renders an interactive map; here the operation returns
the parsed pair that the map would use.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--zoom-level` | number | `13` | Map zoom level (used by the GUI only). |
| `--input-format` | option | `Auto` | Input co-ordinate format (`Auto` detects it). |
| `--input-delimiter` | option | `Auto` | How the latitude/longitude are separated. |

**Simple example**

```bash
cchef show-on-map -i '51.5074, -0.1278'
```

Output:

```
51.5074,-0.1278
```

## Shuffle

Randomly reorders the sections of the input, split on a delimiter (using
cryptographically-secure randomness).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |

> Output order is non-deterministic by design.

**Simple example**

```bash
printf 'a\nb\nc\nd' | cchef shuffle
```

Output:

```
c
a
d
b
```

## Sleep

Pauses for the given number of milliseconds, then passes the input through
unchanged.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--time-ms` | number | `1000` | Time to sleep, in milliseconds. |

**Simple example**

```bash
cchef sleep -i 'hello' --time-ms 0
```

Output:

```
hello
```

## Sort

Sorts the sections of the input, split on a delimiter, using the chosen ordering.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |
| `--reverse` | bool | `false` | Reverse the sort order. |
| `--order` | option | `Alphabetical (case sensitive)` | `Alphabetical (case sensitive)`, `Alphabetical (case insensitive)`, `IP address`, `Numeric`, `Numeric (hexadecimal)`, `Length`. |

**Numeric sort**

```bash
printf '10\n2\n1\n20' | cchef sort --order Numeric
```

Output:

```
1
2
10
20
```

## Split

Splits the input on one delimiter and rejoins the parts with another. Delimiters
are used **literally** (matching CyberChef) — escape sequences are not expanded,
so `\n` joins with a literal backslash-n, not a newline.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--split-delimiter` | string | `,` | Delimiter to split on. |
| `--join-delimiter` | string | `\n` | Delimiter to join with (literal). |

**Simple example**

```bash
cchef split --split-delimiter ',' --join-delimiter ';' -i 'a,b,c'
```

Output:

```
a;b;c
```

## Swap case

Converts uppercase characters to lowercase and vice versa. Takes no options.

**Simple example**

```bash
cchef swap-case -i 'Hello, World!'
```

Output:

```
hELLO, wORLD!
```

## Tail

Keeps only the last N sections (lines) of the input. A negative N drops the
first |N| sections.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |
| `--number` | number | `10` | Number of sections to keep. |

**Simple example**

```bash
printf 'a\nb\nc\nd\ne' | cchef tail --number 2
```

Output:

```
d
e
```

## Take bytes

Keeps only a range of bytes from the input.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--start` | number | `0` | Start offset (negative counts from the end). |
| `--length` | number | `5` | Number of bytes to keep. |
| `--apply-to-each-line` | bool | `false` | Apply per line. |

**Simple example**

```bash
cchef take-bytes --start 0 --length 5 -i 'Hello World'
```

Output:

```
Hello
```

## Take nth bytes

Keeps every nth byte, starting at a given offset.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--take-every` | number | `4` | Keep every nth byte. |
| `--starting-at` | number | `0` | Offset to start from. |
| `--apply-to-each-line` | bool | `false` | Reset the count per line. |

**Simple example**

```bash
cchef take-nth-bytes --take-every 2 -i '0123456789'
```

Output:

```
02468
```

## To Case Insensitive Regex

Converts a case-sensitive regular expression into a case-insensitive one, e.g.
`Mozilla` becomes `[mM][oO][zZ][iI][lL][lL][aA]`, expanding character ranges as
needed. Takes no options.

**Simple example**

```bash
cchef to-case-insensitive-regex -i 'Mozilla'
```

Output:

```
[mM][oO][zZ][iI][lL][lL][aA]
```

**Character range**

```bash
cchef to-case-insensitive-regex -i '[A-Z]'
```

Output:

```
[A-Za-z]
```

## To Lower case

Converts every character in the input to lower case. This operation takes no
options.

**Simple example**

```bash
cchef to-lower-case -i 'Hello, World!'
```

Output:

```
hello, world!
```

## To Table

Renders delimited data (e.g. CSV) as an ASCII, HTML, or Markdown table.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--cell-delimiters` | string | `,` | Characters that separate cells. |
| `--row-delimiters` | string | `\r\n` | Characters that separate rows. |
| `--make-first-row-header` | bool | `false` | Treat the first row as a header. |
| `--format` | option | `ASCII` | `ASCII`, `HTML`, or `Markdown`. |

**Simple example**

```bash
printf 'a,b\n1,2' | cchef to-table --format Markdown --make-first-row-header
```

Output:

```
| a | b |
| - | - |
| 1 | 2 |
```

## To Upper case

Converts the input to upper case, optionally limiting the scope to the first
character of each word, sentence, or paragraph.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--scope` | option | `All` | `All`, `Word`, `Sentence`, or `Paragraph`. |

**Simple example**

```bash
cchef to-upper-case -i 'Hello, World!'
```

Output:

```
HELLO, WORLD!
```

**Title-case each word**

```bash
cchef to-upper-case --scope Word -i 'hello there world'
```

Output:

```
Hello There World
```

## Unescape string

Unescapes backslash escape sequences (`\n`, `\t`, `\xNN`, `\uNNNN`, `\u{...}`,
octal, etc.) into their raw characters. Takes no options.

**Simple example**

```bash
printf 'a\\x41b\\n' | cchef unescape-string
```

Output:

```
aAb
```

## Unique

Removes duplicate sections of the input, split on a delimiter, optionally showing
each section's occurrence count.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`, `Colon`, `Nothing (separate chars)`. |
| `--display-count` | bool | `false` | Prefix each section with its count. |

**Simple example**

```bash
printf 'a\nb\na\nc' | cchef unique
```

Output:

```
a
b
c
```

## Wrap

Wraps text to a fixed line width, breaking it into lines of at most that many
characters.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--line-width` | number | `64` | Maximum characters per line. |

**Simple example**

```bash
cchef wrap -i 'The quick brown fox' --line-width 10
```

Output:

```
The quick 
brown fox
```
