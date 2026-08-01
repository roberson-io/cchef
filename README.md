# cchef

[![CI](https://github.com/roberson-io/cchef/actions/workflows/ci.yml/badge.svg)](https://github.com/roberson-io/cchef/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**CyberChef on the command line.** `cchef` is a Go port of the data-transformation
engine of [CyberChef](https://gchq.github.io/CyberChef/) — the "Cyber Swiss Army
Knife" built for the terminal. The "Swiss Army Knife" analogy is apt. This tool is useful for lower stakes use cases but is not recommended for production or critical infrastructure purposes.

Every operation is a subcommand that reads input
and writes output, so operations chain together through Unix pipes or as a single
recipe, and any recipe can be turned into a shareable CyberChef URL.

## Install

```bash
make build      # produces ./dist/cchef
```

Requires Go 1.26+. The result is a single static binary with no cgo.

## Quickstart

```bash
# An operation reads from a positional arg, -i, --in-file, or stdin:
cchef to-base64 hello                 # aGVsbG8=
cchef to-base64 -i hello              # aGVsbG8=
echo -n hello | cchef to-base64       # aGVsbG8=

# Chain operations with pipes:
echo -n hello | cchef to-base64 | cchef to-hex
# 61 47 56 73 62 47 38 3d

# Hash something:
cchef sha256 -i 'Hello, World!'
# dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f

# Run a whole recipe at once (JSON or compact "Chef" format):
echo -n hello | cchef bake -e "To_Base64()To_Hex()"

# Work out what unknown data is, and how to decode it:
cchef magic -i "41 42 43 44 45"

# Turn a recipe into a CyberChef share URL:
cchef url -e "ROT13()" -i hello
# https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8

# Discover what's available (grouped by category, with a one-line summary each):
cchef list
cchef --version

# Common operations have short aliases:
cchef b64e hello                      # alias for to-base64  -> aGVsbG8=

# Process a whole directory of files (CyberChef's folder input):
cchef to-base64 --in-dir ./messages                    # results to stdout, per-file headers
cchef to-base64 --in-dir ./messages --out-dir ./out --recursive
```

Output is byte-exact when piped or redirected (so chaining is lossless); a
trailing newline is added only when writing to a terminal. Operations accept
`--in-file -` / `--output -` to force stdin/stdout in a pipeline.

Point an operation (or `bake`) at a directory with `--in-dir` to run it once per
file — top-level by default, `--recursive` to walk subdirectories. Results go to
stdout with `==> name <==` headers, or to `--out-dir` as one output file per
input; a file whose recipe fails is reported and skipped (non-zero exit).

## Operations

Operations are grouped using the same categories as CyberChef. Each page
documents every operation's options, examples and reference links:

| Category | Covers |
| --- | --- |
| [Arithmetic / Logic](docs/arithmetic-logic.md) | Sum, Subtract, Multiply, Divide, Mean, Median, Standard Deviation, MOD, Extended GCD, Modular Inverse, and the set operations |
| [Code tidy](docs/code-tidy.md) | Beautify and minify for JavaScript, CSS, SQL, XML and JSON; PHP and BSON serialization; JPath, XPath, jq and CSS selectors; case conversion |
| [Compression](docs/compression.md) | Gzip, Zlib, Raw Deflate, Bzip2, LZMA, LZ4, LZNT1, LZString, Zip and Tar |
| [Data format](docs/data-format.md) | Base32/45/58/62/64/85/92, Hex, Binary, Octal, Decimal, Charcode, BCD, Float, Hexdump, Braille, Punycode, COBS, MessagePack, CBOR, Avro, YAML, ASN.1, TLV, URL and HTML entity encoding |
| [Date / Time](docs/date-time.md) | UNIX and Windows timestamps, DateTime parsing, formatting and deltas |
| [Encryption / Encoding](docs/encryption-encoding.md) | AES, DES, Triple DES, Blowfish, Twofish, RC2, RC4, RC6, ChaCha, Salsa20, Rabbit, TEA, XTEA, XXTEA, SM4, PRESENT, GOST, Ascon, Fernet, JWT; the classical ciphers; the Bletchley Park machines (Enigma, Bombe, Lorenz, Colossus, Typex, SIGABA); and the bitwise operations |
| [Extractors](docs/extractors.md) | Pull IPs, URLs, domains, email addresses, file paths, hashes, dates, EXIF, ID3 and embedded files out of data; regular expressions, RAKE and Jsonata |
| [Flow control](docs/flow-control.md) | Fork, Merge, Subsection, Jump, Conditional Jump, Label, Register, Return, Comment, and Magic |
| [Forensics](docs/forensics.md) | Detect File Type, Scan for Embedded Files, ELF Info, YARA Rules, and the steganography operations |
| [Hashing](docs/hashing.md) | MD2/4/5, SHA-0/1/2/3, Keccak, Shake, BLAKE2/3, RIPEMD, Whirlpool, Streebog, SM3, Ascon, HMAC, CMAC, bcrypt, scrypt, Argon2, CRC and the fuzzy hashes |
| [Language](docs/language.md) | Character-encoding conversion, Unicode escapes and formatting, diacritics, NATO alphabet and Leet Speak |
| [Multimedia](docs/multimedia.md) | Image resizing, cropping, filtering and format conversion; EXIF; audio metadata; QR codes |
| [Networking](docs/networking.md) | IP address arithmetic and formats, CIDR, MAC addresses, URI parsing, HTTP, DNS-over-HTTPS, TLS/JA3 and user agents |
| [Other](docs/other.md) | Entropy, frequency and hash analysis, sequence and password generators, HTML rendering, Numberwang |
| [Public Key](docs/public-key.md) | RSA, ECDSA and PGP; X.509 certificates and CSRs; key generation, signing and verification |
| [Utils](docs/utils.md) | Sorting, filtering, unique, find and replace, padding, escaping, unit and coordinate conversion, diff, file trees |

Flags and options are aliased to support both US and UK spellings: `analyze-hash` runs
`analyse-hash`, `--color` sets `--colour`, and `Grayscale` selects `Greyscale`.
See [British and American spellings](docs/README.md#british-and-american-spellings).

See [docs/](docs/README.md) for the full documentation index, and
[docs/recipes-and-urls.md](docs/recipes-and-urls.md) for `bake`, `url`, and
`recipe convert`.

## Recipes

A recipe is an ordered list of operations, expressible in two formats (auto-detected):

- **JSON:** `[{"op":"To Base64","args":["A-Za-z0-9+/="]}, ...]`
- **Chef** (compact): `To_Base64('A-Za-z0-9+/=')To_Hex('Space')`

Run one with `cchef bake -e <recipe>` / `-r <file>`, convert between formats with
`cchef recipe convert`, or share it with `cchef url`.

## Development

```bash
make all      # the full gate: fmt, fix, vet, test, build, lint, sec
make test     # run all unit tests
make lint     # golangci-lint (make install-tools to install it)
make sec      # gosec SAST + govulncheck (dependency & stdlib CVEs)
make complexity   # report functions over the cyclomatic threshold
make sbom-audit   # generate + scan a CycloneDX SBOM
make fuzz         # run every fuzz target (FUZZTIME=5m for a longer run)
```

Fuzzing covers the parsers that read data cchef did not write — the recipe
parser, the file-format and rule parsers, the decoders — and checks that the
reciprocal operations round-trip. `make fuzz` gives each target 30 seconds;
a failing input is written under `testdata/fuzz/` and becomes a regression
test from then on.

CI runs on linux/amd64 while development is typically on arm64, so anything
touching floating-point precision is worth checking on both:

```bash
docker run --rm --platform linux/amd64 -v "$PWD":/src -w /src \
    -e GOFLAGS=-buildvcs=false -e GOGC=off golang:1.26 go test ./...
```

Security scanning uses [gosec](https://github.com/securego/gosec) (SAST) and
[govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) (known,
reachable CVEs in dependencies and the Go stdlib), on top of the SBOM/grype
supply-chain scan. gosec findings that are by design for a CyberChef port —
intentional MD5/SHA1 operations, bounded byte/bit conversions, reading
user-supplied file paths — are annotated in-code with justified `// #nosec`
comments; `make sec` requires every suppression to name its rule and reason, and
prints the full suppression list for audit.

Operations are developed **test-first**, with test cases transcribed from
CyberChef's own fixtures (`../CyberChef/tests/operations/tests/*.mjs`) for
byte-for-byte parity, and differential-tested against a running CyberChef where
one can be reached. The repeatable workflow for porting an operation is captured
in the [`/add-operation`](.claude/skills/add-operation/SKILL.md) skill.

Where cchef's answer deliberately differs from CyberChef's, it is because
CyberChef has a defect: those are cataloged, with reproductions, in the bug log
kept alongside this repository, and noted in the relevant `docs/` page.

### Generated tables

Some operations are backed by large tables generated from an upstream source
rather than written by hand. Each has a tool with its own README describing how
to refresh it:

- [`tools/htmlentgen`](tools/htmlentgen/) — HTML entity tables, from the WHATWG
  named character reference set.
- [`tools/magicgen`](tools/magicgen/) — the Magic operation's detection checks
  and language byte-frequency profiles.
- [`tools/cpgen`](tools/cpgen/) — code page tables for the text encodings.

### Dependencies

`cchef` leans on the standard library, with a small set of direct dependencies:

| Dependency | Used for |
| --- | --- |
| [`spf13/cobra`](https://github.com/spf13/cobra) | The CLI framework |
| [`dlclark/regexp2`](https://github.com/dlclark/regexp2) | Backtracking regular expressions where RE2 cannot express a CyberChef pattern |
| [`golang.org/x/crypto`](https://pkg.go.dev/golang.org/x/crypto) | Legacy-Keccak, bcrypt, scrypt, Argon2 and other hashers |
| [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text) | Unicode normalization and text encodings |
| [`golang.org/x/image`](https://pkg.go.dev/golang.org/x/image) | BMP, TIFF and WEBP decoding for the image operations |
| [`golang.org/x/arch`](https://pkg.go.dev/golang.org/x/arch) | ARM and ARM64 disassembly |
| [`ProtonMail/go-crypto`](https://github.com/ProtonMail/go-crypto) | OpenPGP, interoperable with CyberChef's kbpgp |
| [`golang-jwt/jwt`](https://github.com/golang-jwt/jwt) | JWT sign, verify and decode |
| [`alecthomas/chroma`](https://github.com/alecthomas/chroma) | Syntax highlighting |
| [`evanw/esbuild`](https://github.com/evanw/esbuild) | JavaScript parsing, beautifying and minifying |
| [`yuin/goldmark`](https://github.com/yuin/goldmark) | Markdown rendering |
| [`itchyny/gojq`](https://github.com/itchyny/gojq) | The jq operation |
| [`recolabs/gnata`](https://github.com/RecoLabs/gnata) | The Jsonata Query operation |
| [`antchfx/xpath`](https://github.com/antchfx/xpath) | XPath expressions |
| [`go.yaml.in/yaml`](https://github.com/yaml/go-yaml) | YAML conversion |
| [`google.golang.org/protobuf`](https://pkg.go.dev/google.golang.org/protobuf), [`bufbuild/protocompile`](https://github.com/bufbuild/protocompile) | Protobuf decoding |
| [`ulikunitz/xz`](https://github.com/ulikunitz/xz) | The LZMA codec |
| [`klaus-tockloth/coco`](https://github.com/klaus-tockloth/coco) | Coordinate formats and conversion |

Ciphers, hashes, compression formats and file parsers are written from scratch
rather than pulled in, so the binary stays self-contained and the behavior stays
identical to CyberChef's. [PLAN.md](PLAN.md) records the rationale for each
dependency, including what it drags in transitively, and which ones are candidates
for replacement.

## Contributing

Bug reports and pull requests are welcome — see
[CONTRIBUTING.md](CONTRIBUTING.md) for how to set up a development
environment and what changes are expected to meet.

## AI Disclosure

Most of this codebase — implementation, tests, and documentation — was written
with [Claude Code](https://claude.com/claude-code), Anthropic's AI coding
agent, directed and reviewed by the maintainer, who makes every commit.

AI output is treated as untrusted until verified: operations are tested
against CyberChef's own fixtures or checked byte for byte against a running
CyberChef instance, and every change passes the full CI gate. If you find
something that verification missed, please open an issue.

## License

Released under the [Apache License 2.0](LICENSE). Attribution for the upstream project and for third-party material included
here is in [NOTICE](NOTICE).

## Credits

`cchef` is an independent port of [CyberChef](https://github.com/gchq/CyberChef)
by GCHQ (Crown Copyright, Apache-2.0). All operation semantics and test vectors
derive from that project. cchef is not affiliated with or endorsed by GCHQ.
