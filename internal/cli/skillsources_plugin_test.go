package cli_test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"sort"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

func TestScanClaudePlugins_CommandsFieldReplacesCommandsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := pluginCacheRoot + "/hud/hud/0.8.0"
	fsys := newFakeSkillFS().
		file(root+"/.claude-plugin/plugin.json", `{"name":"hud","commands":["./commands/setup.md"]}`).
		file(root+"/commands/setup.md", "setup").
		file(root+"/commands/configure.md", "configure")
	writePluginManifests(
		g,
		fsys,
		[]pluginInstall{{key: "hud@hud", installPath: root}},
		map[string]bool{"hud@hud": true},
	)

	result := cli.ScanClaudePlugins(fsys, pluginScanInput())

	g.Expect(pluginCandidateIDs(result)).To(Equal([]string{"plugin:hud command setup"}))
	g.Expect(pluginStatus(result, "hud")).To(Equal(pluginStatusScanned))
}

func TestScanClaudePlugins_ConflictProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		names := rapid.SliceOfNDistinct(rapid.StringMatching(`x[a-z]{1,4}`), 1, 5, rapid.ID[string]).
			Draw(rt, "names")
		markets := []string{"alpha", "beta", "gamma"}

		fsys := newFakeSkillFS()
		installs := make([]pluginInstall, 0)
		enabled := map[string]bool{}
		wantConflicts := map[string][]string{}

		for _, name := range names {
			chosen := rapid.SliceOfNDistinct(rapid.SampledFrom(markets), 1, len(markets), rapid.ID[string]).
				Draw(rt, "markets-"+name)

			for _, market := range chosen {
				root := fmt.Sprintf("%s/%s/%s/1.0.0", pluginCacheRoot, market, name)
				fsys.file(root+"/skills/s/SKILL.md", name+"@"+market)
				installs = append(
					installs,
					pluginInstall{key: name + "@" + market, installPath: root},
				)
				enabled[name+"@"+market] = true
			}

			if len(chosen) > 1 {
				sorted := slices.Clone(chosen)
				sort.Strings(sorted)
				wantConflicts[name] = sorted
			}
		}

		writePluginManifests(g, fsys, installs, enabled)

		result := cli.ScanClaudePlugins(fsys, pluginScanInput())

		gotConflicts := map[string][]string{}
		for _, conflict := range result.PluginConflicts {
			gotConflicts[conflict.Plugin] = conflict.Marketplaces
		}

		g.Expect(gotConflicts).To(Equal(wantConflicts))

		for _, name := range names {
			_, conflicted := wantConflicts[name]
			scope := cli.SkillScopePluginPrefix + name

			hasCandidates := slices.ContainsFunc(
				result.Candidates,
				func(candidate cli.SkillCandidate) bool {
					return candidate.ScopeID == scope
				},
			)
			g.Expect(hasCandidates).To(Equal(!conflicted), name)
			g.Expect(pluginStatus(result, name)).To(Equal(pluginStatusFor(!conflicted)), name)
		}
	})
}

func TestScanClaudePlugins_DisabledPluginReadsNothingBeyondManifests(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mock, imp := MockSkillSourceFS(t)
	done := make(chan cli.ClaudePluginScanResult)

	go func() {
		done <- cli.ScanClaudePlugins(mock, cli.ClaudePluginScan{ClaudeDir: "/c"})
	}()

	dirInfo := func(name string) fs.FileInfo {
		return fakeSkillInfo{name: name, node: &fakeSkillNode{kind: fakeSkillDir}}
	}
	fileInfo := func(name string) fs.FileInfo {
		return fakeSkillInfo{name: name, node: &fakeSkillNode{kind: fakeSkillFile}}
	}

	installed := `{"version":2,"plugins":{"hookify@m":[{"scope":"user","installPath":"/c/plugins/cache/m/hookify/1"}]}}`

	imp.Lstat.ArgsEqual("/c").Return(dirInfo("c"), nil)
	imp.Lstat.ArgsEqual("/c/plugins").Return(dirInfo("plugins"), nil)
	imp.Lstat.ArgsEqual("/c/plugins/installed_plugins.json").
		Return(fileInfo("installed_plugins.json"), nil)
	imp.ReadFile.ArgsEqual("/c/plugins/installed_plugins.json").Return([]byte(installed), nil)
	imp.Lstat.ArgsEqual("/c").Return(dirInfo("c"), nil)
	imp.Lstat.ArgsEqual("/c/settings.json").Return(fileInfo("settings.json"), nil)
	imp.ReadFile.ArgsEqual("/c/settings.json").
		Return([]byte(`{"enabledPlugins":{"hookify@m":false}}`), nil)

	result := <-done

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.ManifestsRead).To(BeTrue())
	g.Expect(pluginStatus(result, "hookify")).To(Equal(pluginStatusNotScanned))
}

