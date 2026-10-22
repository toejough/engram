package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a12o1a_RegisterSkills_APluginSkillsKeyAndSlug verifies 792a12o1a (skill-runbook-registration.md): the
// enabled plugin superpowers@claude-plugins-official shipping skills/brainstorming/SKILL.md makes register-skills
// offer the key superpowers:brainstorming, and accepting that registration creates a note whose basename ends in
// .skill-superpowers-brainstorming.md.
func Test792a12o1a_RegisterSkills_APluginSkillsKeyAndSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const source = skillRegHome +
		"/.claude/plugins/cache/claude-plugins-official/superpowers/6.5.0/skills/brainstorming/SKILL.md"

	skills := map[string]string{"brainstorming": "---\nname: brainstorming\ndescription: d\n---\n\nThink first.\n"}

	stdout := srr3Preview(g, srr1PluginHome(g, "6.5.0", skills))

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr3Register("superpowers:brainstorming", source)}))

	names := srr4Accepted(g, srr1PluginHome(g, "6.5.0", skills), "superpowers:brainstorming")

	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	g.Expect(names[0]).To(HaveSuffix(".skill-superpowers-brainstorming.md"))
}

// Test792a12o1b_RegisterSkills_APluginsSkillAndCommandNeverCollide verifies 792a12o1b: the enabled plugin
// commit@skills shipping skills/commit/SKILL.md and commands/commit.md makes register-skills offer the keys
// commit:commit (the skill) and commit:cmd:commit (the command), each from its own file, with no conflict.
func Test792a12o1b_RegisterSkills_APluginsSkillAndCommandNeverCollide(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := srr3PluginRoot("skills", "commit", "1.0.0")
	fsys := skillsHomeFixture(nil).
		file(root+"/skills/commit/SKILL.md", "the commit skill").
		file(root+"/commands/commit.md", "the commit command")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "commit@skills", installPath: root}}, map[string]bool{"commit@skills": true})

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred(), "the skill and the command do not collide")

	g.Expect(srr3Offers(stdout)).To(ConsistOf(
		srr3Register("commit:commit", root+"/skills/commit/SKILL.md"),
		srr3Register("commit:cmd:commit", root+"/commands/commit.md"),
	))
	g.Expect(stdout).NotTo(ContainSubstring("conflict"))
}

// Test792a12o1c_RegisterSkills_AWorktreeResolvesToOwnerAndRepository verifies 792a12o1c: in a linked worktree
// directory named runbook-vs-skill, whose origin remote is ssh://git@github.com/toejough/engram.git and whose
// git common directory belongs to another checkout, the project skills' keys begin with
// project:github.com/toejough/engram: (not with the worktree's name or the common directory's) and an accepted
// registration's slugs begin with skill-project-github-com-toejough-engram-.
func Test792a12o1c_RegisterSkills_AWorktreeResolvesToOwnerAndRepository(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		worktree  = "/work/runbook-vs-skill"
		keyPrefix = "project:github.com/toejough/engram:"
		slugStart = ".skill-project-github-com-toejough-engram-"
	)

	newFS := func() *fakeSkillFS {
		return skillsHomeFixture(nil).
			file(worktree+"/.claude/skills/deploy/SKILL.md", "deploy").
			file(worktree+"/.claude/commands/ship.md", "ship")
	}
	git := srr4Git(worktree, "ssh://git@github.com/toejough/engram.git")
	git["rev-parse --path-format=absolute --git-common-dir"] = scriptedGitReply{out: "/work/main-checkout/.git\n"}
	inWorktree := srr4InRepo(worktree, git)

	stdout := srr3Preview(g, newFS(), inWorktree)

	g.Expect(srr3Offers(stdout)).To(ConsistOf(
		srr3Register(keyPrefix+"deploy", worktree+"/.claude/skills/deploy/SKILL.md"),
		srr3Register(keyPrefix+"cmd:ship", worktree+"/.claude/commands/ship.md"),
	))

	names := srr4Accepted(g, newFS(), keyPrefix+"*", inWorktree)

	g.Expect(names).To(HaveLen(2))

	for _, name := range names {
		g.Expect(name).To(ContainSubstring(slugStart), "the slug begins with skill-project-github-com-toejough-engram-")
	}
}

