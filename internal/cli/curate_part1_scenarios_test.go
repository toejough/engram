package cli_test

// Curate, part 1 (792a22a Pending offers are left out of normal query results; 792a22c Activating a note commits
// directly; 792a22d Curation judges each offer by its content against the host vault; 792a22f to 792a22i the
// covered, near, absent and rejected rows; 792a22l A bookkeeping step names the version that was judged; 792a22m
// Folding an offer keeps its identity and reads a CRLF note as LF). Curation itself is the agent's: the binary
// owns the bookkeeping steps it calls, so each scenario drives the step the row names (RunQuery, the activate
// command and route, the learn route, RunAmend and the amend command) and checks what it leaves in the vault.
// The commands that need a whole vault (activate, a served route, a pull-down) run over an in-memory file system
// (cv1MemFS, controlled_vocabulary_part1_scenarios_test.go) and the recording parent of the parent-sync tests.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a22a1a_RunQuery_PendingOfferDoesNotSurfaceInRecall: a vault holds a live fact and a pending offer with
// the same situation as the phrase asked for; the query returns the live fact and not the offer, and tells the
// agent that offers exist.
func Test792a22a1a_RunQuery_PendingOfferDoesNotSurfaceInRecall(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const situation = "releasing a module"

	memFS := newInMemoryFS()
	plantNoteWithSidecar(t, memFS, "/vault", curate1LiveName, curate1FactNote(situation, ""))
	plantNoteWithSidecar(t, memFS, "/vault", curate1OfferName, curate1FactNote(situation, "pending: true\n"))

	var out bytes.Buffer

	err := cli.RunQuery(t.Context(),
		cli.QueryArgs{Phrases: []string{situation}, VaultPath: "/vault"}, newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var payload queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &payload)).To(Succeed())

	returned := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		returned = append(returned, filepath.Base(item.Path))
	}

	g.Expect(returned).To(ConsistOf(curate1LiveName), "the live fact surfaces and the pending offer does not")
	g.Expect(out.String()).To(ContainSubstring("pending_offers: true"))
	g.Expect(out.String()).NotTo(ContainSubstring(strings.TrimSuffix(curate1OfferName, ".md")))
}

// Test792a22c1a_ServeActivate_ServedActivateCommitsImmediately: an activate request handled over the served API
// bumps the named note's sidecar LastUsed and nothing else: the note is byte-identical, carries no pending
// marker, and no file other than that sidecar is written (no offer, no curation step).
func Test792a22c1a_ServeActivate_ServedActivateCommitsImmediately(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, false)
	machine.put(curate1LiveName, curate1FactNote("serving a note", ""))
	machine.putSidecar(curate1LiveName)

	notesBefore := machine.noteNames()
	machine.fsys.resetLog()

	body, marshalErr := json.Marshal(map[string][]string{"notes": {curate1LiveName}})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	routes := cli.ServeRoutes(machine.deps(io.Discard, io.Discard), curate1VaultPath, "personal", "")
	resp := routeFor(t, routes, "/activate").Serve(t.Context(), cli.ServeRequest{Body: body})

	g.Expect(resp.Status).To(Equal(200))
	g.Expect(machine.lastUsed(curate1LiveName)).To(Equal(curate1Today), "the sidecar LastUsed is updated")
	g.Expect(machine.read(curate1LiveName)).To(Equal(curate1FactNote("serving a note", "")))
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(machine.read(curate1LiveName)))).To(BeFalse())
	g.Expect(machine.fsys.writesTo(machine.path(curate1SidecarName(curate1LiveName)))).To(Equal(1))
	g.Expect(machine.fsys.writeCount()).To(Equal(1), "no other file is written")
	g.Expect(machine.fsys.removeCount()).To(BeZero())
	g.Expect(machine.noteNames()).To(Equal(notesBefore), "no note is created")
}

// Test792a22c1b_Activate_LocalActivateCommitsImmediately: `engram activate` on a note in the local vault bumps
// the sidecar LastUsed; the note is byte-identical and carries no pending marker, and only the sidecar is written.
func Test792a22c1b_Activate_LocalActivateCommitsImmediately(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, false)
	machine.put(curate1LiveName, curate1FactNote("activating a note", ""))
	machine.putSidecar(curate1LiveName)
	machine.fsys.resetLog()

	stdout, stderr := machine.run("activate", "--note", curate1LiveName)

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(BeEmpty())
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(machine.lastUsed(curate1LiveName)).To(Equal(curate1Today), "the sidecar LastUsed is updated")
	g.Expect(machine.read(curate1LiveName)).To(Equal(curate1FactNote("activating a note", "")))
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(machine.read(curate1LiveName)))).To(BeFalse())
	g.Expect(machine.fsys.writesTo(machine.path(curate1SidecarName(curate1LiveName)))).To(Equal(1))
	g.Expect(machine.fsys.writeCount()).To(Equal(1), "no other file is written")
}

