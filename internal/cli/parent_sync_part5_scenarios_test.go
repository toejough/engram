package cli_test

// Scenario tests for the parent-sync product capability, part 5 (861c2zzb): the re-check of a linked note when
// it is activated (792a21zf), the signal a pull-down sends to the parent (792a21zg), the stable exchange ID
// (792a21zi), the vault-qualified parent links (792a21zj), the aliases a rename or a fold records (792a21zk) and
// the exchange fields every other frontmatter rewrite keeps (792a21zl). Spec: docs/specs/product/parent-sync.md.
//
// The journeys over a parent run on a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation,
// #860) and a fake parent behind Deps.Fetch, with no network and no real home. The rewrites (the fold, the
// reparenting, the vocabulary, scrub and skill-registration rewrites, the identity backfill) run on the
// in-memory vaults of their own commands (foldVault, reparentFixture, skillSurvivalFS), so they touch no file.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792a21zf1c_Activate_ADeclinedUpdateIsNotPulledAgain verifies 792a21zf1c (A declined update is not pulled
// again): the pending offer a re-check pulled down is discarded outright; activating L again with the parent
// note unchanged since writes no note and no further decline, though the parent is asked.
func Test792a21zf1c_Activate_ADeclinedUpdateIsNotPulledAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the pending offer a re-check pulled down was discarded outright
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.declined-update", "c", ""))
	local := acceptedPulledCopy(t, env, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
	env.run("activate", "--note", local)

	update := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == local })
	g.Expect(update).To(HaveLen(1), "the re-check pulled the changed note down")

	if len(update) != 1 {
		return
	}

	env.run("amend", "--target", strings.TrimSuffix(update[0], ".md"), "--discard")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.declined()).To(HaveLen(1))
	g.Expect(env.noteFiles()).To(Equal([]string{local}))

	declinedBefore := readFileString(t, filepath.Join(env.vault, ".engram", "declined.json"))
	askedBefore := len(env.parent.shown())

	// When L is activated again with the parent note unchanged since
	env.run("activate", "--note", local)

	// Then nothing is written
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.parent.shown()).To(HaveLen(askedBefore+1), "the parent was asked for the note once more")
	g.Expect(env.noteFiles()).To(Equal([]string{local}))
	g.Expect(readFileString(t, filepath.Join(env.vault, ".engram", "declined.json"))).To(Equal(declinedBefore))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21zf1d_Activate_AnUnreachableParentDoesNotFailTheLocalActivate verifies 792a21zf1d (An unreachable
// parent does not fail the local activate): with the parent down, activating the linked note succeeds, bumps its
// LastUsed, writes no note, and records the failure for the backoff.
func Test792a21zf1d_Activate_AnUnreachableParentDoesNotFailTheLocalActivate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a linked local note and an unreachable parent
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.offline", "c", ""))
	local := acceptedPulledCopy(t, env, parentNote)
	env.plantSidecar(local, ps5StaleLastUsed)
	env.parent.setDown(true)

	// When the user activates the note
	_, stderr := env.run("activate", "--note", local)

	// Then the command succeeds
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// and the local LastUsed is bumped
	g.Expect(env.lastUsed(local)).To(Equal(env.now().Format(time.DateOnly)))

	// and the failure is recorded for backoff
	cache := ps3Cache(t, env)
	g.Expect(cache.Failures).To(Equal(1))
	g.Expect(cache.BackoffUntil).To(BeTemporally(">", env.now()))
	g.Expect(stderr).To(ContainSubstring("parent unreachable (retry after"))
	g.Expect(env.noteFiles()).To(Equal([]string{local}))
}

// Test792a21zf1e_Activate_ALinkUnderAnotherParentIsNeverChecked verifies 792a21zf1e (A link under another parent
// is never checked): the cached vault ID of the parent differs from the vault of the note's link, so activating
// the note sends the parent no request at all.
func Test792a21zf1e_Activate_ALinkUnderAnotherParentIsNeverChecked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a linked local note whose link is under another parent vault whose ID is cached
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.elsewhere", "c", ""))
	env.stampVault("")
	env.writeState("parent.json", `{"url":"`+parentURL+`","vault_id":"`+parentVaultID+`","failures":0}`)
	env.plant(ps5LocalNote, []byte(localNoteWithLinks(seqID(ps5OtherParentSeed),
		"    - note: "+parentNote+"\n      via: pulled\n      hash: xh1:old\n")))

	// When the user activates the note
	env.run("activate", "--note", ps5LocalNote)

	// Then that link is not contacted
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.parent.requests()).To(BeEmpty())
	g.Expect(env.noteFiles()).To(Equal([]string{ps5LocalNote}))
}

