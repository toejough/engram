package cli_test

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// ---------------------------------------------------------------------------
// ApplyPiSettingsOverrides (ruling R8)
// ---------------------------------------------------------------------------

func TestApplyPiSettingsOverrides_DisablesDefaultFolderSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piUserRoot+"/ping/SKILL.md", "ping").
		file(piUserRoot+"/pong/SKILL.md", "pong").
		file(piUserRoot+"/keep/SKILL.md", "keep")
	scanned := cli.ScanPiUserSkills(fsys, piUserRoot)

	// `!pattern` excludes by name; `+path` re-enables exactly; `-path`
	// force-excludes exactly; a plain include never filters default folders.
	got, warnings := cli.ApplyPiSettingsOverrides(scanned.Candidates,
		[]string{"!p*", "+skills/pong", "-skills/keep", "zzz"}, piAgentDir)

	g.Expect(warnings).To(BeEmpty())
	g.Expect(disabledNames(got)).To(Equal([]string{"keep", "ping"}))
	g.Expect(candidateNames(cliSkillScanResult{Candidates: got})).To(Equal([]string{"keep", "ping", "pong"}))
	// The input is not mutated.
	g.Expect(disabledNames(scanned.Candidates)).To(BeEmpty())
}

// TestApplyPiSettingsOverrides_EnabledProperty checks Pi's
// isEnabledByOverrides (package-manager.js:515-539) for name excludes and
// exact absolute +/- paths: enabled = !(excluded) , then +forced → true,
// then -forced → false.
func TestApplyPiSettingsOverrides_EnabledProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		names := rapid.SliceOfNDistinct(rapid.StringMatching(`[a-z][a-z0-9]{0,4}`), 1, 8, rapid.ID[string]).
			Draw(rt, "names")
		fsys := newFakeSkillFS().dir(piUserRoot)
		patterns := []string{}
		want := []string{}

		for index, name := range names {
			fsys.file(piUserRoot+"/"+name+"/SKILL.md", name)

			excluded := rapid.Bool().Draw(rt, fmt.Sprintf("ex%d", index))
			forced := rapid.Bool().Draw(rt, fmt.Sprintf("plus%d", index))
			forceExcluded := rapid.Bool().Draw(rt, fmt.Sprintf("minus%d", index))

			if excluded {
				patterns = append(patterns, "!"+name)
			}

			if forced {
				patterns = append(patterns, "+"+piUserRoot+"/"+name)
			}

			if forceExcluded {
				patterns = append(patterns, "-"+piUserRoot+"/"+name)
			}

			if forceExcluded || (excluded && !forced) {
				want = append(want, name)
			}
		}

		sort.Strings(want)

		got, warnings := cli.ApplyPiSettingsOverrides(cli.ScanPiUserSkills(fsys, piUserRoot).Candidates, patterns, piAgentDir)

		g.Expect(warnings).To(BeEmpty())
		g.Expect(disabledNames(got)).To(Equal(want))
	})
}

func TestApplyPiSettingsOverrides_FallsBackToSourcePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	candidates := []cli.SkillCandidate{{Name: "x", SourcePath: piUserRoot + "/x/SKILL.md"}}

	got, _ := cli.ApplyPiSettingsOverrides(candidates, []string{"!x"}, piAgentDir)

	g.Expect(disabledNames(got)).To(Equal([]string{"x"}))
}

func TestApplyPiSettingsOverrides_GlobstarDisablesAllWithWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(piUserRoot+"/ping/SKILL.md", "ping")

	got, warnings := cli.ApplyPiSettingsOverrides(cli.ScanPiUserSkills(fsys, piUserRoot).Candidates,
		[]string{"!**/ping"}, piAgentDir)

	g.Expect(warnings).To(HaveLen(1))
	g.Expect(disabledNames(got)).To(Equal([]string{"ping"}))
}

// TestApplyPiSettingsOverrides_MatchesWalkedPathNotResolved: Pi matches the
// path as discovered, so a symlinked skill is matched by its link location.
func TestApplyPiSettingsOverrides_MatchesWalkedPathNotResolved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/engram/skills/route/SKILL.md", "route").
		link(piUserRoot+"/route", piAgentDir+"/engram/skills/route")

	scanned := cli.ScanPiUserSkills(fsys, piUserRoot)
	g.Expect(scanned.Candidates).To(HaveLen(1))
	g.Expect(scanned.Candidates[0].WalkedPath).To(Equal(piUserRoot + "/route/SKILL.md"))

	got, _ := cli.ApplyPiSettingsOverrides(scanned.Candidates, []string{"-skills/route"}, piAgentDir)

	g.Expect(disabledNames(got)).To(Equal([]string{"route"}))
}

