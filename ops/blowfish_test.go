package ops

import (
	"crypto/cipher"
	"errors"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// blowfishRec builds a recipe with the CyberChef argument order:
// Key, IV, Mode, Input, Output. Key and IV use the Hex toggle option.
func blowfishRec(op, key, iv, mode, in, out string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{
		core.ToggleString{Value: key, Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		mode, in, out,
	}}}
}

// Cases transcribed from CyberChef's tests/operations/tests/Crypt.mjs.
func TestBlowfishFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Blowfish Encrypt: ECB, ASCII", "The quick brown fox jumps over the lazy dog.",
			"f7784137ab1bf51546c0b120bdb7fed4509116e49283b35fab0e4292ac86251a9bf908330e3393815e3356bb26524027",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "0000000000000000", "ECB", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt: ECB, Binary", "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			"3d1bf0e87d83782d435a0ca58179ca290184867f52295af5c0fb4dcac7c6c68942906bb421d05925cc7d9cd21532376a0f6ae4c3f008b250381ffa9624f5eb697dbd44de48cf5593ea7dbf5842238474b546ceeb29f6cf327a7d13698786b8d14451f52fb0f5760a",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "0000000000000000", "ECB", "Hex", "Hex"),
		},
		{
			"Blowfish Decrypt: ECB, ASCII", "f7784137ab1bf51546c0b120bdb7fed4509116e49283b35fab0e4292ac86251a9bf908330e3393815e3356bb26524027",
			"The quick brown fox jumps over the lazy dog.",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "0000000000000000", "ECB", "Hex", "Raw"),
		},
		{
			"Blowfish Decrypt: ECB, Binary", "3d1bf0e87d83782d435a0ca58179ca290184867f52295af5c0fb4dcac7c6c68942906bb421d05925cc7d9cd21532376a0f6ae4c3f008b250381ffa9624f5eb697dbd44de48cf5593ea7dbf5842238474b546ceeb29f6cf327a7d13698786b8d14451f52fb0f5760a",
			"7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "0000000000000000", "ECB", "Hex", "Hex"),
		},
		{
			"Blowfish Encrypt: CBC, ASCII", "The quick brown fox jumps over the lazy dog.",
			"398433f39e938286a35fc240521435b6972f3fe96846b54ab9351aa5fa9e10a6a94074e883d1cb36cb9657c817274b60",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "CBC", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt: CBC, Binary", "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			"3b42c51465896524e66c2fd2404c8c2b4eb26c760671f131c3372d374f48283ca9a5404d3d8aabd2a886c6551393ca41c682580f1c81f16046e3bec7b59247bdfca1d40bf2ad8ede9de99cb44b36658f775999d37776b3b1a085b9530e54ece69e1875e1bdc8cdcf",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "CBC", "Hex", "Hex"),
		},
		{
			"Blowfish Decrypt: CBC, ASCII", "398433f39e938286a35fc240521435b6972f3fe96846b54ab9351aa5fa9e10a6a94074e883d1cb36cb9657c817274b60",
			"The quick brown fox jumps over the lazy dog.",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "CBC", "Hex", "Raw"),
		},
		{
			"Blowfish Decrypt: CBC, Binary", "3b42c51465896524e66c2fd2404c8c2b4eb26c760671f131c3372d374f48283ca9a5404d3d8aabd2a886c6551393ca41c682580f1c81f16046e3bec7b59247bdfca1d40bf2ad8ede9de99cb44b36658f775999d37776b3b1a085b9530e54ece69e1875e1bdc8cdcf",
			"7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "CBC", "Hex", "Hex"),
		},
		{
			"Blowfish Encrypt: CFB, ASCII", "The quick brown fox jumps over the lazy dog.",
			"c8ca123592570c1fcb138d4ec08f7af14ad49363245be1ac25029c8ffc508b3217e75faaa5566426180fec8f",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "CFB", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt: CFB, Binary", "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			"e6ac1324d1576beab00e855de3f4ac1f5e3cbf89f4c2a743a5737895067ac5012e5bdb92477e256cc07bf691b58e721179b550e694abb0be7cbdc42586db755bf795f4338f47d356c57453afa6277e46aaeb3405f9744654a477f06c2ad92ede90555759",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "CFB", "Hex", "Hex"),
		},
		{
			"Blowfish Decrypt: CFB, ASCII", "c8ca123592570c1fcb138d4ec08f7af14ad49363245be1ac25029c8ffc508b3217e75faaa5566426180fec8f",
			"The quick brown fox jumps over the lazy dog.",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "CFB", "Hex", "Raw"),
		},
		{
			"Blowfish Decrypt: CFB, Binary", "e6ac1324d1576beab00e855de3f4ac1f5e3cbf89f4c2a743a5737895067ac5012e5bdb92477e256cc07bf691b58e721179b550e694abb0be7cbdc42586db755bf795f4338f47d356c57453afa6277e46aaeb3405f9744654a477f06c2ad92ede90555759",
			"7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "CFB", "Hex", "Hex"),
		},
		{
			"Blowfish Encrypt: OFB, ASCII", "The quick brown fox jumps over the lazy dog.",
			"c8ca123592570c1fffcee88b9823b9450dc9c48e559123c1df1984214212bae7e44114d29dba79683d10cce5",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "OFB", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt: OFB, Binary", "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			"e6ac1324d1576bea4ceb5be7691c35e4919f18be06cc2a926025ef0973222e987de7c63cd71ed3b19190ba006931d9cbdf412f5b1ac7155904ca591f693fe11aa996e17866e0de4b2eb7ff5effabf94b0f49ed159202caf72745ac2f024d86f942d83767",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "ffeeddccbbaa9988", "OFB", "Hex", "Hex"),
		},
		{
			"Blowfish Decrypt: OFB, ASCII", "c8ca123592570c1fffcee88b9823b9450dc9c48e559123c1df1984214212bae7e44114d29dba79683d10cce5",
			"The quick brown fox jumps over the lazy dog.",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "OFB", "Hex", "Raw"),
		},
		{
			"Blowfish Decrypt: OFB, Binary", "e6ac1324d1576bea4ceb5be7691c35e4919f18be06cc2a926025ef0973222e987de7c63cd71ed3b19190ba006931d9cbdf412f5b1ac7155904ca591f693fe11aa996e17866e0de4b2eb7ff5effabf94b0f49ed159202caf72745ac2f024d86f942d83767",
			"7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "ffeeddccbbaa9988", "OFB", "Hex", "Hex"),
		},
		{
			"Blowfish Encrypt: CTR, ASCII", "The quick brown fox jumps over the lazy dog.",
			"e2a5e0f03ad4877101c7cf83861ad93477adb57acac4bebc315a7bae34b4e6a54e5532db457a3131dcd9dda6",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "0000000000000000", "CTR", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt: CTR, Binary", "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			"ccc3e1e179d4e084b2e27cef77255595ebfb694a9999b7ef8e661086058472dad7f3e0350fde9be87059ab43d5b800aa08be4c00f3f2e99402fe2702c39e8663dbcbb146700d63432227f1045f116bfd4b65022ca20b70427ddcfd7441cb3c75f4d3fff0",
			blowfishRec("Blowfish Encrypt", "0011223344556677", "0000000000000000", "CTR", "Hex", "Hex"),
		},
		{
			"Blowfish Decrypt: CTR, ASCII", "e2a5e0f03ad4877101c7cf83861ad93477adb57acac4bebc315a7bae34b4e6a54e5532db457a3131dcd9dda6",
			"The quick brown fox jumps over the lazy dog.",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "0000000000000000", "CTR", "Hex", "Raw"),
		},
		{
			"Blowfish Decrypt: CTR, Binary", "ccc3e1e179d4e084b2e27cef77255595ebfb694a9999b7ef8e661086058472dad7f3e0350fde9be87059ab43d5b800aa08be4c00f3f2e99402fe2702c39e8663dbcbb146700d63432227f1045f116bfd4b65022ca20b70427ddcfd7441cb3c75f4d3fff0",
			"7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018",
			blowfishRec("Blowfish Decrypt", "0011223344556677", "0000000000000000", "CTR", "Hex", "Hex"),
		},
		{
			"Blowfish Encrypt with variable key length: CBC, ASCII, 4 bytes", "The quick brown fox jumps over the lazy dog.",
			"823f337a53ecf121aa9ec1b111bd5064d1d7586abbdaaa0c8fd0c6cc43c831c88bf088ee3e07287e3f36cf2e45f9c7e6",
			blowfishRec("Blowfish Encrypt", "00112233", "0000000000000000", "CBC", "Raw", "Hex"),
		},
		{
			"Blowfish Encrypt with variable key length: CBC, ASCII, 42 bytes", "The quick brown fox jumps over the lazy dog.",
			"19f5a68145b34321cfba72226b0f33922ce44dd6e7869fe328db64faae156471216f12ed2a37fd0bdd7cebf867b3cff0",
			blowfishRec("Blowfish Encrypt", "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdead", "0000000000000000", "CBC", "Raw", "Hex"),
		},
	})
}

