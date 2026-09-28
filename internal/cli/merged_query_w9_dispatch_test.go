package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestShowChunk_NeverContactsParent: with ENGRAM_PARENT set, a local
// show-chunk miss is the local not-found error, and the parent is never
// contacted — chunks never cross vaults (Q1, E56).
func TestShowChunk_NeverContactsParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

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

	var fetches atomic.Int32

	run := func(ref string) (string, string) {
		return executeCapturingBoth(t, []string{"engram", "show-chunk", ref, "--chunks-dir", chunksDir},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
					fetches.Add(1)

					return cli.FetchResponse{Status: 200, Body: []byte("parent chunk text\n")}, nil
				}
			})
	}

	hitOut, hitErr := run("/s/a.jsonl#turn-1")
	g.Expect(hitErr).To(BeEmpty())
	g.Expect(hitOut).To(Equal("local chunk text\n"))

	missOut, missErr := run("src.md#anchor")
	g.Expect(missErr).To(ContainSubstring("chunk not found"))
	g.Expect(missOut).To(BeEmpty())
	g.Expect(fetches.Load()).To(BeZero())
}

// TestShowChunk_ParentFlagIsUnknown: `show-chunk --parent` is rejected as
// an unknown flag, with no request made.
func TestShowChunk_ParentFlagIsUnknown(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var (
		fetches        atomic.Int32
		stdout, stderr bytes.Buffer
	)

	deps := newTestDeps(&stdout, &stderr)
	deps.Getenv = parentOnlyGetenv(parentURL)
	deps.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
		fetches.Add(1)

		return cli.FetchResponse{Status: 200, Body: []byte("parent chunk text\n")}, nil
	}

	result, err := targ.Execute(
		[]string{"engram", "show-chunk", "src.md#anchor", "--parent", "--chunks-dir", t.TempDir()},
		cli.Targets(deps)...)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result.Output).To(ContainSubstring("--parent"))
	g.Expect(stdout.String()).To(BeEmpty())
	g.Expect(stderr.String()).NotTo(ContainSubstring("chunk not found"))
	g.Expect(fetches.Load()).To(BeZero())
}

// TestShowFallback_BackoffSkipsParent: a local miss inside the backoff
// window returns the local not-found error after the one warning, with no
// request.
func TestShowFallback_BackoffSkipsParent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	var fetches atomic.Int32

	clock := testNow
	run := func() (string, string) {
		return executeCapturingBoth(t, []string{"engram", "show", "1.missing", "--vault", vault},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Now = func() time.Time { return clock }
				d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
					fetches.Add(1)

					return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
				}
			})
	}

	_, _ = run()
	g.Expect(fetches.Load()).To(Equal(int32(1)))

	clock = clock.Add(10 * time.Second)
	_, stderr := run()
	g.Expect(fetches.Load()).To(Equal(int32(1)))
	g.Expect(stderr).To(ContainSubstring("engram: parent unreachable (retry after "))
	g.Expect(stderr).To(ContainSubstring("not found"))
}

// TestShowFallback_StampsVaultIDOnParentContact: the local-miss fallback is
// a parent contact, so it stamps a missing vault ID (ruling S3).
func TestShowFallback_StampsVaultIDOnParentContact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	_, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.missing", "--vault", vault},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{Status: 200, Body: []byte("parent note\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(filepath.Join(vault, ".engram-vault-id")).To(BeARegularFile())
}

// TestShowParent_BackoffMakesNoRequest: inside the backoff window,
// `show --parent` makes no request and errors after the one warning.
func TestShowParent_BackoffMakesNoRequest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	var fetches atomic.Int32

	clock := testNow
	run := func() (string, string) {
		return executeCapturingBoth(t, []string{"engram", "show", "1.hub", "--parent", "--vault", vault},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Now = func() time.Time { return clock }
				d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
					fetches.Add(1)

					return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
				}
			})
	}

	_, firstErr := run()
	g.Expect(firstErr).NotTo(BeEmpty())
	g.Expect(fetches.Load()).To(Equal(int32(1)))

	clock = clock.Add(10 * time.Second)
	stdout, stderr := run()
	g.Expect(fetches.Load()).To(Equal(int32(1)))
	g.Expect(stdout).To(BeEmpty())
	g.Expect(stderr).To(ContainSubstring("engram: parent unreachable (retry after "))
}

// TestShowParent_FailedStampSkipsContactBookkeeping: when the vault ID
// can't be stamped, the contact still runs but its bookkeeping is skipped,
// so no later write creates .engram/ without a stamp (ruling S4).
func TestShowParent_FailedStampSkipsContactBookkeeping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	var fetches atomic.Int32

	_, _ = executeCapturingBoth(t, []string{"engram", "show", "1.hub", "--parent", "--vault", vault},
		func(d *cli.Deps) {
			d.FS = &failFirstIDWriteFS{EdgeFS: d.FS}
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetches.Add(1)

				return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
			}
		})

	g.Expect(fetches.Load()).To(Equal(int32(1)))
	g.Expect(filepath.Join(vault, ".engram")).NotTo(BeAnExistingFile())
	g.Expect(filepath.Join(vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
}

// TestShowParent_StampsVaultIDOnParentContact: `show --parent` is a parent
// contact, so it stamps a missing vault ID (ruling S3).
func TestShowParent_StampsVaultIDOnParentContact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1.hub", "--parent", "--vault", vault},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{Status: 200, Body: []byte("parent note content\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(Equal("parent note content\n"))
	g.Expect(filepath.Join(vault, ".engram-vault-id")).To(BeARegularFile())
}