// Test792a22d1d_ServeLearn_AnswerComesBeforeAnyCuration: a served learn that creates a pending offer is
// answered with a pending receipt, and when the answer is back the offer is still the pending note the learn
// wrote: nothing judged, accepted, folded or discarded it, and nothing was sent to a parent or spawned.
func Test792a22d1d_ServeLearn_AnswerComesBeforeAnyCuration(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, false)

	var fetches atomic.Int32

	deps := machine.deps(io.Discard, io.Discard)
	deps.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
		fetches.Add(1)

		return cli.FetchResponse{}, nil
	}

	body, marshalErr := json.Marshal(offeredFact("served-offer", "object", offerOf(childOrigin, "key-one", "")))
	g.Expect(marshalErr).NotTo(HaveOccurred())

	machine.fsys.resetLog()

	resp := routeFor(t, cli.ServeRoutes(deps, curate1VaultPath, "personal", ""), "/learn").
		Serve(t.Context(), cli.ServeRequest{Body: body})

	g.Expect(resp.Status).To(Equal(200))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Pending).To(BeTrue(), "the answer says the offer is pending")

	notes := machine.noteNames()
	g.Expect(notes).To(Equal([]string{receipt.Basename + ".md"}), "one note, the offer the learn wrote")

	if len(notes) != 1 {
		return
	}

	offer := machine.read(notes[0])
	g.Expect(cli.ExportNoteHasPendingMarker([]byte(offer))).To(BeTrue(), "no curation has judged or accepted it")
	g.Expect(machine.fsys.removeCount()).To(BeZero(), "nothing was discarded or folded away")
	g.Expect(fetches.Load()).To(BeZero(), "nothing was sent anywhere")
	g.Expect(machine.exits).To(BeEmpty())
}

// Test792a22f1a_RunAmend_CoveredOfferIsReinforcedAndDiscardedIntoTheExistingNote: curation's covered row is
// `amend --activate` on the existing note E and then `amend --discard --into E` on the offer O: E's LastUsed is
// bumped, O and its sidecar are gone, E gains O's basename and alias, and no note content is created or changed.
func Test792a22f1a_RunAmend_CoveredOfferIsReinforcedAndDiscardedIntoTheExistingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newFoldVault()
	existing := foldNote(foldExchange{xid: foldXIDE, vault: foldParentVault}, false)
	offer := foldNote(foldExchange{xid: foldXIDO, origin: true, aliases: []string{"5.2026-01-01.prev-name"}}, true)
	offerSidecar := strings.TrimSuffix(foldOfferName, ".md") + ".vec.json"
	existingSidecar := strings.TrimSuffix(foldExistingName, ".md") + ".vec.json"

	vault.put(foldExistingName, existing)
	vault.put(existingSidecar, validFoldSidecar())
	vault.put(foldOfferName, offer)
	vault.put(offerSidecar, validFoldSidecar())

	judged := mustHash(t, []byte(offer))

	reinforceErr := vault.amend(t.Context(), cli.AmendArgs{Target: foldExistingBase, Activate: true})
	g.Expect(reinforceErr).NotTo(HaveOccurred())
	g.Expect(string(vault.files[foldExistingName])).To(Equal(existing), "reinforcing E leaves its note alone")

	foldErr := vault.amend(t.Context(),
		cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase, ExpectHash: judged})
	g.Expect(foldErr).NotTo(HaveOccurred())

	g.Expect(curate1LastUsed(g, vault.files[existingSidecar])).To(Equal("2026-09-28"), "E is reinforced")
	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.files).NotTo(HaveKey(offerSidecar))

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Aliases).To(Equal([]string{foldOfferBase, "5.2026-01-01.prev-name"}))
	expectOnlyExchangeKeysChanged(g, existing, vault.files[foldExistingName])
	g.Expect(curate1LiveNotes(vault)).To(Equal([]string{foldExistingName}), "no new note results")
	g.Expect(vault.declines).To(BeEmpty(), "a covered offer is folded, not declined")
}

