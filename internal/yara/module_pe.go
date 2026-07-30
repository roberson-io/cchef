package yara

import (
	"crypto/md5" // #nosec G501 -- the digest of what a file borrows is defined as md5
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

// The pe module, which reads the shape of an executable built for Windows.
//
// A PE file opens with a stub left over from DOS, whose last field points at
// the header proper. That header says what machine the file is for and how many
// sections follow; an optional header after it says how the file is to be laid
// out in memory and where to find the tables that list what it imports, exports
// and carries. Everything a rule asks for is read from those.

// The sizes and limits the format fixes.
const (
	// dosHeaderSize is the stub a PE file opens with, whose last field says
	// where the header proper begins.
	dosHeaderSize = 0x40
	newHeaderAt   = 0x3C
	// fileHeaderSize is the header that names the machine and counts the
	// sections, which follows the four bytes marking the file as a PE.
	peSignatureSize = 4
	fileHeaderSize  = 20
	sectionSize     = 40
	// sectionNameSize is how much room a section's name has before it has to be
	// kept elsewhere.
	sectionNameSize = 8
	// The two shapes the optional header comes in, for programs built to run
	// with 32-bit or 64-bit addresses.
	optionalMagic32 = 0x10B
	optionalMagic64 = 0x20B
	// maxSections is as many as libyara will read, however many the file claims.
	maxSections = 96
	// pageSize and sectorSize are what section offsets are rounded to when
	// working out where in the file an address in memory came from.
	pageSize   = 0x1000
	sectorSize = 0x0200
	// dataDirectoryEntry is how much room each entry of the table of tables
	// takes: an address and a length.
	dataDirectoryEntry = 8
	// checksumSkip is where a file's own checksum sits in the optional header,
	// which is left out when the checksum is worked out again.
	checksumSkip = 64
	// maxDataDirectories is as many tables as the format has room to name,
	// however many a file claims. It is a limit of the format rather than
	// something a rule can ask about, so it is not among the constants.
	maxDataDirectories = 16
)

var peSignature = []byte{'P', 'E', 0, 0}

func peSchema() *modDecl {
	members := map[string]*modDecl{
		"is_pe": decInt(), "machine": decInt(), "number_of_sections": decInt(),
		"timestamp": decInt(), "pointer_to_symbol_table": decInt(),
		"number_of_symbols": decInt(), "size_of_optional_header": decInt(),
		"characteristics": decInt(),

		"entry_point": decInt(), "entry_point_raw": decInt(), "image_base": decInt(),
		"number_of_rva_and_sizes": decInt(), "opthdr_magic": decInt(),
		"size_of_code": decInt(), "size_of_initialized_data": decInt(),
		"size_of_uninitialized_data": decInt(), "base_of_code": decInt(),
		"base_of_data": decInt(), "section_alignment": decInt(),
		"file_alignment": decInt(), "win32_version_value": decInt(),
		"size_of_image": decInt(), "size_of_headers": decInt(), "checksum": decInt(),
		"subsystem": decInt(), "dll_characteristics": decInt(),
		"size_of_stack_reserve": decInt(), "size_of_stack_commit": decInt(),
		"size_of_heap_reserve": decInt(), "size_of_heap_commit": decInt(),
		"loader_flags": decInt(),

		"linker_version":    peVersionDecl(),
		"os_version":        peVersionDecl(),
		"image_version":     peVersionDecl(),
		"subsystem_version": peVersionDecl(),

		"data_directories": decArray(decStruct(map[string]*modDecl{
			"virtual_address": decInt(), "size": decInt(),
		})),
		"sections": decArray(decStruct(map[string]*modDecl{
			"name": decString(), "full_name": decString(), "characteristics": decInt(),
			"virtual_address": decInt(), "virtual_size": decInt(),
			"raw_data_offset": decInt(), "raw_data_size": decInt(),
			"pointer_to_relocations": decInt(), "pointer_to_line_numbers": decInt(),
			"number_of_relocations": decInt(), "number_of_line_numbers": decInt(),
		})),
		"overlay": decStruct(map[string]*modDecl{
			"offset": decInt(), "size": decInt(),
		}),

		"calculate_checksum": decFunc(modInt, ""),
		"rva_to_offset":      decFunc(modInt, "i"),
		"section_index":      decFunc(modInt, "s", "i"),
	}
	importSchema(members)
	resourceSchema(members)
	richSchema(members)
	signatureSchema(members)
	for name := range peConstants {
		members[name] = decInt()
	}
	return decStruct(members)
}

// peVersionDecl is a pair of numbers naming a version, which several fields of
// the optional header are written as.
func peVersionDecl() *modDecl {
	return decStruct(map[string]*modDecl{"major": decInt(), "minor": decInt()})
}

// peFile is a PE file being read: the bytes, where its headers begin, and
// whether it is built for 64-bit addresses.
type peFile struct {
	data []byte
	// nt is where the header proper begins, and opt where the optional header
	// after it does.
	nt, opt  int
	wide     bool
	sections []peSection
}

// peSection is one entry of the section table.
type peSection struct {
	name, fullName                      string
	virtualAddress, virtualSize         uint32
	rawOffset, rawSize, characteristics uint32
	relocations, lineNumbers            uint32
	numRelocations, numLineNumbers      uint32
}

// peModule reads the data as a PE file. When it is not one, only the constants
// and a plain no for is_pe are offered.
func peModule(e *evaluator) modValue {
	fields := make(map[string]modValue, len(peConstants)+40)
	for name, n := range peConstants {
		fields[name] = valueOf(intValue(n))
	}

	f, ok := readPE(e.buf.data)
	if !ok {
		fields["is_pe"] = valueOf(intValue(0))
		return structOf(fields)
	}
	fields["is_pe"] = valueOf(intValue(1))
	f.addFileHeader(fields)
	f.addOptionalHeader(fields)
	f.addDataDirectories(fields)
	f.addSections(fields)
	f.addOverlay(fields)
	f.addImportsAndExports(fields)
	f.addResources(fields)
	f.addPdbPath(fields)
	f.addRichSignature(fields)
	f.addSignatures(fields)
	f.addFunctions(fields)
	return structOf(fields)
}

// readPE checks that the data is a PE file and finds its headers.
func readPE(data []byte) (*peFile, bool) {
	if len(data) < dosHeaderSize || data[0] != 'M' || data[1] != 'Z' {
		return nil, false
	}
	at := int(binary.LittleEndian.Uint32(data[newHeaderAt:]))
	if at < 0 || at+peSignatureSize+fileHeaderSize > len(data) {
		return nil, false
	}
	if string(data[at:at+peSignatureSize]) != string(peSignature) {
		return nil, false
	}

	f := &peFile{data: data, nt: at, opt: at + peSignatureSize + fileHeaderSize}
	magic, ok := f.u16(f.opt)
	if !ok {
		return nil, false
	}
	switch magic {
	case optionalMagic32:
	case optionalMagic64:
		f.wide = true
	default:
		return nil, false
	}
	f.sections = f.readSections()
	return f, true
}

// The ways of reading a number out of the file, each coming to nothing when the
// bytes asked for are not there.
func (f *peFile) u8(at int) (uint32, bool) {
	if at < 0 || at >= len(f.data) {
		return 0, false
	}
	return uint32(f.data[at]), true
}

func (f *peFile) u16(at int) (uint32, bool) {
	if at < 0 || at+2 > len(f.data) {
		return 0, false
	}
	return uint32(binary.LittleEndian.Uint16(f.data[at:])), true
}

func (f *peFile) u32(at int) (uint32, bool) {
	if at < 0 || at+4 > len(f.data) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(f.data[at:]), true
}

func (f *peFile) u64(at int) (uint64, bool) {
	if at < 0 || at+8 > len(f.data) {
		return 0, false
	}
	return binary.LittleEndian.Uint64(f.data[at:]), true
}

// put reads one field into what the module offers, leaving it out when the
// bytes it would come from are not there.
func (f *peFile) put(fields map[string]modValue, name string, at int,
	read func(int) (uint32, bool),
) {
	if n, ok := read(at); ok {
		fields[name] = valueOf(intValue(int64(n)))
	}
}

// addFileHeader reads the header that names the machine and counts what
// follows.
func (f *peFile) addFileHeader(fields map[string]modValue) {
	at := f.nt + peSignatureSize
	f.put(fields, "machine", at, f.u16)
	f.put(fields, "number_of_sections", at+2, f.u16)
	f.put(fields, "timestamp", at+4, f.u32)
	f.put(fields, "pointer_to_symbol_table", at+8, f.u32)
	f.put(fields, "number_of_symbols", at+12, f.u32)
	f.put(fields, "size_of_optional_header", at+16, f.u16)
	f.put(fields, "characteristics", at+18, f.u16)
}

// optionalFields is where each field of the optional header sits. The two
// shapes agree as far as the code's base and part company after it, since one
// writes addresses in four bytes and the other in eight.
func (f *peFile) optionalFields() (imageBase, stack, loader int) {
	if f.wide {
		return 24, 72, 104
	}
	return 28, 72, 88
}

// addOptionalHeader reads how the file is to be laid out once loaded.
func (f *peFile) addOptionalHeader(fields map[string]modValue) {
	at := f.opt
	imageBaseAt, stackAt, loaderAt := f.optionalFields()

	f.put(fields, "opthdr_magic", at, f.u16)
	f.putVersion(fields, "linker_version", at+2, at+3, f.u8)
	f.put(fields, "size_of_code", at+4, f.u32)
	f.put(fields, "size_of_initialized_data", at+8, f.u32)
	f.put(fields, "size_of_uninitialized_data", at+12, f.u32)
	f.put(fields, "entry_point_raw", at+16, f.u32)
	f.put(fields, "base_of_code", at+20, f.u32)
	if !f.wide {
		f.put(fields, "base_of_data", at+24, f.u32)
	}
	if base, ok := f.address(at + imageBaseAt); ok {
		fields["image_base"] = valueOf(intValue(int64(base))) // #nosec G115 -- a field of the file
	}

	f.put(fields, "section_alignment", at+32, f.u32)
	f.put(fields, "file_alignment", at+36, f.u32)
	f.putVersion(fields, "os_version", at+40, at+42, f.u16)
	f.putVersion(fields, "image_version", at+44, at+46, f.u16)
	f.putVersion(fields, "subsystem_version", at+48, at+50, f.u16)
	f.put(fields, "win32_version_value", at+52, f.u32)
	f.put(fields, "size_of_image", at+56, f.u32)
	f.put(fields, "size_of_headers", at+60, f.u32)
	f.put(fields, "checksum", at+64, f.u32)
	f.put(fields, "subsystem", at+68, f.u16)
	f.put(fields, "dll_characteristics", at+70, f.u16)

	step := 4
	if f.wide {
		step = 8
	}
	for i, name := range []string{
		"size_of_stack_reserve", "size_of_stack_commit",
		"size_of_heap_reserve", "size_of_heap_commit",
	} {
		if n, ok := f.address(at + stackAt + i*step); ok {
			fields[name] = valueOf(intValue(int64(n))) // #nosec G115 -- a field of the file
		}
	}
	f.put(fields, "loader_flags", at+loaderAt, f.u32)
	f.put(fields, "number_of_rva_and_sizes", at+loaderAt+4, f.u32)

	// Where a program starts is given as an address in memory; a rule wants the
	// place in the file it corresponds to as well. An address that came from
	// nowhere in the file is reported as -1 here, rather than as no answer:
	// libyara stores whatever the conversion gave, and only the function a rule
	// can call itself turns that into no answer.
	if rva, ok := f.u32(at + 16); ok {
		offset, found := f.rvaToOffset(uint64(rva))
		if !found {
			offset = -1
		}
		fields["entry_point"] = valueOf(intValue(offset))
	}
}

// address reads a number as wide as the file writes its addresses.
func (f *peFile) address(at int) (uint64, bool) {
	if f.wide {
		return f.u64(at)
	}
	n, ok := f.u32(at)
	return uint64(n), ok
}

// putVersion reads a pair of numbers naming a version.
func (f *peFile) putVersion(fields map[string]modValue, name string,
	majorAt, minorAt int, read func(int) (uint32, bool),
) {
	pair := map[string]modValue{}
	if n, ok := read(majorAt); ok {
		pair["major"] = valueOf(intValue(int64(n)))
	}
	if n, ok := read(minorAt); ok {
		pair["minor"] = valueOf(intValue(int64(n)))
	}
	fields[name] = structOf(pair)
}

// dataDirectoriesAt is where the table of tables begins, which follows the rest
// of the optional header.
func (f *peFile) dataDirectoriesAt() int {
	if f.wide {
		return f.opt + 112
	}
	return f.opt + 96
}

// addDataDirectories lists where each of the file's tables is to be found.
func (f *peFile) addDataDirectories(fields map[string]modValue) {
	count, ok := f.u32(f.dataDirectoriesAt() - 4)
	if !ok {
		return
	}
	count = min(count, maxDataDirectories)
	at := f.dataDirectoriesAt()
	items := make([]modValue, 0, count)
	for i := range int(count) {
		address, addressOK := f.u32(at + i*dataDirectoryEntry)
		size, sizeOK := f.u32(at + i*dataDirectoryEntry + 4)
		if !addressOK || !sizeOK {
			break
		}
		items = append(items, structOf(map[string]modValue{
			"virtual_address": valueOf(intValue(int64(address))),
			"size":            valueOf(intValue(int64(size))),
		}))
	}
	fields["data_directories"] = listOf(items)
}

// sectionsAt is where the section table begins, which follows the optional
// header at whatever length that header claims.
func (f *peFile) sectionsAt() int {
	size, ok := f.u16(f.nt + peSignatureSize + 16)
	if !ok {
		return -1
	}
	return f.opt + int(size)
}

// readSections reads the section table.
func (f *peFile) readSections() []peSection {
	at := f.sectionsAt()
	count, ok := f.u16(f.nt + peSignatureSize + 2)
	if at < 0 || !ok {
		return nil
	}
	if count > maxSections {
		count = maxSections
	}
	out := make([]peSection, 0, count)
	for i := range int(count) {
		start := at + i*sectionSize
		if start+sectionSize > len(f.data) {
			break
		}
		out = append(out, f.readSection(start))
	}
	return out
}

// readSection reads one entry of the section table.
func (f *peFile) readSection(at int) peSection {
	var s peSection
	raw := f.data[at : at+sectionNameSize]
	s.fullName = string(raw)
	if cut := indexOfNul(raw); cut >= 0 {
		s.fullName = string(raw[:cut])
	}
	s.name = s.fullName

	s.virtualSize, _ = f.u32(at + 8)
	s.virtualAddress, _ = f.u32(at + 12)
	s.rawSize, _ = f.u32(at + 16)
	s.rawOffset, _ = f.u32(at + 20)
	s.relocations, _ = f.u32(at + 24)
	s.lineNumbers, _ = f.u32(at + 28)
	s.numRelocations, _ = f.u16(at + 32)
	s.numLineNumbers, _ = f.u16(at + 34)
	s.characteristics, _ = f.u32(at + 36)
	return s
}

// indexOfNul is where a name written into a fixed run of bytes ends, or -1 when
// it fills the whole run.
func indexOfNul(raw []byte) int {
	for i, b := range raw {
		if b == 0 {
			return i
		}
	}
	return -1
}

// addSections lists what each section says about itself.
func (f *peFile) addSections(fields map[string]modValue) {
	items := make([]modValue, 0, len(f.sections))
	for _, s := range f.sections {
		items = append(items, structOf(map[string]modValue{
			"name":                    valueOf(stringValue(s.name)),
			"full_name":               valueOf(stringValue(s.fullName)),
			"characteristics":         peInt(s.characteristics),
			"virtual_address":         peInt(s.virtualAddress),
			"virtual_size":            peInt(s.virtualSize),
			"raw_data_offset":         peInt(s.rawOffset),
			"raw_data_size":           peInt(s.rawSize),
			"pointer_to_relocations":  peInt(s.relocations),
			"pointer_to_line_numbers": peInt(s.lineNumbers),
			"number_of_relocations":   peInt(s.numRelocations),
			"number_of_line_numbers":  peInt(s.numLineNumbers),
		}))
	}
	fields["sections"] = listOf(items)
}

// peInt wraps a field of the file as a number a condition can use.
func peInt(n uint32) modValue { return valueOf(intValue(int64(n))) }

// addOverlay reports whatever follows the last section, which is not part of
// the program at all. A file with nothing after its sections reports nought for
// both, rather than leaving them out.
func (f *peFile) addOverlay(fields map[string]modValue) {
	// The section reaching furthest into the file decides where the overlay
	// begins. Two sections may start at the same place with different lengths,
	// so it is the end that is compared, not the length.
	var highestAt, highestSize uint64
	for _, s := range f.sections {
		if end := uint64(s.rawOffset) + uint64(s.rawSize); end > highestAt+highestSize {
			highestAt, highestSize = uint64(s.rawOffset), uint64(s.rawSize)
		}
	}

	end := highestAt + highestSize
	offset, size := uint64(0), uint64(0)
	if end != 0 && uint64(len(f.data)) > end {
		offset, size = end, uint64(len(f.data))-end
	}
	fields["overlay"] = structOf(map[string]modValue{
		"offset": valueOf(intValue(int64(offset))), // #nosec G115 -- bounded by the file's length
		"size":   valueOf(intValue(int64(size))),   // #nosec G115 -- likewise
	})
}

// rvaToOffset turns an address in memory into the place in the file it came
// from, the way libyara does: the section holding it decides, with its offset
// rounded down to how the file is aligned.
func (f *peFile) rvaToOffset(rva uint64) (int64, bool) {
	lowest := uint64(0xFFFFFFFF)
	var sectionRVA, sectionOffset, sectionRawSize uint64

	alignment := uint64(0)
	if n, ok := f.u32(f.opt + 36); ok {
		alignment = min(uint64(n), sectorSize)
	}
	sectionAlignment := uint64(0)
	if n, ok := f.u32(f.opt + 32); ok {
		sectionAlignment = uint64(n)
	}

	for _, s := range f.sections {
		lowest = min(lowest, uint64(s.virtualAddress))
		// What a section covers is the room it takes in memory. A later libyara
		// takes the larger of that and its room on disk; the build CyberChef
		// runs does not, and a section claiming no room in memory therefore
		// covers nothing.
		covers := uint64(s.virtualSize)
		if rva < uint64(s.virtualAddress) || rva-uint64(s.virtualAddress) >= covers ||
			sectionRVA > uint64(s.virtualAddress) {
			continue
		}
		sectionRVA = uint64(s.virtualAddress)
		sectionOffset = uint64(s.rawOffset)
		sectionRawSize = uint64(s.rawSize)
		if alignment != 0 {
			sectionOffset -= sectionOffset % alignment
		}
		if sectionAlignment >= pageSize {
			sectionOffset &^= sectorSize - 1
		}
	}

	// Everything before the first section is laid out just as it is on disk.
	if rva < lowest {
		sectionRVA, sectionOffset = 0, 0
		sectionRawSize = uint64(len(f.data))
	}
	// A section with less room on disk than in memory has addresses that came
	// from nowhere in the file.
	if rva-sectionRVA >= sectionRawSize {
		return 0, false
	}
	at := sectionOffset + (rva - sectionRVA)
	if at >= uint64(len(f.data)) {
		return 0, false
	}
	return int64(at), true // #nosec G115 -- checked against the file's length
}

// addFunctions offers the things a rule works out rather than reads.
func (f *peFile) addFunctions(fields map[string]modValue) {
	fields["rva_to_offset"] = funcOf(func(_ *evaluator, args []value) (value, error) {
		if len(args) != 1 || args[0].kind != valueInt || args[0].i < 0 {
			return undefined, nil
		}
		at, ok := f.rvaToOffset(uint64(args[0].i)) // #nosec G115 -- checked not to be negative above
		if !ok {
			return undefined, nil
		}
		return intValue(at), nil
	})
	fields["section_index"] = funcOf(func(_ *evaluator, args []value) (value, error) {
		return f.sectionIndex(args), nil
	})
	fields["calculate_checksum"] = funcOf(func(_ *evaluator, _ []value) (value, error) {
		return intValue(f.checksum()), nil
	})
}

// sectionIndex is where a section sits in the table, found either by its name
// or by an address it covers.
func (f *peFile) sectionIndex(args []value) value {
	if len(args) != 1 {
		return undefined
	}
	for i, s := range f.sections {
		if args[0].kind == valueString && s.name == args[0].s {
			return intValue(int64(i))
		}
		if args[0].kind == valueInt && coversAddress(s, args[0].i) {
			return intValue(int64(i))
		}
	}
	return undefined
}

// coversAddress reports whether a section takes in a given address once the
// file has been loaded.
func coversAddress(s peSection, address int64) bool {
	if address < 0 {
		return false
	}
	at := uint64(address) // #nosec G115 -- checked not to be negative above
	return at >= uint64(s.virtualAddress) &&
		at < uint64(s.virtualAddress)+uint64(s.virtualSize)
}

// checksum works the file's checksum out again, which is a running sum over the
// whole file with the recorded checksum left out and the length added at the
// end.
func (f *peFile) checksum() int64 {
	const wordBits, wordMask = 32, 0xFFFFFFFF
	skipFrom := f.opt + checksumSkip
	var sum uint64
	for at := 0; at+2 <= len(f.data); at += 2 {
		if at >= skipFrom && at < skipFrom+4 {
			continue
		}
		sum += uint64(binary.LittleEndian.Uint16(f.data[at:]))
		sum = (sum & wordMask) + (sum >> wordBits)
	}
	// A file of odd length ends on a byte with nothing to pair it with.
	if len(f.data)%2 == 1 {
		sum += uint64(f.data[len(f.data)-1])
		sum = (sum & wordMask) + (sum >> wordBits)
	}
	sum = (sum & 0xFFFF) + (sum >> 16)
	sum += sum >> 16
	sum &= 0xFFFF
	return int64(sum) + int64(len(f.data)) // #nosec G115 -- a sum of sixteen-bit words
}

// The run of bytes Microsoft's linker leaves in the stub at the front of a PE
// file.
//
// It names every tool that had a hand in building the file and how many times
// each was used, and it is hidden behind a running exclusive-or with a key kept
// at the very end of it. Finding it means looking backwards from the header
// proper for the closing marker, taking the key that follows it, and then
// looking further back for the opening marker once the key is taken off.

const (
	// richDans and richRich are the markers that open and close the run, which
	// read as "DanS" and "Rich".
	richDans = 0x536E6144
	richRich = 0x68636952
	// richOpeningSize is the marker and the three blanks that follow it, before
	// the tools begin.
	richOpeningSize = 16
	// richToolSize is one tool: what it was and how often, in two numbers.
	richToolSize = 8
	// richIDShift separates which tool it was from which version of it.
	richIDShift    = 16
	richVersionBit = 0xFFFF
)

// richSchema adds to what the pe module declares the parts to do with that run.
func richSchema(members map[string]*modDecl) {
	members["rich_signature"] = decStruct(map[string]*modDecl{
		"offset": decInt(), "length": decInt(), "key": decInt(),
		"raw_data": decString(), "clear_data": decString(),
		"version": decFunc(modInt, "i", "ii"),
		"toolid":  decFunc(modInt, "i", "ii"),
	})
}

// richTool is one tool named in the run.
type richTool struct {
	id, version uint16
	times       uint32
}

// addRichSignature reads the run and offers what a rule may ask about it.
func (f *peFile) addRichSignature(fields map[string]modValue) {
	at, key, length, found := f.findRichSignature()
	if !found {
		// Nothing of the run is offered, but the questions still are, so that a
		// rule asking one gets no answer rather than being refused.
		fields["rich_signature"] = structOf(map[string]modValue{
			"version": richCount(nil, false), "toolid": richCount(nil, false),
		})
		return
	}

	raw := f.data[at : at+length]
	plain := make([]byte, len(raw))
	for i := 0; i+4 <= len(raw); i += 4 {
		binary.LittleEndian.PutUint32(plain[i:], binary.LittleEndian.Uint32(raw[i:])^key)
	}
	tools := richTools(plain)

	fields["rich_signature"] = structOf(map[string]modValue{
		"offset":     valueOf(intValue(int64(at))),
		"length":     valueOf(intValue(int64(length))),
		"key":        peInt(key),
		"raw_data":   valueOf(stringValue(string(raw))),
		"clear_data": valueOf(stringValue(string(plain))),
		"version":    richCount(tools, true),
		"toolid":     richCount(tools, false),
	})
}

// findRichSignature looks backwards from the header proper for the run: first
// the marker that closes it and the key that follows, then the opening marker
// with the key taken off.
func (f *peFile) findRichSignature() (at int, key uint32, length int, found bool) {
	if f.nt < 4 {
		return 0, 0, 0, false
	}
	closing := -1
	for p := f.nt - 4; p >= dosHeaderSize; p -= 4 {
		word, ok := f.u32(p)
		if !ok {
			return 0, 0, 0, false
		}
		if word == richRich {
			closing = p
			break
		}
	}
	if closing < 0 {
		return 0, 0, 0, false
	}
	key, ok := f.u32(closing + 4)
	if !ok || key == 0 {
		return 0, 0, 0, false
	}

	// Everything from here back is within the file, since the closing marker
	// was read from further along it.
	for p := closing - 4; p >= dosHeaderSize; p -= 4 {
		word, _ := f.u32(p)
		if word^key == richDans {
			return p, key, closing - p, true
		}
	}
	return 0, 0, 0, false
}

// richTools reads the tools out of the run once the key has been taken off.
func richTools(plain []byte) []richTool {
	if len(plain) < richOpeningSize {
		return nil
	}
	out := make([]richTool, 0, (len(plain)-richOpeningSize)/richToolSize)
	for at := richOpeningSize; at+richToolSize <= len(plain); at += richToolSize {
		both := binary.LittleEndian.Uint32(plain[at:])
		out = append(out, richTool{
			id:      uint16(both >> richIDShift), // #nosec G115 -- the upper half of a word
			version: uint16(both & richVersionBit),
			times:   binary.LittleEndian.Uint32(plain[at+4:]),
		})
	}
	return out
}

// richCount answers how often a tool was used. The two questions differ only in
// which number is asked for first: one asks by version, the other by which tool
// it was, and either may name both.
func richCount(tools []richTool, byVersion bool) modValue {
	return funcOf(func(_ *evaluator, args []value) (value, error) {
		if tools == nil {
			return undefined, nil
		}
		wanted, other, ok := richWanted(args, byVersion)
		if !ok {
			return undefined, nil
		}
		var total int64
		for _, tool := range tools {
			if wanted >= 0 && int64(tool.version) != wanted {
				continue
			}
			if other >= 0 && int64(tool.id) != other {
				continue
			}
			total += int64(tool.times)
		}
		return intValue(total), nil
	})
}

// richWanted reads what was asked for as a version and a tool, either of which
// may be left open. A number below nought stands for one that was not given.
func richWanted(args []value, byVersion bool) (version, id int64, ok bool) {
	version, id = -1, -1
	if len(args) == 0 || len(args) > 2 {
		return 0, 0, false
	}
	for _, arg := range args {
		if arg.kind != valueInt {
			return 0, 0, false
		}
	}
	// The first number is whichever the question asks by, and the second the
	// other one.
	if byVersion {
		version = args[0].i
		if len(args) == 2 {
			id = args[1].i
		}
		return version, id, true
	}
	id = args[0].i
	if len(args) == 2 {
		version = args[1].i
	}
	return version, id, true
}

// What a PE file borrows from elsewhere and what it lends out.
//
// Both are tables reached through the directory of tables: one lists the
// libraries the file needs and what it takes from each, the other what it makes
// available to others. Either may name a thing by name or by number.

// Which entry of the directory of tables holds what.
const (
	directoryExports = 0
	directoryImports = 1
	directoryDelayed = 13
)

// The limits libyara puts on how much of either table it will read.
const (
	maxImports = 16384
	maxExports = 16384
	// importDescriptorSize is one entry of the list of libraries borrowed from.
	importDescriptorSize = 20
	// exportDirectorySize is the header of the table of what is lent out.
	exportDirectorySize = 40
	// ordinalFlag32 and ordinalFlag64 mark a thing borrowed by number rather
	// than by name, and ordinalMask is where the number itself sits.
	ordinalFlag32 = 0x80000000
	ordinalFlag64 = 0x8000000000000000
	ordinalMask   = 0xFFFF
	// dllCharacteristic is the flag among a file's characteristics saying it is
	// a library rather than a program.
	dllCharacteristic = 0x2000
)

// peImport is one library a file borrows from.
type peImport struct {
	name      string
	functions []peImportedFunction
}

// peImportedFunction is one thing borrowed, which has a name either because it
// was given one or because libyara knows what its number stands for.
type peImportedFunction struct {
	name    string
	ordinal uint16
	byName  bool
}

// peExport is one thing a file lends out.
type peExport struct {
	name, forwardName string
	named, forwarded  bool
	ordinal           uint32
	offset            int64
	hasOffset         bool
}

// importSchema adds to what the pe module declares the parts to do with what a
// file borrows and lends.
func importSchema(members map[string]*modDecl) {
	members["number_of_imports"] = decInt()
	members["number_of_imported_functions"] = decInt()
	members["number_of_delayed_imports"] = decInt()
	members["number_of_delayed_imported_functions"] = decInt()
	members["import_details"] = decArray(decStruct(map[string]*modDecl{
		"library_name": decString(), "number_of_functions": decInt(),
		"functions": decArray(decStruct(map[string]*modDecl{
			"name": decString(), "ordinal": decInt(),
		})),
	}))
	members["number_of_exports"] = decInt()
	members["export_timestamp"] = decInt()
	members["export_details"] = decArray(decStruct(map[string]*modDecl{
		"offset": decInt(), "name": decString(),
		"forward_name": decString(), "ordinal": decInt(),
	}))

	members["imphash"] = decFunc(modString, "")
	members["imports"] = decFunc(modInt, "ss", "si", "s", "rr", "iss", "isi", "is", "irr")
	members["exports"] = decFunc(modInt, "s", "i", "r")
	members["exports_index"] = decFunc(modInt, "s", "i", "r")
	members["is_dll"] = decFunc(modInt, "")
	members["is_32bit"] = decFunc(modInt, "")
	members["is_64bit"] = decFunc(modInt, "")
}

// directoryEntry says where one of the file's own tables is and how big it is.
// which names one of the tables the format fixes, so it is always small.
func (f *peFile) directoryEntry(which uint32) (rva, size uint32, ok bool) {
	count, countOK := f.u32(f.dataDirectoriesAt() - 4)
	if !countOK || which >= min(count, maxDataDirectories) {
		return 0, 0, false
	}
	at := f.dataDirectoriesAt() + int(which)*dataDirectoryEntry
	rva, rvaOK := f.u32(at)
	size, sizeOK := f.u32(at + 4)
	return rva, size, rvaOK && sizeOK && rva != 0
}

// addImportsAndExports lists what the file borrows and lends, and offers the
// questions a rule may ask about either.
func (f *peFile) addImportsAndExports(fields map[string]modValue) {
	imports := f.readImports(directoryImports)
	f.addImports(fields, imports)
	f.addDelayedImportCounts(fields)
	exports := f.readExports()
	f.addExports(fields, exports)

	fields["imphash"] = funcOf(func(*evaluator, []value) (value, error) {
		return stringValue(importHash(imports)), nil
	})
	fields["imports"] = funcOf(func(_ *evaluator, args []value) (value, error) {
		return intValue(countImports(imports, args)), nil
	})
	fields["exports"] = funcOf(func(_ *evaluator, args []value) (value, error) {
		if at := findExport(exports, args); at >= 0 {
			return intValue(1), nil
		}
		return intValue(0), nil
	})
	fields["exports_index"] = funcOf(func(_ *evaluator, args []value) (value, error) {
		if at := findExport(exports, args); at >= 0 {
			return intValue(int64(at)), nil
		}
		return undefined, nil
	})

	fields["is_32bit"] = yesOrNo(!f.wide)
	fields["is_64bit"] = yesOrNo(f.wide)
	// Whether a file is a library is answered with the flag itself rather than
	// with one or nought, since libyara hands back what the masking gave.
	characteristics, _ := f.u16(f.nt + peSignatureSize + 18)
	flag := characteristics & dllCharacteristic
	fields["is_dll"] = funcOf(func(*evaluator, []value) (value, error) {
		return intValue(int64(flag)), nil
	})
}

// yesOrNo wraps a plain question about the file as a function returning one or
// nought, which is how libyara offers them.
func yesOrNo(answer bool) modValue {
	return funcOf(func(*evaluator, []value) (value, error) {
		if answer {
			return intValue(1), nil
		}
		return intValue(0), nil
	})
}

// addImports lists the libraries the file borrows from.
func (f *peFile) addImports(fields map[string]modValue, imports []peImport) {
	borrowed := 0
	items := make([]modValue, 0, len(imports))
	for _, dll := range imports {
		borrowed += len(dll.functions)
		functions := make([]modValue, 0, len(dll.functions))
		for _, fn := range dll.functions {
			member := map[string]modValue{}
			if fn.name != "" {
				member["name"] = valueOf(stringValue(fn.name))
			}
			// A thing borrowed by name has no number to report, so the field is
			// left out rather than being nought.
			if !fn.byName {
				member["ordinal"] = peInt(uint32(fn.ordinal))
			}
			functions = append(functions, structOf(member))
		}
		items = append(items, structOf(map[string]modValue{
			"library_name":        valueOf(stringValue(dll.name)),
			"number_of_functions": valueOf(intValue(int64(len(dll.functions)))),
			"functions":           listOf(functions),
		}))
	}
	fields["import_details"] = listOf(items)
	fields["number_of_imports"] = valueOf(intValue(int64(len(imports))))
	fields["number_of_imported_functions"] = valueOf(intValue(int64(borrowed)))
}

// addDelayedImportCounts counts what the file borrows only when it first needs
// it. The build CyberChef runs offers the counts but not the details.
func (f *peFile) addDelayedImportCounts(fields map[string]modValue) {
	delayed := f.readImports(directoryDelayed)
	borrowed := 0
	for _, dll := range delayed {
		borrowed += len(dll.functions)
	}
	fields["number_of_delayed_imports"] = valueOf(intValue(int64(len(delayed))))
	fields["number_of_delayed_imported_functions"] = valueOf(intValue(int64(borrowed)))
}

// readImports walks the list of libraries a file borrows from, which ends at a
// blank entry.
func (f *peFile) readImports(which uint32) []peImport {
	rva, _, ok := f.directoryEntry(which)
	if !ok {
		return nil
	}
	at, found := f.rvaToOffset(uint64(rva))
	if !found {
		return nil
	}

	var out []peImport
	for i := range maxImports {
		start := int(at) + i*importDescriptorSize
		nameRVA, nameOK := f.u32(start + 12)
		if !nameOK || nameRVA == 0 {
			break
		}
		name, named := f.nameAt(uint64(nameRVA))
		if !named || !printableName(name) {
			continue
		}
		// A library names where its list of things borrowed is twice over, and
		// either will do. A list that is not named, or cannot be read, leaves
		// the library with nothing to offer.
		thunks, _ := f.u32(start)
		if thunks == 0 {
			thunks, _ = f.u32(start + 16)
		}
		if thunks == 0 {
			continue
		}
		functions := f.readThunks(uint64(thunks), name)
		if len(functions) == 0 {
			continue
		}
		out = append(out, peImport{name: name, functions: functions})
	}
	return out
}

// readThunks reads the list of things borrowed from one library, which ends at
// a blank entry.
func (f *peFile) readThunks(rva uint64, dll string) []peImportedFunction {
	at, ok := f.rvaToOffset(rva)
	if !ok {
		return nil
	}
	step := 4
	if f.wide {
		step = 8
	}

	var out []peImportedFunction
	for i := range maxImports {
		thunk, byNumber, read := f.readThunk(int(at) + i*step)
		if !read || thunk == 0 {
			break
		}
		if byNumber {
			number := uint16(thunk & ordinalMask) // #nosec G115 -- masked to two bytes
			out = append(out, peImportedFunction{
				name: ordinalName(dll, number), ordinal: number,
			})
			continue
		}
		// A thing borrowed by name is written as a hint and then the name.
		name, named := f.nameAt(thunk + 2)
		if !named || !printableName(name) {
			continue
		}
		out = append(out, peImportedFunction{name: name, byName: true})
	}
	return out
}

// readThunk reads one entry of the list, saying whether it names a thing by
// number rather than by name.
func (f *peFile) readThunk(at int) (value uint64, byNumber, ok bool) {
	if f.wide {
		n, read := f.u64(at)
		return n &^ ordinalFlag64, n&ordinalFlag64 != 0, read
	}
	n, read := f.u32(at)
	return uint64(n &^ ordinalFlag32), n&ordinalFlag32 != 0, read
}

// nameAt reads a name kept at an address in memory.
func (f *peFile) nameAt(rva uint64) (string, bool) {
	at, ok := f.rvaToOffset(rva)
	if !ok {
		return "", false
	}
	for i := int(at); i < len(f.data); i++ {
		if f.data[i] == 0 {
			return string(f.data[at:i]), true
		}
	}
	return "", false
}

// printableName reports whether a name is one libyara would keep, which is a
// run of ordinary characters with something in it.
func printableName(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		if name[i] < 0x20 || name[i] > 0x7E {
			return false
		}
	}
	return true
}

