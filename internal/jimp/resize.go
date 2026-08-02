package jimp

import (
	"errors"
	"image"
	"math"
)

// Resize/crop/blit primitives ported byte-for-byte from Jimp (@jimp/plugin-resize
// resize2.js, @jimp/plugin-crop, @jimp/plugin-blit) as used by CyberChef's
// Resize/Crop/Contain/Cover Image. Operating on straight-RGBA bitmaps, the
// results are pixel-identical to CyberChef for lossless formats.

// StrategyNames maps cchef's option labels to Jimp's ResizeStrategy keys.
var StrategyNames = map[string]string{
	"Nearest Neighbour": "nearestNeighbor",
	"Bilinear":          "bilinearInterpolation",
	"Bicubic":           "bicubicInterpolation",
	"Hermite":           "hermiteInterpolation",
	"Bezier":            "bezierInterpolation",
}

// resizeBitmap is a dense straight-RGBA bitmap (Jimp's image.bitmap).
type resizeBitmap struct {
	data          []byte
	width, height int
}

func bitmapOf(img *image.NRGBA) resizeBitmap {
	return resizeBitmap{data: img.Pix, width: img.Rect.Dx(), height: img.Rect.Dy()}
}

func nrgbaOf(b resizeBitmap) *image.NRGBA {
	return &image.NRGBA{Pix: b.data, Stride: 4 * b.width, Rect: image.Rect(0, 0, b.width, b.height)}
}

// Resize resizes img to wf×hf (rounded, min 1) using the named strategy.
func Resize(img *image.NRGBA, wf, hf float64, mode string) *image.NRGBA {
	w := int(math.Round(wf))
	if w == 0 {
		w = 1
	}
	h := int(math.Round(hf))
	if h == 0 {
		h = 1
	}
	src := bitmapOf(img)
	dst := resizeBitmap{data: make([]byte, w*h*4), width: w, height: h}
	if src.width == 0 || src.height == 0 {
		// Jimp reads past the end of an empty bitmap and yields a fully
		// transparent result; there is nothing to sample from.
		return nrgbaOf(dst)
	}
	switch mode {
	case "nearestNeighbor":
		resizeNearest(src, dst)
	case "bilinearInterpolation":
		resizeBilinear(src, dst)
	case "bicubicInterpolation":
		interpolate2D(src, dst, bicubicKernel)
	case "hermiteInterpolation":
		interpolate2D(src, dst, hermiteKernel)
	case "bezierInterpolation":
		interpolate2D(src, dst, bezierKernel)
	}
	return nrgbaOf(dst)
}

// Scale scales img by factor f.
func Scale(img *image.NRGBA, f float64, mode string) *image.NRGBA {
	return Resize(img, float64(img.Rect.Dx())*f, float64(img.Rect.Dy())*f, mode)
}

// ScaleToFit scales img to fit within wf×hf, preserving aspect ratio.
func ScaleToFit(img *image.NRGBA, wf, hf float64, mode string) *image.NRGBA {
	w, h := float64(img.Rect.Dx()), float64(img.Rect.Dy())
	var f float64
	if wf/hf > w/h {
		f = hf / h
	} else {
		f = wf / w
	}
	return Scale(img, f, mode)
}

func resizeNearest(src, dst resizeBitmap) {
	for i := 0; i < dst.height; i++ {
		for j := 0; j < dst.width; j++ {
			posDst := (i*dst.width + j) * 4
			iSrc := i * src.height / dst.height
			jSrc := j * src.width / dst.width
			posSrc := (iSrc*src.width + jSrc) * 4
			copy(dst.data[posDst:posDst+4], src.data[posSrc:posSrc+4])
		}
	}
}

