package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestBuildMemoryIndex_EmptyList_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := cli.ExportBuildMemoryIndex(nil)
	g.Expect(result).To(BeEmpty())
}

func TestBuildMemoryIndex_FormatsCorrectly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memories := []*memory.Stored{
		{
			Type:      "feedback",
			Situation: "when running tests",
			FilePath:  "/data/memory/feedback/use-targ.toml",
		},
		{
			Type:      "fact",
			Situation: "Go projects",
			FilePath:  "/data/memory/facts/engram-uses-go.toml",
		},
	}

	result := cli.ExportBuildMemoryIndex(memories)
	g.Expect(result).To(ContainSubstring("feedback | use-targ | when running tests"))
	g.Expect(result).To(ContainSubstring("fact | engram-uses-go | Go projects"))
}

func TestDescribeNewMemory_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{
		Type:      "fact",
		Source:    "agent",
		Situation: "Go projects",
		Content: memory.ContentFields{
			Subject:   "engram",
			Predicate: "uses",
			Object:    "targ",
		},
	}

	result := cli.ExportDescribeNewMemory(record)
	g.Expect(result).To(ContainSubstring("Type: fact"))
	g.Expect(result).To(ContainSubstring("Subject: engram"))
	g.Expect(result).To(ContainSubstring("Predicate: uses"))
	g.Expect(result).To(ContainSubstring("Object: targ"))
	g.Expect(result).To(ContainSubstring("Source: agent"))
	g.Expect(result).NotTo(ContainSubstring("Behavior:"))
}

func TestDescribeNewMemory_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	record := &memory.MemoryRecord{
		Type:      "feedback",
		Source:    "human",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test",
			Impact:   "misses flags",
			Action:   "use targ",
		},
	}

	result := cli.ExportDescribeNewMemory(record)
	g.Expect(result).To(ContainSubstring("Type: feedback"))
	g.Expect(result).To(ContainSubstring("Situation: when running tests"))
	g.Expect(result).To(ContainSubstring("Behavior: use go test"))
	g.Expect(result).To(ContainSubstring("Impact: misses flags"))
	g.Expect(result).To(ContainSubstring("Action: use targ"))
	g.Expect(result).To(ContainSubstring("Source: human"))
	g.Expect(result).NotTo(ContainSubstring("Subject:"))
}

func TestLearnFact_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "learn", "fact", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("learn fact"))
	}
}

func TestLearnFact_InvalidSource_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "fact",
			"--situation", "test",
			"--subject", "test",
			"--predicate", "test",
			"--object", "test",
			"--source", "bot",
			"--no-dup-check",
			"--data-dir", t.TempDir(),
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("source"))
	}
}

func TestLearnFact_NoDupCheck_WritesToFactsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "fact",
			"--situation", "Go projects",
			"--subject", "engram",
			"--predicate", "uses",
			"--object", "targ build system",
			"--source", "agent",
			"--no-dup-check",
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
	g.Expect(output).To(ContainSubstring("CREATED:"))

	// Verify file was written to facts directory
	factsDir := filepath.Join(dataDir, "memory", "facts")
	entries, readErr := os.ReadDir(factsDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Name()).To(HaveSuffix(".toml"))

	// Verify TOML content
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(filepath.Join(factsDir, entries[0].Name()), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.SchemaVersion).To(Equal(2))
	g.Expect(record.Type).To(Equal("fact"))
	g.Expect(record.Source).To(Equal("agent"))
	g.Expect(record.Situation).To(Equal("Go projects"))
	g.Expect(record.Content.Subject).To(Equal("engram"))
	g.Expect(record.Content.Predicate).To(Equal("uses"))
	g.Expect(record.Content.Object).To(Equal("targ build system"))
}

func TestLearnFeedback_AgentSource_Accepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "feedback",
			"--situation", "agent test",
			"--behavior", "observed behavior",
			"--impact", "positive impact",
			"--action", "continue doing this",
			"--source", "agent",
			"--no-dup-check",
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("CREATED:"))
}

