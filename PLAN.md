# CyberChef CLI (`cchef`) — Plan & Status

## Context

`cchef` is a Go CLI port of the data-transformation engine of
[CyberChef](https://gchq.github.io/CyberChef/) (cloned at `../CyberChef`, a JS
project with 486 operations). The goal is a Unix-friendly tool where each
operation is a subcommand, operations chain into recipes (via pipes, JSON, or
CyberChef's "Chef" text format), and recipes round-trip to a shareable
`gchq.github.io/CyberChef` URL.

Development is **test-driven** (test → stub → implement), reusing CyberChef's own
fixture cases for parity, and keeps external dependencies minimal:

- `cobra` — the CLI framework.
- `github.com/elobuff/goamf` — backs the AMF operations (CyberChef likewise wraps
  an external AMF library).
- `golang.org/x/crypto/sha3` — provides the vetted legacy-Keccak constructors for
  Keccak-256/512. Keccak-224/384 (not in that package) use a small local
  Keccak-f sponge, cross-validated against x/crypto (256/512) and the stdlib
  `crypto/sha3` (SHA-3 mode) in tests.

## Current status

The core engine, recipe/URL machinery, CLI, docs, and a **curated set of 40
operations** are implemented, tested, and documented. The remaining CyberChef
operations are added incrementally against the same interfaces (see the
[Operation implementation status](#operation-implementation-status) checklist
below).

**Done:**

- **Core engine** (`internal/core/`): `Dish` (byte-backed type hub),
  `Operation` interface + `ArgDef`/`ToggleString` ingredient model, self-registering
  `Registry`, sequential `Recipe.Execute`, faithful ports of
  `GeneratePrettyRecipe`/`ParseRecipeConfig` (Chef format) and
  `EncodeURIFragment`/`BuildURL` (share URLs), each with byte-exact tests.
- **40 operations** (`internal/ops/`), each a faithful port with tests
  transcribed from CyberChef's `tests/operations/tests/*.mjs` fixtures.
- **CLI** (`cmd/`): auto-generated per-op subcommands (flags derived from arg
  defs, names sanitised), plus `bake`, `url`, `recipe convert`, `list`. Input
  resolves `--in-file` > `-i/--input` > positional args > stdin; output is
  byte-exact when piped and adds a trailing newline only on a TTY.
- **Docs** (`docs/`): per-category pages with options tables, simple + complex
  examples, and external reference links; operations listed alphabetically.
- **Tooling**: `Makefile` (alphabetised targets: build/test/fmt/vet/lint +
  `sbom`/`sbom-scan`/`sbom-audit` via cyclonedx-gomod + grype), `.gitignore`.
  `make test`, `make vet`, and `make lint` are clean.

## Architecture (as built)

```
cchef/
  main.go                      thin entry -> cmd.Execute()
  Makefile  .gitignore  go.mod
  cmd/
    root.go                    root cobra command
    register_ops.go            builds one subcommand per registered op (flags from ArgDefs)
    io.go                      input resolution (--in-file > -i > positional > stdin) + output
    bake.go                    `cchef bake` — run a JSON/Chef recipe (-e expr / -r file)
    url.go                     `cchef url` — emit a CyberChef share URL
    recipe.go                  `cchef recipe convert` — JSON <-> Chef
    list.go                    `cchef list` — list ops by module
  internal/core/
    dish.go        Dish{ data []byte; typ DishType } + conversions
    operation.go   Operation interface, ArgDef, ArgType, ToggleString, arg coercion
    registry.go    Register / Get / All; ops self-register via init()
    recipe.go      RecipeOp{Op,Args,Disabled,Breakpoint}, Recipe.Execute
    chef.go        GeneratePrettyRecipe / ParseRecipeConfig
    url.go         EncodeURIFragment / BuildURL
    naming.go      Kebab (op name -> subcommand)
  internal/ops/
    base64.go base32.go base45.go base58.go base62.go base85.go base92.go
    base_generic.go hex.go octal.go urlcode.go xor.go rot.go hashes.go
    case.go reverse.go amf.go sha3.go keccak.go hmac.go adler32.go
    fixtures_test.go (+ per-op _test.go)
  docs/
    README.md data-format.md encryption-encoding.md hashing.md utils.md recipes-and-urls.md
```

## Recipe formats & URLs

- **JSON**: `[{"op":"To Base64","args":["A-Za-z0-9+/="]}, ...]`
- **Chef** (compact): `To_Base64('A-Za-z0-9+/=')To_Hex('Space')`; auto-detected by
  a leading `[`. Optional `/disabled` `/breakpoint` flags.
- **URL**: `https://gchq.github.io/CyberChef/#recipe=<chef>&input=<base64,fragment-encoded>`.

## Usage

```bash
make build                                   # -> ./dist/cchef
echo -n hello | cchef to-base64 | cchef to-hex
cchef rot13 "Have a nice day."               # positional input
cchef bake -e "To_Base64()To_Hex()" -i hello # multi-op recipe
cchef url  -e "ROT13()" -i hello             # share link
cchef list                                   # discover operations
```

## How to add a new operation (the repeatable pattern)

1. Write `internal/ops/<name>_test.go` first, transcribing the matching cases
   from `../CyberChef/tests/operations/tests/<Op>.mjs` into the shared
   `opCase` table runner.
2. Add a stub type implementing `core.Operation` that compiles but fails (`make test` red).
3. Port the `run()` logic from `../CyberChef/src/core/operations/<Op>.mjs` until
   green. Register it in `init()` and add a docs entry in the relevant
   `docs/<category>.md` (alphabetised) with options + examples.

## Remaining / future work

- Implement more operations from the checklist below, prioritising common
  Data format / Encryption / Hashing ops.
- Additional Dish types as needed for ops that require them.
  (`JSON` was added for AMF; `BigNumber` for the generic To/From Base.)
- Flow control operations (Fork, Merge, Conditional Jump) need engine support
  beyond the current linear `Recipe.Execute`.
- `CRC Checksum` (parameterised over many algorithms via an argSelector) is
  deferred as a larger-than-straight-port effort.
- (Done: a repo-root `README.md` and GitHub Actions CI running fmt/vet/test/lint
  plus an SBOM scan now exist.)

## Verification

- `make test` (unit tests, fixtures), `make vet`, `make lint` — all clean.
- End-to-end: `printf 'hello' | ./dist/cchef to-base64` → `aGVsbG8=`;
  `printf 'hello' | ./dist/cchef md5` → `5d41402abc4b2a76b9719d911017c592`;
  `./dist/cchef url -e "To_Hex()" -i hello` → opens the recipe + input in CyberChef.

## Operation implementation status

All 486 CyberChef operations, grouped by CyberChef category and listed
alphabetically. `[x]` = implemented in cchef, `[ ]` = not yet. The per-category
count is `implemented/total`; some operations appear in more than one category.
Currently **37 unique** CyberChef operations are covered (36 directly plus
`SHA2`, exposed as the `sha256` and `sha512` subcommands).

### Data format (24/78)

- [x] AMF Decode
- [x] AMF Encode
- [ ] Avro to JSON
- [ ] Caret/M-decode
- [ ] CBOR Decode
- [ ] CBOR Encode
- [ ] Change IP format
- [ ] CSV to JSON
- [ ] Decode text
- [ ] Encode text
- [ ] Escape Smart Characters
- [ ] Escape Unicode Characters
- [x] From Base
- [x] From Base32
- [x] From Base45
- [x] From Base58
- [x] From Base62
- [x] From Base64
- [x] From Base85
- [x] From Base92
- [ ] From BCD
- [ ] From Bech32
- [ ] From Binary
- [ ] From Braille
- [ ] From Charcode
- [ ] From Decimal
- [ ] From Float
- [x] From Hex
- [ ] From Hex Content
- [ ] From Hexdump
- [ ] From HTML Entity
- [ ] From MessagePack
- [ ] From Modhex
- [x] From Octal
- [ ] From Punycode
- [ ] From Quoted Printable
- [ ] Hex to PEM
- [ ] JSON to CSV
- [ ] JSON to YAML
- [ ] MIME Decoding
- [ ] Normalise Unicode
- [ ] Parse ASN.1 hex string
- [ ] Parse TLV
- [ ] PEM to Hex
- [ ] Rison Decode
- [ ] Rison Encode
- [ ] Show Base64 offsets
- [ ] Swap endianness
- [ ] Text Encoding Brute Force
- [ ] Text-Integer Conversion
- [x] To Base
- [x] To Base32
- [x] To Base45
- [x] To Base58
- [x] To Base62
- [x] To Base64
- [x] To Base85
- [x] To Base92
- [ ] To BCD
- [ ] To Bech32
- [ ] To Binary
- [ ] To Braille
- [ ] To Charcode
- [ ] To Decimal
- [ ] To Float
- [x] To Hex
- [ ] To Hex Content
- [ ] To Hexdump
- [ ] To HTML Entity
- [ ] To MessagePack
- [ ] To Modhex
- [x] To Octal
- [ ] To Punycode
- [ ] To Quoted Printable
- [ ] Unescape Unicode Characters
- [x] URL Decode
- [x] URL Encode
- [ ] YAML to JSON

### Encryption / Encoding (3/84)

- [ ] A1Z26 Cipher Decode
- [ ] A1Z26 Cipher Encode
- [ ] AES Decrypt
- [ ] AES Encrypt
- [ ] AES Key Unwrap
- [ ] AES Key Wrap
- [ ] Affine Cipher Decode
- [ ] Affine Cipher Encode
- [ ] Atbash Cipher
- [ ] Bacon Cipher Decode
- [ ] Bacon Cipher Encode
- [ ] Bcrypt
- [ ] Bifid Cipher Decode
- [ ] Bifid Cipher Encode
- [ ] Blowfish Decrypt
- [ ] Blowfish Encrypt
- [ ] Bombe
- [ ] Caesar Box Cipher
- [ ] Cetacean Cipher Decode
- [ ] Cetacean Cipher Encode
- [ ] ChaCha
- [ ] CipherSaber2 Decrypt
- [ ] CipherSaber2 Encrypt
- [ ] Citrix CTX1 Decode
- [ ] Citrix CTX1 Encode
- [ ] Colossus
- [ ] Derive EVP key
- [ ] Derive HKDF key
- [ ] Derive PBKDF2 key
- [ ] DES Decrypt
- [ ] DES Encrypt
- [ ] Enigma
- [ ] Fernet Decrypt
- [ ] Fernet Encrypt
- [ ] Flask Session Decode
- [ ] Flask Session Sign
- [ ] Flask Session Verify
- [ ] From Morse Code
- [ ] GOST Decrypt
- [ ] GOST Encrypt
- [ ] GOST Key Unwrap
- [ ] GOST Key Wrap
- [ ] GOST Sign
- [ ] GOST Verify
- [ ] JWT Decode
- [ ] JWT Sign
- [ ] JWT Verify
- [ ] Lorenz
- [ ] LS47 Decrypt
- [ ] LS47 Encrypt
- [ ] Multiple Bombe
- [ ] Pseudo-Random Number Generator
- [ ] Rabbit
- [ ] Rail Fence Cipher Decode
- [ ] Rail Fence Cipher Encode
- [ ] RC2 Decrypt
- [ ] RC2 Encrypt
- [ ] RC4
- [ ] RC4 Drop
- [ ] RC6 Decrypt
- [ ] RC6 Encrypt
- [ ] ROR13
- [x] ROT13
- [ ] ROT13 Brute Force
- [x] ROT47
- [ ] ROT47 Brute Force
- [ ] ROT8000
- [ ] Salsa20
- [ ] Scrypt
- [ ] SIGABA
- [ ] SM4 Decrypt
- [ ] SM4 Encrypt
- [ ] Substitute
- [ ] To Morse Code
- [ ] Triple DES Decrypt
- [ ] Triple DES Encrypt
- [ ] Typex
- [ ] Vigenère Decode
- [ ] Vigenère Encode
- [x] XOR
- [ ] XOR Brute Force
- [ ] XSalsa20
- [ ] XXTEA Decrypt
- [ ] XXTEA Encrypt

### Public Key (0/31)

- [ ] ECDSA Sign
- [ ] ECDSA Signature Conversion
- [ ] ECDSA Verify
- [ ] Generate ECDSA Key Pair
- [ ] Generate PGP Key Pair
- [ ] Generate RSA Key Pair
- [ ] Hex to Object Identifier
- [ ] Hex to PEM
- [ ] JWK to PEM
- [ ] Object Identifier to Hex
- [ ] Parse ASN.1 hex string
- [ ] Parse CSR
- [ ] Parse SSH Host Key
- [ ] Parse X.509 certificate
- [ ] Parse X.509 CRL
- [ ] PEM to Hex
- [ ] PEM to JWK
- [ ] PGP Decrypt
- [ ] PGP Decrypt and Verify
- [ ] PGP Encrypt
- [ ] PGP Encrypt and Sign
- [ ] PGP Sign
- [ ] PGP Verify
- [ ] Public Key from Certificate
- [ ] Public Key from Private Key
- [ ] RSA Decrypt
- [ ] RSA Encrypt
- [ ] RSA Sign
- [ ] RSA Verify
- [ ] SM2 Decrypt
- [ ] SM2 Encrypt

### Arithmetic / Logic (2/30)

- [ ] ADD
- [ ] AND
- [ ] Bit shift left
- [ ] Bit shift right
- [ ] Cartesian Product
- [ ] Divide
- [ ] Extended GCD
- [ ] Mean
- [ ] Median
- [ ] Modular Exponentiation
- [ ] Modular Inverse
- [ ] Multiply
- [ ] NOT
- [ ] OR
- [ ] Power Set
- [ ] ROR13
- [x] ROT13
- [ ] ROT8000
- [ ] Rotate left
- [ ] Rotate right
- [ ] Set Difference
- [ ] Set Intersection
- [ ] Set Union
- [ ] Standard Deviation
- [ ] SUB
- [ ] Subtract
- [ ] Sum
- [ ] Symmetric Difference
- [x] XOR
- [ ] XOR Brute Force

### Networking (2/38)

- [ ] Change IP format
- [ ] Dechunk HTTP response
- [ ] Decode NetBIOS Name
- [ ] Defang IP Addresses
- [ ] Defang URL
- [ ] DNS over HTTPS
- [ ] Encode NetBIOS Name
- [ ] Fang URL
- [ ] Format MAC addresses
- [ ] Group IP addresses
- [ ] HASSH Client Fingerprint
- [ ] HASSH Server Fingerprint
- [ ] HTTP request
- [ ] IPv6 Transition Addresses
- [ ] JA3 Fingerprint
- [ ] JA3S Fingerprint
- [ ] JA4 Fingerprint
- [ ] JA4Server Fingerprint
- [ ] Parse Ethernet frame
- [ ] Parse IP range
- [ ] Parse IPv4 header
- [ ] Parse IPv6 address
- [ ] Parse SSH Host Key
- [ ] Parse TCP
- [ ] Parse TLS record
- [ ] Parse UDP
- [ ] Parse URI
- [ ] Parse User Agent
- [ ] Protobuf Decode
- [ ] Protobuf Encode
- [ ] Strip HTTP headers
- [ ] Strip IPv4 header
- [ ] Strip TCP header
- [ ] Strip UDP header
- [x] URL Decode
- [x] URL Encode
- [ ] VarInt Decode
- [ ] VarInt Encode

### Language (0/7)

- [ ] Convert Leet Speak
- [ ] Convert to NATO alphabet
- [ ] Decode text
- [ ] Encode text
- [ ] Remove Diacritics
- [ ] Unescape Unicode Characters
- [ ] Unicode Text Format

### Utils (3/52)

- [ ] Add line numbers
- [ ] Alternating Caps
- [ ] Convert area
- [ ] Convert co-ordinate format
- [ ] Convert data units
- [ ] Convert distance
- [ ] Convert mass
- [ ] Convert speed
- [ ] Count occurrences
- [ ] Diff
- [ ] Drop bytes
- [ ] Drop nth bytes
- [ ] Escape string
- [ ] Expand alphabet range
- [ ] File Tree
- [ ] Filter
- [ ] Find / Replace
- [ ] From Case Insensitive Regex
- [ ] Fuzzy Match
- [ ] Get All Casings
- [ ] Hamming Distance
- [ ] Head
- [ ] Levenshtein Distance
- [ ] Offset checker
- [ ] Pad lines
- [ ] Parse colour code
- [ ] Parse ObjectID timestamp
- [ ] Parse UNIX file permissions
- [ ] Pseudo-Random Number Generator
- [ ] Regular expression
- [ ] Remove ANSI Escape Codes
- [ ] Remove line numbers
- [ ] Remove null bytes
- [ ] Remove whitespace
- [x] Reverse
- [ ] Show on map
- [ ] Shuffle
- [ ] Sleep
- [ ] Sort
- [ ] Split
- [ ] Swap case
- [ ] Swap endianness
- [ ] Tail
- [ ] Take bytes
- [ ] Take nth bytes
- [ ] To Case Insensitive Regex
- [x] To Lower case
- [ ] To Table
- [x] To Upper case
- [ ] Unescape string
- [ ] Unique
- [ ] Wrap

### Date / Time (0/10)

- [ ] DateTime Delta
- [ ] Extract dates
- [ ] From UNIX Timestamp
- [ ] Get Time
- [ ] Parse DateTime
- [ ] Sleep
- [ ] To UNIX Timestamp
- [ ] Translate DateTime Format
- [ ] UNIX Timestamp to Windows Filetime
- [ ] Windows Filetime to UNIX Timestamp

### Extractors (0/20)

- [ ] CSS selector
- [ ] Extract Audio Metadata
- [ ] Extract dates
- [ ] Extract domains
- [ ] Extract email addresses
- [ ] Extract EXIF
- [ ] Extract file paths
- [ ] Extract Files
- [ ] Extract hashes
- [ ] Extract ID3
- [ ] Extract IP addresses
- [ ] Extract MAC addresses
- [ ] Extract URLs
- [ ] JPath expression
- [ ] Jsonata Query
- [ ] RAKE
- [ ] Regular expression
- [ ] Strings
- [ ] Template
- [ ] XPath expression

### Compression (0/19)

- [ ] Bzip2 Compress
- [ ] Bzip2 Decompress
- [ ] Gunzip
- [ ] Gzip
- [ ] LZ4 Compress
- [ ] LZ4 Decompress
- [ ] LZMA Compress
- [ ] LZMA Decompress
- [ ] LZNT1 Decompress
- [ ] LZString Compress
- [ ] LZString Decompress
- [ ] Raw Deflate
- [ ] Raw Inflate
- [ ] Tar
- [ ] Untar
- [ ] Unzip
- [ ] Zip
- [ ] Zlib Deflate
- [ ] Zlib Inflate

### Hashing (7/48)

- [x] Adler-32 Checksum
- [ ] Analyse hash
- [ ] Argon2
- [ ] Argon2 compare
- [ ] Bcrypt
- [ ] Bcrypt compare
- [ ] Bcrypt parse
- [ ] BLAKE2b
- [ ] BLAKE2s
- [ ] BLAKE3
- [ ] CMAC
- [ ] Compare CTPH hashes
- [ ] Compare SSDEEP hashes
- [ ] CRC Checksum
- [ ] CTPH
- [ ] Fletcher-16 Checksum
- [ ] Fletcher-32 Checksum
- [ ] Fletcher-64 Checksum
- [ ] Fletcher-8 Checksum
- [ ] Generate all checksums
- [ ] Generate all hashes
- [ ] GOST Hash
- [ ] HAS-160
- [x] HMAC
- [x] Keccak
- [ ] LM Hash
- [ ] Luhn Checksum
- [ ] MD2
- [ ] MD4
- [x] MD5
- [ ] MD6
- [ ] MurmurHash3
- [ ] NT Hash
- [ ] Parity Bit
- [ ] RIPEMD
- [ ] Scrypt
- [ ] SHA0
- [x] SHA1
- [x] SHA2 — sha224 / sha256 / sha384 / sha512 subcommands
- [x] SHA3
- [ ] Shake
- [ ] SM3
- [ ] Snefru
- [ ] SSDEEP
- [ ] Streebog
- [ ] TCP/IP Checksum
- [ ] Whirlpool
- [ ] XOR Checksum

### Code tidy (0/30)

- [ ] BSON deserialise
- [ ] BSON serialise
- [ ] CSS Beautify
- [ ] CSS Minify
- [ ] CSS selector
- [ ] Diff
- [ ] From MessagePack
- [ ] Generic Code Beautify
- [ ] JavaScript Beautify
- [ ] JavaScript Minify
- [ ] JavaScript Parser
- [ ] JPath expression
- [ ] Jq
- [ ] JSON Beautify
- [ ] JSON Minify
- [ ] Microsoft Script Decoder
- [ ] PHP Deserialize
- [ ] PHP Serialize
- [ ] Render Markdown
- [ ] SQL Beautify
- [ ] SQL Minify
- [ ] Strip HTML tags
- [ ] Syntax highlighter
- [ ] To Camel case
- [ ] To Kebab case
- [ ] To MessagePack
- [ ] To Snake case
- [ ] XML Beautify
- [ ] XML Minify
- [ ] XPath expression

### Forensics (0/12)

- [ ] Detect File Type
- [ ] ELF Info
- [ ] Extract Audio Metadata
- [ ] Extract EXIF
- [ ] Extract Files
- [ ] Extract LSB
- [ ] Extract RGBA
- [ ] Randomize Colour Palette
- [ ] Remove EXIF
- [ ] Scan for Embedded Files
- [ ] View Bit Plane
- [ ] YARA Rules

### Multimedia (0/29)

- [ ] Add Text To Image
- [ ] Blur Image
- [ ] Contain Image
- [ ] Convert Image Format
- [ ] Cover Image
- [ ] Crop Image
- [ ] Dither Image
- [ ] Extract EXIF
- [ ] Flip Image
- [ ] Generate Image
- [ ] Heatmap chart
- [ ] Hex Density chart
- [ ] Image Brightness / Contrast
- [ ] Image Filter
- [ ] Image Hue/Saturation/Lightness
- [ ] Image Opacity
- [ ] Invert Image
- [ ] Normalise Image
- [ ] Optical Character Recognition
- [ ] Play Media
- [ ] Remove EXIF
- [ ] Render Image
- [ ] Render PDF
- [ ] Resize Image
- [ ] Rotate Image
- [ ] Scatter chart
- [ ] Series chart
- [ ] Sharpen Image
- [ ] Split Colour Channels

### Other (0/22)

- [ ] Analyse UUID
- [ ] Automated Validation Test Op
- [ ] Chi Square
- [ ] Disassemble ARM
- [ ] Disassemble x86
- [ ] Entropy
- [ ] Frequency distribution
- [ ] Generate De Bruijn Sequence
- [ ] Generate HOTP
- [ ] Generate Lorem Ipsum
- [ ] Generate QR Code
- [ ] Generate TOTP
- [ ] Generate UUID
- [ ] Haversine distance
- [ ] HTML To Text
- [ ] Index of Coincidence
- [ ] Numberwang
- [ ] P-list Viewer
- [ ] Parse QR Code
- [ ] Pseudo-Random Integer Generator
- [ ] Pseudo-Random Number Generator
- [ ] XKCD Random Number

### Flow control (0/10)

- [ ] Comment
- [ ] Conditional Jump
- [ ] Fork
- [ ] Jump
- [ ] Label
- [ ] Magic
- [ ] Merge
- [ ] Register
- [ ] Return
- [ ] Subsection