package yara

import (
	"encoding/binary"
	"strconv"
	"testing"
)

// Files built for the runtime. Beyond being a PE file, one carries a header
// naming its metadata, and that metadata holds streams: the names, the numbers
// naming the build, the text the code quotes, and the tables the runtime works
// from.

// dotnetStreamEntry is one stream a test file carries.
type dotnetStreamEntry struct {
	name string
	body []byte
}

// dotnetTableEntry is one table of the metadata: which table it is, and a row
// of bytes for each entry in it.
type dotnetTableEntry struct {
	bit  int
	rows [][]byte
}

// tildeStream lays out the stream of tables: its header, how many rows each
// table has, then the tables themselves in order.
func tildeStream(heapSizes byte, tables []dotnetTableEntry) []byte {
	var valid uint64
	for _, t := range tables {
		valid |= 1 << t.bit
	}
	head := make([]byte, tildeHeaderSize)
	head[6] = heapSizes
	binary.LittleEndian.PutUint64(head[8:], valid)

	// The tables are written lowest first, whatever order they were given in.
	var counts, bodies []byte
	for bit := range 64 {
		for _, t := range tables {
			if t.bit != bit {
				continue
			}
			counts = binary.LittleEndian.AppendUint32(counts, uint32(len(t.rows)))
			for _, row := range t.rows {
				bodies = append(bodies, row...)
			}
		}
	}
	return append(append(head, counts...), bodies...)
}

// u16 and u32 write an index the width the tables use.
func u16(n int) []byte { return binary.LittleEndian.AppendUint16(nil, uint16(n)) }
func u32(n int) []byte { return binary.LittleEndian.AppendUint32(nil, uint32(n)) }

