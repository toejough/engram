package cli_test

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestPiScanners_ReadRootIsARecordedRootProperty: every candidate a Pi
// default-folder scanner emits has a ReadRoot equal to one of its result's
// recorded resolved roots, over real and symlinked roots at every chain
// level — the invariant withAgentsChainOverrides relies on to find each
// candidate's `.agents` base.
func TestPiScanners_ReadRootIsARecordedRootProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		depth := rapid.IntRange(0, maxProjectChainDepth).Draw(rt, "depth")
		fsys := newFakeSkillFS()
		levels := make([]string, 0, depth+1)
		levels = append(levels, projectTop)

		for index := range depth {
			levels = append(levels, fmt.Sprintf("%s/l%d", levels[index], index))
		}

		cwd := levels[len(levels)-1]
		fsys.dir(cwd)

		place := func(dir, label string) {
			if !rapid.Bool().Draw(rt, "present "+label) {
				return
			}

			target := dir
			if rapid.Bool().Draw(rt, "linked "+label) {
				target = "/store/" + strings.ReplaceAll(label, "/", "_")
				fsys.link(dir, target)
			}

			fsys.file(target+"/a/SKILL.md", label).
				file(target+"/nest/b/SKILL.md", label).
				file(target+"/root.md", label)
		}

		for index, level := range levels {
			place(level+"/.agents/skills", fmt.Sprintf("agents%d", index))
		}

		place(cwd+"/.pi/skills", "pi")
		place(cwd+"/.pi/prompts", "prompts")

		scan := cli.PiProjectScan{
			Cwd: cwd, TopLevel: projectTop, UserAgentsRoot: agentsUserRoot, ScopeID: projectScope, Trusted: true,
		}

		for _, result := range []cli.SkillScanResult{
			cli.ScanAgentsProjectSkills(fsys, scan),
			cli.ScanPiProjectSkills(fsys, scan),
			cli.ScanPiProjectPrompts(fsys, scan),
		} {
			resolvedRoots := make([]string, 0, len(result.Roots))
			for _, root := range result.Roots {
				resolvedRoots = append(resolvedRoots, root.Resolved)
			}

			for _, candidate := range result.Candidates {
				g.Expect(resolvedRoots).To(ContainElement(candidate.ReadRoot), candidate.SourcePath)
			}
		}
	})
}

func TestResolveSkillSources_AbsentHarnessSourcesAreNotRead(t *testing.T) {
	t.Parallel()

	table := []struct {
		name       string
		removeDir  string
		wantAbsent []string
		wantKept   []string
		noRootsIn  []string
	}{
		{
			name:       "claude only",
			removeDir:  fakeHome + "/.pi",
			wantAbsent: []string{"pi-user", "agents-user", "pi-settings", "pi-prompt", "pi-pkg:pk", "project|pi"},
			wantKept:   []string{"claude-user", "plugin:tools", "project|"},
			noRootsIn:  []string{fakeHome + "/.pi", fakeHome + "/.agents", projectTop + "/.pi", projectTop + "/.agents"},
		},
		{
			name:       "pi only",
			removeDir:  fakeHome + "/.claude",
			wantAbsent: []string{"claude-user", "claude-cmd", "synced", "plugin:tools", "project|"},
			wantKept:   []string{"pi-user", "agents-user", "pi-pkg:pk", "project|pi"},
			noRootsIn:  []string{fakeHome + "/.claude", projectTop + "/.claude"},
		},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := resolverFixture(g).remove(testCase.removeDir)

			resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))
			g.Expect(err).NotTo(HaveOccurred())

			scopes := resolvedScopePrefixes(resolved)
			for _, absent := range testCase.wantAbsent {
				g.Expect(scopes).NotTo(ContainElement(absent))
			}

			for _, kept := range testCase.wantKept {
				g.Expect(scopes).To(ContainElement(kept))
			}

			for _, root := range resolved.Roots {
				for _, dir := range testCase.noRootsIn {
					g.Expect(root.Path).NotTo(HavePrefix(dir+"/"), root.Path)
				}
			}
		})
	}
}