func resizeBilinear(src, dst resizeBitmap) {
	wSrc, hSrc := src.width, src.height
	interp := func(k float64, kMin int, vMin byte, kMax int, vMax byte) byte {
		if kMin == kMax {
			return vMin
		}
		return byte(math.Round((k-float64(kMin))*float64(vMax) + (float64(kMax)-k)*float64(vMin)))
	}
	assign := func(posDst, offset int, x float64, xMin, xMax int, y float64, yMin, yMax int) {
		vMin := interp(x, xMin, src.data[(yMin*wSrc+xMin)*4+offset], xMax, src.data[(yMin*wSrc+xMax)*4+offset])
		if yMax == yMin {
			dst.data[posDst+offset] = vMin
			return
		}
		vMax := interp(x, xMin, src.data[(yMax*wSrc+xMin)*4+offset], xMax, src.data[(yMax*wSrc+xMax)*4+offset])
		dst.data[posDst+offset] = interp(y, yMin, vMin, yMax, vMax)
	}
	for i := 0; i < dst.height; i++ {
		for j := 0; j < dst.width; j++ {
			posDst := (i*dst.width + j) * 4
			x := float64(j*wSrc) / float64(dst.width)
			xMin := int(math.Floor(x))
			xMax := min(int(math.Ceil(x)), wSrc-1)
			y := float64(i*hSrc) / float64(dst.height)
			yMin := int(math.Floor(y))
			yMax := min(int(math.Ceil(y)), hSrc-1)
			assign(posDst, 0, x, xMin, xMax, y, yMin, yMax)
			assign(posDst, 1, x, xMin, xMax, y, yMin, yMax)
			assign(posDst, 2, x, xMin, xMax, y, yMin, yMax)
			assign(posDst, 3, x, xMin, xMax, y, yMin, yMax)
		}
	}
}

// resizeKernel is a 4-point interpolation kernel over samples x0..x3 at t∈[0,1).
type resizeKernel func(x0, x1, x2, x3, t float64) byte

// round rounds a float to the nearest integer.
func round(f float64) int { return int(math.Round(f)) }

// interpolate2D is Jimp's generic two-pass resampler for bicubic/hermite/bezier.
func interpolate2D(src, dst resizeBitmap, kernel resizeKernel) {
	wSrc, hSrc := src.width, src.height
	wM := max(1, wSrc/dst.width)
	wDst2 := dst.width * wM
	hM := max(1, hSrc/dst.height)
	hDst2 := dst.height * hM

	buf1 := interp2DPass(src.data, wSrc, hSrc, wDst2, hSrc, 4, wSrc, kernel, true)
	buf2 := interp2DPass(buf1, wDst2, hSrc, wDst2, hDst2, wDst2*4, hSrc, kernel, false)

	if wM*hM > 1 {
		interp2DDownsample(buf2, dst, wDst2, wM, hM)
	} else {
		copy(dst.data, buf2)
	}
}

// interp2DPass runs one interpolation pass (horizontal when horizontal is true,
// vertical otherwise). stride is the per-neighbour byte step and dimN the source
// count along the pass axis.
func interp2DPass(srcData []byte, srcW, srcH, outW, outH, nStride, dimN int, kernel resizeKernel, horizontal bool) []byte {
	out := make([]byte, outW*outH*4)
	for i := range outH {
		for j := range outW {
			var pos float64
			var basePos, outPos int
			if horizontal {
				pos = float64(j*(dimN-1)) / float64(outW)
				p := int(math.Floor(pos))
				basePos = (i*srcW + p) * 4
				outPos = (i*outW + j) * 4
				interpKPix(srcData, out, basePos, outPos, nStride, p, dimN, pos-float64(p), kernel)
			} else {
				pos = float64(i*(dimN-1)) / float64(outH)
				p := int(math.Floor(pos))
				basePos = (p*srcW + j) * 4
				outPos = (i*outW + j) * 4
				interpKPix(srcData, out, basePos, outPos, nStride, p, dimN, pos-float64(p), kernel)
			}
		}
	}
	return out
}

