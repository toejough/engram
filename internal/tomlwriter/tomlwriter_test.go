package tomlwriter_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

func TestAtomicWrite_CleansUpOnEncodeError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.toml")

	writer := tomlwriter.New()
	err := writer.AtomicWrite(targetPath, map[string]any{"bad": make(chan int)})
	g.Expect(err).To(HaveOccurred())

	entries, dirErr := os.ReadDir(dir)
	g.Expect(dirErr).NotTo(HaveOccurred())

	if dirErr != nil {
		return
	}

	g.Expect(entries).To(BeEmpty())
}

func TestAtomicWrite_WritesAndRenames(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "test.toml")

	writer := tomlwriter.New()

	record := map[string]string{"key": "value"}
	err := writer.AtomicWrite(targetPath, record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(targetPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring(`key = "value"`))
}

func TestWithMkdirAllError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithMkdirAll(func(string, os.FileMode) error {
			return errors.New("permission denied")
		}),
	)

	record := &memory.MemoryRecord{Situation: "test"}

	_, writeErr := writer.Write(record, "mkdirall-error-test", t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("create memories dir"))
	}
}

func TestWithStatError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithStat(func(string) (os.FileInfo, error) {
			return nil, errors.New("stat failed")
		}),
	)

	record := &memory.MemoryRecord{Situation: "test"}

	_, writeErr := writer.Write(record, "stat-error-test", t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("stat"))
	}
}

func TestWriteAtomicCreateTempError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithCreateTemp(func(string, string) (*os.File, error) {
			return nil, errors.New("disk full")
		}),
	)

	record := &memory.MemoryRecord{Situation: "test"}

	_, writeErr := writer.Write(record, "error-test", t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("creating temp file"))
	}
}

func TestWriteAtomicRenameError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithRename(func(string, string) error {
			return errors.New("cross-device link")
		}),
	)

	record := &memory.MemoryRecord{Situation: "test"}

	_, writeErr := writer.Write(record, "rename-error-test", t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("renaming temp file"))
	}
}

func TestWrite_CreatesTomlFileWithAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	record := &memory.MemoryRecord{
		Type:      "feedback",
		Source:    "observation",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses coverage and lint flags",
			Action:   "use targ test instead",
		},
		CreatedAt: "2026-03-03T18:00:00Z",
		UpdatedAt: "2026-03-03T18:00:00Z",
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(record, "use-targ-not-go-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filePath).To(HaveSuffix(".toml"))

	var parsed memory.MemoryRecord

	_, decodeErr := toml.DecodeFile(filePath, &parsed)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(parsed.Situation).To(Equal(record.Situation))
	g.Expect(parsed.Content.Behavior).To(Equal(record.Content.Behavior))
	g.Expect(parsed.Content.Impact).To(Equal(record.Content.Impact))
	g.Expect(parsed.Content.Action).To(Equal(record.Content.Action))
	g.Expect(parsed.Source).To(Equal("observation"))
	g.Expect(parsed.CreatedAt).To(Equal("2026-03-03T18:00:00Z"))
	g.Expect(parsed.UpdatedAt).To(Equal("2026-03-03T18:00:00Z"))
}

