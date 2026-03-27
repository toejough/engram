package memory_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestListAll_EmptyDirectory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	records, err := memory.ListAll(dir)
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
		rec := memory.MemoryRecord{Title: name}

		var buf bytes.Buffer

		_ = toml.NewEncoder(&buf).Encode(rec)
		_ = os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644)
	}

	// Write a non-TOML file (should be skipped)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("skip"), 0o644)

	records, err := memory.ListAll(dir)
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
	rec := memory.MemoryRecord{Title: "valid"}

	var buf bytes.Buffer

	_ = toml.NewEncoder(&buf).Encode(rec)
	_ = os.WriteFile(filepath.Join(dir, "valid.toml"), buf.Bytes(), 0o644)

	records, err := memory.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Title).To(Equal("valid"))
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
	rec := memory.MemoryRecord{Title: "valid"}

	var buf bytes.Buffer

	_ = toml.NewEncoder(&buf).Encode(rec)
	_ = os.WriteFile(filepath.Join(dir, "valid.toml"), buf.Bytes(), 0o644)

	// Write a non-TOML file
	_ = os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("skip"), 0o644)

	records, err := memory.ListAll(dir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Title).To(Equal("valid"))
}

func TestReadModifyWrite_IncrementsField(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{Title: "test", SurfacedCount: 3}

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

	err = memory.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.SurfacedCount++
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

	g.Expect(result.SurfacedCount).To(Equal(4))
	g.Expect(result.Title).To(Equal("test"))
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

	err = memory.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.SurfacedCount++
	})
	g.Expect(err).To(HaveOccurred())
}

func TestReadModifyWrite_NonexistentFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	err := memory.ReadModifyWrite("/nonexistent/path/test.toml", func(r *memory.MemoryRecord) {
		r.SurfacedCount++
	})
	g.Expect(err).To(HaveOccurred())
}

func TestReadModifyWrite_PreservesAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.toml")

	initial := memory.MemoryRecord{
		Title:             "preserve-test",
		Content:           "some content",
		ObservationType:   "pattern",
		Concepts:          []string{"foo", "bar"},
		Keywords:          []string{"kw1"},
		Principle:         "do the thing",
		AntiPattern:       "don't do the other thing",
		Rationale:         "because reasons",
		Confidence:        "A",
		SurfacedCount:     5,
		FollowedCount:     2,
		ContradictedCount: 1,
		IgnoredCount:      0,
		IrrelevantCount:   1,
		Absorbed: []memory.AbsorbedRecord{
			{From: "old.toml", SurfacedCount: 2, ContentHash: "abc123", MergedAt: "2025-01-01"},
		},
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

	err = memory.ReadModifyWrite(path, func(r *memory.MemoryRecord) {
		r.SurfacedCount++
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

	g.Expect(result.SurfacedCount).To(Equal(6))
	g.Expect(result.Title).To(Equal("preserve-test"))
	g.Expect(result.Content).To(Equal("some content"))
	g.Expect(result.Concepts).To(ConsistOf("foo", "bar"))
	g.Expect(result.Absorbed).To(HaveLen(1))
	g.Expect(result.Absorbed[0].From).To(Equal("old.toml"))
}
