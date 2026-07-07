package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// PEM fixtures transcribed from CyberChef tests/operations/tests/PEMtoHex.mjs.
const (
	pemFOO = "-----BEGIN FOO-----\nRk9P\n-----END FOO-----"
	pemBAR = "-----BEGIN BAR-----\nQkFS\n-----END BAR-----"

	pemECPrivate = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIhdQxQIcMnCHD3X4WqNv+VgycWmFoEZpRl9X0+dT9uHoAoGCCqGSM49
AwEHoUQDQgAEFLQcBbzDweo6af4k3k0gKWMNWOZVn8+9hH2rv4DKKYZ7E1z64LBt
PnB1gMz++HDKySr2ozD3/46dIbQMXUZKpw==
-----END EC PRIVATE KEY-----`

	pemECPrivatePKCS8 = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgiF1DFAhwycIcPdfh
ao2/5WDJxaYWgRmlGX1fT51P24ehRANCAAQUtBwFvMPB6jpp/iTeTSApYw1Y5lWf
z72Efau/gMophnsTXPrgsG0+cHWAzP74cMrJKvajMPf/jp0htAxdRkqn
-----END PRIVATE KEY-----`

	pemECPublic = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFLQcBbzDweo6af4k3k0gKWMNWOZV
n8+9hH2rv4DKKYZ7E1z64LBtPnB1gMz++HDKySr2ozD3/46dIbQMXUZKpw==
-----END PUBLIC KEY-----`

	ecPublicHex = "3059301306072a8648ce3d020106082a8648ce3d0301070342000414b41c05bcc3c1ea3a69fe24de4d2029630d58e6559fcfbd847dabbf80ca29867b135cfae0b06d3e707580ccfef870cac92af6a330f7ff8e9d21b40c5d464aa7"
)

// TestPEMToHex transcribes CyberChef's PEMtoHex.mjs cases: each PEM block's
// base64 body is decoded and hex-encoded, multiple blocks are joined with "\n".
func TestPEMToHex(t *testing.T) {
	pth := core.Recipe{{Op: "PEM to Hex", Args: []any{}}}
	runCases(t, []opCase{
		{"nothing", "", "", pth},
		{
			"multiple PEMs", pemFOO + "\n" + pemBAR, "FOOBAR",
			core.Recipe{{Op: "PEM to Hex"}, {Op: "From Hex", Args: []any{"Auto"}}},
		},
		{
			// A PEM collapsed onto a single line still decodes.
			"single line PEM", "-----BEGIN FOO-----Rk9P-----END FOO-----", "FOO",
			core.Recipe{{Op: "PEM to Hex"}, {Op: "From Hex", Args: []any{"None"}}},
		},
		{
			"EC P-256 private key", pemECPrivate,
			"30770201010420885d43140870c9c21c3dd7e16a8dbfe560c9c5a6168119a5197d5f4f9d4fdb87a00a06082a8648ce3d030107a1440342000414b41c05bcc3c1ea3a69fe24de4d2029630d58e6559fcfbd847dabbf80ca29867b135cfae0b06d3e707580ccfef870cac92af6a330f7ff8e9d21b40c5d464aa7",
			pth,
		},
		{
			"EC P-256 private key PKCS8", pemECPrivatePKCS8,
			"308187020100301306072a8648ce3d020106082a8648ce3d030107046d306b0201010420885d43140870c9c21c3dd7e16a8dbfe560c9c5a6168119a5197d5f4f9d4fdb87a1440342000414b41c05bcc3c1ea3a69fe24de4d2029630d58e6559fcfbd847dabbf80ca29867b135cfae0b06d3e707580ccfef870cac92af6a330f7ff8e9d21b40c5d464aa7",
			pth,
		},
		{
			"EC P-256 public key", pemECPublic, ecPublicHex, pth,
		},
		{
			// A malformed (4n+1) base64 body decodes to nothing rather than
			// erroring, matching CyberChef's lenient (non-strict) fromBase64.
			"short base64 body", "-----BEGIN FOO-----\nR\n-----END FOO-----", "", pth,
		},
	})
}

// TestPEMToHexNoFooter covers the missing-footer error (CyberChef throws
// "PEM footer '...' not found").
func TestPEMToHexNoFooter(t *testing.T) {
	if _, err := runOp(t, "PEM to Hex", "-----BEGIN FOO-----\nRk9P"); err == nil {
		t.Fatal("expected an error when the PEM footer is missing")
	}
}

// TestHexToPEM wraps hex DER as PEM. Outputs are oracle-verified (CyberChef uses
// jsrsasign, which emits CRLF line endings and a trailing CRLF).
func TestHexToPEM(t *testing.T) {
	runCases(t, []opCase{
		{
			"short DER", "3003010100", "-----BEGIN CERTIFICATE-----\r\nMAMBAQA=\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
		{
			"custom header", "3003010100", "-----BEGIN PUBLIC KEY-----\r\nMAMBAQA=\r\n-----END PUBLIC KEY-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"PUBLIC KEY"}}},
		},
		{
			// Whitespace in the hex input is stripped before decoding.
			"whitespace input", "30 03 01 01 00", "-----BEGIN CERTIFICATE-----\r\nMAMBAQA=\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
		{
			// A DER whose base64 exceeds 64 chars wraps at 64 with CRLF.
			"multi-line body", ecPublicHex,
			"-----BEGIN PUBLIC KEY-----\r\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFLQcBbzDweo6af4k3k0gKWMNWOZV\r\nn8+9hH2rv4DKKYZ7E1z64LBtPnB1gMz++HDKySr2ozD3/46dIbQMXUZKpw==\r\n-----END PUBLIC KEY-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"PUBLIC KEY"}}},
		},
		{
			// Odd-length hex keeps the trailing nibble (jsrsasign hextob64).
			"odd-length hex", "303", "-----BEGIN CERTIFICATE-----\r\nMD==\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
		{
			// Uppercase hex digits are accepted.
			"uppercase hex", "AB3003", "-----BEGIN CERTIFICATE-----\r\nqzAD\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
		{
			// Non-hex characters are parsed leniently per 2-char group (jsrsasign
			// parseInt: "3g" -> 0x03), not rejected.
			"non-hex lenient partial", "3g", "-----BEGIN CERTIFICATE-----\r\nAw==\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
		{
			// A group with no leading hex digit is NaN -> 0x00.
			"non-hex lenient zero", "zz", "-----BEGIN CERTIFICATE-----\r\nAA==\r\n-----END CERTIFICATE-----\r\n",
			core.Recipe{{Op: "Hex to PEM", Args: []any{"CERTIFICATE"}}},
		},
	})
}
