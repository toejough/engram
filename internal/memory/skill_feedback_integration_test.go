//go:build integration && sqlite_fts5

package memory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Skill Feedback (TASK-5)
// ============================================================================

// TestRecordSkillFeedbackSuccess verifies that positive feedback increments alpha.
func TestRecordSkillFeedbackSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record positive feedback
	err = memory.RecordSkillFeedbackForTest(db, "test-skill", true)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify alpha was incremented
	updated, err := memory.GetSkillBySlugForTest(db, "test-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.Alpha).To(Equal(4.0)) // 3.0 + 1.0
	g.Expect(updated.Beta).To(Equal(1.0))  // Unchanged
}

// TestRecordSkillFeedbackFailure verifies that negative feedback increments beta.
func TestRecordSkillFeedbackFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "fail-skill",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record negative feedback
	err = memory.RecordSkillFeedbackForTest(db, "fail-skill", false)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify beta was incremented
	updated, err := memory.GetSkillBySlugForTest(db, "fail-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.Alpha).To(Equal(3.0)) // Unchanged
	g.Expect(updated.Beta).To(Equal(2.0))  // 1.0 + 1.0
}

// TestRecordSkillFeedbackNotFound verifies that feedback for non-existent skill errors.
func TestRecordSkillFeedbackNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	err = memory.RecordSkillFeedbackForTest(db, "nonexistent", true)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestListSkillsPublic verifies that ListSkillsPublic returns all non-pruned skills.
func TestListSkillsPublic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert active skill
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug: "active", Theme: "Active", Description: "Active skill",
		Content: "C", SourceMemoryIDs: "[1]", Alpha: 5, Beta: 1,
		Utility: 0.8, CreatedAt: now, UpdatedAt: now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Insert pruned skill
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug: "pruned", Theme: "Pruned", Description: "Pruned skill",
		Content: "C", SourceMemoryIDs: "[2]", Alpha: 1, Beta: 1,
		Utility: 0.3, CreatedAt: now, UpdatedAt: now, Pruned: true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := memory.ListSkillsPublic(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))
	g.Expect(skills[0].Slug).To(Equal("active"))
}

// ============================================================================
// Unit tests for RecordSkillUsage (TASK-9)
// ============================================================================

// TestRecordSkillUsageIncrementsRetrievalCount verifies that RecordSkillUsage increments retrieval_count.
func TestRecordSkillUsageIncrementsRetrievalCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "usage-test",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		RetrievalCount:  5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record usage
	err = memory.RecordSkillUsageForTest(db, "usage-test", false)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify retrieval_count was incremented
	updated, err := memory.GetSkillBySlugForTest(db, "usage-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.RetrievalCount).To(Equal(6)) // 5 + 1
}

// TestRecordSkillUsageUpdatesLastRetrieved verifies that RecordSkillUsage updates last_retrieved timestamp.
func TestRecordSkillUsageUpdatesLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "timestamp-test",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		RetrievalCount:  5,
		LastRetrieved:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record usage
	beforeCall := time.Now().UTC().Truncate(time.Second)
	err = memory.RecordSkillUsageForTest(db, "timestamp-test", false)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify last_retrieved was updated
	updated, err := memory.GetSkillBySlugForTest(db, "timestamp-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.LastRetrieved).ToNot(Equal(now))

	// Parse and verify it's recent (within 2 seconds to account for RFC3339 precision)
	lastRetrieved, err := time.Parse(time.RFC3339, updated.LastRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(lastRetrieved).To(BeTemporally(">=", beforeCall.Add(-1*time.Second)))
}

// TestRecordSkillUsageWithSuccessUpdatesAlpha verifies that success=true also updates alpha.
func TestRecordSkillUsageWithSuccessUpdatesAlpha(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "success-test",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		RetrievalCount:  5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record usage with success=true
	err = memory.RecordSkillUsageForTest(db, "success-test", true)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify both retrieval_count and alpha were updated
	updated, err := memory.GetSkillBySlugForTest(db, "success-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.RetrievalCount).To(Equal(6)) // 5 + 1
	g.Expect(updated.Alpha).To(Equal(4.0))        // 3.0 + 1.0
	g.Expect(updated.Beta).To(Equal(1.0))         // Unchanged
}

// TestRecordSkillUsageWithoutSuccessDoesNotUpdateAlpha verifies that success=false does NOT update alpha.
func TestRecordSkillUsageWithoutSuccessDoesNotUpdateAlpha(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &memory.GeneratedSkill{
		Slug:            "no-success-test",
		Theme:           "Test",
		Description:     "Test skill",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.5,
		RetrievalCount:  5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record usage with success=false
	err = memory.RecordSkillUsageForTest(db, "no-success-test", false)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify retrieval_count was updated but alpha/beta were not
	updated, err := memory.GetSkillBySlugForTest(db, "no-success-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.RetrievalCount).To(Equal(6)) // 5 + 1
	g.Expect(updated.Alpha).To(Equal(3.0))        // Unchanged
	g.Expect(updated.Beta).To(Equal(1.0))         // Unchanged
}

// TestRecordSkillUsageNotFound verifies that usage recording for non-existent skill errors.
func TestRecordSkillUsageNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	err = memory.RecordSkillUsageForTest(db, "nonexistent", true)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}