// joinBytes runs several stretches together into one row.
func joinBytes(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// stringsStream lays out the stream of names and says where each one begins.
func stringsStream(names ...string) ([]byte, map[string]int) {
	body := []byte{0}
	at := map[string]int{"": 0}
	for _, name := range names {
		at[name] = len(body)
		body = append(body, name...)
		body = append(body, 0)
	}
	return body, at
}

// blobStream lays out the stream of runs of bytes and says where each begins.
// Each run states its own length, which counts a closing byte beyond it.
func blobStream(runs ...[]byte) ([]byte, []int) {
	body := []byte{0}
	var at []int
	for _, run := range runs {
		at = append(at, len(body))
		// A length under 128 takes one byte; anything longer takes two, the
		// first of them marked so it is read that way.
		if length := len(run) + 1; length < 0x80 {
			body = append(body, byte(length))
		} else {
			body = append(body, byte(0x80|length>>8), byte(length))
		}
		body = append(body, run...)
		body = append(body, 0)
	}
	return body, at
}

// withDotnet lays the runtime's header and its metadata out in one section and
// points the table of tables at that header.
func (b *peBuilder) withDotnet(version string, streams []dotnetStreamEntry) {
	const sectionRVA, sectionOffset, sectionSize = 0x1000, 0x400, 0x4000
	b.put32(b.directoriesAt()-4, maxDataDirectories)
	b.put32(b.directoriesAt()+directoryCLI*dataDirectoryEntry, sectionRVA)
	b.put32(b.directoriesAt()+directoryCLI*dataDirectoryEntry+4, cliHeaderSize)
	b.sectionTable([]peSectionEntry{{
		name: ".text", virtualAddress: sectionRVA, virtualSize: sectionSize,
		rawOffset: sectionOffset, rawSize: sectionSize,
	}})
	b.pad(sectionOffset)

	body := make([]byte, cliHeaderSize)
	binary.LittleEndian.PutUint32(body, cliHeaderSize)
	binary.LittleEndian.PutUint32(body[cliMetadataAt:], sectionRVA+uint32(len(body)))
	// What the file keeps for itself sits right after the metadata, which is
	// where the resources a rule can ask about are found.
	body = append(body, dotnetMetadata(version, streams)...)
	binary.LittleEndian.PutUint32(body[cliResourcesAt:], sectionRVA+uint32(len(body)))

	b.data = append(b.data, body...)
	b.pad(sectionOffset + sectionSize)
}

// dotnetMetadata lays the metadata out: its own header, the version, then a
// header for each stream and the streams themselves.
func dotnetMetadata(version string, streams []dotnetStreamEntry) []byte {
	text := append([]byte(version), 0)
	for len(text)%versionWord != 0 {
		text = append(text, 0)
	}
	head := make([]byte, metadataHeaderSize)
	binary.LittleEndian.PutUint32(head, metadataMagic)
	binary.LittleEndian.PutUint32(head[metadataLengthAt:], uint32(len(text)))
	head = append(head, text...)
	head = append(head, 0, 0, byte(len(streams)), 0)

	// Each header is as long as its padded name, so where the streams
	// themselves begin is known before any is written.
	headers := 0
	for _, s := range streams {
		headers += streamHeaderSize + len(s.name) + versionWord - len(s.name)%versionWord
	}
	at := len(head) + headers
	var bodies []byte
	for _, s := range streams {
		header := make([]byte, streamHeaderSize)
		binary.LittleEndian.PutUint32(header, uint32(at+len(bodies)))
		binary.LittleEndian.PutUint32(header[4:], uint32(len(s.body)))
		name := append([]byte(s.name), 0)
		for len(name)%versionWord != 0 {
			name = append(name, 0)
		}
		head = append(head, append(header, name...)...)
		bodies = append(bodies, s.body...)
	}
	return append(head, bodies...)
}

// aDotnetFile lays out a file for the runtime carrying the given streams.
func aDotnetFile(streams ...dotnetStreamEntry) []byte {
	b := newPE(true)
	b.withDotnet("v4.0.30319", streams)
	return b.data
}

// scanDotnet reads one field of the dotnet module out over some bytes.
func scanDotnet(t *testing.T, data []byte, path string) value {
	t.Helper()
	e := &evaluator{buf: newBuffer(data), vars: map[string]int64{}, matched: map[string]bool{}}
	set, err := Parse(`import "dotnet" rule R { condition: ` + path + ` == 0 }`)
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

// at writes the way a rule names one entry of a list.
func at(list string, i int) string { return list + "[" + strconv.Itoa(i) + "]" }

// TestDotnetNotForTheRuntime covers files the runtime has nothing to do with.
func TestDotnetNotForTheRuntime(t *testing.T) {
	// A PE file carrying no such header says plainly that it is not one.
	b := newPE(true)
	b.put32(b.directoriesAt()-4, maxDataDirectories)
	wantInt(t, scanDotnet(t, b.data, "dotnet.is_dotnet"), 0, "whether the runtime knows it")

	// Something that is not a PE file at all leaves even that unanswered,
	// since there is nothing to look in.
	wantNothing(t, scanDotnet(t, []byte("not a file at all"), "dotnet.is_dotnet"),
		"whether the runtime knows it")
	wantNothing(t, scanDotnet(t, []byte("not a file at all"), "dotnet.version"),
		"what it was built against")
}

// TestDotnetVersionAndStreams covers the version the file was built against and
// the streams its metadata holds.
func TestDotnetVersionAndStreams(t *testing.T) {
	data := aDotnetFile(
		dotnetStreamEntry{"#~", tildeStream(0, nil)},
		dotnetStreamEntry{"#Strings", []byte("\x00hello\x00")},
		dotnetStreamEntry{"#GUID", make([]byte, guidSize)},
	)
	wantInt(t, scanDotnet(t, data, "dotnet.is_dotnet"), 1, "whether the runtime knows it")
	wantString(t, scanDotnet(t, data, "dotnet.version"), "v4.0.30319",
		"what it was built against")
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_streams"), 3, "how many streams")

	for i, name := range []string{"#~", "#Strings", "#GUID"} {
		wantString(t, scanDotnet(t, data, at("dotnet.streams", i)+".name"), name,
			"a stream's name")
	}
	for i, size := range []int64{int64(tildeHeaderSize), 7, guidSize} {
		wantInt(t, scanDotnet(t, data, at("dotnet.streams", i)+".size"), size,
			"how far a stream runs")
	}
	// Where a stream begins is given against the file, not the metadata.
	first := scanDotnet(t, data, at("dotnet.streams", 0)+".offset")
	if first.kind != valueInt || first.i <= 0 || first.i >= int64(len(data)) {
		t.Errorf("the first stream begins at %d, want somewhere within the file", first.i)
	}
}

// TestDotnetGUIDs covers the numbers naming the build, written out grouped.
func TestDotnetGUIDs(t *testing.T) {
	first := []byte{
		0x78, 0x56, 0x34, 0x12, 0x34, 0x12, 0x78, 0x56,
		0x9A, 0xBC, 0xDE, 0xF0, 0x11, 0x22, 0x33, 0x44,
	}
	data := aDotnetFile(dotnetStreamEntry{
		"#GUID", append(first, make([]byte, guidSize)...),
	})
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_guids"), 2, "how many numbers")
	wantString(t, scanDotnet(t, data, "dotnet.guids[0]"),
		"12345678-1234-5678-9abc-def011223344", "the first of them")
	wantString(t, scanDotnet(t, data, "dotnet.guids[1]"),
		"00000000-0000-0000-0000-000000000000", "the second of them")
}

// TestDotnetGUIDsLimited covers a stream of them longer than is read, which
// stops once as much of it as is read has been.
func TestDotnetGUIDsLimited(t *testing.T) {
	data := aDotnetFile(dotnetStreamEntry{
		"#GUID", make([]byte, guidBytesMax+guidSize*4),
	})
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_guids"),
		guidBytesMax/guidSize, "how many numbers")
}

// TestDotnetNoGUIDs covers a file carrying no such stream, which names none.
func TestDotnetNoGUIDs(t *testing.T) {
	data := aDotnetFile(dotnetStreamEntry{"#~", tildeStream(0, nil)})
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_guids"), 0, "how many numbers")
}

// TestDotnetUserStrings covers the text the file's own code quotes, kept as a
// run of lengths and bytes.
func TestDotnetUserStrings(t *testing.T) {
	body := []byte{0x00}
	for _, text := range []string{"one", "twotwo"} {
		body = append(body, byte(len(text)+1))
		body = append(body, text...)
		body = append(body, 0x00)
	}
	data := aDotnetFile(dotnetStreamEntry{"#US", body})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_user_strings"), 2, "how much text")
	wantString(t, scanDotnet(t, data, "dotnet.user_strings[0]"), "one", "the first of it")
	wantString(t, scanDotnet(t, data, "dotnet.user_strings[1]"), "twotwo", "the second of it")
}

// TestDotnetUserStringsLongForm covers an entry too long to state its length in
// one byte, which states it in two instead.
func TestDotnetUserStringsLongForm(t *testing.T) {
	text := make([]byte, 200)
	for i := range text {
		text[i] = 'x'
	}
	length := len(text) + 1
	body := append([]byte{0x00, byte(0x80 | length>>8), byte(length)}, text...)
	data := aDotnetFile(dotnetStreamEntry{"#US", append(body, 0x00)})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_user_strings"), 1, "how much text")
	if got := scanDotnet(t, data, "dotnet.user_strings[0]"); len(got.s) != len(text) {
		t.Errorf("the text is %d characters, want %d", len(got.s), len(text))
	}
}

// TestDotnetUserStringsRefused covers runs not shaped as they should be, which
// leave the question unanswered rather than half read.
func TestDotnetUserStringsRefused(t *testing.T) {
	cases := []struct {
		name string
		body []byte
	}{
		{"not opening with a nothing", []byte{0x01, 0x04, 'a', 'b', 'c'}},
		{"nothing at all", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := aDotnetFile(dotnetStreamEntry{"#US", c.body})
			wantNothing(t, scanDotnet(t, data, "dotnet.number_of_user_strings"),
				"how much text")
		})
	}
}

// tabled lays out a file whose metadata holds the given tables, over a stream
// of names and a stream of runs of bytes.
func tabled(names []byte, blobs []byte, tables []dotnetTableEntry) []byte {
	return aDotnetFile(
		dotnetStreamEntry{"#~", tildeStream(0, tables)},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", blobs},
	)
}

// TestDotnetModuleName covers the name the file gives itself, which the first
// table holds.
func TestDotnetModuleName(t *testing.T) {
	names, at := stringsStream("Thing.dll")
	blobs, _ := blobStream()
	// A row is a number, a name, and three of the numbers naming the build.
	row := joinBytes(u16(0), u16(at["Thing.dll"]), u16(1), u16(0), u16(0))
	data := tabled(names, blobs, []dotnetTableEntry{{tableModule, [][]byte{row}}})

	wantString(t, scanDotnet(t, data, "dotnet.module_name"), "Thing.dll",
		"the name it gives itself")
}

