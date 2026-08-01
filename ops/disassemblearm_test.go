package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/arch/arm/armasm"

	"github.com/roberson-io/cchef/core"
)

const (
	arm32 = "ARM (32-bit)"
	arm64 = "ARM64 (AArch64)"
)

// armRecipe builds a Disassemble ARM recipe.
func armRecipe(arch, mode, endian string, address float64, showHex, showPos bool) core.Recipe {
	return core.Recipe{{
		Op:   "Disassemble ARM",
		Args: []any{arch, mode, endian, address, showHex, showPos},
	}}
}

// armDefault is the recipe with CyberChef's default arguments for an
// architecture.
func armDefault(arch string) core.Recipe {
	return armRecipe(arch, "ARM", "Little Endian", 0, true, true)
}

// armFixture is one of CyberChef's own test cases, which assert a regular
// expression against the output rather than an exact string.
type armFixture struct {
	name   string
	input  string
	match  string
	recipe core.Recipe
}

// runARMFixtures checks each transcribed upstream fixture.
func runARMFixtures(t *testing.T, cases []armFixture) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := c.recipe.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			re := regexp.MustCompile(c.match)
			if !re.MatchString(out.String()) {
				t.Fatalf("output %q does not match %s", out.String(), c.match)
			}
		})
	}
}

// The fixtures transcribed from ../CyberChef/tests/operations/tests/DisassembleARM.mjs.
func TestDisassembleARMFixtures(t *testing.T) {
	a32 := armDefault(arm32)
	a64 := armDefault(arm64)
	thumb := armRecipe(arm32, "Thumb", "Little Endian", 0, true, true)

	runARMFixtures(t, []armFixture{
		{"ARM32 NOP (mov r0, r0)", "00 00 a0 e1", `mov\s+r0,\s*r0`, a32},
		{"ARM32 bx lr", "1e ff 2f e1", `bx\s+lr`, a32},
		{"ARM32 push {fp, lr}", "00 48 2d e9", `push\s+\{fp,\s*lr\}`, a32},
		{"ARM32 add fp, sp, #4", "04 b0 8d e2", `add\s+fp,\s*sp`, a32},
		{"ARM32 ldr r0, [r1]", "00 00 91 e5", `ldr\s+r0,\s*\[r1\]`, a32},
		{"ARM32 str r0, [r1]", "00 00 81 e5", `str\s+r0,\s*\[r1\]`, a32},
		{"ARM32 bl (branch link)", "00 00 00 eb", `bl\s+`, a32},
		{"ARM32 mul r0, r1, r2", "91 02 00 e0", `mul\s+r0,\s*r1,\s*r2`, a32},
		{"Thumb mov r0, r0", "00 46", `mov\s+r0,\s*r0`, thumb},
		{"Thumb bx lr", "70 47", `bx\s+lr`, thumb},
		{"Thumb push {r4, lr}", "10 b5", `push\s+\{r4,\s*lr\}`, thumb},
		{"Thumb pop {r4, pc}", "10 bd", `pop\s+\{r4,\s*pc\}`, thumb},
		{"ARM64 ret", "c0 03 5f d6", `ret`, a64},
		{"ARM64 mov x0, #0", "00 00 80 d2", `mov[z]?\s+x0,\s*#0`, a64},
		{"ARM64 stp x29, x30, [sp, #-16]!", "fd 7b bf a9", `stp\s+x29,\s*x30,\s*\[sp`, a64},
		{"ARM64 ldp x29, x30, [sp], #16", "fd 7b c1 a8", `ldp\s+x29,\s*x30,\s*\[sp\]`, a64},
		{"ARM64 add x0, x1, x2", "20 00 02 8b", `add\s+x0,\s*x1,\s*x2`, a64},
		{"ARM64 sub x0, x1, x2", "20 00 02 cb", `sub\s+x0,\s*x1,\s*x2`, a64},
		{"ARM64 mul x0, x1, x2", "20 7c 02 9b", `mul\s+x0,\s*x1,\s*x2`, a64},
		{"ARM64 ldr x0, [x1]", "20 00 40 f9", `ldr\s+x0,\s*\[x1\]`, a64},
		{"ARM64 str x0, [x1]", "20 00 00 f9", `str\s+x0,\s*\[x1\]`, a64},
		{"ARM64 bl (branch link)", "00 00 00 94", `bl\s+`, a64},
		{"ARM64 cbz x0", "00 00 00 b4", `cbz\s+x0`, a64},
		{"ARM64 cbnz x0", "00 00 00 b5", `cbnz\s+x0`, a64},
		{"ARM64 sub sp, sp, #0x20", "ff 83 00 d1", `sub\s+sp,\s*sp`, a64},
		{"ARM64 add sp, sp, #0x20", "ff 83 00 91", `add\s+sp,\s*sp`, a64},
		{
			"ARM32 multiple instructions",
			"00 48 2d e9 04 b0 8d e2 00 00 a0 e1 00 88 bd e8",
			`(?s)push.*\n.*add.*\n.*mov.*\n.*pop`, a32,
		},
		{
			"ARM64 function prologue/epilogue",
			"fd 7b bf a9 fd 03 00 91 00 00 80 52 fd 7b c1 a8 c0 03 5f d6",
			`(?s)stp.*\n.*mov.*\n.*mov.*\n.*ldp.*\n.*ret`, a64,
		},
		{
			"ARM64 with start address 0x1000", "c0 03 5f d6", `0x00001000`,
			armRecipe(arm64, "ARM", "Little Endian", 4096, true, true),
		},
		{
			"ARM32 with start address 0x8000", "00 00 a0 e1", `0x00008000`,
			armRecipe(arm32, "ARM", "Little Endian", 32768, true, true),
		},
		{
			"ARM32 Big Endian", "e1 a0 00 00", `mov\s+r0,\s*r0`,
			armRecipe(arm32, "ARM", "Big Endian", 0, true, true),
		},
	})
}

// The upstream fixture for empty input asserts an exact empty result.
func TestDisassembleARMEmptyInput(t *testing.T) {
	runCases(t, []opCase{
		{"empty input", "", "", armDefault(arm64)},
	})
}

// The exact output, captured from the oracle running Capstone. The layout is a
// 0x-prefixed eight-digit address, the instruction bytes padded to sixteen
// columns, then the mnemonic and operands.
func TestDisassembleARMExact(t *testing.T) {
	runCases(t, []opCase{
		{
			"ARM32 nop", "0000a0e1",
			"0x00000000  0000a0e1          mov r0, r0",
			armDefault(arm32),
		},
		{
			"ARM32 pre-indexed store with writeback", "04b02de5",
			"0x00000000  04b02de5          str fp, [sp, #-4]!",
			armDefault(arm32),
		},
		{
			"ARM32 flag-setting subtract", "010050e0",
			"0x00000000  010050e0          subs r0, r0, r1",
			armDefault(arm32),
		},
		{
			"ARM32 branch with link resolves its target", "0f0000eb",
			"0x00000000  0f0000eb          bl #0x44",
			armDefault(arm32),
		},
		{
			"ARM32 several instructions", "00482de904b08de20000a0e10088bde8",
			"0x00000000  00482de9          push {fp, lr}\n" +
				"0x00000004  04b08de2          add fp, sp, #4\n" +
				"0x00000008  0000a0e1          mov r0, r0\n" +
				"0x0000000c  0088bde8          pop {fp, pc}",
			armDefault(arm32),
		},
		{
			"Thumb instructions are two bytes wide", "10b510bd",
			"0x00000000  10b5              push {r4, lr}\n" +
				"0x00000002  10bd              pop {r4, pc}",
			armRecipe(arm32, "Thumb", "Little Endian", 0, true, true),
		},
		{
			"ARM64 stack frame with a negative offset", "fd7bbfa9",
			"0x00000000  fd7bbfa9          stp x29, x30, [sp, #-0x10]!",
			armDefault(arm64),
		},
		{
			"ARM64 post-indexed load", "fd7bc1a8",
			"0x00000000  fd7bc1a8          ldp x29, x30, [sp], #0x10",
			armDefault(arm64),
		},
		{
			"ARM64 keeps Capstone's movz alias", "000080d2",
			"0x00000000  000080d2          movz x0, #0",
			armDefault(arm64),
		},
		{
			"ARM64 extended register operand", "20003f8b",
			"0x00000000  20003f8b          add x0, x1, wzr, uxtb",
			armDefault(arm64),
		},
		{
			"ARM64 shift immediate", "00fc7fd3",
			"0x00000000  00fc7fd3          lsr x0, x0, #0x3f",
			armDefault(arm64),
		},
		{
			"ARM64 compare and branch", "000000b4",
			"0x00000000  000000b4          cbz x0, #0",
			armDefault(arm64),
		},
		{
			"ARM64 prologue and epilogue", "fd7bbfa9fd03009100008052fd7bc1a8c0035fd6",
			"0x00000000  fd7bbfa9          stp x29, x30, [sp, #-0x10]!\n" +
				"0x00000004  fd030091          mov x29, sp\n" +
				"0x00000008  00008052          movz w0, #0\n" +
				"0x0000000c  fd7bc1a8          ldp x29, x30, [sp], #0x10\n" +
				"0x00000010  c0035fd6          ret",
			armDefault(arm64),
		},
		{
			"big-endian ARM64", "d65f03c0",
			"0x00000000  d65f03c0          ret",
			armRecipe(arm64, "ARM", "Big Endian", 0, true, true),
		},
		{
			"a trailing partial word is ignored", "c0035fd6ff",
			"0x00000000  c0035fd6          ret",
			armDefault(arm64),
		},
		{
			"whitespace is stripped", "c0 03 5f d6",
			"0x00000000  c0035fd6          ret",
			armDefault(arm64),
		},
	})
}

// The two display toggles are independent.
func TestDisassembleARMDisplayToggles(t *testing.T) {
	runCases(t, []opCase{
		{
			"address only", "0000a0e1", "0x00000000  mov r0, r0",
			armRecipe(arm32, "ARM", "Little Endian", 0, false, true),
		},
		{
			"bytes only", "0000a0e1", "0000a0e1          mov r0, r0",
			armRecipe(arm32, "ARM", "Little Endian", 0, true, false),
		},
		{
			"neither", "0000a0e1", "mov r0, r0",
			armRecipe(arm32, "ARM", "Little Endian", 0, false, false),
		},
	})
}

// The ARM32 mode option selects between the ARM and Thumb instruction sets;
// Cortex-M and ARMv8 are refinements of those, and ARM64 ignores the setting.
func TestDisassembleARMModes(t *testing.T) {
	runCases(t, []opCase{
		{
			"Thumb + Cortex-M", "10b5", "0x00000000  10b5              push {r4, lr}",
			armRecipe(arm32, "Thumb + Cortex-M", "Little Endian", 0, true, true),
		},
		{
			"ARMv8", "0000a0e1", "0x00000000  0000a0e1          mov r0, r0",
			armRecipe(arm32, "ARMv8", "Little Endian", 0, true, true),
		},
		{
			"ARM64 ignores the mode option", "c0035fd6", "0x00000000  c0035fd6          ret",
			armRecipe(arm64, "Thumb", "Little Endian", 0, true, true),
		},
	})
}

