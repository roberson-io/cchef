// Package qr generates and reads QR codes.
//
// [MatrixFor] encodes data at a chosen error-correction level, choosing the
// smallest version that fits and the mask that scores best under the standard
// penalty rules. [Read] goes the other way, locating a code in a bitmap and
// decoding it.
package qr

import (
	"errors"
	"regexp"
	"strconv"
)

// The QR code foundation, ported from the qr-image package CyberChef wraps
// (CyberChef's node_modules/qr-image/lib/{encode,errorcode,qr-base,matrix}.js).
// The matrix is spec-deterministic, but the choices around it — which mode
// encodes the data, which version holds it, and which mask is applied — are the
// library's, and every renderer depends on all three agreeing.

// qrErrorLevels are the error correction levels, in the order the format
// information numbers them.
var qrErrorLevels = [4]string{"L", "M", "Q", "H"}

// qrMessage holds the encoded data as one bit per element, in the three
// versions of the length field the standard defines. The shorter fields cannot
// hold every length, so those entries may be absent.
type qrMessage struct {
	data1  []byte // versions 1 to 9
	data10 []byte // versions 10 to 26
	data27 []byte // versions 27 to 40
}

// qrPushBits appends the low n bits of value, most significant first.
func qrPushBits(bits []byte, n int, value int) []byte {
	for bit := 1 << (n - 1); bit != 0; bit >>= 1 {
		if bit&value != 0 {
			bits = append(bits, 1)
		} else {
			bits = append(bits, 0)
		}
	}
	return bits
}

// qrHeader starts a segment with its mode indicator and character count.
func qrHeader(mode []byte, countBits, count int) []byte {
	return qrPushBits(append([]byte{}, mode...), countBits, count)
}

// The mode indicators, and the largest input each mode can carry.
var (
	qrModeNumeric      = []byte{0, 0, 0, 1}
	qrModeAlphanumeric = []byte{0, 0, 1, 0}
	qrModeByte         = []byte{0, 1, 0, 0}
)

const (
	qrMaxNumeric      = 7089
	qrMaxAlphanumeric = 4296
	qrMaxByte         = 2953
)

// qrAlphanumeric is the alphabet of the alphanumeric mode, whose index is the
// value each character encodes as.
const qrAlphanumeric = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

// qrAlphanumericValues indexes that alphabet by character, which the mode's
// pattern has already restricted the input to.
var qrAlphanumericValues = func() [256]int {
	var values [256]int
	for i := range len(qrAlphanumeric) {
		values[qrAlphanumeric[i]] = i
	}
	return values
}()

var (
	qrNumericPattern      = regexp.MustCompile(`^[0-9]+$`)
	qrAlphanumericPattern = regexp.MustCompile(`^[0-9A-Z \$%\*\+\.\/\:\-]+$`)
)

// qrEncodeByte encodes the data as raw bytes, which every input can use.
func qrEncodeByte(data []byte) qrMessage {
	bits := []byte{}
	for _, b := range data {
		bits = qrPushBits(bits, 8, int(b))
	}

	var msg qrMessage
	msg.data10 = append(qrHeader(qrModeByte, 16, len(data)), bits...)
	msg.data27 = msg.data10
	if len(data) < 256 {
		msg.data1 = append(qrHeader(qrModeByte, 8, len(data)), bits...)
	}
	return msg
}

// qrEncodeAlphanumeric packs two characters into eleven bits.
func qrEncodeAlphanumeric(str string) qrMessage {
	bits := []byte{}
	for i := 0; i < len(str); i += 2 {
		width, n := 6, qrAlphanumericValues[str[i]]
		if i+1 < len(str) {
			width, n = 11, n*45+qrAlphanumericValues[str[i+1]]
		}
		bits = qrPushBits(bits, width, n)
	}

	var msg qrMessage
	msg.data27 = append(qrHeader(qrModeAlphanumeric, 13, len(str)), bits...)
	if len(str) < 2048 {
		msg.data10 = append(qrHeader(qrModeAlphanumeric, 11, len(str)), bits...)
	}
	if len(str) < 512 {
		msg.data1 = append(qrHeader(qrModeAlphanumeric, 9, len(str)), bits...)
	}
	return msg
}

