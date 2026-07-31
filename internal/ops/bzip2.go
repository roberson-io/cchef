package ops

import (
	"bytes"
	"compress/bzip2"
	"errors"
	"io"

	"github.com/roberson-io/cchef/internal/core"
)

// Bzip2 compression and decompression, the two directions of the format Julian
// Seward built around the Burrows-Wheeler transform.

// A bzip2 encoder whose output matches libbzip2 byte for byte.
//
// bzip2 compresses in five stages. A run-length pass shortens repeats of four
// or more bytes; the result is gathered into a block, which the Burrows-Wheeler
// transform rotates and sorts so that like bytes fall together; a move-to-front
// pass turns the long runs that leaves into runs of zero, which a second
// run-length pass encodes; and the symbols are finally written out under up to
// six Huffman tables, one chosen per fifty symbols.
//
// Byte-exactness rests on the last stage. Where the format leaves a choice —
// how many tables, which symbols each covers, and how the code lengths are
// settled — libbzip2 makes a particular one, and any other choice produces a
// stream that is valid but different. Those decisions are reproduced here.
// The sorting is not among them: the transform is determined by the block
// alone, so the block sort only has to be correct, not identical.

// The block size, in hundreds of kilobytes, bounds how much data one
// Burrows-Wheeler transform covers.
var (
	bzip2BlockSizeMin = float64(1)
	bzip2BlockSizeMax = float64(9)
)

// Bzip2Compress compresses the input into a bzip2 stream.
type Bzip2Compress struct{}

// Meta returns the operation metadata.
func (Bzip2Compress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bzip2 Compress",
		Module:      "Compression",
		Description: "Bzip2 is a compression library developed by Julian Seward (of GHC fame) that uses the Burrows-Wheeler algorithm. It only supports compressing single files and its compression is slow, however is more effective than Deflate (.gz &amp; .zip).",
		InfoURL:     "https://wikipedia.org/wiki/Bzip2",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Bzip2Compress) Args() []core.ArgDef {
	return []core.ArgDef{
		{
			Name: "Block size (100s of kb)", Type: core.ArgNumber, Value: float64(9),
			Min: &bzip2BlockSizeMin, Max: &bzip2BlockSizeMax,
		},
		{Name: "Work factor", Type: core.ArgNumber, Integer: true, Value: float64(30)},
	}
}

// Run compresses the input.
func (Bzip2Compress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	blockSize, _ := args[0].(float64)
	// The work factor is accepted and ignored. In libbzip2 it only decides when
	// the block sort abandons its faster path for a slower one; both reach the
	// same transform, so it changes how long compressing takes and nothing about
	// what comes out.

	data := in.Bytes()
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	return core.NewDish(bzip2Encode(data, int(blockSize)), core.TypeByteArray), nil
}

func init() { core.Register(Bzip2Compress{}) }

// Bzip2Decompress reads a bzip2 stream back into the bytes it was made from.
type Bzip2Decompress struct{}

// Meta returns the operation metadata.
func (Bzip2Decompress) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bzip2 Decompress",
		Module:      "Compression",
		Description: "Decompresses data using the Bzip2 algorithm.",
		InfoURL:     "https://wikipedia.org/wiki/Bzip2",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Bzip2Decompress) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Use low-memory, slower decompression algorithm", Flag: "low-memory", Type: core.ArgBoolean, Value: false},
	}
}

// Run decompresses the input.
func (Bzip2Decompress) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// The low-memory switch is accepted and ignored. It picks between two
	// decoders inside libbzip2 that differ in how much they allocate, not in
	// what they produce.

	data := in.Bytes()
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	out, err := bzip2Decode(data)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// bzip2Decode reads every stream the input holds, which is what the bzip2
// command does with several of them written one after another. Anything left
// over once a stream has ended is ignored, again as bzip2 does.
func bzip2Decode(data []byte) ([]byte, error) {
	out, err := io.ReadAll(bzip2.NewReader(bytes.NewReader(data)))
	if err == nil {
		return out, nil
	}

	var structural bzip2.StructuralError
	switch {
	case errors.As(err, &structural) && string(structural) == bzip2TrailingBytes && len(out) > 0:
		return out, nil
	case errors.Is(err, io.ErrUnexpectedEOF):
		return nil, errors.New("truncated bzip2 stream")
	case errors.As(err, &structural) && string(structural) == bzip2BadMagic:
		return nil, errors.New("not a bzip2 stream")
	}
	// Anything else is a stream that began well and then did not hold together,
	// and the reader's own account of it says more than a general message could.
	return nil, err
}

