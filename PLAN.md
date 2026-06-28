# CyberChef CLI (`cchef`) — Implementation Plan

## Context

We want a Go CLI named `cchef` that ports the data-transformation engine of CyberChef
(cloned at `../CyberChef`, a JS project with 485+ operations) to the terminal. The goal
is a Unix-friendly tool where each operation is a subcommand, operations chain into
recipes (via pipes, JSON, or CyberChef's "Chef" text format), and recipes round-trip to a
shareable `gchq.github.io/CyberChef` URL.

This first effort delivers the **core engine + recipe/URL machinery + a curated set of
~18 operations** that exercises every part of the architecture. The remaining operations
are added later, one hand-written file at a time, against the same interfaces. Development
is **test-driven**: write the test, stub to compile, implement to green.

Decisions confirmed with the user:
- **Scope:** curated starter set (framework proven end-to-end), not all 485.
- **Chaining:** all three — per-op subcommands over stdin/stdout pipes, *and* a `bake`
  command that runs a JSON or Chef recipe.
- **Op implementation:** hand-written Go files, self-registering in a registry (no codegen).

## How CyberChef works (the model we replicate)

Confirmed by reading the source:

- **Operation** (`../CyberChef/src/core/Operation.mjs`, e.g. `operations/ToBase64.mjs`,
  `operations/ROT13.mjs`): metadata (`name`, `module`, `description`, `infoURL`,
  `inputType`, `outputType`) + `args[]` ingredient definitions + a `run(input, args)`.
- **Ingredient/arg types** (`Ingredient.mjs`): `string`, `binaryString`, `toggleString`
  (`{string, option}` with `toggleValues` e.g. Hex/UTF8/Base64/Latin1), `number`,
  `boolean`, `option` (`[]string` dropdown), `editableOption`. Starter ops only need
  string/number/boolean/option/editableOption/toggleString.
- **Dish** (`Dish.mjs`): data container with a type tag; all types convert through
  `ArrayBuffer` as the hub. Relevant types: `string`, `byteArray`, `ArrayBuffer`, `number`.
- **Recipe** (`Recipe.mjs`): ordered list of `{op, args, disabled?, breakpoint?}`; executes
  sequentially, converting the dish to each op's `inputType` before `run()`. `config` getter
  emits the JSON form `[{op:"To Base64", args:[...]}, ...]`.
- **Chef text format & parsing** (`Utils.generatePrettyRecipe` / `Utils.parseRecipeConfig`,
  `Utils.mjs` ~line 978): `To_Base64('A-Za-z0-9+/=')` with spaces→`_`, args are JSON with
  `[]` stripped and `"`→`'`; optional `/disabled` `/breakpoint` flags. We port these two
  functions (and their regexes) faithfully.
- **URL** (`web/waiters/ControlsWaiter.mjs` + `Utils.encodeURIFragment`): fragment
  `#recipe=<encoded pretty recipe>&input=<base64 of input, then fragment-encoded>`.
  `encodeURIFragment` = `encodeURIComponent` then un-escape the safe set `-._~!$'()*,;:@/?`,
  keeping `&`,`+`,`=` escaped.

## Go package layout

```
cchef/
  go.mod                       module github.com/<user>/cchef (cobra only dep)
  main.go                      thin entry -> cmd.Execute()
  Makefile
  cmd/
    root.go                    root cobra cmd, global flags (-i/--input, --in-file, -o)
    bake.go                    `cchef bake` — run JSON/Chef recipe (-r file / -e expr)
    url.go                     `cchef url` — emit CyberChef URL for a recipe (+input)
    recipe.go                  `cchef recipe convert` — JSON <-> Chef format
    list.go                    `cchef list` — list ops + categories
    register_ops.go            builds one subcommand per registered op (flags from ArgDefs)
  internal/core/
    dish.go        Dish{ data []byte; typ DishType }, Get(DishType)->any, conversions
    operation.go   Operation interface, ArgDef, ArgType consts, ArgValue coercion
    registry.go    Register(Operation) / Get(name) / All(); ops self-register via init()
    recipe.go      RecipeOp{Op,Args,Disabled,Breakpoint}, Recipe.Execute(input)->Dish
    chef.go        GeneratePrettyRecipe / ParseRecipeConfig (port from Utils.mjs)
    url.go         EncodeURIFragment, BuildURL(recipe, input)
  internal/ops/
    base64.go hex.go base32.go url.go xor.go rot.go case.go reverse.go hashes.go
    ops_test.go (+ per-file _test.go)
```

## Core engine details

- **Dish** holds canonical `[]byte`. `Get("string")`→`string(data)`, `Get("byteArray")`→
  `[]byte`, `Get("ArrayBuffer")`→`[]byte`, `Get("number")`→parse. `New(value, typ)` builds.
  String/byteArray/ArrayBuffer are all byte-backed, so conversions are trivial for the
  starter set; `number` does strconv. Keep the hub design so new types slot in.
- **Operation interface** (hand-written ops implement it; metadata via small embedded
  `Base` struct to cut boilerplate):
  ```go
  type Operation interface {
      Meta() OpMeta            // Name, Module, Description, InfoURL, InputType, OutputType
      Args() []ArgDef
      Run(in *Dish, args []ArgValue) (*Dish, error)
  }
  ```
- **ArgDef**: `{Name, Type ArgType, Value any, Min,Max *float64, ToggleValues []string}`.
  `ArgType` consts mirror CyberChef: `string`, `number`, `boolean`, `option`,
  `editableOption`, `toggleString`.
- **Registry**: `init()` in each ops file calls `core.Register(&ToBase64{})`. Imported for
  side effects via a blank import in `cmd/register_ops.go`.
- **Recipe.Execute**: for each non-disabled op, look up by name, coerce its `Args` to the
  ArgDefs, convert dish to `InputType`, call `Run`, set output as `OutputType`. Errors stop
  execution with op name context. `breakpoint` halts and returns the intermediate dish.

## CLI behavior

- **Input resolution** (shared helper): `--in-file` > `-i/--input` > stdin. Output to stdout
  (or `-o` file). This makes `cchef to-base64 | cchef to-hex` work.
- **Per-op subcommands** (`register_ops.go`): for each registered op, derive a cobra command
  named as kebab-case of the op (e.g. `To Base64` → `to-base64`). Map each ArgDef to a flag:
  boolean→`Bool`, number→`Int`/`Float`, string/editableOption→`String`, option→`String`
  with allowed-value validation, toggleString→two flags (`--<arg>` + `--<arg>-type`).
  Flag defaults come from `ArgDef.Value`. The command builds a 1-op Recipe and runs it.
- **`cchef bake`**: `-r recipe.json` / `-r recipe.chef` (auto-detect by leading `[`), or
  `-e "<chef expr>"`. Parses to `[]RecipeOp`, executes against resolved input.
- **`cchef url`**: same recipe inputs as `bake`, but emits the CyberChef share URL instead
  of running it (input base64+fragment-encoded, recipe pretty+fragment-encoded).
- **`cchef recipe convert`**: read a recipe in one format, write the other (JSON<->Chef).
- **`cchef list`**: print ops grouped by module/category with descriptions.

## Curated starter operations (~18)

Chosen to cover every arg type and both string and byteArray flows. Each is a faithful port
of the matching `../CyberChef/src/core/operations/*.mjs`:

- **Data format:** To/From Base64 (`editableOption` alphabet), To/From Hex (`option`
  delimiter), To/From Base32, URL Encode/Decode.
- **Ciphers:** XOR (`toggleString` key + `option` scheme), ROT13 (`boolean`×3 + `number`),
  ROT47 (`number`).
- **Hashing:** MD5, SHA1, SHA256, SHA512 (Go stdlib `crypto/*` — zero new deps).
- **Utils:** Reverse, To Upper case, To Lower case.

All hashing/encoding uses the Go standard library; `encoding/base32`, `encoding/base64`,
`encoding/hex`, `crypto/*`. No third-party crypto.

## TDD workflow (per the user's requirement)

For each op and each core component, in order:
1. Write a `_test.go` with expected I/O (use CyberChef's own examples as fixtures, e.g.
   `hello`→`aGVsbG8=`; for hashes/XOR compute expected via CyberChef or known vectors).
2. Stub the type/function so the package compiles and the test fails.
3. Implement until green. Run `make test` between steps.

Golden tests for `chef.go` and `url.go` assert byte-exact parity with strings produced by
the JS `Utils.generatePrettyRecipe` / `encodeURIFragment` for a few sample recipes.

## Makefile targets

Mirrors common Go project conventions plus the Mattermost-style SBOM targets (the template
uses `cyclonedx-gomod` for generation and `grype` for scanning):

- `build` — `go build -o dist/cchef .`
- `test` — `go test ./...`
- `fmt` — `gofmt -w` / `go vet`
- `lint` — `golangci-lint run` (`install-tools` installs it)
- `install-tools` — `go install` golangci-lint, `github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest`, and grype
- `sbom` — `mkdir -p dist/sbom && cyclonedx-gomod mod -json -output dist/sbom/cchef-sbom.json`
- `sbom-scan` — `grype sbom:dist/sbom/cchef-sbom.json --output table --fail-on high`
- `sbom-audit` — `sbom sbom-scan`
- `clean` — remove `dist/`

(Local env: Go 1.26 present; golangci-lint, cyclonedx-gomod, grype not yet installed —
`install-tools` covers them.)

## Build order (milestones)

1. `go mod init`, add cobra, Makefile, `main.go` + empty `cmd.Execute()`. `make build` works.
2. Core: `dish.go`, `operation.go`, `registry.go` + tests. Then `recipe.go` + tests.
3. First two ops (To/From Base64) hand-written + tests → prove Recipe + registry green.
4. `register_ops.go` auto-builds subcommands; wire stdin/stdout input resolution. Manual
   pipe test passes.
5. `chef.go` (+golden tests), then `bake` command (JSON + Chef).
6. `url.go` (+golden tests), then `url` and `recipe convert` commands.
7. Fill in remaining curated ops (hex, base32, url, xor, rot, hashes, case, reverse), each
   TDD. `list` command.
8. SBOM Makefile targets + `make lint sbom-audit` clean.

## Verification (end-to-end)

- `make test` green; `make lint` clean.
- Pipes: `printf 'hello' | dist/cchef to-base64` → `aGVsbG8=`;
  `printf 'hello' | dist/cchef to-base64 | dist/cchef to-hex`.
- Hash: `printf 'hello' | dist/cchef md5` → `5d41402abc4b2a76b9719d911017c592`.
- bake JSON: `echo '[{"op":"To Base64","args":["A-Za-z0-9+/="]}]' > r.json;
  printf 'hello' | dist/cchef bake -r r.json` → `aGVsbG8=`.
- bake Chef: `printf 'hello' | dist/cchef bake -e "To_Base64()"`.
- url: `printf 'hello' | dist/cchef url -e "To_Hex()"`
  → a `https://gchq.github.io/CyberChef/#recipe=To_Hex()&input=aGVsbG8` URL; open it and
  confirm it loads the recipe + input in CyberChef online.
- recipe convert: round-trip a recipe JSON↔Chef and diff back to the original.
- Cross-check a couple of op outputs against the actual CyberChef web app for parity.
