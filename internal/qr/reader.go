package qr

import (
	"math"
	"sort"
)

// The image half of the QR reader, ported from jsQR
// (CyberChef's node_modules/jsqr/dist/jsQR.js), which is what CyberChef reads
// codes with: threshold the image, find the three finder patterns and the
// alignment pattern, then sample the modules through the perspective transform
// those four points define.

// qrBitMatrix is a grid of black and white modules or pixels.
type qrBitMatrix struct {
	data          []bool
	width, height int
}

func newQRBitMatrix(width, height int) *qrBitMatrix {
	return &qrBitMatrix{data: make([]bool, width*height), width: width, height: height}
}

// get reads one cell, reporting white for anything outside the grid so callers
// may scan past the edges.
func (m *qrBitMatrix) get(x, y int) bool {
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		return false
	}
	return m.data[y*m.width+x]
}

// set writes one cell, ignoring anything outside the grid as the typed arrays
// it is ported from do.
func (m *qrBitMatrix) set(x, y int, v bool) {
	if x < 0 || x >= m.width || y < 0 || y >= m.height {
		return
	}
	m.data[y*m.width+x] = v
}

// qrClamped mirrors the byte array the reader holds its intermediate values in,
// which rounds every value it is given and clamps it to a byte. The threshold
// pass reads past the end of the image, which gives a value that is not a
// number; every write is within it.
type qrClamped struct {
	data  []byte
	width int
}

func newQRClamped(width, height int) *qrClamped {
	return &qrClamped{data: make([]byte, width*height), width: width}
}

func (m *qrClamped) get(x, y int) float64 {
	i := y*m.width + x
	if i < 0 || i >= len(m.data) {
		return math.NaN()
	}
	return float64(m.data[i])
}

func (m *qrClamped) set(x, y int, v float64) {
	m.data[y*m.width+x] = qrClampByte(v)
}

