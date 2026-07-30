# Forensics

Operations for examining unknown or binary data. Some also belong to another
category, where their detailed description, options and examples live:
[Extract EXIF](multimedia.md#extract-exif) and
[Remove EXIF](multimedia.md#remove-exif) are documented under
[Multimedia](multimedia.md), and
[Extract Files](extractors.md#extract-files) and
[Extract Audio Metadata](extractors.md#extract-audio-metadata) under
[Extractors](extractors.md).

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Detect File Type | `detect-file-type` | [List of file signatures](https://wikipedia.org/wiki/List_of_file_signatures) |
| Extract Audio Metadata | `extract-audio-metadata` | [Extractors](extractors.md#extract-audio-metadata) |
| Extract EXIF | `extract-exif` | [Multimedia](multimedia.md#extract-exif) |
| Extract Files | `extract-files` | [Extractors](extractors.md#extract-files) |
| Remove EXIF | `remove-exif` | [Multimedia](multimedia.md#remove-exif) |
| YARA Rules | `yara-rules` | [YARA](https://wikipedia.org/wiki/YARA) |

## Detect File Type

Guesses the MIME type and extension of the input by matching its leading bytes
against a table of known **magic-byte signatures**. This is a from-scratch port
of CyberChef's operation and its `FileSignatures` table, covering 141 file types
across seven categories (Images, Video, Audio, Documents, Applications, Archives,
Miscellaneous). Every matching type is reported; if nothing matches, an
"Unknown file type" message is returned.

The input is treated as raw bytes, so it pairs naturally with the global
`--in-file` flag (`cchef detect-file-type --in-file mystery.bin`) or with a
preceding decode step such as `from-hex`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Images | boolean | true | Include image signatures in the search. |
| Video | boolean | true | Include video signatures. |
| Audio | boolean | true | Include audio signatures. |
| Documents | boolean | true | Include document signatures. |
| Applications | boolean | true | Include application/executable signatures. |
| Archives | boolean | true | Include archive/compression signatures. |
| Miscellaneous | boolean | true | Include miscellaneous signatures. |

Each category flag can be disabled (e.g. `--images=false`) to narrow the search.

### Simple example

```bash
cchef from-hex -i "89504e470d0a1a0a0000000d" | cchef detect-file-type
```

Output:

```
File type:   Portable Network Graphics image
Extension:   png
MIME type:   image/png
```

### Complex example

Disabling a category removes its signatures from the search, so a PNG is no
longer recognised once images are excluded:

```bash
cchef from-hex -i "89504e470d0a1a0a" | cchef detect-file-type --images=false
```

Output:

```
Unknown file type. Have you tried checking the entropy of this data to determine whether it might be encrypted or compressed?
```

## YARA Rules

Matches the input against a set of [YARA](https://wikipedia.org/wiki/YARA)
rules — the language malware researchers write to describe and classify
samples. A rule names the strings to look for and a condition saying which
combination of them counts.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Rules | string | (empty) | The rules themselves. |
| Show strings | boolean | `false` | Print the bytes each match found. |
| Show string lengths | boolean | `false` | Print how long each match was. |
| Show metadata | boolean | `false` | Print each rule's `meta` section. |
| Show counts | boolean | `true` | Print how many times a rule's strings were found. |
| Show rule warnings | boolean | `true` | Print warnings about rules that will scan slowly. |
| Show console module messages | boolean | `true` | Print what the `console` module was asked to say. |

The engine is written from scratch in Go — there is no usable pure-Go YARA
library, and the alternatives all need a C or Rust library linked in, which
would cost the single self-contained binary.

**What it covers.** All three kinds of string (plain text, hex patterns with
wildcards, jumps and alternatives, and regular expressions) with the `nocase`,
`wide`, `ascii`, `fullword`, `xor`, `base64` and `base64wide` modifiers; the
whole condition language, including counts, offsets, lengths, `at`, `in`, every
form of `of`, both kinds of `for` loop, arithmetic and the integer readers; rule
metadata, tags, global and private rules, and references between rules; and
**every module CyberChef's own build has**: `hash`, `math`, `console`, `time`,
`elf`, `pe` and `dotnet`. The `pe` module covers the headers, sections and data
directories, imports, exports and `imphash`, resources and version information,
the debug path, the rich signature, and the certificates that signed the file.
The `dotnet` module covers the metadata streams, the assembly and what it leans
on, the resources, the fixed text, and the numbers naming the build.

**What it does not.** The remaining YARA modules — `macho`, `dex`, `magic`,
`cuckoo` and `lnk` — are absent from CyberChef's own build too, so naming one is
refused here as it is there, rather than left to quietly match nothing.

One narrow limit: in a pattern asked for `wide`, the edge of a word falls between
whole wide characters, and `\b` is worked out accordingly. Written at either end
of the pattern, or between two plain characters, it is answered exactly. Written
between things that might each match a letter or not — `hel\b[a-z]o` — it cannot
be settled either way, and is refused rather than answered wrongly.

A string only carries the words its kind allows, as in YARA itself: a hex
pattern takes `private` and nothing else, and a regular expression will not take
`xor` or `base64`. Anything else is refused as it is read.

### Simple example

```bash
cchef yara-rules -i "hello world" --rules 'rule Greeting { strings: $a = "hello" condition: $a }'
```

Output:

```
Input matches rule "Greeting"  (1 time).
```

The doubled space before the count is CyberChef's, kept so that the output
matches it character for character.

### Complex example

Several strings, metadata, and every detail turned on:

```bash
cchef yara-rules -i "hello world" --rules 'rule Greeting { meta: author = "me" strings: $a = "hello" $b = "world" condition: all of them }' --show-strings --show-string-lengths --show-metadata
```

Output:

```
Rule "Greeting" [author: me] matches (2 times):
Pos 0, length 5, identifier $a, data: "hello"
Pos 6, length 5, identifier $b, data: "world"
```

A hex pattern and a regular expression together, either of which will do:

```bash
cchef yara-rules -i "The quick brown fox" --rules 'rule Fox { strings: $a = /f[oa]x/ nocase $b = { 71 75 69 63 6b } condition: any of them }' --show-strings
```

Output:

```
Rule "Fox" matches (2 times):
Pos 16, identifier $a, data: "fox"
Pos 4, identifier $b, data: "quick"
```

The `pe` module picks a Windows executable apart, so a rule can ask about its
headers, what it borrows and lends, and who signed it:

```bash
cchef yara-rules --in-file signed.exe --rules 'import "pe" rule Signed_By_DigiCert { condition: pe.is_pe and pe.number_of_signatures > 0 and for any i in (0..pe.number_of_signatures - 1) : ( pe.signatures[i].issuer contains "DigiCert" ) }'
```

Output:

```
Input matches rule "Signed_By_DigiCert".
```

The `console` module reports what a rule found while it ran, which is a quick
way to read fields out of a file. Bytes that cannot be printed are written as
their value, as YARA itself writes them:

```bash
cchef yara-rules --in-file signed.exe --rules 'import "pe" import "console" rule Report { condition: pe.is_pe and console.log("sections: ", pe.number_of_sections) and console.log("signed by: ", pe.signatures[0].subject) }'
```

Output:

```
sections: 3
signed by: /businessCategory=Private Organization/jurisdictionC=DE/jurisdictionST=Hamburg/serialNumber=HRA 115662/street=Gerritstra\xC3\x9Fe 14/postalCode=22767/C=DE/L=Hamburg/O=AKApplications e.K./CN=AKApplications e.K.
Input matches rule "Report".
```

The `dotnet` module reads the metadata a file built for the common language
runtime carries:

```bash
cchef yara-rules --in-file managed.dll --rules 'import "dotnet" import "console" rule DotnetReport { condition: dotnet.is_dotnet and console.log("assembly: ", dotnet.assembly.name) and console.log("built against: ", dotnet.version) and console.log("leans on: ", dotnet.assembly_refs[0].name) }'
```

Output:

```
assembly: AcWindows
built against: v2.0.50727
leans on: mscorlib
Input matches rule "DotnetReport".
```
