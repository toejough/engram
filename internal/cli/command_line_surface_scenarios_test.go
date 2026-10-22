package cli_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
)

// Test792b2c10a11_ShowChunk_NeverReachesTheParent verifies 792b2c10a11 (command-line-surface.md): with
// ENGRAM_PARENT set, `show-chunk <id> --parent` is rejected as an unknown flag with no request, and
// `show-chunk <id>` for a missing chunk returns the local not-found error with no request.
func Test792b2c10a11_ShowChunk_NeverReachesTheParent(t *testing.T) {
	t.Parallel()

	const chunkRef = "src.md#anchor"

	t.Run("the parent flag is an unknown flag and no request is made", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		var stdout, stderr bytes.Buffer

		recorder := &cls1FetchRecorder{answer: "parent chunk text\n"}
		deps := newTestDeps(&stdout, &stderr)
		deps.Getenv = parentOnlyGetenv(parentURL)
		deps.Fetch = recorder.fetch

		result, err := targ.Execute(
			[]string{"engram", "show-chunk", chunkRef, "--parent", "--chunks-dir", t.TempDir()},
			cli.Targets(deps)...)

		g.Expect(err).To(HaveOccurred())
		g.Expect(result.Output).To(ContainSubstring("flag provided but not defined: --parent"))
		g.Expect(stdout.String()).To(BeEmpty())
		g.Expect(recorder.calls()).To(BeEmpty())
	})

	t.Run("a missing chunk is the local not-found error and no request is made", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		recorder := &cls1FetchRecorder{answer: "parent chunk text\n"}

		stdout, stderr := executeCapturingBoth(t,
			[]string{"engram", "show-chunk", chunkRef, "--chunks-dir", t.TempDir()},
			func(d *cli.Deps) {
				d.Getenv = parentOnlyGetenv(parentURL)
				d.Fetch = recorder.fetch
			})

		g.Expect(stderr).To(ContainSubstring("chunk not found"))
		g.Expect(stdout).To(BeEmpty())
		g.Expect(recorder.calls()).To(BeEmpty())
	})
}

// Test792b2c10a2_ShowParent_WithNoParentIsAnError verifies 792b2c10a2: with ENGRAM_PARENT unset,
// `engram show <ref> --parent` fails, makes no request and runs no local fallback (the note is
// present locally and still is not printed).
func Test792b2c10a2_ShowParent_WithNoParentIsAnError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	recorder := &cls1FetchRecorder{answer: "parent note\n"}

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "show", "1", "--parent", "--vault", vault},
		func(d *cli.Deps) {
			d.Getenv = func(string) string { return "" }
			d.Fetch = recorder.fetch
		})

	g.Expect(stderr).To(ContainSubstring("--parent requires ENGRAM_PARENT to be configured"))
	g.Expect(stdout).To(BeEmpty(), "no local fallback runs")
	g.Expect(recorder.calls()).To(BeEmpty(), "no request is made")
}

// Test792b2c10a3_Show_ALocalHitDoesNotContactTheParent verifies 792b2c10a3: with ENGRAM_PARENT set and
// the note present locally, `engram show <ref>` prints the local note and makes no request.
func Test792b2c10a3_Show_ALocalHitDoesNotContactTheParent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := t.TempDir()
	notePath := writeServeVaultFile(t, vault, "1.2026-01-01.a-note.md")

	localBody, readErr := os.ReadFile(notePath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	recorder := &cls1FetchRecorder{answer: "parent note\n"}

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "1", "--vault", vault},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = recorder.fetch
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(stdout).To(ContainSubstring(strings.TrimSpace(string(localBody))))
	g.Expect(stdout).NotTo(ContainSubstring("parent note"))
	g.Expect(recorder.calls()).To(BeEmpty(), "no request is made")
}

// Test792b2c10a4_Show_ALocalMissFallsBackLabeled verifies 792b2c10a4: with ENGRAM_PARENT set and the
// note not local, the parent's answer is printed after the line "# from_parent: true", and the label
// precedes the parent note's exchange-hash header.
func Test792b2c10a4_Show_ALocalMissFallsBackLabeled(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		parentAnswer = "# exchange_hash: abc123\nparent note body\n"
		wantRequest  = parentURL + "/show?note=1.missing"
	)

	recorder := &cls1FetchRecorder{answer: parentAnswer}

	stdout, stderr := executeCapturingBoth(t,
		[]string{"engram", "show", "1.missing", "--vault", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = recorder.fetch
		})

	g.Expect(stderr).To(BeEmpty())
	g.Expect(recorder.calls()).To(ConsistOf(wantRequest))
	g.Expect(stdout).To(Equal("# from_parent: true\n" + parentAnswer))
	g.Expect(strings.Index(stdout, "# from_parent: true")).
		To(BeNumerically("<", strings.Index(stdout, "# exchange_hash:")))
}