// ordinalName is what libyara calls a thing borrowed by number: the name that
// number stands for in the libraries it has a table for, and otherwise the
// number itself.
func ordinalName(dll string, number uint16) string {
	key := strings.ToLower(dll)
	// The two socket libraries share one table.
	if key == "wsock32.dll" {
		key = "ws2_32.dll"
	}
	if table, known := ordinalNames[key]; known {
		if name, there := table[number]; there {
			return name
		}
	}
	return "ord" + strconv.FormatUint(uint64(number), 10)
}

// importHash is the digest libyara takes over everything a file borrows: each
// library and name in turn, separated by commas, lowercased, with the usual
// extensions cut off the library's name.
func importHash(imports []peImport) string {
	var b strings.Builder
	first := true
	for _, dll := range imports {
		name := dll.name
		if cut := strings.LastIndex(name, "."); cut >= 0 {
			switch strings.ToLower(name[cut:]) {
			case ".ocx", ".sys", ".dll":
				name = name[:cut]
			}
		}
		for _, fn := range dll.functions {
			if !first {
				b.WriteString(",")
			}
			b.WriteString(name + "." + fn.name)
			first = false
		}
	}
	sum := md5.Sum([]byte(strings.ToLower(b.String()))) // #nosec G401 -- the digest is defined as md5
	return hex.EncodeToString(sum[:])
}

