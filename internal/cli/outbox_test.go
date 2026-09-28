package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestClassifyOfferResponse maps a /learn response to the drain's outcome
// (design D6): transport errors and 5xx fail (stop + back off), 4xx
// rejects, a 2xx with a full receipt is accepted, and a receipt without
// vault_id or basename comes from a too-old parent.
func TestClassifyOfferResponse(t *testing.T) {
	t.Parallel()

	receipt := `{"status":"offer received","luhmann":"7","basename":"7.2026-09-27.x",` +
		`"pending":true,"vault_id":"` + seqID(90) + `","stored_hash":"xh1:ab"}`

	cases := []struct {
		name    string
		resp    cli.FetchResponse
		err     error
		outcome cli.OfferOutcomeForTest
	}{
		{"transport error", cli.FetchResponse{}, errors.New("connection refused"), cli.ExportOfferFailed},
		{"500", cli.FetchResponse{Status: 500, Body: []byte(`{"error":"boom"}`)}, nil, cli.ExportOfferFailed},
		{"503", cli.FetchResponse{Status: 503}, nil, cli.ExportOfferFailed},
		{"400", cli.FetchResponse{Status: 400, Body: []byte(`{"error":"bad"}`)}, nil, cli.ExportOfferRejected},
		{"409", cli.FetchResponse{Status: 409, Body: []byte(`{"error":"cycle"}`)}, nil, cli.ExportOfferRejected},
		{"200 receipt", cli.FetchResponse{Status: 200, Body: []byte(receipt)}, nil, cli.ExportOfferAccepted},
		{
			"200 without vault_id",
			cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok","basename":"7.x","stored_hash":"xh1:ab"}`)},
			nil, cli.ExportOfferTooOld,
		},
		{
			"200 without basename",
			cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok","vault_id":"` + seqID(90) + `"}`)},
			nil, cli.ExportOfferTooOld,
		},
		{"200 undecodable", cli.FetchResponse{Status: 200, Body: []byte("not json")}, nil, cli.ExportOfferFailed},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			result := cli.ExportClassifyOfferResponse(testCase.resp, testCase.err)
			g.Expect(result.Outcome).To(Equal(testCase.outcome))

			if testCase.outcome != cli.ExportOfferAccepted {
				g.Expect(result.Err).To(HaveOccurred())
			}
		})
	}
}

// TestClassifyOfferResponse_StatusClassesProperty: every status class maps
// to one outcome — 5xx always fails, 4xx always rejects.
func TestClassifyOfferResponse_StatusClassesProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		status := rapid.IntRange(400, 599).Draw(rt, "status")
		result := cli.ExportClassifyOfferResponse(cli.FetchResponse{Status: status}, nil)

		want := cli.ExportOfferRejected
		if status >= 500 {
			want = cli.ExportOfferFailed
		}

		if result.Outcome != want {
			rt.Fatalf("status %d: outcome %v, want %v", status, result.Outcome, want)
		}
	})
}

// TestDrainOutbox_BuildsPayloadAtSendTime: an offline learn followed by an
// amend goes up as one offer carrying the latest content.
func TestDrainOutbox_BuildsPayloadAtSendTime(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "first", false)
	env.enqueue(xidA)
	env.writeNote("1.2026-09-27.a.md", xidA, "second", false)

	parent := &fakeParent{}
	result, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Sent).To(Equal(1))

	g.Expect(parent.sent).To(HaveLen(1))
	g.Expect(string(parent.sent[0].Raw)).To(ContainSubstring("second"))
	g.Expect(parent.sent[0].Hash).To(Equal(mustHash(t, env.noteRaw("1.2026-09-27.a.md"))))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_CachedSelfParentSendsNothing (D2 self-parent guard): when
// the cached parent reports this vault's own ID, nothing is sent and one
// warning names --regenerate.
func TestDrainOutbox_CachedSelfParentSendsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)
	g.Expect(cli.ExportRecordParentSuccess(env.store, env.vault, parentURL, env.localID)).To(Succeed())

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}))
	g.Expect(nonEmptyLines(env.stderr.String())).To(HaveLen(1))
	g.Expect(env.stderr.String()).To(ContainSubstring("vault-id --regenerate"))
}

// TestDrainOutbox_ChangedDuringSendStaysQueued: the receipt is recorded,
// but a note amended while its offer was in flight stays queued.
func TestDrainOutbox_ChangedDuringSendStaysQueued(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "first", false)
	env.enqueue(xidA)

	parent := &fakeParent{during: func(cli.OfferNoteForTest) {
		env.writeNote("1.2026-09-27.a.md", xidA, "amended mid-flight", false)
	}}

	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.applied).To(Equal([]string{"1.2026-09-27.a"}))
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}))
	g.Expect(env.outbox().Entries[0].State).To(Equal("queued"))
}

