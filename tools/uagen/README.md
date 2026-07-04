# uagen — Parse User Agent rule generator

`internal/ops/useragent_rules.go` is generated, not hand-written. It is a Go
transcription of the default detection tables inside
[ua-parser-js](https://github.com/faisalman/ua-parser-js), pinned to **2.0.10**
(the version the CyberChef-server oracle runs, so cchef's `Parse User Agent`
output matches CyberChef byte-for-byte).

There is no JavaScript at runtime — the rules are baked into Go source. Node is
only needed for step 1 below, when bumping the pinned ua-parser-js version.

## Files

- `dump.mjs` — imports ua-parser-js's `defaultRegexes` plus its `strMapper` /
  `strTest` / `lowerize` / `trim` helpers and serialises the five rule tables
  (browser, cpu, device, engine, os) to JSON on stdout.
- `package.json` — pins `ua-parser-js` to the exact extracted version.
- `uarules.json` — the checked-in extraction output (the source of truth the Go
  generator reads). Refresh it with step 1 only when bumping the version.
- `gen.go` — reads `uarules.json` and emits `internal/ops/useragent_rules.go`,
  gofmt-formatted. Run with `go run` (it carries a `//go:build ignore` tag).

## Regenerating

Step 2 alone reproduces `useragent_rules.go` from the committed `uarules.json`
and needs only Go. Do step 1 first when upgrading ua-parser-js.

```bash
# 1. (only when bumping the version) refresh uarules.json — requires Node.
cd tools/uagen
npm install                 # installs the pinned ua-parser-js
node dump.mjs > uarules.json

# 2. regenerate the Go rule tables from uarules.json — requires only Go.
cd ../..
go generate ./internal/ops/   # or: go run tools/uagen/gen.go
```

After a version bump, run `make test` — `internal/ops/useragent_test.go` checks
50 user-agent vectors captured from the genuine ua-parser-js library, so a
behavioural change in the rules will surface there.
