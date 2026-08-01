// Package deflateraw compresses with DEFLATE the way the browser does.
//
// Go's compress/flate and the pako library CyberChef runs in the browser both
// emit valid DEFLATE, but they choose different matches, so the bytes differ.
// Anything comparing output against CyberChef's needs pako's choices, which is
// what [Deflate] reproduces.
package deflateraw

import "hash/adler32"

// A DEFLATE encoder whose output matches zlib's byte for byte, ported from
// pako (../CyberChef/node_modules/pako/lib/zlib/{deflate,trees}.js), which is a
// faithful transcription of zlib's deflate.c and trees.c. Byte-exactness is not
// incidental: CyberChef's Generate QR Code embeds the compressed bitmap in a
// PNG, and its fixture asserts the whole file.
//
// Only the configuration that operation uses is implemented: the highest
// compression level, which selects the lazy-matching strategy, wrapped in a
// zlib header and trailer.

// The shape of the compressed format.
const (
	dfLengthCodes    = 29                             // length codes, not counting the end marker
	dfLiterals       = 256                            // literal bytes
	dfLCodes         = dfLiterals + 1 + dfLengthCodes // literals, the end marker and the lengths
	dfDCodes         = 30                             // distance codes
	dfBLCodes        = 19                             // codes carrying the code lengths themselves
	dfHeapSize       = 2*dfLCodes + 1
	dfMaxBits        = 15 // no code may be longer
	dfMaxBLBits      = 7  // nor may a code-length code
	dfMinMatch       = 3
	dfMaxMatch       = 258
	dfMinLookahead   = dfMaxMatch + dfMinMatch + 1
	dfEndBlock       = 256
	dfRep3To6        = 16 // repeat the previous length 3 to 6 times
	dfRepZero3To10   = 17
	dfRepZero11To138 = 18
	dfBitBufSize     = 16
)

// The three block encodings, in the order the two-bit header numbers them.
const (
	dfStoredBlock  = 0
	dfStaticTrees  = 1
	dfDynamicTrees = 2
)

// The window and hash geometry, which zlib derives from the window and memory
// level arguments. These are the values for a 32K window at the default memory
// level, which is what every caller here uses.
const (
	dfWindowBits = 15
	dfWindowSize = 1 << dfWindowBits
	dfWindowMask = dfWindowSize - 1
	dfHashBits   = 15
	dfHashSize   = 1 << dfHashBits
	dfHashMask   = dfHashSize - 1
	dfHashShift  = (dfHashBits + dfMinMatch - 1) / dfMinMatch
	dfLitBufSize = 1 << 14
)

// The strategy of the highest compression level: hold a match back while a
// longer one is sought, walking a long hash chain to find it.
const (
	dfGoodLength = 32
	dfMaxLazy    = 258
	dfNiceLength = 258
	dfMaxChain   = 4096
)

// The extra bits each length, distance and code-length code carries.
var (
	dfExtraLengthBits = [dfLengthCodes]int{
		0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2,
		3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0,
	}
	dfExtraDistBits = [dfDCodes]int{
		0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6,
		7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13,
	}
	dfExtraCodeLengthBits = [dfBLCodes]int{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 7,
	}
	// The code lengths are sent in decreasing order of probability, so the
	// unused ones fall off the end and need not be sent at all.
	dfCodeLengthOrder = [dfBLCodes]int{
		16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
	}
)

// dfStaticTree describes one of the trees a block may use, and the fixed tree
// it is measured against.
type dfStaticTree struct {
	tree      []int // the fixed codes, or nil where there are none
	extraBits []int
	extraBase int
	elems     int
	maxLength int
}

// The tables every stream shares, built once from the format's definitions.
var (
	dfStaticLTree = make([]int, (dfLCodes+2)*2)
	dfStaticDTree = make([]int, dfDCodes*2)
	dfDistCode    = make([]int, 512)
	dfLengthCode  = make([]int, dfMaxMatch-dfMinMatch+1)
	dfBaseLength  = make([]int, dfLengthCodes)
	dfBaseDist    = make([]int, dfDCodes)

	dfLiteralDesc  *dfStaticTree
	dfDistanceDesc *dfStaticTree
	dfLengthsDesc  *dfStaticTree
)

