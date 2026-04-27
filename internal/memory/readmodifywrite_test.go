package memory_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

func TestListAll_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(BeEmpty())
}

func TestListAll_ReadsAllTOMLFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	for _, name := range []string{"a.toml", "b.toml"} {
		rec := memory.MemoryRecord{Situation: name}

		var buf bytes.Buffer

		_ = toml.NewEncoder(&buf).Encode(rec)
		_ = os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644)
	}

	// Write a non-TOML file (should be skipped)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("skip"), 0o644)

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(2))
}

func TestListAll_SkipsInvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Write an unparseable TOML file — should be silently skipped
	_ = os.WriteFile(filepath.Join(dir, "bad.toml"), []byte("not = [valid toml"), 0o644)

	// Write one valid TOML file
	rec := memory.MemoryRecord{Situation: "valid situation"}

	var buf bytes.Buffer

	_ = toml.NewEncoder(&buf).Encode(rec)
	_ = os.WriteFile(filepath.Join(dir, "valid.toml"), buf.Bytes(), 0o644)

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Situation).To(Equal("valid situation"))
}

func TestListAll_SkipsSubdirectories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")

	err := os.Mkdir(subdir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Write one valid TOML file
	rec := memory.MemoryRecord{Situation: "valid situation"}

	var buf bytes.Buffer

	_ = toml.NewEncoder(&buf).Encode(rec)
	_ = os.WriteFile(filepath.Join(dir, "valid.toml"), buf.Bytes(), 0o644)

	// Write a non-TOML file
	_ = os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("skip"), 0o644)

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Situation).To(Equal("valid situation"))
}

func TestLister_ListAllMemories_EmptyFeedbackUsesLegacy(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create empty feedback dir and populated legacy dir.
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	legacyDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(legacyDir, 0o750)).To(Succeed())

	legacyContent := `situation = "legacy mem"
updated_at = "2024-01-01T00:00:00Z"

[content]
action = "do something"
`

	writeErr := os.WriteFile(
		filepath.Join(legacyDir, "old.toml"),
		[]byte(legacyContent), 0o644,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	stored, err := lister.ListAllMemories(dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stored).To(HaveLen(1))

	if len(stored) < 1 { // nilaway guard
		return
	}

	g.Expect(stored[0].Situation).To(Equal("legacy mem"))
}

func TestLister_ListAllMemories_LegacyFallback(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create only legacy directory.
	legacyDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(legacyDir, 0o750)).To(Succeed())

	legacyContent := `situation = "legacy mem"
updated_at = "2024-01-01T00:00:00Z"

[content]
action = "do something"
`

	writeErr := os.WriteFile(
		filepath.Join(legacyDir, "old.toml"),
		[]byte(legacyContent), 0o644,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	stored, err := lister.ListAllMemories(dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stored).To(HaveLen(1))

	if len(stored) < 1 { // nilaway guard
		return
	}

	g.Expect(stored[0].Situation).To(Equal("legacy mem"))
}

func TestLister_ListAllMemories_NewLayout(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create new layout directories with files.
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	factsDir := filepath.Join(dataDir, "memory", "facts")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())

	feedbackContent := `type = "feedback"
situation = "feedback mem"
updated_at = "2024-06-01T00:00:00Z"

[content]
behavior = "test"
`
	factContent := `type = "fact"
situation = "fact context"
updated_at = "2024-07-01T00:00:00Z"

[content]
subject = "project"
predicate = "uses"
object = "Go"
`

	writeErr := os.WriteFile(
		filepath.Join(feedbackDir, "fb.toml"),
		[]byte(feedbackContent), 0o644,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	writeErr = os.WriteFile(
		filepath.Join(factsDir, "fact.toml"),
		[]byte(factContent), 0o644,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	stored, err := lister.ListAllMemories(dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stored).To(HaveLen(2))

	if len(stored) < 2 { // nilaway guard
		return
	}

	// Sorted by UpdatedAt descending: fact (July) before feedback (June).
	g.Expect(stored[0].Type).To(Equal("fact"))
	g.Expect(stored[1].Type).To(Equal("feedback"))
}

func TestLister_ListAllMemories_NoDirs_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	// No directories at all - legacy path doesn't exist.

	lister := memory.NewLister()
	_, err := lister.ListAllMemories(dataDir)
	g.Expect(err).To(HaveOccurred())
}