// TestDrainOutbox_ConcurrentEnqueueSurvives (M10): an entry another process
// queues while the drain is sending survives the merge, because the merge
// re-reads the outbox from disk.
func TestDrainOutbox_ConcurrentEnqueueSurvives(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.d.md", xidD, "d", false)
	env.enqueue(xidA)

	parent := &fakeParent{during: func(cli.OfferNoteForTest) { env.enqueue(xidD) }}

	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidD}))
}

// TestDrainOutbox_DeletedDuringSendDiscardsReceipt (M10): a receipt for a
// note deleted while its offer was in flight is discarded, and its entry is
// dropped.
func TestDrainOutbox_DeletedDuringSendDiscardsReceipt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)

	parent := &fakeParent{during: func(cli.OfferNoteForTest) {
		env.fsys.remove(filepath.Join(env.vault, "1.2026-09-27.a.md"))
	}}

	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(HaveLen(1))
	g.Expect(parent.applied).To(BeEmpty())
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_DropsDeletedAndPendingNotes: an entry whose note is gone
// or now pending is dropped without being sent.
func TestDrainOutbox_DropsDeletedAndPendingNotes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)
	env.fsys.remove(filepath.Join(env.vault, "1.2026-09-27.a.md"))
	env.writeNote("2.2026-09-27.b.md", xidB, "b", true)

	parent := &fakeParent{}
	result, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(result.Dropped).To(Equal(2))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_EmptyOutboxTouchesNothing: nothing queued means no
// location check, no send and no write.
func TestDrainOutbox_EmptyOutboxTouchesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	writesBefore := len(env.fsys.atomicWrites())

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(env.fsys.atomicWrites()).To(HaveLen(writesBefore))
	g.Expect(env.stderr.String()).To(BeEmpty())
}

// TestDrainOutbox_FindsRenamedNoteByXID: a Luhmann rename between enqueue
// and drain does not orphan the entry; the receipt lands on the new name.
func TestDrainOutbox_FindsRenamedNoteByXID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)
	env.fsys.rename(filepath.Join(env.vault, "1.2026-09-27.a.md"), filepath.Join(env.vault, "1a.2026-09-27.a.md"))

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(HaveLen(1))
	g.Expect(parent.sent[0].Basename).To(Equal("1a.2026-09-27.a"))
	g.Expect(parent.applied).To(Equal([]string{"1a.2026-09-27.a"}))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_LocalReadFailuresSurface: a corrupt outbox, a corrupt
// parent cache, or an unreadable note fails the drain (nothing is sent)
// rather than silently dropping queued offers.
func TestDrainOutbox_LocalReadFailuresSurface(t *testing.T) {
	t.Parallel()

	cases := map[string]func(env *outboxEnv){
		"corrupt outbox": func(env *outboxEnv) {
			env.fsys.put(filepath.Join(env.vault, ".engram", "outbox.json"), []byte("{not json"))
		},
		"corrupt parent cache": func(env *outboxEnv) {
			env.fsys.put(filepath.Join(env.vault, ".engram", "parent.json"), []byte("{not json"))
		},
		"unreadable note": func(env *outboxEnv) {
			env.fsys.failReadOf(filepath.Join(env.vault, "1.2026-09-27.a.md"), errors.New("EIO"))
		},
	}

	for name, breakVault := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newOutboxEnv(t)
			env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
			env.enqueue(xidA)
			breakVault(env)

			parent := &fakeParent{}
			_, err := env.drain(parent)
			g.Expect(err).To(HaveOccurred())
			g.Expect(parent.sent).To(BeEmpty())
		})
	}
}

// TestDrainOutbox_LocationCheckFailureSendsNothing (ruling S2): a vault that
// fails its location check (here: a clone, with no home.json) queues but
// never sends, and prints the location warning.
func TestDrainOutbox_LocationCheckFailureSendsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)
	env.fsys.remove(filepath.Join(env.vault, ".engram", "home.json"))

	parent := &fakeParent{}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.sent).To(BeEmpty())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}))
	g.Expect(env.stderr.String()).To(ContainSubstring("vault-id --claim"))
}

