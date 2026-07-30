package yara

import (
	"encoding/binary"
	"testing"
)

// The pe module is tested against files built here, so that both shapes of the
// optional header and the ways a file can be malformed can be tried
// deliberately. The values are what CyberChef gives for the same bytes.

// peBuilder lays out a PE file field by field.
type peBuilder struct {
	wide bool
	data []byte
	// opt is where the optional header begins, which everything after the
	// signature is written relative to.
	nt, opt int
}

// newPE lays out the stub, the signature and a header of the given width.
func newPE(wide bool) *peBuilder {
	b := &peBuilder{wide: wide}
	const stubSize = dosHeaderSize
	b.data = make([]byte, stubSize)
	b.data[0], b.data[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(b.data[newHeaderAt:], stubSize)

	b.nt = stubSize
	b.opt = b.nt + peSignatureSize + fileHeaderSize
	optSize := 96 + 16*dataDirectoryEntry
	magic := uint16(optionalMagic32)
	if wide {
		optSize = 112 + 16*dataDirectoryEntry
		magic = optionalMagic64
	}
	b.data = append(b.data, peSignature...)
	b.data = append(b.data, make([]byte, fileHeaderSize+optSize)...)
	b.put16(b.opt, magic)
	// The header has to say how long the optional header is, since the section
	// table follows it.
	b.put16(b.nt+peSignatureSize+16, uint16(optSize))
	return b
}

func (b *peBuilder) put16(at int, n uint16) { binary.LittleEndian.PutUint16(b.data[at:], n) }
func (b *peBuilder) put32(at int, n uint32) { binary.LittleEndian.PutUint32(b.data[at:], n) }
func (b *peBuilder) put64(at int, n uint64) { binary.LittleEndian.PutUint64(b.data[at:], n) }

// putAddress writes a number as wide as the file's own addresses.
func (b *peBuilder) putAddress(at int, n uint64) {
	if b.wide {
		b.put64(at, n)
		return
	}
	b.put32(at, uint32(n))
}

// directoriesAt is where the table of tables begins.
func (b *peBuilder) directoriesAt() int {
	if b.wide {
		return b.opt + 112
	}
	return b.opt + 96
}

// peSectionEntry is what one entry of the section table holds while a test file
// is being laid out.
type peSectionEntry struct {
	name                        string
	virtualSize, virtualAddress uint32
	rawSize, rawOffset          uint32
	relocations, lines          uint32
	numRelocations, numLines    uint16
	characteristics             uint32
}

// sectionTable writes a section table after the optional header.
func (b *peBuilder) sectionTable(entries []peSectionEntry) {
	b.put16(b.nt+peSignatureSize+2, uint16(len(entries)))
	for _, e := range entries {
		row := make([]byte, sectionSize)
		copy(row, e.name)
		binary.LittleEndian.PutUint32(row[8:], e.virtualSize)
		binary.LittleEndian.PutUint32(row[12:], e.virtualAddress)
		binary.LittleEndian.PutUint32(row[16:], e.rawSize)
		binary.LittleEndian.PutUint32(row[20:], e.rawOffset)
		binary.LittleEndian.PutUint32(row[24:], e.relocations)
		binary.LittleEndian.PutUint32(row[28:], e.lines)
		binary.LittleEndian.PutUint16(row[32:], e.numRelocations)
		binary.LittleEndian.PutUint16(row[34:], e.numLines)
		binary.LittleEndian.PutUint32(row[36:], e.characteristics)
		b.data = append(b.data, row...)
	}
}

// pad lengthens the file to a given size, so that offsets a section names are
// actually within it.
func (b *peBuilder) pad(to int) {
	for len(b.data) < to {
		b.data = append(b.data, 0)
	}
}

// scanPE builds the module over some bytes and reads one field out of it.
func scanPE(t *testing.T, data []byte, path string) value {
	t.Helper()
	e := &evaluator{buf: newBuffer(data), vars: map[string]int64{}, matched: map[string]bool{}}
	set, err := Parse(`import "pe" rule R { condition: ` + path + ` == 0 }`)
	if err != nil {
		t.Fatalf("parse %q: %v", path, err)
	}
	ref, ok := set.Rules[0].Condition.(Binary).L.(ModuleRef)
	if !ok {
		t.Fatalf("%q did not read as a reference into a module", path)
	}
	got, err := e.evalModuleRef(ref)
	if err != nil {
		t.Fatalf("reading %q: %v", path, err)
	}
	return got
}

// TestPENotAPE covers data that is not a PE file, which says so plainly rather
// than leaving the question unanswered.
func TestPENotAPE(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"nothing at all", nil},
		{"too short to say", []byte("MZ")},
		{"no stub", append([]byte("NO"), make([]byte, 100)...)},
		{"a header pointing past the end", func() []byte {
			d := make([]byte, dosHeaderSize)
			d[0], d[1] = 'M', 'Z'
			binary.LittleEndian.PutUint32(d[newHeaderAt:], 1<<20)
			return d
		}()},
		{"no signature where the stub says", func() []byte {
			d := make([]byte, 200)
			d[0], d[1] = 'M', 'Z'
			binary.LittleEndian.PutUint32(d[newHeaderAt:], dosHeaderSize)
			return d
		}()},
		{"an optional header of no known shape", func() []byte {
			b := newPE(false)
			b.put16(b.opt, 0x999)
			return b.data
		}()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantInt(t, scanPE(t, c.data, "pe.is_pe"), 0, "whether it is a PE file")
			wantNothing(t, scanPE(t, c.data, "pe.machine"), "the machine")
			wantInt(t, scanPE(t, c.data, "pe.MACHINE_AMD64"), 0x8664, "the constant for x86-64")
		})
	}
}

// TestPEFileHeader covers the header that names the machine and counts what
// follows.
func TestPEFileHeader(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(peWidth(wide), func(t *testing.T) {
			b := newPE(wide)
			at := b.nt + peSignatureSize
			b.put16(at, 0x8664)
			b.put16(at+2, 4)
			b.put32(at+4, 0x5F000000)
			b.put32(at+8, 0x1234)
			b.put32(at+12, 7)
			b.put16(at+18, 0x0102)

			wantInt(t, scanPE(t, b.data, "pe.is_pe"), 1, "whether it is a PE file")
			wantInt(t, scanPE(t, b.data, "pe.machine"), 0x8664, "the machine")
			wantInt(t, scanPE(t, b.data, "pe.number_of_sections"), 4, "how many sections")
			wantInt(t, scanPE(t, b.data, "pe.timestamp"), 0x5F000000, "when it was built")
			wantInt(t, scanPE(t, b.data, "pe.pointer_to_symbol_table"), 0x1234, "where its symbols are")
			wantInt(t, scanPE(t, b.data, "pe.number_of_symbols"), 7, "how many symbols")
			wantInt(t, scanPE(t, b.data, "pe.characteristics"), 0x0102, "what kind of file it is")
		})
	}
}

func peWidth(wide bool) string {
	if wide {
		return "built for 64-bit addresses"
	}
	return "built for 32-bit addresses"
}

// TestPEOptionalHeader covers how the file is to be laid out once loaded. The
// two shapes keep the same fields, but one writes addresses in four bytes and
// the other in eight, so everything after the code's base moves.
func TestPEOptionalHeader(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(peWidth(wide), func(t *testing.T) {
			b := newPE(wide)
			at := b.opt
			b.data[at+2], b.data[at+3] = 14, 29
			b.put32(at+4, 0x1000)
			b.put32(at+8, 0x2000)
			b.put32(at+12, 0x3000)
			b.put32(at+20, 0x4000)
			imageBaseAt, stackAt, loaderAt := 28, 72, 88
			if wide {
				imageBaseAt, stackAt, loaderAt = 24, 72, 104
			} else {
				b.put32(at+24, 0x5000)
			}
			// The narrower shape has only four bytes to say where the file
			// loads, so it cannot name an address above four gigabytes.
			imageBase := uint64(0x400000)
			if wide {
				imageBase = 0x140000000
			}
			b.putAddress(at+imageBaseAt, imageBase)
			b.put32(at+32, 0x1000)
			b.put32(at+36, 0x200)
			b.put16(at+40, 6)
			b.put16(at+42, 1)
			b.put16(at+44, 2)
			b.put16(at+46, 3)
			b.put16(at+48, 6)
			b.put16(at+50, 0)
			b.put32(at+52, 0)
			b.put32(at+56, 0x9000)
			b.put32(at+60, 0x400)
			b.put32(at+64, 0xABCD)
			b.put16(at+68, 3)
			b.put16(at+70, 0x8160)
			step := 4
			if wide {
				step = 8
			}
			for i, n := range []uint64{0x100000, 0x1000, 0x100000, 0x1000} {
				b.putAddress(at+stackAt+i*step, n)
			}
			b.put32(at+loaderAt, 0)
			b.put32(at+loaderAt+4, 16)

			wantInt(t, scanPE(t, b.data, "pe.linker_version.major"), 14, "which linker built it")
			wantInt(t, scanPE(t, b.data, "pe.linker_version.minor"), 29, "which linker built it")
			wantInt(t, scanPE(t, b.data, "pe.size_of_code"), 0x1000, "how much code")
			wantInt(t, scanPE(t, b.data, "pe.size_of_initialized_data"), 0x2000, "how much data")
			wantInt(t, scanPE(t, b.data, "pe.size_of_uninitialized_data"), 0x3000, "how much space")
			wantInt(t, scanPE(t, b.data, "pe.base_of_code"), 0x4000, "where the code loads")
			wantInt(t, scanPE(t, b.data, "pe.image_base"), int64(imageBase), "where the file loads")
			wantInt(t, scanPE(t, b.data, "pe.section_alignment"), 0x1000, "how sections line up")
			wantInt(t, scanPE(t, b.data, "pe.file_alignment"), 0x200, "how the file lines up")
			wantInt(t, scanPE(t, b.data, "pe.os_version.major"), 6, "which system it needs")
			wantInt(t, scanPE(t, b.data, "pe.image_version.major"), 2, "its own version")
			wantInt(t, scanPE(t, b.data, "pe.subsystem_version.major"), 6, "which subsystem")
			wantInt(t, scanPE(t, b.data, "pe.size_of_image"), 0x9000, "its room in memory")
			wantInt(t, scanPE(t, b.data, "pe.size_of_headers"), 0x400, "how much is header")
			wantInt(t, scanPE(t, b.data, "pe.checksum"), 0xABCD, "the checksum it carries")
			wantInt(t, scanPE(t, b.data, "pe.subsystem"), 3, "what it runs under")
			wantInt(t, scanPE(t, b.data, "pe.dll_characteristics"), 0x8160, "how it may be loaded")
			wantInt(t, scanPE(t, b.data, "pe.size_of_stack_reserve"), 0x100000, "its stack")
			wantInt(t, scanPE(t, b.data, "pe.size_of_heap_reserve"), 0x100000, "its heap")
			wantInt(t, scanPE(t, b.data, "pe.number_of_rva_and_sizes"), 16, "how many tables")

			// The base of the data is only written by the narrower shape, since
			// the wider one uses those bytes for the address the file loads at.
			if wide {
				wantNothing(t, scanPE(t, b.data, "pe.base_of_data"), "where the data loads")
			} else {
				wantInt(t, scanPE(t, b.data, "pe.base_of_data"), 0x5000, "where the data loads")
			}
		})
	}
}

// TestPEDataDirectories covers the table saying where each of the file's own
// tables is to be found.
func TestPEDataDirectories(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(peWidth(wide), func(t *testing.T) {
			b := newPE(wide)
			_, _, loaderAt := 28, 72, 88
			if wide {
				loaderAt = 104
			}
			b.put32(b.opt+loaderAt+4, 16)
			at := b.directoriesAt()
			b.put32(at, 0x5000)
			b.put32(at+4, 0x100)
			b.put32(at+dataDirectoryEntry, 0x6000)
			b.put32(at+dataDirectoryEntry+4, 0x200)

			wantInt(t, scanPE(t, b.data, "pe.data_directories[0].virtual_address"), 0x5000,
				"where the first table is")
			wantInt(t, scanPE(t, b.data, "pe.data_directories[0].size"), 0x100,
				"how big the first table is")
			wantInt(t, scanPE(t, b.data, "pe.data_directories[1].virtual_address"), 0x6000,
				"where the second table is")
			wantNothing(t, scanPE(t, b.data, "pe.data_directories[20].size"),
				"a table past the end")
		})
	}
}

