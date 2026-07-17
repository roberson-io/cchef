package ops

import "encoding/binary"

// md64 is a reusable Merkle–Damgård streaming state for 64-byte-block hashes
// with standard length padding (0x80, zero-fill to 56 mod 64, then a 64-bit
// bit-length). It is shared by the crypto-api-compatible SHA0, HAS-160 and
// RIPEMD-128/256/320 ports.
type md64 struct {
	x   [64]byte
	nx  int
	len uint64
}

// write feeds p into the buffer, invoking block for each complete 64-byte block.
func (d *md64) write(p []byte, block func([]byte)) {
	d.len += uint64(len(p))
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == 64 {
			block(d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	for len(p) >= 64 {
		block(p[:64])
		p = p[64:]
	}
	d.nx = copy(d.x[:], p)
}

// pad appends the final padding and flushes the last block(s). littleEndian
// selects the byte order of the appended bit-length (RIPEMD/HAS-160 use little,
// SHA0 uses big).
func (d *md64) pad(block func([]byte), littleEndian bool) {
	bits := d.len << 3
	padLen := 56 - d.nx
	if padLen <= 0 {
		padLen += 64
	}
	buf := make([]byte, padLen+8)
	buf[0] = 0x80
	if littleEndian {
		binary.LittleEndian.PutUint64(buf[padLen:], bits)
	} else {
		binary.BigEndian.PutUint64(buf[padLen:], bits)
	}
	d.write(buf, block)
}
