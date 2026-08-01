package ops

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(DisassembleX86{})
}

// DisassembleX86 translates x86 machine code into assembly language. Ported
// from CyberChef, which wraps Damian Recoskie's X86-64-Disassembler-JS
// (src/core/vendor/DisassembleX86-64.mjs, MIT). The port reproduces that engine
// rather than wrapping a Go disassembler so the output matches CyberChef
// exactly, down to its quirks: it predates ENDBR64, resolves RIP-relative
// operands to absolute addresses, and prints NaN where an instruction runs off
// the end of the input. Its opcode tables are vendored in x86tables/.
type DisassembleX86 struct{}

// Meta returns the operation metadata.
func (DisassembleX86) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Disassemble x86",
		Module: "Shellcode",
		Description: "Disassembly is the process of translating machine language into assembly " +
			"language.<br><br>This operation supports 64-bit, 32-bit and 16-bit code written for " +
			"Intel or AMD x86 processors. It is particularly useful for reverse engineering " +
			"shellcode.<br><br>Input should be in hexadecimal.",
		InfoURL:    "https://wikipedia.org/wiki/X86",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// x86BitModes are CyberChef's bit-mode choices; the engine numbers them 16=0,
// 32=1, 64=2, which is the reverse of the order shown.
var x86BitModes = []string{"64", "32", "16"}

// x86CompatibilityModes are CyberChef's processor choices, in the engine's own
// numbering.
var x86CompatibilityModes = []string{
	"Full x86 architecture",
	"Knights Corner",
	"Larrabee",
	"Cyrix",
	"Geode",
	"Centaur",
	"X86/486",
}

// x86DefaultCodeSegment is CyberChef's default Code Segment value.
const x86DefaultCodeSegment = 16

// Args returns the argument definitions.
func (DisassembleX86) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Bit mode", Type: core.ArgOption, Value: x86BitModes},
		{Name: "Compatibility", Type: core.ArgOption, Value: x86CompatibilityModes},
		{Name: "Code Segment (CS)", Type: core.ArgNumber, Integer: true, Value: float64(x86DefaultCodeSegment)},
		{Name: "Offset (IP)", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		{Name: "Show instruction hex", Type: core.ArgBoolean, Value: true},
		{Name: "Show instruction position", Type: core.ArgBoolean, Value: true},
	}
}

// Bit-mode values the engine uses internally.
const (
	x86Mode16 = 0
	x86Mode32 = 1
	x86Mode64 = 2
)

// Run disassembles the hexadecimal input.
func (DisassembleX86) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var bitMode int
	switch args[0].(string) {
	case "64":
		bitMode = x86Mode64
	case "32":
		bitMode = x86Mode32
	case "16":
		bitMode = x86Mode16
	default:
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errX86InvalidMode
	}

	compat := indexOfString(x86CompatibilityModes, args[1].(string))
	codeSegment := args[2].(float64)
	offset := args[3].(float64)

	d := newX86Decoder(bitMode, compat)
	d.showHex = args[4].(bool)
	d.showPos = args[5].(bool)
	d.setBasePosition(jsNumberToString(codeSegment) + ":" + jsNumberToString(offset))
	d.loadBinCode(x86StripWhitespace(in.String()))

	return core.NewDish([]byte(d.disassemble()), core.TypeString), nil
}

// indexOfString returns the position of v in list, or 0 when it is absent.
func indexOfString(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return 0
}

// x86StripWhitespace mirrors CyberChef's input.replace(/\s/g, "").
func x86StripWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f', 0x00a0, 0xfeff:
			return -1
		}
		if r > 0x2000 && (r <= 0x200a || r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000) {
			return -1
		}
		return r
	}, s)
}

/*
   ---------------------------------------------------------------------------
   JavaScript number semantics.

   The engine leans on them heavily: bitwise operators truncate to signed
   32-bit, shift counts wrap at 32, and reading past the end of the loaded
   bytes yields undefined, which turns arithmetic into NaN and prints as "NaN".
   Values are carried as float64 so that NaN propagates the same way.
   ---------------------------------------------------------------------------
*/

func jsAnd(a, b float64) float64 { return float64(jsInt32(a) & jsInt32(b)) }
func jsOr(a, b float64) float64  { return float64(jsInt32(a) | jsInt32(b)) }
func jsXor(a, b float64) float64 { return float64(jsInt32(a) ^ jsInt32(b)) }

func jsShl(a, b float64) float64 {
	return float64(jsInt32(a) << (uint32(jsInt32(b)) & 31)) // #nosec G115 -- JavaScript masks a shift count to five bits
}

func jsShr(a, b float64) float64 {
	return float64(jsInt32(a) >> (uint32(jsInt32(b)) & 31)) // #nosec G115 -- JavaScript masks a shift count to five bits
}

// jsBool converts a condition to the 0/1 a JavaScript boolean becomes when it
// meets an arithmetic or bitwise operator.
func jsBool(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// jsHex16 is Number.prototype.toString(16): "NaN" stays "NaN", and negative
// values keep a leading minus rather than wrapping.
func jsHex16(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	n := int64(math.Trunc(v))
	if n < 0 {
		return "-" + strconv.FormatInt(-n, 16)
	}
	return strconv.FormatInt(n, 16)
}

// jsPadLeft repeats JavaScript's `for(; s.length < n; s = pad + s)` loops.
func jsPadLeft(s, pad string, n int) string {
	for len(s) < n {
		s = pad + s
	}
	return s
}

// jsSliceFrom is String.prototype.slice with a single, possibly negative,
// start index.
func jsSliceFrom(s string, start int) string {
	if start < 0 {
		start += len(s)
		if start < 0 {
			start = 0
		}
	}
	if start > len(s) {
		return ""
	}
	return s[start:]
}

// jsParseHex is parseInt(s, 16): it reads the leading hex digits and yields NaN
// when there are none. A 0x prefix is allowed and skipped, as it is for that
// radix.
func jsParseHex(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\v\f")
	neg := false
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		neg = s[0] == '-'
		s = s[1:]
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	end := 0
	for end < len(s) && isHexDigit(s[end]) {
		end++
	}
	if end == 0 {
		return math.NaN()
	}
	v, err := strconv.ParseUint(s[:end], 16, 64)
	if err != nil {
		// Longer than 64 bits; accumulate in float64 the way JavaScript does.
		var f float64
		for i := range end {
			d, _ := strconv.ParseUint(string(s[i]), 16, 8)
			f = f*16 + float64(d)
		}
		if neg {
			return -f
		}
		return f
	}
	if neg {
		return -float64(v)
	}
	return float64(v)
}

// x86UndefinedValue is how JavaScript spells undefined once it has been
// concatenated onto a string. Table lookups that fall out of range yield it,
// so it also marks an operand slot that never received a real value.
const x86UndefinedValue = "undefined"

// at returns list[i], or JavaScript's stringified undefined when the index is
// out of range.
func x86At(list []string, i float64) string {
	n := int(jsInt32(i))
	if n < 0 || n >= len(list) {
		return x86UndefinedValue
	}
	return list[n]
}

/*
   ---------------------------------------------------------------------------
   Opcode tables.
   ---------------------------------------------------------------------------
*/

//go:embed x86tables/tables.json
var x86TableFS embed.FS

// x86Node is one entry in the Mnemonics/Operands trees. An entry is either a
// leaf string or a list that is indexed further by ModR/M bits, SIMD mode,
// vector extension or operand size.
type x86Node struct {
	str    string
	list   []x86Node
	isList bool
}

// UnmarshalJSON accepts either a string leaf or a nested list.
func (n *x86Node) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '[' {
		n.isList = true
		return json.Unmarshal(b, &n.list)
	}
	if string(b) == "null" {
		return nil
	}
	return json.Unmarshal(b, &n.str)
}

// len reports the list length, or -1 for a leaf, so the engine's
// `x instanceof Array && x.length == n` checks port directly.
func (n x86Node) length() int {
	if !n.isList {
		return -1
	}
	return len(n.list)
}

// at indexes a list node, returning an empty leaf when out of range.
func (n x86Node) at(i float64) x86Node {
	k := int(jsInt32(i))
	if !n.isList || k < 0 || k >= len(n.list) {
		return x86Node{}
	}
	return n.list[k]
}

// x86ModeDiff holds the sparse table overrides one compatibility mode applies.
type x86ModeDiff struct {
	Mnemonics map[string]x86Node `json:"mnemonics"`
	Operands  map[string]x86Node `json:"operands"`
}

// x86TableSet is the vendored table file.
type x86TableSet struct {
	Mnemonics       []x86Node              `json:"mnemonics"`
	Operands        []x86Node              `json:"operands"`
	Modes           map[string]x86ModeDiff `json:"modes"`
	M3DNow          []string               `json:"m3dnow"`
	MSynthetic      []string               `json:"msynthetic"`
	ConditionCodes  []string               `json:"conditionCodes"`
	REG             []x86Node              `json:"reg"`
	PTR             []string               `json:"ptr"`
	Scale           []string               `json:"scale"`
	RoundModes      []string               `json:"roundModes"`
	RegSwizzleModes []string               `json:"regSwizzleModes"`
	ConversionModes []string               `json:"conversionModes"`
}

// x86BaseTables parses the vendored tables once. They ship with the binary, so
// a failure here means the build is broken rather than the input is bad.
var x86BaseTables = sync.OnceValue(func() *x86TableSet {
	return mustLoadX86Tables(x86TableFS)
})

