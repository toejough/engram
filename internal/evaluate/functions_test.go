package evaluate_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/evaluate"
	"engram/internal/memory"
)

// --- applyVerdict ---

func TestApplyVerdict_Followed_IncrementsFollowedCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	eval := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	record := &memory.MemoryRecord{
		FollowedCount:      0,
		PendingEvaluations: []memory.PendingEvaluation{eval},
	}

	evaluate.ExportApplyVerdictDirect(record, eval, evaluate.VerdictFollowed)

	g.Expect(record.FollowedCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestApplyVerdict_Irrelevant_IncrementsIrrelevantCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	eval := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	record := &memory.MemoryRecord{
		IrrelevantCount:    0,
		PendingEvaluations: []memory.PendingEvaluation{eval},
	}

	evaluate.ExportApplyVerdictDirect(record, eval, evaluate.VerdictIrrelevant)

	g.Expect(record.IrrelevantCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestApplyVerdict_NotFollowed_IncrementsNotFollowedCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	eval := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	record := &memory.MemoryRecord{
		NotFollowedCount:   0,
		PendingEvaluations: []memory.PendingEvaluation{eval},
	}

	evaluate.ExportApplyVerdictDirect(record, eval, evaluate.VerdictNotFollowed)

	g.Expect(record.NotFollowedCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestApplyVerdict_Unknown_NoCounterChangeNoPendingRemoval(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	eval := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	record := &memory.MemoryRecord{
		FollowedCount:      2,
		NotFollowedCount:   1,
		IrrelevantCount:    0,
		PendingEvaluations: []memory.PendingEvaluation{eval},
	}

	evaluate.ExportApplyVerdictDirect(record, eval, evaluate.VerdictUnknown)

	g.Expect(record.FollowedCount).To(Equal(2))
	g.Expect(record.NotFollowedCount).To(Equal(1))
	g.Expect(record.IrrelevantCount).To(Equal(0))
	g.Expect(record.PendingEvaluations).To(HaveLen(1))
}

func TestBuildPrompt_EmptyTemplate_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}

	result := evaluate.ExportBuildPrompt("", record, "transcript")

	g.Expect(result).To(Equal(""))
}

func TestBuildPrompt_NoPlaceholders_ReturnsTemplateUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	template := "this has no placeholders"
	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}

	result := evaluate.ExportBuildPrompt(template, record, "transcript")

	g.Expect(result).To(Equal("this has no placeholders"))
}

// --- buildPrompt ---

func TestBuildPrompt_SubstitutesAllPlaceholders(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	template := "sit:{situation} beh:{behavior} act:{action} tr:{transcript}"
	record := &memory.MemoryRecord{
		Situation: "my-situation",
		Behavior:  "my-behavior",
		Action:    "my-action",
	}

	result := evaluate.ExportBuildPrompt(template, record, "my-transcript")

	g.Expect(result).To(Equal("sit:my-situation beh:my-behavior act:my-action tr:my-transcript"))
}

func TestMemoryNameFromPath_DeepPath_ReturnsFilenameOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result := evaluate.ExportMemoryNameFromPath("/a/b/c/d/file.toml")

	g.Expect(result).To(Equal("file"))
}

func TestMemoryNameFromPath_NoExtension_ReturnsBasename(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result := evaluate.ExportMemoryNameFromPath("/some/dir/my-memory")

	g.Expect(result).To(Equal("my-memory"))
}

// --- memoryNameFromPath ---

func TestMemoryNameFromPath_StripsTOMLExtension(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result := evaluate.ExportMemoryNameFromPath("/some/dir/my-memory.toml")

	g.Expect(result).To(Equal("my-memory"))
}

// --- parseVerdict ---

func TestParseVerdict_Followed_ReturnsFollowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("FOLLOWED")).To(Equal(evaluate.VerdictFollowed))
}

func TestParseVerdict_Irrelevant_ReturnsIrrelevant(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("IRRELEVANT")).To(Equal(evaluate.VerdictIrrelevant))
}

func TestParseVerdict_Lowercase_NormalizedCorrectly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("followed")).To(Equal(evaluate.VerdictFollowed))
}

func TestParseVerdict_NotFollowed_ReturnsNotFollowed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("NOT_FOLLOWED")).To(Equal(evaluate.VerdictNotFollowed))
}

func TestParseVerdict_Unknown_ReturnsUnknown(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("some garbage")).To(Equal(evaluate.VerdictUnknown))
}

func TestParseVerdict_WhitespaceAroundResult_Normalized(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(evaluate.ExportParseVerdict("  IRRELEVANT  ")).To(Equal(evaluate.VerdictIrrelevant))
}

func TestRemovePendingEval_DuplicateMatches_RemovesOnlyFirst(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dup := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	evals := []memory.PendingEvaluation{dup, dup}

	result := evaluate.ExportRemovePendingEval(evals, dup)

	g.Expect(result).To(HaveLen(1))
}

func TestRemovePendingEval_EmptySlice_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	target := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}

	result := evaluate.ExportRemovePendingEval([]memory.PendingEvaluation{}, target)

	g.Expect(result).To(BeEmpty())
}

func TestRemovePendingEval_NoMatch_ReturnsAllEntries(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	e1 := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	e2 := memory.PendingEvaluation{SessionID: "s2", SurfacedAt: "2026-02-01T00:00:00Z"}
	target := memory.PendingEvaluation{SessionID: "s3", SurfacedAt: "2026-03-01T00:00:00Z"}

	result := evaluate.ExportRemovePendingEval([]memory.PendingEvaluation{e1, e2}, target)

	g.Expect(result).To(HaveLen(2))
}

// --- removePendingEval ---

func TestRemovePendingEval_RemovesFirstMatch(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	target := memory.PendingEvaluation{SessionID: "s1", SurfacedAt: "2026-01-01T00:00:00Z"}
	other := memory.PendingEvaluation{SessionID: "s2", SurfacedAt: "2026-02-01T00:00:00Z"}
	evals := []memory.PendingEvaluation{target, other}

	result := evaluate.ExportRemovePendingEval(evals, target)

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].SessionID).To(Equal("s2"))
}
