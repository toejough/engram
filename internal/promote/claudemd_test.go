package promote_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/promote"
	"engram/internal/registry"
)

// TestAddEntry_EmptyContent verifies AddEntry with empty existing content.
func TestAddEntry_EmptyContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	editor := &promote.SectionEditor{}

	result, err := editor.AddEntry("", "## New\n\nContent.")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("## New\n\nContent."))
}

// TestClaudeMDDemote_NotFound verifies Demote returns error for missing entry.
func TestClaudeMDDemote_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{entries: []registry.InstructionEntry{}}

	promoter := &promote.ClaudeMDPromoter{Registry: reg}

	err := promoter.Demote(context.Background(), "nonexistent")
	g.Expect(err).To(HaveOccurred())
}

// TestClaudeMDDemote_StoreReadError verifies Demote returns error on store read failure.
func TestClaudeMDDemote_StoreReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "claude-md:test",
				SourceType: "claude-md",
				SourcePath: "CLAUDE.md",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry: reg,
		Store:    &fakeStore{readErr: errors.New("disk error")},
	}

	err := promoter.Demote(context.Background(), "claude-md:test")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("reading CLAUDE.md")))
}

// TestClaudeMDDemote_UserDeclines verifies Demote returns nil on user decline.
func TestClaudeMDDemote_UserDeclines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	claudeMDContent := "## Test Rule\n\nContent.\n\n" +
		"<!-- promoted from claude-md:test-rule -->"

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "claude-md:test-rule",
				SourceType: "claude-md",
				SourcePath: "CLAUDE.md",
				Title:      "Test Rule",
			},
		},
	}

	store := &fakeStore{content: claudeMDContent}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		SkillGenerator: &fakeGenerator{content: "# demoted"},
		Editor:         &promote.SectionEditor{},
		Store:          store,
		Confirmer:      &fakeConfirmer{response: false},
	}

	err := promoter.Demote(context.Background(), "claude-md:test-rule")
	g.Expect(err).NotTo(HaveOccurred())

	// Store should NOT have been written.
	g.Expect(store.written).To(BeEmpty())
}

// TestClaudeMDPromote_ConfirmError verifies Promote returns error on confirm failure.
func TestClaudeMDPromote_ConfirmError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:test",
				SourceType: "skill",
				SourcePath: "skills/test.md",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		Store:          &fakeStore{content: "# Project"},
		EntryGenerator: &fakeEntryGenerator{entry: "## Test\n\ncontent"},
		Confirmer:      &fakeConfirmerErr{err: errors.New("tty error")},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{Title: "Test", Content: "c"}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:test")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("confirming promotion")))
}

// TestClaudeMDPromote_GenerateError verifies Promote returns error on generate failure.
func TestClaudeMDPromote_GenerateError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:test",
				SourceType: "skill",
				SourcePath: "skills/test.md",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		Store:          &fakeStore{content: "# Project"},
		EntryGenerator: &fakeEntryGenerator{err: errors.New("llm failed")},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{Title: "Test", Content: "c"}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:test")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("generating entry")))
}

// TestClaudeMDPromote_NotFound verifies Promote returns error for missing entry.
func TestClaudeMDPromote_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{entries: []registry.InstructionEntry{}}

	promoter := &promote.ClaudeMDPromoter{Registry: reg}

	err := promoter.Promote(context.Background(), "nonexistent")
	g.Expect(err).To(HaveOccurred())
}

// TestClaudeMDPromote_SkillLoadError verifies Promote returns error on skill load failure.
func TestClaudeMDPromote_SkillLoadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:broken",
				SourceType: "skill",
				SourcePath: "skills/broken.md",
				Title:      "Broken",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry: reg,
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{}, errors.New("file corrupt")
		},
	}

	err := promoter.Promote(context.Background(), "skill:broken")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("loading skill")))
}

// TestClaudeMDPromote_StoreReadError verifies Promote returns error on store read failure.
func TestClaudeMDPromote_StoreReadError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:test",
				SourceType: "skill",
				SourcePath: "skills/test.md",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry: reg,
		Store:    &fakeStore{readErr: errors.New("disk error")},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{Title: "Test", Content: "c"}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:test")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("reading CLAUDE.md")))
}