// Bad input is reported with CyberChef's own wording.
func TestDisassembleARMErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  string
		recipe core.Recipe
		want   string
	}{
		{
			"non-hex characters", "zz", armDefault(arm64),
			"Invalid hexadecimal input. Please provide valid hex characters only.",
		},
		{
			"odd number of digits", "c0035fd", armDefault(arm64),
			"Invalid hexadecimal input. Length must be even.",
		},
		{
			"nothing decodable in ARM64", "ffffffff", armDefault(arm64),
			"No valid ARM64 (AArch64) instructions found in input. " +
				"The bytes may be for a different architecture or mode.",
		},
		{
			"nothing decodable in ARM32", "ffffffff", armDefault(arm32),
			"No valid ARM (32-bit) instructions found in input. " +
				"The bytes may be for a different architecture or mode.",
		},
		{
			"too few bytes for one instruction", "c0", armDefault(arm64),
			"No valid ARM64 (AArch64) instructions found in input. " +
				"The bytes may be for a different architecture or mode.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.recipe.Execute(core.NewDish([]byte(tc.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

// Capstone's two printers disagree: the ARM one uses decimal up to nine, while
// the AArch64 one is hexadecimal for everything but zero. Both keep the sign
// outside the 0x prefix.
func TestARMFormatImmediate(t *testing.T) {
	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "#0"},
		{1, "#1"},
		{9, "#9"},
		{10, "#0xa"},
		{16, "#0x10"},
		{63, "#0x3f"},
		{255, "#0xff"},
		{-1, "#-1"},
		{-9, "#-9"},
		{-10, "#-0xa"},
		{-16, "#-0x10"},
	} {
		if got := armImmediate32(tc.in); got != tc.want {
			t.Errorf("armImmediate32(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}

	for _, tc := range []struct {
		in   int64
		want string
	}{
		{0, "#0"},
		{1, "#0x1"},
		{2, "#0x2"},
		{9, "#0x9"},
		{16, "#0x10"},
		{63, "#0x3f"},
		{-16, "#-0x10"},
	} {
		if got := armImmediate64Wide(tc.in); got != tc.want {
			t.Errorf("armImmediate64Wide(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDisassembleARMGolden replays a corpus of real-world instructions captured
// from the oracle, covering ARM, Thumb and AArch64 in both endiannesses.
func TestDisassembleARMGolden(t *testing.T) {
	f, err := os.Open("testdata/disassemble_arm.jsonl")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer func() { _ = f.Close() }()

	// A case with no want is one the oracle refuses to decode, which is as much
	// a part of matching it as the text of the instructions it accepts.
	type goldenCase struct {
		Hex    string `json:"hex"`
		Arch   string `json:"arch"`
		Mode   string `json:"mode"`
		Endian string `json:"endian"`
		Want   string `json:"want"`
		Reject bool   `json:"reject"`
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	n := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var c goldenCase
		if err := json.Unmarshal(line, &c); err != nil {
			t.Fatalf("corpus line %d: %v", n+1, err)
		}
		n++
		if c.Endian == "" {
			c.Endian = "Little Endian"
		}

		recipe := armRecipe(c.Arch, c.Mode, c.Endian, 0, true, true)
		out, err := recipe.Execute(core.NewDish([]byte(c.Hex), core.TypeString))
		if c.Reject {
			if err == nil {
				t.Errorf("%s (%s, %s): decoded to %q, want no decode",
					c.Hex, c.Arch, c.Mode, out.String())
			}
			continue
		}
		if err != nil {
			t.Errorf("%s (%s, %s): %v", c.Hex, c.Arch, c.Mode, err)
			continue
		}
		if out.String() != c.Want {
			t.Errorf("%s (%s, %s):\n got %q\nwant %q", c.Hex, c.Arch, c.Mode, out.String(), c.Want)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	if n == 0 {
		t.Fatal("corpus is empty")
	}
}

// The Thumb decoder's guards are reached by malformed or truncated input, and
// by encodings the manual leaves unallocated.
func TestThumbDecoderGuards(t *testing.T) {
	decode := func(hexBytes string) (string, int, bool) {
		t.Helper()
		code, err := armDecodeHex(hexBytes)
		if err != nil {
			t.Fatalf("armDecodeHex(%q): %v", hexBytes, err)
		}
		return armDecodeThumbAt(code, 0, false, 0)
	}

	t.Run("truncated input", func(t *testing.T) {
		if _, _, ok := armDecodeThumbAt(nil, 0, false, 0); ok {
			t.Error("an empty buffer should not decode")
		}
		if _, _, ok := decode("00"); ok {
			t.Error("a single byte should not decode")
		}
		// A 32-bit prefix whose second halfword is missing.
		if _, _, ok := decode("00f0"); ok {
			t.Error("a truncated 32-bit encoding should not decode")
		}
	})

	t.Run("unallocated encodings", func(t *testing.T) {
		for _, c := range []struct{ name, hexBytes string }{
			{"reserved byte-reversal slot", "8aba"},
			{"unallocated miscellaneous encoding", "00b6"},
			{"unhandled 16-bit encoding", "0aef"},
			{"unhandled 32-bit opcode map", "00e80000"},
			{"conditional wide branch with a reserved condition", "80f30080"},
		} {
			if text, _, ok := decode(c.hexBytes); ok {
				t.Errorf("%s (%s) decoded to %q, want no decode", c.name, c.hexBytes, text)
			}
		}
	})

	t.Run("IT block masks", func(t *testing.T) {
		// The mask spells out the T/E pattern relative to the first condition.
		for _, c := range []struct {
			firstCond, mask uint32
			want            string
		}{
			{0, 0b1000, "it eq"},
			{0, 0b0100, "itt eq"},
			{0, 0b1100, "ite eq"},
			{0, 0b0001, "itttt eq"},
			{1, 0b0100, "ite ne"},
		} {
			if got := armThumbIT(c.firstCond, c.mask); got != c.want {
				t.Errorf("armThumbIT(%d, %04b) = %q, want %q", c.firstCond, c.mask, got, c.want)
			}
		}
	})
}

// The immediate scanner and the push rewrite have guards that malformed input
// reaches but the decoders never produce.
func TestARMFormattingGuards(t *testing.T) {
	t.Run("armParseImmediate rejects what is not a number", func(t *testing.T) {
		for _, in := range []string{"#", "#-", "#0x", "#zz", "#0xffffffffffffffffff"} {
			if _, _, ok := armParseImmediate(in); ok {
				t.Errorf("armParseImmediate(%q) reported success", in)
			}
		}
		v, width, ok := armParseImmediate("#-0x10, r0")
		if !ok || v != -16 || width != 6 {
			t.Errorf("armParseImmediate = %v,%v,%v; want -16,6,true", v, width, ok)
		}
	})

	t.Run("armRewriteOperands keeps a stray hash", func(t *testing.T) {
		got := armRewriteOperands("r0, #, r1", armImmediate32, armImmediate32, armImmediate32)
		if got != "r0, #, r1" {
			t.Errorf("got %q, want the text unchanged", got)
		}
	})

	t.Run("armFixBareImmediate leaves other operands alone", func(t *testing.T) {
		if got := armFixBareImmediate("r0, r1", armImmediate32); got != "r0, r1" {
			t.Errorf("got %q, want the operands unchanged", got)
		}
		// Wider than an int64, so the parse fails and the text is kept.
		wide := "0xffffffffffffffffff"
		if got := armFixBareImmediate(wide, armImmediate32); got != wide {
			t.Errorf("got %q, want %q", got, wide)
		}
		if got := armFixBareImmediate("0x1e", armImmediate32); got != "#0x1e" {
			t.Errorf("got %q, want %q", got, "#0x1e")
		}
	})

	t.Run("armFixPushPop32 only rewrites a single register", func(t *testing.T) {
		// e52db004: str fp, [sp, #-4]! -- the encoding x/arch prints as push.
		word := []byte{0x04, 0xb0, 0x2d, 0xe5}
		if m, o := armFixPushPop32(word, "push", "{fp}"); m != "str" || o != "fp, [sp, #-4]!" {
			t.Errorf("got %q %q", m, o)
		}
		// A real block transfer keeps its push spelling.
		block := []byte{0x00, 0x48, 0x2d, 0xe9}
		if m, _ := armFixPushPop32(block, "push", "{fp, lr}"); m != "push" {
			t.Errorf("block transfer rewritten to %q", m)
		}
		// A multi-register list in the load/store encoding is left alone.
		if m, _ := armFixPushPop32(word, "push", "{fp, lr}"); m != "push" {
			t.Errorf("multi-register list rewritten to %q", m)
		}
		if m, _ := armFixPushPop32(word, "push", ""); m != "push" {
			t.Errorf("empty list rewritten to %q", m)
		}
	})
}

// Every Thumb-2 group declines the encodings the manual leaves undefined or
// unpredictable. Those refusals are driven directly, since a stream that
// contained one would simply stop at it.
func TestThumb32Rejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decode func() (string, bool)
	}{
		{"an opcode map below the 32-bit escapes", func() (string, bool) {
			return armDecodeThumb32(0x0000, 0x0000, 0)
		}},
		{"an unallocated data-processing opcode", func() (string, bool) {
			return armThumb32DataImmediate(0xF0A0, 0x0004) // op 0b0101
		}},
		{"a block transfer with no index direction", func() (string, bool) {
			return armThumb32BlockTransfer(0xE810, 0x0003) // bits 8:7 clear
		}},
		{"a block transfer indexed both ways", func() (string, bool) {
			return armThumb32BlockTransfer(0xE990, 0x0003) // bits 8:7 set
		}},
		{"an unallocated shifted-register opcode", func() (string, bool) {
			return armThumb32DataShiftedRegister(0xEAA0, 0x0405)
		}},
		{"an unallocated plain-immediate opcode", func() (string, bool) {
			return armThumb32PlainImmediate(0xF220, 0x0004)
		}},
		{"a bitfield with a bit set the encoding fixes", func() (string, bool) {
			return armThumb32BitfieldInsert(0xF362, 0x3064)
		}},
		{"an unallocated third-group encoding", func() (string, bool) {
			return armThumb32LoadStoreSingle(0xF870, 0x0000) // op2 0b0000111
		}},
		{"a memory access wider than a word", func() (string, bool) {
			return armThumb32Memory(0xF870, 0x3000) // size 0b11
		}},
		{"a signed store", func() (string, bool) {
			return armThumb32Memory(0xF902, 0x3000)
		}},
		{"a signed word load", func() (string, bool) {
			return armThumb32Memory(0xF952, 0x3000)
		}},
		{"a register offset with bits set above the shift", func() (string, bool) {
			return armThumb32Memory(0xF852, 0x0403)
		}},
		{"a word store relative to the program counter", func() (string, bool) {
			return armThumb32Memory(0xF84F, 0x3D08)
		}},
		{"a halfword store relative to the program counter", func() (string, bool) {
			return armThumb32Memory(0xF82F, 0x3D08)
		}},
		{"an unallocated register data operation", func() (string, bool) {
			return armThumb32DataRegister(0xFA02, 0x3021)
		}},
		{"an unallocated bit-counting operation", func() (string, bool) {
			return armThumb32Miscellaneous(0xFAB2, 0xF3F2)
		}},
		{"an extend without its marker nibble", func() (string, bool) {
			return armThumb32Miscellaneous(0xFA02, 0x3081)
		}},
		{"an unallocated extend slot", func() (string, bool) {
			return armThumb32Miscellaneous(0xFA60, 0xF381)
		}},
		{"an unallocated long multiply", func() (string, bool) {
			return armThumb32LongMultiply(0xFBD1, 0x2304)
		}},
		{"a dual transfer that neither indexes nor writes back", func() (string, bool) {
			return armThumb32DualTransfer(0xE851, 0, 1, 3, 4)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if text, ok := tc.decode(); ok {
				t.Errorf("decoded to %q, want no decode", text)
			}
		})
	}
}

// A store may not name the program counter, and a transfer needs at least two
// registers to be worth the wide encoding.
func TestThumb32ListValidity(t *testing.T) {
	const pc = 1 << 15
	if armThumb32ListIsValid(pc|1, false) {
		t.Error("a store naming the program counter should be rejected")
	}
	if !armThumb32ListIsValid(pc|1, true) {
		t.Error("a load naming the program counter is allowed")
	}
	if armThumb32ListIsValid(1, true) {
		t.Error("a single-register list should be rejected")
	}
	if !armThumb32ListIsValid(0b11, true) {
		t.Error("a two-register list is allowed")
	}
}

// The coprocessor, multiply and move-wide groups each refuse encodings that are
// reserved, unpredictable, or another unit's territory.
func TestARMGroupRejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decode func() (string, bool)
	}{
		{"a linking branch with the low bits of its encoding set", func() (string, bool) {
			return armThumbSpecialData(0x4785) // blx r0, bits 2 to 0 set
		}},
		{"a wide linking branch to an unaligned target", func() (string, bool) {
			return armThumbLongBranch(0xF000, 0xE001, 0)
		}},
		{"a post-indexed access with no writeback", func() (string, bool) {
			return armThumb32IndexedMemory("ldr", 2, 3, 0x0800)
		}},
		{"a multiply with its reserved bits set", func() (string, bool) {
			return armThumb32Multiply(0xFB12, 0xF3C5)
		}},
		{"an unallocated multiply form", func() (string, bool) {
			return armThumb32Multiply(0xFB72, 0xF315) // usad8 has one variant only
		}},
		{"a plain multiply with a reserved variant", func() (string, bool) {
			return armThumb32PlainMultiply(2, 1, 15, 3, 4)
		}},
		{"a plain multiply that subtracts without an accumulator", func() (string, bool) {
			return armThumb32PlainMultiply(1, 1, 15, 3, 4)
		}},
		{"an unallocated unconditional floating-point encoding in Thumb", func() (string, bool) {
			return armThumb32Coprocessor(0xFE10, 0x0A10)
		}},
		{"an unallocated Advanced SIMD encoding in Thumb", func() (string, bool) {
			return armThumb32Coprocessor(0xEF10, 0x0C10)
		}},
		{"an extension register transfer indexed both ways", func() (string, bool) {
			return armCoprocessor32(0xEDB10B04) // p11, pre-indexed and incrementing
		}},
		{"a supervisor call in the ARM coprocessor space", func() (string, bool) {
			return armCoprocessor32(0xEF000000)
		}},
		{"an ARM encoding outside the coprocessor space", func() (string, bool) {
			return armCoprocessor32(0xE0000000)
		}},
		{"a reserved move-wide opcode", func() (string, bool) {
			return armMoveWide64(0x32800000) // opcode bits 01
		}},
		{"a word that is not a move-wide", func() (string, bool) {
			return armMoveWide64(0xD2000000)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if text, ok := tc.decode(); ok {
				t.Errorf("decoded to %q, want no decode", text)
			}
		})
	}
}

// A handful of encodings exercise the remaining corners: the reserved IT
// condition, the zero register, a shifted move-wide, and a saturating shift of
// a full word.
func TestARMEncodingCorners(t *testing.T) {
	t.Run("the unconditional ARM space names the 2 variants", func(t *testing.T) {
		got, ok := armCoprocessor32(0xFE100C10)
		if !ok || !strings.HasPrefix(got, "mrc2 ") {
			t.Errorf("armCoprocessor32 = %q,%v; want an mrc2", got, ok)
		}
	})

	t.Run("the reserved IT condition reads as always", func(t *testing.T) {
		if got := armThumbIT(0b1111, 0b0010); got != "ittt al" {
			t.Errorf("armThumbIT = %q, want %q", got, "ittt al")
		}
	})

	t.Run("register 31 is the zero register", func(t *testing.T) {
		if got := armReg64(31, true); got != "xzr" {
			t.Errorf("armReg64 = %q, want xzr", got)
		}
		if got := armReg64(31, false); got != "wzr" {
			t.Errorf("armReg64 = %q, want wzr", got)
		}
	})

	t.Run("a shifted move-wide names its shift", func(t *testing.T) {
		// movz x13, #0x1234, lsl #32
		got, ok := armMoveWide64(0xD2C24680 | 13)
		if !ok || !strings.HasSuffix(got, ", lsl #32") {
			t.Errorf("armMoveWide64 = %q,%v", got, ok)
		}
	})
}

// armWord assembles an ARM instruction word from its little-endian bytes.
func armWord(hexBytes string) uint32 {
	b, err := hex.DecodeString(hexBytes)
	if err != nil || len(b) != 4 {
		panic("bad instruction word: " + hexBytes)
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// The ARM families x/arch declines are decoded here, so they are driven
// directly. Each expectation came from the oracle.
func TestARM32ExtraFamilies(t *testing.T) {
	for _, tc := range []struct{ name, word, want string }{
		{
			"block transfer, decrement before", "6a7b5fa9",
			"ldmdbge pc, {r1, r3, r5, r6, r8, sb, fp, ip, sp, lr} ^",
		},
		{
			"block transfer, decrement after", "061966d8",
			"stmdale r6!, {r1, r2, r8, fp, ip} ^",
		},
		{
			"block transfer, increment after", "27caf7d8",
			"ldmle r7!, {r0, r1, r2, r5, sb, fp, lr, pc} ^",
		},
		{
			"block transfer, increment before", "3337e9a9",
			"stmibge sb!, {r0, r1, r4, r5, r8, sb, sl, ip, sp} ^",
		},
		{"halving subtract", "773a7496", "uhsub16ls r3, r4, r7"},
		{"unsigned subtract-add exchange", "5abc52d6", "usaxle fp, r2, sl"},
		{"saturating add-subtract exchange", "36b623c6", "qasxgt fp, r3, r6"},
		{"unsigned byte subtract", "ffc45b26", "usub8hs ip, fp, pc"},
		{"unsigned saturating exchange", "305e6066", "uqasxvs r5, r0, r0"},
		{"signed halfword load, negated register", "f10b18e1", "ldrsh r0, [r8, -r1]"},
		{"halfword load, register offset", "bd7d9621", "ldrhhs r7, [r6, sp]"},
		{"signed halfword load, post-indexed", "f8099fe0", "ldrsh r0, [pc], r8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := armDecodeARM32Extra(armWord(tc.word))
			if !ok || got != tc.want {
				t.Errorf("armDecodeARM32Extra = %q,%v; want %q", got, ok, tc.want)
			}
		})
	}
}

// Each of those families refuses the encodings that belong to another group or
// that the manual leaves unpredictable.
func TestARM32ExtraRejections(t *testing.T) {
	for _, tc := range []struct {
		name   string
		decode func() (string, bool)
	}{
		{"a block transfer in the unconditional space", func() (string, bool) {
			return armBlockTransfer32(0xF8100001)
		}},
		{"a word that is not a block transfer", func() (string, bool) {
			return armBlockTransfer32(0xE0100001)
		}},
		{"parallel arithmetic in the unconditional space", func() (string, bool) {
			return armParallelArithmetic32(0xF6143F17)
		}},
		{"parallel arithmetic with its marker bit clear", func() (string, bool) {
			return armParallelArithmetic32(0xE6143F07)
		}},
		{"an unallocated parallel family", func() (string, bool) {
			return armParallelArithmetic32(0xE6043F17)
		}},
		{"an unallocated parallel operation", func() (string, bool) {
			return armParallelArithmetic32(0xE61430B7)
		}},
		{"a word that is not parallel arithmetic", func() (string, bool) {
			return armParallelArithmetic32(0xE0143F17)
		}},
		{"an extra load in the unconditional space", func() (string, bool) {
			return armExtraLoadStore32(0xF01670B2)
		}},
		{"a doubleword access, which is another group", func() (string, bool) {
			return armExtraLoadStore32(0xE0167092)
		}},
		{"an unprivileged store, which does not exist", func() (string, bool) {
			return armExtraLoadStore32(0xE0A670B5)
		}},
		{"a word that is not an extra access", func() (string, bool) {
			return armExtraLoadStore32(0xE41670B2)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if text, ok := tc.decode(); ok {
				t.Errorf("decoded to %q, want no decode", text)
			}
		})
	}
}

// An extra access with no displacement prints its base alone, and a pre-indexed
// one can write the result back.
func TestARM32ExtraAddressForms(t *testing.T) {
	for _, tc := range []struct{ name, word, want string }{
		{"zero immediate displacement", "b000d6e1", "ldrh r0, [r6]"},
		{"pre-indexed with writeback", "b402f6e1", "ldrh r0, [r6, #0x24]!"},
		{"post-indexed immediate", "b40456e0", "ldrh r0, [r6], #-0x44"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := armExtraLoadStore32(armWord(tc.word))
			if !ok || got != tc.want {
				t.Errorf("armExtraLoadStore32 = %q,%v; want %q", got, ok, tc.want)
			}
		})
	}
}

// The coprocessor and bitfield decoders refuse a few more encodings, and the
// coprocessor keeps the sign on a displacement of zero.
func TestARMCoprocessorAndBitfieldEdges(t *testing.T) {
	t.Run("an unallocated Thumb coprocessor group", func(t *testing.T) {
		if text, ok := armThumb32Coprocessor(0xEC10, 0x0A10); ok {
			t.Errorf("decoded to %q, want no decode", text)
		}
	})

	t.Run("an unallocated ARM coprocessor group", func(t *testing.T) {
		if text, ok := armCoprocessor32(0xE0000C10); ok {
			t.Errorf("decoded to %q, want no decode", text)
		}
	})

	t.Run("the unindexed form needs an increasing address", func(t *testing.T) {
		// Thumb and ARM both refuse the option form when U is clear, and the
		// refusal travels back out through the group dispatch.
		if got, ok := armThumb32Coprocessor(0xEC0F, 0x89DE); ok {
			t.Errorf("Thumb decoded to %q, want no decode", got)
		}
		if got, ok := armCoprocessor32(0x8C0E880C); ok {
			t.Errorf("ARM decoded to %q, want no decode", got)
		}
	})

	t.Run("a displacement of zero keeps its sign", func(t *testing.T) {
		// stcvs p2, c7, [r1], #-0
		got, ok := armCoprocessor32(armWord("0072216c"))
		if !ok || !strings.HasSuffix(got, "#-0") {
			t.Errorf("armCoprocessor32 = %q,%v; want it to end #-0", got, ok)
		}
		// The same rule in the Thumb encoding.
		if got, ok := armThumb32Coprocessor(0xEC21, 0x7200); !ok || !strings.HasSuffix(got, "#-0") {
			t.Errorf("armThumb32Coprocessor = %q,%v; want it to end #-0", got, ok)
		}
	})

	t.Run("a paired transfer with no register to pair", func(t *testing.T) {
		var inst armasm.Inst
		inst.Args[0] = armasm.Imm(4) // not a register
		if got := armPairedTransfer32("strd", inst, "whatever"); got != "whatever" {
			t.Errorf("got %q, want the operands unchanged", got)
		}
		inst.Args[0] = armasm.R3
		if got := armPairedTransfer32("strd", inst, "r3"); got != "r3" {
			t.Errorf("got %q, want the operands unchanged when there is nothing to split", got)
		}
	})
}

// TestARMMnemonicTypeSuffix checks that the data-type suffix of a floating-point
// mnemonic keeps its dot while the flag and condition markers run straight on.
func TestARMMnemonicTypeSuffix(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"86dac99e", "vdivls.f32 s27, s19, s12"},
		{"c77a39ce", "vsubgt.f32 s14, s19, s14"},
		{"4cfab85e", "vcvtpl.f32.u32 s30, s24"},
		{"4a0bfc0e", "vcvtreq.u32.f64 s1, d10"},
		{"08db52ee", "vnmls.f64 d29, d2, d8"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}
}

// TestARMVFPExpandImmediate checks the eight-bit floating-point constant
// encoding against values read back from Capstone.
func TestARMVFPExpandImmediate(t *testing.T) {
	for _, tc := range []struct {
		imm8 uint32
		wide bool
		want float64
	}{
		{0x00, false, 2},
		{0x01, false, 2.125},
		{0x0f, false, 3.875},
		{0x10, false, 4},
		{0x40, false, 0.125},
		{0x44, false, 0.15625},
		{0x7f, false, 1.9375},
		{0x80, false, -2},
		{0x84, false, -2.5},
		{0xc0, false, -0.125},
		{0xff, false, -1.9375},
		{0x00, true, 2},
		{0x3f, true, 31},
		{0xc0, true, -0.125},
		{0xff, true, -1.9375},
	} {
		if got := armVFPExpandImm(tc.imm8, tc.wide); got != tc.want {
			t.Errorf("armVFPExpandImm(%#x, %v) = %v, want %v", tc.imm8, tc.wide, got, tc.want)
		}
	}
}

// TestARMVFPFormatting covers the move-immediate rendering and the fixed-point
// conversion suffixes, both of which x/arch spells differently from Capstone.
func TestARMVFPFormatting(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"047ab49e", "vmovls.f32 s14, #1.562500e-01"},
		{"000abcee", "vmov.f32 s0, #-1.250000e-01"},
		{"0f0bb3ee", "vmov.f64 d0, #3.100000e+01"},
		{"682bfb7e", "vcvtvc.f64.u16 d18, d18, #0xffffffff"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}
}

// TestARMVFPLoadStoreMultiple covers the extension register transfers, whose
// register list is bounded by the size of the bank and, for the 64-bit form, by
// a limit of sixteen. An odd byte count selects the legacy X encoding.
func TestARMVFPLoadStoreMultiple(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"000b91ec", "vldmia r1, {d0}"},
		{"010b91ec", "fldmiax r1, {d0}"},
		{"050b91ec", "fldmiax r1, {d0, d1}"},
		{"400b91ec", "vldmia r1, {d0, d1, d2, d3, d4, d5, d6, d7, d8, d9, d10, d11, d12, d13, d14, d15}"},
		{"000a91ec", "vldmia r1, {s0}"},
		{"030a91ec", "vldmia r1, {s0, s1, s2}"},
		{"020b31ed", "vldmdb r1!, {d0}"},
		{"030b21ed", "fstmdbx r1!, {d0}"},
		{"020bbdec", "vpop {d0}"},
		{"020b2ded", "vpush {d0}"},
		{"030bbdec", "fldmiax sp!, {d0}"},
		{"030abdec", "vpop {s0, s1, s2}"},
		{"2cfb6d5d", "vpushpl {d31}"},
		{"d98b2f0d", "fstmdbxeq pc!, {d8, d9, d10, d11, d12, d13, d14, d15, d16, d17, d18, d19, d20, d21, d22, d23}"},
		{"6a2a972c", "vldmiahs r7, {s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14, s15, s16, s17, s18, s19, s20, s21, s22, s23, s24, s25, s26, s27, s28, s29, s30, s31}"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// The unallocated corners of the same encoding: both index bits alike, and
	// the legacy X form naming a register in the upper bank, which predates it.
	for _, hexBytes := range []string{"040bb1ed", "040b11ec", "738b714d", "dbfb75cd"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMVFPZeroDisplacement checks the sign Capstone keeps on a negative
// displacement of zero, which x/arch drops together with the offset.
func TestARMVFPZeroDisplacement(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"002b13ed", "vldr d2, [r3, #-0]"},
		{"002a03ed", "vstr s4, [r3, #-0]"},
		{"002b93ed", "vldr d2, [r3]"},
		{"012b13ed", "vldr d2, [r3, #-4]"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}
}

// TestARMVFPDataProcessing covers the floating-point operations x/arch has no
// table for: the fused multiply-accumulates, the rounding family, and the
// half-precision conversions that involve a 64-bit register.
func TestARMVFPDataProcessing(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"232aa0ee", "vfma.f32 s4, s0, s7"},
		{"632aa0ee", "vfms.f32 s4, s0, s7"},
		{"232a90ee", "vfnms.f32 s4, s0, s7"},
		{"632a90ee", "vfnma.f32 s4, s0, s7"},
		{"232ba0ee", "vfma.f64 d2, d0, d19"},
		{"632ba0ee", "vfms.f64 d2, d0, d19"},
		{"0fcbea4e", "vfmami.f64 d28, d10, d15"},
		{"accaddee", "vfnms.f32 s25, s27, s25"},
		{"632ab6ee", "vrintr.f32 s4, s7"},
		{"e32ab6ee", "vrintz.f32 s4, s7"},
		{"632ab7ee", "vrintx.f32 s4, s7"},
		{"632bb6ee", "vrintr.f64 d2, d19"},
		{"e45bb6ee", "vrintz.f64 d5, d20"},
		{"632bb7ee", "vrintx.f64 d2, d19"},
		{"632bb2ee", "vcvtb.f64.f16 d2, s7"},
		{"e32bb2ee", "vcvtt.f64.f16 d2, s7"},
		{"632bb3ee", "vcvtb.f16.f64 s4, d19"},
		{"e32bb3ee", "vcvtt.f16.f64 s4, d19"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}
}

// TestARMVFPCoreRegisterPair covers the transfers between a pair of core
// registers and one 64-bit or two consecutive 32-bit extension registers.
func TestARMVFPCoreRegisterPair(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"193a45ec", "vmov s18, s19, r3, r5"},
		{"393a45ec", "vmov s19, s20, r3, r5"},
		{"193b45ec", "vmov d9, r3, r5"},
		{"393b45ec", "vmov d25, r3, r5"},
		{"193a55ec", "vmov r3, r5, s18, s19"},
		{"193b55ec", "vmov r3, r5, d9"},
		{"1ffa4eec", "vmov s30, s31, pc, lr"},
		{"10da4dec", "vmov s0, s1, sp, sp"},
		{"3f3b45ec", "vmov d31, r3, r5"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// The 32-bit form names a pair, so it cannot start on the last register.
	for _, hexBytes := range []string{"3f3a45ec", "3f3a55ec"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMVFPScalarTransfer covers the 8, 16 and 32-bit transfers between a core
// register and a lane of an extension register, the lane broadcast, and the
// floating-point system registers.
func TestARMVFPScalarTransfer(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"303b05ee", "vmov.16 d5[0], r3"},
		{"703b25ee", "vmov.16 d5[3], r3"},
		{"f03b05ee", "vmov.16 d21[1], r3"},
		{"103b45ee", "vmov.8 d5[0], r3"},
		{"703b65ee", "vmov.8 d5[7], r3"},
		{"303b15ee", "vmov.s16 r3, d5[0]"},
		{"703b75ee", "vmov.s8 r3, d5[7]"},
		{"303b95ee", "vmov.u16 r3, d5[0]"},
		{"703bf5ee", "vmov.u8 r3, d5[7]"},
		{"103b85ee", "vdup.32 d5, r3"},
		{"b03b85ee", "vdup.16 d21, r3"},
		{"103bc5ee", "vdup.8 d5, r3"},
		{"103ba4ee", "vdup.32 q2, r3"},
		{"903be4ee", "vdup.8 q10, r3"},
		{"103ae0ee", "vmsr fpsid, r3"},
		{"103aeaee", "vmsr fpinst2, r3"},
		{"10fae8ee", "vmsr fpexc, pc"},
		{"103af5ee", "vmrs r3, mvfr2"},
		{"103af7ee", "vmrs r3, mvfr0"},
		{"10faf0ee", "vmrs pc, fpsid"},
		{"10faf1ee", "vmrs APSR_nzcv, fpscr"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// A lane selector with no size, a broadcast onto an odd quadword, a
	// broadcast with both size bits set, and the read-only system registers.
	for _, hexBytes := range []string{
		"503b05ee", "903bafee", "303be4ee", "103ae5ee", "103aebee",
	} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMVFPCompareZero checks the reserved low bits of a compare against zero.
// x/arch ignores them, so the word decodes and has to be rejected afterwards.
func TestARMVFPCompareZero(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"402ab5ee", "vcmp.f32 s4, #0"},
		{"402bb5ee", "vcmp.f64 d2, #0"},
		{"c02ab5ee", "vcmpe.f32 s4, #0"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	for _, hexBytes := range []string{"412ab5ee", "602ab5ee", "c90bf58e"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMCoprocessorNaming pins the order Capstone puts the unconditional
// marker and the long marker in, and the name it gives the flags when a
// register transfer reads them back into the application status register.
func TestARMCoprocessorNaming(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"0840c3ed", "stcl p0, c4, [r3, #0x20]"},
		{"0840d30d", "ldcleq p0, c4, [r3, #0x20]"},
		{"0840c3fd", "stc2l p0, c4, [r3, #0x20]"},
		{"0840d3fd", "ldc2l p0, c4, [r3, #0x20]"},
		{"38f045ee", "mcr p0, #2, pc, c5, c8, #1"},
		{"38f055ee", "mrc p0, #2, apsr_nzcv, c5, c8, #1"},
		{"38f055fe", "mrc2 p0, #2, apsr_nzcv, c5, c8, #1"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}
}

// TestARMVFPUnconditional covers the floating-point instructions the ARMv8
// architecture added in the unconditional encoding space: the select, the
// IEEE minimum and maximum, and the explicitly rounded round and convert.
func TestARMVFPUnconditional(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"032a00fe", "vseleq.f32 s4, s0, s6"},
		{"232b10fe", "vselvs.f64 d2, d0, d19"},
		{"032a20fe", "vselge.f32 s4, s0, s6"},
		{"032b30fe", "vselgt.f64 d2, d0, d3"},
		{"032a80fe", "vmaxnm.f32 s4, s0, s6"},
		{"632b80fe", "vminnm.f64 d2, d0, d19"},
		{"432ab8fe", "vrinta.f32 s4, s6"},
		{"632bb9fe", "vrintn.f64 d2, d19"},
		{"432abafe", "vrintp.f32 s4, s6"},
		{"432bbbfe", "vrintm.f64 d2, d3"},
		{"432abcfe", "vcvta.u32.f32 s4, s6"},
		{"c32bbcfe", "vcvta.s32.f64 s4, d3"},
		{"632abdfe", "vcvtn.u32.f32 s4, s7"},
		{"c32abefe", "vcvtp.s32.f32 s4, s6"},
		{"e32bbffe", "vcvtm.s32.f64 s4, d19"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// A reserved opcode, a select with bit 6 set, a rounding with bit 6 clear,
	// and a rounding selector below the allocated range.
	for _, hexBytes := range []string{"032a90fe", "432a00fe", "032ab8fe", "432ab0fe"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMExceptionTransfer covers the exception-return and stack-save
// instructions, which live in the unconditional half of the block transfer
// encoding.
func TestARMExceptionTransfer(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"000a10f8", "rfeda r0"},
		{"000a1ff8", "rfeda pc"},
		{"000a33f8", "rfeda r3!"},
		{"000a90f8", "rfeia r0"},
		{"000a1df9", "rfedb sp"},
		{"000a9df9", "rfeib sp"},
		{"00054df8", "srsda sp, #0"},
		{"1f054df8", "srsda sp, #0x1f"},
		{"00056df8", "srsda sp!, #0"},
		{"0005cdf8", "srsia sp, #0"},
		{"1f054df9", "srsdb sp, #0x1f"},
		{"0005edf9", "srsib sp!, #0"},
		// Capstone prints the addressing mode in place of the base register
		// when a writeback form does not match the encoding exactly.
		{"000530f8", "rfeda #3!"},
		{"0005b0f8", "rfeia #1!"},
		{"010a30f9", "rfedb #4!"},
		{"1f05b0f9", "rfeib #2!"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// Without writeback the encoding has to match exactly; a stack save names
	// no register but the stack pointer, and neither form sets bit 22 the
	// other's way.
	for _, hexBytes := range []string{"000510f8", "00054cf8", "000a50f8", "0005ddf8"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestARMChangeProcessorState covers CPS, whose operands Capstone prints from
// three separate fields and whose reserved bits it checks unevenly.
func TestARMChangeProcessorState(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"000000f1", "cps #0"},
		{"920000f1", "cps #0x12"},
		{"0f0002f1", "cps #0xf"},
		{"000002f1", "cps #0"},
		{"1f0002f1", "cps #0x1f"},
		{"000008f1", "cpsie none"},
		{"400008f1", "cpsie f"},
		{"800008f1", "cpsie i"},
		{"c00008f1", "cpsie if"},
		{"000108f1", "cpsie a"},
		{"c00108f1", "cpsie aif"},
		{"86000af1", "cpsie i, #6"},
		{"df010ef1", "cpsid aif, #0x1f"},
	} {
		word, err := hex.DecodeString(tc.hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", tc.hexBytes, err)
		}
		got, ok := armDecodeOne(word, arm32, 0)
		if !ok || got != tc.want {
			t.Errorf("armDecodeOne(%s) = %q,%v; want %q", tc.hexBytes, got, ok, tc.want)
		}
	}

	// The reserved bit above the mode, the unallocated change, and the mode
	// values Capstone rejects when the word does not name one.
	for _, hexBytes := range []string{"200000f1", "000004f1", "060000f1", "d20000f1"} {
		word, err := hex.DecodeString(hexBytes)
		if err != nil {
			t.Fatalf("decode %s: %v", hexBytes, err)
		}
		if got, ok := armDecodeOne(word, arm32, 0); ok {
			t.Errorf("armDecodeOne(%s) = %q, want it rejected", hexBytes, got)
		}
	}
}

// TestThumb32MultiplyPackAndExclusives covers the halfword and unsigned long
// multiplies, the pack instruction, the parallel arithmetic, and the exclusive
// accesses, none of which x/arch has a table for.
func TestThumb32MultiplyPackAndExclusives(t *testing.T) {
	for _, tc := range []struct {
		hexBytes string
		want     string
	}{
		{"c3fb8756", "smlalbb r5, r6, r3, r7"},
		{"c3fb9756", "smlalbt r5, r6, r3, r7"},
		{"c3fba756", "smlaltb r5, r6, r3, r7"},
		{"c3fbb756", "smlaltt r5, r6, r3, r7"},
		{"e3fb6756", "umaal r5, r6, r3, r7"},
		{"93fbf7f6", "sdiv r6, r3, r7"},
		{"b3fbf7f6", "udiv r6, r3, r7"},
		{"c3ea0604", "pkhbt r4, r3, r6"},
		{"c3ea8614", "pkhbt r4, r3, r6, lsl #6"},
		{"c3eac674", "pkhbt r4, r3, r6, lsl #0x1f"},
		{"c3ea2604", "pkhtb r4, r3, r6, asr #0x20"},
		{"c3eaa614", "pkhtb r4, r3, r6, asr #6"},
		{"83fa06f4", "sadd8 r4, r3, r6"},
		{"83fa66f4", "uhadd8 r4, r3, r6"},
		{"93fa16f4", "qadd16 r4, r3, r6"},
		{"a3fa26f4", "shasx r4, r3, r6"},
		{"c3fa46f4", "usub8 r4, r3, r6"},
		{"d3fa56f4", "uqsub16 r4, r3, r6"},
		{"e3fa66f4", "uhsax r4, r3, r6"},
		{"c3e87056", "strexd r0, r5, r6, [r3]"},
		{"c3e8405f", "strexb r0, r5, [r3]"},
		{"c3e8505f", "strexh r0, r5, [r3]"},
		{"53e8005f", "ldrex r5, [r3]"},
		{"43e80056", "strex r6, r5, [r3]"},
		{"d3e807f0", "tbb [r3, r7]"},
		{"d1e87f91", "ldrexd sb, r1, [r1]"},
		{"d1e84f9f", "ldrexb sb, [r1]"},
		{"d1e85f9f", "ldrexh sb, [r1]"},
		{"d3e817f0", "tbh [r3, r7, lsl #1]"},
		{"63e80056", "strd r5, r6, [r3], #-0"},
		{"e3e80056", "strd r5, r6, [r3], #0"},
	} {
		out, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
			Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
		if err != nil {
			t.Errorf("%s: %v", tc.hexBytes, err)
			continue
		}
		if out.String() != tc.want {
			t.Errorf("%s = %q, want %q", tc.hexBytes, out.String(), tc.want)
		}
	}

	// A divide whose low result register is named, a pack with the flag bit or
	// a rotation, an exclusive load without its fixed field, a table branch
	// naming a register where the encoding fixes one, a parallel operation with
	// a reserved selector, and a saturate with the bit its encoding fixes.
	for _, hexBytes := range []string{
		"93fbf756", "d3ea8614", "c3eab614", "53e80056", "d3e80056", "b3fa06f4",
	} {
		if _, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
			Execute(core.NewDish([]byte(hexBytes), core.TypeString)); err == nil {
			t.Errorf("%s decoded, want no decode", hexBytes)
		}
	}
}

// TestARMDecoderGuards exercises the guards and forms that the surrounding
// decoders reach only for encodings x/arch happens to take first, so they are
// called here directly. Every expected string was read back from the oracle for
// the encoding it is built from.
func TestARMDecoderGuards(t *testing.T) {
	t.Run("register-offset transfers in every form", func(t *testing.T) {
		for _, tc := range []struct {
			raw  uint32
			want string
		}{
			{0xE7D7700D, "ldrb r7, [r7, sp]"},
			{0xE7977000, "ldr r7, [r7, r0]"},
			{0xE667700D, "strbt r7, [r7], -sp"},
			{0xE637700D, "ldrt r7, [r7], -sp"},
		} {
			if got, ok := armRegisterOffset32(tc.raw); !ok || got != tc.want {
				t.Errorf("armRegisterOffset32(%#x) = %q,%v; want %q", tc.raw, got, ok, tc.want)
			}
		}
	})

	t.Run("the multiply group with and without an accumulator", func(t *testing.T) {
		// smlabb r1, r4, r3, r2, then the same encoding naming no accumulator.
		if got, ok := armThumb32Multiply(0xFB14, 0x2103); !ok || got != "smlabb r1, r4, r3, r2" {
			t.Errorf("armThumb32Multiply = %q,%v; want an smlabb", got, ok)
		}
		if got, ok := armThumb32Multiply(0xFB14, 0xF103); !ok || got != "smulbb r1, r4, r3" {
			t.Errorf("armThumb32Multiply = %q,%v; want an smulbb", got, ok)
		}
	})

	t.Run("the floating-point conversions that need a wide register", func(t *testing.T) {
		for _, raw := range []uint32{0xEEB20A43, 0xEEB30A43, 0xEEB70AC3} {
			if got, ok := armVFPDataProcessing32(raw, armCoprocVFPLow, ""); ok {
				t.Errorf("armVFPDataProcessing32(%#x) = %q, want no decode", raw, got)
			}
		}
	})

	t.Run("the floating-point unit has no register pair", func(t *testing.T) {
		if got, ok := armCoprocessor32(0xFC432A12); ok {
			t.Errorf("armCoprocessor32 = %q, want no decode", got)
		}
	})

	t.Run("a plain load or store is left to x/arch", func(t *testing.T) {
		if got, ok := armVFPLoadStore32(0xED930A00, armCoprocVFPLow, ""); ok {
			t.Errorf("armVFPLoadStore32 = %q, want no decode", got)
		}
	})

	t.Run("a whole word read out of a lane", func(t *testing.T) {
		if got, ok := armVFPCoreTransfer32(0xEE153B10, armCoprocVFPHigh, ""); !ok ||
			got != "vmov.32 r3, d5[0]" {
			t.Errorf("armVFPCoreTransfer32 = %q,%v; want a vmov.32", got, ok)
		}
	})

	t.Run("vector operands that name an odd half of a pair", func(t *testing.T) {
		for _, decode := range []struct {
			name string
			call func() (string, bool)
		}{
			{"a miscellaneous operation", func() (string, bool) { return armSIMDTwoMiscellaneous(0xF3B70640) }},
			{"a lane broadcast", func() (string, bool) { return armSIMDBroadcastLane(0xF3B51C50) }},
			{"a shift", func() (string, bool) { return armSIMDTwoShift(0xF2901550) }},
			{"a modified immediate", func() (string, bool) { return armSIMDModifiedImmediate(0xF2871650) }},
		} {
			if got, ok := decode.call(); ok {
				t.Errorf("%s decoded to %q, want no decode", decode.name, got)
			}
		}
	})

	t.Run("a shift amount with no size to go with it", func(t *testing.T) {
		if size, ok := armSIMDShiftSize(0); ok {
			t.Errorf("armSIMDShiftSize(0) = %d,%v; want no size", size, ok)
		}
	})

	t.Run("a structure transfer with the reserved bit set", func(t *testing.T) {
		if got, ok := armSIMDElementTransfer(0xF4100000); ok {
			t.Errorf("armSIMDElementTransfer = %q, want no decode", got)
		}
	})

	t.Run("a preload indexed by a shifted register", func(t *testing.T) {
		if got, ok := armThumb32Preload(0xF813, 0xF026, 3); !ok || got != "pld [r3, r6, lsl #2]" {
			t.Errorf("armThumb32Preload = %q,%v; want a shifted pld", got, ok)
		}
	})
}

// TestARMFormattingCorners reaches the last few guards, which the decoders
// above them can no longer produce input for.
func TestARMFormattingCorners(t *testing.T) {
	t.Run("a floating-point move with no operand to keep", func(t *testing.T) {
		var inst armasm.Inst
		inst.Args[1] = armasm.Imm(0x44)
		if got := armVFPMoveImmediate(inst, "vmov.f32", "s0"); got != "s0" {
			t.Errorf("got %q, want the operands unchanged", got)
		}
	})

	t.Run("an immediate that carries its own rotation", func(t *testing.T) {
		var inst armasm.Inst
		inst.Args[1] = armasm.ImmAlt{Val: 0x61, Rot: 8}
		if got := armRotatedImmediate32(inst, "r0, #0x61, 8"); got != "r0, #97, #8" {
			t.Errorf("got %q, want the value and rotation as two immediates", got)
		}
	})

	t.Run("a coprocessor data operation", func(t *testing.T) {
		// cdp p9, #0, c2, c3, c2, #0
		got, ok := armCoprocessor32(0xEE032902)
		if !ok || got != "cdp p9, #0, c2, c3, c2, #0" {
			t.Errorf("armCoprocessor32 = %q,%v; want a cdp", got, ok)
		}
	})

	t.Run("a vector operation naming an odd half of a pair", func(t *testing.T) {
		if got, ok := armSIMDTwoMiscellaneous(0xF3BB1640); ok {
			t.Errorf("armSIMDTwoMiscellaneous = %q, want no decode", got)
		}
	})
}

// TestARM64Spellings checks the names and numbers Capstone prints where x/arch
// chooses differently: a branch tests the same conditions as anything else,
// moving one element of a vector into another is an insert, and a lane number
// is an ordinary immediate.
func TestARM64Spellings(t *testing.T) {
	for _, tc := range []struct{ hexBytes, want string }{
		{"4278f454", "b.hs #0xfffffffffffe8f08"},
		{"aa050a6e", "ins v10.h[2], v13.h[0]"},
		{"c93dbd4d", "st4 {v9.b, v10.b, v11.b, v12.b}[0xf], [x14], x29"},
		{"fa1a914d", "st1 {v26.b}[0xe], [x23], x17"},
		{"2fe11654", "b.nv #0x2dc24"},
		{"e83b61b3", "bfi x8, xzr, #0x1f, #0xf"},
		{"e80f45b3", "bfi x8, xzr, #0x3b, #4"},
		{"e8b75fb3", "bfxil x8, xzr, #0x1f, #0xf"},
		{"0a22ce0c", "ld1 {v10.8b, v11.8b, v12.8b, v13.8b}, [x16], x14"},
	} {
		out, err := armRecipe(arm64, "ARM", "Little Endian", 0, false, false).
			Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
		if err != nil {
			t.Errorf("%s: %v", tc.hexBytes, err)
			continue
		}
		if out.String() != tc.want {
			t.Errorf("%s = %q, want %q", tc.hexBytes, out.String(), tc.want)
		}
	}
}

// TestARM64Constraints covers the AArch64 encodings x/arch accepts and Capstone
// does not: an extend shifted further than four, a load pair naming one
// register twice, a pair written back through a register it also transfers, and
// a logical immediate whose pattern is all ones.
func TestARM64Constraints(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"an extend of four is the widest", "1a72278b", "add x26, x16, x7, uxtx #4"},
		{"and no extend at all is fine", "1a62278b", "add x26, x16, x7, uxtx"},
		{"an extend of five is not", "1a7627cb", ""},
		{"nor is one of seven", "1afe278b", ""},
		{"a store pair may name one register twice", "45168228", "stp w5, w5, [x18], #0x10"},
		{"a load pair may not", "4516c228", ""},
		{"a pair written back through its own base", "a5188228", ""},
		{"and the same before the access", "a5188229", ""},
		{"unless the base is the stack pointer", "e51bc228", "ldp w5, w6, [sp], #0x10"},
		{"or nothing is written back", "451a0229", "stp w5, w6, [x18, #0x10]"},
		{"a logical immediate of every bit", "347d0012", ""},
		{"and its 64-bit spelling", "34fd4092", ""},
		{"a pattern that is not all ones", "343d0012", "and w20, w9, #0xffff"},
		{
			"a floating-point pair may name its own base", "deb7d62c",
			"ldp s30, s13, [x30], #0xb4",
		},
		{"and so may a doubleword one", "63ad8e6c", "stp d3, d11, [x11], #0xe8"},
		{"but it still may not name one register twice", "8f3cff6c", ""},
		{"and a repeating one", "343d0092", "and x20, x9, #0xffff0000ffff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm64, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARM64SystemRegisters covers the generic system register accesses. x/arch
// declines most of them and numbers the rest one lower than Capstone, which
// treats the whole space as though the top field were three.
func TestARM64SystemRegisters(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"a write x/arch declines", "049100d5", "msr s3_0_c9_c1_0, x4"},
		{"and the matching read", "049120d5", "mrs x4, s3_0_c9_c1_0"},
		{"a write it numbers too low", "049110d5", "msr s3_0_c9_c1_0, x4"},
		{"and the matching read", "049130d5", "mrs x4, s3_0_c9_c1_0"},
		{"a write it already agrees on", "64361dd5", "msr s3_5_c3_c6_3, x4"},
		{"a system operation", "049108d5", "sys #0, c9, c1, #0, x4"},
		{"one that transfers no register", "bf4608d5", "sys #0, c4, c6, #5"},
		{"and one that reads back", "049128d5", "sysl x4, #0, c9, c1, #0"},
		{"an address translation", "047808d5", "at s1e1r, x4"},
		{"an instruction cache operation", "047108d5", "ic ialluis"},
		{"a data cache operation", "24740bd5", "dc zva, x4"},
		{"a whole-table invalidate", "048308d5", "tlbi vmalle1is"},
		{"and one that names an address", "248308d5", "tlbi vae1is, x4"},
		{"an operation with no name of its own", "64360dd5", "sys #5, c3, c6, #3, x4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm64, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMExtraLoadStoreForms covers the halfword, signed byte and doubleword
// transfers whose unprivileged and written-back forms x/arch declines.
func TestARMExtraLoadStoreForms(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"an unprivileged halfword load", "b52833e0", "ldrht r2, [r3], -r5"},
		{"an unprivileged signed byte", "d52833e0", "ldrsbt r2, [r3], -r5"},
		{"an unprivileged signed halfword", "f52833e0", "ldrsht r2, [r3], -r5"},
		{"and the same counting upwards", "b528b3e0", "ldrht r2, [r3], r5"},
		{"a halfword store written back", "b528e3e1", "strh r2, [r3, #0x85]!"},
		{"a doubleword load written back", "d528e3e1", "ldrd r2, r3, [r3, #0x85]!"},
		{"a doubleword store written back", "f528e3e1", "strd r2, r3, [r3, #0x85]!"},
		{"a halfword load written back", "b528f3e1", "ldrh r2, [r3, #0x85]!"},
		{"a signed byte written back", "d528f3e1", "ldrsb r2, [r3, #0x85]!"},
		{"a signed halfword written back", "f528f3e1", "ldrsh r2, [r3, #0x85]!"},
		{"a plain offset still works", "b528c3e1", "strh r2, [r3, #0x85]"},
		{"and so does a post-indexed register", "b52803e0", "strh r2, [r3], -r5"},
		{"an unprivileged store has no such form", "b528a3e0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMStatusRegisters covers the transfers to and from the status registers
// and the bitfield insert, all of which x/arch decodes only in part.
func TestARMStatusRegisters(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"a write to the saved status register", "00b04f21", "mrshs fp, spsr"},
		{"and one from the current one", "00b00f21", "mrshs fp, apsr"},
		{"an immediate into every field", "68ff2f83", "msrhi cpsr_fsxc, #104, #30"},
		{"into the control field alone", "68ff2183", "msrhi cpsr_c, #104, #30"},
		{"the flags, which are named for the application", "68ff2883", "msrhi apsr_nzcvq, #104, #30"},
		{"the greater-than field, likewise", "68ff2483", "msrhi apsr_g, #104, #30"},
		{"and both together", "68ff2c83", "msrhi apsr_nzcvqg, #104, #30"},
		{"a mixed field keeps the letters", "68ff2983", "msrhi cpsr_fc, #104, #30"},
		{"the saved register names no fields at all", "68ff6083", "msrhi spsr, #104, #30"},
		{"and its flags keep their letter", "68ff6883", "msrhi spsr_f, #104, #30"},
		{"an unrotated immediate stands alone", "07f02983", "msrhi cpsr_fc, #7"},
		{"the current register must name a field", "68ff2083", ""},
		{"a bitfield insert", "9a40cf17", "bfine r4, sl, #1, #0xf"},
		{"a bitfield clear", "9f40cf17", "bfcne r4, #1, #0xf"},
		{"one whose top bit falls below its position", "9a42c417", "bfine r4, sl, #4, #1"},
		{"and one that falls well below it", "1a4dcf17", "bfine r4, sl, #0xf, #1"},
		{"a clear whose top bit falls below it too", "1f4dcf17", "bfcne r4, #0xf, #1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestThumbBitfieldUnderflow checks the Thumb spelling of a bitfield whose top
// bit lies below its position, which Capstone renders the same way it does in
// ARM: the top bit stands in for the position and the width becomes one.
func TestThumbBitfieldUnderflow(t *testing.T) {
	for _, tc := range []struct{ hexBytes, want string }{
		{"6af34f04", "bfi r4, sl, #1, #0xf"},
		{"6af34414", "bfi r4, sl, #4, #1"},
		{"6af38f64", "bfi r4, sl, #0xf, #1"},
		{"6ff34414", "bfc r4, #4, #1"},
		{"6ff38f64", "bfc r4, #0xf, #1"},
	} {
		out, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
			Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
		if err != nil {
			t.Errorf("%s: %v", tc.hexBytes, err)
			continue
		}
		if out.String() != tc.want {
			t.Errorf("%s = %q, want %q", tc.hexBytes, out.String(), tc.want)
		}
	}
}

// TestThumbProcessorStateAndTraps covers the permanently undefined instruction
// Capstone gives a name of its own, and the 16-bit changes of processor state
// and endianness.
func TestThumbProcessorStateAndTraps(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"the always condition adds no suffix", "e4bf338f", "itt al\nldrh r3, [r6, #0x38]"},
		{"a branch takes the block's condition in place of its own", "d6bf2dda", "itet le\nble #0x60"},
		{"even where the two agree", "08bf01d0", "it eq\nbeq #8"},
		{"the undefined instruction has a name of its own", "fede", "trap"},
		{"but only at that one value", "ffde", "udf #0xff"},
		{"setting the endianness", "50b6", "setend le"},
		{"and the other way", "58b6", "setend be"},
		{"enabling an interrupt", "62b6", "cpsie i"},
		{"disabling one", "72b6", "cpsid i"},
		{"naming none of them", "60b6", "cpsie none"},
		{"and naming all of them", "77b6", "cpsid aif"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMLooseFixedFields covers the ARM families whose nominally fixed fields
// x/arch insists on and Capstone ignores.
func TestARMLooseFixedFields(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"a byte selection", "b9928306", "seleq sb, r3, sb"},
		{"and with the field as encoded", "b99f8306", "seleq sb, r3, sb"},
		{"a signed halfword multiply", "84226ec1", "smulbbgt lr, r4, r2"},
		{"and with a different spare", "84f26ec1", "smulbbgt lr, r4, r2"},
		{"a byte swap", "90224d51", "swpbpl r2, r0, [sp]"},
		{"an accumulating extend", "71b9e396", "uxtabls fp, r3, r1, ror #16"},
		{"a saturating subtract", "56822f11", "qsubne r8, r6, pc"},
		{"a halfword multiply from the other halves", "cf3a6381", "smulbthi r3, pc, sl"},
		{"a read of the saved status register", "00b04221", "mrshs fp, spsr"},
		{"a call to the hypervisor", "75de40e1", "hvc #0xde5"},
		{"an extend that breaks its own marker", "f1bbe396", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestThumbITBlockConditions checks the condition an IT block applies to each
// instruction it covers: in place of the flag-setting suffix an unconditional
// form carries, in place of a branch's own condition, and not at all where the
// block names the condition that always holds or the instruction takes none.
func TestThumbITBlockConditions(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{
			"then and else slots take opposite conditions", "5abf01208840481c",
			"itte pl\nmovpl r0, #1\nlslpl r0, r1\naddmi r0, r1, #1",
		},
		{
			"a wide instruction keeps its marker after the condition", "08bfd3f800100220",
			"it eq\nldreq.w r1, [r3]\nmovs r0, #2",
		},
		{
			"the always condition adds no suffix", "e4bf338f",
			"itt al\nldrh r3, [r6, #0x38]",
		},
		{"a branch gives up its own condition", "d6bf2dda", "itet le\nble #0x60"},
		{"even where the two agree", "08bf01d0", "it eq\nbeq #8"},
		{
			"a compare and branch takes none at all", "5fbfc2bb",
			"itttt pl\ncbnz r2, #0x76",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMOperandSpellings covers the operands Capstone writes differently from
// x/arch: a shift amount that stands alone rather than qualifying a register, a
// displacement written back after the access, and a run of vector registers.
func TestARMOperandSpellings(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, arch, want string }{
		{
			"a pack prints its shift as an immediate", "19978216", arm32,
			"pkhbtne sb, r2, sb, lsl #0xe",
		},
		{
			"and so does a saturating shift", "5a26b786", arm32,
			"ssathi r2, #0x18, sl, asr #0xc",
		},
		{
			"a pair transfer keeps the sign of a post-indexed displacement", "1a36fa28", arm64,
			"ldp w26, w13, [x16], #-0x30",
		},
		{
			"a single transfer wraps it round instead", "e0071fb8", arm64,
			"str w0, [sp], #0xfffffffffffffff0",
		},
		{
			"a run of vector registers is written out in full", "0a22ce0c", arm64,
			"ld1 {v10.8b, v11.8b, v12.8b, v13.8b}, [x16], x14",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(tc.arch, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMRegisterOffsetTransfers covers the loads and stores indexed by a
// shifted register, which x/arch declines for several index registers, and the
// branch-exchange forms whose register and low bits it reads differently.
func TestARMRegisterOffsetTransfers(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, mode, arch, want string }{
		{
			"a store indexed by a register x/arch declines", "0d7d07e7", "ARM", arm32,
			"str r7, [r7, -sp, lsl #26]",
		},
		{"and the same store written back", "0d70a7e7", "ARM", arm32, "str r7, [r7, sp]!"},
		{"the stack pointer is a branch target after all", "6847", "Thumb", arm32, "bx sp"},
		{"a linking branch with its low bits set is not one", "8547", "Thumb", arm32, ""},
		{
			"adding the stack pointer names its register twice", "ec4460d0", "Thumb", arm32,
			"add ip, sp, ip\nbeq #0xc6",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(tc.arch, tc.mode, "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMRegisterListLimits covers the transfers whose register list or pair
// would run past the end of the register file, or hold nothing at all.
func TestARMRegisterListLimits(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, mode, arch, want string }{
		{"an empty register list transfers nothing", "00b4", "Thumb", arm32, ""},
		{"a bitwise not may not name a first source", "0640e3e1", "ARM", arm32, ""},
		{"a doubleword load may not name the program counter", "d2f6e171", "ARM", arm32, ""},
		{
			"but the register below it is fine", "d2e6e171", "ARM", arm32,
			"ldrdvc lr, pc, [r1, #0x62]!",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(tc.arch, tc.mode, "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMHypervisorCall covers HVC, which x/arch has no table for and which
// carries no condition however the word encodes one.
func TestARMHypervisorCall(t *testing.T) {
	for _, tc := range []struct{ hexBytes, want string }{
		{"75de40e1", "hvc #0xde5"},
		{"75e94d91", "hvc #0xde95"},
	} {
		out, err := armRecipe(arm32, "ARM", "Little Endian", 0, false, false).
			Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
		if err != nil {
			t.Errorf("%s: %v", tc.hexBytes, err)
			continue
		}
		if out.String() != tc.want {
			t.Errorf("%s = %q, want %q", tc.hexBytes, out.String(), tc.want)
		}
	}
}

// TestThumbSaturateForms covers the saturating clamps: the halfword forms an
// arithmetic shift of nothing selects, and the bit below the shift amount,
// which Capstone requires clear of the signed forms and of any halfword one.
func TestThumbSaturateForms(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"a signed clamp", "0af30b06", "ssat r6, #0xc, sl"},
		{"and one that shifts left", "0af34b36", "ssat r6, #0xc, sl, lsl #0xd"},
		{"and one that shifts right", "2af34b36", "ssat r6, #0xc, sl, asr #0xd"},
		{"a shift of nothing makes it a halfword clamp", "2af30b06", "ssat16 r6, #0xc, sl"},
		{"an unsigned one", "8af30b06", "usat r6, #0xb, sl"},
		{"an unsigned halfword clamp", "aaf30b06", "usat16 r6, #0xb, sl"},
		{"an unsigned clamp reads past the fixed bit", "8af32b06", "usat r6, #0xb, sl"},
		{"and so does a shifted one", "aaf36b36", "usat r6, #0xb, sl, asr #0xd"},
		{"a signed clamp does not", "0af32b06", ""},
		{"nor does a signed shifted one", "2af36b36", ""},
		{"nor an unsigned halfword one", "aaf32b06", ""},
		{"a clamp whose limit fills the field", "a9f7f439", "usat sb, #0x14, sb, asr #0xf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "Thumb", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARMBankedStatusRegisters covers the reads of the registers a mode banks
// for itself, and the low bits Capstone reads past on a plain status read.
// Where the encoding names no banked register at all Capstone still prints one,
// always the same one and always conditionally; that is reproduced here.
func TestARMBankedStatusRegisters(t *testing.T) {
	for _, tc := range []struct{ name, hexBytes, want string }{
		{"a plain read", "00b00fe1", "mrs fp, apsr"},
		{"one whose low bits are set anyway", "03b40fe1", "mrs fp, apsr"},
		{"one with a bit set below its marker", "10b00fe1", ""},
		{"a banked general register", "00b200e1", "mrs fp, r8_usr"},
		{"a banked stack pointer", "00b205e1", "mrs fp, sp_usr"},
		{"a banked link register", "00b300e1", "mrs fp, lr_irq"},
		{"the exception link register", "00b30ee1", "mrs fp, elr_hyp"},
		{"a banked saved status register", "00b24ee1", "mrs fp, SPSR_fiq"},
		{"and another", "00b34ce1", "mrs fp, SPSR_mon"},
		{"a combination naming no register", "00b240e1", "mrseq fp, lr_fiq"},
		{"a banked read with a bit set below its marker", "01b20fe1", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arm32, "ARM", "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if tc.want == "" {
				if err == nil {
					t.Fatalf("decoded to %q, want no decode", out.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// runARMSubjectCases checks a table of encodings against one architecture and
// mode, where an empty want means the encoding must be refused.
func runARMSubjectCases(t *testing.T, arch, mode string, cases []struct{ name, hexBytes, want string }) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := armRecipe(arch, mode, "Little Endian", 0, false, false).
				Execute(core.NewDish([]byte(tc.hexBytes), core.TypeString))
			if err != nil {
				// Nothing decoded, which is how the operation reports a
				// refusal when the input holds a single instruction.
				if tc.want != "" {
					t.Fatalf("Execute: %v", err)
				}
				return
			}
			if out.String() != tc.want {
				t.Errorf("got %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestARM64ConditionNames covers the two conditions AArch64 spells differently
// from ARM and the one it names where ARM prints nothing at all.
func TestARM64ConditionNames(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"carry set is named for the unsigned comparison", "a3209f9a", "csel x3, x5, xzr, hs"},
		{"and so is carry clear", "a3309f9a", "csel x3, x5, xzr, lo"},
		{"the never condition has a name", "a3f09f9a", "csel x3, x5, xzr, nv"},
		{"where the always condition keeps its own", "a3e09f9a", "csel x3, x5, xzr, al"},
		{"a negating select", "fc258ada", "csneg x28, x15, x10, hs"},
		{"an incrementing one", "e425811a", "csinc w4, w15, w1, hs"},
		{"an inverting one", "a822975a", "csinv w8, w21, w23, hs"},
		{"a conditional comparison", "a4f35bfa", "ccmp x29, x27, #4, nv"},
		{"and a floating-point select", "433c7e1e", "fcsel d3, d2, d30, lo"},
	})
}

// TestARM64LoadAliasingItsBase covers the AArch64 loads Capstone refuses
// because the register they load is the one holding the address. Only the plain
// integer loads carry the constraint, and only in the three addressing modes
// that share their decoder.
func TestARM64LoadAliasingItsBase(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"an unprivileged load of its own base", "001843f8", ""},
		{"a pre-indexed one", "8ccd4378", ""},
		{"a post-indexed one", "a51443f8", ""},
		{"and a pre-indexed doubleword", "a51c43f8", ""},
		{"a sign-extending load into a word", "a514c338", ""},
		{"and its wider spelling", "a514c3b8", ""},
		{"a different register is still fine", "a614c338", "ldrsb w6, [x5], #0x31"},
		{"a different register is fine", "011843f8", "ldtr x1, [x0, #0x31]"},
		{"a store may name its base", "a51803f8", "sttr x5, [x5, #0x31]"},
		{"so may an unscaled load", "001043f8", "ldur x0, [x0, #0x31]"},
		{"and a vector one", "a51c43fc", "ldr d5, [x5, #0x31]!"},
		{"and a sign-extending one into a doubleword", "a51883b8", "ldtrsw x5, [x5, #0x31]"},
		{"or a post-indexed one", "a51483b8", "ldrsw x5, [x5], #0x31"},
		{"the stack pointer is exempt", "ff1f43f8", "ldr xzr, [sp, #0x31]!"},
		{"a register offset carries no constraint", "a56863f8", "ldr x5, [x5, x3]"},
		{"nor does a scaled offset", "a51040f9", "ldr x5, [x5, #0x20]"},
	})
}

// TestARMNegativeZeroDisplacement covers the transfers whose displacement is
// zero and subtracted, which Capstone prints with the sign it was given.
func TestARMNegativeZeroDisplacement(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"a post-indexed halfword", "b07040e0", "strh r7, [r0], #-0"},
		{"the same as an offset", "b07040e1", "strh r7, [r0, #-0]"},
		{"a post-indexed pair", "d07040e0", "ldrd r7, r8, [r0], #-0"},
		{"a post-indexed word", "007010e4", "ldr r7, [r0], #-0"},
		{"the same as an offset", "007010e5", "ldr r7, [r0, #-0]"},
		{"a translated load", "f0c07ab0", "ldrshtlt ip, [sl], #-0"},
	})
	// Thumb spells the same displacement with a hexadecimal prefix where it
	// indexes before the access.
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a post-indexed word", "52f80009", "ldr r0, [r2], #-0"},
		{"a post-indexed signed halfword", "32f90009", "ldrsh r0, [r2], #-0"},
		{"a pre-indexed word written back", "52f8000d", "ldr r0, [r2, #-0x0]!"},
		{"one that is not", "52f8000c", "ldr r0, [r2, #-0x0]"},
		{"and a pre-indexed signed halfword", "32f9000d", "ldrsh r0, [r2, #-0x0]!"},
	})
}

// TestThumbBitfieldReservedBit covers the bit the Thumb bitfield insertions
// require to be clear, which the neighbouring extractions and saturations in
// the same encoding class read past.
func TestThumbBitfieldReservedBit(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"an insertion with the bit clear", "60f30a52", "bfi r2, r0, #0xa, #1"},
		{"the same with it set", "60f70a52", ""},
		{"and a clearing with it set", "6ff70a52", ""},
		{"a signed extraction reads past it", "40f70a52", "sbfx r2, r0, #0x14, #0xb"},
		{"an unsigned one too", "c0f70a52", "ubfx r2, r0, #0x14, #0xb"},
		{"as does a signed saturation", "00f70a52", "ssat r2, #0xb, r0, lsl #0x14"},
		{"an unsigned one", "80f70a52", "usat r2, #0xa, r0, lsl #0x14"},
		{"and a signed halfword one", "20f70a02", "ssat16 r2, #0xb, r0"},
		{"but not an unsigned halfword one", "a0f70a02", ""},
		{"nor its shortest form", "a0f70000", ""},
		{"which is fine with the bit clear", "a0f30a02", "usat16 r2, #0xa, r0"},
		{"and fine with a shift, which is no longer a halfword form", "a0f70a52", "usat r2, #0xa, r0, asr #0x14"},
		{"a halfword clamp counts in four bits", "20f30f02", "ssat16 r2, #0x10, r0"},
		{"and no more", "20f31002", ""},
		{"the unsigned one too", "a0f30f02", "usat16 r2, #0xf, r0"},
		{"and no more", "a0f31002", ""},
		{"a whole-word clamp counts in five", "00f31002", "ssat r2, #0x11, r0"},
		{"and so does a shifted one", "20f31052", "ssat r2, #0x11, r0, asr #0x14"},
	})
}

// TestThumbBranchExchangeToPC covers the two branch exchanges that name the
// program counter, which the architecture leaves unpredictable and Capstone
// prints.
func TestThumbBranchExchangeToPC(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a linking branch exchange", "f847", "blx pc"},
		{"and a plain one", "7847", "bx pc"},
	})
}

// TestThumbSecureAndHypervisorCalls covers the two Thumb calls into a higher
// privilege level: the hypervisor call, which carries a wide marker and joins
// its two immediate fields, and the secure monitor call, which does neither.
func TestThumbSecureAndHypervisorCalls(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a hypervisor call", "e8f78082", "hvc.w #0x8280"},
		{"one of zero", "e0f70080", "hvc.w #0"},
		{"and a secure monitor call", "f5f70080", "smc #5"},
	})
}

// TestThumbModifiedImmediateSign covers the Thumb constants Capstone prints
// unsigned. The bitwise operations print the constant as it stands; every other
// operation, including the moves the bitwise ones become when they name the
// program counter, prints it signed.
func TestThumbModifiedImmediateSign(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a conjunction", "09f0e33a", "and sl, sb, #0xe3e3e3e3"},
		{"one that sets the flags", "19f0e33a", "ands sl, sb, #0xe3e3e3e3"},
		{"a bit clear", "29f0e33a", "bic sl, sb, #0xe3e3e3e3"},
		{"and its flag-setting form", "39f0e33a", "bics sl, sb, #0xe3e3e3e3"},
		{"a disjunction", "49f0e33a", "orr sl, sb, #0xe3e3e3e3"},
		{"and its flag-setting form", "59f0e33a", "orrs sl, sb, #0xe3e3e3e3"},
		{"an exclusive disjunction", "89f0e33a", "eor sl, sb, #0xe3e3e3e3"},
		{"and its flag-setting form", "99f0e33a", "eors sl, sb, #0xe3e3e3e3"},
		{"a negated move", "6ff0e33a", "mvn sl, #0xe3e3e3e3"},
		{"and its flag-setting form", "7ff0e33a", "mvns sl, #0xe3e3e3e3"},
		{"the move a disjunction becomes", "4ff0e33a", "mov.w sl, #-0x1c1c1c1d"},
		{"and its flag-setting form", "5ff0e33a", "movs.w sl, #-0x1c1c1c1d"},
		{"a negated disjunction", "69f0e33a", "orn sl, sb, #-0x1c1c1c1d"},
		{"an addition", "09f1e33a", "add.w sl, sb, #-0x1c1c1c1d"},
		{"and a test", "19f0e33f", "tst.w sb, #-0x1c1c1c1d"},
	})
}

// TestARMStatusReadFields covers the field the ARM status read holds where the
// banked forms hold their register selector. A read of the saved register
// ignores it; a read of the visible one requires its lowest bit.
func TestARMStatusReadFields(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"a clear field", "00500031", ""},
		{"the lowest bit alone", "00500131", "mrslo r5, apsr"},
		{"every bit but the lowest", "00500e31", ""},
		{"and every bit", "00500f31", "mrslo r5, apsr"},
		{"the saved register ignores the field", "00504031", "mrslo r5, spsr"},
		{"whatever it holds", "00504e31", "mrslo r5, spsr"},
	})
}

// TestARMBankedReadFallback covers the banked status reads whose register
// selector names no register. Capstone reads the wrong operands for those,
// taking the register from the condition field and printing a condition of its
// own.
func TestARMBankedReadFallback(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"an allocated selector", "00520301", "mrseq r5, r11_usr"},
		{"one under the always condition", "005203e1", "mrs r5, r11_usr"},
		{"a saved-register selector", "00524e01", "mrseq r5, SPSR_fiq"},
		{"an unallocated selector", "00524001", "mrslo r5, r8_usr"},
		{"one whose condition names no register either", "00524071", "mrslo r5, "},
		{"and one at the last condition", "005240e1", "mrseq r5, lr_fiq"},
	})
}

// TestThumbITBlockUnconditionalOps covers the instructions that keep their own
// spelling inside an IT block rather than taking the block's condition.
func TestThumbITBlockUnconditionalOps(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a breakpoint", "55bf83be", "itete pl\nbkpt #0x83"},
		{"a nested block", "55bf08bf", "itete pl\nit eq"},
		{"an undefined instruction", "55bf01de", "itete pl\nudf #1"},
		{"the one that has a name of its own", "55bffede", "itete pl\ntrap"},
		{"a change of endianness", "55bf50b6", "itete pl\nsetend le"},
		{"enabling an interrupt", "55bf62b6", "itete pl\ncpsie i"},
		{"and disabling one", "55bf72b6", "itete pl\ncpsid i"},
		{"a supervisor call still takes it", "55bf01df", "itete pl\nsvcpl #1"},
		{"and so does a push", "55bf10b4", "itete pl\npushpl {r4}"},
	})
}

// TestARMStatusRegisterWrites covers MSR with a register source, whose field
// mask names the same halves of the status register the immediate form does.
func TestARMStatusRegisterWrites(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"no fields at all", "02f02031", ""},
		{"the control half", "02f02131", "msrlo cpsr_c, r2"},
		{"two halves", "02f02331", "msrlo cpsr_xc, r2"},
		{"the flags under their application name", "02f02431", "msrlo apsr_g, r2"},
		{"and their wider one", "02f02831", "msrlo apsr_nzcvq, r2"},
		{"and both together", "02f02c31", "msrlo apsr_nzcvqg, r2"},
		{"every half", "02f02f31", "msrlo cpsr_fsxc, r2"},
		{"the saved register with no fields", "02f06031", "msrlo spsr, r2"},
		{"one of its halves", "02f06831", "msrlo spsr_f, r2"},
		{"and every one", "02f06f31", "msrlo spsr_fsxc, r2"},
		{"the bits below the source are fixed", "32f02331", ""},
		{"as are the two above the marker", "02fc2331", ""},
		{"and the one below it", "02f12331", ""},
	})
}

// TestARMBankedStatusWrites covers MSR to a banked register, which shares its
// register table with the read. Where the selector names no register Capstone
// reads the wrong operands again, this time taking the register from the source
// field and the immediate from the condition.
func TestARMBankedStatusWrites(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"an allocated selector", "02f22331", "msrlo r11_usr, r2"},
		{"a fast-interrupt register", "02f22c31", "msrlo r12_fiq, r2"},
		{"a saved-register selector", "02f26e31", "msrlo SPSR_fiq, r2"},
		{"the source is any register", "0ff22331", "msrlo r11_usr, pc"},
		{"an unallocated selector", "02f26031", "msrlo r12_usr, #3"},
		{"under the first condition", "02f26001", "msrlo r12_usr, #0"},
		{"under the last", "02f260e1", "msreq r12_usr, #0xe"},
		{"one whose source names no register either", "05f26031", "msrlo , #3"},
		{"the destination field is fixed", "02022331", ""},
	})
}

