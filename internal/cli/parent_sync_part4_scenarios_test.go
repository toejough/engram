package cli_test

// Scenario tests for the parent-sync product capability, part 4 (861c2zza): a receipt the note refuses needs
// attention (792a21w), a looping offer is refused (792a21y), the update report drains through a backoff
// (792a21za), a note with CRLF line endings exchanges as its LF form (792a21zb), activate resolves a ref locally
// first (792a21zc), a pulled-down note is an exact-content local pending offer (792a21zd) and a pull-down is
// idempotent, remembers declines and cannot loop (792a21ze). Spec: docs/specs/product/parent-sync.md.
//
// Every journey runs over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860) and a
// fake parent behind Deps.Fetch, with no network and no real home. The env's git identity is a scripted commander
// (ps2Identity), so no journey reads the machine's git configuration. The queue each drain works through is the one
// the commands themselves leave: notes learned while the parent was down. A drain that must not contact the parent
// for a query of its own runs as `engram update`'s exchange step does (ps4Drain). The scenarios that name a call or
// its absence (792a21w1b, w1e, w1f, y1c and y1e) drive Deps.Fetch through MockPvc1Fetch: a request the script does not
// expect holds the drain, and the run reports it as unfinished.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a21w1a_Targets_Query_ARefusedReceiptNeedsAttention verifies 792a21w1a (A refused receipt needs
// attention): the parent accepts the offer of note L, whose parent: carries an anchor that another key aliases;
// the receipt is applied, L stays byte-identical, its entry is in the attention state with the reason and the
// receipt recorded, and stderr carries one warning naming L and parent.
func Test792a21w1a_Targets_Query_ARefusedReceiptNeedsAttention(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the parent accepts the offer of local note L, and L's parent: carries an anchor that another key aliases
	env := ps2Env(t)
	note := ps4Anchored(t, env)

	// When the receipt is applied
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then L is byte-identical
	g.Expect(env.parent.offers()).To(HaveLen(1), "the parent accepted the offer")
	g.Expect(readFileString(t, note.path)).To(Equal(note.anchored))

	// and the entry is in the attention state with the reason and the receipt recorded
	box := ps3Outbox(t, env)
	g.Expect(box.Entries).To(HaveLen(1))

	if len(box.Entries) != 1 {
		return
	}

	entry := box.Entries[0]
	g.Expect(entry.State).To(Equal("attention"))
	g.Expect(entry.LastError).To(ContainSubstring("YAML anchor: parent"))
	g.Expect(entry.SentHash).To(Equal(env.exchangeHashOf(filepath.Base(note.path))), "the hash that was sent is kept")
	g.Expect(entry.Receipt).NotTo(BeNil())

	if entry.Receipt != nil {
		g.Expect(entry.Receipt.VaultID).To(Equal(parentVaultID))
		g.Expect(entry.Receipt.Basename).To(HaveSuffix("." + ps4Slug))
	}

	// and stderr carries one warning naming L and parent
	lines := nonEmptyLines(stderr)
	g.Expect(lines).To(HaveLen(1))

	if len(lines) == 1 {
		g.Expect(lines[0]).To(ContainSubstring(noteBasename(note.path)))
		g.Expect(lines[0]).To(ContainSubstring("parent"))
	}
}

// Test792a21w1b_UpdateExchange_AnAttentionEntryIsNotSentAgain verifies 792a21w1b (An attention entry is not sent
// again): with an attention entry for note L and L unchanged, two more drains send nothing for L, print no further
// warning, and leave the entry in the attention state.
func Test792a21w1b_UpdateExchange_AnAttentionEntryIsNotSentAgain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an attention entry for L, L unchanged
	env := ps2Env(t)
	note := ps4Attention(t, env)

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}

	// When the outbox drains twice more
	for range 2 {
		run := ps4Drain(env, ps3Script(env, calls, record))

		// Then no further warning is printed
		g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
		g.Expect(run.scripted).To(BeTrue())
		g.Expect(run.stderr).To(BeEmpty())
	}

	// and nothing is sent for L, and the entry stays in the attention state
	g.Expect(record.requests()).To(BeEmpty(), "the parent receives no request")

	box := ps3Outbox(t, env)
	g.Expect(box.Entries).To(HaveLen(1))

	if len(box.Entries) == 1 {
		g.Expect(box.Entries[0].State).To(Equal("attention"))
		g.Expect(box.Entries[0].Receipt).NotTo(BeNil())
	}

	g.Expect(readFileString(t, note.path)).To(Equal(note.anchored))
}

