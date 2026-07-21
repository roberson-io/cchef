package ops

// BLAKE3 fixtures transcribed from ../CyberChef/tests/operations/tests/BLAKE3.mjs
// (single-chunk cases and the official 0/7-byte test vectors). CyberChef wraps
// @noble/hashes; the multi-chunk (>1024-byte) tree path has no upstream fixture,
// so TestBLAKE3Tree adds official BLAKE3 test vectors (input byte i = i%251),
// generated from that same library, to exercise it.

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// blake3Recipe builds a BLAKE3 recipe: Size (bytes), Key.
func blake3Recipe(size int, key string) core.Recipe {
	return core.Recipe{{Op: "BLAKE3", Args: []any{size, key}}}
}

func TestBLAKE3(t *testing.T) {
	const key32 = "ThiskeyisexactlythirtytwoBytesLo"
	runCases(t, []opCase{
		{"8 Hello world", "Hello world", "e7e6fb7d2869d109", blake3Recipe(8, "")},
		{"16 Hello world 2", "Hello world 2", "2a3df5fe5f0d3fcdd995fc203c7f7c52", blake3Recipe(16, "")},
		{
			"32 Hello world", "Hello world",
			"e7e6fb7d2869d109b62cdb1227208d4016cdaa0af6603d95223c6a698137d945", blake3Recipe(32, ""),
		},
		{"keyed", "Hello world", "59dd23ac9d025690", blake3Recipe(8, key32)},
		{
			"keyed 2 (lowercase l)", "Hello world", "c8302c9634c1da42",
			blake3Recipe(8, "ThiskeyisexactlythirtytwoByteslo"),
		},
		// Official BLAKE3 test vectors (XOF output length 131).
		{
			"std 0-byte plain", "",
			"af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262e00f03e7b69af26b7faaf09fcd333050338ddfe085b8cc869ca98b206c08243a26f5487789e8f660afe6c99ef9e0c52b92e7393024a80459cf91f476f9ffdbda7001c22e159b402631f277ca96f2defdf1078282314e763699a31c5363165421cce14d",
			blake3Recipe(131, ""),
		},
		{
			"std 0-byte keyed", "",
			"92b2b75604ed3c761f9d6f62392c8a9227ad0ea3f09573e783f1498a4ed60d26b18171a2f22a4b94822c701f107153dba24918c4bae4d2945c20ece13387627d3b73cbf97b797d5e59948c7ef788f54372df45e45e4293c7dc18c1d41144a9758be58960856be1eabbe22c2653190de560ca3b2ac4aa692a9210694254c371e851bc8f",
			blake3Recipe(131, "whats the Elvish word for friend"),
		},
	})
}

// The 7-byte keyed std test vector, with input supplied as hex.
func TestBLAKE3FromHex(t *testing.T) {
	runCases(t, []opCase{
		{
			"std 7-byte keyed", "0001020304050607",
			"be2f5495c61cba1bb348a34948c004045e3bd4dae8f0fe82bf44d0da245a060048eb5e68ce6dea1eb0229e144f578b3aa7e9f4f85febd135df8525e6fe40c6f0340d13dd09b255ccd5112a94238f2be3c0b5b7ecde06580426a93e0708555a265305abf86d874e34b4995b788e37a823491f25127a502fe0704baa6bfdf04e76c13276",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "BLAKE3", Args: []any{131, "whats the Elvish word for friend"}},
			},
		},
	})
}

// blake3Pattern is the official test-vector input: byte i = i % 251.
func blake3Pattern(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return string(b)
}

// Multi-chunk inputs (>1024 bytes) exercise the chunk tree, parent nodes and the
// subtree stack. Vectors are the official BLAKE3 test vectors (i%251 input),
// generated from @noble/hashes (the library CyberChef wraps).
func TestBLAKE3Tree(t *testing.T) {
	const key = "whats the Elvish word for friend"
	cases := []struct {
		n    int
		key  string
		size int
		want string
	}{
		{1024, "", 32, "42214739f095a406f3fc83deb889744ac00df831c10daa55189b5d121c855af7"},
		{1025, "", 32, "d00278ae47eb27b34faecf67b4fe263f82d5412916c1ffd97c8cb7fb814b8444"},
		{2048, "", 32, "e776b6028c7cd22a4d0ba182a8bf62205d2ef576467e838ed6f2529b85fba24a"},
		{3073, "", 32, "7124b49501012f81cc7f11ca069ec9226cecb8a2c850cfe644e327d22d3e1cd3"},
		{6144, "", 32, "3e2e5b74e048f3add6d21faab3f83aa44d3b2278afb83b80b3c35164ebeca205"},
		{2048, key, 32, "879cf1fa2ea0e79126cb1063617a05b6ad9d0b696d0d757cf053439f60a99dd1"},
	}
	for _, c := range cases {
		got, err := runOp(t, "BLAKE3", blake3Pattern(c.n), c.size, c.key)
		if err != nil || got != c.want {
			t.Fatalf("n=%d key=%q: got %q, %v\nwant %q", c.n, c.key, got, err, c.want)
		}
	}
}

// The 16390-byte XOF output is transcribed as a prefix/suffix/length check,
// matching CyberChef's expectedMatch regexes.
func TestBLAKE3LongXOF(t *testing.T) {
	cases := []struct{ key, prefix, suffix string }{
		{"", "4878", "555fe06b242738d5"},
		{"ThiskeyisexactlythirtytwoBytesLo", "a8d0", "19ccd9b9726b46ae"},
	}
	for _, c := range cases {
		got, err := runOp(t, "BLAKE3", "test", 16390, c.key)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(got) != 16390*2 {
			t.Fatalf("length = %d, want %d", len(got), 16390*2)
		}
		if !strings.HasPrefix(got, c.prefix) || !strings.HasSuffix(got, c.suffix) {
			t.Fatalf("key=%q: got %s...%s, want %s...%s", c.key, got[:4], got[len(got)-16:], c.prefix, c.suffix)
		}
	}
}

// A key that is not exactly 32 bytes is rejected, matching CyberChef.
func TestBLAKE3KeyLength(t *testing.T) {
	_, err := runOp(t, "BLAKE3", "Hello world", 8, "shortkey")
	want := "The key must be exactly 32 bytes long"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v\nwant %q", err, want)
	}
}