// What the standard library reports for input that never looked like bzip2, and
// for a finished stream followed by something that does not begin another one.
const (
	bzip2BadMagic      = "bad magic value"
	bzip2TrailingBytes = "bad magic value in continuation file"
)

func init() { core.Register(Bzip2Decompress{}) }

// The shape of the format.
const (
	bzGroupSize    = 50 // symbols coded under one table
	bzMaxGroups    = 6  // tables at most
	bzRefineRounds = 4  // passes refining the tables
	bzMaxCodeLen   = 17 // no code may be longer
	bzRunA         = 0  // the two symbols a run of zeros is written with
	bzRunB         = 1
	bzLesserCost   = 0 // the lengths the first, guessed tables are built from
	bzGreaterCost  = 15
	bzMaxRunLength = 255 // as long a repeat as the first stage records at once
	bzMinRun       = 4   // repeats shorter than this are written out as they are
	bzBlockReserve = 19  // headroom the block keeps for the last run it accepts
	bzBlockUnit    = 100000
)

// The markers that open a block, close a stream, and open a stream.
var (
	bzBlockMagic  = []byte{0x31, 0x41, 0x59, 0x26, 0x53, 0x59}
	bzStreamEnd   = []byte{0x17, 0x72, 0x45, 0x38, 0x50, 0x90}
	bzStreamStart = []byte{'B', 'Z', 'h'}
)

// bzCRCTable drives bzip2's checksum, which is CRC-32 taken most significant
// bit first and without the reflection the commoner variant applies.
var bzCRCTable = func() [256]uint32 {
	var table [256]uint32
	for i := range table {
		crc := uint32(i) << 24
		for range 8 {
			if crc&0x80000000 != 0 {
				crc = crc<<1 ^ 0x04c11db7
			} else {
				crc <<= 1
			}
		}
		table[i] = crc
	}
	return table
}()

// bzCRCUpdate folds one byte into a checksum.
func bzCRCUpdate(crc uint32, b byte) uint32 {
	return crc<<8 ^ bzCRCTable[byte(crc>>24)^b]
}

// bzWriter packs bits into bytes, most significant bit first.
type bzWriter struct {
	out  []byte
	buf  uint32
	live uint
}

// writeBits appends the low n bits of value.
func (w *bzWriter) writeBits(n uint, value uint32) {
	for w.live+n >= 8 {
		// #nosec G115 -- the shift leaves at most eight bits, which is what a byte holds
		w.out = append(w.out, byte(w.buf<<(8-w.live)|value>>(n-(8-w.live))))
		n -= 8 - w.live
		value &= 1<<n - 1
		w.live = 0
		w.buf = 0
	}
	w.buf = w.buf<<n | value
	w.live += n
}

// writeByte appends one whole byte.
func (w *bzWriter) writeByte(b byte) { w.writeBits(8, uint32(b)) }

// writeBytes appends several.
func (w *bzWriter) writeBytes(b []byte) {
	for _, c := range b {
		w.writeByte(c)
	}
}

// writeUint32 appends a 32-bit value, most significant byte first.
func (w *bzWriter) writeUint32(v uint32) { w.writeBits(16, v>>16); w.writeBits(16, v&0xffff) }

// finish pads the last byte out with zeros.
func (w *bzWriter) finish() {
	if w.live > 0 {
		w.writeBits(8-w.live, 0)
	}
}

// bzEncoder holds one stream being written.
type bzEncoder struct {
	w         bzWriter
	blockMax  int
	block     []byte
	inUse     [256]bool
	blockCRC  uint32
	streamCRC uint32
	started   bool

	// The run the first stage is part way through, carried between blocks
	// because a repeat that straddles a boundary is not broken by it.
	runByte byte
	runLen  int
	hasRun  bool
}