// Test792a21w1c_Targets_Query_AKeptReceiptIsRecordedWhileTheParentIsBackedOff verifies 792a21w1c (A kept receipt
// is recorded while the parent is backed off): once the anchor is removed from L's parent: while the parent is
// inside its backoff window, another command that drains records the kept receipt on L and finishes the entry,
// and the parent receives no request.
func Test792a21w1c_Targets_Query_AKeptReceiptIsRecordedWhileTheParentIsBackedOff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the anchor is removed from L's parent: while the parent is inside its backoff window
	env := ps2Env(t)
	note := ps4Attention(t, env)
	env.plant(filepath.Base(note.path), []byte(note.plain))
	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"vault_id":%q,"backoff_until":%q,"failures":3}`,
		env.parentURL, parentVaultID, ps4FarFuture))

	requestsBefore := len(env.parent.requests())
	xid := decodePulledCopy(t, []byte(note.plain)).XID

	// When another command drains
	another := env.learnFact("another")

	// Then the kept receipt is recorded on L
	linked := readFileString(t, note.path)
	g.Expect(linked).To(ContainSubstring("via: offered"))
	g.Expect(linked).To(ContainSubstring("note: "+ps4ReceiptBasename), "the kept receipt's note is the link")

	// and the entry is done: only the new note's entry is left
	g.Expect(entryXIDs(ps3Outbox(t, env))).To(Equal(ps3XIDs(t, []string{another})))
	g.Expect(entryXIDs(ps3Outbox(t, env))).NotTo(ContainElement(xid))

	// and the parent receives no request
	g.Expect(env.parent.requests()).To(HaveLen(requestsBefore))
	g.Expect(env.lastStderr).To(ContainSubstring("parent unreachable (retry after "), "the window is active")
}

// Test792a21w1d_Targets_Query_AKeptReceiptFromAnotherParentIsDiscarded verifies 792a21w1d (A kept receipt from
// another parent is discarded): an attention entry that keeps a receipt whose vault ID is not the configured
// parent's is discarded by the next draining command with one warning; L is not written, and the entry is queued
// for the current parent with no kept receipt.
func Test792a21w1d_Targets_Query_AKeptReceiptFromAnotherParentIsDiscarded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an attention entry that keeps a receipt whose vault ID is not the configured parent's, on a note that
	// would take the receipt now
	env := ps2Env(t)
	note := ps4Attention(t, env)
	env.plant(filepath.Base(note.path), []byte(note.plain))
	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"vault_id":%q,"backoff_until":%q,"failures":3}`,
		env.parentURL, parentVaultID, ps4FarFuture))
	ps3RewriteOutbox(t, env, func(entries []map[string]any) []map[string]any {
		for _, entry := range entries {
			receipt, isMap := entry["receipt"].(map[string]any)
			if isMap {
				receipt["vault_id"] = seqID(ps4OtherVaultSeed)
			}
		}

		return entries
	})

	xid := decodePulledCopy(t, []byte(note.plain)).XID

	// When a command drains
	env.learnFact("another")

	// Then L is not written
	g.Expect(readFileString(t, note.path)).To(Equal(note.plain))

	// and one warning says the kept receipt is discarded
	discarded := slices.DeleteFunc(nonEmptyLines(env.lastStderr), func(line string) bool {
		return !strings.Contains(line, "discarding the kept receipt")
	})
	g.Expect(discarded).To(HaveLen(1))

	// and the entry is queued for the current parent with no kept receipt
	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(ContainElement(xid))

	for _, entry := range box.Entries {
		if entry.XID == xid {
			g.Expect(entry.State).To(Equal("queued"))
			g.Expect(entry.Receipt).To(BeNil())
		}
	}
}

// Test792a21w1e_UpdateExchange_AnAttentionEntryResumesOnceTheNoteIsFixed verifies 792a21w1e (An attention entry
// resumes once the note is fixed): with the anchor removed from L's parent:, the next drain records the kept
// receipt on L without contacting the parent, and the entry is removed, or queued again when L's exchange hash
// changed since it was sent.
func Test792a21w1e_UpdateExchange_AnAttentionEntryResumesOnceTheNoteIsFixed(t *testing.T) {
	t.Parallel()

	t.Run("unchanged", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given the anchor is removed from L's parent:
		env := ps2Env(t)
		note := ps4Attention(t, env)
		env.plant(filepath.Base(note.path), []byte(note.plain))

		fetch, calls := MockPvc1Fetch(t)
		ps3UseFetch(env, fetch)

		record := &ps3Record{}

		// When the outbox drains
		run := ps4Drain(env, ps3Script(env, calls, record))

		// Then the kept receipt is recorded on L without contacting the parent, and the entry is removed
		g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
		g.Expect(run.scripted).To(BeTrue())
		g.Expect(record.requests()).To(BeEmpty())
		g.Expect(readFileString(t, note.path)).To(ContainSubstring("note: " + ps4ReceiptBasename))
		g.Expect(readFileString(t, note.path)).To(ContainSubstring("via: offered"))
		g.Expect(ps3Outbox(t, env).Entries).To(BeEmpty())
	})

	t.Run("changed since it was sent", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given the anchor is removed from L's parent: and L's content also changed since it was sent
		env := ps2Env(t)
		note := ps4Attention(t, env)
		changed := strings.Replace(note.plain, "object: c\n", "object: changed\n", 1)
		g.Expect(changed).NotTo(Equal(note.plain), "the fixture changes the note")
		env.plant(filepath.Base(note.path), []byte(changed))

		fetch, calls := MockPvc1Fetch(t)
		ps3UseFetch(env, fetch)

		record := &ps3Record{}
		unreachable := func(Pvc1FetchMockArgs) (cli.FetchResponse, error) {
			return cli.FetchResponse{}, errPs3ConnectionRefused
		}

		// When the outbox drains
		run := ps4Drain(env, ps3Script(env, calls, record, ps3Offer(unreachable)))

		// Then the kept receipt is recorded on L, and the entry is queued again: the parent is asked for the
		// changed note's offer only, and the failed send leaves the entry queued
		g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
		g.Expect(run.scripted).To(BeTrue())
		g.Expect(readFileString(t, note.path)).To(ContainSubstring("note: " + ps4ReceiptBasename))

		requests := record.requests()
		g.Expect(requests).To(HaveLen(1), "the receipt was recorded without a request to the parent")

		offers := record.offers(t)
		g.Expect(offers).To(HaveLen(1))

		if len(offers) == 1 {
			g.Expect(offers[0].Object).To(Equal("changed"))
		}

		box := ps3Outbox(t, env)
		g.Expect(box.Entries).To(HaveLen(1))

		if len(box.Entries) == 1 {
			g.Expect(box.Entries[0].State).To(Equal("queued"))
			g.Expect(box.Entries[0].Receipt).To(BeNil())
		}
	})
}

