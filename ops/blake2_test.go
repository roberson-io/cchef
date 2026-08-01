package ops

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// blake2Recipe builds a BLAKE2b/BLAKE2s recipe with a key of the given encoding.
func blake2Recipe(op, size, format, key, keyOpt string) core.Recipe {
	return core.Recipe{{Op: op, Args: []any{size, format, core.ToggleString{Value: key, Option: keyOpt}}}}
}

// BLAKE2b fixtures transcribed from ../CyberChef/tests/operations/tests/BLAKE2b.mjs.
func TestBLAKE2bFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"BLAKE2b: 512", "Hello World",
			"4386a08a265111c9896f56456e2cb61a64239115c4784cf438e36cc851221972da3fb0115f73cd02486254001f878ab1fd126aac69844ef1c1ca152379d0a9bd",
			blake2Recipe("BLAKE2b", "512", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2b: 384", "Hello World",
			"4d388e82ca8f866e606b6f6f0be910abd62ad6e98c0adfc27cf35acf948986d5c5b9c18b6f47261e1e679eb98edf8e2d",
			blake2Recipe("BLAKE2b", "384", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2b: 256", "Hello World",
			"1dc01772ee0171f5f614c673e3c7fa1107a8cf727bdf5a6dadb379e93c0d1d00",
			blake2Recipe("BLAKE2b", "256", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2b: 160", "Hello World",
			"6a8489e6fd6e51fae12ab271ec7fc8134dd5d737",
			blake2Recipe("BLAKE2b", "160", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2b: Key Test", "message data",
			"3d363ff7401e02026f4a4687d4863ced",
			blake2Recipe("BLAKE2b", "128", "Hex", "pseudorandom key", "UTF8"),
		},
	})
}

// BLAKE2s fixtures transcribed from ../CyberChef/tests/operations/tests/BLAKE2s.mjs.
func TestBLAKE2sFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"BLAKE2s: 256", "Hello World",
			"7706af019148849e516f95ba630307a2018bb7bf03803eca5ed7ed2c3c013513",
			blake2Recipe("BLAKE2s", "256", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2s: 160", "Hello World",
			"0e4fcfc2ee0097ac1d72d70b595a39e09a3c7c7e",
			blake2Recipe("BLAKE2s", "160", "Hex", "", "UTF8"),
		},
		{
			"BLAKE2s: 128", "Hello World",
			"9964ee6f36126626bf864363edfa96f6",
			blake2Recipe("BLAKE2s", "128", "Hex", "", "UTF8"),
		},
		// Keyed and multi-block inputs (oracle-derived) exercise the key block and
		// the block loop of the from-scratch BLAKE2s.
		{
			"BLAKE2s: 256 keyed", "Hello World",
			"14b8a010516e8a58ab945c1b828b8b56903c1e1b2286ea4e79ae67764883d97c",
			blake2Recipe("BLAKE2s", "256", "Hex", "secret", "UTF8"),
		},
		{
			"BLAKE2s: 256 multi-block", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"214f24fe1118eb854450238e11bebe22d2e3937ed85c7c96c6c010106b752ad3",
			blake2Recipe("BLAKE2s", "256", "Hex", "", "UTF8"),
		},
		// The empty-input RFC 7693 test vector (covers the zero-length block path).
		{
			"BLAKE2s: 256 empty", "",
			"69217a3079908094e11121d042354a7c1f55b6482ca1a51e1b250dfd1ed0eef9",
			blake2Recipe("BLAKE2s", "256", "Hex", "", "UTF8"),
		},
	})
}

// The non-Hex output encodings, verified against the CyberChef-server oracle.
func TestBLAKE2OutputEncodings(t *testing.T) {
	runCases(t, []opCase{
		{
			"BLAKE2b: 256 Base64", "Hello World",
			"HcAXcu4BcfX2FMZz48f6EQeoz3J731ptrbN56TwNHQA=",
			blake2Recipe("BLAKE2b", "256", "Base64", "", "UTF8"),
		},
		{
			"BLAKE2s: 256 Base64", "Hello World",
			"dwavAZFIhJ5Rb5W6YwMHogGLt78DgD7KXtftLDwBNRM=",
			blake2Recipe("BLAKE2s", "256", "Base64", "", "UTF8"),
		},
	})
}

// Raw output is the digest bytes decoded as UTF-8 (falling back to Latin-1 for
// the invalid sequences a random digest contains). The expected string is the
// per-byte Latin-1 decoding of the known BLAKE2b-256 digest.
func TestBLAKE2Raw(t *testing.T) {
	digest, _ := hex.DecodeString("1dc01772ee0171f5f614c673e3c7fa1107a8cf727bdf5a6dadb379e93c0d1d00")
	runes := make([]rune, len(digest))
	for i, b := range digest {
		runes[i] = rune(b)
	}
	got, err := runOp(t, "BLAKE2b", "Hello World", "256", "Raw", core.ToggleString{Value: "", Option: "UTF8"})
	if err != nil || got != string(runes) {
		t.Fatalf("Raw output mismatch: got %q, %v", got, err)
	}
}

func TestBLAKE2Errors(t *testing.T) {
	t.Run("BLAKE2b key too long", func(t *testing.T) {
		_, err := runOp(t, "BLAKE2b", "x", "512", "Hex", core.ToggleString{Value: strings.Repeat("a", 65), Option: "UTF8"})
		want := "Key cannot be greater than 64 bytes\nIt is currently 65 bytes."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
	t.Run("BLAKE2s key too long", func(t *testing.T) {
		_, err := runOp(t, "BLAKE2s", "x", "256", "Hex", core.ToggleString{Value: strings.Repeat("a", 33), Option: "UTF8"})
		want := "Key cannot be greater than 32 bytes\nIt is currently 33 bytes."
		if err == nil || err.Error() != want {
			t.Fatalf("got %v, want %q", err, want)
		}
	})
	t.Run("invalid Base64 key", func(t *testing.T) {
		_, err := runOp(t, "BLAKE2b", "x", "512", "Hex", core.ToggleString{Value: "!!!not base64", Option: "Base64"})
		if err == nil {
			t.Fatal("want error for invalid Base64 key")
		}
	})
}
