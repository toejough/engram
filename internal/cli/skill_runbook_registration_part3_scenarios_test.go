package cli_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a12k1a_RegisterSkills_APiPromptTemplate verifies 792a12k1a (skill-runbook-registration.md):
// ~/.pi/agent/prompts/review.md makes register-skills offer the Pi prompt template review, keyed
// pi-prompt:review, from that file.
func Test792a12k1a_RegisterSkills_APiPromptTemplate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().file(srr3PiAgent+"/prompts/review.md", "# Review\n")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi-prompt:review", srr3PiAgent+"/prompts/review.md"),
	}))
}

// Test792a12k1b_RegisterSkills_APiSettingsSkillPath verifies 792a12k1b: Pi's global settings.json with
// "skills": ["~/extra-skills"] and ~/extra-skills/fmt/SKILL.md makes register-skills offer fmt as a Pi settings
// skill, keyed pi-settings:fmt, from that file.
func Test792a12k1b_RegisterSkills_APiSettingsSkillPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/settings.json", `{"skills": ["~/extra-skills"]}`).
		file(skillRegHome+"/extra-skills/fmt/SKILL.md", "fmt")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi-settings:fmt", skillRegHome+"/extra-skills/fmt/SKILL.md"),
	}))
}

// Test792a12k1c_RegisterSkills_PackageSkillsComeFromItsManifest verifies 792a12k1c: settings.json listing
// "npm:pi-intercom", whose package.json has "pi": {"skills": ["./skills"]} with skills/pi-intercom/SKILL.md,
// makes register-skills offer pi-intercom as a skill of package pi-intercom (key pi-pkg:pi-intercom:pi-intercom).
func Test792a12k1c_RegisterSkills_PackageSkillsComeFromItsManifest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/settings.json", `{"packages": ["npm:pi-intercom"]}`).
		file(srr3NpmModules+"/pi-intercom/package.json", `{"name": "pi-intercom", "pi": {"skills": ["./skills"]}}`).
		file(srr3NpmModules+"/pi-intercom/skills/pi-intercom/SKILL.md", "intercom")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi-pkg:pi-intercom:pi-intercom", srr3NpmModules+"/pi-intercom/skills/pi-intercom/SKILL.md"),
	}))
}

// Test792a12k1d_RegisterSkills_AStringFormPackageWithAPiKeyIgnoresConventionDirectories verifies 792a12k1d:
// settings.json listing "npm:x", whose package.json has "pi": {"extensions": ["./index.ts"]} while the package
// also holds skills/fmt/SKILL.md, makes no offer for any skill of package x. The control, a package.json with
// no pi key, falls back to the convention directory and offers fmt, so the silence is owed to the pi key.
func Test792a12k1d_RegisterSkills_AStringFormPackageWithAPiKeyIgnoresConventionDirectories(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, manifest string
		want           []string
	}{
		{"a pi key without skills", `{"pi": {"extensions": ["./index.ts"]}}`, nil},
		{"control: no pi key", `{"name": "x"}`, []string{
			srr3Register("pi-pkg:x:fmt", srr3NpmModules+"/x/skills/fmt/SKILL.md"),
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := srr3PiHome().
				file(srr3PiAgent+"/settings.json", `{"packages": ["npm:x"]}`).
				file(srr3NpmModules+"/x/package.json", testCase.manifest).
				file(srr3NpmModules+"/x/skills/fmt/SKILL.md", "fmt")

			stdout := srr3Preview(g, fsys)

			g.Expect(srr3Offers(stdout)).To(Equal(srr3Nonnil(testCase.want)))
		})
	}
}

// Test792a12k1e_RegisterSkills_AnObjectFormOmittedTypeFallsBackPerType verifies 792a12k1e: settings.json listing
// {"source": "npm:x", "extensions": []}, for a package whose package.json has a pi key without skills and which
// holds skills/fmt/SKILL.md, makes register-skills offer fmt from skills/, because the object form falls back to
// the convention directory for the omitted skills type.
func Test792a12k1e_RegisterSkills_AnObjectFormOmittedTypeFallsBackPerType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/settings.json", `{"packages": [{"source": "npm:x", "extensions": []}]}`).
		file(srr3NpmModules+"/x/package.json", `{"pi": {"extensions": ["./index.ts"]}}`).
		file(srr3NpmModules+"/x/skills/fmt/SKILL.md", "fmt")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi-pkg:x:fmt", srr3NpmModules+"/x/skills/fmt/SKILL.md"),
	}))
}

