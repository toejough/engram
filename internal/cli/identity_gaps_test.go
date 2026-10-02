package cli_test

// No offer without a user, and no empty vault: (#789, design D9, D10).

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestBuildOfferPayload_NoUserIdentityRefuses: a note with no user: on a
// host whose user detection is empty builds no payload, rather than
// sending user: "" for the parent to reject.
func TestBuildOfferPayload_NoUserIdentityRefuses(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := strings.Replace(string(offerTestNote{}.render(t)), "user: alice\n", "", 1)
	note := cli.OfferNoteForTest{
		XID: xidA, Basename: "1.2026-09-27.note", Raw: []byte(raw), Hash: mustHash(t, []byte(raw)),
	}

	_, err := cli.ExportBuildOfferPayload(note, seqID(60), parentVaultID, "detected-repo", "", nil)
	g.Expect(err).To(MatchError(cli.ErrOfferNoUserIdentityForTest))
}

// TestDrainOutbox_NoUserIdentityNeedsAttention: an offer that cannot be
// built for want of a user identity puts its entry in attention with one
// warning, sends nothing, stays silent on later drains, and is sent once
// a payload can be built.
func TestDrainOutbox_NoUserIdentityNeedsAttention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)

	noIdentity := cli.OfferSendResultForTest{
		Outcome: cli.ExportOfferSkipped,
		Err:     fmt.Errorf("outbox: building the offer for 1.2026-09-27.a: %w", cli.ErrOfferNoUserIdentityForTest),
	}

	for range 3 {
		parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: noIdentity}}
		_, err := env.drain(parent)
		g.Expect(err).NotTo(HaveOccurred(), "the missing identity is warned, not returned")

		box := env.outbox()
		g.Expect(box.Entries).To(HaveLen(1))
		g.Expect(box.Entries[0].State).To(Equal("attention"))
		g.Expect(box.Entries[0].LastError).To(ContainSubstring("no user identity detected; set git user.email"))
	}

	g.Expect(strings.Count(env.stderr.String(), "cannot offer")).To(Equal(1), env.stderr.String())
	g.Expect(env.stderr.String()).To(ContainSubstring("1.2026-09-27.a"))

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA}), "resumes once a payload can be built")
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestLearn_OmitsEmptyVault: a first write with no resolved vault name
// omits vault: and warns.
func TestLearn_OmitsEmptyVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		written  []byte
		warnings []string
	)

	args := cli.LearnArgs{
		Type: "fact", Slug: "no-vault", Vault: t.TempDir(), Position: "top", Source: "test",
		Situation: "no vault name", Subject: "A", Predicate: "has", Object: "B",
	}
	deps := learnDepsForUserTest(&written)
	deps.DetectUser = func(context.Context) string { return "bob@example.com" }
	deps.LogWarning = func(format string, _ ...any) { warnings = append(warnings, format) }

	err := cli.ExportRunLearn(t.Context(), args, deps, &strings.Builder{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(written)).NotTo(MatchRegexp(`(?m)^vault:`))
	g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("vault"))
}

// TestRunAmend_BookkeepingNeverWritesEmptyVault: a bookkeeping amend on a
// note with no vault: writes the resolved vault name; with none resolved
// it omits the key and warns — never vault: "".
func TestRunAmend_BookkeepingNeverWritesEmptyVault(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, vaultName, want string }{
		{"resolved", "personal", "vault: personal\n"},
		{"unresolved", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			written, warnings := runVaultlessClearPending(t.Context(), tc.vaultName)
			g.Expect(written).NotTo(ContainSubstring("vault: \"\""))

			if tc.want != "" {
				g.Expect(written).To(ContainSubstring(tc.want))
				g.Expect(warnings).To(BeEmpty())

				return
			}

			g.Expect(written).NotTo(MatchRegexp(`(?m)^vault:`))
			g.Expect(strings.Join(warnings, "\n")).To(ContainSubstring("vault"))
		})
	}
}

// TestUpdateExchange_NoUserIdentityWaitsThenOffers: through the production
// wiring, a note learned where no user is detectable is not offered (no
// request reaches the parent's /learn); update reports it as needing
// attention; once detection works, the next exchange offers it.
func TestUpdateExchange_NoUserIdentityWaitsThenOffers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.wrap = withoutUserDetection
	env.learnFact("userless")
	g.Expect(env.lastStderr).To(ContainSubstring("cannot offer"))
	g.Expect(env.parent.offers()).To(BeEmpty(), "no offer is sent without a user")

	notice := cli.ExportUpdateExchange(env.deps())(context.Background(), env.vault, false)
	g.Expect(notice).To(ContainSubstring("1 need attention"))
	g.Expect(notice).To(ContainSubstring("no user identity detected; set git user.email"))
	g.Expect(env.parent.offers()).To(BeEmpty())

	env.wrap = nil
	g.Expect(cli.ExportUpdateExchange(env.deps())(context.Background(), env.vault, false)).To(BeEmpty())
	g.Expect(env.parent.offers()).To(HaveLen(1), "offered once a user is detected")
}

// runVaultlessClearPending clears the pending marker on a note with no
// identity at all, with the given resolved vault name, and returns the
// written note and the warnings.
func runVaultlessClearPending(ctx context.Context, vaultName string) (string, []string) {
	note := "---\ntype: fact\nsituation: ctx\nsubject: A\npredicate: has\nobject: old\nluhmann: \"1aa\"\n" +
		"created: \"2026-01-01\"\nsource: test\npending: true\n---\n\nInformation learned: when in ctx, A has old.\n"

	var (
		written  string
		warnings []string
	)

	notPending := false
	deps := cli.AmendDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: "1aa.2026-01-01.n", LuhmannID: "1aa"}}, nil
		},
		Read: func(string) ([]byte, error) { return []byte(note), nil },
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".md") {
				written = string(data)
			}

			return nil
		},
		Now:        func() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) },
		LogWarning: func(format string, _ ...any) { warnings = append(warnings, format) },
	}

	_ = cli.ExportRunAmend(ctx, cli.AmendArgs{
		Vault: "/vault", Target: "1aa", VaultName: vaultName, Pending: &notPending,
	}, deps, &bytes.Buffer{})

	return written, warnings
}
