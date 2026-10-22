package cli_test

// Scenario tests for the offer-outbox technical component (861c2m): queueing an offer (792b36c4), what a drain
// does before it sends (792b36c5), which entries are sent (792b36c6), an accepted offer (792b36c8) and the update
// exchange with its outbox notice (792b36c16). Spec: docs/specs/technical/offer-outbox.md.
//
// The drain scenarios run over an outboxEnv (an in-memory vault) and the CLI journeys over a wiringEnv: a vault in
// the test's own t.TempDir() (the recorded deviation, #860) and a fake parent behind Deps.Fetch, with no network
// and no real home. The one scenario about git runs real git inside that temp vault. The three interaction
// scenarios drive an imptest mock: MockOo1Send for the drain's send step, and MockPvc1Fetch for Deps.Fetch.

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/internal/cli"
)

// Test792b36c16a1_UpdateExchange_ADryRunOnlyReportsARealRunDrainsThroughTheWindow verifies 792b36c16a1 (A dry run
// only reports, a real run drains through the window): with an entry queued inside an active backoff window, a
// dry run reports the entry and the window and makes no request; a real run then drains despite the window, after
// which the notice is empty.
func Test792b36c16a1_UpdateExchange_ADryRunOnlyReportsARealRunDrainsThroughTheWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)
	env.learnFact("windowed")

	requestsBefore := len(env.parent.requests())
	exchange := cli.ExportUpdateExchange(env.deps())

	dryNotice := exchange(context.Background(), env.vault, true)
	g.Expect(dryNotice).To(ContainSubstring("1 offer(s) queued"))
	g.Expect(dryNotice).To(ContainSubstring("parent unreachable: retrying after"), "the window is reported")
	g.Expect(env.parent.requests()).To(HaveLen(requestsBefore), "the dry run drains nothing")
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	env.parent.setDown(false)

	g.Expect(exchange(context.Background(), env.vault, false)).To(BeEmpty(), "the real run drains; the notice is empty")
	g.Expect(env.parent.offers()).To(HaveLen(1), "the real run sent through the window")
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

// Test792b36c16a2_OutboxNotice_AnIdleOutboxHasNoNotice verifies 792b36c16a2 (An idle outbox has no notice): with
// no entry and no backoff, the notice is empty.
func Test792b36c16a2_OutboxNotice_AnIdleOutboxHasNoNotice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)

	g.Expect(cli.ExportOutboxNotice(env.store, env.vault, parentURL)).To(BeEmpty())
}

// Test792b36c16a3_OutboxNotice_RejectedAndAttentionEntriesAreListed verifies 792b36c16a3 (Rejected and attention
// entries are listed): the notice lists a rejected entry, then an entry that needs attention, each with its note's
// current basename and its error, and its count line separates both from the queued count.
func Test792b36c16a3_OutboxNotice_RejectedAndAttentionEntriesAreListed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.writeNote("3.2026-09-27.c.md", xidC, "c", false)

	base := testNow.Add(-2 * time.Hour)
	env.writeOutbox(map[string]any{"version": 1, "entries": []map[string]any{
		{"xid": xidA, "queued": base, "state": "rejected", "attempts": 1, "last_error": "parent said no"},
		{
			"xid": xidB, "queued": base.Add(time.Minute), "state": "attention", "attempts": 1,
			"last_error": "receipt refused: YAML anchor", "receipt": accepted("9.2026-09-27.x").Receipt,
			"sent_hash": "xh1:sent",
		},
		{"xid": xidC, "queued": base.Add(2 * time.Minute), "state": "queued"},
	}})

	notice := cli.ExportOutboxNotice(env.store, env.vault, parentURL)

	g.Expect(notice).To(HavePrefix(
		"parent outbox: 1 offer(s) queued, 1 rejected, 1 need attention, oldest queued 2h0m0s ago\n"))
	g.Expect(notice).To(ContainSubstring("  rejected: 1.2026-09-27.a: parent said no\n"))
	g.Expect(notice).To(ContainSubstring("  needs attention: 2.2026-09-27.b: receipt refused: YAML anchor\n"))
	g.Expect(strings.Index(notice, "rejected: 1.")).To(BeNumerically("<", strings.Index(notice, "needs attention: 2.")),
		"the rejected entries are listed first")
	g.Expect(notice).NotTo(ContainSubstring("3.2026-09-27.c"), "a queued entry is only counted")
}

