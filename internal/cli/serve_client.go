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
	// envRemovedServer names the removed thin-client variable; it survives
	// only for the pre-dispatch guard that refuses it (design D1).
	envRemovedServer = "ENGRAM_SERVER"
	// httpStatusMultipleChoices is the first non-2xx status code — used to
	// bound the "success" range without importing net/http here.
	httpStatusMultipleChoices = 300
	// parentSourcedMarker labels engram show's local-miss parent fallback
	// output as parent-sourced (vault-merged-recall D8) — the
	// human-readable analog of a merged query payload's per-item
	// from_parent tag. Only the fallback path is labeled; the explicit
	// --parent route (fetchShow) stays byte-identical to a direct parent
	// request, unlabeled. show-chunk never contacts the parent (Q1).
	parentSourcedMarker = "# from_parent: true\n"
)

// unexported variables.
var (
	// errEngramServerRemoved is the D1 hard error for a set ENGRAM_SERVER.
	errEngramServerRemoved = errors.New("ENGRAM_SERVER is no longer supported")
	// errParentBackedOff is returned when a show's parent request is
	// skipped inside the backoff window (design D6).
	errParentBackedOff = errors.New("parent contact skipped inside the backoff window")
	// errParentNotConfigured is returned when --parent is passed to
	// show/activate but ENGRAM_PARENT is not set.
	errParentNotConfigured = errors.New("--parent requires ENGRAM_PARENT to be configured")
	// errParentUnreachable marks a parent request that failed as an outage
	// (a transport error, timeout or 5xx), which backs the parent off.
	errParentUnreachable = errors.New("parent unreachable")
	errServeClientNonOK  = errors.New("serve client: non-OK response")
)

// buildQueryParams builds the parent /query request's query-string params
// from args, for fetchQueryPayload (the ENGRAM_PARENT merge fetch).
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
// an explicit --parent, which takes precedence and never falls through
// here. A local hit, or any error other than a miss
// (e.g. an empty ref), returns as-is without contacting the parent; a miss
// with no parent configured surfaces the same not-found error as before
// this capability existed ("Local miss with no parent configured is still
// an error").
func dispatchShow(ctx context.Context, deps Deps, args ShowArgs, home string, stdout io.Writer) error {
	args.VaultPath = resolveVault(args.VaultPath, home, deps.Getenv)

	ensureErr := ensureVault(deps, args.VaultPath)
	if ensureErr != nil {
		return ensureErr
	}

	localErr := RunShow(ctx, args, newShowDeps(deps), stdout)
	if localErr == nil || !errors.Is(localErr, errShowNoteNotFound) {
		return localErr
	}

	parent := parentBase(deps)
	if parent == "" {
		return localErr
	}

	fallbackErr := fetchShowFallback(ctx, deps, parent, args, stdout)
	if errors.Is(fallbackErr, errParentBackedOff) {
		return localErr
	}

	return fallbackErr
}

