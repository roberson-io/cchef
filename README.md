# cchef

[![CI](https://github.com/roberson-io/cchef/actions/workflows/ci.yml/badge.svg)](https://github.com/roberson-io/cchef/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**CyberChef on the command line.** `cchef` is a Go port of the data-transformation
engine of [CyberChef](https://gchq.github.io/CyberChef/) — the "Cyber Swiss Army
Knife" — built for the terminal. Every operation is a subcommand that reads input
and writes output, so operations chain together through Unix pipes or as a single
recipe, and any recipe can be turned into a shareable CyberChef URL.

> **Status:** a curated, growing subset — **445 operations** so far, each a
> faithful, test-driven port. See [PLAN.md](PLAN.md) for the full implementation
> status against all 495 CyberChef operations (CyberChef's category config names
> 498, but three have no implementation).

## Install

```bash
make build      # produces ./dist/cchef
```

Requires Go 1.26+.

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

The 445 operations are grouped using the same categories as CyberChef. Each page
documents options, examples, and reference links:

- [Arithmetic / Logic](docs/arithmetic-logic.md) — Sum, Subtract, Multiply,
  Divide, Mean, Median, Standard Deviation, and set operations (Union,
  Intersection, Difference, Symmetric Difference, Cartesian Product, Power Set).
- [Data format](docs/data-format.md) — Base/Base32/45/58/62/64/85/92, Hex, PEM, Octal,
  Modhex, BCD, Float, Hexdump, Caret/M-decode, Text-Integer, URL encode/decode, AMF.
- [Encryption / Encoding](docs/encryption-encoding.md) — ADD, AND, Bit shift left/right, NOT, OR, SUB, XOR, ROT13, ROT47.
- [Hashing](docs/hashing.md) — MD5, SHA-1, SHA-2 (224/256/384/512), SHA-3, Keccak,
  HMAC, Adler-32.
- [Utils](docs/utils.md) — Reverse, To Upper/Lower case.

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
make test     # run all unit tests
make vet      # go vet
make lint     # golangci-lint (make install-tools to install it)
make fmt      # gofmt
make sec      # gosec SAST + govulncheck (dependency & stdlib CVEs)
make sbom-audit   # generate + scan a CycloneDX SBOM
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
byte-for-byte parity. The repeatable workflow for porting a new operation is
captured in the [`/add-operation`](.claude/skills/add-operation/SKILL.md) skill.

### Dependencies

Kept deliberately small:

- [`spf13/cobra`](https://github.com/spf13/cobra) — CLI framework.
- [`elobuff/goamf`](https://github.com/elobuff/goamf) — backs the AMF operations.
- [`golang.org/x/crypto`](https://pkg.go.dev/golang.org/x/crypto) — legacy-Keccak
  hashers for Keccak-256/512.
- [`ProtonMail/go-crypto`](https://github.com/ProtonMail/go-crypto) — OpenPGP
  library backing the PGP operations (interoperable with CyberChef's kbpgp).

The full dependency rationale, including transitive dependencies, is in
[PLAN.md](PLAN.md).

## License

Released under the [MIT License](LICENSE). `cchef` is an independent
reimplementation; the upstream CyberChef project it ports is licensed
Apache-2.0 (see Credits).

## Credits

`cchef` is an independent port of [CyberChef](https://github.com/gchq/CyberChef)
by GCHQ (Crown Copyright, Apache-2.0). All operation semantics and test vectors
derive from that project.

The Add Text To Image operation embeds the four 72px Roboto bitmap-font atlases
CyberChef bundles (`internal/ops/bmfonts/`), taken unmodified from that project.
They are generated from the [Roboto](https://github.com/googlefonts/roboto)
typeface by Christian Robertson, licensed Apache-2.0. Using the same atlases is
what makes cchef's rendered text pixel-identical to CyberChef's.
