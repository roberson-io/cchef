package qr

import (
	"strings"
	"testing"
)

// TestQRFieldArithmetic covers the Galois field the error correction is
// computed over, by the laws it has to obey rather than by recorded values.
func TestQRFieldArithmetic(t *testing.T) {
	f := newQRField(qrReedPrimitive, qrReedSize, 0)

	t.Run("multiplication has an identity and an absorbing element", func(t *testing.T) {
		for a := range qrReedSize {
			if got := f.multiply(a, 1); got != a {
				t.Fatalf("%d times one is %d", a, got)
			}
			if got := f.multiply(a, 0); got != 0 {
				t.Fatalf("%d times zero is %d", a, got)
			}
		}
	})

	t.Run("every element but zero has an inverse", func(t *testing.T) {
		for a := 1; a < qrReedSize; a++ {
			if got := f.multiply(a, f.inverse(a)); got != 1 {
				t.Fatalf("%d times its inverse is %d, want 1", a, got)
			}
		}
	})

	t.Run("multiplication commutes and associates", func(t *testing.T) {
		for a := 1; a < 40; a++ {
			for b := 1; b < 40; b++ {
				if f.multiply(a, b) != f.multiply(b, a) {
					t.Fatalf("%d and %d do not commute", a, b)
				}
				if f.multiply(f.multiply(a, b), 7) != f.multiply(a, f.multiply(b, 7)) {
					t.Fatalf("%d and %d do not associate", a, b)
				}
			}
		}
	})
}

// TestQRPolynomialArithmetic covers the polynomials the correction is carried
// out on, including the short paths for zero and one that a clean block never
// reaches.
func TestQRPolynomialArithmetic(t *testing.T) {
	f := newQRField(qrReedPrimitive, qrReedSize, 0)
	p := newQRPoly(f, []int{3, 1, 4, 1, 5})

	t.Run("leading zeros are dropped", func(t *testing.T) {
		if got := newQRPoly(f, []int{0, 0, 7, 2}); got.degree() != 1 {
			t.Errorf("degree is %d, want 1", got.degree())
		}
		if got := newQRPoly(f, []int{0, 0, 0}); !got.isZero() {
			t.Error("a polynomial of only zeros is not zero")
		}
	})

	t.Run("scaling by zero and one", func(t *testing.T) {
		if !p.scale(0).isZero() {
			t.Error("scaling by zero did not give zero")
		}
		if p.scale(1) != p {
			t.Error("scaling by one did not give the same polynomial")
		}
	})

	t.Run("a product with zero is zero", func(t *testing.T) {
		if !p.multiply(f.zero).isZero() {
			t.Error("multiplying by zero did not give zero")
		}
		if !f.zero.multiply(p).isZero() {
			t.Error("zero multiplied by anything did not give zero")
		}
		if !p.multiplyByMonomial(2, 0).isZero() {
			t.Error("multiplying by a zero monomial did not give zero")
		}
		if !f.monomial(3, 0).isZero() {
			t.Error("a monomial with a zero coefficient is not zero")
		}
	})

	t.Run("a sum with zero is unchanged", func(t *testing.T) {
		if p.add(f.zero) != p {
			t.Error("adding zero changed the polynomial")
		}
		if f.zero.add(p) != p {
			t.Error("adding to zero changed the polynomial")
		}
	})

	t.Run("evaluation at zero and one", func(t *testing.T) {
		if got, want := p.evaluateAt(0), p.coefficient(0); got != want {
			t.Errorf("at zero it is %d, want its constant term %d", got, want)
		}
		sum := 0
		for _, c := range p.coefficients {
			sum ^= c
		}
		if got := p.evaluateAt(1); got != sum {
			t.Errorf("at one it is %d, want the sum of its coefficients %d", got, sum)
		}
	})

	t.Run("multiplication distributes over addition", func(t *testing.T) {
		q := newQRPoly(f, []int{9, 2, 6})
		r := newQRPoly(f, []int{5, 3})
		left := p.multiply(q.add(r))
		right := p.multiply(q).add(p.multiply(r))
		if len(left.coefficients) != len(right.coefficients) {
			t.Fatalf("degrees differ: %d and %d", left.degree(), right.degree())
		}
		for i := range left.coefficients {
			if left.coefficients[i] != right.coefficients[i] {
				t.Fatalf("coefficient %d differs: %d and %d", i, left.coefficients[i], right.coefficients[i])
			}
		}
	})
}

