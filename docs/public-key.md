# Public Key

Operations for public-key cryptography material — RSA, ECDSA and PGP keys,
signatures and messages, certificates, and the ASN.1 structures they are built
from.

The RSA, ECDSA and PGP operations are documented in full below. The ASN.1, PEM and
hex operations also belong to [Data format](data-format.md), and
[Parse SSH Host Key](networking.md#parse-ssh-host-key) to
[Networking](networking.md), where their detailed descriptions live.

The RSA and ECDSA operations are backed by the Go standard library
(`crypto/rsa`, `crypto/x509`); CyberChef backs them with node-forge and jsrsasign.
Standard signatures (RSASSA-PKCS1-v1.5) and the RAW scheme are byte-identical to
CyberChef; OAEP/PKCS#1 v1.5 encryption is randomized, so ciphertext differs each
run but round-trips both ways. Two RSA fidelity notes: password-protected private
keys are supported for PKCS#1 (legacy PEM) but not PKCS#8-encrypted keys, and the
`Generate → JSON` output uses cchef's own key-parameter shape rather than
node-forge's internal serialization.

The PGP operations are backed by the maintained
[ProtonMail go-crypto](https://github.com/ProtonMail/go-crypto) OpenPGP library,
and interoperate with CyberChef's Keybase (`kbpgp`) implementation. Output is not
byte-identical to CyberChef (ASCII-armor headers and key structure differ), but
messages and keys round-trip in both directions.

The SM2 operations use the GM/T 0003 `sm2p256v1` curve. Encryption is randomized,
so an SM2 ciphertext round-trips through **SM2 Decrypt** rather than reproducing a
fixed byte string.

> Operations are listed alphabetically.
>
> Every string flag has a `--<flag>-file` companion that reads the value from a
> file, keeping keys and passphrases out of shell history — see
> [Reading argument values from files](README.md#reading-argument-values-from-files).

| Operation | Subcommand | Reference |
| --- | --- | --- |
| ECDSA Sign | `ecdsa-sign` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Signature Conversion | `ecdsa-signature-conversion` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Verify | `ecdsa-verify` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Generate ECDSA Key Pair | `generate-ecdsa-key-pair` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Generate PGP Key Pair | `generate-pgp-key-pair` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| Generate RSA Key Pair | `generate-rsa-key-pair` | [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem)) |
| Hex to Object Identifier | `hex-to-object-identifier` | [OID](https://wikipedia.org/wiki/Object_identifier) |
| Hex to PEM | `hex-to-pem` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
| JWK to PEM | `jwk-to-pem` | [JWK](https://datatracker.ietf.org/doc/html/rfc7517) |
| Object Identifier to Hex | `object-identifier-to-hex` | [OID](https://wikipedia.org/wiki/Object_identifier) |
| PEM to Hex | `pem-to-hex` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
| PEM to JWK | `pem-to-jwk` | [JWK](https://datatracker.ietf.org/doc/html/rfc7517) |
| PGP Decrypt | `pgp-decrypt` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Decrypt and Verify | `pgp-decrypt-and-verify` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Encrypt | `pgp-encrypt` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Encrypt and Sign | `pgp-encrypt-and-sign` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Sign | `pgp-sign` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Verify | `pgp-verify` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| Parse ASN.1 hex string | `parse-asn1-hex-string` | [ASN.1](https://wikipedia.org/wiki/Abstract_Syntax_Notation_One) |
| Parse CSR | `parse-csr` | [CSR](https://wikipedia.org/wiki/Certificate_signing_request) |
| Parse SSH Host Key | `parse-ssh-host-key` | [SSH](https://wikipedia.org/wiki/Secure_Shell) |
| Parse X.509 CRL | `parse-x509-crl` | [CRL](https://wikipedia.org/wiki/Certificate_revocation_list) |
| Parse X.509 certificate | `parse-x509-certificate` | [X.509](https://wikipedia.org/wiki/X.509) |
| Public Key from Certificate | `public-key-from-certificate` | [X.509](https://en.wikipedia.org/wiki/X.509) |
| Public Key from Private Key | `public-key-from-private-key` | [PKCS#8](https://en.wikipedia.org/wiki/PKCS_8) |
| RSA Decrypt | `rsa-decrypt` | [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem)) |
| RSA Encrypt | `rsa-encrypt` | [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem)) |
| RSA Sign | `rsa-sign` | [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem)) |
| RSA Verify | `rsa-verify` | [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem)) |
| SM2 Decrypt | `sm2-decrypt` | [SM2](https://datatracker.ietf.org/doc/html/draft-shen-sm2-ecdsa) |
| SM2 Encrypt | `sm2-encrypt` | [SM2](https://datatracker.ietf.org/doc/html/draft-shen-sm2-ecdsa) |

For Hex to PEM, PEM to Hex, Parse ASN.1 hex string and Parse SSH Host Key, see
[Data format](data-format.md) and [Networking](networking.md).

---

## ECDSA Sign

Reference: [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)

Signs the input message with a PEM-encoded EC private key (SEC1 `EC PRIVATE KEY`
or PKCS#8 `PRIVATE KEY`), over the NIST P-256/P-384/P-521 curves. The signature is
emitted in the chosen format. ECDSA uses a random nonce, so the signature differs
on every run (both are valid).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--ecdsa-private-key-pem` | string | (header only) | The EC private key in PEM. |
| `--message-digest-algorithm` | option | `SHA-256` | `SHA-256`, `SHA-384`, `SHA-512`, `SHA-1` or `MD5`. |
| `--output-format` | option | `ASN.1 HEX` | `ASN.1 HEX`, `P1363 HEX`, `JSON Web Signature` or `Raw JSON`. |

**Simple example** (with the key in `key.pem`):

```bash
cchef ecdsa-sign -i "Hello, World!" --ecdsa-private-key-pem "$(cat key.pem)" --output-format "ASN.1 HEX"
```

Output (a fresh ASN.1 DER signature, differing each run):

```
30440220722b14f5e7467fc7245fc133dda77c4763637bd318d242228637b542390327b402201a7cdcbf9f3a9b55309939aa38fb2eaf0722db4a639380f626dea43533887cf5
```

Verify it with [ECDSA Verify](#ecdsa-verify) and the matching public key.

---

## ECDSA Signature Conversion

Reference: [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)

Converts an ECDSA signature between its common encodings: ASN.1 DER hex, raw
`r‖s` (IEEE P1363) hex, a base64url JSON Web Signature, and a `{"r":…,"s":…}` JSON
object. With `--input-format Auto` the input encoding is detected automatically.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Auto` | `Auto`, `ASN.1 HEX`, `P1363 HEX`, `JSON Web Signature` or `Raw JSON`. |
| `--output-format` | option | `ASN.1 HEX` | `ASN.1 HEX`, `P1363 HEX`, `JSON Web Signature` or `Raw JSON`. |

**Simple example** — ASN.1 DER to raw P1363:

```bash
cchef ecdsa-signature-conversion -i "3046022100e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127022100b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d" --output-format "P1363 HEX"
```

Output:

```
e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d
```

---

## ECDSA Verify

Reference: [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)

Verifies a signature (the input) against a message and a PEM-encoded EC public
key, printing `Verified OK` or `Verification Failure`. The signature format is
detected automatically by default, and the message can be given as raw text, hex
or Base64.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Auto` | Signature format: `Auto`, `ASN.1 HEX`, `P1363 HEX`, `JSON Web Signature` or `Raw JSON`. |
| `--message-digest-algorithm` | option | `SHA-256` | `SHA-256`, `SHA-384`, `SHA-512`, `SHA-1` or `MD5`. |
| `--ecdsa-public-key-pem` | string | (header only) | The EC public key in PEM. |
| `--message` | string | (empty) | The message the signature is over. |
| `--message-format` | option | `Raw` | How to read the message: `Raw`, `Hex` or `Base64`. |

**Simple example** (public key in `pub.pem`):

```bash
cchef ecdsa-verify -i "3046022100e06905608a2fa7dbda9e284c2a7959dfb68fb527a5f003b2d7975ff135145127022100b6baa253793334f8b93ea1dd622bc600124d8090babd807efe3f77b8b324388d" --ecdsa-public-key-pem "$(cat pub.pem)" --message "A common mistake that people make when trying to design something completely foolproof is to underestimate the ingenuity of complete fools."
```

Output:

```
Verified OK
```

---

## Generate ECDSA Key Pair

Reference: [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)

Generates a fresh random EC key pair on the chosen curve. `PEM` emits the public
key then the PKCS#8 private key; `DER` emits the raw private scalar as hex; `JWK`
emits a JSON Web Key set containing both keys.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--elliptic-curve` | option | `P-256` | `P-256`, `P-384` or `P-521`. |
| `--output-format` | option | `PEM` | `PEM`, `DER` or `JWK`. |

**Simple example**

```bash
cchef generate-ecdsa-key-pair --elliptic-curve P-256 --output-format DER
```

Output (a fresh random private scalar, differing each run):

```
bfb723360863d8a402b52a75ff90540c41412d36457320915634f66d5316498b
```

---

## Generate PGP Key Pair

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Generates a fresh random PGP key pair (private key block followed by public key
block). RSA and NIST-curve ECC keys are supported. Optionally protect the private
key with a password and attach a name/email user ID.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--key-type` | option | `RSA-1024` | `RSA-1024/2048/4096` or `ECC-256/384/521`. |
| `--password-optional` | string | (empty) | Passphrase to encrypt the private key. |
| `--name-optional` | string | (empty) | User-ID name. |
| `--email-optional` | string | (empty) | User-ID email. |

**Simple example**

```bash
cchef generate-pgp-key-pair --key-type ECC-256 --name-optional Alice --email-optional alice@example.com > keypair.asc
```

The output contains a `PGP PRIVATE KEY BLOCK` then a `PGP PUBLIC KEY BLOCK` (fresh
random keys, differing each run). Split them into `priv.asc` / `pub.asc` for the
operations below.

---

## PGP Encrypt

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Encrypts the input to a recipient using their ASCII-armoured PGP public key,
producing an armoured `PGP MESSAGE`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--public-key-of-recipient` | string | (empty) | The recipient's armoured public key. |

**Simple example**

```bash
echo -n "secret message" | cchef pgp-encrypt --public-key-of-recipient "$(cat pub.asc)"
```

Output is an armoured `-----BEGIN PGP MESSAGE-----` block (differs each run).
Decrypt it with [PGP Decrypt](#pgp-decrypt) and the matching private key.

---

## PGP Decrypt

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Decrypts an armoured `PGP MESSAGE` with the recipient's private key, unlocking it
with the passphrase if the key is protected.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--private-key-of-recipient` | string | (empty) | The recipient's armoured private key. |
| `--private-key-passphrase` | string | (empty) | Passphrase, if the key is encrypted. |

**Simple example**

```bash
cchef pgp-decrypt --private-key-of-recipient "$(cat priv.asc)" --in-file message.asc
```

Output:

```
secret message
```

---

## PGP Sign

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Signs the input with the signer's private key, producing an armoured signed
`PGP MESSAGE` (verify it with [PGP Verify](#pgp-verify)).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--private-key-of-signer` | string | (empty) | The signer's armoured private key. |
| `--private-key-passphrase-optional` | string | (empty) | Passphrase, if the key is encrypted. |

**Simple example**

```bash
echo -n "signed message" | cchef pgp-sign --private-key-of-signer "$(cat priv.asc)"
```

Output is an armoured `PGP MESSAGE` containing the message and its signature.

---

## PGP Verify

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Verifies a signed `PGP MESSAGE` against the signer's public key, printing the
signer details and the message.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--public-key-of-signer` | string | (empty) | The signer's armoured public key. |

**Simple example**

```bash
cchef pgp-verify --public-key-of-signer "$(cat pub.asc)" --in-file signed.asc
```

Output (key ID, fingerprint and time vary by key and signing moment):

```
Signed by Alice <alice@example.com>
PGP key ID: DC8753A6
PGP fingerprint: 6c36ee8a3bc8414b1fae3d4289371858dc8753a6
Signed on Sun, 19 Jul 2026 00:27:21 GMT
----------------------------------
signed message
```

---

## PGP Encrypt and Sign

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Encrypts the input to a recipient **and** signs it with the sender's private key
in one armoured `PGP MESSAGE`. Decrypt and check the signature with
[PGP Decrypt and Verify](#pgp-decrypt-and-verify).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--private-key-of-signer` | string | (empty) | The sender's armoured private key. |
| `--private-key-passphrase` | string | (empty) | Passphrase, if the signer's key is encrypted. |
| `--public-key-of-recipient` | string | (empty) | The recipient's armoured public key. |

**Simple example**

```bash
echo -n "secret and signed" | cchef pgp-encrypt-and-sign \
    --private-key-of-signer "$(cat sender-priv.asc)" \
    --public-key-of-recipient "$(cat recipient-pub.asc)"
```

Output is an armoured, encrypted-and-signed `PGP MESSAGE`.

---

## PGP Decrypt and Verify

Reference: [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy)

Decrypts an encrypted-and-signed `PGP MESSAGE` with the recipient's private key
and verifies the sender's signature, printing the signer details and the message.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--public-key-of-signer` | string | (empty) | The sender's armoured public key. |
| `--private-key-of-recipient` | string | (empty) | The recipient's armoured private key. |
| `--private-key-password` | string | (empty) | Passphrase, if the recipient's key is encrypted. |

**Simple example**

```bash
cchef pgp-decrypt-and-verify \
    --public-key-of-signer "$(cat sender-pub.asc)" \
    --private-key-of-recipient "$(cat recipient-priv.asc)" \
    --in-file message.asc
```

The output is the same "Signed by …" report as [PGP Verify](#pgp-verify),
followed by the decrypted message.

---

## RSA Encrypt

Reference: [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem))

Encrypts the input with a PEM-encoded RSA public key (PKCS#1 `RSA PUBLIC KEY` or
SPKI `PUBLIC KEY`). Three schemes are offered: `RSA-OAEP` (with a selectable
digest for the label/MGF1 hash), `RSAES-PKCS1-V1_5`, and `RAW` (textbook RSA over
the input bytes). The ciphertext is raw bytes. OAEP and PKCS#1 v1.5 are
randomized; RAW is deterministic.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rsa-public-key-pem` | string | (header only) | The RSA public key in PEM. |
| `--encryption-scheme` | option | `RSA-OAEP` | `RSA-OAEP`, `RSAES-PKCS1-V1_5` or `RAW`. |
| `--message-digest-algorithm` | option | `SHA-1` | OAEP hash: `SHA-1`, `MD5`, `SHA-256`, `SHA-384` or `SHA-512`. |

**Simple example** — encrypt then decrypt, round-tripping through a pipe (with the
public key in `pub.pem` and the private key in `priv.pem`):

```bash
echo -n "secret message" | cchef rsa-encrypt --rsa-public-key-pem "$(cat pub.pem)" --encryption-scheme RSA-OAEP --message-digest-algorithm SHA-256 | cchef rsa-decrypt --rsa-private-key-pem "$(cat priv.pem)" --encryption-scheme RSA-OAEP --message-digest-algorithm SHA-256
```

Output:

```
secret message
```

---

## RSA Decrypt

Reference: [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem))

Decrypts an RSA-encrypted message (the input) with a PEM-encoded private key,
using the scheme it was encrypted with. The key may be PKCS#1 (`RSA PRIVATE KEY`,
optionally legacy-PEM-encrypted with a password) or PKCS#8 (`PRIVATE KEY`).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rsa-private-key-pem` | string | (header only) | The RSA private key in PEM. |
| `--key-password` | string | (empty) | Password for a legacy-PEM-encrypted key. |
| `--encryption-scheme` | option | `RSA-OAEP` | `RSA-OAEP`, `RSAES-PKCS1-V1_5` or `RAW`. |
| `--message-digest-algorithm` | option | `SHA-1` | OAEP hash, matching the encrypt side. |

See [RSA Encrypt](#rsa-encrypt) for a round-trip example. The ciphertext must be
exactly the modulus width (e.g. 256 bytes for a 2048-bit key), or decryption fails
with `Encrypted message length is invalid.`

---

## RSA Sign

Reference: [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem))

Signs the input message with a PEM-encoded RSA private key, producing an
RSASSA-PKCS1-v1.5 signature (raw bytes). The signature is deterministic.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rsa-private-key-pem` | string | (header only) | The RSA private key in PEM. |
| `--key-password` | string | (empty) | Password for a legacy-PEM-encrypted key. |
| `--message-digest-algorithm` | option | `SHA-1` | `SHA-1`, `MD5`, `SHA-256`, `SHA-384` or `SHA-512`. |

**Simple example** (hex-encoding the binary signature for display):

```bash
cchef rsa-sign -i "Hello, World!" --rsa-private-key-pem "$(cat priv.pem)" --message-digest-algorithm SHA-256 | cchef to-hex --delimiter None
```

Output:

```
8d7e39505b40c05e62adac1888a9515fd9c233cb5a4741509dce1fd1938baf174301c07150afef241f9dae27f328d439cc18cff4cd774aff73f2840a9e33f2333ae05e80c84e76170906ad2a74bb19a9f6199134faa480b34c9a49bef510732643a0b2eff8f2b861c94a962f5fe3683ff5291ffc8703de7b55fe647f19b28758b9866ce5955404aac82dc60bf4465d1c6a4d3f04721dbed7c05d725f0d01966ecf1b5f50422f327f3b3299dc2de7834a240c10bf0da5c0081adfee6a7e4acbfa4bc9d109db61dd3e7344dd60a39271c5c3d3dc32cc7a3b29dcb924cadec8cb269cc207171ad78e478351ea58daeea81218043bc8095fa90b0fc14380fcd3c391
```

---

## RSA Verify

Reference: [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem))

Verifies an RSASSA-PKCS1-v1.5 signature (the input) against a message and a
PEM-encoded RSA public key, printing `Verified OK` or `Verification Failure`. The
message can be given as raw text, hex or Base64.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rsa-public-key-pem` | string | (header only) | The RSA public key in PEM. |
| `--message` | string | (empty) | The message the signature covers. |
| `--message-format` | option | `Raw` | `Raw`, `Hex` or `Base64`. |
| `--message-digest-algorithm` | option | `SHA-1` | The digest used when signing. |

**Simple example** — verify the signature produced by [RSA Sign](#rsa-sign)
(passed as raw bytes via `from-hex`):

```bash
cchef rsa-sign -i "Hello, World!" --rsa-private-key-pem "$(cat priv.pem)" --message-digest-algorithm SHA-256 | cchef rsa-verify --rsa-public-key-pem "$(cat pub.pem)" --message "Hello, World!" --message-format Raw --message-digest-algorithm SHA-256
```

Output:

```
Verified OK
```

---

## Generate RSA Key Pair

Reference: [RSA](https://wikipedia.org/wiki/RSA_(cryptosystem))

Generates a fresh RSA key pair of the chosen bit length. `PEM` emits an SPKI public
key followed by a PKCS#1 private key; `DER` emits the raw PKCS#1 private-key DER;
`JSON` emits the key parameters as a JSON object (cchef's own shape — hex-encoded
integers — which differs from node-forge's).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--rsa-key-length` | option | `1024` | `1024`, `2048` or `4096` bits. |
| `--output-format` | option | `PEM` | `PEM`, `JSON` or `DER`. |

**Simple example** (output is random and abbreviated here):

```bash
cchef generate-rsa-key-pair --rsa-key-length 2048 --output-format PEM
```

Output:

```
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A...
-----END PUBLIC KEY-----

-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----
```

## Object Identifier to Hex

Reference: [OID](https://wikipedia.org/wiki/Object_identifier)

Encodes a dotted-decimal object identifier as the hex of its ASN.1 content octets:
the first two arcs collapse into one byte (`40 × arc1 + arc2`) and each remaining
arc is written base-128 (seven bits per byte, the high bit set on all but the last).
This is a port of jsrsasign's `oidIntToHex`, including its quirks — a single-arc
input yields `NaN`, and any character outside `[0-9.]` is a `malformed oid string`
error.

**Example**

```bash
cchef object-identifier-to-hex -i '1.2.840.113549.1.1.1'
```

Output:

```
2a864886f70d010101
```

## Hex to Object Identifier

Reference: [OID](https://wikipedia.org/wiki/Object_identifier)

Decodes the hex of ASN.1 OID content octets back into dotted-decimal notation
(the inverse of Object Identifier to Hex). Whitespace in the input is ignored, and
the entire input is treated as content octets — a leading tag/length is decoded as
extra arcs rather than skipped. Ported from jsrsasign's `oidHexToInt`.

**Example**

```bash
cchef hex-to-object-identifier -i '2a8648ce3d0201'
```

Output:

```
1.2.840.10045.2.1
```

## PEM to JWK

Reference: [JWK](https://datatracker.ietf.org/doc/html/rfc7517)

Converts PEM keys and certificates to [JSON Web Key](https://datatracker.ietf.org/doc/html/rfc7517)
format. Each PEM block in the input is parsed and emitted as a compact JWK; when
the input holds several blocks, their JWKs are newline-separated. RSA keys
(PKCS#1 or PKCS#8, public or private), EC keys (SEC1 or PKCS#8, over P-256/P-384/P-521),
and X.509 certificates (the certificate's public key) are supported. PKCS#1 RSA
public keys (`-----BEGIN RSA PUBLIC KEY-----`) and DSA keys are rejected, as in
CyberChef. The conversion is backed by Go's `crypto/x509`; CyberChef backs it with
jsrsasign, but the JWK output is identical.

**Example**

```bash
cchef pem-to-jwk -i '-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----'
```

Output:

```
{"kty":"RSA","n":"8qvQOnph0i3M5-TpruZrsvgEXgud6Uxgq1ugYuuTqKG2oU9kVEs1wmLrwe-e3yy0ys_nS3qOrBZDYSMx2SPp-w","e":"AQAB"}
```

## JWK to PEM

Reference: [JWK](https://datatracker.ietf.org/doc/html/rfc7517)

Converts keys in JSON Web Key format to PEM (the inverse of PEM to JWK): private
keys as PKCS#8, public keys as SPKI, with CRLF line endings. The input may be a
single JWK, a JSON array of JWKs, or a JWK Set (`{"keys":[…]}`); every key is
converted and the PEM blocks concatenated. Only RSA and EC (P-256/P-384/P-521)
key types are supported.

**Example**

```bash
cchef jwk-to-pem -i '{"kty":"EC","crv":"P-256","x":"DUc8A0EDNKoCYIPWMHz1yUzqE5mJgusgcAE8H6810fk","y":"CfGZkzYggmurC4Edrw9VTYdnYoq1oCjx-D1TCmr-Xuk"}'
```

Output:

```
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEDUc8A0EDNKoCYIPWMHz1yUzqE5mJ
gusgcAE8H6810fkJ8ZmTNiCCa6sLgR2vD1VNh2diirWgKPH4PVMKav5e6Q==
-----END PUBLIC KEY-----
```

## Parse X.509 certificate

Reference: [X.509](https://wikipedia.org/wiki/X.509)

Displays the contents of an X.509 certificate in a human-readable form similar to
`openssl x509 -text`: version, serial number, signature algorithm, validity,
issuer/subject distinguished names, MD5/SHA-1/SHA-256 fingerprints, the public
key (RSA or EC), the certificate signature, and the v3 extensions.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `PEM` | `PEM`, `DER Hex`, `Base64` or `Raw`. |

**Example** (output abbreviated)

```bash
cchef parse-x509-certificate --in-file cert.pem
```

Output:

```
Version:          3 (0x02)
Serial number:    325195407439739868690796149162879408866195452662 (0x38f6...)
Algorithm ID:     SHA256withRSA
Validity
  Not Before:     20/07/2026 01:42:23 (dd-mm-yyyy hh:mm:ss) (260720014223Z)
  Not After:      17/07/2036 01:42:23 (dd-mm-yyyy hh:mm:ss) (360717014223Z)
Issuer
  CN = example.com
Subject
  CN = example.com
...
Extensions
  basicConstraints CRITICAL:
    {}
  keyUsage CRITICAL:
    digitalSignature,keyEncipherment
  extKeyUsage :
    serverAuth, clientAuth
```

## Parse CSR

Reference: [CSR](https://wikipedia.org/wiki/Certificate_signing_request)

Parses a PKCS#10 Certificate Signing Request, showing the subject, public key
(RSA, EC or DSA), signature, and the requested extensions (basic constraints,
key usage, extended key usage, subject alternative name).
(verified against its fixture suite).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `PEM` | `PEM`. |

**Example** (output abbreviated)

```bash
cchef parse-csr --in-file request.csr
```

Output:

```
Subject
  CN = example.com
Public Key
  Algorithm:      ECDSA
  Length:         256 bits
  Pub:            04:09:a9:61:73:61:f8:bf:44:...
  ASN1 OID:       secp256r1
  NIST CURVE:     P-256
Signature
  Algorithm:      SHA256withECDSA
  Signature:      30:45:02:20:42:4b:a6:fe:...
Requested Extensions
  Basic Constraints: critical
    CA = false
  Key Usage: critical
    Digital Signature
    Key encipherment
  Subject Alternative Name:
    DNS: example.com
    DNS: www.example.com
```

## Parse X.509 CRL

Reference: [CRL](https://wikipedia.org/wiki/Certificate_revocation_list)

Parses a Certificate Revocation List, showing its version, signature algorithm,
issuer, update times, CRL extensions, the revoked-certificate entries (with their
entry extensions such as reason code and invalidity date), and the signature.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `PEM` | `PEM`, `DER Hex`, `Base64` or `Raw`. |

**Example** (output abbreviated)

```bash
cchef parse-x509-crl --in-file list.crl
```

Output:

```
Certificate Revocation List (CRL):
    Version: 2 (0x1)
    Signature Algorithm: SHA256withRSA
    Issuer:
        C  = UK
        CN = Test Root CA
    Last Update: Sun, 25 Aug 2024 11:49:10 GMT
    Next Update: Tue, 24 Sep 2024 11:49:10 GMT
Revoked Certificates:
    Serial Number: 1000
        Revocation Date: Sun, 25 Aug 2024 03:23:08 GMT
    	CRL entry extensions:
            X509v3 CRL Reason Code:
                Certificate Hold
Signature Value:
        03:1b:2b:fb:d9:c4:2d:45:...
```

---

## Public Key from Certificate

Reference: [X.509](https://en.wikipedia.org/wiki/X.509)

Extracts the public key (the `SubjectPublicKeyInfo`) from one or more PEM X.509
certificates and emits each as a `PUBLIC KEY` PEM block. If the input contains
several certificates, every extracted key is returned in order. RSA, EC and DSA
keys are supported; EdDSA (Ed25519/Ed448) certificates are rejected as an
unsupported key type, matching CyberChef.

This operation takes no options.

**Example**

```bash
cchef public-key-from-certificate --in-file rsa.crt
```

Output:

```
-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----
```

---

## Public Key from Private Key

Reference: [PKCS#8](https://en.wikipedia.org/wiki/PKCS_8)

Extracts the public key from one or more PEM private keys and emits each as a
`PUBLIC KEY` PEM block. Traditional (PKCS#1 RSA, SEC1 EC, OpenSSL DSA) and PKCS#8
keys are accepted; for RSA and DSA the public value is read directly, and for EC
it is derived from the private scalar. DSA keys in PKCS#8 (which omit the public
value) and EdDSA keys are rejected, matching CyberChef.

This operation takes no options.

**Example**

```bash
cchef public-key-from-private-key --in-file rsa.key
```

Output:

```
-----BEGIN PUBLIC KEY-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAPKr0Dp6YdItzOfk6a7ma7L4BF4LnelM
YKtboGLrk6ihtqFPZFRLNcJi68Hvnt8stMrP50t6jqwWQ2EjMdkj6fsCAwEAAQ==
-----END PUBLIC KEY-----
```

---

## SM2 Decrypt

Reference: [SM2](https://datatracker.ietf.org/doc/html/draft-shen-sm2-ecdsa)

Decrypts a message with the SM2 public-key algorithm (the Chinese GM/T 0003
standard) over the `sm2p256v1` curve. The input is the hex-encoded ciphertext
package (C1 ‖ C3 ‖ C2 or C1 ‖ C2 ‖ C3); the private key is 32 bytes of hex. The
recovered plaintext is authenticated against the embedded C3 (SM3) tag — a
mismatch is reported as an error.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--private-key` | string | `DEADBEEF` | The 32-byte private key, in hex. |
| `--input-format` | option | `C1C3C2` | Component order of the ciphertext: `C1C3C2` or `C1C2C3`. |
| `--curve` | option | `sm2p256v1` | The elliptic curve (only `sm2p256v1` is defined). |

**Example**

```bash
cchef sm2-decrypt --private-key e74a72505084c3269aa9b696d603e3e08c74c6740212c11a31e26cdfe08bdf6a --input-format C1C3C2 -i 9a31bc0adb4677cdc4141479e3949572a55c3e6fb52094721f741c2bd2e179aaa87be6263bc1be602e473be3d5de5dce97f8248948b3a7e15f9f67f64aef21575e0c05e6171870a10ff9ab778dbef24267ad90e1a9d47d68f757d57c4816612e9829f804025dea05a511cda39371c22a2828f976f72e
```

Output:

```
I am a small plaintext
```

---

## SM2 Encrypt

Reference: [SM2](https://datatracker.ietf.org/doc/html/draft-shen-sm2-ecdsa)

Encrypts a message with the SM2 public-key algorithm over the `sm2p256v1` curve,
producing the hex-encoded ciphertext package. The public key is supplied as its
two 32-byte coordinates (hex). Encryption draws a fresh random scalar each run,
so the ciphertext differs every time; it round-trips back through **SM2 Decrypt**
with the corresponding private key, and interoperates with CyberChef.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--public-key-x` | string | `DEADBEEF` | The public key's X coordinate, 32 bytes of hex. |
| `--public-key-y` | string | `DEADBEEF` | The public key's Y coordinate, 32 bytes of hex. |
| `--output-format` | option | `C1C3C2` | Component order of the ciphertext: `C1C3C2` or `C1C2C3`. |
| `--curve` | option | `sm2p256v1` | The elliptic curve (only `sm2p256v1` is defined). |

**Example** (piped straight into SM2 Decrypt, since the ciphertext is randomized)

```bash
echo -n "Secret message" | cchef sm2-encrypt \
  --public-key-x f7d903cab7925066c31150a92b31e548e63f954f92d01eaa0271fb2a336baef8 \
  --public-key-y fb0c45e410ef7a6cdae724e6a78dbff52562e97ede009e762b667d9b14adea6c | \
  cchef sm2-decrypt --private-key e74a72505084c3269aa9b696d603e3e08c74c6740212c11a31e26cdfe08bdf6a
```

Output:

```
Secret message
```
