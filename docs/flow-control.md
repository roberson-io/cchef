# Flow control

Operations that steer a recipe rather than transform data: splitting it into
branches, jumping between steps, and carrying values from one step into
another's arguments.

These only do anything **inside a recipe** — `cchef bake`, a recipe file, or a
recipe URL — because there is nothing to steer when an operation runs on its
own. Each degrades to passing the data through when invoked as a standalone
subcommand, except `fork`, which with no steps to run over the pieces simply
replaces the split delimiter with the merge delimiter.

Patterns are Go's [RE2](https://github.com/google/re2/wiki/Syntax) syntax, as in
[Regular expression](utils.md#regular-expression) and
[Find / Replace](utils.md#find--replace); lookahead and backreferences are not
available.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Comment | `comment` | — |
| Conditional Jump | `conditional-jump` | [Regular expression syntax](https://github.com/google/re2/wiki/Syntax) |
| Fork | `fork` | — |
| Jump | `jump` | — |
| Label | `label` | — |
| Magic | `magic` | [Automatic detection of encoded data](https://github.com/gchq/CyberChef/wiki/Automatic-detection-of-encoded-data-using-CyberChef-Magic) |
| Merge | `merge` | — |
| Register | `register` | [Regular expression syntax](https://github.com/google/re2/wiki/Syntax) |
| Return | `return` | — |
| Subsection | `subsection` | [Regular expression syntax](https://github.com/google/re2/wiki/Syntax) |

## Comment

A place to write a note in a recipe. Has no effect on the data.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| (text) | string | (empty) | The note. |

## Conditional Jump

Jumps to a [Label](#label) when the data matches a pattern — forwards to skip
steps, or backwards to repeat them. A backwards jump is bounded by the maximum,
counted across the whole recipe, so a loop always ends. An empty pattern never
jumps.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Match (regex) | string | (empty) | The test. Empty means never jump. |
| Invert match | boolean | false | Jump when the data does **not** match. |
| Label name | string | (empty) | Where to jump to. A label that does not exist means no jump. |
| Maximum jumps (if jumping backwards) | number | 10 | How many jumps the recipe may take in total. |

### Simple example

Base64-encode numbers and uppercase everything else, in one recipe:

```bash
cchef bake -e '[{"op":"Conditional Jump","args":["^\\d+$",false,"num",10]},{"op":"To Upper case","args":["All"]},{"op":"Return","args":[]},{"op":"Label","args":["num"]},{"op":"To Base64","args":["A-Za-z0-9+/="]}]' -i "42"
```

Output:

```
NDI=
```

The same recipe on text takes the other path:

```bash
cchef bake -e '[{"op":"Conditional Jump","args":["^\\d+$",false,"num",10]},{"op":"To Upper case","args":["All"]},{"op":"Return","args":[]},{"op":"Label","args":["num"]},{"op":"To Base64","args":["A-Za-z0-9+/="]}]' -i "hello"
```

Output:

```
HELLO
```

## Fork

Splits the data on a delimiter and runs every following step over each piece
separately, then rejoins the results. A [Merge](#merge) closes the branch; with
no Merge, the rest of the recipe runs per piece.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Split delimiter | string | `\n` | Escape sequences are written as text (`\n`, `\t`). |
| Merge delimiter | string | `\n` | What the results are joined with. |
| Ignore errors | boolean | false | Leave a failing piece as it was instead of stopping. |

### Simple example

Base64-encode each line on its own:

```bash
cchef bake -e '[{"op":"Fork","args":["\\n","\\n",false]},{"op":"To Base64","args":["A-Za-z0-9+/="]}]' -i "$(printf 'cat\nsat\nmat')"
```

Output:

```
Y2F0
c2F0
bWF0
```

## Jump

Jumps to a [Label](#label) unconditionally.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Label name | string | (empty) | Where to jump to. A label that does not exist means no jump. |
| Maximum jumps (if jumping backwards) | number | 10 | How many jumps the recipe may take in total. |

## Label

Marks a place a jump can reach. Has no effect on the data.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Name | string | (empty) | The name jumps refer to. |

## Magic

Works out what data might be and suggests recipes to make sense of it. Every
operation that claims it can decode data of this shape is run, the result is
looked at the same way, and so on to the given depth; the recipes that led
somewhere are reported best first.

Each suggestion is judged on what its result became: whether it reads as a
known language, whether it is a recognisable file, whether it is valid UTF-8,
how much entropy it carries, and how few operations it took. Suggested recipes
are printed one operation per line, in the form
[`cchef bake -e`](recipes-and-urls.md) accepts, so a promising one can be run
by copying it.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Depth | number | 3 | How many operations may be chained. |
| Intensive mode | boolean | false | Also brute-force every single-byte exclusive-or, bit rotation and character encoding, over the first 100 bytes. |
| Extensive language support | boolean | false | Weigh the data against 285 languages rather than the 39 most used on the Internet. |
| Crib (known plaintext string or regex) | string | (empty) | Keep only the suggestions whose result contains this. |

Results are ranked, so piping through `head` shows the most promising few.

### Simple example

```bash
cchef magic -i "41 42 43 44 45"
```

Output:

```
Recipe:
From_Hex('Space')
  Data:     ABCDE
  Matching ops: From Hexdump
  Valid UTF8
  Entropy: 2.32

Recipe:
From_Decimal('Space',false)
  Data:     )*+,-
  Valid UTF8
  Entropy: 2.32
```

### Complex example

Three rounds of Base64, with a crib to keep only the suggestions that reach the
expected text:

```bash
cchef magic -i "WkVkV2VtUkRRbnBrU0Vwd1ltMWpQUT09" --crib "test string" | head -8
```

Output:

```
Recipe:
From_Base64('A-Za-z0-9+/=',true,false)
From_Base64('A-Za-z0-9+/=',true,false)
From_Base64('A-Za-z0-9+/=',true,false)
  Data:     test string
  Possible languages: Norwegian (Bokmål), Norwegian (Nynorsk), Danish, English, Swedish, German, Estonian, Hungarian, Dutch, Italian, French, Catalan, Indonesian, Latvian, Lithuanian, Portuguese, Spanish, Croatian, Slovenian, Romanian, Polish, Finnish, Turkish
  Valid UTF8
  Entropy: 2.85
```

The alphabets that follow in the full output are the other Base64 variants that
decode the same data, which CyberChef lists too.

## Merge

Closes a [Fork](#fork) or [Subsection](#subsection), bringing the branches back
into one. Has no effect on the data itself.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Merge All | boolean | true | Close every open branch. Untick to close only the nearest one. |

## Register

Captures parts of the data with a regular expression and makes them available
to later steps as `$R0`, `$R1` and so on, numbered by capture group. The data
passes through unchanged. A reference written `\$R0` is left as literal text.

A second Register carries on numbering where the first left off. Inside a
[Fork](#fork), each branch starts again from the arguments as written, so a
register captured in one branch does not leak into the next.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Extractor | string | `([\s\S]*)` | Capture groups select what is stored. |
| Case insensitive | boolean | true | |
| Multiline matching | boolean | false | `^` and `$` match at line breaks. |
| Dot matches all | boolean | false | `.` also matches a line break. |

### Simple example

Lift a key out of the data and use it as another step's argument:

```bash
cchef bake -e '[{"op":"Register","args":["k=(\\w+)",true,false,false]},{"op":"Fork","args":[";","$R0/",false]}]' -i "k=KEY;a;b"
```

Output:

```
k=KEYKEY/aKEY/b
```

## Return

Ends the recipe at this point; nothing below it runs.

Takes no options.

## Subsection

Runs every following step over only the parts of the data that match a pattern,
leaving everything else untouched. With one capture group, only the group is
worked on. A [Merge](#merge) closes the section.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Section (regex) | string | (empty) | Which parts to work on. Empty means none. |
| Case sensitive matching | boolean | true | |
| Global matching | boolean | true | Untick to work on the first match only. |
| Ignore errors | boolean | false | Leave a failing section as it was instead of stopping. |

### Simple example

Uppercase the letters and leave the digits alone:

```bash
cchef bake -e '[{"op":"Subsection","args":["[a-z]+",true,true,false]},{"op":"To Upper case","args":["All"]}]' -i "1abc2def3"
```

Output:

```
1ABC2DEF3
```
