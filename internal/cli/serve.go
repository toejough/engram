package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Exported constants.
const (
	// FetchResponseReadLimit is how many bytes of one parent response the
	// HTTP edge reads (final review F7): one more than the largest response
	// accepted, so NewDeps' Fetch can tell an oversized body from one at
	// the limit instead of silently truncating it.
	FetchResponseReadLimit = maxFetchResponseBytes + 1
	// MaxServeRequestBytes caps one served request body (final review F7):
	// an offer or activate body is a few KB, so 4 MiB leaves ample room
	// while bounding what an unauthenticated caller can make the server
	// buffer. The HTTP edge enforces it; a route answers 413 only when the
	// body itself exceeded this cap, and 400 for any other body-read
	// failure (S34 residual).
	MaxServeRequestBytes = 4 << 20
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
	// BodyErr is the error reading Body in full (for one, a body over
	// MaxServeRequestBytes); every route answers 400 when it is set, or
	// 413 when BodyTooLarge is also set (S34 residual).
	BodyErr error
	// BodyTooLarge reports whether BodyErr is specifically the body
	// exceeding MaxServeRequestBytes, as opposed to any other read
	// failure (e.g. a client disconnect). Meaningless when BodyErr is nil.
	BodyTooLarge bool
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
	// Design D2: serve stamps a missing vault ID at startup (later starts
	// reuse it), and a failed location check only warns — refusing would
	// turn a false positive into a host outage.
	state := exchangeStateFromDeps(deps)

	// A corrupt or empty ID file (a partial write, a git conflict) only
	// warns, naming the remedy — refusing would crash-loop under launchd.
	_, stampErr := stampVaultID(state, args.Vault)
	if stampErr != nil {
		_, _ = fmt.Fprintf(deps.Stderr, "engram: warning: serve: %v; serving anyway\n", stampErr)
	} else {
		warnVaultLocation(state, args.Vault, deps.Stderr)
	}

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
// api, design D7): exactly query and show (GET) and activate and learn
// (POST). amend, query-chunks and show-chunk have no route — children offer
// amendments through learn, and chunks never cross vaults. Every handler
// calls the existing code path for its command — no reimplementation of
// command logic here (tasks.md 2.2) — sharing the CLI's existing vault
// locks (ADR-0013). vault/vaultName/chunksDir are resolved once by RunServe
// and baked into every handler closure; a served request can never redirect
// the server at a different vault path.
func ServeRoutes(deps Deps, vault, vaultName, chunksDir string) []ServeRoute {
	return []ServeRoute{
		{Method: methodGet, Pattern: "/query", Handler: requireReadBody(serveQuery(deps, vault, chunksDir))},
		{Method: methodGet, Pattern: "/show", Handler: requireReadBody(serveShow(deps, vault))},
		{Method: methodPost, Pattern: "/activate", Handler: requireReadBody(serveActivate(deps, vault))},
		{Method: methodPost, Pattern: "/learn", Handler: requireReadBody(serveLearn(deps, vault, vaultName))},
	}
}

// unexported constants.
const (
	// loopRefusalReason discriminates the 409 loop refusal (ruling S13).
	loopRefusalReason = "loop"
	// maxFetchResponseBytes is the largest parent response accepted (final
	// review F7): a merged query asks for unbounded budgets, so it is
	// generous, but bounded so a malicious parent can't exhaust memory.
	maxFetchResponseBytes = 16 << 20
	// maxQueryTextBytes caps the served /query text param (2 KB).
	maxQueryTextBytes         = 2048
	methodGet                 = "GET"
	methodPost                = "POST"
	offerReceivedStatus       = "offer received"
	okStatus                  = "ok"
	statusBadRequest          = 400
	statusConflict            = 409
	statusInternalServerError = 500
	statusNotFound            = 404
	statusOK                  = 200
	// statusRequestTooLarge answers a body over MaxServeRequestBytes.
	statusRequestTooLarge = 413
)

// unexported variables.
var (
	// errServeBodyTooLarge answers 413: the body itself exceeded
	// MaxServeRequestBytes (S34 residual — split from the generic
	// unreadable-body case, which answers 400 instead).
	errServeBodyTooLarge = errors.New("serve: request body over the size limit")
	// errServeBodyUnreadable answers 400: any other body-read failure
	// (e.g. a client disconnect), never conflated with the over-the-cap
	// case above.
	errServeBodyUnreadable = errors.New("serve: request body unreadable")
	// errServeEmptyIdentity guards the identity floor (serve-client-
	// declared-identity): a served learn must claim SOME identity —
	// the server trusts whatever is declared (no edge-authentication header
	// required or consulted), but an empty claim is refused outright.
	errServeEmptyIdentity = errors.New("serve: user: must be non-empty")
)

// activateRequest is the JSON body shape for POST /activate.
type activateRequest struct {
	Notes []string `json:"notes"`
}

// cycleRefusal is the 409 body for an offer whose path already holds this
// vault's ID: the error plus the vault ID, so a child that is its own parent
// (or sits on the cycle) can recognize and cache it (ruling S12).
type cycleRefusal struct {
	Error string `json:"error"`
	// Reason is the explicit discriminator ("loop"), so a child never
	// reads another 409 cause as a loop (ruling S13).
	Reason  string `json:"reason"`
	VaultID string `json:"vault_id"` //nolint:tagliatelle // design D7 fixes the exchange's snake_case keys
}

// errResponse is the JSON response body for any served-route failure.
type errResponse struct {
	Error string `json:"error"`
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

// activateResponse maps a served activate's per-ref outcome to a status,
// so a 5xx only ever means a server-side failure (ruling S18): every ref
// activated is 200 {"status":"ok"}; a ref that failed while being
// activated (not merely missing) is 500; otherwise none found is 404 and a
// partial result 200, each with the per-ref result as the body.
func activateResponse(result activateResult) ServeResponse {
	if len(result.Failed) == 0 {
		return jsonOKResponse()
	}

	status := statusOK

	switch {
	case slices.ContainsFunc(result.Failed, func(failure activateFailure) bool { return !failure.notFound }):
		status = statusInternalServerError
	case len(result.Activated) == 0:
		status = statusNotFound
	}

	body, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return jsonErrorResponse(statusInternalServerError, marshalErr)
	}

	return ServeResponse{Status: status, Body: body}
}

// boolQueryParam parses key's first query value as a bool (strconv.ParseBool);
// absent or unparseable values report false.
func boolQueryParam(query map[string][]string, key string) bool {
	value, _ := strconv.ParseBool(firstQueryParam(query, key))

	return value
}

// capQueryText truncates text to maxQueryTextBytes on a UTF-8 rune boundary,
// bounding the served query string; trigger cues occur early in a message, so
// truncation is silent (runbook-lexical-triggers spec).
func capQueryText(text string) string {
	if len(text) <= maxQueryTextBytes {
		return text
	}

	end := maxQueryTextBytes
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}

	return text[:end]
}