func TestResolveSkillSources_ComposesEverySourceInPrecedenceOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resolved, err := cli.ResolveSkillSources(
		context.Background(), fakeHome, projectTop, resolverDeps(resolverFixture(g)),
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolvedIDs(resolved)).To(Equal([]string{
		"claude-user||skill|c4",
		"claude-cmd||command|audit",
		"synced||skill|pdf",
		"pi-user||skill|ping",
		"agents-user||skill|ag",
		"pi-settings|pi-settings|skill|fmt",
		"pi-prompt||prompt|review",
		"pi-settings|pi-settings|prompt|tidy",
		"pi-pkg:pk|pi-pkg:pk|skill|pks",
		"plugin:tools||skill|tl",
		projectScope + "||skill|openspec-propose",
		projectScope + "||command|commit",
		projectScope + "|pi|skill|pj",
		projectScope + "|agents|skill|aj",
		projectScope + "|pi-settings|skill|px",
		projectScope + "|pi-prompt|prompt|pp",
		projectScope + "|pi-settings|prompt|pq",
		projectScope + "|pi-pkg:ppk|skill|ppks",
	}))
	g.Expect(resolved.Project).To(Equal(cli.ProjectIdentity{ID: projectRemoteID, TopLevel: projectTop}))
	g.Expect(resolved.ProjectFound).To(BeTrue())
	g.Expect(resolved.PluginManifestsRead).To(BeTrue())
	g.Expect(resolved.Plugins).To(Equal([]cli.ClaudePluginStatus{{Name: "tools", Scanned: true}}))
	g.Expect(resolved.PluginConflicts).To(BeEmpty())

	for _, candidate := range resolved.Candidates {
		g.Expect(candidate.Key).To(BeEmpty(), candidate.Name)
		g.Expect(candidate.Disabled).To(BeFalse(), candidate.Name)
	}

	for _, root := range []string{
		userSkillsRoot, fakeHome + "/.claude/commands", piUserRoot, agentsUserRoot, piPromptsRoot,
		projectTop + "/.claude/skills", projectTop + "/.claude/commands",
		projectTop + "/.pi/skills", projectTop + "/.agents/skills", projectTop + "/.pi/prompts",
	} {
		scanned, found := rootScanned(resolved.SkillScanResult, root)
		g.Expect(found).To(BeTrue(), root)
		g.Expect(scanned).To(BeTrue(), root)
	}
}

func TestResolveSkillSources_HarnessProbeErrorFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g).failLstat(fakeHome + "/.claude")

	_, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))

	g.Expect(err).To(MatchError(fs.ErrPermission))
}

func TestResolveSkillSources_OutsideARepoHasNoProjectSources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := resolverDeps(resolverFixture(g))
	deps.Commander = scriptedGit{}

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, deps)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ProjectFound).To(BeFalse())
	g.Expect(resolvedScopePrefixes(resolved)).NotTo(ContainElement(HavePrefix("project")))

	for _, root := range resolved.Roots {
		g.Expect(root.Path).NotTo(HavePrefix(projectTop))
	}
}

// TestResolveSkillSources_PiOverridesDisableDefaultFolders (rulings R8/R10):
// the global settings' `!` overrides switch off default-folder files and the
// project settings' overrides switch off project default-folder files; the
// Disabled candidates stay in the result.
// TestResolveSkillSources_OverridesSkipCandidateWithUnknownLevel: a
// `.agents/skills` candidate whose ReadRoot matches no recorded level is left
// unchanged (never matched against an empty base) and reported.
func TestResolveSkillSources_OverridesSkipCandidateWithUnknownLevel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	level := projectTop + "/.agents/skills"
	result := cli.SkillScanResult{
		Candidates: []cli.SkillCandidate{
			{Name: "known", ReadRoot: level, WalkedPath: level + "/known/SKILL.md"},
			{Name: "stray", ReadRoot: "/elsewhere", WalkedPath: "/elsewhere/stray/SKILL.md"},
		},
		Roots: []cli.ScannedRoot{{Path: level, Resolved: level, Scanned: true}},
	}

	out := cli.ExportWithAgentsChainOverrides(result, []string{"!known", "!stray"})

	g.Expect(disabledNames(out.Candidates)).To(Equal([]string{"known"}))
	g.Expect(out.Warnings).To(ContainElement(ContainSubstring("/elsewhere/stray/SKILL.md")))
}

