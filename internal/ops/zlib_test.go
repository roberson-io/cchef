package ops

import (
	"hash/adler32"
	"testing"
)

// TestZlibDeflateHeader covers the two header bytes and the checksum trailer
// that wrap a raw DEFLATE stream to make a zlib one. The first byte records
// the method and window size; the second carries a guess at the compression
// level and a check value that makes the pair a multiple of 31.
func TestZlibDeflateHeader(t *testing.T) {
	const input = "The quick brown fox jumped over the slow dog"
	for _, tc := range []struct {
		compressionType string
		wantLevel       byte
	}{
		{"None (Store)", 0},
		{"Fixed Huffman Coding", 1},
		{"Dynamic Huffman Coding", 2},
	} {
		t.Run(tc.compressionType, func(t *testing.T) {
			out, err := runOp(t, "Zlib Deflate", input, tc.compressionType)
			if err != nil {
				t.Fatalf("Zlib Deflate: %v", err)
			}
			if len(out) < 6 {
				t.Fatalf("output is only %d bytes", len(out))
			}
			cmf, flg := out[0], out[1]
			if cmf != 0x78 {
				t.Errorf("CMF is %#02x, want 0x78 (deflate, 32K window)", cmf)
			}
			if level := flg >> 6; level != tc.wantLevel {
				t.Errorf("level bits are %d, want %d", level, tc.wantLevel)
			}
			if flg&0x20 != 0 {
				t.Error("the preset-dictionary bit is set, and should not be")
			}
			if (int(cmf)*256+int(flg))%31 != 0 {
				t.Errorf("header %02x%02x is not a multiple of 31", cmf, flg)
			}

			sum := adler32.Checksum([]byte(input))
			trailer := out[len(out)-4:]
			want := string([]byte{
				byte(sum >> 24), byte(sum >> 16), byte(sum >> 8), byte(sum),
			})
			if trailer != want {
				t.Errorf("checksum trailer is %x, want %x", trailer, want)
			}
		})
	}
}

// TestZlibInflateErrors covers the input the reader refuses.
func TestZlibInflateErrors(t *testing.T) {
	good, err := runOp(t, "Zlib Deflate", "The cat sat on the mat.", "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zlib Deflate: %v", err)
	}
	for _, tc := range []struct{ name, input string }{
		{"empty", ""},
		{"no header", "hello"},
		{"truncated", good[:len(good)/2]},
		{"bad checksum", good[:len(good)-1] + "\x00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runOp(t, "Zlib Inflate", tc.input,
				float64(0), float64(0), "Adaptive", false, false)
			if err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// TestZlibInflateStartIndex covers the one argument that changes the result:
// the stream is read from that byte onwards.
func TestZlibInflateStartIndex(t *testing.T) {
	const want = "The cat sat on the mat."
	compressed, err := runOp(t, "Zlib Deflate", want, "Dynamic Huffman Coding")
	if err != nil {
		t.Fatalf("Zlib Deflate: %v", err)
	}
	out, err := runOp(t, "Zlib Inflate", "\x00\x00\x00"+compressed,
		float64(3), float64(0), "Adaptive", false, false)
	if err != nil {
		t.Fatalf("Zlib Inflate: %v", err)
	}
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
