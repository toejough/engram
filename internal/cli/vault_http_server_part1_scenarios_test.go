package cli_test

// Scenario tests for the vault-http-server technical component, part 1 (861c2j): the starting and stopping of the
// server (792b35c2), the query route (792b35c4), decoding a served learn (792b35c8), refusing an invalid offer
// (792b35c9), a same-origin live offer (792b35c12) and resolving where an offer points (792b35c13). Spec:
// docs/specs/technical/vault-http-server.md. The unrouted-path scenario (792b35c1a2) needs the real net/http mux
// and lives in cmd/engram/vault_http_server_part1_scenarios_test.go.
//
// The scenarios run over the route handlers and RunServe with the deps the neighbouring serve tests use: a vault in
// the test's own t.TempDir() and a deterministic stub embedder (the recorded deviation, #860), no network and no
// real home. Only the listener and the mux are replaced.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792b35c12a1_ServeLearn_ANewKeyFromAnAcceptedOriginMakesANewPendingNote verifies 792b35c12a1 (A new key from
// an accepted origin makes a new pending note): a live note still carrying origin O and a served learn from O with
// a new key write a new pending note whose offer.for names the live note, and the receipt carries for.
func Test792b35c12a1_ServeLearn_ANewKeyFromAnAcceptedOriginMakesANewPendingNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, vhs1LiveName+".md",
		liveFactNote("4", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))

	resp := serveLearnRequest(t, vault, offeredFact("accepted", "changed", offerOf(childOrigin, "key-two", "")))
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Basename).NotTo(Equal(vhs1LiveName))
	g.Expect(receipt.Pending).To(BeTrue())
	g.Expect(receipt.For).To(Equal(vhs1LiveName))
	g.Expect(noteFiles(t, vault)).To(HaveLen(2), "the live note and one new pending note")

	written := readFileString(t, filepath.Join(vault, receipt.Basename+".md"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(ContainSubstring("for: " + vhs1LiveName))
	g.Expect(written).To(ContainSubstring("key: key-two"))
}

// Test792b35c12a2_ServeLearn_TheLiveNoteIsUntouched verifies 792b35c12a2 (The live note is untouched): the request
// of 792b35c12a1 leaves the live note, and every other file of the vault, byte-identical to before.
func Test792b35c12a2_ServeLearn_TheLiveNoteIsUntouched(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	live := writeVaultNote(t, vault, vhs1LiveName+".md",
		liveFactNote("4", "offer:\n  origin: "+childOrigin+"\n  key: key-one\n"))
	liveBefore := readFileString(t, live)
	before := snapshotVault(t, vault)

	resp := serveLearnRequest(t, vault, offeredFact("accepted", "changed", offerOf(childOrigin, "key-two", "")))
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	g.Expect(readFileString(t, live)).To(Equal(liveBefore), "the live note is byte-identical")

	after := snapshotVault(t, vault)
	for path, content := range before {
		g.Expect(after).To(HaveKeyWithValue(path, content), "%s is unchanged", path)
	}

	g.Expect(len(after)).To(BeNumerically(">", len(before)), "the offer was written as new files")
}

// Test792b35c13a2_ServeLearn_ARenamedTargetResolvesThroughAliases verifies 792b35c13a2 (A renamed target resolves
// through aliases): a live note listing a basename in its aliases and an offer whose for names that basename
// resolve to the live note's current basename, which the receipt carries as for.
func Test792b35c13a2_ServeLearn_ARenamedTargetResolvesThroughAliases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, "8.2026-01-08.renamed.md", liveFactNote("8", "aliases:\n  - 3.2026-01-03.old-name\n"))

	resp := serveLearnRequest(t, vault,
		offeredFact("amend-offer", "object", offerOf(childOrigin, "key-a", "3.2026-01-03.old-name")))
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.For).To(Equal("8.2026-01-08.renamed"))
	g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
		To(ContainSubstring("for: 8.2026-01-08.renamed"))
}