// qrEncodeNumeric packs three digits into ten bits.
func qrEncodeNumeric(str string) qrMessage {
	bits := []byte{}
	for i := 0; i < len(str); i += 3 {
		end := min(i+3, len(str))
		group := str[i:end]
		n, _ := strconv.Atoi(group)
		bits = qrPushBits(bits, (len(group)*10+2)/3, n)
	}

	var msg qrMessage
	msg.data27 = append(qrHeader(qrModeNumeric, 14, len(str)), bits...)
	if len(str) < 4096 {
		msg.data10 = append(qrHeader(qrModeNumeric, 12, len(str)), bits...)
	}
	if len(str) < 1024 {
		msg.data1 = append(qrHeader(qrModeNumeric, 10, len(str)), bits...)
	}
	return msg
}

// errQRTooMuchData is returned for input no version can hold.
var errQRTooMuchData = errors.New("too much data")

// qrEncode chooses the narrowest mode the data fits and encodes it.
func qrEncode(data []byte) (qrMessage, error) {
	str := string(data)
	switch {
	case qrNumericPattern.MatchString(str):
		if len(data) > qrMaxNumeric {
			return qrMessage{}, errQRTooMuchData
		}
		return qrEncodeNumeric(str), nil
	case qrAlphanumericPattern.MatchString(str):
		if len(data) > qrMaxAlphanumeric {
			return qrMessage{}, errQRTooMuchData
		}
		return qrEncodeAlphanumeric(str), nil
	}
	if len(data) > qrMaxByte {
		return qrMessage{}, errQRTooMuchData
	}
	return qrEncodeByte(data), nil
}

// The field the error correction codewords are computed over.
const (
	qrFieldBase = 285
	qrFieldSize = 255
)

// qrExpTable and qrLogTable are the exponential and logarithm tables of the
// field, which turn its multiplication into addition.
var qrExpTable, qrLogTable = func() ([256]int, [256]int) {
	var exp, log [256]int
	exp[0] = 1
	for i := 1; i < 256; i++ {
		n := exp[i-1] << 1
		if n > 255 {
			n ^= qrFieldBase
		}
		exp[i] = n
	}
	for i := range qrFieldSize {
		log[exp[i]] = i
	}
	return exp, log
}()

// qrExp raises the field's generator to a power, which the polynomials index
// past the end of the table.
func qrExp(k int) int {
	for k > qrFieldSize {
		k -= qrFieldSize
	}
	return qrExpTable[k]
}

// qrGeneratorPolynomial builds the generator polynomial of the given degree,
// each coefficient held as its logarithm.
func qrGeneratorPolynomial(degree int) []int {
	poly := []int{0}
	for d := 1; d <= degree; d++ {
		next := make([]int, d+1)
		next[0] = poly[0]
		for i := 1; i <= d; i++ {
			// The polynomial of the previous degree carries no coefficient at
			// the new top, which counts as zero.
			high := 0
			if i < len(poly) {
				high = qrExp(poly[i])
			}
			next[i] = qrLogTable[high^qrExp(poly[i-1]+d-1)]
		}
		poly = next
	}
	return poly
}

// qrCalculateEC divides the message by the generator polynomial and returns the
// remainder, which is the block's error correction codewords.
func qrCalculateEC(msg []byte, ecLen int) []byte {
	work := make([]int, 0, len(msg)+ecLen)
	for _, b := range msg {
		work = append(work, int(b))
	}
	work = append(work, make([]int, ecLen)...)

	poly := qrGeneratorPolynomial(ecLen)
	for len(work) > ecLen {
		if work[0] == 0 {
			work = work[1:]
			continue
		}
		factor := qrLogTable[work[0]]
		for i := 0; i <= ecLen; i++ {
			work[i] ^= qrExp(poly[i] + factor)
		}
		work = work[1:]
	}

	out := make([]byte, len(work))
	for i, v := range work {
		out[i] = byte(v) // #nosec G115 -- the field holds values of one byte
	}
	return out
}

