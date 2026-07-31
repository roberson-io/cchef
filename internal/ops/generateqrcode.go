package ops

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(GenerateQRCode{})
}

// GenerateQRCode renders the input as a QR code. Ported from CyberChef's
// Generate QR Code, which wraps the qr-image package; the vector renderers come
// from that package's vector.js and are byte-exact.
type GenerateQRCode struct{}

// Meta returns the operation metadata.
func (GenerateQRCode) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Generate QR Code",
		Module: "Image",
		Description: "Generates a Quick Response (QR) code from the input text. " +
			"A QR code is a type of matrix barcode (or two-dimensional barcode) first " +
			"designed in 1994 for the automotive industry in Japan. A barcode is a " +
			"machine-readable optical label that contains information about the item " +
			"to which it is attached.",
		InputType:  core.TypeString,
		OutputType: core.TypeArrayBuffer,
	}
}

var (
	qrModuleSizeMin = float64(1)
	qrMarginMin     = float64(0)
)

// Args returns the argument definitions.
func (GenerateQRCode) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Image Format", Type: core.ArgOption, Value: []string{"PNG", "SVG", "EPS", "PDF"}},
		{Name: "Module size (px)", Type: core.ArgNumber, Integer: true, Value: float64(5), Min: &qrModuleSizeMin},
		{Name: "Margin (num modules)", Type: core.ArgNumber, Integer: true, Value: float64(4), Min: &qrMarginMin},
		{
			Name: "Error correction", Type: core.ArgOption,
			Value:        []string{"Low", "Medium", "Quartile", "High"},
			DefaultIndex: 1,
		},
	}
}

// Run generates the code.
func (GenerateQRCode) Run(in *core.Dish, args []any) (*core.Dish, error) {
	format := args[0].(string)
	moduleSize := int(args[1].(float64))
	margin := int(args[2].(float64))
	correction := args[3].(string)

	// The level is named by its initial, so "Quartile" selects Q.
	level := strings.ToUpper(correction[:1])

	matrix, err := qrMatrixFor(in.Bytes(), level)
	if err != nil {
		return nil, fmt.Errorf("error generating QR code: %w", err)
	}

	var out []byte
	switch strings.ToUpper(format) {
	case "PNG":
		out = qrRenderPNG(matrix, margin, moduleSize)
	case "SVG":
		out = []byte(qrRenderSVG(matrix, margin, moduleSize))
	case "EPS":
		out = []byte(qrRenderEPS(matrix, margin))
	case "PDF":
		out = []byte(qrRenderPDF(matrix, margin))
	default:
		return nil, errors.New("unsupported QR code format")
	}
	return core.NewDish(out, core.TypeArrayBuffer), nil
}

// The PNG container: its signature, and the header describing a greyscale
// image of one byte per pixel.
var (
	pngSignature = []byte{137, 80, 78, 71, 13, 10, 26, 10}
	pngHeader    = []byte{
		0, 0, 0, 13, 'I', 'H', 'D', 'R',
		0, 0, 0, 0, // the width, filled in below
		0, 0, 0, 0, // and the height
		8,       // eight bits a pixel
		0,       // greyscale, with no palette
		0, 0, 0, // the only compression, filtering and interlacing the format defines
	}
	pngEnd = []byte{0, 0, 0, 0, 'I', 'E', 'N', 'D', 174, 66, 96, 130}
)

// qrRenderPNG writes the code as a PNG, one byte a pixel.
func qrRenderPNG(matrix [][]byte, margin, size int) []byte {
	bitmap, extent := qrBitmap(matrix, margin, size)

	header := append([]byte{}, pngHeader...)
	binary.BigEndian.PutUint32(header[8:], uint32(extent))  // #nosec G115 -- the extent is bounded by the argument limits
	binary.BigEndian.PutUint32(header[12:], uint32(extent)) // #nosec G115 -- as above

	out := append([]byte{}, pngSignature...)
	out = append(out, header...)
	out = binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(header[4:]))
	out = pngChunk(out, "IDAT", zlibDeflate(bitmap))
	return append(out, pngEnd...)
}

// pngChunk appends one chunk with its length and checksum.
func pngChunk(out []byte, name string, data []byte) []byte {
	out = binary.BigEndian.AppendUint32(out, uint32(len(data))) // #nosec G115 -- a chunk is far smaller than the field
	body := append([]byte(name), data...)
	out = append(out, body...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(body))
}