// TestARM64ConstantMoves covers the logical immediate that moves a constant
// through the zero register, which Capstone spells in full rather than as the
// move alias.
func TestARM64ConstantMoves(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a 32-bit constant", "fd3b1932", "orr w29, wzr, #0x3fff80"},
		{"a 64-bit one", "eaa36db2", "orr x10, xzr, #0xffffffffff80000"},
		{"the smallest", "fd030032", "orr w29, wzr, #1"},
		{"one whose destination is the stack pointer", "ff030032", "orr wsp, wzr, #1"},
		{"and one spanning a whole word", "fd7f40b2", "orr x29, xzr, #0xffffffff"},
	})
}

// TestARM64InsertLane covers the lane an element insertion reads, whose index
// is scaled by the element size, so the bits below it must be clear.
func TestARM64InsertLane(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a doubleword lane", "7246086e", "ins v18.d[0], v19.d[1]"},
		{"one whose index is unscaled", "720e086e", ""},
		{"and one scaled by too little", "7216086e", ""},
		{"a halfword lane", "7216026e", "ins v18.h[0], v19.h[1]"},
		{"one whose index is unscaled", "720e026e", ""},
		{"a word lane", "72260c6e", "ins v18.s[1], v19.s[1]"},
		{"one scaled by too little", "72160c6e", ""},
		{"and a byte lane, which takes any index", "727e016e", "ins v18.b[0], v19.b[0xf]"},
	})
}

