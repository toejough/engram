package cli_test

// Amend converts CRLF notes (#789 defect 1, design D1; spec
// vault-note-identity "Exchange fields SHALL survive every frontmatter
// rewrite"): amend reads every note it edits as LF, writes a note only
// where it already writes one, and re-embeds a converted note.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestRunAmend_CRLFJudgedVersionUsesTheLFForm: a CRLF served offer's
// offer.origin is read from its LF form, so --clear-pending needs the
// judged hash, and that hash is the LF form's.
func TestRunAmend_CRLFJudgedVersionUsesTheLFForm(t *testing.T) {
	t.Parallel()

	lfNote := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)
	notPending := false

	t.Run("required", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newCRLFAmendVault(toCRLF(lfNote))
		err := vault.amend(t.Context(), cli.AmendArgs{Pending: &notPending})
		g.Expect(err).To(MatchError(ContainSubstring("--expect-hash")))
		g.Expect(vault.files[crlfAmendNoteName]).To(Equal([]byte(toCRLF(lfNote))), "a refused amend writes nothing")
	})

	t.Run("matches the LF form", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		judged, hashErr := cli.ExportExchangeHash([]byte(lfNote))
		g.Expect(hashErr).NotTo(HaveOccurred())

		vault := newCRLFAmendVault(toCRLF(lfNote))
		err := vault.amend(t.Context(), cli.AmendArgs{Pending: &notPending, ExpectHash: judged})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(vault.files[crlfAmendNoteName])).NotTo(ContainSubstring("\r"))
		g.Expect(string(vault.files[crlfAmendNoteName])).NotTo(ContainSubstring("pending: true"))
	})
}

// TestRunAmend_CRLFOutputEqualsLFOutputProperty: for every frontmatter-
// writing amend kind on fact, feedback and runbook notes, and every CRLF
// layout, amend of the CRLF form writes exactly what amend of the LF form
// writes, with no CRLF left.
func TestRunAmend_CRLFOutputEqualsLFOutputProperty(t *testing.T) {
	t.Parallel()

	cases := amendParityCases()

	rapid.Check(t, func(rt *rapid.T) {
		tc := rapid.SampledFrom(cases).Draw(rt, "case")
		layout := rapid.SampledFrom([]string{"all", "frontmatter", "body"}).Draw(rt, "layout")

		lfOut, lfErr := runAmendLFJudged(context.Background(), tc.input, tc.input, tc.args)
		if lfErr != nil {
			rt.Fatalf("LF amend %s: %v", tc.name, lfErr)
		}

		crlfOut, crlfErr := runAmendLFJudged(context.Background(), crlfLayout(tc.input, layout), tc.input, tc.args)
		if crlfErr != nil {
			rt.Fatalf("%s CRLF (%s) amend refused: %v", tc.name, layout, crlfErr)
		}

		if strings.Contains(crlfOut, "\r\n") {
			rt.Fatalf("%s (%s): CRLF left in the written note:\n%q", tc.name, layout, crlfOut)
		}

		if crlfOut != lfOut {
			rt.Fatalf("%s (%s): CRLF amend differs from LF amend\ncrlf:\n%s\nlf:\n%s", tc.name, layout, crlfOut, lfOut)
		}
	})
}

// TestRunAmend_ConvertsCRLFNote: every amend kind that writes the target
// succeeds on an all-CRLF note, writes it as LF — byte-identical to the
// same amend of the LF note — and leaves a fresh sidecar, even for kinds
// that do not otherwise re-embed.
func TestRunAmend_ConvertsCRLFNote(t *testing.T) {
	t.Parallel()

	notPending := false

	kinds := []struct {
		name string
		args cli.AmendArgs
	}{
		{"object", cli.AmendArgs{Object: "a sharper object"}},
		{"supersedes", cli.AmendArgs{Supersedes: []string{"5.2026-01-01.older|narrows|an older claim"}}},
		{"activate", cli.AmendArgs{Activate: true}},
		{"clear-pending", cli.AmendArgs{Pending: &notPending}},
	}

	for _, kind := range kinds {
		t.Run(kind.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			lfNote := amendParityMinimalNote("fact")
			lfNote = strings.Replace(lfNote, "source: test\n", "source: test\npending: true\n", 1)

			crlf := newCRLFAmendVault(toCRLF(lfNote))
			lfVault := newCRLFAmendVault(lfNote)

			crlfErr := crlf.amend(t.Context(), kind.args)
			g.Expect(crlfErr).NotTo(HaveOccurred(), "amend must not refuse a CRLF note")

			lfErr := lfVault.amend(t.Context(), kind.args)
			g.Expect(lfErr).NotTo(HaveOccurred())

			written := crlf.files[crlfAmendNoteName]
			g.Expect(string(written)).NotTo(ContainSubstring("\r\n"))
			g.Expect(string(written)).To(Equal(string(lfVault.files[crlfAmendNoteName])))

			sidecar, sidecarErr := embed.UnmarshalSidecar(crlf.files[crlfAmendSidecarName])
			g.Expect(sidecarErr).NotTo(HaveOccurred())

			if sidecarErr != nil {
				return
			}

			g.Expect(sidecar.ContentHash).To(Equal(embed.ContentHash(written)),
				"a converted note is re-embedded, so its sidecar is fresh")
		})
	}
}

