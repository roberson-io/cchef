package ops

import (
	"bytes"
	"compress/flate"
	"errors"
	"io"

	"github.com/roberson-io/cchef/core"
)

// The DEFLATE codec, and the two operations that use it without any container
// around it. Zlib Deflate and Gzip wrap the same writer; Zlib Inflate and
// Gunzip the same reader.
//
// The writer here is not the one in deflate.go. That one reproduces zlib's own
// encoder, for operations that embed a compressed stream inside a file format.
// This one reproduces zlibjs, which CyberChef's compression operations use, and
// the two disagree: zlibjs takes the longest repeat anywhere in the window,
// where zlib walks a bounded chain and holds a repeat back while it looks for a
// better one. Both write valid DEFLATE; they do not write the same bytes, and
// CyberChef's fixtures pin zlibjs's.

// deflateCompressionTypes are the block encodings a writer can be asked for.
var deflateCompressionTypes = []string{"Dynamic Huffman Coding", "Fixed Huffman Coding", "None (Store)"}

// deflateBufferTypes are the ways a reader can be asked to grow its working
// buffer. Neither has any bearing on what it decodes.
var deflateBufferTypes = []string{"Adaptive", "Block"}

// The three block encodings, in the order the two-bit block header numbers them.
const (
	rdfBlockStored  = 0
	rdfBlockFixed   = 1
	rdfBlockDynamic = 2
)

// The shape of the format.
const (
	rdfMinMatch    = 3     // no shorter repeat is worth recording
	rdfMaxMatch    = 258   // nor any longer one recordable
	rdfWindow      = 32768 // how far back a repeat may reach
	rdfStoredMax   = 65535 // as much as one stored block can hold
	rdfEndOfBlock  = 256   // the symbol closing a block
	rdfLitLenCount = 286   // literal and length symbols
	rdfDistCount   = 30    // distance symbols
	rdfTreeCount   = 19    // symbols describing the code lengths themselves
	rdfMaxCodeLen  = 16    // no code may be longer
	rdfLitLenLimit = 15    // nor any literal or length code
	rdfDistLimit   = 7     // nor any distance or code-length code
)

// rdfTreeOrder is the order the code-length code lengths are written in, chosen
// so the ones most often zero come last and can be left out altogether.
var rdfTreeOrder = [rdfTreeCount]int{16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15}

// The bases and extra-bit counts of the length and distance codes (RFC 1951).
var (
	rdfLengthBase  = [29]int{3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258}
	rdfLengthExtra = [29]int{0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0}
	rdfDistBase    = [30]int{1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577}
	rdfDistExtra   = [30]int{0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13}
)

// rdfWriter packs bits into bytes. DEFLATE fills each byte from its least
// significant bit upwards, so a value written across a boundary continues in
// the next byte's low bits.
type rdfWriter struct {
	out   []byte
	buf   uint32
	count uint
}

// writeBits appends the low n bits of value, least significant first, which is
// how the format writes its header fields and the extra bits after a length or
// distance code.
func (w *rdfWriter) writeBits(value uint32, n uint) {
	w.buf |= (value & (1<<n - 1)) << w.count
	w.count += n
	for w.count >= 8 {
		// #nosec G115 -- the buffer's low byte is what is being written out
		w.out = append(w.out, byte(w.buf))
		w.buf >>= 8
		w.count -= 8
	}
}

// writeCode appends a Huffman code, most significant bit first, which is the
// one place the format reverses the usual order.
func (w *rdfWriter) writeCode(code uint32, n uint) {
	for i := n; i > 0; i-- {
		w.writeBits(code>>(i-1)&1, 1)
	}
}

// finish pads the last byte out with zeros.
func (w *rdfWriter) finish() []byte {
	if w.count > 0 {
		// #nosec G115 -- the buffer's low byte is what is being written out
		w.out = append(w.out, byte(w.buf))
		w.buf, w.count = 0, 0
	}
	return w.out
}