// Test792b2c10a6_Show_AnotherLocalErrorDoesNotReachTheParent verifies 792b2c10a6: with ENGRAM_PARENT
// set, `engram show` with a blank reference fails with the local error and makes no request.
func Test792b2c10a6_Show_AnotherLocalErrorDoesNotReachTheParent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	recorder := &cls1FetchRecorder{answer: "parent note\n"}

	stdout, stderr := executeCapturingBoth(t, []string{"engram", "show", "  ", "--vault", t.TempDir()},
		func(d *cli.Deps) {
			d.Getenv = parentOnlyGetenv(parentURL)
			d.Fetch = recorder.fetch
		})

	g.Expect(stderr).To(ContainSubstring("empty note reference"))
	g.Expect(stdout).To(BeEmpty())
	g.Expect(recorder.calls()).To(BeEmpty(), "no request is made")
}

// Test792b2c2a2_ResolveVaultName_TheEnvironmentNamesTheVault verifies 792b2c2a2: with --vault-name
// empty and ENGRAM_VAULT_NAME team, the resolved vault name is team.
func Test792b2c2a2_ResolveVaultName_TheEnvironmentNamesTheVault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	getenv := cls1EnvOf(map[string]string{"ENGRAM_VAULT_NAME": "team"})

	g.Expect(cli.ExportResolveVaultName("", getenv)).To(Equal("team"))
}

// Test792b2c2a3_ResolveVaultName_TheFlagWins verifies 792b2c2a3: with --vault-name work and
// ENGRAM_VAULT_NAME personal-env, the resolved vault name is work.
func Test792b2c2a3_ResolveVaultName_TheFlagWins(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	getenv := cls1EnvOf(map[string]string{"ENGRAM_VAULT_NAME": "personal-env"})

	g.Expect(cli.ExportResolveVaultName("work", getenv)).To(Equal("work"))
}

// Test792b2c4a2_NewErrHandler_ASuccessNeitherExitsNorWritesStderr verifies 792b2c4a2: a command that
// succeeds, reported through the handler Targets builds, writes nothing to stderr and requests no exit.
func Test792b2c4a2_NewErrHandler_ASuccessNeitherExitsNorWritesStderr(t *testing.T) {
	t.Parallel()

	t.Run("the handler given a nil error", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		var (
			stderr bytes.Buffer
			exits  []int
		)

		handler := cli.ExportNewErrHandler(&stderr, func(code int) { exits = append(exits, code) })
		handler(nil)

		g.Expect(stderr.String()).To(BeEmpty())
		g.Expect(exits).To(BeEmpty())
	})

	t.Run("a succeeding command dispatched through Targets", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		var exits []int

		stdout, stderr := executeCapturingBoth(t, []string{"engram", "check", "--vault", t.TempDir()},
			func(d *cli.Deps) {
				d.Getenv = func(string) string { return "" }
				d.Exit = func(code int) { exits = append(exits, code) }
			})

		g.Expect(stdout).NotTo(BeEmpty(), "the command ran")
		g.Expect(stderr).To(BeEmpty())
		g.Expect(exits).To(BeEmpty())
	})
}

// Test792b2c5a2_Targets_AMissingRequiredFlagIsRefused verifies 792b2c5a2: `learn runbook` marks --done-when
// required; dispatched without it (and otherwise a valid runbook), the exit status is non-zero and the command does
// not run, so nothing is printed and nothing is written to the vault.
func Test792b2c5a2_Targets_AMissingRequiredFlagIsRefused(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vault := newCV1Vault()
	before := vault.fsys.snapshot()

	args := append([]string{"learn", "runbook"}, nw1RunbookFlags().without("--done-when")...)

	result, err := vault.dispatch(args...)

	g.Expect(err).To(HaveOccurred(), "the exit status is non-zero")
	g.Expect(result.stdout).To(BeEmpty(), "the command does not run")
	g.Expect(vault.fsys.snapshot()).To(Equal(before), "nothing is written to the vault")
}

// Test792b2c6a2_ResolveChunksDir_TheEnvironmentIsUsedWithoutAFlag verifies 792b2c6a2: with --chunks-dir
// empty and ENGRAM_CHUNKS_DIR /from/env, the directory is /from/env.
func Test792b2c6a2_ResolveChunksDir_TheEnvironmentIsUsedWithoutAFlag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	getenv := cls1EnvOf(map[string]string{"ENGRAM_CHUNKS_DIR": "/from/env", "XDG_DATA_HOME": "/xdg/data"})

	g.Expect(cli.ResolveChunksDir("", "/home/dev", getenv)).To(Equal("/from/env"))
}