func TestLister_ListAll_UsesDI(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	content := `situation = "test situation"

[content]
action = "test action"
`

	writeErr := os.WriteFile(filepath.Join(dir, "mem1.toml"), []byte(content), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	records, err := lister.ListAll(dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Situation).To(Equal("test situation"))
	g.Expect(records[0].Path).To(HaveSuffix("mem1.toml"))
}

func TestLister_ListStored_ReturnsSortedStored(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	older := `situation = "older memory"
updated_at = "2024-01-01T00:00:00Z"
`
	newer := `situation = "newer memory"
updated_at = "2024-06-15T00:00:00Z"
`

	writeErr := os.WriteFile(filepath.Join(dir, "old.toml"), []byte(older), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	writeErr = os.WriteFile(filepath.Join(dir, "new.toml"), []byte(newer), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister()
	stored, err := lister.ListStored(dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stored).To(HaveLen(2))
	g.Expect(stored[0].Situation).To(Equal("newer memory"))
	g.Expect(stored[1].Situation).To(Equal("older memory"))
}

func TestLister_WithReadDir_InjectsDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	content := `situation = "injected"
`
	called := false
	fakeReadDir := func(_ string) ([]os.DirEntry, error) {
		called = true

		return os.ReadDir(t.TempDir())
	}

	dir := t.TempDir()
	writeErr := os.WriteFile(filepath.Join(dir, "mem.toml"), []byte(content), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	lister := memory.NewLister(memory.WithListerReadDir(fakeReadDir))
	_, err := lister.ListAll(dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(called).To(BeTrue())
}

func TestLister_WithReadFile_InjectsFileRead(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	writeErr := os.WriteFile(filepath.Join(dir, "mem.toml"), []byte(""), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	called := false
	fakeReadFile := func(path string) ([]byte, error) {
		called = true

		return os.ReadFile(path)
	}

	lister := memory.NewLister(memory.WithListerReadFile(fakeReadFile))
	_, err := lister.ListAll(dir)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(called).To(BeTrue())
}

// TestModifier_CleansUpTempOnFailure verifies cleanup on rename failure.
func TestModifier_CleansUpTempOnFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{Situation: "cleanup-test"}

	var buf bytes.Buffer

	encErr := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(encErr).NotTo(HaveOccurred())

	if encErr != nil {
		return
	}

	err := os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	removeCalled := false

	modifier := memory.NewModifier(tomlwriter.New(
		tomlwriter.WithRename(func(_, _ string) error {
			return errors.New("rename failed")
		}),
		tomlwriter.WithRemove(func(_ string) error {
			removeCalled = true

			return nil
		}),
	))

	writeErr := modifier.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(writeErr).To(HaveOccurred())
	g.Expect(removeCalled).To(BeTrue())
}

// TestModifier_ReadFileError verifies that a read error from injected readFile propagates.
func TestModifier_ReadFileError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := memory.NewModifier(
		tomlwriter.New(),
		memory.WithModifierReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("injected read error")
		}),
	)

	err := modifier.ReadModifyWrite("/fake/path.toml", func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("injected read error"))
	}
}

// TestModifier_WithDI verifies that the Modifier struct works with injected dependencies.
func TestModifier_WithDI(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{Situation: "di-test", Source: "original"}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	modifier := memory.NewModifier(tomlwriter.New())

	err = modifier.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var result memory.MemoryRecord

	_, decErr := toml.Decode(string(data), &result)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(result.Source).To(Equal("mutated"))
	g.Expect(result.Situation).To(Equal("di-test"))
}