// deflateEncode compresses data as a raw DEFLATE stream under the block
// encoding named.
func deflateEncode(data []byte, compressionType string) ([]byte, error) {
	switch compressionType {
	case "None (Store)":
		return rdfStoredBlocks(data), nil
	case "Fixed Huffman Coding":
		return rdfHuffmanBlock(data, rdfBlockFixed), nil
	case "Dynamic Huffman Coding":
		return rdfHuffmanBlock(data, rdfBlockDynamic), nil
	}
	return nil, errors.New("invalid compression type")
}

// rdfStoredBlocks writes the data as it is, in blocks of at most 65535 bytes,
// each headed by its length and that length's complement.
func rdfStoredBlocks(data []byte) []byte {
	if len(data) == 0 {
		// One final, empty block — the shortest stream a reader will accept.
		return []byte{1, 0, 0, 0xff, 0xff}
	}
	out := make([]byte, 0, len(data)+5*(len(data)/rdfStoredMax+1))
	for position := 0; position < len(data); {
		block := data[position:min(position+rdfStoredMax, len(data))]
		position += len(block)

		final := byte(0)
		if position == len(data) {
			final = 1
		}
		length := len(block)
		out = append(out, final|rdfBlockStored<<1,
			// #nosec G115 -- a stored block holds at most 65535 bytes, so the length and its complement fit two bytes each
			byte(length), byte(length>>8), byte(^length), byte(^length>>8))
		out = append(out, block...)
	}
	return out
}

// rdfHuffmanBlock writes the whole input as one Huffman-coded block.
func rdfHuffmanBlock(data []byte, blockType int) []byte {
	w := &rdfWriter{}
	w.writeBits(1, 1) // the only block, so the final one
	// #nosec G115 -- the block encoding is one of three small values
	w.writeBits(uint32(blockType), 2)

	tokens, litLenFreq, distFreq := rdfTokenize(data)
	if blockType == rdfBlockFixed {
		rdfWriteFixed(w, tokens)
	} else {
		rdfWriteDynamic(w, tokens, litLenFreq, distFreq)
	}
	return w.finish()
}

// rdfTokenize turns the input into the literals and repeats the Huffman stage
// codes, with a count of how often each symbol occurs.
//
// Repeats are found by keeping, for every three bytes seen, the list of places
// they have appeared, and taking the longest match anywhere in that list. zlibjs
// can be asked to hold a match back and prefer a longer one starting a byte
// later, but nothing in CyberChef turns that on, so only the greedy path is
// reproduced.
//
// Each repeat takes six entries: the length code, its extra bits and how many,
// then the same three for the distance.
func rdfTokenize(data []byte) (tokens []uint16, litLenFreq, distFreq []int32) {
	litLenFreq = make([]int32, rdfLitLenCount)
	distFreq = make([]int32, rdfDistCount)
	// The end-of-block symbol is counted once before anything is read and again
	// when it is written, leaving it a frequency of two. That is zlibjs's doing,
	// and reproducing it is what keeps the Huffman tree the same shape.
	litLenFreq[rdfEndOfBlock] = 1

	tokens = make([]uint16, 0, len(data))
	seen := make(map[uint32][]int32, len(data)/2+1)
	skip := 0

	position := 0
	for ; position < len(data); position++ {
		key := rdfKeyAt(data, position)
		list := seen[key]

		if skip > 0 {
			skip--
			seen[key] = append(list, int32(position))
			continue
		}
		for len(list) > 0 && position-int(list[0]) > rdfWindow {
			list = list[1:]
		}
		if position+rdfMinMatch >= len(data) {
			seen[key] = list
			break
		}

		if len(list) > 0 {
			length, distance := rdfLongestMatch(data, position, list)
			tokens = rdfAppendMatch(tokens, litLenFreq, distFreq, length, distance)
			skip = length - 1
		} else {
			tokens = append(tokens, uint16(data[position]))
			litLenFreq[data[position]]++
		}
		seen[key] = append(list, int32(position))
	}

	// Whatever is left is too short to hold a repeat.
	for ; position < len(data); position++ {
		tokens = append(tokens, uint16(data[position]))
		litLenFreq[data[position]]++
	}

	tokens = append(tokens, rdfEndOfBlock)
	litLenFreq[rdfEndOfBlock]++
	return tokens, litLenFreq, distFreq
}