func TestApplyPiSettingsOverrides_PromptsByFileName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(piPromptsRoot+"/review.md", "r").file(piPromptsRoot+"/fix.md", "f")

	got, _ := cli.ApplyPiSettingsOverrides(cli.ScanPiUserPrompts(fsys, piPromptsRoot).Candidates,
		[]string{"!review.md"}, piAgentDir)

	g.Expect(disabledNames(got)).To(Equal([]string{"review"}))
}

func TestReadPiSettings_LegacySkillsObjectAndErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file("/a/settings.json", `{"skills": {"customDirectories": ["~/x"]}}`).
		file("/b/settings.json", `{"skills": [`).
		file("/c/settings.json", `{}`).failRead("/c/settings.json")

	legacy, warnings := cli.ReadPiSettings(fsys, "/a/settings.json")
	g.Expect(warnings).To(BeEmpty())
	g.Expect(legacy.Skills).To(Equal([]string{"~/x"}))

	missing, warnings := cli.ReadPiSettings(fsys, "/none/settings.json")
	g.Expect(warnings).To(BeEmpty())
	g.Expect(missing.Skills).To(BeEmpty())

	_, warnings = cli.ReadPiSettings(fsys, "/b/settings.json")
	g.Expect(warnings).To(HaveLen(1))

	fsys.file("/d/settings.json", `{"skills": [1], "prompts": "p", "packages": [{"skills": []}]}`)
	shapes, warnings := cli.ReadPiSettings(fsys, "/d/settings.json")
	g.Expect(warnings).To(HaveLen(3))
	g.Expect(shapes.Skills).To(BeEmpty())

	_, warnings = cli.ReadPiSettings(fsys, "/c/settings.json")
	g.Expect(warnings).To(HaveLen(1))
}

// ---------------------------------------------------------------------------
// ReadPiSettings
// ---------------------------------------------------------------------------

// TestReadPiSettings_ReadsResolvedFile drives the FS interactively: the
// settings path is resolved (Lstat) and then read.
func TestReadPiSettings_ReadsResolvedFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mock, imp := MockSkillSourceFS(t)
	done := make(chan cli.PiSettings)

	go func() {
		settings, _ := cli.ReadPiSettings(mock, "/s.json")
		done <- settings
	}()

	imp.Lstat.ArgsEqual("/s.json").Return(fakeSkillInfo{name: "s.json", node: &fakeSkillNode{kind: fakeSkillFile}}, nil)
	imp.ReadFile.ArgsEqual("/s.json").
		Return([]byte(`{"skills": ["a", "!b"], "prompts": ["p"], "theme": "x"}`), nil)

	settings := <-done
	g.Expect(settings.Skills).To(Equal([]string{"a", "!b"}))
	g.Expect(settings.Prompts).To(Equal([]string{"p"}))
}

// ---------------------------------------------------------------------------
// ScanPiConfiguredSources: packages (task 1.5)
// ---------------------------------------------------------------------------

func TestScanPiConfiguredSources_AutoloadFalseIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": [{"source": "npm:x", "autoload": false}]}`).
		file(piNpmModules+"/x/skills/fmt/SKILL.md", "fmt")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(warningsMention(result, "autoload")).To(BeTrue())

	scanned, found := rootScanned(result, piNpmModules+"/x")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
}

// ---------------------------------------------------------------------------
// ScanPiConfiguredSources: settings `skills` / `prompts` (task 1.5)
// ---------------------------------------------------------------------------

func TestScanPiConfiguredSources_DirectoryEntryIncludesRootMarkdown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Ruling R10 Q1 (package-manager.js:241): a settings directory entry
	// loads its own root `.md` files, and relative entries resolve against
	// ~/.pi/agent.
	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"skills": ["mine"]}`).
		file(piAgentDir+"/mine/notes.md", "notes").
		file(piAgentDir+"/mine/deep/a/SKILL.md", "a").
		file(piAgentDir+"/mine/deep/a/extra.md", "extra")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"a", "notes"}))

	scanned, _ := rootScanned(result, piAgentDir+"/mine")
	g.Expect(scanned).To(BeTrue())
}

func TestScanPiConfiguredSources_FileEntriesAreOneSkillEach(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"skills": ["~/x/tool.md", "~/y/lint/SKILL.md", "~/z/readme.txt"]}`).
		file(fakeHome+"/x/tool.md", "tool").
		file(fakeHome+"/y/lint/SKILL.md", "lint").
		file(fakeHome+"/z/readme.txt", "no")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"lint", "tool"}))

	scanned, _ := rootScanned(result, fakeHome+"/x/tool.md")
	g.Expect(scanned).To(BeTrue())
}

