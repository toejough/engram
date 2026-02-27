package memory

import (
	"strconv"
	"strings"
)

// Embedding represents a memory entry with metadata.
// This is a simplified view used for scenario derivation.
type Embedding struct {
	ID       int64
	Content  string
	Metadata map[string]any
}

// TestScenario represents a test case for validating skill effectiveness.
type TestScenario struct {
	Description     string // Human-readable description of what's being tested
	SkillName       string // Name of the skill being tested
	SkillContent    string // The actual skill content to inject (for GREEN phase)
	SuccessCriteria string // Pattern to match in response indicating success
	FailureCriteria string // Pattern indicating the failure mode we're preventing
}

// DeriveScenarioFromEmbeddings creates a test scenario from a cluster of related embeddings.
// The scenario is designed to test whether a skill candidate prevents the failure mode
// captured in the embeddings.
func DeriveScenarioFromEmbeddings(embeddings []Embedding) TestScenario {
	if len(embeddings) == 0 {
		return TestScenario{
			Description:     "Generic test scenario",
			SkillName:       "generic-skill",
			SkillContent:    "Follow best practices.",
			SuccessCriteria: "best practice",
			FailureCriteria: "error",
		}
	}

	// Extract common theme from embeddings
	var corrections, antiPatterns []string

	for _, emb := range embeddings {
		typ, _ := emb.Metadata["type"].(string)
		switch typ {
		case "correction":
			corrections = append(corrections, emb.Content)
		case "anti_pattern":
			antiPatterns = append(antiPatterns, emb.Content)
		}
	}

	// Build skill name from first embedding content (simplified)
	skillName := "skill-" + strconv.FormatInt(embeddings[0].ID, 10)

	if len(corrections) > 0 {
		// Extract first few words as skill name
		words := strings.Fields(corrections[0])
		if len(words) > 3 {
			words = words[:3]
		}

		skillName = strings.ToLower(strings.Join(words, "-"))
	}

	// Build skill content from corrections
	var skillContent strings.Builder
	skillContent.WriteString("# Skill: " + skillName + "\n\n")

	for _, corr := range corrections {
		skillContent.WriteString("- " + corr + "\n")
	}

	// Derive success criteria (what we want to see)
	successCriteria := "correct"

	if len(corrections) > 0 {
		// Extract key phrase from first correction
		words := strings.Fields(corrections[0])
		if len(words) > 0 {
			successCriteria = strings.ToLower(words[0])
		}
	}

	// Derive failure criteria (what we want to avoid)
	failureCriteria := "incorrect"

	if len(antiPatterns) > 0 {
		// Extract key phrase from first anti-pattern
		words := strings.Fields(antiPatterns[0])
		if len(words) > 0 {
			failureCriteria = strings.ToLower(words[0])
		}
	}

	return TestScenario{
		Description:     "Test whether skill prevents common mistakes in " + skillName,
		SkillName:       skillName,
		SkillContent:    skillContent.String(),
		SuccessCriteria: successCriteria,
		FailureCriteria: failureCriteria,
	}
}
