package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// unexported constants.
const (
	envServerBase = "ENGRAM_SERVER"
	// httpStatusMultipleChoices is the first non-2xx status code — used to
	// bound the "success" range without importing net/http here.
	httpStatusMultipleChoices = 300
)

// unexported variables.
var (
	errServeClientNonOK = errors.New("serve client: non-OK response")
)

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
	query := map[string][]string{}

	if len(args.Phrases) > 0 {
		query["phrase"] = args.Phrases
	}

	setIntParam(query, "limit", args.Limit)
	setStringParam(query, "project", args.Project)
	setIntParam(query, "content-budget", args.ContentBudget)
	setIntParam(query, "recent-fill", args.RecentFill)
	setBoolParam(query, "lazy-chunks", args.LazyChunks)
	setBoolParam(query, "timings", args.Timings)

	return fetchAndCopy(ctx, deps, base, "/query", query, stdout)
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
