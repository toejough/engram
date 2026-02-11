package screenshot_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/screenshot"
)

// MockFS implements screenshot.FileSystem for testing.
type MockFS struct {
	Files map[string][]byte
}

type mockFileInfo struct {
	name string
	size int64
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }

func (m *MockFS) Open(path string) (io.ReadCloser, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

func (m *MockFS) Create(path string) (io.WriteCloser, error) {
	return &mockWriteCloser{fs: m, path: path, buf: new(bytes.Buffer)}, nil
}

func (m *MockFS) Stat(path string) (os.FileInfo, error) {
	content, exists := m.Files[path]
	if !exists {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return mockFileInfo{name: filepath.Base(path), size: int64(len(content))}, nil
}

func (m *MockFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.Files[path] = data
	return nil
}

type mockWriteCloser struct {
	fs   *MockFS
	path string
	buf  *bytes.Buffer
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.fs.Files[m.path] = m.buf.Bytes()
	return nil
}

func solidImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetRGBA(x, y, c)
		}
	}

	return img
}

func savePNG(t *testing.T, fs *MockFS, dir, name string, img image.Image) string {
	t.Helper()

	path := filepath.Join(dir, name)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	fs.Files[path] = buf.Bytes()

	return path
}

func TestComputeSSIM(t *testing.T) {
	t.Parallel()
	t.Run("identical images score 1.0", func(t *testing.T) {
		g := NewWithT(t)
		img := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

		result, err := screenshot.ComputeSSIM(img, img)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Score).To(BeNumerically("~", 1.0, 0.001))
	})

	t.Run("different images score below 1.0", func(t *testing.T) {
		g := NewWithT(t)
		img1 := solidImage(50, 50, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		img2 := solidImage(50, 50, color.RGBA{R: 255, G: 255, B: 255, A: 255})

		result, err := screenshot.ComputeSSIM(img1, img2)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Score).To(BeNumerically("<", 0.1))
	})

	t.Run("dimension mismatch returns error", func(t *testing.T) {
		g := NewWithT(t)
		img1 := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		img2 := solidImage(60, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

		_, err := screenshot.ComputeSSIM(img1, img2)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("dimension mismatch"))
	})

	t.Run("heatmap has correct dimensions", func(t *testing.T) {
		g := NewWithT(t)
		img := solidImage(30, 20, color.RGBA{R: 100, G: 100, B: 100, A: 255})

		result, err := screenshot.ComputeSSIM(img, img)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.Width).To(Equal(30))
		g.Expect(result.Height).To(Equal(20))
		g.Expect(result.Heatmap).To(HaveLen(20))
		g.Expect(result.Heatmap[0]).To(HaveLen(30))
	})
}

func TestFindClusters(t *testing.T) {
	t.Parallel()
	t.Run("no clusters for identical images", func(t *testing.T) {
		g := NewWithT(t)
		img := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

		result, err := screenshot.ComputeSSIM(img, img)
		g.Expect(err).ToNot(HaveOccurred())

		clusters := screenshot.FindClusters(result, 0.9)
		g.Expect(clusters).To(BeEmpty())
	})

	t.Run("finds clusters for different regions", func(t *testing.T) {
		g := NewWithT(t)

		img1 := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		img2 := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})

		// Paint a different block in img2
		for y := 20; y < 30; y++ {
			for x := 20; x < 30; x++ {
				img2.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
			}
		}

		result, err := screenshot.ComputeSSIM(img1, img2)
		g.Expect(err).ToNot(HaveOccurred())

		clusters := screenshot.FindClusters(result, 0.9)
		g.Expect(clusters).ToNot(BeEmpty())
		// Should have at least one cluster around the changed region
		g.Expect(clusters[0].PixelCount).To(BeNumerically(">", 0))
	})
}

func TestDiff(t *testing.T) {
	t.Parallel()
	t.Run("compares identical images", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}
		dir := t.TempDir()

		img := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		path1 := savePNG(t, fs, dir, "expected.png", img)
		path2 := savePNG(t, fs, dir, "actual.png", img)

		result, err := screenshot.Diff(path1, path2, screenshot.DiffOpts{}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.OverallSSIM).To(BeNumerically("~", 1.0, 0.001))
		g.Expect(result.DimMismatch).To(BeFalse())
	})

	t.Run("reports dimension mismatch", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}
		dir := t.TempDir()

		path1 := savePNG(t, fs, dir, "expected.png", solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		path2 := savePNG(t, fs, dir, "actual.png", solidImage(60, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255}))

		result, err := screenshot.Diff(path1, path2, screenshot.DiffOpts{}, fs)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result.DimMismatch).To(BeTrue())
		g.Expect(result.Expected.Width).To(Equal(50))
		g.Expect(result.Actual.Width).To(Equal(60))
	})

	t.Run("writes heatmap output", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}
		dir := t.TempDir()

		img := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		path1 := savePNG(t, fs, dir, "expected.png", img)
		path2 := savePNG(t, fs, dir, "actual.png", img)
		heatmapPath := filepath.Join(dir, "heatmap.png")

		_, err := screenshot.Diff(path1, path2, screenshot.DiffOpts{
			HeatmapOutput: heatmapPath,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		_, err = fs.Stat(heatmapPath)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("writes diff output with bounding boxes", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}
		dir := t.TempDir()

		img1 := solidImage(50, 50, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		img2 := solidImage(50, 50, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		path1 := savePNG(t, fs, dir, "expected.png", img1)
		path2 := savePNG(t, fs, dir, "actual.png", img2)
		diffPath := filepath.Join(dir, "diff.png")

		_, err := screenshot.Diff(path1, path2, screenshot.DiffOpts{
			DiffOutput: diffPath,
		}, fs)
		g.Expect(err).ToNot(HaveOccurred())

		_, err = fs.Stat(diffPath)
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("errors on missing file", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}

		_, err := screenshot.Diff("/nonexistent/a.png", "/nonexistent/b.png", screenshot.DiffOpts{}, fs)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("errors on unsupported format", func(t *testing.T) {
		g := NewWithT(t)
		fs := &MockFS{Files: make(map[string][]byte)}
		dir := t.TempDir()

		// Create a file with wrong extension
		path := filepath.Join(dir, "image.bmp")
		g.Expect(fs.WriteFile(path, []byte("not an image"), 0o644)).To(Succeed())

		_, err := screenshot.Diff(path, path, screenshot.DiffOpts{}, fs)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("unsupported"))
	})
}

func TestRenderHeatmap(t *testing.T) {
	t.Parallel()
	t.Run("produces image with correct dimensions", func(t *testing.T) {
		g := NewWithT(t)

		result := screenshot.SSIMResult{
			Width:   10,
			Height:  10,
			Heatmap: make([][]float64, 10),
		}
		for y := range result.Heatmap {
			result.Heatmap[y] = make([]float64, 10)
			for x := range result.Heatmap[y] {
				result.Heatmap[y][x] = 0.5
			}
		}

		img := screenshot.RenderHeatmap(result)
		g.Expect(img.Bounds().Dx()).To(Equal(10))
		g.Expect(img.Bounds().Dy()).To(Equal(10))
	})
}
