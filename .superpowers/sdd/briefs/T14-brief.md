# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T14: <SET AT DISPATCH — after T4 ACK; verify `git log --oneline -1` matches>. Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- ACCUMULATED DISPATCH NOTES (binding):
  - **R11 amendment (ledgered at T6):** the stubEmbedder local-override pattern is ALREADY IN USE by the query cluster's targets tests — you are not introducing it. Two distinct stub needs exist: query-cluster tests need a SUCCEEDING Embed (RunQuery embeds per phrase); ModelID-only sites (T15's scope) use the fail-loud stubEmbedderForTargets. Don't conflate; don't touch T15's sites (osEmbedFS/embed.go's newOsEmbedDeps family stays until T15 — embed.go:156's osVaultFS reference is T15's to delete).
  - **Warning-routing class (ledgered at T12):** any family flipped to logWarningTo(d.Stderr) makes formerly-process-stderr warnings visible to test assertions — expect empty-stderr assertions to flip; pin the exact warning text when they do.
  - Plan cite drift: tallyStates is at embed.go:273 not :275; ALL cited line numbers are pristine-tree — locate by text, symbol gates govern.
  - **reorder-decls HAZARD:** `targ reorder-decls` is UNSCOPED — rewrites the 2 protected dev/eval please_step3_probe fixtures; if run, `git restore` those two paths explicitly afterward and verify `git status` shows only your files.
  - NEVER apply a full-file replacement to a file this brief doesn't own outright (primitives.go/targets.go/export_test.go/main.go get surgical edits only — other tasks' landed helpers live there).
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u`)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + the 2 dev/eval reorder fixtures; lint-full must be 0
  - `targ check-thin-api` gates your new cmd/engram/hugot.go — the E-1 closure shape below was empirically validated against the checker at plan time; if the checker still fails it, capture the exact finding and STOP (escalate; never suppress or restructure past the doctrine).
- House rules: `t.Parallel()` on every test/subtest (imptest/rapid/gomega stack; nilaway guards); named constants; descriptive names; <120 char lines.
- REPORT: write `.superpowers/sdd/briefs/T14-report.md` BEFORE your final message — status, commit SHA(s), gate outcomes verbatim (test / check-full / check-thin-api + standalone re-runs), every deviation with rationale, concerns/watch items. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

---

### Task T14 (A): internal/embed purification — Backend/CacheFS composed internally; thin hugot Runtime at the cmd edge (doctrine D-1)

**Doctrine note (BINDING — this rework applies the revised composition doctrine; it supersedes the pre-correction embed draft):** cmd/engram contributes ONLY thin declarations for the embedder path: one EMPTY struct `hugotRuntime` whose two methods are checker-verified thin shapes, plus one field line in `cmd/engram/main.go`'s `cli.Primitives` literal. ALL orchestration — session→pipeline lifecycle with destroy-on-failure, pipeline config policy (`engram-embed` / `model.onnx`), cache extraction, sentinel policy, permission policy, exist-error classification, and every `%w` wrap — lives in `internal/embed`, parameterized over injected capabilities. `Deps.Embed` stays wired INSIDE `cli.NewDeps` (R6/D-1); this task flips that line to the 3-arg constructor. Task-local design flags (within D-1's stated latitude — the doctrine's supersession map delegates "the exact field shapes" to this brief):

- **E-1 (runtime erasure shape — checker-derived):** `Runtime.NewPipeline` returns the pipeline's run function (`RunPipelineFunc`) as a closure over the concrete hugot pipeline, instead of an opaque handle that a separate `Run` method re-asserts. Reason, verified against targ's `checkFuncThinness`/`isSimpleErrorWrapper` source: the wrapper pattern requires statement 1's RHS to be a call whose receiver is a bare identifier — `pipeline.(*T).RunPipeline(...)` (type-asserted receiver) provably FAILS the gate, so an empty-struct `Run(pipeline any, ...)` mapping method is impossible. The closure form passes: `NewPipeline`'s body is the sanctioned 3-statement wrapper (`x, err := hugot.NewPipeline(...)`, `if err != nil`, `return`), whose third statement need only BE a return; the returned closure is doctrine-capped at a trivially-sequenced single-call body (call on the captured ident, err-check, selector return). No `any`, no re-assertion, no `pipelines` import in cmd.
- **E-2 (no new cache fields on Primitives):** D-1 names "backend/cache capability fields"; the cache side needs NO new fields — the T1-rework `cli.Primitives` FS fields (`Stat`, `MkdirAll`, `MkdirTemp`, `WriteFile`, `Rename`, `RemoveAll`) already carry every raw capability the cache composition needs, and `cli.NewDeps` forwards them into `embed.CacheFSPrims` verbatim. Only ONE new Primitives field lands: `EmbedRuntime embed.Runtime`.
- **E-3 (exist-classification moves internal, os-free):** the old `isExistErr`/`renameIsExist` sniffing cannot live in cmd (multi-statement) nor import `os` in internal. Internal `renameIsExist` uses `errors.Is(err, fs.ErrExist)` — which already covers EEXIST and, via `syscall.Errno.Is`'s ENOTEMPTY mapping, the macOS dir-over-dir case through `*os.LinkError`'s `Unwrap` — plus a `strings.Contains` fallback on the message ("file exists" / "directory not empty") preserving the previous defensive sniffing. The real-OS integration test (rename onto populated dir) keeps this honest on the actual platform.

**Files**

- Create: `internal/embed/runtime.go` (Runtime seam + `NewRuntimeBackend` composition), `internal/embed/cachefs.go` (`CacheFSPrims` + `NewCacheFS` composition), `internal/embed/runtime_test.go`, `internal/embed/cachefs_test.go`, `internal/embed/cachefs_integration_test.go` (real-os `_test` — sanctioned by the purity lint's `!$test` exclusion)
- Modify: `internal/embed/hugot.go` (full rewrite below)
- Modify: `internal/embed/cache.go` (full rewrite below)
- Modify: `internal/embed/export_test.go` (full rewrite below)
- Modify: `internal/embed/hugot_test.go`, `internal/embed/cache_test.go`, `internal/embed/buildembedder_test.go`, `internal/embed/overlength_test.go`, `internal/embed/embedder_fake_test.go`
- Delete: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`
- Create: `cmd/engram/hugot.go` (THIN: empty `hugotRuntime` struct + two thin methods, NOTHING else), `cmd/engram/hugot_test.go` (the sanctioned cmd wiring-smoke tests — `_test.go` is exempt from `check-thin-api`), `cmd/engram/testdata/model-stub.txt`
- Modify: `internal/cli/primitives.go` (Primitives gains `EmbedRuntime embed.Runtime`; `NewDeps`'s guarded Embed line → 3-arg composition — the R6 arity flip lands HERE, not in cmd), `internal/cli/embed.go` (sharedEmbedder → bridge; delete `modelCacheDir`), `internal/cli/targets.go` (wire bridge), `internal/cli/export_test.go` (bridge export), `cmd/engram/main.go` (Primitives literal gains ONE line: `EmbedRuntime: hugotRuntime{},`)
- Create: `internal/cli/embed_bridge_test.go`

**Interfaces**

- Produces (internal/embed): `embed.Backend` — `OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)`; `embed.PipelineHandle` — `RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)`, `Destroy() error`; `embed.FeatureOutput{ Embeddings [][]float32 }`; `embed.CacheFS` (exported rename of `cacheFS`, same 7 methods, Rename contract = `errors.Is(err, fs.ErrExist)`); `embed.RawSession` — `Destroy() error`; `embed.RunPipelineFunc func(ctx context.Context, inputs []string) ([][]float32, error)`; `embed.Runtime` — `NewSession(ctx context.Context) (RawSession, error)`, `NewPipeline(session RawSession, modelPath, name, onnxFilename string) (RunPipelineFunc, error)`; `embed.NewRuntimeBackend(runtime Runtime) Backend`; `embed.CacheFSPrims` (six func fields with signatures identical to the matching `cli.Primitives` fields); `embed.NewCacheFS(prims CacheFSPrims) CacheFS`; `embed.ErrRuntimeMissing`; `embed.BundledModelFS() stdembed.FS`; `embed.BundledModelDir = "assets/model"`; new constructor signatures `NewBundledHugotEmbedder(ctx, backend Backend, cfs CacheFS, cacheDir string)`, `NewHugotEmbedderFromDir(ctx, backend Backend, modelDir, modelID string)`, `NewHugotEmbedderFromFS(ctx, backend Backend, cfs CacheFS, modelFS stdembed.FS, modelDir, modelID, cacheDir string)`, `NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string)`.
- Produces (internal/cli): `Primitives.EmbedRuntime embed.Runtime` field; the 3-arg NewDeps Embed composition; `wireSharedEmbedder(embed.Embedder)` (unexported, called from `Targets`).
- Produces (cmd/engram): `hugotRuntime` — EMPTY struct implementing `embed.Runtime` with exactly two thin methods; NO other new declarations in package main.
- Consumes: T2's landed `NewDeps` guarded Embed wiring (`internal/cli/primitives.go` — exact before-text in step 9); `cli.CacheDirFromHome(home, modelID string, getenv func(string) string) string` (targets.go:56, unchanged); foundation's `cli.Deps.Embed embed.Embedder` field; `hugot.NewGoSession` / `hugot.NewPipeline` / `hugot.FeatureExtractionConfig` (cmd `_test` and `cmd/engram/hugot.go` only).

**Steps**

- [ ] 1. **RED — internal composition tests first.** Create `internal/embed/runtime_test.go` (unit tests of the internally-composed backend lifecycle over a fake Runtime — these are the relocated semantics of the old cmd backend-branch tests):

```go
package embed_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// fakeRuntime scripts the raw runtime seam so every branch of the
// internally-composed backend is unit-testable without hugot.
type fakeRuntime struct {
	sessionErr     error
	pipelineErr    error
	destroyErr     error
	destroyCalls   int
	pipelineCalled bool
	runFn          embed.RunPipelineFunc
	gotModelPath   string
	gotName        string
	gotOnnxFile    string
}

func (f *fakeRuntime) NewPipeline(
	_ embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	f.pipelineCalled = true
	f.gotModelPath = modelPath
	f.gotName = name
	f.gotOnnxFile = onnxFilename

	if f.pipelineErr != nil {
		return nil, f.pipelineErr
	}

	return f.runFn, nil
}

func (f *fakeRuntime) NewSession(context.Context) (embed.RawSession, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}

	return fakeRuntimeSession{runtime: f}, nil
}

type fakeRuntimeSession struct{ runtime *fakeRuntime }

func (s fakeRuntimeSession) Destroy() error {
	s.runtime.destroyCalls++

	return s.runtime.destroyErr
}

// TestRuntimeBackend_NilRuntimeFailsLoud asserts a Deps carrier built from
// Primitives without EmbedRuntime surfaces a clear error, never a panic.
func TestRuntimeBackend_NilRuntimeFailsLoud(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := embed.NewRuntimeBackend(nil).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(embed.ErrRuntimeMissing))
}

// TestRuntimeBackend_SessionFailPropagates exercises the first error branch
// of the composed OpenPipeline: NewSession returns an error.
func TestRuntimeBackend_SessionFailPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessionErr := errors.New("session blocked")
	runtime := &fakeRuntime{sessionErr: sessionErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot session")))
	g.Expect(err).To(MatchError(ContainSubstring("session blocked")))
	g.Expect(runtime.pipelineCalled).To(BeFalse(),
		"NewPipeline must not be called when NewSession fails")
}

// TestRuntimeBackend_PipelineFailDestroysSession exercises the second error
// branch: NewPipeline fails and the session's Destroy is called.
func TestRuntimeBackend_PipelineFailDestroysSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pipeErr := errors.New("pipeline blocked")
	runtime := &fakeRuntime{pipelineErr: pipeErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot pipeline")))
	g.Expect(err).To(MatchError(ContainSubstring("pipeline blocked")))
	g.Expect(runtime.destroyCalls).
		To(Equal(1), "session.Destroy must be called on pipeline failure")
}

// TestRuntimeBackend_ConfigPolicyIsInternal proves the pipeline config
// policy (name, onnx filename) lives internal: the raw runtime receives
// the values without cmd declaring them.
func TestRuntimeBackend_ConfigPolicyIsInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/models/m1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(runtime.gotModelPath).To(Equal("/models/m1"))
	g.Expect(runtime.gotName).To(Equal("engram-embed"))
	g.Expect(runtime.gotOnnxFile).To(Equal("model.onnx"))
}

// TestRuntimeBackend_RunMapsOutput drives the happy path through the
// returned handle: raw [][]float32 maps into embed.FeatureOutput.
func TestRuntimeBackend_RunMapsOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1, 2}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out, runErr := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(runErr).NotTo(HaveOccurred())
	g.Expect(out.Embeddings).To(Equal([][]float32{{1, 2}}))
}

// TestRuntimeBackend_RunErrorPropagates exercises the run error branch of
// the composed handle.
func TestRuntimeBackend_RunErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runErr := errors.New("run blocked")
	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return nil, runErr
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, err = handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).To(MatchError(ContainSubstring("hugot run")))
	g.Expect(err).To(MatchError(ContainSubstring("run blocked")))
}

// TestRuntimeBackend_DestroyErrorPropagates exercises the error branch of
// the composed handle's Destroy.
func TestRuntimeBackend_DestroyErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	destroyErr := errors.New("destroy blocked")
	runtime := &fakeRuntime{
		destroyErr: destroyErr,
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = handle.Destroy()
	g.Expect(err).To(MatchError(ContainSubstring("hugot session destroy")))
	g.Expect(err).To(MatchError(ContainSubstring("destroy blocked")))
}
```

Create `internal/embed/cachefs_test.go` (unit tests of the composed CacheFS over fake primitives — sentinel policy, permission policy, wraps, and the exist contract; the relocated semantics of the old cmd `osCacheFS` method tests):

```go
package embed_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// cachePrimRecorder scripts and records the raw-primitive calls the
// composed CacheFS makes, so sentinel/permission policy and error wraps
// are assertable without a real disk. Each (sub)test builds its own
// recorder — no shared mutable state across parallel tests.
type cachePrimRecorder struct {
	statPath     string
	statErr      error
	mkdirAllPath string
	mkdirAllPerm fs.FileMode
	mkdirAllErr  error
	mkdirTempErr error
	writePath    string
	writeData    []byte
	writePerm    fs.FileMode
	writeErr     error
	renameErr    error
	removeAllErr error
}

func (r *cachePrimRecorder) prims() embed.CacheFSPrims {
	return embed.CacheFSPrims{
		Stat: func(path string) (fs.FileInfo, error) {
			r.statPath = path

			return nil, r.statErr
		},
		MkdirAll: func(path string, perm fs.FileMode) error {
			r.mkdirAllPath = path
			r.mkdirAllPerm = perm

			return r.mkdirAllErr
		},
		MkdirTemp: func(_, _ string) (string, error) {
			return "/tmp/fake-extract", r.mkdirTempErr
		},
		WriteFile: func(path string, data []byte, perm fs.FileMode) error {
			r.writePath = path
			r.writeData = data
			r.writePerm = perm

			return r.writeErr
		},
		Rename: func(_, _ string) error {
			return r.renameErr
		},
		RemoveAll: func(_ string) error {
			return r.removeAllErr
		},
	}
}

// TestCacheFS_StatSentinel covers the sentinel-probe branches and proves
// the ".complete" sentinel name is internal policy (the raw Stat sees the
// joined path).
func TestCacheFS_StatSentinel(t *testing.T) {
	t.Parallel()

	t.Run("missing sentinel is false, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: fs.ErrNotExist}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeFalse())
		g.Expect(recorder.statPath).To(Equal("/cache/m1/.complete"))
	})

	t.Run("stat failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: errors.New("disk gone")}

		_, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).To(MatchError(ContainSubstring("stat sentinel")))
		g.Expect(err).To(MatchError(ContainSubstring("disk gone")))
	})

	t.Run("present sentinel is true, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeTrue())
	})
}

// TestCacheFS_MkdirAll asserts the internal dir-perm policy (0o755) and
// the error wrap.
func TestCacheFS_MkdirAll(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o755 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")).To(Succeed())
		g.Expect(recorder.mkdirAllPath).To(Equal("/cache"))
		g.Expect(recorder.mkdirAllPerm).To(Equal(fs.FileMode(0o755)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirAllErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir all")))
		g.Expect(err).To(MatchError(ContainSubstring("denied")))
	})
}

// TestCacheFS_MkdirTemp covers passthrough and wrap.
func TestCacheFS_MkdirTemp(t *testing.T) {
	t.Parallel()

	t.Run("returns the created dir", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		tmp, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tmp).To(Equal("/tmp/fake-extract"))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirTempErr: errors.New("full")}

		_, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir temp")))
	})
}

// TestCacheFS_WriteFile asserts the internal file-perm policy (0o600) and
// the error wrap.
func TestCacheFS_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o600 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))).
			To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/x/model.onnx"))
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))
		g.Expect(err).To(MatchError(ContainSubstring("write file")))
	})
}

// TestCacheFS_WriteSentinel proves the sentinel write is an empty
// ".complete" file under the internal perm policy.
func TestCacheFS_WriteSentinel(t *testing.T) {
	t.Parallel()

	t.Run("writes empty .complete", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")).To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/extract/.complete"))
		g.Expect(recorder.writeData).To(BeEmpty())
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")
		g.Expect(err).To(MatchError(ContainSubstring("write sentinel")))
	})
}

// TestCacheFS_RenameExistContract pins the load-bearing contract: every
// destination-exists flavor the raw primitive can produce must surface as
// errors.Is(err, fs.ErrExist); everything else wraps without the sentinel.
func TestCacheFS_RenameExistContract(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw       error
		wantExist bool
	}{
		"raw fs.ErrExist":                  {raw: fs.ErrExist, wantExist: true},
		"LinkError wrapping ErrExist":      {raw: &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrExist}, wantExist: true},
		"LinkError directory not empty":    {raw: &os.LinkError{Op: "rename", Old: "a", New: "b", Err: errors.New("directory not empty")}, wantExist: true},
		"bare directory-not-empty message": {raw: errors.New("rename a b: directory not empty"), wantExist: true},
		"unrelated error":                  {raw: os.ErrPermission, wantExist: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			recorder := &cachePrimRecorder{renameErr: testCase.raw}

			err := embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")
			g.Expect(err).To(HaveOccurred())
			g.Expect(errors.Is(err, fs.ErrExist)).To(Equal(testCase.wantExist))

			if !testCase.wantExist {
				g.Expect(err).To(MatchError(ContainSubstring("rename")))
			}
		})
	}

	t.Run("success is nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")).To(Succeed())
	})
}

// TestCacheFS_RemoveAllPassesThroughRaw pins the nil-on-missing contract:
// the raw primitive's error (or nil) passes through unwrapped.
func TestCacheFS_RemoveAllPassesThroughRaw(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(embed.NewCacheFS((&cachePrimRecorder{}).prims()).RemoveAll("/tmp/x")).To(Succeed())

	rawErr := errors.New("busy")
	recorder := &cachePrimRecorder{removeAllErr: rawErr}

	err := embed.NewCacheFS(recorder.prims()).RemoveAll("/tmp/x")
	g.Expect(err).To(MatchError(rawErr))
}

```

(The old cmd `TestRenameIsExist` branch matrix is absorbed by `TestCacheFS_RenameExistContract` above — the classifier is now an unexported internal helper exercised through the composed `Rename`.)

Create `internal/embed/cachefs_integration_test.go` (REAL os functions through the composition — sanctioned in internal `_test` files; this carries the extraction + platform-quirk coverage the old cmd adapter tests held):

```go
package embed_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// errBackendUnused aborts embedder construction after extraction so
// extraction-only behavior is assertable without a hugot runtime.
var errBackendUnused = errors.New("backend intentionally failing")

// failingBackend implements embed.Backend and always refuses to open.
type failingBackend struct{}

func (failingBackend) OpenPipeline(context.Context, string) (embed.PipelineHandle, error) {
	return nil, errBackendUnused
}

// realCacheFSForTest builds the production CacheFS composition over the
// raw os functions — the same wiring cli.NewDeps performs from
// cli.Primitives.
func realCacheFSForTest() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}

// TestCacheFS_RealOS_SentinelRoundTrip proves sentinel write + probe
// against a real tempdir.
func TestCacheFS_RealOS_SentinelRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := realCacheFSForTest()
	dir := t.TempDir()

	present, err := cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeFalse())

	g.Expect(cfs.WriteSentinel(dir)).To(Succeed())

	_, statErr := os.Stat(filepath.Join(dir, ".complete"))
	g.Expect(statErr).NotTo(HaveOccurred())

	present, err = cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeTrue())
}

// TestCacheFS_RealOS_RenameOntoPopulatedDir keeps the exist-classification
// honest on the actual OS: on macOS the raw rename error is ENOTEMPTY, and
// the composed Rename must still satisfy the fs.ErrExist contract.
func TestCacheFS_RealOS_RenameOntoPopulatedDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	dst := filepath.Join(parent, "dst")
	g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
	g.Expect(os.Mkdir(dst, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dst, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

	err := realCacheFSForTest().Rename(src, dst)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(),
		"CacheFS.Rename contract: destination-exists must satisfy errors.Is(err, fs.ErrExist)")
}

// TestExtractToCache_RealOS drives the internal extraction through the
// composed CacheFS on real disk: first call extracts and stamps the
// sentinel; second call reuses without re-extracting. The injected backend
// fails so no hugot runtime is needed (extraction happens before the
// backend opens). nonEmptyTestFS is declared in cache_test.go (same
// embed_test package; its move from unpack_test.go happens in step 8).
func TestExtractToCache_RealOS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "models", "stub@1")

	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	_, sentinelErr := os.Stat(filepath.Join(cacheDir, ".complete"))
	g.Expect(sentinelErr).NotTo(HaveOccurred(),
		".complete sentinel must be written after first extraction")

	entries1, readErr1 := os.ReadDir(cacheDir)
	g.Expect(readErr1).NotTo(HaveOccurred())

	fileCount1 := len(entries1)
	g.Expect(fileCount1).To(BeNumerically(">", 1), "cache dir must contain model files + sentinel")

	_, err = embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	entries2, readErr2 := os.ReadDir(cacheDir)
	g.Expect(readErr2).NotTo(HaveOccurred())
	g.Expect(entries2).To(HaveLen(fileCount1),
		"second call must not add/modify files — cache reused as-is")
}
```

(This replaces the old `internal/embed/production_cache_test.go` real-OS coverage and the deleted cmd-side extract test; the ADR-adjacent concurrent-race branches stay covered by cache_test.go's fake-driven race tests.)

- [ ] 2. **RED — cmd wiring-smoke tests.** Create `cmd/engram/testdata/model-stub.txt` containing `stub model payload` (one line). Create `cmd/engram/hugot_test.go` — the sanctioned cmd smoke suite (`_test.go` files are exempt from `check-thin-api`); it drives the REAL `hugotRuntime` through the internally-composed backend, which is the only direct coverage the thin cmd type gets (the production Primitives literal itself is guarded by cli_test's end-to-end binary tests, doctrine flag DRIFT):

```go
package main

import (
	stdembed "embed"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

//go:embed testdata
var testModelFS stdembed.FS

// realCacheFS mirrors the CacheFSPrims wiring cli.NewDeps builds from the
// production Primitives literal.
func realCacheFS() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}

// TestBundledEmbedder_Smoke exercises the full production wiring
// end-to-end: real hugot runtime (cmd's thin hugotRuntime), internally
// composed backend + cache FS, bundled model assets. Skipped under -short
// because it unpacks the ~90MB ONNX.
func TestBundledEmbedder_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bundled-embedder smoke test under -short")
	}

	t.Parallel()

	g := NewWithT(t)

	embedder, err := embed.NewBundledHugotEmbedder(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		filepath.Join(t.TempDir(), "model-cache"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() {
		_ = embedder.Close()
	}()

	const expectedDims = 384

	g.Expect(embedder.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(embedder.Dims()).To(Equal(expectedDims))

	vec, embErr := embedder.Embed(t.Context(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(expectedDims))

	const longLen = 4000

	longText := make([]byte, longLen)
	for i := range longText {
		longText[i] = 'a' + byte(i%26)
	}

	vec2, embErr2 := embedder.Embed(t.Context(), string(longText))
	g.Expect(embErr2).NotTo(HaveOccurred())
	g.Expect(vec2).To(HaveLen(expectedDims))
}

// TestHugotRejectsInvalidModelDir exercises the embedder-construction
// error branch through the real runtime: extraction succeeds (files
// exist) but hugot rejects the directory because it has no valid
// model.onnx.
func TestHugotRejectsInvalidModelDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		testModelFS, "testdata", "fake@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
}
```

Run: `targ test` — expected RED (compile failures: `embed.Runtime`, `embed.RunPipelineFunc`, `embed.RawSession`, `embed.NewRuntimeBackend`, `embed.CacheFSPrims`, `embed.NewCacheFS`, `embed.ErrRuntimeMissing`, `hugotRuntime`, and the new `embed.*` constructor arities don't exist yet).

- [ ] 3. **GREEN — create `internal/embed/runtime.go`** (the Runtime seam + the internally-composed production Backend; this is where the old cmd `hugotBackend`/`hugotPipeline` orchestration now lives):

```go
package embed

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	// ErrRuntimeMissing reports an embed attempt through a backend composed
	// from a nil Runtime (a Deps carrier whose Primitives had no
	// EmbedRuntime wired). Production main.go always wires one; minimal
	// test Primitives may not — this surfaces that as a clean lazy-init
	// error instead of a panic.
	ErrRuntimeMissing = errors.New(
		"embed runtime not wired: cli.Primitives.EmbedRuntime is required for embedding")
)

// RawSession is the minimal runtime-session surface the composed backend
// needs: cleanup on pipeline-creation failure and on normal close.
// *hugot.Session satisfies it structurally.
type RawSession interface {
	Destroy() error
}

// RunPipelineFunc runs an opened embedding pipeline on inputs and returns
// one vector per input. cmd's Runtime.NewPipeline returns one as a closure
// over the concrete pipeline, erasing the runtime's types without any
// re-assertion at call time (doctrine flag E-1).
type RunPipelineFunc func(ctx context.Context, inputs []string) ([][]float32, error)

// Runtime is the raw model-runtime capability surface. The production
// implementation is cmd/engram's EMPTY hugotRuntime struct whose two
// methods are single-call bodies (targ check-thin-api); ALL lifecycle
// orchestration and config policy live here, behind NewRuntimeBackend
// (#700, doctrine flag D-1).
type Runtime interface {
	// NewSession opens a runtime session.
	NewSession(ctx context.Context) (RawSession, error)
	// NewPipeline opens a feature-extraction pipeline for the model at
	// modelPath on session and returns its run function.
	NewPipeline(session RawSession, modelPath, name, onnxFilename string) (RunPipelineFunc, error)
}

// NewRuntimeBackend composes the production Backend from a raw Runtime:
// the open-session → open-pipeline → destroy-on-failure lifecycle, the
// pipeline config policy, and all error wrapping happen here, internally.
func NewRuntimeBackend(runtime Runtime) Backend {
	return runtimeBackend{runtime: runtime}
}

// unexported constants.
const (
	// pipelineName and pipelineOnnxFilename are the feature-extraction
	// pipeline config policy — kept internal so cmd passes values through
	// without declaring any constants (thin-api).
	pipelineName         = "engram-embed"
	pipelineOnnxFilename = "model.onnx"
)

// runtimeBackend implements Backend over a raw Runtime.
type runtimeBackend struct {
	runtime Runtime
}

// OpenPipeline opens a session, then a feature-extraction pipeline on it,
// destroying the session if pipeline creation fails.
func (b runtimeBackend) OpenPipeline(
	ctx context.Context, modelDir string,
) (PipelineHandle, error) {
	if b.runtime == nil {
		return nil, ErrRuntimeMissing
	}

	session, err := b.runtime.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	run, pipeErr := b.runtime.NewPipeline(session, modelDir, pipelineName, pipelineOnnxFilename)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	return &runtimePipeline{session: session, run: run}, nil
}

// runtimePipeline pairs a pipeline run function with the session that owns
// it so Destroy releases both together.
type runtimePipeline struct {
	session RawSession
	run     RunPipelineFunc
}

// Destroy releases the owning session (which owns the pipeline).
func (p *runtimePipeline) Destroy() error {
	err := p.session.Destroy()
	if err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}

	return nil
}

// RunPipeline runs the model and maps the raw vectors into the
// runtime-neutral FeatureOutput shape.
func (p *runtimePipeline) RunPipeline(
	ctx context.Context, inputs []string,
) (FeatureOutput, error) {
	out, err := p.run(ctx, inputs)
	if err != nil {
		return FeatureOutput{}, fmt.Errorf("hugot run: %w", err)
	}

	return FeatureOutput{Embeddings: out}, nil
}
```

- [ ] 4. **Rewrite `internal/embed/hugot.go`** — hugot and os imports leave internal; `Backend`/`PipelineHandle`/`FeatureOutput` exported; constructors take injected seams; `buildEmbedder` folds into `NewHugotEmbedderFromDir`; dead `tempFS`/`productionTempFS`/`unpackModelToTemp` deleted. Full replacement:

```go
package embed

import (
	"context"
	stdembed "embed"
	"errors"
	"fmt"
	"sync"
)

// Exported constants.
const (
	BundledModelID = "minilm-l6-v2@384"
	// BundledModelDir is the directory inside BundledModelFS holding the
	// bundled model files.
	BundledModelDir = "assets/model"
)

// Exported variables.
var (
	ErrBundledModelUnavailable = errors.New(
		"bundled model missing or empty — rebuild the binary with the model in place, " +
			"or set ENGRAM_MODEL_PATH to a directory containing model.onnx",
	)
	ErrHugotEmbedEmpty = errors.New("hugot embed: empty result")
	ErrHugotProbeEmpty = errors.New("hugot probe returned no embedding")
)

// Backend opens an embedding pipeline for an on-disk model directory. The
// production implementation is composed internally by NewRuntimeBackend
// (runtime.go) over the raw Runtime that cmd wires into cli.Primitives —
// no hugot import anywhere in internal (#700); tests inject fakes to
// exercise every constructor branch.
type Backend interface {
	OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)
}

// FeatureOutput is the embedding shape returned by
// PipelineHandle.RunPipeline, mirrored here so implementations don't leak
// their runtime's own output types.
type FeatureOutput struct {
	Embeddings [][]float32
}

// HugotEmbedder wraps an embedding pipeline. Safe for concurrent use — the
// production pipeline runs the model under its own lock.
type HugotEmbedder struct {
	pipeline interface {
		RunPipeline(ctx context.Context, inputs []string) (out FeatureOutput, err error)
	}
	modelID string
	dims    int

	// Capture the close logic at construction time so the destroy chain
	// stays encapsulated even if the backend's session type changes.
	close func() error
}

// PipelineHandle is the runtime surface of an opened pipeline plus its
// owning session; Destroy releases both together.
type PipelineHandle interface {
	RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)
	Destroy() error
}

// BundledModelFS returns the go:embed-ed bundled model assets, rooted at
// BundledModelDir. Exposed so cmd/engram (and its integration tests) can
// hand the bundled assets to the injectable constructors.
func BundledModelFS() stdembed.FS { return bundledModel }

// NewBundledHugotEmbedder is the production constructor: bundled assets FS,
// fixed model directory, fixed model ID, and caller-supplied backend, cache
// FS, and cache dir. The cache dir is the XDG-keyed path where the model is
// extracted once and reused across all subsequent invocations.
func NewBundledHugotEmbedder(
	ctx context.Context, backend Backend, cfs CacheFS, cacheDir string,
) (*HugotEmbedder, error) {
	return NewHugotEmbedderFromFS(
		ctx, backend, cfs, bundledModel, BundledModelDir, BundledModelID, cacheDir)
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from a
// directory on disk via the injected backend, probing once to learn the
// embedding dimensionality. Every error branch (pipeline open, probe run,
// empty probe) is unit-testable with a fake Backend.
func NewHugotEmbedderFromDir(
	ctx context.Context, backend Backend, modelDir, modelID string,
) (*HugotEmbedder, error) {
	handle, openErr := backend.OpenPipeline(ctx, modelDir)
	if openErr != nil {
		return nil, openErr
	}

	probe, probeErr := handle.RunPipeline(ctx, []string{"probe"})
	if probeErr != nil {
		_ = handle.Destroy()

		return nil, probeErr
	}

	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = handle.Destroy()

		return nil, ErrHugotProbeEmpty
	}

	runner := &pipelineRunner{run: handle.RunPipeline}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
		close:    handle.Destroy,
	}, nil
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS rooted
// at modelDir. cacheDir is the stable directory where the model is
// extracted once via cfs and reused across invocations (XDG-keyed). Tests
// pass an empty FS to verify UAT 10's clear-error path.
func NewHugotEmbedderFromFS(
	ctx context.Context, backend Backend, cfs CacheFS,
	modelFS stdembed.FS, modelDir, modelID, cacheDir string,
) (*HugotEmbedder, error) {
	dir, extractErr := extractToCache(cfs, modelFS, modelDir, cacheDir)
	if extractErr != nil {
		return nil, extractErr
	}

	return NewHugotEmbedderFromDir(ctx, backend, dir, modelID)
}

// Close releases the underlying session. Safe to call multiple times. The
// model cache dir is NOT removed — it is a shared, persistent cache reused
// across all engram invocations.
func (h *HugotEmbedder) Close() error {
	if h.close != nil {
		err := h.close()
		h.close = nil

		return err
	}

	return nil
}

// Dims reports the embedding dimensionality.
func (h *HugotEmbedder) Dims() int { return h.dims }

// Embed runs the pipeline on text (truncated to fit the model's context
// window) and returns the resulting vector.
//
// The char guard assumes prose density; code-dense text can still exceed the
// model's 512-token positional limit within the char limit (observed: 1500
// chars of transcript tokenizing to 538 tokens, panicking graph compilation).
// On failure the input is halved and retried until it succeeds or bottoms out,
// so a single dense chunk degrades to a shorter prefix instead of failing the
// whole ingest.
func (h *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if len(text) > hugotInputCharLimit {
		text = text[:hugotInputCharLimit]
	}

	for {
		out, err := h.pipeline.RunPipeline(ctx, []string{text})
		if err != nil {
			if len(text) >= hugotRetryFloorChars {
				text = text[:len(text)/2]

				continue
			}

			return nil, err
		}

		if len(out.Embeddings) == 0 {
			return nil, ErrHugotEmbedEmpty
		}

		return out.Embeddings[0], nil
	}
}

// ModelID reports the configured model identifier.
func (h *HugotEmbedder) ModelID() string { return h.modelID }

// LazyEmbedder defers construction of an embedder until first use so
// commands that don't need it (help, update, transcript) don't pay the
// model-unpack cost or die if model loading fails. The construction is
// factory-injected so tests can drive both the success and failure
// init paths without a real backend.
type LazyEmbedder struct {
	once    sync.Once
	factory func() (*HugotEmbedder, error)
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder that
// extracts the bundled model to cacheDir at most once (on first Embed /
// ModelID / Dims call) using the injected backend and cache FS. cacheDir
// should be the XDG-keyed stable cache path for the model, e.g.
// $XDG_CACHE_HOME/engram/models/<model_id>/.
func NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string) *LazyEmbedder {
	return &LazyEmbedder{
		// Background context: the lazy init runs at most once per process;
		// a request-scoped context could cancel construction partway through
		// model extraction and leave a partial temp dir.
		factory: func() (*HugotEmbedder, error) {
			return NewBundledHugotEmbedder(context.Background(), backend, cfs, cacheDir)
		},
	}
}

// Dims lazily constructs the embedder, then delegates. Returns 0 when
// construction failed; callers should detect via an Embed error.
func (l *LazyEmbedder) Dims() int {
	l.init()

	if l.initErr != nil {
		return 0
	}

	return l.emb.Dims()
}

// Embed lazily constructs the embedder, then delegates.
func (l *LazyEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	l.init()

	if l.initErr != nil {
		return nil, fmt.Errorf("embedder unavailable: %w", l.initErr)
	}

	return l.emb.Embed(ctx, text)
}

// ModelID lazily constructs the embedder, then delegates. Returns the
// bundled model id when construction has not been attempted yet so
// status-style callers can avoid paying the unpack cost.
func (l *LazyEmbedder) ModelID() string {
	if l.emb == nil && l.initErr == nil {
		return BundledModelID
	}

	if l.initErr != nil {
		return BundledModelID
	}

	return l.emb.ModelID()
}

// init runs at most once per LazyEmbedder via sync.Once. The factory
// is provided at construction time so tests can drive both success and
// failure init paths without a real backend.
func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		l.emb, l.initErr = l.factory()
	})
}

// unexported constants.
const (
	hugotInputCharLimit = 1500
	// hugotRetryFloorChars stops the over-length halving retry: below this
	// the failure is not a token-budget problem and must surface.
	hugotRetryFloorChars = 100
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// pipelineRunner adapts a PipelineHandle's run function to the small
// interface HugotEmbedder depends on; isolating the dependency makes
// backend version bumps a surgical edit instead of a sweep.
type pipelineRunner struct {
	run func(ctx context.Context, inputs []string) (FeatureOutput, error)
}

func (p *pipelineRunner) RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error) {
	return p.run(ctx, inputs)
}
```

- [ ] 5. **Rewrite `internal/embed/cache.go`** — `CacheFS` exported, os import gone, exist-classification becomes the `fs.ErrExist` contract (the classification itself lives in cachefs.go, step 6). Full replacement:

```go
package embed

import (
	stdembed "embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// CacheFS is the I/O surface extractToCache depends on. The production
// implementation is composed internally by NewCacheFS (cachefs.go) over
// raw filesystem primitives; tests inject fakes to exercise every branch
// without touching the real disk.
type CacheFS interface {
	// StatSentinel reports whether the cache dir already has a .complete sentinel.
	StatSentinel(cacheDir string) (bool, error)
	// MkdirAll ensures the parent directory of the cache dir exists.
	MkdirAll(path string) error
	// MkdirTemp creates a temporary directory sibling of cacheDir for atomic extraction.
	MkdirTemp(parent, pattern string) (string, error)
	// WriteFile writes data to path (used to copy model files into the temp dir).
	WriteFile(path string, data []byte) error
	// WriteSentinel writes the .complete sentinel into tmpDir.
	WriteSentinel(tmpDir string) error
	// Rename renames src to dst atomically. When dst already exists
	// (concurrent-race scenario), the returned error MUST satisfy
	// errors.Is(err, fs.ErrExist) — implementations translate platform
	// quirks (e.g. macOS ENOTEMPTY on dir-over-dir renames) before returning.
	Rename(src, dst string) error
	// RemoveAll deletes path recursively (used to clean up temp on rename race).
	RemoveAll(path string) error
}

// commitCache atomically renames tmp into cacheDir. If the rename fails with
// a destination-exists error (concurrent-process race), it re-checks the
// sentinel: if the winner completed the cache, discards tmp and returns
// cacheDir. Otherwise returns the rename error.
func commitCache(cfs CacheFS, tmp, cacheDir string) (string, error) {
	renameErr := cfs.Rename(tmp, cacheDir)
	if renameErr == nil {
		return cacheDir, nil
	}

	// If the rename failed because the destination exists, check whether
	// another process just won the race and completed the cache. If so,
	// discard our temp.
	if errors.Is(renameErr, fs.ErrExist) {
		complete, statErr := cfs.StatSentinel(cacheDir)
		if statErr == nil && complete {
			_ = cfs.RemoveAll(tmp)

			return cacheDir, nil
		}
	}

	// True rename failure (or sentinel absent after race check).
	_ = cfs.RemoveAll(tmp)

	return "", fmt.Errorf("cache rename: %w", renameErr)
}

// copyModelFiles copies every non-directory entry from modelFS/modelDir into tmpDir.
func copyModelFiles(cfs CacheFS, modelFS stdembed.FS, modelDir, tmpDir string) error {
	entries, _ := modelFS.ReadDir(modelDir) // already validated by caller

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		writeErr := cfs.WriteFile(filepath.Join(tmpDir, entry.Name()), data)
		if writeErr != nil {
			return fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return nil
}

// extractToCache ensures that <cacheDir> contains the fully extracted model
// and the .complete sentinel. On first call it extracts into a sibling temp
// dir and atomically renames it into place. On subsequent calls (sentinel
// present) it returns immediately without any I/O. A concurrent-process race
// (rename fails because another process just won) is handled by discarding
// the temp dir and using the pre-existing complete cache.
func extractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Fast path: already extracted.
	ok, statErr := cfs.StatSentinel(cacheDir)
	if statErr != nil {
		return "", statErr
	}

	if ok {
		return cacheDir, nil
	}

	return populateCache(cfs, modelFS, modelDir, cacheDir)
}

// populateCache handles the slow path of extractToCache: verifying the model
// FS, creating a sibling temp dir, copying model files, and atomically renaming
// the temp dir into place.
func populateCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Verify the model FS has files before creating any directories.
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	// Ensure the parent directory exists.
	parent := filepath.Dir(cacheDir)

	mkdirErr := cfs.MkdirAll(parent)
	if mkdirErr != nil {
		return "", fmt.Errorf("cache parent dir: %w", mkdirErr)
	}

	// Extract into a sibling temp dir so the rename is atomic.
	tmp, tmpErr := cfs.MkdirTemp(parent, ".tmp-engram-model-*")
	if tmpErr != nil {
		return "", fmt.Errorf("cache temp dir: %w", tmpErr)
	}

	copyErr := copyModelFiles(cfs, modelFS, modelDir, tmp)
	if copyErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", copyErr
	}

	sentinelErr := cfs.WriteSentinel(tmp)
	if sentinelErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", fmt.Errorf("cache sentinel: %w", sentinelErr)
	}

	return commitCache(cfs, tmp, cacheDir)
}
```

- [ ] 6. **Create `internal/embed/cachefs.go`** — the composed production CacheFS over raw primitives (the old cmd `osCacheFS` logic, now internal; sentinel + perm policy + exist-classification live here):

```go
package embed

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// CacheFSPrims carries the raw filesystem capabilities the production
// CacheFS composition needs. Field signatures are identical to the
// matching cli.Primitives fields — cli.NewDeps forwards its Primitives
// fields into this struct verbatim (doctrine flag E-2: no new cache
// fields on cli.Primitives). The funcs return RAW os errors; all wrapping
// and exist-classification happen here, internally.
type CacheFSPrims struct {
	Stat      func(path string) (fs.FileInfo, error)
	MkdirAll  func(path string, perm fs.FileMode) error
	MkdirTemp func(dir, pattern string) (string, error)
	WriteFile func(path string, data []byte, perm fs.FileMode) error
	Rename    func(oldPath, newPath string) error
	RemoveAll func(path string) error
}

// NewCacheFS composes the production CacheFS from raw filesystem
// primitives: sentinel policy, permission policy, error wrapping, and the
// fs.ErrExist rename contract all live here (#700).
func NewCacheFS(prims CacheFSPrims) CacheFS {
	return primCacheFS{prims: prims}
}

// unexported constants.
const (
	// sentinelFileName marks a fully extracted model cache dir.
	sentinelFileName = ".complete"

	cacheDirPerm  fs.FileMode = 0o755
	cacheFilePerm fs.FileMode = 0o600
)

// primCacheFS is the CacheFS composition over raw primitives.
type primCacheFS struct {
	prims CacheFSPrims
}

// MkdirAll ensures the parent directory of the cache dir exists.
func (c primCacheFS) MkdirAll(path string) error {
	err := c.prims.MkdirAll(path, cacheDirPerm)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	return nil
}

// MkdirTemp creates a temporary sibling dir for atomic extraction.
func (c primCacheFS) MkdirTemp(parent, pattern string) (string, error) {
	tmp, err := c.prims.MkdirTemp(parent, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

// RemoveAll deletes path. The raw primitive's contract (os.RemoveAll: nil
// on missing paths) is already caller-friendly; the error passes through
// unwrapped.
func (c primCacheFS) RemoveAll(path string) error {
	return c.prims.RemoveAll(path) //nolint:wrapcheck // see comment above
}

// Rename renames src to dst atomically. When the destination already
// exists (including macOS ENOTEMPTY for dir-over-dir renames), the
// returned error satisfies errors.Is(err, fs.ErrExist) per the CacheFS
// contract.
func (c primCacheFS) Rename(src, dst string) error {
	err := c.prims.Rename(src, dst)
	if err != nil {
		if renameIsExist(err) {
			return fmt.Errorf("%w: %w", fs.ErrExist, err)
		}

		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// StatSentinel reports whether cacheDir already has a .complete sentinel.
func (c primCacheFS) StatSentinel(cacheDir string) (bool, error) {
	_, err := c.prims.Stat(filepath.Join(cacheDir, sentinelFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat sentinel: %w", err)
	}

	return true, nil
}

// WriteFile writes data to path (copies model files into the temp dir).
func (c primCacheFS) WriteFile(path string, data []byte) error {
	err := c.prims.WriteFile(path, data, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// WriteSentinel writes the .complete sentinel into tmpDir.
func (c primCacheFS) WriteSentinel(tmpDir string) error {
	err := c.prims.WriteFile(filepath.Join(tmpDir, sentinelFileName), []byte{}, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}

	return nil
}

// renameIsExist reports whether err (raw from the rename primitive) is a
// destination-exists error. errors.Is(err, fs.ErrExist) covers EEXIST
// and — via syscall.Errno's Is mapping — ENOTEMPTY through
// *os.LinkError's Unwrap; the string fallback preserves the previous
// defensive platform sniffing for chains that don't unwrap to a mapped
// errno, without importing os (doctrine flag E-3).
func renameIsExist(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrExist) {
		return true
	}

	message := err.Error()

	return strings.Contains(message, "file exists") ||
		strings.Contains(message, "directory not empty")
}
```

- [ ] 7. **Create `cmd/engram/hugot.go`** — the THIN production runtime. This is the entire file; `targ check-thin-api` walks it, and every declaration is a verified-thin shape (empty struct; single-return external call; 3-statement simple-error-wrapper whose closing return carries the doctrine-capped closure — flag E-1):

```go
// Thin hugot capability wrappers. This file (plus its _test siblings) is
// the only place in the repo outside internal/embed's _test files that
// imports hugot — and it holds NO logic: hugotRuntime is an EMPTY struct
// whose methods are single-call / simple-error-wrapper bodies. The
// session/pipeline lifecycle, config policy, output mapping, and error
// wrapping all live in internal/embed (#700).
package main

import (
	"context"

	"github.com/knights-analytics/hugot"

	"github.com/toejough/engram/internal/embed"
)

// hugotRuntime implements embed.Runtime over the real hugot library.
type hugotRuntime struct{}

// NewPipeline opens a feature-extraction pipeline on session and returns
// its run function, erasing hugot's pipeline type via closure capture
// (doctrine flag E-1): the closure body is the sanctioned
// trivially-sequenced single-call shape — run on the captured pipe,
// err-check, selector return.
func (hugotRuntime) NewPipeline(
	session embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	//nolint:forcetypeassert // production invariant: sessions come from NewSession
	pipe, err := hugot.NewPipeline(session.(*hugot.Session), hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         name,
		OnnxFilename: onnxFilename,
	})
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, inputs []string) ([][]float32, error) {
		out, runErr := pipe.RunPipeline(ctx, inputs)
		if runErr != nil {
			return nil, runErr
		}

		return out.Embeddings, nil
	}, nil
}

// NewSession opens a Go-backend hugot session. *hugot.Session satisfies
// embed.RawSession structurally.
func (hugotRuntime) NewSession(ctx context.Context) (embed.RawSession, error) {
	return hugot.NewGoSession(ctx)
}
```

Checker verification (derived from targ's `checkFuncThinness` source — escalate, do not restructure, if the gate disagrees): `type hugotRuntime struct{}` = empty struct → thin; `NewSession` = single return of an external call (`hugot.NewGoSession`) → thin; `NewPipeline` = `isSimpleErrorWrapper` (stmt 1: `pipe, err := hugot.NewPipeline(...)` — RHS is a `pkg.Func` call, arguments are not walked, so the inline type assertion and composite literal are legal; stmt 2: `if err != nil`; stmt 3: any return statement — contents unchecked). Raw errors, zero constants, zero `pipelines` import: config values and wraps are internal.

- [ ] 8. **Adapt internal/embed tests.** Delete files: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`. Rewrite `internal/embed/export_test.go` (full replacement):

```go
package embed

import (
	"context"
	stdembed "embed"
)

// Exported variables.
var (
	ExportNotExist = notExist
)

// ExportExtractToCache exposes the unexported extractToCache helper so
// tests can exercise the sentinel / race / error branches with a fake
// CacheFS without touching the real disk.
func ExportExtractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	return extractToCache(cfs, modelFS, modelDir, cacheDir)
}

// NewHugotEmbedderWithPipelineForTest constructs a HugotEmbedder around
// a caller-supplied pipeline implementation. Tests use this to exercise
// the Embed/Close error branches without depending on a real backend.
func NewHugotEmbedderWithPipelineForTest(
	modelID string, dims int,
	runFn func(text string) ([][]float32, error),
	closeFn func() error,
) *HugotEmbedder {
	runner := &pipelineRunner{
		run: func(_ context.Context, inputs []string) (FeatureOutput, error) {
			out, err := runFn(inputs[0])
			if err != nil {
				return FeatureOutput{}, err
			}

			return FeatureOutput{Embeddings: out}, nil
		},
	}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     dims,
		close:    closeFn,
	}
}

// NewLazyEmbedderWithFactoryForTest constructs a LazyEmbedder with a
// caller-supplied factory so tests can drive both init success and
// failure paths without a real backend.
func NewLazyEmbedderWithFactoryForTest(factory func() (*HugotEmbedder, error)) *LazyEmbedder {
	return &LazyEmbedder{factory: factory}
}

// SetCacheDirForTest is a no-op test helper for the Close-does-not-delete
// test. HugotEmbedder no longer holds a tmpDir field — Close only closes the
// backend session and never removes any directory. The function is kept for
// test readability; the test creates its own dir and verifies it survives.
func SetCacheDirForTest(_ *HugotEmbedder, _ string) {}
```

Then mechanical edits, all in `internal/embed/*_test.go`:
  - Everywhere: `embed.ExportHugotBackend` → `embed.Backend`; `embed.ExportHugotPipelineHandle` → `embed.PipelineHandle`; `embed.ExportFeatureOutput` → `embed.FeatureOutput`; `embed.ExportCacheFS` → `embed.CacheFS`; `embed.BuildEmbedderForTest(` → `embed.NewHugotEmbedderFromDir(` (identical argument lists — verified).
  - `cache_test.go`: (a) delete `TestExtractToCache_RealOS` (lines 118-149); (b) replace both race-fake errors at lines 54 and 107 — current: `renameErr: &os.LinkError{Op: "rename", Err: errors.New("directory not empty")}` — new: `renameErr: fmt.Errorf("%w: directory not empty", fs.ErrExist)` (add `"fmt"`, `"io/fs"` imports); (c) type-assertion line 212: `_ embed.ExportCacheFS = (*fakeCacheFS)(nil)` → `_ embed.CacheFS = (*fakeCacheFS)(nil)`; (d) move the `nonEmptyTestFS` declaration here from the deleted `unpack_test.go` (add `stdembed "embed"` import):

```go
//go:embed testdata/gen-reference.py
var nonEmptyTestFS stdembed.FS
```

  - `hugot_test.go`: delete `TestBundledHugotEmbedder_Smoke` (relocated to `cmd/engram/hugot_test.go`, step 2 — where the real `hugotRuntime` lives); adapt T10 (fakes never reached — extraction fails first on the empty FS):

```go
func TestT10_MissingBundledModel_ClearError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		context.Background(), &fakeBackend{}, &fakeCacheFS{}, emptyFS, "assets/model", "x@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ENGRAM_MODEL_PATH"))
}
```

  (`fakeBackend` lives in `buildembedder_test.go`, `fakeCacheFS` in `cache_test.go` — same `embed_test` package; zero-value `fakeCacheFS.StatSentinel` returns `(false, nil)` — verified.)
  - `embedder_fake_test.go`: `embed.NewLazyEmbedder(t.TempDir())` → `embed.NewLazyEmbedder(&fakeBackend{}, &fakeCacheFS{}, t.TempDir())`.

- [ ] 9. **Wire internal/cli: Primitives field, NewDeps 3-arg flip, bridge, cmd literal line.** Four sub-edits, one commit with the rest of the task. **9a.** In `internal/cli/embed.go` delete the `modelCacheDir()` helper (its `os.UserHomeDir`/`os.Getenv` reads die with it — the `"os"` import leaves this file) and replace the `sharedEmbedder` singleton block:

Current:
```go
// unexported variables.
var (
	//nolint:gochecknoglobals // shared lazy singleton across CLI commands
	sharedEmbedder = embed.NewLazyEmbedder(modelCacheDir())
)
```

Replacement (add `"sync/atomic"` import):
```go
// unexported variables.
var (
	// errEmbedderUnwired reports an Embed call before Targets wired Deps.Embed.
	errEmbedderUnwired = errors.New("embedder not wired: cli.Targets(deps) has not run")

	// sharedEmbedderPtr holds the Deps-wired production embedder, stored by
	// wireSharedEmbedder (called from Targets). Atomic because tests build
	// Targets concurrently.
	// TRANSITIONAL (#700): deleted once every per-command deps constructor
	// takes Deps and reads d.Embed directly.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedderPtr atomic.Pointer[embed.Embedder]

	// sharedEmbedder is the value legacy per-command constructors wire into
	// their deps structs; it forwards to the Targets-wired embedder.
	// TRANSITIONAL (#700) — same removal condition as sharedEmbedderPtr.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedder embed.Embedder = bridgeEmbedder{ptr: &sharedEmbedderPtr}
)

// bridgeEmbedder forwards Embedder calls to the embedder wired by Targets.
// Pre-wiring fallbacks mirror LazyEmbedder's pre-init behavior: ModelID
// reports the bundled constant, Dims reports 0, Embed errors.
type bridgeEmbedder struct {
	ptr *atomic.Pointer[embed.Embedder]
}

// Dims delegates to the wired embedder; 0 before wiring.
func (b bridgeEmbedder) Dims() int {
	emb := b.load()
	if emb == nil {
		return 0
	}

	return emb.Dims()
}

// Embed delegates to the wired embedder; errors before wiring.
func (b bridgeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	emb := b.load()
	if emb == nil {
		return nil, errEmbedderUnwired
	}

	return emb.Embed(ctx, text)
}

// ModelID delegates to the wired embedder; bundled constant before wiring
// (keeps status-style callers unpack-free, matching LazyEmbedder).
func (b bridgeEmbedder) ModelID() string {
	emb := b.load()
	if emb == nil {
		return embed.BundledModelID
	}

	return emb.ModelID()
}

// load returns the wired embedder or nil before wiring.
func (b bridgeEmbedder) load() embed.Embedder {
	ptr := b.ptr.Load()
	if ptr == nil || *ptr == nil {
		return nil
	}

	return *ptr
}

// wireSharedEmbedder points the transitional sharedEmbedder bridge at the
// Deps-wired embedder. Called by Targets.
func wireSharedEmbedder(embedder embed.Embedder) {
	sharedEmbedderPtr.Store(&embedder)
}
```

**9b.** In `internal/cli/targets.go`, insert as the first statement of `Targets` (T2's landed shape `func Targets(deps Deps) []any`): `wireSharedEmbedder(deps.Embed)`.

**9c.** In `internal/cli/primitives.go`: add ONE field to `Primitives` (grouped after the debug-sink field, with the doctrine cross-reference) —

```go
	// Embedding runtime (cmd wires an EMPTY struct with single-call
	// methods; all lifecycle/config/cache policy is internal — doctrine
	// flags D-1/E-1/E-2).
	EmbedRuntime embed.Runtime
```

— and flip `NewDeps`'s guarded Embed wiring (T2's landed R6 handoff line) to the 3-arg composition. Current (landed by T2):

```go
	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6: T14
	// swaps this line to the 3-arg constructor over cmd-injected backend
	// and cache capabilities.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
	}
```

New:

```go
	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6/D-1:
	// backend composed from the raw EmbedRuntime, cache FS from the raw
	// filesystem primitives — no embed wiring in cmd. A nil EmbedRuntime
	// surfaces as embed.ErrRuntimeMissing on first use (fail-loud lazy),
	// never a panic.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			embed.NewRuntimeBackend(prims.EmbedRuntime),
			embed.NewCacheFS(embed.CacheFSPrims{
				Stat:      prims.Stat,
				MkdirAll:  prims.MkdirAll,
				MkdirTemp: prims.MkdirTemp,
				WriteFile: prims.WriteFile,
				Rename:    prims.Rename,
				RemoveAll: prims.RemoveAll,
			}),
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
	}
```

**9d.** In `cmd/engram/main.go`'s `cli.Primitives` literal, add ONE field line (after `WalkDir:`, keeping the literal's direct-reference grouping):

```go
			EmbedRuntime: hugotRuntime{},
```

Package main stays declaration-free — `hugotRuntime{}` is a composite-literal ARGUMENT expression (unchecked by the gate); the type itself is step 7's empty struct. NOTE(cli_test DRIFT flag): `realPrimitives()` in internal/cli tests does NOT gain an EmbedRuntime (cli_test cannot reference package main's `hugotRuntime`); Deps built from it get the fail-loud `ErrRuntimeMissing` lazy embedder, which no cli-level test may trigger (R11's `stubEmbedderForTargets` covers targets-level embed tests; the production literal is guarded by cli_test's end-to-end binary tests).

- [ ] 10. **Bridge behavior tests (parallel-safe, no global state).** Add to `internal/cli/export_test.go`:

```go
// ExportNewBridgeEmbedder returns a fresh transitional shared-embedder
// bridge plus its wire function, backed by an isolated pointer so bridge
// behavior tests never touch the package-global embedder.
func ExportNewBridgeEmbedder() (embed.Embedder, func(embed.Embedder)) {
	ptr := &atomic.Pointer[embed.Embedder]{}
	bridge := bridgeEmbedder{ptr: ptr}
	wire := func(e embed.Embedder) { ptr.Store(&e) }

	return bridge, wire
}
```

Create `internal/cli/embed_bridge_test.go`:

```go
package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestBridgeEmbedder_UnwiredFallbacks asserts the pre-wiring behavior
// mirrors LazyEmbedder's pre-init semantics.
func TestBridgeEmbedder_UnwiredFallbacks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, _ := cli.ExportNewBridgeEmbedder()

	g.Expect(bridge.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(bridge.Dims()).To(Equal(0))

	_, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("embedder not wired")))
}

// TestBridgeEmbedder_DelegatesAfterWire asserts all three methods forward
// to the wired embedder.
func TestBridgeEmbedder_DelegatesAfterWire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, wire := cli.ExportNewBridgeEmbedder()
	wire(bridgeStubEmbedder{})

	g.Expect(bridge.ModelID()).To(Equal("stub@4"))
	g.Expect(bridge.Dims()).To(Equal(4))

	vec, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(Equal([]float32{0, 0, 0, 0}))
}

type bridgeStubEmbedder struct{}

func (bridgeStubEmbedder) Dims() int { return 4 }

func (bridgeStubEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0, 0, 0, 0}, nil
}

func (bridgeStubEmbedder) ModelID() string { return "stub@4" }
```

- [ ] 11. **Verify.** Run `targ test` — expected: all green (internal runtime/cachefs suites green with fakes; real-os integration green; cmd smoke green; bridge tests green; `TestTargets_EmbedStatus` stays green via bridge → LazyEmbedder pre-init ModelID). Run `targ check-full` — expected: clean. Run `targ check-thin-api` — expected: PASS (cmd/engram/hugot.go's declarations are the verified-thin shapes from step 7; the main.go literal line is an argument expression). If the gate flags ANY declaration, ESCALATE the exact finding to the orchestrator — do not suppress, do not restructure ad hoc (Global Constraints / doctrine item 5). Confirm `grep -rn '"os"\|knights-analytics' internal/embed/*.go | grep -v _test` returns nothing. Run `go install ./cmd/engram && engram embed status --vault "$(mktemp -d)"` from a non-repo cwd — expected: the six status lines with all-zero counts (real-binary check per house rules).
- [ ] 12. **Commit:**

```
refactor(embed): internal backend/cache composition, thin cmd (#700)

internal/embed now owns ALL embedder orchestration: session/pipeline
lifecycle (NewRuntimeBackend over the raw Runtime seam) and cache
extraction + sentinel/perm policy + the errors.Is(fs.ErrExist) rename
classification (NewCacheFS over raw FS primitives). cmd/engram/hugot.go
shrinks to an EMPTY hugotRuntime struct with two single-call methods
(targ check-thin-api PASS); cli.NewDeps composes Deps.Embed from
Primitives internally (R6 3-arg flip, doctrine D-1/E-1/E-2). Dead
tempFS/unpackModelToTemp machinery deleted. cli gains the transitional
sharedEmbedder bridge wired from Deps.Embed.

AI-Used: [claude]
```

---

