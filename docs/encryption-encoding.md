# Encryption / Encoding

Classic ciphers and bitwise operations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| A1Z26 Cipher Decode | `a1z26-cipher-decode` | [Letter-number cipher](https://www.dcode.fr/letter-number-cipher) |
| A1Z26 Cipher Encode | `a1z26-cipher-encode` | [Letter-number cipher](https://www.dcode.fr/letter-number-cipher) |
| ADD | `add` | [Bitwise operation](https://wikipedia.org/wiki/Bitwise_operation#Bitwise_operators) |
| AES Decrypt | `aes-decrypt` | [Advanced Encryption Standard](https://wikipedia.org/wiki/Advanced_Encryption_Standard) |
| AES Encrypt | `aes-encrypt` | [Advanced Encryption Standard](https://wikipedia.org/wiki/Advanced_Encryption_Standard) |
| AES Key Unwrap | `aes-key-unwrap` | [Key wrap](https://wikipedia.org/wiki/Key_wrap) |
| AES Key Wrap | `aes-key-wrap` | [Key wrap](https://wikipedia.org/wiki/Key_wrap) |
| Affine Cipher Decode | `affine-cipher-decode` | [Affine cipher](https://wikipedia.org/wiki/Affine_cipher) |
| Affine Cipher Encode | `affine-cipher-encode` | [Affine cipher](https://wikipedia.org/wiki/Affine_cipher) |
| AND | `and` | [Bitwise AND](https://wikipedia.org/wiki/Bitwise_operation#AND) |
| Ascon Decrypt | `ascon-decrypt` | [Ascon (cipher)](https://wikipedia.org/wiki/Ascon_(cipher)) |
| Ascon Encrypt | `ascon-encrypt` | [Ascon (cipher)](https://wikipedia.org/wiki/Ascon_(cipher)) |
| Atbash Cipher | `atbash-cipher` | [Atbash](https://wikipedia.org/wiki/Atbash) |
| Bacon Cipher Decode | `bacon-cipher-decode` | [Bacon's cipher](https://wikipedia.org/wiki/Bacon%27s_cipher) |
| Bacon Cipher Encode | `bacon-cipher-encode` | [Bacon's cipher](https://wikipedia.org/wiki/Bacon%27s_cipher) |
| Bcrypt | `bcrypt` | [Bcrypt](https://wikipedia.org/wiki/Bcrypt) |
| Bifid Cipher Decode | `bifid-cipher-decode` | [Bifid cipher](https://wikipedia.org/wiki/Bifid_cipher) |
| Bifid Cipher Encode | `bifid-cipher-encode` | [Bifid cipher](https://wikipedia.org/wiki/Bifid_cipher) |
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

## AES Decrypt

Reference: [Advanced Encryption Standard](https://wikipedia.org/wiki/Advanced_Encryption_Standard)

Decrypts AES ciphertext. The key length selects the algorithm (16 bytes =
AES-128, 24 = AES-192, 32 = AES-256). In CBC and ECB mode PKCS#7 padding is
removed unless a `NoPadding` mode is chosen; in GCM mode the `--gcm-tag` is
verified and decryption fails if it does not authenticate.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | Decryption key, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | (empty) | Initialization vector; empty defaults to 16 null bytes. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv-length` | number | `16` | IV length in bytes when taken from the input. |
| `--mode` | option | `CBC` | Mode: `CBC`, `CFB`, `OFB`, `CTR`, `GCM`, `ECB`, `CBC/NoPadding`, `ECB/NoPadding`. |
| `--input-format` | option | `Hex` | How to read the input: `Hex` or `Raw` bytes. |
| `--output-format` | option | `Raw` | How to render the output: `Raw` bytes or `Hex`. |
| `--gcm-tag` | string | (empty) | Authentication tag (GCM mode only). |
| `--gcm-tag-type` | option | `Hex` | Tag encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--additional-authenticated-data` | string | (empty) | AAD (GCM mode only). |
| `--additional-authenticated-data-type` | option | `Hex` | AAD encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv-from-input` | option | `Off` | Take the IV from the input: `Off`, `From start`, `From end`. |

**Simple example**

```bash
$ printf '2ef6c3fdb1314b5c2c326a2087fe1a82d5e73bf605ec8431d73e847187fc1c8fbbe969c177df1ecdf8c13f2f505f9498' | cchef aes-decrypt --key 00112233445566778899aabbccddeeff --iv 00000000000000000000000000000000 --mode CBC --input-format Hex --output-format Raw
The quick brown fox jumps over the lazy dog.
```

---

## AES Encrypt

Reference: [Advanced Encryption Standard](https://wikipedia.org/wiki/Advanced_Encryption_Standard)

Encrypts input with AES. The key length selects the algorithm (16 bytes =
AES-128, 24 = AES-192, 32 = AES-256). CBC and ECB use PKCS#7 padding; GCM
appends the authentication tag to the output. An empty IV defaults to 16 null
bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | Encryption key, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | (empty) | Initialization vector; empty defaults to 16 null bytes. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--mode` | option | `CBC` | Mode: `CBC`, `CFB`, `OFB`, `CTR`, `GCM`, `ECB`, `CBC/NoPadding`, `ECB/NoPadding`. |
| `--input-format` | option | `Raw` | How to read the input: `Raw` bytes or `Hex`. |
| `--output-format` | option | `Hex` | How to render the output: `Hex` or `Raw` bytes. |
| `--additional-authenticated-data` | string | (empty) | AAD (GCM mode only). |
| `--additional-authenticated-data-type` | option | `Hex` | AAD encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--include-iv-in-output` | option | `Off` | Add the IV to the output: `Off`, `Prepend`, `Append`. |

**Simple example**

```bash
$ printf 'The quick brown fox jumps over the lazy dog.' | cchef aes-encrypt --key 00112233445566778899aabbccddeeff --iv 00000000000000000000000000000000 --mode CBC
2ef6c3fdb1314b5c2c326a2087fe1a82d5e73bf605ec8431d73e847187fc1c8fbbe969c177df1ecdf8c13f2f505f9498
```

**GCM with additional authenticated data**

In GCM mode the authentication tag is appended after the ciphertext.

```bash
$ printf 'The quick brown fox jumps over the lazy dog.' | cchef aes-encrypt --key 00112233445566778899aabbccddeeff --iv ffeeddccbbaa99887766554433221100 --mode GCM --additional-authenticated-data 'additional data' --additional-authenticated-data-type UTF8
daa58faa056c52756aa488aeafbd265b6effcf4eca58220a97b0005b1a9b1e1c9e7a6725d35f5f79b9493de7

Tag: 3b5378917f67b0aade9891fc6c291646
```

---

## AES Key Unwrap

Reference: [Key wrap](https://wikipedia.org/wiki/Key_wrap)

Reverses AES Key Wrap (RFC 3394): decrypts wrapped 64-bit blocks with a
key-encryption key (KEK) and verifies the 64-bit integrity IV, failing with
`IV mismatch` if the wrapped data is corrupt.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key-kek` | string | (empty) | Key-encryption key; 16, 24 or 32 bytes. |
| `--key-kek-type` | option | `Hex` | KEK encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | `a6a6a6a6a6a6a6a6` | 64-bit integrity IV. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--input-format` | option | `Hex` | Input encoding: `Hex` or `Raw` bytes. |
| `--output-format` | option | `Hex` | Output encoding: `Hex` or `Raw` bytes. |

**Simple example**

```bash
$ printf '1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5' | cchef aes-key-unwrap --key-kek 000102030405060708090a0b0c0d0e0f
00112233445566778899aabbccddeeff
```

---

## AES Key Wrap

Reference: [Key wrap](https://wikipedia.org/wiki/Key_wrap)

Wraps key material using the RFC 3394 AES key-wrap algorithm: a key-encryption
key (KEK) and a 64-bit integrity IV protect 64-bit blocks. The input must be a
multiple of 8 bytes and at least 16 bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key-kek` | string | (empty) | Key-encryption key; 16, 24 or 32 bytes. |
| `--key-kek-type` | option | `Hex` | KEK encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | `a6a6a6a6a6a6a6a6` | 64-bit integrity IV. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--input-format` | option | `Hex` | Input encoding: `Hex` or `Raw` bytes. |
| `--output-format` | option | `Hex` | Output encoding: `Hex` or `Raw` bytes. |

**Simple example**

```bash
$ printf '00112233445566778899aabbccddeeff' | cchef aes-key-wrap --key-kek 000102030405060708090a0b0c0d0e0f
1fa68b0a8112b447aef34bd8fb5a7b829d3e862371d2cfe5
```

---

## Affine Cipher Decode

Reference: [Affine cipher](https://wikipedia.org/wiki/Affine_cipher)

Decrypts text enciphered with the Affine cipher. Each letter is mapped to its
position in the alphabet and transformed by `(y - b) * a⁻¹ % 26`, where `a⁻¹` is
the modular inverse of `a` modulo 26. Case is preserved and non-alphabetic
characters pass through unchanged. `a` and `b` must be non-negative integers and
`a` must be coprime to 26 (the same keys used to encode).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--a` | number | `1` | The multiplier `a`; must be coprime to 26. |
| `--b` | number | `0` | The additive constant `b`. |

**Simple example**

```bash
$ cchef affine-cipher-decode -i "Rclla, Oaplx!" --a 5 --b 8
Hello, World!
```

---

## Affine Cipher Encode

Reference: [Affine cipher](https://wikipedia.org/wiki/Affine_cipher)

Enciphers text with the Affine cipher, a monoalphabetic substitution where each
letter is mapped to its position in the alphabet and transformed by
`(ax + b) % 26`. Case is preserved and non-alphabetic characters pass through
unchanged. `a` and `b` must be non-negative integers and `a` must be coprime to
26; with `a = 1, b = 0` the input is unchanged.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--a` | number | `1` | The multiplier `a`; must be coprime to 26. |
| `--b` | number | `0` | The additive constant `b`. |

**Simple example**

```bash
$ cchef affine-cipher-encode -i "Hello, World!" --a 5 --b 8
Rclla, Oaplx!
```

**Complex example**

```bash
$ cchef affine-cipher-encode -i "some keys are shaped as locks. index[me]" --a 23 --b 23
vhnl tldv xyl vcxelo xv qhrtv. zkolg[nl]
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

## Ascon Decrypt

Reference: [Ascon (cipher)](https://wikipedia.org/wiki/Ascon_(cipher))

Ascon-AEAD128 authenticated decryption (NIST SP 800-232). Decrypts the
ciphertext (with its trailing 128-bit tag) and verifies authenticity: the key,
nonce and associated data must match those used to encrypt, or decryption fails.
The key and nonce must each be exactly 16 bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | 16-byte key, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--nonce` | string | (empty) | 16-byte nonce, interpreted per `--nonce-type`. |
| `--nonce-type` | option | `Hex` | Nonce encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--associated-data` | string | (empty) | Associated data, interpreted per `--associated-data-type`. |
| `--associated-data-type` | option | `Hex` | AD encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--input-format` | option | `Hex` | How to read the ciphertext: `Hex` or `Raw` bytes. |
| `--output-format` | option | `Raw` | How to render the plaintext: `Raw` bytes or `Hex`. |

**Simple example**

```bash
$ cchef ascon-decrypt -i af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0 --key 000102030405060708090a0b0c0d0e0f --nonce 000102030405060708090a0b0c0d0e0f
Hello
```

**With associated data**

The associated data must match what was used at encryption time, or
authentication fails.

```bash
$ cchef ascon-decrypt -i c5f46fb2c8f14b1d1006a0230236f4163573a24c5f30 --key 000102030405060708090a0b0c0d0e0f --nonce 101112131415161718191a1b1c1d1e1f --associated-data hdr-v1 --associated-data-type UTF8
Secret
```

---

## Ascon Encrypt

Reference: [Ascon (cipher)](https://wikipedia.org/wiki/Ascon_(cipher))

Ascon-AEAD128 authenticated encryption (NIST SP 800-232), a lightweight AEAD
scheme designed for constrained devices. The output is the ciphertext followed
by a 128-bit authentication tag. The key and nonce must each be exactly 16
bytes; never reuse a nonce with the same key. Associated data is authenticated
but not encrypted.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | 16-byte key, interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--nonce` | string | (empty) | 16-byte nonce, interpreted per `--nonce-type`. |
| `--nonce-type` | option | `Hex` | Nonce encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--associated-data` | string | (empty) | Associated data, interpreted per `--associated-data-type`. |
| `--associated-data-type` | option | `Hex` | AD encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--input-format` | option | `Raw` | How to read the input: `Raw` bytes or `Hex`. |
| `--output-format` | option | `Hex` | How to render the output: `Hex` or `Raw` bytes. |

**Simple example**

```bash
$ printf 'Hello' | cchef ascon-encrypt --key 000102030405060708090a0b0c0d0e0f --nonce 000102030405060708090a0b0c0d0e0f
af14bce6b9b6588c3aa63f9ddc5a0cf5f565f358b0
```

**With associated data**

Associated data (e.g. a header) is authenticated alongside the ciphertext; the
same value is required to decrypt.

```bash
$ printf 'Secret' | cchef ascon-encrypt --key 000102030405060708090a0b0c0d0e0f --nonce 101112131415161718191a1b1c1d1e1f --associated-data hdr-v1 --associated-data-type UTF8
c5f46fb2c8f14b1d1006a0230236f4163573a24c5f30
```

---

## Atbash Cipher

Reference: [Atbash](https://wikipedia.org/wiki/Atbash)

A mono-alphabetic substitution cipher that maps each letter to its mirror in the
alphabet (`a`<->`z`, `b`<->`y`, …). Case is preserved and non-alphabetic
characters pass through unchanged. Atbash takes no options and is its own
inverse, so running it a second time restores the original text.

**Simple example**

```bash
$ cchef atbash-cipher -i "The quick brown fox."
Gsv jfrxp yildm ulc.
```

---

## Bacon Cipher Decode

Reference: [Bacon's cipher](https://wikipedia.org/wiki/Bacon%27s_cipher)

Recovers a message concealed with the Baconian cipher, where each letter is
represented by five symbols. Invalid characters are stripped, the remaining
symbols are grouped into fives, and each group is looked up in the alphabet;
groups with no letter become `?`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | option | `Standard (I=J and U=V)` | `Standard (I=J and U=V)` (24 letters) or `Complete` (26 letters). |
| `--translation` | option | `0/1` | How the two symbols are represented: `0/1`, `A/B`, `Case` (upper=1, lower=0), or `A-M/N-Z first letter` (each word's first letter). |
| `--invert-translation` | boolean | `false` | Swap the two symbols before decoding. |

**Simple example**

```bash
$ cchef bacon-cipher-decode -i "00111 00100 01010 01010 01101"
HELLO
```

**Case translation**

Upper-case letters are ones and lower-case letters are zeroes; everything else is
ignored.

```bash
$ cchef bacon-cipher-decode -i "hELLo wORLd" --translation Case
PP
```

---

## Bacon Cipher Encode

Reference: [Bacon's cipher](https://wikipedia.org/wiki/Bacon%27s_cipher)

Conceals a message with the Baconian cipher, encoding each letter as five binary
digits (or `A`/`B`). By default non-letters are dropped and the output is grouped
into fives; "keep extra characters" instead preserves them inline.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | option | `Standard (I=J and U=V)` | `Standard (I=J and U=V)` (24 letters) or `Complete` (26 letters). |
| `--translation` | option | `0/1` | Symbols to emit: `0/1` or `A/B`. |
| `--keep-extra-characters` | boolean | `false` | Keep non-letters inline instead of dropping them and grouping into fives. |
| `--invert-translation` | boolean | `false` | Swap the two symbols in the output. |

**Simple example**

```bash
$ cchef bacon-cipher-encode -i "HELLO"
00111 00100 01010 01010 01101
```

**A/B, keeping extra characters**

```bash
$ cchef bacon-cipher-encode -i "Hi!" --translation A/B --keep-extra-characters
AABBBABAAA!
```

---

## Bcrypt

Reference: [Bcrypt](https://wikipedia.org/wiki/Bcrypt)

Hashes the input password with bcrypt, an adaptive password-hashing function
built on the Blowfish cipher. A random salt is generated each run, so the output
differs every time (verify a password with **Bcrypt compare**). The cost is
clamped to the 4–31 range and the output uses the `$2b$` version.

This operation is also listed under [Hashing](hashing.md).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rounds` | number | `10` | Cost (log2 of the iteration count); clamped to 4–31. |

**Simple example**

Because the salt is random, each invocation yields a different hash:

```bash
$ cchef bcrypt -i "hunter2" --rounds 8
$2b$08$S6G81rTdmnLNORA5WFAWIOqCFDmyB0Jpq3KI7m1eSFwV5hKocZmB6
```

---

## Bifid Cipher Decode

Reference: [Bifid cipher](https://wikipedia.org/wiki/Bifid_cipher)

Deciphers text enciphered with the Bifid cipher using the same keyword. The
keyword builds a 5×5 Polybius square (J is folded onto I); letter case is
preserved and non-alphabetic characters pass through unchanged. An empty keyword
uses the plain alphabet.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--keyword` | string | (empty) | Alphabet keyword (letters only); must match the one used to encode. |

**Simple example**

```bash
$ cchef bifid-cipher-decode -i "Kqhknw rm grsn" --keyword "Schrodinger"
Attack at dawn
```

---

## Bifid Cipher Encode

Reference: [Bifid cipher](https://wikipedia.org/wiki/Bifid_cipher)

Enciphers text with the Bifid cipher, which fractionates each letter's
coordinates in a keyword-seeded 5×5 Polybius square and transposes them. J is
folded onto I, letter case is preserved, and non-alphabetic characters pass
through unchanged.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--keyword` | string | (empty) | Alphabet keyword (letters only); an empty keyword uses the plain alphabet. |

**Simple example**

```bash
$ cchef bifid-cipher-encode -i "Attack at dawn" --keyword "Schrodinger"
Kqhknw rm grsn
```

**Without a keyword**

```bash
$ cchef bifid-cipher-encode -i "Hello, World!"
Fnpol, Parrd!
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
