// Package instruct implements instruction quality audit (UC-20, ARCH-48/49).
package instruct

import (
	"path/filepath"
	"strings"
)

// SourceType identifies where an instruction came from.
type SourceType string

const (
	SourceClaudeMD SourceType = "claude-md"
	SourceRule     SourceType = "rule"
	SourceMemory   SourceType = "memory"
	SourceSkill    SourceType = "skill"
)

// salienceRank maps source types to their salience hierarchy.
// Lower number = higher salience.
var salienceRank = map[SourceType]int{
	SourceClaudeMD: 0,
	SourceRule:     1,
	SourceMemory:   2,
	SourceSkill:    3,
}

// InstructionItem represents a single instruction extracted from a source file.
type InstructionItem struct {
	Source             SourceType `json:"source"`
	Path               string     `json:"path"`
	Content            string     `json:"content"`
	EffectivenessScore float64    `json:"effectiveness_score"`
}

// Scanner extracts instructions from all project sources.
type Scanner struct {
	ReadFile  func(string) ([]byte, error)
	GlobFiles func(pattern string) ([]string, error)
	EffData   map[string]float64
}

// ScanAll scans all instruction sources under dataDir and projectDir.
func (s *Scanner) ScanAll(dataDir, projectDir string) ([]InstructionItem, error) {
	items := make([]InstructionItem, 0)

	// 1. Project CLAUDE.md
	projectClaude := filepath.Join(projectDir, "CLAUDE.md")

	if err := s.scanFile(projectClaude, SourceClaudeMD, &items); err != nil {
		// File not found is not an error — skip silently.
	}

	// 2. Global CLAUDE.md (home dir)
	globalClaude := filepath.Join(projectDir, ".claude", "CLAUDE.md")

	if err := s.scanFile(globalClaude, SourceClaudeMD, &items); err != nil {
		// Skip silently.
	}

	// 3. Memories (*.toml in dataDir/memories/)
	memPattern := filepath.Join(dataDir, "memories", "*.toml")

	memFiles, err := s.GlobFiles(memPattern)
	if err == nil {
		for _, path := range memFiles {
			_ = s.scanFile(path, SourceMemory, &items)
		}
	}

	// 4. Rules (.claude/rules/*.md)
	rulesPattern := filepath.Join(projectDir, ".claude", "rules", "*.md")

	ruleFiles, err := s.GlobFiles(rulesPattern)
	if err == nil {
		for _, path := range ruleFiles {
			_ = s.scanFile(path, SourceRule, &items)
		}
	}

	// 5. Skills (plugin skills)
	skillsPattern := filepath.Join(projectDir, ".claude-plugin", "skills", "*.md")

	skillFiles, err := s.GlobFiles(skillsPattern)
	if err == nil {
		for _, path := range skillFiles {
			_ = s.scanFile(path, SourceSkill, &items)
		}
	}

	return items, nil
}

func (s *Scanner) scanFile(
	path string,
	source SourceType,
	items *[]InstructionItem,
) error {
	data, err := s.ReadFile(path)
	if err != nil {
		return err //nolint:wrapcheck // caller handles missing files
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}

	score := s.EffData[path]

	*items = append(*items, InstructionItem{
		Source:             source,
		Path:               path,
		Content:            content,
		EffectivenessScore: score,
	})

	return nil
}