// TestPEDataDirectoriesBeyondTheFormat covers a file claiming more tables than
// the format has room to name, which is cut back rather than read past.
func TestPEDataDirectoriesBeyondTheFormat(t *testing.T) {
	b := newPE(true)
	b.put32(b.opt+108, 1000)
	wantInt(t, scanPE(t, b.data, "pe.data_directories[0].size"), 0, "the first table")
	wantNothing(t, scanPE(t, b.data, "pe.data_directories[16].size"),
		"a table the format has no room to name")
}

// TestPESections covers the section table and the names it holds. A name may
// fill its whole room, leaving no nought after it.
func TestPESections(t *testing.T) {
	b := newPE(true)
	b.sectionTable([]peSectionEntry{
		{
			name: ".text", virtualSize: 0x800, virtualAddress: 0x1000,
			rawSize: 0x600, rawOffset: 0x400, characteristics: 0x60000020,
			relocations: 3, lines: 4, numRelocations: 5, numLines: 6,
		},
		{
			name: ".rdata", virtualSize: 0x200, virtualAddress: 0x2000,
			rawSize: 0x200, rawOffset: 0xA00,
		},
		{name: "12345678", virtualSize: 0x100, virtualAddress: 0x3000},
	})
	b.pad(0xC00)

	if got := scanPE(t, b.data, "pe.sections[0].name"); got.s != ".text" {
		t.Errorf("the first section is named %q, want %q", got.s, ".text")
	}
	if got := scanPE(t, b.data, "pe.sections[0].full_name"); got.s != ".text" {
		t.Errorf("its full name is %q, want %q", got.s, ".text")
	}
	if got := scanPE(t, b.data, "pe.sections[2].name"); got.s != "12345678" {
		t.Errorf("a name filling its room read as %q, want %q", got.s, "12345678")
	}
	wantInt(t, scanPE(t, b.data, "pe.sections[0].virtual_address"), 0x1000, "where it loads")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].virtual_size"), 0x800, "its room in memory")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].raw_data_offset"), 0x400, "where it sits")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].raw_data_size"), 0x600, "its room on disk")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].characteristics"), 0x60000020, "what it holds")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].pointer_to_relocations"), 3, "its fixups")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].pointer_to_line_numbers"), 4, "its line numbers")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].number_of_relocations"), 5, "how many fixups")
	wantInt(t, scanPE(t, b.data, "pe.sections[0].number_of_line_numbers"), 6, "how many lines")
	wantNothing(t, scanPE(t, b.data, "pe.sections[9].name"), "a section past the end")
}

// TestPESectionsBeyondTheFile covers a file claiming more sections than it has
// room for, which stops at what is there.
func TestPESectionsBeyondTheFile(t *testing.T) {
	b := newPE(true)
	b.put16(b.nt+peSignatureSize+2, 500)
	b.sectionTable([]peSectionEntry{{name: ".text", virtualAddress: 0x1000}})
	b.put16(b.nt+peSignatureSize+2, 500)
	if got := scanPE(t, b.data, "pe.sections[0].name"); got.s != ".text" {
		t.Errorf("the section that is there read as %q", got.s)
	}
	wantNothing(t, scanPE(t, b.data, "pe.sections[1].name"), "a section that is not there")
}

// TestPEOverlay covers whatever follows the last section. A file with nothing
// after its sections reports nought for both, rather than leaving them out.
func TestPEOverlay(t *testing.T) {
	t.Run("something after the last section", func(t *testing.T) {
		b := newPE(true)
		b.sectionTable([]peSectionEntry{
			{name: ".text", rawOffset: 0x400, rawSize: 0x200},
		})
		b.pad(0x700)
		wantInt(t, scanPE(t, b.data, "pe.overlay.offset"), 0x600, "where the overlay starts")
		wantInt(t, scanPE(t, b.data, "pe.overlay.size"), 0x100, "how big it is")
	})
	t.Run("nothing after the last section", func(t *testing.T) {
		b := newPE(true)
		b.sectionTable([]peSectionEntry{
			{name: ".text", rawOffset: 0x400, rawSize: 0x200},
		})
		b.pad(0x600)
		wantInt(t, scanPE(t, b.data, "pe.overlay.offset"), 0, "where the overlay starts")
		wantInt(t, scanPE(t, b.data, "pe.overlay.size"), 0, "how big it is")
	})
	t.Run("no sections at all", func(t *testing.T) {
		b := newPE(true)
		wantInt(t, scanPE(t, b.data, "pe.overlay.offset"), 0, "where the overlay starts")
		wantInt(t, scanPE(t, b.data, "pe.overlay.size"), 0, "how big it is")
	})
	t.Run("two sections starting together", func(t *testing.T) {
		// It is the end each section reaches that decides, not its length, so
		// the longer of two starting together wins.
		b := newPE(true)
		b.sectionTable([]peSectionEntry{
			{name: ".a", rawOffset: 0x400, rawSize: 0x400},
			{name: ".b", rawOffset: 0x400, rawSize: 0x100},
		})
		b.pad(0x900)
		wantInt(t, scanPE(t, b.data, "pe.overlay.offset"), 0x800, "where the overlay starts")
	})
}

// TestPERvaToOffset covers turning an address in memory into the place in the
// file it came from.
func TestPERvaToOffset(t *testing.T) {
	b := newPE(true)
	b.put32(b.opt+32, 0x1000) // sections line up to a page
	b.put32(b.opt+36, 0x200)  // the file lines up to a sector
	b.sectionTable([]peSectionEntry{
		{
			name: ".text", virtualAddress: 0x1000, virtualSize: 0x400,
			rawOffset: 0x400, rawSize: 0x400,
		},
	})
	b.pad(0x800)

	wantInt(t, scanPE(t, b.data, "pe.rva_to_offset(0x1000)"), 0x400, "the start of the section")
	wantInt(t, scanPE(t, b.data, "pe.rva_to_offset(0x1010)"), 0x410, "part way into it")
	// Everything before the first section is laid out just as it is on disk.
	wantInt(t, scanPE(t, b.data, "pe.rva_to_offset(0x100)"), 0x100, "before the first section")
	wantNothing(t, scanPE(t, b.data, "pe.rva_to_offset(0x9000)"), "an address in no section")
	wantNothing(t, scanPE(t, b.data, "pe.rva_to_offset(-1)"), "an address below nought")
}

// TestPERvaToOffsetOfSparseSpace covers a section with less room on disk than
// in memory, whose later addresses came from nowhere in the file.
func TestPERvaToOffsetOfSparseSpace(t *testing.T) {
	b := newPE(true)
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{
		{
			name: ".bss", virtualAddress: 0x1000, virtualSize: 0x2000,
			rawOffset: 0x400, rawSize: 0x200,
		},
	})
	b.pad(0x600)
	wantInt(t, scanPE(t, b.data, "pe.rva_to_offset(0x1000)"), 0x400, "where it does come from")
	wantNothing(t, scanPE(t, b.data, "pe.rva_to_offset(0x2000)"), "room with nothing behind it")
}

// TestPEEntryPoint covers where a program starts. An address that came from
// nowhere in the file is reported as -1 rather than as no answer, which is what
// libyara stores.
func TestPEEntryPoint(t *testing.T) {
	t.Run("an address a section holds", func(t *testing.T) {
		b := newPE(true)
		b.put32(b.opt+16, 0x1020)
		b.put32(b.opt+32, 0x1000)
		b.put32(b.opt+36, 0x200)
		b.sectionTable([]peSectionEntry{
			{
				name: ".text", virtualAddress: 0x1000, virtualSize: 0x400,
				rawOffset: 0x400, rawSize: 0x400,
			},
		})
		b.pad(0x800)
		wantInt(t, scanPE(t, b.data, "pe.entry_point_raw"), 0x1020, "the address it starts at")
		wantInt(t, scanPE(t, b.data, "pe.entry_point"), 0x420, "where that is in the file")
	})
	t.Run("an address from nowhere", func(t *testing.T) {
		b := newPE(true)
		b.put32(b.opt+16, 0x900000)
		b.sectionTable([]peSectionEntry{
			{
				name: ".text", virtualAddress: 0x1000, virtualSize: 0x400,
				rawOffset: 0x400, rawSize: 0x400,
			},
		})
		b.pad(0x800)
		wantInt(t, scanPE(t, b.data, "pe.entry_point"), -1, "a start from nowhere")
	})
}

// TestPESectionIndex covers finding a section by name or by an address it
// covers.
func TestPESectionIndex(t *testing.T) {
	b := newPE(true)
	b.sectionTable([]peSectionEntry{
		{name: ".text", virtualAddress: 0x1000, virtualSize: 0x400},
		{name: ".data", virtualAddress: 0x2000, virtualSize: 0x400},
	})

	wantInt(t, scanPE(t, b.data, `pe.section_index(".text")`), 0, "the first section")
	wantInt(t, scanPE(t, b.data, `pe.section_index(".data")`), 1, "the second section")
	wantInt(t, scanPE(t, b.data, "pe.section_index(0x2010)"), 1, "the section holding an address")
	wantNothing(t, scanPE(t, b.data, `pe.section_index(".nope")`), "a section that is not there")
	wantNothing(t, scanPE(t, b.data, "pe.section_index(0x9000)"), "an address in no section")
	wantNothing(t, scanPE(t, b.data, "pe.section_index(-1)"), "an address below nought")
}

// TestPEChecksum covers working the file's checksum out again, which is a
// running sum over the whole file with the recorded checksum left out.
func TestPEChecksum(t *testing.T) {
	b := newPE(true)
	b.sectionTable([]peSectionEntry{{name: ".text", rawOffset: 0x400, rawSize: 0x10}})
	b.pad(0x411)

	got := scanPE(t, b.data, "pe.calculate_checksum()")
	if got.kind != valueInt {
		t.Fatalf("the checksum came to %v, want a number", got.kind)
	}
	// Working it out again over the same bytes has to give the same answer, and
	// changing a byte has to change it.
	again := scanPE(t, b.data, "pe.calculate_checksum()")
	if again.i != got.i {
		t.Errorf("the same file checksummed as %d and then %d", got.i, again.i)
	}
	changed := append([]byte{}, b.data...)
	changed[0x405] ^= 0xFF
	if other := scanPE(t, changed, "pe.calculate_checksum()"); other.i == got.i {
		t.Error("changing a byte left the checksum alone")
	}
	// The checksum the file carries is not part of the sum, so changing it
	// cannot change the answer.
	carried := append([]byte{}, b.data...)
	binary.LittleEndian.PutUint32(carried[b.opt+checksumSkip:], 0xDEADBEEF)
	if other := scanPE(t, carried, "pe.calculate_checksum()"); other.i != got.i {
		t.Errorf("the carried checksum changed the worked-out one: %d, want %d", other.i, got.i)
	}
}

// TestPEReadingPastTheEnd covers the readers themselves asked for bytes the
// file does not have.
func TestPEReadingPastTheEnd(t *testing.T) {
	f := &peFile{data: []byte{1, 2, 3}}
	if _, ok := f.u8(9); ok {
		t.Error("read a byte that is not there")
	}
	if _, ok := f.u8(-1); ok {
		t.Error("read a byte before the start")
	}
	if got, ok := f.u8(1); !ok || got != 2 {
		t.Errorf("a byte in the file read as %d %v", got, ok)
	}
	if _, ok := f.u16(2); ok {
		t.Error("read two bytes that are not there")
	}
	if _, ok := f.u32(0); ok {
		t.Error("read four bytes that are not there")
	}
	if _, ok := f.u64(0); ok {
		t.Error("read eight bytes that are not there")
	}
	if _, ok := f.address(0); ok {
		t.Error("read an address that is not there")
	}
}

// TestPENameFillingItsRoom covers a name written into a fixed run of bytes with
// no nought after it.
func TestPENameFillingItsRoom(t *testing.T) {
	if got := indexOfNul([]byte("abcd")); got != -1 {
		t.Errorf("a name filling its room ended at %d, want -1", got)
	}
	if got := indexOfNul([]byte("ab\x00d")); got != 2 {
		t.Errorf("a name ended at %d, want 2", got)
	}
}

