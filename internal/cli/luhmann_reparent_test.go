package cli_test

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestRenameAndRewriteReferences_AppendsOldBasenameToAliases is the spec
// scenario "A rename records the old name" (vault-note-identity, design D4
// H2): the renamed note's aliases gain its old basename in the rename's one
// write, after any aliases it already had and never twice; its new basename
// is dropped from the list (a note renamed back is not its own alias); its
// xid is kept.
func TestRenameAndRewriteReferences_AppendsOldBasenameToAliases(t *testing.T) {
	t.Parallel()

	const (
		oldBasename = "1100.2026-09-27.x"
		newBasename = "12a.2026-09-27.x"
		head        = "---\ntype: fact\nluhmann: \"1100\"\nxid: 7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e\n"
	)

	const (
		existing = "aliases:\n    - 5.2026-01-01.a\n"
		appended = existing + "    - " + oldBasename + "\n"
		onlyOld  = "aliases:\n    - " + oldBasename + "\n"
	)

	for name, tc := range map[string]struct{ aliases, want string }{
		"no aliases yet":      {"", onlyOld},
		"existing aliases":    {existing, appended},
		"flow-style aliases":  {"aliases: [5.2026-01-01.a]\n", appended},
		"compact sequence":    {"aliases:\n- 5.2026-01-01.a\n", appended},
		"old name already in": {onlyOld, onlyOld},
		"renamed back to a former name": {
			existing + "    - " + newBasename + "\n", appended,
		},
		"only alias is the new name": {"aliases:\n    - " + newBasename + "\n", onlyOld},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			fixture := newReparentFixture(map[string]string{
				oldBasename + ".md": head + tc.aliases + "---\n\nBody.\n",
			})

			_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
				map[string]string{oldBasename: newBasename})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(fixture.written).To(HaveLen(1), "the alias lands in the rename's one write")
			g.Expect(string(fixture.written["/vault/"+newBasename+".md"])).To(Equal(
				"---\ntype: fact\nluhmann: \"12a\"\nxid: 7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e\n" + tc.want + "---\n\nBody.\n"))
		})
	}
}

// TestRenameAndRewriteReferences_AppendsOldBasenameToAliasesProperty: for
// any existing aliases, a renamed note's aliases become the old list minus
// the new basename, plus the old basename (once), and every other
// frontmatter key but luhmann is unchanged; a note that is only a referrer
// keeps its aliases.
func TestRenameAndRewriteReferences_AppendsOldBasenameToAliasesProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		oldBasename := basenameGen().Draw(rt, "old")
		newBasename := "77.2026-01-01.renamed"
		aliases := rapid.SliceOfN(basenameGen(), 0, 3).Draw(rt, "aliases")

		if rapid.Bool().Draw(rt, "oldAlreadyAliased") {
			aliases = append(aliases, oldBasename)
		}

		if rapid.Bool().Draw(rt, "newAlreadyAliased") {
			aliases = append(aliases, newBasename)
		}

		exchange := exchangeFixture{XID: "7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e", Aliases: aliases}
		note := exchangeSurvivalNote("fact", exchange)
		referrer := exchangeSurvivalNote("feedback", exchange)
		referrer = strings.Replace(referrer, exchangeSurvivalLinkTarget, oldBasename, 1)

		fixture := newReparentFixture(map[string]string{oldBasename + ".md": note, "5.2026-01-01.ref.md": referrer})

		_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", map[string]string{oldBasename: newBasename})
		if err != nil {
			rt.Fatalf("rename: %v", err)
		}

		want := slices.DeleteFunc(slices.Clone(aliases), func(name string) bool { return name == newBasename })
		if !slices.Contains(want, oldBasename) {
			want = append(want, oldBasename)
		}

		before := frontmatterOf(note)
		after := frontmatterOf(string(fixture.written["/vault/"+newBasename+".md"]))

		if got := fmt.Sprint(after["aliases"]); got != fmt.Sprint(want) {
			rt.Fatalf("aliases = %s, want %v", got, want)
		}

		delete(before, "aliases")
		delete(before, "luhmann")
		delete(after, "aliases")
		delete(after, "luhmann")

		if !reflect.DeepEqual(before, after) {
			rt.Fatalf("a rename changed more than luhmann and aliases:\nbefore %v\nafter  %v", before, after)
		}

		if written, ok := fixture.written["/vault/5.2026-01-01.ref.md"]; ok &&
			fmt.Sprint(frontmatterOf(string(written))["aliases"]) != fmt.Sprint(frontmatterOf(referrer)["aliases"]) {
			rt.Fatalf("a referrer's aliases changed")
		}
	})
}

