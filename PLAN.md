# CyberChef CLI (`cchef`) — Plan & Status

## Context

`cchef` is a Go CLI port of the data-transformation engine of
[CyberChef](https://gchq.github.io/CyberChef/) (cloned at `../CyberChef`, a JS
project with 498 operations). The goal is a Unix-friendly tool where each
operation is a subcommand, operations chain into recipes (via pipes, JSON, or
CyberChef's "Chef" text format), and recipes round-trip to a shareable
`gchq.github.io/CyberChef` URL.

Development is **test-driven** (test → stub → implement), reusing CyberChef's own
fixture cases for parity, and keeps external dependencies minimal:

- `cobra` — the CLI framework.
- `github.com/elobuff/goamf` — backs the AMF operations (CyberChef likewise wraps
  an external AMF library).
- `golang.org/x/crypto/sha3` — provides the vetted legacy-Keccak constructors for
  Keccak-256/512. Keccak-224/384 (not in that package) use a small local
  Keccak-f sponge, cross-validated against x/crypto (256/512) and the stdlib
  `crypto/sha3` (SHA-3 mode) in tests.
- `golang.org/x/crypto/bcrypt` — backs **Bcrypt** / **Bcrypt compare** /
  **Bcrypt parse** (CyberChef wraps npm `bcryptjs`). Same already-required module
  as `sha3`, so **no new dependency**. A thin bcryptjs-compat layer matches its
  behaviour: cost clamped to [4,31], password truncated to 72 bytes, `$2b$`
  version prefix, and bcryptjs's exact error strings — cross-validated byte-for-byte
  against the `bcryptjs` oracle (hashes verify in both directions).
- `golang.org/x/crypto/blowfish` — backs **Blowfish Encrypt** / **Blowfish
  Decrypt** (CyberChef wraps `node-forge` + `lib/Blowfish.mjs`). Same already-required
  module as `sha3`/`bcrypt`, so **no new dependency**. The CBC/CFB/OFB/CTR/ECB mode
  and PKCS#7-padding plumbing is shared with the AES op (block-size-agnostic
  helpers, 8-byte block); differential-verified byte-for-byte against the node-forge
  oracle across all modes and input lengths, both directions.
- `github.com/dlclark/regexp2` — pure-Go, PCRE-compatible regex (no cgo, no
  transitive deps). Used for the Defang IP / Defang URL / Parse IP range / Parse
  IPv6 address matchers (whose CyberChef regexes rely on lookahead and
  backreferences Go's RE2 `regexp` cannot express) and for the ~340
  `ua-parser-js` detection regexes; all ported verbatim to preserve parity.
- **Codepage engine** (`internal/ops/codepage.go`) — **Decode text** /
  **Encode text** / **Text Encoding Brute Force** are backed by a from-scratch Go
  port of the `codepage` (cptable) npm library CyberChef wraps, reproducing its
  decode/encode byte-for-byte across all **152** charsets (including cptable's
  cached-vs-general dispatch quirks and its UTF-7 encoder truncation bug). No new
  module: the 140 table-backed codepages' decode tables are extracted from
  cptable into an embedded gzipped blob (`codepage_data.bin.gz`, ~1.2 MB) by
  `tools/cpgen/gen.js`, encode tables derive from them at load, and the seven
  magic encodings (UTF-8/16/32, UTF-7, US-ASCII) are algorithmic. The five
  ISO-2022 charsets are unsupported by cptable itself and error, as upstream
  does. Differential-verified against cptable for all 152 charsets.
- `go.yaml.in/yaml/v3` — backs **JSON to YAML** / **YAML to JSON**, which
  CyberChef implements over two different JS YAML libraries (`yaml` and
  `js-yaml`, both YAML 1.2). It was already in the build graph (cobra depends on
  it), so this adds no new module. Output matches CyberChef for the common cases
  (order-preserving with a 2-space block style; timestamps as ISO strings;
  numbers as JS float64); it diverges on a few YAML 1.1-vs-1.2 points documented
  in `docs/data-format.md` (defensive quoting of `yes`/`no`/`on` and bool-family
  keys, single- vs double-quote style, astral-character escaping), all
  differential-verified against the oracle.
- `github.com/golang-jwt/jwt/v5` — backs **JWT Decode** / **JWT Sign** / **JWT
  Verify** (CyberChef wraps the npm `jsonwebtoken` library, which has no logic to
  port). Pure Go, **zero transitive dependencies** (stdlib crypto only). The op
  layer drives only its signing/verifying primitives and PEM key parsing; header
  and payload JSON are serialized in-repo (order-preserving via `jsWriteObject`)
  to match `jsonwebtoken` byte-for-byte, including its `iat` auto-injection and
  the RSA-min-size / ECDSA-curve precondition error messages. Signing is
  non-deterministic where `jsonwebtoken` is (auto-`iat`, randomized `ES*`), so
  those cases round-trip through Decode; HS/RS tokens are verified byte-exact.
- `github.com/ProtonMail/go-crypto` — backs the seven **PGP** operations (CyberChef
  wraps the `kbpgp` npm library, which has no logic to port). The maintained,
  ProtonMail-backed OpenPGP library; its one transitive dependency
  `github.com/cloudflare/circl` (Cloudflare-backed) is pinned to ≥ v1.6.3 to avoid
  GO-2026-4550. Fully interoperable with kbpgp: cchef reads/writes messages and
  keys that round-trip against the CyberChef-server oracle in both directions,
  across RSA and NIST-curve ECC keys.
- `github.com/evanw/esbuild` — backs **JavaScript Minify** (CyberChef wraps the
  npm `terser` library, which has no logic to port). esbuild's minifier is pure
  Go, keeping cchef a single static binary; its only transitive dependency
  (`golang.org/x/sys`) was already in the build graph, so it adds **no new
  transitive dependency**. **Reduced fidelity, by design**: esbuild ≠ terser, so
  the minified output is equivalent but not byte-identical (different identifier
  manglers / compression passes). This is the one JS operation that is not a
  faithful byte-for-byte port; see `docs/code-tidy.md`.
- `github.com/antchfx/xpath` — a standalone XPath 1.0 evaluator (**zero transitive
  dependencies**) used only as the query engine for **CSS selector** and **XPath
  expression**. It is *not* an HTML parser: cchef parses and
  serialises the document with its own from-scratch xmldom-faithful parser
  (`internal/ops/xmlparse.go`, `xmldom.go`) and adapts the tree to the evaluator
  via a `NodeNavigator` (`xmldomnav.go`); CSS selectors are translated to XPath
  (`cssxpath.go`). This preserves byte-for-byte fidelity to CyberChef's
  `@xmldom/xmldom` + `nwmatcher` output, which the general-purpose HTML5 parser
  libraries (cascadia, goquery) could not reproduce.
- `github.com/recolabs/gnata` — backs **Jsonata Query** (CyberChef wraps the
  `jsonata` npm package, which has no logic to port). A pure-Go implementation of
  JSONata 2.x, **no cgo**; its three transitive dependencies
  (`github.com/tidwall/gjson`, `match`, `pretty`) are all pure Go. It was picked
  over `github.com/xiatechs/jsonata-go` on measurement against CyberChef's own 45
  test cases: gnata gets 37 exact with 2 real failures, xiatechs 34 with 5,
  including a division that loses precision (`0.04784689` for
  `0.04784688995215311`). gnata is also current where xiatechs is not.
  **Reduced fidelity, by design**: 6 further cases agree in every value and type
  but write an object's keys in a different order, because gnata keeps the order
  internally and does not expose it; and one expression form
  (`Numbers[0] / Numbers[4]`) does not compile, the `/` after a `]` being read as
  the start of a regular expression. Both are pinned by name in
  `internal/ops/jsonataquery_test.go`, which fails if either starts working.
- **Template** takes **no dependency**: there is no maintained Go port of
  Handlebars (`mailgun/raymond` last released 2022, `flowchartsman/handlebars`
  2021, `aymerick/raymond` 2018), so rather than take an unmaintained one the
  language is implemented from scratch in `internal/ops/handlebars*.go` — tags,
  paths, the built-in block helpers, inline partials, and the whitespace rules.
  **Reduced fidelity, by design**: custom helpers, subexpressions, block
  parameters and partial parameters are not covered, none of which CyberChef
  gives a way to supply.
- `github.com/itchyny/gojq` — backs **Jq** (CyberChef wraps jq-web, jq compiled to
  WASM, which has no logic to port). gojq is a pure-Go reimplementation of jq, so
  cchef stays a single static binary with **no cgo**; its only transitive
  dependency is `github.com/itchyny/timefmt-go` (also pure Go — gojq's other
  module requirements are used solely by its CLI package, which cchef does not
  build). The op reproduces jq-web's `jq.json()` stream collapse (0 results → an
  error, 1 → the value, N → a JSON array) and `JSON.stringify`-style output
  (NaN → null). As gojq is an independent implementation, error-message wording
  and rare numeric edges can differ; differential-verified against the
  CyberChef-server oracle across the common query surface.

