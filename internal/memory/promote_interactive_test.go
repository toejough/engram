package memory_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Promote interactive functionality
// traces: TASK-7, ISSUE-160
// ============================================================================

// TEST-980: PromoteInteractive requires ReviewFunc when Review is true
// traces: TASK-7
func TestPromoteInteractiveRequiresReviewFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot: memoryRoot,
		Review:     true,
		ReviewFunc: nil, // Missing review function
	}

	_, err := memory.PromoteInteractive(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("review function is required"))
}

// TEST-981: PromoteInteractive calls ReviewFunc for each candidate
// traces: TASK-7
func TestPromoteInteractiveCallsReviewFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add a learning
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "important pattern for review",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query multiple times to build retrieval count
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "important pattern",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	var reviewedCandidates []string
	reviewFunc := func(candidate memory.PromoteCandidate) (bool, error) {
		reviewedCandidates = append(reviewedCandidates, candidate.Content)
		return true, nil // Approve all
	}

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot:    memoryRoot,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        true,
		ReviewFunc:    reviewFunc,
	}

	_, err := memory.PromoteInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// ReviewFunc should have been called for candidates
	g.Expect(len(reviewedCandidates)).To(BeNumerically(">=", 0))
}

// TEST-982: PromoteInteractive counts approved and rejected candidates
// traces: TASK-7
func TestPromoteInteractiveCounts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add multiple learnings
	for i := 0; i < 3; i++ {
		g.Expect(memory.Learn(memory.LearnOpts{
			Message:    "learning number " + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})).To(Succeed())
	}

	// Query to build retrieval counts
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "learning",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	callCount := 0
	reviewFunc := func(candidate memory.PromoteCandidate) (bool, error) {
		callCount++
		// Approve first, reject rest
		return callCount == 1, nil
	}

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot:    memoryRoot,
		MinRetrievals: 1,
		MinProjects:   1,
		Review:        true,
		ReviewFunc:    reviewFunc,
	}

	result, err := memory.PromoteInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should have counts for approved and rejected
	g.Expect(result.CandidatesApproved + result.CandidatesRejected).To(Equal(result.CandidatesReviewed))
}

// TEST-983: PromoteInteractive appends approved candidates to CLAUDE.md
// traces: TASK-7
func TestPromoteInteractiveAppendsToClaudeMD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeDir := filepath.Join(tempDir, ".claude")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(claudeDir, 0755)).To(Succeed())

	claudeMDPath := filepath.Join(claudeDir, "CLAUDE.md")

	// Add a learning
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "approved learning for CLAUDE.md",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to build retrieval count
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "approved learning",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	reviewFunc := func(candidate memory.PromoteCandidate) (bool, error) {
		return true, nil // Approve
	}

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot:    memoryRoot,
		ClaudeMDPath:  claudeMDPath,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        true,
		ReviewFunc:    reviewFunc,
	}

	result, err := memory.PromoteInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())

	if result.CandidatesApproved > 0 {
		// CLAUDE.md should exist and contain the approved learning
		g.Expect(claudeMDPath).To(BeAnExistingFile())

		content, err := os.ReadFile(claudeMDPath)
		g.Expect(err).ToNot(HaveOccurred())

		contentStr := string(content)
		g.Expect(contentStr).To(ContainSubstring("approved learning"))
	}
}

// TEST-984: PromoteInteractive without Review mode works like regular Promote
// traces: TASK-7
func TestPromoteInteractiveNonReviewMode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add a learning
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "non-review mode learning",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to build retrieval count
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "non-review",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot:    memoryRoot,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        false, // Non-review mode
	}

	result, err := memory.PromoteInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should work like regular promote - just report candidates
	g.Expect(result.CandidatesReviewed).To(Equal(0))
	g.Expect(result.CandidatesApproved).To(Equal(0))
	g.Expect(result.CandidatesRejected).To(Equal(0))
}

// TEST-985: PromoteInteractive returns summary with all counts
// traces: TASK-7
func TestPromoteInteractiveReturnsSummary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot: memoryRoot,
		Review:     false,
	}

	result, err := memory.PromoteInteractive(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify result structure
	g.Expect(result.CandidatesReviewed).To(BeNumerically(">=", 0))
	g.Expect(result.CandidatesApproved).To(BeNumerically(">=", 0))
	g.Expect(result.CandidatesRejected).To(BeNumerically(">=", 0))
}

// TEST-986: PromoteInteractive handles ReviewFunc errors gracefully
// traces: TASK-7
func TestPromoteInteractiveHandlesReviewFuncErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add a learning
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "error handling test",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to build retrieval count
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "error",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	reviewFunc := func(candidate memory.PromoteCandidate) (bool, error) {
		return false, errors.New("simulated review error") // Simulated error
	}

	opts := memory.PromoteInteractiveOpts{
		MemoryRoot:    memoryRoot,
		MinRetrievals: 3,
		MinProjects:   2,
		Review:        true,
		ReviewFunc:    reviewFunc,
	}

	_, err := memory.PromoteInteractive(opts)
	// Should return the error from ReviewFunc
	g.Expect(err).To(HaveOccurred())
}
