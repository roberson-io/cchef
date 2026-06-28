---
name: add-operation
description: Add a new CyberChef operation to cchef. Use when implementing/porting any CyberChef operation into this repo — covers the strict TDD flow, referencing the CyberChef source and test fixtures in ../CyberChef, registering the op, and updating docs/ and the PLAN.md checklist.
---

# Adding a new operation to cchef

`cchef` is a Go port of CyberChef (cloned at `../CyberChef`). Each operation is a
hand-written Go file implementing `core.Operation`, self-registered via `init()`,
and developed **test-first**. Follow these steps in order for every new operation.

Always use the **Makefile targets** (`make test`, `make build`, `make lint`,
`make vet`) — never raw `go` commands.

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
`expandAlphRange`, `nonHex`, `charRep` already exist in `internal/ops`).

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

## 6. Update PLAN.md status

In `PLAN.md` under **Operation implementation status**, make **all** of these
edits (it is easy to miss one):

1. Flip the op's `[ ]` to `[x]`.
2. Bump that category's `### <Category> (implemented/total)` count in the heading.
3. Update the **unique-count note** in the section intro (e.g. "**N unique**
   CyberChef operations are covered (M directly plus `SHA2`...)").
4. Update the operation counts in the **Current status** section near the top of
   PLAN.md (the "curated set of N operations" sentence and the "**N operations**
   (`internal/ops/`)" bullet) — these count cchef subcommands, which may differ
   from the unique-op count (e.g. `sha256`/`sha512` are two subcommands but one
   CyberChef `SHA2` op).

## 7. Final checks

Run all three and ensure they are clean:

```
make test
make vet
make lint
```

Then sanity-check the new subcommand end-to-end (`make build` + a real
invocation). Do not commit unless the user asks.

## Conventions recap

- **Strict TDD**: test → red stub → green implementation, every op, no exceptions.
- **Makefile targets only** for build/test/lint.
- **Alphabetize** operations in all `docs/` files.
- Module path is `github.com/roberson-io/cchef`.
