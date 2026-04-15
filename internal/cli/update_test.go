package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestUpdate_AllFeedbackFields_UpdatesBehaviorImpactSource(t *testing.T) {
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
source = "human"
situation = "original"

[content]
behavior = "old behavior"
impact = "old impact"
action = "old action"
`
	memPath := filepath.Join(feedbackDir, "all-fields.toml")
	err = os.WriteFile(memPath, []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, stderr := executeForTest(t, []string{
		"engram", "update",
		"--name", "all-fields",
		"--situation", "updated situation",
		"--behavior", "new behavior",
		"--impact", "new impact",
		"--action", "new action",
		"--source", "agent",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(memPath, &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.Situation).To(Equal("updated situation"))
	g.Expect(record.Content.Behavior).To(Equal("new behavior"))
	g.Expect(record.Content.Impact).To(Equal("new impact"))
	g.Expect(record.Content.Action).To(Equal("new action"))
	g.Expect(record.Source).To(Equal("agent"))
}

func TestUpdate_FactFields_UpdatesSubjectPredicateObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	factsDir := filepath.Join(dataDir, "memory", "facts")
	err := os.MkdirAll(factsDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	tomlContent := `schema_version = 2
type = "fact"
source = "agent"
situation = "Go projects"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"

[content]
subject = "engram"
predicate = "uses"
object = "Go"
`
	memPath := filepath.Join(factsDir, "fact-update.toml")
	err = os.WriteFile(memPath, []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "update",
		"--name", "fact-update",
		"--subject", "this project",
		"--predicate", "is built with",
		"--object", "targ build system",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(ContainSubstring("UPDATED: fact-update"))

	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(memPath, &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.Content.Subject).To(Equal("this project"))
	g.Expect(record.Content.Predicate).To(Equal("is built with"))
	g.Expect(record.Content.Object).To(Equal("targ build system"))

	// Preserved
	g.Expect(record.Type).To(Equal("fact"))
	g.Expect(record.Source).To(Equal("agent"))
	g.Expect(record.Situation).To(Equal("Go projects"))
}

func TestUpdate_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeForTest(t, []string{"engram", "update", "--bogus-flag"})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestUpdate_InvalidSource_ReturnsError(t *testing.T) {
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
source = "human"
situation = "when running tests"

[content]
behavior = "use go test"
impact = "misses flags"
action = "use targ"
`
	err = os.WriteFile(
		filepath.Join(feedbackDir, "test-mem.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, stderr := executeForTest(t, []string{
		"engram", "update",
		"--name", "test-mem",
		"--source", "bot",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestUpdate_MemoryNotFound_ReturnsError(t *testing.T) {
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
		"engram", "update",
		"--name", "nonexistent",
		"--situation", "new situation",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestUpdate_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeForTest(t, []string{"engram", "update", "--data-dir", t.TempDir()})
	g.Expect(stderr).NotTo(BeEmpty())
}

func TestUpdate_OutputContainsUpdatedName(t *testing.T) {
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
source = "human"
situation = "test"

[content]
action = "test"
`
	err = os.WriteFile(
		filepath.Join(feedbackDir, "output-check.toml"),
		[]byte(tomlContent),
		0o640,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "update",
		"--name", "output-check",
		"--action", "updated action",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	g.Expect(stdout).To(HavePrefix("UPDATED: "))
	g.Expect(stdout).To(ContainSubstring("output-check"))
}

func TestUpdate_SituationField_UpdatesAndPreservesOthers(t *testing.T) {
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
source = "human"
situation = "original situation"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"

[content]
behavior = "original behavior"
impact = "original impact"
action = "original action"
`
	memPath := filepath.Join(feedbackDir, "update-test.toml")
	err = os.WriteFile(memPath, []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	stdout, stderr := executeForTest(t, []string{
		"engram", "update",
		"--name", "update-test",
		"--situation", "new situation",
		"--data-dir", dataDir,
	})
	g.Expect(stderr).To(BeEmpty())

	// Verify output
	g.Expect(stdout).To(ContainSubstring("UPDATED: update-test"))

	// Verify file was updated
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(memPath, &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	// Updated field
	g.Expect(record.Situation).To(Equal("new situation"))

	// Preserved fields
	g.Expect(record.Type).To(Equal("feedback"))
	g.Expect(record.Source).To(Equal("human"))
	g.Expect(record.Content.Behavior).To(Equal("original behavior"))
	g.Expect(record.Content.Impact).To(Equal("original impact"))
	g.Expect(record.Content.Action).To(Equal("original action"))
	g.Expect(record.CreatedAt).To(Equal("2025-01-01T00:00:00Z"))

	// UpdatedAt should be changed
	g.Expect(record.UpdatedAt).NotTo(Equal("2025-01-01T00:00:00Z"))
	g.Expect(record.UpdatedAt).NotTo(BeEmpty())
}