// Test792a21zg1a_Activate_ParentRecencyIsBumped verifies 792a21zg1a (Parent recency is bumped): a child pulls the
// parent note P from a real served parent, and the parent's sidecar of P now carries the day of the use.
func Test792a21zg1a_Activate_ParentRecencyIsBumped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given parent note P activated from a child and the parent reachable
	parentVault := newServeVault(t)
	livePath := writeVaultNote(t, parentVault, ps5ParentNote+".md", liveFactNote("9", ""))
	sidecarErr := os.WriteFile(embed.SidecarPath(livePath), embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@4", Dims: 4,
		SituationVector: []float32{1, 0, 0, 0}, BodyVector: []float32{1, 0, 0, 0},
		ContentHash: "sha256:x", LastUsed: ps5StaleLastUsed,
	}), 0o600)
	g.Expect(sidecarErr).NotTo(HaveOccurred())

	parentDeps := serveTestDeps()
	parentDeps.Now = func() time.Time { return ps5ParentDay }
	routes := cli.ServeRoutes(parentDeps, parentVault, "team", "")

	env := ps2Env(t)
	env.wrap = func(deps *cli.Deps) {
		ps2Identity(ps2User)(deps)
		deps.Fetch = servedFetch(routes)
	}

	// When the pull-down completes
	env.run("activate", "--note", ps5ParentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.noteFiles()).To(HaveLen(1), "the pull-down wrote P's local copy")

	// Then P's sidecar LastUsed on the parent is updated
	sidecar, err := embed.UnmarshalSidecar(readFileBytes(t, embed.SidecarPath(livePath)))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(sidecar.LastUsed).To(Equal(ps5ParentDay.Format(time.DateOnly)))
}

// Test792a21zg1g_Activate_AFailedLocalWriteSendsNoSignal verifies 792a21zg1g (A failed local write sends no
// signal): the write of the pulled copy is refused, so the command fails and the parent, which was asked for the
// note, receives no activate request for it.
func Test792a21zg1g_Activate_AFailedLocalWriteSendsNoSignal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a parent-only note whose local write fails
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.unwritable", "c", ""))
	env.wrap = func(deps *cli.Deps) {
		ps2Identity(ps2User)(deps)
		deps.FS = ps5NoteCreateFS{EdgeFS: deps.FS}
	}

	// When the user activates it
	_, stderr := env.run("activate", "--note", parentNote+".md")

	// Then the command fails
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(stderr).To(ContainSubstring(errPs5NoteCreate.Error()))
	g.Expect(env.noteFiles()).To(BeEmpty())

	// and the parent receives no activate request for the note
	g.Expect(env.parent.shown()).To(Equal([]string{parentNote + ".md"}), "the parent was asked for the note")
	g.Expect(env.parent.activated()).To(BeEmpty())
}

// Test792a21zi1a_Update_NoBackfill verifies 792a21zi1a (No backfill): in a vault whose notes never took part in
// exchange, engram update stamps no xid on any note and changes no note file.
func Test792a21zi1a_Update_NoBackfill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault whose notes have never taken part in exchange
	machine := iu3NewMachine()
	first := foldNote(foldExchange{}, false)
	notes := map[string]string{
		iu3Vault + "/1.2026-09-20.first.md":  first,
		iu3Vault + "/3.2026-09-21.second.md": strings.Replace(first, "luhmann: \"2\"", "luhmann: \"3\"", 1),
	}

	for path, content := range notes {
		machine.fileSystem.seedFile(path, content)
		machine.fileSystem.seedFile(strings.TrimSuffix(path, ".md")+".vec.json", string(validFoldSidecar()))
	}

	before := machine.fileSystem.snapshot(iu3Vault)

	// When the user runs engram update
	outcome := machine.update(t, cli.UpdateArgs{})

	// Then no note gains an xid and no note file changes
	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(machine.fileSystem.snapshot(iu3Vault)).To(Equal(before))

	for path := range notes {
		g.Expect(machine.fileSystem.text(path)).NotTo(ContainSubstring("xid:"))
	}
}

