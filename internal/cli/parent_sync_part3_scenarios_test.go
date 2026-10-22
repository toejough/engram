package cli_test

// Scenario tests for the parent-sync product capability, part 3 (861c2zz): the outbox drains after the parent was
// reached, without the lock (792a21r), an unreachable parent is backed off (792a21s), a retried offer is idempotent
// (792a21t), a receipt links the local note to its counterpart (792a21u) and a reply that is not a valid receipt is
// not recorded (792a21v). Spec: docs/specs/product/parent-sync.md.
//
// Every scenario is a CLI journey over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation,
// #860) and a fake parent behind Deps.Fetch, with no network and no real home. The env's git identity is a scripted
// commander (ps2Identity), so no journey reads the machine's git configuration. The queue each drain works through
// is the one the commands themselves leave: notes learned while the parent was down. The scenarios that name a call
// or its absence (792a21r1a, r1c, r1k, s1a, t1a and v1c) drive Deps.Fetch through MockPvc1Fetch: a request the
// script does not expect holds the command, and the run reports it as an unfinished command.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a21r1a_Targets_Query_ABacklogIsSentInFirstQueuedOrderAndTheOutboxEmpties verifies 792a21r1a (Query drains
// a backlog): with three notes queued while the parent was down, and an outbox file that lists them in the opposite
// order, a query that succeeds against the parent sends the three offers in first-queued order and leaves the
// outbox empty.
func Test792a21r1a_Targets_Query_ABacklogIsSentInFirstQueuedOrderAndTheOutboxEmpties(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the outbox holds three entries, queued in the order first, second, third
	env := ps2Env(t)
	slugs := []string{"first", "second", "third"}
	paths := ps3Backlog(t, env, slugs...)

	// and the file lists them the other way round, so that only the first-queued rule orders the sends
	ps3ReverseOutbox(t, env)

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}
	script := ps3Script(env, calls, record, ps3Query(), ps3Offer(), ps3Offer(), ps3Offer())

	// When engram query succeeds against the parent
	run := ps3Run(env, script, ps3QueryArgs(env)...)

	// Then the three offers are sent in first-queued order and the outbox is empty afterwards
	g.Expect(run.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue(), "the parent did not get every request the script expects")
	g.Expect(ps3Slugs(record.offers(t))).To(Equal(slugs))
	g.Expect(env.outboxEntries()).To(BeEmpty())

	for _, path := range paths {
		g.Expect(readFileString(t, path)).To(ContainSubstring("via: offered"), "each receipt is recorded")
	}
}

// Test792a21r1b_Targets_Query_ATransportFailureStopsTheDrainAndRecordsTheAttempt verifies 792a21r1b (A transport
// failure stops the drain): of three queued notes, the second send fails with a connection error; the first
// entry is removed, the second and third remain, in first-queued order for the next drain, the second records its
// attempt and its last error, and the third is never tried.
func Test792a21r1b_Targets_Query_ATransportFailureStopsTheDrainAndRecordsTheAttempt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an outbox of three entries, listed in the file the other way round from the order they were queued
	// in, and a second send that fails with a connection error
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "first", "second", "third")
	xids := ps3XIDs(t, paths)

	ps3ReverseOutbox(t, env)

	var failing atomic.Bool

	failing.Store(true)

	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "second" && failing.Load() {
			return &ps3Answer{err: errPs3ConnectionRefused}
		}

		return nil
	}

	// When the outbox drains
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then the first entry is removed, the second and third remain, and the second records attempts and
	// last_error
	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(ConsistOf(xids[1:]))

	if len(box.Entries) != len(xids)-1 {
		return
	}

	attempts := make(map[string]int, len(box.Entries))
	lastErrors := make(map[string]string, len(box.Entries))

	for _, entry := range box.Entries {
		attempts[entry.XID] = entry.Attempts
		lastErrors[entry.XID] = entry.LastError
	}

	g.Expect(attempts[xids[1]]).To(Equal(1))
	g.Expect(lastErrors[xids[1]]).To(ContainSubstring(errPs3ConnectionRefused.Error()))
	g.Expect(attempts[xids[2]]).To(BeZero())
	g.Expect(hook.attempted()).To(Equal([]string{"first", "second"}), "the drain stops at the failure")
	g.Expect(stderr).To(ContainSubstring("engram: parent unreachable (retry after "))
	g.Expect(readFileString(t, paths[0])).To(ContainSubstring("via: offered"))
	g.Expect(readFileString(t, paths[1])).NotTo(ContainSubstring("via: offered"))

	// and the entries that remain are sent in first-queued order by the next drain
	failing.Store(false)
	env.advance(ps3Window)

	_, _ = env.run(ps3QueryArgs(env)...)

	g.Expect(hook.attempted()).To(Equal([]string{"first", "second", "second", "third"}))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21r1c_Targets_Query_ARejectedOfferIsNotSentAgainUntilItsNoteChanges verifies 792a21r1c (A rejected offer
// does not block the queue): the parent answers the first of three entries with a 4xx; that entry is marked
// rejected with the note's exchange hash, the next two are still sent, a later drain does not send the rejected
// entry again, and an amend that changes the note's exchange hash sends it again.
func Test792a21r1c_Targets_Query_ARejectedOfferIsNotSentAgainUntilItsNoteChanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the parent answers the first entry with a 4xx
	env := ps2Env(t)
	slugs := []string{"first", "second", "third"}
	paths := ps3Backlog(t, env, slugs...)
	xids := ps3XIDs(t, paths)

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	refuse := func(Pvc1FetchMockArgs) (cli.FetchResponse, error) {
		return cli.FetchResponse{Status: ps3BadRequest, Body: []byte(`{"error":"not accepted"}`)}, nil
	}
	record := &ps3Record{}
	script := ps3Script(env, calls, record, ps3Query(), ps3Offer(refuse), ps3Offer(), ps3Offer())

	// When the outbox drains
	run := ps3Run(env, script, ps3QueryArgs(env)...)

	// Then that entry is marked rejected, the next entries are still sent
	g.Expect(run.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue(), "the parent did not get every request the script expects")
	g.Expect(ps3Slugs(record.offers(t))).To(Equal(slugs))

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids[:1]))

	if len(box.Entries) != 1 {
		return
	}

	g.Expect(box.Entries[0].State).To(Equal("rejected"))
	g.Expect(box.Entries[0].RejectedHash).To(Equal(env.exchangeHashOf(filepath.Base(paths[0]))))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("not accepted"))

	// and the rejected entry is not sent again: a second drain makes no request but the query's own
	env.advance(ps3Window)

	again := ps3Run(env, ps3Script(env, calls, record, ps3Query()), ps3QueryArgs(env)...)
	g.Expect(again.finished).To(BeTrue(), "the drain sent the rejected entry again")
	g.Expect(again.scripted).To(BeTrue())
	g.Expect(entryXIDs(ps3Outbox(t, env))).To(Equal(xids[:1]), "the rejected entry stays")

	// until its note's exchange hash changes
	env.advance(ps3Window)

	amend := ps3Run(env, ps3Script(env, calls, record, ps3Offer()),
		"amend", "--target", noteBasename(paths[0]), "--object", "changed")
	g.Expect(amend.finished).To(BeTrue(), "the amend made a request the script does not expect")
	g.Expect(amend.scripted).To(BeTrue())

	offers := record.offers(t)
	g.Expect(ps3Slugs(offers)).To(Equal(append(slices.Clone(slugs), "first")))
	g.Expect(offers[len(offers)-1].Object).To(Equal("changed"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21r1d_Targets_Query_AnEntryQueuedMeanwhileSurvivesTheMerge verifies 792a21r1d (A concurrent enqueue
// survives the drain): an entry that another process queues while the drain is sending is still in the outbox
// when the drain has merged its results.
func Test792a21r1d_Targets_Query_AnEntryQueuedMeanwhileSurvivesTheMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a queued note whose entry is not yet in the outbox when the drain starts
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "sending", "meanwhile")
	xids := ps3XIDs(t, paths)
	ps3DropEntry(t, env, xids[1])

	// and another process queues it while the first offer is being sent
	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "sending" {
			ps3Enqueue(t, env, xids[1])
		}

		return nil
	}

	// When the drain merges its results
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then that entry is still in the outbox
	g.Expect(hook.attempted()).To(Equal([]string{"sending"}))
	g.Expect(stderr).NotTo(ContainSubstring("error"))

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids[1:]))

	if len(box.Entries) != 1 {
		return
	}

	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(readFileString(t, paths[0])).To(ContainSubstring("via: offered"), "the sent offer's receipt is recorded")
}

