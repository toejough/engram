package cli_test

// Field-survival tests for the exchange fields (vault-note-identity ADDED
// requirement "Exchange fields SHALL survive every frontmatter rewrite"):
// xid, parent, aliases and offer must come out of every non-exchange
// frontmatter rewrite byte-unchanged, on every note type that carries them.
// Siblings of skillfields_survival_test.go (task 3.2), reusing its in-memory
// vault (skillSurvivalFS) and raw-block comparison (yamlKeyBlock).
//
// Renames are the one licensed change: a rename appends the old basename to
// the renamed note's own aliases (task 3.5), so the renamed-note cases assert
// the original aliases block survives as a prefix.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestExchangeFieldsSurvive_Amend covers every amend flag family on every
// note type (amend.go applyTypedAmend / applyRunbookAmend, both of which
// re-marshal the typed frontmatter doc).
func TestExchangeFieldsSurvive_Amend(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		pending := false

		for name, args := range map[string]cli.AmendArgs{
			"clear-pending": {Pending: &pending},
			"activate":      {Activate: true},
			"supersedes":    {Supersedes: []string{exchangeSurvivalSupersededBase + "|narrows|narrower claim"}},
			"content":       exchangeSurvivalContentAmend(noteType),
		} {
			t.Run(noteType+"/"+name, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())

				var (
					written []byte
					writes  int
				)

				args.Vault, args.Target = "/vault", "1aa"

				var buf bytes.Buffer

				err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &buf)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(writes).To(BeNumerically(">=", 1), "amend must actually have rewritten the note")
				expectExchangeFieldsUnchanged(g, before, string(written))
			})
		}
	}
}

// TestExchangeFieldsSurvive_AmendProperty is the property form over random
// notes: any exchange-field values, on any note type, survive the amend
// re-marshal followed by the line-based rewrites (vocab assignment, wikilink
// rewrite, reference scrub) byte-unchanged.
func TestExchangeFieldsSurvive_AmendProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		noteType := rapid.SampledFrom(exchangeSurvivalTypes).Draw(rt, "type")
		before := exchangeSurvivalNote(noteType, exchangeFixtureGen().Draw(rt, "exchange"))

		var (
			written []byte
			writes  int
		)

		pending := false
		args := cli.AmendArgs{Vault: "/vault", Target: "1aa", Pending: &pending}

		var buf bytes.Buffer

		err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &buf)
		if err != nil {
			rt.Fatalf("amend: %v", err)
		}

		assertExchangeFieldsUnchanged(rt, before, string(written))

		tagged := cli.ExportWriteVocabAssignment(string(written), []string{"new-term"})
		assertExchangeFieldsUnchanged(rt, before, tagged)

		scrubbed, _ := cli.ExportRemoveNoteReferences(tagged, map[string]bool{exchangeSurvivalSupersededBase + ".md": true})
		assertExchangeFieldsUnchanged(rt, before, scrubbed)

		fixture := newReparentFixture(map[string]string{exchangeSurvivalNoteName: scrubbed})

		_, renameErr := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
			map[string]string{exchangeSurvivalLinkTarget: "4.2026-01-01.old-target"})
		if renameErr != nil {
			rt.Fatalf("rename-and-rewrite: %v", renameErr)
		}

		assertExchangeFieldsUnchanged(rt, before, string(fixture.written["/vault/"+exchangeSurvivalNoteName]))
	})
}

// TestExchangeFieldsSurvive_AmendResituateProperty is the property form for
// resituate, which re-renders fact/feedback notes from a hand-copied field
// list rather than the parsed doc.
func TestExchangeFieldsSurvive_AmendResituateProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		noteType := rapid.SampledFrom([]string{"fact", "feedback"}).Draw(rt, "type")
		before := exchangeSurvivalNote(noteType, exchangeFixtureGen().Draw(rt, "exchange"))

		after, writes, err := runExchangeSurvivalResituate(t.Context(), before)
		if err != nil {
			rt.Fatalf("resituate: %v", err)
		}

		if writes != 1 {
			rt.Fatalf("resituate wrote %d times, want 1", writes)
		}

		assertExchangeFieldsUnchanged(rt, before, after)
	})
}

