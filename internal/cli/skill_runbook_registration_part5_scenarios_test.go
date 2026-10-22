package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a12q1a_RegisterSkills_AProjectSkillNoteFromAnotherDirectory verifies 792a12q1a: a note keyed
// project:github.com/toejough/engram:openspec-propose, with register-skills run from a directory outside that
// repository (not in any repository, or in another one), draws no removal offer. The control run from inside the
// repository, whose .claude/skills is read and lacks the skill, is offered the removal.
func Test792a12q1a_RegisterSkills_AProjectSkillNoteFromAnotherDirectory(t *testing.T) {
	t.Parallel()

	const (
		otherTop = "/work/elsewhere"
		skill    = "openspec-propose"
		key      = srr5Project + ":" + skill
		source   = projectTop + "/.claude/skills/" + skill + "/SKILL.md"
	)

	newFS := func() *fakeSkillFS {
		return skillsHomeFixture(nil).dir(projectTop + "/.claude/skills").dir(otherTop + "/.claude/skills")
	}
	otherRepo := srr4InRepo(otherTop, srr4Git(otherTop, "git@github.com:someone/else.git"))
	ownRepo := srr4InRepo(projectTop, srr4Git(projectTop, "git@github.com:toejough/engram.git"))

	cases := []struct {
		name      string
		customize []func(*cli.SkillRegistrationDeps)
		want      []string
	}{
		{"outside any repository", nil, []string{}},
		{"inside another repository", []func(*cli.SkillRegistrationDeps){otherRepo}, []string{}},
		{"control: inside the repository", []func(*cli.SkillRegistrationDeps){ownRepo},
			[]string{srr5Remove(key, source)}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := srr5Vault(srr5Note{key: key, source: source})

			stdout, err := srr1Run(vault, newFS(), cli.SkillRegistrationArgs{DryRun: true}, testCase.customize...)
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr3Offers(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12q1b_UpdateSkillRegistration_AnUpdateRunWithoutAProject verifies 792a12q1b: engram update's
// registration step, run from a directory that is not in a git repository while project skill notes exist,
// offers no removal of any project skill note (nothing is outstanding and every note is still there). The
// control run from inside the repository leaves the removal outstanding.
func Test792a12q1b_UpdateSkillRegistration_AnUpdateRunWithoutAProject(t *testing.T) {
	t.Parallel()

	const (
		deployedKey   = srr5Project + ":deploy"
		deployedPath  = projectTop + "/.claude/skills/deploy/SKILL.md"
		commandKey    = srr5Project + ":cmd:ship"
		commandSource = projectTop + "/.claude/commands/ship.md"
	)

	notes := []srr5Note{{key: deployedKey, source: deployedPath}, {key: commandKey, source: commandSource}}
	newFS := func() *fakeSkillFS {
		return skillsHomeFixture(nil).dir(projectTop + "/.claude/skills").dir(projectTop + "/.claude/commands")
	}

	t.Run("outside any repository", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := srr5Vault(notes...)
		before := len(vault.files)

		stdout, err := srr5Update(vault, newFS())
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(BeEmpty(), "no removal offer is outstanding for any project skill note")
		g.Expect(vault.files).To(HaveLen(before), "no project skill note is touched")
	})

	t.Run("control: inside the repository", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := srr5Vault(notes...)
		inRepo := srr4InRepo(projectTop, srr4Git(projectTop, "git@github.com:toejough/engram.git"))

		stdout, err := srr5Update(vault, newFS(), inRepo)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(ContainSubstring("2 skill runbook offers awaiting an answer: @" + srr5Project + " 2"))
	})
}

// Test792a12q1c_RegisterSkills_ADisabledPluginKeepsItsNotes verifies 792a12q1c: a note keyed
// hookify:writing-rules with hookify installed but disabled draws no removal offer. The control run, with hookify
// enabled and its install path read without the skill, is offered the removal.
func Test792a12q1c_RegisterSkills_ADisabledPluginKeepsItsNotes(t *testing.T) {
	t.Parallel()

	const (
		key    = "hookify:writing-rules"
		source = "~/.claude/plugins/cache/claude-plugins-official/hookify/1.0.0/skills/writing-rules/SKILL.md"
	)

	for name, enabled := range map[string]bool{"disabled": false, "control: enabled": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			root := srr3PluginRoot("claude-plugins-official", "hookify", "1.0.0")
			fsys := skillsHomeFixture(nil).file(root+"/skills/other/SKILL.md", "another hookify skill")
			writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
				[]pluginInstall{{key: "hookify@claude-plugins-official", installPath: root}},
				map[string]bool{"hookify@claude-plugins-official": enabled})

			stdout, err := srr1Run(srr5Vault(srr5Note{key: key, source: source}), fsys,
				cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			removals := srr5Removals(stdout)

			if enabled {
				g.Expect(removals).To(Equal([]string{srr5Remove(key, source)}))

				return
			}

			g.Expect(removals).To(BeEmpty())
			g.Expect(stdout).NotTo(ContainSubstring("would offer: remove"))
		})
	}
}

