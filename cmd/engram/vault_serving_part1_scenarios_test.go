package main

// Scenario tests for the vault-serving product capability, part 1 (861c2zzf), that need the real net/http mux:
// 792a20a1a, 792a20a1b and 792a20a1c. The other scenarios of the part live in
// internal/cli/vault_serving_part1_scenarios_test.go. Spec: docs/specs/product/vault-serving.md.
//
// Each scenario runs the engram serve target itself over the production composition (realServeDeps, so the real
// net/http mux through realNewServeMux and realRegisterRoute and every handler of cli.ServeRoutes): only the
// blocking listen is replaced by a function that sends requests through the mux's ServeHTTP while the server is
// "running", so nothing binds a port. The vaults are the test's own t.TempDir() directories (the recorded
// deviation, #860), the embedder is a deterministic stub, git is a stub that finds nothing and no environment
// variable or home is read.

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a20a1a_Serve_AServedCommandIsReachable verifies 792a20a1a (A served command is reachable): on a running
// engram serve, a request for query, show, activate or learn sent with the method it takes reaches its handler once
// and is answered 200 with the result the equivalent local command gives: the same payload for query and show, the
// same sidecar for activate, and for learn the same note, written as a pending offer.
func Test792a20a1a_Serve_AServedCommandIsReachable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	served, twin := vs1NewRig(t), vs1NewRig(t)
	notePath := served.plant(t)
	twinPath := twin.plant(t)

	const (
		phrase    = "alpha situation"
		queryPath = "/query?phrase=alpha+situation"
	)

	localQuery := served.local(t, "query", "--phrase", phrase, "--vault", served.vault, "--chunks-dir", served.chunks)
	localShow := served.local(t, "show", "1", "--vault", served.vault)
	twin.local(t, "activate", "--note", "1.2026-01-01.alpha", "--vault", twin.vault)
	twin.local(t, "learn", "fact", "--slug", "beta", "--source", "child", "--position", "top",
		"--situation", "beta situation", "--subject", "engram", "--predicate", "serves", "--object", "beta object",
		"--vault", twin.vault)

	var responses []*httptest.ResponseRecorder

	served.serve(t, func(mux *http.ServeMux) {
		responses = []*httptest.ResponseRecorder{
			vs1Do(t, mux, http.MethodGet, queryPath, nil),
			vs1Do(t, mux, http.MethodGet, "/show?note=1", nil),
			vs1Do(t, mux, http.MethodPost, "/activate", []byte(`{"notes":["1.2026-01-01.alpha"]}`)),
			vs1Do(t, mux, http.MethodPost, "/learn", []byte(
				`{"type":"fact","slug":"beta","position":"top","source":"child","situation":"beta situation",`+
					`"subject":"engram","predicate":"serves","object":"beta object","user":"caller@example.com"}`)),
		}
	})

	g.Expect(responses).To(HaveLen(4))

	if len(responses) != 4 {
		return
	}

	for index, route := range []string{"GET /query", "GET /show", "POST /activate", "POST /learn"} {
		g.Expect(responses[index].Code).To(Equal(http.StatusOK), "%s: %s", route, responses[index].Body.String())
		g.Expect(served.counts.of(route)).To(Equal(1), "%s reached its handler once", route)
	}

	g.Expect(localQuery).To(ContainSubstring("alpha"), "the local payload holds the note, so equality is not empty")
	g.Expect(responses[0].Body.String()).To(Equal(localQuery), "the served query is the local query")
	g.Expect(localShow).To(ContainSubstring("alpha object"))
	g.Expect(responses[1].Body.String()).To(Equal(localShow), "the served show is the local show")

	g.Expect(vs1Read(t, embed.SidecarPath(notePath))).To(ContainSubstring(`"last_used":"2026-09-27"`))
	g.Expect(vs1Read(t, embed.SidecarPath(notePath))).To(Equal(vs1Read(t, embed.SidecarPath(twinPath))),
		"the served activate leaves the sidecar the local activate leaves")

	servedNote := vs1ReadOnly(t, served.vault, ".beta.md")
	localNote := vs1ReadOnly(t, twin.vault, ".beta.md")
	g.Expect(filepath.Base(servedNote)).To(Equal(filepath.Base(localNote)), "the same note is written")
	g.Expect(vs1Read(t, servedNote)).To(ContainSubstring("pending: true"), "as a pending offer")
	g.Expect(vs1Body(vs1Read(t, servedNote))).To(Equal(vs1Body(vs1Read(t, localNote))))
}

