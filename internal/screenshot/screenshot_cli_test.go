package screenshot_test

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/screenshot"
)

func TestDiffResult_ToJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := screenshot.DiffResult{
		OverallSSIM: 0.95,
		DimMismatch: false,
		Expected:    screenshot.ImageInfo{Width: 100, Height: 100},
		Actual:      screenshot.ImageInfo{Width: 100, Height: 100},
	}

	json, err := result.ToJSON()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(json).To(ContainSubstring(`"overall_ssim"`))
	g.Expect(json).To(ContainSubstring(`"dimension_mismatch"`))
}

func TestDiffResult_ToJSON_NaNError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// NaN float64 causes json.MarshalIndent to fail, covering the error return path.
	result := screenshot.DiffResult{
		OverallSSIM: math.NaN(),
	}

	_, err := result.ToJSON()
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Create_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := screenshot.RealFS{}

	path := filepath.Join(dir, "out.txt")
	w, err := fs.Create(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(w).ToNot(BeNil())

	if w != nil {
		g.Expect(w.Close()).To(Succeed())
	}

	_, statErr := os.Stat(path)
	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestRealFS_Open_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := screenshot.RealFS{}

	path := filepath.Join(dir, "existing.txt")
	g.Expect(os.WriteFile(path, []byte("hello"), 0o644)).To(Succeed())

	r, err := fs.Open(path)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(r).ToNot(BeNil())

	if r != nil {
		g.Expect(r.Close()).To(Succeed())
	}
}

func TestRealFS_Open_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := screenshot.RealFS{}

	_, err := fs.Open("/nonexistent/path/file.png")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Stat_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := screenshot.RealFS{}

	path := filepath.Join(dir, "file.txt")
	g.Expect(os.WriteFile(path, []byte("data"), 0o644)).To(Succeed())

	info, err := fs.Stat(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info).ToNot(BeNil())

	if info != nil {
		g.Expect(info.Name()).To(Equal("file.txt"))
	}
}

func TestRealFS_Stat_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := screenshot.RealFS{}

	_, err := fs.Stat("/nonexistent/path")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_WriteFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := screenshot.RealFS{}

	path := filepath.Join(dir, "written.txt")
	err := fs.WriteFile(path, []byte("content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	data, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("content"))
}

func TestRunDiff_MissingExpected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := screenshot.RunDiff(screenshot.DiffArgs{
		Expected: "/nonexistent/expected.png",
		Actual:   "/nonexistent/actual.png",
	})
	g.Expect(err).To(HaveOccurred())
}

func TestRunDiff_WithValidImages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	expectedPath := filepath.Join(dir, "expected.png")
	actualPath := filepath.Join(dir, "actual.png")

	writeCliTestPNG(t, expectedPath)
	writeCliTestPNG(t, actualPath)

	err := screenshot.RunDiff(screenshot.DiffArgs{
		Expected: expectedPath,
		Actual:   actualPath,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

// writeCliTestPNG writes a small solid grey PNG to path on disk.
func writeCliTestPNG(t *testing.T, path string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 20, 20))

	for y := range 20 {
		for x := range 20 {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
