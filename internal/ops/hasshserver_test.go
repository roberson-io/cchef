package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// HASSH Server fingerprint verified against the CyberChef-server oracle (no
// upstream fixtures). Input is a synthesised SSH_MSG_KEXINIT server packet.
const hasshServerKEXINIT = "0000012206141111111111111111111111111111111100000024637572766532353531392d7368613235362c656364682d736861322d6e69737470323536000000187373682d656432353531392c7273612d736861322d353132000000156165733132382d6374722c6165733235362d6374720000003463686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d67636d406f70656e7373682e636f6d0000000d686d61632d736861322d32353600000032686d61632d736861322d3531322d65746d406f70656e7373682e636f6d2c756d61632d313238406f70656e7373682e636f6d000000046e6f6e65000000156e6f6e652c7a6c6962406f70656e7373682e636f6d00000000000000000000000000000000000000"

func TestHASSHServerFingerprint(t *testing.T) {
	runCases(t, []opCase{
		{
			"HASSH server hash", hasshServerKEXINIT, "95dc0a7fbfb0eb627394c0e4240dc213",
			core.Recipe{{Op: "HASSH Server Fingerprint", Args: []any{"Hex", "Hash digest"}}},
		},
		{
			"HASSH server string", hasshServerKEXINIT,
			"curve25519-sha256,ecdh-sha2-nistp256;chacha20-poly1305@openssh.com,aes128-gcm@openssh.com;hmac-sha2-512-etm@openssh.com,umac-128@openssh.com;none,zlib@openssh.com",
			core.Recipe{{Op: "HASSH Server Fingerprint", Args: []any{"Hex", "HASSH algorithms string"}}},
		},
	})
}

// TestHASSHServerErrors covers the input-decode error and the not-Key-Exchange
// message-type guard (message byte 0x14 mutated to 0x15).
func TestHASSHServerErrors(t *testing.T) {
	if _, err := runOp(t, "HASSH Server Fingerprint", "A", "Base64", "Hash digest"); err == nil {
		t.Fatal("expected an error for an invalid Base64 input")
	}
	notKEX := hasshServerKEXINIT[:10] + "15" + hasshServerKEXINIT[12:]
	if _, err := runOp(t, "HASSH Server Fingerprint", notKEX, "Hex", "Hash digest"); err == nil {
		t.Fatal("expected an error for a non-KEXINIT message")
	}
}
