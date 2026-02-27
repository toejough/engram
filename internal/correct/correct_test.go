package correct_test

// Tests for ARCH-4: Correction Detection
// Traces through ARCH-4 to REQ-13, REQ-14, REQ-18, DES-3, DES-5

import "testing"

// --- Property-based tests (REQ invariants) ---

// T-19: A message with no correction patterns returns empty string (no system reminder).
// Traces: ARCH-4 → REQ-13
func TestCorrectionDetector_NoMatchReturnsEmpty(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-20: A message matching a correction pattern triggers reconciliation.
// Traces: ARCH-4 → REQ-13, REQ-14
func TestCorrectionDetector_MatchTriggersReconciliation(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-21: Each of the 15 initial patterns matches its expected input.
// "^no," matches "no, use specific files". "^wait" matches "wait, that's wrong". Etc.
// Traces: ARCH-4 → REQ-13
func TestCorrectionDetector_AllFifteenPatternsMatch(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-22: When a correction is detected, the correction is recorded to the session log
// so session-end extraction can deduplicate against it.
// Traces: ARCH-4 → REQ-18 (dedup support)
func TestCorrectionDetector_MatchRecordsToSessionLog(t *testing.T) {
	t.Skip("RED: not implemented")
}

// --- Example-based tests (DES scenarios) ---

// T-23: Correction overlaps existing memory → system reminder says "Enriched:"
// with title, added context, and keywords.
// Traces: ARCH-4 → DES-3 (enriched format)
func TestCorrectionDetector_EnrichedExistingMemory_SystemReminder(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-24: Correction has no overlap → system reminder says "Created:" with title
// and keywords.
// Traces: ARCH-4 → DES-3 (created format)
func TestCorrectionDetector_CreatedNewMemory_SystemReminder(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-25: "remember to run tests" matches \bremember\s+(that|to) → memory created,
// no confirmation prompt, system reminder shows correction captured.
// Traces: ARCH-4 → DES-5 (false positive capture-and-decay)
func TestCorrectionDetector_FalsePositive_CapturedAnyway(t *testing.T) {
	t.Skip("RED: not implemented")
}

// --- Integration tests (component boundaries) ---

// T-26: Fake LLM + real SQLite + real pattern corpus file:
// message in → memory in DB + system reminder on stdout + audit entry in log.
// Traces: ARCH-4 (end-to-end)
func TestCorrectionDetector_EndToEnd(t *testing.T) {
	t.Skip("RED: not implemented")
}
