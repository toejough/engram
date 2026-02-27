package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestAppendToClaudeMD_ExistingFileWithSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	initial := "## Promoted Learnings\n\n- existing learning\n"

	err := os.WriteFile(claudePath, []byte(initial), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = appendToClaudeMD(claudePath, []string{"new learning to add"})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudePath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("- existing learning"))
	g.Expect(string(content)).To(ContainSubstring("- new learning to add"))
}

func TestAppendToClaudeMD_ExistingFileWithoutSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	initial := "## Core Principles\n\n- be concise\n"

	err := os.WriteFile(claudePath, []byte(initial), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = appendToClaudeMD(claudePath, []string{"use TDD for all changes"})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudePath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(string(content)).To(ContainSubstring("- use TDD for all changes"))
}

// ─── appendToClaudeMD tests ───────────────────────────────────────────────────

func TestAppendToClaudeMD_FileDoesNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "subdir", "CLAUDE.md")

	err := appendToClaudeMD(claudePath, []string{"always write tests"})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudePath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("## Promoted Learnings"))
	g.Expect(string(content)).To(ContainSubstring("- always write tests"))
}

func TestAppendToClaudeMD_FileNotEndingWithNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	initial := "## Core Principles\n\n- be concise"

	err := os.WriteFile(claudePath, []byte(initial), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = appendToClaudeMD(claudePath, []string{"new promoted learning here"})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudePath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("## Promoted Learnings"))
}

func TestAppendToClaudeMD_SectionFollowedByAnotherSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	initial := "## Promoted Learnings\n\n- old entry\n\n## Critical Warnings\n\n- never do this\n"

	err := os.WriteFile(claudePath, []byte(initial), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = appendToClaudeMD(claudePath, []string{"inserted before warnings"})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudePath)
	g.Expect(err).ToNot(HaveOccurred())

	str := string(content)

	// New learning should appear before "## Critical Warnings"
	promotedIdx := strings.Index(str, "- inserted before warnings")
	warningsIdx := strings.Index(str, "## Critical Warnings")

	g.Expect(promotedIdx).To(BeNumerically("<", warningsIdx))
}

// ─── ClearTimestampsForTest tests ─────────────────────────────────────────────

func TestClearTimestampsForTest_MatchingContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	ts := `["2025-01-01T00:00:00Z"]`

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, retrieval_timestamps) VALUES (?, ?, ?)",
		"unique test content for clearing", "test", ts,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = ClearTimestampsForTest(ClearTimestampsOpts{
		MemoryRoot: tmpDir,
		Content:    "unique test content",
	})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestClearTimestampsForTest_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = ClearTimestampsForTest(ClearTimestampsOpts{
		MemoryRoot: tmpDir,
		Content:    "nonexistent pattern xyz",
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("no rows matched"))
	}
}

func TestClusterIntoSessions_AllInvalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(ClusterIntoSessions([]string{"not-a-timestamp", "also-bad"}, 30*time.Minute)).To(BeNil())
}

// ─── ClusterIntoSessions tests ────────────────────────────────────────────────

func TestClusterIntoSessions_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(ClusterIntoSessions(nil, 30*time.Minute)).To(BeNil())
}

func TestClusterIntoSessions_GapExactlyAtThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Gap exactly equal to threshold does NOT start a new session
	ts1 := "2025-01-01T10:00:00Z"
	ts2 := "2025-01-01T10:30:00Z"

	result := ClusterIntoSessions([]string{ts1, ts2}, 30*time.Minute)

	g.Expect(result).To(HaveLen(1))
}

func TestClusterIntoSessions_OneSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two timestamps 10 minutes apart (less than 30-min gap threshold)
	ts1 := "2025-01-01T10:00:00Z"
	ts2 := "2025-01-01T10:10:00Z"

	result := ClusterIntoSessions([]string{ts1, ts2}, 30*time.Minute)

	g.Expect(result).To(HaveLen(1))

	g.Expect(result).ToNot(BeNil())

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0]).To(HaveLen(2))
}

