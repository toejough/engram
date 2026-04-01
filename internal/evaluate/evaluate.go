package evaluate

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// Exported constants.
const (
	VerdictFollowed    Verdict = "FOLLOWED"
	VerdictIrrelevant  Verdict = "IRRELEVANT"
	VerdictNotFollowed Verdict = "NOT_FOLLOWED"
	VerdictUnknown     Verdict = "UNKNOWN"
)

// Evaluator runs pending memory evaluations using a Haiku caller.
type Evaluator struct {
	caller         anthropic.CallerFunc
	modifier       memory.ModifyFunc
	promptTemplate string
	model          string
}

// New creates an Evaluator with the provided dependencies.
func New(caller anthropic.CallerFunc, modifier memory.ModifyFunc, promptTemplate, model string) *Evaluator {
	return &Evaluator{
		caller:         caller,
		modifier:       modifier,
		promptTemplate: promptTemplate,
		model:          model,
	}
}

// Run evaluates each PendingMemory against the provided transcript and returns results.
func (e *Evaluator) Run(ctx context.Context, memories []PendingMemory, transcript string) []Result {
	results := make([]Result, 0, len(memories))

	for _, pending := range memories {
		result := e.evaluate(ctx, pending, transcript)
		results = append(results, result)
	}

	return results
}

func (e *Evaluator) evaluate(ctx context.Context, pending PendingMemory, transcript string) Result {
	memoryName := memory.NameFromPath(pending.Path)

	userPrompt := buildPrompt(e.promptTemplate, pending.Record, transcript)

	response, err := e.caller(ctx, e.model, "", userPrompt)
	if err != nil {
		return Result{
			MemoryPath: pending.Path,
			MemoryName: memoryName,
			Verdict:    VerdictUnknown,
			Err:        fmt.Errorf("calling haiku for %s: %w", pending.Path, err),
		}
	}

	verdict := parseVerdict(response)

	if verdict == VerdictUnknown {
		return Result{
			MemoryPath: pending.Path,
			MemoryName: memoryName,
			Verdict:    VerdictUnknown,
			Err:        nil,
		}
	}

	modErr := e.modifier(pending.Path, func(record *memory.MemoryRecord) {
		applyVerdict(record, pending.Eval, verdict)
	})
	if modErr != nil {
		return Result{
			MemoryPath: pending.Path,
			MemoryName: memoryName,
			Verdict:    verdict,
			Err:        fmt.Errorf("updating memory %s: %w", pending.Path, modErr),
		}
	}

	return Result{
		MemoryPath: pending.Path,
		MemoryName: memoryName,
		Verdict:    verdict,
		Err:        nil,
	}
}

// Result holds the outcome of evaluating one PendingMemory.
type Result struct {
	MemoryPath string
	MemoryName string
	Verdict    Verdict
	Err        error
}

// Verdict represents the outcome of a memory evaluation.
type Verdict string

// applyVerdict increments the appropriate counter and removes the matched pending eval.
func applyVerdict(record *memory.MemoryRecord, eval memory.PendingEvaluation, verdict Verdict) {
	switch verdict {
	case VerdictFollowed:
		record.FollowedCount++
	case VerdictNotFollowed:
		record.NotFollowedCount++
	case VerdictIrrelevant:
		record.IrrelevantCount++
	case VerdictUnknown:
		// Unknown verdicts are not applied — no counter is incremented.
		return
	}

	record.PendingEvaluations = removePendingEval(record.PendingEvaluations, eval)
}

// buildPrompt substitutes memory fields and transcript into the prompt template.
func buildPrompt(template string, record *memory.MemoryRecord, transcript string) string {
	prompt := template
	prompt = strings.ReplaceAll(prompt, "{situation}", record.Situation)
	prompt = strings.ReplaceAll(prompt, "{behavior}", record.Behavior)
	prompt = strings.ReplaceAll(prompt, "{action}", record.Action)
	prompt = strings.ReplaceAll(prompt, "{transcript}", transcript)

	return prompt
}

// parseVerdict normalizes a Haiku response string to a Verdict.
func parseVerdict(response string) Verdict {
	normalized := strings.ToUpper(strings.TrimSpace(response))

	switch normalized {
	case string(VerdictFollowed):
		return VerdictFollowed
	case string(VerdictNotFollowed):
		return VerdictNotFollowed
	case string(VerdictIrrelevant):
		return VerdictIrrelevant
	default:
		return VerdictUnknown
	}
}

// removePendingEval filters out the first matching evaluation entry.
func removePendingEval(evals []memory.PendingEvaluation, target memory.PendingEvaluation) []memory.PendingEvaluation {
	result := make([]memory.PendingEvaluation, 0, len(evals))

	removed := false

	for _, eval := range evals {
		if !removed && eval.SessionID == target.SessionID && eval.SurfacedAt == target.SurfacedAt {
			removed = true

			continue
		}

		result = append(result, eval)
	}

	return result
}
