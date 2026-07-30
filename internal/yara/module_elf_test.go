package yara

import (
	"encoding/binary"
	"testing"
)

// The elf module is tested against files built here rather than against real
// binaries, so that each shape the format takes — both widths, both ends first,
// and the ways a file can be malformed — can be tried deliberately. The values
// are what CyberChef gives for the same bytes.

// elfBuilder lays out an ELF file field by field.
type elfBuilder struct {
	wide  bool
	big   bool
	data  []byte
	order binary.ByteOrder
}

func newELF(wide, big bool) *elfBuilder {
	b := &elfBuilder{wide: wide, big: big, order: binary.LittleEndian}
	if big {
		b.order = binary.BigEndian
	}
	size := 52
	if wide {
		size = 64
	}
	b.data = make([]byte, size)
	copy(b.data, elfMagic)
	b.data[elfClassOffset] = elfClass32
	if wide {
		b.data[elfClassOffset] = elfClass64
	}
	b.data[elfDataOffset] = elfDataLittle
	if big {
		b.data[elfDataOffset] = elfDataBig
	}
	return b
}

// put16 and putWord write a field of the header, the second as wide as the
// file's own addresses.
func (b *elfBuilder) put16(at int, n uint16) {
	b.order.PutUint16(b.data[at:], n)
}

func (b *elfBuilder) putWord(at int, n uint64) {
	if b.wide {
		b.order.PutUint64(b.data[at:], n)
		return
	}
	b.order.PutUint32(b.data[at:], uint32(n))
}

// counts is where the run of sizes and counts begins, which differs between
// the two widths.
func (b *elfBuilder) counts() int {
	if b.wide {
		return 52
	}
	return 40
}

// append adds bytes to the end of the file and says where they went.
func (b *elfBuilder) append(chunk []byte) int {
	at := len(b.data)
	b.data = append(b.data, chunk...)
	return at
}

// word writes a number as wide as the file's own addresses.
func (b *elfBuilder) word(n uint64) []byte {
	if b.wide {
		out := make([]byte, 8)
		b.order.PutUint64(out, n)
		return out
	}
	out := make([]byte, 4)
	b.order.PutUint32(out, uint32(n))
	return out
}

func (b *elfBuilder) u16(n uint16) []byte {
	out := make([]byte, 2)
	b.order.PutUint16(out, n)
	return out
}

func (b *elfBuilder) u32(n uint32) []byte {
	out := make([]byte, 4)
	b.order.PutUint32(out, n)
	return out
}

// elfSectionEntry is what one entry of the section table holds while a test
// file is being laid out.
type elfSectionEntry struct {
	name               uint32
	kind, flags, addr  uint64
	offset, size, link uint64
}

