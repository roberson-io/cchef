package ops

import (
	"errors"
	"math/bits"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The data half of the QR reader, ported from jsQR: read the version and format
// from the sampled matrix, unmask and de-interleave the codewords, correct them
// with Reed-Solomon, then read the segments out of the bit stream.

// qrField is the Galois field the error correction is computed over.
type qrField struct {
	exp, log      []int
	size          int
	generatorBase int
	zero, one     *qrPoly
}

// newQRField builds the field's tables from its primitive polynomial.
func newQRField(primitive, size, generatorBase int) *qrField {
	f := &qrField{exp: make([]int, size), log: make([]int, size), size: size, generatorBase: generatorBase}
	x := 1
	for i := range size {
		f.exp[i] = x
		x *= 2
		if x >= size {
			x = (x ^ primitive) & (size - 1)
		}
	}
	for i := range size - 1 {
		f.log[f.exp[i]] = i
	}
	f.zero = newQRPoly(f, []int{0})
	f.one = newQRPoly(f, []int{1})
	return f
}

func (f *qrField) multiply(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return f.exp[(f.log[a]+f.log[b])%(f.size-1)]
}

func (f *qrField) inverse(a int) int {
	return f.exp[f.size-f.log[a]-1]
}

func (f *qrField) monomial(degree, coefficient int) *qrPoly {
	if coefficient == 0 {
		return f.zero
	}
	coefficients := make([]int, degree+1)
	coefficients[0] = coefficient
	return newQRPoly(f, coefficients)
}

// qrPoly is a polynomial over the field, most significant coefficient first.
type qrPoly struct {
	field        *qrField
	coefficients []int
}

// newQRPoly trims the leading zeros, which the degree is taken from.
func newQRPoly(field *qrField, coefficients []int) *qrPoly {
	if len(coefficients) > 1 && coefficients[0] == 0 {
		first := 1
		for first < len(coefficients) && coefficients[first] == 0 {
			first++
		}
		if first == len(coefficients) {
			coefficients = []int{0}
		} else {
			coefficients = append([]int{}, coefficients[first:]...)
		}
	}
	return &qrPoly{field: field, coefficients: coefficients}
}

func (p *qrPoly) degree() int { return len(p.coefficients) - 1 }
func (p *qrPoly) isZero() bool {
	return p.coefficients[0] == 0
}

func (p *qrPoly) coefficient(degree int) int {
	return p.coefficients[len(p.coefficients)-1-degree]
}

// add is also subtraction, the two being the same in this field.
func (p *qrPoly) add(other *qrPoly) *qrPoly {
	if p.isZero() {
		return other
	}
	if other.isZero() {
		return p
	}
	smaller, larger := p.coefficients, other.coefficients
	if len(smaller) > len(larger) {
		smaller, larger = larger, smaller
	}
	sum := make([]int, len(larger))
	offset := len(larger) - len(smaller)
	copy(sum, larger[:offset])
	for i := offset; i < len(larger); i++ {
		sum[i] = smaller[i-offset] ^ larger[i]
	}
	return newQRPoly(p.field, sum)
}

func (p *qrPoly) scale(scalar int) *qrPoly {
	if scalar == 0 {
		return p.field.zero
	}
	if scalar == 1 {
		return p
	}
	product := make([]int, len(p.coefficients))
	for i, c := range p.coefficients {
		product[i] = p.field.multiply(c, scalar)
	}
	return newQRPoly(p.field, product)
}

func (p *qrPoly) multiply(other *qrPoly) *qrPoly {
	if p.isZero() || other.isZero() {
		return p.field.zero
	}
	product := make([]int, len(p.coefficients)+len(other.coefficients)-1)
	for i, a := range p.coefficients {
		for j, b := range other.coefficients {
			product[i+j] ^= p.field.multiply(a, b)
		}
	}
	return newQRPoly(p.field, product)
}

func (p *qrPoly) multiplyByMonomial(degree, coefficient int) *qrPoly {
	if coefficient == 0 {
		return p.field.zero
	}
	product := make([]int, len(p.coefficients)+degree)
	for i, c := range p.coefficients {
		product[i] = p.field.multiply(c, coefficient)
	}
	return newQRPoly(p.field, product)
}

func (p *qrPoly) evaluateAt(a int) int {
	if a == 0 {
		return p.coefficient(0)
	}
	if a == 1 {
		result := 0
		for _, c := range p.coefficients {
			result ^= c
		}
		return result
	}
	result := p.coefficients[0]
	for _, c := range p.coefficients[1:] {
		result = p.field.multiply(a, result) ^ c
	}
	return result
}

// The field the standard computes its error correction over.
const (
	qrReedPrimitive = 0x011D
	qrReedSize      = 256
)

// qrReedDecode corrects a block, returning whether it could be.
func qrReedDecode(codewords []int, ecCount int) ([]int, bool) {
	out := append([]int{}, codewords...)
	field := newQRField(qrReedPrimitive, qrReedSize, 0)
	received := newQRPoly(field, out)

	// The syndromes are the received polynomial at each root of the generator;
	// all zero means the block is already right.
	syndromes := make([]int, ecCount)
	errored := false
	for s := range ecCount {
		evaluation := received.evaluateAt(field.exp[s+field.generatorBase])
		syndromes[len(syndromes)-1-s] = evaluation
		if evaluation != 0 {
			errored = true
		}
	}
	if !errored {
		return out, true
	}

	locator, evaluator, ok := qrEuclidean(field, field.monomial(ecCount, 1), newQRPoly(field, syndromes), ecCount)
	if !ok {
		return nil, false
	}
	positions, ok := qrErrorLocations(field, locator)
	if !ok {
		return nil, false
	}
	magnitudes := qrErrorMagnitudes(field, evaluator, positions)

	for i, position := range positions {
		at := len(out) - 1 - field.log[position]
		if at < 0 {
			return nil, false
		}
		out[at] ^= magnitudes[i]
	}
	return out, true
}

// qrEuclidean runs the extended Euclidean algorithm far enough to separate the
// error locator from the error evaluator. The first polynomial passed is always
// of the higher degree, so the two never need exchanging.
func qrEuclidean(field *qrField, a, b *qrPoly, limit int) (locator, evaluator *qrPoly, ok bool) {
	rLast, r := a, b
	tLast, t := field.zero, field.one

	for r.degree()*2 >= limit {
		rLastLast, tLastLast := rLast, tLast
		rLast, tLast = r, t
		if rLast.isZero() {
			return nil, nil, false
		}

		r = rLastLast
		q := field.zero
		leading := field.inverse(rLast.coefficient(rLast.degree()))
		for r.degree() >= rLast.degree() && !r.isZero() {
			degreeDiff := r.degree() - rLast.degree()
			scale := field.multiply(r.coefficient(r.degree()), leading)
			q = q.add(field.monomial(degreeDiff, scale))
			r = r.add(rLast.multiplyByMonomial(degreeDiff, scale))
		}

		t = q.multiply(tLast).add(tLastLast)
		if r.degree() >= rLast.degree() {
			return nil, nil, false
		}
	}

	atZero := t.coefficient(0)
	if atZero == 0 {
		return nil, nil, false
	}
	inverse := field.inverse(atZero)
	return t.scale(inverse), r.scale(inverse), true
}

// qrErrorLocations finds where the errors are, by trying every position.
func qrErrorLocations(field *qrField, locator *qrPoly) ([]int, bool) {
	count := locator.degree()
	if count == 1 {
		return []int{locator.coefficient(1)}, true
	}
	found := make([]int, 0, count)
	for i := 1; i < field.size && len(found) < count; i++ {
		if locator.evaluateAt(i) == 0 {
			found = append(found, field.inverse(i))
		}
	}
	if len(found) != count {
		return nil, false
	}
	return found, true
}

// qrErrorMagnitudes finds how wrong each of those positions is.
func qrErrorMagnitudes(field *qrField, evaluator *qrPoly, positions []int) []int {
	magnitudes := make([]int, len(positions))
	for i, position := range positions {
		inverse := field.inverse(position)
		denominator := 1
		for j, other := range positions {
			if i != j {
				denominator = field.multiply(denominator, 1^field.multiply(other, inverse))
			}
		}
		magnitudes[i] = field.multiply(evaluator.evaluateAt(inverse), field.inverse(denominator))
	}
	return magnitudes
}

// qrBitStream reads a whole number of bits at a time out of the codewords.
type qrBitStream struct {
	bytes      []int
	byteOffset int
	bitOffset  int
}

var errQRStreamExhausted = errors.New("not enough bits remain")

func (s *qrBitStream) available() int {
	return 8*(len(s.bytes)-s.byteOffset) - s.bitOffset
}

// readBits takes the next n bits, most significant first.
func (s *qrBitStream) readBits(n int) (int, error) {
	if n < 1 || n > 32 || n > s.available() {
		return 0, errQRStreamExhausted
	}
	result := 0

	if s.bitOffset > 0 {
		left := 8 - s.bitOffset
		toRead := min(n, left)
		unread := left - toRead
		mask := (0xFF >> (8 - toRead)) << unread
		result = (s.bytes[s.byteOffset] & mask) >> unread
		n -= toRead
		s.bitOffset += toRead
		if s.bitOffset == 8 {
			s.bitOffset = 0
			s.byteOffset++
		}
	}

	for n >= 8 {
		result = result<<8 | s.bytes[s.byteOffset]&0xFF
		s.byteOffset++
		n -= 8
	}
	if n > 0 {
		unread := 8 - n
		mask := (0xFF >> unread) << unread
		result = result<<n | (s.bytes[s.byteOffset]&mask)>>unread
		s.bitOffset += n
	}
	return result, nil
}

// The segment modes the reader understands.
const (
	qrModeTerminator  = 0x0
	qrModeNumericRead = 0x1
	qrModeAlphaRead   = 0x2
	qrModeByteRead    = 0x4
	qrModeKanjiRead   = 0x8
	qrModeECI         = 0x7
)

// qrAlphanumericCodes is the alphabet the alphanumeric mode encodes.
const qrAlphanumericCodes = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

// qrDecodeData reads the segments out of the corrected codewords.
func qrDecodeData(codewords []int, version int) (string, bool) {
	stream := &qrBitStream{bytes: codewords}

	// The character count field widens with the version.
	size := 0
	switch {
	case version > 26:
		size = 2
	case version > 9:
		size = 1
	}

	var text strings.Builder
	for {
		// Running out of bits ends the segments, as too few to hold a mode.
		mode, err := stream.readBits(4)
		if err != nil {
			break
		}
		if mode == qrModeTerminator {
			return text.String(), true
		}

		segment, err := qrReadSegment(stream, mode, size)
		if err != nil {
			return "", false
		}
		text.WriteString(segment)
	}

	// Whatever remains must be padding rather than another segment.
	if stream.available() == 0 {
		return text.String(), true
	}
	remaining, err := stream.readBits(stream.available())
	if err != nil || remaining != 0 {
		return "", false
	}
	return text.String(), true
}

// qrReadSegment reads one segment in whichever mode its header names.
func qrReadSegment(stream *qrBitStream, mode, size int) (string, error) {
	switch mode {
	case qrModeECI:
		// A character set marker carries no text of its own.
		return "", qrSkipECI(stream)
	case qrModeNumericRead:
		return qrReadNumeric(stream, size)
	case qrModeAlphaRead:
		return qrReadAlphanumeric(stream, size)
	case qrModeByteRead:
		return qrReadBytes(stream, size)
	case qrModeKanjiRead:
		return qrReadKanji(stream, size)
	}
	return "", errors.New("a segment in a mode the reader does not know")
}

// qrSkipECI steps over a character set marker, whose width is given by how many
// of its leading bits are set.
func qrSkipECI(stream *qrBitStream) error {
	for _, width := range []int{7, 14, 21} {
		flag, err := stream.readBits(1)
		if err != nil {
			return err
		}
		if flag == 0 {
			_, err = stream.readBits(width)
			return err
		}
	}
	return nil
}

func qrReadNumeric(stream *qrBitStream, size int) (string, error) {
	length, err := stream.readBits([3]int{10, 12, 14}[size])
	if err != nil {
		return "", err
	}

	var out strings.Builder
	for ; length >= 3; length -= 3 {
		group, err := stream.readBits(10)
		if err != nil {
			return "", err
		}
		if group >= 1000 {
			return "", errors.New("a group of three digits above 999")
		}
		out.WriteString(strconv.Itoa(group/100) + strconv.Itoa(group/10%10) + strconv.Itoa(group%10))
	}
	switch length {
	case 2:
		pair, err := stream.readBits(7)
		if err != nil {
			return "", err
		}
		if pair >= 100 {
			return "", errors.New("a pair of digits above 99")
		}
		out.WriteString(strconv.Itoa(pair/10) + strconv.Itoa(pair%10))
	case 1:
		digit, err := stream.readBits(4)
		if err != nil {
			return "", err
		}
		if digit >= 10 {
			return "", errors.New("a digit above 9")
		}
		out.WriteString(strconv.Itoa(digit))
	}
	return out.String(), nil
}

func qrReadAlphanumeric(stream *qrBitStream, size int) (string, error) {
	length, err := stream.readBits([3]int{9, 11, 13}[size])
	if err != nil {
		return "", err
	}

	var out strings.Builder
	for ; length >= 2; length -= 2 {
		pair, err := stream.readBits(11)
		if err != nil {
			return "", err
		}
		if pair/45 >= len(qrAlphanumericCodes) {
			return "", errors.New("a character outside the alphabet")
		}
		out.WriteByte(qrAlphanumericCodes[pair/45])
		out.WriteByte(qrAlphanumericCodes[pair%45])
	}
	if length == 1 {
		single, err := stream.readBits(6)
		if err != nil {
			return "", err
		}
		if single >= len(qrAlphanumericCodes) {
			return "", errors.New("a character outside the alphabet")
		}
		out.WriteByte(qrAlphanumericCodes[single])
	}
	return out.String(), nil
}

// qrReadBytes reads raw bytes, which the reader takes to be UTF-8. A segment
// that is not valid text contributes nothing rather than failing the code, as
// the escape decoding it is ported from throws and is caught.
func qrReadBytes(stream *qrBitStream, size int) (string, error) {
	length, err := stream.readBits([3]int{8, 16, 16}[size])
	if err != nil {
		return "", err
	}

	out := make([]byte, 0, length)
	for range length {
		b, err := stream.readBits(8)
		if err != nil {
			return "", err
		}
		out = append(out, byte(b)) // #nosec G115 -- eight bits were read
	}
	if !utf8.Valid(out) {
		return "", nil
	}
	return string(out), nil
}

// qrReadKanji reads the two-byte encoding, which the reader renders through a
// table cchef does not carry.
func qrReadKanji(stream *qrBitStream, size int) (string, error) {
	length, err := stream.readBits([3]int{8, 10, 12}[size])
	if err != nil {
		return "", err
	}
	for range length {
		if _, err := stream.readBits(13); err != nil {
			return "", err
		}
	}
	return "", errors.New("the kanji mode is not supported")
}

// qrFunctionPattern marks the modules holding the patterns and information the
// data is laid out around.
func qrFunctionPattern(version qrReaderVersion) *qrBitMatrix {
	dimension := 17 + 4*version.number
	matrix := newQRBitMatrix(dimension, dimension)

	fill := func(left, top, width, height int) {
		for y := top; y < top+height; y++ {
			for x := left; x < left+width; x++ {
				matrix.set(x, y, true)
			}
		}
	}
	fill(0, 0, 9, 9)                      // the top left finder, its separator and the format
	fill(dimension-8, 0, 8, 9)            // the top right finder and the format
	fill(0, dimension-8, 9, 8)            // the bottom left finder and the format
	for _, x := range version.alignment { // the alignment patterns
		for _, y := range version.alignment {
			corner := x == 6 && y == 6 || x == 6 && y == dimension-7 || x == dimension-7 && y == 6
			if !corner {
				fill(x-2, y-2, 5, 5)
			}
		}
	}
	fill(6, 9, 1, dimension-17) // the timing patterns
	fill(9, 6, dimension-17, 1)
	if version.number > 6 {
		fill(dimension-11, 0, 3, 6) // the version information
		fill(0, dimension-11, 6, 3)
	}
	return matrix
}

// qrReadCodewords walks the data area in the order the standard lays it out,
// removing the mask as it goes.
func qrReadCodewords(matrix *qrBitMatrix, version qrReaderVersion, format qrFormatInfo) []int {
	mask := qrReaderMasks[format.mask]
	dimension := matrix.height
	functions := qrFunctionPattern(version)

	var codewords []int
	current, bitsRead := 0, 0
	readingUp := true

	for column := dimension - 1; column > 0; column -= 2 {
		if column == 6 {
			column-- // the vertical timing pattern fills a whole column
		}
		for i := range dimension {
			y := i
			if readingUp {
				y = dimension - 1 - i
			}
			for offset := range 2 {
				x := column - offset
				if functions.get(x, y) {
					continue
				}
				bitsRead++
				bit := matrix.get(x, y)
				if mask(x, y) {
					bit = !bit
				}
				current <<= 1
				if bit {
					current |= 1
				}
				if bitsRead == 8 {
					codewords = append(codewords, current)
					bitsRead, current = 0, 0
				}
			}
		}
		readingUp = !readingUp
	}
	return codewords
}

// qrReadVersion identifies the version, from the matrix size alone on the
// smaller ones and from the recorded field on the rest.
func qrReadVersion(matrix *qrBitMatrix) (qrReaderVersion, bool) {
	dimension := matrix.height
	provisional := (dimension - 17) / 4
	if provisional < 1 || provisional > len(qrReaderVersions) {
		return qrReaderVersion{}, false
	}
	if provisional <= 6 {
		return qrReaderVersions[provisional-1], true
	}

	topRight := 0
	for y := 5; y >= 0; y-- {
		for x := dimension - 9; x >= dimension-11; x-- {
			topRight = qrPushBit(topRight, matrix.get(x, y))
		}
	}
	bottomLeft := 0
	for x := 5; x >= 0; x-- {
		for y := dimension - 9; y >= dimension-11; y-- {
			bottomLeft = qrPushBit(bottomLeft, matrix.get(x, y))
		}
	}

	best, bestDifference := qrReaderVersion{}, qrMaxFieldErrors+1
	for _, version := range qrReaderVersions {
		if version.infoBits == topRight || version.infoBits == bottomLeft {
			return version, true
		}
		for _, seen := range []int{topRight, bottomLeft} {
			if difference := bits.OnesCount(uint(seen ^ version.infoBits)); difference < bestDifference {
				best, bestDifference = version, difference
			}
		}
	}
	return best, bestDifference <= qrMaxFieldErrors
}

// qrReadFormat identifies the correction level and mask, which are recorded
// twice so a damaged copy can be read from the other.
func qrReadFormat(matrix *qrBitMatrix) (qrFormatInfo, bool) {
	topLeft := 0
	for x := 0; x <= 8; x++ {
		if x != 6 { // the timing pattern interrupts the field
			topLeft = qrPushBit(topLeft, matrix.get(x, 8))
		}
	}
	for y := 7; y >= 0; y-- {
		if y != 6 {
			topLeft = qrPushBit(topLeft, matrix.get(8, y))
		}
	}

	dimension := matrix.height
	elsewhere := 0
	for y := dimension - 1; y >= dimension-7; y-- {
		elsewhere = qrPushBit(elsewhere, matrix.get(8, y))
	}
	for x := dimension - 8; x < dimension; x++ {
		elsewhere = qrPushBit(elsewhere, matrix.get(x, 8))
	}

	best, bestDifference := qrFormatInfo{}, qrMaxFieldErrors+1
	for _, bits1 := range qrFormatOrder {
		info := qrFormatTable[bits1]
		if bits1 == topLeft || bits1 == elsewhere {
			return info, true
		}
		if difference := bits.OnesCount(uint(topLeft ^ bits1)); difference < bestDifference {
			best, bestDifference = info, difference
		}
		if topLeft != elsewhere {
			if difference := bits.OnesCount(uint(elsewhere ^ bits1)); difference < bestDifference {
				best, bestDifference = info, difference
			}
		}
	}
	return best, bestDifference <= qrMaxFieldErrors
}

func qrPushBit(value int, bit bool) int {
	value <<= 1
	if bit {
		value |= 1
	}
	return value
}

// qrDataBlock is one block of codewords and how many of them carry data.
type qrDataBlock struct {
	dataCodewords int
	codewords     []int
}

// qrDataBlocks undoes the interleaving that spreads each block across the code.
func qrDataBlocks(codewords []int, version qrReaderVersion, level int) ([]qrDataBlock, bool) {
	info := version.levels[level]

	var blocks []qrDataBlock
	total := 0
	for _, group := range info.blocks {
		for range group.count {
			blocks = append(blocks, qrDataBlock{dataCodewords: group.dataCodewords})
			total += group.dataCodewords + info.ecCodewordsPerBlock
		}
	}
	// A malformed code may yield too few codewords, which nothing can mend, or
	// too many, which are simply ignored.
	if len(codewords) < total {
		return nil, false
	}
	codewords = codewords[:total]

	at := 0
	take := func(i int) {
		blocks[i].codewords = append(blocks[i].codewords, codewords[at])
		at++
	}
	for range info.blocks[0].dataCodewords {
		for i := range blocks {
			take(i)
		}
	}
	if len(info.blocks) > 1 {
		// The longer blocks carry one codeword more than the rest.
		for i := range info.blocks[1].count {
			take(info.blocks[0].count + i)
		}
	}
	for at < len(codewords) {
		for i := range blocks {
			if at < len(codewords) {
				take(i)
			}
		}
	}
	return blocks, true
}

// qrDecodeMatrix reads the text out of a sampled matrix.
func qrDecodeMatrix(matrix *qrBitMatrix) (string, bool) {
	version, ok := qrReadVersion(matrix)
	if !ok {
		return "", false
	}
	format, ok := qrReadFormat(matrix)
	if !ok {
		return "", false
	}

	blocks, ok := qrDataBlocks(qrReadCodewords(matrix, version, format), version, format.level)
	if !ok {
		return "", false
	}

	var data []int
	for _, block := range blocks {
		corrected, ok := qrReedDecode(block.codewords, len(block.codewords)-block.dataCodewords)
		if !ok {
			return "", false
		}
		data = append(data, corrected[:block.dataCodewords]...)
	}
	return qrDecodeData(data, version.number)
}

// qrDecodeMatrixOrMirror reads the matrix, trying it mirrored about its leading
// diagonal if the first attempt fails.
func qrDecodeMatrixOrMirror(matrix *qrBitMatrix) (string, bool) {
	if text, ok := qrDecodeMatrix(matrix); ok {
		return text, true
	}
	for x := range matrix.width {
		for y := x + 1; y < matrix.height; y++ {
			if matrix.get(x, y) != matrix.get(y, x) {
				matrix.set(x, y, !matrix.get(x, y))
				matrix.set(y, x, !matrix.get(y, x))
			}
		}
	}
	return qrDecodeMatrix(matrix)
}

// qrScan finds a code in a thresholded image and reads it.
func qrScan(matrix *qrBitMatrix) (string, bool) {
	for _, location := range qrLocate(matrix) {
		if text, ok := qrDecodeMatrixOrMirror(qrExtract(matrix, location)); ok {
			return text, true
		}
	}
	return "", false
}

// qrRead thresholds the image and reads whatever code it holds, trying the
// image inverted if the first pass finds nothing.
func qrRead(pixels []byte, width, height int) (string, bool) {
	binarized := qrBinarize(pixels, width, height)
	if text, ok := qrScan(binarized); ok {
		return text, true
	}

	inverted := newQRBitMatrix(width, height)
	for i, v := range binarized.data {
		inverted.data[i] = !v
	}
	return qrScan(inverted)
}
