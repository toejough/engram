package cli_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

func TestAssignSkillKeys_BuildsDesignD3Keys(t *testing.T) {
	t.Parallel()

	const project = "project:github.com/toejough/engram"

	skill, command, prompt := cli.SkillSourceKindSkill, cli.SkillSourceKindCommand, cli.SkillSourceKindPrompt

	for _, testCase := range []struct {
		scope, segment string
		kind           cli.SkillSourceKind
		name, key      string
	}{
		{cli.SkillScopeClaudeUser, "", skill, "c4", "claude:c4"},
		{cli.SkillScopeClaudeCmd, "", command, "opsx:apply", "claude:cmd:opsx:apply"},
		{cli.SkillScopeSynced, "", skill, "pdf", "anthropic-skills:pdf"},
		{cli.SkillScopePiUser, "", skill, "ping", "pi:ping"},
		{cli.SkillScopeAgentsUser, "", skill, "ag", "agents:ag"},
		{cli.SkillScopePiSettings, cli.SkillScopePiSettings, skill, "fmt", "pi-settings:fmt"},
		{cli.SkillScopePiPrompt, "", prompt, "review", "pi-prompt:review"},
		{cli.SkillScopePiSettings, cli.SkillScopePiSettings, prompt, "tidy", "pi-settings:pi-prompt:tidy"},
		{"pi-pkg:pi-intercom", "pi-pkg:pi-intercom", skill, "pi-intercom", "pi-pkg:pi-intercom:pi-intercom"},
		{"pi-pkg:pk", "pi-pkg:pk", prompt, "p", "pi-pkg:pk:pi-prompt:p"},
		{"plugin:superpowers", "", skill, "brainstorming", "superpowers:brainstorming"},
		{"plugin:commit", "", skill, "commit", "commit:commit"},
		{"plugin:commit", "", command, "commit", "commit:cmd:commit"},
		{"plugin:db", "", command, "db:migrate", "db:cmd:db:migrate"},
		{project, "", skill, "openspec-propose", project + ":openspec-propose"},
		{project, "", command, "opsx:apply", project + ":cmd:opsx:apply"},
		{project, cli.SkillSegmentPi, skill, "pj", project + ":pi:pj"},
		{project, cli.SkillSegmentAgents, skill, "aj", project + ":agents:aj"},
		{project, cli.SkillScopePiPrompt, prompt, "pp", project + ":pi-prompt:pp"},
		{project, cli.SkillScopePiSettings, skill, "px", project + ":pi-settings:px"},
		{project, cli.SkillScopePiSettings, prompt, "pq", project + ":pi-settings:pi-prompt:pq"},
		{project, "pi-pkg:ppk", skill, "ppks", project + ":pi-pkg:ppk:ppks"},
		{project, "pi-pkg:ppk", prompt, "pr", project + ":pi-pkg:ppk:pi-prompt:pr"},
	} {
		t.Run(testCase.key, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			keyed, warnings := cli.AssignSkillKeys([]cli.SkillCandidate{{
				Name: testCase.name, ScopeID: testCase.scope, SourceSegment: testCase.segment,
				Kind: testCase.kind, SourcePath: "/elsewhere/" + testCase.name,
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

func TestAssignSkillKeys_DisabledCandidatesAreKeyed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	keyed, warnings := cli.AssignSkillKeys([]cli.SkillCandidate{{
		Name: "x", ScopeID: cli.SkillScopePiUser, Kind: cli.SkillSourceKindSkill,
		SourcePath: piUserRoot + "/x/SKILL.md", Disabled: true,
	}})

	g.Expect(warnings).To(BeEmpty())
	g.Expect(keyed).To(HaveLen(1))

	if len(keyed) != 1 {
		return
	}

	g.Expect(keyed[0].Key).To(Equal("pi:x"))
	g.Expect(keyed[0].Disabled).To(BeTrue())
}

// TestAssignSkillKeys_KeyProperty checks D3's key rules over generated
// candidates: every key contains `:` and ends in `:<name>`; the key parser
// recognizes it and maps it to exactly one removal form (a fixed or
// source-rooted form, or a plugin) — the candidate's own — and to the
// candidate's scope; keys of distinct sources never collide; and every
// key's slug has the note-slug shape.
func TestAssignSkillKeys_KeyProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		count := rapid.IntRange(1, 12).Draw(rt, "count")
		candidates := make([]cli.SkillCandidate, 0, count)
		identities := map[string]bool{}

		for index := range count {
			candidate := drawKeyCandidate(rt, index)

			identity := candidate.ScopeID + "|" + candidate.SourceSegment + "|" + string(candidate.Kind) + "|" +
				candidate.Name
			if identities[identity] {
				continue
			}

			identities[identity] = true
			candidates = append(candidates, candidate)
		}

		keyed, warnings := cli.AssignSkillKeys(candidates)
		if len(warnings) != 0 {
			rt.Fatalf("unexpected warnings: %v", warnings)
		}

		if len(keyed) != len(candidates) {
			rt.Fatalf("keyed %d of %d candidates", len(keyed), len(candidates))
		}

		owners := map[string]cli.SkillCandidate{}

		for index, candidate := range keyed {
			assertKeyShape(rt, candidates[index], candidate.Key)

			if other, seen := owners[candidate.Key]; seen {
				rt.Fatalf("key %q shared by %+v and %+v", candidate.Key, other, candidate)
			}

			owners[candidate.Key] = candidate
		}
	})
}

func TestAssignSkillKeys_SkipsUnkeyableCandidates(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		candidate cli.SkillCandidate
		warning   string
	}{
		{
			name: "name containing a colon",
			candidate: cli.SkillCandidate{
				Name: "a:b", ScopeID: cli.SkillScopePiUser, Kind: cli.SkillSourceKindSkill,
				SourcePath: piUserRoot + "/a:b/SKILL.md",
			},
			warning: piUserRoot + "/a:b/SKILL.md",
		},
		{
			name: "prompt name containing a colon",
			candidate: cli.SkillCandidate{
				Name: "x:y", ScopeID: cli.SkillScopePiPrompt, Kind: cli.SkillSourceKindPrompt,
				SourcePath: piPromptsRoot + "/x:y.md",
			},
			warning: "x:y",
		},
		{
			name: "unknown scope",
			candidate: cli.SkillCandidate{
				Name: "x", ScopeID: "mystery", Kind: cli.SkillSourceKindSkill, SourcePath: "/m/x/SKILL.md",
			},
			warning: "mystery",
		},
		{
			name: "command with an empty namespace segment",
			candidate: cli.SkillCandidate{
				Name: "ns::x", ScopeID: cli.SkillScopeClaudeCmd, Kind: cli.SkillSourceKindCommand,
				SourcePath: fakeHome + "/.claude/commands/ns/x.md",
			},
			warning: "claude:cmd:ns::x",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			kept := cli.SkillCandidate{
				Name: "ok", ScopeID: cli.SkillScopeClaudeUser, Kind: cli.SkillSourceKindSkill,
				SourcePath: userSkillsRoot + "/ok/SKILL.md",
			}

			keyed, warnings := cli.AssignSkillKeys([]cli.SkillCandidate{testCase.candidate, kept})

			g.Expect(keyed).To(HaveLen(1))

			if len(keyed) != 1 {
				return
			}

			g.Expect(keyed[0].Key).To(Equal("claude:ok"))
			g.Expect(warnings).To(ConsistOf(ContainSubstring(testCase.warning)))
		})
	}
}