// rdfKeyAt reads the three bytes at position as one number, taking fewer when
// the input ends first.
func rdfKeyAt(data []byte, position int) uint32 {
	var key uint32
	for i := range rdfMinMatch {
		if position+i == len(data) {
			break
		}
		key = key<<8 | uint32(data[position+i])
	}
	return key
}

// rdfLongestMatch returns the longest repeat starting at position, looking at
// the recorded places newest first. A candidate is abandoned as soon as its
// last byte fails to match the best found so far, and the search stops early
// once a repeat as long as the format allows turns up.
func rdfLongestMatch(data []byte, position int, list []int32) (length, distance int) {
	best, bestStart := 0, 0
	for i := len(list) - 1; i >= 0; i-- {
		start := int(list[i])
		matched := rdfMinMatch
		if best > rdfMinMatch {
			if !rdfTailMatches(data, start, position, best) {
				continue
			}
			matched = best
		}
		for matched < rdfMaxMatch && position+matched < len(data) &&
			data[start+matched] == data[position+matched] {
			matched++
		}
		if matched > best {
			best, bestStart = matched, start
		}
		if matched == rdfMaxMatch {
			break
		}
	}
	return best, position - bestStart
}

// rdfTailMatches reports whether a candidate agrees with the current position
// everywhere the best match so far reaches, checked from its far end because
// that is where a candidate is most likely to fail.
func rdfTailMatches(data []byte, start, position, best int) bool {
	for j := best; j > rdfMinMatch; j-- {
		if data[start+j-1] != data[position+j-1] {
			return false
		}
	}
	return true
}

// rdfAppendMatch writes one repeat into the token stream and counts its symbols.
func rdfAppendMatch(tokens []uint16, litLenFreq, distFreq []int32, length, distance int) []uint16 {
	lengthCode, lengthExtra, lengthBits := rdfLengthCode(length)
	distCode, distExtra, distBits := rdfDistanceCode(distance)
	tokens = append(tokens,
		// #nosec G115 -- a length code is at most 285 and its extra bits at most five
		uint16(lengthCode), uint16(lengthExtra), uint16(lengthBits),
		// #nosec G115 -- a distance code is at most 29 and its extra bits at most thirteen
		uint16(distCode), uint16(distExtra), uint16(distBits))
	litLenFreq[lengthCode]++
	distFreq[distCode]++
	return tokens
}

// rdfLengthCode returns the symbol standing for a repeat of that length, the
// value of the extra bits following it, and how many there are.
func rdfLengthCode(length int) (code, extra, bits int) {
	i := len(rdfLengthBase) - 1
	for rdfLengthBase[i] > length {
		i--
	}
	return 257 + i, length - rdfLengthBase[i], rdfLengthExtra[i]
}

// rdfDistanceCode does the same for how far back the repeat reaches.
func rdfDistanceCode(distance int) (code, extra, bits int) {
	i := len(rdfDistBase) - 1
	for rdfDistBase[i] > distance {
		i--
	}
	return i, distance - rdfDistBase[i], rdfDistExtra[i]
}

// rdfFixedLength and rdfFixedCode give the tables the format defines, which need
// not be sent because both sides already know them.
func rdfFixedLength(symbol int) uint {
	switch {
	case symbol < 144:
		return 8
	case symbol < 256:
		return 9
	case symbol < 280:
		return 7
	}
	return 8
}

