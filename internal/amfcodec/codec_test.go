package amfcodec

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// AMF has no CyberChef fixture file, and CyberChef's decoder returns its
// library's annotated parse tree rather than a value, so the two are compared
// on encoding — which is byte-identical — and on the values a decode yields.
//
// The encode vectors were produced by cchef and spot-checked against the
// oracle. CyberChef's AMF3 encoder cannot write null, booleans or arrays at
// all, so for those the specification is the only reference.

// TestAMFEncodeVectors pins the bytes each JSON value encodes to.
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
			got, err := Decode0(&Reader{data: raw}, &refs)
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
	tables := NewTables3()
	r := &Reader{data: raw}
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
	if _, _, err := amf3ReadTraits(&Reader{}, tables, 0x01); err != nil {
		t.Errorf("traits reference: %v", err)
	}
	// A reference past the end of the table is malformed.
	if _, _, err := amf3ReadTraits(&Reader{}, tables, 0x09); err == nil {
		t.Error("a traits reference past the end should be refused")
	}
	// Externalizable is refused rather than guessed at.
	if _, _, err := amf3ReadTraits(&Reader{}, tables, 0x07); err == nil {
		t.Error("externalizable objects should be refused")
	}
}

// TestAMFDecodeComposite covers the container markers no encoder here writes:
// the typed object, the ECMA array, the object back-reference, and the AVM+
// marker that switches an AMF0 stream to AMF3 mid-value.
func TestAMFEncodeRejectsUnsupportedType(t *testing.T) {
	bad := make(chan int) // a type no AMF marker describes
	for _, tc := range []struct {
		name string
		val  any
	}{
		{"bare value", bad},
		{"inside an array", []any{bad}},
		{"inside an object", jsonval.Object{{K: "a", V: bad}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Encode0(&Writer{}, tc.val); err == nil {
				t.Error("AMF0 accepted a value it cannot write")
			}
			if err := Encode3(&Writer{}, tc.val); err == nil {
				t.Error("AMF3 accepted a value it cannot write")
			}
		})
	}
}

// TestAMF0EncodeRejectsOversizeKey covers the object-key length guard: AMF0
// writes a key with a 16-bit length, so a longer one cannot be represented.
func TestAMF0EncodeRejectsOversizeKey(t *testing.T) {
	obj := jsonval.Object{{K: strings.Repeat("k", amf0MaxShortString+1), V: 1.0}}
	if err := Encode0(&Writer{}, obj); err == nil {
		t.Error("a key too long for the 16-bit length was accepted")
	}
}

// TestAMF3DecodeReferences covers the back-reference tables on the paths that
// find something: an object and a string each written once and then referred
// to. cchef's encoder never emits references, so these bytes are hand-built.
func TestAMF3WriterRefusesOversizeLength(t *testing.T) {
	for _, n := range []int{amfMaxU29 + 1, -1} {
		w := &Writer{}
		w.u29(n)
		if w.err == nil {
			t.Errorf("u29(%d) was accepted", n)
		}
		if len(w.buf) != 0 {
			t.Errorf("u29(%d) wrote %d bytes before refusing", n, len(w.buf))
		}
	}
	// The largest value the format holds is still written.
	w := &Writer{}
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