// Test792a21zi1b_Update_TheXIDSurvivesARename verifies 792a21zi1b (The xid survives a rename): reparenting
// renames a note carrying an xid to a continuation of another note, and the renamed note carries the same xid.
func Test792a21zi1b_Update_TheXIDSurvivesARename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note carrying xid
	note := ps5ReparentNote(foldXIDE)

	// When Luhmann reparenting renames it
	renamed := ps5Reparent(t, note)

	// Then the renamed note carries the same xid
	g.Expect(decodeFoldExchange(t, renamed).XID).To(Equal(foldXIDE))
	g.Expect(yamlKeyBlock(string(renamed), "xid")).To(Equal(yamlKeyBlock(note, "xid")))
}

// Test792a21zj1a_Learn_AnOfferedNoteIsLinked verifies 792a21zj1a (Offered note is linked): the parent answers
// the offer of a learned note with a receipt naming its basename B from its vault, and the note then carries that
// vault as parent.vault and B as its primary link, via offered, with the hash the receipt gives.
func Test792a21zj1a_Learn_AnOfferedNoteIsLinked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a learned note whose offer receipt names parent basename B from vault 9a1e
	env := ps2Env(t)
	env.parent.vaultID = ps5ParentVault
	env.parent.storedHash = ps5OfferedHash

	// When the receipt is recorded
	path := env.learnFact("offered")
	g.Expect(env.parent.offers()).To(HaveLen(1))

	// Then the note carries parent.vault 9a1e and a primary link {note: B, via: offered, hash: <offered hash>}
	copied := decodePulledCopy(t, []byte(readFileString(t, path)))
	g.Expect(copied.Parent.Vault).To(Equal(ps5ParentVault))
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{
		{Note: "1100.2026-09-28.offered", Via: "offered", Hash: ps5OfferedHash},
	}))
	g.Expect(copied.Parent.Author).To(Equal(pulledAuthor{}), "an offered note records no author")
}

// Test792a21zj1b_Activate_AChangedParentAddressKeepsLinksValid verifies 792a21zj1b (A changed parent address
// keeps links valid): the accepted copy of a parent note is linked; ENGRAM_PARENT then names a new host for the
// same parent vault, and the link still counts: the merged query still shows the copy in place of the parent
// note, and activating the copy still pulls down the change on the parent.
func Test792a21zj1b_Activate_AChangedParentAddressKeepsLinksValid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given linked notes
	env, parentNote, local := ps5LinkedCopyOfChangedNote(t)

	// When ENGRAM_PARENT changes to a new hostname for the same parent vault
	env.parentURL = ps5MovedParentURL

	// Then existing links still count, because the reported vault ID is unchanged
	stdout, _ := env.run("query", "--phrase", "when pulling", "--chunks-dir", t.TempDir())
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	items := ps5QueryItems(t, stdout)
	g.Expect(slices.Collect(maps.Keys(items))).To(ConsistOf(local), "the copy stands in for the parent note")
	g.Expect(items[local]).To(BeFalse())

	env.run("activate", "--note", local)
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.linkedCopies(parentNote)).To(HaveLen(2), "the change on the parent was pulled down")

	requests := env.parent.requests()
	g.Expect(requests).NotTo(BeEmpty())
	g.Expect(requests[len(requests)-1].url).To(HavePrefix(ps5MovedParentURL), "the new address was contacted")
}

// Test792a21zj1c_Activate_ADifferentParentVaultIgnoresOldLinks verifies 792a21zj1c (A different parent vault
// ignores old links): ENGRAM_PARENT now points at a parent that reports another vault ID, so the old links are
// treated as unlinked: the merged query shows the parent's note as its own item beside the copy, and activating
// the copy pulls nothing down.
func Test792a21zj1c_Activate_ADifferentParentVaultIgnoresOldLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given linked notes
	env, parentNote, local := ps5LinkedCopyOfChangedNote(t)

	// When ENGRAM_PARENT points at a parent that reports a different vault ID
	env.parentURL = ps5MovedParentURL
	env.parent.vaultID = seqID(ps5OtherParentSeed)

	// Then notes linked to the old vault ID are treated as unlinked
	stdout, _ := env.run("query", "--phrase", "when pulling", "--chunks-dir", t.TempDir())
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)

	items := ps5QueryItems(t, stdout)
	g.Expect(slices.Collect(maps.Keys(items))).To(ConsistOf(local, parentNote+".md"))
	g.Expect(items[local]).To(BeFalse())
	g.Expect(items[parentNote+".md"]).To(BeTrue(), "the parent's note is its own item")

	before := env.noteFiles()

	env.run("activate", "--note", local)
	g.Expect(env.exitCodes()).To(BeEmpty(), env.lastStderr)
	g.Expect(env.noteFiles()).To(Equal(before), "no change is pulled down through an old link")
}