// Test792a21r1e_Targets_Query_ANoteAmendedInFlightStaysQueued verifies 792a21r1e (A note amended in flight stays
// queued): a note amended while its offer is being sent takes the receipt of the older content, keeps its entry
// for the newer content, and has that content offered by the next drain.
func Test792a21r1e_Targets_Query_ANoteAmendedInFlightStaysQueued(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note amended while its offer is being sent
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "amended")
	xids := ps3XIDs(t, paths)

	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "amended" && len(hook.attempted()) == 1 {
			env.mustWrite(paths[0], strings.Replace(readFileString(t, paths[0]), "object: c", "object: newer", 1))
		}

		return nil
	}

	// When the receipt is recorded
	env.run(ps3QueryArgs(env)...)

	// Then the note keeps its entry for the newer content
	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids))

	if len(box.Entries) != 1 {
		return
	}

	g.Expect(box.Entries[0].State).To(Equal("queued"))

	content := readFileString(t, paths[0])
	g.Expect(content).To(ContainSubstring("object: newer"), "the newer content is kept")
	g.Expect(content).To(ContainSubstring("via: offered"), "the receipt is recorded")

	// and the next drain offers the newer content
	env.advance(ps3Window)
	env.run(ps3QueryArgs(env)...)

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(2))

	if len(offers) != 2 {
		return
	}

	g.Expect(offers[0].Object).To(Equal("c"))
	g.Expect(offers[1].Object).To(Equal("newer"))
	g.Expect(offers[1].Offer.For).To(Equal("1100.2026-09-28.amended"), "an amend of the note the parent holds")
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21r1f_Targets_Query_ANoteDeletedInFlightDropsItsReceipt verifies 792a21r1f (A note deleted in flight
// drops its receipt): a note deleted while its offer is being sent has its receipt discarded and its entry
// dropped, and the next entry is still sent.
func Test792a21r1f_Targets_Query_ANoteDeletedInFlightDropsItsReceipt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a note deleted while its offer is being sent
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "deleted", "kept")

	hook.offer = func(args cli.LearnArgs) *ps3Answer {
		if args.Slug == "deleted" {
			for _, doomed := range []string{paths[0], strings.TrimSuffix(paths[0], ".md") + ".vec.json"} {
				removeErr := os.Remove(doomed)
				if removeErr != nil {
					t.Error(removeErr)
				}
			}
		}

		return nil
	}

	// When the drain merges
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then the receipt is discarded and the entry is dropped
	g.Expect(hook.attempted()).To(Equal([]string{"deleted", "kept"}))
	g.Expect(env.noteFiles()).To(Equal([]string{filepath.Base(paths[1])}), "the deleted note is not recreated")
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(stderr).NotTo(ContainSubstring("error"))
	g.Expect(readFileString(t, paths[1])).To(ContainSubstring("via: offered"), "the next entry took its receipt")
}

// Test792a21r1g_Targets_Query_AnUnbuildablePayloadDoesNotStopTheDrain verifies 792a21r1g (An unbuildable payload
// does not stop the drain): of three queued notes the second was renamed by hand to a name with no slug, so its
// offer cannot be built; the error is recorded on its entry, the entry stays queued, the first and third are
// still sent, and the parent is not backed off for it.
func Test792a21r1g_Targets_Query_AnUnbuildablePayloadDoesNotStopTheDrain(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given an entry whose offer cannot be built locally
	env := ps2Env(t)
	hook := &ps3Hook{}
	ps3Install(env, hook)

	paths := ps3Backlog(t, env, "first", "second", "third")
	xids := ps3XIDs(t, paths)

	slugless := filepath.Join(env.vault, "5.2026-09-28.md")
	g.Expect(os.Rename(paths[1], slugless)).To(Succeed())

	// When the outbox drains
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then the error is recorded on the entry, the entry stays queued, the later entries are still sent, and the
	// parent is not backed off for it
	g.Expect(hook.attempted()).To(Equal([]string{"first", "third"}))

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids[1:2]))

	if len(box.Entries) != 1 {
		return
	}

	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(box.Entries[0].Attempts).To(Equal(1))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring("building the offer for 5.2026-09-28"))

	cache := ps3Cache(t, env)
	g.Expect(cache.Failures).To(BeZero())
	g.Expect(cache.BackoffUntil.IsZero()).To(BeTrue())
	g.Expect(stderr).NotTo(ContainSubstring("parent unreachable"))
	g.Expect(readFileString(t, paths[0])).To(ContainSubstring("via: offered"))
	g.Expect(readFileString(t, paths[2])).To(ContainSubstring("via: offered"))
}

