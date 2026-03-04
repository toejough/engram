package tomlwriter_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/tomlwriter"
)

func TestT8_WriteCreatesTomlFileWithAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()
	created := time.Date(2026, 3, 3, 18, 0, 0, 0, time.UTC)

	memory := &tomlwriter.EnrichedMemory{
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
	filePath, err := writer.Write(memory, dataDir)
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

	var parsed struct {
		Title           string   `toml:"title"`
		Content         string   `toml:"content"`
		ObservationType string   `toml:"observation_type"`
		Concepts        []string `toml:"concepts"`
		Keywords        []string `toml:"keywords"`
		Principle       string   `toml:"principle"`
		AntiPattern     string   `toml:"anti_pattern"`
		Rationale       string   `toml:"rationale"`
		Confidence      string   `toml:"confidence"`
		CreatedAt       string   `toml:"created_at"`
		UpdatedAt       string   `toml:"updated_at"`
	}

	_, decodeErr := toml.DecodeFile(filePath, &parsed)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(parsed.Title).To(Equal(memory.Title))
	g.Expect(parsed.Content).To(Equal(memory.Content))
	g.Expect(parsed.ObservationType).To(Equal(memory.ObservationType))
	g.Expect(parsed.Concepts).To(Equal(memory.Concepts))
	g.Expect(parsed.Keywords).To(Equal(memory.Keywords))
	g.Expect(parsed.Principle).To(Equal(memory.Principle))
	g.Expect(parsed.AntiPattern).To(Equal(memory.AntiPattern))
	g.Expect(parsed.Rationale).To(Equal(memory.Rationale))
	g.Expect(parsed.Confidence).To(Equal(memory.Confidence))
	g.Expect(parsed.CreatedAt).To(Equal(created.Format(time.RFC3339)))
	g.Expect(parsed.UpdatedAt).To(Equal(created.Format(time.RFC3339)))
}

func TestT9_FilenameSlugIsHyphenatedLowercaseWords(t *testing.T) {
	t.Parallel()

	t.Run("example", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		memory := &tomlwriter.EnrichedMemory{
			FilenameSummary: "Use Targ Not Go Test",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		writer := tomlwriter.New()
		filePath, err := writer.Write(memory, t.TempDir())
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
				rapid.RuneFrom([]rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_")),
			).Draw(rt, "summary")

			dataDir, mkdirErr := os.MkdirTemp("", "engram-tomlwriter-*")
			g.Expect(mkdirErr).NotTo(HaveOccurred())

			if mkdirErr != nil {
				return
			}

			rt.Cleanup(func() { os.RemoveAll(dataDir) })

			memory := &tomlwriter.EnrichedMemory{
				FilenameSummary: summary,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			writer := tomlwriter.New()
			filePath, writeErr := writer.Write(memory, dataDir)
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

func TestT10_DuplicateFilenameGetsNumericSuffix(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	memory := &tomlwriter.EnrichedMemory{
		FilenameSummary: "use targ not go test",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()

	firstPath, err := writer.Write(memory, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(firstPath)).To(Equal("use-targ-not-go-test.toml"))

	secondPath, err := writer.Write(memory, dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(secondPath)).To(Equal("use-targ-not-go-test-2.toml"))
	g.Expect(secondPath).NotTo(Equal(firstPath))
}

func TestT11_WriteIsAtomic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()

	memory := &tomlwriter.EnrichedMemory{
		FilenameSummary: "atomic write test memory",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(memory, dataDir)
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

func TestT12_MemoriesDirectoryCreatedIfMissing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")

	_, preStatErr := os.Stat(memoriesDir)
	g.Expect(os.IsNotExist(preStatErr)).To(BeTrue())

	memory := &tomlwriter.EnrichedMemory{
		FilenameSummary: "dir creation test memory",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	writer := tomlwriter.New()
	filePath, err := writer.Write(memory, dataDir)
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