// TestTargets_Query_Merged_DedupesLinkedLocalNote wires the dedupe end to
// end: a local note linked to the parent's note appears once, as the
// local copy, with no parent chunk in the payload.
func TestTargets_Query_Merged_DedupesLinkedLocalNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.2026-09-01.local.md",
		"---\ntype: fact\ntier: L2\nsituation: x\nparent:\n    vault: "+w9ParentVaultID+
			"\n    links:\n        - note: 900.2026-09-01.parent\n          via: pulled\n---\n\nlocal body\n",
		[]float32{1, 0, 0, 0})

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				return cli.FetchResponse{Status: 200, Body: []byte(
					"version: 1\nvault_id: " + w9ParentVaultID + "\nmodel_id: m@4\nitems:\n" +
						"  - path: 900.2026-09-01.parent.md\n    kind: fact\n    score: 0.9\n" +
						"    provenances: [direct]\n    exchange_hash: xh1:unrelated\n" +
						"  - path: /s/p.jsonl#turn-1\n    kind: chunk\n    score: 0.95\n    provenances: [direct]\n",
				)}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).NotTo(ContainSubstring("exchange_hash"))
	g.Expect(stdout).NotTo(ContainSubstring("vault_id"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
	g.Expect(parsed.Items[0].Path).To(Equal("1.2026-09-01.local.md"))
}

// TestTargets_Query_Merged_FailedStampSkipsContactBookkeeping is the merged
// query's counterpart: the parent is still asked, but no backoff record or
// drain creates .engram/ without a stamp (ruling S4).
func TestTargets_Query_Merged_FailedStampSkipsContactBookkeeping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	var fetches atomic.Int32

	stdout, _ := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.FS = &failFirstIDWriteFS{EdgeFS: d.FS}
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				fetches.Add(1)

				return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
			}
		})

	g.Expect(fetches.Load()).To(Equal(int32(1)))
	g.Expect(stdout).To(ContainSubstring("1.fact.md"))
	g.Expect(filepath.Join(vault, ".engram")).NotTo(BeAnExistingFile())
	g.Expect(filepath.Join(vault, ".engram-vault-id")).NotTo(BeAnExistingFile())
}

// TestTargets_Query_Merged_RequestsDedupeKeysAndCachesVaultID: the merged
// query asks the parent for dedupe keys and caches the vault_id it
// reports, so the drain needs no separate probe (W7 carry-over).
func TestTargets_Query_Merged_RequestsDedupeKeysAndCachesVaultID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	urls := make([]string, 0, 1)

	_, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(_ context.Context, _, url string, _ []byte) (cli.FetchResponse, error) {
				urls = append(urls, url)

				return cli.FetchResponse{Status: 200, Body: []byte(
					"version: 1\nvault_id: " + w9ParentVaultID + "\nmodel_id: m@4\nitems: []\n")}, nil
			}
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(urls).To(HaveLen(1))

	if len(urls) != 1 {
		return
	}

	g.Expect(urls[0]).To(ContainSubstring("dedupe-keys=1"))

	cache, cacheErr := cli.ExportLoadParentCache(
		cli.ExportExchangeStateFromDeps(newTestDeps(io.Discard, io.Discard)), vault)
	g.Expect(cacheErr).NotTo(HaveOccurred())
	g.Expect(cache.VaultID).To(Equal(w9ParentVaultID))
}

// TestTargets_Query_Merged_SelfParentReturnsLocalOnly: a parent reporting
// this vault's own ID is not merged; the query returns local results with
// the self-parent warning.
func TestTargets_Query_Merged_SelfParentReturnsLocalOnly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
			d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
				ownID, readErr := os.ReadFile(filepath.Join(vault, ".engram-vault-id"))
				if readErr != nil {
					return cli.FetchResponse{}, readErr
				}

				return cli.FetchResponse{Status: 200, Body: []byte(
					"version: 1\nvault_id: " + strings.TrimSpace(string(ownID)) + "\nmodel_id: m@4\nitems:\n" +
						"  - path: parent-note.md\n    kind: fact\n    score: 0.5\n    provenances: [direct]\n",
				)}, nil
			}
		})

	g.Expect(stderr).To(ContainSubstring("the parent reports this vault's own ID"))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
}

// unexported constants.
const (
	w9ParentVaultID = "9a1e9a1e9a1e9a1e9a1e9a1e9a1e9a1e"
)

// unexported variables.
var (
	errStampTransient = errors.New("transient: disk busy")
)

// failFirstIDWriteFS fails the first exclusive create of the vault ID file
// (a transient stamp failure) and passes every other call through.
type failFirstIDWriteFS struct {
	cli.EdgeFS

	failed atomic.Bool
}

func (f *failFirstIDWriteFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	if filepath.Base(path) == ".engram-vault-id" && f.failed.CompareAndSwap(false, true) {
		return errStampTransient
	}

	return f.EdgeFS.WriteFileExcl(path, data, perm)
}