// Test792a22f1b_Amend_PulledOfferCoveredByLocalNoteIsFoldedAndNotPulledAgain: a pulled-down offer judged covered
// by local note L is discarded with --into L; L gains a link to the parent note, a later merged query returns L
// and not the parent's copy, and activating the parent note again pulls nothing.
func Test792a22f1b_Amend_PulledOfferCoveredByLocalNoteIsFoldedAndNotPulledAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, true)
	machine.parent.queryNotes = true
	parentNote := machine.parent.addNote(pulledFact("7.2026-09-01.covered-by-local", "c", ""))
	machine.put(foldExistingName, curate1FactNote("when pulling", ""))
	machine.putSidecar(foldExistingName)

	machine.run("activate", "--note", parentNote+".md")

	offers := machine.linkedCopies(parentNote)
	g.Expect(offers).To(HaveLen(1), "the pull-down wrote the offer")

	if len(offers) != 1 {
		return
	}

	_, stderr := machine.run("amend", "--target", strings.TrimSuffix(offers[0], ".md"), "--discard",
		"--into", foldExistingBase, "--expect-hash", mustHash(t, []byte(machine.read(offers[0]))))
	g.Expect(stderr).To(BeEmpty())
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(machine.noteNames()).To(Equal([]string{foldExistingName}), "the offer is discarded")
	g.Expect(machine.linkedCopies(parentNote)).To(Equal([]string{foldExistingName}), "L links the parent note")

	linked := decodePulledCopy(t, []byte(machine.read(foldExistingName)))
	g.Expect(linked.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(linked.Parent.Links).To(Equal([]pulledLink{
		{Note: parentNote, Via: "pulled", Hash: machine.parent.hashOf(t, parentNote)},
	}))

	stdout, _ := machine.run("query", "--phrase", "when pulling", "--chunks-dir", curate1ChunksPath)
	g.Expect(machine.exits).To(BeEmpty())

	var payload queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &payload)).To(Succeed())

	paths := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		paths = append(paths, item.Path)
	}

	g.Expect(paths).To(ContainElement(foldExistingName), "the merged query returns L")
	g.Expect(paths).NotTo(ContainElement(parentNote+".md"), "and dedupes the parent's copy")

	machine.run("activate", "--note", parentNote+".md")
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(machine.noteNames()).To(Equal([]string{foldExistingName}), "the unchanged parent note is not pulled again")
}

// Test792a22g1a_RunAmend_NearOfferIsFoldedThenItsClaimIsAmendedIn: curation's near row is `amend --discard --into E`
// on the offer and then a content amend of E: the offer's removal comes before the write that carries the added
// claim, E ends with the claim and the offer's basename, and no second live note remains.
func Test792a22g1a_RunAmend_NearOfferIsFoldedThenItsClaimIsAmendedIn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const claim = "n plus the offer's additional claim"

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))

	events := make([]string, 0, curate1EventCapacity)
	deps := vault.deps()
	deps.Embedder = stubEmbedder{modelID: "stub@4", dims: 4}
	remove, write := deps.Remove, deps.Write
	deps.Remove = func(path string) error {
		events = append(events, "remove "+filepath.Base(path))

		return remove(path)
	}
	deps.Write = func(path string, data []byte) error {
		if strings.Contains(string(data), claim) {
			events = append(events, "write the claim to "+filepath.Base(path))
		}

		return write(path, data)
	}

	fold := cli.AmendArgs{Vault: "/vault", Target: foldOfferBase, Discard: true, Into: foldExistingBase}
	g.Expect(cli.RunAmend(t.Context(), fold, deps, io.Discard)).To(Succeed())

	amend := cli.AmendArgs{Vault: "/vault", Target: foldExistingBase, Object: claim, VaultName: "personal"}
	g.Expect(cli.RunAmend(t.Context(), amend, deps, io.Discard)).To(Succeed())

	g.Expect(events).To(Equal([]string{
		"remove " + foldOfferName, "remove " + strings.TrimSuffix(foldOfferName, ".md") + ".vec.json",
		"write the claim to " + foldExistingName,
	}), "the offer is folded away before the claim is written")
	g.Expect(string(vault.files[foldExistingName])).To(ContainSubstring("object: " + claim))
	g.Expect(decodeFoldExchange(t, vault.files[foldExistingName]).Aliases).To(ContainElement(foldOfferBase))
	g.Expect(curate1LiveNotes(vault)).To(Equal([]string{foldExistingName}), "no second live note remains")
}

// Test792a22h1b_RunAmend_AcceptingAnOfferKeepsItsAuthor: `amend --target O --clear-pending` on an offer whose user:
// is alice, run by a curator whose detected user is bob, clears the marker and leaves user: alice; the curator's
// identity is not even consulted. Both kinds of offer: a pulled-down one and a served one (judged hash passed).
func Test792a22h1b_RunAmend_AcceptingAnOfferKeepsItsAuthor(t *testing.T) {
	t.Parallel()

	cases := map[string]foldExchange{
		"a pulled-down offer": {
			xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
		},
		"a served offer": {xid: foldXIDO, origin: true},
	}

	for name, exchange := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			offer := strings.Replace(foldNote(exchange, true), "user: alice@example.com", "user: alice", 1)
			vault := newFoldVault()
			vault.put(foldOfferName, offer)

			args := cli.AmendArgs{Target: foldOfferBase, ClearPending: true, Pending: new(bool)}
			if exchange.origin {
				args.ExpectHash = mustHash(t, []byte(offer))
			}

			err := vault.amend(t.Context(), args)
			g.Expect(err).NotTo(HaveOccurred())

			frontmatter, _, found := embed.SplitFrontmatter(vault.files[foldOfferName])
			g.Expect(found).To(BeTrue())

			var accepted struct {
				User    string `yaml:"user"`
				Repo    string `yaml:"repo"`
				Vault   string `yaml:"vault"`
				Pending bool   `yaml:"pending"`
			}

			g.Expect(yaml.Unmarshal(frontmatter, &accepted)).To(Succeed())
			g.Expect(accepted.Pending).To(BeFalse(), "the marker is cleared")
			g.Expect(accepted.User).To(Equal("alice"), "the author is still alice, not the curator bob")
			g.Expect(accepted.Repo).To(Equal("github.com/alice/widgets"))
			g.Expect(accepted.Vault).To(Equal("personal"))
			g.Expect(vault.detected).To(BeZero(), "accepting never consults the curator's identity")
		})
	}
}

