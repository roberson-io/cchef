package ops

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/bits"
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
		{Name: "Starting address (hex)", Type: core.ArgNumber, Integer: true, Value: float64(0)},
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
	var block armITBlock
	thumb := architecture == armArch32 && strings.HasPrefix(mode, "Thumb")

	for pos := 0; pos < len(code); {
		address := startAddress + uint64(pos)
		text, size, ok := armDecodeAt(code, pos, architecture, thumb, bigEndian, address)
		if !ok {
			// Capstone's linear sweep stops at the first instruction it cannot
			// decode rather than resynchronising further along.
			break
		}
		if thumb {
			text = block.apply(text, armThumbHalfword(code, pos, bigEndian))
			block.open(armThumbHalfword(code, pos, bigEndian), size)
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

// The IT instruction, whose top byte marks it and whose low nibble holds the
// pattern of then and else slots.
const (
	armThumbITOpcode = 0xBF
	armThumbITSlots  = 4
)

// armITBlock tracks the condition an IT instruction applies to the
// instructions that follow it. Capstone prints that condition on each of them
// in place of the flag-setting suffix their unconditional form carries.
type armITBlock struct {
	condition uint32
	pattern   uint32
	slot      int
	total     int
}

// open reads an IT instruction and starts the block it introduces.
func (b *armITBlock) open(halfword uint32, size int) {
	pattern := halfword & 0xF
	if size != armThumbUnit || halfword>>8 != armThumbITOpcode || pattern == 0 {
		return
	}
	b.condition = (halfword >> 4) & 0xF
	b.pattern = pattern
	b.slot = 1
	b.total = armThumbITSlots - bits.TrailingZeros32(pattern)
}

// apply gives the next instruction of the block its condition. The first slot
// always takes the condition itself; each one after that takes it or its
// opposite, according to the bit of the pattern that stands for it.
func (b *armITBlock) apply(text string, op uint32) string {
	if b.slot == 0 || b.slot > b.total {
		return text
	}
	condition := b.condition
	if b.slot > 1 && b.pattern>>uint(armThumbITSlots+1-b.slot)&1 != b.condition&1 {
		condition ^= 1
	}
	b.slot++

	if armThumbUnconditionalOps[armMnemonicOf(text)] || op&armThumbMoveMask == armThumbMoveOp {
		return text
	}
	// The condition goes on the operation itself, ahead of any wide marker. It
	// replaces both the flag-setting suffix the unconditional form carries and,
	// on a branch, whatever condition the branch named for itself.
	mnemonic, operands := armSplitMnemonic(text)
	base, wide, _ := strings.Cut(mnemonic, ".")
	name := strings.TrimSuffix(armWithoutCondition(base), "s") + armCondSuffix(condition)
	if wide != "" {
		name += "." + wide
	}
	return armJoin(name, operands)
}

// The narrow shift of nothing, which spells a register move and keeps its
// flag-setting form inside an IT block.
const (
	armThumbMoveMask = 0xFFC0
	armThumbMoveOp   = 0x0000
)

// armThumbUnconditionalOps never take the condition of the block they sit in.
var armThumbUnconditionalOps = map[string]bool{
	"cbz": true, "cbnz": true, "bkpt": true, "it": true,
	"udf": true, "trap": true, "setend": true, "cpsie": true, "cpsid": true,
}

// armMnemonicOf gives the operation an instruction names.
func armMnemonicOf(text string) string {
	mnemonic, _ := armSplitMnemonic(text)
	return mnemonic
}

// armWithoutCondition strips the condition a branch carries, which is the only
// mnemonic inside an IT block that already has one.
func armWithoutCondition(mnemonic string) string {
	if !strings.HasPrefix(mnemonic, "b") {
		return mnemonic
	}
	for _, name := range armConditionNames {
		if name != "" && mnemonic == "b"+name {
			return "b"
		}
	}
	return mnemonic
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

// The AArch64 calls into a higher privilege level, which sit in the exception
// group under the two link levels x/arch declines.
const (
	arm64ExceptionMask = 0xFFE0001C
	arm64ExceptionOp   = 0xD4000000
	arm64HypervisorEnd = 2
	arm64SecureEnd     = 3
)

// armExceptionCall64 renders those two calls.
func armExceptionCall64(raw uint32) (string, bool) {
	if raw&arm64ExceptionMask != arm64ExceptionOp {
		return "", false
	}
	name := ""
	switch raw & 3 {
	case arm64HypervisorEnd:
		name = "hvc"
	case arm64SecureEnd:
		name = "smc"
	default:
		return "", false
	}
	return name + " " + armImmediate32(int64((raw>>5)&0xFFFF)), true
}

// armDecodeOne decodes a single instruction word, reporting whether it decoded.
func armDecodeOne(word []byte, architecture string, address uint64) (string, bool) {
	if architecture == armArch64 {
		raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
		// The generic half of the system space is decoded here whatever x/arch
		// makes of it, since it numbers the top field one lower and spells the
		// coprocessor registers in capitals.
		if (raw>>19)&3 != 0 {
			if text, ok := armSystem64(raw); ok {
				return text, true
			}
		}
		if text, ok := armExceptionCall64(raw); ok {
			return text, true
		}
		inst, err := arm64asm.Decode(word)
		if err != nil {
			return armSystem64(raw)
		}
		if !armIsAllocated64(raw) {
			return "", false
		}
		return armFormat64(inst, word, address), true
	}
	text, ok := armDecodeARM32(word, address)
	if !ok {
		return "", false
	}
	return armSubtractedZero32(word, text), true
}

// The transfer classes that hold a displacement, and the bit that says whether
// to add or subtract it.
const (
	armAddsOffset      = 1 << 23
	armTransferMask    = 0x0E000000
	armWordTransfer    = 0x04000000 // a word or an unsigned byte
	armExtensionMove   = 0x0C000000 // an extension or coprocessor register
	armMiscellaneousOp = 0x00000000 // the class holding the extra transfers
)

// armSubtractedZero32 restores the sign on a displacement of zero the
// instruction subtracts. x/arch renders the displacement alone, which loses the
// distinction Capstone prints.
func armSubtractedZero32(word []byte, text string) string {
	raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
	if raw&armAddsOffset != 0 || strings.Contains(text, armNegativeZeroBare) {
		return text
	}
	zero := false
	switch raw & armTransferMask {
	case armWordTransfer:
		zero = raw&0xFFF == 0
	case armExtensionMove:
		zero = raw&0xFF == 0
	case armMiscellaneousOp:
		zero = raw&(1<<22) != 0 && (raw>>8)&0xF == 0 && raw&0xF == 0
	}
	if !zero {
		return text
	}
	return armNegativeZero(text)
}

// The two spellings Capstone gives a displacement of zero that is subtracted:
// bare, and with the hexadecimal prefix the Thumb pre-indexed forms carry.
const (
	armNegativeZeroBare = "#-0"
	armNegativeZeroHex  = "#-0x0"
)

// armNegativeZero writes the sign onto the displacement, which is either the
// last operand or the one inside the brackets, and which x/arch leaves out
// altogether where it is zero.
func armNegativeZero(text string) string {
	switch {
	case strings.HasSuffix(text, "], #0"):
		return strings.TrimSuffix(text, "#0") + armNegativeZeroBare
	case strings.HasSuffix(text, "]"):
		return strings.TrimSuffix(text, "]") + ", " + armNegativeZeroBare + "]"
	}
	return text
}

// The ARM hint space: the values below armNamedHints have names of their own,
// the top sixteen are the debug hints, and the rest are numbered.
const (
	armHintMask     = 0x0FFFFF00
	armHintOp       = 0x0320F000
	armDebugHintTop = 0xF0
)

// armHint32 renders a hint, which carries a condition like anything else.
func armHint32(raw uint32) (string, bool) {
	cond := raw >> 28
	if cond == 15 || raw&armHintMask != armHintOp {
		return "", false
	}
	return armHint(raw&0xFF, armCondSuffix(cond), ""), true
}

// armHint renders one from its value, given the suffixes to carry.
func armHint(value uint32, condition, wide string) string {
	if int(value) < len(armThumbHints) {
		return armThumbHints[value] + condition + wide
	}
	if value >= armDebugHintTop {
		return "dbg" + condition + " " + armImmediate32(int64(value&0xF))
	}
	return "hint" + condition + wide + " " + armImmediate32(int64(value))
}

// The ARM barriers, and the shareability domains eight of their sixteen values
// name. An instruction barrier names only the whole system, and every value
// they do not name is numbered in hexadecimal whatever its size.
const (
	armBarrierMask   = 0xFFFFFFF0
	armBarrierOp     = 0xF57FF040
	armBarrierSystem = 0xF
)

var armBarrierNames = [3]string{"dsb", "dmb", "isb"}

var armBarrierDomains = map[uint32]string{
	0x2: "oshst", 0x3: "osh", 0x6: "nshst", 0x7: "nsh",
	0xA: "ishst", 0xB: "ish", 0xE: "st", 0xF: "sy",
}

// armBarrier32 renders an ARM barrier.
func armBarrier32(raw uint32) (string, bool) {
	if raw&armBarrierMask != armBarrierOp && raw&armBarrierMask != armBarrierOp+0x10 &&
		raw&armBarrierMask != armBarrierOp+0x20 {
		return "", false
	}
	return armBarrier((raw>>4)&3, raw&0xF), true
}

// armBarrier renders one from its kind and the domain it applies to.
func armBarrier(kind, domain uint32) string {
	name := armBarrierNames[kind]
	if named, ok := armBarrierDomains[domain]; ok && (kind != 2 || domain == armBarrierSystem) {
		return name + " " + named
	}
	return fmt.Sprintf("%s #0x%x", name, domain)
}

// The permanently undefined ARM encoding, which carries the always condition
// and no other, and joins the two halves of its immediate.
const (
	armUndefinedMask = 0xFFF000F0
	armUndefinedOp   = 0xE7F000F0
)

// armUndefined32 renders it.
func armUndefined32(raw uint32) (string, bool) {
	if raw&armUndefinedMask != armUndefinedOp {
		return "", false
	}
	return "udf " + armImmediate32(int64((raw>>4)&0xFFF0|raw&0xF)), true
}

// armDecodeARM32 decodes a single ARM word.
func armDecodeARM32(word []byte, address uint64) (string, bool) {
	raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
	// The status register transfers are decoded here whatever x/arch makes of
	// them, since it names some of the field masks differently.
	if text, ok := armUndefined32(raw); ok {
		return text, true
	}
	if text, ok := armHint32(raw); ok {
		return text, true
	}
	if text, ok := armBarrier32(raw); ok {
		return text, true
	}
	if text, ok := armStatusRegister32(raw); ok {
		return text, true
	}
	if text, ok := armBitfield32(raw); ok {
		return text, true
	}
	if text, ok := armHypervisorCall32(raw); ok {
		return text, true
	}
	inst, err := armasm.Decode(word, armasm.ModeARM)
	if err != nil {
		// Several families carry a field the manual fixes but Capstone reads
		// past; x/arch insists on it, so the word is offered again with the
		// field set to what it expects.
		if canonical, ok := armCanonicalFields32(raw); ok {
			// #nosec G115 -- each conversion takes one byte of the word by design
			word = []byte{byte(canonical), byte(canonical >> 8), byte(canonical >> 16), byte(canonical >> 24)}
			if inst, err = armasm.Decode(word, armasm.ModeARM); err == nil {
				return armFormat32(inst, word, address), true
			}
		}
		// x/arch declines several ARM families, which are decoded here.
		return armDecodeARM32Extra(raw)
	}
	if !armVFPCompareZeroIsValid(raw) || !armMoveNotIsValid(raw) || !armDoublewordIsValid(raw) {
		return "", false
	}
	return armFormat32(inst, word, address), true
}

// The floating-point compare against zero, and the low bits Capstone insists
// are clear even though the manual leaves them unspecified.
const (
	armVFPCompareZeroOpcode   = 0b0101
	armVFPCompareZeroReserved = 0x3F
)

// The data-processing opcode that inverts its operand, which leaves no room
// for a first source register.
const armMoveNotOpcode = 0b1111

// armMoveNotIsValid rejects a bitwise-not whose unused source register field is
// set. x/arch ignores it.
func armMoveNotIsValid(raw uint32) bool {
	// Both marker bits set below an all-zero top field is the multiply and
	// extra load/store space, which reuses the opcode field for something else.
	extra := raw&(1<<25) == 0 && raw&(1<<7) != 0 && raw&(1<<4) != 0
	dataProcessing := (raw>>26)&3 == 0 && !extra && (raw>>21)&0xF == armMoveNotOpcode
	return !dataProcessing || (raw>>16)&0xF == 0
}

// The doubleword transfers of the extra load/store group, which name a pair
// starting at the register the encoding gives.
const (
	armDoublewordMask = 0x0E1000D0
	armDoublewordOp   = 0x000000D0
)

// armDoublewordIsValid rejects a doubleword transfer whose pair would run past
// the last register. x/arch prints it anyway.
func armDoublewordIsValid(raw uint32) bool {
	if raw&armDoublewordMask != armDoublewordOp {
		return true
	}
	return (raw>>12)&0xF != armProgramCounter
}

// armVFPCompareZeroIsValid reports whether a word is either not a compare
// against zero or one whose reserved bits are clear. x/arch ignores them.
func armVFPCompareZeroIsValid(raw uint32) bool {
	compare := (raw>>24)&0xF == 0b1110 &&
		(raw>>20)&armVFPOpcodeMask == armVFPExtendedOpcode &&
		(raw>>16)&0xF == armVFPCompareZeroOpcode &&
		(raw>>9)&7 == 0b101 &&
		raw&(1<<6) != 0 && raw&(1<<4) == 0
	return !compare || raw&armVFPCompareZeroReserved == 0
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
func armRewriteOperands(s string, bare, offset, postIndex func(int64) string) string {
	return armRewriteOperandsWith(s, bare, offset, postIndex, false)
}

// armRewriteOperandsWith does the same, but can be told that this instruction's
// shift amount is printed as an ordinary immediate rather than as part of a
// shifted operand, which turns it to hexadecimal above nine.
func armRewriteOperandsWith(s string, bare, offset, postIndex func(int64) string, shiftIsImmediate bool) string {
	var b strings.Builder
	depth, closed := 0, false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '[':
			depth++
		case ']':
			depth--
			closed = true // what follows is a post-indexed displacement
		case '#':
			value, width, ok := armParseImmediate(s[i:])
			if !ok {
				break
			}
			switch {
			case armFollowsShift(s[:i]) && !shiftIsImmediate:
				b.WriteString("#" + strconv.FormatInt(value, 10))
			case depth > 0:
				b.WriteString(offset(value))
			case closed:
				b.WriteString(postIndex(value))
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

// arm64ConditionSpellings are the names x/arch gives an AArch64 condition
// operand: the two carry conditions are named after the flag, and the never
// condition borrows the always one, which only the encoding tells apart.
var arm64ConditionSpellings = [16]string{
	"eq", "ne", "cs", "cc", "mi", "pl", "vs", "vc",
	"hi", "ls", "ge", "lt", "gt", "le", "al", "al",
}

// arm64ConditionNames are the names Capstone prints for the same, which unlike
// ARM's leaves no slot unspelt.
var arm64ConditionNames = [16]string{
	"eq", "ne", "hs", "lo", "mi", "pl", "vs", "vc",
	"hi", "ls", "ge", "lt", "gt", "le", "al", "nv",
}

// arm64ConditionShift is where the conditional selects and comparisons hold the
// condition they test.
const arm64ConditionShift = 12

// The logical immediate that moves a constant through the zero register, which
// Capstone spells in full where x/arch prints the move alias.
const (
	arm64ConstantMoveMask = 0x7F800000
	arm64ConstantMoveOp   = 0x32000000
)

// armConstantMove64 restores the full spelling of such a move.
func armConstantMove64(mnemonic, operands string, raw uint32) (string, bool) {
	if raw&arm64ConstantMoveMask != arm64ConstantMoveOp || mnemonic != "mov" {
		return "", false
	}
	destination, constant, found := strings.Cut(operands, ", ")
	if !found {
		return "", false
	}
	zero := "wzr"
	if raw&(1<<31) != 0 {
		zero = "xzr"
	}
	return "orr " + destination + ", " + zero + ", " + constant, true
}

// armHexImmediateOps hold a constant Capstone always prints in hexadecimal,
// and armDecimalShiftOps a shift it always prints in decimal.
var (
	armHexImmediateOps = map[string]bool{"movi": true, "mvni": true}
	armDecimalShiftOps = map[string]bool{"shll": true, "shll2": true}
)

// armHexImmediate renders a constant the way Capstone's hexadecimal printer
// does, which leaves zero without the prefix its format would otherwise add.
func armHexImmediate(value int64) string {
	if value == 0 {
		return "#0"
	}
	return fmt.Sprintf("#0x%x", value)
}

// armZeroComparisons are the comparisons against zero, whose constant Capstone
// prints as the floating-point value it stands for.
var armZeroComparisons = map[string]bool{
	"fcmgt": true, "fcmge": true, "fcmeq": true, "fcmle": true, "fcmlt": true,
}

// armExtendAliases are the widening shifts x/arch spells as extends, which it
// may do only where they shift by nothing.
var armExtendAliases = map[string]string{
	"sxtl": "sshll", "sxtl2": "sshll2", "uxtl": "ushll", "uxtl2": "ushll2",
}

// armSpeltInFull64 restores the spellings x/arch shortens: the comparisons
// against zero drop the fraction from their constant, and a widening shift of
// nothing is written as an extend.
func armSpeltInFull64(mnemonic, operands string) (string, string, bool) {
	if mnemonic == "hint" {
		value, _, ok := armParseImmediate(operands)
		if !ok {
			return "", "", false
		}
		return mnemonic, fmt.Sprintf("#0x%x", value), true
	}
	if armHexImmediateOps[mnemonic] || armDecimalShiftOps[mnemonic] {
		cut := strings.LastIndex(operands, ", #")
		if cut < 0 {
			return "", "", false
		}
		value, _, ok := armParseImmediate(operands[cut+2:])
		if !ok {
			return "", "", false
		}
		kept := operands[:cut+2]
		if armDecimalShiftOps[mnemonic] {
			return mnemonic, kept + fmt.Sprintf("#%d", value), true
		}
		return mnemonic, kept + armHexImmediate(value), true
	}
	if armZeroComparisons[mnemonic] && strings.HasSuffix(operands, ", #0") {
		return mnemonic, operands + ".0", true
	}
	if shift, ok := armExtendAliases[mnemonic]; ok {
		return shift, operands + ", #0", true
	}
	return "", "", false
}

// armConditionOperand64 gives a trailing condition operand Capstone's spelling.
func armConditionOperand64(operands string, raw uint32) string {
	cut := strings.LastIndex(operands, ", ")
	if cut < 0 {
		return operands
	}
	cond := (raw >> arm64ConditionShift) & 0xF
	if operands[cut+2:] != arm64ConditionSpellings[cond] {
		return operands
	}
	return operands[:cut+2] + arm64ConditionNames[cond]
}

// armTypeSuffixes are the operand-type markers x/arch appends to an operation
// name. They stay behind a dot, unlike the flag and condition markers that share
// the same dotted spelling; FXS and FXU name the signed and unsigned fixed-point
// forms, which Capstone prints as a bare .s and .u.
var armTypeSuffixes = map[string]string{
	"f16": ".f16", "f32": ".f32", "f64": ".f64",
	"s32": ".s32", "u32": ".u32",
	"fxs16": ".s16", "fxs32": ".s32",
	"fxu16": ".u16", "fxu32": ".u32",
	"32": ".32",
}

// armMnemonic32 builds the mnemonic from the decoded operation so the flag and
// condition suffixes carry Capstone's spelling rather than x/arch's.
func armMnemonic32(inst armasm.Inst) string {
	parts := strings.Split(strings.ToLower(inst.Op.String()), ".")
	var out strings.Builder
	out.WriteString(parts[0])
	for _, part := range parts[1:] {
		if suffix, ok := armTypeSuffixes[part]; ok {
			out.WriteString(suffix)
			continue
		}
		if mapped, ok := armConditions[part]; ok {
			part = mapped
		}
		out.WriteString(part)
	}
	return out.String()
}

// Widths of the two floating-point formats an eight-bit VFP immediate can
// expand into.
const (
	armFloat32ExponentBits = 8
	armFloat32FractionBits = 23
	armFloat64ExponentBits = 11
	armFloat64FractionBits = 52
	armVFPImmFractionBits  = 4 // the fraction the immediate itself carries
)

// armVFPExpandImm reconstructs the constant an eight-bit VFP immediate stands
// for. Bit 7 is the sign; bit 6 is inverted to form the top exponent bit and
// then repeated to fill the rest; bits 5 and 4 finish the exponent and bits 3
// to 0 are the leading fraction.
func armVFPExpandImm(imm8 uint32, wide bool) float64 {
	sign := imm8 >> 7 & 1
	lead := imm8 >> 6 & 1
	tail := imm8 >> 4 & 3
	fraction := imm8 & 0xF

	exponentBits, fractionBits := uint(armFloat32ExponentBits), uint(armFloat32FractionBits)
	if wide {
		exponentBits, fractionBits = armFloat64ExponentBits, armFloat64FractionBits
	}
	repeated := exponentBits - 3 // the width bit 6 is repeated across
	exponent := uint64(lead^1)<<(exponentBits-1) |
		uint64(armRepeatBit(lead, repeated))<<2 |
		uint64(tail)

	bits := uint64(sign)<<(exponentBits+fractionBits) |
		exponent<<fractionBits |
		uint64(fraction)<<(fractionBits-armVFPImmFractionBits)
	if wide {
		return math.Float64frombits(bits)
	}
	return float64(math.Float32frombits(uint32(bits))) // #nosec G115 -- the narrow layout fills 32 bits
}

// armRepeatBit returns count copies of bit.
func armRepeatBit(bit uint32, count uint) uint32 {
	return bit * (1<<count - 1)
}

// armVFPMoveImmediate rewrites the raw eight-bit operand of a floating-point
// move-immediate as the constant it encodes, which is how Capstone prints it.
func armVFPMoveImmediate(inst armasm.Inst, mnemonic, operands string) string {
	wide := strings.HasSuffix(mnemonic, ".f64")
	if !strings.HasPrefix(mnemonic, "vmov") || (!wide && !strings.HasSuffix(mnemonic, ".f32")) {
		return operands
	}
	imm, ok := inst.Args[1].(armasm.Imm)
	if !ok {
		return operands
	}
	register, _, found := strings.Cut(operands, ",")
	if !found {
		return operands
	}
	return register + ", #" + strconv.FormatFloat(armVFPExpandImm(uint32(imm), wide), 'e', 6, 64)
}

// armVFPZeroDisplacement restores the sign Capstone prints on a floating-point
// displacement of minus zero, which x/arch drops along with the offset itself.
func armVFPZeroDisplacement(word []byte, mnemonic, operands string) string {
	if !strings.HasPrefix(mnemonic, "vldr") && !strings.HasPrefix(mnemonic, "vstr") {
		return operands
	}
	raw := uint32(word[0]) | uint32(word[1])<<8 | uint32(word[2])<<16 | uint32(word[3])<<24
	if raw&(1<<23) != 0 || raw&0xFF != 0 || !strings.HasSuffix(operands, "]") {
		return operands
	}
	return strings.TrimSuffix(operands, "]") + ", #-0]"
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

// armGNURegisterAliases are the role names Capstone prints for the high general
// registers; x/arch uses them in register lists but not in every operand.
var armGNURegisterAliases = []struct {
	number string
	alias  string
}{
	{"r9", "sb"}, {"r10", "sl"}, {"r11", "fp"}, {"r12", "ip"},
}

// armAliasRegisters rewrites the numbered high registers to the names Capstone
// uses, leaving anything they appear inside untouched.
func armAliasRegisters(operands string) string {
	for _, r := range armGNURegisterAliases {
		operands = armReplaceWord(operands, r.number, r.alias)
	}
	return operands
}

// armReplaceWord substitutes whole words only, so r1 is not found inside r10.
func armReplaceWord(s, from, to string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], from) && !armIsWordByte(s, i-1) && !armIsWordByte(s, i+len(from)) {
			b.WriteString(to)
			i += len(from)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// armIsWordByte reports whether the byte at i continues an identifier.
func armIsWordByte(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	c := s[i]
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// armLooseField describes one family whose fixed field Capstone reads past:
// the bits that identify the family, the value they take, and the field to
// restore before offering the word to x/arch again.
type armLooseField struct {
	mask, value, field, canonical uint32
}

// armLooseFields are the families that carry such a field.
var armLooseFields = []armLooseField{
	{0x0FF000F0, 0x068000B0, 0x00000F00, 0x00000F00}, // the byte selection
	{0x0F900090, 0x01000080, 0x0000F000, 0},          // the signed halfword multiplies
	{0x0FB000F0, 0x01000090, 0x00000F00, 0},          // the swaps
	{0x0F8000F0, 0x06800070, 0x00000300, 0},          // the accumulating extends
	{0x0F9000F0, 0x01000050, 0x00000F00, 0},          // the saturating arithmetic
}

// The call to the hypervisor, which x/arch has no table for. Its operand is
// split across the two halves of the word.
const (
	armHypervisorMask = 0x0FF000F0
	armHypervisorOp   = 0x01400070
)

// armHypervisorCall32 renders HVC.
func armHypervisorCall32(raw uint32) (string, bool) {
	cond := raw >> 28
	if raw&armHypervisorMask != armHypervisorOp || cond == 15 {
		return "", false
	}
	value := (raw>>8)&0xFFF<<4 | raw&0xF
	return "hvc " + armImmediate32(int64(value)), true
}

// armCanonicalFields32 restores the fixed field of whichever family a word
// belongs to, reporting whether it belonged to one at all.
func armCanonicalFields32(raw uint32) (uint32, bool) {
	if raw>>28 == 15 {
		return 0, false
	}
	for _, family := range armLooseFields {
		if raw&family.mask == family.value && raw&family.field != family.canonical {
			return raw&^family.field | family.canonical, true
		}
	}
	return 0, false
}

// armImmediateShiftOps are the operations whose shifted operand Capstone
// prints as an ordinary immediate rather than as a shift amount, so that it
// turns to hexadecimal above nine. The key is the mnemonic without its
// condition suffix.
var armImmediateShiftOps = []string{"pkhbt", "pkhtb", "ssat", "usat"}

// armHasImmediateShift reports whether a mnemonic belongs to that group.
func armHasImmediateShift(mnemonic string) bool {
	for _, name := range armImmediateShiftOps {
		if strings.HasPrefix(mnemonic, name) {
			return true
		}
	}
	return false
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
	operands = armAliasRegisters(armRewriteOperandsWith(armSpaceAfterCommas(operands),
		armImmediate32Bare, armImmediate32, armImmediate32, armHasImmediateShift(mnemonic)))
	operands = armRotatedImmediate32(inst, operands)
	operands = armPairedTransfer32(mnemonic, inst, operands)
	operands = armVFPMoveImmediate(inst, mnemonic, operands)
	operands = armVFPZeroDisplacement(word, mnemonic, operands)
	// Reading the flags out of the floating-point status register names the
	// application status register, which Capstone spells in mixed case.
	operands = armReplaceWord(operands, "apsr_nzcv", "APSR_nzcv")
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
	arm64MoveWideMask = 0x1F800000
	arm64MoveWideOp   = 0x12800000 // the move-wide family, less its two opcode bits
)

// arm64MoveWideNames are the three move-wide operations, in the order their
// opcode bits number them.
var arm64MoveWideNames = [4]string{"movn", "", "movz", "movk"}

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
	if wide, ok := armMoveWide64(raw); ok {
		return wide
	}

	// The pair transfers keep the sign on a displacement written back after the
	// access; every other form wraps it round to a 64-bit unsigned value.
	postIndex := armUnsignedImmediate64
	if armPairTransfers64[strings.TrimSuffix(mnemonic, ".w")] {
		postIndex = armImmediate32
	}
	operands = armRewriteOperands(armSpaceAfterCommas(operands), armUnsignedImmediate64, armImmediate32, postIndex)
	operands = armConditionOperand64(operands, raw)
	if constant, ok := armConstantMove64(mnemonic, operands, raw); ok {
		return constant
	}
	if mnemonic, operands, ok := armSpeltInFull64(mnemonic, operands); ok {
		return armJoin(mnemonic, operands)
	}
	if inserted, ok := armBitfieldInsert64(mnemonic, raw); ok {
		return inserted
	}
	return armJoin(armMnemonic64(mnemonic, raw), armLaneIndexes64(armExpandVectorList(operands)))
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
// armProgramCounter is the number the program counter answers to in a register
// field; several encodings give it a meaning of its own.
const armProgramCounter = 15

var armRegNames = [16]string{
	"r0", "r1", "r2", "r3", "r4", "r5", "r6", "r7",
	"r8", "sb", "sl", "fp", "ip", "sp", "lr", "pc",
}

// The three AArch64 encodings x/arch reads more loosely than Capstone does.
const (
	arm64ExtendMask  = 0x1FE00000 // add and subtract, extending the second source
	arm64ExtendOp    = 0x0B200000
	arm64WidestShift = 4

	arm64PairMask     = 0x3A000000 // the transfers that move a register pair
	arm64PairOp       = 0x28000000
	arm64PairPreIndex = 3 // the addressing modes that write the base back
	arm64PairPostIdx  = 1
	arm64StackPointer = 31

	arm64LogicalMask = 0x1F800000 // the logical operations on an immediate
	arm64LogicalOp   = 0x12000000
	arm64PatternBits = 6
)

// armIsAllocated64 applies the checks x/arch leaves out. It shifts an extend by
// no more than four, never loads a pair into one register twice, never writes a
// base back through a register the same instruction transfers, and refuses a
// logical immediate whose bit pattern is entirely ones.
func armIsAllocated64(raw uint32) bool {
	switch {
	case raw&arm64ExtendMask == arm64ExtendOp:
		return (raw>>10)&7 <= arm64WidestShift
	case raw&arm64LoadMask == arm64LoadOp:
		return armLoadKeepsItsBase64(raw)
	case raw&arm64InsertElementMask == arm64InsertElementOp:
		return armInsertLaneIsValid64(raw)
	case raw&arm64DuplicateMask == arm64DuplicateOp:
		return armDuplicateSizeIsValid64(raw)
	case raw&arm64ExclusivePairMask == arm64ExclusivePairOp:
		return raw&0x1F != (raw>>10)&0x1F
	case raw&arm64PairMask == arm64PairOp:
		return armPairRegistersAgree64(raw)
	case raw&arm64LogicalMask == arm64LogicalOp:
		return armLogicalPatternIsValid64(raw)
	}
	return true
}

// The integer loads that take a displacement of their own and hold their result
// in a 32-bit register, whose three indexed addressing modes share a decoder.
// The unscaled mode, which is the fourth, carries no constraint.
const (
	arm64LoadMask     = 0x3F600000
	arm64LoadOp       = 0x38400000
	arm64UnscaledMode = 0
)

// The exclusive pair load, whose two destinations must differ. The store in the
// same family reads both registers, so it may name one twice.
const (
	arm64ExclusivePairMask = 0x3FE00000
	arm64ExclusivePairOp   = 0x08600000
)

// The duplication of a general register across a vector, whose selector names
// one element size rather than an index into one.
const (
	arm64DuplicateMask = 0xBFE0FC00
	arm64DuplicateOp   = 0x0E000C00
)

// armDuplicateSizeIsValid64 checks that selector, which carries a single bit
// below the one that would make it an index.
func armDuplicateSizeIsValid64(raw uint32) bool {
	selector := (raw >> 16) & 0x1F
	return selector != 0 && selector < 16 && selector&(selector-1) == 0
}

// armInsertLaneIsValid64 checks the lane an element insertion reads. The index
// is scaled by the element size the selector names, so the bits below it are
// clear, and a selector naming no size at all is unallocated.
func armInsertLaneIsValid64(raw uint32) bool {
	selector := (raw >> 16) & 0xF
	if selector == 0 {
		return false
	}
	size := bits.TrailingZeros32(selector)
	return (raw>>11)&0xF&(1<<size-1) == 0
}

// armLoadKeepsItsBase64 reports whether a load leaves the register holding its
// address alone. Loading into that register is refused, except where it is the
// stack pointer, which the encoding names apart from the zero register.
func armLoadKeepsItsBase64(raw uint32) bool {
	if (raw>>10)&3 == arm64UnscaledMode {
		return true
	}
	rt, rn := raw&0x1F, (raw>>5)&0x1F
	return rn == arm64StackPointer || rt != rn
}

// armPairRegistersAgree64 checks the register pairings of a pair transfer.
func armPairRegistersAgree64(raw uint32) bool {
	rt, rn, rt2 := raw&0x1F, (raw>>5)&0x1F, (raw>>10)&0x1F
	if raw&(1<<22) != 0 && rt == rt2 {
		return false // a load would write one register twice
	}
	index := (raw >> 23) & 3
	if index != arm64PairPreIndex && index != arm64PairPostIdx {
		return true // nothing is written back, so the base may be transferred
	}
	if raw&(1<<26) != 0 {
		return true // a floating-point pair holds registers of its own
	}
	return rn == arm64StackPointer || (rn != rt && rn != rt2)
}

// armLogicalPatternIsValid64 reproduces the architecture's own decoding of a
// logical immediate far enough to tell an allocated pattern from a reserved
// one: the width of the repeating element comes from the highest bit set in the
// selector, and a pattern of every bit within it is not encodable.
func armLogicalPatternIsValid64(raw uint32) bool {
	wide := raw&(1<<31) != 0
	replicate := (raw >> 22) & 1
	pattern := (raw >> 10) & 0x3F
	if !wide && replicate != 0 {
		return false // the widest element has no 32-bit form
	}

	width := bits.Len32(replicate<<arm64PatternBits | ^pattern&0x3F)
	if width < 2 {
		return false
	}
	ones := uint32(1)<<(width-1) - 1
	return pattern&ones != ones
}

// The AArch64 system space, whose top field Capstone always prints as three
// however the encoding numbers it, and the two selectors that pick between a
// register access and a system operation.
const (
	arm64SystemMask      = 0xFFC00000
	arm64SystemOp        = 0xD5000000
	arm64SystemOperation = 1 // in bits 20 and 19; every other value is a register
	arm64SystemTopField  = 3
)

// arm64SystemAlias is one named system operation.
type arm64SystemAlias struct {
	name     string
	operand  string
	register bool
}

// armSystemRegisterName64 names a system register, which Capstone does for each
// direction separately: one that cannot be written keeps its number there.
func armSystemRegisterName64(key uint32, read bool) string {
	direction := arm64WriteOnlyRegisters
	if read {
		direction = arm64ReadOnlyRegisters
	}
	if name, ok := direction[key]; ok {
		return name
	}
	return arm64SystemRegisters[key]
}

// arm64SystemRegisters name the system registers Capstone spells out, keyed by
// the five selector fields packed from the top one down. x/arch prints every
// one of them by number.
var arm64SystemRegisters = map[uint32]string{
	0x0002: "osdtrrx_el1",
	0x0004: "dbgbvr0_el1",
	0x0005: "dbgbcr0_el1",
	0x0006: "dbgwvr0_el1",
	0x0007: "dbgwcr0_el1",
	0x000c: "dbgbvr1_el1",
	0x000d: "dbgbcr1_el1",
	0x000e: "dbgwvr1_el1",
	0x000f: "dbgwcr1_el1",
	0x0010: "mdccint_el1",
	0x0012: "mdscr_el1",
	0x0014: "dbgbvr2_el1",
	0x0015: "dbgbcr2_el1",
	0x0016: "dbgwvr2_el1",
	0x0017: "dbgwcr2_el1",
	0x001a: "osdtrtx_el1",
	0x001c: "dbgbvr3_el1",
	0x001d: "dbgbcr3_el1",
	0x001e: "dbgwvr3_el1",
	0x001f: "dbgwcr3_el1",
	0x0024: "dbgbvr4_el1",
	0x0025: "dbgbcr4_el1",
	0x0026: "dbgwvr4_el1",
	0x0027: "dbgwcr4_el1",
	0x002c: "dbgbvr5_el1",
	0x002d: "dbgbcr5_el1",
	0x002e: "dbgwvr5_el1",
	0x002f: "dbgwcr5_el1",
	0x0032: "oseccr_el1",
	0x0034: "dbgbvr6_el1",
	0x0035: "dbgbcr6_el1",
	0x0036: "dbgwvr6_el1",
	0x0037: "dbgwcr6_el1",
	0x003c: "dbgbvr7_el1",
	0x003d: "dbgbcr7_el1",
	0x003e: "dbgwvr7_el1",
	0x003f: "dbgwcr7_el1",
	0x0044: "dbgbvr8_el1",
	0x0045: "dbgbcr8_el1",
	0x0046: "dbgwvr8_el1",
	0x0047: "dbgwcr8_el1",
	0x004c: "dbgbvr9_el1",
	0x004d: "dbgbcr9_el1",
	0x004e: "dbgwvr9_el1",
	0x004f: "dbgwcr9_el1",
	0x0054: "dbgbvr10_el1",
	0x0055: "dbgbcr10_el1",
	0x0056: "dbgwvr10_el1",
	0x0057: "dbgwcr10_el1",
	0x005c: "dbgbvr11_el1",
	0x005d: "dbgbcr11_el1",
	0x005e: "dbgwvr11_el1",
	0x005f: "dbgwcr11_el1",
	0x0064: "dbgbvr12_el1",
	0x0065: "dbgbcr12_el1",
	0x0066: "dbgwvr12_el1",
	0x0067: "dbgwcr12_el1",
	0x006c: "dbgbvr13_el1",
	0x006d: "dbgbcr13_el1",
	0x006e: "dbgwvr13_el1",
	0x006f: "dbgwcr13_el1",
	0x0074: "dbgbvr14_el1",
	0x0075: "dbgbcr14_el1",
	0x0076: "dbgwvr14_el1",
	0x0077: "dbgwcr14_el1",
	0x007c: "dbgbvr15_el1",
	0x007d: "dbgbcr15_el1",
	0x007e: "dbgwvr15_el1",
	0x007f: "dbgwcr15_el1",
	0x009c: "osdlr_el1",
	0x00a4: "dbgprcr_el1",
	0x03c6: "dbgclaimset_el1",
	0x03ce: "dbgclaimclr_el1",
	0x0801: "trctraceidr",
	0x0802: "trcvictlr",
	0x0804: "trcseqevr0",
	0x0805: "trccntrldvr0",
	0x0807: "trcimspec0",
	0x0808: "trcprgctlr",
	0x0809: "trcqctlr",
	0x080a: "trcviiectlr",
	0x080c: "trcseqevr1",
	0x080d: "trccntrldvr1",
	0x080f: "trcimspec1",
	0x0810: "trcprocselr",
	0x0812: "trcvissctlr",
	0x0814: "trcseqevr2",
	0x0815: "trccntrldvr2",
	0x0817: "trcimspec2",
	0x081a: "trcvipcssctlr",
	0x081d: "trccntrldvr3",
	0x081f: "trcimspec3",
	0x0820: "trcconfigr",
	0x0825: "trccntctlr0",
	0x0827: "trcimspec4",
	0x082d: "trccntctlr1",
	0x082f: "trcimspec5",
	0x0830: "trcauxctlr",
	0x0834: "trcseqrstevr",
	0x0835: "trccntctlr2",
	0x0837: "trcimspec6",
	0x083c: "trcseqstr",
	0x083d: "trccntctlr3",
	0x083f: "trcimspec7",
	0x0840: "trceventctl0r",
	0x0842: "trcvdctlr",
	0x0844: "trcextinselr",
	0x0845: "trccntvr0",
	0x0848: "trceventctl1r",
	0x084a: "trcvdsacctlr",
	0x084d: "trccntvr1",
	0x0852: "trcvdarcctlr",
	0x0855: "trccntvr2",
	0x0858: "trcstallctlr",
	0x085d: "trccntvr3",
	0x0860: "trctsctlr",
	0x0868: "trcsyncpr",
	0x0870: "trcccctlr",
	0x0878: "trcbbctlr",
	0x0881: "trcrsctlr16",
	0x0882: "trcssccr0",
	0x0883: "trcsspcicr0",
	0x0889: "trcrsctlr17",
	0x088a: "trcssccr1",
	0x088b: "trcsspcicr1",
	0x0890: "trcrsctlr2",
	0x0891: "trcrsctlr18",
	0x0892: "trcssccr2",
	0x0893: "trcsspcicr2",
	0x0898: "trcrsctlr3",
	0x0899: "trcrsctlr19",
	0x089a: "trcssccr3",
	0x089b: "trcsspcicr3",
	0x08a0: "trcrsctlr4",
	0x08a1: "trcrsctlr20",
	0x08a2: "trcssccr4",
	0x08a3: "trcsspcicr4",
	0x08a4: "trcpdcr",
	0x08a8: "trcrsctlr5",
	0x08a9: "trcrsctlr21",
	0x08aa: "trcssccr5",
	0x08ab: "trcsspcicr5",
	0x08b0: "trcrsctlr6",
	0x08b1: "trcrsctlr22",
	0x08b2: "trcssccr6",
	0x08b3: "trcsspcicr6",
	0x08b8: "trcrsctlr7",
	0x08b9: "trcrsctlr23",
	0x08ba: "trcssccr7",
	0x08bb: "trcsspcicr7",
	0x08c0: "trcrsctlr8",
	0x08c1: "trcrsctlr24",
	0x08c2: "trcsscsr0",
	0x08c8: "trcrsctlr9",
	0x08c9: "trcrsctlr25",
	0x08ca: "trcsscsr1",
	0x08d0: "trcrsctlr10",
	0x08d1: "trcrsctlr26",
	0x08d2: "trcsscsr2",
	0x08d8: "trcrsctlr11",
	0x08d9: "trcrsctlr27",
	0x08da: "trcsscsr3",
	0x08e0: "trcrsctlr12",
	0x08e1: "trcrsctlr28",
	0x08e2: "trcsscsr4",
	0x08e8: "trcrsctlr13",
	0x08e9: "trcrsctlr29",
	0x08ea: "trcsscsr5",
	0x08f0: "trcrsctlr14",
	0x08f1: "trcrsctlr30",
	0x08f2: "trcsscsr6",
	0x08f8: "trcrsctlr15",
	0x08f9: "trcrsctlr31",
	0x08fa: "trcsscsr7",
	0x0900: "trcacvr0",
	0x0901: "trcacvr8",
	0x0902: "trcacatr0",
	0x0903: "trcacatr8",
	0x0904: "trcdvcvr0",
	0x0905: "trcdvcvr4",
	0x0906: "trcdvcmr0",
	0x0907: "trcdvcmr4",
	0x0910: "trcacvr1",
	0x0911: "trcacvr9",
	0x0912: "trcacatr1",
	0x0913: "trcacatr9",
	0x0920: "trcacvr2",
	0x0921: "trcacvr10",
	0x0922: "trcacatr2",
	0x0923: "trcacatr10",
	0x0924: "trcdvcvr1",
	0x0925: "trcdvcvr5",
	0x0926: "trcdvcmr1",
	0x0927: "trcdvcmr5",
	0x0930: "trcacvr3",
	0x0931: "trcacvr11",
	0x0932: "trcacatr3",
	0x0933: "trcacatr11",
	0x0940: "trcacvr4",
	0x0941: "trcacvr12",
	0x0942: "trcacatr4",
	0x0943: "trcacatr12",
	0x0944: "trcdvcvr2",
	0x0945: "trcdvcvr6",
	0x0946: "trcdvcmr2",
	0x0947: "trcdvcmr6",
	0x0950: "trcacvr5",
	0x0951: "trcacvr13",
	0x0952: "trcacatr5",
	0x0953: "trcacatr13",
	0x0960: "trcacvr6",
	0x0961: "trcacvr14",
	0x0962: "trcacatr6",
	0x0963: "trcacatr14",
	0x0964: "trcdvcvr3",
	0x0965: "trcdvcvr7",
	0x0966: "trcdvcmr3",
	0x0967: "trcdvcmr7",
	0x0970: "trcacvr7",
	0x0971: "trcacvr15",
	0x0972: "trcacatr7",
	0x0973: "trcacatr15",
	0x0980: "trccidcvr0",
	0x0981: "trcvmidcvr0",
	0x0982: "trccidcctlr0",
	0x098a: "trccidcctlr1",
	0x0990: "trccidcvr1",
	0x0991: "trcvmidcvr1",
	0x0992: "trcvmidcctlr0",
	0x099a: "trcvmidcctlr1",
	0x09a0: "trccidcvr2",
	0x09a1: "trcvmidcvr2",
	0x09b0: "trccidcvr3",
	0x09b1: "trcvmidcvr3",
	0x09c0: "trccidcvr4",
	0x09c1: "trcvmidcvr4",
	0x09d0: "trccidcvr5",
	0x09d1: "trcvmidcvr5",
	0x09e0: "trccidcvr6",
	0x09e1: "trcvmidcvr6",
	0x09f0: "trccidcvr7",
	0x09f1: "trcvmidcvr7",
	0x0b84: "trcitctrl",
	0x0bc6: "trcclaimset",
	0x0bce: "trcclaimclr",
	0x1000: "teecr32_el1",
	0x1080: "teehbr32_el1",
	0x1820: "dbgdtr_el0",
	0x2038: "dbgvcr32_el2",
	0x4080: "sctlr_el1",
	0x4081: "actlr_el1",
	0x4082: "cpacr_el1",
	0x4100: "ttbr0_el1",
	0x4101: "ttbr1_el1",
	0x4102: "tcr_el1",
	0x4200: "spsr_el1",
	0x4201: "elr_el1",
	0x4208: "sp_el0",
	0x4210: "spsel",
	0x4212: "currentel",
	0x4213: "pan",
	0x4214: "uao",
	0x4230: "icc_pmr_el1",
	0x4288: "afsr0_el1",
	0x4289: "afsr1_el1",
	0x4290: "esr_el1",
	0x4300: "far_el1",
	0x43a0: "par_el1",
	0x44c8: "pmscr_el1",
	0x44ca: "pmsicr_el1",
	0x44cb: "pmsirr_el1",
	0x44cc: "pmsfcr_el1",
	0x44cd: "pmsevfr_el1",
	0x44ce: "pmslatfr_el1",
	0x44d0: "pmblimitr_el1",
	0x44d1: "pmbptr_el1",
	0x44d3: "pmbsr_el1",
	0x44f1: "pmintenset_el1",
	0x44f2: "pmintenclr_el1",
	0x4510: "mair_el1",
	0x4518: "amair_el1",
	0x4520: "lorsa_el1",
	0x4521: "lorea_el1",
	0x4522: "lorn_el1",
	0x4523: "lorc_el1",
	0x4600: "vbar_el1",
	0x4602: "rmr_el1",
	0x4643: "icc_bpr0_el1",
	0x4644: "icc_ap0r0_el1",
	0x4645: "icc_ap0r1_el1",
	0x4646: "icc_ap0r2_el1",
	0x4647: "icc_ap0r3_el1",
	0x4648: "icc_ap1r0_el1",
	0x4649: "icc_ap1r1_el1",
	0x464a: "icc_ap1r2_el1",
	0x464b: "icc_ap1r3_el1",
	0x4663: "icc_bpr1_el1",
	0x4664: "icc_ctlr_el1",
	0x4665: "icc_sre_el1",
	0x4666: "icc_igrpen0_el1",
	0x4667: "icc_igrpen1_el1",
	0x4668: "icc_seien_el1",
	0x4681: "contextidr_el1",
	0x4684: "tpidr_el1",
	0x4708: "cntkctl_el1",
	0x5000: "csselr_el1",
	0x5a10: "nzcv",
	0x5a11: "daif",
	0x5a20: "fpcr",
	0x5a21: "fpsr",
	0x5a28: "dspsr_el0",
	0x5a29: "dlr_el0",
	0x5ce0: "pmcr_el0",
	0x5ce1: "pmcntenset_el0",
	0x5ce2: "pmcntenclr_el0",
	0x5ce3: "pmovsclr_el0",
	0x5ce5: "pmselr_el0",
	0x5ce8: "pmccntr_el0",
	0x5ce9: "pmxevtyper_el0",
	0x5cea: "pmxevcntr_el0",
	0x5cf0: "pmuserenr_el0",
	0x5cf3: "pmovsset_el0",
	0x5e82: "tpidr_el0",
	0x5e83: "tpidrro_el0",
	0x5f00: "cntfrq_el0",
	0x5f10: "cntp_tval_el0",
	0x5f11: "cntp_ctl_el0",
	0x5f12: "cntp_cval_el0",
	0x5f18: "cntv_tval_el0",
	0x5f19: "cntv_ctl_el0",
	0x5f1a: "cntv_cval_el0",
	0x5f40: "pmevcntr0_el0",
	0x5f41: "pmevcntr1_el0",
	0x5f42: "pmevcntr2_el0",
	0x5f43: "pmevcntr3_el0",
	0x5f44: "pmevcntr4_el0",
	0x5f45: "pmevcntr5_el0",
	0x5f46: "pmevcntr6_el0",
	0x5f47: "pmevcntr7_el0",
	0x5f48: "pmevcntr8_el0",
	0x5f49: "pmevcntr9_el0",
	0x5f4a: "pmevcntr10_el0",
	0x5f4b: "pmevcntr11_el0",
	0x5f4c: "pmevcntr12_el0",
	0x5f4d: "pmevcntr13_el0",
	0x5f4e: "pmevcntr14_el0",
	0x5f4f: "pmevcntr15_el0",
	0x5f50: "pmevcntr16_el0",
	0x5f51: "pmevcntr17_el0",
	0x5f52: "pmevcntr18_el0",
	0x5f53: "pmevcntr19_el0",
	0x5f54: "pmevcntr20_el0",
	0x5f55: "pmevcntr21_el0",
	0x5f56: "pmevcntr22_el0",
	0x5f57: "pmevcntr23_el0",
	0x5f58: "pmevcntr24_el0",
	0x5f59: "pmevcntr25_el0",
	0x5f5a: "pmevcntr26_el0",
	0x5f5b: "pmevcntr27_el0",
	0x5f5c: "pmevcntr28_el0",
	0x5f5d: "pmevcntr29_el0",
	0x5f5e: "pmevcntr30_el0",
	0x5f60: "pmevtyper0_el0",
	0x5f61: "pmevtyper1_el0",
	0x5f62: "pmevtyper2_el0",
	0x5f63: "pmevtyper3_el0",
	0x5f64: "pmevtyper4_el0",
	0x5f65: "pmevtyper5_el0",
	0x5f66: "pmevtyper6_el0",
	0x5f67: "pmevtyper7_el0",
	0x5f68: "pmevtyper8_el0",
	0x5f69: "pmevtyper9_el0",
	0x5f6a: "pmevtyper10_el0",
	0x5f6b: "pmevtyper11_el0",
	0x5f6c: "pmevtyper12_el0",
	0x5f6d: "pmevtyper13_el0",
	0x5f6e: "pmevtyper14_el0",
	0x5f6f: "pmevtyper15_el0",
	0x5f70: "pmevtyper16_el0",
	0x5f71: "pmevtyper17_el0",
	0x5f72: "pmevtyper18_el0",
	0x5f73: "pmevtyper19_el0",
	0x5f74: "pmevtyper20_el0",
	0x5f75: "pmevtyper21_el0",
	0x5f76: "pmevtyper22_el0",
	0x5f77: "pmevtyper23_el0",
	0x5f78: "pmevtyper24_el0",
	0x5f79: "pmevtyper25_el0",
	0x5f7a: "pmevtyper26_el0",
	0x5f7b: "pmevtyper27_el0",
	0x5f7c: "pmevtyper28_el0",
	0x5f7d: "pmevtyper29_el0",
	0x5f7e: "pmevtyper30_el0",
	0x5f7f: "pmccfiltr_el0",
	0x6000: "vpidr_el2",
	0x6005: "vmpidr_el2",
	0x6080: "sctlr_el2",
	0x6081: "actlr_el2",
	0x6088: "hcr_el2",
	0x6089: "mdcr_el2",
	0x608a: "cptr_el2",
	0x608b: "hstr_el2",
	0x608f: "hacr_el2",
	0x6100: "ttbr0_el2",
	0x6101: "ttbr1_el2",
	0x6102: "tcr_el2",
	0x6108: "vttbr_el2",
	0x610a: "vtcr_el2",
	0x6180: "dacr32_el2",
	0x6200: "spsr_el2",
	0x6201: "elr_el2",
	0x6208: "sp_el1",
	0x6218: "spsr_irq",
	0x6219: "spsr_abt",
	0x621a: "spsr_und",
	0x621b: "spsr_fiq",
	0x6281: "ifsr32_el2",
	0x6288: "afsr0_el2",
	0x6289: "afsr1_el2",
	0x6290: "esr_el2",
	0x6298: "fpexc32_el2",
	0x6300: "far_el2",
	0x6304: "hpfar_el2",
	0x64c8: "pmscr_el2",
	0x6510: "mair_el2",
	0x6518: "amair_el2",
	0x6600: "vbar_el2",
	0x6602: "rmr_el2",
	0x6640: "ich_ap0r0_el2",
	0x6641: "ich_ap0r1_el2",
	0x6642: "ich_ap0r2_el2",
	0x6643: "ich_ap0r3_el2",
	0x6648: "ich_ap1r0_el2",
	0x6649: "ich_ap1r1_el2",
	0x664a: "ich_ap1r2_el2",
	0x664b: "ich_ap1r3_el2",
	0x664c: "ich_vseir_el2",
	0x664d: "icc_sre_el2",
	0x6658: "ich_hcr_el2",
	0x665a: "ich_misr_el2",
	0x665f: "ich_vmcr_el2",
	0x6660: "ich_lr0_el2",
	0x6661: "ich_lr1_el2",
	0x6662: "ich_lr2_el2",
	0x6663: "ich_lr3_el2",
	0x6664: "ich_lr4_el2",
	0x6665: "ich_lr5_el2",
	0x6666: "ich_lr6_el2",
	0x6667: "ich_lr7_el2",
	0x6668: "ich_lr8_el2",
	0x6669: "ich_lr9_el2",
	0x666a: "ich_lr10_el2",
	0x666b: "ich_lr11_el2",
	0x666c: "ich_lr12_el2",
	0x666d: "ich_lr13_el2",
	0x666e: "ich_lr14_el2",
	0x666f: "ich_lr15_el2",
	0x6681: "contextidr_el2",
	0x6682: "tpidr_el2",
	0x6703: "cntvoff_el2",
	0x6708: "cnthctl_el2",
	0x6710: "cnthp_tval_el2",
	0x6711: "cnthp_ctl_el2",
	0x6712: "cnthp_cval_el2",
	0x6718: "cnthv_tval_el2",
	0x6719: "cnthv_ctl_el2",
	0x671a: "cnthv_cval_el2",
	0x6880: "sctlr_el12",
	0x6882: "cpacr_el12",
	0x6900: "ttbr0_el12",
	0x6901: "ttbr1_el12",
	0x6902: "tcr_el12",
	0x6a00: "spsr_el12",
	0x6a01: "elr_el12",
	0x6a88: "afsr0_el12",
	0x6a89: "afsr1_el12",
	0x6a90: "esr_el12",
	0x6b00: "far_el12",
	0x6cc8: "pmscr_el12",
	0x6d10: "mair_el12",
	0x6d18: "amair_el12",
	0x6e00: "vbar_el12",
	0x6e81: "contextidr_el12",
	0x6f08: "cntkctl_el12",
	0x6f10: "cntp_tval_el02",
	0x6f12: "cntp_cval_el02",
	0x6f18: "cntv_tval_el02",
	0x6f19: "cntv_ctl_el02",
	0x6f1a: "cntv_cval_el02",
	0x7080: "sctlr_el3",
	0x7081: "actlr_el3",
	0x7088: "scr_el3",
	0x7089: "sder32_el3",
	0x708a: "cptr_el3",
	0x7099: "mdcr_el3",
	0x7100: "ttbr0_el3",
	0x7102: "tcr_el3",
	0x7200: "spsr_el3",
	0x7201: "elr_el3",
	0x7208: "sp_el2",
	0x7288: "afsr0_el3",
	0x7289: "afsr1_el3",
	0x7290: "esr_el3",
	0x7300: "far_el3",
	0x7510: "mair_el3",
	0x7518: "amair_el3",
	0x7600: "vbar_el3",
	0x7602: "rmr_el3",
	0x7664: "icc_ctlr_el3",
	0x7665: "icc_sre_el3",
	0x7667: "icc_igrpen1_el3",
	0x7682: "tpidr_el3",
	0x7f10: "cntps_tval_el1",
	0x7f11: "cntps_ctl_el1",
	0x7f12: "cntps_cval_el1",
	0x7f90: "cpm_ioacc_ctl_el3",
}

// arm64ReadOnlyRegisters are named on a read alone; written to, they keep their
// number.
var arm64ReadOnlyRegisters = map[uint32]string{
	0x0080: "mdrar_el1",
	0x008c: "oslsr_el1",
	0x03f6: "dbgauthstatus_el1",
	0x0806: "trcidr8",
	0x080e: "trcidr9",
	0x0816: "trcidr10",
	0x0818: "trcstatr",
	0x081e: "trcidr11",
	0x0826: "trcidr12",
	0x082e: "trcidr13",
	0x0847: "trcidr0",
	0x084f: "trcidr1",
	0x0857: "trcidr2",
	0x085f: "trcidr3",
	0x0867: "trcidr4",
	0x086f: "trcidr5",
	0x0877: "trcidr6",
	0x087f: "trcidr7",
	0x088c: "trcoslsr",
	0x08ac: "trcpdsr",
	0x0b97: "trcdevid",
	0x0b9f: "trcdevtype",
	0x0ba7: "trcpidr4",
	0x0baf: "trcpidr5",
	0x0bb7: "trcpidr6",
	0x0bbf: "trcpidr7",
	0x0bc7: "trcpidr0",
	0x0bcf: "trcpidr1",
	0x0bd6: "trcdevaff0",
	0x0bd7: "trcpidr2",
	0x0bde: "trcdevaff1",
	0x0bdf: "trcpidr3",
	0x0be7: "trccidr0",
	0x0bee: "trclsr",
	0x0bef: "trccidr1",
	0x0bf6: "trcauthstatus",
	0x0bf7: "trccidr2",
	0x0bfe: "trcdevarch",
	0x0bff: "trccidr3",
	0x1808: "mdccsr_el0",
	0x1828: "dbgdtrrx_el0",
	0x4000: "midr_el1",
	0x4005: "mpidr_el1",
	0x4006: "revidr_el1",
	0x4008: "id_pfr0_el1",
	0x4009: "id_pfr1_el1",
	0x400a: "id_dfr0_el1",
	0x400b: "id_afr0_el1",
	0x400c: "id_mmfr0_el1",
	0x400d: "id_mmfr1_el1",
	0x400e: "id_mmfr2_el1",
	0x400f: "id_mmfr3_el1",
	0x4010: "id_isar0_el1",
	0x4011: "id_isar1_el1",
	0x4012: "id_isar2_el1",
	0x4013: "id_isar3_el1",
	0x4014: "id_isar4_el1",
	0x4015: "id_isar5_el1",
	0x4016: "id_mmfr4_el1",
	0x4018: "mvfr0_el1",
	0x4019: "mvfr1_el1",
	0x401a: "mvfr2_el1",
	0x4020: "id_aa64pfr0_el1",
	0x4021: "id_aa64pfr1_el1",
	0x4028: "id_aa64dfr0_el1",
	0x4029: "id_aa64dfr1_el1",
	0x402c: "id_aa64afr0_el1",
	0x402d: "id_aa64afr1_el1",
	0x4030: "id_aa64isar0_el1",
	0x4031: "id_aa64isar1_el1",
	0x4038: "id_aa64mmfr0_el1",
	0x4039: "id_aa64mmfr1_el1",
	0x403a: "id_aa64mmfr2_el1",
	0x44cf: "pmsidr_el1",
	0x44d7: "pmbidr_el1",
	0x4527: "lorid_el1",
	0x4601: "rvbar_el1",
	0x4608: "isr_el1",
	0x4640: "icc_iar0_el1",
	0x4642: "icc_hppir0_el1",
	0x465b: "icc_rpr_el1",
	0x4660: "icc_iar1_el1",
	0x4662: "icc_hppir1_el1",
	0x4800: "ccsidr_el1",
	0x4801: "clidr_el1",
	0x4807: "aidr_el1",
	0x5801: "ctr_el0",
	0x5807: "dczid_el0",
	0x5ce6: "pmceid0_el0",
	0x5ce7: "pmceid1_el0",
	0x5f01: "cntpct_el0",
	0x5f02: "cntvct_el0",
	0x6601: "rvbar_el2",
	0x6659: "ich_vtr_el2",
	0x665b: "ich_eisr_el2",
	0x665d: "ich_elsr_el2",
	0x7601: "rvbar_el3",
}

// arm64WriteOnlyRegisters are named on a write alone.
var arm64WriteOnlyRegisters = map[uint32]string{
	0x0084: "oslar_el1",
	0x0884: "trcoslar",
	0x0be6: "trclar",
	0x1828: "dbgdtrtx_el0",
	0x4641: "icc_eoir0_el1",
	0x4659: "icc_dir_el1",
	0x465d: "icc_sgi1r_el1",
	0x465e: "icc_asgi1r_el1",
	0x465f: "icc_sgi0r_el1",
	0x4661: "icc_eoir1_el1",
	0x5ce4: "pmswinc_el0",
}

// arm64SystemAliases name the system operations Capstone spells out, keyed by
// the four selector fields packed together. The flag says whether the
// operation names a register to work on.
var arm64SystemAliases = map[uint32]arm64SystemAlias{
	0x0388: {"ic", "ialluis", false},
	0x03a8: {"ic", "iallu", false},
	0x03b1: {"dc", "ivac", true},
	0x03b2: {"dc", "isw", true},
	0x03c0: {"at", "s1e1r", true},
	0x03c1: {"at", "s1e1w", true},
	0x03c2: {"at", "s1e0r", true},
	0x03c3: {"at", "s1e0w", true},
	0x03d2: {"dc", "csw", true},
	0x03f2: {"dc", "cisw", true},
	0x0418: {"tlbi", "vmalle1is", false},
	0x0419: {"tlbi", "vae1is", true},
	0x041a: {"tlbi", "aside1is", true},
	0x041b: {"tlbi", "vaae1is", true},
	0x041d: {"tlbi", "vale1is", true},
	0x041f: {"tlbi", "vaale1is", true},
	0x0438: {"tlbi", "vmalle1", false},
	0x0439: {"tlbi", "vae1", true},
	0x043a: {"tlbi", "aside1", true},
	0x043b: {"tlbi", "vaae1", true},
	0x043d: {"tlbi", "vale1", true},
	0x043f: {"tlbi", "vaale1", true},
	0x1ba1: {"dc", "zva", true},
	0x1ba9: {"ic", "ivau", true},
	0x1bd1: {"dc", "cvac", true},
	0x1bd9: {"dc", "cvau", true},
	0x1bf1: {"dc", "civac", true},
	0x23c0: {"at", "s1e2r", true},
	0x23c1: {"at", "s1e2w", true},
	0x23c4: {"at", "s12e1r", true},
	0x23c5: {"at", "s12e1w", true},
	0x23c6: {"at", "s12e0r", true},
	0x23c7: {"at", "s12e0w", true},
	0x2401: {"tlbi", "ipas2e1is", true},
	0x2405: {"tlbi", "ipas2le1is", true},
	0x2418: {"tlbi", "alle2is", false},
	0x2419: {"tlbi", "vae2is", true},
	0x241c: {"tlbi", "alle1is", false},
	0x241d: {"tlbi", "vale2is", true},
	0x241e: {"tlbi", "vmalls12e1is", false},
	0x2421: {"tlbi", "ipas2e1", true},
	0x2425: {"tlbi", "ipas2le1", true},
	0x2438: {"tlbi", "alle2", false},
	0x2439: {"tlbi", "vae2", true},
	0x243c: {"tlbi", "alle1", false},
	0x243d: {"tlbi", "vale2", true},
	0x243e: {"tlbi", "vmalls12e1", false},
	0x33c0: {"at", "s1e3r", true},
	0x33c1: {"at", "s1e3w", true},
	0x3418: {"tlbi", "alle3is", false},
	0x3419: {"tlbi", "vae3is", true},
	0x341d: {"tlbi", "vale3is", true},
	0x3438: {"tlbi", "alle3", false},
	0x3439: {"tlbi", "vae3", true},
	0x343d: {"tlbi", "vale3", true},
}

// armSystem64 decodes the generic system register accesses and system
// operations, which x/arch declines.
func armSystem64(raw uint32) (string, bool) {
	if raw&arm64SystemMask != arm64SystemOp {
		return "", false
	}
	read := raw&(1<<21) != 0
	op1, crn, crm, op2 := (raw>>16)&7, (raw>>12)&0xF, (raw>>8)&0xF, (raw>>5)&7
	register := armReg64(raw&0x1F, true)

	if (raw>>19)&3 != arm64SystemOperation {
		name := armSystemRegisterName64((raw>>19)&1<<14|op1<<11|crn<<7|crm<<3|op2, read)
		if name == "" {
			name = fmt.Sprintf("s%d_%d_c%d_c%d_%d", arm64SystemTopField, op1, crn, crm, op2)
		}
		if read {
			return "mrs " + register + ", " + name, true
		}
		return "msr " + name + ", " + register, true
	}

	if read {
		return fmt.Sprintf("sysl %s, #%d, c%d, c%d, #%d", register, op1, crn, crm, op2), true
	}
	if alias, ok := arm64SystemAliases[op1<<11|crn<<7|crm<<3|op2]; ok {
		if alias.register {
			return alias.name + " " + alias.operand + ", " + register, true
		}
		return alias.name + " " + alias.operand, true
	}
	out := fmt.Sprintf("sys #%d, c%d, c%d, #%d", op1, crn, crm, op2)
	if raw&0x1F != arm64StackPointer {
		out += ", " + register
	}
	return out, true
}

// The encoding that copies one element of a vector into another, which x/arch
// names as a move and Capstone as an insert.
const (
	arm64InsertElementMask = 0xFFE08400
	arm64InsertElementOp   = 0x6E000400
)

// The conditional branch, whose last condition x/arch spells as the one that
// always holds rather than the one that never does.
const (
	arm64BranchMask     = 0xFF000010
	arm64BranchOp       = 0x54000000
	arm64NeverCondition = 0xF
)

// armMnemonic64 gives an AArch64 mnemonic the spelling Capstone uses.
func armMnemonic64(mnemonic string, raw uint32) string {
	if raw&arm64InsertElementMask == arm64InsertElementOp {
		return "ins"
	}
	base, condition, found := strings.Cut(mnemonic, ".")
	if !found {
		return mnemonic
	}
	if raw&arm64BranchMask == arm64BranchOp && raw&0xF == arm64NeverCondition {
		return base + ".nv"
	}
	if mapped, ok := armConditions[condition]; ok {
		condition = mapped
	}
	return base + "." + condition
}

// armLanePattern matches the lane a vector operand names.
var armLanePattern = regexp.MustCompile(`\[(\d+)\]`)

// armLaneIndexes64 renders a lane number the way Capstone does, as an ordinary
// immediate rather than always in decimal.
func armLaneIndexes64(operands string) string {
	return armLanePattern.ReplaceAllStringFunc(operands, func(match string) string {
		index, _ := strconv.Atoi(match[1 : len(match)-1])
		return "[" + strings.TrimPrefix(armImmediate32(int64(index)), "#") + "]"
	})
}

// armBitfieldInsert64 rewrites the bitfield move Capstone names as an insert.
// x/arch leaves it as the raw operation whenever the two field bounds cross,
// which is exactly the case the insert covers: the position counts backwards
// from the width of the register.
func armBitfieldInsert64(mnemonic string, raw uint32) (string, bool) {
	if mnemonic != "bfm" {
		return "", false
	}
	width := uint32(32)
	if raw&(1<<31) != 0 {
		width = 64
	}
	immr, imms := (raw>>16)&0x3F, (raw>>10)&0x3F
	if imms >= immr {
		return "", false
	}
	return fmt.Sprintf("bfi %s, %s, %s, %s",
		armReg64(raw&0x1F, width == 64), armReg64((raw>>5)&0x1F, width == 64),
		armImmediate32(int64(width-immr)), armImmediate32(int64(imms+1))), true
}

// armPairTransfers64 are the AArch64 instructions that move a register pair,
// which are the only ones to keep the sign on a post-indexed displacement.
var armPairTransfers64 = map[string]bool{
	"ldp": true, "stp": true, "ldpsw": true, "ldnp": true, "stnp": true,
}

// armVectorRange matches the shorthand x/arch uses for a run of consecutive
// vector registers, which Capstone spells out one by one.
var armVectorRange = regexp.MustCompile(`\{v(\d+)\.(\w+)-v(\d+)\.\w+\}`)

// armExpandVectorList writes out a run of vector registers in full.
func armExpandVectorList(operands string) string {
	return armVectorRange.ReplaceAllStringFunc(operands, func(match string) string {
		parts := armVectorRange.FindStringSubmatch(match)
		first, _ := strconv.Atoi(parts[1])
		last, _ := strconv.Atoi(parts[3])
		names := make([]string, 0, last-first+1)
		for n := first; n <= last; n++ {
			names = append(names, fmt.Sprintf("v%d.%s", n, parts[2]))
		}
		return "{" + strings.Join(names, ", ") + "}"
	})
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
			// A linking branch leaves the low bits of the encoding clear.
			if op&7 != 0 {
				return "", false
			}
			return "blx " + rm, true
		}
		return "bx " + rm, true
	}

	mnemonic := [3]string{"add", "cmp", "mov"}[(op>>8)&3]
	rd := armReg((op & 7) | (op>>7)&1<<3)
	if mnemonic == "add" && (op>>3)&0xF == armVFPStackPointerRn {
		// Adding the stack pointer accumulates into the register it names, so
		// Capstone prints that register on both sides.
		return fmt.Sprintf("add %s, %s, %s", rd, rm, rd), true
	}
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
	if op&0xFF == 0 {
		return "", false // there is nothing to transfer
	}
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
	"hi", "ls", "ge", "lt", "gt", "le", "al", "",
}

// armThumbConditionalBranch covers conditional branches and the supervisor
// call, which share the encoding.
func armThumbConditionalBranch(op uint32, address uint64) (string, bool) {
	cond := (op >> 8) & 0xF
	switch cond {
	case 0xF:
		return "svc " + armImmediate32(int64(op&0xFF)), true
	case 0xE:
		if op&0xFF == armThumbTrapValue {
			return "trap", true // the value Capstone gives a name of its own
		}
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

	case op>>8 == 0b10110110:
		return armThumbProcessorState(op)

	case op>>8 == 0b10111110:
		return "bkpt " + armImmediate32(int64(op&0xFF)), true

	case op>>8 == 0b10111111:
		return armThumbHint(op)

	case (op>>9)&0b011 == 0b010: // push and pop
		return armThumbPushPop(op)
	}
	return "", false
}

// The 16-bit processor-state group: setting the endianness, and the two
// changes of the interrupt masks.
const (
	armThumbSetEndian  = 0b0101
	armThumbChangeMask = 0b011
	armThumbTrapValue  = 0xFE
)

// armThumbProcessorState renders SETEND and the narrow spelling of CPS.
func armThumbProcessorState(op uint32) (string, bool) {
	switch {
	case (op>>4)&0xF == armThumbSetEndian && op&7 == 0:
		if op&(1<<3) != 0 {
			return "setend be", true
		}
		return "setend le", true
	case (op>>5)&7 == armThumbChangeMask && op&(1<<3) == 0:
		name := "cpsie"
		if op&(1<<4) != 0 {
			name = "cpsid"
		}
		return name + " " + armChangeStateFlags(op<<6), true
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
	// The reserved condition is treated as always, which also decides whether
	// each following instruction runs on the condition or its inverse.
	if firstCond == 0b1111 {
		firstCond = 0b1110
	}
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
	if list == 0 {
		return "", false // there is nothing to transfer
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
	if text, ok := armThumb32PrivilegedCall(first, second); ok {
		return text, true
	}
	if text, ok := armThumb32HintOrBarrier(first, second); ok {
		return text, true
	}
	return armThumbLongBranch(first, second, address)
}

// The wide Thumb hints and barriers, which sit in the same group as the
// branches and are told from them by their first halfword.
const (
	armThumbHintOp       = 0xF3AF
	armThumbBarrierOp    = 0xF3BF
	armThumbHintMask     = 0xFF00
	armThumbHintMarker   = 0x8000
	armThumbBarrierTop   = 0x8F00
	armThumbFirstBarrier = 4
	armThumbLastBarrier  = 6
)

// The wide change of processor state, which shares the hints' encoding: the two
// bits above the mode flag say what it changes, and the flag says whether a
// processor mode is named as well.
const (
	armThumbStateMask = 0xF800
	armThumbStateTop  = 0x8000
	armThumbModeFlag  = 1 << 8
)

// armThumb32ChangeState renders it. Changing the masks alone carries the wide
// marker, since the narrow encoding says the same thing; naming a mode does
// not, since no narrow encoding can.
func armThumb32ChangeState(second uint32) (string, bool) {
	if second&armThumbStateMask != armThumbStateTop {
		return "", false
	}
	change := (second >> 9) & 3
	mode := second & armChangeModeMask
	flags := armChangeStateFlags(second << 1) // the masks sit one bit lower here

	if second&armThumbModeFlag == 0 {
		name := armChangeStateNames[change]
		if name == "" || mode != 0 {
			return "", false
		}
		return name + ".w " + flags, true
	}
	if change == armChangeNone {
		return "cps " + armImmediate32(int64(mode)), true
	}
	name := armChangeStateNames[change]
	if name == "" {
		return "", false
	}
	return name + " " + flags + ", " + armImmediate32(int64(mode)), true
}

// armThumb32HintOrBarrier renders them. A hint carries the wide marker, since
// it has a narrow encoding to be told apart from; a barrier does not.
func armThumb32HintOrBarrier(first, second uint32) (string, bool) {
	switch first {
	case armThumbHintOp:
		if second&armThumbHintMask != armThumbHintMarker {
			return armThumb32ChangeState(second)
		}
		return armHint(second&0xFF, "", ".w"), true
	case armThumbBarrierOp:
		kind := (second >> 4) & 0xF
		if second&0xFF00 != armThumbBarrierTop ||
			kind < armThumbFirstBarrier || kind > armThumbLastBarrier {
			return "", false
		}
		return armBarrier(kind-armThumbFirstBarrier, second&0xF), true
	}
	return "", false
}

// The two Thumb calls into a higher privilege level, and the marker that tells
// them from the branches they share an encoding with.
const (
	armThumbHypervisorCall = 0xF7E0
	armThumbSecureCall     = 0xF7F0
	armThumbCallMask       = 0xFFF0
	armThumbCallMarkerMask = 0xF000
	armThumbCallMarker     = 0x8000
)

// armThumb32PrivilegedCall renders the hypervisor and secure monitor calls. The
// hypervisor call joins its two immediate fields and carries a wide marker; the
// secure monitor call takes only the field in its first halfword and reads past
// the second.
func armThumb32PrivilegedCall(first, second uint32) (string, bool) {
	if second&armThumbCallMarkerMask != armThumbCallMarker {
		return "", false
	}
	switch first & armThumbCallMask {
	case armThumbHypervisorCall:
		return "hvc.w " + armImmediate32(int64((first&0xF)<<12|second&0xFFF)), true
	case armThumbSecureCall:
		return "smc " + armImmediate32(int64(first&0xF)), true
	}
	return "", false
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

// The opcode that packs two halfwords, and the two shifts it accepts: shifting
// left takes the bottom half of the second source, shifting right the top.
const (
	armThumb32PackOpcode = 0b0110
	armThumb32PackLeft   = 0b00
	armThumb32PackRight  = 0b10
)

// armThumb32Pack renders PKHBT and PKHTB. Unlike every other shifted operand
// theirs is printed as an ordinary immediate, so it turns to hexadecimal above
// nine rather than staying in decimal.
func armThumb32Pack(first, second uint32) (string, bool) {
	if first&(1<<4) != 0 {
		return "", false // packing sets no flags
	}
	amount := (second>>12)&7<<2 | (second>>6)&3
	rn, rd, rm := first&0xF, (second>>8)&0xF, second&0xF

	var name, shift string
	switch (second >> 4) & 3 {
	case armThumb32PackLeft:
		name = "pkhbt"
		if amount != 0 {
			shift = ", lsl " + armImmediate32(int64(amount))
		}
	case armThumb32PackRight:
		name = "pkhtb"
		if amount == 0 {
			amount = 32 // an arithmetic shift encodes its widest as zero
		}
		shift = ", asr " + armImmediate32(int64(amount))
	default:
		return "", false
	}
	return fmt.Sprintf("%s %s, %s, %s%s", name, armReg(rd), armReg(rn), armReg(rm), shift), true
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
	signed := armImmediate32(int64(int32(value))) // #nosec G115 -- the expanded value is a 32-bit constant
	unsigned := armImmediate32(int64(value))

	// With no destination the operation is a comparison instead.
	if rd == 15 && setsFlags {
		if compare, ok := armThumb32CompareOps[op]; ok {
			return fmt.Sprintf("%s.w %s, %s", compare, armReg(rn), signed), true
		}
	}

	// ORR and ORN against the unused register are the move aliases.
	if rn == 15 {
		switch base {
		case "orr":
			return fmt.Sprintf("%s %s, %s", armThumb32Mnemonic("mov", setsFlags, true), armReg(rd), signed), true
		case "orn":
			return fmt.Sprintf("%s %s, %s", armThumb32Mnemonic("mvn", setsFlags, false), armReg(rd), unsigned), true
		}
	}

	imm := signed
	if armThumbUnsignedImmediateOps[base] {
		imm = unsigned
	}
	mnemonic := armThumb32Mnemonic(base, setsFlags, armThumb32WideImmediateOps[base])
	return fmt.Sprintf("%s %s, %s, %s", mnemonic, armReg(rd), armReg(rn), imm), true
}

// armThumbUnsignedImmediateOps are the bitwise operations whose constant
// Capstone prints as it stands. Every other operation on a modified immediate
// prints it signed, including the comparisons these become when they discard
// their result and the move a disjunction becomes.
var armThumbUnsignedImmediateOps = map[string]bool{
	"and": true, "bic": true, "orr": true, "eor": true,
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
		if second&1 != 0 {
			return "", false // an unaligned target is unpredictable
		}
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
	case op2&0b1000000 == 0b1000000:
		return armThumb32Coprocessor(first, second)
	case op2&0b1100000 == 0b0100000:
		return armThumb32DataShiftedRegister(first, second)
	case op2&0b0000100 == 0b0000100:
		return armThumb32DualOrExclusive(first, second)
	}
	// The bits above have accounted for every value of op2, so what is left is
	// the block transfer.
	return armThumb32BlockTransfer(first, second)
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
	if rn == armVFPStackPointerRn && writeback {
		if load && (first>>7)&1 == 1 {
			if second&(1<<armVFPStackPointerRn) != 0 {
				return "", false // popping the stack pointer has no meaning
			}
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
	if op == armThumb32PackOpcode {
		return armThumb32Pack(first, second)
	}
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
	// Standing alone the amount is an ordinary immediate rather than part of a
	// shifted operand, so it turns to hexadecimal above nine.
	mnemonic := armThumb32Mnemonic(armThumb32ShiftNames[kind], setsFlags, true)
	return fmt.Sprintf("%s %s, %s, %s", mnemonic, armReg(rd), armReg(rm), armImmediate32(int64(amount)))
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

	case 0b10000, 0b10010, 0b11000, 0b11010: // signed and unsigned saturate
		return armThumb32Saturate(first, second)
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
	if second&(1<<5) != 0 || first&armThumbPlainImmediateBit != 0 {
		return "", false // two bits the encoding requires to be clear
	}
	return armBitfield((second>>12)&7<<2|(second>>6)&3, second&0x1F, rd, rn, ""), true
}

// armBitfield renders a bitfield insert or clear. Where the top bit lies below
// the position the width would come out at nothing, and Capstone prints the top
// bit in place of the position and a width of one.
func armBitfield(lsb, msb, rd, rn uint32, suffix string) string {
	position, extent := lsb, msb-lsb+1
	if msb < lsb {
		position, extent = msb, 1
	}
	operands := fmt.Sprintf("%s, %s, %s", armReg(rd),
		armImmediate32(int64(position)), armImmediate32(int64(extent)))
	if rn == armProgramCounter {
		return "bfc" + suffix + " " + operands
	}
	return "bfi" + suffix + " " + armReg(rd) + ", " + armReg(rn) +
		operands[len(armReg(rd)):]
}

// The bitfield insert and clear, whose ARM encoding x/arch declines when the
// top bit lies below the position.
const (
	armBitfieldMask = 0x0FE00070
	armBitfieldOp   = 0x07C00010
)

// armBitfield32 renders the ARM spelling of a bitfield insert or clear.
func armBitfield32(raw uint32) (string, bool) {
	cond := raw >> 28
	if raw&armBitfieldMask != armBitfieldOp || cond == 15 {
		return "", false
	}
	return armBitfield((raw>>7)&0x1F, (raw>>16)&0x1F, (raw>>12)&0xF, raw&0xF,
		armCondSuffix(cond)), true
}

// armThumb32LoadStoreSingle covers the third top-level group: single-item
// memory access, data processing on plain registers, the multiplies, and the
// coprocessor space.
func armThumb32LoadStoreSingle(first, second uint32) (string, bool) {
	op2 := (first >> 4) & 0x7F
	switch {
	case first>>8 == armThumbVectorTransfer && first&(1<<4) == 0:
		return armThumb32VectorTransfer(first, second)
	case op2&0b1000000 == 0b1000000:
		return armThumb32Coprocessor(first, second)
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
	if rn == armProgramCounter && first&(1<<4) == 0 {
		return "", false // there is no store relative to the program counter
	}

	// A load whose destination is the unused register is a preload hint, but
	// only in the forms that do not index the base register.
	if rt == armProgramCounter && strings.HasPrefix(mnemonic, "ldr") && armThumb32HasHintShape(first, second) {
		switch hint := armThumb32PreloadHints[(first>>8)&1][(first>>5)&3]; {
		case hint != "":
			return armThumb32Preload(first, second, rn)
		case mnemonic == "ldrsh":
			// The signed halfword has no hint of its own, and is unallocated
			// in exactly the shapes one would have taken. The whole word has
			// none either, but goes on to be an ordinary load.
			return "", false
		}
	}

	switch {
	case rn == 15 && first&(1<<4) != 0: // a literal load names the program counter
		offset := int64(second & 0xFFF)
		if first&(1<<7) == 0 {
			offset = -offset
		}
		return mnemonic + ".w " + armReg(rt) + ", [pc, " + armImmediate32(offset) + "]", true

	case first&(1<<7) != 0: // twelve-bit positive offset
		return mnemonic + ".w " + armReg(rt) + ", " + armThumb32Offset(rn, int64(second&0xFFF)), true

	case (second>>8)&0xF == 0b1110: // unprivileged access
		return mnemonic + "t " + armReg(rt) + ", " + armThumb32Offset(rn, int64(second&0xFF)), true

	case second&(1<<11) != 0: // eight-bit signed offset, optionally written back
		return armThumb32IndexedMemory(mnemonic, rn, rt, second)
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
	operand := "[" + armReg(rn) + ", " + armReg(second&0xF)
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
func armThumb32IndexedMemory(mnemonic string, rn, rt, second uint32) (string, bool) {
	offset := int64(second & 0xFF)
	if second&(1<<9) == 0 {
		offset = -offset
	}
	preIndexed := second&(1<<10) != 0
	writeback := second&(1<<8) != 0
	if !preIndexed && !writeback {
		return "", false // post-indexed with no writeback: the offset is lost
	}

	// A displacement of zero that is subtracted keeps its sign, spelt bare after
	// the brackets and with a hexadecimal prefix inside them.
	subtracted := second&(1<<9) == 0 && second&0xFF == 0
	if !preIndexed {
		displacement := armImmediate32(offset)
		if subtracted {
			displacement = armNegativeZeroBare
		}
		return fmt.Sprintf("%s %s, [%s], %s", mnemonic, armReg(rt), armReg(rn), displacement), true
	}
	address := armThumb32Offset(rn, offset)
	if subtracted {
		address = "[" + armReg(rn) + ", " + armNegativeZeroHex + "]"
	}
	out := mnemonic + " " + armReg(rt) + ", " + address
	if writeback {
		out += "!"
	}
	return out, true
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
		if (second>>12)&0xF != 0xF {
			return "", false
		}
		mnemonic := armThumb32Mnemonic(armThumb32RegisterShifts[(op1>>1)&3], op1&1 != 0, true)
		return fmt.Sprintf("%s %s, %s, %s", mnemonic, armReg(rd), armReg(rn), armReg(rm)), true
	}

	if op2&0b1000 != 0 { // extend, byte reversal or bit counting
		return armThumb32Miscellaneous(first, second)
	}
	if op1&0b1000 != 0 {
		return armThumb32ParallelArithmetic(first, second)
	}
	return "", false
}

// armThumb32ParallelOps name the operation a parallel instruction performs,
// indexed by the low three bits of the first halfword's opcode, and
// armThumb32ParallelGroups the prefix that says how it treats overflow.
var (
	armThumb32ParallelOps    = [8]string{"add8", "add16", "asx", "", "sub8", "sub16", "sax", ""}
	armThumb32ParallelGroups = [8]string{"s", "q", "sh", "", "u", "uq", "uh", ""}
)

// armThumb32ParallelArithmetic renders the operations that work on the halves
// or quarters of a word at once.
func armThumb32ParallelArithmetic(first, second uint32) (string, bool) {
	if (second>>12)&0xF != 0xF || second&(1<<7) != 0 {
		return "", false
	}
	operation := armThumb32ParallelOps[(first>>4)&7]
	group := armThumb32ParallelGroups[(second>>4)&7]
	if operation == "" || group == "" {
		return "", false
	}
	return fmt.Sprintf("%s%s %s, %s, %s", group, operation,
		armReg((second>>8)&0xF), armReg(first&0xF), armReg(second&0xF)), true
}

// armThumb32Extends name the sign- and zero-extending operations, which take a
// second source register in their accumulate form.
var armThumb32Extends = [8]string{"sxth", "uxth", "sxtb16", "uxtb16", "sxtb", "uxtb", "", ""}

// armThumb32MiscOps name the bit-counting and reversal operations, and
// armThumb32SaturatingOps the saturating add and subtract that share the group.
var (
	armThumb32MiscOps       = [4]string{"rev.w", "rev16.w", "rbit", "revsh.w"}
	armThumb32SaturatingOps = [4]string{"qadd", "qdadd", "qsub", "qdsub"}
)

// The four operations the group holds, read from bits 6 to 4 of the first
// halfword.
const (
	armThumb32Saturating = 0b000
	armThumb32Reversal   = 0b001
	armThumb32Select     = 0b010
	armThumb32CountZeros = 0b011
)

// armThumb32SaturateOrReverse covers the saturating arithmetic, the byte
// reversals, the byte selection and the leading-zero count. The reversals and
// the count name one source twice; where the two fields disagree Capstone
// prints the bits of both together, which is how its decoder merges them.
func armThumb32SaturateOrReverse(first, second, rn, rd, rm uint32) (string, bool) {
	if (second>>12)&0xF != 0xF || second&(1<<6) != 0 {
		return "", false
	}
	selector := (second >> 4) & 3

	switch (first >> 4) & 7 {
	case armThumb32Saturating:
		return fmt.Sprintf("%s %s, %s, %s", armThumb32SaturatingOps[selector],
			armReg(rd), armReg(rm), armReg(rn)), true
	case armThumb32Reversal:
		return fmt.Sprintf("%s %s, %s", armThumb32MiscOps[selector], armReg(rd), armReg(rn|rm)), true
	case armThumb32Select:
		if selector != 0 {
			return "", false
		}
		return fmt.Sprintf("sel %s, %s, %s", armReg(rd), armReg(rn), armReg(rm)), true
	case armThumb32CountZeros:
		if selector != 0 {
			return "", false
		}
		return fmt.Sprintf("clz %s, %s", armReg(rd), armReg(rn|rm)), true
	}
	return "", false
}

// armThumb32Miscellaneous covers the extends, byte reversals and CLZ.
func armThumb32Miscellaneous(first, second uint32) (string, bool) {
	op1 := (first >> 4) & 0xF
	rn, rd, rm := first&0xF, (second>>8)&0xF, second&0xF

	if op1&0b1000 != 0 {
		return armThumb32SaturateOrReverse(first, second, rn, rd, rm)
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

// armThumb32MultiplyForms describe the signed multiply group: the mnemonic
// with and without an accumulator, and the suffixes its variant bits select.
var armThumb32MultiplyForms = map[uint32]struct {
	plain, accumulate string
	suffixes          []string
	always            bool // prints the accumulator even when it is unused
}{
	0b001: {"smul", "smla", []string{"bb", "bt", "tb", "tt"}, false},
	0b010: {"smuad", "smlad", []string{"", "x"}, false},
	0b011: {"smulw", "smlaw", []string{"b", "t"}, false},
	0b100: {"smusd", "smlsd", []string{"", "x"}, false},
	0b101: {"smmul", "smmla", []string{"", "r"}, false},
	0b110: {"smmls", "smmls", []string{"", "r"}, true},
	0b111: {"usad8", "usada8", []string{""}, false},
}

// armThumb32Multiply covers the multiply group: the plain multiply and
// multiply-accumulate, and the signed halfword and dual multiplies.
func armThumb32Multiply(first, second uint32) (string, bool) {
	if (second>>6)&3 != 0 {
		return "", false
	}
	rn, ra, rd, rm := first&0xF, (second>>12)&0xF, (second>>8)&0xF, second&0xF
	variant := (second >> 4) & 3

	op1 := (first >> 4) & 7
	if op1 == 0 {
		return armThumb32PlainMultiply(variant, rn, ra, rd, rm)
	}

	form, ok := armThumb32MultiplyForms[op1]
	if !ok || int(variant) >= len(form.suffixes) {
		return "", false
	}

	name := form.plain
	if ra != 15 || form.always {
		name = form.accumulate
	}
	name += form.suffixes[variant]

	if ra == 15 && !form.always {
		return fmt.Sprintf("%s %s, %s, %s", name, armReg(rd), armReg(rn), armReg(rm)), true
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", name, armReg(rd), armReg(rn), armReg(rm), armReg(ra)), true
}

// armThumb32PlainMultiply covers the multiply, multiply-accumulate and
// multiply-subtract that share the first slot of the group.
func armThumb32PlainMultiply(variant, rn, ra, rd, rm uint32) (string, bool) {
	if variant > 1 {
		return "", false
	}
	if ra == 15 {
		if variant != 0 {
			return "", false
		}
		return fmt.Sprintf("mul %s, %s, %s", armReg(rd), armReg(rn), armReg(rm)), true
	}
	mnemonic := "mla"
	if variant == 1 {
		mnemonic = "mls"
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", mnemonic, armReg(rd), armReg(rn), armReg(rm), armReg(ra)), true
}

// armThumb32Halves name which half of each source a halfword multiply reads.
var armThumb32Halves = [4]string{"bb", "bt", "tb", "tt"}

// armThumb32LongPairs name the 64-bit multiplies and the divides by their
// operation and second-operand fields.
var armThumb32LongPairs = map[[2]uint32]string{
	{0b000, 0b0000}: "smull",
	{0b010, 0b0000}: "umull",
	{0b100, 0b0000}: "smlal",
	{0b110, 0b0000}: "umlal",
	{0b001, 0b1111}: "sdiv",
	{0b011, 0b1111}: "udiv",
}

// armThumb32LongMultiply covers the 64-bit multiplies, the divides, and the
// dual multiplies that accumulate into a register pair.
func armThumb32LongMultiply(first, second uint32) (string, bool) {
	op1 := (first >> 4) & 7
	op2 := (second >> 4) & 0xF
	rn, rdLo, rdHi, rm := first&0xF, (second>>12)&0xF, (second>>8)&0xF, second&0xF

	if name, ok := armThumb32LongPairs[[2]uint32{op1, op2}]; ok {
		if strings.HasSuffix(name, "div") { // a divide writes one register
			if rdLo != armProgramCounter {
				return "", false
			}
			return fmt.Sprintf("%s %s, %s, %s", name, armReg(rdHi), armReg(rn), armReg(rm)), true
		}
		return fmt.Sprintf("%s %s, %s, %s, %s", name, armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	}

	// The halfword multiplies pick their two halves out of the opcode.
	if op1 == 0b100 && op2&0b1100 == 0b1000 {
		name := "smlal" + armThumb32Halves[op2&3]
		return fmt.Sprintf("%s %s, %s, %s, %s", name, armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	}
	if op1 == 0b110 && op2 == 0b0110 {
		return fmt.Sprintf("umaal %s, %s, %s, %s", armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm)), true
	}

	if op2&0b1110 == 0b1100 {
		switch op1 {
		case 0b100:
			return armThumb32DualLong("smlald", op2, rdLo, rdHi, rn, rm), true
		case 0b101:
			return armThumb32DualLong("smlsld", op2, rdLo, rdHi, rn, rm), true
		}
	}
	return "", false
}

// armThumb32ListIsValid rejects the register lists the manual leaves
// unpredictable, which Capstone declines to decode: a store naming the program
// counter, a load naming both it and the link register, a list with fewer than
// two registers, and writeback to a register the list already contains.
func armThumb32ListIsValid(list uint32, load bool) bool {
	const stackAndProgramCounter = 1<<15 | 1<<13

	if !load && list&stackAndProgramCounter != 0 {
		// A store may name neither the stack pointer nor the program counter.
		return false
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

	// The exclusive accesses and the table branches only exist where the
	// addressing bits leave no room for a displacement.
	if op1 == 0b00 {
		switch op2 {
		case 0b00:
			return fmt.Sprintf("strex %s, %s, %s", armReg(rt2), armReg(rt),
				armThumb32Offset(rn, int64(second&0xFF)*4)), true
		case 0b01:
			if rt2 != armProgramCounter {
				return "", false
			}
			return fmt.Sprintf("ldrex %s, %s", armReg(rt),
				armThumb32Offset(rn, int64(second&0xFF)*4)), true
		}
	}
	if op1 == 0b01 {
		switch op2 {
		case 0b00:
			return armThumb32ExclusiveNarrow(first, second, rn, rt, rt2)
		case 0b01:
			if selector := (second >> 4) & 0xF; selector > 1 {
				return armThumb32ExclusiveNarrow(first, second, rn, rt, rt2)
			}
			return armThumb32TableBranch(first, second, rt, rt2)
		}
	}

	return armThumb32DualTransfer(first, second, rn, rt, rt2)
}

// armThumb32ExclusiveWidths name the narrower exclusive stores by the field
// that selects them; the doubleword form names a second data register instead
// of leaving the field fixed.
var armThumb32ExclusiveWidths = map[uint32]string{0b0100: "strexb", 0b0101: "strexh"}

// The exclusive store that covers a register pair.
const armThumb32ExclusiveDouble = 0b0111

// armThumb32ExclusiveNarrow renders the exclusive stores that carry no
// displacement.
func armThumb32ExclusiveNarrow(first, second, rn, rt, rt2 uint32) (string, bool) {
	load := first&(1<<4) != 0
	if load && second&0xF != 0xF {
		return "", false // a load has no status register, so the field is fixed
	}
	selector := (second >> 4) & 0xF
	if selector == armThumb32ExclusiveDouble {
		if load {
			return fmt.Sprintf("ldrexd %s, %s, [%s]", armReg(rt), armReg(rt2), armReg(rn)), true
		}
		return fmt.Sprintf("strexd %s, %s, %s, [%s]", armReg(second&0xF),
			armReg(rt), armReg(rt2), armReg(rn)), true
	}
	name, ok := armThumb32ExclusiveWidths[selector]
	if !ok || rt2 != armProgramCounter {
		return "", false
	}
	if load {
		return fmt.Sprintf("ldrex%s %s, [%s]", name[len("strex"):], armReg(rt), armReg(rn)), true
	}
	return fmt.Sprintf("%s %s, %s, [%s]", name, armReg(second&0xF), armReg(rt), armReg(rn)), true
}

// armThumb32TableBranch renders the byte and halfword table branches.
func armThumb32TableBranch(first, second, rt, rt2 uint32) (string, bool) {
	if rt != armProgramCounter || rt2 != 0 || (second>>5)&7 != 0 {
		return "", false
	}
	if second&(1<<4) != 0 {
		return fmt.Sprintf("tbh [%s, %s, lsl #1]", armReg(first&0xF), armReg(second&0xF)), true
	}
	return fmt.Sprintf("tbb [%s, %s]", armReg(first&0xF), armReg(second&0xF)), true
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
	negative := first&(1<<7) == 0
	offset := int64(second&0xFF) * 4
	if negative {
		offset = -offset
	}
	// Capstone keeps the sign even where the displacement is nothing at all.
	displacement := armImmediate32(offset)
	if negative && offset == 0 {
		displacement = "#-0"
	}
	registers := armReg(rt) + ", " + armReg(rt2) + ", "

	if !preIndexed {
		return fmt.Sprintf("%s %s[%s], %s", mnemonic, registers, armReg(rn), displacement), true
	}
	// Where it counts downwards the pre-indexed displacement is printed as an
	// address rather than as a plain immediate, so it stays in hexadecimal.
	if negative {
		displacement = fmt.Sprintf("#-0x%x", -offset)
	}
	out := mnemonic + " " + registers + "[" + armReg(rn) + "]"
	if offset != 0 || negative || writeback {
		out = mnemonic + " " + registers + "[" + armReg(rn) + ", " + displacement + "]"
	}
	if writeback {
		out += "!"
	}
	return out, true
}

// armMoveWide64 renders MOVN, MOVZ and MOVK. Capstone prints the sixteen-bit
// field the encoding carries and the shift that places it, where x/arch prints
// the value they combine into.
func armMoveWide64(raw uint32) (string, bool) {
	if raw&arm64MoveWideMask != arm64MoveWideOp {
		return "", false
	}
	name := arm64MoveWideNames[(raw>>29)&3]
	if name == "" {
		return "", false
	}

	register := armReg64(raw&0x1F, raw&(1<<31) != 0)
	out := fmt.Sprintf("%s %s, %s", name, register, armImmediate64Wide(int64((raw>>5)&0xFFFF)))
	if shift := (raw >> 21) & 3; shift != 0 {
		out += fmt.Sprintf(", lsl #%d", shift*16)
	}
	return out, true
}

// armReg64 names an AArch64 general register, which is spelled x or w by width
// and zr in place of register 31.
func armReg64(n uint32, wide bool) string {
	prefix := "w"
	if wide {
		prefix = "x"
	}
	if n == 31 {
		return prefix + "zr"
	}
	return fmt.Sprintf("%s%d", prefix, n)
}

// armThumb32DualLong renders the dual long multiplies, whose low bit selects
// the exchanged form.
func armThumb32DualLong(name string, op2, rdLo, rdHi, rn, rm uint32) string {
	if op2&1 != 0 {
		name += "x"
	}
	return fmt.Sprintf("%s %s, %s, %s, %s", name, armReg(rdLo), armReg(rdHi), armReg(rn), armReg(rm))
}

// armThumb32HasHintShape reports whether a load into the unused register has
// one of the shapes a memory hint takes: a wide offset either way, a negative
// byte offset, or a shifted register. The indexed forms remain loads.
func armThumb32HasHintShape(first, second uint32) bool {
	switch {
	case first&(1<<7) != 0: // twelve-bit positive offset
		return true
	case first&0xF == armProgramCounter: // a literal, whichever way it counts
		return true
	case (second>>8)&0xF == 0b1100: // eight-bit negative offset
		return true
	case second&(1<<11) != 0: // an indexed form, so not a hint
		return false
	}
	return (second>>6)&0x3F == 0
}

// armThumb32PreloadHints name the hint a load with no destination register
// stands for, by its sign bit and element size. A whole word is never a hint,
// and the signed halfword has no meaning at all.
var armThumb32PreloadHints = [2][4]string{
	0: {"pld", "pldw"},
	1: {"pli"},
}

// armThumb32PreloadLiteral renders a hint about an address relative to the
// program counter. Only the plain hint reaches that form, and the widening one
// reaches it only while the displacement counts downwards.
func armThumb32PreloadLiteral(first, second uint32, hint string) (string, bool) {
	up := first&(1<<7) != 0
	if hint == "pldw" {
		if up {
			return "", false
		}
		hint = "pld"
	}
	offset := int64(second & 0xFFF)
	if !up {
		offset = -offset
	}
	return hint + " [pc, " + armImmediate32(offset) + "]", true
}

// armThumb32Preload renders the preload hints, which reuse the load encodings
// with the destination left unused. A byte access preloads data, a halfword one
// preloads it for writing, and a signed access preloads an instruction.
func armThumb32Preload(first, second, rn uint32) (string, bool) {
	hint := armThumb32PreloadHints[(first>>8)&1][(first>>5)&3]
	if rn == armProgramCounter {
		return armThumb32PreloadLiteral(first, second, hint)
	}

	switch {
	case first&(1<<7) != 0:
		return hint + " " + armThumb32Offset(rn, int64(second&0xFFF)), true
	case (second>>8)&0xF == 0b1100:
		return hint + " " + armThumb32Offset(rn, -int64(second&0xFF)), true
	}

	operand := "[" + armReg(rn) + ", " + armReg(second&0xF)
	if shift := (second >> 4) & 3; shift != 0 {
		operand += fmt.Sprintf(", lsl #%d", shift)
	}
	return hint + " " + operand + "]", true
}

// Coprocessors 10 and 11 are the floating-point and Advanced SIMD units, whose
// instructions Capstone prints in their own syntax rather than as generic
// coprocessor transfers.
const (
	armCoprocVFPLow  = 10
	armCoprocVFPHigh = 11
)

// The two halves of the Thumb coprocessor space, read from bits 11 to 8 of the
// first halfword, and the conditions their ARM counterparts carry.
const (
	armThumbSIMDGroup      = 0b1111
	armThumbVectorTransfer = 0xF9 // the top byte of a vector load or store
	armThumbAlwaysCond     = 14
	armThumbUnusedCond     = 15
	armSIMDDataOperation   = 0xF2000000 // the ARM Advanced SIMD data-processing space
	armSIMDTransferSpace   = 0xF4000000 // and its element and structure transfers
)

// armThumb32Coprocessor covers the coprocessor, floating-point and Advanced
// SIMD instructions. Thumb repeats the ARM encodings word for word below the
// top nibble, so each is decoded by rebuilding the ARM word it mirrors; bit 12
// of the first halfword stands in for the condition, choosing between the
// ordinary encodings and the 2 variants.
func armThumb32Coprocessor(first, second uint32) (string, bool) {
	if (first>>8)&0xF == armThumbSIMDGroup {
		return armAdvancedSIMD32(armSIMDDataOperation | first>>12&1<<24 |
			(first&0xFF)<<16 | second)
	}
	cond := uint32(armThumbAlwaysCond)
	if first&(1<<12) != 0 {
		cond = armThumbUnusedCond
	}
	return armDecodeWord32(cond<<28 | (first&0x0FFF)<<16 | second)
}

// armDecodeWord32 decodes an ARM instruction held as a word rather than as the
// bytes it was read from.
func armDecodeWord32(raw uint32) (string, bool) {
	// #nosec G115 -- each conversion takes one byte of the word by design
	return armDecodeOne([]byte{byte(raw), byte(raw >> 8), byte(raw >> 16), byte(raw >> 24)}, armArch32, 0)
}

// armThumb32VectorTransfer decodes the Advanced SIMD element and structure
// loads and stores, which sit outside the coprocessor space in Thumb but share
// the ARM encoding below the top byte.
func armThumb32VectorTransfer(first, second uint32) (string, bool) {
	return armSIMDElementTransfer(armSIMDTransferSpace | (first&0xFF)<<16 | second)
}

// armCondSuffix names the condition an ARM instruction carries, which is blank
// for the always condition since that is the unmarked form.
func armCondSuffix(cond uint32) string {
	if cond >= 14 {
		return ""
	}
	return armConditionNames[cond]
}

// armCoprocessor32 decodes the ARM coprocessor instructions. x/arch declines
// the whole family, so they are decoded here from the same fields the Thumb
// encodings use, with a condition on top.
func armCoprocessor32(raw uint32) (string, bool) {
	coproc := (raw >> 8) & 0xF
	vfp := coproc == armCoprocVFPLow || coproc == armCoprocVFPHigh

	// The unconditional encodings are the 2 variants rather than a condition.
	// Where an instruction carries both markers the 2 comes first.
	cond := raw >> 28
	condition := armCondSuffix(cond)
	variant := ""
	if cond == 15 {
		variant = "2"
	}
	suffix := variant + condition

	switch (raw >> 25) & 7 {
	case 0b110:
		// Only the conditional encodings reach the extension registers; the
		// unconditional ones are ordinary coprocessor transfers.
		if vfp && cond != 15 {
			return armVFPLoadStore32(raw, coproc, suffix)
		}
		if (raw>>21)&0xF == 0b0010 {
			if vfp {
				return "", false // the floating-point unit has no such pair
			}
			return armCoproc32Pair(raw, coproc, suffix), true
		}
		if text := armCoproc32Memory(raw, coproc, variant, condition); text != "" {
			return text, true
		}
		return "", false
	case 0b111:
		if raw&(1<<24) != 0 {
			return "", false // a supervisor call, not a coprocessor instruction
		}
		if vfp {
			if cond == 15 {
				return armVFPUnconditional32(raw, coproc)
			}
			return armVFPDataProcessing32(raw, coproc, suffix)
		}
		return armCoproc32Data(raw, coproc, suffix), true
	}
	return "", false
}

// Extension register load and store fields. The 64-bit form never lists more
// than sixteen registers, and neither form runs past the end of its bank.
const (
	armVFPMaxDoubles     = 16
	armVFPBankRegisters  = 32
	armVFPStackPointerRn = 13 // the base register the push and pop aliases use
)

// armVFPLoadStore32 decodes the extension register load and store instructions,
// which share coprocessors 10 and 11 with the floating-point unit. An odd byte
// count selects the legacy X encoding, whose length was never architecturally
// fixed; Capstone spells those with an f prefix rather than v.
func armVFPLoadStore32(raw, coproc uint32, suffix string) (string, bool) {
	preIndexed := raw&(1<<24) != 0
	up := raw&(1<<23) != 0
	if preIndexed && raw&(1<<21) == 0 {
		return "", false // VLDR and VSTR, which x/arch decodes
	}
	if preIndexed == up {
		if !preIndexed {
			return armVFPCoreRegisterPair(raw, coproc, suffix)
		}
		return "", false // set in both is unallocated
	}

	doubles := coproc == armCoprocVFPHigh
	writeback := raw&(1<<21) != 0
	load := raw&(1<<20) != 0
	rn := (raw >> 16) & 0xF
	first, count, extended := armVFPListBounds(raw, doubles)
	if extended && first >= armVFPMaxDoubles {
		// The X form predates the upper half of the register bank, so its
		// numbering has no room for the D bit.
		return "", false
	}

	name := armVFPTransferName(load, !preIndexed, extended)
	list := armVFPRegisterList(first, count, doubles)
	if !extended && writeback && rn == armVFPStackPointerRn && load == !preIndexed {
		if load {
			return "vpop" + suffix + " " + list, true
		}
		return "vpush" + suffix + " " + list, true
	}

	base := armReg(rn)
	if writeback {
		base += "!"
	}
	return name + suffix + " " + base + ", " + list, true
}

// armVFPCoreRegisterPair decodes the transfers between two core registers and
// either one 64-bit or two consecutive 32-bit extension registers.
func armVFPCoreRegisterPair(raw, coproc uint32, suffix string) (string, bool) {
	if raw&(1<<22) == 0 || raw&(1<<21) != 0 || raw&(3<<6) != 0 || raw&(1<<4) == 0 {
		return "", false
	}

	doubles := coproc == armCoprocVFPHigh
	extra := (raw >> 5) & 1
	field := raw & 0xF
	extension := armVFPRegister(field, extra, doubles)
	if !doubles {
		first := field<<1 | extra
		if first+1 >= armVFPBankRegisters {
			return "", false // the pair would run past the end of the bank
		}
		extension += fmt.Sprintf(", s%d", first+1)
	}

	core := fmt.Sprintf("%s, %s", armReg((raw>>12)&0xF), armReg((raw>>16)&0xF))
	if raw&(1<<20) != 0 {
		return "vmov" + suffix + " " + core + ", " + extension, true
	}
	return "vmov" + suffix + " " + extension + ", " + core, true
}

// armVFPListBounds works out which extension registers a transfer covers. The
// 64-bit registers are numbered with the D bit on top and the 32-bit ones with
// it underneath, and the byte count is halved for the wider bank.
func armVFPListBounds(raw uint32, doubles bool) (first, count int, extended bool) {
	top := int(raw >> 22 & 1)
	vd := int((raw >> 12) & 0xF)
	imm8 := int(raw & 0xFF)

	limit := armVFPBankRegisters
	if doubles {
		first = top<<4 | vd
		count = imm8 / 2
		extended = imm8%2 == 1
		limit = armVFPMaxDoubles
	} else {
		first = vd<<1 | top
		count = imm8
	}
	return first, min(max(count, 1), min(limit, armVFPBankRegisters-first)), extended
}

// armVFPTransferName picks the mnemonic for an extension register transfer.
func armVFPTransferName(load, increment, extended bool) string {
	name, direction := "vstm", "db"
	if load {
		name = "vldm"
	}
	if increment {
		direction = "ia"
	}
	if extended {
		return "f" + name[1:] + direction + "x"
	}
	return name + direction
}

// Floating-point data-processing opcodes, as bits 23, 21 and 20 of the
// instruction; bit 22 numbers the destination register and takes no part.
const (
	armVFPOpcodeMask     = 0b1011
	armVFPFusedNegate    = 0b1001 // VFNMS and VFNMA
	armVFPFusedMultiply  = 0b1010 // VFMA and VFMS
	armVFPExtendedOpcode = 0b1011 // the group bits 19 to 16 index

	armVFPConvertToWide   = 0b0010 // half precision widened to 64 bits
	armVFPConvertFromWide = 0b0011 // and the same conversion back
	armVFPRoundCurrent    = 0b0110 // VRINTR, and VRINTZ on the high half
	armVFPRoundExact      = 0b0111
)

// armVFPDataProcessing32 decodes the floating-point operations x/arch has no
// table for: the fused multiply-accumulates, the rounding family, and the
// half-precision conversions that involve a 64-bit register.
func armVFPDataProcessing32(raw, coproc uint32, suffix string) (string, bool) {
	if raw&(1<<4) != 0 {
		return armVFPCoreTransfer32(raw, coproc, suffix)
	}
	doubles := coproc == armCoprocVFPHigh
	kind, rd, rn, rm := armVFPOperands32(raw, doubles)
	high := raw&(1<<6) != 0

	switch (raw >> 20) & armVFPOpcodeMask {
	case armVFPFusedNegate:
		return armVFPFused("vfnms", "vfnma", high, suffix, kind, rd, rn, rm), true
	case armVFPFusedMultiply:
		return armVFPFused("vfma", "vfms", high, suffix, kind, rd, rn, rm), true
	case armVFPExtendedOpcode:
		if !high {
			return "", false // the move-immediate, which x/arch decodes
		}
		return armVFPExtended32(raw, doubles, suffix, kind, rd, rm)
	}
	return "", false
}

// armVFPOperands32 pulls the three register operands and the type suffix out of
// a floating-point instruction, all of which sit in the same fields throughout
// the unit.
func armVFPOperands32(raw uint32, doubles bool) (kind, rd, rn, rm string) {
	kind = ".f32"
	if doubles {
		kind = ".f64"
	}
	return kind,
		armVFPRegister((raw>>12)&0xF, (raw>>22)&1, doubles),
		armVFPRegister((raw>>16)&0xF, (raw>>7)&1, doubles),
		armVFPRegister(raw&0xF, (raw>>5)&1, doubles)
}

// armVFPSelectConditions are the four conditions VSEL can test, in the order
// bits 21 and 20 number them.
var armVFPSelectConditions = [4]string{"eq", "vs", "ge", "gt"}

// armVFPRoundingModes name the four explicit rounding directions, in the order
// the selector field numbers them: to nearest with ties away from zero, to
// nearest with ties to even, towards plus infinity, and towards minus infinity.
var armVFPRoundingModes = [4]string{"a", "n", "p", "m"}

// The selector field of the round and convert group, whose low two bits give
// the rounding mode and whose bit 2 tells a rounding from a conversion.
const (
	armVFPRoundSelectorBase = 0b1000
	armVFPConvertSelector   = 0b0100
)

// armVFPUnconditional32 decodes the floating-point instructions ARMv8 added in
// the unconditional encoding space, which take their rounding or their
// condition from the encoding rather than from the status register.
func armVFPUnconditional32(raw, coproc uint32) (string, bool) {
	if raw&(1<<4) != 0 {
		return "", false
	}
	doubles := coproc == armCoprocVFPHigh
	kind, rd, rn, rm := armVFPOperands32(raw, doubles)

	if raw&(1<<23) == 0 {
		if raw&(1<<6) != 0 {
			return "", false
		}
		condition := armVFPSelectConditions[(raw>>20)&3]
		return fmt.Sprintf("vsel%s%s %s, %s, %s", condition, kind, rd, rn, rm), true
	}

	switch (raw >> 20) & 3 {
	case 0b00:
		name := "vmaxnm"
		if raw&(1<<6) != 0 {
			name = "vminnm"
		}
		return fmt.Sprintf("%s%s %s, %s, %s", name, kind, rd, rn, rm), true
	case 0b11:
		return armVFPRoundOrConvert32(raw, doubles, kind, rd, rm)
	}
	return "", false
}

// armVFPRoundOrConvert32 decodes the explicitly rounded round-to-integer and
// convert-to-integer operations, which share a selector field. A conversion
// always writes a 32-bit register whatever the width of its source.
func armVFPRoundOrConvert32(raw uint32, doubles bool, kind, rd, rm string) (string, bool) {
	selector := (raw >> 16) & 0xF
	if raw&(1<<6) == 0 || selector < armVFPRoundSelectorBase {
		return "", false
	}
	mode := armVFPRoundingModes[selector&3]

	if selector&armVFPConvertSelector == 0 {
		if raw&(1<<7) != 0 {
			return "", false
		}
		return fmt.Sprintf("vrint%s%s %s, %s", mode, kind, rd, rm), true
	}

	sign := "u"
	if raw&(1<<7) != 0 {
		sign = "s"
	}
	narrow := armVFPRegister((raw>>12)&0xF, (raw>>22)&1, false)
	return fmt.Sprintf("vcvt%s.%s32%s %s, %s", mode, sign, kind, narrow, rm), true
}

// armVFPSystemRegisters names the floating-point system registers a transfer
// can reach. The media and VFP feature registers are read-only, so VMSR takes
// the shorter list.
var armVFPSystemRegisters = map[uint32]string{
	0: "fpsid", 1: "fpscr", 5: "mvfr2", 6: "mvfr1", 7: "mvfr0",
	8: "fpexc", 9: "fpinst", 10: "fpinst2",
}

// armVFPWritableSystemRegisters are the ones VMSR may name.
var armVFPWritableSystemRegisters = map[uint32]bool{0: true, 1: true, 8: true, 9: true, 10: true}

// The opcode field of a transfer between a core and an extension register,
// taken from bits 23 to 21.
const armVFPSystemOpcode = 0b111

// armVFPCoreTransfer32 decodes the 8, 16 and 32-bit transfers between a core
// register and an extension register: a single lane, a broadcast onto every
// lane, or one of the floating-point system registers.
func armVFPCoreTransfer32(raw, coproc uint32, suffix string) (string, bool) {
	opcode := (raw >> 21) & 7
	load := raw&(1<<20) != 0
	rt := armReg((raw >> 12) & 0xF)

	if coproc == armCoprocVFPLow {
		if opcode != armVFPSystemOpcode {
			return "", false // the transfers to and from a single register
		}
		return armVFPSystemTransfer32(raw, suffix, load, rt)
	}
	if raw&0xF != 0 {
		return "", false
	}

	selector := (raw >> 21) & 3
	element := (raw >> 5) & 3
	register := (raw>>7)&1<<4 | (raw>>16)&0xF

	if !load && opcode&0b100 != 0 {
		return armVFPBroadcast32(suffix, selector, element, register, rt)
	}

	size, index, ok := armVFPLane(selector, element)
	if !ok {
		return "", false
	}
	if !load {
		return fmt.Sprintf("vmov%s.%d d%d[%d], %s", suffix, size, register, index, rt), true
	}
	if size == armVFPLaneWord {
		if raw&(1<<23) != 0 {
			return "", false // a whole word has no sign to extend
		}
		return fmt.Sprintf("vmov%s.32 %s, d%d[%d]", suffix, rt, register, index), true
	}
	sign := "s"
	if raw&(1<<23) != 0 {
		sign = "u"
	}
	return fmt.Sprintf("vmov%s.%s%d %s, d%d[%d]", suffix, sign, size, rt, register, index), true
}

// armVFPSystemTransfer32 renders VMSR and VMRS. The manual leaves several bits
// unspecified in both; Capstone insists on a clear bit 7 for the write and on
// clear reserved fields throughout for the read.
func armVFPSystemTransfer32(raw uint32, suffix string, load bool, rt string) (string, bool) {
	number := (raw >> 16) & 0xF
	name, known := armVFPSystemRegisters[number]
	if !known {
		return "", false
	}
	if !load {
		if raw&(1<<7) != 0 || !armVFPWritableSystemRegisters[number] {
			return "", false
		}
		return fmt.Sprintf("vmsr%s %s, %s", suffix, name, rt), true
	}
	if raw&(7<<5) != 0 || raw&0xF != 0 {
		return "", false
	}
	return fmt.Sprintf("vmrs%s %s, %s", suffix, rt, name), true
}

// Element widths a lane selector can name, in bits.
const (
	armVFPLaneByte     = 8
	armVFPLaneHalfword = 16
	armVFPLaneWord     = 32
)

// armVFPLane splits the element selector of a scalar transfer into the width of
// an element and the lane it picks. The selector arrives in two pieces, one
// beside the opcode and one beside the register fields.
func armVFPLane(selector, element uint32) (size, index uint32, ok bool) {
	switch {
	case selector&2 != 0:
		return armVFPLaneByte, selector&1<<2 | element, true
	case element&1 != 0:
		return armVFPLaneHalfword, selector&1<<1 | element>>1, true
	case element == 0:
		return armVFPLaneWord, selector & 1, true
	}
	return 0, 0, false
}

// armVFPBroadcastSizes give the element width of a VDUP, indexed by its two
// size bits; the fourth combination is unallocated.
var armVFPBroadcastSizes = [4]int{armVFPLaneWord, armVFPLaneHalfword, armVFPLaneByte, 0}

// armVFPBroadcast32 renders VDUP, which copies a core register onto every lane
// of a 64-bit or 128-bit register.
func armVFPBroadcast32(suffix string, selector, element, register uint32, rt string) (string, bool) {
	if element&2 != 0 {
		return "", false
	}
	size := armVFPBroadcastSizes[selector&2|element&1]
	if size == 0 {
		return "", false
	}
	if selector&1 == 0 {
		return fmt.Sprintf("vdup%s.%d d%d, %s", suffix, size, register, rt), true
	}
	if register%2 != 0 {
		return "", false // a quadword register is a pair of the 64-bit ones
	}
	return fmt.Sprintf("vdup%s.%d q%d, %s", suffix, size, register/2, rt), true
}

// armVFPFused renders one of the fused multiply-accumulate pairs, whose two
// members differ only in bit 6.
func armVFPFused(low, high string, isHigh bool, suffix, kind, rd, rn, rm string) string {
	name := low
	if isHigh {
		name = high
	}
	return fmt.Sprintf("%s%s%s %s, %s, %s", name, suffix, kind, rd, rn, rm)
}

// armVFPExtended32 decodes the two-register operations selected by bits 19 to
// 16, of which only the ones x/arch declines are handled here. Bit 7 picks the
// half of a 32-bit register a half-precision conversion uses, and separates
// VRINTR from VRINTZ.
func armVFPExtended32(raw uint32, doubles bool, suffix, kind, rd, rm string) (string, bool) {
	top := raw&(1<<7) != 0
	half := "b"
	if top {
		half = "t"
	}

	switch (raw >> 16) & 0xF {
	case armVFPConvertToWide:
		if !doubles {
			return "", false // the 32-bit conversions, which x/arch decodes
		}
		source := armVFPRegister(raw&0xF, (raw>>5)&1, false)
		return fmt.Sprintf("vcvt%s%s.f64.f16 %s, %s", half, suffix, rd, source), true
	case armVFPConvertFromWide:
		if !doubles {
			return "", false
		}
		destination := armVFPRegister((raw>>12)&0xF, (raw>>22)&1, false)
		return fmt.Sprintf("vcvt%s%s.f16.f64 %s, %s", half, suffix, destination, rm), true
	case armVFPRoundCurrent:
		name := "vrintr"
		if top {
			name = "vrintz"
		}
		return fmt.Sprintf("%s%s%s %s, %s", name, suffix, kind, rd, rm), true
	case armVFPRoundExact:
		if top {
			return "", false // the conversion between the two widths
		}
		return fmt.Sprintf("vrintx%s%s %s, %s", suffix, kind, rd, rm), true
	}
	return "", false
}

// armVFPRegister names an extension register. The 64-bit bank puts the extra
// numbering bit above the four-bit field and the 32-bit bank below it.
func armVFPRegister(field, extra uint32, doubles bool) string {
	if doubles {
		return fmt.Sprintf("d%d", extra<<4|field)
	}
	return fmt.Sprintf("s%d", field<<1|extra)
}

// armVFPRegisterList renders a run of consecutive extension registers.
func armVFPRegisterList(first, count int, doubles bool) string {
	prefix := "s"
	if doubles {
		prefix = "d"
	}
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("%s%d", prefix, first+i)
	}
	return "{" + strings.Join(names, ", ") + "}"
}

// armCoproc32Pair renders MCRR and MRRC.
func armCoproc32Pair(raw, coproc uint32, suffix string) string {
	name := "mcrr" + suffix
	if raw&(1<<20) != 0 {
		name = "mrrc" + suffix
	}
	return fmt.Sprintf("%s p%d, %s, %s, %s, c%d", name, coproc,
		armImmediate32(int64((raw>>4)&0xF)), armReg((raw>>12)&0xF), armReg((raw>>16)&0xF), raw&0xF)
}

// armCoproc32Data renders CDP and the single-register transfers MCR and MRC.
func armCoproc32Data(raw, coproc uint32, suffix string) string {
	crn, crm := (raw>>16)&0xF, raw&0xF
	opc2 := armImmediate32(int64((raw >> 5) & 7))

	if raw&(1<<4) == 0 {
		return fmt.Sprintf("cdp%s p%d, %s, c%d, c%d, c%d, %s", suffix, coproc,
			armImmediate32(int64((raw>>20)&0xF)), (raw>>12)&0xF, crn, crm, opc2)
	}

	name := "mcr" + suffix
	destination := armReg((raw >> 12) & 0xF)
	if raw&(1<<20) != 0 {
		name = "mrc" + suffix
		destination = armCoprocDestination((raw >> 12) & 0xF)
	}
	return fmt.Sprintf("%s p%d, %s, %s, c%d, c%d, %s", name, coproc,
		armImmediate32(int64((raw>>21)&7)), destination, crn, crm, opc2)
}

// armCoprocDestination names the register a coprocessor read writes back to.
// Naming the program counter there sets the condition flags instead.
func armCoprocDestination(rt uint32) string {
	if rt == armProgramCounter {
		return "apsr_nzcv"
	}
	return armReg(rt)
}

// armCoproc32Memory renders LDC and STC.
func armCoproc32Memory(raw, coproc uint32, variant, condition string) string {
	name := "stc"
	if raw&(1<<20) != 0 {
		name = "ldc"
	}
	name += variant
	if raw&(1<<22) != 0 {
		name += "l"
	}
	name += condition

	crd, rn := (raw>>12)&0xF, (raw>>16)&0xF
	preIndexed := raw&(1<<24) != 0
	writeback := raw&(1<<21) != 0
	offset := int64(raw&0xFF) * 4
	negative := raw&(1<<23) == 0
	if negative {
		offset = -offset
	}
	// Capstone keeps the sign even when the displacement is zero.
	displacement := armImmediate32(offset)
	if negative && offset == 0 {
		displacement = "#-0"
	}

	head := fmt.Sprintf("%s p%d, c%d, ", name, coproc, crd)
	switch {
	case !preIndexed && !writeback:
		if negative {
			return "" // the unindexed option form is only defined for U set
		}
		option := strings.TrimPrefix(armImmediate32(int64(raw&0xFF)), "#")
		return head + fmt.Sprintf("[%s], {%s}", armReg(rn), option)
	case !preIndexed:
		return head + fmt.Sprintf("[%s], %s", armReg(rn), displacement)
	}
	// A displacement of nothing at all is left out, but only where the word
	// neither writes the base back nor counts downwards.
	out := head + "[" + armReg(rn) + "]"
	if offset != 0 || negative || writeback {
		out = head + "[" + armReg(rn) + ", " + displacement + "]"
	}
	if writeback {
		out += "!"
	}
	return out
}

// armRotatedImmediate32 rewrites the ARM immediate that carries its own
// rotation. Capstone prints both halves in decimal and marks the rotation as an
// immediate; x/arch prints the value in hexadecimal and the rotation bare. It
// runs after the general rewriting, which would otherwise put the value back
// into hexadecimal.
func armRotatedImmediate32(inst armasm.Inst, operands string) string {
	for _, arg := range inst.Args {
		alt, ok := arg.(armasm.ImmAlt)
		if !ok {
			continue
		}
		rewritten := armImmediate32(int64(alt.Val)) + ", " + strconv.FormatUint(uint64(alt.Rot), 10)
		return strings.Replace(operands, rewritten, fmt.Sprintf("#%d, #%d", alt.Val, alt.Rot), 1)
	}
	return operands
}

// armThumbHalfwordWidthBit lies above the width a halfword clamp counts to.
const armThumbHalfwordWidthBit = 1 << 4

// armThumbPlainImmediateBit carries the top bit of a plain immediate, and is
// fixed for the operations of this group that hold no immediate.
const armThumbPlainImmediateBit = 1 << 10

// armThumb32Saturate renders SSAT and USAT, which clamp a shifted register to a
// given bit width. The signed form counts from one, the unsigned from zero.
func armThumb32Saturate(first, second uint32) (string, bool) {
	op := (first >> 4) & 0x1F
	rn, rd := first&0xF, (second>>8)&0xF
	saturate := second & 0x1F
	amount := (second>>12)&7<<2 | (second>>6)&3
	arithmetic := op&0b00010 != 0
	signed := op&0b01000 == 0

	// The bit below the shift amount is fixed. Capstone reads past it for the
	// unsigned clamps, but not for the signed ones nor for either halfword one.
	if second&(1<<5) != 0 && (signed || (arithmetic && amount == 0)) {
		return "", false
	}

	name := "usat"
	if signed {
		name = "ssat"
		saturate++
	}
	if arithmetic && amount == 0 {
		// A shift of nothing selects the form that clamps each halfword, whose
		// unsigned spelling alone keeps the bit the others read past, and the
		// width it clamps to counts in one bit fewer than the whole-word form.
		if second&armThumbHalfwordWidthBit != 0 {
			return "", false
		}
		if !signed && first&armThumbPlainImmediateBit != 0 {
			return "", false
		}
		return fmt.Sprintf("%s16 %s, %s, %s", name, armReg(rd),
			armImmediate32(int64(saturate)), armReg(rn)), true
	}

	out := fmt.Sprintf("%s %s, %s, %s", name, armReg(rd), armImmediate32(int64(saturate)), armReg(rn))
	if arithmetic {
		return out + ", asr " + armImmediate32(int64(amount)), true
	}
	if amount != 0 {
		return out + ", lsl " + armImmediate32(int64(amount)), true
	}
	return out, true
}

// armDecodeARM32Extra decodes the ARM families golang.org/x/arch declines: the
// coprocessor instructions, the block transfers that name the user-mode
// registers, and the parallel arithmetic.
func armDecodeARM32Extra(raw uint32) (string, bool) {
	if text, ok := armAdvancedSIMD32(raw); ok {
		return text, true
	}
	if text, ok := armRegisterOffset32(raw); ok {
		return text, true
	}
	if text, ok := armCoprocessor32(raw); ok {
		return text, true
	}
	if text, ok := armBlockTransfer32(raw); ok {
		return text, true
	}
	if text, ok := armExceptionTransfer32(raw); ok {
		return text, true
	}
	if text, ok := armChangeState32(raw); ok {
		return text, true
	}
	if text, ok := armExtraLoadStore32(raw); ok {
		return text, true
	}
	return armParallelArithmetic32(raw)
}

// armBlockAddressing names the four addressing orders a block transfer can use,
// indexed by its pre-index and increment bits. Increment-after is the default
// and carries no suffix.
var armBlockAddressing = [4]string{"da", "", "db", "ib"}

// armBlockTransfer32 renders LDM and STM. x/arch declines the forms with the S
// bit set, which transfer the user-mode registers and print a trailing caret.
func armBlockTransfer32(raw uint32) (string, bool) {
	if (raw>>25)&7 != 0b100 {
		return "", false
	}
	cond := raw >> 28
	if cond == 15 {
		return "", false
	}

	name := "stm"
	if raw&(1<<20) != 0 {
		name = "ldm"
	}
	name += armBlockAddressing[(raw>>23)&3] + armCondSuffix(cond)

	out := name + " " + armReg((raw>>16)&0xF)
	if raw&(1<<21) != 0 {
		out += "!"
	}
	out += ", " + armRegList(raw&0xFFFF)
	if raw&(1<<22) != 0 {
		out += " ^" // the user-mode register bank
	}
	return out, true
}

// How the data type of an Advanced SIMD instruction is spelled. Most integer
// operations take their signedness from the U bit; a few fix it, and the
// bitwise ones name no type at all.
type armSIMDType byte

const (
	armSIMDSigned   armSIMDType = 's' // signed or unsigned, chosen by the U bit
	armSIMDInteger  armSIMDType = 'i'
	armSIMDBare     armSIMDType = 'b' // the element size alone
	armSIMDFixedSig armSIMDType = 'S' // always signed, whatever the U bit says
	armSIMDFixedUns armSIMDType = 'U' // and always unsigned
	armSIMDPoly     armSIMDType = 'p'
	armSIMDNone     armSIMDType = '-'
	armSIMDFloat    armSIMDType = 'f'
)

// armSIMDOp describes one entry of an Advanced SIMD opcode table.
type armSIMDOp struct {
	name  string
	kind  armSIMDType
	sizes uint8 // one bit per element size the encoding allows
	solo  bool  // the pairwise operations have no 128-bit form
	swap  bool  // the shifts name the register they shift first
}

// Element size sets, as a bit per value of the two-bit size field.
const (
	armSizeAll   = 0b1111
	armSizeShort = 0b0111 // every size but 64-bit
	armSizeWide  = 0b0110 // the two middle sizes, used by the doubling multiplies
	armSizeByte  = 0b0001
)

// armSIMDIntegerSame is the integer half of the three-registers-of-the-same-
// length table, indexed by the opcode field, bit 4, and the U bit.
var armSIMDIntegerSame = [12][2][2]armSIMDOp{
	0b0000: {
		{{"vhadd", armSIMDSigned, armSizeShort, false, false}, {"vhadd", armSIMDSigned, armSizeShort, false, false}},
		{{"vqadd", armSIMDSigned, armSizeAll, false, false}, {"vqadd", armSIMDSigned, armSizeAll, false, false}},
	},
	0b0001: {
		{{"vrhadd", armSIMDSigned, armSizeShort, false, false}, {"vrhadd", armSIMDSigned, armSizeShort, false, false}},
	},
	0b0010: {
		{{"vhsub", armSIMDSigned, armSizeShort, false, false}, {"vhsub", armSIMDSigned, armSizeShort, false, false}},
		{{"vqsub", armSIMDSigned, armSizeAll, false, false}, {"vqsub", armSIMDSigned, armSizeAll, false, false}},
	},
	0b0011: {
		{{"vcgt", armSIMDSigned, armSizeShort, false, false}, {"vcgt", armSIMDSigned, armSizeShort, false, false}},
		{{"vcge", armSIMDSigned, armSizeShort, false, false}, {"vcge", armSIMDSigned, armSizeShort, false, false}},
	},
	0b0100: {
		{{"vshl", armSIMDSigned, armSizeAll, false, true}, {"vshl", armSIMDSigned, armSizeAll, false, true}},
		{{"vqshl", armSIMDSigned, armSizeAll, false, true}, {"vqshl", armSIMDSigned, armSizeAll, false, true}},
	},
	0b0101: {
		{{"vrshl", armSIMDSigned, armSizeAll, false, true}, {"vrshl", armSIMDSigned, armSizeAll, false, true}},
		{{"vqrshl", armSIMDSigned, armSizeAll, false, true}, {"vqrshl", armSIMDSigned, armSizeAll, false, true}},
	},
	0b0110: {
		{{"vmax", armSIMDSigned, armSizeShort, false, false}, {"vmax", armSIMDSigned, armSizeShort, false, false}},
		{{"vmin", armSIMDSigned, armSizeShort, false, false}, {"vmin", armSIMDSigned, armSizeShort, false, false}},
	},
	0b0111: {
		{{"vabd", armSIMDSigned, armSizeShort, false, false}, {"vabd", armSIMDSigned, armSizeShort, false, false}},
		{{"vaba", armSIMDSigned, armSizeShort, false, false}, {"vaba", armSIMDSigned, armSizeShort, false, false}},
	},
	0b1000: {
		{{"vadd", armSIMDInteger, armSizeAll, false, false}, {"vsub", armSIMDInteger, armSizeAll, false, false}},
		{{"vtst", armSIMDBare, armSizeShort, false, false}, {"vceq", armSIMDInteger, armSizeShort, false, false}},
	},
	0b1001: {
		{{"vmla", armSIMDInteger, armSizeShort, false, false}, {"vmls", armSIMDInteger, armSizeShort, false, false}},
		{{"vmul", armSIMDInteger, armSizeShort, false, false}, {"vmul", armSIMDPoly, armSizeByte, false, false}},
	},
	0b1010: {
		{{"vpmax", armSIMDSigned, armSizeShort, true, false}, {"vpmax", armSIMDSigned, armSizeShort, true, false}},
		{{"vpmin", armSIMDSigned, armSizeShort, true, false}, {"vpmin", armSIMDSigned, armSizeShort, true, false}},
	},
	0b1011: {
		{{"vqdmulh", armSIMDFixedSig, armSizeWide, false, false}, {"vqrdmulh", armSIMDFixedSig, armSizeWide, false, false}},
		{{"vpadd", armSIMDInteger, armSizeShort, true, false}, {}},
	},
}

// armSIMDFloatSame is the floating-point half of the same table. Bit 21 of the
// size field chooses between the two members of each pair; the other two size
// values are unallocated.
var armSIMDFloatSame = [4][2][2][2]armSIMDOp{
	0b00: {1: {0: {1: {"vfms", armSIMDFloat, 0, false, false}, 0: {"vfma", armSIMDFloat, 0, false, false}}}},
	0b01: {
		0: {
			0: {0: {"vadd", armSIMDFloat, 0, false, false}, 1: {"vsub", armSIMDFloat, 0, false, false}},
			1: {0: {"vpadd", armSIMDFloat, 0, true, false}, 1: {"vabd", armSIMDFloat, 0, false, false}},
		},
		1: {
			0: {0: {"vmla", armSIMDFloat, 0, false, false}, 1: {"vmls", armSIMDFloat, 0, false, false}},
			1: {0: {"vmul", armSIMDFloat, 0, false, false}},
		},
	},
	0b10: {
		0: {
			0: {0: {"vceq", armSIMDFloat, 0, false, false}},
			1: {0: {"vcge", armSIMDFloat, 0, false, false}, 1: {"vcgt", armSIMDFloat, 0, false, false}},
		},
		1: {1: {0: {"vacge", armSIMDFloat, 0, false, false}, 1: {"vacgt", armSIMDFloat, 0, false, false}}},
	},
	0b11: {
		0: {
			0: {0: {"vmax", armSIMDFloat, 0, false, false}, 1: {"vmin", armSIMDFloat, 0, false, false}},
			1: {0: {"vpmax", armSIMDFloat, 0, true, false}, 1: {"vpmin", armSIMDFloat, 0, true, false}},
		},
		1: {0: {0: {"vrecps", armSIMDFloat, 0, false, false}, 1: {"vrsqrts", armSIMDFloat, 0, false, false}}},
	},
}

// armSIMDBitwiseOps are the eight untyped register operations that share one
// opcode, chosen by the U bit and the size field.
var armSIMDBitwiseOps = [2][4]string{
	{"vand", "vbic", "vorr", "vorn"},
	{"veor", "vbsl", "vbit", "vbif"},
}

// Opcode boundaries within the three-registers-of-the-same-length table.
const (
	armSIMDBitwiseOpcode = 0b0001 // with bit 4 set
	armSIMDFloatOpcode   = 0b1100 // the first of the four floating-point opcodes
)

// armAdvancedSIMD32 decodes the Advanced SIMD instructions, which occupy the
// unconditional half of two encoding spaces of their own.
func armAdvancedSIMD32(raw uint32) (string, bool) {
	if raw>>28 != 15 {
		return "", false
	}
	if (raw>>24)&0xF == 0b0100 {
		return armSIMDElementTransfer(raw)
	}
	if (raw>>25)&7 != 0b001 {
		return "", false
	}
	if raw&(1<<23) == 0 {
		return armSIMDThreeSame(raw)
	}
	if raw&(1<<4) != 0 {
		// The shift amount doubles as the selector: all zero above the low
		// three bits means there is no shift and the word carries an immediate.
		if raw&(1<<7) == 0 && (raw>>19)&7 == 0 {
			return armSIMDModifiedImmediate(raw)
		}
		return armSIMDTwoShift(raw)
	}
	if (raw>>20)&3 == armSIMDWidestSize {
		if raw&(1<<24) == 0 {
			return armSIMDExtract(raw)
		}
		return armSIMDPermuteOrConvert(raw)
	}
	if raw&(1<<6) != 0 {
		return armSIMDScalar(raw)
	}
	return armSIMDThreeLong(raw)
}

// The widest element a byte offset can be rounded up to, for each register
// width. Capstone names the largest element the offset divides evenly.
const (
	armSIMDExtractWidestD = 2
	armSIMDExtractWidestQ = 3
)

// armSIMDExtract decodes VEXT, which reads a window spanning two registers.
// Capstone restates the byte offset in terms of the widest element that
// divides it, so an offset of four becomes one 32-bit element.
func armSIMDExtract(raw uint32) (string, bool) {
	offset := raw >> 8 & 0xF
	quad := raw&(1<<6) != 0
	if !quad && offset >= 8 {
		return "", false // the window would run past a 64-bit register
	}

	widest := uint32(armSIMDExtractWidestD)
	if quad {
		widest = armSIMDExtractWidestQ
	}
	size := widest
	for step := uint32(0); step < widest; step++ {
		if offset&(1<<step) != 0 {
			size = step
			break
		}
	}

	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, quad)
	rn, okn := armSIMDRegister((raw>>16)&0xF, (raw>>7)&1, quad)
	rm, okm := armSIMDRegister(raw&0xF, (raw>>5)&1, quad)
	if !okd || !okn || !okm {
		return "", false
	}
	return fmt.Sprintf("vext.%d %s, %s, %s, %s", armVFPLaneByte<<size,
		rd, rn, rm, armImmediate32(int64(offset>>size))), true
}

// armSIMDMiscShape is how wide the two operands of a miscellaneous operation
// are relative to each other.
type armSIMDMiscShape byte

const (
	armMiscSame armSIMDMiscShape = iota
	armMiscNarrow
	armMiscWiden
)

// armSIMDMiscOp describes one entry of the two-registers-miscellaneous table.
type armSIMDMiscOp struct {
	name  string
	kind  armSIMDType
	sizes uint8
	shape armSIMDMiscShape
	zero  bool // the comparisons name the zero they test against
	shift bool // the maximum-shift move spells its element size out
}

// Element size sets used only by the miscellaneous operations.
const (
	armSizeBytePair = 0b0011 // 8 and 16-bit elements
	armSizeHalfword = 0b0010
	armSizeWord     = 0b0100
)

// armSIMDMiscellaneous is the two-registers-miscellaneous table, indexed by
// bits 17 and 16 and then by bits 10 to 6. Bit 6 usually chooses the width of
// the operands, but in the third column it is part of the opcode.
var armSIMDMiscellaneous = [4][32]armSIMDMiscOp{
	0b00: {
		0b00000: {"vrev64", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b00001: {"vrev64", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b00010: {"vrev32", armSIMDBare, armSizeBytePair, armMiscSame, false, false},
		0b00011: {"vrev32", armSIMDBare, armSizeBytePair, armMiscSame, false, false},
		0b00100: {"vrev16", armSIMDBare, armSizeByte, armMiscSame, false, false},
		0b00101: {"vrev16", armSIMDBare, armSizeByte, armMiscSame, false, false},
		0b01000: {"vpaddl", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b01001: {"vpaddl", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b01010: {"vpaddl", armSIMDFixedUns, armSizeShort, armMiscSame, false, false},
		0b01011: {"vpaddl", armSIMDFixedUns, armSizeShort, armMiscSame, false, false},
		0b10000: {"vcls", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b10001: {"vcls", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b10010: {"vclz", armSIMDInteger, armSizeShort, armMiscSame, false, false},
		0b10011: {"vclz", armSIMDInteger, armSizeShort, armMiscSame, false, false},
		0b10100: {"vcnt", armSIMDBare, armSizeByte, armMiscSame, false, false},
		0b10101: {"vcnt", armSIMDBare, armSizeByte, armMiscSame, false, false},
		0b10110: {"vmvn", armSIMDNone, armSizeByte, armMiscSame, false, false},
		0b10111: {"vmvn", armSIMDNone, armSizeByte, armMiscSame, false, false},
		0b11000: {"vpadal", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b11001: {"vpadal", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b11010: {"vpadal", armSIMDFixedUns, armSizeShort, armMiscSame, false, false},
		0b11011: {"vpadal", armSIMDFixedUns, armSizeShort, armMiscSame, false, false},
		0b11100: {"vqabs", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b11101: {"vqabs", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b11110: {"vqneg", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b11111: {"vqneg", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
	},
	0b01: {
		0b00000: {"vcgt", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b00001: {"vcgt", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b00010: {"vcge", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b00011: {"vcge", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b00100: {"vceq", armSIMDInteger, armSizeShort, armMiscSame, true, false},
		0b00101: {"vceq", armSIMDInteger, armSizeShort, armMiscSame, true, false},
		0b00110: {"vcle", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b00111: {"vcle", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b01000: {"vclt", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b01001: {"vclt", armSIMDFixedSig, armSizeShort, armMiscSame, true, false},
		0b01100: {"vabs", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b01101: {"vabs", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b01110: {"vneg", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b01111: {"vneg", armSIMDFixedSig, armSizeShort, armMiscSame, false, false},
		0b10000: {"vcgt", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10001: {"vcgt", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10010: {"vcge", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10011: {"vcge", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10100: {"vceq", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10101: {"vceq", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10110: {"vcle", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b10111: {"vcle", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b11000: {"vclt", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b11001: {"vclt", armSIMDFloat, armSizeWord, armMiscSame, true, false},
		0b11100: {"vabs", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b11101: {"vabs", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b11110: {"vneg", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b11111: {"vneg", armSIMDFloat, armSizeWord, armMiscSame, false, false},
	},
	0b10: {
		0b00000: {"vswp", armSIMDNone, armSizeByte, armMiscSame, false, false},
		0b00001: {"vswp", armSIMDNone, armSizeByte, armMiscSame, false, false},
		0b00010: {"vtrn", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b00011: {"vtrn", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b00100: {"vuzp", armSIMDBare, armSizeBytePair, armMiscSame, false, false},
		0b00101: {"vuzp", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b00110: {"vzip", armSIMDBare, armSizeBytePair, armMiscSame, false, false},
		0b00111: {"vzip", armSIMDBare, armSizeShort, armMiscSame, false, false},
		0b01000: {"vmovn", armSIMDInteger, armSizeShort, armMiscNarrow, false, false},
		0b01001: {"vqmovun", armSIMDFixedSig, armSizeShort, armMiscNarrow, false, false},
		0b01010: {"vqmovn", armSIMDFixedSig, armSizeShort, armMiscNarrow, false, false},
		0b01011: {"vqmovn", armSIMDFixedUns, armSizeShort, armMiscNarrow, false, false},
		0b01100: {"vshll", armSIMDInteger, armSizeShort, armMiscWiden, false, true},
		0b11000: {"vcvt.f16.f32", armSIMDNone, armSizeHalfword, armMiscNarrow, false, false},
		0b11100: {"vcvt.f32.f16", armSIMDNone, armSizeHalfword, armMiscWiden, false, false},
	},
	0b11: {
		0b10000: {"vrecpe", armSIMDFixedUns, armSizeWord, armMiscSame, false, false},
		0b10001: {"vrecpe", armSIMDFixedUns, armSizeWord, armMiscSame, false, false},
		0b10010: {"vrsqrte", armSIMDFixedUns, armSizeWord, armMiscSame, false, false},
		0b10011: {"vrsqrte", armSIMDFixedUns, armSizeWord, armMiscSame, false, false},
		0b10100: {"vrecpe", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b10101: {"vrecpe", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b10110: {"vrsqrte", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b10111: {"vrsqrte", armSIMDFloat, armSizeWord, armMiscSame, false, false},
		0b11000: {"vcvt.f32.s32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11001: {"vcvt.f32.s32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11010: {"vcvt.f32.u32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11011: {"vcvt.f32.u32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11100: {"vcvt.s32.f32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11101: {"vcvt.s32.f32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11110: {"vcvt.u32.f32", armSIMDNone, armSizeWord, armMiscSame, false, false},
		0b11111: {"vcvt.u32.f32", armSIMDNone, armSizeWord, armMiscSame, false, false},
	},
}

// The four opcode groups that share the top of the Advanced SIMD space, read
// from bits 11 to 8.
const (
	armSIMDLookupOpcode    = 0b1000 // and 10xx generally
	armSIMDBroadcastOpcode = 0b1100
)

// armSIMDPermuteOrConvert decodes the operations that take two registers and no
// third operand, together with the table lookup and the lane broadcast that
// share the same corner of the encoding.
func armSIMDPermuteOrConvert(raw uint32) (string, bool) {
	switch opcode := raw >> 8 & 0xF; {
	case opcode&0b1000 == 0:
		return armSIMDTwoMiscellaneous(raw)
	case opcode&0b1100 == armSIMDLookupOpcode:
		return armSIMDTableLookup(raw)
	case opcode == armSIMDBroadcastOpcode:
		return armSIMDBroadcastLane(raw)
	}
	return "", false
}

// armSIMDTwoMiscellaneous decodes the operations that read one vector register
// and write another.
func armSIMDTwoMiscellaneous(raw uint32) (string, bool) {
	size := raw >> 18 & 3
	selector := raw >> 6 & 0x1F
	entry := armSIMDMiscellaneous[raw>>16&3][selector]
	if entry.name == "" || entry.sizes&(1<<size) == 0 {
		return "", false
	}

	// Bit 6 chooses the width of both operands, except where an operation
	// already fixes them, in which case it is part of the opcode instead.
	quad := selector&1 != 0
	wideDest := entry.shape == armMiscWiden || (entry.shape == armMiscSame && quad)
	wideSource := entry.shape == armMiscNarrow || (entry.shape == armMiscSame && quad)
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, wideDest)
	rm, okm := armSIMDRegister(raw&0xF, (raw>>5)&1, wideSource)
	if !okd || !okm {
		return "", false
	}

	// A narrowing move names the width it reads, which is one size up.
	typeSize := size
	if entry.shape == armMiscNarrow {
		typeSize++
	}
	elements := uint32(armVFPLaneByte) << size
	name := entry.name + armSIMDSuffix(entry.kind, 0, typeSize)
	switch {
	case entry.zero:
		return fmt.Sprintf("%s %s, %s, #0", name, rd, rm), true
	case entry.shift:
		return fmt.Sprintf("%s %s, %s, %s", name, rd, rm, armImmediate32(int64(elements))), true
	}
	return fmt.Sprintf("%s %s, %s", name, rd, rm), true
}

// armSIMDTableLookup decodes VTBL and VTBX, which index a run of up to four
// registers.
func armSIMDTableLookup(raw uint32) (string, bool) {
	name := "vtbl.8"
	if raw&(1<<6) != 0 {
		name = "vtbx.8"
	}
	first := int(raw>>7&1<<4 | raw>>16&0xF)
	count := int(raw>>8&3) + 1
	if count == 2 && first == armVFPBankRegisters-1 {
		// Capstone has no consecutive pair beginning at the last register,
		// though it happily names longer tables that run past it.
		return "", false
	}

	names := make([]string, count)
	for i := range names {
		names[i] = armSIMDTableRegister(first + i)
	}
	return fmt.Sprintf("%s d%d, {%s}, d%d", name, raw>>22&1<<4|raw>>12&0xF,
		strings.Join(names, ", "), raw>>5&1<<4|raw&0xF), true
}

// armSIMDPastBank are the names Capstone gives a table that runs off the end of
// the register bank: it keeps reading its own register table, where the
// floating-point system registers come next.
var armSIMDPastBank = [3]string{"fpinst2", "mvfr0", "mvfr1"}

// armSIMDTableRegister names one entry of a lookup table.
func armSIMDTableRegister(number int) string {
	if number < armVFPBankRegisters {
		return fmt.Sprintf("d%d", number)
	}
	return armSIMDPastBank[number-armVFPBankRegisters]
}

// armSIMDLaneSize reads the element size out of a lane selector, which marks it
// with its lowest set bit and holds the lane index above.
func armSIMDLaneSize(selector uint32) (uint32, bool) {
	for size := range uint32(armSIMDWidestSize) {
		if selector&(1<<size) != 0 {
			return size, true
		}
	}
	return 0, false
}

// armSIMDBroadcastLane decodes the VDUP that copies one lane of a register onto
// every lane of another. The lane selector marks the element size with its
// lowest set bit, the way the shift amounts do.
func armSIMDBroadcastLane(raw uint32) (string, bool) {
	if raw&(1<<7) != 0 {
		return "", false
	}
	selector := raw >> 16 & 0xF
	size, ok := armSIMDLaneSize(selector)
	if !ok {
		return "", false
	}

	quad := raw&(1<<6) != 0
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, quad)
	if !okd {
		return "", false
	}
	return fmt.Sprintf("vdup.%d %s, d%d[%d]", armVFPLaneByte<<size, rd,
		raw>>5&1<<4|raw&0xF, selector>>(size+1)), true
}

// armSIMDStructure describes one entry of the load-and-store-multiple table:
// how many structures an element has, how many registers the transfer covers,
// and whether those registers are consecutive or every other one.
type armSIMDStructure struct {
	structures int
	registers  int
	spacing    int
}

// armSIMDStructures is the type field of a multiple-structure transfer.
var armSIMDStructures = [16]armSIMDStructure{
	0b0000: {4, 4, 1},
	0b0001: {4, 4, 2},
	0b0010: {1, 4, 1},
	0b0011: {2, 4, 1},
	0b0100: {3, 3, 1},
	0b0101: {3, 3, 2},
	0b0110: {1, 3, 1},
	0b0111: {1, 1, 1},
	0b1000: {2, 2, 1},
	0b1001: {2, 2, 2},
	0b1010: {1, 2, 1},
}

// armSIMDAlignLimit gives the largest alignment field a transfer of so many
// registers accepts. Three registers span an odd number of quadwords, so they
// stop where two would.
var armSIMDAlignLimit = [5]uint32{1: 0b01, 2: 0b10, 3: 0b01, 4: 0b11}

// The smallest alignment a multiple-structure transfer can name, in bits; the
// field counts upwards in doublings from there.
const armSIMDAlignBase = 32

// armSIMDElementTransfer decodes the Advanced SIMD element and structure loads
// and stores, which move whole registers, one lane, or one lane broadcast onto
// all of them.
func armSIMDElementTransfer(raw uint32) (string, bool) {
	if raw&(1<<20) != 0 {
		return "", false
	}
	load := raw&(1<<21) != 0
	if raw&(1<<23) != 0 {
		return armSIMDLaneTransfer(raw, load)
	}
	return armSIMDMultipleTransfer(raw, load)
}

// armSIMDLaneAlignments give the alignment, in bits, that a single-lane
// transfer names, indexed by element size, by the number of structures less
// one, and by the alignment field. A zero forbids the combination; a field of
// zero always means no alignment at all.
var armSIMDLaneAlignments = [3][4][4]uint32{
	0b00: {1: {1: 16}, 3: {1: 32}},
	0b01: {0: {1: 16}, 1: {1: 32}, 3: {1: 64}},
	0b10: {0: {0b11: 32}, 1: {0b01: 64}, 3: {0b01: 64, 0b10: 128}},
}

// armSIMDBroadcastAlignments are the same for a transfer that fills every lane,
// whose alignment field is one bit wide. The fourth element size is not a size
// at all: it marks the widest alignment the four-structure form can name.
var armSIMDBroadcastAlignments = [4][4]uint32{
	0b00: {1: 16, 3: 32},
	0b01: {0: 16, 1: 32, 3: 64},
	0b10: {0: 32, 1: 64, 3: 64},
	0b11: {3: 128},
}

// The element size whose lane selector carries two alignment bits.
const armSIMDLaneWordSize = 0b10

// armSIMDLaneTransfer renders a transfer of one lane, or of one lane broadcast
// onto every lane of its registers.
func armSIMDLaneTransfer(raw uint32, load bool) (string, bool) {
	size := raw >> 10 & 3
	structures := int(raw>>8&3) + 1
	selector := raw >> 4 & 0xF

	if size == armSIMDWidestSize {
		if !load {
			return "", false
		}
		return armSIMDBroadcastTransfer(raw, structures, selector)
	}

	index, spacing, alignment, ok := armSIMDLaneSelector(size, structures, selector)
	if !ok {
		return "", false
	}

	// A single-lane transfer never names a register beyond the bank.
	list, ok := armSIMDLaneList(int(raw>>22&1<<4|raw>>12&0xF), structures, spacing,
		fmt.Sprintf("[%d]", index), armOverrunReject)
	if !ok {
		return "", false
	}
	name := fmt.Sprintf("%s%d.%d", armSIMDTransferName(load), structures, armVFPLaneByte<<size)
	return name + " " + list + ", " + armSIMDAddress(raw, alignment), true
}

// armSIMDLaneSelector splits the field below the index register into a lane
// number, a register spacing, and an alignment flag. The wider the element the
// fewer lanes there are, so the field gives up its high bits as the size grows.
func armSIMDLaneSelector(size uint32, structures int, selector uint32) (index, spacing int, alignment uint32, ok bool) {
	var alignWidth uint32 = 1
	if size == armSIMDLaneWordSize {
		alignWidth = 2
	}
	index = int(selector >> (alignWidth + min(size, 1)))
	spacing = 1

	// Every size but the smallest keeps a spacing bit above the alignment.
	if size != 0 && selector>>alignWidth&1 != 0 {
		if structures == 1 {
			return 0, 0, 0, false // a lone register has nothing to skip
		}
		spacing = 2
	}

	field := selector & (1<<alignWidth - 1)
	if field == 0 {
		return index, spacing, 0, true
	}
	alignment = armSIMDLaneAlignments[size][structures-1][field]
	return index, spacing, alignment, alignment != 0
}

// armSIMDBroadcastTransfer renders a load that copies one element onto every
// lane. Its selector carries the element size rather than a lane index, and the
// single-structure form uses the spacing bit to ask for a second register.
func armSIMDBroadcastTransfer(raw uint32, structures int, selector uint32) (string, bool) {
	size := selector >> 2
	alignment := uint32(0)
	switch {
	case selector&1 != 0:
		if alignment = armSIMDBroadcastAlignments[size][structures-1]; alignment == 0 {
			return "", false
		}
	case size == armSIMDWidestSize:
		return "", false // that field marks an alignment, so it needs the bit
	}
	if size == armSIMDWidestSize {
		size = armSIMDLaneWordSize
	}

	registers, spacing := structures, 1
	if selector&2 != 0 {
		if structures == 1 {
			registers = 2
		} else {
			spacing = 2
		}
	}

	list, ok := armSIMDLaneList(int(raw>>22&1<<4|raw>>12&0xF), registers, spacing, "[]",
		armSIMDStructureOverrun(structures, registers))
	if !ok {
		return "", false
	}
	name := fmt.Sprintf("vld%d.%d", structures, armVFPLaneByte<<size)
	return name + " " + list + ", " + armSIMDAddress(raw, alignment), true
}

// armSIMDMultipleTransfer renders a transfer of whole registers.
func armSIMDMultipleTransfer(raw uint32, load bool) (string, bool) {
	entry := armSIMDStructures[raw>>8&0xF]
	size := raw >> 6 & 3
	align := raw >> 4 & 3
	if entry.registers == 0 || align > armSIMDAlignLimit[entry.registers] {
		return "", false
	}
	if size == armSIMDWidestSize && entry.structures != 1 {
		return "", false
	}

	list, ok := armSIMDLaneList(int(raw>>22&1<<4|raw>>12&0xF), entry.registers, entry.spacing, "",
		armSIMDStructureOverrun(entry.structures, entry.registers))
	if !ok {
		return "", false
	}
	name := fmt.Sprintf("%s%d.%d", armSIMDTransferName(load), entry.structures, armVFPLaneByte<<size)
	return name + " " + list + ", " + armSIMDAddress(raw, alignmentBits(align)), true
}

// alignmentBits turns an alignment field into the number of bits it stands for,
// or zero where the transfer names no alignment at all.
func alignmentBits(align uint32) uint32 {
	if align == 0 {
		return 0
	}
	return armSIMDAlignBase << align
}

// armSIMDTransferName gives the stem of a vector transfer mnemonic.
func armSIMDTransferName(load bool) string {
	if load {
		return "vld"
	}
	return "vst"
}

// The smallest structure whose registers Capstone holds in a tuple that counts
// round the bank rather than stopping at its end.
const armSIMDTupleStructures = 3

// armSIMDStructureOverrun picks how a whole-register transfer handles a list
// that runs past the last register: the wider structures come from a tuple that
// counts round, while a plain list of three or more keeps reading names.
func armSIMDStructureOverrun(structures, registers int) armSIMDOverrun {
	switch {
	case structures >= armSIMDTupleStructures:
		return armOverrunWrap
	case registers >= armSIMDTupleStructures:
		return armOverrunPastBank
	}
	return armOverrunReject
}

// armSIMDOverrun says what becomes of a register list that runs past the end of
// the bank. Capstone's answer depends on which of its register classes the
// encoding uses, so it differs from one form of transfer to the next.
type armSIMDOverrun byte

const (
	armOverrunReject   armSIMDOverrun = iota
	armOverrunWrap                    // the structure classes count round again
	armOverrunPastBank                // the plain lists keep reading register names
)

// armSIMDLaneList renders the registers a structure transfer covers, each with
// the same lane suffix.
func armSIMDLaneList(first, count, spacing int, lane string, overrun armSIMDOverrun) (string, bool) {
	names := make([]string, count)
	for i := range names {
		number := first + i*spacing
		switch {
		case number < armVFPBankRegisters:
		case overrun == armOverrunWrap:
			number %= armVFPBankRegisters
		case overrun == armOverrunPastBank:
			names[i] = armSIMDTableRegister(number) + lane
			continue
		default:
			return "", false
		}
		names[i] = fmt.Sprintf("d%d%s", number, lane)
	}
	return "{" + strings.Join(names, ", ") + "}", true
}

// armSIMDAddress renders the addressing of a structure transfer: a base
// register with an optional alignment, then the writeback the index register
// selects. Naming the stack pointer there means write back without an index,
// and naming the program counter means leave the base alone.
func armSIMDAddress(raw, alignment uint32) string {
	base := armReg((raw >> 16) & 0xF)
	if alignment != 0 {
		base += fmt.Sprintf(":0x%x", alignment)
	}
	switch index := raw & 0xF; index {
	case armProgramCounter:
		return "[" + base + "]"
	case armVFPStackPointerRn:
		return "[" + base + "]!"
	default:
		return "[" + base + "], " + armReg(index)
	}
}

// armSIMDLongShape is the pattern of operand widths in an operation whose
// result is not the width of its sources.
type armSIMDLongShape byte

const (
	armLongWiden      armSIMDLongShape = iota // a wide result from two narrow sources
	armLongAccumulate                         // a wide result accumulated onto a wide source
	armLongNarrow                             // a narrow result from two wide sources
)

// armSIMDLongOp describes one entry of the three-registers-of-different-lengths
// table.
type armSIMDLongOp struct {
	name  string
	kind  armSIMDType
	shape armSIMDLongShape
	sizes uint8
}

// The element sizes the operations that change width can name; none of them
// reaches 64 bits, and the doubling multiplies start at 16.
const (
	armSizeNarrow    = 0b0111
	armSizeDoubling  = 0b0110
	armSizePolyBytes = 0b0001
)

// armSIMDLongs is the three-registers-of-different-lengths table, indexed by the
// opcode field and the U bit.
var armSIMDLongs = [16][2]armSIMDLongOp{
	0b0000: {{"vaddl", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vaddl", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b0001: {{"vaddw", armSIMDSigned, armLongAccumulate, armSizeNarrow}, {"vaddw", armSIMDSigned, armLongAccumulate, armSizeNarrow}},
	0b0010: {{"vsubl", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vsubl", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b0011: {{"vsubw", armSIMDSigned, armLongAccumulate, armSizeNarrow}, {"vsubw", armSIMDSigned, armLongAccumulate, armSizeNarrow}},
	0b0100: {{"vaddhn", armSIMDInteger, armLongNarrow, armSizeNarrow}, {"vraddhn", armSIMDInteger, armLongNarrow, armSizeNarrow}},
	0b0101: {{"vabal", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vabal", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b0110: {{"vsubhn", armSIMDInteger, armLongNarrow, armSizeNarrow}, {"vrsubhn", armSIMDInteger, armLongNarrow, armSizeNarrow}},
	0b0111: {{"vabdl", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vabdl", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b1000: {{"vmlal", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vmlal", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b1001: {{"vqdmlal", armSIMDFixedSig, armLongWiden, armSizeDoubling}, {}},
	0b1010: {{"vmlsl", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vmlsl", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b1011: {{"vqdmlsl", armSIMDFixedSig, armLongWiden, armSizeDoubling}, {}},
	0b1100: {{"vmull", armSIMDSigned, armLongWiden, armSizeNarrow}, {"vmull", armSIMDSigned, armLongWiden, armSizeNarrow}},
	0b1101: {{"vqdmull", armSIMDFixedSig, armLongWiden, armSizeDoubling}, {}},
	0b1110: {{"vmull", armSIMDPoly, armLongWiden, armSizePolyBytes}, {}},
}

// armSIMDThreeLong decodes the Advanced SIMD operations whose result is twice
// or half the width of their sources. The narrowing ones name the width of
// their sources, so their type suffix is one size up.
func armSIMDThreeLong(raw uint32) (string, bool) {
	unsigned := raw >> 24 & 1
	size := raw >> 20 & 3

	entry := armSIMDLongs[raw>>8&0xF][unsigned]
	if entry.name == "" || entry.sizes&(1<<size) == 0 {
		return "", false
	}

	typeSize := size
	if entry.shape == armLongNarrow {
		typeSize++
	}
	name := entry.name + armSIMDSuffix(entry.kind, unsigned, typeSize)

	narrow := entry.shape == armLongNarrow
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, !narrow)
	rn, okn := armSIMDRegister((raw>>16)&0xF, (raw>>7)&1, entry.shape != armLongWiden)
	rm, okm := armSIMDRegister(raw&0xF, (raw>>5)&1, narrow)
	if !okd || !okn || !okm {
		return "", false
	}
	return fmt.Sprintf("%s %s, %s, %s", name, rd, rn, rm), true
}

// armSIMDScalarOp describes one entry of the two-registers-and-a-scalar table.
// The widening operations always name a 128-bit destination and a 64-bit
// source; for the rest the U bit chooses the width of both.
type armSIMDScalarOp struct {
	name   string
	kind   armSIMDType
	widen  bool
	sizes  uint8
	signed bool // the U bit picks the signedness rather than the operand width
}

// armSIMDScalars is the two-registers-and-a-scalar table, indexed by the opcode
// field. Entries the U bit forbids carry no name in the second column.
var armSIMDScalars = [16][2]armSIMDScalarOp{
	0b0000: {{"vmla", armSIMDInteger, false, armSizeDoubling, false}, {"vmla", armSIMDInteger, false, armSizeDoubling, false}},
	0b0001: {{"vmla", armSIMDFloat, false, 0b0100, false}, {"vmla", armSIMDFloat, false, 0b0100, false}},
	0b0010: {{"vmlal", armSIMDSigned, true, armSizeDoubling, true}, {"vmlal", armSIMDSigned, true, armSizeDoubling, true}},
	0b0011: {{"vqdmlal", armSIMDFixedSig, true, armSizeDoubling, true}, {}},
	0b0100: {{"vmls", armSIMDInteger, false, armSizeDoubling, false}, {"vmls", armSIMDInteger, false, armSizeDoubling, false}},
	0b0101: {{"vmls", armSIMDFloat, false, 0b0100, false}, {"vmls", armSIMDFloat, false, 0b0100, false}},
	0b0110: {{"vmlsl", armSIMDSigned, true, armSizeDoubling, true}, {"vmlsl", armSIMDSigned, true, armSizeDoubling, true}},
	0b0111: {{"vqdmlsl", armSIMDFixedSig, true, armSizeDoubling, true}, {}},
	0b1000: {{"vmul", armSIMDInteger, false, armSizeDoubling, false}, {"vmul", armSIMDInteger, false, armSizeDoubling, false}},
	0b1001: {{"vmul", armSIMDFloat, false, 0b0100, false}, {"vmul", armSIMDFloat, false, 0b0100, false}},
	0b1010: {{"vmull", armSIMDSigned, true, armSizeDoubling, true}, {"vmull", armSIMDSigned, true, armSizeDoubling, true}},
	0b1011: {{"vqdmull", armSIMDFixedSig, true, armSizeDoubling, true}, {}},
	0b1100: {{"vqdmulh", armSIMDFixedSig, false, armSizeDoubling, false}, {"vqdmulh", armSIMDFixedSig, false, armSizeDoubling, false}},
	0b1101: {{"vqrdmulh", armSIMDFixedSig, false, armSizeDoubling, false}, {"vqrdmulh", armSIMDFixedSig, false, armSizeDoubling, false}},
}

// armSIMDScalar decodes the Advanced SIMD operations that multiply a vector by
// one lane of another register.
func armSIMDScalar(raw uint32) (string, bool) {
	unsigned := raw >> 24 & 1
	size := raw >> 20 & 3

	entry := armSIMDScalars[raw>>8&0xF][unsigned]
	if entry.name == "" || entry.sizes&(1<<size) == 0 {
		return "", false
	}

	// Only the widening forms fix their operand widths; elsewhere the U bit
	// that would carry a signedness picks between the 64 and 128-bit forms.
	quad := !entry.widen && unsigned != 0
	name := entry.name + armSIMDSuffix(entry.kind, unsigned, size)
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, entry.widen || quad)
	rn, okn := armSIMDRegister((raw>>16)&0xF, (raw>>7)&1, quad)
	if !okd || !okn {
		return "", false
	}
	return fmt.Sprintf("%s %s, %s, %s", name, rd, rn, armSIMDScalarOperand(raw, size)), true
}

// armSIMDScalarOperand names the lane a scalar multiply reads. The narrower
// elements leave room for only eight registers, so the field they free up
// widens the lane index instead.
func armSIMDScalarOperand(raw uint32, size uint32) string {
	field, top := raw&0xF, raw>>5&1
	if size == 1 {
		return fmt.Sprintf("d%d[%d]", field&7, top<<1|field>>3)
	}
	return fmt.Sprintf("d%d[%d]", field, top)
}

// armSIMDShiftForm is the shape of a two-registers-and-a-shift operation: how
// wide its operands are and which way its amount field counts.
type armSIMDShiftForm byte

const (
	armShiftRight armSIMDShiftForm = iota
	armShiftLeft
	armShiftNarrow  // a wide source read down into a narrow destination
	armShiftWiden   // and the reverse
	armShiftConvert // between floating point and a fixed-point integer
)

// armSIMDShiftOp describes one entry of the shift table. The narrowing
// operations come in pairs, the rounding one chosen by the Q bit, which they
// use for that rather than for an operand width.
type armSIMDShiftOp struct {
	name    string
	rounded string
	kind    armSIMDType
	form    armSIMDShiftForm
}

// armSIMDShifts is the two-registers-and-a-shift-amount table, indexed by the U
// bit and the opcode field.
var armSIMDShifts = [2][16]armSIMDShiftOp{
	{
		0b0000: {"vshr", "", armSIMDSigned, armShiftRight},
		0b0001: {"vsra", "", armSIMDSigned, armShiftRight},
		0b0010: {"vrshr", "", armSIMDSigned, armShiftRight},
		0b0011: {"vrsra", "", armSIMDSigned, armShiftRight},
		0b0101: {"vshl", "", armSIMDInteger, armShiftLeft},
		0b0111: {"vqshl", "", armSIMDSigned, armShiftLeft},
		0b1000: {"vshrn", "vrshrn", armSIMDInteger, armShiftNarrow},
		0b1001: {"vqshrn", "vqrshrn", armSIMDSigned, armShiftNarrow},
		0b1010: {"vshll", "", armSIMDSigned, armShiftWiden},
		0b1110: {"vcvt.f32.s32", "", armSIMDNone, armShiftConvert},
		0b1111: {"vcvt.s32.f32", "", armSIMDNone, armShiftConvert},
	},
	{
		0b0000: {"vshr", "", armSIMDSigned, armShiftRight},
		0b0001: {"vsra", "", armSIMDSigned, armShiftRight},
		0b0010: {"vrshr", "", armSIMDSigned, armShiftRight},
		0b0011: {"vrsra", "", armSIMDSigned, armShiftRight},
		0b0100: {"vsri", "", armSIMDBare, armShiftRight},
		0b0101: {"vsli", "", armSIMDBare, armShiftLeft},
		0b0110: {"vqshlu", "", armSIMDFixedSig, armShiftLeft},
		0b0111: {"vqshl", "", armSIMDSigned, armShiftLeft},
		0b1000: {"vqshrun", "vqrshrun", armSIMDFixedSig, armShiftNarrow},
		0b1001: {"vqshrn", "vqrshrn", armSIMDSigned, armShiftNarrow},
		0b1010: {"vshll", "", armSIMDSigned, armShiftWiden},
		0b1110: {"vcvt.f32.u32", "", armSIMDNone, armShiftConvert},
		0b1111: {"vcvt.u32.f32", "", armSIMDNone, armShiftConvert},
	},
}

// The element size a conversion between floating point and fixed point works
// on, and the widest one an operation that changes width can name.
const (
	armSIMDConvertSize = 2
	armSIMDWidestSize  = 3
)

// armSIMDTwoShift decodes the Advanced SIMD operations that shift a register by
// a constant. The amount field doubles as the element-size selector: its
// highest set bit says how wide an element is, and what is left counts the
// shift, upwards for a left shift and downwards for a right one.
func armSIMDTwoShift(raw uint32) (string, bool) {
	unsigned := raw >> 24 & 1
	amount := raw>>7&1<<6 | raw>>16&0x3F
	quad := raw&(1<<6) != 0

	entry := armSIMDShifts[unsigned][raw>>8&0xF]
	size, ok := armSIMDShiftSize(amount)
	if entry.name == "" || !ok {
		return "", false
	}
	elements := uint32(armVFPLaneByte) << size
	name := entry.name + armSIMDSuffix(entry.kind, unsigned, size)

	switch entry.form {
	case armShiftRight:
		return armSIMDShiftPair(raw, name, quad, quad, int64(2*elements-amount))
	case armShiftLeft:
		return armSIMDShiftPair(raw, name, quad, quad, int64(amount-elements))
	case armShiftConvert:
		if size != armSIMDConvertSize {
			return "", false
		}
		return armSIMDShiftPair(raw, entry.name, quad, quad, int64(2*elements-amount))
	case armShiftNarrow:
		if size == armSIMDWidestSize {
			return "", false
		}
		wide := entry.name
		if quad {
			wide = entry.rounded
		}
		wide += armSIMDSuffix(entry.kind, unsigned, size+1)
		return armSIMDShiftPair(raw, wide, false, true, int64(2*elements-amount))
	default: // armShiftWiden
		if size == armSIMDWidestSize || quad {
			return "", false
		}
		if amount == elements {
			return armSIMDShiftPair(raw, "vmovl"+armSIMDSuffix(entry.kind, unsigned, size), true, false, -1)
		}
		return armSIMDShiftPair(raw, name, true, false, int64(amount-elements))
	}
}

// armSIMDShiftSize reads the element size out of a shift amount: the highest
// set bit above the low three names it, and none set at all means the word is
// not a shift.
func armSIMDShiftSize(amount uint32) (uint32, bool) {
	for size := uint32(armSIMDWidestSize); ; size-- {
		if amount>>3&(1<<size) != 0 {
			return size, true
		}
		if size == 0 {
			return 0, false
		}
	}
}

// armSIMDShiftPair renders a shift's two register operands and its amount. A
// negative amount is one the widening move leaves out.
func armSIMDShiftPair(raw uint32, mnemonic string, wideDest, wideSource bool, amount int64) (string, bool) {
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, wideDest)
	rm, okm := armSIMDRegister(raw&0xF, (raw>>5)&1, wideSource)
	if !okd || !okm {
		return "", false
	}
	if amount < 0 {
		return fmt.Sprintf("%s %s, %s", mnemonic, rd, rm), true
	}
	return fmt.Sprintf("%s %s, %s, %s", mnemonic, rd, rm, armImmediate32(amount)), true
}

// The four ways a modified immediate can be spread across an element, and the
// two that finish it off with a run of one bits.
const (
	armSIMDImmediateWord     = 0b0000 // cmode 0xxx: a byte anywhere in a word
	armSIMDImmediateHalfword = 0b1000 // cmode 10xx: a byte in either half
	armSIMDImmediateOnesLow  = 0b1100
	armSIMDImmediateOnesHigh = 0b1101
	armSIMDImmediateByte     = 0b1110
	armSIMDImmediateFloat    = 0b1111
)

// armSIMDModifiedImmediate decodes the Advanced SIMD operations that take one
// register and a constant built out of an eight-bit field. The cmode field says
// how the byte is spread across an element, and bit 5 turns a move into its
// inverted or masking counterpart.
func armSIMDModifiedImmediate(raw uint32) (string, bool) {
	cmode := raw >> 8 & 0xF
	invert := raw&(1<<5) != 0
	imm8 := raw>>24&1<<7 | raw>>16&7<<4 | raw&0xF
	quad := raw&(1<<6) != 0

	rd, ok := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, quad)
	if !ok {
		return "", false
	}

	name, width, value := armSIMDImmediateValue(cmode, invert, imm8)
	switch {
	case name == "":
		return "", false
	case width == armSIMDImmediateIsFloat:
		constant := strconv.FormatFloat(armVFPExpandImm(imm8, false), 'e', 6, 64)
		return fmt.Sprintf("%s.f32 %s, #%s", name, rd, constant), true
	}
	return fmt.Sprintf("%s.i%d %s, %s", name, width, rd, armSIMDImmediate(value)), true
}

// armSIMDImmediateIsFloat is the element width that stands for the one modified
// immediate that spells out a floating-point constant rather than an integer.
const armSIMDImmediateIsFloat = 0

// armSIMDImmediateValue works out what a modified immediate stands for, giving
// the mnemonic, the element width in bits, and the constant itself.
func armSIMDImmediateValue(cmode uint32, invert bool, imm8 uint32) (name string, width int, value uint64) {
	masking := cmode&1 != 0 && cmode < armSIMDImmediateOnesLow
	switch {
	case masking:
		name = "vorr"
		if invert {
			name = "vbic"
		}
	case invert:
		name = "vmvn"
	default:
		name = "vmov"
	}

	switch {
	case cmode&0b1000 == armSIMDImmediateWord:
		return name, armVFPLaneWord, uint64(imm8) << (8 * (cmode >> 1 & 3))
	case cmode&0b1100 == armSIMDImmediateHalfword:
		return name, armVFPLaneHalfword, uint64(imm8) << (8 * (cmode >> 1 & 1))
	case cmode == armSIMDImmediateOnesLow:
		return name, armVFPLaneWord, uint64(imm8)<<8 | 0xFF
	case cmode == armSIMDImmediateOnesHigh:
		return name, armVFPLaneWord, uint64(imm8)<<16 | 0xFFFF
	case cmode == armSIMDImmediateByte && !invert:
		return name, armVFPLaneByte, uint64(imm8)
	case cmode == armSIMDImmediateByte:
		return "vmov", 64, armSIMDExpandBytes(imm8)
	case cmode == armSIMDImmediateFloat && !invert:
		return "vmov", armSIMDImmediateIsFloat, 0
	}
	return "", 0, 0
}

// armSIMDExpandBytes turns each bit of an eight-bit immediate into a whole byte
// of the 64-bit constant it stands for.
func armSIMDExpandBytes(imm8 uint32) uint64 {
	var value uint64
	for bit := range 8 {
		if imm8&(1<<bit) != 0 {
			value |= 0xFF << (8 * bit)
		}
	}
	return value
}

// armSIMDImmediate renders a vector constant, which follows the same rule as
// any other ARM immediate but can fill a whole 64-bit element.
func armSIMDImmediate(value uint64) string {
	if value <= 9 {
		return "#" + strconv.FormatUint(value, 10)
	}
	return "#0x" + strconv.FormatUint(value, 16)
}

// armSIMDThreeSame decodes the Advanced SIMD operations that take three
// registers of the same length.
func armSIMDThreeSame(raw uint32) (string, bool) {
	unsigned := raw >> 24 & 1
	size := raw >> 20 & 3
	opcode := raw >> 8 & 0xF
	bit4 := raw >> 4 & 1
	quad := raw&(1<<6) != 0

	if opcode == armSIMDBitwiseOpcode && bit4 == 1 {
		return armSIMDRegisters(raw, armSIMDBitwiseOps[unsigned][size], quad, false)
	}

	var entry armSIMDOp
	if opcode >= armSIMDFloatOpcode {
		if size&1 != 0 {
			return "", false
		}
		entry = armSIMDFloatSame[opcode-armSIMDFloatOpcode][bit4][unsigned][size>>1]
	} else {
		entry = armSIMDIntegerSame[opcode][bit4][unsigned]
		if entry.sizes&(1<<size) == 0 {
			return "", false
		}
	}
	if entry.name == "" || (quad && entry.solo) {
		return "", false
	}
	return armSIMDRegisters(raw, entry.name+armSIMDSuffix(entry.kind, unsigned, size), quad, entry.swap)
}

// armSIMDSuffix spells the data type of an Advanced SIMD operation.
func armSIMDSuffix(kind armSIMDType, unsigned, size uint32) string {
	elements := armVFPLaneByte << size
	switch kind {
	case armSIMDNone:
		return ""
	case armSIMDFloat:
		return ".f32"
	case armSIMDSigned:
		if unsigned != 0 {
			return fmt.Sprintf(".u%d", elements)
		}
		return fmt.Sprintf(".s%d", elements)
	case armSIMDFixedSig:
		return fmt.Sprintf(".s%d", elements)
	case armSIMDFixedUns:
		return fmt.Sprintf(".u%d", elements)
	case armSIMDBare:
		return fmt.Sprintf(".%d", elements)
	default: // armSIMDInteger and armSIMDPoly carry their letter directly
		return fmt.Sprintf(".%c%d", kind, elements)
	}
}

// armSIMDRegisters appends the three register operands, which are numbered by a
// four-bit field and one bit held elsewhere. The 128-bit form pairs them up, so
// each field has to be even.
func armSIMDRegisters(raw uint32, mnemonic string, quad, swap bool) (string, bool) {
	rd, okd := armSIMDRegister((raw>>12)&0xF, (raw>>22)&1, quad)
	rn, okn := armSIMDRegister((raw>>16)&0xF, (raw>>7)&1, quad)
	rm, okm := armSIMDRegister(raw&0xF, (raw>>5)&1, quad)
	if !okd || !okn || !okm {
		return "", false
	}
	if swap {
		rn, rm = rm, rn
	}
	return fmt.Sprintf("%s %s, %s, %s", mnemonic, rd, rn, rm), true
}

// armSIMDRegister names one vector register.
func armSIMDRegister(field, extra uint32, quad bool) (string, bool) {
	number := extra<<4 | field
	if !quad {
		return fmt.Sprintf("d%d", number), true
	}
	if number%2 != 0 {
		return "", false
	}
	return fmt.Sprintf("q%d", number/2), true
}

// The interrupt-mask changes CPS can ask for. Leaving the field clear only
// reads the processor mode; the second value is unallocated.
const (
	armChangeNone     = 0b00
	armChangeEnable   = 0b10
	armChangeTopMask  = 8 // the asynchronous abort mask, with the other two below
	armChangeMaskBits = 3

	armChangeModeMask   = 0x1F   // the processor mode, at the bottom of the word
	armChangeUnusedMask = 0xFE00 // the bits between the mode flag and the masks
)

// armChangeStateNames spell the enable and disable forms.
var armChangeStateNames = [4]string{armChangeEnable: "cpsie", 0b11: "cpsid"}

// armChangeState32 decodes CPS, which changes the interrupt masks, the
// processor mode, or both. Where it names no mode Capstone still reads the mode
// field, and rejects the word unless it agrees with the mask bits.
func armChangeState32(raw uint32) (string, bool) {
	// Of the bits the manual leaves clear, Capstone insists on the one below
	// the mode field and the one just above the processor-mode flag.
	const reserved = 1<<16 | 1<<5
	if raw>>20 != 0xF10 || raw&reserved != 0 {
		return "", false
	}
	change := raw >> 18 & 3
	name := armChangeStateNames[change]
	if (change != armChangeNone && name == "") || !armChangeStateIsAllocated(raw) {
		return "", false
	}

	mode := armImmediate32(int64(raw & armChangeModeMask))
	if change == armChangeNone {
		return "cps " + mode, true
	}
	if raw&(1<<17) == 0 {
		return name + " " + armChangeStateFlags(raw), true
	}
	return name + " " + armChangeStateFlags(raw) + ", " + mode, true
}

// armChangeStateIsAllocated applies the checks Capstone makes on a CPS word.
// They do not follow the manual: naming a processor mode excuses everything
// else, but only while the bits above the masks are clear, and failing that the
// interrupt masks have to agree with the top bit of the mode field.
func armChangeStateIsAllocated(raw uint32) bool {
	mode := raw & armChangeModeMask
	if raw&armChangeUnusedMask == 0 && (raw&(1<<17) != 0 || mode == 0) {
		return true
	}
	agreement := raw>>7&1<<2 | raw>>6&1<<1 | mode>>4
	return agreement >= 0b011 && agreement <= 0b110
}

// armChangeStateFlags names the masks a change applies, in the order Capstone
// prints them.
func armChangeStateFlags(raw uint32) string {
	var flags strings.Builder
	for i, letter := range [armChangeMaskBits]string{"a", "i", "f"} {
		if raw&(1<<(armChangeTopMask-i)) != 0 {
			flags.WriteString(letter)
		}
	}
	if flags.Len() == 0 {
		return "none"
	}
	return flags.String()
}

// armRegisterOffset32 decodes the loads and stores whose displacement is a
// shifted register. x/arch declines several of them.
func armRegisterOffset32(raw uint32) (string, bool) {
	cond := raw >> 28
	if (raw>>25)&7 != 0b011 || raw&(1<<4) != 0 || cond == 15 {
		return "", false
	}

	name := "str"
	if raw&(1<<20) != 0 {
		name = "ldr"
	}
	if raw&(1<<22) != 0 {
		name += "b"
	}
	preIndexed := raw&(1<<24) != 0
	writeback := raw&(1<<21) != 0
	if !preIndexed && writeback {
		name += "t" // the unprivileged form, which has no pre-indexed shape
	}
	name += armCondSuffix(cond)

	index := armReg(raw & 0xF)
	if raw&(1<<23) == 0 {
		index = "-" + index
	}
	if shift, shifted := armThumb32Shift((raw>>5)&3, (raw>>7)&0x1F); shifted {
		index += ", " + shift
	}

	head := fmt.Sprintf("%s %s, ", name, armReg((raw>>12)&0xF))
	base := armReg((raw >> 16) & 0xF)
	if !preIndexed {
		return head + "[" + base + "], " + index, true
	}
	out := head + "[" + base + ", " + index + "]"
	if writeback {
		out += "!"
	}
	return out, true
}

// The transfers to and from a status register, and the fields their mask
// selects. Writing the flags or the greater-than field of the current register
// alone is named for the application rather than the whole register.
const (
	armStatusMask     = 0x0FB00000
	armStatusReadOp   = 0x01000000 // MRS, whose remaining fields are all fixed
	armStatusWriteOp  = 0x03200000 // MSR with an immediate
	armStatusMoveOp   = 0x01200000 // MSR with a register source
	armStatusSavedBit = 1 << 22
)

// armStatusFields name the mask a status write applies, from bit 19 down.
var armStatusFields = [4]string{"f", "s", "x", "c"}

// armStatusAliases name the three masks of the current register that Capstone
// spells as the application status register instead.
var armStatusAliases = map[uint32]string{
	0b0100: "apsr_g", 0b1000: "apsr_nzcvq", 0b1100: "apsr_nzcvqg",
}

// armStatusRegister32 renders the reads of a status register and the writes
// that take an immediate. x/arch decodes only the read of the current one.
func armStatusRegister32(raw uint32) (string, bool) {
	cond := raw >> 28
	if cond == 15 {
		return "", false
	}
	saved := raw&armStatusSavedBit != 0
	switch raw & armStatusMask {
	case armStatusReadOp:
		// The nibble above the destination is fixed in the manual, but Capstone
		// reads past it; the bits below the marker it does check.
		if raw&0xF0 != 0 {
			return "", false
		}
		if raw&armBankedMarker != 0 {
			return armBankedRead32(raw, cond)
		}
		name := "spsr"
		if !saved {
			// The read of the visible register keeps the lowest bit of the
			// field the banked forms use to select a register.
			if raw&armStatusFieldBit == 0 {
				return "", false
			}
			name = "apsr"
		}
		return fmt.Sprintf("mrs%s %s, %s", armCondSuffix(cond), armReg((raw>>12)&0xF), name), true
	case armStatusMoveOp:
		return armStatusMove32(raw, cond, saved)
	case armStatusWriteOp:
		if (raw>>12)&0xF != 0xF {
			return "", false
		}
		return armStatusWrite32(raw, cond, saved)
	}
	return "", false
}

// The bit that turns a status read into one of a banked register, and the
// answer Capstone gives when the remaining fields name none.
const (
	armBankedMarker   = 1 << 9
	armBankedModeBit  = 1 << 8
	armStatusFieldBit = 1 << 16
)

// An unallocated banked selector leaves Capstone reading the wrong operands: it
// names the register the condition field selects, and prints a condition of its
// own, which changes at the last of them.
const (
	armBankedStrayCondition = "lo"
	armBankedLastCondition  = 14
	armBankedLastSpelling   = "eq"
)

// armBankedRegisters name the registers a processor mode keeps for itself,
// keyed by the three fields that select one. A key with no name is a
// combination the architecture does not allocate.
var armBankedRegisters = map[uint32]string{
	0x00: "r8_usr",
	0x01: "r9_usr",
	0x02: "r10_usr",
	0x03: "r11_usr",
	0x04: "r12_usr",
	0x05: "sp_usr",
	0x06: "lr_usr",
	0x08: "r8_fiq",
	0x09: "r9_fiq",
	0x0a: "r10_fiq",
	0x0b: "r11_fiq",
	0x0c: "r12_fiq",
	0x0d: "sp_fiq",
	0x0e: "lr_fiq",
	0x10: "lr_irq",
	0x11: "sp_irq",
	0x12: "lr_svc",
	0x13: "sp_svc",
	0x14: "lr_abt",
	0x15: "sp_abt",
	0x16: "lr_und",
	0x17: "sp_und",
	0x1c: "lr_mon",
	0x1d: "sp_mon",
	0x1e: "elr_hyp",
	0x1f: "sp_hyp",
	0x2e: "SPSR_fiq",
	0x30: "SPSR_irq",
	0x32: "SPSR_svc",
	0x34: "SPSR_abt",
	0x36: "SPSR_und",
	0x3c: "SPSR_mon",
	0x3e: "SPSR_hyp",
}

// armBankedRead32 renders a read of a register a processor mode banks for
// itself. Where the fields name no such register Capstone prints the one the
// condition field selects instead, under a condition of its own.
func armBankedRead32(raw, cond uint32) (string, bool) {
	if raw&0xC00 != 0 || raw&0xFF != 0 {
		return "", false // the bits either side of the marker are fixed
	}
	key := (raw>>22)&1<<5 | (raw>>8)&1<<4 | (raw>>16)&0xF
	destination := armReg((raw >> 12) & 0xF)
	if name, ok := armBankedRegisters[key]; ok {
		return fmt.Sprintf("mrs%s %s, %s", armCondSuffix(cond), destination, name), true
	}

	suffix := armBankedStrayCondition
	if cond == armBankedLastCondition {
		suffix = armBankedLastSpelling
	}
	return fmt.Sprintf("mrs%s %s, %s", suffix, destination, armBankedRegisters[cond]), true
}

// armStatusMove32 renders MSR with a register source, which reaches either the
// fields of a status register or one of the registers a mode keeps for itself.
func armStatusMove32(raw, cond uint32, saved bool) (string, bool) {
	if (raw>>12)&0xF != 0xF || raw&0xC00 != 0 || raw&0xF0 != 0 {
		return "", false
	}
	if raw&armBankedMarker != 0 {
		return armBankedWrite32(raw, cond)
	}
	if raw&armBankedModeBit != 0 {
		return "", false // the bit below the marker is fixed without it
	}
	name := armStatusRegisterName(raw, saved)
	if name == "" {
		return "", false
	}
	return fmt.Sprintf("msr%s %s, %s", armCondSuffix(cond), name, armReg(raw&0xF)), true
}

// armBankedStrayWrite are the registers Capstone names for a write whose
// selector names none, indexed by the source field it reads in their place.
var armBankedStrayWrite = [16]string{
	"r10_usr", "r11_usr", "r12_usr", "sp_usr", "lr_usr", "", "r8_fiq", "r9_fiq",
	"r10_fiq", "r11_fiq", "r12_fiq", "sp_fiq", "lr_fiq", "r12_fiq", "r10_fiq", "r11_fiq",
}

// armBankedWrite32 renders MSR to a banked register.
func armBankedWrite32(raw, cond uint32) (string, bool) {
	key := (raw>>22)&1<<5 | (raw>>8)&1<<4 | (raw>>16)&0xF
	if name, ok := armBankedRegisters[key]; ok {
		return fmt.Sprintf("msr%s %s, %s", armCondSuffix(cond), name, armReg(raw&0xF)), true
	}

	suffix := armBankedStrayCondition
	if cond == armBankedLastCondition {
		suffix = armBankedLastSpelling
	}
	return fmt.Sprintf("msr%s %s, %s", suffix, armBankedStrayWrite[raw&0xF],
		armImmediate32(int64(cond))), true
}

// armStatusWrite32 renders MSR with an immediate, whose value carries its own
// rotation the same way any other data-processing immediate does.
func armStatusWrite32(raw, cond uint32, saved bool) (string, bool) {
	name := armStatusRegisterName(raw, saved)
	if name == "" {
		return "", false
	}
	// A rotated value is printed as a pair of plain decimals, the same way any
	// other immediate that carries its own rotation is; on its own it follows
	// the ordinary rule.
	head := fmt.Sprintf("msr%s %s, ", armCondSuffix(cond), name)
	value, rotation := raw&0xFF, (raw>>8)&0xF
	if folded, ok := armFoldRotatedImmediate(value, rotation); ok {
		return head + armImmediate32(int64(folded)), true
	}
	return head + fmt.Sprintf("#%d, #%d", value, rotation*2), true
}

// armFoldRotatedImmediate turns a value and its rotation into the constant they
// stand for, reporting whether that pair is the shortest way to write it. Where
// it is not, Capstone prints the two halves instead of the constant.
func armFoldRotatedImmediate(value, rotation uint32) (uint32, bool) {
	folded := bits.RotateLeft32(value, -int(rotation)*2)
	for candidate := range uint32(16) {
		if rotated := bits.RotateLeft32(folded, int(candidate)*2); rotated <= 0xFF {
			return folded, candidate == rotation && rotated == value
		}
	}
	return folded, false
}

// armStatusRegisterName spells the register and the fields a write reaches.
func armStatusRegisterName(raw uint32, saved bool) string {
	fields := (raw >> 16) & 0xF
	if !saved {
		if alias, ok := armStatusAliases[fields]; ok {
			return alias
		}
		if fields == 0 {
			return "" // the current register must name a field
		}
	}
	name := "cpsr"
	if saved {
		name = "spsr"
	}
	var letters strings.Builder
	for i, letter := range armStatusFields {
		if fields&(1<<(3-uint32(i))) != 0 {
			letters.WriteString(letter)
		}
	}
	if letters.Len() == 0 {
		return name
	}
	return name + "_" + letters.String()
}

// armExceptionAddressing names the four addressing orders the exception
// transfers use, indexed by their pre-index and increment bits. Unlike the
// ordinary block transfer, increment-after is spelled out.
var armExceptionAddressing = [4]string{"da", "ia", "db", "ib"}

// armExceptionSubModes are the numbers LLVM gives the same four orders, which
// Capstone prints in place of the base register where a writeback form does not
// match the encoding exactly.
var armExceptionSubModes = [4]int64{3, 1, 4, 2}

// The bits an exception transfer fixes, below the base register.
const (
	armReturnFixedBits = 0x0A00 // RFE spells out its transfer list
	armSaveFixedBits   = 0x0500 // and SRS its own, leaving the mode below
	armSaveModeMask    = 0x1F   // the processor mode a stack save names
	armStackPointerReg = 13     // the only base register a stack save takes
	armSaveFixedMask   = 0xFFFF &^ 0x1F
)

// armExceptionTransfer32 renders RFE and SRS, the unconditional half of the
// block transfer encoding.
func armExceptionTransfer32(raw uint32) (string, bool) {
	if (raw>>25)&7 != 0b100 || raw>>28 != 15 {
		return "", false
	}
	mode := armExceptionAddressing[(raw>>23)&3]
	writeback := raw&(1<<21) != 0
	user := raw&(1<<22) != 0

	if raw&(1<<20) != 0 {
		if user {
			return "", false
		}
		return armExceptionReturn32(raw, mode, writeback)
	}
	if !user || (raw>>16)&0xF != armStackPointerReg || raw&armSaveFixedMask != armSaveFixedBits {
		return "", false
	}
	base := "sp"
	if writeback {
		base += "!"
	}
	return fmt.Sprintf("srs%s %s, %s", mode, base, armImmediate32(int64(raw&armSaveModeMask))), true
}

// armExceptionReturn32 renders RFE. Where writeback is set Capstone accepts an
// encoding whose fixed bits are wrong, and prints the numbered addressing mode
// instead of the base register.
func armExceptionReturn32(raw uint32, mode string, writeback bool) (string, bool) {
	if raw&0xFFFF != armReturnFixedBits {
		if !writeback {
			return "", false
		}
		number := armExceptionSubModes[(raw>>23)&3]
		return fmt.Sprintf("rfe%s %s!", mode, armImmediate32(number)), true
	}
	out := "rfe" + mode + " " + armReg((raw>>16)&0xF)
	if writeback {
		out += "!"
	}
	return out, true
}

// armParallelGroups name the six parallel arithmetic families by the field that
// selects them, and armParallelOps the operation within a family.
var (
	armParallelGroups = map[uint32]string{
		0b001: "s", 0b010: "q", 0b011: "sh",
		0b101: "u", 0b110: "uq", 0b111: "uh",
	}
	armParallelOps = [8]string{"add16", "asx", "sax", "sub16", "add8", "", "", "sub8"}
)

// armParallelArithmetic32 renders the parallel addition and subtraction group,
// which operates on the halves or quarters of a word at once.
func armParallelArithmetic32(raw uint32) (string, bool) {
	// Bit 4 marks the group; the nibble above the operand registers is
	// nominally all ones, but Capstone does not enforce it.
	if (raw>>23)&0x1F != 0b01100 || raw&(1<<4) == 0 {
		return "", false
	}
	cond := raw >> 28
	if cond == 15 {
		return "", false
	}

	prefix, ok := armParallelGroups[(raw>>20)&7]
	if !ok {
		return "", false
	}
	operation := armParallelOps[(raw>>5)&7]
	if operation == "" {
		return "", false
	}

	return fmt.Sprintf("%s%s%s %s, %s, %s", prefix, operation, armCondSuffix(cond),
		armReg((raw>>12)&0xF), armReg((raw>>16)&0xF), armReg(raw&0xF)), true
}

// armPairedTransfer32 adds the second register of a paired load or store.
// Capstone names both halves of the pair; x/arch prints only the first.
func armPairedTransfer32(mnemonic string, inst armasm.Inst, operands string) string {
	if !strings.HasPrefix(mnemonic, "strd") && !strings.HasPrefix(mnemonic, "ldrd") {
		return operands
	}
	first, ok := inst.Args[0].(armasm.Reg)
	if !ok {
		return operands
	}

	// The pair is consecutive, which x/arch does not always report.
	second := armReg(uint32(first-armasm.R0) + 1)
	name, rest, found := strings.Cut(operands, ", ")
	if !found {
		return operands
	}
	return name + ", " + second + ", " + rest
}

// The access widths that name a register pair rather than a single register,
// and the one that never does.
const (
	armExtraHalfword   = 0b01
	armExtraDoubleword = 0b10
)

// armExtraTransfers name the extra load/store group by the two bits that select
// the access width and by the load bit. The doubleword forms move a register
// pair, so they sit on the opposite side of the table from the widths that move
// a single one.
var armExtraTransfers = [4][2]string{
	0b01: {"strh", "ldrh"},
	0b10: {"ldrd", "ldrsb"},
	0b11: {"strd", "ldrsh"},
}

// armExtraLoadStore32 renders the halfword and signed byte accesses, which the
// ARM encoding keeps apart from the word and unsigned byte ones.
func armExtraLoadStore32(raw uint32) (string, bool) {
	const marker = 1<<7 | 1<<4
	if (raw>>25)&7 != 0 || raw&marker != marker {
		return "", false
	}
	cond := raw >> 28
	if cond == 15 {
		return "", false
	}
	load := raw&(1<<20) != 0
	width := (raw >> 5) & 3
	name := armExtraTransfers[width][(raw>>20)&1]
	if name == "" {
		return "", false // an exclusive access, not this group
	}

	// A post-indexed access writes back on its own, so the writeback bit
	// selects the unprivileged form instead. Only the loads have one.
	// Only the loads have an unprivileged form, and the doublewords are stores
	// in this half of the table, so excluding the stores excludes them too.
	if raw&(1<<24) == 0 && raw&(1<<21) != 0 {
		if !load {
			return "", false
		}
		name += "t"
	}
	name += armCondSuffix(cond)

	rn, rt := (raw>>16)&0xF, (raw>>12)&0xF
	registers := armReg(rt)
	if !load && width != armExtraHalfword {
		if rt == armProgramCounter {
			return "", false // the pair would run past the last register
		}
		registers += ", " + armReg(rt+1) // the doublewords move a pair
	}
	sign := ""
	if raw&(1<<23) == 0 {
		sign = "-"
	}

	operand := sign + armReg(raw&0xF)
	if raw&(1<<22) != 0 { // an immediate offset, split across two nibbles
		offset := int64((raw>>4)&0xF0 | raw&0xF)
		if offset == 0 {
			operand = ""
		} else {
			operand = sign + strings.TrimPrefix(armImmediate32(offset), "#")
			operand = "#" + operand
		}
	}

	return name + " " + registers + ", " + armExtraAddress(rn, operand, raw), true
}

// armExtraAddress assembles the address of an extra load or store, which
// indexes before or after the access and may write the result back.
func armExtraAddress(rn uint32, operand string, raw uint32) string {
	base := armReg(rn)
	if raw&(1<<24) == 0 { // post-indexed
		return "[" + base + "], " + operand
	}
	if operand == "" {
		return "[" + base + "]"
	}
	out := "[" + base + ", " + operand + "]"
	if raw&(1<<21) != 0 {
		out += "!"
	}
	return out
}
