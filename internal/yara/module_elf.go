package yara

import "encoding/binary"

// The elf module, which reads the shape of an executable built for Unix.
//
// An ELF file opens with a header naming its kind, the machine it was built
// for, and where to find two tables: sections, which say how the file is laid
// out on disk, and segments, which say how it is laid out once loaded. The
// sections in turn hold the symbols the file defines and the libraries it
// depends on.

// The numbers the elf module offers a rule to compare against. The names are
// YARA's own, so they are spelled as YARA spells them.
//
//nolint:misspell // RELA is a relocation with an addend, not a misspelling
var elfConstants = map[string]int64{
	"ET_NONE": 0, "ET_REL": 1, "ET_EXEC": 2, "ET_DYN": 3, "ET_CORE": 4,

	"EM_NONE": 0, "EM_M32": 1, "EM_SPARC": 2, "EM_386": 3, "EM_68K": 4,
	"EM_88K": 5, "EM_860": 7, "EM_MIPS": 8, "EM_MIPS_RS3_LE": 0x0A,
	"EM_PPC": 0x14, "EM_PPC64": 0x15, "EM_ARM": 0x28, "EM_X86_64": 0x3E,
	"EM_AARCH64": 0xB7,

	"SHT_NULL": 0, "SHT_PROGBITS": 1, "SHT_SYMTAB": 2, "SHT_STRTAB": 3,
	"SHT_RELA": 4, "SHT_HASH": 5, "SHT_DYNAMIC": 6, "SHT_NOTE": 7,
	"SHT_NOBITS": 8, "SHT_REL": 9, "SHT_SHLIB": 10, "SHT_DYNSYM": 11,

	"SHF_WRITE": 0x1, "SHF_ALLOC": 0x2, "SHF_EXECINSTR": 0x4,

	"PT_NULL": 0, "PT_LOAD": 1, "PT_DYNAMIC": 2, "PT_INTERP": 3, "PT_NOTE": 4,
	"PT_SHLIB": 5, "PT_PHDR": 6, "PT_TLS": 7,
	"PT_GNU_EH_FRAME": 0x6474e550, "PT_GNU_STACK": 0x6474e551,

	"DT_NULL": 0, "DT_NEEDED": 1, "DT_PLTRELSZ": 2, "DT_PLTGOT": 3, "DT_HASH": 4,
	"DT_STRTAB": 5, "DT_SYMTAB": 6, "DT_RELA": 7, "DT_RELASZ": 8, "DT_RELAENT": 9,
	"DT_STRSZ": 10, "DT_SYMENT": 11, "DT_INIT": 12, "DT_FINI": 13, "DT_SONAME": 14,
	"DT_RPATH": 15, "DT_SYMBOLIC": 16, "DT_REL": 17, "DT_RELSZ": 18, "DT_RELENT": 19,
	"DT_PLTREL": 20, "DT_DEBUG": 21, "DT_TEXTREL": 22, "DT_JMPREL": 23,
	"DT_BIND_NOW": 24, "DT_INIT_ARRAY": 25, "DT_FINI_ARRAY": 26,
	"DT_INIT_ARRAYSZ": 27, "DT_FINI_ARRAYSZ": 28, "DT_RUNPATH": 29, "DT_FLAGS": 30,
	"DT_ENCODING": 32,

	"STT_NOTYPE": 0, "STT_OBJECT": 1, "STT_FUNC": 2, "STT_SECTION": 3,
	"STT_FILE": 4, "STT_COMMON": 5, "STT_TLS": 6,

	"STB_LOCAL": 0, "STB_GLOBAL": 1, "STB_WEAK": 2,

	"PF_X": 0x1, "PF_W": 0x2, "PF_R": 0x4,
}

// The sizes and limits the format fixes.
const (
	// elfIdentSize is the run of bytes an ELF file opens with, which says how
	// the rest of it is written before any of it can be read.
	elfIdentSize = 16
	// elfClassOffset says whether the file is written with 32-bit or 64-bit
	// numbers, and elfDataOffset which end of each number comes first.
	elfClassOffset = 4
	elfDataOffset  = 5
	elfClass32     = 1
	elfClass64     = 2
	elfDataLittle  = 1
	elfDataBig     = 2
	// shnLoReserve is where section numbers stop counting sections and start
	// standing for something else, and pnXNum likewise for segments.
	shnLoReserve = 0xFF00
	pnXNum       = 0xFFFF
	// symTypeBits is how much of a symbol's information byte says what kind of
	// thing it names; the rest says how widely it is visible.
	symTypeBits = 4
	symTypeMask = 0x0F
)

