package cli_test

// Rename pre-flight and CRLF→LF conversion (fix-show-amend-reparent-frontmatter
// design D5; update-reparent-luhmann-batch "Rename and rewrite SHALL refuse
// undecodable frontmatter and convert written CRLF notes to LF"; tasks
// 4.1-4.3).

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestAdoptSkillNote_ConvertsCRLFNote: adopt converts a CRLF runbook note to
// LF instead of refusing it (design D5) — both when the adopt renames it and
// when it already carries the key's slug — keeping its fields and luhmann:.
func TestAdoptSkillNote_ConvertsCRLFNote(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct{ basename, content string }{
		"renamed":         {"1049.2026-09-21.curate-review-pending-offers", curatePromotedNoteFixture()},
		"already slugged": {"1049.2026-09-21.skill-claude-curate", curateSkillNoteFixture(cli.SkillContentHash([]byte("x")))},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put(tc.basename+".md", crlf(tc.content))

			deps := cli.SkillAdoptDeps{
				Lock:     noLock,
				Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
				Rename:   skillAcceptRenameDeps(vault),
				Embedder: skillAcceptFakeEmbedder{},
			}

			var stdout bytes.Buffer

			err := cli.AdoptSkillNote(t.Context(), "/vault", installedEngramSkill("curate", []byte("# Curate\n")), "1049",
				deps, &stdout)
			g.Expect(err).NotTo(HaveOccurred())

			adopted, ok := vault.get("1049.2026-09-21.skill-claude-curate.md")
			g.Expect(ok).To(BeTrue())
			g.Expect(adopted).NotTo(ContainSubstring("\r"))

			doc := parseSkillAcceptFrontmatter(g, adopted)
			g.Expect(doc.SkillKey).To(Equal("claude:curate"))
			g.Expect(frontmatterOf(adopted)["luhmann"]).To(Equal("1049"))
			g.Expect(frontmatterOf(adopted)["situation"]).To(Equal(frontmatterOf(tc.content)["situation"]))
		})
	}
}

// TestRenameAndRewriteReferences_AnchoredLuhmannRefusedUntouched is the
// scenario "Anchored luhmann value is refused, not corrupted": rewriting
// `luhmann: &a "1050"` would leave `issue: *a` dangling, so the invocation
// fails naming the note and renames or writes nothing — even when another
// note in the same map is safe and listed first.
func TestRenameAndRewriteReferences_AnchoredLuhmannRefusedUntouched(t *testing.T) {
	t.Parallel()

	const (
		safeOld     = "1049.2026-01-01.safe"
		anchoredOld = "1050.2026-01-01.anchored"
		safeNote    = "---\ntype: fact\nluhmann: \"1049\"\n---\n\nSafe.\n"
		anchored    = "---\ntype: fact\nluhmann: &a \"1050\"\nissue: *a\n---\n\nAnchored.\n"
	)

	for name, renameMap := range map[string]map[string]string{
		"anchored only": {anchoredOld: "7a.2026-01-01.anchored"},
		"safe note listed first": {
			safeOld:     "7b.2026-01-01.safe",
			anchoredOld: "7a.2026-01-01.anchored",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fixture := newReparentFixture(map[string]string{safeOld + ".md": safeNote, anchoredOld + ".md": anchored})
			deps := fixture.deps()
			deps.ListMD = sortedListMD(fixture)

			_, err := cli.RenameAndRewriteReferences(deps, "/vault", renameMap)
			g.Expect(err).To(HaveOccurred())

			if err == nil {
				return
			}

			g.Expect(err.Error()).To(ContainSubstring(anchoredOld))
			g.Expect(err.Error()).NotTo(ContainSubstring(safeOld))
			g.Expect(err).To(MatchError(cli.ErrRenameUndecodableForTest))
			g.Expect(fixture.renamed).To(BeEmpty())
			g.Expect(fixture.written).To(BeEmpty())
		})
	}
}