// Test792b2c6a3_ResolveChunksDir_XDGDataHomeSetsTheDefault verifies 792b2c6a3: with no flag, no
// ENGRAM_CHUNKS_DIR and XDG_DATA_HOME /xdg/data, the directory is /xdg/data/engram/chunks.
func Test792b2c6a3_ResolveChunksDir_XDGDataHomeSetsTheDefault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	getenv := cls1EnvOf(map[string]string{"XDG_DATA_HOME": "/xdg/data"})

	g.Expect(cli.ResolveChunksDir("", "/home/dev", getenv)).To(Equal("/xdg/data/engram/chunks"))
}

// Test792b2c6a4_ResolveChunksDir_TheHomeDirectoryIsTheLastFallback verifies 792b2c6a4: with no flag, no
// ENGRAM_CHUNKS_DIR, no XDG_DATA_HOME and home /home/dev, the directory is
// /home/dev/.local/share/engram/chunks.
func Test792b2c6a4_ResolveChunksDir_TheHomeDirectoryIsTheLastFallback(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(cli.ResolveChunksDir("", "/home/dev", cls1EnvOf(nil))).
		To(Equal("/home/dev/.local/share/engram/chunks"))
}

// Test792b2c9a1_Targets_EveryCommandIsRefusedBeforeAnythingIsTouched verifies 792b2c9a1: with
// ENGRAM_SERVER set to a URL, every subcommand (serve included) exits non-zero with a message naming
// ENGRAM_PARENT and the same URL, and no filesystem, lock, fetch, bind or command capability is used.
func Test792b2c9a1_Targets_EveryCommandIsRefusedBeforeAnythingIsTouched(t *testing.T) {
	t.Parallel()

	const serverURL = "http://old-host:8093"

	for _, args := range everySubcommandArgs() {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			cls1AssertRefusedBeforeAnythingIsTouched(t, args, serverURL)
		})
	}
}

// Test792b2c9a2_Targets_TheServerValueIsOnlyEchoed verifies 792b2c9a2: with ENGRAM_SERVER set and
// ENGRAM_PARENT not, no fetch is issued and the message names ENGRAM_PARENT with exactly the value
// ENGRAM_SERVER held (a mixed-case value with a path and a query, so any rewriting shows).
func Test792b2c9a2_Targets_TheServerValueIsOnlyEchoed(t *testing.T) {
	t.Parallel()

	const serverValue = "https://Old-Host.example:8093/base/path?x=1&y=two"

	for _, args := range everySubcommandArgs() {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			rec := &capabilityRecorder{}

			stderr, _ := runWithRemovedServer(rec, serverValue, args)

			g.Expect(rec.fetchCalls.Load()).To(BeZero(), "no fetch is issued")
			g.Expect(stderr).To(ContainSubstring("set ENGRAM_PARENT=" + serverValue + " instead"))
		})
	}
}

// cls1FetchRecorder is a parent that answers every request with answer and records each request URL.
type cls1FetchRecorder struct {
	answer string

	mu   sync.Mutex
	urls []string
}

// calls returns the recorded request URLs in order.
func (r *cls1FetchRecorder) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string{}, r.urls...)
}

// fetch records the request and answers 200 with the recorder's answer.
func (r *cls1FetchRecorder) fetch(_ context.Context, _, url string, _ []byte) (cli.FetchResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.urls = append(r.urls, url)

	return cli.FetchResponse{Status: 200, Body: []byte(r.answer)}, nil
}

// cls1AssertRefusedBeforeAnythingIsTouched runs args with ENGRAM_SERVER set to serverURL and asserts the
// refusal of 792b2c9a1 (shared with Test792a21a1a): a non-zero exit, a message naming ENGRAM_PARENT and
// the same URL, and no filesystem, lock, fetch, bind or command capability used.
func cls1AssertRefusedBeforeAnythingIsTouched(t *testing.T, args []string, serverURL string) {
	t.Helper()

	g := NewWithT(t)
	rec := &capabilityRecorder{}

	stderr, exitCodes := runWithRemovedServer(rec, serverURL, args)

	g.Expect(exitCodes).To(ContainElement(1), "exits non-zero")
	g.Expect(stderr).To(ContainSubstring("ENGRAM_SERVER is no longer supported"))
	g.Expect(stderr).To(ContainSubstring("ENGRAM_PARENT=" + serverURL))
	g.Expect(rec.fsCalls.Load()).To(BeZero(), "no filesystem capability")
	g.Expect(rec.lockCalls.Load()).To(BeZero(), "no lock capability")
	g.Expect(rec.fetchCalls.Load()).To(BeZero(), "no fetch capability")
	g.Expect(rec.bindCalls.Load()).To(BeZero(), "no bind capability")
	g.Expect(rec.execCalls.Load()).To(BeZero(), "no external command")
}

// cls1EnvOf returns a Getenv stub backed by values; a nil map reports every variable unset.
func cls1EnvOf(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
