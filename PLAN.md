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

Note: **Parse User Agent** is a faithful port of `ua-parser-js` **2.0.10** (the
exact version the CyberChef-server oracle runs). Its rule tables
(`internal/ops/useragent_rules.go`) are *generated* from that library's source and
differential-tested against it; no runtime dependency on the JS library is added.

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

The core engine, recipe/URL machinery, CLI, docs, and a **curated set of 275
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
- **275 operations** (`internal/ops/`), each a faithful port with tests
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

## Remaining / future work

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

## Operation implementation status

All CyberChef operations, grouped by CyberChef category and listed
alphabetically. `[x]` = implemented in cchef, `[ ]` = not yet, `[—]` = phantom
(named in CyberChef's config but never implemented upstream — see note below).
The per-category count is `implemented/total`; some operations appear in more
than one category.
Currently **272 unique** CyberChef operations are covered (271 directly plus
`SHA2`, exposed as the `sha256` and `sha512` subcommands).

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

### Encryption / Encoding (61/94)

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
- [ ] LS47 Decrypt
- [ ] LS47 Encrypt
- [x] Multiple Bombe
- [ ] PRESENT Decrypt
- [ ] PRESENT Encrypt
- [x] Pseudo-Random Number Generator
- [ ] Rabbit
- [ ] Rail Fence Cipher Decode
- [ ] Rail Fence Cipher Encode
- [ ] RC2 Decrypt
- [ ] RC2 Encrypt
- [ ] RC4
- [ ] RC4 Drop
- [ ] RC6 Decrypt
- [ ] RC6 Encrypt
- [x] ROR13
- [x] ROT13
- [ ] ROT13 Brute Force
- [x] ROT47
- [ ] ROT47 Brute Force
- [x] ROT8000
- [ ] Salsa20
- [ ] Scrypt
- [ ] SIGABA
- [ ] SM4 Decrypt
- [ ] SM4 Encrypt
- [ ] Substitute
- [ ] TEA Decrypt
- [ ] TEA Encrypt
- [x] To Morse Code
- [x] Triple DES Decrypt
- [x] Triple DES Encrypt
- [ ] Twofish Decrypt
- [ ] Twofish Encrypt
- [ ] Typex
- [ ] Vigenère Decode
- [ ] Vigenère Encode
- [x] XOR
- [x] XOR Brute Force
- [ ] XSalsa20
- [ ] XTEA Decrypt
- [ ] XTEA Encrypt
- [ ] XXTEA Decrypt
- [ ] XXTEA Encrypt

### Public Key (4/31)

- [ ] ECDSA Sign
- [ ] ECDSA Signature Conversion
- [ ] ECDSA Verify
- [ ] Generate ECDSA Key Pair
- [ ] Generate PGP Key Pair
- [ ] Generate RSA Key Pair
- [ ] Hex to Object Identifier
- [x] Hex to PEM
- [ ] JWK to PEM
- [ ] Object Identifier to Hex
- [x] Parse ASN.1 hex string
- [ ] Parse CSR
- [x] Parse SSH Host Key
- [ ] Parse X.509 certificate
- [ ] Parse X.509 CRL
- [x] PEM to Hex
- [ ] PEM to JWK
- [ ] PGP Decrypt
- [ ] PGP Decrypt and Verify
- [ ] PGP Encrypt
- [ ] PGP Encrypt and Sign
- [ ] PGP Sign
- [ ] PGP Verify
- [ ] Public Key from Certificate
- [ ] Public Key from Private Key
- [ ] RSA Decrypt
- [ ] RSA Encrypt
- [ ] RSA Sign
- [ ] RSA Verify
- [ ] SM2 Decrypt
- [ ] SM2 Encrypt

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

### Extractors (2/20)

- [ ] CSS selector
- [ ] Extract Audio Metadata
- [x] Extract dates
- [ ] Extract domains
- [ ] Extract email addresses
- [ ] Extract EXIF
- [ ] Extract file paths
- [ ] Extract Files
- [ ] Extract hashes
- [ ] Extract ID3
- [ ] Extract IP addresses
- [ ] Extract MAC addresses
- [ ] Extract URLs
- [ ] JPath expression
- [ ] Jsonata Query
- [ ] RAKE
- [x] Regular expression
- [ ] Strings
- [ ] Template
- [ ] XPath expression

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

### Hashing (18/50)

- [x] Adler-32 Checksum
- [ ] Analyse hash
- [ ] Argon2
- [ ] Argon2 compare
- [ ] Ascon Hash
- [ ] Ascon MAC
- [x] Bcrypt
- [x] Bcrypt compare
- [x] Bcrypt parse
- [ ] BLAKE2b
- [ ] BLAKE2s
- [ ] BLAKE3
- [ ] CMAC
- [ ] Compare CTPH hashes
- [ ] Compare SSDEEP hashes
- [ ] CRC Checksum
- [ ] CTPH
- [ ] Fletcher-16 Checksum
- [ ] Fletcher-32 Checksum
- [ ] Fletcher-64 Checksum
- [ ] Fletcher-8 Checksum
- [ ] Generate all checksums
- [ ] Generate all hashes
- [x] GOST Hash
- [x] HAS-160
- [x] HMAC
- [x] Keccak
- [ ] LM Hash
- [ ] Luhn Checksum
- [x] MD2
- [x] MD4
- [x] MD5
- [ ] MD6
- [ ] MurmurHash3
- [ ] NT Hash
- [ ] Parity Bit
- [x] RIPEMD
- [ ] Scrypt
- [x] SHA0
- [x] SHA1
- [x] SHA2 — sha224 / sha256 / sha384 / sha512 subcommands
- [x] SHA3
- [ ] Shake
- [ ] SM3
- [x] Snefru
- [ ] SSDEEP
- [ ] Streebog
- [ ] TCP/IP Checksum
- [x] Whirlpool
- [ ] XOR Checksum

### Code tidy (3/30)

- [ ] BSON deserialise
- [ ] BSON serialise
- [ ] CSS Beautify
- [ ] CSS Minify
- [ ] CSS selector
- [x] Diff
- [x] From MessagePack
- [ ] Generic Code Beautify
- [ ] JavaScript Beautify
- [ ] JavaScript Minify
- [ ] JavaScript Parser
- [ ] JPath expression
- [ ] Jq
- [ ] JSON Beautify
- [ ] JSON Minify
- [ ] Microsoft Script Decoder
- [ ] PHP Deserialize
- [ ] PHP Serialize
- [ ] Render Markdown
- [ ] SQL Beautify
- [ ] SQL Minify
- [ ] Strip HTML tags
- [ ] Syntax highlighter
- [ ] To Camel case
- [ ] To Kebab case
- [x] To MessagePack
- [ ] To Snake case
- [ ] XML Beautify
- [ ] XML Minify
- [ ] XPath expression

### Forensics (0/12)

- [ ] Detect File Type
- [ ] ELF Info
- [ ] Extract Audio Metadata
- [ ] Extract EXIF
- [ ] Extract Files
- [ ] Extract LSB
- [ ] Extract RGBA
- [ ] Randomize Colour Palette
- [ ] Remove EXIF
- [ ] Scan for Embedded Files
- [ ] View Bit Plane
- [ ] YARA Rules

### Multimedia (0/29)

- [ ] Add Text To Image
- [ ] Blur Image
- [ ] Contain Image
- [ ] Convert Image Format
- [ ] Cover Image
- [ ] Crop Image
- [ ] Dither Image
- [ ] Extract EXIF
- [ ] Flip Image
- [ ] Generate Image
- [ ] Heatmap chart
- [ ] Hex Density chart
- [ ] Image Brightness / Contrast
- [ ] Image Filter
- [ ] Image Hue/Saturation/Lightness
- [ ] Image Opacity
- [ ] Invert Image
- [ ] Normalise Image
- [ ] Optical Character Recognition
- [ ] Play Media
- [ ] Remove EXIF
- [ ] Render Image
- [ ] Render PDF
- [ ] Resize Image
- [ ] Rotate Image
- [ ] Scatter chart
- [ ] Series chart
- [ ] Sharpen Image
- [ ] Split Colour Channels

### Other (1/22)

- [ ] Analyse UUID
- [ ] Automated Validation Test Op
- [ ] Chi Square
- [ ] Disassemble ARM
- [ ] Disassemble x86
- [ ] Entropy
- [ ] Frequency distribution
- [ ] Generate De Bruijn Sequence
- [ ] Generate HOTP
- [ ] Generate Lorem Ipsum
- [ ] Generate QR Code
- [ ] Generate TOTP
- [ ] Generate UUID
- [ ] Haversine distance
- [ ] HTML To Text
- [ ] Index of Coincidence
- [ ] Numberwang
- [ ] P-list Viewer
- [ ] Parse QR Code
- [ ] Pseudo-Random Integer Generator
- [x] Pseudo-Random Number Generator
- [ ] XKCD Random Number

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