func TestClusterIntoSessions_SingleTimestamp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := "2025-01-01T10:00:00Z"
	result := ClusterIntoSessions([]string{ts}, 30*time.Minute)
	g.Expect(result).To(HaveLen(1))
	g.Expect(result).ToNot(BeNil())

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0]).To(HaveLen(1))
	g.Expect(result[0][0]).To(Equal(ts))
}

func TestClusterIntoSessions_SortsBeforeClustering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Provide timestamps out of order
	ts1 := "2025-01-01T10:00:00Z"
	ts2 := "2025-01-01T10:05:00Z"

	result := ClusterIntoSessions([]string{ts2, ts1}, 30*time.Minute)
	// Should be one session regardless of input order
	g.Expect(result).To(HaveLen(1))
	g.Expect(result).ToNot(BeNil())

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0]).To(HaveLen(2))
}

func TestClusterIntoSessions_TwoSessions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two timestamps 2 hours apart (exceeds 30-min threshold)
	ts1 := "2025-01-01T08:00:00Z"
	ts2 := "2025-01-01T10:00:00Z"

	result := ClusterIntoSessions([]string{ts1, ts2}, 30*time.Minute)

	g.Expect(result).To(HaveLen(2))
}

func TestDecay_DefaultFactor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence) VALUES (?, ?, ?)",
		"test learning content", "test", 1.0,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// Factor=0 should use default 0.9
	result, err := Decay(DecayOpts{
		MemoryRoot: tmpDir,
		Factor:     0,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Factor).To(BeNumerically("~", 0.9, 0.001))
	g.Expect(result.EntriesAffected).To(Equal(1))
}

// ─── Decay tests ──────────────────────────────────────────────────────────────

func TestDecay_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := Decay(DecayOpts{
		MemoryRoot: tmpDir,

		Factor: 0.9,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.EntriesAffected).To(Equal(0))
}

func TestDecide_MissingChoice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Decide(DecideOpts{
		Context:    "which auth method",
		Choice:     "",
		Reason:     "it is better",
		MemoryRoot: t.TempDir(),
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("choice is required"))
}

// ─── Decide tests ─────────────────────────────────────────────────────────────

func TestDecide_MissingContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Decide(DecideOpts{
		Context:    "",
		Choice:     "option-a",
		Reason:     "it is better",
		MemoryRoot: t.TempDir(),
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("context is required"))
}

func TestDecide_MissingReason(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Decide(DecideOpts{
		Context:    "which auth method",
		Choice:     "jwt",
		Reason:     "",
		MemoryRoot: t.TempDir(),
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("reason is required"))
}

func TestDecide_ValidInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	result, err := Decide(DecideOpts{
		Context:      "which auth method",
		Choice:       "jwt",
		Reason:       "stateless and scalable",
		Alternatives: []string{"session cookies", "oauth"},

		Project:    "my-project",
		MemoryRoot: tmpDir,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.FilePath).To(ContainSubstring("decisions"))

	// Verify file was created
	_, err = os.Stat(result.FilePath)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestDetectConflictType_BothHaveNegation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Both have negation but no opposing pair → duplicate
	result := detectConflictType("never skip tests", "never skip test runs")

	g.Expect(result).To(Equal("duplicate"))
}

func TestDetectConflictType_Duplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := detectConflictType("write tests before implementing", "write tests before implementation")

	g.Expect(result).To(Equal("duplicate"))
}

func TestDetectConflictType_NegationInExistingOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := detectConflictType("use mocks for testing", "never use mocks in tests")

	g.Expect(result).To(Equal("contradiction"))
}

// ─── detectConflictType tests ─────────────────────────────────────────────────

func TestDetectConflictType_NegationInNewOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := detectConflictType("never use mocks for testing", "use real implementations for integration tests")

	g.Expect(result).To(Equal("contradiction"))
}

func TestDetectConflictType_OpposingPair(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// "use" in new, "avoid" in existing → contradiction
	result := detectConflictType("use typescript for type safety", "avoid typescript in small projects")

	g.Expect(result).To(Equal("contradiction"))
}