// TestDrainOutbox_RecordsParentVaultIDAndResetsFailures: an answered offer
// is a successful contact: the failure count resets and the parent's
// reported vault ID is cached.
func TestDrainOutbox_RecordsParentVaultIDAndResetsFailures(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)

	_, failErr := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(failErr).NotTo(HaveOccurred())

	_, err := env.drain(&fakeParent{})
	g.Expect(err).NotTo(HaveOccurred())

	cache, cacheErr := cli.ExportLoadParentCache(env.state, env.vault)
	g.Expect(cacheErr).NotTo(HaveOccurred())
	g.Expect(cache.Failures).To(BeZero())
	g.Expect(cache.VaultID).To(Equal(parentVaultID))
	g.Expect(cache.URL).To(Equal(parentURL))
}

// TestDrainOutbox_RejectedEntryReArmsOnlyOnHashChange: a 4xx marks the
// entry rejected with its hash and the drain continues; the entry is not
// resent until the note's exchange hash changes.
func TestDrainOutbox_RejectedEntryReArmsOnlyOnHashChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)

	first := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: rejected("bad offer")}}
	result, err := env.drain(first)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Rejected).To(Equal(1))
	g.Expect(sentXIDs(first)).To(Equal([]string{xidA, xidB}))

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA}))
	g.Expect(box.Entries[0].State).To(Equal("rejected"))
	g.Expect(box.Entries[0].RejectedHash).To(Equal(mustHash(t, env.noteRaw("1.2026-09-27.a.md"))))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("bad offer"))

	unchanged := &fakeParent{}
	_, err = env.drain(unchanged)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(unchanged.sent).To(BeEmpty())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}))

	env.writeNote("1.2026-09-27.a.md", xidA, "a, fixed", false)
	env.enqueue(xidA)

	rearmed := &fakeParent{}
	_, err = env.drain(rearmed)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sentXIDs(rearmed)).To(Equal([]string{xidA}))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_RejectionWithoutMessageRecordsEmptyError: a rejection a
// sender reports without a cause still marks the entry rejected.
func TestDrainOutbox_RejectionWithoutMessageRecordsEmptyError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)

	bare := cli.OfferSendResultForTest{Outcome: cli.ExportOfferRejected}
	_, err := env.drain(&fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: bare}})
	g.Expect(err).NotTo(HaveOccurred())

	box := env.outbox()
	g.Expect(box.Entries).To(HaveLen(1))
	g.Expect(box.Entries[0].State).To(Equal("rejected"))
	g.Expect(box.Entries[0].LastError).To(BeEmpty())
}

// TestDrainOutbox_SelfParentReceiptStops: a receipt reporting this vault's
// own ID is the self-parent case — nothing is recorded, the drain stops,
// the entry stays and one warning names --regenerate.
func TestDrainOutbox_SelfParentReceiptStops(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)

	receipt := accepted("1.2026-09-27.a")
	receipt.Receipt.VaultID = env.localID

	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: receipt}}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(parent.applied).To(BeEmpty())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}))
	g.Expect(env.stderr.String()).To(ContainSubstring("vault-id --regenerate"))
}

// TestDrainOutbox_SendsInFirstQueuedOrder: oldest queued first, whatever
// the file order.
func TestDrainOutbox_SendsInFirstQueuedOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.writeNote("3.2026-09-27.c.md", xidC, "c", false)

	base := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{
		{"xid": xidC, "queued": base.Add(2 * time.Minute), "state": "queued"},
		{"xid": xidA, "queued": base, "state": "queued"},
		{"xid": xidB, "queued": base.Add(time.Minute), "state": "queued"},
	}})

	parent := &fakeParent{}
	result, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Sent).To(Equal(3))
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA, xidB, xidC}))
	g.Expect(env.outbox().Entries).To(BeEmpty())
}

// TestDrainOutbox_SendsWithNoLockHeld: the snapshot and the merge run under
// the vault lock, but every send happens with the lock released.
func TestDrainOutbox_SendsWithNoLockHeld(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)
	env.resetEvents()

	parent := &fakeParent{events: env.events}
	_, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(env.events.list()).To(Equal([]string{
		"lock", "unlock", "send " + xidA, "send " + xidB, "lock", "apply", "apply", "unlock",
	}))
}

// TestDrainOutbox_SkippedSendKeepsEntryAndContinues: a local payload
// failure records the error on its entry, keeps it queued, and does not
// stop the drain or back off (the parent was never contacted).
func TestDrainOutbox_SkippedSendKeepsEntryAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)

	skip := cli.OfferSendResultForTest{Outcome: cli.ExportOfferSkipped, Err: errors.New("unrenderable")}
	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: skip}}

	result, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Unreachable).To(BeFalse())
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA, xidB}))

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA}))
	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("unrenderable"))

	cache, cacheErr := cli.ExportLoadParentCache(env.state, env.vault)
	g.Expect(cacheErr).NotTo(HaveOccurred())
	g.Expect(cache.Failures).To(BeZero())
}

