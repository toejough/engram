package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b30c11a9_ResolveSkillSources_TrustedProjectScansPiAndAgentsLevels verifies 792b30c11a9
// (skill-source-discovery.md): for a trusted project, resolving the sources scans both the `.pi` levels (the
// working directory only) and the `.agents/skills` chain (the working directory and every ancestor up to the
// top-level), each existing level recorded as its own scanned root, with the pi and agents segments stamped.
// It runs over the in-memory skill file system; the working directory is a subdirectory so the `.pi` scan
// (working directory only) and the `.agents` scan (the chain) cannot be mistaken for each other.
func Test792b30c11a9_ResolveSkillSources_TrustedProjectScansPiAndAgentsLevels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		subDir        = projectTop + "/sub"
		subPiSkills   = subDir + "/.pi/skills"
		subPiPrompts  = subDir + "/.pi/prompts"
		subAgents     = subDir + "/.agents/skills"
		topAgents     = projectTop + "/.agents/skills"
		topPiSkills   = projectTop + "/.pi/skills"
		topPiPrompts  = projectTop + "/.pi/prompts"
		piSkillName   = "ssd1pi"
		subAgentsName = "ssd1sub"
		topAgentsName = "aj"
		promptName    = "ssd1prompt"
	)

	fsys := resolverFixture(g).
		file(subPiSkills+"/"+piSkillName+"/SKILL.md", "pi skill").
		file(subPiPrompts+"/"+promptName+".md", "pi prompt").
		file(subAgents+"/"+subAgentsName+"/SKILL.md", "agents skill below the top-level")

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, subDir, resolverDeps(fsys))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	namesBySegment := map[string][]string{}

	for _, candidate := range resolved.Candidates {
		if candidate.ScopeID != projectScope {
			continue
		}

		switch candidate.SourceSegment {
		case cli.SkillSegmentPi, cli.SkillSegmentAgents, cli.SkillScopePiPrompt:
			namesBySegment[candidate.SourceSegment] = append(namesBySegment[candidate.SourceSegment], candidate.Name)
		}
	}

	g.Expect(namesBySegment).To(Equal(map[string][]string{
		cli.SkillSegmentPi:     {piSkillName},
		cli.SkillSegmentAgents: {topAgentsName, subAgentsName},
		cli.SkillScopePiPrompt: {promptName},
	}))

	for _, root := range []string{subPiSkills, subPiPrompts, subAgents, topAgents} {
		scanned, found := rootScanned(resolved.SkillScanResult, root)
		g.Expect(found).To(BeTrue(), root)
		g.Expect(scanned).To(BeTrue(), root)
	}

	// The `.pi` levels are the working directory's only: the top-level's are never read.
	for _, root := range []string{topPiSkills, topPiPrompts} {
		_, found := rootScanned(resolved.SkillScanResult, root)
		g.Expect(found).To(BeFalse(), root)
	}
}

// Test792b30c3a5_ScanClaudeUserSkills_DanglingLinkIsSkipped verifies 792b30c3a5 (skill-source-discovery.md):
// a skills-root child that is a link whose target is missing is skipped without failing: it yields no
// candidate and no warning, the root is still scanned (so no removal is blocked by a mere dangling link),
// and the other children are still found. Cases: the dangling link alone, beside a real skill, and the
// second hop of a link chain missing. It runs over the in-memory skill file system, which models a dangling
// link as a link node with no target node, as the real tree does.
func Test792b30c3a5_ScanClaudeUserSkills_DanglingLinkIsSkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fsys      func() *fakeSkillFS
		wantNames []string
	}{
		{
			name: "the dangling link alone",
			fsys: func() *fakeSkillFS {
				return newFakeSkillFS().link(userSkillsRoot+"/qa", "/missing/qa")
			},
			wantNames: []string{},
		},
		{
			name: "beside a real skill",
			fsys: func() *fakeSkillFS {
				return newFakeSkillFS().
					file(userSkillsRoot+"/c4/SKILL.md", "c4").
					link(userSkillsRoot+"/qa", "/missing/qa")
			},
			wantNames: []string{"c4"},
		},
		{
			name: "a chain whose second hop is missing",
			fsys: func() *fakeSkillFS {
				return newFakeSkillFS().
					file(userSkillsRoot+"/c4/SKILL.md", "c4").
					link(userSkillsRoot+"/qa", "/hop/one").
					link("/hop/one", "/hop/missing")
			},
			wantNames: []string{"c4"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			result := cli.ScanClaudeUserSkills(testCase.fsys(), userSkillsRoot)

			g.Expect(candidateNames(result)).To(Equal(testCase.wantNames))
			g.Expect(result.Warnings).To(BeEmpty())

			scanned, found := rootScanned(result, userSkillsRoot)
			g.Expect(found).To(BeTrue())
			g.Expect(scanned).To(BeTrue())
		})
	}
}
