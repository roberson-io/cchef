# Other

Operations that do not fit CyberChef's other categories: the disassemblers, the
random generators, identifiers and one-time passwords, and a few jokes.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Analyse UUID | `analyse-uuid` | [Universally unique identifier](https://wikipedia.org/wiki/Universally_unique_identifier) |
| Automated Validation Test Op | `automated-validation-test-op` | [Parameter validation](https://wikipedia.org/wiki/Data_validation) |
| Chi Square | `chi-square` | [Chi-squared distribution](https://wikipedia.org/wiki/Chi-squared_distribution) |
| Disassemble ARM | `disassemble-arm` | [ARM architecture family](https://wikipedia.org/wiki/ARM_architecture_family) |
| Disassemble x86 | `disassemble-x86` | [x86](https://wikipedia.org/wiki/X86) |
| Entropy | `entropy` | [Entropy (information theory)](https://wikipedia.org/wiki/Entropy_(information_theory)) |
| Frequency distribution | `frequency-distribution` | [Frequency distribution](https://wikipedia.org/wiki/Frequency_distribution) |
| Generate De Bruijn Sequence | `generate-de-bruijn-sequence` | [De Bruijn sequence](https://wikipedia.org/wiki/De_Bruijn_sequence) |
| Generate HOTP | `generate-hotp` | [HMAC-based One-time Password algorithm](https://wikipedia.org/wiki/HMAC-based_One-time_Password_algorithm) |
| Generate Lorem Ipsum | `generate-lorem-ipsum` | [Lorem ipsum](https://wikipedia.org/wiki/Lorem_ipsum) |
| Generate QR Code | `generate-qr-code` | [QR code](https://wikipedia.org/wiki/QR_code) |
| Generate TOTP | `generate-totp` | [Time-based One-time Password algorithm](https://wikipedia.org/wiki/Time-based_One-time_Password_algorithm) |
| Generate UUID | `generate-uuid` | [Universally unique identifier](https://wikipedia.org/wiki/Universally_unique_identifier) |
| Haversine distance | `haversine-distance` | [Haversine formula](https://wikipedia.org/wiki/Haversine_formula) |
| HTML To Text | `html-to-text` | [HTML](https://wikipedia.org/wiki/HTML) |
| Index of Coincidence | `index-of-coincidence` | [Index of coincidence](https://wikipedia.org/wiki/Index_of_coincidence) |
| Numberwang | `numberwang` | [That Mitchell and Webb Look](https://wikipedia.org/wiki/That_Mitchell_and_Webb_Look) |
| P-list Viewer | `p-list-viewer` | [Property list](https://wikipedia.org/wiki/Property_list) |
| Parse QR Code | `parse-qr-code` | [QR code](https://wikipedia.org/wiki/QR_code) |
| Pseudo-Random Integer Generator | `pseudo-random-integer-generator` | [Pseudorandom number generator](https://wikipedia.org/wiki/Pseudorandom_number_generator) |
| Pseudo-Random Number Generator | `pseudo-random-number-generator` | [Cryptographically secure PRNG](https://wikipedia.org/wiki/Cryptographically-secure_pseudorandom_number_generator) |
| XKCD Random Number | `xkcd-random-number` | [xkcd 221](https://xkcd.com/221/) |

## Analyse UUID

Reads a [UUID](https://wikipedia.org/wiki/Universally_unique_identifier) and
reports what it carries: the version, the 128-bit value as a decimal integer,
and — for the versions that embed one — the moment it was made.

Versions 1 and 6 hold a timestamp, a node identifier and a clock sequence.
Version 7 holds a timestamp and two stretches of randomness. Every other version
holds nothing to read back.

Only versions 1 to 8 with a standard variant digit are accepted, along with the
nil and max UUIDs the standard reserves. Surrounding whitespace is ignored and
either case is read.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--include-metadata` | boolean | `true` | Report the timestamp and the other embedded fields |

**Example**

```bash
cchef analyse-uuid -i "cefa1760-28ee-11f1-9f95-1fb76af3e239"
```

Output:

```
Version:
1

Timestamp:
1774514156502

Timestamp (ISO):
2026-03-26T08:35:56.502Z

Node:
1F:B7:6A:F3:E2:39

Clock:
8085

UUID Integer:
275119515460318071558429785403790975545
```

**Complex example**

A version 7 UUID, with the metadata left out:

```bash
cchef analyse-uuid --include-metadata=false -i "019d294a-af64-7728-9524-26da08f50708"
```

Output:

```
Version:
7

UUID Integer:
2145256098533991595556290452700595976
```

---

## Automated Validation Test Op

Does nothing but report `Success` once its arguments have been accepted. It
exists so the checks made on arguments can be exercised end to end: every kind
of limit an argument may carry is declared on it, so running it is really a test
of the checker.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--integer-number` | number | `5` | A whole number from 5 to 10 |
| `--real-number` | number | `1.5` | A number from 1.5 to 5.5, whole or not |
| `--non-empty-string` | string | `hello` | At most 5 characters, and not empty |
| `--empty-allowed-string` | string | *(empty)* | Anything, including nothing |
| `--non-empty-toggle-string` | toggleString | `test` (Option A) | Not empty; mode `Option A` or `Option B` |
| `--option-ingredient` | option | `Option 1` | `Option 1`, `Option 2` or `Option 3` |

**Example**

```bash
cchef automated-validation-test-op -i "test"
```

Output:

```
Success
```

**Complex example**

An argument outside its limits is reported rather than used:

```bash
cchef automated-validation-test-op -i "test" --integer-number 4
```

Output:

```
cchef: step 1 (Automated Validation Test Op): Integer Number must be greater than or equal to 5.
```

---

## Chi Square

Calculates the Chi Square distribution of the input's byte values: how far the
counts stray from an even spread across all 256 values. A low figure suggests
the bytes are evenly spread, as encrypted or compressed data tends to be.

This operation takes no options.

**Example**

```bash
cchef chi-square -i "Hello world!"
```

Output:

```
403.0885416666666
```

---

## Disassemble ARM

Disassembles ARM machine code into assembly language. The input is hexadecimal;
whitespace is ignored.

Output follows the formatting conventions of Capstone, the disassembly framework
CyberChef uses.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--architecture` | option | `ARM (32-bit)` | `ARM (32-bit)` or `ARM64 (AArch64)`. |
| `--mode` | option | `ARM` | `ARM`, `Thumb`, `Thumb + Cortex-M`, or `ARMv8`. Ignored for AArch64. |
| `--endianness` | option | `Little Endian` | `Little Endian` or `Big Endian`. |
| `--starting-address-hex` | number | `0` | Address the first instruction is taken to be at. |
| `--show-instruction-hex` | bool | `true` | Show the instruction bytes. |
| `--show-instruction-position` | bool | `true` | Show the instruction address. |

> Disassembly stops at the first instruction that cannot be decoded, as
> Capstone's linear sweep does. ARM, Thumb, Thumb-2 and AArch64 are all covered,
> including the floating-point unit and Advanced SIMD (`vadd`, `vld1`, `vmov`
> and the rest of NEON/VFP), the coprocessor, saturating, parallel-arithmetic
> and signed-multiply groups, and the exception-return and processor-state
> instructions.

**Simple example**

```bash
cchef disassemble-arm -i "00482de904b08de20000a0e10088bde8"
```

Output:

```
0x00000000  00482de9          push {fp, lr}
0x00000004  04b08de2          add fp, sp, #4
0x00000008  0000a0e1          mov r0, r0
0x0000000c  0088bde8          pop {fp, pc}
```

**Vector example**

A NEON sequence: load a vector, add and multiply it, store it back.

```bash
cchef disassemble-arm -i "8d0720f4400d00f2100d00f38f0701f4" \
  --show-instruction-hex=false
```

Output:

```
0x00000000  vld1.32 {d0}, [r0]!
0x00000004  vadd.f32 q0, q0, q0
0x00000008  vmul.f32 d0, d0, d0
0x0000000c  vst1.32 {d0}, [r1]
```

**Complex example**

An AArch64 function prologue and epilogue, disassembled from address `0x1000`
with the byte column hidden:

```bash
cchef disassemble-arm -i "fd7bbfa9fd03009100008052fd7bc1a8c0035fd6" \
  --architecture "ARM64 (AArch64)" --starting-address-hex 4096 \
  --show-instruction-hex=false
```

Output:

```
0x00001000  stp x29, x30, [sp, #-0x10]!
0x00001004  mov x29, sp
0x00001008  movz w0, #0
0x0000100c  ldp x29, x30, [sp], #0x10
0x00001010  ret
```

## Disassemble x86

Disassembles x86 machine code into assembly language, supporting 64-bit, 32-bit
and 16-bit code. The input is hexadecimal; whitespace is ignored.

This is a port of the X86-64 disassembler CyberChef vendors, so the output
matches CyberChef exactly — including its resolution of RIP-relative operands to
absolute addresses, and its own quirks (it predates `ENDBR64`, and prints `NaN`
where an instruction runs off the end of the input).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--bit-mode` | option | `64` | `64`, `32`, or `16`. |
| `--compatibility` | option | `Full x86 architecture` | `Full x86 architecture`, `Knights Corner`, `Larrabee`, `Cyrix`, `Geode`, `Centaur`, or `X86/486`. |
| `--code-segment-cs` | number | `16` | Code segment. Read as hexadecimal. |
| `--offset-ip` | number | `0` | Instruction pointer offset. Read as hexadecimal. |
| `--show-instruction-hex` | bool | `true` | Show the instruction bytes. |
| `--show-instruction-position` | bool | `true` | Show the instruction address. |

> The code segment and offset are read as **hexadecimal**, matching CyberChef:
> `--offset-ip 4096` starts at address `0x4096`, not `0x1000`.

**Simple example**

```bash
cchef disassemble-x86 -i "554889e54883ec20"
```

Output:

```
0000000000000000 55                              PUSH RBP
0000000000000001 4889E5                          MOV RBP,RSP
0000000000000004 4883EC20                        SUB RSP,0000000000000020
```

**Complex example**

Sixteen-bit code with a code segment, addresses shown as `segment:offset`:

```bash
cchef disassemble-x86 -i "b80100cd21" --bit-mode 16 --code-segment-cs 16
```

Output:

```
0016:0000 B80100                          MOV AX,0001
0016:0003 CD21                            INT 21
```

## Entropy

Measures how much information the input carries, and draws it five ways. Zero
means no randomness at all — every byte the same — and eight, the maximum, means
the bytes are spread evenly.

As a rule of thumb: ordinary English text falls between 3.5 and 5, and properly
encrypted or compressed data of any length should be over 7.5. A stretch of
unusually high entropy within a file often marks an encrypted or compressed
section, which the scanning views are for.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--visualisation` | option | `Shannon scale` | `Shannon scale` gives the figure for the whole input. `Histogram (Bar)` and `Histogram (Line)` chart how often each byte value occurs. `Curve` and `Image` chart the entropy of each block of the input in turn. |

The four chart views produce an SVG; write it to a file with `-o`. CyberChef
draws a scale bar beside the Shannon figure, which needs a browser to size and
render, so cchef gives the figure alone.

**Simple example**

```bash
cchef entropy -i "Hello world!"
```

Output:

```
Shannon entropy: 3.0220552088742005
```

**Complex example**

Charting how the entropy varies across a file, which shows where any compressed
or encrypted sections lie:

```bash
cchef entropy --visualisation Curve --in-file archive.bin -o entropy.svg
```

The file written is an SVG of the entropy of each block in turn, 500 by 500.
`--visualisation Image` draws the same readings as a grid of shaded cells
instead, from black at the lowest entropy to white at the highest.

---

## Frequency distribution

Reports how often each byte value occurs, as a table of percentages with a bar
beside each.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--show-0s` | bool | `true` | Include byte values that do not occur at all. |
| `--show-ascii` | bool | `true` | Show the character each byte stands for. The ones that do not print are shown as their control pictures. |

> CyberChef draws a bar chart into a canvas above the table, which needs a
> browser to size and render. The table carries the same figures.

**Example**

```bash
cchef frequency-distribution --show-0s=false -i "Hello"
```

Output (truncated):

```
Total data length: 5
Number of bytes represented: 4
Number of bytes not represented: 252

<table class="table table-hover table-sm">
    <tr><th>Byte</th><th>ASCII</th><th>Percentage</th><th></th></tr><tr><td>48</td><td>H</td><td>20%     </td><td>||||||||||||||||||||</td></tr>...
```

---

## Generate De Bruijn Sequence

Builds a [De Bruijn sequence](https://wikipedia.org/wiki/De_Bruijn_sequence):
read round as a loop, it contains every key of the given length over the given
alphabet exactly once, making it the shortest string that tries them all. The
input is ignored.

The alphabet is written with the digits `0` upwards, so it runs from 2 to 9
symbols. The sequence is as long as the number of keys it covers — the alphabet
size raised to the key length — and that is capped at 50,000.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--alphabet-size-k` | number | `2` | How many symbols, from 2 to 9 |
| `--key-length-n` | number | `3` | How long each key is, at least 2 |

**Example**

Every three-bit key, in eight bits:

```bash
cchef generate-de-bruijn-sequence --alphabet-size-k 2 --key-length-n 3
```

Output:

```
00010111
```

**Complex example**

Every two-symbol key over three symbols, in nine characters:

```bash
cchef generate-de-bruijn-sequence --alphabet-size-k 3 --key-length-n 2
```

Output:

```
001021122
```

---

## Generate HOTP

Works out an
[HMAC-based one-time password](https://wikipedia.org/wiki/HMAC-based_One-time_Password_algorithm)
(RFC 4226) from a shared secret and a counter, and reports the `otpauth://` key
URI alongside it so an authenticator app can be set up from the same output.

The input is the secret, written in base32 (characters A–Z and 2–7). Case and
spacing are ignored, and trailing padding is dropped. Leave the input empty for
a secret to be drawn at random.

The secret shown in the URI is the secret as it reads back, which is not always
what was typed: padding is gone, case is levelled, and a group at the end too
short to make a whole byte is dropped.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--name` | string | `Account` | The account name that appears in the URI |
| `--code-length` | number | `6` | Digits in the password, from 6 to 8 |
| `--counter` | number | `0` | The counter the password is worked out for |

**Example**

```bash
cchef generate-hotp -i "JBSWY3DPEHPK3PXP"
```

Output:

```
URI: otpauth://hotp/Account?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=6&counter=0

Password: 282760
```

**Complex example**

An eight-digit password at counter 42, under a name that needs escaping in the
URI:

```bash
cchef generate-hotp --name "user@example.com" --code-length 8 --counter 42 -i "JBSWY3DPEHPK3PXP"
```

Output:

```
URI: otpauth://hotp/user%40example.com?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=8&counter=42

Password: 79090604
```

---

## Generate Lorem Ipsum

Generates varying lengths of
[lorem ipsum](https://wikipedia.org/wiki/Lorem_ipsum) placeholder text. The
input is ignored.

Sentence and paragraph lengths are drawn about a mean, so no two runs give the
same text, and the passage always opens with `Lorem ipsum dolor sit amet`.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--length` | number | `3` | How much text to make |
| `--length-in` | option | `Paragraphs` | `Paragraphs`, `Sentences`, `Words` or `Bytes` |

Lengths up to 100,000 are accepted for paragraphs, sentences and words, and up
to 1,000,000 for bytes.

**Example**

```bash
cchef generate-lorem-ipsum --length 2 --length-in Sentences
```

Output:

```
Lorem ipsum dolor sit amet deserunt reprehenderit Lorem fugiat occaecat officia qui consectetur et Lorem ullamco quis ut aute ad quis qui nostrud tempor labore fugiat adipisicing pariatur anim non nostrud minim eu magna sunt dolore irure anim. Est ut ad, velit amet.
```

**Complex example**

Cut to an exact number of characters, which is useful for filling a fixed-width
field:

```bash
cchef generate-lorem-ipsum --length 30 --length-in Bytes
```

Output:

```
Lorem ipsum dolor sit amet fug
```

---

## Generate QR Code

Generates a Quick Response (QR) code from the input text, as a raster or vector
image.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--image-format` | option | `PNG` | `PNG`, `SVG`, `EPS`, or `PDF`. |
| `--module-size-px` | number | `5` | Pixels per module. Read by `PNG` and `SVG` only; `EPS` and `PDF` scale the page instead. |
| `--margin-num-modules` | number | `4` | Modules of quiet zone around the code. |
| `--error-correction` | option | `Medium` | `Low`, `Medium`, `Quartile`, or `High`. More correction withstands more damage but holds less data. |

The encoding mode is chosen from the input: digits alone use the numeric mode,
uppercase text and a few punctuation marks the alphanumeric one, and anything
else is encoded as bytes. The version is the smallest that holds the result.

**Simple example**

```bash
cchef generate-qr-code -i "Hello world!" -o hello.png
```

The file written is a 145 by 145 grayscale PNG.

**Complex example**

An SVG at four pixels a module with a two module quiet zone:

```bash
cchef generate-qr-code --image-format SVG --module-size-px 4 \
  --margin-num-modules 2 -i "https://github.com/roberson-io/cchef"
```

Output (truncated):

```
<svg xmlns="http://www.w3.org/2000/svg" width="132" height="132" viewBox="0 0 33 33"><path d="M2 2h7v7h-7zM11 2h5v1h1v-1h1v2h-2v1h-1v1h-3v1h1v2h1v-2h1v4h-1v1h-2v-1h1v-1h-1v1h-1v-1h-1v-6h2v-1...
```

Encapsulated PostScript at the highest error correction:

```bash
cchef generate-qr-code --image-format EPS --error-correction High -i "cchef"
```

Output (truncated):

```
%!PS-Adobe-3.0 EPSF-3.0
%%BoundingBox: 0 0 261 261
/h { 0 rlineto } bind def
...
```

---

## Generate TOTP

Works out a
[time-based one-time password](https://wikipedia.org/wiki/Time-based_One-time_Password_algorithm)
(RFC 6238) from a shared secret and the current time, and reports the
`otpauth://` key URI alongside it.

The secret is given and read exactly as for [Generate HOTP](#generate-hotp); the
counter is taken from the clock instead, as the number of whole intervals since
the Unix epoch.

> **Note:** the epoch offset (T0) is accepted but has no effect. This matches
> CyberChef, which hands the offset to a version of its library that has no such
> setting, and the setting is quietly dropped. The password moves with the clock,
> so the one below will differ each time it is run.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--name` | string | `Account` | The account name that appears in the URI |
| `--code-length` | number | `6` | Digits in the password, from 6 to 8 |
| `--epoch-offset-t0` | number | `0` | Accepted, but has no effect |
| `--interval-t1` | number | `30` | Seconds each password stands for |

**Example**

```bash
cchef generate-totp -i "JBSWY3DPEHPK3PXP"
```

Output:

```
URI: otpauth://totp/Account?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=6&period=30

Password: 687643
```

**Complex example**

An eight-digit password that changes every ninety seconds:

```bash
cchef generate-totp --name "user@example.com" --code-length 8 --interval-t1 90 -i "JBSWY3DPEHPK3PXP"
```

Output:

```
URI: otpauth://totp/user%40example.com?secret=JBSWY3DPEHPK3PXP&algorithm=SHA1&digits=8&period=90

Password: 36412949
```

---

## Generate UUID

Generates a [UUID](https://wikipedia.org/wiki/Universally_unique_identifier)
(RFC 9562, formerly RFC 4122), also known as a GUID.

Versions 3 and 5 hash the input under the namespace given, so the same pair
always produces the same UUID; version 3 hashes with MD5 and version 5 with
SHA-1. The remaining versions ignore the input: 1 and 6 are timestamp-based, 4
is random throughout, and 7 counts milliseconds since the Unix epoch and sorts
by the order it was made in.

> **Alternative to** [`uuidgen`](https://man7.org/linux/man-pages/man1/uuidgen.1.html). cchef offers UUID versions 1–5 (and NIL/max); `uuidgen` does only v1 and v4.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--version` | option | `v4` | One of `v1`, `v3`, `v4`, `v5`, `v6`, `v7` |
| `--namespace` | string | `1b671a64-40d5-491e-99b0-da01ff1f3341` | The namespace to hash under, for `v3` and `v5` only |

**Example**

```bash
cchef generate-uuid
```

Output:

```
5e67f9ac-2944-4edc-b225-f7fbb4056f75
```

**Complex example**

A version 3 UUID for a host name, under the standard DNS namespace. The answer
is the same every time:

```bash
cchef generate-uuid --version v3 --namespace "6ba7b810-9dad-11d1-80b4-00c04fd430c8" -i "www.example.com"
```

Output:

```
5df41881-3aed-3515-88a7-2f4a814cf09e
```

---

## Haversine distance

Returns the distance in meters between two points given as latitude and
longitude, using the
[haversine formula](https://wikipedia.org/wiki/Haversine_formula) on a sphere of
mean Earth radius.

The input is four numbers separated by commas — `lat1, lng1, lat2, lng2` — each
optionally followed by a single space. Nothing else is accepted, including space
at either end of the input.

This operation takes no options.

> The last digit or two may differ from CyberChef's. Go's cosine and the one a
> browser runs are different implementations that disagree by a unit in the last
> place for some arguments; the difference is around 1 part in 10^15, or a few
> nanometers across the whole globe.
>
> Two points that name the same place (a pole reached either way round, or the
> two spellings of the date line) come out as a few nanometers rather than
> nothing. The formula loses almost all its precision there, in any
> implementation.

**Example**

London to Washington DC:

```bash
cchef haversine-distance -i "51.487263,-0.124323, 38.9517,-77.1467"
```

Output:

```
5902542.836307819
```

---

## HTML To Text

Shows HTML as raw text rather than rendered markup.

In CyberChef this changes how a result is *displayed*: the interface renders a
value it has been told is HTML, and this operation hands the same value on as
plain text so the markup shows as it stands. Nothing renders anything at a
command line, so here the input is passed through unchanged.

This operation takes no options.

**Example**

```bash
cchef html-to-text -i "<b>bold</b> &amp; <i>italic</i>"
```

Output:

```
<b>bold</b> &amp; <i>italic</i>
```

---

## Index of Coincidence

The probability that two letters drawn from the text at random are the same.
Letters only: everything else is ignored, and case does not matter.

Zero means every letter is different; one means they are all the same. English
text generally falls between 0.067 and 0.078, so a much lower figure suggests
the text is random, compressed or encrypted. The normalized figure is the same
value measured in twenty-sixths, so English sits near 1.7 to 2.0.

This operation takes no options.

**Example**

```bash
cchef index-of-coincidence -i "Hello world, this is a test to determine the correct IC value."
```

Output:

```
Index of Coincidence: 0.07142857142857142
Normalized: 1.857142857142857
```

> CyberChef draws a scale bar below these figures, which needs a browser to size
> and render.

---

## Numberwang

Tells you whether the input is Numberwang, and offers a fact about Numberwang.
A number anywhere in the input counts; a number with a letter after it is
AlphaNumericWang instead. `f0rty-s1x`, `shinty-six` and
`filth-hundred and neeb` count as numbers in their own right.

This operation takes no options.

**Example**

```bash
cchef numberwang -i "42"
```

Output:

```
42! That's Numberwang!

Did you know: Astronomers suspect God is Numberwang.
```

The fact is picked at random, so it differs each time.

---

## P-list Viewer

Lays a [property list](https://wikipedia.org/wiki/Property_list) out in a form
that can be read: braces round a dictionary, brackets round an array, one entry
to a line, and each name or array position joined to its value by `=>`.

Property lists are how macOS, iOS, NeXTSTEP and GNUstep store serialized
objects, usually in files ending `.plist`.

Input that is not a well-formed property list is reported as an error.

This operation takes no options.

**Example**

```bash
cchef p-list-viewer --in-file com.example.agent.plist
```

Output:

```
plist => {
	Label => "com.example.agent"
	RunAtLoad => true
	ProgramArguments => [
		0 => "/usr/bin/say"
		1 => "hello world"
	]
}
/plist
```

---

## Parse QR Code

Reads an image and returns the text of any QR code in it. The image may be a
PNG, JPEG, GIF, BMP or WebP.

The reader thresholds the image over local regions, finds the three finder
patterns and the alignment pattern,
samples the modules through the perspective those four points define, and
corrects the result with Reed-Solomon. A code photographed at an angle reads
correctly, as does one mirrored or damaged within its correction level.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--normalise-image` | bool | `false` | Convert to grayscale and stretch the contrast before reading, which helps with a faint or unevenly lit photograph. |

> Codes in the kanji mode are refused rather than read, since the reader does
> not carry the character table they need. Every other mode is supported.

**Simple example**

```bash
cchef parse-qr-code --in-file hello.png
```

Output:

```
Hello world!
```

**Complex example**

Generating a code and reading it back, with the image normalized first:

```bash
cchef generate-qr-code --error-correction High -i "cchef round trip" -o code.png
cchef parse-qr-code --normalise-image --in-file code.png
```

Output:

```
cchef round trip
```

---

## Pseudo-Random Integer Generator

Draws random integers from a range, using the system's cryptographic source.
The input is ignored.

Values are drawn by rejection sampling, so every integer in the range is equally
likely — no value is favored by the remainder that simple folding would leave.

Bounds may run from `-(2^53 - 1)` to `2^53 - 1`, and the span between them may
be as wide as `2^53 - 1`. A bound that is not a whole number is drawn inwards,
so `0.5` to `3.5` gives integers from 1 to 3.

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `--number-of-integers` | number | `1` | How many to draw |
| `--min-value` | number | `0` | The lowest value that may be drawn |
| `--max-value` | number | `99` | The highest value that may be drawn |
| `--delimiter` | option | `Space` | `Space`, `Comma`, `Semi-colon`, `Colon`, `Line feed` or `CRLF` |
| `--output-format` | option | `Raw` | `Raw`, `Hex` or `Decimal` |

> `Raw` writes each integer as the character it stands for and runs them
> together, so the delimiter does not apply. `Hex` is not zero-padded, so a
> value below 16 comes out as a single digit.

**Example**

Five rolls of a die:

```bash
cchef pseudo-random-integer-generator --number-of-integers 5 \
  --min-value 1 --max-value 6 --output-format Decimal
```

Output:

```
1 3 3 1 4
```

**Complex example**

Eight random bytes as comma-separated hexadecimal:

```bash
cchef pseudo-random-integer-generator --number-of-integers 8 \
  --min-value 0 --max-value 255 --output-format Hex --delimiter Comma
```

Output:

```
f4,4c,ba,91,d9,e7,62,25
```

---

## Pseudo-Random Number Generator

Generates a number of cryptographically-secure random bytes (using Go's
`crypto/rand`) and outputs them in the chosen representation.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--number-of-bytes` | number | `32` | How many random bytes to generate. |
| `--output-as` | option | `Hex` | `Hex`, `Integer`, `Byte array`, or `Raw`. |

> Output is non-deterministic by design.

**Simple example**

```bash
cchef pseudo-random-number-generator --number-of-bytes 4 --output-as Hex
```

Output:

```
1ed9ec81
```

---

## XKCD Random Number

Returns a random number, as specified by [xkcd 221](https://xkcd.com/221/):

> `return 4;  // chosen by fair dice roll. guaranteed to be random.`

The input is ignored, and the answer is always `4`.

This operation takes no options.

**Example**

```bash
cchef xkcd-random-number -i ""
```

Output:

```
4
```
