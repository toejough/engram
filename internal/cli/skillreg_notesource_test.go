package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestAdoptSkillNote_QualifiedKeyStampsKeySlugAndSource covers `--adopt
// <key>=<note-ref>` with a source-qualified key: the note is renamed to the
// key-derived slug and carries skill_key, skill_source and a preamble naming
// the real file.
func TestAdoptSkillNote_QualifiedKeyStampsKeySlugAndSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	source := pluginSkillSource("6.4.1", "# Brainstorming\n")

	err := cli.AdoptSkillNote(t.Context(), "/vault", source, "1049", skillAdoptDepsFor(vault), &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	adopted, found := vault.get("1049.2026-09-21.skill-superpowers-brainstorming.md")
	g.Expect(found).To(BeTrue())

	doc := parseSkillAcceptFrontmatter(g, adopted)
	g.Expect(doc.SkillKey).To(Equal("superpowers:brainstorming"))
	g.Expect(doc.SkillSource).To(Equal(source.SkillSource))
	g.Expect(doc.Pending).To(BeFalse())
	g.Expect(skillNoteBodyOf(adopted)).To(Equal(preambleNaming(source.SkillSource) + "\n# Brainstorming\n"))
}

// TestNewSkillNoteSource covers design D8's inputs: every source, engram's
// installed skills included, records its `~`-relative resolved path,
// through the home or its resolved form, and a path outside both homes
// stays absolute.
func TestNewSkillNoteSource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		candidate   cli.SkillCandidate
		homes       []string
		wantSource  string
		wantKeyName string
	}{
		{
			name:        "engram's installed skill",
			candidate:   offerCand("claude:route", cli.SkillScopeClaudeUser, engramClaudeSkills+"/route/SKILL.md", "r"),
			homes:       []string{fakeHome},
			wantSource:  "~/.claude/engram/skills/route/SKILL.md",
			wantKeyName: "claude:route",
		},
		{
			name:        "user skill",
			candidate:   offerCand("c4", cli.SkillScopeClaudeUser, fakeHome+"/.claude/skills/c4/SKILL.md", "c"),
			homes:       []string{fakeHome},
			wantSource:  "~/.claude/skills/c4/SKILL.md",
			wantKeyName: "c4",
		},
		{
			name: "source under the resolved home",
			candidate: offerCand("anthropic-skills:pdf", cli.SkillScopeSynced,
				"/real/joe/.claude/skills/synced/b1/pdf/SKILL.md", "p"),
			homes:       []string{"/links/joe", "/real/joe"},
			wantSource:  "~/.claude/skills/synced/b1/pdf/SKILL.md",
			wantKeyName: "anthropic-skills:pdf",
		},
		{
			name: "source outside every home",
			candidate: offerCand("project:local/x:cmd:go", "project:local/x",
				"/work/x/.claude/commands/go.md", "g"),
			homes:       []string{fakeHome, ""},
			wantSource:  "/work/x/.claude/commands/go.md",
			wantKeyName: "project:local/x:cmd:go",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			source := cli.NewSkillNoteSource(testCase.candidate, testCase.homes...)

			g.Expect(source.Key).To(Equal(testCase.wantKeyName))
			g.Expect(source.SkillSource).To(Equal(testCase.wantSource))
			g.Expect(source.Content).To(Equal(testCase.candidate.Content))
		})
	}
}