// Test792a21zk1a_Update_ARenameRecordsTheOldName verifies 792a21zk1a (A rename records the old name): reparenting
// renames note 1100.2026-09-27.x to 12a.2026-09-27.x, and the renamed note's aliases includes its old basename.
func Test792a21zk1a_Update_ARenameRecordsTheOldName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given note 1100.2026-09-27.x
	note := ps5ReparentNote("")

	// When reparenting renames it to 12a.2026-09-27.x
	renamed := ps5Reparent(t, note)

	// Then the renamed note's aliases includes 1100.2026-09-27.x
	g.Expect(decodeFoldExchange(t, renamed).Aliases).To(ContainElement(ps5ReparentOld))
	g.Expect(string(renamed)).To(ContainSubstring(`luhmann: "12a"`), "the note was rewritten for its new name")
}

// Test792a21zk1b_Amend_AFoldUnionsAliases verifies 792a21zk1b (A fold unions aliases): the fold of offer O, whose
// aliases lists A, into E deletes O and its sidecar and leaves E's aliases holding both O's basename and A.
func Test792a21zk1b_Amend_AFoldUnionsAliases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given offer O whose aliases lists A
	const aliasA = "5.2026-01-01.prev-name"

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, aliases: []string{aliasA},
		links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	offerSidecar := strings.TrimSuffix(foldOfferName, ".md") + ".vec.json"
	vault.put(offerSidecar, validFoldSidecar())

	// When the user runs engram amend --target O --discard --into E
	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	// Then O and its sidecar are gone and E's aliases includes both O's basename and A
	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.files).NotTo(HaveKey(offerSidecar))
	g.Expect(decodeFoldExchange(t, vault.files[foldExistingName]).Aliases).To(ConsistOf(foldOfferBase, aliasA))
}

// Test792a21zk1c_Amend_AFoldKeepsEsPrimaryLink verifies 792a21zk1c (A fold keeps E's primary link): E has a
// primary link to P1 and O a primary link, via pulled, to P2; after the fold E still has P1 as its primary link
// and holds P2 as a covered link, with the hash O had for it.
func Test792a21zk1c_Amend_AFoldKeepsEsPrimaryLink(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given E has a primary link to P1 and O has a primary link via pulled to P2
	const (
		firstParent  = "9.2026-09-01.p1"
		secondParent = "7.2026-09-01.p2"
	)

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{
		xid: foldXIDE, vault: foldParentVault, links: []foldLink{{firstParent, "offered", "xh1:p1"}},
	}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, links: []foldLink{{secondParent, "pulled", "xh1:p2"}},
	}, true))

	// When O is folded into E
	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	// Then E keeps P1 as its primary link and gains {note: P2, via: covered}
	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: firstParent, Via: "offered", Hash: "xh1:p1"},
		{Note: secondParent, Via: "covered", Hash: "xh1:p2"},
	}))
}

// Test792a21zl1a_Amend_PreservesTheExchangeFields verifies 792a21zl1a (Amend preserves the exchange fields): a
// note carrying xid, parent, aliases and offer, rewritten by amend --clear-pending or by a content amend, keeps
// all four byte for byte.
func Test792a21zl1a_Amend_PreservesTheExchangeFields(t *testing.T) {
	t.Parallel()

	for _, noteType := range exchangeSurvivalTypes {
		for name, args := range map[string]cli.AmendArgs{
			"clear-pending": {},
			"content":       exchangeSurvivalContentAmend(noteType),
		} {
			t.Run(noteType+"/"+name, func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)

				if name == "clear-pending" {
					clearPending := false
					args.Pending = &clearPending
				}

				// Given a note carrying xid, parent, aliases and offer
				before := exchangeSurvivalNote(noteType, exchangeSurvivalFixture())

				var (
					written []byte
					writes  int
				)

				// When engram amend --target L --clear-pending, or any content amend, rewrites it
				args.Vault, args.Target, args.ExpectHash = "/vault", "1aa", mustExchangeHash(t, before)

				var out bytes.Buffer

				err := cli.ExportRunAmend(t.Context(), args, runbookAmendDeps([]byte(before), &written, &writes), &out)

				// Then all four are unchanged in the written note
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(writes).To(BeNumerically(">=", 1), "the note was rewritten")
				expectExchangeFieldsUnchanged(g, before, string(written))
			})
		}
	}
}