// Test792a12q1d_RegisterSkills_APiDisabledSkillKeepsItsNote verifies 792a12q1d: a note keyed pi:ping, with
// ~/.pi/agent/skills read and the global Pi settings.json skills holding "!ping", draws no register, refresh or
// removal offer, whether the note is stale (a refresh would be due) or absent (a register would be due). The
// control runs, with the setting off, are offered the refresh and the register.
func Test792a12q1d_RegisterSkills_APiDisabledSkillKeepsItsNote(t *testing.T) {
	t.Parallel()

	const (
		key    = "pi:ping"
		path   = srr3PiAgent + "/skills/ping/SKILL.md"
		source = "~/.pi/agent/skills/ping/SKILL.md"
	)

	cases := []struct {
		name     string
		settings string
		notes    []srr5Note
		want     []string
	}{
		{"disabled, stale note", `{"skills": ["!ping"]}`, []srr5Note{{key: key, hash: "stale", source: source}}, []string{}},
		{"disabled, no note", `{"skills": ["!ping"]}`, nil, []string{}},
		{"control: enabled, stale note", `{"skills": []}`, []srr5Note{{key: key, hash: "stale", source: source}},
			[]string{"refresh " + key + " (" + path + ")"}},
		{"control: enabled, no note", `{"skills": []}`, nil, []string{srr3Register(key, path)}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := srr5PiSkillsHome().
				file(path, string(srr1SkillText("ping"))).
				file(srr3PiAgent+"/settings.json", testCase.settings)

			stdout, err := srr1Run(srr5Vault(testCase.notes...), fsys, cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr3Offers(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12q1e_RegisterSkills_AnUninstalledPluginIsOfferedForRemoval verifies 792a12q1e: a note keyed
// ralph-loop:cmd:help with ralph-loop absent from a readable installed_plugins.json (another plugin is listed)
// is offered for removal.
func Test792a12q1e_RegisterSkills_AnUninstalledPluginIsOfferedForRemoval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		key    = "ralph-loop:cmd:help"
		source = "~/.claude/plugins/cache/claude-plugins-official/ralph-loop/1.0.0/commands/help.md"
	)

	root := srr3PluginRoot("claude-plugins-official", "hookify", "1.0.0")
	fsys := skillsHomeFixture(nil).file(root+"/skills/other/SKILL.md", "another hookify skill")
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude",
		[]pluginInstall{{key: "hookify@claude-plugins-official", installPath: root}},
		map[string]bool{"hookify@claude-plugins-official": true})

	stdout, err := srr1Run(srr5Vault(srr5Note{key: key, source: source}), fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(ContainSubstring("@plugin:ralph-loop (1)\n"))
	g.Expect(srr5Removals(stdout)).To(Equal([]string{srr5Remove(key, source)}))
}

// Test792a12q1f_RegisterSkills_AnUnreadableUserSkillsDirectory verifies 792a12q1f: with listing ~/.claude/skills
// failing and claude: notes existing, register-skills offers no removal for them. The control run, with the
// directory listed and lacking the skills, is offered both removals.
func Test792a12q1f_RegisterSkills_AnUnreadableUserSkillsDirectory(t *testing.T) {
	t.Parallel()

	notes := []srr5Note{
		{key: "claude:c4", source: "~/.claude/skills/c4/SKILL.md"},
		{key: "claude:dev", source: "~/.claude/skills/dev/SKILL.md"},
	}

	t.Run("listing fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := skillsHomeFixture(nil).failRead(skillRegHome + "/.claude/skills")

		stdout, err := srr1Run(srr5Vault(notes...), fsys, cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).NotTo(ContainSubstring("would offer: remove"))
		g.Expect(stdout).To(ContainSubstring("engram: "), "the unreadable directory is reported, not treated as empty")
	})

	t.Run("control: listing succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		stdout, err := srr1Run(srr5Vault(notes...), skillsHomeFixture(nil), cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(srr3Offers(stdout)).To(Equal([]string{
			srr5Remove("claude:c4", notes[0].source), srr5Remove("claude:dev", notes[1].source),
		}))
	})
}