// bzip2Encode compresses data as a bzip2 stream. blockSize100k is the block
// size in hundreds of kilobytes, 1 to 9.
func bzip2Encode(data []byte, blockSize100k int) []byte {
	e := &bzEncoder{blockMax: bzBlockUnit*blockSize100k - bzBlockReserve}
	e.blockCRC = 0xffffffff

	e.w.writeBytes(bzStreamStart)
	// #nosec G115 -- the block size is bounded to 1..9 by the argument definition
	e.w.writeByte(byte('0' + blockSize100k))

	for _, b := range data {
		if len(e.block) >= e.blockMax {
			e.emitBlock()
		}
		e.addByte(b)
	}
	e.flushRun()
	e.emitBlock()

	e.w.writeBytes(bzStreamEnd)
	e.w.writeUint32(e.streamCRC)
	e.w.finish()
	return e.w.out
}

// addByte feeds one byte through the run-length stage.
func (e *bzEncoder) addByte(b byte) {
	switch {
	case !e.hasRun:
		e.runByte, e.runLen, e.hasRun = b, 1, true
	case b != e.runByte || e.runLen == bzMaxRunLength:
		e.flushRun()
		e.runByte, e.runLen, e.hasRun = b, 1, true
	default:
		e.runLen++
	}
}

// flushRun writes the run in hand into the block. Up to three repeats are
// written as they are; four or more become four bytes and a count of the rest,
// and that count is itself a symbol the block uses.
func (e *bzEncoder) flushRun() {
	if !e.hasRun {
		return
	}
	for range e.runLen {
		e.blockCRC = bzCRCUpdate(e.blockCRC, e.runByte)
	}
	e.inUse[e.runByte] = true
	if e.runLen < bzMinRun {
		for range e.runLen {
			e.block = append(e.block, e.runByte)
		}
	} else {
		e.inUse[e.runLen-bzMinRun] = true
		e.block = append(e.block,
			// #nosec G115 -- a run is at most 255 long, so the count past four fits a byte
			e.runByte, e.runByte, e.runByte, e.runByte, byte(e.runLen-bzMinRun))
	}
	e.hasRun, e.runLen = false, 0
}

// emitBlock writes the block gathered so far, if any, and starts a new one. The
// run in hand is deliberately left alone, so a repeat spanning the boundary
// carries on into the next block.
func (e *bzEncoder) emitBlock() {
	if len(e.block) == 0 {
		return
	}
	e.blockCRC = ^e.blockCRC
	e.streamCRC = e.streamCRC<<1 | e.streamCRC>>31
	e.streamCRC ^= e.blockCRC

	last, origPtr := bzTransform(e.block)

	e.w.writeBytes(bzBlockMagic)
	e.w.writeUint32(e.blockCRC)
	e.w.writeBits(1, 0) // never randomised
	// #nosec G115 -- the origin is a position in a block of at most 900,000 bytes
	e.w.writeBits(24, uint32(origPtr))

	seqToUnseq, unseqToSeq := e.symbolMaps()
	mtf, freq := bzMoveToFront(last, unseqToSeq, len(seqToUnseq))
	bzSendSymbols(&e.w, mtf, freq, len(seqToUnseq), e.inUse)

	e.block = e.block[:0]
	e.inUse = [256]bool{}
	e.blockCRC = 0xffffffff
	e.started = true
}

// symbolMaps lists the byte values the block uses and the reverse lookup from a
// byte value to its place in that list.
func (e *bzEncoder) symbolMaps() (seqToUnseq []byte, unseqToSeq [256]byte) {
	for value := range 256 {
		if e.inUse[value] {
			// #nosec G115 -- at most 256 byte values can be in use
			unseqToSeq[value] = byte(len(seqToUnseq))
			seqToUnseq = append(seqToUnseq, byte(value))
		}
	}
	return seqToUnseq, unseqToSeq
}

// bzTransform applies the Burrows-Wheeler transform, returning the last column
// of the sorted rotations and the row the block itself ended up in.
func bzTransform(block []byte) (last []byte, origPtr int) {
	n := len(block)
	order := bzSortRotations(block)
	last = make([]byte, n)
	for i, start := range order {
		last[i] = block[(start+n-1)%n]
		if start == 0 {
			origPtr = i
		}
	}
	return last, origPtr
}

