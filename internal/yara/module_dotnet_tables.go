package yara

import (
	"encoding/binary"
	"strings"
)

// The tables the runtime works from, which sit in one stream of the metadata.
//
// The stream opens with a header saying which tables are present and how wide
// the places into the other streams are, then how many rows each present table
// has, then the tables laid one after another. A table's rows are all the same
// width, but that width depends on how many rows other tables hold: a column
// pointing into a table of more than 65535 rows takes four bytes rather than
// two. So the counts are read first and the tables walked afterwards.

const (
	// tildeHeaderSize is what opens the stream: something set aside, a version
	// of the tables, how wide the places into the other streams are, another
	// byte set aside, then which tables are present and which are sorted.
	tildeHeaderSize = 24
	// tildeHeapSizesAt is the byte saying which places are written wide, and
	// tildeValidAt where the mark of which tables are present begins.
	tildeHeapSizesAt = 6
	tildeValidAt     = 8
	// The bits of that byte, one for each stream a place may point into.
	heapWideStrings = 0x01
	heapWideGUID    = 0x02
	heapWideBlob    = 0x04
	// tableCount is how many tables the mark has room for.
	tableCount = 64
	// maxTableRows is more rows than any real file holds. A table claiming more
	// ends the walk, since some files state a wild number.
	maxTableRows = 10000
	// wideRows is how many rows a table may hold before a place into it has to
	// be written in four bytes rather than two.
	wideRows = 0xFFFF
	// elementTypeString marks a fixed value written as text, which is the only
	// kind offered.
	elementTypeString = 0x0E

	// What the number a file is known by is reached through. An attribute is
	// the one wanted when it hangs off the assembly itself, names a member of
	// another file, and that member belongs to a type called by this name.
	tagAssembly   = 0x0E
	tagMemberRef  = 0x03
	tagTypeRef    = 0x01
	typelibName   = "GuidAttribute"
	typelibOpener = 0x0001
)

// The tables, numbered by their place in the mark saying which are present.
const (
	tableModule                 = 0x00
	tableTypeRef                = 0x01
	tableTypeDef                = 0x02
	tableFieldPtr               = 0x03
	tableField                  = 0x04
	tableMethodDefPtr           = 0x05
	tableMethodDef              = 0x06
	tableParamPtr               = 0x07
	tableParam                  = 0x08
	tableInterfaceImpl          = 0x09
	tableMemberRef              = 0x0A
	tableConstant               = 0x0B
	tableCustomAttribute        = 0x0C
	tableFieldMarshal           = 0x0D
	tableDeclSecurity           = 0x0E
	tableClassLayout            = 0x0F
	tableFieldLayout            = 0x10
	tableStandAloneSig          = 0x11
	tableEventMap               = 0x12
	tableEventPtr               = 0x13
	tableEvent                  = 0x14
	tablePropertyMap            = 0x15
	tablePropertyPtr            = 0x16
	tableProperty               = 0x17
	tableMethodSemantics        = 0x18
	tableMethodImpl             = 0x19
	tableModuleRef              = 0x1A
	tableTypeSpec               = 0x1B
	tableImplMap                = 0x1C
	tableFieldRVA               = 0x1D
	tableENCLog                 = 0x1E
	tableENCMap                 = 0x1F
	tableAssembly               = 0x20
	tableAssemblyProcessor      = 0x21
	tableAssemblyOS             = 0x22
	tableAssemblyRef            = 0x23
	tableAssemblyRefProcessor   = 0x24
	tableAssemblyRefOS          = 0x25
	tableFile                   = 0x26
	tableExportedType           = 0x27
	tableManifestResource       = 0x28
	tableNestedClass            = 0x29
	tableGenericParam           = 0x2A
	tableMethodSpec             = 0x2B
	tableGenericParamConstraint = 0x2C
)

// dotnetTables is what reading the header tells: how many rows each table
// holds, and how wide a place into each stream or table is.
type dotnetTables struct {
	rows [tableCount]uint32
	// str, guid and blob are how wide a place into each of those streams is.
	str, guid, blob int
	// table is how wide a place into each table is.
	table [tableCount]int
}

// coded gives how wide a place is that may point into any of several tables.
// Some of its low bits say which table is meant, leaving fewer for the row, so
// it goes wide sooner. The count of those bits is `tag`.
func (t *dotnetTables) coded(tag int, tables ...int) int {
	var most uint32
	for _, table := range tables {
		most = max(most, t.rows[table])
	}
	if most > wideRows>>tag {
		return 4
	}
	return 2
}