// TestPEHeadersThatRunOut covers headers the file stops in the middle of, which
// are read as far as they go rather than past the end. A rule cannot reach
// these through a well-formed file, so they are set up directly.
func TestPEHeadersThatRunOut(t *testing.T) {
	t.Run("an optional header cut short", func(t *testing.T) {
		b := newPE(true)
		b.data = b.data[:b.opt+2]
		if _, ok := readPE(b.data); !ok {
			t.Error("refused a file whose optional header is cut short")
		}
	})
	t.Run("no room for the count of tables", func(t *testing.T) {
		b := newPE(true)
		b.data = b.data[:b.opt+2]
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		fields := map[string]modValue{}
		f.addDataDirectories(fields)
		if _, there := fields["data_directories"]; there {
			t.Error("listed tables a file with no count of them")
		}
	})
	t.Run("no room for the length of the optional header", func(t *testing.T) {
		f := &peFile{data: make([]byte, 4)}
		if got := f.sectionsAt(); got != -1 {
			t.Errorf("found a section table at %d, want none", got)
		}
		if got := f.readSections(); got != nil {
			t.Errorf("read %d sections from a file that says nothing", len(got))
		}
	})
	t.Run("no room for the count of sections", func(t *testing.T) {
		b := newPE(true)
		b.put16(b.nt+peSignatureSize+2, 3)
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		// The table is named but the entries are not there, so none are read.
		if got := len(f.sections); got != 0 {
			t.Errorf("read %d sections that are not there", got)
		}
	})
}

// TestPESectionIndexGivenNothing covers asking which section is meant without
// saying, which has no answer.
func TestPESectionIndexGivenNothing(t *testing.T) {
	b := newPE(true)
	b.sectionTable([]peSectionEntry{{name: ".text", virtualAddress: 0x1000, virtualSize: 0x10}})
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	if got := f.sectionIndex(nil); got.kind != valueUndefined {
		t.Errorf("found a section without being told which, %v", got.kind)
	}
	if got := f.sectionIndex([]value{floatValue(1)}); got.kind != valueUndefined {
		t.Errorf("found a section named by a fraction, %v", got.kind)
	}
}

// TestPERvaToOffsetOfANegativeAddress covers an address below nought handed
// straight to the conversion, which no file holds.
func TestPERvaToOffsetOfANegativeAddress(t *testing.T) {
	b := newPE(true)
	b.sectionTable([]peSectionEntry{
		{
			name: ".text", virtualAddress: 0x1000, virtualSize: 0x400,
			rawOffset: 0x400, rawSize: 0x400,
		},
	})
	b.pad(0x800)
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	// A section named after one already found does not replace it, so a file
	// listing its sections out of order keeps the earliest match.
	if _, found := f.rvaToOffset(1 << 40); found {
		t.Error("found a place in the file for an address far past its end")
	}
}

// TestPEOptionalHeaderWithNoMagic covers a file that stops before saying which
// shape its optional header takes, which cannot be read as a PE file at all.
func TestPEOptionalHeaderWithNoMagic(t *testing.T) {
	b := newPE(true)
	b.data = b.data[:b.opt+1]
	if _, ok := readPE(b.data); ok {
		t.Error("read a file that never says which shape its header takes")
	}
}

// TestPEDataDirectoriesRunningOut covers a file naming more tables than it has
// the bytes for, which stops at the last one that is there.
func TestPEDataDirectoriesRunningOut(t *testing.T) {
	b := newPE(true)
	b.put32(b.opt+108, 16)
	// Cut the file so that only the first two entries of the table are there.
	b.data = b.data[:b.directoriesAt()+2*dataDirectoryEntry]
	b.put32(b.directoriesAt(), 0x5000)
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	fields := map[string]modValue{}
	f.addDataDirectories(fields)
	if got := len(fields["data_directories"].list); got != 2 {
		t.Errorf("listed %d tables, want the 2 that are there", got)
	}
}

// TestPERvaToOffsetPastTheEnd covers a section whose contents are said to sit
// past the end of the file, so an address in it came from nowhere.
func TestPERvaToOffsetPastTheEnd(t *testing.T) {
	b := newPE(true)
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{
		{
			name: ".text", virtualAddress: 0x1000, virtualSize: 0x400,
			rawOffset: 0x100000, rawSize: 0x400,
		},
	})
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	if _, found := f.rvaToOffset(0x1000); found {
		t.Error("found a place past the end of the file")
	}
}

// The run of bytes Microsoft's linker leaves in the stub at the front of a PE
// file, saying which tools built it and how many times each was used. It is
// hidden behind a running exclusive-or with a key kept at the end of it.

// withRichSignature writes the run into the stub, ending just before the header
// proper, and stretches the stub to fit.
func (b *peBuilder) withRichSignature(key uint32, tools []richTool) {
	// The run opens with a marker and three blanks, then a pair of numbers per
	// tool, and closes with another marker and the key.
	body := make([]byte, 16)
	binary.LittleEndian.PutUint32(body, richDans)
	for _, tool := range tools {
		body = binary.LittleEndian.AppendUint32(body,
			uint32(tool.id)<<16|uint32(tool.version))
		body = binary.LittleEndian.AppendUint32(body, tool.times)
	}
	for i := 0; i < len(body); i += 4 {
		n := binary.LittleEndian.Uint32(body[i:])
		binary.LittleEndian.PutUint32(body[i:], n^key)
	}
	closing := make([]byte, 8)
	binary.LittleEndian.PutUint32(closing, richRich)
	binary.LittleEndian.PutUint32(closing[4:], key)

	// The run sits after the stub's own header and before the header proper,
	// so everything after it moves along.
	at := dosHeaderSize
	rest := append([]byte{}, b.data[at:]...)
	b.data = append(b.data[:at], body...)
	b.data = append(b.data, closing...)
	moved := len(body) + len(closing)
	b.data = append(b.data, rest...)

	b.nt += moved
	b.opt += moved
	binary.LittleEndian.PutUint32(b.data[newHeaderAt:], uint32(b.nt))
}

// TestPERichSignature covers the run itself: where it is, how long, its key,
// and the bytes both as written and once the key is taken off again.
func TestPERichSignature(t *testing.T) {
	const key = 0x12345678
	b := newPE(true)
	b.withRichSignature(key, []richTool{
		{id: 0x00DB, version: 0x1FE8, times: 3},
		{id: 0x0102, version: 0x0004, times: 7},
	})

	wantInt(t, scanPE(t, b.data, "pe.rich_signature.offset"), dosHeaderSize,
		"where the run begins")
	// Four blanks and two pairs of numbers, sixteen bytes and sixteen more.
	wantInt(t, scanPE(t, b.data, "pe.rich_signature.length"), 32, "how long it is")
	wantInt(t, scanPE(t, b.data, "pe.rich_signature.key"), key, "the key it is hidden behind")

	raw := scanPE(t, b.data, "pe.rich_signature.raw_data")
	plain := scanPE(t, b.data, "pe.rich_signature.clear_data")
	if len(raw.s) != 32 || len(plain.s) != 32 {
		t.Fatalf("the run reads as %d bytes written and %d plain, want 32 of each",
			len(raw.s), len(plain.s))
	}
	// Taking the key off the written bytes gives the plain ones.
	for i := 0; i+4 <= len(raw.s); i += 4 {
		written := binary.LittleEndian.Uint32([]byte(raw.s[i : i+4]))
		want := binary.LittleEndian.Uint32([]byte(plain.s[i : i+4]))
		if written^key != want {
			t.Errorf("byte %d: %#x with the key off is %#x, want %#x",
				i, written, written^key, want)
		}
	}
	if binary.LittleEndian.Uint32([]byte(plain.s[:4])) != richDans {
		t.Error("the plain bytes do not open with the marker they should")
	}
}

// TestPERichSignatureCounts covers asking how often a tool was used, by the
// version it was, by which tool it was, or by both together.
func TestPERichSignatureCounts(t *testing.T) {
	b := newPE(true)
	b.withRichSignature(0x0F0F0F0F, []richTool{
		{id: 0x00DB, version: 0x1FE8, times: 3},
		{id: 0x0102, version: 0x1FE8, times: 7},
		{id: 0x00DB, version: 0x0004, times: 5},
	})

	cases := []struct {
		src  string
		want int64
	}{
		// By the version, however many tools were of it.
		{"pe.rich_signature.version(0x1FE8)", 10},
		{"pe.rich_signature.version(0x0004)", 5},
		{"pe.rich_signature.version(0x9999)", 0},
		// By which tool it was, however many versions of it.
		{"pe.rich_signature.toolid(0x00DB)", 8},
		{"pe.rich_signature.toolid(0x0102)", 7},
		{"pe.rich_signature.toolid(0x9999)", 0},
		// And by both at once.
		{"pe.rich_signature.version(0x1FE8, 0x00DB)", 3},
		{"pe.rich_signature.version(0x1FE8, 0x0102)", 7},
		{"pe.rich_signature.toolid(0x00DB, 0x1FE8)", 3},
		{"pe.rich_signature.toolid(0x00DB, 0x0004)", 5},
		{"pe.rich_signature.version(0x0004, 0x0102)", 0},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			wantInt(t, scanPE(t, b.data, c.src), c.want, "how often it was used")
		})
	}
}

// TestPERichSignatureOfAFileWithoutOne covers a file whose stub holds no such
// run, which leaves every part of it without an answer.
func TestPERichSignatureOfAFileWithoutOne(t *testing.T) {
	b := newPE(true)
	for _, part := range []string{"offset", "length", "key", "raw_data", "clear_data"} {
		wantNothing(t, scanPE(t, b.data, "pe.rich_signature."+part), part)
	}
	wantNothing(t, scanPE(t, b.data, "pe.rich_signature.version(1)"),
		"how often a tool was used")
}

// TestPERichSignatureWithoutItsOpening covers a stub holding the closing marker
// and a key but nothing the key opens, which is no run at all.
func TestPERichSignatureWithoutItsOpening(t *testing.T) {
	b := newPE(true)
	b.withRichSignature(0x11111111, []richTool{{id: 1, version: 2, times: 3}})
	// Spoiling the opening marker leaves nothing for the key to find.
	b.data[dosHeaderSize] ^= 0xFF
	wantNothing(t, scanPE(t, b.data, "pe.rich_signature.length"), "how long the run is")
}

// TestPERichSignatureNamingNoTools covers a run with the markers and a key but
// no tools between them, which has a length but nothing to count.
func TestPERichSignatureNamingNoTools(t *testing.T) {
	b := newPE(true)
	b.withRichSignature(0x22222222, nil)
	wantInt(t, scanPE(t, b.data, "pe.rich_signature.length"), 16, "how long it is")
	wantInt(t, scanPE(t, b.data, "pe.rich_signature.version(1)"), 0, "how often a tool was used")
}

// What a PE file borrows from elsewhere and what it lends out. Both are tables
// reached through the directory of tables, and both name things either by name
// or by number.

// importedDLL is one library a test file borrows from.
type importedDLL struct {
	name      string
	functions []importedFunction
}

// importedFunction is one thing borrowed, named either way.
type importedFunction struct {
	name    string
	ordinal uint16
}

// withImports lays out an import table and points the directory of tables at
// it. Everything is placed in one section so that addresses in memory and
// places in the file line up simply.
func (b *peBuilder) withImports(dlls []importedDLL) {
	const sectionRVA, sectionOffset = 0x1000, 0x400
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{{
		name: ".idata", virtualAddress: sectionRVA, virtualSize: 0x1000,
		rawOffset: sectionOffset, rawSize: 0x1000,
	}})
	b.pad(sectionOffset)

	// The section is laid out in one buffer and written in at the end, so that
	// an address can be worked out before what it points at is placed.
	body := make([]byte, 0, 0x1000)
	at := func() uint32 { return sectionRVA + uint32(len(body)) }
	put32 := func(dst []byte, off int, n uint32) { binary.LittleEndian.PutUint32(dst[off:], n) }

	// The descriptors come first, one per library and a blank one to close the
	// list, so that what they point at can follow.
	descriptors := make([]byte, (len(dlls)+1)*20)
	body = append(body, descriptors...)

	for i, dll := range dlls {
		nameAt := at()
		body = append(body, dll.name...)
		body = append(body, 0)

		// Each name a library lends out is written as a hint and then the name.
		places := make([]uint32, 0, len(dll.functions))
		for _, fn := range dll.functions {
			if fn.name == "" {
				places = append(places, 0)
				continue
			}
			places = append(places, at())
			body = append(body, 0, 0)
			body = append(body, fn.name...)
			body = append(body, 0)
		}
		if len(body)%2 == 1 {
			body = append(body, 0)
		}

		thunksAt := at()
		for j, fn := range dll.functions {
			if fn.name == "" {
				// A thing borrowed by number has the top bit set and the number
				// in the bottom half.
				if b.wide {
					body = binary.LittleEndian.AppendUint64(body,
						0x8000000000000000|uint64(fn.ordinal))
					continue
				}
				body = binary.LittleEndian.AppendUint32(body, 0x80000000|uint32(fn.ordinal))
				continue
			}
			if b.wide {
				body = binary.LittleEndian.AppendUint64(body, uint64(places[j]))
				continue
			}
			body = binary.LittleEndian.AppendUint32(body, places[j])
		}
		if b.wide {
			body = binary.LittleEndian.AppendUint64(body, 0)
		} else {
			body = binary.LittleEndian.AppendUint32(body, 0)
		}

		put32(body, i*20, thunksAt)    // where the names are listed
		put32(body, i*20+12, nameAt)   // the library's own name
		put32(body, i*20+16, thunksAt) // and where they end up once loaded
	}

	b.data = append(b.data, body...)
	b.pad(sectionOffset + 0x1000)
	// The second table of the directory is the one that lists what is borrowed.
	b.put32(b.directoriesAt()+dataDirectoryEntry, sectionRVA)
	b.put32(b.directoriesAt()+dataDirectoryEntry+4, uint32(len(descriptors)))
	loaderAt := 88
	if b.wide {
		loaderAt = 104
	}
	b.put32(b.opt+loaderAt+4, 16)
}

