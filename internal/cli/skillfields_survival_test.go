package cli_test

// Field-survival tests for the skill-note identity fields (vault-note-identity
// ADDED requirement "Skill-note identity fields SHALL survive every
// frontmatter rewrite"): skill_hash, skill_key and skill_source must come out
// of every non-registration frontmatter rewrite byte-unchanged. Note
// identity matches by skill_key with a legacy slug-remainder fallback, so a
// rewrite that silently dropped skill_key would re-key the note (e.g. `a:b`
// falling back to bare `a-b`) and let it capture an unrelated skill.
//
// One test per rewrite site (task 4.2); each compares the raw YAML block of
// every identity field (key line plus any continuation lines) before and
// after the rewrite.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestSkillIdentityFieldsSurvive_AmendClearPending covers `engram amend
// --clear-pending` end to end through the CLI target (amend.go
// applyRunbookAmend -> renderAmendedRunbook, the no-content-change branch).
func TestSkillIdentityFieldsSurvive_AmendClearPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, skillSurvivalNoteName)
	before := skillSurvivalNote()
	g.Expect(os.WriteFile(notePath, []byte(before), 0o600)).To(Succeed())

	stderr := executeForTest(t, []string{"engram", "amend", "--vault", vault, "--target", "1a", "--clear-pending"})
	g.Expect(stderr).To(BeEmpty())

	raw, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).NotTo(ContainSubstring("pending: true"))
	expectSkillIdentityUnchanged(g, before, string(raw))
}

// TestSkillIdentityFieldsSurvive_AmendClearPendingProperty is the property
// form of the clear-pending case: any skill_key/skill_source value, rendered
// the way yaml.v3 renders it, survives the amend re-marshal byte-unchanged.
func TestSkillIdentityFieldsSurvive_AmendClearPendingProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		key := skillKeyGen().Draw(rt, "skillKey")
		source := skillSourceGen().Draw(rt, "skillSource")
		before := skillSurvivalNoteWith(key, source)

		var (
			written []byte
			writes  int
		)

		pending := false
		args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Pending: &pending}

		var buf bytes.Buffer

		err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &buf)
		if err != nil {
			rt.Fatalf("amend: %v", err)
		}

		assertSkillIdentityUnchanged(rt, before, string(written))
	})
}

// TestSkillIdentityFieldsSurvive_AmendContentChanged covers amend's
// content-changed branch (renderAmendedRunbook rebuilding the body), the
// path an agent takes when authoring a registered skill note's situation.
func TestSkillIdentityFieldsSurvive_AmendContentChanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()

	var (
		written []byte
		writes  int
	)

	args := cli.AmendArgs{
		Vault: "/vault", Target: "1aa",
		Situation: "brainstorming a design before building", DoneWhen: "the design is approved",
	}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(writes).To(Equal(1))
	g.Expect(string(written)).To(ContainSubstring("situation: brainstorming a design before building"))
	expectSkillIdentityUnchanged(g, before, string(written))
}

// TestSkillIdentityFieldsSurvive_AmendKeyEndingInDashes pins the case the
// clear-pending property found: a skill_key ending in "---" must not be read
// as the frontmatter's closing delimiter (resituate.go splitFrontmatter,
// shared by amend), which truncated the frontmatter and dropped the key.
func TestSkillIdentityFieldsSurvive_AmendKeyEndingInDashes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNoteWith("superpowers:odd---", skillSurvivalSource)

	var (
		written []byte
		writes  int
	)

	pending := false
	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Pending: &pending}

	var buf bytes.Buffer

	err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &buf)
	g.Expect(err).NotTo(HaveOccurred())
	expectSkillIdentityUnchanged(g, before, string(written))
}

// TestSkillIdentityFieldsSurvive_BackfillIdentity covers `engram update
// --backfill-identity` (identity_backfill.go): it stamps fact/feedback notes
// only, so a skill (runbook) note missing user:/vault: is never rewritten.
func TestSkillIdentityFieldsSurvive_BackfillIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(skillSurvivalNote(), "user: agent@example.com\nvault: personal\n", "", 1)
	fs := newSkillSurvivalFS(map[string]string{skillSurvivalNoteName: before})

	deps := cli.IdentityDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     fs.listMD,
		ReadFile:   fs.readFile,
		WriteFile:  fs.writeFile,
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}

	_, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())
	expectSkillIdentityUnchanged(g, before, fs.content(skillSurvivalNoteName))
}

