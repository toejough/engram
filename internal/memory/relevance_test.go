package memory_test

import (
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestComputeAverageRetrievalPrecision_EmptyInput(t *testing.T) {
	var scores []memory.RetrievalRelevance

	avg := memory.ComputeAverageRetrievalPrecision(scores)

	if avg != 0.0 {
		t.Errorf("expected 0.0 for empty input, got %f", avg)
	}
}

func TestComputeAverageRetrievalPrecision_MixedScores(t *testing.T) {
	scores := []memory.RetrievalRelevance{
		{Relevant: true, Precision: 1.0},
		{Relevant: false, Precision: 0.0},
		{Relevant: true, Precision: 1.0},
		{Relevant: true, Precision: 1.0},
	}

	avg := memory.ComputeAverageRetrievalPrecision(scores)
	expected := 0.75 // 3/4 = 0.75

	if avg != expected {
		t.Errorf("expected average precision=%f, got %f", expected, avg)
	}
}

func TestScoreRetrievalRelevance_CorrectionFollows_NotRelevant(t *testing.T) {
	// Setup: retrieval at T0, correction at T0+2min on same topic
	now := time.Now()
	retrieval := memory.RetrievalLogEntry{
		Timestamp: now.Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "git commit trailer format",
		Results: []memory.RetrievalResult{
			{ID: 1, Content: "Use AI-Used trailer", Score: 0.95, Tier: "embedding"},
		},
	}

	corrections := []memory.ChangelogEntry{
		{
			Timestamp:       now.Add(2 * time.Minute),
			Action:          "store_correction",
			ContentSummary:  "Use AI-Used trailer not Co-Authored-By",
			DestinationTier: "embedding",
		},
	}
	timeWindow := 10 * time.Minute

	scores := memory.ScoreRetrievalRelevance(retrieval, corrections, timeWindow)

	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}

	if scores[0].Relevant {
		t.Errorf("expected relevant=false when correction follows")
	}

	if scores[0].Precision != 0.0 {
		t.Errorf("expected precision=0.0, got %f", scores[0].Precision)
	}
}

func TestScoreRetrievalRelevance_CorrectionOutsideWindow_StillRelevant(t *testing.T) {
	// Setup: retrieval at T0, correction at T0+15min (outside 10min window)
	now := time.Now()
	retrieval := memory.RetrievalLogEntry{
		Timestamp: now.Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "git commit trailer format",
		Results: []memory.RetrievalResult{
			{ID: 1, Content: "Use AI-Used trailer", Score: 0.95, Tier: "embedding"},
		},
	}

	corrections := []memory.ChangelogEntry{
		{
			Timestamp:       now.Add(15 * time.Minute),
			Action:          "store_correction",
			ContentSummary:  "Use AI-Used trailer not Co-Authored-By",
			DestinationTier: "embedding",
		},
	}
	timeWindow := 10 * time.Minute

	scores := memory.ScoreRetrievalRelevance(retrieval, corrections, timeWindow)

	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}

	if !scores[0].Relevant {
		t.Errorf("expected relevant=true when correction outside time window")
	}

	if scores[0].Precision != 1.0 {
		t.Errorf("expected precision=1.0, got %f", scores[0].Precision)
	}
}

func TestScoreRetrievalRelevance_EmptyResults(t *testing.T) {
	retrieval := memory.RetrievalLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "some query",
		Results:   []memory.RetrievalResult{},
	}

	var corrections []memory.ChangelogEntry

	timeWindow := 10 * time.Minute

	scores := memory.ScoreRetrievalRelevance(retrieval, corrections, timeWindow)

	if len(scores) != 0 {
		t.Errorf("expected 0 scores for empty results, got %d", len(scores))
	}
}

func TestScoreRetrievalRelevance_NoCorrection_IsRelevant(t *testing.T) {
	// Setup: retrieval at T0, no subsequent corrections
	retrieval := memory.RetrievalLogEntry{
		Timestamp: time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "git commit trailer format",
		Results: []memory.RetrievalResult{
			{ID: 1, Content: "Use AI-Used trailer", Score: 0.95, Tier: "embedding"},
		},
	}

	var noCorrections []memory.ChangelogEntry

	timeWindow := 10 * time.Minute

	scores := memory.ScoreRetrievalRelevance(retrieval, noCorrections, timeWindow)

	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}

	if !scores[0].Relevant {
		t.Errorf("expected relevant=true when no corrections follow")
	}

	if scores[0].Precision != 1.0 {
		t.Errorf("expected precision=1.0, got %f", scores[0].Precision)
	}
}
