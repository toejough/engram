package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestRunServe_RegistersAllRoutesAndClassifiesShutdownErr covers RunServe's
// own composition (not just ServeRoutes' handler set): every route gets
// registered exactly once via deps.RegisterRoute, and a ListenAndServe
// error is wrapped when ctx was NOT canceled but swallowed to nil when it
// WAS (the context.AfterFunc-triggered-Close/expected-shutdown case
// realListenAndServe's caller must classify, per serve.go's doc comment).
func TestRunServe_RegistersAllRoutesAndClassifiesShutdownErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registered := map[string]bool{}
	deps := newTestDeps(io.Discard, io.Discard)
	deps.NewServeMux = func() cli.RawServeMux { return "fake-mux" }
	deps.RegisterRoute = func(mux cli.RawServeMux, method, pattern string, _ cli.ServeHandler) {
		g.Expect(mux).To(Equal(cli.RawServeMux("fake-mux")))
		registered[method+" "+pattern] = true
	}

	errListenFailed := errors.New("listen failed")

	deps.ListenAndServe = func(_ context.Context, _ cli.RawServeMux, addr string) error {
		g.Expect(addr).To(Equal("127.0.0.1:0"))

		return errListenFailed
	}

	args := cli.ServeArgs{Addr: "127.0.0.1:0", Vault: t.TempDir(), VaultName: "personal", ChunksDir: t.TempDir()}

	// ctx not canceled: a ListenAndServe error is a real failure, wrapped.
	err := cli.RunServe(context.Background(), args, deps)
	g.Expect(err).To(MatchError(ContainSubstring("listen failed")))

	g.Expect(registered).To(Equal(map[string]bool{
		"GET /query": true, "GET /query-chunks": true, "GET /show": true, "GET /show-chunk": true,
		"POST /activate": true, "POST /learn": true, "POST /amend": true,
	}))

	// ctx canceled: the SAME ListenAndServe error is treated as expected
	// shutdown (nil), not a failure.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = cli.RunServe(ctx, args, deps)
	g.Expect(err).NotTo(HaveOccurred())
}

// TestServeActivate_CommitsDirectly covers POST /activate: the sidecar's
// LastUsed is bumped in the same request (design.md Decisions — activate
// never becomes a pending offer, it's not a content mutation).
func TestServeActivate_CommitsDirectly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")
	sidecarPath := embed.SidecarPath(notePath)
	g.Expect(os.WriteFile(
		sidecarPath, embed.MarshalSidecar(embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion}), 0o600,
	)).To(Succeed())

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())

	body, marshalErr := json.Marshal(map[string][]string{"notes": {"1.2026-01-01.a-note.md"}})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	resp := routeFor(t, routes, "/activate").Serve(t.Context(), cli.ServeRequest{Body: body})
	g.Expect(resp.Status).To(Equal(200))

	raw, readErr := os.ReadFile(sidecarPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	sidecar, unmarshalErr := embed.UnmarshalSidecar(raw)
	g.Expect(unmarshalErr).NotTo(HaveOccurred())
	g.Expect(sidecar.LastUsed).To(Equal(time.Now().Format("2006-01-02")))
}

