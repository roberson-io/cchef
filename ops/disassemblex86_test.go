package ops

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/roberson-io/cchef/core"
)

// x86Recipe builds a Disassemble x86 recipe. CyberChef has no fixture file for
// this operation, so every expectation here was taken from the CyberChef-server
// oracle running the real vendored disassembler.
func x86Recipe(mode, compat string, codeSegment, offset float64, showHex, showPos bool) core.Recipe {
	return core.Recipe{{
		Op:   "Disassemble x86",
		Args: []any{mode, compat, codeSegment, offset, showHex, showPos},
	}}
}

// x86Default is the recipe with CyberChef's default arguments.
func x86Default(hexInput string) opCase {
	return opCase{"", hexInput, "", x86Recipe("64", "Full x86 architecture", 16, 0, true, true)}
}

func TestDisassembleX86Basics(t *testing.T) {
	runCases(t, []opCase{
		{
			"64-bit function prologue", "554889e54883ec20",
			"0000000000000000 55                              PUSH RBP\r\n" +
				"0000000000000001 4889E5                          MOV RBP,RSP\r\n" +
				"0000000000000004 4883EC20                        SUB RSP,0000000000000020\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"32-bit mode uses 8-digit addresses", "5589e583ec10",
			"00000000 55                              PUSH EBP\r\n" +
				"00000001 89E5                            MOV EBP,ESP\r\n" +
				"00000003 83EC10                          SUB ESP,00000010\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"16-bit mode uses segment:offset", "b80100cd21",
			"0016:0000 B80100                          MOV AX,0001\r\n" +
				"0016:0003 CD21                            INT 21\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"whitespace is stripped from the input", "48 31 c0\nc3",
			"0000000000000000 4831C0                          XOR RAX,RAX\r\n" +
				"0000000000000003 C3                              RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// The two display toggles are independent, and the hex column is padded to a
// fixed width only when it is shown.
func TestDisassembleX86DisplayToggles(t *testing.T) {
	const input = "4831c0c3"
	runCases(t, []opCase{
		{
			"position and hex", input,
			"0000000000000000 4831C0                          XOR RAX,RAX\r\n" +
				"0000000000000003 C3                              RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"position only", input,
			"0000000000000000 XOR RAX,RAX\r\n0000000000000003 RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, false, true),
		},
		{
			"hex only", input,
			"4831C0                          XOR RAX,RAX\r\nC3                              RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, false),
		},
		{
			"neither", input,
			"XOR RAX,RAX\r\nRET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, false, false),
		},
	})
}

// CyberChef passes the code segment and offset to the engine as a "seg:offset"
// string, which parses both as hexadecimal and keeps only the last four digits
// of a short offset. Both quirks are visible in the addresses below.
func TestDisassembleX86BasePosition(t *testing.T) {
	runCases(t, []opCase{
		{
			"offset is read as hex, not decimal", "4831c0c3",
			"0000000000004096 4831C0                          XOR RAX,RAX\r\n" +
				"0000000000004099 C3                              RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 4096, true, true),
		},
		{
			"32-bit with a high code segment switches to seg:offset", "5589e5",
			"4096:0006 55                              PUSH EBP\r\n" +
				"4096:0007 89E5                            MOV EBP,ESP\r\n",
			x86Recipe("32", "Full x86 architecture", 4096, 256, true, true),
		},
	})
}

// Addresses embedded in operands are resolved against the instruction pointer
// rather than printed as relative displacements.
func TestDisassembleX86ResolvesAddresses(t *testing.T) {
	runCases(t, []opCase{
		{
			"RIP-relative LEA", "488d0d05000000",
			"0000000000000000 488D0D05000000                  LEA RCX,[000000000000000C]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"CALL target", "e800000000",
			"0000000000000000 E800000000                      CALL 0000000000000005\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"SIB with scale and displacement", "488b04cd10000000",
			"0000000000000000 488B04CD10000000                MOV RAX,QWORD PTR [RCX*8+00000010]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"segment override", "648b042528000000",
			"0000000000000000 648B042528000000                MOV EAX,DWORD PTR FS:[0000000000000028]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"indirect JMP through an absolute address", "ff2425efbeadde",
			"0000000000000000 FF2425EFBEADDE                  JMP QWORD PTR [FFFFFFFFDEADBEEF]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// The engine covers the vector extensions, group opcodes and the x87 stack.
func TestDisassembleX86Encodings(t *testing.T) {
	runCases(t, []opCase{
		{
			"EVEX-encoded AVX-512", "62f17c48280424",
			"0000000000000000 62F17C48280424                  VMOVAPS ZMM0,ZMMWORD PTR [RSP]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"VEX-encoded", "c5f877",
			"0000000000000000 C5F877                          VZEROUPPER\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"x87 stack registers", "d9c0dde1",
			"0000000000000000 D9C0                            FLD ST(0)\r\n" +
				"0000000000000002 DDE1                            FUCOM ST(1)\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"group opcodes select on the ModR/M reg field", "83c001ff0083f801",
			"0000000000000000 83C001                          ADD EAX,00000001\r\n" +
				"0000000000000003 FF00                            INC DWORD PTR [RAX]\r\n" +
				"0000000000000005 83F801                          CMP EAX,00000001\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"repeat prefixes are printed before the instruction", "f3a4f2ae",
			"0000000000000000 F3A4                            REP  MOVS BYTE PTR [RDI],BYTE PTR [RSI]\r\n" +
				"0000000000000002 F2AE                            REPNE  SCAS AL,BYTE PTR [RDI]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// The compatibility settings remap parts of the opcode tables. Cyrix and
// X86/486 reassign 0F A6/A7, and X86/486 also turns the 0F 1x SSE moves into
// the older UMOV.
func TestDisassembleX86CompatibilityModes(t *testing.T) {
	const sse = "0f110f12"
	const legacy = "0f110f120fa60fa7"
	sseOutput := "0000000000000000 0F110F                          MOVUPS XMMWORD PTR [RDI],XMM1\r\n" +
		"0000000000000003 12                              ADC AL,BYTE PTR [RAX]\r\n"

	cases := []opCase{
		{
			"Cyrix maps 0F A6/A7 to CMPS and IBTS", legacy,
			"0000000000000000 0F110F                          MOVUPS XMMWORD PTR [RDI],XMM1\r\n" +
				"0000000000000003 120F                            ADC CL,BYTE PTR [RDI]\r\n" +
				"0000000000000005 A6                              CMPS BYTE PTR [RDI],BYTE PTR [RSI]\r\n" +
				"0000000000000006 0FA7                            IBTS DWORD PTR [RAX],EAX\r\n",
			x86Recipe("64", "Cyrix", 16, 0, true, true),
		},
		{
			"X86/486 uses UMOV and CMPXCHG", legacy,
			"0000000000000000 0F110F                          UMOV DWORD PTR [RDI],ECX\r\n" +
				"0000000000000003 120F                            ADC CL,BYTE PTR [RDI]\r\n" +
				"0000000000000005 A6                              CMPS BYTE PTR [RDI],BYTE PTR [RSI]\r\n" +
				"0000000000000006 0FA7                            CMPXCHG\r\n",
			x86Recipe("64", "X86/486", 16, 0, true, true),
		},
	}
	for _, mode := range []string{"Knights Corner", "Larrabee", "Geode", "Centaur"} {
		cases = append(cases, opCase{
			mode + " leaves the SSE moves alone", sse, sseOutput,
			x86Recipe("64", mode, 16, 0, true, true),
		})
	}
	runCases(t, cases)
}

// Input the engine cannot use is dropped rather than reported: invalid hex
// yields no output at all, and a trailing half-byte is discarded.
func TestDisassembleX86UnusableInput(t *testing.T) {
	runCases(t, []opCase{
		{
			"invalid hex produces no output", "zzzz", "",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"empty input produces no output", "", "",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"a trailing half-byte is dropped", "4831c",
			"0000000000000000 4831                            XOR QWORD PTR [RAX],RAX\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"a truncated instruction reads zero bytes past the end", "488b",
			"0000000000000000 488B                            MOV RAX,QWORD PTR [RAX]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// Some mnemonics carry their condition code in the middle, spelled in the
// tables with two commas ("VPCOM,UB,"). Both commas have to disappear whether
// or not a condition code is spliced in.
func TestDisassembleX86ConditionCodeMnemonics(t *testing.T) {
	runCases(t, []opCase{
		{
			"condition code spliced into the mnemonic", "8fe85cecc000",
			"0000000000000000 8FE85CECC000                    VPCOMLTUB XMM0,XMM4,XMM0\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"immediate too large stays an operand", "8fe85cecc009",
			"0000000000000000 8FE85CECC009                    VPCOMUB XMM0,XMM4,XMM0,09\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"an immediate read past the end", "8fe85cec9b6b",
			"0016:0000 8FE85CEC9B6B                    VPCOMUB XMM3,XMM4,XMMWORD PTR [BP+DI+0NAN],NAN\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// An operand whose register lookup falls off the end of the table prints as
// nothing, not as the word "undefined" that the same value produces when it is
// concatenated onto an address.
func TestDisassembleX86UndefinedOperand(t *testing.T) {
	runCases(t, []opCase{
		{
			"register index outside the table", "2e4f0fd9",
			"0000000000004096 2E4F0FD9                        PSUBUSW ,MMWORD PTR CS:[R8]\r\n",
			x86Recipe("64", "Full x86 architecture", 4096, 4096, true, true),
		},
	})
}

// CyberChef's engine predates ENDBR64 and decodes F3 0F 1E FA as a repeat
// prefix on an unknown opcode. cchef reproduces the same output.
func TestDisassembleX86UnknownOpcode(t *testing.T) {
	runCases(t, []opCase{
		{
			"ENDBR64 is not recognised", "f30f1efa",
			"0000000000000000 F30F1E                          REP  ???\r\n" +
				"0000000000000003 FA                              CLI\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// The bit mode option is constrained by the engine, so the guard is exercised
// directly.
func TestDisassembleX86InvalidBitMode(t *testing.T) {
	op := DisassembleX86{}
	_, err := op.Run(core.NewDish([]byte("90"), core.TypeString),
		[]any{"128", "Full x86 architecture", 16.0, 0.0, true, true})
	if err == nil || !strings.Contains(err.Error(), "Invalid mode value") {
		t.Errorf("error = %v, want CyberChef's \"Invalid mode value\"", err)
	}
}

// Decoding must not depend on state left behind by an earlier run.
func TestDisassembleX86RunsAreIndependent(t *testing.T) {
	first := x86Default("554889e5")
	if _, err := first.recipe.Execute(core.NewDish([]byte("b80100cd21"), core.TypeString)); err != nil {
		t.Fatalf("priming run: %v", err)
	}
	out, err := first.recipe.Execute(core.NewDish([]byte("4831c0c3"), core.TypeString))
	if err != nil {
		t.Fatal(err)
	}
	want := "0000000000000000 4831C0                          XOR RAX,RAX\r\n" +
		"0000000000000003 C3                              RET\r\n"
	if out.String() != want {
		t.Errorf("got %q\nwant %q", out.String(), want)
	}
}

// The engine has a long tail of encodings the common cases never reach:
// three-byte vector prefixes, the Larrabee opcode maps, 3DNow!, moffs and
// string addressing, HLE and branch-hint prefixes, and the 15-byte length
// limit. Every expectation below came from the oracle.
func TestDisassembleX86Corpus(t *testing.T) {
	runCases(t, []opCase{
		{
			"VEX3 three-byte prefix", "c4e27d1806",
			"0000000000000000 C4E27D1806                      VBROADCASTSS YMM0,DWORD PTR [RSI]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"VEX3 with W bit set", "c4e3f94a0110",
			"0000000000000000 C4E3F94A0110                    VBLENDVPS XMM0,XMM0,XMMWORD PTR [RCX],XMM1\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"XOP prefix", "8fe8c801c0",
			"0000000000000000 8FE8C801                        ???\r\n0000000000000004 C0                              ROL BYTE PTR [RAX],NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"L1OM vector prefix", "d6138620",
			"0000000000000000 D6138620                        VCMPNEQPS K0{K6},V0,DWORD PTR [R8]{1To16}\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"L1OM up conversion", "d6532004",
			"0000000000000000 D6532004                        DELAY V4\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"L1OM swizzle", "d6009a10",
			"0000000000000000 D6009A10                        VLOADD V0{K2},[RAX]{UNorm2D}\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"L1OM short form under Larrabee", "6210c0",
			"0000000000000000 6210C0                          BSFI R8D,R8D\r\n",
			x86Recipe("64", "Larrabee", 16, 0, true, true),
		},
		{
			"L1OM short form, 16-bit", "6230d9",
			"0016:0000 6230D9                          BSFI BX,R9W\r\n",
			x86Recipe("16", "Larrabee", 16, 0, true, true),
		},
		{
			"3DNow! trailing opcode byte", "0f0fc00d",
			"0000000000000000 0F0FC00D                        PI2FD MM0,MM0\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"3DNow! unknown trailing byte", "0f0fc0ff",
			"0000000000000000 0F0FC0FF                        ???\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"EVEX broadcast round", "62f17d581004",
			"0000000000000000 62F17D5810                      ???\r\n0000000000000005 04                              ADD AL,NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"EVEX with mask register", "62f17d4b1004",
			"0000000000000000 62F17D4B10                      ???\r\n0000000000000005 04                              ADD AL,NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"EVEX with zero merge", "62f17dcb1004",
			"0000000000000000 62F17DCB10                      ???\r\n0000000000000005 04                              ADD AL,NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"EVEX reserved bits set", "62f47d481004",
			"0000000000000000 62F47D4810                      ???\r\n0000000000000005 04                              ADD AL,NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"MVEX conversion mode", "62f17d181004",
			"0000000000000000 62F17D1810                      ???\r\n0000000000000005 04                              ADD AL,NAN\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"moffs direct address", "a08877665544332211",
			"0000000000000000 A08877665544332211              MOV AL,BYTE PTR [1122334455667788]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"moffs 32-bit", "a144332211",
			"00000000 A144332211                      MOV EAX,DWORD PTR [11223344]\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"moffs 16-bit", "a14433",
			"0016:0000 A14433                          MOV AX,WORD PTR [3344]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"string source and destination", "a4a5a6a7",
			"0000000000000000 A4                              MOVS BYTE PTR [RDI],BYTE PTR [RSI]\r\n0000000000000001 A5                              MOVS DWORD PTR [RDI],DWORD PTR [RSI]\r\n0000000000000002 A6                              CMPS BYTE PTR [RDI],BYTE PTR [RSI]\r\n0000000000000003 A7                              CMPS DWORD PTR [RDI],DWORD PTR [RSI]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"XLAT uses the RBX pointer", "d7",
			"0000000000000000 D7                              XLAT BYTE PTR [RBX]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"XLAT with address override", "67d7",
			"0000000000000000 67D7                            XLAT BYTE PTR [EBX]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"x87 explicit ST operand", "d8c1dcc1",
			"0000000000000000 D8C1                            FADD ST,ST(1)\r\n0000000000000002 DCC1                            FADD ST(1),ST\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"explicit constant operands", "d0e0d1e0",
			"0000000000000000 D0E0                            SHL AL,1\r\n0000000000000002 D1E0                            SHL EAX,1\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"segment registers FS and GS", "0fa00fa8",
			"0000000000000000 0FA0                            PUSH FS\r\n0000000000000002 0FA8                            PUSH GS\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"over-long instruction", "666666666666666666666666666666664831c0",
			"0000000000000000 666666666666666666666666666666  ???\r\n000000000000000F 664831C0                        XOR RAX,RAX\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"over-long run of prefixes", "2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2e2ec3",
			"0000000000000000 2E2E2E2E2E2E2E2E2E2E2E2E2E2E2E  ???\r\n000000000000000F 2E2EC3                          RET\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"HLE XRELEASE", "f0f3870424",
			"0000000000000000 F0F3870424                      LOCK XRELEASE XCHG EAX,DWORD PTR [RSP]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"HLE XACQUIRE", "f0f2870424",
			"0000000000000000 F0F2870424                      LOCK XACQUIRE XCHG EAX,DWORD PTR [RSP]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"HLE flip with lock last", "f3f0870424",
			"0000000000000000 F3F0870424                      XRELEASE LOCK XCHG EAX,DWORD PTR [RSP]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"MPX bound prefix", "f2e800000000",
			"0000000000000000 F2E800000000                    BND  CALL 0000000000000006\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"branch taken hint", "3e0f8400000000",
			"0000000000000000 3E0F8400000000                  HT  JE 0000000000000007\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"branch not taken hint", "2e0f8400000000",
			"0000000000000000 2E0F8400000000                  HNT  JE 0000000000000007\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"16-bit BX+SI addressing", "8b00",
			"0016:0000 8B00                            MOV AX,WORD PTR [BX+SI]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"16-bit BP+DI with disp8", "8b4b10",
			"0016:0000 8B4B10                          MOV CX,WORD PTR [BP+DI+10]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"16-bit direct disp16", "8b0e3412",
			"0016:0000 8B0E3412                        MOV CX,WORD PTR [1234]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"16-bit BX only", "8b07",
			"0016:0000 8B07                            MOV AX,WORD PTR [BX]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"address override in 16-bit", "678b00",
			"0016:0000 678B00                          MOV AX,WORD PTR [EAX]\r\n",
			x86Recipe("16", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"address override in 32-bit", "678b00",
			"00000000 678B00                          MOV EAX,DWORD PTR [BX+SI]\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"SIB with no base", "8b04a5f0debc9a",
			"00000000 8B04A5F0DEBC9A                  MOV EAX,DWORD PTR [9ABCDEF0]\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"SIB with base and index", "8b448800",
			"00000000 8B448800                        MOV EAX,DWORD PTR [EAX+ECX*4+00]\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"far pointer call", "9a1122334455",
			"00000000 9A1122334455                    CALL 0NAN:44332211\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"far pointer jump", "ea1122334455",
			"00000000 EA1122334455                    JMP 0NAN:44332211\r\n",
			x86Recipe("32", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"group opcode static form", "0f01c1",
			"0000000000000000 0F01C1                          VMCALL\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"group opcode with ModR/M", "0f01d0",
			"0000000000000000 0F01D0                          XGETBV\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"Knights Corner vector", "62f17c48280424",
			"0000000000000000 62F17C48280424                  VMOVAPS ZMM0,ZMMWORD PTR [RSP]\r\n",
			x86Recipe("64", "Knights Corner", 16, 0, true, true),
		},
		{
			"Centaur alternate opcode", "0fa60fa7",
			"0000000000000000 0FA6                            ???\r\n0000000000000002 0FA7                            ???\r\n",
			x86Recipe("64", "Centaur", 16, 0, true, true),
		},
		{
			"Geode alternate opcode", "0f0d000f0e",
			"0000000000000000 0F0D00                          PREFETCH [RAX]\r\n0000000000000003 0F0E                            FEMMS\r\n",
			x86Recipe("64", "Geode", 16, 0, true, true),
		},
		{
			"condition code from immediate", "8fe85cecc002",
			"0000000000000000 8FE85CECC002                    VPCOMGTUB XMM0,XMM4,XMM0\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"VSIB gather", "c4e2794000",
			"0000000000000000 C4E2794000                      VPMULLD XMM0,XMM0,XMMWORD PTR [RAX]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"EVEX VSIB", "62f27d48900424",
			"0000000000000000 62F27D48900424                  VPGATHERDD ZMM0,DWORD PTR [RSP+ZMM4]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"synthetic VM opcode", "0fc7c80000",
			"0000000000000000 0FC7C80000                      VMGETINFO\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"synthetic VM opcode from the 5x5 map", "0fc7c80102",
			"0000000000000000 0FC7C80102                      VMSPLAF\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"synthetic VM opcode out of range", "0fc7c80500",
			"0000000000000000 0FC7C80500                      ???\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"synthetic VM opcode with no operation", "0fc7c80404",
			"0000000000000000 0FC7C80404                      ???\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"Larrabee condition code from the vector register", "d6000021",
			"0000000000000000 D6000021                        VCMPEQPI K0,V0,[RAX]\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
		{
			"Larrabee condition code with a broadcast", "d6100021",
			"0000000000000000 D6100021                        VCMPEQPI K0,V0,DWORD PTR [RAX]{1To16}\r\n",
			x86Recipe("64", "Full x86 architecture", 16, 0, true, true),
		},
	})
}

// The JavaScript shims the engine relies on have defensive branches the decoder
// itself rarely reaches, so they are exercised directly.
func TestX86JSShims(t *testing.T) {
	t.Run("jsParseHex", func(t *testing.T) {
		for _, c := range []struct {
			in   string
			want float64
		}{
			{"ff", 255},
			{"  1a", 26},
			{"-10", -16},
			{"+10", 16},
			{"12zz", 18}, // stops at the first non-hex digit
			{"", math.NaN()},
			{"zz", math.NaN()},
			{"-", math.NaN()},
			{strings.Repeat("f", 20), 1.2089258196146292e+24}, // wider than 64 bits
		} {
			got := jsParseHex(c.in)
			if math.IsNaN(c.want) {
				if !math.IsNaN(got) {
					t.Errorf("jsParseHex(%q) = %v, want NaN", c.in, got)
				}
				continue
			}
			if got != c.want {
				t.Errorf("jsParseHex(%q) = %v, want %v", c.in, got, c.want)
			}
		}
		if got := jsParseHex("-" + strings.Repeat("f", 20)); got >= 0 {
			t.Errorf("jsParseHex of a wide negative = %v, want a negative value", got)
		}
	})

	t.Run("jsHex16", func(t *testing.T) {
		for _, c := range []struct {
			in   float64
			want string
		}{
			{255, "ff"},
			{0, "0"},
			{-16, "-10"},
			{math.NaN(), "NaN"},
		} {
			if got := jsHex16(c.in); got != c.want {
				t.Errorf("jsHex16(%v) = %q, want %q", c.in, got, c.want)
			}
		}
	})

	t.Run("jsSliceFrom", func(t *testing.T) {
		for _, c := range []struct {
			s     string
			start int
			want  string
		}{
			{"abcdef", 2, "cdef"},
			{"abcdef", -2, "ef"},
			{"abc", -10, "abc"}, // clamped to the start
			{"abc", 10, ""},     // past the end
		} {
			if got := jsSliceFrom(c.s, c.start); got != c.want {
				t.Errorf("jsSliceFrom(%q,%d) = %q, want %q", c.s, c.start, got, c.want)
			}
		}
	})

	t.Run("jsBool and jsShl wrap like JavaScript", func(t *testing.T) {
		if jsBool(true) != 1 || jsBool(false) != 0 {
			t.Error("jsBool should yield 1 and 0")
		}
		// A negative shift count wraps to 31, as JavaScript masks it to five bits.
		if got := jsShl(1, -1); got != -2147483648 {
			t.Errorf("jsShl(1,-1) = %v, want -2147483648", got)
		}
		if got := jsShr(math.NaN(), 4); got != 0 {
			t.Errorf("jsShr(NaN,4) = %v, want 0", got)
		}
	})

	t.Run("indexOfString", func(t *testing.T) {
		list := []string{"a", "b"}
		if got := indexOfString(list, "b"); got != 1 {
			t.Errorf("indexOfString = %d, want 1", got)
		}
		if got := indexOfString(list, "absent"); got != 0 {
			t.Errorf("indexOfString of an absent value = %d, want 0", got)
		}
	})

	t.Run("x86StripWhitespace", func(t *testing.T) {
		in := "48 31\t\n\v\f\r c0\u00a0\u2028\u2029\u3000\ufeffc3 "
		if got := x86StripWhitespace(in); got != "4831c0c3" {
			t.Errorf("x86StripWhitespace = %q, want %q", got, "4831c0c3")
		}
		if got := x86StripWhitespace("4x8"); got != "4x8" {
			t.Errorf("x86StripWhitespace should leave non-space runes alone, got %q", got)
		}
	})
}

// Out-of-range table lookups must produce JavaScript's undefined rather than
// panicking, since the engine indexes its tables with computed values.
func TestX86TableLookups(t *testing.T) {
	d := newX86Decoder(x86Mode64, 0)

	t.Run("out of range yields undefined", func(t *testing.T) {
		if got := x86At([]string{"a"}, 9); got != x86UndefinedValue {
			t.Errorf("x86At past the end = %q", got)
		}
		if got := x86At([]string{"a"}, -1); got != x86UndefinedValue {
			t.Errorf("x86At before the start = %q", got)
		}
		if got := d.reg(9999, 0); got != x86UndefinedValue {
			t.Errorf("reg with an unknown size = %q", got)
		}
		if got := d.reg(1, 9999); got != x86UndefinedValue {
			t.Errorf("reg with an unknown register = %q", got)
		}
		if got := d.reg8(99, 0); got != x86UndefinedValue {
			t.Errorf("reg8 with an unknown REX state = %q", got)
		}
		if got := d.mnemonic(99999); !got.isEmptyLeaf() {
			t.Errorf("mnemonic past the end = %+v, want an empty leaf", got)
		}
		if got := d.operandEntry(99999); !got.isEmptyLeaf() {
			t.Errorf("operandEntry past the end = %+v, want an empty leaf", got)
		}
	})

	t.Run("node indexing", func(t *testing.T) {
		leaf := x86Leaf("MOV")
		if leaf.length() != -1 || leaf.str != "MOV" {
			t.Errorf("x86Leaf = %+v", leaf)
		}
		if got := leaf.at(0); !got.isEmptyLeaf() {
			t.Errorf("indexing a leaf = %+v, want an empty leaf", got)
		}
		if got := leaf.strAt(0); got != x86UndefinedValue {
			t.Errorf("strAt on a leaf = %q", got)
		}

		var list x86Node
		if err := list.UnmarshalJSON([]byte(`["A","B"]`)); err != nil {
			t.Fatal(err)
		}
		if list.length() != 2 || list.at(1).str != "B" {
			t.Errorf("list node = %+v", list)
		}
		if got := list.strAt(5); got != x86UndefinedValue {
			t.Errorf("strAt past the end = %q", got)
		}

		var null x86Node
		if err := null.UnmarshalJSON([]byte("null")); err != nil {
			t.Fatal(err)
		}
		if !null.isEmptyLeaf() {
			t.Errorf("null node = %+v, want an empty leaf", null)
		}
	})

	t.Run("x86LookupOrInvalid", func(t *testing.T) {
		list := []string{"A", ""}
		if v, ok := x86LookupOrInvalid(list, 0); !ok || v != "A" {
			t.Errorf("lookup = %q,%v", v, ok)
		}
		for _, i := range []float64{1, 9, -1, math.NaN()} {
			if _, ok := x86LookupOrInvalid(list, i); ok {
				t.Errorf("lookup(%v) reported a hit", i)
			}
		}
	})

	t.Run("applyX86Diff ignores unusable keys", func(t *testing.T) {
		table := []x86Node{x86Leaf("A"), x86Leaf("B")}
		out := applyX86Diff(table, map[string]x86Node{
			"1":       x86Leaf("PATCHED"),
			"notanum": x86Leaf("X"),
			"99":      x86Leaf("X"),
			"-1":      x86Leaf("X"),
		})
		if out[0].str != "A" || out[1].str != "PATCHED" {
			t.Errorf("applyX86Diff = %+v", out)
		}
		if same := applyX86Diff(table, nil); &same[0] != &table[0] {
			t.Error("an empty diff should share the original table")
		}
	})
}

// An operand slot that never received a value still accepts the EVEX mask and
// zero-merge suffixes, which JavaScript appends to undefined.
func TestX86OperandListUndefinedSlot(t *testing.T) {
	var list x86OperandList
	list.appendTo(0, "{K1}")
	if got := list.join(); got != "undefined{K1}" {
		t.Errorf("join = %q, want %q", got, "undefined{K1}")
	}

	var bare x86OperandList
	bare.put(1, "RAX")
	if got := bare.join(); got != ",RAX" {
		t.Errorf("join with a hole = %q, want %q", got, ",RAX")
	}
}

// TestDisassembleX86Golden replays a broad corpus captured from the oracle:
// real instruction sequences, random bytes, prefix soup and the vector opcode
// maps, across all three bit modes and every compatibility setting. It is what
// pins the long tail of the engine that the readable cases above do not reach.
func TestDisassembleX86Golden(t *testing.T) {
	f, err := os.Open("testdata/disassemble_x86.jsonl")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer func() { _ = f.Close() }()

	type goldenCase struct {
		Hex    string `json:"hex"`
		Mode   string `json:"mode"`
		Compat string `json:"compat"`
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

		recipe := x86Recipe(c.Mode, c.Compat, 16, 0, true, true)
		out, err := recipe.Execute(core.NewDish([]byte(c.Hex), core.TypeString))
		if err != nil {
			t.Fatalf("%s (%s, %s): %v", c.Hex, c.Mode, c.Compat, err)
		}
		if out.String() != c.Want {
			t.Errorf("%s (%s, %s):\n got %q\nwant %q", c.Hex, c.Mode, c.Compat, out.String(), c.Want)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	if n == 0 {
		t.Fatal("corpus is empty")
	}
}

// The address bookkeeping has branches the decoder only reaches at the extremes
// of the 64-bit address space or when an instruction is rewound, so they are
// driven directly.
func TestX86AddressBookkeeping(t *testing.T) {
	t.Run("a full-width offset sets both halves", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.setBasePosition("16:123456789ABCDEF0")
		if d.pos64 != 0x12345678 || d.pos32 != 0x9ABCDEF0 {
			t.Errorf("pos64=%X pos32=%X, want 12345678 9ABCDEF0", int64(d.pos64), int64(d.pos32))
		}
		if got := d.getPosition(); got != "123456789ABCDEF0" {
			t.Errorf("getPosition = %q", got)
		}
	})

	t.Run("a negative offset wraps into unsigned range", func(t *testing.T) {
		d := newX86Decoder(x86Mode32, 0)
		d.setBasePosition("16:FFFFFFFF")
		if d.pos32 != 0xFFFFFFFF {
			t.Errorf("pos32 = %v, want 4294967295", d.pos32)
		}
	})

	t.Run("the address counter carries into the high half", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.loadBinCode("9090")
		d.pos32 = 0xFFFFFFFF
		d.nextByte()
		if d.pos32 != 0 || d.pos64 != 1 {
			t.Errorf("pos32=%v pos64=%v, want 0 and 1", d.pos32, d.pos64)
		}
		d.pos32, d.pos64 = 0xFFFFFFFF, 0xFFFFFFFF
		d.nextByte()
		if d.pos32 != 0 || d.pos64 != 0 {
			t.Errorf("pos32=%v pos64=%v, want both 0 after wrapping", d.pos32, d.pos64)
		}
	})

	t.Run("nextByte stops at the end of the buffer", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.loadBinCode("90")
		d.nextByte()
		before := d.codePos
		d.nextByte()
		if d.codePos != before || d.instructionHex != "90" {
			t.Errorf("codePos=%d hex=%q, want no movement past the end", d.codePos, d.instructionHex)
		}
	})

	t.Run("gotoPosition rejects an address outside the buffer", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.loadBinCode("90909090")
		d.codePos = 2
		d.pos32 = 2
		if d.gotoPosition("00000000FFFFFFFF") {
			t.Error("expected an out-of-bounds address to be rejected")
		}
		if d.codePos != 2 {
			t.Errorf("codePos = %d, want the position restored to 2", d.codePos)
		}
		if !d.gotoPosition("0000000000000001") {
			t.Error("expected an in-range address to be accepted")
		}
		if d.codePos != 1 {
			t.Errorf("codePos = %d, want 1", d.codePos)
		}
	})

	t.Run("gotoPosition accounts for the code segment in 16-bit mode", func(t *testing.T) {
		d := newX86Decoder(x86Mode16, 0)
		d.loadBinCode("9090909090909090")
		d.setBasePosition("16:0000")
		d.codePos = 4
		d.pos32 = 4
		if !d.gotoPosition("0016:0002") {
			t.Error("expected the segmented address to be accepted")
		}
		if d.codePos != 2 {
			t.Errorf("codePos = %d, want 2", d.codePos)
		}
	})

	t.Run("loadBinCode reports invalid hex", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		if d.loadBinCode("zz") {
			t.Error("expected invalid hex to be rejected")
		}
		if len(d.binCode) != 0 {
			t.Errorf("binCode = %v, want nothing loaded", d.binCode)
		}
		if !d.loadBinCode("4831c0") {
			t.Error("expected valid hex to load")
		}
		if len(d.binCode) != 3 {
			t.Errorf("binCode = %v, want three bytes", d.binCode)
		}
	})

	t.Run("reg8 with no register table", func(t *testing.T) {
		empty := &x86Decoder{tables: &x86TableSet{}}
		if got := empty.reg8(0, 0); got != x86UndefinedValue {
			t.Errorf("reg8 = %q, want undefined", got)
		}
	})
}

// The vendored tables ship with the binary, so the loader's failure paths are
// exercised against synthetic filesystems rather than the real one.
func TestLoadX86Tables(t *testing.T) {
	if _, err := loadX86Tables(fstest.MapFS{}); err == nil {
		t.Error("expected an error when the tables are missing")
	}
	broken := fstest.MapFS{"x86tables/tables.json": &fstest.MapFile{Data: []byte("{not json")}}
	if _, err := loadX86Tables(broken); err == nil {
		t.Error("expected an error when the tables cannot be parsed")
	}
	if _, err := loadX86Tables(x86TableFS); err != nil {
		t.Errorf("the vendored tables should load: %v", err)
	}
}

// A build without usable tables cannot decode anything, so the loader panics.
func TestMustLoadX86TablesPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic when the tables are missing")
		}
	}()
	mustLoadX86Tables(fstest.MapFS{})
}

// Arithmetic the decoder only performs at the extremes of the address space and
// the operand-size attribute.
func TestX86NumericEdges(t *testing.T) {
	t.Run("a base position that wraps negative", func(t *testing.T) {
		d := newX86Decoder(x86Mode32, 0)
		d.pos32 = 0x80000000 // the high bit survives a short offset
		d.setBasePosition("16:1")
		if d.pos32 != 0x80000001 {
			t.Errorf("pos32 = %v, want 2147483649", d.pos32)
		}
	})

	t.Run("an empty size attribute", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		// A zero attribute has no highest bit, so the engine marks it -1 and the
		// subtraction wraps; the default selector then lands on the middle size.
		if got := d.getOperandSize(0); got != 1 {
			t.Errorf("getOperandSize(0) = %v, want 1", got)
		}
	})

	t.Run("a size selector outside the table", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.sizeAttrSelect = 99
		if got := d.getOperandSize(0x0E); got != 0 {
			t.Errorf("getOperandSize = %v, want 0 for an out-of-range selector", got)
		}
	})

	t.Run("a relative immediate that carries into the high half", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.pos32 = 0xFFFFFFF0
		d.pos64 = 0xFFFFFFFF
		v32, v64, _, _ := d.resolveRelative(0x20, 0, 3)
		if v32 != 0x10 {
			t.Errorf("v32 = %v, want 16 after the carry", v32)
		}
		if v64 != 0 {
			t.Errorf("v64 = %v, want 0 after wrapping", v64)
		}
	})

	t.Run("an over-long instruction rewinds past the start", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.loadBinCode(strings.Repeat("66", 20) + "90")
		d.pos32 = 2 // fewer bytes consumed than the overshoot, so the difference wraps
		d.instructionHex = strings.Repeat("66", 20)
		d.enforceInstructionLimit()
		if d.instruction != x86Invalid {
			t.Errorf("instruction = %q, want %q", d.instruction, x86Invalid)
		}
	})
}

// Two size-attribute branches only fire under combinations of bit mode and
// vector state that the common encodings do not produce.
func TestX86OperandSizeBranches(t *testing.T) {
	t.Run("a single size is clamped to 32 below 64-bit mode", func(t *testing.T) {
		d := newX86Decoder(x86Mode32, 0)
		// A lone 64-bit attribute: every derived size aliases to it, so the
		// engine drops all three to 32.
		if got := d.getOperandSize(0x08); got != 2 {
			t.Errorf("getOperandSize = %v, want 2", got)
		}
	})

	t.Run("a broadcast round takes the vector to its maximum", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.vect = true
		d.conversionMode = 1
		d.sizeAttrSelect = 0
		if got := d.getOperandSize(0x70); got != 6 {
			t.Errorf("getOperandSize = %v, want 6", got)
		}
	})
}

// A handful of branches depend on vector state that no single encoding reaches
// through Run, so the decode helpers are driven directly.
func TestX86VectorStateBranches(t *testing.T) {
	t.Run("a vector register is never smaller than XMM", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.vect = true
		d.extension = 1
		// A byte-sized attribute would resolve below XMM, so it is raised to it.
		if got := d.decodeRegValue(0, true, 0x01); got != "XMM0" {
			t.Errorf("decodeRegValue = %q, want XMM0", got)
		}
	})

	t.Run("MVEX register mode folds in the rounding setting", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.extension = 3
		d.hintZeroMerg = 1
		d.roundingSetting = 0x18
		out, sc := d.decodeRegisterMode([3]float64{3, 0, 0}, false, 0x10, "{")
		if d.roundMode != 0x18 {
			t.Errorf("roundMode = %v, want 24", d.roundMode)
		}
		if out == "" || sc == "" {
			t.Errorf("decodeRegisterMode = %q, %q", out, sc)
		}
	})

	t.Run("a non-vector 128-bit operand uses the OWORD pointer", func(t *testing.T) {
		for _, tc := range []struct {
			mode int
			want string
		}{
			{x86Mode64, "OWORD PTR ["},
			{x86Mode32, "QWORD PTR ["},
		} {
			d := newX86Decoder(tc.mode, 0)
			d.vect = false
			out, _ := d.decodeEffectiveAddress([3]float64{0, 0, 0}, true, 16, "{")
			if !strings.HasPrefix(out, tc.want) {
				t.Errorf("mode %d: got %q, want it to start %q", tc.mode, out, tc.want)
			}
		}
	})

	t.Run("MM becomes QWORD under a vector extension", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.extension = 1
		out, _ := d.decodeEffectiveAddress([3]float64{0, 0, 0}, false, 9, "{")
		if !strings.HasPrefix(out, "QWORD PTR [") {
			t.Errorf("got %q, want it to start with QWORD PTR [", out)
		}
	})

	t.Run("a VSIB index below XMM is raised to it", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.vsib = 1
		d.loadBinCode("00")
		// ModR/M r/m of 4 routes into the SIB byte, which VSIB reads as a vector.
		out, _, _ := d.decode32BitAddress("", [3]float64{0, 0, 4}, 0, x86ImmDisp, x86Addr64, 2)
		if !strings.Contains(out, "XMM") {
			t.Errorf("got %q, want a vector index register", out)
		}
	})

	t.Run("MVEX selects the second name of a size-varying mnemonic", func(t *testing.T) {
		d := newX86Decoder(x86Mode64, 0)
		d.extension = 3
		d.hintZeroMerg = 1
		instr := x86Node{isList: true, list: []x86Node{x86Leaf("A"), x86Leaf("B"), x86Leaf("C")}}
		oper := x86Node{isList: true, list: []x86Node{x86Leaf("0"), x86Leaf("1"), x86Leaf("2")}}
		gotInstr, gotOper := d.narrowBySize(instr, oper)
		if gotInstr.str != "B" || gotOper.str != "1" {
			t.Errorf("narrowBySize = %q,%q; want B,1", gotInstr.str, gotOper.str)
		}
		if d.hintZeroMerg != 0 {
			t.Error("the cache hint should be consumed")
		}
	})
}