// sectionTable writes a section table and points the header at it.
func (b *elfBuilder) sectionTable(entries []elfSectionEntry, namesIndex uint16) {
	var table []byte
	for _, e := range entries {
		row := append([]byte{}, b.u32(e.name)...)
		row = append(row, b.u32(uint32(e.kind))...)
		row = append(row, b.word(e.flags)...)
		row = append(row, b.word(e.addr)...)
		row = append(row, b.word(e.offset)...)
		row = append(row, b.word(e.size)...)
		row = append(row, b.u32(uint32(e.link))...)
		// The rest of the entry — information, alignment and entry size — is
		// not read, but the entry has to be its full length.
		size := 40
		if b.wide {
			size = 64
		}
		row = append(row, make([]byte, size-len(row))...)
		table = append(table, row...)
	}
	at := b.append(table)
	shOffsetAt := 32
	if b.wide {
		shOffsetAt = 40
	}
	b.putWord(shOffsetAt, uint64(at))
	b.put16(b.counts()+6, uint16(40+24*boolToInt(b.wide)))
	b.put16(b.counts()+8, uint16(len(entries)))
	b.put16(b.counts()+10, namesIndex)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// scanELF builds the module over some bytes and reads one field out of it.
func scanELF(t *testing.T, data []byte, path string) value {
	t.Helper()
	e := &evaluator{buf: newBuffer(data), vars: map[string]int64{}, matched: map[string]bool{}}
	set, err := Parse(`import "elf" rule R { condition: ` + path + ` == 0 }`)
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

// wantInt checks a field came to a given number.
func wantInt(t *testing.T, got value, want int64, what string) {
	t.Helper()
	if got.kind != valueInt || got.i != want {
		t.Errorf("%s came to %v %d, want the number %d", what, got.kind, got.i, want)
	}
}

// wantNothing checks a field has no answer.
func wantNothing(t *testing.T, got value, what string) {
	t.Helper()
	if got.kind != valueUndefined {
		t.Errorf("%s came to %v, want nothing at all", what, got.kind)
	}
}

// TestELFNotAnELF covers data that is not an ELF file, which leaves every field
// but the constants without an answer.
func TestELFNotAnELF(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"nothing at all", nil},
		{"too short to say", []byte("\x7fELF")},
		{"the wrong opening bytes", append([]byte("NOPE"), make([]byte, 60)...)},
		{"a width it does not have", func() []byte {
			d := make([]byte, 64)
			copy(d, elfMagic)
			d[elfClassOffset] = 9
			d[elfDataOffset] = elfDataLittle
			return d
		}()},
		{"an end it does not have", func() []byte {
			d := make([]byte, 64)
			copy(d, elfMagic)
			d[elfClassOffset] = elfClass64
			d[elfDataOffset] = 9
			return d
		}()},
		{"a header cut short", func() []byte {
			d := make([]byte, 40)
			copy(d, elfMagic)
			d[elfClassOffset] = elfClass64
			d[elfDataOffset] = elfDataLittle
			return d
		}()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantNothing(t, scanELF(t, c.data, "elf.type"), "the kind of file")
			wantNothing(t, scanELF(t, c.data, "elf.machine"), "the machine")
			// The constants are there whatever the data is.
			wantInt(t, scanELF(t, c.data, "elf.ET_EXEC"), 2, "the constant for an executable")
		})
	}
}

// TestELFHeader covers what the opening header says, in both widths and with
// either end of a number first.
func TestELFHeader(t *testing.T) {
	for _, shape := range elfShapes() {
		t.Run(shape.name, func(t *testing.T) {
			b := newELF(shape.wide, shape.big)
			b.put16(16, 2)    // an executable
			b.put16(18, 0x3E) // built for x86-64
			b.put16(b.counts()+2, 56)
			b.put16(b.counts()+4, 3)
			b.put16(b.counts()+6, 64)
			b.put16(b.counts()+8, 5)

			wantInt(t, scanELF(t, b.data, "elf.type"), 2, "the kind of file")
			wantInt(t, scanELF(t, b.data, "elf.machine"), 0x3E, "the machine")
			wantInt(t, scanELF(t, b.data, "elf.ph_entry_size"), 56, "a segment's size")
			wantInt(t, scanELF(t, b.data, "elf.number_of_segments"), 3, "how many segments")
			wantInt(t, scanELF(t, b.data, "elf.sh_entry_size"), 64, "a section's size")
			wantInt(t, scanELF(t, b.data, "elf.number_of_sections"), 5, "how many sections")
		})
	}
}

// elfShapes is every combination of width and byte order a file may take.
func elfShapes() []struct {
	name      string
	wide, big bool
} {
	return []struct {
		name      string
		wide, big bool
	}{
		{"32-bit, least significant first", false, false},
		{"64-bit, least significant first", true, false},
		{"32-bit, most significant first", false, true},
		{"64-bit, most significant first", true, true},
	}
}