// qrClampByte rounds to the nearest byte, halves to even, with anything that is
// not a number becoming zero.
func qrClampByte(v float64) byte {
	switch {
	case math.IsNaN(v):
		return 0
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return byte(math.RoundToEven(v))
}

// The size of the squares the threshold is estimated over, and the contrast
// below which a square is taken to be all one colour.
const (
	qrRegionSize      = 8
	qrMinDynamicRange = 24
)

// qrBinarize thresholds the image, estimating the black point over each square
// of the image and smoothing it against the neighbouring squares.
func qrBinarize(pixels []byte, width, height int) *qrBitMatrix {
	grey := qrGreyscale(pixels, width, height)
	across := (width + qrRegionSize - 1) / qrRegionSize
	down := (height + qrRegionSize - 1) / qrRegionSize
	return qrThreshold(grey, qrBlackPoints(grey, across, down), width, height, across, down)
}

// qrGreyscale weights the colour channels the way the eye does.
func qrGreyscale(pixels []byte, width, height int) *qrClamped {
	grey := newQRClamped(width, height)
	for x := range width {
		for y := range height {
			i := (y*width + x) * 4
			grey.set(x, y, 0.2126*float64(pixels[i])+0.7152*float64(pixels[i+1])+0.0722*float64(pixels[i+2]))
		}
	}
	return grey
}

// qrBlackPoints estimates the point separating dark from light over each square
// of the image.
func qrBlackPoints(grey *qrClamped, across, down int) *qrClamped {
	blackPoints := newQRClamped(across, down)
	for vertical := range down {
		for horizontal := range across {
			sum, lowest, highest := 0.0, math.Inf(1), 0.0
			for y := range qrRegionSize {
				for x := range qrRegionSize {
					lum := grey.get(horizontal*qrRegionSize+x, vertical*qrRegionSize+y)
					sum += lum
					lowest = math.Min(lowest, lum)
					highest = math.Max(highest, lum)
				}
			}
			average := sum / (qrRegionSize * qrRegionSize)

			if highest-lowest <= qrMinDynamicRange {
				// Too little contrast to threshold on: take the square to be
				// background, unless its neighbours suggest otherwise.
				average = lowest / 2
				if vertical > 0 && horizontal > 0 {
					neighbours := (blackPoints.get(horizontal, vertical-1) +
						2*blackPoints.get(horizontal-1, vertical) +
						blackPoints.get(horizontal-1, vertical-1)) / 4
					if lowest < neighbours {
						average = neighbours
					}
				}
			}
			blackPoints.set(horizontal, vertical, average)
		}
	}
	return blackPoints
}

// qrThreshold marks every pixel dark or light, against the black point of its
// square smoothed over those around it.
func qrThreshold(grey, blackPoints *qrClamped, width, height, across, down int) *qrBitMatrix {
	binarized := newQRBitMatrix(width, height)
	for vertical := range down {
		for horizontal := range across {
			left := qrBetween(horizontal, 2, across-3)
			top := qrBetween(vertical, 2, down-3)
			sum := 0.0
			for x := -2; x <= 2; x++ {
				for y := -2; y <= 2; y++ {
					sum += blackPoints.get(left+x, top+y)
				}
			}
			threshold := sum / 25

			for x := range qrRegionSize {
				for y := range qrRegionSize {
					px, py := horizontal*qrRegionSize+x, vertical*qrRegionSize+y
					binarized.set(px, py, grey.get(px, py) <= threshold)
				}
			}
		}
	}
	return binarized
}

func qrBetween(value, low, high int) int {
	return min(max(value, low), high)
}

// qrPoint is a position in the image, which the finder patterns hold to a
// fraction of a pixel.
type qrPoint struct{ x, y float64 }

func qrDistance(a, b qrPoint) float64 {
	return math.Hypot(b.x-a.x, b.y-a.y)
}

// qrAbsInt is the magnitude of a whole number.
func qrAbsInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func qrSum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

// qrRun counts the lengths of the alternating black and white stretches through
// a point, along the line towards another, in both directions.
func qrRun(origin, end qrPoint, matrix *qrBitMatrix, length int) []float64 {
	rise, run := end.y-origin.y, end.x-origin.x
	half := (length + 1) / 2
	towards := qrRunTowards(origin, end, matrix, half)
	away := qrRunTowards(origin, qrPoint{origin.x - run, origin.y - rise}, matrix, half)

	// The stretch the point sits in is counted from both sides, so one pixel of
	// it would otherwise be counted twice.
	middle := towards[0] + away[0] - 1
	out := append([]float64{}, away[1:]...)
	out = append(out, middle)
	return append(out, towards[1:]...)
}

// qrRunTowards walks a line from the origin and records where the colour
// changes, returning the length of each stretch.
func qrRunTowards(origin, end qrPoint, matrix *qrBitMatrix, length int) []float64 {
	switches := []qrPoint{{math.Floor(origin.x), math.Floor(origin.y)}}
	steep := math.Abs(end.y-origin.y) > math.Abs(end.x-origin.x)

	fromX, fromY := int(math.Floor(origin.x)), int(math.Floor(origin.y))
	toX, toY := int(math.Floor(end.x)), int(math.Floor(end.y))
	if steep {
		fromX, fromY = fromY, fromX
		toX, toY = toY, toX
	}

	dx, dy := qrAbsInt(toX-fromX), qrAbsInt(toY-fromY)
	err := int(math.Floor(float64(-dx) / 2))
	xStep, yStep := 1, 1
	if fromX >= toX {
		xStep = -1
	}
	if fromY >= toY {
		yStep = -1
	}

	current := true
	for x, y := fromX, fromY; x != toX+xStep; x += xStep {
		realX, realY := x, y
		if steep {
			realX, realY = y, x
		}
		if matrix.get(realX, realY) != current {
			current = !current
			switches = append(switches, qrPoint{float64(realX), float64(realY)})
			if len(switches) == length+1 {
				break
			}
		}
		err += dy
		if err > 0 {
			if y == toY {
				break
			}
			y += yStep
			err -= dx
		}
	}

	distances := make([]float64, length)
	for i := range length {
		if i+1 < len(switches) {
			distances[i] = qrDistance(switches[i], switches[i+1])
		}
	}
	return distances
}

// qrScoreRun measures how far a run of stretches falls from the ratio a pattern
// should show, and how large its modules are.
func qrScoreRun(sequence, ratios []float64) (averageSize, err float64) {
	averageSize = qrSum(sequence) / qrSum(ratios)
	for i, ratio := range ratios {
		difference := sequence[i] - ratio*averageSize
		err += difference * difference
	}
	return averageSize, err
}

// qrScorePattern scores a point against a pattern's ratio, measured across the
// horizontal, the vertical and both diagonals.
func qrScorePattern(point qrPoint, ratios []float64, matrix *qrBitMatrix) float64 {
	horizontal := qrRun(point, qrPoint{-1, point.y}, matrix, len(ratios))
	vertical := qrRun(point, qrPoint{point.x, -1}, matrix, len(ratios))
	upLeft := qrRun(point, qrPoint{
		math.Max(0, point.x-point.y) - 1,
		math.Max(0, point.y-point.x) - 1,
	}, matrix, len(ratios))
	downLeft := qrRun(point, qrPoint{
		math.Min(float64(matrix.width), point.x+point.y) + 1,
		math.Min(float64(matrix.height), point.y+point.x) + 1,
	}, matrix, len(ratios))

	hSize, hErr := qrScoreRun(horizontal, ratios)
	vSize, vErr := qrScoreRun(vertical, ratios)
	dSize, dErr := qrScoreRun(upLeft, ratios)
	uSize, uErr := qrScoreRun(downLeft, ratios)

	ratioError := math.Sqrt(hErr*hErr + vErr*vErr + dErr*dErr + uErr*uErr)
	average := (hSize + vSize + dSize + uSize) / 4
	spread := func(size float64) float64 { return (size - average) * (size - average) }
	sizeError := (spread(hSize) + spread(vSize) + spread(dSize) + spread(uSize)) / average
	return ratioError + sizeError
}

// qrRecenter moves a point to the middle of the black area it sits in, which
// suits an image whose apparent skew is only compression noise.
func qrRecenter(matrix *qrBitMatrix, p qrPoint) qrPoint {
	leftX := int(math.Round(p.x))
	for matrix.get(leftX, int(math.Round(p.y))) {
		leftX--
	}
	rightX := int(math.Round(p.x))
	for matrix.get(rightX, int(math.Round(p.y))) {
		rightX++
	}
	x := float64(leftX+rightX) / 2

	topY := int(math.Round(p.y))
	for matrix.get(int(math.Round(x)), topY) {
		topY--
	}
	bottomY := int(math.Round(p.y))
	for matrix.get(int(math.Round(x)), bottomY) {
		bottomY++
	}
	return qrPoint{x, float64(topY+bottomY) / 2}
}

// qrLine is one row of a candidate pattern, and qrQuad the stack of rows that
// makes up its centre square.
type (
	qrLine struct{ startX, endX, y int }
	qrQuad struct{ top, bottom qrLine }
)

// The bounds within which one row may be taken to continue the square above it,
// and how many of the best finder patterns are tried as a group.
const (
	qrMinQuadRatio    = 0.5
	qrMaxQuadRatio    = 1.5
	qrMaxFindersTried = 4
)

// qrLocation is one reading of where a code sits and how many modules it holds.
type qrLocation struct {
	topLeft, topRight, bottomLeft, alignment qrPoint
	dimension                                int
}

// qrLocate scans the image for the three finder patterns and returns the
// placements worth trying, best first.
func qrLocate(matrix *qrBitMatrix) []qrLocation {
	var finders, alignments []qrQuad
	var activeFinders, activeAlignments []qrQuad

	for y := 0; y <= matrix.height; y++ {
		activeFinders, activeAlignments = qrScanRow(matrix, y, activeFinders, activeAlignments)
		finders, activeFinders = qrCloseQuads(finders, activeFinders, y, 2)
		alignments, activeAlignments = qrCloseQuads(alignments, activeAlignments, y, 0)
	}
	alignments = append(alignments, activeAlignments...)

	groups := qrFinderGroups(finders, matrix)
	if len(groups) == 0 {
		return nil
	}

	topRight, topLeft, bottomLeft := qrReorderFinders(groups[0][0], groups[0][1], groups[0][2])

	var result []qrLocation
	if alignment, dimension, ok := qrFindAlignment(matrix, alignments, topRight, topLeft, bottomLeft); ok {
		result = append(result, qrLocation{topLeft, topRight, bottomLeft, alignment, dimension})
	}

	// The centres of the quads suit a genuinely skewed image; centring each
	// point in its black area suits one whose skew is an artefact.
	midTopRight := qrRecenter(matrix, topRight)
	midTopLeft := qrRecenter(matrix, topLeft)
	midBottomLeft := qrRecenter(matrix, bottomLeft)
	if alignment, dimension, ok := qrFindAlignment(matrix, alignments, midTopRight, midTopLeft, midBottomLeft); ok {
		result = append(result, qrLocation{midTopLeft, midTopRight, midBottomLeft, alignment, dimension})
	}
	return result
}

// qrScanRow walks one row of the image, adding the stretches that could be part
// of a finder or alignment pattern to the squares being built.
func qrScanRow(matrix *qrBitMatrix, y int, finders, alignments []qrQuad) ([]qrQuad, []qrQuad) {
	length := 0
	lastBit := false
	scans := []float64{0, 0, 0, 0, 0}

	for x := -1; x <= matrix.width; x++ {
		v := matrix.get(x, y)
		if v == lastBit {
			length++
			continue
		}
		scans = []float64{scans[1], scans[2], scans[3], scans[4], float64(length)}
		length = 1
		lastBit = v

		// A finder pattern shows stretches in the ratio 1:1:3:1:1, and is
		// bordered in white.
		module := qrSum(scans) / 7
		validFinder := math.Abs(scans[0]-module) < module &&
			math.Abs(scans[1]-module) < module &&
			math.Abs(scans[2]-3*module) < 3*module &&
			math.Abs(scans[3]-module) < module &&
			math.Abs(scans[4]-module) < module && !v

		// An alignment pattern shows 1:1:1, and is bordered in black.
		alignModule := qrSum(scans[2:]) / 3
		validAlignment := math.Abs(scans[2]-alignModule) < alignModule &&
			math.Abs(scans[3]-alignModule) < alignModule &&
			math.Abs(scans[4]-alignModule) < alignModule && v

		if validFinder {
			endX := x - int(scans[3]) - int(scans[4])
			finders = qrExtendQuads(finders, qrLine{endX - int(scans[2]), endX, y}, scans[2])
		}
		if validAlignment {
			endX := x - int(scans[4])
			alignments = qrExtendQuads(alignments, qrLine{endX - int(scans[3]), endX, y}, scans[2])
		}
	}
	return finders, alignments
}

// qrExtendQuads adds a row to the square it continues, or starts a new one.
func qrExtendQuads(active []qrQuad, line qrLine, width float64) []qrQuad {
	for i, q := range active {
		overlaps := (line.startX >= q.bottom.startX && line.startX <= q.bottom.endX) ||
			(line.endX >= q.bottom.startX && line.startX <= q.bottom.endX)
		spans := line.startX <= q.bottom.startX && line.endX >= q.bottom.endX
		if spans {
			ratio := width / float64(q.bottom.endX-q.bottom.startX)
			spans = ratio < qrMaxQuadRatio && ratio > qrMinQuadRatio
		}
		if overlaps || spans {
			active[i].bottom = line
			return active
		}
	}
	return append(active, qrQuad{top: line, bottom: line})
}

// qrCloseQuads moves the squares that ended on the previous row into the
// finished list, keeping those tall enough to hold a pattern.
func qrCloseQuads(done, active []qrQuad, y, minHeight int) (finished, stillActive []qrQuad) {
	for _, q := range active {
		switch {
		case q.bottom.y == y:
			stillActive = append(stillActive, q)
		case q.bottom.y-q.top.y >= minHeight:
			done = append(done, q)
		}
	}
	return done, stillActive
}

// qrCandidate is one scored pattern found in the image.
type qrCandidate struct {
	point qrPoint
	score float64
	size  float64
}

// qrFinderGroups scores every finder pattern found and returns the groups of
// three that best agree in size, best first.
func qrFinderGroups(quads []qrQuad, matrix *qrBitMatrix) [][3]qrCandidate {
	var found []qrCandidate
	for _, q := range quads {
		x := float64(q.top.startX+q.top.endX+q.bottom.startX+q.bottom.endX) / 4
		y := float64(q.top.y+q.bottom.y+1) / 2
		if !matrix.get(int(math.Round(x)), int(math.Round(y))) {
			continue
		}
		lengths := []float64{
			float64(q.top.endX - q.top.startX),
			float64(q.bottom.endX - q.bottom.startX),
			float64(q.bottom.y - q.top.y + 1),
		}
		centre := qrPoint{math.Round(x), math.Round(y)}
		found = append(found, qrCandidate{
			point: qrPoint{x, y},
			score: qrScorePattern(centre, []float64{1, 1, 3, 1, 1}, matrix),
			size:  qrSum(lengths) / float64(len(lengths)),
		})
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].score < found[j].score })

	type group struct {
		points [3]qrCandidate
		score  float64
	}
	var groups []group
	for i, point := range found {
		if i > qrMaxFindersTried {
			break
		}
		others := make([]qrCandidate, 0, len(found))
		for ii, p := range found {
			if i == ii {
				continue
			}
			// A pattern of a different size than the one under test is a worse
			// partner for it.
			difference := p.size - point.size
			p.score += difference * difference / point.size
			others = append(others, p)
		}
		sort.SliceStable(others, func(a, b int) bool { return others[a].score < others[b].score })
		if len(others) < 2 {
			continue
		}
		groups = append(groups, group{
			points: [3]qrCandidate{point, others[0], others[1]},
			score:  point.score + others[0].score + others[1].score,
		})
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].score < groups[j].score })

	out := make([][3]qrCandidate, len(groups))
	for i, g := range groups {
		out[i] = g.points
	}
	return out
}

