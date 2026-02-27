package memory

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"
)

func TestMemoryContextAssembler_BeforeAfter(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	assembler := &MemoryContextAssembler{
		ClaudeMDContent:   "# CLAUDE.md\nAlways use TDD.\n",
		SkillDescriptions: []string{"commit: stages and commits code"},
		Embeddings: []string{
			"When managing teams, delegate authority",
			"When using multi-agent teams, use active polling",
		},
	}

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  "0,1",
		Preview: "Keep: When managing teams, delegate authority\nDelete: When using multi-agent teams, use active polling",
	}

	before := assembler.AssembleContext(proposal, false)
	g.Expect(before).To(gomega.ContainSubstring("Always use TDD"))
	g.Expect(before).To(gomega.ContainSubstring("delegate authority"))
	g.Expect(before).To(gomega.ContainSubstring("active polling"))

	after := assembler.AssembleContext(proposal, true)
	g.Expect(after).To(gomega.ContainSubstring("Always use TDD"))
	g.Expect(after).To(gomega.ContainSubstring("delegate authority"))
	g.Expect(after).ToNot(gomega.ContainSubstring("active polling"))
}

func TestMemoryContextAssembler_DemoteFromClaudeMD(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	assembler := &MemoryContextAssembler{
		ClaudeMDContent:   "# CLAUDE.md\nAlways use TDD.\nDomain specific rule.\n",
		SkillDescriptions: []string{},
		Embeddings:        []string{},
	}

	proposal := MaintenanceProposal{
		Tier:    "claude-md",
		Action:  "demote",
		Target:  "Domain specific rule.",
		Preview: "Domain specific rule.",
	}

	before := assembler.AssembleContext(proposal, false)
	g.Expect(before).To(gomega.ContainSubstring("Domain specific rule"))

	after := assembler.AssembleContext(proposal, true)
	g.Expect(after).ToNot(gomega.ContainSubstring("Domain specific rule"))
}

func TestMemoryContextAssembler_PromoteMovesToSkills(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	assembler := &MemoryContextAssembler{
		ClaudeMDContent:   "# CLAUDE.md\n",
		SkillDescriptions: []string{"commit: stages code"},
		Embeddings:        []string{"Use TDD always", "Promote this entry"},
	}

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "promote",
		Target:  "1",
		Preview: "Promote this entry",
	}

	before := assembler.AssembleContext(proposal, false)
	g.Expect(before).To(gomega.ContainSubstring("Promote this entry"))

	after := assembler.AssembleContext(proposal, true)
	// After promotion: removed from embeddings, added to skills
	g.Expect(after).To(gomega.ContainSubstring("Promote this entry"))
	// Verify it appears in skills section, not memories
	skillsSection := strings.Split(after, "## Memories")[0]
	memoriesSection := strings.Split(after, "## Memories")[1]

	g.Expect(skillsSection).To(gomega.ContainSubstring("Promote this entry"))
	g.Expect(memoriesSection).ToNot(gomega.ContainSubstring("Promote this entry"))
}
