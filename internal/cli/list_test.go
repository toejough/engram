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

func TestList_EmptyDataDir_ReturnsEmptyOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "list", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(BeEmpty())
}

func TestList_FeedbackMemory_OutputsTypeNameSituation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	err := os.MkdirAll(feedbackDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `schema_version = 2
type = "feedback"
source = "user"
situation = "When running build commands in the engram project"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-02T00:00:00Z"

[content]
behavior = "using go test directly"
impact = "misses coverage flags"
action = "use targ test instead"
`
	err = os.WriteFile(
		filepath.Join(feedbackDir, "use-targ-for-tests.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{"engram", "list", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("feedback"))
	g.Expect(output).To(ContainSubstring("use-targ-for-tests"))
	g.Expect(output).To(ContainSubstring("When running build commands in the engram project"))
}

func TestList_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "list", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("list"))
	}
}

func TestList_MultipleMixedMemories_OutputsBoth(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	factsDir := filepath.Join(dataDir, "memory", "facts")
	err := os.MkdirAll(feedbackDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.MkdirAll(factsDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	feedbackTOML := `schema_version = 2
type = "feedback"
source = "user"
situation = "When running build commands"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-02T00:00:00Z"

[content]
behavior = "using go test directly"
impact = "misses flags"
action = "use targ"
`
	err = os.WriteFile(
		filepath.Join(feedbackDir, "use-targ-for-tests.toml"),
		[]byte(feedbackTOML),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	factTOML := `schema_version = 2
type = "fact"
source = "user"
situation = "engram uses targ for all build operations"
created_at = "2026-01-01T00:00:00Z"
updated_at = "2026-01-01T00:00:00Z"

[content]
subject = "engram"
predicate = "uses"
object = "targ build system"
`
	err = os.WriteFile(
		filepath.Join(factsDir, "engram-uses-targ.toml"),
		[]byte(factTOML),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err = cli.Run(
		[]string{"engram", "list", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("feedback | use-targ-for-tests"))
	g.Expect(output).To(ContainSubstring("fact | engram-uses-targ"))
}
