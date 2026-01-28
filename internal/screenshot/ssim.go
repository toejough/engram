// Package screenshot provides SSIM-based image comparison and spatial clustering.
package screenshot

import (
	"image"
	"image/color"
	"math"
)

// SSIM constants from Wang et al. 2004.
const (
	k1         = 0.01
	k2         = 0.03
	windowSize = 11
	// L is the dynamic range for 8-bit images.
	dynamicRange = 255.0
)

var (
	c1 = math.Pow(k1*dynamicRange, 2)
	c2 = math.Pow(k2*dynamicRange, 2)
)

// SSIMResult holds the result of an SSIM comparison.
type SSIMResult struct {
	Score   float64     `json:"score"`
	Heatmap [][]float64 `json:"-"` // per-pixel SSIM values
	Width   int         `json:"width"`
	Height  int         `json:"height"`
}

// ComputeSSIM computes the structural similarity between two images.
// Returns a score between 0.0 (completely different) and 1.0 (identical).
func ComputeSSIM(img1, img2 image.Image) (SSIMResult, error) {
	b1 := img1.Bounds()
	b2 := img2.Bounds()

	if b1.Dx() != b2.Dx() || b1.Dy() != b2.Dy() {
		return SSIMResult{}, &DimensionMismatchError{
			Width1: b1.Dx(), Height1: b1.Dy(),
			Width2: b2.Dx(), Height2: b2.Dy(),
		}
	}

	w, h := b1.Dx(), b1.Dy()
	lum1 := toLuminance(img1)
	lum2 := toLuminance(img2)

	// Compute SSIM for each window position
	heatmap := make([][]float64, h)
	for y := range heatmap {
		heatmap[y] = make([]float64, w)
		for x := range heatmap[y] {
			heatmap[y][x] = 1.0 // default to identical for border pixels
		}
	}

	halfWin := windowSize / 2
	var sum float64
	var count int

	for y := halfWin; y < h-halfWin; y++ {
		for x := halfWin; x < w-halfWin; x++ {
			s := windowSSIM(lum1, lum2, x, y, w)
			heatmap[y][x] = s
			sum += s
			count++
		}
	}

	score := 1.0
	if count > 0 {
		score = sum / float64(count)
	}

	return SSIMResult{
		Score:   score,
		Heatmap: heatmap,
		Width:   w,
		Height:  h,
	}, nil
}

// DimensionMismatchError is returned when images have different dimensions.
type DimensionMismatchError struct {
	Width1, Height1 int
	Width2, Height2 int
}

func (e *DimensionMismatchError) Error() string {
	return "dimension mismatch"
}

func windowSSIM(lum1, lum2 []float64, cx, cy, stride int) float64 {
	halfWin := windowSize / 2

	var sum1, sum2, sumSq1, sumSq2, sumCross float64
	n := float64(windowSize * windowSize)

	for dy := -halfWin; dy <= halfWin; dy++ {
		for dx := -halfWin; dx <= halfWin; dx++ {
			idx := (cy+dy)*stride + (cx + dx)
			v1 := lum1[idx]
			v2 := lum2[idx]
			sum1 += v1
			sum2 += v2
			sumSq1 += v1 * v1
			sumSq2 += v2 * v2
			sumCross += v1 * v2
		}
	}

	mu1 := sum1 / n
	mu2 := sum2 / n
	sigma1Sq := sumSq1/n - mu1*mu1
	sigma2Sq := sumSq2/n - mu2*mu2
	sigma12 := sumCross/n - mu1*mu2

	numerator := (2*mu1*mu2 + c1) * (2*sigma12 + c2)
	denominator := (mu1*mu1 + mu2*mu2 + c1) * (sigma1Sq + sigma2Sq + c2)

	if denominator == 0 {
		return 1.0
	}

	return numerator / denominator
}

// toLuminance converts an image to a flat array of luminance values [0, 255].
func toLuminance(img image.Image) []float64 {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	lum := make([]float64, w*h)

	for y := range h {
		for x := range w {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// Convert from 16-bit to 8-bit and compute luminance
			rf := float64(r) / 256.0
			gf := float64(g) / 256.0
			bf := float64(bl) / 256.0
			lum[y*w+x] = 0.2126*rf + 0.7152*gf + 0.0722*bf
		}
	}

	return lum
}

// RenderHeatmap creates a visual representation of the SSIM heatmap.
// Green = similar, Red = different.
func RenderHeatmap(result SSIMResult) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, result.Width, result.Height))

	for y := range result.Height {
		for x := range result.Width {
			s := result.Heatmap[y][x]
			// Map SSIM [0,1] to green (1.0) → red (0.0)
			r := uint8(255 * (1 - s))
			g := uint8(255 * s)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: 0, A: 255})
		}
	}

	return img
}
