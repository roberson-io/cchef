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