// TestDotnetModuleRefs covers the other files it leans on by name.
func TestDotnetModuleRefs(t *testing.T) {
	names, at := stringsStream("kernel32.dll", "user32.dll")
	blobs, _ := blobStream()
	rows := [][]byte{u16(at["kernel32.dll"]), u16(at["user32.dll"])}
	data := tabled(names, blobs, []dotnetTableEntry{{tableModuleRef, rows}})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_modulerefs"), 2, "how many it leans on")
	wantString(t, scanDotnet(t, data, "dotnet.modulerefs[0]"), "kernel32.dll", "the first")
	wantString(t, scanDotnet(t, data, "dotnet.modulerefs[1]"), "user32.dll", "the second")
}

// TestDotnetAssembly covers what the file says of itself as an assembly.
func TestDotnetAssembly(t *testing.T) {
	names, at := stringsStream("MyThing", "en-GB")
	blobs, _ := blobStream()
	// A hashing method, four numbers of version, some flags, a key, a name and
	// where it is meant for.
	row := joinBytes(u32(0x8004), u16(4), u16(3), u16(2), u16(1), u32(0),
		u16(0), u16(at["MyThing"]), u16(at["en-GB"]))
	data := tabled(names, blobs, []dotnetTableEntry{{tableAssembly, [][]byte{row}}})

	wantInt(t, scanDotnet(t, data, "dotnet.assembly.version.major"), 4, "the major number")
	wantInt(t, scanDotnet(t, data, "dotnet.assembly.version.minor"), 3, "the minor number")
	wantInt(t, scanDotnet(t, data, "dotnet.assembly.version.build_number"), 2, "the build")
	wantInt(t, scanDotnet(t, data, "dotnet.assembly.version.revision_number"), 1, "the revision")
	wantString(t, scanDotnet(t, data, "dotnet.assembly.name"), "MyThing", "its name")
	wantString(t, scanDotnet(t, data, "dotnet.assembly.culture"), "en-GB", "where it is meant for")
}

// TestDotnetAssemblyWithoutCulture covers one meant for nowhere in particular,
// which says so with an empty name and leaves the question unanswered.
func TestDotnetAssemblyWithoutCulture(t *testing.T) {
	names, at := stringsStream("MyThing")
	blobs, _ := blobStream()
	row := joinBytes(u32(0x8004), u16(1), u16(0), u16(0), u16(0), u32(0),
		u16(0), u16(at["MyThing"]), u16(0))
	data := tabled(names, blobs, []dotnetTableEntry{{tableAssembly, [][]byte{row}}})

	wantString(t, scanDotnet(t, data, "dotnet.assembly.name"), "MyThing", "its name")
	wantNothing(t, scanDotnet(t, data, "dotnet.assembly.culture"), "where it is meant for")
}

// TestDotnetAssemblyRefs covers the assemblies the file leans on.
func TestDotnetAssemblyRefs(t *testing.T) {
	names, at := stringsStream("mscorlib", "System")
	token := []byte{0xB7, 0x7A, 0x5C, 0x56, 0x19, 0x34, 0xE0, 0x89}
	blobs, blobAt := blobStream(token)
	// Four numbers of version, some flags, a key, a name, where it is meant
	// for, and a mark.
	first := joinBytes(u16(4), u16(0), u16(0), u16(0), u32(0),
		u16(blobAt[0]), u16(at["mscorlib"]), u16(0), u16(0))
	second := joinBytes(u16(2), u16(1), u16(0), u16(0), u32(0),
		u16(0), u16(at["System"]), u16(0), u16(0))
	data := tabled(names, blobs, []dotnetTableEntry{
		{tableAssemblyRef, [][]byte{first, second}},
	})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_assembly_refs"), 2, "how many it leans on")
	wantInt(t, scanDotnet(t, data, "dotnet.assembly_refs[0].version.major"), 4, "the major number")
	wantString(t, scanDotnet(t, data, "dotnet.assembly_refs[0].name"), "mscorlib", "the first name")
	wantString(t, scanDotnet(t, data, "dotnet.assembly_refs[0].public_key_or_token"),
		string(token), "the key it is known by")
	wantInt(t, scanDotnet(t, data, "dotnet.assembly_refs[1].version.minor"), 1, "the minor number")
	wantString(t, scanDotnet(t, data, "dotnet.assembly_refs[1].name"), "System", "the second name")
	// A key of nothing at all is passed over rather than given as empty.
	wantNothing(t, scanDotnet(t, data, "dotnet.assembly_refs[1].public_key_or_token"),
		"the key the second is known by")
}

// TestDotnetConstants covers the fixed text the file carries, of which only
// what is written as text is offered.
func TestDotnetConstants(t *testing.T) {
	names, _ := stringsStream()
	blobs, blobAt := blobStream([]byte("h\x00i\x00"), []byte{0x2A})
	// A kind, a byte set aside, what it belongs to, and the bytes themselves.
	text := joinBytes([]byte{elementTypeString, 0}, u16(0), u16(blobAt[0]))
	number := joinBytes([]byte{0x08, 0}, u16(0), u16(blobAt[1]))
	data := tabled(names, blobs, []dotnetTableEntry{
		{tableConstant, [][]byte{number, text, number}},
	})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_constants"), 1, "how many are text")
	wantString(t, scanDotnet(t, data, "dotnet.constants[0]"), "h\x00i\x00", "the text of it")
}

// TestDotnetFieldOffsets covers where in the file the fixed data sits.
func TestDotnetFieldOffsets(t *testing.T) {
	names, _ := stringsStream()
	blobs, _ := blobStream()
	// An address and which field it belongs to. The section runs from 0x1000
	// in memory and 0x400 in the file, so the two differ by 0xC00.
	rows := [][]byte{
		joinBytes(u32(0x1100), u16(1)),
		joinBytes(u32(0x1200), u16(2)),
		// One pointing nowhere within the file is passed over.
		joinBytes(u32(0x900000), u16(3)),
	}
	data := tabled(names, blobs, []dotnetTableEntry{{tableFieldRVA, rows}})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_field_offsets"), 2, "how many there are")
	wantInt(t, scanDotnet(t, data, "dotnet.field_offsets[0]"), 0x500, "the first")
	wantInt(t, scanDotnet(t, data, "dotnet.field_offsets[1]"), 0x600, "the second")
}

