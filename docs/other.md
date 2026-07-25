# Other

Operations that do not fit CyberChef's other categories: the disassemblers and
the pseudo-random generator.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Disassemble ARM | `disassemble-arm` | [ARM architecture family](https://wikipedia.org/wiki/ARM_architecture_family) |
| Disassemble x86 | `disassemble-x86` | [x86](https://wikipedia.org/wiki/X86) |
| Pseudo-Random Number Generator | `pseudo-random-number-generator` | [Cryptographically secure PRNG](https://wikipedia.org/wiki/Cryptographically-secure_pseudorandom_number_generator) |

## Disassemble ARM

Disassembles ARM machine code into assembly language. The input is hexadecimal;
whitespace is ignored.

CyberChef runs the Capstone disassembly framework compiled to WebAssembly.
cchef decodes ARM and AArch64 with `golang.org/x/arch` (pure Go) and decodes
Thumb from scratch, formatting all three to Capstone's conventions so the output
matches.

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
> Capstone's linear sweep does. The coprocessor, system-register and NEON/VFP
> encodings are not decoded, and Capstone recognises a few ARM32 forms
> `golang.org/x/arch` declines; on ordinary compiler-generated code the output
> matches Capstone exactly.

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