// qrVersions holds, for each version, the total number of codewords followed by
// the error correction codeword count and block count of each level in turn.
var qrVersions = [41][9]int{
	{}, // there is no version 0
	{26, 7, 1, 10, 1, 13, 1, 17, 1},
	{44, 10, 1, 16, 1, 22, 1, 28, 1},
	{70, 15, 1, 26, 1, 36, 2, 44, 2},
	{100, 20, 1, 36, 2, 52, 2, 64, 4},
	{134, 26, 1, 48, 2, 72, 4, 88, 4},
	{172, 36, 2, 64, 4, 96, 4, 112, 4},
	{196, 40, 2, 72, 4, 108, 6, 130, 5},
	{242, 48, 2, 88, 4, 132, 6, 156, 6},
	{292, 60, 2, 110, 5, 160, 8, 192, 8},
	{346, 72, 4, 130, 5, 192, 8, 224, 8},
	{404, 80, 4, 150, 5, 224, 8, 264, 11},
	{466, 96, 4, 176, 8, 260, 10, 308, 11},
	{532, 104, 4, 198, 9, 288, 12, 352, 16},
	{581, 120, 4, 216, 9, 320, 16, 384, 16},
	{655, 132, 6, 240, 10, 360, 12, 432, 18},
	{733, 144, 6, 280, 10, 408, 17, 480, 16},
	{815, 168, 6, 308, 11, 448, 16, 532, 19},
	{901, 180, 6, 338, 13, 504, 18, 588, 21},
	{991, 196, 7, 364, 14, 546, 21, 650, 25},
	{1085, 224, 8, 416, 16, 600, 20, 700, 25},
	{1156, 224, 8, 442, 17, 644, 23, 750, 25},
	{1258, 252, 9, 476, 17, 690, 23, 816, 34},
	{1364, 270, 9, 504, 18, 750, 25, 900, 30},
	{1474, 300, 10, 560, 20, 810, 27, 960, 32},
	{1588, 312, 12, 588, 21, 870, 29, 1050, 35},
	{1706, 336, 12, 644, 23, 952, 34, 1110, 37},
	{1828, 360, 12, 700, 25, 1020, 34, 1200, 40},
	{1921, 390, 13, 728, 26, 1050, 35, 1260, 42},
	{2051, 420, 14, 784, 28, 1140, 38, 1350, 45},
	{2185, 450, 15, 812, 29, 1200, 40, 1440, 48},
	{2323, 480, 16, 868, 31, 1290, 43, 1530, 51},
	{2465, 510, 17, 924, 33, 1350, 45, 1620, 54},
	{2611, 540, 18, 980, 35, 1440, 48, 1710, 57},
	{2761, 570, 19, 1036, 37, 1530, 51, 1800, 60},
	{2876, 570, 19, 1064, 38, 1590, 53, 1890, 63},
	{3034, 600, 20, 1120, 40, 1680, 56, 1980, 66},
	{3196, 630, 21, 1204, 43, 1770, 59, 2100, 70},
	{3362, 660, 22, 1260, 45, 1860, 62, 2220, 74},
	{3532, 720, 24, 1316, 47, 1950, 65, 2310, 77},
	{3706, 750, 25, 1372, 49, 2040, 68, 2430, 81},
}

// qrTemplate describes how one version at one error correction level lays its
// codewords out: how many the data occupies, how they divide into blocks, and
// how many error correction codewords each block carries.
type qrTemplate struct {
	version int
	ecLevel string
	dataLen int
	ecLen   int
	blocks  [][]byte
	ec      [][]byte
	sizes   []int
}

// qrTemplateFor builds the template of one version and level.
func qrTemplateFor(version int, level int) qrTemplate {
	row := qrVersions[version]
	field := level*2 + 1
	dataLen := row[0] - row[field]
	count := row[field+1]

	template := qrTemplate{
		version: version,
		ecLevel: qrErrorLevels[level],
		dataLen: dataLen,
		ecLen:   row[field] / count,
	}
	for k, n := count, dataLen; k > 0; k-- {
		block := n / k
		template.sizes = append(template.sizes, block)
		n -= block
	}
	return template
}

// The version ranges the three length fields cover.
const (
	qrShortVersions  = 10
	qrMediumVersions = 27
	qrVersionCount   = 41
)

