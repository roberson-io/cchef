package yara

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// What a file built for the common language runtime carries beyond being a PE
// file: a header naming its metadata, and within that metadata a set of streams
// holding the names, the runs of bytes, and the tables the runtime works from.

const (
	// directoryCLI names the header marking a file as one the runtime knows.
	directoryCLI = 14
	// cliHeaderSize is how long that header is, which it also states; a file
	// stating anything else is not read.
	cliHeaderSize = 72
	// Where within that header the metadata and the file's own resources are
	// named.
	cliMetadataAt  = 8
	cliResourcesAt = 24
	// metadataMagic opens the metadata, reading as "BSJB".
	metadataMagic = 0x424A5342
	// metadataHeaderSize is what comes before the version: the mark, a version
	// of the metadata itself, something set aside, and how long the version is.
	metadataHeaderSize = 16
	metadataLengthAt   = 12
	// versionMax is the longest version a file may state, which must also be a
	// whole number of words.
	versionMax  = 255
	versionWord = 4
	// streamHeaderSize is what opens each stream: where it is and how long,
	// with its name following.
	streamHeaderSize = 8
	// streamNameMax is the most room a stream's name may take.
	streamNameMax = 32
	// guidSize is how long one of the numbers naming a build is, and
	// guidBytesMax how much of the stream of them is read.
	guidSize     = 16
	guidBytesMax = 256
	// dotnetNameMax is the longest name read out of the stream of them.
	dotnetNameMax = 1024
)

// dotnetSchema is what the dotnet module declares.
func dotnetSchema() *modDecl {
	version := func() *modDecl {
		return decStruct(map[string]*modDecl{
			"major": decInt(), "minor": decInt(),
			"build_number": decInt(), "revision_number": decInt(),
		})
	}
	return decStruct(map[string]*modDecl{
		"is_dotnet": decInt(), "version": decString(), "module_name": decString(),
		"streams": decArray(decStruct(map[string]*modDecl{
			"name": decString(), "offset": decInt(), "size": decInt(),
		})),
		"number_of_streams": decInt(),
		"guids":             decArray(decString()),
		"number_of_guids":   decInt(),
		"resources": decArray(decStruct(map[string]*modDecl{
			"offset": decInt(), "length": decInt(), "name": decString(),
		})),
		"number_of_resources": decInt(),
		"assembly_refs": decArray(decStruct(map[string]*modDecl{
			"version":             version(),
			"public_key_or_token": decString(), "name": decString(),
		})),
		"number_of_assembly_refs": decInt(),
		"assembly": decStruct(map[string]*modDecl{
			"version": version(), "name": decString(), "culture": decString(),
		}),
		"modulerefs":              decArray(decString()),
		"number_of_modulerefs":    decInt(),
		"user_strings":            decArray(decString()),
		"number_of_user_strings":  decInt(),
		"typelib":                 decString(),
		"constants":               decArray(decString()),
		"number_of_constants":     decInt(),
		"field_offsets":           decArray(decInt()),
		"number_of_field_offsets": decInt(),
	})
}

// dotnetFile is a file being read as one the runtime knows: the PE file
// underneath, where its metadata begins, and the streams it holds.
type dotnetFile struct {
	pe *peFile
	// root is where the metadata begins, which every stream is placed against.
	root int
	// cli is where the header marking the file for the runtime begins.
	cli     int
	streams []dotnetStream
}

// dotnetStream is one stream of the metadata.
type dotnetStream struct {
	name string
	// at is where the stream begins within the file, and size how far it runs.
	at, size int
}

// dotnetModule reads the data as a file the runtime knows. Data that is not a
// PE file at all leaves every part unanswered, since there is nothing to look
// in at all.
func dotnetModule(e *evaluator) modValue {
	fields := map[string]modValue{}
	pe, isPE := readPE(e.buf.data)
	if !isPE {
		return structOf(fields)
	}
	f, known := readDotnet(pe)
	if !known {
		fields["is_dotnet"] = valueOf(intValue(0))
		return structOf(fields)
	}
	fields["is_dotnet"] = valueOf(intValue(1))
	f.addVersion(fields)
	f.addStreams(fields)
	f.addGUIDs(fields)
	f.addUserStrings(fields)
	f.addTables(fields)
	return structOf(fields)
}

