package corrections_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/corrections"
)

// TestAnalyzeGlobal_FindsPatterns verifies AnalyzeGlobal detects repeated corrections.
func TestAnalyzeGlobal_FindsPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	_ = corrections.LogGlobal("amend push commit", "ctx1", corrections.LogOpts{}, "testhome", nowFunc(), fs)
	_ = corrections.LogGlobal("amend push commit", "ctx2", corrections.LogOpts{}, "testhome", nowFunc(), fs)

	patterns, err := corrections.AnalyzeGlobal("testhome", corrections.AnalyzeOpts{MinOccurrences: 2}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(HaveLen(1))
}

// TestAnalyzeGlobal_NoCorrections verifies AnalyzeGlobal returns empty when no file.
func TestAnalyzeGlobal_NoCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	patterns, err := corrections.AnalyzeGlobal("testhome", corrections.AnalyzeOpts{}, fs)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(patterns).To(BeEmpty())
}

// TestLog_NilNow verifies Log with nil now uses time.Now.
func TestLog_NilNow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MockFS{}

	err := corrections.Log("testdir", "nil now correction", "context", corrections.LogOpts{}, nil, fs)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := fs.ReadFile(filepath.Join("testdir", "corrections.jsonl"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).ToNot(BeNil())

	var entry corrections.Entry

	if len(content) > 0 {
		err = json.Unmarshal(content[:len(content)-1], &entry)
		g.Expect(err).ToNot(HaveOccurred())
	}

	g.Expect(entry.Message).To(Equal("nil now correction"))
	g.Expect(entry.Timestamp).ToNot(BeEmpty())
}

// TestRealFS_AppendFile creates and appends to a real file.
func TestRealFS_AppendFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "append.txt")

	fs := corrections.RealFS{}

	g.Expect(fs.AppendFile(path, []byte("hello"))).To(Succeed())
	g.Expect(fs.AppendFile(path, []byte("world"))).To(Succeed())

	content, err := os.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(Equal("helloworld"))
}

// TestRealFS_FileExists_Corrections checks existing/non-existing files.
func TestRealFS_FileExists_Corrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	fs := corrections.RealFS{}

	g.Expect(fs.FileExists(path)).To(BeFalse())

	g.Expect(os.WriteFile(path, []byte("{}"), 0o644)).To(Succeed())
	g.Expect(fs.FileExists(path)).To(BeTrue())
}

// TestRealFS_MkdirAll creates a nested directory.
func TestRealFS_MkdirAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	fs := corrections.RealFS{}

	g.Expect(fs.MkdirAll(nested)).To(Succeed())

	info, err := os.Stat(nested)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(info).ToNot(BeNil())

	if info != nil {
		g.Expect(info.IsDir()).To(BeTrue())
	}
}

// TestRealFS_ReadFile_Corrections reads a real file.
func TestRealFS_ReadFile_Corrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	g.Expect(os.WriteFile(path, []byte("data"), 0o644)).To(Succeed())

	fs := corrections.RealFS{}

	data, err := fs.ReadFile(path)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(data)).To(Equal("data"))
}