// TestARMDecoderRejections exercises the guards each decoder applies to fields
// no ordinary encoding reaches, which the whole-instruction paths cannot cover.
func TestARMDecoderRejections(t *testing.T) {
	t.Run("a mnemonic beginning with b that carries no condition", func(t *testing.T) {
		for _, mnemonic := range []string{"bl", "bic", "bx", "b"} {
			if got := armWithoutCondition(mnemonic); got != mnemonic {
				t.Errorf("armWithoutCondition(%q) = %q, want it unchanged", mnemonic, got)
			}
		}
	})

	t.Run("a constant move with a single operand", func(t *testing.T) {
		if got, ok := armConstantMove64("mov", "w29", 0x32000000); ok {
			t.Errorf("armConstantMove64 = %q,%v; want no rewrite", got, ok)
		}
	})

	t.Run("an element insertion naming no element size", func(t *testing.T) {
		if armInsertLaneIsValid64(0x6E000400) {
			t.Error("armInsertLaneIsValid64 accepted a selector of zero")
		}
	})

	t.Run("logical immediate patterns with no encoding", func(t *testing.T) {
		// The widest element under a 32-bit operation, then a selector
		// leaving an element narrower than two bits.
		if armLogicalPatternIsValid64(0x12400000) {
			t.Error("accepted the widest element in a 32-bit form")
		}
		if armLogicalPatternIsValid64(0x1200FC00) {
			t.Error("accepted an element narrower than two bits")
		}
	})

	t.Run("a bitfield insertion whose fields do not overlap", func(t *testing.T) {
		// The bounds cross only where the top bit lies below the position,
		// which is the case the insert spelling covers.
		if got, ok := armBitfieldInsert64("bfm", 0xB3407C00); ok {
			t.Errorf("armBitfieldInsert64 = %q,%v; want no insertion", got, ok)
		}
	})

	t.Run("a narrow Thumb encoding in no group", func(t *testing.T) {
		if got, ok := armDecodeThumb16(0xB800, 0); ok {
			t.Errorf("armDecodeThumb16 = %q,%v; want no decode", got, ok)
		}
	})

	t.Run("a status write that keeps its destination field", func(t *testing.T) {
		if got, ok := armStatusRegister32(0xE3200000); ok {
			t.Errorf("armStatusRegister32 = %q,%v; want no decode", got, ok)
		}
	})

	t.Run("a rotated immediate no rotation folds", func(t *testing.T) {
		if _, shortest := armFoldRotatedImmediate(0x101, 0); shortest {
			t.Error("armFoldRotatedImmediate called a value with no encoding the shortest")
		}
	})

	t.Run("a doubleword transfer through the last register", func(t *testing.T) {
		// strd would move the pair beginning at pc, which runs past the bank.
		if got, ok := armExtraLoadStore32(0xE00CF0F0); ok {
			t.Errorf("armExtraLoadStore32 = %q,%v; want no decode", got, ok)
		}
		if got, ok := armExtraLoadStore32(0xE00C60F0); !ok || got != "strd r6, r7, [ip], -r0" {
			t.Errorf("armExtraLoadStore32 = %q,%v; want a pair transfer", got, ok)
		}
	})
}

