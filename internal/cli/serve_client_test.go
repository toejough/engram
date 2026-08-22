package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestEngramServer_Activate_RoutesThroughFetch covers `engram activate`
// client-mode dispatch (a write with no receipt to print — commits
// directly, per design.md Decisions).
func TestEngramServer_Activate_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "activate", "--note", "1.a.md"}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			got = fakeFetchCall{method: method, url: url, body: body}

			return cli.FetchResponse{Status: 200, Body: []byte(`{"status":"ok"}`)}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(BeEmpty())
	g.Expect(got.method).To(Equal("POST"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/activate"))
	g.Expect(string(got.body)).To(ContainSubstring("1.a.md"))
}

// TestEngramServer_Amend_DiscardRefusedLocally covers the client-side guard:
// `--discard` is refused before any network call when ENGRAM_SERVER is set,
// rather than being POSTed to a server that would silently force it off
// (serveAmend) and re-pend the target note instead of discarding it.
func TestEngramServer_Amend_DiscardRefusedLocally(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	fetchCalled := false

	_, stderr := executeCapturingBoth(t,
		[]string{"engram", "amend", "--target", "1", "--discard"},
		func(d *cli.Deps) {
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetchCalled = true

				return cli.FetchResponse{}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("host-local"))
	g.Expect(fetchCalled).To(BeFalse())
}

// TestEngramServer_Amend_StampsRepoAndPrintsReceipt mirrors
// TestEngramServer_Learn_StampsRepoAndPrintsReceipt for `engram amend`.
func TestEngramServer_Amend_StampsRepoAndPrintsReceipt(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "amend", "--target", "1", "--object", "amended"},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url, body: body}
				receipt, _ := json.Marshal(map[string]string{"status": "offer received", "luhmann": "1"})

				return cli.FetchResponse{Status: 200, Body: receipt}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("offer received: 1\n"))
	g.Expect(got.method).To(Equal("POST"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/amend"))

	var sent cli.AmendArgs
	g.Expect(json.Unmarshal(got.body, &sent)).To(Succeed())
	g.Expect(sent.Object).To(Equal("amended"))
}

// TestEngramServer_Learn_StampsRepoAndPrintsReceipt covers task 8.1 for a
// write command: `engram learn fact` with ENGRAM_SERVER set POSTs
// LearnArgs-shaped JSON (including this client's own detected repo:) and
// prints the offer receipt rather than a note path.
func TestEngramServer_Learn_StampsRepoAndPrintsReceipt(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{
		"engram", "learn", "fact",
		"--slug", "served", "--source", "test",
		"--situation", "a served write", "--subject", "engram", "--predicate", "serves", "--object", "writes",
	}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			got = fakeFetchCall{method: method, url: url, body: body}
			receipt, _ := json.Marshal(map[string]string{"status": "offer received", "luhmann": "9"})

			return cli.FetchResponse{Status: 200, Body: receipt}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("offer received: 9\n"))
	g.Expect(got.method).To(Equal("POST"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/learn"))

	var sent cli.LearnArgs
	g.Expect(json.Unmarshal(got.body, &sent)).To(Succeed())
	g.Expect(sent.Situation).To(Equal("a served write"))
	g.Expect(sent.Type).To(Equal("fact"))
}

// TestEngramServer_MalformedReceipt_SurfacesDecodeError covers
// printOfferReceipt's error branch: a served write's success response that
// isn't valid {status,luhmann} JSON surfaces as a client-side error rather
// than being silently mis-printed.
func TestEngramServer_MalformedReceipt_SurfacesDecodeError(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	_, stderr := executeCapturingBoth(t, []string{
		"engram", "learn", "fact",
		"--slug", "served", "--source", "test",
		"--situation", "s", "--subject", "a", "--predicate", "b", "--object", "c",
	}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			return cli.FetchResponse{Status: 200, Body: []byte("not json")}, nil
		}
	})

	g.Expect(stderr).To(ContainSubstring("decoding offer receipt"))
}

// TestEngramServer_NonOKResponse_MalformedBody_FallsBackToRawText covers
// describeErrorBody's non-JSON fallback branch: a non-2xx response whose
// body isn't {"error": "..."} JSON still surfaces its raw text.
func TestEngramServer_NonOKResponse_MalformedBody_FallsBackToRawText(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "missing-note"}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			return cli.FetchResponse{Status: 500, Body: []byte("  internal error, not json  ")}, nil
		}
	})

	g.Expect(stderr).To(ContainSubstring("internal error, not json"))
}

// TestEngramServer_NonOKResponse_SurfacesError covers the client-side
// error path: a non-2xx served response surfaces as a CLI error rather
// than being silently swallowed.
func TestEngramServer_NonOKResponse_SurfacesError(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "missing-note"}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			errBody, _ := json.Marshal(map[string]string{"error": "note not found"})

			return cli.FetchResponse{Status: 404, Body: errBody}, nil
		}
	})

	g.Expect(stderr).To(ContainSubstring("note not found"))
}

