package ops

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/arm64/arm64asm"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(DisassembleARM{})
}

// DisassembleARM translates ARM machine code into assembly language. CyberChef
// runs Capstone compiled to WebAssembly; cchef decodes with
// golang.org/x/arch, which is pure Go, and formats the result to Capstone's
// conventions so the output matches.
type DisassembleARM struct{}

// Meta returns the operation metadata.
func (DisassembleARM) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Disassemble ARM",
		Module: "Shellcode",
		Description: "Disassembles ARM machine code into assembly language.<br><br>Supports ARM " +
			"(32-bit), Thumb, and ARM64 (AArch64) architectures using the Capstone disassembly " +
			"framework.<br><br>Input should be in hexadecimal.",
		InfoURL:    "https://wikipedia.org/wiki/ARM_architecture_family",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// CyberChef's option lists.
var (
	armArchitectures = []string{"ARM (32-bit)", "ARM64 (AArch64)"}
	armModes         = []string{"ARM", "Thumb", "Thumb + Cortex-M", "ARMv8"}
	armEndianness    = []string{"Little Endian", "Big Endian"}
)

const (
	armArch32 = "ARM (32-bit)"
	armArch64 = "ARM64 (AArch64)"
)

// Args returns the argument definitions.
func (DisassembleARM) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Architecture", Type: core.ArgOption, Value: armArchitectures},
		{Name: "Mode", Type: core.ArgOption, Value: armModes},
		{Name: "Endianness", Type: core.ArgOption, Value: armEndianness},
		{Name: "Starting address (hex)", Type: core.ArgNumber, Value: float64(0)},
		{Name: "Show instruction hex", Type: core.ArgBoolean, Value: true},
		{Name: "Show instruction position", Type: core.ArgBoolean, Value: true},
	}
}

// CyberChef's verbatim OperationError texts.
var (
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	errARMBadHex = errors.New("Invalid hexadecimal input. Please provide valid hex characters only.")
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	errARMOddHex = errors.New("Invalid hexadecimal input. Length must be even.")
)

// armWordSize is the width of an ARM or ARM64 instruction in bytes, and
// armThumbUnit the width of a Thumb halfword.
const (
	armWordSize  = 4
	armThumbUnit = 2
)

// armHexColumn is the width the instruction bytes are padded to.
const armHexColumn = 16

// Run disassembles the hexadecimal input.
func (DisassembleARM) Run(in *core.Dish, args []any) (*core.Dish, error) {
	architecture := args[0].(string)
	mode := args[1].(string)
	bigEndian := args[2].(string) == "Big Endian"
	startAddress := uint64(args[3].(float64))
	showHex := args[4].(bool)
	showPos := args[5].(bool)

	code, err := armDecodeHex(in.String())
	if err != nil {
		return nil, err
	}
	if len(code) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	lines, err := armDisassemble(code, architecture, mode, bigEndian, startAddress, showHex, showPos)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(strings.Join(lines, "\n")), core.TypeString), nil
}

// armDecodeHex validates and decodes the hexadecimal input, reporting
// CyberChef's messages for the two ways it can be malformed.
func armDecodeHex(input string) ([]byte, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' {
			return -1
		}
		return r
	}, input)

	for i := range len(cleaned) {
		if !isHexDigit(cleaned[i]) {
			return nil, errARMBadHex
		}
	}
	if len(cleaned)%2 != 0 {
		return nil, errARMOddHex
	}
	return hex.DecodeString(cleaned)
}

// armDisassemble decodes each instruction and renders it in CyberChef's layout:
// address, instruction bytes, then mnemonic and operands.
func armDisassemble(code []byte, architecture, mode string, bigEndian bool, startAddress uint64, showHex, showPos bool) ([]string, error) {
	var lines []string
	thumb := architecture == armArch32 && strings.HasPrefix(mode, "Thumb")

	for pos := 0; pos < len(code); {
		address := startAddress + uint64(pos)
		text, size, ok := armDecodeAt(code, pos, architecture, thumb, bigEndian, address)
		if !ok {
			// Capstone's linear sweep stops at the first instruction it cannot
			// decode rather than resynchronising further along.
			break
		}

		var line strings.Builder
		if showPos {
			fmt.Fprintf(&line, "0x%08x  ", address)
		}
		if showHex {
			bytes := hex.EncodeToString(code[pos : pos+size])
			line.WriteString(bytes + strings.Repeat(" ", max(armHexColumn-len(bytes), 0)) + "  ")
		}
		line.WriteString(text)
		lines = append(lines, line.String())
		pos += size
	}

	if len(lines) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("No valid %s instructions found in input. "+
			"The bytes may be for a different architecture or mode.", architecture)
	}
	return lines, nil
}

// armDecodeAt decodes the instruction at pos, returning its text and width.
func armDecodeAt(code []byte, pos int, architecture string, thumb, bigEndian bool, address uint64) (string, int, bool) {
	if thumb {
		return armDecodeThumbAt(code, pos, bigEndian, address)
	}
	if pos+armWordSize > len(code) {
		return "", 0, false
	}
	word := code[pos : pos+armWordSize]
	if bigEndian {
		word = armSwap(word)
	}
	text, ok := armDecodeOne(word, architecture, address)
	return text, armWordSize, ok
}

// armSwap reverses a word, turning a big-endian instruction into the
// little-endian form the decoders expect.
func armSwap(word []byte) []byte {
	out := make([]byte, len(word))
	for i, b := range word {
		out[len(word)-1-i] = b
	}
	return out
}

// armDecodeOne decodes a single instruction word, reporting whether it decoded.
func armDecodeOne(word []byte, architecture string, address uint64) (string, bool) {
	if architecture == armArch64 {
		inst, err := arm64asm.Decode(word)
		if err != nil {
			return "", false
		}
		return armFormat64(inst, word, address), true
	}
	inst, err := armasm.Decode(word, armasm.ModeARM)
	if err != nil {
		return "", false
	}
	return armFormat32(inst, word, address), true
}

// armImmediate32 renders an immediate the way Capstone's ARM printer does: in
// decimal up to nine, and in lowercase hexadecimal beyond that, with the sign
// outside the 0x.
func armImmediate32(v int64) string {
	neg, mag := armImmediateSign(v)
	if mag <= 9 {
		return "#" + neg + strconv.FormatInt(mag, 10)
	}
	return "#" + neg + "0x" + strconv.FormatInt(mag, 16)
}

// armImmediateSign splits an immediate into its sign and magnitude.
func armImmediateSign(v int64) (string, int64) {
	if v < 0 {
		return "-", -v
	}
	return "", v
}

// armPCRelPattern matches x/arch's relative branch target.
var armPCRelPattern = regexp.MustCompile(`\.[+-]0x[0-9a-fA-F]+|\.[+-]\d+`)

// armShiftOps are the shift and rotate keywords; Capstone prints the amount
// that follows one of them in decimal, whatever its size.
var armShiftOps = []string{"lsl", "lsr", "asr", "ror", "rrx"}