// dispatchShowParent runs `engram show --parent`: the ref is resolved
// against ENGRAM_PARENT only, as a parent contact (rulings S3, S11).
func dispatchShowParent(ctx context.Context, deps Deps, args ShowArgs, home string, stdout io.Writer) error {
	parent, parentErr := resolveParentOrError(deps)
	if parentErr != nil {
		return parentErr
	}

	args.VaultPath = resolveVault(args.VaultPath, home, deps.Getenv)

	ensureErr := ensureVault(deps, args.VaultPath)
	if ensureErr != nil {
		return ensureErr
	}

	return fetchShowContact(ctx, deps, parent, args, stdout)
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

// fetchQueryPayload routes a parent query (ENGRAM_PARENT merge mode)
// through the parent's /query route, asking for the dedupe keys (design
// D9), and decodes the response into the same queryPayload type the local
// pipeline produces — instead of piping bytes to stdout — so the merge
// orchestrator can combine it with the local payload before a single
// render.
func fetchQueryPayload(ctx context.Context, deps Deps, base string, args QueryArgs) (queryPayload, error) {
	params := buildQueryParams(args)
	params["dedupe-keys"] = []string{"1"}

	resp, fetchErr := fetchRaw(ctx, deps, base, methodGet, "/query", params, nil)
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

// fetchRaw issues one parent request and returns its response,
// erroring on transport failure or a non-2xx status (the error message
// prefers the server's {"error": "..."} body when present).
func fetchRaw(
	ctx context.Context, deps Deps, base, method, path string, query map[string][]string, body []byte,
) (FetchResponse, error) {
	resp, fetchErr := deps.Fetch(ctx, method, buildURL(base, path, query), body)
	if fetchErr != nil {
		return FetchResponse{}, fmt.Errorf("serve client: %s %s: %w (%w)", method, path, fetchErr, errParentUnreachable)
	}

	if resp.Status >= statusInternalServerError {
		return resp, fmt.Errorf("%w: %s %s: %s (%w)",
			errServeClientNonOK, method, path, describeErrorBody(resp.Body), errParentUnreachable)
	}

	if resp.Status < statusOK || resp.Status >= httpStatusMultipleChoices {
		return resp, fmt.Errorf("%w: %s %s: %s", errServeClientNonOK, method, path, describeErrorBody(resp.Body))
	}

	return resp, nil
}

// fetchShow fetches `engram show` output for a ref from the parent.
func fetchShow(ctx context.Context, deps Deps, base string, args ShowArgs, stdout io.Writer) error {
	return fetchAndCopy(ctx, deps, base, "/show", map[string][]string{"note": {args.Ref}}, stdout)
}

// fetchShowContact runs one show request against the parent as a parent
// contact: the vault ID is stamped and the location checked first (ruling
// S3), the backoff gate may skip the request (errParentBackedOff, after its
// one warning), and the outcome is recorded for backoff (ruling S11). A
// failed stamp skips the gate and the record (ruling S4).
func fetchShowContact(ctx context.Context, deps Deps, parent string, args ShowArgs, stdout io.Writer) error {
	if !prepareParentContact(deps, "show", args.VaultPath) {
		// Unstamped: contact without bookkeeping (ruling S4).
		return fetchShow(ctx, deps, parent, args, stdout)
	}

	store := outboxStoreFromDeps(deps)
	if !gateParentContact(store, args.VaultPath, parent, false) {
		return errParentBackedOff
	}

	fetchErr := fetchShow(ctx, deps, parent, args, stdout)
	recordParentContact(store, args.VaultPath, parent, "", fetchErr)

	return fetchErr
}

// fetchShowFallback fetches ref from the parent for dispatchShow's
// local-miss fallback and labels the output as parent-sourced. It buffers
// fetchShow's result rather than writing straight to stdout, so a fetch
// failure (parent also has no match, or is unreachable) never leaves a
// stray marker line ahead of an error.
func fetchShowFallback(ctx context.Context, deps Deps, base string, args ShowArgs, stdout io.Writer) error {
	var buf bytes.Buffer

	fetchErr := fetchShowContact(ctx, deps, base, args, &buf)
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
// or "" when unset. It is additive: a set ENGRAM_PARENT never replaces local
// behavior, it adds a merge step on top of it (design.md Decision 5).
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

// removedServerError returns the D1 hard error when ENGRAM_SERVER is set,
// naming ENGRAM_PARENT with the same URL, or nil when it is unset. Never an
// alias: the value is only echoed back in the message, never contacted.
func removedServerError(deps Deps) error {
	removed := envOrEmpty(deps.Getenv, envRemovedServer)
	if removed == "" {
		return nil
	}

	return fmt.Errorf(
		"%w; set ENGRAM_PARENT=%s instead — every environment now keeps its own local vault "+
			"and offers notes to the parent",
		errEngramServerRemoved, removed)
}

// resolveParentOrError returns ENGRAM_PARENT's base URL, or
// errParentNotConfigured when unset — the shared guard for show's and
// activate's --parent flag.
func resolveParentOrError(deps Deps) (string, error) {
	parent := parentBase(deps)
	if parent == "" {
		return "", errParentNotConfigured
	}

	return parent, nil
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
// verbatim — the tail of fetchShowFallback.
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
