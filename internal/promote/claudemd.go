package promote

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"engram/internal/registry"
)

// ClaudeMDCandidate represents an entry eligible for CLAUDE.md promotion or demotion.
type ClaudeMDCandidate struct {
	Entry registry.InstructionEntry
}

// ClaudeMDEditor manipulates sections in CLAUDE.md content.
type ClaudeMDEditor interface {
	AddEntry(content, entry string) (string, error)
	RemoveEntry(content, entryID string) (string, error)
	ExtractEntry(content, entryID string) (string, error)
}

// ClaudeMDEntryGenerator generates a CLAUDE.md entry from a skill.
type ClaudeMDEntryGenerator interface {
	Generate(ctx context.Context, skill SkillContent, existingClaudeMD string) (string, error)
}

// ClaudeMDPromoter orchestrates skill->CLAUDE.md promotion and CLAUDE.md->skill demotion.
type ClaudeMDPromoter struct {
	Registry       RegistryReader
	EntryGenerator ClaudeMDEntryGenerator
	SkillGenerator SkillGenerator
	Editor         ClaudeMDEditor
	Store          ClaudeMDStore
	SkillWriter    SkillWriter
	Merger         RegistryMerger
	Registerer     RegistryRegisterer
	Confirmer      Confirmer
	SkillLoader    func(path string) (SkillContent, error)
}

// Demote executes CLAUDE.md->skill demotion for a single candidate.
//
//nolint:cyclop,funlen // demotion pipeline
func (p *ClaudeMDPromoter) Demote(
	ctx context.Context, candidateID string,
) error {
	entry, err := p.Registry.Get(candidateID)
	if err != nil {
		return fmt.Errorf("getting candidate: %w", err)
	}

	if entry == nil {
		return fmt.Errorf("getting candidate: %w", registry.ErrNotFound)
	}

	claudeMD, err := p.Store.Read()
	if err != nil {
		return fmt.Errorf("reading CLAUDE.md: %w", err)
	}

	entryContent, err := p.Editor.ExtractEntry(claudeMD, candidateID)
	if err != nil {
		return fmt.Errorf("extracting entry: %w", err)
	}

	mem := MemoryContent{
		Title:    entry.Title,
		Content:  entryContent,
		Keywords: []string{Slugify(entry.Title)},
	}

	skillContent, err := p.SkillGenerator.Generate(ctx, mem)
	if err != nil {
		return fmt.Errorf("generating skill: %w", err)
	}

	confirmed, err := p.Confirmer.Confirm(skillContent)
	if err != nil {
		return fmt.Errorf("confirming demotion: %w", err)
	}

	if !confirmed {
		return nil
	}

	skillName := Slugify(entry.Title)
	skillID := "skill:" + skillName

	path, err := p.SkillWriter.Write(skillName, skillContent)
	if err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}

	updated, err := p.Editor.RemoveEntry(claudeMD, candidateID)
	if err != nil {
		return fmt.Errorf("removing entry: %w", err)
	}

	writeErr := p.Store.Write(updated)
	if writeErr != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", writeErr)
	}

	regEntry := registry.InstructionEntry{
		ID:         skillID,
		SourceType: "skill",
		SourcePath: path,
		Title:      entry.Title,
	}

	regErr := p.Registerer.Register(regEntry)
	if regErr != nil {
		return fmt.Errorf("registering skill: %w", regErr)
	}

	mergeErr := p.Merger.Merge(candidateID, skillID)
	if mergeErr != nil {
		return fmt.Errorf("merging registry: %w", mergeErr)
	}

	return nil
}

// DemotionCandidates returns CLAUDE.md entries eligible for demotion to skill.
// Filters: source_type == "claude-md", Leech quadrant.
func (p *ClaudeMDPromoter) DemotionCandidates() ([]ClaudeMDCandidate, error) {
	entries, err := p.Registry.List()
	if err != nil {
		return nil, fmt.Errorf("listing registry: %w", err)
	}

	candidates := make([]ClaudeMDCandidate, 0, len(entries))

	for i := range entries {
		entry := entries[i]
		if entry.SourceType != "claude-md" {
			continue
		}

		quadrant := registry.Classify(
			&entry,
			defaultSurfacingThreshold,
			defaultEffectivenessThreshold,
		)
		if quadrant != registry.Leech {
			continue
		}

		candidates = append(candidates, ClaudeMDCandidate{Entry: entry})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Entry.SurfacedCount > candidates[j].Entry.SurfacedCount
	})

	return candidates, nil
}

