package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// unexported constants.
const (
	envParentBase = "ENGRAM_PARENT"
	envServerBase = "ENGRAM_SERVER"
	// httpStatusMultipleChoices is the first non-2xx status code — used to
	// bound the "success" range without importing net/http here.
	httpStatusMultipleChoices = 300
	// parentSourcedMarker labels engram show/show-chunk's local-miss parent
	// fallback output as parent-sourced (vault-merged-recall D8) — the
	// human-readable analog of a merged query payload's per-item
	// from_parent tag. Only the fallback path is labeled; the explicit
	// --parent route (fetchShow/fetchShowChunk) stays byte-identical to a
	// direct parent request, unlabeled, per the delta spec's "Without
	// --parent, behavior is unchanged" scenario.
	parentSourcedMarker = "# from_parent: true\n"
)

// unexported variables.
var (
	// errParentNotConfigured is returned when --parent is passed to
	// show/show-chunk but ENGRAM_PARENT is not set.
	errParentNotConfigured = errors.New("--parent requires ENGRAM_PARENT to be configured")
	errServeClientNonOK    = errors.New("serve client: non-OK response")
)

// buildQueryParams builds /query's query-string params from args, shared by
// fetchQuery (byte-copy, ENGRAM_SERVER-exclusive mode) and
// fetchQueryPayload (parsed, ENGRAM_PARENT merge fetch).
func buildQueryParams(args QueryArgs) map[string][]string {
	query := map[string][]string{}

	if len(args.Phrases) > 0 {
		query["phrase"] = args.Phrases
	}

	setIntParam(query, "limit", args.Limit)
	setStringParam(query, "project", args.Project)
	setStringParam(query, "text", args.Text)
	setIntParam(query, "content-budget", args.ContentBudget)
	setIntParam(query, "recent-fill", args.RecentFill)
	setBoolParam(query, "lazy-chunks", args.LazyChunks)
	setBoolParam(query, "timings", args.Timings)

	return query
}

// buildURL joins base+path and appends query as a percent-encoded query
// string. internal/ may not import net/url (depguard #700's
// internal-purity rule), so encoding is hand-rolled here rather than in
// cmd/engram (which must stay a single-call/simple-wrapper Fetch
// primitive — targ check-thin-api forbids loops there).
func buildURL(base, path string, query map[string][]string) string {
	target := strings.TrimSuffix(base, "/") + path

	encoded := encodeQuery(query)
	if encoded == "" {
		return target
	}

	return target + "?" + encoded
}

// describeErrorBody extracts the message from a served route's
// {"error": "..."} body, falling back to the raw body when it doesn't
// parse as one.
func describeErrorBody(body []byte) string {
	var errBody errResponse

	if json.Unmarshal(body, &errBody) == nil && errBody.Error != "" {
		return errBody.Error
	}

	return strings.TrimSpace(string(body))
}

// dispatchShow runs `engram show` against the local vault, then — only on a
// not-found miss with ENGRAM_PARENT configured — falls back to the parent
// and labels the result as parent-sourced (vault-merged-recall D8: "Local
// miss falls back to the parent"). Callers reach this only after ruling out
// ENGRAM_SERVER and an explicit --parent, both of which take precedence and
// never fall through here. A local hit, or any error other than a miss
// (e.g. an empty ref), returns as-is without contacting the parent; a miss
// with no parent configured surfaces the same not-found error as before
// this capability existed ("Local miss with no parent configured is still
// an error").
func dispatchShow(ctx context.Context, deps Deps, args ShowArgs, home string, stdout io.Writer) error {
	args.VaultPath = resolveVault(args.VaultPath, home, deps.Getenv)

	localErr := RunShow(ctx, args, newShowDeps(deps), stdout)
	if localErr == nil || !errors.Is(localErr, errShowNoteNotFound) {
		return localErr
	}

	parent := parentBase(deps)
	if parent == "" {
		return localErr
	}

	return fetchShowFallback(ctx, deps, parent, args, stdout)
}