// bzSortRotations returns the starting positions of the block's rotations in
// sorted order.
//
// This follows libbzip2's fallback sort — an exponential radix sort after
// Manber and Myers — rather than any equivalent ordering, because equivalence
// is not enough. When a block repeats with a period dividing its length, whole
// groups of rotations are identical, and which of them is called row zero goes
// into the stream as the origin pointer. libbzip2 has a faster sort as well,
// but its own comment records that the two produce the same stream, so only
// this one is needed.
func bzSortRotations(block []byte) []int {
	n := len(block)
	fmap := make([]uint32, n)
	eclass := make([]uint32, n)
	// Room for the sentinel bits written past the end of the block below.
	heads := make([]uint32, (n+128)/32+4)

	bzRadixSortByFirstByte(block, fmap, heads)

	// Sentinels past the end of the block, alternating so that the scan for the
	// next bucket always terminates.
	for i := range 32 {
		bzSetHead(heads, n+2*i)
		bzClearHead(heads, n+2*i+1)
	}

	// The width doubles after each pass, and the pass that reaches the block
	// length is the last one, so the width never exceeds it.
	for h := 1; ; {
		// Give every rotation the bucket its predecessor h places back sits in.
		bucket := 0
		for i := range n {
			if bzIsHead(heads, i) {
				bucket = i
			}
			k := int(fmap[i]) - h
			if k < 0 {
				k += n
			}
			eclass[k] = uint32(bucket)
		}

		unresolved := bzRefineBuckets(fmap, eclass, heads, n)
		h *= 2
		if h > n || unresolved == 0 {
			break
		}
	}

	order := make([]int, n)
	for i, start := range fmap {
		order[i] = int(start)
	}
	return order
}

// bzRadixSortByFirstByte lays the rotations out in order of the byte they start
// with, and marks where each bucket begins. Within a bucket the later starting
// positions come first, which is what filling each bucket from its end does.
func bzRadixSortByFirstByte(block []byte, fmap []uint32, heads []uint32) {
	var ftab [257]int32
	for _, b := range block {
		ftab[b]++
	}
	for i := 1; i < 257; i++ {
		ftab[i] += ftab[i-1]
	}
	for i, b := range block {
		ftab[b]--
		fmap[ftab[b]] = uint32(i)
	}
	for i := range 256 {
		bzSetHead(heads, int(ftab[i]))
	}
}

// bzRefineBuckets sorts each bucket holding more than one rotation by the class
// of its predecessor, splitting it into smaller buckets, and reports how many
// rotations were still sharing a bucket when it started.
func bzRefineBuckets(fmap, eclass, heads []uint32, n int) int {
	unresolved := 0
	right := -1
	for {
		bucket, ok := bzNextBucket(heads, right, n)
		if !ok {
			return unresolved
		}
		left := bucket.from
		right = bucket.to
		// The scan never yields a single-entry bucket: it stops at the last of a
		// run of bucket heads, so the position after the one it returns is always
		// inside the same bucket.
		unresolved += right - left + 1
		bzQuickSortByClass(fmap, eclass, left, right)
		last := int64(-1)
		for i := left; i <= right; i++ {
			cls := int64(eclass[fmap[i]])
			if cls != last {
				bzSetHead(heads, i)
				last = cls
			}
		}
	}
}

// bzBucket is a range of equal rotations found by the scan.
type bzBucket struct{ from, to int }

// bzNextBucket finds the bucket after the one ending at prev, walking the
// bucket-head bits a word at a time where it can.
func bzNextBucket(heads []uint32, prev, n int) (bzBucket, bool) {
	k := prev + 1
	for bzIsHead(heads, k) && k&31 != 0 {
		k++
	}
	if bzIsHead(heads, k) {
		for heads[k>>5] == 0xffffffff {
			k += 32
		}
		for bzIsHead(heads, k) {
			k++
		}
	}
	from := k - 1
	for !bzIsHead(heads, k) && k&31 != 0 {
		k++
	}
	if !bzIsHead(heads, k) {
		for heads[k>>5] == 0 {
			k += 32
		}
		for !bzIsHead(heads, k) {
			k++
		}
	}
	to := k - 1
	if from >= n || to >= n {
		return bzBucket{}, false
	}
	return bzBucket{from: from, to: to}, true
}

// The bucket-head bits, one per position.
func bzSetHead(heads []uint32, i int)   { heads[i>>5] |= 1 << (i & 31) }
func bzClearHead(heads []uint32, i int) { heads[i>>5] &^= 1 << (i & 31) }
func bzIsHead(heads []uint32, i int) bool {
	return heads[i>>5]&(1<<(i&31)) != 0
}