// qrChooseTemplate picks the smallest version that holds the message, taking
// the length field that version's range uses.
func qrChooseTemplate(msg qrMessage, level int) (qrTemplate, error) {
	search := func(from, to int, bits []byte) (qrTemplate, bool) {
		if bits == nil {
			return qrTemplate{}, false
		}
		length := (len(bits) + 7) / 8
		for version := from; version < to; version++ {
			if template := qrTemplateFor(version, level); template.dataLen >= length {
				return template, true
			}
		}
		return qrTemplate{}, false
	}

	if template, ok := search(1, qrShortVersions, msg.data1); ok {
		return template, nil
	}
	if template, ok := search(qrShortVersions, qrMediumVersions, msg.data10); ok {
		return template, nil
	}
	if template, ok := search(qrMediumVersions, qrVersionCount, msg.data27); ok {
		return template, nil
	}
	return qrTemplate{}, errQRTooMuchData
}

// The two padding codewords, which alternate to fill the data area.
const (
	qrFirstPad      = 236
	qrSecondPad     = 17
	qrTerminatorLen = 4
)

// qrFillTemplate packs the message bits into codewords, pads the remainder, and
// computes the error correction codewords of each block.
func qrFillTemplate(msg qrMessage, template qrTemplate) qrTemplate {
	bits := msg.data27
	switch {
	case template.version < qrShortVersions:
		bits = msg.data1
	case template.version < qrMediumVersions:
		bits = msg.data10
	}

	codewords := make([]byte, template.dataLen)
	for i := 0; i < len(bits); i += 8 {
		var b int
		for j := range 8 {
			b <<= 1
			if i+j < len(bits) && bits[i+j] != 0 {
				b |= 1
			}
		}
		codewords[i/8] = byte(b)
	}

	// The terminator rounds the message up to a whole codeword; everything
	// after it alternates between the two padding values.
	pad := byte(qrFirstPad)
	for i := (len(bits) + qrTerminatorLen + 7) / 8; i < len(codewords); i++ {
		codewords[i] = pad
		if pad == qrFirstPad {
			pad = qrSecondPad
		} else {
			pad = qrFirstPad
		}
	}

	offset := 0
	for _, n := range template.sizes {
		block := codewords[offset : offset+n]
		offset += n
		template.blocks = append(template.blocks, block)
		template.ec = append(template.ec, qrCalculateEC(block, template.ecLen))
	}
	return template
}

// The two high bits mark a cell as part of a function pattern rather than data:
// qrReserved alone is a light cell, qrReserved|1 a dark one.
const (
	qrReserved = 0x80
	qrDark     = 0x81
)

// qrInitMatrix makes an empty matrix of the version's size.
func qrInitMatrix(version int) [][]byte {
	n := version*4 + 17
	matrix := make([][]byte, n)
	for i := range matrix {
		matrix[i] = make([]byte, n)
	}
	return matrix
}

// qrFillFinders places the three finder patterns and the separators round them.
func qrFillFinders(matrix [][]byte) {
	n := len(matrix)
	for i := -3; i <= 3; i++ {
		for j := -3; j <= 3; j++ {
			hi, lo := max(i, j), min(i, j)
			pixel := byte(qrDark)
			if (hi == 2 && lo >= -2) || (lo == -2 && hi <= 2) {
				pixel = qrReserved
			}
			matrix[3+i][3+j] = pixel
			matrix[3+i][n-4+j] = pixel
			matrix[n-4+i][3+j] = pixel
		}
	}
	for i := range 8 {
		matrix[7][i] = qrReserved
		matrix[i][7] = qrReserved
		matrix[7][n-i-1] = qrReserved
		matrix[i][n-8] = qrReserved
		matrix[n-8][i] = qrReserved
		matrix[n-1-i][7] = qrReserved
	}
}

// The alignment patterns, whose spacing the library derives rather than looks
// up, and the size below which there are none.
const (
	qrSmallestAligned = 21
	qrAlignSpan       = 28
)