// qrBitmap paints the matrix into rows of pixels, each row preceded by the byte
// naming the filter it uses, and returns the side of the square.
func qrBitmap(matrix [][]byte, margin, size int) ([]byte, int) {
	n := len(matrix)
	extent := (n + 2*margin) * size
	stride := extent + 1

	bitmap := bytes.Repeat([]byte{0xFF}, stride*extent)
	for row := range extent {
		bitmap[row*stride] = 0 // no filtering
	}

	for i := range n {
		for j := range n {
			if matrix[i][j] == 0 {
				continue
			}
			offset := ((margin+i)*stride+(margin+j))*size + 1
			for row := range size {
				start := offset + row*stride
				clear(bitmap[start : start+size])
			}
		}
	}
	return bitmap, extent
}

// qrPathOp is one step of an outline: a move to a point, or a horizontal or
// vertical run from wherever the previous step left off.
type qrPathOp struct {
	kind byte // 'M', 'h' or 'v'
	x    int  // the column for a move, the length otherwise
	y    int  // the row, for a move
}

// qrTrace walks the matrix and returns the outline of every dark region, each
// as a closed subpath. Regions are traced anticlockwise and the holes inside
// them clockwise, so filling with the even-odd rule leaves the holes open.
func qrTrace(matrix [][]byte) [][]qrPathOp {
	n := len(matrix)
	// The walk writes one cell beyond each edge, so the record of visited cells
	// runs from -1 to n and is offset by one.
	filled := make([][]bool, n+2)
	for i := range filled {
		filled[i] = make([]bool, n+2)
	}
	visit := func(row, col int) { filled[row+1][col+1] = true }
	visited := func(row, col int) bool { return filled[row+1][col+1] }

	isDark := func(row, col int) bool {
		if row < 0 || col < 0 || row >= n || col >= n {
			return false
		}
		return matrix[row][col] != 0
	}

	var path [][]qrPathOp
	for row := range n {
		for col := range n {
			if visited(row, col) {
				continue
			}
			visit(row, col)
			if isDark(row, col) {
				if !isDark(row-1, col) {
					path = append(path, qrPlot(visit, isDark, row, col, "right"))
				}
			} else if isDark(row, col-1) {
				path = append(path, qrPlot(visit, isDark, row, col, "down"))
			}
		}
	}
	return path
}

// qrTurn describes how the tracer behaves heading in one direction: which two
// cells it probes, what it emits when it turns, and where it goes next.
type qrTurn struct {
	first, second [2]int // the cells probed, as row and column offsets
	kind          byte   // the command a turn emits
	sign          int    // its sign, since a run may go either way
	step          [2]int // how far to move when the outline continues
	onSecondDark  string // where to turn when the second cell is dark
	onFirstLight  string // and where when the first is light
}

// qrTurns is that behaviour for each of the four directions. The outline keeps
// dark to one side, so each direction probes the two cells across the edge it
// is following.
var qrTurns = map[string]qrTurn{
	"right": {[2]int{0, 0}, [2]int{-1, 0}, 'h', 1, [2]int{0, 1}, "up", "down"},
	"left":  {[2]int{-1, -1}, [2]int{0, -1}, 'h', -1, [2]int{0, -1}, "down", "up"},
	"down":  {[2]int{0, -1}, [2]int{0, 0}, 'v', 1, [2]int{1, 0}, "right", "left"},
	"up":    {[2]int{-1, 0}, [2]int{-1, -1}, 'v', -1, [2]int{-1, 0}, "left", "right"},
}

// qrPlot follows one outline from its starting corner back round to itself,
// turning at every change of colour.
func qrPlot(visit func(row, col int), isDark func(row, col int) bool,
	row0, col0 int, dir string,
) []qrPathOp {
	visit(row0, col0)
	res := []qrPathOp{{kind: 'M', x: col0, y: row0}}

	row, col, length := row0, col0, 0
	for {
		turn := qrTurns[dir]
		visit(row+turn.first[0], col+turn.first[1])

		switch {
		case !isDark(row+turn.first[0], col+turn.first[1]):
			res = append(res, qrPathOp{kind: turn.kind, x: turn.sign * length})
			length, dir = 0, turn.onFirstLight

		default:
			visit(row+turn.second[0], col+turn.second[1])
			if isDark(row+turn.second[0], col+turn.second[1]) {
				res = append(res, qrPathOp{kind: turn.kind, x: turn.sign * length})
				length, dir = 0, turn.onSecondDark
				break
			}
			length++
			row, col = row+turn.step[0], col+turn.step[1]
		}

		if row == row0 && col == col0 {
			return res
		}
	}
}

