package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

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

func TestShow_HappyPath_PrintsAllFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "Use targ check-full"
principle = "Run targ check-full before declaring done"
anti_pattern = "Running targ check which stops at first error"
content = "Always use targ check-full for comprehensive validation"
keywords = ["targ", "check", "full"]
concepts = ["build-tools", "validation"]
surfaced_count = 10
followed_count = 8
contradicted_count = 1
ignored_count = 1
updated_at = "2025-01-01T00:00:00Z"
`
	err = os.WriteFile(
		filepath.Join(memDir, "use-targ-check-full.toml"),
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
			"use-targ-check-full",
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
	g.Expect(output).To(ContainSubstring("Title: Use targ check-full"))
	g.Expect(output).To(ContainSubstring(
		"Principle: Run targ check-full before declaring done",
	))
	g.Expect(output).To(ContainSubstring(
		"Anti-pattern: Running targ check which stops at first error",
	))
	g.Expect(output).To(ContainSubstring(
		"Content: Always use targ check-full for comprehensive validation",
	))
	g.Expect(output).To(ContainSubstring("Keywords: targ, check, full"))
	g.Expect(output).To(ContainSubstring("Effectiveness: 80%"))
	g.Expect(output).To(ContainSubstring("8 followed"))
	g.Expect(output).To(ContainSubstring("1 contradicted"))
	g.Expect(output).To(ContainSubstring("1 ignored"))
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

func TestShow_MissingDataDir_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "show", "some-slug"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
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

	// Memory with only title and content — no principle, anti-pattern, keywords.
	tomlContent := `title = "Minimal Memory"
content = "Just content"
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
	g.Expect(output).To(ContainSubstring("Title: Minimal Memory"))
	g.Expect(output).To(ContainSubstring("Content: Just content"))
	g.Expect(output).NotTo(ContainSubstring("Principle:"))
	g.Expect(output).NotTo(ContainSubstring("Anti-pattern:"))
	g.Expect(output).NotTo(ContainSubstring("Keywords:"))
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

	tomlContent := `title = "After Flags"
content = "Slug comes after --data-dir"
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

	// Slug after flags: engram show --data-dir /path after-flags
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
	g.Expect(output).To(ContainSubstring("Title: After Flags"))
}

func TestShow_SlugBeforeFlags_Works(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `title = "My Memory"
content = "Some content"
updated_at = "2025-01-01T00:00:00Z"
`
	err = os.WriteFile(
		filepath.Join(memDir, "my-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	// Slug before flags: engram show my-mem --data-dir /path
	err = cli.Run(
		[]string{
			"engram", "show",
			"my-mem",
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
	g.Expect(output).To(ContainSubstring("Title: My Memory"))
	g.Expect(output).To(ContainSubstring("Content: Some content"))
}