// countImports answers what a rule asked about what the file borrows: how many
// things match, or whether a named one is there at all.
func countImports(imports []peImport, args []value) int64 {
	// A leading number says which kinds of borrowing to count, which this build
	// offers but does not act on differently.
	if len(args) > 0 && args[0].kind == valueInt {
		args = args[1:]
	}
	if len(args) == 0 {
		return 0
	}

	var count int64
	for _, dll := range imports {
		if !importMatches(args[0], dll.name) {
			continue
		}
		if len(args) == 1 {
			count += int64(len(dll.functions))
			continue
		}
		for _, fn := range dll.functions {
			if functionMatches(args[1], fn) {
				count++
			}
		}
	}
	return count
}

// importMatches says whether a library's name is the one asked about, which may
// be written out or given as a pattern. A name written out is compared without
// regard to case, as Windows does.
func importMatches(asked value, name string) bool {
	switch asked.kind {
	case valueString:
		return strings.EqualFold(asked.s, name)
	case valueRegex:
		return asked.re != nil && asked.re.MatchString(name)
	}
	return false
}

// functionMatches says whether one thing borrowed is the one asked about, by
// name, by pattern, or by the number it was borrowed under.
func functionMatches(asked value, fn peImportedFunction) bool {
	switch asked.kind {
	case valueString:
		return asked.s == fn.name
	case valueRegex:
		return asked.re != nil && asked.re.MatchString(fn.name)
	case valueInt:
		return !fn.byName && int64(fn.ordinal) == asked.i
	}
	return false
}

