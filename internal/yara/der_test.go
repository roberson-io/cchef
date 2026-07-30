package yara

import (
	"bytes"
	"testing"
)

// The way certificates and the blobs holding them are written down. Each
// element opens with what it is, then how long it is, then that many bytes,
// and an element may hold others.

// TestReadDER covers reading one element off the front of some bytes.
func TestReadDER(t *testing.T) {
	cases := []struct {
		name  string
		in    []byte
		class byte
		tag   int
		holds bool
		body  []byte
		rest  []byte
	}{
		{
			"a plain number",
			[]byte{0x02, 0x01, 0x05},
			derUniversal, derInteger, false,
			[]byte{0x05},
			nil,
		},
		{
			"a run holding two numbers",
			[]byte{0x30, 0x06, 0x02, 0x01, 0x05, 0x02, 0x01, 0x07},
			derUniversal, derSequence, true,
			[]byte{0x02, 0x01, 0x05, 0x02, 0x01, 0x07},
			nil,
		},
		{
			"whatever follows is left alone",
			[]byte{0x02, 0x01, 0x05, 0xAA, 0xBB},
			derUniversal, derInteger, false,
			[]byte{0x05},
			[]byte{0xAA, 0xBB},
		},
		{
			"nothing in it at all",
			[]byte{0x05, 0x00},
			derUniversal, 5, false,
			[]byte{},
			nil,
		},
		{
			"one the writer numbered itself",
			[]byte{0xA0, 0x03, 0x02, 0x01, 0x05},
			derContext, 0, true,
			[]byte{0x02, 0x01, 0x05},
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, rest, ok := readDER(c.in)
			if !ok {
				t.Fatal("the element did not read at all")
			}
			if got.class != c.class || got.tag != c.tag || got.holds != c.holds {
				t.Errorf("it reads as class %d tag %d holding %v, want class %d tag %d holding %v",
					got.class, got.tag, got.holds, c.class, c.tag, c.holds)
			}
			if !bytes.Equal(got.body, c.body) {
				t.Errorf("its contents are %x, want %x", got.body, c.body)
			}
			if !bytes.Equal(rest, c.rest) {
				t.Errorf("what follows is %x, want %x", rest, c.rest)
			}
			if want := c.in[:len(c.in)-len(c.rest)]; !bytes.Equal(got.raw, want) {
				t.Errorf("the whole of it is %x, want %x", got.raw, want)
			}
		})
	}
}

// TestReadDERLongLength covers an element too long to say its length in one
// byte, which says instead how many bytes the length itself takes.
func TestReadDERLongLength(t *testing.T) {
	body := bytes.Repeat([]byte{0xCD}, 300)
	in := append([]byte{0x04, 0x82, 0x01, 0x2C}, body...)
	got, rest, ok := readDER(in)
	if !ok {
		t.Fatal("the element did not read at all")
	}
	if got.tag != derOctet || !bytes.Equal(got.body, body) {
		t.Errorf("it reads as tag %d over %d bytes, want tag %d over %d",
			got.tag, len(got.body), derOctet, len(body))
	}
	if len(rest) != 0 {
		t.Errorf("%d bytes are left over, want none", len(rest))
	}
}

// TestReadDERHighTag covers a tag too large to fit beside the class, which is
// written on in as many following bytes as it needs.
func TestReadDERHighTag(t *testing.T) {
	// Tag 128 takes two bytes: the first carries the top bits and says another
	// follows, the second carries the rest.
	got, _, ok := readDER([]byte{0x3F, 0x81, 0x00, 0x01, 0xFF})
	if !ok {
		t.Fatal("the element did not read at all")
	}
	if got.tag != 128 || !got.holds {
		t.Errorf("it reads as tag %d holding %v, want tag 128 holding true", got.tag, got.holds)
	}
}

