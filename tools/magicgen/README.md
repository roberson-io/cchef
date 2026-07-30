# magicgen

Generates `internal/ops/magic_data.go`, the tables the **Magic** operation
needs, from a CyberChef checkout.

Three things come across:

- **The detection checks.** Every CyberChef operation may declare, in
  `OperationConfig.json`, what data it claims to be able to decode: a pattern
  the data must match, an entropy range it must fall in, the arguments to run
  it with, and optionally what the result must look like afterwards. There are
  120 of these across 49 operations, and all 49 are ported in cchef.
- **The language profiles.** How often each byte value appears in each
  language, as a percentage — 39 languages in the common set and 285 in the
  extensive one, 256 numbers each. Magic compares the data's own byte
  frequencies against these to guess a language.
- **The language names**, so a guess can be reported as "Yiddish" rather than
  "yi".

## Refreshing

Two steps, from the repository root, with CyberChef checked out alongside:

```bash
node tools/magicgen/dump.mjs ../CyberChef > tools/magicgen/magicdata.json
go run tools/magicgen/gen.go
```

The first reads the tables out of CyberChef; the second turns them into Go.
`magicdata.json` is committed so the second step works without a CyberChef
checkout to hand.

## Notes on the translation

- **Patterns** are copied across unchanged — all 120 compile under Go's RE2
  without alteration, and none use lookaround or backreferences. A JavaScript
  `i` or `m` flag becomes an inline `(?i)` or `(?m)`; a `g` flag is dropped, as
  it means nothing to a match test.
- **Patterns are matched against text, not bytes.** Several checks describe
  binary signatures with escapes like `\xff`, which mean the *character* of
  that value. Magic therefore reads data that is not valid UTF-8 one character
  per byte before testing, exactly as CyberChef does, so those signatures match.
- **One check gives an option argument the whole list of choices** rather than
  one of them (`Raw Inflate`'s buffer expansion type). CyberChef looks that
  list up as though it were a single choice, finds nothing, and falls back to
  the operation's default — so the generator takes the first choice, which is
  that default.
