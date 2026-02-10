package memory

// MemoryStoreSemanticMatcher implements SemanticMatcher using the local memory store.
type MemoryStoreSemanticMatcher struct {
	MemoryRoot string
	ModelDir   string
}

// NewMemoryStoreSemanticMatcher creates a new MemoryStoreSemanticMatcher.
func NewMemoryStoreSemanticMatcher(memoryRoot string) *MemoryStoreSemanticMatcher {
	return &MemoryStoreSemanticMatcher{MemoryRoot: memoryRoot}
}

// FindSimilarMemories queries the memory store for semantically similar memories.
func (m *MemoryStoreSemanticMatcher) FindSimilarMemories(text string, threshold float64, limit int) ([]string, error) {
	results, err := Query(QueryOpts{
		Text:       text,
		Limit:      limit,
		MemoryRoot: m.MemoryRoot,
		ModelDir:   m.ModelDir,
	})
	if err != nil {
		return nil, err
	}
	if results == nil || len(results.Results) == 0 {
		return nil, nil
	}

	var matches []string
	for _, r := range results.Results {
		if r.Score >= threshold {
			matches = append(matches, r.Content)
		}
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	if len(matches) == 0 {
		return nil, nil
	}
	return matches, nil
}
