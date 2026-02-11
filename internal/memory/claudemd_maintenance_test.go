package memory_test

import (
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