// The tables a place carrying its own tag may point into. They are named here
// because the widths and the reading both need them.
var (
	codedTypeRefScope    = []int{tableModule, tableModuleRef, tableAssemblyRef, tableTypeRef}
	codedTypeDefOrRef    = []int{tableTypeDef, tableTypeRef, tableTypeSpec}
	codedMemberRefScope  = []int{tableMethodDef, tableModuleRef, tableTypeRef, tableTypeSpec}
	codedConstParent     = []int{tableParam, tableField, tableProperty}
	codedHasSemantics    = []int{tableEvent, tableProperty}
	codedMethodDefOrRef  = []int{tableMethodDef, tableMemberRef}
	codedImplementation  = []int{tableFile, tableAssemblyRef}
	codedExportedScope   = []int{tableFile, tableAssemblyRef, tableExportedType}
	codedMemberForwarded = []int{tableField, tableMethodDef}
	codedTypeOrMethodDef = []int{tableTypeDef, tableMethodDef}
	codedHasDeclSecurity = []int{tableTypeDef, tableMethodDef, tableAssembly}
	codedHasFieldMarshal = []int{tableField, tableParam}
	// Anything at all may carry an attribute, so the place naming what does
	// reaches into more tables than any other.
	codedHasCustomAttribute = []int{
		tableMethodDef, tableField, tableTypeRef, tableTypeDef, tableParam,
		tableInterfaceImpl, tableMemberRef, tableModule, tableProperty,
		tableEvent, tableStandAloneSig, tableModuleRef, tableTypeSpec,
		tableAssembly, tableAssemblyRef, tableFile, tableExportedType,
		tableManifestResource, tableGenericParam, tableGenericParamConstraint,
		tableMethodSpec,
	}
)