func TestLearnFeedback_FlagParseError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "learn", "feedback", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("learn feedback"))
	}
}

func TestLearnFeedback_InvalidSource_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "feedback",
			"--situation", "test",
			"--behavior", "test",
			"--impact", "test",
			"--action", "test",
			"--source", "invalid",
			"--no-dup-check",
			"--data-dir", t.TempDir(),
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("source"))
		g.Expect(err.Error()).To(ContainSubstring("human"))
		g.Expect(err.Error()).To(ContainSubstring("agent"))
	}
}

func TestLearnFeedback_NoDupCheck_WritesToFeedbackDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "feedback",
			"--situation", "when running tests",
			"--behavior", "use go test directly",
			"--impact", "misses coverage flags",
			"--action", "use targ test instead",
			"--source", "human",
			"--no-dup-check",
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
	g.Expect(output).To(ContainSubstring("CREATED:"))

	// Verify file was written to feedback directory
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	entries, readErr := os.ReadDir(feedbackDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Name()).To(HaveSuffix(".toml"))

	// Verify TOML content
	var record memory.MemoryRecord

	_, decErr := toml.DecodeFile(filepath.Join(feedbackDir, entries[0].Name()), &record)
	g.Expect(decErr).NotTo(HaveOccurred())

	if decErr != nil {
		return
	}

	g.Expect(record.SchemaVersion).To(Equal(2))
	g.Expect(record.Type).To(Equal("feedback"))
	g.Expect(record.Source).To(Equal("human"))
	g.Expect(record.Situation).To(Equal("when running tests"))
	g.Expect(record.Content.Behavior).To(Equal("use go test directly"))
	g.Expect(record.Content.Impact).To(Equal("misses coverage flags"))
	g.Expect(record.Content.Action).To(Equal("use targ test instead"))
	g.Expect(record.CreatedAt).NotTo(BeEmpty())
	g.Expect(record.UpdatedAt).NotTo(BeEmpty())
}

func TestLearnFeedback_OutputFormatIncludesCreatedName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn", "feedback",
			"--situation", "testing output format",
			"--behavior", "check format",
			"--impact", "ensures consistency",
			"--action", "verify output",
			"--source", "human",
			"--no-dup-check",
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
	g.Expect(output).To(HavePrefix("CREATED: "))
	g.Expect(output).To(ContainSubstring("testing-output-format"))
}

func TestLearn_NoSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "learn"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("learn"))
	}
}

func TestLearn_UnknownSubcommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "learn", "bogus"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown learn subcommand"))
	}
}

func TestParseConflictLine_MalformedLine_NoOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	cli.ExportParseConflictLine("no-colon-here", t.TempDir(), &buf)
	g.Expect(buf.String()).To(BeEmpty())
}

func TestParseConflictLine_ValidLine_PrintsConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	cli.ExportParseConflictLine("DUPLICATE: some-memory", t.TempDir(), &buf)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("DUPLICATE: some-memory"))
}

func TestParseConflictResponse_Contradiction_ReturnsTrue(t *testing.T) {
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
	err = os.WriteFile(filepath.Join(factsDir, "engram-uses-go.toml"), []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	result := cli.ExportParseConflictResponse("CONTRADICTION: engram-uses-go", dataDir, &buf)
	g.Expect(result).To(BeTrue())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("CONTRADICTION: engram-uses-go"))
	g.Expect(output).To(ContainSubstring("subject: engram"))
}

