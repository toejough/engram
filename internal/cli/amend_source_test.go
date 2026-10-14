package cli_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

func Test793d38b1_AmendSourceAloneChangesNoContentSoNothingIsEmbeddedAgain(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	embedCalled := false
	note := makeFactNote("ctx", "A")
	deps := sourceAmendDeps(note, new([]byte))
	deps.Embedder = &spyEmbedder{called: &embedCalled}

	var out bytes.Buffer

	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Source: amendSourceLine}
	err := cli.ExportRunAmend(t.Context(), args, deps, &out)

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(embedCalled).To(gomega.BeFalse(), "a provenance line is not a content change")
}

// Test793d38b1_AmendSourceAloneKeepsTheDeclaredIdentity: a provenance line is bookkeeping, as --activate is
// (792b14c5): on every note type, an amend that supplies only --source appends the line and leaves repo:, user: and
// vault: as the note declared them, even when the running environment detects others.
func Test793d38b1_AmendSourceAloneKeepsTheDeclaredIdentity(t *testing.T) {
	t.Parallel()

	for _, noteType := range identityAmendTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)

			written, err := runIdentityAmend(t.Context(), noteType, cli.AmendArgs{Source: identityAmendSource})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(written).To(gomega.ContainSubstring("source: test; " + identityAmendSource))
			expectDeclaredIdentity(g, written)
		})
	}
}

func Test793d38b1_AmendSourceAppendsAProvenanceLineToTheSourceOfAnyKindOfNote(t *testing.T) {
	t.Parallel()

	notes := map[string][]byte{
		"fact":     makeFactNote("ctx", "A"),
		"feedback": makeFeedbackNote("ctx", "b", "i", "a", ""),
		"runbook":  makeRunbookNote("ctx", "done", "1. step"),
	}

	for kind, note := range notes {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			g := gomega.NewWithT(t)
			amended, err := amendWithSource(t, note, amendSourceLine)

			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(amended).To(gomega.ContainSubstring("source: test; " + amendSourceLine))
		})
	}
}

func Test793d38b1_AmendSourceIsAppendedOnceAndNeverReplacesWhatWasThere(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	once, err := amendWithSource(t, makeFactNote("ctx", "A"), amendSourceLine)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	twice, err := amendWithSource(t, []byte(once), amendSourceLine)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(twice).To(gomega.Equal(once))
	g.Expect(twice).To(gomega.ContainSubstring("source: test; "))
}

func Test793d38b1_AmendWithoutSourceLeavesTheSourceAsItWas(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	amended, err := amendWithSource(t, makeFactNote("ctx", "A"), "")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(amended).To(gomega.ContainSubstring("source: test\n"))
}

// unexported constants.
const (
	amendSourceLine = "amended automatically from offer 5.2026-10-12.offer-one.md"
)

// amendWithSource amends a note with --source and the content flag that makes the amend a revision, and returns the
// note as written.
func amendWithSource(t *testing.T, note []byte, source string) (string, error) {
	t.Helper()

	var written []byte

	var out bytes.Buffer

	args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Source: source, Situation: "ctx"}
	err := cli.ExportRunAmend(t.Context(), args, sourceAmendDeps(note, &written), &out)

	return string(written), err
}

// sourceAmendDeps are the dependencies of an amend of one note, written to `written`.
func sourceAmendDeps(note []byte, written *[]byte) cli.AmendDeps {
	const basename = "1aa.2026-01-01.test.md"

	return cli.AmendDeps{
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "" },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: basename, LuhmannID: "1aa"}}, nil
		},
		Read:  func(string) ([]byte, error) { return note, nil },
		Write: func(_ string, data []byte) error { *written = data; return nil },
		LoadChunkIDs: func(string, func(string) ([]string, error), func(string) ([]byte, error)) (map[string]bool, error) {
			return map[string]bool{}, nil
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
}