// tableWidths gives how wide one row of each table is. A table with no entry
// here is one the walk cannot step over, so nothing past it is read.
var tableWidths = map[int]func(t *dotnetTables) int{
	tableModule:       func(t *dotnetTables) int { return 2 + t.str + t.guid*3 },
	tableTypeRef:      func(t *dotnetTables) int { return t.coded(2, codedTypeRefScope...) + t.str*2 },
	tableFieldPtr:     func(t *dotnetTables) int { return t.table[tableField] },
	tableField:        func(t *dotnetTables) int { return 2 + t.str + t.blob },
	tableMethodDefPtr: func(t *dotnetTables) int { return t.table[tableMethodDef] },
	tableParam:        func(t *dotnetTables) int { return 2 + 2 + t.str },
	tableMemberRef: func(t *dotnetTables) int {
		return t.coded(3, codedMemberRefScope...) + t.str + t.blob
	},
	tableFieldMarshal: func(t *dotnetTables) int {
		return t.coded(1, codedHasFieldMarshal...) + t.blob
	},
	tableDeclSecurity: func(t *dotnetTables) int {
		return 2 + t.coded(2, codedHasDeclSecurity...) + t.blob
	},
	tableClassLayout:   func(t *dotnetTables) int { return 2 + 4 + t.table[tableTypeDef] },
	tableFieldLayout:   func(t *dotnetTables) int { return 4 + t.table[tableField] },
	tableStandAloneSig: func(t *dotnetTables) int { return t.blob },
	tableEventMap: func(t *dotnetTables) int {
		return t.table[tableTypeDef] + t.table[tableEvent]
	},
	tableEventPtr: func(t *dotnetTables) int { return t.table[tableEvent] },
	tableEvent: func(t *dotnetTables) int {
		return 2 + t.str + t.coded(2, codedTypeDefOrRef...)
	},
	tablePropertyMap: func(t *dotnetTables) int {
		return t.table[tableTypeDef] + t.table[tableProperty]
	},
	tablePropertyPtr: func(t *dotnetTables) int { return t.table[tableProperty] },
	tableProperty:    func(t *dotnetTables) int { return 2 + t.str + t.blob },
	tableMethodSemantics: func(t *dotnetTables) int {
		return 2 + t.table[tableMethodDef] + t.coded(1, codedHasSemantics...)
	},
	tableMethodImpl: func(t *dotnetTables) int {
		return t.table[tableTypeDef] + t.coded(1, codedMethodDefOrRef...)*2
	},
	tableModuleRef: func(t *dotnetTables) int { return t.str },
	tableTypeSpec:  func(t *dotnetTables) int { return t.blob },
	tableENCLog:    func(t *dotnetTables) int { return 4 + 4 },
	tableENCMap:    func(t *dotnetTables) int { return 4 },
	tableAssembly: func(t *dotnetTables) int {
		return 4 + 2 + 2 + 2 + 2 + 4 + t.blob + t.str*2
	},
	tableAssemblyProcessor: func(t *dotnetTables) int { return 4 },
	tableAssemblyOS:        func(t *dotnetTables) int { return 4 + 4 + 4 },
	tableAssemblyRef: func(t *dotnetTables) int {
		return 2 + 2 + 2 + 2 + 4 + t.blob*2 + t.str*2
	},
	tableAssemblyRefProcessor: func(t *dotnetTables) int {
		return 4 + t.table[tableAssemblyRefProcessor]
	},
	tableAssemblyRefOS: func(t *dotnetTables) int {
		return 4 + 4 + 4 + t.table[tableAssemblyRef]
	},
	tableFile: func(t *dotnetTables) int { return 4 + t.str + t.blob },
	tableExportedType: func(t *dotnetTables) int {
		return 4 + 4 + t.str*2 + t.coded(2, codedExportedScope...)
	},
	tableNestedClass: func(t *dotnetTables) int { return t.table[tableTypeDef] * 2 },
	tableGenericParam: func(t *dotnetTables) int {
		return 2 + 2 + t.coded(1, codedTypeOrMethodDef...) + t.str
	},
	tableMethodSpec: func(t *dotnetTables) int {
		return t.coded(1, codedMethodDefOrRef...) + t.blob
	},
	tableGenericParamConstraint: func(t *dotnetTables) int {
		return t.table[tableGenericParam] + t.coded(2, codedTypeDefOrRef...)
	},
	// The rest are read rather than only stepped over, so their widths sit
	// beside the reading.
	tableTypeDef: func(t *dotnetTables) int {
		return 4 + t.str*2 + t.coded(2, codedTypeDefOrRef...) +
			t.table[tableField] + t.table[tableMethodDef]
	},
	tableMethodDef: func(t *dotnetTables) int {
		return 4 + 2 + 2 + t.str + t.blob + t.table[tableParam]
	},
	tableInterfaceImpl: func(t *dotnetTables) int {
		return t.table[tableTypeDef] + t.coded(2, codedTypeDefOrRef...)
	},
	tableConstant: func(t *dotnetTables) int {
		return 1 + 1 + t.coded(2, codedConstParent...) + t.blob
	},
	tableCustomAttribute: func(t *dotnetTables) int {
		return t.coded(5, codedHasCustomAttribute...) +
			t.coded(3, codedMethodDefOrRef...) + t.blob
	},
	tableImplMap: func(t *dotnetTables) int {
		return 2 + t.coded(1, codedMemberForwarded...) + t.str +
			t.table[tableModuleRef]
	},
	tableFieldRVA: func(t *dotnetTables) int { return 4 + t.table[tableField] },
	tableManifestResource: func(t *dotnetTables) int {
		return 4 + 4 + t.str + t.coded(2, codedImplementation...)
	},
}

// tableReaders gives, for each table something is read out of, how to read it.
var tableReaders = map[int]func(w *tableWalk, rows int){
	tableModule:           (*tableWalk).readModule,
	tableTypeRef:          (*tableWalk).noteTypeRef,
	tableMemberRef:        (*tableWalk).noteMemberRef,
	tableConstant:         (*tableWalk).readConstants,
	tableCustomAttribute:  (*tableWalk).readTypelib,
	tableModuleRef:        (*tableWalk).readModuleRefs,
	tableFieldRVA:         (*tableWalk).readFieldOffsets,
	tableAssembly:         (*tableWalk).readAssembly,
	tableAssemblyRef:      (*tableWalk).readAssemblyRefs,
	tableManifestResource: (*tableWalk).readResources,
}