// TestQRBitStreamBounds covers the reader's refusal to take more bits than the
// codewords hold.
func TestQRBitStreamBounds(t *testing.T) {
	s := &qrBitStream{bytes: []int{0xAB, 0xCD}}
	if got, err := s.readBits(4); err != nil || got != 0xA {
		t.Fatalf("read %#x, %v; want 0xa", got, err)
	}
	if got, err := s.readBits(8); err != nil || got != 0xBC {
		t.Fatalf("read %#x, %v; want 0xbc", got, err)
	}
	for _, n := range []int{0, 33, 5} {
		if _, err := s.readBits(n); err == nil {
			t.Errorf("reading %d bits was allowed with %d remaining", n, s.available())
		}
	}
}

// qrBits builds codewords from a list of values and the width each occupies,
// padding the last byte with zeros as a real code does.
func qrBits(pairs ...[2]int) []int {
	var bits []int
	for _, p := range pairs {
		for i := p[1] - 1; i >= 0; i-- {
			bits = append(bits, p[0]>>i&1)
		}
	}
	codewords := make([]int, (len(bits)+7)/8)
	for i, bit := range bits {
		codewords[i/8] |= bit << (7 - i%8)
	}
	return codewords
}

// The segment headers, which the reader dispatches on.
const (
	qrTestNumeric = 1
	qrTestAlpha   = 2
	qrTestByte    = 4
	qrTestKanji   = 8
	qrTestECI     = 7
	qrTestEnd     = 0
)

// TestQRDecodeDataSegments covers the segment reader over the shapes a real
// code never produced in the round trips: character sets, kanji, remainders and
// truncated or invalid segments.
func TestQRDecodeDataSegments(t *testing.T) {
	for _, tc := range []struct {
		name      string
		codewords []int
		version   int
		want      string
		ok        bool
	}{
		{
			"a character set marker of one byte",
			qrBits([2]int{qrTestECI, 4}, [2]int{0, 1}, [2]int{26, 7},
				[2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{qrTestEnd, 4}),
			1, "7", true,
		},
		{
			"one of two bytes",
			qrBits([2]int{qrTestECI, 4}, [2]int{1, 1}, [2]int{0, 1}, [2]int{999, 14},
				[2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{qrTestEnd, 4}),
			1, "7", true,
		},
		{
			"one of three bytes",
			qrBits([2]int{qrTestECI, 4}, [2]int{1, 1}, [2]int{1, 1}, [2]int{0, 1}, [2]int{5, 21},
				[2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{qrTestEnd, 4}),
			1, "7", true,
		},
		{
			"a corrupted character set marker",
			qrBits([2]int{qrTestECI, 4}, [2]int{1, 1}, [2]int{1, 1}, [2]int{1, 1},
				[2]int{qrTestEnd, 4}),
			1, "", true,
		},
		{
			"numeric leaving two digits",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{2, 10}, [2]int{42, 7}, [2]int{qrTestEnd, 4}),
			1, "42", true,
		},
		{
			"a numeric group above nine hundred and ninety-nine",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{3, 10}, [2]int{1000, 10}),
			1, "", false,
		},
		{
			"a numeric pair above ninety-nine",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{2, 10}, [2]int{100, 7}),
			1, "", false,
		},
		{
			"a single numeric digit above nine",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{10, 4}),
			1, "", false,
		},
		{
			"an alphanumeric pair outside the alphabet",
			qrBits([2]int{qrTestAlpha, 4}, [2]int{2, 9}, [2]int{2047, 11}),
			1, "", false,
		},
		{
			"a single alphanumeric outside the alphabet",
			qrBits([2]int{qrTestAlpha, 4}, [2]int{1, 9}, [2]int{63, 6}),
			1, "", false,
		},
		{
			"kanji, which the reader does not carry the table for",
			qrBits([2]int{qrTestKanji, 4}, [2]int{1, 8}, [2]int{0x123, 13}),
			1, "", false,
		},
		{
			"a mode the reader does not know",
			qrBits([2]int{3, 4}, [2]int{0, 8}),
			1, "", false,
		},
		{
			"a segment whose length runs past the codewords",
			qrBits([2]int{qrTestByte, 4}, [2]int{40, 8}, [2]int{1, 8}),
			1, "", false,
		},
		{
			"a numeric segment cut short",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{9, 10}),
			1, "", false,
		},
		{
			"an alphanumeric segment cut short",
			qrBits([2]int{qrTestAlpha, 4}, [2]int{9, 9}),
			1, "", false,
		},
		{
			"padding that is not zero",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{0xFF, 8}),
			1, "", false,
		},
		{
			"the widest character counts, past version twenty-six",
			qrBits([2]int{qrTestNumeric, 4}, [2]int{1, 14}, [2]int{7, 4}, [2]int{qrTestEnd, 4}),
			27, "7", true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := qrDecodeData(tc.codewords, tc.version)
			if ok != tc.ok {
				t.Fatalf("read %q, %v; want ok=%v", got, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("read %q, want %q", got, tc.want)
			}
		})
	}
}