func TestScanClaudePlugins_EnablementProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		setting := rapid.SampledFrom([]string{"true", "false", "absent"}).Draw(rt, "enabledPlugins")
		defaultEnabled := rapid.SampledFrom([]string{"true", "false", "absent", "no-manifest"}).
			Draw(rt, "defaultEnabled")

		root := pluginCacheRoot + "/m/p/1.0.0"
		fsys := newFakeSkillFS().file(root+"/skills/s/SKILL.md", "s")

		switch defaultEnabled {
		case "true", "false":
			fsys.file(
				root+"/.claude-plugin/plugin.json",
				`{"name":"p","defaultEnabled":`+defaultEnabled+`}`,
			)
		case "absent":
			fsys.file(root+"/.claude-plugin/plugin.json", `{"name":"p"}`)
		}

		enabled := map[string]bool{}
		if setting != "absent" {
			enabled["p@m"] = setting == "true"
		}

		writePluginManifests(g, fsys, []pluginInstall{{key: "p@m", installPath: root}}, enabled)

		result := cli.ScanClaudePlugins(fsys, pluginScanInput())

		want := setting == "true" || (setting == "absent" && defaultEnabled != "false")

		g.Expect(pluginStatus(result, "p")).To(Equal(pluginStatusFor(want)))
		g.Expect(result.Candidates).To(HaveLen(boolCount(want)))
	})
}

func TestScanClaudePlugins_MachineShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	superOld := pluginCacheRoot + "/" + pluginMarketOff + "/superpowers/6.3.0"
	superNew := pluginCacheRoot + "/" + pluginMarketOff + "/superpowers/6.4.1"
	hookify := pluginCacheRoot + "/" + pluginMarketOff + "/hookify/fa59bc903774"
	ralph := pluginCacheRoot + "/" + pluginMarketOff + "/ralph-loop/1.0.0"
	hud := pluginCacheRoot + "/claude-hud/claude-hud/0.8.0"
	commit := pluginCacheRoot + "/skills/commit/49ce826094da"
	engram := pluginCacheRoot + "/engram/engram/05306a342aa4"
	workOn := pluginCacheRoot + "/work-on/work-on/137fd9ab6aad"

	fsys := newFakeSkillFS().
		// A stale cached version holds a skill the installed version dropped.
		file(superOld+"/.claude-plugin/plugin.json", `{"name":"superpowers"}`).
		file(superOld+"/skills/brainstorming/SKILL.md", "old brainstorming").
		file(superOld+"/skills/retired/SKILL.md", "retired").
		file(superNew+"/.claude-plugin/plugin.json", `{"name":"superpowers"}`).
		file(superNew+"/skills/brainstorming/SKILL.md", "brainstorming").
		file(superNew+"/skills/writing-plans/SKILL.md", "writing-plans").
		file(superNew+"/commands/brainstorm.md", "brainstorm command").
		file(hookify+"/.claude-plugin/plugin.json", `{"name":"hookify"}`).
		file(hookify+"/skills/writing-rules/SKILL.md", "writing-rules").
		file(hookify+"/commands/hookify.md", "hookify").
		file(ralph+"/commands/help.md", "help").
		file(hud+"/.claude-plugin/plugin.json",
			`{"name":"claude-hud","commands":["./commands/setup.md","./commands/configure.md"]}`).
		file(hud+"/commands/setup.md", "setup").
		file(hud+"/commands/configure.md", "configure").
		file(commit+"/skills/commit/SKILL.md", "commit skill").
		file(commit+"/commands/commit.md", "commit command").
		file(engram+"/skills/learn/SKILL.md", "stale learn").
		dir(pluginCacheRoot + "/work-on/work-on")

	writePluginManifests(g, fsys, []pluginInstall{
		{key: "superpowers@" + pluginMarketOff, installPath: superNew},
		{key: "hookify@" + pluginMarketOff, installPath: hookify},
		{key: "ralph-loop@" + pluginMarketOff, installPath: ralph},
		{key: "claude-hud@claude-hud", installPath: hud},
		{key: "commit@skills", installPath: commit},
		{key: "engram@engram", installPath: engram},
		{key: "work-on@work-on", installPath: workOn},
	}, map[string]bool{
		"superpowers@" + pluginMarketOff: true,
		"hookify@" + pluginMarketOff:     false,
		"ralph-loop@" + pluginMarketOff:  false,
		"claude-hud@claude-hud":          true,
		"commit@skills":                  true,
		"engram@engram":                  false,
		// uninstalled: enabled in settings, absent from the manifest.
		"gone@" + pluginMarketOff: true,
	})

	result := cli.ScanClaudePlugins(fsys, pluginScanInput())

	g.Expect(pluginCandidateIDs(result)).To(Equal([]string{
		"plugin:claude-hud command configure",
		"plugin:claude-hud command setup",
		"plugin:commit command commit",
		"plugin:commit skill commit",
		"plugin:superpowers command brainstorm",
		"plugin:superpowers skill brainstorming",
		"plugin:superpowers skill writing-plans",
	}))

	for _, candidate := range result.Candidates {
		g.Expect(candidate.SourcePath).
			NotTo(HavePrefix(superOld), "never reads a stale cached version")
		g.Expect(candidate.Key).To(BeEmpty())
	}

	g.Expect(result.ManifestsRead).To(BeTrue())
	g.Expect(result.PluginConflicts).To(BeEmpty())
	g.Expect(result.Plugins).To(Equal([]cli.ClaudePluginStatus{
		{Name: "claude-hud", Scanned: true},
		{Name: "commit", Scanned: true},
		{Name: "engram", Scanned: false},
		{Name: "hookify", Scanned: false},
		{Name: "ralph-loop", Scanned: false},
		{Name: "superpowers", Scanned: true},
		{Name: "work-on", Scanned: false},
	}))
	g.Expect(pluginStatus(result, "gone")).To(Equal(pluginStatusAbsent))
	g.Expect(result.Warnings).To(BeEmpty())
}