// TestDrainOutbox_TooOldParentKeepsEntryAndErrors (H7): a receipt with no
// vault ID stops exchange with a "parent too old" error and leaves the
// entry queued.
func TestDrainOutbox_TooOldParentKeepsEntryAndErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)

	tooOld := cli.ExportClassifyOfferResponse(
		cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok","basename":"1.x"}`)}, nil)

	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: tooOld}}
	_, err := env.drain(parent)
	g.Expect(err).To(MatchError(cli.ErrParentTooOldForTest))
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA}))
	g.Expect(parent.applied).To(BeEmpty())
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA, xidB}))
}

// TestDrainOutbox_TransportFailureStopsAndRecords: a connection error stops
// the drain; the sent entry is removed, the failed and later ones remain in
// order, the failed one records attempts and last_error, and the parent
// cache backs off for 30s.
func TestDrainOutbox_TransportFailureStopsAndRecords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.writeNote("3.2026-09-27.c.md", xidC, "c", false)
	env.enqueue(xidA)
	env.enqueue(xidB)
	env.enqueue(xidC)

	failure := cli.ExportClassifyOfferResponse(cli.FetchResponse{}, errors.New("connection refused"))
	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidB: failure}}

	result, err := env.drain(parent)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result.Unreachable).To(BeTrue())
	g.Expect(sentXIDs(parent)).To(Equal([]string{xidA, xidB}))

	box := env.outbox()
	g.Expect(entryXIDs(box)).To(Equal([]string{xidB, xidC}))
	g.Expect(box.Entries[0].Attempts).To(Equal(1))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("connection refused"))
	g.Expect(box.Entries[1].Attempts).To(BeZero())

	cache, cacheErr := cli.ExportLoadParentCache(env.state, env.vault)
	g.Expect(cacheErr).NotTo(HaveOccurred())
	g.Expect(cache.Failures).To(Equal(1))
	g.Expect(cache.BackoffUntil).To(BeTemporally("==", env.now().Add(30*time.Second)))
	g.Expect(result.RetryAfter).To(BeTemporally("==", env.now().Add(30*time.Second)))
}

// TestEnqueueOutbox_CoalescesByXID: at most one entry per note; a repeat
// enqueue keeps the entry's first-queued time (and its FIFO slot).
func TestEnqueueOutbox_CoalescesByXID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	first := env.now()

	env.enqueue(xidA)
	env.advance(time.Minute)
	env.enqueue(xidB)
	env.advance(time.Minute)
	env.enqueue(xidA)

	box := env.outbox()
	g.Expect(box.Version).To(Equal(1))
	g.Expect(entryXIDs(box)).To(Equal([]string{xidA, xidB}))
	g.Expect(box.Entries[0].Queued).To(BeTemporally("==", first))
	g.Expect(box.Entries[0].State).To(Equal("queued"))
}

// TestEnqueueOutbox_FailuresSurface: a corrupt outbox, a failed atomic
// write, or a vault ID that cannot be stamped fails the enqueue.
func TestEnqueueOutbox_FailuresSurface(t *testing.T) {
	t.Parallel()

	t.Run("corrupt outbox", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newOutboxEnv(t)
		env.fsys.put(filepath.Join(env.vault, ".engram", "outbox.json"), []byte("[]"))
		g.Expect(cli.ExportEnqueueOutbox(env.store, env.vault, xidA)).NotTo(Succeed())
	})

	t.Run("write fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		env := newOutboxEnv(t)
		env.fsys.failAtomicWrite(filepath.Join(env.vault, ".engram", "outbox.json"), errors.New("disk full"))
		g.Expect(cli.ExportEnqueueOutbox(env.store, env.vault, xidA)).To(MatchError(ContainSubstring("disk full")))
	})

	t.Run("stamp fails", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		fsys := newMemVaultFS()
		fsys.mkdir("/fresh")
		fsys.failAtomicWrite("/fresh/.engram/home.json", errors.New("read-only"))

		store, _ := newTestOutboxStore(fsys, newDrainEventLog(), func() time.Time { return testNow }, io.Discard)
		g.Expect(cli.ExportEnqueueOutbox(store, "/fresh", xidA)).To(MatchError(ContainSubstring("read-only")))
		g.Expect(fsys.read("/fresh/.engram/outbox.json")).To(BeEmpty())
	})
}