// dispatchShowChunk is dispatchShow's show-chunk counterpart.
func dispatchShowChunk(ctx context.Context, deps Deps, args ShowChunkArgs, home string, stdout io.Writer) error {
	args.ChunksDir = ResolveChunksDir(args.ChunksDir, home, deps.Getenv)

	localErr := RunShowChunk(ctx, args, newShowChunkDeps(deps), stdout)
	if localErr == nil || !errors.Is(localErr, errShowChunkNotFound) {
		return localErr
	}

	parent := parentBase(deps)
	if parent == "" {
		return localErr
	}

	return fetchShowChunkFallback(ctx, deps, parent, args, stdout)
}

// encodeQuery percent-encodes query into a "k=v&k=v" string, keys sorted
// for deterministic output. Repeatable keys (e.g. phrase) emit one k=v
// pair per value.
func encodeQuery(query map[string][]string) string {
	keys := make([]string, 0, len(query))

	for key := range query {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var parts []string

	for _, key := range keys {
		for _, value := range query[key] {
			parts = append(parts, percentEncode(key)+"="+percentEncode(value))
		}
	}

	return strings.Join(parts, "&")
}

// fetchActivate routes `engram activate` through ENGRAM_SERVER: commits
// directly on the host (design.md Decisions — never an offer).
func fetchActivate(ctx context.Context, deps Deps, base string, args ActivateArgs) error {
	//nolint:errchkjson // activateRequest is a plain []string field — never fails to encode
	body, _ := json.Marshal(activateRequest{Notes: args.Notes})

	_, fetchErr := fetchRaw(ctx, deps, base, methodPost, "/activate", nil, body)

	return fetchErr
}

// fetchAmend routes `engram amend` through ENGRAM_SERVER. Same
// repo/user-stamping and offer-receipt handling as fetchLearn.
func fetchAmend(ctx context.Context, deps Deps, base string, args AmendArgs, stdout io.Writer) error {
	args.Repo = detectRepo(ctx, deps.Getwd, deps.Commander)
	args.User = detectUser(ctx, deps.Commander, deps.Username)

	//nolint:errchkjson // AmendArgs is all strings/[]string/bool/*bool fields — never fails to encode
	body, _ := json.Marshal(args)

	resp, fetchErr := fetchRaw(ctx, deps, base, methodPost, "/amend", nil, body)
	if fetchErr != nil {
		return fetchErr
	}

	return printOfferReceipt(resp.Body, stdout)
}

// fetchAndCopy issues a GET request and writes the response body verbatim
// to stdout — byte-identical to a local invocation's own stdout, since the
// served handler captured that same command's Run* stdout output
// unmodified (design.md API Contract).
func fetchAndCopy(
	ctx context.Context, deps Deps, base, path string, query map[string][]string, stdout io.Writer,
) error {
	resp, fetchErr := fetchRaw(ctx, deps, base, methodGet, path, query, nil)
	if fetchErr != nil {
		return fetchErr
	}

	_, writeErr := stdout.Write(resp.Body)
	if writeErr != nil {
		return fmt.Errorf("serve client: write response: %w", writeErr)
	}

	return nil
}

// fetchLearn routes `engram learn` through ENGRAM_SERVER: stamps args.Repo
// and args.User with this client's own detected repo/user (client-detected,
// no privilege/verification — design.md Decisions, serve-client-declared-
// identity), lands as a pending offer on the host, and prints the offer
// receipt rather than a note path — the response deliberately never carries
// note content.
func fetchLearn(ctx context.Context, deps Deps, base string, args LearnArgs, stdout io.Writer) error {
	args.Repo = detectRepo(ctx, deps.Getwd, deps.Commander)
	args.User = detectUser(ctx, deps.Commander, deps.Username)

	//nolint:errchkjson // LearnArgs is all strings/[]string/bool fields — never fails to encode
	body, _ := json.Marshal(args)

	resp, fetchErr := fetchRaw(ctx, deps, base, methodPost, "/learn", nil, body)
	if fetchErr != nil {
		return fetchErr
	}

	return printOfferReceipt(resp.Body, stdout)
}

// fetchQuery routes `engram query` through ENGRAM_SERVER.
func fetchQuery(ctx context.Context, deps Deps, base string, args QueryArgs, stdout io.Writer) error {
	return fetchAndCopy(ctx, deps, base, "/query", buildQueryParams(args), stdout)
}

// fetchQueryChunks routes `engram query-chunks` through ENGRAM_SERVER.
func fetchQueryChunks(ctx context.Context, deps Deps, base string, args ChunkQueryArgs, stdout io.Writer) error {
	query := map[string][]string{}

	if len(args.Phrases) > 0 {
		query["phrase"] = args.Phrases
	}

	setIntParam(query, "limit", args.Limit)

	return fetchAndCopy(ctx, deps, base, "/query-chunks", query, stdout)
}

// fetchQueryPayload routes a parent query (ENGRAM_PARENT merge mode)
// through the same /query route fetchQuery uses, but decodes the response
// into the same queryPayload type the local pipeline produces — instead of
// piping bytes to stdout — so the merge orchestrator can combine it with
// the local payload before a single render.
func fetchQueryPayload(ctx context.Context, deps Deps, base string, args QueryArgs) (queryPayload, error) {
	resp, fetchErr := fetchRaw(ctx, deps, base, methodGet, "/query", buildQueryParams(args), nil)
	if fetchErr != nil {
		return queryPayload{}, fetchErr
	}

	var payload queryPayload

	unmarshalErr := yaml.Unmarshal(resp.Body, &payload)
	if unmarshalErr != nil {
		return queryPayload{}, fmt.Errorf("serve client: parent query: decoding payload: %w", unmarshalErr)
	}

	return payload, nil
}

// fetchRaw issues one ENGRAM_SERVER-mode request and returns its response,
// erroring on transport failure or a non-2xx status (the error message
// prefers the server's {"error": "..."} body when present).
func fetchRaw(
	ctx context.Context, deps Deps, base, method, path string, query map[string][]string, body []byte,
) (FetchResponse, error) {
	resp, fetchErr := deps.Fetch(ctx, method, buildURL(base, path, query), body)
	if fetchErr != nil {
		return FetchResponse{}, fmt.Errorf("serve client: %s %s: %w", method, path, fetchErr)
	}

	if resp.Status < statusOK || resp.Status >= httpStatusMultipleChoices {
		return resp, fmt.Errorf("%w: %s %s: %s", errServeClientNonOK, method, path, describeErrorBody(resp.Body))
	}

	return resp, nil
}

// fetchShow routes `engram show` through ENGRAM_SERVER.
func fetchShow(ctx context.Context, deps Deps, base string, args ShowArgs, stdout io.Writer) error {
	return fetchAndCopy(ctx, deps, base, "/show", map[string][]string{"note": {args.Ref}}, stdout)
}

// fetchShowChunk routes `engram show-chunk` through ENGRAM_SERVER.
func fetchShowChunk(ctx context.Context, deps Deps, base string, args ShowChunkArgs, stdout io.Writer) error {
	return fetchAndCopy(ctx, deps, base, "/show-chunk", map[string][]string{"id": {args.Ref}}, stdout)
}

// fetchShowChunkFallback is fetchShowFallback's show-chunk counterpart.
func fetchShowChunkFallback(ctx context.Context, deps Deps, base string, args ShowChunkArgs, stdout io.Writer) error {
	var buf bytes.Buffer

	fetchErr := fetchShowChunk(ctx, deps, base, args, &buf)
	if fetchErr != nil {
		return fetchErr
	}

	return writeParentSourced(stdout, buf.Bytes())
}

// fetchShowFallback fetches ref from the parent for dispatchShow's
// local-miss fallback and labels the output as parent-sourced. It buffers
// fetchShow's result rather than writing straight to stdout, so a fetch
// failure (parent also has no match, or is unreachable) never leaves a
// stray marker line ahead of an error.
func fetchShowFallback(ctx context.Context, deps Deps, base string, args ShowArgs, stdout io.Writer) error {
	var buf bytes.Buffer

	fetchErr := fetchShow(ctx, deps, base, args, &buf)
	if fetchErr != nil {
		return fetchErr
	}

	return writeParentSourced(stdout, buf.Bytes())
}

// isURLUnreserved reports whether c is an RFC 3986 unreserved character
// (safe unescaped in a URL query component).
func isURLUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-' || c == '_' || c == '.' || c == '~':
		return true
	default:
		return false
	}
}

