package cli_test

// Tests for openspec change register-skills-all-skill-folders section 8,
// "qualify every key" (design D3, D11): every key is qualified by its
// source, engram-installed skills are keyed by their folder, unkeyed legacy
// notes are not skill notes, and an unrecognized skill_key is never matched
// or removed.

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestAdoptSkillNote_MigratesAnUnkeyedLegacyNote is the spec scenario
// "Adopting a previously promoted note" for D11's migration: `--adopt
// claude:curate=1049` over the unkeyed legacy note 1049 renames it to the
// key-derived slug, rewrites the referrer's link and rebuilds its sidecar,
// names skill_source in the preamble, stamps skill_key and skill_source,
// keeps skill_hash and clears pending.
func TestAdoptSkillNote_MigratesAnUnkeyedLegacyNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		oldBasename = "1049.2026-09-21.skill-curate"
		newBasename = "1049.2026-09-21.skill-claude-curate"
		referrer    = "2000.2026-09-22.some-other-note"
	)

	content := []byte("# Curate\n\n1. Judge offers.\n")
	hash := cli.SkillContentHash(content)

	vault := newSkillAcceptFixtureVault()
	vault.put(oldBasename+".md", strings.Replace(legacySkillNote("1049", "curate", hash),
		"situation: s\n", "situation: s\npending: true\n", 1))

	referrerContent := referencingNoteFixture(oldBasename)
	vault.put(referrer+".md", referrerContent)

	fakeEmbedder := skillAcceptFakeEmbedder{}
	sidecar, buildErr := embed.BuildSidecar(t.Context(), fakeEmbedder, []byte(referrerContent))
	g.Expect(buildErr).NotTo(HaveOccurred())
	vault.put(referrer+".vec.json", string(embed.MarshalSidecar(sidecar)))

	var sources cli.ResolvedSkillSources

	sources.Candidates = []cli.SkillCandidate{
		offerCand("claude:curate", cli.SkillScopeClaudeUser, userSkillsRoot+"/curate/SKILL.md", string(content)),
	}

	var stdout bytes.Buffer

	err := cli.ExportAnswerSkillSources(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: fakeHome,
		Adopt: map[string]string{"claude:curate": "1049"},
	}, sources, skillRegistrationDepsFor(vault, nil), &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, oldStillThere := vault.get(oldBasename + ".md")
	g.Expect(oldStillThere).To(BeFalse())

	adopted, found := vault.get(newBasename + ".md")
	g.Expect(found).To(BeTrue())

	doc := parseSkillAcceptFrontmatter(g, adopted)
	g.Expect(doc.SkillKey).To(Equal("claude:curate"))
	g.Expect(doc.SkillSource).To(Equal("~/.claude/skills/curate/SKILL.md"))
	g.Expect(doc.SkillHash).To(Equal(hash))
	g.Expect(doc.Luhmann).To(Equal("1049"))
	g.Expect(doc.Pending).To(BeFalse())
	g.Expect(adopted).NotTo(ContainSubstring("pending"))
	g.Expect(skillNoteBodyOf(adopted)).To(Equal("> Mirrors skill `~/.claude/skills/curate/SKILL.md`.\n\n"+string(content)),
		"the preamble is exactly the D8 form, with no edit-location clause")

	rewritten, _ := vault.get(referrer + ".md")
	g.Expect(rewritten).To(ContainSubstring("[[" + newBasename + "]]"))
	g.Expect(rewritten).NotTo(ContainSubstring("[[" + oldBasename + "]]"))
	g.Expect(embed.ComputeState(vault, "/vault/"+referrer+".md", fakeEmbedder.ModelID())).To(Equal(embed.StateOK))
	g.Expect(stdout.String()).NotTo(ContainSubstring("awaiting an answer"), "the adopted key makes no offer")
}