// Test792a21zl1b_Update_IdentityBackfillPreservesTheLink verifies 792a21zl1b (Identity backfill preserves the
// link): the identity backfill stamps user and vault on a note carrying parent, and parent is unchanged.
func Test792a21zl1b_Update_IdentityBackfillPreservesTheLink(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a note carrying parent
			exchange := exchangeFixture{Parent: exchangeSurvivalFixture().Parent}
			before := strings.Replace(exchangeSurvivalNote(noteType, exchange),
				"user: agent@example.com\nvault: personal\n", "", 1)
			vault := newSkillSurvivalFS(map[string]string{exchangeSurvivalNoteName: before})

			// When engram update --backfill-identity rewrites it
			stamped, err := cli.ExportBackfillIdentity(t.Context(), "/vault", exchangeSurvivalIdentityDeps(vault), false)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stamped).To(Equal(1), "the backfill rewrote the note")

			// Then parent is unchanged
			after := vault.content(exchangeSurvivalNoteName)
			g.Expect(after).To(ContainSubstring("user: agent@example.com"))
			g.Expect(yamlKeyBlock(before, "parent")).NotTo(BeEmpty())
			g.Expect(yamlKeyBlock(after, "parent")).To(Equal(yamlKeyBlock(before, "parent")))
		})
	}
}

// Test792a21zl1c_Amend_AFoldRefusesAnAnchoredParent verifies 792a21zl1c (Fold refuses an anchored parent): E's
// parent: carries an anchor that another key aliases and the fold would change its links; the command fails
// naming parent, E is byte-identical and the offer is not deleted.
func Test792a21zl1c_Amend_AFoldRefusesAnAnchoredParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note E whose parent: carries an anchor that another key aliases, and a fold that would change E's
	// parent links
	vault := newFoldVault()
	existing := anchorBlock(foldNote(foldExchange{
		xid: foldXIDE, vault: foldParentVault, links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
	}, false), "parent")
	vault.put(foldExistingName, existing)
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	before := maps.Clone(vault.files)

	// When the user runs engram amend --discard --into E
	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})

	// Then the command fails naming parent, E is byte-identical, and the offer is not deleted
	g.Expect(err).To(MatchError(cli.ErrFrontmatterAnchoredKeyForTest))
	g.Expect(err).To(MatchError(ContainSubstring("parent")))
	g.Expect(string(vault.files[foldExistingName])).To(Equal(existing))
	g.Expect(vault.files).To(HaveKey(foldOfferName))
	g.Expect(vault.files).To(Equal(before), "the refusal left every file untouched")
	g.Expect(vault.declines).To(BeEmpty())
}

// Test792a21zl1d_Amend_AFoldKeepsUnknownKeysUnderParent verifies 792a21zl1d (Fold keeps unknown keys under
// parent): E's parent: and one of its links carry keys the exchange model does not define; after the fold the
// written E still carries both keys with their values, beside the folded link.
func Test792a21zl1d_Amend_AFoldKeepsUnknownKeysUnderParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given E's parent: and one of its links carry keys the exchange model does not define
	vault := newFoldVault()
	vault.put(foldExistingName, withUnknownParentKeys(foldNote(foldExchange{
		xid: foldXIDE, vault: foldParentVault, links: []foldLink{{"9.2026-09-01.other", "offered", "xh1:other"}},
	}, false)))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	// When the fold rewrites E's parent:
	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	// Then the written E still carries both keys with their values, alongside the folded links
	expectUnknownParentKeysKept(g, string(vault.files[foldExistingName]))
	g.Expect(decodeFoldExchange(t, vault.files[foldExistingName]).Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "9.2026-09-01.other", Via: "offered", Hash: "xh1:other"},
		{Note: "7.2026-09-01.parent", Via: "covered", Hash: "xh1:parent"},
	}))
}

