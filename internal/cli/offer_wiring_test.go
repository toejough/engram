package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestTargets_Amend_BookkeepingQueuesNothing (spec "Bookkeeping amends
// stay local"): --activate and a --supersedes-only amend neither queue
// nor send an offer.
func TestTargets_Amend_BookkeepingQueuesNothing(t *testing.T) {
	t.Parallel()

	for name, flags := range map[string][]string{
		"activate":        {"--activate"},
		"supersedes only": {"--supersedes", "2.2026-09-27.old|updates|c"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			env := newWiringEnv(t)
			env.plant("1.2026-09-27.note.md", offerTestNote{}.render(t))

			_, stderr := env.run(append([]string{"amend", "--target", "1.2026-09-27.note"}, flags...)...)
			g.Expect(stderr).NotTo(ContainSubstring("error"))
			g.Expect(env.parent.requests()).To(BeEmpty())
			g.Expect(filepath.Join(env.vault, ".engram", "outbox.json")).NotTo(BeAnExistingFile())
		})
	}
}

// TestTargets_Amend_ClearPendingOfParentsOfferNeverReturns (D12, spec "An
// accepted offer never returns to its origin vault").
func TestTargets_Amend_ClearPendingOfParentsOfferNeverReturns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.run("learn", "fact", "--slug", "first", "--situation", "s", "--subject", "a", "--predicate", "b",
		"--object", "c", "--source", "t", "--position", "top") // caches the parent's vault ID
	env.plant("5.2026-09-27.served.md", offerTestNote{
		pending: true, origin: parentVaultID + ":" + xidB, path: []string{parentVaultID},
	}.render(t))

	env.run("amend", "--target", "5.2026-09-27.served", "--clear-pending")
	g.Expect(env.parent.offers()).To(HaveLen(1), "only the first learn was offered")
}

// TestTargets_Amend_ClearPendingOfServedOfferPropagates (M14, spec
// "Accepting a served offer propagates upward"): clearing the pending
// marker of a child's served offer sends a learn-offer carrying the
// offer's path plus this vault's ID.
func TestTargets_Amend_ClearPendingOfServedOfferPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	child := seqID(40)
	env.plant("1.2026-09-27.note.md",
		offerTestNote{pending: true, origin: child + ":" + xidB, path: []string{child}}.render(t))

	_, stderr := env.run("amend", "--target", "1.2026-09-27.note", "--clear-pending")
	g.Expect(stderr).To(BeEmpty())

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Offer.Path).To(Equal([]string{child, env.localID()}))
	g.Expect(offers[0].Offer.Origin).To(Equal(env.localID() + ":" + xidA))
}

// TestTargets_Amend_ContentOfLinkedNoteIsAmendOffer (spec "A content amend
// of a linked note is an amend-offer"): after a receipt links the note, a
// content amend's offer names the parent note in offer.for.
func TestTargets_Amend_ContentOfLinkedNoteIsAmendOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	notePath := env.learnFact("linked")

	env.run("amend", "--target", noteBasename(notePath), "--object", "changed")

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(2))
	g.Expect(offers[0].Offer.For).To(BeEmpty())
	g.Expect(offers[1].Offer.For).To(Equal("1100.2026-09-28.linked"))
	g.Expect(offers[1].Object).To(Equal("changed"))
}

// TestTargets_Amend_StampsXIDOnFirstOffer (design D4): a note without an
// xid gets one, in the same write, the first time it is offered; the
// offer's origin names it.
func TestTargets_Amend_StampsXIDOnFirstOffer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	raw := strings.Replace(string(offerTestNote{}.render(t)), "xid: "+xidA+"\n", "", 1)
	env.plant("1.2026-09-27.note.md", []byte(raw))

	env.run("amend", "--target", "1.2026-09-27.note", "--object", "changed")

	content := readFileString(t, filepath.Join(env.vault, "1.2026-09-27.note.md"))
	g.Expect(content).To(MatchRegexp(`(?m)^xid: [0-9a-f]{32}$`))

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))

	xidLine := regexp.MustCompile(`(?m)^xid: ([0-9a-f]{32})$`).FindStringSubmatch(content)
	g.Expect(xidLine).To(HaveLen(2))

	if len(xidLine) == 2 {
		g.Expect(offers[0].Offer.Origin).To(Equal(env.localID() + ":" + xidLine[1]))
	}
}