// TestPEImports covers the libraries a file borrows from and the things it
// borrows, named either way.
func TestPEImports(t *testing.T) {
	for _, wide := range []bool{false, true} {
		t.Run(peWidth(wide), func(t *testing.T) {
			b := newPE(wide)
			b.withImports([]importedDLL{
				{name: "KERNEL32.dll", functions: []importedFunction{
					{name: "CreateFileW"}, {name: "ExitProcess"},
				}},
				{name: "WS2_32.dll", functions: []importedFunction{
					{ordinal: 1}, {ordinal: 115},
				}},
			})

			wantInt(t, scanPE(t, b.data, "pe.number_of_imports"), 2, "how many libraries")
			wantInt(t, scanPE(t, b.data, "pe.number_of_imported_functions"), 4,
				"how many things borrowed")

			if got := scanPE(t, b.data, "pe.import_details[0].library_name"); got.s != "KERNEL32.dll" {
				t.Errorf("the first library is %q, want %q", got.s, "KERNEL32.dll")
			}
			wantInt(t, scanPE(t, b.data, "pe.import_details[0].number_of_functions"), 2,
				"how many from the first library")
			if got := scanPE(t, b.data, "pe.import_details[0].functions[0].name"); got.s != "CreateFileW" {
				t.Errorf("the first thing borrowed is %q, want %q", got.s, "CreateFileW")
			}
			if got := scanPE(t, b.data, "pe.import_details[1].library_name"); got.s != "WS2_32.dll" {
				t.Errorf("the second library is %q, want %q", got.s, "WS2_32.dll")
			}
			// A thing borrowed by number from a library libyara knows is given
			// the name that number stands for.
			if got := scanPE(t, b.data, "pe.import_details[1].functions[0].name"); got.s != "accept" {
				t.Errorf("number 1 of the sockets library is %q, want %q", got.s, "accept")
			}
			wantInt(t, scanPE(t, b.data, "pe.import_details[1].functions[0].ordinal"), 1,
				"the number it was borrowed by")
			if got := scanPE(t, b.data, "pe.import_details[1].functions[1].name"); got.s != "WSAStartup" {
				t.Errorf("number 115 is %q, want %q", got.s, "WSAStartup")
			}
			wantNothing(t, scanPE(t, b.data, "pe.import_details[9].library_name"),
				"a library past the end")
		})
	}
}

// TestPEImportsByNumberFromAnUnknownLibrary covers a thing borrowed by number
// from a library libyara has no table for, which is named after the number.
func TestPEImportsByNumberFromAnUnknownLibrary(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "custom.dll", functions: []importedFunction{{ordinal: 42}}},
	})
	if got := scanPE(t, b.data, "pe.import_details[0].functions[0].name"); got.s != "ord42" {
		t.Errorf("number 42 of an unknown library is %q, want %q", got.s, "ord42")
	}
}

// TestPEImportsAsked covers asking whether something is borrowed, which a rule
// may do by name, by number, or by a pattern.
func TestPEImportsAsked(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{
			{name: "CreateFileW"}, {name: "ExitProcess"},
		}},
		{name: "USER32.dll", functions: []importedFunction{{name: "MessageBoxW"}}},
	})

	cases := []struct {
		src  string
		want int64
	}{
		{`pe.imports("KERNEL32.dll", "CreateFileW")`, 1},
		{`pe.imports("kernel32.DLL", "CreateFileW")`, 1},
		{`pe.imports("KERNEL32.dll", "Missing")`, 0},
		{`pe.imports("Nope.dll", "CreateFileW")`, 0},
		{`pe.imports("KERNEL32.dll")`, 2},
		{`pe.imports("USER32.dll")`, 1},
		{`pe.imports("Nope.dll")`, 0},
		{`pe.imports(/KERNEL32/i, /CreateFile/)`, 1},
		{`pe.imports(/32\.dll/i, /./)`, 3},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			wantInt(t, scanPE(t, b.data, c.src), c.want, "what was asked")
		})
	}
}

// TestPEImphash covers the digest of what a file borrows, which is taken over
// every library and name in turn, lowercased, with the usual extensions cut off.
func TestPEImphash(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{
			{name: "CreateFileW"}, {name: "ExitProcess"},
		}},
		{name: "USER32.dll", functions: []importedFunction{{name: "MessageBoxW"}}},
	})

	// The digest is taken over
	// "kernel32.createfilew,kernel32.exitprocess,user32.messageboxw".
	got := scanPE(t, b.data, "pe.imphash()")
	if got.kind != valueString {
		t.Fatalf("the digest came to %v, want text", got.kind)
	}

	// Taking it again over the same file has to give the same answer, and a
	// file borrowing something else has to give a different one.
	if again := scanPE(t, b.data, "pe.imphash()"); again.s != got.s {
		t.Errorf("the same file hashed as %q and then %q", got.s, again.s)
	}
	other := newPE(true)
	other.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "CreateFileW"}}},
	})
	if o := scanPE(t, other.data, "pe.imphash()"); o.s == got.s {
		t.Error("two files borrowing different things hashed the same")
	}
	// The extension is cut off, so naming the same library another way cannot
	// change the answer.
	renamed := newPE(true)
	renamed.withImports([]importedDLL{
		{name: "KERNEL32.ocx", functions: []importedFunction{
			{name: "CreateFileW"}, {name: "ExitProcess"},
		}},
		{name: "USER32.sys", functions: []importedFunction{{name: "MessageBoxW"}}},
	})
	if r := scanPE(t, renamed.data, "pe.imphash()"); r.s != got.s {
		t.Errorf("cutting the extension changed the digest: %q, want %q", r.s, got.s)
	}
}

// TestPEImphashOfAFileBorrowingNothing covers a file with no import table,
// whose digest is that of nothing at all.
func TestPEImphashOfAFileBorrowingNothing(t *testing.T) {
	b := newPE(true)
	got := scanPE(t, b.data, "pe.imphash()")
	// d41d8... is the digest of no bytes at all, which is what libyara gives.
	if got.kind != valueString || got.s != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("a file borrowing nothing hashed as %q", got.s)
	}
	wantInt(t, scanPE(t, b.data, "pe.number_of_imports"), 0, "how many libraries")
}

// exportedFunction is one thing a test file lends out.
type exportedFunction struct {
	name    string
	address uint32
}

// withExports lays out an export table and points the directory of tables at
// it.
func (b *peBuilder) withExports(dllName string, base uint32, fns []exportedFunction) {
	const sectionRVA, sectionOffset = 0x1000, 0x400
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{{
		name: ".edata", virtualAddress: sectionRVA, virtualSize: 0x1000,
		rawOffset: sectionOffset, rawSize: 0x1000,
	}})
	b.pad(sectionOffset)

	body := make([]byte, 40) // the directory itself comes first
	at := func() uint32 { return sectionRVA + uint32(len(body)) }
	put32 := func(off int, n uint32) { binary.LittleEndian.PutUint32(body[off:], n) }

	nameAt := at()
	body = append(body, dllName...)
	body = append(body, 0)

	namePlaces := make([]uint32, 0, len(fns))
	for _, fn := range fns {
		if fn.name == "" {
			namePlaces = append(namePlaces, 0)
			continue
		}
		namePlaces = append(namePlaces, at())
		body = append(body, fn.name...)
		body = append(body, 0)
	}

	addressesAt := at()
	for _, fn := range fns {
		body = binary.LittleEndian.AppendUint32(body, fn.address)
	}
	namesAt := at()
	named := 0
	for i, fn := range fns {
		if fn.name == "" {
			continue
		}
		named++
		body = binary.LittleEndian.AppendUint32(body, namePlaces[i])
	}
	ordinalsAt := at()
	for i, fn := range fns {
		if fn.name == "" {
			continue
		}
		body = binary.LittleEndian.AppendUint16(body, uint16(i))
	}

	put32(4, 0x5F5F5F5F) // when it was built
	put32(12, nameAt)
	put32(16, base)
	put32(20, uint32(len(fns)))
	put32(24, uint32(named))
	put32(28, addressesAt)
	put32(32, namesAt)
	put32(36, ordinalsAt)

	b.data = append(b.data, body...)
	b.pad(sectionOffset + 0x1000)
	// The first table of the directory is the one that lists what is lent out.
	b.put32(b.directoriesAt(), sectionRVA)
	b.put32(b.directoriesAt()+4, 40)
	loaderAt := 88
	if b.wide {
		loaderAt = 104
	}
	b.put32(b.opt+loaderAt+4, 16)
}

// TestPEExports covers what a file lends out, by name and by number.
func TestPEExports(t *testing.T) {
	b := newPE(true)
	b.withExports("sample.dll", 1, []exportedFunction{
		{name: "Alpha", address: 0x2000},
		{name: "Beta", address: 0x2100},
		{address: 0x2200},
	})

	wantInt(t, scanPE(t, b.data, "pe.number_of_exports"), 3, "how many things lent out")
	wantInt(t, scanPE(t, b.data, "pe.export_timestamp"), 0x5F5F5F5F, "when it was built")
	if got := scanPE(t, b.data, "pe.export_details[0].name"); got.s != "Alpha" {
		t.Errorf("the first is named %q, want %q", got.s, "Alpha")
	}
	if got := scanPE(t, b.data, "pe.export_details[1].name"); got.s != "Beta" {
		t.Errorf("the second is named %q, want %q", got.s, "Beta")
	}
	// Numbers start from the base the file names, not from nought.
	wantInt(t, scanPE(t, b.data, "pe.export_details[0].ordinal"), 1, "the first number")
	wantInt(t, scanPE(t, b.data, "pe.export_details[2].ordinal"), 3, "the third number")
	// One lent out without a name has none, but is still counted.
	wantNothing(t, scanPE(t, b.data, "pe.export_details[2].name"), "a name that was never given")
	wantNothing(t, scanPE(t, b.data, "pe.export_details[9].name"), "one past the end")

	cases := []struct {
		src  string
		want int64
	}{
		{`pe.exports("Alpha")`, 1},
		{`pe.exports("Missing")`, 0},
		{`pe.exports(1)`, 1},
		{`pe.exports(9)`, 0},
		{`pe.exports(/^A/)`, 1},
		{`pe.exports_index("Beta")`, 1},
		{`pe.exports_index(3)`, 2},
		{`pe.exports_index(/^B/)`, 1},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			wantInt(t, scanPE(t, b.data, c.src), c.want, "what was asked")
		})
	}
	wantNothing(t, scanPE(t, b.data, `pe.exports_index("Missing")`), "one that is not lent out")
}

// TestPEExportsOfAFileLendingNothing covers a file with no export table.
func TestPEExportsOfAFileLendingNothing(t *testing.T) {
	b := newPE(true)
	wantInt(t, scanPE(t, b.data, "pe.number_of_exports"), 0, "how many things lent out")
	wantInt(t, scanPE(t, b.data, `pe.exports("Alpha")`), 0, "whether it lends one out")
	wantNothing(t, scanPE(t, b.data, "pe.export_timestamp"), "when it was built")
}