// readExports reads the table of what a file lends out.
func (f *peFile) readExports() []peExport {
	rva, size, ok := f.directoryEntry(directoryExports)
	if !ok {
		return nil
	}
	start, found := f.rvaToOffset(uint64(rva))
	if !found {
		return nil
	}
	at := int(start)
	if at+exportDirectorySize > len(f.data) {
		return nil
	}

	base, _ := f.u32(at + 16)
	count, _ := f.u32(at + 20)
	nameCount, _ := f.u32(at + 24)
	addressesRVA, _ := f.u32(at + 28)
	namesRVA, _ := f.u32(at + 32)
	ordinalsRVA, _ := f.u32(at + 36)
	count = min(count, maxExports)
	nameCount = min(nameCount, count)

	addresses, addressesOK := f.rvaToOffset(uint64(addressesRVA))
	if !addressesOK {
		return nil
	}
	names, _ := f.rvaToOffset(uint64(namesRVA))
	ordinals, ordinalsOK := f.rvaToOffset(uint64(ordinalsRVA))

	out := make([]peExport, 0, count)
	for i := range int(count) {
		address, read := f.u32(int(addresses) + i*4)
		if !read {
			break
		}
		export := peExport{ordinal: base + uint32(i)} // #nosec G115 -- bounded by maxExports
		f.placeExport(&export, address, start, size)
		if namesRVA != 0 && ordinalsOK {
			export.name, export.named = f.exportName(i, int(names), int(ordinals), int(nameCount))
		}
		out = append(out, export)
	}
	return out
}

