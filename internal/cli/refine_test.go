package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestFindAllTranscripts_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()

	// Create a project dir with no .jsonl files.
	g.Expect(os.MkdirAll(filepath.Join(projectsDir, "proj1"), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(projectsDir, "proj1", "readme.txt"), []byte("hi"), 0o644)).To(Succeed())

	result, err := cli.ExportFindAllTranscripts(projectsDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestFindAllTranscripts_NonExistent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result, err := cli.ExportFindAllTranscripts("/nonexistent/path")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeNil())
}

func TestFindAllTranscripts_WithFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()
	projDir := filepath.Join(projectsDir, "proj1")
	g.Expect(os.MkdirAll(projDir, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(projDir, "s1.jsonl"), []byte("{}"), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(projDir, "s2.jsonl"), []byte("{}"), 0o644)).To(Succeed())

	result, err := cli.ExportFindAllTranscripts(projectsDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
}

func TestFindTranscriptForMemory_MatchesClosest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()

	// Create three transcript files with different mtimes.
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	transcripts := make([]string, 3)

	for i, offset := range []time.Duration{-10 * time.Hour, -2 * time.Hour, -36 * time.Hour} {
		path := filepath.Join(projectsDir, "project1", "session"+string(rune('A'+i))+".jsonl")
		g.Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		g.Expect(os.WriteFile(path, []byte(`{}`), 0o644)).To(Succeed())

		mtime := base.Add(offset)
		g.Expect(os.Chtimes(path, mtime, mtime)).To(Succeed())

		transcripts[i] = path
	}

	// Memory created at base time — closest should be the 2h before file.
	record := memory.MemoryRecord{
		CreatedAt: base.Format(time.RFC3339),
	}

	result := cli.ExportFindTranscriptForMemory(record, transcripts)
	g.Expect(result).NotTo(BeEmpty(), "should find a matching transcript")

	// The 2h-before file (index 1) should be closest.
	g.Expect(result).To(Equal(transcripts[1]))
}

func TestFindTranscriptForMemory_NoMatchBeyond24h(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()

	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// All transcripts are more than 24h away.
	path := filepath.Join(projectsDir, "project1", "session.jsonl")
	g.Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(path, []byte(`{}`), 0o644)).To(Succeed())

	farMtime := base.Add(-48 * time.Hour)
	g.Expect(os.Chtimes(path, farMtime, farMtime)).To(Succeed())

	record := memory.MemoryRecord{
		CreatedAt: base.Format(time.RFC3339),
	}

	result := cli.ExportFindTranscriptForMemory(record, []string{path})
	g.Expect(result).To(BeEmpty(), "should return empty string when all transcripts >24h away")
}

func TestRunRefineWith_ExtractionError_Skips(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(dataDir, "policy.toml"), []byte(""), 0o644)).To(Succeed())

	now := time.Now().UTC()
	memTOML := fmt.Sprintf(`situation = "test"
behavior = "test"
impact = "test"
action = "test action"
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "err-mem.toml"), []byte(memTOML), 0o644)).To(Succeed())

	home, homeErr := os.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())

	if homeErr != nil {
		return
	}

	testProjectDir := filepath.Join(home, ".claude", "projects", "refine-err-test")
	g.Expect(os.MkdirAll(testProjectDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(testProjectDir, "err-session.jsonl")
	g.Expect(os.WriteFile(transcriptPath,
		[]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"ctx"}]}}`+"\n"),
		0o644)).To(Succeed())
	g.Expect(os.Chtimes(transcriptPath, now, now)).To(Succeed())

	t.Cleanup(func() { _ = os.RemoveAll(testProjectDir) })

	// Mock caller returns invalid JSON → extraction error → skip.
	failCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "not json", nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunRefineWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		failCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("0 refined, 1 skipped (of 1)"))
}

