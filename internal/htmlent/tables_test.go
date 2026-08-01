package htmlent

import "testing"

// The tables are generated (tools/htmlentgen), so what is worth checking is
// not their contents but that the two agree with each other: anything the
// encoder can emit, the decoder has to read back as the same character. A
// regeneration that dropped or renamed an entry on one side only would break
// round-tripping, and nothing else would notice.
func TestTablesAgree(t *testing.T) {
	if len(ByteToEntity) == 0 || len(EntityToByte) == 0 {
		t.Fatal("tables are empty; the generator did not run")
	}
	for cp, name := range ByteToEntity {
		got, ok := EntityToByte[name]
		if !ok {
			t.Errorf("&%s; is emitted for U+%04X but cannot be decoded", name, cp)
			continue
		}
		if got != cp {
			t.Errorf("&%s; encodes U+%04X but decodes to U+%04X", name, cp, got)
		}
	}
}

// The decode table deliberately holds more names than the encode table: a
// character often has several accepted spellings and only one is written out.
// The extra names are the point of the table, so losing them would be a silent
// loss of input cchef used to accept.
func TestDecodeTableHasAliases(t *testing.T) {
	if len(EntityToByte) <= len(ByteToEntity) {
		t.Errorf("decode table has %d names and encode table %d; the aliases are missing",
			len(EntityToByte), len(ByteToEntity))
	}
	// Two spellings of the same character, one of which is never emitted.
	for _, alias := range []string{"amp", "AMP", "lt", "LT", "quot", "QUOT"} {
		if _, ok := EntityToByte[alias]; !ok {
			t.Errorf("&%s; is not decodable", alias)
		}
	}
}