func TestParseConflictResponse_Duplicate_ReturnsTrue(t *testing.T) {
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
situation = "when running tests"

[content]
behavior = "use go test"
impact = "misses flags"
action = "use targ"
`
	err = os.WriteFile(filepath.Join(feedbackDir, "use-targ.toml"), []byte(tomlContent), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var buf bytes.Buffer

	result := cli.ExportParseConflictResponse("DUPLICATE: use-targ", dataDir, &buf)
	g.Expect(result).To(BeTrue())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("DUPLICATE: use-targ"))
	g.Expect(output).To(ContainSubstring("situation: when running tests"))
	g.Expect(output).To(ContainSubstring("behavior: use go test"))
}

func TestParseConflictResponse_MissingFile_StillReturnsTrueButSkipsContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	result := cli.ExportParseConflictResponse("DUPLICATE: nonexistent-memory", t.TempDir(), &buf)
	g.Expect(result).To(BeTrue())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("DUPLICATE: nonexistent-memory"))
	g.Expect(output).NotTo(ContainSubstring("situation:"))
}

func TestParseConflictResponse_None_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	result := cli.ExportParseConflictResponse("NONE", t.TempDir(), &buf)
	g.Expect(result).To(BeFalse())
	g.Expect(buf.String()).To(BeEmpty())
}

func TestParseConflictResponse_UnrecognizedLine_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	result := cli.ExportParseConflictResponse("something unexpected", t.TempDir(), &buf)
	g.Expect(result).To(BeFalse())
}

func TestRenderConflictContent_FactFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type:      "fact",
		Situation: "Go projects",
		Content: memory.ContentFields{
			Subject:   "engram",
			Predicate: "uses",
			Object:    "Go",
		},
	}

	cli.ExportRenderConflictContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("situation: Go projects"))
	g.Expect(output).To(ContainSubstring("subject: engram"))
	g.Expect(output).To(ContainSubstring("predicate: uses"))
	g.Expect(output).To(ContainSubstring("object: Go"))
	g.Expect(output).NotTo(ContainSubstring("behavior:"))
}

func TestRenderConflictContent_FeedbackFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type:      "feedback",
		Situation: "when running tests",
		Content: memory.ContentFields{
			Behavior: "use go test",
			Impact:   "misses flags",
			Action:   "use targ",
		},
	}

	cli.ExportRenderConflictContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("situation: when running tests"))
	g.Expect(output).To(ContainSubstring("behavior: use go test"))
	g.Expect(output).To(ContainSubstring("impact: misses flags"))
	g.Expect(output).To(ContainSubstring("action: use targ"))
	g.Expect(output).NotTo(ContainSubstring("subject:"))
}

func TestRenderConflictContent_OmitsEmptyFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	mem := &memory.MemoryRecord{
		Type: "feedback",
		Content: memory.ContentFields{
			Action: "just action",
		},
	}

	cli.ExportRenderConflictContent(&buf, mem)
	output := buf.String()
	g.Expect(output).To(ContainSubstring("action: just action"))
	g.Expect(output).NotTo(ContainSubstring("situation:"))
	g.Expect(output).NotTo(ContainSubstring("behavior:"))
	g.Expect(output).NotTo(ContainSubstring("impact:"))
}

func TestValidateSource_Agent_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportValidateSource("agent")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestValidateSource_Empty_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportValidateSource("")
	g.Expect(err).To(HaveOccurred())
}

func TestValidateSource_Human_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportValidateSource("human")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestValidateSource_Invalid_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportValidateSource("bot")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("human"))
		g.Expect(err.Error()).To(ContainSubstring("agent"))
	}
}

func TestWriteMemory_NoDupCheck_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "test write memory",
		Content: memory.ContentFields{
			Behavior: "test",
			Impact:   "test",
			Action:   "test",
		},
	}

	err := cli.ExportWriteMemoryForTest(record, "test write memory", dataDir, true, &buf, "test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("CREATED:"))

	feedbackDir := filepath.Join(dataDir, "memory", "feedback")
	entries, readErr := os.ReadDir(feedbackDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
}

func TestWriteMemory_WithDupCheck_NoToken_SkipsAndCreates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "dup check no token",
		Content: memory.ContentFields{
			Behavior: "test",
			Impact:   "test",
			Action:   "test",
		},
	}

	// noDupCheck=false but no API token available -- should skip check and create
	err := cli.ExportWriteMemoryForTest(record, "dup check no token", dataDir, false, &buf, "test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("CREATED:"))
}
