package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestAssembleContext_ApplyToClaudeMD_Demote verifies demote removes target line from CLAUDE.md.
func TestAssembleContext_ApplyToClaudeMD_Demote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := &MemoryContextAssembler{
		ClaudeMDContent: "line to keep\ntarget line to remove\nanother line",
	}

	proposal := MaintenanceProposal{
		Tier:   "claude-md",
		Action: "demote",
		Target: "target line",
	}

	after := a.AssembleContext(proposal, true)

	g.Expect(after).To(ContainSubstring("line to keep"))
	g.Expect(after).ToNot(ContainSubstring("target line to remove"))
	g.Expect(after).To(ContainSubstring("another line"))
}

// TestAssembleContext_ApplyToEmbeddings_Consolidate verifies consolidate removes entry.
func TestAssembleContext_ApplyToEmbeddings_Consolidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := &MemoryContextAssembler{
		Embeddings: []string{"embed-0", "embed-1", "embed-2"},
	}

	// consolidate removes the second one (index 1)
	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "consolidate",
		Target: "0,1", // remove index 1
	}

	after := a.AssembleContext(proposal, true)

	g.Expect(after).To(ContainSubstring("embed-0"))
	g.Expect(after).ToNot(ContainSubstring("embed-1"))
	g.Expect(after).To(ContainSubstring("embed-2"))
}

// TestAssembleContext_ApplyToEmbeddings_Promote verifies promote removes entry from embeddings.
func TestAssembleContext_ApplyToEmbeddings_Promote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := &MemoryContextAssembler{
		SkillDescriptions: []string{},
		Embeddings:        []string{"embed-0", "embed-1"},
	}

	// Promote index 0 → removes from embeddings, adds to skills
	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "promote",
		Target:  "0",
		Preview: "promoted skill",
	}

	after := a.AssembleContext(proposal, true)

	g.Expect(after).ToNot(ContainSubstring("embed-0"))
	g.Expect(after).To(ContainSubstring("embed-1"))
	g.Expect(after).To(ContainSubstring("promoted skill"))
}

// TestAssembleContext_ApplyToSkills_NonPromote verifies skills unchanged for non-promote actions.
func TestAssembleContext_ApplyToSkills_NonPromote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := &MemoryContextAssembler{
		SkillDescriptions: []string{"skill one", "skill two"},
		Embeddings:        []string{},
	}

	proposal := MaintenanceProposal{
		Tier:    "skills",
		Action:  "prune",
		Target:  "skill one",
		Preview: "modified content",
	}

	after := a.AssembleContext(proposal, true)

	// Skills section should still contain both original skills (prune is not handled by applyToSkills)
	g.Expect(after).To(ContainSubstring("skill one"))
	g.Expect(after).To(ContainSubstring("skill two"))
}

// TestAssembleContext_ApplyToSkills_Promote verifies skills list grows when proposal promotes.
func TestAssembleContext_ApplyToSkills_Promote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := &MemoryContextAssembler{
		SkillDescriptions: []string{"existing skill"},
		Embeddings:        []string{},
	}

	proposal := MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Preview: "new skill content",
	}

	// Before: no preview in skills
	before := a.AssembleContext(proposal, false)
	g.Expect(before).ToNot(ContainSubstring("new skill content"))

	// After: preview appears in skills section
	after := a.AssembleContext(proposal, true)

	g.Expect(after).To(ContainSubstring("new skill content"))
}