var elfMagic = []byte{0x7F, 'E', 'L', 'F'}

// elfConst is one of the module's own numbers, read as the file's own fields
// are. They are all small and positive, being kinds and flags the format
// defines.
func elfConst(name string) uint64 {
	return uint64(elfConstants[name]) // #nosec G115 -- the module's own constants are small and positive
}

// elfSymbolDecl is what one symbol offers, which the two symbol tables share.
func elfSymbolDecl() *modDecl {
	return decArray(decStruct(map[string]*modDecl{
		"name": decString(), "value": decInt(), "size": decInt(),
		"type": decInt(), "bind": decInt(), "shndx": decInt(),
	}))
}

func elfSchema() *modDecl {
	members := map[string]*modDecl{
		"type": decInt(), "machine": decInt(), "entry_point": decInt(),
		"number_of_sections": decInt(), "sh_offset": decInt(), "sh_entry_size": decInt(),
		"number_of_segments": decInt(), "ph_offset": decInt(), "ph_entry_size": decInt(),
		"sections": decArray(decStruct(map[string]*modDecl{
			"type": decInt(), "flags": decInt(), "address": decInt(),
			"name": decString(), "size": decInt(), "offset": decInt(),
		})),
		"segments": decArray(decStruct(map[string]*modDecl{
			"type": decInt(), "flags": decInt(), "offset": decInt(),
			"virtual_address": decInt(), "physical_address": decInt(),
			"file_size": decInt(), "memory_size": decInt(), "alignment": decInt(),
		})),
		"dynamic_section_entries": decInt(),
		"dynamic": decArray(decStruct(map[string]*modDecl{
			"type": decInt(), "val": decInt(),
		})),
		"symtab_entries": decInt(), "symtab": elfSymbolDecl(),
		"dynsym_entries": decInt(), "dynsym": elfSymbolDecl(),
	}
	for name := range elfConstants {
		members[name] = decInt()
	}
	return decStruct(members)
}

// elfFile is an ELF file being read: the bytes, and how the numbers in them are
// written.
type elfFile struct {
	data  []byte
	wide  bool
	order binary.ByteOrder
}

// elfModule reads the data as an ELF file. Everything but the constants is left
// out when the data is not one, so a rule asking about it comes to nothing.
func elfModule(e *evaluator) modValue {
	fields := make(map[string]modValue, len(elfConstants)+20)
	for name, n := range elfConstants {
		fields[name] = valueOf(intValue(n))
	}

	f, ok := readELF(e.buf.data)
	if !ok {
		return structOf(fields)
	}
	f.addHeader(fields)
	sections := f.sections()
	f.addSections(fields, sections)
	f.addSegments(fields)
	f.addSymbols(fields, sections)
	return structOf(fields)
}

// readELF checks the opening bytes and works out how the rest is written.
func readELF(data []byte) (elfFile, bool) {
	if len(data) < elfIdentSize || string(data[:len(elfMagic)]) != string(elfMagic) {
		return elfFile{}, false
	}
	f := elfFile{data: data}
	switch data[elfClassOffset] {
	case elfClass32:
	case elfClass64:
		f.wide = true
	default:
		return elfFile{}, false
	}
	switch data[elfDataOffset] {
	case elfDataLittle:
		f.order = binary.LittleEndian
	case elfDataBig:
		f.order = binary.BigEndian
	default:
		return elfFile{}, false
	}
	return f, len(data) >= f.headerSize()
}

// headerSize is how long the opening header is, which differs between the two
// widths.
func (f elfFile) headerSize() int {
	if f.wide {
		return 64
	}
	return 52
}

// The three ways of reading a number out of the file, each coming to nothing
// when the bytes asked for are not there.
func (f elfFile) u16(at int) (uint64, bool) {
	if at < 0 || at+2 > len(f.data) {
		return 0, false
	}
	return uint64(f.order.Uint16(f.data[at:])), true
}

func (f elfFile) u32(at int) (uint64, bool) {
	if at < 0 || at+4 > len(f.data) {
		return 0, false
	}
	return uint64(f.order.Uint32(f.data[at:])), true
}

func (f elfFile) u64(at int) (uint64, bool) {
	if at < 0 || at+8 > len(f.data) {
		return 0, false
	}
	return f.order.Uint64(f.data[at:]), true
}

// word reads a number whose width follows the file's own: four bytes in a
// 32-bit file, eight in a 64-bit one.
func (f elfFile) word(at int) (uint64, bool) {
	if f.wide {
		return f.u64(at)
	}
	return f.u32(at)
}