// qrMatrixFromEncoder builds a sampled matrix the way the reader would see one,
// by asking the generator for the modules of a code.
func qrMatrixFromEncoder(t *testing.T, text, level string) *qrBitMatrix {
	t.Helper()
	modules, err := MatrixFor([]byte(text), level)
	if err != nil {
		t.Fatalf("build a matrix: %v", err)
	}
	matrix := newQRBitMatrix(len(modules), len(modules))
	for y, row := range modules {
		for x, module := range row {
			matrix.set(x, y, module != 0)
		}
	}
	return matrix
}

// TestQRReedDecodeLimits covers the correction at and past what it can mend.
// A block carrying n check codewords corrects up to n/2 wrong ones.
func TestQRReedDecodeLimits(t *testing.T) {
	const ecCount = 10
	data := []int{0x20, 0x5B, 0x0B, 0x78, 0xD1, 0x72, 0xDC, 0x4D, 0x43, 0x40, 0xEC, 0x11, 0xEC, 0x11, 0xEC, 0x11}
	block := append([]int{}, data...)
	block = append(block, make([]int, ecCount)...)

	// Give the block correct check codewords by trusting the generator's own.
	check := qrCalculateEC(qrBytesOf(data), ecCount)
	for i, b := range check {
		block[len(data)+i] = int(b)
	}

	t.Run("a clean block is returned untouched", func(t *testing.T) {
		got, ok := qrReedDecode(block, ecCount)
		if !ok {
			t.Fatal("refused a block with no errors")
		}
		for i := range data {
			if got[i] != data[i] {
				t.Fatalf("codeword %d changed from %#x to %#x", i, data[i], got[i])
			}
		}
	})

	t.Run("errors within its capacity are corrected", func(t *testing.T) {
		for _, count := range []int{1, 3, ecCount / 2} {
			damaged := append([]int{}, block...)
			for i := range count {
				damaged[i*2] ^= 0xFF
			}
			got, ok := qrReedDecode(damaged, ecCount)
			if !ok {
				t.Fatalf("refused a block with %d wrong codewords", count)
			}
			for i := range data {
				if got[i] != data[i] {
					t.Fatalf("with %d wrong, codeword %d came back %#x, want %#x",
						count, i, got[i], data[i])
				}
			}
		}
	})

	t.Run("errors past its capacity are refused", func(t *testing.T) {
		damaged := append([]int{}, block...)
		for i := range len(damaged) {
			damaged[i] ^= 0xA5
		}
		if _, ok := qrReedDecode(damaged, ecCount); ok {
			t.Error("claimed to correct a wholly corrupted block")
		}
	})
}