// qrFillAlignAndTiming places the alignment patterns and the two timing lines.
func qrFillAlignAndTiming(matrix [][]byte) {
	n := len(matrix)
	if n > qrSmallestAligned {
		span := n - 13
		steps := (span + qrAlignSpan - 1) / qrAlignSpan
		delta := (span + steps/2) / steps // rounded to nearest
		if delta%2 != 0 {
			delta++
		}

		var centres []int
		for p := span + 6; p > 10; p -= delta {
			centres = append([]int{p}, centres...)
		}
		centres = append([]int{6}, centres...)

		for _, x := range centres {
			for _, y := range centres {
				if matrix[x][y] != 0 {
					continue
				}
				for r := -2; r <= 2; r++ {
					for c := -2; c <= 2; c++ {
						hi, lo := max(r, c), min(r, c)
						pixel := byte(qrDark)
						if (hi == 1 && lo >= -1) || (lo == -1 && hi <= 1) {
							pixel = qrReserved
						}
						matrix[x+r][y+c] = pixel
					}
				}
			}
		}
	}

	for i := 8; i < n-8; i++ {
		pixel := byte(qrDark)
		if i%2 != 0 {
			pixel = qrReserved
		}
		matrix[6][i] = pixel
		matrix[i][6] = pixel
	}
}

// qrFillStub reserves the areas the format and version information occupy, so
// the data placement walks past them.
func qrFillStub(matrix [][]byte) {
	n := len(matrix)
	for i := range 8 {
		if i != 6 {
			matrix[8][i] = qrReserved
			matrix[i][8] = qrReserved
		}
		matrix[8][n-1-i] = qrReserved
		matrix[n-1-i][8] = qrReserved
	}
	matrix[8][8] = qrReserved
	matrix[n-8][8] = qrDark

	if n < 45 {
		return
	}
	for i := n - 11; i < n-8; i++ {
		for j := range 6 {
			matrix[i][j] = qrReserved
			matrix[j][i] = qrReserved
		}
	}
}

// The generator polynomials protecting the format and version information, and
// the mask the format is exclusive-ored with so it is never all zeros.
const (
	qrFormatPoly     = 0x0537
	qrVersionPoly    = 0x1f25
	qrFormatMask     = 0x5412
	qrFirstVersioned = 7
)

// qrFormats and qrVersionInfo hold those two fields with their check bits.
var qrFormats, qrVersionInfo = func() ([32]int, [41]int) {
	var formats [32]int
	for format := range 32 {
		res := format << 10
		for i := 5; i > 0; i-- {
			if res>>(9+i) != 0 {
				res ^= qrFormatPoly << (i - 1)
			}
		}
		formats[format] = (res | format<<10) ^ qrFormatMask
	}

	var versions [41]int
	for version := qrFirstVersioned; version <= 40; version++ {
		res := version << 12
		for i := 6; i > 0; i-- {
			if res>>(11+i) != 0 {
				res ^= qrVersionPoly << (i - 1)
			}
		}
		versions[version] = res | version<<12
	}
	return formats, versions
}()

// qrFormatLevels number the error correction levels the way the format
// information does, which is not the order they are named in.
var qrFormatLevels = map[string]int{"L": 1, "M": 0, "Q": 3, "H": 2}

// qrFillReserved writes the format information, and the version information on
// the versions large enough to carry it.
func qrFillReserved(matrix [][]byte, ecLevel string, mask int) {
	n := len(matrix)
	format := qrFormats[qrFormatLevels[ecLevel]<<3|mask]
	bit := func(k int) byte {
		if format>>k&1 != 0 {
			return qrDark
		}
		return qrReserved
	}

	for i := range 8 {
		matrix[8][n-1-i] = bit(i)
		if i < 6 {
			matrix[i][8] = bit(i)
		}
	}
	for i := 8; i < 15; i++ {
		matrix[n-15+i][8] = bit(i)
		if i > 8 {
			matrix[8][14-i] = bit(i)
		}
	}
	matrix[7][8] = bit(6)
	matrix[8][8] = bit(7)
	matrix[8][7] = bit(8)

	version := qrVersionInfo[(n-17)/4]
	if version == 0 {
		return
	}
	versionBit := func(k int) byte {
		if version>>k&1 != 0 {
			return qrDark
		}
		return qrReserved
	}
	for i := range 6 {
		for j := range 3 {
			matrix[n-11+j][i] = versionBit(i*3 + j)
			matrix[i][n-11+j] = versionBit(i*3 + j)
		}
	}
}

