package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestRenderFactContent_DisplaysSubjectPredicateObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type:      "fact",
		Situation: "Go projects",
		Content: memory.ContentFields{
			Subject:   "this project",
			Predicate: "uses",
			Object:    "targ build system",
		},
	}
	cli.ExportRenderFactContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("Situation: Go projects"))
	g.Expect(output).To(ContainSubstring("Subject: this project"))
	g.Expect(output).To(ContainSubstring("Predicate: uses"))
	g.Expect(output).To(ContainSubstring("Object: targ build system"))
}

func TestRenderMemoryContent_FactType_ShowsFactFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type:   "fact",
		Source: "user",
		Content: memory.ContentFields{
			Subject:   "engram",
			Predicate: "is written in",
			Object:    "Go",
		},
	}
	cli.ExportRenderMemoryContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("Type: fact"))
	g.Expect(output).To(ContainSubstring("Source: user"))
	g.Expect(output).To(ContainSubstring("Subject: engram"))
	g.Expect(output).NotTo(ContainSubstring("Behavior:"))
}

func TestRenderMemoryContent_FeedbackType_ShowsSBIAFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type:   "feedback",
		Source: "observation",
		Content: memory.ContentFields{
			Behavior: "running go test",
			Impact:   "misses flags",
			Action:   "use targ",
		},
	}
	cli.ExportRenderMemoryContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("Type: feedback"))
	g.Expect(output).To(ContainSubstring("Source: observation"))
	g.Expect(output).To(ContainSubstring("Behavior: running go test"))
	g.Expect(output).NotTo(ContainSubstring("Subject:"))
}

func TestRenderMemoryContent_IncludesTimestamps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}
	cli.ExportRenderMemoryContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("Created: 2026-01-01T00:00:00Z"))
	g.Expect(output).To(ContainSubstring("Updated: 2026-01-02T00:00:00Z"))
}

func TestShow_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeForTest(t, []string{"engram", "show", "--bogus-flag"})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestShow_HappyPath_PrintsSBIAFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `situation = "when running tests"
updated_at = "2025-01-01T00:00:00Z"

[content]
behavior = "use go test directly"
impact = "misses coverage and lint flags"
action = "use targ test instead"
`
	err = os.WriteFile(
		filepath.Join(memDir, "use-targ-test.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "show",
		"--name", "use-targ-test",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Situation: when running tests"))
	g.Expect(stdout).To(ContainSubstring("Behavior: use go test directly"))
	g.Expect(stdout).To(ContainSubstring("Impact: misses coverage and lint flags"))
	g.Expect(stdout).To(ContainSubstring("Action: use targ test instead"))
	g.Expect(stdout).To(ContainSubstring("Updated: 2025-01-01T00:00:00Z"))
	g.Expect(stdout).NotTo(ContainSubstring("Effectiveness:"))
}

func TestShow_MemoryNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, stderr := executeForTest(t, []string{
		"engram", "show",
		"--name", "nonexistent-memory",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestShow_MissingSlug_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	_, stderr := executeForTest(t, []string{"engram", "show", "--data-dir", dataDir})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestShow_NameFlag_Works(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(memDir, "flag-test.toml"),
		[]byte("situation = \"Flag Test Situation\"\n\n[content]\naction = \"Use --name flag\"\n"),
		0o640,
	)).To(Succeed())

	stdout, stderr := executeForTest(t, []string{"engram", "show", "--name", "flag-test", "--data-dir", dataDir})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Situation: Flag Test Situation"))
}

func TestShow_OmitsEmptyFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Memory with only situation and action -- no behavior, impact.
	tomlContent := `situation = "Minimal Memory"
updated_at = "2025-01-01T00:00:00Z"

[content]
action = "Just action"
`
	err = os.WriteFile(
		filepath.Join(memDir, "minimal.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "show",
		"--name", "minimal",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Situation: Minimal Memory"))
	g.Expect(stdout).To(ContainSubstring("Action: Just action"))
	g.Expect(stdout).NotTo(ContainSubstring("Behavior:"))
	g.Expect(stdout).NotTo(ContainSubstring("Impact:"))
	g.Expect(stdout).NotTo(ContainSubstring("Effectiveness:"))
}

func TestShow_ReadsFromFactsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	factsDir := filepath.Join(dataDir, "memory", "facts")
	err := os.MkdirAll(factsDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `type = "fact"
situation = "Go projects"

[content]
subject = "engram"
predicate = "uses"
object = "Go"
`
	err = os.WriteFile(
		filepath.Join(factsDir, "fact-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "show",
		"--name", "fact-mem",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Type: fact"))
	g.Expect(stdout).To(ContainSubstring("Subject: engram"))
	g.Expect(stdout).To(ContainSubstring("Predicate: uses"))
	g.Expect(stdout).To(ContainSubstring("Object: Go"))
	g.Expect(stdout).NotTo(ContainSubstring("Behavior:"))
}

func TestShow_ReadsFromFeedbackDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	err := os.MkdirAll(feedbackDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `type = "feedback"
situation = "from feedback dir"

[content]
behavior = "test behavior"
action = "test action"
`
	err = os.WriteFile(
		filepath.Join(feedbackDir, "fb-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "show",
		"--name", "fb-mem",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Situation: from feedback dir"))
	g.Expect(stdout).To(ContainSubstring("Type: feedback"))
}

func TestShow_SlugAfterFlags_Works(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `situation = "After Flags"
updated_at = "2025-01-01T00:00:00Z"

[content]
action = "Slug comes after --data-dir"
`
	err = os.WriteFile(
		filepath.Join(memDir, "after-flags.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "show",
		"--data-dir", dataDir,
		"--name", "after-flags",
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("Situation: After Flags"))
}