func init() {
	dfBuildStaticTables()
}

// dfBuildStaticTables fills the shared tables: the code each length and
// distance maps to, and the fixed trees a block may be sent under.
func dfBuildStaticTables() {
	length := 0
	for code := range dfLengthCodes - 1 {
		dfBaseLength[code] = length
		for range 1 << dfExtraLengthBits[code] {
			dfLengthCode[length] = code
			length++
		}
	}
	// A match of the greatest length has two encodings; the shorter one wins.
	dfLengthCode[length-1] = dfLengthCodes - 1

	dist := 0
	code := 0
	for ; code < 16; code++ {
		dfBaseDist[code] = dist
		for range 1 << dfExtraDistBits[code] {
			dfDistCode[dist] = code
			dist++
		}
	}
	dist >>= 7 // beyond this point the distances are counted in units of 128
	for ; code < dfDCodes; code++ {
		dfBaseDist[code] = dist << 7
		for range 1 << (dfExtraDistBits[code] - 7) {
			dfDistCode[256+dist] = code
			dist++
		}
	}

	var counts [dfMaxBits + 1]int
	n := 0
	for ; n <= 143; n++ {
		dfStaticLTree[n*2+1] = 8
		counts[8]++
	}
	for ; n <= 255; n++ {
		dfStaticLTree[n*2+1] = 9
		counts[9]++
	}
	for ; n <= 279; n++ {
		dfStaticLTree[n*2+1] = 7
		counts[7]++
	}
	// The last two codes cannot occur, but the tree is only canonical with
	// them in it.
	for ; n <= 287; n++ {
		dfStaticLTree[n*2+1] = 8
		counts[8]++
	}
	dfGenerateCodes(dfStaticLTree, dfLCodes+1, counts[:])

	for n := range dfDCodes {
		dfStaticDTree[n*2+1] = 5
		dfStaticDTree[n*2] = dfReverseBits(n, 5)
	}

	dfLiteralDesc = &dfStaticTree{dfStaticLTree, dfExtraLengthBits[:], dfLiterals + 1, dfLCodes, dfMaxBits}
	dfDistanceDesc = &dfStaticTree{dfStaticDTree, dfExtraDistBits[:], 0, dfDCodes, dfMaxBits}
	dfLengthsDesc = &dfStaticTree{nil, dfExtraCodeLengthBits[:], 0, dfBLCodes, dfMaxBLBits}
}

// dfReverseBits mirrors the low len bits of a code, which is the order the
// format sends them in.
func dfReverseBits(code, length int) int {
	res := 0
	for ; length > 0; length-- {
		res |= code & 1
		code >>= 1
		res <<= 1
	}
	return res >> 1
}

// dfTreeDesc pairs a tree being built with the fixed tree it is measured
// against.
type dfTreeDesc struct {
	tree    []int
	maxCode int
	static  *dfStaticTree
}

// dfState is one compression in progress.
type dfState struct {
	input   []byte
	nextIn  int
	availIn int
	out     []byte

	window []byte
	head   []int
	prev   []int

	insertHash  int
	strStart    int
	blockStart  int
	lookahead   int
	matchStart  int
	matchLength int
	prevLength  int
	prevMatch   int
	matchAvail  bool

	pending    []byte
	pendingLen int
	distBuf    int
	litBuf     int
	lastLit    int
	matches    int

	dynLTree []int
	dynDTree []int
	blTree   []int
	lDesc    dfTreeDesc
	dDesc    dfTreeDesc
	blDesc   dfTreeDesc

	blCount [dfMaxBits + 1]int
	heap    [dfHeapSize]int
	heapLen int
	heapMax int
	depth   [dfHeapSize]int

	optLen    int
	staticLen int

	bitBuf   int
	bitValid int
}