// parentBase returns the ENGRAM_PARENT base URL (e.g. "http://host:port"),
// or "" when unset — mirrors serverBase, but additive rather than
// exclusive: unlike ENGRAM_SERVER, a set ENGRAM_PARENT does not replace
// local behavior, it adds a merge step on top of it (design.md Decision 5).
func parentBase(deps Deps) string {
	if deps.Getenv == nil {
		return ""
	}

	return deps.Getenv(envParentBase)
}

// percentEncode RFC 3986-escapes s for use as a URL query key or value.
func percentEncode(s string) string {
	var b strings.Builder

	for i := range len(s) {
		char := s[i]
		if isURLUnreserved(char) {
			b.WriteByte(char)

			continue
		}

		fmt.Fprintf(&b, "%%%02X", char)
	}

	return b.String()
}

// printOfferReceipt decodes a served learn/amend's {status, luhmann} body
// and prints it — the client-mode counterpart to local learn/amend
// printing the written note's path, deliberately different since a served
// write's fate is a pending offer, not an immediately-live note.
func printOfferReceipt(body []byte, stdout io.Writer) error {
	var receipt offerReceipt

	unmarshalErr := json.Unmarshal(body, &receipt)
	if unmarshalErr != nil {
		return fmt.Errorf("serve client: decoding offer receipt: %w", unmarshalErr)
	}

	_, writeErr := fmt.Fprintf(stdout, "%s: %s\n", receipt.Status, receipt.Luhmann)
	if writeErr != nil {
		return fmt.Errorf("serve client: write response: %w", writeErr)
	}

	return nil
}