// Test792a12k1f_RegisterSkills_AnObjectFormEmptySkillsFilter verifies 792a12k1f: settings.json listing
// {"source": "npm:pi-intercom", "skills": []} makes no offer for the package's pi-intercom skill, and an
// existing note keyed to it is not offered for removal because the skill counts as present (disabled). The
// control, the same package with its skill file gone, does offer to remove that note.
func Test792a12k1f_RegisterSkills_AnObjectFormEmptySkillsFilter(t *testing.T) {
	t.Parallel()

	const (
		key      = "pi-pkg:pi-intercom:pi-intercom"
		basename = "1.2026-09-21.skill-pi-pkg-pi-intercom-pi-intercom"
		source   = srr3NpmModules + "/pi-intercom/skills/pi-intercom/SKILL.md"
	)

	cases := []struct {
		name         string
		withSkill    bool
		wantRemovals []string
	}{
		{"the skill is present and filtered off", true, []string{}},
		{"control: the skill is gone", false, []string{"remove " + key + " (" + source + ")"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := srr3PiHome().
				file(srr3PiAgent+"/settings.json", `{"packages": [{"source": "npm:pi-intercom", "skills": []}]}`).
				file(srr3NpmModules+"/pi-intercom/package.json", `{"pi": {"skills": ["./skills"]}}`).
				dir(srr3NpmModules + "/pi-intercom/skills")

			if testCase.withSkill {
				fsys.file(source, "intercom")
			}

			vault := newSkillAcceptFixtureVault()
			vault.put(basename+".md", sourcedSkillNote(offerNote{key: key, hash: "h", source: source}))

			stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr3Offers(stdout)).To(Equal(testCase.wantRemovals),
				"no registration is offered, and removal is offered only when the skill is really gone")
		})
	}
}

// Test792a12k1g_RegisterSkills_APiRootMarkdownSkill verifies 792a12k1g: ~/.pi/agent/skills/notes.md, a root file,
// makes register-skills offer it as the Pi user skill notes (key pi:notes).
func Test792a12k1g_RegisterSkills_APiRootMarkdownSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().file(srr3PiAgent+"/skills/notes.md", "# Notes\n")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi:notes", srr3PiAgent+"/skills/notes.md"),
	}))
}

// Test792a12k1h_RegisterSkills_AConfiguredPackageWithAMissingPathContributesNothing verifies 792a12k1h: Pi's
// settings.json listing a local package path that does not exist makes register-skills scan no skill from it
// and not fail; the Pi user skill beside it is still offered, so the run carried on.
func Test792a12k1h_RegisterSkills_AConfiguredPackageWithAMissingPathContributesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/settings.json", `{"packages": ["~/pkgs/gone"]}`).
		file(srr3PiAgent+"/skills/ok/SKILL.md", "ok")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("pi:ok", srr3PiAgent+"/skills/ok/SKILL.md"),
	}))
}

// Test792a12l1a_RegisterSkills_PluginJSONCommandsReplaceTheCommandsDirectory verifies 792a12l1a: an enabled
// plugin's plugin.json has "commands": ["./commands/setup.md"] and its commands/ directory also holds
// configure.md; register-skills offers only setup (hud:cmd:setup) as that plugin's command.
func Test792a12l1a_RegisterSkills_PluginJSONCommandsReplaceTheCommandsDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := srr3PluginRoot("hud", "hud", "0.8.0")
	fsys := skillsHomeFixture(nil).
		file(root+"/.claude-plugin/plugin.json", `{"name":"hud","commands":["./commands/setup.md"]}`).
		file(root+"/commands/setup.md", "setup").
		file(root+"/commands/configure.md", "configure")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "hud@hud", installPath: root}}, map[string]bool{"hud@hud": true})

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("hud:cmd:setup", root+"/commands/setup.md"),
	}))
}

// Test792a12l1b_RegisterSkills_ADisabledPluginContributesNothing verifies 792a12l1b: a plugin in
// installed_plugins.json whose enabledPlugins entry is false contributes none of its skills or commands, while
// an enabled plugin beside it is still scanned.
func Test792a12l1b_RegisterSkills_ADisabledPluginContributesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	disabledRoot := srr3PluginRoot("m", "hookify", "1.0.0")
	enabledRoot := srr3PluginRoot("m", "fine", "1.0.0")
	fsys := skillsHomeFixture(nil).
		file(disabledRoot+"/skills/writing-rules/SKILL.md", "writing-rules").
		file(disabledRoot+"/commands/hookify.md", "hookify").
		file(enabledRoot+"/skills/ok/SKILL.md", "ok")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "hookify@m", installPath: disabledRoot}, {key: "fine@m", installPath: enabledRoot}},
		map[string]bool{"hookify@m": false, "fine@m": true})

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("fine:ok", enabledRoot+"/skills/ok/SKILL.md"),
	}))
}