// TestRenameAndRewriteReferences_CRLFProperty is P1 and P1-CRLF (design D7):
// over notes with unknown keys, an optionally anchored luhmann: value, and
// every CRLF shape, either the rename refuses (always when anchored) and
// touches nothing, or every renamed note decodes with its new luhmann and
// every other key unchanged, no written note holds CRLF, unwritten notes
// are byte-identical, converted notes are listed for a sidecar rebuild, and
// each written note is byte-identical to — and hashes the same as — the
// note the same rename writes for an LF-authored vault.
func TestRenameAndRewriteReferences_CRLFProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		vault := reparentVaultGen().Draw(rt, "vault")

		shaped := make(map[string]string, len(vault.notes))
		lfOnly := make(map[string]string, len(vault.notes))

		for name, note := range vault.notes {
			shaped[name] = note.render()
			lfOnly[name] = note.renderLF()
		}

		fixture := newReparentFixture(shaped)
		rewritten, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", vault.renameMap)

		if err != nil {
			if !vault.anchored {
				rt.Fatalf("an anchor-free vault was refused: %v", err)
			}

			if len(fixture.renamed) != 0 || len(fixture.written) != 0 {
				rt.Fatalf("a refused rename touched the vault: renamed %v, wrote %d", fixture.renamed, len(fixture.written))
			}

			return
		}

		if vault.anchored {
			rt.Fatalf("an anchored luhmann: value was renamed instead of refused")
		}

		lfFixture := newReparentFixture(lfOnly)

		lfRewritten, lfErr := cli.RenameAndRewriteReferences(lfFixture.deps(), "/vault", vault.renameMap)
		if lfErr != nil {
			rt.Fatalf("LF rename: %v", lfErr)
		}

		assertRenamedNotesIntact(rt, vault, fixture)
		assertWrittenNotesMatchLF(rt, fixture, lfFixture)
		assertUnwrittenNotesUntouched(rt, vault, shaped, fixture)
		assertConvertedNotesListed(rt, vault, shaped, fixture, rewritten)
		assertCRLFBodyHashesKept(rt, vault, shaped, fixture)

		if len(rewritten) < len(lfRewritten) {
			rt.Fatalf("CRLF run listed %v, fewer than the LF run's %v", rewritten, lfRewritten)
		}
	})
}

// TestRenameAndRewriteReferences_CRLFReferrerSupersedesRewritten is the
// scenario "CRLF referrer gets its supersedes rewritten": the referrer is
// converted to LF and its frontmatter supersedes: note is rewritten.
func TestRenameAndRewriteReferences_CRLFReferrerSupersedesRewritten(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		oldBasename = "12.2026-01-02.target"
		newBasename = "7a.2026-01-02.target"
		referrer    = "20.2026-01-03.referrer.md"
	)

	referrerLF := "---\ntype: fact\nluhmann: \"20\"\nsupersedes:\n    - note: " + oldBasename +
		".md\n      type: narrows\n      claim: old\n---\n\nBody text.\n"
	fixture := newReparentFixture(map[string]string{
		oldBasename + ".md": "---\ntype: fact\nluhmann: \"12\"\n---\n\nTarget.\n",
		referrer:            crlf(referrerLF),
	})

	rewritten, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
		map[string]string{oldBasename: newBasename})
	g.Expect(err).NotTo(HaveOccurred())

	written, ok := fixture.written["/vault/"+referrer]
	g.Expect(ok).To(BeTrue(), "the CRLF referrer must be written")
	g.Expect(string(written)).To(Equal(strings.Replace(referrerLF, oldBasename+".md", newBasename+".md", 1)))
	g.Expect(rewritten).To(ContainElement("/vault/" + referrer))
}

