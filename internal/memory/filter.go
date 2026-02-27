package memory

// FilterResult is returned by the Filter() method on LLMExtractor.
// Each candidate memory gets a binary relevant/not-relevant decision plus classification tag.
type FilterResult struct {
	MemoryID         int64
	Content          string
	Relevant         bool
	Tag              string  // "relevant", "noise", "should-be-hook", "should-be-earlier"
	RelevanceScore   float64 // 0.0-1.0 confidence
	ShouldSynthesize bool
	MemoryType       string
}