// TestResolveSkillSources_PiLinkThroughSymlinkedHome: `$HOME` is a symlink
// and Pi's route resolves into engram's installed copy; the entry is
// keyed by its folder (`pi:route`), and the resolved home is reported for
// writing `~`-relative skill_source values (design D8).
func TestResolveSkillSources_PiLinkThroughSymlinkedHome(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(engramPiSkills+"/route/SKILL.md", "route").
		link(piUserRoot+"/route", engramPiSkills+"/route").
		link("/links/home", fakeHome)

	resolved, err := cli.ResolveSkillSources(context.Background(), "/links/home", "/", cli.SkillSourceDeps{
		FS: fsys, Commander: scriptedGit{},
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolvedKeys(resolved)).To(Equal([]string{"pi:route"}))
	g.Expect(resolved.ResolvedHome).To(Equal(fakeHome), "skill_source is written relative to the resolved home too")
}

// TestResolveSkillSources_PiOnlyKeysByFolderAndIgnoresLegacyNote is the
// Pi-only fork case (design Risks): no ~/.claude, Pi's route resolves under
// Pi's engram root and is keyed `pi:route`; the unkeyed legacy note 1036 is
// no skill note, so it is neither refreshed nor an alias, and `pi:route` is
// offered for registration.
func TestResolveSkillSources_PiOnlyKeysByFolderAndIgnoresLegacyNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().
		file(engramPiSkills+"/route/SKILL.md", "route").
		link(piUserRoot+"/route", engramPiSkills+"/route")

	resolved, err := cli.ResolveSkillSources(context.Background(), fakeHome, "/", cli.SkillSourceDeps{
		FS: fsys, Commander: scriptedGit{},
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolvedKeys(resolved)).To(Equal([]string{"pi:route"}))

	vault := newSkillregFixtureVault()
	vault.put(legacyRouteNote, legacySkillNote("1036", "route", cli.SkillContentHash([]byte("route"))))

	comparison, offerErr := cli.CompareSkillOffers(cli.SkillOfferInput{
		Vault: skillregFixtureVaultRoot, Names: []string{legacyRouteNote}, ReadFile: vault.readFile,
		Declined: map[string]string{}, Home: fakeHome, Sources: resolved,
	})

	g.Expect(offerErr).NotTo(HaveOccurred())
	g.Expect(offerScopes(comparison)).To(Equal([]string{"register pi:route pi-user"}))
}

func TestResolveSkillSources_StampsKeysAndDropsColonNames(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := resolverFixture(g).file(userSkillsRoot+"/bad:name/SKILL.md", "bad")

	resolved, err := cli.ResolveSkillSources(
		context.Background(), fakeHome, projectTop, resolverDeps(fsys),
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolvedKeys(resolved)).To(Equal([]string{
		"claude:c4",
		"claude:cmd:audit",
		"anthropic-skills:pdf",
		"pi:ping",
		"agents:ag",
		"pi-settings:fmt",
		"pi-prompt:review",
		"pi-settings:pi-prompt:tidy",
		"pi-pkg:pk:pks",
		"tools:tl",
		projectScope + ":openspec-propose",
		projectScope + ":cmd:commit",
		projectScope + ":pi:pj",
		projectScope + ":agents:aj",
		projectScope + ":pi-settings:px",
		projectScope + ":pi-prompt:pp",
		projectScope + ":pi-settings:pi-prompt:pq",
		projectScope + ":pi-pkg:ppk:ppks",
	}))
	g.Expect(warningsMention(resolved.SkillScanResult, "bad:name")).To(BeTrue())
}

func TestScanCommandDir_SkipsEntriesContainingAColon(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	root := fakeHome + "/.claude/commands"
	fsys := newFakeSkillFS().
		file(root+"/ok.md", "ok").
		file(root+"/a:b.md", "flat").
		file(root+"/x:y/z.md", "nested").
		file(root+"/ns/c:d.md", "deep")

	result := cli.ScanCommandDir(fsys, root, cli.SkillScopeClaudeCmd)

	g.Expect(candidateNames(result)).To(Equal([]string{"ok"}))

	scanned, found := rootScanned(result, root)
	g.Expect(found).To(BeTrue())
	g.Expect(scanned).To(BeTrue())
	g.Expect(result.Warnings).To(ConsistOf(
		ContainSubstring("a:b.md"), ContainSubstring("x:y"), ContainSubstring("c:d.md"),
	))
}

func TestScanPluginCommands_SkipsDeclaredFileWithAColon(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newFakeSkillFS().file("/plug/extra/a:b.md", "flat")

	result := cli.ScanPluginCommands(fsys, "/plug", []byte(`["extra/a:b.md"]`), "plugin:plug")

	g.Expect(result.Candidates).To(BeEmpty())
	g.Expect(warningsMention(result, "a:b.md")).To(BeTrue())
}

func TestSkillKeySlug(t *testing.T) {
	t.Parallel()

	for key, slug := range map[string]string{
		"x":                         "skill-x",
		"route":                     "skill-route",
		"claude:route":              "skill-claude-route",
		"claude:cmd:audit":          "skill-claude-cmd-audit",
		"superpowers:brainstorming": "skill-superpowers-brainstorming",
		"project:github.com/toejough/engram:openspec-propose": "skill-project-github-com-toejough-engram-openspec-propose",
		"project:github.com/toejough/engram:cmd:opsx:apply":   "skill-project-github-com-toejough-engram-cmd-opsx-apply",
		"cmd:Opsx:Apply":   "skill-cmd-opsx-apply",
		"-a__b-":           "skill-a-b",
		"pi-prompt:review": "skill-pi-prompt-review",
	} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			g.Expect(cli.SkillKeySlug(key)).To(Equal(slug))
		})
	}
}

