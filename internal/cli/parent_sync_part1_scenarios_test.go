package cli_test

// Scenario tests for the parent-sync product capability, part 1 (861c2zx): the ENGRAM_SERVER refusal (792a21a), the
// one parent (792a21b), the merged query's reach across the CLI (792a21c, 792a21f, 792a21g, 792a21h). Spec:
// docs/specs/product/parent-sync.md. The merge's own ordering, budgets and dedupe are in
// parent_sync_part1_scenarios_internal_test.go (white-box: mergeQueryPayloads is unexported).
//
// The CLI journeys run over a wiringEnv: a vault in the test's own t.TempDir() (the recorded deviation, #860) and a
// fake parent behind Deps.Fetch, with no network and no real home. The merged query scores its local side with the
// real query engine over planted sidecars, so a journey needs a real vault directory.

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

// Test792a21a1b_Targets_Learn_RefusedWhenEngramServerIsSet verifies 792a21a1b (Writes are refused too): with
// ENGRAM_SERVER set, `engram learn fact` exits non-zero with the ENGRAM_PARENT error, writes no note in the local
// vault (the vault directory stays empty) and sends nothing to the parent that ENGRAM_PARENT names.
func Test792a21a1b_Targets_Learn_RefusedWhenEngramServerIsSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const serverURL = "http://host:8093"

	env := newWiringEnv(t)
	env.wrap = func(deps *cli.Deps) {
		withParent := deps.Getenv
		deps.Getenv = func(key string) string {
			if key == removedServerEnvVar {
				return serverURL
			}

			return withParent(key)
		}
	}

	stdout, stderr := env.run("learn", "fact", "--slug", "refused", "--situation", "s", "--subject", "a",
		"--predicate", "b", "--object", "c", "--source", "t", "--position", "top")

	entries, readErr := os.ReadDir(env.vault)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty(), "no note, sidecar, vault ID or outbox is written locally")
	g.Expect(stdout).To(BeEmpty(), "no note path is reported")
	g.Expect(stderr).To(ContainSubstring("ENGRAM_SERVER is no longer supported"))
	g.Expect(stderr).To(ContainSubstring("ENGRAM_PARENT=" + serverURL))
	g.Expect(env.exitCodes()).To(Equal([]int{1}))
	g.Expect(env.parent.requests()).To(BeEmpty(), "no offer is sent, so nothing is written remotely")
}

// Test792a21a1c_Targets_Serve_RefusedWhenEngramServerIsSet verifies 792a21a1c (serve is not exempt): with
// ENGRAM_SERVER set, `engram serve --addr 127.0.0.1:8093` exits non-zero with the same ENGRAM_PARENT error, and no
// mux is built, no route registered and no listener started (the recorded bind capability is never called).
func Test792a21a1c_Targets_Serve_RefusedWhenEngramServerIsSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const serverURL = "http://host:8093"

	rec := &capabilityRecorder{}

	stderr, exitCodes := runWithRemovedServer(rec, serverURL,
		[]string{"engram", "serve", "--addr", "127.0.0.1:8093"})

	g.Expect(exitCodes).To(ContainElement(1), "exits non-zero")
	g.Expect(stderr).To(ContainSubstring("ENGRAM_SERVER is no longer supported"))
	g.Expect(stderr).To(ContainSubstring("ENGRAM_PARENT=" + serverURL))
	g.Expect(rec.bindCalls.Load()).To(BeZero(), "no mux, no route and no listener")
	g.Expect(rec.fsCalls.Load()).To(BeZero(), "the vault is not read either")
}

// Test792a21b1a_Targets_Query_ENGRAMPARENTNamesTheOneParent verifies 792a21b1a (ENGRAM_PARENT names the one
// parent): with ENGRAM_PARENT set to a served vault's base URL, `engram query` asks that URL and no other for the
// parent's answer, and the note only that parent holds comes back in the merged payload tagged from_parent.
func Test792a21b1a_Targets_Query_ENGRAMPARENTNamesTheOneParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)

	parentNote := pulledFact("7.2026-09-01.ps1-parent-only", "c", "")
	env.parent.addNote(parentNote)
	env.parent.queryNotes = true

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())

	requests := env.parent.requests()
	g.Expect(requests).NotTo(BeEmpty(), "the parent is asked")

	for _, request := range requests {
		g.Expect(request.url).To(HavePrefix(env.parentURL+"/"), "every request goes to the one parent")
	}

	answer := ps1ParseAnswer(t, stdout)
	item, found := ps1ItemAt(answer, parentNote.basename+".md")
	g.Expect(found).To(BeTrue(), "the parent's note is in the merged payload")
	g.Expect(item.FromParent).To(HaveValue(BeTrue()))
}