// qrMasks are the eight patterns a matrix may be exclusive-ored with, one of
// which is chosen by the penalty score it leaves behind.
var qrMasks = [8]func(i, j int) bool{
	func(i, j int) bool { return (i+j)%2 == 0 },
	func(i, j int) bool { return i%2 == 0 },
	func(i, j int) bool { return j%3 == 0 },
	func(i, j int) bool { return (i+j)%3 == 0 },
	func(i, j int) bool { return (i/2+j/3)%2 == 0 },
	func(i, j int) bool { return (i*j)%2+(i*j)%3 == 0 },
	func(i, j int) bool { return ((i*j)%2+(i*j)%3)%2 == 0 },
	func(i, j int) bool { return ((i*j)%3+(i+j)%2)%2 == 0 },
}

// qrPlacement walks the data area in the zigzag the standard defines, skipping
// every cell a function pattern already holds.
type qrPlacement struct {
	matrix [][]byte
	n      int
	row    int
	col    int
	dir    int
	mask   func(i, j int) bool
}

// next steps to the following data cell, reporting whether one remains.
func (p *qrPlacement) next() bool {
	for {
		upward := 0
		if p.col < 6 {
			upward = 1
		}
		if p.col%2^upward != 0 {
			if p.dir < 0 && p.row == 0 || p.dir > 0 && p.row == p.n-1 {
				p.col--
				p.dir = -p.dir
			} else {
				p.col++
				p.row += p.dir
			}
		} else {
			p.col--
		}
		// The vertical timing pattern occupies a whole column, which the walk
		// steps over rather than through.
		if p.col == 6 {
			p.col--
		}
		if p.col < 0 {
			return false
		}
		if p.matrix[p.row][p.col]&0xF0 == 0 {
			return true
		}
	}
}

// put writes one codeword, most significant bit first, masking as it goes.
func (p *qrPlacement) put(codeword byte) {
	for bit := byte(0x80); bit != 0; bit >>= 1 {
		pixel := bit&codeword != 0
		if p.mask(p.row, p.col) {
			pixel = !pixel
		}
		if pixel {
			p.matrix[p.row][p.col] = 1
		} else {
			p.matrix[p.row][p.col] = 0
		}
		p.next()
	}
}

// qrFillData interleaves the blocks and their error correction codewords into
// the data area.
func qrFillData(matrix [][]byte, data qrTemplate, mask int) {
	n := len(matrix)
	p := &qrPlacement{matrix: matrix, n: n, row: n - 1, col: n - 1, dir: -1, mask: qrMasks[mask]}

	longest := len(data.blocks[len(data.blocks)-1])
	for i := range longest {
		for _, block := range data.blocks {
			if len(block) > i {
				p.put(block[i])
			}
		}
	}
	for i := range data.ecLen {
		for _, block := range data.ec {
			p.put(block[i])
		}
	}

	// The last few versions leave remainder cells the codewords do not reach,
	// which carry the mask pattern alone.
	if p.col > -1 {
		for {
			if p.mask(p.row, p.col) {
				p.matrix[p.row][p.col] = 1
			} else {
				p.matrix[p.row][p.col] = 0
			}
			if !p.next() {
				break
			}
		}
	}
}

// The weights the standard gives three of the four penalty rules; a run scores
// its own length less two.
const (
	qrBlockWeight   = 3
	qrFinderWeight  = 40
	qrBalanceWeight = 10
	qrRunLength     = 5
)

// qrRunPenaltyOf scores one line of the matrix for runs of five or more cells
// of the same colour.
func qrRunPenaltyOf(at func(k int) byte, length int) int {
	penalty := 0
	pixel := at(0) & 1
	run := 1
	for k := 1; k < length; k++ {
		p := at(k) & 1
		if p == pixel {
			run++
			continue
		}
		if run >= qrRunLength {
			penalty += run - 2
		}
		pixel = p
		run = 1
	}
	if run >= qrRunLength {
		penalty += run - 2
	}
	return penalty
}

// qrCalculatePenalty scores a masked matrix by the four rules, the lowest of
// which chooses the mask.
func qrCalculatePenalty(matrix [][]byte) int {
	return qrRunPenalty(matrix) + qrBlockPenaltyOf(matrix) +
		qrFinderLikePenalty(matrix) + qrBalancePenaltyOf(matrix)
}