// bzQuickSortSmall is the size below which the quicksort hands over to an
// insertion sort, and bzQuickSortStack is how deep its own stack can go.
const (
	bzQuickSortSmall = 10
	bzQuickSortStack = 100
)

// bzQuickSortByClass sorts fmap[lo..hi] by the class of each entry. It is a
// three-way partition whose pivot is chosen by a small generator seeded afresh
// on every call, so the result is settled entirely by the input — including the
// order it leaves entries of equal class in, which is what has to be matched.
func bzQuickSortByClass(fmap, eclass []uint32, loSt, hiSt int) {
	var stackLo, stackHi [bzQuickSortStack]int
	sp := 0
	push := func(lo, hi int) { stackLo[sp], stackHi[sp] = lo, hi; sp++ }

	rand := uint32(0)
	push(loSt, hiSt)
	for sp > 0 {
		sp--
		lo, hi := stackLo[sp], stackHi[sp]
		if hi-lo < bzQuickSortSmall {
			bzInsertionSortByClass(fmap, eclass, lo, hi)
			continue
		}

		rand = (rand*7621 + 1) % 32768
		var pivot uint32
		switch rand % 3 {
		case 0:
			pivot = eclass[fmap[lo]]
		case 1:
			pivot = eclass[fmap[(lo+hi)>>1]]
		default:
			pivot = eclass[fmap[hi]]
		}

		unLo, ltLo := lo, lo
		unHi, gtHi := hi, hi
		for {
			for unLo <= unHi {
				n := int64(eclass[fmap[unLo]]) - int64(pivot)
				if n == 0 {
					fmap[unLo], fmap[ltLo] = fmap[ltLo], fmap[unLo]
					ltLo++
					unLo++
					continue
				}
				if n > 0 {
					break
				}
				unLo++
			}
			for unLo <= unHi {
				n := int64(eclass[fmap[unHi]]) - int64(pivot)
				if n == 0 {
					fmap[unHi], fmap[gtHi] = fmap[gtHi], fmap[unHi]
					gtHi--
					unHi--
					continue
				}
				if n < 0 {
					break
				}
				unHi--
			}
			if unLo > unHi {
				break
			}
			fmap[unLo], fmap[unHi] = fmap[unHi], fmap[unLo]
			unLo++
			unHi--
		}
		if gtHi < ltLo {
			continue
		}

		// Move the entries equal to the pivot into the middle.
		bzSwapRange(fmap, lo, unLo-min(ltLo-lo, unLo-ltLo), min(ltLo-lo, unLo-ltLo))
		bzSwapRange(fmap, unLo, hi-min(hi-gtHi, gtHi-unHi)+1, min(hi-gtHi, gtHi-unHi))

		n := lo + unLo - ltLo - 1
		m := hi - (gtHi - unHi) + 1
		if n-lo > hi-m {
			push(lo, n)
			push(m, hi)
		} else {
			push(m, hi)
			push(lo, n)
		}
	}
}

// bzSwapRange exchanges count entries starting at each of two positions.
func bzSwapRange(fmap []uint32, a, b, count int) {
	for range count {
		fmap[a], fmap[b] = fmap[b], fmap[a]
		a++
		b++
	}
}

// bzInsertionSortByClass sorts a short run, first at a stride of four and then
// at a stride of one, which is how libbzip2 does it.
func bzInsertionSortByClass(fmap, eclass []uint32, lo, hi int) {
	if lo == hi {
		return
	}
	if hi-lo > 3 {
		for i := hi - 4; i >= lo; i-- {
			held := fmap[i]
			cls := eclass[held]
			j := i + 4
			for ; j <= hi && cls > eclass[fmap[j]]; j += 4 {
				fmap[j-4] = fmap[j]
			}
			fmap[j-4] = held
		}
	}
	for i := hi - 1; i >= lo; i-- {
		held := fmap[i]
		cls := eclass[held]
		j := i + 1
		for ; j <= hi && cls > eclass[fmap[j]]; j++ {
			fmap[j-1] = fmap[j]
		}
		fmap[j-1] = held
	}
}