// TestServeAmend_StampsIdentityAndPendingMarker covers POST /amend: same
// identity/pending contract as learn, applied to an existing note.
func TestServeAmend_StampsIdentityAndPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "5.2026-01-01.existing.md")

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())

	args := cli.AmendArgs{Target: "5", Object: "amended-object", Repo: "git@github.com:example/remote.git"}
	body, marshalErr := json.Marshal(args)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	resp := routeFor(t, routes, "/amend").Serve(t.Context(), cli.ServeRequest{
		Header: map[string][]string{cloudflareHeader: {"cloudflare-user@example.com"}},
		Body:   body,
	})

	g.Expect(resp.Status).To(Equal(200))

	raw, readErr := os.ReadFile(filepath.Join(vault, "5.2026-01-01.existing.md"))
	g.Expect(readErr).NotTo(HaveOccurred())

	written := string(raw)
	g.Expect(written).To(ContainSubstring("user: cloudflare-user@example.com"))
	g.Expect(written).To(ContainSubstring("repo: git@github.com:example/remote.git"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(ContainSubstring("object: amended-object"))
}

// TestServeLearn_ConcurrentWithLocalLearn_NoLostUpdate covers tasks.md
// 10.3/ADR-0013: a local `engram learn` writer and a served POST /learn
// writer racing the SAME vault never lose an update or collide on a
// Luhmann id. Both paths ultimately share the identical production lock
// (newLearnDeps(deps).Lock, real flock on .luhmann.lock) since the served
// handler is built from the same deps as the local RunLearn call — this
// proves that sharing holds under concurrency, mirroring
// TestInvariant_K1_ConcurrentLearnNeverCollides but mixing local and
// served writers instead of all-local.
func TestServeLearn_ConcurrentWithLocalLearn_NoLostUpdate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())
	learnRoute := routeFor(t, routes, "/learn")

	const workersPerKind = 5

	total := workersPerKind * 2

	var wg sync.WaitGroup

	wg.Add(total)

	errs := make([]error, total)

	for i := range workersPerKind {
		go func(i int) {
			defer wg.Done()

			var out strings.Builder

			args := cli.LearnArgs{
				Type: "fact", Slug: fmt.Sprintf("local-%d", i), Vault: vault, Position: "top", Source: "race",
				Situation: fmt.Sprintf("local writer %d", i), Subject: "the vault write-lock",
				Predicate: "prevents", Object: "lost updates under concurrency",
			}

			errs[i] = cli.ExportRunLearn(context.Background(), args, cli.ExportNewLearnDeps(deps), &out)
		}(i)
	}

	for i := range workersPerKind {
		go func(i int) {
			defer wg.Done()

			body, marshalErr := json.Marshal(cli.LearnArgs{
				Type: "fact", Slug: fmt.Sprintf("served-%d", i), Position: "top", Source: "race",
				Situation: fmt.Sprintf("served writer %d", i), Subject: "the vault write-lock",
				Predicate: "prevents", Object: "lost updates under concurrency",
			})
			g.Expect(marshalErr).NotTo(HaveOccurred())

			resp := learnRoute.Serve(context.Background(), cli.ServeRequest{
				Header: map[string][]string{cloudflareHeader: {"racer@example.com"}},
				Body:   body,
			})

			if resp.Status != 200 {
				errs[workersPerKind+i] = fmt.Errorf("served learn %d: status %d: %s", i, resp.Status, resp.Body)
			}
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		g.Expect(err).NotTo(HaveOccurred(), "worker %d failed", i)
	}

	entries, readErr := os.ReadDir(vault)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	ids := map[string]bool{}
	notes := 0

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		notes++

		id, ok := cli.ExportExtractLuhmannFromFilename(entry.Name())
		g.Expect(ok).To(BeTrue(), "note %q has no luhmann id", entry.Name())
		ids[id] = true
	}

	g.Expect(notes).To(Equal(total), "expected %d note files on disk, no lost writes", total)
	g.Expect(ids).To(HaveLen(total), "expected %d distinct luhmann ids, no collisions", total)
}

