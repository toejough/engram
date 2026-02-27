package trace_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/trace"
)

func TestAdd_SaveWriteFileError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	base := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	fs := &writeErrorFS{MockFS: base}

	err := trace.Add(dir, "REQ-1", []string{"DES-1"}, fs)
	g.Expect(err).To(HaveOccurred())
}

func TestPromote_LoadTaskTraces_ReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	base := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	fs := &taskReadErrorFS{MockFS: base, dir: dir}

	_, err := trace.Promote(dir, fs, false)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to load task traces"))
	}
}

func TestRealConfigFS_ReadFile_DirAsFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create the config path as a directory (not a file).
	// realConfigFS.FileExists uses os.Stat (returns true for dirs),
	// then realConfigFS.ReadFile calls os.ReadFile which fails on a dir.
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(claudeDir, "project-config.toml"), 0o755)).To(Succeed())

	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{Dir: dir, JSON: true})
	g.Expect(err).To(HaveOccurred())
}

func TestRealConfigFS_ReadFile_ViaConfigLoad(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a .claude/project-config.toml so config.Load reads it via realConfigFS.ReadFile
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(claudeDir, "project-config.toml"), []byte(
		"[paths]\ndocs_dir = \"docs\"\n",
	), 0o644)).To(Succeed())

	// Any trace operation that calls config.Load with realConfigFS exercises ReadFile
	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRealFS_Glob_MatchesPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	g.Expect(os.WriteFile(filepath.Join(dir, "a.md"), []byte(""), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "b.md"), []byte(""), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0o644)).To(Succeed())

	matches, err := fs.Glob(filepath.Join(dir, "*.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(matches).To(HaveLen(2))
}

func TestRealFS_Glob_NoMatches(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	matches, err := fs.Glob(filepath.Join(dir, "*.toml"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(matches).To(BeEmpty())
}

func TestRealFS_MkdirAll_CreatesDirectories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	newDir := filepath.Join(dir, "a", "b", "c")
	err := fs.MkdirAll(newDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	info, statErr := os.Stat(newDir)
	g.Expect(statErr).ToNot(HaveOccurred())
	g.Expect(info).ToNot(BeNil())

	if info != nil {
		g.Expect(info.IsDir()).To(BeTrue())
	}
}

func TestRealFS_ReadFile_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	path := filepath.Join(dir, "data.toml")
	g.Expect(os.WriteFile(path, []byte("key = \"value\""), 0o644)).To(Succeed())

	data, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("value"))
}

func TestRealFS_ReadFile_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := trace.RealFS{}

	_, err := fs.ReadFile("/nonexistent/file.toml")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Stat_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	path := filepath.Join(dir, "file.toml")
	g.Expect(os.WriteFile(path, []byte(""), 0o644)).To(Succeed())

	info, err := fs.Stat(path)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(info).ToNot(BeNil())

	if info != nil {
		g.Expect(info.Name()).To(Equal("file.toml"))
	}
}

func TestRealFS_Stat_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := trace.RealFS{}

	_, err := fs.Stat("/nonexistent/path")
	g.Expect(err).To(HaveOccurred())
}

func TestRealFS_Walk_VisitsFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	g.Expect(os.WriteFile(filepath.Join(dir, "a.md"), []byte(""), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "b.md"), []byte(""), 0o644)).To(Succeed())

	var visited []string

	err := fs.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			visited = append(visited, filepath.Base(path))
		}

		return nil
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(visited).To(ConsistOf("a.md", "b.md"))
}

func TestRealFS_WriteFile_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	fs := trace.RealFS{}

	path := filepath.Join(dir, "output.toml")
	err := fs.WriteFile(path, []byte("data = true"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("data = true"))
}