// TestSkillIdentityFieldsSurvive_ClearRemovedVocabTerms covers vocab refit's
// term removal (vocab_commands.go clearRemovedTermsFromNote), which rewrites
// the skill note's tags: block.
func TestSkillIdentityFieldsSurvive_ClearRemovedVocabTerms(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	fs := newSkillSurvivalFS(map[string]string{skillSurvivalNoteName: before})

	deps := cli.VocabDeps{
		ListMD: fs.listMD, ReadFile: fs.readFile, WriteFile: fs.writeFile,
		LogWarning: func(string, ...any) {},
	}

	err := cli.ExportClearRemovedTermsFromMembers(deps, "/vault", []string{"old-term"})
	g.Expect(err).NotTo(HaveOccurred())

	after := fs.content(skillSurvivalNoteName)
	g.Expect(after).NotTo(ContainSubstring("vocab/old-term"), "the rewrite must actually have run")
	expectSkillIdentityUnchanged(g, before, after)
}

// TestSkillIdentityFieldsSurvive_RemoveDeletedNoteReferences covers vocab
// refit's reference scrub (vocab_apply.go removeNoteReferences), which drops
// a supersedes: entry naming a deleted definition note.
func TestSkillIdentityFieldsSurvive_RemoveDeletedNoteReferences(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()

	after, changed := cli.ExportRemoveNoteReferences(before, map[string]bool{skillSurvivalSupersededName: true})
	g.Expect(changed).To(BeTrue(), "the scrub must actually have run")
	g.Expect(after).NotTo(ContainSubstring(skillSurvivalSupersededName))
	expectSkillIdentityUnchanged(g, before, after)
}

// TestSkillIdentityFieldsSurvive_ReparentRenamesSkillNote covers Luhmann
// reparenting renaming the skill note itself (luhmann_reparent.go
// renameOneNote -> rewriteLuhmannIDField).
func TestSkillIdentityFieldsSurvive_ReparentRenamesSkillNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	fixture := newReparentFixture(map[string]string{skillSurvivalNoteName: before})
	oldBasename := strings.TrimSuffix(skillSurvivalNoteName, ".md")
	newBasename := "3.2026-01-01.skill-superpowers-brainstorming"

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", map[string]string{oldBasename: newBasename})
	g.Expect(err).NotTo(HaveOccurred())

	after := string(fixture.written["/vault/"+newBasename+".md"])
	g.Expect(after).To(ContainSubstring(`luhmann: "3"`), "the rename must actually have rewritten the note")
	expectSkillIdentityUnchanged(g, before, after)
}

// TestSkillIdentityFieldsSurvive_Resituate covers `engram resituate`
// (resituate.go resituateContent): it re-renders fact/feedback notes only, so
// a skill (runbook) note is rejected and never written.
func TestSkillIdentityFieldsSurvive_Resituate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	writes := 0

	deps := cli.ResituateDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: skillSurvivalNoteName, LuhmannID: "1a"}}, nil
		},
		Read:  func(string) ([]byte, error) { return []byte(before), nil },
		Write: func(string, []byte) error { writes++; return nil },
		Now:   func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}

	var buf bytes.Buffer

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{Vault: "/vault", Note: "1a", Situation: "new"}, deps, &buf)
	g.Expect(err).To(HaveOccurred())
	g.Expect(writes).To(Equal(0), "resituate must never write a skill note")
}

// TestSkillIdentityFieldsSurvive_StripLegacyVocabChannel covers `engram
// update --regen-vocab`'s legacy cleanup (vocab_regen.go
// stripLegacyVocabChannel), which removes a vocab: frontmatter line.
func TestSkillIdentityFieldsSurvive_StripLegacyVocabChannel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := strings.Replace(skillSurvivalNote(), "tier: L2\n", "tier: L2\nvocab: old-term\n", 1)

	after, changed := cli.ExportStripLegacyVocabChannel(before)
	g.Expect(changed).To(BeTrue(), "the strip must actually have run")
	g.Expect(after).NotTo(ContainSubstring("vocab: old-term"))
	expectSkillIdentityUnchanged(g, before, after)
}

// TestSkillIdentityFieldsSurvive_VocabAssignment covers vocab term
// assignment (vocab.go WriteVocabAssignment -> rewriteTagsFrontmatterSplit),
// shared by the post-write assignment in learn/amend/resituate and by `engram
// vocab` retagging (vocab_commands.go assignVocabToNote).
func TestSkillIdentityFieldsSurvive_VocabAssignment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()

	after := cli.ExportWriteVocabAssignment(before, []string{"new-term", "other-term"})
	g.Expect(after).To(ContainSubstring("vocab/new-term"), "the assignment must actually have run")
	expectSkillIdentityUnchanged(g, before, after)
}

