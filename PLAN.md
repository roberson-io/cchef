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
- [x] **Make the CyberChef base URL configurable.** `cchef url` points at a
  self-hosted or air-gapped instance through `--base-url`, `$CCHEF_BASE_URL` or
  `base-url` in the config file, in that order of precedence, falling back to
  `core.DefaultBaseURL`. Anything that is not an `http`/`https` URL is refused,
  naming whichever of the three supplied it. `bake` and `recipe convert` are
  untouched.

  The config file settled as YAML at `$XDG_CONFIG_HOME/cchef/config.yaml`,
  `$XDG_CONFIG_HOME` defaulting to `~/.config`, with `$CCHEF_CONFIG` naming the
  file outright. YAML because it is already a dependency and takes comments;
  the XDG layout on every platform so the path is the same one everywhere.
  Having no file is the normal case; one that exists but will not parse is an
  error naming it. `base-url` is the only key, and the rule for adding more is
  that a setting belongs to the machine, never to a recipe — a recipe has to
  mean the same thing wherever it is run.
- [x] **Revisit [clig.dev](https://clig.dev/) end to end.** Audited against the
  whole checklist. Most of it already held: exit codes, stdout for results and
  stderr for everything else, `-h`/`--help`/`help`, help on no arguments,
  examples and a support link in the help, typo suggestions, `-` for
  stdin/stdout, XDG config with flag > environment > file precedence,
  `CCHEF_`-prefixed environment variables, a single static binary, and no
  telemetry of any kind. Three gaps were closed:

  - **cchef no longer hangs on an interactive terminal.** With no `-i`,
    `--in-file`, `--in-dir` or positional argument, it used to block on stdin
    with nothing on screen to say so. It now says what to do. The check is
    `term.IsTerminal` rather than the character-device test used before,
    because `/dev/null` is a character device and `< /dev/null` has to keep
    meaning empty input. `golang.org/x/term` was already in the module graph.
  - **`--ansi auto` stands down for `TERM=dumb`**, alongside `NO_COLOR`.
  - **`cchef list --json`** gives the listing as data — subcommand, operation
    name, summary, categories — for completions and wrappers.

  Left open, with reasons. The first has its own item below; the rest are
  judgement calls recorded here rather than scheduled.

  - **No pager for long output.** `cchef list` is 561 lines. `| less` works,
    and spawning a pager is a behaviour change worth deciding separately.
  - **No progress indication** for the operations that take real time (Argon2
    and Bcrypt at high cost, YARA over large input). Nothing writes anything
    until it finishes.
  - **No `-q`/`--quiet`.** There is no non-essential output to suppress: the
    result is the output. The one candidate is the `==> name <==` header under
    `--in-dir`.
- [x] **Decide how far to loosen JavaScript parity.** Decided: keep it, and
  handle relaxations one at a time when a concrete case calls for one, as
  deliberate differences with a named cause. The proposed tiering was to keep
  byte parity "wherever interop is the point, including anything reachable from
  a shared CyberChef URL" — which is every recipe, so the rule selected
  everything and could not decide a single case.

  Nor is there much to gain. `internal/jsnum` is 153 lines behind one function,
  and its output is not a wart: it prints `0.000001` where Go prints `1e-06`,
  and `123456789012345680000` where Go prints `1.2345678901234568e+20`. Sixteen
  operations depend on it, among them coordinate conversion, where interop
  genuinely matters, and all five d3 charts, whose byte-exact goldens a
  formatting change would invalidate wholesale. Keeping parity also stays
  reversible; shipping divergence in v1.0.0 and retracting it later is a
  breaking change twice over.

  Note that `jsRound` is not a formatting question at all: JavaScript rounds
  −0.5 to −0 where Go rounds to −1, so relaxing it would change answers.
- [ ] **Take secrets off the command line.** `--key`, `--passphrase` and their
  kin are read as flag values, so they land in shell history and are visible in
  `ps` to anyone on the machine. clig.dev is unambiguous that a CLI should never
  do this, and it is the one real violation the audit found. Fixing it needs a
  general convention for reading an argument from a file or the environment —
  every operation that takes a key is affected, so a per-operation change is the
  wrong shape. Whatever the convention is, it has to leave a literal value that
  happens to look like the escape (`@notes.txt` as an actual passphrase) still
  expressible.
- [ ] **Decide on the V8 integer-key ordering.** JSON output enumerates
  integer-like keys first, ascending, because V8 does. Unlike the rest of the
  JavaScript parity kept above, this one is a genuine wart — JSON object order
  carries no meaning, and no consumer should depend on it. It is confined to a
  single file (`jsWriteObject`), so the decision is cheap either way; it was
  split out of the parity item rather than settled with it.

### 2. Verification

- [x] **Build a real-input corpus.** Built at `~/repos/cchef-corpus`,
  deliberately outside this repository and uncommitted: 398 files from a dozen
  unrelated producers — the PNG conformance suite, ExifTool's camera samples,
  Debian's GCC-built ELF, signed Windows PE from 7-Zip and CPython, GNU tar and
  gzip output, PDFs from three producers. Generated inputs were discarded on
  purpose; a binary from `go build` has no packing, overlay or signature, and an
  archive from the local `tar` is one implementation of the format.

  First full sweep: **6,726 checks over 398 files, no failures** — no mismatch,
  panic, timeout or hang. 2,262 of those are byte-for-byte agreements with
  CyberChef 11.3.0 across six operations, covering the whole PNG suite and
  camera-sample set. Base64, hex and gzip round-trip exactly on every file.
  Written up in the corpus under `results/`.

  What comes back into this repository is a fix and a regression test for
  anything the corpus finds. Nothing yet, because nothing has failed.
- [ ] **Compare the large files against the Node oracle.** The CyberChef server
  takes a 100 KB JSON body and the input travels as hex, so it will not compare
  anything over ~50 KB. That leaves 21 corpus files with no differential at all,
  and they are the ones with the most structure: the archives, the PDFs, the
  Debian and Go ELF binaries, and nine Windows PE files including OpenSSL,
  SQLite and the MSVC runtimes. The Node API oracle does not go over HTTP and
  has no such limit.

  Two narrower gaps to close in the same pass: only six operations are compared
  and all are whole-input transforms, so nothing structural (`elf-info`,
  `extract-exif`, `unzip`) is checked against CyberChef at all; and the corpus
  has no Office documents, whose authoring-tool metadata is worth exercising.
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

- [x] **`github.com/elobuff/goamf`** (AMF Encode/Decode) — replaced by an
  in-repo AMF0 and AMF3 codec (`internal/ops/amfcodec.go`). It and the
  `jcoene/gologger` it dragged in are both gone.

  The case for removing it turned out to be stronger than the age and the two
  fuzzing defects. goamf could not read back its own output in six ordinary
  cases — an empty string, an empty array, or an empty object, in either format
  — and an empty AMF0 strict array decoded to `null` with no error at all.
  Encoding was also nondeterministic: object keys came from Go map iteration,
  so the same input produced different bytes run to run and matched CyberChef
  only by chance. Reading the JSON with an ordered decoder fixed that; the
  encoding now follows the order the input gave, as CyberChef does, and is
  byte-identical to it.

  The reader checks its length before every read, so the `recover` that used to
  contain goamf's panics is gone; a test feeds every prefix of a valid encoding
  through both formats. Both upstream defects found along the way are logged in
  `../CYBERCHEF-BUGS.md`.
- [x] **`github.com/sergi/go-diff`** (Diff) — replaced by an in-repo port of
  jsdiff, the library CyberChef itself uses: the greedy Myers search with
  jsdiff's own pruning of diagonals that have reached the edge of the edit
  graph, plus its six tokenizers.

  Removing it closed the gap the wrapper left. go-diff is a port of Google's
  diff-match-patch, a different algorithm with different tie-breaking, so five
  of the six granularities had been approximated by encoding tokens as runes
  and diffing those, and Word with **Ignore whitespace** was documented as not
  matching. JSON mode was a plain line diff, missing the two things that make
  it a JSON diff — lines are compared with a trailing comma disregarded, and
  the longer of two such lines is the one kept. All six modes are now exact
  against the oracle over 72 cases.

  go-diff also imposed a one-second deadline, after which it returned a
  suboptimal diff; the same input could give two answers on two machines. The
  port has no deadline, which is what CyberChef does, so the result depends
  only on the input. The cost is set by how much the samples differ rather than
  by how large they are: 488 KB with forty changed lines takes under 0.1s in
  every mode, while two 32 KB samples of unrelated random text — the worst case
  the algorithm has — take 36s and 211 MB.
- [x] **`github.com/mmcloughlin/geohash`** (Convert co-ordinate format) —
  replaced by a port of ngeohash, the library CyberChef uses, in
  `internal/ops/coordinates.go`.

  This one was a divergence rather than only hygiene. ngeohash assigns a
  coordinate sitting exactly on a cell boundary to the *lower* cell; the Go
  library assigns it to the upper one. The origin encoded as `s0000000` where
  CyberChef gives `7zzzzzzz`, and every pole and quadrant boundary was wrong
  the same way. Worse, the centre of a decoded cell is itself an exact boundary
  at any finer precision, so **Geohash in and Geohash out disagreed on ordinary
  hashes** — `ezs42` came back as `ezs42000` instead of `ezs427zz`. Two smaller
  faults went with it: an upper-case hash decoded differently from its
  lower-case form, and a letter outside the geohash alphabet (`a`, `i`, `l`,
  `o`) was handled differently from ngeohash, which reads it as five zero bits.
  A differential probe over 277 inputs per seed found 25 disagreements before
  and none after.
- [ ] **`github.com/wroge/wgs84`** (Convert co-ordinate format) — the remaining
  datum transforms; precision-sensitive, verify against the oracle.
- [ ] **`golang.org/x/text/encoding/charmap`** (MIME Decoding) — route through
  the in-repo codepage engine, which already covers all 16 ISO-8859 charsets.
  This removes the *usage*, not the `x/text` module, which stays for
  `unicode/norm`.

  Worth doing on its own terms, because charmap has no ISO-8859-11 table and
  the codepage engine does (cp28601). `=?ISO-8859-11?Q?=A1=A2?=` decodes to
  Thai in CyberChef and returns `Unhandled Charset` in cchef, so this is a
  divergence to fix rather than only dependency hygiene. ISO-8859-12 was never
  standardised and errors on both sides, which is already correct. Dropping the
  package also takes 109 KB of duplicate charset tables out of the binary.

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
