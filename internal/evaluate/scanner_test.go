package evaluate_test

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/evaluate"
	"engram/internal/memory"
)

// TestNewFileScanner_DifferentSession verifies a memory with a different session ID is not returned.
func TestNewFileScanner_DifferentSession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	record := memory.MemoryRecord{
		Situation: "different session",
		PendingEvaluations: []memory.PendingEvaluation{
			{
				SessionID:  "other-session",
				SurfacedAt: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(_ string) ([]byte, error) {
			return tomlBytes(t, record), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{fakeDirEntry{name: "mem1.toml"}}, nil
		},
	)

	results, err := scanner("session-xyz")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

// TestNewFileScanner_EmptyDirectory verifies an empty directory returns empty slice with no error.
func TestNewFileScanner_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scanner := evaluate.NewFileScanner(
		"/data",
		func(_ string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{}, nil
		},
	)

	results, err := scanner("any-session")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

// TestNewFileScanner_FileReadError verifies a file read error causes the file to be skipped.
func TestNewFileScanner_FileReadError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const targetSession = "session-read-err"

	goodRecord := memory.MemoryRecord{
		Situation: "good memory",
		PendingEvaluations: []memory.PendingEvaluation{
			{SessionID: targetSession, SurfacedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(name string) ([]byte, error) {
			if name == "/data/memories/bad.toml" {
				return nil, errors.New("read error")
			}

			return tomlBytes(t, goodRecord), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				fakeDirEntry{name: "bad.toml"},
				fakeDirEntry{name: "good.toml"},
			}, nil
		},
	)

	results, err := scanner(targetSession)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal("/data/memories/good.toml"))
}

// TestNewFileScanner_InvalidTOML verifies an invalid TOML file is skipped.
func TestNewFileScanner_InvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const targetSession = "session-invalid-toml"

	goodRecord := memory.MemoryRecord{
		Situation: "good memory",
		PendingEvaluations: []memory.PendingEvaluation{
			{SessionID: targetSession, SurfacedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(name string) ([]byte, error) {
			if name == "/data/memories/invalid.toml" {
				return []byte("not = [valid toml"), nil
			}

			return tomlBytes(t, goodRecord), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				fakeDirEntry{name: "invalid.toml"},
				fakeDirEntry{name: "good.toml"},
			}, nil
		},
	)

	results, err := scanner(targetSession)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal("/data/memories/good.toml"))
}