// Test792b36c16a4_UpdateExchange_AnAttentionEntryIsCountedAndListedThroughTheProductionWiring verifies
// 792b36c16a4 (An attention entry is counted and listed through the production wiring): a receipt the note
// refuses puts its entry in attention; the first update notice counts and lists it, and the parent is not offered
// the note again; once the anchor is removed, the next update records the receipt and the notice is empty.
func Test792b36c16a4_UpdateExchange_AnAttentionEntryIsCountedAndListedThroughTheProductionWiring(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)

	path := env.learnFact("refused")
	original := readFileString(t, path)
	env.plant(filepath.Base(path), []byte(strings.Replace(original, "\n---\n",
		"\nparent: &p\n    vault: "+strings.Repeat("ef", 16)+"\nparent_copy: *p\n---\n", 1)))
	env.parent.setDown(false)

	exchange := cli.ExportUpdateExchange(env.deps())

	first := exchange(context.Background(), env.vault, false)
	g.Expect(first).To(ContainSubstring("0 offer(s) queued, 0 rejected, 1 need attention"))
	g.Expect(first).To(MatchRegexp(`needs attention: \S+\.refused: .*YAML anchor: parent`))
	g.Expect(env.parent.offers()).To(HaveLen(1), "the note was offered once")

	second := exchange(context.Background(), env.vault, false)
	g.Expect(second).To(ContainSubstring("1 need attention"), "still counted while the anchor stays")
	g.Expect(env.parent.offers()).To(HaveLen(1), "the parent is not offered the note again")

	env.plant(filepath.Base(path), []byte(original))

	g.Expect(exchange(context.Background(), env.vault, false)).To(BeEmpty(), "the receipt is recorded; the notice is idle")
	g.Expect(env.parent.offers()).To(HaveLen(1))
	g.Expect(readFileString(t, path)).To(ContainSubstring("via: offered"), "the receipt links the note")
	g.Expect(env.outboxEntries()).To(BeEmpty())
}

//go:generate impgen oo1SendFunc --dependency --name MockOo1Send

// Test792b36c4a7_OfferHooks_NoParentClassifiesStampsAndQueuesNothing verifies 792b36c4a7 (No parent, no exchange
// state): with no parent configured, a learn, an amend and a resituate classify, stamp and queue nothing and write
// nothing under .engram; the hooks of that configuration are the zero value, which does nothing at all.
func Test792b36c4a7_OfferHooks_NoParentClassifiesStampsAndQueuesNothing(t *testing.T) {
	t.Parallel()

	const unstampedNote = "1.2026-09-27.note.md"

	// a note with no xid, so that a stamp would show
	unstamped := func(env *wiringEnv) {
		raw := strings.Replace(string(offerTestNote{}.render(env.t)), "xid: "+xidA+"\n", "", 1)
		env.plant(unstampedNote, []byte(raw))
	}

	cases := map[string]func(env *wiringEnv) string{
		"learn": func(env *wiringEnv) string { return env.learnFact("plain") },
		"amend": func(env *wiringEnv) string {
			unstamped(env)
			env.run("amend", "--target", "1.2026-09-27.note", "--object", "changed")

			return filepath.Join(env.vault, unstampedNote)
		},
		"resituate": func(env *wiringEnv) string {
			unstamped(env)
			env.run("resituate", "--note", "1.2026-09-27.note", "--situation", "when resituated")

			return filepath.Join(env.vault, unstampedNote)
		},
	}

	for name, write := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.parentURL = ""

			notePath := write(env)

			g.Expect(env.lastStderr).NotTo(ContainSubstring("error"))
			g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("xid:"), "nothing is stamped")
			g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory(), "nothing is written to .engram")
			g.Expect(filepath.Join(env.vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
			g.Expect(env.parent.requests()).To(BeEmpty(), "nothing is sent")
			g.Expect(env.outboxEntries()).To(BeEmpty(), "nothing is queued")
		})
	}

	t.Run("the hooks", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var hooks cli.OfferHooksForTest

		raw := []byte("raw note")
		staged, xid := cli.ExportOfferHooksStageWrite(hooks, "/vault", raw)

		g.Expect(staged).To(Equal(raw), "the note is not classified or stamped")
		g.Expect(xid).To(BeEmpty())
		g.Expect(cli.ExportOfferHooksQueue(hooks, "/vault", xidA)).To(BeFalse(), "nothing is queued")
	})
}