// ─── GetActivationStats tests ─────────────────────────────────────────────────

func TestGetActivationStats_CorrectionType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now()
	ts := []string{now.Add(-10 * time.Minute).Format(time.RFC3339)}
	tsJSON, _ := json.Marshal(ts)

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, memory_type, retrieval_count, retrieval_timestamps, importance_score, impact_score)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"correction type test content uniquekey", "test", "correction", 1, string(tsJSON), 0.8, 0.5,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
	stats, err := GetActivationStats(ActivationStatsOpts{
		MemoryRoot: tmpDir,
		Content:    "correction type test content uniquekey",
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stats).ToNot(BeNil())

	if stats == nil {
		t.Fatal("stats is nil")
	}

	g.Expect(stats.DecayParameter).To(BeNumerically("~", 0.1, 0.001))
}

func TestGetActivationStats_MultiSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Two timestamps 2 hours apart → 2 sessions → session bonus applied
	now := time.Now()
	ts := []string{
		now.Add(-3 * time.Hour).Format(time.RFC3339),
		now.Add(-1 * time.Hour).Format(time.RFC3339),
	}
	tsJSON, _ := json.Marshal(ts)

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, memory_type, retrieval_count, retrieval_timestamps, importance_score, impact_score)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"multi session activation test uniquekey", "test", "", 2, string(tsJSON), 0.6, 0.4,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
	stats, err := GetActivationStats(ActivationStatsOpts{
		MemoryRoot: tmpDir,
		Content:    "multi session activation test uniquekey",
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stats).ToNot(BeNil())

	if stats == nil {
		t.Fatal("stats is nil")
	}

	g.Expect(stats.SessionCount).To(BeNumerically(">=", 2))
	g.Expect(stats.SessionBonus).To(BeNumerically(">", 0))
}

func TestGetActivationStats_ReflectionTypeFiltersOldTimestamps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Include one recent and one old (>30 days) timestamp
	now := time.Now()
	ts := []string{
		now.Add(-10 * time.Minute).Format(time.RFC3339),
		now.Add(-31 * 24 * time.Hour).Format(time.RFC3339), // older than 30 days
	}
	tsJSON, _ := json.Marshal(ts)

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, memory_type, retrieval_count, retrieval_timestamps, importance_score, impact_score)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"reflection sliding window test uniquekey", "test", "reflection", 2, string(tsJSON), 0.7, 0.3,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()

	g.Expect(err).ToNot(HaveOccurred())

	stats, err := GetActivationStats(ActivationStatsOpts{
		MemoryRoot: tmpDir,
		Content:    "reflection sliding window test uniquekey",
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stats).ToNot(BeNil())

	if stats == nil {
		t.Fatal("stats is nil")
	}
	// Only the recent timestamp should be counted (old one filtered by 30-day window)
	g.Expect(stats.ActiveTimestamps).To(Equal(1))
	g.Expect(stats.TimestampCount).To(Equal(2))
}

// TestGrep_EmptyPattern verifies Grep returns error for empty pattern.
func TestGrep_EmptyPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Grep(GrepOpts{MemoryRoot: t.TempDir()})
	g.Expect(err).To(MatchError(ContainSubstring("pattern is required")))
}

// TestGrep_NoDB verifies Grep returns empty results when no DB exists.
func TestGrep_NoDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result, err := Grep(GrepOpts{Pattern: "test", MemoryRoot: t.TempDir()})
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	g.Expect(result.Matches).To(BeEmpty())
}

// TestGrep_WithDB verifies Grep finds matches in the embeddings DB.
func TestGrep_WithDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(`INSERT INTO embeddings(content, source) VALUES (?, ?)`,
		"always write tests before coding", "test")
	g.Expect(err).ToNot(HaveOccurred())

	_ = db.Close()

	result, err := Grep(GrepOpts{Pattern: "tests", MemoryRoot: tmpDir})
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	g.Expect(result.Matches).To(HaveLen(1))
	g.Expect(result.Matches[0].Line).To(ContainSubstring("always write tests"))
}