// placeExport works out where one thing lent out actually is: somewhere in the
// file, or a name pointing on to another library.
func (f *peFile) placeExport(export *peExport, address uint32, start int64, size uint32) {
	at, found := f.rvaToOffset(uint64(address))
	// An address landing inside the table itself is not code at all: it names
	// another library to go and ask.
	if found && at > start && at < start+int64(size) {
		if name, named := f.nameAtOffset(int(at)); named {
			export.forwardName, export.forwarded = name, true
			return
		}
	}
	if found {
		export.offset, export.hasOffset = at, true
	}
}

// exportName finds the name given to the thing lent out in a given place, which
// is listed the other way round: by name, each saying which place it belongs to.
func (f *peFile) exportName(which, names, ordinals, nameCount int) (string, bool) {
	for j := range nameCount {
		place, read := f.u16(ordinals + j*2)
		if !read || int(place) != which {
			continue
		}
		nameRVA, gotRVA := f.u32(names + j*4)
		if !gotRVA {
			return "", false
		}
		return f.nameAt(uint64(nameRVA))
	}
	return "", false
}

// nameAtOffset reads a name kept at a place in the file.
func (f *peFile) nameAtOffset(at int) (string, bool) {
	if at < 0 || at >= len(f.data) {
		return "", false
	}
	for i := at; i < len(f.data); i++ {
		if f.data[i] == 0 {
			return string(f.data[at:i]), true
		}
	}
	return "", false
}

