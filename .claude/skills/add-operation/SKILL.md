---
name: add-operation
description: Add a new CyberChef operation to cchef. Use when implementing/porting any CyberChef operation into this repo — covers the strict TDD flow, referencing the CyberChef source and test fixtures in ../CyberChef, verifying against the oracle, registering the op, and updating docs/ and the operation counts.
---

# Adding a new operation to cchef

`cchef` is a Go port of CyberChef. Each operation is a hand-written Go file in
`ops/` implementing `core.Operation`, self-registered via `init()`, and
developed **test-first**. All of CyberChef's operations are already ported, so
a new operation normally means CyberChef gained one upstream; follow these
steps in order for every new operation.

Always use the **Makefile targets** (`make test`, `make build`, `make lint`,
`make vet`, `make complexity`, `make all` — which itself runs
fmt-check/fix-check/vet/test/build/lint/**sec**) — never raw `go` commands (except
`go tool cover` for coverage inspection).

## 0. Prerequisites (one-time setup)

The port is written against CyberChef's source and verified against a running
CyberChef. Neither lives in this repo; set them up as siblings of it.

- **The CyberChef source** at `../CyberChef`:

  ```
  git clone https://github.com/gchq/CyberChef ../CyberChef
  cd ../CyberChef && npm install
  ```

  Check out the version cchef is aligned with (named in AGENTS.md's Orientation
  section) unless the task is porting something newer. `npm install` matters:
  many operations wrap an npm library, and the authoritative source for those
  is the library's code under `../CyberChef/node_modules/` — read it rather
  than guessing at its behavior.

- **The CyberChef-server oracle** at `../CyberChef-server`, run via Docker:

  ```
  git clone https://github.com/gchq/CyberChef-server ../CyberChef-server
  cd ../CyberChef-server
  docker build -t cchef-oracle-1130 .
  docker run -d --name cchef-oracle-1130-run -p 3001:3000 cchef-oracle-1130
  ```

  The image's build installs the current `cyberchef` npm release, so the tag
  and container names carry the CyberChef version they bake with (`1130` =
  11.3.0) — name a rebuilt image after the version it contains. Then
  `POST localhost:3001/bake` with `{input, recipe:[{op, args}]}` returns real
  CyberChef output for any input/options. If the container dies (a WASM op
  such as Argon2 can kill it), `docker start cchef-oracle-1130-run`.

- **The Node API oracle** needs no extra setup — it runs inside that same
  container, and covers what the HTTP endpoint cannot (async/WASM operations,
  `List<File>` outputs, inputs past the ~100 KB body limit):

  ```
  docker exec -w /CyberChef-server cchef-oracle-1130-run node -e \
    'import("cyberchef").then(async m => {
       const r = await (m.default ?? m).bake("input", [{op:"To Base64", args:["A-Za-z0-9+/="]}]);
       console.log(String(r.value));
     })'
  ```

- **Upstream defects** discovered while porting are logged in
  `../CYBERCHEF-BUGS.md` (a sibling file, not part of this repo — create it if
  it does not exist). cchef implements the *correct* behavior and records the
  CyberChef defect there; it does not reproduce upstream bugs.

## 1. Research the operation in ../CyberChef

For an operation named e.g. `To Base45`:

- **Implementation:** `../CyberChef/src/core/operations/ToBase45.mjs` — read its
  constructor for `name`, `module`, `description`, `infoURL`, `inputType`,
  `outputType`, and `args` (the ingredient definitions), plus the `run()` body.
  Follow any imports into `../CyberChef/src/core/lib/*.mjs` for helper logic.
- **Tests / fixtures:** `../CyberChef/tests/operations/tests/<Op>.mjs` — the
  authoritative `{name, input, expectedOutput, recipeConfig}` cases. `recipeConfig`
  maps directly onto a `core.Recipe`.
- **Category:** find which category (or categories — some ops have several)
  lists the op in `../CyberChef/src/core/config/Categories.json`. This
  determines the `docs/` file(s) and the `cmd/opmeta.go` entry.

## 1a. Decide whether it's a straight port — STOP and flag if not

Most operations are self-contained ports of the `.mjs` `run()` plus `lib/`
helpers. Some are not. **Before writing any code**, check for these and, if any
apply, surface the situation to the user with options rather than silently
implementing a half-faithful version:

- **External-library-backed:** the `.mjs` is a thin wrapper around an npm package
  (e.g. AMF wraps `@astronautlabs/amf`) — there is no logic in CyberChef to port.
  Read the library's source under `../CyberChef/node_modules/`. Options: port
  the library's relevant behavior from scratch into an `internal/` package (the
  usual choice — see the structure note in step 4), add a Go dependency (cuts
  against the low-dependency goal; nothing GPL/AGPL, and scrutinize
  maintenance), or defer.
- **No test fixtures:** no `tests/operations/tests/<Op>.mjs`. You cannot transcribe
  authoritative cases — but you can still get authoritative outputs from the
  **CyberChef-server oracle** (step 0): use it to derive test vectors and to
  differential-test your implementation across many inputs. (Decode `.value`
  with `jq -r` and inspect bytes with `xxd`.) Only fall back to hand-computed
  spec vectors if the oracle is unavailable, and flag reduced fidelity then.
  - **Byte-exact comparison:** the oracle decodes its JSON `input` string as
    **Latin-1** (code point → byte), so non-ASCII input differs from the raw
    UTF-8 bytes cchef sees. To compare like-for-like, drive **both** sides through
    `From Hex` (e.g. `recipe:[{op:"From Hex",args:["Auto"]},{op:"<Op>",...}]`) so
    the input bytes are identical; wrap byteArray output in `To Hex`.
  - **Broad differential sweep:** for non-trivial ports (formatting, line
    wrapping, number formatting), test the fixtures *and* sweep dozens of varied
    inputs through cchef vs. the oracle — the fixtures rarely exercise the tricky
    branches.
  - **Oracle currency:** the oracle bakes with whatever CyberChef version its
    image was built with; a newer op may be missing from it, and an older image
    may predate upstream fixes. When the oracle and the checked-out
    `../CyberChef` source disagree, the source (plus its fixtures) is
    authoritative — or rebuild the image.
  - **Node zlib caveat:** the Node-based oracles use Chromium's zlib while the
    browser uses pako; for deflate-family byte output, pako (vendored under
    `../CyberChef/node_modules/`) is the authority.
- **Needs a new Dish type or engine feature:** e.g. `inputType`/`outputType` of
  a kind not yet in `core` (add the type test-first, as `TypeJSON` was added for
  AMF), or a flow-control op needing non-linear `Recipe.Execute`.

## 2. Write the test FIRST (red)

Create `ops/<name>_test.go` and transcribe the relevant fixture cases into the
shared `opCase` table runner (`runCases`, defined in `ops/fixtures_test.go`,
which also provides `runOp` for single-op error-path tests and `mustHex` for
binary fixtures). Each case is input → expected output via a `core.Recipe`,
with args in the same order as CyberChef's `recipeConfig` args.

```go
func TestFooFixtures(t *testing.T) {
    runCases(t, []opCase{
        {"Foo: example", "input", "expected",
            core.Recipe{{Op: "Foo", Args: []any{"arg0"}}}},
    })
}
```

- **Transcribe upstream fixtures; do not invent values.** If the op has no
  fixture file, author cases whose expected outputs come from the oracle.
- For byteArray-output ops, wrap the result in `To Hex` (`{Op: "To Hex", Args:
  []any{"None"}}`) for readable comparison, mirroring CyberChef's own approach.
- Keep all of an op's tests in its own `<name>_test.go` — no separate
  `*_branches_test.go` or grouped coverage files.

## 3. Add a compiling stub, confirm red

Create `ops/<name>.go` with a type implementing `core.Operation`
(`Meta`, `Args`, `Run`) registered in `init()` via `core.Register`, but with a
no-op `Run`. Run `make test` and confirm the new tests **FAIL**.

## 4. Implement (green)

Port `run()` faithfully from the `.mjs`. Run `make test` until the new tests pass.

**Encoding semantics — decide this up front.** Getting it wrong is silent. Check
whether the `.mjs` iterates:

- **UTF-16 code units** (JS `input[i]`, `charCodeAt`, `String.fromCharCode`) —
  use `utf16.Encode([]rune(s))` / surrogate handling;
- **runes / code points** (`codePointAt`, `for..of`, spread) — range the string;
- **raw bytes** (`ArrayBuffer`/`Uint8Array` input) — `in.Bytes()`.

Also note CyberChef's byteArray→string conversion is UTF-8 in some ops and
Latin-1 in others — verify against the fixtures/oracle. The helpers for both
readings live in `internal/opsutil` (`BytesAsText`, `BytesAsLatin1`,
`TextAsBytes`).

### Where code goes: the repo mirrors CyberChef's layout

`ops/` is the flat equivalent of CyberChef's `src/core/operations/` — **one
file per operation** (`ops/<name>.go`), with reciprocal pairs sharing one file
per algorithm (To/From X together). `internal/` is the equivalent of
`src/core/lib/`: engines and shared machinery, one package per subject
(`internal/jimp`, `internal/xmldom`, `internal/filesig`, …), never exported
from the module.

- If the port needs a substantial engine — a parser, a codec, a from-scratch
  library port, anything the `.mjs` pulls from `lib/` or an npm package that
  other ops might share — put it in an `internal/<pkg>` with a package doc
  comment, and keep only the operation type in `ops/`. The engine's direct
  tests live with the engine; the op's tests stay in `ops/`.
- **Before writing a helper, grep for it.** JS semantics helpers already
  exist: `internal/jsnum` (Number formatting/parsing, `Math.round`),
  `ops/jsbuiltins.go` (substr/slice/charcode-style mirrors), `internal/jsonval`
  (JSON.stringify semantics, ordered maps), `internal/opsutil`
  (`Utils.mjs`-style helpers: HTML escaping, alphabet expansion, byte↔text).
  Duplicates like `padEnd`/`isHexByte` have been added and removed before.
- Large data tables go in a companion file named for what it holds
  (`<op>_oids.go`) or, for non-Go assets, a `//go:embed` directory beside the
  embedding file (`x86tables/`, `bmfonts/`). A generated file carries its
  generator: put it under `tools/` and make it emit the right package and path.

### Keep it readable: complexity and magic numbers

A faithful port does not have to be a monolith. As you implement:

- **Watch function complexity.** Don't let `Run` (or a helper) grow into one long
  tangle of nested branches and phases. Split distinct phases (parse → transform
  → format) into small, well-named helpers. `make complexity` reports functions
  over the cyclomatic threshold; a new op should not add functions to that list
  without good reason. When you extract a helper, follow TDD for the extraction
  too: write its direct unit test first, then extract.
- **No magic numbers.** Give domain-specific numeric literals named `const`s with
  a short comment, and reference the constant everywhere (including error message
  text via `%d`). Byte-level constants that are self-evident in context
  (`& 0xff`, `< 256`) are fine; a limit or code that carries meaning gets a name.

### Mapping CyberChef ingredients → `core.ArgType`

| CyberChef type | core.ArgType | Notes |
| --- | --- | --- |
| `string` / `binaryString` | `ArgString` | |
| `number` | `ArgNumber` | value stored as `float64`; cast in `Run` |
| `boolean` | `ArgBoolean` | |
| `option` | `ArgOption` | `Value` is `[]string`; set `DefaultIndex` if default isn't index 0 |
| `editableOption` | `ArgEditableOption` | `Value` is the default string |
| `toggleString` | `ArgToggleString` | set `ToggleValues`; value is `core.ToggleString{Value, Option}` |

Data types (`InputType`/`OutputType`): `TypeString`, `TypeByteArray`,
`TypeArrayBuffer`, `TypeNumber`, `TypeJSON`, … — see `core`. Key/operand
toggleString args are usually decoded with `convertToByteArray`
(`ops/keyformat.go`).

### Cover the new code (aim for 100%)

The transcribed fixtures usually only exercise happy paths. Check the new file's
coverage and close the gaps with **genuine** tests — every option/format, each
error path, edge cases — verified against the oracle where possible:

```
go test ./ops/ -coverprofile=/tmp/ops.cov
go tool cover -func=/tmp/ops.cov | grep -i <name>.go
```

(`make cover` reports the same for the whole module.) Aim for 100% on the new
op's functions, but **do not fake-test**: if a branch is genuinely unreachable,
remove the dead code instead. Error-path tests use `runOp(...)` and assert
`err != nil`; declared `Min`/`Max` bounds are exercised via `core.CoerceArgs`.
Test helper branches directly even when `Run` cannot reach them.

## 5. Update the docs

In the matching `docs/<category>.md` (e.g. `data-format.md`) — every category
the op belongs to:

- Add the op to the category summary table **and** as a detailed section, both in
  **alphabetical order by operation name**. To find the slot, insert the new
  detailed section immediately before the next operation's `##` heading (or the
  trailing `---` separator) that sorts after it; insert the table row before the
  first existing row that sorts after the new op.
- Include: an external reference link (the op's `infoURL`), an options table
  whose first column names the actual **flag** (`--salt-type`), a **simple
  example**, and — for ops with several options — a **complex example**.
- **Example format — command and output in separate code blocks.** Put the
  runnable command (no `$` prompt) in a ```` ```bash ```` block, then the expected
  output in a following plain ```` ``` ```` block introduced by a line reading
  `Output:`. Omit the `Output:` block for a command with no meaningful output:

  ````
  ```bash
  cchef <subcommand> -i "input"
  ```

  Output:

  ```
  expected output
  ```
  ````

- **Verify every example** by running it: `make build` then
  `./dist/cchef <subcommand> ...`, and paste the real output into the `Output:`
  block. Input/output format selection uses `--input-format`/`--output-format`.
- **`docs/README.md` master table:** add the op (alphabetized) to its category
  row in the per-category operations table. Verify the row matches the registry:
  the `cat<Category>` entries in `cmd/opmeta.go` and the table row should agree.

## 6. Register the CLI presentation metadata

CLI metadata lives centrally in `cmd/`, not in the op's `Meta()`. Update it or
tests will fail:

1. **Category (required).** Add the op's display name to `opCategories` in
   `cmd/opmeta.go`, using the `catXxx` constants — **every** category the op has
   in Categories.json, matching the `docs/README.md` master table. This drives
   the grouping in `cchef list`. `TestOpCategoriesMatchRegistry` fails if a
   registered op has no entry (or an entry names no registered op).
2. **Summary (as needed).** The one-line help/`list` summary (cobra `Short`) is
   auto-derived from the first sentence of `Description`. Run `./dist/cchef list`
   (or `./dist/cchef <subcommand> --help`) and check it: if it truncates
   mid-thought (ends in `…`) or reads awkwardly, add a curated entry to
   `opSummaries` in `cmd/opsummaries.go`, kept under `maxSummaryLen`.
   `TestSummariesFitAndPresent` enforces the length bound.
3. **Alias (optional).** For a very common encode/decode-style op, add a short,
   explicit alias to `opAliases` in `cmd/opaliases.go`. Keep these few (see
   clig.dev); `TestOpAliasesValid` checks aliases are unique, name a real op, and
   never shadow a canonical subcommand name.

## 7. Update the operation counts

AGENTS.md does not track per-operation status — every CyberChef operation is
ported — but the total appears in several count-bearing spots that tests do
**not** catch:

1. **`AGENTS.md` Orientation section:** the "curated set of **N operations**" sentence,
   the "**N operations** (`ops/`)" bullet, and the unique-CyberChef-op count in
   the same paragraph (subcommands can outnumber unique ops — `SHA2` is one op
   but four subcommands).
2. **`docs/README.md`:** the "**Scope:** N operations, covering …" blockquote.
   If the new op exists only in a CyberChef version newer than the one cchef is
   aligned with, say so there rather than claiming blanket coverage.

Grep the old number across `AGENTS.md docs/README.md README.md` to confirm none
were missed. The authoritative subcommand count is the registry size:
`./dist/cchef list` (or count `cmd/opmeta.go` entries).

## 8. Final checks

Run the full gate and ensure it is clean:

```
make all   # fmt-check + fix-check + vet + test + build + lint + sec
```

- **`make all`** runs the whole gate, including **`sec`** (gosec + govulncheck);
  don't substitute a bare `make test`.
- **`make complexity`** is separate from `make all` — run it after implementing
  and confirm no new function is over the cyclomatic threshold.
- If the op trips a gosec finding that is **by design** — a narrowing byte/bit
  conversion (G115), an intentional MD5/SHA1 use (G401/G501/G505), reading a
  user-supplied path (G304) — add an inline **`// #nosec <RULE> -- <reason>`**
  annotation (the `#` and a `--` reason are mandatory; `make sast` enforces
  both). Never blanket-suppress; only annotate genuinely-safe sites.
- **Static binary:** the project is single-binary, no cgo. If anything about the
  op could touch that (a new dependency especially), verify with
  `CGO_ENABLED=0 go build ./...`.
- **SBOM:** only if the op added a dependency, run `make sbom-audit` and confirm
  no high vulns; note the dep (and its transitive deps, after `go mod tidy`) in the PR
  description. Never vendor (`go mod vendor`), and check `git status` afterwards so
  no stray files land in the tree.

Then sanity-check the new subcommand end-to-end (`make build` + a real
invocation). Do not commit unless the user asks.

## Conventions recap

- **Strict TDD**: test → red stub → green implementation, every op, no exceptions.
- **Makefile targets only** for build/test/lint.
- **One file per op** in `ops/`; engines and shared machinery in `internal/`;
  grep before adding a helper.
- **Port behavior, not bugs**: implement what is correct, log CyberChef defects
  in `../CYBERCHEF-BUGS.md`.
- **Alphabetize** operations in all `docs/` files; example commands and outputs
  in separate code blocks.
- **Cover the new code** to ~100% with genuine tests; delete dead branches rather
  than fake-testing them; keep the op's tests in its own `<name>_test.go`.
- **Keep it readable**: split multi-phase logic into named helpers (helper test
  first, then extract); named `const`s instead of magic numbers.
- **CLI metadata is central**: `cmd/opmeta.go` category entry (all categories,
  enforced by test); curated summary in `cmd/opsummaries.go` if the derived one
  truncates.
- **Keep counts in sync**: AGENTS.md Orientation and `docs/README.md` — no test
  catches these.
- **`make all` must be clean** before finishing; also run **`make complexity`**;
  annotate by-design gosec findings with justified `// #nosec`.
- Module path is `github.com/roberson-io/cchef`.
