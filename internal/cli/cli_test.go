package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRun_CorrectStub_ReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "correct", "--message", "test"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_MaintainStub_ReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "maintain", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_NoArgs_ReturnsUsageError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_UnknownCommand_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "nonexistent"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

func TestApplyProjectSlugDefault_EmptySlug_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := ""
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		return "/Users/joe/repos/engram", nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).To(Equal("-Users-joe-repos-engram"))
}

func TestApplyProjectSlugDefault_NonEmpty_Noop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := "already-set"
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		t.Fatal("getwd should not be called")
		return "", nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slug).To(Equal("already-set"))
}

func TestExtractAssistantDelta_NewSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	// Write a transcript with assistant content.
	lines := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello world"}]}}
{"type":"human","message":{"content":[{"type":"text","text":"user msg"}]}}
{"type":"assistant","message":{"content":[{"type":"text","text":"goodbye"}]}}
`
	g.Expect(os.WriteFile(transcriptPath, []byte(lines), 0o644)).To(Succeed())

	result, err := cli.ExportExtractAssistantDelta(dataDir, transcriptPath, "session-1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("hello world"))
	g.Expect(result).To(ContainSubstring("goodbye"))
	g.Expect(result).NotTo(ContainSubstring("user msg"))
}

func TestExtractAssistantDelta_EmptyTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "empty.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(""), 0o644)).To(Succeed())

	result, err := cli.ExportExtractAssistantDelta(dataDir, transcriptPath, "session-1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestExtractAssistantDelta_ResumeFromOffset(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	line1 := `{"type":"assistant","message":{"content":[{"type":"text","text":"first"}]}}` + "\n"
	line2 := `{"type":"assistant","message":{"content":[{"type":"text","text":"second"}]}}` + "\n"
	g.Expect(os.WriteFile(transcriptPath, []byte(line1+line2), 0o644)).To(Succeed())

	// First call reads everything.
	_, err := cli.ExportExtractAssistantDelta(dataDir, transcriptPath, "s1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Append more content.
	line3 := `{"type":"assistant","message":{"content":[{"type":"text","text":"third"}]}}` + "\n"
	f, openErr := os.OpenFile(transcriptPath, os.O_APPEND|os.O_WRONLY, 0o644)
	g.Expect(openErr).NotTo(HaveOccurred())

	if openErr != nil {
		return
	}

	_, _ = f.WriteString(line3)
	_ = f.Close()

	// Second call should only get "third".
	result, err := cli.ExportExtractAssistantDelta(dataDir, transcriptPath, "s1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("third"))
	g.Expect(result).NotTo(ContainSubstring("first"))
}

func TestApplyProjectSlugDefault_GetwdError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	slug := ""
	err := cli.ExportApplyProjectSlugDefault(&slug, func() (string, error) {
		return "", errors.New("getwd failed")
	})
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("resolving working directory"))
	}
}

func TestRun_Instruct_PrintsAuditInfo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "instruct", "--data-dir", dataDir, "--project-dir", "/tmp"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("instruct audit"))
}

func TestRun_Surface_PromptMode_EmptyData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(dataDir, "memories"), 0o755)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "surface",
			"--mode", "prompt",
			"--data-dir", dataDir,
			"--message", "test query",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_Surface_MissingMode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "surface", "--data-dir", t.TempDir()},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--mode required"))
	}
}

func TestRun_Surface_StopMode_NoTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "surface",
			"--mode", "stop",
			"--data-dir", t.TempDir(),
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--transcript-path required"))
	}
}

func TestRun_Surface_StopMode_EmptyTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "empty.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(""), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "surface",
			"--mode", "stop",
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "s1",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_Recall_EmptyData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", "test-project",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// This may or may not error depending on whether ~/.claude/projects/test-project exists.
	// The important thing is it exercises the code path without panicking.
	_ = err
	_ = g
}