// TestARM64CompareWithZero covers the floating-point comparisons against zero,
// whose constant Capstone prints as a floating-point one.
func TestARM64CompareWithZero(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"greater than", "33cae05e", "fcmgt d19, d17, #0.0"},
		{"greater or equal", "33cae07e", "fcmge d19, d17, #0.0"},
		{"equal", "33dae05e", "fcmeq d19, d17, #0.0"},
		{"less or equal", "33dae07e", "fcmle d19, d17, #0.0"},
		{"less than", "33eae05e", "fcmlt d19, d17, #0.0"},
		{"and the vector form", "33caa04e", "fcmgt v19.4s, v17.4s, #0.0"},
		{"an operation on one source keeps no constant", "33c2601e", "fabs d19, d17"},
	})
}

// TestARM64LongShifts covers the widening shifts, which Capstone spells in full
// even where the shift is zero and an extend would say the same thing.
func TestARM64LongShifts(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a signed shift of nothing", "e4a4080f", "sshll v4.8h, v7.8b, #0"},
		{"an unsigned one", "e4a4082f", "ushll v4.8h, v7.8b, #0"},
		{"a signed one over the upper half", "e4a4084f", "sshll2 v4.8h, v7.16b, #0"},
		{"an unsigned one over the upper half", "e4a4086f", "ushll2 v4.8h, v7.16b, #0"},
		{"and one that shifts", "e4a4094f", "sshll2 v4.8h, v7.16b, #1"},
	})
}

