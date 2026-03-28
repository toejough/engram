// Package instruct implements memory-only instruction quality audit (UC-26).
package instruct

import (
	"path/filepath"
	"strings"
)

// Exported constants.
const (
	SourceMemory SourceType = "memory"
)

// InstructionItem represents a single instruction extracted from a memory file.
type InstructionItem struct {
	Source             SourceType `json:"source"`
	Path               string     `json:"path"`
	Content            string     `json:"content"`
	EffectivenessScore float64    `json:"effectiveness_score"` //nolint:tagliatelle // external JSON contract
}

// Scanner extracts instructions from memory entries only.
type Scanner struct {
	ReadFile  func(string) ([]byte, error)
	GlobFiles func(pattern string) ([]string, error)
	EffData   map[string]float64
}

// ScanAll scans memory entries under dataDir.
func (s *Scanner) ScanAll(dataDir string) ([]InstructionItem, error) {
	items := make([]InstructionItem, 0)

	// Memories (*.toml in dataDir/memories/)
	memPattern := filepath.Join(dataDir, "memories", "*.toml")

	memFiles, err := s.GlobFiles(memPattern)
	if err == nil {
		for _, path := range memFiles {
			_ = s.scanFile(path, SourceMemory, &items)
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