// TestExchangeFieldsSurvive_BackfillIdentity covers `engram update
// --backfill-identity` (identity_backfill.go), which re-marshals fact and
// feedback notes missing user:/vault:.
func TestExchangeFieldsSurvive_BackfillIdentity(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := strings.Replace(exchangeSurvivalNote(noteType, exchangeSurvivalFixture()),
				"user: agent@example.com\nvault: personal\n", "", 1)
			fs := newSkillSurvivalFS(map[string]string{exchangeSurvivalNoteName: before})

			stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", exchangeSurvivalIdentityDeps(fs), false)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stamped).To(Equal(1), "the backfill must actually have rewritten the note")
			expectExchangeFieldsUnchanged(g, before, fs.content(exchangeSurvivalNoteName))
		})
	}
}

// TestExchangeFieldsSurvive_ClearRemovedVocabTerms covers vocab refit's term
// removal (vocab_commands.go clearRemovedTermsFromNote).
func TestExchangeFieldsSurvive_ClearRemovedVocabTerms(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())
			fs := newSkillSurvivalFS(map[string]string{exchangeSurvivalNoteName: before})

			deps := cli.VocabDeps{
				ListMD: fs.listMD, ReadFile: fs.readFile, WriteFile: fs.writeFile,
				LogWarning: func(string, ...any) {},
			}

			g.Expect(cli.ExportClearRemovedTermsFromMembers(deps, "/vault", []string{"old-term"})).To(Succeed())

			after := fs.content(exchangeSurvivalNoteName)
			g.Expect(after).NotTo(ContainSubstring("vocab/old-term"), "the rewrite must actually have run")
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_RegisterSkillsAdopt covers register-skills adopt
// (skillreg_accept.go AdoptSkillNote -> applySkillNoteBody), both when the
// note already carries the key's slug (no rename) and when adopt renames it.
func TestExchangeFieldsSurvive_RegisterSkillsAdopt(t *testing.T) {
	t.Parallel()

	for name, basename := range map[string]string{
		"no rename": "1049.2026-09-21.skill-claude-curate",
		"rename":    "1049.2026-09-21.curate-review-pending-offers",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			before := exchangeSurvivalSkillNote(exchangeSurvivalFixture())
			vault.put(basename+".md", before)

			deps := cli.SkillAdoptDeps{
				Lock:     noLock,
				Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
				Rename:   skillAcceptRenameDeps(vault),
				Embedder: skillAcceptFakeEmbedder{},
			}

			var stdout bytes.Buffer

			skill := installedEngramSkill("curate", []byte("# Curate\n\n1. Judge offers.\n"))
			g.Expect(cli.AdoptSkillNote(t.Context(), "/vault", skill, "1049", deps, &stdout)).To(Succeed())

			after, found := vault.get("1049.2026-09-21.skill-claude-curate.md")
			g.Expect(found).To(BeTrue())
			g.Expect(after).To(ContainSubstring("Judge offers."), "adopt must actually have re-rendered the note")

			if basename == "1049.2026-09-21.skill-claude-curate" {
				expectExchangeFieldsUnchanged(g, before, after)

				return
			}

			expectExchangeFieldsSurviveRename(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_RegisterSkillsRefresh covers register-skills
// refresh (skillreg_accept.go RefreshSkill -> applySkillNoteBody).
func TestExchangeFieldsSurvive_RegisterSkillsRefresh(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	basename := "1049.2026-09-21.skill-claude-curate"
	before := exchangeSurvivalSkillNote(exchangeSurvivalFixture())
	vault.put(basename+".md", before)

	var stdout bytes.Buffer

	skill := installedEngramSkill("curate", []byte("# Curate\n\nRevised.\n"))
	g.Expect(cli.RefreshSkill(t.Context(), "/vault", skill, basename, skillAcceptDeps(vault), &stdout)).To(Succeed())

	after, _ := vault.get(basename + ".md")
	g.Expect(after).To(ContainSubstring("Revised."), "refresh must actually have re-rendered the note")
	expectExchangeFieldsUnchanged(g, before, after)
}

// TestExchangeFieldsSurvive_RemoveDeletedNoteReferences covers the per-note
// reference scrub (vocab_apply.go removeNoteReferences).
func TestExchangeFieldsSurvive_RemoveDeletedNoteReferences(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())

			after, changed := cli.ExportRemoveNoteReferences(before,
				map[string]bool{exchangeSurvivalSupersededBase + ".md": true})
			g.Expect(changed).To(BeTrue(), "the scrub must actually have run")
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_ReparentRenamesNote covers Luhmann reparenting
// renaming the note itself (luhmann_reparent.go renameOneNote ->
// rewriteLuhmannIDField).
func TestExchangeFieldsSurvive_ReparentRenamesNote(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())
			fixture := newReparentFixture(map[string]string{exchangeSurvivalNoteName: before})
			oldBasename := strings.TrimSuffix(exchangeSurvivalNoteName, ".md")
			newBasename := "3.2026-01-01.exchanged"

			_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
				map[string]string{oldBasename: newBasename})
			g.Expect(err).NotTo(HaveOccurred())

			after := string(fixture.written["/vault/"+newBasename+".md"])
			g.Expect(after).To(ContainSubstring(`luhmann: "3"`), "the rename must actually have rewritten the note")
			expectExchangeFieldsSurviveRename(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_Resituate covers `engram resituate`
// (resituate.go rerenderFact / rerenderFeedback), which rebuilds the
// frontmatter from a hand-copied field list.
func TestExchangeFieldsSurvive_Resituate(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())

			after, writes, err := runExchangeSurvivalResituate(t.Context(), before)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(writes).To(Equal(1))
			g.Expect(after).To(ContainSubstring("situation: resituated context"))
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_ScrubDeletedNoteReferences covers the vault-wide
// scrub driver (vocab_apply.go scrubDeletedNoteReferences).
func TestExchangeFieldsSurvive_ScrubDeletedNoteReferences(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())
			fs := newSkillSurvivalFS(map[string]string{exchangeSurvivalNoteName: before})

			deps := cli.VocabDeps{
				ListMD: fs.listMD, ReadFile: fs.readFile, WriteFile: fs.writeFile,
				LogWarning: func(string, ...any) {},
			}

			cli.ExportScrubDeletedNoteReferences(deps, "/vault", fs.names, []string{exchangeSurvivalSupersededBase + ".md"})

			g.Expect(fs.writes).To(HaveLen(1), "the scrub must actually have rewritten the note")
			expectExchangeFieldsUnchanged(g, before, fs.content(exchangeSurvivalNoteName))
		})
	}
}