// Test792b36c4a8_EnqueueOutbox_QueueingLeavesTheVaultsTrackedFilesAlone verifies 792b36c4a8 (Queueing leaves the
// vault's tracked files alone): in a git-backed vault with a root .gitignore, queueing an offer changes nothing git
// reports but the note the learn wrote and its sidecar, and the root .gitignore is unchanged.
func Test792b36c4a8_EnqueueOutbox_QueueingLeavesTheVaultsTrackedFilesAlone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const rootIgnore = ".luhmann.lock\n.obsidian/workspace*\n"

	env := newWiringEnv(t)
	env.parent.setDown(true)

	commander := env.deps().Commander
	git := func(args ...string) string {
		t.Helper()

		stdout, stderr, err := commander.Run(t.Context(), env.vault, "git", args...)
		g.Expect(err).NotTo(HaveOccurred(), string(stderr))

		return string(stdout)
	}

	git("init", "-q")
	env.plant(".gitignore", []byte(rootIgnore))

	// the first learn stamps the vault ID and creates .engram/; commit all of it as the baseline
	env.learnFact("baseline")
	git("add", "-A")
	git("-c", "user.email=oo1@example.invalid", "-c", "user.name=oo1", "commit", "-q", "-m", "baseline")
	g.Expect(git("ls-files")).NotTo(ContainSubstring(".engram/"), "the state directory is never tracked")

	queuedNote := env.learnFact("queued")

	g.Expect(env.outboxEntries()).To(HaveLen(2), "the second offer was queued")
	sidecar := strings.TrimSuffix(filepath.Base(queuedNote), ".md") + ".vec.json"

	g.Expect(git("status", "--porcelain")).To(Equal("?? "+filepath.Base(queuedNote)+"\n?? "+sidecar+"\n"),
		"only the note and the sidecar the learn wrote are reported")
	g.Expect(readFileString(t, filepath.Join(env.vault, ".gitignore"))).To(Equal(rootIgnore))
}

// Test792b36c4a9_OfferHooks_ASelfParentVaultQueuesNothing verifies 792b36c4a9 (A self-parent vault queues
// nothing): when the cached parent vault ID is this vault's own, a learn, an amend and a resituate still write
// the note, but queue nothing, send nothing, and print one warning that names engram vault-id --regenerate; the
// amend and the resituate also stamp no xid.
func Test792b36c4a9_OfferHooks_ASelfParentVaultQueuesNothing(t *testing.T) {
	t.Parallel()

	const unstampedNote = "1.2026-09-27.note.md"

	// a note with no xid, so that a stamp would show
	unstamped := func(env *wiringEnv) {
		raw := strings.Replace(string(offerTestNote{}.render(env.t)), "xid: "+xidA+"\n", "", 1)
		env.plant(unstampedNote, []byte(raw))
	}

	cases := map[string]func(env *wiringEnv) string{
		"learn": func(env *wiringEnv) string { return env.learnFact("selfparent") },
		"amend": func(env *wiringEnv) string {
			unstamped(env)
			env.run("amend", "--target", "1.2026-09-27.note", "--object", "changed")

			return filepath.Join(env.vault, unstampedNote)
		},
		"resituate": func(env *wiringEnv) string {
			unstamped(env)
			env.run("resituate", "--note", "1.2026-09-27.note", "--situation", "when resituated")

			return filepath.Join(env.vault, unstampedNote)
		},
	}

	for name, write := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.stampVault("")
			env.parent.vaultID = env.localID()
			env.writeState("parent.json", `{"url":"`+env.parentURL+`","vault_id":"`+env.localID()+`","failures":0}`)

			notePath := write(env)

			g.Expect(notePath).To(BeAnExistingFile(), "the note is written")
			g.Expect(env.outboxEntries()).To(BeEmpty(), "no entry is queued")
			g.Expect(env.parent.offers()).To(BeEmpty(), "nothing is sent")
			g.Expect(nonEmptyLines(env.lastStderr)).To(HaveLen(1), env.lastStderr)
			g.Expect(env.lastStderr).To(ContainSubstring("engram vault-id --regenerate"))

			if name != "learn" {
				g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("xid:"), "the note is not staged")
			}
		})
	}
}

