# cchef Documentation

`cchef` is a command-line port of [CyberChef](https://gchq.github.io/CyberChef/),
the "Cyber Swiss Army Knife". Every operation is a subcommand that reads input and
writes output, so operations chain together through Unix pipes or as a single
recipe.

> **Scope:** 254 operations are currently ported, covering the whole architecture.
> More operations are added over time against the same interfaces. Run `cchef list`
> to see everything currently available.

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

## Operation categories

Operations are grouped using the same categories as the original CyberChef tool.

Within each category, operations are listed alphabetically.

| Category | Operations |
| --- | --- |
| [Arithmetic / Logic](arithmetic-logic.md) | Cartesian Product, Divide, Mean, Median, Multiply, Power Set, Set Difference, Set Intersection, Set Union, Standard Deviation, Subtract, Sum, Symmetric Difference |
| [Code tidy](code-tidy.md) | Diff, From MessagePack, To MessagePack |
| [Data format](data-format.md) | AMF Decode, AMF Encode, Avro to JSON, Caret/M-decode, CBOR Decode, CBOR Encode, CSV to JSON, Decode text, Encode text, Escape Smart Characters, Escape Unicode Characters, From Base, From Base32, From Base45, From Base58, From Base62, From Base64, From Base85, From Base92, From BCD, From Bech32, From Binary, From Braille, From Charcode, From Decimal, From Float, From Hex, From Hex Content, From Hexdump, From HTML Entity, From MessagePack, From Modhex, From Octal, From Punycode, From Quoted Printable, Hex to PEM, JSON to CSV, JSON to YAML, MIME Decoding, Normalise Unicode, Parse ASN.1 hex string, Parse TLV, PEM to Hex, Rison Decode, Rison Encode, Show Base64 offsets, Swap endianness, Text Encoding Brute Force, Text-Integer Conversion, To Base, To Base32, To Base45, To Base58, To Base62, To Base64, To Base85, To Base92, To BCD, To Bech32, To Binary, To Braille, To Charcode, To Decimal, To Float, To Hex, To Hex Content, To Hexdump, To HTML Entity, To MessagePack, To Modhex, To Octal, To Punycode, To Quoted Printable, Unescape Unicode Characters, URL Decode, URL Encode, YAML to JSON |
| [Date / Time](date-time.md) | DateTime Delta, Extract dates, From UNIX Timestamp, Get Time, Parse DateTime, To UNIX Timestamp, Translate DateTime Format, UNIX Timestamp to Windows Filetime, Windows Filetime to UNIX Timestamp |
| [Encryption / Encoding](encryption-encoding.md) | A1Z26 Cipher Decode, A1Z26 Cipher Encode, ADD, AES Decrypt, AES Encrypt, AES Key Unwrap, AES Key Wrap, Affine Cipher Decode, Affine Cipher Encode, AND, Ascon Decrypt, Ascon Encrypt, Atbash Cipher, Bacon Cipher Decode, Bacon Cipher Encode, Bcrypt, Bifid Cipher Decode, Bifid Cipher Encode, Bit shift left, Bit shift right, Blowfish Decrypt, Blowfish Encrypt, Bombe, Caesar Box Cipher, Cetacean Cipher Decode, Cetacean Cipher Encode, ChaCha, CipherSaber2 Decrypt, CipherSaber2 Encrypt, Citrix CTX1 Decode, Citrix CTX1 Encode, Colossus, Derive EVP key, Derive HKDF key, Derive PBKDF2 key, Enigma, Lorenz, Multiple Bombe, NOT, OR, ROR13, ROT13, ROT47, ROT8000, Rotate left, Rotate right, SUB, XOR, XOR Brute Force |
| [Hashing](hashing.md) | Adler-32 Checksum, Bcrypt, Bcrypt compare, Bcrypt parse, HAS-160, HMAC, Keccak, MD2, MD4, MD5, RIPEMD, SHA0, SHA1, SHA224, SHA256, SHA3, SHA384, SHA512, Snefru, Whirlpool |
| [Language](language.md) | Decode text, Encode text, Unescape Unicode Characters |
| [Networking](networking.md) | Change IP format, Dechunk HTTP response, Decode NetBIOS Name, Defang IP Addresses, Defang URL, DNS over HTTPS, Encode NetBIOS Name, Fang URL, Format MAC addresses, Group IP addresses, HASSH Client Fingerprint, HASSH Server Fingerprint, HTTP request, IPv6 Transition Addresses, JA3 Fingerprint, JA3S Fingerprint, JA4 Fingerprint, JA4Server Fingerprint, Parse Ethernet frame, Parse IP range, Parse IPv4 header, Parse IPv6 address, Parse SSH Host Key, Parse TCP, Parse TLS record, Parse UDP, Parse URI, Parse User Agent, Protobuf Decode, Protobuf Encode, Strip HTTP headers, Strip IPv4 header, Strip TCP header, Strip UDP header, URL Decode, URL Encode, VarInt Decode, VarInt Encode |
| [Public Key](public-key.md) | Hex to PEM, PEM to Hex, Parse ASN.1 hex string, Parse SSH Host Key |
| [Utils](utils.md) | Add line numbers, Alternating Caps, Convert area, Convert co-ordinate format, Convert data units, Convert distance, Convert mass, Convert speed, Count occurrences, Diff, Drop bytes, Drop nth bytes, Escape string, Expand alphabet range, File Tree, Filter, Find / Replace, From Case Insensitive Regex, Fuzzy Match, Get All Casings, Hamming Distance, Head, Levenshtein Distance, Offset checker, Pad lines, Parse colour code, Parse ObjectID timestamp, Parse UNIX file permissions, Pseudo-Random Number Generator, Regular expression, Remove ANSI Escape Codes, Remove line numbers, Remove null bytes, Remove whitespace, Reverse, Show on map, Shuffle, Sleep, Sort, Split, Swap case, Tail, Take bytes, Take nth bytes, To Case Insensitive Regex, To Lower case, To Table, To Upper case, Unescape string, Unique, Wrap |

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
| `cchef --version` | Print the cchef version |
