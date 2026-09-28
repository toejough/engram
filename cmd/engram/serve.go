// Thin net/http capability wrappers for `engram serve` and the
// ENGRAM_PARENT parent-sync client. This file is the only place in the repo
// (outside its own tests) that imports net/http — mirrors hugot.go's "only place that
// imports hugot" convention. All routing, identity resolution, and
// offer-write logic lives in internal/cli (#700); every declaration here
// stays a single-call/simple-error-wrapper body (targ check-thin-api) — no
// loops, so route registration happens one call at a time via RegisterRoute
// (internal/cli's RunServe does the per-route looping, not this file).
// (Blank line below keeps this file comment from doubling as a second
// package godoc — main.go owns the package comment.)

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/toejough/engram/internal/cli"
)

// unexported variables.
var (
	errFetchNilResponse = errors.New("fetch: nil response with no error")
	// fetchClientTimeout bounds one parent-sync (ENGRAM_PARENT) request in
	// total. A var, not a computed const: thin-api requires cmd/engram
	// consts to be literals or re-exports.
	fetchClientTimeout = 30 * time.Second //nolint:gochecknoglobals // real net/http config
	// fetchConnectDialer dials parent connections with fetchConnectTimeout.
	//nolint:gochecknoglobals // shared dialer, real net config
	fetchConnectDialer = &net.Dialer{Timeout: fetchConnectTimeout}
	// fetchConnectTimeout bounds connecting to the parent (design D6, M9):
	// an unreachable host costs at most 3s, once per backoff window.
	fetchConnectTimeout = 3 * time.Second //nolint:gochecknoglobals // real net/http config
	// fetchHTTPClient is the client parent-sync requests fetch through; a
	// bounded total timeout beats http.DefaultClient's unbounded one, and
	// the dialer bounds the connect step.
	//nolint:gochecknoglobals // shared client, real net/http state
	fetchHTTPClient = &http.Client{Timeout: fetchClientTimeout, Transport: newFetchTransport()}
	// readHeaderTimeout bounds how long the server waits to read a
	// request's headers (mitigates slow-header/slowloris-style stalls).
	// Same var-not-const reasoning as fetchClientTimeout above.
	readHeaderTimeout = 10 * time.Second //nolint:gochecknoglobals // real net/http config
)

// bodyReadErrIsTooLarge reports whether err is specifically
// http.MaxBytesReader's over-the-cap error (S34 residual) — the one case
// httpHandlerFor's caller (internal/cli's requireReadBody) answers 413
// for. Any other body-read failure (a client disconnect, a reset
// connection) is not this case, and answers 400 instead.
func bodyReadErrIsTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError

	return errors.As(err, &maxBytesErr)
}

// httpHandlerFor adapts one cli.ServeHandler to a real net/http.HandlerFunc.
// The body is read through http.MaxBytesReader (cli.MaxServeRequestBytes);
// a read error rides on the request as BodyErr, classified via BodyTooLarge
// (S34 residual) so the routes answer 413 only when the body itself
// exceeded the cap, and 400 for any other read failure (final review F7).
// Query/Header are assigned straight from the real request's own map-typed
// fields (net/url.Values / net/http.Header both have underlying type
// map[string][]string, so they're directly assignable to cli.ServeRequest's
// fields — no conversion or per-entry copy needed).
func httpHandlerFor(handler cli.ServeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, cli.MaxServeRequestBytes))
		reqURL := r.URL

		resp := handler.Serve(r.Context(), cli.ServeRequest{
			Query:        reqURL.Query(),
			Header:       r.Header,
			Body:         body,
			BodyErr:      readErr,
			BodyTooLarge: bodyReadErrIsTooLarge(readErr),
		})

		w.WriteHeader(resp.Status)
		_, _ = w.Write(resp.Body)
	}
}

// httpPrimitives groups the raw HTTP-listen and HTTP-fetch capabilities.
func httpPrimitives() cli.HTTPPrims {
	return cli.HTTPPrims{
		NewServeMux:    realNewServeMux,
		RegisterRoute:  realRegisterRoute,
		ListenAndServe: realListenAndServe,
		Fetch:          realFetch,
	}
}

// newFetchTransport clones http.DefaultTransport — keeping its proxy,
// TLS-handshake and idle-connection defaults — and bounds the connect step
// with fetchConnectDialer.
func newFetchTransport() *http.Transport {
	//nolint:forcetypeassert // http.DefaultTransport is always an *http.Transport
	defaults := http.DefaultTransport.(*http.Transport)
	transport := defaults.Clone()
	transport.DialContext = fetchConnectDialer.DialContext

	return transport
}

