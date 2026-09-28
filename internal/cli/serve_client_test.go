package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestFetchQueryPayload_AllParamsSet exercises every setBoolParam/
// setIntParam/setStringParam "value present" branch of the parent /query
// request in one call (the absent branches are covered by the other
// TestFetchQueryPayload_* tests, which set none of them).
func TestFetchQueryPayload_AllParamsSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var got fakeFetchCall

	deps := cli.Deps{Fetch: func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
		got = fakeFetchCall{method: method, url: url}

		return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nitems: []\n")}, nil
	}}

	_, err := cli.ExportFetchQueryPayload(context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{
		Phrases: []string{"hello"}, Limit: 7, Project: "engram", ContentBudget: 9, RecentFill: 2,
		LazyChunks: true, Timings: true,
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got.method).To(Equal("GET"))
	g.Expect(got.url).To(Equal(
		"http://parent-host:8420/query?content-budget=9&lazy-chunks=true&limit=7" +
			"&phrase=hello&project=engram&recent-fill=2&timings=true",
	))
}

// TestFetchQueryPayload_DecodesSuccessfulResponse covers the happy path: a
// parent's /query YAML response decodes into the same queryPayload type
// RunQuery's own rendering produces.
func TestFetchQueryPayload_DecodesSuccessfulResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.Deps{Fetch: func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
		return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nmodel_id: m@4\nitems: []\n")}, nil
	}}

	payload, err := cli.ExportFetchQueryPayload(
		context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{Phrases: []string{"x"}})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.ExportQueryPayloadModelID(payload)).To(Equal("m@4"))
}

// TestFetchQueryPayload_MalformedBodyErrors covers a response body that
// isn't valid YAML.
func TestFetchQueryPayload_MalformedBodyErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.Deps{Fetch: func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
		return cli.FetchResponse{Status: 200, Body: []byte("not: [valid: yaml")}, nil
	}}

	_, err := cli.ExportFetchQueryPayload(
		context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{})

	g.Expect(err).To(HaveOccurred())
}

// TestFetchQueryPayload_NonOKStatusErrors covers a non-2xx response.
func TestFetchQueryPayload_NonOKStatusErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.Deps{Fetch: func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
		return cli.FetchResponse{Status: 500, Body: []byte(`{"error":"boom"}`)}, nil
	}}

	_, err := cli.ExportFetchQueryPayload(
		context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{})

	g.Expect(err).To(MatchError(ContainSubstring("boom")))
}

// TestFetchQueryPayload_TextRoundTripsByteIdentically: a 1 KB --text with
// quotes, newlines, '&', '=', '%' and non-ASCII characters survives the
// hand-rolled query encoder unchanged when decoded by a standard parser.
func TestFetchQueryPayload_TextRoundTripsByteIdentically(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const unit = "say \"hi\"\nfoo&bar=baz 100% naïve привет мир /please\t"

	text := strings.Repeat(unit, 1024/len(unit))
	g.Expect(utf8.ValidString(text)).To(BeTrue(), "fixture must be valid UTF-8")

	var got fakeFetchCall

	deps := cli.Deps{Fetch: func(_ context.Context, method, target string, _ []byte) (cli.FetchResponse, error) {
		got = fakeFetchCall{method: method, url: target}

		return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nitems: []\n")}, nil
	}}

	_, err := cli.ExportFetchQueryPayload(
		context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{Text: text})
	g.Expect(err).NotTo(HaveOccurred())

	parsed, parseErr := url.Parse(got.url)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(parsed.Query().Get("text")).To(Equal(text))
	g.Expect(parsed.Query()["phrase"]).To(BeEmpty())
}

// TestFetchQueryPayload_TransportErrorPropagates covers a transport-level
// failure (e.g. connection refused).
func TestFetchQueryPayload_TransportErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.Deps{Fetch: func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
		return cli.FetchResponse{}, errors.New("connection refused")
	}}

	_, err := cli.ExportFetchQueryPayload(
		context.Background(), deps, "http://parent-host:8420", cli.QueryArgs{})

	g.Expect(err).To(MatchError(ContainSubstring("connection refused")))
}

// TestLocalAmend_ClearPendingClearsTheMarker covers the curate runbook's
// core mechanism: `engram amend --clear-pending` is the only CLI-facing way
// to clear a pending-offer note's marker (local amend never sets it —
// TestLocalAmend_NeverSetsPendingMarker — but must be able to clear one a
// served write left behind).
func TestLocalAmend_ClearPendingClearsTheMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "1.2026-01-01.offer.md")
	g.Expect(os.WriteFile(notePath, []byte(pendingFactNote), 0o600)).To(Succeed())

	stderr := executeForTest(t, []string{"engram", "amend", "--vault", vault, "--target", "1", "--clear-pending"})
	g.Expect(stderr).To(BeEmpty())

	raw, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).NotTo(ContainSubstring("pending: true"))
}