// Test792b36c5a2_DrainOutbox_AFailedLocationCheckSendsNothing verifies 792b36c5a2 (A failed location check sends
// nothing): in a cloned vault, which has no location record, a drain with a queued entry never calls the send
// step (a call would wait on the mock, and the drain would not return), prints the location warning, and leaves
// the entry queued.
func Test792b36c5a2_DrainOutbox_AFailedLocationCheckSendsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.enqueue(xidA)
	env.fsys.remove(filepath.Join(env.vault, ".engram", "home.json"))

	send, _ := MockOo1Send(t)
	outcome := oo1AwaitDrain(t, oo1StartDrain(env, send, (&fakeParent{}).apply))

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.result.Sent).To(BeZero())
	g.Expect(env.stderr.String()).To(ContainSubstring("vault-id --claim"), "the location warning is printed")
	g.Expect(entryXIDs(env.outbox())).To(Equal([]string{xidA}), "the entry stays queued")
}

// Test792b36c5a5_EnsureParentVaultID_AProbeThatFailsKeepsTheOffersQueued verifies 792b36c5a5 (A probe that fails
// keeps the offers queued): with the parent's vault ID uncached, a probe answered with a 4xx, with no vault ID, or
// followed by an unreachable parent leaves the offer queued and prints one warning; the unreachable case also
// records a failure and prints the backoff warning with the queued count.
func Test792b36c5a5_EnsureParentVaultID_AProbeThatFailsKeepsTheOffersQueued(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		configure func(parent *recordingParent)
		warning   string
		failures  int
	}{
		"a 4xx answer": {func(p *recordingParent) { p.queryStatus = 400 }, "bad probe", 0},
		"no vault ID":  {func(p *recordingParent) { p.queryNoID = true }, "too old", 0},
		"unreachable":  {func(p *recordingParent) { p.setDown(true) }, "parent unreachable (retry after", 1},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			testCase.configure(env.parent)

			env.learnFact("probe")

			g.Expect(env.parent.offers()).To(BeEmpty(), "no offer is sent")
			g.Expect(env.outboxEntries()).To(HaveLen(1), "the offer stays queued")
			g.Expect(nonEmptyLines(env.lastStderr)).To(HaveLen(1), "one warning: "+env.lastStderr)
			g.Expect(env.lastStderr).To(ContainSubstring(testCase.warning))
			g.Expect(env.parentCache().Failures).To(Equal(testCase.failures))

			if testCase.failures > 0 {
				g.Expect(env.lastStderr).To(ContainSubstring("1 offer(s) queued"), "the backoff warning counts the queue")
			}
		})
	}
}