// TestELFSections covers the section table and the names read out of it.
func TestELFSections(t *testing.T) {
	for _, shape := range elfShapes() {
		t.Run(shape.name, func(t *testing.T) {
			b := newELF(shape.wide, shape.big)
			names := b.append([]byte("\x00.text\x00.data\x00"))
			b.sectionTable([]elfSectionEntry{
				{name: 0, kind: 0},
				{name: 1, kind: 1, flags: 6, addr: 0x1000, offset: 0x200, size: 0x40},
				{name: 7, kind: 1, flags: 3, addr: 0x2000, offset: 0x240, size: 0x10},
				{name: 0, kind: 3, offset: uint64(names), size: 13},
			}, 3)

			if got := scanELF(t, b.data, "elf.sections[1].name"); got.s != ".text" {
				t.Errorf("the second section is named %q, want %q", got.s, ".text")
			}
			if got := scanELF(t, b.data, "elf.sections[2].name"); got.s != ".data" {
				t.Errorf("the third section is named %q, want %q", got.s, ".data")
			}
			wantInt(t, scanELF(t, b.data, "elf.sections[1].address"), 0x1000, "where it loads")
			wantInt(t, scanELF(t, b.data, "elf.sections[1].size"), 0x40, "how big it is")
			wantInt(t, scanELF(t, b.data, "elf.sections[1].offset"), 0x200, "where it sits")
			wantInt(t, scanELF(t, b.data, "elf.sections[2].flags"), 3, "what it may be used for")
			wantNothing(t, scanELF(t, b.data, "elf.sections[9].name"), "a section past the end")
			wantNothing(t, scanELF(t, b.data, "elf.sections[-1].name"), "a section before the start")
		})
	}
}

// TestELFSectionNamesFromNowhere covers the cases where a name cannot be read:
// a table of names at the very start of the file, which libyara takes to be no
// table at all, and one the header points past the end of.
func TestELFSectionNamesFromNowhere(t *testing.T) {
	t.Run("a table of names at the start of the file", func(t *testing.T) {
		b := newELF(true, false)
		b.sectionTable([]elfSectionEntry{
			{name: 1, kind: 1},
			{name: 0, kind: 3, offset: 0, size: 8},
		}, 1)
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].name"), "a name from offset nought")
	})
	t.Run("a table of names past the end", func(t *testing.T) {
		b := newELF(true, false)
		b.sectionTable([]elfSectionEntry{
			{name: 1, kind: 1},
			{name: 0, kind: 3, offset: 1 << 20, size: 8},
		}, 1)
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].name"), "a name from past the end")
	})
	t.Run("the names section named by a number there is no section for", func(t *testing.T) {
		b := newELF(true, false)
		b.sectionTable([]elfSectionEntry{{name: 1, kind: 1}}, 9)
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].name"), "a name with no table")
	})
}

// TestELFSegments covers the table saying how the file is laid out once loaded,
// whose fields the two widths keep in a different order.
func TestELFSegments(t *testing.T) {
	for _, shape := range elfShapes() {
		t.Run(shape.name, func(t *testing.T) {
			b := newELF(shape.wide, shape.big)
			b.segmentTable([]elfSegmentEntry{
				{
					kind: 1, flags: 5, offset: 0, virtual: 0x400000, physical: 0x400000,
					fileSize: 0x1000, memorySize: 0x1000, alignment: 0x1000,
				},
				{
					kind: 2, flags: 6, offset: 0x2000, virtual: 0x600000, physical: 0x600000,
					fileSize: 0x100, memorySize: 0x200, alignment: 8,
				},
			})

			wantInt(t, scanELF(t, b.data, "elf.segments[0].type"), 1, "what the segment is for")
			wantInt(t, scanELF(t, b.data, "elf.segments[0].flags"), 5, "what may be done with it")
			wantInt(t, scanELF(t, b.data, "elf.segments[0].virtual_address"), 0x400000,
				"where it loads")
			wantInt(t, scanELF(t, b.data, "elf.segments[1].file_size"), 0x100, "its room on disk")
			wantInt(t, scanELF(t, b.data, "elf.segments[1].memory_size"), 0x200, "its room in memory")
			wantInt(t, scanELF(t, b.data, "elf.segments[1].alignment"), 8, "what it lines up to")
			wantNothing(t, scanELF(t, b.data, "elf.segments[5].type"), "a segment past the end")
		})
	}
}

