package yara

// The way certificates and the blobs holding them are written down. Each
// element opens with what it is, then how long it is, then that many bytes, and
// an element may hold others, which is how the shapes nest.

const (
	// The classes an element can belong to: the shapes everyone agrees on, and
	// the ones a particular blob numbers for itself.
	derUniversal = 0
	derContext   = 2

	// The tags of the shapes looked for by name.
	derInteger       = 2
	derBitString     = 3
	derOctet         = 4
	derOID           = 6
	derUTF8          = 12
	derSequence      = 16
	derSet           = 17
	derUTCTime       = 23
	derGeneralTime   = 24
	derGeneralString = 27
)

// derElement is one element: which class it belongs to, what it is within that
// class, whether it holds others, its contents, and the whole of it including
// what says how long it is.
type derElement struct {
	class byte
	tag   int
	holds bool
	body  []byte
	raw   []byte
}

// named says whether an element is the one being looked for.
func (e derElement) named(class byte, tag int) bool {
	return e.class == class && e.tag == tag
}

// readDER reads the element at the front of some bytes and gives back what
// follows it.
func readDER(b []byte) (derElement, []byte, bool) {
	e, at, ok := readDERTag(b)
	if !ok {
		return derElement{}, nil, false
	}
	length, at, unending, ok := readDERLength(b, at)
	if !ok {
		return derElement{}, nil, false
	}
	if unending {
		// One that does not say how long it is must hold others, since the pair
		// of nothings that closes it is read as an element in its own right.
		if !e.holds {
			return derElement{}, nil, false
		}
		return readDERUnending(b, e, at)
	}
	if length > len(b)-at {
		return derElement{}, nil, false
	}
	e.body, e.raw = b[at:at+length], b[:at+length]
	return e, rest(b[at+length:]), true
}

// readDERTag reads what an element is, which is a class, whether it holds
// others, and a number within that class. A number too large to fit beside the
// class is written on in as many following bytes as it needs, seven bits at a
// time.
func readDERTag(b []byte) (derElement, int, bool) {
	if len(b) == 0 {
		return derElement{}, 0, false
	}
	// The top two bits are the class, the next says whether it holds others,
	// and the low five are the number unless they are all set.
	const (
		classShift = 6
		holdsBit   = 0x20
		tagMask    = 0x1F
		moreBit    = 0x80
		tagBits    = 7
		// A number is not read past what an int can hold without losing bits.
		maxTagBytes = 4
	)
	e := derElement{class: b[0] >> classShift, holds: b[0]&holdsBit != 0}
	if tag := int(b[0] & tagMask); tag != tagMask {
		e.tag = tag
		return e, 1, true
	}
	at := 1
	for taken := 0; ; taken++ {
		if at >= len(b) || taken == maxTagBytes {
			return derElement{}, 0, false
		}
		e.tag = e.tag<<tagBits | int(b[at]&^moreBit)
		more := b[at]&moreBit != 0
		at++
		if !more {
			return e, at, true
		}
	}
}

// readDERLength reads how long an element is. A length under 128 is the byte
// itself; anything longer says instead how many bytes the length takes. The one
// value that says nothing about the length means the element is closed by a
// pair of nothings rather than measured.
func readDERLength(b []byte, at int) (length, next int, unending, ok bool) {
	const (
		longBit  = 0x80
		countMax = 0x7F
		// A length is not read past what an int can hold on any machine.
		maxLengthBytes = 4
	)
	if at >= len(b) {
		return 0, 0, false, false
	}
	first := b[at]
	at++
	if first&longBit == 0 {
		return int(first), at, false, true
	}
	count := int(first & countMax)
	if count == 0 {
		return 0, at, true, true
	}
	if count > maxLengthBytes || at+count > len(b) {
		return 0, 0, false, false
	}
	for range count {
		length = length<<8 | int(b[at])
		at++
	}
	return length, at, false, true
}

// readDERUnending reads an element that does not say how long it is by reading
// what it holds until the pair of nothings that closes it.
func readDERUnending(b []byte, e derElement, at int) (derElement, []byte, bool) {
	const closerSize = 2
	for start := at; ; {
		if at+closerSize > len(b) {
			return derElement{}, nil, false
		}
		if b[at] == 0 && b[at+1] == 0 {
			e.body, e.raw = b[start:at], b[:at+closerSize]
			return e, rest(b[at+closerSize:]), true
		}
		inner, _, ok := readDER(b[at:])
		if !ok {
			return derElement{}, nil, false
		}
		at += len(inner.raw)
	}
}

// rest keeps an empty tail as nothing at all, so that what follows the last
// element of a body reads the same whichever way it was written.
func rest(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}

// derParts reads every element of a body in turn, stopping where the bytes stop
// making sense.
func derParts(body []byte) []derElement {
	var parts []derElement
	for len(body) > 0 {
		e, next, ok := readDER(body)
		if !ok {
			break
		}
		parts = append(parts, e)
		body = next
	}
	return parts
}
