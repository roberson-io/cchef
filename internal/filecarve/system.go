package filecarve

import "github.com/roberson-io/cchef/internal/bytestream"

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
	s.MoveTo(peHeaderPointer)
	s.MoveTo(s.ReadIntLE(4))

	// The file header gives the number of sections.
	s.MoveForwardsBy(peSectionCount)
	sections := s.ReadIntLE(2)

	// The optional header says whether addresses are 32- or 64-bit, which sets
	// how far into it the data directory sits.
	s.MoveForwardsBy(peOptionalMagic)
	directory := peDataDirectory32
	if s.ReadIntLE(2) == pe32PlusMagic {
		directory = peDataDirectory64
	}
	s.MoveForwardsBy(directory - 2)

	// A portable executable may carry arbitrary appended data that no header
	// accounts for. The certificate table lives in it, so where one is declared
	// its end is the end of the file.
	s.MoveForwardsBy(peCertificateHdr)
	certAddress := s.ReadIntLE(4)
	certSize := s.ReadIntLE(4)
	if certAddress > 0 {
		s.MoveTo(certAddress + certSize)
		return s.Carve(0, s.Pos)
	}

	// Otherwise the file ends with its last section.
	s.MoveForwardsBy(peOptionalRest)
	s.MoveForwardsBy((sections - 1) * peSectionWidth)
	s.MoveForwardsBy(peRawDataOffset)
	rawSize := s.ReadIntLE(4)
	rawAddress := s.ReadIntLE(4)

	s.MoveTo(rawAddress + rawSize)
	return s.Carve(0, s.Pos)
}

// carveELF ends the file at the end of the section header table.
func carveELF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	// Over the magic to the class and byte order.
	s.MoveForwardsBy(4)
	is32 := s.ReadInt(1) == 1
	little := s.ReadInt(1) == 1

	// Over the rest of the identification and the type, machine, version and
	// entry-point fields to the section header table's offset.
	s.MoveForwardsBy(26)
	if !is32 {
		s.MoveForwardsBy(8)
	}
	var sectionsAt int
	if is32 {
		sectionsAt = s.ReadIntOrder(4, little)
	} else {
		sectionsAt = s.ReadIntOrder(8, little)
	}

	// Over the flags, header size and program header details to the section
	// header size and count.
	s.MoveForwardsBy(10)
	entrySize := s.ReadIntOrder(2, little)
	entries := s.ReadIntOrder(2, little)

	s.MoveTo(sectionsAt)
	s.MoveForwardsBy(entrySize * entries)
	return s.Carve(0, s.Pos)
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

	magic := s.ReadInt(4)
	is64 := magic == machoMagic64 || magic == machoCigam64
	little := magic == machoCigam || magic == machoCigam64

	// The header gives the number of load commands; the commands themselves
	// start after it.
	s.MoveTo(16)
	commands := s.ReadIntOrder(4, little)

	commandsAt := 28
	if is64 {
		commandsAt += 4
	}

	s.MoveTo(machoSegmentEnd(s, commandsAt, commands, little))
	return s.Carve(0, s.Pos)
}

// machoSegmentEnd walks the load commands and returns the total size of the
// segments among them, which is where the file ends.
func machoSegmentEnd(s *bytestream.Stream, at, commands int, little bool) int {
	total := 0
	for range commands {
		s.MoveTo(at)
		switch s.ReadIntOrder(4, little) {
		case machoSegment64:
			s.MoveTo(at + 48)
			total += s.ReadIntOrder(8, little)
		case machoSegment:
			s.MoveTo(at + 36)
			total += s.ReadIntOrder(4, little)
		default:
			// Not a segment; only its size is needed to step over it.
		}
		s.MoveTo(at + 4)
		at += s.ReadIntOrder(4, little)
	}
	return total
}

// carveSQLITE ends the file after the last of its fixed-size pages.
func carveSQLITE(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.MoveTo(16)
	pageSize := s.ReadInt(2)

	s.MoveTo(28)
	pages := s.ReadInt(4)

	s.MoveTo(pageSize * pages)
	return s.Carve(0, s.Pos)
}

// carveMacOSXKeychain reads the file's length out of its header.
func carveMacOSXKeychain(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.MoveTo(0x14)
	s.MoveForwardsBy(s.ReadInt(4))
	return s.Carve(0, s.Pos)
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
	s.MoveTo(0x28)
	s.MoveForwardsBy(s.ReadIntLE(4) - 0x2c)

	for s.HasMore() && s.LookingAt(evtxChunk) {
		s.MoveForwardsBy(len(evtxChunk) + evtxChunkSize)
	}
	s.ConsumeWhile(0x00)
	return s.Carve(0, s.Pos)
}

// carveEVT ends the file after the end-of-file record its header points at.
func carveEVT(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.MoveTo(0x14)
	s.MoveTo(s.ReadIntLE(4))

	size := s.ReadIntLE(4)
	s.MoveForwardsBy(size - 4)
	return s.Carve(0, s.Pos)
}

// dmpPageSize is the page a Windows crash dump is written in, the first of which
// is the header.
const dmpPageSize = 0x1000

// carveDMP ends the file after the last of its pages.
func carveDMP(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.MoveTo(0x70)
	s.MoveTo((s.ReadIntLE(4) + 1) * dmpPageSize)
	return s.Carve(0, s.Pos)
}

// carvePF reads the file's length out of a Windows prefetch header.
func carvePF(data []byte, offset int) []byte {
	s := carveStream(data, offset)

	s.MoveTo(12)
	s.MoveTo(s.ReadInt(4))
	return s.Carve(0, s.Pos)
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

	s.MoveTo(0x34)
	s.MoveTo(s.ReadIntLE(4))
	return s.Carve(0, s.Pos)
}