// Where each field of the opening header sits. The two widths agree as far as
// the entry point and part company after it.
func (f elfFile) headerFields() (entry, phOffset, shOffset, counts int) {
	if f.wide {
		return 24, 32, 40, 52
	}
	return 24, 28, 32, 40
}

// addHeader reads what the opening header says about the file as a whole.
func (f elfFile) addHeader(fields map[string]modValue) {
	entryAt, phAt, shAt, countsAt := f.headerFields()
	f.put(fields, "type", 16, f.u16)
	f.put(fields, "machine", 18, f.u16)
	f.put(fields, "ph_offset", phAt, f.word)
	f.put(fields, "sh_offset", shAt, f.word)
	// After the header's own size come the program header's size and count,
	// then the section header's size and count.
	f.put(fields, "ph_entry_size", countsAt+2, f.u16)
	f.put(fields, "number_of_segments", countsAt+4, f.u16)
	f.put(fields, "sh_entry_size", countsAt+6, f.u16)
	f.put(fields, "number_of_sections", countsAt+8, f.u16)

	// Where a program starts is given as an address in memory; a rule wants the
	// place in the file it corresponds to.
	if entry, ok := f.word(entryAt); ok && entry != 0 {
		if at, found := f.addressToOffset(entry); found {
			fields["entry_point"] = elfInt(at)
		}
	}
}

// put reads one field of the file into what the module offers, leaving it out
// when the bytes it would come from are not there.
func (f elfFile) put(fields map[string]modValue, name string, at int,
	read func(int) (uint64, bool),
) {
	if n, ok := read(at); ok {
		fields[name] = elfInt(n)
	}
}

// elfSection is one entry of the section table.
type elfSection struct {
	name, kind, link          uint64
	flags, addr, offset, size uint64
}

// sectionTable says where the section table is and how big each entry is.
func (f elfFile) sectionTable() (at, entrySize, count int, ok bool) {
	_, _, shAt, countsAt := f.headerFields()
	offset, offOK := f.word(shAt)
	n, countOK := f.u16(countsAt + 8)
	if !offOK || !countOK || n >= shnLoReserve {
		return 0, 0, 0, false
	}
	entrySize = 40
	if f.wide {
		entrySize = 64
	}
	if offset == 0 || offset > uint64(len(f.data)) ||
		offset+n*uint64(entrySize) > uint64(len(f.data)) {
		return 0, 0, 0, false
	}
	return int(offset), entrySize, int(n), true // #nosec G115 -- bounded by the file's length
}

// sections reads the section table, which says how the file is laid out on
// disk and where its symbols and their names are kept.
func (f elfFile) sections() []elfSection {
	at, entrySize, count, ok := f.sectionTable()
	if !ok {
		return nil
	}
	out := make([]elfSection, 0, count)
	for i := range count {
		out = append(out, f.readSection(at+i*entrySize))
	}
	return out
}

// readSection reads one entry of the section table.
func (f elfFile) readSection(at int) elfSection {
	var s elfSection
	s.name, _ = f.u32(at)
	s.kind, _ = f.u32(at + 4)
	if f.wide {
		s.flags, _ = f.u64(at + 8)
		s.addr, _ = f.u64(at + 16)
		s.offset, _ = f.u64(at + 24)
		s.size, _ = f.u64(at + 32)
		s.link, _ = f.u32(at + 40)
		return s
	}
	s.flags, _ = f.u32(at + 8)
	s.addr, _ = f.u32(at + 12)
	s.offset, _ = f.u32(at + 16)
	s.size, _ = f.u32(at + 20)
	s.link, _ = f.u32(at + 24)
	return s
}

// addSections lists what each section says about itself, with its name looked
// up in whichever section holds the names.
func (f elfFile) addSections(fields map[string]modValue, sections []elfSection) {
	if len(sections) == 0 {
		return
	}
	names := f.sectionNames(sections)
	items := make([]modValue, 0, len(sections))
	for _, s := range sections {
		member := map[string]modValue{
			"type": elfInt(s.kind), "flags": elfInt(s.flags), "address": elfInt(s.addr),
			"size": elfInt(s.size), "offset": elfInt(s.offset),
		}
		if name, ok := stringAt(f.data, names, s.name, uint64(len(f.data))); ok {
			member["name"] = valueOf(stringValue(name))
		}
		items = append(items, structOf(member))
	}
	fields["sections"] = listOf(items)
}

