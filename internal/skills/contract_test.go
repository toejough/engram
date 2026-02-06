package skills_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// findProjectRoot walks up from current directory to find go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// TestProducerContracts verifies all producer SKILL.md files have valid Contract sections
func TestProducerContracts(t *testing.T) {
	producers := []string{
		"pm-interview-producer",
		"pm-infer-producer",
		"design-interview-producer",
		"design-infer-producer",
		"arch-interview-producer",
		"arch-infer-producer",
		"breakdown-producer",
		"tdd-red-producer",
		"tdd-red-infer-producer",
		"tdd-green-producer",
		"tdd-refactor-producer",
		"doc-producer",
		"alignment-producer",
		"retro-producer",
		"summary-producer",
	}

	for _, producer := range producers {
		t.Run(producer, func(t *testing.T) {
			g := NewWithT(t)

			root, err := findProjectRoot()
			g.Expect(err).ToNot(HaveOccurred(), "should find project root")

			skillPath := filepath.Join(root, "skills", producer, "SKILL.md")
			content, err := os.ReadFile(skillPath)
			g.Expect(err).ToNot(HaveOccurred(), "should read SKILL.md for %s", producer)

			contract := extractContract(t, g, string(content), producer)

			// Verify contract has required sections
			g.Expect(contract).To(HaveKey("outputs"), "%s contract should have outputs section", producer)
			g.Expect(contract).To(HaveKey("traces_to"), "%s contract should have traces_to section", producer)
			g.Expect(contract).To(HaveKey("checks"), "%s contract should have checks section", producer)

			// Verify outputs structure
			outputs, ok := contract["outputs"].([]interface{})
			g.Expect(ok).To(BeTrue(), "%s outputs should be array", producer)
			g.Expect(outputs).ToNot(BeEmpty(), "%s outputs should not be empty", producer)

			for i, output := range outputs {
				outputMap, ok := output.(map[string]interface{})
				g.Expect(ok).To(BeTrue(), "%s output[%d] should be map", producer, i)
				g.Expect(outputMap).To(HaveKey("path"), "%s output[%d] should have path", producer, i)
				g.Expect(outputMap).To(HaveKey("id_format"), "%s output[%d] should have id_format", producer, i)
			}

			// Verify traces_to structure
			traces, ok := contract["traces_to"].([]interface{})
			g.Expect(ok).To(BeTrue(), "%s traces_to should be array", producer)
			g.Expect(traces).ToNot(BeEmpty(), "%s traces_to should not be empty", producer)

			// Verify checks structure
			checks, ok := contract["checks"].([]interface{})
			g.Expect(ok).To(BeTrue(), "%s checks should be array", producer)
			g.Expect(checks).ToNot(BeEmpty(), "%s checks should not be empty", producer)

			for i, check := range checks {
				checkMap, ok := check.(map[string]interface{})
				g.Expect(ok).To(BeTrue(), "%s check[%d] should be map", producer, i)
				g.Expect(checkMap).To(HaveKey("id"), "%s check[%d] should have id", producer, i)
				g.Expect(checkMap).To(HaveKey("description"), "%s check[%d] should have description", producer, i)
				g.Expect(checkMap).To(HaveKey("severity"), "%s check[%d] should have severity", producer, i)

				// Verify severity is error or warning
				severity, _ := checkMap["severity"].(string)
				g.Expect(severity).To(Or(Equal("error"), Equal("warning")), "%s check[%d] severity should be error or warning", producer, i)
			}
		})
	}
}