func TestResolveSkillSources_PiOverridesDisableDefaultFolders(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g).
		file(piAgentDir+"/settings.json",
			`{"skills": ["~/extra-skills", "!skills/ping", "!skills/ag2"], "prompts": ["!prompts/review.md"]}`).
		file(agentsUserRoot+"/ag2/SKILL.md", "ag2").
		file(projectTop+"/.pi/settings.json", `{"skills": ["!skills/pj", "!skills/aj"], "prompts": ["!prompts/pp.md"]}`)

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(disabledNames(resolved.Candidates)).To(ConsistOf("ping", "ag2", "review", "pj", "aj", "pp"))
	g.Expect(candidateNames(resolved.SkillScanResult)).To(ContainElements("ping", "review", "ag", "ag2", "pj", "aj", "pp"))
}

func TestResolveSkillSources_PluginConflictPassesThrough(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g)
	other := resolverPluginCache + "/other/tools/2.0"
	fsys.file(other+"/skills/tl/SKILL.md", "other tl")
	writePluginManifestsAt(g, fsys, fakeHome+"/.claude",
		[]pluginInstall{
			{key: "tools@m", installPath: resolverPluginCache + "/m/tools/1.0"},
			{key: "tools@other", installPath: other},
		},
		map[string]bool{"tools@m": true, "tools@other": true},
	)

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.PluginConflicts).To(Equal([]cli.ClaudePluginConflict{
		{Plugin: "tools", Marketplaces: []string{"m", "other"}},
	}))
	g.Expect(resolved.Plugins).To(Equal([]cli.ClaudePluginStatus{{Name: "tools", Scanned: false}}))
	g.Expect(resolved.PluginManifestsRead).To(BeTrue())
	g.Expect(resolvedScopePrefixes(resolved)).NotTo(ContainElement("plugin:tools"))
}

// TestResolveSkillSources_PrecedenceProperty: whichever subset of sources
// exists, the resolver's candidates appear in design D2 precedence order —
// the rank of each candidate's source never decreases along the result.
func TestResolveSkillSources_PrecedenceProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		fsys := resolverFixture(g)

		for _, removable := range resolverRemovableSources {
			if rapid.Bool().Draw(rt, "remove "+removable) {
				fsys.remove(removable)
			}
		}

		resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))
		g.Expect(err).NotTo(HaveOccurred())

		ranks := make([]int, 0, len(resolved.Candidates))
		for _, candidate := range resolved.Candidates {
			ranks = append(ranks, slices.Index(resolverPrecedence, resolvedID(candidate)))
		}

		g.Expect(ranks).NotTo(ContainElement(-1))
		g.Expect(slices.IsSorted(ranks)).To(BeTrue(), "%v", resolvedIDs(resolved))
	})
}

// TestResolveSkillSources_ProjectGlobstarOverrideDisablesAgentsChain: a
// project `**` override cannot be evaluated, so every candidate of each
// `.agents/skills` level is Disabled, with one warning per level.
func TestResolveSkillSources_ProjectGlobstarOverrideDisablesAgentsChain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g).
		file(projectTop+"/.agents/skills/aj2/SKILL.md", "aj2").
		file(projectTop+"/sub/.agents/skills/deep/SKILL.md", "deep").
		file(projectTop+"/sub/.pi/settings.json", `{"skills": ["!**/aj"]}`).
		file(piAgentDir+"/trust.json", `{"`+projectTop+`": true}`)

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop+"/sub", resolverDeps(fsys))

	g.Expect(err).NotTo(HaveOccurred())

	agentsDisabled := map[string]bool{}

	for _, candidate := range resolved.Candidates {
		if candidate.SourceSegment == cli.SkillSegmentAgents {
			agentsDisabled[candidate.Name] = candidate.Disabled
		}
	}

	g.Expect(agentsDisabled).To(Equal(map[string]bool{"aj": true, "aj2": true, "deep": true}))

	globstarWarnings := 0

	for _, warning := range resolved.Warnings {
		if strings.Contains(warning, "**") && strings.Contains(warning, "/.agents") {
			globstarWarnings++
		}
	}

	g.Expect(globstarWarnings).To(Equal(2), "%v", resolved.Warnings)
}