// TestSkillKeySlug_Property: any key holding a slug character yields a slug
// of the note-slug shape; a key already of that shape maps to `skill-<key>`;
// and slugging is idempotent on the remainder.
func TestSkillKeySlug_Property(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		key := rapid.String().Draw(rt, "key") + rapid.StringMatching(`[A-Za-z0-9]`).Draw(rt, "anchor") +
			rapid.String().Draw(rt, "tail")

		slug := cli.SkillKeySlug(key)
		if !noteSlugPattern.MatchString(slug) {
			rt.Fatalf("slug %q of key %q does not match %s", slug, key, noteSlugPattern)
		}

		remainder := strings.TrimPrefix(slug, "skill-")
		if cli.SkillKeySlug(remainder) != slug {
			rt.Fatalf("slug of remainder %q is %q, want %q", remainder, cli.SkillKeySlug(remainder), slug)
		}

		plain := rapid.StringMatching(`[a-z0-9]+(-[a-z0-9]+)*`).Draw(rt, "plain")
		if cli.SkillKeySlug(plain) != "skill-"+plain {
			rt.Fatalf("slug-shaped input %q gives slug %q", plain, cli.SkillKeySlug(plain))
		}
	})
}

// unexported constants.
const (
	engramClaudeSkills = fakeHome + "/.claude/engram/skills"
	engramPiSkills     = piAgentDir + "/engram/skills"
	legacyRouteNote    = "1036.2026-09-18.skill-route.md"
)