// qrRunPenalty scores every row and column for runs of five or more modules of
// the same colour.
func qrRunPenalty(matrix [][]byte) int {
	n := len(matrix)
	penalty := 0
	for i := range n {
		penalty += qrRunPenaltyOf(func(k int) byte { return matrix[i][k] }, n)
	}
	for j := range n {
		penalty += qrRunPenaltyOf(func(k int) byte { return matrix[k][j] }, n)
	}
	return penalty
}

// qrBlockPenaltyOf scores every square of four modules that share a colour. The
// sum keeps the markers the function patterns carry, so it is taken modulo
// eight before being tested, as the library does.
func qrBlockPenaltyOf(matrix [][]byte) int {
	n := len(matrix)
	penalty := 0
	for i := range n - 1 {
		for j := range n - 1 {
			s := (int(matrix[i][j]) + int(matrix[i][j+1]) +
				int(matrix[i+1][j]) + int(matrix[i+1][j+1])) & 7
			if s == 0 || s == 4 {
				penalty += qrBlockWeight
			}
		}
	}
	return penalty
}

// qrFinderLikePenalty scores sequences resembling a finder pattern, in either
// direction, with four light modules to one side or the other.
func qrFinderLikePenalty(matrix [][]byte) int {
	n := len(matrix)
	penalty := 0
	for i := range n {
		for j := range n {
			penalty += qrFinderLikeAt(func(k int) bool { return matrix[i][j+k]&1 != 0 }, j, n)
			penalty += qrFinderLikeAt(func(k int) bool { return matrix[i+k][j]&1 != 0 }, i, n)
		}
	}
	return penalty
}

// qrFinderLikeAt scores one line where the sequence starts at the given
// position, once for each side that carries four light modules beside it.
func qrFinderLikeAt(module func(k int) bool, at, n int) int {
	if at >= n-6 || !qrIsFinderSequence(module) {
		return 0
	}
	penalty := 0
	if at >= 4 && !module(-4) && !module(-3) && !module(-2) && !module(-1) {
		penalty += qrFinderWeight
	}
	if at < n-10 && !module(7) && !module(8) && !module(9) && !module(10) {
		penalty += qrFinderWeight
	}
	return penalty
}

// qrIsFinderSequence reports whether a line reads dark, light, three dark,
// light, dark, which is the proportion a finder pattern shows.
func qrIsFinderSequence(module func(k int) bool) bool {
	return module(0) && !module(1) && module(2) && module(3) &&
		module(4) && !module(5) && module(6)
}

// qrBalancePenaltyOf scores how far the proportion of dark modules strays from
// half, in steps of a twentieth.
func qrBalancePenaltyOf(matrix [][]byte) int {
	n := len(matrix)
	dark := 0
	for i := range n {
		for j := range n {
			if matrix[i][j]&1 != 0 {
				dark++
			}
		}
	}
	balance := 10 - 20*float64(dark)/float64(n*n)
	if balance < 0 {
		balance = -balance
	}
	return qrBalanceWeight * int(balance)
}

// qrGetMatrix builds the finished matrix, trying every mask and keeping the one
// that scores lowest.
func qrGetMatrix(data qrTemplate) [][]byte {
	matrix := qrInitMatrix(data.version)
	qrFillFinders(matrix)
	qrFillAlignAndTiming(matrix)
	qrFillStub(matrix)

	best, lowest := 0, -1
	for mask := range 8 {
		qrFillData(matrix, data, mask)
		qrFillReserved(matrix, data.ecLevel, mask)
		if p := qrCalculatePenalty(matrix); lowest < 0 || p < lowest {
			lowest = p
			best = mask
		}
	}

	qrFillData(matrix, data, best)
	qrFillReserved(matrix, data.ecLevel, best)

	for _, row := range matrix {
		for j := range row {
			row[j] &= 1
		}
	}
	return matrix
}

// MatrixFor encodes the input and returns the finished matrix.
func MatrixFor(data []byte, ecLevel string) ([][]byte, error) {
	level := 1 // the medium level, which an unknown name falls back to
	for i, name := range qrErrorLevels {
		if name == ecLevel {
			level = i
		}
	}

	msg, err := qrEncode(data)
	if err != nil {
		return nil, err
	}
	template, err := qrChooseTemplate(msg, level)
	if err != nil {
		return nil, err
	}
	return qrGetMatrix(qrFillTemplate(msg, template)), nil
}