// TestRefreshSkill_FollowsPluginVersionBump covers "Refresh follows a plugin
// version bump": skill_source and the preamble name the new version's path.
func TestRefreshSkill_FollowsPluginVersionBump(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1100.2026-09-26.skill-superpowers-brainstorming"
	oldSource := pluginSkillSource("6.4.1", "# Brainstorming v1\n")
	vault.put(basename+".md", keyedSkillNoteFixture(oldSource))

	newSource := pluginSkillSource("6.5.0", "# Brainstorming v2\n")

	err := cli.RefreshSkill(t.Context(), "/vault", newSource, basename, skillAcceptDeps(vault), &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	updated, _ := vault.get(basename + ".md")
	doc := parseSkillAcceptFrontmatter(g, updated)

	g.Expect(doc.SkillKey).To(Equal("superpowers:brainstorming"))
	g.Expect(doc.SkillSource).To(HaveSuffix("/superpowers/6.5.0/skills/brainstorming/SKILL.md"))
	g.Expect(doc.SkillHash).To(Equal(cli.SkillContentHash(newSource.Content)))
	g.Expect(doc.Pending).To(BeTrue())
	g.Expect(skillNoteBodyOf(updated)).To(Equal(preambleNaming(newSource.SkillSource) + "\n# Brainstorming v2\n"))
}

// TestRegisterSkill_EngramInstalledPreambleNamesDeployedFile asserts, for
// the six engram skills behind the real notes (route, please, curate,
// write-memory, learn, recall), that a copy read through ~/.claude/skills
// gives the key-derived slug, skill_key, skill_source, and a preamble and
// `source:` naming the deployed file with no special case (design D8).
func TestRegisterSkill_EngramInstalledPreambleNamesDeployedFile(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"route", "please", "curate", "write-memory", "learn", "recall"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			content := "---\nname: " + name + "\ndescription: d\n---\n\nbody of " + name + "\n"
			written := registerForTest(g, installedEngramSkill(name, []byte(content)))

			g.Expect(written.path).To(Equal("/vault/1.2026-09-25.skill-claude-" + name + ".md"))
			g.Expect(skillNoteBodyOf(written.content)).To(Equal(deployedPreamble(name) + "\n" + content))
			g.Expect(written.content).To(ContainSubstring(
				"source: 'skill registration: ~/.claude/engram/skills/" + name + "/SKILL.md'\n"))

			doc := parseSkillAcceptFrontmatter(g, written.content)
			g.Expect(doc.SkillKey).To(Equal("claude:" + name))
			g.Expect(doc.SkillSource).To(Equal("~/.claude/engram/skills/" + name + "/SKILL.md"))
			g.Expect(doc.Pending).To(BeTrue())
		})
	}
}

// TestRegisterSkill_NonEngramSourcesNamePathAndKeySlug covers the plugin
// skill, project command and Pi prompt registrations: the key-derived slug,
// skill_key, skill_source and a preamble naming the `~`-relative file.
func TestRegisterSkill_NonEngramSourcesNamePathAndKeySlug(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		source   cli.SkillNoteSource
		wantSlug string
	}{
		{
			name:     "plugin skill",
			source:   pluginSkillSource("6.4.1", "# Brainstorming\n"),
			wantSlug: "skill-superpowers-brainstorming",
		},
		{
			name: "project command",
			source: cli.SkillNoteSource{
				Key:         "project:github.com/toejough/engram:cmd:opsx:apply",
				SkillSource: "~/repos/engram/.claude/commands/opsx/apply.md",
				Content:     []byte("---\ndescription: apply\nargument-hint: <change>\n---\n\nApply it.\n"),
			},
			wantSlug: "skill-project-github-com-toejough-engram-cmd-opsx-apply",
		},
		{
			name: "Pi prompt template",
			source: cli.SkillNoteSource{
				Key:         "pi-prompt:review",
				SkillSource: "~/.pi/agent/prompts/review.md",
				Content:     []byte("Review $1.\n"),
			},
			wantSlug: "skill-pi-prompt-review",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			written := registerForTest(g, testCase.source)

			g.Expect(written.path).To(Equal("/vault/1.2026-09-25." + testCase.wantSlug + ".md"))
			g.Expect(skillNoteBodyOf(written.content)).To(Equal(
				preambleNaming(testCase.source.SkillSource) + "\n" + string(testCase.source.Content)))

			doc := parseSkillAcceptFrontmatter(g, written.content)
			g.Expect(doc.SkillKey).To(Equal(testCase.source.Key))
			g.Expect(doc.SkillSource).To(Equal(testCase.source.SkillSource))
		})
	}
}