// Test792a21r1h_Targets_Query_AnEmptyOutboxTouchesNothing verifies 792a21r1h (An empty outbox touches nothing):
// with nothing queued, a command that drains sends no offer, makes no request but its own, and writes nothing to
// the vault, whether the outbox was emptied by a send or never held anything and the parent cannot report its
// vault ID.
func Test792a21r1h_Targets_Query_AnEmptyOutboxTouchesNothing(t *testing.T) {
	t.Parallel()

	cases := map[string]func(env *wiringEnv){
		"an outbox emptied by a send": func(env *wiringEnv) {
			env.learnFact("sent")
			env.advance(ps3Window)
		},
		"an outbox that never held an entry, under a parent too old to report its vault ID": func(env *wiringEnv) {
			env.parent.queryNoID = true
			env.run(ps3QueryArgs(env)...)
			env.advance(ps3Window)
		},
	}

	for name, prepare := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given an empty outbox
			env := ps2Env(t)
			prepare(env)
			g.Expect(ps3Outbox(t, env).Entries).To(BeEmpty())

			before := ps3Stamps(t, env.vault)
			requestsBefore := len(env.parent.requests())
			offersBefore := len(env.parent.offers())

			// When a command that drains runs
			_, stderr := env.run(ps3QueryArgs(env)...)

			// Then no send and no write happens
			requests := env.parent.requests()
			g.Expect(requests).To(HaveLen(requestsBefore+1), "the query's own request is the only one")

			if len(requests) != requestsBefore+1 {
				return
			}

			g.Expect(requests[requestsBefore].method).To(Equal("GET"))
			g.Expect(requests[requestsBefore].url).To(ContainSubstring("/query"))
			g.Expect(env.parent.offers()).To(HaveLen(offersBefore), "no offer is sent")
			g.Expect(ps3Stamps(t, env.vault)).To(Equal(before), "nothing in the vault is written")
			g.Expect(stderr).To(BeEmpty())
		})
	}
}

// Test792a21r1i_Targets_Query_SendsHoldNoLock verifies 792a21r1i (Sends hold no lock): while a drain sends its
// three offers, the vault lock is not held.
func Test792a21r1i_Targets_Query_SendsHoldNoLock(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a drain in progress, over three queued notes
	env := ps2Env(t)
	ps3Backlog(t, env, "first", "second", "third")

	probe := &lockProbe{}
	identity := ps2Identity(ps2User)
	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		probe.wrap(deps)
	}

	// When an offer is being sent
	env.run(ps3QueryArgs(env)...)

	// Then the vault lock is not held
	offerFetches := slices.DeleteFunc(probe.fetched(), func(target string) bool {
		return !strings.HasSuffix(target, "/learn")
	})
	g.Expect(offerFetches).To(HaveLen(3), "the three offers were sent")
	g.Expect(probe.fetchesUnderLock()).To(BeEmpty(), "no request is made with the lock held")
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21r1k_Targets_Query_AFailedLocationCheckSendsNothing verifies 792a21r1k (A vault that fails its
// location check sends nothing): in a vault whose location record is missing, a command that drains makes no
// request but its own, sends no offer, prints the location warning, and leaves the three entries queued.
func Test792a21r1k_Targets_Query_AFailedLocationCheckSendsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a vault whose location record is missing, with three entries queued
	env := ps2Env(t)
	ps3Backlog(t, env, "first", "second", "third")
	g.Expect(os.Remove(filepath.Join(env.vault, ".engram", "home.json"))).To(Succeed())

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}
	script := ps3Script(env, calls, record, ps3Query())

	// When a command drains
	run := ps3Run(env, script, ps3QueryArgs(env)...)

	// Then nothing is sent and the location warning is printed
	g.Expect(run.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(run.scripted).To(BeTrue())
	g.Expect(record.offers(t)).To(BeEmpty())
	g.Expect(run.stderr).To(ContainSubstring("vault-id --claim"), "the location warning is printed")

	box := ps3Outbox(t, env)
	g.Expect(box.Entries).To(HaveLen(3), "the entries stay queued")

	for _, entry := range box.Entries {
		g.Expect(entry.State).To(Equal("queued"))
		g.Expect(entry.Attempts).To(BeZero())
	}
}

// Test792a21s1a_Targets_Query_ACommandInsideTheBackoffWindowMakesNoRequest verifies 792a21s1a (Commands inside the
// backoff window make no request): after a query that failed to reach the parent, a second query 10 seconds later
// makes no parent request, returns the local results, and prints one warning.
func Test792a21s1a_Targets_Query_ACommandInsideTheBackoffWindowMakesNoRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := ps2Env(t)
	plantRealVaultNote(t, env.vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}

	// Given a query that failed to reach the parent
	refused := func(Pvc1FetchMockArgs) (cli.FetchResponse, error) {
		return cli.FetchResponse{}, errPs3ConnectionRefused
	}
	first := ps3Run(env, ps3Script(env, calls, record, ps3Query(refused)), ps3QueryArgs(env)...)
	g.Expect(first.finished).To(BeTrue())
	g.Expect(first.scripted).To(BeTrue())

	// When a second query runs 10 seconds later
	env.advance(ps3InsideWindow)

	second := ps3Run(env, nil, ps3QueryArgs(env)...)

	// Then it makes no parent request, returns local results, and prints one warning
	g.Expect(second.finished).To(BeTrue(), "the second query made a parent request")
	g.Expect(record.requests()).To(HaveLen(1), "only the first query reached for the parent")

	var parsed queryParsed

	g.Expect(yaml.Unmarshal([]byte(second.stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) == 1 {
		g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
	}

	lines := nonEmptyLines(second.stderr)
	g.Expect(lines).To(HaveLen(1))

	if len(lines) == 1 {
		g.Expect(lines[0]).To(HavePrefix("engram: parent unreachable (retry after "))
	}
}

// Test792a21s1c_Targets_Query_TheDelayGrowsAndIsCapped verifies 792a21s1c (The delay grows and is capped): across
// eight consecutive failures against the parent, each recorded retry time is 30 seconds after the first failure,
// twice the previous delay after each later one, and never more than 15 minutes.
func Test792a21s1c_Targets_Query_TheDelayGrowsAndIsCapped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given consecutive failures against the parent
	env := ps2Env(t)
	env.parent.setDown(true)

	delays := []time.Duration{
		30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute,
		ps3Cap, ps3Cap, ps3Cap,
	}

	for index, want := range delays {
		// When each retry time is recorded
		env.run(ps3QueryArgs(env)...)

		// Then the delay is 30 seconds after the first failure, doubles with each, and never exceeds 15 minutes
		cache := ps3Cache(t, env)
		g.Expect(cache.Failures).To(Equal(index+1), "failure %d", index+1)
		g.Expect(cache.BackoffUntil.Sub(env.now())).To(Equal(want), "the delay after failure %d", index+1)

		env.advance(want + time.Second)
	}
}