// Deflate compresses the data exactly as zlib does at its highest level,
// wrapped in the zlib header and adler checksum.
func Deflate(data []byte) []byte {
	s := &dfState{
		input:    data,
		availIn:  len(data),
		window:   make([]byte, 2*dfWindowSize),
		head:     make([]int, dfHashSize),
		prev:     make([]int, dfWindowSize),
		pending:  make([]byte, dfLitBufSize*4),
		distBuf:  dfLitBufSize,
		litBuf:   3 * dfLitBufSize,
		dynLTree: make([]int, dfHeapSize*2),
		dynDTree: make([]int, (2*dfDCodes+1)*2),
		blTree:   make([]int, (2*dfBLCodes+1)*2),
	}
	s.lDesc = dfTreeDesc{s.dynLTree, 0, dfLiteralDesc}
	s.dDesc = dfTreeDesc{s.dynDTree, 0, dfDistanceDesc}
	s.blDesc = dfTreeDesc{s.blTree, 0, dfLengthsDesc}
	s.matchLength = dfMinMatch - 1
	s.prevLength = dfMinMatch - 1
	s.initBlock()

	// The zlib header: the method and window size, then the level, chosen so
	// the pair is a multiple of thirty-one.
	header := (8 + (dfWindowBits-8)<<4) << 8
	header |= 3 << 6 // the highest compression level
	header += 31 - header%31
	// #nosec G115 -- the header is two bytes by construction
	s.putByte(byte(header >> 8))
	s.putByte(byte(header)) // #nosec G115 -- as above
	s.flushPending()

	s.deflateSlow()
	s.flushPending()

	sum := adler32.Checksum(data)
	// #nosec G115 -- the checksum is written one byte at a time by design
	s.putByte(byte(sum >> 24))
	s.putByte(byte(sum >> 16)) // #nosec G115 -- as above
	s.putByte(byte(sum >> 8))  // #nosec G115 -- as above
	s.putByte(byte(sum))       // #nosec G115 -- as above
	s.flushPending()

	return s.out
}

func (s *dfState) putByte(b byte) {
	s.pending[s.pendingLen] = b
	s.pendingLen++
}

func (s *dfState) putShort(w int) {
	// #nosec G115 -- a short is written as its two bytes by design
	s.putByte(byte(w))
	s.putByte(byte(w >> 8)) // #nosec G115 -- as above
}

// flushPending moves the finished bytes to the output. The output here is never
// short, so this always empties the buffer.
func (s *dfState) flushPending() {
	s.out = append(s.out, s.pending[:s.pendingLen]...)
	s.pendingLen = 0
}

// sendBits appends a value of the given width, least significant bit first.
func (s *dfState) sendBits(value, length int) {
	if s.bitValid > dfBitBufSize-length {
		s.bitBuf |= (value << s.bitValid) & 0xFFFF
		s.putShort(s.bitBuf)
		s.bitBuf = value >> (dfBitBufSize - s.bitValid)
		s.bitValid += length - dfBitBufSize
		return
	}
	s.bitBuf |= (value << s.bitValid) & 0xFFFF
	s.bitValid += length
}

func (s *dfState) sendCode(c int, tree []int) {
	s.sendBits(tree[c*2], tree[c*2+1])
}

// windup pads the bit buffer out to a byte boundary.
func (s *dfState) windup() {
	switch {
	case s.bitValid > 8:
		s.putShort(s.bitBuf)
	case s.bitValid > 0:
		s.putByte(byte(s.bitBuf)) // #nosec G115 -- fewer than eight bits remain
	}
	s.bitBuf = 0
	s.bitValid = 0
}

// initBlock clears the frequencies for a new block.
func (s *dfState) initBlock() {
	for n := range dfLCodes {
		s.dynLTree[n*2] = 0
	}
	for n := range dfDCodes {
		s.dynDTree[n*2] = 0
	}
	for n := range dfBLCodes {
		s.blTree[n*2] = 0
	}
	s.dynLTree[dfEndBlock*2] = 1
	s.optLen, s.staticLen = 0, 0
	s.lastLit, s.matches = 0, 0
}