// elfSegmentEntry is what one entry of the segment table holds while a test
// file is being laid out.
type elfSegmentEntry struct {
	kind, flags                     uint32
	offset, virtual, physical       uint64
	fileSize, memorySize, alignment uint64
}

// segmentTable writes a segment table and points the header at it.
func (b *elfBuilder) segmentTable(entries []elfSegmentEntry) {
	var table []byte
	for _, e := range entries {
		var row []byte
		if b.wide {
			row = append(row, b.u32(e.kind)...)
			row = append(row, b.u32(e.flags)...)
			row = append(row, b.word(e.offset)...)
			row = append(row, b.word(e.virtual)...)
			row = append(row, b.word(e.physical)...)
			row = append(row, b.word(e.fileSize)...)
			row = append(row, b.word(e.memorySize)...)
			row = append(row, b.word(e.alignment)...)
		} else {
			row = append(row, b.u32(e.kind)...)
			row = append(row, b.word(e.offset)...)
			row = append(row, b.word(e.virtual)...)
			row = append(row, b.word(e.physical)...)
			row = append(row, b.word(e.fileSize)...)
			row = append(row, b.word(e.memorySize)...)
			row = append(row, b.u32(e.flags)...)
			row = append(row, b.word(e.alignment)...)
		}
		table = append(table, row...)
	}
	at := b.append(table)
	phOffsetAt := 28
	if b.wide {
		phOffsetAt = 32
	}
	b.putWord(phOffsetAt, uint64(at))
	size := 32
	if b.wide {
		size = 56
	}
	b.put16(b.counts()+2, uint16(size))
	b.put16(b.counts()+4, uint16(len(entries)))
}

// TestELFDynamic covers the list of what a file needs at run time, which is
// read out of the segment that holds it and stops at the entry saying so.
func TestELFDynamic(t *testing.T) {
	for _, shape := range elfShapes() {
		t.Run(shape.name, func(t *testing.T) {
			b := newELF(shape.wide, shape.big)
			var entries []byte
			for _, pair := range [][2]uint64{{1, 0x10}, {5, 0x20}, {0, 0}} {
				entries = append(entries, b.word(pair[0])...)
				entries = append(entries, b.word(pair[1])...)
			}
			at := b.append(entries)
			b.segmentTable([]elfSegmentEntry{{kind: 2, offset: uint64(at)}})

			wantInt(t, scanELF(t, b.data, "elf.dynamic_section_entries"), 3, "how many entries")
			wantInt(t, scanELF(t, b.data, "elf.dynamic[0].type"), 1, "the first entry's kind")
			wantInt(t, scanELF(t, b.data, "elf.dynamic[0].val"), 0x10, "the first entry's value")
			wantInt(t, scanELF(t, b.data, "elf.dynamic[1].type"), 5, "the second entry's kind")
			// The list stops at the entry saying it has ended, so nothing after
			// it is read even if the file goes on.
			wantNothing(t, scanELF(t, b.data, "elf.dynamic[3].type"), "an entry past the end")
		})
	}
}

