package instruct_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/instruct"
)

// T-215: Scanner extracts instructions from all sources.
func TestScanAll_ExtractsAllSources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/project/CLAUDE.md":                    "project instructions",
		"/project/.claude/CLAUDE.md":            "global instructions",
		"/data/memories/mem1.toml":              "memory one",
		"/data/memories/mem2.toml":              "memory two",
		"/data/memories/mem3.toml":              "memory three",
		"/project/.claude/rules/go.md":          "rule one",
		"/project/.claude/rules/style.md":       "rule two",
		"/project/.claude-plugin/skills/foo.md": "skill one",
	}

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			switch pattern {
			case "/data/memories/*.toml":
				return []string{
					"/data/memories/mem1.toml",
					"/data/memories/mem2.toml",
					"/data/memories/mem3.toml",
				}, nil
			case "/project/.claude/rules/*.md":
				return []string{
					"/project/.claude/rules/go.md",
					"/project/.claude/rules/style.md",
				}, nil
			case "/project/.claude-plugin/skills/*.md":
				return []string{
					"/project/.claude-plugin/skills/foo.md",
				}, nil
			default:
				return nil, nil
			}
		},
		EffData: map[string]float64{},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// 2 CLAUDE.md + 3 memories + 2 rules + 1 skill = 8
	const expectedCount = 8
	g.Expect(items).To(HaveLen(expectedCount))

	// Check source types
	sourceCount := map[instruct.SourceType]int{}
	for _, item := range items {
		sourceCount[item.Source]++
	}

	const expectedClaudeMD = 2
	const expectedMemories = 3
	const expectedRules = 2
	const expectedSkills = 1

	g.Expect(sourceCount[instruct.SourceClaudeMD]).To(Equal(expectedClaudeMD))
	g.Expect(sourceCount[instruct.SourceMemory]).To(Equal(expectedMemories))
	g.Expect(sourceCount[instruct.SourceRule]).To(Equal(expectedRules))
	g.Expect(sourceCount[instruct.SourceSkill]).To(Equal(expectedSkills))

	// Verify paths and content populated
	for _, item := range items {
		g.Expect(item.Path).NotTo(BeEmpty())
		g.Expect(item.Content).NotTo(BeEmpty())
	}
}

// T-216: Scanner joins effectiveness data to instructions.
func TestScanAll_JoinsEffectivenessData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files := map[string]string{
		"/project/CLAUDE.md":       "instructions",
		"/data/memories/mem1.toml": "memory with data",
		"/data/memories/mem2.toml": "memory without data",
	}

	const mem1Score = 85.5

	scanner := &instruct.Scanner{
		ReadFile: func(path string) ([]byte, error) {
			content, ok := files[path]
			if !ok {
				return nil, fmt.Errorf("not found: %s", path)
			}

			return []byte(content), nil
		},
		GlobFiles: func(pattern string) ([]string, error) {
			if pattern == "/data/memories/*.toml" {
				return []string{
					"/data/memories/mem1.toml",
					"/data/memories/mem2.toml",
				}, nil
			}

			return nil, nil
		},
		EffData: map[string]float64{
			"/data/memories/mem1.toml": mem1Score,
		},
	}

	items, err := scanner.ScanAll("/data", "/project")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Find items by path
	var mem1, mem2 *instruct.InstructionItem

	for idx := range items {
		switch items[idx].Path {
		case "/data/memories/mem1.toml":
			mem1 = &items[idx]
		case "/data/memories/mem2.toml":
			mem2 = &items[idx]
		}
	}

	g.Expect(mem1).NotTo(BeNil())

	if mem1 != nil {
		g.Expect(mem1.EffectivenessScore).To(Equal(mem1Score))
	}

	g.Expect(mem2).NotTo(BeNil())

	if mem2 != nil {
		g.Expect(mem2.EffectivenessScore).To(Equal(0.0))
	}
}