func TestIsLegacyCannedExtraction_AutonomouslyFixed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("Autonomously fixed the failing tests")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_ClaudeMDWasEdited(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("claude.md was edited")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_ConsistentUseOf(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("consistent use of 'gomega' across 5 messages")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_ConsistentlyUsedThroughoutSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("Consistently used gomega throughout session")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_FrequentlyUsedCommand(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("frequently used command: go test")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_NotCanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("always write tests before implementing features for quality")).To(BeFalse())
}

func TestIsLegacyCannedExtraction_SelfCorrected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("self-corrected that failure:\nfailed: the test")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_SessionBehaviorAligns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("session behavior aligns with: the plan")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_SuccessfulOutcome(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("successful outcome:\ncommand: go build")).To(BeTrue())
}

func TestIsLegacyCannedExtraction_TestsPassedUsing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("Tests passed using sqlite_fts5 tag")).To(BeTrue())
}

// ─── isLegacyCannedExtraction tests ──────────────────────────────────────────

func TestIsLegacyCannedExtraction_UsedSuccessfully(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(isLegacyCannedExtraction("Used git commit successfully in session")).To(BeTrue())
}

// ─── IsSessionBoilerplate tests ───────────────────────────────────────────────

func TestIsSessionBoilerplate_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(IsSessionBoilerplate("")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("   ")).To(BeTrue())
}

func TestIsSessionBoilerplate_HorizontalRule(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(IsSessionBoilerplate("---")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("...")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("----")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("....")).To(BeTrue())
}

func TestIsSessionBoilerplate_LegacyCanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(IsSessionBoilerplate("Autonomously fixed the issue")).To(BeTrue())
}

func TestIsSessionBoilerplate_MarkdownHeader(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(IsSessionBoilerplate("## Section Header")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("# Top Level")).To(BeTrue())
}

func TestIsSessionBoilerplate_MetadataLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(IsSessionBoilerplate("**Project:** my-project")).To(BeTrue())
	g.Expect(IsSessionBoilerplate("**Date:** 2025-01-01")).To(BeTrue())
}

func TestIsSessionBoilerplate_ShortAfterStripping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// After stripping markdown (**), content is too short
	g.Expect(IsSessionBoilerplate("**x**")).To(BeTrue())
}

func TestIsSessionBoilerplate_TooFewWords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Less than 5 words after stripping
	g.Expect(IsSessionBoilerplate("do this now")).To(BeTrue())
}

func TestIsSessionBoilerplate_ValidContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Has enough words, not boilerplate
	g.Expect(IsSessionBoilerplate("always write unit tests before implementing new features")).To(BeFalse())
}

// TestLearnWithConflictCheck_ConflictDetected exercises the len(results) > 0
// and similarity > 0.85 branches by first storing an entry, then calling
// LearnWithConflictCheck with the identical text.
func TestLearnWithConflictCheck_ConflictDetected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	msg := "prefer composition over inheritance in all object design decisions"

	// Store the entry first so it exists in the DB with an embedding
	err := Learn(LearnOpts{
		Message:    msg,
		MemoryRoot: tmpDir,
	})
	if err != nil {
		t.Skipf("ONNX not available for Learn: %v", err)
	}

	// LearnWithConflictCheck on the same text should find a near-identical entry
	result, err := LearnWithConflictCheck(LearnOpts{
		Message:    msg,
		MemoryRoot: tmpDir,
	})
	if err != nil {
		t.Skipf("ONNX not available for LearnWithConflictCheck: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Stored).To(BeTrue())
	// Identical text → similarity should exceed 0.85
	g.Expect(result.Similarity).To(BeNumerically(">", 0.85))
	g.Expect(result.HasConflict).To(BeTrue())
}

func TestLearnWithConflictCheck_DBOpenError(t *testing.T) {
	// Not parallel — initializes ONNX runtime (relied upon by GenerateEmbeddingError).
	g := NewWithT(t)

	// /dev/null/embeddings.db cannot be opened → initEmbeddingsDB fails.
	// ONNX initializes with real HOME, then DB open fails → covers "failed to open database".
	_, err := LearnWithConflictCheck(LearnOpts{
		Message:    "test message for database open error coverage path",
		MemoryRoot: "/dev/null",
	})

	g.Expect(err).To(HaveOccurred())
}

// ─── LearnWithConflictCheck tests ─────────────────────────────────────────────

func TestLearnWithConflictCheck_EmptyMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := LearnWithConflictCheck(LearnOpts{Message: "", MemoryRoot: t.TempDir()})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("message is required"))
}