func TestRunRefineWith_SuccessfulExtraction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write valid policy.
	g.Expect(os.WriteFile(filepath.Join(dataDir, "policy.toml"), []byte(""), 0o644)).To(Succeed())

	// Use a distinct timestamp offset from other tests to avoid transcript collision
	// when findTranscriptForMemory scans all projects.
	now := time.Now().UTC().Add(-2 * time.Hour)
	memTOML := fmt.Sprintf(`situation = "old situation\nKeywords: foo, bar"
behavior = "old behavior"
impact = "old impact"
action = "old action"
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "test-mem.toml"), []byte(memTOML), 0o644)).To(Succeed())

	// Create transcript with matching mtime.
	home, homeErr := os.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())

	if homeErr != nil {
		return
	}

	testProjectDir := filepath.Join(home, ".claude", "projects", "refine-extract-test")
	g.Expect(os.MkdirAll(testProjectDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(testProjectDir, "extract-session.jsonl")
	g.Expect(os.WriteFile(transcriptPath,
		[]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"context"}]}}`+"\n"),
		0o644)).To(Succeed())
	g.Expect(os.Chtimes(transcriptPath, now, now)).To(Succeed())

	t.Cleanup(func() { _ = os.RemoveAll(testProjectDir) })

	// Mock caller returns valid SBIA extraction JSON.
	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return `{"situation":"new situation","behavior":"new behavior","impact":"new impact",` +
			`"action":"new action","filename_slug":"test-mem"}`, nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunRefineWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("1 refined, 0 skipped"))

	// Verify the memory was updated.
	data, readErr := os.ReadFile(filepath.Join(memoriesDir, "test-mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	content := string(data)
	g.Expect(content).To(ContainSubstring("new situation"))
	g.Expect(content).To(ContainSubstring("new action"))
}

func TestRunRefine_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memTOML := `situation = "when writing Go code"
behavior = "returning bare errors"
impact = "hard to debug"
action = "wrap errors with context"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "wrap-errors.toml"),
		[]byte(memTOML),
		0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "refine",
			"--data-dir", dataDir,
			"--dry-run",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// In dry-run mode, the file must not be modified.
	original, readErr := os.ReadFile(filepath.Join(memoriesDir, "wrap-errors.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(original)).To(ContainSubstring("wrap errors with context"),
		"dry-run must not modify memory files")

	// Output should report what it found.
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("refine"),
		"output should mention refine operation")
}

func TestRunRefine_DryRunWithMatchingTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write valid policy.
	g.Expect(os.WriteFile(filepath.Join(dataDir, "policy.toml"), []byte(""), 0o644)).To(Succeed())

	// Use a distinct timestamp offset from other tests to avoid transcript collision.
	now := time.Now().UTC().Add(-4 * time.Hour)
	memTOML := fmt.Sprintf(`situation = "test\nKeywords: baz"
behavior = "test"
impact = "test"
action = "test action"
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "dryrun-match.toml"), []byte(memTOML), 0o644)).To(Succeed())

	// Create transcript with matching mtime.
	home, homeErr := os.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())

	if homeErr != nil {
		return
	}

	testProjectDir := filepath.Join(home, ".claude", "projects", "refine-dryrun-test")
	g.Expect(os.MkdirAll(testProjectDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(testProjectDir, "dryrun-session.jsonl")
	g.Expect(os.WriteFile(transcriptPath,
		[]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`+"\n"),
		0o644)).To(Succeed())
	g.Expect(os.Chtimes(transcriptPath, now, now)).To(Succeed())

	t.Cleanup(func() { _ = os.RemoveAll(testProjectDir) })

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir, "--dry-run"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("1 refined, 0 skipped"))
}

func TestRunRefine_EmptyMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("0 refined, 0 skipped"))
}

func TestRunRefine_ListError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	// Create a file where memories dir should be, causing a non-NotExist error.
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "memories"),
		[]byte("not a dir"),
		0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("listing memories"))
	}
}

func TestRunRefine_MemoryWithEmptyAction_Skipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write policy.toml (needed by runRefine after listing memories).
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "policy.toml"),
		[]byte(""),
		0o644,
	)).To(Succeed())

	// Memory with empty action — should be skipped in the refine loop.
	// The created_at time must match a transcript so we get past the findTranscriptForMemory check.
	now := time.Now().UTC()
	memTOML := fmt.Sprintf(`situation = "test"
behavior = "test"
impact = "test"
action = ""
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))

	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "empty-action.toml"),
		[]byte(memTOML),
		0o644,
	)).To(Succeed())

	// Create a transcript file with matching mtime (within 24h of the memory).
	home, homeErr := os.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())

	if homeErr != nil {
		return
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	testProjectDir := filepath.Join(projectsDir, "refine-test-project")
	g.Expect(os.MkdirAll(testProjectDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(testProjectDir, "refine-test-session.jsonl")
	transcriptContent := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello"}]}}` + "\n"
	g.Expect(os.WriteFile(transcriptPath, []byte(transcriptContent), 0o644)).To(Succeed())

	// Set mtime close to the memory created_at.
	g.Expect(os.Chtimes(transcriptPath, now, now)).To(Succeed())

	// Cleanup
	t.Cleanup(func() {
		_ = os.Remove(transcriptPath)
		_ = os.Remove(testProjectDir)
	})

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Memory had empty action, should be skipped.
	g.Expect(stdout.String()).To(ContainSubstring("0 refined, 1 skipped (of 1)"))
}

func TestRunRefine_NoMemoriesDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("no memories found"))
}

