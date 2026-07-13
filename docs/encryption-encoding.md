# Encryption / Encoding

Classic ciphers and bitwise operations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| A1Z26 Cipher Decode | `a1z26-cipher-decode` | [Letter-number cipher](https://www.dcode.fr/letter-number-cipher) |
| A1Z26 Cipher Encode | `a1z26-cipher-encode` | [Letter-number cipher](https://www.dcode.fr/letter-number-cipher) |
| ADD | `add` | [Bitwise operation](https://wikipedia.org/wiki/Bitwise_operation#Bitwise_operators) |
| AND | `and` | [Bitwise AND](https://wikipedia.org/wiki/Bitwise_operation#AND) |
| Bit shift left | `bit-shift-left` | [Bit shifts](https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts) |
| Bit shift right | `bit-shift-right` | [Bit shifts](https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts) |
| NOT | `not` | [Bitwise NOT](https://wikipedia.org/wiki/Bitwise_operation#NOT) |
| OR | `or` | [Bitwise OR](https://wikipedia.org/wiki/Bitwise_operation#OR) |
| ROR13 | `ror13` | [Circular shift](https://wikipedia.org/wiki/Circular_shift) |
| ROT13 | `rot13` | [ROT13](https://wikipedia.org/wiki/ROT13) |
| ROT47 | `rot47` | [ROT13 variants](https://wikipedia.org/wiki/ROT13#Variants) |
| ROT8000 | `rot8000` | [ROT8000](https://rot8000.com/info) |
| Rotate left | `rotate-left` | [Bit shifts](https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts) |
| Rotate right | `rotate-right` | [Bit shifts](https://wikipedia.org/wiki/Bitwise_operation#Bit_shifts) |
| SUB | `sub` | [Bitwise operation](https://wikipedia.org/wiki/Bitwise_operation#Bitwise_operators) |
| XOR | `xor` | [XOR](https://wikipedia.org/wiki/XOR) |
| XOR Brute Force | `xor-brute-force` | [Exclusive or](https://wikipedia.org/wiki/Exclusive_or) |

The bitwise key operations (`add`, `and`, `or`, `sub`, `xor`) all take a `--key`
whose encoding is chosen with `--key-type` (`Hex`, `Decimal`, `Binary`, `Base64`,
`UTF8`, `Latin1`). The key repeats to cover the input; an empty key is treated as
a single zero byte (the identity). They produce raw bytes, so the examples pipe
through `to-hex` to make the output readable.

> **Decimal / Binary keys.** Matching CyberChef, a `Decimal` key uses only the
> first integer in the string (`82 226` → the single byte `82`), and a `Binary`
> key has its whitespace stripped and is read as fixed 8-bit groups.

---

## A1Z26 Cipher Decode

Converts alphabet order numbers back into their corresponding letters (`1` → `a`,
`26` → `z`). Every number must be between 1 and 26, or the operation errors.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | Separator between numbers: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

**Simple example**

```bash
$ printf '8 5 12 12 15' | cchef a1z26-cipher-decode
hello
```

**Comma delimiter**

```bash
$ printf '8,5,12,12,15' | cchef a1z26-cipher-decode --delimiter Comma
hello
```

## A1Z26 Cipher Encode

Converts alphabet characters into their corresponding alphabet order number
(`a` → `1`, `b` → `2`). Input is lowercased first, and non-alphabet characters
are dropped.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | Separator between numbers: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

**Simple example**

```bash
$ printf 'Hello, World!' | cchef a1z26-cipher-encode
8 5 12 12 15 23 15 18 12 4
```

**Comma delimiter**

```bash
$ printf 'Hello, World!' | cchef a1z26-cipher-encode --delimiter Comma
8,5,12,12,15,23,15,18,12,4
```

---

## ADD

Adds the key to each byte of the input, modulo 256.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `Decimal`, `Binary`, `Base64`, `UTF8`, `Latin1`. |

**Simple example**

```bash
$ printf 'hello' | cchef add --key 01 --key-type Hex | cchef to-hex --delimiter None
69666d6d70
```

**Repeating multi-byte key**

```bash
$ printf 'hello' | cchef add --key '01 02' --key-type Hex | cchef to-hex --delimiter None
69676d6e70
```

---

## AND

ANDs each byte of the input with the repeating key.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `Decimal`, `Binary`, `Base64`, `UTF8`, `Latin1`. |

**Simple example**

```bash
$ printf 'hello' | cchef and --key 0f --key-type Hex | cchef to-hex --delimiter None
08050c0c0f
```

---

## Bit shift left

Shifts the bits in each byte left by a fixed amount (bits shifted past the top of
the byte are dropped).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--amount` | number | `1` | Number of bits to shift (0–7). |

**Simple example**

```bash
$ printf 'Hi' | cchef bit-shift-left --amount 1 | cchef to-hex --delimiter None
90d2
```

---

## Bit shift right

Shifts the bits in each byte right by a fixed amount. A *logical* shift fills the
vacated top bits with zeros; an *arithmetic* shift preserves the most significant
(sign) bit of the original byte.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--amount` | number | `1` | Number of bits to shift. |
| `--type` | option | `Logical shift` | `Logical shift` or `Arithmetic shift`. |

**Simple example**

```bash
$ printf 'Hi' | cchef bit-shift-right --amount 1 --type 'Logical shift' | cchef to-hex --delimiter None
2434
```

**Logical vs. arithmetic shift**

With the high bit set (`0x80`, `0xff`), the arithmetic shift keeps the sign bit
while the logical shift clears it:

```bash
$ printf '80ff' | cchef from-hex | cchef bit-shift-right --amount 1 --type 'Logical shift' | cchef to-hex --delimiter None
407f
$ printf '80ff' | cchef from-hex | cchef bit-shift-right --amount 1 --type 'Arithmetic shift' | cchef to-hex --delimiter None
c0ff
```

---

## NOT

Returns the inverse (bitwise complement) of each byte.

**Simple example**

```bash
$ printf 'hello' | cchef not | cchef to-hex --delimiter None
979a939390
```

---

## OR

ORs each byte of the input with the repeating key.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `Decimal`, `Binary`, `Base64`, `UTF8`, `Latin1`. |

**Simple example**

```bash
$ printf 'hello' | cchef or --key 80 --key-type Hex | cchef to-hex --delimiter None
e8e5ececef
```

---

## ROR13

Computes the ROR13 hash of the input: the 32-bit accumulator is rotated right 13
bits and each byte added in turn, and the result is emitted as `0x`-prefixed,
8-digit uppercase hex. This is the API-name hashing convention used by some
Windows shellcode to resolve exports without embedding their names.

**Simple example**

```bash
$ printf 'LoadLibraryA' | cchef ror13
0xEC0E4E8E
```

---

## ROT13

A Caesar substitution cipher that rotates alphabet characters (and optionally
digits) by a configurable amount.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rotate-lower-case-chars` | bool | `true` | Rotate `a`–`z`. |
| `--rotate-upper-case-chars` | bool | `true` | Rotate `A`–`Z`. |
| `--rotate-numbers` | bool | `false` | Also rotate `0`–`9`. |
| `--amount` | number | `13` | Rotation amount (negative values rotate backwards). |

**Simple example**

```bash
$ cchef rot13 -i 'Hello, World!'
Uryyb, Jbeyq!
```

**Also rotate digits**

```bash
$ cchef rot13 --rotate-numbers -i 'abc 123'
nop 456
```

---

## ROT47

A variant of the Caesar cipher covering printable ASCII characters from `!` (33)
to `~` (126). The default rotation is 47.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--amount` | number | `47` | Rotation amount (negative values rotate backwards). |

**Simple example**

```bash
$ cchef rot47 -i 'Hello, World!'
w6==@[ (@C=5P
```

---

## ROT8000

A Caesar cipher over the valid BMP code points that shifts each character by
0x8000 positions along the (compacted) alphabet, leaving anything outside the
mapping unchanged. The shift is exactly half the alphabet, so the operation is
its own inverse — running it twice returns the original text.

**Simple example**

```bash
$ cchef rot8000 -i 'Hi'
籑籲
```

**Round-trip (decrypt)**

```bash
$ cchef rot8000 -i 'Hi' | cchef rot8000
Hi
```

---

## Rotate left

Rotates the bits of each byte to the left by a fixed amount. With `--carry-through`
the bits that fall off the front of a byte are carried into the next byte and the
array is rotated as a whole (wrapping around from the end back to the start).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--amount` | number | `1` | Number of bit positions to rotate by. |
| `--carry-through` | bool | `false` | Carry excess bits into the neighbouring byte. |

**Simple example**

```bash
$ printf 'abc' | cchef rotate-left | cchef to-hex --delimiter Space
c2 c4 c6
```

**Carry through**

```bash
$ printf 'abc123' | cchef rotate-left --amount 2 --carry-through | cchef to-hex --delimiter Space
85 89 8c c4 c8 cd
```

---

## Rotate right

Rotates the bits of each byte to the right by a fixed amount. With
`--carry-through` the bits that fall off the end of a byte are carried into the
next byte and the array is rotated as a whole (wrapping around from the start back
to the end).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--amount` | number | `1` | Number of bit positions to rotate by. |
| `--carry-through` | bool | `false` | Carry excess bits into the neighbouring byte. |

**Simple example**

```bash
$ printf 'abc123' | cchef rotate-right | cchef to-hex --delimiter Space
b0 31 b1 98 19 99
```

**Carry through**

```bash
$ printf 'abc123' | cchef rotate-right --amount 2 --carry-through | cchef to-hex --delimiter Space
d8 58 98 cc 4c 8c
```

---

## SUB

Subtracts the key from each byte of the input, modulo 256.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `Decimal`, `Binary`, `Base64`, `UTF8`, `Latin1`. |

**Simple example**

```bash
$ printf 'hello' | cchef sub --key 01 --key-type Hex | cchef to-hex --delimiter None
67646b6b6e
```

---

## XOR

XORs the input with a repeating key. The key can be supplied in several
encodings, and several key-update schemes are available.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | The key value, interpreted according to `--key-type`. |
| `--key-type` | option | `Hex` | How to interpret the key: `Hex`, `Decimal`, `Binary`, `Base64`, `UTF8`, `Latin1`. |
| `--scheme` | option | `Standard` | Key-update scheme: `Standard`, `Input differential`, `Output differential`, `Cascade`. |
| `--null-preserving` | bool | `false` | Skip bytes that are `0x00` or equal to the key. |

**Simple example**

XOR produces raw bytes, so pipe through `to-hex` to view the result:

```bash
$ cchef xor --key 42 --key-type Hex -i 'Hello' | cchef to-hex --delimiter None
0a272e2e2d
```

**UTF-8 key and round trip**

XOR is symmetric — applying the same key twice restores the input:

```bash
$ echo -n 'Secret message' \
    | cchef xor --key 'k3y' --key-type UTF8 \
    | cchef xor --key 'k3y' --key-type UTF8
Secret message
```

---

## XOR Brute Force

Enumerates every XOR key up to the given length (max 2) and prints each candidate
plaintext. Supply a **crib** — a string you expect in the plaintext — to filter
the results; the match is case-insensitive. Input is taken as raw bytes, so pipe
ciphertext in via `from-hex` (or a file).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key-length` | number | `1` | Key length in bytes (1–2). |
| `--sample-length` | number | `100` | Number of input bytes to test. |
| `--sample-offset` | number | `0` | Byte offset to start the sample at. |
| `--scheme` | option | `Standard` | Key-update scheme: `Standard`, `Input differential`, `Output differential`. |
| `--null-preserving` | bool | `false` | Skip bytes that are `0x00` or equal to the key. |
| `--print-key` | bool | `true` | Prefix each line with the key that produced it. |
| `--output-as-hex` | bool | `false` | Print candidates as hex instead of text. |
| `--crib-known-plaintext-string` | string | (empty) | Only show candidates containing this string. |

**Simple example**

Recover the key for `Hello` XORed with `0x42`, filtering on the crib `hello`
(which also matches the case-swapped `0x62` result):

```bash
$ printf '0a272e2e2d' | cchef from-hex --delimiter None \
    | cchef xor-brute-force --crib-known-plaintext-string hello
Key = 42: Hello
Key = 62: hELLO
```

**Hex output**

```bash
$ printf '0a272e2e2d' | cchef from-hex --delimiter None \
    | cchef xor-brute-force --crib-known-plaintext-string hello --output-as-hex
Key = 42: 48 65 6c 6c 6f
Key = 62: 68 45 4c 4c 4f
```