// qrBytesOf narrows codewords for the generator's error correction, which works
// in bytes.
func qrBytesOf(codewords []int) []byte {
	out := make([]byte, len(codewords))
	for i, c := range codewords {
		out[i] = byte(c)
	}
	return out
}

// TestQRDecodeDataTruncation covers each reader running out of bits before it
// has even read how many characters to expect.
func TestQRDecodeDataTruncation(t *testing.T) {
	for _, mode := range []struct {
		name string
		bits int
	}{
		{"numeric", qrTestNumeric},
		{"alphanumeric", qrTestAlpha},
		{"byte", qrTestByte},
		{"kanji", qrTestKanji},
		{"a character set marker", qrTestECI},
	} {
		t.Run(mode.name+" with no room for its header", func(t *testing.T) {
			if got, ok := qrDecodeData(qrBits([2]int{mode.bits, 4}), 1); ok {
				t.Errorf("read %q from a segment with nothing after its mode", got)
			}
		})
	}

	t.Run("a segment ending exactly at the last codeword", func(t *testing.T) {
		// Four bits of mode, ten of length and four of digit fill two and a
		// quarter codewords, so the padding runs out with nothing left over.
		got, ok := qrDecodeData(qrBits([2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}), 1)
		if !ok || got != "7" {
			t.Errorf("read %q, %v; want \"7\"", got, ok)
		}
	})
}

// TestQRReadFieldsWithDamage covers the search for the nearest version and
// format when the recorded fields do not match exactly.
func TestQRReadFieldsWithDamage(t *testing.T) {
	t.Run("a format field a few bits wrong", func(t *testing.T) {
		matrix := qrMatrixFromEncoder(t, "Hello world!", "M")
		want, ok := qrReadFormat(matrix)
		if !ok {
			t.Fatal("could not read an undamaged format field")
		}

		// Flip two bits of each copy, which is within what the field withstands.
		for _, at := range [][2]int{{0, 8}, {1, 8}} {
			matrix.set(at[0], at[1], !matrix.get(at[0], at[1]))
		}
		dimension := matrix.height
		for _, at := range [][2]int{{8, dimension - 1}, {8, dimension - 2}} {
			matrix.set(at[0], at[1], !matrix.get(at[0], at[1]))
		}

		got, ok := qrReadFormat(matrix)
		if !ok {
			t.Fatal("refused a format field two bits wrong in each copy")
		}
		if got != want {
			t.Errorf("read level %d mask %d, want level %d mask %d",
				got.level, got.mask, want.level, want.mask)
		}
	})

	t.Run("a version field a few bits wrong", func(t *testing.T) {
		// Version seven is the smallest carrying the field at all.
		matrix := qrMatrixFromEncoder(t, strings.Repeat("A", 400), "M")
		want, ok := qrReadVersion(matrix)
		if !ok {
			t.Fatal("could not read an undamaged version field")
		}
		if want.number < 7 {
			t.Fatalf("version %d carries no version field", want.number)
		}

		dimension := matrix.height
		for _, at := range [][2]int{{dimension - 9, 0}, {dimension - 10, 0}} {
			matrix.set(at[0], at[1], !matrix.get(at[0], at[1]))
		}
		got, ok := qrReadVersion(matrix)
		if !ok || got.number != want.number {
			t.Errorf("read version %d, %v; want %d", got.number, ok, want.number)
		}
	})

	t.Run("a matrix of no valid size", func(t *testing.T) {
		if _, ok := qrReadVersion(newQRBitMatrix(9, 9)); ok {
			t.Error("accepted a matrix smaller than the smallest version")
		}
	})
}