func TestRunRefine_NoTranscriptMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memTOML := `situation = "test"
behavior = "test"
impact = "test"
action = "test action"
created_at = "2020-01-01T00:00:00Z"
updated_at = "2020-01-01T00:00:00Z"
`
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "old-mem.toml"),
		[]byte(memTOML),
		0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("0 refined, 1 skipped (of 1)"))
}

func TestRunRefine_ParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("refine"))
	}
}

func TestRunRefine_PolicyLoadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	now := time.Now().UTC()
	memTOML := fmt.Sprintf(`situation = "test"
behavior = "test"
impact = "test"
action = "test action"
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "test.toml"), []byte(memTOML), 0o644)).To(Succeed())

	// Create invalid policy.toml.
	g.Expect(os.WriteFile(filepath.Join(dataDir, "policy.toml"), []byte("{{invalid toml"), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "refine", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("loading policy"))
	}
}

func TestRunRefine_TranscriptReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write valid policy.
	g.Expect(os.WriteFile(filepath.Join(dataDir, "policy.toml"), []byte(""), 0o644)).To(Succeed())

	// Use a distinct timestamp offset from other tests to avoid transcript collision.
	now := time.Now().UTC().Add(-6 * time.Hour)
	memTOML := fmt.Sprintf(`situation = "test"
behavior = "test"
impact = "test"
action = "test action"
created_at = "%s"
updated_at = "%s"
`, now.Format(time.RFC3339), now.Format(time.RFC3339))
	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "test.toml"), []byte(memTOML), 0o644)).To(Succeed())

	// Create transcript in ~/.claude/projects/ with matching mtime.
	home, homeErr := os.UserHomeDir()
	g.Expect(homeErr).NotTo(HaveOccurred())

	if homeErr != nil {
		return
	}

	testProjectDir := filepath.Join(home, ".claude", "projects", "refine-transcript-err")
	g.Expect(os.MkdirAll(testProjectDir, 0o755)).To(Succeed())

	// Create a transcript file with no read permission (causes read error).
	transcriptPath := filepath.Join(testProjectDir, "bad-session.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte("data"), 0o000)).To(Succeed())

	g.Expect(os.Chtimes(transcriptPath, now, now)).To(Succeed())

	t.Cleanup(func() {
		_ = os.RemoveAll(testProjectDir)
	})

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return `{"situation":"s","behavior":"b","impact":"i","action":"a"}`, nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunRefineWith(
		[]string{"--data-dir", dataDir},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Transcript is a directory (not a file), read should fail, memory should be skipped.
	g.Expect(stdout.String()).To(ContainSubstring("0 refined, 1 skipped (of 1)"))
}