// TestRunAmend_DiscardIntoConvertsCRLFExisting: a fold that changes a CRLF
// E writes E as LF and re-embeds it; a fold that changes nothing leaves a
// CRLF E byte-identical.
func TestRunAmend_DiscardIntoConvertsCRLFExisting(t *testing.T) {
	t.Parallel()

	t.Run("changed", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newFoldVault()
		vault.put(foldExistingName, toCRLF(foldNote(foldExchange{}, false)))
		vault.put(foldOfferName, foldNote(foldExchange{
			xid: foldXIDO, vault: foldParentVault,
			links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
		}, true))

		deps := vault.deps()
		deps.Embedder = stubEmbedder{modelID: "stub@4", dims: 4}

		err := cli.RunAmend(t.Context(), cli.AmendArgs{
			Vault: "/vault", Target: foldOfferBase, Discard: true, Into: foldExistingBase,
		}, deps, &bytes.Buffer{})
		g.Expect(err).NotTo(HaveOccurred(), "a fold must not refuse a CRLF existing note")

		written := vault.files[foldExistingName]
		g.Expect(string(written)).NotTo(ContainSubstring("\r\n"))
		g.Expect(vault.files).NotTo(HaveKey(foldOfferName))

		folded := decodeFoldExchange(t, written)
		g.Expect(folded.Aliases).To(Equal([]string{foldOfferBase}))

		sidecar, sidecarErr := embed.UnmarshalSidecar(vault.files[strings.TrimSuffix(foldExistingName, ".md")+".vec.json"])
		g.Expect(sidecarErr).NotTo(HaveOccurred())

		if sidecarErr != nil {
			return
		}

		g.Expect(sidecar.ContentHash).To(Equal(embed.ContentHash(written)), "a converted E is re-embedded")
	})

	t.Run("unchanged", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		existing := toCRLF(foldNote(foldExchange{aliases: []string{foldOfferBase}}, false))

		vault := newFoldVault()
		vault.put(foldExistingName, existing)
		vault.put(foldOfferName, foldNote(foldExchange{xid: foldXIDO}, true))

		err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(vault.files[foldExistingName])).To(Equal(existing), "a fold that changes nothing writes nothing")
		g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	})
}

// TestRunAmend_DiscardRecordsDeclineOfCRLFPulledNote: a bare discard reads
// a CRLF pulled note as LF, so its decline is still recorded.
func TestRunAmend_DiscardRecordsDeclineOfCRLFPulledNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	vault.put(foldOfferName, toCRLF(foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault,
		links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true)))

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.declines).To(Equal([]cli.ExportDeclinedPull{
		{Vault: foldParentVault, Basename: "7.2026-09-01.parent", Hash: "xh1:parent"},
	}))
}

// unexported constants.
const (
	crlfAmendNoteName    = "1aa.2026-01-01.rb.md"
	crlfAmendSidecarName = "1aa.2026-01-01.rb.vec.json"
)

// crlfAmendVault is a one-note in-memory vault with an embedder, for the
// CRLF amend tests.
type crlfAmendVault struct {
	files map[string][]byte
}

func (v *crlfAmendVault) amend(ctx context.Context, args cli.AmendArgs) error {
	args.Vault, args.Target, args.VaultName = "/vault", "1aa", "personal"

	deps := cli.AmendDeps{
		DetectRepo: func(context.Context) string { return "github.com/acme/widgets" },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Scan:       crlfAmendScan,
		Read: func(path string) ([]byte, error) {
			return v.files[path[strings.LastIndex(path, "/")+1:]], nil
		},
		Write: func(path string, data []byte) error {
			v.files[path[strings.LastIndex(path, "/")+1:]] = data

			return nil
		},
		Embedder: stubEmbedder{modelID: "stub@4", dims: 4},
		Now:      crlfAmendNow,
	}

	return cli.RunAmend(ctx, args, deps, &bytes.Buffer{})
}

func crlfAmendNow() time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) }

func crlfAmendScan(string) ([]vaultgraph.Note, error) {
	return []vaultgraph.Note{{Basename: strings.TrimSuffix(crlfAmendNoteName, ".md"), LuhmannID: "1aa"}}, nil
}

// crlfLayout is note with CRLF line endings in the given part: "all",
// "frontmatter" (through the closing delimiter) or "body" (after it).
func crlfLayout(note, layout string) string {
	const closing = "\n---\n"

	end := strings.Index(note, closing) + len(closing)

	switch layout {
	case "frontmatter":
		return toCRLF(note[:end]) + note[end:]
	case "body":
		return note[:end] + toCRLF(note[end:])
	default:
		return toCRLF(note)
	}
}

func newCRLFAmendVault(note string) *crlfAmendVault {
	return &crlfAmendVault{files: map[string][]byte{
		crlfAmendNoteName: []byte(note),
		crlfAmendSidecarName: embed.MarshalSidecar(embed.Sidecar{
			SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "stub@4", Dims: 4,
			SituationVector: []float32{0, 0, 0, 0}, BodyVector: []float32{0, 0, 0, 0},
			ContentHash: embed.ContentHash([]byte(note)), LastUsed: "2026-01-01",
		}),
	}}
}

// runAmendLFJudged amends input like runAmendParityCase, but judges the
// version on lfInput — the form amend edits (design D1).
func runAmendLFJudged(ctx context.Context, input, lfInput string, args cli.AmendArgs) (string, error) {
	vault := newCRLFAmendVault(input)

	if args.Pending != nil {
		args.ExpectHash, _ = cli.ExportExchangeHash([]byte(lfInput))
	}

	err := vault.amend(ctx, args)
	if err != nil {
		return "", fmt.Errorf("amend: %w", err)
	}

	return string(vault.files[crlfAmendNoteName]), nil
}

// toCRLF is s with every LF line ending turned into CRLF.
func toCRLF(s string) string {
	return strings.ReplaceAll(s, "\n", "\r\n")
}
