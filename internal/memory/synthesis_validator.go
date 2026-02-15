package memory

import (
	"strings"
)

// SynthesisValidation represents the validation result for a synthesized pattern.
type SynthesisValidation struct {
	Content         string
	IsActionable    bool
	IsSpecific      bool
	IsNonRedundant  bool
	Quality         float64
	Issues          []string
}

// ValidateSynthesis validates a synthesized pattern for actionability, specificity, and non-redundancy.
// Returns a SynthesisValidation struct with quality score (0.0-1.0).
// Quality floor is 0.8 - patterns below this should be rejected.
func ValidateSynthesis(content string, existingPatterns []string) SynthesisValidation {
	result := SynthesisValidation{
		Content:        content,
		IsActionable:   false,
		IsSpecific:     false,
		IsNonRedundant: true, // Default to true, set false if redundant
		Issues:         []string{},
	}

	// Special case: empty content fails all checks
	if content == "" {
		result.IsNonRedundant = false
		result.Issues = append(result.Issues, "Pattern is empty")
		result.Quality = 0.0
		return result
	}

	// Check actionability: contains imperative keywords
	result.IsActionable = isActionable(content)
	if !result.IsActionable {
		result.Issues = append(result.Issues, "Pattern lacks imperative keywords (always, never, use, run, add, remove, ensure, must)")
	}

	// Check specificity: >50 chars, >8 words, mentions concrete tools/patterns
	result.IsSpecific = isSpecific(content)
	if !result.IsSpecific {
		result.Issues = append(result.Issues, "Pattern is not specific enough (needs >50 chars, >8 words, and concrete tools/patterns)")
	}

	// Check non-redundancy: Jaccard similarity <0.8 with existing patterns
	if len(existingPatterns) > 0 {
		for _, existing := range existingPatterns {
			similarity := jaccardSimilarity(content, existing)
			if similarity >= 0.8 {
				result.IsNonRedundant = false
				result.Issues = append(result.Issues, "Pattern is redundant with existing pattern")
				break
			}
		}
	}

	// Calculate quality score: number of passing criteria / 3
	passingCriteria := 0
	if result.IsActionable {
		passingCriteria++
	}
	if result.IsSpecific {
		passingCriteria++
	}
	if result.IsNonRedundant {
		passingCriteria++
	}
	result.Quality = float64(passingCriteria) / 3.0

	return result
}

// isActionable checks if content contains imperative keywords.
func isActionable(content string) bool {
	if content == "" {
		return false
	}

	lower := strings.ToLower(content)
	actionableKeywords := []string{
		"always",
		"never",
		"use ",
		"run ",
		"add ",
		"remove ",
		"ensure ",
		"must ",
	}

	for _, keyword := range actionableKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

// isSpecific checks if content is specific enough:
// - >50 characters
// - >8 words
// - Mentions concrete tools/patterns
func isSpecific(content string) bool {
	if content == "" {
		return false
	}

	// Check length (>50 chars)
	if len(content) <= 50 {
		return false
	}

	// Check word count (>8 words)
	words := strings.Fields(content)
	if len(words) <= 8 {
		return false
	}

	// Check for concrete tools/patterns (specific tool names, file types, commands)
	// Use word boundaries to avoid false matches (e.g., "the" shouldn't match "the")
	lower := strings.ToLower(content)

	// Tool and language names
	toolKeywords := []string{
		"targ", "mage", "git", "docker", "npm", "pytest",
		"python", "javascript", "typescript", "rust", "java", "golang",
	}

	// File extensions (with dots to be specific)
	fileExts := []string{
		".go", ".py", ".js", ".ts", ".rs", ".java",
	}

	// Technical terms (must be whole words)
	technicalTerms := []string{
		" api ", " database ", " sql ", " http ", " cli ", " config ",
		" test ", " lint ", " build ", " deploy ", " commit ", " merge ",
	}

	// Check tool/language keywords
	for _, tool := range toolKeywords {
		if strings.Contains(lower, tool) {
			return true
		}
	}

	// Check file extensions
	for _, ext := range fileExts {
		if strings.Contains(lower, ext) {
			return true
		}
	}

	// Check technical terms (with spaces to ensure word boundaries)
	lowerWithSpaces := " " + lower + " "
	for _, term := range technicalTerms {
		if strings.Contains(lowerWithSpaces, term) {
			return true
		}
	}

	return false
}

// jaccardSimilarity calculates token-based Jaccard similarity between two strings.
// Uses a combination of word tokens and character trigrams to handle variations.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func jaccardSimilarity(s1, s2 string) float64 {
	if s1 == "" && s2 == "" {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Tokenize and normalize
	tokens1 := tokenize(s1)
	tokens2 := tokenize(s2)

	// Generate character trigrams for fuzzy matching
	trigrams1 := generateTrigrams(strings.ToLower(s1))
	trigrams2 := generateTrigrams(strings.ToLower(s2))

	// Build token sets
	tokenSet1 := make(map[string]bool)
	for _, token := range tokens1 {
		tokenSet1[token] = true
	}

	tokenSet2 := make(map[string]bool)
	for _, token := range tokens2 {
		tokenSet2[token] = true
	}

	// Build trigram sets
	trigramSet1 := make(map[string]bool)
	for _, trigram := range trigrams1 {
		trigramSet1[trigram] = true
	}

	trigramSet2 := make(map[string]bool)
	for _, trigram := range trigrams2 {
		trigramSet2[trigram] = true
	}

	// Calculate token-based Jaccard similarity
	tokenIntersection := 0
	for token := range tokenSet1 {
		if tokenSet2[token] {
			tokenIntersection++
		}
	}
	tokenUnion := len(tokenSet1) + len(tokenSet2) - tokenIntersection
	tokenSimilarity := 0.0
	if tokenUnion > 0 {
		tokenSimilarity = float64(tokenIntersection) / float64(tokenUnion)
	}

	// Calculate trigram-based Jaccard similarity
	trigramIntersection := 0
	for trigram := range trigramSet1 {
		if trigramSet2[trigram] {
			trigramIntersection++
		}
	}
	trigramUnion := len(trigramSet1) + len(trigramSet2) - trigramIntersection
	trigramSimilarity := 0.0
	if trigramUnion > 0 {
		trigramSimilarity = float64(trigramIntersection) / float64(trigramUnion)
	}

	// Weighted average: 70% token-based, 30% trigram-based
	// This balances exact word matching with fuzzy character-level matching
	return 0.7*tokenSimilarity + 0.3*trigramSimilarity
}

// tokenize splits a string into normalized tokens (lowercase words).
func tokenize(s string) []string {
	lower := strings.ToLower(s)
	words := strings.Fields(lower)

	// Remove common punctuation
	var tokens []string
	for _, word := range words {
		cleaned := strings.Trim(word, ".,;:!?\"'()[]{}")
		if cleaned != "" {
			tokens = append(tokens, cleaned)
		}
	}

	return tokens
}

// generateTrigrams generates character trigrams from a string for fuzzy matching.
func generateTrigrams(s string) []string {
	// Remove spaces for character-level analysis
	cleaned := strings.ReplaceAll(s, " ", "")

	if len(cleaned) < 3 {
		return []string{cleaned}
	}

	trigrams := make([]string, 0, len(cleaned)-2)
	for i := 0; i <= len(cleaned)-3; i++ {
		trigrams = append(trigrams, cleaned[i:i+3])
	}

	return trigrams
}
