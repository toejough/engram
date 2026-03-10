// Package crossref contains extractors for non-memory instruction sources
// (CLAUDE.md, MEMORY.md, rules, skills). Used by the cross-source scanner (UC-29).
package crossref

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"engram/internal/registry"
)

// ClaudeMDExtractor extracts instructions from CLAUDE.md content.
type ClaudeMDExtractor struct {
	Content    string
	SourcePath string
}

// Extract parses CLAUDE.md bullets into instruction entries.
func (e ClaudeMDExtractor) Extract() ([]registry.InstructionEntry, error) {
	return extractBullets(e.Content, "claude-md", e.SourcePath)
}

// InstructionExtractor extracts registrable instructions from a source.
type InstructionExtractor interface {
	Extract() ([]registry.InstructionEntry, error)
}

// MemoryMDExtractor extracts instructions from MEMORY.md content.
type MemoryMDExtractor struct {
	Content    string
	SourcePath string
}

// Extract parses MEMORY.md bullets into instruction entries.
func (e MemoryMDExtractor) Extract() ([]registry.InstructionEntry, error) {
	return extractBullets(e.Content, "memory-md", e.SourcePath)
}

// RuleExtractor extracts a single instruction from a rule file.
type RuleExtractor struct {
	Filename string
	Content  string
}

// Extract produces one entry for the entire rule file.
func (e RuleExtractor) Extract() ([]registry.InstructionEntry, error) {
	if strings.TrimSpace(e.Content) == "" {
		return nil, nil
	}

	now := time.Now()

	return []registry.InstructionEntry{
		{
			ID:           "rule:" + e.Filename,
			SourceType:   "rule",
			SourcePath:   e.Filename,
			Title:        e.Filename,
			ContentHash:  hashContent(e.Content),
			RegisteredAt: now,
			UpdatedAt:    now,
		},
	}, nil
}

// SkillExtractor extracts a single instruction from a skill.
type SkillExtractor struct {
	SkillName string
	Content   string
}

// Extract produces one entry for the skill.
func (e SkillExtractor) Extract() ([]registry.InstructionEntry, error) {
	if strings.TrimSpace(e.Content) == "" {
		return nil, nil
	}

	now := time.Now()

	return []registry.InstructionEntry{
		{
			ID:           "skill:" + e.SkillName,
			SourceType:   "skill",
			SourcePath:   e.SkillName,
			Title:        e.SkillName,
			ContentHash:  hashContent(e.Content),
			RegisteredAt: now,
			UpdatedAt:    now,
		},
	}, nil
}

// unexported constants.
const (
	maxSlugWords = 4
)

// unexported variables.
var (
	bulletPrefix = regexp.MustCompile(`^\s*[-*]\s+`)
)

// extractBullets parses markdown content for bullet items and converts each
// into an InstructionEntry with a stable slug-based ID.
func extractBullets(
	content, sourceType, sourcePath string,
) ([]registry.InstructionEntry, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	lines := strings.Split(content, "\n")
	entries := make([]registry.InstructionEntry, 0, len(lines))
	now := time.Now()

	for _, line := range lines {
		if !bulletPrefix.MatchString(line) {
			continue
		}

		text := bulletPrefix.ReplaceAllString(line, "")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		// Strip markdown formatting for slug generation
		slugSource := stripMarkdown(text)

		slug := makeSlug(slugSource)
		entryID := fmt.Sprintf("%s:%s:%s", sourceType, sourcePath, slug)

		entries = append(entries, registry.InstructionEntry{
			ID:           entryID,
			SourceType:   sourceType,
			SourcePath:   sourcePath,
			Title:        text,
			ContentHash:  hashContent(text),
			RegisteredAt: now,
			UpdatedAt:    now,
		})
	}

	return entries, nil
}

// hashContent produces a SHA-256 hex digest of content.
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))

	return hex.EncodeToString(h[:])
}

// makeSlug generates a stable slug from the first maxSlugWords words,
// lowercased and hyphen-joined.
func makeSlug(text string) string {
	// Remove markdown formatting
	clean := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}

		return -1
	}, text)

	words := strings.Fields(strings.ToLower(clean))

	if len(words) > maxSlugWords {
		words = words[:maxSlugWords]
	}

	return strings.Join(words, "-")
}

// stripMarkdown removes bold/italic markers and colons after bold text.
func stripMarkdown(text string) string {
	// Remove ** markers
	result := strings.ReplaceAll(text, "**", "")
	// Remove trailing colon+space after what was a bold label
	result = strings.TrimSpace(result)

	return result
}