// TestRunAnalyze_GlobalPath tests RunAnalyze using global home dir.
func TestRunAnalyze_GlobalPath(t *testing.T) {
	// No t.Parallel(): uses t.Setenv which is incompatible with t.Parallel()
	g := NewWithT(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	claudeDir := filepath.Join(tmpHome, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())

	path := filepath.Join(claudeDir, "corrections.jsonl")
	line1 := `{"timestamp":"2026-02-01T10:00:00Z","message":"pattern message","context":"ctx1"}` + "\n"
	line2 := `{"timestamp":"2026-02-01T11:00:00Z","message":"pattern message","context":"ctx2"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line1+line2), 0o644)).To(Succeed())

	err := corrections.RunAnalyze(corrections.AnalyzeArgs{MinOccurrences: 2})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunAnalyze_NoPatterns tests RunAnalyze with no matching patterns.
func TestRunAnalyze_NoPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	line := `{"timestamp":"2026-02-01T10:00:00Z","message":"unique correction","context":"ctx1"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line), 0o644)).To(Succeed())

	err := corrections.RunAnalyze(corrections.AnalyzeArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunAnalyze_WithPatterns tests RunAnalyze finding patterns.
func TestRunAnalyze_WithPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	line1 := `{"timestamp":"2026-02-01T10:00:00Z","message":"same pattern message","context":"ctx1"}` + "\n"
	line2 := `{"timestamp":"2026-02-01T11:00:00Z","message":"same pattern message","context":"ctx2"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line1+line2), 0o644)).To(Succeed())

	err := corrections.RunAnalyze(corrections.AnalyzeArgs{Dir: dir, MinOccurrences: 2})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunCount_GlobalPath tests RunCount with no dir (global path).
func TestRunCount_GlobalPath(t *testing.T) {
	// No t.Parallel(): uses t.Setenv which is incompatible with t.Parallel()
	g := NewWithT(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	claudeDir := filepath.Join(tmpHome, ".claude")
	g.Expect(os.MkdirAll(claudeDir, 0o755)).To(Succeed())

	path := filepath.Join(claudeDir, "corrections.jsonl")
	g.Expect(os.WriteFile(path, []byte(`{"timestamp":"2026-01-01T00:00:00Z","message":"m","context":"c"}`+"\n"), 0o644)).To(Succeed())

	err := corrections.RunCount(corrections.CountArgs{})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunCount_WithDir tests RunCount with dir provided.
func TestRunCount_WithDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	line1 := `{"timestamp":"2026-02-01T10:00:00Z","message":"first","context":"ctx1","session_id":"sess-1"}` + "\n"
	line2 := `{"timestamp":"2026-02-01T11:00:00Z","message":"second","context":"ctx2","session_id":"sess-2"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line1+line2), 0o644)).To(Succeed())

	err := corrections.RunCount(corrections.CountArgs{Dir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunCount_WithSession filters by session.
func TestRunCount_WithSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	line1 := `{"timestamp":"2026-02-01T10:00:00Z","message":"first","context":"ctx1","session_id":"sess-1"}` + "\n"
	line2 := `{"timestamp":"2026-02-01T11:00:00Z","message":"second","context":"ctx2","session_id":"sess-2"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line1+line2), 0o644)).To(Succeed())

	err := corrections.RunCount(corrections.CountArgs{Dir: dir, Session: "sess-1"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunCount_WithSince filters by timestamp.
func TestRunCount_WithSince(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "corrections.jsonl")

	line1 := `{"timestamp":"2026-02-01T10:00:00Z","message":"early","context":"ctx1"}` + "\n"
	line2 := `{"timestamp":"2026-02-01T12:00:00Z","message":"late","context":"ctx2"}` + "\n"
	g.Expect(os.WriteFile(path, []byte(line1+line2), 0o644)).To(Succeed())

	err := corrections.RunCount(corrections.CountArgs{Dir: dir, Since: "2026-02-01T11:00:00Z"})
	g.Expect(err).ToNot(HaveOccurred())
}

// TestRunLog_GlobalPath tests RunLog writing to the global path.
func TestRunLog_GlobalPath(t *testing.T) {
	// No t.Parallel(): uses t.Setenv which is incompatible with t.Parallel()
	g := NewWithT(t)
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	err := corrections.RunLog(corrections.LogArgs{
		Message: "global correction",
		Context: "global context",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(filepath.Join(tmpHome, ".claude", "corrections.jsonl")).To(BeAnExistingFile())
}

// TestRunLog_WithDir tests RunLog writing to a specific directory.
func TestRunLog_WithDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := corrections.RunLog(corrections.LogArgs{
		Dir:     dir,
		Message: "test correction message",
		Context: "test context",
		Session: "sess-001",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(filepath.Join(dir, "corrections.jsonl")).To(BeAnExistingFile())
}