// addExports lists what the file lends out.
func (f *peFile) addExports(fields map[string]modValue, exports []peExport) {
	items := make([]modValue, 0, len(exports))
	for _, export := range exports {
		member := map[string]modValue{"ordinal": peInt(export.ordinal)}
		if export.named {
			member["name"] = valueOf(stringValue(export.name))
		}
		if export.forwarded {
			member["forward_name"] = valueOf(stringValue(export.forwardName))
		}
		if export.hasOffset {
			member["offset"] = valueOf(intValue(export.offset))
		}
		items = append(items, structOf(member))
	}
	fields["export_details"] = listOf(items)
	fields["number_of_exports"] = valueOf(intValue(int64(len(exports))))

	if rva, _, ok := f.directoryEntry(directoryExports); ok {
		if at, found := f.rvaToOffset(uint64(rva)); found {
			if when, read := f.u32(int(at) + 4); read {
				fields["export_timestamp"] = peInt(when)
			}
		}
	}
}

// findExport is where in the table a thing lent out sits, found by its name, by
// a pattern, or by the number it is lent out under.
func findExport(exports []peExport, args []value) int {
	if len(args) != 1 {
		return -1
	}
	for i, export := range exports {
		switch args[0].kind {
		case valueString:
			if export.named && export.name == args[0].s {
				return i
			}
		case valueRegex:
			if export.named && args[0].re != nil && args[0].re.MatchString(export.name) {
				return i
			}
		case valueInt:
			if int64(export.ordinal) == args[0].i {
				return i
			}
		}
	}
	return -1
}

