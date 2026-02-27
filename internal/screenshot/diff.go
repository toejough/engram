package screenshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiffOpts holds options for the diff operation.
type DiffOpts struct {
	HeatmapOutput string
	DiffOutput    string
	Threshold     float64 // SSIM threshold for cluster detection (default 0.9)
}

// DiffResult holds the complete result of a screenshot comparison.
type DiffResult struct {
	OverallSSIM float64   `json:"overall_ssim"`
	Clusters    []Cluster `json:"clusters"`
	Stats       DiffStats `json:"stats"`
	DimMismatch bool      `json:"dimension_mismatch"`
	Expected    ImageInfo `json:"expected"`
	Actual      ImageInfo `json:"actual"`
}

// ToJSON returns the diff result as a JSON string.
func (r DiffResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// DiffStats holds intensity statistics for differences.
type DiffStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	StdDev float64 `json:"stddev"`
}

// FileSystem provides file system operations for screenshot diffing.
type FileSystem interface {
	Open(path string) (io.ReadCloser, error)
	Create(path string) (io.WriteCloser, error)
	Stat(path string) (os.FileInfo, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
}

// ImageInfo holds basic image metadata.
type ImageInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RealFS implements FileSystem using the real file system.
type RealFS struct{}

// Create creates a file for writing.
func (RealFS) Create(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

// Open opens a file for reading.
func (RealFS) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// Stat returns file information.
func (RealFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// WriteFile writes data to a file.
func (RealFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Diff compares two images and returns a detailed comparison result.
func Diff(expectedPath, actualPath string, opts DiffOpts, fs FileSystem) (DiffResult, error) {
	img1, err := loadImage(expectedPath, fs)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to load expected image: %w", err)
	}

	img2, err := loadImage(actualPath, fs)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to load actual image: %w", err)
	}

	if img1 == nil {
		return DiffResult{}, errors.New("failed to decode expected image: nil result")
	}

	if img2 == nil {
		return DiffResult{}, errors.New("failed to decode actual image: nil result")
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

		err := saveImage(opts.HeatmapOutput, heatmap, fs)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to save heatmap: %w", err)
		}
	}

	// Write diff image with bounding boxes if requested
	if opts.DiffOutput != "" {
		diffImg := RenderDiffWithBoxes(img1, img2, result.Clusters)

		err := saveImage(opts.DiffOutput, diffImg, fs)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to save diff image: %w", err)
		}
	}

	return result, nil
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

func loadImage(path string, fs FileSystem) (image.Image, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() { _ = f.Close() }()

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

func saveImage(path string, img image.Image, fs FileSystem) error {
	f, err := fs.Create(path)
	if err != nil {
		return err
	}

	defer func() { _ = f.Close() }()

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 90})
	default:
		return png.Encode(f, img)
	}
}
