package ops

import (
	"image"
	"math"

	"github.com/roberson-io/cchef/internal/jimp"
)

// Blur kernels ported byte-for-byte from @jimp/plugin-blur (the Superfast Blur
// box filter and a Gaussian filter). jimpGaussian is also used by Sharpen Image.

// jimpBlurFast applies Mario Klingemann's Superfast Blur (two passes) with the
// mulTable/shgTable fixed-point tables. r must be in [1, 256].
func jimpBlurFast(img *image.NRGBA, r int) {
	data := img.Pix
	w, h := img.Rect.Dx(), img.Rect.Dy()
	wm, hm := w-1, h-1
	rad1 := r + 1
	mulSum := blurMulTable[r]
	shgSum := uint(blurShgTable[r]) // #nosec G115 -- shift amount, small and positive
	n := w * h
	red := make([]int64, n)
	green := make([]int64, n)
	blue := make([]int64, n)
	alpha := make([]int64, n)
	vmin := make([]int, max(w, h))
	vmax := make([]int, max(w, h))

	for range 2 {
		yi, yw := 0, 0
		for y := range h {
			rsum := int64(data[yw]) * int64(rad1)
			gsum := int64(data[yw+1]) * int64(rad1)
			bsum := int64(data[yw+2]) * int64(rad1)
			asum := int64(data[yw+3]) * int64(rad1)
			for i := 1; i <= r; i++ {
				p := yw + (min(i, wm) << 2)
				rsum += int64(data[p])
				gsum += int64(data[p+1])
				bsum += int64(data[p+2])
				asum += int64(data[p+3])
			}
			for x := range w {
				red[yi], green[yi], blue[yi], alpha[yi] = rsum, gsum, bsum, asum
				if y == 0 {
					vmin[x] = min(x+rad1, wm) << 2
					vmax[x] = posOrZero(x-r) << 2
				}
				p1 := yw + vmin[x]
				p2 := yw + vmax[x]
				rsum += int64(data[p1]) - int64(data[p2])
				gsum += int64(data[p1+1]) - int64(data[p2+1])
				bsum += int64(data[p1+2]) - int64(data[p2+2])
				asum += int64(data[p1+3]) - int64(data[p2+3])
				yi++
			}
			yw += w << 2
		}
		for x := range w {
			yp := x
			rsum := red[yp] * int64(rad1)
			gsum := green[yp] * int64(rad1)
			bsum := blue[yp] * int64(rad1)
			asum := alpha[yp] * int64(rad1)
			for i := 1; i <= r; i++ {
				if i <= hm {
					yp += w
				}
				rsum += red[yp]
				gsum += green[yp]
				bsum += blue[yp]
				asum += alpha[yp]
			}
			yi := x << 2
			for y := range h {
				data[yi] = blurSample(rsum, mulSum, shgSum)
				data[yi+1] = blurSample(gsum, mulSum, shgSum)
				data[yi+2] = blurSample(bsum, mulSum, shgSum)
				data[yi+3] = blurSample(asum, mulSum, shgSum)
				if x == 0 {
					vmin[y] = min(y+rad1, hm) * w
					vmax[y] = posOrZero(y-r) * w
				}
				p1 := x + vmin[y]
				p2 := x + vmax[y]
				rsum += red[p1] - red[p2]
				gsum += green[p1] - green[p2]
				bsum += blue[p1] - blue[p2]
				asum += alpha[p1] - alpha[p2]
				yi += w << 2
			}
		}
	}
}

// blurSample reduces a fixed-point channel sum to a 0-255 byte, reproducing
// Jimp's `limit255((sum * mul) >>> shg)` (an unsigned 32-bit shift).
func blurSample(sum, mul int64, shg uint) byte {
	// #nosec G115 -- ToUint32 of the product then a 0-255 limit; both intentional
	return byte(jimp.Limit255(int(uint32(sum*mul) >> shg)))
}

// posOrZero returns v if positive, else 0 (Jimp's `p > 0 ? p : 0`).
func posOrZero(v int) int {
	if v > 0 {
		return v
	}
	return 0
}

// jimpGaussian applies a Gaussian blur. It reproduces Jimp's in-place scan,
// where the destination pixel is written inside the kernel's row loop, so later
// pixels convolve over already-blurred neighbours.
func jimpGaussian(img *image.NRGBA, r float64) {
	data := img.Pix
	w, h := img.Rect.Dx(), img.Rect.Dy()
	rs := int(math.Ceil(r * 2.57))
	rng := rs*2 + 1
	rr2 := r * r * 2
	rr2pi := rr2 * math.Pi
	weights := make([][]float64, rng)
	for y := range rng {
		weights[y] = make([]float64, rng)
		for x := range rng {
			dsq := float64((x-rs)*(x-rs) + (y-rs)*(y-rs))
			weights[y][x] = math.Exp(-dsq/rr2) / rr2pi
		}
	}
	for y := range h {
		for x := range w {
			var red, green, blue, alpha, wsum float64
			for iy := range rng {
				for ix := range rng {
					x1 := min(w-1, max(0, ix+x-rs))
					y1 := min(h-1, max(0, iy+y-rs))
					weight := weights[iy][ix]
					idx := (y1*w + x1) * 4
					red += float64(data[idx]) * weight
					green += float64(data[idx+1]) * weight
					blue += float64(data[idx+2]) * weight
					alpha += float64(data[idx+3]) * weight
					wsum += weight
				}
				idx := (y*w + x) * 4
				data[idx] = byte(math.Round(red / wsum))
				data[idx+1] = byte(math.Round(green / wsum))
				data[idx+2] = byte(math.Round(blue / wsum))
				data[idx+3] = byte(math.Round(alpha / wsum))
			}
		}
	}
}
