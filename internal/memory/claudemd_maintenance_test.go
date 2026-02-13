package memory_test

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for CLAUDE.md maintenance (ISSUE-212, Task #4)
// ============================================================================

// TEST-1200: scanClaudeMD detects redundant entries
func TestScanClaudeMDDetectsRedundantEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `# Working With Joe

## Promoted Learnings

- Always use TDD approach for all code changes
- Use TDD methodology for all code changes
- Use property-based testing for edge cases

## Other Section

Some content.
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.8)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect redundancy between first two entries
	var consolidateProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "consolidate" {
			consolidateProposals = append(consolidateProposals, p)
		}
	}

	g.Expect(len(consolidateProposals)).To(BeNumerically(">", 0))
	g.Expect(consolidateProposals[0].Tier).To(Equal("claude-md"))
	g.Expect(consolidateProposals[0].Reason).To(ContainSubstring("similar"))
}

// TEST-1201: scanClaudeMD detects overly broad entries
func TestScanClaudeMDDetectsBroadEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Generate entry with >100 tokens to trigger split detection
	words := make([]string, 110)
	for i := range words {
		words[i] = "word"
	}
	longEntry := "Success ISSUE-152: Foundation task executed cleanly with zero rework and " + strings.Join(words, " ")

	content := `## Promoted Learnings

- ` + longEntry + `
- Short entry here
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.8)
	g.Expect(err).ToNot(HaveOccurred())

	// Should propose splitting the long entry
	var splitProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "split" {
			splitProposals = append(splitProposals, p)
		}
	}

	g.Expect(len(splitProposals)).To(BeNumerically(">", 0))
	g.Expect(splitProposals[0].Tier).To(Equal("claude-md"))
	g.Expect(splitProposals[0].Target).To(ContainSubstring("ISSUE-152"))
}

// TEST-1202: scanClaudeMD detects too-specific entries
func TestScanClaudeMDDetectsTooSpecificEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- projctl uses targ build system for tests
- Always validate inputs at system boundaries
- Use internal/ for non-public implementation code
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.8)
	g.Expect(err).ToNot(HaveOccurred())

	// Should propose demoting the first entry (too specific to projctl)
	var demoteProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "demote" {
			demoteProposals = append(demoteProposals, p)
		}
	}

	g.Expect(len(demoteProposals)).To(BeNumerically(">", 0))
	var foundProjectSpecific bool
	for _, p := range demoteProposals {
		if p.Target == "projctl uses targ build system for tests" {
			foundProjectSpecific = true
			g.Expect(p.Reason).To(ContainSubstring("project"))
		}
	}
	g.Expect(foundProjectSpecific).To(BeTrue())
}

// TEST-1203: scanClaudeMD on empty file returns no proposals
func TestScanClaudeMDEmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(""),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.8)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(proposals)).To(Equal(0))
}

// TEST-1204: scanClaudeMD on missing file returns no error
func TestScanClaudeMDMissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/nonexistent/CLAUDE.md", 0.8)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(len(proposals)).To(Equal(0))
}

// TEST-1205: applyClaudeMDProposal prunes entry
func TestApplyClaudeMDProposalPrune(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- 2026-02-08 21:40: entry to remove
- 2026-02-08 21:41: entry to keep
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:   "claude-md",
		Action: "prune",
		Target: "entry to remove",
		Reason: "stale",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring("entry to remove"))
	g.Expect(result).To(ContainSubstring("entry to keep"))
}

// TEST-1206: applyClaudeMDProposal consolidates entries
func TestApplyClaudeMDProposalConsolidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Always use TDD: write tests first
- Write failing tests before implementation
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "consolidate",
		Target:  "Always use TDD: write tests first|Write failing tests before implementation",
		Preview: "Always use TDD: write failing tests before implementation",
		Reason:  "redundant entries with similarity > 0.8",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("Always use TDD: write failing tests before implementation"))
	g.Expect(result).ToNot(ContainSubstring("write tests first"))
}