func TestBlowfishErrors(t *testing.T) {
	ts := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Hex"} }
	t.Run("empty key", func(t *testing.T) {
		_, err := runOp(t, "Blowfish Encrypt", "x", ts(""), ts("0000000000000000"), "CBC", "Raw", "Hex")
		want := "Invalid key length: 0 bytes\n\nBlowfish's key length needs to be between 4 and 56 bytes (32-448 bits)."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v\nwant %q", err, want)
		}
	})
	t.Run("key too short (2 bytes)", func(t *testing.T) {
		_, err := runOp(t, "Blowfish Encrypt", "x", ts("0011"), ts("0000000000000000"), "CBC", "Raw", "Hex")
		want := "Invalid key length: 2 bytes\n\nBlowfish's key length needs to be between 4 and 56 bytes (32-448 bits)."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v\nwant %q", err, want)
		}
	})
	t.Run("key too long (57 bytes)", func(t *testing.T) {
		key := strings.Repeat("aa", 57)
		_, err := runOp(t, "Blowfish Encrypt", "x", ts(key), ts("0000000000000000"), "CBC", "Raw", "Hex")
		want := "Invalid key length: 57 bytes\n\nBlowfish's key length needs to be between 4 and 56 bytes (32-448 bits)."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v\nwant %q", err, want)
		}
	})
	t.Run("bad IV length for CBC", func(t *testing.T) {
		_, err := runOp(t, "Blowfish Encrypt", "x", ts("0011223344556677"), ts("00"), "CBC", "Raw", "Hex")
		want := "Invalid IV length: 1 bytes. Expected 8 bytes."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v\nwant %q", err, want)
		}
	})
	t.Run("ECB ignores empty IV", func(t *testing.T) {
		if _, err := runOp(t, "Blowfish Encrypt", "x", ts("0011223344556677"), ts(""), "ECB", "Raw", "Hex"); err != nil {
			t.Fatalf("ECB with empty IV should not error: %v", err)
		}
	})
	t.Run("decrypt with bad padding", func(t *testing.T) {
		// Valid-length ciphertext (8 bytes) that does not decrypt to valid PKCS#7.
		_, err := runOp(t, "Blowfish Decrypt", "ffffffffffffffff", ts("0011223344556677"), ts("0000000000000000"), "CBC", "Hex", "Raw")
		want := "Unable to decrypt input with these parameters."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v\nwant %q", err, want)
		}
	})
	// CBC/ECB decrypt of a non-block-multiple length fails rather than panicking.
	for _, mode := range []string{"CBC", "ECB"} {
		t.Run("decrypt non-block-multiple "+mode, func(t *testing.T) {
			_, err := runOp(t, "Blowfish Decrypt", "0011223344", ts("0011223344556677"), ts("0000000000000000"), mode, "Hex", "Raw")
			want := "Unable to decrypt input with these parameters."
			if err == nil || err.Error() != want {
				t.Fatalf("got %v\nwant %q", err, want)
			}
		})
	}
}

