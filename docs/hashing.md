# Hashing

Cryptographic hash functions and checksums. Most take input and output a
lower-case hexadecimal digest; a few of the checksums (Luhn Checksum, Parity Bit)
emit their own text format, and several operations take options.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Adler-32 Checksum | `adler-32-checksum` | [Adler-32](https://wikipedia.org/wiki/Adler-32) |
| Analyse hash | `analyse-hash` | [Hash functions](https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions) |
| Argon2 | `argon2` | [Argon2](https://wikipedia.org/wiki/Argon2) |
| Argon2 compare | `argon2-compare` | [Argon2](https://wikipedia.org/wiki/Argon2) |
| Ascon Hash | `ascon-hash` | [Ascon](https://wikipedia.org/wiki/Ascon_(cipher)) |
| Ascon MAC | `ascon-mac` | [Ascon](https://wikipedia.org/wiki/Ascon_(cipher)) |
| BLAKE2b | `blake2b` | [BLAKE](https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2b_algorithm) |
| BLAKE2s | `blake2s` | [BLAKE](https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2) |
| Bcrypt | `bcrypt` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| Bcrypt compare | `bcrypt-compare` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| Bcrypt parse | `bcrypt-parse` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| CMAC | `cmac` | [CMAC](https://wikipedia.org/wiki/CMAC) |
| CRC Checksum | `crc-checksum` | [CRC](https://wikipedia.org/wiki/Cyclic_redundancy_check) |
| Fletcher-16 Checksum | `fletcher-16-checksum` | [Fletcher](https://wikipedia.org/wiki/Fletcher%27s_checksum) |
| Fletcher-32 Checksum | `fletcher-32-checksum` | [Fletcher](https://wikipedia.org/wiki/Fletcher%27s_checksum) |
| Fletcher-64 Checksum | `fletcher-64-checksum` | [Fletcher](https://wikipedia.org/wiki/Fletcher%27s_checksum) |
| Fletcher-8 Checksum | `fletcher-8-checksum` | [Fletcher](https://wikipedia.org/wiki/Fletcher%27s_checksum) |
| Generate all checksums | `generate-all-checksums` | [Checksum](https://wikipedia.org/wiki/Checksum) |
| GOST Hash | `gost-hash` | [GOST (hash function)](https://wikipedia.org/wiki/GOST_(hash_function)) |
| HAS-160 | `has-160` | [HAS-160](https://wikipedia.org/wiki/HAS-160) |
| HMAC | `hmac` | [HMAC](https://wikipedia.org/wiki/HMAC) |
| Keccak | `keccak` | [SHA-3 / Keccak](https://wikipedia.org/wiki/SHA-3) |
| LM Hash | `lm-hash` | [LAN Manager](https://wikipedia.org/wiki/LAN_Manager#Password_hashing_algorithm) |
| Luhn Checksum | `luhn-checksum` | [Luhn mod N](https://wikipedia.org/wiki/Luhn_mod_N_algorithm) |
| MD2 | `md2` | [MD2](https://wikipedia.org/wiki/MD2_(cryptography)) |
| MD4 | `md4` | [MD4](https://wikipedia.org/wiki/MD4) |
| MD5 | `md5` | [MD5](https://wikipedia.org/wiki/MD5) |
| MurmurHash3 | `murmurhash3` | [MurmurHash](https://wikipedia.org/wiki/MurmurHash) |
| NT Hash | `nt-hash` | [NT LAN Manager](https://wikipedia.org/wiki/NT_LAN_Manager) |
| Parity Bit | `parity-bit` | [Parity bit](https://wikipedia.org/wiki/Parity_bit) |
| RIPEMD | `ripemd` | [RIPEMD](https://wikipedia.org/wiki/RIPEMD) |
| Scrypt | `scrypt` | [Scrypt](https://wikipedia.org/wiki/Scrypt) |
| SHA0 | `sha0` | [SHA-0](https://wikipedia.org/wiki/SHA-1#SHA-0) |
| SHA1 | `sha1` | [SHA-1](https://wikipedia.org/wiki/SHA-1) |
| SHA224 | `sha224` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA256 | `sha256` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA3 | `sha3` | [SHA-3](https://wikipedia.org/wiki/SHA-3) |
| SHA384 | `sha384` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA512 | `sha512` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SM3 | `sm3` | [SM3](https://wikipedia.org/wiki/SM3_(hash_function)) |
| Shake | `shake` | [SHAKE](https://wikipedia.org/wiki/SHA-3#Instances) |
| Snefru | `snefru` | [Snefru](https://wikipedia.org/wiki/Snefru) |
| Streebog | `streebog` | [Streebog](https://wikipedia.org/wiki/Streebog) |
| TCP/IP Checksum | `tcp-ip-checksum` | [IPv4 checksum](https://wikipedia.org/wiki/IPv4_header_checksum) |
| Whirlpool | `whirlpool` | [Whirlpool](https://wikipedia.org/wiki/Whirlpool_(hash_function)) |
| XOR Checksum | `xor-checksum` | [XOR](https://wikipedia.org/wiki/XOR) |

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

## Analyse hash

Reference: [Comparison of hash functions](https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions)

Reports a hash's length (in characters, bytes and bits) and lists the hashing
functions that produce a digest of that size. The input must be hexadecimal
(whitespace is ignored); anything else is rejected. Takes no options.

```bash
cchef analyse-hash -i d41d8cd98f00b204e9800998ecf8427e
```

Output:

```
Hash length: 32
Byte length: 16
Bit length:  128

Based on the length, this hash could have been generated by one of the following hashing functions:
MD5
MD4
MD2
HAVAL-128
RIPEMD-128
Snefru
Tiger-128
```

## Argon2

Reference: [Argon2](https://wikipedia.org/wiki/Argon2)

Derives a hash from the input password with the Argon2 password-hashing function
(the Password Hashing Competition winner). All three variants are supported, and
the result can be emitted as a PHC-encoded hash, hex, or raw bytes. Argon2i and
Argon2id are computed with Go's `x/crypto/argon2`; Argon2d uses a from-scratch
implementation (`x/crypto` does not provide it). Verified against argon2-cffi —
the reference phc-winner-argon2 library CyberChef's argon2-browser is built from.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--salt` | toggleString | `somesalt` | Salt (min 8 bytes), interpreted by `--salt-type` (`UTF8`, `Hex`, `Base64`, `Latin1`). |
| `--iterations` | number | `3` | Time cost (passes over memory). |
| `--memory-kib` | number | `4096` | Memory cost in KiB (min 8 × parallelism). |
| `--parallelism` | number | `1` | Number of lanes. |
| `--hash-length-bytes` | number | `32` | Output length in bytes (min 4). |
| `--type` | option | `Argon2i` | `Argon2i`, `Argon2d` or `Argon2id`. |
| `--output-format` | option | `Encoded hash` | `Encoded hash` (PHC string), `Hex hash` or `Raw hash`. |

**Simple example**

```bash
cchef argon2 --salt somesalt --iterations 2 --memory-kib 256 --type Argon2id -i password
```

Output:

```
$argon2id$v=19$m=256,t=2,p=1$c29tZXNhbHQ$nf65EOgLrQMR/uIPnA4rEsF5h7TKyQwu9U1bMCHGi/4
```

**Hex output (Argon2d)**

```bash
cchef argon2 --salt somesalt --iterations 2 --memory-kib 256 --type Argon2d --output-format "Hex hash" -i password
```

Output:

```
25c4ee8ba448054b49efc804e478b9d823be1f9bd2e99f51d6ec4007a3a1501f
```

## Argon2 compare

Reference: [Argon2](https://wikipedia.org/wiki/Argon2)

Tests whether the input password matches a given Argon2 PHC-encoded hash (any of
the three variants), returning `Match: <password>` or `No match`. A malformed
encoded hash also returns `No match`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--encoded-hash` | string | (empty) | The Argon2 encoded hash to test against. |

**Simple example**

```bash
cchef argon2-compare --encoded-hash '$argon2id$v=19$m=256,t=2,p=1$c29tZXNhbHQ$nf65EOgLrQMR/uIPnA4rEsF5h7TKyQwu9U1bMCHGi/4' -i password
```

Output:

```
Match: password
```

## Ascon Hash

Reference: [Ascon](https://wikipedia.org/wiki/Ascon_(cipher))

Ascon-Hash256 produces a fixed 256-bit (32-byte) hash, part of the Ascon
lightweight-cryptography family standardised in NIST SP 800-232 (selected in
2023). It is designed for constrained devices such as IoT sensors. Output is
hex. Takes no options.

**Simple example**

```bash
cchef ascon-hash -i "Hello, World!"
```

Output:

```
f40e1ce8d4272e628e9535193f196f4ff2a720b00f6380c5d6f16b975f3a7777
```

## Ascon MAC

Reference: [Ascon](https://wikipedia.org/wiki/Ascon_(cipher))

Ascon-Mac produces a 128-bit (16-byte) message authentication code under a
16-byte key, from the same NIST SP 800-232 Ascon family. Output is hex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | toggleString | `` (Hex) | The MAC key — must be exactly 16 bytes. Encoding: `Hex`, `UTF8`, `Latin1` or `Base64`. |

**Simple example**

```bash
cchef ascon-mac --key 000102030405060708090a0b0c0d0e0f -i "Hello, World!"
```

Output:

```
cf2675e26d71ddcd760be5f6455b1f53
```

## BLAKE2b

Reference: [BLAKE2b](https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2b_algorithm)

Computes the BLAKE2b hash (the 64-bit-optimised BLAKE2 flavour) at a chosen digest
size, with an optional key (turning it into a MAC). Output can be hex, Base64 or
raw bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | option | `512` | Digest size in bits: `512`, `384`, `256`, `160` or `128`. |
| `--output-encoding` | option | `Hex` | `Hex`, `Base64` or `Raw`. |
| `--key` | toggleString | (empty) | Optional key (max 64 bytes), interpreted by `--key-type` (`UTF8`, `Decimal`, `Base64`, `Hex`, `Latin1`). |

**Simple example**

```bash
cchef blake2b --size 256 -i "Hello World"
```

Output:

```
1dc01772ee0171f5f614c673e3c7fa1107a8cf727bdf5a6dadb379e93c0d1d00
```

**Keyed example**

```bash
cchef blake2b --size 128 --key "pseudorandom key" -i "message data"
```

Output:

```
3d363ff7401e02026f4a4687d4863ced
```

## BLAKE2s

Reference: [BLAKE2s](https://wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2)

Computes the BLAKE2s hash (the 8- to 32-bit-optimised BLAKE2 flavour) at a chosen
digest size, with an optional key. A from-scratch port (Go's `x/crypto/blake2s`
only offers the 256-bit and keyed-128-bit variants, not the 160-bit or unkeyed
128-bit digests this operation needs). Output can be hex, Base64 or raw bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--size` | option | `256` | Digest size in bits: `256`, `160` or `128`. |
| `--output-encoding` | option | `Hex` | `Hex`, `Base64` or `Raw`. |
| `--key` | toggleString | (empty) | Optional key (max 32 bytes), interpreted by `--key-type` (`UTF8`, `Decimal`, `Base64`, `Hex`, `Latin1`). |

**Simple example**

```bash
cchef blake2s --size 160 -i "Hello World"
```

Output:

```
0e4fcfc2ee0097ac1d72d70b595a39e09a3c7c7e
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

## CMAC

Reference: [CMAC](https://wikipedia.org/wiki/CMAC)

CMAC is a block-cipher-based message authentication code (RFC 4493 defines
AES-CMAC; NIST SP 800-38B also covers Triple DES). Input is the message bytes;
output is the tag as hex (16 bytes for AES, 8 for Triple DES).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | toggleString | `` (Hex) | The key. AES needs 16, 24 or 32 bytes; Triple DES needs 16 or 24. Encoding: `Hex`, `UTF8`, `Latin1` or `Base64`. |
| `--encryption-algorithm` | option | `AES` | Block cipher: `AES` or `Triple DES`. |

**Simple example**

The AES-128 CMAC of an empty message (NIST CSRC example):

```bash
cchef cmac --key 2b7e151628aed2a6abf7158809cf4f3c -i ""
```

Output:

```
bb1d6929e95937287fa37d129b756746
```

**Complex example**

Triple DES CMAC over a 16-byte message supplied as hex:

```bash
echo -n "6bc1bee22e409f96e93d7e117393172a" | cchef from-hex | cchef cmac --key 0123456789abcdef23456789abcdef01456789abcdef0123 --encryption-algorithm "Triple DES"
```

Output:

```
30239cf1f52e6609
```

## CRC Checksum

Reference: [CRC](https://wikipedia.org/wiki/Cyclic_redundancy_check)

Computes a Cyclic Redundancy Check over the raw input bytes. Around 170 named CRC
variants (widths 3–82 bits) are built in, or a fully custom width / polynomial /
initialisation / reflection / xor configuration can be given. A faithful port of
CyberChef's operation (the Rocksoft parameterised model), byte-for-byte identical
to upstream. Output is the checksum as hex, padded to the variant's width.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--algorithm` | option | `Custom` | The CRC variant (e.g. `CRC-32`, `CRC-16`, `CRC-8`), or `Custom` to specify the parameters below. |
| `--width-bits` | toggleString (`Decimal`) | `0` | Custom width in bits. |
| `--polynomial` | toggleString (`Hex`) | `0` | Custom generator polynomial. |
| `--initialization` | toggleString (`Hex`) | `0` | Custom initial value. |
| `--reflect-input` | option | `True` | Reflect each input byte. |
| `--reflect-output` | option | `True` | Reflect the final CRC. |
| `--xor-output` | toggleString (`Hex`) | `0` | Value XORed into the final CRC. |

**Simple example**

```bash
cchef crc-checksum --algorithm CRC-32 -i 123456789
```

Output:

```
cbf43926
```

**Custom example** (the same CRC-32 specified by hand)

```bash
cchef crc-checksum --algorithm Custom --width-bits 32 --polynomial 04C11DB7 \
  --initialization FFFFFFFF --reflect-input True --reflect-output True --xor-output FFFFFFFF -i 123456789
```

Output:

```
cbf43926
```

## Fletcher-8 Checksum

Reference: [Fletcher's checksum](https://wikipedia.org/wiki/Fletcher%27s_checksum)

Computes the 8-bit Fletcher checksum (two running 4-bit sums, mod 15) of the raw
input, output as hex. See also **Fletcher-16/32/64 Checksum**. Takes no options.

```bash
cchef fletcher-8-checksum -i abcde
```

Output:

```
50
```

## Fletcher-16 Checksum

Reference: [Fletcher's checksum](https://wikipedia.org/wiki/Fletcher%27s_checksum)

Computes the 16-bit Fletcher checksum (two running 8-bit sums, mod 255) of the raw
input, output as hex. Takes no options.

```bash
cchef fletcher-16-checksum -i abcde
```

Output:

```
c8f0
```

## Fletcher-32 Checksum

Reference: [Fletcher's checksum](https://wikipedia.org/wiki/Fletcher%27s_checksum)

Computes the 32-bit Fletcher checksum over 16-bit little-endian words (two running
16-bit sums, mod 65535), output as hex. Takes no options.

```bash
cchef fletcher-32-checksum -i abcde
```

Output:

```
f04fc729
```

## Fletcher-64 Checksum

Reference: [Fletcher's checksum](https://wikipedia.org/wiki/Fletcher%27s_checksum)

Computes the 64-bit Fletcher checksum over 32-bit little-endian words (two running
32-bit sums, mod 2³²−1), output as hex. Takes no options.

```bash
cchef fletcher-64-checksum -i abcde
```

Output:

```
c8c6c527646362c6
```

## Generate all checksums

Reference: [Checksum](https://wikipedia.org/wiki/Checksum)

Runs every built-in checksum over the input and lists the results: all ~170 CRC
variants plus the Fletcher-8/16/32/64 and Adler-32 checksums. The listing can be
filtered to a single bit width, and the algorithm names can be omitted.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--length-bits` | option | `All` | Restrict the output to checksums of this width (`All`, `3`, `4`, … `82`). |
| `--include-names` | boolean | `true` | Prefix each value with its aligned algorithm name. |

**Simple example** (only the 3-bit checksums)

```bash
cchef generate-all-checksums --length-bits 3 -i 123456789
```

Output:

```
CRC-3/GSM:                4
CRC-3/ROHC:               6
```

With `--length-bits All` (the default) the operation emits all 176 checksums;
`--include-names=false` outputs just the values, one per line.

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

## LM Hash

Reference: [LAN Manager](https://wikipedia.org/wiki/LAN_Manager#Password_hashing_algorithm)

The LM (LAN Manager) hash is a deprecated Windows password hash. The password is
uppercased, truncated/padded to 14 bytes, split into two 7-byte halves, and each
half becomes a DES key that encrypts the constant `KGS!@#$%`; the two ciphertexts
are concatenated. It is extremely weak and takes no options.

**Simple example**

```bash
cchef lm-hash -i "password"
```

Output:

```
E52CAC67419A9A224A3B108F3FA6CB6D
```

## Luhn Checksum

Reference: [Luhn mod N](https://wikipedia.org/wiki/Luhn_mod_N_algorithm)

Computes the Luhn mod-N checksum of a string, reporting the checksum, the check
digit, and the input with the check digit appended (the "Luhn validated string").
The radix must be even and in the range 2–36.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--radix` | number | `10` | The even base (2–36) the input digits are in. |

**Simple example**

```bash
cchef luhn-checksum -i 35641709012469
```

Output:

```
Checksum: 7
Checkdigit: 0
Luhn Validated String: 356417090124690
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

## MurmurHash3

Reference: [MurmurHash](https://wikipedia.org/wiki/MurmurHash)

Computes the 32-bit MurmurHash v3 (x86_32 variant) of the input, a fast
non-cryptographic hash. The result is a 32-bit integer, optionally converted to a
signed value. A faithful port of CyberChef's operation (each character contributes
its low byte).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--seed` | number | `0` | Seed value for the hash. |
| `--convert-to-signed` | boolean | `false` | Output as a signed 32-bit integer instead of unsigned. |

**Simple example**

```bash
cchef murmurhash3 -i "Hello World!"
```

Output:

```
3691591037
```

**Signed example** (with a seed)

```bash
cchef murmurhash3 --seed 0 --convert-to-signed -i foo
```

Output:

```
-156908512
```

## NT Hash

Reference: [NT LAN Manager](https://wikipedia.org/wiki/NT_LAN_Manager)

The NT hash (also called an NTLM hash) is how Windows stores passwords: MD4 of
the password encoded as UTF-16LE. Output is uppercase hex. It takes no options
and is considered weak against modern brute-forcing.

**Simple example**

```bash
cchef nt-hash -i "password"
```

Output:

```
8846F7EAEE8FB117AD06BDD830B7586C
```

## Parity Bit

Reference: [Parity bit](https://wikipedia.org/wiki/Parity_bit)

Adds (Encode) or removes (Decode) a parity bit on a string of binary digits, at
the start or end. With a delimiter set, each delimited token is handled
independently (e.g. one parity bit per byte). On encode, only `0`, `1`, spaces and
the delimiter are permitted; any other character is an error.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--mode` | option | `Even Parity` | `Even Parity` or `Odd Parity`. |
| `--postion` | option | `Start` | Put the parity bit at the `Start` or `End`. |
| `--encode-or-decode` | option | `Encode` | `Encode` to add a parity bit, `Decode` to remove it. |
| `--delimiter` | string | (empty) | If set, split on this delimiter and process each token. |

**Simple example**

```bash
cchef parity-bit -i "01010101 10101010" --mode "Even Parity" --postion Start --encode-or-decode Encode
```

Output:

```
001010101 10101010
```

**Per-byte example** (binary of "hello world!", one parity bit per byte)

```bash
echo -n "hello world!" | cchef to-binary --delimiter Space | cchef parity-bit --mode "Even Parity" --postion Start --delimiter " "
```

Output:

```
101101000 001100101 001101100 001101100 001101111 100100000 001110111 001101111 001110010 001101100 101100100 000100001
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

## SM3

Reference: [SM3](https://wikipedia.org/wiki/SM3_(hash_function))

The SM3 cryptographic hash (Chinese National Standard GM/T 0004), a 256-bit hash
used in the SM2/SM9 signature schemes. A from-scratch port of the crypto-api
implementation CyberChef wraps, so — like upstream — the output length and round
count are configurable. Note this follows crypto-api's behaviour for non-default
values: `Length` truncates the digest to `floor(Length / 32)` 32-bit words (a
value below 32 yields the full digest), and `Rounds` above 64 reads past the
internal message schedule, collapsing the state toward the initial value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--length` | number | `256` | Output length in bits (truncated to whole 32-bit words). |
| `--rounds` | number | `64` | Number of compression rounds (minimum 16). |

**Example**

```bash
echo -n "abc" | cchef sm3
```

Output:

```
66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0
```

**Complex example** (truncated to 128 bits)

```bash
echo -n "abc" | cchef sm3 --length 128
```

Output:

```
66c7f0f462eeedd9d1f2d46bdc10e4e2
```

## Shake

Reference: [SHAKE](https://wikipedia.org/wiki/SHA-3#Instances)

The SHAKE extendable-output function (XOF) of SHA-3, producing a digest of any
requested length. The `Size` is given in bits, and the output is `floor(size/8)`
bytes as hex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--capacity` | option | `256` | The SHAKE variant: `256` (SHAKE256) or `128` (SHAKE128). |
| `--size` | number | `512` | Output length in bits. |

**Simple example**

```bash
cchef shake --capacity 256 --size 256 -i "Hello World"
```

Output:

```
840d1ce81a4327840b54cb1d419907fd1f62359bad33656e058653d2e4172a43
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

## Streebog

Reference: [Streebog](https://wikipedia.org/wiki/Streebog)

Streebog is the cryptographic hash function of the Russian national standard
GOST R 34.11-2012, created to replace the older GOST R 34.11-94 hash. It shares
the GOST digest engine with the `gost-hash` operation. Output is hex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--digest-length` | option | `256` | Digest length in bits: `256` or `512`. |

**Simple example**

```bash
cchef streebog -i "Hello"
```

Output:

```
3c10d2ffe0787bc8bd6eacd337d59c314ce689c847a422f6c34b4b75f45751bc
```

**Complex example**

```bash
cchef streebog --digest-length 512 -i "Hello"
```

Output:

```
5bfa12a667f8da6ec0d3101f02122b6f8b7686fffcc524a7acc4c202f7b2d8f50f135405b8f4626f9ae97a8dcbec714f5294ae7b9fb32a0d6bf3dbf98a3c1d90
```

## TCP/IP Checksum

Reference: [IPv4 header checksum](https://wikipedia.org/wiki/IPv4_header_checksum)

Computes the 16-bit one's-complement Internet checksum (RFC 1071) over the raw
input bytes — the checksum used in IPv4, TCP, UDP and ICMP headers. Output is hex.
Takes no options.

```bash
echo -n "45 00 00 3c 1c 46 40 00 40 06 00 00 ac 10 0a 63 ac 10 0a 0c" | cchef from-hex | cchef tcp-ip-checksum
```

Output:

```
b1e6
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

## XOR Checksum

Reference: [XOR](https://wikipedia.org/wiki/XOR)

Splits the input into blocks of the given size and XORs the blocks together, one
byte position at a time (a short final block leaves its missing positions
unchanged). Output is the resulting block as hex.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--blocksize` | number | `4` | The block size, in bytes (must be a positive integer). |

**Simple example**

```bash
cchef xor-checksum --blocksize 4 -i "The ships hung in the sky in much the same way that bricks don't."
```

Output:

```
4918421b
```
