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
)

// ClaudeMDExtractor extracts instructions from CLAUDE.md content.
type ClaudeMDExtractor struct {
	Content    string
	SourcePath string
}

// Extract parses CLAUDE.md bullets into instruction entries.
func (e ClaudeMDExtractor) Extract() ([]Instruction, error) {
	return extractBullets(e.Content, "claude-md", e.SourcePath)
}

// Instruction represents a single extracted instruction from a source file.
type Instruction struct {
	ID          string
	SourceType  string
	SourcePath  string
	Title       string
	Content     string
	ContentHash string
	ExtractedAt time.Time
}

// InstructionExtractor extracts instructions from a source.
type InstructionExtractor interface {
	Extract() ([]Instruction, error)
}

// MemoryMDExtractor extracts instructions from MEMORY.md content.
type MemoryMDExtractor struct {
	Content    string
	SourcePath string
}

// Extract parses MEMORY.md bullets into instruction entries.
func (e MemoryMDExtractor) Extract() ([]Instruction, error) {
	return extractBullets(e.Content, "memory-md", e.SourcePath)
}

// RuleExtractor extracts a single instruction from a rule file.
type RuleExtractor struct {
	Filename string
	Content  string
}

// Extract produces one entry for the entire rule file.
func (e RuleExtractor) Extract() ([]Instruction, error) {
	if strings.TrimSpace(e.Content) == "" {
		return nil, nil
	}

	now := time.Now()

	return []Instruction{
		{
			ID:          "rule:" + e.Filename,
			SourceType:  "rule",
			SourcePath:  e.Filename,
			Title:       e.Filename,
			Content:     e.Content,
			ContentHash: hashContent(e.Content),
			ExtractedAt: now,
		},
	}, nil
}

// SkillExtractor extracts a single instruction from a skill.
type SkillExtractor struct {
	SkillName string
	Content   string
}

// Extract produces one entry for the skill.
func (e SkillExtractor) Extract() ([]Instruction, error) {
	if strings.TrimSpace(e.Content) == "" {
		return nil, nil
	}

	now := time.Now()

	return []Instruction{
		{
			ID:          "skill:" + e.SkillName,
			SourceType:  "skill",
			SourcePath:  e.SkillName,
			Title:       e.SkillName,
			Content:     e.Content,
			ContentHash: hashContent(e.Content),
			ExtractedAt: now,
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
// into an Instruction with a stable slug-based ID.
func extractBullets(
	content, sourceType, sourcePath string,
) ([]Instruction, error) {
	if strings.TrimSpace(content) == "" {
		return nil, nil
	}

	lines := strings.Split(content, "\n")
	entries := make([]Instruction, 0, len(lines))
	now := time.Now()

	for _, line := range lines {
		text := bulletPrefix.ReplaceAllString(line, "")
		if text == line {
			continue
		}

		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		// Strip markdown formatting for slug generation
		slugSource := stripMarkdown(text)

		slug := makeSlug(slugSource)
		entryID := fmt.Sprintf("%s:%s:%s", sourceType, sourcePath, slug)

		entries = append(entries, Instruction{
			ID:          entryID,
			SourceType:  sourceType,
			SourcePath:  sourcePath,
			Title:       text,
			Content:     text,
			ContentHash: hashContent(text),
			ExtractedAt: now,
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