// TestSkillIdentityFieldsSurvive_VocabAssignmentAndWikilinkProperty is the
// property form of the line-based rewrites: for any skill_key/skill_source
// value, vocab tag assignment followed by a frontmatter wikilink rewrite
// leaves all three identity blocks byte-unchanged.
func TestSkillIdentityFieldsSurvive_VocabAssignmentAndWikilinkProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		key := skillKeyGen().Draw(rt, "skillKey")
		source := skillSourceGen().Draw(rt, "skillSource")
		before := skillSurvivalNoteWith(key, source)

		tagged := cli.ExportWriteVocabAssignment(before, []string{"new-term"})
		assertSkillIdentityUnchanged(rt, before, tagged)

		fixture := newReparentFixture(map[string]string{skillSurvivalNoteName: tagged})

		_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
			map[string]string{skillSurvivalLinkTarget: "4.2026-01-01.old-target"})
		if err != nil {
			rt.Fatalf("rename-and-rewrite: %v", err)
		}

		assertSkillIdentityUnchanged(rt, before, string(fixture.written["/vault/"+skillSurvivalNoteName]))
	})
}

// TestSkillIdentityFieldsSurvive_VocabTagDefinitions covers `engram vocab
// tag-definitions` (vocab_commands.go runVocabTagDefinitions), which rewrites
// definition notes only — a skill note is never written.
func TestSkillIdentityFieldsSurvive_VocabTagDefinitions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	fs := newSkillSurvivalFS(map[string]string{skillSurvivalNoteName: before})

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: fs.listMD, ReadFile: fs.readFile, WriteFile: fs.writeFile,
		LogWarning: func(string, ...any) {},
	}

	var buf bytes.Buffer

	err := cli.ExportRunVocabTagDefinitions(t.Context(), "/vault", deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fs.writes).To(BeEmpty(), "tag-definitions must never write a skill note")
}

// TestSkillIdentityFieldsSurvive_VocabVersionKey covers the vocab_version
// rewrite (vocab_commands.go writeVocabVersionToFamilyNote ->
// rewriteVocabVersionKey), which targets the vocab family note only — a
// skill note alongside it is never written.
func TestSkillIdentityFieldsSurvive_VocabVersionKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const familyName = "5.2026-01-01.vocab-definition.md"

	fs := newSkillSurvivalFS(map[string]string{
		skillSurvivalNoteName: skillSurvivalNote(),
		familyName:            "---\ntype: fact\nvocab_version: \"1.0\"\ntags:\n    - vocab\n---\n\nFamily.\n",
	})

	err := cli.ExportWriteVocabVersionToFamilyNote("/vault", "1.1", fs.listMD, fs.readFile, fs.writeFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fs.writes).To(ConsistOf(filepath.Join("/vault", familyName)))
}

// TestSkillIdentityFieldsSurvive_WikilinkRewrite covers rename-and-rewrite
// of a wikilink inside a skill note's frontmatter and body
// (luhmann_reparent.go rewriteNoteReferences), when a note the skill note
// links to is renamed.
func TestSkillIdentityFieldsSurvive_WikilinkRewrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	before := skillSurvivalNote()
	fixture := newReparentFixture(map[string]string{
		skillSurvivalNoteName:           before,
		skillSurvivalLinkTarget + ".md": "---\ntype: fact\nluhmann: \"2\"\n---\n\nTarget.\n",
	})

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
		map[string]string{skillSurvivalLinkTarget: "4.2026-01-01.old-target"})
	g.Expect(err).NotTo(HaveOccurred())

	after := string(fixture.written["/vault/"+skillSurvivalNoteName])
	g.Expect(after).To(ContainSubstring("red_flags:\n    - skipped the checklist in [[4.2026-01-01.old-target]]"),
		"the frontmatter wikilink must actually have been rewritten")
	expectSkillIdentityUnchanged(g, before, after)
}

// unexported constants.
const (
	skillSurvivalHash           = "3f2a9c1d8e7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f"
	skillSurvivalKey            = "superpowers:brainstorming"
	skillSurvivalLinkTarget     = "2.2026-01-01.old-target"
	skillSurvivalNoteName       = "1a.2026-01-01.skill-superpowers-brainstorming.md"
	skillSurvivalSource         = "~/.claude/plugins/cache/superpowers/5.0.1/skills/brainstorming/SKILL.md"
	skillSurvivalSupersededName = "9.2026-01-01.deleted-def.md"
)

// unexported variables.
var (
	skillIdentityKeys = []string{"skill_hash", "skill_key", "skill_source"}
)

// skillSurvivalFS is an in-memory vault for the site tests: reads see the
// latest write, and every written path is recorded in order.
type skillSurvivalFS struct {
	files  map[string]string
	names  []string
	writes []string
}

func (fs *skillSurvivalFS) content(name string) string {
	return fs.files[filepath.Join("/vault", name)]
}

func (fs *skillSurvivalFS) listMD(string) ([]string, error) {
	return fs.names, nil
}