// TestEnqueueOutbox_SavesAtomically: the outbox is only ever written with
// the atomic temp+rename primitive, as outbox.json under .engram/.
func TestEnqueueOutbox_SavesAtomically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.enqueue(xidA)

	g.Expect(env.fsys.atomicWrites()).To(ContainElement(filepath.Join(env.vault, ".engram", "outbox.json")))

	var decoded map[string]any
	g.Expect(json.Unmarshal(env.fsys.read(filepath.Join(env.vault, ".engram", "outbox.json")), &decoded)).To(Succeed())
	g.Expect(decoded).To(HaveKeyWithValue("version", BeNumerically("==", 1)))
}

// TestEnqueueOutbox_StampsVaultIDBeforeStateDir (ruling S4): the first
// enqueue in a vault with no ID stamps the ID (and its location record)
// before .engram/ exists, so .engram/ keeps meaning created-by-engram.
func TestEnqueueOutbox_StampsVaultIDBeforeStateDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fsys := newMemVaultFS()
	vault := "/vault"
	fsys.mkdir(vault)

	store, state := newTestOutboxStore(fsys, newDrainEventLog(), func() time.Time { return testNow }, io.Discard)

	g.Expect(cli.ExportEnqueueOutbox(store, vault, xidA)).To(Succeed())
	g.Expect(fsys.stateDirBeforeID()).To(BeFalse())

	status, checkErr := cli.ExportCheckVaultLocation(state, vault)
	g.Expect(checkErr).NotTo(HaveOccurred())
	g.Expect(status).To(Equal(cli.ExportLocationOK))
}

// TestNewOfferSender_PostsBuiltPayloadToLearn: the production sender builds
// the payload from the note at send time and POSTs it to the parent's
// /learn route; a payload build failure is skipped without contacting the
// parent.
func TestNewOfferSender_PostsBuiltPayloadToLearn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var method, url string

	var body []byte

	fetch := func(_ context.Context, gotMethod, gotURL string, gotBody []byte) (cli.FetchResponse, error) {
		method, url, body = gotMethod, gotURL, gotBody

		return cli.FetchResponse{Status: 200, Body: []byte(`{"status":"offer received","luhmann":"7",` +
			`"basename":"7.2026-09-27.x","pending":true,"vault_id":"` + parentVaultID + `","stored_hash":"xh1:ab"}`)}, nil
	}
	build := func(note cli.OfferNoteForTest) ([]byte, error) { return []byte("payload for " + note.XID), nil }

	send := cli.ExportNewOfferSender(fetch, parentURL+"/", build)
	result := send(context.Background(), cli.OfferNoteForTest{XID: xidA})

	g.Expect(result.Outcome).To(Equal(cli.ExportOfferAccepted))
	g.Expect(result.Receipt.Basename).To(Equal("7.2026-09-27.x"))
	g.Expect(method).To(Equal("POST"))
	g.Expect(url).To(Equal(parentURL + "/learn"))
	g.Expect(string(body)).To(Equal("payload for " + xidA))

	failing := cli.ExportNewOfferSender(
		func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
			t.Error("fetch must not run when the payload cannot be built")

			return cli.FetchResponse{}, nil
		},
		parentURL,
		func(cli.OfferNoteForTest) ([]byte, error) { return nil, errors.New("unrenderable") },
	)
	skipped := failing(context.Background(), cli.OfferNoteForTest{XID: xidA})
	g.Expect(skipped.Outcome).To(Equal(cli.ExportOfferSkipped))
	g.Expect(skipped.Err).To(MatchError(ContainSubstring("unrenderable")))
}

// TestOutbox_NoOfferLostAtMostOneEntryProperty (E52): across any sequence
// of writes, deletes, pending-marks and drains (with accepted, rejected and
// failed sends, and concurrent writes mid-drain), (1) the outbox holds at
// most one entry per note, and (2) every note that exists, is not pending,
// and whose current content the parent has not accepted keeps an entry.
func TestOutbox_NoOfferLostAtMostOneEntryProperty(t *testing.T) {
	t.Parallel()

	xids := []string{xidA, xidB, xidC}

	rapid.Check(t, func(rt *rapid.T) {
		env := newOutboxEnvFor(rt)
		model := newOutboxModel()

		steps := rapid.IntRange(1, 25).Draw(rt, "steps")
		for step := range steps {
			xid := rapid.SampledFrom(xids).Draw(rt, "xid")
			name := noteNameFor(xid)

			switch rapid.IntRange(0, 3).Draw(rt, "op") {
			case 0: // an offerable write: note written live and enqueued
				body := fmt.Sprintf("content %d", rapid.IntRange(0, 3).Draw(rt, "body"))
				env.writeNote(name, xid, body, false)
				env.enqueue(xid)
				model.exists[xid], model.pending[xid] = true, false
			case 1:
				env.fsys.remove(filepath.Join(env.vault, name))
				model.exists[xid] = false
			case 2:
				if model.exists[xid] {
					env.writeNote(name, xid, "pending "+xid, true)
					model.pending[xid] = true
				}
			default:
				runPropertyDrain(rt, env, model, xids, step)
			}

			checkOutboxInvariants(rt, env, model, xids)
		}
	})
}