// Test792a12q1g_RegisterSkills_AnotherNamespacesMissingRootDoesNotBlockRemoval verifies 792a12q1g: with
// ~/.agents/skills missing, ~/.pi/agent/skills failing with a permission error, ~/.claude/skills read and lacking
// c4, and a note keyed claude:c4, the removal of the claude:c4 note is offered.
func Test792a12q1g_RegisterSkills_AnotherNamespacesMissingRootDoesNotBlockRemoval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const source = "~/.claude/skills/c4/SKILL.md"

	fsys := srr5PiSkillsHome().failRead(srr3PiAgent + "/skills")

	stdout, err := srr1Run(srr5Vault(srr5Note{key: "claude:c4", source: source}), fsys,
		cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr5Remove("claude:c4", source)}))
}

// Test792a12q1h_RegisterSkills_AnUnkeyedLegacyNoteIsNeverOfferedForRemoval verifies 792a12q1h: the runbook note
// 1036.2026-09-18.skill-route.md carrying skill_hash and no skill_key draws no removal offer while every source
// root was read. The control, the same note with the key claude:route, is offered for removal on the same
// sources, so the missing key is what keeps the legacy note.
func Test792a12q1h_RegisterSkills_AnUnkeyedLegacyNoteIsNeverOfferedForRemoval(t *testing.T) {
	t.Parallel()

	const legacyName = "1036.2026-09-18.skill-route.md"

	legacy := "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_hash: \"h\"\n---\n\nbody\n"

	t.Run("unkeyed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillAcceptFixtureVault()
		vault.put(legacyName, legacy)

		stdout, err := srr1Run(vault, srr5EveryRootRead(g), cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(BeEmpty())
	})

	t.Run("control: keyed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newSkillAcceptFixtureVault()
		keyed := strings.Replace(legacy, "skill_hash: \"h\"\n", "skill_hash: \"h\"\nskill_key: \"claude:route\"\n", 1)
		vault.put(legacyName, keyed)

		stdout, err := srr1Run(vault, srr5EveryRootRead(g), cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(srr3Offers(stdout)).To(Equal([]string{srr5Remove("claude:route", "note 1036.2026-09-18.skill-route")}))
	})
}

// Test792a12q1i_RegisterSkills_AMissingSyncedManifest verifies 792a12q1i: with ~/.claude/skills/synced/ holding
// no bucket with a readable manifest.json (one bucket has none, one has an unparseable one) and
// anthropic-skills: notes existing, register-skills offers no removal for them. The control, with a parsing
// manifest, is offered the removal.
func Test792a12q1i_RegisterSkills_AMissingSyncedManifest(t *testing.T) {
	t.Parallel()

	const (
		synced = skillRegHome + "/.claude/skills/synced"
		bucket = synced + "/4227b111_e76eb279"
		source = "~/.claude/skills/synced/4227b111_e76eb279/pdf/SKILL.md"
	)

	notes := []srr5Note{
		{key: "anthropic-skills:pdf", source: source},
		{key: "anthropic-skills:docx", source: "~/.claude/skills/synced/9c0de5a4_17b2f3d8/docx/SKILL.md"},
	}

	t.Run("no readable manifest", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := skillsHomeFixture(nil).dir(bucket).
			file(synced+"/9c0de5a4_17b2f3d8/manifest.json", "{not json")

		stdout, err := srr1Run(srr5Vault(notes...), fsys, cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).NotTo(ContainSubstring("would offer: remove"))
	})

	t.Run("control: a manifest parses", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := skillsHomeFixture(nil).file(bucket+"/manifest.json", `{"skills":[]}`)

		stdout, err := srr1Run(srr5Vault(notes[:1]...), fsys, cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(srr3Offers(stdout)).To(Equal([]string{srr5Remove("anthropic-skills:pdf", source)}))
	})
}