// Test792a22i1a_Amend_RejectedPulledOfferIsRecordedAsDeclined: curation's rejected row, a bare `amend --discard`
// of a pulled-down offer, deletes the offer and records the parent note as declined at the exchange hash it was
// pulled at; the same unchanged parent note is not pulled down again.
func Test792a22i1a_Amend_RejectedPulledOfferIsRecordedAsDeclined(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, true)
	parentNote := machine.parent.addNote(pulledFact("7.2026-09-01.rejected", "c", ""))
	hash := machine.parent.hashOf(t, parentNote)

	machine.run("activate", "--note", parentNote+".md")

	offers := machine.linkedCopies(parentNote)
	g.Expect(offers).To(HaveLen(1), "the pull-down wrote the offer")

	if len(offers) != 1 {
		return
	}

	_, stderr := machine.run("amend", "--target", strings.TrimSuffix(offers[0], ".md"), "--discard")
	g.Expect(stderr).To(BeEmpty())
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(machine.noteNames()).To(BeEmpty(), "the offer is deleted")
	g.Expect(machine.declined()).To(Equal([]declinedEntry{{Vault: parentVaultID, Basename: parentNote, Hash: hash}}))

	machine.run("activate", "--note", parentNote+".md")
	g.Expect(machine.noteNames()).To(BeEmpty(), "the declined, unchanged parent note is not pulled again")
}

// Test792a22l1a_RunAmend_StepOnAServedOfferWithoutTheHashIsRefused: each bookkeeping step on a pending offer that
// carries offer.origin (--clear-pending, --discard, --discard --into) fails without --expect-hash, tells the
// curator to pass the exchange hash that was judged, and changes nothing.
func Test792a22l1a_RunAmend_StepOnAServedOfferWithoutTheHashIsRefused(t *testing.T) {
	t.Parallel()

	steps := map[string]cli.AmendArgs{
		"clear-pending": {ClearPending: true, Pending: new(bool)},
		"discard":       {Discard: true},
		"discard-into":  {Discard: true, Into: foldExistingBase},
	}

	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newFoldVault()
			vault.put(foldExistingName, foldNote(foldExchange{}, false))
			vault.put(foldOfferName, foldNote(foldExchange{xid: foldXIDO, origin: true}, true))
			before := maps.Clone(vault.files)

			step.Target = foldOfferBase

			err := vault.amend(t.Context(), step)
			g.Expect(err).To(MatchError(ContainSubstring("pass --expect-hash with the exchange hash you judged")))
			g.Expect(vault.files).To(Equal(before), "the offer is unchanged")
			g.Expect(vault.declines).To(BeEmpty())
		})
	}
}

// Test792a22l1b_RunAmend_OfferUpdatedAfterJudgmentIsNotAccepted: the curator judged pending offer N at H1 and the
// child then updated N in place to H2; `amend --clear-pending --expect-hash H1` fails as a changed version and N
// stays pending, byte for byte the H2 content.
func Test792a22l1b_RunAmend_OfferUpdatedAfterJudgmentIsNotAccepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	judged := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)
	updated := strings.Replace(judged, "object: n", "object: updated by the child", 1)
	g.Expect(updated).NotTo(Equal(judged))

	vault := newFoldVault()
	vault.put(foldOfferName, updated)
	vault.put(strings.TrimSuffix(foldOfferName, ".md")+".vec.json", validFoldSidecar())
	before := maps.Clone(vault.files)

	h1, h2 := mustHash(t, []byte(judged)), mustHash(t, []byte(updated))
	g.Expect(h1).NotTo(Equal(h2))

	err := vault.amend(t.Context(),
		cli.AmendArgs{Target: foldOfferBase, ClearPending: true, Pending: new(bool), ExpectHash: h1})
	g.Expect(err).To(MatchError(ContainSubstring("changed since it was judged")))
	g.Expect(vault.files).To(Equal(before), "nothing is written")
	g.Expect(string(vault.files[foldOfferName])).To(Equal(updated))
	g.Expect(cli.ExportNoteHasPendingMarker(vault.files[foldOfferName])).To(BeTrue(), "N stays pending")
}

