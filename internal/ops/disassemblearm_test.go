package ops

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
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

	type goldenCase struct {
		Hex    string `json:"hex"`
		Arch   string `json:"arch"`
		Mode   string `json:"mode"`
		Endian string `json:"endian"`
		Want   string `json:"want"`
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
		got := armRewriteOperands("r0, #, r1", armImmediate32, armImmediate32)
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
		{"an unallocated second-group encoding", func() (string, bool) {
			return armThumb32LoadMultipleOrDataRegister(0xEC00, 0x0000) // the coprocessor space
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
		{"a bitfield that ends before it begins", func() (string, bool) {
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
		{"the stack pointer as a shifted index", func() (string, bool) {
			return armThumb32Memory(0xF852, 0x300D)
		}},
		{"the program counter as a shifted index", func() (string, bool) {
			return armThumb32Memory(0xF852, 0x300F)
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
		{"an unallocated multiply", func() (string, bool) {
			return armThumb32Multiply(0xFB11, 0xF302)
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