// TestRegisterSkill_SkillSourceRoundTripsThroughRemovalEligibility (property):
// a note registered from a candidate under the home, or under its resolved
// form when the home is a symlink, records a `~`-relative skill_source that
// the removal-eligibility check expands against the unresolved home back
// into the candidate's own read root, so the note is offered for removal
// once its key is gone.
func TestRegisterSkill_SkillSourceRoundTripsThroughRemovalEligibility(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		home := "/home/" + rapid.StringMatching(`[a-z]{1,6}`).Draw(rt, "user")
		resolvedHome := home

		if rapid.Bool().Draw(rt, "symlinkedHome") {
			resolvedHome = "/private" + home
		}

		bucket := rapid.StringMatching(`[a-z0-9]{1,6}`).Draw(rt, "bucket")
		name := rapid.StringMatching(`[a-z][a-z0-9-]{0,8}`).Draw(rt, "name")
		rootRel := "/.claude/skills/synced/" + bucket
		candidate := offerCand("anthropic-skills:"+name, cli.SkillScopeSynced,
			resolvedHome+rootRel+"/"+name+"/SKILL.md", "content of "+name)

		source := cli.NewSkillNoteSource(candidate, home, resolvedHome)
		if !strings.HasPrefix(source.SkillSource, "~/") {
			rt.Fatalf("skill_source %q is not ~-relative", source.SkillSource)
		}

		vault := newSkillAcceptFixtureVault()

		var (
			writtenPath    string
			writtenContent []byte
		)

		registerErr := cli.RegisterSkill(t.Context(), "/vault", "personal", source,
			registerSkillLearnDeps(vault, &writtenPath, &writtenContent), &bytes.Buffer{})
		if registerErr != nil {
			rt.Fatalf("RegisterSkill: %v", registerErr)
		}

		names, _ := vault.ListMD("/vault")

		var sources cli.ResolvedSkillSources

		sources.Roots = []cli.ScannedRoot{{
			Path: home + rootRel, Resolved: resolvedHome + rootRel, Scanned: true,
			Form: cli.SkillRootFormSynced, KeyPrefixes: []string{"anthropic-skills:"},
		}}

		comparison, compareErr := cli.CompareSkillOffers(cli.SkillOfferInput{
			Vault: "/vault", Names: names, ReadFile: vault.ReadFile, Home: home, Sources: sources,
		})
		if compareErr != nil {
			rt.Fatalf("CompareSkillOffers: %v", compareErr)
		}

		if len(comparison.Offers) != 1 || comparison.Offers[0].Kind != cli.SkillOfferRemove ||
			comparison.Offers[0].Key != candidate.Key {
			rt.Fatalf("want one removal offer for %s, got %+v (skill_source %q)",
				candidate.Key, comparison.Offers, source.SkillSource)
		}
	})
}