// TestExchangeFieldsSurvive_StripLegacyVocabChannel covers `engram update
// --regen-vocab`'s legacy cleanup (vocab_regen.go stripLegacyVocabChannel).
func TestExchangeFieldsSurvive_StripLegacyVocabChannel(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := strings.Replace(exchangeSurvivalNote(noteType, exchangeSurvivalFixture()),
				"tier: L2\n", "tier: L2\nvocab: old-term\n", 1)

			after, changed := cli.ExportStripLegacyVocabChannel(before)
			g.Expect(changed).To(BeTrue(), "the strip must actually have run")
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_VocabAssignment covers vocab term assignment
// (vocab.go WriteVocabAssignment -> rewriteTagsFrontmatterSplit).
func TestExchangeFieldsSurvive_VocabAssignment(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())

			after := cli.ExportWriteVocabAssignment(before, []string{"new-term", "other-term"})
			g.Expect(after).To(ContainSubstring("vocab/new-term"), "the assignment must actually have run")
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// TestExchangeFieldsSurvive_VocabSelfTag covers `engram vocab
// tag-definitions` (vocab_commands.go runVocabTagDefinitions) adding the
// missing vocab/<term> self-tag to a definition note that carries exchange
// fields.
func TestExchangeFieldsSurvive_VocabSelfTag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const name = "5a.2026-01-01.vocab-widget-definition.md"

	before := strings.Replace(exchangeSurvivalNote("fact", exchangeSurvivalFixture()),
		"tags:\n    - vocab/old-term\n", "tags:\n    - vocab\n", 1)
	fs := newSkillSurvivalFS(map[string]string{name: before})

	deps := cli.VocabDeps{
		Lock:   func(string) (func(), error) { return func() {}, nil },
		ListMD: fs.listMD, ReadFile: fs.readFile, WriteFile: fs.writeFile,
		LogWarning: func(string, ...any) {},
	}

	var buf bytes.Buffer

	g.Expect(cli.ExportRunVocabTagDefinitions(t.Context(), "/vault", deps, &buf)).To(Succeed())

	after := fs.content(name)
	g.Expect(after).To(ContainSubstring("vocab/widget"), "the self-tag must actually have been added")
	expectExchangeFieldsUnchanged(g, before, after)
}