// bzMoveToFront runs the move-to-front and second run-length stages together,
// returning the symbols and how often each occurs. A byte that is already at
// the front of the list adds to a run of zeros, and such a run is written as a
// number in the two digits RUNA and RUNB rather than as that many symbols.
func bzMoveToFront(last []byte, unseqToSeq [256]byte, inUseCount int) (mtf []uint16, freq []int32) {
	eob := inUseCount + 1
	freq = make([]int32, eob+1)
	mtf = make([]uint16, 0, len(last)+1)

	recent := make([]byte, inUseCount)
	for i := range recent {
		recent[i] = byte(i)
	}

	zeros := 0
	flushZeros := func() {
		if zeros == 0 {
			return
		}
		for n := zeros - 1; ; n = (n - 2) / 2 {
			symbol := uint16(bzRunA)
			if n&1 == 1 {
				symbol = bzRunB
			}
			mtf = append(mtf, symbol)
			freq[symbol]++
			if n < 2 {
				break
			}
		}
		zeros = 0
	}

	for _, b := range last {
		symbol := unseqToSeq[b]
		if recent[0] == symbol {
			zeros++
			continue
		}
		flushZeros()
		// Move the symbol to the front, and write down how far it had to come.
		j := 1
		prev := recent[0]
		for recent[j] != symbol {
			recent[j], prev = prev, recent[j]
			j++
		}
		recent[j] = prev
		recent[0] = symbol
		mtf = append(mtf, uint16(j+1))
		freq[j+1]++
	}
	flushZeros()

	// #nosec G115 -- the end marker is at most 257, the largest symbol the format has
	mtf = append(mtf, uint16(eob))
	freq[eob]++
	return mtf, freq
}

// bzCodeLengths settles the Huffman code lengths for one table, the way
// libbzip2 does: build a tree over weights that carry each node's depth in
// their low byte, so that ties break towards the shallower node, and if any
// code still comes out too long, flatten the weights and try again.
func bzCodeLengths(freq []int32, alphaSize, maxLen int) []byte {
	weight := make([]int64, alphaSize*2+2)
	parent := make([]int32, alphaSize*2+2)

	for i := range alphaSize {
		w := int64(freq[i])
		if w == 0 {
			w = 1
		}
		weight[i+1] = w << 8
	}

	for {
		bzBuildTree(weight, parent, alphaSize)
		lengths, tooLong := bzTreeDepths(parent, alphaSize, maxLen)
		if !tooLong {
			return lengths
		}
		// Halve the spread between the weights and build again, which shortens
		// the deepest codes at the cost of a slightly worse fit.
		for i := 1; i <= alphaSize; i++ {
			weight[i] = (1 + weight[i]>>8/2) << 8
		}
	}
}

// bzBuildTree joins the symbols into a Huffman tree, filling in parent. Each
// weight carries the depth of its node in the low byte, so that when two nodes
// weigh the same the shallower one is taken first and the tree stays even.
func bzBuildTree(weight []int64, parent []int32, alphaSize int) {
	heap := make([]int32, alphaSize+2)
	size := 0
	nodes := alphaSize

	up := func(pos int) {
		node := heap[pos]
		for weight[node] < weight[heap[pos>>1]] {
			heap[pos] = heap[pos>>1]
			pos >>= 1
		}
		heap[pos] = node
	}
	down := func(pos int) {
		node := heap[pos]
		for {
			child := pos << 1
			if child > size {
				break
			}
			if child < size && weight[heap[child+1]] < weight[heap[child]] {
				child++
			}
			if weight[node] < weight[heap[child]] {
				break
			}
			heap[pos] = heap[child]
			pos = child
		}
		heap[pos] = node
	}
	take := func() int32 {
		top := heap[1]
		heap[1] = heap[size]
		size--
		down(1)
		return top
	}

	heap[0], weight[0], parent[0] = 0, 0, -2
	for i := 1; i <= alphaSize; i++ {
		parent[i] = -1
		size++
		heap[size] = int32(i)
		up(size)
	}

	for size > 1 {
		a, b := take(), take()
		nodes++
		// #nosec G115 -- a tree over 258 symbols has fewer than 517 nodes
		parent[a], parent[b] = int32(nodes), int32(nodes)
		weight[nodes] = bzAddWeights(weight[a], weight[b])
		parent[nodes] = -1
		size++
		// #nosec G115 -- a tree over 258 symbols has fewer than 517 nodes
		heap[size] = int32(nodes)
		up(size)
	}
}