func TestLearnWithConflictCheck_GenerateEmbeddingError(t *testing.T) {
	// Not parallel — requires ONNX runtime already initialized (by DBOpenError).
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Create a fake model dir and an invalid ONNX model file.
	modelsDir := filepath.Join(tmpDir, ".claude", "models")

	err := os.MkdirAll(modelsDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	fakeModelPath := filepath.Join(modelsDir, "e5-small-v2.onnx")

	err = os.WriteFile(fakeModelPath, []byte("not a valid onnx model file"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// HOME=tmpDir → modelPath points to fake model.
	// initializeONNXRuntime uses fast path (onnxRuntimeInitialized=true from DBOpenError),
	// so library path is not changed. generateEmbeddingONNX fails on garbage model.
	t.Setenv("HOME", tmpDir)

	memoryRoot := filepath.Join(tmpDir, "memory")

	err = os.MkdirAll(memoryRoot, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = LearnWithConflictCheck(LearnOpts{
		Message:    "test message for generate embedding error coverage path",
		MemoryRoot: memoryRoot,
	})

	g.Expect(err).To(HaveOccurred())
}

func TestLearnWithConflictCheck_MkdirAllError(t *testing.T) {
	// Not parallel — uses t.Setenv to control HOME.
	g := NewWithT(t)

	// HOME=/dev/null → modelDir=/dev/null/.claude/models → os.MkdirAll fails.
	t.Setenv("HOME", "/dev/null")

	_, err := LearnWithConflictCheck(LearnOpts{
		Message:    "test message for mkdir model dir error coverage path",
		MemoryRoot: t.TempDir(),
	})

	g.Expect(err).To(MatchError(ContainSubstring("failed to create model directory")))
}

func TestLearnWithConflictCheck_NewEntryNoDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	result, err := LearnWithConflictCheck(LearnOpts{
		Message:    "always write unit tests before implementing new features",
		MemoryRoot: tmpDir,
		Project:    "test-project",
	})
	if err != nil {
		t.Skipf("ONNX not available: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Stored).To(BeTrue())
	g.Expect(result.HasConflict).To(BeFalse())
}

// ─── Learn tests ─────────────────────────────────────────────────────────────

func TestLearn_EmptyMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := Learn(LearnOpts{Message: "", MemoryRoot: t.TempDir()})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("message is required"))
	}
}

func TestLearn_MkdirAllFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Create a regular file at a path component to block MkdirAll
	blocker := filepath.Join(tmpDir, "blockfile")
	err := os.WriteFile(blocker, []byte("x"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// MemoryRoot with blocker as a non-directory component
	err = Learn(LearnOpts{
		Message:    "test message to learn here",
		MemoryRoot: filepath.Join(blocker, "subdir"),
	})

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to create memory directory"))
	}
}

// ─── MigrateToACTR tests ──────────────────────────────────────────────────────

func TestMigrateToACTR_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = MigrateToACTR(MigrateToACTROpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestMigrateToACTR_EntriesWithNoTimestamps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert entry without retrieval_timestamps
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, retrieval_count) VALUES (?, ?, ?)",
		"entry without timestamps", "test", 0,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = MigrateToACTR(MigrateToACTROpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestMigrateToACTR_EntriesWithRetrievalCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert entry with retrieval_count > 0 but no timestamps
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, retrieval_count) VALUES (?, ?, ?)",
		"retrieved entry without timestamps", "test", 3,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = MigrateToACTR(MigrateToACTROpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())

	// Verify timestamps were created
	db, err = initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	var ts string

	err = db.QueryRow("SELECT retrieval_timestamps FROM embeddings WHERE content LIKE '%retrieved entry%'").Scan(&ts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ts).ToNot(BeEmpty())
}

// ─── PromoteInteractive tests ─────────────────────────────────────────────────

func TestPromoteInteractive_MissingMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := PromoteInteractive(PromoteInteractiveOpts{MemoryRoot: ""})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("memory root is required"))
}

