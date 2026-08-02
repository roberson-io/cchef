# Language

Operations for working with human-language text. Some are shared with
[Data format](data-format.md), where their detailed descriptions, options and
examples live: [Decode text](data-format.md#decode-text),
[Encode text](data-format.md#encode-text) and
[Unescape Unicode Characters](data-format.md#unescape-unicode-characters).

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Convert Leet Speak | `convert-leet-speak` | [Leet](https://wikipedia.org/wiki/Leet) |
| Convert to NATO alphabet | `convert-to-nato-alphabet` | [NATO phonetic alphabet](https://wikipedia.org/wiki/NATO_phonetic_alphabet) |
| Decode text | `decode-text` | [Data format](data-format.md#decode-text) |
| Encode text | `encode-text` | [Data format](data-format.md#encode-text) |
| Remove Diacritics | `remove-diacritics` | [Diacritic](https://wikipedia.org/wiki/Diacritic) |
| Unescape Unicode Characters | `unescape-unicode-characters` | [Data format](data-format.md#unescape-unicode-characters) |
| Unicode Text Format | `unicode-text-format` | [Combining character](https://wikipedia.org/wiki/Combining_character) |

## Convert Leet Speak

Converts to and from [Leet Speak](https://wikipedia.org/wiki/Leet). Six
letters trade places with digits — `a e i o s t` against `4 3 1 0 5 7` — and
everything else passes through. Going to leet either case of the letter
becomes the digit; coming back the digit becomes the lowercase letter, since a
digit carries no case to restore.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--direction` | option | `To Leet Speak` | Or `From Leet Speak`. |

### Simple example

```bash
cchef convert-leet-speak -i "leet speak is cool"
```

Output:

```
l337 5p34k 15 c00l
```

### Complex example

Coming back, letters keep their case but recovered ones are lowercase:

```bash
cchef convert-leet-speak --direction "From Leet Speak" -i "H3LL0 W0RLD"
```

Output:

```
HeLLo WoRLD
```

## Convert to NATO alphabet

Spells letters, digits and the punctuation `, / .` out in the
[NATO phonetic alphabet](https://wikipedia.org/wiki/NATO_phonetic_alphabet).
Each spelled character carries its own trailing space; anything else — other
punctuation, spaces, accented text — passes through unchanged. The operation
takes no options.

### Simple example

```bash
cchef convert-to-nato-alphabet -i "Go 4 it."
```

Output:

```
Golf Oscar  Four  India Tango Full stop 
```

(The doubled spaces are the input's own spaces following a spelled
character's trailing space.)

## Remove Diacritics

Replaces accented characters with their plain equivalents by decomposing to
[Unicode NFD](https://wikipedia.org/wiki/Unicode_equivalence) and dropping the
combining diacritical marks, so Unicode text formatting such as strikethroughs
and underlines is removed too. A character whose accent is part of the letter
itself rather than a combining mark (`ø`, `đ`, `ß`) has nothing to strip and
survives. The operation takes no options.

### Simple example

```bash
cchef remove-diacritics -i "naïve façade Zürich"
```

Output:

```
naive facade Zurich
```

## Unicode Text Format

Formats plaintext by adding Unicode
[combining characters](https://wikipedia.org/wiki/Combining_character): U+0336
for strikethrough and U+0332 for underline, after every character, so the text
carries its formatting anywhere Unicode is rendered.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `--underline` | boolean | false | Add a combining low line to each character. |
| `--strikethrough` | boolean | false | Add a combining long stroke to each character. |

One departure from CyberChef, fixing a fault in its version (also logged
upstream): it inserts the combining characters after every *byte*, splitting
multi-byte characters into invalid UTF-8 and destroying any text beyond
ASCII. cchef appends them after each whole character. ASCII input matches
CyberChef byte for byte.

### Simple example

```bash
cchef unicode-text-format -i "hello" --underline
```

Output:

```
h̲e̲l̲l̲o̲
```

### Complex example

The combining characters are ordinary bytes in the output, visible under a
hex view — each `cc b6` is the strikethrough mark following a character:

```bash
cchef unicode-text-format -i "hi" --strikethrough | cchef to-hex
```

Output:

```
68 cc b6 69 cc b6
```
