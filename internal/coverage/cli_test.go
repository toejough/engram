package coverage_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/coverage"
)

func TestRealCoverageFS_DirExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &coverage.RealCoverageFS{}

	g.Expect(fs.DirExists("/non-existent-coverage-dir-xyz")).To(BeFalse())
}

func TestRealCoverageFS_DirExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &coverage.RealCoverageFS{}

	g.Expect(fs.DirExists(dir)).To(BeTrue())
}

func TestRealCoverageFS_FileExists_False(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &coverage.RealCoverageFS{}

	g.Expect(fs.FileExists("/non-existent-coverage-file-xyz")).To(BeFalse())
}

func TestRealCoverageFS_FileExists_True(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	g.Expect(os.WriteFile(path, []byte("package main"), 0o644)).To(Succeed())

	fs := &coverage.RealCoverageFS{}

	g.Expect(fs.FileExists(path)).To(BeTrue())
}

func TestRealCoverageFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	g.Expect(os.WriteFile(path, []byte("hello coverage"), 0o644)).To(Succeed())

	fs := &coverage.RealCoverageFS{}

	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("hello coverage"))
}

func TestRealCoverageFS_ReadFile_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &coverage.RealCoverageFS{}

	_, err := fs.ReadFile("/non-existent-coverage-read-xyz")
	g.Expect(err).To(HaveOccurred())
}

func TestRealCoverageFS_Walk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub")
	g.Expect(os.Mkdir(subdir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "a.go"), []byte("package p"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subdir, "b.go"), []byte("package p"), 0o644)).To(Succeed())

	fs := &coverage.RealCoverageFS{}

	var paths []string

	err := fs.Walk(dir, func(path string, _ bool) error {
		paths = append(paths, path)
		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(paths).To(ContainElement(filepath.Join(dir, "a.go")))
	g.Expect(paths).To(ContainElement(filepath.Join(subdir, "b.go")))
}

func TestRealCoverageFS_Walk_SkipError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	subdir := filepath.Join(dir, "skipme")
	g.Expect(os.Mkdir(subdir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "kept.go"), []byte("p"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subdir, "skipped.go"), []byte("p"), 0o644)).To(Succeed())

	fs := &coverage.RealCoverageFS{}

	var paths []string

	err := fs.Walk(dir, func(path string, isDir bool) error {
		if isDir && filepath.Base(path) == "skipme" {
			return errCoverageSkip
		}

		paths = append(paths, path)

		return nil
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(paths).To(ContainElement(filepath.Join(dir, "kept.go")))
	g.Expect(paths).ToNot(ContainElement(filepath.Join(subdir, "skipped.go")))
}

func TestRunAnalyze_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := coverage.RunAnalyze(coverage.AnalyzeArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunAnalyze_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir:"" triggers os.Getwd(); CWD during tests is the package directory
	err := coverage.RunAnalyze(coverage.AnalyzeArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunAnalyze_WithBadConfig(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("skipping chmod test when running as root")
	}

	g := NewWithT(t)
	dir := t.TempDir()

	// Create .claude/project-config.toml then make it unreadable
	// realConfigFS.FileExists returns true (os.Stat works on chmod 000 files)
	// realConfigFS.ReadFile returns error (os.ReadFile fails on chmod 000 files)
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	configPath := filepath.Join(claudeDir, "project-config.toml")
	g.Expect(os.WriteFile(configPath, []byte("[paths]\n"), 0o644)).To(Succeed())
	g.Expect(os.Chmod(configPath, 0o000)).To(Succeed())
	t.Cleanup(func() { _ = os.Chmod(configPath, 0o644) })

	err := coverage.RunAnalyze(coverage.AnalyzeArgs{Dir: dir})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to load config"))
	}
}

func TestRunAnalyze_WithConfig(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create .claude/project-config.toml to trigger realConfigFS.ReadFile
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(claudeDir, "project-config.toml"), []byte(`[paths]
readme = "README.md"
`), 0o644)).To(Succeed())

	err := coverage.RunAnalyze(coverage.AnalyzeArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunAnalyze_WithGoFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a Go file with exports
	g.Expect(os.WriteFile(filepath.Join(dir, "api.go"), []byte(`package main

func ExportedFunc() {}
type ExportedType struct{}
`), 0o644)).To(Succeed())

	err := coverage.RunAnalyze(coverage.AnalyzeArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := coverage.RunReport(coverage.ReportArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_Preserve(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create requirements.md at root (DocsDir defaults to "" = root)
	// With documented > 0 and inferred = 0: ratio = 1.0 → "preserve"
	g.Expect(os.WriteFile(filepath.Join(dir, "requirements.md"), []byte(`# Requirements

## REQ-001: First requirement
## REQ-002: Second requirement
## REQ-003: Third requirement
`), 0o644)).To(Succeed())

	err := coverage.RunReport(coverage.ReportArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir:"" triggers os.Getwd(); CWD during tests is the package directory
	err := coverage.RunReport(coverage.ReportArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_WithDocumentation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a requirements file with IDs
	docsDir := filepath.Join(dir, "docs")
	g.Expect(os.MkdirAll(docsDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(docsDir, "requirements.md"), []byte(`
## REQ-001: Requirement one
## REQ-002: Requirement two
`), 0o644)).To(Succeed())

	err := coverage.RunReport(coverage.ReportArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

// unexported constants.
const (
	errCoverageSkip = skipError("skip")
)

type skipError string

func (e skipError) Error() string { return string(e) }
