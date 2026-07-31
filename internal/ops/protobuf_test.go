package ops

import "testing"

// TestProtobufParserInternals covers raw-parser branches that parse()'s loop
// guard keeps unreachable through the operation: the past-end overrun check and
// the high-shift arm of the multi-byte field-number decoder.
func TestProtobufParserInternals(t *testing.T) {
	// An exhausted buffer reports the overrun sentinel.
	if _, _, err := newProtobufParser(nil).parseField(); err == nil {
		t.Fatal("parseField(exhausted): expected an overrun error")
	}
	// A six-byte continuation tag drives the field number past the shift==28 arm.
	p := &protobufParser{data: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, fieldTypes: map[string]any{}}
	if got := p.fieldNumber(); got <= 0 {
		t.Fatalf("fieldNumber(large tag) = %d, want a large positive value", got)
	}
	// protobufTypeInfo returns "" for reserved wire types.
	if got := protobufTypeInfo(3); got != "" {
		t.Fatalf("protobufTypeInfo(3) = %q, want empty", got)
	}
}

// TestProtobufTruncatedLengthDelimited covers a length-delimited field whose
// length runs past the end of the buffer. Reading the length can leave the
// offset beyond the data, so the slice that follows has to tolerate a start
// past its end rather than panicking; CyberChef reports these as an exhausted
// buffer, and so does cchef.
func TestProtobufTruncatedLengthDelimited(t *testing.T) {
	for _, in := range []string{"2", "22", "\x12", "\x12\xff", "\x0a\x7f", "\x12\x80\x80\x80\x80\x10"} {
		if _, err := runOp(t, "Protobuf Decode", in); err == nil {
			t.Errorf("expected an error for truncated input %q", in)
		}
	}
}

// TestProtobufHugeLength covers a length-delimited field whose length is
// larger than the buffer, including values too large for an int. The buffer
// cannot satisfy them, so each is an overrun rather than a crash.
func TestProtobufHugeLength(t *testing.T) {
	for _, in := range []string{
		"2\x91\x8d\xae\x86\xfe\xc6\xe5\xdb\x8600",
		"\x12\xff\xff\xff\xff\xff\xff\xff\xff\x7f",
		"\x12\x80\x80\x80\x80\x80\x80\x80\x80\x80\x01",
	} {
		if _, err := runOp(t, "Protobuf Decode", in); err == nil {
			t.Errorf("expected an overrun error for %q", in)
		}
	}
}