// readDotnet says whether a PE file is one the runtime knows, and if so where
// its metadata begins.
func readDotnet(pe *peFile) (*dotnetFile, bool) {
	cli, ok := pe.cliHeader()
	if !ok {
		return nil, false
	}
	address, _ := pe.u32(cli + cliMetadataAt)
	root, found := pe.rvaToOffset(uint64(address))
	if !found || !pe.holds(int(root), metadataHeaderSize) {
		return nil, false
	}
	if magic, _ := pe.u32(int(root)); magic != metadataMagic {
		return nil, false
	}
	length, _ := pe.u32(int(root) + metadataLengthAt)
	if length == 0 || length > versionMax || length%versionWord != 0 ||
		!pe.holds(int(root)+metadataHeaderSize, int(length)) {
		return nil, false
	}
	if !pe.runsManaged() {
		return nil, false
	}
	return &dotnetFile{pe: pe, root: int(root), cli: cli}, true
}

// cliHeader finds the header marking a file as one the runtime knows.
func (f *peFile) cliHeader() (int, bool) {
	address, _, ok := f.directoryEntry(directoryCLI)
	if !ok {
		return 0, false
	}
	at, found := f.rvaToOffset(uint64(address))
	if !found || !f.holds(int(at), cliHeaderSize) {
		return 0, false
	}
	// The header states its own length, and one stating anything else is not
	// the header this reads.
	if stated, _ := f.u32(int(at)); stated != cliHeaderSize {
		return 0, false
	}
	return int(at), true
}

// runsManaged is the last of the checks marking a file as one the runtime
// knows, and it differs by how the file was built. A wide file must claim the
// entry naming that header among its tables. A narrow one that is not a library
// must begin by jumping straight out to the runtime.
func (f *peFile) runsManaged() bool {
	if f.wide {
		count, ok := f.u32(f.dataDirectoriesAt() - 4)
		return ok && count >= directoryCLI
	}
	// The whole of the file header is there, or this would not have been read
	// as a PE file at all.
	characteristics, _ := f.u16(f.nt + peSignatureSize + 18)
	const isLibrary = 0x2000
	if characteristics&isLibrary != 0 {
		return true
	}
	// Likewise the optional header, which the table of tables was already read
	// out of the far end of.
	entry, _ := f.u32(f.opt + 16)
	at, found := f.rvaToOffset(uint64(entry))
	if !found || !f.holds(int(at), 2) {
		return false
	}
	// The jump the runtime is entered by.
	return f.data[at] == 0xFF && f.data[at+1] == 0x25
}

// holds says whether a stretch of the given length beginning somewhere lies
// wholly within the file.
func (f *peFile) holds(at, length int) bool {
	return at >= 0 && length >= 0 && at <= len(f.data) && length <= len(f.data)-at
}

// addVersion offers the version of the runtime the file was built against,
// written after the metadata's own header and padded out to a whole number of
// words.
func (f *dotnetFile) addVersion(fields map[string]modValue) {
	length, _ := f.pe.u32(f.root + metadataLengthAt)
	text := f.pe.data[f.root+metadataHeaderSize:][:length]
	// The stated length takes in the closing nothing and the padding after it,
	// so the version proper ends at the first of those.
	if end := indexByte(text, 0); end >= 0 {
		fields["version"] = valueOf(stringValue(string(text[:end])))
	}
}

// addStreams offers each stream of the metadata: its name, where it begins, and
// how far it runs.
func (f *dotnetFile) addStreams(fields map[string]modValue) {
	length, _ := f.pe.u32(f.root + metadataLengthAt)
	// Two bytes of flags follow the version, then how many streams there are.
	// Only the low byte of that count is read, which is what libyara reads.
	at := f.root + metadataHeaderSize + int(length) + 2
	if !f.pe.holds(at, 2) {
		fields["number_of_streams"] = valueOf(intValue(0))
		return
	}
	count := int(f.pe.data[at])
	at += 2

	items := make([]modValue, 0, count)
	for range count {
		stream, next, ok := f.readStream(at)
		if !ok {
			break
		}
		items = append(items, structOf(map[string]modValue{
			"name":   valueOf(stringValue(stream.name)),
			"offset": valueOf(intValue(int64(stream.at))),
			"size":   valueOf(intValue(int64(stream.size))),
		}))
		f.streams = append(f.streams, stream)
		at = next
	}
	fields["streams"] = listOf(items)
	fields["number_of_streams"] = valueOf(intValue(int64(len(items))))
}

// readStream reads one stream's header and says where the next begins. The name
// is closed off within the room set aside for it and padded to a whole word.
func (f *dotnetFile) readStream(at int) (dotnetStream, int, bool) {
	if !f.pe.holds(at, streamHeaderSize) ||
		!f.pe.holds(at+streamHeaderSize, streamNameMax) {
		return dotnetStream{}, 0, false
	}
	room := f.pe.data[at+streamHeaderSize:][:streamNameMax]
	end := indexByte(room, 0)
	if end < 0 {
		return dotnetStream{}, 0, false
	}
	name := string(room[:end])
	offset, _ := f.pe.u32(at)
	size, _ := f.pe.u32(at + 4)
	// Where a stream begins is stated against the metadata rather than the
	// file, so it is placed against the file here once and for all.
	stream := dotnetStream{name: name, at: f.root + int(offset), size: int(size)}
	next := at + streamHeaderSize + len(name) + versionWord - len(name)%versionWord
	return stream, next, true
}

