package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// --- plan-producer SKILL.md ---

// TestPlanProducer_SkillExists verifies plan-producer SKILL.md exists
func TestPlanProducer_SkillExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "plan-producer", "SKILL.md")
	_, err = os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred(), "plan-producer SKILL.md should exist")
}

// TestPlanProducer_Frontmatter verifies plan-producer has correct frontmatter
func TestPlanProducer_Frontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "plan-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("name: plan-producer"), "should have name: plan-producer")
	g.Expect(text).To(ContainSubstring("model: opus"), "should specify model: opus")
	g.Expect(text).To(ContainSubstring("user-invocable: true"), "should be user-invocable")
	g.Expect(text).To(ContainSubstring("role: producer"), "should have role: producer")
	g.Expect(text).To(ContainSubstring("phase: plan"), "should have phase: plan")
}

// TestPlanProducer_GatherSynthesizeProduce verifies plan-producer follows GATHER -> SYNTHESIZE -> PRODUCE
func TestPlanProducer_GatherSynthesizeProduce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "plan-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("GATHER"), "should contain GATHER section")
	g.Expect(text).To(ContainSubstring("SYNTHESIZE"), "should contain SYNTHESIZE section")
	g.Expect(text).To(ContainSubstring("PRODUCE"), "should contain PRODUCE section")
}

// TestPlanProducer_Contract verifies plan-producer has a Contract section
func TestPlanProducer_Contract(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "plan-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("Contract"), "should contain Contract section")
	g.Expect(text).To(ContainSubstring("plan.md"), "should reference plan.md as output")
}

// TestPlanProducer_PlanMode verifies plan-producer uses plan mode for interactive review
func TestPlanProducer_PlanMode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "plan-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(Or(
		ContainSubstring("EnterPlanMode"),
		ContainSubstring("plan mode"),
	), "should mention plan mode or EnterPlanMode")
	g.Expect(text).To(ContainSubstring("interactive"), "should document interactive user review")
}

// --- evaluation-producer SKILL.md ---

// TestEvaluationProducer_SkillExists verifies evaluation-producer SKILL.md exists
func TestEvaluationProducer_SkillExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "evaluation-producer", "SKILL.md")
	_, err = os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred(), "evaluation-producer SKILL.md should exist")
}

// TestEvaluationProducer_Frontmatter verifies evaluation-producer has correct frontmatter
func TestEvaluationProducer_Frontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "evaluation-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("name: evaluation-producer"), "should have name: evaluation-producer")
	g.Expect(text).To(ContainSubstring("model: sonnet"), "should specify model: sonnet")
	g.Expect(text).To(ContainSubstring("user-invocable: true"), "should be user-invocable")
	g.Expect(text).To(ContainSubstring("role: producer"), "should have role: producer")
	g.Expect(text).To(ContainSubstring("phase: evaluation"), "should have phase: evaluation")
}

// TestEvaluationProducer_GatherSynthesizeProduce verifies evaluation-producer follows GATHER -> SYNTHESIZE -> PRODUCE
func TestEvaluationProducer_GatherSynthesizeProduce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "evaluation-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("GATHER"), "should contain GATHER section")
	g.Expect(text).To(ContainSubstring("SYNTHESIZE"), "should contain SYNTHESIZE section")
	g.Expect(text).To(ContainSubstring("PRODUCE"), "should contain PRODUCE section")
}

// TestEvaluationProducer_Contract verifies evaluation-producer has a Contract section
func TestEvaluationProducer_Contract(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "evaluation-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("Contract"), "should contain Contract section")
	g.Expect(text).To(ContainSubstring("evaluation.md"), "should reference evaluation.md as output")
}

// TestEvaluationProducer_TracesToUpstream verifies evaluation-producer traces to upstream artifacts
func TestEvaluationProducer_TracesToUpstream(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root, err := findProjectRoot()
	g.Expect(err).ToNot(HaveOccurred())

	skillPath := filepath.Join(root, "skills", "evaluation-producer", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	g.Expect(err).ToNot(HaveOccurred())

	text := string(content)

	g.Expect(text).To(ContainSubstring("traces_to"), "should have traces_to in contract")
	g.Expect(text).To(ContainSubstring("requirements.md"), "should trace to requirements.md")
	g.Expect(text).To(ContainSubstring("design.md"), "should trace to design.md")
	g.Expect(text).To(ContainSubstring("architecture.md"), "should trace to architecture.md")
	g.Expect(text).To(ContainSubstring("tasks.md"), "should trace to tasks.md")
}