// Test792a20a1b_Serve_AHostOnlyCommandHasNoRoute verifies 792a20a1b (A host-only command has no route): on a
// running engram serve, a request that names amend, query-chunks, show-chunk, ingest, vocab refit, prune, check,
// update or resituate, by either method, is answered with an error (404), reaches no route handler and leaves the
// vault and the chunk directory exactly as they were, while a request for a served command in the same run does
// reach its handler.
func Test792a20a1b_Serve_AHostOnlyCommandHasNoRoute(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rig := vs1NewRig(t)
	rig.plant(t)

	paths := []string{
		"/amend", "/query-chunks", "/show-chunk", "/ingest", "/vocab/refit", "/vocab-refit", "/vocab", "/prune",
		"/check", "/update", "/resituate",
	}

	var (
		answers []int
		before  map[string]string
		after   map[string]string
		served  int
	)

	rig.serve(t, func(mux *http.ServeMux) {
		before = vs1Snapshot(t, rig.vault, rig.chunks)

		for _, path := range paths {
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				answers = append(answers, vs1Do(t, mux, method, path, []byte(`{"notes":["x"]}`)).Code)
			}
		}

		after = vs1Snapshot(t, rig.vault, rig.chunks)
		served = vs1Do(t, mux, http.MethodGet, "/show?note=1", nil).Code
	})

	g.Expect(answers).To(HaveLen(len(paths) * 2))

	for _, status := range answers {
		g.Expect(status).To(Equal(http.StatusNotFound))
	}

	g.Expect(rig.counts.total()-rig.counts.of("GET /show")).To(BeZero(), "no route handler ran for a host-only command")
	g.Expect(after).To(Equal(before), "no vault operation touched the vault or the chunk directory")
	g.Expect(served).To(Equal(http.StatusOK), "a served command in the same run is answered")
	g.Expect(rig.counts.of("GET /show")).To(Equal(1), "and reaches its handler")
}

// Test792a20a1c_Serve_AServedCommandRefusesTheWrongMethod verifies 792a20a1c (A served command refuses the wrong
// method): on a running engram serve, a GET sent to learn is answered 405 naming POST as the method it takes, and
// so is each other served command sent the method it does not take; no handler runs and no vault file changes.
func Test792a20a1c_Serve_AServedCommandRefusesTheWrongMethod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		method string
		path   string
		allow  string
	}{
		{http.MethodGet, "/learn", http.MethodPost},
		{http.MethodGet, "/activate", http.MethodPost},
		{http.MethodPost, "/query", http.MethodGet},
		{http.MethodPost, "/show", http.MethodGet},
	}

	for _, testCase := range cases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := vs1NewRig(t)
			rig.plant(t)

			var (
				recorded *httptest.ResponseRecorder
				before   map[string]string
				after    map[string]string
			)

			rig.serve(t, func(mux *http.ServeMux) {
				before = vs1Snapshot(t, rig.vault, rig.chunks)
				recorded = vs1Do(t, mux, testCase.method, testCase.path,
					[]byte(`{"type":"fact","slug":"refused","user":"caller@example.com"}`))
				after = vs1Snapshot(t, rig.vault, rig.chunks)
			})

			g.Expect(recorded).NotTo(BeNil())

			if recorded == nil {
				return
			}

			g.Expect(recorded.Code).To(Equal(http.StatusMethodNotAllowed))
			g.Expect(recorded.Header().Get("Allow")).To(ContainSubstring(testCase.allow))
			g.Expect(rig.counts.total()).To(BeZero(), "no route handler ran")
			g.Expect(after).To(Equal(before), "no vault file changed")
		})
	}
}

// unexported constants.
const (
	vs1Addr      = "127.0.0.1:0"
	vs1Dims      = 4
	vs1Model     = "vs1-model@4"
	vs1NoteName  = "1.2026-01-01.alpha.md"
	vs1Object    = "alpha object"
	vs1Scale     = 100
	vs1Situation = "alpha situation"
	vs1UserRW    = 0o600
)

// unexported variables.
var (
	errVs1NoGit   = errors.New("vs1: git finds nothing")
	errVs1NotAMux = errors.New("vs1: the route table is not a net/http mux")
)

// vs1CountingHandler counts the requests that reach the wrapped route handler.
type vs1CountingHandler struct {
	next   cli.ServeHandler
	route  string
	counts *vs1Counts
}

func (h vs1CountingHandler) Serve(ctx context.Context, req cli.ServeRequest) cli.ServeResponse {
	h.counts.add(h.route)

	return h.next.Serve(ctx, req)
}

// vs1Counts counts, per route, how many requests reached the route's handler.
type vs1Counts struct {
	mu      sync.Mutex
	byRoute map[string]int
}

func (c *vs1Counts) add(route string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byRoute[route]++
}

func (c *vs1Counts) of(route string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.byRoute[route]
}

func (c *vs1Counts) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	sum := 0
	for _, count := range c.byRoute {
		sum += count
	}

	return sum
}

// vs1Embedder is a deterministic embedder: the vector sums the text bytes into a few slots.
type vs1Embedder struct{}

func (vs1Embedder) Dims() int { return vs1Dims }

func (vs1Embedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, vs1Dims)

	for index, char := range []byte(text) {
		out[index%vs1Dims] += float32(char) / vs1Scale
	}

	return out, nil
}

func (vs1Embedder) ModelID() string { return vs1Model }

// vs1NoGit is a Commander for which git finds nothing, so no detection reads the machine's repository or config.
type vs1NoGit struct{}

func (vs1NoGit) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
	return nil, nil, errVs1NoGit
}

// vs1Rig is one served node: its production deps with counting handlers, a vault and a chunk directory.
type vs1Rig struct {
	deps   cli.Deps
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	vault  string
	chunks string
	counts *vs1Counts
}

