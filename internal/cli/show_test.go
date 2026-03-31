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
behavior = "use go test directly"
impact = "misses coverage and lint flags"
action = "use targ test instead"
project_scoped = true
project_slug = "engram"
surfaced_count = 10
followed_count = 8
not_followed_count = 2
updated_at = "2025-01-01T00:00:00Z"
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
		[]byte("situation = \"Flag Test Situation\"\naction = \"Use --name flag\"\n"),
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
action = "Just action"
updated_at = "2025-01-01T00:00:00Z"
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
action = "Slug comes after --data-dir"
updated_at = "2025-01-01T00:00:00Z"
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
