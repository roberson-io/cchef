package ops

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// lz4Sentence is the text the repeated-text cases are built from.
const lz4Sentence = "The quick brown fox jumps over the lazy dog. "

// lz4Golden is one case from testdata/lz4.jsonl: an input, described so it can
// be rebuilt here, and the frame CyberChef compresses it into. Frames small
// enough to read are kept whole; the two that run to megabytes are kept only as
// a length and a digest.
type lz4Golden struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Spec struct {
		Hex  string `json:"hex"`
		Len  int    `json:"len"`
		N    int    `json:"n"`
		Seed int64  `json:"seed"`
		Val  int    `json:"val"`
	} `json:"spec"`
	PlainLen     int    `json:"plainLen"`
	PlainSHA256  string `json:"plainSHA256"`
	StreamLen    int    `json:"streamLen"`
	StreamSHA256 string `json:"streamSHA256"`
	StreamHex    string `json:"streamHex"`
}

// plain rebuilds the bytes a golden was made from.
func (g lz4Golden) plain(t *testing.T) []byte {
	t.Helper()
	switch g.Kind {
	case "raw":
		return unhex(t, g.Spec.Hex)
	case "range":
		b := make([]byte, g.Spec.Len)
		for i := range b {
			b[i] = byte(i)
		}
		return b
	case "text":
		return []byte(strings.Repeat(lz4Sentence, g.Spec.N))
	case "random":
		return deflateRandom(g.Spec.Len, g.Spec.Seed)
	case "byte":
		b := make([]byte, g.Spec.Len)
		for i := range b {
			b[i] = byte(g.Spec.Val)
		}
		return b
	case "repeat":
		unit := unhex(t, g.Spec.Hex)
		out := make([]byte, 0, len(unit)*g.Spec.N)
		for range g.Spec.N {
			out = append(out, unit...)
		}
		return out
	}
	t.Fatalf("%s: unknown input kind %q", g.Name, g.Kind)
	return nil
}

// unhex decodes hex, failing the test rather than the caller.
func unhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// digest is the shorthand the comparisons below are written in.
func digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

// readJSONL reads a golden file a line at a time into fn's argument type.
func readJSONL[T any](t *testing.T, path string) []T {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open goldens: %v", err)
	}
	defer func() { _ = f.Close() }()

	var out []T
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	for sc.Scan() {
		var row T
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatalf("parse goldens: %v", err)
		}
		out = append(out, row)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read goldens: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("no goldens in %s", path)
	}
	return out
}

// lz4CommandFrame is one stream the lz4 command wrote, recorded to check that
// the reader copes with the frame options CyberChef never writes. Its input is
// the sentence repeated N times.
type lz4CommandFrame struct {
	Name        string `json:"name"`
	N           int    `json:"n"`
	PlainLen    int    `json:"plainLen"`
	PlainSHA256 string `json:"plainSHA256"`
	StreamHex   string `json:"streamHex"`
}

// TestLZ4Fixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/Compress.mjs).
func TestLZ4Fixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"LZ4 Compress",
			"The cat sat on the mat.",
			"04224d184070df170000805468652063617420736174206f6e20746865206d61742e00000000",
			core.Recipe{
				{Op: "LZ4 Compress", Args: []any{}},
				{Op: "To Hex", Args: []any{"None"}},
			},
		},
		{
			"LZ4 Decompress",
			"04224d184070df170000805468652063617420736174206f6e20746865206d61742e00000000",
			"The cat sat on the mat.",
			core.Recipe{
				{Op: "From Hex", Args: []any{"None"}},
				{Op: "LZ4 Decompress", Args: []any{}},
			},
		},
	})
}

// TestLZ4CompressGoldens compresses each input and compares the frame with the
// one CyberChef produced. LZ4 leaves the encoder free to choose how hard it
// looks for repeats, so this only holds because the search is ported exactly.
func TestLZ4CompressGoldens(t *testing.T) {
	for _, g := range readJSONL[lz4Golden](t, "testdata/lz4.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "LZ4 Compress", string(g.plain(t)))
			if err != nil {
				t.Fatalf("LZ4 Compress: %v", err)
			}
			if g.StreamHex != "" {
				if got := hex.EncodeToString([]byte(out)); got != g.StreamHex {
					t.Fatalf("got  %s\nwant %s", got, g.StreamHex)
				}
				return
			}
			if len(out) != g.StreamLen {
				t.Errorf("wrote %d bytes, want %d", len(out), g.StreamLen)
			}
			if sum := digest([]byte(out)); sum != g.StreamSHA256 {
				t.Errorf("digest %s, want %s", sum, g.StreamSHA256)
			}
		})
	}
}