// typelibTables lays out the chain the number a file is known by is reached
// through: an attribute hanging off the assembly, naming a member, which
// belongs to a type called "GuidAttribute".
func typelibTables(names map[string]int, blobIndex int, parentTag, typeTag, classTag int) []dotnetTableEntry {
	return []dotnetTableEntry{
		// Where the type was found, its name, and where it belongs.
		{tableTypeRef, [][]byte{joinBytes(u16(0), u16(names[typelibName]), u16(0))}},
		// What the member belongs to, its name, and how it is written.
		{tableMemberRef, [][]byte{joinBytes(u16(classTag), u16(names[""]), u16(0))}},
		// What the attribute hangs off, which member names it, and its bytes.
		{tableCustomAttribute, [][]byte{
			joinBytes(u16(parentTag), u16(typeTag), u16(blobIndex)),
		}},
	}
}

// TestDotnetTypelib covers the number a file is known by to the older way of
// naming components, which is carried as an attribute on the assembly.
func TestDotnetTypelib(t *testing.T) {
	const known = "{12345678-1234-1234-1234-123456789012}"
	names, at := stringsStream(typelibName)
	// The bytes open with a fixed pair, then say how long the text is.
	value := append([]byte{0x01, 0x00, byte(len(known))}, known...)
	blobs, blobAt := blobStream(value)

	// The attribute hangs off the assembly, names a member, and that member
	// belongs to the first type.
	data := tabled(names, blobs, typelibTables(at, blobAt[0], tagAssembly, 1<<3|tagMemberRef, 1<<3|tagTypeRef))
	wantString(t, scanDotnet(t, data, "dotnet.typelib"), known,
		"the number it is known by")
}

// TestDotnetTypelibRefused covers chains that do not lead where they should,
// which leave the question unanswered rather than reading something else.
func TestDotnetTypelibRefused(t *testing.T) {
	const known = "{12345678-1234-1234-1234-123456789012}"
	names, at := stringsStream(typelibName)
	value := append([]byte{0x01, 0x00, byte(len(known))}, known...)
	blobs, blobAt := blobStream(value)

	cases := []struct {
		name                         string
		parentTag, typeTag, classTag int
	}{
		{
			"hanging off something other than the assembly",
			0x02, 1<<3 | tagMemberRef, 1<<3 | tagTypeRef,
		},
		{
			"named by something other than a member of another file",
			tagAssembly, 1<<3 | 0x02, 1<<3 | tagTypeRef,
		},
		{
			"a member belonging to something other than a type",
			tagAssembly, 1<<3 | tagMemberRef, 1<<3 | 0x02,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := tabled(names, blobs,
				typelibTables(at, blobAt[0], c.parentTag, c.typeTag, c.classTag))
			wantNothing(t, scanDotnet(t, data, "dotnet.typelib"),
				"the number it is known by")
		})
	}
}

// TestDotnetTypelibWrittenAsNothing covers an attribute whose text is written
// as nothing at all, which is read as being known by nothing rather than by
// whatever the bytes happen to hold.
func TestDotnetTypelibWrittenAsNothing(t *testing.T) {
	names, at := stringsStream(typelibName)
	blobs, blobAt := blobStream([]byte{0x01, 0x00, 0x04, 0xFF, 'a', 'b', 'c'})
	data := tabled(names, blobs,
		typelibTables(at, blobAt[0], tagAssembly, 1<<3|tagMemberRef, 1<<3|tagTypeRef))
	wantString(t, scanDotnet(t, data, "dotnet.typelib"), "",
		"the number it is known by")
}

// TestDotnetResources covers what the file keeps for itself, which sits after
// the metadata and is named by a table.
func TestDotnetResources(t *testing.T) {
	names, at := stringsStream("Thing.resources", "Other.resources")
	blobs, _ := blobStream()
	// Where it sits within what the file keeps, its flags, its name, and
	// whether it is held elsewhere.
	rows := [][]byte{
		joinBytes(u32(0), u32(0), u16(at["Thing.resources"]), u16(0)),
		// One held in another file altogether is passed over.
		joinBytes(u32(0), u32(0), u16(at["Other.resources"]), u16(1)),
	}
	data := tabled(names, blobs, []dotnetTableEntry{{tableManifestResource, rows}})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_resources"), 1, "how many it keeps")
	wantString(t, scanDotnet(t, data, "dotnet.resources[0].name"), "Thing.resources",
		"the name of the first")
	// What it keeps begins right after the metadata, opening with how long it
	// is, and what is offered is the bytes after that.
	length := scanDotnet(t, data, "dotnet.resources[0].length")
	place := scanDotnet(t, data, "dotnet.resources[0].offset")
	if length.kind != valueInt || place.kind != valueInt {
		t.Fatalf("the first is %v long at %v", length.kind, place.kind)
	}
	if place.i <= 0 || place.i >= int64(len(data)) {
		t.Errorf("the first begins at %d, want somewhere within the file", place.i)
	}
}

// TestDotnetWideIndexes covers metadata whose streams are large enough that the
// places within them are written in four bytes rather than two.
func TestDotnetWideIndexes(t *testing.T) {
	names, at := stringsStream("Wide.dll")
	blobs, _ := blobStream()
	// The low three bits of the heap sizes say which places are written wide.
	row := joinBytes(u16(0), u32(at["Wide.dll"]), u32(1), u32(0), u32(0))
	data := aDotnetFile(
		dotnetStreamEntry{"#~", tildeStream(0x07, []dotnetTableEntry{
			{tableModule, [][]byte{row}},
		})},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", blobs},
	)
	wantString(t, scanDotnet(t, data, "dotnet.module_name"), "Wide.dll",
		"the name it gives itself")
}