// qrReorderFinders works out which of the three patterns is which, using the
// one furthest from the others as the corner and the cross product to tell the
// remaining two apart.
func qrReorderFinders(a, b, c qrCandidate) (topRight, topLeft, bottomLeft qrPoint) {
	ab := qrDistance(a.point, b.point)
	bc := qrDistance(b.point, c.point)
	ac := qrDistance(a.point, c.point)

	switch {
	case bc >= ab && bc >= ac:
		bottomLeft, topLeft, topRight = b.point, a.point, c.point
	case ac >= bc && ac >= ab:
		bottomLeft, topLeft, topRight = a.point, b.point, c.point
	default:
		bottomLeft, topLeft, topRight = a.point, c.point, b.point
	}

	if (topRight.x-topLeft.x)*(bottomLeft.y-topLeft.y)-
		(topRight.y-topLeft.y)*(bottomLeft.x-topLeft.x) < 0 {
		bottomLeft, topRight = topRight, bottomLeft
	}
	return topRight, topLeft, bottomLeft
}

// qrComputeDimension works out how many modules the code holds from the spacing
// of its finder patterns.
func qrComputeDimension(topLeft, topRight, bottomLeft qrPoint, matrix *qrBitMatrix) (dimension int, moduleSize float64, ok bool) {
	moduleSize = (qrSum(qrRun(topLeft, bottomLeft, matrix, 5))/7 +
		qrSum(qrRun(topLeft, topRight, matrix, 5))/7 +
		qrSum(qrRun(bottomLeft, topLeft, matrix, 5))/7 +
		qrSum(qrRun(topRight, topLeft, matrix, 5))/7) / 4
	if moduleSize < 1 {
		return 0, 0, false
	}

	across := math.Round(qrDistance(topLeft, topRight) / moduleSize)
	down := math.Round(qrDistance(topLeft, bottomLeft) / moduleSize)
	dimension = int(math.Floor((across+down)/2)) + 7

	// The count must leave a remainder of one when divided by four.
	switch dimension % 4 {
	case 0:
		dimension++
	case 2:
		dimension--
	}
	return dimension, moduleSize, true
}

