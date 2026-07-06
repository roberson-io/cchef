---
name: add-operation
description: Add a new CyberChef operation to cchef. Use when implementing/porting any CyberChef operation into this repo — covers the strict TDD flow, referencing the CyberChef source and test fixtures in ../CyberChef, registering the op, and updating docs/ and the PLAN.md checklist.
---

# Adding a new operation to cchef

`cchef` is a Go port of CyberChef (cloned at `../CyberChef`). Each operation is a
hand-written Go file implementing `core.Operation`, self-registered via `init()`,
and developed **test-first**. Follow these steps in order for every new operation.

Always use the **Makefile targets** (`make test`, `make build`, `make lint`,
`make vet`, `make all`, `make sec`) — never raw `go` commands (except
`go tool cover` for coverage inspection).

## 1. Research the operation in ../CyberChef

For an operation named e.g. `To Base45`:

- **Implementation:** `../CyberChef/src/core/operations/ToBase45.mjs` — read its
  constructor for `name`, `module`, `description`, `infoURL`, `inputType`,
  `outputType`, and `args` (the ingredient definitions), plus the `run()` body.
  Follow any imports into `../CyberChef/src/core/lib/*.mjs` for helper logic.
- **Tests / fixtures:** `../CyberChef/tests/operations/tests/<Op>.mjs` — the
  authoritative `{name, input, expectedOutput, recipeConfig}` cases. `recipeConfig`
  maps directly onto a `core.Recipe`.
- **Category:** find which category lists the op in
  `../CyberChef/src/core/config/Categories.json`. This determines the `docs/` file.

## 1a. Decide whether it's a straight port — STOP and flag if not

Most operations are self-contained ports of the `.mjs` `run()` plus `lib/`
helpers. Some are not. **Before writing any code**, check for these and, if any
apply, surface the situation to the user with options rather than silently
implementing a half-faithful version:

- **External-library-backed:** the `.mjs` is a thin wrapper around an npm package
  (e.g. AMF wraps `@astronautlabs/amf`) — there is no logic in CyberChef to port.
  Options: add a Go library dependency (note it cuts against the low-dependency
  goal), reimplement from scratch, or defer.
- **No test fixtures:** no `tests/operations/tests/<Op>.mjs`. You cannot transcribe
  authoritative cases — but you can still get authoritative outputs from the
  **CyberChef-server oracle** (Docker, at `../CyberChef-server`): run it and POST
  `{input, recipe:[{op, args}]}` to `localhost:3000/bake` to obtain real CyberChef
  output for any input/options. Use it to derive test vectors and to
  differential-test your implementation across many inputs. (Decode `.value` with
  `jq -r` and inspect bytes with `xxd`.) Only fall back to hand-computed spec
  vectors if the server is unavailable, and flag reduced fidelity in that case.
  - **Byte-exact comparison:** the oracle decodes its JSON `input` string as
    **Latin-1** (code point → byte), so non-ASCII input differs from the raw
    UTF-8 bytes cchef sees. To compare like-for-like, drive **both** sides through
    `From Hex` (e.g. `recipe:[{op:"From Hex",args:["Auto"]},{op:"<Op>",...}]`) so
    the input bytes are identical; wrap byteArray output in `To Hex`.
  - **Broad differential sweep:** for non-trivial ports (formatting, line
    wrapping, number formatting), test the fixtures *and* sweep dozens of varied
    inputs through cchef vs. the oracle — the fixtures rarely exercise the tricky
    branches. This is how the Quoted-Printable `\r` handling and Float rounding
    bugs were caught.
  - **Oracle currency:** the bundled CyberChef lags git master; very new (2025+)
    ops may be missing or config-present-but-unloadable (e.g. Text-Integer
    Conversion), and some ops (e.g. `Median`) predate upstream fixes. If the
    oracle doesn't have the op or disagrees with current source, the checked-out
    `../CyberChef` source + its fixtures are authoritative.
- **Needs a new Dish type or engine feature:** e.g. `inputType`/`outputType` of
  `JSON`, `BigNumber`, or `List<File>` not yet in `core` (add the type test-first,
  like `TypeJSON` was added for AMF), or a flow-control op needing non-linear
  `Recipe.Execute`.