// TestDotnetTableNotKnown covers a table the walk does not know, which ends the
// walk rather than stepping over something of unknown size.
func TestDotnetTableNotKnown(t *testing.T) {
	names, at := stringsStream("Thing.dll")
	blobs, _ := blobStream()
	row := joinBytes(u16(0), u16(at["Thing.dll"]), u16(1), u16(0), u16(0))
	// The table of pointers to parameters is one libyara steps over by name;
	// anything past it is not read.
	data := tabled(names, blobs, []dotnetTableEntry{
		{tableModule, [][]byte{row}},
		{tableParamPtr, [][]byte{u16(0)}},
		{tableModuleRef, [][]byte{u16(at["Thing.dll"])}},
	})

	wantString(t, scanDotnet(t, data, "dotnet.module_name"), "Thing.dll",
		"the name read before the walk stopped")
	wantNothing(t, scanDotnet(t, data, "dotnet.number_of_modulerefs"),
		"how many it leans on")
}

// TestDotnetTooManyRows covers a table claiming more rows than any real file
// holds, which ends the walk rather than being trusted.
func TestDotnetTooManyRows(t *testing.T) {
	names, _ := stringsStream()
	blobs, _ := blobStream()
	tables := []dotnetTableEntry{{tableModuleRef, [][]byte{u16(0)}}}
	stream := tildeStream(0, tables)
	// The count sits right after the header.
	binary.LittleEndian.PutUint32(stream[tildeHeaderSize:], maxTableRows+1)
	data := aDotnetFile(
		dotnetStreamEntry{"#~", stream},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", blobs},
	)
	wantNothing(t, scanDotnet(t, data, "dotnet.number_of_modulerefs"),
		"how many it leans on")
}

// everyTableWidth is how wide one row of each table is when every place into a
// stream or a table takes two bytes, worked out from the shapes the tables are
// documented to have rather than from the code being tested.
var everyTableWidth = map[int]int{
	tableModule: 10, tableTypeRef: 6, tableTypeDef: 14, tableFieldPtr: 2,
	tableField: 6, tableMethodDefPtr: 2, tableMethodDef: 14, tableParam: 6,
	tableInterfaceImpl: 4, tableMemberRef: 6, tableConstant: 6,
	tableCustomAttribute: 6, tableFieldMarshal: 4, tableDeclSecurity: 6,
	tableClassLayout: 8, tableFieldLayout: 6, tableStandAloneSig: 2,
	tableEventMap: 4, tableEventPtr: 2, tableEvent: 6, tablePropertyMap: 4,
	tablePropertyPtr: 2, tableProperty: 6, tableMethodSemantics: 6,
	tableMethodImpl: 6, tableModuleRef: 2, tableTypeSpec: 2, tableImplMap: 8,
	tableFieldRVA: 6, tableENCLog: 8, tableENCMap: 4, tableAssembly: 22,
	tableAssemblyProcessor: 4, tableAssemblyOS: 12, tableAssemblyRef: 20,
	tableAssemblyRefProcessor: 6, tableAssemblyRefOS: 14, tableFile: 8,
	tableExportedType: 14, tableManifestResource: 12, tableNestedClass: 4,
	tableGenericParam: 8, tableMethodSpec: 4, tableGenericParamConstraint: 4,
}

// TestDotnetWalksEveryTable covers a file holding every table the walk knows.
// Each is stepped over by its own width, so getting any one of them wrong puts
// every table after it out of place. Reading something out of the last table
// that carries anything is what shows the whole walk stayed in step.
func TestDotnetWalksEveryTable(t *testing.T) {
	names, at := stringsStream("Thing.dll", "kernel32.dll", "Thing.resources")
	blobs, _ := blobStream()

	var tables []dotnetTableEntry
	for bit := range tableCount {
		width, known := everyTableWidth[bit]
		if !known {
			continue
		}
		row := make([]byte, width)
		switch bit {
		case tableModule:
			copy(row[2:], u16(at["Thing.dll"]))
		case tableModuleRef:
			copy(row, u16(at["kernel32.dll"]))
		case tableManifestResource:
			copy(row[8:], u16(at["Thing.resources"]))
		}
		tables = append(tables, dotnetTableEntry{bit, [][]byte{row}})
	}
	data := tabled(names, blobs, tables)

	// The first table the walk reads, and one near the very end of it.
	wantString(t, scanDotnet(t, data, "dotnet.module_name"), "Thing.dll",
		"the name it gives itself")
	wantString(t, scanDotnet(t, data, "dotnet.modulerefs[0]"), "kernel32.dll",
		"the file it leans on")
	wantString(t, scanDotnet(t, data, "dotnet.resources[0].name"), "Thing.resources",
		"the name of what it keeps")
}

// TestDotnetNotForTheRuntimeRefused covers files carrying something like the
// runtime's header that is not one.
func TestDotnetNotForTheRuntimeRefused(t *testing.T) {
	spoil := map[string]func(b *peBuilder){
		"a header of a length it does not have": func(b *peBuilder) {
			b.put32(cliHeaderPlace(b), cliHeaderSize+1)
		},
		"a header pointing nowhere": func(b *peBuilder) {
			b.put32(b.directoriesAt()+directoryCLI*dataDirectoryEntry, 0x900000)
		},
		"metadata pointing nowhere": func(b *peBuilder) {
			b.put32(cliHeaderPlace(b)+cliMetadataAt, 0x900000)
		},
		"metadata not opening as it should": func(b *peBuilder) {
			b.put32(metadataPlace(b), 0x11223344)
		},
		"a version of no length at all": func(b *peBuilder) {
			b.put32(metadataPlace(b)+metadataLengthAt, 0)
		},
		"a version longer than one may be": func(b *peBuilder) {
			b.put32(metadataPlace(b)+metadataLengthAt, versionMax+1)
		},
		"a version that is not a whole number of words": func(b *peBuilder) {
			b.put32(metadataPlace(b)+metadataLengthAt, 6)
		},
		"a version running past the end": func(b *peBuilder) {
			b.put32(metadataPlace(b)+metadataLengthAt, 252)
			b.data = b.data[:metadataPlace(b)+metadataHeaderSize+8]
		},
		"too few tables of tables claimed": func(b *peBuilder) {
			b.put32(b.directoriesAt()-4, directoryCLI)
		},
	}
	for name, spoiler := range spoil {
		t.Run(name, func(t *testing.T) {
			b := newPE(true)
			b.withDotnet("v4.0.30319", []dotnetStreamEntry{{"#~", tildeStream(0, nil)}})
			spoiler(b)
			wantInt(t, scanDotnet(t, b.data, "dotnet.is_dotnet"), 0,
				"whether the runtime knows it")
		})
	}
}

