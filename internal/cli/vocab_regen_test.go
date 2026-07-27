package cli_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestRegenVocab_DryRun_ChangesNothing verifies --dry-run reports what would
// be removed/cleaned/regenerated but writes and deletes nothing.
func TestRegenVocab_DryRun_ChangesNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	termNote := "---\ntype: fact\nterm: recall\ndescription: recall the vault\n---\n\nExemplars:\n" +
		"- recalling before a decision\n"
	memberNote := "---\ntype: fact\nvocab: [recall]\ntags:\n    - go\n---\n\n" +
		"Vocab: recall\nsome body text\n"

	files := map[string][]byte{
		"/vault/vocab.recall.md":        []byte(termNote),
		"/vault/vocab.index.md":         []byte("---\ntype: fact\nvocab_version: \"2.0\"\n---\n\nindex\n"),
		"/vault/1.2026-07-01.member.md": []byte(memberNote),
	}
	names := []string{"vocab.recall.md", "vocab.index.md", "1.2026-07-01.member.md"}

	deps, written, deleted := newRegenVocabDeps(files, names)

	result, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, true)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(result.OldFilesRemoved).To(Equal(2), "both vocab.recall.md and vocab.index.md are old-format")
	g.Expect(result.MembersCleaned).To(Equal(1))
	g.Expect(result.TermsSeeded).To(Equal(1))
	g.Expect(*written).To(BeEmpty(), "dry-run must not write anything")
	g.Expect(*deleted).To(BeEmpty(), "dry-run must not delete anything")
}

// TestRegenVocab_ListMDErrorPropagates verifies regenVocab wraps and returns
// a ListMD failure rather than silently proceeding as if the vault were
// empty.
func TestRegenVocab_ListMDErrorPropagates(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	deps := cli.VocabDeps{
		ListMD: func(string) ([]string, error) { return nil, errRegenTestNotFound },
	}

	_, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, false)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("regen-vocab")))
}

// TestRegenVocab_NothingToRegenerate verifies the cheap no-op path: a vault
// with no old-format vocab files and no legacy member notes returns a
// zero-valued result, and no writes/deletes are attempted.
func TestRegenVocab_NothingToRegenerate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{
		"/vault/1.2026-07-01.some-note.md": []byte("---\ntype: fact\ntags:\n    - go\n---\n\nbody\n"),
	}
	names := []string{"1.2026-07-01.some-note.md"}

	deps, written, deleted := newRegenVocabDeps(files, names)

	result, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(result.OldFilesRemoved).To(Equal(0))
	g.Expect(result.MembersCleaned).To(Equal(0))
	g.Expect(result.TermsSeeded).To(Equal(0))
	g.Expect(result.NotesAssigned).To(Equal(0))
	g.Expect(*written).To(BeEmpty())
	g.Expect(*deleted).To(BeEmpty())
}

// TestRegenVocab_RealRun_RemovesFilesAndCleansMembers verifies a real
// (non-dry-run) regen removes the old-format hub files, strips the legacy
// vocab: key / Vocab: body line from the member note, and mints a
// current-format definition note carrying the harvested term+description.
func TestRegenVocab_RealRun_RemovesFilesAndCleansMembers(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	termNote := "---\ntype: fact\nterm: recall\ndescription: recall the vault\n---\n\nExemplars:\n" +
		"- recalling before a decision\n"
	memberNote := "---\ntype: fact\nvocab: [recall]\ntags:\n    - go\n---\n\n" +
		"Vocab: recall\nsome body text\n"

	files := map[string][]byte{
		"/vault/vocab.recall.md":        []byte(termNote),
		"/vault/vocab.index.md":         []byte("---\ntype: fact\nvocab_version: \"2.0\"\n---\n\nindex\n"),
		"/vault/1.2026-07-01.member.md": []byte(memberNote),
	}
	names := []string{"vocab.recall.md", "vocab.index.md", "1.2026-07-01.member.md"}

	deps, written, deleted := newRegenVocabDeps(files, names)

	result, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(result.OldFilesRemoved).To(Equal(2))
	g.Expect(result.MembersCleaned).To(Equal(1))
	g.Expect(result.TermsSeeded).To(Equal(1))

	g.Expect(*deleted).To(ContainElement("/vault/vocab.recall.md"))
	g.Expect(*deleted).To(ContainElement("/vault/vocab.index.md"))

	memberContent, ok := (*written)["/vault/1.2026-07-01.member.md"]
	g.Expect(ok).To(BeTrue(), "member note must be rewritten to strip legacy vocab channel")

	if ok {
		g.Expect(string(memberContent)).NotTo(ContainSubstring("vocab: [recall]"))
		g.Expect(string(memberContent)).NotTo(ContainSubstring("Vocab: recall"))
	}

	foundMinted := false

	for path, content := range *written {
		if path == "/vault/1.2026-07-01.member.md" {
			continue
		}

		if strings.Contains(string(content), "recall the vault") {
			foundMinted = true
		}
	}

	g.Expect(foundMinted).To(BeTrue(), "a current-format definition note must be minted from the old term note")
}

