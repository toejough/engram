// Package instruct implements instruction quality audit (UC-20, ARCH-48/49).
package instruct

import (
	"path/filepath"
	"strings"
)

// Exported constants.
const (
	SourceClaudeMD SourceType = "claude-md"
	SourceMemory   SourceType = "memory"
	SourceRule     SourceType = "rule"
	SourceSkill    SourceType = "skill"
)

// InstructionItem represents a single instruction extracted from a source file.
type InstructionItem struct {
	Source             SourceType `json:"source"`
	Path               string     `json:"path"`
	Content            string     `json:"content"`
	EffectivenessScore float64    `json:"effectiveness_score"` //nolint:tagliatelle // external JSON contract
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

	_ = s.scanFile(projectClaude, SourceClaudeMD, &items)

	// 2. Global CLAUDE.md (home dir)
	globalClaude := filepath.Join(projectDir, ".claude", "CLAUDE.md")

	_ = s.scanFile(globalClaude, SourceClaudeMD, &items)

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
		return err
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

// SourceType identifies where an instruction came from.
type SourceType string

// unexported constants.
const (
	salienceHigh   = 2
	salienceMedium = 3
)

// unexported variables.
var (
	salienceRank = map[SourceType]int{ //nolint:gochecknoglobals // package-level constant table
		SourceClaudeMD: 0,
		SourceRule:     1,
		SourceMemory:   salienceHigh,
		SourceSkill:    salienceMedium,
	}
)