// TestLocalAmend_ClearPendingOnSkillRunbookNote_MakesItLive covers the
// skill-runbook-registration extension of the curate runbook's core
// mechanism (vault-offer-curation ADDED requirement): `engram amend
// --clear-pending` clears the marker on a runbook note carrying skill_hash
// (not just fact/feedback), skill_hash survives untouched, and the note
// then surfaces in `engram query` like any ordinary runbook.
func TestLocalAmend_ClearPendingOnSkillRunbookNote_MakesItLive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := filepath.Join(vault, "4.2026-01-04.skill-demo.md")
	g.Expect(os.WriteFile(notePath, []byte(pendingSkillRunbookNote), 0o600)).To(Succeed())

	stderr := executeForTest(t, []string{"engram", "amend", "--vault", vault, "--target", "4", "--clear-pending"})
	g.Expect(stderr).To(BeEmpty())

	raw, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).NotTo(ContainSubstring("pending: true"))
	g.Expect(string(raw)).To(ContainSubstring("skill_hash: abc123"))

	// The cleared note now surfaces like any ordinary runbook: it is no
	// longer a pending offer.
	g.Expect(cli.ExportNoteHasPendingMarker(raw)).To(BeFalse())
}

// TestLocalAmend_DiscardDeletesNoteAndSidecar covers the curate runbook's
// "covered" outcome (vault-offer-curation): `engram amend --discard` removes
// both the note and its sidecar rather than amending content.
func TestLocalAmend_DiscardDeletesNoteAndSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	notePath := writeServeVaultFile(t, vault, "1.2026-01-01.offer.md")
	sidecarPath := embed.SidecarPath(notePath)
	g.Expect(os.WriteFile(
		sidecarPath, embed.MarshalSidecar(embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion}), 0o600,
	)).To(Succeed())

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "amend", "--vault", vault, "--target", "1", "--discard"}, func(*cli.Deps) {})
	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal(notePath + "\n"))

	_, noteErr := os.Stat(notePath)
	g.Expect(os.IsNotExist(noteErr)).To(BeTrue())

	_, sidecarErr := os.Stat(sidecarPath)
	g.Expect(os.IsNotExist(sidecarErr)).To(BeTrue())
}

func TestLocalAmend_NeverSetsPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.existing.md")

	stderr := executeForTest(t, []string{"engram", "amend", "--vault", vault, "--target", "1", "--object", "amended"})
	g.Expect(stderr).To(BeEmpty())

	raw, readErr := os.ReadFile(filepath.Join(vault, "1.2026-01-01.existing.md"))
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).NotTo(ContainSubstring("pending:"))
}

// TestLocalLearn_NeverSetsPendingMarker and TestLocalAmend_NeverSetsPendingMarker
// cover tasks.md 10.6's negative case: a local (non-served) learn/amend
// never carries the pending-offer marker.
func TestLocalLearn_NeverSetsPendingMarker(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stderr := executeForTest(t, []string{
		"engram", "learn", "fact", "--vault", vault,
		"--slug", "local-fact", "--source", "test", "--position", "top",
		"--situation", "a local write", "--subject", "engram", "--predicate", "writes", "--object", "locally",
	})
	g.Expect(stderr).To(BeEmpty())

	matches, globErr := filepath.Glob(filepath.Join(vault, "*.md"))
	g.Expect(globErr).NotTo(HaveOccurred())
	g.Expect(matches).To(HaveLen(1))

	if len(matches) == 0 {
		return
	}

	raw, readErr := os.ReadFile(matches[0])
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(raw)).NotTo(ContainSubstring("pending:"))
}

// TestParentBase_NilGetenv covers parentBase's nil-safety guard, mirroring
// TestServerBase_NilGetenv.
func TestParentBase_NilGetenv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportParentBase(cli.Deps{})).To(Equal(""))
}

// TestParentBase_ReadsEngramParentEnv covers parentBase resolving from the
// ENGRAM_PARENT environment variable — dispatch-level, not a QueryArgs
// flag threaded through args (design.md Decision 5).
func TestParentBase_ReadsEngramParentEnv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.Deps{Getenv: func(key string) string {
		if key == "ENGRAM_PARENT" {
			return "http://parent-host:8420"
		}

		return ""
	}}

	g.Expect(cli.ExportParentBase(deps)).To(Equal("http://parent-host:8420"))
}