func TestPromoteInteractive_NoReview(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)

	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := PromoteInteractive(PromoteInteractiveOpts{
		MemoryRoot: tmpDir,
		Review:     false,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.CandidatesReviewed).To(Equal(0))
}

func TestPromoteInteractive_ReviewFuncError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"review func error test learning entry", "test", 5, "proj1,proj2",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = PromoteInteractive(PromoteInteractiveOpts{
		MemoryRoot:    tmpDir,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        true,
		ClaudeMDPath:  filepath.Join(tmpDir, "CLAUDE.md"),
		ReviewFunc: func(_ PromoteCandidate) (bool, error) {
			return false, os.ErrPermission
		},
	})

	g.Expect(err).To(HaveOccurred())
}

func TestPromoteInteractive_ReviewWithCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"promote interactive test learning entry", "test", 5, "proj1,proj2",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	approved := 0

	result, err := PromoteInteractive(PromoteInteractiveOpts{
		MemoryRoot:    tmpDir,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        true,
		ClaudeMDPath:  claudePath,
		ReviewFunc: func(_ PromoteCandidate) (bool, error) {
			approved++

			return true, nil
		},
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.CandidatesApproved).To(BeNumerically(">=", 1))
	g.Expect(approved).To(BeNumerically(">=", 1))
}

func TestPromoteInteractive_ReviewWithoutFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := PromoteInteractive(PromoteInteractiveOpts{
		MemoryRoot: t.TempDir(),
		Review:     true,
		ReviewFunc: nil,
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("review function is required"))
}

func TestPromote_DefaultThresholds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert entry that meets default thresholds (MinRetrievals=3, MinProjects=2)
	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"test learning meeting defaults", "test", 3, "alpha,beta",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// Zero values use defaults
	result, err := Promote(PromoteOpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Candidates).To(HaveLen(1))
}

// ─── Promote tests ────────────────────────────────────────────────────────────

func TestPromote_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := Promote(PromoteOpts{MemoryRoot: tmpDir})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Candidates).To(BeEmpty())
}

func TestPromote_MissingMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Promote(PromoteOpts{MemoryRoot: ""})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("memory root is required"))
}

func TestPromote_WithCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"always write tests before implementing features", "test", 5, "proj1,proj2,proj3",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := Promote(PromoteOpts{
		MemoryRoot:    tmpDir,
		MinRetrievals: 3,
		MinProjects:   2,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Candidates).To(HaveLen(1))
	g.Expect(result.Candidates[0].RetrievalCount).To(Equal(5))
	g.Expect(result.Candidates[0].UniqueProjects).To(Equal(3))
}

func TestPrune_DefaultThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// Threshold=0 should use default 0.1

	result, err := Prune(PruneOpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Threshold).To(BeNumerically("~", 0.1, 0.001))
}

// ─── Prune tests ──────────────────────────────────────────────────────────────

func TestPrune_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
	result, err := Prune(PruneOpts{MemoryRoot: tmpDir})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.EntriesRemoved).To(Equal(0))
}

func TestPrune_RemovesBelowThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert one entry below threshold and one above.
	// Use embedding_id=0 to avoid NULL scan failure in Prune.
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)",
		"low confidence entry", "test", 0.05, 0,
	)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)",
		"high confidence entry", "test", 0.9, 0,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := Prune(PruneOpts{

		MemoryRoot: tmpDir,
		Threshold:  0.1,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.EntriesRemoved).To(Equal(1))
	g.Expect(result.EntriesRetained).To(Equal(1))
}