// Test792a21zl1e_Query_AReceiptRefusesAnAnchoredParent verifies 792a21zl1e (Receipt refuses an anchored parent):
// the parent accepts the offer of a note whose parent: carries an anchor that another key aliases; the receipt is
// refused with a warning naming parent, and the note is not written.
func Test792a21zl1e_Query_AReceiptRefusesAnAnchoredParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note whose parent: carries an anchor that another key aliases
	env := ps2Env(t)
	note := ps4Anchored(t, env)
	before := snapshotVault(t, env.vault)

	// When an offer receipt would rewrite it
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then the receipt is refused naming parent and the note is not written
	g.Expect(env.parent.offers()).To(HaveLen(1), "the parent accepted the offer")
	g.Expect(stderr).To(ContainSubstring("YAML anchor: parent"))
	g.Expect(readFileString(t, note.path)).To(Equal(note.anchored))

	after := snapshotVault(t, env.vault)
	for path, content := range before {
		if strings.HasSuffix(path, ".md") {
			g.Expect(after[path]).To(Equal(content), path)
		}
	}
}

// Test792a21zl1f_Query_AReceiptKeepsUnknownKeysUnderParent verifies 792a21zl1f (Receipt keeps unknown keys under
// parent): a queued note whose parent: and kept covered link carry keys the exchange model does not define gets
// its offer receipt, and the written note still carries both keys with their values beside the new primary link.
func Test792a21zl1f_Query_AReceiptKeepsUnknownKeysUnderParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note whose parent: and one of its kept links carry keys the exchange model does not define
	env := ps2Env(t)
	path := ps3Backlog(t, env, ps5ReceiptSlug)[0]
	plain := readFileString(t, path)
	withKeys := strings.Replace(plain, "\n---\n", "\nparent:\n    extra_parent_field: keep\n    vault: "+parentVaultID+
		"\n    links:\n        - note: 9.2026-09-01.other\n          via: covered\n          hash: xh1:other\n"+
		"          extra_link_field: keep\n---\n", 1)
	g.Expect(withKeys).NotTo(Equal(plain), "the learned note has frontmatter to extend")
	env.plant(filepath.Base(path), []byte(withKeys))

	// When an offer receipt rewrites it
	env.run(ps3QueryArgs(env)...)
	g.Expect(env.parent.offers()).To(HaveLen(1))

	// Then the written note still carries both keys with their values
	written := readFileString(t, path)
	g.Expect(written).NotTo(Equal(withKeys), "the receipt was recorded")
	expectUnknownParentKeysKept(g, written)

	linked := decodePulledCopy(t, []byte(written)).Parent.Links
	g.Expect(linked).To(ConsistOf(
		pulledLink{Note: "1100.2026-09-28." + ps5ReceiptSlug, Via: "offered", Hash: "xh1:stored"},
		pulledLink{Note: "9.2026-09-01.other", Via: "covered", Hash: "xh1:other"},
	))
}

