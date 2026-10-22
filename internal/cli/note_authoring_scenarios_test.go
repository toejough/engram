package cli_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792b12c6a3_RunLearnQA_TheQuestionOfAPairIsNotTagged verifies 792b12c6a3 (note-authoring.md:301): over a
// vault with a term whose vector is the same for the question and for its answer, and a readable sidecar for each,
// learning the pair rewrites the answer note with the term's vocab/ tag and writes the question note no more.
func Test792b12c6a3_RunLearnQA_TheQuestionOfAPairIsNotTagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault with a term close to both a question and its answer
	vault := newNAU1QAVault()

	// When the pair is learned
	var stdout strings.Builder

	err := cli.RunLearnQA(context.Background(), cli.LearnQAArgs{
		Vault:    nau1VaultPath,
		Slug:     nau1Slug,
		Question: "What is the weekly routine?",
		Answer:   "The team follows the routine.",
		Source:   nau1Source,
	}, vault.deps(), &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	paths := strings.Fields(stdout.String())
	g.Expect(paths).To(HaveLen(nau1PairNoteCount), "the command prints the question path then the answer path")

	if len(paths) != nau1PairNoteCount {
		return
	}

	questionPath, answerPath := paths[0], paths[1]

	// the Given holds: both notes were embedded, so the question could have been tagged
	g.Expect(vault.sidecars).To(HaveKey(embed.SidecarPath(questionPath)))
	g.Expect(vault.sidecars).To(HaveKey(embed.SidecarPath(answerPath)))

	// Then only the answer note is tagged: it is the one note rewritten, and it carries the term's tag
	g.Expect(vault.rewrittenNotes).To(Equal([]string{answerPath}))
	g.Expect(vault.notes[answerPath]).To(ContainSubstring("tags:\n    - vocab/" + nau1Term))
	g.Expect(vault.notes[questionPath]).NotTo(ContainSubstring("vocab/"))
}

// Test792b12c8a1_Targets_LearnFact_ALocalLearnWritesNoPendingMarker verifies 792b12c8a1 (note-authoring.md:343):
// engram learn fact run on the host over an in-memory vault writes a fact note whose frontmatter has no pending
// line.
func Test792b12c8a1_Targets_LearnFact_ALocalLearnWritesNoPendingMarker(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given the user runs engram learn fact on the host (no parent, no served door: the host's own vault)
	vault := newCV1Vault()

	// When the note is written
	result := vault.learnFact("local-note", "a plain local situation")

	// Then it carries no pending line
	name := vault.requireNote(g, `\.local-note\.md$`)
	content := vault.read(name)

	block, _, found := embed.SplitFrontmatter([]byte(content))
	g.Expect(found).To(BeTrue(), "the note has frontmatter")

	if !found {
		return
	}

	var front map[string]any

	g.Expect(yaml.Unmarshal(block, &front)).To(Succeed())
	g.Expect(front).To(HaveKeyWithValue("type", "fact"), "the frontmatter parsed is the fact note's")
	g.Expect(front).NotTo(HaveKey("pending"))
	g.Expect(content).NotTo(MatchRegexp(`(?m)^\s*pending\s*:`))

	// the write itself was clean; checked after the Then so that a pending side effect (for example the
	// pending-offers warning) cannot stop the test before the Then runs
	g.Expect(result.stderr).To(BeEmpty())
}

// unexported constants.
const (
	nau1PairNoteCount = 2
	nau1Slug          = "weekly-routine"
	nau1Source        = "nau1"
	nau1Term          = "routines"
	nau1VaultPath     = "/vault"
)

// nau1FixedEmbedder embeds every text as the same vector, the one of nau1Term, so that a question and its answer
// are equally close to the term.
type nau1FixedEmbedder struct{}

func (nau1FixedEmbedder) Dims() int { return len(nau1TermVector()) }

func (nau1FixedEmbedder) Embed(context.Context, string) ([]float32, error) {
	return nau1TermVector(), nil
}

func (nau1FixedEmbedder) ModelID() string { return "nau1-fixed@2" }

// nau1QAVault is the in-memory state behind the LearnQADeps of one scenario: the notes and sidecars written, and
// the notes rewritten after their sidecar (the vocabulary tagging).
type nau1QAVault struct {
	notes          map[string]string
	sidecars       map[string][]byte
	rewrittenNotes []string
}

// deps wires the vault into the deps of the question-and-answer learn, with a vocabulary of one term.
func (v *nau1QAVault) deps() cli.LearnQADeps {
	return cli.LearnQADeps{
		Now:    func() time.Time { return time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC) },
		Getenv: func(string) string { return "" },
		ListMD: func(string) ([]string, error) { return nil, nil },
		Lock:   func(string) (func(), error) { return func() {}, nil },
		WriteNew: func(path string, data []byte) error {
			v.notes[path] = string(data)

			return nil
		},
		RemoveFile: func(string) error { return nil },
		ReadFile:   func(string) ([]byte, error) { return nil, fs.ErrNotExist },
		Embedder:   nau1FixedEmbedder{},
		WriteSidecar: func(path string, data []byte) error {
			v.sidecars[path] = data

			return nil
		},
		LogWarning: nil,
		LoadTermVectors: func(string) ([]cli.TermWithVector, error) {
			return []cli.TermWithVector{{Term: nau1Term, Vector: nau1TermVector()}}, nil
		},
		ReadSidecar: func(path string) ([]byte, error) {
			data, found := v.sidecars[path]
			if !found {
				return nil, fs.ErrNotExist
			}

			return data, nil
		},
		WriteNote: func(path string, data []byte) error {
			if filepath.Ext(path) == ".md" {
				v.notes[path] = string(data)
				v.rewrittenNotes = append(v.rewrittenNotes, path)

				return nil
			}

			v.sidecars[path] = data // the refit baseline file the trigger check seeds

			return nil
		},
	}
}

func nau1TermVector() []float32 { return []float32{1, 0} }

func newNAU1QAVault() *nau1QAVault {
	return &nau1QAVault{notes: map[string]string{}, sidecars: map[string][]byte{}}
}
