package cli_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestApplyVocabAssignmentAfterResituate_TagsNote verifies that
// applyVocabAssignmentAfterResituate assigns a vocab term to the note body
// when the body vector is close enough to the term vector.
func TestApplyVocabAssignmentAfterResituate_TagsNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	termVec := []float32{1, 0, 0, 0}
	noteVec := []float32{0.9, 0.1, 0, 0} // cosine ≈ 0.99 — above the default floor

	fakeSidecar := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "test", Dims: 4,
		BodyVector: noteVec, SituationVector: make([]float32, 4),
	})

	const rawNote = "---\ntype: fact\n---\n\nBody.\n"

	var written []byte

	deps := cli.ResituateDeps{
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".vec.json") {
				return fakeSidecar, nil
			}

			return nil, os.ErrNotExist
		},
		Write: func(_ string, data []byte) error {
			written = data

			return nil
		},
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: "agentic-recall-triggers", Vector: termVec}}, nil
		},
	}

	cli.ExportApplyVocabAssignmentAfterResituate(deps, "/vault", "/vault/1.note.md", rawNote)
	g.Expect(written).NotTo(BeNil())
	g.Expect(string(written)).To(ContainSubstring("agentic-recall-triggers"))
}

// TestRunResituate_CallsVocabAssignment verifies that RunResituate invokes
// the vocab-assignment path (LoadTermVectors is called) after writing the note.
func TestRunResituate_CallsVocabAssignment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const noteContent = "---\ntype: fact\ntier: L2\nsituation: old\nsubject: A\npredicate: has\nobject: B\n" +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\nInformation learned: when in old, A has B.\n"

	fakeVec := make([]float32, 4)
	fakeSidecar := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "test", Dims: 4,
		BodyVector: fakeVec, SituationVector: fakeVec,
	})

	loadTermsCalled := false

	deps := cli.ResituateDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: "1aa.2026-01-01.note.md", LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".md") {
				return []byte(noteContent), nil
			}

			if strings.HasSuffix(path, ".vec.json") {
				return fakeSidecar, nil
			}

			return nil, os.ErrNotExist
		},
		Write: func(string, []byte) error {
			return nil
		},
		Embedder: successEmbedder{},
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			loadTermsCalled = true

			return nil, nil // no terms → no assignment, but call path exercised
		},
	}

	var buf strings.Builder

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{
		Vault:     "/vault",
		Note:      "1aa",
		Situation: "new situation",
	}, deps, &buf)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loadTermsCalled).To(BeTrue(), "LoadTermVectors must be called after writing the note")
}

// TestRunResituate_ContentErrors drives the render-path failure branches:
// a note with no frontmatter, a note whose delimited frontmatter is not
// valid YAML (routed to the unknown-type arm), and a note whose created
// date will not parse. Each is served via injected Read so no temp files
// are needed; the write/embedder are no-ops because resituateContent fails
// before either runs.
func TestRunResituate_ContentErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		note string
	}{
		{name: "no frontmatter", note: "just a body, no frontmatter\n"},
		{name: "malformed frontmatter yaml", note: "---\n\ttype: fact\n---\n\nbody\n"},
		{name: "fact unparseable created", note: factNoteWithCreated("not-a-date")},
		{name: "feedback unparseable created", note: feedbackNoteWithCreated("not-a-date")},
		{name: "body without newline", note: factNoteBody("single line no newline")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps := injectedNoteDeps(tc.note, successEmbedder{}, nil)

			var stdout strings.Builder

			err := cli.RunResituate(t.Context(), cli.ResituateArgs{
				Vault:     "/v",
				Note:      injectedNoteID,
				Situation: resituateNewSituation,
			}, deps, &stdout)

			// "body without newline" is a valid rewrite (empty related tail);
			// the others are hard failures. Either way the render path runs.
			if tc.name == "body without newline" {
				g.Expect(err).NotTo(HaveOccurred())

				return
			}

			g.Expect(err).To(HaveOccurred())
		})
	}
}