// Test792a21c1b_Targets_Query_AParentChunkNeverAppears verifies 792a21c1b (A parent chunk never appears): a
// parent whose answer holds a direct chunk and a recency chunk beside a note contributes the note only; neither
// chunk is in the merged payload, and no item of the payload has the chunk kind.
func Test792a21c1b_Targets_Query_AParentChunkNeverAppears(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		parentNote   = "7.2026-09-01.ps1-parent-note.md"
		directChunk  = "/sessions/parent.jsonl#turn-1"
		recentChunk  = "/sessions/parent.jsonl#turn-2"
		parentAnswer = "version: 1\nvault_id: " + ps1ParentVaultID + "\nmodel_id: m@4\nitems:\n" +
			"  - path: " + parentNote + "\n    kind: fact\n    score: 0.5\n    provenances: [direct]\n" +
			"  - path: " + directChunk + "\n    kind: chunk\n    score: 0.95\n    provenances: [direct]\n" +
			"  - path: " + recentChunk + "\n    kind: chunk\n    score: 0\n    provenances: [recent]\n"
	)

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)

	parent := &ps1Parent{body: parentAnswer}
	env.wrap = func(deps *cli.Deps) { deps.Fetch = parent.fetch }

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(parent.queried()).To(BeTrue(), "the parent was asked, so the chunks were dropped by the merge")

	answer := ps1ParseAnswer(t, stdout)
	paths := ps1Paths(answer)

	g.Expect(paths).To(ConsistOf("1.fact.md", parentNote))
	g.Expect(stdout).NotTo(ContainSubstring(directChunk))
	g.Expect(stdout).NotTo(ContainSubstring(recentChunk))

	for _, item := range answer.Items {
		g.Expect(item.Kind).NotTo(Equal("chunk"))
	}
}

// Test792a21f1a_Targets_Query_AMismatchedModelIDDoesNotBlockTheMerge verifies 792a21f1a (Mismatched model_id does
// not block the merge): a local vault on model m@4 and a parent on another model still give one payload with the
// items of both sources, no error and no warning, and no exit.
func Test792a21f1a_Targets_Query_AMismatchedModelIDDoesNotBlockTheMerge(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const parentNote = "7.2026-09-01.ps1-other-model.md"

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)

	parent := &ps1Parent{body: ps1ParentAnswer("other-model@8", parentNote)}
	env.wrap = func(deps *cli.Deps) { deps.Fetch = parent.fetch }

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(env.exitCodes()).To(BeEmpty())

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(ps1Paths(answer)).To(ConsistOf("1.fact.md", parentNote), "one payload holding both sources' items")
}

// Test792a21f1b_Targets_Query_EveryMergedItemCarriesItsSourcesModel verifies 792a21f1b (Every merged item carries
// its source's model): in a merged payload the local item carries the local model_id and the parent's item carries
// the parent's, whatever the payload-level model_id says.
func Test792a21f1b_Targets_Query_EveryMergedItemCarriesItsSourcesModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		parentNote  = "7.2026-09-01.ps1-other-model.md"
		parentModel = "other-model@8"
	)

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)

	parent := &ps1Parent{body: ps1ParentAnswer(parentModel, parentNote)}
	env.wrap = func(deps *cli.Deps) { deps.Fetch = parent.fetch }

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(answer.Items).To(HaveLen(2))

	for _, item := range answer.Items {
		g.Expect(item.ModelID).NotTo(BeEmpty(), "item %s carries a model_id", item.Path)
	}

	local, localFound := ps1ItemAt(answer, "1.fact.md")
	g.Expect(localFound).To(BeTrue())
	g.Expect(local.ModelID).To(Equal("m@4"), "the local item has the local node's model")

	remote, remoteFound := ps1ItemAt(answer, parentNote)
	g.Expect(remoteFound).To(BeTrue())
	g.Expect(remote.ModelID).To(Equal(parentModel), "the parent's item has the parent's model")
}

// Test792a21g1b_Targets_Query_APayloadWithNoMergeHasNoFromParent verifies 792a21g1b (A payload with no merge has
// no from_parent): with ENGRAM_PARENT unset, `engram query` prints its items with no from_parent key at all (not
// even false) and contacts no parent.
func Test792a21g1b_Targets_Query_APayloadWithNoMergeHasNoFromParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	env.parentURL = ""
	ps1PlantLocalNote(t, env.vault)

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(ps1Paths(answer)).To(Equal([]string{"1.fact.md"}), "the local item is returned")
	g.Expect(stdout).NotTo(ContainSubstring("from_parent"))
	g.Expect(env.parent.requests()).To(BeEmpty())
}

// Test792a21h1b_Targets_Query_ThePendingOfferFlagIsTheLocalVaults verifies 792a21h1b (The pending-offer flag is
// the local vault's): a parent whose answer says it holds a pending offer, with a local vault that holds none,
// gives a merged payload with neither pending_offers nor its hint.
func Test792a21h1b_Targets_Query_ThePendingOfferFlagIsTheLocalVaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parentAnswer := "version: 1\nvault_id: " + ps1ParentVaultID + "\npending_offers: true\n" +
		"pending_offers_hint: curate them\nmodel_id: m@4\nitems:\n" +
		"  - path: 7.2026-09-01.ps1-parent.md\n    kind: fact\n    score: 0.5\n    provenances: [direct]\n"

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)

	parent := &ps1Parent{body: parentAnswer}
	env.wrap = func(deps *cli.Deps) { deps.Fetch = parent.fetch }

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(parent.queried()).To(BeTrue(), "the payload is a merged one")
	g.Expect(stdout).To(ContainSubstring("from_parent: true"), "the payload is a merged one")
	g.Expect(stdout).NotTo(ContainSubstring("pending_offers"))
	g.Expect(stdout).NotTo(ContainSubstring("curate them"))
}