// TestRenameAndRewriteReferences_CRLFRenamedNoteConverted is the scenario
// "CRLF renamed note is converted and rewritten": one rename, one write,
// no CRLF left, luhmann: is the new id, the alias is recorded, every other
// key decodes unchanged, and the note is listed for a sidecar rebuild.
func TestRenameAndRewriteReferences_CRLFRenamedNoteConverted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		oldBasename = "12.2026-01-02.target"
		newBasename = "7a.2026-01-02.target"
	)

	lfNote := "---\ntype: fact\nsituation: working on it\nluhmann: \"12\"\nluhmann_old: \"3\"\n" +
		"extra:\n    nested: [a, b]\n---\n\nTarget body.\n\nSecond paragraph.\n"
	fixture := newReparentFixture(map[string]string{oldBasename + ".md": crlf(lfNote)})

	rewritten, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
		map[string]string{oldBasename: newBasename})
	g.Expect(err).NotTo(HaveOccurred())

	newPath := "/vault/" + newBasename + ".md"

	g.Expect(fixture.renamed).To(ContainElement([2]string{"/vault/" + oldBasename + ".md", newPath}))
	g.Expect(fixture.written).To(HaveLen(1))

	written := string(fixture.written[newPath])
	g.Expect(written).NotTo(ContainSubstring("\r\n"))

	after := frontmatterOf(written)
	g.Expect(after["luhmann"]).To(Equal("7a"))
	g.Expect(after["aliases"]).To(Equal([]any{oldBasename}))

	before := frontmatterOf(lfNote)

	for _, key := range []string{"luhmann", "aliases"} {
		delete(before, key)
		delete(after, key)
	}

	g.Expect(after).To(Equal(before))
	g.Expect(rewritten).To(ContainElement(newPath), "a converted note's sidecar must be rebuilt")
}

// TestRenameAndRewriteReferences_RefusalsJoinedNamingEachNote: a luhmann:
// the line rewrite cannot reach (a quoted key) would stay stale, and is
// refused alongside an anchored note in one joined error naming both.
func TestRenameAndRewriteReferences_RefusalsJoinedNamingEachNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		staleOld    = "12.2026-01-01.stale"
		anchoredOld = "13.2026-01-01.anchored"
	)

	fixture := newReparentFixture(map[string]string{
		staleOld + ".md":    "---\ntype: fact\n\"luhmann\": \"12\"\n---\n\nStale.\n",
		anchoredOld + ".md": "---\ntype: fact\nluhmann: &a \"13\"\nissue: *a\n---\n\nAnchored.\n",
	})

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", map[string]string{
		staleOld:    "7a.2026-01-01.stale",
		anchoredOld: "7b.2026-01-01.anchored",
	})
	g.Expect(err).To(MatchError(cli.ErrRenameStaleLuhmannForTest))
	g.Expect(err).To(MatchError(cli.ErrRenameUndecodableForTest))
	g.Expect(err).To(MatchError(ContainSubstring(staleOld)))
	g.Expect(err).To(MatchError(ContainSubstring(anchoredOld)))
	g.Expect(fixture.renamed).To(BeEmpty())
	g.Expect(fixture.written).To(BeEmpty())
}

// TestRenameAndRewriteReferences_UntouchedCRLFNoteNeverWritten is the
// scenario "Untouched CRLF notes stay byte-identical".
func TestRenameAndRewriteReferences_UntouchedCRLFNoteNeverWritten(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const untouched = "30.2026-01-04.untouched.md"

	original := crlf("---\ntype: fact\nluhmann: \"30\"\n---\n\nNo links here.\n")
	fixture := newReparentFixture(map[string]string{
		"12.2026-01-02.target.md": "---\ntype: fact\nluhmann: \"12\"\n---\n\nTarget.\n",
		untouched:                 original,
	})

	rewritten, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
		map[string]string{"12.2026-01-02.target": "7a.2026-01-02.target"})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fixture.written).NotTo(HaveKey("/vault/" + untouched))
	g.Expect(string(fixture.files["/vault/"+untouched])).To(Equal(original))
	g.Expect(rewritten).NotTo(ContainElement("/vault/" + untouched))
}

// TestRunReparentLuhmann_ApplyRebuildsConvertedNoteSidecars pins the
// freshness half of design D5: after apply, every note the rename converted
// from CRLF — the renamed note and a CRLF referrer — has a fresh sidecar.
func TestRunReparentLuhmann_ApplyRebuildsConvertedNoteSidecars(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()

	const renamed = "/vault/12.2026-01-02.second-note.md"

	files[renamed] = []byte(crlf(string(files[renamed])))

	referrer := "/vault/20.2026-01-03.unrelated-note.md"
	files[referrer] = []byte(crlf(
		"---\ntype: fact\nluhmann: \"20\"\ncreated: 2026-01-03\n---\n\nsee [[12.2026-01-02.second-note]] also.\n",
	))

	deps, _ := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)
	files["/answers.json"] = []byte(`{"reparenting":[{"note":"12","position":"continuation","target":"7"}],` +
		`"fingerprint":"` + fingerprint + `"}`)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	state := reparentFileMapFS(files)
	modelID := reparentFakeEmbedder{}.ModelID()

	for _, path := range []string{"/vault/7a.2026-01-02.second-note.md", referrer} {
		g.Expect(string(files[path])).NotTo(ContainSubstring("\r"), path)
		g.Expect(embed.ComputeState(state, path, modelID)).To(Equal(embed.StateOK), path)
	}
}