// Test792a12o1d_RegisterSkills_TwoSameNamedProjectsStayDistinct verifies 792a12o1d: repositories with the origins
// ssh://git@GitHub.com:22/toejough/engram.git and https://gitlab.com/toejough/engram, each with a project skill
// deploy, get the keys project:github.com/toejough/engram:deploy and project:gitlab.com/toejough/engram:deploy.
func Test792a12o1d_RegisterSkills_TwoSameNamedProjectsStayDistinct(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		skillPath          = projectTop + "/.claude/skills/deploy/SKILL.md"
		offerKeyFieldCount = 2 // "register", then the key
	)

	cases := []struct{ origin, wantKey string }{
		{"ssh://git@GitHub.com:22/toejough/engram.git", "project:github.com/toejough/engram:deploy"},
		{"https://gitlab.com/toejough/engram", "project:gitlab.com/toejough/engram:deploy"},
	}

	keys := make([]string, 0, len(cases))

	for _, testCase := range cases {
		fsys := skillsHomeFixture(nil).file(skillPath, "deploy")

		stdout := srr3Preview(g, fsys, srr4InRepo(projectTop, srr4Git(projectTop, testCase.origin)))
		offers := srr3Offers(stdout)

		g.Expect(offers).To(Equal([]string{srr3Register(testCase.wantKey, skillPath)}), testCase.origin)

		if len(offers) != 1 {
			return
		}

		offerFields := strings.Fields(offers[0])
		if len(offerFields) < offerKeyFieldCount {
			return
		}

		keys = append(keys, offerFields[1])
	}

	g.Expect(keys).To(HaveLen(len(cases)))

	if len(keys) != len(cases) {
		return
	}

	g.Expect(keys[0]).NotTo(Equal(keys[1]), "the two projects' keys are distinct")
}

// Test792a12o1e_RegisterSkills_EngramInstalledSkillsAreKeyedByTheirFolder verifies 792a12o1e: with
// ~/.claude/skills/route linking to ~/.claude/engram/skills/route and ~/.pi/agent/skills/route linking to
// ~/.pi/agent/engram/skills/route, register-skills keys the skills claude:route and pi:route (each resolved to its
// own deployed copy) and offers no bare key. The two copies differ in bytes, so neither is an alias of the other.
func Test792a12o1e_RegisterSkills_EngramInstalledSkillsAreKeyedByTheirFolder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const piDeployed = srr3PiAgent + "/engram/skills/route/SKILL.md"

	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).
		file(piDeployed, "the Pi flavour of route").
		link(srr3PiAgent+"/skills/route", srr3PiAgent+"/engram/skills/route")

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred(), "the two copies are separate keys, so there is no conflict")

	offers := srr3Offers(stdout)

	g.Expect(offers).To(ConsistOf(
		srr3Register("claude:route", skillRegSourcePath("route")),
		srr3Register("pi:route", piDeployed),
	))

	for _, offer := range offers {
		key := strings.Fields(offer)[1]
		g.Expect(key).To(ContainSubstring(":"), "no key is bare")
	}
}

// Test792a12o1f_RegisterSkills_AClaudeUserCommandsKey verifies 792a12o1f: ~/.claude/commands/audit.md, scanned,
// is offered with the key claude:cmd:audit, and an accepted registration creates a note whose slug is
// skill-claude-cmd-audit.
func Test792a12o1f_RegisterSkills_AClaudeUserCommandsKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const source = skillRegHome + "/.claude/commands/audit.md"

	newFS := func() *fakeSkillFS { return skillsHomeFixture(nil).file(source, "# Audit\n") }

	stdout := srr3Preview(g, newFS())

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr3Register("claude:cmd:audit", source)}))

	names := srr4Accepted(g, newFS(), "claude:cmd:audit")

	g.Expect(names).To(HaveLen(1))

	if len(names) != 1 {
		return
	}

	g.Expect(names[0]).To(HaveSuffix(".skill-claude-cmd-audit.md"))
}

// Test792a12o1g_RegisterSkills_APluginNamedClaudeIsReserved verifies 792a12o1g: an enabled plugin claude@m, which
// ships a skill x and a command y, is skipped with a warning naming it, and no offer is made for a claude: key
// (claude:x, claude:cmd:y) from it. The Claude user skill beside it is still offered, so the run carried on.
func Test792a12o1g_RegisterSkills_APluginNamedClaudeIsReserved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := srr3PluginRoot("m", "claude", "1.0.0")
	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).
		file(root+"/skills/x/SKILL.md", "x").
		file(root+"/commands/y.md", "y")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "claude@m", installPath: root}}, map[string]bool{"claude@m": true})

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr3Register("claude:route", skillRegSourcePath("route"))}))
	g.Expect(stdout).To(ContainSubstring("engram: plugin \"claude\" uses a reserved name; it is not scanned\n"))
}