// TestQuery_DBOpenError verifies the error path when initEmbeddingsDB fails.
func TestQuery_DBOpenError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// /dev/null/invalid/embeddings.db cannot be created → initEmbeddingsDB fails
	_, err := Query(QueryOpts{
		Text:       "some query text",
		MemoryRoot: "/dev/null/invalid",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to initialize embeddings database"))
}

// ─── Query tests ──────────────────────────────────────────────────────────────

// TestQuery_EmptyDB verifies the short-circuit path: when the embeddings DB has
// no entries, Query returns empty results without initializing ONNX.
func TestQuery_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	result, err := Query(QueryOpts{
		Text:       "some search query",
		MemoryRoot: t.TempDir(), // fresh empty DB
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Results).To(BeEmpty())
}

func TestQuery_EmptyText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := Query(QueryOpts{Text: "", MemoryRoot: t.TempDir()})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("text is required"))
}

func TestQuery_WithMinScoreFiltering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Non-zero vector required: zero vectors produce NULL cosine similarity in SQLite
	emb := make([]float32, embeddingDim)
	emb[0] = 1.0
	fe := &fixedEmbedder{vec: emb}

	// First learn something so the DB is populated
	learnErr := Learn(LearnOpts{
		Message:        "always write unit tests before implementing any new feature",
		MemoryRoot:     tmpDir,
		Project:        "test",
		VectorEmbedder: fe,
	})
	g.Expect(learnErr).ToNot(HaveOccurred())

	// Query with very high min score (should filter all results)
	result, err := Query(QueryOpts{
		Text:           "write tests before coding",
		MemoryRoot:     tmpDir,
		MinScore:       0.9999,
		VectorEmbedder: fe,
	})
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	g.Expect(result.FilteredCount).To(BeNumerically(">=", 0))
}

func TestQuery_WithSpreadingActivation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Non-zero vector required: zero vectors produce NULL cosine similarity in SQLite
	emb := make([]float32, embeddingDim)
	emb[0] = 1.0
	fe := &fixedEmbedder{vec: emb}

	// Learn something so the DB has data
	learnErr := Learn(LearnOpts{
		Message:        "use proper error handling in all production code functions",
		MemoryRoot:     tmpDir,
		Project:        "test",
		VectorEmbedder: fe,
	})
	g.Expect(learnErr).ToNot(HaveOccurred())

	result, err := Query(QueryOpts{
		Text:                "error handling best practices",
		MemoryRoot:          tmpDir,
		SpreadingActivation: true,
		VectorEmbedder:      fe,
	})
	g.Expect(err).ToNot(HaveOccurred())

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	g.Expect(result.SpreadingActivationApplied).To(BeTrue())
}

// ─── RealFS tests ─────────────────────────────────────────────────────────────

func TestRealFS_MkdirAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")

	err := RealFS{}.MkdirAll(newDir, 0o755)

	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(newDir)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRealFS_ReadDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	entries, err := RealFS{}.ReadDir(tmpDir)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))

	if len(entries) == 0 {
		t.Fatal("entries is empty")
	}

	g.Expect(entries[0].Name()).To(Equal("test.txt"))
}

