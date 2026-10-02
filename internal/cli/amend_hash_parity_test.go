package cli_test

// Amend's exchange-hash parity with the pre-change amend (ruling V6, final
// review F4): moving amend's frontmatter writes onto the YAML-node edit must
// leave every exchange hash exactly as the typed re-marshal produced it for
// notes without unknown keys. The goldens under testdata/amend_hash_parity
// were written by the pre-change amend (HEAD 49cfc120) on these same inputs,
// and cross-checked against a binary built from that commit: its
// `engram show` exchange hash matched the golden's for all 26 cases that do
// not need a chunk index (the 6 --chunk-source cases were checked
// in-process only).

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunAmend_ExchangeHashParityWithPreChangeAmend: for every amend kind on
// fact, feedback and runbook notes without unknown keys, the amended note
// hashes exactly as the pre-change amend's output did — and is in fact
// byte-identical to it, so the offer paths (which stage and compare these
// bytes and hashes) see no difference.
func TestRunAmend_ExchangeHashParityWithPreChangeAmend(t *testing.T) {
	t.Parallel()

	for _, tc := range amendParityCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, err := runAmendParityCase(t.Context(), tc)
			g.Expect(err).NotTo(HaveOccurred())

			want, readErr := os.ReadFile(filepath.Join("testdata", "amend_hash_parity", tc.name+".md"))
			g.Expect(readErr).NotTo(HaveOccurred())

			// The pre-change amend wrote user: "" and vault: "" when a
			// bookkeeping amend met a note with no identity (the typed
			// writer's fields had no omitempty). #789 never writes either
			// empty: user: is omitted (design D6, ruling W1) and vault: takes
			// the resolved vault name (design D10, ruling W2). Those two lines
			// are the only difference; every other byte, and the exchange
			// hash, match.
			want = bytes.Replace(want, []byte("user: \"\"\nvault: \"\"\n"), []byte("vault: personal\n"), 1)

			gotHash, _ := cli.ExportExchangeHash([]byte(got))
			wantHash, _ := cli.ExportExchangeHash(want)
			g.Expect(gotHash).To(Equal(wantHash), "exchange hash moved:\ngot:\n%s\nwant:\n%s", got, want)
			g.Expect(got).To(Equal(string(want)), "a note without unknown keys must be written byte-for-byte as before")
		})
	}
}

// amendParityCase is one amend of one input note.
type amendParityCase struct {
	name  string
	input string
	args  cli.AmendArgs
}

// amendParityCases crosses fact, feedback and runbook notes — each as a
// fully-populated exchange note and as a minimal note with no identity —
// with every amend kind that writes frontmatter.
func amendParityCases() []amendParityCase {
	notPending := false
	supersedes := []string{"5.2026-01-01.older|narrows|an older claim"}
	chunks := []string{"session.jsonl#a1"}

	kinds := map[string][]struct {
		name string
		args cli.AmendArgs
	}{
		"fact": {
			{"object", cli.AmendArgs{Object: "a sharper object"}},
			{"situation", cli.AmendArgs{Situation: "a new situation"}},
			{"supersedes", cli.AmendArgs{Supersedes: supersedes}},
			{"chunk-source", cli.AmendArgs{ChunkSources: chunks}},
			{"clear-pending", cli.AmendArgs{Pending: &notPending}},
		},
		"feedback": {
			{"impact", cli.AmendArgs{Impact: "a sharper impact"}},
			{"supersedes", cli.AmendArgs{Supersedes: supersedes}},
			{"chunk-source", cli.AmendArgs{ChunkSources: chunks}},
			{"clear-pending", cli.AmendArgs{Pending: &notPending}},
		},
		"runbook": {
			{"done-when", cli.AmendArgs{DoneWhen: "a sharper done-when"}},
			{"body", cli.AmendArgs{Body: "1. a new step\n2. another\n"}},
			{"red-flags", cli.AmendArgs{RedFlags: []string{"first flag", "second flag"}}},
			{"triggers", cli.AmendArgs{Triggers: []string{"a trigger"}}},
			{"supersedes", cli.AmendArgs{Supersedes: supersedes}},
			{"chunk-source", cli.AmendArgs{ChunkSources: chunks}},
			{"clear-pending", cli.AmendArgs{Pending: &notPending}},
		},
	}

	cases := make([]amendParityCase, 0, 2*len(kinds)*len(kinds["runbook"]))

	for _, noteType := range exchangeSurvivalTypes {
		inputs := map[string]string{
			"exchange": exchangeSurvivalNote(noteType, exchangeSurvivalFixture()),
			"minimal":  amendParityMinimalNote(noteType),
		}

		for inputName, input := range inputs {
			for _, kind := range kinds[noteType] {
				cases = append(cases, amendParityCase{
					name:  fmt.Sprintf("%s-%s-%s", noteType, inputName, kind.name),
					input: input,
					args:  kind.args,
				})
			}
		}
	}

	return cases
}

// amendParityMinimalNote is a note as an older engram wrote it: no identity
// fields, an unquoted created date, no exchange fields.
func amendParityMinimalNote(noteType string) string {
	content := map[string]string{
		"fact":     "situation: ctx\nsubject: A\npredicate: has\nobject: old\n",
		"feedback": "situation: ctx\nbehavior: skipped\nimpact: broke\naction: old\n",
		"runbook":  "situation: ctx\ndone_when: old\nred_flags:\n    - a flag\n",
	}[noteType]
	body := map[string]string{
		"fact":     "Information learned: when in ctx, A has old.\n",
		"feedback": "Lesson learned: when ctx, old.\n",
		"runbook":  "1. old step\n",
	}[noteType]

	return "---\ntype: " + noteType + "\n" + content +
		"luhmann: \"1aa\"\ncreated: 2026-01-01\nsource: test\n---\n\n" + body
}

// runAmendParityCase amends tc.input in memory with fixed identity
// detection and returns the written note.
func runAmendParityCase(ctx context.Context, tc amendParityCase) (string, error) {
	var written []byte

	deps := cli.AmendDeps{
		DetectRepo: func(context.Context) string { return "github.com/acme/widgets" },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: exchangeSurvivalNoteName, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".md") {
				return []byte(tc.input), nil
			}

			return []byte(`{"last_used":"2026-01-01"}`), nil
		},
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".md") {
				written = data
			}

			return nil
		},
		Now: func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
	}

	args := tc.args
	args.Vault, args.Target, args.VaultName = "/vault", "1aa", "personal"

	if args.Pending != nil {
		args.ExpectHash, _ = cli.ExportExchangeHash([]byte(tc.input))
	}

	err := cli.ExportRunAmend(ctx, args, deps, &bytes.Buffer{})

	return string(written), err
}