func TestResolveSkillSources_UntrustedProjectSkipsPiProjectSources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g).file(piAgentDir+"/trust.json", `{"`+projectTop+`": false}`)

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, projectTop, resolverDeps(fsys))

	g.Expect(err).NotTo(HaveOccurred())

	scopes := resolvedScopePrefixes(resolved)
	g.Expect(scopes).To(ContainElement("project|"))
	g.Expect(scopes).NotTo(ContainElement("project|pi"))
	g.Expect(scopes).NotTo(ContainElement("project|agents"))
	g.Expect(scopes).NotTo(ContainElement("project|pi-settings"))
	g.Expect(scopes).NotTo(ContainElement("project|pi-prompt"))

	for _, root := range resolved.Roots {
		g.Expect(root.Path).NotTo(HavePrefix(projectTop + "/.pi"))
		g.Expect(root.Path).NotTo(HavePrefix(projectTop + "/.agents"))
	}
}

// unexported constants.
const (
	resolverPluginCache = fakeHome + "/.claude/plugins/cache"
)

// unexported variables.
var (
	// resolverPrecedence is design D2's order over the fixture's sources.
	resolverPrecedence = []string{
		"claude-user||skill|c4",
		"claude-cmd||command|audit",
		"synced||skill|pdf",
		"pi-user||skill|ping",
		"agents-user||skill|ag",
		"pi-settings|pi-settings|skill|fmt",
		"pi-prompt||prompt|review",
		"pi-settings|pi-settings|prompt|tidy",
		"pi-pkg:pk|pi-pkg:pk|skill|pks",
		"plugin:tools||skill|tl",
		projectScope + "||skill|openspec-propose",
		projectScope + "||command|commit",
		projectScope + "|pi|skill|pj",
		projectScope + "|agents|skill|aj",
		projectScope + "|pi-settings|skill|px",
		projectScope + "|pi-prompt|prompt|pp",
		projectScope + "|pi-settings|prompt|pq",
		projectScope + "|pi-pkg:ppk|skill|ppks",
	}
	// resolverRemovableSources are fixture subtrees a property run may drop.
	resolverRemovableSources = []string{
		userSkillsRoot + "/c4",
		fakeHome + "/.claude/commands",
		userSkillsRoot + "/synced",
		piUserRoot,
		agentsUserRoot,
		fakeHome + "/extra-skills",
		piPromptsRoot,
		fakeHome + "/extra-prompts",
		piAgentDir + "/npm",
		resolverPluginCache,
		projectTop + "/.claude/skills",
		projectTop + "/.claude/commands",
		projectTop + "/.pi/skills",
		projectTop + "/.agents",
		projectTop + "/.pi/extra",
		projectTop + "/.pi/prompts",
		projectTop + "/.pi/extra-prompts",
		projectTop + "/.pi/npm",
	}
)

// remove deletes path and everything beneath it.
func (f *fakeSkillFS) remove(path string) *fakeSkillFS {
	for existing := range f.nodes {
		if existing == path || strings.HasPrefix(existing, path+"/") {
			delete(f.nodes, existing)
		}
	}

	return f
}

func resolvedID(candidate cli.SkillCandidate) string {
	return fmt.Sprintf("%s|%s|%s|%s", candidate.ScopeID, candidate.SourceSegment, candidate.Kind, candidate.Name)
}