func rdfFixedCode(symbol int) uint32 {
	switch {
	case symbol < 144:
		// #nosec G115 -- the fixed code for a literal is at most 511
		return uint32(0x30 + symbol)
	case symbol < 256:
		return uint32(0x190 + symbol - 144)
	case symbol < 280:
		return uint32(symbol - 256)
	}
	// #nosec G115 -- the fixed code for a length symbol is at most 199
	return uint32(0xc0 + symbol - 280)
}

// rdfWriteFixed writes the tokens under those tables.
func rdfWriteFixed(w *rdfWriter, tokens []uint16) {
	for i := 0; i < len(tokens); i++ {
		symbol := int(tokens[i])
		w.writeCode(rdfFixedCode(symbol), rdfFixedLength(symbol))
		if symbol <= rdfEndOfBlock {
			if symbol == rdfEndOfBlock {
				break
			}
			continue
		}
		w.writeBits(uint32(tokens[i+1]), uint(tokens[i+2]))
		w.writeCode(uint32(tokens[i+3]), 5)
		w.writeBits(uint32(tokens[i+4]), uint(tokens[i+5]))
		i += 5
	}
}

// rdfWriteDynamic works out tables fitted to this block, sends them, and writes
// the tokens under them.
func rdfWriteDynamic(w *rdfWriter, tokens []uint16, litLenFreq, distFreq []int32) {
	litLenLengths := rdfCodeLengths(litLenFreq, rdfLitLenLimit)
	litLenCodes := rdfCanonicalCodes(litLenLengths)
	distLengths := rdfCodeLengths(distFreq, rdfDistLimit)
	distCodes := rdfCanonicalCodes(distLengths)

	hlit := rdfLitLenCount
	for hlit > 257 && litLenLengths[hlit-1] == 0 {
		hlit--
	}
	hdist := rdfDistCount
	for hdist > 1 && distLengths[hdist-1] == 0 {
		hdist--
	}

	treeSymbols, treeFreq := rdfPackLengths(litLenLengths[:hlit], distLengths[:hdist])
	treeLengths := rdfCodeLengths(treeFreq, rdfDistLimit)
	treeCodes := rdfCanonicalCodes(treeLengths)

	var ordered [rdfTreeCount]byte
	for i, at := range rdfTreeOrder {
		ordered[i] = treeLengths[at]
	}
	hclen := rdfTreeCount
	for hclen > 4 && ordered[hclen-1] == 0 {
		hclen--
	}

	w.writeBits(uint32(hlit-257), 5)
	w.writeBits(uint32(hdist-1), 5)
	w.writeBits(uint32(hclen-4), 4)
	for i := range hclen {
		w.writeBits(uint32(ordered[i]), 3)
	}
	rdfWriteTreeSymbols(w, treeSymbols, treeCodes, treeLengths)

	for i := 0; i < len(tokens); i++ {
		symbol := int(tokens[i])
		w.writeCode(litLenCodes[symbol], uint(litLenLengths[symbol]))
		if symbol <= rdfEndOfBlock {
			if symbol == rdfEndOfBlock {
				break
			}
			continue
		}
		w.writeBits(uint32(tokens[i+1]), uint(tokens[i+2]))
		code := int(tokens[i+3])
		w.writeCode(distCodes[code], uint(distLengths[code]))
		w.writeBits(uint32(tokens[i+4]), uint(tokens[i+5]))
		i += 5
	}
}

// rdfWriteTreeSymbols sends the packed code lengths, each repeat marker followed
// by the count it stands for.
func rdfWriteTreeSymbols(w *rdfWriter, symbols []int, codes []uint32, lengths []byte) {
	for i := 0; i < len(symbols); i++ {
		symbol := symbols[i]
		w.writeCode(codes[symbol], uint(lengths[symbol]))
		switch symbol {
		case 16:
			i++
			// #nosec G115 -- a repeat count is written in two bits
			w.writeBits(uint32(symbols[i]), 2)
		case 17:
			i++
			// #nosec G115 -- a zero-run count is written in three bits
			w.writeBits(uint32(symbols[i]), 3)
		case 18:
			i++
			// #nosec G115 -- a long zero-run count is written in seven bits
			w.writeBits(uint32(symbols[i]), 7)
		}
	}
}