// Test792a21s1d_Targets_Query_ASuccessResetsTheCount verifies 792a21s1d (A success resets the count): after two
// failures, a query that reaches the parent leaves the failure count at zero with no window, and the next failure
// backs off for 30 seconds.
func Test792a21s1d_Targets_Query_ASuccessResetsTheCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given a parent that has failed twice
	env := ps2Env(t)
	env.parent.setDown(true)
	env.run(ps3QueryArgs(env)...)
	env.advance(ps3Window)
	env.run(ps3QueryArgs(env)...)
	g.Expect(ps3Cache(t, env).Failures).To(Equal(2))

	// When a contact succeeds
	env.parent.setDown(false)
	env.advance(2 * time.Minute)
	env.run(ps3QueryArgs(env)...)

	// Then the failure count is zero and the next failure backs off for 30 seconds
	cache := ps3Cache(t, env)
	g.Expect(cache.Failures).To(BeZero())
	g.Expect(cache.BackoffUntil.IsZero()).To(BeTrue())

	env.parent.setDown(true)
	env.run(ps3QueryArgs(env)...)

	cache = ps3Cache(t, env)
	g.Expect(cache.Failures).To(Equal(1))
	g.Expect(cache.BackoffUntil.Sub(env.now())).To(Equal(30 * time.Second))
}

// Test792a21s1e_Targets_Query_ADifferentParentURLIsNotBackedOff verifies 792a21s1e (A different parent URL is not
// backed off): a backoff recorded for one parent URL skips that URL's next query, but a query against a different
// parent URL inside the same window is not skipped.
func Test792a21s1e_Targets_Query_ADifferentParentURLIsNotBackedOff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const otherParent = "http://other-parent:8093"

	// Given a backoff recorded for one parent URL
	env := ps2Env(t)
	env.parent.setDown(true)
	env.run(ps3QueryArgs(env)...)
	env.parent.setDown(false)
	env.advance(ps3InsideWindow)

	env.run(ps3QueryArgs(env)...)
	g.Expect(env.parent.requests()).To(HaveLen(1), "inside the window the recorded URL is skipped")
	g.Expect(env.lastStderr).To(ContainSubstring("engram: parent unreachable (retry after "))

	// When a command contacts a different parent URL
	env.parentURL = otherParent
	_, stderr := env.run(ps3QueryArgs(env)...)

	// Then it is not skipped
	requests := env.parent.requests()
	g.Expect(requests).To(HaveLen(2))

	if len(requests) == 2 {
		g.Expect(requests[1].url).To(HavePrefix(otherParent))
		g.Expect(requests[1].failed).To(BeFalse())
	}

	g.Expect(stderr).NotTo(ContainSubstring("parent unreachable"))
}

// Test792a21s1f_Targets_Query_TheWindowEnds verifies 792a21s1f (The window ends): a query before the retry time
// is skipped, and a query at the retry time or after it contacts the parent.
func Test792a21s1f_Targets_Query_TheWindowEnds(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wait      time.Duration
		contacted bool
	}{
		"a second before the retry time": {wait: 29 * time.Second, contacted: false},
		"at the retry time":              {wait: 30 * time.Second, contacted: true},
		"a second after the retry time":  {wait: 31 * time.Second, contacted: true},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Given a backoff whose retry time is 30 seconds after the failure
			env := ps2Env(t)
			env.parent.setDown(true)
			env.run(ps3QueryArgs(env)...)
			env.parent.setDown(false)

			// When a command contacts the parent
			env.advance(testCase.wait)
			_, stderr := env.run(ps3QueryArgs(env)...)

			// Then the contact is allowed once the retry time has passed
			requests := env.parent.requests()

			if !testCase.contacted {
				g.Expect(requests).To(HaveLen(1), "the window has not ended")
				g.Expect(stderr).To(ContainSubstring("engram: parent unreachable (retry after "))

				return
			}

			g.Expect(requests).To(HaveLen(2), "the contact is allowed")
			g.Expect(stderr).NotTo(ContainSubstring("parent unreachable"))
			g.Expect(ps3Cache(t, env).Failures).To(BeZero())
		})
	}
}