// dfDistanceCode maps a distance to the code that carries it, the larger half
// of the range sharing codes between every 128 distances.
func dfDistanceCode(dist int) int {
	if dist < 256 {
		return dfDistCode[dist]
	}
	return dfDistCode[256+(dist>>7)]
}

// smaller orders two subtrees, breaking a tie on frequency by depth so the
// longest code comes out as short as it can.
func (s *dfState) smaller(tree []int, n, m int) bool {
	return tree[n*2] < tree[m*2] || (tree[n*2] == tree[m*2] && s.depth[n] <= s.depth[m])
}

// downHeap restores the heap below k.
func (s *dfState) downHeap(tree []int, k int) {
	v := s.heap[k]
	for j := k << 1; j <= s.heapLen; j <<= 1 {
		if j < s.heapLen && s.smaller(tree, s.heap[j+1], s.heap[j]) {
			j++
		}
		if s.smaller(tree, v, s.heap[j]) {
			break
		}
		s.heap[k] = s.heap[j]
		k = j
	}
	s.heap[k] = v
}

// generateBitLengths walks the finished tree and records how long each code is,
// shortening the longest ones where the format's limit is exceeded.
func (s *dfState) generateBitLengths(desc *dfTreeDesc) {
	tree := desc.tree
	maxCode := desc.maxCode
	static := desc.static
	overflow := 0

	for bits := range dfMaxBits + 1 {
		s.blCount[bits] = 0
	}
	tree[s.heap[s.heapMax]*2+1] = 0

	for h := s.heapMax + 1; h < dfHeapSize; h++ {
		n := s.heap[h]
		bits := tree[tree[n*2+1]*2+1] + 1
		if bits > static.maxLength {
			bits = static.maxLength
			overflow++
		}
		tree[n*2+1] = bits
		if n > maxCode {
			continue
		}

		s.blCount[bits]++
		extra := 0
		if n >= static.extraBase {
			extra = static.extraBits[n-static.extraBase]
		}
		freq := tree[n*2]
		s.optLen += freq * (bits + extra)
		if static.tree != nil {
			s.staticLen += freq * (static.tree[n*2+1] + extra)
		}
	}
	if overflow == 0 {
		return
	}

	// Move leaves up from the deepest level until nothing overflows.
	for {
		bits := static.maxLength - 1
		for s.blCount[bits] == 0 {
			bits--
		}
		s.blCount[bits]--
		s.blCount[bits+1] += 2
		s.blCount[static.maxLength]--
		overflow -= 2
		if overflow <= 0 {
			break
		}
	}

	h := dfHeapSize
	for bits := static.maxLength; bits != 0; bits-- {
		n := s.blCount[bits]
		for n != 0 {
			h--
			m := s.heap[h]
			if m > maxCode {
				continue
			}
			if tree[m*2+1] != bits {
				s.optLen += (bits - tree[m*2+1]) * tree[m*2]
				tree[m*2+1] = bits
			}
			n--
		}
	}
}

// dfGenerateCodes assigns the canonical code of each length, mirrored into the
// order the format sends.
func dfGenerateCodes(tree []int, maxCode int, blCount []int) {
	var nextCode [dfMaxBits + 1]int
	code := 0
	for bits := 1; bits <= dfMaxBits; bits++ {
		code = (code + blCount[bits-1]) << 1
		nextCode[bits] = code
	}
	for n := 0; n <= maxCode; n++ {
		length := tree[n*2+1]
		if length == 0 {
			continue
		}
		tree[n*2] = dfReverseBits(nextCode[length], length)
		nextCode[length]++
	}
}