// TestPEWhatKindOfFile covers the three questions a rule may ask about the file
// as a whole.
func TestPEWhatKindOfFile(t *testing.T) {
	t.Run("a program built for 32-bit addresses", func(t *testing.T) {
		b := newPE(false)
		wantInt(t, scanPE(t, b.data, "pe.is_32bit()"), 1, "whether it is 32-bit")
		wantInt(t, scanPE(t, b.data, "pe.is_64bit()"), 0, "whether it is 64-bit")
		wantInt(t, scanPE(t, b.data, "pe.is_dll()"), 0, "whether it is a library")
	})
	t.Run("a library built for 64-bit addresses", func(t *testing.T) {
		b := newPE(true)
		// The flag saying a file is a library sits among the characteristics.
		b.put16(b.nt+peSignatureSize+18, 0x2000)
		wantInt(t, scanPE(t, b.data, "pe.is_32bit()"), 0, "whether it is 32-bit")
		wantInt(t, scanPE(t, b.data, "pe.is_64bit()"), 1, "whether it is 64-bit")
		// The answer is the flag itself, which libyara hands back as it is.
		wantInt(t, scanPE(t, b.data, "pe.is_dll()"), dllCharacteristic,
			"whether it is a library")
	})
	t.Run("something that is not a PE file at all", func(t *testing.T) {
		wantNothing(t, scanPE(t, []byte("nope"), "pe.is_dll()"), "whether it is a library")
	})
}

// TestPEImportsAndExportsThatRunOut covers tables that point past the end of
// the file, name things nothing can be read from, or are simply not there. A
// well-formed file cannot reach these, so they are set up directly.
func TestPEImportsAndExportsThatRunOut(t *testing.T) {
	t.Run("a table of libraries pointing nowhere", func(t *testing.T) {
		b := newPE(true)
		b.put32(b.opt+108, 16)
		b.put32(b.directoriesAt()+dataDirectoryEntry, 0x900000)
		wantInt(t, scanPE(t, b.data, "pe.number_of_imports"), 0, "how many libraries")
	})
	t.Run("a table of things lent out pointing nowhere", func(t *testing.T) {
		b := newPE(true)
		b.put32(b.opt+108, 16)
		b.put32(b.directoriesAt(), 0x900000)
		wantInt(t, scanPE(t, b.data, "pe.number_of_exports"), 0, "how many things lent out")
	})
	t.Run("what a file borrows only when it needs it", func(t *testing.T) {
		b := newPE(true)
		b.withImports([]importedDLL{
			{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
		})
		// The table for these sits elsewhere in the directory, and this file
		// names none.
		wantInt(t, scanPE(t, b.data, "pe.number_of_delayed_imports"), 0, "how many")
		wantInt(t, scanPE(t, b.data, "pe.number_of_delayed_imported_functions"), 0, "how many")
	})
	t.Run("a name with no end to it", func(t *testing.T) {
		f := &peFile{data: []byte("abc")}
		if _, ok := f.nameAtOffset(0); ok {
			t.Error("read a name that never ends")
		}
		if _, ok := f.nameAtOffset(-1); ok {
			t.Error("read a name from before the start")
		}
		if _, ok := f.nameAtOffset(9); ok {
			t.Error("read a name from past the end")
		}
	})
	t.Run("names that are not names", func(t *testing.T) {
		for _, name := range []string{"", "with\x01control", "with\x7fdelete"} {
			if printableName(name) {
				t.Errorf("%q was taken for a name", name)
			}
		}
		if !printableName("Ordinary.Name_1") {
			t.Error("an ordinary name was refused")
		}
	})
}

// TestPEImportsAskedOddly covers asking about what a file borrows in ways that
// have no answer, and with the leading number libyara allows.
func TestPEImportsAskedOddly(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "WS2_32.dll", functions: []importedFunction{{ordinal: 1}}},
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	// A leading number says which kinds of borrowing to count.
	wantInt(t, scanPE(t, b.data, `pe.imports(3, "WS2_32.dll")`), 1, "counted with a leading number")
	// Something borrowed by number can be asked for by that number.
	wantInt(t, scanPE(t, b.data, `pe.imports("WS2_32.dll", 1)`), 1, "asked by number")
	wantInt(t, scanPE(t, b.data, `pe.imports("WS2_32.dll", 9)`), 0, "a number not borrowed")
	// One borrowed by name is not found by number, whatever the number.
	wantInt(t, scanPE(t, b.data, `pe.imports("KERNEL32.dll", 0)`), 0, "a named one asked by number")

	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	imports := f.readImports(directoryImports)
	if got := countImports(imports, nil); got != 0 {
		t.Errorf("counted %d with nothing asked", got)
	}
	if got := countImports(imports, []value{floatValue(1)}); got != 0 {
		t.Errorf("counted %d for a library named by a fraction", got)
	}
	if got := countImports(imports, []value{stringValue("WS2_32.dll"), floatValue(1)}); got != 0 {
		t.Errorf("counted %d for a thing named by a fraction", got)
	}
	if got := findExport(nil, nil); got != -1 {
		t.Errorf("found something lent out at %d, want none", got)
	}
}

// TestPEExportsForwarded covers a thing lent out that points on to another
// library rather than to code in this file.
func TestPEExportsForwarded(t *testing.T) {
	b := newPE(true)
	// An address landing inside the table itself names another library. The
	// table starts at 0x1000 and is 40 bytes of header, so the name written
	// after it is inside the span the directory claims.
	b.withExports("sample.dll", 1, []exportedFunction{
		{name: "Onward", address: 0x1000 + 40},
	})
	b.put32(b.directoriesAt()+4, 0x200) // the table covers what follows it

	got := scanPE(t, b.data, "pe.export_details[0].forward_name")
	if got.kind != valueString || got.s == "" {
		t.Errorf("a thing pointing onward has no name to point to: %v %q", got.kind, got.s)
	}
}

// What a PE file carries alongside its code: icons, dialogs, and the block of
// text saying what the file claims to be. They are kept in a tree three deep —
// by kind, then by name or number, then by language — and the version block is
// read out of it into a table a rule can look names up in.

// resourceLeaf is one thing carried, as a test file lays it out.
type resourceLeaf struct {
	kind, id, language uint32
	// kindName and idName are set when that level names the thing rather than
	// numbering it.
	kindName, idName string
	body             string
}

// withResources lays out a resource tree and points the directory of tables at
// it.
func (b *peBuilder) withResources(when uint32, major, minor uint16, leaves []resourceLeaf) {
	const sectionRVA, sectionOffset = 0x1000, 0x400
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{{
		name: ".rsrc", virtualAddress: sectionRVA, virtualSize: 0x2000,
		rawOffset: sectionOffset, rawSize: 0x2000,
	}})
	b.pad(sectionOffset)

	r := &resourceWriter{base: sectionRVA}
	r.write(when, major, minor, leaves)
	b.data = append(b.data, r.body...)
	b.pad(sectionOffset + 0x2000)

	// The third table of the directory is the one holding what the file carries.
	b.put32(b.directoriesAt()+2*dataDirectoryEntry, sectionRVA)
	b.put32(b.directoriesAt()+2*dataDirectoryEntry+4, uint32(len(r.body)))
	loaderAt := 88
	if b.wide {
		loaderAt = 104
	}
	b.put32(b.opt+loaderAt+4, 16)
}

// resourceWriter lays out the tree, which points at itself by offsets from its
// own start.
type resourceWriter struct {
	base uint32
	body []byte
}

func (r *resourceWriter) at() uint32 { return uint32(len(r.body)) }

// pad4 lines the next thing up on a four-byte boundary, as the format expects.
func (r *resourceWriter) pad4() {
	for len(r.body)%4 != 0 {
		r.body = append(r.body, 0)
	}
}

// wide writes a name the way the tree keeps them: a length, then the characters
// two bytes each.
func (r *resourceWriter) wide(text string) uint32 {
	r.pad4()
	at := r.at()
	r.body = binary.LittleEndian.AppendUint16(r.body, uint16(len(text)))
	for _, c := range text {
		r.body = binary.LittleEndian.AppendUint16(r.body, uint16(c))
	}
	return at
}

// write lays out a tree three deep over the leaves given. Every leaf is put
// under its own branch at each level, which is enough to exercise the walk.
func (r *resourceWriter) write(when uint32, major, minor uint16, leaves []resourceLeaf) {
	// Room is left for the three levels of directory before anything they point
	// at, since an entry has to name where it leads.
	const dirSize, entrySize, dataSize = 16, 8, 16
	top := dirSize + len(leaves)*entrySize
	middles := top + len(leaves)*(dirSize+entrySize)
	datas := middles + len(leaves)*(dirSize+entrySize)
	r.body = make([]byte, datas+len(leaves)*dataSize)

	put32 := func(at int, n uint32) { binary.LittleEndian.PutUint32(r.body[at:], n) }
	put16 := func(at int, n uint16) { binary.LittleEndian.PutUint16(r.body[at:], n) }

	put32(4, when)
	put16(8, major)
	put16(10, minor)
	put16(14, uint16(len(leaves))) // all numbered at the top level for now

	for i, leaf := range leaves {
		body := r.at()
		r.body = append(r.body, leaf.body...)

		kind := leaf.kind
		if leaf.kindName != "" {
			kind = 0x80000000 | r.wide(leaf.kindName)
		}
		id := leaf.id
		if leaf.idName != "" {
			id = 0x80000000 | r.wide(leaf.idName)
		}

		// The top level names the kind and leads to a directory of its own.
		put32(dirSize+i*entrySize, kind)
		put32(dirSize+i*entrySize+4, 0x80000000|uint32(top+i*(dirSize+entrySize)))
		// The middle level names the thing and leads to one of languages.
		put16(top+i*(dirSize+entrySize)+14, 1)
		put32(top+i*(dirSize+entrySize)+dirSize, id)
		put32(top+i*(dirSize+entrySize)+dirSize+4,
			0x80000000|uint32(middles+i*(dirSize+entrySize)))
		// The bottom level names the language and leads to the thing itself.
		put16(middles+i*(dirSize+entrySize)+14, 1)
		put32(middles+i*(dirSize+entrySize)+dirSize, leaf.language)
		put32(middles+i*(dirSize+entrySize)+dirSize+4, uint32(datas+i*dataSize))
		// And the thing itself says where its bytes are and how many.
		put32(datas+i*dataSize, r.base+body)
		put32(datas+i*dataSize+4, uint32(len(leaf.body)))
	}
}

// TestPEResources covers what a file carries, numbered and named.
func TestPEResources(t *testing.T) {
	b := newPE(true)
	b.withResources(0x5E000000, 4, 2, []resourceLeaf{
		{kind: 3, id: 1, language: 1033, body: "an icon"},
		{kind: 10, id: 7, language: 2057, body: "some data"},
		{kindName: "MYKIND", idName: "MYNAME", language: 0, body: "named"},
	})

	wantInt(t, scanPE(t, b.data, "pe.number_of_resources"), 3, "how many things carried")
	wantInt(t, scanPE(t, b.data, "pe.resource_timestamp"), 0x5E000000, "when they were built")
	wantInt(t, scanPE(t, b.data, "pe.resource_version.major"), 4, "the version they are written to")
	wantInt(t, scanPE(t, b.data, "pe.resource_version.minor"), 2, "the version they are written to")

	wantInt(t, scanPE(t, b.data, "pe.resources[0].type"), 3, "what the first one is")
	wantInt(t, scanPE(t, b.data, "pe.resources[0].id"), 1, "its number")
	wantInt(t, scanPE(t, b.data, "pe.resources[0].language"), 1033, "its language")
	wantInt(t, scanPE(t, b.data, "pe.resources[0].length"), 7, "how long it is")
	wantInt(t, scanPE(t, b.data, "pe.resources[1].type"), 10, "what the second one is")
	wantInt(t, scanPE(t, b.data, "pe.resources[1].length"), 9, "how long it is")

	// One named rather than numbered has the name and not the number.
	wantNothing(t, scanPE(t, b.data, "pe.resources[2].type"), "a kind that was named")
	if got := scanPE(t, b.data, "pe.resources[2].type_string"); got.s != wideText("MYKIND") {
		t.Errorf("the third one's kind is %q, want the name written wide", got.s)
	}
	if got := scanPE(t, b.data, "pe.resources[2].name_string"); got.s != wideText("MYNAME") {
		t.Errorf("the third one's name is %q, want the name written wide", got.s)
	}
	wantNothing(t, scanPE(t, b.data, "pe.resources[9].type"), "one past the end")
}