// Test792a12l1c_RegisterSkills_APluginWithoutAnEnabledPluginsEntryAndWithoutAnInstallPath verifies 792a12l1c: a
// plugin in installed_plugins.json with no enabledPlugins entry and a missing installPath contributes no skill
// or command. The control, the same plugin with its installPath present, is enabled by default and offered, so
// the silence is owed to the missing path.
func Test792a12l1c_RegisterSkills_APluginWithoutAnEnabledPluginsEntryAndWithoutAnInstallPath(t *testing.T) {
	t.Parallel()

	root := srr3PluginRoot("m", "p", "1.0.0")

	cases := []struct {
		name        string
		installed   bool
		wantOffered []string
	}{
		{"the install path is missing", false, []string{}},
		{"control: the install path exists", true, []string{srr3Register("p:one", root+"/skills/one/SKILL.md")}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := skillsHomeFixture(nil)

			if testCase.installed {
				fsys.file(root+"/skills/one/SKILL.md", "one").file(root+"/commands/two.md", "two")
			}

			writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
				[]pluginInstall{{key: "p@m", installPath: root}}, map[string]bool{})

			stdout := srr3Preview(g, fsys)

			if testCase.installed {
				g.Expect(srr3Offers(stdout)).To(ConsistOf(
					srr3Register("p:one", root+"/skills/one/SKILL.md"),
					srr3Register("p:cmd:two", root+"/commands/two.md"),
				))

				return
			}

			g.Expect(srr3Offers(stdout)).To(BeEmpty())
		})
	}
}

// Test792a12l1d_RegisterSkills_OnlyTheInstalledVersionIsRead verifies 792a12l1d: with the plugin cache holding an
// older and a newer version of a plugin and installed_plugins.json naming the newer installPath, register-skills
// scans only that installPath's skills and commands and nothing from the older version.
func Test792a12l1d_RegisterSkills_OnlyTheInstalledVersionIsRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	oldRoot := srr3PluginRoot("m", "p", "1.0.0")
	newRoot := srr3PluginRoot("m", "p", "2.0.0")
	fsys := skillsHomeFixture(nil).
		file(oldRoot+"/skills/retired/SKILL.md", "retired").
		file(oldRoot+"/skills/current/SKILL.md", "an older current").
		file(oldRoot+"/commands/old.md", "old").
		file(newRoot+"/skills/current/SKILL.md", "current").
		file(newRoot+"/commands/now.md", "now")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "p@m", installPath: newRoot}}, map[string]bool{"p@m": true})

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(ConsistOf(
		srr3Register("p:current", newRoot+"/skills/current/SKILL.md"),
		srr3Register("p:cmd:now", newRoot+"/commands/now.md"),
	))
	g.Expect(stdout).NotTo(ContainSubstring(oldRoot))
	g.Expect(stdout).NotTo(ContainSubstring("conflict"))
}

// Test792a12m1a_RegisterSkills_AgentsSkillsInAnAncestor verifies 792a12m1a: a trusted project with
// .agents/skills/lint/SKILL.md at its top-level, registered from a subdirectory of it, makes register-skills
// offer lint as a project agents skill (key project:<r>:agents:lint).
func Test792a12m1a_RegisterSkills_AgentsSkillsInAnAncestor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/trust.json", `{"`+projectTop+`": true}`).
		file(projectTop+"/.agents/skills/lint/SKILL.md", "lint").
		dir(projectTop + "/internal/cli")

	stdout := srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop+"/internal/cli"))

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register(projectScope+":agents:lint", projectTop+"/.agents/skills/lint/SKILL.md"),
	}))
}

// Test792a12m1b_RegisterSkills_TrustFallsBackToDefaultProjectTrustAlways verifies 792a12m1b: with the working
// directory holding .pi/skills/foo/SKILL.md, a trust.json with no decision for it or any parent, and the global
// settings.json "defaultProjectTrust": "always", register-skills offers foo as a Pi project skill
// (key project:<r>:pi:foo).
func Test792a12m1b_RegisterSkills_TrustFallsBackToDefaultProjectTrustAlways(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := srr3PiHome().
		file(srr3PiAgent+"/trust.json", `{"/elsewhere": true}`).
		file(srr3PiAgent+"/settings.json", `{"defaultProjectTrust": "always"}`).
		file(projectTop+"/.pi/skills/foo/SKILL.md", "foo")

	stdout := srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop))

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register(projectScope+":pi:foo", projectTop+"/.pi/skills/foo/SKILL.md"),
	}))
}