func TestScanPiConfiguredSources_GitAndLocalPackageRoots(t *testing.T) {
	t.Parallel()

	cases := []struct{ source, root, pkgID string }{
		{"git:github.com/u/repo@v1", piAgentDir + "/git/github.com/u/repo", "github.com/u/repo"},
		{"https://gitlab.com/a/b.git", piAgentDir + "/git/gitlab.com/a/b", "gitlab.com/a/b"},
		{"git:git@github.com:u/r2", piAgentDir + "/git/github.com/u/r2", "github.com/u/r2"},
		{"ssh://git@example.org/o/p", piAgentDir + "/git/example.org/o/p", "example.org/o/p"},
		{"../../pkgs/local", fakeHome + "/pkgs/local", "~/pkgs/local"},
		{"~/pkgs/tilde", fakeHome + "/pkgs/tilde", "~/pkgs/tilde"},
		{"/opt/abs", "/opt/abs", "/opt/abs"},
	}

	for _, testCase := range cases {
		t.Run(testCase.source, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().
				file(piAgentDir+"/settings.json", `{"packages": ["`+testCase.source+`"]}`).
				file(testCase.root+"/skills/fmt/SKILL.md", "fmt")

			result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

			g.Expect(result.Candidates).To(HaveLen(1))

			if len(result.Candidates) != 1 {
				return
			}

			g.Expect(result.Candidates[0].ScopeID).To(Equal("pi-pkg:" + testCase.pkgID))
			g.Expect(result.Candidates[0].SourceSegment).To(Equal("pi-pkg:" + testCase.pkgID))
			g.Expect(result.Candidates[0].ReadRoot).To(Equal(testCase.root))

			scanned, _ := rootScanned(result, testCase.root)
			g.Expect(scanned).To(BeTrue())
		})
	}
}

func TestScanPiConfiguredSources_GlobstarPatternIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"skills": ["~/extra-skills", "!**/fmt"]}`).
		file(fakeHome+"/extra-skills/fmt/SKILL.md", "fmt")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(warningsMention(result, "**")).To(BeTrue())

	scanned, found := rootScanned(result, fakeHome+"/extra-skills")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
}

// TestScanPiConfiguredSources_MachineShape: this machine's Pi settings list
// one local package that does not exist, and pi-intercom is installed under
// ~/.pi/agent/npm but not configured. Nothing is found, the missing package
// is recorded as not scanned without a warning, and pi-intercom is never
// read.
func TestScanPiConfiguredSources_MachineShape(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	inner := newFakeSkillFS().
		file(piAgentDir+"/settings.json",
			`{"theme": "light/dark", "packages": ["../../repos/personal/pi-skills/bare-skills"]}`).
		file(piNpmModules+"/pi-intercom/package.json", `{"pi": {"skills": ["./skills"]}}`).
		file(piNpmModules+"/pi-intercom/skills/pi-intercom/SKILL.md", "intercom")
	recorder := &recordingSkillFS{inner: inner}

	result := cli.ScanPiConfiguredSources(recorder, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Warnings).To(BeEmpty())

	scanned, found := rootScanned(result, fakeHome+"/repos/personal/pi-skills/bare-skills")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())

	for _, call := range recorder.calls {
		g.Expect(call).NotTo(ContainSubstring("pi-intercom"))
	}
}

func TestScanPiConfiguredSources_ManifestGlobsAndOverrides(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Manifest entries: a glob expands per segment, `!` drops (not even
	// Disabled — Pi's addManifestEntries adds only enabled files).
	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:x"]}`).
		file(piNpmModules+"/x/package.json", `{"pi": {"skills": ["./sk/*", "!old"], "prompts": ["./pr"]}}`).
		file(piNpmModules+"/x/sk/fmt/SKILL.md", "fmt").
		file(piNpmModules+"/x/sk/old/SKILL.md", "old").
		file(piNpmModules+"/x/sk/.dot/SKILL.md", "dot").
		file(piNpmModules+"/x/pr/deep/review.md", "review")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"fmt", "review"}))
	g.Expect(disabledNames(result.Candidates)).To(BeEmpty())
}

