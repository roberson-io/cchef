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
| LZ4 Compress | `lz4-compress` | [LZ4](https://wikipedia.org/wiki/LZ4_(compression_algorithm)) |
| LZ4 Decompress | `lz4-decompress` | [LZ4](https://wikipedia.org/wiki/LZ4_(compression_algorithm)) |
| LZMA Compress | `lzma-compress` | [LZMA](https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm) |
| LZMA Decompress | `lzma-decompress` | [LZMA](https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm) |
| LZNT1 Decompress | `lznt1-decompress` | [MS-XCA](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xca/5655f4a3-6ba4-489b-959f-e1f407c52f15) |
| LZString Compress | `lzstring-compress` | [lz-string](https://pieroxy.net/blog/pages/lz-string/index.html) |
| LZString Decompress | `lzstring-decompress` | [lz-string](https://pieroxy.net/blog/pages/lz-string/index.html) |
| Raw Deflate | `raw-deflate` | [DEFLATE](https://wikipedia.org/wiki/DEFLATE) |
| Raw Inflate | `raw-inflate` | [DEFLATE](https://wikipedia.org/wiki/DEFLATE) |
| Tar | `tar` | [tar](https://wikipedia.org/wiki/Tar_(computing)) |
| Untar | `untar` | [tar](https://wikipedia.org/wiki/Tar_(computing)) |
| Unzip | `unzip` | [ZIP](https://wikipedia.org/wiki/Zip_(file_format)) |
| Zip | `zip` | [ZIP](https://wikipedia.org/wiki/Zip_(file_format)) |
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

## LZ4 Compress

Compresses the input into an
[LZ4](https://wikipedia.org/wiki/LZ4_(compression_algorithm)) frame — a format
built for speed above all else. It compresses far more loosely than Deflate and
runs many times faster in both directions, which is why it turns up inside
filesystems, databases and network protocols rather than in archives. It takes
no options.

The frame written here asks for four-megabyte blocks that may refer back to one
another, and carries no checksum beyond the one over its own header. That is
what CyberChef writes, and the bytes match it exactly.

### Simple example

```bash
cchef lz4-compress -i "The cat sat on the mat." | cchef to-hex --delimiter None
```

Output:

```
04224d184070df170000805468652063617420736174206f6e20746865206d61742e00000000
```

There is nothing worth compressing in twenty-three bytes of English, so the
block is stored as it stands: `04224d18` names the format, `4070df` is the
descriptor and its checksum, `17000080` is a block of 0x17 bytes with the top
bit saying it was left alone, and `00000000` closes the frame.

### Complex example

Given something that does repeat, the block earns its keep:

```bash
cchef lz4-compress -i "hello hello hello hello hello hello" | cchef to-hex --delimiter None
```

Output:

```
04224d184070df100000006f68656c6c6f200600055068656c6c6f00000000
```

Thirty-five bytes become a sixteen-byte block: six literals (`hello `), then a
repeat of twenty-four bytes starting six back, then the five that are left.

## LZ4 Decompress

Reads an [LZ4](https://wikipedia.org/wiki/LZ4_(compression_algorithm)) frame
back into the bytes it was made from. It takes no options.

It reads more than CyberChef writes: a stated content size, a checksum over
each block or over the whole content, any of the four block sizes the format
allows, a dictionary identifier, and any number of frames written one after
another. Where a checksum is present it is checked, so damaged data is reported
rather than handed back quietly.

### Simple example

```bash
cchef lz4-compress -i "hello hello hello" | cchef lz4-decompress
```

Output:

```
hello hello hello
```

### Complex example

Frames written one after another are a normal way to hold LZ4 data, and are read
back as one:

```bash
printf 'one' | lz4 -c > a.lz4 && printf ' and two' | lz4 -c > b.lz4
cat a.lz4 b.lz4 | cchef lz4-decompress
```

Output:

```
one and two
```

## LZMA Compress

Compresses the input into an
[LZMA](https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm) stream —
the format behind `.lzma` files and, in another wrapper, `.xz` and 7-Zip. It
compresses more tightly than Deflate or bzip2 and takes longer to do it.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression Mode | option | `7` | How far back a repeat may reach, from 1 to 9. Larger costs memory and time and usually compresses better. The size chosen is recorded in the stream, so a reader needs no telling. |

The nine modes, as window sizes:

| Mode | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Window | 64 KB | 1 MB | 512 KB | 1 MB | 2 MB | 4 MB | 8 MB | 16 MB | 32 MB |

Modes 2 and 4 ask for the same window; the difference between them in CyberChef
is a setting that never reaches the stream. That ordering is not a typo, it is
what the mode table says.

**Fidelity.** The stream is valid LZMA that CyberChef, the `lzma` command and
7-Zip all read, but it is **not byte-identical to CyberChef's** — the two use
different encoders, and any number of streams decode to the same data. cchef's
output is the same size to within 0.1% on repetitive or random input, and 2.6%
to 6.3% larger on text and source code. The header also leaves the length
unstated and closes with an end marker, where CyberChef writes the length
outright; both forms are ordinary and every reader takes either.

### Simple example

```bash
cchef lzma-compress -i "The cat sat on the mat." | cchef lzma-decompress
```

Output:

```
The cat sat on the mat.
```

### Complex example

The window size is recorded in bytes 2 to 5 of the header, so the mode can be
read straight back out — `00000100` is 64 KB at mode 1, `00000002` is 32 MB at
mode 9:

```bash
cchef lzma-compress -i "hello" --compression-mode 1 | cchef to-hex --delimiter None
```

Output:

```
5d00000100ffffffffffffffff00341949ee8e6821ffffffb9e00000
```

## LZMA Decompress

Reads an [LZMA](https://wikipedia.org/wiki/Lempel-Ziv-Markov_chain_algorithm)
stream back into the bytes it was made from. It takes no options — everything
it needs is in the stream.

Both header forms are read: the length stated outright, as CyberChef writes it,
and the length left unknown with an end marker closing the stream, as the `lzma`
command writes it.

### Simple example

```bash
cchef lzma-compress -i "hello hello hello" | cchef lzma-decompress
```

Output:

```
hello hello hello
```

### Complex example

Reading a file written by `lzma` itself:

```bash
printf 'The cat sat on the mat.' | lzma -c | cchef lzma-decompress
```

Output:

```
The cat sat on the mat.
```

## LZNT1 Decompress

Reads an [LZNT1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xca/5655f4a3-6ba4-489b-959f-e1f407c52f15)
stream back into the bytes it was made from — the compression NTFS applies to a
file marked compressed, and what the Windows call `RtlDecompressBuffer` reads.
It takes no options, and there is no operation to write one, here or in
CyberChef.

A stream is a run of chunks holding up to four kilobytes each. A chunk either
keeps its bytes as they stand or encodes them as flag groups: a byte of flags
and then eight items, each a literal byte or a reference back over what the
chunk has produced. How a reference divides its sixteen bits between distance
and length changes as the chunk fills, so reaching further back costs a shorter
run.

**Fidelity.** Two deliberate differences from CyberChef, both matching the
specification above:

- A chunk holding exactly one byte is read. Its length field records the length
  less one, so it holds zero — which CyberChef takes for the end of the stream,
  dropping that byte and everything after it. A stream ends on a chunk header of
  `0x0000`, not on a length of zero.
- A stream a byte short of its last chunk is refused rather than handed back
  with a byte missing.

### Simple example

```bash
cchef from-hex --delimiter None -i "1ab000636f6d70726573730065647465737464610474610788616c6f74" | cchef lznt1-decompress
```

Output:

```
compressedtestdatacompressedalot
```

### Complex example

Reading a compressed stream out of one file and into another:

```bash
cchef lznt1-decompress --in-file compressed.bin -o recovered.bin
```

## LZString Compress

Compresses text with [lz-string](https://pieroxy.net/blog/pages/lz-string/index.html),
a library written to fit more into browser local storage — which is why it works
in characters rather than bytes and offers several shapes of output.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression Format | option | `default` | Which shape to write. See below. |

| Format | Writes | Good for |
| --- | --- | --- |
| `default` | Characters carrying sixteen bits each, the densest of the three | Keeping in memory, or anywhere sixteen-bit values survive |
| `UTF16` | Characters carrying fifteen bits, all lifted clear of the control range | Storing as text, which it was designed for |
| `Base64` | Ordinary Base64, six bits a character | URLs, JSON, anything that expects ASCII |

**A warning about the `default` format.** Filling all sixteen bits of a character
means about half of all inputs produce at least one *lone surrogate* — a value in
the range reserved for pairs, with no character behind it. Those have no spelling
in UTF-8. cchef writes them as three bytes holding the number itself and reads
them back the same way, so the round trip below always works; the output is not
valid UTF-8, and a terminal will show it as rubbish. Where no such value comes
up, the bytes are the same as CyberChef's. **For anything you mean to store or
send, use `UTF16` or `Base64`** — that is what they are there for, and lz-string
says the same.

### Simple example

```bash
cchef lzstring-compress -i "hello world" --compression-format Base64
```

Output:

```
BYUwNmD2AEDukCcwBMg=
```

### Complex example

The three formats over the same input, showing what each costs:

```bash
for f in default UTF16 Base64; do printf '%-8s %s bytes\n' "$f" "$(cchef lzstring-compress -i "the quick brown fox jumps over the lazy dog" --compression-format $f | wc -c | tr -d ' ')"; done
```

Output:

```
default  73 bytes
UTF16    80 bytes
Base64   76 bytes
```

Those are bytes, not characters. The plain format writes the fewest characters
of the three but spends up to three bytes on each of them, so on text like this
it is no smaller than Base64 once written out.

## LZString Decompress

Reads text back out of an [lz-string](https://pieroxy.net/blog/pages/lz-string/index.html)
stream. The format has to be the one it was written with — nothing in the stream
says which.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Compression Format | option | `default` | The shape the stream is in, as above. |

A stream that stops before the mark that ends it, or that names a dictionary
entry that was never built, is refused. CyberChef hands back what it had, or
nothing at all, without saying so.

### Simple example

```bash
cchef lzstring-compress -i "hello world" --compression-format Base64 | cchef lzstring-decompress --compression-format Base64
```

Output:

```
hello world
```

### Complex example

Reading a stream made somewhere else — this one came from lz-string in a
browser:

```bash
cchef lzstring-decompress -i "BYUwNmD2AEDukCcwBMg=" --compression-format Base64
```

Output:

```
hello world
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

## Tar

Packs the input into a [tar](https://wikipedia.org/wiki/Tar_(computing)) archive
under one name. Tar does not compress — it puts files end to end behind a header
each — so this is usually followed by Gzip or Bzip2 Compress.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Filename | string | `file.txt` | The name the file is stored under. Up to 100 bytes, which is all the header keeps for it. |

The archive is the ustar form: a 512-byte header, the data padded out to whole
blocks, and two blocks of nothing to close it. Only one file goes in, as in
CyberChef.

**The output is not the same twice.** The header records the time the archive
was written, and the checksum over the header changes with it. Everything else
is fixed.

**Fidelity.** Three deliberate differences from CyberChef:

- The data is padded to whole 512-byte blocks. CyberChef pads to one block and
  no further, so anything over 512 bytes that is not a multiple of it leaves the
  archive ending mid-block — which Go's tar reader refuses outright.
- A filename longer than the 100 bytes the header holds is refused. CyberChef
  writes it anyway, pushing every field after it out of place.
- A filename outside ASCII is written as UTF-8, which is what a tar reader
  expects. CyberChef narrows each character to a single byte, turning `café.txt`
  into Latin-1 and losing anything above the basic plane outright.

### Simple example

```bash
cchef tar -i "hello world" --filename greeting.txt | cchef untar --out-dir ./unpacked
cat ./unpacked/greeting.txt
```

Output:

```
hello world
```

### Complex example

An archive the `tar` command reads:

```bash
cchef tar -i "hello world" --filename greeting.txt -o hello.tar
tar tf hello.tar
```

Output:

```
greeting.txt
```

Tar first, then compress, which is what a `.tar.gz` is:

```bash
cchef tar -i "hello world" --filename greeting.txt | cchef gzip -o hello.tar.gz
```

## Untar

Unpacks a [tar](https://wikipedia.org/wiki/Tar_(computing)) archive into the
files it holds. It takes no options.

Because it produces several files, it needs `--out-dir` to write them:

```bash
cchef untar --in-file archive.tar --out-dir ./unpacked
```

Only regular files come out. Directories and links carry no contents of their
own and are passed over; the directories a file sits in are created as needed.

It reads more than CyberChef writes: archives from the `tar` command, including
the extensions it reaches for when a name will not fit in the header, and the
short-ending archives CyberChef itself produces.

### Simple example

```bash
cchef tar -i "hello world" --filename greeting.txt | cchef untar --out-dir ./unpacked
cat ./unpacked/greeting.txt
```

Output:

```
hello world
```

### Complex example

Unpacking a compressed archive made elsewhere:

```bash
cchef gunzip --in-file archive.tar.gz | cchef untar --out-dir ./unpacked
```

## Unzip

Unpacks the files from a [ZIP](https://wikipedia.org/wiki/Zip_(file_format))
archive. Because it produces several files rather than one stream, it needs
`--out-dir` to say where they should go.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Password | string | (empty) | Needed for an archive whose entries are encrypted. |
| Verify result | boolean | `false` | Check every file against the checksum recorded for it, and refuse the archive if one does not match. |

Entries stored either way are read, the directories inside an archive are
recreated, and entries for the directories themselves are skipped. An entry
naming a path that would climb out of the output directory is refused rather
than written.

A wrong password is always reported, whether or not `Verify result` is on: an
encrypted file is checked against its checksum regardless, since without that
there is nothing to tell a wrong password from a right one.

### Simple example

```bash
cchef zip -i "hello" --filename a.txt | cchef unzip --out-dir ./unpacked
cat ./unpacked/a.txt
```

Output:

```
hello
```

### Complex example

Unpacking an encrypted archive, checking each file as it goes:

```bash
cchef unzip --in-file secrets.zip --password hunter2 --verify-result --out-dir ./secrets
```

## Zip

Packs the input into a [ZIP](https://wikipedia.org/wiki/Zip_(file_format))
archive holding one file. Use [Unzip](#unzip) to read one back.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Filename | string | `file.txt` | The name the file is given inside the archive. |
| Comment | string | (empty) | Recorded against the file in the archive's directory. |
| Password | string | (empty) | Encrypts the file. Leave empty for an archive anyone can open. |
| Compression method | option | `Deflate` | `Deflate` compresses the file; `None (Store)` puts it in as it is. |
| Operating system | option | `MSDOS` | Recorded in the directory and nothing depends on it. |
| Compression type | option | `Dynamic Huffman Coding` | How each DEFLATE block is encoded, as for [Raw Deflate](#raw-deflate). Ignored when the method is `None (Store)`. |

**The output is not the same twice.** The archive records the time it was
written, in two places. With a password it varies further, because encryption
begins from twelve random bytes.

### Simple example

```bash
cchef zip -i "hello" --filename a.txt | cchef unzip --out-dir ./unpacked
cat ./unpacked/a.txt
```

Output:

```
hello
```

### Complex example

An encrypted archive, which `unzip` and other tools will open given the
password:

```bash
cchef zip --in-file report.pdf --filename report.pdf --password hunter2 -o report.zip
unzip -P hunter2 report.zip
```

**Fidelity.** Archives without a password are byte-identical to CyberChef's,
apart from the recorded time. Encrypted ones deliberately are not: CyberChef
fills all twelve bytes of the encryption header at random, where the last is
supposed to carry the top byte of the file's checksum so that a reader can tell
a wrong password from a right one. Its archives are therefore rejected by
`unzip`, 7-Zip and every other tool. cchef writes that byte correctly, so its
encrypted archives open anywhere — and its reader identifies a password by the
file's checksum rather than by that byte, so it still opens CyberChef's.

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

Every writer here bar **LZMA Compress** and **Tar**, which say so in their own
sections, produces the same bytes CyberChef does. That is not something that comes for
free: any number of different DEFLATE streams decode to the same
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