// TestStampNoteXID_LazyAndStable: a note without an xid gets a fresh one
// the first time it enters the outbox; one that has an xid keeps it.
func TestStampNoteXID_LazyAndStable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	minted, stamped, err := cli.ExportStampNoteXID("", seqRand(7))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(BeTrue())
	g.Expect(minted).To(Equal(seqID(7)))

	kept, restamped, keepErr := cli.ExportStampNoteXID(xidA, func([]byte) (int, error) {
		t.Error("an existing xid must not draw randomness")

		return 0, nil
	})
	g.Expect(keepErr).NotTo(HaveOccurred())
	g.Expect(restamped).To(BeFalse())
	g.Expect(kept).To(Equal(xidA))
}

// unexported constants.
const (
	parentURL = "http://parent:8093"
	xidA      = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	xidB      = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	xidC      = "cccccccccccccccccccccccccccccccc"
	xidD      = "dddddddddddddddddddddddddddddddd"
)

// unexported variables.
var (
	parentVaultID = seqID(90)
	testNow       = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
)

// drainEventLog records lock/unlock/send/apply events in order.
type drainEventLog struct {
	mu     sync.Mutex
	events []string
}

func (l *drainEventLog) add(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, event)
}

func (l *drainEventLog) list() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return slices.Clone(l.events)
}

func (l *drainEventLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = nil
}

// failer is the slice of testing.TB that *rapid.T also satisfies.
type failer interface {
	Helper()
	Fatal(args ...any)
	Fatalf(format string, args ...any)
}

// fakeParent is a scripted /learn endpoint: each send answers from script
// (by xid) or with an accepted receipt, records what was sent, and runs
// during (another process's work) while the offer is in flight.
type fakeParent struct {
	script  map[string]cli.OfferSendResultForTest
	during  func(cli.OfferNoteForTest)
	events  *drainEventLog
	sent    []cli.OfferNoteForTest
	applied []string
}

func (p *fakeParent) apply(note cli.OfferNoteForTest, _ cli.OfferReceiptForTest) error {
	if p.events != nil {
		p.events.add("apply")
	}

	p.applied = append(p.applied, note.Basename)

	return nil
}

func (p *fakeParent) send(_ context.Context, note cli.OfferNoteForTest) cli.OfferSendResultForTest {
	if p.events != nil {
		p.events.add("send " + note.XID)
	}

	p.sent = append(p.sent, note)

	if p.during != nil {
		p.during(note)
	}

	if result, found := p.script[note.XID]; found {
		return result
	}

	return accepted(note.Basename)
}

// memVaultFS is an in-memory filesystem fake for the exchange-state and
// outbox adapters. It records atomic writes and whether .engram/ was ever
// created while the vault had no ID (ruling S4).
type memVaultFS struct {
	mu          sync.Mutex
	files       map[string][]byte
	dirs        map[string]bool
	atomic      []string
	dirBeforeID bool
	failRead    map[string]error
	failAtomic  map[string]error
}

func (m *memVaultFS) ListMD(dir string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := make([]string, 0, len(m.files))

	for path := range m.files {
		if filepath.Dir(path) == dir && strings.HasSuffix(path, ".md") {
			names = append(names, filepath.Base(path))
		}
	}

	sort.Strings(names)

	return names, nil
}

func (m *memVaultFS) MkdirAll(path string, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if filepath.Base(path) == ".engram" {
		if _, hasID := m.files[filepath.Join(filepath.Dir(path), ".engram-vault-id")]; !hasID {
			m.dirBeforeID = true
		}
	}

	m.dirs[path] = true

	return nil
}

func (m *memVaultFS) ReadFile(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if failure, failing := m.failRead[path]; failing {
		return nil, failure
	}

	data, found := m.files[path]
	if !found {
		return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
	}

	return slices.Clone(data), nil
}

func (m *memVaultFS) WriteFileAtomic(path string, data []byte, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if failure, failing := m.failAtomic[path]; failing {
		return failure
	}

	m.files[path] = slices.Clone(data)
	m.atomic = append(m.atomic, path)

	return nil
}