// Test792b36c5a6_EnsureParentVaultID_TheProbeCachesTheVaultID verifies 792b36c5a6 (The probe caches the vault
// ID): with the parent's vault ID uncached, the drain's first request is the vault-ID probe, and the ID is in the
// parent cache by the time the first offer is sent, after being built.
func Test792b36c5a6_EnsureParentVaultID_TheProbeCachesTheVaultID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		mu       sync.Mutex
		requests []oo1Request
	)

	env := newWiringEnv(t)
	env.wrap = func(deps *cli.Deps) {
		inner := deps.Fetch
		deps.Fetch = func(ctx context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			mu.Lock()
			requests = append(requests, oo1Request{method: method, url: url, cachedID: env.parentCache().VaultID})
			mu.Unlock()

			return inner(ctx, method, url, body)
		}
	}

	env.learnFact("cached")

	mu.Lock()
	defer mu.Unlock()

	g.Expect(requests).To(HaveLen(2))

	if len(requests) != 2 {
		return
	}

	g.Expect(requests[0].method).To(Equal("GET"))
	g.Expect(requests[0].url).To(ContainSubstring("/query"))
	g.Expect(requests[0].url).To(ContainSubstring("dedupe-keys"))
	g.Expect(requests[0].cachedID).To(BeEmpty(), "the ID was not known before the probe")
	g.Expect(requests[1].method).To(Equal("POST"))
	g.Expect(requests[1].url).To(HaveSuffix("/learn"))
	g.Expect(requests[1].cachedID).To(Equal(parentVaultID), "the ID was cached before the offer was sent")
	g.Expect(env.parent.offers()).To(HaveLen(1))
}

// Test792b36c5a7_DrainForCommand_ABackedOffParentIsNotDrained verifies 792b36c5a7 (A backed-off parent is not
// drained): a command other than update, run inside a backoff window, sends nothing (Deps.Fetch is a mock that
// is never given an expectation, so a request would hold the command), yet the kept receipt of an attention entry
// was still recorded on its note.
func Test792b36c5a7_DrainForCommand_ABackedOffParentIsNotDrained(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env, path := heldReceiptWiringEnv(t, parentURL, seqID(91), seqID(91))

	fetch, _ := MockPvc1Fetch(t)
	env.wrap = func(deps *cli.Deps) { deps.Fetch = fetch }

	_, stderr := pvc1RunWithin(t, env, nil, "learn", "fact", "--slug", "backoff", "--situation", "s",
		"--subject", "a", "--predicate", "b", "--object", "c", "--source", "t", "--position", "top")

	g.Expect(stderr).To(ContainSubstring("engram: parent unreachable (retry after "), "the window is honored")
	g.Expect(readFileString(t, path)).To(ContainSubstring("note: 9.2026-09-27.parent-copy"),
		"the kept receipt was recorded")
	g.Expect(env.outboxEntries()).To(HaveLen(1), "only the new note's entry is left, unsent")
}

// Test792b36c6a6_DrainOutbox_ACRLFNoteIsFoundAndSent verifies 792b36c6a6 (A CRLF note is found and sent): the
// entry of a queued note with CRLF line endings is not dropped as a note that is gone; its offer is sent, carrying
// the note's exchange hash.
func Test792b36c6a6_DrainOutbox_ACRLFNoteIsFoundAndSent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	crlf := toCRLF(plainParentNote("body one"))
	env.fsys.put(attentionNotePath(env), []byte(crlf))
	env.enqueue(xidA)

	send, sendCalls := MockOo1Send(t)
	done := oo1StartDrain(env, send, (&fakeParent{}).apply)

	// the expectation runs apart, so that a call that does not match it cannot hold the test
	sent := make(chan cli.OfferNoteForTest, 1)

	go func() {
		call := sendCalls.ArgsShould(match.BeAny, HaveField("XID", xidA))
		sent <- call.GetArgs().Note

		call.Return(accepted("9.2026-09-27.parent-copy"))
	}()

	outcome := oo1AwaitDrain(t, done)

	g.Expect(outcome.err).NotTo(HaveOccurred())
	g.Expect(outcome.result.Dropped).To(BeZero(), "the entry is not treated as gone")
	g.Expect(outcome.result.Sent).To(Equal(1))

	select {
	case note := <-sent:
		g.Expect(note.Basename).To(Equal("1.2026-09-27.a"))
		g.Expect(note.Hash).To(Equal(mustHash(t, []byte(crlf))), "the hash is the note's exchange hash")
	case <-time.After(mockTimeout):
		t.Fatal("the offer was not sent")
	}
}