// TestLZ4DecompressGoldens reads back the frames CyberChef wrote.
func TestLZ4DecompressGoldens(t *testing.T) {
	for _, g := range readJSONL[lz4Golden](t, "testdata/lz4.jsonl") {
		if g.StreamHex == "" {
			continue
		}
		t.Run(g.Name, func(t *testing.T) {
			out, err := runOp(t, "LZ4 Decompress", string(unhex(t, g.StreamHex)))
			if err != nil {
				t.Fatalf("LZ4 Decompress: %v", err)
			}
			if len(out) != g.PlainLen {
				t.Errorf("read back %d bytes, want %d", len(out), g.PlainLen)
			}
			if sum := digest([]byte(out)); sum != g.PlainSHA256 {
				t.Errorf("digest %s, want %s", sum, g.PlainSHA256)
			}
		})
	}
}

// TestLZ4RoundTrips puts every golden input through both operations, which
// covers the two multi-block cases the goldens are too large to hold whole.
func TestLZ4RoundTrips(t *testing.T) {
	for _, g := range readJSONL[lz4Golden](t, "testdata/lz4.jsonl") {
		t.Run(g.Name, func(t *testing.T) {
			want := string(g.plain(t))
			frame, err := runOp(t, "LZ4 Compress", want)
			if err != nil {
				t.Fatalf("LZ4 Compress: %v", err)
			}
			back, err := runOp(t, "LZ4 Decompress", frame)
			if err != nil {
				t.Fatalf("LZ4 Decompress: %v", err)
			}
			if back != want {
				t.Errorf("round trip changed the data (%d bytes in, %d out)",
					len(want), len(back))
			}
		})
	}
}

// TestLZ4DecompressCommandFrames reads frames written by the lz4 command, which
// uses frame options CyberChef never writes: content size, block checksums, a
// content checksum, smaller blocks and blocks that stand alone.
func TestLZ4DecompressCommandFrames(t *testing.T) {
	for _, f := range readJSONL[lz4CommandFrame](t, "testdata/lz4frames.jsonl") {
		t.Run(f.Name, func(t *testing.T) {
			out, err := runOp(t, "LZ4 Decompress", string(unhex(t, f.StreamHex)))
			if err != nil {
				t.Fatalf("LZ4 Decompress: %v", err)
			}
			if len(out) != f.PlainLen {
				t.Errorf("read back %d bytes, want %d", len(out), f.PlainLen)
			}
			if sum := digest([]byte(out)); sum != f.PlainSHA256 {
				t.Errorf("digest %s, want %s", sum, f.PlainSHA256)
			}
		})
	}
}