// namedStream finds the first stream whose name opens with one of those given,
// which is how libyara picks them out.
func (f *dotnetFile) namedStream(names ...string) (dotnetStream, bool) {
	for _, stream := range f.streams {
		for _, name := range names {
			if strings.HasPrefix(stream.name, name) {
				return stream, true
			}
		}
	}
	return dotnetStream{}, false
}

// addGUIDs offers the numbers naming the build, each written out grouped.
func (f *dotnetFile) addGUIDs(fields map[string]modValue) {
	var items []modValue
	if stream, found := f.namedStream("#GUID"); found {
		room := min(stream.size, guidBytesMax)
		for at := stream.at; room >= guidSize && f.pe.holds(at, guidSize); at += guidSize {
			items = append(items, valueOf(stringValue(guidText(f.pe.data[at:]))))
			room -= guidSize
		}
	}
	fields["guids"] = listOf(items)
	fields["number_of_guids"] = valueOf(intValue(int64(len(items))))
}

// guidText writes one of those numbers out: three groups read as numbers, then
// the rest byte by byte.
func guidText(b []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		binary.LittleEndian.Uint32(b), binary.LittleEndian.Uint16(b[4:]),
		binary.LittleEndian.Uint16(b[6:]),
		b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15])
}

// addUserStrings offers the text the file's own code quotes, which is kept as a
// run of lengths and bytes.
func (f *dotnetFile) addUserStrings(fields map[string]modValue) {
	stream, found := f.namedStream("#US")
	if !found || stream.size == 0 || !f.pe.holds(stream.at, stream.size) {
		return
	}
	// The run opens with a single nothing, and anything else is not this.
	if f.pe.data[stream.at] != 0 {
		return
	}

	var items []modValue
	for at, end := stream.at+1, stream.at+stream.size; at < end; {
		length, taken, ok := f.readBlobLength(at)
		if !ok {
			break
		}
		at += taken
		// Text of nothing at all is passed over, which is what the padding at
		// the end of the run comes to.
		if length > 0 && f.pe.holds(at, length) {
			items = append(items, valueOf(stringValue(string(f.pe.data[at:at+length]))))
			at += length
		}
	}
	fields["user_strings"] = listOf(items)
	fields["number_of_user_strings"] = valueOf(intValue(int64(len(items))))
}

// readBlobLength reads how long a run of bytes is, which is written in one, two
// or four bytes according to how the first of them opens. The stated length
// counts a closing byte that is not part of what was written, so it is one
// less.
func (f *dotnetFile) readBlobLength(at int) (length, taken int, ok bool) {
	if !f.pe.holds(at, 1) {
		return 0, 0, false
	}
	switch first := f.pe.data[at]; {
	case first&0x80 == 0:
		length, taken = int(first), 1
	case first&0xC0 == 0x80:
		if !f.pe.holds(at, 2) {
			return 0, 0, false
		}
		length, taken = int(first&0x3F)<<8|int(f.pe.data[at+1]), 2
	case at+4 < len(f.pe.data) && first&0xE0 == 0xC0:
		length = int(first&0x1F)<<24 | int(f.pe.data[at+1])<<16 |
			int(f.pe.data[at+2])<<8 | int(f.pe.data[at+3])
		taken = 4
	default:
		return 0, 0, false
	}
	if length > 0 {
		length--
	}
	return length, taken, true
}

// blobAt reads the run of bytes kept at a place in the stream of them.
func (f *dotnetFile) blobAt(stream dotnetStream, index int) ([]byte, bool) {
	at := stream.at + index
	length, taken, ok := f.readBlobLength(at)
	if !ok {
		return nil, false
	}
	at += taken
	if !f.pe.holds(at, length) {
		return nil, false
	}
	return f.pe.data[at : at+length], true
}

// nameAt reads the name kept at a place in the stream of them, which runs to
// the next nothing. One running further than any name does is passed over.
func (f *dotnetFile) nameAt(stream dotnetStream, index int) (string, bool) {
	at := stream.at + index
	if at < 0 || at >= len(f.pe.data) {
		return "", false
	}
	end := indexByte(f.pe.data[at:], 0)
	if end < 0 || end > dotnetNameMax {
		return "", false
	}
	return string(f.pe.data[at : at+end]), true
}

// indexByte says where a byte first appears in a stretch, or below nought when
// it does not appear at all.
func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}