// Test792a12o1h_RegisterSkills_ADisabledReservedNamePluginIsSilent verifies 792a12o1h: a plugin pi@m with
// enabledPlugins set to false makes no offer and prints no warning at all; the output is only the offer of the
// Claude user skill beside it.
func Test792a12o1h_RegisterSkills_ADisabledReservedNamePluginIsSilent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := srr3PluginRoot("m", "pi", "1.0.0")
	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).
		file(root+"/skills/x/SKILL.md", "x")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "pi@m", installPath: root}}, map[string]bool{"pi@m": false})

	stdout := srr3Preview(g, fsys)

	g.Expect(stdout).To(Equal("@claude-user (1)\n" + srr1RegisterOffers("route")))
}

// Test792a12o1i_RegisterSkills_APluginNamedCmdIsAnOrdinaryPlugin verifies 792a12o1i: an enabled plugin cmd@m
// shipping skills/x/SKILL.md makes register-skills offer the key cmd:x, with no warning.
func Test792a12o1i_RegisterSkills_APluginNamedCmdIsAnOrdinaryPlugin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := srr3PluginRoot("m", "cmd", "1.0.0")
	fsys := skillsHomeFixture(nil).file(root+"/skills/x/SKILL.md", "x")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "cmd@m", installPath: root}}, map[string]bool{"cmd@m": true})

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr3Register("cmd:x", root+"/skills/x/SKILL.md")}))
	g.Expect(stdout).NotTo(ContainSubstring("reserved"))
}

// Test792a12o1j_RegisterSkills_APromptTemplatesKey verifies 792a12o1j: ~/.pi/agent/prompts/review.md, scanned,
// is offered with the key pi-prompt:review.
func Test792a12o1j_RegisterSkills_APromptTemplatesKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const source = srr3PiAgent + "/prompts/review.md"

	stdout := srr3Preview(g, srr3PiHome().file(source, "# Review\n"))

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr3Register("pi-prompt:review", source)}))
}

// Test792a12p1a_RegisterSkills_APiCopyOfAnEngramSkillMakesNoOffer verifies 792a12p1a: ~/.pi/agent/skills/route and
// ~/.claude/skills/route being byte-identical, with a note keyed claude:route carrying that hash, makes
// register-skills offer nothing for claude:route or pi:route. The Pi copy counts as present for its key: a stale
// note keyed pi:route is neither refreshed nor offered for removal.
func Test792a12p1a_RegisterSkills_APiCopyOfAnEngramSkillMakesNoOffer(t *testing.T) {
	t.Parallel()

	route := srr1SkillText("route")
	claudeNote := sourcedSkillNote(offerNote{
		key: "claude:route", hash: srr1Hash(route), source: "~/.claude/engram/skills/route/SKILL.md",
	})
	staleNote := sourcedSkillNote(offerNote{key: "pi:route", hash: "stale", source: "~/.pi/agent/skills/route/SKILL.md"})

	cases := []struct {
		name  string
		notes map[string]string
	}{
		{"a note keyed claude:route", map[string]string{"1100.2026-09-26.skill-claude-route.md": claudeNote}},
		{"a stale note keyed pi:route besides", map[string]string{
			"1100.2026-09-26.skill-claude-route.md": claudeNote,
			"1101.2026-09-26.skill-pi-route.md":     staleNote,
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := skillsHomeFixture(map[string][]byte{"route": route}).
				file(srr3PiAgent+"/skills/route/SKILL.md", string(route))
			vault := newSkillAcceptFixtureVault()

			for name, content := range testCase.notes {
				vault.put(name, content)
			}

			stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(stdout).To(BeEmpty(), "no offer for claude:route or pi:route, and no removal of a pi:route note")
		})
	}
}