// TestAssignSkillKeys_QualifiesClaudeUserSources covers design D3's Claude
// user rows: skills key as `claude:<n>` and commands as `claude:cmd:<ns:…:n>`
// (a skill named `cmd` is `claude:cmd`), and engram's installed skills are
// keyed by the folder they are found in, with no path-based exception.
func TestAssignSkillKeys_QualifiesClaudeUserSources(t *testing.T) {
	t.Parallel()

	skill, command := cli.SkillSourceKindSkill, cli.SkillSourceKindCommand

	for _, testCase := range []struct {
		scope      string
		kind       cli.SkillSourceKind
		name, path string
		key        string
	}{
		{cli.SkillScopeClaudeUser, skill, "c4", userSkillsRoot + "/c4/SKILL.md", "claude:c4"},
		{cli.SkillScopeClaudeUser, skill, "cmd", userSkillsRoot + "/cmd/SKILL.md", "claude:cmd"},
		{cli.SkillScopeClaudeCmd, command, "audit", fakeHome + "/.claude/commands/audit.md", "claude:cmd:audit"},
		{cli.SkillScopeClaudeCmd, command, "opsx:apply", fakeHome + "/.claude/commands/opsx/apply.md",
			"claude:cmd:opsx:apply"},
		{cli.SkillScopeClaudeUser, skill, "route", engramClaudeSkills + "/route/SKILL.md", "claude:route"},
		{cli.SkillScopePiUser, skill, "route", engramPiSkills + "/route/SKILL.md", "pi:route"},
	} {
		t.Run(testCase.key+" from "+testCase.path, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			keyed, warnings := cli.AssignSkillKeys([]cli.SkillCandidate{{
				Name: testCase.name, ScopeID: testCase.scope, Kind: testCase.kind, SourcePath: testCase.path,
			}})

			g.Expect(warnings).To(BeEmpty())
			g.Expect(keyed).To(HaveLen(1))

			if len(keyed) != 1 {
				return
			}

			g.Expect(keyed[0].Key).To(Equal(testCase.key))
		})
	}
}

// TestCompareSkillOffers_ClaudeKeysParseToClaudeForms covers the key parser
// for `claude:` keys (design D3, D5): `claude:<n>` is a Claude user skill
// (scope claude-user, root ~/.claude/skills), `claude:cmd:…` with three or
// more segments a Claude user command (scope claude-cmd, root
// ~/.claude/commands), and `claude:cmd` a skill named cmd.
func TestCompareSkillOffers_ClaudeKeysParseToClaudeForms(t *testing.T) {
	t.Parallel()

	commandsRoot := fakeHome + "/.claude/commands"
	notes := func(fixture *offerFixture) {
		fixture.note("claude:c4", "h1", "")
		fixture.note("claude:cmd", "h2", "")
		fixture.note("claude:cmd:audit", "h3", "")
		fixture.note("claude:cmd:opsx:apply", "h4", "")
	}

	t.Run("both roots read", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.home = fakeHome
		fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
		fixture.root(commandsRoot, cli.SkillRootFormClaudeCmd, true)
		notes(fixture)

		g.Expect(offerScopes(fixture.compare(g))).To(ConsistOf(
			"remove claude:c4 claude-user",
			"remove claude:cmd claude-user",
			"remove claude:cmd:audit claude-cmd",
			"remove claude:cmd:opsx:apply claude-cmd",
		))
	})

	t.Run("an unread commands root keeps only the command notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.home = fakeHome
		fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, true)
		fixture.root(commandsRoot, cli.SkillRootFormClaudeCmd, false)
		notes(fixture)

		g.Expect(offerScopes(fixture.compare(g))).To(ConsistOf(
			"remove claude:c4 claude-user",
			"remove claude:cmd claude-user",
		))
	})

	t.Run("an unread skills root keeps only the skill notes", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fixture := newOfferFixture()
		fixture.home = fakeHome
		fixture.root(userSkillsRoot, cli.SkillRootFormClaudeUser, false)
		fixture.root(commandsRoot, cli.SkillRootFormClaudeCmd, true)
		notes(fixture)

		g.Expect(offerScopes(fixture.compare(g))).To(ConsistOf(
			"remove claude:cmd:audit claude-cmd",
			"remove claude:cmd:opsx:apply claude-cmd",
		))
	})
}

