package surface

// GenFactor returns the BM25 relevance penalty factor for a memory based on its
// generalizability and whether it belongs to the current project.
// Same-project or missing slug = 1.0 (no penalty).
func GenFactor(generalizability int, memProject, currentProject string) float64 {
	if memProject == "" || currentProject == "" || memProject == currentProject {
		return 1.0
	}

	// penaltyByGeneralizability maps generalizability score (0–5) to a BM25 penalty factor.
	penaltyByGeneralizability := [6]float64{
		0.5,  // 0: unset — conservative default
		0.05, // 1: this-project-only
		0.2,  // 2: narrow
		0.5,  // 3: moderate
		0.8,  // 4: similar projects
		1.0,  // 5: universal
	}

	if generalizability < 0 || generalizability > 5 {
		return penaltyByGeneralizability[0]
	}

	return penaltyByGeneralizability[generalizability]
}