// TestServeTarget_RequiresAddr covers tasks.md 3.1/10.2: `engram serve`
// with no explicit bind address refuses to start rather than silently
// binding a default (targ's own required-flag validation enforces this —
// ServeArgs.Addr carries no env fallback either, so there is no path to a
// default address at all). targ.Execute's returned error is a generic
// "exit code 1" here (targ.Main, the real binary's entry point, is what
// prints the detailed "missing required flag: --addr" — confirmed via a
// real `engram serve` invocation), so this only guards against the
// specific regression already caught once: a stray comma inside Addr's
// desc= struct-tag text broke targ's tag parser entirely, producing an
// "invalid tag" error instead of the required-flag one.
func TestServeTarget_RequiresAddr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stderr := executeForTest(t, []string{"engram", "serve"})
	g.Expect(stderr).NotTo(BeEmpty())
	g.Expect(stderr).NotTo(ContainSubstring("invalid tag"))
}

// TestServeTarget_ResolvesArgsAndCallsRunServe covers the serveTargets
// success path (task 3.1's other half — a valid --addr, not just the
// missing-flag refusal): Vault/VaultName/ChunksDir get resolved the same
// way every other target resolves them, then RunServe is reached. A fake
// ListenAndServe returns immediately so the test doesn't block on a real
// listener.
func TestServeTarget_ResolvesArgsAndCallsRunServe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	var gotAddr string

	stderr := executeForTestWithDeps(t, []string{"engram", "serve", "--addr", "127.0.0.1:0", "--vault", vault},
		func(d *cli.Deps) {
			d.NewServeMux = func() cli.RawServeMux { return "mux" }
			d.RegisterRoute = func(cli.RawServeMux, string, string, cli.ServeHandler) {}
			d.ListenAndServe = func(_ context.Context, _ cli.RawServeMux, addr string) error {
				gotAddr = addr

				return nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(gotAddr).To(Equal("127.0.0.1:0"))
}

// TestShowChunkFallback_LocalHitDoesNotContactParent mirrors
// TestShowFallback_LocalHitDoesNotContactParent for show-chunk: a locally
// resolvable chunk id is returned without ever consulting ENGRAM_PARENT.
func TestShowChunkFallback_LocalHitDoesNotContactParent(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	chunksDir := t.TempDir()
	records := []chunk.Record{
		{Source: "/s/a.jsonl", Anchor: "turn-1", ContentHash: "sha256:aa", Text: "local chunk text"},
	}

	data, encodeErr := chunk.EncodeRecords(records)
	g.Expect(encodeErr).NotTo(HaveOccurred())

	if encodeErr != nil {
		return
	}

	g.Expect(os.WriteFile(filepath.Join(chunksDir, "idx.jsonl"), data, 0o600)).To(Succeed())

	fetchCalled := false

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "show-chunk", "/s/a.jsonl#turn-1", "--chunks-dir", chunksDir}, func(d *cli.Deps) {
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetchCalled = true

				return cli.FetchResponse{}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
	g.Expect(stdout).To(Equal("local chunk text\n"))
}

// TestShowChunkFallback_LocalMissNoParentConfiguredErrorsUnchanged mirrors
// TestShowFallback_LocalMissNoParentConfiguredErrorsUnchanged for
// show-chunk.
func TestShowChunkFallback_LocalMissNoParentConfiguredErrorsUnchanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	chunksDir := t.TempDir()
	fetchCalled := false

	_, stderr := executeCapturingBoth(t,
		[]string{"engram", "show-chunk", "src.md#anchor", "--chunks-dir", chunksDir}, func(d *cli.Deps) {
			d.Getenv = func(string) string { return "" }
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetchCalled = true

				return cli.FetchResponse{}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("chunk not found"))
	g.Expect(fetchCalled).To(BeFalse())
}

// TestShowChunkFallback_LocalMissRoutesToParentLabeled mirrors
// TestShowFallback_LocalMissRoutesToParentLabeled for show-chunk.
func TestShowChunkFallback_LocalMissRoutesToParentLabeled(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	chunksDir := t.TempDir()

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "show-chunk", "src.md#anchor", "--chunks-dir", chunksDir}, func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url}

				return cli.FetchResponse{Status: 200, Body: []byte("parent chunk text\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(got.url).To(Equal("http://parent-host:8420/show-chunk?id=src.md%23anchor"))
	g.Expect(stdout).To(Equal("# from_parent: true\nparent chunk text\n"))
}

// TestShowChunkParent_RoutesThroughFetch mirrors TestShowParent_RoutesThroughFetch
// for `engram show-chunk --parent`.
func TestShowChunkParent_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "show-chunk", "src.md#anchor", "--parent"}, func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url}

				return cli.FetchResponse{Status: 200, Body: []byte("parent chunk text\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("parent chunk text\n"))
	g.Expect(got.url).To(Equal("http://parent-host:8420/show-chunk?id=src.md%23anchor"))
}

// TestShowChunkParent_WithoutEngramParentErrors mirrors
// TestShowParent_WithoutEngramParentErrors for show-chunk. Getenv is
// stubbed to "" for the same reason: t.Parallel() forbids t.Setenv, so
// this can't force ENGRAM_PARENT unset that way, and must not depend on
// the ambient environment actually leaving it unset.
func TestShowChunkParent_WithoutEngramParentErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fetchCalled := false

	_, stderr := executeCapturingBoth(t, []string{"engram", "show-chunk", "src.md#anchor", "--parent"},
		func(d *cli.Deps) {
			d.Getenv = func(string) string { return "" }
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetchCalled = true

				return cli.FetchResponse{}, nil
			}
		})

	g.Expect(stderr).NotTo(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
}