// bzTreeDepths reads each symbol's code length off the tree by counting the
// steps from it up to the root, and reports whether any came out over the
// limit.
func bzTreeDepths(parent []int32, alphaSize, maxLen int) (lengths []byte, tooLong bool) {
	lengths = make([]byte, alphaSize)
	for i := 1; i <= alphaSize; i++ {
		depth := 0
		for node := int32(i); parent[node] >= 0; node = parent[node] {
			depth++
		}
		lengths[i-1] = byte(depth)
		if depth > maxLen {
			tooLong = true
		}
	}
	return lengths, tooLong
}

// bzAddWeights joins two nodes: the weights add, and the depth becomes one more
// than the deeper of the two.
func bzAddWeights(a, b int64) int64 {
	depth := max(b&0xff, a&0xff)
	return (a&^0xff + b&^0xff) | (1 + depth)
}

// bzAssignCodes gives each symbol its code, shortest length first and in
// symbol order within a length, which is the canonical arrangement the decoder
// rebuilds from the lengths alone.
func bzAssignCodes(lengths []byte, minLen, maxLen int) []int32 {
	codes := make([]int32, len(lengths))
	code := int32(0)
	for n := minLen; n <= maxLen; n++ {
		for i, l := range lengths {
			if int(l) == n {
				codes[i] = code
				code++
			}
		}
		code <<= 1
	}
	return codes
}

// bzSendSymbols writes everything after the block header: which byte values the
// block uses, the Huffman tables, which table covers each group of fifty
// symbols, and the symbols themselves.
func bzSendSymbols(w *bzWriter, mtf []uint16, freq []int32, inUseCount int, inUse [256]bool) {
	alphaSize := inUseCount + 2
	groups := bzGroupCount(len(mtf))
	lengths := bzInitialTables(freq, groups, alphaSize, len(mtf))
	selectors := bzRefineTables(mtf, lengths, groups, alphaSize)

	codes := make([][]int32, groups)
	for t := range groups {
		minLen, maxLen := 32, 0
		for _, l := range lengths[t] {
			if int(l) > maxLen {
				maxLen = int(l)
			}
			if int(l) < minLen {
				minLen = int(l)
			}
		}
		codes[t] = bzAssignCodes(lengths[t], minLen, maxLen)
	}

	bzWriteSymbolMap(w, inUse)
	// #nosec G115 -- the table count is 2 to 6
	w.writeBits(3, uint32(groups))
	// #nosec G115 -- there is one selector per fifty symbols of a bounded block
	w.writeBits(15, uint32(len(selectors)))
	bzWriteSelectors(w, selectors, groups)
	bzWriteCodeLengths(w, lengths, alphaSize)
	bzWriteCodedSymbols(w, mtf, selectors, lengths, codes)
}

// bzGroupCount decides how many Huffman tables to use, on how many symbols
// there are to spread over them.
func bzGroupCount(symbols int) int {
	switch {
	case symbols < 200:
		return 2
	case symbols < 600:
		return 3
	case symbols < 1200:
		return 4
	case symbols < 2400:
		return 5
	}
	return bzMaxGroups
}

// bzInitialTables guesses a first set of tables by cutting the alphabet into
// bands of roughly equal weight, one per table, and making each table cheap
// over its own band and dear everywhere else. The bands are laid out from the
// last table backwards, and every other one gives back the symbol that took it
// over its share, which spreads the boundaries a little more evenly.
func bzInitialTables(freq []int32, groups, alphaSize, symbols int) [][]byte {
	lengths := make([][]byte, groups)
	for t := range lengths {
		lengths[t] = make([]byte, alphaSize)
		for v := range lengths[t] {
			lengths[t][v] = bzGreaterCost
		}
	}

	// #nosec G115 -- the symbol count is bounded by the block size
	remaining := int32(symbols)
	low := 0
	for part := groups; part > 0; part-- {
		// #nosec G115 -- the table count is 2 to 6
		target := remaining / int32(part)
		high := low - 1
		var weight int32
		for weight < target && high < alphaSize-1 {
			high++
			weight += freq[high]
		}
		if high > low && part != groups && part != 1 && (groups-part)%2 == 1 {
			weight -= freq[high]
			high--
		}
		for v := low; v <= high; v++ {
			lengths[part-1][v] = bzLesserCost
		}
		low = high + 1
		remaining -= weight
	}
	return lengths
}