// TestCompareSkillOffers_SixLegacyNotesYieldSixRegisterOffers is the spec
// scenario "Unkeyed legacy notes are ignored until adopted" (design D11):
// the six unkeyed legacy notes, whose skill_hash equals the scanned bytes,
// are not skill notes, so each of engram's six skills found through
// ~/.claude/skills is offered for registration as `claude:<n>`; the
// byte-identical Pi copies alias them, and nothing is refreshed or removed.
func TestCompareSkillOffers_SixLegacyNotesYieldSixRegisterOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault, names := legacyVault()

	var sources cli.ResolvedSkillSources

	sources.Roots = []cli.ScannedRoot{
		{Path: userSkillsRoot, Resolved: userSkillsRoot, Scanned: true, Form: cli.SkillRootFormClaudeUser},
		{Path: piUserRoot, Resolved: piUserRoot, Scanned: true, Form: cli.SkillRootFormPiUser},
	}

	want := make([]string, 0, len(legacyNotes))

	for _, name := range legacyNotes {
		sources.Candidates = append(sources.Candidates,
			offerCand("claude:"+name, cli.SkillScopeClaudeUser, engramClaudeSkills+"/"+name+"/SKILL.md", name),
			offerCand("pi:"+name, cli.SkillScopePiUser, engramPiSkills+"/"+name+"/SKILL.md", name))
		want = append(want, "register claude:"+name+" claude-user")
	}

	comparison, err := cli.CompareSkillOffers(cli.SkillOfferInput{
		Vault: skillregFixtureVaultRoot, Names: names, ReadFile: vault.readFile,
		Declined: map[string]string{}, Home: fakeHome, Sources: sources,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(offerScopes(comparison)).To(ConsistOf(want))
	g.Expect(comparison.Conflicts).To(BeEmpty())
	g.Expect(comparison.Warnings).To(BeEmpty())
}

// TestCompareSkillOffers_UnkeyedNotesNeverChangeTheOfferSet is 8.2's
// property: for any vault, adding or removing unkeyed notes (skill_hash, no
// skill_key) — whatever their slug or hash — never changes the offer set.
func TestCompareSkillOffers_UnkeyedNotesNeverChangeTheOfferSet(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		fixture, _ := drawRemovalUniverse(rt)
		fixture.sources.Candidates = append(fixture.sources.Candidates, drawDedupeUniverse(rt).sources.Candidates...)

		baseline, baseErr := compareWithUnkeyedNotes(fixture, nil)
		if baseErr != nil {
			rt.Fatalf("baseline: %v", baseErr)
		}

		unkeyed := drawUnkeyedNotes(rt, fixture)

		withUnkeyed, withErr := compareWithUnkeyedNotes(fixture, unkeyed)
		if withErr != nil {
			rt.Fatalf("with unkeyed notes %v: %v", unkeyed, withErr)
		}

		if strings.Join(offerScopes(baseline), "\n") != strings.Join(offerScopes(withUnkeyed), "\n") {
			rt.Fatalf("unkeyed notes %v changed the offers:\nbefore %v\nafter  %v",
				slices.Sorted(maps.Keys(unkeyed)), offerScopes(baseline), offerScopes(withUnkeyed))
		}
	})
}

// TestCompareSkillOffers_UnrecognizedKeyProperty is 8.2b's property: over
// generated keys, root read states, plugin facts and candidates, a note
// whose key is unrecognized is never removal-eligible, never offered for
// removal and never makes a candidate an alias; every other generated
// `<x>:<n…>` with <x> not a reserved form is judged by the plugin rule.
func TestCompareSkillOffers_UnrecognizedKeyProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		key, wantRecognized := drawSkillKeyForParsing(rt)

		recognized, _, _, plugin, _ := cli.ExportParseSkillKey(key)
		if recognized != wantRecognized {
			rt.Fatalf("key %q: recognized=%v, want %v", key, recognized, wantRecognized)
		}

		fixture := drawEligibilityUniverse(rt)
		note := offerNote{key: key, hash: cli.SkillContentHash([]byte("note bytes")), source: drawNoteSource(rt)}
		eligible := cli.ExportSkillKeyRemovalEligible(fixture.input(), key, note.source)

		if !recognized {
			assertUnrecognizedNoteIsInert(rt, fixture, note, eligible)

			return
		}

		if plugin == "" {
			return
		}

		scanned, installed := fixture.pluginState(plugin)
		want := fixture.sources.PluginManifestsRead && !fixture.pluginConflicted(plugin) && (!installed || scanned)

		if eligible != want {
			rt.Fatalf("plugin key %q: eligible=%v, want %v (plugin rule)", key, eligible, want)
		}
	})
}