// TestModifier_WriterError verifies that a writer error propagates.
func TestModifier_WriterError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	initial := memory.MemoryRecord{Situation: "writer-error-test"}

	var buf bytes.Buffer

	encErr := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(encErr).NotTo(HaveOccurred())

	if encErr != nil {
		return
	}

	tomlData := buf.Bytes()

	modifier := memory.NewModifier(
		tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, _ string) (*os.File, error) {
				return nil, errors.New("injected create error")
			}),
		),
		memory.WithModifierReadFile(func(_ string) ([]byte, error) {
			return tomlData, nil
		}),
	)

	err := modifier.ReadModifyWrite("/fake/path.toml", func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("injected create error"))
	}
}

func TestReadModifyWrite_DecodeError_InjectedInvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := memory.NewModifier(
		tomlwriter.New(),
		memory.WithModifierReadFile(func(_ string) ([]byte, error) {
			return []byte("not = [valid toml"), nil
		}),
	)

	err := modifier.ReadModifyWrite("/fake/path.toml", func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("decoding"))
	}
}

func TestReadModifyWrite_InvalidTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")

	err := os.WriteFile(path, []byte("not = [valid toml"), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	modifier := memory.NewModifier(tomlwriter.New())

	err = modifier.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).To(HaveOccurred())
}

func TestReadModifyWrite_MutateIsCalledWithDecodedRecord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	initial := memory.MemoryRecord{Situation: "mutate-test", Source: "original"}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlData := buf.Bytes()

	var capturedRecord *memory.MemoryRecord

	modifier := memory.NewModifier(
		tomlwriter.New(
			tomlwriter.WithCreateTemp(func(_, _ string) (*os.File, error) {
				return nil, errors.New("stop after mutate")
			}),
		),
		memory.WithModifierReadFile(func(_ string) ([]byte, error) {
			return tomlData, nil
		}),
	)

	_ = modifier.ReadModifyWrite("/fake/path.toml", func(r *memory.MemoryRecord) {
		capturedRecord = r
		r.Source = "mutated"
	})

	g.Expect(capturedRecord).NotTo(BeNil())

	if capturedRecord == nil {
		return
	}

	g.Expect(capturedRecord.Source).To(Equal("mutated"))
	g.Expect(capturedRecord.Situation).To(Equal("mutate-test"))
}

func TestReadModifyWrite_MutatesField(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{Situation: "test", Source: "original"}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	modifier := memory.NewModifier(tomlwriter.New())

	err = modifier.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result memory.MemoryRecord

	_, err = toml.Decode(string(data), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Source).To(Equal("mutated"))
	g.Expect(result.Situation).To(Equal("test"))
}

func TestReadModifyWrite_NonexistentFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	modifier := memory.NewModifier(tomlwriter.New())

	err := modifier.ReadModifyWrite("/nonexistent/path/test.toml", func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).To(HaveOccurred())
}

func TestReadModifyWrite_PreservesAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{
		Type:      "feedback",
		Source:    "observation",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage",
			Action:   "use targ test",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(initial)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(path, buf.Bytes(), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	modifier := memory.NewModifier(tomlwriter.New())

	err = modifier.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.Source = "mutated"
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var result memory.MemoryRecord

	_, err = toml.Decode(string(data), &result)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Source).To(Equal("mutated"))
	g.Expect(result.Situation).To(Equal("when running tests"))
	g.Expect(result.Content.Behavior).To(Equal("use go test directly"))
	g.Expect(result.Content.Impact).To(Equal("misses coverage"))
	g.Expect(result.Content.Action).To(Equal("use targ test"))
	g.Expect(result.CreatedAt).To(Equal("2026-01-01T00:00:00Z"))
	g.Expect(result.UpdatedAt).To(Equal("2026-01-02T00:00:00Z"))
}