// TestRenameAndRewriteReferences_CascadingRenamesUseFinalMap asserts note A,
// itself being renamed, and referencing note B — also being renamed in the
// same run — ends up pointing at B's NEW basename, never B's old one.
func TestRenameAndRewriteReferences_CascadingRenamesUseFinalMap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const noteAName = "9a.2026-01-01.note-a.md"

	const noteABody = "---\ntype: fact\nluhmann: \"9a\"\n---\n\nSee [[9b.2026-01-01.note-b]] for context.\n"

	const noteBName = "9b.2026-01-01.note-b.md"

	const noteBBody = "---\ntype: fact\nluhmann: \"9b\"\n---\n\nBase fact.\n"

	fixture := newReparentFixture(map[string]string{
		noteAName: noteABody,
		noteBName: noteBBody,
	})

	renameMap := map[string]string{
		"9a.2026-01-01.note-a": "9c1.2026-01-01.note-a",
		"9b.2026-01-01.note-b": "9c2.2026-01-01.note-b",
	}

	rewritten, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", renameMap)
	g.Expect(err).NotTo(HaveOccurred())
	// Only note-a's references changed (note-b links to nothing), and it is
	// reported at its post-rename path so its sidecar can be rebuilt there.
	g.Expect(rewritten).To(Equal([]string{"/vault/9c1.2026-01-01.note-a.md"}))

	newAContent := string(fixture.written["/vault/9c1.2026-01-01.note-a.md"])
	g.Expect(newAContent).To(ContainSubstring("[[9c2.2026-01-01.note-b]]"))
	g.Expect(newAContent).NotTo(ContainSubstring("[[9b.2026-01-01.note-b]]"))
	g.Expect(newAContent).To(ContainSubstring(`luhmann: "9c1"`))
}

// TestRenameAndRewriteReferences_EmptyMapIsNoOp asserts an empty rename map
// touches nothing — ListMD is never even called.
func TestRenameAndRewriteReferences_EmptyMapIsNoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	listCalled := false
	deps := cli.RenameRewriteDeps{
		ListMD: func(string) ([]string, error) {
			listCalled = true

			return nil, nil
		},
	}

	_, err := cli.RenameAndRewriteReferences(deps, "/vault", nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(listCalled).To(BeFalse())
}

// TestRenameAndRewriteReferences_ListMDErrorPropagates asserts a ListMD
// failure is wrapped and returned, not swallowed.
func TestRenameAndRewriteReferences_ListMDErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.RenameRewriteDeps{
		ListMD: func(string) ([]string, error) { return nil, errReparentFixtureListMD },
	}

	_, err := cli.RenameAndRewriteReferences(deps, "/vault", map[string]string{"9a": "9b1"})
	g.Expect(err).To(MatchError(errReparentFixtureListMD))
}

