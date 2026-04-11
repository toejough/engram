package stripkeywords_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/stripkeywords"
)

func TestRunCLI_FailsOnMissingFeedbackDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	exitCode := stripkeywords.RunCLI([]string{"--data-dir", dataDir})
	g.Expect(exitCode).To(Equal(1))
}

func TestRunCLI_SucceedsWithValidDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	exitCode := stripkeywords.RunCLI([]string{"--data-dir", dataDir})
	g.Expect(exitCode).To(Equal(0))
}

func TestRun_CreateTempFails_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	rec := memRecord{
		Type:      "feedback",
		Situation: "when running tests\nKeywords: go, targ",
		UpdatedAt: "2026-01-01T00:00:00Z",
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	var buf bytes.Buffer

	g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), buf.Bytes(), 0o640)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}
	deps.CreateTemp = func(string, string) (*os.File, error) {
		return nil, errors.New("disk full")
	}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("creating temp file")))
}

func TestRun_Idempotent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	rec := memRecord{
		Type:      "feedback",
		Situation: "when running tests\nKeywords: go, targ",
		UpdatedAt: "2026-01-01T00:00:00Z",
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	var buf bytes.Buffer

	g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), buf.Bytes(), 0o640)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	g.Expect(stripkeywords.Run(dataDir, deps)).To(Succeed())

	secondOut := &bytes.Buffer{}
	deps.Stdout = secondOut

	g.Expect(stripkeywords.Run(dataDir, deps)).To(Succeed())

	data, readErr := os.ReadFile(filepath.Join(feedbackDir, "mem.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var result memRecord

	_, decErr := toml.Decode(string(data), &result)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(result.Situation).To(Equal("when running tests"))
	g.Expect(secondOut.String()).To(ContainSubstring("Stripped: 0"))
}

func TestRun_InvalidTOML_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(feedbackDir, "bad.toml"),
		[]byte("not = [valid toml"),
		0o640,
	)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("bad.toml")))
}

func TestRun_MissingFeedbackDir_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	// No memory/feedback directory created

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_RenameFails_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	rec := memRecord{
		Type:      "feedback",
		Situation: "when running tests\nKeywords: go, targ",
		UpdatedAt: "2026-01-01T00:00:00Z",
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	var buf bytes.Buffer

	g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(feedbackDir, "mem.toml"), buf.Bytes(), 0o640)).To(Succeed())

	deps := stripkeywords.DefaultDeps()
	deps.Stdout = &bytes.Buffer{}
	deps.Rename = func(string, string) error {
		return errors.New("rename failed")
	}

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("renaming temp to destination")))
}

func TestRun_StripsKeywordsFromFeedbackAndFacts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	factsDir := filepath.Join(dataDir, "memory", "facts")

	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())

	writeMem := func(dir, filename, situation string) {
		rec := memRecord{
			Type:      "feedback",
			Situation: situation,
			UpdatedAt: "2026-01-01T00:00:00Z",
			CreatedAt: "2026-01-01T00:00:00Z",
		}

		var buf bytes.Buffer

		g.Expect(toml.NewEncoder(&buf).Encode(rec)).To(Succeed())
		g.Expect(os.WriteFile(filepath.Join(dir, filename), buf.Bytes(), 0o640)).To(Succeed())
	}

	writeMem(feedbackDir, "fb.toml", "when running tests\nKeywords: go, targ")
	writeMem(factsDir, "fact.toml", "context for project\nKeywords: project, context")
	writeMem(feedbackDir, "clean.toml", "when deploying")

	stdout := &bytes.Buffer{}
	deps := stripkeywords.DefaultDeps()
	deps.Stdout = stdout

	err := stripkeywords.Run(dataDir, deps)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	readSituation := func(path string) string {
		data, readErr := os.ReadFile(path)
		g.Expect(readErr).NotTo(HaveOccurred())

		if readErr != nil {
			return ""
		}

		var rec memRecord

		_, decErr := toml.Decode(string(data), &rec)
		g.Expect(decErr).NotTo(HaveOccurred())

		return rec.Situation
	}

	g.Expect(readSituation(filepath.Join(feedbackDir, "fb.toml"))).
		To(Equal("when running tests"))
	g.Expect(readSituation(filepath.Join(factsDir, "fact.toml"))).
		To(Equal("context for project"))
	g.Expect(readSituation(filepath.Join(feedbackDir, "clean.toml"))).
		To(Equal("when deploying"))

	out := stdout.String()
	g.Expect(out).To(ContainSubstring("STRIPPED: fb.toml"))
	g.Expect(out).To(ContainSubstring("STRIPPED: fact.toml"))
	g.Expect(out).To(ContainSubstring("OK (no change): clean.toml"))
	g.Expect(out).To(ContainSubstring("Stripped: 2, Unchanged: 1"))
}

// memRecord is a minimal struct for reading/writing test memory TOML files.
// (Avoids importing internal/memory which may pull in heavy deps.)
type memRecord struct {
	Type      string `toml:"type"`
	Situation string `toml:"situation"`
	UpdatedAt string `toml:"updated_at"`
	CreatedAt string `toml:"created_at"`
}