// TestARM64DuplicateFromRegister covers the element size a duplication from a
// general register names, which is one bit of the selector and no more. The
// duplication from a vector lane reads the same field as an index and takes any
// value.
func TestARM64DuplicateFromRegister(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"bytes", "0f0c014e", "dup v15.16b, w0"},
		{"halfwords", "0f0c024e", "dup v15.8h, w0"},
		{"words", "0f0c044e", "dup v15.4s, w0"},
		{"doublewords", "0f0c084e", "dup v15.2d, x0"},
		{"a selector naming two sizes", "0f0c034e", ""},
		{"another", "0f0c0c4e", ""},
		{"one naming none", "0f0c004e", ""},
		{"and one past the last", "0f0c104e", ""},
		{"a lane duplication takes any selector", "ef040c4e", "dup v15.4s, v7.s[1]"},
	})
}

// TestThumbITBlockRegisterMove covers the register move a shift of nothing
// spells, which keeps its flag-setting form inside an IT block where the move
// of a constant takes the block's condition like anything else.
func TestThumbITBlockRegisterMove(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a register move", "cebf1400", "itee gt\nmovs r4, r2"},
		{"and another", "cebf0200", "itee gt\nmovs r2, r0"},
		{"a constant move takes the condition", "cebf6120", "itee gt\nmovgt r0, #0x61"},
		{"and so does a shift of something", "cebf1408", "itee gt\nlsrgt r4, r2, #0x20"},
	})
}