func TestWrite_DuplicateFilenameGetsNumericSuffix(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	record := &memory.MemoryRecord{
		Situation: "when testing",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses flags",
			Action:   "use targ test",
		},
	}

	writer := tomlwriter.New()

	firstPath, err := writer.Write(record, "use-targ-not-go-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(firstPath)).To(Equal("use-targ-not-go-test.toml"))

	secondRecord := &memory.MemoryRecord{
		Situation: "when testing again",
		Content: memory.ContentFields{
			Behavior: "use go test directly",
			Impact:   "misses flags",
			Action:   "use targ test",
		},
	}

	secondPath, err := writer.Write(secondRecord, "use-targ-not-go-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(secondPath)).To(Equal("use-targ-not-go-test-2.toml"))
	g.Expect(secondPath).NotTo(Equal(firstPath))
}

func TestWrite_FilenameSlugIsHyphenatedLowercase(t *testing.T) {
	t.Parallel()

	t.Run("example", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		record := &memory.MemoryRecord{
			Situation: "test situation",
			Content:   memory.ContentFields{Action: "test action"},
		}

		writer := tomlwriter.New()
		filePath, err := writer.Write(record, "Use Targ Not Go Test", t.TempDir())
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(filepath.Base(filePath)).To(Equal("use-targ-not-go-test.toml"))
	})

	t.Run("property", func(t *testing.T) {
		t.Parallel()

		validSlug := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

		rapid.Check(t, func(rt *rapid.T) {
			g := NewGomegaWithT(rt)

			summary := rapid.StringOf(
				rapid.RuneFrom(
					[]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_"),
				),
			).Draw(rt, "summary")

			dataDir := t.TempDir()

			record := &memory.MemoryRecord{
				Situation: "test",
				Content:   memory.ContentFields{Action: "test"},
			}

			writer := tomlwriter.New()
			filePath, writeErr := writer.Write(record, summary, dataDir)
			g.Expect(writeErr).NotTo(HaveOccurred())

			if writeErr != nil {
				return
			}

			filename := filepath.Base(filePath)
			g.Expect(filename).To(HaveSuffix(".toml"))

			slug := strings.TrimSuffix(filename, ".toml")
			g.Expect(validSlug.MatchString(slug)).To(BeTrue(),
				"slug %q does not match [a-z0-9]+(-[a-z0-9]+)*", slug)
		})
	})
}

func TestWrite_IncludesContentFieldKeys(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	record := &memory.MemoryRecord{
		Type:      "feedback",
		Source:    "user",
		Situation: "test situation",
		Content: memory.ContentFields{
			Behavior: "test behavior",
			Impact:   "test impact",
			Action:   "test action",
		},
	}

	writer := tomlwriter.New()

	path, err := writer.Write(record, "content-field-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	data, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	raw := string(data)
	g.Expect(raw).To(ContainSubstring("situation"), "situation field must be present")
	g.Expect(raw).To(ContainSubstring("behavior"), "behavior content field must be present")
	g.Expect(raw).To(ContainSubstring("impact"), "impact content field must be present")
	g.Expect(raw).To(ContainSubstring("action"), "action content field must be present")
}

func TestWrite_IsAtomic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	record := &memory.MemoryRecord{
		Situation: "atomic write test",
		Content: memory.ContentFields{
			Behavior: "test behavior",
			Impact:   "test impact",
			Action:   "test action",
		},
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(record, "atomic-write-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	memoriesDir := filepath.Join(dataDir, "memories")
	entries, readErr := os.ReadDir(memoriesDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	for _, entry := range entries {
		g.Expect(entry.Name()).To(HaveSuffix(".toml"),
			"unexpected non-toml file %q found", entry.Name())
	}

	_, statErr := os.Stat(filePath)
	g.Expect(statErr).NotTo(HaveOccurred())
}

func TestWrite_MemoriesDirectoryCreatedIfMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")

	_, preStatErr := os.Stat(memoriesDir)
	g.Expect(os.IsNotExist(preStatErr)).To(BeTrue())

	record := &memory.MemoryRecord{
		Situation: "dir creation test",
		Content: memory.ContentFields{
			Behavior: "test behavior",
			Impact:   "test impact",
			Action:   "test action",
		},
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(record, "dir-creation-test", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	info, postStatErr := os.Stat(memoriesDir)
	g.Expect(postStatErr).NotTo(HaveOccurred())

	if postStatErr != nil {
		return
	}

	g.Expect(info.IsDir()).To(BeTrue())

	_, fileStatErr := os.Stat(filePath)
	g.Expect(fileStatErr).NotTo(HaveOccurred())
}
