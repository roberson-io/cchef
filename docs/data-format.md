# Data format

Operations for encoding and decoding data between common textual representations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| From Base32 | `from-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| From Base64 | `from-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| From Hex | `from-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| From Octal | `from-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| To Base32 | `to-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| To Base64 | `to-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| To Hex | `to-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| To Octal | `to-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| URL Decode | `url-decode` | [Percent-encoding](https://wikipedia.org/wiki/Percent-encoding) |
| URL Encode | `url-encode` | [Percent-encoding](https://wikipedia.org/wiki/Percent-encoding) |

---

## From Base32

Decodes a Base32 string back into its raw byte value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `A-Z2-7=` | The Base32 alphabet. |
| `--remove-non-alphabet-chars` | bool | `true` | Strip characters outside the alphabet before decoding. |

**Simple example**

```bash
$ cchef from-base32 -i 'JBSWY3DP'
Hello
```

## From Base64

Decodes data from an ASCII Base64 string back into its raw form.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `A-Za-z0-9+/=` | The Base64 alphabet. |
| `--remove-non-alphabet-chars` | bool | `true` | Strip characters outside the alphabet (e.g. newlines) before decoding. |
| `--strict-mode` | bool | `false` | Reserved for stricter validation. |

**Simple example**

```bash
$ cchef from-base64 -i 'SGVsbG8sIFdvcmxkIQ=='
Hello, World!
```

## From Hex

Converts a hexadecimal byte string back into its raw value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Auto` | `Auto` splits on any non-hex character. Other values match `to-hex`. |

**Simple example**

```bash
$ cchef from-hex -i '48 65 6c 6c 6f'
Hello
```

**Auto delimiter (mixed separators)**

```bash
$ cchef from-hex --delimiter 'Auto' -i '48:65,6c-6c6f'
Hello
```

## From Octal

Converts an octal byte string back into its raw value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

**Simple example**

```bash
$ cchef from-octal -i '110 145 154 154 157'
Hello
```

---

## To Base32

Base32 encodes arbitrary byte data using a restricted symbol set (usually `A-Z`
and `2-7`).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `A-Z2-7=` | The Base32 alphabet. Use `0-9A-V=` for Hex Extended. |

**Simple example**

```bash
$ cchef to-base32 -i 'Hello'
JBSWY3DP
```

## To Base64

Encodes raw data into an ASCII Base64 string.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `A-Za-z0-9+/=` | The Base64 alphabet (supercedes ranges like `A-Z`). A 64-character alphabet produces no padding. |

**Simple example**

```bash
$ cchef to-base64 -i 'Hello, World!'
SGVsbG8sIFdvcmxkIQ==
```

**Custom alphabet (URL-safe, no padding)**

```bash
$ printf '\xfb\xff' | cchef to-base64 --alphabet 'A-Za-z0-9-_'
-_8
```

## To Hex

Converts the input to hexadecimal bytes separated by the chosen delimiter.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Percent`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`, `0x`, `0x with comma`, `\x`, `None`. |

**Simple example**

```bash
$ cchef to-hex -i 'Hello'
48 65 6c 6c 6f
```

**Alternative delimiters**

```bash
$ cchef to-hex --delimiter 'Colon' -i 'Hello'
48:65:6c:6c:6f

$ cchef to-hex --delimiter '0x with comma' -i 'abc'
0x61,0x62,0x63
```

## To Octal

Converts the input to octal bytes separated by the chosen delimiter.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

**Simple example**

```bash
$ cchef to-octal -i 'Hello'
110 145 154 154 157
```

**Alternative delimiter**

```bash
$ cchef to-octal --delimiter 'Comma' -i 'Hello'
110,145,154,154,157
```

---

## URL Decode

Converts percent-encoded characters back to their raw values.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--treat-as-space` | bool | `true` | Treat `+` as a space (form encoding). |

**Simple example**

```bash
$ cchef url-decode -i 'Hello%20World%21'
Hello World!
```

## URL Encode

Encodes problematic characters into percent-encoding.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--encode-all-special-chars` | bool | `false` | When false, characters valid in URLs (e.g. `:/?#[]@!$&'()*+,;=`) are kept; when true, everything except `A-Za-z0-9` is encoded. |

**Simple example**

```bash
$ cchef url-encode -i 'Hello World!'
Hello%20World!
```

**Encode all special characters**

```bash
$ cchef url-encode --encode-all-special-chars -i 'Hello World!'
Hello%20World%21
```
