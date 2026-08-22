# cchef — Agent guide

`cchef` is a Go CLI port of the data-transformation engine of
[CyberChef](https://gchq.github.io/CyberChef/). This file orients a coding
agent (or any contributor) working in the repository: what the project
guarantees, how the tree is laid out, and how work is verified.
[CONTRIBUTING.md](CONTRIBUTING.md) covers the mechanics of submitting changes;
`docs/` documents every operation.

## Orientation

A curated set of 505 operations is implemented, tested and documented, tracked
against CyberChef 11.3.0. Those subcommands cover 501 unique CyberChef
operations; the difference is `SHA2`, which cchef also exposes as the
no-argument `sha224`/`sha256`/`sha384`/`sha512`.

- **505 operations** (`ops/`) — one file per operation, mirroring CyberChef's
  flat `src/core/operations/` directory. Reciprocal pairs (To/From X) share
  one file per algorithm.
- **Engines** (`internal/`) — parsers, codecs and ported libraries the
  operations are built on, mirroring CyberChef's `src/core/lib/`. One package
  per subject; nothing in `internal/` is part of the public API.
- **Core engine** (`core/`) — the Dish/Recipe/Registry machinery, importable
  as a library. **CLI** (`cmd/`) — cobra commands plus centrally registered
  presentation metadata (categories, summaries, aliases). **Docs** (`docs/`)
  — one page per category. **Generators** (`tools/`) — each generated table
  carries the tool that emits it.
- `make all` (fmt, fix, vet, test, build, lint, sec) and `make fuzz` are clean;
  keep them that way.

The operation counts quoted here and in `docs/README.md` are pinned to the
registry by tests in `cmd/counts_test.go`.

## Constraints

These shape every change.

- **A single static binary, no cgo.** This rules out otherwise obvious
  libraries and is why so much is implemented in-repo. OCR is the one
  exception: it drives an installed `tesseract` and errors clearly without it.
- **Operations match CyberChef**, verified against upstream fixtures or a
  running CyberChef. Where cchef differs on purpose it is because CyberChef
  has a defect: implement the correct behavior, never reproduce the bug, and
  note the divergence on the operation's `docs/` page. Differences that are
  *not* bug fixes are listed under
  [Deliberate differences](#deliberate-differences-from-cyberchef).
- **Test-driven**: test → red stub → implementation. The full workflow for
  porting an operation — research, fixtures, oracle verification,
  registration, docs — is in
  [.claude/skills/add-operation/SKILL.md](.claude/skills/add-operation/SKILL.md),
  including one-time setup of the CyberChef checkout and oracle it relies on.
- **Following CyberChef is a choice, not an obligation.** Tracking upstream is
  what makes the oracle useful, but the maintainer decides what cchef
  contains. An operation CyberChef removes may stay, and an upstream change
  may be declined. Semantic versioning is likewise a convention to weigh, not
  a rule that overrides that judgment.
- **Dependencies are a liability.** `golang.org/x/*` and libraries with
  large-organization backing are fine long-term; modules maintained by
  individuals, especially unmaintained ones, are candidates for in-repo
  reimplementation. Nothing GPL or AGPL: the project is Apache-2.0.

## Verification

How to check work.

- **`make all`** — fmt, fix, vet, test, build, lint, sec. Plus `make
  complexity` and `make fuzz`. Use the Makefile targets, not raw `go`
  commands.
- **Both architectures.** CI runs linux/amd64 while development is often on
  arm64, and Go's `math` functions differ in the last bit between them.
  Anything with floating-point maths is checked on both:

  ```bash
  docker run --rm --platform linux/amd64 -v "$PWD":/src -w /src \
      -e GOFLAGS=-buildvcs=false -e GOGC=off golang:1.26 go test ./...
  ```

- **The oracle.** For operations without upstream fixtures, a running
  CyberChef gives authoritative output. Two are used: the CyberChef-server
  HTTP endpoint (`POST /bake`, run via Docker) and the CyberChef Node API for
  what the server cannot do — operations whose `run()` is async or loads
  WASM, and `List<File>` output. Setup for both is in the add-operation
  skill. Keep the image current: it installs the last *published* npm
  `cyberchef`, so an operation merged upstream after that release is in the
  source checkout but absent from the oracle. The server also refuses empty
  input for every operation, so those cases are covered by fixtures instead.
  Flow control is excluded from both oracles by design.
- **Differential sweeps.** For anything with formatting, numeric or
  binary-layout subtleties, fixtures are not enough: sweep many varied inputs
  through cchef and the oracle and compare byte for byte. Where cchef
  diverges deliberately, the sweep asserts the divergence precisely rather
  than skipping the case.
- **Fuzzing.** `make fuzz` runs eight targets across `core`, `ops` and
  `internal/yara`: the parsers that read data cchef did not write — a recipe, a
  share URL — plus round-trip properties over the byte-level codecs. Failing
  inputs land in `testdata/fuzz/` and become regression tests. The target fails
  when a package has no fuzz targets, so a package that moves cannot leave the
  gate silently passing.

## Deliberate differences from CyberChef

Choices, not bug fixes — where cchef corrects an upstream defect instead, the
divergence is noted on the operation's `docs/` page. Add to this list when a
new deliberate difference lands.

- **An option value is matched whatever its case, and normalised.** CyberChef
  validates an `option` argument case-insensitively and then hands the
  operation the string the caller wrote, so `"comma"` is accepted and silently
  selects the default — an upstream defect. cchef accepts the same casings, so
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
- **Non-numeric input to a numeric operation is an error, not `NaN`.**
  CyberChef's `To Base`, `From Base`, `Multiply`, `Sum`, `Divide` and their
  kin convert through a `BigNumber`; a non-numeric or empty value becomes a
  `BigNumber` NaN whose `.toString()` is the literal string `"NaN"`, so these
  operations emit `NaN` instead of failing (the `if (!input) throw` guard never
  fires, because a `BigNumber` object is truthy). cchef rejects a value that is
  not a number, on the same "errors are errors" principle, rather than emitting a
  `NaN` that silently poisons the rest of a recipe. A community recipe that
  deliberately relies on the `NaN` string — the base-45 decoder pipes columns
  through `To Base` and scrubs the resulting `NaN`s with a later `Find / Replace`
  — therefore does not run here.
- **No browser, so no HTML previews.** The byte-emitting operations (Render
  Image, Play Media, Render PDF) validate their input and pass the bytes
  through. Presentation is an IO-layer concern rather than an operation
  argument, so two global flags serve every operation that emits bytes:
  `--preview` renders an image inline (iTerm2/WezTerm or kitty) and
  `--data-uri` writes a `data:<mime>;base64,…` URI. On the same principle,
  Syntax highlighter's hljs-class HTML is rendered as ANSI color when the
  output is going to a terminal, under `--ansi`. Report-style operations
  (Magic, YARA Rules, ELF Info) print text instead of a table. A `List<File>`
  operation (Unzip, Untar, Extract Files, Split Colour Channels, Zip) writes into
  `--out-dir` when it is the recipe's last step; chained into a following
  operation it is not terminal — its files' contents feed on concatenated in
  order, matching CyberChef's `List<File>`→`ArrayBuffer`, so `Unzip | Extract
  URLs` works.
- **Imaging reproduces Jimp's pixel maths** on an `image.NRGBA`. Lossless
  formats are pixel-identical; JPEG output is a lossy re-encode, and Rotate by
  non-multiples of 90° matches dimensions rather than every pixel. Text
  rendering uses CyberChef's own bitmap font atlases, so Add Text To Image is
  pixel-identical.
- **Charts emit standalone SVG.** Byte-identical to goldens produced under
  Node except for three mechanical deviations that make the result valid SVG:
  `__data__` bindings are not serialized, the clip element is spelled
  `clipPath`, and the root carries an `xmlns`. Two portability details are
  pinned deliberately: hexagon angles are tabulated because Go and V8 disagree
  by an ulp on `sin`/`cos`, and the interpolation sites force intermediate
  rounding because Go on arm64 fuses multiply-add where JavaScript rounds
  twice.
- **PGP** runs on `ProtonMail/go-crypto` rather than reproducing kbpgp's exact
  output. Armor headers and generated-key subkey structure differ;
  interoperability is complete in both directions and is what the tests pin.
- **User-supplied regular expressions run on RE2, falling back to a
  JavaScript-compatible engine.** A user pattern is compiled with Go's RE2 first
  — linear-time and safe — and only when RE2 rejects it (lookahead, lookbehind,
  backreferences, or JavaScript `(?<name>)` group syntax) is it retried under
  `regexp2` in ECMAScript mode (`internal/uregex`). The six operations that take
  a user regex share this path: Regular expression, Find / Replace, Subsection,
  Register, Filter and Conditional Jump. The fallback backtracks, so it is
  bounded by a match timeout, and a pattern that exceeds it is treated as no
  match. Two edge differences remain on the fallback path: `\b` is Unicode-aware
  (JavaScript's is ASCII), and a named group is referenced in a replacement as
  `${name}` (Go and .NET syntax) rather than JavaScript's `$<name>`.
- **An Avro `long` is read as a 64-bit integer.** Avro defines `long` as
  signed 64-bit; avsc decodes it through a float64, so CyberChef refuses a
  magnitude above 9007199254740990 and silently returns the wrong sign below
  -2^52 — an upstream defect. cchef reads the whole range exactly, so it
  accepts files CyberChef refuses and disagrees with it on that band. Copying
  either failure would mean returning a number that is not the one in the
  file.

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
  numeric arguments `integer`, so the rest silently truncate a fractional
  value: `Bit shift right` with an amount of 1.5 runs. cchef declares
  `Integer` on 138, refusing the value with `Amount must be an integer.` A
  typo on a command line should fail rather than quietly produce a different
  answer, so a shared URL carrying a fractional argument errors here instead
  of running. All 14 that CyberChef marks are among them.

  cchef also caps twenty parameters CyberChef leaves open, in three groups.
  **Ten bound the cost of a password-based key derivation**, where an open
  parameter turns a typo into an allocation or a run that does not finish:
  `Argon2` (Iterations ≤ 4096, Memory ≤ 2 GiB, Parallelism ≤ 255, Hash length
  ≤ 4096), `Bcrypt` (Rounds 4–31, which bcryptjs silently clamps to instead),
  `Derive PBKDF2 key` and `Derive EVP key` (Key size ≤ 8192 bits, Iterations ≤
  10,000,000) and `Scrypt` (Key length ≤ 4096). **Three bound a hash round
  count** at the length of its constant table, past which CyberChef reads an
  undefined entry and returns a digest built partly from `NaN`: `SHA1` (Rounds
  ≤ 80) and `SHA2` (≤ 64 for the 256 family, ≤ 160 for the 512 family).
  **Seven are older**: `AES Decrypt` (IV Length ≥ 0), `Generate Image` (Pixel
  Scale Factor ≤ 64, Pixels per row ≤ 2048), `Pseudo-Random Integer Generator`
  (Min and Max Value to ±2^53−1), `To Hexdump` (Width ≤ 65536) and `Wrap`
  (Line Width ≤ 65536).

- **The lodash case conversions split words on code points, not UTF-16 code
  units.** `To Camel case`, `To Snake case` and `To Kebab case` split a name into
  words with a pattern transcribed from lodash, which was written against
  JavaScript strings. There a character outside the basic plane is a surrogate
  pair, and alternatives in the pattern match such a pair on its own — as a word,
  and as the base of an emoji sequence — so CyberChef renders `ￍ𐀀` as `ￍ_𐀀`.
  Here the pattern runs over code points, those alternatives never match, and the
  character joins the letters beside it. Text inside the basic plane, which is
  what these operations are used on, is identical.

  Feeding the pattern surrogates does not fix it: regexp2 will not match a lone
  surrogate, so `[\ud800-\udbff]` never fires however the input is encoded.
  Closing the gap means restating the pattern's surrogate, Fitzpatrick-modifier
  and regional-indicator constructs in terms of code points. The rest of the case
  operations — `To Upper case`, `To Lower case` and `Swap case` — apply Unicode
  full case mapping and match CyberChef exactly.