// TestCompareSkillOffers_UnrecognizedKeysAreNeverMatchedOrRemoved is the
// spec scenario "An unrecognized skill_key is never matched or removed"
// (design D3): with every source root read and the plugin manifests
// parsed, a note whose key has no `:`, an empty segment, or a reserved
// prefix with a malformed tail is never offered for removal and makes no
// scanned entry an alias; any other `<x>:<n>` is a plugin key, removable
// when plugin <x> is not installed (like `ralph-loop:cmd:help`).
func TestCompareSkillOffers_UnrecognizedKeysAreNeverMatchedOrRemoved(t *testing.T) {
	t.Parallel()

	for _, key := range unrecognizedKeyExamples() {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fixture := everyRootReadFixture()
			fixture.note(key, cli.SkillContentHash([]byte("same bytes")), "~/.claude/skills/route/SKILL.md")
			fixture.candidate(offerCand("claude:route", cli.SkillScopeClaudeUser,
				userSkillsRoot+"/route/SKILL.md", "same bytes"))

			g.Expect(offerScopes(fixture.compare(g))).To(Equal([]string{"register claude:route claude-user"}),
				"no removal, and the note's hash makes no alias")
		})
	}

	for _, key := range []string{"engram:route", "ralph-loop:cmd:help", "cmd:x", "claude-hud:x:y"} {
		t.Run(key+" is a plugin key", func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fixture := everyRootReadFixture()
			fixture.note(key, "h", "")

			plugin, _, _ := strings.Cut(key, ":")
			g.Expect(offerScopes(fixture.compare(g))).To(Equal([]string{"remove " + key + " plugin:" + plugin}))
		})
	}
}

func TestFindSkillNote_DuplicateQualifiedKeyIsAnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		first  = "1049.2026-09-21.skill-claude-curate.md"
		second = "2001.2026-09-26.skill-claude-curate.md"
	)

	vault := newSkillregFixtureVault()
	vault.put(first, keyedSkillNote("abc", "claude:curate"))
	vault.put(second, keyedSkillNote("def", "claude:curate"))

	_, _, found, err := cli.FindSkillNote("/vault", "claude:curate", []string{first, second}, vault.readFile)

	g.Expect(found).To(BeFalse())
	g.Expect(err).To(MatchError(cli.ErrDuplicateSkillNoteForTest))
	g.Expect(err).To(MatchError(ContainSubstring("1049.2026-09-21.skill-claude-curate")))
	g.Expect(err).To(MatchError(ContainSubstring("2001.2026-09-26.skill-claude-curate")))
}

// TestFindSkillNote_SlugAloneLocatesNothing is the spec scenario "Note
// located by slug": an unkeyed legacy note is no key's note, whatever its
// slug says.
func TestFindSkillNote_SlugAloneLocatesNothing(t *testing.T) {
	t.Parallel()

	vault, names := legacyVault()

	for _, name := range legacyNotes {
		for _, key := range []string{name, "claude:" + name, "pi:" + name} {
			t.Run(key, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				_, _, found, err := cli.FindSkillNote("/vault", key, names, vault.readFile)

				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(found).To(BeFalse())
			})
		}
	}
}