// TestRunSkillRegistration_AcceptLeavesSourceFilesByteIdentical covers the
// MODIFIED "no engram-specific metadata" scenarios over fixture offers: a
// plugin skill, a project command and a Pi prompt are registered with the
// key-derived slugs, and every write lands in the vault — nothing is written
// to or beside a source file, and the source bytes are unchanged.
func TestRunSkillRegistration_AcceptLeavesSourceFilesByteIdentical(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	deps := skillRegistrationDepsFor(vault, nil)

	var writes []string

	recordWrites(&deps, vault, &writes)

	skill := offerCand("superpowers:brainstorming", "plugin:superpowers",
		fakeHome+"/.claude/plugins/cache/m/superpowers/6.4.1/skills/brainstorming/SKILL.md",
		"---\nname: brainstorming\ndescription: d\n---\n\nThink.\n")
	command := offerCand("project:github.com/toejough/engram:cmd:opsx:apply", "project:github.com/toejough/engram",
		"/work/engram/.claude/commands/opsx/apply.md", "---\ndescription: a\nargument-hint: <c>\n---\n\nApply.\n")
	command.Name, command.Kind = "opsx:apply", cli.SkillSourceKindCommand
	prompt := offerCand("pi-prompt:review", cli.SkillScopePiPrompt,
		fakeHome+"/.pi/agent/prompts/review.md", "Review $1.\n")
	prompt.Kind = cli.SkillSourceKindPrompt

	var sources cli.ResolvedSkillSources

	sources.Candidates = []cli.SkillCandidate{skill, command, prompt}
	before := make([]string, 0, len(sources.Candidates))

	for _, candidate := range sources.Candidates {
		before = append(before, string(candidate.Content))
	}

	err := cli.ExportAnswerSkillSources(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: fakeHome, Accept: []string{"*"},
	}, sources, deps, &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	for index, candidate := range sources.Candidates {
		g.Expect(string(candidate.Content)).To(Equal(before[index]))
	}

	for _, path := range writes {
		g.Expect(path).To(HavePrefix("/vault/"))
	}

	names, _ := vault.ListMD("/vault")
	g.Expect(names).To(ConsistOf(
		"1.2026-09-25.skill-superpowers-brainstorming.md",
		"1.2026-09-25.skill-project-github-com-toejough-engram-cmd-opsx-apply.md",
		"1.2026-09-25.skill-pi-prompt-review.md",
	))

	commandNote, _ := vault.get("1.2026-09-25.skill-project-github-com-toejough-engram-cmd-opsx-apply.md")
	g.Expect(skillNoteBodyOf(commandNote)).To(Equal(
		preambleNaming("/work/engram/.claude/commands/opsx/apply.md") + "\n" + string(command.Content)))
}

// TestRunSkillRegistration_AdoptOfAConflictedKeyErrors: a key whose copies
// differ (design D4) has no single source to mirror, so adopting it is
// refused and the note is left untouched.
func TestRunSkillRegistration_AdoptOfAConflictedKeyErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	var sources cli.ResolvedSkillSources

	sources.Candidates = []cli.SkillCandidate{
		offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/a/foo/SKILL.md", "one"),
		offerCand("pi:foo", cli.SkillScopePiUser, piUserRoot+"/b/foo/SKILL.md", "two"),
	}

	err := cli.ExportAnswerSkillSources(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", Home: fakeHome, Adopt: map[string]string{"pi:foo": "1049"},
	}, sources, skillRegistrationDepsFor(vault, nil), &bytes.Buffer{})
	g.Expect(err).To(MatchError(ContainSubstring("key conflict")))

	_, untouched := vault.get("1049.2026-09-21.curate-review-pending-offers.md")
	g.Expect(untouched).To(BeTrue())
}

// TestRunSkillRegistration_AdoptTakesKeyOfAScannedSource covers `--adopt
// <key>=<note-ref>` over fixture sources: the key names a scanned candidate,
// whose file, key and source the adopted note then mirrors.
func TestRunSkillRegistration_AdoptTakesKeyOfAScannedSource(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1049.2026-09-21.curate-review-pending-offers.md", curatePromotedNoteFixture())

	var sources cli.ResolvedSkillSources

	sources.Candidates = []cli.SkillCandidate{offerCand("superpowers:brainstorming", "plugin:superpowers",
		fakeHome+"/.claude/plugins/cache/m/superpowers/6.4.1/skills/brainstorming/SKILL.md", "# B\n")}

	var stdout bytes.Buffer

	err := cli.ExportAnswerSkillSources(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: fakeHome,
		Adopt: map[string]string{"superpowers:brainstorming": "1049"},
	}, sources, skillRegistrationDepsFor(vault, nil), &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	adopted, found := vault.get("1049.2026-09-21.skill-superpowers-brainstorming.md")
	g.Expect(found).To(BeTrue())

	doc := parseSkillAcceptFrontmatter(g, adopted)
	g.Expect(doc.SkillKey).To(Equal("superpowers:brainstorming"))
	g.Expect(doc.SkillSource).To(Equal("~/.claude/plugins/cache/m/superpowers/6.4.1/skills/brainstorming/SKILL.md"))
	g.Expect(stdout.String()).NotTo(ContainSubstring("awaiting an answer"))
}