// TestClaudeMDPromote_UserDeclines verifies Promote returns nil on user decline.
func TestClaudeMDPromote_UserDeclines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:test",
				SourceType: "skill",
				SourcePath: "skills/test.md",
				Title:      "Test",
			},
		},
	}

	store := &fakeStore{content: "# Project"}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		Store:          store,
		EntryGenerator: &fakeEntryGenerator{entry: "## Test\n\ncontent"},
		Confirmer:      &fakeConfirmer{response: false},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{Title: "Test", Content: "c"}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:test")
	g.Expect(err).NotTo(HaveOccurred())

	// Store should NOT have been written.
	g.Expect(store.written).To(BeEmpty())
}

// TestClaudeMDPromote_WriteError verifies Promote returns error on store write failure.
func TestClaudeMDPromote_WriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:         "skill:test",
				SourceType: "skill",
				SourcePath: "skills/test.md",
				Title:      "Test",
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		Store:          &fakeStore{content: "# Project", writeErr: errors.New("read only")},
		EntryGenerator: &fakeEntryGenerator{entry: "## Test\n\ncontent"},
		Editor:         &promote.SectionEditor{},
		Confirmer:      &fakeConfirmer{response: true},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{Title: "Test", Content: "c"}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:test")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("writing CLAUDE.md")))
}

// T-248: Promotion candidate detection — Working skills.
func TestT248_PromotionCandidateDetectionWorkingSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			// Working: surfaced=150, eff=80% (high). Passes threshold=100.
			makeSkillEntry("working-skill", 150, 80, 20),
			// Leech: surfaced=100, eff=20% (low). Passes threshold but not Working.
			makeSkillEntry("leech-skill", 100, 20, 80),
			// HiddenGem: surfaced=30. Below threshold=100.
			makeSkillEntry("hidden-skill", 30, 80, 20),
		},
	}

	promoter := &promote.ClaudeMDPromoter{Registry: reg}

	candidates, err := promoter.PromotionCandidates(100)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(HaveLen(1))
	g.Expect(candidates[0].Entry.ID).To(Equal("working-skill"))
}

// T-249: Demotion candidate detection — Leech claude-md entries.
func TestT249_DemotionCandidateDetectionLeechClaudeMD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			// Working claude-md: eff=85% (high). Always-loaded → binary Working.
			makeClaudeMDEntry("working-entry", 200, 85, 15),
			// Leech claude-md: eff=20% (low). Always-loaded → binary Leech.
			makeClaudeMDEntry("leech-entry", 100, 20, 80),
		},
	}

	promoter := &promote.ClaudeMDPromoter{Registry: reg}

	candidates, err := promoter.DemotionCandidates()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(HaveLen(1))
	g.Expect(candidates[0].Entry.ID).To(Equal("leech-entry"))
}

// T-250: CLAUDE.md entry generation — matches style with traceability.
func TestT250_ClaudeMDEntryGenerationMatchesStyle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := promote.SkillContent{
		Title:   "Use Targ Build System",
		Content: "Always use targ for builds, not raw go commands.",
	}

	result := promote.FormatClaudeMDEntry(skill, "skill:use-targ-build")

	g.Expect(result).To(ContainSubstring("## Use Targ Build System"))
	g.Expect(result).To(ContainSubstring(
		"Always use targ for builds, not raw go commands.",
	))
	g.Expect(result).To(ContainSubstring(
		"<!-- promoted from skill:use-targ-build -->",
	))
}