// TestScanClaudePlugins_ReservedNameCheckFollowsEnablement covers the spec
// scenarios "A plugin named claude is reserved", "A disabled reserved-name
// plugin is silent" and "A plugin named cmd is an ordinary plugin" (design
// D3): the reserved set is exactly the top-level key forms, and the check
// runs only once the plugin is known to be enabled.
func TestScanClaudePlugins_ReservedNameCheckFollowsEnablement(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name         string
		plugin       string
		enabled      bool
		wantWarning  bool
		wantKeys     []string
		manifestJSON string
	}{
		{name: "enabled claude@m warns", plugin: "claude", enabled: true, wantWarning: true},
		{name: "disabled pi@m is silent", plugin: "pi", enabled: false},
		{name: "disabled engram@engram is silent", plugin: "engram", enabled: false},
		{
			name: "claude@m disabled by defaultEnabled is silent", plugin: "claude", enabled: true,
			manifestJSON: `{"name":"claude","defaultEnabled":false}`,
		},
		{name: "enabled cmd@m is an ordinary plugin", plugin: "cmd", enabled: true, wantKeys: []string{"cmd:x"}},
		{name: "enabled engram@m is an ordinary plugin", plugin: "engram", enabled: true, wantKeys: []string{"engram:x"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			marketplace := "m"
			if testCase.plugin == "engram" && !testCase.enabled {
				marketplace = "engram"
			}

			pluginKey := testCase.plugin + "@" + marketplace
			root := pluginCacheRoot + "/" + marketplace + "/" + testCase.plugin + "/1"
			fsys := newFakeSkillFS().file(root+"/skills/x/SKILL.md", "x")

			enabled := map[string]bool{pluginKey: testCase.enabled}
			if testCase.manifestJSON != "" {
				fsys.file(root+"/.claude-plugin/plugin.json", testCase.manifestJSON)
				enabled = map[string]bool{}
			}

			writePluginManifests(g, fsys, []pluginInstall{{key: pluginKey, installPath: root}}, enabled)

			result := cli.ScanClaudePlugins(fsys, pluginScanInput())
			keyed, _ := cli.AssignSkillKeys(result.Candidates)

			g.Expect(resolvedKeys(cli.ResolvedSkillSources{
				SkillScanResult: cli.SkillScanResult{Candidates: keyed},
			})).To(Equal(stringsOrEmpty(testCase.wantKeys)))
			g.Expect(warningsMention(result.SkillScanResult, "reserved")).To(Equal(testCase.wantWarning),
				"warnings: %v", result.Warnings)
		})
	}
}

func (fixture *offerFixture) input() cli.SkillOfferInput {
	return cli.SkillOfferInput{Home: fixture.home, Sources: fixture.sources, Declined: fixture.declined}
}

func (fixture *offerFixture) pluginConflicted(plugin string) bool {
	return slices.ContainsFunc(fixture.sources.PluginConflicts, func(conflict cli.ClaudePluginConflict) bool {
		return conflict.Plugin == plugin
	})
}

func (fixture *offerFixture) pluginState(plugin string) (scanned, installed bool) {
	for _, status := range fixture.sources.Plugins {
		if status.Name == plugin {
			return status.Scanned, true
		}
	}

	return false, false
}

// assertUnrecognizedNoteIsInert checks an unrecognized note against D3:
// not removal-eligible, never offered, and its hash aliases nothing (a
// candidate with the same bytes and a fresh key is still offered).
func assertUnrecognizedNoteIsInert(rt *rapid.T, fixture *offerFixture, note offerNote, eligible bool) {
	if eligible {
		rt.Fatalf("unrecognized key %q is removal-eligible", note.key)
	}

	fixture.notes = append(fixture.notes, note)
	fixture.candidate(offerCand("claude:fresh-alias-probe", cli.SkillScopeClaudeUser,
		userSkillsRoot+"/fresh-alias-probe/SKILL.md", "note bytes"))

	comparison, err := fixture.compareRapid()
	if err != nil {
		rt.Fatalf("CompareSkillOffers: %v", err)
	}

	for _, offer := range comparison.Offers {
		if offer.Key == note.key {
			rt.Fatalf("unrecognized key %q got an offer: %+v", note.key, offer)
		}
	}

	if !slices.Contains(offerScopes(comparison), "register claude:fresh-alias-probe claude-user") {
		rt.Fatalf("unrecognized key %q made a candidate an alias: %v", note.key, offerScopes(comparison))
	}
}