// TestLZ4XXH32 covers the hash the frame format checks itself with, against
// values taken from the implementation CyberChef uses. The first is the
// published xxHash32 answer for no input at all.
func TestLZ4XXH32(t *testing.T) {
	cases := []struct {
		seed uint32
		hex  string
		want uint32
	}{
		{0x00000000, "", 0x02cc5d05},
		{0x00000000, "ec", 0x9aed67f1},
		{0x00000000, "44b0", 0xa60e50ac},
		{0x00000000, "9c1809", 0xc364efe5},
		{0x00000000, "f580147e", 0x577bbb5c},
		{0x00000000, "4de91f799f", 0x87ffe94b},
		{0x00000000, "a5512a740929", 0xa11af942},
		{0x00000000, "feb9356f721ae2", 0xbc1ea355},
		{0x00000000, "5622406bdc0b5cf9", 0x5362eaf8},
		{0x00000000, "ae8a4b6645fbd5af63", 0xea0c488a},
		{0x00000000, "07f35661afec4f64be26", 0x830ab14b},
		{0x00000000, "5f5b615d18dcc81a1ae71d", 0xe6847877},
		{0x00000000, "b7c36c5881cd42d075a73a77", 0x35205406},
		{0x00000000, "102c7753ebbdbb85d06857a31b", 0x32da951e},
		{0x00000000, "6894824f54ae353b2c2874cfd725", 0x28f383fd},
		{0x00000000, "c0fc8d4abe9eaff187e991fa945eec", 0x4c5913b5},
		{0x00000000, "19659845278f28a6e3a9ad2650964141", 0x934d3eb4},
		{0x00000000, "71cda3419180a25c3e6aca520cce96ce23", 0xce013e4e},
		{0x00000000, "c936ae3cfa701b11992ae77dc806ea5c0c78", 0xcf6eb99e},
		{0x00000000, "229eba37636195c7f5eb04a9843f3fe9f511f8", 0x2bbff836},
		{0x00000000, "7a06c532cd510e7d50ab20d540779476deaa618b", 0x9f866a9f},
		{0x00000000, "d26fd02e36428832ab6c3d00fdafe804c843ca56c0", 0x8484e1c2},
		{0x00000000, "2bd7db29a03201e8072c5a2cb9e73d91b1dc3321eede", 0x67c4820d},
		{0x00000000, "833fe62409237b9d62ec77587520911e9a759cec1d1975", 0x9c8eea4c},
		{0x00000000, "dba8f1207313f553bead94833158e6ac840e05b74c5423ca", 0x7585854d},
		{0x00000000, "3410fc1bdc046e09196db0afed903b396da76d827a8fd1394b", 0x03361601},
		{0x00000000, "8c79071645f4e8be742ecddbaac88fc65640d64da9ca7fa732ab", 0x23610473},
		{0x00000000, "e4e11212afe56174d0eeea066601e4543fd93f18d8052d161900a9", 0x8d670bee},
		{0x00000000, "3c491d0d18d6db2a2baf0732223939e12972a8e30640db85005612d3", 0x0920fd27},
		{0x00000000, "95b2280882c654df866f245ede718d6f120b11ae357b89f3e7ac7c71c3", 0x4106ee7c},
		{0x00000000, "ed1a3303ebb7ce95e23040899aa9e2fcfba47a7964b63762ce02e610a6cb", 0xe7e3f510},
		{0x00000000, "45823eff55a7474a3df05db556e23689e53de34392f1e4d1b55750af8a142c", 0x36b9b6a1},
		{0x00000000, "9eeb49fabe98c10099b17ae1131a8b17ced64c0ec12c923f9cadba4e6d5d2a8a", 0x0de59ce8},
		{0x00000000, "f65354f527883bb6f471970ccf52e0a4b76fb5d9f06740ae830323ed51a728218b", 0x6e2607ca},
		{0x00000000, "4ebc5ff19179b46b4f32b4388b8a3431a1081ea41ea2ee1d6a598d8b35f025b73485", 0x5b821c74},
		{0x00000000, "a7246aecfa692e21abf2d06447c389bf8aa1866f4ddd9c8b51aef72a1839234ddde385", 0x5627f08b},
		{0x00000000, "ff8c75e7645aa7d606b3ed8f03fbdd4c733aef3a7c184afa380461c9fc8221e3864163a8", 0x6dfdee4d},
		{0x00000000, "57f580e3cd4b218c61730abbbf3332da5cd25805ab53f868205aca68dfcc1f7a30a0402d54", 0x51597899},
		{0x00000000, "b05d8bde373b9a42bd3427e77c6b8767466bc1d0d98ea6d707b03407c3151d10d9fe1eb36fa3", 0xdceb19b4},
		{0x00000000, "08c596d9a02c14f718f4441238a4dbf42f042a9b08c95446ee069ea5a75e1ba6825cfb388a476a", 0xb770f6c8},
		{0x00000000, "602ea1d4091c8dad74b5603ef4dc3082189d9366370402b4d55b08448aa7183d2bbad9bea6eaed33", 0x8529e585},
		{0x00000001, "7ef4e84544236752fbb56b8f31a23a10e42814f5f55ca037cdcc11c64c9a3b2949c1bb6070", 0xcc50c36b},
		{0x9e3779b1, "7ef4e84544236752fbb56b8f31a23a10e42814f5f55ca037cdcc11c64c9a3b2949c1bb6070", 0x615baf42},
		{0xffffffff, "7ef4e84544236752fbb56b8f31a23a10e42814f5f55ca037cdcc11c64c9a3b2949c1bb6070", 0x805ae0ff},
	}
	for _, c := range cases {
		name := fmt.Sprintf("%d bytes seed %#x", len(c.hex)/2, c.seed)
		t.Run(name, func(t *testing.T) {
			if got := lz4XXH32(c.seed, unhex(t, c.hex)); got != c.want {
				t.Errorf("hash %#08x, want %#08x", got, c.want)
			}
		})
	}
}

