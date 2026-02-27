package retro_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/retro"
)

func TestExtractOpenQuestions_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	items, err := retro.ExtractOpenQuestions(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(BeEmpty())
}

func TestExtractOpenQuestions_ReadsRetroMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Retrospective

## Open Questions

### Q1: Should we use option A or B?

**Context:** Background context here.

**Decision needed before:** Next sprint

---

### Q2: How to handle edge cases?

**Context:** Edge case context.

**Decision needed before:** Implementation

`
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	items, err := retro.ExtractOpenQuestions(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(2))
	g.Expect(items).ToNot(BeNil())

	if len(items) >= 2 {
		g.Expect(items[0].ID).To(Equal("Q1"))
		g.Expect(items[0].Title).To(Equal("Should we use option A or B?"))
		g.Expect(items[0].Context).To(ContainSubstring("Background context"))
		g.Expect(items[1].ID).To(Equal("Q2"))
	}
}

func TestExtractOpenQuestions_ReadsRetrospectiveMd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Retrospective

## Open Questions

### Q1: Only question

**Context:** Some context.

`
	g.Expect(os.WriteFile(filepath.Join(dir, "retrospective.md"), []byte(content), 0o644)).To(Succeed())

	items, err := retro.ExtractOpenQuestions(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))

	g.Expect(items).ToNot(BeNil())

	if len(items) > 0 {
		g.Expect(items[0].ID).To(Equal("Q1"))
	}
}

func TestExtractRecommendations_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	items, err := retro.ExtractRecommendations(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(BeEmpty())
}

func TestExtractRecommendations_ParsesAllPriorities(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Retrospective

## Process Improvement Recommendations

### High Priority

#### R1: Critical fix

**Action:** Fix the critical issue.

**Rationale:** Blocks all work.

---

### Medium Priority

#### R2: Improvement

**Action:** Add better logging.

**Rationale:** Helps debugging.

---

### Low Priority

#### R3: Nice to have

**Action:** Add documentation.

**Rationale:** Improves developer experience.

`
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	items, err := retro.ExtractRecommendations(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(3))
	g.Expect(items).ToNot(BeNil())

	if len(items) >= 3 {
		g.Expect(items[0].ID).To(Equal("R1"))
		g.Expect(items[0].Priority).To(Equal("High"))
		g.Expect(items[0].Action).To(ContainSubstring("critical"))
		g.Expect(items[1].ID).To(Equal("R2"))
		g.Expect(items[1].Priority).To(Equal("Medium"))
		g.Expect(items[2].ID).To(Equal("R3"))
		g.Expect(items[2].Priority).To(Equal("Low"))
	}
}

func TestExtractRecommendations_StopsAtNextSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Retrospective

## Process Improvement Recommendations

### High Priority

#### R1: First rec

**Action:** Do something.

## Another Section

Some content that should not be parsed.
`
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	items, err := retro.ExtractRecommendations(dir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(items).To(HaveLen(1))

	g.Expect(items).ToNot(BeNil())

	if len(items) > 0 {
		g.Expect(items[0].ID).To(Equal("R1"))
	}
}

func TestFilterByPriority_EmptyList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := retro.FilterByPriority(nil, "High")
	g.Expect(result).To(BeEmpty())
}

func TestFilterByPriority_FiltersCorrectly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []retro.Recommendation{
		{ID: "R1", Priority: "High"},
		{ID: "R2", Priority: "Medium"},
		{ID: "R3", Priority: "Low"},
	}
	highOnly := retro.FilterByPriority(items, "High")
	g.Expect(highOnly).To(HaveLen(1))

	if len(highOnly) > 0 {
		g.Expect(highOnly[0].ID).To(Equal("R1"))
	}

	mediumUp := retro.FilterByPriority(items, "Medium")
	g.Expect(mediumUp).To(HaveLen(2))

	all := retro.FilterByPriority(items, "Low")
	g.Expect(all).To(HaveLen(3))
}

func TestFilterByPriority_UnknownPriorityIncludesAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	items := []retro.Recommendation{
		{ID: "R1", Priority: "High"},
		{ID: "R2", Priority: "Low"},
	}

	result := retro.FilterByPriority(items, "Unknown")
	g.Expect(result).To(HaveLen(2))
}

func TestRunExtract_DryRun_WithOpenQuestions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	content := `# Retrospective

## Open Questions

### Q1: What direction to take?

**Context:** We need to decide on the approach.

**Decision needed before:** Next sprint

`
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	err := retro.RunExtract(retro.ExtractArgs{
		Dir:     dir,
		RepoDir: dir,
		DryRun:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_DryRun_WithRecommendations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Write a retro.md with a long action body to exercise truncate
	longAction := strings.Repeat("This is a very long action description. ", 5)
	content := "# Retrospective\n\n## Process Improvement Recommendations\n\n### High Priority\n\n#### R1: Fix critical issue\n\n**Action:** " + longAction + "\n\n**Rationale:** Very important.\n\n"
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	err := retro.RunExtract(retro.ExtractArgs{
		Dir:         dir,
		RepoDir:     dir,
		MinPriority: "High",
		DryRun:      true,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	err := retro.RunExtract(retro.ExtractArgs{Dir: dir, RepoDir: dir})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_EmptyDir_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Dir="" → uses os.Getwd(). No retro.md in the package dir.
	err := retro.RunExtract(retro.ExtractArgs{
		Dir:     "",
		RepoDir: t.TempDir(),
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_EmptyRepoDir_UsesGetwd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()

	// Very short action/rationale produces body < 100 chars → truncate returns s unchanged.
	content := "# Retrospective\n\n## Process Improvement Recommendations\n\n### High Priority\n\n#### R1: Fix\n\n**Action:**\n\n**Rationale:**\n\n"
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	// RepoDir="" → uses os.Getwd(). DryRun=true avoids actual issue creation.
	err := retro.RunExtract(retro.ExtractArgs{
		Dir:     dir,
		RepoDir: "",
		DryRun:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_NonDryRun_WithOpenQuestion(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	repoDir := t.TempDir()

	content := `# Retrospective

## Open Questions

### Q1: What direction to take?

**Context:** We need to decide on the approach.

**Decision needed before:** Next sprint

`
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	err := retro.RunExtract(retro.ExtractArgs{
		Dir:     dir,
		RepoDir: repoDir,
		DryRun:  false,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestRunExtract_NonDryRun_WithRecommendation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	repoDir := t.TempDir()

	content := "# Retrospective\n\n## Process Improvement Recommendations\n\n### High Priority\n\n#### R1: Fix something\n\n**Action:** Do the fix.\n\n**Rationale:** Improves quality.\n\n"
	g.Expect(os.WriteFile(filepath.Join(dir, "retro.md"), []byte(content), 0o644)).To(Succeed())

	err := retro.RunExtract(retro.ExtractArgs{
		Dir:         dir,
		RepoDir:     repoDir,
		MinPriority: "High",
		DryRun:      false,
	})
	g.Expect(err).ToNot(HaveOccurred())
}