// Test792a22l1c_RunAmend_ChangedOfferIsReadAgainAndTheStepRedoneWithTheNewHash: after a failed --expect-hash the
// curator re-reads the whole offer with engram show, whose first line names the new hash, and redoes the actions
// of the row the new content lands on with that hash: for an absent offer the clear-pending, for a covered one
// the reinforcement and the fold. The redone steps succeed.
func Test792a22l1c_RunAmend_ChangedOfferIsReadAgainAndTheStepRedoneWithTheNewHash(t *testing.T) {
	t.Parallel()

	for name, row := range map[string]string{"absent row": "absent", "covered row": "covered"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			judged := foldNote(foldExchange{xid: foldXIDO, origin: true}, true)
			updated := strings.Replace(judged, "object: n", "object: updated by the child", 1)

			vault := newFoldVault()
			vault.put(foldExistingName, foldNote(foldExchange{}, false))
			vault.put(strings.TrimSuffix(foldExistingName, ".md")+".vec.json", validFoldSidecar())
			vault.put(foldOfferName, updated)

			step := cli.AmendArgs{Target: foldOfferBase, ClearPending: true, Pending: new(bool)}
			if row == "covered" {
				step = cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase}
			}

			step.ExpectHash = mustHash(t, []byte(judged))
			g.Expect(vault.amend(t.Context(), step)).To(MatchError(ContainSubstring("changed since it was judged")))

			// The curator re-reads the whole offer; the hash comes from the first line of engram show.
			var shown bytes.Buffer

			deps := vault.deps()
			showDeps := cli.ShowDeps{Scan: deps.Scan, Read: deps.Read}
			g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: foldOfferBase, VaultPath: "/vault"}, showDeps, &shown)).
				To(Succeed())
			g.Expect(shown.String()).To(ContainSubstring("object: updated by the child"), "the whole new offer is read")

			firstLine, _, _ := strings.Cut(shown.String(), "\n")
			newHash := strings.TrimPrefix(firstLine, "# exchange_hash: ")
			g.Expect(newHash).To(Equal(mustHash(t, []byte(updated))))

			if row == "covered" {
				g.Expect(vault.amend(t.Context(), cli.AmendArgs{Target: foldExistingBase, Activate: true})).To(Succeed())
			}

			step.ExpectHash = newHash
			g.Expect(vault.amend(t.Context(), step)).To(Succeed())

			if row == "covered" {
				g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
				g.Expect(decodeFoldExchange(t, vault.files[foldExistingName]).Aliases).To(Equal([]string{foldOfferBase}))

				return
			}

			g.Expect(cli.ExportNoteHasPendingMarker(vault.files[foldOfferName])).To(BeFalse(), "the offer is accepted")
			g.Expect(string(vault.files[foldOfferName])).To(ContainSubstring("object: updated by the child"))
		})
	}
}

// Test792a22l1d_RunAmend_NoteWithoutExchangeHeaderNeedsNoHash: a pending skill runbook note has no xid, so engram
// show prints no exchange hash line for it and `amend --clear-pending` without --expect-hash succeeds.
func Test792a22l1d_RunAmend_NoteWithoutExchangeHeaderNeedsNoHash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const skillNote = "4.2026-01-04.skill-demo"

	vault := newFoldVault()
	vault.put(skillNote+".md", pendingSkillRunbookNote)

	var shown bytes.Buffer

	deps := vault.deps()
	showDeps := cli.ShowDeps{Scan: deps.Scan, Read: deps.Read}
	g.Expect(cli.RunShow(t.Context(), cli.ShowArgs{Ref: skillNote, VaultPath: "/vault"}, showDeps, &shown)).To(Succeed())
	g.Expect(shown.String()).NotTo(ContainSubstring("exchange_hash"), "there is no hash line to name")

	err := vault.amend(t.Context(), cli.AmendArgs{Target: skillNote, ClearPending: true, Pending: new(bool)})
	g.Expect(err).NotTo(HaveOccurred())

	cleared := vault.files[skillNote+".md"]
	g.Expect(cli.ExportNoteHasPendingMarker(cleared)).To(BeFalse(), "the note is live")
	g.Expect(string(cleared)).To(ContainSubstring("skill_hash: abc123"))
}

// Test792a22m1a_RunAmend_FoldUnionsAliasesAndLinks: folding offer O (an alias and parent links, one of them its
// primary) into E, which has its own primary link, deletes O and its sidecar; E's aliases gain O's basename and
// alias and E holds O's links, its own primary still the primary.
func Test792a22m1a_RunAmend_FoldUnionsAliasesAndLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offerSidecar := strings.TrimSuffix(foldOfferName, ".md") + ".vec.json"

	vault := newFoldVault()
	vault.put(foldExistingName, foldNote(foldExchange{
		vault: foldParentVault, aliases: []string{"2.2026-01-01.old-name"},
		links: []foldLink{{"9.2026-09-01.own", "offered", "xh1:own"}},
	}, false))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, aliases: []string{"5.2026-01-01.prev-name"},
		links: []foldLink{{"7.2026-09-01.pulled", "pulled", "xh1:pulled"}, {"8.2026-09-01.near", "covered", "xh1:near"}},
	}, true))
	vault.put(offerSidecar, validFoldSidecar())

	err := vault.amend(t.Context(), cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldExistingBase})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.files).NotTo(HaveKey(offerSidecar))

	folded := decodeFoldExchange(t, vault.files[foldExistingName])
	g.Expect(folded.Aliases).To(Equal([]string{"2.2026-01-01.old-name", foldOfferBase, "5.2026-01-01.prev-name"}))
	g.Expect(folded.Parent.Vault).To(Equal(foldParentVault))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "9.2026-09-01.own", Via: "offered", Hash: "xh1:own"},
		{Note: "7.2026-09-01.pulled", Via: "covered", Hash: "xh1:pulled"},
		{Note: "8.2026-09-01.near", Via: "covered", Hash: "xh1:near"},
	}), "E keeps its own primary link and holds O's links")
}

