package main

// Scenario tests for the vault-serving product capability, part 3 (861c2zzh), that need the real net/http edge:
// 792a20n1a and 792a20n1b. The other scenarios of the part live in
// internal/cli/vault_serving_part3_scenarios_test.go. Spec: docs/specs/product/vault-serving.md.
//
// The edge reads a request body through http.MaxBytesReader and hands the routes a BodyErr and BodyTooLarge flag
// (httpHandlerFor), so each scenario runs the engram serve target itself over the production composition (the
// shared vs1Rig: the real net/http mux through realNewServeMux and realRegisterRoute and every handler of
// cli.ServeRoutes), with requests sent through the mux's ServeHTTP while the server is "running", so nothing binds
// a port. The vaults are the test's own t.TempDir() directories (the recorded deviation, #860), the embedder is a
// deterministic stub, git is a stub that finds nothing and no environment variable or home is read.

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// Test792a20n1a_Serve_AnOversizedRequestIsRefused verifies 792a20n1a (An oversized request is refused): on every
// route a request whose body exceeds 4 MiB (the body is a well-formed request padded with white space, so a route
// that read it all would act on it) is answered 413 with an error body and leaves the vault unchanged; a body of
// exactly 4 MiB is read, so the size alone decides.
func Test792a20n1a_Serve_AnOversizedRequestIsRefused(t *testing.T) {
	t.Parallel()

	for _, route := range vs3Routes() {
		t.Run(route.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := vs1NewRig(t)
			notePath := rig.plant(t)

			var (
				recorded *httptest.ResponseRecorder
				before   map[string]string
				after    map[string]string
			)

			rig.serve(t, func(mux *http.ServeMux) {
				before = vs1Snapshot(t, rig.vault, rig.chunks)
				recorded = vs3Do(t, mux, route, bytes.NewReader(vs3Padded(route.body, cli.MaxServeRequestBytes+1)))
				after = vs1Snapshot(t, rig.vault, rig.chunks)
			})

			g.Expect(recorded).NotTo(BeNil())

			if recorded == nil {
				return
			}

			g.Expect(recorded.Code).To(Equal(http.StatusRequestEntityTooLarge), recorded.Body.String())
			g.Expect(recorded.Body.String()).To(ContainSubstring(`"error"`))
			g.Expect(after).To(Equal(before), "the vault is unchanged")
			g.Expect(vs1Read(t, embed.SidecarPath(notePath))).NotTo(ContainSubstring("last_used"))
		})
	}

	t.Run("a body of exactly 4 MiB is read", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := vs1NewRig(t)
		notePath := rig.plant(t)
		route := vs3Routes()[1]

		var recorded *httptest.ResponseRecorder

		rig.serve(t, func(mux *http.ServeMux) {
			recorded = vs3Do(t, mux, route, bytes.NewReader(vs3Padded(route.body, cli.MaxServeRequestBytes)))
		})

		g.Expect(recorded).NotTo(BeNil())

		if recorded == nil {
			return
		}

		g.Expect(recorded.Code).To(Equal(http.StatusOK), recorded.Body.String())
		g.Expect(vs1Read(t, embed.SidecarPath(notePath))).To(ContainSubstring(`"last_used":"2026-09-27"`))
	})
}

// Test792a20n1b_Serve_AnotherUnreadableBodyIsABadRequest verifies 792a20n1b (Another unreadable body is a bad
// request): on every route a request whose body cannot be read in full for a reason other than size, here a client
// that disconnects after sending a well-formed body but before the end of the stream, is answered 400 (never 413)
// and nothing is written.
func Test792a20n1b_Serve_AnotherUnreadableBodyIsABadRequest(t *testing.T) {
	t.Parallel()

	for _, route := range vs3Routes() {
		t.Run(route.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := vs1NewRig(t)
			notePath := rig.plant(t)

			var (
				recorded *httptest.ResponseRecorder
				before   map[string]string
				after    map[string]string
			)

			rig.serve(t, func(mux *http.ServeMux) {
				before = vs1Snapshot(t, rig.vault, rig.chunks)
				recorded = vs3Do(t, mux, route, io.MultiReader(bytes.NewReader([]byte(route.body)), vs3Disconnect{}))
				after = vs1Snapshot(t, rig.vault, rig.chunks)
			})

			g.Expect(recorded).NotTo(BeNil())

			if recorded == nil {
				return
			}

			g.Expect(recorded.Code).To(Equal(http.StatusBadRequest), recorded.Body.String())
			g.Expect(recorded.Body.String()).To(ContainSubstring(`"error"`))
			g.Expect(after).To(Equal(before), "nothing is written")
			g.Expect(vs1Read(t, embed.SidecarPath(notePath))).NotTo(ContainSubstring("last_used"))
		})
	}
}

// unexported variables.
var (
	errVs3Disconnect = errors.New("vs3: the client disconnected")
)

// vs3Disconnect is the end of a body stream that fails instead of ending.
type vs3Disconnect struct{}

func (vs3Disconnect) Read([]byte) (int, error) {
	return 0, errVs3Disconnect
}

// vs3Route is one served route with a well-formed request body for it.
type vs3Route struct {
	name   string
	method string
	target string
	body   string
}

// vs3Do sends one request with the body through the mux and returns what it answered.
func vs3Do(t *testing.T, mux *http.ServeMux, route vs3Route, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), route.method, route.target, body))

	return recorder
}

// vs3Padded is body followed by white space up to size bytes: still well-formed JSON for a route that parses it.
func vs3Padded(body string, size int) []byte {
	padded := make([]byte, size)
	copy(padded, body)

	for index := len(body); index < size; index++ {
		padded[index] = ' '
	}

	return padded
}

// vs3Routes are the four served routes with a request that would change the vault (learn, activate) or answer 200
// (query, show) if its body were read.
func vs3Routes() []vs3Route {
	return []vs3Route{
		{
			name: "POST /learn", method: http.MethodPost, target: "/learn",
			body: `{"type":"fact","slug":"padded","position":"top","source":"child","situation":"padded situation",` +
				`"subject":"engram","predicate":"serves","object":"padded object","user":"caller@example.com"}`,
		},
		{name: "POST /activate", method: http.MethodPost, target: "/activate", body: `{"notes":["` + vs1NoteName + `"]}`},
		{name: "GET /query", method: http.MethodGet, target: "/query?phrase=alpha+situation", body: ""},
		{name: "GET /show", method: http.MethodGet, target: "/show?note=1", body: ""},
	}
}
