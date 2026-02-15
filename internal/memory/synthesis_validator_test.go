package memory_test

import (
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestValidateSynthesis_GoodPattern(t *testing.T) {
	// Good pattern: actionable, specific, non-redundant
	content := "Always use targ check before committing to catch quality issues early"
	existing := []string{} // No existing patterns

	result := memory.ValidateSynthesis(content, existing)

	if !result.IsActionable {
		t.Errorf("Expected pattern to be actionable, got false")
	}
	if !result.IsSpecific {
		t.Errorf("Expected pattern to be specific, got false")
	}
	if !result.IsNonRedundant {
		t.Errorf("Expected pattern to be non-redundant, got false")
	}
	if result.Quality != 1.0 {
		t.Errorf("Expected quality 1.0, got %.2f", result.Quality)
	}
	if len(result.Issues) != 0 {
		t.Errorf("Expected no issues, got %d: %v", len(result.Issues), result.Issues)
	}
}

func TestValidateSynthesis_BadPattern_NotActionable(t *testing.T) {
	// Bad pattern: vague, no imperative keywords
	content := "important pattern for review"
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	if result.IsActionable {
		t.Errorf("Expected pattern to not be actionable, got true")
	}
	if result.Quality >= 0.8 {
		t.Errorf("Expected quality < 0.8, got %.2f", result.Quality)
	}
	if len(result.Issues) == 0 {
		t.Error("Expected issues to be reported, got none")
	}
}

func TestValidateSynthesis_NotSpecific_TooShort(t *testing.T) {
	// Pattern too short (< 50 chars, < 8 words)
	content := "Always use git"
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	if result.IsSpecific {
		t.Errorf("Expected pattern to not be specific (too short), got true")
	}
	if result.Quality >= 0.8 {
		t.Errorf("Expected quality < 0.8, got %.2f", result.Quality)
	}
}

func TestValidateSynthesis_NotSpecific_NoConcreteTools(t *testing.T) {
	// Pattern long enough but no concrete tools/patterns
	content := "Always make sure to do the thing properly when working on stuff that matters significantly"
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	if result.IsSpecific {
		t.Errorf("Expected pattern to not be specific (no concrete tools), got true")
	}
	if result.Quality >= 0.8 {
		t.Errorf("Expected quality < 0.8, got %.2f", result.Quality)
	}
}

func TestValidateSynthesis_Redundant(t *testing.T) {
	// Pattern very similar to existing pattern (high Jaccard similarity > 0.8)
	// 11/13 shared tokens = 0.846 similarity, well above 0.8 threshold
	content := "Always use targ check command before creating commits to ensure code quality"
	existing := []string{
		"Always use targ check command before creating commits to ensure test quality",
	}

	result := memory.ValidateSynthesis(content, existing)

	if result.IsNonRedundant {
		t.Errorf("Expected pattern to be redundant, got non-redundant")
	}
	if result.Quality >= 0.8 {
		t.Errorf("Expected quality < 0.8 due to redundancy, got %.2f", result.Quality)
	}
}

func TestValidateSynthesis_EmptyContent(t *testing.T) {
	// Edge case: empty content
	content := ""
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	if result.IsActionable || result.IsSpecific {
		t.Error("Expected empty content to fail all checks")
	}
	if result.Quality != 0.0 {
		t.Errorf("Expected quality 0.0 for empty content, got %.2f", result.Quality)
	}
}

func TestValidateSynthesis_VeryLongContent(t *testing.T) {
	// Edge case: very long content (should still work if actionable & specific)
	content := "Always run targ check before creating commits because " +
		"this helps catch quality issues early in the development process " +
		"and prevents broken code from entering the repository history " +
		"which would require reverting commits or creating fix commits " +
		"that clutter the git log and make code review more difficult"
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	// Long content with actionable keyword and concrete tool should pass
	if !result.IsActionable {
		t.Error("Expected long actionable pattern to be marked actionable")
	}
	if !result.IsSpecific {
		t.Error("Expected long specific pattern to be marked specific")
	}
}

func TestValidateSynthesis_QualityCalculation(t *testing.T) {
	// Test quality calculation: 2/3 criteria pass = 0.67 quality
	content := "Never commit code that breaks" // Actionable (never), but not specific (< 50 chars, no tools)
	existing := []string{}

	result := memory.ValidateSynthesis(content, existing)

	expectedQuality := 2.0 / 3.0 // Actionable + non-redundant = 2/3
	tolerance := 0.01
	if result.Quality < expectedQuality-tolerance || result.Quality > expectedQuality+tolerance {
		t.Errorf("Expected quality ~%.2f, got %.2f", expectedQuality, result.Quality)
	}
}

func TestValidateSynthesis_MultipleExistingPatterns(t *testing.T) {
	// Test against multiple existing patterns
	content := "Use go test -tags sqlite_fts5 for testing"
	existing := []string{
		"Always run unit tests before committing",
		"Never skip integration tests",
		"Ensure all tests pass before creating PR",
	}

	result := memory.ValidateSynthesis(content, existing)

	// Should be non-redundant (different topic)
	if !result.IsNonRedundant {
		t.Error("Expected pattern to be non-redundant against different topics")
	}
}

func TestValidateSynthesis_JaccardSimilarityThreshold(t *testing.T) {
	// Test Jaccard similarity threshold (0.8)
	// Patterns with < 80% token overlap should be non-redundant
	content := "Always validate input data before processing API requests"
	existing := []string{
		"Always validate user input in web forms", // Some overlap but < 80%
	}

	result := memory.ValidateSynthesis(content, existing)

	// Should be non-redundant (< 0.8 similarity)
	if !result.IsNonRedundant {
		t.Error("Expected pattern with low similarity to be non-redundant")
	}
}