// TestExchangeFieldsSurvive_VocabVersionStamp covers the vocab_version
// rewrite on the family note (vocab_commands.go
// writeVocabVersionToFamilyNote -> rewriteVocabVersionKey).
func TestExchangeFieldsSurvive_VocabVersionStamp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const name = "5.2026-01-01.vocab-definition.md"

	before := strings.Replace(exchangeSurvivalNote("fact", exchangeSurvivalFixture()),
		"tags:\n    - vocab/old-term\n", "vocab_version: \"1.0\"\ntags:\n    - vocab\n", 1)
	fs := newSkillSurvivalFS(map[string]string{name: before})

	g.Expect(cli.ExportWriteVocabVersionToFamilyNote("/vault", "1.1", fs.listMD, fs.readFile, fs.writeFile)).
		To(Succeed())

	after := fs.content(name)
	g.Expect(after).To(ContainSubstring(`vocab_version: "1.1"`), "the stamp must actually have run")
	expectExchangeFieldsUnchanged(g, before, after)
}

// TestExchangeFieldsSurvive_WikilinkRewrite covers rename-and-rewrite of the
// references inside a note that is not itself renamed
// (luhmann_reparent.go rewriteNoteReferences). The rename map also renames
// local notes that happen to share basenames with the note's parent link,
// alias and offer.for values: those name notes in other vaults (or this
// note's own former names), never local references, so a local rename must
// leave them untouched.
func TestExchangeFieldsSurvive_WikilinkRewrite(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			exchange := exchangeSurvivalFixture()
			before := exchangeSurvivalNote(noteType, exchange)
			target := "---\ntype: fact\nluhmann: \"2\"\n---\n\nTarget.\n"
			fixture := newReparentFixture(map[string]string{
				exchangeSurvivalNoteName:              before,
				exchangeSurvivalLinkTarget + ".md":    target,
				exchange.Parent.Links[0].Note + ".md": target,
				exchange.Parent.Links[1].Note + ".md": target,
				exchange.Aliases[0] + ".md":           target,
			})

			_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault", map[string]string{
				exchangeSurvivalLinkTarget:    "4.2026-01-01.old-target",
				exchange.Parent.Links[0].Note: "40.2026-01-01.renamed-a",
				exchange.Parent.Links[1].Note: "41.2026-01-01.renamed-b",
				exchange.Aliases[0]:           "42.2026-01-01.renamed-c",
			})
			g.Expect(err).NotTo(HaveOccurred())

			after := string(fixture.written["/vault/"+exchangeSurvivalNoteName])
			g.Expect(after).To(ContainSubstring("[[4.2026-01-01.old-target]]"),
				"the wikilink must actually have been rewritten")
			expectExchangeFieldsUnchanged(g, before, after)
		})
	}
}

// unexported constants.
const (
	exchangeSurvivalLinkTarget     = "2.2026-01-01.old-target"
	exchangeSurvivalNoteName       = "1aa.2026-01-01.rb.md"
	exchangeSurvivalSupersededBase = "9.2026-01-01.deleted-def"
)