func TestScanPiConfiguredSources_ManifestGlobstarIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:x"]}`).
		file(piNpmModules+"/x/package.json", `{"pi": {"skills": ["./**/SKILL.md"]}}`).
		file(piNpmModules+"/x/sk/fmt/SKILL.md", "fmt")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(warningsMention(result, "**")).To(BeTrue())

	scanned, _ := rootScanned(result, piNpmModules+"/x")
	g.Expect(scanned).To(BeFalse())
}

func TestScanPiConfiguredSources_MissingEntryIsNotScannedSilently(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file(piAgentDir+"/settings.json", `{"skills": ["~/gone"]}`)

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Warnings).To(BeEmpty())

	scanned, found := rootScanned(result, fakeHome+"/gone")
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeFalse())
}

func TestScanPiConfiguredSources_NoPiKeyUsesConventionDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:@scope/conv"]}`).
		file(piNpmModules+"/@scope/conv/package.json", `{"name": "@scope/conv"}`).
		file(piNpmModules+"/@scope/conv/skills/a/b/SKILL.md", "b").
		file(piNpmModules+"/@scope/conv/skills/top.md", "top").
		file(piNpmModules+"/@scope/conv/prompts/x/y.md", "y")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"b", "top", "y"}))

	for _, candidate := range result.Candidates {
		g.Expect(candidate.ScopeID).To(Equal("pi-pkg:@scope/conv"))
	}
}

// TestScanPiConfiguredSources_NpmManifestSkills covers the spec scenario
// "npm package skills from its manifest".
func TestScanPiConfiguredSources_NpmManifestSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:pi-intercom@0.6.0"]}`).
		file(piNpmModules+"/pi-intercom/package.json",
			`{"name": "pi-intercom", "pi": {"extensions": ["./index.ts"], "skills": ["./skills"]}}`).
		file(piNpmModules+"/pi-intercom/skills/pi-intercom/SKILL.md", "intercom").
		file(piNpmModules+"/pi-intercom/skills/root.md", "root")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	// A listed skill dir loads its root `.md` too (ruling R10 Q1).
	g.Expect(candidateNames(result)).To(Equal([]string{"pi-intercom", "root"}))

	for _, candidate := range result.Candidates {
		g.Expect(candidate.ScopeID).To(Equal("pi-pkg:pi-intercom"))
		g.Expect(candidate.ReadRoot).To(Equal(piNpmModules + "/pi-intercom"))
	}

	scanned, _ := rootScanned(result, piNpmModules+"/pi-intercom")
	g.Expect(scanned).To(BeTrue())
}

// TestScanPiConfiguredSources_ObjectFormFilters covers the spec scenarios
// "String-form package with a pi key ignores convention dirs", "Object-form
// omitted type falls back per type" and "Object-form empty skills filter".
func TestScanPiConfiguredSources_ObjectFormFilters(t *testing.T) {
	t.Parallel()

	piKeyOnlyExtensions := `{"pi": {"extensions": ["./index.ts"]}}`
	intercomManifest := `{"pi": {"skills": ["./skills"]}}`

	cases := []struct {
		name, entry, pkg, manifest string
		skills                     []string
		want, disabled             []string
	}{
		{"string form with pi key", `"npm:x"`, "x", piKeyOnlyExtensions, []string{"fmt"}, []string{}, []string{}},
		{
			"object form omitted skills", `{"source": "npm:x", "extensions": []}`, "x", piKeyOnlyExtensions,
			[]string{"fmt"}, []string{"fmt"}, []string{},
		},
		{
			"object form empty skills", `{"source": "npm:pi-intercom", "skills": []}`, "pi-intercom", intercomManifest,
			[]string{"pi-intercom"}, []string{"pi-intercom"}, []string{"pi-intercom"},
		},
		{
			"object form pattern filter", `{"source": "npm:x", "skills": ["!b"]}`, "x", `{}`,
			[]string{"a", "b"}, []string{"a", "b"}, []string{"b"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().
				file(piAgentDir+"/settings.json", `{"packages": [`+testCase.entry+`]}`).
				file(piNpmModules+"/"+testCase.pkg+"/package.json", testCase.manifest)

			for _, skill := range testCase.skills {
				fsys.file(piNpmModules+"/"+testCase.pkg+"/skills/"+skill+"/SKILL.md", skill)
			}

			result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

			g.Expect(candidateNames(result)).To(Equal(testCase.want))
			g.Expect(disabledNames(result.Candidates)).To(Equal(testCase.disabled))

			scanned, _ := rootScanned(result, piNpmModules+"/"+testCase.pkg)
			g.Expect(scanned).To(BeTrue())
		})
	}
}