// Test792a12p1b_RegisterSkills_TwoPiSkillsWithOneNameAndDifferentContent verifies 792a12p1b: Pi recursion finding
// ~/.pi/agent/skills/a/foo/SKILL.md and ~/.pi/agent/skills/b/foo/SKILL.md with different bytes makes
// register-skills report a conflict for pi:foo naming both paths and make no offer for it, handle every other
// offer (here the accepted claude:route is registered) and end with the conflict failure status.
func Test792a12p1b_RegisterSkills_TwoPiSkillsWithOneNameAndDifferentContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		firstPath  = srr3PiAgent + "/skills/a/foo/SKILL.md"
		secondPath = srr3PiAgent + "/skills/b/foo/SKILL.md"
	)

	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).
		file(firstPath, "the first foo").
		file(secondPath, "the second foo")
	vault := newSkillAcceptFixtureVault()

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{Accept: []string{"claude:route", "pi:foo"}})

	g.Expect(err).To(MatchError(cli.ErrSkillOfferConflictForTest))
	g.Expect(stdout).To(ContainSubstring(
		"engram: skill key conflict: pi:foo at " + firstPath + " and " + secondPath + "\n"))
	g.Expect(stdout).To(ContainSubstring("no pending offer for skill \"pi:foo\""), "no offer is made for pi:foo")

	names, listErr := vault.ListMD(srr1Vault)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(names).To(HaveLen(1), "the other offer was handled and pi:foo registered nothing")

	if len(names) != 1 {
		return
	}

	g.Expect(names[0]).To(HaveSuffix(".skill-claude-route.md"))
}

// Test792a12p1c_RegisterSkills_TheSameProjectSkillAtTwoLevels verifies 792a12p1c: a repository with
// .claude/skills/foo/SKILL.md at its top-level and a different sub/.claude/skills/foo/SKILL.md, registered from
// sub/, gets a conflict for project:<r>:foo naming both paths and no offer for it (no nearest-wins).
func Test792a12p1c_RegisterSkills_TheSameProjectSkillAtTwoLevels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		topPath = projectTop + "/.claude/skills/foo/SKILL.md"
		subPath = projectTop + "/sub/.claude/skills/foo/SKILL.md"
	)

	fsys := skillsHomeFixture(nil).file(topPath, "the top-level foo").file(subPath, "the sub foo")
	inSub := srr4InRepo(projectTop+"/sub", srr4Git(projectTop, "git@github.com:toejough/engram.git"))

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true}, inSub)

	g.Expect(err).To(MatchError(cli.ErrSkillOfferConflictForTest))
	g.Expect(stdout).To(Equal(
		"engram: skill key conflict: " + projectScope + ":foo at " + topPath + " and " + subPath + "\n"))
}

// Test792a12p1d_RegisterSkills_APluginNameInTwoMarketplaces verifies 792a12p1d: plugins tools@alpha and
// tools@beta, both installed and enabled (each shipping a skill x), give a plugin conflict for tools, no offer for
// any tools: key and no removal of the existing tools:x note, then the conflict failure status.
func Test792a12p1d_RegisterSkills_APluginNameInTwoMarketplaces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	alpha := srr3PluginRoot("alpha", "tools", "1.0.0")
	beta := srr3PluginRoot("beta", "tools", "1.0.0")
	fsys := skillsHomeFixture(nil).
		file(alpha+"/skills/x/SKILL.md", "the alpha x").
		file(beta+"/skills/x/SKILL.md", "the beta x")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "tools@alpha", installPath: alpha}, {key: "tools@beta", installPath: beta}},
		map[string]bool{"tools@alpha": true, "tools@beta": true})

	vault := newSkillAcceptFixtureVault()
	vault.put("1100.2026-09-26.skill-tools-x.md",
		sourcedSkillNote(offerNote{key: "tools:x", hash: "h", source: alpha + "/skills/x/SKILL.md"}))

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})

	g.Expect(err).To(MatchError(cli.ErrSkillOfferConflictForTest))
	g.Expect(stdout).To(Equal("engram: plugin name conflict: tools in alpha, beta\n"),
		"the conflict is the only output: no offer for a tools: key and no removal of the tools:x note")
}

// Test792a12p1e_RegisterSkills_DivergedHarnessCopiesAreSeparateKeys verifies 792a12p1e: ~/.claude/skills/route and
// ~/.pi/agent/skills/route differing in bytes, with a note keyed claude:route carrying the Claude copy's hash and
// no note or decline covering the Pi copy's hash, makes no offer for claude:route, offers to register pi:route,
// and reports no conflict or warning.
func Test792a12p1e_RegisterSkills_DivergedHarnessCopiesAreSeparateKeys(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const piPath = srr3PiAgent + "/skills/route/SKILL.md"

	claudeRoute := srr1SkillText("route")
	fsys := skillsHomeFixture(map[string][]byte{"route": claudeRoute}).file(piPath, "the diverged Pi route")
	vault := newSkillAcceptFixtureVault()
	vault.put("1100.2026-09-26.skill-claude-route.md", sourcedSkillNote(offerNote{
		key: "claude:route", hash: srr1Hash(claudeRoute), source: "~/.claude/engram/skills/route/SKILL.md",
	}))

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(Equal("@pi-user (1)\n  would offer: " + srr3Register("pi:route", piPath) + "\n"))
}

