# Other

Operations that do not fit CyberChef's other categories: the disassemblers and
the pseudo-random generator.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Disassemble ARM | `disassemble-arm` | [ARM architecture family](https://wikipedia.org/wiki/ARM_architecture_family) |
| Disassemble x86 | `disassemble-x86` | [x86](https://wikipedia.org/wiki/X86) |
| Generate QR Code | `generate-qr-code` | [QR code](https://wikipedia.org/wiki/QR_code) |
| Parse QR Code | `parse-qr-code` | [QR code](https://wikipedia.org/wiki/QR_code) |
| Pseudo-Random Number Generator | `pseudo-random-number-generator` | [Cryptographically secure PRNG](https://wikipedia.org/wiki/Cryptographically-secure_pseudorandom_number_generator) |

## Disassemble ARM

Disassembles ARM machine code into assembly language. The input is hexadecimal;
whitespace is ignored.

CyberChef runs the Capstone disassembly framework compiled to WebAssembly.
cchef decodes ARM and AArch64 with `golang.org/x/arch` (pure Go) and decodes
Thumb, the floating-point unit and Advanced SIMD from scratch, formatting them
all to Capstone's conventions so the output matches.

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

## Generate QR Code

Generates a Quick Response (QR) code from the input text, as a raster or vector
image.

The QR matrix, the four renderers and the PNG's compression are all ported from
the `qr-image` package CyberChef wraps, so every format is byte-for-byte what
CyberChef produces.

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

The file written is a 145 by 145 greyscale PNG.

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

## Parse QR Code

Reads an image and returns the text of any QR code in it. The image may be a
PNG, JPEG, GIF, BMP or WebP.

The reader is ported from `jsQR`, which CyberChef uses: it thresholds the image
over local regions, finds the three finder patterns and the alignment pattern,
samples the modules through the perspective those four points define, and
corrects the result with Reed-Solomon. A code photographed at an angle reads
correctly, as does one mirrored or damaged within its correction level.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--normalise-image` | bool | `false` | Convert to greyscale and stretch the contrast before reading, which helps with a faint or unevenly lit photograph. |

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

Generating a code and reading it back, with the image normalised first:

```bash
cchef generate-qr-code --error-correction High -i "cchef round trip" -o code.png
cchef parse-qr-code --normalise-image --in-file code.png
```

Output:

```
cchef round trip
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