// Test792a21zl1g_Rewrites_EveryOtherPathPreservesTheFields verifies 792a21zl1g (Every other rewrite path
// preserves the fields): a note carrying xid, parent, aliases and offer comes out of resituate, reparenting, the
// vocabulary rewrites, the reference scrub and the skill registration refresh and adopt with all four unchanged
// (a rename appends the old basename to aliases, the one change the requirement licenses).
func Test792a21zl1g_Rewrites_EveryOtherPathPreservesTheFields(t *testing.T) {
	t.Parallel()

	for name, rewrite := range map[string]func(ctx context.Context, g *WithT, before string){
		"resituate": func(ctx context.Context, g *WithT, before string) {
			after, writes, err := runExchangeSurvivalResituate(ctx, before)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(writes).To(Equal(1))
			g.Expect(after).To(ContainSubstring("situation: resituated context"))
			expectExchangeFieldsUnchanged(g, before, after)
		},
		"reparenting renames the note": func(_ context.Context, g *WithT, before string) {
			fixture := newReparentFixture(map[string]string{exchangeSurvivalNoteName: before})
			oldBasename := strings.TrimSuffix(exchangeSurvivalNoteName, ".md")

			_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
				map[string]string{oldBasename: "3.2026-01-01.exchanged"})
			g.Expect(err).NotTo(HaveOccurred())

			after := string(fixture.written["/vault/3.2026-01-01.exchanged.md"])
			g.Expect(after).To(ContainSubstring(`luhmann: "3"`))
			expectExchangeFieldsSurviveRename(g, before, after, oldBasename)
		},
		"reparenting rewrites its wikilinks": func(_ context.Context, g *WithT, before string) {
			fixture := newReparentFixture(map[string]string{
				exchangeSurvivalNoteName:           before,
				exchangeSurvivalLinkTarget + ".md": "---\ntype: fact\nluhmann: \"2\"\n---\n\nTarget.\n",
			})

			_, err := cli.RenameAndRewriteReferences(fixture.deps(), "/vault",
				map[string]string{exchangeSurvivalLinkTarget: "4.2026-01-01.old-target"})
			g.Expect(err).NotTo(HaveOccurred())

			after := string(fixture.written["/vault/"+exchangeSurvivalNoteName])
			g.Expect(after).To(ContainSubstring("[[4.2026-01-01.old-target]]"))
			expectExchangeFieldsUnchanged(g, before, after)
		},
		"vocabulary assignment": func(_ context.Context, g *WithT, before string) {
			after := cli.ExportWriteVocabAssignment(before, []string{"new-term", "other-term"})
			g.Expect(after).To(ContainSubstring("vocab/new-term"))
			expectExchangeFieldsUnchanged(g, before, after)
		},
		"vocabulary legacy cleanup": func(_ context.Context, g *WithT, before string) {
			legacy := strings.Replace(before, "tier: L2\n", "tier: L2\nvocab: old-term\n", 1)
			after, changed := cli.ExportStripLegacyVocabChannel(legacy)
			g.Expect(changed).To(BeTrue())
			expectExchangeFieldsUnchanged(g, legacy, after)
		},
		"reference scrub": func(_ context.Context, g *WithT, before string) {
			after, changed := cli.ExportRemoveNoteReferences(before,
				map[string]bool{exchangeSurvivalSupersededBase + ".md": true})
			g.Expect(changed).To(BeTrue())
			expectExchangeFieldsUnchanged(g, before, after)
		},
		"skill registration refresh": func(ctx context.Context, g *WithT, _ string) {
			vault := newSkillAcceptFixtureVault()
			basename := "1049.2026-09-21.skill-claude-curate"
			before := exchangeSurvivalSkillNote(exchangeSurvivalFixture())
			vault.put(basename+".md", before)

			var stdout bytes.Buffer

			skill := installedEngramSkill("curate", []byte("# Curate\n\nRevised.\n"))
			g.Expect(cli.RefreshSkill(ctx, "/vault", skill, basename, skillAcceptDeps(vault), &stdout)).To(Succeed())

			after, _ := vault.get(basename + ".md")
			g.Expect(after).To(ContainSubstring("Revised."))
			expectExchangeFieldsUnchanged(g, before, after)
		},
		"skill registration adopt": func(ctx context.Context, g *WithT, _ string) {
			vault := newSkillAcceptFixtureVault()
			before := exchangeSurvivalSkillNote(exchangeSurvivalFixture())
			vault.put("1049.2026-09-21.skill-claude-curate.md", before)

			deps := cli.SkillAdoptDeps{
				Lock:     noLock,
				Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
				Rename:   skillAcceptRenameDeps(vault),
				Embedder: skillAcceptFakeEmbedder{},
			}

			var stdout bytes.Buffer

			skill := installedEngramSkill("curate", []byte("# Curate\n\n1. Judge offers.\n"))
			g.Expect(cli.AdoptSkillNote(ctx, "/vault", skill, "1049", deps, &stdout)).To(Succeed())

			after, _ := vault.get("1049.2026-09-21.skill-claude-curate.md")
			g.Expect(after).To(ContainSubstring("Judge offers."))
			expectExchangeFieldsUnchanged(g, before, after)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rewrite(t.Context(), NewWithT(t), exchangeSurvivalNote("fact", exchangeSurvivalFixture()))
		})
	}
}

// unexported constants.
const (
	// ps5LocalNote is a local note planted with a link under another parent vault.
	ps5LocalNote = "3.2026-09-20.local.md"
	// ps5MovedParentURL is the new address of the parent in the address scenarios.
	ps5MovedParentURL = "http://parent-moved:9100"
	// ps5OfferedHash is the hash a receipt gives the primary link of an offered note.
	ps5OfferedHash = "xh1:5b6f1d2c3a49587e6d0f1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f"
	// ps5OtherParentSeed seeds the vault ID of a parent that is not the env's.
	ps5OtherParentSeed = 120
	// ps5ParentNote is the live note of the served parent.
	ps5ParentNote = "9.2026-01-09.live"
	// ps5ParentVault is the vault ID the parent reports in the offered-link scenario.
	ps5ParentVault = "9a1e2b3c4d5e6f708192a3b4c5d6e7f8"
	// ps5ReceiptSlug names the note learned for the receipt scenario.
	ps5ReceiptSlug = "kept"
	// ps5ReparentOld is the basename of the note the reparenting scenarios rename.
	ps5ReparentOld = "1100.2026-09-27.x"
	// ps5StaleLastUsed is the day before every clock of these tests.
	ps5StaleLastUsed = "2026-02-02"
	// ps5TargetNote is the note 12, the target of the reparenting scenarios.
	ps5TargetNote = "---\ntype: fact\nluhmann: \"12\"\ncreated: 2026-09-20\n---\n\ntarget body.\n"
)