func TestScanClaudePlugins_ManifestFailures(t *testing.T) {
	t.Parallel()

	type manifestCase struct {
		setup    func(*fakeSkillFS)
		warns    bool
		scansAny bool
	}

	good := `{"version":2,"plugins":{"p@m":[{"scope":"user","installPath":"/pp"}]}}`

	cases := map[string]manifestCase{
		"installed missing": {setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginSettingsRel, `{}`)
		}},
		"installed bad json": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, `{`).
				file(pluginClaudeDir+pluginSettingsRel, `{}`)
		}},
		"installed unreadable": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, good).
				file(pluginClaudeDir+pluginSettingsRel, `{}`).
				failRead(pluginClaudeDir + pluginInstalledRel)
		}},
		"installed wrong version": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, `{"version":1,"plugins":{}}`).
				file(pluginClaudeDir+pluginSettingsRel, `{}`)
		}},
		"settings bad json": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, good).
				file(pluginClaudeDir+pluginSettingsRel, `[`)
		}},
		"settings bad enabledPlugins": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, good).
				file(pluginClaudeDir+pluginSettingsRel, `{"enabledPlugins":[]}`)
		}},
		// A missing settings.json is not parsed (D5), but Claude Code still
		// loads default-enabled plugins, so they are still scanned.
		"settings missing": {scansAny: true, setup: func(f *fakeSkillFS) {
			f.file(pluginClaudeDir+pluginInstalledRel, good)
		}},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().file("/pp/skills/s/SKILL.md", "s")
			testCase.setup(fsys)

			result := cli.ScanClaudePlugins(fsys, pluginScanInput())

			g.Expect(result.ManifestsRead).To(BeFalse())
			g.Expect(result.Warnings != nil).To(Equal(testCase.warns), "%v", result.Warnings)
			g.Expect(result.Candidates != nil).To(Equal(testCase.scansAny))
		})
	}
}

func TestScanClaudePlugins_MostSpecificScopeWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	userRoot := pluginCacheRoot + "/m/p/1"
	localRoot := pluginCacheRoot + "/m/p/2"
	fsys := newFakeSkillFS().
		file(userRoot+"/skills/s/SKILL.md", "user").
		file(localRoot+"/skills/s/SKILL.md", "local")
	writePluginManifests(g, fsys, []pluginInstall{
		{key: "p@m", installPath: userRoot},
		{key: "p@m", installPath: localRoot, scope: "local", projectPath: pluginRepoTop},
	}, map[string]bool{"p@m": true})

	result := cli.ScanClaudePlugins(fsys, pluginScanInput())

	g.Expect(result.Candidates).To(HaveLen(1))

	if len(result.Candidates) != 1 {
		return
	}

	g.Expect(string(result.Candidates[0].Content)).To(Equal("local"))
}