// Test792a22m1b_RunAmend_BadIntoIsRefused: --into without --discard, an --into that names no note and an --into
// that names the offer itself each fail and change nothing.
func Test792a22m1b_RunAmend_BadIntoIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args cli.AmendArgs
		fail string
	}{
		"into without discard": {
			args: cli.AmendArgs{Target: foldOfferBase, Into: foldExistingBase, ClearPending: true, Pending: new(bool)},
			fail: "--into is only valid with --discard",
		},
		"into a missing note": {
			args: cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: "99.2026-01-01.none"},
			fail: "--into note not found",
		},
		"into the offer itself": {
			args: cli.AmendArgs{Target: foldOfferBase, Discard: true, Into: foldOfferBase},
			fail: "--into must name a different note than --target",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newFoldVault()
			vault.put(foldExistingName, foldNote(foldExchange{}, false))
			vault.put(foldOfferName, foldNote(foldExchange{
				xid: foldXIDO, vault: foldParentVault, links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:p"}},
			}, true))
			vault.put(strings.TrimSuffix(foldOfferName, ".md")+".vec.json", validFoldSidecar())
			before := maps.Clone(vault.files)

			err := vault.amend(t.Context(), testCase.args)
			g.Expect(err).To(MatchError(ContainSubstring(testCase.fail)))
			g.Expect(vault.files).To(Equal(before), "nothing changes")
			g.Expect(vault.declines).To(BeEmpty())
		})
	}
}

// Test792a22m1c_Amend_FoldDoesNotRestampIdentity: folding a pulled-down offer into E with a parent configured
// leaves E's repo, user, vault, xid and content as they were, queues no offer (the outbox and the parent see
// nothing) and records no decline.
func Test792a22m1c_Amend_FoldDoesNotRestampIdentity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newCurate1Machine(t, true)
	parentNote := machine.parent.addNote(pulledFact("7.2026-09-01.folded", "c", ""))
	existing := foldNote(foldExchange{xid: foldXIDE}, false)
	machine.put(foldExistingName, existing)
	machine.putSidecar(foldExistingName)

	machine.run("activate", "--note", parentNote+".md")

	offers := machine.linkedCopies(parentNote)
	g.Expect(offers).To(HaveLen(1), "the pull-down wrote the offer")

	if len(offers) != 1 {
		return
	}

	outboxBefore := machine.read(curate1OutboxName)
	sentBefore := len(machine.parent.offers())
	declinedBefore := machine.declined()

	_, stderr := machine.run("amend", "--target", strings.TrimSuffix(offers[0], ".md"), "--discard",
		"--into", foldExistingBase, "--expect-hash", mustHash(t, []byte(machine.read(offers[0]))))
	g.Expect(stderr).To(BeEmpty())
	g.Expect(machine.exits).To(BeEmpty())
	g.Expect(machine.noteNames()).To(Equal([]string{foldExistingName}), "the offer is folded away")

	folded := machine.read(foldExistingName)
	expectOnlyExchangeKeysChanged(g, existing, []byte(folded))
	g.Expect(decodeFoldExchange(t, []byte(folded)).XID).To(Equal(foldXIDE), "E keeps its xid")
	g.Expect(machine.read(curate1OutboxName)).To(Equal(outboxBefore), "no offer is queued")
	g.Expect(machine.parent.offers()).To(HaveLen(sentBefore), "and none is sent")
	g.Expect(machine.declined()).To(Equal(declinedBefore), "no decline is recorded")
}

// Test792a22m1d_RunAmend_FoldConvertsACRLFExistingNote: E's lines end in \r\n and the fold changes its aliases and
// parent links; the fold succeeds, E is written with no \r\n, and O and its sidecar are gone.
func Test792a22m1d_RunAmend_FoldConvertsACRLFExistingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	offerSidecar := strings.TrimSuffix(foldOfferName, ".md") + ".vec.json"

	vault := newFoldVault()
	vault.put(foldExistingName, toCRLF(foldNote(foldExchange{}, false)))
	vault.put(foldOfferName, foldNote(foldExchange{
		xid: foldXIDO, vault: foldParentVault, aliases: []string{"5.2026-01-01.prev-name"},
		links: []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}},
	}, true))
	vault.put(offerSidecar, validFoldSidecar())

	deps := vault.deps()
	deps.Embedder = stubEmbedder{modelID: "stub@4", dims: 4}

	err := cli.RunAmend(t.Context(), cli.AmendArgs{
		Vault: "/vault", Target: foldOfferBase, Discard: true, Into: foldExistingBase,
	}, deps, &bytes.Buffer{})
	g.Expect(err).NotTo(HaveOccurred(), "the fold succeeds on a CRLF existing note")

	written := vault.files[foldExistingName]
	g.Expect(string(written)).NotTo(ContainSubstring("\r\n"))
	g.Expect(vault.files).NotTo(HaveKey(foldOfferName))
	g.Expect(vault.files).NotTo(HaveKey(offerSidecar))

	folded := decodeFoldExchange(t, written)
	g.Expect(folded.Aliases).To(Equal([]string{foldOfferBase, "5.2026-01-01.prev-name"}))
	g.Expect(folded.Parent.Links).To(Equal([]foldLinkYAML{
		{Note: "7.2026-09-01.parent", Via: "pulled", Hash: "xh1:parent"},
	}))
}