// TestBlowfishArgDecodeErrors covers undecodable Key/IV (invalid Base64).
func TestBlowfishArgDecodeErrors(t *testing.T) {
	hexKey := core.ToggleString{Value: "0011223344556677", Option: "Hex"}
	hexIV := core.ToggleString{Value: "0000000000000000", Option: "Hex"}
	bad := core.ToggleString{Value: "@@@", Option: "Base64"}

	if _, err := runOp(t, "Blowfish Encrypt", "x", bad, hexIV, "CBC", "Raw", "Hex"); err == nil {
		t.Fatal("expected error for undecodable key")
	}
	if _, err := runOp(t, "Blowfish Encrypt", "x", hexKey, bad, "CBC", "Raw", "Hex"); err == nil {
		t.Fatal("expected error for undecodable IV")
	}
	// Decrypt shares the same validation path (empty key rejected).
	empty := core.ToggleString{Value: "", Option: "Hex"}
	if _, err := runOp(t, "Blowfish Decrypt", "00", empty, hexIV, "CBC", "Hex", "Raw"); err == nil {
		t.Fatal("expected error for empty key on decrypt")
	}
}

// TestBlowfishCipherError covers the (normally unreachable) cipher-construction
// error branch via the seam.
func TestBlowfishCipherError(t *testing.T) {
	orig := blowfishNewCipher
	defer func() { blowfishNewCipher = orig }()
	blowfishNewCipher = func([]byte) (cipher.Block, error) { return nil, errors.New("boom") }

	ts := func(v string) core.ToggleString { return core.ToggleString{Value: v, Option: "Hex"} }
	for _, op := range []string{"Blowfish Encrypt", "Blowfish Decrypt"} {
		if _, err := runOp(t, op, "x", ts("0011223344556677"), ts("0000000000000000"), "CBC", "Raw", "Hex"); err == nil {
			t.Fatalf("%s: expected cipher error to surface", op)
		}
	}
}
