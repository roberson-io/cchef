package ops

// The layout of a Windows portable executable, in bytes.
const (
	peHeaderPointer   = 0x3c
	peSectionCount    = 6
	peOptionalMagic   = 16
	peDataDirectory32 = 96
	peDataDirectory64 = 112
	peCertificateHdr  = 32
	peOptionalRest    = 88
	peSectionWidth    = 0x28
	peRawDataOffset   = 16
	pe32PlusMagic     = 0x20b
)

// carveMZPE ends the file at the end of its last section, or at the end of the
// attribute certificate table when there is one.
func carveMZPE(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// The MS-DOS stub points at the PE header.
	s.moveTo(peHeaderPointer)
	s.moveTo(s.readIntLE(4))

	// The file header gives the number of sections.
	s.moveForwardsBy(peSectionCount)
	sections := s.readIntLE(2)

	// The optional header says whether addresses are 32- or 64-bit, which sets
	// how far into it the data directory sits.
	s.moveForwardsBy(peOptionalMagic)
	directory := peDataDirectory32
	if s.readIntLE(2) == pe32PlusMagic {
		directory = peDataDirectory64
	}
	s.moveForwardsBy(directory - 2)

	// A portable executable may carry arbitrary appended data that no header
	// accounts for. The certificate table lives in it, so where one is declared
	// its end is the end of the file.
	s.moveForwardsBy(peCertificateHdr)
	certAddress := s.readIntLE(4)
	certSize := s.readIntLE(4)
	if certAddress > 0 {
		s.moveTo(certAddress + certSize)
		return s.carve(0, s.pos)
	}

	// Otherwise the file ends with its last section.
	s.moveForwardsBy(peOptionalRest)
	s.moveForwardsBy((sections - 1) * peSectionWidth)
	s.moveForwardsBy(peRawDataOffset)
	rawSize := s.readIntLE(4)
	rawAddress := s.readIntLE(4)

	s.moveTo(rawAddress + rawSize)
	return s.carve(0, s.pos)
}

// carveELF ends the file at the end of the section header table.
func carveELF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Over the magic to the class and byte order.
	s.moveForwardsBy(4)
	is32 := s.readInt(1) == 1
	little := s.readInt(1) == 1

	// Over the rest of the identification and the type, machine, version and
	// entry-point fields to the section header table's offset.
	s.moveForwardsBy(26)
	if !is32 {
		s.moveForwardsBy(8)
	}
	var sectionsAt int
	if is32 {
		sectionsAt = s.readIntOrder(4, little)
	} else {
		sectionsAt = s.readIntOrder(8, little)
	}

	// Over the flags, header size and program header details to the section
	// header size and count.
	s.moveForwardsBy(10)
	entrySize := s.readIntOrder(2, little)
	entries := s.readIntOrder(2, little)

	s.moveTo(sectionsAt)
	s.moveForwardsBy(entrySize * entries)
	return s.carve(0, s.pos)
}

// The Mach-O magic numbers, which say both the word size and the byte order.
const (
	machoMagic64   = 0xfeedfacf
	machoCigam64   = 0xcffaedfe
	machoCigam     = 0xcefaedfe
	machoSegment   = 0x1
	machoSegment64 = 0x19
)

// carveMACHO ends the file at the end of its last segment.
func carveMACHO(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	magic := s.readInt(4)
	is64 := magic == machoMagic64 || magic == machoCigam64
	little := magic == machoCigam || magic == machoCigam64

	// The header gives the number of load commands; the commands themselves
	// start after it.
	s.moveTo(16)
	commands := s.readIntOrder(4, little)

	commandsAt := 28
	if is64 {
		commandsAt += 4
	}

	s.moveTo(machoSegmentEnd(s, commandsAt, commands, little))
	return s.carve(0, s.pos)
}