// readTableHeader learns which tables are present, how many rows each holds,
// and how wide every place is, giving back where the tables themselves begin.
func (f *dotnetFile) readTableHeader(stream dotnetStream) (t *dotnetTables, valid uint64, at int, ok bool) {
	if !f.pe.holds(stream.at, tildeHeaderSize) {
		return nil, 0, 0, false
	}
	t = &dotnetTables{str: 2, guid: 2, blob: 2}
	for i := range t.table {
		t.table[i] = 2
	}
	heaps := f.pe.data[stream.at+tildeHeapSizesAt]
	if heaps&heapWideStrings != 0 {
		t.str = 4
	}
	if heaps&heapWideGUID != 0 {
		t.guid = 4
	}
	if heaps&heapWideBlob != 0 {
		t.blob = 4
	}

	valid = binary.LittleEndian.Uint64(f.pe.data[stream.at+tildeValidAt:])
	at = stream.at + tildeHeaderSize
	for table := range tableCount {
		if valid>>table&1 == 0 {
			continue
		}
		if !f.pe.holds(at, 4) {
			return nil, 0, 0, false
		}
		t.rows[table] = binary.LittleEndian.Uint32(f.pe.data[at:])
		if t.rows[table] > wideRows {
			t.table[table] = 4
		}
		at += 4
	}
	return t, valid, at, true
}

// addTables reads what a rule can ask about out of the tables. The stream of
// names and the stream of runs of bytes are both needed, since the tables point
// into them; without either, none of the tables is read.
func (f *dotnetFile) addTables(fields map[string]modValue) {
	tilde, hasTilde := f.namedStream("#~", "#-")
	names, hasNames := f.namedStream("#Strings")
	blobs, hasBlobs := f.namedStream("#Blob")
	if !hasTilde || !hasNames || !hasBlobs {
		return
	}
	t, valid, at, ok := f.readTableHeader(tilde)
	if !ok {
		return
	}
	w := &tableWalk{
		f: f, t: t, valid: valid, names: names, blobs: blobs,
		fields: fields, at: at,
	}
	w.walk()
}

// tableWalk is one pass over the tables, reading what is wanted out of them and
// stepping over the rest.
type tableWalk struct {
	f      *dotnetFile
	t      *dotnetTables
	valid  uint64
	names  dotnetStream
	blobs  dotnetStream
	fields map[string]modValue
	// at is where the table being read begins.
	at int
	// The two tables the number a file is known by is reached through. They are
	// noted as the walk passes them, which is always before the attributes.
	typeRefAt, typeRefWidth     int
	memberRefAt, memberRefWidth int
	hasTypeRef, hasMemberRef    bool
}

// walk goes over every table the file holds, in order.
func (w *tableWalk) walk() {
	for table := range tableCount {
		if w.valid>>table&1 == 0 {
			continue
		}
		rows := int(w.t.rows[table])
		// A table claiming more rows than any real file holds ends the walk
		// rather than being trusted.
		if rows > maxTableRows {
			return
		}
		width, known := tableWidths[table]
		if !known {
			// A table the walk does not know cannot be stepped over, since how
			// wide its rows are is unknown, so nothing past it is read.
			return
		}
		if read, wanted := tableReaders[table]; wanted {
			read(w, rows)
		}
		w.at += width(w.t) * rows
	}
}

// number reads a place or a count of the given width out of the file. Every
// place read this way lies within a row that has already been found to be
// wholly there, so there is nothing left to check.
func (w *tableWalk) number(at, width int) int {
	if width == 4 {
		return int(binary.LittleEndian.Uint32(w.f.pe.data[at:]))
	}
	return int(binary.LittleEndian.Uint16(w.f.pe.data[at:]))
}

// name reads the name that a place in a row points at.
func (w *tableWalk) name(at int) (string, bool) {
	return w.f.nameAt(w.names, w.number(at, w.t.str))
}

// blob reads the run of bytes that a place in a row points at.
func (w *tableWalk) blob(at int) ([]byte, bool) {
	return w.f.blobAt(w.blobs, w.number(at, w.t.blob))
}

// rowsOf walks the rows of the table being read, stopping where the file does.
func (w *tableWalk) rowsOf(table, rows int) func(func(int) bool) {
	width := tableWidths[table](w.t)
	return func(yield func(int) bool) {
		for i := range rows {
			at := w.at + width*i
			if !w.f.pe.holds(at, width) || !yield(at) {
				return
			}
		}
	}
}

// readModule reads the name the file gives itself.
func (w *tableWalk) readModule(rows int) {
	for at := range w.rowsOf(tableModule, rows) {
		if name, ok := w.name(at + 2); ok {
			w.fields["module_name"] = valueOf(stringValue(name))
		}
		return
	}
}

// noteTypeRef and noteMemberRef remember where those tables are, so that the
// number a file is known by can be reached through them later.
func (w *tableWalk) noteTypeRef(int) {
	w.typeRefAt, w.typeRefWidth = w.at, tableWidths[tableTypeRef](w.t)
	w.hasTypeRef = true
}

