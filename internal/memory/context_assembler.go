package memory

import (
	"strconv"
	"strings"
)

// MemoryContextAssembler builds simulated context windows for behavioral testing.
// It holds the current state of each tier and can simulate before/after for a proposal.
type MemoryContextAssembler struct {
	ClaudeMDContent   string
	SkillDescriptions []string
	Embeddings        []string
}

// AssembleContext returns the context window as it would appear before or after the proposal.
func (a *MemoryContextAssembler) AssembleContext(proposal MaintenanceProposal, applied bool) string {
	var sb strings.Builder

	// CLAUDE.md section
	sb.WriteString("## CLAUDE.md (always loaded)\n")
	claudemd := a.ClaudeMDContent
	if applied && proposal.Tier == "claude-md" {
		claudemd = a.applyToClaudeMD(proposal)
	}
	sb.WriteString(claudemd)
	sb.WriteString("\n")

	// Skills section
	sb.WriteString("## Skills (matched by context)\n")
	skills := a.SkillDescriptions
	if applied && proposal.Tier == "skills" {
		skills = a.applyToSkills(proposal)
	}
	if applied && proposal.Tier == "embeddings" && proposal.Action == "promote" {
		skills = append(skills, proposal.Preview)
	}
	for _, s := range skills {
		sb.WriteString("- ")
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Embeddings section
	sb.WriteString("## Memories (retrieved by similarity)\n")
	embeddings := a.Embeddings
	if applied && proposal.Tier == "embeddings" {
		embeddings = a.applyToEmbeddings(proposal)
	}
	for _, e := range embeddings {
		sb.WriteString("- ")
		sb.WriteString(e)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (a *MemoryContextAssembler) applyToEmbeddings(p MaintenanceProposal) []string {
	switch p.Action {
	case "consolidate":
		parts := strings.SplitN(p.Target, ",", 2)
		if len(parts) == 2 {
			removeIdx, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && removeIdx < len(a.Embeddings) {
				result := make([]string, 0, len(a.Embeddings)-1)
				for i, e := range a.Embeddings {
					if i != removeIdx {
						result = append(result, e)
					}
				}
				return result
			}
		}
	case "promote":
		idx, err := strconv.Atoi(strings.TrimSpace(p.Target))
		if err == nil && idx < len(a.Embeddings) {
			result := make([]string, 0, len(a.Embeddings)-1)
			for i, e := range a.Embeddings {
				if i != idx {
					result = append(result, e)
				}
			}
			return result
		}
	}
	return a.Embeddings
}

func (a *MemoryContextAssembler) applyToSkills(p MaintenanceProposal) []string {
	if p.Action == "promote" {
		return append(a.SkillDescriptions, p.Preview)
	}
	return a.SkillDescriptions
}

func (a *MemoryContextAssembler) applyToClaudeMD(p MaintenanceProposal) string {
	if p.Action == "demote" {
		lines := strings.Split(a.ClaudeMDContent, "\n")
		var result []string
		for _, line := range lines {
			if !strings.Contains(line, p.Target) {
				result = append(result, line)
			}
		}
		return strings.Join(result, "\n")
	}
	return a.ClaudeMDContent
}