// armRewriteOperands puts every immediate in an operand string into Capstone's
// format. Three things vary: shift amounts are always decimal, memory offsets
// keep their sign, and a bare 32-bit immediate is printed unsigned.
func armRewriteOperands(s string, bare, offset func(int64) string) string {
	var b strings.Builder
	depth := 0

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case '#':
			value, width, ok := armParseImmediate(s[i:])
			if !ok {
				break
			}
			switch {
			case armFollowsShift(s[:i]):
				b.WriteString("#" + strconv.FormatInt(value, 10))
			case depth > 0:
				b.WriteString(offset(value))
			default:
				b.WriteString(bare(value))
			}
			i += width - 1
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// armParseImmediate reads the immediate at the start of s, which begins with
// '#', returning its value and how many bytes it spans.
func armParseImmediate(s string) (int64, int, bool) {
	i := 1
	neg := false
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	start := i
	base := 10
	if strings.HasPrefix(s[i:], "0x") || strings.HasPrefix(s[i:], "0X") {
		base = 16
		i += 2
		start = i
		for i < len(s) && isHexDigit(s[i]) {
			i++
		}
	} else {
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	if i == start {
		return 0, 0, false
	}
	v, err := strconv.ParseInt(s[start:i], base, 64)
	if err != nil {
		return 0, 0, false
	}
	if neg {
		v = -v
	}
	return v, i, true
}

// armFollowsShift reports whether the text just before an immediate ends in a
// shift keyword.
func armFollowsShift(before string) bool {
	trimmed := strings.TrimRight(before, " ")
	for _, op := range armShiftOps {
		if strings.HasSuffix(trimmed, op) {
			return true
		}
	}
	return false
}

// armUnsignedImmediate64 formats an AArch64 value that Capstone prints as an
// unsigned 64-bit word rather than a negative displacement: branch targets and
// post-indexed memory offsets.
func armUnsignedImmediate64(v int64) string {
	if v < 0 {
		return "#0x" + strconv.FormatUint(uint64(v), 16) // #nosec G115 -- deliberate 64-bit reinterpretation
	}
	return armImmediate32(v)
}

// armImmediate64Wide formats the immediate of a move-wide instruction, the one
// AArch64 family Capstone prints in hexadecimal whatever its size.
func armImmediate64Wide(v int64) string {
	neg, mag := armImmediateSign(v)
	if mag == 0 {
		return "#0"
	}
	return "#" + neg + "0x" + strconv.FormatInt(mag, 16)
}

// armMoveWideOps are the mnemonics that use the wide immediate format.
var armMoveWideOps = map[string]bool{"movz": true, "movk": true, "movn": true}

// armBareHex matches an operand x/arch prints as a plain hexadecimal literal,
// such as the comment field of SVC, where Capstone prints an immediate.
var armBareHex = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)

// armFixBareImmediate turns such a literal into Capstone's immediate syntax.
func armFixBareImmediate(operands string, format func(int64) string) string {
	if !armBareHex.MatchString(operands) {
		return operands
	}
	v, err := strconv.ParseInt(operands[2:], 16, 64)
	if err != nil {
		return operands
	}
	return format(v)
}

// armSpaceAfterCommas inserts the space Capstone puts after every operand
// separator; x/arch omits it inside memory operands.
func armSpaceAfterCommas(s string) string {
	var b strings.Builder
	for i := range len(s) {
		b.WriteByte(s[i])
		if s[i] == ',' && i+1 < len(s) && s[i+1] != ' ' {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// armUnsigned32 reinterprets a negative value as the unsigned 32-bit word
// Capstone prints for a bare ARM immediate or branch target.
func armUnsigned32(v int64) int64 {
	if v < 0 {
		return int64(uint32(v)) // #nosec G115 -- deliberate 32-bit reinterpretation
	}
	return v
}

// armImmediate32Bare formats an ARM immediate that is not a memory offset, for
// which Capstone shows the unsigned 32-bit value.
func armImmediate32Bare(v int64) string {
	return armImmediate32(armUnsigned32(v))
}

// armConditions maps the condition-code spellings x/arch uses to Capstone's.
var armConditions = map[string]string{"cs": "hs", "cc": "lo"}

// armMnemonic32 builds the mnemonic from the decoded operation so the flag and
// condition suffixes carry Capstone's spelling rather than x/arch's.
func armMnemonic32(inst armasm.Inst) string {
	parts := strings.Split(strings.ToLower(inst.Op.String()), ".")
	var out strings.Builder
	out.WriteString(parts[0])
	for _, part := range parts[1:] {
		if mapped, ok := armConditions[part]; ok {
			part = mapped
		}
		out.WriteString(part)
	}
	return out.String()
}

// armSplitMnemonic separates the mnemonic from the operand string.
func armSplitMnemonic(s string) (string, string) {
	mnemonic, operands, found := strings.Cut(s, " ")
	if !found {
		return s, ""
	}
	return mnemonic, operands
}

// armJoin puts a mnemonic and its operands back together, dropping the space
// when there are no operands.
func armJoin(mnemonic, operands string) string {
	if operands == "" {
		return mnemonic
	}
	return mnemonic + " " + operands
}

// armFormat32 renders a 32-bit ARM instruction in Capstone's syntax.
func armFormat32(inst armasm.Inst, word []byte, address uint64) string {
	_, operands := armSplitMnemonic(armasm.GNUSyntax(inst))
	mnemonic := armMnemonic32(inst)

	// Capstone resolves a branch target to its absolute address; x/arch prints
	// it relative to the instruction. ARM's program counter reads two
	// instructions ahead.
	for _, arg := range inst.Args {
		if rel, ok := arg.(armasm.PCRel); ok {
			target := armUnsigned32(int64(address) + 2*armWordSize + int64(rel)) // #nosec G115 -- instruction addresses stay far below 2^63
			operands = armPCRelPattern.ReplaceAllString(operands, armImmediate32(target))
			break
		}
	}

	mnemonic, operands = armFixPushPop32(word, mnemonic, operands)
	operands = armFixBareImmediate(operands, armImmediate32)
	operands = armRewriteOperands(armSpaceAfterCommas(operands), armImmediate32Bare, armImmediate32)
	return armJoin(mnemonic, operands)
}

// ARM32 encoding fields used to tell a single-register store apart from a real
// block transfer.
const (
	armLoadStoreMask  = 0x0E000000 // bits 27..25
	armLoadStoreValue = 0x04000000 // 0b010: single-register load/store
)

// armFixPushPop32 undoes x/arch's habit of printing a single-register
// pre-indexed store as PUSH. Capstone keeps the str form there, but does print
// the matching post-indexed load as POP, so only the store is rewritten.
func armFixPushPop32(word []byte, mnemonic, operands string) (string, string) {
	if !strings.HasPrefix(mnemonic, "push") {
		return mnemonic, operands
	}
	raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
	if raw&armLoadStoreMask != armLoadStoreValue {
		return mnemonic, operands
	}

	// The register list is already rendered with the names Capstone uses, so it
	// is read back out of the operand string rather than off the typed argument.
	reg := strings.Trim(operands, "{}")
	if reg == "" || strings.Contains(reg, ",") {
		return mnemonic, operands
	}

	return "str", reg + ", [sp, #-4]!"
}

// ARM64 encoding fields for the move-wide family, which Capstone spells movz
// where x/arch prefers the mov alias.
const (
	arm64MoveWideMask   = 0x7F800000
	arm64MoveWideZero   = 0x52800000 // 32-bit MOVZ
	arm64MoveWideZero64 = 0xD2800000
)

// armFormat64 renders an ARM64 instruction in Capstone's syntax.
func armFormat64(inst arm64asm.Inst, word []byte, address uint64) string {
	mnemonic, operands := armSplitMnemonic(arm64asm.GNUSyntax(inst))

	for _, arg := range inst.Args {
		if rel, ok := arg.(arm64asm.PCRel); ok {
			target := int64(address) + int64(rel) // #nosec G115 -- instruction addresses stay far below 2^63
			operands = armPCRelPattern.ReplaceAllString(operands, armUnsignedImmediate64(target))
			break
		}
	}

	raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
	if mnemonic == "mov" && (raw&arm64MoveWideMask == arm64MoveWideZero ||
		raw&arm64MoveWideMask == arm64MoveWideZero64) {
		mnemonic = "movz"
	}

	bare := armUnsignedImmediate64
	if armMoveWideOps[mnemonic] {
		bare = armImmediate64Wide
	}
	operands = armRewriteOperands(armSpaceAfterCommas(operands), bare, armImmediate32)
	return armJoin(mnemonic, operands)
}

/*
   ---------------------------------------------------------------------------
   Thumb.

   golang.org/x/arch decodes ARM and AArch64 but rejects Thumb outright, so the
   Thumb instruction set is decoded here from scratch, printing what Capstone
   prints. Encodings follow the ARM Architecture Reference Manual's Thumb
   sections; the comments name each group as the manual does.
   ---------------------------------------------------------------------------
*/

// armRegNames are the register names Capstone prints for r0-r15.
var armRegNames = [16]string{
	"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7",
	"r8", "sb", "sl", "fp", "ip", "sp", "lr", "pc",
}

// armReg names a register by number.
func armReg(n uint32) string { return armRegNames[n&0xF] }

// armRegList renders a register list as Capstone does, in ascending order.
func armRegList(mask uint32) string {
	var regs []string
	for i := range uint32(16) {
		if mask&(1<<i) != 0 {
			regs = append(regs, armReg(i))
		}
	}
	return "{" + strings.Join(regs, ", ") + "}"
}

// armDecodeThumbAt decodes the Thumb instruction at pos.
func armDecodeThumbAt(code []byte, pos int, bigEndian bool, address uint64) (string, int, bool) {
	if pos+armThumbUnit > len(code) {
		return "", 0, false
	}
	first := armThumbHalfword(code, pos, bigEndian)

	// Bits 15:11 of 0b11101, 0b11110 or 0b11111 introduce a 32-bit encoding.
	if top := first >> 11; top == 0x1D || top == 0x1E || top == 0x1F {
		if pos+2*armThumbUnit > len(code) {
			return "", 0, false
		}
		second := armThumbHalfword(code, pos+armThumbUnit, bigEndian)
		text, ok := armDecodeThumb32(first, second, address)
		return text, 2 * armThumbUnit, ok
	}

	text, ok := armDecodeThumb16(first, address)
	return text, armThumbUnit, ok
}

// armThumbHalfword reads one 16-bit instruction halfword.
func armThumbHalfword(code []byte, pos int, bigEndian bool) uint32 {
	if bigEndian {
		return uint32(code[pos])<<8 | uint32(code[pos+1])
	}
	return uint32(code[pos+1])<<8 | uint32(code[pos])
}

// armThumbTarget resolves a Thumb branch displacement, which is measured from
// the program counter two halfwords ahead of the instruction.
func armThumbTarget(address uint64, offset int32) string {
	return armImmediate32(armUnsigned32(int64(address) + 4 + int64(offset))) // #nosec G115 -- instruction addresses stay far below 2^63
}

// armSignExtend sign-extends an n-bit value.
func armSignExtend(v uint32, bits uint) int32 {
	shift := 32 - bits
	return int32(v<<shift) >> shift // #nosec G115 -- sign extension by construction
}

// armDecodeThumb16 decodes a 16-bit Thumb instruction.
func armDecodeThumb16(op uint32, address uint64) (string, bool) {
	switch {
	case op>>13 == 0b000:
		return armThumbShiftOrAddSub(op)
	case op>>13 == 0b001:
		return armThumbImmediate(op)
	case op>>10 == 0b010000:
		return armThumbDataProcessing(op)
	case op>>10 == 0b010001:
		return armThumbSpecialData(op)
	case op>>11 == 0b01001:
		return armThumbLoadLiteral(op)
	case op>>12 == 0b0101:
		return armThumbLoadStoreRegister(op)
	case op>>13 == 0b011:
		return armThumbLoadStoreWordByte(op)
	case op>>12 == 0b1000:
		return armThumbLoadStoreHalfword(op)
	case op>>12 == 0b1001:
		return armThumbLoadStoreStack(op)
	case op>>12 == 0b1010:
		return armThumbAddress(op)
	case op>>12 == 0b1011:
		return armThumbMisc(op, address)
	case op>>12 == 0b1100:
		return armThumbBlockTransfer(op)
	case op>>12 == 0b1101:
		return armThumbConditionalBranch(op, address)
	}

	// Everything else in the 16-bit space is the unconditional branch: the
	// caller has already routed the three 32-bit prefixes elsewhere.
	offset := armSignExtend(op&0x7FF, 11) * 2
	return "b " + armThumbTarget(address, offset), true
}

// armThumbShiftOrAddSub covers the shift-by-immediate and add/subtract
// register or 3-bit immediate encodings.
func armThumbShiftOrAddSub(op uint32) (string, bool) {
	rd, rn := armReg(op&7), armReg((op>>3)&7)

	if op>>11 == 0b00011 {
		mnemonic := "adds"
		if op&(1<<9) != 0 {
			mnemonic = "subs"
		}
		if op&(1<<10) != 0 { // 3-bit immediate
			return fmt.Sprintf("%s %s, %s, %s", mnemonic, rd, rn, armImmediate32(int64((op>>6)&7))), true
		}
		return fmt.Sprintf("%s %s, %s, %s", mnemonic, rd, rn, armReg((op>>6)&7)), true
	}

	shift := [3]string{"lsls", "lsrs", "asrs"}[(op>>11)&3]
	amount := (op >> 6) & 0x1F
	// LSL by zero is the register move.
	if shift == "lsls" && amount == 0 {
		return fmt.Sprintf("movs %s, %s", rd, rn), true
	}
	// LSR and ASR encode a shift of 32 as zero.
	if amount == 0 {
		amount = 32
	}
	return fmt.Sprintf("%s %s, %s, %s", shift, rd, rn, armImmediate32(int64(amount))), true
}

// armThumbImmediate covers move, compare, add and subtract with an 8-bit
// immediate.
func armThumbImmediate(op uint32) (string, bool) {
	mnemonic := [4]string{"movs", "cmp", "adds", "subs"}[(op>>11)&3]
	return fmt.Sprintf("%s %s, %s", mnemonic, armReg((op>>8)&7), armImmediate32(int64(op&0xFF))), true
}

// armThumbDataOps are the sixteen register-to-register operations.
var armThumbDataOps = [16]string{
	"ands", "eors", "lsls", "lsrs", "asrs", "adcs", "sbcs", "rors",
	"tst", "rsbs", "cmp", "cmn", "orrs", "muls", "bics", "mvns",
}

// armThumbDataProcessing covers the register-to-register data operations.
func armThumbDataProcessing(op uint32) (string, bool) {
	mnemonic := armThumbDataOps[(op>>6)&0xF]
	rd, rm := armReg(op&7), armReg((op>>3)&7)

	switch mnemonic {
	case "rsbs": // negate, which Capstone prints with its zero operand
		return fmt.Sprintf("rsbs %s, %s, #0", rd, rm), true
	case "muls": // the destination is also the second source
		return fmt.Sprintf("muls %s, %s, %s", rd, rm, rd), true
	}
	return fmt.Sprintf("%s %s, %s", mnemonic, rd, rm), true
}

// armThumbSpecialData covers the high-register operations and branch exchange.
func armThumbSpecialData(op uint32) (string, bool) {
	rm := armReg((op >> 3) & 0xF)

	if (op>>8)&3 == 0b11 {
		if op&(1<<7) != 0 {
			return "blx " + rm, true
		}
		return "bx " + rm, true
	}

	mnemonic := [3]string{"add", "cmp", "mov"}[(op>>8)&3]
	rd := armReg((op & 7) | (op>>7)&1<<3)
	return fmt.Sprintf("%s %s, %s", mnemonic, rd, rm), true
}

// armThumbLoadLiteral covers a load relative to the program counter.
func armThumbLoadLiteral(op uint32) (string, bool) {
	offset := int64(op&0xFF) * 4
	return fmt.Sprintf("ldr %s, [pc, %s]", armReg((op>>8)&7), armImmediate32(offset)), true
}

// armThumbRegisterOps are the load and store operations with a register offset.
var armThumbRegisterOps = [8]string{"str", "strh", "strb", "ldrsb", "ldr", "ldrh", "ldrb", "ldrsh"}

// armThumbLoadStoreRegister covers loads and stores with a register offset.
func armThumbLoadStoreRegister(op uint32) (string, bool) {
	mnemonic := armThumbRegisterOps[(op>>9)&7]
	return fmt.Sprintf("%s %s, [%s, %s]", mnemonic, armReg(op&7), armReg((op>>3)&7), armReg((op>>6)&7)), true
}

// armThumbLoadStoreWordByte covers word and byte access with an immediate
// offset.
func armThumbLoadStoreWordByte(op uint32) (string, bool) {
	mnemonic, scale := "str", uint32(4)
	if op&(1<<12) != 0 {
		mnemonic, scale = "strb", 1
	}
	if op&(1<<11) != 0 {
		mnemonic = "ld" + mnemonic[2:]
	}
	offset := int64((op>>6)&0x1F) * int64(scale)
	return armThumbMemory(mnemonic, armReg(op&7), armReg((op>>3)&7), offset), true
}

// armThumbLoadStoreHalfword covers halfword access with an immediate offset.
func armThumbLoadStoreHalfword(op uint32) (string, bool) {
	mnemonic := "strh"
	if op&(1<<11) != 0 {
		mnemonic = "ldrh"
	}
	offset := int64((op>>6)&0x1F) * 2
	return armThumbMemory(mnemonic, armReg(op&7), armReg((op>>3)&7), offset), true
}

// armThumbLoadStoreStack covers access relative to the stack pointer.
func armThumbLoadStoreStack(op uint32) (string, bool) {
	mnemonic := "str"
	if op&(1<<11) != 0 {
		mnemonic = "ldr"
	}
	offset := int64(op&0xFF) * 4
	return armThumbMemory(mnemonic, armReg((op>>8)&7), "sp", offset), true
}

// armThumbMemory renders a memory operand, omitting a zero offset the way
// Capstone does.
func armThumbMemory(mnemonic, rt, rn string, offset int64) string {
	if offset == 0 {
		return fmt.Sprintf("%s %s, [%s]", mnemonic, rt, rn)
	}
	return fmt.Sprintf("%s %s, [%s, %s]", mnemonic, rt, rn, armImmediate32(offset))
}

// armThumbAddress covers forming an address from the program counter or the
// stack pointer.
func armThumbAddress(op uint32) (string, bool) {
	rd := armReg((op >> 8) & 7)
	offset := int64(op&0xFF) * 4
	if op&(1<<11) != 0 {
		return fmt.Sprintf("add %s, sp, %s", rd, armImmediate32(offset)), true
	}
	return fmt.Sprintf("adr %s, %s", rd, armImmediate32(offset)), true
}

// armThumbBlockTransfer covers the multiple load and store forms. A load whose
// base register is itself in the list does not write back, and Capstone leaves
// off the marker accordingly.
func armThumbBlockTransfer(op uint32) (string, bool) {
	mnemonic, writeback := "stm", "!"
	base := (op >> 8) & 7
	if op&(1<<11) != 0 {
		mnemonic = "ldm"
		if op&(1<<base) != 0 {
			writeback = ""
		}
	}
	return fmt.Sprintf("%s %s%s, %s", mnemonic, armReg(base), writeback, armRegList(op&0xFF)), true
}

// armConditionNames are the condition suffixes Capstone prints.
var armConditionNames = [16]string{
	"eq", "ne", "hs", "lo", "mi", "pl", "vs", "vc",
	"hi", "ls", "ge", "lt", "gt", "le", "", "",
}

// armThumbConditionalBranch covers conditional branches and the supervisor
// call, which share the encoding.
func armThumbConditionalBranch(op uint32, address uint64) (string, bool) {
	cond := (op >> 8) & 0xF
	switch cond {
	case 0xF:
		return "svc " + armImmediate32(int64(op&0xFF)), true
	case 0xE:
		return "udf " + armImmediate32(int64(op&0xFF)), true
	}
	offset := armSignExtend(op&0xFF, 8) * 2
	return "b" + armConditionNames[cond] + " " + armThumbTarget(address, offset), true
}

// armThumbExtendOps are the sign- and zero-extend operations.
var armThumbExtendOps = [4]string{"sxth", "sxtb", "uxth", "uxtb"}

// armThumbReverseOps are the byte-reversal operations; the third slot is
// unallocated.
var armThumbReverseOps = [4]string{"rev", "rev16", "", "revsh"}

// armThumbHints are the hint instructions encoded in the IT space.
var armThumbHints = [5]string{"nop", "yield", "wfe", "wfi", "sev"}

// armThumbMisc covers the miscellaneous 16-bit group: stack adjustment,
// compare-and-branch, extends, push and pop, byte reversal, breakpoints and
// the hint instructions.
func armThumbMisc(op uint32, address uint64) (string, bool) {
	switch {
	case op>>8 == 0b10110000:
		amount := armImmediate32(int64(op&0x7F) * 4)
		if op&(1<<7) != 0 {
			return "sub sp, " + amount, true
		}
		return "add sp, " + amount, true

	case (op>>12 == 0b1011) && (op>>8)&0b0101 == 0b0001: // compare and branch
		mnemonic := "cbz"
		if op&(1<<11) != 0 {
			mnemonic = "cbnz"
		}
		offset := int32((op>>9)&1<<6 | (op>>3)&0x1F<<1) // #nosec G115 -- six-bit unsigned displacement
		return fmt.Sprintf("%s %s, %s", mnemonic, armReg(op&7), armThumbTarget(address, offset)), true

	case op>>8 == 0b10110010:
		return fmt.Sprintf("%s %s, %s", armThumbExtendOps[(op>>6)&3], armReg(op&7), armReg((op>>3)&7)), true

	case op>>8 == 0b10111010:
		mnemonic := armThumbReverseOps[(op>>6)&3]
		if mnemonic == "" {
			return "", false
		}
		return fmt.Sprintf("%s %s, %s", mnemonic, armReg(op&7), armReg((op>>3)&7)), true

	case op>>8 == 0b10111110:
		return "bkpt " + armImmediate32(int64(op&0xFF)), true

	case op>>8 == 0b10111111:
		return armThumbHint(op)

	case (op>>9)&0b011 == 0b010: // push and pop
		return armThumbPushPop(op)
	}
	return "", false
}

// armThumbHint covers the hint instructions and the IT block that shares their
// encoding.
func armThumbHint(op uint32) (string, bool) {
	mask := op & 0xF
	if mask != 0 {
		return armThumbIT((op>>4)&0xF, mask), true
	}
	hint := (op >> 4) & 0xF
	if int(hint) < len(armThumbHints) {
		return armThumbHints[hint], true
	}
	return "hint " + armImmediate32(int64(hint)), true
}

// armThumbIT renders an IT block. The mask spells out whether each of the up to
// three following instructions runs on the condition (T) or its inverse (E).
func armThumbIT(firstCond, mask uint32) string {
	lowest := 0
	for lowest < 4 && mask&(1<<lowest) == 0 {
		lowest++
	}

	var suffix strings.Builder
	for i := 3; i > lowest; i-- {
		if (mask>>i)&1 == firstCond&1 {
			suffix.WriteString("t")
		} else {
			suffix.WriteString("e")
		}
	}
	return "it" + suffix.String() + " " + armConditionNames[firstCond]
}

// armThumbPushPop covers the stack push and pop forms, which fold the link
// register or program counter into the register list.
func armThumbPushPop(op uint32) (string, bool) {
	list := op & 0xFF
	mnemonic := "push"
	extra := uint32(1 << 14) // lr
	if op&(1<<11) != 0 {
		mnemonic, extra = "pop", 1<<15 // pc
	}
	if op&(1<<8) != 0 {
		list |= extra
	}
	return mnemonic + " " + armRegList(list), true
}

// armDecodeThumb32 decodes a 32-bit Thumb-2 instruction from its two halfwords.
// The two bits below the 0b111 escape choose between the three top-level
// groups the manual defines.
func armDecodeThumb32(first, second uint32, address uint64) (string, bool) {
	switch (first >> 11) & 3 {
	case 0b01:
		return armThumb32LoadMultipleOrDataRegister(first, second)
	case 0b10:
		if second&(1<<15) != 0 {
			return armThumb32BranchOrControl(first, second, address)
		}
		return armThumb32DataImmediate(first, second)
	case 0b11:
		return armThumb32LoadStoreSingle(first, second)
	}
	return "", false
}

// armThumb32BranchOrControl covers the wide branches and the miscellaneous
// control instructions that share their encoding.
func armThumb32BranchOrControl(first, second uint32, address uint64) (string, bool) {
	return armThumbLongBranch(first, second, address)
}

// armThumbExpandImm reproduces the manual's ThumbExpandImm: the low eight bits
// are either replicated across the word or rotated into place, depending on the
// top four bits of the twelve-bit field.
func armThumbExpandImm(value uint32) uint32 {
	imm8 := value & 0xFF
	if value&0xC00 == 0 {
		switch (value >> 8) & 3 {
		case 0b00:
			return imm8
		case 0b01:
			return imm8<<16 | imm8
		case 0b10:
			return imm8<<24 | imm8<<8
		default:
			return imm8<<24 | imm8<<16 | imm8<<8 | imm8
		}
	}
	// A set top bit means the value is a rotated byte with an implicit leading 1.
	rotation := (value >> 7) & 0x1F
	unrotated := 0x80 | (imm8 & 0x7F)
	return unrotated>>rotation | unrotated<<(32-rotation)
}

// armThumb32DataOps names the data-processing operations by their opcode field.
var armThumb32DataOps = map[uint32]string{
	0b0000: "and", 0b0001: "bic", 0b0010: "orr", 0b0011: "orn", 0b0100: "eor",
	0b1000: "add", 0b1010: "adc", 0b1011: "sbc", 0b1101: "sub", 0b1110: "rsb",
}

// The .w marker says a 32-bit encoding was used where a 16-bit one exists, so
// which mnemonics carry it depends on the operand form as well as the
// operation. These two sets were read off Capstone's own output.
var (
	armThumb32WideImmediateOps = map[string]bool{
		"add": true, "sub": true, "rsb": true, "mov": true,
	}
	armThumb32WideRegisterOps = map[string]bool{
		"and": true, "bic": true, "orr": true, "eor": true,
		"add": true, "adc": true, "sbc": true, "sub": true, "mov": true,
	}
)

// armThumb32Mnemonic assembles a mnemonic from its base, the flag-setting bit
// and whether the wide marker applies.
func armThumb32Mnemonic(base string, setsFlags, wide bool) string {
	if setsFlags {
		base += "s"
	}
	if wide {
		base += ".w"
	}
	return base
}

// armThumb32DataImmediate covers data processing with a modified immediate and
// the plain binary immediate forms that share the encoding.
func armThumb32DataImmediate(first, second uint32) (string, bool) {
	if first&(1<<9) != 0 {
		return armThumb32PlainImmediate(first, second)
	}

	op := (first >> 5) & 0xF
	base, ok := armThumb32DataOps[op]
	if !ok {
		return "", false
	}
	setsFlags := first&(1<<4) != 0
	rn, rd := first&0xF, (second>>8)&0xF
	value := armThumbExpandImm((first>>10)&1<<11 | (second>>12)&7<<8 | second&0xFF)
	imm := armImmediate32(int64(int32(value))) // #nosec G115 -- Capstone prints the expanded value signed

	// With no destination the operation is a comparison instead.
	if rd == 15 && setsFlags {
		if compare, ok := armThumb32CompareOps[op]; ok {
			return fmt.Sprintf("%s.w %s, %s", compare, armReg(rn), imm), true
		}
	}

	// ORR and ORN against the unused register are the move aliases.
	if rn == 15 {
		switch base {
		case "orr":
			return fmt.Sprintf("%s %s, %s", armThumb32Mnemonic("mov", setsFlags, true), armReg(rd), imm), true
		case "orn":
			return fmt.Sprintf("%s %s, %s", armThumb32Mnemonic("mvn", setsFlags, false), armReg(rd), imm), true
		}
	}

	mnemonic := armThumb32Mnemonic(base, setsFlags, armThumb32WideImmediateOps[base])
	return fmt.Sprintf("%s %s, %s, %s", mnemonic, armReg(rd), armReg(rn), imm), true
}

// armThumb32CompareOps are the data-processing opcodes that become comparisons
// when they discard their result.
var armThumb32CompareOps = map[uint32]string{
	0b0000: "tst", 0b0100: "teq", 0b1000: "cmn", 0b1101: "cmp",
}

// armThumbLongBranch covers the wide branch forms, whose displacement is split
// across both halfwords with two inverted bits.
func armThumbLongBranch(first, second uint32, address uint64) (string, bool) {
	s := (first >> 10) & 1

	i1, i2 := 1-((second>>13)&1^s), 1-((second>>11)&1^s)
	wide := s<<24 | i1<<23 | i2<<22 | (first&0x3FF)<<12 | (second&0x7FF)<<1

	switch (second >> 12) & 0b101 {
	case 0b100: // blx, whose target is aligned to a word
		target := (int64(address)+4)&^3 + int64(armSignExtend(wide&^3, 25)) // #nosec G115 -- instruction addresses stay far below 2^63
		return "blx " + armImmediate32(armUnsigned32(target)), true

	case 0b101: // bl
		return "bl " + armThumbTarget(address, armSignExtend(wide, 25)), true

	case 0b001: // b.w
		return "b.w " + armThumbTarget(address, armSignExtend(wide, 25)), true
	}

	// Conditional wide branch.
	cond := (first >> 6) & 0xF
	if cond >= 0xE {
		return "", false
	}
	imm := s<<20 | ((second>>11)&1)<<19 | ((second>>13)&1)<<18 | (first&0x3F)<<12 | (second&0x7FF)<<1
	return "b" + armConditionNames[cond] + ".w " + armThumbTarget(address, armSignExtend(imm, 21)), true
}

// armThumb32ShiftNames are the shift types encoded in a shifted-register
// operand.
var armThumb32ShiftNames = [4]string{"lsl", "lsr", "asr", "ror"}

// armThumb32Shift renders the shift applied to a register operand, and reports
// whether there is one at all. LSR and ASR encode a shift of 32 as zero, and a
// zero-length rotate is the rotate-with-extend.
func armThumb32Shift(kind, amount uint32) (string, bool) {
	if amount == 0 {
		switch kind {
		case 0b00:
			return "", false
		case 0b11:
			return "rrx", true
		}
		amount = 32
	}
	return fmt.Sprintf("%s #%d", armThumb32ShiftNames[kind], amount), true
}

// armThumb32LoadMultipleOrDataRegister covers the second top-level group: block
// transfers, the dual and exclusive accesses, data processing on a shifted
// register, and the coprocessor space.
func armThumb32LoadMultipleOrDataRegister(first, second uint32) (string, bool) {
	op2 := (first >> 4) & 0x7F
	switch {
	case op2&0b1100100 == 0b0000000:
		return armThumb32BlockTransfer(first, second)
	case op2&0b1100100 == 0b0000100:
		return armThumb32DualOrExclusive(first, second)
	case op2&0b1100000 == 0b0100000:
		return armThumb32DataShiftedRegister(first, second)
	}
	return "", false
}

// armThumb32BlockTransfer covers the wide block transfers, including the push
// and pop aliases against the stack pointer.
func armThumb32BlockTransfer(first, second uint32) (string, bool) {
	load := first&(1<<4) != 0
	writeback := first&(1<<5) != 0
	rn := first & 0xF
	if !armThumb32ListIsValid(second, load) {
		return "", false
	}
	list := armRegList(second)

	// Increment-after on the stack pointer is pop; decrement-before is push.
	if rn == 13 && writeback {
		if load && (first>>7)&1 == 1 {
			return "pop.w " + list, true
		}
		if !load && (first>>7)&1 == 0 {
			return "push.w " + list, true
		}
	}

	mnemonic := "stm"
	if load {
		mnemonic = "ldm"
	}
	switch (first >> 7) & 3 {
	case 0b01: // increment after
		mnemonic += ".w"
	case 0b10: // decrement before
		mnemonic += "db"
	default:
		return "", false
	}
	if writeback {
		return mnemonic + " " + armReg(rn) + "!, " + list, true
	}
	return mnemonic + " " + armReg(rn) + ", " + list, true
}

// armThumb32DataShiftedRegister covers data processing whose second operand is
// a register with an optional shift.
func armThumb32DataShiftedRegister(first, second uint32) (string, bool) {
	op := (first >> 5) & 0xF
	base, ok := armThumb32DataOps[op]
	if !ok {
		return "", false
	}
	setsFlags := first&(1<<4) != 0
	rn, rd, rm := first&0xF, (second>>8)&0xF, second&0xF
	amount := (second>>12)&7<<2 | (second>>6)&3
	kind := (second >> 4) & 3
	shift, shifted := armThumb32Shift(kind, amount)

	if rd == 15 && setsFlags {
		if compare, ok := armThumb32CompareOps[op]; ok {
			return armThumb32Join(compare+".w", armReg(rn), armReg(rm), shift, shifted), true
		}
	}

	if rn == 15 {
		switch base {
		case "orr":
			return armThumb32MoveShifted(rd, rm, kind, amount, setsFlags, shifted, shift), true
		case "orn":
			return armThumb32Join(armThumb32Mnemonic("mvn", setsFlags, true), armReg(rd), armReg(rm), shift, shifted), true
		}
	}

	mnemonic := armThumb32Mnemonic(base, setsFlags, armThumb32WideRegisterOps[base])
	operands := armReg(rd) + ", " + armReg(rn) + ", " + armReg(rm)
	if shifted {
		operands += ", " + shift
	}
	return mnemonic + " " + operands, true
}

// armThumb32MoveShifted renders the MOV form of a shifted register, which
// Capstone prints as the shift instruction when there is a shift to apply.
func armThumb32MoveShifted(rd, rm, kind, amount uint32, setsFlags, shifted bool, shift string) string {
	if !shifted {
		return armThumb32Mnemonic("mov", setsFlags, true) + " " + armReg(rd) + ", " + armReg(rm)
	}
	if amount == 0 { // rotate with extend takes no amount
		return armThumb32Mnemonic("rrx", setsFlags, false) + " " + armReg(rd) + ", " + armReg(rm)
	}
	mnemonic := armThumb32Mnemonic(armThumb32ShiftNames[kind], setsFlags, true)
	return fmt.Sprintf("%s %s, %s, #%d", mnemonic, armReg(rd), armReg(rm), amount)
}

// armThumb32Join assembles a two-register operand list with an optional shift.
func armThumb32Join(mnemonic, a, b, shift string, shifted bool) string {
	out := mnemonic + " " + a + ", " + b
	if shifted {
		out += ", " + shift
	}
	return out
}

// armThumb32PlainImmediate covers the immediate forms that use their operand
// literally rather than expanding it: the wide add and subtract, the move
// halves, and the bitfield and saturating instructions.
func armThumb32PlainImmediate(first, second uint32) (string, bool) {
	rn, rd := first&0xF, (second>>8)&0xF
	imm12 := (first>>10)&1<<11 | (second>>12)&7<<8 | second&0xFF

	switch (first >> 4) & 0x1F {
	case 0b00000: // addw
		return fmt.Sprintf("addw %s, %s, %s", armReg(rd), armReg(rn), armImmediate32(int64(imm12))), true

	case 0b01010: // subw
		return fmt.Sprintf("subw %s, %s, %s", armReg(rd), armReg(rn), armImmediate32(int64(imm12))), true

	case 0b00100: // movw
		return fmt.Sprintf("movw %s, %s", armReg(rd), armImmediate32(int64(rn<<12|imm12))), true

	case 0b01100: // movt
		return fmt.Sprintf("movt %s, %s", armReg(rd), armImmediate32(int64(rn<<12|imm12))), true

	case 0b10100, 0b11100: // signed and unsigned bitfield extract
		return armThumb32BitfieldExtract(first, second), true

	case 0b10110: // bitfield insert, or clear when there is no source
		return armThumb32BitfieldInsert(first, second)
	}
	return "", false
}

// armThumb32BitfieldExtract renders SBFX and UBFX, whose operands are a bit
// position and a width.
func armThumb32BitfieldExtract(first, second uint32) string {
	mnemonic := "sbfx"
	if (first>>4)&0x1F == 0b11100 {
		mnemonic = "ubfx"
	}
	lsb := (second>>12)&7<<2 | (second>>6)&3
	width := second&0x1F + 1
	return fmt.Sprintf("%s %s, %s, %s, %s", mnemonic, armReg((second>>8)&0xF), armReg(first&0xF),
		armImmediate32(int64(lsb)), armImmediate32(int64(width)))
}

// armThumb32BitfieldInsert renders BFI, which becomes BFC when its source is
// the unused register.
func armThumb32BitfieldInsert(first, second uint32) (string, bool) {
	rn, rd := first&0xF, (second>>8)&0xF
	lsb := (second>>12)&7<<2 | (second>>6)&3
	msb := second & 0x1F
	if msb < lsb {
		return "", false // a bitfield that ends before it begins
	}
	position := armImmediate32(int64(lsb))
	extent := armImmediate32(int64(msb - lsb + 1))
	if rn == 15 {
		return fmt.Sprintf("bfc %s, %s, %s", armReg(rd), position, extent), true
	}
	return fmt.Sprintf("bfi %s, %s, %s, %s", armReg(rd), armReg(rn), position, extent), true
}

// armThumb32LoadStoreSingle covers the third top-level group: single-item
// memory access, data processing on plain registers, the multiplies, and the
// coprocessor space.
func armThumb32LoadStoreSingle(first, second uint32) (string, bool) {
	op2 := (first >> 4) & 0x7F
	switch {
	case op2&0b1110000 == 0b0100000:
		return armThumb32DataRegister(first, second)
	case op2&0b1111000 == 0b0110000:
		return armThumb32Multiply(first, second)
	case op2&0b1111000 == 0b0111000:
		return armThumb32LongMultiply(first, second)
	case op2&0b0000001 == 0 || op2&0b0000111 <= 0b101:
		return armThumb32Memory(first, second)
	}
	return "", false
}

// armThumb32MemorySizes name the access widths by their size field.
var armThumb32MemorySizes = [3]string{"b", "h", ""}

// armThumb32Memory renders a single load or store.
func armThumb32Memory(first, second uint32) (string, bool) {
	mnemonic, ok := armThumb32MemoryMnemonic(first)
	if !ok {
		return "", false
	}
	rn, rt := first&0xF, (second>>12)&0xF

	switch {
	case rn == 15 && first&(1<<4) != 0: // a literal load names the program counter
		offset := int64(second & 0xFFF)
		if first&(1<<7) == 0 {
			offset = -offset
		}
		return mnemonic + ".w " + armReg(rt) + ", [pc, " + armImmediate32(offset) + "]", true

	case first&(1<<7) != 0: // twelve-bit positive offset
		return mnemonic + ".w " + armReg(rt) + ", " + armThumb32Offset(rn, int64(second&0xFFF)), true

	case second&(1<<11) != 0: // eight-bit signed offset, optionally written back
		return armThumb32IndexedMemory(mnemonic, rn, rt, second), true
	}
	return armThumb32RegisterOffsetMemory(mnemonic, rn, rt, second)
}

// armThumb32MemoryMnemonic names the access from its width and direction,
// rejecting the combinations that do not exist.
func armThumb32MemoryMnemonic(first uint32) (string, bool) {
	size := (first >> 5) & 3
	if size > 2 {
		return "", false
	}
	load := first&(1<<4) != 0
	signed := first&(1<<8) != 0
	if signed && (!load || size == 2) {
		return "", false // a signed store, or a signed word, does not exist
	}

	mnemonic := "str"
	if load {
		mnemonic = "ldr"
	}
	if signed {
		mnemonic += "s"
	}
	return mnemonic + armThumb32MemorySizes[size], true
}

// armThumb32RegisterOffsetMemory renders the shifted-register offset form. The
// bits above the shift must be clear, and the stack pointer cannot be the index.
func armThumb32RegisterOffsetMemory(mnemonic string, rn, rt, second uint32) (string, bool) {
	if (second>>6)&0x3F != 0 {
		return "", false
	}
	rm := second & 0xF
	if rm == 13 || rm == 15 {
		return "", false
	}

	operand := "[" + armReg(rn) + ", " + armReg(rm)
	if shift := (second >> 4) & 3; shift != 0 {
		operand += fmt.Sprintf(", lsl #%d", shift)
	}
	return mnemonic + ".w " + armReg(rt) + ", " + operand + "]", true
}

// armThumb32Offset renders a base register with an optional displacement.
func armThumb32Offset(rn uint32, offset int64) string {
	if offset == 0 {
		return "[" + armReg(rn) + "]"
	}
	return "[" + armReg(rn) + ", " + armImmediate32(offset) + "]"
}

// armThumb32IndexedMemory renders the pre- and post-indexed forms, which apply
// their displacement before or after the access and may write the result back.
func armThumb32IndexedMemory(mnemonic string, rn, rt, second uint32) string {
	offset := int64(second & 0xFF)
	if second&(1<<9) == 0 {
		offset = -offset
	}
	preIndexed := second&(1<<10) != 0
	writeback := second&(1<<8) != 0

	if !preIndexed {
		return fmt.Sprintf("%s %s, [%s], %s", mnemonic, armReg(rt), armReg(rn), armImmediate32(offset))
	}
	out := mnemonic + " " + armReg(rt) + ", " + armThumb32Offset(rn, offset)
	if writeback {
		out += "!"
	}
	return out
}

// armThumb32RegisterShifts are the register-controlled shift operations.
var armThumb32RegisterShifts = [4]string{"lsl", "lsr", "asr", "ror"}

// armThumb32DataRegister covers data processing on plain registers: the
// register-controlled shifts, the extends, and the bit-counting operations.
func armThumb32DataRegister(first, second uint32) (string, bool) {
	op1 := (first >> 4) & 0xF
	op2 := (second >> 4) & 0xF
	rn, rd, rm := first&0xF, (second>>8)&0xF, second&0xF

	if op2 == 0 && op1 <= 0b0111 { // register-controlled shift
		mnemonic := armThumb32Mnemonic(armThumb32RegisterShifts[(op1>>1)&3], op1&1 != 0, true)
		return fmt.Sprintf("%s %s, %s, %s", mnemonic, armReg(rd), armReg(rn), armReg(rm)), true
	}

	if op2&0b1000 != 0 { // extend, byte reversal or bit counting
		return armThumb32Miscellaneous(first, second)
	}
	return "", false
}

// armThumb32Extends name the sign- and zero-extending operations, which take a
// second source register in their accumulate form.
var armThumb32Extends = [8]string{"sxth", "uxth", "sxtb16", "uxtb16", "sxtb", "uxtb", "", ""}

// armThumb32MiscOps name the bit-counting and reversal operations.
var armThumb32MiscOps = [4]string{"rev.w", "rev16.w", "rbit", "revsh.w"}

// armThumb32Miscellaneous covers the extends, byte reversals and CLZ.
func armThumb32Miscellaneous(first, second uint32) (string, bool) {
	op1 := (first >> 4) & 0xF
	rn, rd, rm := first&0xF, (second>>8)&0xF, second&0xF

	if op1&0b1000 != 0 { // bit counting and reversal
		if op1 == 0b1011 && (second>>4)&0xF == 0b1000 {
			return fmt.Sprintf("clz %s, %s", armReg(rd), armReg(rm)), true
		}
		if op1 == 0b1001 {
			return fmt.Sprintf("%s %s, %s", armThumb32MiscOps[(second>>4)&3], armReg(rd), armReg(rm)), true
		}
		return "", false
	}

	name := armThumb32Extends[op1&7]
	// The nibble the encoding leaves unused must be set for this to be an extend.
	if name == "" || (second>>12)&0xF != 0xF {
		return "", false
	}

	rotation := (second >> 4) & 3
	operands := armReg(rd) + ", " + armReg(rm)
	if rn == 15 {
		name += ".w" // the plain extend has a 16-bit form of its own
	} else {
		// The accumulating form adds a base register.
		name = strings.Replace(name, "xt", "xta", 1)
		operands = armReg(rd) + ", " + armReg(rn) + ", " + armReg(rm)
	}
	if rotation != 0 {
		operands += fmt.Sprintf(", ror #%d", rotation*8)
	}
	return name + " " + operands, true
}

// armThumb32Multiply covers the 32-bit multiply and multiply-accumulate forms.
func armThumb32Multiply(first, second uint32) (string, bool) {
	rn, ra, rd, rm := first&0xF, (second>>12)&0xF, (second>>8)&0xF, second&0xF

	if (first>>4)&7 != 0 || (second>>4)&0xF > 1 {
		return "", false
	}
	if ra == 15 {
		return fmt.Sprintf("mul %s, %s, %s", armReg(rd), armReg(rn), armReg(rm)), true
	}
	mnemonic := "mla"
	if second&(1<<4) != 0 {
		mnemonic = "mls"
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", mnemonic, armReg(rd), armReg(rn), armReg(rm), armReg(ra)), true
}

// armThumb32LongMultiply covers the 64-bit multiplies and the divides, which
// share their encoding.
func armThumb32LongMultiply(first, second uint32) (string, bool) {
	op1 := (first >> 4) & 7
	op2 := (second >> 4) & 0xF
	rn, rdLo, rdHi, rm := first&0xF, (second>>12)&0xF, (second>>8)&0xF, second&0xF

	switch {
	case op1 == 0b000 && op2 == 0:
		return fmt.Sprintf("smull %s, %s, %s, %s", armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	case op1 == 0b010 && op2 == 0:
		return fmt.Sprintf("umull %s, %s, %s, %s", armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	case op1 == 0b100 && op2 == 0:
		return fmt.Sprintf("smlal %s, %s, %s, %s", armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	case op1 == 0b110 && op2 == 0:
		return fmt.Sprintf("umlal %s, %s, %s, %s", armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	case op1 == 0b001 && op2 == 0b1111:
		return fmt.Sprintf("sdiv %s, %s, %s", armReg(rdHi), armReg(rn), armReg(rm)), true
	case op1 == 0b011 && op2 == 0b1111:
		return fmt.Sprintf("udiv %s, %s, %s", armReg(rdHi), armReg(rn), armReg(rm)), true
	}
	return "", false
}

// armThumb32ListIsValid rejects the register lists the manual leaves
// unpredictable, which Capstone declines to decode: a store naming the program
// counter, a load naming both it and the link register, a list with fewer than
// two registers, and writeback to a register the list already contains.
func armThumb32ListIsValid(list uint32, load bool) bool {
	const pc = 1 << 15

	if !load && list&pc != 0 {
		return false // a store cannot name the program counter
	}
	return armPopCount(list) >= 2
}

// armPopCount counts the registers in a list.
func armPopCount(list uint32) int {
	n := 0
	for ; list != 0; list &= list - 1 {
		n++
	}
	return n
}

// armThumb32DualOrExclusive covers the paired loads and stores, the exclusive
// accesses, and the table branches, which share one encoding.
func armThumb32DualOrExclusive(first, second uint32) (string, bool) {
	op1 := (first >> 7) & 3
	op2 := (first >> 4) & 3
	rn := first & 0xF
	rt, rt2 := (second>>12)&0xF, (second>>8)&0xF

	// The two exclusive forms and the table branches only exist when neither
	// index bit is set.
	if op1 == 0b00 {
		switch op2 {
		case 0b00:
			return fmt.Sprintf("strex %s, %s, %s", armReg(rt2), armReg(rt),
				armThumb32Offset(rn, int64(second&0xFF)*4)), true
		case 0b01:
			return fmt.Sprintf("ldrex %s, %s", armReg(rt),
				armThumb32Offset(rn, int64(second&0xFF)*4)), true
		}
	}
	if op1 == 0b01 && op2 == 0b01 && (second>>4)&0xF <= 1 {
		return armThumb32TableBranch(first, second), true
	}

	return armThumb32DualTransfer(first, second, rn, rt, rt2)
}

// armThumb32TableBranch renders the byte and halfword table branches.
func armThumb32TableBranch(first, second uint32) string {
	if second&(1<<4) != 0 {
		return fmt.Sprintf("tbh [%s, %s, lsl #1]", armReg(first&0xF), armReg(second&0xF))
	}
	return fmt.Sprintf("tbb [%s, %s]", armReg(first&0xF), armReg(second&0xF))
}

// armThumb32DualTransfer renders the paired load and store, whose displacement
// is a word count and which can index before or after the access.
func armThumb32DualTransfer(first, second, rn, rt, rt2 uint32) (string, bool) {
	preIndexed := first&(1<<8) != 0
	writeback := first&(1<<5) != 0
	if !preIndexed && !writeback {
		return "", false // neither indexed nor written back: not a dual transfer
	}

	mnemonic := "strd"
	if first&(1<<4) != 0 {
		mnemonic = "ldrd"
	}
	offset := int64(second&0xFF) * 4
	if first&(1<<7) == 0 {
		offset = -offset
	}
	registers := armReg(rt) + ", " + armReg(rt2) + ", "

	if !preIndexed {
		return fmt.Sprintf("%s %s[%s], %s", mnemonic, registers, armReg(rn), armImmediate32(offset)), true
	}
	out := mnemonic + " " + registers + armThumb32Offset(rn, offset)
	if writeback {
		out += "!"
	}
	return out, true
}