// Test792a12q1j_RegisterSkills_ANestedOnlyProjectSkillIsSafeFromTheTopLevel verifies 792a12q1j: a note keyed
// project:github.com/toejough/engram:foo registered from sub/ has its skill_source under sub/.claude/skills, and
// register-skills run from the repository top-level offers no removal for it. The control run from sub/, whose
// .claude/skills is read, is offered the removal.
func Test792a12q1j_RegisterSkills_ANestedOnlyProjectSkillIsSafeFromTheTopLevel(t *testing.T) {
	t.Parallel()

	const (
		key    = srr5Project + ":foo"
		source = projectTop + "/sub/.claude/skills/foo/SKILL.md"
	)

	git := srr4Git(projectTop, "git@github.com:toejough/engram.git")
	cases := []struct {
		name string
		cwd  string
		want []string
	}{
		{"from the top-level", projectTop, []string{}},
		{"control: from sub", projectTop + "/sub", []string{srr5Remove(key, source)}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := skillsHomeFixture(nil).
				dir(projectTop + "/.claude/skills").
				dir(projectTop + "/sub/.claude/skills")

			stdout, err := srr1Run(srr5Vault(srr5Note{key: key, source: source}), fsys,
				cli.SkillRegistrationArgs{DryRun: true}, srr4InRepo(testCase.cwd, git))
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr3Offers(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12q1k_RegisterSkills_OneFailingSyncedBucket verifies 792a12q1k: with two synced buckets, bucket A's
// manifest.json parsing and bucket B's not, the note whose skill_source lies in bucket B draws no removal offer
// while the note in bucket A, absent from its bucket, is offered for removal.
func Test792a12q1k_RegisterSkills_OneFailingSyncedBucket(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		synced  = skillRegHome + "/.claude/skills/synced"
		sourceA = "~/.claude/skills/synced/4227b111_e76eb279/xlsx/SKILL.md"
		sourceB = "~/.claude/skills/synced/9c0de5a4_17b2f3d8/docx/SKILL.md"
	)

	fsys := skillsHomeFixture(nil).
		file(synced+"/4227b111_e76eb279/manifest.json", `{"skills":[]}`).
		file(synced+"/9c0de5a4_17b2f3d8/manifest.json", "{not json")
	vault := srr5Vault(
		srr5Note{key: "anthropic-skills:docx", source: sourceB},
		srr5Note{key: "anthropic-skills:xlsx", source: sourceA},
	)

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr3Offers(stdout)).To(Equal([]string{srr5Remove("anthropic-skills:xlsx", sourceA)}))
}

// Test792a12q1l_RegisterSkills_APiSettingsNoteIsTiedToItsOwnEntry verifies 792a12q1l: with settings.json holding
// two skills entries, the first unreadable, a pi-settings: note whose skill_source lies under the first draws no
// removal offer, while the note under the readable second entry, absent from it, is offered for removal.
func Test792a12q1l_RegisterSkills_APiSettingsNoteIsTiedToItsOwnEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		first   = "/extra/first"
		second  = "/extra/second"
		sourceA = first + "/alpha/SKILL.md"
		sourceB = second + "/beta/SKILL.md"
	)

	fsys := srr5PiSkillsHome().
		file(first+"/other/SKILL.md", "another skill").
		file(second+"/other/SKILL.md", "another skill").
		file(srr3PiAgent+"/settings.json", `{"skills": ["`+first+`", "`+second+`"]}`).
		failRead(first)
	vault := srr5Vault(
		srr5Note{key: "pi-settings:alpha", source: sourceA},
		srr5Note{key: "pi-settings:beta", source: sourceB},
	)

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr3Offers(stdout)).To(ContainElement(srr3Register("pi-settings:other", second+"/other/SKILL.md")),
		"the second entry was read")
	g.Expect(srr5Removals(stdout)).To(Equal([]string{srr5Remove("pi-settings:beta", sourceB)}))
}