// regexValue wraps a pattern written into a condition so that a module function
// can be handed it.
func regexValue(re *regexp.Regexp) value {
	return value{kind: valueRegex, re: re}
}

// What a PE file carries alongside its code.
//
// Resources are kept in a tree three deep: by kind, then by name or number,
// then by language. Each level is a directory whose entries either lead to
// another directory or to the thing itself. One kind of resource holds a block
// of text saying what the file claims to be, which is read out into a table a
// rule can look names up in.

// Which entry of the directory of tables holds what, for the parts read here.
const (
	directoryResources = 2
	directoryDebug     = 6
)

// The shapes the resource tree is written in.
const (
	// resourceDirSize is the header of one level, and resourceEntrySize one of
	// its entries.
	resourceDirSize   = 16
	resourceEntrySize = 8
	// resourceDataSize is the record naming where a thing carried actually is.
	resourceDataSize = 16
	// highBit marks an entry that leads to another directory rather than to a
	// thing, and a name written out rather than a number.
	highBit = 0x80000000
	// resourceDepth is how many levels down the things themselves sit.
	resourceDepth = 3
	// maxResources is as many as libyara will read.
	maxResources = 65536
	// versionResource is the kind of resource holding what a file claims to be.
	versionResource = 16
	// versionInfoSkip is how far past the start of that block libyara steps to
	// reach the first thing inside it, over the name and the run of numbers
	// that follows it.
	versionInfoSkip = 92
	// versionHeaderSize is the opening of one block within it: how long it is,
	// how long its value is, and what kind of value that is. The name follows,
	// and what the block holds follows that.
	versionHeaderSize = 6
	// stringFileInfoSkip is how far past a block of names its first table sits,
	// which libyara fixes rather than working out.
	stringFileInfoSkip = 30
	// versionAlign is what every part of the block is lined up to, which the
	// lengths written into it do not count.
	versionAlign = 4
	// debugRecordSize is one record naming where a file's debugging
	// information was written, and codeViewSkip how far into what it points at
	// the path itself begins.
	debugRecordSize = 28
	codeViewSkip    = 24
	// debugCodeView is the kind of debugging record that names a path.
	debugCodeView = 2
)

// resourceSchema adds to what the pe module declares the parts to do with what
// a file carries.
func resourceSchema(members map[string]*modDecl) {
	members["number_of_resources"] = decInt()
	members["resource_timestamp"] = decInt()
	members["resource_version"] = peVersionDecl()
	members["resources"] = decArray(decStruct(map[string]*modDecl{
		"rva": decInt(), "offset": decInt(), "length": decInt(),
		"type": decInt(), "id": decInt(), "language": decInt(),
		"type_string": decString(), "name_string": decString(),
		"language_string": decString(),
	}))
	members["pdb_path"] = decString()
	members["number_of_version_infos"] = decInt()
	members["version_info"] = decDict(decString())
	members["version_info_list"] = decArray(decStruct(map[string]*modDecl{
		"key": decString(), "value": decString(),
	}))
}

// peResource is one thing a file carries.
type peResource struct {
	rva, length uint32
	offset      int64
	hasOffset   bool
	// Each level of the tree either numbers a thing or names it, never both.
	level [resourceDepth]resourceLabel
}

// resourceLabel is what one level of the tree says about a thing: a number, or
// a name kept as the bytes it was written in.
type resourceLabel struct {
	number  uint32
	name    string
	isName  bool
	present bool
}

// addResources lists what the file carries and reads what it claims to be.
func (f *peFile) addResources(fields map[string]modValue) {
	rva, _, ok := f.directoryEntry(directoryResources)
	if !ok {
		fields["number_of_resources"] = valueOf(intValue(0))
		return
	}
	base, found := f.rvaToOffset(uint64(rva))
	if !found {
		fields["number_of_resources"] = valueOf(intValue(0))
		return
	}

	if when, read := f.u32(int(base) + 4); read {
		fields["resource_timestamp"] = peInt(when)
	}
	major, _ := f.u16(int(base) + 8)
	minor, _ := f.u16(int(base) + 10)
	fields["resource_version"] = structOf(map[string]modValue{
		"major": peInt(major), "minor": peInt(minor),
	})

	var carried []peResource
	f.walkResources(int(base), int(base), 0, resourceLabel{}, resourceLabel{}, &carried)

	items := make([]modValue, 0, len(carried))
	for _, r := range carried {
		items = append(items, resourceValue(r))
	}
	fields["resources"] = listOf(items)
	fields["number_of_resources"] = valueOf(intValue(int64(len(carried))))
	f.addVersionInfo(fields, carried)
}

// resourceValue is one thing carried, as the module offers it.
func resourceValue(r peResource) modValue {
	member := map[string]modValue{"rva": peInt(r.rva), "length": peInt(r.length)}
	if r.hasOffset {
		member["offset"] = valueOf(intValue(r.offset))
	}
	// Each level either numbers the thing or names it, and the two are not
	// always called the same: the middle level is numbered by "id" but named by
	// "name_string".
	levels := []struct{ number, name string }{
		{"type", "type_string"}, {"id", "name_string"}, {"language", "language_string"},
	}
	for at, level := range levels {
		label := r.level[at]
		switch {
		case !label.present:
		case label.isName:
			member[level.name] = valueOf(stringValue(label.name))
		default:
			member[level.number] = peInt(label.number)
		}
	}
	return structOf(member)
}

// walkResources goes down the tree, carrying what each level said about the
// thing until it reaches the thing itself.
func (f *peFile) walkResources(
	base, at, depth int, kind, id resourceLabel, out *[]peResource,
) {
	if depth >= resourceDepth || len(*out) >= maxResources {
		return
	}
	named, namedOK := f.u16(at + 12)
	numbered, numberedOK := f.u16(at + 14)
	if !namedOK || !numberedOK {
		return
	}

	for i := range int(named + numbered) {
		entry := at + resourceDirSize + i*resourceEntrySize
		label, labelOK := f.resourceLabel(base, entry)
		leadsTo, leadsOK := f.u32(entry + 4)
		if !labelOK || !leadsOK {
			return
		}
		f.walkResourceEntry(base, depth, label, leadsTo, kind, id, out)
	}
}

// walkResourceEntry follows one entry: on to another level, or to the thing
// itself.
func (f *peFile) walkResourceEntry(
	base, depth int, label resourceLabel, leadsTo uint32,
	kind, id resourceLabel, out *[]peResource,
) {
	if leadsTo&highBit != 0 {
		next := base + int(leadsTo&^highBit)
		switch depth {
		case 0:
			f.walkResources(base, next, depth+1, label, resourceLabel{}, out)
		default:
			f.walkResources(base, next, depth+1, kind, label, out)
		}
		return
	}
	f.addResourceLeaf(base+int(leadsTo), kind, id, label, out)
}