// TEST-1207: applyClaudeMDProposal splits entry
func TestApplyClaudeMDProposalSplit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	longEntry := "Success ISSUE-152: Foundation task executed cleanly. Batch parallelization effective. TDD red phase required rework."

	content := `## Promoted Learnings

- ` + longEntry + `
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "split",
		Target:  longEntry,
		Preview: "Foundation task ISSUE-152 executed cleanly|Batch parallelization is effective|TDD red phase may require rework",
		Reason:  "entry covers multiple topics (850 tokens)",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring(longEntry))
	g.Expect(result).To(ContainSubstring("Foundation task"))
	g.Expect(result).To(ContainSubstring("Batch parallelization"))
	g.Expect(result).To(ContainSubstring("TDD red phase"))
}

// TEST-1208: applyClaudeMDProposal demotes to skill
func TestApplyClaudeMDProposalDemote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- projctl uses targ build system
- Always validate inputs
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:   "claude-md",
		Action: "demote",
		Target: "projctl uses targ build system",
		Reason: "too specific to single project",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring("projctl uses targ"))
	g.Expect(result).To(ContainSubstring("Always validate"))
}

// TEST-1209: applyClaudeMDProposal on unknown action returns error
func TestApplyClaudeMDProposalUnknownAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte("## Promoted Learnings\n\n- entry\n"),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:   "claude-md",
		Action: "unknown",
		Target: "entry",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown action"))
}

// TEST-1210: scanClaudeMD respects similarity threshold
func TestScanClaudeMDSimilarityThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Use TDD for all code
- TDD is required for all implementations
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	// High threshold - should find no redundancy
	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.99)
	g.Expect(err).ToNot(HaveOccurred())

	var consolidateProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "consolidate" {
			consolidateProposals = append(consolidateProposals, p)
		}
	}

	g.Expect(len(consolidateProposals)).To(Equal(0))
}

// ============================================================================
// Additional tests for ISSUE-184: CLAUDE.md Maintenance Gaps
// ============================================================================

// TEST-1211: scanClaudeMD detects stale entries (ISSUE-184)
func TestScanClaudeMD_DetectsStaleness(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- 2020-01-01 10:00: Old learning that should be pruned
- 2026-02-08 21:40: Recent learning to keep
- Learning without timestamp
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.9)
	g.Expect(err).ToNot(HaveOccurred())

	// Should detect stale entry (>90 days old)
	var pruneProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "prune" {
			pruneProposals = append(pruneProposals, p)
		}
	}

	g.Expect(len(pruneProposals)).To(BeNumerically(">", 0))

	// Find the stale entry proposal
	var foundStale bool
	for _, p := range pruneProposals {
		if strings.Contains(p.Target, "Old learning") {
			foundStale = true
			g.Expect(p.Tier).To(Equal("claude-md"))
			g.Expect(p.Reason).To(ContainSubstring("stale"))
			g.Expect(p.Reason).To(ContainSubstring(">90 days"))
		}
	}
	g.Expect(foundStale).To(BeTrue())
}

// TEST-1212: scanClaudeMD does not flag entries without timestamps (ISSUE-184)
func TestScanClaudeMD_NoTimestampNotFlagged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Learning without timestamp should not be flagged
- Another learning without timestamp
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposals, err := memory.ScanClaudeMD(fs, "/test/CLAUDE.md", 0.9)
	g.Expect(err).ToNot(HaveOccurred())

	// Should not generate prune proposals for entries without timestamps
	var pruneProposals []memory.MaintenanceProposal
	for _, p := range proposals {
		if p.Action == "prune" {
			pruneProposals = append(pruneProposals, p)
		}
	}

	// No stale pruning should occur since there are no timestamps
	g.Expect(len(pruneProposals)).To(Equal(0))
}

// ============================================================================
// Additional tests for ISSUE-218: Content Refinement Operations
// ============================================================================

// TEST-1213: applyClaudeMDProposal handles rewrite action (ISSUE-218)
func TestApplyClaudeMDProposal_Rewrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Use TDD
- Always validate inputs
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "rewrite",
		Target:  "Use TDD",
		Preview: "Always use Test-Driven Development for all code changes",
		Reason:  "improve clarity and specificity",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring("Use TDD"))
	g.Expect(result).To(ContainSubstring("Always use Test-Driven Development"))
	g.Expect(result).To(ContainSubstring("Always validate inputs"))
}

// TEST-1214: applyClaudeMDProposal handles add-rationale action (ISSUE-218)
func TestApplyClaudeMDProposal_AddRationale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Never use git amend on pushed commits
- Always validate inputs
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "add-rationale",
		Target:  "Never use git amend on pushed commits",
		Preview: "Never use git amend on pushed commits - this rewrites history and breaks collaboration",
		Reason:  "add explanation of why this matters",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).To(ContainSubstring("rewrites history"))
	g.Expect(result).To(ContainSubstring("breaks collaboration"))
	g.Expect(result).To(ContainSubstring("Always validate inputs"))
}

// TEST-1215: applyClaudeMDProposal handles extract-examples action (ISSUE-218)
func TestApplyClaudeMDProposal_ExtractExamples(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := `## Promoted Learnings

