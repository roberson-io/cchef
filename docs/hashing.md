# Hashing

Cryptographic hash functions and checksums. Each operation takes input and
outputs the lower-case hexadecimal digest. Most take no options (SHA3 and HMAC
are the exceptions).

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Adler-32 Checksum | `adler-32-checksum` | [Adler-32](https://wikipedia.org/wiki/Adler-32) |
| Bcrypt | `bcrypt` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| Bcrypt compare | `bcrypt-compare` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| Bcrypt parse | `bcrypt-parse` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| GOST Hash | `gost-hash` | [GOST (hash function)](https://wikipedia.org/wiki/GOST_(hash_function)) |
| HAS-160 | `has-160` | [HAS-160](https://wikipedia.org/wiki/HAS-160) |
| HMAC | `hmac` | [HMAC](https://wikipedia.org/wiki/HMAC) |
| Keccak | `keccak` | [SHA-3 / Keccak](https://wikipedia.org/wiki/SHA-3) |
| MD2 | `md2` | [MD2](https://wikipedia.org/wiki/MD2_(cryptography)) |
| MD4 | `md4` | [MD4](https://wikipedia.org/wiki/MD4) |
| MD5 | `md5` | [MD5](https://wikipedia.org/wiki/MD5) |
| RIPEMD | `ripemd` | [RIPEMD](https://wikipedia.org/wiki/RIPEMD) |
| Scrypt | `scrypt` | [Scrypt](https://wikipedia.org/wiki/Scrypt) |
| SHA0 | `sha0` | [SHA-0](https://wikipedia.org/wiki/SHA-1#SHA-0) |
| SHA1 | `sha1` | [SHA-1](https://wikipedia.org/wiki/SHA-1) |
| SHA224 | `sha224` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA256 | `sha256` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA3 | `sha3` | [SHA-3](https://wikipedia.org/wiki/SHA-3) |
| SHA384 | `sha384` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA512 | `sha512` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| Snefru | `snefru` | [Snefru](https://wikipedia.org/wiki/Snefru) |
| Whirlpool | `whirlpool` | [Whirlpool](https://wikipedia.org/wiki/Whirlpool_(hash_function)) |

> **Note:** MD5 and SHA1 are not collision resistant and should not be used for
> security-sensitive purposes such as signatures or certificates. They remain
> useful for checksums and interoperability.

---

## Adler-32 Checksum

Computes the Adler-32 checksum, output as an 8-digit hex string. Takes no options.

```bash
cchef adler-32-checksum -i 'Wikipedia'
```

Output:

```
11e60398
```

## Bcrypt

Generates a bcrypt password hash. See the detailed entry under
[Encryption / Encoding](encryption-encoding.md#bcrypt).

## Bcrypt compare

Tests whether the input password matches a given bcrypt hash, returning
`Match: <password>` or `No match`. A malformed hash yields an error describing the
problem (e.g. an invalid salt version).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--hash` | string | (empty) | The bcrypt hash to test the input against. |

**Simple example**

```bash
cchef bcrypt-compare -i "dolphin" --hash '$2a$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK'
```

Output:

```
Match: dolphin
```

A non-matching password returns `No match`:

```bash
cchef bcrypt-compare -i "wrong" --hash '$2a$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK'
```

Output:

```
No match
```

## Bcrypt parse

Parses a bcrypt hash into its components: the cost (rounds), the salt, the
password-hash portion, and the full hash. Takes no options.

```bash
cchef bcrypt-parse -i '$2a$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK'
```

Output:

```
Rounds: 10
Salt: $2a$10$qyon0LQCmMxpFFjwWH6Qh.
Password hash: dDdhqntQh./IN0RXCc3XIMILuOYZKgK
Full hash: $2a$10$qyon0LQCmMxpFFjwWH6Qh.dDdhqntQh./IN0RXCc3XIMILuOYZKgK
```

## GOST Hash

Reference: [GOST (hash function)](https://wikipedia.org/wiki/GOST_(hash_function))

Computes a GOST cryptographic hash. Two algorithms are offered: the original
256-bit GOST R 34.11-94 (built on the GOST 28147-89 block cipher, selectable
paramset S-box) and the newer GOST R 34.11-2012 "Streebog" (256- or 512-bit).
Output is lowercase hex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--algorithm` | option | `GOST 28147 (1994)` | `GOST 28147 (1994)` or `GOST R 34.11 (Streebog, 2012)`. |
| `--digest-length` | option | `256` | Streebog digest size: `256` or `512` (ignored for the 1994 hash). |
| `--sbox` | option | `E-TEST` | Paramset S-box for the 1994 hash: `E-TEST`, `E-A`…`E-Z`, `D-TEST`, `D-A`, `D-SC`. |

**Simple example (Streebog-256)**

```bash
cchef gost-hash -i "The quick brown fox" --algorithm "GOST R 34.11 (Streebog, 2012)" --digest-length 256
```

Output:

```
2a47e26fb8fd4b46668fb8835b3f8966a692ad062d17398a907f025ba4762aa7
```

**GOST R 34.11-94 (E-A S-box)**

```bash
cchef gost-hash -i "The quick brown fox" --algorithm "GOST 28147 (1994)" --sbox E-A
```

Output:

```
17d0212fab4ede36ca5c302bcb3bebd675324d3cfe04122dbb97f7cd1f6bf9e6
```

## HAS-160

The Korean HAS-160 hash (used with the KCDSA signature algorithm); a 160-bit,
SHA-1-like function.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rounds` | number | `80` | Number of rounds (1-80). |

**Simple example**

```bash
cchef has-160 -i Hello
```

Output:

```
3bca9a7d61c6107e88f9c4fcb2728cc7e4fc13ac
```

## HMAC

Computes a keyed-hash message authentication code (HMAC).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted according to `--key-type`. |
| `--key-type` | option | `Hex` | How to interpret the key: `Hex`, `Decimal`, `Base64`, `UTF8`, `Latin1`. |
| `--hashing-function` | option | `SHA256` | One of `SHA256`, `MD5`, `SHA1`, `SHA224`, `SHA384`, `SHA512`. |

**Simple example**

```bash
cchef hmac --key test --key-type Latin1 --hashing-function SHA256 -i 'Hello, World!'
```

Output:

```
52589bd80ccfa4acbb3f9512dfaf4f700fa5195008aae0b77a9e47dcca75beac
```

## Keccak

Computes the **legacy** Keccak digest at the selected size. Keccak predates and
differs from the standardised SHA-3 (FIPS 202) — they use different padding, so
the digests differ. Keccak-256 is the hash used by Ethereum (e.g.
`keccak256("")` = `c5d2460186f7233c…`).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | option | `512` | Output size in bits: `512`, `384`, `256`, or `224`. |

**Simple example**

```bash
cchef keccak --size 256 -i 'Hello, World!'
```

Output:

```
acaf3289d7b601cbd114fb36c4d29c85bbfd5e133f14cb355c3fd8d99367964f
```

## MD2

The 128-bit MD2 message digest (RFC 1319). The `--rounds` argument mirrors
crypto-api, including its quirk that a value of `0` is treated as the default 18.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rounds` | number | `18` | Number of rounds. |

**Simple example**

```bash
cchef md2 -i Hello
```

Output:

```
b27af65e6a4096536dd1252e308c2427
```

## MD4

The 128-bit MD4 message digest (RFC 1320).

```bash
cchef md4 -i Hello
```

Output:

```
a58fc871f5f68e4146474ac1e2f07419
```

## MD5

```bash
cchef md5 -i 'Hello, World!'
```

Output:

```
65a8e27d8879283831b664bd8b7f0ad4
```

## RIPEMD

The RIPEMD family of hashes, at one of four output sizes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | option | `320` | Output size in bits: `320`, `256`, `160`, or `128`. |

**Simple example**

```bash
cchef ripemd --size 160 -i Hello
```

Output:

```
d44426aca8ae0a69cdbc4021c64fa5ad68ca32fe
```

## Scrypt

Derives a key from a password using the scrypt PBKDF. See the detailed entry
under [Encryption / Encoding](encryption-encoding.md#scrypt).

## SHA0

The withdrawn 1993 SHA-0 (the original SHA, before SHA-1's message-schedule fix).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rounds` | number | `80` | Number of rounds (>= 16). |

**Simple example**

```bash
cchef sha0 -i Hello
```

Output:

```
d7f56f62cde2a044d0259adf01953bbb8f971a33
```

## SHA1

```bash
cchef sha1 -i 'Hello, World!'
```

Output:

```
0a0a9f2a6772942557ab5355d76af442f8f65e01
```

## SHA224

```bash
cchef sha224 -i 'Hello, World!'
```

Output:

```
72a23dfa411ba6fde01dbfabf3b00a709c93ebf273dc29e2d8b261ff
```

## SHA256

```bash
cchef sha256 -i 'Hello, World!'
```

Output:

```
dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
```

## SHA3

Computes a SHA-3 (FIPS 202) digest at the selected output size. Note: SHA-3 is
not the same as legacy/Ethereum Keccak — they use different padding.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | option | `512` | Output size in bits: `512`, `384`, `256`, or `224`. |

**Simple example**

```bash
cchef sha3 --size 256 -i 'Hello, World!'
```

Output:

```
1af17a664e3fa8e419b8ba05c2a173169df76162a5a286e0c405b460d478f7ef
```

## SHA384

```bash
cchef sha384 -i 'Hello, World!'
```

Output:

```
5485cc9b3365b4305dfb4e8337e0a598a574f8242bf17289e0dd6c20a3cd44a089de16ab4ab308f63e44b1170eb5f515
```

## SHA512

```bash
cchef sha512 -i 'Hello, World!'
```

Output:

```
374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387
```

---

### Hashing a file

Because input can come from a file, hashing a file's contents is straightforward:

```bash
cchef sha256 --in-file ./document.pdf
```

## Snefru

The Snefru hash (Merkle, 1990) at a configurable output length and round count.
Note: this follows crypto-api, which uses a 64-bit length field rather than the
original design, so output differs from the reference vectors.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | number | `128` | Output length in bits (32-480, step 32). |
| `--rounds` | option | `8` | Number of main-pass rounds: `8`, `4` or `2`. |

**Simple example**

```bash
cchef snefru --size 256 -i Hello
```

Output:

```
bd456c6c33df28257b8736f798e40ac57d9b61996d94ada339abaa8d2a97ec86
```

## Whirlpool

The 512-bit Whirlpool hash and its two earlier variants. Note: this follows
crypto-api, which uses a 64-bit length field rather than Whirlpool's 256-bit
one, so output differs from the ISO reference vectors.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--variant` | option | `Whirlpool` | `Whirlpool` (2003), `Whirlpool-T` (2001) or `Whirlpool-0` (2000). |
| `--rounds` | number | `10` | Number of rounds (1-10). |

**Simple example**

```bash
cchef whirlpool -i Hello
```

Output:

```
00acca7b4456c52a74c589d668b48e1b3d33c9620a0a9b61635111aa92ed8488f21372e27b2122735e561491f8050ed2775a6fb55f7f8b24075d1166bf326bca
```