// Test792a21w1f_UpdateExchange_AnAttentionEntryWithoutAReceiptIsSent verifies 792a21w1f (An attention entry
// without a receipt is sent): an attention entry that holds no receipt has its offer built and sent as a queued
// one.
func Test792a21w1f_UpdateExchange_AnAttentionEntryWithoutAReceiptIsSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an attention entry that holds no receipt
	env := ps2Env(t)
	paths := ps3Backlog(t, env, "unreceipted")
	ps4WriteCache(env, parentVaultID, 0)
	ps3RewriteOutbox(t, env, func(entries []map[string]any) []map[string]any {
		for _, entry := range entries {
			entry["state"] = "attention"
			entry["last_error"] = "left by an older binary"
		}

		return entries
	})

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}

	// When the outbox drains
	run := ps4Drain(env, ps3Script(env, calls, record, ps3Offer()))

	// Then its offer is built and sent as a queued one
	g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue())
	g.Expect(ps3Slugs(record.offers(t))).To(Equal([]string{"unreceipted"}))
	g.Expect(readFileString(t, paths[0])).To(ContainSubstring("via: offered"), "the receipt is recorded")
	g.Expect(ps3Outbox(t, env).Entries).To(BeEmpty())
}

// Test792a21w1g_UpdateExchange_ARetryFailureIsReported verifies 792a21w1g (A retry failure is reported): an
// attention entry whose kept receipt still cannot be recorded for a reason other than the frontmatter has the
// failure reported by the drain, and the entry stays in attention.
func Test792a21w1g_UpdateExchange_ARetryFailureIsReported(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an attention entry whose kept receipt still cannot be recorded, for a reason other than the
	// frontmatter: the note is fixed, and its write is refused
	env := ps2Env(t)
	note := ps4Attention(t, env)
	env.plant(filepath.Base(note.path), []byte(note.plain))

	failing := &atomic.Bool{}
	failing.Store(true)

	identity := ps2Identity(ps2User)
	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		deps.FS = ps3NoteWriteFS{EdgeFS: deps.FS, failing: failing}
	}

	// When the outbox drains
	run := ps4Drain(env, nil)

	// Then the failure is reported
	g.Expect(run.finished).To(BeTrue())
	g.Expect(run.stderr).To(ContainSubstring("recording the kept receipt on " + noteBasename(note.path)))
	g.Expect(run.stderr).To(ContainSubstring(errPs3NoteWrite.Error()))

	// and the entry stays in attention
	box := ps3Outbox(t, env)
	g.Expect(box.Entries).To(HaveLen(1))

	if len(box.Entries) == 1 {
		g.Expect(box.Entries[0].State).To(Equal("attention"))
		g.Expect(box.Entries[0].LastError).To(ContainSubstring(errPs3NoteWrite.Error()))
		g.Expect(box.Entries[0].Receipt).NotTo(BeNil(), "the receipt is kept")
	}

	g.Expect(readFileString(t, note.path)).To(Equal(note.plain), "nothing is recorded")
}

// Test792a21w1h_UpdateExchange_AnAttentionEntryOfADeletedNoteIsDropped verifies 792a21w1h (An attention entry of
// a deleted note is dropped): an attention entry whose note was deleted is dropped by the next drain.
func Test792a21w1h_UpdateExchange_AnAttentionEntryOfADeletedNoteIsDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an attention entry whose note was deleted
	env := ps2Env(t)
	note := ps4Attention(t, env)

	g.Expect(os.Remove(note.path)).To(Succeed())
	g.Expect(os.Remove(strings.TrimSuffix(note.path, ".md") + ".vec.json")).To(Succeed())
	g.Expect(ps3Outbox(t, env).Entries).To(HaveLen(1), "the entry outlives its note")

	// When the outbox drains
	run := ps4Drain(env, nil)

	// Then the entry is dropped
	g.Expect(run.finished).To(BeTrue())
	g.Expect(ps3Outbox(t, env).Entries).To(BeEmpty())
	g.Expect(run.stderr).To(BeEmpty())
	g.Expect(env.parent.offers()).To(HaveLen(1), "nothing more is sent")
}

// Test792a21y1a_UpdateExchange_ACycleRejectsTheEntryAndContinues verifies 792a21y1a (A cycle rejects the entry
// and continues): a parent that refuses an offer as a loop and is not this vault gets that entry rejected, the
// drain continues with the next entry, one warning names the refusing vault and not --regenerate, and the parent
// is not backed off.
func Test792a21y1a_UpdateExchange_ACycleRejectsTheEntryAndContinues(t *testing.T) {
	t.Parallel()

	t.Run("the entry is rejected and the drain continues", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a parent that refuses the first offer as a loop and is not this vault
		env := ps2Env(t)
		hook := &ps3Hook{}
		ps3Install(env, hook)

		paths := ps3Backlog(t, env, "looped", "after")
		xids := ps3XIDs(t, paths)

		ps4WriteCache(env, parentVaultID, 0)

		refusing := seqID(ps4OtherVaultSeed)
		hook.offer = func(args cli.LearnArgs) *ps3Answer {
			if args.Slug == "looped" {
				return ps4LoopRefusal(refusing)
			}

			return nil
		}

		// When the outbox drains
		run := ps4Drain(env, nil)

		// Then that entry is rejected and the drain continues
		g.Expect(run.finished).To(BeTrue())
		g.Expect(hook.attempted()).To(Equal([]string{"looped", "after"}))
		g.Expect(readFileString(t, paths[1])).To(ContainSubstring("via: offered"), "the next entry is accepted")

		box := ps3Outbox(t, env)
		g.Expect(entryXIDs(box)).To(Equal(xids[:1]))

		if len(box.Entries) == 1 {
			g.Expect(box.Entries[0].State).To(Equal("rejected"))
		}

		// and one warning names the refusing vault and not --regenerate
		lines := nonEmptyLines(run.stderr)
		g.Expect(lines).To(HaveLen(1))

		if len(lines) == 1 {
			g.Expect(lines[0]).To(ContainSubstring(refusing))
			g.Expect(lines[0]).NotTo(ContainSubstring("--regenerate"))
		}
	})

	t.Run("the parent is not backed off", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		// Given a parent that refuses the only offer as a loop and is not this vault, after earlier failures
		env := ps2Env(t)
		hook := &ps3Hook{}
		ps3Install(env, hook)

		ps3Backlog(t, env, "looped")

		const earlierFailures = 3

		ps4WriteCache(env, parentVaultID, earlierFailures)
		hook.offer = func(cli.LearnArgs) *ps3Answer { return ps4LoopRefusal(seqID(ps4OtherVaultSeed)) }

		before := ps3Cache(t, env)
		g.Expect(before.Failures).To(Equal(earlierFailures))

		// When the outbox drains
		run := ps4Drain(env, nil)

		// Then the refusal is an answer: the failure count is reset and no backoff window is set
		g.Expect(run.finished).To(BeTrue())

		after := ps3Cache(t, env)
		g.Expect(after.Failures).To(BeZero())
		g.Expect(after.BackoffUntil).To(BeZero())
	})
}