// TestRenameAndRewriteReferences_MultipleReferencesAcrossNotes asserts every
// referencing note across the vault gets rewritten, including a legacy
// [[X.md]]-suffixed wikilink, a Supersedes: body line, and a frontmatter
// supersedes: note: field (both the bare and .md-suffixed forms).
func TestRenameAndRewriteReferences_MultipleReferencesAcrossNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "9a.2026-01-01.old-topic.md"

	const oldBody = "---\ntype: fact\nluhmann: \"9a\"\n---\n\nSome fact.\n"

	const legacyLinkName = "9c.2026-01-01.legacy-link.md"

	const legacyLinkBody = "---\ntype: fact\nluhmann: \"9c\"\n---\n\nSee [[9a.2026-01-01.old-topic.md]].\n"

	const supersedesBodyName = "9d.2026-01-01.supersedes-body.md"

	const supersedesBodyBody = "---\ntype: fact\nluhmann: \"9d\"\n---\n\n" +
		"Newer info.\n\nSupersedes: [[9a.2026-01-01.old-topic]] — updates: refined the claim\n"

	const supersedesFMBareName = "9e.2026-01-01.supersedes-fm-bare.md"

	const supersedesFMBareBody = "---\ntype: fact\nluhmann: \"9e\"\nsupersedes:\n" +
		"    - note: 9a.2026-01-01.old-topic\n      type: updates\n      claim: refined\n---\n\nBody.\n"

	const supersedesFMMDName = "9f.2026-01-01.supersedes-fm-md.md"

	const supersedesFMMDBody = "---\ntype: fact\nluhmann: \"9f\"\nsupersedes:\n" +
		"    - note: 9a.2026-01-01.old-topic.md\n      type: updates\n      claim: refined\n---\n\nBody.\n"

	fixture := newReparentFixture(map[string]string{
		oldName:              oldBody,
		legacyLinkName:       legacyLinkBody,
		supersedesBodyName:   supersedesBodyBody,
		supersedesFMBareName: supersedesFMBareBody,
		supersedesFMMDName:   supersedesFMMDBody,
	})

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", renameMap)
	g.Expect(err).NotTo(HaveOccurred())

	legacy := string(fixture.written["/vault/"+legacyLinkName])
	g.Expect(legacy).To(ContainSubstring("[[9b1.2026-01-01.old-topic]]"))

	superBody := string(fixture.written["/vault/"+supersedesBodyName])
	g.Expect(superBody).To(ContainSubstring("Supersedes: [[9b1.2026-01-01.old-topic]] — updates: refined the claim"))

	superFMBare := string(fixture.written["/vault/"+supersedesFMBareName])
	g.Expect(superFMBare).To(ContainSubstring("note: 9b1.2026-01-01.old-topic\n"))

	superFMMD := string(fixture.written["/vault/"+supersedesFMMDName])
	g.Expect(superFMMD).To(ContainSubstring("note: 9b1.2026-01-01.old-topic.md\n"))
}

// TestRenameAndRewriteReferences_OnlySupersedesNoteFieldsAreRewritten pins
// the scope of the frontmatter note: rewrite: a supersedes: entry is
// rewritten whether its sequence is indented or at column 0, while a note:
// key under any other top-level key (a parent link naming a note in another
// vault, design D4) is left alone.
func TestRenameAndRewriteReferences_OnlySupersedesNoteFieldsAreRewritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const referrerName = "9g.2026-01-01.referrer.md"

	const referrerBody = "---\ntype: fact\nluhmann: \"9g\"\nsupersedes:\n" +
		"- note: 9a.2026-01-01.old-topic\n  type: updates\n  claim: refined\n" +
		"parent:\n    vault: abc\n    links:\n        - note: 9a.2026-01-01.old-topic\n          via: offered\n" +
		"---\n\nBody.\n"

	fixture := newReparentFixture(map[string]string{
		"9a.2026-01-01.old-topic.md": "---\ntype: fact\nluhmann: \"9a\"\n---\n\nSome fact.\n",
		referrerName:                 referrerBody,
	})

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
		map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"})
	g.Expect(err).NotTo(HaveOccurred())

	referrer := string(fixture.written["/vault/"+referrerName])
	g.Expect(referrer).To(ContainSubstring("- note: 9b1.2026-01-01.old-topic\n  type: updates"))
	g.Expect(referrer).To(ContainSubstring("        - note: 9a.2026-01-01.old-topic\n          via: offered"))
}