// buildTree constructs one Huffman tree from the frequencies recorded for it.
func (s *dfState) buildTree(desc *dfTreeDesc) {
	tree := desc.tree
	static := desc.static
	maxCode := -1

	s.heapLen = 0
	s.heapMax = dfHeapSize
	for n := range static.elems {
		if tree[n*2] != 0 {
			s.heapLen++
			s.heap[s.heapLen] = n
			maxCode = n
			s.depth[n] = 0
		} else {
			tree[n*2+1] = 0
		}
	}

	// The format needs at least two codes, even where the data uses fewer.
	for s.heapLen < 2 {
		s.heapLen++
		node := 0
		if maxCode < 2 {
			maxCode++
			node = maxCode
		}
		s.heap[s.heapLen] = node
		tree[node*2] = 1
		s.depth[node] = 0
		s.optLen--
		if static.tree != nil {
			s.staticLen -= static.tree[node*2+1]
		}
	}
	desc.maxCode = maxCode

	for n := s.heapLen >> 1; n >= 1; n-- {
		s.downHeap(tree, n)
	}

	// Repeatedly join the two least frequent nodes.
	node := static.elems
	for {
		n := s.heap[1]
		s.heap[1] = s.heap[s.heapLen]
		s.heapLen--
		s.downHeap(tree, 1)
		m := s.heap[1]

		s.heapMax--
		s.heap[s.heapMax] = n
		s.heapMax--
		s.heap[s.heapMax] = m

		tree[node*2] = tree[n*2] + tree[m*2]
		s.depth[node] = max(s.depth[n], s.depth[m]) + 1
		tree[n*2+1] = node
		tree[m*2+1] = node

		s.heap[1] = node
		node++
		s.downHeap(tree, 1)

		if s.heapLen < 2 {
			break
		}
	}

	s.heapMax--
	s.heap[s.heapMax] = s.heap[1]

	s.generateBitLengths(desc)
	dfGenerateCodes(tree, maxCode, s.blCount[:])
}

// scanTree counts how often each code length occurs, so the lengths themselves
// can be sent compressed.
func (s *dfState) scanTree(tree []int, maxCode int) {
	prevLen := -1
	nextLen := tree[1]
	count := 0
	maxCount, minCount := 7, 4
	if nextLen == 0 {
		maxCount, minCount = 138, 3
	}
	tree[(maxCode+1)*2+1] = 0xFFFF // a guard the walk stops against

	for n := 0; n <= maxCode; n++ {
		curLen := nextLen
		nextLen = tree[(n+1)*2+1]
		count++
		switch {
		case count < maxCount && curLen == nextLen:
			continue
		case count < minCount:
			s.blTree[curLen*2] += count
		case curLen != 0:
			if curLen != prevLen {
				s.blTree[curLen*2]++
			}
			s.blTree[dfRep3To6*2]++
		case count <= 10:
			s.blTree[dfRepZero3To10*2]++
		default:
			s.blTree[dfRepZero11To138*2]++
		}

		count = 0
		prevLen = curLen
		switch {
		case nextLen == 0:
			maxCount, minCount = 138, 3
		case curLen == nextLen:
			maxCount, minCount = 6, 3
		default:
			maxCount, minCount = 7, 4
		}
	}
}

// sendTree writes those lengths out under the tree scanTree measured.
func (s *dfState) sendTree(tree []int, maxCode int) {
	prevLen := -1
	nextLen := tree[1]
	count := 0
	maxCount, minCount := 7, 4
	if nextLen == 0 {
		maxCount, minCount = 138, 3
	}

	for n := 0; n <= maxCode; n++ {
		curLen := nextLen
		nextLen = tree[(n+1)*2+1]
		count++
		switch {
		case count < maxCount && curLen == nextLen:
			continue
		case count < minCount:
			for ; count != 0; count-- {
				s.sendCode(curLen, s.blTree)
			}
		case curLen != 0:
			if curLen != prevLen {
				s.sendCode(curLen, s.blTree)
				count--
			}
			s.sendCode(dfRep3To6, s.blTree)
			s.sendBits(count-3, 2)
		case count <= 10:
			s.sendCode(dfRepZero3To10, s.blTree)
			s.sendBits(count-3, 3)
		default:
			s.sendCode(dfRepZero11To138, s.blTree)
			s.sendBits(count-11, 7)
		}

		count = 0
		prevLen = curLen
		switch {
		case nextLen == 0:
			maxCount, minCount = 138, 3
		case curLen == nextLen:
			maxCount, minCount = 6, 3
		default:
			maxCount, minCount = 7, 4
		}
	}
}