func TestScanClaudePlugins_NonBooleanEnabledValueIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := pluginCacheRoot + "/m/p/1"
	fsys := newFakeSkillFS().file(root+"/skills/s/SKILL.md", "s")
	writePluginManifests(g, fsys, []pluginInstall{{key: "p@m", installPath: root}}, map[string]bool{})
	fsys.file(pluginClaudeDir+pluginSettingsRel, `{"enabledPlugins":{"p@m":"yes"}}`)

	result := cli.ScanClaudePlugins(fsys, pluginScanInput())

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(pluginStatus(result, "p")).To(Equal(pluginStatusNotScanned))
	g.Expect(warningsMention(result.SkillScanResult, "p@m")).To(BeTrue())
}

func TestScanClaudePlugins_PluginJSONSkillsAddToDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := pluginCacheRoot + "/m/p/1.0.0"
	fsys := newFakeSkillFS().
		file(root+"/.claude-plugin/plugin.json", `{"name":"p","skills":["./extra/","./single","./skills"]}`).
		file(root+"/skills/base/SKILL.md", "base").
		file(root+"/extra/more/SKILL.md", "more").
		file(root+"/single/SKILL.md", "single")
	writePluginManifests(g, fsys, []pluginInstall{{key: "p@m", installPath: root}}, map[string]bool{})

	result := cli.ScanClaudePlugins(fsys, pluginScanInput())

	g.Expect(pluginCandidateIDs(result)).To(Equal([]string{
		"plugin:p skill base",
		"plugin:p skill more",
		"plugin:p skill single",
	}))
	g.Expect(pluginStatus(result, "p")).To(Equal(pluginStatusScanned))
}

func TestScanClaudePlugins_ReservedAndColonNamesAreSkipped(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"pi", "agents", "project", "anthropic-skills", "cmd", "pi-settings", "pi-pkg", "pi-prompt", "a:b",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			root := pluginCacheRoot + "/m/x/1"
			fsys := newFakeSkillFS().file(root+"/skills/s/SKILL.md", "s")
			writePluginManifests(g, fsys, []pluginInstall{{key: name + "@m", installPath: root}},
				map[string]bool{name + "@m": true})

			result := cli.ScanClaudePlugins(fsys, pluginScanInput())

			g.Expect(result.Candidates).To(BeEmpty())
			g.Expect(pluginStatus(result, name)).To(Equal(pluginStatusNotScanned))
			g.Expect(warningsMention(result.SkillScanResult, name)).To(BeTrue())
		})
	}
}

func TestScanClaudePlugins_ScopeProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		scope := rapid.SampledFrom([]string{"user", "project", "local", "managed"}).
			Draw(rt, "scope")
		projectPath := rapid.SampledFrom([]string{"", pluginRepoTop, "/home/repos/other"}).
			Draw(rt, "projectPath")
		repoTop := rapid.SampledFrom([]string{"", pluginRepoTop}).Draw(rt, "repoTop")

		root := pluginCacheRoot + "/m/p/1"
		fsys := newFakeSkillFS().file(root+"/skills/s/SKILL.md", "s")
		writePluginManifests(
			g,
			fsys,
			[]pluginInstall{
				{key: "p@m", installPath: root, scope: scope, projectPath: projectPath},
			},
			map[string]bool{"p@m": true},
		)

		result := cli.ScanClaudePlugins(
			fsys,
			cli.ClaudePluginScan{ClaudeDir: pluginClaudeDir, RepoTopLevel: repoTop},
		)

		want := scope == "user" ||
			((scope == "project" || scope == "local") && repoTop != "" && projectPath == repoTop)

		g.Expect(pluginStatus(result, "p")).To(Equal(pluginStatusFor(want)))
		g.Expect(result.Candidates).To(HaveLen(boolCount(want)))
	})
}

func TestScanClaudePlugins_UnscannablePluginContents(t *testing.T) {
	t.Parallel()

	root := pluginCacheRoot + "/m/p/1"

	cases := map[string]struct {
		setup func(*fakeSkillFS)
		warns bool
	}{
		"unreadable skill child": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").file(root+"/skills/bad/SKILL.md", "bad").
				failRead(root + "/skills/bad/SKILL.md")
		}},
		"unreadable install path": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").failRead(root)
		}},
		"bad plugin.json": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").file(root+"/.claude-plugin/plugin.json", `{`)
		}},
		"object-map commands": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").
				file(root+"/.claude-plugin/plugin.json", `{"commands":{"x":{"source":"./x.md"}}}`)
		}},
		"bad skills field": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").
				file(root+"/.claude-plugin/plugin.json", `{"skills":{"a":1}}`)
		}},
		"declared skills path missing": {setup: func(f *fakeSkillFS) {
			f.file(root+"/skills/ok/SKILL.md", "ok").
				file(root+"/.claude-plugin/plugin.json", `{"skills":["./nowhere"]}`)
		}},
		"plugin root holds SKILL.md": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/SKILL.md", "root").
				file(root+"/.claude-plugin/plugin.json", `{"skills":["."]}`)
		}},
		"unreadable declared skill": {warns: true, setup: func(f *fakeSkillFS) {
			f.file(root+"/one/SKILL.md", "one").failRead(root+"/one/SKILL.md").
				file(root+"/.claude-plugin/plugin.json", `{"skills":"./one"}`)
		}},
		"missing install path": {setup: func(*fakeSkillFS) {}},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS()
			testCase.setup(fsys)
			writePluginManifests(
				g,
				fsys,
				[]pluginInstall{{key: "p@m", installPath: root}},
				map[string]bool{"p@m": true},
			)

			result := cli.ScanClaudePlugins(fsys, pluginScanInput())

			g.Expect(pluginStatus(result, "p")).To(Equal(pluginStatusNotScanned))
			g.Expect(result.Warnings != nil).To(Equal(testCase.warns), "%v", result.Warnings)
		})
	}
}