// rdfPackLengths turns the two tables' code lengths into the symbols describing
// them: the lengths themselves, plus markers standing for a run of the previous
// length or a run of zeros.
func rdfPackLengths(litLenLengths, distLengths []byte) (symbols []int, freq []int32) {
	src := make([]byte, 0, len(litLenLengths)+len(distLengths))
	src = append(src, litLenLengths...)
	src = append(src, distLengths...)

	freq = make([]int32, rdfTreeCount)
	for i := 0; i < len(src); {
		run := 1
		for i+run < len(src) && src[i+run] == src[i] {
			run++
		}
		if src[i] == 0 {
			symbols, freq = rdfPackZeroRun(symbols, freq, run)
		} else {
			symbols, freq = rdfPackValueRun(symbols, freq, int(src[i]), run)
		}
		i += run
	}
	return symbols, freq
}

// rdfPackZeroRun writes a run of unused symbols: as themselves when there are
// fewer than three, and otherwise as one of the two zero-run markers.
func rdfPackZeroRun(symbols []int, freq []int32, run int) ([]int, []int32) {
	if run < 3 {
		for range run {
			symbols = append(symbols, 0)
			freq[0]++
		}
		return symbols, freq
	}
	for run > 0 {
		take := min(run, 138)
		// Never leave a remainder of one or two behind, which would cost more to
		// write than shortening this run does.
		if take > run-3 && take < run {
			take = run - 3
		}
		if take <= 10 {
			symbols = append(symbols, 17, take-3)
			freq[17]++
		} else {
			symbols = append(symbols, 18, take-11)
			freq[18]++
		}
		run -= take
	}
	return symbols, freq
}

// rdfPackValueRun writes a run of one length: the length itself, then the rest
// as repeat markers.
func rdfPackValueRun(symbols []int, freq []int32, value, run int) ([]int, []int32) {
	symbols = append(symbols, value)
	freq[value]++
	run--

	if run < 3 {
		for range run {
			symbols = append(symbols, value)
			freq[value]++
		}
		return symbols, freq
	}
	for run > 0 {
		take := min(run, 6)
		if take > run-3 && take < run {
			take = run - 3
		}
		symbols = append(symbols, 16, take-3)
		freq[16]++
		run -= take
	}
	return symbols, freq
}

// rdfCanonicalCodes gives each symbol its code: shortest length first, and in
// symbol order within a length, which is the arrangement the reader rebuilds
// from the lengths alone.
func rdfCanonicalCodes(lengths []byte) []uint32 {
	var count [rdfMaxCodeLen + 1]int
	for _, l := range lengths {
		count[l]++
	}
	var next [rdfMaxCodeLen + 1]uint32
	code := uint32(0)
	for i := 1; i <= rdfMaxCodeLen; i++ {
		next[i] = code
		// #nosec G115 -- there are at most 286 symbols of any one code length
		code = (code + uint32(count[i])) << 1
	}

	codes := make([]uint32, len(lengths))
	for i, l := range lengths {
		if l == 0 {
			continue
		}
		codes[i] = next[l]
		next[l]++
	}
	return codes
}

// rdfCodeLengths settles the Huffman code lengths for one table, none longer
// than limit. The symbols are ordered by how often they occur, most frequent
// first, and the lengths found by package-merge, which gives the best code
// obeying the limit outright rather than building a tree and flattening it.
func rdfCodeLengths(freq []int32, limit int) []byte {
	lengths := make([]byte, len(freq))

	heap := &rdfHeap{}
	for symbol, f := range freq {
		if f > 0 {
			heap.push(symbol, f)
		}
	}
	switch heap.len() {
	case 0:
		return lengths
	case 1:
		symbol, _ := heap.pop()
		lengths[symbol] = 1
		return lengths
	}

	symbols := make([]int, 0, heap.len())
	values := make([]int32, 0, heap.len())
	for heap.len() > 0 {
		symbol, value := heap.pop()
		symbols = append(symbols, symbol)
		values = append(values, value)
	}

	for i, l := range rdfPackageMerge(values, limit) {
		lengths[symbols[i]] = l
	}
	return lengths
}