- Use go test -tags sqlite_fts5 for all tests. Example: ` + "`go test -tags sqlite_fts5 -count=1`" + `
- Always validate inputs
`
	fs := &MockFS{
		Files: map[string][]byte{
			"/test/CLAUDE.md": []byte(content),
		},
	}

	proposal := memory.MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "extract-examples",
		Target:  "Use go test -tags sqlite_fts5 for all tests. Example: `go test -tags sqlite_fts5 -count=1`",
		Preview: "Use go test -tags sqlite_fts5 for all tests",
		Reason:  "extract code examples to keep principle clean",
	}

	err := memory.ApplyClaudeMDProposal(fs, "/test/CLAUDE.md", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	result := string(fs.Files["/test/CLAUDE.md"])
	g.Expect(result).ToNot(ContainSubstring("Example:"))
	g.Expect(result).ToNot(ContainSubstring("`go test -tags sqlite_fts5 -count=1`"))
	g.Expect(result).To(ContainSubstring("Use go test -tags sqlite_fts5 for all tests"))
	g.Expect(result).To(ContainSubstring("Always validate inputs"))
}

// ============================================================================
// Tests for ISSUE-224: Feedback-aware CLAUDE.md staleness
// ============================================================================

// TestScanClaudeMDFeedback_NoFlaggedEmbeddings verifies no proposals when no embeddings are flagged
func TestScanClaudeMDFeedback_NoFlaggedEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn some memories and promote them
	err := memory.Learn(memory.LearnOpts{
		Message:    "Always use TDD approach",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB and promote the memory
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	_, err = db.Exec("UPDATE embeddings SET promoted = 1 WHERE content LIKE '%Always use TDD%'")
	g.Expect(err).To(BeNil())

	// Create CLAUDE.md with promoted learning
	claudeMDPath := memoryRoot + "/CLAUDE.md"
	err = os.WriteFile(claudeMDPath, []byte(`## Promoted Learnings

- Always use TDD approach
`), 0644)
	g.Expect(err).To(BeNil())

	// Call ScanClaudeMDFeedback (no flagged embeddings)
	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).To(BeNil())
	g.Expect(len(proposals)).To(Equal(0), "should return no proposals when no embeddings are flagged")
}

// TestScanClaudeMDFeedback_FlaggedMatchesPromoted verifies proposals when flagged embedding matches promoted learning
func TestScanClaudeMDFeedback_FlaggedMatchesPromoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Always use TDD approach",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// Query to get embedding ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "TDD",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embID := result.Results[0].ID

	// Flag the embedding and mark as promoted
	err = memory.RecordFeedback(db, embID, memory.FeedbackWrong)
	g.Expect(err).To(BeNil())

	_, err = db.Exec("UPDATE embeddings SET promoted = 1 WHERE id = ?", embID)
	g.Expect(err).To(BeNil())

	// Create CLAUDE.md with the promoted learning
	claudeMDPath := memoryRoot + "/CLAUDE.md"
	err = os.WriteFile(claudeMDPath, []byte(`## Promoted Learnings

- Always use TDD approach
`), 0644)
	g.Expect(err).To(BeNil())

	// Call ScanClaudeMDFeedback
	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).To(BeNil())
	g.Expect(len(proposals)).To(BeNumerically(">", 0), "should return at least one proposal")

	// Verify proposal details
	found := false
	for _, p := range proposals {
		if strings.Contains(p.Target, "Always use TDD") {
			found = true
			g.Expect(p.Tier).To(Equal("claude-md"))
			g.Expect(p.Action).To(Equal("review"))
			g.Expect(p.Reason).To(ContainSubstring("source embedding flagged"))
			g.Expect(p.Reason).To(ContainSubstring("wrong"))
		}
	}
	g.Expect(found).To(BeTrue(), "should find proposal for flagged TDD learning")
}

// TestScanClaudeMDFeedback_FlaggedNoMatch verifies no proposals when flagged embedding doesn't match promoted learnings
func TestScanClaudeMDFeedback_FlaggedNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn two different memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "Always validate inputs",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Use property-based testing",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// Query to get first embedding ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "validate",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embID := result.Results[0].ID

	// Flag the first embedding and mark as promoted
	err = memory.RecordFeedback(db, embID, memory.FeedbackUnclear)
	g.Expect(err).To(BeNil())

	_, err = db.Exec("UPDATE embeddings SET promoted = 1 WHERE id = ?", embID)
	g.Expect(err).To(BeNil())

	// Create CLAUDE.md with DIFFERENT learning (not matching flagged one)
	claudeMDPath := memoryRoot + "/CLAUDE.md"
	err = os.WriteFile(claudeMDPath, []byte(`## Promoted Learnings

- Use property-based testing for edge cases
`), 0644)
	g.Expect(err).To(BeNil())

	// Call ScanClaudeMDFeedback
	proposals, err := memory.ScanClaudeMDFeedback(db, claudeMDPath)
	g.Expect(err).To(BeNil())
	g.Expect(len(proposals)).To(Equal(0), "should return no proposals when flagged embedding doesn't match CLAUDE.md entries")
}