// TestTargets_Learn_NoParentWritesNoExchangeState (spec "No parent
// configured"): nothing is queued, stamped or sent.
func TestTargets_Learn_NoParentWritesNoExchangeState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parentURL = ""

	notePath := env.learnFact("plain")
	g.Expect(env.parent.requests()).To(BeEmpty())
	g.Expect(filepath.Join(env.vault, ".engram")).NotTo(BeADirectory())
	g.Expect(filepath.Join(env.vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
	g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("xid:"))
}

// TestTargets_Learn_OffersAndLinksReceipt (spec "learn is written locally
// and offered", "Receipt links the note"): one learn-offer carrying this
// vault's origin; the receipt links the note without re-embedding it.
func TestTargets_Learn_OffersAndLinksReceipt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	unlinked := newWiringEnv(t)
	unlinked.parentURL = ""
	unlinked.learnFact("plain")

	env := newWiringEnv(t)
	notePath := env.learnFact("offered")

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Offer.Origin).To(HavePrefix(env.localID() + ":"))
	g.Expect(offers[0].Object).To(Equal("c"))

	content := readFileString(t, notePath)
	g.Expect(content).To(ContainSubstring("xid: "))
	g.Expect(content).To(ContainSubstring("vault: " + parentVaultID))
	g.Expect(content).To(ContainSubstring("note: 1100.2026-09-28.offered"))
	g.Expect(content).To(ContainSubstring("hash: xh1:stored"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(env.embeds.Load()).To(Equal(unlinked.embeds.Load()), "the receipt write never re-embeds")
}

// TestTargets_Learn_ParentUnreachableQueues (spec "Parent unreachable
// during learn"): the command succeeds, the note is live, one entry is
// queued and one warning names the count.
func TestTargets_Learn_ParentUnreachableQueues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)

	stdout, stderr := env.run("learn", "fact", "--slug", "offline", "--situation", "s", "--subject", "a",
		"--predicate", "b", "--object", "c", "--source", "t", "--position", "top")
	g.Expect(strings.TrimSpace(stdout)).To(BeAnExistingFile())

	lines := nonEmptyLines(stderr)
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(HavePrefix("engram: parent unreachable (retry after "))
	g.Expect(lines[0]).To(ContainSubstring("1 offer(s) queued"))
	g.Expect(env.outboxEntries()).To(HaveLen(1))
}

// TestTargets_Learn_TooOldParentKeepsEntry (spec "A pre-change parent is
// detected"): a receipt without vault_id reports a too-old parent and
// leaves the entry queued and the note unlinked.
func TestTargets_Learn_TooOldParentKeepsEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setReceipt(`{"status":"offer received","luhmann":"7"}`)

	notePath := env.learnFact("old")
	g.Expect(env.lastStderr).To(ContainSubstring("too old"))
	g.Expect(env.outboxEntries()).To(HaveLen(1))
	g.Expect(readFileString(t, notePath)).NotTo(ContainSubstring("parent:"))
}

// TestTargets_OfflineWritesCoalesceThenQueryDrains (spec "Offline learn
// then amend coalesce", "Commands inside the backoff window make no
// request", "Query drains a backlog").
func TestTargets_OfflineWritesCoalesceThenQueryDrains(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)

	notePath := env.learnFact("coalesce")
	g.Expect(env.parent.requests()).To(HaveLen(1))

	env.advance(5 * time.Second)
	env.run("amend", "--target", noteBasename(notePath), "--object", "second")
	env.run("amend", "--target", noteBasename(notePath), "--object", "third")
	g.Expect(env.parent.requests()).To(HaveLen(1), "no request inside the backoff window")
	g.Expect(env.lastStderr).To(HavePrefix("engram: parent unreachable (retry after "))
	g.Expect(env.outboxEntries()).To(HaveLen(1))

	env.parent.setDown(false)
	env.advance(time.Minute)
	env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Object).To(Equal("third"))
	g.Expect(env.outboxEntries()).To(BeEmpty())
	g.Expect(readFileString(t, notePath)).To(ContainSubstring("note: 1100.2026-09-28.coalesce"))
}