// buildCodeLengthTree builds the tree carrying the other two trees' lengths,
// and reports how many of its codes need sending.
func (s *dfState) buildCodeLengthTree() int {
	s.scanTree(s.dynLTree, s.lDesc.maxCode)
	s.scanTree(s.dynDTree, s.dDesc.maxCode)
	s.buildTree(&s.blDesc)

	maxIndex := dfBLCodes - 1
	for ; maxIndex >= 3; maxIndex-- {
		if s.blTree[dfCodeLengthOrder[maxIndex]*2+1] != 0 {
			break
		}
	}
	s.optLen += 3*(maxIndex+1) + 5 + 5 + 4
	return maxIndex
}

// sendAllTrees writes the header of a block using its own trees.
func (s *dfState) sendAllTrees(lCodes, dCodes, blCodes int) {
	s.sendBits(lCodes-257, 5)
	s.sendBits(dCodes-1, 5)
	s.sendBits(blCodes-4, 4)
	for rank := range blCodes {
		s.sendBits(s.blTree[dfCodeLengthOrder[rank]*2+1], 3)
	}
	s.sendTree(s.dynLTree, lCodes-1)
	s.sendTree(s.dynDTree, dCodes-1)
}

// compressBlock writes the block's symbols under the given trees.
func (s *dfState) compressBlock(lTree, dTree []int) {
	for lx := 0; lx < s.lastLit; lx++ {
		dist := int(s.pending[s.distBuf+lx*2])<<8 | int(s.pending[s.distBuf+lx*2+1])
		lc := int(s.pending[s.litBuf+lx])

		if dist == 0 {
			s.sendCode(lc, lTree)
			continue
		}
		code := dfLengthCode[lc]
		s.sendCode(code+dfLiterals+1, lTree)
		if extra := dfExtraLengthBits[code]; extra != 0 {
			s.sendBits(lc-dfBaseLength[code], extra)
		}
		dist--
		code = dfDistanceCode(dist)
		s.sendCode(code, dTree)
		if extra := dfExtraDistBits[code]; extra != 0 {
			s.sendBits(dist-dfBaseDist[code], extra)
		}
	}
	s.sendCode(dfEndBlock, lTree)
}

// tally records one symbol, reporting whether the block is now full.
func (s *dfState) tally(dist, lc int) bool {
	// #nosec G115 -- a distance is recorded as its two bytes, a literal as one
	s.pending[s.distBuf+s.lastLit*2] = byte(dist >> 8)
	s.pending[s.distBuf+s.lastLit*2+1] = byte(dist) // #nosec G115 -- as above
	s.pending[s.litBuf+s.lastLit] = byte(lc)        // #nosec G115 -- as above
	s.lastLit++

	if dist == 0 {
		s.dynLTree[lc*2]++
	} else {
		s.matches++
		dist--
		s.dynLTree[(dfLengthCode[lc]+dfLiterals+1)*2]++
		s.dynDTree[dfDistanceCode(dist)*2]++
	}
	return s.lastLit == dfLitBufSize-1
}

// flushBlock chooses the cheapest of the three encodings and writes the block.
func (s *dfState) flushBlock(buf, storedLen int, last bool) {
	s.buildTree(&s.lDesc)
	s.buildTree(&s.dDesc)
	maxIndex := s.buildCodeLengthTree()

	optBytes := (s.optLen + 3 + 7) >> 3
	staticBytes := (s.staticLen + 3 + 7) >> 3
	if staticBytes <= optBytes {
		optBytes = staticBytes
	}

	lastBit := 0
	if last {
		lastBit = 1
	}
	switch {
	case storedLen+4 <= optBytes && buf >= 0:
		// Storing the bytes plainly costs less than coding them.
		s.sendBits(dfStoredBlock<<1+lastBit, 3)
		s.windup()
		s.putShort(storedLen)
		s.putShort(^storedLen & 0xFFFF)
		copy(s.pending[s.pendingLen:], s.window[buf:buf+storedLen])
		s.pendingLen += storedLen
	case staticBytes == optBytes:
		s.sendBits(dfStaticTrees<<1+lastBit, 3)
		s.compressBlock(dfStaticLTree, dfStaticDTree)
	default:
		s.sendBits(dfDynamicTrees<<1+lastBit, 3)
		s.sendAllTrees(s.lDesc.maxCode+1, s.dDesc.maxCode+1, maxIndex+1)
		s.compressBlock(s.dynLTree, s.dynDTree)
	}

	s.initBlock()
	if last {
		s.windup()
	}
}