func (m *memVaultFS) WriteFileExcl(path string, data []byte, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, found := m.files[path]; found {
		return fmt.Errorf("create %s: %w", path, fs.ErrExist)
	}

	m.files[path] = slices.Clone(data)

	return nil
}

func (m *memVaultFS) atomicWrites() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.atomic)
}

func (m *memVaultFS) failAtomicWrite(path string, failure error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failAtomic[path] = failure
}

func (m *memVaultFS) failReadOf(path string, failure error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failRead[path] = failure
}

func (m *memVaultFS) mkdir(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dirs[path] = true
}

func (m *memVaultFS) put(path string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files[path] = slices.Clone(data)
}

func (m *memVaultFS) read(path string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.files[path])
}

func (m *memVaultFS) remove(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.files, path)
}

func (m *memVaultFS) rename(from, to string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files[to] = m.files[from]
	delete(m.files, from)
}

func (m *memVaultFS) stateDirBeforeID() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.dirBeforeID
}

// outboxEnv is one test vault over memVaultFS: an outbox store with a
// recording lock, a settable clock, and a stamped vault ID with a passing
// location record.
type outboxEnv struct {
	tb      failer
	fsys    *memVaultFS
	vault   string
	store   cli.OutboxStoreForTest
	state   cli.ExchangeStateForTest
	events  *drainEventLog
	stderr  *bytes.Buffer
	localID string
	clockMu sync.Mutex
	clock   time.Time
}

func (e *outboxEnv) advance(by time.Duration) {
	e.clockMu.Lock()
	defer e.clockMu.Unlock()

	e.clock = e.clock.Add(by)
}

func (e *outboxEnv) drain(parent *fakeParent) (cli.DrainResultForTest, error) {
	return cli.ExportDrainOutbox(context.Background(), e.store, e.vault, parentURL, parent.send, parent.apply)
}

func (e *outboxEnv) enqueue(xid string) {
	e.tb.Helper()

	err := cli.ExportEnqueueOutbox(e.store, e.vault, xid)
	if err != nil {
		e.tb.Fatalf("enqueue %s: %v", xid, err)
	}
}

func (e *outboxEnv) noteRaw(name string) []byte {
	return e.fsys.read(filepath.Join(e.vault, name))
}

func (e *outboxEnv) now() time.Time {
	e.clockMu.Lock()
	defer e.clockMu.Unlock()

	return e.clock
}

func (e *outboxEnv) outbox() cli.OutboxFileForTest {
	e.tb.Helper()

	box, err := cli.ExportLoadOutbox(e.state, e.vault)
	if err != nil {
		e.tb.Fatalf("load outbox: %v", err)
	}

	return box
}

func (e *outboxEnv) resetEvents() { e.events.reset() }

func (e *outboxEnv) writeNote(name, xid, body string, pending bool) {
	pendingLine := ""
	if pending {
		pendingLine = "pending: true\n"
	}

	raw := "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		pendingLine + "xid: " + xid + "\n---\n\n" + body + "\n"
	e.fsys.put(filepath.Join(e.vault, name), []byte(raw))
}

func (e *outboxEnv) writeOutbox(content any) {
	e.tb.Helper()

	encoded, err := json.Marshal(content)
	if err != nil {
		e.tb.Fatal(err)
	}

	e.fsys.put(filepath.Join(e.vault, ".engram", "outbox.json"), encoded)
}

// outboxModel is the property test's reference state: which notes exist,
// which are pending, and the hash the parent last accepted for each.
type outboxModel struct {
	exists    map[string]bool
	pending   map[string]bool
	delivered map[string]string
}

func accepted(basename string) cli.OfferSendResultForTest {
	return cli.OfferSendResultForTest{
		Outcome: cli.ExportOfferAccepted,
		Receipt: cli.OfferReceiptForTest{
			Status: "offer received", Luhmann: "9", Basename: basename, Pending: true,
			VaultID: parentVaultID, StoredHash: "xh1:stored",
		},
	}
}

func checkOutboxInvariants(rt *rapid.T, env *outboxEnv, model *outboxModel, xids []string) {
	box := env.outbox()
	seen := map[string]bool{}

	for _, entry := range box.Entries {
		if seen[entry.XID] {
			rt.Fatalf("two entries for %s: %v", entry.XID, entryXIDs(box))
		}

		seen[entry.XID] = true
	}

	for _, xid := range xids {
		if !model.exists[xid] || model.pending[xid] {
			continue
		}

		hash := mustHash(rt, env.noteRaw(noteNameFor(xid)))
		if model.delivered[xid] != hash && !seen[xid] {
			rt.Fatalf("offer for %s lost: live, undelivered (hash %s), no entry in %v", xid, hash, entryXIDs(box))
		}
	}
}