// TestRunSkillRegistration_Property_SkillsDirKeysClaudeUserAndWritesNothing
// is design D9's invariant over any preview dir and vault: every offer a
// `--skills-dir` run lists is a register or refresh keyed `claude:<n>` under
// the single @claude-user header, notes whose hash is current get no offer,
// no removal is listed even for orphaned notes of every key form (one
// recording a skill_source inside the preview dir), and nothing in the vault
// changes, even with a terminal answering yes.
func TestRunSkillRegistration_Property_SkillsDirKeysClaudeUserAndWritesNothing(t *testing.T) {
	t.Parallel()

	const previewDir = "/preview/skills"

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		names := rapid.SliceOfNDistinct(rapid.StringMatching(`[a-m][a-z0-9]{0,5}`), 1, 5, rapid.ID[string]).
			Draw(rt, "names")

		vault := newSkillAcceptFixtureVault()
		fsys := newFakeSkillFS()
		lines := make(map[string]string, len(names))

		for index, name := range names {
			content := "procedure " + name
			fsys.file(previewDir+"/"+name+"/SKILL.md", content)

			key := "claude:" + name
			note := fmt.Sprintf("%d.2026-09-26.%s.md", skillsDirNoteLuhmannBase+index, cli.SkillKeySlug(key))
			line := "  would offer: %s " + key + " (" + previewDir + "/" + name + "/SKILL.md)\n"

			switch rapid.SampledFrom([]string{"none", "stale", "current"}).Draw(rt, "note-"+name) {
			case "none":
				lines[key] = fmt.Sprintf(line, "register")
			case "stale":
				vault.put(note, keyedSkillNote(cli.SkillContentHash([]byte("old")), key))
				lines[key] = fmt.Sprintf(line, "refresh")
			default:
				vault.put(note, keyedSkillNote(cli.SkillContentHash([]byte(content)), key))
			}
		}

		for index, key := range []string{
			"claude:zgone", "claude:cmd:zgone", "pi:zgone", "agents:zgone", "anthropic-skills:zgone",
			"superpowers:zgone", "project:repo:zgone",
		} {
			vault.put(fmt.Sprintf("%d.2026-09-26.%s.md", skillsDirOrphanLuhmannBase+index, cli.SkillKeySlug(key)),
				strings.Replace(keyedSkillNote("gone-hash", key), "~/x/SKILL.md", previewDir+"/zgone/SKILL.md", 1))
		}

		before := maps.Clone(vault.files)

		deps := skillRegistrationDepsFor(vault, fsys)
		deps.Getwd = failingGetwd(g)
		deps.IsTerminal = func() bool { return true }
		deps.Stdin = strings.NewReader(strings.Repeat("y\n", len(names)+skillsDirOrphanCount))

		var writes []string

		recordWrites(&deps, vault, &writes)

		var stdout bytes.Buffer

		err := cli.RunSkillRegistration(context.Background(), cli.SkillRegistrationArgs{
			Vault: "/vault", VaultName: "personal", Home: fakeHome, PreviewDirs: []string{previewDir},
		}, deps, &stdout)
		g.Expect(err).NotTo(HaveOccurred())

		var want strings.Builder

		if len(lines) > 0 {
			_, _ = fmt.Fprintf(&want, "@claude-user (%d)\n", len(lines))
		}

		for _, key := range slices.Sorted(maps.Keys(lines)) {
			want.WriteString(lines[key])
		}

		g.Expect(stdout.String()).To(Equal(want.String()))
		g.Expect(writes).To(BeEmpty())
		g.Expect(vault.files).To(Equal(before))
	})
}

