# Data format

Operations for encoding and decoding data between common textual representations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| AMF Decode | `amf-decode` | [Action Message Format](https://wikipedia.org/wiki/Action_Message_Format) |
| AMF Encode | `amf-encode` | [Action Message Format](https://wikipedia.org/wiki/Action_Message_Format) |
| From Base | `from-base` | [Radix](https://wikipedia.org/wiki/Radix) |
| From Base32 | `from-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| From Base45 | `from-base45` | [Base45](https://wikipedia.org/wiki/Base45) |
| From Base58 | `from-base58` | [Base58](https://wikipedia.org/wiki/Binary-to-text_encoding#Base58) |
| From Base62 | `from-base62` | [Base62](https://wikipedia.org/wiki/Base62) |
| From Base64 | `from-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| From Base85 | `from-base85` | [Ascii85](https://wikipedia.org/wiki/Ascii85) |
| From Base92 | `from-base92` | [Base92](https://wikipedia.org/wiki/List_of_numeral_systems) |
| From Hex | `from-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| From Octal | `from-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| To Base | `to-base` | [Radix](https://wikipedia.org/wiki/Radix) |
| To Base32 | `to-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| To Base45 | `to-base45` | [Base45](https://wikipedia.org/wiki/Base45) |
| To Base58 | `to-base58` | [Base58](https://wikipedia.org/wiki/Binary-to-text_encoding#Base58) |
| To Base62 | `to-base62` | [Base62](https://wikipedia.org/wiki/Base62) |
| To Base64 | `to-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| To Base85 | `to-base85` | [Ascii85](https://wikipedia.org/wiki/Ascii85) |
| To Base92 | `to-base92` | [Base92](https://wikipedia.org/wiki/List_of_numeral_systems) |
| To Hex | `to-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| To Octal | `to-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| URL Decode | `url-decode` | [Percent-encoding](https://wikipedia.org/wiki/Percent-encoding) |
| URL Encode | `url-encode` | [Percent-encoding](https://wikipedia.org/wiki/Percent-encoding) |

---

## AMF Decode

Deserializes Action Message Format (AMF) binary data into JSON. AMF is a binary
format used to serialize object graphs, e.g. between an Adobe Flash client and a
remote service.

Backed by the [`github.com/elobuff/goamf`](https://github.com/elobuff/goamf)
library (CyberChef likewise wraps an AMF library); the JSON representation of
decoded values follows that library.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--format` | option | `AMF3` | AMF version: `AMF0` or `AMF3`. |

**Simple example**

```bash
$ cchef from-hex --delimiter None -i 0200026869 | cchef amf-decode --format AMF0
"hi"
```

## AMF Encode

Serializes JSON into Action Message Format (AMF) binary data. The output is raw
bytes, so pipe through `to-hex` to view it.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--format` | option | `AMF3` | AMF version: `AMF0` or `AMF3`. |

**Simple example**

```bash
$ cchef amf-encode -i '{"a":1,"b":true}' | cchef to-hex --delimiter None
0a230103610362053ff000000000000003
```

**Round trip (encode then decode)**

```bash
$ printf '[1,2,3]' | cchef amf-encode | cchef amf-decode
[1,2,3]
```

---

## From Base

Converts a number from a given numerical base (radix 2–36) to decimal. Only
integer values are supported.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--radix` | number | `36` | The base of the input number (2–36). |

**Simple example**

```bash
$ cchef from-base --radix 16 -i ff
255
```

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

## From Base45

Decodes a Base45 string back into its raw byte value (RFC 9285).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `` 0-9A-Z $%*+\-./: `` | The Base45 alphabet. |
| `--remove-non-alphabet-chars` | bool | `true` | Strip characters outside the alphabet before decoding. |

**Simple example**

```bash
$ cchef from-base45 -i 'QED8WEX0'
ietf!
```

## From Base58

Decodes a Base58 string back into its raw byte value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | Bitcoin alphabet | The 58-character alphabet (Bitcoin by default; Ripple also common). |
| `--remove-non-alphabet-chars` | bool | `true` | Skip characters outside the alphabet. |

**Simple example**

```bash
$ cchef from-base58 -i 'StV1DL6CwTryKyV'
hello world
```

## From Base62

Decodes a Base62 string back into its raw byte value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `0-9A-Za-z` | The Base62 alphabet. |

**Simple example**

```bash
$ cchef from-base62 -i '1wJfrzvdbtXUOlUjUf'
Hello, World!
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

## From Base85

Decodes a Base85 (Ascii85) string back into its raw byte value.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `!-u` | The 85-character alphabet (Standard, Z85, or IPv6). |
| `--remove-non-alphabet-chars` | bool | `true` | Skip characters outside the alphabet. |
| `--all-zero-group-char` | string | `z` | Character representing an all-zero 4-byte group. |

**Simple example**

```bash
$ cchef from-base85 -i '9jqo^'
Man
```

(Delimited input like `<~9jqo^~>` is also accepted.)

## From Base92

Decodes a Base92 string back into its raw byte value. Takes no options.

**Simple example**

```bash
$ cchef from-base92 -i "G'_DW[B"
ietf!
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

## To Base

Converts a decimal number to a different numerical base (radix 2–36).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--radix` | number | `36` | The target base (2–36). |

**Simple example**

```bash
$ cchef to-base --radix 16 -i 255
ff
```

**Binary**

```bash
$ cchef to-base --radix 2 -i 255
11111111
```

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

## To Base45

Base45 encodes arbitrary byte data, used notably in QR codes (RFC 9285).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `` 0-9A-Z $%*+\-./: `` | The Base45 alphabet. |

**Simple example**

```bash
$ cchef to-base45 -i 'Hello!!'
%69 VD92EX0
```

## To Base58

Base58 encodes arbitrary byte data using an alphabet that omits easily-confused
characters. Commonly used for cryptocurrency addresses.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | Bitcoin alphabet | The 58-character alphabet. |

**Simple example**

```bash
$ cchef to-base58 -i 'hello world'
StV1DL6CwTryKyV
```

**Ripple alphabet**

```bash
$ cchef to-base58 --alphabet 'rpshnaf39wBUDNEGHJKLM4PQRST7VWXYZ2bcdeCg65jkm8oFqi1tuvAxyz' -i 'hello world'
StVrDLaUATiyKyV
```

## To Base62

Base62 encodes arbitrary byte data using alphanumeric characters by treating the
data as a large integer.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `0-9A-Za-z` | The Base62 alphabet. |

**Simple example**

```bash
$ cchef to-base62 -i 'Hello, World!'
1wJfrzvdbtXUOlUjUf
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

## To Base85

Base85 (Ascii85) encodes arbitrary byte data using 85 printable ASCII
characters, more space-efficient than Base64.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet` | string | `!-u` | The 85-character alphabet (Standard, Z85, or IPv6). |
| `--include-delimiter` | bool | `false` | Wrap the output in `<~` … `~>` delimiters. |

**Simple example**

```bash
$ cchef to-base85 -i 'Man '
9jqo^
```

**With delimiters**

```bash
$ cchef to-base85 --include-delimiter -i 'Man '
<~9jqo^~>
```

## To Base92

Base92 encodes arbitrary byte data using 91 printable ASCII characters. Takes no
options.

**Simple example**

```bash
$ cchef to-base92 -i 'Hello!!'
;K_$aOTo&
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
