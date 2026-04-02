package evaluate_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/evaluate"
	"engram/internal/memory"
)

func TestRun_EmptyMemories_ReturnsEmptyResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerCalled := false

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		callerCalled = true

		return "FOLLOWED", nil
	}

	modifier := func(_ string, _ func(*memory.MemoryRecord)) error {
		return nil
	}

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, testModel)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{}, "transcript")

	g.Expect(results).To(BeEmpty())
	g.Expect(callerCalled).To(BeFalse())
}

func TestRun_EvaluatesMemoriesConcurrently(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record1 := &memory.MemoryRecord{Situation: "s1", Behavior: "b1", Action: "a1"}
	eval1 := makeEval("session-1")
	record1.PendingEvaluations = []memory.PendingEvaluation{eval1}

	record2 := &memory.MemoryRecord{Situation: "s2", Behavior: "b2", Action: "a2"}
	eval2 := makeEval("session-2")
	record2.PendingEvaluations = []memory.PendingEvaluation{eval2}

	var barrier sync.WaitGroup
	barrier.Add(2)

	caller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		barrier.Done()
		barrier.Wait()

		if strings.Contains(userPrompt, "s1") {
			return "FOLLOWED", nil
		}

		return "NOT_FOLLOWED", nil
	}

	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		if path == "/mem/mem1.toml" {
			mutate(record1)
		} else {
			mutate(record2)
		}

		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, testModel)

	memories := []evaluate.PendingMemory{
		makePendingMemory("/mem/mem1.toml", record1, eval1),
		makePendingMemory("/mem/mem2.toml", record2, eval2),
	}

	results := evaluator.Run(ctx, memories, "transcript")

	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(results[1].Verdict).To(Equal(evaluate.VerdictNotFollowed))
}

func TestRun_HaikuErrorOnFirstDoesNotBlockSecond(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record1 := &memory.MemoryRecord{Situation: "s1", Behavior: "b1", Action: "a1"}
	eval1 := makeEval("session-1")
	record1.PendingEvaluations = []memory.PendingEvaluation{eval1}

	record2 := &memory.MemoryRecord{Situation: "s2", Behavior: "b2", Action: "a2"}
	eval2 := makeEval("session-2")
	record2.PendingEvaluations = []memory.PendingEvaluation{eval2}

	capMod2 := &captureModifier{record: record2}

	caller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		if strings.Contains(userPrompt, "s1") {
			return "", errors.New("API failure")
		}

		return "FOLLOWED", nil
	}

	modifier := func(_ string, mutate func(*memory.MemoryRecord)) error {
		capMod2.called = true
		mutate(capMod2.record)

		return nil
	}

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, testModel)

	memories := []evaluate.PendingMemory{
		makePendingMemory("/mem/mem1.toml", record1, eval1),
		makePendingMemory("/mem/mem2.toml", record2, eval2),
	}

	results := evaluator.Run(context.Background(), memories, "transcript")

	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0].Err).To(HaveOccurred())
	g.Expect(results[1].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(record2.FollowedCount).To(Equal(1))
}

func TestRun_HaikuError_ResultHasError_NoCounterChange(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	apiErr := errors.New("haiku API error")
	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", apiErr
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Err).To(HaveOccurred())
	g.Expect(results[0].Err.Error()).To(ContainSubstring("calling haiku"))
	g.Expect(capMod.called).To(BeFalse())
	g.Expect(record.FollowedCount).To(Equal(0))
	g.Expect(record.PendingEvaluations).To(HaveLen(1))
}

func TestRun_HaikuReturnsFollowedLowercase_ParsesCorrectly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}
	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "followed", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(record.FollowedCount).To(Equal(1))
}

func TestRun_HaikuReturnsFollowedWithWhitespace_ParsesCorrectly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}
	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "  followed  ", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(record.FollowedCount).To(Equal(1))
}

// Exported Functions (test functions).

func TestRun_HaikuReturnsFollowed_IncrementsFollowedCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "committing code",
		Behavior:  "amending pushed commits",
		Action:    "use /commit skill",
	}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "FOLLOWED", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/commit.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "some transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(results[0].Err).ToNot(HaveOccurred())
	g.Expect(capMod.called).To(BeTrue())
	g.Expect(record.FollowedCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestRun_HaikuReturnsGarbage_VerdictUnknown_NoCounterChange(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation:        "s",
		Behavior:         "b",
		Action:           "a",
		FollowedCount:    0,
		NotFollowedCount: 0,
		IrrelevantCount:  0,
	}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "I think it was followed", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictUnknown))
	g.Expect(results[0].Err).ToNot(HaveOccurred())
	g.Expect(capMod.called).To(BeFalse())
	g.Expect(record.FollowedCount).To(Equal(0))
	g.Expect(record.NotFollowedCount).To(Equal(0))
	g.Expect(record.IrrelevantCount).To(Equal(0))
	g.Expect(record.PendingEvaluations).To(HaveLen(1))
}

func TestRun_HaikuReturnsIrrelevant_IncrementsIrrelevantCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "committing code",
		Behavior:  "amending pushed commits",
		Action:    "use /commit skill",
	}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "IRRELEVANT", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/commit.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "some transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictIrrelevant))
	g.Expect(results[0].Err).ToNot(HaveOccurred())
	g.Expect(capMod.called).To(BeTrue())
	g.Expect(record.IrrelevantCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestRun_HaikuReturnsNotFollowed_IncrementsNotFollowedCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "committing code",
		Behavior:  "amending pushed commits",
		Action:    "use /commit skill",
	}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "NOT_FOLLOWED", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/commit.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "some transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictNotFollowed))
	g.Expect(results[0].Err).ToNot(HaveOccurred())
	g.Expect(capMod.called).To(BeTrue())
	g.Expect(record.NotFollowedCount).To(Equal(1))
	g.Expect(record.PendingEvaluations).To(BeEmpty())
}