// TestEngramServer_QueryChunks_RoutesThroughFetch covers `engram
// query-chunks` client-mode dispatch.
func TestEngramServer_QueryChunks_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query-chunks", "--phrase", "hello", "--limit", "3"},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url, body: body}

				return cli.FetchResponse{Status: 200, Body: []byte("items: []\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("items: []\n"))
	g.Expect(got.method).To(Equal("GET"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/query-chunks?limit=3&phrase=hello"))
}

// TestEngramServer_Query_AllParamsSet exercises every setBoolParam/
// setIntParam/setStringParam "value present" branch in one call (the
// negative/absent branches are already covered by
// TestEngramServer_Query_RoutesThroughFetch, which sets none of them).
func TestEngramServer_Query_AllParamsSet(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{
		"engram", "query", "--phrase", "hello",
		"--limit", "7", "--project", "engram", "--content-budget", "9", "--recent-fill", "2",
		"--lazy-chunks", "--timings",
	}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			got = fakeFetchCall{method: method, url: url, body: body}

			return cli.FetchResponse{Status: 200, Body: []byte("ok\n")}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("ok\n"))
	g.Expect(got.url).To(Equal(
		"http://vault-host:8420/query?content-budget=9&lazy-chunks=true&limit=7" +
			"&phrase=hello&project=engram&recent-fill=2&timings=true",
	))
}

// TestEngramServer_Query_RoutesThroughFetch covers task 8.1: with
// ENGRAM_SERVER set, `engram query` issues a GET /query request instead of
// touching local files, and copies the server's response body verbatim to
// stdout (design.md API Contract: byte-identical to local).
func TestEngramServer_Query_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "query", "--phrase", "hello world"},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url, body: body}

				return cli.FetchResponse{Status: 200, Body: []byte("version: 1\nitems: []\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("version: 1\nitems: []\n"), "GET response body is copied verbatim")
	g.Expect(got.method).To(Equal("GET"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/query?phrase=hello%20world"))
}

// TestEngramServer_ShowChunk_RoutesThroughFetch covers `engram show-chunk`
// client-mode dispatch.
func TestEngramServer_ShowChunk_RoutesThroughFetch(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show-chunk", "src.md#anchor"}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
			got = fakeFetchCall{method: method, url: url, body: body}

			return cli.FetchResponse{Status: 200, Body: []byte("chunk text\n")}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("chunk text\n"))
	g.Expect(got.method).To(Equal("GET"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/show-chunk?id=src.md%23anchor"))
}

// TestEngramServer_Show_PercentEncodesQueryValues covers the hand-rolled
// query encoder (internal/ may not import net/url): a note ref containing
// characters needing escaping round-trips correctly into the URL.
func TestEngramServer_Show_PercentEncodesQueryValues(t *testing.T) {
	g := NewWithT(t)
	t.Setenv("ENGRAM_SERVER", "http://vault-host:8420")

	var got fakeFetchCall

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.2026-01-01.a note"},
		func(d *cli.Deps) {
			d.Fetch = func(_ context.Context, method, url string, body []byte) (cli.FetchResponse, error) {
				got = fakeFetchCall{method: method, url: url, body: body}

				return cli.FetchResponse{Status: 200, Body: []byte("note content\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("note content\n"))
	g.Expect(got.url).To(Equal("http://vault-host:8420/show?note=1.2026-01-01.a%20note"))
}

// TestEngramServer_Unset_RunsLocally covers the negative case: with
// ENGRAM_SERVER unset, deps.Fetch is never invoked and the command runs
// against local files as usual.
func TestEngramServer_Unset_RunsLocally(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	fetchCalled := false

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "1", "--vault", vault}, func(d *cli.Deps) {
		d.Fetch = func(_ context.Context, _, _ string, _ []byte) (cli.FetchResponse, error) {
			fetchCalled = true

			return cli.FetchResponse{}, nil
		}
	})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(fetchCalled).To(BeFalse())
}

// TestLocalAmend_ClearPendingClearsTheMarker covers the curation skill's
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

// TestLocalAmend_DiscardDeletesNoteAndSidecar covers the curation skill's
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

// TestServerBase_NilGetenv covers serverBase's nil-safety guard (a minimal
// zero-value Deps, as production code can construct when Getenv is
// unwired — matches the same nil-tolerance convention as homeOrEmpty).
func TestServerBase_NilGetenv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportServerBase(cli.Deps{})).To(Equal(""))
}

// TestShowChunkTarget_LocalDispatch covers show-chunk's local (non-served)
// branch through Targets() — the ENGRAM_SERVER-set branch is covered by
// TestEngramServer_ShowChunk_RoutesThroughFetch.
func TestShowChunkTarget_LocalDispatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stderr := executeForTest(t, []string{"engram", "show-chunk", "missing#anchor", "--chunks-dir", t.TempDir()})
	g.Expect(stderr).To(ContainSubstring("chunk not found"))
}

// fakeFetchCall records one deps.Fetch invocation for ENGRAM_SERVER-mode
// client tests.
type fakeFetchCall struct {
	method string
	url    string
	body   []byte
}

// executeCapturingBoth runs an engram CLI command through targ like
// executeForTestWithDeps, but returns BOTH stdout and stderr — needed here
// because ENGRAM_SERVER-mode reads/receipts print to stdout, unlike
// executeForTestWithDeps's error-path-only stderr capture.
func executeCapturingBoth(t *testing.T, args []string, customize func(*cli.Deps)) (string, string) {
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
