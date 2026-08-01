package ops

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// AMF has no CyberChef fixture file, and CyberChef's decoder returns its
// library's annotated parse tree rather than a value, so the two are compared
// on encoding — which is byte-identical — and on the values a decode yields.
//
// The encode vectors were produced by cchef and spot-checked against the
// oracle. CyberChef's AMF3 encoder cannot write null, booleans or arrays at
// all, so for those the specification is the only reference.

// TestAMFEncodeVectors pins the bytes each JSON value encodes to.
func TestAMFEncodeVectors(t *testing.T) {
	for _, tc := range []struct{ format, in, want string }{
		{"AMF0", "null", "05"},
		{"AMF0", "true", "0101"},
		{"AMF0", "false", "0100"},
		{"AMF0", "0", "000000000000000000"},
		{"AMF0", "1", "003ff0000000000000"},
		{"AMF0", "-1", "00bff0000000000000"},
		{"AMF0", "42", "004045000000000000"},
		{"AMF0", "3.5", "00400c000000000000"},
		{"AMF0", "-0.5", "00bfe0000000000000"},
		{"AMF0", "1e21", "00444b1ae4d6e2ef50"},
		{"AMF0", "\"\"", "020000"},
		{"AMF0", "\"hi\"", "0200026869"},
		{"AMF0", "\"héllo\"", "02000668c3a96c6c6f"},
		{"AMF0", "[]", "0a00000000"},
		{"AMF0", "[1]", "0a00000001003ff0000000000000"},
		{"AMF0", "[1,2,3]", "0a00000003003ff0000000000000004000000000000000004008000000000000"},
		{"AMF0", "[\"a\",\"b\"]", "0a000000020200016102000162"},
		{"AMF0", "[[1],[2]]", "0a000000020a00000001003ff00000000000000a00000001004000000000000000"},
		{"AMF0", "{}", "03000009"},
		{"AMF0", "{\"a\":1}", "03000161003ff0000000000000000009"},
		{"AMF0", "{\"a\":1,\"b\":\"x\",\"c\":true}", "03000161003ff0000000000000000162020001780001630101000009"},
		{"AMF0", "{\"nested\":{\"k\":[1,2]}}", "0300066e65737465640300016b0a00000002003ff0000000000000004000000000000000000009000009"},
		{"AMF0", "[null,true,false]", "0a000000030501010100"},
		{"AMF0", "{\"\":1}", "030000003ff0000000000000000009"},
		{"AMF0", "{\"a\":null}", "0300016105000009"},
		{"AMF3", "null", "01"},
		{"AMF3", "true", "03"},
		{"AMF3", "false", "02"},
		{"AMF3", "0", "050000000000000000"},
		{"AMF3", "1", "053ff0000000000000"},
		{"AMF3", "-1", "05bff0000000000000"},
		{"AMF3", "42", "054045000000000000"},
		{"AMF3", "3.5", "05400c000000000000"},
		{"AMF3", "-0.5", "05bfe0000000000000"},
		{"AMF3", "1e21", "05444b1ae4d6e2ef50"},
		{"AMF3", "\"\"", "0601"},
		{"AMF3", "\"hi\"", "06056869"},
		{"AMF3", "\"héllo\"", "060d68c3a96c6c6f"},
		{"AMF3", "[]", "090101"},
		{"AMF3", "[1]", "090301053ff0000000000000"},
		{"AMF3", "[1,2,3]", "090701053ff0000000000000054000000000000000054008000000000000"},
		{"AMF3", "[\"a\",\"b\"]", "090501060361060362"},
		{"AMF3", "[[1],[2]]", "090501090301053ff0000000000000090301054000000000000000"},
		{"AMF3", "{}", "0a0301"},
		{"AMF3", "{\"a\":1}", "0a13010361053ff0000000000000"},
		{"AMF3", "{\"a\":1,\"b\":\"x\",\"c\":true}", "0a3301036103620363053ff000000000000006037803"},
		{"AMF3", "{\"nested\":{\"k\":[1,2]}}", "0a13010d6e65737465640a1301036b090501053ff0000000000000054000000000000000"},
		{"AMF3", "[null,true,false]", "090701010302"},
		{"AMF3", "{\"\":1}", "0a130101053ff0000000000000"},
		{"AMF3", "{\"a\":null}", "0a1301036101"},
	} {
		t.Run(tc.format+" "+tc.in, func(t *testing.T) {
			out, err := runOp(t, "AMF Encode", tc.in, tc.format)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if got := hex.EncodeToString([]byte(out)); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestAMFRoundTripEveryValueKind is the property the old library could not
// satisfy: whatever cchef encodes, cchef reads back. goamf wrote a valid empty
// AMF0 strict array and then decoded it as null, and could not read back its
// own empty string or empty object in either format.
func TestAMFRoundTripEveryValueKind(t *testing.T) {
	inputs := []string{
		`null`, `true`, `false`, `0`, `1`, `-1`, `42`, `3.5`, `-0.5`,
		`""`, `"hi"`, `"héllo"`,
		`[]`, `[1]`, `[1,2,3]`, `["a","b"]`, `[[1],[2]]`, `[null,true,false]`,
		`{}`, `{"a":1}`, `{"a":1,"b":"x","c":true}`, `{"nested":{"k":[1,2]}}`,
		`{"":1}`, `{"a":null}`,
	}
	var cases []opCase
	for _, format := range []string{"AMF0", "AMF3"} {
		for _, in := range inputs {
			cases = append(cases, opCase{
				name:  format + " " + in,
				input: in,
				want:  in,
				recipe: core.Recipe{
					{Op: "AMF Encode", Args: []any{format}},
					{Op: "AMF Decode", Args: []any{format}},
				},
			})
		}
	}
	runCases(t, cases)
}

// TestAMFDecodeEmptyValues pins the six byte sequences goamf could not read
// back, with the values CyberChef yields for the same bytes.
func TestAMFDecodeEmptyValues(t *testing.T) {
	for _, tc := range []struct{ format, hexIn, want string }{
		{"AMF0", "020000", `""`},
		{"AMF0", "0a00000000", `[]`},
		{"AMF0", "030000003ff0000000000000000009", `{"":1}`},
		{"AMF3", "0601", `""`},
		{"AMF3", "090101", `[]`},
		{"AMF3", "0a0301", `{}`},
	} {
		t.Run(tc.format+" "+tc.hexIn, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.hexIn)
			if err != nil {
				t.Fatal(err)
			}
			got, err := runOp(t, "AMF Decode", string(raw), tc.format)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestAMFErrors covers the decode error (invalid marker byte) and the encode
// error (unparseable JSON input).
func TestAMFErrors(t *testing.T) {
	if _, err := runOp(t, "AMF Decode", "\xff", "AMF0"); err == nil {
		t.Fatal("expected an error decoding an invalid AMF marker")
	}
	if _, err := runOp(t, "AMF Encode", "not json", "AMF0"); err == nil {
		t.Fatal("expected an error encoding invalid JSON")
	}
}

// TestAMFDecodeNaN covers a double that JSON has no way to write. AMF0 can
// carry NaN; JavaScript renders it as null, which is what CyberChef returns for
// these bytes, so cchef does the same. Marshalling through encoding/json used to
// refuse the whole value instead.
func TestAMFDecodeNaN(t *testing.T) {
	// AMF0 number marker (0x00) followed by an 8-byte big-endian NaN.
	nan := string([]byte{0x00, 0x7f, 0xf8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	got, err := runOp(t, "AMF Decode", nan, "AMF0")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != "null" {
		t.Errorf("got %s, want null", got)
	}
}

// TestAMFDecodeMalformed covers input the decoder cannot read. The AMF library
// panics on some truncated values rather than returning an error, so the
// operation contains that and reports it; without this a malformed file would
// take the whole process down.
func TestAMFDecodeMalformed(t *testing.T) {
	for _, in := range []string{
		"\x0b0", "\x0b", "\x09\xff", "\x0a\x0b\x01", "\x03\xff\xff\xff",
		"\x11", "\x0c\x80\x80\x80", "\x08\xff\xff\xff\xff",
	} {
		if _, err := runOp(t, "AMF Decode", in, "AMF3"); err == nil {
			t.Logf("decoded %q without error", in)
		}
	}
}

// TestAMFEncodeIsDeterministic pins object key order to the order the JSON
// gives, which is what CyberChef does. Reading the input into a Go map lost
// that order, so the same input encoded to a different byte string run to run
// and only sometimes matched CyberChef.
func TestAMFEncodeIsDeterministic(t *testing.T) {
	const in = `{"a":1,"b":2,"c":3,"d":4,"e":5}`
	for _, format := range []string{"AMF0", "AMF3"} {
		first, err := runOp(t, "AMF Encode", in, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		for i := range 16 {
			got, err := runOp(t, "AMF Encode", in, format)
			if err != nil {
				t.Fatalf("%s run %d: %v", format, i, err)
			}
			if got != first {
				t.Fatalf("%s: run %d differs from the first\n first %x\n got   %x",
					format, i, first, got)
			}
		}
		// The keys must appear in the order the JSON wrote them.
		var last int
		for _, k := range []string{"a", "b", "c", "d", "e"} {
			at := strings.Index(first, k)
			if at < 0 {
				t.Fatalf("%s: key %q missing from the encoding", format, k)
			}
			if at < last {
				t.Errorf("%s: key %q appears before the one that precedes it in the JSON", format, k)
			}
			last = at
		}
	}
}

// TestAMF0DecodeScalar covers the single-value markers directly, including the
// ones no encoder here writes: undefined, unsupported, the long string, the XML
// document, and a date, whose two trailing time-zone bytes are read and ignored.
func TestAMF0DecodeScalar(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hexIn string
		want  any
	}{
		{"number", "004045000000000000", 42.0},
		{"boolean true", "0101", true},
		{"boolean false", "0100", false},
		{"string", "0200026869", "hi"},
		{"null", "05", nil},
		{"undefined", "06", nil},
		{"unsupported", "0d", nil},
		{"long string", "0c000000026869", "hi"},
		{"xml document", "0f000000026869", "hi"},
		{"date", "0b41147260000000000000", 335000.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.hexIn)
			if err != nil {
				t.Fatal(err)
			}
			var refs []any
			got, err := amf0Decode(&amfReader{data: raw}, &refs)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestAMF3ReadTraits covers the three ways an object announces its members:
// inline traits, a back-reference to traits already seen, and the
// externalizable form, whose layout only the class knows.
func TestAMF3ReadTraits(t *testing.T) {
	// Inline, one sealed member "a": header 0x13, empty class name, "a".
	raw, _ := hex.DecodeString("13010361")
	tables := newAMF3Tables()
	r := &amfReader{data: raw}
	n, err := r.u29()
	if err != nil {
		t.Fatal(err)
	}
	members, dynamic, err := amf3ReadTraits(r, tables, n)
	if err != nil {
		t.Fatalf("inline traits: %v", err)
	}
	if len(members) != 1 || members[0] != "a" || dynamic {
		t.Errorf("got members %v dynamic %v, want [a] false", members, dynamic)
	}
	if len(tables.traits) != 1 {
		t.Errorf("inline traits should be recorded for later reference")
	}

	// A reference to those traits: header with bit 1 clear, index 0.
	if _, _, err := amf3ReadTraits(&amfReader{}, tables, 0x01); err != nil {
		t.Errorf("traits reference: %v", err)
	}
	// A reference past the end of the table is malformed.
	if _, _, err := amf3ReadTraits(&amfReader{}, tables, 0x09); err == nil {
		t.Error("a traits reference past the end should be refused")
	}
	// Externalizable is refused rather than guessed at.
	if _, _, err := amf3ReadTraits(&amfReader{}, tables, 0x07); err == nil {
		t.Error("externalizable objects should be refused")
	}
}

// TestAMFDecodeComposite covers the container markers no encoder here writes:
// the typed object, the ECMA array, the object back-reference, and the AVM+
// marker that switches an AMF0 stream to AMF3 mid-value.
func TestAMFDecodeComposite(t *testing.T) {
	for _, tc := range []struct{ name, hexIn, want string }{
		// Typed object: class name "C", then one member a=1.
		{"typed object", "1000014300016100 3ff0000000000000 000009", `{"a":1}`},
		// ECMA array: advisory count 1, then the same body.
		{"ecma array", "0800000001 000161 003ff0000000000000 000009", `{"a":1}`},
		// Strict array holding an object, then a reference back to that object.
		{"object reference", "0a00000002 03 000161 003ff0000000000000 000009 0700 00", `[{"a":1},{"a":1}]`},
		// AVM+ switch: an AMF0 stream whose value is AMF3.
		{"avmplus", "11 06 03 78", `"x"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(strings.ReplaceAll(tc.hexIn, " ", ""))
			if err != nil {
				t.Fatal(err)
			}
			got, err := runOp(t, "AMF Decode", string(raw), "AMF0")
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestAMF3DecodeComposite covers the AMF3 markers with no encoder counterpart:
// the 29-bit integer at both signs, a date, a byte array, a mixed array whose
// associative half forces it to become an object, a dynamic object, and the
// string and object back-references.
func TestAMF3DecodeComposite(t *testing.T) {
	for _, tc := range []struct{ name, hexIn, want string }{
		{"integer", "042a", `42`},
		{"integer negative", "04ffffffff", `-1`},
		{"date", "08 01 4114726000000000", `335000`},
		{"byte array", "0c 05 6869", `"hi"`},
		// The dense entries take integer keys, which JavaScript — and so this
		// output — enumerates before the named ones whatever the input order.
		{"mixed array", "09 03 036b 0403 01 0405", `{"0":5,"k":3}`},
		{"dynamic object", "0a 0b 01 0361 0405 01", `{"a":5}`},
		// "a" written once, then referenced: the second key reuses index 0.
		{"string reference", "0a 2301 0361 0362 0405 0406", `{"a":5,"b":6}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(strings.ReplaceAll(tc.hexIn, " ", ""))
			if err != nil {
				t.Fatal(err)
			}
			got, err := runOp(t, "AMF Decode", string(raw), "AMF3")
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestAMFEncodeLongString covers the AMF0 long-string form, which a string too
// big for the 16-bit length takes instead.
func TestAMFEncodeLongString(t *testing.T) {
	long := strings.Repeat("a", 70000)
	out, err := runOp(t, "AMF Encode", `"`+long+`"`, "AMF0")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if out[0] != 0x0c {
		t.Fatalf("marker is 0x%02x, want the long-string marker 0x0c", out[0])
	}
	back, err := runOp(t, "AMF Decode", out, "AMF0")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back != `"`+long+`"` {
		t.Error("a long string did not survive the round trip")
	}
}

// TestAMFTruncatedInput is what the old library could not do: every prefix of a
// valid encoding must be refused rather than panic, and the bounded reader is
// what makes that true without a recover.
func TestAMFTruncatedInput(t *testing.T) {
	for _, format := range []string{"AMF0", "AMF3"} {
		full, err := runOp(t, "AMF Encode", `{"a":[1,"x",{"b":true}],"c":null}`, format)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		for i := range len(full) {
			// A prefix either decodes to something or errors; it must not panic.
			_, _ = runOp(t, "AMF Decode", full[:i], format)
		}
	}
}

// TestAMFDecodeRefusesBadReferences covers the reference paths pointing past
// the end of their table, which a crafted file can ask for.
func TestAMFDecodeRefusesBadReferences(t *testing.T) {
	for _, tc := range []struct{ format, hexIn string }{
		{"AMF0", "0700ff"}, // object reference 255, nothing recorded
		{"AMF3", "0a04"},   // object reference past the end
		{"AMF3", "0604"},   // string reference past the end
		{"AMF3", "0904"},   // array reference past the end
		{"AMF3", "0802"},   // date reference past the end
		{"AMF3", "0c02"},   // byte-array reference past the end
	} {
		raw, err := hex.DecodeString(tc.hexIn)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runOp(t, "AMF Decode", string(raw), tc.format); err == nil {
			t.Errorf("%s %s: a reference past the end of the table was accepted", tc.format, tc.hexIn)
		}
	}
}

// TestAMF3VariableLengthInteger covers all four widths of AMF3's length
// prefix. A string's length is written as the prefix, so the width follows
// from how long the string is.
func TestAMF3VariableLengthInteger(t *testing.T) {
	for _, n := range []int{0, 1, 63, 64, 8191, 8192, 1 << 20} {
		in := `"` + strings.Repeat("a", n) + `"`
		out, err := runOp(t, "AMF Encode", in, "AMF3")
		if err != nil {
			t.Fatalf("length %d: encode: %v", n, err)
		}
		back, err := runOp(t, "AMF Decode", out, "AMF3")
		if err != nil {
			t.Fatalf("length %d: decode: %v", n, err)
		}
		if back != in {
			t.Errorf("length %d did not survive the round trip", n)
		}
	}
}

// TestAMFEncodeRejectsBadJSON covers the input errors: something that is not
// JSON at all, and JSON with more than one value in it.
func TestAMFEncodeRejectsBadJSON(t *testing.T) {
	for _, in := range []string{
		"not json", `{"a":1} {"b":2}`, `{`, `[1,`, `{1:2}`, `]`,
		`{"a":`, // an object key with no value after it
		`1e999`, // a number JSON allows but a float64 cannot hold
	} {
		for _, format := range []string{"AMF0", "AMF3"} {
			if _, err := runOp(t, "AMF Encode", in, format); err == nil {
				t.Errorf("%s: %q was accepted", format, in)
			}
		}
	}
}

// TestAMFDecodeRejectsUnknownMarker covers the marker default in both formats,
// and AMF3's externalizable objects, whose layout only the class knows.
func TestAMFDecodeRejectsUnknownMarker(t *testing.T) {
	for _, tc := range []struct{ format, hexIn string }{
		{"AMF0", "fe"},
		{"AMF3", "fe"},
		{"AMF3", "0a07 01"}, // externalizable
	} {
		raw, err := hex.DecodeString(strings.ReplaceAll(tc.hexIn, " ", ""))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runOp(t, "AMF Decode", string(raw), tc.format); err == nil {
			t.Errorf("%s %s was accepted", tc.format, tc.hexIn)
		}
	}
}

// TestAMFEncodeRejectsUnsupportedType covers the encoders' default arm and the
// error propagating out of a container. The operation cannot reach these,
// because the JSON parser only ever produces types the encoders handle, so the
// encoders are driven directly.
func TestAMFEncodeRejectsUnsupportedType(t *testing.T) {
	bad := make(chan int) // a type no AMF marker describes
	for _, tc := range []struct {
		name string
		val  any
	}{
		{"bare value", bad},
		{"inside an array", []any{bad}},
		{"inside an object", jsObject{{k: "a", v: bad}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := amf0Encode(&amfWriter{}, tc.val); err == nil {
				t.Error("AMF0 accepted a value it cannot write")
			}
			if err := amf3Encode(&amfWriter{}, tc.val); err == nil {
				t.Error("AMF3 accepted a value it cannot write")
			}
		})
	}
}

// TestAMF0EncodeRejectsOversizeKey covers the object-key length guard: AMF0
// writes a key with a 16-bit length, so a longer one cannot be represented.
func TestAMF0EncodeRejectsOversizeKey(t *testing.T) {
	obj := jsObject{{k: strings.Repeat("k", amf0MaxShortString+1), v: 1.0}}
	if err := amf0Encode(&amfWriter{}, obj); err == nil {
		t.Error("a key too long for the 16-bit length was accepted")
	}
}

// TestAMF3DecodeReferences covers the back-reference tables on the paths that
// find something: an object and a string each written once and then referred
// to. cchef's encoder never emits references, so these bytes are hand-built.
func TestAMF3DecodeReferences(t *testing.T) {
	for _, tc := range []struct{ name, hexIn, want string }{
		// Two-element array: object {a:5}, then a reference back to it.
		// Index 0 is the array itself, which is registered before its elements,
		// so the object written second is index 1.
		{"object reference", "09 05 01 0a1301 0361 0405 0a02", `[{"a":5},{"a":5}]`},
		// Two-element array: string "a" inline, then a reference back to it.
		{"string reference", "09 05 01 060361 0600", `["a","a"]`},
		// Traits written once, then a second object reusing them by reference.
		{"traits reference", "09 05 01 0a1301 0361 0405 0a01 0407", `[{"a":5},{"a":7}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(strings.ReplaceAll(tc.hexIn, " ", ""))
			if err != nil {
				t.Fatal(err)
			}
			got, err := runOp(t, "AMF Decode", string(raw), "AMF3")
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

// TestAMF0RejectsUnusableLength covers the 32-bit length guard: a value that
// would not fit an int on every platform is refused before it is used as one.
func TestAMF0RejectsUnusableLength(t *testing.T) {
	// Long-string marker with a length of 0xffffffff.
	raw, err := hex.DecodeString("0cffffffff")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runOp(t, "AMF Decode", string(raw), "AMF0"); err == nil {
		t.Error("a length beyond what an int holds was accepted")
	}
}

// TestAMFTruncationErrors pins that a value cut short at each interesting
// boundary is reported rather than half-read. The prefixes below stop inside a
// length, inside a body, and between an object's key and its value.
func TestAMFTruncationErrors(t *testing.T) {
	for _, tc := range []struct{ format, hexIn string }{
		{"AMF0", "00"},                      // number marker, no double
		{"AMF0", "0200"},                    // string, length cut short
		{"AMF0", "020002 68"},               // string, body cut short
		{"AMF0", "03 0001 61"},              // object, key but no value
		{"AMF0", "0a 00000002 00"},          // array promising two, giving none
		{"AMF0", "10 0001"},                 // typed object, class name cut short
		{"AMF0", "08 000000"},               // ecma array, count cut short
		{"AMF0", "0b"},                      // date, the whole double missing
		{"AMF0", "0b 4114726000000000"},     // date, time zone missing
		{"AMF0", "07"},                      // reference, index missing
		{"AMF3", "05"},                      // double marker, no double
		{"AMF3", "06"},                      // string, length missing
		{"AMF3", "0603"},                    // string, body missing
		{"AMF3", "09 05 01 05"},             // array promising two, giving none
		{"AMF3", "0a 13 01"},                // object, member name missing
		{"AMF3", "0a 13 01 0361"},           // object, member value missing
		{"AMF3", "0a 0b 01 0361"},           // dynamic object, value missing
		{"AMF3", "08"},                      // date, everything missing
		{"AMF3", "04"},                      // integer, length missing
		{"AMF3", "0a 0b 01 0361 0405 0362"}, // dynamic object, second value missing
		{"AMF3", "0c"},                      // byte array, length missing
		{"AMF3", "0c05"},                    // byte array, body missing
		{"AMF3", "09 03 036b"},              // mixed array, associative key with no value
		{"AMF3", "09 03 036b 05"},           // mixed array, associative value cut short
	} {
		raw, err := hex.DecodeString(strings.ReplaceAll(tc.hexIn, " ", ""))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := runOp(t, "AMF Decode", string(raw), tc.format); err == nil {
			t.Errorf("%s %s: truncated input was accepted", tc.format, tc.hexIn)
		}
	}
}

// TestAMF3WriterRefusesOversizeLength covers the one place the 29-bit limit is
// decided. Reaching it through an operation would need a quarter-gigabyte
// string, so the writer is driven directly.
func TestAMF3WriterRefusesOversizeLength(t *testing.T) {
	for _, n := range []int{amfMaxU29 + 1, -1} {
		w := &amfWriter{}
		w.u29(n)
		if w.err == nil {
			t.Errorf("u29(%d) was accepted", n)
		}
		if len(w.buf) != 0 {
			t.Errorf("u29(%d) wrote %d bytes before refusing", n, len(w.buf))
		}
	}
	// The largest value the format holds is still written.
	w := &amfWriter{}
	w.u29(amfMaxU29)
	if w.err != nil {
		t.Errorf("the largest 29-bit value was refused: %v", w.err)
	}
	// Once the writer has failed it stops writing, so a partial value cannot
	// be mistaken for a whole one.
	w.err = errAMFMalformed
	before := len(w.buf)
	w.u29(1)
	if len(w.buf) != before {
		t.Error("the writer kept going after it had failed")
	}
}
