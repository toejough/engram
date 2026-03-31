package surface

// GenFactor returns the BM25 relevance penalty factor for a memory based on
// whether it's project-scoped and whether it belongs to the current project.
// Same-project, unscoped, or missing slug = 1.0 (no penalty).
// Cross-project project-scoped = 0.0 (full penalty).
func GenFactor(projectScoped bool, memProject, currentProject string) float64 {
	if !projectScoped {
		return 1.0
	}

	if memProject == "" || currentProject == "" || memProject == currentProject {
		return 1.0
	}

	// Project-scoped memory from a different project: full penalty.
	return 0.0
}