// Test792b35c13a3_ServeLearn_ThereIsNoCrossOriginOverwrite verifies 792b35c13a3 (There is no cross-origin
// overwrite): an offer from origin B whose for names a pending note of origin A leaves A's note unchanged, writes a
// new pending note naming it in offer.for, and its receipt has no for.
func Test792b35c13a3_ServeLearn_ThereIsNoCrossOriginOverwrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const pendingName = "5.2026-01-05.from-a"

	vault := newServeVault(t)
	other := writeVaultNote(t, vault, pendingName+".md",
		pendingOfferFactNote("5", "offer:\n  origin: "+otherOrigin+"\n  key: key-a\n"))
	otherBefore := readFileString(t, other)

	resp := serveLearnRequest(t, vault, offeredFact("from-b", "object", offerOf(childOrigin, "key-b", pendingName)))
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Basename).NotTo(Equal(pendingName))
	g.Expect(receipt.For).To(BeEmpty(), "for names only a live target")
	g.Expect(string(resp.Body)).NotTo(ContainSubstring(`"for"`))
	g.Expect(readFileString(t, other)).To(Equal(otherBefore), "origin A's pending note is unchanged")
	g.Expect(readFileString(t, filepath.Join(vault, receipt.Basename+".md"))).
		To(ContainSubstring("for: " + pendingName))
}

// Test792b35c13a4_ServeLearn_AnUnresolvableTargetIsDropped verifies 792b35c13a4 (An unresolvable target is
// dropped): an offer whose for names no note, alias or pending note is written as a new pending note without
// offer.for, and the response carries no for.
func Test792b35c13a4_ServeLearn_AnUnresolvableTargetIsDropped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	writeVaultNote(t, vault, vhs1LiveName+".md", liveFactNote("4", ""))

	resp := serveLearnRequest(t, vault,
		offeredFact("amend-offer", "object", offerOf(childOrigin, "key-a", "99.2026-01-01.nowhere")))
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))
	g.Expect(string(resp.Body)).NotTo(ContainSubstring(`"for"`))

	receipt := decodeReceipt(t, resp)
	g.Expect(receipt.Pending).To(BeTrue())

	written := readFileString(t, filepath.Join(vault, receipt.Basename+".md"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).NotTo(ContainSubstring("for:"))
	g.Expect(written).NotTo(ContainSubstring("nowhere"))
}

// Test792b35c2a2_RunServe_ACloseAfterCancellationIsNotAnError verifies 792b35c2a2 (A close after cancellation is
// not an error): the listener fails the way a closed server does, the context is canceled, and RunServe returns no
// error, whether the cancellation came before it started or while it served.
func Test792b35c2a2_RunServe_ACloseAfterCancellationIsNotAnError(t *testing.T) {
	t.Parallel()

	errListenerClosed := errors.New("listen on 127.0.0.1:0: server closed")

	cases := map[string]struct {
		cancelBeforeStart bool
	}{
		"canceled before it starts": {cancelBeforeStart: true},
		"canceled while it serves":  {cancelBeforeStart: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if testCase.cancelBeforeStart {
				cancel()
			}

			listened := false
			deps := newTestDeps(io.Discard, io.Discard)
			deps.NewServeMux = func() cli.RawServeMux { return "fake-mux" }
			deps.RegisterRoute = func(cli.RawServeMux, string, string, cli.ServeHandler) {}
			deps.ListenAndServe = func(_ context.Context, _ cli.RawServeMux, addr string) error {
				listened = true

				g.Expect(addr).To(Equal("127.0.0.1:0"))
				cancel()

				return errListenerClosed
			}

			args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: t.TempDir(), VaultName: "personal", ChunksDir: t.TempDir()}

			err := cli.RunServe(ctx, args, deps)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(listened).To(BeTrue(), "the listener was reached")
		})
	}
}

