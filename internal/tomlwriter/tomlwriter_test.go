package tomlwriter_test

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/memory"
	"engram/internal/tomlwriter"
)

// T-10: Duplicate filename gets numeric suffix
func TestT10_DuplicateFilenameGetsNumericSuffix(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	mem := &memory.Enriched{
		FilenameSummary: "use targ not go test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()

	firstPath, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(firstPath)).To(Equal("use-targ-not-go-test.toml"))

	secondPath, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(secondPath)).To(Equal("use-targ-not-go-test-2.toml"))
	g.Expect(secondPath).NotTo(Equal(firstPath))
}

// T-11: Write is atomic (temp file + rename)
func TestT11_WriteIsAtomic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	mem := &memory.Enriched{
		FilenameSummary: "atomic write test memory",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// After Write completes, no temp files should remain — only the final .toml
	memoriesDir := filepath.Join(dataDir, "memories")
	entries, readErr := os.ReadDir(memoriesDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	for _, entry := range entries {
		g.Expect(entry.Name()).To(HaveSuffix(".toml"),
			"unexpected non-toml file %q found (temp file not cleaned up)", entry.Name())
	}

	_, statErr := os.Stat(filePath)
	g.Expect(statErr).NotTo(HaveOccurred())
}

// T-12: Memories directory is created if missing
func TestT12_MemoriesDirectoryCreatedIfMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")

	_, preStatErr := os.Stat(memoriesDir)
	g.Expect(os.IsNotExist(preStatErr)).To(BeTrue())

	mem := &memory.Enriched{
		FilenameSummary: "dir creation test memory",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(mem, dataDir)
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

// T-8: Write creates TOML file with all fields
func TestT8_WriteCreatesTomlFileWithAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()
	created := time.Date(2026, 3, 3, 18, 0, 0, 0, time.UTC)

	mem := &memory.Enriched{
		Title:           "Use targ test not go test",
		Content:         "This project uses the targ build system...",
		ObservationType: "correction",
		Concepts:        []string{"targ", "build-system", "testing"},
		Keywords:        []string{"targ", "test", "go-test", "build", "check"},
		Principle:       "Use project-specific build tools",
		AntiPattern:     "Running go test directly",
		Rationale:       "targ wraps go test with proper flags and coverage requirements",
		FilenameSummary: "use targ not go test",
		Confidence:      "B",
		CreatedAt:       created,
		UpdatedAt:       created,
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filePath).To(HaveSuffix(".toml"))

	_, statErr := os.Stat(filePath)
	g.Expect(statErr).NotTo(HaveOccurred())

	if statErr != nil {
		return
	}

	var parsed memory.MemoryRecord

	_, decodeErr := toml.DecodeFile(filePath, &parsed)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(parsed.Title).To(Equal(mem.Title))
	g.Expect(parsed.Content).To(Equal(mem.Content))
	g.Expect(parsed.ObservationType).To(Equal(mem.ObservationType))
	g.Expect(parsed.Concepts).To(Equal(mem.Concepts))
	g.Expect(parsed.Keywords).To(Equal(mem.Keywords))
	g.Expect(parsed.Principle).To(Equal(mem.Principle))
	g.Expect(parsed.AntiPattern).To(Equal(mem.AntiPattern))
	g.Expect(parsed.Rationale).To(Equal(mem.Rationale))
	g.Expect(parsed.Confidence).To(Equal(mem.Confidence))
	g.Expect(parsed.CreatedAt).To(Equal(created.Format(time.RFC3339)))
	g.Expect(parsed.UpdatedAt).To(Equal(created.Format(time.RFC3339)))
}

// T-9: Filename slug is 3-5 hyphenated lowercase words
func TestT9_FilenameSlugIsHyphenatedLowercaseWords(t *testing.T) {
	t.Parallel()

	t.Run("example", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		mem := &memory.Enriched{
			FilenameSummary: "Use Targ Not Go Test",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		writer := tomlwriter.New()
		filePath, err := writer.Write(mem, t.TempDir())
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

			mem := &memory.Enriched{
				FilenameSummary: summary,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			writer := tomlwriter.New()
			filePath, writeErr := writer.Write(mem, dataDir)
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

// TestWithMkdirAllError verifies that an injected mkdirAll failure propagates.
func TestWithMkdirAllError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithMkdirAll(func(string, os.FileMode) error {
			return errors.New("permission denied")
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "mkdirall error test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("create memories dir"))
	}
}

// TestWithRemoveIsWired verifies that the injected remove function is called
// during cleanup when encoding fails.
func TestWithRemoveIsWired(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	removeCalled := false

	writer := tomlwriter.New(
		tomlwriter.WithCreateTemp(func(dir, pattern string) (*os.File, error) {
			f, err := os.CreateTemp(dir, pattern)
			if err != nil {
				return nil, err
			}

			_ = f.Close()

			return f, nil
		}),
		tomlwriter.WithRemove(func(string) error {
			removeCalled = true
			return nil
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "remove wired test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(removeCalled).To(BeTrue())
	}
}

// TestWithStatError verifies that an injected stat failure propagates.
func TestWithStatError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithStat(func(string) (os.FileInfo, error) {
			return nil, errors.New("stat failed")
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "stat error test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("stat"))
	}
}

// TestWriteAtomicCreateTempError verifies that a CreateTemp failure
// propagates as a "create temp file" error.
func TestWriteAtomicCreateTempError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithCreateTemp(func(string, string) (*os.File, error) {
			return nil, errors.New("disk full")
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "error test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("create temp file"))
	}
}

// TestWriteAtomicEncodeError verifies that a TOML encode failure (from a
// pre-closed temp file) propagates as an "encode TOML" error.
func TestWriteAtomicEncodeError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	writer := tomlwriter.New(
		tomlwriter.WithCreateTemp(func(dir, pattern string) (*os.File, error) {
			// Create a real temp file but close it immediately so encoding fails.
			f, err := os.CreateTemp(dir, pattern)
			if err != nil {
				return nil, err
			}

			_ = f.Close()

			return f, nil
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "encode error test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, dataDir)
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("encode TOML"))
	}
}

// TestWriteAtomicRenameError verifies that a rename failure propagates
// as a "rename to final path" error and cleans up the temp file.
func TestWriteAtomicRenameError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	writer := tomlwriter.New(
		tomlwriter.WithRename(func(string, string) error {
			return errors.New("cross-device link")
		}),
	)

	mem := &memory.Enriched{
		FilenameSummary: "rename error test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, writeErr := writer.Write(mem, t.TempDir())
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("rename to final path"))
	}
}

// TestWrite_ProjectSlugAndGeneralizabilityWrittenToTOML verifies that ProjectSlug and
// Generalizability on Enriched are persisted to the TOML file.
func TestWrite_ProjectSlugAndGeneralizabilityWrittenToTOML(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	mem := &memory.Enriched{
		Title:            "project slug test memory",
		Content:          "test content",
		FilenameSummary:  "project-slug-generalizability-test",
		ProjectSlug:      "-Users-joe-repos-foo",
		Generalizability: 4,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	writer := tomlwriter.New()

	path, err := writer.Write(mem, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed memory.MemoryRecord

	_, decodeErr := toml.DecodeFile(path, &parsed)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(parsed.ProjectSlug).To(Equal("-Users-joe-repos-foo"))
	g.Expect(parsed.Generalizability).To(Equal(4))
}

func TestWrite_IncludesTrackingFieldKeys(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	mem := &memory.Enriched{
		Title:           "test memory",
		Content:         "test content",
		FilenameSummary: "tracking-field-test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()

	path, err := writer.Write(mem, dataDir)
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
	g.Expect(raw).To(ContainSubstring("surfaced_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("followed_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("contradicted_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("ignored_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("irrelevant_count"), "tracking field key must be present")
	g.Expect(raw).To(ContainSubstring("last_surfaced_at"), "tracking field key must be present")
}