// TestScanPiConfiguredSources_PatternFilterProperty checks Pi's
// applyPatterns (package-manager.js:540-584) over one settings directory:
// plain globs include by name (all when none), `!` excludes by name, `+`
// re-adds by exact path, `-` removes by exact path. Filtered-out skills are
// still emitted, marked Disabled (ruling R10 Q3).
func TestScanPiConfiguredSources_PatternFilterProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		names := rapid.SliceOfNDistinct(rapid.StringMatching(`[a-z][a-z0-9]{0,4}`), 1, 8, rapid.ID[string]).
			Draw(rt, "names")
		patterns := []string{"~/extra-skills"}
		fsys := newFakeSkillFS()

		var includes []string

		for index, name := range names {
			fsys.file(fakeHome+"/extra-skills/"+name+"/SKILL.md", name)

			included := rapid.Bool().Draw(rt, fmt.Sprintf("in%d", index))
			excluded := rapid.Bool().Draw(rt, fmt.Sprintf("ex%d", index))
			forced := rapid.Bool().Draw(rt, fmt.Sprintf("plus%d", index))
			forceExcluded := rapid.Bool().Draw(rt, fmt.Sprintf("minus%d", index))

			if included {
				includes = append(includes, name)
				patterns = append(patterns, name+"*")
			}

			if excluded {
				patterns = append(patterns, "!"+name)
			}

			if forced {
				patterns = append(patterns, "+"+fakeHome+"/extra-skills/"+name)
			}

			if forceExcluded {
				patterns = append(patterns, "-"+fakeHome+"/extra-skills/"+name)
			}
		}

		want := piFilterOracle(names, includes, patterns)

		sort.Strings(want)

		settings := fmt.Sprintf(`{"skills": [%s]}`, quoteJoin(patterns))
		fsys.file(piAgentDir+"/settings.json", settings)

		result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

		g.Expect(result.Candidates).To(HaveLen(len(names)))
		g.Expect(disabledNames(result.Candidates)).To(Equal(want))
	})
}

// TestScanPiConfiguredSources_ProjectPackageOverrideProperty: for any global
// and project npm package lists, exactly one root per identity is scanned,
// and the project's wins.
func TestScanPiConfiguredSources_ProjectPackageOverrideProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		pool := []string{"a", "b", "c", "d"}
		global := rapid.SliceOfDistinct(rapid.SampledFrom(pool), rapid.ID[string]).Draw(rt, "global")
		project := rapid.SliceOfDistinct(rapid.SampledFrom(pool), rapid.ID[string]).Draw(rt, "project")

		fsys := newFakeSkillFS().
			file(piAgentDir+"/settings.json", `{"packages": [`+quoteJoin(prefixAll("npm:", global))+`]}`).
			file(projectTop+"/.pi/settings.json", `{"packages": [`+quoteJoin(prefixAll("npm:", project))+`]}`)

		want := []string{}

		for _, name := range pool {
			fsys.file(piNpmModules+"/"+name+"/skills/"+name+"/SKILL.md", name)
			fsys.file(projectTop+"/.pi/npm/node_modules/"+name+"/skills/"+name+"/SKILL.md", name)

			switch {
			case slices.Contains(project, name):
				want = append(want, projectTop+"/.pi/npm/node_modules/"+name)
			case slices.Contains(global, name):
				want = append(want, piNpmModules+"/"+name)
			}
		}

		result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(true))

		got := make([]string, 0, len(result.Roots))
		for _, root := range result.Roots {
			got = append(got, root.Path)
		}

		sort.Strings(got)
		sort.Strings(want)
		g.Expect(got).To(Equal(want))
	})
}