// cliHeaderPlace and metadataPlace are where those two sit in a test file.
func cliHeaderPlace(b *peBuilder) int { return 0x400 }
func metadataPlace(b *peBuilder) int  { return 0x400 + cliHeaderSize }

// TestDotnetNarrowFile covers files built for narrower addresses, which are
// known for the runtime a different way: a library is taken as one, and
// anything else has to begin by jumping straight out to the runtime.
func TestDotnetNarrowFile(t *testing.T) {
	build := func(library bool, entry []byte) []byte {
		b := newPE(false)
		if library {
			b.put16(b.nt+peSignatureSize+18, 0x2000)
		}
		b.withDotnet("v4.0.30319", []dotnetStreamEntry{{"#~", tildeStream(0, nil)}})
		if entry != nil {
			// The entry point is put well past the runtime's header, so that
			// writing it there does not spoil the header itself.
			b.put32(b.opt+16, 0x3000)
			copy(b.data[0x400+0x2000:], entry)
		}
		return b.data
	}
	cases := []struct {
		name string
		data []byte
		want int64
	}{
		{"a library", build(true, nil), 1},
		{"one jumping out to the runtime", build(false, []byte{0xFF, 0x25}), 1},
		{"one beginning some other way", build(false, []byte{0x90, 0x90}), 0},
		{"one whose beginning is nowhere", build(false, nil), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantInt(t, scanDotnet(t, c.data, "dotnet.is_dotnet"), c.want,
				"whether the runtime knows it")
		})
	}
}

// TestDotnetStreamsRefused covers metadata whose stream headers do not hold
// together, which stops where they stop making sense.
func TestDotnetStreamsRefused(t *testing.T) {
	// A file claiming more streams than it carries stops where the headers stop
	// making sense: a name that never closes off within the room set aside for
	// it is not a stream at all.
	b := newPE(true)
	b.withDotnet("v4.0.30319", []dotnetStreamEntry{{"#~", tildeStream(0, nil)}})
	countAt := metadataPlace(b) + metadataHeaderSize + 12 + 2
	b.data[countAt] = 4
	for i := range 64 {
		b.data[metadataPlace(b)+metadataHeaderSize+12+4+streamHeaderSize+4+i] = 0xFF
	}
	wantInt(t, scanDotnet(t, b.data, "dotnet.number_of_streams"), 1, "how many streams")

	// One whose headers are cut off entirely names none at all.
	b = newPE(true)
	b.withDotnet("v4.0.30319", []dotnetStreamEntry{{"#~", tildeStream(0, nil)}})
	b.data = b.data[:metadataPlace(b)+metadataHeaderSize+12+2]
	wantInt(t, scanDotnet(t, b.data, "dotnet.number_of_streams"), 0, "how many streams")
}

// TestDotnetTablesNeedTheOtherStreams covers metadata holding tables but none
// of the streams they point into, which leaves the tables unread.
func TestDotnetTablesNeedTheOtherStreams(t *testing.T) {
	names, at := stringsStream("Thing.dll")
	row := joinBytes(u16(0), u16(at["Thing.dll"]), u16(1), u16(0), u16(0))
	tables := []dotnetTableEntry{{tableModule, [][]byte{row}}}

	// Without the stream of names, and without the stream of runs of bytes.
	for _, streams := range [][]dotnetStreamEntry{
		{{"#~", tildeStream(0, tables)}, {"#Blob", []byte{0}}},
		{{"#~", tildeStream(0, tables)}, {"#Strings", names}},
	} {
		data := aDotnetFile(streams...)
		wantNothing(t, scanDotnet(t, data, "dotnet.module_name"),
			"the name it gives itself")
	}

	// And a stream of tables too short to hold even its own header.
	data := aDotnetFile(
		dotnetStreamEntry{"#~", []byte{1, 2, 3}},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", []byte{0}},
	)
	wantNothing(t, scanDotnet(t, data, "dotnet.module_name"), "the name it gives itself")
}

// TestDotnetLongBlobLength covers a run of bytes stating its length in four
// bytes, which is the widest form there is.
func TestDotnetLongBlobLength(t *testing.T) {
	text := make([]byte, 0x4000)
	for i := range text {
		text[i] = 'z'
	}
	length := len(text) + 1
	body := append([]byte{
		0x00, byte(0xC0 | length>>24), byte(length >> 16),
		byte(length >> 8), byte(length),
	}, text...)
	data := aDotnetFile(dotnetStreamEntry{"#US", append(body, 0)})

	wantInt(t, scanDotnet(t, data, "dotnet.number_of_user_strings"), 1, "how much text")
	if got := scanDotnet(t, data, "dotnet.user_strings[0]"); len(got.s) != len(text) {
		t.Errorf("the text is %d characters, want %d", len(got.s), len(text))
	}
}

// TestDotnetBlobLengthRefused covers lengths that cannot be read, which stop
// the run rather than being guessed at.
func TestDotnetBlobLengthRefused(t *testing.T) {
	cases := map[string][]byte{
		"a length in a form there is not": {0x00, 0xF8, 0x01},
		"a two-byte length cut short":     {0x00, 0x80},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			data := aDotnetFile(dotnetStreamEntry{"#US", body})
			wantInt(t, scanDotnet(t, data, "dotnet.number_of_user_strings"), 0,
				"how much text")
		})
	}
}

// TestDotnetNamesRefused covers places in the stream of names that point
// nowhere useful, which give no name rather than any old bytes.
func TestDotnetNamesRefused(t *testing.T) {
	// A name never closed off runs further than any name may.
	long := make([]byte, dotnetNameMax+8)
	for i := range long {
		long[i] = 'q'
	}
	names := append([]byte{0}, long...)
	blobs, _ := blobStream()
	row := joinBytes(u16(0), u16(1), u16(0), u16(0), u16(0))
	data := tabled(names, blobs, []dotnetTableEntry{{tableModule, [][]byte{row}}})
	wantNothing(t, scanDotnet(t, data, "dotnet.module_name"), "the name it gives itself")

	// And one beginning past the end of the file gives nothing either.
	shorter, _ := stringsStream("Thing.dll")
	row = joinBytes(u16(0), u16(0xFFFF), u16(0), u16(0), u16(0))
	data = tabled(shorter, blobs, []dotnetTableEntry{{tableModule, [][]byte{row}}})
	wantNothing(t, scanDotnet(t, data, "dotnet.module_name"), "the name it gives itself")
}