// Test792a12q1m_RegisterSkills_APackageDroppedFromSettingsLeavesAnOrphan verifies 792a12q1m: a pi-pkg:x: note
// whose npm:x package is no longer listed in settings.json draws no removal offer. The control, with npm:x still
// listed and its package root read without the skill, is offered the removal.
func Test792a12q1m_RegisterSkills_APackageDroppedFromSettingsLeavesAnOrphan(t *testing.T) {
	t.Parallel()

	const (
		key     = "pi-pkg:x:foo"
		root    = srr3NpmModules + "/x"
		source  = "~/.pi/agent/npm/node_modules/x/skills/foo/SKILL.md"
		otherPk = srr3NpmModules + "/y"
	)

	cases := []struct {
		name     string
		packages string
		want     []string
	}{
		{"dropped", `"npm:y"`, []string{}},
		{"control: still listed", `"npm:y", "npm:x"`, []string{srr5Remove(key, source)}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := srr5PiSkillsHome().
				file(root+"/skills/other/SKILL.md", "another skill of x").
				file(otherPk+"/skills/kept/SKILL.md", "a skill of y").
				file(srr3PiAgent+"/settings.json", `{"packages": [`+testCase.packages+`]}`)

			stdout, err := srr1Run(srr5Vault(srr5Note{key: key, source: source}), fsys,
				cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr5Removals(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12q1n_RegisterSkills_AnAliasKeepsItsNote verifies 792a12q1n: a note keyed pi:ping with the Pi ping
// skill now byte-identical to a higher-precedence skill (claude:ping) draws no removal offer, and nothing else is
// offered. The control, with the Pi ping skill deleted, is offered the removal.
func Test792a12q1n_RegisterSkills_AnAliasKeepsItsNote(t *testing.T) {
	t.Parallel()

	const source = "~/.pi/agent/skills/ping/SKILL.md"

	ping := srr1SkillText("ping")
	note := srr5Note{key: "pi:ping", hash: srr1Hash(ping), source: source}

	t.Run("alias", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := srr5PiSkillsHome().
			file(skillRegHome+"/.claude/skills/ping/SKILL.md", string(ping)).
			file(srr3PiAgent+"/skills/ping/SKILL.md", string(ping))

		stdout, err := srr1Run(srr5Vault(note), fsys, cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(stdout).To(BeEmpty())
	})

	t.Run("control: the Pi skill is gone", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := srr5PiSkillsHome().file(skillRegHome+"/.claude/skills/ping/SKILL.md", string(ping))

		stdout, err := srr1Run(srr5Vault(note), fsys, cli.SkillRegistrationArgs{DryRun: true})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(srr5Removals(stdout)).To(Equal([]string{srr5Remove("pi:ping", source)}))
	})
}

// Test792a12q1o_RegisterSkills_AMachineWithoutTheHarness verifies 792a12q1o: with ~/.pi missing and a note keyed
// pi:ping, register-skills offers no removal for it. The control, with ~/.pi/agent/skills present, is offered
// the removal.
func Test792a12q1o_RegisterSkills_AMachineWithoutTheHarness(t *testing.T) {
	t.Parallel()

	const source = "~/.pi/agent/skills/ping/SKILL.md"

	cases := []struct {
		name string
		fsys *fakeSkillFS
		want []string
	}{
		{"no Pi harness", skillsHomeFixture(nil), []string{}},
		{"control: the Pi harness", srr5PiSkillsHome(), []string{srr5Remove("pi:ping", source)}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			stdout, err := srr1Run(srr5Vault(srr5Note{key: "pi:ping", source: source}), testCase.fsys,
				cli.SkillRegistrationArgs{DryRun: true})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(srr5Removals(stdout)).To(Equal(testCase.want))
		})
	}
}

// Test792a12r1a_RegisterSkills_DeclineAWholePlugin verifies 792a12r1a: with the 15 superpowers register offers
// outstanding, register-skills --decline @plugin:superpowers records all 15 current hashes as declined under
// their keys and writes no note.
func Test792a12r1a_RegisterSkills_DeclineAWholePlugin(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skills := srr5Superpowers()
	vault := newSkillAcceptFixtureVault()

	_, err := srr1Run(vault, srr1PluginHome(g, "6.5.0", skills),
		cli.SkillRegistrationArgs{Decline: []string{"@plugin:superpowers"}})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(skills).To(HaveLen(srr5SuperpowersCount))
	g.Expect(srr5Declined(g, vault)).To(Equal(srr5Hashes(skills)))
	g.Expect(srr5Notes(vault)).To(BeEmpty(), "no note is written")
}

// Test792a12r1b_RegisterSkills_TheLongestPatternWins verifies 792a12r1b: --decline 'superpowers:*' with
// --accept 'superpowers:writing-*' registers superpowers:writing-plans and superpowers:writing-skills and
// declines every other superpowers offer. Swapping the flags swaps the outcome, so the longer pattern wins rather
// than the accept or the decline flag.
func Test792a12r1b_RegisterSkills_TheLongestPatternWins(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		args            cli.SkillRegistrationArgs
		wantRegistered  string
		wantDeclineRest bool
	}{
		{"decline the plugin, accept writing-*", cli.SkillRegistrationArgs{
			Decline: []string{"superpowers:*"}, Accept: []string{"superpowers:writing-*"},
		}, "writing-", true},
		{"accept the plugin, decline writing-*", cli.SkillRegistrationArgs{
			Accept: []string{"superpowers:*"}, Decline: []string{"superpowers:writing-*"},
		}, "writing-", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			skills := srr5Superpowers()
			vault := newSkillAcceptFixtureVault()

			_, err := srr1Run(vault, srr1PluginHome(g, "6.5.0", skills), testCase.args)
			g.Expect(err).NotTo(HaveOccurred())

			writing, others := srr5Split(skills, testCase.wantRegistered)

			if testCase.wantDeclineRest {
				g.Expect(srr5Notes(vault)).To(ConsistOf(
					HaveSuffix(".skill-superpowers-writing-plans.md"), HaveSuffix(".skill-superpowers-writing-skills.md")))
				g.Expect(srr5Declined(g, vault)).To(Equal(srr5Hashes(others)))

				return
			}

			g.Expect(srr5Notes(vault)).To(HaveLen(len(others)))
			g.Expect(srr5Declined(g, vault)).To(Equal(srr5Hashes(writing)))
		})
	}
}

// Test792a12r1c_RegisterSkills_AnExactKeyBeatsAPattern verifies 792a12r1c: --decline 'superpowers:*' with
// --accept superpowers:brainstorming registers superpowers:brainstorming and declines the other superpowers
// offers. Swapping the flags swaps the outcome, so the exact key wins over the pattern whichever flag holds it.
func Test792a12r1c_RegisterSkills_AnExactKeyBeatsAPattern(t *testing.T) {
	t.Parallel()

	const exact = "brainstorming"

	cases := []struct {
		name               string
		args               cli.SkillRegistrationArgs
		wantExactRegistred bool
	}{
		{"decline the plugin, accept the key", cli.SkillRegistrationArgs{
			Decline: []string{"superpowers:*"}, Accept: []string{"superpowers:" + exact},
		}, true},
		{"accept the plugin, decline the key", cli.SkillRegistrationArgs{
			Accept: []string{"superpowers:*"}, Decline: []string{"superpowers:" + exact},
		}, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			skills := srr5Superpowers()
			vault := newSkillAcceptFixtureVault()

			_, err := srr1Run(vault, srr1PluginHome(g, "6.5.0", skills), testCase.args)
			g.Expect(err).NotTo(HaveOccurred())

			only, rest := srr5Split(skills, exact)

			if testCase.wantExactRegistred {
				g.Expect(srr5Notes(vault)).To(ConsistOf(HaveSuffix(".skill-superpowers-brainstorming.md")))
				g.Expect(srr5Declined(g, vault)).To(Equal(srr5Hashes(rest)))

				return
			}

			g.Expect(srr5Notes(vault)).To(HaveLen(len(rest)))
			g.Expect(srr5Declined(g, vault)).To(Equal(srr5Hashes(only)))
		})
	}
}

