package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// TestObjectIdentifierToHex verifies dotted-decimal OID -> ASN.1 content-octet
// hex, with vectors captured from the CyberChef-server oracle.
func TestObjectIdentifierToHex(t *testing.T) {
	one := func(name, in, want string) opCase {
		return opCase{name, in, want, core.Recipe{{Op: "Object Identifier to Hex"}}}
	}
	runCases(t, []opCase{
		one("RSA encryption", "1.2.840.113549.1.1.1", "2a864886f70d010101"),
		one("EC public key", "1.2.840.10045.2.1", "2a8648ce3d0201"),
		one("commonName", "2.5.4.3", "550403"),
		one("large arcs", "1.3.6.1.4.1.11129.2.4.2", "2b06010401d679020402"),
		one("extKeyUsage", "2.5.29.37", "551d25"),
		one("UID (very large arc)", "0.9.2342.19200300.100.1.25", "0992268993f22c640119"),
		one("emailAddress", "1.2.840.113549.1.9.1", "2a864886f70d010901"),
		one("sha256", "2.16.840.1.101.3.4.2.1", "608648016503040201"),
		one("two arcs", "1.2", "2a"),
		// jsrsasign quirk: a single arc has no second sub-identifier, so
		// parseInt(undefined) -> NaN -> "NaN".
		one("single arc quirk", "1", "NaN"),
	})
}

// TestHexToObjectIdentifier verifies ASN.1 content-octet hex -> dotted-decimal
// OID (reusing the shared jsrsasign-ported decoder), including whitespace
// stripping and the fact that the whole input is treated as content octets.
func TestHexToObjectIdentifier(t *testing.T) {
	one := func(name, in, want string) opCase {
		return opCase{name, in, want, core.Recipe{{Op: "Hex to Object Identifier"}}}
	}
	runCases(t, []opCase{
		one("RSA encryption", "2a864886f70d010101", "1.2.840.113549.1.1.1"),
		one("EC public key", "2a8648ce3d0201", "1.2.840.10045.2.1"),
		one("commonName", "550403", "2.5.4.3"),
		one("large arcs", "2b06010401d679020402", "1.3.6.1.4.1.11129.2.4.2"),
		one("whitespace stripped", "55 04 03", "2.5.4.3"),
		one("two arcs", "2a", "1.2"),
		// The input is decoded as raw content octets: a full TLV (tag 06, len 08)
		// is decoded byte-for-byte too.
		one("full TLV decoded raw", "06082a8648ce3d030107", "0.6.8.42.840.10045.3.1.7"),
	})
}

// TestObjectIdentifierToHexErrors covers the malformed-input error.
func TestObjectIdentifierToHexErrors(t *testing.T) {
	for _, c := range []struct{ name, in, wantErr string }{
		{"non-numeric arc", "1.2.abc", "malformed oid string: 1.2.abc"},
		{"letters", "abc", "malformed oid string: abc"},
		{"spaces rejected", "1.2 3", "malformed oid string: 1.2 3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Object Identifier to Hex", c.in)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v, want %q", err, c.wantErr)
			}
		})
	}
}