// TestTargets_Resituate_Offers (E41): resituate is offered like a content
// amend.
func TestTargets_Resituate_Offers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.plant("1.2026-09-27.note.md", offerTestNote{}.render(t))

	env.run("resituate", "--note", "1.2026-09-27.note", "--situation", "when resituated")

	offers := env.parent.offers()
	g.Expect(offers).To(HaveLen(1))
	g.Expect(offers[0].Situation).To(Equal("when resituated"))
}

// TestUpdateExchange_DrainsIgnoringBackoffThenGoesIdle (spec "update always
// tries", "Update reports a stuck outbox"): a dry run only reports; a real
// run drains inside the backoff window, after which the notice is empty.
func TestUpdateExchange_DrainsIgnoringBackoffThenGoesIdle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setDown(true)
	env.learnFact("queued")

	exchange := cli.ExportUpdateExchange(env.deps())

	notice := exchange(context.Background(), env.vault, true)
	g.Expect(notice).To(ContainSubstring("1 offer(s) queued"))
	g.Expect(notice).To(ContainSubstring("oldest queued 0s ago"))
	g.Expect(notice).To(ContainSubstring("parent unreachable"))
	g.Expect(env.parent.requests()).To(HaveLen(1), "a dry run sends nothing")

	env.parent.setDown(false)
	g.Expect(exchange(context.Background(), env.vault, false)).To(BeEmpty())
	g.Expect(env.parent.offers()).To(HaveLen(1))
}

// TestUpdateExchange_IdleVaultHasNoNotice: nothing queued and no backoff —
// no notice.
func TestUpdateExchange_IdleVaultHasNoNotice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	g.Expect(cli.ExportUpdateExchange(env.deps())(context.Background(), env.vault, false)).To(BeEmpty())
}

// TestUpdateExchange_ReportsRejectedEntry: a rejected entry is listed with
// its note and error.
func TestUpdateExchange_ReportsRejectedEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parent.setRejection(`{"error":"no thanks"}`)
	env.learnFact("refused")

	notice := cli.ExportUpdateExchange(env.deps())(context.Background(), env.vault, true)
	g.Expect(notice).To(ContainSubstring("0 offer(s) queued"))
	g.Expect(notice).To(ContainSubstring("1 rejected"))
	g.Expect(notice).To(MatchRegexp(`rejected: \S+\.refused: .*no thanks`))
}

// TestWriteUpdateReport_OutboxNotice: the report carries the notice.
func TestWriteUpdateReport_OutboxNotice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var out strings.Builder

	report := cli.UpdateReportWithOutboxNotice("parent outbox: 2 offer(s) queued\n")
	g.Expect(cli.ExportWriteUpdateReport(&out, report)).To(Succeed())
	g.Expect(out.String()).To(ContainSubstring("parent outbox: 2 offer(s) queued\n"))
}

// embedCounter counts Embed calls (safe across goroutines).
type embedCounter struct {
	calls *atomic.Int32
}

func (c embedCounter) Dims() int { return 4 }

func (c embedCounter) Embed(context.Context, string) ([]float32, error) {
	c.calls.Add(1)

	return []float32{1, 0, 0, 0}, nil
}

func (c embedCounter) ModelID() string { return "m@4" }

type recordedRequest struct {
	method string
	url    string
	body   []byte
	failed bool
}

// recordingParent is a fake parent over Deps.Fetch: POST /learn answers
// with a receipt (or is down), GET /query with an empty payload.
type recordingParent struct {
	mu        sync.Mutex
	down      bool
	receipt   string
	rejection string
	log       []recordedRequest
}

func (p *recordingParent) fetch(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.log = append(p.log, recordedRequest{method: method, url: url, body: body, failed: p.down})

	if p.down {
		return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
	}

	if strings.Contains(url, "/query") {
		return cli.FetchResponse{Status: 200, Body: []byte("version: 1\n")}, nil
	}

	if p.rejection != "" {
		return cli.FetchResponse{Status: 400, Body: []byte(p.rejection)}, nil
	}

	if p.receipt != "" {
		return cli.FetchResponse{Status: 200, Body: []byte(p.receipt)}, nil
	}

	var args cli.LearnArgs
	_ = json.Unmarshal(body, &args)

	receipt, marshalErr := json.Marshal(cli.OfferReceiptForTest{
		Status: "offer received", Luhmann: "1100", Basename: "1100.2026-09-28." + args.Slug,
		Pending: true, VaultID: parentVaultID, StoredHash: "xh1:stored",
	})
	if marshalErr != nil {
		return cli.FetchResponse{}, marshalErr
	}

	return cli.FetchResponse{Status: 200, Body: receipt}, nil
}