// Test792a21y1b_UpdateExchange_AnUnmarkedConflictIsAnOrdinaryRejection verifies 792a21y1b (An unmarked conflict
// is an ordinary rejection): a parent that answers 409 without the loop mark gets the entry rejected and the
// drain continues.
func Test792a21y1b_UpdateExchange_AnUnmarkedConflictIsAnOrdinaryRejection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a parent that answers the first offer 409 without the loop mark
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "conflicted", "after")
	xids := ps3XIDs(t, paths)

	ps4WriteCache(env, parentVaultID, 0)

	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "conflicted" {
			return &ps3Answer{resp: cli.FetchResponse{Status: ps4Conflict, Body: []byte(`{"error":"conflicting note"}`)}}
		}

		return nil
	}

	// When the outbox drains
	run := ps4Drain(env, nil)

	// Then the entry is rejected and the drain continues
	g.Expect(run.finished).To(BeTrue())
	g.Expect(hook.attempted()).To(Equal([]string{"conflicted", "after"}))
	g.Expect(readFileString(t, paths[1])).To(ContainSubstring("via: offered"), "the next entry is accepted")

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids[:1]))

	if len(box.Entries) == 1 {
		g.Expect(box.Entries[0].State).To(Equal("rejected"))
		g.Expect(box.Entries[0].LastError).To(ContainSubstring("conflicting note"))
	}

	g.Expect(run.stderr).NotTo(ContainSubstring("cycle"), "an unmarked conflict is not read as a loop")
}

// Test792a21y1c_UpdateExchange_ASelfParentRefusalWaits verifies 792a21y1c (A self-parent refusal waits): a parent
// that refuses an offer as a loop and reports this vault's own ID leaves every entry queued and none rejected,
// caches the parent's ID, stops the drain after the refused send, and prints one warning that names engram
// vault-id --regenerate.
func Test792a21y1c_UpdateExchange_ASelfParentRefusalWaits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a parent that refuses an offer as a loop and reports this vault's own ID, with two entries queued
	env := ps2Env(t)
	paths := ps3Backlog(t, env, "first", "second")
	xids := ps3XIDs(t, paths)

	ps4WriteCache(env, parentVaultID, 0)

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}
	own := func(Pvc1FetchMockArgs) (cli.FetchResponse, error) { return ps4LoopRefusal(env.localID()).resp, nil }

	// When the outbox drains: a second send would hold the drain on the mock
	run := ps4Drain(env, ps3Script(env, calls, record, ps3Offer(own)))

	// Then no entry is accepted or rejected, and the drain stops
	g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue())
	g.Expect(ps3Slugs(record.offers(t))).To(Equal([]string{"first"}), "the drain stops at the refusal")

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids))

	for _, entry := range box.Entries {
		g.Expect(entry.State).To(Equal("queued"))
	}

	for _, path := range paths {
		g.Expect(readFileString(t, path)).NotTo(ContainSubstring("via: offered"))
	}

	// and the parent's ID is cached
	g.Expect(env.parentCache().VaultID).To(Equal(env.localID()))

	// and one warning names engram vault-id --regenerate
	lines := nonEmptyLines(run.stderr)
	g.Expect(lines).To(HaveLen(1))

	if len(lines) == 1 {
		g.Expect(lines[0]).To(ContainSubstring("engram vault-id --regenerate"))
	}
}

// Test792a21y1d_UpdateExchange_ASelfParentReceiptIsNotRecorded verifies 792a21y1d (A self-parent receipt is not
// recorded): a receipt that reports this vault's own ID records nothing, stops the drain, leaves the entry
// neither sent on to a next one nor rejected, and prints one warning that names engram vault-id --regenerate.
func Test792a21y1d_UpdateExchange_ASelfParentReceiptIsNotRecorded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a receipt that reports this vault's own ID, for the first of two queued entries
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "first", "second")
	xids := ps3XIDs(t, paths)

	ps4WriteCache(env, parentVaultID, 0)

	own := ps3Receipt(t, cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "1100", Basename: "1100.2026-09-28.first", Pending: true,
		VaultID: env.localID(), StoredHash: "xh1:stored",
	})
	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "first" {
			return &ps3Answer{resp: cli.FetchResponse{Status: ps4OK, Body: []byte(own)}}
		}

		return nil
	}

	// When the outbox drains
	run := ps4Drain(env, nil)

	// Then nothing is recorded and the drain stops
	g.Expect(run.finished).To(BeTrue())
	g.Expect(hook.attempted()).To(Equal([]string{"first"}), "the drain stops")

	for _, path := range paths {
		g.Expect(readFileString(t, path)).NotTo(ContainSubstring("via: offered"), "nothing is recorded")
	}

	// and the entries are neither removed nor rejected
	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids))

	for _, entry := range box.Entries {
		g.Expect(entry.State).To(Equal("queued"))
	}

	// and one warning names engram vault-id --regenerate
	lines := nonEmptyLines(run.stderr)
	g.Expect(lines).To(HaveLen(1))

	if len(lines) == 1 {
		g.Expect(lines[0]).To(ContainSubstring("engram vault-id --regenerate"))
	}
}