// TestRenameAndRewriteReferences_ReadFileErrorPropagates asserts a ReadFile
// failure on any listed note is wrapped and returned.
func TestRenameAndRewriteReferences_ReadFileErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.RenameRewriteDeps{
		ListMD:   func(string) ([]string, error) { return []string{"9a.2026-01-01.old.md"}, nil },
		ReadFile: func(string) ([]byte, error) { return nil, errReparentFixtureReadFile },
	}

	_, err := cli.RenameAndRewriteReferences(deps, "/vault", map[string]string{"9a": "9b1"})
	g.Expect(err).To(MatchError(errReparentFixtureReadFile))
}

// TestRenameAndRewriteReferences_RenameErrorPropagates asserts a Rename
// failure on the note file itself (not the best-effort sidecar) is wrapped
// and returned.
func TestRenameAndRewriteReferences_RenameErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "9a.2026-01-01.old-topic.md"

	const oldBody = "---\ntype: fact\nluhmann: \"9a\"\n---\n\nSome fact.\n"

	fixture := newReparentFixture(map[string]string{oldName: oldBody})
	deps := fixture.deps()
	deps.Rename = func(string, string) error { return errReparentFixtureRename }

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(deps, "/vault", renameMap)
	g.Expect(err).To(MatchError(errReparentFixtureRename))
}

// TestRenameAndRewriteReferences_RenamedNoteWithoutFrontmatterKeepsBody
// asserts a renamed note with no YAML frontmatter block is still renamed
// (and its wikilinks still rewritten), just with no luhmann: field to update.
func TestRenameAndRewriteReferences_RenamedNoteWithoutFrontmatterKeepsBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "9a.2026-01-01.old-topic.md"

	const oldBody = "No frontmatter here, just prose.\n"

	fixture := newReparentFixture(map[string]string{oldName: oldBody})

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", renameMap)
	g.Expect(err).NotTo(HaveOccurred())

	newContent := string(fixture.written["/vault/9b1.2026-01-01.old-topic.md"])
	g.Expect(newContent).To(Equal(oldBody))
}

// TestRenameAndRewriteReferences_RenamedNoteWithoutLuhmannKeyUnchanged
// asserts a renamed note whose frontmatter has no luhmann: key is renamed
// with its frontmatter otherwise untouched, apart from the old-basename
// alias every rename records.
func TestRenameAndRewriteReferences_RenamedNoteWithoutLuhmannKeyUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "9a.2026-01-01.old-topic.md"

	const oldBody = "---\ntype: fact\n---\n\nSome fact.\n"

	fixture := newReparentFixture(map[string]string{oldName: oldBody})

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", renameMap)
	g.Expect(err).NotTo(HaveOccurred())

	newContent := string(fixture.written["/vault/9b1.2026-01-01.old-topic.md"])
	g.Expect(newContent).To(Equal("---\ntype: fact\naliases:\n    - 9a.2026-01-01.old-topic\n---\n\nSome fact.\n"),
		"only the old-basename alias is added")
}

// TestRenameAndRewriteReferences_SingleReferenceRewritten asserts one note
// linking to a renamed note gets its wikilink rewritten, and the renamed
// note itself (and its sidecar) is renamed with its luhmann: field updated.
func TestRenameAndRewriteReferences_SingleReferenceRewritten(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const oldName = "9a.2026-01-01.old-topic.md"

	const oldBody = "---\ntype: fact\nluhmann: \"9a\"\n---\n\nSome fact.\n"

	const referrerName = "9c.2026-01-01.referrer.md"

	const referrerBody = "---\ntype: fact\nluhmann: \"9c\"\n---\n\nSee [[9a.2026-01-01.old-topic]] for detail.\n"

	fixture := newReparentFixture(map[string]string{
		oldName:      oldBody,
		referrerName: referrerBody,
	})
	fixture.files["/vault/9a.2026-01-01.old-topic.vec.json"] = []byte(`{"schema_version":1}`)

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", renameMap)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fixture.renamed).To(ConsistOf(
		[2]string{"/vault/9a.2026-01-01.old-topic.md", "/vault/9b1.2026-01-01.old-topic.md"},
		[2]string{"/vault/9a.2026-01-01.old-topic.vec.json", "/vault/9b1.2026-01-01.old-topic.vec.json"},
	))

	newContent := string(fixture.written["/vault/9b1.2026-01-01.old-topic.md"])
	g.Expect(newContent).To(ContainSubstring(`luhmann: "9b1"`))

	referrerUpdated := string(fixture.written["/vault/"+referrerName])
	g.Expect(referrerUpdated).To(ContainSubstring("[[9b1.2026-01-01.old-topic]]"))
	g.Expect(referrerUpdated).NotTo(ContainSubstring("[[9a.2026-01-01.old-topic]]"))
}

