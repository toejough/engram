package memory_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestCheckClaudeMDSize(t *testing.T) {
	t.Run("returns nil when file under threshold", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		claudeMD := filepath.Join(dir, "CLAUDE.md")

		// Create file with 50 lines
		content := strings.Repeat("line\n", 50)
		err := os.WriteFile(claudeMD, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns nil when file exactly at threshold", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		claudeMD := filepath.Join(dir, "CLAUDE.md")

		// Create file with 100 lines
		content := strings.Repeat("line\n", 100)
		err := os.WriteFile(claudeMD, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error when file exceeds threshold", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		claudeMD := filepath.Join(dir, "CLAUDE.md")

		// Create file with 101 lines
		content := strings.Repeat("line\n", 101)
		err := os.WriteFile(claudeMD, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("CLAUDE.md exceeds"))
		g.Expect(err.Error()).To(ContainSubstring("101"))
		g.Expect(err.Error()).To(ContainSubstring("100"))
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		claudeMD := filepath.Join(dir, "nonexistent.md")

		err := memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("counts lines correctly with empty lines", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()
		claudeMD := filepath.Join(dir, "CLAUDE.md")

		// Create file with 50 non-empty and 50 empty lines
		var lines []string
		for range 50 {
			lines = append(lines, "content")
			lines = append(lines, "")
		}

		content := strings.Join(lines, "\n") + "\n"
		err := os.WriteFile(claudeMD, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Now add one more line to exceed
		content = strings.Join(append(lines, "extra"), "\n") + "\n"
		err = os.WriteFile(claudeMD, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckClaudeMDSize(memory.CheckClaudeMDSizeOpts{
			ClaudeMDPath: claudeMD,
			MaxLines:     100,
		})
		g.Expect(err).To(HaveOccurred())
	})
}

func TestCheckEmbeddingMetadata(t *testing.T) {
	t.Run("returns nil when hook JSON does not contain memory learn command", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		hookJSON := `{"tool_name": "Bash", "tool_input": {"command": "echo hello"}}`
		stdin := strings.NewReader(hookJSON)

		err := memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
			MemoryRoot: dir,
			Stdin:      stdin,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error when most recent embedding has empty enriched_content and no observation_type", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Initialize DB and insert a learning with empty metadata
		db, err := memory.InitDBForTest(dir)
		g.Expect(err).ToNot(HaveOccurred())

		defer db.Close()

		// Insert an embedding with empty enriched_content and observation_type
		_, err = db.Exec(`INSERT INTO embeddings (content, source, enriched_content, observation_type, concepts, confidence, embedding_id)
			VALUES ('test', 'memory', '', '', '', 1.0, 1)`)
		g.Expect(err).ToNot(HaveOccurred())

		hookJSON := `{"tool_name": "Bash", "tool_input": {"command": "projctl memory learn test"}}`
		stdin := strings.NewReader(hookJSON)

		err = memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
			MemoryRoot: dir,
			Stdin:      stdin,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("enriched_content"))
		g.Expect(err.Error()).To(ContainSubstring("observation_type"))
	})

	t.Run("returns error when most recent embedding has empty concepts", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		db, err := memory.InitDBForTest(dir)
		g.Expect(err).ToNot(HaveOccurred())

		defer db.Close()

		_, err = db.Exec(`INSERT INTO embeddings (content, source, enriched_content, observation_type, concepts, confidence, embedding_id)
			VALUES ('test', 'memory', 'enriched', 'pattern', '', 1.0, 1)`)
		g.Expect(err).ToNot(HaveOccurred())

		hookJSON := `{"tool_name": "Bash", "tool_input": {"command": "projctl memory learn test"}}`
		stdin := strings.NewReader(hookJSON)

		err = memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
			MemoryRoot: dir,
			Stdin:      stdin,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("concepts"))
	})

	t.Run("returns error when confidence out of range", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		db, err := memory.InitDBForTest(dir)
		g.Expect(err).ToNot(HaveOccurred())

		defer db.Close()

		_, err = db.Exec(`INSERT INTO embeddings (content, source, enriched_content, observation_type, concepts, confidence, embedding_id)
			VALUES ('test', 'memory', 'enriched', 'pattern', 'concept1,concept2', 1.5, 1)`)
		g.Expect(err).ToNot(HaveOccurred())

		hookJSON := `{"tool_name": "Bash", "tool_input": {"command": "projctl memory learn test"}}`
		stdin := strings.NewReader(hookJSON)

		err = memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
			MemoryRoot: dir,
			Stdin:      stdin,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("confidence"))
	})

	t.Run("returns nil when all metadata is valid", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		db, err := memory.InitDBForTest(dir)
		g.Expect(err).ToNot(HaveOccurred())

		defer db.Close()

		_, err = db.Exec(`INSERT INTO embeddings (content, source, enriched_content, observation_type, concepts, confidence, embedding_id)
			VALUES ('test', 'memory', 'enriched content', 'pattern', 'concept1,concept2', 0.85, 1)`)
		g.Expect(err).ToNot(HaveOccurred())

		hookJSON := `{"tool_name": "Bash", "tool_input": {"command": "projctl memory learn --message test"}}`
		stdin := strings.NewReader(hookJSON)

		err = memory.CheckEmbeddingMetadata(memory.CheckEmbeddingMetaOpts{
			MemoryRoot: dir,
			Stdin:      stdin,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func TestCheckSkillContract(t *testing.T) {
	t.Run("returns nil when no SKILL.md files exist", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		err := memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns nil when SKILL.md files are valid", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Create a valid SKILL.md
		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		validContent := `---
description: Test skill description with enough characters to meet the minimum length requirement of 100 characters total for validation purposes.
---

# Test Skill

This is a valid skill file with proper frontmatter.
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(validContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// Note: File just created has recent mtime, so it will be checked

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error when SKILL.md missing frontmatter", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		invalidContent := `# Test Skill

No frontmatter here.
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(invalidContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("missing YAML frontmatter"))
	})

	t.Run("returns error when SKILL.md missing description field", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		invalidContent := `---
other_field: value
---

# Test Skill
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(invalidContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("missing description"))
	})

	t.Run("returns error when SKILL.md contains TODO", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		invalidContent := `---
description: Test skill with a description that is long enough to pass the minimum length validation requirement of 100 characters.
---

# Test Skill

TODO: Finish this section
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(invalidContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("contains TODO"))
	})

	t.Run("returns error when SKILL.md contains FIXME", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		invalidContent := `---
description: Test skill with a description that is long enough to pass the minimum length validation requirement of 100 characters.
---

# Test Skill

FIXME: Fix this bug
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(invalidContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("contains FIXME"))
	})

	t.Run("warns when token count exceeds 2500", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		// Create a large content (approximate 2500+ tokens = ~10000+ characters)
		largeContent := "---\ndescription: Test skill with a description that is long enough to pass the minimum length validation requirement.\n---\n\n" + strings.Repeat("word ", 3000)
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(largeContent), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		// This should not error, just warn (we'll check stderr in integration tests)
		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		// No error expected for warning
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("returns error when description is too short (< 100 chars)", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		shortDesc := `---
description: Short.
---

# Test Skill

Body content here.
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(shortDesc), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("description too short"))
	})

	t.Run("returns error when description is exactly 99 chars", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		// Create exactly 99 char description
		desc99 := strings.Repeat("x", 99)
		content := `---
description: ` + desc99 + `
---

# Test Skill
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("description too short"))
	})

	t.Run("accepts description with exactly 100 chars", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		// Create exactly 100 char description
		desc100 := strings.Repeat("x", 100)
		content := `---
description: ` + desc100 + `
---

# Test Skill
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(content), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("accepts multiline description with structured sections", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		skillDir := filepath.Join(dir, "test-skill")
		err := os.MkdirAll(skillDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		structuredDesc := `---
description: |
  Core: Produces tests for acceptance criteria following TDD red phase.
  Triggers: write tests, tdd red, failing tests.
  Domains: testing, tdd, quality.
---

# Test Skill
`
		skillFile := filepath.Join(skillDir, "SKILL.md")
		err = os.WriteFile(skillFile, []byte(structuredDesc), 0644)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.CheckSkillContract(memory.CheckSkillContractOpts{
			SkillsDir: dir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})
}

func TestLearnWithValidation(t *testing.T) {
	t.Run("rejects learning when LLM enrichment incomplete and extractor provided", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// Mock extractor that returns incomplete data
		extractor := &mockExtractor{
			extractResult: &memory.Observation{
				Type:     "",         // Empty
				Concepts: []string{}, // Empty
			},
		}

		err := memory.Learn(memory.LearnOpts{
			Message:    "test learning",
			MemoryRoot: dir,
			Extractor:  extractor,
		})
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("validation failed"))
	})

	t.Run("allows learning when no extractor provided", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		// No extractor = --no-llm mode, should allow
		err := memory.Learn(memory.LearnOpts{
			Message:    "test learning",
			MemoryRoot: dir,
			Extractor:  nil,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})

	t.Run("allows learning when LLM enrichment complete", func(t *testing.T) {
		g := NewWithT(t)
		dir := t.TempDir()

		extractor := &mockExtractor{
			extractResult: &memory.Observation{
				Type:     "pattern",
				Concepts: []string{"concept1", "concept2"},
			},
		}

		err := memory.Learn(memory.LearnOpts{
			Message:    "test learning",
			MemoryRoot: dir,
			Extractor:  extractor,
		})
		g.Expect(err).ToNot(HaveOccurred())
	})
}

// mockExtractor is a mock LLM extractor for testing
type mockExtractor struct {
	extractResult *memory.Observation
	extractError  error
}

func (m *mockExtractor) AddRationale(ctx context.Context, content string) (string, error) {
	// Mock implementation: just return content as-is
	return content, nil
}

func (m *mockExtractor) Curate(ctx context.Context, query string, candidates []memory.QueryResult) ([]memory.CuratedResult, error) {
	// Mock implementation: just return empty list
	return nil, nil
}

func (m *mockExtractor) Decide(ctx context.Context, message string, existing []memory.ExistingMemory) (*memory.IngestDecision, error) {
	return &memory.IngestDecision{Action: memory.IngestAdd}, nil
}

func (m *mockExtractor) Extract(ctx context.Context, message string) (*memory.Observation, error) {
	return m.extractResult, m.extractError
}

func (m *mockExtractor) Filter(ctx context.Context, query string, candidates []memory.QueryResult) ([]memory.FilterResult, error) {
	return nil, nil
}

func (m *mockExtractor) PostEval(_ context.Context, _, _ string) (*memory.PostEvalResult, error) {
	return &memory.PostEvalResult{Faithfulness: 0.5, Signal: "positive"}, nil
}

func (m *mockExtractor) Rewrite(ctx context.Context, content string) (string, error) {
	// Mock implementation: just return content as-is
	return content, nil
}

func (m *mockExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	// Mock implementation: just return empty string
	return "", nil
}