// TestReadDERUnendingLength covers an element that does not say how long it is
// and is closed by a pair of nothings instead, which some writers of these
// blobs use even though the strict form forbids it.
func TestReadDERUnendingLength(t *testing.T) {
	in := []byte{0x30, 0x80, 0x02, 0x01, 0x05, 0x02, 0x01, 0x07, 0x00, 0x00, 0xEE}
	got, rest, ok := readDER(in)
	if !ok {
		t.Fatal("the element did not read at all")
	}
	want := []byte{0x02, 0x01, 0x05, 0x02, 0x01, 0x07}
	if !bytes.Equal(got.body, want) {
		t.Errorf("its contents are %x, want %x", got.body, want)
	}
	if !bytes.Equal(got.raw, in[:len(in)-1]) {
		t.Errorf("the whole of it is %x, want everything up to the last byte", got.raw)
	}
	if !bytes.Equal(rest, []byte{0xEE}) {
		t.Errorf("what follows is %x, want ee", rest)
	}
}

// TestReadDERUnendingHoldingAnother covers one unending element inside another,
// so that the closing pair of nothings is matched to the right one.
func TestReadDERUnendingHoldingAnother(t *testing.T) {
	in := []byte{0x30, 0x80, 0x30, 0x80, 0x02, 0x01, 0x05, 0x00, 0x00, 0x02, 0x01, 0x09, 0x00, 0x00}
	got, rest, ok := readDER(in)
	if !ok {
		t.Fatal("the element did not read at all")
	}
	if len(rest) != 0 {
		t.Errorf("%d bytes are left over, want none", len(rest))
	}
	parts := derParts(got.body)
	if len(parts) != 2 {
		t.Fatalf("it holds %d elements, want 2", len(parts))
	}
	if parts[0].tag != derSequence || parts[1].tag != derInteger {
		t.Errorf("it holds tags %d and %d, want %d and %d",
			parts[0].tag, parts[1].tag, derSequence, derInteger)
	}
}

// TestReadDERRefused covers bytes that are not a whole element, which are
// turned away rather than read as far as they go.
func TestReadDERRefused(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{"nothing at all", nil},
		{"a tag and no length", []byte{0x02}},
		{"a length running past the end", []byte{0x02, 0x05, 0x01}},
		{"a long length whose bytes are not all there", []byte{0x04, 0x82, 0x01}},
		{"a long length too large to hold", []byte{0x04, 0x88, 1, 1, 1, 1, 1, 1, 1, 1}},
		{"a high tag that never ends", []byte{0x3F, 0x81, 0x81}},
		{"a high tag too large to hold", []byte{0x3F, 0x81, 0x81, 0x81, 0x81, 0x81, 0x01, 0x00}},
		{"an unending element never closed", []byte{0x30, 0x80, 0x02, 0x01, 0x05}},
		{"an unending element holding a broken one", []byte{0x30, 0x80, 0x02, 0x09, 0x05}},
		{"an unending element that does not hold others", []byte{0x02, 0x80, 0x00, 0x00}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, ok := readDER(c.in); ok {
				t.Error("the bytes read as an element, want them turned away")
			}
		})
	}
}

// TestDERParts covers reading every element of a body in turn, and stopping
// where the bytes stop making sense.
func TestDERParts(t *testing.T) {
	parts := derParts([]byte{0x02, 0x01, 0x05, 0x0C, 0x02, 'h', 'i', 0x01, 0x01, 0xFF})
	if len(parts) != 3 {
		t.Fatalf("%d elements read, want 3", len(parts))
	}
	if parts[1].tag != derUTF8 || string(parts[1].body) != "hi" {
		t.Errorf("the second is tag %d over %q, want tag %d over \"hi\"",
			parts[1].tag, parts[1].body, derUTF8)
	}

	// A broken element ends the run, and what came before it is kept.
	parts = derParts([]byte{0x02, 0x01, 0x05, 0x04, 0x7F, 0x00})
	if len(parts) != 1 || parts[0].tag != derInteger {
		t.Errorf("%d elements read before the broken one, want just the number", len(parts))
	}
	if got := derParts(nil); got != nil {
		t.Errorf("an empty body read as %d elements, want none", len(got))
	}
}

// TestDERNamed covers asking whether an element is the one being looked for.
func TestDERNamed(t *testing.T) {
	e, _, _ := readDER([]byte{0xA0, 0x03, 0x02, 0x01, 0x05})
	if !e.named(derContext, 0) {
		t.Error("the element is not taken for the one the writer numbered nought")
	}
	if e.named(derUniversal, 0) || e.named(derContext, 1) {
		t.Error("the element is taken for one it is not")
	}
}