// wideText is a name as the resource tree keeps it, two bytes to a character.
func wideText(text string) string {
	out := make([]byte, 0, len(text)*2)
	for _, c := range text {
		out = append(out, byte(c), 0)
	}
	return string(out)
}

// TestPEResourcesOfAFileCarryingNothing covers a file with no resource tree.
func TestPEResourcesOfAFileCarryingNothing(t *testing.T) {
	b := newPE(true)
	wantInt(t, scanPE(t, b.data, "pe.number_of_resources"), 0, "how many things carried")
	wantNothing(t, scanPE(t, b.data, "pe.resource_timestamp"), "when they were built")
	wantNothing(t, scanPE(t, b.data, "pe.resources[0].type"), "the first thing carried")
}

// TestPEVersionInfo covers the block of text saying what a file claims to be,
// which is read out of the resource of that kind into a table of names.
func TestPEVersionInfo(t *testing.T) {
	b := newPE(true)
	b.withResources(0, 0, 0, []resourceLeaf{
		{kind: 16, id: 1, language: 1033, body: versionBlock(map[string]string{
			"CompanyName":  "A Company",
			"FileVersion":  "1.2.3.4",
			"ProductName":  "A Product",
			"InternalName": "thing.exe",
		})},
	})

	cases := []struct{ key, want string }{
		{"CompanyName", "A Company"},
		{"FileVersion", "1.2.3.4"},
		{"ProductName", "A Product"},
		{"InternalName", "thing.exe"},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			got := scanPE(t, b.data, `pe.version_info["`+c.key+`"]`)
			if got.s != c.want {
				t.Errorf("%s is %q, want %q", c.key, got.s, c.want)
			}
		})
	}
	wantNothing(t, scanPE(t, b.data, `pe.version_info["Missing"]`), "a name that is not there")
	wantInt(t, scanPE(t, b.data, "pe.number_of_version_infos"), 4, "how many names")

	// The same names are also listed in the order they were written, so that a
	// rule may walk them.
	seen := map[string]string{}
	for i := range 4 {
		key := scanPE(t, b.data, "pe.version_info_list["+itoa(i)+"].key")
		val := scanPE(t, b.data, "pe.version_info_list["+itoa(i)+"].value")
		seen[key.s] = val.s
	}
	for _, c := range cases {
		if seen[c.key] != c.want {
			t.Errorf("the list gives %s as %q, want %q", c.key, seen[c.key], c.want)
		}
	}
}

// itoa is a small number written out, for building a reference in a test.
func itoa(n int) string { return string(rune('0' + n)) }

// versionBlock lays out the block of text a file uses to say what it is. The
// shape is fixed: an outer block, then one naming the language, then one entry
// per name.
func versionBlock(entries map[string]string) string {
	// The names are written in a settled order so the test can rely on it.
	order := []string{"CompanyName", "FileVersion", "InternalName", "ProductName"}

	var strings []byte
	for _, key := range order {
		text, there := entries[key]
		if !there {
			continue
		}
		strings = append(strings, versionEntry(key, text)...)
		// What follows an entry is lined up, and that padding is not counted in
		// the entry's own length, just as a real file lays it out.
		strings = alignTo4(strings)
	}

	// The block naming the language wraps the entries.
	table := versionHeader(len("08090000"), 0, 1, "08090000", len(strings))
	table = append(table, strings...)
	binary.LittleEndian.PutUint16(table, uint16(len(table)))

	// The block naming what kind of information this is wraps that.
	info := versionHeader(0, 0, 1, "StringFileInfo", len(table))
	info = append(info, table...)
	binary.LittleEndian.PutUint16(info, uint16(len(info)))

	// The outer block opens with a fixed run libyara steps straight over.
	outer := make([]byte, 6)
	outer = append(outer, wideBytes("VS_VERSION_INFO")...)
	// A real file keeps a run of numbers after the name, which libyara steps
	// straight over, so the same amount of room is left here.
	for len(outer) < versionInfoSkip {
		outer = append(outer, 0)
	}
	outer = append(outer, info...)
	binary.LittleEndian.PutUint16(outer, uint16(len(outer)))
	binary.LittleEndian.PutUint16(outer[2:], 0)
	binary.LittleEndian.PutUint16(outer[4:], 0)
	return string(outer)
}

// versionHeader is the opening of one block: how long it is, how long its value
// is, what kind of value that is, and the name it goes by.
func versionHeader(valueLength, _, kind int, key string, _ int) []byte {
	out := make([]byte, 6)
	binary.LittleEndian.PutUint16(out[2:], uint16(valueLength))
	binary.LittleEndian.PutUint16(out[4:], uint16(kind))
	return alignTo4(append(out, wideBytes(key)...))
}

// alignTo4 lengthens a run so the next thing after it begins on a boundary the
// format lines everything up to.
func alignTo4(out []byte) []byte {
	for len(out)%4 != 0 {
		out = append(out, 0)
	}
	return out
}

// versionEntry is one name and what it is set to.
func versionEntry(key, text string) []byte {
	out := make([]byte, 6)
	binary.LittleEndian.PutUint16(out[2:], uint16(len(text)+1))
	binary.LittleEndian.PutUint16(out[4:], 1)
	out = alignTo4(append(out, wideBytes(key)...))
	out = append(out, wideBytes(text)...)
	// The length runs to the end of the text, not past whatever pads it out.
	binary.LittleEndian.PutUint16(out, uint16(len(out)))
	return out
}

// wideBytes is text written two bytes to a character, ending in a nought.
func wideBytes(text string) []byte {
	out := make([]byte, 0, len(text)*2+2)
	for _, c := range text {
		out = binary.LittleEndian.AppendUint16(out, uint16(c))
	}
	return append(out, 0, 0)
}

// TestPEPdbPath covers the path to the file's debugging information, which is
// kept in a record of its own that the directory of tables points at.
func TestPEPdbPath(t *testing.T) {
	const path = `C:\build\thing.pdb`
	b := newPE(true)
	b.withDebugRecord(path)
	if got := scanPE(t, b.data, "pe.pdb_path"); got.s != path {
		t.Errorf("the path is %q, want %q", got.s, path)
	}
}

// withDebugRecord lays out a record naming where the debugging information for
// a file was written.
func (b *peBuilder) withDebugRecord(path string) {
	const sectionRVA, sectionOffset = 0x1000, 0x400
	b.put32(b.opt+32, 0x1000)
	b.put32(b.opt+36, 0x200)
	b.sectionTable([]peSectionEntry{{
		name: ".rdata", virtualAddress: sectionRVA, virtualSize: 0x1000,
		rawOffset: sectionOffset, rawSize: 0x1000,
	}})
	b.pad(sectionOffset)

	// The record itself comes first, then what it points at. Where that is is
	// given both as an address and as a place in the file.
	body := make([]byte, 28)
	binary.LittleEndian.PutUint32(body[12:], 2) // the kind that names a path
	binary.LittleEndian.PutUint32(body[20:], sectionRVA+uint32(len(body)))
	binary.LittleEndian.PutUint32(body[24:], sectionOffset+uint32(len(body)))

	cv := append([]byte("RSDS"), make([]byte, 20)...)
	cv = append(cv, path...)
	cv = append(cv, 0)
	body = append(body, cv...)

	b.data = append(b.data, body...)
	b.pad(sectionOffset + 0x1000)
	// The seventh table of the directory names where the debugging record is.
	b.put32(b.directoriesAt()+6*dataDirectoryEntry, sectionRVA)
	b.put32(b.directoriesAt()+6*dataDirectoryEntry+4, 28)
	loaderAt := 88
	if b.wide {
		loaderAt = 104
	}
	b.put32(b.opt+loaderAt+4, 16)
}

// TestPEPdbPathOfAFileWithoutOne covers a file naming no debugging information.
func TestPEPdbPathOfAFileWithoutOne(t *testing.T) {
	b := newPE(true)
	wantNothing(t, scanPE(t, b.data, "pe.pdb_path"), "a path that was never given")
}

// TestPEResourcesThatRunOut covers a resource tree that points past the end of
// the file, or claims more than is there. A well-formed file cannot reach
// these, so they are set up directly.
func TestPEResourcesThatRunOut(t *testing.T) {
	t.Run("a tree pointing nowhere", func(t *testing.T) {
		b := newPE(true)
		b.put32(b.opt+108, 16)
		b.put32(b.directoriesAt()+2*dataDirectoryEntry, 0x900000)
		wantInt(t, scanPE(t, b.data, "pe.number_of_resources"), 0, "how many things carried")
	})
	t.Run("a level claiming entries that are not there", func(t *testing.T) {
		b := newPE(true)
		b.withResources(0, 0, 0, []resourceLeaf{{kind: 3, id: 1, body: "x"}})
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		// A level right at the end of the file has nothing to read.
		var out []peResource
		f.walkResources(0, len(f.data)-2, 0, resourceLabel{}, resourceLabel{}, &out)
		if len(out) != 0 {
			t.Errorf("read %d things from past the end", len(out))
		}
		// And a thing said to sit past the end is not read either.
		out = nil
		f.addResourceLeaf(len(f.data)-2, resourceLabel{}, resourceLabel{}, resourceLabel{}, &out)
		if len(out) != 0 {
			t.Errorf("read %d things from past the end", len(out))
		}
	})
	t.Run("a name reaching past the end", func(t *testing.T) {
		b := newPE(true)
		b.withResources(0, 0, 0, []resourceLeaf{{kind: 3, id: 1, body: "x"}})
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		// An entry naming a place past the end has no name to read.
		f.data = copyOf(f.data)
		at := len(f.data) - 8
		f.data[at+3] = 0x80
		f.data[at], f.data[at+1], f.data[at+2] = 0xFF, 0xFF, 0x7F
		label, _ := f.resourceLabel(0, at)
		if label.isName && label.name != "" {
			t.Errorf("read a name from past the end: %q", label.name)
		}
	})
	t.Run("text with no end to it", func(t *testing.T) {
		f := &peFile{data: []byte{'a', 0, 'b', 0}}
		if _, ok := f.wideString(0); ok {
			t.Error("read text that never ends")
		}
		if _, ok := f.wideString(-1); ok {
			t.Error("read text from before the start")
		}
		if _, ok := f.wideString(99); ok {
			t.Error("read text from past the end")
		}
	})
}

// TestPEVersionInfoOfSomethingElse covers a version resource whose block does
// not open the way one should, which is left alone.
func TestPEVersionInfoOfSomethingElse(t *testing.T) {
	b := newPE(true)
	b.withResources(0, 0, 0, []resourceLeaf{
		{kind: 16, id: 1, body: "not a version block at all"},
	})
	wantNothing(t, scanPE(t, b.data, `pe.version_info["CompanyName"]`), "a name from nothing")
	wantNothing(t, scanPE(t, b.data, "pe.number_of_version_infos"), "how many names")
}

// TestPEPdbPathOfAnotherKindOfRecord covers a debugging record that is not the
// kind naming a path.
func TestPEPdbPathOfAnotherKindOfRecord(t *testing.T) {
	b := newPE(true)
	b.withDebugRecord(`C:\thing.pdb`)
	// The kind sits twelve bytes into the record, which begins the section.
	b.data[0x400+12] = 9
	wantNothing(t, scanPE(t, b.data, "pe.pdb_path"), "a path from another kind of record")
}

// The guards that keep a malformed PE file from being read past its end. A
// well-formed file never reaches them, so they are called directly.

// TestPEDelayedImports covers what a file borrows only when it first needs it,
// which is counted from a table of its own.
func TestPEDelayedImports(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	// The same table is named again as the one for what is borrowed late, so
	// both counts are taken from it.
	b.put32(b.directoriesAt()+directoryDelayed*dataDirectoryEntry, 0x1000)
	b.put32(b.directoriesAt()+directoryDelayed*dataDirectoryEntry+4, 40)

	wantInt(t, scanPE(t, b.data, "pe.number_of_delayed_imports"), 1, "how many libraries")
	wantInt(t, scanPE(t, b.data, "pe.number_of_delayed_imported_functions"), 1, "how many things")
}