func TestRun_MultipleMemories_EachEvaluatedIndependently(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record1 := &memory.MemoryRecord{Situation: "s1", Behavior: "b1", Action: "a1"}
	eval1 := makeEval("session-1")
	record1.PendingEvaluations = []memory.PendingEvaluation{eval1}

	record2 := &memory.MemoryRecord{Situation: "s2", Behavior: "b2", Action: "a2"}
	eval2 := makeEval("session-2")
	record2.PendingEvaluations = []memory.PendingEvaluation{eval2}

	capMod1 := &captureModifier{record: record1}
	capMod2 := &captureModifier{record: record2}

	caller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		if strings.Contains(userPrompt, "s1") {
			return "FOLLOWED", nil
		}

		return "NOT_FOLLOWED", nil
	}

	modifier := func(path string, mutate func(*memory.MemoryRecord)) error {
		if path == "/mem/mem1.toml" {
			capMod1.called = true
			mutate(capMod1.record)
		} else {
			capMod2.called = true
			mutate(capMod2.record)
		}

		return nil
	}

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, testModel)

	memories := []evaluate.PendingMemory{
		makePendingMemory("/mem/mem1.toml", record1, eval1),
		makePendingMemory("/mem/mem2.toml", record2, eval2),
	}

	results := evaluator.Run(context.Background(), memories, "transcript")

	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(results[1].Verdict).To(Equal(evaluate.VerdictNotFollowed))
	g.Expect(record1.FollowedCount).To(Equal(1))
	g.Expect(record2.NotFollowedCount).To(Equal(1))
}

func TestRun_PassesCorrectModelToHaiku(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}
	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	var capturedModel string

	caller := func(_ context.Context, model, _, _ string) (string, error) {
		capturedModel = model

		return "FOLLOWED", nil
	}

	modifier := func(_ string, mutate func(*memory.MemoryRecord)) error {
		mutate(record)

		return nil
	}

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, "specific-model")
	mem := makePendingMemory("/mem/test.toml", record, eval)

	_ = evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(capturedModel).To(Equal("specific-model"))
}

func TestRun_RemovePendingEval_OnlyRemovesMatchingEntry(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	otherEval := memory.PendingEvaluation{
		SessionID:  "session-other",
		SurfacedAt: "2026-06-01T00:00:00Z",
	}
	targetEval := memory.PendingEvaluation{
		SessionID:  testSessionID,
		SurfacedAt: testSurfacedAt,
	}

	record := &memory.MemoryRecord{
		Situation: "s",
		Behavior:  "b",
		Action:    "a",
		PendingEvaluations: []memory.PendingEvaluation{
			otherEval,
			targetEval,
		},
	}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "FOLLOWED", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, targetEval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Verdict).To(Equal(evaluate.VerdictFollowed))
	g.Expect(record.PendingEvaluations).To(HaveLen(1))
	g.Expect(record.PendingEvaluations[0].SessionID).To(Equal("session-other"))
}

func TestRun_ResultContainsMemoryPathAndName(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{Situation: "s", Behavior: "b", Action: "a"}
	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	capMod := &captureModifier{record: record}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "FOLLOWED", nil
	}

	evaluator := evaluate.New(caller, capMod.modify, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/commit-safety.toml", record, eval)

	results := evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "transcript")

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].MemoryPath).To(Equal("/mem/commit-safety.toml"))
	g.Expect(results[0].MemoryName).To(Equal("commit-safety"))
}

func TestRun_UserPromptContainsMemoryFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "my-situation",
		Behavior:  "my-behavior",
		Action:    "my-action",
	}

	eval := makeEval(testSessionID)
	record.PendingEvaluations = []memory.PendingEvaluation{eval}

	var capturedUserPrompt string

	caller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		capturedUserPrompt = userPrompt

		return "FOLLOWED", nil
	}

	modifier := func(_ string, mutate func(*memory.MemoryRecord)) error {
		mutate(record)

		return nil
	}

	evaluator := evaluate.New(caller, modifier, testPromptTemplate, testModel)
	mem := makePendingMemory("/mem/test.toml", record, eval)

	_ = evaluator.Run(context.Background(), []evaluate.PendingMemory{mem}, "the-transcript")

	g.Expect(capturedUserPrompt).To(ContainSubstring("my-situation"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("my-behavior"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("my-action"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("the-transcript"))
}

// unexported constants.
const (
	testModel          = "test-model"
	testPromptTemplate = "situation:{situation} behavior:{behavior} action:{action} transcript:{transcript}"
	testSessionID      = "session-abc"
	testSurfacedAt     = "2026-01-01T00:00:00Z"
)

// Unexported types.

// captureModifier records calls to modifier and applies the mutation to the record.
type captureModifier struct {
	called bool
	record *memory.MemoryRecord
}

// Unexported functions.

func (c *captureModifier) modify(_ string, mutate func(*memory.MemoryRecord)) error {
	c.called = true
	mutate(c.record)

	return nil
}

func makeEval(sessionID string) memory.PendingEvaluation {
	return memory.PendingEvaluation{
		SessionID:  sessionID,
		SurfacedAt: testSurfacedAt,
	}
}

func makePendingMemory(path string, record *memory.MemoryRecord, eval memory.PendingEvaluation) evaluate.PendingMemory {
	return evaluate.PendingMemory{
		Path:   path,
		Record: record,
		Eval:   eval,
	}
}