// TestDotnetWideTableIndexes covers a table holding so many rows that a place
// into it has to be written in four bytes rather than two.
func TestDotnetWideTableIndexes(t *testing.T) {
	t.Run("a place into a wide table", func(t *testing.T) {
		var tables dotnetTables
		tables.rows[tableField] = wideRows + 1
		if got := tables.coded(1, tableField, tableParam); got != 4 {
			t.Errorf("a place into it takes %d bytes, want 4", got)
		}
		if got := tables.coded(1, tableParam); got != 2 {
			t.Errorf("a place into a small table takes %d bytes, want 2", got)
		}
	})
}

// aRichDotnetFile lays out a file exercising every part of the module at once:
// tables from one end of the walk to the other, text, numbers, and the chain
// the number it is known by is reached through.
func aRichDotnetFile() []byte {
	const known = "{12345678-1234-1234-1234-123456789012}"
	names, at := stringsStream(typelibName, "Rich.dll", "kernel32.dll",
		"Rich.resources", "Rich", "en-GB", "mscorlib")
	blobs, blobAt := blobStream(
		append([]byte{0x01, 0x00, byte(len(known))}, known...),
		func() []byte {
			long := make([]byte, 300)
			for i := range long {
				long[i] = 't'
			}
			return long
		}(),
		[]byte{0xB7, 0x7A, 0x5C, 0x56},
	)
	tables := []dotnetTableEntry{
		{tableModule, [][]byte{joinBytes(u16(0), u16(at["Rich.dll"]), u16(1), u16(0), u16(0))}},
		{tableTypeRef, [][]byte{joinBytes(u16(0), u16(at[typelibName]), u16(0))}},
		{tableMemberRef, [][]byte{joinBytes(u16(1<<3|tagTypeRef), u16(0), u16(0))}},
		{tableConstant, [][]byte{
			joinBytes([]byte{elementTypeString, 0}, u16(0), u16(blobAt[1])),
		}},
		{tableCustomAttribute, [][]byte{
			joinBytes(u16(tagAssembly), u16(1<<3|tagMemberRef), u16(blobAt[0])),
		}},
		{tableModuleRef, [][]byte{u16(at["kernel32.dll"])}},
		{tableFieldRVA, [][]byte{joinBytes(u32(0x1100), u16(1))}},
		{tableAssembly, [][]byte{joinBytes(u32(0x8004), u16(4), u16(3), u16(2), u16(1),
			u32(0), u16(0), u16(at["Rich"]), u16(at["en-GB"]))}},
		{tableAssemblyRef, [][]byte{joinBytes(u16(4), u16(0), u16(0), u16(0), u32(0),
			u16(blobAt[2]), u16(at["mscorlib"]), u16(0), u16(0))}},
		{tableManifestResource, [][]byte{
			joinBytes(u32(0), u32(0), u16(at["Rich.resources"]), u16(0)),
		}},
	}
	return aDotnetFile(
		dotnetStreamEntry{"#~", tildeStream(0, tables)},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", blobs},
		dotnetStreamEntry{"#GUID", make([]byte, guidSize)},
		dotnetStreamEntry{"#US", func() []byte {
			long := make([]byte, 200)
			for i := range long {
				long[i] = 'y'
			}
			body := []byte{0x00, 0x03, 'h', 'i', 0x00}
			body = append(body, byte(0x80|(len(long)+1)>>8), byte(len(long)+1))
			return append(append(body, long...), 0x00)
		}()},
	)
}

// dotnetFieldsRead is every field worth reading out of such a file, which the
// tests below read over and over as the file is worn down.
var dotnetFieldsRead = []string{
	"dotnet.is_dotnet", "dotnet.version", "dotnet.module_name",
	"dotnet.number_of_streams", "dotnet.number_of_guids",
	"dotnet.number_of_user_strings", "dotnet.number_of_constants",
	"dotnet.number_of_modulerefs", "dotnet.number_of_field_offsets",
	"dotnet.number_of_assembly_refs", "dotnet.number_of_resources",
	"dotnet.typelib", "dotnet.assembly.name", "dotnet.assembly.culture",
	"dotnet.streams[0].name", "dotnet.guids[0]", "dotnet.user_strings[0]",
	"dotnet.constants[0]", "dotnet.modulerefs[0]", "dotnet.field_offsets[0]",
	"dotnet.assembly_refs[0].name", "dotnet.assembly_refs[0].public_key_or_token",
	"dotnet.resources[0].name", "dotnet.resources[0].offset",
}

// TestDotnetRichFile covers a file exercising every part of the module at once,
// so that the pieces are known to work together and not only one at a time.
func TestDotnetRichFile(t *testing.T) {
	data := aRichDotnetFile()
	wantString(t, scanDotnet(t, data, "dotnet.module_name"), "Rich.dll", "its own name")
	wantString(t, scanDotnet(t, data, "dotnet.typelib"),
		"{12345678-1234-1234-1234-123456789012}", "the number it is known by")
	wantString(t, scanDotnet(t, data, "dotnet.assembly.name"), "Rich", "its assembly")
	wantString(t, scanDotnet(t, data, "dotnet.assembly.culture"), "en-GB", "where it is meant for")
	wantString(t, scanDotnet(t, data, "dotnet.modulerefs[0]"), "kernel32.dll", "what it leans on")
	if got := scanDotnet(t, data, "dotnet.constants[0]"); len(got.s) != 300 {
		t.Errorf("its fixed text is %d characters, want 300", len(got.s))
	}
	wantString(t, scanDotnet(t, data, "dotnet.assembly_refs[0].name"), "mscorlib", "what it borrows")
	wantString(t, scanDotnet(t, data, "dotnet.resources[0].name"), "Rich.resources", "what it keeps")
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_field_offsets"), 1, "where its data sits")
	wantString(t, scanDotnet(t, data, "dotnet.user_strings[0]"), "hi", "the text it quotes")
}

// TestDotnetWornDownFile covers the same file cut short at every length. A file
// can stop anywhere, and every part of the module has to give up gracefully
// wherever that happens rather than reading past the end of what is there.
func TestDotnetWornDownFile(t *testing.T) {
	whole := aRichDotnetFile()
	for length := len(whole); length > 0; length -= 7 {
		for _, field := range dotnetFieldsRead {
			// Reading must not reach past the end whatever is missing; what it
			// comes to is checked on the whole file above.
			scanDotnet(t, whole[:length], field)
		}
	}
}

