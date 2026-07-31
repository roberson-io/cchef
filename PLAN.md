# CyberChef CLI (`cchef`) — Plan & Status

## Context

`cchef` is a Go CLI port of the data-transformation engine of
[CyberChef](https://gchq.github.io/CyberChef/) (cloned alongside at
`../CyberChef`). Each operation is a subcommand, operations chain into recipes
(pipes, JSON, or CyberChef's compact "Chef" format), and recipes round-trip to a
shareable `gchq.github.io/CyberChef` URL.

Two rules have shaped everything:

- **Test-driven, against CyberChef's own fixtures.** Test → red stub →
  implementation, with cases transcribed from
  `../CyberChef/tests/operations/tests/*.mjs`. Where an operation has no
  fixtures, behavior is derived from a running CyberChef (see
  [Verification](#verification)).
- **A single static binary, no cgo.** This rules out several otherwise obvious
  libraries and is why so much is implemented from scratch.

Where cchef's answer differs from CyberChef's on purpose, it is because
CyberChef has a defect. Those are cataloged with reproductions in
`../CYBERCHEF-BUGS.md` and noted in the relevant `docs/` page. See
[Deliberate differences](#deliberate-differences-from-cyberchef) for the
differences that are *not* bug fixes.

## Dependencies

**Policy.** `golang.org/x/*` and libraries with large-organization backing are
acceptable long-term. The concern is modules maintained by individuals —
especially unmaintained ones — which carry supply-chain and bit-rot risk. Those
are candidates for in-repo reimplementation, following the precedent set by the
codepage engine and the generated rule tables: reimplement rather than depend,
and differential-verify byte-for-byte against the oracle.

The README lists every direct dependency and what it is for. What follows is the
reasoning that is not obvious from that list.

**Written from scratch rather than pulled in.** Every cipher, hash, compression
format and file parser. Also: the **codepage engine** (a port of the `codepage`
npm library CyberChef wraps, byte-for-byte across all 152 charsets, its decode
tables extracted into an embedded ~1.2 MB blob by `tools/cpgen`); the
**JavaScript parser/beautifier/minifier**; the **d3 subset** behind the charts;
the **x86 disassembler**; the **Bletchley machines**; the **YARA engine** (no
usable pure-Go library exists, and the alternatives need a C or Rust library
linked in); and **GOST**, **Ascon**, **TEA/XTEA/XXTEA**, **PRESENT**,
**Twofish**, **RC6**, **SM4**, **Salsa20**, **SIGABA** and **Typex**.

**Notable carve-outs.**

- **PGP** (7 operations) runs on `ProtonMail/go-crypto` rather than reproducing
  kbpgp's exact output. Armor headers and generated-key subkey structure differ;
  interoperability is complete in both directions and is what the tests pin.
- **OCR** is the one operation cchef does not perform itself — it drives the
  installed `tesseract` binary, its only external tool. Every cgo-free Go route
  was examined and rejected: `gosseract` needs cgo; `gogosseract` is
  unmaintained and much slower; no Tesseract WASI build exists; no maintained
  pure-Go engine exists; and embedding the WASM plus language data would add
  ~13 MB. A clear error is returned when it is absent, and nothing else is
  affected.

### License problems (resolved)

Two dependencies were incompatible with distributing under Apache 2.0; both
have been replaced. The GPL-3.0 `im7mortal/UTM` module was replaced by
`internal/ops/utm.go`, a port of the MIT-licensed `utm` Python package. The
AGPL-3.0 ua-parser-js detection tables that had been transcribed into
`internal/ops/useragent_rules.go` were replaced by hand-derived rules built
from the structure of real user-agent strings and black-box probing of
CyberChef's observed output; parity was verified over a 187-string corpus and
against the live oracle.

### Other planned reimplementations

Each is removed once its Go replacement is oracle-verified over the same inputs.

- [ ] `github.com/elobuff/goamf` (**AMF Encode/Decode**) — highest priority:
  unmaintained since 2014, and removing it also drops an indirect dependency.
- [ ] `github.com/sergi/go-diff` (**Diff**) — reimplement the Myers diff.
- [ ] `github.com/mmcloughlin/geohash` (**Convert co-ordinate format**) —
  bit-interleaving; small.
- [ ] `github.com/wroge/wgs84` (**Convert co-ordinate format**) — the remaining
  datum transforms; precision-sensitive, verify against the oracle.
- [ ] `golang.org/x/text/encoding/charmap` (**MIME Decoding**) — route through
  the in-repo codepage engine, which already covers all 16 ISO-8859 charsets.
  This removes the *usage*, not the `x/text` module, which stays for
  `unicode/norm`.

**Explicitly kept:** `dlclark/regexp2` (backtracking PCRE, which RE2 cannot
replace), `google.golang.org/protobuf` + `bufbuild/protocompile` (a full `.proto`
compiler), `golang.org/x/text/unicode/norm`, `golang.org/x/crypto`, and
`go.yaml.in/yaml/v3`.

## Current status

The core engine, recipe/URL machinery, CLI, docs, and a **curated set of 504
operations** are implemented, tested and documented — every operation CyberChef
implements is ported, tracked against CyberChef 11.3.0. Those subcommands cover
501 unique CyberChef operations; the difference is `SHA2`, which cchef exposes
as `sha224`/`sha256`/`sha384`/`sha512`.

- **Core engine** (`internal/core/`): `Dish` (byte-backed type hub), the
  `Operation` interface with its `ArgDef`/`ToggleString` ingredient model, a
  self-registering `Registry`, `Recipe.Execute` with flow control, and faithful
  ports of `GeneratePrettyRecipe`/`ParseRecipeConfig` and
  `EncodeURIFragment`/`BuildURL`, each with byte-exact tests.
- **504 operations** (`internal/ops/`), each a faithful port with tests
  transcribed from CyberChef's fixtures.
- **CLI** (`cmd/`): auto-generated per-operation subcommands with flags derived
  from the argument definitions, plus `bake`, `url`, `recipe convert` and
  `list`. Input resolves `--in-file` > `-i/--input` > positional > stdin; output
  is byte-exact when piped and adds a trailing newline only on a terminal.
  `--in-dir`/`--out-dir` run a recipe over a directory.
- **Docs** (`docs/`): a page per category with options tables, simple and
  complex examples, and reference links.
- **Tooling**: `make all` runs fmt, fix, vet, test, build, lint and sec; plus
  `complexity` and `sbom-audit`. Every gosec suppression must name its rule and
  reason, enforced by the build.

## Architecture (as built)

```
cchef/
  main.go                    thin entry -> cmd.Execute()
  cmd/
    root.go                  root cobra command
    register_ops.go          one subcommand per registered op (flags from ArgDefs)
    io.go                    input resolution and output, --in-dir/--out-dir
    bake.go url.go recipe.go list.go
    opmeta.go                op -> category, plus the derived-summary machinery
    opsummaries.go           curated one-line summaries
    opaliases.go             short aliases for high-traffic ops
  internal/core/
    dish.go                  Dish + conversions between the type hub's forms
    operation.go             Operation, ArgDef, ArgType, ToggleString, coercion
    registry.go              Register / Get / All; ops self-register via init()
    recipe.go                RecipeOp, Recipe.Execute
    flow.go                  FlowState / FlowOperation, sub-recipe execution
    chef.go url.go naming.go
  internal/ops/              one file per operation, plus the shared engines
  internal/yara/             the YARA rule engine
  internal/jsnum/            JavaScript number formatting
  tools/                     generators for the large tables (see the READMEs)
  docs/                      one page per category
```

`internal/ops` is a single flat package: one file per operation alongside the
engines and static tables they share. See
[Proposed reorganization](#proposed-reorganization-of-internalops).

## Recipe formats and URLs

- **JSON**: `[{"op":"To Base64","args":["A-Za-z0-9+/="]}, ...]`, recognized by
  its leading `[`.
- **Chef** (compact): `To_Base64('A-Za-z0-9+/=')To_Hex('Space')`, with optional
  `/disabled` and `/breakpoint` flags.
- A recipe in either form that does not parse is refused, rather than silently
  doing nothing.
- **URL**: `https://gchq.github.io/CyberChef/#recipe=<chef>&input=<base64>`.

## Adding an operation

The repeatable workflow is the [`/add-operation`](.claude/skills/add-operation/SKILL.md)
skill: research upstream, write the test first from its fixtures, add a red
stub, port `run()` until green, cover the new code, then update the docs, the
category table and the counts.

## Deliberate differences from CyberChef

These are choices, not bug fixes — the bug fixes live in `../CYBERCHEF-BUGS.md`.

- **Errors are errors.** CyberChef catches an `OperationError` and renders its
  message as the recipe's output; cchef returns it, so a shell sees a non-zero
  exit and the message on stderr.
- **No browser, so no HTML previews.** CyberChef's `presentType: "html"`
  operations render in the page. The byte-emitting ones (Render Image, Play
  Media, Render PDF) validate their input and pass the bytes through, to be
  saved with `-o`, and add a cchef-specific `--output-format`: `Raw`, `Base64`
  (a `data:` URI), and for Render Image `Terminal` (an inline preview via the
  iTerm2 or kitty protocol). Report-style ones (Magic, YARA Rules, ELF Info)
  print text instead of a table.
- **Imaging reproduces Jimp's pixel maths from scratch** on an `image.NRGBA`,
  decoding with the standard library plus `golang.org/x/image`. Lossless formats
  are pixel-identical; JPEG output is a lossy re-encode, and Rotate by
  non-multiples of 90° matches dimensions rather than every pixel. Verified
  against Jimp run under Node, since the oracle cannot execute the image module.
- **Text rendering uses CyberChef's own bitmap font atlases** (vendored in
  `internal/ops/bmfonts/`), so Add Text To Image is pixel-identical, including
  Jimp's quirk of sizing the bitmap at one line per word plus one.
- **Charts emit SVG text.** The parts of d3 they need are ported, so scales,
  ticks, color ramps and hexagonal binning are byte-identical to goldens
  produced by running the real operations under Node. Three mechanical
  deviations make the result valid, standalone SVG: `__data__` bindings are not
  serialized, the clip element is spelled `clipPath`, and the root carries an
  `xmlns`. Two portability details are pinned deliberately: the hexagon angles
  are tabulated because Go and V8 disagree by an ulp on `sin`/`cos`, and the
  interpolation sites force intermediate rounding because Go on arm64 fuses
  multiply-add where JavaScript rounds twice.
- **Multi-file output.** Split Colour Channels is the only operation with a
  `List<File>` output. `core.Dish` has a `TypeFileList`; such a dish is terminal
  and the CLI writes it into `--out-dir`, giving each input its own
  subdirectory when reading a directory.
- **User-supplied regular expressions are Go's RE2.** Lookahead and
  backreferences are unavailable; `regexp2` is reserved for internal patterns
  that need them.

## Proposed reorganization of `internal/ops`

**Status: proposal — not started.** `internal/ops` is one flat package of
**782 files / 165k LOC** (414 non-test at 106k, 368 test at 59k) implementing
the 504 registered operations. Nothing is broken; the concern is navigability
and build/test granularity. Re-measure before acting — these figures drift.

- **The operation files are healthy.** 315 files declare a `Meta()`, most
  registering one operation and some an encode/decode pair. One operation per
  file mirrors CyberChef and keeps the test loop tight; **keep it**. The file
  count is a symptom of scope, not of bad layout.
- **99 files register nothing.** These are the engines and static data — the
  JavaScript parser, the entity and codepage tables, the XML DOM, protobuf, d3,
  the disassemblers, the Bletchley machines, the signature and rule tables —
  sitting alphabetically interleaved with forty-line operations.
- **Coupling is low**, but a handful of shared helpers (`convertToByteArray`,
  `charRep`, `parseEscapedChars`, `imageTransform` and a few others) hold the
  reference graph together. Extracting them breaks it into many small
  components, so the package is genuinely splittable.

Three problems worth fixing:

1. **One compilation unit and one test binary.** A one-line edit recompiles the
   whole package, and there is no per-package caching or parallel test
   execution. Every unexported helper is visible to every file, which is why the
   add-operation skill has to say "grep before adding a helper" — package
   boundaries would enforce that instead of prose.
2. **Category is a side table rather than structure.** `opCategories` in
   `cmd/opmeta.go` hand-maintains 504 entries, policed by a test.
3. **Engines and operations carry equal visual weight** in a directory listing.

The staged approach: extract the shared helpers into `internal/opsutil` and the
standalone engines into their own packages (worthwhile on its own), then split
the remainder one package per category, then generate the category table from
the result. The move is also the moment to audit file-splitting consistency:
reciprocal operations (encode/decode, encrypt/decrypt) normally share one file
per algorithm, so any operation spread across several files should be
re-justified or merged.

## Remaining / future work

Roughly ordered: what blocks going public, then what a polished release looks
like, then open-ended improvements. The
[reorganization of `internal/ops`](#proposed-reorganization-of-internalops)
above is also open.

### Before going public

- **Fuzz the from-scratch parsers.** Go's native fuzzing (`go test -fuzz`),
  applied per package rather than to the built binary, which would only add
  process overhead around the same code. The high-value targets are the parsers
  that take untrusted input: the recipe/Chef-format parser, the file-format
  parsers, the JavaScript parser, the XML DOM, the YARA rule compiler and the
  disassemblers. Reciprocal operations also give cheap round-trip properties
  (decode of encode returns the input).
- **Make the breaking CLI changes** below — renamed flags and stricter
  validation are free only while nobody else is using the tool.

### Distribution

- **GoReleaser.** A no-cgo static binary is its ideal case: macOS, Linux and
  Windows on amd64/arm64 from one config, with archives, checksums, a Homebrew
  tap and nfpm-built deb/rpm once the repository is public. Document
  `tesseract` as the one optional runtime dependency on each platform.
- **Man pages.** cobra can generate a full man tree (`cobra/doc`), but with 504
  subcommands that is a large tree: hand-write a quality `cchef(1)` following
  man-page best practice, then decide whether generated per-subcommand pages
  earn their bulk.

### CLI and user experience

- **Revisit [clig.dev](https://clig.dev/) end to end.** It shaped the interface
  early in development and has not been re-checked since.
- **Rename the web-form flags.** Flags are auto-derived from CyberChef's
  ingredient labels, which were written for a browser UI —
  `magic --crib-known-plaintext-string-or-regex` should be `--crib`, with the
  detail moved into the flag's help text. A curated rename table in `cmd/`
  (following the `opSummaries`/`opAliases` precedent) covers this without
  touching the operations; likewise option *values* that only make sense as
  web-app dropdown labels.
- **US-spelling aliases.** Operation names keep CyberChef's original spellings,
  many of them UK English (`Split Colour Channels`, `Analyse hash`), and the
  derived subcommands, flags and option values inherit them. Alongside those
  originals, accept US-spelled aliases (`split-color-channels`, `--color`,
  `Center`) through the same alias machinery, so a US-English user never has to
  remember which variant an option wants.
- **Tighten input validation.** Every `ArgNumber` is a float64 because
  CyberChef's `number` type is; most are semantically integers.
  `core.CoerceArgs` already enforces declared `Min`/`Max`, so the audit is
  largely about declaring: which arguments are integers, where zero or negative
  values are meaningless, and rejecting rather than truncating or accepting
  nonsense.
- **Evaluate the input convention.** cchef uses explicit flags (`-i` for a
  literal string, `--in-file`, `--in-dir`, stdin as fallback) and — unlike most
  Unix tools — treats a positional argument as a *string* rather than a
  filename. Worth reviewing against the `cat`/`grep` mainstream, curl's `@file`
  sigil, and openssl's `-in`/`-out`, without breaking existing usage.
- **Audit the non-text outputs.** CyberChef operations that present as HTML
  landed in cchef case by case: some byte-emitting ones grew `--output-format`
  (`Raw`, `Base64`, a `Terminal` preview), report-style ones print text, and
  others were handled ad hoc. Sweep every operation whose upstream output or
  presentation is not plain text and make the treatment consistent — the same
  flag name and value set wherever bytes come out, a terminal preview wherever
  one makes sense (any image-producing operation, not just Render Image), and a
  text rendering for every report-style operation.
  [Deliberate differences](#deliberate-differences-from-cyberchef) records the
  intended model; the audit brings every operation up to it.
- **Interactive recipe building — open question.** Something between nothing
  and the web app: a full TUI would pull in a heavy dependency (against
  policy), while a git-style staged model (`recipe add`/`rm`/`toggle`/`reorder`
  against a working recipe file) is small and sits on machinery that already
  exists. Decide the shape before building anything.

### Loosening JavaScript parity

Strict parity was scaffolding: matching CyberChef byte for byte is what made
the oracle and the sweeps possible, and gave confidence the port works. Some of
what it preserved, though, is JavaScript rather than CyberChef — `internal/jsnum`
exists solely to reproduce JS float formatting, and JSON output enumerates
integer keys first because V8 does. Consider tiering the requirement:

- **Keep byte parity where interop is the point** — data transforms whose
  output feeds other tools, and anything reachable from a shared CyberChef URL,
  where the same recipe must give the same answer.
- **Allow Go-native behavior where the quirk is a wart** nobody depends on,
  taking advantage of Go's strengths instead of reproducing JavaScript's
  peculiarities.

Every relaxation is a deliberate difference: record it in
[that section](#deliberate-differences-from-cyberchef), and have the sweeps
assert the divergence precisely (the existing convention for bug fixes). The
cost to watch: each divergence makes the oracle a little less useful.

### Documentation and comments

Comments and docs have accumulated process talk — provenance claims
("faithful", "byte-for-byte"), descriptions of upstream's behavior,
development narrative — where they should state what the code does and the
constraints a reader needs. Clean this up opportunistically as files are
touched rather than as a dedicated audit pass, and hold new code to the
tighter standard so the debt stops growing.

**Documentation is US English**, except where it quotes an operation, flag or
option value whose original spelling is UK English (`Split Colour Channels`
stays as CyberChef named it). The `docs/` pages still need this conversion; it
cannot be a blind find-and-replace, because the examples embed verified command
output and functional option values that must keep their exact spellings.

**Cross-reference the classic tools.** Many operations do the same job as a
well-known CLI tool: the Base64/Base32 codecs (coreutils `base64`/`base32`),
hex and hexdump (`xxd`, `od`), the hash subcommands (`md5sum`/`sha256sum`,
`openssl dgst`), Strings (binutils `strings`), the compression codecs (`gzip`,
`bzip2`, `zip`), text-encoding conversion (`iconv`), UUID generation
(`uuidgen`), jq/JPath (`jq`), Diff (`diff`), and so on. The docs page for such
an operation should say it is an alternative to that tool and link to the
tool's home page or repository, so a reader arriving from either direction can
map one onto the other. Where cchef's behavior differs from the classic tool
(because parity is with CyberChef), the note should say how.

### Fidelity and verification

- **Real-input corpus.** Test the built tool against real inputs, not just
  contrived fixtures: actual ELF and PE binaries, photographs, archives,
  documents, through the relevant operations and against the oracle. Record
  every mismatch — a corpus that drops mismatches hides bugs.
- **Standardized test vectors.** For the standardized algorithms (AES, DES, the
  SHA family, HMAC, RSA, ECDSA), consider cross-checking against NIST's
  [CAVP](https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program)
  vectors. Being independent of CyberChef, they would catch anywhere CyberChef
  itself diverges from the standard — and confirm the places cchef matches it
  deliberately. They validate the core algorithm, not the option plumbing, so
  the oracle checks stay.
- **`Hex to PEM` — mixed hex/non-hex input.** Faithful for all well-formed hex
  and lenient on stray characters as upstream is, but input that *interleaves*
  hex and non-hex (`"3g3"`) is not reproduced: jsrsasign routes through
  CryptoJS's word packing and fractional-`sigBytes` clamp. Closing it means
  porting that pipeline. Low priority — garbage in, garbage out.
- **`Avro to JSON` — 64-bit longs above 2^53.** cchef reads `long` as exact
  `int64`; avsc reads into a float64. Unreachable for avsc-produced files, since
  its encoder rejects such values.

## Verification

- **`make all`** — fmt, fix, vet, test, build, lint and sec, all clean. Plus
  `make complexity`.
- **Both architectures.** CI runs linux/amd64 while development is on arm64, and
  Go's `math` functions differ in the last bit between them. Anything with
  floating-point maths is checked on both:

  ```bash
  docker run --rm --platform linux/amd64 -v "$PWD":/src -w /src \
      -e GOFLAGS=-buildvcs=false -e GOGC=off golang:1.26 go test ./...
  ```

- **The oracle.** For operations without upstream fixtures, a real CyberChef
  gives authoritative output. Two are used: the HTTP server (Docker, from
  `../CyberChef-server`, `POST /bake`), and the Node API for what the server
  cannot do — operations whose `run()` is async or loads WASM, and `List<File>`
  output. Keep the image current: it installs the last *published* npm
  `cyberchef`, so an operation merged after that release is present in the
  source checkout but absent from the oracle. The server also refuses an empty
  input for every operation, so those cases are covered by fixtures instead.
  Flow control is excluded from both oracles by design, so those operations are
  verified against upstream's fixtures alone.
- **Differential sweeps.** For anything with formatting, numeric or
  binary-layout subtleties, fixtures are not enough: sweep many varied inputs
  through cchef and the oracle and compare byte for byte. Where cchef diverges
  deliberately, the sweep asserts the divergence precisely rather than skipping
  the case.