// Test792b35c2a3_RunServe_ACorruptIDOrAMismatchedLocationWarnsAndServes verifies 792b35c2a3 (A corrupt ID or a
// mismatched location warns and serves): a vault whose ID file is corrupt, or whose location record is for another
// path, makes the server start with one warning that names the remedy and then serve anyway.
func Test792b35c2a3_RunServe_ACorruptIDOrAMismatchedLocationWarnsAndServes(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		prepare  func(t *testing.T) string
		remedies []string
	}{
		"an empty ID file": {
			prepare: func(t *testing.T) string {
				t.Helper()

				return vhs1VaultWithIDFile(t, "")
			},
			remedies: []string{"engram vault-id --regenerate"},
		},
		"an ID file left with a conflict marker": {
			prepare: func(t *testing.T) string {
				t.Helper()

				return vhs1VaultWithIDFile(t, "<<<<<<< HEAD\n")
			},
			remedies: []string{"engram vault-id --regenerate"},
		},
		"a location record for another path": {
			prepare: func(t *testing.T) string {
				t.Helper()

				moved := filepath.Join(t.TempDir(), "moved")
				if err := os.Rename(createdVault(t), moved); err != nil {
					t.Fatalf("moving the vault: %v", err)
				}

				return moved
			},
			remedies: []string{"engram vault-id --regenerate", "engram vault-id --claim"},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := testCase.prepare(t)
			idBefore := vhs1ReadIfPresent(t, filepath.Join(vault, ".engram-vault-id"))

			var stderr bytes.Buffer

			served := false
			args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: vault, VaultName: "personal", ChunksDir: t.TempDir()}

			g.Expect(cli.RunServe(t.Context(), args, fakeServeDeps(&stderr, 1, &served))).To(Succeed())
			g.Expect(served).To(BeTrue(), "the server served anyway")

			lines := nonEmptyLines(stderr.String())
			g.Expect(lines).To(HaveLen(1), "one warning")

			if len(lines) == 0 {
				return
			}

			g.Expect(lines[0]).To(ContainSubstring("warning"))

			for _, remedy := range testCase.remedies {
				g.Expect(lines[0]).To(ContainSubstring(remedy))
			}

			g.Expect(vhs1ReadIfPresent(t, filepath.Join(vault, ".engram-vault-id"))).To(Equal(idBefore),
				"the warning leaves the ID file alone")
		})
	}
}

// Test792b35c4a3_ServeQuery_TheParametersReachTheQuery verifies 792b35c4a3 (The parameters reach the query): a
// request with two phrases, a limit, a project, a content budget, a recent fill, lazy-chunks and timings is
// answered with the payload RunQuery prints for those same arguments against the server's own vault and chunk
// index, and that payload shows each parameter at work.
func Test792b35c4a3_ServeQuery_TheParametersReachTheQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		contentBudget = 1
		limit         = 10
		recentFill    = 1
	)

	vault := t.TempDir()
	chunksDir := t.TempDir()
	phrases := []string{"coverage of queries", "offers"}

	deps := newTestDeps(io.Discard, io.Discard)
	deps.Embed = fixedVectorEmbedder{modelID: "test-model@4", vector: []float32{1, 0, 0, 0}}
	deps.Now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

	vhs1PlantQueryVault(t, deps, vault, chunksDir)

	response := vhs1ServeQueryRoute(t, deps, vault, chunksDir).Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{
			"phrase":         phrases,
			"limit":          {"10"},
			"project":        {"alpha"},
			"content-budget": {"1"},
			"recent-fill":    {"1"},
			"lazy-chunks":    {"true"},
			"timings":        {"true"},
		},
	})
	g.Expect(response.Status).To(Equal(vhs1StatusOK), string(response.Body))

	expected := vhs1RunLocalQuery(t, deps, cli.QueryArgs{
		Phrases: phrases, VaultPath: vault, ChunksDir: chunksDir, Limit: limit, Project: "alpha",
		ContentBudget: contentBudget, RecentFill: recentFill, LazyChunks: true, Timings: true,
	})
	g.Expect(string(response.Body)).To(Equal(expected), "the same payload as the local query with those arguments")

	defaults := vhs1RunLocalQuery(t, deps, cli.QueryArgs{Phrases: phrases, VaultPath: vault, ChunksDir: chunksDir})
	g.Expect(expected).NotTo(Equal(defaults), "the arguments change the payload, so the equality above is not vacuous")

	var unfiltered vhs1QueryPayload

	g.Expect(yaml.Unmarshal([]byte(defaults), &unfiltered)).To(Succeed())
	g.Expect(slices.ContainsFunc(unfiltered.Items, func(item vhs1QueryItem) bool {
		return strings.Contains(item.Path, "beta-")
	})).To(BeTrue(), "without the project the beta note is among the items")

	var payload vhs1QueryPayload

	g.Expect(yaml.Unmarshal(response.Body, &payload)).To(Succeed())
	g.Expect(payload.Phrases).To(Equal(phrases))
	g.Expect(payload.Budget.PhrasesQueried).To(Equal(len(phrases)))
	g.Expect(payload.Budget.Limit).To(Equal(limit))
	g.Expect(payload.Budget.ContentBudget).To(Equal(contentBudget))
	g.Expect(payload.Budget.LazyChunks).To(BeTrue())
	g.Expect(payload.Timings).NotTo(BeNil(), "timings were requested")

	recent := 0

	for _, item := range payload.Items {
		g.Expect(item.Path).To(Or(ContainSubstring("alpha-"), HavePrefix("/sessions/s1.jsonl#")),
			"every item is an alpha note of the server's vault or a chunk of its index")

		if slices.Contains(item.Provenances, "recent") {
			recent++
		}
	}

	g.Expect(recent).To(Equal(recentFill), "the recent fill caps the recency channel at one chunk")
}

