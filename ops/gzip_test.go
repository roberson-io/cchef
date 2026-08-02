package ops

import (
	"hash/crc32"
	"testing"
	"time"

	"github.com/roberson-io/cchef/core"
)

// TestGzipFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/Gzip.mjs). The first four drop the
// ten-byte header before comparing, because it records the time gzip ran.
func TestGzipFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Gzip: No comment, no checksum and no filename",
			"The quick brown fox jumped over the slow dog",
			"0dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "", "", false}},
				{Op: "Drop bytes", Args: []any{0.0, 10.0, false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"Gzip: No comment, no checksum and has a filename",
			"The quick brown fox jumped over the slow dog",
			"636f6d6d656e74000dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "comment", "", false}},
				{Op: "Drop bytes", Args: []any{0.0, 10.0, false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"Gzip: Has a comment, no checksum and no filename",
			"The quick brown fox jumped over the slow dog",
			"636f6d6d656e74000dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "", "comment", false}},
				{Op: "Drop bytes", Args: []any{0.0, 10.0, false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"Gzip: Has a comment, no checksum and has a filename",
			"The quick brown fox jumped over the slow dog",
			"66696c656e616d6500636f6d6d656e74000dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "filename", "comment", false}},
				{Op: "Drop bytes", Args: []any{0.0, 10.0, false}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"Gzip: Comment with checksum round-trips through Gunzip",
			"hello hello hello", "hello hello hello",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "", "test", true}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
		{
			"Gzip: Filename and comment with checksum round-trips through Gunzip",
			"The quick brown fox jumped over the slow dog",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "file.txt", "a comment", true}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
		{
			"Gzip: No comment, with checksum round-trips through Gunzip",
			"The quick brown fox jumped over the slow dog",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "", "", true}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
		{
			"Gzip: No options round-trips through Gunzip",
			"The quick brown fox jumped over the slow dog",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "Gzip", Args: []any{"Dynamic Huffman Coding", "", "", false}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
	})
}

// TestGunzipFixtures covers CyberChef's own cases
// (CyberChef's tests/operations/tests/Gunzip.mjs).
func TestGunzipFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Gunzip: No comment, no checksum and no filename",
			"1f8b0800f7c8f85d00ff0dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "From Hex", Args: []any{"None"}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
		{
			"Gunzip: No comment, no checksum and filename",
			"1f8b080843c9f85d00ff66696c656e616d65000dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "From Hex", Args: []any{"None"}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
		{
			"Gunzip: Has a comment, no checksum and has a filename",
			"1f8b08186fc9f85d00ff66696c656e616d6500636f6d6d656e74000dc9dd0180200804e0556ea8262848fb3dc588c6a7e76faa8aeedb726036c68d951f76bf9a0af8aae1f97d9c0c084b02509cbf8c2c000000",
			"The quick brown fox jumped over the slow dog",
			core.Recipe{
				{Op: "From Hex", Args: []any{"None"}},
				{Op: "Gunzip", Args: []any{}},
			},
		},
	})
}

// TestGzipHeaderFlags covers the header the options produce: which flag bits
// are set, and the order the optional fields are written in.
func TestGzipHeaderFlags(t *testing.T) {
	const (
		flagFHCRC    = 1 << 1
		flagFNAME    = 1 << 3
		flagFCOMMENT = 1 << 4
	)
	for _, tc := range []struct {
		name, filename, comment string
		fhcrc                   bool
		wantFlag                byte
		wantFields              string
	}{
		{"none", "", "", false, 0, ""},
		{"filename", "a.txt", "", false, flagFNAME, "a.txt\x00"},
		{"comment", "", "hi", false, flagFCOMMENT, "hi\x00"},
		{"both", "a.txt", "hi", false, flagFNAME | flagFCOMMENT, "a.txt\x00hi\x00"},
		{"checksum", "", "", true, flagFHCRC, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Gzip", "hello", "Dynamic Huffman Coding",
				tc.filename, tc.comment, tc.fhcrc)
			if err != nil {
				t.Fatalf("Gzip: %v", err)
			}
			if len(out) < 10 {
				t.Fatalf("output is only %d bytes", len(out))
			}
			if out[0] != 0x1f || out[1] != 0x8b || out[2] != 0x08 {
				t.Errorf("magic is %x, want 1f8b08", out[:3])
			}
			if out[3] != tc.wantFlag {
				t.Errorf("flags are %#02x, want %#02x", out[3], tc.wantFlag)
			}
			if out[8] != 0x00 || out[9] != 0xff {
				t.Errorf("XFL/OS are %x, want 00ff", out[8:10])
			}
			if got := out[10 : 10+len(tc.wantFields)]; got != tc.wantFields {
				t.Errorf("optional fields are %q, want %q", got, tc.wantFields)
			}
		})
	}
}