// Test792a21t1a_Targets_Query_ARetriedOfferFindsTheNoteTheParentAlreadyHolds verifies 792a21t1a (Lost response,
// then retry): the served parent stores an offer but its response is lost; the entry is sent again unchanged, and
// the parent answers with the receipt of the pending note it already holds, which stays the one pending note for
// that offer.
func Test792a21t1a_Targets_Query_ARetriedOfferFindsTheNoteTheParentAlreadyHolds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given the parent stored an offer but the response was lost
	parentVault := newServeVault(t)
	served := servedFetch(cli.ServeRoutes(serveTestDeps(), parentVault, "team", ""))
	serve := func(args Pvc1FetchMockArgs) (cli.FetchResponse, error) {
		return served(args.Ctx, args.Method, args.Url, args.Body)
	}
	lost := func(args Pvc1FetchMockArgs) (cli.FetchResponse, error) {
		_, serveErr := serve(args)
		if serveErr != nil {
			return cli.FetchResponse{}, serveErr
		}

		return cli.FetchResponse{}, errPs3ResponseLost
	}

	env := ps2Env(t)
	paths := ps3Backlog(t, env, "retried")

	fetch, calls := MockPvc1Fetch(t)
	ps3UseFetch(env, fetch)

	record := &ps3Record{}
	first := ps3Run(env, ps3Script(env, calls, record, ps3Query(serve), ps3Offer(lost)), ps3QueryArgs(env)...)
	g.Expect(first.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(first.scripted).To(BeTrue())
	g.Expect(noteFiles(t, parentVault)).To(HaveLen(1), "the parent holds the offered note")
	g.Expect(ps3Outbox(t, env).Entries).To(HaveLen(1), "the child never saw the receipt")

	// When the entry is sent again unchanged
	env.advance(ps3Window)

	second := ps3Run(env, ps3Script(env, calls, record, ps3Query(serve), ps3Offer(serve)), ps3QueryArgs(env)...)
	g.Expect(second.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(second.scripted).To(BeTrue())

	// Then the parent returns the receipt of the pending note it already has, and holds one pending note for that
	// offer
	offers := record.offers(t)
	g.Expect(offers).To(HaveLen(2))

	if len(offers) != 2 {
		return
	}

	g.Expect(offers[0].Offer.Key).NotTo(BeEmpty())
	g.Expect(offers[1].Offer).To(Equal(offers[0].Offer), "the retry carries the same origin, key and path")

	held := noteFiles(t, parentVault)
	g.Expect(held).To(HaveLen(1), "one pending note for that offer")

	if len(held) != 1 {
		return
	}

	g.Expect(readFileString(t, held[0])).To(ContainSubstring("pending: true"))

	linked := decodePulledCopy(t, []byte(readFileString(t, paths[0])))
	g.Expect(linked.Parent.Links).To(HaveLen(1))

	if len(linked.Parent.Links) == 1 {
		g.Expect(linked.Parent.Links[0].Note).To(Equal(strings.TrimSuffix(filepath.Base(held[0]), ".md")),
			"the receipt is the one of the note the parent already has")
	}

	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21u1a_Targets_Query_AReceiptLinksTheNote verifies 792a21u1a (Receipt links the note): the parent answers
// a learn-offer for the local note with basename 1100.2026-09-27.x and a stored hash of its own; the note's
// primary link becomes that basename, offered, with the stored hash, under the parent's vault ID, and the note's
// sidecar is untouched.
func Test792a21u1a_Targets_Query_AReceiptLinksTheNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		counterpart = "1100.2026-09-27.x"
		storedHash  = "xh1:parentstored"
	)

	// Given the parent answers a learn-offer for local note L with basename 1100.2026-09-27.x
	env := ps2Env(t)
	paths := ps3Backlog(t, env, "linked")
	sidecarPath := strings.TrimSuffix(paths[0], ".md") + ".vec.json"
	sidecarBefore := readFileString(t, sidecarPath)
	localHash := env.exchangeHashOf(filepath.Base(paths[0]))
	env.parent.setReceipt(ps3Receipt(t, cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "1100", Basename: counterpart, Pending: true,
		VaultID: parentVaultID, StoredHash: storedHash,
	}))

	// When the receipt is recorded
	env.run(ps3QueryArgs(env)...)

	// Then L's primary link is {note: 1100.2026-09-27.x, via: offered, hash: <stored_hash>} under the parent's vault
	// ID, and L's sidecar vector is unchanged
	linked := decodePulledCopy(t, []byte(readFileString(t, paths[0])))
	g.Expect(linked.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(linked.Parent.Links).To(Equal([]pulledLink{{Note: counterpart, Via: "offered", Hash: storedHash}}))
	g.Expect(storedHash).NotTo(Equal(localHash), "the hash is the parent's, not the one the child computed")

	g.Expect(readFileString(t, sidecarPath)).To(Equal(sidecarBefore), "the sidecar is not rewritten")

	before, beforeErr := embed.UnmarshalSidecar([]byte(sidecarBefore))
	g.Expect(beforeErr).NotTo(HaveOccurred())

	after, afterErr := embed.UnmarshalSidecar([]byte(readFileString(t, sidecarPath)))
	g.Expect(afterErr).NotTo(HaveOccurred())
	g.Expect(after.SituationVector).To(Equal(before.SituationVector))
	g.Expect(after.BodyVector).To(Equal(before.BodyVector))
}

// Test792a21u1b_Targets_Amend_AReceiptRelinksToTheResolvedTarget verifies 792a21u1b (Re-link to the resolved
// target): an amend-offer for a linked note is answered with a receipt that names a resolved target E other than
// the note's current primary; the note's primary link becomes E, offered, with the receipt's stored hash.
func Test792a21u1b_Targets_Amend_AReceiptRelinksToTheResolvedTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		resolved   = "1100.2026-09-20.resolved-target"
		storedHash = "xh1:resolvedstored"
	)

	env := ps2Env(t)
	path := env.learnFact("relinked")
	current := decodePulledCopy(t, []byte(readFileString(t, path)))
	g.Expect(current.Parent.Links).To(HaveLen(1))

	if len(current.Parent.Links) != 1 {
		return
	}

	// Given an amend-offer's receipt names a resolved target E different from L's current primary
	g.Expect(current.Parent.Links[0].Note).NotTo(Equal(resolved))
	env.parent.setReceipt(ps3Receipt(t, cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "1200", Basename: "1200.2026-09-28.pending-copy", Pending: true,
		VaultID: parentVaultID, StoredHash: storedHash, For: resolved,
	}))

	// When the receipt is recorded
	env.run("amend", "--target", noteBasename(path), "--object", "changed")

	// Then L's primary link becomes E
	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(2))

	if len(offers) == 2 {
		g.Expect(offers[1].Offer.For).To(Equal(current.Parent.Links[0].Note), "the offer named the old primary")
	}

	relinked := decodePulledCopy(t, []byte(readFileString(t, path)))
	g.Expect(relinked.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(relinked.Parent.Links).NotTo(BeEmpty())

	if len(relinked.Parent.Links) > 0 {
		g.Expect(relinked.Parent.Links[0]).To(Equal(pulledLink{Note: resolved, Via: "offered", Hash: storedHash}))
	}

	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792a21u1c_Targets_Query_AReceiptKeepsTheNotesAuthor verifies 792a21u1c (A receipt keeps the note's author):
// a note whose user: is alice, drained by a command run as bob, takes the receipt with its repo:, user: and
// vault: lines unchanged.
func Test792a21u1c_Targets_Query_AReceiptKeepsTheNotesAuthor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		alice = "alice@example.com"
		bob   = "bob@example.com"
	)

	asUser := func(email, repo string) func(*cli.Deps) {
		return func(deps *cli.Deps) {
			deps.Commander = fakeGitCommander{
				remoteOut: repo + "\n", configOut: email + "\n", revParseErr: errFakeGitFailed,
			}
			deps.Username = func() (string, error) { return "", errFakeGitFailed }
		}
	}

	// Given a note whose user: is alice
	env := newWiringEnv(t)
	env.wrap = asUser(alice, "github.com/alice/repo")
	path := ps3Backlog(t, env, "authored")[0]
	authorBefore := ps3AuthorLines(readFileString(t, path))
	g.Expect(authorBefore).To(ContainElement("user: " + alice))
	g.Expect(authorBefore).To(ContainElement("repo: github.com/alice/repo"))

	// and a command run as bob
	env.wrap = asUser(bob, "github.com/bob/repo")

	// When the receipt is recorded on the note
	env.run(ps3QueryArgs(env)...)

	// Then its repo:, user: and vault: are unchanged
	content := readFileString(t, path)
	g.Expect(content).To(ContainSubstring("via: offered"), "the receipt is recorded")
	g.Expect(ps3AuthorLines(content)).To(Equal(authorBefore))
	g.Expect(content).NotTo(ContainSubstring("bob"))
}