// TestRunResituate_Fact rewrites a fact note's situation and asserts the
// whole document equals the original with every occurrence of the old
// situation replaced — this single oracle covers the frontmatter
// situation: field, the body formula clause, and preservation of every
// other field (created, tier, luhmann, issue, relations). It also asserts
// the re-embedded sidecar's content_hash tracks the new embed source.
func TestRunResituate_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "9aa.2026-05-10.nilaway-guard.md")
	writeResituateFixture(t, notePath, resituateFactNote)

	originalHash := embed.ContentHash([]byte(resituateFactNote))

	deps := cli.ExportNewOsResituateDeps(successEmbedder{})

	var stdout strings.Builder

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{
		Vault:     vault,
		Note:      "9aa",
		Situation: resituateNewSituation,
	}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	want := strings.ReplaceAll(resituateFactNote, resituateOldSituation, resituateNewSituation)
	g.Expect(string(got)).To(Equal(want))

	newHash := readSidecarHash(t, notePath)
	g.Expect(newHash).NotTo(Equal(originalHash))
	g.Expect(newHash).To(Equal(embed.ContentHash([]byte(want))))
}

// TestRunResituate_Feedback rewrites a feedback note's situation. Both the
// frontmatter situation: field and the body formula's "when <situation>"
// clause become the new value; every other field and the related-to tail
// are preserved. Same ReplaceAll oracle as the fact case.
func TestRunResituate_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "9ac.2026-05-12.nilaway-guard.md")
	writeResituateFixture(t, notePath, resituateFeedbackNote)

	originalHash := embed.ContentHash([]byte(resituateFeedbackNote))

	deps := cli.ExportNewOsResituateDeps(successEmbedder{})

	var stdout strings.Builder

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{
		Vault:     vault,
		Note:      "9ac",
		Situation: resituateNewSituation,
	}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	want := strings.ReplaceAll(resituateFeedbackNote, resituateOldSituation, resituateNewSituation)
	g.Expect(string(got)).To(Equal(want))

	newHash := readSidecarHash(t, notePath)
	g.Expect(newHash).NotTo(Equal(originalHash))
	g.Expect(newHash).To(Equal(embed.ContentHash([]byte(want))))
}

// TestRunResituate_IOErrors drives the I/O failure branches of RunResituate
// and writeResituatedSidecar via injected deps: scan failure, read failure,
// note-write failure, embed failure, and sidecar-write failure.
func TestRunResituate_IOErrors(t *testing.T) {
	t.Parallel()

	t.Run("scan error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := injectedNoteDeps(resituateFactNote, successEmbedder{}, nil)
		deps.Scan = func(string) ([]vaultgraph.Note, error) { return nil, errInjectedIO }

		err := cli.RunResituate(t.Context(), resituateArgs(), deps, &strings.Builder{})
		g.Expect(err).To(MatchError(errInjectedIO))
	})

	t.Run("read error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := injectedNoteDeps(resituateFactNote, successEmbedder{}, nil)
		deps.Read = func(string) ([]byte, error) { return nil, errInjectedIO }

		err := cli.RunResituate(t.Context(), resituateArgs(), deps, &strings.Builder{})
		g.Expect(err).To(MatchError(errInjectedIO))
	})

	t.Run("note write error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := injectedNoteDeps(resituateFactNote, successEmbedder{}, func(string, []byte) error {
			return errInjectedIO
		})

		err := cli.RunResituate(t.Context(), resituateArgs(), deps, &strings.Builder{})
		g.Expect(err).To(MatchError(errInjectedIO))
	})

	t.Run("embed error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		deps := injectedNoteDeps(resituateFactNote, failingEmbedder{}, nil)

		err := cli.RunResituate(t.Context(), resituateArgs(), deps, &strings.Builder{})
		g.Expect(err).To(MatchError(errEmbedDown))
	})

	t.Run("sidecar write error", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Note write succeeds; only the sidecar (.vec.json) write fails.
		deps := injectedNoteDeps(
			resituateFactNote,
			successEmbedder{},
			func(path string, _ []byte) error {
				if strings.HasSuffix(path, ".vec.json") {
					return errInjectedIO
				}

				return nil
			},
		)

		err := cli.RunResituate(t.Context(), resituateArgs(), deps, &strings.Builder{})
		g.Expect(err).To(MatchError(errInjectedIO))
	})
}