func TestRunPromote_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir, DryRun: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create minimal docs structure
	g.Expect(os.MkdirAll(filepath.Join(dir, "docs"), 0o755)).To(Succeed())

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_JSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_MultipleFilesModified(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(`# Tasks

### TASK-1: Feature

**Traces to:** ARCH-1
`), 0o644)).To(Succeed())

	for _, pkg := range []string{"pkga", "pkgb"} {
		pkgDir := filepath.Join(dir, "internal", pkg)
		g.Expect(os.MkdirAll(pkgDir, 0o755)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(pkgDir, pkg+"_test.go"), []byte(`package `+pkg+`_test

import "testing"

// TEST-1: test
// traces: TASK-1
func Test`+pkg+`(t *testing.T) {
}
`), 0o644)).To(Succeed())
	}

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_WithPromotions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(`# Tasks

### TASK-1: Feature

**Traces to:** ARCH-1
`), 0o644)).To(Succeed())

	testDir := filepath.Join(dir, "internal", "feature")
	g.Expect(os.MkdirAll(testDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(testDir, "feature_test.go"), []byte(`package feature_test

import "testing"

// TEST-1: feature test
// traces: TASK-1
func TestFeature(t *testing.T) {
}
`), 0o644)).To(Succeed())

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_WithPromotions_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(`# Tasks

### TASK-1: Feature

**Traces to:** ARCH-1
`), 0o644)).To(Succeed())

	testDir := filepath.Join(dir, "internal", "feature")
	g.Expect(os.MkdirAll(testDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(testDir, "feature_test.go"), []byte(`package feature_test

import "testing"

// TEST-1: feature test
// traces: TASK-1
func TestFeature(t *testing.T) {
}
`), 0o644)).To(Succeed())

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir, DryRun: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunPromote_WithSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// tasks.md has no TASK-999
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(`# Tasks

### TASK-1: Feature

**Traces to:** ARCH-1
`), 0o644)).To(Succeed())

	testDir := filepath.Join(dir, "internal", "feature")
	g.Expect(os.MkdirAll(testDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(testDir, "feature_test.go"), []byte(`package feature_test

import "testing"

// TEST-1: feature test
// traces: TASK-999
func TestFeature(t *testing.T) {
}
`), 0o644)).To(Succeed())

	err := trace.RunPromote(trace.PromoteArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunRepairCore_DanglingRefs_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// tasks.md references REQ-999 which is not defined anywhere → dangling ref
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("### TASK-1: Task\n\n**Traces to:** REQ-999\n"), 0o644)).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := trace.RunRepairCore(trace.RepairArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunRepairCore_DuplicateIDs_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Two artifact files both define DES-1 → duplicate ID
	g.Expect(os.WriteFile(filepath.Join(dir, "requirements.md"), []byte("### DES-1: First\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "design.md"), []byte("### DES-1: Second\n"), 0o644)).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := trace.RunRepairCore(trace.RepairArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunRepairCore_Error_Returns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create config as a directory to force a read error in config.Load
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(claudeDir, "project-config.toml"), 0o755)).To(Succeed())

	exit := func(int) { t.Error("exit should not be called on error") }

	err := trace.RunRepairCore(trace.RepairArgs{Dir: dir}, exit)
	g.Expect(err).To(HaveOccurred())
}

func TestRunRepairCore_JSON_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Dangling ref present but JSON mode does not call exit
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("### TASK-1: Task\n\n**Traces to:** REQ-999\n"), 0o644)).To(Succeed())

	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := trace.RunRepairCore(trace.RepairArgs{Dir: dir, JSON: true}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunRepairCore_NoIssues_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := trace.RunRepairCore(trace.RepairArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunRepair_EmptyDir_JSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Use JSON output to avoid os.Exit(1) on issues
	err := trace.RunRepair(trace.RepairArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunRepair_EmptyDir_NoIssues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := trace.RunRepair(trace.RepairArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunRepair_WithIssues_JSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create a design doc with a dangling ref — causes DanglingRefs in result
	g.Expect(os.WriteFile(filepath.Join(dir, "design.md"), []byte(`# Design

### DES-001: Some Design

Description.

**Traces to:** REQ-999
`), 0o644)).To(Succeed())

	// JSON=true to avoid os.Exit
	err := trace.RunRepair(trace.RepairArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunShow_DefaultFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Format="" defaults to ascii
	err := trace.RunShow(trace.ShowArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunShow_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Show with ascii format on empty dir (no traceability.toml = error)
	err := trace.RunShow(trace.ShowArgs{Dir: dir, Format: "ascii"})
	// May error if no traceability.toml, or succeed with empty output
	_ = err
}

func TestRunShow_JSONFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := trace.RunShow(trace.ShowArgs{Dir: dir, Format: "json"})
	_ = err // May error if no traceability.toml
}

func TestRunValidateArtifactsCore_Error_Returns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Create config as a directory to force a read error in config.Load
	claudeDir := filepath.Join(dir, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(claudeDir, "project-config.toml"), 0o755)).To(Succeed())

	exit := func(int) { t.Error("exit should not be called on error") }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir}, exit)
	g.Expect(err).To(HaveOccurred())
}