// Test792a21u1d_Targets_Amend_AReceiptUnderAnotherParentStartsOver verifies 792a21u1d (A receipt under another
// parent starts over): a note linked under one parent vault, amended and offered to a parent with another vault
// ID, ends with the links of the answering parent only.
func Test792a21u1d_Targets_Amend_AReceiptUnderAnotherParentStartsOver(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		freshNote   = "2100.2026-09-28.fresh"
		freshHash   = "xh1:freshstored"
		otherParent = "http://new-parent:9"
	)

	otherVault := seqID(72)

	// Given a note linked under a parent vault other than the one that answers, with a covered link besides its
	// offered one
	env := ps2Env(t)
	path := env.learnFact("moved")
	offeredLink := "          hash: xh1:stored\n"
	coveredLink := "        - note: 9.2026-09-20.old-covered\n          via: covered\n          hash: xh1:covered\n"
	env.mustWrite(path, strings.Replace(readFileString(t, path), offeredLink, offeredLink+coveredLink, 1))

	before := decodePulledCopy(t, []byte(readFileString(t, path)))
	g.Expect(before.Parent.Vault).To(Equal(parentVaultID))
	g.Expect(before.Parent.Links).To(HaveLen(2))

	env.parentURL = otherParent
	env.parent = &recordingParent{vaultID: otherVault}
	env.parent.setReceipt(ps3Receipt(t, cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "2100", Basename: freshNote, Pending: true,
		VaultID: otherVault, StoredHash: freshHash,
	}))

	// When the receipt is recorded
	env.run("amend", "--target", noteBasename(path), "--object", "changed")

	// Then the note's links are those of the answering parent only
	g.Expect(env.parent.offers()).To(HaveLen(1))

	after := decodePulledCopy(t, []byte(readFileString(t, path)))
	g.Expect(after.Parent.Vault).To(Equal(otherVault))
	g.Expect(after.Parent.Links).To(Equal([]pulledLink{{Note: freshNote, Via: "offered", Hash: freshHash}}))
	g.Expect(readFileString(t, path)).NotTo(ContainSubstring("old-covered"), "the old parent's covered link is gone")
	g.Expect(readFileString(t, path)).NotTo(ContainSubstring(before.Parent.Links[0].Note))
}

// Test792a21v1c_Targets_Query_AFailedWriteKeepsTheEntry verifies 792a21v1c (A failed write keeps the entry): when
// recording a valid receipt on the note fails, the entry stays queued with the error, and the next drain sends the
// same offer again, with the same key, and then records the receipt.
func Test792a21v1c_Targets_Query_AFailedWriteKeepsTheEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Given recording a valid receipt on the note fails
	env := ps2Env(t)
	paths := ps3Backlog(t, env, "unwritable")
	xids := ps3XIDs(t, paths)

	failing := &atomic.Bool{}
	failing.Store(true)

	fetch, calls := MockPvc1Fetch(t)
	identity := ps2Identity(ps2User)
	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		deps.Fetch = fetch
		deps.FS = ps3NoteWriteFS{EdgeFS: deps.FS, failing: failing}
	}

	record := &ps3Record{}

	// When the drain reports it
	first := ps3Run(env, ps3Script(env, calls, record, ps3Query(), ps3Offer()), ps3QueryArgs(env)...)

	// Then the entry stays queued with the error
	g.Expect(first.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(first.scripted).To(BeTrue())
	g.Expect(first.stderr).To(ContainSubstring("recording the receipt on"))

	box := ps3Outbox(t, env)
	g.Expect(entryXIDs(box)).To(Equal(xids))

	if len(box.Entries) != 1 {
		return
	}

	g.Expect(box.Entries[0].State).To(Equal("queued"))
	g.Expect(box.Entries[0].Attempts).To(Equal(1))
	g.Expect(box.Entries[0].LastError).To(ContainSubstring(errPs3NoteWrite.Error()))
	g.Expect(readFileString(t, paths[0])).NotTo(ContainSubstring("via: offered"), "nothing is recorded")

	// and it is sent again later, idempotently
	failing.Store(false)
	env.advance(ps3Window)

	second := ps3Run(env, ps3Script(env, calls, record, ps3Query(), ps3Offer()), ps3QueryArgs(env)...)
	g.Expect(second.finished).To(BeTrue(), "the query made a request the script does not expect")
	g.Expect(second.scripted).To(BeTrue())

	offers := record.offers(t)
	g.Expect(offers).To(HaveLen(2))

	if len(offers) == 2 {
		g.Expect(offers[0].Offer.Key).NotTo(BeEmpty())
		g.Expect(offers[1].Offer.Key).To(Equal(offers[0].Offer.Key), "the same offer key is sent again")
	}

	g.Expect(readFileString(t, paths[0])).To(ContainSubstring("via: offered"), "the receipt is now recorded")
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// unexported constants.
const (
	// ps3BadRequest is the 4xx the parent answers a rejected offer with.
	ps3BadRequest = 400
	// ps3Cap is the longest backoff window (792a21s: 15 minutes).
	ps3Cap = 15 * time.Minute
	// ps3Gap spaces the queued notes apart, inside the backoff window the first failed contact starts.
	ps3Gap = 10 * time.Second
	// ps3InsideWindow is a wait that stays inside the first 30 second window.
	ps3InsideWindow = 10 * time.Second
	// ps3Window is a wait that outlasts the first backoff window.
	ps3Window = time.Minute
)

// unexported variables.
var (
	errPs3ConnectionRefused = errors.New("dial tcp: connection refused")
	errPs3NoteWrite         = errors.New("ps3: the note write is refused")
	errPs3ResponseLost      = errors.New("read tcp: connection reset by peer")
)

// ps3Answer is a canned answer to one offer.
type ps3Answer struct {
	resp cli.FetchResponse
	err  error
}

// ps3AnswerFunc answers one request of a ps3Script.
type ps3AnswerFunc func(args Pvc1FetchMockArgs) (cli.FetchResponse, error)

// ps3Hook sees every offer a command sends, and may answer it instead of the env's fake parent or act while it is
// in flight.
type ps3Hook struct {
	mu    sync.Mutex
	offer func(args cli.LearnArgs) *ps3Answer
	slugs []string
}

// attempted lists the slug of every offer sent, in order.
func (h *ps3Hook) attempted() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.slugs)
}