// unexported constants.
const (
	pluginCacheRoot    = pluginClaudeDir + "/plugins/cache"
	pluginClaudeDir    = "/home/.claude"
	pluginInstalledRel = "/plugins/installed_plugins.json"
	// pluginManifestVersion is installed_plugins.json's current format.
	pluginManifestVersion  = 2
	pluginMarketOff        = "claude-plugins-official"
	pluginRepoTop          = "/home/repos/engram"
	pluginSettingsRel      = "/settings.json"
	pluginStatusAbsent     = "absent"
	pluginStatusNotScanned = "not scanned"
	pluginStatusScanned    = "scanned"
)

type pluginInstall struct {
	key         string
	installPath string
	scope       string
	projectPath string
}

func boolCount(value bool) int {
	if value {
		return 1
	}

	return 0
}

// pluginCandidateIDs renders each candidate as "<scope> <kind> <name>",
// sorted.
func pluginCandidateIDs(result cli.ClaudePluginScanResult) []string {
	ids := make([]string, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		ids = append(
			ids,
			fmt.Sprintf("%s %s %s", candidate.ScopeID, candidate.Kind, candidate.Name),
		)
	}

	sort.Strings(ids)

	return ids
}

func pluginScanInput() cli.ClaudePluginScan {
	return cli.ClaudePluginScan{ClaudeDir: pluginClaudeDir, RepoTopLevel: pluginRepoTop}
}

func pluginStatus(result cli.ClaudePluginScanResult, name string) string {
	for _, status := range result.Plugins {
		if status.Name == name {
			return pluginStatusFor(status.Scanned)
		}
	}

	return pluginStatusAbsent
}

func pluginStatusFor(scanned bool) string {
	if scanned {
		return pluginStatusScanned
	}

	return pluginStatusNotScanned
}

// writePluginManifests writes installed_plugins.json (v2) and settings.json
// with the given enabledPlugins map under pluginClaudeDir.
func writePluginManifests(g Gomega, fsys *fakeSkillFS, installs []pluginInstall, enabled map[string]bool) {
	writePluginManifestsAt(g, fsys, pluginClaudeDir, installs, enabled)
}

// writePluginManifestsAt is writePluginManifests under claudeDir.
func writePluginManifestsAt(
	g Gomega, fsys *fakeSkillFS, claudeDir string, installs []pluginInstall, enabled map[string]bool,
) {
	type entry struct {
		Scope       string `json:"scope"`
		InstallPath string `json:"installPath"`
		ProjectPath string `json:"projectPath,omitempty"`
		Version     string `json:"version"`
	}

	plugins := map[string][]entry{}

	for _, install := range installs {
		scope := install.scope
		if scope == "" {
			scope = "user"
		}

		plugins[install.key] = append(plugins[install.key], entry{
			Scope:       scope,
			InstallPath: install.installPath,
			ProjectPath: install.projectPath,
			Version:     "x",
		})
	}

	installed, installedErr := json.Marshal(struct {
		Version int                `json:"version"`
		Plugins map[string][]entry `json:"plugins"`
	}{Version: pluginManifestVersion, Plugins: plugins})
	settings, settingsErr := json.Marshal(struct {
		EnabledPlugins map[string]bool `json:"enabledPlugins"`
		Model          string          `json:"model"`
	}{EnabledPlugins: enabled, Model: "opus"})

	g.Expect(installedErr).NotTo(HaveOccurred())
	g.Expect(settingsErr).NotTo(HaveOccurred())
	fsys.file(claudeDir+pluginInstalledRel, string(installed))
	fsys.file(claudeDir+pluginSettingsRel, string(settings))
}