// TestServeLearn_MissingIdentity_Returns401 covers the identity gate: a
// served learn/amend with no Cloudflare Access header is refused before
// any write happens.
func TestServeLearn_MissingIdentity_Returns401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())

	resp := routeFor(t, routes, "/learn").Serve(t.Context(), cli.ServeRequest{Body: []byte(`{}`)})
	g.Expect(resp.Status).To(Equal(401))

	matches, globErr := filepath.Glob(filepath.Join(vault, "*.md"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(matches).To(BeEmpty(), "no note written when identity is missing")
}

// TestServeLearn_StampsCloudflareIdentityAndPendingMarker covers the core
// offer-write contract: the server's own configured Vault wins over
// anything in the request, user: comes from the Cloudflare header (never
// trusted from the body), repo: passes through the client-supplied value
// unchanged, the note lands pending: true, and the response is a
// {status, luhmann} receipt that never carries note content.
func TestServeLearn_StampsCloudflareIdentityAndPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())

	args := cli.LearnArgs{
		Type: "fact", Slug: "served-fact", Position: "top", Source: "remote",
		Situation: "a served write", Subject: "engram", Predicate: "serves", Object: "writes",
		Repo: "git@github.com:example/remote.git",
	}
	body, marshalErr := json.Marshal(args)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	resp := routeFor(t, routes, "/learn").Serve(t.Context(), cli.ServeRequest{
		Header: map[string][]string{cloudflareHeader: {"cloudflare-user@example.com"}},
		Body:   body,
	})

	g.Expect(resp.Status).To(Equal(200))

	var receipt struct {
		Status  string `json:"status"`
		Luhmann string `json:"luhmann"`
	}
	g.Expect(json.Unmarshal(resp.Body, &receipt)).To(Succeed())
	g.Expect(receipt.Status).To(Equal("offer received"))
	g.Expect(receipt.Luhmann).NotTo(BeEmpty())
	g.Expect(string(resp.Body)).NotTo(ContainSubstring("served-fact"), "response never leaks note content")

	matches, globErr := filepath.Glob(filepath.Join(vault, "*.md"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(matches).To(HaveLen(1))

	if len(matches) == 0 {
		return
	}

	raw, readErr := os.ReadFile(matches[0])
	g.Expect(readErr).NotTo(HaveOccurred())

	written := string(raw)
	g.Expect(written).To(ContainSubstring("user: cloudflare-user@example.com"))
	g.Expect(written).To(ContainSubstring("repo: git@github.com:example/remote.git"))
	g.Expect(written).To(ContainSubstring("pending: true"))
	g.Expect(written).To(ContainSubstring("vault: personal"),
		"empty VaultName falls back to the server's configured value")
}

// TestServeQueryChunks_EmptyIndexSucceeds covers GET /query-chunks: an
// empty chunks dir is a valid, successful (not error) response — matches
// RunChunkQuery's own "empty index: emit the empty payload without waking
// the embedder" documented behavior.
func TestServeQueryChunks_EmptyIndexSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, t.TempDir(), "personal", t.TempDir())

	resp := routeFor(t, routes, "/query-chunks").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"phrase": {"anything"}, "limit": {"3"}},
	})

	g.Expect(resp.Status).To(Equal(200))
	g.Expect(string(resp.Body)).To(ContainSubstring("total_chunks: 0"))
}

// TestServeQuery_ExcludesPendingOffersAndSetsModelID covers GET /query:
// a pending-offer note is excluded from results, the payload's model_id
// matches the wired embedder, and pending_offers reflects the marker's
// presence.
func TestServeQuery_ExcludesPendingOffersAndSetsModelID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	deps := newTestDeps(io.Discard, io.Discard)
	deps.Embed = stubEmbedder{modelID: "test-model@4", dims: 4}

	// Write a normal note locally, then a pending offer through the served
	// path — both auto-embed via the wired stub embedder.
	learnLocal(t, deps, vault, cli.LearnArgs{
		Type: "fact", Slug: "normal-note", Position: "top", Source: "local",
		Situation: "a normal note about coverage", Subject: "engram", Predicate: "covers", Object: "queries",
	})

	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())
	learnBody, marshalErr := json.Marshal(cli.LearnArgs{
		Type: "fact", Slug: "pending-note", Position: "top", Source: "remote",
		Situation: "a pending note about coverage", Subject: "engram", Predicate: "covers", Object: "offers",
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	learnResp := routeFor(t, routes, "/learn").Serve(t.Context(), cli.ServeRequest{
		Header: map[string][]string{cloudflareHeader: {"u@example.com"}},
		Body:   learnBody,
	})
	g.Expect(learnResp.Status).To(Equal(200))

	resp := routeFor(t, routes, "/query").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"phrase": {"coverage"}},
	})

	g.Expect(resp.Status).To(Equal(200))
	payload := string(resp.Body)
	g.Expect(payload).To(ContainSubstring("model_id: test-model@4"))
	g.Expect(payload).To(ContainSubstring("pending_offers: true"))
	g.Expect(payload).To(ContainSubstring("normal-note"))
	g.Expect(payload).NotTo(ContainSubstring("pending-note"))
}