// TestPEImportsFromNames covers libraries and things borrowed whose names
// cannot be used: not there, not readable, or not a name at all.
func TestPEImportsFromNames(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}

	t.Run("a library whose name is not one", func(t *testing.T) {
		spoilt := *f
		spoilt.data = copyOf(b.data)
		// The library's name sits just past the descriptors at the start of the
		// section; putting a control character in it makes it no name at all.
		spoilt.data[0x400+40] = 0x01
		if got := len(spoilt.readImports(directoryImports)); got != 0 {
			t.Errorf("read %d libraries whose name is not one", got)
		}
	})
	t.Run("a thing borrowed whose name is not one", func(t *testing.T) {
		spoilt := *f
		spoilt.data = copyOf(b.data)
		// The name of the thing borrowed follows the library's, after a hint.
		spoilt.data[0x400+40+13+2] = 0x01
		imports := spoilt.readImports(directoryImports)
		if len(imports) != 0 {
			t.Errorf("kept a library whose only thing borrowed has no name: %v", imports)
		}
	})
	t.Run("a list of things borrowed pointing nowhere", func(t *testing.T) {
		if got := f.readThunks(0x900000, "x.dll"); got != nil {
			t.Errorf("read %d things from nowhere", len(got))
		}
	})
	t.Run("a name pointing nowhere", func(t *testing.T) {
		if _, read := f.nameAt(0x900000); read {
			t.Error("read a name from nowhere")
		}
	})
	t.Run("a name with no end to it", func(t *testing.T) {
		short := &peFile{data: []byte("abc")}
		if _, read := short.nameAt(0); read {
			t.Error("read a name that never ends")
		}
	})
}

// TestPEImportsThroughTheSecondList covers a library whose first list of things
// borrowed is not there, so the second is used instead.
func TestPEImportsThroughTheSecondList(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	// The first list is named at the start of the descriptor and the second
	// sixteen bytes in; clearing the first makes the second the one used.
	binary.LittleEndian.PutUint32(b.data[0x400:], 0)
	wantInt(t, scanPE(t, b.data, "pe.number_of_imports"), 1, "how many libraries")
	if got := scanPE(t, b.data, "pe.import_details[0].functions[0].name"); got.s != "ExitProcess" {
		t.Errorf("the thing borrowed is %q, want %q", got.s, "ExitProcess")
	}
}

// TestPEOrdinalNamesOfTheOtherSocketLibrary covers the older name for the
// sockets library, which shares the newer one's table.
func TestPEOrdinalNamesOfTheOtherSocketLibrary(t *testing.T) {
	if got := ordinalName("wsock32.dll", 1); got != "accept" {
		t.Errorf("number 1 is %q, want %q", got, "accept")
	}
	if got := ordinalName("WSOCK32.DLL", 115); got != "WSAStartup" {
		t.Errorf("number 115 is %q, want %q", got, "WSAStartup")
	}
	// A number the table does not hold falls back to the number itself.
	if got := ordinalName("ws2_32.dll", 9999); got != "ord9999" {
		t.Errorf("a number not in the table is %q, want %q", got, "ord9999")
	}
}

// TestPEExportsThatCannotBeRead covers a table of things lent out that runs off
// the end of the file.
func TestPEExportsThatCannotBeRead(t *testing.T) {
	b := newPE(true)
	b.withExports("sample.dll", 1, []exportedFunction{{name: "Alpha", address: 0x2000}})
	f, ok := readPE(b.data)
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}

	t.Run("a table with no room for its own header", func(t *testing.T) {
		short := *f
		short.data = b.data[:0x400+8]
		if got := short.readExports(); got != nil {
			t.Errorf("read %d things from a table with no header", len(got))
		}
	})
	t.Run("addresses pointing nowhere", func(t *testing.T) {
		spoilt := *f
		spoilt.data = copyOf(b.data)
		// Where the addresses are is named twenty-eight bytes into the table.
		binary.LittleEndian.PutUint32(spoilt.data[0x400+28:], 0x900000)
		if got := spoilt.readExports(); got != nil {
			t.Errorf("read %d things whose addresses are nowhere", len(got))
		}
	})
	t.Run("a name list pointing nowhere", func(t *testing.T) {
		spoilt := *f
		spoilt.data = copyOf(b.data)
		binary.LittleEndian.PutUint32(spoilt.data[0x400+32:], 0x900000)
		got := spoilt.readExports()
		if len(got) != 1 || got[0].named {
			t.Errorf("read a name from nowhere: %+v", got)
		}
	})
	t.Run("more things than the file holds", func(t *testing.T) {
		spoilt := *f
		spoilt.data = copyOf(b.data)
		// The table says there are far more than the file has room for, so the
		// reading stops at what is there.
		binary.LittleEndian.PutUint32(spoilt.data[0x400+20:], 100000)
		if got := len(spoilt.readExports()); got == 0 || got >= 100000 {
			t.Errorf("read %d things, want as many as the file holds", got)
		}
	})
}

// TestPEExportOffsetIsReported covers a thing lent out whose address does come
// from somewhere in the file.
func TestPEExportOffsetIsReported(t *testing.T) {
	b := newPE(true)
	b.withExports("sample.dll", 1, []exportedFunction{{name: "Alpha", address: 0x1800}})
	got := scanPE(t, b.data, "pe.export_details[0].offset")
	if got.kind != valueInt {
		t.Errorf("where it sits came to %v, want a number", got.kind)
	}
}

// TestPEResourceLabelsThatCannotBeRead covers entries of the resource tree that
// name nothing readable.
func TestPEResourceLabelsThatCannotBeRead(t *testing.T) {
	f := &peFile{data: make([]byte, 8)}
	if _, ok := f.resourceLabel(0, 100); ok {
		t.Error("read an entry from past the end")
	}
	// An entry naming a place whose length runs past the end has no name.
	binary.LittleEndian.PutUint32(f.data, highBit|4)
	binary.LittleEndian.PutUint16(f.data[4:], 1000)
	if label, _ := f.resourceLabel(0, 0); label.isName {
		t.Errorf("read a name running past the end: %q", label.name)
	}

	// A level with nothing to read at all yields nothing.
	var out []peResource
	f.walkResources(0, 4, 0, resourceLabel{}, resourceLabel{}, &out)
	if len(out) != 0 {
		t.Errorf("read %d things from a level that is not there", len(out))
	}
	// And a level below the last is not gone into.
	f.walkResources(0, 0, resourceDepth, resourceLabel{}, resourceLabel{}, &out)
	if len(out) != 0 {
		t.Errorf("read %d things from below the last level", len(out))
	}
}

// TestPEResourceWithNothingAtALevel covers a thing carried that one level of
// the tree says nothing about, which is left out rather than being nought.
func TestPEResourceWithNothingAtALevel(t *testing.T) {
	r := peResource{rva: 1, length: 2}
	got := resourceValue(r)
	for _, name := range []string{"type", "id", "language", "type_string"} {
		if _, there := got.fields[name]; there {
			t.Errorf("%s was reported for a level that says nothing", name)
		}
	}
}

// TestPEVersionBlocksThatRunOut covers version blocks that stop part way, which
// are read as far as they go.
func TestPEVersionBlocksThatRunOut(t *testing.T) {
	f := &peFile{data: make([]byte, 4)}
	var keys, values []string
	f.readStringTables(0, 4, &keys, &values)
	f.readStrings(0, 4, &keys, &values)
	if len(keys) != 0 {
		t.Errorf("read %d names from nothing", len(keys))
	}

	// A block whose name cannot be read stops the walk.
	nameless := &peFile{data: append([]byte{8, 0, 0, 0, 0, 0}, 'a', 0)}
	keys, values = nil, nil
	nameless.readStringTables(0, 8, &keys, &values)
	nameless.readStrings(0, 8, &keys, &values)
	if len(keys) != 0 {
		t.Errorf("read %d names with no name to read", len(keys))
	}
}

// TestPEVersionInfoPastOtherBlocks covers a version resource whose blocks of
// names come after ones of another kind, which are stepped over.
func TestPEVersionInfoPastOtherBlocks(t *testing.T) {
	b := newPE(true)
	b.withResources(0, 0, 0, []resourceLeaf{
		{kind: 16, id: 1, body: versionBlockAfter("VarFileInfo",
			map[string]string{"CompanyName": "A Company"})},
	})
	if got := scanPE(t, b.data, `pe.version_info["CompanyName"]`); got.s != "A Company" {
		t.Errorf("the name past another block is %q, want %q", got.s, "A Company")
	}
}

// versionBlockAfter is a version block with one of another kind in front of the
// names, which is what a real file carries.
func versionBlockAfter(other string, entries map[string]string) string {
	first := versionHeader(0, 0, 1, other, 0)
	first = append(first, 0, 0, 0, 0)
	binary.LittleEndian.PutUint16(first, uint16(len(first)))

	block := []byte(versionBlock(entries))
	// The other block is put in front of the names, inside the outer one.
	out := copyOf(block[:versionInfoSkip])
	out = append(out, first...)
	out = append(out, block[versionInfoSkip:]...)
	binary.LittleEndian.PutUint16(out, uint16(len(out)))
	return string(out)
}

// TestPEPdbPathThroughThePlaceInTheFile covers a debugging record whose address
// leads nowhere, so the place in the file is used instead.
func TestPEPdbPathThroughThePlaceInTheFile(t *testing.T) {
	const path = `C:\other.pdb`
	b := newPE(true)
	b.withDebugRecord(path)
	// Clearing the address leaves only the place in the file to go on.
	binary.LittleEndian.PutUint32(b.data[0x400+20:], 0)
	if got := scanPE(t, b.data, "pe.pdb_path"); got.s != path {
		t.Errorf("the path is %q, want %q", got.s, path)
	}

	t.Run("neither one nor the other", func(t *testing.T) {
		none := copyOf(b.data)
		binary.LittleEndian.PutUint32(none[0x400+24:], 0)
		wantNothing(t, scanPE(t, none, "pe.pdb_path"), "a path named neither way")
	})
	t.Run("a record pointing nowhere", func(t *testing.T) {
		away := newPE(true)
		away.put32(away.opt+108, 16)
		away.put32(away.directoriesAt()+directoryDebug*dataDirectoryEntry, 0x900000)
		wantNothing(t, scanPE(t, away.data, "pe.pdb_path"), "a path from nowhere")
	})
	t.Run("a record with no room for itself", func(t *testing.T) {
		short := newPE(true)
		short.put32(short.opt+108, 16)
		short.put32(short.directoriesAt()+directoryDebug*dataDirectoryEntry, 0x10)
		wantNothing(t, scanPE(t, short.data, "pe.pdb_path"), "a path from a record with no room")
	})
}

// TestPEVersionNameTooLong covers text longer than libyara keeps, which is cut
// off rather than read to the end.
func TestPEVersionNameTooLong(t *testing.T) {
	long := make([]byte, 0, (maxVersionName+10)*2)
	for range maxVersionName + 10 {
		long = append(long, 'x', 0)
	}
	f := &peFile{data: append(long, 0, 0)}
	got, ok := f.wideString(0)
	if !ok {
		t.Fatal("text that is there was not read")
	}
	if len(got) > maxVersionName+1 {
		t.Errorf("read %d characters, want it cut off near %d", len(got), maxVersionName)
	}
}

// TestModuleCallWithABadPattern covers a pattern handed to a module that cannot
// be read, which stops the scan rather than being taken for no match.
func TestModuleCallWithABadPattern(t *testing.T) {
	e := &evaluator{
		buf: newBuffer([]byte("x")), vars: map[string]int64{}, matched: map[string]bool{},
		modules: map[string]modValue{
			"m": structOf(map[string]modValue{
				"f": funcOf(func(*evaluator, []value) (value, error) {
					return intValue(0), nil
				}),
			}),
		},
	}
	_, err := e.evalModuleRef(ModuleRef{Module: "m", Steps: []ModuleStep{{
		Name: "f", Call: true, Args: []Expr{RegexLit{Body: "([unclosed"}},
	}}})
	if err == nil {
		t.Error("a pattern that cannot be read was taken for an answer")
	}
}

