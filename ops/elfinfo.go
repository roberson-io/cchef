package ops

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ELFInfo{})
}

// elfLabelWidth is the column every label is padded to before its value.
const elfLabelWidth = 30

// elfMagic begins every ELF file: DEL, then "ELF".
var elfMagic = []byte{0x7f, 0x45, 0x4c, 0x46}

// ELF formats, as named by the fifth byte of the identification field.
const (
	elf32 = 1
	elf64 = 2
)

// ELFInfo reports the header, program headers, section headers and symbol table
// of an ELF file. Ported from CyberChef ELFInfo.mjs.
type ELFInfo struct{}

// Meta returns the operation metadata.
func (ELFInfo) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ELF Info",
		Module:      "Default",
		Description: "Implements readelf-like functionality. This operation will extract the ELF Header, Program Headers, Section Headers and Symbol Table for an ELF file.",
		InfoURL:     "https://wikipedia.org/wiki/Executable_and_Linkable_Format",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns no arguments; the operation takes none.
func (ELFInfo) Args() []core.ArgDef { return nil }

// Run reports what the ELF file describes.
func (ELFInfo) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := elfReport(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// elfCursor walks a file a field at a time, refusing to read or seek outside it.
type elfCursor struct {
	data []byte
	pos  uint64
}

// seek moves to an absolute position. A position past the end is refused, in the
// same words the upstream operation uses.
func (c *elfCursor) seek(pos uint64) error {
	if pos > uint64(len(c.data)) {
		return fmt.Errorf("Cannot move to position %d in stream. Out of bounds.", pos) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	c.pos = pos
	return nil
}

// skip moves forwards by n bytes.
func (c *elfCursor) skip(n uint64) error {
	return c.seek(elfAdd(c.pos, n))
}

// readInt reads an n-byte unsigned integer and steps over it.
func (c *elfCursor) readInt(n uint64, little bool) (uint64, error) {
	if elfAdd(c.pos, n) > uint64(len(c.data)) {
		return 0, fmt.Errorf("Cannot read %d bytes at position %d in stream. Out of bounds.", n, c.pos) //nolint:staticcheck,revive // matches the seek wording above
	}
	field := c.data[c.pos : c.pos+n]
	var val uint64
	if little {
		for i := len(field) - 1; i >= 0; i-- {
			val = val<<8 | uint64(field[i])
		}
	} else {
		for _, b := range field {
			val = val<<8 | uint64(b)
		}
	}
	c.pos += n
	return val, nil
}

// readString reads the string at base+offset, stopping at a zero byte or the end
// of the file, and leaves the position where it found it.
func (c *elfCursor) readString(base, offset uint64) (string, error) {
	here := c.pos
	if err := c.seek(elfAdd(base, offset)); err != nil {
		return "", err
	}
	rest := c.data[c.pos:]
	if end := strings.IndexByte(string(rest), 0); end >= 0 {
		rest = rest[:end]
	}
	c.pos = here
	return string(rest), nil
}

// elfAdd adds two file offsets, saturating rather than wrapping so that a
// nonsensical header cannot fold back around into the file.
func elfAdd(a, b uint64) uint64 {
	if sum := a + b; sum >= a {
		return sum
	}
	return math.MaxUint64
}

// elfFile holds what the header says about the rest of the file.
type elfFile struct {
	cur    *elfCursor
	format uint64
	little bool

	entry, phoff, shoff             uint64
	phEntries, shEntries, shentSize uint64
	shstrtab, namesOffset           uint64

	symtabOffset, symtabSize, symtabEntSize uint64
	strtabOffset                            uint64
}

// wordSize is the width of an address in this file: four bytes or eight.
func (f *elfFile) wordSize() uint64 {
	if f.format == elf32 {
		return 4
	}
	return 8
}

// read is readInt with this file's endianness already applied.
func (f *elfFile) read(n uint64) (uint64, error) {
	return f.cur.readInt(n, f.little)
}

// elfRow lays a label out in its column and appends the value.
func elfRow(label, value string) string {
	return padEndSpace(label, elfLabelWidth) + value
}

// elfBanner is the rule CyberChef draws above each part of the report.
func elfBanner(title string) string {
	rule := strings.Repeat("=", elfLabelWidth)
	return rule + " " + title + " " + rule
}

// elfReport is the whole operation: parse the file and lay out what it says.
func elfReport(data []byte) (string, error) {
	f := &elfFile{cur: &elfCursor{data: data}}

	out := []string{elfBanner("ELF Header")}
	header, err := f.header()
	if err != nil {
		return "", err
	}
	out = append(out, strings.Join(header, "\n")+"\n")

	if err := f.findSectionNames(); err != nil {
		return "", err
	}

	segments, err := f.segments()
	if err != nil {
		return "", err
	}
	out = append(out, segments...)

	sections, err := f.sections()
	if err != nil {
		return "", err
	}
	out = append(out, sections...)

	symbols, err := f.symbols()
	if err != nil {
		return "", err
	}
	out = append(out, symbols...)

	return strings.Join(out, "\n"), nil
}

// header reads the identification field and the file-wide values that follow it.
func (f *elfFile) header() ([]string, error) {
	magic := f.cur.data
	if len(magic) < len(elfMagic) || string(magic[:len(elfMagic)]) != string(elfMagic) {
		return nil, errors.New("Invalid ELF") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	f.cur.pos = uint64(len(elfMagic))
	rows := []string{elfRow("Magic:", string(elfMagic))}

	ident, err := f.identity()
	if err != nil {
		return nil, err
	}
	rows = append(rows, ident...)

	kinds, err := f.kinds()
	if err != nil {
		return nil, err
	}
	rows = append(rows, kinds...)

	places, err := f.places()
	if err != nil {
		return nil, err
	}
	return append(rows, places...), nil
}

// identity reads the format, endianness, version and ABI, then steps over the
// seven bytes of padding that close the identification field.
func (f *elfFile) identity() ([]string, error) {
	format, err := f.cur.readInt(1, false)
	if err != nil {
		return nil, err
	}
	f.format = format
	width := "64-bit"
	if format == elf32 {
		width = "32-bit"
	}

	order, err := f.cur.readInt(1, false)
	if err != nil {
		return nil, err
	}
	f.little = order == 1
	endianness := "Big"
	if f.little {
		endianness = "Little"
	}

	version, err := f.cur.readInt(1, false)
	if err != nil {
		return nil, err
	}

	osabi, err := f.cur.readInt(1, false)
	if err != nil {
		return nil, err
	}
	abi := elfABIs[osabi]

	abiVersion, err := f.cur.readInt(1, false)
	if err != nil {
		return nil, err
	}

	rows := []string{
		elfRow("Format:", width),
		elfRow("Endianness:", endianness),
		elfRow("Version:", fmt.Sprint(version)),
		elfRow("ABI:", abi),
	}
	// Linux does not use the ABI version byte, so it is not reported.
	if abi != "Linux" {
		rows = append(rows, elfRow("ABI Version:", fmt.Sprint(abiVersion)))
	}
	return rows, f.cur.skip(7)
}

// kinds reads the object file type, the instruction set and the ELF version.
func (f *elfFile) kinds() ([]string, error) {
	objType, err := f.read(2)
	if err != nil {
		return nil, err
	}
	machine, err := f.read(2)
	if err != nil {
		return nil, err
	}
	isa, named := elfMachines[machine]
	if !named {
		isa = "Unimplemented"
	}
	version, err := f.read(4)
	if err != nil {
		return nil, err
	}
	return []string{
		elfRow("Type:", elfObjectTypes[objType]),
		elfRow("Instruction Set Architecture:", isa),
		elfRow("ELF Version:", fmt.Sprint(version)),
	}, nil
}

// places reads the entry point, the two table offsets and the sizes and counts
// that describe those tables.
func (f *elfFile) places() ([]string, error) {
	word := f.wordSize()
	for _, field := range []*uint64{&f.entry, &f.phoff, &f.shoff} {
		val, err := f.read(word)
		if err != nil {
			return nil, err
		}
		*field = val
	}
	flags, err := f.read(4)
	if err != nil {
		return nil, err
	}
	rows := []string{
		elfRow("Entry Point:", fmt.Sprintf("0x%02x", f.entry)),
		elfRow("Entry PHOFF:", fmt.Sprintf("0x%02x", f.phoff)),
		elfRow("Entry SHOFF:", fmt.Sprintf("0x%02x", f.shoff)),
		elfRow("Flags:", fmt.Sprintf("%08b", flags)),
	}

	var phentSize uint64
	sizes := []struct {
		label string
		field *uint64
		unit  string
	}{
		{"ELF Header Size:", new(uint64), " bytes"},
		{"Program Header Size:", &phentSize, " bytes"},
		{"Program Header Entries:", &f.phEntries, ""},
		{"Section Header Size:", &f.shentSize, " bytes"},
		{"Section Header Entries:", &f.shEntries, ""},
		{"Section Header Names:", &f.shstrtab, ""},
	}
	for _, s := range sizes {
		val, err := f.read(2)
		if err != nil {
			return nil, err
		}
		*s.field = val
		rows = append(rows, elfRow(s.label, fmt.Sprint(val)+s.unit))
	}
	return rows, nil
}

// findSectionNames reads the offset of the section holding the section names,
// which is the one the header points at, and returns to where it started.
func (f *elfFile) findSectionNames() error {
	here := f.cur.pos
	if err := f.cur.seek(elfAdd(f.shoff, f.shentSize*f.shstrtab)); err != nil {
		return err
	}
	// The offset field sits past the name, type, flags and address.
	skip, word := uint64(0x18), uint64(8)
	if f.format == elf32 {
		skip, word = 0x10, 4
	}
	if err := f.cur.skip(skip); err != nil {
		return err
	}
	names, err := f.read(word)
	if err != nil {
		return err
	}
	f.namesOffset = names
	f.cur.pos = here
	return nil
}

// segments reports every program header.
func (f *elfFile) segments() ([]string, error) {
	out := []string{elfBanner("Program Header")}
	if err := f.cur.seek(f.phoff); err != nil {
		return nil, err
	}
	for range f.phEntries {
		rows, err := f.segment()
		if err != nil {
			return nil, err
		}
		out = append(out, strings.Join(rows, "\n")+"\n")
	}
	return out, nil
}

// segment reads one program header: what the segment is for, where it sits in
// the file and in memory, and how it may be used.
func (f *elfFile) segment() ([]string, error) {
	segType, err := f.read(4)
	if err != nil {
		return nil, err
	}
	rows := []string{elfRow("Program Header Type:", elfSegmentType(segType))}

	// The flags word swapped places with the offsets when ELF grew to 64 bits.
	if f.format != elf32 {
		flags, err := f.read(4)
		if err != nil {
			return nil, err
		}
		rows = append(rows, elfRow("Flags:", elfSegmentFlags(flags)))
	}

	word := f.wordSize()
	for _, field := range []struct{ label, unit string }{
		{"Offset Of Segment:", ""},
		{"Virtual Address of Segment:", ""},
		{"Physical Address of Segment:", ""},
		{"Size of Segment:", " bytes"},
		{"Size of Segment in Memory:", " bytes"},
	} {
		val, err := f.read(word)
		if err != nil {
			return nil, err
		}
		rows = append(rows, elfRow(field.label, fmt.Sprint(val)+field.unit))
	}

	if f.format == elf32 {
		flags, err := f.read(4)
		if err != nil {
			return nil, err
		}
		rows = append(rows, elfRow("Flags:", elfSegmentFlags(flags)))
	}
	// Step over the alignment, which is not reported.
	return rows, f.cur.skip(word)
}

// sections reports every section header.
func (f *elfFile) sections() ([]string, error) {
	out := []string{elfBanner("Section Header")}
	// findSectionNames has already sought at least this far, so the table
	// offset is known to be within the file.
	f.cur.pos = f.shoff
	for range f.shEntries {
		rows, err := f.section()
		if err != nil {
			return nil, err
		}
		out = append(out, strings.Join(rows, "\n")+"\n")
	}
	return out, nil
}

// section reads one section header. A section of no known type is not named,
// and the two tables the symbol list needs are noted as they go past.
func (f *elfFile) section() ([]string, error) {
	nameOffset, err := f.read(4)
	if err != nil {
		return nil, err
	}
	secType, err := f.read(4)
	if err != nil {
		return nil, err
	}
	kind := elfSectionType(secType)
	rows := []string{elfRow("Type:", kind)}

	name := ""
	if kind != "Unused" {
		if name, err = f.cur.readString(f.namesOffset, nameOffset); err != nil {
			return nil, err
		}
		rows = append(rows, elfRow("Section Name: ", name))
	}

	word := f.wordSize()
	flags, err := f.read(word)
	if err != nil {
		return nil, err
	}
	rows = append(rows, elfRow("Flags:", elfSectionFlags(flags)))

	var vaddr, offset, size uint64
	for _, field := range []struct {
		label string
		into  *uint64
	}{
		{"Section Vaddr in memory:", &vaddr},
		{"Offset of the section:", &offset},
		{"Section Size:", &size},
	} {
		val, err := f.read(word)
		if err != nil {
			return nil, err
		}
		*field.into = val
		rows = append(rows, elfRow(field.label, fmt.Sprint(val)))
	}

	for _, label := range []string{"Associated Section:", "Section Extra Information:"} {
		val, err := f.read(4)
		if err != nil {
			return nil, err
		}
		rows = append(rows, elfRow(label, fmt.Sprint(val)))
	}

	// Step over the alignment, which is not reported, to reach the entry size.
	if err := f.cur.skip(word); err != nil {
		return nil, err
	}
	entSize, err := f.read(word)
	if err != nil {
		return nil, err
	}
	switch name {
	case ".strtab":
		f.strtabOffset = offset
	case ".symtab":
		f.symtabOffset, f.symtabSize, f.symtabEntSize = offset, size, entSize
	}
	return rows, nil
}

// symbols reports the name of every symbol in the symbol table that has one.
func (f *elfFile) symbols() ([]string, error) {
	out := []string{elfBanner("Symbol Table")}
	if err := f.cur.seek(f.symtabOffset); err != nil {
		return nil, err
	}
	// The upstream count is a plain division, so a table size that is not a
	// whole number of entries reaches one entry too far, and an entry size of
	// zero reads on until the file runs out. Both are kept: they are how a
	// malformed table is refused rather than half-reported.
	count := float64(f.symtabSize) / float64(f.symtabEntSize)
	for i := 0; float64(i) < count; i++ {
		name, err := f.symbolName()
		if err != nil {
			return nil, err
		}
		if name != "" {
			out = append(out, elfRow("Symbol Name:", name))
		}
	}
	return out, nil
}

// symbolName reads one symbol table entry and returns the name it points at.
func (f *elfFile) symbolName() (string, error) {
	nameOffset, err := f.read(4)
	if err != nil {
		return "", err
	}
	// The rest of the entry is not reported; its width differs by format.
	rest := uint64(20)
	if f.format == elf32 {
		rest = 12
	}
	if err := f.cur.skip(rest); err != nil {
		return "", err
	}
	return f.cur.readString(f.strtabOffset, nameOffset)
}

// elfABIs names the target OS ABI. A number not listed here has no name, and
// the ABI row is left blank, as upstream leaves it.
var elfABIs = map[uint64]string{
	0x00: "System V",
	0x01: "HP-UX",
	0x02: "NetBSD",
	0x03: "Linux",
	0x04: "GNU Hurd",
	0x06: "Solaris",
	0x07: "AIX",
	0x08: "IRIX",
	0x09: "FreeBSD",
	0x0A: "Tru64",
	0x0B: "Novell Modesto",
	0x0C: "OpenBSD",
	0x0D: "OpenVMS",
	0x0E: "NonStop Kernel",
	0x0F: "AROS",
	0x10: "Fenix OS",
	0x11: "CloudABI",
	0x12: "Stratus Technologies OpenVOS",
}

// elfObjectTypes names what kind of object file this is.
var elfObjectTypes = map[uint64]string{
	0x0000: "Unknown",
	0x0001: "Relocatable File",
	0x0002: "Executable File",
	0x0003: "Shared Object",
	0x0004: "Core File",
	0xFE00: "LOOS",
	0xFEFF: "HIOS",
	0xFF00: "LOPROC",
	0xFFFF: "HIPROC",
}

// elfMachines names the instruction set architecture.
var elfMachines = map[uint64]string{
	0x0000: "No specific instruction set",
	0x0001: "AT&T WE 32100",
	0x0002: "SPARC",
	0x0003: "x86",
	0x0004: "Motorola 68000 (M68k)",
	0x0005: "Motorola 88000 (M88k)",
	0x0006: "Intel MCU",
	0x0007: "Intel 80860",
	0x0008: "MIPS",
	0x0009: "IBM System/370",
	0x000A: "MIPS RS3000 Little-endian",
	0x000B: "Reserved for future use",
	0x000C: "Reserved for future use",
	0x000D: "Reserved for future use",
	0x000E: "Reserved for future use",
	0x000F: "Hewlett-Packard PA-RISC",
	0x0011: "Fujitsu VPP500",
	0x0012: "Enhanced instruction set SPARC",
	0x0013: "Intel 80960",
	0x0014: "PowerPC",
	0x0015: "PowerPC (64-bit)",
	0x0016: "S390, including S390",
	0x0017: "IBM SPU/SPC",
	0x0018: "Reserved for future use",
	0x0019: "Reserved for future use",
	0x001A: "Reserved for future use",
	0x001B: "Reserved for future use",
	0x001C: "Reserved for future use",
	0x001D: "Reserved for future use",
	0x001E: "Reserved for future use",
	0x001F: "Reserved for future use",
	0x0020: "Reserved for future use",
	0x0021: "Reserved for future use",
	0x0022: "Reserved for future use",
	0x0023: "Reserved for future use",
	0x0024: "NEC V800",
	0x0025: "Fujitsu FR20",
	0x0026: "TRW RH-32",
	0x0027: "Motorola RCE",
	0x0028: "ARM (up to ARMv7/Aarch32)",
	0x0029: "Digital Alpha",
	0x002A: "SuperH",
	0x002B: "SPARC Version 9",
	0x002C: "Siemens TriCore embedded processor",
	0x002D: "Argonaut RISC Core",
	0x002E: "Hitachi H8/300",
	0x002F: "Hitachi H8/300H",
	0x0030: "Hitachi H8S",
	0x0031: "Hitachi H8/500",
	0x0032: "IA-64",
	0x0033: "Standford MIPS-X",
	0x0034: "Motorola ColdFire",
	0x0035: "Motorola M68HC12",
	0x0036: "Fujitsu MMA Multimedia Accelerator",
	0x0037: "Siemens PCP",
	0x0038: "Sony nCPU embedded RISC processor",
	0x0039: "Denso NDR1 microprocessor",
	0x003A: "Motorola Star*Core processor",
	0x003B: "Toyota ME16 processor",
	0x003C: "STMicroelectronics ST100 processor",
	0x003D: "Advanced Logic Corp. TinyJ embedded processor family",
	0x003E: "AMD x86-64",
	0x003F: "Sony DSP Processor",
	0x0040: "Digital Equipment Corp. PDP-10",
	0x0041: "Digital Equipment Corp. PDP-11",
	0x0042: "Siemens FX66 microcontroller",
	0x0043: "STMicroelectronics ST9+ 8/16 bit microcontroller",
	0x0044: "STMicroelectronics ST7 8-bit microcontroller",
	0x0045: "Motorola MC68HC16 Microcontroller",
	0x0046: "Motorola MC68HC11 Microcontroller",
	0x0047: "Motorola MC68HC08 Microcontroller",
	0x0048: "Motorola MC68HC05 Microcontroller",
	0x0049: "Silicon Graphics SVx",
	0x004A: "STMicroelectronics ST19 8-bit microcontroller",
	0x004B: "Digital VAX",
	0x004C: "Axis Communications 32-bit embedded processor",
	0x004D: "Infineon Technologies 32-bit embedded processor",
	0x004E: "Element 14 64-bit DSP Processor",
	0x004F: "LSI Logic 16-bit DSP Processor",
	0x0050: "Donald Knuth's educational 64-bit processor",
	0x0051: "Harvard University machine-independent object files",
	0x0052: "SiTera Prism",
	0x0053: "Atmel AVR 8-bit microcontroller",
	0x0054: "Fujitsu FR30",
	0x0055: "Mitsubishi D10V",
	0x0056: "Mitsubishi D30V",
	0x0057: "NEC v850",
	0x0058: "Mitsubishi M32R",
	0x0059: "Matsushita MN10300",
	0x005A: "Matsushita MN10200",
	0x005B: "picoJava",
	0x005C: "OpenRISC 32-bit embedded processor",
	0x005D: "ARC Cores Tangent-A5",
	0x005E: "Tensilica Xtensa Architecture",
	0x005F: "Alphamosaic VideoCore processor",
	0x0060: "Thompson Multimedia General Purpose Processor",
	0x0061: "National Semiconductor 32000 series",
	0x0062: "Tenor Network TPC processor",
	0x0063: "Trebia SNP 1000 processor",
	0x0064: "STMicroelectronics (www.st.com) ST200 microcontroller",
	0x008C: "TMS320C6000 Family",
	0x00AF: "MCST Elbrus e2k",
	0x00B7: "ARM 64-bits (ARMv8/Aarch64)",
	0x00F3: "RISC-V",
	0x00F7: "Berkeley Packet Filter",
	0x0101: "WDC 65C816",
}

// elfSegmentTypes names the defined segment types; the reserved ranges are
// handled in elfSegmentType.
var elfSegmentTypes = map[uint64]string{
	0x00000000: "Unused",
	0x00000001: "Loadable Segment",
	0x00000002: "Dynamic linking information",
	0x00000003: "Interpreter Information",
	0x00000004: "Auxiliary Information",
	0x00000005: "Reserved",
	0x00000006: "Program Header Table",
	0x00000007: "Thread-Local Storage Template",
}

// elfSectionTypes names the defined section types; the reserved ranges and the
// "Unused" fallback are handled in elfSectionType.
var elfSectionTypes = map[uint64]string{
	0x00000001: "Program Data",
	0x00000002: "Symbol Table",
	0x00000003: "String Table",
	0x00000004: "Relocation Entries with Addens",
	0x00000005: "Symbol Hash Table",
	0x00000006: "Dynamic Linking Information",
	0x00000007: "Notes",
	0x00000008: "Program Space with No Data",
	0x00000009: "Relocation Entries with no Addens",
	0x0000000A: "Reserved",
	0x0000000B: "Dynamic Linker Symbol Table",
	0x0000000E: "Array of Constructors",
	0x0000000F: "Array of Destructors",
	0x00000010: "Array of pre-constructors",
	0x00000011: "Section group",
	0x00000012: "Extended section indices",
	0x00000013: "Number of defined types",
}

// elfSegmentType names what a segment is for.
func elfSegmentType(t uint64) string {
	if name, ok := elfSegmentTypes[t]; ok {
		return name
	}
	switch {
	case t >= 0x60000000 && t <= 0x6FFFFFFF:
		return "Reserved Inclusive Range. OS Specific"
	case t >= 0x70000000 && t <= 0x7FFFFFFF:
		return "Reserved Inclusive Range. Processor Specific"
	}
	return ""
}

// elfSegmentFlags names how a segment may be used.
func elfSegmentFlags(flags uint64) string {
	var set []string
	for _, bit := range []struct {
		mask uint64
		name string
	}{
		{0x1, "Execute"},
		{0x2, "Write"},
		{0x4, "Read"},
		{0xf0000000, "Unspecified"},
	} {
		if flags&bit.mask != 0 {
			set = append(set, bit.name)
		}
	}
	return strings.Join(set, ",")
}

// elfSectionType names what a section holds.
func elfSectionType(t uint64) string {
	if name, ok := elfSectionTypes[t]; ok {
		return name
	}
	switch {
	case t >= 0x60000000 && t <= 0x6fffffff:
		return "OS-specific"
	case t >= 0x70000000 && t <= 0x7fffffff:
		return "Processor-specific"
	case t >= 0x80000000 && t <= 0x8fffffff:
		return "Application-specific"
	}
	return "Unused"
}

// elfSectionFlags names what may be done with a section. Only the low half of
// the word is described, so a wider flags field says no more than a narrow one.
func elfSectionFlags(flags uint64) string {
	var set []string
	for _, bit := range []struct {
		mask uint64
		name string
	}{
		{0x00000001, "Writable"},
		{0x00000002, "Alloc"},
		{0x00000004, "Executable"},
		{0x00000010, "Merge"},
		{0x00000020, "Strings"},
		{0x00000040, "SHT Info Link"},
		{0x00000080, "Link Order"},
		{0x00000100, "OS Specific Handling"},
		{0x00000200, "Group"},
		{0x00000400, "Thread Local Data"},
		{0x0FF00000, "OS-Specific"},
		{0xF0000000, "Processor Specific"},
		{0x04000000, "Special Ordering (Solaris)"},
		{0x08000000, "Excluded (Solaris)"},
	} {
		if flags&bit.mask != 0 {
			set = append(set, bit.name)
		}
	}
	return strings.Join(set, ",")
}