// cycleRefusalResponse is the 409 loop refusal carrying this vault's ID.
func cycleRefusalResponse(err error, vaultID string) ServeResponse {
	//nolint:errchkjson // plain string fields never fail to encode
	body, _ := json.Marshal(cycleRefusal{Error: err.Error(), Reason: loopRefusalReason, VaultID: vaultID})

	return ServeResponse{Status: statusConflict, Body: body}
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

// jsonBodyResponse marshals body under 200. Every caller passes a struct of
// plain string and bool fields, which never fails to encode.
func jsonBodyResponse(body any) ServeResponse {
	encoded, _ := json.Marshal(body) //nolint:errchkjson // see doc comment

	return ServeResponse{Status: statusOK, Body: encoded}
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

// requireReadBody answers for a request whose body could not be read in
// full (final review F7) — 413 only when the body itself exceeded
// MaxServeRequestBytes at the HTTP edge (BodyTooLarge), and 400 for any
// other body-read failure (S34 residual) — and otherwise hands the
// request to next, so no route ever acts on a truncated body.
func requireReadBody(next ServeHandler) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		if req.BodyErr != nil && req.BodyTooLarge {
			return jsonErrorResponse(statusRequestTooLarge,
				fmt.Errorf("%w: %w", errServeBodyTooLarge, req.BodyErr))
		}

		if req.BodyErr != nil {
			return jsonErrorResponse(statusBadRequest,
				fmt.Errorf("%w: %w", errServeBodyUnreadable, req.BodyErr))
		}

		return next.Serve(ctx, req)
	})
}

// serveActivate handles POST /activate: commits directly, no pending-offer
// marker, no curation (design.md Decisions — activate never mutates note
// content, so there's no new claim for curation to judge).
func serveActivate(deps Deps, vault string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		var body activateRequest

		unmarshalErr := json.Unmarshal(req.Body, &body)
		if unmarshalErr != nil {
			return jsonErrorResponse(statusBadRequest, unmarshalErr)
		}

		refErr := validateServedActivateRefs(body.Notes)
		if refErr != nil {
			return jsonErrorResponse(statusBadRequest, refErr)
		}

		args := ActivateArgs{Vault: vault, Notes: body.Notes}

		// Local only: a served activate never reaches past this vault, and
		// resolves refs only against its listed note names (final review F2).
		activate := newActivateDeps(deps)
		activate.Resolve = listedNoteResolver(deps, vault)

		result, runErr := activateRefs(ctx, args, activate)
		if runErr != nil {
			return jsonErrorResponse(statusInternalServerError, runErr)
		}

		return activateResponse(result)
	})
}