func (p *recordingParent) offers() []cli.LearnArgs {
	p.mu.Lock()
	defer p.mu.Unlock()

	offers := make([]cli.LearnArgs, 0, len(p.log))

	for _, request := range p.log {
		if request.method != "POST" || !strings.HasSuffix(request.url, "/learn") || request.failed {
			continue
		}

		var args cli.LearnArgs
		_ = json.Unmarshal(request.body, &args)
		offers = append(offers, args)
	}

	return offers
}

func (p *recordingParent) requests() []recordedRequest {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]recordedRequest(nil), p.log...)
}

func (p *recordingParent) setDown(down bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.down = down
}

func (p *recordingParent) setReceipt(receipt string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.receipt = receipt
}

func (p *recordingParent) setRejection(rejection string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.rejection = rejection
}

// wiringEnv is one child vault on disk with a fake parent and a settable
// clock, driven through cli.Targets.
type wiringEnv struct {
	t          *testing.T
	vault      string
	parentURL  string
	parent     *recordingParent
	embeds     *atomic.Int32
	clockMu    sync.Mutex
	clock      time.Time
	lastStderr string
}

func (e *wiringEnv) advance(by time.Duration) {
	e.clockMu.Lock()
	defer e.clockMu.Unlock()

	e.clock = e.clock.Add(by)
}

func (e *wiringEnv) customize(deps *cli.Deps) {
	deps.Getenv = parentOnlyGetenv(e.parentURL)
	deps.Now = e.now
	deps.Fetch = e.parent.fetch
	deps.Embed = embedCounter{calls: e.embeds}
}

func (e *wiringEnv) deps() cli.Deps {
	deps := newTestDeps(&strings.Builder{}, &strings.Builder{})
	e.customize(&deps)

	return deps
}

// learnFact learns a fact note and returns its path.
func (e *wiringEnv) learnFact(slug string) string {
	e.t.Helper()

	stdout, stderr := e.run("learn", "fact", "--slug", slug, "--situation", "s", "--subject", "a",
		"--predicate", "b", "--object", "c", "--source", "t", "--position", "top")

	path := strings.TrimSpace(stdout)
	if path == "" {
		e.t.Fatalf("learn wrote no note: %s", stderr)
	}

	return path
}

func (e *wiringEnv) localID() string {
	return strings.TrimSpace(readFileString(e.t, filepath.Join(e.vault, ".engram-vault-id")))
}

func (e *wiringEnv) now() time.Time {
	e.clockMu.Lock()
	defer e.clockMu.Unlock()

	return e.clock
}

func (e *wiringEnv) outboxEntries() []any {
	e.t.Helper()

	raw, err := os.ReadFile(filepath.Join(e.vault, ".engram", "outbox.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		e.t.Fatal(err)
	}

	var box struct {
		Entries []any `json:"entries"`
	}

	unmarshalErr := json.Unmarshal(raw, &box)
	if unmarshalErr != nil {
		e.t.Fatal(unmarshalErr)
	}

	return box.Entries
}

func (e *wiringEnv) plant(name string, raw []byte) {
	e.t.Helper()

	err := os.WriteFile(filepath.Join(e.vault, name), raw, 0o600)
	if err != nil {
		e.t.Fatal(err)
	}
}

// run executes `engram <args> --vault <vault>` and returns its output.
func (e *wiringEnv) run(args ...string) (string, string) {
	e.t.Helper()

	stdout, stderr := executeCapturingBoth(e.t, append(append([]string{"engram"}, args...), "--vault", e.vault),
		e.customize)
	e.lastStderr = stderr

	return stdout, stderr
}

func newWiringEnv(t *testing.T) *wiringEnv {
	t.Helper()

	return &wiringEnv{
		t: t, vault: t.TempDir(), parentURL: parentURL, parent: &recordingParent{},
		embeds: &atomic.Int32{}, clock: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC),
	}
}

func noteBasename(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".md")
}