// machoSegmentEnd walks the load commands and returns the total size of the
// segments among them, which is where the file ends.
func machoSegmentEnd(s *byteStream, at, commands int, little bool) int {
	total := 0
	for range commands {
		s.moveTo(at)
		switch s.readIntOrder(4, little) {
		case machoSegment64:
			s.moveTo(at + 48)
			total += s.readIntOrder(8, little)
		case machoSegment:
			s.moveTo(at + 36)
			total += s.readIntOrder(4, little)
		default:
			// Not a segment; only its size is needed to step over it.
		}
		s.moveTo(at + 4)
		at += s.readIntOrder(4, little)
	}
	return total
}

// carveSQLITE ends the file after the last of its fixed-size pages.
func carveSQLITE(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(16)
	pageSize := s.readInt(2)

	s.moveTo(28)
	pages := s.readInt(4)

	s.moveTo(pageSize * pages)
	return s.carve(0, s.pos)
}

// carveMacOSXKeychain reads the file's length out of its header.
func carveMacOSXKeychain(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(0x14)
	s.moveForwardsBy(s.readInt(4))
	return s.carve(0, s.pos)
}

// evtxChunk is the name every chunk opens with, and evtxChunkSize is the rest of
// a chunk after it: a chunk is 64 KiB including the name.
var evtxChunk = []byte("ElfChnk")

const evtxChunkSize = 0x10000 - len("ElfChnk")

// carveEVTX walks the chunks of a Windows event log to the padding that ends it.
//
// CyberChef reads the seven-byte chunk name with getBytes before comparing it
// (src/core/lib/FileSignatures.mjs), which moves the stream on whether or not it
// matched, so the carve keeps seven bytes of whatever followed the log. At the
// end of a buffer that is harmless, since the read clamps; for a log embedded in
// something larger it is seven bytes too many. This looks without moving.
func carveEVTX(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// The header says how far the first chunk is.
	s.moveTo(0x28)
	s.moveForwardsBy(s.readIntLE(4) - 0x2c)

	for s.hasMore() && s.lookingAt(evtxChunk) {
		s.moveForwardsBy(len(evtxChunk) + evtxChunkSize)
	}
	s.consumeWhile(0x00)
	return s.carve(0, s.pos)
}

// carveEVT ends the file after the end-of-file record its header points at.
func carveEVT(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(0x14)
	s.moveTo(s.readIntLE(4))

	size := s.readIntLE(4)
	s.moveForwardsBy(size - 4)
	return s.carve(0, s.pos)
}

// dmpPageSize is the page a Windows crash dump is written in, the first of which
// is the header.
const dmpPageSize = 0x1000

// carveDMP ends the file after the last of its pages.
func carveDMP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(0x70)
	s.moveTo((s.readIntLE(4) + 1) * dmpPageSize)
	return s.carve(0, s.pos)
}

// carvePF reads the file's length out of a Windows prefetch header.
func carvePF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(12)
	s.moveTo(s.readInt(4))
	return s.carve(0, s.pos)
}

// carvePFWin10 declines to guess where a Windows 10 prefetch file ends.
//
// From Windows 10 the file is compressed: a four-byte "MAM\x04" signature, the
// size the contents take once expanded, and then Xpress Huffman data running to
// the end of the file. No field says how long the compressed data is, so the end
// cannot be found without expanding it.
//
// CyberChef reads the four signature bytes as a big-endian length
// (src/core/lib/FileSignatures.mjs), which makes every such file appear to end
// at byte 1296125188 and the extraction fail out of bounds. Saying plainly that
// the length is not recorded is the same outcome with a reason attached.
func carvePFWin10([]byte, int) []byte {
	panic(carveFailure{msg: "A Windows 10 prefetch file records no compressed length, so its end cannot be found"})
}

// carveLNK reads the file's length out of a Windows shortcut header.
func carveLNK(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.moveTo(0x34)
	s.moveTo(s.readIntLE(4))
	return s.carve(0, s.pos)
}