// TestELFSymbols covers the two tables of names a file defines, which share a
// shape and take their names from a table of their own.
func TestELFSymbols(t *testing.T) {
	for _, shape := range elfShapes() {
		t.Run(shape.name, func(t *testing.T) {
			for _, table := range []struct {
				kind uint64
				into string
			}{
				{2, "symtab"}, {11, "dynsym"},
			} {
				b := newELF(shape.wide, shape.big)
				names := b.append([]byte("\x00main\x00puts\x00"))
				var symbols []byte
				for _, sym := range []struct {
					name        uint32
					info        byte
					shndx       uint16
					value, size uint64
				}{
					{1, 0x12, 1, 0x1000, 0x20},
					{6, 0x22, 0, 0x2000, 0x10},
				} {
					var row []byte
					if b.wide {
						row = append(row, b.u32(sym.name)...)
						row = append(row, sym.info, 0)
						row = append(row, b.u16(sym.shndx)...)
						row = append(row, b.word(sym.value)...)
						row = append(row, b.word(sym.size)...)
					} else {
						row = append(row, b.u32(sym.name)...)
						row = append(row, b.word(sym.value)...)
						row = append(row, b.word(sym.size)...)
						row = append(row, sym.info, 0)
						row = append(row, b.u16(sym.shndx)...)
					}
					symbols = append(symbols, row...)
				}
				entry := 16
				if b.wide {
					entry = 24
				}
				at := b.append(symbols)
				b.sectionTable([]elfSectionEntry{
					{kind: table.kind, offset: uint64(at), size: uint64(2 * entry), link: 1},
					{kind: 3, offset: uint64(names), size: 11},
				}, 1)

				t.Run(table.into, func(t *testing.T) {
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"_entries"), 2, "how many")
					if got := scanELF(t, b.data, "elf."+table.into+"[0].name"); got.s != "main" {
						t.Errorf("the first is named %q, want %q", got.s, "main")
					}
					if got := scanELF(t, b.data, "elf."+table.into+"[1].name"); got.s != "puts" {
						t.Errorf("the second is named %q, want %q", got.s, "puts")
					}
					// The information byte holds two things: how widely the name
					// is visible in its upper half and what it names in its lower.
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[0].bind"), 1, "how visible")
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[0].type"), 2, "what it names")
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[1].bind"), 2, "how visible")
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[0].value"), 0x1000, "its address")
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[0].size"), 0x20, "its size")
					wantInt(t, scanELF(t, b.data, "elf."+table.into+"[0].shndx"), 1, "its section")
				})
			}
		})
	}
}

// TestELFSymbolsWithoutATable covers a symbol table pointing at something that
// is not a table of names, and one reaching past the end of the file, neither
// of which is read.
func TestELFSymbolsWithoutATable(t *testing.T) {
	t.Run("names that are not a table of names", func(t *testing.T) {
		b := newELF(true, false)
		at := b.append(make([]byte, 48))
		b.sectionTable([]elfSectionEntry{
			{kind: 2, offset: uint64(at), size: 48, link: 1},
			{kind: 1, offset: uint64(at), size: 8},
		}, 1)
		wantNothing(t, scanELF(t, b.data, "elf.symtab_entries"), "symbols with no names")
	})
	t.Run("a table naming a section there is none of", func(t *testing.T) {
		b := newELF(true, false)
		at := b.append(make([]byte, 48))
		b.sectionTable([]elfSectionEntry{{kind: 2, offset: uint64(at), size: 48, link: 9}}, 0)
		wantNothing(t, scanELF(t, b.data, "elf.symtab_entries"), "symbols with no names")
	})
	t.Run("a table reaching past the end", func(t *testing.T) {
		b := newELF(true, false)
		b.sectionTable([]elfSectionEntry{
			{kind: 2, offset: 1 << 20, size: 48, link: 1},
			{kind: 3, offset: 0, size: 8},
		}, 1)
		wantNothing(t, scanELF(t, b.data, "elf.symtab_entries"), "symbols from past the end")
	})
}

