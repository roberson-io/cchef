package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// The four sample binaries and their expected reports are CyberChef's own
// fixtures (tests/samples/Executables.mjs and tests/operations/tests/ELFInfo.mjs).

// elf32LE is a hand-built ELF: 32-bit, little endian.
const elf32LE = "" +
	"7f454c460101010000000000000000000200030001000000502106083400000054000000000000003400200001002800" +
	"030000000600000034000000348004083480040800010000000100000500000004000000000000000300000000000000" +
	"00000000cc0000001c0000000000000000000000000000000000000009000000020000000000000000000000e6000000" +
	"100000000000000000000000000000001000000011000000030000000000000000000000f50000000400000000000000" +
	"0000000000000000000000002e73687374726162002e73796d746162002e737472746162000000000000000000000000" +
	"000000000074657374"

const elf32LEWant = "" +
	"============================== ELF Header ==============================\n" +
	"Magic:                        \x7fELF\n" +
	"Format:                       32-bit\n" +
	"Endianness:                   Little\n" +
	"Version:                      1\n" +
	"ABI:                          System V\n" +
	"ABI Version:                  0\n" +
	"Type:                         Executable File\n" +
	"Instruction Set Architecture: x86\n" +
	"ELF Version:                  1\n" +
	"Entry Point:                  0x8062150\n" +
	"Entry PHOFF:                  0x34\n" +
	"Entry SHOFF:                  0x54\n" +
	"Flags:                        00000000\n" +
	"ELF Header Size:              52 bytes\n" +
	"Program Header Size:          32 bytes\n" +
	"Program Header Entries:       1\n" +
	"Section Header Size:          40 bytes\n" +
	"Section Header Entries:       3\n" +
	"Section Header Names:         0\n" +
	"\n" +
	"============================== Program Header ==============================\n" +
	"Program Header Type:          Program Header Table\n" +
	"Offset Of Segment:            52\n" +
	"Virtual Address of Segment:   134512692\n" +
	"Physical Address of Segment:  134512692\n" +
	"Size of Segment:              256 bytes\n" +
	"Size of Segment in Memory:    256 bytes\n" +
	"Flags:                        Execute,Read\n" +
	"\n" +
	"============================== Section Header ==============================\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .shstrab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        204\n" +
	"Section Size:                 28\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         Symbol Table\n" +
	"Section Name:                 .symtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        230\n" +
	"Section Size:                 16\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .strtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        245\n" +
	"Section Size:                 4\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"============================== Symbol Table ==============================\n" +
	"Symbol Name:                  test"

// elf32BE is a hand-built ELF: 32-bit, big endian.
const elf32BE = "" +
	"7f454c460102010000000000000000000002000300000001080621500000003400000054000000000034002000010028" +
	"000300000000000600000034080480340804803400000100000001000000000500000004000000000000000300000000" +
	"00000000000000cc0000001c0000000000000000000000000000000000000009000000020000000000000000000000e6" +
	"000000100000000000000000000000000000001000000011000000030000000000000000000000f50000000400000000" +
	"0000000000000000000000002e73687374726162002e73796d746162002e737472746162000000000000000000000000" +
	"000000000074657374"

const elf32BEWant = "" +
	"============================== ELF Header ==============================\n" +
	"Magic:                        \x7fELF\n" +
	"Format:                       32-bit\n" +
	"Endianness:                   Big\n" +
	"Version:                      1\n" +
	"ABI:                          System V\n" +
	"ABI Version:                  0\n" +
	"Type:                         Executable File\n" +
	"Instruction Set Architecture: x86\n" +
	"ELF Version:                  1\n" +
	"Entry Point:                  0x8062150\n" +
	"Entry PHOFF:                  0x34\n" +
	"Entry SHOFF:                  0x54\n" +
	"Flags:                        00000000\n" +
	"ELF Header Size:              52 bytes\n" +
	"Program Header Size:          32 bytes\n" +
	"Program Header Entries:       1\n" +
	"Section Header Size:          40 bytes\n" +
	"Section Header Entries:       3\n" +
	"Section Header Names:         0\n" +
	"\n" +
	"============================== Program Header ==============================\n" +
	"Program Header Type:          Program Header Table\n" +
	"Offset Of Segment:            52\n" +
	"Virtual Address of Segment:   134512692\n" +
	"Physical Address of Segment:  134512692\n" +
	"Size of Segment:              256 bytes\n" +
	"Size of Segment in Memory:    256 bytes\n" +
	"Flags:                        Execute,Read\n" +
	"\n" +
	"============================== Section Header ==============================\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .shstrab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        204\n" +
	"Section Size:                 28\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         Symbol Table\n" +
	"Section Name:                 .symtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        230\n" +
	"Section Size:                 16\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .strtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        245\n" +
	"Section Size:                 4\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"============================== Symbol Table ==============================\n" +
	"Symbol Name:                  test"