// readFetchResponse reduces a non-nil *http.Response to cli.FetchResponse.
// Split out of realFetch so the nil check that satisfies nilaway
// (http.Client.Do's resp-is-nil-on-error contract isn't proven by static
// analysis) stays a single plain != nil guard.
func readFetchResponse(resp *http.Response) (cli.FetchResponse, error) {
	if resp != nil {
		return readNonNilResponse(resp)
	}

	return cli.FetchResponse{}, errFetchNilResponse
}

// readNonNilResponse reads and closes resp's body, reducing it to
// cli.FetchResponse. It reads at most cli.FetchResponseReadLimit bytes;
// internal/cli rejects a body over the accepted maximum (final review F7). No defer (targ check-thin-api forbids defer
// statements in cmd/engram's declarations): body is closed unconditionally,
// right after the read, sequentially.
func readNonNilResponse(resp *http.Response) (cli.FetchResponse, error) {
	body := resp.Body
	respBody, readErr := io.ReadAll(io.LimitReader(body, cli.FetchResponseReadLimit))
	closeErr := body.Close()

	if readErr != nil {
		return cli.FetchResponse{}, fmt.Errorf("fetch: reading response: %w", readErr)
	}

	if closeErr != nil {
		return cli.FetchResponse{}, fmt.Errorf("fetch: closing response: %w", closeErr)
	}

	return cli.FetchResponse{Status: resp.StatusCode, Body: respBody}, nil
}

// realFetch implements cli.HTTPPrims.Fetch: issues one net/http request
// against the fully-formed url (query encoding already applied by
// internal/cli — depguard #700's internal-purity rule bans net/url there)
// and reduces the response to cli.FetchResponse.
func realFetch(ctx context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
	req, reqErr := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if reqErr != nil {
		return cli.FetchResponse{}, fmt.Errorf("fetch: building request: %w", reqErr)
	}

	//nolint:bodyclose // closed in readNonNilResponse, one call away — static analysis can't see across it
	resp, doErr := fetchHTTPClient.Do(req)
	if doErr != nil {
		return cli.FetchResponse{}, fmt.Errorf("fetch: %w", doErr)
	}

	return readFetchResponse(resp)
}

// realListenAndServe implements cli.HTTPPrims.ListenAndServe: blocks
// serving mux on addr until ctx is canceled or the listener fails
// irrecoverably. context.AfterFunc — not a goroutine — closes the server on
// cancellation: targ check-thin-api requires a go statement's call to
// qualify a plain identifier (pkg.F or recv.M), which a same-package
// closure-launching helper can't satisfy, but context.AfterFunc(ctx, ...)
// itself does (it's pkg.F).
func realListenAndServe(ctx context.Context, mux cli.RawServeMux, addr string) error {
	//nolint:forcetypeassert // production invariant: mux always comes from realNewServeMux
	realMux := mux.(*http.ServeMux)
	server := &http.Server{Addr: addr, Handler: realMux, ReadHeaderTimeout: readHeaderTimeout}

	// context.AfterFunc closes server once ctx is canceled, which makes
	// ListenAndServe return http.ErrServerClosed below. RunServe
	// (internal/cli) checks ctx.Err() to tell that expected-shutdown case
	// apart from a real listen failure — that classification needs a
	// branch targ check-thin-api forbids here.
	context.AfterFunc(ctx, func() { _ = server.Close() })

	err := server.ListenAndServe()
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	return nil
}

// realNewServeMux implements cli.HTTPPrims.NewServeMux.
func realNewServeMux() cli.RawServeMux {
	return http.NewServeMux()
}

// realRegisterRoute implements cli.HTTPPrims.RegisterRoute: one
// mux.HandleFunc call per invocation (Go 1.22+ method-pattern syntax,
// "GET /query" etc). RunServe (internal/cli) calls this once per route —
// the loop lives there, never here.
func realRegisterRoute(mux cli.RawServeMux, method, pattern string, handler cli.ServeHandler) {
	//nolint:forcetypeassert // production invariant: mux always comes from realNewServeMux
	realMux := mux.(*http.ServeMux)
	realMux.HandleFunc(method+" "+pattern, httpHandlerFor(handler))
}