func TestScanPiConfiguredSources_ProjectPackageOverridesGlobal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:x@1", "npm:y"]}`).
		file(piNpmModules+"/x/skills/global/SKILL.md", "g").
		file(piNpmModules+"/y/skills/why/SKILL.md", "y").
		file(projectTop+"/.pi/settings.json", `{"packages": ["npm:x@2"]}`).
		file(projectTop+"/.pi/npm/node_modules/x/skills/local/SKILL.md", "l")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(true))

	g.Expect(candidateNames(result)).To(Equal([]string{"local", "why"}))

	for _, candidate := range result.Candidates {
		switch candidate.Name {
		case "local":
			g.Expect(candidate.ScopeID).To(Equal(projectScope))
			g.Expect(candidate.SourceSegment).To(Equal("pi-pkg:x"))
		case "why":
			g.Expect(candidate.ScopeID).To(Equal("pi-pkg:y"))
		}
	}

	_, globalRecorded := rootScanned(result, piNpmModules+"/x")
	g.Expect(globalRecorded).To(BeFalse())
}

func TestScanPiConfiguredSources_ProjectSettingsNeedTrust(t *testing.T) {
	t.Parallel()

	for _, trusted := range []bool{true, false} {
		t.Run(strconv.FormatBool(trusted), func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Project entries resolve against <cwd>/.pi.
			fsys := newFakeSkillFS().
				file(projectTop+"/.pi/settings.json", `{"skills": ["team"], "prompts": ["tp"]}`).
				file(projectTop+"/.pi/team/fmt/SKILL.md", "fmt").
				file(projectTop+"/.pi/tp/go.md", "go")

			result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(trusted))

			if !trusted {
				g.Expect(result.Candidates).To(BeEmpty())
				g.Expect(result.Roots).To(BeEmpty())

				return
			}

			g.Expect(candidateNames(result)).To(Equal([]string{"fmt", "go"}))

			for _, candidate := range result.Candidates {
				g.Expect(candidate.ScopeID).To(Equal(projectScope))
				g.Expect(candidate.SourceSegment).To(Equal(cli.SkillScopePiSettings))
			}
		})
	}
}

func TestScanPiConfiguredSources_PromptTreeDepthLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deep := fakeHome + "/p" + strings.Repeat("/d", piDeepNesting)
	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"prompts": ["~/p"]}`).
		file(deep+"/x.md", "x")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	scanned, _ := rootScanned(result, fakeHome+"/p")
	g.Expect(scanned).To(BeFalse())
}

func TestScanPiConfiguredSources_PromptTreeEdgeCases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// A dangling link is skipped, a cycle terminates, an unreadable prompt
	// or subdir marks only its own entry not scanned (ruling R5).
	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"prompts": ["~/ok", "~/badfile", "~/baddir", "~/bad.md"]}`).
		file(fakeHome+"/ok/a.md", "a").
		link(fakeHome+"/ok/gone.md", fakeHome+"/nowhere.md").
		link(fakeHome+"/ok/loop", fakeHome+"/ok").
		file(fakeHome+"/badfile/b.md", "b").failRead(fakeHome+"/badfile/b.md").
		dir(fakeHome+"/baddir/sub").failRead(fakeHome+"/baddir/sub").
		file(fakeHome+"/bad.md", "c").failRead(fakeHome + "/bad.md")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"a"}))

	for root, want := range map[string]bool{
		fakeHome + "/ok": true, fakeHome + "/badfile": false, fakeHome + "/baddir": false, fakeHome + "/bad.md": false,
	} {
		scanned, found := rootScanned(result, root)
		g.Expect(found).To(BeTrue(), root)
		g.Expect(scanned).To(Equal(want), root)
	}

	g.Expect(result.Warnings).To(HaveLen(3))
}