// TestRunReparentLuhmann_DryRunReportsAnchoredRefusal: --dry-run runs the
// same pre-flight as apply, so an anchored note is reported without any
// write.
func TestRunReparentLuhmann_DryRunReportsAnchoredRefusal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	files, names := twoRelatedTopLevelNotesFixture()

	const renamed = "/vault/12.2026-01-02.second-note.md"

	files[renamed] = []byte("---\ntype: fact\nluhmann: &a \"12\"\nissue: *a\ncreated: 2026-01-02\n---\n\nsecond.\n")

	deps, renames := newReparentDeps(files, names)
	fingerprint := reparentDeriveFingerprint(t, deps)
	files["/answers.json"] = []byte(`{"reparenting":[{"note":"12","position":"continuation","target":"7"}],` +
		`"fingerprint":"` + fingerprint + `"}`)

	before := maps.Clone(files)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(context.Background(), "/vault", "/chunks", "/answers.json", true, deps, &stdout)
	g.Expect(err).To(MatchError(cli.ErrRenameUndecodableForTest))
	g.Expect(err).To(MatchError(ContainSubstring("12.2026-01-02.second-note")))
	g.Expect(renames).To(BeEmpty())
	g.Expect(files).To(Equal(before))
}

// unexported constants.
const (
	shapeAllCRLF         = "crlf"
	shapeAllLF           = "lf"
	shapeCRLFBody        = "crlf-body"
	shapeCRLFFrontmatter = "crlf-frontmatter"
)

// reparentFileMapFS adapts a test file map to embed.FS.
type reparentFileMapFS map[string][]byte

func (fs reparentFileMapFS) ReadFile(path string) ([]byte, error) {
	data, ok := fs[path]
	if !ok {
		return nil, &reparentNotFoundError{path: path}
	}

	return data, nil
}

// reparentPropNote is one generated vault note: its frontmatter lines, body
// lines, and CRLF shape.
type reparentPropNote struct {
	frontmatter []string
	body        []string
	shape       string
}

func (note reparentPropNote) render() string {
	frontmatterEOL, bodyEOL := "\n", "\n"

	switch note.shape {
	case shapeAllCRLF:
		frontmatterEOL, bodyEOL = "\r\n", "\r\n"
	case shapeCRLFFrontmatter:
		frontmatterEOL = "\r\n"
	case shapeCRLFBody:
		bodyEOL = "\r\n"
	}

	lines := append([]string{"---"}, note.frontmatter...)
	lines = append(lines, "---")

	return strings.Join(lines, frontmatterEOL) + frontmatterEOL + bodyEOL + strings.Join(note.body, bodyEOL) + bodyEOL
}

func (note reparentPropNote) renderLF() string {
	note.shape = shapeAllLF

	return note.render()
}

// reparentPropVault is a generated vault: notes by filename, the rename map,
// and whether any renamed note's luhmann: value is anchored.
type reparentPropVault struct {
	notes     map[string]reparentPropNote
	renameMap map[string]string
	anchored  bool
	renamed   []string
	untouched []string
}

func assertCRLFBodyHashesKept(
	rt *rapid.T, vault reparentPropVault, shaped map[string]string, fixture *reparentFixture,
) {
	for _, name := range vault.renamed {
		if vault.notes[name].shape != shapeCRLFBody {
			continue
		}

		newPath := "/vault/" + vault.renameMap[strings.TrimSuffix(name, ".md")] + ".md"

		before, _ := cli.ExportExchangeHash([]byte(shaped[name]))
		after, _ := cli.ExportExchangeHash(fixture.written[newPath])

		if before != after {
			rt.Fatalf("%s (LF frontmatter, CRLF body): exchange hash changed %s -> %s", name, before, after)
		}
	}
}