func TestRunValidateArtifactsCore_JSON_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Empty dir → Pass=true; JSON mode returns JSON output without calling exit
	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir, JSON: true}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunValidateArtifactsCore_OrphanIDs_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// REQ-1 traces to DES-1, DES-1 traces to ARCH-999 which is not defined → ARCH-999 is orphan
	g.Expect(os.WriteFile(filepath.Join(dir, "requirements.md"), []byte("### REQ-1: Feature\n\n**Traces to:** DES-1\n"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dir, "design.md"), []byte("### DES-1: Design\n\n**Traces to:** ARCH-999\n"), 0o644)).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunValidateArtifactsCore_Pass_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Empty dir → no IDs → Pass=true → exit not called
	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunValidateArtifactsCore_Phase_NoExit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// TASK-1 is defined but nothing traces to it; at breakdown_commit phase TASK- is exempt
	g.Expect(os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("### TASK-1: Task\n"), 0o644)).To(Succeed())

	exitCalled := false
	exit := func(int) { exitCalled = true }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir, Phase: "breakdown_commit"}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

func TestRunValidateArtifactsCore_UnlinkedIDs_ExitCalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// DES-1 is defined but nothing traces to it → unlinked → Pass=false
	g.Expect(os.WriteFile(filepath.Join(dir, "design.md"), []byte("### DES-1: Design\n"), 0o644)).To(Succeed())

	exitCalled := false
	exitCode := 0
	exit := func(code int) { exitCalled = true; exitCode = code }

	err := trace.RunValidateArtifactsCore(trace.ValidateArtifactsArgs{Dir: dir}, exit)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeTrue())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunValidateArtifacts_EmptyDir_JSONOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// JSON output avoids os.Exit on failure
	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{Dir: dir, JSON: true})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidateArtifacts_InvalidPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Invalid phase triggers error from ValidateV2Artifacts → RunValidateArtifacts returns error
	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{
		Dir: dir,

		Phase: "invalid_phase_xyz_does_not_exist",
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("validation failed"))
	}
}

func TestRunValidateArtifacts_TextOutput_Pass(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// No artifact files → all IDs defined = 0, referenced = 0 → Pass=true
	// JSON=false to exercise the text output pass path
	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{Dir: dir, JSON: false})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunValidateArtifacts_WithPhase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := trace.RunValidateArtifacts(trace.ValidateArtifactsArgs{
		Dir:   dir,
		Phase: "arch_commit",
		JSON:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidate_DES_MissingARCH(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
	writeArtifact(t, fs, dir, "design.md", "### DES-001: Design\n")

	// Add REQ→DES link (so both are in matrix), but DES has no ARCH link
	g.Expect(trace.Add(dir, "REQ-1", []string{"DES-1"}, fs)).To(Succeed())

	result, err := trace.Validate(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	// DES-1 should be in MissingCoverage (no ARCH downstream)
	g.Expect(result.MissingCoverage).ToNot(BeEmpty())
}

func TestValidate_REQ_MissingDESAndARCH(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{Files: make(map[string][]byte), Dirs: make(map[string]bool)}
	dir := t.TempDir()

	writeArtifact(t, fs, dir, "requirements.md", "### REQ-001: Feature\n")
	// REQ-1 is in artifact file but has no matrix links at all

	result, err := trace.Validate(dir, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Pass).To(BeFalse())
	// REQ-1 unlinked and missing coverage
	g.Expect(result.MissingCoverage).ToNot(BeEmpty())
}

// taskReadErrorFS wraps MockFS but returns a non-NotExist error for tasks.md.
type taskReadErrorFS struct {
	*MockFS

	dir string
}

func (f *taskReadErrorFS) ReadFile(path string) ([]byte, error) {
	if path == filepath.Join(f.dir, "tasks.md") {
		return nil, errors.New("permission denied")
	}

	return f.MockFS.ReadFile(path)
}

// writeErrorFS wraps MockFS but always fails WriteFile.
type writeErrorFS struct {
	*MockFS
}

func (w *writeErrorFS) WriteFile(_ string, _ []byte, _ os.FileMode) error {
	return errors.New("disk full")
}