// TestQRDecodeMatrixMirrored covers the second attempt the reader makes with
// the matrix reflected, which reads a code photographed through its backing.
func TestQRDecodeMatrixMirrored(t *testing.T) {
	const text = "Hello world!"
	matrix := qrMatrixFromEncoder(t, text, "M")

	mirrored := newQRBitMatrix(matrix.width, matrix.height)
	for y := range matrix.height {
		for x := range matrix.width {
			mirrored.set(x, y, matrix.get(y, x))
		}
	}

	got, ok := qrDecodeMatrixOrMirror(mirrored)
	if !ok {
		t.Fatal("could not read a mirrored code")
	}
	if got != text {
		t.Errorf("read %q, want %q", got, text)
	}
}

// TestQRDecodeMatrixRejections covers the matrices the reader gives up on.
func TestQRDecodeMatrixRejections(t *testing.T) {
	t.Run("a matrix of noise", func(t *testing.T) {
		matrix := newQRBitMatrix(21, 21)
		for i := range matrix.data {
			matrix.data[i] = i%3 == 0
		}
		if got, ok := qrDecodeMatrixOrMirror(matrix); ok {
			t.Errorf("read %q from noise", got)
		}
	})

	t.Run("a matrix holding too few codewords", func(t *testing.T) {
		version := qrReaderVersions[0]
		if _, ok := qrDataBlocks([]int{1, 2, 3}, version, 0); ok {
			t.Error("accepted fewer codewords than the version holds")
		}
	})
}

