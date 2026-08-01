package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// rabbitRecipe builds a single-op Rabbit recipe with Hex-encoded key/IV.
func rabbitRecipe(key, iv, endian, inType, outType string) core.Recipe {
	return core.Recipe{{Op: "Rabbit", Args: []any{
		core.ToggleString{Value: key, Option: "Hex"},
		core.ToggleString{Value: iv, Option: "Hex"},
		endian, inType, outType,
	}}}
}

// Rabbit fixtures transcribed from ../CyberChef/tests/operations/tests/Rabbit.mjs.
// The RFC vectors come from RFC 4503.
func TestRabbitFixtures(t *testing.T) {
	const zeros48 = "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	const zeroKey = "00000000000000000000000000000000"
	runCases(t, []opCase{
		{
			"Rabbit: RFC without IV 1", zeros48,
			"b15754f036a5d6ecf56b45261c4af70288e8d815c59c0c397b696c4789c68aa7f416a1c3700cd451da68d1881673d696",
			rabbitRecipe(zeroKey, "", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: RFC without IV 2", zeros48,
			"3d2df3c83ef627a1e97fc38487e2519cf576cd61f4405b8896bf53aa8554fc19e5547473fbdb43508ae53b20204d4c5e",
			rabbitRecipe("912813292e3d36fe3bfc62f1dc51c3ac", "", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: RFC without IV 3", zeros48,
			"0cb10dcda041cdac32eb5cfd02d0609b95fc9fca0f17015a7b7092114cff3ead9649e5de8bfc7f3f924147ad3a947428",
			rabbitRecipe("8395741587e0c733e9e9ab01c09b0043", "", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: RFC with IV 1", zeros48,
			"c6a7275ef85495d87ccd5d376705b7ed5f29a6ac04f5efd47b8f293270dc4a8d2ade822b29de6c1ee52bdb8a47bf8f66",
			rabbitRecipe(zeroKey, "0000000000000000", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: RFC with IV 2", zeros48,
			"1fcd4eb9580012e2e0dccc9222017d6da75f4e10d12125017b2499ffed936f2eebc112c393e738392356bdd012029ba7",
			rabbitRecipe(zeroKey, "c373f575c1267e59", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: RFC with IV 3", zeros48,
			"445ad8c805858dbf70b6af23a151104d96c8f27947f42c5baeae67c6acc35b039fcbfc895fa71c17313df034f01551cb",
			rabbitRecipe(zeroKey, "a6eb561ad2f41727", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: XOR with input",
			"cedda96c054e3ddd93da7ed05e2a4b7bdb0c00fe214f03502e2708b2c2bfc77aa2311b0b9af8aa78d119f92b26db0a6b",
			"7f8afd9c33ebeb3166b13bf64260bc7953e4d8ebe4d30f69554e64f54b794ddd5627bac8eaf47e290b7128a330a8dcfd",
			rabbitRecipe(zeroKey, "", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: LSBs used for last block", "0000000000000000", "f56b45261c4af702",
			rabbitRecipe(zeroKey, "", "Big", "Hex", "Hex"),
		},
		{
			"Rabbit: little-endian (Crypto++)", "Rabbit stream cipher test",
			"1ae2d4edcf9b6063b00fd6fda0b223aded157e77031cf0440b",
			rabbitRecipe("23c2731e8b5469fd8dabb5bc592a0f3a", "712906405ef03201", "Little", "Raw", "Hex"),
		},
	})
}

// TestRabbitErrors covers the key- and IV-length validation messages.
func TestRabbitErrors(t *testing.T) {
	cases := []struct {
		name, key, iv, sub string
	}{
		{"invalid key length", "0000000000000000", "", "Invalid key length: 8 bytes (expected: 16)"},
		{"invalid IV length", "00000000000000000000000000000000", "00000000", "Invalid IV length: 4 bytes (expected: 0 or 8)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Rabbit", "",
				core.ToggleString{Value: c.key, Option: "Hex"},
				core.ToggleString{Value: c.iv, Option: "Hex"},
				"Big", "Hex", "Hex")
			if err == nil || !strings.Contains(err.Error(), c.sub) {
				t.Fatalf("got %v, want %q", err, c.sub)
			}
		})
	}
}

// TestRabbitDecodeErrors covers the key/IV Base64 decode error paths.
func TestRabbitDecodeErrors(t *testing.T) {
	// Invalid Base64 key.
	if _, err := runOp(t, "Rabbit", "",
		core.ToggleString{Value: "!!!not base64!!!", Option: "Base64"},
		core.ToggleString{Value: "", Option: "Hex"},
		"Big", "Hex", "Hex"); err == nil {
		t.Fatal("bad base64 key should error")
	}
	// Invalid Base64 IV (key is valid so the IV decode is reached).
	if _, err := runOp(t, "Rabbit", "",
		core.ToggleString{Value: "00000000000000000000000000000000", Option: "Hex"},
		core.ToggleString{Value: "!!bad!!", Option: "Base64"},
		"Big", "Hex", "Hex"); err == nil {
		t.Fatal("bad base64 IV should error")
	}
}