func assertConvertedNotesListed(
	rt *rapid.T, vault reparentPropVault, shaped map[string]string, fixture *reparentFixture, rewritten []string,
) {
	for name, content := range shaped {
		if !strings.Contains(content, "\r\n") {
			continue
		}

		path := "/vault/" + name
		if newBasename, ok := vault.renameMap[strings.TrimSuffix(name, ".md")]; ok {
			path = "/vault/" + newBasename + ".md"
		}

		if _, written := fixture.written[path]; written && !slices.Contains(rewritten, path) {
			rt.Fatalf("converted note %s missing from the rewritten list %v", path, rewritten)
		}
	}
}

func assertRenamedNotesIntact(rt *rapid.T, vault reparentPropVault, fixture *reparentFixture) {
	for _, name := range vault.renamed {
		newBasename := vault.renameMap[strings.TrimSuffix(name, ".md")]
		newID, _, _ := strings.Cut(newBasename, ".")

		written, ok := fixture.written["/vault/"+newBasename+".md"]
		if !ok {
			rt.Fatalf("renamed note %s was not written", name)
		}

		var decoded map[string]any

		decodeErr := yaml.Unmarshal([]byte(frontmatterBlock(string(written))), &decoded)
		if decodeErr != nil {
			rt.Fatalf("renamed note %s does not decode: %v\n%s", name, decodeErr, written)
		}

		if decoded["luhmann"] != newID {
			rt.Fatalf("renamed note %s luhmann = %v, want %s", name, decoded["luhmann"], newID)
		}

		before := frontmatterOf(vault.notes[name].renderLF())

		for _, key := range []string{"luhmann", "aliases"} {
			delete(before, key)
			delete(decoded, key)
		}

		if !reflect.DeepEqual(before, decoded) {
			rt.Fatalf("renamed note %s changed more than luhmann and aliases:\nbefore %v\nafter  %v", name, before, decoded)
		}
	}
}

func assertUnwrittenNotesUntouched(
	rt *rapid.T, vault reparentPropVault, shaped map[string]string, fixture *reparentFixture,
) {
	for name, content := range shaped {
		path := "/vault/" + name
		if _, renamed := vault.renameMap[strings.TrimSuffix(name, ".md")]; renamed {
			continue
		}

		if _, written := fixture.written[path]; written {
			continue
		}

		if string(fixture.files[path]) != content {
			rt.Fatalf("unwritten note %s changed", name)
		}
	}

	for _, name := range vault.untouched {
		if _, written := fixture.written["/vault/"+name]; written {
			rt.Fatalf("untouched note %s was written", name)
		}
	}
}

func assertWrittenNotesMatchLF(rt *rapid.T, fixture, lfFixture *reparentFixture) {
	if !reflect.DeepEqual(slices.Sorted(maps.Keys(fixture.written)), slices.Sorted(maps.Keys(lfFixture.written))) {
		rt.Fatalf("written set differs from the LF-authored run:\n%v\n%v",
			slices.Sorted(maps.Keys(fixture.written)), slices.Sorted(maps.Keys(lfFixture.written)))
	}

	for path, data := range fixture.written {
		if bytes.Contains(data, []byte("\r\n")) {
			rt.Fatalf("written note %s still holds CRLF", path)
		}

		if !bytes.Equal(data, lfFixture.written[path]) {
			rt.Fatalf("written note %s differs from the LF-authored run:\n%q\n%q", path, data, lfFixture.written[path])
		}

		got, _ := cli.ExportExchangeHash(data)
		want, _ := cli.ExportExchangeHash(lfFixture.written[path])

		if got != want {
			rt.Fatalf("written note %s hashes %s, LF-authored equivalent %s", path, got, want)
		}
	}
}

// crlf converts every LF in s to CRLF.
func crlf(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// frontmatterBlock returns the text between a note's frontmatter delimiters.
func frontmatterBlock(content string) string {
	rest, _ := strings.CutPrefix(content, "---\n")
	block, _, _ := strings.Cut(rest, "\n---\n")

	return block
}

// reparentShapeGen draws one of the four CRLF layouts design D7 names.
func reparentShapeGen() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{shapeAllLF, shapeAllCRLF, shapeCRLFFrontmatter, shapeCRLFBody})
}