// lz4Header is the seven bytes every frame here opens with: the magic number, a
// descriptor asking for four-megabyte linked blocks, and the descriptor's own
// checksum.
const lz4Header = "04224d184070df"

// lz4EndMark closes a frame.
const lz4EndMark = "00000000"

// lz4Block frames a block payload with its length.
func lz4Block(payload string) string {
	n := len(payload) / 2
	return fmt.Sprintf("%02x%02x%02x%02x", n&0xff, n>>8&0xff, n>>16&0xff, n>>24&0xff) + payload
}

// TestLZ4DecompressRejectsBadInput covers the malformed frames CyberChef reads
// without complaint, filling the gaps with zeroes. Each one is refused here
// instead.
func TestLZ4DecompressRejectsBadInput(t *testing.T) {
	// A stored block holding "The cat sat on the mat.".
	stored := "170000805468652063617420736174206f6e20746865206d61742e"

	cases := []struct {
		name string
		hex  string
	}{
		{"nothing at all", ""},
		{"shorter than a header", "04224d1840"},
		{"not an LZ4 frame", "deadbeef4070df" + lz4EndMark},
		{"the legacy format", "02214c18" + stored},
		{"a descriptor from another version", "04224d188070df" + stored + lz4EndMark},
		{"a block size that means nothing", "04224d18400031" + stored + lz4EndMark},
		{"a corrupt descriptor checksum", "04224d18407000" + stored + lz4EndMark},
		{"no block length", lz4Header + "0000"},
		{"no end mark", lz4Header + stored},
		{"a block longer than what is left", lz4Header + "ff000080" + "5468652063617420736174" + lz4EndMark},
		{"a match reaching before the start", lz4Header + lz4Block("40616161610500") + lz4EndMark},
		{"a match with no distance", lz4Header + lz4Block("40616161610000") + lz4EndMark},
		{"fewer literals than promised", lz4Header + lz4Block("40616161") + lz4EndMark},
		{"a literal count that never ends", lz4Header + lz4Block("f0ffff") + lz4EndMark},
		{"a match length that never ends", lz4Header + lz4Block("4f616161610100ff") + lz4EndMark},
		{"no room for a match distance", lz4Header + lz4Block("406161616105") + lz4EndMark},
		{"rubbish after the end mark", lz4Header + lz4Block("00") + lz4EndMark + "deadbeef"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, "LZ4 Decompress", string(unhex(t, c.hex))); err == nil {
				t.Fatal("read a frame that should have been refused")
			}
		})
	}
}

// TestLZ4DecompressRejectsCorruption covers the three checksums the format
// carries. CyberChef reads past all of them, so a damaged frame there comes back
// as damaged data with nothing said.
func TestLZ4DecompressRejectsCorruption(t *testing.T) {
	frames := map[string]lz4CommandFrame{}
	for _, f := range readJSONL[lz4CommandFrame](t, "testdata/lz4frames.jsonl") {
		frames[f.Name] = f
	}

	// The byte to damage, and how far into each stream it sits: the block
	// checksum sits after the block, the content checksum at the very end.
	cases := []struct {
		name  string
		frame string
		at    func(n int) int
	}{
		{"a corrupt block checksum", "block checksum", func(int) int { return 11 }},
		{"a corrupt content checksum", "default", func(n int) int { return n - 1 }},
		{"corrupt compressed data", "default", func(int) int { return 20 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, ok := frames[c.frame]
			if !ok {
				t.Fatalf("no golden frame named %q", c.frame)
			}
			stream := unhex(t, f.StreamHex)
			stream[c.at(len(stream))] ^= 0xff
			if _, err := runOp(t, "LZ4 Decompress", string(stream)); err == nil {
				t.Fatal("read a damaged frame without complaint")
			}
		})
	}
}