// TestGunzipErrors covers the input the reader refuses.
func TestGunzipErrors(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"not gzip", "this is not gzip at all"},
		{"truncated", "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x0d"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := runOp(t, "Gunzip", tc.input); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestGzipHeaderChecksum covers the optional header checksum. It cannot be
// pinned by a golden, because it is taken over the timestamp and so differs
// from one run to the next; what is fixed is the rule, so that is what is
// checked — the two bytes are the low half of the checksum of everything before
// them.
func TestGzipHeaderChecksum(t *testing.T) {
	for _, tc := range []struct{ name, filename, comment string }{
		{"no fields", "", ""},
		{"filename", "a.txt", ""},
		{"comment", "", "hi"},
		{"both", "a.txt", "hi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runOp(t, "Gzip", "hello", "Dynamic Huffman Coding",
				tc.filename, tc.comment, true)
			if err != nil {
				t.Fatalf("Gzip: %v", err)
			}
			// The header runs to the checksum: ten fixed bytes, then each
			// optional field with its terminating zero.
			end := 10
			if tc.filename != "" {
				end += len(tc.filename) + 1
			}
			if tc.comment != "" {
				end += len(tc.comment) + 1
			}
			if len(out) < end+2 {
				t.Fatalf("output is only %d bytes", len(out))
			}
			want := uint16(crc32.ChecksumIEEE([]byte(out[:end])))
			got := uint16(out[end]) | uint16(out[end+1])<<8
			if got != want {
				t.Errorf("header checksum is %#04x, want %#04x", got, want)
			}
		})
	}
}

// TestGzipMultipleMembers checks that a file holding several gzip streams one
// after another is read whole, which is what the gzip command does.
func TestGzipMultipleMembers(t *testing.T) {
	one, err := runOp(t, "Gzip", "one ", "Dynamic Huffman Coding", "", "", false)
	if err != nil {
		t.Fatalf("Gzip: %v", err)
	}
	two, err := runOp(t, "Gzip", "two", "Dynamic Huffman Coding", "", "", false)
	if err != nil {
		t.Fatalf("Gzip: %v", err)
	}
	out, err := runOp(t, "Gunzip", one+two)
	if err != nil {
		t.Fatalf("Gunzip: %v", err)
	}
	if out != "one two" {
		t.Errorf("got %q, want %q", out, "one two")
	}
}

// TestGzipRecordsTheTime checks that the header carries the time the stream was
// written, which is what makes the output differ between runs.
func TestGzipRecordsTheTime(t *testing.T) {
	frozen := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	restore := gzNow
	gzNow = func() time.Time { return frozen }
	defer func() { gzNow = restore }()

	out, err := runOp(t, "Gzip", "hello", "Dynamic Huffman Coding", "", "", false)
	if err != nil {
		t.Fatalf("Gzip: %v", err)
	}
	got := uint32(out[4]) | uint32(out[5])<<8 | uint32(out[6])<<16 | uint32(out[7])<<24
	if want := uint32(frozen.Unix()); got != want {
		t.Errorf("timestamp is %d, want %d", got, want)
	}
}

// TestGzipTextField covers how a filename or comment reaches the header. The
// field holds bytes, so a character needing more than one is written most
// significant byte first, which is how CyberChef writes it a UTF-16 unit at a
// time.
func TestGzipTextField(t *testing.T) {
	for _, tc := range []struct {
		name, in string
		want     []byte
	}{
		{"ascii", "a.txt", []byte("a.txt")},
		{"latin-1", "café", []byte{'c', 'a', 'f', 0xe9}},
		{"beyond one byte", "ü中", []byte{0xfc, 0x4e, 0x2d}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := gzText(tc.in); string(got) != string(tc.want) {
				t.Errorf("got %x, want %x", got, tc.want)
			}
		})
	}
}
