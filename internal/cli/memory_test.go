package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestCallHaikuForConflicts_PropagatesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errors.New("api down")
	}

	_, err := cli.ExportCallHaikuForConflicts(
		context.Background(), fakeCaller, "idx", "desc",
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("calling Haiku"))
	}
}

func TestCallHaikuForConflicts_ReturnsResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fakeCaller := func(
		_ context.Context, model, systemPrompt, userPrompt string,
	) (string, error) {
		g.Expect(model).To(Equal("claude-haiku-4-5-20251001"))
		g.Expect(systemPrompt).NotTo(BeEmpty())
		g.Expect(userPrompt).To(ContainSubstring("test-index"))
		g.Expect(userPrompt).To(ContainSubstring("test-description"))

		return "NONE", nil
	}

	result, err := cli.ExportCallHaikuForConflicts(
		context.Background(), fakeCaller, "test-index", "test-description",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("NONE"))
}

func TestCheckForConflicts_APIError_NonFatal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	writeTestMemory(t, g, dataDir)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{Type: "fact"}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("api error")
	}

	conflict, checkErr := cli.ExportCheckForConflicts(
		context.Background(), record, dataDir, &buf, fakeCaller, memory.NewLister(),
	)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(conflict).To(BeFalse())
}

func TestCheckForConflicts_NilCaller_SkipsCheck(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{Type: "fact"}

	conflict, err := cli.ExportCheckForConflicts(
		context.Background(), record, t.TempDir(), &buf, nil, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(conflict).To(BeFalse())
}

func TestCheckForConflicts_NoMemories_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{Type: "fact"}
	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NONE", nil
	}

	// Empty data dir — no memories exist
	conflict, err := cli.ExportCheckForConflicts(
		context.Background(), record, t.TempDir(), &buf, fakeCaller, memory.NewLister(),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(conflict).To(BeFalse())
}

func TestCheckForConflicts_WithMemories_FindsDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	writeTestMemory(t, g, dataDir)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		Type: "fact",
		Content: memory.ContentFields{
			Subject: "x", Predicate: "is", Object: "y",
		},
	}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "DUPLICATE: existing", nil
	}

	conflict, checkErr := cli.ExportCheckForConflicts(
		context.Background(), record, dataDir, &buf, fakeCaller, memory.NewLister(),
	)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(conflict).To(BeTrue())
}

func TestCheckForConflicts_WithMemories_NoConflict(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	writeTestMemory(t, g, dataDir)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		Type: "fact",
		Content: memory.ContentFields{
			Subject: "a", Predicate: "is", Object: "b",
		},
	}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NONE", nil
	}

	conflict, checkErr := cli.ExportCheckForConflicts(
		context.Background(), record, dataDir, &buf, fakeCaller, memory.NewLister(),
	)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(conflict).To(BeFalse())
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

func TestParseConflictResponse_IgnoresContradictionLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	got := cli.ExportParseConflictResponse("CONTRADICTION: foo", dataDir, &bytes.Buffer{})
	g.Expect(got).To(BeFalse())
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

func TestParseConflictResponse_RecognizesDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	got := cli.ExportParseConflictResponse("DUPLICATE: foo", dataDir, &bytes.Buffer{})
	g.Expect(got).To(BeTrue())
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

	err := cli.ExportWriteMemoryForTest(context.Background(), record, "test write memory", dataDir, true, &buf, "test")
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

func TestWriteMemory_ReturnsNameAndPersisted_OnSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	rec := &memory.MemoryRecord{
		SchemaVersion: 2,
		Source:        "agent",
		Situation:     "test situation",
		Type:          "feedback",
		Content:       memory.ContentFields{Behavior: "b", Impact: "i", Action: "a"},
	}

	name, persisted, err := cli.ExportWriteMemory(
		context.Background(),
		rec, "test situation",
		&dataDir, true, // noDupCheck
		&bytes.Buffer{}, "test", nil, nil,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(persisted).To(BeTrue())
	g.Expect(name).To(Equal("test-situation"))
}