// reparentUnknownKeysGen draws 0-3 frontmatter lines for keys no note model
// defines, with scalar or list values.
func reparentUnknownKeysGen() *rapid.Generator[[]string] {
	return rapid.Custom(func(rt *rapid.T) []string {
		count := rapid.IntRange(0, 3).Draw(rt, "unknownCount")
		lines := make([]string, 0, count*3)
		value := rapid.StringMatching(`[a-z][a-z0-9]{0,8}`)

		for index := range count {
			key := fmt.Sprintf("x_unknown_%d", index)

			if rapid.Bool().Draw(rt, key+"IsList") {
				lines = append(lines, key+":")
				for _, item := range rapid.SliceOfN(value, 1, 3).Draw(rt, key+"Items") {
					lines = append(lines, "    - "+item)
				}

				continue
			}

			lines = append(lines, fmt.Sprintf("%s: %q", key, value.Draw(rt, key)))
		}

		return lines
	})
}

// reparentVaultGen draws a vault of two renamed notes (each optionally
// anchored), a referrer naming both (wikilink and supersedes:), and an
// untouched note — every note with its own CRLF shape and unknown keys.
func reparentVaultGen() *rapid.Generator[reparentPropVault] {
	return rapid.Custom(func(rt *rapid.T) reparentPropVault {
		vault := reparentPropVault{notes: map[string]reparentPropNote{}, renameMap: map[string]string{}}

		for index, id := range []string{"12", "13"} {
			oldBasename := id + ".2026-01-02.note-" + id
			newBasename := fmt.Sprintf("7%c.2026-01-02.note-%s", 'a'+index, id)
			vault.renameMap[oldBasename] = newBasename
			vault.renamed = append(vault.renamed, oldBasename+".md")

			luhmannLine := fmt.Sprintf("luhmann: %q", id)
			frontmatter := []string{"type: fact", "situation: working on " + id}

			if rapid.Bool().Draw(rt, "anchored"+id) {
				vault.anchored = true
				luhmannLine = fmt.Sprintf("luhmann: &a %q", id)
				frontmatter = append(frontmatter, luhmannLine, "issue: *a")
			} else {
				frontmatter = append(frontmatter, luhmannLine)
			}

			frontmatter = append(frontmatter, `created: "2026-01-02"`)
			frontmatter = append(frontmatter, reparentUnknownKeysGen().Draw(rt, "unknown"+id)...)

			vault.notes[oldBasename+".md"] = reparentPropNote{
				frontmatter: frontmatter,
				body:        []string{"Body of " + id + ".", "", "More text."},
				shape:       reparentShapeGen().Draw(rt, "shape"+id),
			}
		}

		referrer := append([]string{"type: fact", `luhmann: "20"`},
			"supersedes:", "    - note: 12.2026-01-02.note-12.md", "      type: narrows", "      claim: old")
		vault.notes["20.2026-01-03.referrer.md"] = reparentPropNote{
			frontmatter: append(referrer, reparentUnknownKeysGen().Draw(rt, "unknownReferrer")...),
			body:        []string{"See [[13.2026-01-02.note-13]] too."},
			shape:       reparentShapeGen().Draw(rt, "shapeReferrer"),
		}

		vault.notes["30.2026-01-04.untouched.md"] = reparentPropNote{
			frontmatter: append([]string{"type: fact", `luhmann: "30"`},
				reparentUnknownKeysGen().Draw(rt, "unknownUntouched")...),
			body:  []string{"No links here."},
			shape: reparentShapeGen().Draw(rt, "shapeUntouched"),
		}
		vault.untouched = []string{"30.2026-01-04.untouched.md"}

		return vault
	})
}

// sortedListMD lists the fixture's notes in name order, so a test can pin
// which note RenameAndRewriteReferences reaches first.
func sortedListMD(fixture *reparentFixture) func(string) ([]string, error) {
	return func(string) ([]string, error) {
		names := make([]string, 0, len(fixture.files))
		for path := range fixture.files {
			names = append(names, strings.TrimPrefix(path, "/vault/"))
		}

		sort.Strings(names)

		return names, nil
	}
}
