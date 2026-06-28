# Encryption / Encoding

Classic ciphers and bitwise operations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| ROT13 | `rot13` | [ROT13](https://wikipedia.org/wiki/ROT13) |
| ROT47 | `rot47` | [ROT13 variants](https://wikipedia.org/wiki/ROT13#Variants) |
| XOR | `xor` | [XOR](https://wikipedia.org/wiki/XOR) |

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
