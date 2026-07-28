# Compression

Operations that shrink data and restore it again.

Compressed output is binary, so the examples below pipe it through
[To Hex](data-format.md#to-hex) to make it readable. In real use, send it to a
file with `-o` instead.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Bzip2 Compress | `bzip2-compress` | [bzip2](https://wikipedia.org/wiki/Bzip2) |
| Bzip2 Decompress | `bzip2-decompress` | [bzip2](https://wikipedia.org/wiki/Bzip2) |

## Bzip2 Compress

Compresses the input into a [bzip2](https://wikipedia.org/wiki/Bzip2) stream —
the format Julian Seward built around the Burrows-Wheeler transform. It
compresses more tightly than Deflate and takes rather longer to do it.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Block size (100s of kb) | number | `9` | How much data one Burrows-Wheeler transform covers, from 1 to 9. Larger blocks compress better; input longer than one block is split across several. The size is recorded in the stream so the reader does not need telling. |
| Work factor | number | `30` | Accepted and ignored. In bzip2 it decides when the block sort abandons its faster path for a slower one; both reach the same answer, so it affects how long compressing takes and nothing about what comes out. |

The output is byte-for-byte what bzip2 itself produces, so `bzip2 -d` and any
other reader will take it.

Empty input is refused with `Please provide an input.`

### Simple example

```bash
cchef bzip2-compress -i "The cat sat on the mat." | cchef to-hex --delimiter None
```

Output:

```
425a6839314159265359b218ed630000031380400104002a438c00200021a68261a840d0342821a65675f0db844263f177245385090b218ed630
```

### Complex example

The block size shows up in the fourth byte of the stream, as the digit itself —
`BZh1` here rather than `BZh9`:

```bash
cchef bzip2-compress -i "The cat sat on the mat." --block-size-100s-of-kb 1 | cchef to-hex --delimiter None
```

Output:

```
425a6831314159265359b218ed630000031380400104002a438c00200021a68261a840d0342821a65675f0db844263f177245385090b218ed630
```

Round-tripping a file:

```bash
cchef bzip2-compress --in-file notes.txt -o notes.txt.bz2
cchef bzip2-decompress --in-file notes.txt.bz2
```

## Bzip2 Decompress

Reads a [bzip2](https://wikipedia.org/wiki/Bzip2) stream back into the bytes it
was made from.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Use low-memory, slower decompression algorithm | boolean | `false` | Accepted and ignored. In bzip2 it picks between two decoders that differ in how much they allocate, not in what they produce. |

Input that is not a bzip2 stream is reported as `not a bzip2 stream`, and one
that stops part way through as `truncated bzip2 stream`. Empty input is refused
with `Please provide an input.`

**Fidelity.** Two deliberate differences from CyberChef, both matching what the
`bzip2` command does:

- Several streams written one after another are **all** read, and their contents
  joined. CyberChef reads only the first and silently discards the rest.
- Anything left over after a stream has ended is ignored rather than being
  treated as the start of another.

CyberChef also reports every kind of bad input as just `Error`; the messages
above say which kind it was.

### Simple example

```bash
cchef from-hex -i "425a6839314159265359b218ed630000031380400104002a438c00200021a68261a840d0342821a65675f0db844263f177245385090b218ed630" | cchef bzip2-decompress
```

Output:

```
The cat sat on the mat.
```

### Complex example

Streams written one after another are all read:

```bash
cchef bzip2-compress -i "one " -o /tmp/a.bz2 && cchef bzip2-compress -i "two" -o /tmp/b.bz2 && cat /tmp/a.bz2 /tmp/b.bz2 | cchef bzip2-decompress
```

Output:

```
one two
```