// TestShowChunkTarget_LocalDispatch covers show-chunk's local (non-served)
// branch through Targets(). Getenv is stubbed to "" for
// the same reason as TestTargets_QueryEmptyVault: this test means to
// exercise the local-only not-found path, and newTestDeps wires the real
// os.Getenv, so an ambient ENGRAM_PARENT would otherwise route the miss
// into dispatchShowChunk's vault-merged-recall fallback, which needs
// deps.Fetch — never wired here.
func TestShowChunkTarget_LocalDispatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "show-chunk", "missing#anchor", "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = func(string) string { return "" }
		})
	g.Expect(stderr).To(ContainSubstring("chunk not found"))
}

// TestShowFallback_LocalHitDoesNotContactParent covers "Without --parent,
// behavior is unchanged": with ENGRAM_PARENT configured but the ref found
// locally, `engram show` (no --parent) returns the local note and never
// contacts the parent — vault-merged-recall D8's fallback fires only on a
// local miss.
func TestShowFallback_LocalHitDoesNotContactParent(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	fetchCalled := false

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1", "--vault", vault}, func(d *cli.Deps) {
		d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
			fetchCalled = true

			return cli.FetchResponse{}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
	g.Expect(stdout).NotTo(ContainSubstring("from_parent"))
}

// TestShowFallback_LocalMissNoParentConfiguredErrorsUnchanged covers "Local
// miss with no parent configured is still an error": the same not-found
// error as before this capability existed, and the parent is never
// contacted. Getenv is stubbed to "" (t.Parallel() forbids t.Setenv) so this
// doesn't depend on ENGRAM_PARENT actually being unset in the ambient
// environment running the test.
func TestShowFallback_LocalMissNoParentConfiguredErrorsUnchanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	fetchCalled := false

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.missing", "--vault", vault},
		func(d *cli.Deps) {
			d.Getenv = func(string) string { return "" }
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetchCalled = true

				return cli.FetchResponse{}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("not found"))
	g.Expect(fetchCalled).To(BeFalse())
}

// TestShowFallback_LocalMissRoutesToParentLabeled covers "Local miss falls
// back to the parent": with ENGRAM_PARENT configured and
// no local match, bare `engram show <ref>` resolves the ref against the
// parent and labels the output as parent-sourced (vault-merged-recall D8).
func TestShowFallback_LocalMissRoutesToParentLabeled(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	vault := t.TempDir()

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.missing", "--vault", vault},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url}

				return cli.FetchResponse{Status: 200, Body: []byte("parent note content\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(got.url).To(Equal("http://parent-host:8420/show?note=1.missing"))
	g.Expect(stdout).To(Equal("# from_parent: true\nparent note content\n"))
}

// TestShowParent_NonOKResponse_MalformedBody_FallsBackToRawText covers
// describeErrorBody's non-JSON fallback branch: a non-2xx parent response
// whose body isn't {"error": "..."} JSON still surfaces its raw text.
func TestShowParent_NonOKResponse_MalformedBody_FallsBackToRawText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "missing-note", "--parent"},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv("http://parent-host:8420")
			d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{Status: 500, Body: []byte("  internal error, not json  ")}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("internal error, not json"))
}

// TestShowParent_NonOKResponse_SurfacesError covers the parent client's
// error path: a non-2xx parent response surfaces as a CLI error rather
// than being silently swallowed.
func TestShowParent_NonOKResponse_SurfacesError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "missing-note", "--parent"},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv("http://parent-host:8420")
			d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
				errBody, _ := json.Marshal(map[string]string{"error": "note not found"})

				return cli.FetchResponse{Status: 404, Body: errBody}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("note not found"))
}