// Test792a12p1f_RegisterSkills_TwoSyncedBucketsHoldTheSameSkill verifies 792a12p1f: two synced buckets both
// holding pdf/SKILL.md with identical bytes give one candidate anthropic-skills:pdf (from the bucket first in
// path order) and no conflict.
func Test792a12p1f_RegisterSkills_TwoSyncedBucketsHoldTheSameSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		firstBucket  = skillRegHome + "/.claude/skills/synced/4227b111_e76eb279"
		secondBucket = skillRegHome + "/.claude/skills/synced/9c0de5a4_17b2f3d8"
	)

	fsys := skillsHomeFixture(nil)

	for _, bucket := range []string{firstBucket, secondBucket} {
		fsys.file(bucket+"/manifest.json", `{"skills":[{"skillId":"pdf"}]}`).file(bucket+"/pdf/SKILL.md", "pdf")
	}

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred(), "identical copies collapse, so no conflict is reported")

	g.Expect(stdout).To(Equal("@synced (1)\n  would offer: " +
		srr3Register("anthropic-skills:pdf", firstBucket+"/pdf/SKILL.md") + "\n"))
}

// Test792a12p1g_RegisterSkills_ADivergedPluginSkillOfTheSameNameIsItsOwnSkill verifies 792a12p1g:
// skill-creator@claude-plugins-official's skill-creator and the synced skill-creator differing in bytes make
// register-skills offer both skill-creator:skill-creator and anthropic-skills:skill-creator, with no conflict.
func Test792a12p1g_RegisterSkills_ADivergedPluginSkillOfTheSameNameIsItsOwnSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const bucket = skillRegHome + "/.claude/skills/synced/4227b111_e76eb279"

	root := srr3PluginRoot("claude-plugins-official", "skill-creator", "1.0.0")
	fsys := skillsHomeFixture(nil).
		file(root+"/skills/skill-creator/SKILL.md", "the plugin's skill-creator").
		file(bucket+"/manifest.json", `{"skills":[{"skillId":"skill-creator"}]}`).
		file(bucket+"/skill-creator/SKILL.md", "the synced skill-creator")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "skill-creator@claude-plugins-official", installPath: root}},
		map[string]bool{"skill-creator@claude-plugins-official": true})

	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred(), "the two skills are separate keys, so there is no conflict")

	g.Expect(srr3Offers(stdout)).To(ConsistOf(
		srr3Register("skill-creator:skill-creator", root+"/skills/skill-creator/SKILL.md"),
		srr3Register("anthropic-skills:skill-creator", bucket+"/skill-creator/SKILL.md"),
	))
	g.Expect(stdout).NotTo(ContainSubstring("conflict"))
}

// srr4Accepted accepts token over fsys (after customize has adjusted the deps) in a fresh vault, asserts the run
// succeeds, and returns the basenames of the notes it wrote.
func srr4Accepted(g Gomega, fsys *fakeSkillFS, token string, customize ...func(*cli.SkillRegistrationDeps)) []string {
	vault := newSkillAcceptFixtureVault()

	_, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{Accept: []string{token}}, customize...)
	g.Expect(err).NotTo(HaveOccurred())

	names, listErr := vault.ListMD(srr1Vault)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return nil
	}

	return names
}

// srr4Git scripts git for a repository with the given top-level and origin remote.
func srr4Git(top, origin string) scriptedGit {
	return scriptedGit{
		"rev-parse --show-toplevel": {out: top + "\n"},
		"remote get-url origin":     {out: origin + "\n"},
	}
}

// srr4InRepo customizes a registration run to start in cwd with git answering as scripted.
func srr4InRepo(cwd string, git scriptedGit) func(*cli.SkillRegistrationDeps) {
	return func(deps *cli.SkillRegistrationDeps) {
		deps.Sources.Commander = git
		deps.Getwd = func() (string, error) { return cwd, nil }
	}
}
