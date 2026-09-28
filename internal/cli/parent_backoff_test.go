package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestBackoffDelay_ExponentialFrom30sCappedAt15m (design D6, M9).
func TestBackoffDelay_ExponentialFrom30sCappedAt15m(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(cli.ExportBackoffDelay(1)).To(Equal(30 * time.Second))
	g.Expect(cli.ExportBackoffDelay(2)).To(Equal(time.Minute))
	g.Expect(cli.ExportBackoffDelay(3)).To(Equal(2 * time.Minute))
	g.Expect(cli.ExportBackoffDelay(4)).To(Equal(4 * time.Minute))
	g.Expect(cli.ExportBackoffDelay(5)).To(Equal(8 * time.Minute))
	g.Expect(cli.ExportBackoffDelay(6)).To(Equal(15 * time.Minute))
	g.Expect(cli.ExportBackoffDelay(1000)).To(Equal(15 * time.Minute))
}

// TestBackoffDelay_Property: for any failure count ≥ 1 the delay is within
// [30s, 15m], never shrinks as failures grow, and doubles until the cap.
func TestBackoffDelay_Property(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		failures := rapid.IntRange(1, 1<<20).Draw(rt, "failures")
		delay := cli.ExportBackoffDelay(failures)
		next := cli.ExportBackoffDelay(failures + 1)

		if delay < 30*time.Second || delay > 15*time.Minute {
			rt.Fatalf("delay(%d) = %v outside [30s, 15m]", failures, delay)
		}

		if next < delay {
			rt.Fatalf("delay shrank: %v then %v", delay, next)
		}

		if next != 15*time.Minute && next != 2*delay {
			rt.Fatalf("below the cap the delay doubles: %v then %v", delay, next)
		}
	})
}

// TestGateParentContact_AllowsAfterWindow: once backoff_until passes, the
// parent is contacted again, silently.
func TestGateParentContact_AllowsAfterWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	_, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())

	env.advance(31 * time.Second)
	g.Expect(cli.ExportGateParentContact(env.store, env.vault, parentURL, false)).To(BeTrue())
	g.Expect(env.stderr.String()).To(BeEmpty())
}

// TestGateParentContact_BlocksInsideWindowWithOneWarning: inside the window
// the command skips the parent and prints exactly one warning naming the
// retry time and the queued-offer count.
func TestGateParentContact_BlocksInsideWindowWithOneWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	env.writeNote("1.2026-09-27.a.md", xidA, "a", false)
	env.writeNote("2.2026-09-27.b.md", xidB, "b", false)
	env.enqueue(xidA)
	env.enqueue(xidB)

	retry, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())

	env.advance(10 * time.Second)
	g.Expect(cli.ExportGateParentContact(env.store, env.vault, parentURL, false)).To(BeFalse())

	lines := nonEmptyLines(env.stderr.String())
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(Equal(
		"engram: parent unreachable (retry after " + retry.Format(time.RFC3339) + "); 2 offer(s) queued"))
}

// TestGateParentContact_OtherParentURLIsNotBackedOff: the backoff belongs
// to the URL that failed; a reconfigured parent is tried.
func TestGateParentContact_OtherParentURLIsNotBackedOff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	_, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(cli.ExportGateParentContact(env.store, env.vault, "http://other:8093", false)).To(BeTrue())
}

// TestGateParentContact_UpdateIgnoresWindow: `engram update` always tries.
func TestGateParentContact_UpdateIgnoresWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	_, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(cli.ExportGateParentContact(env.store, env.vault, parentURL, true)).To(BeTrue())
	g.Expect(env.stderr.String()).To(BeEmpty())
}