// T-251: CLAUDE.md add entry — section appended.
func TestT251_ClaudeMDAddEntrySectionAppended(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	existing := "# Project\n\n## Section 1\n\nContent 1.\n\n" +
		"## Section 2\n\nContent 2.\n\n" +
		"## Section 3\n\nContent 3."

	newEntry := "## New Section\n\nNew content.\n\n" +
		"<!-- promoted from skill:new -->"

	editor := &promote.SectionEditor{}

	result, err := editor.AddEntry(existing, newEntry)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Original sections unchanged.
	g.Expect(result).To(ContainSubstring("## Section 1"))
	g.Expect(result).To(ContainSubstring("## Section 2"))
	g.Expect(result).To(ContainSubstring("## Section 3"))
	// New section appended.
	g.Expect(result).To(ContainSubstring("## New Section"))
	g.Expect(result).To(ContainSubstring("<!-- promoted from skill:new -->"))
}

// T-252: CLAUDE.md remove entry — section removed.
func TestT252_ClaudeMDRemoveEntrySectionRemoved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	content := "# Project\n\n## Section 1\n\nContent 1.\n\n" +
		"## Promoted Section\n\nPromoted content.\n\n" +
		"<!-- promoted from skill:X -->\n\n" +
		"## Section 3\n\nContent 3."

	editor := &promote.SectionEditor{}

	result, err := editor.RemoveEntry(content, "skill:X")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Promoted section removed.
	g.Expect(result).NotTo(ContainSubstring("Promoted Section"))
	g.Expect(result).NotTo(ContainSubstring("skill:X"))
	// Other sections unchanged.
	g.Expect(result).To(ContainSubstring("## Section 1"))
	g.Expect(result).To(ContainSubstring("## Section 3"))
}

// T-253: Demotion execution — CLAUDE.md entry to skill.
func TestT253_DemotionExecutionClaudeMDToSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	claudeMDContent := "# Project\n\n## Leech Rule\n\nLeech content.\n\n" +
		"<!-- promoted from claude-md:leech-rule -->\n\n" +
		"## Good Rule\n\nGood content."

	store := &fakeStore{content: claudeMDContent}
	merger := &fakeMerger{}
	registerer := &fakeRegisterer{}
	writer := newFakeWriter()

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:            "claude-md:leech-rule",
				SourceType:    "claude-md",
				SourcePath:    "CLAUDE.md",
				Title:         "Leech Rule",
				SurfacedCount: 100,
				Evaluations: registry.EvaluationCounters{
					Followed: 10, Ignored: 90,
				},
			},
		},
	}

	promoter := &promote.ClaudeMDPromoter{
		Registry:       reg,
		SkillGenerator: &fakeGenerator{content: "# demoted skill"},
		Editor:         &promote.SectionEditor{},
		Store:          store,
		SkillWriter:    writer,
		Merger:         merger,
		Registerer:     registerer,
		Confirmer:      &fakeConfirmer{response: true},
	}

	err := promoter.Demote(context.Background(), "claude-md:leech-rule")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Skill file generated.
	g.Expect(writer.written).To(HaveKey("leech-rule"))

	// Entry removed from CLAUDE.md.
	g.Expect(store.written).NotTo(ContainSubstring("Leech Rule"))
	g.Expect(store.written).To(ContainSubstring("Good Rule"))

	// Registry updated.
	g.Expect(registerer.registered).To(HaveLen(1))
	g.Expect(registerer.registered[0].ID).To(Equal("skill:leech-rule"))
	g.Expect(registerer.registered[0].SourceType).To(Equal("skill"))

	// Merged.
	g.Expect(merger.merged).To(HaveLen(1))
	g.Expect(merger.merged[0].sourceID).To(Equal("claude-md:leech-rule"))
	g.Expect(merger.merged[0].targetID).To(Equal("skill:leech-rule"))
}