// mustLoadX86Tables loads the tables or panics, which is the right response to
// a broken build: without them no x86 instruction can be decoded at all.
func mustLoadX86Tables(fsys fs.FS) *x86TableSet {
	t, err := loadX86Tables(fsys)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// loadX86Tables reads and parses the vendored opcode tables.
func loadX86Tables(fsys fs.FS) (*x86TableSet, error) {
	data, err := fs.ReadFile(fsys, "x86tables/tables.json")
	if err != nil {
		return nil, fmt.Errorf("x86 opcode tables missing: %w", err)
	}
	t := &x86TableSet{}
	if err := json.Unmarshal(data, t); err != nil {
		return nil, fmt.Errorf("x86 opcode tables unreadable: %w", err)
	}
	return t, nil
}

// x86TablesFor returns the tables with a compatibility mode's overrides
// applied. Mode 0 is the tables as vendored.
func x86TablesFor(mode int) *x86TableSet {
	base := x86BaseTables()
	diff, ok := base.Modes[strconv.Itoa(mode)]
	if !ok {
		return base
	}

	// Only the two mutated tables need copying; the rest are shared.
	out := *base
	out.Mnemonics = applyX86Diff(base.Mnemonics, diff.Mnemonics)
	out.Operands = applyX86Diff(base.Operands, diff.Operands)
	return &out
}

// applyX86Diff copies table and writes the sparse overrides into it.
func applyX86Diff(table []x86Node, diff map[string]x86Node) []x86Node {
	if len(diff) == 0 {
		return table
	}
	out := make([]x86Node, len(table))
	copy(out, table)
	for key, node := range diff {
		i, err := strconv.Atoi(key)
		if err != nil || i < 0 || i >= len(out) {
			continue
		}
		out[i] = node
	}
	return out
}

/*
   ---------------------------------------------------------------------------
   Decoder state.

   The vendored engine keeps every one of these as a module-level global, so a
   second disassembly inherits whatever the first left behind. Holding them on
   a value instead keeps runs independent and the operation safe to use
   concurrently.
   ---------------------------------------------------------------------------
*/

// x86Operand is one slot of the engine's decoder array: an operand's type, its
// size settings, and the position it takes in the printed operand list.
type x86Operand struct {
	typ    float64
	bySize bool
	size   float64
	opNum  int
	active bool
}

// Slots along the decoder array, in the order the CPU decodes them.
const (
	x86SlotRegOpcode = 0
	x86SlotModRMAddr = 1
	x86SlotModRMReg  = 2
	x86SlotImm1      = 3
	x86SlotVectorReg = 6
	x86SlotImmReg    = 7
	x86SlotExplicit  = 8
	x86SlotCount     = 12
)

// x86MaxInstructionHex is the engine's 15-byte instruction limit, in hex digits.
const x86MaxInstructionHex = 30

// x86HexColumn is the width the instruction hex column is padded to.
const x86HexColumn = 32

type x86Decoder struct {
	tables *x86TableSet

	binCode []byte
	codePos int
	pos64   float64
	pos32   float64
	codeSeg float64

	bitMode        int
	showHex        bool
	showPos        bool
	instructionHex string
	instructionPos string

	opcode      float64
	instruction string
	insOperands string

	sizeAttrSelect  float64
	widthBit        float64
	farPointer      float64
	addressOverride bool
	regExtend       float64
	baseExtend      float64
	indexExtend     float64
	segOverride     string
	rexActive       float64

	simd            float64
	vect            bool
	ignoresWidthbit float64
	vsib            float64
	roundingSetting float64
	swizzle         float64
	up              float64
	float           float64
	vectS           float64
	extension       float64
	conversionMode  float64
	roundMode       float64
	vectorRegister  float64
	maskRegister    float64
	hintZeroMerg    float64

	immValue    float64
	prefixG1    string
	prefixG2    string
	xRelease    float64
	xAcquire    float64
	hleFlipG1G2 bool
	ht          float64
	bnd         float64
	invalidOp   float64

	decoder [x86SlotCount]x86Operand
}

// newX86Decoder starts a decoder in the given bit mode with one compatibility
// mode's opcode tables.
func newX86Decoder(bitMode, compatibility int) *x86Decoder {
	d := &x86Decoder{tables: x86TablesFor(compatibility), bitMode: bitMode}
	d.sizeAttrSelect = 1
	d.segOverride = "["
	return d
}

// reset clears the per-instruction state. Ported from the engine's Reset().
func (d *x86Decoder) reset() {
	d.opcode = 0
	d.sizeAttrSelect = 1
	d.instruction = ""
	d.insOperands = ""

	d.rexActive = 0
	d.regExtend = 0
	d.baseExtend = 0
	d.indexExtend = 0
	d.segOverride = "["
	d.addressOverride = false
	d.farPointer = 0

	d.extension = 0
	d.simd = 0
	d.vect = false
	d.conversionMode = 0
	d.widthBit = 0
	d.vectorRegister = 0
	d.maskRegister = 0
	d.hintZeroMerg = 0
	d.roundMode = 0

	d.ignoresWidthbit = 0
	d.vsib = 0
	d.roundingSetting = 0
	d.swizzle = 0
	d.up = 0
	d.float = 0
	d.vectS = 0

	d.immValue = 0

	d.prefixG1 = ""
	d.prefixG2 = ""
	d.xRelease = 0
	d.xAcquire = 0
	d.hleFlipG1G2 = false
	d.ht = 0
	d.bnd = 0

	d.invalidOp = 0
	d.instructionHex = ""

	for i := range d.decoder {
		d.decoder[i].active = false
	}
}

// byteAt reads a loaded byte, yielding NaN past the end the way indexing off
// the end of a JavaScript array yields undefined.
func (d *x86Decoder) byteAt(i int) float64 {
	if i < 0 || i >= len(d.binCode) {
		return math.NaN()
	}
	return float64(d.binCode[i])
}

// cur is the byte at the current position.
func (d *x86Decoder) cur() float64 { return d.byteAt(d.codePos) }

// mnemonic and operandEntry look up an opcode's table entries.
func (d *x86Decoder) mnemonic(op float64) x86Node {
	i := int(jsInt32(op))
	if i < 0 || i >= len(d.tables.Mnemonics) {
		return x86Node{}
	}
	return d.tables.Mnemonics[i]
}

func (d *x86Decoder) operandEntry(op float64) x86Node {
	i := int(jsInt32(op))
	if i < 0 || i >= len(d.tables.Operands) {
		return x86Node{}
	}
	return d.tables.Operands[i]
}

// mnemonicStr is Mnemonics[op] where the entry is known to be a leaf.
func (d *x86Decoder) mnemonicStr(op float64) string { return d.mnemonic(op).str }

// strAt indexes a list node for a string, mirroring JavaScript's undefined.
func (n x86Node) strAt(i float64) string {
	k := int(jsInt32(i))
	if !n.isList || k < 0 || k >= len(n.list) {
		return x86UndefinedValue
	}
	return n.list[k].str
}

// reg is REG[setting][value].
func (d *x86Decoder) reg(setting, value float64) string {
	k := int(jsInt32(setting))
	if k < 0 || k >= len(d.tables.REG) {
		return x86UndefinedValue
	}
	return d.tables.REG[k].strAt(value)
}

// reg8 is REG[0][rexActive][value], the high/low 8-bit register pair.
func (d *x86Decoder) reg8(rexActive, value float64) string {
	if len(d.tables.REG) == 0 {
		return x86UndefinedValue
	}
	return d.tables.REG[0].at(rexActive).strAt(value)
}

/*
   ---------------------------------------------------------------------------
   Loading and addressing.
   ---------------------------------------------------------------------------
*/

// loadBinCode converts the hex input into bytes. Ported from LoadBinCode(),
// including its four-byte rotation and the sign-bit correction that goes with
// it. Invalid hex stops the conversion and leaves whatever decoded before it,
// which is why bad input disassembles to nothing at all.
func (d *x86Decoder) loadBinCode(hexStr string) bool {
	d.binCode = nil
	d.codePos = 0

	length := len(hexStr)
	for i := 0; i < length; i += 8 {
		int32v := jsParseHex(jsSlice(hexStr, i, i+8))
		if math.IsNaN(int32v) {
			return false
		}
		if (length - i) < 8 {
			int32v = jsShl(int32v, jsShl(float64(8-length-i), 2))
		}
		sign := int32v
		int32v = jsXor(int32v, jsAnd(int32v, 0x80000000))

		for shift := 24.0; shift >= 0; shift -= 8 {
			int32v = jsOr(jsShr(int32v, 24), jsAnd(jsShl(int32v, 8), 0x7FFFFFFF))
			b := jsAnd(jsOr(jsAnd(jsShr(sign, shift), 0x80), int32v), 0xFF)
			d.binCode = append(d.binCode, byte(jsInt32(b))) // #nosec G115 -- b was masked to 0xFF on the line above
		}
	}

	// An eight-digit group always yields four bytes, so an odd tail overshoots.
	if want := length >> 1; want < len(d.binCode) {
		d.binCode = d.binCode[:want]
	}
	return true
}

// setBasePosition sets the address the loaded bytes are taken to start at.
// Ported from SetBasePosition(): both halves of "segment:offset" are read as
// hexadecimal, and a short offset keeps only its last four digits.
func (d *x86Decoder) setBasePosition(address string) {
	if seg, off, ok := strings.Cut(address, ":"); ok {
		d.codeSeg = jsParseHex(jsSliceFrom(seg, len(seg)-4))
		address = off
	}

	length := len(address)
	if length >= 9 && d.bitMode == x86Mode64 {
		d.pos64 = jsParseHex(jsSliceNeg(address, length-16, length-8))
	}
	segmented := jsAnd(jsBool(d.bitMode == x86Mode32), jsBool(d.codeSeg >= 36)) != 0
	if length >= 5 && d.bitMode >= x86Mode32 && !segmented {
		d.pos32 = jsParseHex(jsSliceFrom(address, length-8))
	} else if length >= 1 {
		d.pos32 = jsOr(jsAnd(d.pos32, 0xFFFF0000), jsParseHex(jsSliceFrom(address, length-4)))
	}

	if d.pos32 < 0 {
		d.pos32 += 0x100000000
	}
}

// getPosition renders the current instruction address. Ported from
// GetPosition(): 16-bit code, and 32-bit code with a high code segment, print
// as segment:offset; otherwise the flat instruction pointer is printed.
func (d *x86Decoder) getPosition() string {
	segmented := jsOr(
		jsBool(d.bitMode == x86Mode16),
		jsAnd(jsBool(d.bitMode == x86Mode32), jsBool(d.codeSeg >= 36)),
	) != 0

	if segmented {
		s16 := jsPadLeft(jsHex16(jsAnd(d.pos32, 0xFFFF)), "0", 4)
		seg := jsPadLeft(jsHex16(d.codeSeg), "0", 4)
		return strings.ToUpper(seg + ":" + s16)
	}

	var s64, s32 string
	if d.bitMode >= x86Mode32 {
		s32 = jsPadLeft(jsHex16(d.pos32), "0", 8)
	}
	if d.bitMode == x86Mode64 {
		s64 = jsPadLeft(jsHex16(d.pos64), "0", 8)
	}
	return strings.ToUpper(s64 + s32)
}

// gotoPosition moves the decoder to an address, reporting whether it landed
// inside the loaded bytes. Ported from GotoPosition().
func (d *x86Decoder) gotoPosition(address string) bool {
	locPos32, locPos64, locCodeSeg := d.pos32, d.pos64, d.codeSeg

	if seg, off, ok := strings.Cut(address, ":"); ok {
		locCodeSeg = jsParseHex(jsSliceFrom(seg, len(seg)-4))
		address = off
	}

	length := len(address)
	if length >= 9 && d.bitMode == x86Mode64 {
		locPos64 = jsParseHex(jsSliceNeg(address, length-16, length-8))
	}
	segmented := jsAnd(jsBool(d.bitMode == x86Mode32), jsBool(d.codeSeg >= 36)) != 0
	if length >= 5 && jsAnd(jsBool(d.bitMode >= x86Mode32), jsBool(!segmented)) != 0 {
		locPos32 = jsParseHex(jsSliceFrom(address, length-8))
	} else if length >= 1 {
		locPos32 = locPos32 - locPos32 + jsParseHex(jsSliceFrom(address, length-4))
	}

	dif32, dif64 := d.pos32-locPos32, d.pos64-locPos64

	if segmented || d.bitMode == x86Mode16 {
		dif32 += jsShl(d.codeSeg-locCodeSeg, 4)
	}

	backup := d.codePos
	d.codePos -= int(dif64*4294967296 + dif32)

	if d.codePos < 0 || d.codePos > len(d.binCode) {
		d.codePos = backup
		return false
	}

	d.codeSeg = locCodeSeg
	d.pos32 = locPos32
	d.pos64 = locPos64
	return true
}

// nextByte records the byte just consumed and advances both the buffer
// position and the instruction address. Ported from NextByte().
func (d *x86Decoder) nextByte() {
	if d.codePos >= len(d.binCode) {
		return
	}
	t := jsHex16(float64(d.binCode[d.codePos]))
	d.codePos++
	if len(t) == 1 {
		t = "0" + t
	}
	d.instructionHex += t

	d.pos32++
	if d.pos32 > 0xFFFFFFFF {
		d.pos32 = 0
		d.pos64++
		if d.pos64 > 0xFFFFFFFF {
			d.pos64 = 0
		}
	}
}

/*
   ---------------------------------------------------------------------------
   Operand size, ModR/M and immediates.
   ---------------------------------------------------------------------------
*/

// x86HighBitPosition finds the highest set bit of a size-attribute byte,
// returning its index. Ported from the repeated expression in
// GetOperandSize().
func x86HighBitPosition(s float64) float64 {
	var a, b, c float64
	if jsAnd(s, 0xF0) != 0 {
		s = jsShr(s, 4)
		a = 4
	}
	if jsAnd(s, 0xC) != 0 {
		s = jsShr(s, 2)
		b = 2
	}
	s = jsShr(s, 1)
	if s != 0 {
		c = 1
	}
	return jsOr(jsOr(a, b), c)
}

// getOperandSize turns a size-attribute bitmap into an index into the register
// and pointer tables, picking between the attribute's three sizes according to
// the current prefixes. Ported from GetOperandSize().
func (d *x86Decoder) getOperandSize(sizeAttribute float64) float64 {
	s1, s2, s3 := x86SizeAttributeTriple(sizeAttribute)
	s1, s2, s3 = d.clampSizesToBitMode(s1, s2, s3)

	// Under a vector extension the W bit acts as a 32/64 selector.
	if (d.vect || d.extension > 0) &&
		jsOr(jsBool(s1+s2+s3 == 7), jsBool(s1+s2+s3 == 5)) != 0 {
		d.vect = false
		return [2]float64{s2, s1}[int(jsAnd(d.widthBit, 1))]
	}

	// A broadcast round takes the vector to its maximum size; s0 is otherwise
	// the unused 1024-bit slot.
	s0 := -1.0
	if d.vect && d.conversionMode == 1 {
		s0, s3, s2 = s1, s1, s1
	}

	sizes := [4]float64{s3, s2, s1, s0}
	i := int(jsInt32(d.sizeAttrSelect))
	if i < 0 || i >= len(sizes) {
		return 0
	}
	return sizes[i]
}

// x86SizeAttributeTriple peels the three highest set bits off a size-attribute
// bitmap, giving the largest, default and smallest sizes an operand can take.
// A slot with no bit of its own aliases to the next one up.
func x86SizeAttributeTriple(sizeAttribute float64) (s1, s2, s3 float64) {
	s1 = x86HighBitPosition(sizeAttribute)
	if sizeAttribute == 0 {
		s1 = -1
	}
	sizeAttribute -= jsShl(1, s1)

	s2 = x86HighBitPosition(sizeAttribute)
	if s2 != 0 {
		sizeAttribute -= jsShl(1, s2)
	} else {
		s2 = s1
	}

	s3 = x86HighBitPosition(sizeAttribute)
	if s3 == 0 {
		// The engine subtracts this bit too, but nothing reads the attribute
		// again, so only the aliasing has any effect.
		s3 = s2
		if s2 != 2 {
			s2 = s1
		}
	}
	return s1, s2, s3
}

// clampSizesToBitMode applies the two rules that depend on the processor mode:
// below 64-bit an operand never exceeds 32, and in 16-bit mode the operand
// override is active until used, which swaps the default and smallest sizes.
func (d *x86Decoder) clampSizesToBitMode(s1, s2, s3 float64) (float64, float64, float64) {
	if d.bitMode <= x86Mode32 && s2 >= 3 && !d.vect {
		if jsOr(jsOr(s1, s2), s3) == s3 {
			s1, s3 = 2, 2 // a single size: everything drops to 32
		}
		s2 = 2
	}
	if d.bitMode == x86Mode16 && !d.vect {
		s2, s3 = s3, s2
	}
	return s1, s2, s3
}

// decodeModRMSIBValue splits the byte at the current position into its mode,
// reg and r/m fields, which the SIB byte reuses as scale, index and base.
// Ported from Decode_ModRM_SIB_Value().
func (d *x86Decoder) decodeModRMSIBValue() [3]float64 {
	v := d.cur()
	out := [3]float64{
		jsAnd(jsShr(v, 6), 0x03),
		jsAnd(jsShr(v, 3), 0x07),
		jsAnd(v, 0x07),
	}
	d.nextByte()
	return out
}

// Immediate decode types, matching the engine's `type` argument.
const (
	x86ImmPlain    = 0 // sized immediate
	x86ImmRegUpper = 1 // immediate whose top four bits select a register
	x86ImmRelative = 2 // relative address, resolved against the instruction pointer
	x86ImmDisp     = 3 // displacement, signed around a centre point
)

// resolveRelative turns a relative immediate into the absolute address the
// engine prints, splitting the carry across the two 32-bit halves.
func (d *x86Decoder) resolveRelative(v32, v64, s float64) (float64, float64, int, int) {
	pad32 := int(math.Min(float64(d.bitMode), 1))<<2 + 4
	pad64 := int(math.Max(math.Min(float64(d.bitMode), 2), 1)) << 3

	size := math.Min(0x100000000, math.Pow(2, jsShl(4, s+1)))
	if v32 >= size/2 {
		v32 -= size
	}
	v32 += d.pos32

	carry := v32 >= 0x100000000
	if carry {
		v32 -= 0x100000000
	}
	if s <= 2 {
		carry = false
	}
	v64 += d.pos64 + jsBool(carry)
	if v64 > 0xFFFFFFFF {
		v64 -= 0x100000000
	}
	return v32, v64, pad32, pad64
}

// resolveDisplacement folds a displacement around its centre point, returning
// the magnitude and whether it is added or subtracted.
func (d *x86Decoder) resolveDisplacement(v32, n, s float64) (float64, int) {
	centre := 2 * jsShl(1, jsShl(n, 3)-2)

	if d.vsib != 0 && s == 0 {
		vScale := jsOr(d.widthBit, 2)
		centre = jsShl(centre, vScale)
		v32 = jsShl(v32, vScale)
	}

	if v32 >= centre {
		return centre*2 - v32, 2
	}
	return v32, 1
}

// decodeImmediate reads an immediate and renders it as CyberChef prints it:
// zero-padded hexadecimal, sign-extended where the size attribute asks for it,
// and prefixed with + or - for displacements. Ported from DecodeImmediate().
func (d *x86Decoder) decodeImmediate(typ int, bySize bool, sizeSetting float64) string {
	var v32, v64 float64
	var pad32, pad64 int
	sign := 0

	s := jsAnd(sizeSetting, 0x0F)
	extend := jsShr(sizeSetting, 4)

	if bySize {
		s = d.getOperandSize(s)
		if extend > 0 {
			extend = d.getOperandSize(extend)
		}
	}

	n := jsShl(1, s)
	pad32 = int(math.Min(n, 4))
	if n >= 8 {
		pad64 = 8
	}

	d.immValue = d.cur()

	i := 0
	for v := 1.0; i < pad32; i, v = i+1, v*256 {
		v32 += d.cur() * v
		d.nextByte()
	}
	for v := 1.0; i < pad64; i, v = i+1, v*256 {
		v64 += d.cur() * v
		d.nextByte()
	}

	pad32 <<= 1
	pad64 <<= 1

	if typ == x86ImmRegUpper {
		v32 = jsAnd(v32, jsShl(1, jsShl(n, 3)-4)-1)
	}

	switch typ {
	case x86ImmRelative:
		v32, v64, pad32, pad64 = d.resolveRelative(v32, v64, s)
	case x86ImmDisp:
		v32, sign = d.resolveDisplacement(v32, n, s)
		pad64 = 0
	}

	imm := jsPadLeft(jsHex16(v32), "0", pad32)
	if pad64 > 8 {
		imm = jsPadLeft(jsHex16(v64)+imm, "0", pad64)
	}

	if extend != s {
		width := int(math.Pow(2, extend) * 2)
		spd := "00"
		if jsShr(jsAnd(jsParseHex(jsSlice(imm, 0, 1)), 8), 3) != 0 {
			spd = "FF"
		}
		for len(imm) < width {
			imm = spd + imm
		}
	}

	prefix := ""
	if sign == 1 {
		prefix = "+"
	} else if sign > 1 {
		prefix = "-"
	}
	return prefix + strings.ToUpper(imm)
}

// decodeRegValue names a register by value and size setting.
// Ported from DecodeRegValue().
func (d *x86Decoder) decodeRegValue(rValue float64, bySize bool, setting float64) string {
	if d.vect && d.extension == 0 {
		d.sizeAttrSelect = 0
	}

	if bySize {
		setting = d.getOperandSize(setting)
		// XMM is the smallest vector register there is.
		if d.vect && setting < 4 {
			setting = 4
		}
	}

	if d.opcode >= 0x400 {
		rValue = jsAnd(rValue, 15)
	} else if d.bitMode <= x86Mode32 && d.extension >= 1 {
		rValue = jsAnd(rValue, 7)
	}

	if d.opcode >= 0x700 && setting == 6 {
		setting = 16
	} else if setting == 0 {
		return d.reg8(d.rexActive, rValue)
	}

	return d.reg(setting, rValue)
}

// x86ModRMRegisterMode is the ModR/M mode value that selects a register rather
// than a memory address.
const x86ModRMRegisterMode = 3

// decodeModRMSIBAddress renders a ModR/M operand: either a register, or an
// effective address including any SIB byte, displacement and vector conversion
// suffix. Ported from Decode_ModRM_SIB_Address().
func (d *x86Decoder) decodeModRMSIBAddress(modRM [3]float64, bySize bool, setting float64) string {
	out := ""
	sc := "{"

	if modRM[0] != x86ModRMRegisterMode {
		out, sc = d.decodeEffectiveAddress(modRM, bySize, setting, sc)
	} else {
		out, sc = d.decodeRegisterMode(modRM, bySize, setting, sc)
	}

	if d.opcode >= 0x700 {
		out, sc = d.decodeL1OMConversion(out, sc)
	}
	if sc == "," {
		sc = "}"
	}
	if sc == "}" {
		out += sc
	}

	if d.hintZeroMerg != 0 {
		if d.extension == 3 {
			out += "{EH}"
		} else if d.opcode >= 0x700 {
			out += "{NT}"
		}
	}
	return out
}

// decodeRegisterMode handles ModR/M mode 11, where the r/m field names a
// register instead of an address.
func (d *x86Decoder) decodeRegisterMode(modRM [3]float64, bySize bool, setting float64, sc string) (string, string) {
	if (d.extension == 3 && d.hintZeroMerg != 0) || (d.extension == 2 && d.conversionMode == 1) {
		d.roundMode = jsOr(d.roundMode, d.roundingSetting)
	}

	// With no size attribute the upper four bits select the register directly.
	if jsAnd(setting, 0xF0) > 0 && !bySize {
		setting = jsShr(setting, 4)
	}

	out := d.decodeRegValue(jsOr(d.baseExtend, modRM[2]), bySize, setting)

	if d.opcode >= 0x700 || (d.extension == 3 && d.hintZeroMerg == 0 && d.swizzle != 0) {
		if d.opcode >= 0x700 && d.conversionMode >= 3 {
			d.conversionMode++ // L1OM skips swizzle type DACB.
		}
		if d.conversionMode != 0 {
			out += sc + x86At(d.tables.RegSwizzleModes, d.conversionMode)
			sc = ","
		}
	}
	if d.extension != 2 {
		d.hintZeroMerg = 0
	}
	return out, sc
}

// x86AddressSize values, as the engine numbers them.
const (
	x86Addr16 = 1
	x86Addr32 = 2
	x86Addr64 = 3
)

// decodeEffectiveAddress handles ModR/M modes 00, 01 and 10.
func (d *x86Decoder) decodeEffectiveAddress(modRM [3]float64, bySize bool, setting float64, sc string) (string, string) {
	if d.vect && d.extension == 0 {
		d.sizeAttrSelect = 0
	}

	setting = d.pointerSetting(bySize, setting)
	out := d.pointerPrefix(setting) + d.segOverride

	addressSize := d.effectiveAddressSize()
	disp, dispType := x86DisplacementFor(modRM, addressSize)

	if addressSize == x86Addr16 {
		out, disp, dispType = d.decode16BitAddress(out, modRM, disp, dispType)
	} else {
		out, disp, dispType = d.decode32BitAddress(out, modRM, disp, dispType, addressSize, setting)
	}

	if disp >= 0 {
		out += d.decodeImmediate(dispType, false, disp)
	}
	out += "]"

	return d.decodeMemoryConversion(out, sc)
}

// pointerSetting resolves the size attribute into an index into the pointer
// table.
func (d *x86Decoder) pointerSetting(bySize bool, setting float64) float64 {
	if bySize {
		if setting != 16 || d.vect {
			setting = jsOr(jsShl(d.getOperandSize(setting), 1), d.farPointer)
		} else {
			// Non-vector 128 aliases to QWORD PTR in 32-bit mode and below.
			setting = 11 - jsBool(d.bitMode <= x86Mode32)*5
		}
	}
	setting = jsAnd(setting, 0x0F)

	if d.extension != 0 && setting == 9 {
		setting = 6 // MM becomes QWORD under a vector extension.
	}
	return setting
}

// pointerPrefix names the memory pointer. A broadcast or VSIB operand always
// prints an element-sized pointer rather than the operand's own size.
func (d *x86Decoder) pointerPrefix(setting float64) string {
	if d.conversionMode == 1 || d.conversionMode == 2 || d.vsib != 0 {
		element := 4.0
		if d.widthBit > 0 {
			element = 6
		}
		return x86At(d.tables.PTR, element)
	}
	return x86At(d.tables.PTR, setting)
}

// effectiveAddressSize is the address width the ModR/M encoding uses, which the
// address-size override moves one step down — and, from 16-bit mode, up to 32.
func (d *x86Decoder) effectiveAddressSize() int {
	size := d.bitMode + 1
	if d.addressOverride {
		size--
		if size == 0 {
			size = x86Addr32
		}
	}
	return size
}

// x86DisplacementFor derives the displacement size and kind from the ModR/M
// mode bits: mode 1 always adds a disp8, and mode 2 adds a disp16 or disp32
// depending on the address width.
func x86DisplacementFor(modRM [3]float64, addressSize int) (float64, int) {
	disp := modRM[0] - 1
	if addressSize >= x86Addr32 && modRM[0] == 2 {
		disp++
	}
	return disp, x86ImmDisp
}

// decode16BitAddress renders the 16-bit ModR/M base/index register pairs.
func (d *x86Decoder) decode16BitAddress(out string, modRM [3]float64, disp float64, dispType int) (string, float64, int) {
	if modRM[0] == 0 && modRM[2] == 6 {
		disp = 1
		dispType = x86ImmPlain
	}

	// BX and BP alternate on bit 2 of the r/m value.
	if modRM[2] < 4 {
		out += d.reg(x86Addr16, 3+jsAnd(modRM[2], 2)) + "+"
	}
	// Bit 0 switches between destination and source index.
	if modRM[2] < 6 {
		out += d.reg(x86Addr16, 6+jsAnd(modRM[2], 1))
	} else if dispType != x86ImmPlain {
		out += d.reg(x86Addr16, 17-jsShl(modRM[2], 1))
	}
	return out, disp, dispType
}

// decode32BitAddress renders the 32/64-bit ModR/M address, decoding a SIB byte
// where the r/m field calls for one.
func (d *x86Decoder) decode32BitAddress(out string, modRM [3]float64, disp float64, dispType, addressSize int, setting float64) (string, float64, int) {
	// Mode 0 with base 5 is a RIP-relative disp32.
	if modRM[0] == 0 && modRM[2] == 5 {
		disp = 2
		dispType = x86ImmRelative
	}

	switch {
	case modRM[2] == 4: // the base field routes into the SIB byte
		out, disp, dispType = d.decodeSIBAddress(out, modRM, disp, dispType, addressSize, setting)
	case dispType != x86ImmRelative:
		out += d.reg(float64(addressSize), jsOr(jsAnd(d.baseExtend, 8), modRM[2]))
	}

	return out, disp, dispType
}

// decodeSIBAddress renders the scale-index-base form, whose base and index can
// each cancel out and leave a bare displacement behind.
func (d *x86Decoder) decodeSIBAddress(out string, modRM [3]float64, disp float64, dispType, addressSize int, setting float64) (string, float64, int) {
	sib := d.decodeModRMSIBValue()
	indexReg := jsOr(d.indexExtend, sib[1])

	baseCancelled := modRM[0] == 0 && sib[2] == 5 && d.vsib == 0
	if baseCancelled {
		disp = 2
		if indexReg == 4 { // the index cancels out too, leaving a bare address
			dispType = x86ImmPlain
			if addressSize == x86Addr64 {
				disp = 50 // pad the 32-bit immediate out to a full 64-bit address
			}
		}
	} else {
		out += d.reg(float64(addressSize), jsOr(jsAnd(d.baseExtend, 8), sib[2]))
		if indexReg != 4 || d.vsib != 0 {
			out += "+"
		}
	}

	return out + d.sibIndexOperand(sib, indexReg, addressSize, setting), disp, dispType
}

// sibIndexOperand renders the index register and its scale, which is a vector
// register under VSIB and nothing at all when the index cancels out.
func (d *x86Decoder) sibIndexOperand(sib [3]float64, indexReg float64, addressSize int, setting float64) string {
	switch {
	case indexReg != 4 && d.vsib == 0:
		return d.reg(float64(addressSize), jsOr(d.indexExtend, indexReg)) + x86At(d.tables.Scale, sib[0])

	case d.vsib != 0:
		if setting < 8 {
			setting = 4 // no vector register is smaller than XMM
		} else {
			setting = jsShr(setting, 1)
		}
		if d.opcode < 0x700 {
			indexReg = jsOr(indexReg, jsAnd(d.vectorRegister, 0x10))
		}
		return d.decodeRegValue(jsOr(d.indexExtend, indexReg), false, setting) +
			x86At(d.tables.Scale, sib[0])
	}
	return ""
}

// decodeMemoryConversion appends the MVEX/EVEX broadcast or conversion suffix
// that follows a memory operand.
func (d *x86Decoder) decodeMemoryConversion(out, sc string) (string, string) {
	if d.conversionMode == 0 {
		return out, sc
	}
	if !d.conversionIsUsable() {
		return out + sc + "Error", ","
	}

	d.adjustConversionMode()

	// The K1OM special case inverts the width bit.
	l1om := d.opcode >= 0x700
	idx := jsOr(jsShl(d.conversionMode, 1),
		jsAnd(jsXor(d.widthBit, jsAnd(jsBool(!l1om), jsBool(d.vectS == 7))), 1))
	return out + sc + x86At(d.tables.ConversionModes, idx), ","
}

// conversionIsUsable reports whether the current conversion setting is valid
// for the instruction, which depends on the opcode map and the vector format.
func (d *x86Decoder) conversionIsUsable() bool {
	l1om := d.opcode >= 0x700

	badFloat := d.conversionMode == 3 && (l1om || (!l1om && d.float == 0))
	swizzleCase := jsXor(
		jsBool(d.conversionMode != 1 && d.vectS == 1),
		jsBool(d.conversionMode < 3 && d.swizzle == 0),
	) != 0
	badSwizzle := !l1om && (d.vectS == 0 || (d.conversionMode == 5 && d.vectS == 5) || swizzleCase)

	return !badFloat && !badSwizzle
}

// adjustConversionMode applies the spacing and Larrabee corrections the engine
// makes before indexing the conversion table.
func (d *x86Decoder) adjustConversionMode() {
	if d.conversionMode >= 4 {
		d.conversionMode += 2
	}
	if d.conversionMode >= 8 {
		d.conversionMode += 2
	}
	if d.opcode < 0x700 {
		return
	}

	if d.swizzle == 0 && d.conversionMode > 2 {
		d.conversionMode = 31
		return
	}
	if d.float != 0 {
		if d.conversionMode == 7 {
			d.conversionMode++
		}
		if d.conversionMode == 10 {
			d.conversionMode = 3
		}
	}
}

// decodeL1OMConversion appends the Larrabee swizzle and up/down conversion
// suffixes that only the L1OM opcode map uses.
func (d *x86Decoder) decodeL1OMConversion(out, sc string) (string, string) {
	if d.swizzle != 0 {
		switch {
		case d.opcode == 0x79A:
			out += sc + x86At(d.tables.ConversionModes, jsShl(jsOr(18, jsAnd(d.vectorRegister, 3)), 1))
			sc = "}"
		case d.opcode == 0x79B:
			out += sc + x86At(d.tables.ConversionModes, jsShl(22+jsAnd(d.vectorRegister, 3), 1))
			sc = "}"
		case jsAnd(d.roundingSetting, 8) == 8:
			out += sc + x86At(d.tables.RoundModes, jsOr(24, jsAnd(d.vectorRegister, 7)))
			sc = "}"
		}
		return out, sc
	}

	if d.vectorRegister != 0 {
		upOK := d.up != 0 && d.vectorRegister != 2
		downOK := d.up == 0 && d.vectorRegister != 3 && d.vectorRegister <= 15
		if upOK || downOK {
			out += sc + x86At(d.tables.ConversionModes, jsOr(jsShl(d.vectorRegister+2, 1), d.widthBit))
		} else {
			out += sc + "Error"
		}
		sc = "}"
	}
	return out, sc
}

// isEmptyLeaf reports whether the node is the empty string the tables use to
// mark an unused slot.
func (n x86Node) isEmptyLeaf() bool { return !n.isList && n.str == "" }

// x86Leaf builds a leaf node.
func x86Leaf(s string) x86Node { return x86Node{str: s} }

// x86Invalid is the engine's placeholder for an unrecognised instruction.
const x86Invalid = "???"

// decodePrefixAdjustments consumes every prefix byte, applying its effect to
// the decoder, and leaves d.opcode holding a real operation code. Ported from
// DecodePrefixAdjustments(), whose tail recursion becomes a loop here.
func (d *x86Decoder) decodePrefixAdjustments() {
	for {
		d.opcode = jsOr(jsAnd(d.opcode, 0x300), d.cur())
		d.nextByte()

		if d.applyOpcodeMapEscape() || d.applyREXPrefix() {
			continue
		}
		if d.decodeVectorPrefix() {
			return
		}
		if d.applyLegacyPrefix() {
			continue
		}

		d.rejectOpcodesAbsentIn64Bit()
		return
	}
}

// applyOpcodeMapEscape moves the opcode onto the two- or three-byte map when
// the byte just read was one of the escapes, reporting whether it did.
func (d *x86Decoder) applyOpcodeMapEscape() bool {
	switch {
	case d.opcode == 0x0F:
		d.opcode = 0x100
	case d.opcode == 0x138 && d.mnemonic(0x138).isEmptyLeaf():
		d.opcode = 0x200
	case d.opcode == 0x13A && d.mnemonic(0x13A).isEmptyLeaf():
		d.opcode = 0x300
	default:
		return false
	}
	return true
}

// applyREXPrefix consumes a REX byte, which only exists in 64-bit mode and
// carries the register extensions and the operand width bit.
func (d *x86Decoder) applyREXPrefix() bool {
	inRange := jsAnd(jsBool(d.opcode >= 0x40), jsBool(d.opcode <= 0x4F)) != 0
	if !inRange || d.bitMode != x86Mode64 {
		return false
	}

	d.rexActive = 1
	d.baseExtend = jsShl(jsAnd(d.opcode, 0x01), 3)
	d.indexExtend = jsShl(jsAnd(d.opcode, 0x02), 2)
	d.regExtend = jsShl(jsAnd(d.opcode, 0x04), 1)
	d.widthBit = jsShr(jsAnd(d.opcode, 0x08), 3)

	d.sizeAttrSelect = 1
	if d.widthBit != 0 {
		d.sizeAttrSelect = 2 // the width bit opens all 64 bits
	}
	return true
}

// applyLegacyPrefix consumes a segment override, an operand- or address-size
// override, or a repeat or lock prefix, reporting whether it did.
func (d *x86Decoder) applyLegacyPrefix() bool {
	switch {
	case jsAnd(d.opcode, 0x7E7) == 0x26 || jsAnd(d.opcode, 0x7FE) == 0x64:
		d.segOverride = d.mnemonicStr(d.opcode)

	case d.opcode == 0x66: // operand size override
		d.simd = 1
		d.sizeAttrSelect = 0

	case d.opcode == 0x67: // address size override
		d.addressOverride = true

	case d.opcode == 0xF2 || d.opcode == 0xF3: // repeat
		d.simd = jsOr(jsAnd(d.opcode, 0x02), 1-jsAnd(d.opcode, 0x01))
		d.prefixG1 = d.mnemonicStr(d.opcode)
		d.hleFlipG1G2 = true

	case d.opcode == 0xF0: // lock
		d.prefixG2 = d.mnemonicStr(d.opcode)
		d.hleFlipG1G2 = false

	default:
		return false
	}
	return true
}

// rejectOpcodesAbsentIn64Bit marks the opcodes 64-bit mode reassigned or
// removed, which the engine reports as an invalid instruction.
func (d *x86Decoder) rejectOpcodesAbsentIn64Bit() {
	if d.bitMode != x86Mode64 {
		return
	}
	for _, absent := range []float64{
		jsAnd(jsBool(jsAnd(d.opcode, 0x07) >= 0x06), jsBool(d.opcode <= 0x40)),
		jsOr(jsBool(d.opcode == 0x60), jsBool(d.opcode == 0x61)),
		jsOr(jsBool(d.opcode == 0xD4), jsBool(d.opcode == 0xD5)),
		jsOr(jsBool(d.opcode == 0x9A), jsBool(d.opcode == 0xEA)),
		jsBool(d.opcode == 0x82),
	} {
		d.invalidOp = jsOr(d.invalidOp, absent)
	}
}

// decodeVectorPrefix handles the VEX, XOP, MVEX/EVEX and L1OM prefixes, each of
// which carries the opcode itself and so ends prefix decoding. It reports
// whether one was consumed.
func (d *x86Decoder) decodeVectorPrefix() bool {
	switch {
	case d.opcode == 0xC5 && d.vectorPrefixApplies():
		d.decodeVEX2()
	case d.opcode == 0xC4 && d.vectorPrefixApplies():
		d.decodeVEX3()
	case d.opcode == 0x8F && d.isXOPPrefix():
		d.decodeXOP()
	case d.opcode == 0xD6:
		d.decodeL1OMVector()
	case d.opcode == 0x62 && d.mnemonic(0x62).isEmptyLeaf():
		d.decodeL1OMShort()
	case d.opcode == 0x62 && d.vectorPrefixApplies():
		d.decodeEVEX()
	default:
		return false
	}
	return true
}

// vectorPrefixApplies reports whether the byte after a VEX or EVEX escape can
// be read as prefix settings. Outside 64-bit mode the escape is only a prefix
// when the next byte would otherwise be an invalid ModR/M.
func (d *x86Decoder) vectorPrefixApplies() bool {
	return d.cur() >= 0xC0 || d.bitMode == x86Mode64
}

// isXOPPrefix reports whether 8F introduces an AMD XOP instruction rather than
// the POP it otherwise encodes.
func (d *x86Decoder) isXOPPrefix() bool {
	code := jsAnd(d.cur(), 0x0F)
	return code >= 8 && code <= 10
}

// readOpcodeByte reads the byte at the current position into the low eight bits
// of the opcode, keeping the map bits given by mask.
func (d *x86Decoder) readOpcodeByte(mask float64) {
	d.opcode = jsOr(jsAnd(d.opcode, mask), d.cur())
	d.nextByte()
}

// readPrefixByte reads one settings byte into the opcode at the given shift.
func (d *x86Decoder) readPrefixByte(shift float64) {
	if shift == 0 {
		d.opcode = d.cur()
	} else {
		d.opcode = jsOr(d.opcode, jsShl(d.cur(), shift))
	}
	d.nextByte()
}

func (d *x86Decoder) decodeVEX2() {
	d.extension = 1
	d.readPrefixByte(0)
	d.opcode = jsXor(d.opcode, 0xF8)

	if d.bitMode == x86Mode64 {
		d.regExtend = jsShr(jsAnd(d.opcode, 0x80), 4)
		d.vectorRegister = jsShr(jsAnd(d.opcode, 0x78), 3)
	}
	d.sizeAttrSelect = jsShr(jsAnd(d.opcode, 0x04), 2)
	d.simd = jsAnd(d.opcode, 0x03)

	d.opcode = 0x100
	d.readOpcodeByte(0x300)
}

func (d *x86Decoder) decodeVEX3() {
	d.extension = 1
	d.readPrefixByte(0)
	d.readPrefixByte(8)
	d.opcode = jsXor(d.opcode, 0x78E0)

	if d.bitMode == x86Mode64 {
		d.regExtend = jsShr(jsAnd(d.opcode, 0x0080), 4)
		d.indexExtend = jsShr(jsAnd(d.opcode, 0x0040), 3)
		d.baseExtend = jsShr(jsAnd(d.opcode, 0x0020), 2)
	}
	d.widthBit = jsShr(jsAnd(d.opcode, 0x8000), 15)
	d.vectorRegister = jsShr(jsAnd(d.opcode, 0x7800), 11)
	d.sizeAttrSelect = jsShr(jsAnd(d.opcode, 0x0400), 10)
	d.simd = jsShr(jsAnd(d.opcode, 0x0300), 8)
	d.opcode = jsShl(jsAnd(d.opcode, 0x001F), 8)

	d.readOpcodeByte(0x300)
}

func (d *x86Decoder) decodeXOP() {
	d.extension = 1
	d.readPrefixByte(0)
	d.readPrefixByte(8)
	d.opcode = jsXor(d.opcode, 0x78E0)

	d.regExtend = jsShr(jsAnd(d.opcode, 0x0080), 4)
	d.indexExtend = jsShr(jsAnd(d.opcode, 0x0040), 3)
	d.baseExtend = jsShr(jsAnd(d.opcode, 0x0020), 2)
	d.widthBit = jsShr(jsAnd(d.opcode, 0x8000), 15)
	d.vectorRegister = jsShr(jsAnd(d.opcode, 0x7800), 11)
	d.sizeAttrSelect = jsShr(jsAnd(d.opcode, 0x0400), 10)
	d.simd = jsShr(jsAnd(d.opcode, 0x0300), 8)
	if d.simd > 0 {
		d.invalidOp = 1
	}
	d.opcode = jsOr(0x400, jsShl(jsAnd(d.opcode, 0x0003), 8))

	d.readOpcodeByte(0x700)
}

func (d *x86Decoder) decodeL1OMVector() {
	d.readPrefixByte(0)
	d.readPrefixByte(8)

	d.widthBit = jsAnd(d.simd, 1)
	d.vectorRegister = jsShr(jsAnd(d.opcode, 0xF800), 11)
	d.roundMode = jsShr(d.vectorRegister, 3)
	d.maskRegister = jsShr(jsAnd(d.opcode, 0x0700), 8)
	d.hintZeroMerg = jsShr(jsAnd(d.opcode, 0x0080), 7)
	d.conversionMode = jsShr(jsAnd(d.opcode, 0x0070), 4)
	d.regExtend = jsShl(jsAnd(d.opcode, 0x000C), 1)
	d.baseExtend = jsShl(jsAnd(d.opcode, 0x0003), 3)
	d.indexExtend = jsShl(jsAnd(d.opcode, 0x0002), 2)

	d.opcode = jsOr(0x700, d.cur())
	d.nextByte()
}

func (d *x86Decoder) decodeL1OMShort() {
	d.readPrefixByte(0)
	d.opcode = jsXor(d.opcode, 0xF0)

	d.indexExtend = jsShr(jsAnd(d.opcode, 0x80), 4)
	d.baseExtend = jsShr(jsAnd(d.opcode, 0x40), 3)
	d.regExtend = jsShr(jsAnd(d.opcode, 0x20), 2)

	if d.simd != 1 {
		if jsAnd(d.opcode, 0x10) == 0x10 {
			d.sizeAttrSelect = 2
		} else {
			d.sizeAttrSelect = 1
		}
	} else {
		d.simd = 0
	}

	d.opcode = jsOr(jsOr(0x800, jsShr(jsAnd(d.opcode, 0x30), 4)), jsShl(jsAnd(d.opcode, 0x0F), 2))
}

func (d *x86Decoder) decodeEVEX() {
	d.extension = 2
	d.readPrefixByte(0)
	d.readPrefixByte(8)
	d.readPrefixByte(16)
	d.opcode = jsXor(d.opcode, 0x0878F0)

	// Reserved bits must be clear.
	d.invalidOp = jsBool(jsAnd(d.opcode, 0x00000C) > 0)

	if d.bitMode == x86Mode64 {
		d.regExtend = jsOr(jsShr(jsAnd(d.opcode, 0x80), 4), jsAnd(d.opcode, 0x10))
		d.baseExtend = jsShr(jsAnd(d.opcode, 0x60), 2)
		d.indexExtend = jsShr(jsAnd(d.opcode, 0x40), 3)
	}

	d.vectorRegister = jsOr(jsShr(jsAnd(d.opcode, 0x7800), 11), jsShr(jsAnd(d.opcode, 0x080000), 15))
	d.widthBit = jsShr(jsAnd(d.opcode, 0x8000), 15)
	d.simd = jsShr(jsAnd(d.opcode, 0x0300), 8)
	d.hintZeroMerg = jsShr(jsAnd(d.opcode, 0x800000), 23)

	if jsAnd(d.opcode, 0x0400) > 0 {
		d.sizeAttrSelect = jsShr(jsAnd(d.opcode, 0x600000), 21)
		d.roundMode = jsOr(d.sizeAttrSelect, 4)
		d.conversionMode = jsShr(jsAnd(d.opcode, 0x100000), 20)
	} else {
		d.sizeAttrSelect = 2
		d.conversionMode = jsShr(jsAnd(d.opcode, 0x700000), 20)
		d.roundMode = d.conversionMode
		d.extension = 3
	}

	d.maskRegister = jsShr(jsAnd(d.opcode, 0x070000), 16)
	d.opcode = jsShl(jsAnd(d.opcode, 0x03), 8)

	d.readOpcodeByte(0x300)
}

// narrowByModRM resolves the entries that split on the ModR/M byte: a
// two-entry list separates register mode from memory mode, and an eight-entry
// list is a group opcode selected by the reg field, optionally followed by a
// static opcode on the r/m field.
func (d *x86Decoder) narrowByModRM(instr, oper x86Node, modRM float64) (x86Node, x86Node) {
	if instr.length() == 2 {
		bits := jsAnd(jsShr(modRM, 6), jsShr(modRM, 7))
		instr, oper = instr.at(bits), oper.at(bits)
	}

	if instr.length() == 8 {
		bits := jsShr(jsAnd(modRM, 0x38), 3)
		instr, oper = instr.at(bits), oper.at(bits)

		if instr.length() == 8 {
			bits = jsAnd(modRM, 0x07)
			instr, oper = instr.at(bits), oper.at(bits)
			d.nextByte()
		}
	}
	return instr, oper
}

// narrowBySIMD resolves a four-entry list, which is a SIMD instruction across
// modes N/A, 66, F3 and F2, and any nested list separating the vector
// extensions.
func (d *x86Decoder) narrowBySIMD(instr, oper x86Node) (x86Node, x86Node) {
	if instr.length() != 4 {
		if d.opcode >= 0x700 && d.simd > 0 {
			return x86Leaf(x86Invalid), x86Leaf("")
		}
		return instr, oper
	}

	d.vect = true
	if !instr.at(2).isEmptyLeaf() && !instr.at(3).isEmptyLeaf() {
		d.prefixG1 = ""
	} else {
		d.simd = jsAnd(jsBool(d.simd == 1), 1)
	}
	instr, oper = instr.at(d.simd), oper.at(d.simd)

	if instr.length() == 4 {
		if instr.at(d.extension).isEmptyLeaf() {
			return x86Leaf(x86Invalid), x86Leaf("")
		}
		return instr.at(d.extension), oper.at(d.extension)
	}
	if d.extension == 3 {
		return x86Leaf(x86Invalid), x86Leaf("")
	}
	return instr, oper
}

// narrowBySize resolves a three-entry list, which names the instruction by its
// operand size.
func (d *x86Decoder) narrowBySize(instr, oper x86Node) (x86Node, x86Node) {
	if instr.length() != 3 {
		return instr, oper
	}

	bits := jsXor(
		jsAnd(jsBool(d.extension == 0), jsBool(d.bitMode != x86Mode16)),
		jsBool(d.sizeAttrSelect >= 1),
	)
	if d.widthBit != 0 {
		bits = 2
	}
	if d.extension == 3 && d.hintZeroMerg != 0 && !instr.at(1).isEmptyLeaf() {
		d.hintZeroMerg = 0
		bits = 1
	}
	if instr.at(bits).isEmptyLeaf() {
		bits = 0
	}
	return instr.at(bits), oper.at(bits)
}

// decodeOpcode narrows the opcode's table entries down to one mnemonic and one
// operand string, following the ModR/M bits, SIMD mode, vector extension and
// operand size. Ported from DecodeOpcode().
func (d *x86Decoder) decodeOpcode() {
	instr := d.mnemonic(d.opcode)
	oper := d.operandEntry(d.opcode)
	modRM := d.cur()

	instr, oper = d.narrowByModRM(instr, oper, modRM)
	instr, oper = d.narrowBySIMD(instr, oper)
	instr, oper = d.narrowBySize(instr, oper)

	d.instruction = instr.str
	d.insOperands = oper.str

	// A vector extension adds the leading V, except on the mask instructions.
	if d.opcode <= 0x400 && d.extension > 0 &&
		!strings.HasPrefix(d.instruction, "K") && d.instruction != x86Invalid {
		d.instruction = "V" + d.instruction
	}

	// Below 64-bit mode the one instruction MOVSXD is replaced by ARPL.
	if d.bitMode <= x86Mode32 && d.instruction == "MOVSXD" {
		d.instruction = "ARPL"
		d.insOperands = "06020A01"
	}
}

// x86OperandList is the sparse array the engine fills by operand number. A slot
// never written stringifies to nothing when joined, but to "undefined" when
// something is appended to it, exactly as in JavaScript.
type x86OperandList struct {
	vals []string
	set  []bool
}

func (o *x86OperandList) grow(i int) {
	for len(o.vals) <= i {
		o.vals = append(o.vals, "")
		o.set = append(o.set, false)
	}
}

func (o *x86OperandList) put(i int, s string) {
	o.grow(i)
	o.vals[i], o.set[i] = s, true
}

func (o *x86OperandList) appendTo(i int, s string) {
	o.grow(i)
	if !o.set[i] {
		o.vals[i], o.set[i] = x86UndefinedValue, true
	}
	o.vals[i] += s
}

// join renders the list the way Array.prototype.toString() does. A slot holding
// undefined — an out-of-range table lookup that was assigned straight into the
// list rather than concatenated onto something — contributes nothing, even
// though the same value spells "undefined" when appended to a string.
func (o *x86OperandList) join() string {
	parts := make([]string, len(o.vals))
	for i, v := range o.vals {
		if v == x86UndefinedValue {
			continue
		}
		parts[i] = v
	}
	return strings.Join(parts, ",")
}

// Operand encoding codes used by the operand string.
const (
	x86CodeSettings  = 0
	x86CodeRegOpcode = 1
	x86CodeModRMLow  = 2
	x86CodeModRMHigh = 4
	x86CodeModRMReg  = 5
	x86CodeImmLow    = 6
	x86CodeImmHigh   = 8
	x86CodeVectorReg = 9
	x86CodeImmReg    = 10
	x86CodeExplicit  = 11
)

// decodeOperandString reads the four-hex-digit operand codes and marks the
// decoder slots each instruction uses. Ported from DecodeOperandString().
func (d *x86Decoder) decodeOperandString() {
	slots := x86SlotAssigner{imm: x86SlotImm1, explicit: x86SlotExplicit}
	opNum := 0

	for i := 0; i < len(d.insOperands); i += 4 {
		operandValue := jsParseHex(jsSlice(d.insOperands, i, i+4))
		code := jsShr(jsAnd(operandValue, 0xFE00), 9)
		bySize := jsShr(jsAnd(operandValue, 0x0100), 8) != 0
		setting := jsAnd(operandValue, 0x00FF)

		if code == x86CodeSettings {
			d.applyOperandSettings(bySize, setting)
			continue
		}

		slot, typ, ok := slots.next(code, d.extension > 0 || d.opcode >= 0x700)
		if !ok {
			continue
		}
		if code == x86CodeModRMHigh {
			d.farPointer = 1
		}
		d.decoder[slot] = x86Operand{typ: typ, bySize: bySize, size: setting, opNum: opNum, active: true}
		opNum++
	}
}

// x86SlotAssigner tracks the decoder slots an operand string has claimed, so
// that successive immediates and explicit operands land in consecutive slots.
type x86SlotAssigner struct {
	imm      int
	explicit int
}

// x86MaxImmSlot and x86MaxExplicitSlot bound the two runs of slots.
const (
	x86MaxImmSlot      = 5
	x86MaxExplicitSlot = 11
)

// next maps an operand code to the slot it decodes in and the type that slot
// takes, reporting false for a code with nowhere left to go. hasVector says
// whether a vector register is available, which the vector-register code needs.
func (a *x86SlotAssigner) next(code float64, hasVector bool) (int, float64, bool) {
	switch {
	case code == x86CodeRegOpcode:
		return x86SlotRegOpcode, 0, true

	case code >= x86CodeModRMLow && code <= x86CodeModRMHigh:
		return x86SlotModRMAddr, code - x86CodeModRMLow, true

	case code == x86CodeModRMReg:
		return x86SlotModRMReg, 0, true

	case code >= x86CodeImmLow && code <= x86CodeImmHigh && a.imm <= x86MaxImmSlot:
		slot := a.imm
		a.imm++
		return slot, code - x86CodeImmLow, true

	case code == x86CodeVectorReg && hasVector:
		return x86SlotVectorReg, 0, true

	case code == x86CodeImmReg:
		return x86SlotImmReg, 0, true

	case code >= x86CodeExplicit && a.explicit <= x86MaxExplicitSlot:
		slot := a.explicit
		a.explicit++
		return slot, code - x86CodeExplicit, true
	}
	return 0, 0, false
}

// applyOperandSettings handles operand code 0, which carries prefix and vector
// settings rather than an operand.
func (d *x86Decoder) applyOperandSettings(bySize bool, setting float64) {
	if bySize {
		d.roundingSetting = jsShl(jsAnd(setting, 0x03), 3)
		if d.opcode >= 0x700 && d.roundingSetting >= 0x10 {
			d.roundMode = jsOr(d.roundMode, 0x10)
		}
		d.vsib = jsAnd(jsShr(setting, 2), 1)
		d.ignoresWidthbit = jsAnd(jsShr(setting, 3), 1)
		d.vectS = jsAnd(jsShr(setting, 4), 7)
		d.swizzle = jsAnd(jsShr(d.vectS, 2), 1)
		d.up = jsAnd(jsShr(d.vectS, 1), 1)
		d.float = jsAnd(d.vectS, 1)
		if jsAnd(setting, 0x80) == 0x80 {
			d.vect = false
		}
		return
	}
	d.xRelease = jsAnd(setting, 0x01)
	d.xAcquire = jsShr(jsAnd(setting, 0x02), 1)
	d.ht = jsShr(jsAnd(setting, 0x04), 2)
	d.bnd = jsShr(jsAnd(setting, 0x08), 3)
}

// decodeOperands walks the decoder slots in the order the CPU decodes them and
// builds the printed operand list. Ported from DecodeOperands().
func (d *x86Decoder) decodeOperands() {
	out := &x86OperandList{}

	if op := d.decoder[x86SlotRegOpcode]; op.active {
		out.put(op.opNum, d.decodeRegValue(jsOr(d.regExtend, jsAnd(d.opcode, 0x07)), op.bySize, op.size))
	}

	d.decodeModRMOperands(out)
	immUsed := d.decodeImmediateOperands(out)
	d.decodeVectorOperands(out, immUsed)
	d.decodeExplicitOperands(out)
	d.applyDestinationMask(out)

	d.insOperands = out.join()
}

// decodeModRMOperands renders the ModR/M address and register operands. The
// byte is read at most once, since the two halves can be used independently.
func (d *x86Decoder) decodeModRMOperands(out *x86OperandList) {
	modRM := [3]float64{-1, 0, 0}

	if op := d.decoder[x86SlotModRMAddr]; op.active {
		if op.typ != 0 {
			modRM = d.decodeModRMSIBValue()
			out.put(op.opNum, d.decodeModRMSIBAddress(modRM, op.bySize, op.size))
		} else {
			out.put(op.opNum, d.decodeMoffsAddress(op))
		}
	}

	if op := d.decoder[x86SlotModRMReg]; op.active {
		if modRM[0] == -1 {
			modRM = d.decodeModRMSIBValue()
		}
		out.put(op.opNum, d.decodeRegValue(jsOr(d.regExtend, jsAnd(modRM[1], 0x07)), op.bySize, op.size))
	}
}

// decodeMoffsAddress renders a moffs operand: a bare address immediate rather
// than a ModR/M encoding.
func (d *x86Decoder) decodeMoffsAddress(op x86Operand) string {
	var pointer, addrSize float64
	if op.bySize {
		addrSize = jsShl(math.Pow(2, float64(d.bitMode)), 1)
		pointer = jsShl(d.getOperandSize(op.size), 1)
	} else {
		addrSize = float64(d.bitMode + 1)
		pointer = op.size
	}
	return x86At(d.tables.PTR, pointer) + d.segOverride +
		d.decodeImmediate(x86ImmPlain, op.bySize, addrSize) + "]"
}

// decodeImmediateOperands renders the up to three immediate operands, reporting
// whether any immediate byte was read.
func (d *x86Decoder) decodeImmediateOperands(out *x86OperandList) bool {
	immUsed := false

	if op := d.decoder[x86SlotImm1]; op.active {
		d.applyConditionCode(out, op.opNum, d.decodeImmediate(int(op.typ), op.bySize, op.size))
		immUsed = true
	}
	for _, slot := range []int{x86SlotImm1 + 1, x86SlotImm1 + 2} {
		if op := d.decoder[slot]; op.active {
			out.put(op.opNum, d.decodeImmediate(int(op.typ), op.bySize, op.size))
		}
	}
	return immUsed
}

// decodeVectorOperands renders the operands a vector extension adds: the extra
// vector register, and the register named by the immediate's upper nibble.
func (d *x86Decoder) decodeVectorOperands(out *x86OperandList, immUsed bool) {
	if op := d.decoder[x86SlotVectorReg]; op.active {
		out.put(op.opNum, d.decodeRegValue(d.vectorRegister, op.bySize, op.size))
	}

	if op := d.decoder[x86SlotImmReg]; op.active {
		if !immUsed {
			d.decodeImmediate(x86ImmPlain, false, 0) // forces an IMM8 for the register
		}
		value := jsOr(jsShr(jsAnd(d.immValue, 0xF0), 4), jsShl(jsAnd(d.immValue, 0x08), 1))
		out.put(op.opNum, d.decodeRegValue(value, op.bySize, op.size))
	}
}

// applyDestinationMask appends the EVEX mask and zero-merge suffixes, which
// belong to the destination operand.
func (d *x86Decoder) applyDestinationMask(out *x86OperandList) {
	if d.maskRegister != 0 {
		out.appendTo(0, "{K"+jsNum(d.maskRegister)+"}")
	}
	if d.extension == 2 && d.hintZeroMerg != 0 {
		out.appendTo(0, "{Z}")
	}
}

// applyConditionCode folds an immediate into the mnemonic for instructions
// whose name carries a condition code, marked in the tables by a trailing
// comma.
func (d *x86Decoder) applyConditionCode(out *x86OperandList, opNum int, imm string) {
	if !strings.HasSuffix(d.instruction, ",") {
		out.put(opNum, imm)
		return
	}

	parts := strings.Split(d.instruction, ",")
	if (d.extension >= 1 && d.extension <= 2 && d.opcode <= 0x400 && d.immValue < 0x20) || d.immValue < 0x08 {
		d.immValue = jsOr(d.immValue, jsShl(jsAnd(jsBool(d.opcode > 0x400), 1), 5))
		d.instruction = parts[0] + x86At(d.tables.ConditionCodes, d.immValue) + parts[1]
		return
	}
	d.instruction = parts[0] + parts[1]
	out.put(opNum, imm)
}

// x86ExplicitOperands are the fixed operands selected by type 7 and above.
var x86ExplicitOperands = []string{"ST", "FS", "GS", "1", "3", "XMM0", "M10"}

// decodeExplicitOperands renders the operands an instruction names implicitly.
func (d *x86Decoder) decodeExplicitOperands(out *x86OperandList) {
	for i := x86SlotExplicit; i < 11; i++ {
		op := d.decoder[i]
		if !op.active {
			return
		}

		switch {
		case op.typ <= 3:
			out.put(op.opNum, d.decodeRegValue(op.typ, op.bySize, op.size))

		case op.typ == 4:
			s := 3.0 // 32/64-bit ModR/M
			if (d.bitMode == x86Mode16 && !d.addressOverride) || (d.bitMode == x86Mode32 && d.addressOverride) {
				s = 7 // 16-bit ModR/M
			}
			out.put(op.opNum, d.decodeModRMSIBAddress([3]float64{0, 0, s}, op.bySize, op.size))

		case op.typ == 5 || op.typ == 6:
			s := 1.0
			if (d.bitMode == x86Mode16 && !d.addressOverride) ||
				jsAnd(jsBool(d.bitMode == x86Mode32), jsBool(d.addressOverride)) != 0 {
				s = -1
			}
			out.put(op.opNum, d.decodeModRMSIBAddress([3]float64{0, 0, op.typ + s}, op.bySize, op.size))

		case op.typ >= 7:
			out.put(op.opNum, x86At(x86ExplicitOperands, op.typ-7))
		}
	}
}

// errX86InvalidMode is CyberChef's verbatim error for an unknown bit mode.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errX86InvalidMode = errors.New("Invalid mode value")

// x86LookupOrInvalid returns list[i], reporting false where JavaScript would
// have found undefined or an empty entry.
func x86LookupOrInvalid(list []string, i float64) (string, bool) {
	if math.IsNaN(i) {
		return "", false
	}
	k := int(jsInt32(i))
	if k < 0 || k >= len(list) || list[k] == "" {
		return "", false
	}
	return list[k], true
}

// decodeInstruction decodes one instruction and returns its printed form.
// Ported from DecodeInstruction().
func (d *x86Decoder) decodeInstruction() string {
	d.reset()
	d.instructionPos = d.getPosition()
	d.decodePrefixAdjustments()

	if d.invalidOp == 0 {
		d.decodeOpcode()
		d.applyLarrabeeConditionCode()
		d.decodeOperandString()

		// Only SIMD instructions may carry a vector extension.
		if !d.vect && d.extension > 0 && d.opcode <= 0x400 {
			d.invalidOp = 1
		}
		// Under MVEX/EVEX the width bit must agree with the vector size.
		if d.vect && d.ignoresWidthbit == 0 && d.extension >= 2 {
			d.invalidOp = jsBool(jsAnd(d.simd, 1) != jsAnd(d.widthBit, 1))
		}
		if d.opcode >= 0x700 {
			d.widthBit = jsXor(d.widthBit, d.ignoresWidthbit)
		}
	}

	if d.invalidOp != 0 {
		return x86Invalid
	}

	d.decodeOperands()
	d.applyLateOpcodes()

	// 9A and EA print as segment:offset rather than two immediates.
	if d.opcode == 0x9A || d.opcode == 0xEA {
		if a, b, ok := strings.Cut(d.insOperands, ","); ok {
			d.insOperands = b + ":" + a
		}
	}

	d.applyPrefixFixups()
	d.enforceInstructionLimit()

	out := d.prefixG1 + " " + d.prefixG2 + " " + d.instruction + " " + d.insOperands
	out = strings.Trim(out, " ")

	if d.opcode >= 0x700 || d.roundMode != 0 {
		out += x86At(d.tables.RoundModes, d.roundMode)
	}
	return out
}

// applyLarrabeeConditionCode folds the L1OM condition code or high/low selector
// into the mnemonic.
func (d *x86Decoder) applyLarrabeeConditionCode() {
	if !(d.opcode >= 0x700 && strings.HasSuffix(d.instruction, ",")) {
		return
	}
	parts := strings.Split(d.instruction, ",")

	if d.opcode >= 0x720 && d.opcode <= 0x72F {
		d.immValue = jsShr(d.vectorRegister, 2)
		if d.float != 0 || (d.immValue != 3 && d.immValue != 7) {
			d.instruction = parts[0] + x86At(d.tables.ConditionCodes, d.immValue) + parts[1]
		} else {
			d.instruction = parts[0] + parts[1]
		}
		d.immValue = 0
		d.vectorRegister = jsAnd(d.vectorRegister, 0x03)
		return
	}

	mid := "L"
	if jsAnd(d.vectorRegister, 1) == 1 {
		mid = "H"
	}
	d.instruction = parts[0] + mid + parts[1]
}

// applyLateOpcodes resolves the two instruction families whose name comes from
// bytes read after the operands: 3DNow! and the synthetic VM opcodes.
func (d *x86Decoder) applyLateOpcodes() {
	if d.opcode == 0x10F {
		name, ok := x86LookupOrInvalid(d.tables.M3DNow, d.cur())
		d.nextByte()
		if !ok {
			d.instruction, d.insOperands = x86Invalid, ""
			return
		}
		d.instruction = name
		return
	}

	if d.instruction != "SSS" {
		return
	}
	code1 := d.cur()
	d.nextByte()
	code2 := d.cur()
	d.nextByte()

	if code1 >= 5 || code2 >= 5 {
		d.instruction = x86Invalid
		return
	}
	if name, ok := x86LookupOrInvalid(d.tables.MSynthetic, code1*5+code2); ok {
		d.instruction = name
	} else {
		d.instruction = x86Invalid
	}
}

// applyPrefixFixups rewrites the repeat and lock prefixes where the decoded
// instruction gives them a different meaning: HLE, MPX bounds, or a branch hint.
func (d *x86Decoder) applyPrefixFixups() {
	if d.prefixG1 == d.mnemonicStr(0xF3) && d.prefixG2 == d.mnemonicStr(0xF0) && d.xRelease != 0 {
		d.prefixG1 = "XRELEASE"
	}
	if d.prefixG1 == d.mnemonicStr(0xF2) && d.prefixG2 == d.mnemonicStr(0xF0) && d.xAcquire != 0 {
		d.prefixG1 = "XACQUIRE"
	}
	if (d.prefixG1 == "XRELEASE" || d.prefixG1 == "XACQUIRE") && d.hleFlipG1G2 {
		d.prefixG1, d.prefixG2 = d.prefixG2, d.prefixG1
	}

	if d.ht != 0 {
		if d.segOverride == d.mnemonicStr(0x2E) {
			d.prefixG1 = "HNT"
		} else if d.segOverride == d.mnemonicStr(0x3E) {
			d.prefixG1 = "HT"
		}
	}

	if d.prefixG1 == d.mnemonicStr(0xF2) && d.bnd != 0 {
		d.prefixG1 = "BND"
	}
}

// enforceInstructionLimit rejects anything longer than the 15 bytes an x86
// decoder can consume, rewinding to just after the truncated instruction.
func (d *x86Decoder) enforceInstructionLimit() {
	if len(d.instructionHex) <= x86MaxInstructionHex {
		return
	}

	over := float64((len(d.instructionHex) - x86MaxInstructionHex) >> 1)
	d.instructionHex = d.instructionHex[:x86MaxInstructionHex]

	dif32 := d.pos32 - over
	if dif32 < 0 {
		dif32 += 0x100000000
	}
	s32 := jsPadLeft(jsHex16(dif32), "0", 8)
	s64 := jsPadLeft(jsHex16(d.pos64), "0", 8)
	d.gotoPosition(s64 + s32)

	d.prefixG1, d.prefixG2 = "", ""
	d.instruction, d.insOperands = x86Invalid, ""
}

// disassemble decodes the loaded bytes in one linear pass.
// Ported from LDisassemble().
func (d *x86Decoder) disassemble() string {
	var out strings.Builder
	basePos64, basePos32 := d.pos64, d.pos32

	for d.codePos < len(d.binCode) {
		instruction := d.decodeInstruction()

		if d.showPos {
			out.WriteString(d.instructionPos + " ")
		}
		if d.showHex {
			hex := strings.ToUpper(d.instructionHex)
			for len(hex) < x86HexColumn {
				hex += " "
			}
			out.WriteString(hex)
		}
		out.WriteString(instruction + "\r\n")

		d.instructionPos = ""
		d.instructionHex = ""
	}

	d.codePos = 0
	d.pos32, d.pos64 = basePos32, basePos64
	return out.String()
}