// elf64LE is a hand-built ELF: 64-bit, little endian.
const elf64LE = "" +
	"7f454c4602010100000000000000000002003e0001000000502106080000000040000000000000007800000000000000" +
	"000000004000380001004000030000000600000005000000340000000000000034800408000000003480040800000000" +
	"000100000000000000010000000000000400000000000000000000000300000000000000000000000000000000000000" +
	"38010000000000001c000000000000000000000000000000000000000000000000000000000000000900000002000000" +
	"000000000000000000000000000000005001000000000000100000000000000000000000000000000000000000000000" +
	"180000000000000011000000030000000000000000000000000000000000000069010000000000000400000000000000" +
	"0000000000000000000000000000000000000000000000002e73687374726162002e73796d746162002e737472746162" +
	"0000000000000000000000000000000000000000000000000074657374"

const elf64LEWant = "" +
	"============================== ELF Header ==============================\n" +
	"Magic:                        \x7fELF\n" +
	"Format:                       64-bit\n" +
	"Endianness:                   Little\n" +
	"Version:                      1\n" +
	"ABI:                          System V\n" +
	"ABI Version:                  0\n" +
	"Type:                         Executable File\n" +
	"Instruction Set Architecture: AMD x86-64\n" +
	"ELF Version:                  1\n" +
	"Entry Point:                  0x8062150\n" +
	"Entry PHOFF:                  0x40\n" +
	"Entry SHOFF:                  0x78\n" +
	"Flags:                        00000000\n" +
	"ELF Header Size:              64 bytes\n" +
	"Program Header Size:          56 bytes\n" +
	"Program Header Entries:       1\n" +
	"Section Header Size:          64 bytes\n" +
	"Section Header Entries:       3\n" +
	"Section Header Names:         0\n" +
	"\n" +
	"============================== Program Header ==============================\n" +
	"Program Header Type:          Program Header Table\n" +
	"Flags:                        Execute,Read\n" +
	"Offset Of Segment:            52\n" +
	"Virtual Address of Segment:   134512692\n" +
	"Physical Address of Segment:  134512692\n" +
	"Size of Segment:              256 bytes\n" +
	"Size of Segment in Memory:    256 bytes\n" +
	"\n" +
	"============================== Section Header ==============================\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .shstrab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        312\n" +
	"Section Size:                 28\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         Symbol Table\n" +
	"Section Name:                 .symtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        336\n" +
	"Section Size:                 16\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .strtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        361\n" +
	"Section Size:                 4\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"============================== Symbol Table ==============================\n" +
	"Symbol Name:                  test"

// elf64BE is a hand-built ELF: 64-bit, big endian.
const elf64BE = "" +
	"7f454c460202010000000000000000000002003e00000001000000000806215000000000000000400000000000000078" +
	"000000000040003800010040000300000000000600000005000000000000003400000000080480340000000008048034" +
	"000000000000010000000000000001000400000000000000000000000000000300000000000000000000000000000000" +
	"0000000000000138000000000000001c0000000000000000000000000000000000000000000000000000000900000002" +
	"000000000000000000000000000000000000000000000150000000000000001000000000000000000000000000000000" +
	"000000000000001800000011000000030000000000000000000000000000000000000000000001690000000000000004" +
	"0000000000000000000000000000000000000000000000002e73687374726162002e73796d746162002e737472746162" +
	"0000000000000000000000000000000000000000000000000074657374"