// unexported variables.
var (
	keyNamePattern  = `[a-z0-9][a-z0-9._-]{0,7}`
	noteSlugPattern = regexp.MustCompile(`^skill-[a-z0-9]+(-[a-z0-9]+)*$`)
)

// assertKeyShape checks one generated candidate's key against D3's shape:
// qualified, ending in the name, recognized by the key parser as exactly
// the candidate's removal form and scope, with a note-slug-shaped slug.
func assertKeyShape(rt *rapid.T, candidate cli.SkillCandidate, key string) {
	if !strings.Contains(key, ":") || !strings.HasSuffix(key, ":"+candidate.Name) {
		rt.Fatalf("key %q of %+v is not qualified or does not end in :%s", key, candidate, candidate.Name)
	}

	recognized, form, sourced, plugin, scopeID := cli.ExportParseSkillKey(key)
	wantForm, wantSourced, wantPlugin := expectedRemovalForm(candidate)

	if !recognized || form != wantForm || sourced != wantSourced || plugin != wantPlugin {
		rt.Fatalf("key %q of %+v parses as recognized=%v form=%q sourced=%v plugin=%q, want form=%q sourced=%v plugin=%q",
			key, candidate, recognized, form, sourced, plugin, wantForm, wantSourced, wantPlugin)
	}

	if (form == "") == (plugin == "") {
		rt.Fatalf("key %q maps to form %q and plugin %q, not exactly one", key, form, plugin)
	}

	if scopeID != candidate.ScopeID {
		rt.Fatalf("key %q parses to scope %q, want %q", key, scopeID, candidate.ScopeID)
	}

	if !noteSlugPattern.MatchString(cli.SkillKeySlug(key)) {
		rt.Fatalf("slug %q of key %q has the wrong shape", cli.SkillKeySlug(key), key)
	}
}