When you add a dependency, run `go mod tidy` and check for **transitive** deps it
drags in; mention them. When fidelity is reduced (different backing library, no
fixtures), say so explicitly and update the PLAN/docs notes accordingly.

## 2. Write the test FIRST (red)

Create `internal/ops/<name>_test.go` and transcribe the relevant fixture cases
into the shared `opCase` table runner (`runCases`, defined in
`internal/ops/fixtures_test.go`). Each case is input → expected output via a
`core.Recipe`, with args in the same order as CyberChef's `recipeConfig` args.

```go
func TestFooFixtures(t *testing.T) {
    runCases(t, []opCase{
        {"Foo: example", "input", "expected",
            core.Recipe{{Op: "Foo", Args: []any{"arg0"}}},
    })
}
```

- **Transcribe upstream fixtures; do not invent values.** If the op has no
  fixture file, author hand-verified cases and **compute** expected outputs
  (e.g. with a quick `python3 -c`) rather than guessing.
- For byteArray-output ops, wrap the result in `To Hex` (`{Op: "To Hex", Args:
  []any{"None"}}`) for readable comparison, mirroring CyberChef's own approach.

## 3. Add a compiling stub, confirm red

Create `internal/ops/<name>.go` with a type implementing `core.Operation`
(`Meta`, `Args`, `Run`) registered in `init()` via `core.Register`, but with a
no-op `Run`. Run `make test` and confirm the new tests **FAIL**.

## 4. Implement (green)

Port `run()` faithfully from the `.mjs`. Run `make test` until the new tests pass.

**Encoding semantics — decide this up front.** Getting it wrong is silent. Check
whether the `.mjs` iterates:

- **UTF-16 code units** (JS `input[i]`, `charCodeAt`, `String.fromCharCode`) —
  use `utf16.Encode([]rune(s))` / surrogate handling (e.g. Escape Unicode
  Characters, Float NaN patterns);
- **runes / code points** (`codePointAt`, `for..of`, spread) — range the string;
- **raw bytes** (`ArrayBuffer`/`Uint8Array` input) — `in.Bytes()`.

Also note CyberChef's byteArray→string conversion is UTF-8 in some ops and
Latin-1 in others — verify against the fixtures/oracle.

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
`TypeArrayBuffer`, `TypeNumber`. Reuse existing helpers where possible (e.g.
`expandAlphRange`, `nonHex`, `charRep` already exist in `internal/ops`). Before
adding a helper, `grep` for it — duplicates like `padEnd`/`isHexByte` already
exist and will fail to compile.

### Cover the new code (aim for 100%)

The transcribed fixtures usually only exercise happy paths. Check the new file's
coverage and close the gaps with **genuine** tests — every option/format, each
error path, edge cases — verified against the oracle where possible:

```
go test ./internal/ops/ -coverprofile=/tmp/ops.cov
go tool cover -func=/tmp/ops.cov | grep -i <name>.go
```

Aim for 100% on the new op's functions, but **do not fake-test**: if a branch is
genuinely unreachable, remove the dead code instead (e.g. the modhex `ParseUint`
error path and the Base32 padding branch were deleted, not annotated). Error-path
tests use `runOp(...)` and assert `err != nil`; declared `Min`/`Max` bounds are
exercised via `core.CoerceArgs`.

## 5. Update the docs

In the matching `docs/<category>.md` (e.g. `data-format.md`):

- Add the op to the category summary table **and** as a detailed section, both in
  **alphabetical order by operation name**. To find the slot, insert the new
  detailed section immediately before the next operation's `##` heading (or the
  trailing `---` separator) that sorts after it; insert the table row before the
  first existing row that sorts after the new op.
