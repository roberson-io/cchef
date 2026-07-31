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

- **505 operations** (`internal/ops/`), each a faithful port.
- **Core engine** (`internal/core/`), CLI (`cmd/`), docs (`docs/`), and
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

- [x] **Move Syntax highlighter's output format off the operation.** Its
  cchef-added `Output format` (`HTML`/`Terminal`) argument is gone, so the
  operation now takes CyberChef's single `Language` argument and a generated
  share URL carries nothing CyberChef cannot read. The terminal rendering moved
  to a global `--ansi` flag (`auto`/`always`/`never`, honoring `NO_COLOR`), which
  converts the hljs spans to ANSI at the IO layer. Named `--ansi` rather than the
  conventional `--color` because operation arguments keep CyberChef's spellings
  and two operations have a `Colour` argument, which already answers to
  `--color`; a global flag of that name would have taken it from them.
- [x] **Close the SHA1 and SHA2 argument gap.** `SHA1` now takes CyberChef's
  `Rounds` argument, and a new `SHA2` operation carries the `Size` selector over
  all six digest sizes plus the round count per family, so `SHA2('512/256',…)`
  in a recipe or share URL works in both directions. The no-argument
  `sha224`/`sha256`/`sha384`/`sha512` subcommands stay as the ergonomic way to
  ask, and a test pins them to `SHA2` at its defaults so the two cannot drift.
  Go's `crypto/sha1` and `crypto/sha256` fix the round count, so the compression
  functions are in-repo: SHA-1 reuses the SHA-0 core (the two differ by one
  rotation) and SHA-2 is written out in `sha2.go`. The 512 family counts rounds
  in the half-steps CyberChef counts, so its default of 160 is the standard 80.