// sectionNames is where the names of the sections are kept, which the header
// points at by its place in the section table.
func (f elfFile) sectionNames(sections []elfSection) int {
	_, _, _, countsAt := f.headerFields()
	which, ok := f.u16(countsAt + 10)
	if !ok || which >= uint64(len(sections)) {
		return -1
	}
	// libyara takes a table at the very start of the file to be no table at
	// all, and will not read a name from one.
	at := sections[which].offset
	if at == 0 || at >= uint64(len(f.data)) {
		return -1
	}
	return int(at) // #nosec G115 -- bounded by the file's length
}

// stringAt reads a name out of a table of them, which run one after another
// each ending in a nought.
func stringAt(data []byte, table int, at, end uint64) (string, bool) {
	if table < 0 || at >= uint64(len(data)) {
		return "", false
	}
	if end > uint64(len(data)) {
		end = uint64(len(data))
	}
	start := uint64(table) + at
	if start >= end {
		return "", false
	}
	for i := start; i < end; i++ {
		if data[i] == 0 {
			return string(data[start:i]), true
		}
	}
	return "", false
}

// elfInt wraps a field of the file as a number a condition can use.
func elfInt(n uint64) modValue { return valueOf(intValue(int64(n))) } // #nosec G115 -- a field of the file

// segmentTable says where the segment table is and how big each entry is.
func (f elfFile) segmentTable() (at, entrySize, count int, ok bool) {
	_, phAt, _, countsAt := f.headerFields()
	offset, offOK := f.word(phAt)
	n, countOK := f.u16(countsAt + 4)
	if !offOK || !countOK || n == 0 || n >= pnXNum {
		return 0, 0, 0, false
	}
	entrySize = 32
	if f.wide {
		entrySize = 56
	}
	if offset >= uint64(len(f.data)) ||
		offset+n*uint64(entrySize) > uint64(len(f.data)) {
		return 0, 0, 0, false
	}
	return int(offset), entrySize, int(n), true // #nosec G115 -- bounded by the file's length
}

// addSegments lists how the file is laid out once it has been loaded.
func (f elfFile) addSegments(fields map[string]modValue) {
	at, entrySize, count, ok := f.segmentTable()
	if !ok {
		return
	}
	items := make([]modValue, 0, count)
	for i := range count {
		segment, kind, offset := f.readSegment(at + i*entrySize)
		items = append(items, segment)
		// The segment that lists what the file needs at run time is read out
		// entry by entry as well as being listed as a segment.
		if kind == elfConst("PT_DYNAMIC") {
			f.addDynamic(fields, offset)
		}
	}
	fields["segments"] = listOf(items)
}

// addDynamic reads the list of what a file needs at run time, which runs from a
// given place until an entry says it has ended or the file runs out.
func (f elfFile) addDynamic(fields map[string]modValue, at uint64) {
	entrySize := uint64(8)
	if f.wide {
		entrySize = 16
	}
	var items []modValue
	if at < uint64(len(f.data)) {
		for place := at; place+entrySize <= uint64(len(f.data)); place += entrySize {
			kind, _ := f.word(int(place))              // #nosec G115 -- checked against the file's length
			val, _ := f.word(int(place + entrySize/2)) // #nosec G115 -- likewise
			items = append(items, structOf(map[string]modValue{
				"type": elfInt(kind), "val": elfInt(val),
			}))
			if kind == elfConst("DT_NULL") {
				break
			}
		}
	}
	fields["dynamic"] = listOf(items)
	fields["dynamic_section_entries"] = valueOf(intValue(int64(len(items))))
}

// readSegment reads one entry of the segment table. The two widths keep the
// same fields in a different order.
func (f elfFile) readSegment(at int) (modValue, uint64, uint64) {
	kind, _ := f.u32(at)
	member := map[string]modValue{"type": elfInt(kind)}
	places := []struct {
		name string
		at   int
	}{
		{"offset", 4},
		{"virtual_address", 8},
		{"physical_address", 12},
		{"file_size", 16},
		{"memory_size", 20},
		{"alignment", 28},
	}
	flagsAt := at + 24
	if f.wide {
		places = []struct {
			name string
			at   int
		}{
			{"offset", 8},
			{"virtual_address", 16},
			{"physical_address", 24},
			{"file_size", 32},
			{"memory_size", 40},
			{"alignment", 48},
		}
		flagsAt = at + 4
	}
	flags, _ := f.u32(flagsAt)
	member["flags"] = elfInt(flags)
	for _, place := range places {
		n, _ := f.word(at + place.at)
		member[place.name] = elfInt(n)
	}
	offset, _ := f.word(at + places[0].at)
	return structOf(member), kind, offset
}