// Test792a12m1c_RegisterSkills_AnUntrustedPiProjectIsNotScanned verifies 792a12m1c: with the working directory
// holding .pi/skills/foo/SKILL.md, no trusting decision in trust.json for it or any parent, and
// defaultProjectTrust not "always" (absent, "ask" or "never"), register-skills offers no Pi project skill. The
// Claude project skill beside it is still offered, so the project itself was scanned.
func Test792a12m1c_RegisterSkills_AnUntrustedPiProjectIsNotScanned(t *testing.T) {
	t.Parallel()

	for name, settings := range map[string]string{
		"no setting": `{}`,
		"ask":        `{"defaultProjectTrust": "ask"}`,
		"never":      `{"defaultProjectTrust": "never"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := srr3PiHome().
				file(srr3PiAgent+"/trust.json", `{"/elsewhere": true}`).
				file(srr3PiAgent+"/settings.json", settings).
				file(projectTop+"/.pi/skills/foo/SKILL.md", "foo").
				file(projectTop+"/.claude/skills/bar/SKILL.md", "bar")

			stdout := srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop))

			g.Expect(srr3Offers(stdout)).To(Equal([]string{
				srr3Register(projectScope+":bar", projectTop+"/.claude/skills/bar/SKILL.md"),
			}))
		})
	}
}

// Test792a12m1d_RegisterSkills_ClaudeProjectSourcesFromASubdirectory verifies 792a12m1d: a repository whose
// top-level has .claude/skills/openspec-propose/SKILL.md and .claude/commands/commit.md, registered from
// internal/, offers both (project:<r>:openspec-propose and project:<r>:cmd:commit), exactly as from the
// top-level.
func Test792a12m1d_RegisterSkills_ClaudeProjectSourcesFromASubdirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(nil).
		file(projectTop+"/.claude/skills/openspec-propose/SKILL.md", "propose").
		file(projectTop+"/.claude/commands/commit.md", "commit").
		dir(projectTop + "/internal")

	want := []string{
		srr3Register(projectScope+":cmd:commit", projectTop+"/.claude/commands/commit.md"),
		srr3Register(projectScope+":openspec-propose", projectTop+"/.claude/skills/openspec-propose/SKILL.md"),
	}

	fromSubdirectory := srr3Offers(srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop+"/internal")))
	fromTopLevel := srr3Offers(srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop)))

	g.Expect(fromSubdirectory).To(ConsistOf(want))
	g.Expect(fromTopLevel).To(Equal(fromSubdirectory), "exactly as from the top-level")
}

// Test792a12m1e_RegisterSkills_RootMarkdownInTheProjectSkillsDirectoryIsIgnored verifies 792a12m1e: a
// repository's .claude/skills/ holding root files commit.md and engram-go-conventions.md, with
// .claude/commands/commit.md present, makes register-skills offer neither root file as a skill and offer
// .claude/commands/commit.md as the command project:<r>:cmd:commit.
func Test792a12m1e_RegisterSkills_RootMarkdownInTheProjectSkillsDirectoryIsIgnored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(nil).
		file(projectTop+"/.claude/skills/commit.md", "a root file").
		file(projectTop+"/.claude/skills/engram-go-conventions.md", "another root file").
		file(projectTop+"/.claude/commands/commit.md", "commit")

	stdout := srr3Preview(g, fsys, srr3InRepo(projectTop, projectTop))

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register(projectScope+":cmd:commit", projectTop+"/.claude/commands/commit.md"),
	}))
}

// Test792a12m1f_RegisterSkills_OutsideARepositoryThereAreNoProjectEntries verifies 792a12m1f: a working directory
// outside any git repository with a .claude/skills directory makes register-skills offer no project skill. The
// control, the same directory inside a repository, offers it, so the silence is owed to the missing repository.
func Test792a12m1f_RegisterSkills_OutsideARepositoryThereAreNoProjectEntries(t *testing.T) {
	t.Parallel()

	const outside = "/outside/any/repo"

	cases := []struct {
		name      string
		customize []func(*cli.SkillRegistrationDeps)
		want      []string
	}{
		{"outside a repository", nil, []string{}},
		{"control: inside a repository", []func(*cli.SkillRegistrationDeps){srr3InRepo(outside, outside)}, []string{
			srr3Register(projectScope+":foo", outside+"/.claude/skills/foo/SKILL.md"),
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := skillsHomeFixture(nil).file(outside+"/.claude/skills/foo/SKILL.md", "foo")

			stdout := srr3Preview(g, fsys, testCase.customize...)

			g.Expect(srr3Offers(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12n1a_RegisterSkills_AMachineWithoutPiReadsNoPiSources verifies 792a12n1a: with ~/.claude present,
// ~/.pi absent and ~/.agents/skills/lint/SKILL.md present, register-skills scans no Pi or agents entry; the
// Claude user skill beside it is still offered.
func Test792a12n1a_RegisterSkills_AMachineWithoutPiReadsNoPiSources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).
		file(skillRegHome+"/.agents/skills/lint/SKILL.md", "lint")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("claude:route", skillRegSourcePath("route")),
	}))
}

// Test792a12n1b_RegisterSkills_ANameWithAColonIsSkipped verifies 792a12n1b: ~/.claude/skills/a:b/SKILL.md is
// skipped with a warning naming it and no offer is made for it; the skill beside it is still offered.
func Test792a12n1b_RegisterSkills_ANameWithAColonIsSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const colonSkill = skillRegHome + "/.claude/skills/a:b/SKILL.md"

	fsys := skillsHomeFixture(map[string][]byte{"route": srr1SkillText("route")}).file(colonSkill, "a:b")

	stdout := srr3Preview(g, fsys)

	g.Expect(srr3Offers(stdout)).To(Equal([]string{
		srr3Register("claude:route", skillRegSourcePath("route")),
	}))
	g.Expect(stdout).To(ContainSubstring(
		"engram: skipping skill \"a:b\" at "+colonSkill+": its name contains `:`"), "skipped with a warning")
}

// unexported constants.
const (
	srr3NpmModules = srr3PiAgent + "/npm/node_modules"
	srr3PiAgent    = skillRegHome + "/.pi/agent"
)

// srr3InRepo customizes a registration run to start in cwd inside the git repository whose top-level is top
// and whose origin is the engram repository on GitHub.
func srr3InRepo(top, cwd string) func(*cli.SkillRegistrationDeps) {
	return func(deps *cli.SkillRegistrationDeps) {
		deps.Sources.Commander = scriptedGit{
			"rev-parse --show-toplevel": {out: top + "\n"},
			"remote get-url origin":     {out: "git@github.com:toejough/engram.git\n"},
		}
		deps.Getwd = func() (string, error) { return cwd, nil }
	}
}

// srr3Nonnil returns lines, or an empty list for nil, so a comparison with srr3Offers' result is exact.
func srr3Nonnil(lines []string) []string {
	if lines == nil {
		return []string{}
	}

	return lines
}

// srr3Offers lists the offers a dry-run printed, each as "<kind> <key> (<source>)".
func srr3Offers(stdout string) []string {
	const offerPrefix = "  would offer: "

	offers := make([]string, 0)

	for line := range strings.SplitSeq(stdout, "\n") {
		if offer, found := strings.CutPrefix(line, offerPrefix); found {
			offers = append(offers, offer)
		}
	}

	return offers
}

// srr3PiHome is skillsHomeFixture(nil) with the Pi harness (~/.pi/agent) present as well.
func srr3PiHome() *fakeSkillFS {
	return skillsHomeFixture(nil).dir(srr3PiAgent)
}

// srr3PluginRoot is the plugin cache directory of version of plugin from market, under ~/.claude.
func srr3PluginRoot(market, plugin, version string) string {
	return skillRegHome + "/.claude/plugins/cache/" + market + "/" + plugin + "/" + version
}

// srr3Preview runs the dry-run registration over fsys (after customize has adjusted the deps), asserts it
// succeeds, and returns what it printed.
func srr3Preview(g Gomega, fsys *fakeSkillFS, customize ...func(*cli.SkillRegistrationDeps)) string {
	stdout, err := srr1Run(newSkillAcceptFixtureVault(), fsys, cli.SkillRegistrationArgs{DryRun: true}, customize...)
	g.Expect(err).NotTo(HaveOccurred())

	return stdout
}

// srr3Register is the dry-run offer to register key from source.
func srr3Register(key, source string) string {
	return "register " + key + " (" + source + ")"
}