// drawKeyCandidate draws a candidate of any D3 source with a colon-free
// name (commands may have namespace segments), sometimes placed under
// engram's installed skills folder, which never changes its key.
func drawKeyCandidate(rt *rapid.T, index int) cli.SkillCandidate {
	label := func(what string) string { return fmt.Sprintf("%s%d", what, index) }

	name := rapid.StringMatching(keyNamePattern).Draw(rt, label("name"))
	pkg := "pi-pkg:" + rapid.StringMatching(`[a-z][a-z0-9/._-]{0,6}`).Draw(rt, label("pkg"))
	plugin := "plugin:" + rapid.StringMatching(`[a-z][a-z0-9-]{0,6}`).
		Filter(func(name string) bool { return !cli.IsReservedPluginNameForTest(name) }).
		Draw(rt, label("plugin"))
	project := "project:" + rapid.SampledFrom([]string{"github.com/a/b", "gitlab.com/a/b", "local/b"}).
		Draw(rt, label("project"))

	skill, command, prompt := cli.SkillSourceKindSkill, cli.SkillSourceKindCommand, cli.SkillSourceKindPrompt

	shapes := []cli.SkillCandidate{
		{ScopeID: cli.SkillScopeClaudeUser, Kind: skill},
		{ScopeID: cli.SkillScopeClaudeCmd, Kind: command},
		{ScopeID: cli.SkillScopeSynced, Kind: skill},
		{ScopeID: cli.SkillScopePiUser, Kind: skill},
		{ScopeID: cli.SkillScopeAgentsUser, Kind: skill},
		{ScopeID: cli.SkillScopePiPrompt, Kind: prompt},
		{ScopeID: cli.SkillScopePiSettings, SourceSegment: cli.SkillScopePiSettings, Kind: skill},
		{ScopeID: cli.SkillScopePiSettings, SourceSegment: cli.SkillScopePiSettings, Kind: prompt},
		{ScopeID: pkg, SourceSegment: pkg, Kind: skill},
		{ScopeID: pkg, SourceSegment: pkg, Kind: prompt},
		{ScopeID: plugin, Kind: skill},
		{ScopeID: plugin, Kind: command},
		{ScopeID: project, Kind: skill},
		{ScopeID: project, Kind: command},
		{ScopeID: project, SourceSegment: cli.SkillSegmentPi, Kind: skill},
		{ScopeID: project, SourceSegment: cli.SkillSegmentAgents, Kind: skill},
		{ScopeID: project, SourceSegment: cli.SkillScopePiPrompt, Kind: prompt},
		{ScopeID: project, SourceSegment: cli.SkillScopePiSettings, Kind: skill},
		{ScopeID: project, SourceSegment: cli.SkillScopePiSettings, Kind: prompt},
		{ScopeID: project, SourceSegment: pkg, Kind: skill},
		{ScopeID: project, SourceSegment: pkg, Kind: prompt},
	}

	candidate := rapid.SampledFrom(shapes).Draw(rt, label("shape"))
	if candidate.Kind == command && rapid.Bool().Draw(rt, label("namespaced")) {
		name = rapid.StringMatching(keyNamePattern).Draw(rt, label("namespace")) + ":" + name
	}

	candidate.Name = name
	candidate.SourcePath = "/elsewhere/" + strconv.Itoa(index) + "/" + name

	if rapid.IntRange(0, 4).Draw(rt, label("owned")) == 0 && candidate.Kind == skill {
		root := rapid.SampledFrom([]string{engramClaudeSkills, engramPiSkills}).Draw(rt, label("root"))
		candidate.SourcePath = root + "/" + name + "/SKILL.md"
	}

	return candidate
}

// expectedRemovalForm is the test's own oracle for the removal form a
// generated candidate's scope gives its key (design D3, D5).
func expectedRemovalForm(candidate cli.SkillCandidate) (cli.SkillRootForm, bool, string) {
	fixed := map[string]cli.SkillRootForm{
		cli.SkillScopeClaudeUser: cli.SkillRootFormClaudeUser, cli.SkillScopeClaudeCmd: cli.SkillRootFormClaudeCmd,
		cli.SkillScopePiUser: cli.SkillRootFormPiUser, cli.SkillScopeAgentsUser: cli.SkillRootFormAgentsUser,
		cli.SkillScopePiPrompt: cli.SkillRootFormPiPrompt,
	}

	scope := candidate.ScopeID

	switch {
	case fixed[scope] != "":
		return fixed[scope], false, ""
	case scope == cli.SkillScopeSynced:
		return cli.SkillRootFormSynced, true, ""
	case scope == cli.SkillScopePiSettings:
		return cli.SkillRootFormPiSettings, true, ""
	case strings.HasPrefix(scope, "pi-pkg:"):
		return cli.SkillRootFormPiPkg, true, ""
	case strings.HasPrefix(scope, "project:"):
		return cli.SkillRootFormProject, true, ""
	default:
		return "", false, strings.TrimPrefix(scope, "plugin:")
	}
}

// legacySkillNote renders a skill note shaped like the six legacy vault
// notes (1036 route … 1068 recall): a runbook with skill_hash, no skill_key.
func legacySkillNote(luhmann, name, hash string) string {
	return "---\ntype: runbook\ntier: L2\nsituation: s\ndone_when: d\nluhmann: \"" + luhmann +
		"\"\ncreated: \"2026-09-18\"\nsource: 'skill registration: agent-instructions/skills/" + name +
		"/SKILL.md'\nskill_hash: " + hash + "\n---\n\nbody\n"
}

func resolvedKeys(resolved cli.ResolvedSkillSources) []string {
	keys := make([]string, 0, len(resolved.Candidates))
	for _, candidate := range resolved.Candidates {
		keys = append(keys, candidate.Key)
	}

	return keys
}