func (w *tableWalk) noteMemberRef(int) {
	w.memberRefAt, w.memberRefWidth = w.at, tableWidths[tableMemberRef](w.t)
	w.hasMemberRef = true
}

// readModuleRefs reads the other files the file leans on by name.
func (w *tableWalk) readModuleRefs(rows int) {
	var items []modValue
	for at := range w.rowsOf(tableModuleRef, rows) {
		if name, ok := w.name(at); ok {
			items = append(items, valueOf(stringValue(name)))
		}
	}
	w.fields["modulerefs"] = listOf(items)
	w.fields["number_of_modulerefs"] = valueOf(intValue(int64(len(items))))
}

// readFieldOffsets reads where in the file the fixed data sits. One whose
// address points nowhere within the file is passed over.
func (w *tableWalk) readFieldOffsets(rows int) {
	var items []modValue
	for at := range w.rowsOf(tableFieldRVA, rows) {
		address := w.number(at, 4)
		// #nosec G115 -- four bytes read out of the file, so never negative
		if place, found := w.f.pe.rvaToOffset(uint64(address)); found {
			items = append(items, valueOf(intValue(place)))
		}
	}
	w.fields["field_offsets"] = listOf(items)
	w.fields["number_of_field_offsets"] = valueOf(intValue(int64(len(items))))
}

// readConstants reads the fixed values the file carries, of which only those
// written as text are offered.
func (w *tableWalk) readConstants(rows int) {
	var items []modValue
	parent := w.t.coded(2, codedConstParent...)
	for at := range w.rowsOf(tableConstant, rows) {
		if w.f.pe.data[at] != elementTypeString {
			continue
		}
		if run, ok := w.blob(at + 2 + parent); ok {
			items = append(items, valueOf(stringValue(string(run))))
		}
	}
	w.fields["constants"] = listOf(items)
	w.fields["number_of_constants"] = valueOf(intValue(int64(len(items))))
}

// dotnetVersion reads the four numbers a version is made of.
func (w *tableWalk) dotnetVersion(at int) modValue {
	part := func(i int) modValue {
		n := w.number(at+i*2, 2)
		return valueOf(intValue(int64(n)))
	}
	return structOf(map[string]modValue{
		"major": part(0), "minor": part(1),
		"build_number": part(2), "revision_number": part(3),
	})
}

// readAssembly reads what the file says of itself as an assembly. Only the
// first row is read, since a file is one assembly.
func (w *tableWalk) readAssembly(rows int) {
	for at := range w.rowsOf(tableAssembly, rows) {
		// A hashing method, the four numbers of the version, some flags, the
		// key it is known by, its name, and where it is meant for.
		const beforeVersion = 4
		nameAt := at + 4 + 2 + 2 + 2 + 2 + 4 + w.t.blob
		member := map[string]modValue{"version": w.dotnetVersion(at + beforeVersion)}
		if name, ok := w.name(nameAt); ok {
			member["name"] = valueOf(stringValue(name))
		}
		// Where it is meant for is sometimes written as nothing at all, which
		// the specification does not allow but which happens, and is passed
		// over rather than given as empty.
		if culture, ok := w.name(nameAt + w.t.str); ok && culture != "" {
			member["culture"] = valueOf(stringValue(culture))
		}
		w.fields["assembly"] = structOf(member)
		return
	}
}

// readAssemblyRefs reads the assemblies the file leans on.
func (w *tableWalk) readAssemblyRefs(rows int) {
	var items []modValue
	counted := 0
	for at := range w.rowsOf(tableAssemblyRef, rows) {
		counted++
		nameAt := at + 2 + 2 + 2 + 2 + 4 + w.t.blob
		member := map[string]modValue{"version": w.dotnetVersion(at)}
		// The key it is known by comes before the name and is not always
		// there; a key of nothing at all is passed over.
		key, ok := w.blob(at + 2 + 2 + 2 + 2 + 4)
		if !ok {
			items = append(items, structOf(member))
			continue
		}
		if len(key) > 0 {
			member["public_key_or_token"] = valueOf(stringValue(string(key)))
		}
		if name, named := w.name(nameAt); named {
			member["name"] = valueOf(stringValue(name))
		}
		items = append(items, structOf(member))
	}
	w.fields["assembly_refs"] = listOf(items)
	w.fields["number_of_assembly_refs"] = valueOf(intValue(int64(counted)))
}