// serveLearn handles POST /learn: a LearnArgs-shaped JSON body, a child's
// offer (design D7). The body's own User field (client-detected, no
// verification — serve-client-declared-identity) stamps user:, rejected
// outright when empty; its Repo field (also client-detected, no privilege)
// passes through unchanged rather than being re-detected server-side, which
// would resolve to the server process's own repo context instead of the
// remote caller's. Vault is forced to this server's configured value;
// VaultName falls back to it when the client didn't supply one. The offer
// is validated first (400, or 409 for a cycle, with nothing written), then
// lands as a pending offer — new, or the same origin's updated in place —
// never as an immediately-live note (vault-offer-curation). The response is
// the offer receipt, never note content.
func serveLearn(deps Deps, vault, vaultName string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		var args LearnArgs

		unmarshalErr := json.Unmarshal(req.Body, &args)
		if unmarshalErr != nil {
			return jsonErrorResponse(statusBadRequest, unmarshalErr)
		}

		validateErr := validateServedLearn(args)
		if validateErr != nil {
			return jsonErrorResponse(statusBadRequest, validateErr)
		}

		vaultID, idErr := stampVaultID(exchangeStateFromDeps(deps), vault)
		if idErr != nil {
			return jsonErrorResponse(statusInternalServerError, idErr)
		}

		args.Vault = vault
		if args.VaultName == "" {
			args.VaultName = vaultName
		}

		identity, clientRepo := args.User, args.Repo
		learnDeps := newLearnDeps(deps)
		learnDeps.DetectUser = func(context.Context) string { return identity }
		learnDeps.DetectRepo = func(context.Context) string { return clientRepo }

		receipt, runErr := runServedLearn(ctx, args, servedLearnDeps{
			learn:    learnDeps,
			readFile: deps.FS.ReadFile,
			mintXID:  func() (string, error) { return mintXID(deps.RandRead) },
			vaultID:  vaultID,
		})

		switch {
		case errors.Is(runErr, errOfferCycle):
			return cycleRefusalResponse(runErr, vaultID)
		case runErr != nil:
			return jsonErrorResponse(statusInternalServerError, runErr)
		}

		return jsonBodyResponse(receipt)
	})
}

// serveQuery handles GET /query: QueryArgs-shaped query params, response
// byte-identical to a local `engram query` invocation (design.md API
// Contract) — the handler captures RunQuery's stdout verbatim rather than
// re-deriving the payload. With dedupe-keys=1 (design D7) the payload also
// carries the vault ID and each note item's exchange hash and aliases, the
// keys a child's merged query dedupes by.
func serveQuery(deps Deps, vault, chunksDir string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		args := QueryArgs{
			Phrases:       req.Query["phrase"],
			VaultPath:     vault,
			ChunksDir:     chunksDir,
			Limit:         intQueryParam(req.Query, "limit"),
			Project:       firstQueryParam(req.Query, "project"),
			Text:          capQueryText(firstQueryParam(req.Query, "text")),
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

		if !boolQueryParam(req.Query, "dedupe-keys") {
			return ServeResponse{Status: statusOK, Body: buf.Bytes()}
		}

		keyed, keyErr := addDedupeKeys(deps, vault, buf.Bytes())
		if keyErr != nil {
			return jsonErrorResponse(statusInternalServerError, keyErr)
		}

		return ServeResponse{Status: statusOK, Body: keyed}
	})
}

// serveShow handles GET /show?note=<ref>, response matching local `engram
// show`. With raw=1 (design D7 H7) it returns the note's raw JSON envelope
// instead. A missing note is a 404.
func serveShow(deps Deps, vault string) ServeHandler {
	return serveHandlerFunc(func(ctx context.Context, req ServeRequest) ServeResponse {
		ref := firstQueryParam(req.Query, "note")

		if boolQueryParam(req.Query, "raw") {
			envelope, rawErr := rawShowEnvelope(deps, vault, ref)
			if rawErr != nil {
				return jsonErrorResponse(showErrorStatus(rawErr), rawErr)
			}

			return jsonBodyResponse(envelope)
		}

		args := ShowArgs{Ref: ref, VaultPath: vault}

		var buf bytes.Buffer

		err := RunShow(ctx, args, newShowDeps(deps), &buf)
		if err != nil {
			return jsonErrorResponse(showErrorStatus(err), err)
		}

		return ServeResponse{Status: statusOK, Body: buf.Bytes()}
	})
}

// showErrorStatus maps a show failure to its HTTP status: a missing note is
// a 404, an empty ref a 400, anything else a 500.
func showErrorStatus(err error) int {
	switch {
	case errors.Is(err, errShowNoteNotFound):
		return statusNotFound
	case errors.Is(err, errShowEmptyRef):
		return statusBadRequest
	default:
		return statusInternalServerError
	}
}