// TestQRDecodeDataSegmentsCutShort covers each reader running out of bits part
// way through, after it has read how many characters to expect.
func TestQRDecodeDataSegmentsCutShort(t *testing.T) {
	for _, tc := range []struct {
		name      string
		codewords []int
	}{
		{"numeric", qrBits([2]int{qrTestNumeric, 4}, [2]int{6, 10}, [2]int{1, 6})},
		{"numeric leaving a pair", qrBits([2]int{qrTestNumeric, 4}, [2]int{2, 10}, [2]int{1, 1})},
		{"numeric leaving one digit", qrBits([2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{1, 1})},
		{"alphanumeric", qrBits([2]int{qrTestAlpha, 4}, [2]int{4, 9}, [2]int{1, 6})},
		{"alphanumeric leaving one", qrBits([2]int{qrTestAlpha, 4}, [2]int{1, 9}, [2]int{1, 2})},
		{"kanji", qrBits([2]int{qrTestKanji, 4}, [2]int{2, 8}, [2]int{1, 6})},
		{"a character set marker", qrBits([2]int{qrTestECI, 4}, [2]int{0, 1})},
	} {
		t.Run(tc.name+" cut short", func(t *testing.T) {
			if got, ok := qrDecodeData(tc.codewords, 1); ok {
				t.Errorf("read %q from a segment cut short", got)
			}
		})
	}
}

// TestQRDecodeDataPadding covers what the reader does with the bits after the
// last segment when there is no terminator to end it.
func TestQRDecodeDataPadding(t *testing.T) {
	t.Run("no bits left at all", func(t *testing.T) {
		// Four bits of mode, ten of length, four of digit and six of padding
		// fill three codewords exactly.
		got, ok := qrDecodeData(qrBits(
			[2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{0, 6},
		), 1)
		if !ok || got != "7" {
			t.Errorf("read %q, %v; want \"7\"", got, ok)
		}
	})

	t.Run("too few bits left to be a segment", func(t *testing.T) {
		got, ok := qrDecodeData(qrBits(
			[2]int{qrTestNumeric, 4}, [2]int{1, 10}, [2]int{7, 4}, [2]int{0, 3},
		), 1)
		if !ok || got != "7" {
			t.Errorf("read %q, %v; want \"7\"", got, ok)
		}
	})
}

// TestQRReadVersionBothCopiesDamaged covers the search for the nearest version
// when neither recorded copy matches exactly.
func TestQRReadVersionBothCopiesDamaged(t *testing.T) {
	matrix := qrMatrixFromEncoder(t, strings.Repeat("A", 400), "M")
	want, ok := qrReadVersion(matrix)
	if !ok {
		t.Fatal("could not read an undamaged version field")
	}

	dimension := matrix.height
	for _, at := range [][2]int{{dimension - 9, 0}, {0, dimension - 9}} {
		matrix.set(at[0], at[1], !matrix.get(at[0], at[1]))
	}
	got, ok := qrReadVersion(matrix)
	if !ok || got.number != want.number {
		t.Errorf("read version %d, %v; want %d", got.number, ok, want.number)
	}
}

// TestQRGeometry covers the arrangements of finder patterns and the projections
// they define that an ordinary photograph does not produce.
func TestQRGeometry(t *testing.T) {
	t.Run("the corner is whichever pattern the other two are furthest from", func(t *testing.T) {
		// The second and third are furthest apart, so the first is the corner.
		a := qrCandidate{point: qrPoint{0, 0}}
		b := qrCandidate{point: qrPoint{10, 0}}
		c := qrCandidate{point: qrPoint{0, 10}}
		topRight, topLeft, bottomLeft := qrReorderFinders(a, b, c)
		if topLeft != (qrPoint{0, 0}) {
			t.Errorf("the corner came out at %v, want the origin", topLeft)
		}
		if topRight == bottomLeft {
			t.Error("the other two came out the same")
		}
	})

	t.Run("a parallelogram needs no perspective", func(t *testing.T) {
		// Opposite sides equal, so the projection is affine and its last column
		// carries no perspective terms.
		square := qrSquareToQuad(
			qrPoint{0, 0}, qrPoint{10, 0}, qrPoint{10, 10}, qrPoint{0, 10},
		)
		if square.a13 != 0 || square.a23 != 0 {
			t.Errorf("perspective terms are %v and %v, want zero", square.a13, square.a23)
		}
		if square.a33 != 1 {
			t.Errorf("the scale is %v, want one", square.a33)
		}
	})

	t.Run("patterns too close together give no module size", func(t *testing.T) {
		matrix := newQRBitMatrix(10, 10)
		if _, _, ok := qrComputeDimension(
			qrPoint{1, 1}, qrPoint{1, 1}, qrPoint{1, 1}, matrix,
		); ok {
			t.Error("accepted three patterns at the same point")
		}
		if _, _, ok := qrFindAlignment(matrix, nil,
			qrPoint{1, 1}, qrPoint{1, 1}, qrPoint{1, 1}); ok {
			t.Error("placed an alignment pattern for a code of no size")
		}
	})
}

// TestQRDecodeDataUnterminated covers a code whose last segment runs to the end
// of the codewords with no terminator, which the padding has to stand in for.
func TestQRDecodeDataUnterminated(t *testing.T) {
	t.Run("ending exactly on a codeword", func(t *testing.T) {
		// Four bits of mode, nine of length and eleven of a pair fill three
		// codewords with nothing over.
		got, ok := qrDecodeData(qrBits(
			[2]int{qrTestAlpha, 4}, [2]int{2, 9}, [2]int{10*45 + 20, 11},
		), 1)
		if !ok || got != "AK" {
			t.Errorf("read %q, %v; want \"AK\"", got, ok)
		}
	})

	t.Run("ending with too few bits for another mode", func(t *testing.T) {
		// Four bits of mode, ten of length and seven of a pair leave three.
		got, ok := qrDecodeData(qrBits(
			[2]int{qrTestNumeric, 4}, [2]int{2, 10}, [2]int{42, 7},
		), 1)
		if !ok || got != "42" {
			t.Errorf("read %q, %v; want \"42\"", got, ok)
		}
	})
}

// TestQRComputeDimensionRounding covers the correction applied when the module
// count comes out even, which no version is.
func TestQRComputeDimensionRounding(t *testing.T) {
	// A chequerboard of two-pixel squares gives a module size the count can be
	// derived from, and sweeping the spacing of the corners walks the count
	// through every remainder.
	const side = 120
	matrix := newQRBitMatrix(side, side)
	for y := range side {
		for x := range side {
			matrix.set(x, y, (x/2+y/2)%2 == 0)
		}
	}

	sized := 0
	for span := 10; span <= 80; span++ {
		corner := qrPoint{4, 4}
		across := qrPoint{4 + float64(span), 4}
		down := qrPoint{4, 4 + float64(span)}
		got, _, ok := qrComputeDimension(corner, across, down, matrix)
		if !ok {
			continue
		}
		sized++
		// The correction only moves an even count to an odd one; an odd count
		// it leaves alone, even where no version has it.
		if got%2 == 0 {
			t.Errorf("a span of %d gave %d modules, which is not an odd count", span, got)
		}
	}
	if sized == 0 {
		t.Fatal("no spacing gave a module size at all")
	}
}

// TestQRDecodeMatrixTooSmall covers a matrix smaller than any version.
func TestQRDecodeMatrixTooSmall(t *testing.T) {
	if got, ok := qrDecodeMatrixOrMirror(newQRBitMatrix(9, 9)); ok {
		t.Errorf("read %q from a matrix of no valid size", got)
	}
}

// TestQRReadBytesInvalidUTF8 covers a byte segment whose contents are not valid
// text, which the reader drops rather than refusing the whole code.
func TestQRReadBytesInvalidUTF8(t *testing.T) {
	// A lone continuation byte, which no character starts with.
	got, ok := qrDecodeData(qrBits(
		[2]int{qrTestByte, 4}, [2]int{1, 8}, [2]int{0x80, 8}, [2]int{qrTestEnd, 4},
	), 1)
	if !ok {
		t.Fatal("refused a code whose bytes are not text")
	}
	if got != "" {
		t.Errorf("read %q, want the segment dropped", got)
	}
}

// TestQRDecodeDataTrailingBits covers bits after the last segment that are not
// the padding they should be.
func TestQRDecodeDataTrailingBits(t *testing.T) {
	// Twenty-one bits of segment leave three, which hold a value rather than
	// the zeros a real code pads with.
	if got, ok := qrDecodeData(qrBits(
		[2]int{qrTestNumeric, 4}, [2]int{2, 10}, [2]int{42, 7}, [2]int{5, 3},
	), 1); ok {
		t.Errorf("read %q from a code whose padding carries a value", got)
	}
}

// TestQRSkipECIExhausted covers the character set marker running out of bits
// before its first flag.
func TestQRSkipECIExhausted(t *testing.T) {
	if err := qrSkipECI(&qrBitStream{}); err == nil {
		t.Error("read a character set marker from nothing")
	}
}

// TestQRLocateTooFewPatterns covers an image holding fewer than the three
// finder patterns a code needs.
func TestQRLocateTooFewPatterns(t *testing.T) {
	// One finder pattern alone: a seven-module square with a white border.
	matrix := newQRBitMatrix(40, 40)
	for y := range 7 {
		for x := range 7 {
			edge := x == 0 || x == 6 || y == 0 || y == 6
			centre := x >= 2 && x <= 4 && y >= 2 && y <= 4
			matrix.set(x+4, y+4, edge || centre)
		}
	}
	if locations := qrLocate(matrix); len(locations) != 0 {
		t.Errorf("placed a code from %d finder pattern, want none", len(locations))
	}
}

// TestQRDataBlocksShort covers a matrix yielding fewer codewords than its
// version lays out, which nothing can mend.
func TestQRDataBlocksShort(t *testing.T) {
	matrix := qrMatrixFromEncoder(t, "Hello world!", "M")

	// Shrink the matrix to a smaller version's size, so it still reads a
	// version and format but holds too few codewords for them.
	small := newQRBitMatrix(matrix.width, matrix.height)
	copy(small.data, matrix.data)
	if _, ok := qrDataBlocks([]int{1, 2, 3}, qrReaderVersions[9], 0); ok {
		t.Error("accepted three codewords for a version holding hundreds")
	}
}

// TestQRReedDecodeNeverWrong covers the correction over many corruptions: it
// must either recover the original block or give up, and never hand back
// something else. Corruption past its capacity drives the several ways the
// search can fail.
func TestQRReedDecodeNeverWrong(t *testing.T) {
	const ecCount = 20
	data := make([]int, 40)
	for i := range data {
		data[i] = (i*37 + 11) & 0xFF
	}
	block := append([]int{}, data...)
	for _, b := range qrCalculateEC(qrBytesOf(data), ecCount) {
		block = append(block, int(b))
	}

	corrected, refused := 0, 0
	seed := uint32(2463534242)
	next := func(n int) int {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return int(seed % uint32(n))
	}

	for round := range 400 {
		damaged := append([]int{}, block...)
		for range round%25 + 1 {
			damaged[next(len(damaged))] ^= 1 + next(255)
		}

		got, ok := qrReedDecode(damaged, ecCount)
		if !ok {
			refused++
			continue
		}
		corrected++
		for i := range data {
			if got[i] != data[i] {
				t.Fatalf("round %d: codeword %d came back %#x, want %#x", round, i, got[i], data[i])
			}
		}
	}
	if corrected == 0 || refused == 0 {
		t.Errorf("corrected %d and refused %d; wanted both to happen", corrected, refused)
	}
}

// qrWriteVersionField writes a version's identifying bits into both places a
// code records them.
func qrWriteVersionField(matrix *qrBitMatrix, infoBits int) {
	dimension := matrix.height
	for i := range 6 {
		for j := range 3 {
			bit := infoBits>>(i*3+j)&1 == 1
			matrix.set(i, dimension-11+j, bit)
			matrix.set(dimension-11+j, i, bit)
		}
	}
}

// TestQRDecodeMatrixVersionMismatch covers a matrix whose recorded version does
// not match its size, so the layout it names holds more codewords than the
// matrix can supply.
func TestQRDecodeMatrixVersionMismatch(t *testing.T) {
	matrix := qrMatrixFromEncoder(t, strings.Repeat("A", 400), "M")
	qrWriteVersionField(matrix, qrReaderVersions[39].infoBits)

	if got, ok := qrDecodeMatrix(matrix); ok {
		t.Errorf("read %q from a matrix naming a version it is too small for", got)
	}
}

// TestQREuclideanDegenerate covers the two ways the search for the error
// locator gives up: the remainder reaching zero, and a division that fails to
// reduce its degree. Neither arises from a real block, whose check codewords
// always number more than zero.
func TestQREuclideanDegenerate(t *testing.T) {
	field := newQRField(qrReedPrimitive, qrReedSize, 0)

	t.Run("a remainder already at zero", func(t *testing.T) {
		if _, _, ok := qrEuclidean(field, field.monomial(2, 1), field.zero, 0); ok {
			t.Error("claimed a locator from a zero remainder")
		}
	})

	t.Run("a division that does not reduce the degree", func(t *testing.T) {
		// Dividing by a constant leaves a remainder of degree zero, which is
		// not smaller than the divisor's.
		if _, _, ok := qrEuclidean(field, field.one, field.one, 0); ok {
			t.Error("claimed a locator from a division making no progress")
		}
	})
}

// TestQRReedDecodeShortBlock covers a corrupted block so short that an error is
// located past its start.
func TestQRReedDecodeShortBlock(t *testing.T) {
	// Two check codewords over one of data locates at most one error, and a
	// block this short puts most locations outside it.
	refused := 0
	for value := range 256 {
		if _, ok := qrReedDecode([]int{value, 0x11, 0x22}, 2); !ok {
			refused++
		}
	}
	if refused == 0 {
		t.Error("every corruption of a three codeword block was claimed correctable")
	}
}