// TestARM64ExceptionCalls covers the two calls into a higher privilege level
// x/arch declines, alongside the neighbours in their group that it decodes.
func TestARM64ExceptionCalls(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a hypervisor call", "c2af15d4", "hvc #0xad7e"},
		{"a secure monitor call", "c3af15d4", "smc #0xad7e"},
		{"a supervisor call", "c1af15d4", "svc #0xad7e"},
		{"a breakpoint", "c0af35d4", "brk #0xad7e"},
		{"and a halt", "c0af55d4", "hlt #0xad7e"},
		{"the first of the group is unallocated", "c0af15d4", ""},
		{"as is a breakpoint with a link level", "c1af35d4", ""},
	})
}

// TestARMUndefinedInstruction covers the permanently undefined ARM encoding,
// which carries no condition and joins two immediate fields.
func TestARMUndefinedInstruction(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"an undefined instruction", "f393f7e7", "udf #0x7933"},
		{"one of zero", "f000f0e7", "udf #0"},
		{"under any other condition it is not one", "f393f737", ""},
		{"nor in the unconditional space", "f393f7f7", ""},
	})
}

// TestThumbITBlockConditionalBranch covers a branch inside an IT block whose
// own condition ends in the letter the flag-setting forms use, which must not
// be mistaken for one.
func TestThumbITBlockConditionalBranch(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a branch if lower or same", "c2bf2fd9", "ittt gt\nbgt #0x64"},
		{"one if higher", "c2bf2fd8", "ittt gt\nbgt #0x64"},
		{"and a flag-setting operation still loses its letter", "cebf941c", "itee gt\naddgt r4, r2, #2"},
	})
}

