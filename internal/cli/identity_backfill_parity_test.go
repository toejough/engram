package cli_test

// Identity backfill's byte parity with the pre-change backfill (#789,
// design D2): moving backfill onto the YAML-node edit must write exactly
// what the typed re-marshal wrote for notes without unknown keys or
// anchors. The goldens under testdata/backfill_identity_parity were written
// by the backfill at 0f5d91b7, before any change, on these same inputs, and
// cross-checked against a binary built from that commit.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestBackfillIdentity_ParityWithPreChangeBackfill: every input note is
// stamped byte-for-byte as the pre-change backfill stamped it, so its
// exchange hash is unchanged too.
func TestBackfillIdentity_ParityWithPreChangeBackfill(t *testing.T) {
	t.Parallel()

	for _, tc := range backfillParityCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			got, err := runBackfillParityCase(t.Context(), tc)
			g.Expect(err).NotTo(HaveOccurred())

			want, readErr := os.ReadFile(filepath.Join("testdata", "backfill_identity_parity", tc.name+".md"))
			g.Expect(readErr).NotTo(HaveOccurred())

			gotHash, _ := cli.ExportExchangeHash([]byte(got))
			wantHash, _ := cli.ExportExchangeHash(want)
			g.Expect(gotHash).To(Equal(wantHash))
			g.Expect(got).To(Equal(string(want)), "a note without unknown keys must be stamped byte-for-byte as before")
		})
	}
}

// backfillParityCase is one note missing identity, and the repo the
// environment detects for it.
type backfillParityCase struct {
	name  string
	input string
	repo  string
}

// backfillParityCases are fact and feedback notes as an engram that
// predates identity wrote them: minimal, with project:, with every
// optional modeled key, with exchange fields, with explicitly empty
// identity, and with no detectable repo.
func backfillParityCases() []backfillParityCase {
	const detected = "git@github.com:example/vault.git"

	content := map[string]struct{ fields, body string }{
		"fact": {
			"situation: working on widgets\nsubject: the widget\npredicate: uses\nobject: a gear\n",
			"Information learned: when working on widgets, the widget uses a gear.\n",
		},
		"feedback": {
			"situation: working on widgets\nbehavior: skipped it\nimpact: it broke\naction: do it\n",
			"Lesson learned: when working on widgets, do it.\n",
		},
	}

	cases := make([]backfillParityCase, 0, 2*7)

	for _, noteType := range []string{"fact", "feedback"} {
		fields, body := content[noteType].fields, content[noteType].body
		head := "---\ntype: " + noteType + "\n"
		ident := "luhmann: \"1aa\"\n"

		cases = append(cases,
			backfillParityCase{
				name:  noteType + "-minimal",
				input: head + fields + ident + "created: 2026-01-01\nsource: agent\n---\n\n" + body,
				repo:  detected,
			},
			backfillParityCase{
				name: noteType + "-project",
				input: head + fields + ident + "created: \"2026-01-01\"\nsource: agent\nproject: my-project\n---\n\n" +
					body,
				repo: detected,
			},
			backfillParityCase{
				name: noteType + "-full",
				input: head + "tier: L2\n" + fields + ident + "created: \"2026-01-01\"\nsource: agent\n" +
					"project: my-project\npending: true\nissue: \"12\"\nsources:\n    - session.jsonl#a1\n" +
					"tags:\n    - vocab/old-term\n" +
					"supersedes:\n    - note: 5.2026-01-01.older.md\n      type: narrows\n      claim: old claim\n" +
					"---\n\n" + body + "\nSupersedes: [[5.2026-01-01.older]] — narrows: old claim\n",
				repo: detected,
			},
			backfillParityCase{
				name: noteType + "-exchange",
				input: head + fields + ident + "created: \"2026-01-01\"\nsource: agent\n" +
					"xid: 7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e\n" +
					"parent:\n    vault: 9a1e2b3c4d5e6f708192a3b4c5d6e7f8\n    links:\n" +
					"        - note: 1100.2026-09-27.x\n          via: offered\n          hash: xh1:" +
					strings.Repeat("ab", 32) + "\n" +
					"aliases:\n    - 12.2026-06-01.old-name\n---\n\n" + body,
				repo: detected,
			},
			backfillParityCase{
				name:  noteType + "-empty-identity",
				input: head + fields + ident + "created: \"2026-01-01\"\nsource: agent\nuser: \"\"\nvault: \"\"\n---\n\n" + body,
				repo:  detected,
			},
			backfillParityCase{
				name:  noteType + "-no-repo",
				input: head + fields + ident + "created: \"2026-01-01\"\nsource: agent\n---\n\n" + body,
				repo:  "",
			},
			backfillParityCase{
				name: noteType + "-vocab",
				input: head + fields + ident + "created: \"2026-01-01\"\nsource: agent\n" +
					"tags:\n    - vocab/widgets\n    - vocab/gears\n---\n\n" + body,
				repo: detected,
			},
		)
	}

	return cases
}

// runBackfillParityCase backfills tc.input alone in an in-memory vault with
// fixed identity detection and returns the written note.
func runBackfillParityCase(ctx context.Context, tc backfillParityCase) (string, error) {
	const notePath = "/vault/1aa.2026-01-01.note.md"

	written := ""

	deps := cli.IdentityDeps{
		Lock:     func(string) (func(), error) { return func() {}, nil },
		ListMD:   func(string) ([]string, error) { return []string{filepath.Base(notePath)}, nil },
		ReadFile: func(string) ([]byte, error) { return []byte(tc.input), nil },
		WriteFile: func(path string, data []byte) error {
			if path == notePath {
				written = string(data)
			}

			return nil
		},
		DetectRepo: func(context.Context) string { return tc.repo },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Getenv:     func(string) string { return "" },
	}

	_, err := cli.ExportBackfillIdentity(ctx, "/vault", deps, false)

	return written, err
}
