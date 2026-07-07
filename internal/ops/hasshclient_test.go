package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// HASSH Client fingerprint verified against the CyberChef-server oracle (no
// upstream fixtures). Input is a synthesised SSH_MSG_KEXINIT client packet.
const hasshClientKEXINIT = "0000018404140000000000000000000000000000000000000042637572766532353531392d7368613235362c656364682d736861322d6e697374703235362c6469666669652d68656c6c6d616e2d67726f757031342d736861323536000000257373682d656432353531392c7273612d736861322d3531322c7273612d736861322d3235360000003363686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d6374722c6165733235362d6374720000003363686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d6374722c6165733235362d63747200000025756d61632d36342d65746d406f70656e7373682e636f6d2c686d61632d736861322d32353600000025756d61632d36342d65746d406f70656e7373682e636f6d2c686d61632d736861322d323536000000156e6f6e652c7a6c6962406f70656e7373682e636f6d000000156e6f6e652c7a6c6962406f70656e7373682e636f6d0000000000000000000000000000000000"

func TestHASSHClientFingerprint(t *testing.T) {
	runCases(t, []opCase{
		{
			"HASSH client hash", hasshClientKEXINIT, "6559ab006495e3044da5a8821704047e",
			core.Recipe{{Op: "HASSH Client Fingerprint", Args: []any{"Hex", "Hash digest"}}},
		},
		{
			"HASSH client string", hasshClientKEXINIT,
			"curve25519-sha256,ecdh-sha2-nistp256,diffie-hellman-group14-sha256;chacha20-poly1305@openssh.com,aes128-ctr,aes256-ctr;umac-64-etm@openssh.com,hmac-sha2-256;none,zlib@openssh.com",
			core.Recipe{{Op: "HASSH Client Fingerprint", Args: []any{"Hex", "HASSH algorithms string"}}},
		},
	})
}

// TestHASSHClientErrors covers the input-decode error and the not-Key-Exchange
// message-type guard (message byte 0x14 mutated to 0x15).
func TestHASSHClientErrors(t *testing.T) {
	if _, err := runOp(t, "HASSH Client Fingerprint", "A", "Base64", "Hash digest"); err == nil {
		t.Fatal("expected an error for an invalid Base64 input")
	}
	notKEX := hasshClientKEXINIT[:10] + "15" + hasshClientKEXINIT[12:]
	if _, err := runOp(t, "HASSH Client Fingerprint", notKEX, "Hex", "Hash digest"); err == nil {
		t.Fatal("expected an error for a non-KEXINIT message")
	}
}
