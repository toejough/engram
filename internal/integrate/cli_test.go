package integrate_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/integrate"
)

func TestRealMergeFS_FileExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &integrate.RealMergeFS{}

	g.Expect(fs.FileExists(filepath.Join(dir, "nonexistent.txt"))).To(BeFalse())

	path := filepath.Join(dir, "exists.txt")
	g.Expect(os.WriteFile(path, []byte("content"), 0o644)).To(Succeed())
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

func TestRealMergeFS_Glob(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &integrate.RealMergeFS{}

	g.Expect(os.WriteFile(filepath.Join(dir, "a.md"), []byte(""), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "b.md"), []byte(""), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0o644)).To(Succeed())

	matches, err := fs.Glob(filepath.Join(dir, "*.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(matches).To(HaveLen(2))
}

func TestRealMergeFS_ReadFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &integrate.RealMergeFS{}

	_, err := fs.ReadFile(filepath.Join(dir, "missing.txt"))
	g.Expect(err).To(HaveOccurred())

	path := filepath.Join(dir, "file.txt")
	g.Expect(os.WriteFile(path, []byte("hello world"), 0o644)).To(Succeed())

	content, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("hello world"))
}

func TestRealMergeFS_RemoveAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &integrate.RealMergeFS{}

	subDir := filepath.Join(dir, "subdir")
	g.Expect(os.MkdirAll(subDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(subDir, "file.txt"), []byte(""), 0o644)).To(Succeed())

	err := fs.RemoveAll(subDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(fs.FileExists(subDir)).To(BeFalse())
}

func TestRealMergeFS_WriteFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := &integrate.RealMergeFS{}

	path := filepath.Join(dir, "out.txt")
	err := fs.WriteFile(path, "written content")
	g.Expect(err).ToNot(HaveOccurred())

	data, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("written content"))
}

func TestRunCleanup_EmptyDir_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" → uses os.Getwd(). Per-project dir won't exist there.
	err := integrate.RunCleanup(integrate.CleanupArgs{
		Dir:     "",
		Project: "nonexistent-zz-project-xyzzy",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

func TestRunCleanup_ErrorsIfDirMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := integrate.RunCleanup(integrate.CleanupArgs{
		Dir:     dir,
		Project: "nonexistent-project",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("does not exist"))
	}
}

func TestRunCleanup_RemovesDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	perProjectDir := filepath.Join(dir, ".claude", "projects", "myproject")
	g.Expect(os.MkdirAll(perProjectDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "file.md"), []byte("content"), 0o644)).To(Succeed())

	err := integrate.RunCleanup(integrate.CleanupArgs{
		Dir:     dir,
		Project: "myproject",
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, statErr := os.Stat(perProjectDir)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestRunFeatures_EmptyDir_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" → uses os.Getwd(). No feature files there.
	err := integrate.RunFeatures(integrate.FeaturesArgs{Dir: ""})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunFeatures_NoFeatureFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "design.md"), []byte("# Design\n"), 0o644)).To(Succeed())

	err := integrate.RunFeatures(integrate.FeaturesArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunFeatures_WithFeatureFilesAndIDsRenumbered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

	// Top-level requirements with REQ-1 already present.
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "requirements.md"), []byte("# Requirements\n\n### REQ-1: Existing requirement\n\nContent.\n"), 0o644)).To(Succeed())

	// Feature file with conflicting REQ-1 (will be renumbered to REQ-2).
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "requirements-feat.md"), []byte("# Feature Requirements\n\n### REQ-1: New requirement\n\nNew content.\n"), 0o644)).To(Succeed())

	err := integrate.RunFeatures(integrate.FeaturesArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunFeatures_WithJSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

	err := integrate.RunFeatures(integrate.FeaturesArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_EmptyDir_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" → uses os.Getwd(). No per-project files there.
	err := integrate.RunMerge(integrate.MergeArgs{
		Dir:     "",
		Project: "nonexistent-zz-project-xyzzy",
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_NoPerProjectFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

	err := integrate.RunMerge(integrate.MergeArgs{Dir: dir, Project: "nonexistent"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_WithDesignAndTasksAdded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "design.md"), []byte("# Design\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "architecture.md"), []byte("# Architecture\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "tasks.md"), []byte("# Tasks\n"), 0o644)).To(Succeed())

	perProjectDir := filepath.Join(dir, ".claude", "projects", "myfeature")
	g.Expect(os.MkdirAll(perProjectDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "design.md"), []byte("# Design\n\n## DES-1: Feature design\n\nFeature design decision.\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "architecture.md"), []byte("# Architecture\n\n## ARCH-1: Feature arch\n\nArchitecture decision.\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "tasks.md"), []byte("# Tasks\n\n## TASK-1: Feature task\n\nTask description.\n"), 0o644)).To(Succeed())

	err := integrate.RunMerge(integrate.MergeArgs{Dir: dir, Project: "myfeature"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_WithIDsRenumberedAndLinksUpdated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "requirements.md"), []byte("# Requirements\n\n## REQ-1: Existing\n\nThe current content.\n"), 0o644)).To(Succeed())

	perProjectDir := filepath.Join(dir, ".claude", "projects", "feat")
	g.Expect(os.MkdirAll(perProjectDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "requirements.md"), []byte("# Feature Requirements\n\n## REQ-1: Revised\n\nFeature body.\n"), 0o644)).To(Succeed())

	// Traceability file with a reference to the old REQ-1.
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "traceability.toml"), []byte("[[links]]\nfrom = \"REQ-1\"\nto = \"TASK-1\"\n"), 0o644)).To(Succeed())

	err := integrate.RunMerge(integrate.MergeArgs{Dir: dir, Project: "feat"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_WithJSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

	err := integrate.RunMerge(integrate.MergeArgs{Dir: dir, Project: "testproject", JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunMerge_WithSummaryOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "docs", "requirements.md"), []byte("# Requirements\n"), 0o644)).To(Succeed())

	perProjectDir := filepath.Join(dir, ".claude", "projects", "myfeature")
	g.Expect(os.MkdirAll(perProjectDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(perProjectDir, "requirements.md"), []byte(`# Requirements

## REQ-001: Feature Req

Feature requirement.
`), 0o644)).To(Succeed())

	err := integrate.RunMerge(integrate.MergeArgs{Dir: dir, Project: "myfeature"})
	g.Expect(err).ToNot(HaveOccurred())
}
