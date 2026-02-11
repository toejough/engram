//go:build integration

package retro_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/retro"
)

func TestExtractRecommendations(t *testing.T) {
	t.Parallel()
	t.Run("extracts high priority recommendations", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		content := `# Retrospective

## Process Improvement Recommendations

### High Priority

#### R1: Create projctl validate-spec command

**Action:** Implement validation.

**Rationale:** Prevents aspirational specs from blocking work.

---

#### R2: Add integration test AC

**Action:** Update project skill to require integration test.

**Rationale:** Unit tests aren't sufficient.

---

### Medium Priority

#### R3: Document parallel execution

**Action:** Add documentation.

---

### Low Priority

#### R4: Create SKILL-full.md generator

**Action:** Tool that generates comprehensive documentation.
`
		g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

		items, err := retro.ExtractRecommendations(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(items).To(HaveLen(4))

		// Check high priority items
		g.Expect(items[0].ID).To(Equal("R1"))
		g.Expect(items[0].Title).To(Equal("Create projctl validate-spec command"))
		g.Expect(items[0].Priority).To(Equal("High"))
		g.Expect(items[0].Action).To(ContainSubstring("validation"))

		g.Expect(items[1].ID).To(Equal("R2"))
		g.Expect(items[1].Priority).To(Equal("High"))

		// Check medium priority
		g.Expect(items[2].ID).To(Equal("R3"))
		g.Expect(items[2].Priority).To(Equal("Medium"))

		// Check low priority
		g.Expect(items[3].ID).To(Equal("R4"))
		g.Expect(items[3].Priority).To(Equal("Low"))
	})

	t.Run("extracts open questions", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		content := `# Retrospective

## Open Questions

### Q1: Should Layer 0 use /project skill or start fresh?

**Context:** TASK-29 updated skill but deferred work.

**Options:**
- **A:** Continue using /project
- **B:** Implement projctl project

**Decision needed before:** Layer 0 intake

---

### Q2: Should context-explorer use Claude Tasks?

**Context:** context-explorer needs parallel queries.

**Options:**
- **A:** Use Task tool
- **B:** Use projctl command

**Decision needed before:** context-explorer test
`
		g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

		items, err := retro.ExtractOpenQuestions(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(items).To(HaveLen(2))

		g.Expect(items[0].ID).To(Equal("Q1"))
		g.Expect(items[0].Title).To(Equal("Should Layer 0 use /project skill or start fresh?"))
		g.Expect(items[0].Context).To(ContainSubstring("TASK-29"))

		g.Expect(items[1].ID).To(Equal("Q2"))
	})

	t.Run("returns empty when no retro.md", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		items, err := retro.ExtractRecommendations(dir)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(items).To(BeEmpty())
	})
}

func TestFilterByPriority(t *testing.T) {
	t.Parallel()
	t.Run("filters recommendations by minimum priority", func(t *testing.T) {
		g := NewWithT(t)

		items := []retro.Recommendation{
			{ID: "R1", Title: "High", Priority: "High"},
			{ID: "R2", Title: "Medium", Priority: "Medium"},
			{ID: "R3", Title: "Low", Priority: "Low"},
		}

		highOnly := retro.FilterByPriority(items, "High")
		g.Expect(highOnly).To(HaveLen(1))
		g.Expect(highOnly[0].ID).To(Equal("R1"))

		mediumUp := retro.FilterByPriority(items, "Medium")
		g.Expect(mediumUp).To(HaveLen(2))

		all := retro.FilterByPriority(items, "Low")
		g.Expect(all).To(HaveLen(3))
	})
}