// unexported variables.
var (
	exchangeFieldKeys     = []string{"xid", "parent", "aliases", "offer"}
	exchangeSurvivalTypes = []string{"fact", "feedback", "runbook"}
)

// exchangeFixture mirrors the exchange frontmatter fields (design D4) with
// the same keys and key order the frontmatter writer uses, so a fixture
// rendered from it is exactly what a typed re-marshal emits.
type exchangeFixture struct {
	XID     string                `yaml:"xid,omitempty"`
	Parent  exchangeFixtureParent `yaml:"parent,omitempty"`
	Aliases []string              `yaml:"aliases,omitempty"`
	Offer   exchangeFixtureOffer  `yaml:"offer,omitempty"`
}

type exchangeFixtureAuthor struct {
	Repo  string `yaml:"repo,omitempty"`
	User  string `yaml:"user,omitempty"`
	Vault string `yaml:"vault,omitempty"`
}

type exchangeFixtureLink struct {
	Note string `yaml:"note,omitempty"`
	Via  string `yaml:"via,omitempty"`
	Hash string `yaml:"hash,omitempty"`
}

type exchangeFixtureOffer struct {
	Origin string   `yaml:"origin,omitempty"`
	Key    string   `yaml:"key,omitempty"`
	For    string   `yaml:"for,omitempty"`
	Path   []string `yaml:"path,omitempty"`
}

type exchangeFixtureParent struct {
	Vault  string                `yaml:"vault,omitempty"`
	Links  []exchangeFixtureLink `yaml:"links,omitempty"`
	Author exchangeFixtureAuthor `yaml:"author,omitempty"`
}

// assertExchangeFieldsUnchanged is expectExchangeFieldsUnchanged for rapid.
func assertExchangeFieldsUnchanged(rt *rapid.T, before, after string) {
	for _, key := range exchangeFieldKeys {
		want := yamlKeyBlock(before, key)
		if want == "" {
			rt.Fatalf("fixture has no %s block:\n%s", key, before)
		}

		if got := yamlKeyBlock(after, key); got != want {
			rt.Fatalf("%s changed: got %q want %q\nfull:\n%s", key, got, want, after)
		}
	}
}

