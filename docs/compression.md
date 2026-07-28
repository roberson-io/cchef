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
| Gunzip | `gunzip` | [gzip](https://wikipedia.org/wiki/Gzip) |
| Gzip | `gzip` | [gzip](https://wikipedia.org/wiki/Gzip) |
| Raw Deflate | `raw-deflate` | [DEFLATE](https://wikipedia.org/wiki/DEFLATE) |
| Raw Inflate | `raw-inflate` | [DEFLATE](https://wikipedia.org/wiki/DEFLATE) |
| Zlib Deflate | `zlib-deflate` | [zlib](https://wikipedia.org/wiki/Zlib) |
| Zlib Inflate | `zlib-inflate` | [zlib](https://wikipedia.org/wiki/Zlib) |

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

## Gunzip

Reads a [gzip](https://wikipedia.org/wiki/Gzip) stream back into the bytes it
was made from. It takes no options — everything it needs is in the stream.

A file holding several gzip streams one after another is read whole and the
results joined, which is what the `gzip` command does.

### Simple example

```bash
cchef gzip -i "hello hello hello" | cchef gunzip
```

Output:

```
hello hello hello
```

### Complex example

Reading a file written by `gzip` itself:

```bash
gzip -c notes.txt | cchef gunzip
```

## Gzip

Compresses the input into a [gzip](https://wikipedia.org/wiki/Gzip) stream: a
DEFLATE stream behind a header that can carry a filename and a comment, and a
trailer holding a checksum and the original length.

The `Compression type` option chooses how each block is encoded:

| Setting | Does |
| --- | --- |
| `Dynamic Huffman Coding` | Works out a code fitted to this input and sends it with the data. Usually the smallest, and the default. |
| `Fixed Huffman Coding` | Uses the code the format defines, which need not be sent. Smaller than dynamic for very short input, where sending a code costs more than it saves. |
| `None (Store)` | Does not compress at all, writing the data behind a short block header. Always slightly larger than the input. |

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression type | option | `Dynamic Huffman Coding` | As above. |
| Filename (optional) | string | (empty) | Recorded in the header. Left out when empty. |
| Comment (optional) | string | (empty) | Likewise. |
| Include file checksum | boolean | `false` | Adds a two-byte checksum of the header, which a reader can use to spot a damaged one. |

**The output is not the same twice.** The header records the time the stream was
written, so compressing the same input a minute later gives four different
bytes. Everything else is fixed.

### Simple example

```bash
cchef gzip -i "hello hello hello" | cchef gunzip
```

Output:

```
hello hello hello
```

### Complex example

The filename and comment go into the header, where `gunzip` and other readers
can find them:

```bash
cchef gzip -i "hello" --filename-optional "greeting.txt" --comment-optional "written by cchef" --include-file-checksum -o hello.gz
cchef gunzip --in-file hello.gz
```

Output:

```
hello
```

## Raw Deflate

Compresses the input into a [DEFLATE](https://wikipedia.org/wiki/DEFLATE)
stream with nothing around it — no header, no checksum, no length. Use
[Gzip](#gzip) or [Zlib Deflate](#zlib-deflate) for a stream other tools will
recognise.

The `Compression type` option chooses how each block is encoded:

| Setting | Does |
| --- | --- |
| `Dynamic Huffman Coding` | Works out a code fitted to this input and sends it with the data. Usually the smallest, and the default. |
| `Fixed Huffman Coding` | Uses the code the format defines, which need not be sent. Smaller than dynamic for very short input, where sending a code costs more than it saves. |
| `None (Store)` | Does not compress at all, writing the data behind a short block header. Always slightly larger than the input. |

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression type | option | `Dynamic Huffman Coding` | As above. |

### Simple example

```bash
cchef raw-deflate -i "The quick brown fox jumped over the slow dog" | cchef to-hex --delimiter None
```

Output:

```
0dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02
```

### Complex example

For short, repetitive input the fixed code wins, because the dynamic one has to
be sent along with the data:

```bash
cchef raw-deflate -i "hello hello hello" --compression-type "Fixed Huffman Coding" | cchef to-hex --delimiter None
```

Output:

```
cb48cdc9c957402201
```

## Raw Inflate

Reads a raw [DEFLATE](https://wikipedia.org/wiki/DEFLATE) stream back into the
bytes it was made from.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Start index | number | `0` | Begin reading this many bytes into the input, for a stream that does not start at the beginning. |
| Initial output buffer size | number | `0` | Accepted and ignored. |
| Buffer expansion type | option | `Adaptive` | Accepted and ignored. |
| Resize buffer after decompression | boolean | `false` | Accepted and ignored. |
| Verify result | boolean | `false` | Accepted and ignored. |

The last four size and grow the working buffer inside CyberChef's reader. They
change how much memory it uses and nothing about what it produces, so cchef
accepts them and pays them no attention.

### Simple example

```bash
cchef raw-deflate -i "hello hello hello" | cchef raw-inflate
```

Output:

```
hello hello hello
```

### Complex example

A DEFLATE stream buried at a known offset — here four bytes in — is read by
saying where it starts:

```bash
cchef raw-inflate --in-file embedded.bin --start-index 4
```

## Zlib Deflate

Compresses the input into a [zlib](https://wikipedia.org/wiki/Zlib) stream: a
DEFLATE stream behind two header bytes, with an Adler-32 checksum of the
original data after it.

The `Compression type` option chooses how each block is encoded:

| Setting | Does |
| --- | --- |
| `Dynamic Huffman Coding` | Works out a code fitted to this input and sends it with the data. Usually the smallest, and the default. |
| `Fixed Huffman Coding` | Uses the code the format defines, which need not be sent. Smaller than dynamic for very short input, where sending a code costs more than it saves. |
| `None (Store)` | Does not compress at all, writing the data behind a short block header. Always slightly larger than the input. |

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression type | option | `Dynamic Huffman Coding` | As above. |

### Simple example

```bash
cchef zlib-deflate -i "The quick brown fox jumped over the slow dog" | cchef to-hex --delimiter None
```

Output:

```
789c0dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b026bcc1035
```

### Complex example

Stored rather than compressed, the two header bytes and the four checksum bytes
are easy to pick out around the data:

```bash
cchef zlib-deflate -i "hi" --compression-type "None (Store)" | cchef to-hex --delimiter None
```

Output:

```
7801010200fdff6869013b00d2
```

## Zlib Inflate

Reads a [zlib](https://wikipedia.org/wiki/Zlib) stream back into the bytes it
was made from, checking the Adler-32 trailer as it goes.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Start index | number | `0` | Begin reading this many bytes into the input. |
| Initial output buffer size | number | `0` | Accepted and ignored. |
| Buffer expansion type | option | `Adaptive` | Accepted and ignored. |
| Resize buffer after decompression | boolean | `false` | Accepted and ignored. |
| Verify result | boolean | `false` | Accepted and ignored. |

As with [Raw Inflate](#raw-inflate), the last four control the reader's working
buffer and have no bearing on the result.

### Simple example

```bash
cchef zlib-deflate -i "hello hello hello" | cchef zlib-inflate
```

Output:

```
hello hello hello
```

### Complex example

A stream whose checksum does not match the data is refused rather than returned:

```bash
cchef zlib-deflate -i "hello" -o hello.zz
```


## A note on the compressed bytes

Every writer here produces the same bytes CyberChef does, which is not something
that comes for free: any number of different DEFLATE streams decode to the same
data, and which one a compressor writes depends on how it looks for repeats and
how it builds its codes. CyberChef uses the zlibjs library, whose choices differ
from those of zlib — the library behind `gzip`, `pigz` and most everything else
— so cchef reproduces zlibjs rather than reaching for the nearest available
compressor. Streams from either are read by any reader.

The one place cchef deliberately differs is **empty input**. CyberChef writes a
stream there that its own reader rejects: under `None (Store)` it returns a
32768-byte working buffer that was never filled in, and under the other two a
stream cut a couple of bits short. cchef writes the shortest well-formed stream
instead, which reads back as empty.