// Test792b35c4a4_ServeQuery_AFailingQueryIsA500 verifies 792b35c4a4 (A failing query is a 500): a query that fails
// is answered with 500 and an error body, whether the vault has no usable embeddings or the request names nothing
// to query.
func Test792b35c4a4_ServeQuery_AFailingQueryIsA500(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		query map[string][]string
		cause string
	}{
		"a vault whose notes have no embeddings": {
			query: map[string][]string{"phrase": {"coverage"}},
			cause: "no current-model embeddings",
		},
		"a request with no phrase and no text": {
			query: map[string][]string{},
			cause: "empty query string",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := t.TempDir()
			writeVaultNote(t, vault, vhs1LiveName+".md", liveFactNote("4", ""))

			response := vhs1ServeQueryRoute(t, serveTestDeps(), vault, t.TempDir()).
				Serve(t.Context(), cli.ServeRequest{Query: testCase.query})
			g.Expect(response.Status).To(Equal(vhs1StatusServerError))

			var body struct {
				Error string `json:"error"`
			}

			g.Expect(json.Unmarshal(response.Body, &body)).To(Succeed())
			g.Expect(body.Error).To(ContainSubstring(testCase.cause))
		})
	}
}

// Test792b35c8a7_ServeLearn_TheServersVaultWins verifies 792b35c8a7 (The server's vault wins): a body that names
// another vault, posted to learn, writes the note in the server's vault and nothing in the named one.
func Test792b35c8a7_ServeLearn_TheServersVaultWins(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	named := t.TempDir()

	args := offeredFact("wins", "object", offerOf("", "", ""))
	args.Vault = named

	resp := serveLearnRequest(t, vault, args)
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	receipt := decodeReceipt(t, resp)
	g.Expect(noteFiles(t, vault)).To(ConsistOf(filepath.Join(vault, receipt.Basename+".md")))

	entries, err := os.ReadDir(named)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty(), "nothing is written in the vault the body named")
}