// Test792a12r1d_RegisterSkills_TheClaudeUserScopeIsNameable verifies 792a12r1d: with register offers for
// claude:c4 and claude:dev, --decline @claude-user records both as declined (and registers nothing).
func Test792a12r1d_RegisterSkills_TheClaudeUserScopeIsNameable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c4Text, devText := srr1SkillText("c4"), srr1SkillText("dev")
	vault := newSkillAcceptFixtureVault()

	_, err := srr1Run(vault, skillsHomeFixture(map[string][]byte{"c4": c4Text, "dev": devText}),
		cli.SkillRegistrationArgs{Decline: []string{"@claude-user"}})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr5Declined(g, vault)).To(Equal(map[string]string{
		"claude:c4": srr1Hash(c4Text), "claude:dev": srr1Hash(devText),
	}))
	g.Expect(srr5Notes(vault)).To(BeEmpty())
}

// Test792a12r1e_RegisterSkills_RemovalsAreNeverBulkAccepted verifies 792a12r1e: with one register offer
// (claude:c4) and one removal offer (claude:dev) outstanding, --accept '*' carries out the register offer and
// only reports the removal, whose note stays and which stays outstanding. Accepting its exact key then removes
// it.
func Test792a12r1e_RegisterSkills_RemovalsAreNeverBulkAccepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := srr5Vault(srr5Note{key: "claude:dev", hash: "h", source: "~/.claude/skills/dev/SKILL.md"})
	devNote := srr5Notes(vault)

	g.Expect(devNote).To(HaveLen(1))

	fsys := skillsHomeFixture(map[string][]byte{"c4": srr1SkillText("c4")})

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{Accept: []string{"*"}})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr5Notes(vault)).To(ConsistOf(devNote[0], HaveSuffix(".skill-claude-c4.md")),
		"c4 is registered and the dev note is not removed")
	g.Expect(stdout).To(ContainSubstring("1 skill runbook offer awaiting an answer: @claude-user 1"),
		"the removal is only reported")

	_, err = srr1Run(vault, fsys, cli.SkillRegistrationArgs{Accept: []string{"claude:dev"}})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(srr5Notes(vault)).To(ConsistOf(HaveSuffix(".skill-claude-c4.md")), "its exact key removes it")
}