- `github.com/yuin/goldmark` — backs **Render Markdown** (CyberChef wraps
  markdown-it + highlight.js, wrapping the output in a div). goldmark is a pure-Go
  CommonMark renderer with **zero transitive dependencies**. With the Table and
  Strikethrough extensions enabled (markdown-it's default surface), plus a custom
  renderer to escape raw HTML (matching markdown-it's `html:false`), render
  strikethrough as `<s>`, and a `target="_blank"` pass, cchef matches markdown-it
  byte-for-byte across the common Markdown surface (verified against the oracle).
  **Reduced fidelity, by design** on two points: fenced-code syntax highlighting
  is not ported (no highlight.js, so the *Enable syntax highlighting* option is
  inert) and block-level raw HTML is escaped without markdown-it's surrounding
  `<p>`. See `docs/code-tidy.md`.

- `github.com/alecthomas/chroma/v2` — backs **Syntax highlighter** (CyberChef
  highlights with highlight.js, which has ~190 hand-written grammars and an
  auto-detector that no Go library reproduces). chroma is a pure-Go highlighter
  (**no cgo**; its one transitive dependency, `github.com/dlclark/regexp2/v2`, is
  also pure Go). The op maps chroma's token types onto highlight.js's `hljs-*` CSS
  class vocabulary so the default HTML output keeps CyberChef's shape, and adds a
  CLI-native *Terminal* output format (chroma's ANSI formatter) for direct display
  in a terminal. **Reduced fidelity, by design**: chroma ≠ highlight.js, so token
  boundaries and especially language auto-detection differ. The op is excluded
  from CyberChef's own Node build, so there are no upstream fixtures or oracle
  output; tests assert the structural contract. See `docs/code-tidy.md`.

Note: **SQL Beautify** (CyberChef wraps the `sql-formatter` npm library) is a
from-scratch pure-Go port (`internal/ops/sqlbeautify.go`) — **no new dependency**.
It reimplements sql-formatter's tokenizer, a small clause/expression parser and a
direct port of its whitespace-layout engine (Layout / Indentation / InlineLayout),
fixed to CyberChef's config (MySQL dialect, standard indent style, keywordCase
preserve) plus the `:name` bind-variable placeholder shuffle. The layout-relevant
MySQL keyword categories (clauses, set operations, joins) and the function/data-type
name sets (which govern function-call spacing) are embedded; a full grammar port was
avoided. Differential-verified byte-for-byte against the CyberChef-server oracle
across a broad SQL corpus (100/100). Exotic dialect-specific constructs may differ.

Note: **JPath expression** (CyberChef wraps the `jsonpath-plus` npm library) is a
from-scratch pure-Go JSONPath evaluator (`internal/ops/jpath.go`) — **no new
dependency**. It reuses the order-preserving JSON representation in `jsonvalue.go`
(`jsonParseOrdered` + `jsStringify`) so matched values serialize byte-for-byte like
`jsonpath-plus`, including ECMAScript object-key ordering. Supports child/wildcard/
recursive-descent/index-union/slice/filter/script-expression syntax; differential-
verified against the CyberChef-server oracle (127/129), the two divergences being
degenerate inputs (a trailing unterminated `[`, and a bare `null` document on which
jsonpath-plus itself throws an uncaught TypeError). The general-purpose Go JSONPath
libraries (e.g. `ohler55/ojg`) were rejected: they sort object keys and lack script
expressions, so they could not reproduce `jsonpath-plus` output.

Note: **JSON Beautify** (CyberChef parses with the `json5` npm library, then
`JSON.stringify`) is a from-scratch pure-Go port (`internal/ops/jsonbeautify.go`)
— **no new dependency**. It adds a lenient JSON5 parser (comments, unquoted and
single-quoted keys, trailing commas, hexadecimal and non-finite numbers,
leading/trailing decimal points, signs, string line-continuations and \x/\u
escapes) that feeds the shared order-preserving serialiser in `jsonvalue.go`
(which accepts an arbitrary indent-unit string, not just a space count). All numbers are float64, matching JS Number semantics (large-integer
precision loss, NaN/Infinity → null, ECMAScript integer-key ordering). The third
argument (Formatted) is inert — it only drives CyberChef's browser tree view.
Differential-verified byte-for-byte against the CyberChef-server oracle across the
lenient-parsing and formatting surface.

Note: **BSON serialise** / **BSON deserialise** (CyberChef wraps the `bson` npm
library) share a from-scratch pure-Go codec (`internal/ops/bson.go`) — **no new
dependency**. serialise reproduces js-bson's `serialize()` byte-for-byte: the
number-type rule (int32-range integer → int32; larger integers, fractional numbers
and negative zero → double), ECMAScript key ordering, and the exact root-input
error text. deserialise renders each element type as js-bson's
`JSON.stringify(_, null, 2)` does (ObjectId → hex string, UTC datetime → ISO
string, Binary → base64, Timestamp → `{"$timestamp":"…"}`, RegExp/MinKey/MaxKey →
`{}`), reusing `jsonvalue.go` for output. Differential-verified against the oracle
across a broad corpus. **Reduced fidelity, by design** for rare externally-sourced
types: Decimal128 (which needs a full IEEE-754 decimal decode), JavaScript code,
DBPointer and Symbol are not decoded and error; see `docs/code-tidy.md`.

Note: the **vkbeautify family** — **JSON Minify**, **XML Beautify**, **XML
Minify**, **SQL Minify**, **CSS Beautify**, **CSS Minify** (CyberChef wraps the
`vkbeautify` npm library) — is a from-scratch pure-Go port — **no new dependency**.
A shared `internal/ops/vkbeautify.go` holds the pieces used across ops
(`createShiftArr` and a JS-`\s` character class, since Go's RE2 `\s` is narrower
than JavaScript's); each operation lives in its own `<op>.go`. JSON Minify reuses
`jsonvalue.go` (`JSON.stringify(JSON.parse(text), null, 0)`); the other five are
faithful transliterations of vkbeautify's regex/loop logic, preserving its quirks
byte-for-byte (non-global paren replaces in `sqlmin`, the digit-leading indent
falling back to four spaces, CSS Beautify's trailing newline and its literal
`"undefined"` output for unbalanced braces). Differential-verified against the
library directly (`node -e "require('vkbeautify')..."`), since only JSON Minify has
upstream fixtures.

Note: the **case conversions** — **To Snake case**, **To Camel case**, **To Kebab
case** (CyberChef wraps lodash's `snakeCase`/`camelCase`/`kebabCase`) — share a
from-scratch port of lodash's word splitter in `internal/ops/lodashcase.go` —
**no new dependency**. It reproduces `deburr` (the ~120-entry accent table), the
ASCII-vs-Unicode word dispatch, and lodash's `reUnicodeWord`. That regex uses
lookahead, which Go's RE2 cannot express, so it runs under the already-present
`github.com/dlclark/regexp2`; the "context aware" mode ports lib/Code.mjs's
`replaceVariableNames`. Differential-verified byte-for-byte against the oracle
across a broad corpus (accents, acronyms, digits, ordinals, smart mode). **Reduced
fidelity, by design**: lodash's word regex is UTF-16-oriented, so astral characters
(emoji, surrogate pairs) may split into words differently — cchef treats them as
word constituents where lodash isolates them; BMP text is byte-identical. See
`docs/code-tidy.md`.

Note: the final **Code tidy** batch — **PHP Serialize** / **PHP Deserialize**,
**Strip HTML tags**, **Generic Code Beautify**, **Microsoft Script Decoder** — are
self-contained pure-Go ports of CyberChef's own operation logic (no external
library), so **no new dependency**. PHP Serialize/Deserialize reproduce the
serializer/recursive parser (string lengths are JS UTF-16 code units, not PHP byte
lengths; the "Output valid JSON" flag quotes integer keys); Strip HTML tags and
Generic Code Beautify are faithful transliterations of the RE2-compatible regex
chains (Generic Code Beautify's preserve-token exec/lastIndex loop and index-based
indenter included); Microsoft Script Decoder ports the decode with its D_DECODE /
D_COMBINATION tables (generated from source). PHP Serialize and PHP Deserialize
have upstream fixtures; the rest are differential-verified byte-for-byte against
the CyberChef-server oracle (Generic Code Beautify across 30 varied inputs). This
batch completes the **Code tidy** category (30/30).

Note: **Parse User Agent** is a faithful port of `ua-parser-js` **2.0.10** (the
exact version the CyberChef-server oracle runs). Its rule tables
(`internal/ops/useragent_rules.go`) are *generated* from that library's source and
differential-tested against it; no runtime dependency on the JS library is added.

Note: **RC2 Encrypt/Decrypt** (CyberChef wraps `node-forge`) are a from-scratch
pure-Go port of the RFC 2268 cipher (`internal/ops/rc2.go`) — **no new
dependency** — reusing the shared ECB/CBC + PKCS#7 block plumbing. Fixed at 128
effective key bits (forge's default). Byte-for-byte differential-verified against
the node-forge oracle across key lengths 1–130 bytes, all input alignments, and
both modes, including forge's lenient unpadding and its all-`0xd9` empty-key
register. One deliberate divergence: cchef rejects non-standard IV lengths
(requires 0 for ECB or 8 for CBC), whereas CyberChef feeds any length to forge and
emits buggy output.

Note: **RC4 / RC4 Drop** (CyberChef wraps `crypto-js`) are a from-scratch pure-Go
port (`internal/ops/rc4.go`) — **no new dependency**. Standard RC4 with a raw key
(the passphrase is parsed to bytes, not run through a password KDF), `drop` counted
in 32-bit dwords. Includes the CryptoJS format system (Hex/Base64/UTF8/Latin1 and
UTF16/UTF16LE/UTF16BE) for key/input/output. Byte-for-byte differential-verified
against the oracle across all key/input formats and Hex/Base64/Latin1 output; UTF8
output errors on malformed UTF-8 as CryptoJS does. **Limitation:** UTF16* *output*
matches only for representable code units — CryptoJS emits lone surrogates (valid
in a JS string but not in Go's UTF-8), so surrogate-range ciphertext decodes to
U+FFFD instead. UTF16* *input* is fully faithful.

Note: **RC6 Encrypt/Decrypt** are a from-scratch pure-Go port of CyberChef's
self-contained RC6 engine (`internal/ops/rc6.go`) — **no new dependency**. Fully
parameterised (word size 8–256 bits, rounds 1–255), with words held as `math/big`
integers so 128- and 256-bit word sizes work. All five modes (ECB/CBC/CFB/OFB/CTR,
the last with a little-endian counter) and PKCS5/NO/ZERO/RANDOM/BIT padding are
supported; the ECB/CBC padding is the `blockApplyPadding`/`blockRemovePadding`
helper now shared with PRESENT. Verified against the upstream IETF draft-krovetz
fixtures (w=8/16/32/64/128 plus non-standard w=24/80) and a broad oracle sweep over
word sizes, modes, rounds and paddings.

Note: **Salsa20 / XSalsa20** are a from-scratch pure-Go port of CyberChef's
self-contained Salsa20 engine (`internal/ops/salsa20.go`) — **no new dependency**.
Both share the `salsa20Permute` / `salsa20Block` core; XSalsa20 adds only the
`hsalsa20` subkey derivation over the first 16 nonce bytes. Supports 16/32-byte
keys, 8/12/20 rounds, and the Hex/UTF8/Latin1/Base64/Integer nonce formats.
Byte-for-byte verified against the upstream ECRYPT fixtures and a broad oracle
sweep over key/nonce sizes, rounds, counters and input/output formats. One
divergence for a degenerate input: an *Integer* nonce yields only 8 bytes, so for
XSalsa20 (which needs 24) cchef zero-pads the sub-nonces rather than reproducing
CyberChef's out-of-bounds/NaN output.

Note: **Scrypt** is listed under both Encryption / Encoding and Hashing. CyberChef
wraps the `scryptsy` npm package; cchef backs it with the canonical
`golang.org/x/crypto/scrypt` (RFC 7914; **already a dependency**, so no new one).
The parameter validation reproduces scryptsy's error strings (N power-of-two, N/r
size bounds — matched using JS float division). Output is byte-for-byte identical
to CyberChef on all standard parameters (RFC 7914 vectors plus a broad oracle sweep
over salt encodings, N/r/p and key length). Divergences occur only at
RFC-forbidden degenerate parameters that scryptsy still computes but the standard
scrypt rejects (e.g. N=1 or p=0); a zero key length short-circuits to empty output
to match scryptsy while avoiding a panic in the canonical implementation.

Note: **SIGABA** is a from-scratch pure-Go port of CyberChef's self-contained
SIGABA emulator (`internal/ops/sigabamachine.go` + `sigaba.go`) — **no new
dependency**. All 41 arguments (5 cipher, 5 control, 5 index rotors plus the mode)
match CyberChef's ordering, defaulting to the built-in "Example" rotor wirings so a
bare invocation is a valid machine. Byte-for-byte verified against the three
upstream fixtures and a broad oracle sweep over varied inputs (letters, spaces,
mixed case); the space↔`Z`/`Z`→`X` substitution and rotor stepping match exactly.

Note: **SM4 Encrypt/Decrypt** are a from-scratch pure-Go port of CyberChef's
self-contained SM4 engine (`internal/ops/sm4engine.go`) — **no new dependency** —
implementing the GB/T 32907-2016 cipher and the IETF draft-ribose-cfrg-sm4-09 block
modes (ECB/CBC with PKCS#7 or NoPadding, and CFB/OFB/CTR, the CTR counter added to
the low 32-bit word as upstream does). Both ops share the engine. Byte-for-byte
verified against the upstream draft-ribose fixtures and a broad oracle sweep over
all modes and input lengths (with round-trip decryption). The block loops guard
against a short final read so a degenerate misaligned NoPadding input can't panic
(CyberChef reads past the array and emits NaN garbage there instead).

Note: **Argon2 / Argon2 compare** (CyberChef wraps the `argon2-browser` WASM
package) are a hybrid: **Argon2i** and **Argon2id** use `golang.org/x/crypto/argon2`
(the already-required `x/crypto` module — **no new dependency**), and **Argon2d**
uses a from-scratch RFC 9106 port (`internal/ops/argon2core.go`), since `x/crypto`
does not provide it. The op layer adds the PHC encoded-hash format, the hex/raw
outputs, the reference parameter-validation error messages (output/salt/memory/
time/lanes order), and PHC parsing + constant-time verify for compare. The
CyberChef-server oracle cannot load the Argon2 WASM, so vectors were taken from
`argon2-cffi` (the reference phc-winner-argon2 C library `argon2-browser` is built
from); `x/crypto` reproduces the Argon2i/Argon2id values exactly, and the
from-scratch Argon2d matches the reference across single- and multi-lane params.

Note: **TEA / XTEA Encrypt/Decrypt** (4 ops) are a from-scratch pure-Go port of
CyberChef's self-contained `lib/TEA.mjs` (`internal/ops/tea.go`) — **no new
dependency**. Both algorithms share one file: 64-bit-block Feistel ciphers with a
128-bit key, the golden-ratio DELTA schedule, all five modes (ECB/CBC/CFB/OFB/CTR,
big-endian CTR) and PKCS5/NO/ZERO/RANDOM/BIT padding (reusing the shared
`blockApplyPadding`/`blockRemovePadding` helpers; only TEA's longer "NO"-padding
error message is handled locally). XTEA adds a configurable 1–255 round count.
These operations are **newer than the bundled CyberChef-server oracle** (Crown
Copyright 2026), so vectors were generated by running the checked-out `lib/TEA.mjs`
directly; byte-for-byte verified across both algorithms, all modes/paddings/rounds
and both directions (200-case sweeps each way, including the cases where BIT
padding on an already-aligned block makes upstream itself error).

Note: **XXTEA Encrypt/Decrypt** (Corrected Block TEA) are a from-scratch pure-Go
port of CyberChef's self-contained `lib/XXTEA.mjs` (`internal/ops/xxtea.go`) —
**no new dependency**. Unlike TEA/XTEA it is a variable-length-block cipher
operating on the whole message as little-endian 32-bit words (with the byte length
appended before encryption), so there is no mode or padding; the 128-bit key is
truncated/zero-padded to 16 bytes (CyberChef's `fixk`). The ops are ArrayBuffer
in/out. Verified against the two upstream fixtures and a broad oracle sweep over
data lengths and key sizes/formats (round-trip decryption); decrypting data whose
recovered length word is inconsistent reports CyberChef's "Unable to decrypt using
this key".

Note: **Twofish Encrypt/Decrypt** are a from-scratch pure-Go port of CyberChef's
self-contained `lib/Twofish.mjs` (`internal/ops/twofish.go`) — **no new
dependency**. Twofish is a 128-bit-block AES finalist (Bruce Schneier) taking a
128-, 192- or 256-bit key over 16 Feistel rounds; the engine and both operations
live in one file (the block-cipher convention). All five modes
(ECB/CBC/CFB/OFB/CTR, little-endian CTR) and PKCS5/NO/ZERO/RANDOM/BIT padding are
supported, reusing the shared `blockApplyPadding`/`blockRemovePadding` helpers;
output is continuous (non-spaced) hex. These operations are **newer than the
bundled CyberChef-server oracle**, so authoritative vectors were generated by
running the checked-out `lib/Twofish.mjs` directly: byte-for-byte verified against
the official Twofish-paper ECB vectors plus a broad differential sweep (560 encrypt
+ 512 decrypt comparisons across every mode, all three key sizes, all paddings and
lengths 0–39, including error-path parity where BIT padding on an already-aligned
block makes upstream itself error).

Note: **Typex** is a from-scratch pure-Go port of CyberChef's self-contained
`lib/Typex.mjs` (`internal/ops/typex.go` + `typexmachine.go`) — **no new
dependency**. It is a British WW2 rotor machine built on Enigma, so it reuses the
existing Enigma rotor/reflector engine (`enigma.go`): the machine engine adds
Typex's five rotors (the two right-hand ones static, stepping from index 2), the
reversible rotor cores, the Rotor-based input plugboard (with Typex's mirrored
backwards input wiring, distinct forward/reverse transforms), and the keyboard
emulation that maps shifted symbols/digits to letter sequences (Encrypt/Decrypt
directions). Verified against the three upstream fixtures plus a broad oracle
sweep (240 comparisons across all three keyboard modes, strict on/off, reversed
rotors and random ring/initial settings, including full Encrypt→Decrypt round
trips).

Note: **Vigenère Encode/Decode** are a from-scratch pure-Go port of CyberChef's
self-contained Vigenère operations (`internal/ops/vigenere.go`) — **no new
dependency**. The classic polyalphabetic cipher: each letter is shifted by the
repeating key letter, case is preserved, and non-letters pass through without
advancing the key. The op display names keep the accented "Vigenère"; the CLI
subcommands are the ASCII `vigenere-encode` / `vigenere-decode` (the shared
`core.Kebab` slug helper now folds accented Latin letters to ASCII). Verified
against the upstream Ciphers.mjs fixtures plus a broad oracle sweep (320
comparisons + 160 Encrypt→Decrypt round trips over random keys and mixed
case/punctuation/digit inputs). This completes the Encryption / Encoding category.

Note: **ECDSA Sign / ECDSA Verify / ECDSA Signature Conversion / Generate ECDSA
Key Pair** (Public Key) are a from-scratch port of CyberChef's `jsrsasign`-backed
operations, built entirely on the Go **standard library**
(`crypto/ecdsa`/`elliptic`/`x509`, `encoding/pem`/`asn1`/`base64`) — **no new
dependency** (`internal/ops/ecdsa.go`). The signature-format helpers replicate
jsrsasign's exact string algorithms (ASN.1 ↔ P1363 ↔ JWS base64url ↔ `{r,s}` JSON,
including its curve-length heuristics and Raw-JSON leading-zero preservation), and
Generate matches jsrsasign's PEM/DER(scalar-hex)/JWK output shapes over P-256/384/521.
Verified against the upstream fixtures plus a broad oracle cross-compatibility
sweep (byte-exact Signature Conversion; cchef↔oracle cross-verify across all five
digests incl. SHA-1/MD5; Generate keys interoperable both directions). Sign and
Generate are non-deterministic (random nonce/key), so they are validated by
round-trip and cross-verification rather than byte-equality. One documented
divergence: for the **Hex/Base64 message formats**, CyberChef re-UTF-8-encodes the
decoded bytes (double-encoding bytes ≥ 0x80); cchef hashes the raw decoded bytes,
which matches the oracle for all text and keeps Sign/Verify self-consistent.

Note: **RSA Encrypt / RSA Decrypt / RSA Sign / RSA Verify / Generate RSA Key Pair**
(Public Key) are a from-scratch port of CyberChef's `node-forge`-backed operations,
built entirely on the Go **standard library** (`crypto/rsa`/`x509`,
`encoding/pem`, `math/big`) — **no new dependency** (`internal/ops/rsa.go`). RSA-OAEP,
RSAES-PKCS1-V1_5 and the RAW textbook scheme (reimplemented over `math/big` to match
forge's fixed modulus-width block, byte-exact) are all supported, across the SHA-1/
MD5/SHA-256/384/512 digests. Signing (RSASSA-PKCS1-v1.5) is deterministic and
byte-verified against the oracle; OAEP/PKCS#1 v1.5 encryption is randomized, so it is
validated by round-trip and forge cross-compatibility. Private keys parse from PKCS#1
(incl. legacy PEM encryption) and PKCS#8. Two documented divergences: PKCS#8-encrypted
private keys are unsupported (no stdlib support without a dependency), and the
`Generate → JSON` format uses cchef's own hex-parameter shape rather than forge's
non-portable internal BigInteger serialization (PEM and DER are faithful).

Note: **PGP Encrypt/Decrypt, PGP Sign/Verify, PGP Encrypt and Sign, PGP Decrypt
and Verify, and Generate PGP Key Pair** (7 ops, Public Key) port CyberChef's
`kbpgp`-backed operations onto the maintained `github.com/ProtonMail/go-crypto`
OpenPGP library (`internal/ops/pgp.go`) — a **new dependency** (with a
Cloudflare-backed `circl` transitive dep). kbpgp's exact OpenPGP output is not
reproduced byte-for-byte (ASCII-armor headers and generated-key subkey structure
differ), but full interoperability holds: cchef decrypts/verifies the upstream
fixtures byte-exact (including the signer key ID, fingerprint and signing time in
the verify report), and an oracle sweep confirms bidirectional round-trips
(cchef↔kbpgp encrypt/decrypt, sign/verify, encrypt+sign/decrypt+verify, and
cchef-generated RSA/ECC keys usable by kbpgp). Generate and the encrypt/sign ops
are non-deterministic, so they are validated by round-trip and cross-verification.

### Dependency policy and planned reductions

**Policy.** `golang.org/x/*` modules and libraries with large-organization
backing (e.g. `google.golang.org/protobuf`, `go.yaml.in/yaml`) are acceptable
long-term dependencies. The concern is with modules maintained by individuals —
especially clearly-unmaintained ones (`elobuff/goamf`, last touched 2014) — which
carry supply-chain and bit-rot risk. Those are candidates for in-repo
reimplementation, following the precedent already set by the codepage engine and
the generated ua-parser rule tables (reimplement rather than depend, and
differential-verify byte-for-byte against the CyberChef-server oracle).

**Planned reimplementations** (each to be removed once its Go replacement is
oracle-verified across the same inputs as the current op):

- [ ] `github.com/sergi/go-diff` (**Diff**) — reimplement the diff-match-patch
  Myers diff (`DiffMain`/`DiffMainRunes`) in-repo.
- [ ] `github.com/mmcloughlin/geohash` (**Convert co-ordinate format**) —
  reimplement geohash encode/decode (bit-interleaving; small).
- [ ] `github.com/im7mortal/UTM`, `github.com/klaus-tockloth/coco` (MGRS),
  `github.com/wroge/wgs84` (**Convert co-ordinate format**) — reimplement the
  remaining coordinate math (transverse Mercator, MGRS grid lettering, WGS84/
  OSGB36 Helmert datum transforms); precision-sensitive, verify against the oracle.
- [ ] `github.com/elobuff/goamf` (**AMF Encode/Decode**) — reimplement AMF0/AMF3
  serialization in-repo. Highest priority: unmaintained, and removing it also
  drops its indirect `github.com/jcoene/gologger` dependency.
- [ ] `golang.org/x/text/encoding/charmap` (**MIME Decoding**) — route the
  ISO-8859 decoders through the existing in-repo codepage engine, which already
  covers all 16 ISO-8859 charsets. Note: this removes the `charmap`/`encoding`
  *usage* but **not** the `golang.org/x/text` module, which stays for
  `unicode/norm` (Unicode normalization is out of scope to reimplement — it needs
  the full Unicode Character Database — and `x/text` is an acceptable `x/*` dep).

**Explicitly kept** (reimplementation uneconomical or the module is an acceptable
`x/*`/large-org dependency): `dlclark/regexp2` (backtracking PCRE engine RE2
cannot replace), `google.golang.org/protobuf` + `bufbuild/protocompile` (full
`.proto` compiler), `golang.org/x/text/unicode/norm`, `golang.org/x/crypto`, and
`go.yaml.in/yaml/v3` (already pulled in transitively by cobra).

## Current status

The core engine, recipe/URL machinery, CLI, docs, and a **curated set of 452
operations** are implemented, tested, and documented. The remaining CyberChef
operations are added incrementally against the same interfaces (see the
[Operation implementation status](#operation-implementation-status) checklist
below).

**Done:**

- **Core engine** (`internal/core/`): `Dish` (byte-backed type hub),
  `Operation` interface + `ArgDef`/`ToggleString` ingredient model, self-registering
  `Registry`, sequential `Recipe.Execute`, faithful ports of
  `GeneratePrettyRecipe`/`ParseRecipeConfig` (Chef format) and
  `EncodeURIFragment`/`BuildURL` (share URLs), each with byte-exact tests.
- **452 operations** (`internal/ops/`), each a faithful port with tests
  transcribed from CyberChef's `tests/operations/tests/*.mjs` fixtures.
- **CLI** (`cmd/`): auto-generated per-op subcommands (flags derived from arg
  defs, names sanitised), plus `bake`, `url`, `recipe convert`, `list`. Input
  resolves `--in-file` > `-i/--input` > positional args > stdin; output is
  byte-exact when piped and adds a trailing newline only on a TTY.
- **Docs** (`docs/`): per-category pages with options tables, simple + complex
  examples, and external reference links; operations listed alphabetically.
- **Tooling**: `Makefile` (alphabetised targets: build/test/fmt/vet/lint +
  `sast`/`vuln`/`sec` via gosec + govulncheck + `sbom`/`sbom-scan`/`sbom-audit`
  via cyclonedx-gomod + grype), `.gitignore`. `make test`, `make vet`,
  `make lint`, and `make sec` are clean. gosec's by-design findings (weak-crypto
  ports, bounded byte/bit conversions, CLI file args) carry justified
  `// #nosec` annotations, enforced by `-nosec-require-rules
  -nosec-require-justification` and auditable via `-track-suppressions`.

## Architecture (as built)

```
cchef/
  main.go                      thin entry -> cmd.Execute()
  Makefile  .gitignore  go.mod
  cmd/
    root.go                    root cobra command
    register_ops.go            builds one subcommand per registered op (flags from ArgDefs)
    io.go                      input resolution (--in-file > -i > positional > stdin) + output
    bake.go                    `cchef bake` — run a JSON/Chef recipe (-e expr / -r file)
    url.go                     `cchef url` — emit a CyberChef share URL
    recipe.go                  `cchef recipe convert` — JSON <-> Chef
    list.go                    `cchef list` — list ops by module
  internal/core/
    dish.go        Dish{ data []byte; typ DishType } + conversions
    operation.go   Operation interface, ArgDef, ArgType, ToggleString, arg coercion
    registry.go    Register / Get / All; ops self-register via init()
    recipe.go      RecipeOp{Op,Args,Disabled,Breakpoint}, Recipe.Execute
    chef.go        GeneratePrettyRecipe / ParseRecipeConfig
    url.go         EncodeURIFragment / BuildURL
    naming.go      Kebab (op name -> subcommand)
  internal/ops/
    base64.go base32.go base45.go base58.go base62.go base85.go base92.go
    base_generic.go binary.go decimal.go charcode.go swapendianness.go
    hex.go octal.go urlcode.go xor.go rot.go hashes.go
    case.go reverse.go amf.go sha3.go keccak.go hmac.go adler32.go
    utils_simple.go findreplace.go utils_lines.go utils_slice.go utils_text2.go
    convert.go convert_data.go utils_case.go escapestring.go
    metrics.go unixperms.go arithmetic.go bignum.go bitwise.go sets.go
    ror13.go rotate.go rot8000.go xorbruteforce.go
    extractdates.go filetime.go unixtimestamp.go datetime.go datetimeformat.go
    changeipformat.go dechunkhttp.go netbios.go striphttpheaders.go defang.go
    stripheaders.go formatmac.go ipv6transition.go groupip.go varint.go
    parsenet.go parseudp.go parsetcp.go parseiprange.go parsesshhostkey.go
    parseipv4header.go parseethernetframe.go parseuri.go parseipv6address.go
    parsetlsrecord.go useragent.go useragent_rules.go ipprotocols.go
    fixtures_test.go (+ per-op _test.go)
  docs/
    README.md arithmetic-logic.md data-format.md date-time.md encryption-encoding.md hashing.md networking.md utils.md recipes-and-urls.md
```

## Recipe formats & URLs

- **JSON**: `[{"op":"To Base64","args":["A-Za-z0-9+/="]}, ...]`
- **Chef** (compact): `To_Base64('A-Za-z0-9+/=')To_Hex('Space')`; auto-detected by
  a leading `[`. Optional `/disabled` `/breakpoint` flags.
- **URL**: `https://gchq.github.io/CyberChef/#recipe=<chef>&input=<base64,fragment-encoded>`.

## Usage

```bash
make build                                   # -> ./dist/cchef
echo -n hello | cchef to-base64 | cchef to-hex
cchef rot13 "Have a nice day."               # positional input
cchef bake -e "To_Base64()To_Hex()" -i hello # multi-op recipe
cchef url  -e "ROT13()" -i hello             # share link
cchef list                                   # discover operations
```

## How to add a new operation (the repeatable pattern)

1. Write `internal/ops/<name>_test.go` first, transcribing the matching cases
   from `../CyberChef/tests/operations/tests/<Op>.mjs` into the shared
   `opCase` table runner.
2. Add a stub type implementing `core.Operation` that compiles but fails (`make test` red).
3. Port the `run()` logic from `../CyberChef/src/core/operations/<Op>.mjs` until
   green. Register it in `init()` and add a docs entry in the relevant
   `docs/<category>.md` (alphabetised) with options + examples.

## Proposed reorganisation of `internal/ops`

**Status: proposal — not started.** `internal/ops` is a single flat Go package
that has grown to **615 files / 118k LOC** (322 non-test at 78k LOC, 293 test at
40k LOC) implementing the ~452 registered operations. Nothing about it is broken;
the concern is navigability and build/test granularity. The figures below come
from an AST-level cross-file reference analysis of the package (July 2026) and
will drift as operations are added — re-measure before acting on them.

### What the package looks like today

- **The operation files themselves are healthy.** 246 files declare a `Meta()`:
  151 register a single operation and 69 register two (encode/decode pairs).
  One-op-per-file mirrors CyberChef, keeps the TDD loop tight, and should be
  **kept as-is** — the file count is a symptom of scope, not of bad layout.
- **76 files register nothing.** These are engines and static data totalling
  **25k LOC — roughly a third of the non-test code**: `jsparser_*` (5.0k across
  four files), `htmlentity_tables.go` (2.9k), `jsbeautify_*`, `codepage.go`,
  `xmldom*`, `protobuf*`, `d3*`, `disassemble*`, the Bletchley machines
  (`bombe`, `colossuscomputer`, `lorenzmachine`, `typexmachine`,
  `sigabamachine`), and lookup tables (`snefru_table`, `filesignatures`,
  `useragent_rules`, `exiftags`, `colournames`). They sit alphabetically
  interleaved with forty-line operations.
- **Coupling between files is low.** Across the 322 non-test files there are
  only **505 cross-file references**. 78 files reference nothing else in the
  package, and 194 are referenced by nothing.
- **…but a handful of hubs make it *look* entangled.** The reference graph is one
  weakly-connected component of 254 files, held together by a small shared-helper
  layer: `convertToByteArray` (`xor.go`, 28 referring files), `lo`
  (`utils_case.go`, 19), `parseEscapedChars` (`findreplace.go`, 18), `charRep`
  (`hex.go`, 18), `jsObject`/`jsStringify` (`jsonvalue.go`, 17/13),
  `decodeAESInput` (`aes.go`, 13), `imageTransform` (`imageops.go`, 12).
  Simulating the extraction of the ~177 symbols referenced by two or more files
  breaks the graph into **200 components, the largest of them 16 files**. The
  package is therefore genuinely splittable; it is not a ball of mud.

### The three problems worth fixing

1. **One compilation unit and one test binary.** A one-line edit recompiles 78k
   LOC, and the suite is a single 40k-LOC test binary — no per-package build
   caching and no parallel package-level test execution. Every unexported helper
   is also visible to all 322 files, which is why
   `.claude/skills/add-operation` has to instruct *"before adding a helper,
   `grep` for it — duplicates like `padEnd`/`isHexByte` already exist and will
   fail to compile"*. Package boundaries would enforce that instead of prose.
2. **Category is a side table rather than structure.** `opCategories` in
   `cmd/opmeta.go` hand-maintains 424 entries, policed by
   `TestOpCategoriesMatchRegistry`, and the operation counts are duplicated
   across five places in `PLAN.md`, `README.md` and `docs/README.md` with **no
   test catching drift** — steps 5–7 of the add-operation skill are largely this
   bookkeeping.
3. **Engines and operations carry equal visual weight.** A 2.7k-line JavaScript
   parser is no more prominent in a directory listing than `atbash.go`, and is
   exercised only indirectly through the operations that use it.

### Stage 1 — extract the shared helpers and the engines (prerequisite)

Highest value per unit of risk, and worth doing on its own even if the rest is
never done.

- **`internal/opsutil`** — the cross-cutting helper layer: byte/hex conversion
  (`convertToByteArray`, `charRep`, `toHex`, `splitHexToBytes`, `toHexFast`),
  `expandAlphRange`, escaping (`parseEscapedChars`, `escapeHTML`), the
  JS-semantics shims (`jsNum`, `jsObject`, `jsStringify`, `jsParseInt`,
  `jsonParseOrdered`, `jsFormatNumber`), and the block-cipher mode/padding
  plumbing shared by AES/Blowfish/DES. Despite the ~177-symbol count these live
  in about a dozen files. Each gets a doc comment and direct unit tests instead
  of incidental coverage.
- **One package per engine**, moved out of `internal/ops` entirely:
  `internal/jsparser`, `internal/jsbeautify`, `internal/xmldom`,
  `internal/codepage`, `internal/protobuf`, `internal/d3`, `internal/exif`,
  `internal/disasm`, `internal/bletchley`, plus the pure lookup tables into
  `internal/opsdata` (or alongside their engine). Each then becomes testable on
  its own terms.

This alone removes ~25k LOC from the `ops` namespace and eliminates the
grep-before-you-add-a-helper hazard.

### Stage 2 — one package per CyberChef category

`internal/ops/<category>/` — `encoding`, `hashing`, `dataformat`, `codetidy`,
`networking`, `publickey`, `multimedia`, `utils`, `arithmetic`, `datetime`,
`extractors`, `forensics`, `language`, `other` — mirroring `docs/` and the
grouping already shown by `cchef list`. `internal/ops` itself becomes a thin
aggregator that blank-imports each subpackage, so `cmd/register_ops.go` is
unchanged. The largest package lands at roughly 52 files / 14k LOC (Encryption /
Encoding); most are far smaller.

Two findings make **Stage 1 a hard prerequisite**:

- **A naive split creates import cycles.** Measured mutual references include
  Data format ↔ Public Key, Encryption / Encoding ↔ Utils, Networking ↔ Utils,
  Data format ↔ Networking, Code tidy ↔ Data format and Multimedia ↔ Utils.
  Every one of those runs through the hub symbols that `opsutil` takes, so they
  disappear once Stage 1 lands.
- **19 operation files span more than one category** (e.g. `ascon.go`,
  `bcrypt.go` and `scrypt.go` are Encryption/Encoding + Hashing; `pem.go` and
  `parseasn1hexstring.go` are Data format + Public Key; `urlcode.go` is Data
  format + Networking). Each needs a deliberate home package; the *display*
  categories can still be multiple.

Payoffs beyond navigation:

- Category becomes a **compile-time fact**. `opCategories` and its policing test
  can be derived from package membership (or from a new `Category` field on
  `core.OpMeta` set per subpackage), deleting step 6.1 of the add-operation
  skill.
- Per-package build caching and parallel test execution: editing one operation
  rebuilds and re-tests one small package.
- The shared test harness moves to **`internal/optest`** with `opCase`,
  `runCases` and `runOp` exported, imported by each category's tests in place of
  today's package-private `fixtures_test.go`.

### Stage 3 — generate the counts (independent of the above)

The operation count appears in five places across `PLAN.md`, `README.md` and
`docs/README.md`, and the `docs/README.md` master category table duplicates
`opCategories`. None of it is test-covered. A `go generate` step or a
`make docs-check` target that derives those lines from `core.Default.All()` and
fails CI on drift removes step 7 of the add-operation skill entirely. This is
cheap and can be done first, independently of any file moves.

### Migration notes

- The work is mechanical and incremental: move one category at a time and let
  the compiler find the breakages. `make all` remains the gate at each step.
- Suggested pilot: **Date / Time** (4 operation files) to validate the
  aggregator and `optest` pattern, then **Multimedia** (27 files, but almost all
  of its cross-file references target just `imageops.go` and `imageresize.go`,
  so it moves cleanly once Stage 1 exists).
- Update `.claude/skills/add-operation` in the same change — the "where does the
  file go" step, the `opsutil` helper-reuse note, and whichever of steps 5–7
  become automated.
- Keep one operation per file, and keep the `docs/<category>.md` layout as the
  user-facing mirror of the new package layout.

## Remaining / future work

- **Reorganise `internal/ops`** — see *Proposed reorganisation of
  `internal/ops`* above: extract `internal/opsutil` and the standalone
  engines, then split the package one-per-category. Not started.
- Implement more operations from the checklist below, prioritising common
  Data format / Encryption / Hashing ops.
- Additional Dish types as needed for ops that require them.
  (`JSON` was added for AMF; `BigNumber` for the generic To/From Base.)
- Flow control operations (Fork, Merge, Conditional Jump) need engine support
  beyond the current linear `Recipe.Execute`.
- `CRC Checksum` (parameterised over many algorithms via an argSelector) is
  deferred as a larger-than-straight-port effort.
- **`Hex to PEM` — malformed mixed hex/non-hex input divergence.** The impl
  (`internal/ops/pem.go`, `hexToBase64`) is byte-for-byte faithful for all
  well-formed hexadecimal input (differential-tested 195/195 random cases + the
  fixtures) and is lenient (no error) on stray characters, matching CyberChef for
  the simple cases (`"3g"→Aw==`, `"zz"→AA==`). It does **not** reproduce
  CyberChef's exact output for input that *interleaves* hex and non-hex characters
  (e.g. `"3g3"`, `"1g2h3i"`): jsrsasign routes hex→base64 through CryptoJS's
  `Hex.parse` + `Base64.stringify`, whose 32-bit word packing and fractional-
  `sigBytes` clamp produce garbage bytes we don't emulate. To close it: faithfully
  port that CryptoJS pipeline (emulating JS 32-bit shift/`>>>`/clamp semantics),
  then differential-test against the oracle to 100%. To reverse the algorithm
  precisely, run the original jsrsasign `hextob64`/CryptoJS in node
  (`~/.nvm/versions/node/*/bin/node` — not on `PATH` as bare `node`) against the
  oracle. Low priority — affects only malformed input, garbage-in/garbage-out.
- **`Avro to JSON` — 64-bit longs above 2^53.** The from-scratch OCF decoder
  (`internal/ops/avro.go`) is byte-for-byte faithful to avsc (CyberChef's backing
  library) across ~2000 differential cases — all Avro types, `null`/`deflate`
  codecs, unwrapped **and** ambiguous/wrapped unions, and avsc's lenient
  truncation behaviour. It reads `long` values as exact `int64`; avsc reads them
  into JS numbers (float64), so a `long` **above 2^53** in a file produced by a
  non-JS Avro writer would render with full integer precision here versus avsc's
  lossy value. avsc's own encoder rejects such longs, so this is unreachable for
  avsc-produced files; low priority. Note also that the **CyberChef-server oracle
  needs a one-line patch** to test this op at all (`AvroToJSON.run` must be
  `async`; the upstream CyberChef bug is documented on the `CyberChef` fork's
  `bugfix/node-api-async-promise-ops` branch).
- **Evaluate the file/string input convention.** cchef currently uses explicit
  separate flags (`-i/--input` for a literal string, `--in-file` for a file,
  `--in-dir` for a directory, stdin as fallback), and — unlike most Unix tools —
  treats the positional argument as a *string* rather than a *filename*. Review
  this against common CLI conventions: (1) positional = file + stdin (the
  `cat`/`grep`/`sha256sum` mainstream), (2) curl's `@file` sigil for
  inline-or-file on one flag, (3) `-in`/`-out` file flags (openssl), (4) paired
  `--x`/`--x-file` flags (secrets/12-factor). Decide whether to keep the current
  explicit scheme, add `@file` shorthand on `-i` as a familiar convenience, or
  otherwise nudge closer to convention — without breaking existing usage.
- (Done: a repo-root `README.md` and GitHub Actions CI running fmt/vet/test/lint,
  a gosec + govulncheck security job, plus an SBOM scan now exist.)

## Verification

- `make test` (unit tests, fixtures), `make vet`, `make lint` — all clean.
- End-to-end: `printf 'hello' | ./dist/cchef to-base64` → `aGVsbG8=`;
  `printf 'hello' | ./dist/cchef md5` → `5d41402abc4b2a76b9719d911017c592`;
  `./dist/cchef url -e "To_Hex()" -i hello` → opens the recipe + input in CyberChef.
- **Oracle:** for operations without upstream fixtures, the CyberChef-server
  (Docker, `../CyberChef-server`) gives authoritative output — `docker run -d -p
  3000:3000 cyberchef-server`, then POST `{input, recipe}` to
  `localhost:3000/bake`. Used to derive/differential-test e.g. Escape string and
  To/From Case Insensitive Regex.
- **Future work — standardised test vectors:** for the standardised cryptographic
  algorithms in this repo (AES, DES/Triple DES, SHA family, HMAC, RSA/ECDSA, etc.),
  consider cross-checking against the NIST **Cryptographic Algorithm Validation
  Program** (CAVP) test vectors at
  <https://csrc.nist.gov/projects/cryptographic-algorithm-validation-program> for
  more robust, spec-authoritative coverage than the CyberChef fixtures/oracle
  alone. These are known-answer tests independent of CyberChef, so they would catch
  any place where CyberChef itself diverges from the standard (and confirm the ones
  where we deliberately match CyberChef's behaviour). Note they validate the *core
  algorithm*, not CyberChef's specific option/format plumbing, so keep the oracle
  checks too.

## Operation implementation status

All CyberChef operations, grouped by CyberChef category and listed
alphabetically. `[x]` = implemented in cchef, `[ ]` = not yet, `[—]` = phantom
(named in CyberChef's config but never implemented upstream — see note below).
The per-category count is `implemented/total`; some operations appear in more
than one category.
Currently **449 unique** CyberChef operations are covered (448 directly plus
`SHA2`, exposed as the `sha224`, `sha256`, `sha384` and `sha512` subcommands),
which is where the 452 cchef subcommands come from.

> **495 real operations, not 498.** CyberChef's `Categories.json` names **498**
> operations, but only **495** have a backing `Operation` class. Three names —
> **Extended GCD**, **Modular Exponentiation**, **Modular Inverse** — appear in
> the category config (and a staged `lib/BigIntUtils.mjs` helper, Crown Copyright
> 2025) but were never given operation files, so they don't exist as usable
> CyberChef operations. They are marked `[—]` below and excluded from the
> category totals; there is nothing to port until GCHQ ships them.

### Data format (78/78)

- [x] AMF Decode
- [x] AMF Encode
- [x] Avro to JSON
- [x] Caret/M-decode
- [x] CBOR Decode
- [x] CBOR Encode
- [x] Change IP format
- [x] CSV to JSON
- [x] Decode text
- [x] Encode text
- [x] Escape Smart Characters
- [x] Escape Unicode Characters
- [x] From Base
- [x] From Base32
- [x] From Base45
- [x] From Base58
- [x] From Base62
- [x] From Base64
- [x] From Base85
- [x] From Base92
- [x] From BCD
- [x] From Bech32
- [x] From Binary
- [x] From Braille
- [x] From Charcode
- [x] From Decimal
- [x] From Float
- [x] From Hex
- [x] From Hex Content
- [x] From Hexdump
- [x] From HTML Entity
- [x] From MessagePack
- [x] From Modhex
- [x] From Octal
- [x] From Punycode
- [x] From Quoted Printable
- [x] Hex to PEM
- [x] JSON to CSV
- [x] JSON to YAML
- [x] MIME Decoding
- [x] Normalise Unicode
- [x] Parse ASN.1 hex string
- [x] Parse TLV
- [x] PEM to Hex
- [x] Rison Decode
- [x] Rison Encode
- [x] Show Base64 offsets
- [x] Swap endianness
- [x] Text Encoding Brute Force
- [x] Text-Integer Conversion
- [x] To Base
- [x] To Base32
- [x] To Base45
- [x] To Base58
- [x] To Base62
- [x] To Base64
- [x] To Base85
- [x] To Base92
- [x] To BCD
- [x] To Bech32
- [x] To Binary
- [x] To Braille
- [x] To Charcode
- [x] To Decimal
- [x] To Float
- [x] To Hex
- [x] To Hex Content
- [x] To Hexdump
- [x] To HTML Entity
- [x] To MessagePack
- [x] To Modhex
- [x] To Octal
- [x] To Punycode
- [x] To Quoted Printable
- [x] Unescape Unicode Characters
- [x] URL Decode
- [x] URL Encode
- [x] YAML to JSON

### Encryption / Encoding (94/94)

- [x] A1Z26 Cipher Decode
- [x] A1Z26 Cipher Encode
- [x] AES Decrypt
- [x] AES Encrypt
- [x] AES Key Unwrap
- [x] AES Key Wrap
- [x] Affine Cipher Decode
- [x] Affine Cipher Encode
- [x] Ascon Decrypt
- [x] Ascon Encrypt
- [x] Atbash Cipher
- [x] Bacon Cipher Decode
- [x] Bacon Cipher Encode
- [x] Bcrypt
- [x] Bifid Cipher Decode
- [x] Bifid Cipher Encode
- [x] Blowfish Decrypt
- [x] Blowfish Encrypt
- [x] Bombe
- [x] Caesar Box Cipher
- [x] Cetacean Cipher Decode
- [x] Cetacean Cipher Encode
- [x] ChaCha
- [x] CipherSaber2 Decrypt
- [x] CipherSaber2 Encrypt
- [x] Citrix CTX1 Decode
- [x] Citrix CTX1 Encode
- [x] Colossus
- [x] Derive EVP key
- [x] Derive HKDF key
- [x] Derive PBKDF2 key
- [x] DES Decrypt
- [x] DES Encrypt
- [x] Enigma
- [x] Fernet Decrypt
- [x] Fernet Encrypt
- [x] Flask Session Decode
- [x] Flask Session Sign
- [x] Flask Session Verify
- [x] From Morse Code
- [x] GOST Decrypt
- [x] GOST Encrypt
- [x] GOST Key Unwrap
- [x] GOST Key Wrap
- [x] GOST Sign
- [x] GOST Verify
- [x] JWT Decode
- [x] JWT Sign
- [x] JWT Verify
- [x] Lorenz
- [x] LS47 Decrypt
- [x] LS47 Encrypt
- [x] Multiple Bombe
- [x] PRESENT Decrypt
- [x] PRESENT Encrypt
- [x] Pseudo-Random Number Generator
- [x] Rabbit
- [x] Rail Fence Cipher Decode
- [x] Rail Fence Cipher Encode
- [x] RC2 Decrypt
- [x] RC2 Encrypt
- [x] RC4
- [x] RC4 Drop
- [x] RC6 Decrypt
- [x] RC6 Encrypt
- [x] ROR13
- [x] ROT13
- [x] ROT13 Brute Force
- [x] ROT47
- [x] ROT47 Brute Force
- [x] ROT8000
- [x] Salsa20
- [x] Scrypt
- [x] SIGABA
- [x] SM4 Decrypt
- [x] SM4 Encrypt
- [x] Substitute
- [x] TEA Decrypt
- [x] TEA Encrypt
- [x] To Morse Code
- [x] Triple DES Decrypt
- [x] Triple DES Encrypt
- [x] Twofish Decrypt
- [x] Twofish Encrypt
- [x] Typex
- [x] Vigenère Decode
- [x] Vigenère Encode
- [x] XOR
- [x] XOR Brute Force
- [x] XSalsa20
- [x] XTEA Decrypt
- [x] XTEA Encrypt
- [x] XXTEA Decrypt
- [x] XXTEA Encrypt

### Public Key (31/31)

- [x] ECDSA Sign
- [x] ECDSA Signature Conversion
- [x] ECDSA Verify
- [x] Generate ECDSA Key Pair
- [x] Generate PGP Key Pair
- [x] Generate RSA Key Pair
- [x] Hex to Object Identifier
- [x] Hex to PEM
- [x] JWK to PEM
- [x] Object Identifier to Hex
- [x] Parse ASN.1 hex string
- [x] Parse CSR
- [x] Parse SSH Host Key
- [x] Parse X.509 certificate
- [x] Parse X.509 CRL
- [x] PEM to Hex
- [x] PEM to JWK
- [x] PGP Decrypt
- [x] PGP Decrypt and Verify
- [x] PGP Encrypt
- [x] PGP Encrypt and Sign
- [x] PGP Sign
- [x] PGP Verify
- [x] Public Key from Certificate
- [x] Public Key from Private Key
- [x] RSA Decrypt
- [x] RSA Encrypt
- [x] RSA Sign
- [x] RSA Verify
- [x] SM2 Decrypt
- [x] SM2 Encrypt

### Arithmetic / Logic (27/27)

> Complete. The category names 30 ops, but three — Extended GCD, Modular
> Exponentiation, Modular Inverse — are phantom `Categories.json` entries with no
> CyberChef implementation (see the section note above), so they are excluded
> from the total.

- [x] ADD
- [x] AND
- [x] Bit shift left
- [x] Bit shift right
- [x] Cartesian Product
- [x] Divide
- [—] Extended GCD — phantom Categories.json entry, no CyberChef operation
- [x] Mean
- [x] Median
- [—] Modular Exponentiation — phantom Categories.json entry, no CyberChef operation
- [—] Modular Inverse — phantom Categories.json entry, no CyberChef operation
- [x] Multiply
- [x] NOT
- [x] OR
- [x] Power Set
- [x] ROR13
- [x] ROT13
- [x] ROT8000
- [x] Rotate left
- [x] Rotate right
- [x] Set Difference
- [x] Set Intersection
- [x] Set Union
- [x] Standard Deviation
- [x] SUB
- [x] Subtract
- [x] Sum
- [x] Symmetric Difference
- [x] XOR
- [x] XOR Brute Force

### Networking (38/38)

- [x] Change IP format
- [x] Dechunk HTTP response
- [x] Decode NetBIOS Name
- [x] Defang IP Addresses
- [x] Defang URL
- [x] DNS over HTTPS
- [x] Encode NetBIOS Name
- [x] Fang URL
- [x] Format MAC addresses
- [x] Group IP addresses
- [x] HASSH Client Fingerprint
- [x] HASSH Server Fingerprint
- [x] HTTP request
- [x] IPv6 Transition Addresses
- [x] JA3 Fingerprint
- [x] JA3S Fingerprint
- [x] JA4 Fingerprint
- [x] JA4Server Fingerprint
- [x] Parse Ethernet frame
- [x] Parse IP range
- [x] Parse IPv4 header
- [x] Parse IPv6 address
- [x] Parse SSH Host Key
- [x] Parse TCP
- [x] Parse TLS record
- [x] Parse UDP
- [x] Parse URI
- [x] Parse User Agent
- [x] Protobuf Decode
- [x] Protobuf Encode
- [x] Strip HTTP headers
- [x] Strip IPv4 header
- [x] Strip TCP header
- [x] Strip UDP header
- [x] URL Decode
- [x] URL Encode
- [x] VarInt Decode
- [x] VarInt Encode

### Language (3/7)

- [ ] Convert Leet Speak
- [ ] Convert to NATO alphabet
- [x] Decode text
- [x] Encode text
- [ ] Remove Diacritics
- [x] Unescape Unicode Characters
- [ ] Unicode Text Format

### Utils (52/52)

- [x] Add line numbers
- [x] Alternating Caps
- [x] Convert area
- [x] Convert co-ordinate format
- [x] Convert data units
- [x] Convert distance
- [x] Convert mass
- [x] Convert speed
- [x] Count occurrences
- [x] Diff
- [x] Drop bytes
- [x] Drop nth bytes
- [x] Escape string
- [x] Expand alphabet range
- [x] File Tree
- [x] Filter
- [x] Find / Replace
- [x] From Case Insensitive Regex
- [x] Fuzzy Match
- [x] Get All Casings
- [x] Hamming Distance
- [x] Head
- [x] Levenshtein Distance
- [x] Offset checker
- [x] Pad lines
- [x] Parse colour code
- [x] Parse ObjectID timestamp
- [x] Parse UNIX file permissions
- [x] Pseudo-Random Number Generator
- [x] Regular expression
- [x] Remove ANSI Escape Codes
- [x] Remove line numbers
- [x] Remove null bytes
- [x] Remove whitespace
- [x] Reverse
- [x] Show on map
- [x] Shuffle
- [x] Sleep
- [x] Sort
- [x] Split
- [x] Swap case
- [x] Swap endianness
- [x] Tail
- [x] Take bytes
- [x] Take nth bytes
- [x] To Case Insensitive Regex
- [x] To Lower case
- [x] To Table
- [x] To Upper case
- [x] Unescape string
- [x] Unique
- [x] Wrap

### Date / Time (10/10)

- [x] DateTime Delta
- [x] Extract dates
- [x] From UNIX Timestamp
- [x] Get Time
- [x] Parse DateTime
- [x] Sleep
- [x] To UNIX Timestamp
- [x] Translate DateTime Format
- [x] UNIX Timestamp to Windows Filetime
- [x] Windows Filetime to UNIX Timestamp

### Extractors (14/20)

- [x] CSS selector
- [x] Extract Audio Metadata
- [x] Extract dates
- [ ] Extract domains
- [ ] Extract email addresses
- [x] Extract EXIF
- [ ] Extract file paths
- [x] Extract Files
- [ ] Extract hashes
- [x] Extract ID3
- [x] Extract IP addresses
- [ ] Extract MAC addresses
- [ ] Extract URLs
- [x] JPath expression
- [x] Jsonata Query
- [x] RAKE
- [x] Regular expression
- [x] Strings
- [x] Template
- [x] XPath expression

### Compression (0/19)

- [ ] Bzip2 Compress
- [ ] Bzip2 Decompress
- [ ] Gunzip
- [ ] Gzip
- [ ] LZ4 Compress
- [ ] LZ4 Decompress
- [ ] LZMA Compress
- [ ] LZMA Decompress
- [ ] LZNT1 Decompress
- [ ] LZString Compress
- [ ] LZString Decompress
- [ ] Raw Deflate
- [ ] Raw Inflate
- [ ] Tar
- [ ] Untar
- [ ] Unzip
- [ ] Zip
- [ ] Zlib Deflate
- [ ] Zlib Inflate

### Hashing (50/50)

- [x] Adler-32 Checksum
- [x] Analyse hash
- [x] Argon2
- [x] Argon2 compare
- [x] Ascon Hash
- [x] Ascon MAC
- [x] Bcrypt
- [x] Bcrypt compare
- [x] Bcrypt parse
- [x] BLAKE2b
- [x] BLAKE2s
- [x] BLAKE3
- [x] CMAC
- [x] Compare CTPH hashes
- [x] Compare SSDEEP hashes
- [x] CRC Checksum
- [x] CTPH
- [x] Fletcher-16 Checksum
- [x] Fletcher-32 Checksum
- [x] Fletcher-64 Checksum
- [x] Fletcher-8 Checksum
- [x] Generate all checksums
- [x] Generate all hashes
- [x] GOST Hash
- [x] HAS-160
- [x] HMAC
- [x] Keccak
- [x] LM Hash
- [x] Luhn Checksum
- [x] MD2
- [x] MD4
- [x] MD5
- [x] MD6
- [x] MurmurHash3
- [x] NT Hash
- [x] Parity Bit
- [x] RIPEMD
- [x] Scrypt
- [x] SHA0
- [x] SHA1
- [x] SHA2 — sha224 / sha256 / sha384 / sha512 subcommands
- [x] SHA3
- [x] Shake
- [x] SM3
- [x] Snefru
- [x] SSDEEP
- [x] Streebog
- [x] TCP/IP Checksum
- [x] Whirlpool
- [x] XOR Checksum

### Code tidy (30/30)

- [x] BSON deserialise
- [x] BSON serialise
- [x] CSS Beautify
- [x] CSS Minify
- [x] CSS selector
- [x] Diff
- [x] From MessagePack
- [x] Generic Code Beautify
- [x] JavaScript Beautify
- [x] JavaScript Minify
- [x] JavaScript Parser
- [x] JPath expression
- [x] Jq
- [x] JSON Beautify
- [x] JSON Minify
- [x] Microsoft Script Decoder
- [x] PHP Deserialize
- [x] PHP Serialize
- [x] Render Markdown
- [x] SQL Beautify
- [x] SQL Minify
- [x] Strip HTML tags
- [x] Syntax highlighter
- [x] To Camel case
- [x] To Kebab case
- [x] To MessagePack
- [x] To Snake case
- [x] XML Beautify
- [x] XML Minify
- [x] XPath expression

### Forensics (5/12)

- [x] Detect File Type
- [ ] ELF Info
- [x] Extract Audio Metadata
- [x] Extract EXIF
- [x] Extract Files
- [ ] Extract LSB
- [ ] Extract RGBA
- [ ] Randomize Colour Palette
- [x] Remove EXIF
- [ ] Scan for Embedded Files
- [ ] View Bit Plane
- [ ] YARA Rules

### Multimedia (29/29)

> **CLI presentation:** CyberChef's Multimedia ops preview their result in the
> browser (`presentType: "html"`). cchef has no browser, so the byte-emitting
> ops (Render Image, Play Media, Render PDF) validate the input and pass the
> bytes through — saved with `-o`/redirect — and add a cchef-specific `Output`
> option (`--output-format`): `Raw` (default), `Base64` (a `data:` URI), and, for
> Render Image, `Terminal` (an inline preview via the iTerm2 protocol for any
> format or the kitty protocol for PNG; broader kitty and sixel arrive with the
> imaging batch, which brings an image decoder). `Detect File Type` (Forensics)
> was ported alongside the shared `FileSignatures` engine to enable a
> detect-then-render workflow.
>
> **Imaging (Flip/Rotate/Invert/Image Filter/Image Opacity, and later batches):**
> CyberChef runs these over the Jimp library. cchef decodes with the standard
> library plus `golang.org/x/image` (BMP/TIFF/WEBP) and reproduces **Jimp's pixel
> maths from scratch** on an `image.NRGBA`, so lossless formats (PNG/BMP/TIFF) are
> pixel-identical to CyberChef — only the encoded bytes differ. This is more
> faithful and lighter than wrapping a Go imaging library. Reduced-fidelity
> corners: **JPEG** output (lossy re-encode) and **Rotate** by non-multiples of 90°
> (CyberChef upscales first; cchef matches output dimensions, not every pixel).
> Fidelity is verified against Jimp run directly under Node, since the bundled
> CyberChef-server oracle cannot execute the image module.
>
> **Text rendering:** Add Text To Image draws from the same four 72px Roboto
> BMFont atlases CyberChef bundles, vendored into `internal/ops/bmfonts/` (~320KB,
> Apache-2.0, credited in the README) and embedded with `go:embed`. Reusing the
> bitmap atlases rather than rasterising a TTF keeps the rendered text
> **pixel-identical** to CyberChef, including Jimp's quirk of sizing the text
> bitmap at one line per word plus one. No new Go dependency.
>
> **Charts:** the four chart operations emit **SVG text**. CyberChef draws them
> with d3 + d3-hexbin into `nodom`, a fake DOM; cchef ports the parts of d3-scale,
> d3-array, d3-axis, d3-format, d3-color and d3-hexbin they use, so scales, tick
> selection and placement, CIELAB colour ramps and hexagonal binning are
> **byte-identical**. Verified against goldens produced by running the real
> operations under Node (`internal/ops/testdata/chart_*.svg`); the upstream
> fixtures only assert `/^<svg width/`, so they pin nothing.
>
> Three mechanical deviations make the output valid, safe, standalone SVG, and the
> goldens are normalised by exactly these before comparison: D3's `__data__`
> bindings are not serialised (nodom leaks them as attributes carrying unescaped
> input — upstream added this fix to Series chart only), the clip element is
> spelled `clipPath` rather than nodom's `clippath` (not an SVG element, so the
> clip is silently ignored by real renderers), and the root carries
> `xmlns="http://www.w3.org/2000/svg"`, without which a saved `.svg` does not
> render at all. The namespace goes after `viewBox` so the document still opens
> `<svg width=`, which upstream's own fixture asserts.
>
> Two portability details the ports had to pin: Go and V8 disagree by an ulp or
> two on `sin`/`cos` for the fixed hexagon angles, so those six values are
> tabulated rather than computed; and Go on arm64 fuses multiply-then-add into an
> FMA, rounding once where JavaScript rounds twice, so the interpolation sites
> force the intermediate rounding with an explicit `float64()` conversion.
>
> **OCR is the one operation cchef does not perform itself.** CyberChef runs
> Tesseract compiled to WebAssembly in the browser; it throws outright outside
> one, so there is no oracle and no fixtures. Every cgo-free Go route was
> examined: `gosseract` needs cgo, which would cost the static single-binary
> build; `gogosseract` (Tesseract WASM under wazero) is unmaintained, broken
> against wazero 1.8.0+, and ~6x slower; no Tesseract WASI build exists, so
> hosting the module means owning Emscripten glue — the burden that killed
> gogosseract; and no maintained pure-Go OCR engine exists. Embedding the WASM
> plus `eng.traineddata` would also add ~13MB to a 40MB binary. cchef therefore
> drives the installed `tesseract` binary, its only external tool: the same
> engine and language data, so the text matches, and the binary stays pure Go
> with no cgo. A clear, actionable error is returned when it is absent, and no
> other operation is affected.
>
> **Multi-file output:** Split Colour Channels is CyberChef's only Multimedia op
> with a `List<File>` output. `core.Dish` gained a `TypeFileList` holding named
> files; such a dish is terminal (it cannot convert back to bytes, so the op must
> be a recipe's last step) and the CLI writes it into `--out-dir` — which for
> these ops no longer requires `--in-dir`. With `--in-dir`, each input's files go
> into their own subdirectory named after the input so they cannot collide.

- [x] Add Text To Image
- [x] Blur Image
- [x] Contain Image
- [x] Convert Image Format
- [x] Cover Image
- [x] Crop Image
- [x] Dither Image
- [x] Extract EXIF
- [x] Flip Image
- [x] Generate Image
- [x] Heatmap chart
- [x] Hex Density chart
- [x] Image Brightness / Contrast
- [x] Image Filter
- [x] Image Hue/Saturation/Lightness
- [x] Image Opacity
- [x] Invert Image
- [x] Normalise Image
- [x] Optical Character Recognition
- [x] Play Media
- [x] Remove EXIF
- [x] Render Image
- [x] Render PDF
- [x] Resize Image
- [x] Rotate Image
- [x] Scatter chart
- [x] Series chart
- [x] Sharpen Image
- [x] Split Colour Channels

### Other (22/22)

- [x] Analyse UUID
- [x] Automated Validation Test Op
- [x] Chi Square
- [x] Disassemble ARM
- [x] Disassemble x86
- [x] Entropy
- [x] Frequency distribution
- [x] Generate De Bruijn Sequence
- [x] Generate HOTP
- [x] Generate Lorem Ipsum
- [x] Generate QR Code
- [x] Generate TOTP
- [x] Generate UUID
- [x] Haversine distance
- [x] HTML To Text
- [x] Index of Coincidence
- [x] Numberwang
- [x] P-list Viewer
- [x] Parse QR Code
- [x] Pseudo-Random Integer Generator
- [x] Pseudo-Random Number Generator
- [x] XKCD Random Number

### Flow control (0/10)

- [ ] Comment
- [ ] Conditional Jump
- [ ] Fork
- [ ] Jump
- [ ] Label
- [ ] Magic
- [ ] Merge
- [ ] Register
- [ ] Return
- [ ] Subsection