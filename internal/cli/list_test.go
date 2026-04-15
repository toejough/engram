package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestList_EmptyDataDir_ReturnsEmptyOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	stdout, stderr := executeForTest(t, []string{"engram", "list", "--data-dir", dataDir})
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(BeEmpty())
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

	stdout, stderr := executeForTest(t, []string{"engram", "list", "--data-dir", dataDir})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("feedback"))
	g.Expect(stdout).To(ContainSubstring("use-targ-for-tests"))
	g.Expect(stdout).To(ContainSubstring("When running build commands in the engram project"))
}

func TestList_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeForTest(t, []string{"engram", "list", "--bogus-flag"})
	g.Expect(stderr).NotTo(BeEmpty())
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

	stdout, stderr := executeForTest(t, []string{"engram", "list", "--data-dir", dataDir})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("feedback | use-targ-for-tests"))
	g.Expect(stdout).To(ContainSubstring("fact | engram-uses-targ"))
}