// TestServeRoutes_MethodsAndPatterns covers the API contract's route table:
// the four read routes are GET, the three write routes are POST, and no
// host-only command (ingest/vocab refit/prune/check/update/resituate) gets
// a route.
func TestServeRoutes_MethodsAndPatterns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, t.TempDir(), "personal", t.TempDir())

	got := map[string]string{}
	for _, route := range routes {
		got[route.Pattern] = route.Method
	}

	g.Expect(got).To(Equal(map[string]string{
		"/query":        "GET",
		"/query-chunks": "GET",
		"/show":         "GET",
		"/show-chunk":   "GET",
		"/activate":     "POST",
		"/learn":        "POST",
		"/amend":        "POST",
	}))
}

// TestServeShowChunk_NotFoundReturnsError covers GET /show-chunk: an
// unmatched chunk id surfaces as a 500 JSON error body, not a panic or a
// silently-empty success.
func TestServeShowChunk_NotFoundReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, t.TempDir(), "personal", t.TempDir())

	resp := routeFor(t, routes, "/show-chunk").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"id": {"missing#anchor"}},
	})

	g.Expect(resp.Status).To(Equal(500))

	var errBody struct {
		Error string `json:"error"`
	}
	g.Expect(json.Unmarshal(resp.Body, &errBody)).To(Succeed())
	g.Expect(errBody.Error).NotTo(BeEmpty())
}

// TestServeShow_MatchesLocalOutput covers GET /show: response matches
// local `engram show`'s own output shape (a read-only route, no identity
// or offer logic involved).
func TestServeShow_MatchesLocalOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	deps := newTestDeps(io.Discard, io.Discard)
	routes := cli.ServeRoutes(deps, vault, "personal", t.TempDir())

	resp := routeFor(t, routes, "/show").Serve(t.Context(), cli.ServeRequest{
		Query: map[string][]string{"note": {"1"}},
	})

	g.Expect(resp.Status).To(Equal(200))
	g.Expect(string(resp.Body)).To(ContainSubstring("subject: a"))
}

// unexported constants.
const (
	cloudflareHeader = "Cf-Access-Authenticated-User-Email"
)

// learnLocal writes a note directly (not through the served offer path)
// for query-test fixtures — same shape a local `engram learn` produces.
func learnLocal(t *testing.T, deps cli.Deps, vault string, args cli.LearnArgs) {
	t.Helper()
	g := NewWithT(t)

	args.Vault = vault
	args.VaultName = "personal"

	err := cli.ExportRunLearn(context.Background(), args, cli.ExportNewLearnDeps(deps), io.Discard)
	g.Expect(err).NotTo(HaveOccurred())
}

// routeFor returns the handler registered for pattern, failing the test
// immediately if none matches — never returns a nil ServeHandler (revive
// sees t.Fatalf as terminating and flags the trailing panic as dead code;
// nilaway does not make that same assumption, so the panic stays to keep
// this function's return type provably non-nil for it).
func routeFor(t *testing.T, routes []cli.ServeRoute, pattern string) cli.ServeHandler {
	t.Helper()

	for _, route := range routes {
		if route.Pattern == pattern {
			return route.Handler
		}
	}

	t.Fatalf("routeFor: no route registered for pattern %q", pattern) //nolint:revive // see doc comment

	panic("unreachable")
}

// writeServeVaultFile writes normalFactNote to name under vault, returning
// the full path.
func writeServeVaultFile(t *testing.T, vault, name string) string {
	t.Helper()
	g := NewWithT(t)

	path := filepath.Join(vault, name)
	g.Expect(os.WriteFile(path, []byte(normalFactNote), 0o600)).To(Succeed())

	return path
}
