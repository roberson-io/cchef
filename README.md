# cchef

[![CI](https://github.com/roberson-io/cchef/actions/workflows/ci.yml/badge.svg)](https://github.com/roberson-io/cchef/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/roberson-io/cchef.svg)](https://pkg.go.dev/github.com/roberson-io/cchef)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**CyberChef on the command line.** `cchef` is a Go port of the data-transformation
engine of [CyberChef](https://gchq.github.io/CyberChef/) — the "Cyber Swiss Army
Knife" built for the terminal.

Every operation is a subcommand that reads input
and writes output, so operations chain together through Unix pipes or as a single
recipe, and any recipe can be turned into a shareable CyberChef URL.

![Terminal recording: cchef encodes to Base64 and chains that into a hex dump,
then writes a QR code for a URL to a PNG file and reads the URL back out of it;
builds the same two-step recipe twice — once from bare subcommand names, once in
CyberChef's own recipe syntax, for identical results — disables the second step
and bakes it; turns a recipe into a CyberChef share link and reads one back
again; and finishes with magic working out unaided that some opaque text is
base64-wrapped gzip.](assets/demo.gif)

## Install

**Homebrew** (macOS and Linux) — also installs the man page and completions:

```bash
brew install roberson-io/tap/cchef
```

**Debian / Ubuntu** (`.deb`) and **Fedora / RHEL** (`.rpm`) — these fetch
whatever the latest release is; use `linux_arm64` on 64-bit Arm:

```bash
curl -LO "$(curl -fsSL https://api.github.com/repos/roberson-io/cchef/releases/latest \
  | grep -o 'https://[^"]*linux_amd64\.deb"' | tr -d '"')"
sudo dpkg -i cchef_*_linux_amd64.deb
```

```bash
curl -LO "$(curl -fsSL https://api.github.com/repos/roberson-io/cchef/releases/latest \
  | grep -o 'https://[^"]*linux_amd64\.rpm"' | tr -d '"')"
sudo rpm -i cchef_*_linux_amd64.rpm
```

**Windows** — with [Scoop](https://scoop.sh/):

```powershell
scoop bucket add roberson-io https://github.com/roberson-io/scoop-bucket
scoop install cchef
```

…or from the zip, which also carries the man page, docs and PowerShell
completions:

```powershell
$Url = (Invoke-RestMethod https://api.github.com/repos/roberson-io/cchef/releases/latest).assets |
  Where-Object name -like '*windows_amd64.zip' |
  Select-Object -ExpandProperty browser_download_url
Invoke-WebRequest -Uri $Url -OutFile cchef.zip
Expand-Archive cchef.zip -DestinationPath $env:LOCALAPPDATA\cchef
```

Then add `%LOCALAPPDATA%\cchef` to your `PATH`.

**With Go** (Go 1.27+; installs no man page or completions):

```bash
go install github.com/roberson-io/cchef@latest
```

**From source:**

```bash
make build      # produces ./dist/cchef
```

The result is a single static binary with no cgo; the only optional runtime
dependency is `tesseract`, for the Optical Character Recognition operation.

Shell completion is built in — `cchef completion bash|zsh|fish|powershell`
prints a script. Homebrew and the deb/rpm packages install the bash, zsh and
fish scripts for you; the PowerShell one ships in the Windows archive.

> See **[docs/install.md](docs/install.md)** for per-platform detail, what each
> package installs where, uninstalling, and
> [verifying a release](docs/install.md#verifying-a-release) (checksums, cosign
> signature, SLSA provenance and SBOMs).

## Quickstart

An operation reads from a positional argument, `-i`, `--in-file`, or stdin —
all three of these print `aGVsbG8=`:

```bash
cchef to-base64 "hello"
```

```bash
cchef to-base64 -i "hello"
```

```bash
echo -n "hello" | cchef to-base64
```

Chain operations with pipes (`61 47 56 73 62 47 38 3d`):

```bash
cchef to-base64 "hello" | cchef to-hex
```

Hash something:

```bash
cchef sha256 -i 'Hello, World!'
```

Run a whole recipe at once (JSON or compact "Chef" format):

```bash
cchef bake -e "To_Base64()To_Hex()" "hello"
```

Work out what unknown data is, and how to decode it:

```bash
cchef magic -i "41 42 43 44 45"
```

Turn a recipe into a CyberChef share URL:

```bash
cchef url -e "ROT13()" -i "hello"
```

Discover what's available — every operation grouped by category with a
one-line summary — and note that common operations have short aliases
(`b64e` runs `to-base64`):

```bash
cchef list
cchef b64e "hello"
```

Process a whole directory of files (CyberChef's folder input):

```bash
cchef to-base64 --in-dir ./messages
cchef to-base64 --in-dir ./messages --out-dir ./out --recursive
```

A trailing newline is added only when writing to a terminal. Operations accept
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
| [Arithmetic / Logic](docs/arithmetic-logic.md) | Sum, Subtract, Multiply, Divide, Mean, Median, Standard Deviation, MOD, Extended GCD, Modular Exponentiation, Modular Inverse, and the set operations |
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

A CyberChef share link works in both directions — `cchef url` writes one, and
`--from-url` reads the recipe (and any input) back out of one, offline:

```bash
cchef bake --from-url "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8"
```

A recipe can also be built up interactively, one operation at a time. The
staged recipe lives in `.cchef-recipe.json` in the working directory, and
`bake`, `url` and `recipe convert` all use it when given no recipe of their own:

```bash
cchef recipe add "To_Base64()"
cchef recipe add "To_Hex('Space')"
cchef recipe show
cchef bake "hello"
```

`show` lists the staged steps numbered, each marked `[X]` when it runs or `[ ]`
when it is disabled; `rm`, `move` and `toggle` edit them by number, and `clear`
discards the recipe. `load` replaces the whole staged recipe with one from a
file, string or share link, and `recipe convert` prints it back out. See
[`cchef recipe add` and friends](docs/recipes-and-urls.md#cchef-recipe-add-and-friends--build-a-recipe-step-by-step).

## Use as a Go library

The engine is importable, so a Go program can bake recipes without shelling
out. [`core`](https://pkg.go.dev/github.com/roberson-io/cchef/core) is the
engine; importing [`ops`](https://pkg.go.dev/github.com/roberson-io/cchef/ops)
for its side effects registers every operation.

```bash
go get github.com/roberson-io/cchef
```

```go
import (
    "github.com/roberson-io/cchef/core"
    _ "github.com/roberson-io/cchef/ops" // register the operations
)

r, err := core.ParseRecipeConfig(`[{"op":"To Base64"}]`)
out, err := r.Execute(core.NewDish([]byte("hello"), core.TypeByteArray))
// out.String() == "aGVsbG8="
```

An operation can also be named directly, which the compiler checks where a
lookup by name cannot:

```go
op := ops.ToBase64{}
out, err := op.Run(core.NewDish([]byte("hello"), core.TypeByteArray),
    core.DefaultArgs(op.Args()))
```

Implement `core.Operation` and pass it to `core.Register` to add an operation of
your own; it is then usable by name in any recipe, alongside the built-in ones.

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
