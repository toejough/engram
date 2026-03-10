// Package promote implements memory→skill tier promotion (UC-4, ARCH-62/63).
package promote

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"engram/internal/registry"
)

// Candidate represents a memory eligible for promotion.
type Candidate struct {
	Entry registry.InstructionEntry
}

// Confirmer asks for user confirmation before proceeding.
type Confirmer interface {
	Confirm(preview string) (bool, error)
}

// MemoryContent holds the content of a memory file for skill generation.
type MemoryContent struct {
	Title       string
	Content     string
	Principle   string
	AntiPattern string
	Keywords    []string
}

// MemoryRemover deletes a memory TOML file.
type MemoryRemover interface {
	Remove(path string) error
}

// Promoter orchestrates memory→skill promotion.
type Promoter struct {
	Registry     RegistryReader
	Generator    SkillGenerator
	Writer       SkillWriter
	Merger       RegistryMerger
	Remover      MemoryRemover
	Registerer   RegistryRegisterer
	Confirmer    Confirmer
	MemoryLoader func(path string) (*MemoryContent, error)
	Content      string // Pre-generated skill content (ARCH-78); skips Generator.
	SkipConfirm  bool   // Skip Confirmer.Confirm (ARCH-78).
}

// Candidates returns memory entries eligible for skill promotion,
// filtered by surfaced_count >= threshold and quadrant != Insufficient.
// Results are sorted by surfaced_count descending.
func (p *Promoter) Candidates(threshold int) ([]Candidate, error) {
	entries, err := p.Registry.List()
	if err != nil {
		return nil, fmt.Errorf("listing registry: %w", err)
	}

	candidates := make([]Candidate, 0, len(entries))

	for i := range entries {
		entry := entries[i]
		if entry.SourceType != "memory" {
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
		if quadrant == registry.Insufficient {
			continue
		}

		candidates = append(candidates, Candidate{Entry: entry})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Entry.SurfacedCount > candidates[j].Entry.SurfacedCount
	})

	return candidates, nil
}

// Promote executes the full promotion flow for a single candidate.
//
//nolint:cyclop,funlen // promotion orchestration
func (p *Promoter) Promote(ctx context.Context, candidateID string) error {
	entry, err := p.Registry.Get(candidateID)
	if err != nil {
		return fmt.Errorf("getting candidate: %w", err)
	}

	if entry == nil {
		return fmt.Errorf("getting candidate: %w", registry.ErrNotFound)
	}

	mem, err := p.MemoryLoader(entry.SourcePath)
	if err != nil {
		return fmt.Errorf("loading memory: %w", err)
	}

	skillContent := p.Content
	if skillContent == "" {
		generated, genErr := p.Generator.Generate(ctx, *mem)
		if genErr != nil {
			return fmt.Errorf("generating skill: %w", genErr)
		}

		skillContent = generated
	}

	if !p.SkipConfirm {
		confirmed, confirmErr := p.Confirmer.Confirm(skillContent)
		if confirmErr != nil {
			return fmt.Errorf("confirming promotion: %w", confirmErr)
		}

		if !confirmed {
			return nil
		}
	}

	skillName := Slugify(mem.Title)
	skillID := "skill:" + skillName

	path, err := p.Writer.Write(skillName, skillContent)
	if err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}

	regEntry := registry.InstructionEntry{
		ID:         skillID,
		SourceType: "skill",
		SourcePath: path,
		Title:      mem.Title,
	}

	regErr := p.Registerer.Register(regEntry)
	if regErr != nil {
		return fmt.Errorf("registering skill: %w", regErr)
	}

	mergeErr := p.Merger.Merge(candidateID, skillID)
	if mergeErr != nil {
		return fmt.Errorf("merging registry: %w", mergeErr)
	}

	rmErr := p.Remover.Remove(entry.SourcePath)
	if rmErr != nil {
		return fmt.Errorf("removing memory: %w", rmErr)
	}

	return nil
}

// RegistryMerger merges a source entry into a target in the registry.
type RegistryMerger interface {
	Merge(sourceID, targetID string) error
}

// RegistryReader reads entries from the instruction registry.
type RegistryReader interface {
	List() ([]registry.InstructionEntry, error)
	Get(id string) (*registry.InstructionEntry, error)
}

// RegistryRegisterer registers a new entry in the registry.
type RegistryRegisterer interface {
	Register(entry registry.InstructionEntry) error
}

// SkillGenerator generates skill file content from a memory.
type SkillGenerator interface {
	Generate(ctx context.Context, memory MemoryContent) (string, error)
}

// SkillWriter writes a skill file to the plugin skills directory.
type SkillWriter interface {
	Write(name, content string) (string, error)
}

// FormatSkill generates a skill file from memory content using the DES-34 template.
func FormatSkill(mem MemoryContent) string {
	var buf strings.Builder

	description := "Use when " + strings.Join(mem.Keywords, ", ")

	buf.WriteString("---\n")
	fmt.Fprintf(&buf, "description: %q\n", description)
	buf.WriteString("---\n\n")
	buf.WriteString("# " + mem.Title + "\n\n")
	buf.WriteString(mem.Principle + "\n")

	if mem.AntiPattern != "" {
		buf.WriteString("\n## What to avoid\n\n")
		buf.WriteString(mem.AntiPattern + "\n")
	}

	buf.WriteString("\n## Context\n\n")
	buf.WriteString(mem.Content + "\n")

	return buf.String()
}

// Slugify converts a title to a hyphenated lowercase slug.
func Slugify(title string) string {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}

		return -1
	}, title)

	words := strings.Fields(strings.ToLower(clean))

	const maxSlugWords = 5
	if len(words) > maxSlugWords {
		words = words[:maxSlugWords]
	}

	return strings.Join(words, "-")
}

// unexported constants.
const (
	defaultEffectivenessThreshold = 50.0
	defaultSurfacingThreshold     = 3
)