// TestARM64SystemRegisterNames covers the system registers Capstone names,
// which it does separately for each direction: a register that cannot be
// written keeps its number there.
func TestARM64SystemRegisterNames(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"the condition flags", "00423bd5", "mrs x0, nzcv"},
		{"a debug register", "400030d5", "mrs x0, osdtrrx_el1"},
		{"a breakpoint value", "800130d5", "mrs x0, dbgbvr1_el1"},
		{"a watchpoint control", "e00d30d5", "mrs x0, dbgwcr13_el1"},
		{"a trace register", "c02031d5", "mrs x0, trcdvcmr0"},
		{"a performance monitor", "009c3bd5", "mrs x0, pmcr_el0"},
		{"an event type counter", "00ec3bd5", "mrs x0, pmevtyper0_el0"},
		{"an implementation-defined one", "00f23fd5", "mrs x0, cpm_ioacc_ctl_el3"},
		{"a write", "00101ed5", "msr sctlr_el3, x0"},
		{"a register with no name keeps its number", "000030d5", "mrs x0, s3_0_c0_c0_0"},
		{"and so does another", "e07930d5", "mrs x0, s3_0_c7_c9_7"},
		{"as does a read-only one written to", "e00118d5", "msr s3_0_c0_c1_7, x0"},
	})
}

// TestARM64ExclusivePairs covers the exclusive pair accesses, whose load may not
// name one register twice. The store may, since it reads both.
func TestARM64ExclusivePairs(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a load naming one register twice", "7d746088", ""},
		{"its acquiring form", "7df46088", ""},
		{"its doubleword form", "7d7460c8", ""},
		{"the zero register twice", "7f7c6088", ""},
		{"and through the stack pointer", "e5176088", ""},
		{"two registers is fine", "7d786088", "ldxp w29, w30, [x3]"},
		{"and one of them may be the base", "63106088", "ldxp w3, w4, [x3]"},
		{"a store may name one twice", "7d742088", "stxp w0, w29, w29, [x3]"},
		{"and so may a doubleword store", "7d7420c8", "stxp w0, x29, x29, [x3]"},
	})
}

// TestARMHints covers the hint space, which names its first few values and
// prints the rest as a numbered hint, with the debug hints at the very top.
func TestARMHints(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"no operation", "00f020e3", "nop"},
		{"a yield", "01f020e3", "yield"},
		{"waiting for an event", "02f020e3", "wfe"},
		{"waiting for an interrupt", "03f020e3", "wfi"},
		{"sending one", "04f020e3", "sev"},
		{"the first unnamed hint", "05f020e3", "hint #5"},
		{"one that reaches hexadecimal", "0ff020e3", "hint #0xf"},
		{"the last of them", "eff020e3", "hint #0xef"},
		{"a hint under a condition", "05f02003", "hinteq #5"},
		{"the first debug hint", "f0f020e3", "dbg #0"},
		{"and the last", "fff020e3", "dbg #0xf"},
	})
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"no operation", "1f2003d5", "nop"},
		{"sending a local event", "bf2003d5", "sevl"},
		{"the first unnamed hint, which is always in hexadecimal", "df2003d5", "hint #0x6"},
		{"one from the next selector", "1f2103d5", "hint #0x8"},
		{"and the last of them", "ff2f03d5", "hint #0x7f"},
	})
	// The wide Thumb hints carry the marker, except the debug ones, which have
	// no narrow encoding to be told apart from.
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"no operation", "aff30080", "nop.w"},
		{"sending an event", "aff30480", "sev.w"},
		{"the first unnamed hint", "aff30580", "hint.w #5"},
		{"one that reaches hexadecimal", "aff30f80", "hint.w #0xf"},
		{"the last of them", "aff3ef80", "hint.w #0xef"},
		{"the first debug hint", "aff3f080", "dbg #0"},
		{"and the last", "aff3ff80", "dbg #0xf"},
	})
}

// TestARMBarriers covers the shareability domain a barrier names, which it does
// for eight of its sixteen values and numbers the rest, always in hexadecimal.
func TestARMBarriers(t *testing.T) {
	runARMSubjectCases(t, arm32, "ARM", []struct{ name, hexBytes, want string }{
		{"the outer shareable store domain", "42f07ff5", "dsb oshst"},
		{"the whole outer shareable domain", "43f07ff5", "dsb osh"},
		{"the non-shareable store domain", "46f07ff5", "dsb nshst"},
		{"the whole non-shareable domain", "47f07ff5", "dsb nsh"},
		{"the inner shareable store domain", "4af07ff5", "dsb ishst"},
		{"the whole inner shareable domain", "4bf07ff5", "dsb ish"},
		{"the full system store domain", "4ef07ff5", "dsb st"},
		{"the whole system", "4ff07ff5", "dsb sy"},
		{"a value naming no domain", "40f07ff5", "dsb #0x0"},
		{"and another", "44f07ff5", "dsb #0x4"},
		{"a memory barrier names them too", "52f07ff5", "dmb oshst"},
		{"and over the whole system", "5ff07ff5", "dmb sy"},
		{"an instruction barrier names only the whole system", "6ff07ff5", "isb sy"},
		{"and numbers the rest", "60f07ff5", "isb #0x0"},
		{"including the ones the others name", "62f07ff5", "isb #0x2"},
		{"a conditional hint keeps its own name", "00f02003", "nopeq"},
		{"as does a conditional debug hint", "f0f02003", "dbgeq #0"},
	})
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"the outer shareable store domain", "bff3428f", "dsb oshst"},
		{"the inner shareable one", "bff34a8f", "dsb ishst"},
		{"the whole system", "bff34f8f", "dsb sy"},
		{"a value naming no domain", "bff3448f", "dsb #0x4"},
		{"a memory barrier", "bff3528f", "dmb oshst"},
		{"and over the whole system", "bff35f8f", "dmb sy"},
		{"an instruction barrier names only the whole system", "bff36f8f", "isb sy"},
		{"and numbers the rest", "bff3628f", "isb #0x2"},
	})
}

// TestThumbWideProcessorState covers the wide change of processor state, which
// shares its encoding with the wide hints and is told from them by the two bits
// naming what it changes.
func TestThumbWideProcessorState(t *testing.T) {
	runARMSubjectCases(t, arm32, "Thumb", []struct{ name, hexBytes, want string }{
		{"a change of mode alone", "aff30081", "cps #0"},
		{"to another mode", "aff30581", "cps #5"},
		{"which ignores the mask bits", "aff32181", "cps #1"},
		{"enabling no interrupt", "aff30084", "cpsie.w none"},
		{"enabling one", "aff34084", "cpsie.w i"},
		{"enabling all of them", "aff3e084", "cpsie.w aif"},
		{"naming a mode as well", "aff30085", "cpsie none, #0"},
		{"with a mask and a mode", "aff32185", "cpsie f, #1"},
		{"disabling one", "aff34086", "cpsid.w i"},
		{"disabling with a mode", "aff32187", "cpsid f, #1"},
		{"a mode with no mask change is refused", "aff30584", ""},
		{"and the reserved change is refused", "aff30082", ""},
	})
}

// TestARM64VectorImmediates covers the two AArch64 operands that do not follow
// the ordinary immediate rule: a vector constant is always in hexadecimal, bar
// zero, and a widening shift states its amount in decimal whatever its size.
func TestARM64VectorImmediates(t *testing.T) {
	runARMSubjectCases(t, arm64, "ARM", []struct{ name, hexBytes, want string }{
		{"a small constant is still in hexadecimal", "6004004f", "movi v0.4s, #0x3"},
		{"and so is its inverted form", "6004006f", "mvni v0.4s, #0x3"},
		{"zero alone carries no prefix", "0004004f", "movi v0.4s, #0"},
		{"a byte constant", "e0e7074f", "movi v0.16b, #0xff"},
		{"a halfword one", "4085006f", "mvni v0.8h, #0xa"},
		{"a doubleword one", "a0e6022f", "movi d0, #0xff00ff00ff00ff"},
		{"a shift of eight", "2038212e", "shll v0.8h, v1.8b, #8"},
		{"one of sixteen", "2038612e", "shll v0.4s, v1.4h, #16"},
		{"one of thirty-two", "2038a12e", "shll v0.2d, v1.2s, #32"},
		{"and over the upper half", "2038616e", "shll2 v0.4s, v1.8h, #16"},
	})
}

// TestARMSpellingGuards exercises the guards the AArch64 spelling rewrites and
// the wide Thumb group apply to operand shapes no ordinary encoding produces.
func TestARMSpellingGuards(t *testing.T) {
	t.Run("a constant with no immediate to rewrite", func(t *testing.T) {
		for _, operands := range []string{"v0.4s, v1.4s", "v0.4s, #"} {
			if m, o, ok := armSpeltInFull64("movi", operands); ok {
				t.Errorf("armSpeltInFull64(movi, %q) = %q %q; want no rewrite", operands, m, o)
			}
		}
	})

	t.Run("a hint with no immediate", func(t *testing.T) {
		if m, o, ok := armSpeltInFull64("hint", "#"); ok {
			t.Errorf("armSpeltInFull64(hint, #) = %q %q; want no rewrite", m, o)
		}
	})

	t.Run("a wide processor state outside its own group", func(t *testing.T) {
		if text, ok := armThumb32ChangeState(0x9000); ok {
			t.Errorf("armThumb32ChangeState = %q; want no decode", text)
		}
		// The reserved change of the masks, which names no operation.
		if text, ok := armThumb32ChangeState(0x8300); ok {
			t.Errorf("armThumb32ChangeState = %q; want no decode", text)
		}
	})

	t.Run("a barrier outside the three kinds", func(t *testing.T) {
		for _, second := range []uint32{0x8F0F, 0x8F7F} {
			if text, ok := armThumb32HintOrBarrier(0xF3BF, second); ok {
				t.Errorf("armThumb32HintOrBarrier(%#x) = %q; want no decode", second, text)
			}
		}
	})
}