// Test792b35c8a8_ServeLearn_AServedOfferIsTaggedByTheVaultsVocabulary verifies 792b35c8a8 (A served offer is tagged
// by the vault's vocabulary): in a vault with a vocabulary, a body with no tags is written as a pending note that
// carries the vocab/ tags the vault assigns, the term it is near and not the term it is far from.
func Test792b35c8a8_ServeLearn_AServedOfferIsTaggedByTheVaultsVocabulary(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	nearTerm := []float32{1, 0, 0, 0}
	farTerm := []float32{0, 1, 0, 0}

	vhs1PlantVocabTerm(t, vault, "5.2026-01-05.vocab-coverage-definition", nearTerm)
	vhs1PlantVocabTerm(t, vault, "6.2026-01-06.vocab-unrelated-definition", farTerm)

	deps := newTestDeps(io.Discard, io.Discard)
	deps.Embed = fixedVectorEmbedder{modelID: "test-model@4", vector: nearTerm}

	args := offeredFact("tagged", "object", offerOf(childOrigin, "key-a", ""))
	g.Expect(args.Tags).To(BeEmpty(), "the body carries no tags")

	resp := serveLearnWith(t, deps, vault, args)
	g.Expect(resp.Status).To(Equal(vhs1StatusOK), string(resp.Body))

	written := readFileString(t, filepath.Join(vault, decodeReceipt(t, resp).Basename+".md"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(ContainSubstring("- vocab/coverage"))
	g.Expect(written).NotTo(ContainSubstring("vocab/unrelated"))
}

// Test792b35c9a4_ServeLearn_AKeyWithoutAnOriginIsA400 verifies 792b35c9a4 (A key without an origin is a 400): an
// offer that carries offer.key and no offer.origin is answered with 400 naming the origin, and nothing is written.
func Test792b35c9a4_ServeLearn_AKeyWithoutAnOriginIsA400(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	before := snapshotVault(t, vault)

	resp := serveLearnRequest(t, vault, offeredFact("rejected", "object", offerOf("", "key-a", "")))
	g.Expect(resp.Status).To(Equal(vhs1StatusBadRequest))

	var body struct {
		Error string `json:"error"`
	}

	g.Expect(json.Unmarshal(resp.Body, &body)).To(Succeed())
	g.Expect(body.Error).To(ContainSubstring("offer.origin"))
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")
	g.Expect(noteFiles(t, vault)).To(BeEmpty())
}

// Test792b35c9a5_ServeLearn_InvalidNoteContentIsA400 verifies 792b35c9a5 (Invalid note content is a 400): a body
// whose note type is unknown is answered with 400 before any lock is taken, and nothing is written. A valid body on
// the same recording deps takes the lock, so the empty record of the invalid one is not an artifact of the fakes.
func Test792b35c9a5_ServeLearn_InvalidNoteContentIsA400(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newServeVault(t)
	log := &eventLog{}
	deps := serveTestDeps()
	deps.FS = recordingFS{EdgeFS: deps.FS, log: log}
	deps.Lock = recordingLocker{inner: deps.Lock, log: log}

	before := snapshotVault(t, vault)

	invalid := offeredFact("rejected", "object", offerOf(childOrigin, "key-a", ""))
	invalid.Type = "bogus-type"

	resp := serveLearnWith(t, deps, vault, invalid)
	g.Expect(resp.Status).To(Equal(vhs1StatusBadRequest), string(resp.Body))
	g.Expect(log.snapshot()).To(BeEmpty(), "no lock, no read and no write happened")
	g.Expect(snapshotVault(t, vault)).To(Equal(before), "nothing is written")

	valid := serveLearnWith(t, deps, vault, offeredFact("accepted", "object", offerOf(childOrigin, "key-b", "")))
	g.Expect(valid.Status).To(Equal(vhs1StatusOK), string(valid.Body))
	g.Expect(log.snapshot()).To(ContainElement("lock"), "a valid body takes the lock")
}

// unexported constants.
const (
	vhs1LiveName          = "4.2026-01-04.accepted"
	vhs1StatusBadRequest  = 400
	vhs1StatusOK          = 200
	vhs1StatusServerError = 500
)

// vhs1QueryItem is one item of the query payload the parameter scenario reads.
type vhs1QueryItem struct {
	Path        string   `yaml:"path"`
	Provenances []string `yaml:"provenances"`
}

// vhs1QueryPayload is the part of the query payload the parameter scenario reads.
type vhs1QueryPayload struct {
	Phrases []string        `yaml:"phrases"`
	Items   []vhs1QueryItem `yaml:"items"`
	Budget  struct {
		PhrasesQueried int  `yaml:"phrases_queried"`
		Limit          int  `yaml:"limit"`
		ContentBudget  int  `yaml:"content_budget"`
		LazyChunks     bool `yaml:"lazy_chunks"`
	} `yaml:"budget"`
	Timings map[string]any `yaml:"timings"`
}

// vhs1PlantQueryVault writes three notes (two of project alpha, one of project beta) through the local learn path
// and a chunk index of four recency-only chunks into chunksDir.
func vhs1PlantQueryVault(t *testing.T, deps cli.Deps, vault, chunksDir string) {
	t.Helper()

	notes := []struct{ slug, project, object string }{
		{"alpha-one", "alpha", "queries"},
		{"alpha-two", "alpha", "offers"},
		{"beta-one", "beta", "coverage"},
	}

	for _, note := range notes {
		learnLocal(t, deps, vault, cli.LearnArgs{
			Type: "fact", Slug: note.slug, Position: "top", Source: "local", Project: note.project,
			Situation: "a note about " + note.slug, Subject: "engram", Predicate: "covers", Object: note.object,
		})
	}

	// Orthogonal to every note and query vector, so a chunk reaches the payload through the recency channel only.
	recencyOnly := []float32{0, 1, 0, 0}
	ingested := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	texts := []string{"chunk about coverage", "chunk about offers", "chunk about queries", "chunk about nothing"}
	records := make([]chunk.Record, 0, len(texts))

	for index, text := range texts {
		records = append(records, chunk.Record{
			Source: "/sessions/s1.jsonl", Anchor: "turn-" + string(rune('1'+index)),
			ContentHash: chunk.HashText(text), Text: text, Vector: recencyOnly,
			IngestedAt: ingested.Add(time.Duration(index) * time.Hour),
		})
	}

	data, encodeErr := chunk.EncodeRecords(records)
	if encodeErr != nil {
		t.Fatalf("encoding chunks: %v", encodeErr)
	}

	const indexPerm = 0o600

	err := os.WriteFile(filepath.Join(chunksDir, "s1.jsonl"), data, indexPerm)
	if err != nil {
		t.Fatalf("writing the chunk index: %v", err)
	}
}

// vhs1PlantVocabTerm writes a vocab term definition note named name and its sidecar, whose body vector is vector.
func vhs1PlantVocabTerm(t *testing.T, vault, name string, vector []float32) {
	t.Helper()

	content := "---\ntype: fact\ntier: L2\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"5\"\ncreated: 2026-01-01\nsource: prior\ntags:\n    - vocab\n---\n\nInformation learned: ...\n"
	notePath := writeVaultNote(t, vault, name+".md", content)

	sidecar := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "test-model@4", Dims: len(vector),
		SituationVector: vector, BodyVector: vector, ContentHash: embed.ContentHash([]byte(content)),
	}
	writeVaultNote(t, vault, filepath.Base(embed.SidecarPath(notePath)), string(embed.MarshalSidecar(sidecar)))
}

// vhs1ReadIfPresent returns the content of path, or "" when it does not exist.
func vhs1ReadIfPresent(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}

	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(raw)
}

// vhs1RunLocalQuery runs the local query command with args over deps and returns its payload.
func vhs1RunLocalQuery(t *testing.T, deps cli.Deps, args cli.QueryArgs) string {
	t.Helper()

	var out bytes.Buffer

	err := cli.RunQuery(t.Context(), args, cli.ExportNewQueryDeps(deps), &out)
	if err != nil {
		t.Fatalf("local query: %v", err)
	}

	return out.String()
}

// vhs1ServeQueryRoute returns the handler of GET /query over vault and chunksDir.
func vhs1ServeQueryRoute(t *testing.T, deps cli.Deps, vault, chunksDir string) cli.ServeHandler {
	t.Helper()

	return routeFor(t, cli.ServeRoutes(deps, vault, "personal", chunksDir), "/query")
}

// vhs1VaultWithIDFile returns a fresh vault directory whose ID file holds content.
func vhs1VaultWithIDFile(t *testing.T, content string) string {
	t.Helper()

	vault := t.TempDir()
	writeVaultNote(t, vault, ".engram-vault-id", content)

	return vault
}