// TestPEPiecesReadDirectly covers the last few guards, reached by calling the
// readers on files built to trip each one.
func TestPEPiecesReadDirectly(t *testing.T) {
	t.Run("a library with no list of things borrowed", func(t *testing.T) {
		b := newPE(true)
		b.withImports([]importedDLL{
			{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
		})
		// With neither list named, the library has nothing to offer.
		binary.LittleEndian.PutUint32(b.data[0x400:], 0)
		binary.LittleEndian.PutUint32(b.data[0x400+16:], 0x900000)
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		if got := len(f.readImports(directoryImports)); got != 0 {
			t.Errorf("kept %d libraries with nothing borrowed from them", got)
		}
	})
	t.Run("a name list whose entries cannot be read", func(t *testing.T) {
		b := newPE(true)
		b.withExports("s.dll", 1, []exportedFunction{{name: "Alpha", address: 0x1800}})
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		// Asking for a name past the end of the file has none to give.
		if _, named := f.exportName(0, len(f.data)-2, len(f.data)-2, 1); named {
			t.Error("read a name from past the end")
		}
	})
	t.Run("a thing carried that cannot be read", func(t *testing.T) {
		f := &peFile{data: make([]byte, 20)}
		var out []peResource
		f.addResourceLeaf(-1, resourceLabel{}, resourceLabel{}, resourceLabel{}, &out)
		if len(out) != 0 {
			t.Errorf("read %d things from before the start", len(out))
		}
	})
	t.Run("an entry of a level that cannot be read", func(t *testing.T) {
		// A level saying it has entries which sit past the end of the file.
		data := make([]byte, resourceDirSize)
		binary.LittleEndian.PutUint16(data[14:], 4)
		f := &peFile{data: data}
		var out []peResource
		f.walkResources(0, 0, 0, resourceLabel{}, resourceLabel{}, &out)
		if len(out) != 0 {
			t.Errorf("read %d things from entries that are not there", len(out))
		}
	})
	t.Run("a version block that never opens properly", func(t *testing.T) {
		f := &peFile{data: make([]byte, 4)}
		if keys, _ := f.readVersionInfo(0); keys != nil {
			t.Errorf("read %d names from a block with no opening", len(keys))
		}
	})
	t.Run("a table of names running past its block", func(t *testing.T) {
		// A table claiming to be longer than the block that holds it.
		data := make([]byte, 64)
		binary.LittleEndian.PutUint16(data, 200)
		data[6], data[8] = 'k', 0
		f := &peFile{data: data}
		var keys, values []string
		f.readStringTables(0, 40, &keys, &values)
		if len(keys) != 0 {
			t.Errorf("read %d names from a table that overruns its block", len(keys))
		}
	})
	t.Run("a path from a record naming an address that leads nowhere", func(t *testing.T) {
		b := newPE(true)
		b.withDebugRecord(`C:\x.pdb`)
		// An address past the end and no place in the file leaves nothing.
		binary.LittleEndian.PutUint32(b.data[0x400+20:], 0x900000)
		binary.LittleEndian.PutUint32(b.data[0x400+24:], 0)
		wantNothing(t, scanPE(t, b.data, "pe.pdb_path"), "a path from nowhere")
	})
}

// TestPEPiecesCutShort covers records and blocks the file stops in the middle
// of, which are given up on rather than read past the end.
func TestPEPiecesCutShort(t *testing.T) {
	t.Run("a descriptor with no list named", func(t *testing.T) {
		b := newPE(true)
		b.withImports([]importedDLL{
			{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
		})
		// The file stops between the library's name and where its list would be.
		f, ok := readPE(b.data[:0x400+16])
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		if got := len(f.readImports(directoryImports)); got != 0 {
			t.Errorf("read %d libraries from a descriptor cut short", got)
		}
	})
	t.Run("a version block whose name runs to the end", func(t *testing.T) {
		// A block claiming a length, whose name never ends.
		data := append([]byte{20, 0, 0, 0, 0, 0}, []byte("VS_VERSION_INFO")...)
		f := &peFile{data: data}
		if keys, _ := f.readVersionInfo(0); keys != nil {
			t.Errorf("read %d names from a block cut short", len(keys))
		}
	})
	t.Run("a block of names whose own name runs to the end", func(t *testing.T) {
		data := make([]byte, versionInfoSkip+8)
		copy(data[versionHeaderSize:], wideBytes("VS_VERSION_INFO"))
		binary.LittleEndian.PutUint16(data[versionInfoSkip:], 40)
		// The name after that opening never ends, so the walk gives up.
		for i := versionInfoSkip + versionHeaderSize; i < len(data); i++ {
			data[i] = 'x'
		}
		f := &peFile{data: data}
		if keys, _ := f.readVersionInfo(0); keys != nil {
			t.Errorf("read %d names from a block with no name", len(keys))
		}
	})
	t.Run("a table whose name runs to the end", func(t *testing.T) {
		data := make([]byte, 40)
		binary.LittleEndian.PutUint16(data, 20)
		for i := versionHeaderSize; i < len(data); i++ {
			data[i] = 'x'
		}
		f := &peFile{data: data}
		var keys, values []string
		f.readStringTables(-versionHeaderSize-stringFileInfoSkip, 40, &keys, &values)
		f.readStrings(0, 40, &keys, &values)
		if len(keys) != 0 {
			t.Errorf("read %d names with no name to read", len(keys))
		}
	})
	t.Run("a name whose text runs to the end", func(t *testing.T) {
		// One entry whose name reads but whose text never ends.
		entry := make([]byte, versionHeaderSize)
		binary.LittleEndian.PutUint16(entry[2:], 2)
		binary.LittleEndian.PutUint16(entry[4:], 1)
		entry = append(entry, wideBytes("Key")...)
		entry = append(entry, 0, 0)
		entry = append(entry, 'a', 0, 'b', 0)
		binary.LittleEndian.PutUint16(entry, uint16(len(entry)))
		f := &peFile{data: entry}
		var keys, values []string
		f.readStrings(0, len(entry), &keys, &values)
		if len(keys) != 1 || values[0] != "" {
			t.Errorf("read %q as %q, want one name set to nothing", keys, values)
		}
	})
	t.Run("a debugging record with no room for itself", func(t *testing.T) {
		b := newPE(true)
		b.withDebugRecord(`C:\x.pdb`)
		short := b.data[:0x400+8]
		wantNothing(t, scanPE(t, short, "pe.pdb_path"), "a path from a record cut short")
	})
	t.Run("a path starting past the end", func(t *testing.T) {
		b := newPE(true)
		b.withDebugRecord(`C:\x.pdb`)
		// The record names a place so near the end that the path cannot begin.
		binary.LittleEndian.PutUint32(b.data[0x400+20:], 0)
		binary.LittleEndian.PutUint32(b.data[0x400+24:], uint32(len(b.data)-2))
		wantNothing(t, scanPE(t, b.data, "pe.pdb_path"), "a path starting past the end")
	})
}

// copyOf is the file's bytes on their own, so that a test may spoil them
// without disturbing the file they came from.
func copyOf(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

// TestPERichSignatureGuards covers the run in the stub when it cannot be read:
// no room before the header proper, a key of nought, or a run too short to hold
// anything.
func TestPERichSignatureGuards(t *testing.T) {
	t.Run("no room before the header proper", func(t *testing.T) {
		f := &peFile{data: make([]byte, 8), nt: 2}
		if _, _, _, found := f.findRichSignature(); found {
			t.Error("found a run in a file with no room for one")
		}
	})
	t.Run("a header proper past the end", func(t *testing.T) {
		f := &peFile{data: make([]byte, 8), nt: 100}
		if _, _, _, found := f.findRichSignature(); found {
			t.Error("found a run past the end")
		}
	})
	t.Run("a key of nought", func(t *testing.T) {
		b := newPE(true)
		b.withRichSignature(1, []richTool{{id: 1, version: 2, times: 3}})
		// The key follows the closing marker; a key of nought is no key at all.
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		_, _, length, found := f.findRichSignature()
		if !found {
			t.Fatal("the run was not found to begin with")
		}
		binary.LittleEndian.PutUint32(b.data[dosHeaderSize+length+4:], 0)
		wantNothing(t, scanPE(t, b.data, "pe.rich_signature.length"), "a run with no key")
	})
	t.Run("a run with no room for its opening", func(t *testing.T) {
		if got := richTools(make([]byte, 4)); got != nil {
			t.Errorf("read %d tools from a run too short to hold any", len(got))
		}
	})
	t.Run("asked in ways that have no answer", func(t *testing.T) {
		b := newPE(true)
		b.withRichSignature(0x33333333, []richTool{{id: 1, version: 2, times: 3}})
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		fields := map[string]modValue{}
		f.addRichSignature(fields)
		ask := fields["rich_signature"].fields["version"]
		asked := [][]value{
			nil,
			{stringValue("x")},
			{intValue(1), intValue(2), intValue(3)},
		}
		for _, args := range asked {
			got, err := ask.call(nil, args)
			if err != nil {
				t.Fatalf("asking: %v", err)
			}
			if got.kind != valueUndefined {
				t.Errorf("%v came to %v, want nothing", args, got.kind)
			}
		}
	})
}

// TestPEVersionValueSaidToBeEmpty covers a name whose own length says it is set
// to nothing, so what happens to follow it is not read as its value.
func TestPEVersionValueSaidToBeEmpty(t *testing.T) {
	f := &peFile{data: []byte("unused")}
	if got := f.versionValue(0, "Key", 0); got != "" {
		t.Errorf("a name set to nothing read as %q", got)
	}
}

// TestPEImportsWithNoRoomForEitherList covers a descriptor the file stops in
// the middle of, before either list of things borrowed is named.
func TestPEImportsWithNoRoomForEitherList(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	// The first list is named at the start of the descriptor; clearing it sends
	// the reading to the second, which the file is cut short of.
	binary.LittleEndian.PutUint32(b.data[0x400:], 0)
	f, ok := readPE(b.data[:0x400+16])
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	if got := len(f.readImports(directoryImports)); got != 0 {
		t.Errorf("read %d libraries from a descriptor cut short", got)
	}
}

// TestPERichSignatureRunningOffTheStart covers a stub whose closing marker sits
// so near the front that nothing could open the run before it.
func TestPERichSignatureRunningOffTheStart(t *testing.T) {
	// A file with the closing marker and a key right at the start of the stub,
	// leaving nowhere for the opening to be.
	data := make([]byte, dosHeaderSize+16)
	binary.LittleEndian.PutUint32(data[dosHeaderSize:], richRich)
	binary.LittleEndian.PutUint32(data[dosHeaderSize+4:], 0x1234)
	f := &peFile{data: data, nt: dosHeaderSize + 8}
	if _, _, _, found := f.findRichSignature(); found {
		t.Error("found a run with nothing opening it")
	}
}

// TestPEImportsCountedPastWhatTheFileHolds covers a table of libraries claiming
// far more than the file has room for.
func TestPEImportsCountedPastWhatTheFileHolds(t *testing.T) {
	b := newPE(true)
	b.withImports([]importedDLL{
		{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
	})
	// Cutting the file just past the first descriptor leaves the closing blank
	// one unreadable, so the reading stops at what is there.
	f, ok := readPE(b.data[:0x400+importDescriptorSize+30])
	if !ok {
		t.Fatal("the file did not read as a PE file")
	}
	if got := len(f.readImports(directoryImports)); got != 0 {
		t.Errorf("read %d libraries past what the file holds", got)
	}
}

// TestPEPiecesCutShorterStill covers two more places a file can stop, which no
// well-formed one reaches.
func TestPEPiecesCutShorterStill(t *testing.T) {
	t.Run("a descriptor with room for neither list", func(t *testing.T) {
		b := newPE(true)
		b.withImports([]importedDLL{
			{name: "KERNEL32.dll", functions: []importedFunction{{name: "ExitProcess"}}},
		})
		f, ok := readPE(b.data)
		if !ok {
			t.Fatal("the file did not read as a PE file")
		}
		// Neither list is named, so the library has nothing to offer.
		spoilt := *f
		spoilt.data = copyOf(b.data)
		binary.LittleEndian.PutUint32(spoilt.data[0x400:], 0)
		binary.LittleEndian.PutUint32(spoilt.data[0x400+16:], 0)
		if got := len(spoilt.readImports(directoryImports)); got != 0 {
			t.Errorf("read %d libraries with no room for their lists", got)
		}
	})
	t.Run("a stub the file stops inside", func(t *testing.T) {
		// The closing marker is found, but the file stops before anything that
		// could have opened the run.
		data := make([]byte, dosHeaderSize+10)
		binary.LittleEndian.PutUint32(data[dosHeaderSize+2:], richRich)
		binary.LittleEndian.PutUint32(data[dosHeaderSize+6:], 0x1234)
		f := &peFile{data: data[:dosHeaderSize+9], nt: dosHeaderSize + 6}
		if _, _, _, found := f.findRichSignature(); found {
			t.Error("found a run in a file that stops inside its stub")
		}
	})
}