func entryXIDs(box cli.OutboxFileForTest) []string {
	xids := make([]string, 0, len(box.Entries))
	for _, entry := range box.Entries {
		xids = append(xids, entry.XID)
	}

	return xids
}

// mustHash is the note's exchange hash.
func mustHash(tb failer, raw []byte) string {
	tb.Helper()

	hash, err := cli.ExportExchangeHash(raw)
	if err != nil {
		tb.Fatal(err)
	}

	return hash
}

func newDrainEventLog() *drainEventLog { return &drainEventLog{} }

func newMemVaultFS() *memVaultFS {
	return &memVaultFS{
		files: map[string][]byte{}, dirs: map[string]bool{},
		failRead: map[string]error{}, failAtomic: map[string]error{},
	}
}

func newOutboxEnv(t *testing.T) *outboxEnv {
	t.Helper()

	return newOutboxEnvFor(t)
}

// newOutboxEnvFor builds an env whose vault has a stamped ID and a passing
// location record.
func newOutboxEnvFor(tester failer) *outboxEnv {
	tester.Helper()

	env := &outboxEnv{
		tb: tester, fsys: newMemVaultFS(), vault: "/vault", events: newDrainEventLog(),
		stderr: &bytes.Buffer{}, clock: testNow,
	}
	env.fsys.mkdir(env.vault)
	env.store, env.state = newTestOutboxStore(env.fsys, env.events, env.now, env.stderr)

	id, stampErr := cli.ExportStampVaultID(env.state, env.vault)
	if stampErr != nil {
		tester.Fatal(stampErr)
	}

	env.localID = id

	return env
}

func newOutboxModel() *outboxModel {
	return &outboxModel{exists: map[string]bool{}, pending: map[string]bool{}, delivered: map[string]string{}}
}

// newTestOutboxStore composes a store over fsys: seqRand(1) IDs, identity
// symlinks, "/" as the working dir, and a lock that records lock/unlock.
func newTestOutboxStore(
	fsys *memVaultFS, events *drainEventLog, now func() time.Time, stderr io.Writer,
) (cli.OutboxStoreForTest, cli.ExchangeStateForTest) {
	state := cli.ExportNewExchangeState(fsys, seqRand(1), identityPath, func() (string, error) { return "/", nil })
	lock := func(string) (func(), error) {
		events.add("lock")

		return func() { events.add("unlock") }, nil
	}

	return cli.ExportNewOutboxStore(state, lock, fsys.ListMD, now, stderr), state
}

func noteNameFor(xid string) string { return xid[:1] + ".2026-09-27." + xid[:1] + ".md" }

func rejected(message string) cli.OfferSendResultForTest {
	return cli.ExportClassifyOfferResponse(
		cli.FetchResponse{Status: 400, Body: []byte(`{"error":"` + message + `"}`)}, nil)
}

// runPropertyDrain drains once with a random outcome per send and an
// optional concurrent write mid-send, then updates the model's delivered
// hashes from what the parent accepted.
func runPropertyDrain(rt *rapid.T, env *outboxEnv, model *outboxModel, xids []string, step int) {
	outcomes := map[string]cli.OfferSendResultForTest{}

	for _, xid := range xids {
		switch rapid.IntRange(0, 2).Draw(rt, fmt.Sprintf("outcome-%d-%s", step, xid)) {
		case 1:
			outcomes[xid] = rejected("no")
		case 2:
			outcomes[xid] = cli.ExportClassifyOfferResponse(cli.FetchResponse{}, errors.New("down"))
		}
	}

	concurrent := rapid.SampledFrom(append([]string{""}, xids...)).Draw(rt, fmt.Sprintf("concurrent-%d", step))
	acceptedHashes := map[string]string{}
	parent := &fakeParent{script: outcomes}
	parent.during = func(note cli.OfferNoteForTest) {
		if _, scripted := outcomes[note.XID]; !scripted {
			acceptedHashes[note.XID] = note.Hash
		}

		if concurrent != "" {
			env.writeNote(noteNameFor(concurrent), concurrent, "concurrent "+note.XID, false)
			env.enqueue(concurrent)
			model.exists[concurrent], model.pending[concurrent] = true, false
			concurrent = ""
		}
	}

	_, err := env.drain(parent)
	if err != nil {
		rt.Fatalf("drain: %v", err)
	}

	maps.Copy(model.delivered, acceptedHashes)
}

func sentXIDs(parent *fakeParent) []string {
	xids := make([]string, 0, len(parent.sent))
	for _, note := range parent.sent {
		xids = append(xids, note.XID)
	}

	return xids
}
