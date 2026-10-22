package cli_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b20c10a5_VocabularyEngine_BadSeedFailsBeforeTheLock verifies that a seed file that cannot be read, or is
// not a YAML list of entries, fails bootstrap with an error naming the problem before the vault lock is taken,
// and that nothing is written and nothing is printed.
func Test792b20c10a5_VocabularyEngine_BadSeedFailsBeforeTheLock(t *testing.T) {
	t.Parallel()

	const (
		seedPath  = "/seed.yaml"
		vaultPath = "/vault"
	)

	errSeedUnreadable := errors.New("seed unreadable")

	cases := []struct {
		name     string
		readErr  error
		seed     string
		wantText string
	}{
		{
			name:     "an unreadable seed",
			readErr:  errSeedUnreadable,
			wantText: "reading seed",
		},
		{
			name:     "a YAML mapping instead of a list",
			seed:     "term: eval-topic\ndescription: how evals are designed\n",
			wantText: "cannot parse seed YAML",
		},
		{
			name:     "a YAML list of scalars instead of entries",
			seed:     "- eval-topic\n- another-topic\n",
			wantText: "cannot parse seed YAML",
		},
	}

	for _, tc := range cases {
		g := NewWithT(t)

		var lockCalls, writes int

		deps := cli.VocabDeps{
			Lock: func(string) (func(), error) {
				lockCalls++

				return func() {}, nil
			},
			ListMD: func(string) ([]string, error) { return []string{"1aa.note-a.md"}, nil },
			ReadFile: func(path string) ([]byte, error) {
				if path == seedPath && tc.readErr != nil {
					return nil, tc.readErr
				}

				if path == seedPath {
					return []byte(tc.seed), nil
				}

				return nil, &testNotFoundError{path: path}
			},
			WriteFile: func(string, []byte) error {
				writes++

				return nil
			},
			WriteSidecar: func(string, []byte) error {
				writes++

				return nil
			},
			DeleteFile: func(string) error {
				writes++

				return nil
			},
			LogWarning: func(string, ...any) {},
			Now:        func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		}

		var stdout strings.Builder

		err := cli.RunVocabBootstrap(t.Context(), cli.VocabBootstrapArgs{Vault: vaultPath, SeedFile: seedPath},
			deps, &stdout)

		g.Expect(err).To(MatchError(ContainSubstring(tc.wantText)), tc.name+": the error must name the problem")

		if tc.readErr != nil {
			g.Expect(err).To(MatchError(tc.readErr), tc.name+": the read failure must be wrapped")
		}

		g.Expect(lockCalls).To(Equal(0), tc.name+": the vault lock must not be taken")
		g.Expect(writes).To(Equal(0), tc.name+": nothing may be written")
		g.Expect(stdout.String()).To(BeEmpty(), tc.name+": nothing may be reported as done")
	}
}

// Test792b20c5a1_VocabularyEngine_WrittenNoteIsTagged verifies that tagging a written note that has a sidecar, in a
// vault with a term close to it, writes the note with that term's vocab/ tag, keeps the note's other tags and
// leaves a term far from the note untagged.
func Test792b20c5a1_VocabularyEngine_WrittenNoteIsTagged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		vault    = "/vault"
		notePath = "/vault/1.2026-01-01.written-note.md"
		closeVec = float32(0.95)
	)

	note := "---\ntype: fact\nsituation: s\nsubject: subj\npredicate: covers\nobject: obj\n" +
		"tags:\n    - project/engram\nluhmann: \"1\"\n---\n\nInformation learned: ...\n"
	sidecar := mustMarshalSidecarWithBodyVector(t, []float32{1, 0})

	loadTerms := func(gotVault string) ([]cli.TermWithVector, error) {
		g.Expect(gotVault).To(Equal(vault), "the terms are loaded from the note's vault")

		return []cli.TermWithVector{
			{Term: "eval-methodology", Vector: makeUnitVec(closeVec)},
			{Term: "unrelated-term", Vector: []float32{0, 1}},
		}, nil
	}

	read := func(path string) ([]byte, error) {
		if path == "/vault/1.2026-01-01.written-note.vec.json" {
			return sidecar, nil
		}

		return nil, &testNotFoundError{path: path}
	}

	var (
		writtenPaths []string
		writtenData  []string
	)

	write := func(path string, data []byte) error {
		writtenPaths = append(writtenPaths, path)
		writtenData = append(writtenData, string(data))

		return nil
	}

	cli.ExportApplyVocabAssignmentCore(loadTerms, read, write, nil, vault, notePath, note, "test")

	g.Expect(writtenPaths).To(Equal([]string{notePath}), "the note is written once, at its own path")

	if len(writtenData) != 1 {
		return
	}

	g.Expect(writtenData[0]).To(ContainSubstring("tags:\n    - project/engram\n    - vocab/eval-methodology\n"))
	g.Expect(writtenData[0]).NotTo(ContainSubstring("vocab/unrelated-term"))
	g.Expect(writtenData[0]).To(HaveSuffix("\n---\n\nInformation learned: ...\n"), "the body is untouched")
}