// Test792a21h1c_Targets_Query_ALocalPendingOfferShowsInAMergedPayload verifies 792a21h1c (A local pending offer
// shows in a merged payload): a local vault holding a pending offer gives a merged payload with pending_offers
// true and its hint, though the parent reports none.
func Test792a21h1c_Targets_Query_ALocalPendingOfferShowsInAMergedPayload(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newWiringEnv(t)
	ps1PlantLocalNote(t, env.vault)
	plantRealVaultNote(t, env.vault, "2.pending.md",
		"---\ntype: fact\ntier: L2\nsituation: x\npending: true\n---\n\npending body\n", []float32{1, 0, 0, 0})

	parent := &ps1Parent{body: ps1ParentAnswer("m@4", "7.2026-09-01.ps1-parent.md")}
	env.wrap = func(deps *cli.Deps) { deps.Fetch = parent.fetch }

	stdout, stderr := env.run("query", "--phrase", "x", "--chunks-dir", t.TempDir())

	g.Expect(stderr).To(BeEmpty())
	g.Expect(parent.queried()).To(BeTrue(), "the payload is a merged one")

	answer := ps1ParseAnswer(t, stdout)
	g.Expect(answer.PendingOffers).To(BeTrue())
	g.Expect(answer.PendingOffersHint).To(ContainSubstring("curat"), "the hint says to curate the pending offers")
	g.Expect(ps1Paths(answer)).To(ConsistOf("1.fact.md", "7.2026-09-01.ps1-parent.md"),
		"the pending offer itself is not an item")
}

// unexported constants.
const (
	// ps1ParentVaultID is the vault ID the scripted parent reports.
	ps1ParentVaultID = "9a1e9a1e9a1e9a1e9a1e9a1e9a1e9a1e"
)

// unexported variables.
var (
	errPs1UnexpectedRoute = errors.New("ps1 parent: only /query is scripted")
)

// ps1Answer is the part of a printed query payload that the parent-sync part 1 journeys read.
type ps1Answer struct {
	PendingOffers     bool      `yaml:"pending_offers"`
	PendingOffersHint string    `yaml:"pending_offers_hint"`
	Items             []ps1Item `yaml:"items"`
}

// ps1Item is one item of a printed query payload.
type ps1Item struct {
	Path       string `yaml:"path"`
	Kind       string `yaml:"kind"`
	ModelID    string `yaml:"model_id"`
	FromParent *bool  `yaml:"from_parent"`
}

// ps1Parent is a scripted parent: GET /query answers a fixed YAML body, any other route fails, and every URL asked
// is recorded.
type ps1Parent struct {
	mu   sync.Mutex
	body string
	urls []string
}

// fetch is the Deps.Fetch of the scripted parent.
func (p *ps1Parent) fetch(_ context.Context, _, url string, _ []byte) (cli.FetchResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.urls = append(p.urls, url)

	if !strings.Contains(url, "/query") {
		return cli.FetchResponse{}, errPs1UnexpectedRoute
	}

	return cli.FetchResponse{Status: 200, Body: []byte(p.body)}, nil
}

// queried reports whether the parent's /query route was asked.
func (p *ps1Parent) queried() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, url := range p.urls {
		if strings.Contains(url, "/query") {
			return true
		}
	}

	return false
}

// ps1ItemAt finds the item of a printed payload with the given path.
func ps1ItemAt(answer ps1Answer, path string) (ps1Item, bool) {
	for _, item := range answer.Items {
		if item.Path == path {
			return item, true
		}
	}

	return ps1Item{}, false
}

// ps1ParentAnswer is a parent's answer holding one direct note, on the given model.
func ps1ParentAnswer(modelID, notePath string) string {
	return "version: 1\nvault_id: " + ps1ParentVaultID + "\nmodel_id: " + modelID + "\nitems:\n" +
		"  - path: " + notePath + "\n    kind: fact\n    score: 0.5\n    provenances: [direct]\n"
}

// ps1ParseAnswer decodes a printed query payload.
func ps1ParseAnswer(t *testing.T, stdout string) ps1Answer {
	t.Helper()

	g := NewWithT(t)

	var answer ps1Answer

	g.Expect(yaml.Unmarshal([]byte(stdout), &answer)).To(Succeed())

	return answer
}

// ps1Paths lists the paths of a printed payload's items in order.
func ps1Paths(answer ps1Answer) []string {
	paths := make([]string, 0, len(answer.Items))
	for _, item := range answer.Items {
		paths = append(paths, item.Path)
	}

	return paths
}

// ps1PlantLocalNote puts one local fact, with its sidecar, in the vault.
func ps1PlantLocalNote(t *testing.T, vault string) {
	t.Helper()

	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})
}