func TestScanPiConfiguredSources_SettingsPromptDirIsRecursive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Ruling R10 Q2 (package-manager.js:149-192, 463).
	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"prompts": ["~/p", "~/one.md"]}`).
		file(fakeHome+"/p/a.md", "a").
		file(fakeHome+"/p/sub/b.md", "b").
		file(fakeHome+"/p/.hidden/c.md", "c").
		file(fakeHome+"/p/node_modules/d.md", "d").
		file(fakeHome+"/p/e.txt", "e").
		file(fakeHome+"/one.md", "one")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(candidateNames(result)).To(Equal([]string{"a", "b", "one"}))

	for _, candidate := range result.Candidates {
		g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindPrompt))
		g.Expect(candidate.ScopeID).To(Equal(cli.SkillScopePiSettings))
	}
}

// TestScanPiConfiguredSources_SettingsSkillPath covers the spec scenario
// "Pi settings skill path", with the entry naming the skills folder and the
// skill folder itself.
func TestScanPiConfiguredSources_SettingsSkillPath(t *testing.T) {
	t.Parallel()

	for _, entry := range []string{"~/extra-skills", "~/extra-skills/fmt"} {
		t.Run(entry, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fsys := newFakeSkillFS().
				file(piAgentDir+"/settings.json", `{"skills": ["`+entry+`"]}`).
				file(fakeHome+"/extra-skills/fmt/SKILL.md", "fmt")

			result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

			g.Expect(result.Candidates).To(HaveLen(1))

			if len(result.Candidates) != 1 {
				return
			}

			candidate := result.Candidates[0]
			g.Expect(candidate.Name).To(Equal("fmt"))
			g.Expect(candidate.ScopeID).To(Equal(cli.SkillScopePiSettings))
			g.Expect(candidate.SourceSegment).To(Equal(cli.SkillScopePiSettings))
			g.Expect(candidate.Kind).To(Equal(cli.SkillSourceKindSkill))
			g.Expect(candidate.SourcePath).To(Equal(fakeHome + "/extra-skills/fmt/SKILL.md"))
			g.Expect(candidate.ReadRoot).To(Equal(strings.Replace(entry, "~", fakeHome, 1)))
			g.Expect(candidate.Key).To(BeEmpty())
			g.Expect(candidate.Disabled).To(BeFalse())
		})
	}
}

func TestScanPiConfiguredSources_UnreadableChildMarksEntryNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"skills": ["~/a", "~/b"]}`).
		file(fakeHome+"/a/x/SKILL.md", "x").failRead(fakeHome+"/a/x/SKILL.md").
		file(fakeHome+"/b/y/SKILL.md", "y")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	aScanned, _ := rootScanned(result, fakeHome+"/a")
	bScanned, _ := rootScanned(result, fakeHome+"/b")
	g.Expect(aScanned).To(BeFalse())
	g.Expect(bScanned).To(BeTrue())
	g.Expect(result.Warnings).NotTo(BeEmpty())
}

func TestScanPiConfiguredSources_UnreadableManifestIsNotScanned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(piAgentDir+"/settings.json", `{"packages": ["npm:x"]}`).
		file(piNpmModules+"/x/package.json", `{}`).failRead(piNpmModules+"/x/package.json").
		file(piNpmModules+"/x/skills/fmt/SKILL.md", "fmt")

	result := cli.ScanPiConfiguredSources(fsys, piConfiguredScan(false))

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(result.Warnings).NotTo(BeEmpty())

	scanned, _ := rootScanned(result, piNpmModules+"/x")
	g.Expect(scanned).To(BeFalse())
}

// unexported constants.
const (
	fakeHome     = "/home/joe"
	piNpmModules = piAgentDir + "/npm/node_modules"
)

// disabledNames returns the sorted names of the Disabled candidates.
func disabledNames(candidates []cli.SkillCandidate) []string {
	names := []string{}

	for _, candidate := range candidates {
		if candidate.Disabled {
			names = append(names, candidate.Name)
		}
	}

	sort.Strings(names)

	return names
}

// matchesAnyPrefix reports whether name starts with any of prefixes.
func matchesAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

func piConfiguredScan(trusted bool) cli.PiConfiguredScan {
	return cli.PiConfiguredScan{
		Home:     fakeHome,
		AgentDir: piAgentDir,
		Cwd:      projectTop,
		ScopeID:  projectScope,
		Trusted:  trusted,
	}
}

// piFilterOracle computes, from the generated patterns, the names Pi's
// applyPatterns disables: included (by prefix glob, all when none) and not
// `!`-excluded, then exact `+` re-adds and exact `-` removes.
func piFilterOracle(names, includes, patterns []string) []string {
	disabled := []string{}

	for _, name := range names {
		path := fakeHome + "/extra-skills/" + name
		enabled := (len(includes) == 0 || matchesAnyPrefix(name, includes)) && !slices.Contains(patterns, "!"+name)

		if slices.Contains(patterns, "+"+path) {
			enabled = true
		}

		if slices.Contains(patterns, "-"+path) {
			enabled = false
		}

		if !enabled {
			disabled = append(disabled, name)
		}
	}

	return disabled
}

func prefixAll(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, prefix+value)
	}

	return out
}

// quoteJoin renders values as a comma-separated list of JSON strings.
func quoteJoin(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return strings.Join(quoted, ", ")
}
