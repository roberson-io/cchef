# Utils

General-purpose text utilities.

> Operations are listed alphabetically.

| Operation | Subcommand |
| --- | --- |
| Add line numbers | `add-line-numbers` |
| Alternating Caps | `alternating-caps` |
| Convert area | `convert-area` |
| Convert data units | `convert-data-units` |
| Convert distance | `convert-distance` |
| Convert mass | `convert-mass` |
| Convert speed | `convert-speed` |
| Count occurrences | `count-occurrences` |
| Drop bytes | `drop-bytes` |
| Drop nth bytes | `drop-nth-bytes` |
| Escape string | `escape-string` |
| Expand alphabet range | `expand-alphabet-range` |
| Filter | `filter` |
| Find / Replace | `find-replace` |
| From Case Insensitive Regex | `from-case-insensitive-regex` |
| Get All Casings | `get-all-casings` |
| Hamming Distance | `hamming-distance` |
| Head | `head` |
| Levenshtein Distance | `levenshtein-distance` |
| Pad lines | `pad-lines` |
| Parse UNIX file permissions | `parse-unix-file-permissions` |
| Remove ANSI Escape Codes | `remove-ansi-escape-codes` |
| Remove line numbers | `remove-line-numbers` |
| Remove null bytes | `remove-null-bytes` |
| Remove whitespace | `remove-whitespace` |
| Reverse | `reverse` |
| Sort | `sort` |
| Split | `split` |
| Swap case | `swap-case` |
| Tail | `tail` |
| Take bytes | `take-bytes` |
| Take nth bytes | `take-nth-bytes` |
| To Case Insensitive Regex | `to-case-insensitive-regex` |
| To Lower case | `to-lower-case` |
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
$ printf 'a\nb\nc' | cchef add-line-numbers
1 a
2 b
3 c
```

## Alternating Caps

Applies aLtErNaTiNg capitalisation, starting with lower case (non-letters are
left unchanged). Takes no options.

**Simple example**

```bash
$ cchef alternating-caps -i 'hello world'
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
$ cchef convert-area --input-units 'Square foot (sq ft)' --output-units 'Square metre (sq m)' -i 100
9.290304
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
$ cchef convert-data-units --input-units 'Gibibytes (GiB)' --output-units 'Mebibytes (MiB)' -i 1
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
$ cchef convert-distance --input-units 'Miles (mi)' --output-units 'Kilometers (km)' -i 1
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
$ cchef convert-mass --input-units 'Pound (lb)' --output-units 'Gram (g)' -i 1
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
$ cchef convert-speed --input-units 'Kilometres per hour (km/h)' --output-units 'Metres per second (m/s)' -i 100
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
$ cchef count-occurrences --search-string foo --search-string-type 'Simple string' -i 'foofoofoo'
3
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
$ cchef drop-bytes --start 0 --length 6 -i 'Hello World'
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
$ cchef drop-nth-bytes --drop-every 2 -i '0123456789'
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
$ cchef escape-string -i "it's a café"
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
$ cchef expand-alphabet-range -i 'a-j'
abcdefghij
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
$ printf 'apple\nbanana\ncherry' | cchef filter --regex a
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
$ cchef find-replace --find 'foo' --find-type 'Simple string' --replace 'bar' -i 'foofoo'
barbar
```

**Regex with capture groups**

```bash
$ cchef find-replace --find '(\w+) (\w+)' --find-type Regex --replace '$2 $1' -i 'John Smith'
Smith John
```

## From Case Insensitive Regex

Converts a case-insensitive regex of the form `[aA][bB]` back to a case-sensitive
one (`ab`). Character classes with distinct letters are left unchanged. Takes no
options.

**Simple example**

```bash
$ cchef from-case-insensitive-regex -i '[tT][eE][sS][tT]'
test
```

## Get All Casings

Outputs every combination of upper- and lower-case for the input, one per line.
Takes no options.

> The number of results doubles with each character, so keep the input short.

**Simple example**

```bash
$ cchef get-all-casings -i ab
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
$ printf 'karolin\n\nkathrin' | cchef hamming-distance --unit Bit
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
$ printf 'a\nb\nc\nd\ne' | cchef head --number 2
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
$ printf 'kitten\nsitting' | cchef levenshtein-distance
3
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
$ printf 'ab\ncd' | cchef pad-lines --position Start --length 2 --character '*'
**ab
**cd
```

## Parse UNIX file permissions

Parses a UNIX file permission string — octal (e.g. `755`) or textual (e.g.
`drwxr-xr-x`) — and prints the textual/octal representations, any special flags,
and a read/write/execute matrix. Takes no options.

**Simple example**

```bash
$ cchef parse-unix-file-permissions -i 755
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

## Remove ANSI Escape Codes

Removes ANSI escape codes (e.g. terminal colour codes) from the input. Takes no
options.

**Simple example**

```bash
$ printf '\x1b[31mred\x1b[0m text' | cchef remove-ansi-escape-codes
red text
```

## Remove line numbers

Removes line numbers from the beginning of each line, where they can be found.
Takes no options.

**Simple example**

```bash
$ printf '1 a\n2 b\n3 c' | cchef remove-line-numbers
a
b
c
```

## Remove null bytes

Removes all null bytes (`0x00`) from the input. Takes no options.

**Simple example**

```bash
$ printf 'a\x00b\x00c' | cchef remove-null-bytes
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
$ printf 'a b\tc' | cchef remove-whitespace
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
$ cchef reverse -i 'Hello, World!'
!dlroW ,olleH
```

**Reverse line order**

```bash
$ printf 'one\ntwo\nthree' | cchef reverse --by Line
three
two
one
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
$ printf '10\n2\n1\n20' | cchef sort --order Numeric
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
$ cchef split --split-delimiter ',' --join-delimiter ';' -i 'a,b,c'
a;b;c
```

## Swap case

Converts uppercase characters to lowercase and vice versa. Takes no options.

**Simple example**

```bash
$ cchef swap-case -i 'Hello, World!'
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
$ printf 'a\nb\nc\nd\ne' | cchef tail --number 2
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
$ cchef take-bytes --start 0 --length 5 -i 'Hello World'
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
$ cchef take-nth-bytes --take-every 2 -i '0123456789'
02468
```

## To Case Insensitive Regex

Converts a case-sensitive regular expression into a case-insensitive one, e.g.
`Mozilla` becomes `[mM][oO][zZ][iI][lL][lL][aA]`, expanding character ranges as
needed. Takes no options.

**Simple example**

```bash
$ cchef to-case-insensitive-regex -i 'Mozilla'
[mM][oO][zZ][iI][lL][lL][aA]
```

**Character range**

```bash
$ cchef to-case-insensitive-regex -i '[A-Z]'
[A-Za-z]
```

## To Lower case

Converts every character in the input to lower case. This operation takes no
options.

**Simple example**

```bash
$ cchef to-lower-case -i 'Hello, World!'
hello, world!
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
$ cchef to-upper-case -i 'Hello, World!'
HELLO, WORLD!
```

**Title-case each word**

```bash
$ cchef to-upper-case --scope Word -i 'hello there world'
Hello There World
```

## Unescape string

Unescapes backslash escape sequences (`\n`, `\t`, `\xNN`, `\uNNNN`, `\u{...}`,
octal, etc.) into their raw characters. Takes no options.

**Simple example**

```bash
$ printf 'a\\x41b\\n' | cchef unescape-string
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
$ printf 'a\nb\na\nc' | cchef unique
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
$ printf 'The quick brown fox' | cchef wrap --line-width 10
The quick 
brown fox
```