// The spacing below which a code is of the smallest version and carries no
// alignment pattern.
const qrSmallestWithAlignment = 15

// qrFindAlignment locates the alignment pattern, falling back to where it ought
// to be when there is none to find.
func qrFindAlignment(matrix *qrBitMatrix, quads []qrQuad, topRight, topLeft, bottomLeft qrPoint) (qrPoint, int, bool) {
	dimension, moduleSize, ok := qrComputeDimension(topLeft, topRight, bottomLeft, matrix)
	if !ok {
		return qrPoint{}, 0, false
	}

	bottomRight := qrPoint{
		topRight.x - topLeft.x + bottomLeft.x,
		topRight.y - topLeft.y + bottomLeft.y,
	}
	spacing := (qrDistance(topLeft, bottomLeft) + qrDistance(topLeft, topRight)) / 2 / moduleSize
	correction := 1 - 3/spacing
	expected := qrPoint{
		topLeft.x + correction*(bottomRight.x-topLeft.x),
		topLeft.y + correction*(bottomRight.y-topLeft.y),
	}

	var candidates []qrCandidate
	for _, q := range quads {
		x := float64(q.top.startX+q.top.endX+q.bottom.startX+q.bottom.endX) / 4
		y := float64(q.top.y+q.bottom.y+1) / 2
		if !matrix.get(int(math.Floor(x)), int(math.Floor(y))) {
			continue
		}
		centre := qrPoint{math.Floor(x), math.Floor(y)}
		candidates = append(candidates, qrCandidate{
			point: qrPoint{x, y},
			score: qrScorePattern(centre, []float64{1, 1, 1}, matrix) + qrDistance(qrPoint{x, y}, expected),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score < candidates[j].score })

	if spacing >= qrSmallestWithAlignment && len(candidates) > 0 {
		return candidates[0].point, dimension, true
	}
	return expected, dimension, true
}

// qrTransform is the projection between the square the modules sit on and the
// quadrilateral they appear as in the image.
type qrTransform struct {
	a11, a12, a13 float64
	a21, a22, a23 float64
	a31, a32, a33 float64
}

// qrSquareToQuad builds the projection from the unit square onto four points.
func qrSquareToQuad(p1, p2, p3, p4 qrPoint) qrTransform {
	dx3 := p1.x - p2.x + p3.x - p4.x
	dy3 := p1.y - p2.y + p3.y - p4.y
	if dx3 == 0 && dy3 == 0 {
		// The quadrilateral is a parallelogram, so the projection is affine.
		return qrTransform{
			a11: p2.x - p1.x, a12: p2.y - p1.y, a13: 0,
			a21: p3.x - p2.x, a22: p3.y - p2.y, a23: 0,
			a31: p1.x, a32: p1.y, a33: 1,
		}
	}
	dx1, dx2 := p2.x-p3.x, p4.x-p3.x
	dy1, dy2 := p2.y-p3.y, p4.y-p3.y
	denominator := dx1*dy2 - dx2*dy1
	a13 := (dx3*dy2 - dx2*dy3) / denominator
	a23 := (dx1*dy3 - dx3*dy1) / denominator
	return qrTransform{
		a11: p2.x - p1.x + a13*p2.x, a12: p2.y - p1.y + a13*p2.y, a13: a13,
		a21: p4.x - p1.x + a23*p4.x, a22: p4.y - p1.y + a23*p4.y, a23: a23,
		a31: p1.x, a32: p1.y, a33: 1,
	}
}

// qrQuadToSquare inverts that projection, for which the adjoint serves.
func qrQuadToSquare(p1, p2, p3, p4 qrPoint) qrTransform {
	s := qrSquareToQuad(p1, p2, p3, p4)
	return qrTransform{
		a11: s.a22*s.a33 - s.a23*s.a32, a12: s.a13*s.a32 - s.a12*s.a33, a13: s.a12*s.a23 - s.a13*s.a22,
		a21: s.a23*s.a31 - s.a21*s.a33, a22: s.a11*s.a33 - s.a13*s.a31, a23: s.a13*s.a21 - s.a11*s.a23,
		a31: s.a21*s.a32 - s.a22*s.a31, a32: s.a12*s.a31 - s.a11*s.a32, a33: s.a11*s.a22 - s.a12*s.a21,
	}
}

// qrTimes composes two projections.
func qrTimes(a, b qrTransform) qrTransform {
	return qrTransform{
		a11: a.a11*b.a11 + a.a21*b.a12 + a.a31*b.a13,
		a12: a.a12*b.a11 + a.a22*b.a12 + a.a32*b.a13,
		a13: a.a13*b.a11 + a.a23*b.a12 + a.a33*b.a13,
		a21: a.a11*b.a21 + a.a21*b.a22 + a.a31*b.a23,
		a22: a.a12*b.a21 + a.a22*b.a22 + a.a32*b.a23,
		a23: a.a13*b.a21 + a.a23*b.a22 + a.a33*b.a23,
		a31: a.a11*b.a31 + a.a21*b.a32 + a.a31*b.a33,
		a32: a.a12*b.a31 + a.a22*b.a32 + a.a32*b.a33,
		a33: a.a13*b.a31 + a.a23*b.a32 + a.a33*b.a33,
	}
}

// The offsets of the sampling corners from the edges of the code, which sit at
// the centres of the finder and alignment patterns.
const (
	qrFinderCentre    = 3.5
	qrAlignmentCentre = 6.5
)

// qrExtract samples the modules out of the image through the projection the
// located corners define.
func qrExtract(image *qrBitMatrix, location qrLocation) *qrBitMatrix {
	side := float64(location.dimension)
	toSquare := qrQuadToSquare(
		qrPoint{qrFinderCentre, qrFinderCentre},
		qrPoint{side - qrFinderCentre, qrFinderCentre},
		qrPoint{side - qrAlignmentCentre, side - qrAlignmentCentre},
		qrPoint{qrFinderCentre, side - qrFinderCentre},
	)
	toQuad := qrSquareToQuad(location.topLeft, location.topRight, location.alignment, location.bottomLeft)
	transform := qrTimes(toQuad, toSquare)

	matrix := newQRBitMatrix(location.dimension, location.dimension)
	for y := range location.dimension {
		for x := range location.dimension {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			denominator := transform.a13*fx + transform.a23*fy + transform.a33
			sourceX := (transform.a11*fx + transform.a21*fy + transform.a31) / denominator
			sourceY := (transform.a12*fx + transform.a22*fy + transform.a32) / denominator
			matrix.set(x, y, image.get(int(math.Floor(sourceX)), int(math.Floor(sourceY))))
		}
	}
	return matrix
}