- [x] **Match every bound CyberChef declares.** `ArgDef` carries
  `Min`/`Max`/`Integer`, enforced by `coerceNumber` before the operation runs.
  Of 218 numeric arguments, nine declared a bound upstream that cchef did not
  (`RC6 Encrypt`/`RC6 Decrypt` Word Size and Rounds, `XTEA Encrypt`/`XTEA
  Decrypt` Rounds, `Derive HKDF key` L, `Pseudo-Random Number Generator` Number
  of bytes, `To Binary` Byte Length) and two declared `integer` that cchef did
  not (`Generate Image` Pixel Scale Factor, `Wrap` Line Width). Every message
  was checked against the oracle. Declaring the bounds made three internal
  checks unreachable, which were removed. Where cchef stays stricter than
  upstream it is recorded under
  [Deliberate differences](#deliberate-differences-from-cyberchef), and the
  upstream defect is logged in `../CYBERCHEF-BUGS.md`.
- [x] **Bound the parameters that size work or memory.** Ten arguments across
  the password-based key derivations now declare a maximum: `Argon2` (Memory,
  Iterations, Parallelism, Hash length), `Bcrypt` (Rounds), `Derive PBKDF2 key`
  and `Derive EVP key` (Key size, Iterations), and `Scrypt` (Key length).
  `argon2 --memory-kib=50000000` no longer allocates until it is killed. Each
  limit sits far above published guidance, so no real use is refused, and each
  is recorded under
  [Deliberate differences](#deliberate-differences-from-cyberchef).

  Two defects turned up in cchef itself. The lane count reaches the backend as
  a `uint8`, so `--parallelism 256` wrapped to zero and panicked, and
  `--parallelism 260` wrapped to four — hashing at a different cost while
  labelling the output `p=260`. Both are refused now. The bounds are pinned by
  the operations' own tests rather than `TestNoAllocationBombs`, which drives
  input rather than arguments and would have to run the bomb to observe it.

  The other ~136 unbounded numeric arguments are left open: they size nothing.
  Four are worth a look if this comes up again — `Generate De Bruijn Sequence`
  (Alphabet size × Key length is an exponential output), `Generate Lorem Ipsum`
  (Length), `Sleep` (Time (ms)) and the inflate operations' `Initial output
  buffer size`.
- [ ] **Make the CyberChef base URL configurable.** `cyberChefBaseURL` in
  `internal/core/url.go` hardcodes `https://gchq.github.io/CyberChef/`, so
  `cchef url` cannot point at a self-hosted or air-gapped instance — a normal
  arrangement for the sort of work this tool is used for. Keep the GCHQ URL as
  the default and allow both an environment variable and a config file, with
  precedence flag > environment > config file > default. Decide the config
  file's location and format alongside it (`$XDG_CONFIG_HOME/cchef/`, and
  whether anything else belongs in it), since this is the first thing to need
  one. `recipe convert` and `bake` are unaffected; only URL generation is.
- [ ] **Revisit [clig.dev](https://clig.dev/) end to end.** It shaped the
  interface early in development and has not been re-checked since. Do this
  before the docs pass, since it may produce more surface changes.
- [ ] **Decide how far to loosen JavaScript parity.** Strict parity was
  scaffolding: it made the oracle and the sweeps possible. Some of what it
  preserved is JavaScript rather than CyberChef — `internal/jsnum` exists solely
  to reproduce JS float formatting, and JSON output enumerates integer keys
  first because V8 does. Proposed tiering: keep byte parity where interop is the
  point (data transforms feeding other tools, anything reachable from a shared
  CyberChef URL), and allow Go-native behavior where the quirk is a wart nobody
  depends on. Each relaxation becomes a deliberate difference, recorded below,
  with the sweeps asserting the divergence precisely. The cost to watch: every
  divergence makes the oracle a little less useful.

### 2. Verification

- [ ] **Build a real-input corpus.** Test the built tool against real inputs,
  not only contrived fixtures: actual ELF and PE binaries, photographs,
  archives, documents, through the relevant operations and against the oracle.
  Record every mismatch — a corpus that drops mismatches hides bugs.
- [ ] **Cross-check against standardized test vectors.** For AES, DES, the SHA
  family, HMAC, RSA and ECDSA, check against NIST's
  [CAVP](https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program)
  vectors. Being independent of CyberChef, they catch anywhere CyberChef itself
  diverges from the standard, and confirm the places cchef matches it
  deliberately. They validate the algorithm, not the option plumbing, so the
  oracle checks stay.
- [ ] **`Hex to PEM` — mixed hex/non-hex input.** Faithful for well-formed hex
  and lenient on stray characters as upstream is, but input that *interleaves*
  hex and non-hex (`"3g3"`) is not reproduced: jsrsasign routes through
  CryptoJS's word packing and fractional-`sigBytes` clamp. Closing it means
  porting that pipeline. Low priority — garbage in, garbage out.
- [ ] **`Avro to JSON` — 64-bit longs above 2^53.** cchef reads `long` as exact
  `int64`; avsc reads into a float64. Unreachable for avsc-produced files, since
  its encoder rejects such values. Low priority.

### 3. Dependencies

Each is removed once its Go replacement is oracle-verified over the same inputs.

- [ ] **`github.com/elobuff/goamf`** (AMF Encode/Decode) — highest priority:
  unmaintained since 2014, and removing it drops an indirect dependency too.
  Fuzzing found two defects in it: it indexes into its buffers without checking
  the length, so truncated input panicked (`amfDecode` now contains that and
  reports an error, where CyberChef decodes the same bytes into a partial
  structure), and it spends about a second and a large allocation on a
  four-byte length prefix. A reimplementation should bound both.
- [ ] **`github.com/sergi/go-diff`** (Diff) — reimplement the Myers diff.
- [ ] **`github.com/mmcloughlin/geohash`** (Convert co-ordinate format) —
  bit-interleaving; small.
- [ ] **`github.com/wroge/wgs84`** (Convert co-ordinate format) — the remaining
  datum transforms; precision-sensitive, verify against the oracle.
- [ ] **`golang.org/x/text/encoding/charmap`** (MIME Decoding) — route through
  the in-repo codepage engine, which already covers all 16 ISO-8859 charsets.
  This removes the *usage*, not the `x/text` module, which stays for
  `unicode/norm`.

Explicitly kept: `dlclark/regexp2` (backtracking PCRE, which RE2 cannot
replace), `google.golang.org/protobuf` + `bufbuild/protocompile` (a full
`.proto` compiler), `golang.org/x/text/unicode/norm`, `golang.org/x/crypto`,
`go.yaml.in/yaml/v3`.

### 4. Structure

- [ ] **Decide whether cchef is importable as a library.** Everything but
  `cmd/` lives under `internal/`, so no other Go program can use the engine
  today — a caller wanting to bake a recipe has to shell out.

  The smallest thing that would work: export the `Dish` type hub,
  `Operation`/`ArgDef`, the registry and `Recipe.Execute` — today's
  `internal/core` — plus a blank import that pulls in the operations for their
  registrations, so a caller can bake a recipe and register operations of its
  own.

  Start with the operation types internal, because the direction is cheap to
  reverse: internal to exported is additive and breaks nobody, exported to
  internal breaks every importer. What is known is that `cmd/` — the only
  consumer so far, and a demanding one — never names a concrete operation
  type: a blank import registers them and `core.Default.Get`/`All` does the
  rest, so 505 subcommands, `list`, `bake` and the staging commands are all
  built without one. What is not known is whether a caller would want a
  compile-time reference to a specific operation (`ops.ToBase64{}` type-checks
  where `Get("To Base64")` cannot). Export them when someone asks and can say
  what for. Same reasoning for the engines (`yara`, `jsnum`, `termimage`).

  Settle this **before** the split below, since the split decides package
  boundaries and it would be wasteful to draw them twice. Then add a
  [pkg.go.dev](https://pkg.go.dev) badge to the README and write package-level
  doc comments for whatever becomes public.
- [ ] **Split `internal/ops`.** It is one flat package of ~782 files / 165k LOC
  implementing the 505 operations. Nothing is broken; the concern is
  navigability and build granularity. Re-measure before acting — these figures
  drift. What the measurements showed:

  - The operation files are healthy: ~315 declare a `Meta()`, most registering
    one operation. One operation per file mirrors CyberChef and keeps the test
    loop tight — **keep it**.
  - ~99 files register nothing. These are the engines and static data — the
    JavaScript parser, the entity and codepage tables, the XML DOM, protobuf,
    d3, the disassemblers, the Bletchley machines — interleaved alphabetically
    with forty-line operations.
  - Coupling is low, but a few shared helpers (`convertToByteArray`, `charRep`,
    `parseEscapedChars`, `imageTransform`) hold the reference graph together.
    Extracting them breaks it into many small components.

  Three problems worth fixing: one compilation unit and one test binary, so a
  one-line edit recompiles everything and every unexported helper is visible to
  every file; category is a side table (`opCategories` hand-maintains 505
  entries) rather than structure; and engines carry the same visual weight as
  operations in a directory listing.

  Staged approach: extract the shared helpers into `internal/opsutil` and the
  standalone engines into their own packages (worthwhile on its own), then split
  the remainder one package per category, then generate the category table from
  the result. Do it when nothing else is in flight — it is large mechanical
  churn. It is also the moment to audit file-splitting consistency: reciprocal
  operations normally share one file per algorithm, so any operation spread
  across several files should be re-justified or merged.

### 5. Documentation

- [ ] **Name the actual flag in every options table.** Pages disagree on what
  the first column holds: `date-time.md` and eight others give the flag
  (`` `--days` ``), while `code-tidy.md`, `compression.md`, `extractors.md`,
  `flow-control.md`, `forensics.md`, `language.md` and `multimedia.md` give
  CyberChef's display name, leaving the reader to infer `--colour-pattern-1`
  from an example. Roughly 276 of 1,082 option rows are affected, plus a few
  stragglers in `arithmetic-logic.md` and `encryption-encoding.md`. Convert
  them to the flag, which is the majority convention and the useful one; a
  derived name is not always guessable from the label, so check each against
  `cchef <op> --help` rather than deriving it by hand. Worth doing in the same
  pass as the two items below, since it touches the same pages.
- [ ] **Convert `docs/` to US English**, except where it quotes an operation,
  flag or option value whose original spelling is UK English (`Split Colour
  Channels` stays as CyberChef named it). Not a blind find-and-replace: the
  examples embed verified command output and functional option values that must
  keep their exact spellings.
- [ ] **Cross-reference the classic tools.** Many operations do the same job as
  a well-known CLI tool: the Base64/Base32 codecs (`base64`/`base32`), hex and
  hexdump (`xxd`, `od`), the hash subcommands (`md5sum`/`sha256sum`, `openssl
  dgst`), Strings (`strings`), the compression codecs (`gzip`, `bzip2`, `zip`),
  text-encoding conversion (`iconv`), UUID generation (`uuidgen`), jq/JPath
  (`jq`), Diff (`diff`). Each such page should say it is an alternative to that
  tool, link to it, and note where cchef's behavior differs because parity is
  with CyberChef.
- [ ] **Clean up process talk in comments and docs.** Provenance claims
  ("faithful", "byte-for-byte"), descriptions of upstream's behavior, and
  development narrative belong nowhere; state what the code does and the
  constraints a reader needs. Do this opportunistically as files are touched
  rather than as a dedicated audit, and hold new code to the tighter standard so
  the debt stops growing.

### 6. Release

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
- **Fuzzing.** `make fuzz` runs seven targets across `internal/core`,
  `internal/ops` and `internal/yara`: the parsers that read data cchef did not
  write, plus round-trip properties over the byte-level codecs. Failing inputs
  land in `testdata/fuzz/` and become regression tests.

## Deliberate differences from CyberChef

Choices, not bug fixes — the bug fixes are in `../CYBERCHEF-BUGS.md`. Add to
this list when a relaxation from stage 1 lands.

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
- **Whole-number arguments are enforced.** CyberChef marks only 14 of its 220
  numeric arguments `integer`, so the rest silently truncate a fractional value:
  `Bit shift right` with an amount of 1.5 runs. cchef declares `Integer` on 138,
  refusing the value with `Amount must be an integer.` A typo on a command line
  should fail rather than quietly produce a different answer, so a shared URL
  carrying a fractional argument errors here instead of running. All 14 that
  CyberChef marks are among them, and cchef additionally caps seven parameters
  CyberChef leaves open. Ten of them bound the cost of a password-based key
  derivation, where an open parameter turns a typo into an allocation or a run
  that does not finish: `Argon2` (Iterations ≤ 4096, Memory ≤ 2 GiB,
  Parallelism ≤ 255, Hash length ≤ 4096), `Bcrypt` (Rounds 4–31, which bcryptjs
  silently clamps to instead), `Derive PBKDF2 key` and `Derive EVP key` (Key
  size ≤ 8192 bits, Iterations ≤ 10,000,000) and `Scrypt` (Key length ≤ 4096).
  The other seven are older: `AES Decrypt` (IV
  Length ≥ 0), `Generate Image` (Pixel Scale Factor ≤ 64, Pixels per row ≤
  2048), `Pseudo-Random Integer Generator` (Min and Max Value to ±2^53−1), `To
  Hexdump` (Width ≤ 65536) and `Wrap` (Line Width ≤ 65536).
