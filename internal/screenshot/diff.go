package screenshot

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiffResult holds the complete result of a screenshot comparison.
type DiffResult struct {
	OverallSSIM float64       `json:"overall_ssim"`
	Clusters    []Cluster     `json:"clusters"`
	Stats       DiffStats     `json:"stats"`
	DimMismatch bool          `json:"dimension_mismatch"`
	Expected    ImageInfo     `json:"expected"`
	Actual      ImageInfo     `json:"actual"`
}

// ImageInfo holds basic image metadata.
type ImageInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DiffStats holds intensity statistics for differences.
type DiffStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	StdDev float64 `json:"stddev"`
}

// DiffOpts holds options for the diff operation.
type DiffOpts struct {
	HeatmapOutput string
	DiffOutput    string
	Threshold     float64 // SSIM threshold for cluster detection (default 0.9)
}

// Diff compares two images and returns a detailed comparison result.
func Diff(expectedPath, actualPath string, opts DiffOpts) (DiffResult, error) {
	img1, err := loadImage(expectedPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to load expected image: %w", err)
	}

	img2, err := loadImage(actualPath)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to load actual image: %w", err)
	}

	b1, b2 := img1.Bounds(), img2.Bounds()
	result := DiffResult{
		Expected: ImageInfo{Width: b1.Dx(), Height: b1.Dy()},
		Actual:   ImageInfo{Width: b2.Dx(), Height: b2.Dy()},
	}

	if b1.Dx() != b2.Dx() || b1.Dy() != b2.Dy() {
		result.DimMismatch = true
		return result, nil
	}

	if opts.Threshold == 0 {
		opts.Threshold = 0.9
	}

	ssimResult, err := ComputeSSIM(img1, img2)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to compute SSIM: %w", err)
	}

	result.OverallSSIM = ssimResult.Score
	result.Clusters = FindClusters(ssimResult, opts.Threshold)
	result.Stats = computeStats(ssimResult)

	// Write heatmap if requested
	if opts.HeatmapOutput != "" {
		heatmap := RenderHeatmap(ssimResult)
		if err := saveImage(opts.HeatmapOutput, heatmap); err != nil {
			return DiffResult{}, fmt.Errorf("failed to save heatmap: %w", err)
		}
	}

	// Write diff image with bounding boxes if requested
	if opts.DiffOutput != "" {
		diffImg := RenderDiffWithBoxes(img1, img2, result.Clusters)
		if err := saveImage(opts.DiffOutput, diffImg); err != nil {
			return DiffResult{}, fmt.Errorf("failed to save diff image: %w", err)
		}
	}

	return result, nil
}

// ToJSON returns the diff result as a JSON string.
func (r DiffResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func computeStats(result SSIMResult) DiffStats {
	var values []float64
	halfWin := windowSize / 2

	for y := halfWin; y < result.Height-halfWin; y++ {
		for x := halfWin; x < result.Width-halfWin; x++ {
			values = append(values, result.Heatmap[y][x])
		}
	}

	if len(values) == 0 {
		return DiffStats{}
	}

	sort.Float64s(values)

	var sum, sumSq float64
	minVal := values[0]
	maxVal := values[len(values)-1]

	for _, v := range values {
		sum += v
		sumSq += v * v
	}

	n := float64(len(values))
	mean := sum / n
	variance := sumSq/n - mean*mean
	stddev := math.Sqrt(math.Max(0, variance))

	median := values[len(values)/2]

	return DiffStats{
		Min:    minVal,
		Max:    maxVal,
		Mean:   mean,
		Median: median,
		StdDev: stddev,
	}
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".png":
		return png.Decode(f)
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	default:
		return nil, fmt.Errorf("unsupported image format: %s (use PNG or JPEG)", ext)
	}
}

func saveImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	default:
		return png.Encode(f, img)
	}
}