// addSymbols lists the names a file defines and the names it looks for
// elsewhere, which are kept in two sections of the same shape.
func (f elfFile) addSymbols(fields map[string]modValue, sections []elfSection) {
	for _, s := range sections {
		var into, count string
		switch s.kind {
		case elfConst("SHT_SYMTAB"):
			into, count = "symtab", "symtab_entries"
		case elfConst("SHT_DYNSYM"):
			into, count = "dynsym", "dynsym_entries"
		default:
			continue
		}
		if s.link >= uint64(len(sections)) {
			continue
		}
		names := sections[s.link]
		if names.kind != elfConst("SHT_STRTAB") {
			continue
		}
		items, ok := f.readSymbols(s, names)
		if !ok {
			continue
		}
		fields[into] = listOf(items)
		fields[count] = valueOf(intValue(int64(len(items))))
	}
}

// readSymbols reads one symbol table, whose names are kept in a section of
// their own.
func (f elfFile) readSymbols(table, names elfSection) ([]modValue, bool) {
	entrySize := uint64(16)
	if f.wide {
		entrySize = 24
	}
	if !f.holds(table.offset, table.size) || !f.holds(names.offset, names.size) {
		return nil, false
	}
	items := make([]modValue, 0, table.size/entrySize)
	for at := table.offset; at+entrySize <= table.offset+table.size; at += entrySize {
		// #nosec G115 -- both are checked against the file's length above
		items = append(items, f.readSymbol(int(at), int(names.offset), names.offset+names.size))
	}
	return items, true
}

// holds reports whether a stretch of the file is there to be read.
func (f elfFile) holds(at, size uint64) bool {
	return at <= uint64(len(f.data)) && at+size <= uint64(len(f.data))
}

// readSymbol reads one symbol. The two widths keep the same fields in a
// different order.
func (f elfFile) readSymbol(at, names int, namesEnd uint64) modValue {
	name, _ := f.u32(at)
	var info byte
	var shndx, value, size uint64
	if f.wide {
		info = f.byteAt(at + 4)
		shndx, _ = f.u16(at + 6)
		value, _ = f.u64(at + 8)
		size, _ = f.u64(at + 16)
	} else {
		value, _ = f.u32(at + 4)
		size, _ = f.u32(at + 8)
		info = f.byteAt(at + 12)
		shndx, _ = f.u16(at + 14)
	}
	member := map[string]modValue{
		"bind": elfInt(uint64(info >> symTypeBits)), "type": elfInt(uint64(info & symTypeMask)),
		"shndx": elfInt(shndx), "value": elfInt(value), "size": elfInt(size),
	}
	if text, ok := stringAt(f.data, names, name, namesEnd); ok {
		member["name"] = valueOf(stringValue(text))
	}
	return structOf(member)
}

// byteAt reads one byte, or nought where there is none.
func (f elfFile) byteAt(at int) byte {
	if at < 0 || at >= len(f.data) {
		return 0
	}
	return f.data[at]
}

// addressToOffset turns an address in memory into the place in the file it
// came from. A program that is meant to run at a fixed address is looked up in
// the segment table; anything else in the section table.
func (f elfFile) addressToOffset(address uint64) (uint64, bool) {
	kind, ok := f.u16(16)
	if !ok {
		return 0, false
	}
	if kind == elfConst("ET_EXEC") {
		return f.addressInSegments(address)
	}
	return f.addressInSections(address)
}

func (f elfFile) addressInSegments(address uint64) (uint64, bool) {
	at, entrySize, count, ok := f.segmentTable()
	if !ok {
		return 0, false
	}
	for i := range count {
		start := at + i*entrySize
		virtual, memory, offset := 16, 40, 8
		if !f.wide {
			virtual, memory, offset = 8, 20, 4
		}
		base, _ := f.word(start + virtual)
		size, _ := f.word(start + memory)
		if address >= base && address < base+size {
			place, _ := f.word(start + offset)
			return place + (address - base), true
		}
	}
	return 0, false
}

func (f elfFile) addressInSections(address uint64) (uint64, bool) {
	for _, s := range f.sections() {
		if s.kind == elfConst("SHT_NULL") ||
			s.kind == elfConst("SHT_NOBITS") {
			continue
		}
		if address >= s.addr && address < s.addr+s.size {
			return s.offset + (address - s.addr), true
		}
	}
	return 0, false
}