// TestRecordParentFailure_GrowsThenSuccessResets: consecutive failures grow
// the window; a success resets the count so the next failure is 30s again.
func TestRecordParentFailure_GrowsThenSuccessResets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)

	first, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(first).To(BeTemporally("==", env.now().Add(30*time.Second)))

	second, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(second).To(BeTemporally("==", env.now().Add(time.Minute)))

	g.Expect(cli.ExportRecordParentSuccess(env.store, env.vault, parentURL, parentVaultID)).To(Succeed())

	cache, cacheErr := cli.ExportLoadParentCache(env.state, env.vault)
	g.Expect(cacheErr).NotTo(HaveOccurred())
	g.Expect(cache.Failures).To(BeZero())
	g.Expect(cache.VaultID).To(Equal(parentVaultID))
	g.Expect(cli.ExportGateParentContact(env.store, env.vault, parentURL, false)).To(BeTrue())

	third, err := cli.ExportRecordParentFailure(env.store, env.vault, parentURL)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(third).To(BeTemporally("==", env.now().Add(30*time.Second)))
}

// TestRecordParentSuccess_IdleWritesNothing: a success with nothing to
// reset or learn does not rewrite parent.json (no churn per query).
func TestRecordParentSuccess_IdleWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	env := newOutboxEnv(t)
	g.Expect(cli.ExportRecordParentSuccess(env.store, env.vault, parentURL, parentVaultID)).To(Succeed())

	writes := len(env.fsys.atomicWrites())
	g.Expect(cli.ExportRecordParentSuccess(env.store, env.vault, parentURL, parentVaultID)).To(Succeed())
	g.Expect(env.fsys.atomicWrites()).To(HaveLen(writes))
}

// TestTargets_Query_BackoffSkipsParentInsideWindow (E51, M9): a query that
// fails to reach the parent backs off; a second query 10s later makes no
// parent request, returns local results, and prints exactly one warning.
func TestTargets_Query_BackoffSkipsParentInsideWindow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	var fetches atomic.Int32

	clock := testNow
	run := func() (string, string) {
		return executeCapturingBoth(t,
			[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Now = func() time.Time { return clock }
				d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
				d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
					fetches.Add(1)

					return cli.FetchResponse{}, errors.New("dial tcp: connection refused")
				}
			})
	}

	_, firstErr := run()
	g.Expect(fetches.Load()).To(Equal(int32(1)))
	g.Expect(nonEmptyLines(firstErr)).To(HaveLen(1))

	clock = clock.Add(10 * time.Second)
	stdout, stderr := run()
	g.Expect(fetches.Load()).To(Equal(int32(1)))

	lines := nonEmptyLines(stderr)
	g.Expect(lines).To(HaveLen(1))
	g.Expect(lines[0]).To(HavePrefix("engram: parent unreachable (retry after "))

	var parsed queryParsed
	g.Expect(yaml.Unmarshal([]byte(stdout), &parsed)).To(Succeed())
	g.Expect(parsed.Items).To(HaveLen(1))
	g.Expect(parsed.Items[0].Path).To(Equal("1.fact.md"))
}

// TestTargets_Query_ParentClientErrorDoesNotBackOff: a 4xx is an answer,
// not an unreachable parent — the next query tries again.
func TestTargets_Query_ParentClientErrorDoesNotBackOff(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	plantRealVaultNote(t, vault, "1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: x\n---\n\nlocal body\n", []float32{1, 0, 0, 0})

	var fetches atomic.Int32

	run := func() {
		_, _ = executeCapturingBoth(t,
			[]string{"engram", "query", "--phrase", "x", "--vault", vault, "--chunks-dir", t.TempDir()},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Now = func() time.Time { return testNow }
				d.Embed = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
				d.Fetch = func(context.Context, string, string, []byte) (cli.FetchResponse, error) {
					fetches.Add(1)

					return cli.FetchResponse{Status: 400, Body: []byte(`{"error":"bad query"}`)}, nil
				}
			})
	}

	run()
	run()
	g.Expect(fetches.Load()).To(Equal(int32(2)))

	raw, readErr := os.ReadFile(filepath.Join(vault, ".engram", "parent.json"))
	if readErr == nil {
		g.Expect(string(raw)).NotTo(ContainSubstring(`"failures":1`))
	}
}