// interpKPix interpolates the four channels of one output pixel, extrapolating
// beyond the edges as Jimp does (2*edge - inner).
func interpKPix(src, out []byte, basePos, outPos, stride, p, dimN int, t float64, kernel resizeKernel) {
	for k := range 4 {
		kPos := basePos + k
		var x0, x3 float64
		if p > 0 {
			x0 = float64(src[kPos-stride])
		} else {
			x0 = 2*float64(src[kPos]) - float64(src[kPos+stride])
		}
		x1 := float64(src[kPos])
		x2 := float64(src[kPos+stride])
		if p < dimN-2 {
			x3 = float64(src[kPos+2*stride])
		} else {
			x3 = 2*float64(src[kPos+stride]) - float64(src[kPos])
		}
		out[outPos+k] = kernel(x0, x1, x2, x3, t)
	}
}

// interp2DDownsample averages the oversampled buffer down to dst.
func interp2DDownsample(buf2 []byte, dst resizeBitmap, wDst2, wM, hM int) {
	m := wM * hM
	for i := 0; i < dst.height; i++ {
		for j := 0; j < dst.width; j++ {
			var r, g, b, a, realColors int
			for y := range hM {
				for x := range wM {
					xyPos := ((i*hM+y)*wDst2 + (j*wM + x)) * 4
					pixelAlpha := int(buf2[xyPos+3])
					if pixelAlpha != 0 {
						r += int(buf2[xyPos])
						g += int(buf2[xyPos+1])
						b += int(buf2[xyPos+2])
						realColors++
					}
					a += pixelAlpha
				}
			}
			pos := (i*dst.width + j) * 4
			if realColors != 0 {
				dst.data[pos] = byte(math.Round(float64(r) / float64(realColors)))
				dst.data[pos+1] = byte(math.Round(float64(g) / float64(realColors)))
				dst.data[pos+2] = byte(math.Round(float64(b) / float64(realColors)))
			}
			dst.data[pos+3] = byte(math.Round(float64(a) / float64(m)))
		}
	}
}

func bicubicKernel(x0, x1, x2, x3, t float64) byte {
	a0 := x3 - x2 - x0 + x1
	a1 := x0 - x1 - a0
	a2 := x2 - x0
	a3 := x1
	return byte(math.Max(0, math.Min(255, a0*t*t*t+a1*t*t+a2*t+a3)))
}

func hermiteKernel(x0, x1, x2, x3, t float64) byte {
	c0 := x1
	c1 := 0.5 * (x2 - x0)
	c2 := x0 - 2.5*x1 + 2*x2 - 0.5*x3
	c3 := 0.5*(x3-x0) + 1.5*(x1-x2)
	return byte(math.Max(0, math.Min(255, math.Round(((c3*t+c2)*t+c1)*t+c0))))
}

func bezierKernel(x0, x1, x2, x3, t float64) byte {
	cp1 := x1 + (x2-x0)/4
	cp2 := x2 - (x3-x1)/4
	nt := 1 - t
	c0 := x1 * nt * nt * nt
	c1 := 3 * cp1 * nt * nt * t
	c2 := 3 * cp2 * nt * t * t
	c3 := x2 * t * t * t
	return byte(math.Max(0, math.Min(255, math.Round(c0+c1+c2+c3))))
}

// CropExact extracts the w×h region at (x,y). Returns an error if the region
// falls outside the source (Jimp throws a RangeError, which CyberChef surfaces).
func CropExact(img *image.NRGBA, x, y, w, h int) (*image.NRGBA, error) {
	W := img.Rect.Dx()
	src := img.Pix
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	offset := 0
	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			idx := (W*cy + cx) * 4
			if idx < 0 || idx+4 > len(src) {
				return nil, errors.New("crop region is outside the image")
			}
			copy(out.Pix[offset:offset+4], src[idx:idx+4])
			offset += 4
		}
	}
	return out, nil
}

// Blit composites src onto dst at (x,y) using Jimp's over operator.
func Blit(dst, src *image.NRGBA, x, y int) {
	BlitRect(dst, src, x, y, 0, 0, src.Rect.Dx(), src.Rect.Dy())
}

