package ops

import (
	"encoding/hex"
	"strings"
	"testing"
)

// TestFingerprintOutputFormats exercises the non-default output branches, the
// Base64/Raw input formats, and an error path across the fingerprint operations.
func TestFingerprintOutputFormats(t *testing.T) {
	ja3in := "16030100a4010000a00301543dd2dd48f517ca9a93b1e599f019fdece704a23e86c1dcac588427abbaddf200005cc014c00a0039003800880087c00fc00500350084c012c00800160013c00dc003000ac013c00900330032009a009900450044c00ec004002f009600410007c011c007c00cc002000500040015001200090014001100080006000300ff0100001b000b000403000102000a000600040018001700230000000f000101"
	ja3Hash := "503053a0c5b2bd9b9334bf7f3d3b8852"
	ja3Str := "769,49172-49162-57-56-136-135-49167-49157-53-132-49170-49160-22-19-49165-49155-10-49171-49161-51-50-154-153-69-68-49166-49156-47-150-65-7-49169-49159-49164-49154-5-4-21-18-9-20-17-8-6-3-255,11-10-35-15,24-23,0-1-2"

	// JA3 string output.
	if out, err := runOp(t, "JA3 Fingerprint", ja3in, "Hex", "JA3 string"); err != nil || out != ja3Str {
		t.Errorf("JA3 string: got %q err %v", out, err)
	}
	// JA3 Full details contains the hash and the JA3 string.
	if out, err := runOp(t, "JA3 Fingerprint", ja3in, "Hex", "Full details"); err != nil ||
		!strings.Contains(out, ja3Hash) || !strings.Contains(out, ja3Str) {
		t.Errorf("JA3 full details: %q err %v", out, err)
	}
	// Base64 input format.
	raw, _ := hex.DecodeString(ja3in)
	b64 := toBase64(raw, "A-Za-z0-9+/=")
	if out, err := runOp(t, "JA3 Fingerprint", b64, "Base64", "Hash digest"); err != nil || out != ja3Hash {
		t.Errorf("JA3 base64: got %q err %v", out, err)
	}
	// Raw input format (raw bytes passed as a string).
	if out, err := runOp(t, "JA3 Fingerprint", string(raw), "Raw", "Hash digest"); err != nil || out != ja3Hash {
		t.Errorf("JA3 raw: got %q err %v", out, err)
	}
	// Error path: non-handshake data.
	if _, err := runOp(t, "JA3 Fingerprint", "0000", "Hex", "Hash digest"); err == nil {
		t.Error("JA3 expected error for non-handshake data")
	}

	// JA3S Full details contains the hash.
	ja3sIn := "160301003d020000390301543dd2ddedbfe33895bd6bc676a3fa6b9fe5773a6e04d5476d1af3bcbc1dcbbb00c011000011ff01000100000b00040300010200230000"
	if out, err := runOp(t, "JA3S Fingerprint", ja3sIn, "Hex", "Full details"); err != nil ||
		!strings.Contains(out, "bed95e1b525d2f41db3a6d68fac5b566") {
		t.Errorf("JA3S full details: %q err %v", out, err)
	}
	if out, err := runOp(t, "JA3S Fingerprint", ja3sIn, "Hex", "JA3S string"); err != nil || out != "769,49169,65281-11-35" {
		t.Errorf("JA3S string: %q err %v", out, err)
	}

	// HASSH client/server Full details.
	if out, err := runOp(t, "HASSH Client Fingerprint", hasshClientKEXINIT, "Hex", "Full details"); err != nil ||
		!strings.Contains(out, "6559ab006495e3044da5a8821704047e") {
		t.Errorf("HASSH client full details: %q err %v", out, err)
	}
	if out, err := runOp(t, "HASSH Server Fingerprint", hasshServerKEXINIT, "Hex", "Full details"); err != nil ||
		!strings.Contains(out, "95dc0a7fbfb0eb627394c0e4240dc213") {
		t.Errorf("HASSH server full details: %q err %v", out, err)
	}

	// JA4 "All" and JA4S "Both".
	ja4in := "1603010200010001fc0303b2c03e7ba990ef540c316a665d4d925f8e9079ac4b15687e587dc99016e75a6c20d0b0099243c9296a0c84153ea4ada7d87ad017f4211c2ea1350b0b3cc5514d5f00205a5a130113021303c02bc02fc02cc030cca9cca8c013c014009c009d002f003501000193fafa000000000024002200001f636f6e74656e742d6175746f66696c6c2e676f6f676c65617069732e636f6d0033002b00293a3a000100001d0020fb2cd8ef3d605b96ab03119ec4f30a6e2088cb1af86c41a81feace8706068c50000d001200100403080404010503080505010806060100230000000b00020100ff01000100000a000a00083a3a001d00170018001b000302000244690005000302683200120000002d000201010010000e000c02683208687474702f312e31000500050100000000002b0007060a0a03040303001700001a1a000100001500b800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	if out, err := runOp(t, "JA4 Fingerprint", ja4in, "Hex", "All"); err != nil ||
		!strings.Contains(out, "t13d1516h2_8daaf6152771_e5627efa2ab1") {
		t.Errorf("JA4 all: %q err %v", out, err)
	}
	// JA4 Raw Original Rendering (full unsorted, unhashed rendering).
	wantRO := "t13d1516h2_1301,1302,1303,c02b,c02f,c02c,c030,cca9,cca8,c013,c014,009c,009d,002f,0035_0000,0033,000d,0023,000b,ff01,000a,001b,4469,0012,002d,0010,0005,002b,0017,0015_0403,0804,0401,0503,0805,0501,0806,0601"
	if out, err := runOp(t, "JA4 Fingerprint", ja4in, "Hex", "JA4 Raw Original Rendering"); err != nil || out != wantRO {
		t.Errorf("JA4 ro: got %q err %v", out, err)
	}
	ja4sIn := "16030300640200006003035f0236c07f47bfb12dc2da706ecb3fe7f9eeac9968cc2ddf444f574e4752440120b89ff1ab695278c69b8a73f76242ef755e0b13dc6d459aaaa784fec9c2dfce34cca900001800000000ff01000100000b00020100001000050003026832"
	if out, err := runOp(t, "JA4Server Fingerprint", ja4sIn, "Hex", "Both"); err != nil ||
		!strings.Contains(out, "t1204h2_cca9_1428ce7b4018") {
		t.Errorf("JA4S both: %q err %v", out, err)
	}
}