// compareWithUnkeyedNotes runs fixture's comparison over a vault holding
// fixture's keyed notes plus the given unkeyed notes (full name → content).
func compareWithUnkeyedNotes(fixture *offerFixture, unkeyed map[string]string) (cli.SkillOfferComparison, error) {
	vault := newSkillregFixtureVault()
	names := make([]string, 0, len(fixture.notes)+len(unkeyed))

	for index, note := range fixture.notes {
		name := fmt.Sprintf("%d.2026-09-26.%s.md", index+1, cli.SkillKeySlug(note.key))
		vault.put(name, sourcedSkillNote(note))
		names = append(names, name)
	}

	for _, name := range slices.Sorted(maps.Keys(unkeyed)) {
		vault.put(name, unkeyed[name])
		names = append(names, name)
	}

	return cli.CompareSkillOffers(cli.SkillOfferInput{
		Vault: skillregFixtureVaultRoot, Names: names, ReadFile: vault.readFile,
		Declined: fixture.declined, Home: fixture.home, Sources: fixture.sources, NoRemovals: fixture.noRemovals,
	})
}

// drawEligibilityUniverse draws read states for every fixed root, some
// source-rooted roots holding the drawn note sources, the manifest read
// status and plugin facts over a small pool of plugin names.
func drawEligibilityUniverse(rt *rapid.T) *offerFixture {
	fixture := newOfferFixture()
	fixture.home = fakeHome

	for _, name := range slices.Sorted(maps.Keys(removalFixedForms())) {
		spec := removalFixedForms()[name]
		fixture.root(spec.root, spec.form, rapid.Bool().Draw(rt, "fixedRead:"+name))
	}

	for _, spec := range removalSourcedForms() {
		if rapid.Bool().Draw(rt, "sourcedRoot:"+string(spec.form)) {
			fixture.root("/r/own", spec.form, rapid.Bool().Draw(rt, "sourcedRead:"+string(spec.form)), spec.prefix)
		}
	}

	fixture.sources.PluginManifestsRead = rapid.Bool().Draw(rt, "manifestsRead")

	for _, plugin := range eligibilityPluginPool() {
		fixture.drawPluginState(rt, plugin, rapid.IntRange(0, removalStateCount-1).Draw(rt, "pluginState:"+plugin))
	}

	return fixture
}

// drawNoteSource draws a note's recorded skill_source: none, or one under
// a drawn source root.
func drawNoteSource(rt *rapid.T) string {
	return rapid.SampledFrom([]string{"", "/r/own/x/SKILL.md", "~/.claude/skills/x/SKILL.md"}).Draw(rt, "noteSource")
}

// drawSkillKeyForParsing draws a key and whether design D3 recognizes it:
// a well-shaped key of any form (plugin keys included, with a plugin name
// that is not a reserved form), or one of the three unrecognized shapes.
func drawSkillKeyForParsing(rt *rapid.T) (string, bool) {
	// A name never equals a sub-source segment, so `claude:<n>:<n>` and
	// `project:<r>:<n>:<n>` stay malformed.
	name := rapid.StringMatching(`[a-z0-9][a-z0-9._-]{0,5}`).
		Filter(func(name string) bool { return !slices.Contains([]string{"cmd", "pi", "agents", "pi-pkg"}, name) }).
		Draw(rt, "name")
	plugin := rapid.SampledFrom(eligibilityPluginPool()).Draw(rt, "plugin")

	recognizedShapes := []string{
		"claude:" + name, "claude:cmd", "claude:cmd:" + name, "claude:cmd:ns:" + name,
		"pi:" + name, "agents:" + name, "pi-prompt:" + name, "anthropic-skills:" + name,
		"pi-settings:" + name, "pi-settings:pi-prompt:" + name,
		"pi-pkg:pk:" + name, "pi-pkg:pk:pi-prompt:" + name, "pi-pkg:a:b:" + name,
		"project:github.com/a/b:" + name, "project:github.com/a/b:cmd:" + name, "project:github.com/a/b:cmd",
		"project:github.com/a/b:pi:" + name, "project:github.com/a/b:agents:" + name,
		"project:github.com/a/b:pi-prompt:" + name, "project:github.com/a/b:pi-settings:" + name,
		"project:github.com/a/b:pi-settings:pi-prompt:" + name, "project:github.com/a/b:pi-pkg:pk:" + name,
		plugin + ":" + name, plugin + ":cmd:" + name, plugin + ":" + name + ":" + name,
	}

	unrecognizedShapes := append(unrecognizedKeyExamples(),
		name, name+":", ":"+name, "claude::"+name, plugin+"::"+name,
		"claude:"+name+":"+name, "pi:"+name+":"+name, "project:github.com/a/b",
		"project:github.com/a/b:"+name+":"+name, "pi-pkg:"+name, "pi-settings:"+name+":"+name)

	if rapid.Bool().Draw(rt, "recognized") {
		return rapid.SampledFrom(recognizedShapes).Draw(rt, "recognizedKey"), true
	}

	return rapid.SampledFrom(unrecognizedShapes).Draw(rt, "unrecognizedKey"), false
}