// Promote executes skill->CLAUDE.md promotion for a single candidate.
//
//nolint:cyclop,funlen // promotion pipeline
func (p *ClaudeMDPromoter) Promote(
	ctx context.Context, candidateID string,
) error {
	entry, err := p.Registry.Get(candidateID)
	if err != nil {
		return fmt.Errorf("getting candidate: %w", err)
	}

	if entry == nil {
		return fmt.Errorf("getting candidate: %w", registry.ErrNotFound)
	}

	skill, err := p.SkillLoader(entry.SourcePath)
	if err != nil {
		return fmt.Errorf("loading skill: %w", err)
	}

	existing, err := p.Store.Read()
	if err != nil {
		return fmt.Errorf("reading CLAUDE.md: %w", err)
	}

	generatedEntry, err := p.EntryGenerator.Generate(ctx, skill, existing)
	if err != nil {
		return fmt.Errorf("generating entry: %w", err)
	}

	confirmed, err := p.Confirmer.Confirm(generatedEntry)
	if err != nil {
		return fmt.Errorf("confirming promotion: %w", err)
	}

	if !confirmed {
		return nil
	}

	updated, err := p.Editor.AddEntry(existing, generatedEntry)
	if err != nil {
		return fmt.Errorf("adding entry: %w", err)
	}

	writeErr := p.Store.Write(updated)
	if writeErr != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", writeErr)
	}

	claudeMDID := "claude-md:" + Slugify(skill.Title)

	regEntry := registry.InstructionEntry{
		ID:         claudeMDID,
		SourceType: "claude-md",
		SourcePath: "CLAUDE.md",
		Title:      skill.Title,
	}

	regErr := p.Registerer.Register(regEntry)
	if regErr != nil {
		return fmt.Errorf("registering entry: %w", regErr)
	}

	mergeErr := p.Merger.Merge(candidateID, claudeMDID)
	if mergeErr != nil {
		return fmt.Errorf("merging registry: %w", mergeErr)
	}

	return nil
}

// PromotionCandidates returns skills eligible for CLAUDE.md promotion.
// Filters: source_type == "skill", Working quadrant, surfaced_count >= threshold.
func (p *ClaudeMDPromoter) PromotionCandidates(
	threshold int,
) ([]ClaudeMDCandidate, error) {
	entries, err := p.Registry.List()
	if err != nil {
		return nil, fmt.Errorf("listing registry: %w", err)
	}

	candidates := make([]ClaudeMDCandidate, 0, len(entries))

	for i := range entries {
		entry := entries[i]
		if entry.SourceType != "skill" {
			continue
		}

		if entry.SurfacedCount < threshold {
			continue
		}

		quadrant := registry.Classify(
			&entry,
			defaultSurfacingThreshold,
			defaultEffectivenessThreshold,
		)
		if quadrant != registry.Working {
			continue
		}

		candidates = append(candidates, ClaudeMDCandidate{Entry: entry})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Entry.SurfacedCount > candidates[j].Entry.SurfacedCount
	})

	return candidates, nil
}

// ClaudeMDStore reads and writes CLAUDE.md content.
type ClaudeMDStore interface {
	Read() (string, error)
	Write(content string) error
}

// SkillContent holds the content of a skill file for CLAUDE.md entry generation.
type SkillContent struct {
	Title   string
	Content string
}

// FormatClaudeMDEntry generates a CLAUDE.md entry from skill content
// with a source traceability comment.
func FormatClaudeMDEntry(skill SkillContent, entryID string) string {
	var buf strings.Builder

	buf.WriteString("## " + skill.Title + "\n\n")
	buf.WriteString(skill.Content + "\n\n")
	buf.WriteString("<!-- promoted from " + entryID + " -->\n")

	return buf.String()
}