// TestFingerprintErrors covers the parse/validation error branches, including
// handshake-type mismatches (feeding a client op a server hello and vice versa).
func TestFingerprintErrors(t *testing.T) {
	// Non-handshake data errors for every fingerprint op.
	for _, c := range []struct{ op, out string }{
		{"JA3 Fingerprint", "Hash digest"},
		{"JA3S Fingerprint", "Hash digest"},
		{"HASSH Client Fingerprint", "Hash digest"},
		{"HASSH Server Fingerprint", "Hash digest"},
		{"JA4 Fingerprint", "JA4"},
		{"JA4Server Fingerprint", "JA4S"},
	} {
		if _, err := runOp(t, c.op, "0000", "Hex", c.out); err == nil {
			t.Errorf("%s: expected error for non-handshake input", c.op)
		}
	}

	clientHello := "16030100a4010000a00301543dd2dd48f517ca9a93b1e599f019fdece704a23e86c1dcac588427abbaddf200005cc014c00a0039003800880087c00fc00500350084c012c00800160013c00dc003000ac013c00900330032009a009900450044c00ec004002f009600410007c011c007c00cc002000500040015001200090014001100080006000300ff0100001b000b000403000102000a000600040018001700230000000f000101"
	serverHello := "160301003d020000390301543dd2ddedbfe33895bd6bc676a3fa6b9fe5773a6e04d5476d1af3bcbc1dcbbb00c011000011ff01000100000b00040300010200230000"

	// Client op fed a Server Hello, and server op fed a Client Hello, must error.
	if _, err := runOp(t, "JA3 Fingerprint", serverHello, "Hex", "Hash digest"); err == nil {
		t.Error("JA3 on server hello: expected error")
	}
	if _, err := runOp(t, "JA3S Fingerprint", clientHello, "Hex", "Hash digest"); err == nil {
		t.Error("JA3S on client hello: expected error")
	}
	if _, err := runOp(t, "JA4 Fingerprint", serverHello, "Hex", "JA4"); err == nil {
		t.Error("JA4 on server hello: expected error")
	}
	if _, err := runOp(t, "JA4Server Fingerprint", clientHello, "Hex", "JA4S"); err == nil {
		t.Error("JA4Server on client hello: expected error")
	}
}

// TestJA4Helpers unit-tests the pure JA4 helpers across their edge cases.
func TestJA4Helpers(t *testing.T) {
	for v, want := range map[int]string{
		0x0304: "13", 0x0303: "12", 0x0302: "11", 0x0301: "10",
		0x0300: "s3", 0x0200: "s2", 0x0100: "s1", 0x9999: "00",
	} {
		if got := tlsVersionMapper(v); got != want {
			t.Errorf("tlsVersionMapper(%#x) = %q, want %q", v, got, want)
		}
	}

	// ALPN: alphanumeric first/last -> those chars; otherwise hex fallback; empty -> "00".
	if got := alpnFingerprint([]byte("h2")); got != "h2" {
		t.Errorf("alpn h2: %q", got)
	}
	if got := alpnFingerprint([]byte{0x02, 0x1f}); got != "0f" {
		t.Errorf("alpn non-alnum: %q, want 0f", got)
	}
	if got := alpnFingerprint(nil); got != "00" {
		t.Errorf("alpn empty: %q", got)
	}

	// parseFirstALPNValue: too-short inputs return nil.
	if v := parseFirstALPNValue([]byte{0x00, 0x01}); v != nil {
		t.Errorf("alpn ext len < 2: %v", v)
	}
	if v := parseFirstALPNValue([]byte{0x00, 0x02, 0x00}); v != nil {
		t.Errorf("alpn str len < 1: %v", v)
	}
}