func TestWriteMemory_ReturnsNotPersisted_OnDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	dupCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "DUPLICATE: existing", nil
	}

	planted := &memory.MemoryRecord{
		SchemaVersion: 2, Source: "agent", Situation: "existing",
		Type: "feedback", Content: memory.ContentFields{Behavior: "b1"},
	}
	_, _, _ = cli.ExportWriteMemory(
		context.Background(),
		planted, "existing", &dataDir, true,
		&bytes.Buffer{}, "test", nil, nil,
	)

	rec := &memory.MemoryRecord{
		SchemaVersion: 2, Source: "agent", Situation: "different",
		Type: "feedback", Content: memory.ContentFields{Behavior: "b2"},
	}

	name, persisted, err := cli.ExportWriteMemory(
		context.Background(),
		rec, "different",
		&dataDir, false,
		&bytes.Buffer{}, "test", dupCaller, memory.NewLister(),
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(persisted).To(BeFalse())
	g.Expect(name).To(BeEmpty())
}

func TestWriteMemory_WithDupCheck_CallerDetectsConflict_SkipsWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	writeTestMemory(t, g, dataDir)

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "fact",
		Source:        "agent",
		Situation:     "test",
		Content: memory.ContentFields{
			Subject: "x", Predicate: "is", Object: "y",
		},
	}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "DUPLICATE: existing", nil
	}

	writeErr := cli.ExportWriteMemoryWithDeps(
		context.Background(), record, "test", dataDir, false, &buf, "test",
		fakeCaller, memory.NewLister(),
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	// Should NOT have created a file since conflict was detected
	g.Expect(buf.String()).NotTo(ContainSubstring("CREATED:"))
}

func TestWriteMemory_WithDupCheck_ListerError_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "lister error",
		Content: memory.ContentFields{
			Behavior: "test", Impact: "test", Action: "test",
		},
	}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NONE", nil
	}

	writeErr := cli.ExportWriteMemoryWithDeps(
		context.Background(), record, "lister error", dataDir, false, &buf, "test",
		fakeCaller, &failingLister{err: errors.New("disk error")},
	)
	g.Expect(writeErr).To(HaveOccurred())

	if writeErr != nil {
		g.Expect(writeErr.Error()).To(ContainSubstring("listing memories"))
	}
}

func TestWriteMemory_WithDupCheck_NoConflict_CreatesFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var buf bytes.Buffer

	record := &memory.MemoryRecord{
		SchemaVersion: 2,
		Type:          "feedback",
		Source:        "human",
		Situation:     "dup check pass",
		Content: memory.ContentFields{
			Behavior: "test", Impact: "test", Action: "test",
		},
	}

	fakeCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NONE", nil
	}

	writeErr := cli.ExportWriteMemoryWithDeps(
		context.Background(), record, "dup check pass", dataDir, false, &buf, "test",
		fakeCaller, memory.NewLister(),
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("CREATED:"))
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
	err := cli.ExportWriteMemoryForTest(context.Background(), record, "dup check no token", dataDir, false, &buf, "test")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(buf.String()).To(ContainSubstring("CREATED:"))
}

// failingLister is a test double that always returns an error.
type failingLister struct {
	err error
}

func (f *failingLister) ListAllMemories(_ string) ([]*memory.Stored, error) {
	return nil, f.err
}

// writeTestMemory creates a fact TOML in the new layout (both feedback and facts dirs)
// so that memory.Lister detects the new layout and finds the test memory.
func writeTestMemory(t *testing.T, g Gomega, dataDir string) {
	t.Helper()

	factsDir := filepath.Join(dataDir, "memory", "facts")
	feedbackDir := filepath.Join(dataDir, "memory", "feedback")

	g.Expect(os.MkdirAll(factsDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(feedbackDir, 0o750)).To(Succeed())

	factContent := `schema_version = 2
type = "fact"
situation = "test"
source = "agent"

[content]
subject = "x"
predicate = "is"
object = "y"
`
	g.Expect(os.WriteFile(
		filepath.Join(factsDir, "existing.toml"), []byte(factContent), 0o640,
	)).To(Succeed())

	// Feedback dir needs at least one TOML for hasNewLayout detection
	feedbackContent := `schema_version = 2
type = "feedback"
situation = "test"
source = "agent"

[content]
behavior = "test"
impact = "test"
action = "test"
`
	g.Expect(os.WriteFile(
		filepath.Join(feedbackDir, "placeholder.toml"), []byte(feedbackContent), 0o640,
	)).To(Succeed())
}