// Test792a12r1f_RegisterSkills_SkipLeavesTheScopeForNextTime verifies 792a12r1f: answering skip for a scope at the
// interactive prompt writes and records nothing for that scope's offers, and the next run offers them again.
func Test792a12r1f_RegisterSkills_SkipLeavesTheScopeForNextTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(map[string][]byte{"c4": srr1SkillText("c4"), "dev": srr1SkillText("dev")})
	vault := newSkillAcceptFixtureVault()

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{}, srr5Terminal("s\n"))
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout).To(ContainSubstring(
		"@claude-user (2): [a]ccept all / [d]ecline all / [r]eview each / [s]kip for now"))
	g.Expect(vault.files).To(BeEmpty(), "nothing is written or recorded")

	again, againErr := srr1Run(vault, fsys, cli.SkillRegistrationArgs{DryRun: true})
	g.Expect(againErr).NotTo(HaveOccurred())

	g.Expect(srr3Offers(again)).To(Equal([]string{
		srr3Register("claude:c4", skillRegSourcePath("c4")), srr3Register("claude:dev", skillRegSourcePath("dev")),
	}))
}

// Test792a12r1g_RegisterSkills_AScopeWithSeveralOffersIsAskedOnce verifies 792a12r1g: on a terminal, a scope with
// three register offers has the three listed and asked about once, and answering accept all registers them all
// without a per-skill question.
func Test792a12r1g_RegisterSkills_AScopeWithSeveralOffersIsAskedOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := skillsHomeFixture(map[string][]byte{
		"c4": srr1SkillText("c4"), "dev": srr1SkillText("dev"), "route": srr1SkillText("route"),
	})
	vault := newSkillAcceptFixtureVault()

	// The answer is accept all; the further lines would decline the offers if each were asked about on its own.
	answers := "a\n" + strings.Repeat("n\n", srr5ExtraAnswers)

	stdout, err := srr1Run(vault, fsys, cli.SkillRegistrationArgs{}, srr5Terminal(answers))
	g.Expect(err).NotTo(HaveOccurred())

	for _, name := range []string{"c4", "dev", "route"} {
		g.Expect(stdout).To(ContainSubstring("  register claude:" + name + " (" + skillRegSourcePath(name) + ")\n"))
	}

	g.Expect(strings.Count(stdout, "[a]ccept all")).To(Equal(1), "the scope is asked about once")
	g.Expect(stdout).NotTo(ContainSubstring("Register skill"), "no per-skill question")
	g.Expect(srr5Notes(vault)).To(ConsistOf(
		HaveSuffix(".skill-claude-c4.md"), HaveSuffix(".skill-claude-dev.md"), HaveSuffix(".skill-claude-route.md")))
}

// Test792a12r1h_RegisterSkills_EndOfInputAtAPromptRecordsNothing verifies 792a12r1h: on a terminal whose input
// ends at the first prompt, no offer is accepted or declined, whether that prompt is a scope's grouped question
// or a single offer's y/N question.
func Test792a12r1h_RegisterSkills_EndOfInputAtAPromptRecordsNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		skills map[string][]byte
		prompt string
	}{
		{"grouped question", map[string][]byte{"c4": srr1SkillText("c4"), "dev": srr1SkillText("dev")}, "[a]ccept all"},
		{"single offer", map[string][]byte{"c4": srr1SkillText("c4")}, "[y/N]"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()

			stdout, err := srr1Run(vault, skillsHomeFixture(testCase.skills), cli.SkillRegistrationArgs{},
				srr5Terminal(""))
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(stdout).To(ContainSubstring(testCase.prompt), "the prompt was asked")
			g.Expect(vault.files).To(BeEmpty(), "no offer is accepted or declined")
		})
	}
}

// unexported constants.
const (
	srr5DeclinesFile     = "skill-registrations.json"
	srr5ExtraAnswers     = 3
	srr5NoteIDBase       = 1100
	srr5Project          = "project:github.com/toejough/engram"
	srr5SuperpowersCount = 15
)

// srr5Note is a skill note to seed in a fixture vault: its key, skill_hash and recorded skill_source.
type srr5Note struct {
	key, hash, source string
}

// srr5UpdateEnv is the environment of an update run: no variable set, and the registration fixtures' home.
type srr5UpdateEnv struct{}

func (srr5UpdateEnv) Getenv(string) string { return "" }

func (srr5UpdateEnv) Getwd() (string, error) { return "/outside/any/repo", nil }

func (srr5UpdateEnv) UserHomeDir() (string, error) { return skillRegHome, nil }

// srr5Declined reads the decline state of vault: skill key to declined hash.
func srr5Declined(g Gomega, vault *skillAcceptFixtureVault) map[string]string {
	content, found := vault.get(srr5DeclinesFile)
	if !found {
		return map[string]string{}
	}

	var state struct {
		Declined map[string]string `json:"declined"`
	}

	g.Expect(json.Unmarshal([]byte(content), &state)).To(Succeed())

	return state.Declined
}