// basenameGen draws vault note basenames (<luhmann>.<date>.<slug>).
func basenameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[1-9][0-9]{0,3}[a-z]{0,2}\.2026-0[1-9]-[12][0-9]\.[a-z][a-z0-9-]{0,24}`)
}

// exchangeFixtureGen draws exchange fields of every shape design D4 allows:
// a primary link plus covered links, an author, aliases, and an offer with a
// path.
func exchangeFixtureGen() *rapid.Generator[exchangeFixture] {
	hex32 := rapid.StringMatching(`[0-9a-f]{32}`)
	hash := rapid.StringMatching(`xh1:[0-9a-f]{64}`)
	text := rapid.StringMatching(`[a-zA-Z0-9@._:/ -]{1,30}`)

	return rapid.Custom(func(rt *rapid.T) exchangeFixture {
		covered := rapid.IntRange(0, 3).Draw(rt, "covered")
		links := make([]exchangeFixtureLink, 0, covered+1)
		links = append(links, exchangeFixtureLink{
			Note: basenameGen().Draw(rt, "primary"),
			Via:  rapid.SampledFrom([]string{"offered", "pulled"}).Draw(rt, "via"),
			Hash: hash.Draw(rt, "primaryHash"),
		})

		for i := range covered {
			links = append(links, exchangeFixtureLink{
				Note: basenameGen().Draw(rt, fmt.Sprintf("covered%d", i)),
				Via:  "covered",
				Hash: hash.Draw(rt, fmt.Sprintf("coveredHash%d", i)),
			})
		}

		return exchangeFixture{
			XID: hex32.Draw(rt, "xid"),
			Parent: exchangeFixtureParent{
				Vault: hex32.Draw(rt, "parentVault"),
				Links: links,
				Author: exchangeFixtureAuthor{
					Repo: text.Draw(rt, "repo"), User: text.Draw(rt, "user"), Vault: text.Draw(rt, "vault"),
				},
			},
			Aliases: rapid.SliceOfN(basenameGen(), 1, 4).Draw(rt, "aliases"),
			Offer: exchangeFixtureOffer{
				Origin: hex32.Draw(rt, "originVault") + ":" + hex32.Draw(rt, "originXID"),
				Key:    hash.Draw(rt, "key"),
				For:    basenameGen().Draw(rt, "for"),
				Path:   rapid.SliceOfN(hex32, 1, 3).Draw(rt, "path"),
			},
		}
	})
}

// exchangeSurvivalContentAmend returns a content amend for noteType.
func exchangeSurvivalContentAmend(noteType string) cli.AmendArgs {
	switch noteType {
	case "fact":
		return cli.AmendArgs{Object: "a sharper object"}
	case "feedback":
		return cli.AmendArgs{Impact: "a sharper impact"}
	default:
		return cli.AmendArgs{DoneWhen: "a sharper done-when"}
	}
}

// exchangeSurvivalFixture is a fixed set of exchange fields with a primary
// and a covered link, an author, two aliases and an offer.
func exchangeSurvivalFixture() exchangeFixture {
	return exchangeFixture{
		XID: "7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e",
		Parent: exchangeFixtureParent{
			Vault: "9a1e2b3c4d5e6f708192a3b4c5d6e7f8",
			Links: []exchangeFixtureLink{
				{Note: "1100.2026-09-27.x", Via: "offered", Hash: "xh1:" + strings.Repeat("ab", 32)},
				{Note: "812.2026-08-01.y", Via: "covered", Hash: "xh1:" + strings.Repeat("cd", 32)},
			},
			Author: exchangeFixtureAuthor{Repo: "github.com/acme/widgets", User: "alice@example.com", Vault: "team"},
		},
		Aliases: []string{"1101.2026-09-27.z", "12.2026-06-01.old-name"},
		Offer: exchangeFixtureOffer{
			Origin: "0123456789abcdef0123456789abcdef:fedcba9876543210fedcba9876543210",
			Key:    "xh1:" + strings.Repeat("ef", 32),
			For:    "1100.2026-09-27.x",
			Path:   []string{"0123456789abcdef0123456789abcdef", "9a1e2b3c4d5e6f708192a3b4c5d6e7f8"},
		},
	}
}

func exchangeSurvivalIdentityDeps(fs *skillSurvivalFS) cli.IdentityDeps {
	return cli.IdentityDeps{
		Lock:       func(string) (func(), error) { return func() {}, nil },
		ListMD:     fs.listMD,
		ReadFile:   fs.readFile,
		WriteFile:  fs.writeFile,
		DetectRepo: func(context.Context) string { return "" },
		DetectUser: func(context.Context) string { return "agent@example.com" },
		Getenv:     func(string) string { return "" },
	}
}

// exchangeSurvivalNote renders a note of noteType carrying the exchange
// fields plus every structure a rewrite site touches: a wikilink, a vocab
// tag, a supersedes entry (frontmatter and body line), and pending: true.
func exchangeSurvivalNote(noteType string, exchange exchangeFixture) string {
	block, _ := yaml.Marshal(exchange)

	var content, body string

	switch noteType {
	case "fact":
		content = "situation: working on [[" + exchangeSurvivalLinkTarget + "]]\n" +
			"subject: the widget\npredicate: uses\nobject: a gear\n"
		body = "Information learned: when working on [[" + exchangeSurvivalLinkTarget + "]], the widget uses a gear.\n"
	case "feedback":
		content = "situation: working on [[" + exchangeSurvivalLinkTarget + "]]\n" +
			"behavior: skipped it\nimpact: it broke\naction: do it\n"
		body = "Lesson learned: when working on [[" + exchangeSurvivalLinkTarget + "]], do it.\n"
	default:
		content = "situation: releasing a widget\ndone_when: it ships\n" +
			"red_flags:\n    - skipped the checklist in [[" + exchangeSurvivalLinkTarget + "]]\n"
		body = "1. step one, see [[" + exchangeSurvivalLinkTarget + "]]\n"
	}

	return "---\n" +
		"type: " + noteType + "\n" +
		"tier: L2\n" +
		content +
		"luhmann: \"1aa\"\n" +
		"created: \"2026-01-01\"\n" +
		"source: test\n" +
		"user: agent@example.com\n" +
		"vault: personal\n" +
		"pending: true\n" +
		"tags:\n    - vocab/old-term\n" +
		"supersedes:\n    - note: " + exchangeSurvivalSupersededBase + ".md\n      type: narrows\n      claim: old claim\n" +
		string(block) +
		"---\n\n" +
		body + "\n" +
		"Supersedes: [[" + exchangeSurvivalSupersededBase + "]] — narrows: old claim\n"
}

// exchangeSurvivalSkillNote renders a registered skill runbook note carrying
// the exchange fields.
func exchangeSurvivalSkillNote(exchange exchangeFixture) string {
	block, _ := yaml.Marshal(exchange)

	return "---\n" +
		"type: runbook\n" +
		"situation: reviewing pending offers\n" +
		"done_when: every offer is judged\n" +
		"luhmann: \"1049\"\n" +
		"created: \"2026-09-21\"\n" +
		"source: s\n" +
		"user: u\n" +
		"vault: v\n" +
		"skill_hash: " + skillSurvivalHash + "\n" +
		"skill_key: claude:curate\n" +
		"skill_source: ~/.claude/engram/skills/curate/SKILL.md\n" +
		string(block) +
		"---\n\n" +
		"old curate body\n"
}

// expectExchangeFieldsSurviveRename asserts the renamed note keeps xid,
// parent and offer byte-unchanged and keeps its original aliases as a prefix
// (a rename may append the old basename, task 3.5).
func expectExchangeFieldsSurviveRename(g Gomega, before, after string) {
	for _, key := range []string{"xid", "parent", "offer"} {
		want := yamlKeyBlock(before, key)
		g.Expect(want).NotTo(BeEmpty(), "fixture must carry "+key)
		g.Expect(yamlKeyBlock(after, key)).To(Equal(want), key+" must be byte-unchanged")
	}

	wantAliases := yamlKeyBlock(before, "aliases")
	g.Expect(wantAliases).NotTo(BeEmpty(), "fixture must carry aliases")
	g.Expect(yamlKeyBlock(after, "aliases")).To(HavePrefix(wantAliases), "the original aliases must be kept")
}

// expectExchangeFieldsUnchanged asserts every exchange field's raw YAML block
// in after is byte-identical to its block in before (which must carry all
// four).
func expectExchangeFieldsUnchanged(g Gomega, before, after string) {
	for _, key := range exchangeFieldKeys {
		want := yamlKeyBlock(before, key)
		g.Expect(want).NotTo(BeEmpty(), "fixture must carry "+key)
		g.Expect(yamlKeyBlock(after, key)).To(Equal(want), key+" must be byte-unchanged")
	}
}

// runExchangeSurvivalResituate resituates before (as note 1aa) and returns
// the written note content and the note write count (sidecar writes are not
// counted).
func runExchangeSurvivalResituate(ctx context.Context, before string) (string, int, error) {
	var (
		written []byte
		writes  int
	)

	deps := cli.ResituateDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{{Basename: exchangeSurvivalNoteName, LuhmannID: "1aa"}}, nil
		},
		Read: func(string) ([]byte, error) { return []byte(before), nil },
		Write: func(path string, data []byte) error {
			if strings.HasSuffix(path, ".md") {
				written = data
				writes++
			}

			return nil
		},
		Now:      func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		Embedder: skillAcceptFakeEmbedder{},
	}

	var buf bytes.Buffer

	err := cli.RunResituate(ctx, cli.ResituateArgs{Vault: "/vault", Note: "1aa", Situation: "resituated context"},
		deps, &buf)

	return string(written), writes, err
}