// unexported variables.
var (
	errPs5NoteCreate = errors.New("ps5: the creation of the note is refused")
	// ps5ParentDay is the day of the served parent's clock, distinct from the child's.
	ps5ParentDay = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
)

// ps5NoteCreateFS refuses the exclusive creation of any note file, as the pull-down writes a new copy.
type ps5NoteCreateFS struct {
	cli.EdgeFS
}

func (f ps5NoteCreateFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if strings.HasSuffix(path, ".md") {
		return errPs5NoteCreate
	}

	return f.EdgeFS.WriteFileExcl(path, data, perm)
}

// ps5LinkedCopyOfChangedNote pulls a parent note down, accepts the copy, and then changes the parent note, so
// that only the link (never an equal hash) can tie the copy to the parent note. It returns the env, the parent
// note's basename and the accepted copy's file name.
func ps5LinkedCopyOfChangedNote(t *testing.T) (*wiringEnv, string, string) {
	t.Helper()

	env := ps2Env(t)
	env.parent.queryNotes = true
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.moved", "c", ""))
	local := acceptedPulledCopy(t, env, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))

	return env, parentNote, local
}

// ps5QueryItems maps the path of each item of a query payload to whether it came from the parent.
func ps5QueryItems(t *testing.T, stdout string) map[string]bool {
	t.Helper()

	var payload struct {
		Items []struct {
			Path       string `yaml:"path"`
			FromParent bool   `yaml:"from_parent"`
		} `yaml:"items"`
	}

	err := yaml.Unmarshal([]byte(stdout), &payload)
	if err != nil {
		t.Fatalf("the query payload does not decode: %v\n%s", err, stdout)
	}

	items := make(map[string]bool, len(payload.Items))
	for _, item := range payload.Items {
		items[item.Path] = item.FromParent
	}

	return items
}

// ps5Reparent runs the apply phase of engram update --reparent-luhmann on an in-memory vault of the target note
// 12 and the given note 1100.2026-09-27.x, which the answers move to the continuation 12a of note 12, and
// returns the content written for the renamed note.
func ps5Reparent(t *testing.T, note string) []byte {
	t.Helper()

	const targetName = "12.2026-09-20.target.md"

	files := map[string][]byte{
		"/vault/" + targetName:             []byte(ps5TargetNote),
		"/vault/" + ps5ReparentOld + ".md": []byte(note),
	}
	files["/vault/12.2026-09-20.target.vec.json"] = reparentSidecar([]float32{1, 0, 0})
	files["/vault/"+ps5ReparentOld+".vec.json"] = reparentSidecar([]float32{0.99, 0.01, 0})

	deps, renames := newReparentDeps(files, []string{targetName, ps5ReparentOld + ".md"})
	fingerprint := reparentDeriveFingerprint(t, deps)
	files["/answers.json"] = []byte(`{"reparenting":[{"note":"1100","position":"continuation","target":"12"}],` +
		`"fingerprint":"` + fingerprint + `"}`)

	var stdout bytes.Buffer

	err := cli.RunReparentLuhmann(t.Context(), "/vault", "/chunks", "/answers.json", false, deps, &stdout)
	if err != nil {
		t.Fatalf("reparenting: %v", err)
	}

	renamedPath := renames["/vault/"+ps5ReparentOld+".md"]
	if renamedPath != "/vault/12a.2026-09-27.x.md" {
		t.Fatalf("the note was renamed to %q, not to 12a.2026-09-27.x", renamedPath)
	}

	return files[renamedPath]
}

// ps5ReparentNote is the note 1100.2026-09-27.x: a fact, carrying xid when xid is not empty.
func ps5ReparentNote(xid string) string {
	frontmatter := "---\ntype: fact\nluhmann: \"1100\"\ncreated: 2026-09-27\n"
	if xid != "" {
		frontmatter += "xid: " + xid + "\n"
	}

	return frontmatter + "---\n\nthe note body text.\n"
}