// Test792a22m1e_RunAmend_FoldThatChangesNothingLeavesTheExistingNoteAlone: E already holds O's basename and
// links, so the fold writes nothing to E (a CRLF E stays byte-identical) and deletes O.
func Test792a22m1e_RunAmend_FoldThatChangesNothingLeavesTheExistingNoteAlone(t *testing.T) {
	t.Parallel()

	for name, layout := range map[string]func(string) string{
		"an LF existing note":  func(note string) string { return note },
		"a CRLF existing note": toCRLF,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			held := []foldLink{{"7.2026-09-01.parent", "pulled", "xh1:parent"}}
			existing := layout(foldNote(foldExchange{
				vault: foldParentVault, aliases: []string{foldOfferBase}, links: held,
			}, false))

			vault := newFoldVault()
			vault.put(foldExistingName, existing)
			vault.put(foldOfferName, foldNote(foldExchange{xid: foldXIDO, vault: foldParentVault, links: held}, true))

			writes := make([]string, 0, 1)
			deps := vault.deps()
			write := deps.Write
			deps.Write = func(path string, data []byte) error {
				writes = append(writes, filepath.Base(path))

				return write(path, data)
			}

			err := cli.RunAmend(t.Context(), cli.AmendArgs{
				Vault: "/vault", Target: foldOfferBase, Discard: true, Into: foldExistingBase,
			}, deps, &bytes.Buffer{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(writes).To(BeEmpty(), "E is not written")
			g.Expect(string(vault.files[foldExistingName])).To(Equal(existing))
			g.Expect(vault.files).NotTo(HaveKey(foldOfferName), "O is deleted")
		})
	}
}

// unexported constants.
const (
	curate1ChunksPath    = "/curate1/chunks"
	curate1DirPerm       = fs.FileMode(0o750)
	curate1EventCapacity = 4
	curate1HomePath      = "/curate1/home"
	curate1LiveName      = "1.2026-01-01.live-fact.md"
	curate1OfferName     = "2.2026-01-02.pending-offer.md"
	curate1OutboxName    = ".engram/outbox.json"
	curate1StaleDate     = "2026-01-01"
	curate1Today         = "2026-09-28"
	curate1VaultKey      = "curate1/vault"
	curate1VaultPath     = "/curate1/vault"
)

// unexported variables.
var (
	curate1Clock = time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
)

// curate1Machine is one in-memory vault with the commands that run against it, optionally under a recording
// parent: the whole Deps of the production composition over a cv1MemFS, so no real file system, home, git or
// network is touched.
type curate1Machine struct {
	t      failer
	fsys   *cv1MemFS
	parent *recordingParent
	// parented wires ENGRAM_PARENT and the recording parent into the commands.
	parented bool
	embeds   *atomic.Int32
	rand     func([]byte) (int, error)
	exits    []int
}

// declined is the vault's declined-pull record.
func (m *curate1Machine) declined() []declinedEntry {
	m.t.Helper()

	raw := m.read(".engram/declined.json")
	if raw == "" {
		return nil
	}

	var file struct {
		Entries []declinedEntry `json:"entries"`
	}

	err := json.Unmarshal([]byte(raw), &file)
	if err != nil {
		m.t.Fatal(err)
	}

	return file.Entries
}

// deps is the production Deps composition over the machine's file system.
func (m *curate1Machine) deps(stdout, stderr io.Writer) cli.Deps {
	getenv := func(string) string { return "" }

	var fetch func(context.Context, string, string, []byte) (cli.FetchResponse, error)

	if m.parented {
		getenv = wiringEnvGetenv(parentURL, curate1HomePath)
		fetch = m.parent.fetch
	}

	return cli.Deps{
		Stdout:       stdout,
		Stderr:       stderr,
		Exit:         func(code int) { m.exits = append(m.exits, code) },
		Getenv:       getenv,
		Now:          func() time.Time { return curate1Clock },
		Getwd:        func() (string, error) { return "/", nil },
		UserHomeDir:  func() (string, error) { return curate1HomePath, nil },
		Username:     func() (string, error) { return "curator", nil },
		FS:           m.fsys,
		Lock:         fakeLocker{lock: func(string) (func() error, error) { return func() error { return nil }, nil }},
		Commander:    stubCommander{},
		Embed:        embedCounter{calls: m.embeds},
		Fetch:        fetch,
		RandRead:     m.rand,
		EvalSymlinks: identityPath,
	}
}

