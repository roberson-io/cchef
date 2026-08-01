# Contributing to cchef

Thanks for your interest in improving cchef. This document covers how to set
up a development environment, the standards changes are held to, and how to
submit them. For the project's design and planned work, see
[PLAN.md](PLAN.md).

## Ground rules

Two constraints shape everything else:

- **cchef ships as a single static binary with no cgo.** A change must build
  and pass tests with `CGO_ENABLED=0`. Dependencies that require cgo, external
  shared libraries, or runtime downloads will not be accepted. (The one
  existing exception: the OCR operation drives an installed `tesseract`
  binary and returns a clear error without it.)
- **Operations match CyberChef.** Each operation is a port of its
  [CyberChef](https://github.com/gchq/CyberChef) counterpart, verified
  against CyberChef's own test fixtures. Don't change an operation's behavior
  away from CyberChef's without raising it in an issue first; if you believe
  CyberChef itself is wrong, that's a bug to report upstream, not a quirk to
  silently diverge from.

## Getting started

You need Go (the version pinned in [go.mod](go.mod)) and `make`. Optional:
`tesseract` for the OCR operation, Docker for cross-architecture checks.

```bash
git clone https://github.com/roberson-io/cchef.git
cd cchef
make install-tools   # golangci-lint, gosec, and friends
make all             # the full gate: fmt-check fix-check vet test build lint sec
```

If `make all` passes on a fresh clone, you're set. The built binary lands in
`dist/cchef`.

## Making changes

Use the Makefile targets (`make test`, `make build`, `make lint`, …) rather
than raw `go` commands — several targets do more than the obvious command.

### Tests

- **Every change needs tests, and new code is expected to be covered.**
  Overall coverage should trend up, not down. For operation behavior, take
  test cases from CyberChef's fixtures
  (`tests/operations/tests/*.mjs` in the CyberChef repository) rather than
  inventing expected values.
- Cover options, error paths, and edge cases — not just the happy path. If a
  branch is genuinely unreachable, delete it rather than writing a test that
  pretends to reach it.
- An operation lives in `ops/<op>.go` with its tests in
  `ops/<op>_test.go` — one operation per file pair, edge cases
  included; don't add separate coverage-test files.
- **Floating-point tests must hold on more than one architecture.** CI runs
  linux/amd64, and Go's `math` functions can differ in the last bit across
  architectures. Derive tolerances from what the algorithm guarantees, not
  from what your machine produces. To check under amd64:

  ```bash
  docker run --rm --platform linux/amd64 -v "$PWD":/src -w /src \
      -e GOFLAGS=-buildvcs=false -e GOGC=off golang:1.26 go test ./...
  ```

### Code standards

Most standards are enforced by the gate — run `make all` and
`make complexity` before pushing and fix what they flag:

- Formatting (`gofmt`) and modernization (`go fix`) must be clean.
- `make complexity` reports functions over the cyclomatic threshold; split
  multi-phase logic into small named helpers instead of adding to the list.
- gosec findings are fixed, not silenced. If a finding is genuinely by design
  (a narrowing conversion, an intentional MD5), annotate the site with
  `// #nosec <RULE> -- <reason>` — rule ID and reason are both required.
- Give meaningful numeric literals named constants; match the style of the
  surrounding code.
- **No vendoring** (`go mod vendor` is never used). New dependencies face a
  high bar — see the dependency policy in PLAN.md — and GPL/AGPL-licensed
  code cannot be accepted, since the project is Apache-2.0. If you add one,
  run `go mod tidy` and mention what it pulls in transitively.

### Documentation

Operation changes usually touch `docs/`:

- Category pages list operations **alphabetically** — summary table, detailed
  sections, and the `docs/README.md` index.
- Examples put the runnable command (no `$` prompt) in a `bash` code block
  and the expected output in a separate plain block under an `Output:` line.
  Run every example against the built binary and paste the real output.
- New operations also register CLI metadata in `cmd/` (a category entry, and
  a curated summary if the derived one reads badly) — tests enforce this, as
  they do the operation counts quoted in PLAN.md and `docs/README.md`.

## Pull requests

- **Open an issue first** for anything beyond a small fix — new dependencies,
  behavioral changes, or CLI surface changes. It saves everyone time.
- Branch from `main`; keep each PR to one logical change.
- Write commit messages as short imperative sentences ("Add Magic
  operation.", "Fix COBS decoding of empty groups.").
- Make sure `make all` and `make complexity` are clean locally — CI runs the
  same gate plus an SBOM vulnerability scan that fails on high-severity
  findings.
- In the PR description, say what the change does and how it was verified.
  For operation behavior, that means which CyberChef fixtures cover it or how
  the output was checked against a running CyberChef.

## Reporting bugs

A good bug report includes the exact command, the input (or a way to
reproduce it), the output you got, and the output you expected. If the
expectation comes from CyberChef, include the recipe or a
`gchq.github.io/CyberChef` URL demonstrating it — cchef's behavior is defined
by CyberChef's, so a divergence between the two is the clearest possible
report.

## License

cchef is licensed under the [Apache License 2.0](LICENSE). By contributing,
you agree that your contributions are licensed under the same terms. Keep
[NOTICE](NOTICE) up to date if a change adds or removes third-party material.