// TestRunSkillRegistration_SkillsDirIsAReadOnlyPreview covers design D9 and
// "Skills-dir runs are preview-only": several dirs replace the default set
// and are scanned with Claude-user rules (a symlinked skill dir is found,
// keys are `claude:<n>`); the run lists offers under a scope header, prompts
// nothing, writes nothing (a stale note is not refreshed, so its preamble is
// never rewritten), and lists no removal offer.
func TestRunSkillRegistration_SkillsDirIsAReadOnlyPreview(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("1036.2026-09-18.skill-claude-route.md", keyedSkillNote(cli.SkillContentHash([]byte("old")), "claude:route"))
	vault.put("1053.2026-09-21.skill-claude-write-memory.md", keyedSkillNote("gone-hash", "claude:write-memory"))

	fsys := newFakeSkillFS().
		file("/checkout/agent-instructions/skills/route/SKILL.md", "route v2").
		file("/elsewhere/real/c4/SKILL.md", "c4").
		link("/extra/skills/c4", "/elsewhere/real/c4").
		file("/extra/skills/synced/b/pdf/SKILL.md", "pdf")

	deps := skillRegistrationDepsFor(vault, fsys)
	deps.Getwd = failingGetwd(g)
	deps.IsTerminal = func() bool { return true }
	deps.Stdin = strings.NewReader(strings.Repeat("y\n", 3))

	var writes []string

	recordWrites(&deps, vault, &writes)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: fakeHome,
		PreviewDirs: []string{"/checkout/agent-instructions/skills", "/extra/skills"},
	}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(Equal("@claude-user (2)\n" +
		"  would offer: register claude:c4 (/elsewhere/real/c4/SKILL.md)\n" +
		"  would offer: refresh claude:route (/checkout/agent-instructions/skills/route/SKILL.md)\n"))
	g.Expect(writes).To(BeEmpty())
}

// TestRunSkillRegistration_SkillsDirRefusesAnswersBeforeScanning: with
// `--skills-dir`, each of `--accept`, `--decline` and `--adopt` is refused
// with an error before any directory is scanned or the vault is read.
func TestRunSkillRegistration_SkillsDirRefusesAnswersBeforeScanning(t *testing.T) {
	t.Parallel()

	cases := map[string]cli.SkillRegistrationArgs{
		"accept":  {Accept: []string{"route"}},
		"decline": {Decline: []string{"route"}},
		"adopt":   {Adopt: map[string]string{"route": "1036"}},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			deps := skillRegistrationDepsFor(vault, untouchableSkillFS{g: g})
			deps.Getwd = failingGetwd(g)
			deps.ListMD = func(string) ([]string, error) {
				g.Expect("the vault").To(BeEmpty(), "read before the refusal")

				return nil, nil
			}

			args.Vault = "/vault"
			args.PreviewDirs = []string{"/checkout/agent-instructions/skills"}

			err := cli.RunSkillRegistration(t.Context(), args, deps, &bytes.Buffer{})
			g.Expect(err).To(MatchError(cli.ErrSkillsDirReadOnlyForTest))
		})
	}
}

// unexported constants.
const (
	// skillsDirNoteLuhmannBase and skillsDirOrphanLuhmannBase number the
	// property test's preview-dir notes and orphaned notes apart.
	skillsDirNoteLuhmannBase   = 2000
	skillsDirOrphanCount       = 7
	skillsDirOrphanLuhmannBase = 3000
)

// unexported types.

// untouchableSkillFS fails the test on any read: the refusal must come
// before scanning.
type untouchableSkillFS struct{ g Gomega }

func (u untouchableSkillFS) Lstat(path string) (fs.FileInfo, error) { return nil, u.fail(path) }

func (u untouchableSkillFS) ReadDir(path string) ([]fs.DirEntry, error) { return nil, u.fail(path) }

func (u untouchableSkillFS) ReadFile(path string) ([]byte, error) { return nil, u.fail(path) }

func (u untouchableSkillFS) Readlink(path string) (string, error) { return "", u.fail(path) }

func (u untouchableSkillFS) fail(path string) error {
	u.g.Expect(path).To(BeEmpty(), "scanned before the refusal")

	return fs.ErrPermission
}

// writtenNote is one note RegisterSkill wrote.
type writtenNote struct {
	path    string
	content string
}

// deployedPreamble is the preamble of an engram skill installed under the
// Claude Code engram skills folder, spelled out byte for byte (design D8).
func deployedPreamble(name string) string {
	return "> Mirrors skill `~/.claude/engram/skills/" + filepath.Join(name, "SKILL.md") + "`.\n"
}