// TestRenameAndRewriteReferences_UnrelatedNoteUntouched asserts a note with no
// reference to any renamed basename is never written.
func TestRenameAndRewriteReferences_UnrelatedNoteUntouched(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const unrelatedName = "9c.2026-01-01.unrelated.md"

	const unrelatedBody = "---\ntype: fact\nluhmann: \"9c\"\n---\n\nNo links here.\n"

	fixture := newReparentFixture(map[string]string{unrelatedName: unrelatedBody})

	_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", map[string]string{"9a": "9b1"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fixture.written).To(BeEmpty())
	g.Expect(fixture.renamed).To(BeEmpty())
}

// TestRenameAndRewriteReferences_WriteFileErrorPropagates asserts a WriteFile
// failure while rewriting an unrenamed note's references is wrapped and
// returned.
func TestRenameAndRewriteReferences_WriteFileErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const referrerName = "9c.2026-01-01.referrer.md"

	const referrerBody = "---\ntype: fact\nluhmann: \"9c\"\n---\n\nSee [[9a.2026-01-01.old-topic]] for detail.\n"

	fixture := newReparentFixture(map[string]string{referrerName: referrerBody})
	deps := fixture.deps()
	deps.WriteFile = func(string, []byte) error { return errReparentFixtureWriteFile }

	renameMap := map[string]string{"9a.2026-01-01.old-topic": "9b1.2026-01-01.old-topic"}

	_, err := cli.RenameAndRewriteReferences(deps, "/vault", renameMap)
	g.Expect(err).To(MatchError(errReparentFixtureWriteFile))
}

// unexported variables.
var (
	errReparentFixtureListMD    = errors.New("fixture: ListMD failed")
	errReparentFixtureReadFile  = errors.New("fixture: ReadFile failed")
	errReparentFixtureRename    = errors.New("fixture: Rename failed")
	errReparentFixtureWriteFile = errors.New("fixture: WriteFile failed")
)

// reparentFixture is a hand-rolled closure-based fake matching the existing
// internal/cli test convention (see VocabDeps fixtures in vocab_apply_test.go).
type reparentFixture struct {
	files   map[string][]byte
	written map[string][]byte
	renamed [][2]string
}

func (fixture *reparentFixture) deps() cli.RenameRewriteDeps {
	names := make([]string, 0, len(fixture.files))
	for path := range fixture.files {
		names = append(names, path[len("/vault/"):])
	}

	return cli.RenameRewriteDeps{
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			if data, ok := fixture.written[path]; ok {
				return data, nil
			}

			if data, ok := fixture.files[path]; ok {
				return data, nil
			}

			return nil, fmt.Errorf("no such file: %s", path)
		},
		WriteFile: func(path string, data []byte) error {
			fixture.written[path] = data

			return nil
		},
		Rename: func(oldPath, newPath string) error {
			if _, ok := fixture.files[oldPath]; !ok {
				return fmt.Errorf("rename: no such file: %s", oldPath)
			}

			fixture.files[newPath] = fixture.files[oldPath]
			delete(fixture.files, oldPath)
			fixture.renamed = append(fixture.renamed, [2]string{oldPath, newPath})

			return nil
		},
	}
}

func newReparentFixture(byName map[string]string) *reparentFixture {
	files := make(map[string][]byte, len(byName))
	for name, body := range byName {
		files["/vault/"+name] = []byte(body)
	}

	return &reparentFixture{
		files:   files,
		written: make(map[string][]byte),
	}
}
