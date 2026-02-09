package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for ParseCLAUDEMD and ConsolidateClaudeMD
// traces: ISSUE-177
// ============================================================================

// TEST-4030: ParseCLAUDEMD parses sections correctly
// traces: ISSUE-177
func TestParseCLAUDEMD(t *testing.T) {
	g := NewWithT(t)

	content := `# Top Level

Some preamble text.

## Section One

Line A
Line B

## Section Two

Line C

## Promoted Learnings

- learning one
- learning two
`

	sections := memory.ParseCLAUDEMD(content)
	g.Expect(sections).To(HaveKey("Section One"))
	g.Expect(sections).To(HaveKey("Section Two"))
	g.Expect(sections).To(HaveKey("Promoted Learnings"))

	g.Expect(sections["Section One"]).To(ContainElement(ContainSubstring("Line A")))
	g.Expect(sections["Section One"]).To(ContainElement(ContainSubstring("Line B")))
	g.Expect(sections["Promoted Learnings"]).To(ContainElement(ContainSubstring("learning one")))
	g.Expect(sections["Promoted Learnings"]).To(ContainElement(ContainSubstring("learning two")))
}

// TEST-4031: ParseCLAUDEMD returns empty map for empty content
// traces: ISSUE-177
func TestParseCLAUDEMDEmpty(t *testing.T) {
	g := NewWithT(t)

	sections := memory.ParseCLAUDEMD("")
	g.Expect(sections).To(BeEmpty())
}

// TEST-4032: ConsolidateClaudeMD finds redundancy between CLAUDE.md and memory DB
// traces: ISSUE-177
func TestConsolidateClaudeMDFindsRedundancy(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Store a learning in the memory DB
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "important pattern for review",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "important pattern",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Create CLAUDE.md with the same learning in Promoted Learnings
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	claudeContent := `# Config

## Promoted Learnings

- important pattern for review
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(claudeContent), 0644)).To(Succeed())

	result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.RedundantCount).To(BeNumerically(">", 0),
		"should detect redundancy when CLAUDE.md learning is also in memory DB")

	// Check that at least one proposal is of type "redundant"
	hasRedundant := false
	for _, p := range result.Proposals {
		if p.Type == "redundant" {
			hasRedundant = true
			g.Expect(p.Similarity).To(BeNumerically(">", 0.5))
			g.Expect(p.Action).To(Equal("remove"))
			break
		}
	}
	g.Expect(hasRedundant).To(BeTrue(), "should have at least one redundant proposal")
}

// TEST-4033: ConsolidateClaudeMD no redundancy when content differs
// traces: ISSUE-177
func TestConsolidateClaudeMDNoRedundancy(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Store a learning about a completely different topic
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "use dependency injection for testability",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "dependency injection",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Create CLAUDE.md with completely unrelated content
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	claudeContent := `# Config

## Promoted Learnings

- always eat breakfast before noon
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(claudeContent), 0644)).To(Succeed())

	result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.RedundantCount).To(Equal(0),
		"should not detect redundancy when content differs")
}

// TEST-4034: ConsolidateClaudeMD handles empty CLAUDE.md
// traces: ISSUE-177
func TestConsolidateClaudeMDEmptyFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Create empty CLAUDE.md
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.WriteFile(claudeMDPath, []byte(""), 0644)).To(Succeed())

	result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Proposals).To(BeEmpty())
	g.Expect(result.RedundantCount).To(Equal(0))
	g.Expect(result.PromoteCount).To(Equal(0))
}

// TEST-4035: Property: proposal counts match actual items found
// traces: ISSUE-177
func TestPropertyConsolidateClaudeMDProposalCounts(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		baseDir := filepath.Join(tempDir, "claudemd-count-"+suffix)
		defer func() { _ = os.RemoveAll(baseDir) }()

		memoryRoot := filepath.Join(baseDir, "memory")
		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

		// Create a CLAUDE.md with some learnings
		numLearnings := rapid.IntRange(0, 3).Draw(t, "numLearnings")
		var learningLines []string
		for i := 0; i < numLearnings; i++ {
			learningLines = append(learningLines, "- learning number "+string(rune('A'+i)))
		}

		var claudeContent string
		if numLearnings > 0 {
			claudeContent = "# Config\n\n## Promoted Learnings\n\n" + strings.Join(learningLines, "\n") + "\n"
		} else {
			claudeContent = "# Config\n"
		}

		claudeMDPath := filepath.Join(baseDir, "CLAUDE.md")
		g.Expect(os.WriteFile(claudeMDPath, []byte(claudeContent), 0644)).To(Succeed())

		result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
			MemoryRoot:   memoryRoot,
			ClaudeMDPath: claudeMDPath,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Count actual redundant and promote proposals
		redundantCount := 0
		promoteCount := 0
		for _, p := range result.Proposals {
			switch p.Type {
			case "redundant":
				redundantCount++
			case "promote":
				promoteCount++
			}
		}
		g.Expect(result.RedundantCount).To(Equal(redundantCount),
			"RedundantCount should match actual redundant proposals")
		g.Expect(result.PromoteCount).To(Equal(promoteCount),
			"PromoteCount should match actual promote proposals")
	})
}

// TEST-4036: ConsolidateClaudeMD interactive review applies approved changes
// traces: ISSUE-177
func TestConsolidateClaudeMDInteractiveApplies(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Store a learning in the memory DB
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "important pattern for review",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "important pattern",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Create CLAUDE.md with the same learning
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	claudeContent := `# Config

## Promoted Learnings

- important pattern for review
`
	g.Expect(os.WriteFile(claudeMDPath, []byte(claudeContent), 0644)).To(Succeed())

	approvedCount := 0
	result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		ReviewFunc: func(p memory.ConsolidateProposal) (bool, error) {
			approvedCount++
			return true, nil // Approve all proposals
		},
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	// If there were proposals, they should have been reviewed
	if len(result.Proposals) > 0 {
		g.Expect(approvedCount).To(BeNumerically(">", 0))
		g.Expect(result.Applied).To(Equal(approvedCount))
	}
}

// TEST-4037: ConsolidateClaudeMD handles missing CLAUDE.md file
// traces: ISSUE-177
func TestConsolidateClaudeMDMissingFile(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Point to non-existent CLAUDE.md
	claudeMDPath := filepath.Join(tempDir, "nonexistent", "CLAUDE.md")

	result, err := memory.ConsolidateClaudeMD(memory.ConsolidateClaudeMDOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Proposals).To(BeEmpty())
}