const elf64BEWant = "" +
	"============================== ELF Header ==============================\n" +
	"Magic:                        \x7fELF\n" +
	"Format:                       64-bit\n" +
	"Endianness:                   Big\n" +
	"Version:                      1\n" +
	"ABI:                          System V\n" +
	"ABI Version:                  0\n" +
	"Type:                         Executable File\n" +
	"Instruction Set Architecture: AMD x86-64\n" +
	"ELF Version:                  1\n" +
	"Entry Point:                  0x8062150\n" +
	"Entry PHOFF:                  0x40\n" +
	"Entry SHOFF:                  0x78\n" +
	"Flags:                        00000000\n" +
	"ELF Header Size:              64 bytes\n" +
	"Program Header Size:          56 bytes\n" +
	"Program Header Entries:       1\n" +
	"Section Header Size:          64 bytes\n" +
	"Section Header Entries:       3\n" +
	"Section Header Names:         0\n" +
	"\n" +
	"============================== Program Header ==============================\n" +
	"Program Header Type:          Program Header Table\n" +
	"Flags:                        Execute,Read\n" +
	"Offset Of Segment:            52\n" +
	"Virtual Address of Segment:   134512692\n" +
	"Physical Address of Segment:  134512692\n" +
	"Size of Segment:              256 bytes\n" +
	"Size of Segment in Memory:    256 bytes\n" +
	"\n" +
	"============================== Section Header ==============================\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .shstrab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        312\n" +
	"Section Size:                 28\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         Symbol Table\n" +
	"Section Name:                 .symtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        336\n" +
	"Section Size:                 16\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"Type:                         String Table\n" +
	"Section Name:                 .strtab\n" +
	"Flags:                        \n" +
	"Section Vaddr in memory:      0\n" +
	"Offset of the section:        361\n" +
	"Section Size:                 4\n" +
	"Associated Section:           0\n" +
	"Section Extra Information:    0\n" +
	"\n" +
	"============================== Symbol Table ==============================\n" +
	"Symbol Name:                  test"

// TestELFInfoFixtures runs CyberChef's four ELF Info fixture cases.
func TestELFInfoFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"ELF Info 32-bit ELF Little Endian.", elf32LE, elf32LEWant,
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "ELF Info", Args: []any{}}},
		},
		{
			"ELF Info 32-bit ELF Big Endian.", elf32BE, elf32BEWant,
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "ELF Info", Args: []any{}}},
		},
		{
			"ELF Info 64-bit ELF Little Endian.", elf64LE, elf64LEWant,
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "ELF Info", Args: []any{}}},
		},
		{
			"ELF Info 64-bit ELF Big Endian.", elf64BE, elf64BEWant,
			core.Recipe{{Op: "From Hex", Args: []any{"None"}}, {Op: "ELF Info", Args: []any{}}},
		},
	})
}

// elfGolden is one case from testdata/elfinfo.jsonl: a hand-built ELF file and
// either the report it produces or the fault it is refused with. Every case is
// anchored to CyberChef: byte-identical output where upstream is sound, and
// where upstream's int32 coercion or silent out-of-range reads corrupt the
// answer, the corrected value verified to truncate exactly to upstream's.
type elfGolden struct {
	Name     string `json:"name"`
	InputHex string `json:"inputHex"`
	Output   string `json:"output"`
	Error    string `json:"error"`
}

// TestELFInfoGoldens replays the whole synthetic corpus: header edge values,
// every segment and section type, symbol-table arithmetic, and one truncation
// per distinct failure shape.
func TestELFInfoGoldens(t *testing.T) {
	for _, g := range readJSONL[elfGolden](t, "testdata/elfinfo.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			got, err := runOp(t, "ELF Info", string(unhex(t, g.InputHex)))
			if g.Error != "" {
				if err == nil {
					t.Fatalf("accepted a file that should be refused with %q", g.Error)
				}
				if err.Error() != g.Error {
					t.Errorf("refused with %q\nwant         %q", err.Error(), g.Error)
				}
				return
			}
			if err != nil {
				t.Fatalf("refused a file that should be read: %v", err)
			}
			if got != g.Output {
				t.Errorf("got  %q\nwant %q", got, g.Output)
			}
		})
	}
}

// TestELFInfoInvalid covers the fixture whose expected output is the error
// message: input that does not begin with the ELF magic number.
func TestELFInfoInvalid(t *testing.T) {
	for _, in := range []string{"\x7f\x00\x00\x00", "\x7fEL", "", "not an elf at all"} {
		_, err := runOp(t, "ELF Info", in)
		if err == nil {
			t.Fatalf("%q: want an error, got none", in)
		}
		if !strings.Contains(err.Error(), "Invalid ELF") {
			t.Fatalf("%q: got %q, want it to mention Invalid ELF", in, err)
		}
	}
}
