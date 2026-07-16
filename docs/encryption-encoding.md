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
| Blowfish Decrypt | `blowfish-decrypt` | [Blowfish](https://wikipedia.org/wiki/Blowfish_(cipher)) |
| Blowfish Encrypt | `blowfish-encrypt` | [Blowfish](https://wikipedia.org/wiki/Blowfish_(cipher)) |
| Bombe | `bombe` | [Bombe](https://wikipedia.org/wiki/Bombe) |
| Caesar Box Cipher | `caesar-box-cipher` | [Caesar Box](https://www.dcode.fr/caesar-box-cipher) |
| Cetacean Cipher Decode | `cetacean-cipher-decode` | [Dolphins](https://hitchhikers.fandom.com/wiki/Dolphins) |
| Cetacean Cipher Encode | `cetacean-cipher-encode` | [Dolphins](https://hitchhikers.fandom.com/wiki/Dolphins) |
| ChaCha | `chacha` | [ChaCha variant](https://wikipedia.org/wiki/Salsa20#ChaCha_variant) |
| CipherSaber2 Decrypt | `ciphersaber2-decrypt` | [CipherSaber](https://wikipedia.org/wiki/CipherSaber) |
| CipherSaber2 Encrypt | `ciphersaber2-encrypt` | [CipherSaber](https://wikipedia.org/wiki/CipherSaber) |
| Citrix CTX1 Decode | `citrix-ctx1-decode` | [Citrix CTX1](https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/) |
| Citrix CTX1 Encode | `citrix-ctx1-encode` | [Citrix CTX1](https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/) |
| Colossus | `colossus` | [Colossus computer](https://wikipedia.org/wiki/Colossus_computer) |
| Derive EVP key | `derive-evp-key` | [Key derivation function](https://wikipedia.org/wiki/Key_derivation_function) |
| Enigma | `enigma` | [Enigma machine](https://wikipedia.org/wiki/Enigma_machine) |
| Lorenz | `lorenz` | [Lorenz cipher](https://wikipedia.org/wiki/Lorenz_cipher) |
| Multiple Bombe | `multiple-bombe` | [Bombe](https://wikipedia.org/wiki/Bombe) |
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

## Blowfish Decrypt

Reference: [Blowfish](https://wikipedia.org/wiki/Blowfish_(cipher))

Decrypts Blowfish ciphertext. The key must be 4–56 bytes and, for every mode
except ECB, the IV must be exactly 8 bytes. CBC and ECB expect PKCS#7-padded,
block-aligned input; decryption fails if the padding or length is invalid.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | Key (4–56 bytes), interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | (empty) | 8-byte IV (ignored for ECB), interpreted per `--iv-type`. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--mode` | option | `CBC` | Mode: `CBC`, `CFB`, `OFB`, `CTR`, `ECB`. |
| `--input-format` | option | `Hex` | How to read the ciphertext: `Hex` or `Raw` bytes. |
| `--output-format` | option | `Raw` | How to render the plaintext: `Raw` bytes or `Hex`. |

**Simple example**

```bash
$ cchef blowfish-decrypt -i 398433f39e938286a35fc240521435b6972f3fe96846b54ab9351aa5fa9e10a6a94074e883d1cb36cb9657c817274b60 --key 0011223344556677 --iv ffeeddccbbaa9988 --mode CBC
The quick brown fox jumps over the lazy dog.
```

---

## Blowfish Encrypt

Reference: [Blowfish](https://wikipedia.org/wiki/Blowfish_(cipher))

Encrypts input with the Blowfish block cipher (64-bit block). The key must be
4–56 bytes and, for every mode except ECB, the IV must be exactly 8 bytes. CBC
and ECB apply PKCS#7 padding; CFB, OFB and CTR are streaming and leave the length
unchanged.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key` | string | (empty) | Key (4–56 bytes), interpreted per `--key-type`. |
| `--key-type` | option | `Hex` | Key encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--iv` | string | (empty) | 8-byte IV (ignored for ECB), interpreted per `--iv-type`. |
| `--iv-type` | option | `Hex` | IV encoding: `Hex`, `UTF8`, `Latin1`, `Base64`. |
| `--mode` | option | `CBC` | Mode: `CBC`, `CFB`, `OFB`, `CTR`, `ECB`. |
| `--input-format` | option | `Raw` | How to read the input: `Raw` bytes or `Hex`. |
| `--output-format` | option | `Hex` | How to render the output: `Hex` or `Raw` bytes. |

**Simple example**

```bash
$ cchef blowfish-encrypt -i "The quick brown fox jumps over the lazy dog." --key 0011223344556677 --iv ffeeddccbbaa9988 --mode CBC
398433f39e938286a35fc240521435b6972f3fe96846b54ab9351aa5fa9e10a6a94074e883d1cb36cb9657c817274b60
```

**Streaming mode (CTR)**

CTR leaves the length unchanged (no padding):

```bash
$ cchef blowfish-encrypt -i "secret" --key 0011223344556677 --iv 0000000000000000 --mode CTR
c5a8e6a22ed5
```

---

## Bombe

Reference: [Bombe](https://wikipedia.org/wiki/Bombe)

Emulation of the Bombe machine used at Bletchley Park to attack Enigma. Given the
ciphertext, a **crib** (known plaintext for part of it) and the rotors used, it
suggests Enigma configurations — each a set of rotor start positions (left to
right), the plugboard pairs it could determine, and a decryption preview. Choose
a crib whose menu has loops (2+ is desirable); the checking machine discards
stops that fail verification. Output is an HTML table.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--model` | option | `3-rotor` | `3-rotor` or `4-rotor`. |
| `--left-most-4th-rotor` | string | Beta wiring | 4th-slot rotor wiring (4-rotor only). |
| `--left-hand-rotor` / `--middle-rotor` / `--right-hand-rotor` | string | I / II / III | Rotor wirings (stepping is ignored). |
| `--reflector` | string | reflector B | Reflector pairs. |
| `--crib` | string | (empty) | Known plaintext to match against the ciphertext. |
| `--crib-offset` | number | `0` | Offset into the ciphertext where the crib begins. |
| `--use-checking-machine` | boolean | `true` | Verify each stop and discard failures. |

**Simple example**

```bash
$ cchef bombe -i "BBYFLTHHYIJQAYBBYS" --crib "THISISATESTMESSAGE"
Bombe run on menu with 6 loops (2+ desirable). Note: Rotor positions are listed left to right and start at the beginning of the crib, and ignore stepping and the ring setting. Some plugboard settings are determined. A decryption preview starting at the beginning of the crib and ignoring stepping is also provided.

<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Rotor stops</th>  <th>Partial plugboard</th>  <th>Decryption preview</th></tr>
<tr><td>LGA</td>  <td>SS AG BO CL EK FF HH II JJ TT YY</td>  <td>THISISATESTMESSAGE</td></tr>
</table>
```

(The default rotors are I/II/III with reflector B, so they can be omitted here.)

---

## Caesar Box Cipher

Reference: [Caesar Box](https://www.dcode.fr/caesar-box-cipher)

A transposition cipher: the message (with spaces removed) is written row by row
into a box `Box Height` rows tall, then read back column by column. Encryption and
decryption are the same operation with complementary heights — a message encoded
with height *h* into a box of width *w* is decoded by re-running it with height *w*.

| Option | Description |
| --- | --- |
| Box Height | The number of rows in the box. |

Example — encode with a box three rows tall:

```
$ cchef caesar-box-cipher --box-height 3 -i "Hello World!"
Hlodeor!lWl
```

Decoding reverses it. `Hello World!` (11 letters after stripping the space) fills a
3×4 box, so the inverse height is 4:

```
$ cchef caesar-box-cipher --box-height 4 -i "Hlodeor!lWl"
HelloWorld!
```

---

## Cetacean Cipher Decode

Reference: [Dolphins](https://hitchhikers.fandom.com/wiki/Dolphins)

Decodes Cetacean Cipher text back to the original message. Each run of `e`/`E`
characters is read in groups of sixteen as a 16-bit character code (`e` is a 1
bit, `E` a 0 bit); a literal space stands for a space character.

This operation takes no arguments.

```
$ cchef cetacean-cipher-decode -i "EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe"
hi
```

---

## Cetacean Cipher Encode

Reference: [Dolphins](https://hitchhikers.fandom.com/wiki/Dolphins)

Converts any input into Cetacean Cipher: each character is written as its 16-bit
code, with `e` for a 1 bit and `E` for a 0 bit. Spaces are passed through
unchanged, so words stay separated.

This operation takes no arguments.

```
$ cchef cetacean-cipher-encode -i "hi"
EEEEEEEEEeeEeEEEEEEEEEEEEeeEeEEe
```

---

## ChaCha

Reference: [ChaCha variant](https://wikipedia.org/wiki/Salsa20#ChaCha_variant)

ChaCha is Daniel J. Bernstein's stream cipher (a Salsa20 variant). This is a
parameterised implementation covering both the original construction and the
RFC-8439 variant. As a stream cipher, encryption and decryption are the same
operation: re-running the ciphertext with the same key, nonce and counter
returns the plaintext.

| Option | Description |
| --- | --- |
| Key | 16- or 32-byte key (128 or 256 bits), given as Hex, UTF8, Latin1 or Base64. |
| Nonce | 8- or 12-byte nonce, as Hex/UTF8/Latin1/Base64, or an integer (which becomes a 12-byte nonce). The nonce and counter together must total 16 bytes. |
| Counter | Starting block counter (default 0); incremented every 64 bytes of keystream. |
| Rounds | Number of rounds: 20, 12 or 8. |
| Input | How to read the input: `Hex` or `Raw`. |
| Output | How to write the output: `Raw` or `Hex`. |

Encrypt a message (256-bit key, 12-byte nonce, 20 rounds, raw text in, hex out):

```
$ cchef chacha --key 000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f \
    --nonce 000000000000004a00000000 --counter 1 --rounds 20 \
    --input-format Raw --output-format Hex -i "Hello, ChaCha!"
6a 2a 3d 9f 2f 37 f9 a2 47 bf 64 07 d9 42
```

Decrypt by feeding that ciphertext back in with the same parameters:

```
$ cchef chacha --key 000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f \
    --nonce 000000000000004a00000000 --counter 1 --rounds 20 \
    --input-format Hex --output-format Raw -i "6a 2a 3d 9f 2f 37 f9 a2 47 bf 64 07 d9 42"
Hello, ChaCha!
```

An 8-round keystream with a 128-bit key and an 8-byte nonce (so the counter is 8
bytes) — the draft-strombergson TC7.1 test vector:

```
$ cchef chacha --key 00112233445566778899aabbccddeeff --nonce 0f1e2d3c4b5a6978 \
    --counter 0 --rounds 8 --input-format Hex --output-format Hex \
    -i "00 00 00 00 00 00 00 00"
29 56 0d 28 0b 45 28 40
```

---

## CipherSaber2 Decrypt

Reference: [CipherSaber](https://wikipedia.org/wiki/CipherSaber)

Decrypts CipherSaber-2 ciphertext. The first 10 bytes of the input are the
initialisation vector and the rest is the message, keyed with the RC4 stream
cipher mixed over the given number of rounds. Use the same key and round count
that were used to encrypt.

| Option | Description |
| --- | --- |
| Key | The shared key, as Hex, UTF8, Latin1 or Base64. |
| Rounds | Number of key-scheduling rounds (20 for CipherSaber-2; 1 is classic CipherSaber/RC4). |

The classic worked example (key `asdfg`, 1 round), reading the raw ciphertext
bytes on stdin:

```
$ printf '\x6f\x6d\x0b\xab\xf3\xaa\x67\x19\x03\x15\x30\xed\xb6\x77\xca\x74\xe0\x08\x9d\xd0\xe7\xb8\x85\x43\x56\xbb\x14\x48\xe3\x7c\xdb\xef\xe7\xf3\xa8\x4f\x4f\x5f\xb3\xfd' \
    | cchef ciphersaber2-decrypt --key asdfg --key-type Latin1 --rounds 1
This is a test of CipherSaber.
```

---

## CipherSaber2 Encrypt

Reference: [CipherSaber](https://wikipedia.org/wiki/CipherSaber)

Encrypts with CipherSaber-2. A fresh random 10-byte initialisation vector is
generated and prepended to the output, so the ciphertext is always 10 bytes
longer than the input and differs on each run. Decrypt with the same key and
round count.

| Option | Description |
| --- | --- |
| Key | The shared key, as Hex, UTF8, Latin1 or Base64. |
| Rounds | Number of key-scheduling rounds (20 for CipherSaber-2; 1 is classic CipherSaber/RC4). |

Because the IV is random, encryption is shown here round-tripped back through
decryption with the same key and rounds:

```
$ printf 'Meet at dawn.' \
    | cchef ciphersaber2-encrypt --key hunter2 --key-type Latin1 --rounds 20 \
    | cchef ciphersaber2-decrypt --key hunter2 --key-type Latin1 --rounds 20
Meet at dawn.
```

---

## Citrix CTX1 Decode

Reference: [Citrix CTX1](https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/)

Decodes a Citrix CTX1 password hash back to plaintext. The input length must be a
multiple of four (each source character encodes to four `A`–`P` characters);
otherwise the operation errors with `Incorrect hash length`.

This operation takes no arguments.

```
$ cchef citrix-ctx1-decode -i "NFHALEBBMHGCLEBBMDGGKMAJNOHLLKBP"
password
```

---

## Citrix CTX1 Encode

Reference: [Citrix CTX1](https://www.reddit.com/r/AskNetsec/comments/1s3r6y/citrix_ctx1_hash_decoding/)

Encodes a string to the Citrix CTX1 password format. The text is UTF-16LE
encoded, folded through a running XOR chain, and emitted as pairs of `A`–`P`
characters (one per nibble).

This operation takes no arguments.

```
$ cchef citrix-ctx1-encode -i "password"
NFHALEBBMHGCLEBBMDGGKMAJNOHLLKBP
```

---

## Colossus

Reference: [Colossus computer](https://wikipedia.org/wiki/Colossus_computer)

Emulates Colossus, the WW2 codebreaking computer built to attack the Lorenz
cipher. It runs the ciphertext tape repeatedly against the Chi/Psi/Motor wheel
patterns, programmed via the "K rack" of Q-bus switches, and counts how often a
condition holds — the statistical test cryptanalysts used to recover wheel
settings. Input is ITA2 text (`A`–`Z`, `3`–`9`, `+ - . /`); output is a JSON
object with the printer output, the five counters and the run count.

This operation has ~57 arguments mirroring the physical machine's controls;
`cchef colossus --help` lists them all. The most useful are below. The simplest
way to drive it is `--k-rack-option "Select Program"` with a preset
`--program-to-run`.

**Key options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--pattern` | option | `KH Pattern` | Wheel pattern: `KH Pattern`, `ZMUG Pattern` or `BREAM Pattern`. |
| `--qbusz` / `--qbus-2` / `--qbus-3` | option | (empty) | Q-bus inputs: Z (cipher), Χ (chi) and Ψ (psi), each blank, direct or delta (`Δ`). |
| `--limitation` | option | `None` | Motor limitation (`Χ2`, `Χ2 + P5`, `X2 + Ψ1`, …). |
| `--k-rack-option` | option | `Select Program` | `Select Program` runs a preset; the others expose the raw switches. |
| `--program-to-run` | option | (empty) | Preset: `Letter Count`, `1+2=.`, `4=5=/1=2`, `/,5,U`. |
| `--set-total` | number | `0` | Only print counter lines above this threshold. |
| `--fast-step` / `--slow-step` | option | (empty) | Wheels stepped between runs (`X1`–`X5`, `M37`, `M61`, `S1`–`S5`). |
| `--start-1` … | number | `1` | Per-wheel start positions (Χ, Ψ and motor wheels). |

**Simple example**

The "Letter Count" program counts every character on the tape (here 30):

```bash
$ cchef colossus -i "CTBKJUVXHZ-H3L4QV+YEZUK+SXOZ/N" \
    --k-rack-option "Select Program" --program-to-run "Letter Count" --qbusz Z
{"printout":" \n00 00 : a30 \n","counters":[30,0,0,0,0],"runcount":2}
```

**Stepping a wheel**

Setting a fast step runs the tape once per position of that wheel, printing a
line per run:

```bash
$ cchef colossus -i "CTBKJUVXHZ-H3L4QV+YEZUK+SXOZ/N" \
    --k-rack-option "Select Program" --program-to-run "Letter Count" \
    --qbusz Z --fast-step X1
{"printout":"X1 \n01 00 : a30 \n02 00 : a30 \n03 00 : a30 \n...41 00 : a30 \n","counters":[30,0,0,0,0],"runcount":42}
```

---

## Derive EVP key

Reference: [Key derivation function](https://wikipedia.org/wiki/Key_derivation_function)

Runs the OpenSSL `EVP_BytesToKey` password-based key derivation function (as used
by crypto-js). It repeatedly hashes the passphrase and salt to produce key
material of the requested size, returned as a lowercase hex string. The operation
**input is ignored** — the passphrase and salt come from the arguments.

Despite the description text (inherited from CyberChef), this operation does
**not** generate a random salt: the salt is exactly the decoded salt argument, so
an empty salt means no salt. That makes the output fully deterministic.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--passphrase` / `--passphrase-type` | toggleString | (empty) / `UTF8` | The passphrase and how to decode it (`UTF8`, `Latin1`, `Hex`, `Base64`). |
| `--key-size` | number | `128` | Derived key size in **bits**. |
| `--iterations` | number | `1` | Hash iterations per block. |
| `--hashing-function` | option | `SHA1` | `SHA1`, `SHA256`, `SHA384`, `SHA512` or `MD5`. |
| `--salt` / `--salt-type` | toggleString | (empty) / `Hex` | The salt and how to decode it (`Hex`, `UTF8`, `Latin1`, `Base64`). |

**Simple example**

Derive a 128-bit key from a passphrase and a hex salt with SHA1:

```bash
$ cchef derive-evp-key --passphrase password --salt 73616c74 --salt-type Hex
c88e9c67041a74e0357befdff93f87dd
```

**Complex example**

A 256-bit key with SHA256, 3 iterations and a UTF8 salt:

```bash
$ cchef derive-evp-key --passphrase password --key-size 256 --iterations 3 \
    --hashing-function SHA256 --salt salt --salt-type UTF8
cc19a87959d70ba1d9d2979b5fc2323e0d62a40fb2545492e9ec4d57ce79956d
```

---

## Enigma

Reference: [Enigma machine](https://wikipedia.org/wiki/Enigma_machine)

Enciphers/deciphers with the WW2 Enigma machine. The standard German military
rotors (I–VIII, Beta, Gamma) and reflectors (B, C, and the thin variants) are
built in. Enigma is its own inverse: decrypt by running the ciphertext through a
machine set up the same way. Because this is a substitution machine, encryption
and decryption are the same operation.

Rotors are given as the 26-letter wiring the rotor maps A→Z onto, optionally
followed by `<` and the stepping points (e.g. `EKMFLGDQVZNTOWYHXUSPAIBRCJ<R`);
reflectors and the plugboard are whitespace-separated letter pairs
(e.g. `AB CD EF`).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--model` | option | `3-rotor` | `3-rotor` or `4-rotor` (the 4th slot uses a thin rotor/reflector). |
| `--left-most-4th-rotor` | string | Beta wiring | 4th-slot rotor wiring (4-rotor only). |
| `--left-most-rotor-ring-setting` / `--left-most-rotor-initial-value` | option | `A` | Ring setting and start position of the 4th rotor. |
| `--left-hand-rotor` | string | rotor I | Left rotor wiring (`wiring<steps`). |
| `--left-hand-rotor-ring-setting` / `--left-hand-rotor-initial-value` | option | `A` | Ring setting and start position of the left rotor. |
| `--middle-rotor` | string | rotor II | Middle rotor wiring. |
| `--middle-rotor-ring-setting` / `--middle-rotor-initial-value` | option | `A` | Ring setting and start position of the middle rotor. |
| `--right-hand-rotor` | string | rotor III | Right rotor wiring. |
| `--right-hand-rotor-ring-setting` / `--right-hand-rotor-initial-value` | option | `A` | Ring setting and start position of the right rotor. |
| `--reflector` | string | reflector B | Reflector pairs (13 pairs covering every letter). |
| `--plugboard` | string | (empty) | Plugboard pairs, e.g. `AB CD`. |
| `--strict-output` | boolean | `true` | Drop non-letters and group the output into blocks of five. |

**Simple example**

With the default rotors (I, II, III at position A) `--strict-output` groups the
output into fives:

```bash
$ cchef enigma -i "HELLOWORLD"
ILBDA AMTAZ
```

**Round trip**

The same settings decrypt it (Enigma is self-inverse):

```bash
$ cchef enigma -i "ILBDAAMTAZ"
HELLO WORLD
```

**With a plugboard and rotor start positions**

```bash
$ cchef enigma -i "ATTACKATDAWN" --left-hand-rotor-initial-value Q --middle-rotor-initial-value E --right-hand-rotor-initial-value V --plugboard "AB CD"
UQKUK TOKGQ CB
```

---

## Lorenz

Reference: [Lorenz cipher](https://wikipedia.org/wiki/Lorenz_cipher)

Enciphers/deciphers with the Lorenz SZ40/42 cipher attachment — a twelve-wheel
Vernam machine that XORs the plaintext (in ITA2) with a key stream from five chi
wheels, five psi wheels and two motor wheels. Three models (`SZ40`, `SZ42a`,
`SZ42b`) and three historical wheel patterns (KH, ZMUG, BREAM) are built in, plus
a `Custom` pattern that reads twelve lug strings (`.`/`x`).

Set `--mode Send` to encipher and `--mode Receive` to decipher. Plaintext is
converted to/from ITA2 with figure/letter shifts; `--input-type ITA2` /
`--output-type ITA2` skip that conversion. The full per-wheel start and lug flags
are listed by `cchef lorenz --help`.

**Key options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--model` | option | `SZ40` | `SZ40`, `SZ42a` or `SZ42b` (the SZ42 models add limitations). |
| `--wheel-pattern` | option | `KH Pattern` | `KH Pattern`, `ZMUG Pattern`, `BREAM Pattern`, `No Pattern` or `Custom`. |
| `--kt-schalter` | boolean | `false` | KT-Schalter limitation (SZ42a/b). |
| `--mode` | option | `Send` | `Send` to encipher, `Receive` to decipher. |
| `--input-type` / `--output-type` | option | `Plaintext` | `Plaintext` or `ITA2`. |
| `--ita2-format` | option | `5/8/9` | Represent figure-shift/letter-shift/space as `5/8/9` or `+/-/.`. |
| `--<wheel>-start-…` | number | `1` | Start position of each Ψ/Μ/Χ wheel. |
| `--<wheel>-lugs-…` | string | pattern | Custom lug strings (used when `--wheel-pattern Custom`). |

**Simple example**

Encipher plaintext with the KH pattern (ITA2 output, `9` = space):

```bash
$ cchef lorenz -i "HELLO WORLD, THIS IS A TEST MESSAGE." --model SZ40 --wheel-pattern "KH Pattern" --mode Send
VIC3TS/CUJA/3II9W9JWDI5DAFXT4SOIF3999IZD9T
```

**Round trip**

The same settings in `Receive` mode recover the plaintext:

```bash
$ cchef lorenz -i "VIC3TS/CUJA/3II9W9JWDI5DAFXT4SOIF3999IZD9T" --model SZ40 --wheel-pattern "KH Pattern" --mode Receive --output-type Plaintext
HELLO WORLD, THIS IS A TEST MESSAGE.
```

---

## Multiple Bombe

Reference: [Bombe](https://wikipedia.org/wiki/Bombe)

Like [Bombe](#bombe), but runs the attack over many rotor configurations when the
rotors are unknown. Supply candidate rotors (one wiring per line) and reflectors;
it tries every ordering and reports the configurations that produce stops. Test
your crib with the single Bombe first — a 4-rotor search is very slow.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--main-rotors` | string | (empty) | Candidate rotor wirings, one per line (3+ required). |
| `--4th-rotor` | string | (empty) | Candidate 4th-slot rotors, one per line (optional). |
| `--reflectors` | string | (empty) | Candidate reflectors, one per line. |
| `--crib` | string | (empty) | Known plaintext to match against the ciphertext. |
| `--crib-offset` | number | `0` | Offset into the ciphertext where the crib begins. |
| `--use-checking-machine` | boolean | `true` | Verify each stop and discard failures. |

**Simple example**

```bash
$ cchef multiple-bombe -i "BBYFLTHHYIJQAYBBYS" --main-rotors "$(printf 'EKMFLGDQVZNTOWYHXUSPAIBRCJ\nAJDKSIRUXBLHWTMCQGZNPYFVOE\nBDFHJLCPRTXVZNYEIWGAKMUSQO')" --reflectors "AY BR CU DH EQ FS GL IP JX KN MO TZ VW" --crib "THISISATESTMESSAGE"
Bombe run on menu with 6 loops (2+ desirable). Note: Rotors and rotor positions are listed left to right, ignore stepping and the ring setting, and positions start at the beginning of the crib. Some plugboard settings are determined. A decryption preview starting at the beginning of the crib and ignoring stepping is also provided.

Rotors: EKMFLGDQVZNTOWYHXUSPAIBRCJ, AJDKSIRUXBLHWTMCQGZNPYFVOE, BDFHJLCPRTXVZNYEIWGAKMUSQO
Reflector: AY BR CU DH EQ FS GL IP JX KN MO TZ VW
<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Rotor stops</th>  <th>Partial plugboard</th>  <th>Decryption preview</th></tr>
<tr><td>LGA</td>  <td>SS AG BO CL EK FF HH II JJ TT YY</td>  <td>THISISATESTMESSAGE</td></tr>
</table>
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