// srr5EveryRootRead is a home where every source root of the default set was read: the Claude user skills,
// commands and a parsing synced bucket, the Pi and agents roots, and plugin manifests listing no plugin.
func srr5EveryRootRead(g Gomega) *fakeSkillFS {
	fsys := srr5PiSkillsHome().
		dir(skillRegHome+"/.agents/skills").
		dir(skillRegHome+"/.claude/commands").
		dir(srr3PiAgent+"/prompts").
		file(skillRegHome+"/.claude/skills/synced/4227b111_e76eb279/manifest.json", `{"skills":[]}`)
	writePluginManifestsAt(g, fsys, skillRegHome+"/.claude", nil, map[string]bool{})

	return fsys
}

// srr5Hashes maps the key superpowers:<name> of each skill to the skill's hash.
func srr5Hashes(skills map[string]string) map[string]string {
	hashes := make(map[string]string, len(skills))

	for name, text := range skills {
		hashes["superpowers:"+name] = srr1Hash([]byte(text))
	}

	return hashes
}

// srr5Notes lists the basenames of the skill notes in vault.
func srr5Notes(vault *skillAcceptFixtureVault) []string {
	names, _ := vault.ListMD(srr1Vault)

	return names
}

// srr5PiSkillsHome is skillsHomeFixture(nil) with the Pi harness and its (empty) ~/.pi/agent/skills present.
func srr5PiSkillsHome() *fakeSkillFS {
	return srr3PiHome().dir(srr3PiAgent + "/skills")
}

// srr5Removals lists the removal offers a dry-run printed.
func srr5Removals(stdout string) []string {
	removals := make([]string, 0)

	for _, offer := range srr3Offers(stdout) {
		if strings.HasPrefix(offer, "remove ") {
			removals = append(removals, offer)
		}
	}

	return removals
}

// srr5Remove is the dry-run offer to remove key's note, recorded as from source.
func srr5Remove(key, source string) string {
	return "remove " + key + " (" + source + ")"
}

// srr5Split splits the skills into those whose name starts with prefix and the others.
func srr5Split(skills map[string]string, prefix string) (matching, others map[string]string) {
	matching, others = map[string]string{}, map[string]string{}

	for name, text := range skills {
		if strings.HasPrefix(name, prefix) {
			matching[name] = text
		} else {
			others[name] = text
		}
	}

	return matching, others
}

// srr5Superpowers is the 15 skills of the superpowers plugin, name to SKILL.md text.
func srr5Superpowers() map[string]string {
	names := []string{
		"brainstorming", "diagnosing-superpowers", "dispatching-parallel-agents", "executing-plans",
		"finishing-a-development-branch", "receiving-code-review", "requesting-code-review",
		"subagent-driven-development", "systematic-debugging", "test-driven-development", "using-git-worktrees",
		"using-superpowers", "verification-before-completion", "writing-plans", "writing-skills",
	}
	skills := make(map[string]string, len(names))

	for _, name := range names {
		skills[name] = string(srr1SkillText(name))
	}

	return skills
}

// srr5Terminal customizes a registration run to read its answers from a terminal holding input.
func srr5Terminal(input string) func(*cli.SkillRegistrationDeps) {
	return func(deps *cli.SkillRegistrationDeps) {
		deps.IsTerminal = func() bool { return true }
		deps.Stdin = strings.NewReader(input)
	}
}

// srr5Update runs engram update's registration hook (a real run, not a dry one) over vault and fsys, after
// customize has adjusted the composed deps, and returns what it printed with its error.
func srr5Update(
	vault *skillAcceptFixtureVault, fsys *fakeSkillFS, customize ...func(*cli.SkillRegistrationDeps),
) (string, error) {
	deps := skillRegistrationDepsFor(vault, fsys)

	for _, adjust := range customize {
		adjust(&deps)
	}

	var stdout bytes.Buffer

	err := cli.ExportRunUpdateSkillRegistration(
		context.Background(), false, srr1Vault, skillRegHome, deps, srr5UpdateEnv{}, &stdout)

	return stdout.String(), err
}

// srr5Vault is a fixture vault holding a skill note for each of notes.
func srr5Vault(notes ...srr5Note) *skillAcceptFixtureVault {
	vault := newSkillAcceptFixtureVault()

	for index, note := range notes {
		hash := note.hash
		if hash == "" {
			hash = "h"
		}

		name := fmt.Sprintf("%d.2026-09-26.%s.md", srr5NoteIDBase+index, cli.SkillKeySlug(note.key))
		vault.put(name, sourcedSkillNote(offerNote{key: note.key, hash: hash, source: note.source}))
	}

	return vault
}