// drawUnkeyedNotes draws unkeyed legacy-shaped notes whose slugs and hashes
// collide with fixture's keys, key remainders and contents, so any fallback
// that read them would change the offers.
func drawUnkeyedNotes(rt *rapid.T, fixture *offerFixture) map[string]string {
	size := len(fixture.notes) + len(fixture.sources.Candidates)
	slugs := append(make([]string, 0, 3+2*size), "skill-route", "skill-a", "skill-claude-a")
	hashes := append(make([]string, 0, 3+size),
		cli.SkillContentHash([]byte("one")), cli.SkillContentHash([]byte("two")), "stale")

	for _, note := range fixture.notes {
		key := note.key
		slugs = append(slugs, cli.SkillKeySlug(key), "skill-"+key[strings.LastIndex(key, ":")+1:])
		hashes = append(hashes, note.hash)
	}

	for _, candidate := range fixture.sources.Candidates {
		slugs = append(slugs, cli.SkillKeySlug(candidate.Key), "skill-"+candidate.Name)
		hashes = append(hashes, cli.SkillContentHash(candidate.Content))
	}

	count := rapid.IntRange(1, 4).Draw(rt, "unkeyedCount")
	unkeyed := make(map[string]string, count)

	for index := range count {
		slug := rapid.SampledFrom(slugs).Draw(rt, fmt.Sprintf("unkeyedSlug%d", index))
		hash := rapid.SampledFrom(hashes).Draw(rt, fmt.Sprintf("unkeyedHash%d", index))
		name := fmt.Sprintf("%d.2026-09-18.%s.md", 900+index, slug)
		unkeyed[name] = legacySkillNote(strconv.Itoa(900+index), strings.TrimPrefix(slug, "skill-"), hash)
	}

	return unkeyed
}

// eligibilityPluginPool is a small pool of plugin names (none reserved),
// including names that echo other forms' segments.
func eligibilityPluginPool() []string {
	return []string{"engram", "cmd", "ralph-loop", "claude-hud", "pi-intercom"}
}

// everyRootReadFixture reads every single-root user form's root and parses
// the plugin manifests, with no plugin installed.
func everyRootReadFixture() *offerFixture {
	fixture := newOfferFixture()
	fixture.home = fakeHome

	for _, spec := range removalFixedForms() {
		fixture.root(spec.root, spec.form, true)
	}

	fixture.sources.PluginManifestsRead = true

	return fixture
}

// offerScopes renders each offer as "<kind> <key> <scope>", sorted.
func offerScopes(comparison cli.SkillOfferComparison) []string {
	summaries := make([]string, 0, len(comparison.Offers))
	for _, offer := range comparison.Offers {
		summaries = append(summaries, fmt.Sprintf("%s %s %s", offer.Kind, offer.Key, offer.ScopeID))
	}

	slices.Sort(summaries)

	return summaries
}

// unrecognizedKeyExamples are skill_key values of design D3's three
// unrecognized cases: no `:`, an empty segment, and a reserved form's
// prefix with a tail that lacks that form's shape.
func unrecognizedKeyExamples() []string {
	return []string{
		"route", "curate",
		"claude:", ":x", "pi::x", "anthropic-skills:", "x::y",
		"project:github.com/toejough/engram", "pi-pkg:x", "claude:cmd:", "claude:a:b",
		"pi:a:b", "agents:a:b", "anthropic-skills:a:b", "pi-prompt:a:b",
		"pi-settings:a:b", "pi-settings:pi-prompt:a:b",
		"project:github.com/toejough/engram:mystery:x", "project:github.com/toejough/engram:pi:a:b",
	}
}
