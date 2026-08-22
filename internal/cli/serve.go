package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// FetchResponse is one HTTP response reduced to primitive types.
type FetchResponse struct {
	Status int
	Body   []byte
}

// RawServeMux is an opaque handle to the real net/http.ServeMux, held by
// internal/cli only to pass back into Deps.RegisterRoute/ListenAndServe —
// never inspected or called directly here (internal/ may not import
// net/http, depguard #700's internal-purity rule). Mirrors
// embed.RawSession's erasure pattern.
type RawServeMux any

// ServeArgs holds parsed flags for `engram serve`.
type ServeArgs struct {
	// Addr is the bind address (e.g. "127.0.0.1:8420"). Required — no env
	// fallback, no default — so `engram serve` refuses to start rather than
	// silently binding 0.0.0.0 (tasks.md 3.1).
	Addr      string `targ:"flag,name=addr,required,desc=bind address e.g. 127.0.0.1:8420 (required -- never defaults to 0.0.0.0)"` //nolint:lll // single unbreakable struct-tag string
	Vault     string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	VaultName string `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on served notes' vault: field (default \"personal\")"` //nolint:lll // single unbreakable struct-tag string
	ChunksDir string `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
}

// ServeHandler answers one ServeRequest. An interface (http.Handler's own
// idiom), not a bare func type: targ check-thin-api requires every call
// head in cmd/engram to qualify a plain identifier (pkg.F or recv.M) —
// invoking a bare func-typed parameter directly doesn't qualify, but
// handler.Serve(...) does.
type ServeHandler interface {
	Serve(ctx context.Context, req ServeRequest) ServeResponse
}

// ServeRequest is one HTTP request reduced to primitive types — the
// boundary cmd/engram's real net/http.ServeMux translates against.
// Query/Header are assigned directly from the real net/http.Request's own
// map-typed fields (net/url.Values / net/http.Header structurally satisfy
// map[string][]string) — never converted or range-copied in cmd/engram.
type ServeRequest struct {
	// Query holds parsed URL query values (repeatable params keep every
	// value, e.g. repeated ?phrase=...). GET routes only.
	Query map[string][]string
	// Header is the request's headers, keyed by canonical MIME header name
	// (e.g. "Cf-Access-Authenticated-User-Email") with every value for that
	// key — the same shape net/http.Request.Header already has.
	Header map[string][]string
	// Body is the raw request body. POST routes only.
	Body []byte
}

// ServeResponse is one HTTP response reduced to primitive types.
type ServeResponse struct {
	Status int
	Body   []byte
}

// ServeRoute pairs one HTTP method+pattern with its handler. RunServe
// registers each via Deps.RegisterRoute (one call per route — the loop
// lives here, in internal/cli, never in cmd/engram).
type ServeRoute struct {
	Method  string
	Pattern string
	Handler ServeHandler
}

// RunServe starts the vault-serve-api HTTP server, blocking until ctx is
// canceled or an unrecoverable listen error occurs. args.Vault/VaultName/
// ChunksDir must already be resolved by the caller (targets.go), same as
// every other command — resolved once at startup, never from a served
// request. Registers each route individually via deps.RegisterRoute — the
// per-route loop is here (internal/cli), never in cmd/engram, which
// targ check-thin-api forbids from containing loops.
func RunServe(ctx context.Context, args ServeArgs, deps Deps) error {
	mux := deps.NewServeMux()

	for _, route := range ServeRoutes(deps, args.Vault, args.VaultName, args.ChunksDir) {
		deps.RegisterRoute(mux, route.Method, route.Pattern, route.Handler)
	}

	err := deps.ListenAndServe(ctx, mux, args.Addr)
	// ctx already canceled means this error is realListenAndServe's own
	// context.AfterFunc-triggered Close (expected shutdown, e.g. Ctrl-C) —
	// not a real listen failure. Checked here, not in cmd/engram, since
	// classifying "expected vs real" needs a branch targ check-thin-api
	// forbids in cmd/engram's declarations.
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

// ServeRoutes composes the served command set's HTTP routes (vault-serve-
// api): the four read-only GET routes and the three write POST routes.
// Every handler calls the existing Run* function for its command directly
// — no reimplementation of command logic here (tasks.md 2.2) — sharing the
// CLI's existing vault locks (ADR-0013). vault/vaultName/chunksDir are
// resolved once by RunServe and baked into every handler closure; a served
// request can never redirect the server at a different vault path.
func ServeRoutes(deps Deps, vault, vaultName, chunksDir string) []ServeRoute {
	return []ServeRoute{
		{Method: methodGet, Pattern: "/query", Handler: serveQuery(deps, vault, chunksDir)},
		{Method: methodGet, Pattern: "/query-chunks", Handler: serveQueryChunks(deps, chunksDir)},
		{Method: methodGet, Pattern: "/show", Handler: serveShow(deps, vault)},
		{Method: methodGet, Pattern: "/show-chunk", Handler: serveShowChunk(deps, chunksDir)},
		{Method: methodPost, Pattern: "/activate", Handler: serveActivate(deps, vault)},
		{Method: methodPost, Pattern: "/learn", Handler: serveLearn(deps, vault, vaultName)},
		{Method: methodPost, Pattern: "/amend", Handler: serveAmend(deps, vault, vaultName, chunksDir)},
	}
}

// unexported constants.
const (
	// cloudflareIdentityHeader is the header Cloudflare Access injects after
	// edge SSO validates the caller's session (design.md API Contract).
	// Trusted directly in v1 — no JWT verification (design.md Non-Goals).
	cloudflareIdentityHeader  = "Cf-Access-Authenticated-User-Email"
	methodGet                 = "GET"
	methodPost                = "POST"
	offerReceivedStatus       = "offer received"
	okStatus                  = "ok"
	statusBadRequest          = 400
	statusInternalServerError = 500
	statusOK                  = 200
	statusUnauthorized        = 401
)

// unexported variables.
var (
	errServeMissingIdentity = errors.New(
		"serve: missing " + cloudflareIdentityHeader + " header — request did not come through Cloudflare Access",
	)
)

// activateRequest is the JSON body shape for POST /activate.
type activateRequest struct {
	Notes []string `json:"notes"`
}

// errResponse is the JSON response body for any served-route failure.
type errResponse struct {
	Error string `json:"error"`
}

// offerReceipt is the JSON response body for a served learn/amend that
// created a pending offer — deliberately never the note content (design.md
// API Contract): fire-and-forget, the offering caller never learns the
// curation outcome.
type offerReceipt struct {
	Status  string `json:"status"`
	Luhmann string `json:"luhmann"`
}

// okResponse is the JSON response body for a served activate that
// succeeded.
type okResponse struct {
	Status string `json:"status"`
}

// serveHandlerFunc adapts a plain function to ServeHandler, mirroring
// net/http.HandlerFunc.
type serveHandlerFunc func(ctx context.Context, req ServeRequest) ServeResponse

// Serve calls f.
func (f serveHandlerFunc) Serve(ctx context.Context, req ServeRequest) ServeResponse {
	return f(ctx, req)
}

// boolQueryParam parses key's first query value as a bool (strconv.ParseBool);
// absent or unparseable values report false.
func boolQueryParam(query map[string][]string, key string) bool {
	value, _ := strconv.ParseBool(firstQueryParam(query, key))

	return value
}

// firstHeaderValue returns key's first header value, or "" when absent.
func firstHeaderValue(header map[string][]string, key string) string {
	values := header[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

// firstQueryParam returns key's first query value, or "" when absent.
func firstQueryParam(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

// intQueryParam parses key's first query value as an int (strconv.Atoi);
// absent or unparseable values report 0.
func intQueryParam(query map[string][]string, key string) int {
	n, _ := strconv.Atoi(firstQueryParam(query, key))

	return n
}

// jsonErrorResponse marshals err into an errResponse body under status.
func jsonErrorResponse(status int, err error) ServeResponse {
	//nolint:errchkjson // a plain string field never fails to encode
	body, _ := json.Marshal(errResponse{Error: err.Error()})

	return ServeResponse{Status: status, Body: body}
}

// jsonOKResponse marshals a plain okResponse body under 200.
func jsonOKResponse() ServeResponse {
	body, _ := json.Marshal(okResponse{Status: okStatus}) //nolint:errchkjson // a plain string field never fails to encode

	return ServeResponse{Status: statusOK, Body: body}
}

// luhmannFromNotePath extracts the leading Luhmann-ID segment from a note
// path printed by RunLearn/RunAmend (learnPath's "<luhmann>.<date>.<slug>.md"
// convention — the ID is always the first dot-separated component).
func luhmannFromNotePath(path string) string {
	base := filepath.Base(path)

	id, _, found := strings.Cut(base, ".")
	if !found {
		return base
	}

	return id
}

// offerReceiptResponse builds the {status, luhmann} response for a served
// learn/amend that created a pending offer (design.md API Contract) —
// deliberately never the note content, extracting only the Luhmann ID from
// the path RunLearn/RunAmend printed to their captured stdout buffer.
func offerReceiptResponse(printedPath []byte) ServeResponse {
	path := strings.TrimSpace(string(printedPath))

	//nolint:errchkjson // plain string fields never fail to encode
	body, _ := json.Marshal(offerReceipt{
		Status:  offerReceivedStatus,
		Luhmann: luhmannFromNotePath(path),
	})

	return ServeResponse{Status: statusOK, Body: body}
}

// serveActivate handles POST /activate: commits directly, no pending-offer
// marker, no curation (design.md Decisions — activate never mutates note
// content, so there's no new claim for curation to judge).
func serveActivate(deps Deps, vault string) ServeHandler {
	return serveHandlerFunc(func(_ context.Context, req ServeRequest) ServeResponse {
		var body activateRequest

		unmarshalErr := json.Unmarshal(req.Body, &body)
		if unmarshalErr != nil {
			return jsonErrorResponse(statusBadRequest, unmarshalErr)
		}

		args := ActivateArgs{Vault: vault, Notes: body.Notes}

		runErr := RunActivate(args, newActivateDeps(deps))
		if runErr != nil {
			return jsonErrorResponse(statusInternalServerError, runErr)
		}

		return jsonOKResponse()
	})
}

// serveAmend handles POST /amend: an AmendArgs-shaped JSON body. Same
// identity/repo/vaultName handling as serveLearn. The amended note is
// marked a pending offer (Pending=&true) rather than left immediately live.
func serveAmend(deps Deps, vault, vaultName, chunksDir string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		identity := firstHeaderValue(req.Header, cloudflareIdentityHeader)
		if identity == "" {
			return jsonErrorResponse(statusUnauthorized, errServeMissingIdentity)
		}

		var args AmendArgs

		unmarshalErr := json.Unmarshal(req.Body, &args)
		if unmarshalErr != nil {
			return jsonErrorResponse(statusBadRequest, unmarshalErr)
		}

		args.Vault = vault
		if args.VaultName == "" {
			args.VaultName = vaultName
		}

		args.ChunksDir = chunksDir
		pending := true
		args.Pending = &pending
		// Discard is host-local only (vault-offer-curation's design: curation
		// "never crosses the wire") — force false so a client body can never
		// turn a served amend into a delete of an arbitrary vault note.
		args.Discard = false

		clientRepo := args.Repo
		amendDeps := newAmendDeps(deps)
		amendDeps.DetectUser = func(context.Context) string { return identity }
		amendDeps.DetectRepo = func(context.Context) string { return clientRepo }

		var buf bytes.Buffer

		runErr := RunAmend(ctx, args, amendDeps, &buf)
		if runErr != nil {
			return jsonErrorResponse(statusInternalServerError, runErr)
		}

		return offerReceiptResponse(buf.Bytes())
	})
}

// serveLearn handles POST /learn: a LearnArgs-shaped JSON body. Only the
// Cloudflare-authenticated identity stamps user: — the body's own Repo
// field (client-detected, no privilege) passes through unchanged rather
// than being re-detected server-side, which would resolve to the server
// process's own repo context instead of the remote caller's (design.md
// Decisions). Vault is forced to this server's configured value; VaultName
// falls back to it when the client didn't supply one. Always lands as a
// pending offer — never commits as an immediately-live note (vault-offer-
// curation) — and the response never includes note content (design.md API
// Contract).
func serveLearn(deps Deps, vault, vaultName string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		identity := firstHeaderValue(req.Header, cloudflareIdentityHeader)
		if identity == "" {
			return jsonErrorResponse(statusUnauthorized, errServeMissingIdentity)
		}

		var args LearnArgs

		unmarshalErr := json.Unmarshal(req.Body, &args)
		if unmarshalErr != nil {
			return jsonErrorResponse(statusBadRequest, unmarshalErr)
		}

		args.Vault = vault
		if args.VaultName == "" {
			args.VaultName = vaultName
		}

		args.Pending = true

		clientRepo := args.Repo
		learnDeps := newLearnDeps(deps)
		learnDeps.DetectUser = func(context.Context) string { return identity }
		learnDeps.DetectRepo = func(context.Context) string { return clientRepo }

		var buf bytes.Buffer

		runErr := RunLearn(ctx, args, learnDeps, &buf)
		if runErr != nil {
			return jsonErrorResponse(statusInternalServerError, runErr)
		}

		return offerReceiptResponse(buf.Bytes())
	})
}

// serveQuery handles GET /query: QueryArgs-shaped query params, response
// byte-identical to a local `engram query` invocation (design.md API
// Contract) — the handler captures RunQuery's stdout verbatim rather than
// re-deriving the payload.
func serveQuery(deps Deps, vault, chunksDir string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		args := QueryArgs{
			Phrases:       req.Query["phrase"],
			VaultPath:     vault,
			ChunksDir:     chunksDir,
			Limit:         intQueryParam(req.Query, "limit"),
			Project:       firstQueryParam(req.Query, "project"),
			ContentBudget: intQueryParam(req.Query, "content-budget"),
			RecentFill:    intQueryParam(req.Query, "recent-fill"),
			LazyChunks:    boolQueryParam(req.Query, "lazy-chunks"),
			Timings:       boolQueryParam(req.Query, "timings"),
		}

		var buf bytes.Buffer

		err := RunQuery(ctx, args, newQueryDeps(deps), &buf)
		if err != nil {
			return jsonErrorResponse(statusInternalServerError, err)
		}

		return ServeResponse{Status: statusOK, Body: buf.Bytes()}
	})
}

// serveQueryChunks handles GET /query-chunks: ChunkQueryArgs-shaped query
// params, response shape matching local `engram query-chunks`.
func serveQueryChunks(deps Deps, chunksDir string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		args := ChunkQueryArgs{
			Phrases:   req.Query["phrase"],
			ChunksDir: chunksDir,
			Limit:     intQueryParam(req.Query, "limit"),
		}

		var buf bytes.Buffer

		err := RunChunkQuery(ctx, args, newChunkQueryDeps(deps), &buf)
		if err != nil {
			return jsonErrorResponse(statusInternalServerError, err)
		}

		return ServeResponse{Status: statusOK, Body: buf.Bytes()}
	})
}

// serveShow handles GET /show?note=<ref>, response matching local `engram show`.
func serveShow(deps Deps, vault string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		args := ShowArgs{Ref: firstQueryParam(req.Query, "note"), VaultPath: vault}

		var buf bytes.Buffer

		err := RunShow(ctx, args, newShowDeps(deps), &buf)
		if err != nil {
			return jsonErrorResponse(statusInternalServerError, err)
		}

		return ServeResponse{Status: statusOK, Body: buf.Bytes()}
	})
}

// serveShowChunk handles GET /show-chunk?id=<source#anchor>, response
// matching local `engram show-chunk`.
func serveShowChunk(deps Deps, chunksDir string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		args := ShowChunkArgs{Ref: firstQueryParam(req.Query, "id"), ChunksDir: chunksDir}

		var buf bytes.Buffer

		err := RunShowChunk(ctx, args, newShowChunkDeps(deps), &buf)
		if err != nil {
			return jsonErrorResponse(statusInternalServerError, err)
		}

		return ServeResponse{Status: statusOK, Body: buf.Bytes()}
	})
}