// TestRunResituate_LocatesByFullBasename verifies the note can be found by
// its complete basename, not just the leading luhmann id.
func TestRunResituate_LocatesByFullBasename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const basename = "9aa.2026-05-10.nilaway-guard"

	vault := t.TempDir()
	notePath := filepath.Join(vault, basename+".md")
	writeResituateFixture(t, notePath, resituateFactNote)

	deps := cli.ExportNewOsResituateDeps(successEmbedder{})

	var stdout strings.Builder

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{
		Vault:     vault,
		Note:      basename,
		Situation: resituateNewSituation,
	}, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	got, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(got)).To(ContainSubstring("situation: " + resituateNewSituation))
}

// TestRunResituate_LocksVaultAroundReadModifyWrite asserts that RunResituate
// acquires the vault lock BEFORE reading the note and releases it AFTER
// writing the sidecar, so concurrent amend/resituate/learn runs do not lose
// each other's writes.
func TestRunResituate_LocksVaultAroundReadModifyWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var order []string

	noteName := "9zz.2026-05-10.lock-test"
	noteContent := `---
type: fact
tier: L2
situation: original situation
subject: A
predicate: has
object: B
luhmann: "9zz"
created: "2026-05-10"
source: test
---

Information learned: when in original situation, A has B.
`

	deps := cli.ResituateDeps{
		Lock: func(string) (func(), error) {
			order = append(order, "lock")

			return func() { order = append(order, "unlock") }, nil
		},
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: noteName, LuhmannID: "9zz"}}, nil
		},
		Read: func(string) ([]byte, error) {
			order = append(order, "read")

			return []byte(noteContent), nil
		},
		Write: func(string, []byte) error {
			order = append(order, "write")

			return nil
		},
		Embedder: successEmbedder{},
	}

	args := cli.ResituateArgs{
		Vault:     "/vault",
		Note:      "9zz",
		Situation: "new test situation",
	}

	var stdout strings.Builder

	err := cli.RunResituate(t.Context(), args, deps, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(order).To(ContainElements("lock", "read", "write", "unlock"),
		"all lock events must be recorded")

	lockIdx := sliceIndex(order, "lock")
	readIdx := sliceIndex(order, "read")
	writeIdx := sliceIndex(order, "write")
	unlockIdx := sliceIndex(order, "unlock")

	g.Expect(lockIdx).To(BeNumerically("<", readIdx), "lock must precede read")
	g.Expect(readIdx).To(BeNumerically("<", writeIdx), "read must precede write")
	g.Expect(writeIdx).To(BeNumerically("<", unlockIdx), "final write must precede unlock")
}

// TestRunResituate_NotFound asserts the sentinel error when no note in the
// vault matches the requested id or basename.
func TestRunResituate_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(Succeed())

	deps := cli.ExportNewOsResituateDeps(successEmbedder{})

	var stdout strings.Builder

	err := cli.RunResituate(t.Context(), cli.ResituateArgs{
		Vault:     vault,
		Note:      "does-not-exist",
		Situation: resituateNewSituation,
	}, deps, &stdout)
	g.Expect(err).To(MatchError(cli.ErrResituateNoteNotFoundForTest))
}

