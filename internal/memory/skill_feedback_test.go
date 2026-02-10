//go:build sqlite_fts5

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