// TestGapAnalysisIncorporated verifies gap analysis decisions are included in contracts
func TestGapAnalysisIncorporated(t *testing.T) {
	// High priority gaps from gap-analysis.md that should be in contracts
	testCases := []struct {
		producer    string
		description string
		checkDesc   string
	}{
		{
			producer:    "pm-interview-producer",
			description: "Measurable AC",
			checkDesc:   "measurable",
		},
		{
			producer:    "pm-infer-producer",
			description: "Measurable AC",
			checkDesc:   "measurable",
		},
		{
			producer:    "design-interview-producer",
			description: "Coverage validation",
			checkDesc:   "coverage",
		},
		{
			producer:    "design-infer-producer",
			description: "Coverage validation",
			checkDesc:   "coverage",
		},
		{
			producer:    "arch-interview-producer",
			description: "Completeness criteria",
			checkDesc:   "complete",
		},
		{
			producer:    "arch-infer-producer",
			description: "Completeness criteria",
			checkDesc:   "complete",
		},
		{
			producer:    "breakdown-producer",
			description: "Architecture coverage and testable criteria",
			checkDesc:   "testable",
		},
		{
			producer:    "tdd-red-producer",
			description: "No implementation in RED",
			checkDesc:   "implementation",
		},
		{
			producer:    "tdd-red-infer-producer",
			description: "No implementation in RED",
			checkDesc:   "implementation",
		},
		{
			producer:    "doc-producer",
			description: "Example validation",
			checkDesc:   "example",
		},
		{
			producer:    "summary-producer",
			description: "Accuracy verification",
			checkDesc:   "accuracy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.producer+"_"+tc.description, func(t *testing.T) {
			g := NewWithT(t)

			root, err := findProjectRoot()
			g.Expect(err).ToNot(HaveOccurred())

			skillPath := filepath.Join(root, "skills", tc.producer, "SKILL.md")
			content, err := os.ReadFile(skillPath)
			g.Expect(err).ToNot(HaveOccurred())

			contract := extractContract(t, g, string(content), tc.producer)

			// Verify gap is addressed in checks
			checks, ok := contract["checks"].([]interface{})
			g.Expect(ok).To(BeTrue())

			found := false
			for _, check := range checks {
				checkMap, _ := check.(map[string]interface{})
				desc, _ := checkMap["description"].(string)
				if strings.Contains(strings.ToLower(desc), tc.checkDesc) {
					found = true
					break
				}
			}

			g.Expect(found).To(BeTrue(), "%s contract should include check for %s", tc.producer, tc.description)
		})
	}
}

// TestContractFormat verifies contracts follow CONTRACT.md standard
func TestContractFormat(t *testing.T) {
	producers := []string{
		"pm-interview-producer",
		"pm-infer-producer",
		"design-interview-producer",
		"design-infer-producer",
		"arch-interview-producer",
		"arch-infer-producer",
		"breakdown-producer",
		"tdd-red-producer",
		"tdd-red-infer-producer",
		"tdd-green-producer",
		"tdd-refactor-producer",
		"doc-producer",
		"alignment-producer",
		"retro-producer",
		"summary-producer",
	}

	for _, producer := range producers {
		t.Run(producer, func(t *testing.T) {
			g := NewWithT(t)

			root, err := findProjectRoot()
			g.Expect(err).ToNot(HaveOccurred())

			skillPath := filepath.Join(root, "skills", producer, "SKILL.md")
			content, err := os.ReadFile(skillPath)
			g.Expect(err).ToNot(HaveOccurred())

			// Verify Contract section exists
			g.Expect(string(content)).To(ContainSubstring("## Contract"), "%s should have ## Contract section", producer)

			// Verify YAML code block exists after Contract section
			lines := strings.Split(string(content), "\n")
			contractIdx := -1
			for i, line := range lines {
				if strings.Contains(line, "## Contract") {
					contractIdx = i
					break
				}
			}

			g.Expect(contractIdx).To(BeNumerically(">=", 0), "%s should have Contract section", producer)

			// Find YAML block after Contract section
			foundYamlBlock := false
			for i := contractIdx + 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) == "```yaml" {
					foundYamlBlock = true
					break
				}
				// Stop if we hit another section
				if strings.HasPrefix(lines[i], "## ") {
					break
				}
			}

			g.Expect(foundYamlBlock).To(BeTrue(), "%s should have ```yaml block after ## Contract", producer)
		})
	}
}

// extractContract extracts and parses the contract YAML from a SKILL.md file
func extractContract(t *testing.T, g *WithT, content, producer string) map[string]interface{} {
	lines := strings.Split(content, "\n")

	// Find Contract section
	contractIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "## Contract") {
			contractIdx = i
			break
		}
	}

	g.Expect(contractIdx).To(BeNumerically(">=", 0), "%s should have ## Contract section", producer)

	// Find YAML block
	yamlStart := -1
	yamlEnd := -1
	for i := contractIdx + 1; i < len(lines); i++ {
		if yamlStart == -1 && strings.TrimSpace(lines[i]) == "```yaml" {
			yamlStart = i + 1
			continue
		}
		if yamlStart != -1 && strings.TrimSpace(lines[i]) == "```" {
			yamlEnd = i
			break
		}
	}

	g.Expect(yamlStart).To(BeNumerically(">", 0), "%s should have ```yaml block", producer)
	g.Expect(yamlEnd).To(BeNumerically(">", yamlStart), "%s should have closing ``` for yaml block", producer)

	// Extract YAML content
	yamlContent := strings.Join(lines[yamlStart:yamlEnd], "\n")

	// Parse YAML
	var parsed map[string]interface{}
	err := yaml.Unmarshal([]byte(yamlContent), &parsed)
	g.Expect(err).ToNot(HaveOccurred(), "%s contract YAML should parse", producer)

	// Verify top-level contract key
	contract, ok := parsed["contract"].(map[string]interface{})
	g.Expect(ok).To(BeTrue(), "%s should have contract: key at top level", producer)

	return contract
}