// unexported constants.
const (
	injectedNoteID    = "9zz.2026-05-10.injected"
	resituateFactNote = `---
type: fact
tier: L2
situation: debugging a flaky nilaway guard
subject: nilaway
predicate: does not recognize
object: gomega calls as nil guards
luhmann: "9aa"
created: "2026-05-10"
source: agent
issue: "642"
---

Information learned: when in debugging a flaky nilaway guard, nilaway does not recognize gomega calls as nil guards.

Related to:
- [[9a.2026-05-01.nilaway-basics]] — foundational concept.
`
	resituateFeedbackNote = `---
type: feedback
tier: L2
situation: debugging a flaky nilaway guard
behavior: accessed a pointer field without a nil guard
impact: nilaway flagged a potential nil panic
action: add an explicit nil guard before the field access
luhmann: "9ac"
created: "2026-05-12"
source: agent
---

Lesson learned: when debugging a flaky nilaway guard, add an explicit nil guard before the field access.

Related to:
- [[9a.2026-05-01.nilaway-basics]] — foundational concept.
`
	resituateNewSituation = "auditing pointer nil-checks before field access"
	resituateOldSituation = "debugging a flaky nilaway guard"
)

// unexported variables.
var (
	errInjectedIO = errors.New("injected io failure")
)

// factNoteBody builds a fact note whose body is a single line with no
// trailing newline, exercising relatedTail's no-newline branch.
func factNoteBody(body string) string {
	return "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"9zz\"\ncreated: \"2026-05-10\"\nsource: agent\n---\n" + body
}

// factNoteWithCreated builds a minimal fact note with the given created
// value, used to drive the created-date parse-error branch.
func factNoteWithCreated(created string) string {
	return fmt.Sprintf(
		"---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n"+
			"luhmann: \"9zz\"\ncreated: %q\nsource: agent\n---\n\nbody\n",
		created,
	)
}

// feedbackNoteWithCreated builds a minimal feedback note with the given
// created value, driving the feedback created-date parse-error branch.
func feedbackNoteWithCreated(created string) string {
	return fmt.Sprintf(
		"---\ntype: feedback\nsituation: s\nbehavior: a\nimpact: b\naction: c\n"+
			"luhmann: \"9zz\"\ncreated: %q\nsource: agent\n---\n\nbody\n",
		created,
	)
}

// injectedNoteDeps returns ResituateDeps whose Scan yields a single note
// (basename injectedNoteID) and whose Read returns the fixed content for any
// path. write overrides the Write closure; nil means a no-op success.
func injectedNoteDeps(
	content string,
	emb embed.Embedder,
	write func(string, []byte) error,
) cli.ResituateDeps {
	if write == nil {
		write = func(string, []byte) error { return nil }
	}

	return cli.ResituateDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: injectedNoteID, LuhmannID: "9zz"}}, nil
		},
		Read:     func(string) ([]byte, error) { return []byte(content), nil },
		Write:    write,
		Embedder: emb,
	}
}

// readSidecarHash reads the sidecar beside notePath and returns its
// content_hash.
func readSidecarHash(t *testing.T, notePath string) string {
	t.Helper()
	g := NewWithT(t)

	data, err := os.ReadFile(embed.SidecarPath(notePath))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return ""
	}

	sidecar, unmErr := embed.UnmarshalSidecar(data)
	g.Expect(unmErr).NotTo(HaveOccurred())

	return sidecar.ContentHash
}

// resituateArgs returns the standard ResituateArgs targeting the injected
// note, used by the I/O-error subtests.
func resituateArgs() cli.ResituateArgs {
	return cli.ResituateArgs{Vault: "/v", Note: injectedNoteID, Situation: resituateNewSituation}
}

// writeResituateFixture writes a note plus a stale sidecar (so the re-embed
// has something to overwrite) under a temp vault.
func writeResituateFixture(t *testing.T, notePath, content string) {
	t.Helper()
	g := NewWithT(t)

	g.Expect(os.MkdirAll(filepath.Dir(notePath), 0o750)).To(Succeed())
	g.Expect(os.WriteFile(notePath, []byte(content), 0o600)).To(Succeed())

	stale := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "m@4",
		Dims:             4,
		SituationVector:  []float32{0, 0, 0, 0},
		BodyVector:       []float32{0, 0, 0, 0},
		ContentHash:      "sha256:stale",
	}
	sidecarPath := embed.SidecarPath(notePath)
	g.Expect(os.WriteFile(sidecarPath, embed.MarshalSidecar(stale), 0o600)).To(Succeed())
}