func (fs *skillSurvivalFS) readFile(path string) ([]byte, error) {
	data, ok := fs.files[path]
	if !ok {
		return nil, &testNotFoundError{path: path}
	}

	return []byte(data), nil
}

func (fs *skillSurvivalFS) writeFile(path string, data []byte) error {
	fs.files[path] = string(data)
	fs.writes = append(fs.writes, path)

	return nil
}

// assertSkillIdentityUnchanged is expectSkillIdentityUnchanged for rapid.
func assertSkillIdentityUnchanged(rt *rapid.T, before, after string) {
	for _, key := range skillIdentityKeys {
		want := yamlKeyBlock(before, key)
		if want == "" {
			rt.Fatalf("fixture has no %s block:\n%s", key, before)
		}

		if got := yamlKeyBlock(after, key); got != want {
			rt.Fatalf("%s changed: got %q want %q\nfull:\n%s", key, got, want, after)
		}
	}
}

// expectSkillIdentityUnchanged asserts every identity field's raw YAML block
// in after is byte-identical to its block in before (which must carry all
// three).
func expectSkillIdentityUnchanged(g Gomega, before, after string) {
	for _, key := range skillIdentityKeys {
		want := yamlKeyBlock(before, key)
		g.Expect(want).NotTo(BeEmpty(), "fixture must carry "+key)
		g.Expect(yamlKeyBlock(after, key)).To(Equal(want), key+" must be byte-unchanged")
	}
}

func newSkillSurvivalFS(byName map[string]string) *skillSurvivalFS {
	fs := &skillSurvivalFS{files: make(map[string]string, len(byName)), names: make([]string, 0, len(byName))}

	for name, body := range byName {
		fs.files[filepath.Join("/vault", name)] = body
		fs.names = append(fs.names, name)
	}

	return fs
}

// skillKeyGen draws source-qualified keys shaped like design D3's forms
// (`<plugin>:<n>`, `project:<r>:cmd:<n>`, `pi-pkg:<id>:pi-prompt:<n>`, ...).
func skillKeyGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9-]{0,12}(:[a-zA-Z0-9@._/-]{1,20}){1,3}`)
}

// skillSourceGen draws `~`-relative resolved paths, including spaces (e.g.
// macOS "Application Support") and long paths yaml.v3 may fold.
func skillSourceGen() *rapid.Generator[string] {
	return rapid.StringMatching(`~/[a-zA-Z0-9._@ -]{1,30}(/[a-zA-Z0-9._@ -]{1,30}){0,5}/SKILL\.md`)
}

// skillSurvivalNote is a registered skill note carrying all three identity
// fields plus every structure a rewrite site touches: a frontmatter wikilink
// (red_flags), a vocab tag, a supersedes entry, and pending: true.
func skillSurvivalNote() string {
	return skillSurvivalNoteWith(skillSurvivalKey, skillSurvivalSource)
}

// skillSurvivalNoteWith renders the fixture with the given key and source,
// each rendered by yaml.v3 exactly as the frontmatter writer would.
func skillSurvivalNoteWith(key, source string) string {
	identity, _ := yaml.Marshal(struct {
		SkillHash   string `yaml:"skill_hash"`
		SkillKey    string `yaml:"skill_key"`
		SkillSource string `yaml:"skill_source"`
	}{skillSurvivalHash, key, source})

	return "---\n" +
		"type: runbook\n" +
		"tier: L2\n" +
		"red_flags:\n    - skipped the checklist in [[" + skillSurvivalLinkTarget + "]]\n" +
		"luhmann: \"1a\"\n" +
		"created: \"2026-01-01\"\n" +
		"source: skill registration\n" +
		"user: agent@example.com\n" +
		"vault: personal\n" +
		string(identity) +
		"pending: true\n" +
		"tags:\n    - vocab/old-term\n" +
		"supersedes:\n    - note: " + skillSurvivalSupersededName + "\n      type: fact\n      claim: old claim\n" +
		"---\n\n" +
		"> Mirrors skill `brainstorming`.\n\n1. step one, see [[" + skillSurvivalLinkTarget + "]]\n"
}

// yamlKeyBlock returns key's raw top-level block in content's frontmatter —
// the "key:" line plus any indented continuation lines — or "" when absent.
func yamlKeyBlock(content, key string) string {
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		return ""
	}

	frontmatter, _, found := strings.Cut(rest, "\n---\n")
	if !found {
		return ""
	}

	lines := strings.Split(frontmatter, "\n")
	block := make([]string, 0, len(lines))

	for _, line := range lines {
		if len(block) == 0 {
			if strings.HasPrefix(line, key+":") {
				block = append(block, line)
			}

			continue
		}

		if !strings.HasPrefix(line, " ") {
			break
		}

		block = append(block, line)
	}

	return strings.Join(block, "\n")
}