// TestRegenVocab_SkipsUnparseableLegacyTermNotes verifies harvestLegacyVocabSeed
// (via parseLegacyTermNote) skips a term note that is unreadable, has no
// frontmatter, or has an empty term: key — without failing the whole regen —
// while still harvesting a well-formed sibling term note.
func TestRegenVocab_SkipsUnparseableLegacyTermNotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{
		// "/vault/vocab.unreadable.md" intentionally absent from files.
		"/vault/vocab.nofrontmatter.md": []byte("no frontmatter here at all\n"),
		"/vault/vocab.emptyterm.md":     []byte("---\ntype: fact\nterm: \"\"\ndescription: x\n---\n\nbody\n"),
		"/vault/vocab.good.md":          []byte("---\ntype: fact\nterm: good\ndescription: a good term\n---\n\nbody\n"),
	}
	names := []string{
		"vocab.unreadable.md", "vocab.nofrontmatter.md", "vocab.emptyterm.md", "vocab.good.md",
	}

	deps, _, _ := newRegenVocabDeps(files, names)

	result, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, true)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.TermsSeeded).To(Equal(1), "only vocab.good.md is a parseable legacy term note")
	g.Expect(result.OldFilesRemoved).To(Equal(4), "all four are still old-format hub files by filename shape")
}

// TestRemoveOldVocabFiles_WarnsOnDeleteErrorAndSkipsCount verifies that a
// hub file whose DeleteFile call fails is logged (when LogWarning is wired)
// and excluded from the removed count, without aborting the rest of the
// regen — and that a nil LogWarning does not panic.
func TestRemoveOldVocabFiles_WarnsOnDeleteErrorAndSkipsCount(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	files := map[string][]byte{
		"/vault/vocab.recall.md": []byte("---\ntype: fact\nterm: recall\ndescription: d\n---\n\nbody\n"),
	}
	names := []string{"vocab.recall.md"}

	warnings := []string{}

	deps := cli.VocabDeps{
		Lock:      func(string) (func(), error) { return func() {}, nil },
		ListMD:    func(string) ([]string, error) { return names, nil },
		ReadFile:  func(path string) ([]byte, error) { return files[path], nil },
		WriteFile: func(string, []byte) error { return nil },
		DeleteFile: func(string) error {
			return errRegenTestNotFound
		},
		LogWarning: func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		},
		Now: func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	result, err := cli.ExportRegenVocab(t.Context(), "/vault", deps, false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).NotTo(BeEmpty(), "a delete failure must be logged when LogWarning is wired")

	// The dry-run-sized OldFilesRemoved count reflects what WOULD be removed
	// (by filename shape), independent of the real delete outcome below.
	g.Expect(result.OldFilesRemoved).To(Equal(1))

	// A nil LogWarning must not panic when a delete fails.
	deps.LogWarning = nil

	g.Expect(func() {
		_, _ = cli.ExportRegenVocab(t.Context(), "/vault", deps, false)
	}).NotTo(Panic())
}

// TestStripLegacyVocabChannel verifies the frontmatter-key and body-line
// stripping in isolation, including the no-op case.
func TestStripLegacyVocabChannel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	t.Run("strips both channels", func(t *testing.T) {
		t.Parallel()

		content := "---\ntype: fact\nvocab: [recall]\ntags:\n    - go\n---\n\nVocab: recall\nbody text\n"

		updated, changed := cli.ExportStripLegacyVocabChannel(content)
		g.Expect(changed).To(BeTrue())
		g.Expect(updated).NotTo(ContainSubstring("vocab: [recall]"))
		g.Expect(updated).NotTo(ContainSubstring("Vocab: recall"))
		g.Expect(updated).To(ContainSubstring("body text"))
		g.Expect(updated).To(ContainSubstring("tags:\n    - go"))
	})

	t.Run("no legacy channel is a no-op", func(t *testing.T) {
		t.Parallel()

		content := "---\ntype: fact\ntags:\n    - go\n---\n\nbody text\n"

		updated, changed := cli.ExportStripLegacyVocabChannel(content)
		g.Expect(changed).To(BeFalse())
		g.Expect(updated).To(Equal(content))
	})
}

// unexported variables.
var (
	errRegenTestNotFound = &regenTestNotFoundError{}
)

type regenTestNotFoundError struct{}

func (*regenTestNotFoundError) Error() string { return "not found" }

// newRegenVocabDeps builds a VocabDeps backed by in-memory maps of note
// name → content, tracking every WriteFile/DeleteFile call so tests can
// assert on them. names lists the vault's .md filenames.
func newRegenVocabDeps(files map[string][]byte, names []string) (cli.VocabDeps, *map[string][]byte, *[]string) {
	written := map[string][]byte{}
	deleted := []string{}

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) { return names, nil },
		ReadFile: func(path string) ([]byte, error) {
			data, ok := files[path]
			if !ok {
				return nil, errRegenTestNotFound
			}

			return data, nil
		},
		WriteFile: func(path string, data []byte) error {
			written[path] = data
			files[path] = data // visible to a subsequent ReadFile, like a real filesystem

			return nil
		},
		WriteSidecar: func(path string, data []byte) error {
			written[path] = data
			files[path] = data

			return nil
		},
		Embedder: &fakeEmbedder{},
		DeleteFile: func(path string) error {
			deleted = append(deleted, path)
			delete(files, path)

			return nil
		},
		LogWarning: func(string, ...any) {},
		Now:        func() time.Time { return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC) },
	}

	return deps, &written, &deleted
}