// TestELFEntryPoint covers where a program starts, which is given as an address
// in memory and reported as the place in the file it came from. An executable
// is looked up in the segment table and anything else in the section table.
func TestELFEntryPoint(t *testing.T) {
	t.Run("an executable, through its segments", func(t *testing.T) {
		b := newELF(true, false)
		b.put16(16, 2) // an executable
		b.putWord(24, 0x400100)
		b.segmentTable([]elfSegmentEntry{
			{kind: 1, offset: 0x40, virtual: 0x400000, fileSize: 0x1000, memorySize: 0x1000},
		})
		wantInt(t, scanELF(t, b.data, "elf.entry_point"), 0x140, "where the program starts")
	})
	t.Run("a shared object, through its sections", func(t *testing.T) {
		b := newELF(true, false)
		b.put16(16, 3) // a shared object
		b.putWord(24, 0x1010)
		b.sectionTable([]elfSectionEntry{
			{kind: 1, addr: 0x1000, offset: 0x200, size: 0x100},
		}, 0)
		wantInt(t, scanELF(t, b.data, "elf.entry_point"), 0x210, "where the program starts")
	})
	t.Run("an address in no segment at all", func(t *testing.T) {
		b := newELF(true, false)
		b.put16(16, 2)
		b.putWord(24, 0x900000)
		b.segmentTable([]elfSegmentEntry{
			{kind: 1, offset: 0x40, virtual: 0x400000, fileSize: 0x10, memorySize: 0x10},
		})
		wantNothing(t, scanELF(t, b.data, "elf.entry_point"), "a start from nowhere")
	})
	t.Run("an address in a section holding no bytes", func(t *testing.T) {
		b := newELF(true, false)
		b.put16(16, 3)
		b.putWord(24, 0x1010)
		// A section of this kind takes room in memory but none in the file, so
		// nothing in it came from anywhere.
		b.sectionTable([]elfSectionEntry{
			{kind: 8, addr: 0x1000, offset: 0x200, size: 0x100},
		}, 0)
		wantNothing(t, scanELF(t, b.data, "elf.entry_point"), "a start from a section with no bytes")
	})
	t.Run("no starting address at all", func(t *testing.T) {
		b := newELF(true, false)
		b.put16(16, 2)
		wantNothing(t, scanELF(t, b.data, "elf.entry_point"), "a start that was never given")
	})
}

// TestELFTablesThatDoNotFit covers tables the header claims are longer than the
// file, which are left out rather than read past the end.
func TestELFTablesThatDoNotFit(t *testing.T) {
	t.Run("more sections than there is room for", func(t *testing.T) {
		b := newELF(true, false)
		b.putWord(40, 64)
		b.put16(b.counts()+8, 1000)
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].type"), "a section past the end")
	})
	t.Run("a section table pointing past the end", func(t *testing.T) {
		b := newELF(true, false)
		b.putWord(40, 1<<20)
		b.put16(b.counts()+8, 2)
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].type"), "a section past the end")
	})
	t.Run("more sections than can be counted", func(t *testing.T) {
		b := newELF(true, false)
		b.putWord(40, 64)
		// A count this high stops counting sections and starts standing for
		// something else.
		b.put16(b.counts()+8, uint16(shnLoReserve))
		wantNothing(t, scanELF(t, b.data, "elf.sections[0].type"), "a section that is not one")
	})
	t.Run("more segments than there is room for", func(t *testing.T) {
		b := newELF(true, false)
		b.putWord(32, 64)
		b.put16(b.counts()+2, 56)
		b.put16(b.counts()+4, 1000)
		wantNothing(t, scanELF(t, b.data, "elf.segments[0].type"), "a segment past the end")
	})
	t.Run("no segments at all", func(t *testing.T) {
		b := newELF(true, false)
		b.putWord(32, 64)
		b.put16(b.counts()+4, 0)
		wantNothing(t, scanELF(t, b.data, "elf.segments[0].type"), "a segment there is none of")
	})
}

