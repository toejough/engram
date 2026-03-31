package surface

import "engram/internal/memory"

// ApplyColdStartBudget limits the number of unproven (never surfaced) memories.
// Proven memories (SurfacedCount > 0) always pass through.
// Budget of 0 means unlimited.
func ApplyColdStartBudget(candidates []*memory.Stored, budget int) []*memory.Stored {
	if budget == 0 {
		return candidates
	}

	result := make([]*memory.Stored, 0, len(candidates))
	unprovenCount := 0

	for _, candidate := range candidates {
		if candidate.SurfacedCount > 0 {
			result = append(result, candidate)
			continue
		}

		if unprovenCount < budget {
			result = append(result, candidate)
			unprovenCount++
		}
	}

	return result
}