// lastUsed is the LastUsed of a note's sidecar.
func (m *curate1Machine) lastUsed(name string) string {
	m.t.Helper()

	sidecar, err := embed.UnmarshalSidecar([]byte(m.read(curate1SidecarName(name))))
	if err != nil {
		m.t.Fatal(err)
	}

	return sidecar.LastUsed
}

// linkedCopies lists the vault's notes with any link to parentNote under the parent's vault ID.
func (m *curate1Machine) linkedCopies(parentNote string) []string {
	m.t.Helper()

	linked := make([]string, 0, 1)

	for _, name := range m.noteNames() {
		copied := decodePulledCopy(m.t, []byte(m.read(name)))
		if copied.Parent.Vault != parentVaultID {
			continue
		}

		if slices.ContainsFunc(copied.Parent.Links, func(link pulledLink) bool { return link.Note == parentNote }) {
			linked = append(linked, name)
		}
	}

	return linked
}

// noteNames is the vault's note file names (sorted).
func (m *curate1Machine) noteNames() []string {
	names := make([]string, 0, 1)

	for path := range m.fsys.snapshot() {
		if filepath.Dir(path) == curate1VaultPath && strings.HasSuffix(path, ".md") {
			names = append(names, filepath.Base(path))
		}
	}

	slices.Sort(names)

	return names
}

// path is a file's absolute path in the vault.
func (m *curate1Machine) path(name string) string { return filepath.Join(curate1VaultPath, name) }

// put places a file in the vault without counting it as a write of the command under test.
func (m *curate1Machine) put(name, content string) { m.fsys.put(m.path(name), []byte(content)) }

// putSidecar places a sidecar for the note, last used on curate1StaleDate, with every vector the one the
// embedder answers.
func (m *curate1Machine) putSidecar(name string) {
	m.fsys.put(m.path(curate1SidecarName(name)), embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@4", Dims: 4,
		SituationVector: []float32{1, 0, 0, 0}, BodyVector: []float32{1, 0, 0, 0},
		ContentHash: "sha256:x", LastUsed: curate1StaleDate,
	}))
}

// read is a vault file's content ("" when absent).
func (m *curate1Machine) read(name string) string {
	data, err := m.fsys.ReadFile(m.path(name))
	if err != nil {
		return ""
	}

	return string(data)
}

// run executes `engram <args> --vault <vault>` through the production targets.
func (m *curate1Machine) run(args ...string) (string, string) {
	m.t.Helper()

	var stdout, stderr bytes.Buffer

	command := append(append([]string{"engram"}, args...), "--vault", curate1VaultPath)

	executed, err := targ.Execute(command, cli.Targets(m.deps(&stdout, &stderr))...)
	if err != nil {
		stderr.WriteString(executed.Output)
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stdout.String(), stderr.String()
}

// curate1CountingRand is a RandRead whose every call answers different bytes, so each minted ID is new.
func curate1CountingRand() func([]byte) (int, error) {
	var calls atomic.Uint32

	return func(buf []byte) (int, error) {
		start := byte(calls.Add(1))
		for index := range buf {
			buf[index] = start + byte(index)
		}

		return len(buf), nil
	}
}

// curate1FactNote is a fact note with the given situation and extra frontmatter lines.
func curate1FactNote(situation, extra string) string {
	return "---\ntype: fact\ntier: L2\nsituation: " + situation + "\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: test\nrepo: github.com/alice/widgets\nuser: alice\n" +
		"vault: personal\n" + extra + "---\n\nInformation learned: " + situation + ", a b c.\n"
}

// curate1LastUsed is the LastUsed of a sidecar's bytes.
func curate1LastUsed(g *WithT, raw []byte) string {
	sidecar, err := embed.UnmarshalSidecar(raw)
	g.Expect(err).NotTo(HaveOccurred())

	return sidecar.LastUsed
}

// curate1LiveNotes lists the note files of an in-memory fold vault (sorted).
func curate1LiveNotes(v *foldVault) []string {
	names := make([]string, 0, len(v.files))

	for name := range v.files {
		if strings.HasSuffix(name, ".md") {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return names
}

// curate1SidecarName is the sidecar file name of a note file.
func curate1SidecarName(note string) string { return strings.TrimSuffix(note, ".md") + ".vec.json" }

// newCurate1Machine is an empty in-memory vault, with a recording parent when asked.
func newCurate1Machine(t failer, withParent bool) *curate1Machine {
	return &curate1Machine{
		t:        t,
		fsys:     &cv1MemFS{files: fstest.MapFS{curate1VaultKey: {Mode: fs.ModeDir | curate1DirPerm}}},
		parent:   &recordingParent{},
		parented: withParent,
		embeds:   &atomic.Int32{},
		rand:     curate1CountingRand(),
	}
}