// ps3NoteWriteFS refuses every atomic write of a note while failing is set.
type ps3NoteWriteFS struct {
	cli.EdgeFS

	failing *atomic.Bool
}

func (f ps3NoteWriteFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	if f.failing.Load() && strings.HasSuffix(path, ".md") {
		return errPs3NoteWrite
	}

	return f.EdgeFS.WriteFileAtomic(path, data, perm)
}

// ps3ParentCache is the part of .engram/parent.json the backoff scenarios read.
type ps3ParentCache struct {
	BackoffUntil time.Time
	Failures     int
}

// ps3Record is the requests a ps3Script saw, in order.
type ps3Record struct {
	mu   sync.Mutex
	seen []ps3Request
}

// add records one request.
func (r *ps3Record) add(method, url string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seen = append(r.seen, ps3Request{method: method, url: url, body: slices.Clone(body)})
}

// offers decodes the body of every POST /learn seen.
func (r *ps3Record) offers(t *testing.T) []cli.LearnArgs {
	t.Helper()

	offers := make([]cli.LearnArgs, 0)

	for _, request := range r.requests() {
		if request.method != "POST" || !strings.HasSuffix(request.url, "/learn") {
			continue
		}

		var args cli.LearnArgs

		decodeErr := json.Unmarshal(request.body, &args)
		if decodeErr != nil {
			t.Fatalf("an offer body does not decode: %v", decodeErr)
		}

		offers = append(offers, args)
	}

	return offers
}

// requests lists every request seen.
func (r *ps3Record) requests() []ps3Request {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.seen)
}

// ps3Request is one request to the parent.
type ps3Request struct {
	method string
	url    string
	body   []byte
}

// ps3Result is what one command did.
type ps3Result struct {
	stdout   string
	stderr   string
	finished bool
	scripted bool
}

// ps3Step is one request a ps3Script expects from the command, in order. A nil answer leaves it to the env's fake
// parent.
type ps3Step struct {
	method string
	route  string
	answer ps3AnswerFunc
}

// ps3AuthorLines lists the repo:, user: and vault: lines of a note's frontmatter.
func ps3AuthorLines(content string) []string {
	frontmatter, _, found := embed.SplitFrontmatter([]byte(content))
	if !found {
		return nil
	}

	return regexp.MustCompile(`(?m)^(repo|user|vault): .*$`).FindAllString(string(frontmatter), -1)
}

// ps3Backlog queues one note per slug the way the commands do: each is learned while the parent is down, a little
// later than the one before. Afterwards the parent is back, and the first backoff window is over.
func ps3Backlog(t *testing.T, env *wiringEnv, slugs ...string) []string {
	t.Helper()

	env.parent.setDown(true)

	paths := make([]string, 0, len(slugs))

	for _, slug := range slugs {
		paths = append(paths, env.learnFact(slug))
		env.advance(ps3Gap)
	}

	env.parent.setDown(false)
	env.advance(ps3Window)

	if queued := len(env.outboxEntries()); queued != len(slugs) {
		t.Fatalf("the backlog holds %d entries, want %d (stderr: %s)", queued, len(slugs), env.lastStderr)
	}

	return paths
}

// ps3Cache is the vault's parent cache.
func ps3Cache(t *testing.T, env *wiringEnv) ps3ParentCache {
	t.Helper()

	cache, err := cli.ExportLoadParentCache(cli.ExportExchangeStateFromDeps(env.deps()), env.vault)
	if err != nil {
		t.Fatalf("the parent cache does not load: %v", err)
	}

	return ps3ParentCache{BackoffUntil: cache.BackoffUntil, Failures: cache.Failures}
}

// ps3DropEntry removes xid's entry from the outbox file, leaving its note.
func ps3DropEntry(t *testing.T, env *wiringEnv, xid string) {
	t.Helper()

	ps3RewriteOutbox(t, env, func(entries []map[string]any) []map[string]any {
		return slices.DeleteFunc(entries, func(entry map[string]any) bool { return entry["xid"] == xid })
	})
}

// ps3Enqueue queues xid's note as another process would: through the outbox adapter, with no drain.
func ps3Enqueue(t *testing.T, env *wiringEnv, xid string) {
	t.Helper()

	deps := env.deps()
	lock := func(string) (func(), error) { return func() {}, nil }
	list := func(string) ([]string, error) { return []string{}, nil }
	store := cli.ExportNewOutboxStore(cli.ExportExchangeStateFromDeps(deps), lock, list, env.now, io.Discard)

	err := cli.ExportEnqueueOutbox(store, env.vault, xid)
	if err != nil {
		t.Errorf("queueing %s: %v", xid, err)
	}
}

