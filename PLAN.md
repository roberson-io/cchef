# cchef — Plan

`cchef` is a Go CLI port of the data-transformation engine of
[CyberChef](https://gchq.github.io/CyberChef/) (cloned alongside at
`../CyberChef`). This file tracks what is left to do; the README describes
what the tool is, and `docs/` documents every operation.

## Status

A **curated set of 505 operations** is implemented, tested and documented,
tracked against CyberChef 11.3.0. Those subcommands cover 501 unique CyberChef
operations; the difference is `SHA2`, which cchef also exposes as the
no-argument `sha224`/`sha256`/`sha384`/`sha512`.

- **505 operations** (`ops/`), each a faithful port.
- **Core engine** (`core/`), CLI (`cmd/`), docs (`docs/`), and
  generators for the large tables (`tools/`).
- `make all` (fmt, fix, vet, test, build, lint, sec) and `make fuzz` are clean.

## Constraints

These shape every item below.

- **A single static binary, no cgo.** This rules out otherwise obvious
  libraries and is why so much is implemented from scratch. OCR is the one
  exception: it drives an installed `tesseract` and errors clearly without it.
- **Operations match CyberChef**, verified against upstream fixtures or a
  running CyberChef. Where cchef differs on purpose it is because CyberChef has
  a defect; those are cataloged in `../CYBERCHEF-BUGS.md`. Differences that are
  *not* bug fixes are listed under
  [Deliberate differences](#deliberate-differences-from-cyberchef).
- **Test-driven**: test → red stub → implementation, per the
  [`/add-operation`](.claude/skills/add-operation/SKILL.md) skill.
- **Following CyberChef is a choice, not an obligation.** Tracking upstream is
  what makes the oracle useful, but the maintainer decides what cchef contains.
  An operation CyberChef removes may stay, and an upstream change may be
  declined. Semantic versioning is likewise a convention to weigh, not a rule
  that overrides that judgment.
- **Dependencies are a liability.** `golang.org/x/*` and libraries with
  large-organization backing are fine long-term; modules maintained by
  individuals, especially unmaintained ones, are candidates for in-repo
  reimplementation. Nothing GPL or AGPL: the project is Apache-2.0.

## Remaining work

Ordered so that changes to behavior and the CLI surface land before the
verification, documentation and packaging that describe them. Reorder freely —
only the ordering *within* a stage reflects a real dependency.

### 1. Behavior and CLI surface

- [x] **Take secrets off the command line.** Every string-valued argument flag
  now has a `--<flag>-file` companion (`--key-file`, `--passphrase-file`, …)
  that reads the value from a file, registered uniformly in the flag builder
  rather than per operation. The inline flags stay — deliberately less rigid
  than clig.dev — so nothing breaks and non-secret values keep their
  ergonomics; naming both variants is an error, and one trailing newline is
  stripped so ordinary text files work as-is. No escape syntax means a literal
  `@notes.txt` passphrase needs no special casing. Recipes, `bake` and the
  CyberChef URL feature are untouched: a recipe embeds argument values
  literally (a share URL must carry them to work in the browser), which the
  docs now call out so secrets go in file flags, not shared recipes or URLs.
- [ ] **Decide on the V8 integer-key ordering.** JSON output enumerates
  integer-like keys first, ascending, because V8 does. Unlike the rest of the
  JavaScript parity kept above, this one is a genuine wart — JSON object order
  carries no meaning, and no consumer should depend on it. It is confined to a
  single file (`jsWriteObject`), so the decision is cheap either way; it was
  split out of the parity item rather than settled with it.

### 2. Verification

- [ ] **Cross-check against standardized test vectors.** For AES, DES, the SHA
  family, HMAC, RSA and ECDSA, check against NIST's
  [CAVP](https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program)
  vectors. Being independent of CyberChef, they catch anywhere CyberChef itself
  diverges from the standard, and confirm the places cchef matches it
  deliberately. They validate the algorithm, not the option plumbing, so the
  oracle checks stay.

### 3. Release

- [ ] **GoReleaser.** A no-cgo static binary is its ideal case: macOS, Linux and
  Windows on amd64/arm64 from one config, with archives, checksums, a Homebrew
  tap and nfpm-built deb/rpm once the repository is public. Document `tesseract`
  as the one optional runtime dependency on each platform.
- [ ] **Adopt the versioning scheme and record the aligned CyberChef version.**
  cchef's version tracks CyberChef's, one component down:

  | CyberChef | cchef | Meaning |
  | --- | --- | --- |
  | v11.3.0 | **v1.0.0** | The first public tag |
  | v11.4.0 | v1.1.0 | A CyberChef minor release |
  | v12.0.0 | v2.0.0 | A CyberChef major release |
  | v11.3.1, or nothing | v1.0.1 | A CyberChef patch, or a cchef-only fix |

  So the whole v1.0.x line stays aligned with CyberChef v11.3.0. Patch is
  cchef's own and is used freely; minor and major follow CyberChef even when
  the release brings cchef no change of its own.

  Record the aligned CyberChef version as a constant beside `version` in
  `cmd/version.go`, report it from `cchef --version`, and pin it with a test so
  it cannot drift from what the port was built against. Give the docs and this
  file a single source to quote — several places currently spell out "11.3.0"
  by hand.
- [ ] **Add a CHANGELOG.** Keep-a-Changelog format, one section per release,
  each naming the CyberChef version it aligns with. The v1.0.0 entry is just a
  baseline: initial release, aligned with CyberChef v11.3.0, no itemisation of
  what it contains. Entries after that record cchef-only fixes and what each
  CyberChef release brought in.
- [ ] **Sign and attest releases.** The SBOM already exists (`make sbom-audit`,
  CycloneDX via cyclonedx-gomod, scanned with grype); publish it as a release
  asset rather than only a CI artifact. Sign checksums and archives — Sigstore
  cosign keyless signing is the low-friction route for a public repo, and
  GoReleaser supports it directly — and publish build provenance (SLSA
  attestation via `actions/attest-build-provenance`). Document how to verify a
  download in the README, since an unverifiable signature helps nobody.
- [ ] **Man pages.** cobra can generate a full man tree (`cobra/doc`), but with
  505 subcommands that is a large tree: hand-write a quality `cchef(1)`
  following man-page best practice, then decide whether generated
  per-subcommand pages earn their bulk.

## Verification

How to check the work above.

- **`make all`** — fmt, fix, vet, test, build, lint, sec. Plus `make complexity`
  and `make fuzz`.
- **Both architectures.** CI runs linux/amd64 while development is on arm64, and
  Go's `math` functions differ in the last bit between them. Anything with
  floating-point maths is checked on both:

  ```bash
  docker run --rm --platform linux/amd64 -v "$PWD":/src -w /src \
      -e GOFLAGS=-buildvcs=false -e GOGC=off golang:1.26 go test ./...
  ```

- **The oracle.** For operations without upstream fixtures, a running CyberChef
  gives authoritative output. Two are used: the HTTP server (Docker, from
  `../CyberChef-server`, `POST /bake`), and the Node API for what the server
  cannot do — operations whose `run()` is async or loads WASM, and `List<File>`
  output. Keep the image current: it installs the last *published* npm
  `cyberchef`, so an operation merged after that release is in the source
  checkout but absent from the oracle. The server also refuses empty input for
  every operation, so those cases are covered by fixtures instead. Flow control
  is excluded from both oracles by design.
- **Differential sweeps.** For anything with formatting, numeric or
  binary-layout subtleties, fixtures are not enough: sweep many varied inputs
  through cchef and the oracle and compare byte for byte. Where cchef diverges
  deliberately, the sweep asserts the divergence precisely rather than skipping
  the case.
- **Fuzzing.** `make fuzz` runs seven targets across `core`,
  `ops` and `internal/yara`: the parsers that read data cchef did not
  write, plus round-trip properties over the byte-level codecs. Failing inputs
  land in `testdata/fuzz/` and become regression tests.

## Deliberate differences from CyberChef

Choices, not bug fixes — the bug fixes are in `../CYBERCHEF-BUGS.md`. Add to
this list when a relaxation from stage 1 lands.

- **An option value is matched whatever its case, and normalised.** CyberChef
  validates an `option` argument case-insensitively and then hands the
  operation the string the caller wrote, so `"comma"` is accepted and silently
  selects the default — logged as a bug. cchef accepts the same casings, so
  anything that works upstream works here, but resolves the value to the
  declared spelling first, so `"comma"` means `Comma`.

  An exact match always wins, because `To Morse Code` offers `Dash/Dot`,
  `DASH/DOT` and `dash/dot` and there the casing *is* the setting; it is the
  only argument in the 505 operations whose choices differ only by case. A
  value matching several choices only by case — `"Dash/DOT"` — is refused,
  where CyberChef renders a casing that was never on offer.

- **Errors are errors.** CyberChef catches an `OperationError` and renders its
  message as the recipe's output; cchef returns it, so a shell sees a non-zero
  exit and the message on stderr.
- **No browser, so no HTML previews.** The byte-emitting operations (Render
  Image, Play Media, Render PDF) validate their input and pass the bytes
  through. Presentation is an IO-layer concern rather than an operation
  argument, so two global flags serve every operation that emits bytes:
  `--preview` renders an image inline (iTerm2/WezTerm or kitty) and `--data-uri`
  writes a `data:<mime>;base64,…` URI. On the same principle, Syntax
  highlighter's hljs-class HTML is rendered as ANSI color when the output is
  going to a terminal, under `--ansi`. Report-style operations (Magic, YARA
  Rules, ELF Info) print text instead of a table, and the `List<File>` ones
  write into `--out-dir`.
- **Imaging reproduces Jimp's pixel maths from scratch** on an `image.NRGBA`.
  Lossless formats are pixel-identical; JPEG output is a lossy re-encode, and
  Rotate by non-multiples of 90° matches dimensions rather than every pixel.
  Text rendering uses CyberChef's own bitmap font atlases, so Add Text To Image
  is pixel-identical.
- **Charts emit standalone SVG.** Byte-identical to goldens produced under Node
  except for three mechanical deviations that make the result valid SVG:
  `__data__` bindings are not serialized, the clip element is spelled
  `clipPath`, and the root carries an `xmlns`. Two portability details are
  pinned deliberately: hexagon angles are tabulated because Go and V8 disagree
  by an ulp on `sin`/`cos`, and the interpolation sites force intermediate
  rounding because Go on arm64 fuses multiply-add where JavaScript rounds twice.
- **PGP** runs on `ProtonMail/go-crypto` rather than reproducing kbpgp's exact
  output. Armor headers and generated-key subkey structure differ;
  interoperability is complete in both directions and is what the tests pin.
- **User-supplied regular expressions are Go's RE2.** Lookahead and
  backreferences are unavailable; `regexp2` is reserved for internal patterns
  that need them.
- **An Avro `long` is read as a 64-bit integer.** Avro defines `long` as signed
  64-bit; avsc decodes it through a float64, so CyberChef refuses a magnitude
  above 9007199254740990 and silently returns the wrong sign below -2^52 —
  logged as a bug. cchef reads the whole range exactly, so it accepts files
  CyberChef refuses and disagrees with it on that band. Copying either failure
  would mean returning a number that is not the one in the file.

- **AMF Decode returns the value, not a parse tree.** CyberChef hands back the
  object model of the npm package it wraps, annotated with markers, lengths,
  traits and class names: an AMF0 double of 42 comes out as
  `{"marker":0,"$value":42}`, and a small object as a nest of `properties`
  entries. cchef returns `42` and `{"a":1}`. The annotation is an artifact of
  that library rather than anything the format requires — it would change if
  upstream swapped packages — and plain JSON pipes into `jq` and lets
  encode/decode round-trip, which the tests pin. Encoding is unaffected and
  stays byte-identical.
- **Whole-number arguments are enforced.** CyberChef marks only 14 of its 220
  numeric arguments `integer`, so the rest silently truncate a fractional value:
  `Bit shift right` with an amount of 1.5 runs. cchef declares `Integer` on 138,
  refusing the value with `Amount must be an integer.` A typo on a command line
  should fail rather than quietly produce a different answer, so a shared URL
  carrying a fractional argument errors here instead of running. All 14 that
  CyberChef marks are among them.

  cchef also caps twenty parameters CyberChef leaves open, in three groups.
  **Ten bound the cost of a password-based key derivation**, where an open
  parameter turns a typo into an allocation or a run that does not finish:
  `Argon2` (Iterations ≤ 4096, Memory ≤ 2 GiB, Parallelism ≤ 255, Hash length ≤
  4096), `Bcrypt` (Rounds 4–31, which bcryptjs silently clamps to instead),
  `Derive PBKDF2 key` and `Derive EVP key` (Key size ≤ 8192 bits, Iterations ≤
  10,000,000) and `Scrypt` (Key length ≤ 4096). **Three bound a hash round
  count** at the length of its constant table, past which CyberChef reads an
  undefined entry and returns a digest built partly from `NaN`: `SHA1` (Rounds ≤
  80) and `SHA2` (≤ 64 for the 256 family, ≤ 160 for the 512 family). **Seven
  are older**: `AES Decrypt` (IV Length ≥ 0), `Generate Image` (Pixel Scale
  Factor ≤ 64, Pixels per row ≤ 2048), `Pseudo-Random Integer Generator` (Min
  and Max Value to ±2^53−1), `To Hexdump` (Width ≤ 65536) and `Wrap` (Line Width
  ≤ 65536).
