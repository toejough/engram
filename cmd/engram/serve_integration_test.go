package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestServeAndFetch_LearnRoundTrip drives POST /learn over a real TCP
// connection with the Cloudflare Access header set (realFetch carries no
// header parameter by design — the client never sets this header in
// production, Cloudflare Access injects it at the edge — so this test
// builds the request directly with net/http to exercise the server's
// header-reading path end to end).
func TestServeAndFetch_LearnRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	var stdout, stderr bytes.Buffer

	server := httptest.NewServer(newRealServeMux(t, realServeDeps(&stdout, &stderr), vault, t.TempDir()))
	defer server.Close()

	body, marshalErr := json.Marshal(cli.LearnArgs{
		Type: "fact", Slug: "round-trip", Position: "top", Source: "test",
		Situation: "a round trip", Subject: "engram", Predicate: "serves", Object: "http",
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	req, reqErr := http.NewRequestWithContext(
		context.Background(), http.MethodPost, server.URL+"/learn", bytes.NewReader(body),
	)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if req == nil {
		return
	}

	req.Header.Set("Cf-Access-Authenticated-User-Email", "roundtrip@example.com")

	resp, doErr := http.DefaultClient.Do(req)
	g.Expect(doErr).NotTo(HaveOccurred())

	if resp == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var receipt struct {
		Status  string `json:"status"`
		Luhmann string `json:"luhmann"`
	}
	g.Expect(json.NewDecoder(resp.Body).Decode(&receipt)).To(Succeed())
	g.Expect(receipt.Status).To(Equal("offer received"))

	matches, globErr := filepath.Glob(filepath.Join(vault, "*.md"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(matches).To(HaveLen(1))

	if len(matches) == 0 {
		return
	}

	raw, readErr := os.ReadFile(matches[0])
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).To(ContainSubstring("user: roundtrip@example.com"))
	g.Expect(string(raw)).To(ContainSubstring("pending: true"))
}

// TestServeAndFetch_ShowRoundTrip drives GET /show through a real
// net/http.ServeMux (realNewServeMux/realRegisterRoute) served by
// httptest.Server, fetched via realFetch — the same primitives `engram
// serve`/ENGRAM_SERVER wire in production, exercised over an actual TCP
// connection rather than direct handler calls.
func TestServeAndFetch_ShowRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-01-01.a-note.md"), []byte(roundTripNote), 0o600)).To(Succeed())

	var stdout, stderr bytes.Buffer

	server := httptest.NewServer(newRealServeMux(t, realServeDeps(&stdout, &stderr), vault, t.TempDir()))
	defer server.Close()

	resp, err := realFetch(context.Background(), "GET", server.URL+"/show?note=1", nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resp.Status).To(Equal(http.StatusOK))
	g.Expect(string(resp.Body)).To(ContainSubstring("subject: a"))
}

// unexported constants.
const (
	roundTripNote = "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"1\"\ncreated: 2026-01-01\nsource: agent\nuser: u\nvault: personal\n---\n\n" +
		"Information learned: when in s, a b c.\n\n"
)

// newRealServeMux registers vault's served routes onto a real
// net/http.ServeMux via the production primitives (realNewServeMux/
// realRegisterRoute) — the same path `engram serve` itself uses, just
// without the blocking ListenAndServe call.
func newRealServeMux(t *testing.T, deps cli.Deps, vault, chunksDir string) *http.ServeMux {
	t.Helper()
	g := NewWithT(t)

	rawMux := realNewServeMux()
	for _, route := range cli.ServeRoutes(deps, vault, "personal", chunksDir) {
		realRegisterRoute(rawMux, route.Method, route.Pattern, route.Handler)
	}

	realMux, ok := rawMux.(*http.ServeMux)
	g.Expect(ok).To(BeTrue())

	return realMux
}

// realServeDeps composes production cli.Deps exactly like main() does,
// with test-controlled stdout/stderr and a no-op exit — used to build a
// real HTTP server over ServeRoutes for round-trip verification.
func realServeDeps(stdout, stderr *bytes.Buffer) cli.Deps {
	return cli.NewDeps(cli.Primitives{
		FS:    fsPrimitives(),
		Lock:  lockPrimitives(),
		Exec:  execPrimitives(),
		Spawn: spawnPrimitives(),
		Proc:  procPrimitives(),
		HTTP:  httpPrimitives(),
	}, stdout, stderr, func(int) {})
}
