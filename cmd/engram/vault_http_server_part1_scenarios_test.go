package main

// Scenario test for the vault-http-server technical component, part 1 (861c2j), that needs the real net/http mux:
// 792b35c1a2. The route table itself is pinned in internal/cli; the 404 and 405 answers are net/http's own, so the
// scenario runs over the production mux (realNewServeMux and realRegisterRoute) with no listener: requests are
// served in memory through the mux's ServeHTTP. Spec: docs/specs/technical/vault-http-server.md.

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792b35c1a2_ServeMux_AnUnroutedPathOrMethodIsRefused verifies 792b35c1a2 (An unrouted path or method is
// refused): on a running server a request that names /amend, /query-chunks or another host-only command is
// answered 404, and a served path with another method (GET to /learn) is answered 405 naming the method it takes;
// no route handler runs and the vault and chunk directory stay empty. A request to a served path and method does
// reach its handler, so the zero count is not an artifact of the counting wrapper.
func Test792b35c1a2_ServeMux_AnUnroutedPathOrMethodIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		method string
		path   string
		status int
		allow  string
	}{
		"POST /amend":          {http.MethodPost, "/amend", http.StatusNotFound, ""},
		"GET /amend":           {http.MethodGet, "/amend", http.StatusNotFound, ""},
		"GET /query-chunks":    {http.MethodGet, "/query-chunks", http.StatusNotFound, ""},
		"POST /query-chunks":   {http.MethodPost, "/query-chunks", http.StatusNotFound, ""},
		"GET /show-chunk":      {http.MethodGet, "/show-chunk", http.StatusNotFound, ""},
		"POST /ingest":         {http.MethodPost, "/ingest", http.StatusNotFound, ""},
		"GET /vocab":           {http.MethodGet, "/vocab", http.StatusNotFound, ""},
		"GET /learn":           {http.MethodGet, "/learn", http.StatusMethodNotAllowed, http.MethodPost},
		"GET /activate":        {http.MethodGet, "/activate", http.StatusMethodNotAllowed, http.MethodPost},
		"POST /query":          {http.MethodPost, "/query", http.StatusMethodNotAllowed, http.MethodGet},
		"DELETE /show":         {http.MethodDelete, "/show", http.StatusMethodNotAllowed, http.MethodGet},
		"POST /learn/extra":    {http.MethodPost, "/learn/extra", http.StatusNotFound, ""},
		"GET /query-chunks/":   {http.MethodGet, "/query-chunks/", http.StatusNotFound, ""},
		"PUT /prune":           {http.MethodPut, "/prune", http.StatusNotFound, ""},
		"GET / (the root)":     {http.MethodGet, "/", http.StatusNotFound, ""},
		"POST /resituate":      {http.MethodPost, "/resituate", http.StatusNotFound, ""},
		"POST /update":         {http.MethodPost, "/update", http.StatusNotFound, ""},
		"POST /show-chunk":     {http.MethodPost, "/show-chunk", http.StatusNotFound, ""},
		"GET /check":           {http.MethodGet, "/check", http.StatusNotFound, ""},
		"POST /vocab-refit":    {http.MethodPost, "/vocab-refit", http.StatusNotFound, ""},
		"POST /activate/extra": {http.MethodPost, "/activate/extra", http.StatusNotFound, ""},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault, chunksDir := t.TempDir(), t.TempDir()

			var handled atomic.Int64

			mux := vhs1CountingMux(t, vault, chunksDir, &handled)

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), testCase.method, testCase.path, nil))

			g.Expect(recorder.Code).To(Equal(testCase.status))

			if testCase.allow == "" {
				g.Expect(recorder.Header().Get("Allow")).To(BeEmpty())
			} else {
				g.Expect(recorder.Header().Get("Allow")).To(ContainSubstring(testCase.allow))
			}

			g.Expect(handled.Load()).To(BeZero(), "no route handler ran")

			for _, dir := range []string{vault, chunksDir} {
				entries, err := os.ReadDir(dir)
				g.Expect(err).NotTo(HaveOccurred())

				if err != nil {
					return
				}

				g.Expect(entries).To(BeEmpty(), "no vault operation touched %s", dir)
			}
		})
	}

	t.Run("a served path and method reaches its handler", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		var handled atomic.Int64

		mux := vhs1CountingMux(t, t.TempDir(), t.TempDir(), &handled)

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/show?note=1", nil))

		g.Expect(handled.Load()).To(Equal(int64(1)))
		g.Expect(strings.TrimSpace(recorder.Body.String())).NotTo(BeEmpty())
	})
}

// vhs1CountingHandler counts how many requests reach the wrapped route handler.
type vhs1CountingHandler struct {
	next    cli.ServeHandler
	counter *atomic.Int64
}

func (h vhs1CountingHandler) Serve(ctx context.Context, req cli.ServeRequest) cli.ServeResponse {
	h.counter.Add(1)

	return h.next.Serve(ctx, req)
}

// vhs1CountingMux registers the production routes over vault, each behind a counting handler, on the real mux
// through the production primitives, as newRealServeMux does for the round-trip tests.
func vhs1CountingMux(t *testing.T, vault, chunksDir string, counter *atomic.Int64) *http.ServeMux {
	t.Helper()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	rawMux := realNewServeMux()
	for _, route := range cli.ServeRoutes(realServeDeps(&stdout, &stderr), vault, "personal", chunksDir) {
		realRegisterRoute(rawMux, route.Method, route.Pattern, vhs1CountingHandler{next: route.Handler, counter: counter})
	}

	realMux, ok := rawMux.(*http.ServeMux)
	g.Expect(ok).To(BeTrue())

	return realMux
}