// lz4MakeHeader builds a frame header out of the descriptor bytes, working out
// the checksum a reader will hold it to.
func lz4MakeHeader(t *testing.T, flags, blockDesc byte, extra string) string {
	t.Helper()
	desc := append([]byte{flags, blockDesc}, unhex(t, extra)...)
	return "04224d18" + hex.EncodeToString(desc) +
		fmt.Sprintf("%02x", byte(lz4XXH32(0, desc)>>8))
}

// lz4Stored frames a payload as a block the writer left as it was.
func lz4Stored(payload string) string {
	n := len(payload)/2 | 0x80000000
	return fmt.Sprintf("%02x%02x%02x%02x", n&0xff, n>>8&0xff, n>>16&0xff, n>>24&0xff) + payload
}

// TestLZ4DecompressHeaderOptions covers the parts of a frame descriptor
// CyberChef never writes and, in the case of a dictionary identifier, never
// makes room for when reading either.
func TestLZ4DecompressHeaderOptions(t *testing.T) {
	const hi = "6869" // "hi", small enough to be worth storing as it is

	cases := []struct {
		name string
		hex  string
		want string
	}{
		{
			"a dictionary identifier",
			lz4MakeHeader(t, lz4Version|lz4FlagDictID, 0x70, "01020304") +
				lz4Stored(hi) + lz4EndMark,
			"hi",
		},
		{
			"a stated content size",
			lz4MakeHeader(t, lz4Version|lz4FlagContentSize, 0x70, "0200000000000000") +
				lz4Stored(hi) + lz4EndMark,
			"hi",
		},
		{
			"the smallest blocks allowed",
			lz4MakeHeader(t, lz4Version, 0x40, "") + lz4Stored(hi) + lz4EndMark,
			"hi",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := runOp(t, "LZ4 Decompress", string(unhex(t, c.hex)))
			if err != nil {
				t.Fatalf("LZ4 Decompress: %v", err)
			}
			if out != c.want {
				t.Errorf("got %q, want %q", out, c.want)
			}
		})
	}
}

// TestLZ4DecompressRejectsBadHeaders covers descriptors that promise something
// the rest of the frame does not deliver.
func TestLZ4DecompressRejectsBadHeaders(t *testing.T) {
	const hi = "6869"

	cases := []struct {
		name string
		hex  string
	}{
		{
			"a content size cut off partway",
			"04224d18" + fmt.Sprintf("%02x", lz4Version|lz4FlagContentSize) + "70" + "02000000",
		},
		{
			"a dictionary identifier cut off partway",
			"04224d18" + fmt.Sprintf("%02x", lz4Version|lz4FlagDictID) + "70" + "0102",
		},
		{"no descriptor checksum", "04224d184070"},
		{
			"a content size that does not match",
			lz4MakeHeader(t, lz4Version|lz4FlagContentSize, 0x70, "0300000000000000") +
				lz4Stored(hi) + lz4EndMark,
		},
		{
			"a block bigger than the frame allows",
			lz4MakeHeader(t, lz4Version, 0x40, "") + "00000200" + strings.Repeat("00", 8),
		},
		{
			"no block checksum",
			lz4MakeHeader(t, lz4Version|lz4FlagBlockChecksum, 0x70, "") + lz4Stored(hi),
		},
		{
			"no content checksum",
			lz4MakeHeader(t, lz4Version|lz4FlagContentChecksum, 0x70, "") +
				lz4Stored(hi) + lz4EndMark,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := runOp(t, "LZ4 Decompress", string(unhex(t, c.hex))); err == nil {
				t.Fatal("read a frame that should have been refused")
			}
		})
	}
}

// TestLZ4CompressEmpty covers the shortest frame there is: a header, no blocks
// and an end mark.
func TestLZ4CompressEmpty(t *testing.T) {
	out, err := runOp(t, "LZ4 Compress", "")
	if err != nil {
		t.Fatalf("LZ4 Compress: %v", err)
	}
	if got := hex.EncodeToString([]byte(out)); got != lz4Header+lz4EndMark {
		t.Errorf("got %s, want %s", got, lz4Header+lz4EndMark)
	}
	back, err := runOp(t, "LZ4 Decompress", out)
	if err != nil {
		t.Fatalf("LZ4 Decompress: %v", err)
	}
	if back != "" {
		t.Errorf("read back %q, want nothing", back)
	}
}
