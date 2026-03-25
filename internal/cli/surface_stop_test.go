//nolint:testpackage // whitebox test — exercises unexported extractAssistantDelta and runSurface
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestExtractAssistantDelta_EmptyTranscript(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	err := os.WriteFile(transcriptPath, []byte(""), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	result, extractErr := extractAssistantDelta(dataDir, transcriptPath, "sess-1")
	g.Expect(extractErr).NotTo(HaveOccurred())

	if extractErr != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestExtractAssistantDelta_OnlyUserLines(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	jsonl := `{"type":"user","message":{"role":"user","content":"hello"}}` + "\n"

	err := os.WriteFile(transcriptPath, []byte(jsonl), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	result, extractErr := extractAssistantDelta(dataDir, transcriptPath, "sess-1")
	g.Expect(extractErr).NotTo(HaveOccurred())

	if extractErr != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestExtractAssistantDelta_ReturnsAssistantText(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	// JSONL with one assistant message containing "pre-existing"
	jsonl := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":"I found the pre-existing issue"}}` + "\n"

	err := os.WriteFile(transcriptPath, []byte(jsonl), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	result, extractErr := extractAssistantDelta(dataDir, transcriptPath, "sess-1")
	g.Expect(extractErr).NotTo(HaveOccurred())

	if extractErr != nil {
		return
	}

	g.Expect(result).To(ContainSubstring("pre-existing"))
}

func TestExtractAssistantDelta_SessionChangeResetsOffset(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	jsonl := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":"first response"}}` + "\n"

	err := os.WriteFile(transcriptPath, []byte(jsonl), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// First call with session-1
	result1, err1 := extractAssistantDelta(dataDir, transcriptPath, "sess-1")
	g.Expect(err1).NotTo(HaveOccurred())

	if err1 != nil {
		return
	}

	g.Expect(result1).To(ContainSubstring("first response"))

	// Second call with same session — should get empty (no new data)
	result2, err2 := extractAssistantDelta(dataDir, transcriptPath, "sess-1")
	g.Expect(err2).NotTo(HaveOccurred())

	if err2 != nil {
		return
	}

	g.Expect(result2).To(BeEmpty())

	// Third call with different session — should re-read from 0
	result3, err3 := extractAssistantDelta(dataDir, transcriptPath, "sess-2")
	g.Expect(err3).NotTo(HaveOccurred())

	if err3 != nil {
		return
	}

	g.Expect(result3).To(ContainSubstring("first response"))
}

func TestRunSurface_StopMode_EmptyTranscript(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")

	writeErr := os.WriteFile(transcriptPath, []byte(""), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	runErr := runSurface([]string{
		"--mode", "stop",
		"--transcript-path", transcriptPath,
		"--session-id", "test-session",
		"--data-dir", dataDir,
		"--format", "json",
	}, &stdout)
	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	output := stdout.String()
	// Empty transcript produces no output (empty assistant text -> early return)
	g.Expect(output).To(BeEmpty())
}

func TestRunSurface_StopMode_MissingTranscriptPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := runSurface([]string{
		"--mode", "stop",
		"--data-dir", dataDir,
	}, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("transcript-path")))
}

func TestRunSurface_StopMode_SurfacesMatchingMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Memory about pre-existing issues
	memContent := `title = "Pre-existing Issues"
content = "check for pre-existing issues before refactoring"
observation_type = "correction"
concepts = []
keywords = ["pre-existing", "refactoring"]
principle = "always check for pre-existing issues"
anti_pattern = ""
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`

	memErr := os.WriteFile(
		filepath.Join(memoriesDir, "pre-existing.toml"),
		[]byte(memContent), 0o640)
	g.Expect(memErr).NotTo(HaveOccurred())

	if memErr != nil {
		return
	}

	// Add filler memories so keyword scoring works
	for _, name := range []string{"testing", "linting", "docker"} {
		filler := `title = "` + name + `"
content = "` + name + ` stuff"
observation_type = "observation"
concepts = []
keywords = ["` + name + `"]
principle = "use ` + name + `"
anti_pattern = ""
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`
		writeErr := os.WriteFile(
			filepath.Join(memoriesDir, name+".toml"),
			[]byte(filler), 0o640)
		g.Expect(writeErr).NotTo(HaveOccurred())
	}

	// Transcript with assistant text mentioning "pre-existing"
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	jsonl := `{"type":"assistant","message":{"role":"assistant",` +
		`"content":"I found a pre-existing issue in the refactoring"}}` + "\n"

	writeErr := os.WriteFile(transcriptPath, []byte(jsonl), 0o640)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout bytes.Buffer

	runErr := runSurface([]string{
		"--mode", "stop",
		"--transcript-path", transcriptPath,
		"--session-id", "test-session",
		"--data-dir", dataDir,
	}, &stdout)
	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("pre-existing"))
}