// readResources reads what the file keeps for itself. Only what is held in this
// file is offered; a resource kept in another is passed over.
func (w *tableWalk) readResources(rows int) {
	base, found := w.f.resourceBase()
	var items []modValue
	for at := range w.rowsOf(tableManifestResource, rows) {
		if !found {
			break
		}
		place := w.number(at, 4)
		heldElsewhere := w.number(at+4+4+w.t.str, w.t.coded(2, codedImplementation...))
		if heldElsewhere != 0 {
			continue
		}
		// The resource opens by saying how long it is, and the bytes offered
		// are what follows that.
		begins := base + place
		if !w.f.pe.holds(begins, 4) {
			continue
		}
		length := w.number(begins, 4)
		if !w.f.pe.holds(begins, length) {
			continue
		}
		member := map[string]modValue{
			"offset": valueOf(intValue(int64(begins + 4))),
			"length": valueOf(intValue(int64(length))),
		}
		if name, ok := w.name(at + 4 + 4); ok {
			member["name"] = valueOf(stringValue(name))
		}
		items = append(items, structOf(member))
	}
	w.fields["resources"] = listOf(items)
	w.fields["number_of_resources"] = valueOf(intValue(int64(len(items))))
}

// readTypelib reads the number a file is known by to the older way of naming
// components, which is carried as an attribute hanging off the assembly.
//
// Getting to it means following a chain: the attribute says what it hangs off
// and which member names it; the member says which type it belongs to; and the
// type gives that type's name. Only when the name is the one this looks for are
// the attribute's own bytes worth reading.
func (w *tableWalk) readTypelib(rows int) {
	if !w.hasTypeRef || !w.hasMemberRef {
		return
	}
	parent := w.t.coded(5, codedHasCustomAttribute...)
	kind := w.t.coded(3, codedMethodDefOrRef...)
	for at := range w.rowsOf(tableCustomAttribute, rows) {
		hangsOff := w.number(at, parent)
		if hangsOff&0x1F != tagAssembly {
			continue
		}
		named := w.number(at+parent, kind)
		if named&0x07 != tagMemberRef {
			continue
		}
		if !w.attributeIsTheOne(named >> 3) {
			continue
		}
		if text, ok := w.typelibText(at + parent + kind); ok {
			w.fields["typelib"] = valueOf(stringValue(text))
		}
	}
}

// attributeIsTheOne follows a member back to the type it belongs to and says
// whether that type carries the name being looked for.
func (w *tableWalk) attributeIsTheOne(member int) bool {
	if member > 0 {
		member--
	}
	row := w.memberRefAt + w.memberRefWidth*member
	if !w.f.pe.holds(row, w.memberRefWidth) {
		return false
	}
	belongsTo := w.number(row, w.t.coded(3, codedMemberRefScope...))
	if belongsTo&0x07 != tagTypeRef {
		return false
	}
	kind := belongsTo >> 3
	if kind > 0 {
		kind--
	}
	typeRow := w.typeRefAt + w.typeRefWidth*kind
	if !w.f.pe.holds(typeRow, w.typeRefWidth) {
		return false
	}
	// The name follows what says where the type was found.
	name, ok := w.name(typeRow + w.t.coded(2, codedTypeRefScope...))
	return ok && strings.HasPrefix(name, typelibName)
}

// typelibText reads the bytes of that attribute, which open with a fixed pair
// and then say how long the text is. A file writing nothing there is read as
// being known by nothing.
func (w *tableWalk) typelibText(at int) (string, bool) {
	index := w.number(at, w.t.blob)
	if index == 0 {
		return "", false
	}
	run, ok := w.f.blobAt(w.blobs, index)
	// Two bytes of opener and one of length, at the very least.
	const leastRun = 3
	if !ok || len(run) < leastRun ||
		binary.LittleEndian.Uint16(run) != typelibOpener {
		return "", false
	}
	length := int(run[2])
	if length > len(run)-leastRun {
		return "", false
	}
	text := run[leastRun : leastRun+length]
	if length == 0 || text[0] == 0xFF || text[0] == 0x00 {
		return "", true
	}
	return string(text), true
}

// resourceBase is where in the file what it keeps for itself begins, which the
// runtime's own header names.
func (f *dotnetFile) resourceBase() (int, bool) {
	// The whole of the runtime's header is there, or the file would not have
	// been read as one built for it.
	address, _ := f.pe.u32(f.cli + cliResourcesAt)
	at, found := f.pe.rvaToOffset(uint64(address))
	return int(at), found
}