// failingGetwd is a Getwd that fails the test: a --skills-dir run never
// resolves the default source set, so it never asks for the working
// directory.
func failingGetwd(g Gomega) func() (string, error) {
	return func() (string, error) {
		g.Expect("the working directory").To(BeEmpty(), "the default source set was resolved")

		return "", fs.ErrPermission
	}
}

// unexported functions.

// installedEngramSkill is an engram skill deployed under
// ~/.claude/engram/skills and read through ~/.claude/skills, as
// NewSkillNoteSource builds it.
func installedEngramSkill(name string, content []byte) cli.SkillNoteSource {
	return cli.SkillNoteSource{
		Key:         "claude:" + name,
		SkillSource: "~/.claude/engram/skills/" + name + "/SKILL.md", Content: content,
	}
}

// keyedSkillNoteFixture renders a registered, curated skill note for source.
func keyedSkillNoteFixture(source cli.SkillNoteSource) string {
	return "---\n" +
		"type: runbook\n" +
		"situation: brainstorming a design\n" +
		"done_when: a design is agreed\n" +
		"luhmann: \"1100\"\n" +
		"created: 2026-09-26\n" +
		"source: s\n" +
		"user: joe\n" +
		"vault: personal\n" +
		"skill_hash: " + cli.SkillContentHash([]byte("stale")) + "\n" +
		"skill_key: " + source.Key + "\n" +
		"skill_source: " + source.SkillSource + "\n" +
		"---\n\n" +
		preambleNaming(source.SkillSource) + "\n" + string(source.Content)
}

// pluginSkillSource is superpowers' brainstorming skill at a plugin version.
func pluginSkillSource(version, content string) cli.SkillNoteSource {
	return cli.SkillNoteSource{
		Key: "superpowers:brainstorming",
		SkillSource: "~/.claude/plugins/cache/claude-plugins-official/superpowers/" + version +
			"/skills/brainstorming/SKILL.md",
		Content: []byte(content),
	}
}

// preambleNaming is the expected preamble line for any source (design D8):
// exactly “> Mirrors skill `<skill_source>`.“, with no edit-location clause.
func preambleNaming(path string) string {
	return "> Mirrors skill `" + path + "`.\n"
}

// recordWrites wraps every vault write dep of deps to record its path.
func recordWrites(deps *cli.SkillRegistrationDeps, vault *skillAcceptFixtureVault, writes *[]string) {
	record := func(path string, data []byte) error {
		*writes = append(*writes, path)

		return vault.WriteFile(path, data)
	}

	deps.Accept.Write = record
	deps.Adopt.Rename.WriteFile = record
	deps.Learn.WriteNew = record
	deps.Learn.WriteSidecar = record
}

// registerForTest runs RegisterSkill for source over a fresh fixture vault
// and returns the note it wrote.
func registerForTest(g Gomega, source cli.SkillNoteSource) writtenNote {
	vault := newSkillAcceptFixtureVault()

	var (
		writtenPath    string
		writtenContent []byte
	)

	err := cli.RegisterSkill(context.Background(), "/vault", "personal", source,
		registerSkillLearnDeps(vault, &writtenPath, &writtenContent), &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred())

	return writtenNote{path: writtenPath, content: string(writtenContent)}
}

// skillAdoptDepsFor composes SkillAdoptDeps over vault.
func skillAdoptDepsFor(vault *skillAcceptFixtureVault) cli.SkillAdoptDeps {
	return cli.SkillAdoptDeps{
		Lock:     noLock,
		Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
		Rename:   skillAcceptRenameDeps(vault),
		Embedder: skillAcceptFakeEmbedder{},
	}
}

// skillNoteBodyOf returns a note's body: everything after the frontmatter's
// closing fence and the blank line after it, less the one blank line the
// runbook renderer always ends a body with.
func skillNoteBodyOf(content string) string {
	_, body, _ := strings.Cut(strings.TrimPrefix(content, "---\n"), "\n---\n\n")

	return strings.TrimSuffix(body, "\n")
}
