//go:build sqlite_fts5

package memory_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestSkillDiscovery verifies that skill descriptions are semantically discoverable
// by testing that common trigger queries match the expected skills.
func TestSkillDiscovery(t *testing.T) {
	g := NewWithT(t)

	// Test cases: query -> expected best-match skill
	testCases := []struct {
		query         string
		expectedSkill string
	}{
		{"tdd red phase write failing tests", "tdd-red-producer"},
		{"make tests pass tdd green", "tdd-green-producer"},
		{"commit changes to git", "commit"},
		{"validate producer output quality", "qa"},
		{"decompose architecture into implementation tasks", "breakdown-producer"},
		{"gather requirements via interview", "pm-interview-producer"},
		{"gather architecture decisions", "arch-interview-producer"},
		{"produce structured project plan", "plan-producer"},
		{"refactor while keeping tests passing", "tdd-refactor-producer"},
		{"validate traceability across artifacts", "alignment-producer"},
	}

	// Find skills directory (go up from test file location to project root)
	cwd, err := os.Getwd()
	g.Expect(err).ToNot(HaveOccurred())

	projectRoot := filepath.Join(cwd, "../..")
	skillsDir := filepath.Join(projectRoot, "skills")

	// Load all SKILL.md descriptions
	skillDescriptions, err := loadSkillDescriptions(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skillDescriptions).ToNot(BeEmpty(), "should find skill files")

	// Get model path and initialize ONNX runtime
	modelPath, err := memory.GetModelPath()
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize ONNX runtime (required before generating embeddings)
	err = memory.InitializeONNXRuntimeForTest(filepath.Dir(modelPath))
	g.Expect(err).ToNot(HaveOccurred())

	// Generate embeddings for all skill descriptions
	skillEmbeddings := make(map[string][]float32)

	for skillName, description := range skillDescriptions {
		embedding, _, _, err := memory.GenerateEmbeddingONNX(description, modelPath)
		g.Expect(err).ToNot(HaveOccurred(), "failed to generate embedding for skill: "+skillName)
		skillEmbeddings[skillName] = embedding
	}

	// Test each query
	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			g := NewWithT(t)

			// Generate embedding for query
			queryEmbedding, _, _, err := memory.GenerateEmbeddingONNX(tc.query, modelPath)
			g.Expect(err).ToNot(HaveOccurred())

			// Find best match
			var (
				bestSkill string
				bestScore float32 = -1.0
			)

			for skillName, skillEmbedding := range skillEmbeddings {
				score := cosineSimilarity(queryEmbedding, skillEmbedding)
				if score > bestScore {
					bestScore = score
					bestSkill = skillName
				}
			}

			// Assert best match is expected skill
			g.Expect(bestSkill).To(Equal(tc.expectedSkill),
				"query '%s' should match '%s' (got '%s' with score %.3f)",
				tc.query, tc.expectedSkill, bestSkill, bestScore)
		})
	}
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// extractDescriptionFromFrontmatter extracts the description field value from YAML frontmatter.
// Handles both single-line and multi-line (|) YAML format.
func extractDescriptionFromFrontmatter(content string) string {
	// Find frontmatter boundaries
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return ""
	}

	frontmatter := parts[0]
	lines := strings.Split(frontmatter, "\n")

	var (
		inDescription bool
		description   strings.Builder
	)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this line starts the description field
		if strings.HasPrefix(trimmed, "description:") {
			inDescription = true
			// Check for single-line description
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if rest != "" && rest != "|" {
				// Single-line description
				return rest
			}
			// Multi-line description (|), continue to next lines
			continue
		}

		// If in description, collect indented lines
		if inDescription {
			// Check if line is indented (part of multi-line value)
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				description.WriteString(strings.TrimSpace(line))
				description.WriteString("\n")
			} else if i > 0 && trimmed != "" {
				// Non-indented line means end of multi-line value
				break
			}
		}
	}

	return strings.TrimSpace(description.String())
}

// loadSkillDescriptions reads all SKILL.md files and extracts their description fields.
func loadSkillDescriptions(skillsDir string) (map[string]string, error) {
	descriptions := make(map[string]string)

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")

		// Check if SKILL.md exists
		if _, err := os.Stat(skillFile); os.IsNotExist(err) {
			continue
		}

		// Read file
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		// Extract description from frontmatter
		description := extractDescriptionFromFrontmatter(string(data))
		if description != "" {
			descriptions[skillName] = description
		}
	}

	return descriptions, nil
}