// TestShowParent_PercentEncodesQueryValues covers the hand-rolled query
// encoder (internal/ may not import net/url): a note ref containing
// characters needing escaping round-trips correctly into the parent URL.
func TestShowParent_PercentEncodesQueryValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.2026-01-01.a note", "--parent"},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv("http://parent-host:8420")
			d.Fetch = func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url}

				return cli.FetchResponse{Status: 200, Body: []byte("note content\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("note content\n"))
	g.Expect(got.url).To(Equal("http://parent-host:8420/show?note=1.2026-01-01.a%20note"))
}

// TestShowParent_RoutesThroughFetch covers `engram show <ref> --parent`
// routing to the parent (ENGRAM_PARENT) via fetchShow.
func TestShowParent_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_PARENT", "http://parent-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.hub", "--parent"}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, method, url string, _ []byte) (cli.FetchResponse, error) {
			got = fakeFetchCall{method: method, url: url}

			return cli.FetchResponse{Status: 200, Body: []byte("parent note content\n")}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("parent note content\n"))
	g.Expect(got.url).To(Equal("http://parent-host:8420/show?note=1.hub"))
}

// TestShowParent_WithoutEngramParentErrors covers --parent passed with
// ENGRAM_PARENT unset: an error, no fetch attempted, no local fallback.
// Getenv is stubbed to "" (rather than t.Setenv, which panics after
// t.Parallel()) so this doesn't depend on ENGRAM_PARENT actually being
// unset in the ambient environment running the test.
func TestShowParent_WithoutEngramParentErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fetchCalled := false

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.hub", "--parent"}, func(d *cli.Deps) {
		d.Getenv = func(string) string { return "" }
		d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
			fetchCalled = true

			return cli.FetchResponse{}, nil
		}
	})

	g.Expect(stderr).NotTo(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
}

// TestShow_NoParent_RunsLocally covers the negative case: with no
// ENGRAM_PARENT, deps.Fetch is never invoked and the command runs against
// local files as usual.
func TestShow_NoParent_RunsLocally(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	fetchCalled := false

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "1", "--vault", vault}, func(d *cli.Deps) {
		d.Getenv = func(string) string { return "" }
		d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			fetchCalled = true

			return cli.FetchResponse{}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
}

// TestWriteParentSourced_BodyWriteErrorWraps covers writeParentSourced's
// second write-error branch: the marker line writes fine, but the body
// write fails.
func TestWriteParentSourced_BodyWriteErrorWraps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportWriteParentSourced(&writeNTimesThenFail{remaining: 1}, []byte("body"))
	g.Expect(err).To(MatchError(ContainSubstring("write response")))
}

// TestWriteParentSourced_MarkerWriteErrorWraps covers writeParentSourced's
// first write-error branch: the marker line itself fails to write.
func TestWriteParentSourced_MarkerWriteErrorWraps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportWriteParentSourced(&writeNTimesThenFail{remaining: 0}, []byte("body"))
	g.Expect(err).To(MatchError(ContainSubstring("write response")))
}

// unexported variables.
var (
	errWriteNTimesThenFail = errors.New("writeNTimesThenFail: write failed")
)

// fakeFetchCall records one deps.Fetch invocation for parent-client tests.
type fakeFetchCall struct {
	method string
	url    string
}

// writeNTimesThenFail succeeds on its first n calls to Write, then fails —
// used to exercise writeParentSourced's two write-error branches (marker,
// then body) independently.
type writeNTimesThenFail struct {
	remaining int
}

func (w *writeNTimesThenFail) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, errWriteNTimesThenFail
	}

	w.remaining--

	return len(p), nil
}

// executeCapturingBoth runs an engram CLI command through targ like
// executeForTestWithDeps, but returns BOTH stdout and stderr — needed here
// because parent reads print to stdout, unlike
// executeForTestWithDeps's error-path-only stderr capture.
func executeCapturingBoth(t failer, args []string, customize func(*cli.Deps)) (string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	if customize != nil {
		customize(&deps)
	}

	_, err := targ.Execute(args, cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stdout.String(), stderr.String()
}

// parentOnlyGetenv returns a Getenv stub that reports only
// ENGRAM_PARENT=parent, so a parallel test needs no t.Setenv and never
// depends on the ambient environment.
func parentOnlyGetenv(parent string) func(string) string {
	return func(key string) string {
		if key == "ENGRAM_PARENT" {
			return parent
		}

		return ""
	}
}
