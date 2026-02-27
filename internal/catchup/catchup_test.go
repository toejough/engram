package catchup_test

// Tests for ARCH-6: Session-End Catch-Up Processor
// Traces through ARCH-6 to REQ-15, DES-8

import "testing"

// --- Property-based tests (REQ invariants) ---

// T-32: When the evaluator finds no missed corrections, no new memories are created.
// Traces: ARCH-6 → REQ-15
func TestCatchupProcessor_NoMissedCorrections_NoNewMemories(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-33: A missed correction goes through the reconciler and produces a memory
// (created or enriched).
// Traces: ARCH-6 → REQ-15
func TestCatchupProcessor_MissedCorrectionReconciled(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-34: A correction phrase extracted from a missed correction is appended to the
// pattern corpus as a candidate (not active), not as an immediately active pattern.
// Traces: ARCH-6 → REQ-15
func TestCatchupProcessor_NewPatternAddedAsCandidate(t *testing.T) {
	t.Skip("RED: not implemented")
}

// --- Example-based tests (DES scenarios) ---

// T-35: Session with "you didn't shut them down" missed by inline detection →
// memory created with title about shutting down teammates + \byou didn't\b
// added as candidate pattern.
// Traces: ARCH-6 → DES-8
func TestCatchupProcessor_CorpusGrowth_Scenario(t *testing.T) {
	t.Skip("RED: not implemented")
}