// flushBlockOnly closes the block covering everything read since the last one.
func (s *dfState) flushBlockOnly(last bool) {
	start := s.blockStart
	if start < 0 {
		start = -1
	}
	s.flushBlock(start, s.strStart-s.blockStart, last)
	s.blockStart = s.strStart
	s.flushPending()
}

// readBuf copies the next stretch of input into the window.
func (s *dfState) readBuf(start, size int) int {
	length := min(s.availIn, size)
	copy(s.window[start:start+length], s.input[s.nextIn:s.nextIn+length])
	s.availIn -= length
	s.nextIn += length
	return length
}

// fillWindow tops the window up, sliding it down once it fills.
func (s *dfState) fillWindow() {
	for {
		more := 2*dfWindowSize - s.lookahead - s.strStart

		if s.strStart >= dfWindowSize+(dfWindowSize-dfMinLookahead) {
			copy(s.window[:dfWindowSize], s.window[dfWindowSize:2*dfWindowSize])
			s.matchStart -= dfWindowSize
			s.strStart -= dfWindowSize
			s.blockStart -= dfWindowSize

			// The hash chains hold window positions, so they slide too.
			for p := range dfHashSize {
				if m := s.head[p]; m >= dfWindowSize {
					s.head[p] = m - dfWindowSize
				} else {
					s.head[p] = 0
				}
			}
			for p := range dfWindowSize {
				if m := s.prev[p]; m >= dfWindowSize {
					s.prev[p] = m - dfWindowSize
				} else {
					s.prev[p] = 0
				}
			}
			more += dfWindowSize
		}
		if s.availIn == 0 {
			return
		}

		s.lookahead += s.readBuf(s.strStart+s.lookahead, more)

		// Prime the rolling hash over the two bytes before the next match
		// position; entering the string supplies the third.
		if s.lookahead >= dfMinMatch {
			s.insertHash = int(s.window[s.strStart])
			s.insertHash = ((s.insertHash << dfHashShift) ^ int(s.window[s.strStart+1])) & dfHashMask
		}

		if s.lookahead >= dfMinLookahead || s.availIn == 0 {
			return
		}
	}
}

// longestMatch walks the hash chain from the given position and returns the
// longest match it finds, recording where it starts.
func (s *dfState) longestMatch(curMatch int) int {
	chainLength := dfMaxChain
	bestLen := s.prevLength
	niceMatch := dfNiceLength

	limit := 0
	if s.strStart > dfWindowSize-dfMinLookahead {
		limit = s.strStart - (dfWindowSize - dfMinLookahead)
	}
	if s.prevLength >= dfGoodLength {
		chainLength >>= 2
	}
	// Never look past the input, so the result cannot depend on stale bytes.
	if niceMatch > s.lookahead {
		niceMatch = s.lookahead
	}

	scanEnd1 := s.window[s.strStart+bestLen-1]
	scanEnd := s.window[s.strStart+bestLen]

	for {
		// The last two bytes of the best match so far are compared first, since
		// a match no longer than it is of no interest.
		promising := s.window[curMatch+bestLen] == scanEnd &&
			s.window[curMatch+bestLen-1] == scanEnd1 &&
			s.window[curMatch] == s.window[s.strStart] &&
			s.window[curMatch+1] == s.window[s.strStart+1]

		if promising {
			if length := s.agreementAt(curMatch); length > bestLen {
				s.matchStart = curMatch
				bestLen = length
				if length >= niceMatch {
					break
				}
				scanEnd1 = s.window[s.strStart+bestLen-1]
				scanEnd = s.window[s.strStart+bestLen]
			}
		}

		curMatch = s.prev[curMatch&dfWindowMask]
		if curMatch <= limit {
			break
		}
		chainLength--
		if chainLength == 0 {
			break
		}
	}

	return min(bestLen, s.lookahead)
}

