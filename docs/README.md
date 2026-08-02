# cchef Documentation

`cchef` is a command-line port of [CyberChef](https://gchq.github.io/CyberChef/),
the "Cyber Swiss Army Knife". Every operation is a subcommand that reads input and
writes output, so operations chain together through Unix pipes or as a single
recipe.

> **Scope:** 505 operations, covering every CyberChef operation. Run `cchef list`
> to see them grouped by category, each with a one-line summary.

## Installing / building

```bash
make build      # produces ./dist/cchef
```

## How input and output work

Each operation resolves its input from the first source available, in this order:

1. `--in-file <path>` — read input from a file
2. `-i, --input <string>` — input given directly on the command line
3. **positional argument** — `cchef rot13 "Have a nice day."`
4. **stdin** — `echo hello | cchef rot13` (this is what makes pipes work)

Run with none of those at an interactive terminal and cchef says so rather than
waiting silently for you to type and press Ctrl-D:

```bash
cchef rot13
# cchef: no input given, and stdin is a terminal.
# Give text with -i, a file with --in-file, a directory with --in-dir, or pipe data in:
#   echo -n hello | cchef rot13
```

Redirecting from `/dev/null` still means empty input, not an error.

A positional argument is *text*, not a filename, which differs from tools like
`cat` and `grep`. Naming a file that exists is refused rather than silently
encoding the path:

```bash
cchef to-base64 notes.txt
# cchef: "notes.txt" names a file, but positional input is used as literal text.
# Read the file with --in-file notes.txt, keep the text with -i "notes.txt", or pass -- first
```

Output goes to stdout, or to a file with `-o, --output <path>`. When writing to a
terminal, a trailing newline is added for readability; when piped or redirected,
output stays byte-exact so operations chain cleanly. A `-` given to `--in-file`
or `--output` means stdin/stdout explicitly.

A few high-traffic operations have short aliases (e.g. `b64e`/`b64d` for
To/From Base64, `hex`/`unhex` for To/From Hex); `cchef <op> --help` lists them.

```bash
# All four input styles are equivalent here:
cchef to-base64 -i hello
cchef to-base64 hello
echo -n hello | cchef to-base64
cchef to-base64 --in-file ./greeting.txt
```

### Running over a directory

`--in-dir <path>` runs the operation (or `bake` recipe) once per file in a
directory. By default only the top-level files are processed; add `--recursive`
to walk subdirectories. Without `--out-dir`, results go to stdout with a
`==> name <==` header per file; with `--out-dir <path>`, one output file per
input is written there, mirroring the input tree. A file whose recipe fails is
reported to stderr and skipped, and the command exits non-zero.

This is the CLI counterpart to CyberChef's folder input. The same effect can be
had with a shell loop (`for f in dir/*; do cchef <op> --in-file "$f"; done`);
`--in-dir` is the built-in convenience.

```bash
# Base64-encode every file, results to stdout with per-file headers
cchef to-base64 --in-dir ./messages

# Recurse, writing one output file per input into ./encoded/
cchef to-base64 --in-dir ./messages --out-dir ./encoded --recursive
```

## Chaining operations

Two ways to combine operations:

```bash
# 1. Unix pipes — one subcommand per operation
echo -n hello | cchef to-base64 | cchef to-hex

# 2. A recipe — multiple operations in one command (see Recipes & URLs)
echo -n hello | cchef bake -e "To_Base64()To_Hex()"
```

## Option values

An option flag takes one of a fixed set of values, listed in its help text.
Case does not matter — `--delimiter comma` and `--delimiter Comma` both select
`Comma` — and the same holds in recipes and URLs, and combines with the
spellings below, so `--mode grayscale` selects `Greyscale`. One operation offers choices
that differ only in case (`To Morse Code`'s `Dash/Dot`, `DASH/DOT`, `dash/dot`,
where the casing is the setting); there an exact match wins, and a value that
matches several choices only by case is rejected rather than guessed at.

## Reading argument values from files

Every string-valued argument flag has a `--<flag>-file` companion that reads
the value from a file instead of the command line, keeping secrets like keys
and passphrases out of shell history and `ps` output. One trailing newline is
stripped, so an ordinary text file works as-is. Giving both the flag and its
`-file` companion is an error.

```bash
cchef aes-decrypt --key-file key.txt --iv 00000000000000000000000000000000 --in-file secret.bin
```

Recipes, URLs, and `bake` are unaffected: a recipe embeds its argument values
literally (a CyberChef share URL must carry them to work in the browser), so
put a secret in a file-flag invocation rather than a shared recipe or URL.

## British and American spellings

Operation, flag, and option names keep the spellings CyberChef uses, which are
often British. Every one of them also answers to its American spelling, so you
never have to remember which variant a name uses:

```bash
cchef analyse-hash -i 0800fc577294c34e0b28ad2839435945
cchef analyze-hash -i 0800fc577294c34e0b28ad2839435945   # same command

cchef view-bit-plane --in-file in.png --colour Red --bit 0 -o out.png
cchef view-bit-plane --in-file in.png --color  Red --bit 0 -o out.png

cchef generate-image -i 41424344 --mode Greyscale -o out.png
cchef generate-image -i 41424344 --mode Grayscale -o out.png

cchef convert-co-ordinate-format -i '51.5074, -0.1278' --output-format Geohash
cchef convert-coordinate-format  -i '51.5074, -0.1278' --output-format Geohash
```

Help text and this documentation show the CyberChef spelling; the American
spelling works everywhere but is not listed. The words covered are *analyse*,
*colour*, *centre*, *grey*, *metre*, *normalise*, *randomise*, and *serialise*,
plus the hyphenated *co-ordinate*.
Recipes, URLs, and `bake` are unaffected — they take operation and argument
names exactly as CyberChef writes them.

## Operation categories

Operations are grouped using the same categories as the original CyberChef tool.

Within each category, operations are listed alphabetically.

| Category | Operations |
| --- | --- |
| [Arithmetic / Logic](arithmetic-logic.md) | Cartesian Product, Divide, Extended GCD, MOD, Mean, Median, Modular Inverse, Multiply, Power Set, Set Difference, Set Intersection, Set Union, Standard Deviation, Subtract, Sum, Symmetric Difference |
| [Code tidy](code-tidy.md) | BSON deserialise, BSON serialise, CSS Beautify, CSS Minify, CSS selector, Diff, From MessagePack, Generic Code Beautify, JavaScript Beautify, JavaScript Minify, JavaScript Parser, JPath expression, Jq, JSON Beautify, JSON Minify, Microsoft Script Decoder, PHP Deserialize, PHP Serialize, Render Markdown, SQL Beautify, SQL Minify, Strip HTML tags, Syntax highlighter, To Camel case, To Kebab case, To MessagePack, To Snake case, XML Beautify, XML Minify, XPath expression |
| [Compression](compression.md) | Bzip2 Compress, Bzip2 Decompress, Gunzip, Gzip, LZ4 Compress, LZ4 Decompress, LZMA Compress, LZMA Decompress, LZNT1 Decompress, LZString Compress, LZString Decompress, Raw Deflate, Raw Inflate, Tar, Untar, Unzip, Zip, Zlib Deflate, Zlib Inflate |
| [Data format](data-format.md) | AMF Decode, AMF Encode, Avro to JSON, CBOR Decode, CBOR Encode, CSV to JSON, Caret/M-decode, Decode text, Encode text, Escape Smart Characters, Escape Unicode Characters, From BCD, From Base, From Base32, From Base45, From Base58, From Base62, From Base64, From Base85, From Base92, From Bech32, From Binary, From Braille, From COBS, From Charcode, From Decimal, From Float, From HTML Entity, From Hex, From Hex Content, From Hexdump, From MessagePack, From Modhex, From Octal, From Punycode, From Quoted Printable, Hex to PEM, JSON to CSV, JSON to YAML, MIME Decoding, Normalise Unicode, PEM to Hex, Parse ASN.1 hex string, Parse TLV, Rison Decode, Rison Encode, Show Base64 offsets, Swap endianness, Text Encoding Brute Force, Text-Integer Conversion, To BCD, To Base, To Base32, To Base45, To Base58, To Base62, To Base64, To Base85, To Base92, To Bech32, To Binary, To Braille, To COBS, To Charcode, To Decimal, To Float, To HTML Entity, To Hex, To Hex Content, To Hexdump, To MessagePack, To Modhex, To Octal, To Punycode, To Quoted Printable, URL Decode, URL Encode, Unescape Unicode Characters, YAML to JSON |
| [Date / Time](date-time.md) | DateTime Delta, Extract dates, From UNIX Timestamp, Get Time, Parse DateTime, To UNIX Timestamp, Translate DateTime Format, UNIX Timestamp to Windows Filetime, Windows Filetime to UNIX Timestamp |
| [Encryption / Encoding](encryption-encoding.md) | A1Z26 Cipher Decode, A1Z26 Cipher Encode, ADD, AES Decrypt, AES Encrypt, AES Key Unwrap, AES Key Wrap, AND, Affine Cipher Decode, Affine Cipher Encode, Ascon Decrypt, Ascon Encrypt, Atbash Cipher, Bacon Cipher Decode, Bacon Cipher Encode, Bcrypt, Bifid Cipher Decode, Bifid Cipher Encode, Bit shift left, Bit shift right, Blowfish Decrypt, Blowfish Encrypt, Bombe, Caesar Box Cipher, Cetacean Cipher Decode, Cetacean Cipher Encode, ChaCha, CipherSaber2 Decrypt, CipherSaber2 Encrypt, Citrix CTX1 Decode, Citrix CTX1 Encode, Colossus, DES Decrypt, DES Encrypt, Derive EVP key, Derive HKDF key, Derive PBKDF2 key, Enigma, Fernet Decrypt, Fernet Encrypt, Flask Session Decode, Flask Session Sign, Flask Session Verify, From Morse Code, GOST Decrypt, GOST Encrypt, GOST Key Unwrap, GOST Key Wrap, GOST Sign, GOST Verify, JWT Decode, JWT Sign, JWT Verify, LS47 Decrypt, LS47 Encrypt, Lorenz, Multiple Bombe, NOT, OR, PRESENT Decrypt, PRESENT Encrypt, Pseudo-Random Prime Generator, RC2 Decrypt, RC2 Encrypt, RC4, RC4 Drop, RC6 Decrypt, RC6 Encrypt, ROR13, ROT13, ROT13 Brute Force, ROT47, ROT47 Brute Force, ROT8000, Rabbit, Rail Fence Cipher Decode, Rail Fence Cipher Encode, Rotate left, Rotate right, SIGABA, SM4 Decrypt, SM4 Encrypt, SUB, Salsa20, Scrypt, Substitute, TEA Decrypt, TEA Encrypt, To Morse Code, Triple DES Decrypt, Triple DES Encrypt, Twofish Decrypt, Twofish Encrypt, Typex, Vigenère Decode, Vigenère Encode, XOR, XOR Brute Force, XSalsa20, XTEA Decrypt, XTEA Encrypt, XXTEA Decrypt, XXTEA Encrypt |
| [Extractors](extractors.md) | CSS selector, Extract Audio Metadata, Extract EXIF, Extract Files, Extract ID3, Extract IP addresses, Extract MAC addresses, Extract URLs, Extract dates, Extract domains, Extract email addresses, Extract file paths, Extract hashes, JPath expression, Jsonata Query, RAKE, Regular expression, Strings, Template, XPath expression |
| [Flow control](flow-control.md) | Comment, Conditional Jump, Fork, Jump, Label, Magic, Merge, Register, Return, Subsection |
| [Forensics](forensics.md) | Detect File Type, ELF Info, Extract Audio Metadata, Extract EXIF, Extract Files, Extract LSB, Extract RGBA, Randomize Colour Palette, Remove EXIF, Scan for Embedded Files, View Bit Plane, YARA Rules |
| [Hashing](hashing.md) | Adler-32 Checksum, Analyse hash, Argon2, Argon2 compare, Ascon Hash, Ascon MAC, BLAKE2b, BLAKE2s, BLAKE3, Bcrypt, Bcrypt compare, Bcrypt parse, CMAC, CRC Checksum, CTPH, Compare CTPH hashes, Compare SSDEEP hashes, Fletcher-16 Checksum, Fletcher-32 Checksum, Fletcher-64 Checksum, Fletcher-8 Checksum, Generate all checksums, Generate all hashes, GOST Hash, HAS-160, HMAC, Keccak, LM Hash, Luhn Checksum, MD2, MD4, MD5, MD6, MurmurHash3, NT Hash, Parity Bit, RIPEMD, Scrypt, SHA0, SHA1, SHA2, SHA224, SHA256, SHA3, SHA384, SHA512, SM3, SSDEEP, Shake, Snefru, Streebog, TCP/IP Checksum, Whirlpool, XOR Checksum |
| [Language](language.md) | Convert Leet Speak, Convert to NATO alphabet, Decode text, Encode text, Remove Diacritics, Unescape Unicode Characters, Unicode Text Format |
| [Multimedia](multimedia.md) | Add Text To Image, Blur Image, Contain Image, Convert Image Format, Cover Image, Crop Image, Dither Image, Extract EXIF, Flip Image, Generate Image, Heatmap chart, Hex Density chart, Image Brightness / Contrast, Image Filter, Image Hue/Saturation/Lightness, Image Opacity, Invert Image, Normalise Image, Optical Character Recognition, Play Media, Remove EXIF, Render Image, Render PDF, Resize Image, Rotate Image, Scatter chart, Series chart, Sharpen Image, Split Colour Channels |
| [Networking](networking.md) | Change IP format, Dechunk HTTP response, Decode NetBIOS Name, Defang IP Addresses, Defang URL, DNS over HTTPS, Encode NetBIOS Name, Fang URL, Format MAC addresses, Group IP addresses, HASSH Client Fingerprint, HASSH Server Fingerprint, HTTP request, IPv6 Transition Addresses, JA3 Fingerprint, JA3S Fingerprint, JA4 Fingerprint, JA4Server Fingerprint, Parse Ethernet frame, Parse IP range, Parse IPv4 header, Parse IPv6 address, Parse SSH Host Key, Parse TCP, Parse TLS record, Parse UDP, Parse URI, Parse User Agent, Protobuf Decode, Protobuf Encode, Strip HTTP headers, Strip IPv4 header, Strip TCP header, Strip UDP header, URL Decode, URL Encode, VarInt Decode, VarInt Encode |
| [Other](other.md) | Analyse UUID, Automated Validation Test Op, Chi Square, Disassemble ARM, Disassemble x86, Entropy, Frequency distribution, Generate De Bruijn Sequence, Generate HOTP, Generate Lorem Ipsum, Generate QR Code, Generate TOTP, Generate UUID, Haversine distance, HTML To Text, Index of Coincidence, Numberwang, P-list Viewer, Parse QR Code, Pseudo-Random Integer Generator, Pseudo-Random Number Generator, XKCD Random Number |
| [Public Key](public-key.md) | ECDSA Sign, ECDSA Signature Conversion, ECDSA Verify, Generate ECDSA Key Pair, Generate PGP Key Pair, Generate RSA Key Pair, Hex to Object Identifier, Hex to PEM, JWK to PEM, Object Identifier to Hex, PEM to Hex, PEM to JWK, PGP Decrypt, PGP Decrypt and Verify, PGP Encrypt, PGP Encrypt and Sign, PGP Sign, PGP Verify, Parse ASN.1 hex string, Parse CSR, Parse SSH Host Key, Parse X.509 CRL, Parse X.509 certificate, Public Key from Certificate, Public Key from Private Key, RSA Decrypt, RSA Encrypt, RSA Sign, RSA Verify, SM2 Decrypt, SM2 Encrypt |
| [Utils](utils.md) | Add line numbers, Alternating Caps, Convert area, Convert co-ordinate format, Convert data units, Convert distance, Convert mass, Convert speed, Count occurrences, Diff, Drop bytes, Drop nth bytes, Escape string, Expand alphabet range, File Tree, Filter, Find / Replace, From Case Insensitive Regex, Fuzzy Match, Get All Casings, Hamming Distance, Head, Levenshtein Distance, Offset checker, Pad lines, Parse colour code, Parse ObjectID timestamp, Parse UNIX file permissions, Regular expression, Remove ANSI Escape Codes, Remove line numbers, Remove null bytes, Remove whitespace, Reverse, Show on map, Shuffle, Sleep, Sort, Split, Swap case, Tail, Take bytes, Take nth bytes, To Case Insensitive Regex, To Lower case, To Table, To Upper case, Unescape string, Unique, Wrap |

## Recipes, URLs, and tooling

- [Recipes & URLs](recipes-and-urls.md) — `bake`, `url`, `recipe convert`, recipe
  formats (JSON and Chef), and generating CyberChef share links.

## Command reference

| Command | Purpose |
| --- | --- |
| `cchef <operation> [flags]` | Run a single operation |
| `cchef bake -e/-r <recipe>` | Run a multi-operation recipe (JSON or Chef format) |
| `cchef url -e/-r <recipe>` | Print a CyberChef share URL for a recipe |
| `cchef recipe convert` | Convert a recipe between JSON and Chef formats |
| `cchef list` | List available operations grouped by category, each with a one-line summary |
| `cchef list --json` | The same listing as JSON, for scripts and shell completions |
| `cchef --version` | Print the cchef version |