// qrRenderSVG writes the code as SVG, whose module size scales the viewport
// rather than the path.
func qrRenderSVG(matrix [][]byte, margin, size int) string {
	extent := len(matrix) + 2*margin

	var out strings.Builder
	out.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" `)
	if size > 0 {
		side := strconv.Itoa(extent * size)
		out.WriteString(`width="` + side + `" height="` + side + `" `)
	}
	out.WriteString(`viewBox="0 0 ` + strconv.Itoa(extent) + ` ` + strconv.Itoa(extent) + `">`)
	out.WriteString(`<path d="`)

	for _, subpath := range qrTrace(matrix) {
		for _, item := range subpath {
			if item.kind == 'M' {
				out.WriteString("M" + strconv.Itoa(item.x+margin) + " " + strconv.Itoa(item.y+margin))
				continue
			}
			out.WriteString(string(item.kind) + strconv.Itoa(item.x))
		}
		out.WriteString("z")
	}

	out.WriteString(`"/></svg>`)
	return out.String()
}

// qrVectorScale is the module size the EPS and PDF renderers use; unlike SVG
// they ignore the size argument and scale the whole page instead.
const qrVectorScale = 9

// qrRenderEPS writes the code as encapsulated PostScript.
func qrRenderEPS(matrix [][]byte, margin int) string {
	n := len(matrix)
	extent := (n + 2*margin) * qrVectorScale

	var out strings.Builder
	out.WriteString(strings.Join([]string{
		"%!PS-Adobe-3.0 EPSF-3.0",
		"%%BoundingBox: 0 0 " + strconv.Itoa(extent) + " " + strconv.Itoa(extent),
		"/h { 0 rlineto } bind def",
		"/v { 0 exch neg rlineto } bind def",
		"/M { neg " + strconv.Itoa(n+margin) + " add moveto } bind def",
		"/z { closepath } bind def",
		strconv.Itoa(qrVectorScale) + " " + strconv.Itoa(qrVectorScale) + " scale",
		"",
	}, "\n"))

	for _, subpath := range qrTrace(matrix) {
		for _, item := range subpath {
			if item.kind == 'M' {
				out.WriteString(strconv.Itoa(item.x+margin) + " " + strconv.Itoa(item.y) + " M ")
				continue
			}
			out.WriteString(strconv.Itoa(item.x) + " " + string(item.kind) + " ")
		}
		out.WriteString("z\n")
	}

	out.WriteString("fill\n%%EOF\n")
	return out.String()
}

// qrRenderPDF writes the code as a one-page PDF, whose cross-reference table
// records the byte offset of every object.
func qrRenderPDF(matrix [][]byte, margin int) string {
	n := len(matrix)
	extent := (n + 2*margin) * qrVectorScale

	objects := []string{
		"%PDF-1.0\n\n",
		"1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n",
		"2 0 obj << /Type /Pages /Count 1 /Kids [ 3 0 R ] >> endobj\n",
		"3 0 obj << /Type /Page /Parent 2 0 R /Resources <<>> /Contents 4 0 R /MediaBox [ 0 0 " +
			strconv.Itoa(extent) + " " + strconv.Itoa(extent) + " ] >> endobj\n",
	}

	scale := strconv.Itoa(qrVectorScale)
	var subpaths []string
	for _, subpath := range qrTrace(matrix) {
		var res strings.Builder
		var x, y int
		for _, item := range subpath {
			switch item.kind {
			case 'M':
				x, y = item.x+margin, n-item.y+margin
				fmt.Fprintf(&res, "%d %d m ", x, y)
			case 'h':
				x += item.x
				fmt.Fprintf(&res, "%d %d l ", x, y)
			case 'v':
				y -= item.x
				fmt.Fprintf(&res, "%d %d l ", x, y)
			}
		}
		res.WriteString("h")
		subpaths = append(subpaths, res.String())
	}

	content := scale + " 0 0 " + scale + " 0 0 cm\n" + strings.Join(subpaths, "\n") + "\nf\n"
	objects = append(objects, "4 0 obj << /Length "+strconv.Itoa(len(content))+" >> stream\n"+
		content+"endstream\nendobj\n")

	var xref strings.Builder
	xref.WriteString("xref\n0 5\n0000000000 65535 f \n")
	offset := len(objects[0])
	for i := 1; i < 5; i++ {
		fmt.Fprintf(&xref, "%010d 00000 n \n", offset)
		offset += len(objects[i])
	}

	objects = append(objects, xref.String(),
		"trailer << /Root 1 0 R /Size 5 >>\n",
		"startxref\n"+strconv.Itoa(offset)+"\n%%EOF\n")

	return strings.Join(objects, "")
}