// TestELFNameRunningToTheEnd covers a name with no nought after it, which is
// not a name at all rather than running to the end of what it was given.
func TestELFNameRunningToTheEnd(t *testing.T) {
	if _, ok := stringAt([]byte("abc"), 0, 0, 3); ok {
		t.Error("read a name that never ends")
	}
	if got, ok := stringAt([]byte("ab\x00cd"), 0, 0, 5); !ok || got != "ab" {
		t.Errorf("read %q %v, want \"ab\"", got, ok)
	}
	// A name is bounded by its own table even when the file goes on further.
	if _, ok := stringAt([]byte("abcdef"), 0, 4, 2); ok {
		t.Error("read a name from beyond its table")
	}
	if _, ok := stringAt([]byte("abc"), -1, 0, 3); ok {
		t.Error("read a name from a table that is not there")
	}
	if _, ok := stringAt([]byte("abc"), 0, 99, 3); ok {
		t.Error("read a name from past the end")
	}
}

// TestELFReadingPastTheEnd covers the readers themselves asked for bytes the
// file does not have, which come to nothing rather than reading past the end.
func TestELFReadingPastTheEnd(t *testing.T) {
	f := elfFile{data: []byte{1, 2, 3}, order: binary.LittleEndian}
	for _, c := range []struct {
		name string
		read func(int) (uint64, bool)
		at   int
	}{
		{"two bytes past the end", f.u16, 2},
		{"two bytes before the start", f.u16, -1},
		{"four bytes past the end", f.u32, 0},
		{"four bytes before the start", f.u32, -1},
		{"eight bytes past the end", f.u64, 0},
		{"eight bytes before the start", f.u64, -1},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := c.read(c.at); ok {
				t.Error("read bytes that are not there")
			}
		})
	}

	if got := f.byteAt(9); got != 0 {
		t.Errorf("a byte past the end read as %d, want 0", got)
	}
	if got := f.byteAt(-1); got != 0 {
		t.Errorf("a byte before the start read as %d, want 0", got)
	}
	if got := f.byteAt(1); got != 2 {
		t.Errorf("a byte in the file read as %d, want 2", got)
	}
}

// TestELFNameBoundedByTheFile covers a table claiming to reach past the end,
// which is cut back to what is there.
func TestELFNameBoundedByTheFile(t *testing.T) {
	if got, ok := stringAt([]byte("ab\x00"), 0, 0, 1<<20); !ok || got != "ab" {
		t.Errorf("read %q %v, want \"ab\"", got, ok)
	}
}

// TestELFEntryPointOfATruncatedHeader covers a file whose header stops before
// it says what kind of file it is, which cannot be looked up either way.
func TestELFEntryPointOfATruncatedHeader(t *testing.T) {
	f := elfFile{data: make([]byte, 8), wide: true, order: binary.LittleEndian}
	if _, ok := f.addressToOffset(0x1000); ok {
		t.Error("turned an address into a place in a file that says nothing")
	}
}

// TestELFEntryPointWithNoSegmentTable covers an executable whose segment table
// is not there, so an address in memory came from nowhere in particular.
func TestELFEntryPointWithNoSegmentTable(t *testing.T) {
	b := newELF(true, false)
	b.put16(16, 2)
	if _, ok := b.elf().addressInSegments(0x400000); ok {
		t.Error("found a place in a file with no segments")
	}
}

// elf reads what has been built so far as an ELF file.
func (b *elfBuilder) elf() elfFile {
	f, ok := readELF(b.data)
	if !ok {
		panic("the file built for this test is not an ELF file")
	}
	return f
}

// TestELFEntryPointOfA32BitExecutable covers looking an address up in the
// segment table of a narrower file, whose fields sit in a different order.
func TestELFEntryPointOfA32BitExecutable(t *testing.T) {
	b := newELF(false, false)
	b.put16(16, 2) // an executable
	b.putWord(24, 0x8048100)
	b.segmentTable([]elfSegmentEntry{
		{kind: 1, offset: 0x40, virtual: 0x8048000, fileSize: 0x1000, memorySize: 0x1000},
	})
	wantInt(t, scanELF(t, b.data, "elf.entry_point"), 0x140, "where the program starts")
}