// agreementAt measures how far the window agrees between the current position
// and an earlier one. The first two bytes are known to match, and the third
// follows from the hash, so the comparison starts past them.
func (s *dfState) agreementAt(match int) int {
	strEnd := s.strStart + dfMaxMatch
	scan := s.strStart + 2
	match += 2
	for scan < strEnd && s.window[scan] == s.window[match] {
		scan++
		match++
	}
	return dfMaxMatch - (strEnd - scan)
}

// insertString enters the string at the current position into the hash chains
// and returns the position that previously headed its chain.
func (s *dfState) insertString(at int) int {
	s.insertHash = ((s.insertHash << dfHashShift) ^ int(s.window[at+dfMinMatch-1])) & dfHashMask
	head := s.head[s.insertHash]
	s.prev[at&dfWindowMask] = head
	s.head[s.insertHash] = at
	return head
}

// The distance beyond which a shortest match costs more than it saves.
const dfTooFar = 4096

// findMatch looks for a match at the current position, holding the previous
// one back to compare against. A shortest match far enough away costs more to
// record than the literals it would replace.
func (s *dfState) findMatch(hashHead int) {
	s.prevLength = s.matchLength
	s.prevMatch = s.matchStart
	s.matchLength = dfMinMatch - 1

	if hashHead != 0 && s.prevLength < dfMaxLazy &&
		s.strStart-hashHead <= dfWindowSize-dfMinLookahead {
		s.matchLength = s.longestMatch(hashHead)
		if s.matchLength == dfMinMatch && s.strStart-s.matchStart > dfTooFar {
			s.matchLength = dfMinMatch - 1
		}
	}
}

// emitHeldMatch records the match held from the previous position and enters
// every string it covers, reporting whether the block is now full.
func (s *dfState) emitHeldMatch() bool {
	// Do not enter strings past the end of the match.
	maxInsert := s.strStart + s.lookahead - dfMinMatch
	full := s.tally(s.strStart-1-s.prevMatch, s.prevLength-dfMinMatch)

	s.lookahead -= s.prevLength - 1
	s.prevLength -= 2
	for {
		s.strStart++
		if s.strStart <= maxInsert {
			s.insertString(s.strStart)
		}
		s.prevLength--
		if s.prevLength == 0 {
			break
		}
	}
	s.matchAvail = false
	s.matchLength = dfMinMatch - 1
	s.strStart++
	return full
}

// deflateSlow compresses the whole input, holding each match back to see
// whether the next position starts a longer one.
func (s *dfState) deflateSlow() {
	for {
		if s.lookahead < dfMinLookahead {
			s.fillWindow()
			if s.lookahead == 0 {
				break
			}
		}

		hashHead := 0
		if s.lookahead >= dfMinMatch {
			hashHead = s.insertString(s.strStart)
		}

		s.findMatch(hashHead)

		switch {
		case s.prevLength >= dfMinMatch && s.matchLength <= s.prevLength:
			// The held match is the better one, so emit it.
			if s.emitHeldMatch() {
				s.flushBlockOnly(false)
			}

		case s.matchAvail:
			// The held match was no better than a literal.
			if s.tally(0, int(s.window[s.strStart-1])) {
				s.flushBlockOnly(false)
			}
			s.strStart++
			s.lookahead--

		default:
			s.matchAvail = true
			s.strStart++
			s.lookahead--
		}
	}

	if s.matchAvail {
		s.tally(0, int(s.window[s.strStart-1]))
		s.matchAvail = false
	}
	s.flushBlockOnly(true)
}