// local runs the local engram command with args and returns what it printed.
func (r vs1Rig) local(t *testing.T, args ...string) string {
	t.Helper()

	r.stdout.Reset()
	r.stderr.Reset()

	_, err := targ.Execute(append([]string{"engram"}, args...), cli.Targets(r.deps)...)
	if err != nil {
		t.Fatalf("engram %v: %v (stderr %s)", args, err, r.stderr.String())
	}

	return r.stdout.String()
}

// plant writes a fact note and its sidecar into the rig's vault and returns the note's path.
func (r vs1Rig) plant(t *testing.T) string {
	t.Helper()

	content := "---\ntype: fact\nsituation: " + vs1Situation + "\nsubject: engram\npredicate: serves\nobject: " +
		vs1Object + "\nluhmann: \"1\"\ncreated: \"2026-01-01\"\nsource: agent\nuser: u\nvault: personal\n---\n\n" +
		"Information learned: when in " + vs1Situation + ", engram serves " + vs1Object + ".\n\n"
	path := filepath.Join(r.vault, vs1NoteName)

	sidecar, buildErr := embed.BuildSidecar(t.Context(), vs1Embedder{}, []byte(content))
	if buildErr != nil {
		t.Fatalf("building the sidecar: %v", buildErr)
	}

	files := map[string][]byte{path: []byte(content), embed.SidecarPath(path): embed.MarshalSidecar(sidecar)}

	for file, data := range files {
		writeErr := os.WriteFile(file, data, vs1UserRW)
		if writeErr != nil {
			t.Fatalf("writing %s: %v", file, writeErr)
		}
	}

	return path
}

// serve runs engram serve over the rig and, once the server is running, hands the real mux to drive. The blocking
// listen is the only replaced capability.
func (r vs1Rig) serve(t *testing.T, drive func(mux *http.ServeMux)) {
	t.Helper()

	driven := false

	deps := r.deps
	register := deps.RegisterRoute

	deps.RegisterRoute = func(mux cli.RawServeMux, method, pattern string, handler cli.ServeHandler) {
		register(mux, method, pattern, vs1CountingHandler{next: handler, route: method + " " + pattern, counts: r.counts})
	}
	deps.ListenAndServe = func(_ context.Context, raw cli.RawServeMux, _ string) error {
		mux, isMux := raw.(*http.ServeMux)
		if !isMux {
			return errVs1NotAMux
		}

		drive(mux)

		driven = true

		return nil
	}

	_, err := targ.Execute([]string{
		"engram", "serve", "--addr", vs1Addr, "--vault", r.vault, "--chunks-dir", r.chunks, "--vault-name", "personal",
	}, cli.Targets(deps)...)
	if err != nil {
		t.Fatalf("engram serve: %v (stderr %s)", err, r.stderr.String())
	}

	if !driven {
		t.Fatalf("the server never ran: %s", r.stderr.String())
	}
}

// vs1Body returns the part of a note after its frontmatter.
func vs1Body(note string) string {
	_, body, _ := strings.Cut(strings.TrimPrefix(note, "---\n"), "\n---\n")

	return body
}

// vs1Do sends one request through the mux and returns what it answered.
func vs1Do(t *testing.T, mux *http.ServeMux, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), method, target, bytes.NewReader(body)))

	return recorder
}

// vs1NewRig builds a served node over fresh directories.
func vs1NewRig(t *testing.T) vs1Rig {
	t.Helper()

	var stdout, stderr bytes.Buffer

	deps := realServeDeps(&stdout, &stderr)
	deps.Getenv = func(string) string { return "" }
	deps.UserHomeDir = func() (string, error) { return t.TempDir(), nil }
	deps.Now = func() time.Time { return time.Date(2026, time.September, 27, 12, 0, 0, 0, time.UTC) }
	deps.Embed = vs1Embedder{}
	deps.Commander = vs1NoGit{}
	deps.Username = func() (string, error) { return "vs1-os-user", nil }

	return vs1Rig{
		deps: deps, stdout: &stdout, stderr: &stderr, vault: t.TempDir(), chunks: t.TempDir(),
		counts: &vs1Counts{byRoute: map[string]int{}},
	}
}

// vs1Read returns the content of path.
func vs1Read(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(raw)
}

// vs1ReadOnly returns the path of the one note of vault whose name ends in suffix.
func vs1ReadOnly(t *testing.T, vault, suffix string) string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(vault, "*"+suffix))
	if err != nil || len(matches) != 1 {
		t.Fatalf("want one note ending in %s in %s, got %v (%v)", suffix, vault, matches, err)
	}

	return matches[0]
}

// vs1Snapshot maps every file under the directories to its content.
func vs1Snapshot(t *testing.T, dirs ...string) map[string]string {
	t.Helper()

	files := map[string]string{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("reading %s: %v", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == ".luhmann.lock" {
				continue
			}

			files[filepath.Join(dir, entry.Name())] = vs1Read(t, filepath.Join(dir, entry.Name()))
		}
	}

	return files
}