// rdfPackageMerge returns a code length for each of the given frequencies, which
// must be in descending order, with none longer than limit.
//
// It is the package-merge algorithm run backwards: build, for each length, the
// cheapest set of packages that could be paid for at that depth, then walk back
// down taking the ones actually used and shortening their symbols' codes.
func rdfPackageMerge(freqs []int32, limit int) []byte {
	symbols := len(freqs)
	minimumCost, flag := rdfPackageBudgets(symbols, limit)

	codeLength := make([]byte, symbols)
	value := make([][]int32, limit)
	kind := make([][]int, limit)
	position := make([]int, limit)
	for j := range limit {
		value[j] = make([]int32, minimumCost[j])
		kind[j] = make([]int, minimumCost[j])
	}

	for i := range symbols {
		// #nosec G115 -- no code may be longer than fifteen bits
		codeLength[i] = byte(limit)
	}
	for t := range minimumCost[limit-1] {
		value[limit-1][t] = freqs[t]
		kind[limit-1][t] = t
	}

	// take marks one package at a depth as used: a joined pair takes both of the
	// packages below it in turn, and a symbol has its code shortened by one.
	var take func(j int)
	take = func(j int) {
		x := kind[j][position[j]]
		if x == symbols {
			take(j + 1)
			take(j + 1)
		} else {
			codeLength[x]--
		}
		position[j]++
	}

	if flag[limit-1] {
		codeLength[0]--
		position[limit-1]++
	}
	for j := limit - 2; j >= 0; j-- {
		rdfMergeRow(freqs, value[j+1], value[j], kind[j], position[j+1])
		position[j] = 0
		if flag[j] {
			take(j)
		}
	}
	return codeLength
}

// rdfPackageBudgets works out how many packages each depth can pay for, and
// where the count leaves one owed. The deepest holds one package per symbol,
// and each shallower depth at most half the one below plus a symbol apiece.
func rdfPackageBudgets(symbols, limit int) (minimumCost []int, flag []bool) {
	minimumCost = make([]int, limit)
	flag = make([]bool, limit)

	excess := (1 << limit) - symbols
	half := 1 << (limit - 1)

	minimumCost[limit-1] = symbols
	for j := range limit {
		if excess >= half {
			flag[j] = true
			excess -= half
		}
		excess <<= 1
		if limit-2-j >= 0 {
			minimumCost[limit-2-j] = minimumCost[limit-1-j]/2 + symbols
		}
	}

	minimumCost[0] = 0
	if flag[0] {
		minimumCost[0] = 1
	}
	for j := 1; j < limit; j++ {
		bound := 2 * minimumCost[j-1]
		if flag[j] {
			bound++
		}
		minimumCost[j] = min(minimumCost[j], bound)
	}
	return minimumCost, flag
}

// rdfMergeRow fills one depth's packages from the depth below: at each place it
// takes either the two cheapest packages below joined together, or the next
// symbol on its own, whichever weighs more — the frequencies being in
// descending order, so the dearest still unplaced comes first.
func rdfMergeRow(freqs, below, row []int32, kinds []int, start int) {
	symbols := len(freqs)
	i := 0
	next := start
	for t := range row {
		paired := next+1 < len(below)
		var weight int32
		if paired {
			weight = below[next] + below[next+1]
		}
		if i >= symbols || (paired && weight > freqs[i]) {
			row[t] = weight
			kinds[t] = symbols
			next += 2
			continue
		}
		row[t] = freqs[i]
		kinds[t] = i
		i++
	}
}

