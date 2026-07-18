# Public Key

Operations for public-key cryptography material — ECDSA keys and signatures,
certificates, and the ASN.1 structures they are built from.

The ECDSA operations are documented in full below. The ASN.1, PEM and hex
operations also belong to [Data format](data-format.md), and
[Parse SSH Host Key](networking.md#parse-ssh-host-key) to
[Networking](networking.md), where their detailed descriptions live.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| ECDSA Sign | `ecdsa-sign` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Signature Conversion | `ecdsa-signature-conversion` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| ECDSA Verify | `ecdsa-verify` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Generate ECDSA Key Pair | `generate-ecdsa-key-pair` | [ECDSA](https://wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm) |
| Hex to PEM | `hex-to-pem` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
| PEM to Hex | `pem-to-hex` | [PEM](https://wikipedia.org/wiki/Privacy-Enhanced_Mail) |
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
