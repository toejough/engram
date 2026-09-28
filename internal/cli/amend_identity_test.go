package cli_test

// Which amends re-stamp repo:/user:/vault: (vault-note-identity MODIFIED
// "Amend re-stamps identity fields on every write", design D10 M6/G11; task
// 3.6): content, --supersedes and --chunk-source amends re-stamp; bookkeeping
// amends (--activate alone, --clear-pending) preserve the note's declared
// identity, so an accepted offer keeps its author.

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunAmend_BookkeepingAmendsKeepIdentity: --activate alone and
// --clear-pending (alone or together) leave repo:, user: and vault: as the
// note declared them, even when the running environment detects others.
func TestRunAmend_BookkeepingAmendsKeepIdentity(t *testing.T) {
	t.Parallel()

	cleared := false

	for name, args := range map[string]cli.AmendArgs{
		"activate alone":           {Activate: true},
		"clear-pending":            {Pending: &cleared},
		"activate + clear-pending": {Activate: true, Pending: &cleared},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			written, err := runIdentityAmend(t.Context(), args)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(written).NotTo(BeEmpty(), "the bookkeeping amend must actually have rewritten the note")
			expectDeclaredIdentity(g, written)
		})
	}
}

// TestRunAmend_ClearPendingKeepsOfferAuthor is the spec scenario "Accepting
// an offer keeps its author": clearing a pending offer written by alice,
// from an environment whose detected user is bob, keeps user: alice.
func TestRunAmend_ClearPendingKeepsOfferAuthor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cleared := false

	written, err := runIdentityAmend(t.Context(), cli.AmendArgs{Pending: &cleared})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(written).To(ContainSubstring("user: alice@example.com"))
	g.Expect(written).NotTo(ContainSubstring("pending: true"))
}

// TestRunAmend_IdentityRestampProperty: over any combination of amend flags,
// identity is re-stamped exactly when a content flag, --supersedes or
// --chunk-source is present; otherwise the declared identity survives.
func TestRunAmend_IdentityRestampProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		args := cli.AmendArgs{Activate: rapid.Bool().Draw(rt, "activate")}

		if rapid.Bool().Draw(rt, "clearPending") {
			cleared := false
			args.Pending = &cleared
		}

		restamps := drawRestampingFlags(rt, &args)

		written, err := runIdentityAmend(t.Context(), args)
		if err != nil {
			rt.Fatalf("amend: %v", err)
		}

		declared := strings.Contains(written, "user: alice@example.com") &&
			strings.Contains(written, "repo: github.com/alice/widgets") &&
			strings.Contains(written, "vault: alice-vault")
		detected := strings.Contains(written, "user: bob@example.com") &&
			strings.Contains(written, "repo: github.com/bob/gadgets") &&
			strings.Contains(written, "vault: bob-vault")

		if restamps != detected || restamps == declared {
			rt.Fatalf("restamp want %v, got declared=%v detected=%v:\n%s", restamps, declared, detected, written)
		}
	})
}

// unexported constants.
const (
	identityAmendChunk = "session.jsonl#a1"
)

// drawRestampingFlags sets a random subset of the content, --supersedes and
// --chunk-source flags on args, reporting whether it set any.
func drawRestampingFlags(rt *rapid.T, args *cli.AmendArgs) bool {
	restamps := false
	text := rapid.StringMatching(`[a-z][a-z ]{0,12}[a-z]`)

	for _, flag := range []struct {
		name string
		set  func(value string)
	}{
		{"situation", func(value string) { args.Situation = value }},
		{"subject", func(value string) { args.Subject = value }},
		{"predicate", func(value string) { args.Predicate = value }},
		{"object", func(value string) { args.Object = value }},
		{"supersedes", func(value string) {
			args.Supersedes = []string{"9.2026-01-01.old|narrows|" + value}
		}},
		{"chunkSource", func(string) { args.ChunkSources = []string{identityAmendChunk} }},
	} {
		if rapid.Bool().Draw(rt, flag.name) {
			flag.set(text.Draw(rt, flag.name+"Value"))

			restamps = true
		}
	}

	return restamps
}

func expectDeclaredIdentity(g Gomega, written string) {
	g.Expect(written).To(ContainSubstring("repo: github.com/alice/widgets"))
	g.Expect(written).To(ContainSubstring("user: alice@example.com"))
	g.Expect(written).To(ContainSubstring("vault: alice-vault"))
	g.Expect(written).NotTo(ContainSubstring("bob"))
}

// runIdentityAmend amends a pending fact note declared by alice from an
// environment that detects bob, returning the written note ("" when the note
// was never written).
func runIdentityAmend(ctx context.Context, args cli.AmendArgs) (string, error) {
	const basename = "1aa.2026-01-01.offer.md"

	note := []byte("---\ntype: fact\ntier: L2\nsituation: ctx\nsubject: A\npredicate: has\nobject: B\n" +
		"luhmann: \"1aa\"\ncreated: \"2026-01-01\"\nsource: test\n" +
		"repo: github.com/alice/widgets\nuser: alice@example.com\nvault: alice-vault\npending: true\n" +
		"---\n\nInformation learned: when in ctx, A has B.\n\n")

	var written []byte

	deps := cli.AmendDeps{
		DetectRepo: func(context.Context) string { return "github.com/bob/gadgets" },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, ".md") {
				return note, nil
			}

			return []byte(`{"last_used":"2026-01-01"}`), nil
		},
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".md") {
				written = data
			}

			return nil
		},
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{identityAmendChunk: true}, nil
		},
		Now: func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
	}

	args.Vault, args.Target, args.VaultName = "/vault", "1aa", "bob-vault"

	err := cli.ExportRunAmend(ctx, args, deps, &bytes.Buffer{})

	return string(written), err
}