func resolvedIDs(resolved cli.ResolvedSkillSources) []string {
	ids := make([]string, 0, len(resolved.Candidates))
	for _, candidate := range resolved.Candidates {
		ids = append(ids, resolvedID(candidate))
	}

	return ids
}

// resolvedScopePrefixes lists each candidate's scope, with a project scope
// rendered as "project|<segment>".
func resolvedScopePrefixes(resolved cli.ResolvedSkillSources) []string {
	scopes := make([]string, 0, len(resolved.Candidates))
	for _, candidate := range resolved.Candidates {
		scope := candidate.ScopeID
		if scope == projectScope {
			scope = "project|" + candidate.SourceSegment
			if strings.HasPrefix(candidate.SourceSegment, cli.SkillScopePiPkgPrefix) {
				scope = "project|pi-pkg"
			}
		}

		scopes = append(scopes, scope)
	}

	return scopes
}

func resolverDeps(fsys cli.SkillSourceFS) cli.SkillSourceDeps {
	return cli.SkillSourceDeps{
		FS: fsys,
		Commander: scriptedGit{
			"rev-parse --show-toplevel": {out: projectTop + "\n"},
			"remote get-url origin":     {out: "git@github.com:toejough/engram.git\n"},
		},
	}
}

// resolverFixture builds a home with one entry per design D2 source, both
// harnesses installed, and a trusted project at projectTop.
func resolverFixture(g Gomega) *fakeSkillFS {
	fsys := newFakeSkillFS().
		file(userSkillsRoot+"/c4/SKILL.md", "c4").
		file(fakeHome+"/.claude/commands/audit.md", "audit").
		file(userSkillsRoot+"/synced/b1/manifest.json", `{"skills": []}`).
		file(userSkillsRoot+"/synced/b1/pdf/SKILL.md", "pdf").
		file(userSkillsRoot+"/synced/.bucket-b1", "").
		file(piUserRoot+"/ping/SKILL.md", "ping").
		file(agentsUserRoot+"/ag/SKILL.md", "ag").
		file(piPromptsRoot+"/review.md", "review").
		file(piAgentDir+"/settings.json",
			`{"skills": ["~/extra-skills"], "prompts": ["~/extra-prompts"], "packages": ["npm:pk"]}`).
		file(piAgentDir+"/trust.json", `{"`+projectTop+`": true}`).
		file(fakeHome+"/extra-skills/fmt/SKILL.md", "fmt").
		file(fakeHome+"/extra-prompts/tidy.md", "tidy").
		file(piAgentDir+"/npm/node_modules/pk/package.json", `{"name": "pk"}`).
		file(piAgentDir+"/npm/node_modules/pk/skills/pks/SKILL.md", "pks").
		file(resolverPluginCache+"/m/tools/1.0/skills/tl/SKILL.md", "tl").
		file(projectTop+"/.claude/skills/openspec-propose/SKILL.md", "propose").
		file(projectTop+"/.claude/skills/commit.md", "root markdown").
		file(projectTop+"/.claude/commands/commit.md", "commit").
		file(projectTop+"/.pi/skills/pj/SKILL.md", "pj").
		file(projectTop+"/.agents/skills/aj/SKILL.md", "aj").
		file(projectTop+"/.pi/prompts/pp.md", "pp").
		file(projectTop+"/.pi/settings.json",
			`{"skills": ["extra"], "prompts": ["extra-prompts"], "packages": ["npm:ppk"]}`).
		file(projectTop+"/.pi/extra/px/SKILL.md", "px").
		file(projectTop+"/.pi/extra-prompts/pq.md", "pq").
		file(projectTop+"/.pi/npm/node_modules/ppk/package.json", `{"name": "ppk"}`).
		file(projectTop+"/.pi/npm/node_modules/ppk/skills/ppks/SKILL.md", "ppks")

	writePluginManifestsAt(g, fsys, fakeHome+"/.claude",
		[]pluginInstall{{key: "tools@m", installPath: resolverPluginCache + "/m/tools/1.0"}},
		map[string]bool{"tools@m": true},
	)

	return fsys
}