// TestNewFileScanner_ListDirError verifies a listDir error is propagated.
func TestNewFileScanner_ListDirError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	listErr := errors.New("directory not found")

	scanner := evaluate.NewFileScanner(
		"/data",
		func(_ string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
		func(_ string) ([]fs.DirEntry, error) {
			return nil, listErr
		},
	)

	results, err := scanner("any-session")
	g.Expect(err).To(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TestNewFileScanner_MatchingSession verifies a memory with a matching session ID is returned.
func TestNewFileScanner_MatchingSession(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const sessionID = "session-abc"

	const memPath = "/data/memories/mem1.toml"

	record := memory.MemoryRecord{
		Situation: "test situation",
		PendingEvaluations: []memory.PendingEvaluation{
			{
				SessionID:  sessionID,
				SurfacedAt: time.Now().UTC().Format(time.RFC3339),
				UserPrompt: "help me",
			},
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(name string) ([]byte, error) {
			if name == memPath {
				return tomlBytes(t, record), nil
			}

			return nil, errors.New("unexpected path: " + name)
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{fakeDirEntry{name: "mem1.toml"}}, nil
		},
	)

	results, err := scanner(sessionID)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal(memPath))
	g.Expect(results[0].Eval.SessionID).To(Equal(sessionID))
	g.Expect(results[0].Record).NotTo(BeNil())

	if results[0].Record != nil {
		g.Expect(results[0].Record.Situation).To(Equal("test situation"))
	}
}

// TestNewFileScanner_MultipleMemoriesOnlyOneMatches verifies only the matching memory is returned.
func TestNewFileScanner_MultipleMemoriesOnlyOneMatches(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const targetSession = "target-session"

	records := map[string]memory.MemoryRecord{
		"/data/memories/mem1.toml": {
			Situation: "matching memory",
			PendingEvaluations: []memory.PendingEvaluation{
				{SessionID: targetSession, SurfacedAt: time.Now().UTC().Format(time.RFC3339)},
			},
		},
		"/data/memories/mem2.toml": {
			Situation: "non-matching memory",
			PendingEvaluations: []memory.PendingEvaluation{
				{SessionID: "other-session", SurfacedAt: time.Now().UTC().Format(time.RFC3339)},
			},
		},
		"/data/memories/mem3.toml": {
			Situation:          "no pending memory",
			PendingEvaluations: nil,
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(name string) ([]byte, error) {
			rec, ok := records[name]
			if !ok {
				return nil, errors.New("unexpected path: " + name)
			}

			return tomlBytes(t, rec), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				fakeDirEntry{name: "mem1.toml"},
				fakeDirEntry{name: "mem2.toml"},
				fakeDirEntry{name: "mem3.toml"},
			}, nil
		},
	)

	results, err := scanner(targetSession)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Path).To(Equal("/data/memories/mem1.toml"))
}

// TestNewFileScanner_MultiplePendingEvalsOnOneMemory verifies each matching eval produces a PendingMemory.
func TestNewFileScanner_MultiplePendingEvalsOnOneMemory(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const targetSession = "multi-eval-session"

	record := memory.MemoryRecord{
		Situation: "multi-eval memory",
		PendingEvaluations: []memory.PendingEvaluation{
			{SessionID: targetSession, SurfacedAt: "2024-01-01T00:00:00Z", UserPrompt: "first"},
			{SessionID: "other-session", SurfacedAt: "2024-01-02T00:00:00Z"},
			{SessionID: targetSession, SurfacedAt: "2024-01-03T00:00:00Z", UserPrompt: "second"},
		},
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(_ string) ([]byte, error) {
			return tomlBytes(t, record), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{fakeDirEntry{name: "mem1.toml"}}, nil
		},
	)

	results, err := scanner(targetSession)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	const expectedCount = 2
	g.Expect(results).To(HaveLen(expectedCount))

	prompts := make([]string, 0, expectedCount)
	for _, result := range results {
		prompts = append(prompts, result.Eval.UserPrompt)
	}

	g.Expect(prompts).To(ConsistOf("first", "second"))
}

// TestNewFileScanner_NoPendingEvals verifies a memory with no pending evaluations is not returned.
func TestNewFileScanner_NoPendingEvals(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	record := memory.MemoryRecord{
		Situation:          "no pending",
		PendingEvaluations: nil,
	}

	scanner := evaluate.NewFileScanner(
		"/data",
		func(_ string) ([]byte, error) {
			return tomlBytes(t, record), nil
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{fakeDirEntry{name: "mem1.toml"}}, nil
		},
	)

	results, err := scanner("any-session")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

// TestNewFileScanner_NonTomlFilesIgnored verifies non-.toml files in the directory are ignored.
func TestNewFileScanner_NonTomlFilesIgnored(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	scanner := evaluate.NewFileScanner(
		"/data",
		func(name string) ([]byte, error) {
			return nil, errors.New("should not be called for non-toml: " + name)
		},
		func(_ string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{
				fakeDirEntry{name: "README.md"},
				fakeDirEntry{name: "notes.txt"},
				fakeDirEntry{name: "subdir", isDir: true},
			}, nil
		},
	)

	results, err := scanner("any-session")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(BeEmpty())
}

// unexported variables.
var (
	errFakeFileInfo = errors.New("file info not available in fake dir entry")
)

// fakeDirEntry implements fs.DirEntry for testing.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, errFakeFileInfo }

func (f fakeDirEntry) IsDir() bool { return f.isDir }

func (f fakeDirEntry) Name() string { return f.name }

func (f fakeDirEntry) Type() fs.FileMode { return 0 }

// tomlBytes encodes a MemoryRecord to TOML bytes for use in tests.
func tomlBytes(t *testing.T, record memory.MemoryRecord) []byte {
	t.Helper()

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	if err != nil {
		t.Fatalf("encoding test record to TOML: %v", err)
	}

	return buf.Bytes()
}
