# Data format

Operations for encoding and decoding data between common textual representations.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| AMF Decode | `amf-decode` | [Action Message Format](https://wikipedia.org/wiki/Action_Message_Format) |
| AMF Encode | `amf-encode` | [Action Message Format](https://wikipedia.org/wiki/Action_Message_Format) |
| Caret/M-decode | `caret-m-decode` | [Caret notation](https://en.wikipedia.org/wiki/Caret_notation) |
| Escape Unicode Characters | `escape-unicode-characters` | [Unicode](https://wikipedia.org/wiki/Unicode) |
| From Base | `from-base` | [Radix](https://wikipedia.org/wiki/Radix) |
| From Base32 | `from-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| From Base45 | `from-base45` | [Base45](https://wikipedia.org/wiki/Base45) |
| From Base58 | `from-base58` | [Base58](https://wikipedia.org/wiki/Binary-to-text_encoding#Base58) |
| From Base62 | `from-base62` | [Base62](https://wikipedia.org/wiki/Base62) |
| From Base64 | `from-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| From Base85 | `from-base85` | [Ascii85](https://wikipedia.org/wiki/Ascii85) |
| From Base92 | `from-base92` | [Base92](https://wikipedia.org/wiki/List_of_numeral_systems) |
| From BCD | `from-bcd` | [Binary-coded decimal](https://wikipedia.org/wiki/Binary-coded_decimal) |
| From Binary | `from-binary` | [Binary](https://wikipedia.org/wiki/Binary_number) |
| From Charcode | `from-charcode` | [Character encoding](https://wikipedia.org/wiki/Character_encoding) |
| From Decimal | `from-decimal` | [Decimal](https://wikipedia.org/wiki/Decimal) |
| From Float | `from-float` | [IEEE 754](https://wikipedia.org/wiki/IEEE_754) |
| From Hex | `from-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| From Hexdump | `from-hexdump` | [Hex dump](https://wikipedia.org/wiki/Hex_dump) |
| From HTML Entity | `from-html-entity` | [HTML character entities](https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references) |
| From Modhex | `from-modhex` | [ModHex](https://en.wikipedia.org/wiki/YubiKey#ModHex) |
| From Octal | `from-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| From Quoted Printable | `from-quoted-printable` | [Quoted-Printable](https://wikipedia.org/wiki/Quoted-printable) |
| Swap endianness | `swap-endianness` | [Endianness](https://wikipedia.org/wiki/Endianness) |
| Text-Integer Conversion | `text-integer-conversion` | [Endianness](https://wikipedia.org/wiki/Endianness) |
| To Base | `to-base` | [Radix](https://wikipedia.org/wiki/Radix) |
| To Base32 | `to-base32` | [Base32](https://wikipedia.org/wiki/Base32) |
| To Base45 | `to-base45` | [Base45](https://wikipedia.org/wiki/Base45) |
| To Base58 | `to-base58` | [Base58](https://wikipedia.org/wiki/Binary-to-text_encoding#Base58) |
| To Base62 | `to-base62` | [Base62](https://wikipedia.org/wiki/Base62) |
| To Base64 | `to-base64` | [Base64](https://wikipedia.org/wiki/Base64) |
| To Base85 | `to-base85` | [Ascii85](https://wikipedia.org/wiki/Ascii85) |
| To Base92 | `to-base92` | [Base92](https://wikipedia.org/wiki/List_of_numeral_systems) |
| To BCD | `to-bcd` | [Binary-coded decimal](https://wikipedia.org/wiki/Binary-coded_decimal) |
| To Binary | `to-binary` | [Binary](https://wikipedia.org/wiki/Binary_number) |
| To Charcode | `to-charcode` | [Character encoding](https://wikipedia.org/wiki/Character_encoding) |
| To Decimal | `to-decimal` | [Decimal](https://wikipedia.org/wiki/Decimal) |
| To Float | `to-float` | [IEEE 754](https://wikipedia.org/wiki/IEEE_754) |
| To Hex | `to-hex` | [Hexadecimal](https://wikipedia.org/wiki/Hexadecimal) |
| To Hexdump | `to-hexdump` | [Hex dump](https://wikipedia.org/wiki/Hex_dump) |
| To HTML Entity | `to-html-entity` | [HTML character entities](https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references) |
| To Modhex | `to-modhex` | [ModHex](https://en.wikipedia.org/wiki/YubiKey#ModHex) |
| To Octal | `to-octal` | [Octal](https://wikipedia.org/wiki/Octal) |
| To Quoted Printable | `to-quoted-printable` | [Quoted-Printable](https://wikipedia.org/wiki/Quoted-printable) |
| Unescape Unicode Characters | `unescape-unicode-characters` | [Unicode](https://wikipedia.org/wiki/Unicode) |
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

## Caret/M-decode

Decodes caret-notation and M-notation escapes as produced by tools such as
`cat -v`: `^M` becomes a carriage return (`0x0d`), `^I` a tab, and `M-^A`
becomes `0x81`. Note that `cat -v` leaves `^_` unencoded even though it is a
valid encoding of `0x1f`.

**Options**

This operation takes no options.

**Simple example**

```bash
$ cchef caret-m-decode -i '^M^JHello M-^A' | cchef to-hex
0d 0a 48 65 6c 6c 6f 20 81
```

## Escape Unicode Characters

Converts characters to their unicode-escaped notation. By default only
non-printable and non-ASCII characters are escaped and printable ASCII is left
unchanged; pass `--encode-all-chars` to escape everything. Non-BMP characters are
escaped as UTF-16 surrogate pairs.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--prefix` | option | `\u` | Escape prefix: `\u`, `%u`, or `U+`. |
| `--encode-all-chars` | boolean | `false` | Escape every character, not just non-printable/non-ASCII ones. |
| `--padding` | number | `4` | Minimum hex digits per escape (zero-padded; never truncates). |
| `--uppercase-hex` | boolean | `true` | Use upper-case hex digits. |

**Simple example**

```bash
$ cchef escape-unicode-characters -i 'Héllo'
H\u00E9llo
```

**Encode everything with the U+ prefix**

```bash
$ cchef escape-unicode-characters --prefix 'U+' --encode-all-chars -i 'Hi'
U+0048U+0069
```

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

## From BCD

Decodes a [Binary-Coded Decimal](https://wikipedia.org/wiki/Binary-coded_decimal)
value into a decimal number. Each decimal digit is represented by a fixed group
of bits under the chosen encoding scheme; a trailing nibble may carry the sign.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--scheme` | option | `8 4 2 1` | Encoding scheme: `8 4 2 1`, `7 4 2 1`, `4 2 2 1`, `2 4 2 1`, `8 4 -2 -1`, `Excess-3`, `IBM 8 4 2 1`. |
| `--packed` | boolean | `true` | Two digits per byte (packed) vs one digit per byte (unpacked). |
| `--signed` | boolean | `false` | Treat the final nibble as a sign (`D`/`B` = negative). |
| `--input-format` | option | `Nibbles` | Input representation: `Nibbles`, `Bytes`, or `Raw`. |

**Simple example**

```bash
$ cchef from-bcd -i '0001 0010 0011 0100'
1234
```

**Packed, signed bytes (negative value)**

```bash
$ cchef from-bcd --packed --signed --input-format Bytes \
    -i '00000001 00100011 01000101 01100111 10001001 00001101'
-1234567890
```

## From Binary

Converts a binary string back into its raw form.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`, `None`. |
| `--byte-length` | number | `8` | Number of bits per byte. |

**Simple example**

```bash
$ cchef from-binary -i '01001000 01101001'
Hi
```

## From Charcode

Converts unicode character codes (in the given base) back into text.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |
| `--base` | number | `16` | Radix of the character codes (2–36). |

**Simple example**

```bash
$ cchef from-charcode -i '41 42'
AB
```

## From Decimal

Converts a delimited list of decimal byte values back into raw bytes.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |
| `--support-signed-values` | bool | `false` | Interpret negative values as their unsigned byte equivalents. |

**Simple example**

```bash
$ cchef from-decimal -i '72 73'
HI
```

## From Float

Converts decimal numbers into their [IEEE 754](https://wikipedia.org/wiki/IEEE_754)
floating-point byte representation.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--endianness` | option | `Big Endian` | `Big Endian` or `Little Endian`. |
| `--size` | option | `Float (4 bytes)` | `Float (4 bytes)` (single precision) or `Double (8 bytes)` (double precision). |
| `--delimiter` | option | `Space` | Separator between numbers: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

**Simple example**

```bash
$ cchef from-float -i '0.5 0.5' | cchef to-hex --delimiter None
3f0000003f000000
```

**Little-endian double precision**

```bash
$ cchef from-float --endianness 'Little Endian' --size 'Double (8 bytes)' -i '0.5' \
    | cchef to-hex --delimiter None
000000000000e03f
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

## From Hexdump

Attempts to convert a hexdump back into raw data. Many tool formats are
supported (xxd, Wireshark, 010 Editor, `hexdump -C`, …); the offset and ASCII
preview columns are ignored and only the hex bytes are decoded. Verify the
result is correct before relying on it.

**Options**

This operation takes no options.

**Simple example**

```bash
$ cchef from-hexdump -i '00000000  48 65 6c 6c 6f 2c 20 57 6f 72 6c 64 21           |Hello, World!|'
Hello, World!
```

## From HTML Entity

Converts [HTML character entities](https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references)
back into raw characters. Named entities (`&amp;`), decimal entities (`&#233;`),
and hexadecimal entities (`&#xe9;`) are all decoded; unrecognised entities are
left untouched.

**Options**

This operation takes no options.

**Simple example**

```bash
$ cchef from-html-entity -i '&amp; &lt; &#233; &#x20ac;'
& < é €
```

## From Modhex

Converts a [modhex](https://en.wikipedia.org/wiki/YubiKey#ModHex) byte string
back into its raw value. Modhex substitutes the 16 hex nibbles with the consonant
alphabet `cbdefghijklnrtuv` (used by YubiKey to be keyboard-layout independent).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Auto` | `Auto` splits on any non-modhex character. Also: `Space`, `Percent`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`, `None`. |

**Simple example**

```bash
$ cchef from-modhex -i 'hb hd hg id ik ie if ii ik if hj'
aberystwyth
```

**Auto delimiter (mixed case, mixed separators)**

```bash
$ cchef from-modhex --delimiter Auto -i 'uhKGkb,UHkgkB,UGltlk,ugltkc'
救救孩子
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

## From Quoted Printable

Decodes [Quoted-Printable](https://wikipedia.org/wiki/Quoted-printable) text back
into raw bytes: `=XX` escapes become their byte value, soft line breaks (`=` at
end of line) are removed, and everything else passes through.

**Options**

This operation takes no options.

**Simple example**

```bash
$ cchef from-quoted-printable -i 'a=3Db =26 caf=C3=A9'
a=b & café
```

## Swap endianness

Reverses the byte order within fixed-length words.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--data-format` | option | `Hex` | `Hex` or `Raw`. Hex output is space-delimited. |
| `--word-length-bytes` | number | `4` | Bytes per word. |
| `--pad-incomplete-words` | bool | `true` | Zero-pad a trailing word shorter than the word length. |

**Simple example**

```bash
$ cchef swap-endianness --data-format Hex --word-length-bytes 4 -i 0a0b0c0d
0d 0c 0b 0a
```

**Raw data**

```bash
$ cchef swap-endianness --data-format Raw --word-length-bytes 2 -i ABCD
BADC
```

## Text-Integer Conversion

Converts between text and a large integer, treating the text as a big-endian
sequence of character codes (e.g. `ABC` is `0x414243` is `4276803`). The input
format is auto-detected: `0x…` is hexadecimal, plain digits are decimal, and
anything else (optionally wrapped in single or double quotes) is text. Text may
only contain ASCII/Latin-1 characters (code point < 256); multi-byte Unicode
characters produce an error.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--output-format` | option | `String` | Output as `String`, `Decimal`, or `Hexadecimal`. |

**Simple example**

```bash
$ cchef text-integer-conversion --output-format Hexadecimal -i '"CyberChef"'
0x437962657243686566
```

**Integer back to text**

```bash
$ cchef text-integer-conversion --output-format String -i '0x48656C6C6F'
Hello
```

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

## To BCD

Encodes a decimal number as [Binary-Coded Decimal](https://wikipedia.org/wiki/Binary-coded_decimal),
representing each digit with a fixed group of bits under the chosen scheme.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--scheme` | option | `8 4 2 1` | Encoding scheme: `8 4 2 1`, `7 4 2 1`, `4 2 2 1`, `2 4 2 1`, `8 4 -2 -1`, `Excess-3`, `IBM 8 4 2 1`. |
| `--packed` | boolean | `true` | Two digits per byte (packed) vs one digit per byte (unpacked). |
| `--signed` | boolean | `false` | Append a sign nibble (`C` = +, `D` = -). |
| `--output-format` | option | `Nibbles` | Output representation: `Nibbles`, `Bytes`, or `Raw`. |

**Simple example**

```bash
$ cchef to-bcd -i '1234'
0001 0010 0011 0100
```

**Packed, signed bytes**

```bash
$ cchef to-bcd --packed --signed --output-format Bytes -i '1234567890'
00000001 00100011 01000101 01100111 10001001 00001100
```

## To Binary

Displays the input as a binary string, each byte zero-padded to the given length.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`, `None`. |
| `--byte-length` | number | `8` | Number of bits per byte. |

**Simple example**

```bash
$ cchef to-binary -i 'Hi'
01001000 01101001
```

## To Charcode

Converts text to its unicode character codes, in the given base.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |
| `--base` | number | `16` | Radix of the character codes (2–36). |

**Simple example**

```bash
$ cchef to-charcode -i 'AB'
41 42
```

**Base 10**

```bash
$ cchef to-charcode --base 10 -i 'AB'
65 66
```

## To Decimal

Converts the input to a delimited list of decimal byte values.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |
| `--support-signed-values` | bool | `false` | Treat each byte as a signed value (−128…127). |

**Simple example**

```bash
$ cchef to-decimal -i 'ABC'
65 66 67
```

## To Float

Interprets the input bytes as [IEEE 754](https://wikipedia.org/wiki/IEEE_754)
floating-point numbers and prints their decimal values.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--endianness` | option | `Big Endian` | `Big Endian` or `Little Endian`. |
| `--size` | option | `Float (4 bytes)` | `Float (4 bytes)` (single precision) or `Double (8 bytes)` (double precision). |
| `--delimiter` | option | `Space` | Separator between numbers: `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`. |

The input length must be a multiple of the chosen size (4 or 8 bytes).

**Simple example**

```bash
$ cchef from-hex -i '3f0000003f000000' | cchef to-float
0.5 0.5
```

**Big-endian double precision**

```bash
$ cchef from-hex -i '3fe0000000000000' | cchef to-float --size 'Double (8 bytes)'
0.5
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

## To Hexdump

Creates a hexdump of the input: an offset column, the hexadecimal value of each
byte, and an ASCII preview alongside (non-printable bytes shown as `.`).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--width` | number | `16` | Bytes per line (1–65536). |
| `--upper-case-hex` | boolean | `false` | Upper-case the hex and offset columns. |
| `--include-final-length` | boolean | `false` | Append a final line with the total byte length. |
| `--unix-format` | boolean | `false` | Preview only ASCII `0x20`–`0x7e`; otherwise Latin-1 printable characters are shown. |

**Simple example**

```bash
$ cchef to-hexdump -i 'Hello, World!'
00000000  48 65 6c 6c 6f 2c 20 57 6f 72 6c 64 21           |Hello, World!|
```

**Narrow width, upper-case, with final length**

```bash
$ cchef to-hexdump --width 8 --upper-case-hex --include-final-length -i 'Hello, World!'
00000000  48 65 6C 6C 6F 2C 20 57  |Hello, W|
00000008  6F 72 6C 64 21           |orld!|
0000000d
```

## To HTML Entity

Converts characters to [HTML character entities](https://wikipedia.org/wiki/List_of_XML_and_HTML_character_entity_references).
By default only characters with a named entity (and code points above 255) are
converted; the rest pass through. Use `--convert-all-characters` and
`--convert-to` to control the output form.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--convert-all-characters` | boolean | `false` | Convert every character, not just those with a named entity. |
| `--convert-to` | option | `Named entities` | `Named entities`, `Numeric entities` (`&#233;`), or `Hex entities` (`&#xe9;`). |

**Simple example**

```bash
$ cchef to-html-entity -i 'a & b < "c"'
a &amp; b &lt; &quot;c&quot;
```

**Numeric entities, converting everything**

```bash
$ cchef to-html-entity --convert-all-characters --convert-to 'Numeric entities' -i 'Hé'
&#72;&#233;
```

## To Modhex

Converts the input to [modhex](https://en.wikipedia.org/wiki/YubiKey#ModHex)
bytes separated by the chosen delimiter. Modhex substitutes the 16 hex nibbles
with the consonant alphabet `cbdefghijklnrtuv`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Space` | One of: `Space`, `Percent`, `Comma`, `Semi-colon`, `Colon`, `Line feed`, `CRLF`, `None`. |
| `--bytes-per-line` | number | `0` | Insert a line break after this many bytes (`0` = never). |

**Simple example**

```bash
$ cchef to-modhex -i 'aberystwyth'
hb hd hg id ik ie if ii ik if hj
```

**Alternative delimiter with line wrapping**

```bash
$ cchef to-modhex --delimiter Comma --bytes-per-line 4 -i 'aberystwyth'
hb,hd,hg,id,
ik,ie,if,ii,
ik,if,hj
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

## To Quoted Printable

Encodes bytes as [Quoted-Printable](https://wikipedia.org/wiki/Quoted-printable)
(RFC 2045): bytes outside the printable set become `=XX`, lines are kept to 76
characters with `=`-terminated soft breaks, and trailing whitespace is escaped.

**Options**

This operation takes no options.

**Simple example**

```bash
$ cchef to-quoted-printable -i 'a=b & café'
a=3Db & caf=C3=A9
```

## Unescape Unicode Characters

Converts unicode-escaped character notation back into the raw characters it
represents. Text outside the escapes is passed through unchanged. With the `U+`
prefix, 4- to 6-digit escapes are accepted (so astral code points work); the
`\u` and `%u` prefixes take exactly 4 digits.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--prefix` | option | `\u` | Escape prefix to decode: `\u`, `%u`, or `U+`. |

**Simple example**

```bash
$ cchef unescape-unicode-characters -i 'H\u00E9llo'
Héllo
```

**Astral code point with the U+ prefix**

```bash
$ cchef unescape-unicode-characters --prefix 'U+' -i 'U+1F600'
😀
```

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
