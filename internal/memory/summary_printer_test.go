package memory_test

import (
	"strings"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestPrintSessionSummary_EmptySession(t *testing.T) {
	summary := memory.SessionSummary{
		SessionID:   "empty-session",
		ExtractedAt: time.Now(),
		Learnings:   []memory.LearningItem{},
	}

	var output strings.Builder
	memory.PrintSessionSummary(summary, &output)

	result := output.String()

	// Should still print header but indicate no learnings
	if !strings.Contains(result, "Learning Summary") {
		t.Errorf("Expected header even for empty session, got:\n%s", result)
	}

	if !strings.Contains(result, "No new learnings") && !strings.Contains(result, "0 items") {
		t.Errorf("Expected indication of no learnings, got:\n%s", result)
	}
}

func TestPrintSessionSummary_MultipleTypes(t *testing.T) {
	summary := memory.SessionSummary{
		SessionID:   "multi-type-session",
		ExtractedAt: time.Now(),
		Learnings: []memory.LearningItem{
			{Type: "correction", Content: "Never use git checkout .", Confidence: 1.0},
			{Type: "repeated-pattern", Content: "go test -tags sqlite_fts5", Confidence: 0.7},
			{Type: "error-fix", Content: "Fixed timeout by increasing buffer", Confidence: 0.7},
			{Type: "tool-usage-pattern", Content: "used targ successfully", Confidence: 0.5},
		},
	}

	var output strings.Builder
	memory.PrintSessionSummary(summary, &output)

	result := output.String()

	// All types should be represented
	if !strings.Contains(result, "correction") {
		t.Errorf("Expected correction type, got:\n%s", result)
	}

	if !strings.Contains(result, "pattern") || !strings.Contains(result, "repeated-pattern") {
		t.Errorf("Expected pattern type, got:\n%s", result)
	}

	// All content should be present
	if !strings.Contains(result, "Never use git checkout") {
		t.Errorf("Expected correction content, got:\n%s", result)
	}

	if !strings.Contains(result, "go test -tags sqlite_fts5") {
		t.Errorf("Expected pattern content, got:\n%s", result)
	}
}

func TestPrintSessionSummary_WithLearnings(t *testing.T) {
	summary := memory.SessionSummary{
		SessionID:   "test-session-123",
		ExtractedAt: time.Date(2026, 2, 14, 10, 30, 0, 0, time.UTC),
		Learnings: []memory.LearningItem{
			{
				Type:       "correction",
				Content:    "Use AI-Used trailer, not Co-Authored-By",
				Confidence: 1.0,
			},
			{
				Type:       "repeated-pattern",
				Content:    "chi middleware ordering",
				Confidence: 0.7,
			},
		},
		RetrievalsCount:    14,
		RetrievalsRelevant: 12,
	}

	var output strings.Builder
	memory.PrintSessionSummary(summary, &output)

	result := output.String()

	// Verify format includes header
	if !strings.Contains(result, "Learning Summary") {
		t.Errorf("Expected header 'Learning Summary', got:\n%s", result)
	}

	// Verify actual learning content is included
	if !strings.Contains(result, "Use AI-Used trailer") {
		t.Errorf("Expected correction content, got:\n%s", result)
	}

	if !strings.Contains(result, "chi middleware ordering") {
		t.Errorf("Expected pattern content, got:\n%s", result)
	}

	// Verify confidence values shown
	if !strings.Contains(result, "1.0") {
		t.Errorf("Expected confidence 1.0, got:\n%s", result)
	}

	if !strings.Contains(result, "0.7") {
		t.Errorf("Expected confidence 0.7, got:\n%s", result)
	}

	// Verify retrieval counts
	if !strings.Contains(result, "14") {
		t.Errorf("Expected retrieval count 14, got:\n%s", result)
	}

	if !strings.Contains(result, "12 relevant") {
		t.Errorf("Expected 12 relevant retrievals, got:\n%s", result)
	}
}

func TestPrintSessionSummary_WithSkillCandidates(t *testing.T) {
	summary := memory.SessionSummary{
		SessionID:   "test-session",
		ExtractedAt: time.Now(),
		Learnings: []memory.LearningItem{
			{
				Type:       "correction",
				Content:    "Always use -tags sqlite_fts5",
				Confidence: 1.0,
			},
		},
		SkillCandidates: []string{"go-test-tags"},
	}

	var output strings.Builder
	memory.PrintSessionSummary(summary, &output)

	result := output.String()

	// Verify skill candidate shown
	if !strings.Contains(result, "go-test-tags") {
		t.Errorf("Expected skill candidate 'go-test-tags', got:\n%s", result)
	}

	// Verify optimization prompt
	if !strings.Contains(result, "optimize") {
		t.Errorf("Expected mention of optimize command, got:\n%s", result)
	}
}