// rdfHeap orders symbols by how often they occur, the most frequent first. Where
// two occur equally often the order falls out of how the heap happens to hold
// them, which is part of what has to be reproduced for the codes to match.
type rdfHeap struct {
	values  []int32
	symbols []int
}

func (h *rdfHeap) len() int { return len(h.values) }

func (h *rdfHeap) swap(a, b int) {
	h.values[a], h.values[b] = h.values[b], h.values[a]
	h.symbols[a], h.symbols[b] = h.symbols[b], h.symbols[a]
}

func (h *rdfHeap) push(symbol int, value int32) {
	h.values = append(h.values, value)
	h.symbols = append(h.symbols, symbol)
	for current := len(h.values) - 1; current > 0; {
		parent := (current - 1) / 2
		if h.values[current] <= h.values[parent] {
			break
		}
		h.swap(current, parent)
		current = parent
	}
}

func (h *rdfHeap) pop() (symbol int, value int32) {
	value, symbol = h.values[0], h.symbols[0]
	last := len(h.values) - 1
	h.swap(0, last)
	h.values = h.values[:last]
	h.symbols = h.symbols[:last]

	for parent := 0; ; {
		child := 2*parent + 1
		if child >= len(h.values) {
			break
		}
		if child+1 < len(h.values) && h.values[child+1] > h.values[child] {
			child++
		}
		if h.values[child] <= h.values[parent] {
			break
		}
		h.swap(child, parent)
		parent = child
	}
	return symbol, value
}

// deflateDecode reads a raw DEFLATE stream, starting at the given byte.
func deflateDecode(data []byte, startIndex int) ([]byte, error) {
	if startIndex < 0 || startIndex > len(data) {
		return nil, errors.New("start index is outside the input")
	}
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	out, err := io.ReadAll(flate.NewReader(bytes.NewReader(data[startIndex:])))
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RawDeflate compresses the input into a raw DEFLATE stream.
type RawDeflate struct{}

// Meta returns the operation metadata.
func (RawDeflate) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Raw Deflate",
		Module:      "Compression",
		Description: "Compresses data using the deflate algorithm with no headers.",
		InfoURL:     "https://wikipedia.org/wiki/DEFLATE",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RawDeflate) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Compression type", Type: core.ArgOption, Value: deflateCompressionTypes},
	}
}

// Run compresses the input.
func (RawDeflate) Run(in *core.Dish, args []any) (*core.Dish, error) {
	compressionType, _ := args[0].(string)
	out, err := deflateEncode(in.Bytes(), compressionType)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(RawDeflate{}) }

// RawInflate reads a raw DEFLATE stream back into the bytes it was made from.
type RawInflate struct{}

// Meta returns the operation metadata.
func (RawInflate) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Raw Inflate",
		Module:      "Compression",
		Description: "Decompresses data which has been compressed using the deflate algorithm with no headers.",
		InfoURL:     "https://wikipedia.org/wiki/DEFLATE",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (RawInflate) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Start index", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		{Name: "Initial output buffer size", Type: core.ArgNumber, Integer: true, Value: float64(0)},
		{Name: "Buffer expansion type", Type: core.ArgOption, Value: deflateBufferTypes},
		{Name: "Resize buffer after decompression", Flag: "resize-buffer", Type: core.ArgBoolean, Value: false},
		{Name: "Verify result", Type: core.ArgBoolean, Value: false},
	}
}

// Run decompresses the input. Only the start index has any bearing on the
// result: the other three size and grow the reader's working buffer, which does
// not change what it decodes.
func (RawInflate) Run(in *core.Dish, args []any) (*core.Dish, error) {
	startIndex, _ := args[0].(float64)
	out, err := deflateDecode(in.Bytes(), int(startIndex))
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func init() { core.Register(RawInflate{}) }