- Include: an external reference link (the op's `infoURL`), an options table, a
  **simple example**, and — for ops with several options — a **complex example**.
- **Verify every example** by running it: `make build` then
  `./dist/cchef <subcommand> ...`, and paste the real output.
- If the op is the first of its category, add a new `docs/<category>.md` and link
  it from `docs/README.md`; otherwise just keep the README index table's op list
  alphabetized.
- **`docs/README.md` master table:** add the op (alphabetised) to its category
  row in the per-category operations table. Verify the row matches the registry:
  `grep -E 'cat<Category>' cmd/opmeta.go` vs. the table row should have no diff.

### Bump the operation counts in both READMEs

Easy to forget — there are several count-bearing spots, and tests do **not** catch
stale counts:

- **`README.md`:** the status-line "**N operations** so far", the "status against
  all M CyberChef operations (… names K …)" numbers, and "The N operations are
  grouped…".
- **`docs/README.md`:** the "**Scope:** N operations are currently ported" line.

The authoritative subcommand count is the registry size: `./dist/cchef list` (or
count `cmd/opmeta.go` entries). Keep these in sync with the PLAN.md counts (step
7).

## 6. Register the CLI presentation metadata

CLI metadata lives centrally in `cmd/`, not in the op's `Meta()`. Update it or
tests will fail:

1. **Category (required).** Add the op's display name to `opCategories` in
   `cmd/opmeta.go`, using the `catXxx` constants and the same category as the
   `docs/README.md` master table (a few ops list more than one). This drives the
   grouping in `cchef list`. `TestOpCategoriesMatchRegistry` fails if a
   registered op has no entry (or an entry names no registered op), so this
   cannot be skipped silently.
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

## 7. Update PLAN.md status

In `PLAN.md` under **Operation implementation status**, make **all** of these
edits (it is easy to miss one):

1. Flip the op's `[ ]` to `[x]`. **Some ops are listed in more than one category
   section** (e.g. Unescape Unicode Characters is under Data format *and*
   Language) — flip every occurrence.
2. Bump the `### <Category> (implemented/total)` count in **each** heading the op
   appears under.
3. Update the **unique-count note** in the section intro (e.g. "**N unique**
   CyberChef operations are covered (M directly plus `SHA2`...)").
4. Update the operation counts in the **Current status** section near the top of
   PLAN.md (the "curated set of N operations" sentence and the "**N operations**
   (`internal/ops/`)" bullet) — these count cchef subcommands, which may differ
   from the unique-op count (e.g. `sha256`/`sha512` are two subcommands but one
   CyberChef `SHA2` op).

## 8. Final checks

Run the full gate and ensure it is clean:

```
make all   # fmt-check + fix-check + vet + test + build + lint
make sec   # gosec SAST + govulncheck
```

- **`make all`** includes **`fix-check`** (`go fix` must have nothing to
  modernise) and **`fmt-check`** — don't skip it for a bare `make test`.
- **`make sec`** runs gosec and govulncheck. If the op trips a gosec finding that
  is **by design** — a narrowing byte/bit conversion (G115), an intentional
  MD5/SHA1 use (G401/G501/G505), reading a user-supplied path (G304) — add an
  inline **`// #nosec <RULE> -- <reason>`** annotation (the `#` and a `--` reason
  are mandatory; `make sast` enforces both). Never blanket-suppress; only annotate
  genuinely-safe sites, and fix real findings.
- **SBOM:** only if the op added a dependency, run `make sbom-audit` and confirm
  no high vulns; note the dep (and its transitive deps) in PLAN.

Then sanity-check the new subcommand end-to-end (`make build` + a real
invocation). Do not commit unless the user asks.

## Conventions recap

- **Strict TDD**: test → red stub → green implementation, every op, no exceptions.
- **Makefile targets only** for build/test/lint.
- **Alphabetize** operations in all `docs/` files.
- **Cover the new code** to ~100% with genuine tests; delete dead branches rather
  than fake-testing them.
- **CLI metadata is central**: every new op needs a `cmd/opmeta.go` category
  entry (enforced by test); check its derived `list`/help summary and add a
  curated one in `cmd/opsummaries.go` if it truncates.
- **Keep counts in sync**: PLAN.md category totals + unique/subcommand counts,
  and both READMEs (`README.md`, `docs/README.md`) — no test catches these.
- **`make all` + `make sec` must be clean** before finishing; annotate by-design
  gosec findings with justified `// #nosec`.
- Module path is `github.com/roberson-io/cchef`.