// T-254: Promotion flow — skill to CLAUDE.md.
func TestT254_PromotionFlowSkillToClaudeMD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	existingClaudeMD := "# Project\n\n## Existing Rule\n\nExisting content."

	store := &fakeStore{content: existingClaudeMD}
	merger := &fakeMerger{}
	registerer := &fakeRegisterer{}

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			{
				ID:            "skill:use-targ",
				SourceType:    "skill",
				SourcePath:    "skills/use-targ.md",
				Title:         "Use Targ",
				SurfacedCount: 150,
				Evaluations: registry.EvaluationCounters{
					Followed: 120, Ignored: 30,
				},
			},
		},
	}

	generatedEntry := "## Use Targ\n\nAlways use targ.\n\n" +
		"<!-- promoted from skill:use-targ -->"

	promoter := &promote.ClaudeMDPromoter{
		Registry: reg,
		EntryGenerator: &fakeEntryGenerator{
			entry: generatedEntry,
		},
		Editor:     &promote.SectionEditor{},
		Store:      store,
		Merger:     merger,
		Registerer: registerer,
		Confirmer:  &fakeConfirmer{response: true},
		SkillLoader: func(_ string) (promote.SkillContent, error) {
			return promote.SkillContent{
				Title:   "Use Targ",
				Content: "Always use targ.",
			}, nil
		},
	}

	err := promoter.Promote(context.Background(), "skill:use-targ")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// CLAUDE.md updated with new entry.
	g.Expect(store.written).To(ContainSubstring("## Existing Rule"))
	g.Expect(store.written).To(ContainSubstring("## Use Targ"))
	g.Expect(store.written).To(ContainSubstring(
		"<!-- promoted from skill:use-targ -->",
	))

	// Registered in registry.
	g.Expect(registerer.registered).To(HaveLen(1))
	g.Expect(registerer.registered[0].ID).To(Equal("claude-md:use-targ"))
	g.Expect(registerer.registered[0].SourceType).To(Equal("claude-md"))

	// Merged.
	g.Expect(merger.merged).To(HaveLen(1))
	g.Expect(merger.merged[0].sourceID).To(Equal("skill:use-targ"))
	g.Expect(merger.merged[0].targetID).To(Equal("claude-md:use-targ"))
}

// T-255: Demotion candidates display — Leech claude-md entries listed.
func TestT255_DemotionCandidatesLeechEntriesListed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	reg := &fakeRegistry{
		entries: []registry.InstructionEntry{
			// Leech claude-md (low eff).
			makeClaudeMDEntry("claude-md:bad-rule", 50, 5, 95),
			// Working claude-md (high eff).
			makeClaudeMDEntry("claude-md:good-rule", 50, 90, 10),
			// Skill entry — should not appear in demotion candidates.
			makeSkillEntry("skill:unrelated", 200, 80, 20),
		},
	}

	promoter := &promote.ClaudeMDPromoter{Registry: reg}

	candidates, err := promoter.DemotionCandidates()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(candidates).To(HaveLen(1))
	g.Expect(candidates[0].Entry.ID).To(Equal("claude-md:bad-rule"))
}

// fakeConfirmerErr always returns an error.
type fakeConfirmerErr struct {
	err error
}

func (f *fakeConfirmerErr) Confirm(_ string) (bool, error) {
	return false, f.err
}

// --- Fakes for ClaudeMDPromoter ---

type fakeEntryGenerator struct {
	entry string
	err   error
}

func (f *fakeEntryGenerator) Generate(
	_ context.Context, _ promote.SkillContent, _ string,
) (string, error) {
	return f.entry, f.err
}

type fakeStore struct {
	content  string
	written  string
	readErr  error
	writeErr error
}

func (f *fakeStore) Read() (string, error) {
	return f.content, f.readErr
}

func (f *fakeStore) Write(content string) error {
	f.written = content

	return f.writeErr
}

func makeClaudeMDEntry(
	id string, surfacedCount, followed, ignored int,
) registry.InstructionEntry {
	return registry.InstructionEntry{
		ID:            id,
		SourceType:    "claude-md",
		SourcePath:    "CLAUDE.md",
		Title:         id,
		SurfacedCount: surfacedCount,
		Evaluations: registry.EvaluationCounters{
			Followed: followed,
			Ignored:  ignored,
		},
	}
}

func makeSkillEntry(
	id string, surfacedCount, followed, ignored int,
) registry.InstructionEntry {
	return registry.InstructionEntry{
		ID:            id,
		SourceType:    "skill",
		SourcePath:    "skills/" + id + ".md",
		Title:         id,
		SurfacedCount: surfacedCount,
		Evaluations: registry.EvaluationCounters{
			Followed: followed,
			Ignored:  ignored,
		},
	}
}
