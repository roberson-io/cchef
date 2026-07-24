# Forensics

Operations for examining unknown or binary data. Some also belong to another
category, where their detailed description, options and examples live:
[Extract EXIF](multimedia.md#extract-exif) and
[Remove EXIF](multimedia.md#remove-exif) are documented under
[Multimedia](multimedia.md).

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Detect File Type | `detect-file-type` | [List of file signatures](https://wikipedia.org/wiki/List_of_file_signatures) |
| Extract EXIF | `extract-exif` | [Multimedia](multimedia.md#extract-exif) |
| Remove EXIF | `remove-exif` | [Multimedia](multimedia.md#remove-exif) |

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
