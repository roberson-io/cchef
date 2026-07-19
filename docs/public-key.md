# Public Key

Operations for public-key cryptography material — ECDSA and PGP keys, signatures
and messages, certificates, and the ASN.1 structures they are built from.

The ECDSA and PGP operations are documented in full below. The ASN.1, PEM and hex
operations also belong to [Data format](data-format.md), and
[Parse SSH Host Key](networking.md#parse-ssh-host-key) to
[Networking](networking.md), where their detailed descriptions live.

The PGP operations are backed by the maintained
[ProtonMail go-crypto](https://github.com/ProtonMail/go-crypto) OpenPGP library,
and interoperate with CyberChef's Keybase (`kbpgp`) implementation. Output is not
byte-identical to CyberChef (ASCII-armor headers and key structure differ), but
messages and keys round-trip in both directions.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| ECDSA Sign | `ecdsa-sign` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Signature Conversion | `ecdsa-signature-conversion` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Verify | `ecdsa-verify` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Generate ECDSA Key Pair | `generate-ecdsa-key-pair` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Generate PGP Key Pair | `generate-pgp-key-pair` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| Hex to PEM | `hex-to-pem` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
| PEM to Hex | `pem-to-hex` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
| PGP Decrypt | `pgp-decrypt` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Decrypt and Verify | `pgp-decrypt-and-verify` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Encrypt | `pgp-encrypt` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Encrypt and Sign | `pgp-encrypt-and-sign` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Sign | `pgp-sign` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| PGP Verify | `pgp-verify` | [PGP](https://wikipedia.org/wiki/Pretty_Good_Privacy) |
| Parse ASN.1 hex string | `parse-asn1-hex-string` | [ASN.1](https://wikipedia.org/wiki/Abstract_Syntax_Notation_One) |
| Parse SSH Host Key | `parse-ssh-host-key` | [SSH](https://wikipedia.org/wiki/Secure_Shell) |

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