// Test792a21y1e_UpdateExchange_ACachedSelfParentSendsNothing verifies 792a21y1e (A cached self-parent sends
// nothing): when the cached parent vault ID equals this vault's own, the drain sends nothing and one warning
// names engram vault-id --regenerate.
func Test792a21y1e_UpdateExchange_ACachedSelfParentSendsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the cached parent vault ID equals this vault's own
	env := ps2Env(t)
	paths := ps3Backlog(t, env, "waiting")
	xids := ps3XIDs(t, paths)

	ps4WriteCache(env, env.localID(), 0)

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}

	// When the outbox drains: any request would hold the drain on the mock
	run := ps4Drain(env, ps3Script(env, calls, record))

	// Then nothing is sent
	g.Expect(run.finished).To(BeTrue(), "the drain made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue())
	g.Expect(record.requests()).To(BeEmpty())
	g.Expect(entryXIDs(ps3Outbox(t, env))).To(Equal(xids), "the entry stays queued")

	// and one warning names engram vault-id --regenerate
	lines := nonEmptyLines(run.stderr)
	g.Expect(lines).To(HaveLen(1))

	if len(lines) == 1 {
		g.Expect(lines[0]).To(ContainSubstring("engram vault-id --regenerate"))
	}
}

// Test792a21za1d_UpdateExchange_UpdateDrainsThroughABackoff verifies 792a21za1d (Update drains through a
// backoff): with a backlog and an active backoff window, engram update's exchange step drains the backlog when
// the parent answers, and the next update reports an idle outbox.
func Test792a21za1d_UpdateExchange_UpdateDrainsThroughABackoff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a backlog and an active backoff window
	env := ps2Env(t)
	slugs := []string{"first", "second"}
	paths := ps3Backlog(t, env, slugs...)

	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"backoff_until":%q,"failures":3}`,
		env.parentURL, ps4FarFuture))

	requestsBefore := len(env.parent.requests())

	_, inside := env.run(ps3QueryArgs(env)...)
	g.Expect(inside).To(ContainSubstring("parent unreachable (retry after "), "the window blocks an ordinary command")
	g.Expect(env.parent.requests()).To(HaveLen(requestsBefore), "nothing contacts the parent inside the window")
	g.Expect(env.outboxEntries()).To(HaveLen(len(slugs)))

	// When the user runs engram update and the parent answers
	first := ps4Drain(env, nil)

	// Then the backlog drains
	g.Expect(first.finished).To(BeTrue())
	g.Expect(ps3Slugs(env.parent.offers())).To(Equal(slugs))
	g.Expect(env.outboxEntries()).To(BeEmpty())

	for _, path := range paths {
		g.Expect(readFileString(t, path)).To(ContainSubstring("via: offered"))
	}

	// and the next update reports an idle outbox
	second := ps4Drain(env, nil)
	g.Expect(second.finished).To(BeTrue())
	g.Expect(second.stdout).To(BeEmpty())
	g.Expect(ps3Cache(t, env).Failures).To(BeZero())
}

// Test792a21zb1a_ExchangeHash_ACRLFNoteHashesAsItsLFForm verifies 792a21zb1a (A CRLF note hashes as its LF
// form): a note whose frontmatter and body end in \r\n has the exchange hash of its LF form, and a different note
// does not.
func Test792a21zb1a_ExchangeHash_ACRLFNoteHashesAsItsLFForm(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note whose frontmatter and body end in \r\n
	lfForm := string(offerTestNote{}.render(t))
	crlf := strings.ReplaceAll(lfForm, "\n", "\r\n")
	g.Expect(strings.Count(crlf, "\r\n")).To(Equal(strings.Count(lfForm, "\n")), "every line ends in CRLF")

	// When its exchange hash is computed
	crlfHash, crlfErr := cli.ExportExchangeHash([]byte(crlf))
	g.Expect(crlfErr).NotTo(HaveOccurred())

	lfHash, lfErr := cli.ExportExchangeHash([]byte(lfForm))
	g.Expect(lfErr).NotTo(HaveOccurred())

	// Then it equals the exchange hash of its LF form
	g.Expect(crlfHash).To(Equal(lfHash))
	g.Expect(crlfHash).To(HavePrefix("xh1:"))

	// and the hash tells notes apart
	other, otherErr := cli.ExportExchangeHash(offerTestNote{object: "another"}.render(t))
	g.Expect(otherErr).NotTo(HaveOccurred())
	g.Expect(other).NotTo(Equal(crlfHash))
}

// Test792a21zb1c_Targets_Query_ACRLFPendingOfferStaysPending verifies 792a21zb1c (A CRLF pending offer stays
// pending): a note that carries pending: true and has CRLF line endings is excluded by engram query, and the
// served raw show refuses it as a pending offer, while a live CRLF note is served.
func Test792a21zb1c_Targets_Query_ACRLFPendingOfferStaysPending(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note that carries pending: true and has CRLF line endings, beside a live note
	env := ps2Env(t)
	ps1PlantLocalNote(t, env.vault)
	plantRealVaultNote(t, env.vault, "2.pending.md",
		toCRLF("---\ntype: fact\ntier: L2\nsituation: x\npending: true\n---\n\npending body\n"), []float32{1, 0, 0, 0})

	// When engram query runs
	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	// Then query excludes it
	g.Expect(stderr).To(BeEmpty())

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(ps1Paths(answer)).To(Equal([]string{"1.fact.md"}), "the live note is returned and the pending one is not")
	g.Expect(answer.PendingOffers).To(BeTrue(), "the CRLF note counts as a pending offer")

	// and when the note is requested over the served raw show, it is refused as a pending offer
	parentVault := newServeVault(t)
	writeVaultNote(t, parentVault, "2.2026-01-02.pending.md", toCRLF(pendingOfferFactNote("2", "")))
	writeVaultNote(t, parentVault, "3.2026-01-03.live.md", toCRLF(liveFactNote("3", "")))

	routes := cli.ServeRoutes(serveTestDeps(), parentVault, "personal", "")
	show := func(note string) cli.ServeResponse {
		return routeFor(t, routes, "/show").Serve(t.Context(), cli.ServeRequest{
			Query: map[string][]string{"note": {note}, "raw": {"1"}},
		})
	}

	refused := show("2.2026-01-02.pending")
	g.Expect(refused.Status).To(Equal(ps4NotFound))
	g.Expect(show("3.2026-01-03.live").Status).To(Equal(ps4OK), "a live CRLF note is served")
}

// Test792a21zc1h_Activate_AMissWithNoParentFailsTheRef verifies 792a21zc1h (A miss with no parent fails the
// ref): with ENGRAM_PARENT unset and the ref not local, engram activate --note reports the ref as failed and
// fetches nothing.
func Test792a21zc1h_Activate_AMissWithNoParentFailsTheRef(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given ENGRAM_PARENT is unset and the ref is not local
	env := ps2Env(t)
	env.parentURL = ""
	env.plant("3.2026-09-20.local.md", offerTestNote{}.render(t))

	// When the user runs engram activate --note <ref>
	_, stderr := env.run("activate", "--note", "7.2026-09-01.gone.md")

	// Then the ref is reported as failed and nothing is fetched
	g.Expect(stderr).To(ContainSubstring("7.2026-09-01.gone.md"))
	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.parent.requests()).To(BeEmpty())
}

// Test792a21zd1a_Activate_APulledNoteIsPendingAndLinked verifies 792a21zd1a (Pulled note is pending and linked):
// pulling down parent note P writes a new local note that carries pending true, a via pulled primary link to P
// with P's exchange hash, and parent.author equal to P's authorship; it is excluded from local query results, and
// the next query payload has pending_offers true.
func Test792a21zd1a_Activate_APulledNoteIsPendingAndLinked(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given parent note P
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.pulled", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	// When P is pulled down
	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then a new local note exists carrying pending true, a via pulled primary link to P with P's exchange hash,
	// and parent.author equal to P's authorship
	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, files[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: hash}}))
	g.Expect(copied.Parent.Author).To(Equal(pulledAuthor{Repo: "github.com/parent/repo", User: "bob", Vault: "team"}))

	// and it is excluded from local query results, while a live local note is not, and the next query payload has
	// pending_offers true
	plantRealVaultNote(t, env.vault, "9.2026-09-01.control.md",
		"---\ntype: fact\ntier: L2\nsituation: pulling\n---\n\ncontrol body\n", []float32{1, 0, 0, 0})

	stdout, _ := env.run("query", "--phrase", "pulling", "--chunks-dir", t.TempDir())
	answer := ps1ParseAnswer(t, stdout)
	g.Expect(ps1Paths(answer)).To(Equal([]string{"9.2026-09-01.control.md"}))
	g.Expect(answer.PendingOffers).To(BeTrue())
}

// Test792a21zd1b_Activate_RemoteIdentityAndRegistrationFieldsAreStripped verifies 792a21zd1b (Remote identity and
// registration fields are stripped): a fetched parent note that carries skill_hash, skill_key, aliases, xid and
// its own parent link is pulled down as a local pending note that carries none of those values, and whose parent
// link is the one this pull-down set.
func Test792a21zd1b_Activate_RemoteIdentityAndRegistrationFieldsAreStripped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a fetched parent note that carries skill_hash, skill_key, aliases, xid and its own parent link
	env := ps2Env(t)
	runbook := pulledRunbook("8.2026-09-02.stripped")
	runbook.content = strings.Replace(runbook.content, "\n---\n\n1.",
		"\naliases:\n    - 4.2026-08-01.older-name\nparent:\n  vault: "+xidD+
			"\n  links:\n    - note: 1.2026-01-01.grand\n      via: offered\n      hash: xh1:grand\n---\n\n1.", 1)
	g.Expect(runbook.content).To(ContainSubstring("older-name"), "the fixture adds the parent's own aliases and link")

	parentNote := env.parent.addNote(runbook)
	hash := env.parent.hashOf(t, parentNote)

	// When it is pulled down
	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then the local pending note carries none of those values
	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	raw := readFileString(t, filepath.Join(env.vault, files[0]))
	copied := decodePulledCopy(t, []byte(raw))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.SkillHash).To(BeEmpty())
	g.Expect(copied.SkillKey).To(BeEmpty())
	g.Expect(copied.SkillSource).To(BeEmpty())
	g.Expect(copied.Aliases).To(BeEmpty())
	g.Expect(copied.XID).To(MatchRegexp(`^[0-9a-f]{32}$`))
	g.Expect(copied.XID).NotTo(Equal(xidC), "the parent's own xid is not carried")
	g.Expect(raw).NotTo(ContainSubstring("older-name"))
	g.Expect(raw).NotTo(ContainSubstring("grand"))

	// and its parent link is the one this pull-down set
	g.Expect(copied.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: hash}}))
}

// Test792a21zd1c_Activate_ChunkProvenanceDoesNotTravel verifies 792a21zd1c (Chunk provenance does not travel): a
// fetched parent note that carries sources chunk IDs is pulled down as a local note that carries no sources.
func Test792a21zd1c_Activate_ChunkProvenanceDoesNotTravel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a fetched parent note that carries sources chunk IDs
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.sourced", "c", "sources:\n    - chunk.jsonl#1\n"))
	g.Expect(env.parent.notes[0].content).To(ContainSubstring("chunk.jsonl#1"))

	// When it is pulled down
	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then the local note carries no sources
	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	raw := readFileString(t, filepath.Join(env.vault, files[0]))
	g.Expect(decodePulledCopy(t, []byte(raw)).Sources).To(BeEmpty())
	g.Expect(raw).NotTo(ContainSubstring("sources:"))
	g.Expect(raw).NotTo(ContainSubstring("chunk.jsonl"))
}

// Test792a21zd1h_Activate_APendingParentNoteIsNeverPulled verifies 792a21zd1h (A pending parent note is never
// pulled): a served parent whose note is pending has nothing pulled down when the user activates it, and a live
// parent note is pulled.
func Test792a21zd1h_Activate_APendingParentNoteIsNeverPulled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a parent whose note is pending, beside a live one
	parentVault := newServeVault(t)
	writeVaultNote(t, parentVault, "8.2026-01-08.pending-offer.md", pendingOfferFactNote("8", ""))
	writeVaultNote(t, parentVault, "9.2026-01-09.live.md", liveFactNote("9", ""))

	routes := cli.ServeRoutes(serveTestDeps(), parentVault, "team", "")
	env := ps2Env(t)
	identity := ps2Identity(ps2User)
	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		deps.Fetch = servedFetch(routes)
	}

	// When the user activates the pending note
	_, stderr := env.run("activate", "--note", "8.2026-01-08.pending-offer.md")

	// Then nothing is pulled down
	g.Expect(stderr).To(ContainSubstring("8.2026-01-08.pending-offer"))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.noteFiles()).To(BeEmpty())

	// and a live parent note is pulled
	_, liveStderr := env.run("activate", "--note", "9.2026-01-09.live.md")
	g.Expect(env.exitCodes()).To(BeEmpty(), liveStderr)

	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, files[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Vault).To(Equal(serverVaultID))
	g.Expect(copied.Parent.Links).To(HaveLen(1))

	if len(copied.Parent.Links) == 1 {
		g.Expect(copied.Parent.Links[0].Note).To(Equal("9.2026-01-09.live"))
	}
}

// Test792a21zd1i_Activate_ThePulledNoteIsEmbeddedAndDoesNotKeepTheParentsTags verifies 792a21zd1i (The pulled
// note is embedded and does not keep the parent's tags): a fetched parent note that carries vocabulary tags is
// pulled down with a sidecar, and the parent's tags are not on the local note.
func Test792a21zd1i_Activate_ThePulledNoteIsEmbeddedAndDoesNotKeepTheParentsTags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a fetched parent note that carries vocabulary tags
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.tagged", "c",
		"tags:\n    - vocab/parent-term\nvocab_version: v3\n"))
	g.Expect(env.parent.notes[0].content).To(ContainSubstring("vocab/parent-term"))

	// When it is pulled down
	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then its sidecar exists
	files := env.noteFiles()
	g.Expect(files).To(HaveLen(1))

	if len(files) != 1 {
		return
	}

	rawSidecar, readErr := os.ReadFile(filepath.Join(env.vault, strings.TrimSuffix(files[0], ".md")+".vec.json"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	sidecar, parseErr := embed.UnmarshalSidecar(rawSidecar)
	g.Expect(parseErr).NotTo(HaveOccurred())
	g.Expect(sidecar.Dims).To(Equal(ps4EmbedDims))
	g.Expect(sidecar.BodyVector).To(HaveLen(ps4EmbedDims))

	// and the parent's tags are not on the local note
	raw := readFileString(t, filepath.Join(env.vault, files[0]))
	g.Expect(decodePulledCopy(t, []byte(raw)).Tags).NotTo(ContainElement("vocab/parent-term"))
	g.Expect(raw).NotTo(ContainSubstring("vocab/parent-term"))
}

// Test792a21ze1d_Activate_ADeclineSurvivesAParentRename verifies 792a21ze1d (A decline survives a parent
// rename): a declined parent note that is renamed on the parent, with its old basename now in its aliases, is
// not written when it is activated again unchanged.
func Test792a21ze1d_Activate_ADeclineSurvivesAParentRename(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a declined parent note renamed on the parent, with its old basename now in its aliases
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.declined", "c", ""))
	hash := env.parent.hashOf(t, parentNote)

	env.run("activate", "--note", parentNote+".md")

	copies := env.linkedCopies(parentNote)
	g.Expect(copies).To(HaveLen(1))

	if len(copies) != 1 {
		return
	}

	_, discardErr := env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--discard")
	g.Expect(env.exitCodes()).To(BeEmpty(), discardErr)
	g.Expect(env.declined()).To(ConsistOf(declinedEntry{Vault: parentVaultID, Basename: parentNote, Hash: hash}))

	env.parent.rename(parentNote, "12.2026-09-25.declined-renamed")
	shownBefore := len(env.parent.shown())

	// When it is activated again unchanged
	env.run("activate", "--note", parentNote+".md")

	// Then nothing is written
	g.Expect(env.parent.shown()[shownBefore:]).To(Equal([]string{parentNote + ".md"}), "the parent was asked for it")
	g.Expect(env.noteFiles()).To(BeEmpty())
	g.Expect(env.exitCodes()).To(BeEmpty())
}

// Test792a21ze1e_Activate_AChangedParentNoteArrivesAsANewOffer verifies 792a21ze1e (A changed parent note
// arrives as a new offer): parent note P pulled down and accepted, and then changed on the parent, is pulled down
// again as a second local pending offer that carries the new exchange hash.
func Test792a21ze1e_Activate_AChangedParentNoteArrivesAsANewOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given P pulled down and accepted, and then changed on the parent
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.changing", "c", ""))
	accepted := acceptedPulledCopy(t, env, parentNote)
	oldHash := env.parent.hashOf(t, parentNote)

	env.parent.replace(pulledFact(parentNote, "changed upstream", ""))
	newHash := env.parent.hashOf(t, parentNote)
	g.Expect(newHash).NotTo(Equal(oldHash))

	// When P is activated again
	_, stderr := env.run("activate", "--note", parentNote+".md")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then a second local pending offer for P exists, carrying the new exchange hash
	offers := slices.DeleteFunc(env.linkedCopies(parentNote), func(name string) bool { return name == accepted })
	g.Expect(offers).To(HaveLen(1))

	if len(offers) != 1 {
		return
	}

	copied := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, offers[0]))))
	g.Expect(copied.Pending).To(BeTrue())
	g.Expect(copied.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: newHash}}))

	kept := decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, accepted))))
	g.Expect(kept.Pending).To(BeFalse(), "the accepted copy stays live")
	g.Expect(kept.Parent.Links).To(Equal([]pulledLink{{Note: parentNote, Via: "pulled", Hash: oldHash}}))
}

// Test792a21ze1f_Amend_AcceptingAPulledNoteSendsNothingUp verifies 792a21ze1f (Accepting a pulled note sends
// nothing up): when curation clears a pulled note's pending marker, no offer is queued for it and nothing is sent
// to the parent.
func Test792a21ze1f_Amend_AcceptingAPulledNoteSendsNothingUp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a pulled note
	env := ps2Env(t)
	parentNote := env.parent.addNote(pulledFact("7.2026-09-01.accepted", "c", ""))

	env.run("activate", "--note", parentNote+".md")

	copies := env.linkedCopies(parentNote)
	g.Expect(copies).To(HaveLen(1))

	if len(copies) != 1 {
		return
	}

	// When curation clears its pending marker
	_, stderr := env.run("amend", "--target", strings.TrimSuffix(copies[0], ".md"), "--clear-pending")
	g.Expect(env.exitCodes()).To(BeEmpty(), stderr)

	// Then no offer is queued for it
	g.Expect(decodePulledCopy(t, []byte(readFileString(t, filepath.Join(env.vault, copies[0])))).Pending).To(BeFalse())
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(env.parent.offers()).To(BeEmpty())
}

// unexported constants.
const (
	// ps4Conflict is the status of a refusal with a conflict.
	ps4Conflict = 409
	// ps4EmbedDims is the dimension of the env's embedder.
	ps4EmbedDims = 4
	// ps4FarFuture ends a backoff window the test never outlasts.
	ps4FarFuture = "2999-01-01T00:00:00Z"
	// ps4HalfID is the number of two-character pairs in a 32-character vault ID.
	ps4HalfID = 16
	// ps4NotFound is the status of a served note that is not there.
	ps4NotFound = 404
	// ps4OK is the status of an answered request.
	ps4OK = 200
	// ps4OtherVaultSeed seeds the ID of a vault that is neither the child nor its parent.
	ps4OtherVaultSeed = 70
	// ps4ReceiptBasename is the note the env's fake parent names in a receipt for the note ps4Anchored learns.
	ps4ReceiptBasename = "1100.2026-09-28." + ps4Slug
	// ps4Slug names the note learned for the attention scenarios.
	ps4Slug = "anchored"
)

// ps4Note is a learned note and its two forms: as it was learned, and with an anchored parent: that another key
// aliases, which a receipt cannot be recorded on.
type ps4Note struct {
	path     string
	plain    string
	anchored string
}

// ps4Anchored queues one note while the parent is down, as the commands leave it, and then anchors its parent: so
// that the receipt of its offer will be refused. The parent is back and its first backoff window is over.
func ps4Anchored(t *testing.T, env *wiringEnv) ps4Note {
	t.Helper()

	path := ps3Backlog(t, env, ps4Slug)[0]
	plain := readFileString(t, path)
	anchored := strings.Replace(plain, "\n---\n",
		"\nparent: &p\n    vault: "+strings.Repeat("ab", ps4HalfID)+"\nparent_copy: *p\n---\n", 1)

	if anchored == plain {
		t.Fatal("the learned note has no frontmatter to anchor")
	}

	env.plant(filepath.Base(path), []byte(anchored))

	return ps4Note{path: path, plain: plain, anchored: anchored}
}

// ps4Attention is ps4Anchored followed by the drain that moves the entry to the attention state.
func ps4Attention(t *testing.T, env *wiringEnv) ps4Note {
	t.Helper()

	note := ps4Anchored(t, env)
	env.run(ps3QueryArgs(env)...)

	box := ps3Outbox(t, env)
	if len(box.Entries) != 1 || box.Entries[0].State != "attention" {
		t.Fatalf("the entry is not in attention: %+v (stderr: %s)", box.Entries, env.lastStderr)
	}

	return note
}

// ps4Drain runs the drain of `engram update` (its exchange step, which makes no query of its own and ignores the
// backoff window) in a goroutine and waits for it, and for the script when one is given, with a time limit, as
// ps3Run does. The result's stdout is the notice of the outbox.
func ps4Drain(env *wiringEnv, script <-chan struct{}) ps3Result {
	type output struct{ notice, stderr string }

	done := make(chan output, 1)

	go func() {
		var stderr strings.Builder

		deps := newTestDeps(&strings.Builder{}, &stderr)
		env.customize(&deps)

		notice := cli.ExportUpdateExchange(deps)(context.Background(), env.vault, false)
		done <- output{notice: notice, stderr: stderr.String()}
	}()

	var result ps3Result

	select {
	case out := <-done:
		result.stdout, result.stderr, result.finished = out.notice, out.stderr, true
	case <-time.After(mockTimeout):
		return result
	}

	if script == nil {
		result.scripted = true

		return result
	}

	select {
	case <-script:
		result.scripted = true
	case <-time.After(mockTimeout):
	}

	return result
}

// ps4LoopRefusal is the answer of a parent that refuses an offer as a loop through the vault with the given ID.
func ps4LoopRefusal(vaultID string) *ps3Answer {
	body := `{"error":"serve: offer already passed through this vault (offer.path holds its vault ID)",` +
		`"reason":"loop","vault_id":"` + vaultID + `"}`

	return &ps3Answer{resp: cli.FetchResponse{Status: ps4Conflict, Body: []byte(body)}}
}

// ps4WriteCache records the parent's vault ID and the failure count in the vault's parent cache, with no backoff
// window, as a drain that reached the parent leaves it.
func ps4WriteCache(env *wiringEnv, vaultID string, failures int) {
	env.writeState("parent.json", fmt.Sprintf(`{"url":%q,"vault_id":%q,"failures":%d}`,
		env.parentURL, vaultID, failures))
}