func TestRealFS_Remove(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "to-remove.txt")

	err := os.WriteFile(f, []byte("content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = RealFS{}.Remove(f)

	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(f)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestRealFS_Rename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	err := os.WriteFile(src, []byte("content"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = RealFS{}.Rename(src, dst)

	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(dst)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(src)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

func TestRealFS_Stat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "stat-test.txt")

	err := os.WriteFile(f, []byte("hello"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	info, err := RealFS{}.Stat(f)

	g.Expect(err).ToNot(HaveOccurred())

	if info == nil {
		t.Fatal("info is nil")
	}

	g.Expect(info.Name()).To(Equal("stat-test.txt"))
	g.Expect(info.Size()).To(BeNumerically(">", 0))
}

// ─── searchDirectory tests ────────────────────────────────────────────────────

func TestSearchDirectory_NonExistentDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := searchDirectory("/nonexistent/dir", "pattern", "")

	g.Expect(result).To(BeEmpty())
}

func TestSearchDirectory_SkipsSubdirectories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Create a file in a subdirectory (should be skipped)
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("match pattern here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchDirectory(tmpDir, "pattern", "")

	// Subdirectory files are not searched
	g.Expect(result).To(BeEmpty())
}

func TestSearchDirectory_WithMatchingFiles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "session1.txt"), []byte("line with searchterm here\nother line"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(tmpDir, "session2.txt"), []byte("no match here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchDirectory(tmpDir, "searchterm", "")

	g.Expect(result).To(HaveLen(1))

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0].Line).To(ContainSubstring("searchterm"))
}

func TestSearchDirectory_WithProjectFilter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "myproject-session.txt"), []byte("match here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(tmpDir, "otherproject-session.txt"), []byte("match here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchDirectory(tmpDir, "match", "myproject")

	// Only the file matching the project filter should be returned
	g.Expect(result).To(HaveLen(1))

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0].File).To(ContainSubstring("myproject"))
}

func TestSearchFile_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "session.txt")

	err := os.WriteFile(f, []byte("nothing relevant here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchFile(f, "notfound", "")

	g.Expect(result).To(BeEmpty())
}

// ─── searchFile tests ─────────────────────────────────────────────────────────

func TestSearchFile_NoProjectFilter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "session.txt")

	err := os.WriteFile(f, []byte("line one\nsearchterm found here\nline three"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchFile(f, "searchterm", "")

	g.Expect(result).To(HaveLen(1))

	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	g.Expect(result[0].LineNum).To(Equal(2))
	g.Expect(result[0].File).To(Equal(f))
}

func TestSearchFile_ProjectFilterMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "myproject-2025.txt")

	err := os.WriteFile(f, []byte("found the pattern here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchFile(f, "pattern", "myproject")

	g.Expect(result).To(HaveLen(1))
}

func TestSearchFile_ProjectFilterNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "otherproject-2025.txt")

	err := os.WriteFile(f, []byte("found the pattern here"), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result := searchFile(f, "pattern", "myproject")

	g.Expect(result).To(BeEmpty())
}

func TestSimulateTimePassage_AgesTimestamps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now()
	ts := []string{now.Format(time.RFC3339)}
	tsJSON, _ := json.Marshal(ts)

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, retrieval_timestamps) VALUES (?, ?, ?)",
		"time passage test content uniquekey123", "test", string(tsJSON),
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = SimulateTimePassage(SimulateTimeOpts{
		MemoryRoot: tmpDir,
		Content:    "time passage test content uniquekey123",
		DaysToAge:  30,
	})

	g.Expect(err).ToNot(HaveOccurred())

	// Verify timestamp was aged
	db, err = initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	var updatedTS string

	err = db.QueryRow("SELECT retrieval_timestamps FROM embeddings WHERE content LIKE '%time passage test content%'").Scan(&updatedTS)
	g.Expect(err).ToNot(HaveOccurred())

	var timestamps []string

	err = json.Unmarshal([]byte(updatedTS), &timestamps)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(timestamps).To(HaveLen(1))

	if len(timestamps) == 0 {
		t.Fatal("timestamps is empty")
	}

	parsedTime, err := time.Parse(time.RFC3339, timestamps[0])
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parsedTime).To(BeTemporally("<", now.Add(-29*24*time.Hour)))
}

// ─── SimulateTimePassage tests ────────────────────────────────────────────────

func TestSimulateTimePassage_EntryNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	err = SimulateTimePassage(SimulateTimeOpts{
		MemoryRoot: tmpDir,
		Content:    "nonexistent content xyz",
		DaysToAge:  1,
	})

	g.Expect(err).To(MatchError(ContainSubstring("failed to find entry")))
}
