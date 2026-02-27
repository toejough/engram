package usage_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/log"
	"github.com/toejough/projctl/internal/usage"
)

// TestRunCheckCore_StatusLimit verifies RunCheckCore calls exit(2) when token limit is exceeded.
func TestRunCheckCore_StatusLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	err := os.MkdirAll(claudeDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	configContent := "[budget]\nwarning_tokens = 0\nlimit_tokens = 1\n"
	err = os.WriteFile(filepath.Join(claudeDir, "project-config.toml"), []byte(configContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "x",
	})
	g.Expect(err).ToNot(HaveOccurred())

	var exitCode int

	err = usage.RunCheckCore(usage.CheckArgs{Dir: dir}, func(code int) { exitCode = code })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCode).To(Equal(2))
}

// TestRunCheckCore_StatusOK verifies RunCheckCore does not call exit when usage is within budget.
func TestRunCheckCore_StatusOK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	exitCalled := false

	err := usage.RunCheckCore(usage.CheckArgs{Dir: dir}, func(_ int) { exitCalled = true })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCalled).To(BeFalse())
}

// TestRunCheckCore_StatusWarning verifies RunCheckCore calls exit(1) when warning threshold is exceeded.
func TestRunCheckCore_StatusWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	err := os.MkdirAll(claudeDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	configContent := "[budget]\nwarning_tokens = 1\nlimit_tokens = 100\n"
	err = os.WriteFile(filepath.Join(claudeDir, "project-config.toml"), []byte(configContent), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "x",
	})
	g.Expect(err).ToNot(HaveOccurred())

	var exitCode int

	err = usage.RunCheckCore(usage.CheckArgs{Dir: dir}, func(code int) { exitCode = code })
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exitCode).To(Equal(1))
}

func TestRunCheck_OK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := usage.RunCheck(usage.CheckArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_JSONFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := usage.RunReport(usage.ReportArgs{Dir: dir, Format: "json"})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunReport_NeitherDirNorProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := usage.RunReport(usage.ReportArgs{})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("either --dir or --project is required"))
	}
}

func TestRunReport_Project(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := usage.RunReport(usage.ReportArgs{Project: "nonexistent-project-for-testing"})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("project not found"))
	}
}

func TestRunReport_WithDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := usage.RunReport(usage.ReportArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunReport_WithModels verifies RunReport prints per-model breakdown when entries have models.
func TestRunReport_WithModels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	err := log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "test with model",
		Model:   "haiku",
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = log.RunWrite(log.WriteArgs{
		Dir:     dir,
		Level:   "status",
		Subject: "task-status",
		Message: "test without model",
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = usage.RunReport(usage.ReportArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}