// TestDotnetWornDownWithinTheMetadata covers the same file cut short byte by
// byte through the metadata itself, where every place one part of it points at
// another can be made to fall past the end of what is left.
func TestDotnetWornDownWithinTheMetadata(t *testing.T) {
	whole := aRichDotnetFile()
	// The metadata begins right after the runtime's header, which sits at the
	// front of the only section.
	const metadataBegins = 0x400 + cliHeaderSize
	fields := []string{
		"dotnet.module_name", "dotnet.typelib", "dotnet.assembly.name",
		"dotnet.constants[0]", "dotnet.resources[0].offset",
		"dotnet.user_strings[0]", "dotnet.number_of_streams",
	}
	for length := metadataBegins; length < metadataBegins+900; length++ {
		for _, field := range fields {
			scanDotnet(t, whole[:length], field)
		}
	}
}

// TestDotnetNarrowFileWornDown covers a file built for narrower addresses cut
// short, where what marks it for the runtime cannot be read at all.
func TestDotnetNarrowFileWornDown(t *testing.T) {
	b := newPE(false)
	b.withDotnet("v4.0.30319", []dotnetStreamEntry{{"#~", tildeStream(0, nil)}})
	b.put32(b.opt+16, 0x3000)
	for length := len(b.data); length > 0x40; length -= 5 {
		scanDotnet(t, b.data[:length], "dotnet.is_dotnet")
	}
}

// TestDotnetTableClaimingWideRows covers a table claiming more rows than two
// bytes can count, which makes every place into it wider even though the walk
// then gives up on so large a claim.
func TestDotnetTableClaimingWideRows(t *testing.T) {
	names, at := stringsStream("Thing.dll")
	blobs, _ := blobStream()
	tables := []dotnetTableEntry{
		{tableField, [][]byte{joinBytes(u16(0), u16(0), u16(0))}},
		{tableModuleRef, [][]byte{u16(at["Thing.dll"])}},
	}
	stream := tildeStream(0, tables)
	// The counts follow the header, lowest table first.
	binary.LittleEndian.PutUint32(stream[tildeHeaderSize:], wideRows+1)
	data := aDotnetFile(
		dotnetStreamEntry{"#~", stream},
		dotnetStreamEntry{"#Strings", names},
		dotnetStreamEntry{"#Blob", blobs},
	)
	wantInt(t, scanDotnet(t, data, "dotnet.is_dotnet"), 1, "whether the runtime knows it")
	wantNothing(t, scanDotnet(t, data, "dotnet.number_of_modulerefs"),
		"how many it leans on")
}

// TestDotnetResourceOutOfReach covers a resource the file says sits somewhere
// it does not, which is passed over rather than read from nowhere.
func TestDotnetResourceOutOfReach(t *testing.T) {
	names, at := stringsStream("Gone.resources")
	blobs, _ := blobStream()
	rows := [][]byte{
		// Beginning past the end of the file altogether.
		joinBytes(u32(0x900000), u32(0), u16(at["Gone.resources"]), u16(0)),
		// And one beginning just past the end of what the file holds.
		joinBytes(u32(0x4400), u32(0), u16(at["Gone.resources"]), u16(0)),
		// And one beginning within the file but saying it runs on past the end.
		joinBytes(u32(0), u32(0), u16(at["Gone.resources"]), u16(0)),
	}
	data := tabled(names, blobs, []dotnetTableEntry{{tableManifestResource, rows}})
	// The third begins where the file keeps things, which the runtime's header
	// names, and is told there that it runs on well past the end of the file.
	address := binary.LittleEndian.Uint32(data[0x400+cliResourcesAt:])
	binary.LittleEndian.PutUint32(data[int(address)-0xC00:], 1<<24)
	wantInt(t, scanDotnet(t, data, "dotnet.number_of_resources"), 0, "how many it keeps")
}

// TestDotnetTypelibWithoutItsChain covers a file whose attributes cannot be
// followed back, because the tables they lead through are not there at all.
func TestDotnetTypelibWithoutItsChain(t *testing.T) {
	names, _ := stringsStream(typelibName)
	blobs, blobAt := blobStream([]byte{0x01, 0x00, 0x01, 'x'})
	data := tabled(names, blobs, []dotnetTableEntry{
		{tableCustomAttribute, [][]byte{
			joinBytes(u16(tagAssembly), u16(1<<3|tagMemberRef), u16(blobAt[0])),
		}},
	})
	wantNothing(t, scanDotnet(t, data, "dotnet.typelib"), "the number it is known by")
}

// TestDotnetTypelibPointingNowhere covers an attribute naming a member or a
// type that the file does not carry, and one whose bytes are not there.
func TestDotnetTypelibPointingNowhere(t *testing.T) {
	names, at := stringsStream(typelibName)
	blobs, blobAt := blobStream([]byte{0x01, 0x00, 0x01, 'x'})
	cases := map[string][]dotnetTableEntry{
		"naming a member the file does not carry": typelibTables(
			at, blobAt[0], tagAssembly, 0x1FFF<<3|tagMemberRef, 1<<3|tagTypeRef),
		"naming a type the file does not carry": typelibTables(
			at, blobAt[0], tagAssembly, 1<<3|tagMemberRef, 0x1FFF<<3|tagTypeRef),
		"whose bytes are nowhere": typelibTables(
			at, 0, tagAssembly, 1<<3|tagMemberRef, 1<<3|tagTypeRef),
	}
	for name, tables := range cases {
		t.Run(name, func(t *testing.T) {
			data := tabled(names, blobs, tables)
			wantNothing(t, scanDotnet(t, data, "dotnet.typelib"),
				"the number it is known by")
		})
	}
}

// TestDotnetTypelibSayingMoreThanItHas covers an attribute claiming its text is
// longer than the bytes it carries.
func TestDotnetTypelibSayingMoreThanItHas(t *testing.T) {
	names, at := stringsStream(typelibName)
	blobs, blobAt := blobStream([]byte{0x01, 0x00, 0x40, 'x'})
	data := tabled(names, blobs,
		typelibTables(at, blobAt[0], tagAssembly, 1<<3|tagMemberRef, 1<<3|tagTypeRef))
	wantNothing(t, scanDotnet(t, data, "dotnet.typelib"), "the number it is known by")
}
