# Utils

General-purpose text utilities.

> Operations are listed alphabetically.

| Operation | Subcommand |
| --- | --- |
| Filter | `filter` |
| Find / Replace | `find-replace` |
| Pad lines | `pad-lines` |
| Remove null bytes | `remove-null-bytes` |
| Remove whitespace | `remove-whitespace` |
| Reverse | `reverse` |
| Sort | `sort` |
| Swap case | `swap-case` |
| To Lower case | `to-lower-case` |
| To Upper case | `to-upper-case` |
| Unique | `unique` |

---

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

## Swap case

Converts uppercase characters to lowercase and vice versa. Takes no options.

**Simple example**

```bash
$ cchef swap-case -i 'Hello, World!'
hELLO, wORLD!
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
