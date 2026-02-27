package reconcile_test

// Tests for ARCH-5: Reconciler (shared component)
// Traces through ARCH-5 to REQ-5, REQ-14, DES-3

import "testing"

// --- Property-based tests (REQ invariants) ---

// T-27: When the store is empty (no existing memories), reconciler always creates.
// Traces: ARCH-5 → REQ-5, REQ-14
func TestReconciler_NoExistingMemories_Creates(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-28: When the overlap gate returns true for a candidate, the reconciler enriches
// the best-scoring candidate rather than creating a new memory.
// Traces: ARCH-5 → REQ-5, REQ-14
func TestReconciler_OverlapGateSaysYes_Enriches(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-29: When the overlap gate returns false for all K candidates, the reconciler
// creates a new memory.
// Traces: ARCH-5 → REQ-5, REQ-14
func TestReconciler_OverlapGateSaysNo_Creates(t *testing.T) {
	t.Skip("RED: not implemented")
}

// T-30: The reconciler passes exactly K candidates to the overlap gate, not more,
// even when FindSimilar could return more.
// Traces: ARCH-5 → REQ-5
func TestReconciler_RespectsKBudget(t *testing.T) {
	t.Skip("RED: not implemented")
}

// --- Example-based tests (DES scenarios) ---

// T-31: After enrichment, the memory has merged keywords (old + new deduplicated)
// and updated enriched_content reflecting the new context.
// Traces: ARCH-5 → DES-3
func TestReconciler_EnrichAddsKeywordsAndContext(t *testing.T) {
	t.Skip("RED: not implemented")
}