// bzRefineTables improves the tables by turns: each pass gives every group of
// fifty symbols to whichever table codes it most cheaply, then rebuilds each
// table from the symbols it was given. The selectors returned are the ones the
// last pass chose, which is what the final table rebuild is measured against —
// tables and selectors are therefore one step out of step, exactly as
// libbzip2 leaves them.
func bzRefineTables(mtf []uint16, lengths [][]byte, groups, alphaSize int) []byte {
	var selectors []byte
	counts := make([][]int32, groups)
	for t := range counts {
		counts[t] = make([]int32, alphaSize)
	}

	for range bzRefineRounds {
		for t := range counts {
			for v := range counts[t] {
				counts[t][v] = 0
			}
		}
		selectors = selectors[:0]

		for start := 0; start < len(mtf); start += bzGroupSize {
			end := min(start+bzGroupSize, len(mtf))

			best, bestCost := 0, int32(1<<30)
			for t := range groups {
				var cost int32
				for _, symbol := range mtf[start:end] {
					cost += int32(lengths[t][symbol])
				}
				if cost < bestCost {
					best, bestCost = t, cost
				}
			}
			selectors = append(selectors, byte(best))
			for _, symbol := range mtf[start:end] {
				counts[best][symbol]++
			}
		}

		for t := range groups {
			lengths[t] = bzCodeLengths(counts[t], alphaSize, bzMaxCodeLen)
		}
	}
	return selectors
}

// bzWriteSymbolMap writes which byte values the block uses, as sixteen flags
// saying which runs of sixteen values are represented at all, followed by the
// values themselves for each run that is.
func bzWriteSymbolMap(w *bzWriter, inUse [256]bool) {
	var used [16]bool
	for i := range used {
		for j := range 16 {
			if inUse[i*16+j] {
				used[i] = true
				break
			}
		}
	}
	for _, u := range used {
		w.writeBits(1, bzBit(u))
	}
	for i, u := range used {
		if !u {
			continue
		}
		for j := range 16 {
			w.writeBits(1, bzBit(inUse[i*16+j]))
		}
	}
}

// bzWriteSelectors writes the table numbers, each moved to the front of a list
// of the tables and written as that many ones and a zero, so that repeatedly
// choosing the same table costs one bit a time.
func bzWriteSelectors(w *bzWriter, selectors []byte, groups int) {
	order := make([]byte, groups)
	for i := range order {
		order[i] = byte(i)
	}
	for _, want := range selectors {
		j := 0
		held := order[0]
		for want != held {
			j++
			held, order[j] = order[j], held
		}
		order[0] = held
		for range j {
			w.writeBits(1, 1)
		}
		w.writeBits(1, 0)
	}
}

// bzWriteCodeLengths writes each table's code lengths as a starting length and
// then, for every symbol, the steps up or down needed to reach the next one.
func bzWriteCodeLengths(w *bzWriter, lengths [][]byte, alphaSize int) {
	for _, table := range lengths {
		curr := int(table[0])
		// #nosec G115 -- a code length is at most 17, and five bits are written
		w.writeBits(5, uint32(curr))
		for i := range alphaSize {
			for curr < int(table[i]) {
				w.writeBits(2, 2)
				curr++
			}
			for curr > int(table[i]) {
				w.writeBits(2, 3)
				curr--
			}
			w.writeBits(1, 0)
		}
	}
}

// bzWriteCodedSymbols writes the symbols themselves, fifty at a time under
// whichever table the matching selector names.
func bzWriteCodedSymbols(w *bzWriter, mtf []uint16, selectors []byte, lengths [][]byte, codes [][]int32) {
	for i, start := 0, 0; start < len(mtf); i, start = i+1, start+bzGroupSize {
		end := min(start+bzGroupSize, len(mtf))
		table := selectors[i]
		for _, symbol := range mtf[start:end] {
			// #nosec G115 -- a code is at most 17 bits wide
			w.writeBits(uint(lengths[table][symbol]), uint32(codes[table][symbol]))
		}
	}
}

// bzBit renders a flag as the bit that stands for it.
func bzBit(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}