// BlitRect composites the srcW×srcH region of src at (srcX,srcY) onto dst at
// (x,y). Source pixels outside src read as 0, as they do in Jimp.
func BlitRect(dst, src *image.NRGBA, x, y, srcX, srcY, srcW, srcH int) {
	stride := src.Rect.Dx()
	maxW, maxH := dst.Rect.Dx(), dst.Rect.Dy()
	for sy := range srcH {
		for sx := range srcW {
			xOffset, yOffset := x+sx, y+sy
			if xOffset < 0 || yOffset < 0 || maxW-xOffset <= 0 || maxH-yOffset <= 0 {
				continue
			}
			idx := (stride*(srcY+sy) + srcX + sx) * 4
			if idx < 0 || idx+4 > len(src.Pix) {
				continue
			}
			dstIdx := (maxW*yOffset + xOffset) * 4
			sr, sg, sb, sa := int(src.Pix[idx]), int(src.Pix[idx+1]), int(src.Pix[idx+2]), int(src.Pix[idx+3])
			dr, dg, db, da := int(dst.Pix[dstIdx]), int(dst.Pix[dstIdx+1]), int(dst.Pix[dstIdx+2]), int(dst.Pix[dstIdx+3])
			dst.Pix[dstIdx] = blendChannel(sa, sr, dr)
			dst.Pix[dstIdx+1] = blendChannel(sa, sg, dg)
			dst.Pix[dstIdx+2] = blendChannel(sa, sb, db)
			// #nosec G115 -- limit255 returns a value in 0-255
			dst.Pix[dstIdx+3] = byte(Limit255(da + sa))
		}
	}
}

// Limit255 clamps to the 0–255 range (Jimp's Limit255).
func Limit255(n int) int {
	return max(0, min(255, n))
}

// blendChannel applies Jimp's over-composite to one colour channel; the result
// is within 0-255 for 0-255 inputs.
func blendChannel(sa, s, d int) byte {
	// #nosec G115 -- the over-composite of two 0-255 channels stays in 0-255
	return byte(((sa*(s-d) - d + 255) >> 8) + d)
}

// HAlignIndex/VAlignIndex map alignment labels to Jimp's 0/1/2 factors.
var (
	HAlignIndex = map[string]int{"Left": 0, "Center": 1, "Right": 2}
	VAlignIndex = map[string]int{"Top": 0, "Middle": 1, "Bottom": 2}
)

// Contain scales img to fit within w×h and centres it (per alignment) on a
// transparent canvas.
func Contain(img *image.NRGBA, w, h, alignH, alignV int, mode string) *image.NRGBA {
	W, H := float64(img.Rect.Dx()), float64(img.Rect.Dy())
	var f float64
	if float64(w)/float64(h) > W/H {
		f = float64(h) / H
	} else {
		f = float64(w) / W
	}
	c := Scale(img, f, mode)
	canvas := image.NewNRGBA(image.Rect(0, 0, w, h))
	bx := round((float64(w-c.Rect.Dx()) / 2) * float64(alignH))
	by := round((float64(h-c.Rect.Dy()) / 2) * float64(alignV))
	Blit(canvas, c, bx, by)
	return canvas
}

// Cover scales img to fill w×h and crops the aligned region.
func Cover(img *image.NRGBA, w, h, alignH, alignV int, mode string) (*image.NRGBA, error) {
	W, H := float64(img.Rect.Dx()), float64(img.Rect.Dy())
	var f float64
	if float64(w)/float64(h) > W/H {
		f = float64(w) / W
	} else {
		f = float64(h) / H
	}
	scaled := Scale(img, f, mode)
	x := round((float64(scaled.Rect.Dx()-w) / 2) * float64(alignH))
	y := round((float64(scaled.Rect.Dy()-h) / 2) * float64(alignV))
	return CropExact(scaled, x, y, w, h)
}