// resolveParentOrError returns ENGRAM_PARENT's base URL, or
// errParentNotConfigured when unset — the shared guard for show/show-chunk's
// --parent flag.
func resolveParentOrError(deps Deps) (string, error) {
	parent := parentBase(deps)
	if parent == "" {
		return "", errParentNotConfigured
	}

	return parent, nil
}

// serverBase returns the ENGRAM_SERVER base URL (e.g. "http://host:port"),
// or "" when unset — the signal that a served CLI target should run
// locally instead of routing through the HTTP client (tasks.md 8.1).
func serverBase(deps Deps) string {
	if deps.Getenv == nil {
		return ""
	}

	return deps.Getenv(envServerBase)
}

// setBoolParam sets key to a single "true"/"false" query value when value
// is true; false is the query encoding's default (absent), matching
// boolQueryParam's ParseBool-of-absent-is-false behavior server-side.
func setBoolParam(query map[string][]string, key string, value bool) {
	if value {
		query[key] = []string{strconv.FormatBool(value)}
	}
}

// setIntParam sets key to a single decimal query value when value is
// non-zero (0 is every int flag's baked-default sentinel here).
func setIntParam(query map[string][]string, key string, value int) {
	if value != 0 {
		query[key] = []string{strconv.Itoa(value)}
	}
}

// setStringParam sets key to a single query value when value is non-empty.
func setStringParam(query map[string][]string, key string, value string) {
	if value != "" {
		query[key] = []string{value}
	}
}

// writeParentSourced writes the parent-sourced marker line followed by body
// verbatim — the shared tail of fetchShowFallback and
// fetchShowChunkFallback.
func writeParentSourced(stdout io.Writer, body []byte) error {
	_, writeErr := io.WriteString(stdout, parentSourcedMarker)
	if writeErr != nil {
		return fmt.Errorf("serve client: write response: %w", writeErr)
	}

	_, writeErr = stdout.Write(body)
	if writeErr != nil {
		return fmt.Errorf("serve client: write response: %w", writeErr)
	}

	return nil
}