// addResourceLeaf reads the record naming where one thing carried actually is.
func (f *peFile) addResourceLeaf(
	at int, kind, id, language resourceLabel, out *[]peResource,
) {
	if at < 0 || at+resourceDataSize > len(f.data) {
		return
	}
	rva, _ := f.u32(at)
	length, _ := f.u32(at + 4)
	r := peResource{rva: rva, length: length, level: [resourceDepth]resourceLabel{
		kind, id, language,
	}}
	if offset, found := f.rvaToOffset(uint64(rva)); found {
		r.offset, r.hasOffset = offset, true
	}
	*out = append(*out, r)
}

// resourceLabel reads what one entry says about the thing it leads to: a
// number, or a name kept elsewhere in the tree.
func (f *peFile) resourceLabel(base, entry int) (resourceLabel, bool) {
	raw, ok := f.u32(entry)
	if !ok {
		return resourceLabel{}, false
	}
	if raw&highBit == 0 {
		return resourceLabel{number: raw, present: true}, true
	}
	at := base + int(raw&^highBit)
	length, lengthOK := f.u16(at)
	if !lengthOK {
		return resourceLabel{}, true
	}
	// A name is kept as a length and then the characters, two bytes each, and
	// libyara reports those bytes as they stand.
	from, to := at+2, at+2+int(length)*2
	if from > len(f.data) || to > len(f.data) {
		return resourceLabel{}, true
	}
	return resourceLabel{name: string(f.data[from:to]), isName: true, present: true}, true
}

// addVersionInfo reads the block saying what the file claims to be, out of the
// resource of that kind.
func (f *peFile) addVersionInfo(fields map[string]modValue, carried []peResource) {
	for _, r := range carried {
		if !r.level[0].present || r.level[0].isName ||
			r.level[0].number != versionResource || !r.hasOffset {
			continue
		}
		keys, values := f.readVersionInfo(int(r.offset))
		if len(keys) == 0 {
			continue
		}
		table := make(map[string]modValue, len(keys))
		items := make([]modValue, 0, len(keys))
		for i, key := range keys {
			table[key] = valueOf(stringValue(values[i]))
			items = append(items, structOf(map[string]modValue{
				"key":   valueOf(stringValue(key)),
				"value": valueOf(stringValue(values[i])),
			}))
		}
		fields["version_info"] = tableOf(table)
		fields["version_info_list"] = listOf(items)
		fields["number_of_version_infos"] = valueOf(intValue(int64(len(keys))))
		return
	}
}

// readVersionInfo walks the blocks of the version resource, gathering every
// name and what it is set to.
func (f *peFile) readVersionInfo(at int) (keys, values []string) {
	if opening, ok := f.wideString(at + versionHeaderSize); !ok ||
		opening != "VS_VERSION_INFO" {
		return nil, nil
	}
	// libyara steps a fixed distance past the opening, over the block of
	// numbers that follows the name.
	block := at + versionInfoSkip

	for {
		length, ok := f.u16(block)
		if !ok || length == 0 {
			return keys, values
		}
		name, named := f.wideString(block + versionHeaderSize)
		if !named {
			return keys, values
		}
		if name != "StringFileInfo" {
			block = lineUp(block + int(length))
			continue
		}
		f.readStringTables(block, block+int(length), &keys, &values)
		block = lineUp(block + int(length))
	}
}

// readStringTables walks the tables inside a block of names, each of which
// holds the names for one language.
func (f *peFile) readStringTables(block, end int, keys, values *[]string) {
	table := block + versionHeaderSize + stringFileInfoSkip
	for table < end {
		length, ok := f.u16(table)
		if !ok || length == 0 {
			return
		}
		name, named := f.wideString(table + versionHeaderSize)
		if !named {
			return
		}
		f.readStrings(lineUp(table+versionHeaderSize+2*(len(name)+1)),
			table+int(length), keys, values)
		table = lineUp(table + int(length))
	}
}

// readStrings walks the names in one table, each of which is a name and the
// text it is set to.
func (f *peFile) readStrings(at, end int, keys, values *[]string) {
	for at < end {
		length, ok := f.u16(at)
		valueLength, gotLength := f.u16(at + 2)
		if !ok || !gotLength || length == 0 {
			return
		}
		key, named := f.wideString(at + versionHeaderSize)
		if !named {
			return
		}
		*keys = append(*keys, key)
		*values = append(*values, f.versionValue(at, key, valueLength))
		at = lineUp(at + int(length))
	}
}

// versionValue is the text one name is set to, which follows the name once it
// has been lined up. A name set to nothing says so in its length, and what
// happens to follow it is not read as its value.
func (f *peFile) versionValue(at int, key string, valueLength uint32) string {
	if valueLength == 0 {
		return ""
	}
	text, hasText := f.wideString(lineUp(at + versionHeaderSize + 2*(len(key)+1)))
	if !hasText {
		return ""
	}
	return text
}

// lineUp rounds a place up to what the block is laid out on.
func lineUp(at int) int { return at + (versionAlign-at%versionAlign)%versionAlign }

// wideString reads text written two bytes to a character, ending in a nought.
// Only the ordinary characters are kept, which is what libyara does when it
// narrows one of these down.
func (f *peFile) wideString(at int) (string, bool) {
	if at < 0 || at >= len(f.data) {
		return "", false
	}
	var out []byte
	for i := at; i+1 < len(f.data); i += 2 {
		if f.data[i] == 0 && f.data[i+1] == 0 {
			return string(out), true
		}
		out = append(out, f.data[i])
		if len(out) > maxVersionName {
			return string(out), true
		}
	}
	return "", false
}

// maxVersionName is as long a name or value as libyara keeps.
const maxVersionName = 255

// addPdbPath reads where the file's debugging information was written, which is
// named by a record of its own.
func (f *peFile) addPdbPath(fields map[string]modValue) {
	rva, _, ok := f.directoryEntry(directoryDebug)
	if !ok {
		return
	}
	at, found := f.rvaToOffset(uint64(rva))
	if !found {
		return
	}
	if int(at)+debugRecordSize > len(f.data) {
		return
	}
	kind, kindOK := f.u32(int(at) + 12)
	if !kindOK || kind != debugCodeView {
		return
	}

	// The record says where the path is both as an address and as a place in
	// the file. libyara takes the address first and falls back to the place
	// whenever that comes to nothing — including when it comes to the very
	// start of the file, which nothing useful sits at.
	recordRVA, _ := f.u32(int(at) + 20)
	recordAt, gotOffset := f.rvaToOffset(uint64(recordRVA))
	if !gotOffset || recordAt <= 0 {
		raw, rawOK := f.u32(int(at) + 24)
		if !rawOK || raw == 0 {
			return
		}
		recordAt = int64(raw)
	}

	start := int(recordAt) + codeViewSkip
	if start < 0 || start >= len(f.data) {
		return
	}
	if path, read := f.nameAtOffset(start); read && path != "" {
		fields["pdb_path"] = valueOf(stringValue(path))
	}
}

// tableOf gathers a table, which is read by key.
func tableOf(entries map[string]modValue) modValue { return modValue{table: entries} }

// decDict declares a table read by key.
func decDict(item *modDecl) *modDecl { return &modDecl{kind: modDict, item: item} }