// ps3FirstAnswer is the one answer given, or nil for the env's fake parent.
func ps3FirstAnswer(answers []ps3AnswerFunc) ps3AnswerFunc {
	if len(answers) == 0 {
		return nil
	}

	return answers[0]
}

// ps3Install wires the env's commands to the hook, with the env's own git identity.
func ps3Install(env *wiringEnv, hook *ps3Hook) {
	identity := ps2Identity(ps2User)

	env.wrap = func(deps *cli.Deps) {
		identity(deps)

		inner := deps.Fetch
		deps.Fetch = func(ctx context.Context, method, target string, body []byte) (cli.FetchResponse, error) {
			if method != "POST" || !strings.HasSuffix(target, "/learn") {
				return inner(ctx, method, target, body)
			}

			var args cli.LearnArgs

			_ = json.Unmarshal(body, &args)

			hook.mu.Lock()
			hook.slugs = append(hook.slugs, args.Slug)
			offer := hook.offer
			hook.mu.Unlock()

			if offer != nil {
				if answer := offer(args); answer != nil {
					return answer.resp, answer.err
				}
			}

			return inner(ctx, method, target, body)
		}
	}
}

// ps3Offer expects the POST of one offer, answered by answer or the env's fake parent.
func ps3Offer(answer ...ps3AnswerFunc) ps3Step {
	return ps3Step{method: "POST", route: "/learn", answer: ps3FirstAnswer(answer)}
}

// ps3Outbox is the vault's outbox.
func ps3Outbox(t *testing.T, env *wiringEnv) cli.OutboxFileForTest {
	t.Helper()

	box, err := cli.ExportLoadOutbox(cli.ExportExchangeStateFromDeps(env.deps()), env.vault)
	if err != nil {
		t.Fatalf("the outbox does not load: %v", err)
	}

	return box
}

// ps3Query expects the query's own request to the parent, answered by answer or the env's fake parent.
func ps3Query(answer ...ps3AnswerFunc) ps3Step {
	return ps3Step{method: "GET", route: "/query", answer: ps3FirstAnswer(answer)}
}

// ps3QueryArgs is the command line of a drain: a query, which drains once the parent has answered it.
func ps3QueryArgs(env *wiringEnv) []string {
	return []string{"query", "--phrase", "x", "--chunks-dir", env.tempDir()}
}

// ps3Receipt is the JSON body of a receipt.
func ps3Receipt(t *testing.T, receipt cli.OfferReceiptForTest) string {
	t.Helper()

	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}

	return string(encoded)
}

// ps3ReverseOutbox lists the outbox file's entries in the opposite order.
func ps3ReverseOutbox(t *testing.T, env *wiringEnv) {
	t.Helper()

	ps3RewriteOutbox(t, env, func(entries []map[string]any) []map[string]any {
		slices.Reverse(entries)

		return entries
	})
}

// ps3RewriteOutbox rewrites the outbox file's entries.
func ps3RewriteOutbox(t *testing.T, env *wiringEnv, change func([]map[string]any) []map[string]any) {
	t.Helper()

	var box struct {
		Version int              `json:"version"`
		Entries []map[string]any `json:"entries"`
	}

	raw := readFileString(t, filepath.Join(env.vault, ".engram", "outbox.json"))

	decodeErr := json.Unmarshal([]byte(raw), &box)
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}

	box.Entries = change(box.Entries)

	encoded, encodeErr := json.Marshal(box)
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}

	env.writeState("outbox.json", string(encoded))
}

// ps3Run runs `engram <args>` in a goroutine and waits for it, and for the script when one is given, with a time
// limit: a request the script does not expect holds the command, and a request it expects but never gets leaves
// the script waiting. The result says which, so that the test fails at its own assertion.
func ps3Run(env *wiringEnv, script <-chan struct{}, args ...string) ps3Result {
	type output struct{ stdout, stderr string }

	done := make(chan output, 1)

	go func() {
		stdout, stderr := env.run(args...)
		done <- output{stdout: stdout, stderr: stderr}
	}()

	var result ps3Result

	select {
	case out := <-done:
		result.stdout, result.stderr, result.finished = out.stdout, out.stderr, true
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

// ps3Script drives the mock Deps.Fetch through steps, in order, and closes the returned channel when the last one
// is answered.
func ps3Script(env *wiringEnv, calls *Pvc1FetchMockMethod, record *ps3Record, steps ...ps3Step) <-chan struct{} {
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		for _, step := range steps {
			call := calls.ArgsShould(match.BeAny, Equal(step.method), ContainSubstring(step.route), match.BeAny)
			args := call.GetArgs()
			record.add(args.Method, args.Url, args.Body)

			answer := step.answer
			if answer == nil {
				answer = func(args Pvc1FetchMockArgs) (cli.FetchResponse, error) {
					return env.parent.fetch(args.Ctx, args.Method, args.Url, args.Body)
				}
			}

			resp, err := answer(args)
			call.Return(resp, err)
		}
	}()

	return finished
}

// ps3Slugs lists the slug of each offer.
func ps3Slugs(offers []cli.LearnArgs) []string {
	slugs := make([]string, 0, len(offers))

	for _, offer := range offers {
		slugs = append(slugs, offer.Slug)
	}

	return slugs
}

// ps3Stamps maps every file under the vault, but its lock file, to its size, modification time and content, so
// that a rewrite shows even when it writes the same bytes.
func ps3Stamps(t *testing.T, vault string) map[string]string {
	t.Helper()

	stamps := map[string]string{}

	walkErr := filepath.WalkDir(vault, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() == ".luhmann.lock" {
			return err
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		stamps[path] = info.ModTime().String() + "\n" + string(raw)

		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}

	return stamps
}

// ps3UseFetch makes the env's commands send their requests through fetch, with the env's own git identity.
func ps3UseFetch(env *wiringEnv, fetch pvc1FetchFunc) {
	identity := ps2Identity(ps2User)

	env.wrap = func(deps *cli.Deps) {
		identity(deps)
		deps.Fetch = fetch
	}
}

// ps3XIDs lists the xid of each note.
func ps3XIDs(t *testing.T, paths []string) []string {
	t.Helper()

	xids := make([]string, 0, len(paths))

	for _, path := range paths {
		xids = append(xids, decodePulledCopy(t, []byte(readFileString(t, path))).XID)
	}

	return xids
}