// Test792b36c8a6_DrainOutbox_ACRLFNoteTakesItsReceiptAsLF verifies 792b36c8a6 (A CRLF note takes its receipt as
// LF): when the offer of a queued CRLF note is accepted, the receipt is recorded by a write that converts the note
// to LF, and its entry is removed.
func Test792b36c8a6_DrainOutbox_ACRLFNoteTakesItsReceiptAsLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.fsys.put(attentionNotePath(env), []byte(toCRLF(plainParentNote("body two"))))
	env.enqueue(xidA)

	parent := &fakeParent{script: map[string]cli.OfferSendResultForTest{xidA: accepted("8.2026-09-27.receipt-copy")}}

	g.Expect(env.drainApplying(parent)).To(Succeed())

	written := string(env.noteRaw(attentionNoteName))
	g.Expect(written).To(ContainSubstring("note: 8.2026-09-27.receipt-copy"), "the receipt is recorded")
	g.Expect(written).NotTo(ContainSubstring("\r"), "the note is written as LF")
	g.Expect(written).To(ContainSubstring("body two"), "the body is kept")
	g.Expect(env.outbox().Entries).To(BeEmpty(), "the entry is removed")
}

// TestCachedSelfParent_AnUnreadableVaultIDIsWarnedAndTheWriteProceeds supports 792b36c4a9: when the parent is
// cached but this vault's own ID cannot be read, the self-parent check cannot be made; the staging failure is
// warned, and the write proceeds unstaged and unqueued (792b36c4a6).
func TestCachedSelfParent_AnUnreadableVaultIDIsWarnedAndTheWriteProceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.stampVault("")
	env.plant(".engram-vault-id", []byte("not-a-vault-id\n"))
	env.writeState("parent.json", `{"url":"`+env.parentURL+`","vault_id":"`+parentVaultID+`","failures":0}`)

	notePath := env.learnFact("unreadable")

	g.Expect(notePath).To(BeAnExistingFile(), "the write proceeds")
	g.Expect(env.lastStderr).To(ContainSubstring("could not stage the offer"))
	g.Expect(env.outboxEntries()).To(BeEmpty(), "nothing is queued")
	g.Expect(env.parent.offers()).To(BeEmpty(), "nothing is sent")
}

// oo1DrainOutcome is what one drain returned.
type oo1DrainOutcome struct {
	result cli.DrainResultForTest
	err    error
}

// oo1Request is one request the fake parent saw, with the parent vault ID that was cached when it arrived.
type oo1Request struct {
	method   string
	url      string
	cachedID string
}

// oo1SendFunc is the signature of the drain's send step; impgen generates MockOo1Send from it.
type oo1SendFunc func(ctx context.Context, note cli.OfferNoteForTest) cli.OfferSendResultForTest

// oo1AwaitDrain waits for a started drain with a time limit: a send the test gives no expectation for holds the
// drain, so the wait fails the test instead of hanging it.
func oo1AwaitDrain(t *testing.T, done <-chan oo1DrainOutcome) oo1DrainOutcome {
	t.Helper()

	select {
	case outcome := <-done:
		return outcome
	case <-time.After(mockTimeout):
		t.Fatal("the drain did not return: it called the send step, which the test expects not to be called")

		return oo1DrainOutcome{}
	}
}

// oo1StartDrain drains env's outbox in a goroutine with the given send step and receipt step.
func oo1StartDrain(
	env *outboxEnv, send oo1SendFunc,
	apply func(cli.OfferNoteForTest, cli.OfferReceiptForTest) error,
) <-chan oo1DrainOutcome {
	done := make(chan oo1DrainOutcome, 1)

	go func() {
		result, err := cli.ExportDrainOutbox(context.Background(), env.store, env.vault, parentURL, send, apply)
		done <- oo1DrainOutcome{result: result, err: err}
	}()

	return done
}
