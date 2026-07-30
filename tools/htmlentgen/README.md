# htmlentgen

Generates `internal/ops/htmlentity_tables.go`, the lookup tables behind **To
HTML Entity** and **From HTML Entity**, from the WHATWG named character
reference set.

This mirrors CyberChef's own `generateHTMLEntities.mjs`, added in 11.3.0, so the
two implementations are derived from the same source by the same rules and
cannot drift apart. Before that, both projects carried hand-written tables; that
is where the three malformed values (`&epsi;,`, `&rlhar;;`, `&nge;;`) and a
handful of deprecated code points came from.

## Files

- **`entity.json`** — the spec data, taken verbatim from
  <https://html.spec.whatwg.org/entities.json>. `entity.txt` records its
  provenance and licence.
- **`overrides.json`** — the canonical name to use when encoding a code point
  that the spec gives several names. Same values as CyberChef's
  `htmlEntityOverrides.mjs`.
- **`gen.go`** — turns the two into Go.

## Regenerating

From the repository root:

```bash
go run tools/htmlentgen/gen.go
```

To take a newer spec release, replace `entity.json` and run that again.

## The rules

- **Decoding** understands every name the spec defines that is written with a
  trailing semicolon and stands for a single code point — 2,032 of them. The
  spec's 93 multi-code-point names and its 106 legacy semicolon-less forms are
  left out, as upstream leaves them out.
- **Encoding** uses one canonical name per code point — 1,446 of them. Where the
  spec offers several, the choice is the override if `overrides.json` names one,
  and otherwise a deterministic tiebreak: prefer a name that is not written
  entirely in capitals, then the shortest, then the alphabetically first.
- **The tables hold bare names.** The `&` and `;` are added by the operation, so
  a stored name cannot carry stray punctuation of its own — which is exactly how
  the three malformed values arose.

Both tables are verified to match CyberChef's generated `HTMLEntities.mjs`
entry for entry.
