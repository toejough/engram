package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestRenderMemoryMeta_CreatedAt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	cli.ExportRenderMemoryMeta(&buf, mem)
	g.Expect(buf.String()).To(ContainSubstring("Created: 2026-01-01T00:00:00Z"))
}

func TestRenderMemoryMeta_IrrelevantCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		FollowedCount:    6,
		NotFollowedCount: 2,
		IrrelevantCount:  2,
	}
	cli.ExportRenderMemoryMeta(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("Effectiveness: 75%"))
	g.Expect(output).To(ContainSubstring("Relevance: 80%"))
	g.Expect(output).To(ContainSubstring("2 irrelevant"))
}

func TestRenderMemoryMeta_ProjectScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		ProjectScoped: true,
		ProjectSlug:   "my-project",
	}
	cli.ExportRenderMemoryMeta(&buf, mem)
	g.Expect(buf.String()).To(ContainSubstring("Scope: project (my-project)"))
}

func TestShow_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("show"))
	}
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
project_scoped = true
project_slug = "engram"
surfaced_count = 10
followed_count = 8
not_followed_count = 2
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"use-targ-test",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Situation: when running tests"))
	g.Expect(output).To(ContainSubstring("Behavior: use go test directly"))
	g.Expect(output).To(ContainSubstring("Impact: misses coverage and lint flags"))
	g.Expect(output).To(ContainSubstring("Action: use targ test instead"))
	g.Expect(output).To(ContainSubstring("Effectiveness: 80%"))
	g.Expect(output).To(ContainSubstring("8 followed"))
	g.Expect(output).To(ContainSubstring("2 not followed"))
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"nonexistent-memory",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("nonexistent-memory"))
	}
}

func TestShow_MissingSlug_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("slug"))
	}
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

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "--name", "flag-test", "--data-dir", dataDir},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Situation: Flag Test Situation"))
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

	// Memory with only situation and action — no behavior, impact.
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"minimal",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Situation: Minimal Memory"))
	g.Expect(output).To(ContainSubstring("Action: Just action"))
	g.Expect(output).NotTo(ContainSubstring("Behavior:"))
	g.Expect(output).NotTo(ContainSubstring("Impact:"))
	g.Expect(output).NotTo(ContainSubstring("Effectiveness:"))
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"fact-mem",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Type: fact"))
	g.Expect(output).To(ContainSubstring("Subject: engram"))
	g.Expect(output).To(ContainSubstring("Predicate: uses"))
	g.Expect(output).To(ContainSubstring("Object: Go"))
	g.Expect(output).NotTo(ContainSubstring("Behavior:"))
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"fb-mem",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Situation: from feedback dir"))
	g.Expect(output).To(ContainSubstring("Type: feedback"))
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

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "show",
			"--data-dir", dataDir,
			"after-flags",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Situation: After Flags"))
}
