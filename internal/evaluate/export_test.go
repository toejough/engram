package evaluate

import "engram/internal/memory"

// ExportApplyVerdictDirect calls applyVerdict and exposes it for whitebox testing.
func ExportApplyVerdictDirect(record *memory.MemoryRecord, eval memory.PendingEvaluation, verdict Verdict) {
	applyVerdict(record, eval, verdict)
}

// ExportBuildPrompt exposes buildPrompt for whitebox testing.
func ExportBuildPrompt(template string, record *memory.MemoryRecord, transcript string) string {
	return buildPrompt(template, record, transcript)
}

// ExportMemoryNameFromPath exposes memory.NameFromPath for whitebox testing.
func ExportMemoryNameFromPath(path string) string {
	return memory.NameFromPath(path)
}

// ExportParseVerdict exposes parseVerdict for whitebox testing.
func ExportParseVerdict(response string) Verdict {
	return parseVerdict(response)
}

// ExportRemovePendingEval exposes removePendingEval for whitebox testing.
func ExportRemovePendingEval(
	evals []memory.PendingEvaluation,
	target memory.PendingEvaluation,
) []memory.PendingEvaluation {
	return removePendingEval(evals, target)
}
