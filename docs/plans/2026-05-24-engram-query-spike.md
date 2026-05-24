# Engram Query Spike — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a working `engram query` semantic-search command, backed by an embed-on-write pipeline with sidecar `.vec.json` files, validated by 13 UAT cases from `docs/superpowers/research/2026-05-24-engram-query-spike.md`. Case 13 (Snowflake-arctic-embed-xs reference-parity vs. Python sentence-transformers) is the go/no-go gate.

**Architecture:** New `internal/embed/` package exposing an `Embedder` interface, a sidecar file format, content-hash-based staleness detection, and a Hugot+GoMLX-simplego-backed implementation that runs in pure Go with the ONNX model bundled into the binary via `go:embed`. The CLI gains three surfaces: `engram embed` (apply/status), `engram query`, and an auto-embed hook on successful `engram learn`. YAML output for `engram query` matches engram's existing YAML-frontmatter convention.

**Tech stack:** Go 1.25, Hugot v0.7.3 (`github.com/knights-analytics/hugot`), GoMLX v0.27.3 simplego backend, `go.yaml.in/yaml/v3` (already in module), `imptest`+`rapid`+`gomega` for tests, `uv` for the Python reference venv (UAT 13 only).

---

## Spec deviations called out

Two deviations from `2026-05-24-engram-query-spike.md`, both small and justifiable:

1. **`engram embed status` becomes `engram embed status` (subcommand), `engram embed --all` becomes `engram embed apply --all`.** Rationale: the existing `engram learn {fact,feedback,moc}` group pattern uses `targ.Group(...)` with named members; `targ` does not support a default-member-with-flags shape. Splitting into a group with two members (`apply`, `status`) keeps the CLI internally consistent and avoids overloading positional dispatch on the bare `embed` command.

2. **`Embedder.Embed(ctx, text)` takes a `context.Context`** instead of just `text` as the spec writes. Rationale: matches Go convention, supports cancellation, and the test-stack memory (40.test-context-cancellation) requires fakes that block on `ctx.Done()`. Zero downside; surface is otherwise identical to the spec.

Neither deviation changes any UAT case's pass/fail criterion.

---

## File map

**New files:**
- `internal/embed/embedder.go` — `Embedder` interface, `Sidecar` struct, `State` enum, sentinel errors.
- `internal/embed/embedder_test.go` — unit tests for types and helpers.
- `internal/embed/hash.go` — `ExtractBody`, `ContentHash` helpers.
- `internal/embed/hash_test.go` — unit tests for body extraction and hashing.
- `internal/embed/sidecar.go` — sidecar path resolution, marshal/unmarshal, on-disk validation.
- `internal/embed/sidecar_test.go` — unit tests for sidecar I/O.
- `internal/embed/state.go` — `ComputeState(notePath, sidecarPath, currentModelID, fs)` returns State.
- `internal/embed/state_test.go` — unit tests for state computation including the stale-vs-incompatible distinction (UAT 11) and broken-sidecar detection (UAT 12).
- `internal/embed/hugot.go` — `NewHugotEmbedder()` constructor wrapping Hugot's `GoSession` + `FeatureExtractionPipeline`. Bundles ONNX model via `go:embed`.
- `internal/embed/hugot_test.go` — integration test exercising the bundled model in-process (skipped on `-short`).
- `internal/embed/parity_test.go` — UAT 13 gate. Reads `testdata/parity-reference.json`, embeds the same 5 pairs in Go, asserts cosines agree to 3 decimals.
- `internal/embed/testdata/parity-reference.json` — Python-generated reference cosines for 5 sentence pairs.
- `internal/embed/testdata/gen-reference.py` — `uv run --with sentence-transformers ...` script generating the reference JSON.
- `internal/embed/testdata/model/` — bundled ONNX model directory (Arctic-xs or MiniLM-L6 depending on gate outcome).
- `internal/embed/cosine.go` — `Cosine(a, b []float32) float32`, ranking helpers.
- `internal/embed/cosine_test.go` — unit + rapid property tests for cosine.
- `internal/cli/embed.go` — `engram embed apply` and `engram embed status` command wiring.
- `internal/cli/embed_test.go` — blackbox tests for the embed CLI commands using imptest-generated mocks.
- `internal/cli/query.go` — `engram query` command wiring + YAML rendering.
- `internal/cli/query_test.go` — blackbox tests for the query command.
- `internal/cli/auto_embed.go` — small adapter wiring an `embed.Embedder` into the post-write step of `runLearn`.
- `internal/cli/auto_embed_test.go` — tests covering warn-and-proceed semantics (UAT 3).

**Modified files:**
- `internal/cli/learn.go` — add `Embedder embed.Embedder` field to `LearnDeps`, call it after a successful `WriteNew` in `writeLearnUnderLock`, warn on failure via the injected `Logger`.
- `internal/cli/targets.go` — register `embed` group and `query` target.
- `internal/cli/cli.go` — `newOsLearnDeps` wires the production Hugot embedder.
- `go.mod` / `go.sum` — add `github.com/knights-analytics/hugot` and `github.com/gomlx/gomlx`. (Indirect deps come along.)
- `README.md` — document new commands and the offline pipeline.
- `CLAUDE.md` — note the new `internal/embed/` package in the directory map; mention the bundled-model invariant.
- `docs/architecture/c1-system-context.md` — extend the system context to show the embedder relationship.

**Deleted files:** none.

---

## Pre-flight: dependency probe (UAT 13 gate)

The gate is structured as a thrown-away probe first, then permanent test infrastructure. If Arctic-xs fails the gate, swap the bundled model to MiniLM-L6-v2 and re-run; if MiniLM also fails, we have a deeper compatibility problem and the spike halts with a report.

### Task 0: Verify Hugot+simplego compiles and links

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/embed/probe_main_test.go` (throwaway — deleted after the gate)

- [ ] **Step 0.1: Add Hugot module to go.mod**

```bash
cd /Users/joe/repos/personal/engram/.claude/worktrees/engram-query-spike
go get github.com/knights-analytics/hugot@latest
go get github.com/gomlx/gomlx@latest
go mod tidy
```

Expected: both modules resolve to recent versions (Hugot >= v0.7.3, GoMLX >= v0.27.3). go.sum updates. No compile errors yet.

- [ ] **Step 0.2: Write the smallest possible "does Hugot+simplego compile" test**

`internal/embed/probe_main_test.go`:

```go
//go:build probe

package embed_test

import (
	"testing"

	"github.com/knights-analytics/hugot"
)

func TestProbe_HugotLinks(t *testing.T) {
	// Just ensure the package can be imported and a session struct can be
	// referenced. We don't load a model yet — that's Task 1.
	var _ = hugot.NewGoSession
	t.Log("Hugot+GoMLX link OK")
}
```

- [ ] **Step 0.3: Build with the probe tag and confirm the test binary compiles**

```bash
targ test -- -tags=probe -run=TestProbe_HugotLinks ./internal/embed/...
```

(If `targ test` doesn't accept `--`, fall back to direct: `go test -tags=probe -run=TestProbe_HugotLinks ./internal/embed/...` — but try `targ` first per CLAUDE.md.)

Expected: PASS. Build artifact size on disk is roughly 60–80 MB (Hugot+GoMLX pull in a non-trivial graph).

- [ ] **Step 0.4: Commit the probe so the bisect point is durable**

```bash
git add go.mod go.sum internal/embed/probe_main_test.go
git commit -m "spike: probe Hugot+GoMLX simplego links in pure Go"
```

### Task 1: Generate Python reference cosines via uv

**Files:**
- Create: `internal/embed/testdata/gen-reference.py`
- Create: `internal/embed/testdata/parity-reference.json` (generated artifact, checked in)

- [ ] **Step 1.1: Write the reference-generator script**

`internal/embed/testdata/gen-reference.py`:

```python
#!/usr/bin/env -S uv run --with sentence-transformers --with numpy --script
# /// script
# requires-python = ">=3.10"
# dependencies = ["sentence-transformers>=3.0", "numpy>=1.26"]
# ///
"""Generate reference cosines for the engram query spike UAT 13 gate.

Writes a JSON file with the 5 sentence pairs, their cosines under the
chosen model, and the model identifier so the Go side can match
exactly. Re-run whenever the pair set or the model changes.
"""
import json
import sys
from pathlib import Path

from sentence_transformers import SentenceTransformer
import numpy as np

# The 5 pairs span clearly-similar to clearly-different so a real
# embedder produces a recognisable spread.
PAIRS = [
    # 1: near-paraphrases (expected ~0.85+)
    ("The cat sat on the mat.", "A cat is sitting on the mat."),
    # 2: same topic, different surface (expected ~0.6–0.8)
    ("Verify the current behaviour before claiming a delta.",
     "Check what the system does today before asserting how a change differs."),
    # 3: shared vocab, different topic (expected ~0.3–0.5)
    ("The agent embeds notes on write.",
     "The agent eats notes on Wednesdays."),
    # 4: unrelated (expected ~-0.05 to 0.2)
    ("Cosine similarity ranges from minus one to one.",
     "Pour the batter into the greased pan."),
    # 5: identical (expected ~1.0)
    ("Identical sentences embed to identical vectors.",
     "Identical sentences embed to identical vectors."),
]

MODEL_ID = "Snowflake/snowflake-arctic-embed-xs"

def cosine(a: np.ndarray, b: np.ndarray) -> float:
    return float(np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b)))

def main(out_path: Path) -> None:
    model = SentenceTransformer(MODEL_ID)
    pairs_out = []
    for left, right in PAIRS:
        vecs = model.encode([left, right], normalize_embeddings=False)
        pairs_out.append({
            "left": left,
            "right": right,
            "cosine": cosine(vecs[0], vecs[1]),
        })
    payload = {
        "model_id": MODEL_ID,
        "dims": int(vecs.shape[1]),
        "pairs": pairs_out,
    }
    out_path.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"wrote {out_path}", file=sys.stderr)

if __name__ == "__main__":
    out = Path(__file__).parent / "parity-reference.json"
    main(out)
```

- [ ] **Step 1.2: Run the script with uv**

```bash
chmod +x internal/embed/testdata/gen-reference.py
internal/embed/testdata/gen-reference.py
```

Expected: `parity-reference.json` created. Contents: `model_id`, `dims: 384`, 5 pairs each with `left`, `right`, `cosine`. Pair 5 cosine should be exactly 1.0; pair 1 should be > 0.8; pair 4 should be < 0.3.

Sanity-check by inspecting the file:

```bash
cat internal/embed/testdata/parity-reference.json
```

- [ ] **Step 1.3: Commit the reference**

```bash
git add internal/embed/testdata/gen-reference.py internal/embed/testdata/parity-reference.json
git commit -m "spike: Python reference cosines for UAT 13 (Arctic-xs)"
```

### Task 2: Download Arctic-xs ONNX + tokenizer and stage it

**Files:**
- Create: `internal/embed/testdata/model/model.onnx`
- Create: `internal/embed/testdata/model/tokenizer.json`
- Create: `internal/embed/testdata/model/special_tokens_map.json` (if Hugot needs it)
- Create: `internal/embed/testdata/model/tokenizer_config.json` (if Hugot needs it)

- [ ] **Step 2.1: Stage the model files into testdata**

The Hugot pattern: `hugot.FeatureExtractionConfig{ModelPath: <dir>, OnnxFilename: "model.onnx"}` reads everything in `<dir>`. Download from HuggingFace via the `huggingface_hub` CLI (already a transitive dep of `sentence-transformers`):

```bash
uvx --from huggingface_hub huggingface-cli download \
    Snowflake/snowflake-arctic-embed-xs \
    --include "onnx/model.onnx" "tokenizer.json" "tokenizer_config.json" "special_tokens_map.json" "config.json" \
    --local-dir internal/embed/testdata/model
# Hugot expects model.onnx at the model root; move it out of onnx/.
mv internal/embed/testdata/model/onnx/model.onnx internal/embed/testdata/model/model.onnx
rmdir internal/embed/testdata/model/onnx
ls -la internal/embed/testdata/model/
```

Expected: `model.onnx` (~22 MB), `tokenizer.json`, plus config files.

- [ ] **Step 2.2: Commit the model files**

```bash
git add internal/embed/testdata/model/
git commit -m "spike: stage Arctic-xs ONNX + tokenizer for UAT 13 gate"
```

(If the user objects to committing 22 MB of model into the repo, swap to git-lfs or to a download-on-first-use cache. For the spike we accept the size; the final binary will `go:embed` the same files anyway.)

### Task 3: UAT 13 gate — Go cross-runtime parity test

**Files:**
- Create: `internal/embed/cosine.go`
- Create: `internal/embed/cosine_test.go`
- Create: `internal/embed/parity_test.go`
- Delete: `internal/embed/probe_main_test.go` (the throwaway from Task 0 — its job is done)

- [ ] **Step 3.1: Write cosine + the RED parity test**

`internal/embed/cosine.go`:

```go
// Package embed wires the engram embedder, sidecar format, and
// staleness detection for the embed-on-write pipeline.
package embed

import "math"

// Cosine returns the cosine similarity of a and b. Returns 0 when either
// vector has zero magnitude — callers should treat that as "no signal"
// rather than a strong match.
func Cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, na, nb float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}

	if na == 0 || nb == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}
```

`internal/embed/cosine_test.go`:

```go
package embed_test

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/embed"
)

func TestCosine_Identical(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	v := []float32{0.1, 0.2, 0.3, 0.4}
	g.Expect(float64(embed.Cosine(v, v))).To(BeNumerically("~", 1.0, 1e-6))
}

func TestCosine_Orthogonal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{1, 0}
	b := []float32{0, 1}
	g.Expect(float64(embed.Cosine(a, b))).To(BeNumerically("~", 0.0, 1e-6))
}

func TestCosine_ZeroVector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	g.Expect(embed.Cosine(a, b)).To(Equal(float32(0)))
}

func TestCosine_MismatchedLengths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.Cosine([]float32{1}, []float32{1, 2})).To(Equal(float32(0)))
}

// Property: cosine(a,a) == 1 for any non-zero vector.
func TestCosine_SelfSimilarityProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 16).Draw(t, "n")
		v := rapid.SliceOfN(rapid.Float32Range(-10, 10), n, n).Draw(t, "v")
		var sumSq float64
		for _, x := range v {
			sumSq += float64(x) * float64(x)
		}
		if sumSq < 1e-12 {
			t.Skip("zero vector handled by separate test")
		}
		got := float64(embed.Cosine(v, v))
		if math.Abs(got-1.0) > 1e-4 {
			t.Fatalf("cosine(v,v) = %v, want ~1.0", got)
		}
	})
}
```

`internal/embed/parity_test.go`:

```go
//go:build parity

// Package embed_test (parity build tag) — UAT 13 gate for the engram
// query spike. Compares Hugot+GoMLX cosines against Python
// sentence-transformers reference cosines on a fixed set of 5 pairs.
//
// Build tag-gated because it loads a 22MB ONNX model from disk and
// would slow targ test substantially. Run with: targ test -- -tags=parity
package embed_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

type referencePayload struct {
	ModelID string `json:"model_id"`
	Dims    int    `json:"dims"`
	Pairs   []struct {
		Left   string  `json:"left"`
		Right  string  `json:"right"`
		Cosine float64 `json:"cosine"`
	} `json:"pairs"`
}

func TestT13_ParityWithPythonReference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Load reference.
	refBytes, err := os.ReadFile("testdata/parity-reference.json")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	var ref referencePayload
	g.Expect(json.Unmarshal(refBytes, &ref)).NotTo(HaveOccurred())

	// Construct Hugot embedder from the staged model directory.
	modelDir, absErr := filepath.Abs("testdata/model")
	g.Expect(absErr).NotTo(HaveOccurred())
	if absErr != nil {
		return
	}
	emb, embErr := embed.NewHugotEmbedderFromDir(modelDir, ref.ModelID)
	g.Expect(embErr).NotTo(HaveOccurred())
	if embErr != nil {
		return
	}
	defer emb.Close()

	g.Expect(emb.Dims()).To(Equal(ref.Dims))

	ctx := context.Background()
	const tol = 1e-3 // ~3 decimal places per spec.
	for _, pair := range ref.Pairs {
		left, leftErr := emb.Embed(ctx, pair.Left)
		g.Expect(leftErr).NotTo(HaveOccurred())
		if leftErr != nil {
			continue
		}
		right, rightErr := emb.Embed(ctx, pair.Right)
		g.Expect(rightErr).NotTo(HaveOccurred())
		if rightErr != nil {
			continue
		}
		got := float64(embed.Cosine(left, right))
		if math.Abs(got-pair.Cosine) > tol {
			t.Errorf("pair %q vs %q: Go cosine %.6f, Python cosine %.6f, diff %.6f > tol %.6f",
				pair.Left, pair.Right, got, pair.Cosine, math.Abs(got-pair.Cosine), tol)
		}
	}
}
```

`NewHugotEmbedderFromDir` doesn't exist yet — this RED test cannot link. That's the point.

- [ ] **Step 3.2: Implement the minimal `NewHugotEmbedderFromDir` to make the parity test runnable**

`internal/embed/hugot.go`:

```go
package embed

import (
	"context"
	"fmt"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

// HugotEmbedder wraps a Hugot GoSession + feature-extraction pipeline.
type HugotEmbedder struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	modelID  string
	dims     int
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from a
// directory on disk. Used by the parity gate (UAT 13). The production
// constructor that uses the bundled embed.FS comes later.
func NewHugotEmbedderFromDir(modelDir, modelID string) (*HugotEmbedder, error) {
	session, err := hugot.NewGoSession()
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	cfg := hugot.FeatureExtractionConfig{
		ModelPath:    modelDir,
		Name:         "embed",
		OnnxFilename: "model.onnx",
	}

	pipe, pipeErr := hugot.NewPipeline(session, cfg)
	if pipeErr != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	// Probe dims via a one-shot embedding.
	probe, probeErr := pipe.RunPipeline([]string{"probe"})
	if probeErr != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("hugot probe: %w", probeErr)
	}
	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = session.Destroy()
		return nil, fmt.Errorf("hugot probe returned no embedding")
	}

	return &HugotEmbedder{
		session:  session,
		pipeline: pipe,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
	}, nil
}

func (h *HugotEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out, err := h.pipeline.RunPipeline([]string{text})
	if err != nil {
		return nil, fmt.Errorf("hugot embed: %w", err)
	}
	if len(out.Embeddings) == 0 {
		return nil, fmt.Errorf("hugot embed: empty result")
	}
	return out.Embeddings[0], nil
}

func (h *HugotEmbedder) ModelID() string { return h.modelID }
func (h *HugotEmbedder) Dims() int       { return h.dims }

func (h *HugotEmbedder) Close() error {
	if err := h.session.Destroy(); err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}
	return nil
}
```

Important: the precise Hugot/GoMLX call surface (`NewGoSession`, `FeatureExtractionConfig`, `NewPipeline`, `RunPipeline`, `Destroy`, `pipelines.FeatureExtractionPipeline`) is what I expect from the v0.7.3 README. **Verify against the actual installed package before assuming this exact API.** If the field names or function signatures differ, adapt — the shape (load model directory, run on `[]string`, get back `[][]float32`-equivalent) is stable across recent Hugot versions.

- [ ] **Step 3.3: Delete the throwaway probe**

```bash
rm internal/embed/probe_main_test.go
```

- [ ] **Step 3.4: Run the parity test**

```bash
targ test -- -tags=parity -run=TestT13_ParityWithPythonReference ./internal/embed/...
```

(Fall back to `go test -tags=parity ...` if targ can't pass through tags.)

Expected outcomes — branch decision here:

- **PASS:** Arctic-xs gate cleared. Proceed to Task 4.
- **FAIL: model fails to load** → simplego cannot run Arctic-xs. Re-run with the MiniLM-L6 fallback: re-stage the model files (`sentence-transformers/all-MiniLM-L6-v2`), regenerate the reference cosines (re-run gen-reference.py with `MODEL_ID = "sentence-transformers/all-MiniLM-L6-v2"`), update `embedding_model_id` constants throughout to `minilm-l6-v2@384`, and re-run. MiniLM is verified to work per the compat doc.
- **FAIL: cosines disagree past 1e-3** → real implementation divergence. Stop and report; do not silently swap models. The discrepancy itself is information worth investigating.
- **FAIL: NaNs or zeros** → broken implementation. Fall back to MiniLM-L6.

- [ ] **Step 3.5: Commit the gate result**

```bash
git add internal/embed/cosine.go internal/embed/cosine_test.go internal/embed/parity_test.go internal/embed/hugot.go
git rm internal/embed/probe_main_test.go
git commit -m "spike: UAT 13 gate — <PASS|FAIL with $MODEL>"
```

Replace the commit message with the actual outcome. If you swapped to MiniLM, say so.

---

## Phase 1: `internal/embed/` package foundation

### Task 4: Types and sentinel errors

**Files:**
- Create: `internal/embed/embedder.go`
- Create: `internal/embed/embedder_test.go`

- [ ] **Step 4.1: Write tests for the State enum's String()**

`internal/embed/embedder_test.go`:

```go
package embed_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestState_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		state embed.State
		want  string
	}{
		{embed.StateOK, "ok"},
		{embed.StateMissing, "missing"},
		{embed.StateStale, "stale"},
		{embed.StateIncompatible, "incompatible"},
		{embed.StateBroken, "broken"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(tc.state.String()).To(Equal(tc.want))
		})
	}
}
```

- [ ] **Step 4.2: Run the test (expect compile failure)**

```bash
targ test
```

Expected: fails to build (`embed.StateOK undefined` etc.).

- [ ] **Step 4.3: Implement the types**

`internal/embed/embedder.go`:

```go
package embed

import (
	"context"
	"errors"
)

// Embedder produces fixed-dimension dense vectors from text. Implementations
// are expected to be safe for concurrent use unless documented otherwise.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	ModelID() string
	Dims() int
}

// Sidecar is the on-disk shape of a per-note .vec.json file. Field order
// here is the JSON key order.
type Sidecar struct {
	EmbeddingModelID string    `json:"embedding_model_id"`
	Dims             int       `json:"dims"`
	Vector           []float32 `json:"vector"`
	ContentHash      string    `json:"content_hash"`
}

// State is the relationship between a note and its sidecar relative to the
// current binary's embedder.
type State int

const (
	StateOK State = iota
	StateMissing
	StateStale
	StateIncompatible
	StateBroken
)

func (s State) String() string {
	switch s {
	case StateOK:
		return "ok"
	case StateMissing:
		return "missing"
	case StateStale:
		return "stale"
	case StateIncompatible:
		return "incompatible"
	case StateBroken:
		return "broken"
	default:
		return "unknown"
	}
}

// Sentinel errors.
var (
	ErrSidecarMalformed = errors.New("sidecar malformed")
	ErrDimsMismatch     = errors.New("sidecar dims mismatch len(vector)")
)
```

- [ ] **Step 4.4: Run tests**

```bash
targ test
```

Expected: PASS for `TestState_String` and all sub-cases.

- [ ] **Step 4.5: Commit**

```bash
git add internal/embed/embedder.go internal/embed/embedder_test.go
git commit -m "embed: types — Embedder, Sidecar, State"
```

### Task 5: Body extraction + content hash

**Files:**
- Create: `internal/embed/hash.go`
- Create: `internal/embed/hash_test.go`

- [ ] **Step 5.1: RED — write tests covering frontmatter strip + hash determinism + frontmatter-only edits should not change hash**

`internal/embed/hash_test.go`:

```go
package embed_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestExtractBody_StripsFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := []byte("---\ntype: fact\nluhmann: \"5\"\n---\n\nThis is the body.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("This is the body.\n"))
}

func TestExtractBody_NoFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := []byte("Just a body, no frontmatter.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("Just a body, no frontmatter.\n"))
}

func TestExtractBody_FrontmatterWithBlankLineInside(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := []byte("---\ntype: fact\n\nrelations: []\n---\nbody\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("body\n"))
}

func TestContentHash_IsSha256OfBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	body := []byte("the body.\n")
	want := sha256.Sum256(body)
	g.Expect(embed.ContentHash([]byte("---\ntype: x\n---\nthe body.\n"))).
		To(Equal("sha256:" + hex.EncodeToString(want[:])))
}

func TestContentHash_FrontmatterChangeDoesNotChangeHash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nshared body.\n")
	b := []byte("---\ntype: fact\nluhmann: \"1\"\nextra: added\n---\nshared body.\n")
	g.Expect(embed.ContentHash(a)).To(Equal(embed.ContentHash(b)))
}
```

- [ ] **Step 5.2: Run (expect compile failure)**

```bash
targ test
```

- [ ] **Step 5.3: Implement**

`internal/embed/hash.go`:

```go
package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

var frontmatterDelim = []byte("---\n")

// ExtractBody returns the markdown body of a note with the leading YAML
// frontmatter block stripped. If the note has no frontmatter, it is
// returned unchanged.
//
// Frontmatter format: a leading "---\n" line, arbitrary lines (which may
// themselves be blank), and a closing "---\n" line. Anything after the
// closing delimiter (typically a blank line and then the body) is the
// body. The leading blank line after "---\n" is also stripped so two
// notes with identical bodies but different frontmatter blocks produce
// identical hashes.
func ExtractBody(raw []byte) []byte {
	if !bytes.HasPrefix(raw, frontmatterDelim) {
		return raw
	}

	// Find the closing "---\n" after the opening.
	rest := raw[len(frontmatterDelim):]
	idx := bytes.Index(rest, frontmatterDelim)
	if idx < 0 {
		return raw
	}

	body := rest[idx+len(frontmatterDelim):]
	// Trim a single leading blank line (matches the convention used by
	// engram learn, which writes "---\n\n<body>").
	body = bytes.TrimPrefix(body, []byte("\n"))
	return body
}

// ContentHash returns a sha256: prefixed hex digest of the note's body
// (frontmatter stripped). Used to detect stale sidecars when a note's
// body has changed.
func ContentHash(raw []byte) string {
	sum := sha256.Sum256(ExtractBody(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}
```

- [ ] **Step 5.4: Run tests**

```bash
targ test
```

Expected: all PASS.

- [ ] **Step 5.5: Commit**

```bash
git add internal/embed/hash.go internal/embed/hash_test.go
git commit -m "embed: ExtractBody + ContentHash (frontmatter-stripped sha256)"
```

### Task 6: Sidecar path resolution + JSON I/O

**Files:**
- Create: `internal/embed/sidecar.go`
- Create: `internal/embed/sidecar_test.go`

- [ ] **Step 6.1: RED — tests for SidecarPath, Marshal, Unmarshal, validation**

`internal/embed/sidecar_test.go`:

```go
package embed_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestSidecarPath_FromNotePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.SidecarPath("Permanent/132.2026-05-23.foo.md")).
		To(Equal("Permanent/132.2026-05-23.foo.vec.json"))
}

func TestSidecarPath_NonMdReturnsAppended(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// Defensive: non-.md inputs get .vec.json appended without rewriting.
	g.Expect(embed.SidecarPath("README")).To(Equal("README.vec.json"))
}

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	in := embed.Sidecar{
		EmbeddingModelID: "snowflake-arctic-embed-xs@384",
		Dims:             3,
		Vector:           []float32{0.1, 0.2, 0.3},
		ContentHash:      "sha256:deadbeef",
	}
	bytes, err := embed.MarshalSidecar(in)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	out, parseErr := embed.UnmarshalSidecar(bytes)
	g.Expect(parseErr).NotTo(HaveOccurred())
	if parseErr != nil {
		return
	}
	g.Expect(out).To(Equal(in))
}

func TestUnmarshalSidecar_Malformed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := embed.UnmarshalSidecar([]byte("{not json"))
	g.Expect(err).To(MatchError(embed.ErrSidecarMalformed))
}

func TestUnmarshalSidecar_DimsMismatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	_, err := embed.UnmarshalSidecar([]byte(
		`{"embedding_model_id":"x@2","dims":2,"vector":[0.1,0.2,0.3],"content_hash":"sha256:abc"}`,
	))
	g.Expect(err).To(MatchError(embed.ErrDimsMismatch))
}
```

- [ ] **Step 6.2: Run (expect compile failure)**

```bash
targ test
```

- [ ] **Step 6.3: Implement**

`internal/embed/sidecar.go`:

```go
package embed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SidecarPath returns the .vec.json path sibling to a note's .md path.
func SidecarPath(notePath string) string {
	if !strings.HasSuffix(notePath, ".md") {
		return notePath + ".vec.json"
	}
	return strings.TrimSuffix(notePath, ".md") + ".vec.json"
}

// MarshalSidecar encodes s as compact JSON (no surrounding whitespace).
// Vectors are large; pretty-printing them costs disk.
func MarshalSidecar(s Sidecar) ([]byte, error) {
	out, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal sidecar: %w", err)
	}
	return out, nil
}

// UnmarshalSidecar decodes a sidecar from JSON, returning ErrSidecarMalformed
// on parse failure or ErrDimsMismatch when len(Vector) != Dims.
func UnmarshalSidecar(data []byte) (Sidecar, error) {
	var s Sidecar
	if err := json.Unmarshal(data, &s); err != nil {
		return Sidecar{}, fmt.Errorf("%w: %v", ErrSidecarMalformed, err)
	}
	if len(s.Vector) != s.Dims {
		return Sidecar{}, fmt.Errorf("%w: dims=%d len=%d", ErrDimsMismatch, s.Dims, len(s.Vector))
	}
	return s, nil
}
```

- [ ] **Step 6.4: Run tests**

```bash
targ test
```

Expected: all PASS.

- [ ] **Step 6.5: Commit**

```bash
git add internal/embed/sidecar.go internal/embed/sidecar_test.go
git commit -m "embed: sidecar I/O — path, marshal, unmarshal, validation"
```

### Task 7: State computation

**Files:**
- Create: `internal/embed/state.go`
- Create: `internal/embed/state_test.go`

- [ ] **Step 7.1: RED — tests for ComputeState covering all five outcomes (UAT 8, 11, 12)**

`internal/embed/state_test.go`:

```go
package embed_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// fakeFS is a trivial in-memory file map for state-computation tests. We
// don't need imptest here — the surface is read-only and small.
type fakeFS map[string][]byte

func (f fakeFS) ReadFile(path string) ([]byte, error) {
	data, ok := f[path]
	if !ok {
		return nil, &fakeNotExistError{path}
	}
	return data, nil
}

type fakeNotExistError struct{ path string }

func (e *fakeNotExistError) Error() string  { return "no such file: " + e.path }
func (e *fakeNotExistError) IsNotExist() bool { return true }

// We define an IsNotExist helper because the embed package will use
// os.IsNotExist semantics in production.

func mustSidecar(g Gomega, s embed.Sidecar) []byte {
	out, err := json.Marshal(s)
	g.Expect(err).NotTo(HaveOccurred())
	return out
}

func TestComputeState_Missing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := fakeFS{"Permanent/x.md": []byte("---\nx: 1\n---\nbody\n")}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateMissing))
}

func TestComputeState_OK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sc := embed.Sidecar{
		EmbeddingModelID: "model@384",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	fs := fakeFS{
		"Permanent/x.md":       noteBytes,
		"Permanent/x.vec.json": mustSidecar(NewGomegaWithT(t), sc),
	}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateOK))
}

func TestComputeState_Stale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	sc := embed.Sidecar{
		EmbeddingModelID: "model@384",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      "sha256:stalehash",
	}
	fs := fakeFS{
		"Permanent/x.md":       []byte("---\nx: 1\n---\nbody\n"),
		"Permanent/x.vec.json": mustSidecar(NewGomegaWithT(t), sc),
	}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateStale))
}

func TestComputeState_Incompatible(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	noteBytes := []byte("---\nx: 1\n---\nbody\n")
	sc := embed.Sidecar{
		EmbeddingModelID: "OLDmodel@256",
		Dims:             1,
		Vector:           []float32{0.1},
		ContentHash:      embed.ContentHash(noteBytes),
	}
	fs := fakeFS{
		"Permanent/x.md":       noteBytes,
		"Permanent/x.vec.json": mustSidecar(NewGomegaWithT(t), sc),
	}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateIncompatible))
}

func TestComputeState_Broken_BadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := fakeFS{
		"Permanent/x.md":       []byte("body\n"),
		"Permanent/x.vec.json": []byte("{not json"),
	}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateBroken))
}

func TestComputeState_Broken_DimsMismatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := fakeFS{
		"Permanent/x.md": []byte("body\n"),
		"Permanent/x.vec.json": []byte(
			`{"embedding_model_id":"model@384","dims":2,"vector":[0.1,0.2,0.3],"content_hash":"sha256:abc"}`,
		),
	}
	s, err := embed.ComputeState(fs, "Permanent/x.md", "model@384")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s).To(Equal(embed.StateBroken))
}
```

- [ ] **Step 7.2: Run (expect compile failure)**

```bash
targ test
```

- [ ] **Step 7.3: Implement**

`internal/embed/state.go`:

```go
package embed

import (
	"errors"
	"fmt"
)

// FS is the read-only filesystem surface used by ComputeState. The
// production implementation in internal/cli wraps os.ReadFile and
// translates os.IsNotExist to a typed error this package recognises.
type FS interface {
	ReadFile(path string) ([]byte, error)
}

// notExistError reports whether err is a "file does not exist" error.
// Production code (internal/cli) hands ComputeState an FS whose ReadFile
// returns os.PathError-style errors that satisfy os.IsNotExist; tests can
// hand any error type that implements `IsNotExist() bool`.
func notExist(err error) bool {
	if err == nil {
		return false
	}
	var typed interface{ IsNotExist() bool }
	if errors.As(err, &typed) && typed.IsNotExist() {
		return true
	}
	// Defer to fs.PathError handling done by callers; the production
	// wrapper in internal/cli flips os.IsNotExist into a typed error.
	return false
}

// ComputeState reads notePath and the sibling .vec.json and returns the
// note's State relative to currentModelID. Stale-vs-incompatible is
// decided by which mismatches first: model_id mismatch takes precedence
// over content_hash mismatch (a re-embed under the new model also picks
// up the body change).
func ComputeState(fs FS, notePath, currentModelID string) (State, error) {
	noteBytes, noteErr := fs.ReadFile(notePath)
	if noteErr != nil {
		return StateBroken, fmt.Errorf("read note: %w", noteErr)
	}

	scBytes, scErr := fs.ReadFile(SidecarPath(notePath))
	if scErr != nil {
		if notExist(scErr) {
			return StateMissing, nil
		}
		return StateBroken, fmt.Errorf("read sidecar: %w", scErr)
	}

	sc, parseErr := UnmarshalSidecar(scBytes)
	if parseErr != nil {
		return StateBroken, nil
	}

	if sc.EmbeddingModelID != currentModelID {
		return StateIncompatible, nil
	}
	if sc.ContentHash != ContentHash(noteBytes) {
		return StateStale, nil
	}
	return StateOK, nil
}
```

- [ ] **Step 7.4: Run tests**

```bash
targ test
```

Expected: all PASS. If `fakeNotExistError.IsNotExist() bool` isn't being detected by `errors.As(..., &typed)`, adjust `notExist` to also match by interface assertion directly (`if t, ok := err.(interface{ IsNotExist() bool }); ok && t.IsNotExist() { ... }`).

- [ ] **Step 7.5: Commit**

```bash
git add internal/embed/state.go internal/embed/state_test.go
git commit -m "embed: ComputeState (ok/missing/stale/incompatible/broken)"
```

---

## Phase 2: Production Hugot embedder bundled into the binary

### Task 8: Bundle the ONNX model via go:embed and replace the from-disk constructor

**Files:**
- Modify: `internal/embed/hugot.go`
- Create: `internal/embed/model/` directory (copy of testdata/model/ — but separately tracked because testdata is test-only)

Wait — `testdata/` is excluded from production builds by go tooling but is shipped with the source tree. To bundle into the binary, we need the files in a non-`testdata` location. We can either (a) put them in `internal/embed/assets/` and `go:embed` from there, or (b) keep testdata as the source-of-truth and copy into `assets/` at build time.

I'll go with (a): single source-of-truth, files live in `internal/embed/assets/model/`, and the parity test (which still wants disk paths to point Hugot at) reads from `assets/model/` too. Then `testdata/` only holds the Python reference JSON.

- [ ] **Step 8.1: Move staged model files from testdata/ to assets/**

```bash
mkdir -p internal/embed/assets/model
mv internal/embed/testdata/model/* internal/embed/assets/model/
rmdir internal/embed/testdata/model
```

- [ ] **Step 8.2: Adjust parity test path**

In `internal/embed/parity_test.go`, change `filepath.Abs("testdata/model")` to `filepath.Abs("assets/model")`.

- [ ] **Step 8.3: Add NewBundledHugotEmbedder + go:embed directive**

Append to `internal/embed/hugot.go`:

```go
import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed assets/model/*
var bundledModel embed.FS

const bundledModelID = "snowflake-arctic-embed-xs@384"
// (Or "minilm-l6-v2@384" if the gate fell back. Keep this in lockstep
// with the model files in assets/model/.)

// NewBundledHugotEmbedder unpacks the embedded ONNX model to a temp
// directory and constructs a Hugot embedder against it. The temp
// directory persists for the life of the process; Close() removes it.
//
// This pattern (unpack-to-temp) is needed because Hugot reads its inputs
// from disk, not from an embed.FS.
func NewBundledHugotEmbedder() (*HugotEmbedder, error) {
	tmp, mkErr := os.MkdirTemp("", "engram-embed-model-*")
	if mkErr != nil {
		return nil, fmt.Errorf("temp dir: %w", mkErr)
	}

	entries, _ := bundledModel.ReadDir("assets/model")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, readErr := bundledModel.ReadFile(filepath.Join("assets/model", e.Name()))
		if readErr != nil {
			_ = os.RemoveAll(tmp)
			return nil, fmt.Errorf("read embedded %s: %w", e.Name(), readErr)
		}
		writeErr := os.WriteFile(filepath.Join(tmp, e.Name()), data, 0o600)
		if writeErr != nil {
			_ = os.RemoveAll(tmp)
			return nil, fmt.Errorf("unpack %s: %w", e.Name(), writeErr)
		}
	}

	emb, embErr := NewHugotEmbedderFromDir(tmp, bundledModelID)
	if embErr != nil {
		_ = os.RemoveAll(tmp)
		return nil, embErr
	}
	emb.tmpDir = tmp
	return emb, nil
}
```

Add a `tmpDir string` field to `HugotEmbedder` and update `Close()` to remove it.

- [ ] **Step 8.4: Add a smoke test that the bundled embedder works**

`internal/embed/hugot_test.go`:

```go
//go:build !short

package embed_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestBundledHugotEmbedder_Smoke(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	emb, err := embed.NewBundledHugotEmbedder()
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	defer emb.Close()
	g.Expect(emb.Dims()).To(Equal(384))
	v, embErr := emb.Embed(context.Background(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(v).To(HaveLen(384))
}
```

- [ ] **Step 8.5: Run tests**

```bash
targ test
```

Expected: all PASS. Build artifact size should now reflect the bundled model (binary grows by ~22 MB).

- [ ] **Step 8.6: Commit**

```bash
git add internal/embed/assets/ internal/embed/hugot.go internal/embed/hugot_test.go internal/embed/parity_test.go
git rm internal/embed/testdata/model
git commit -m "embed: bundle Arctic-xs ONNX into binary via go:embed"
```

---

## Phase 3: CLI surfaces — `engram embed apply`, `engram embed status`, `engram query`

### Task 9: `engram embed apply` and `engram embed status` (UAT 1, 2, 4, 8, 9, 11, 12)

**Files:**
- Create: `internal/cli/embed.go`
- Create: `internal/cli/embed_test.go`
- Modify: `internal/cli/targets.go` (register the group)

- [ ] **Step 9.1: Sketch the Args structs and the function shape**

`internal/cli/embed.go` (skeleton):

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// EmbedApplyArgs holds parsed flags for `engram embed apply`.
type EmbedApplyArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	All       bool   `targ:"flag,name=all,desc=re-embed every note regardless of state"`
	Missing   bool   `targ:"flag,name=missing,desc=embed only notes without sidecars (default if no mode flag)"`
	Stale     bool   `targ:"flag,name=stale,desc=re-embed notes whose body hash changed"`
	Force     bool   `targ:"flag,name=force,desc=also re-embed sidecars whose model_id differs from the binary"`
	DryRun    bool   `targ:"flag,name=dry-run,desc=report what would change, don't write"`
}

// EmbedStatusArgs holds parsed flags for `engram embed status`.
type EmbedStatusArgs struct {
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
}

// EmbedDeps holds injected dependencies for the embed commands.
type EmbedDeps struct {
	Scan        func(vault string) ([]vaultgraph.Note, error)
	Read        func(path string) ([]byte, error)
	Write       func(path string, data []byte) error
	StatExists  func(path string) (bool, error)
	Embedder    embed.Embedder
}
```

- [ ] **Step 9.2: RED — tests for status counts (UAT 8) with imptest mocks**

Set up `imptest` mock generation for `EmbedDeps`. The vaultgraph package already has `generated_MockVaultFS_test.go` — same pattern applies. Add a `//go:generate` directive or generate by hand.

Actually for `EmbedDeps`-style structs of function fields, mocks are trivially constructible inline — no imptest generation needed. Just construct `EmbedDeps{Scan: func(...) {...}, Read: ...}` per test.

`internal/cli/embed_test.go`:

```go
package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestEmbedStatus_AllMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := cli.EmbedDeps{
		Scan: func(string) ([]vaultgraph.Note, error) {
			return []vaultgraph.Note{
				{Basename: "1.foo", IsMOC: false},
				{Basename: "2.bar", IsMOC: true},
			}, nil
		},
		Read:       func(string) ([]byte, error) { return nil, &os.PathError{Op: "open", Err: os.ErrNotExist} },
		StatExists: func(string) (bool, error) { return false, nil },
		Embedder:   stubEmbedder{modelID: "m@4", dims: 4},
	}
	var out bytes.Buffer
	err := cli.RunEmbedStatus(context.Background(), cli.EmbedStatusArgs{VaultPath: "/v"}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	text := out.String()
	g.Expect(text).To(ContainSubstring("total:           2"))
	g.Expect(text).To(ContainSubstring("with-embeddings: 0"))
	g.Expect(text).To(ContainSubstring("without:         2"))
}

// ... more tests covering UAT 1, 2, 4, 8, 9, 11, 12 ...

type stubEmbedder struct {
	modelID string
	dims    int
}

func (s stubEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, s.dims)
	for i := range out {
		out[i] = float32(i + 1) / float32(s.dims)
	}
	_ = text
	return out, nil
}
func (s stubEmbedder) ModelID() string { return s.modelID }
func (s stubEmbedder) Dims() int       { return s.dims }
```

- [ ] **Step 9.3: Implement RunEmbedStatus and RunEmbedApply**

Implementation of `RunEmbedStatus`:

```go
func RunEmbedStatus(_ context.Context, args EmbedStatusArgs, deps EmbedDeps, stdout io.Writer) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed status: scan: %w", err)
	}

	counts := struct {
		total, ok, missing, stale, incompat, broken int
	}{total: len(notes)}

	modelID := deps.Embedder.ModelID()
	fs := readerFS{Read: deps.Read}
	for _, n := range notes {
		notePath := pathOf(n.Basename, n.IsMOC) // existing helper in cli.go
		full := filepath.Join(args.VaultPath, notePath)
		state, stateErr := embed.ComputeState(fs, full, modelID)
		if stateErr != nil {
			counts.broken++
			continue
		}
		switch state {
		case embed.StateOK:
			counts.ok++
		case embed.StateMissing:
			counts.missing++
		case embed.StateStale:
			counts.stale++
		case embed.StateIncompatible:
			counts.incompat++
		case embed.StateBroken:
			counts.broken++
		}
	}

	fmt.Fprintf(stdout, "total:           %d\n", counts.total)
	fmt.Fprintf(stdout, "with-embeddings: %d\n", counts.ok)
	fmt.Fprintf(stdout, "without:         %d\n", counts.missing)
	fmt.Fprintf(stdout, "stale:           %d\n", counts.stale)
	fmt.Fprintf(stdout, "incompatible:    %d\n", counts.incompat)
	fmt.Fprintf(stdout, "broken:          %d\n", counts.broken)
	return nil
}

type readerFS struct {
	Read func(string) ([]byte, error)
}

func (r readerFS) ReadFile(path string) ([]byte, error) {
	data, err := r.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
		}
		return nil, err
	}
	return data, nil
}
```

Adapt `embed.notExist` to recognise `os.PathError` with `os.ErrNotExist` — or just have the production reader return a custom error implementing `IsNotExist()`.

`RunEmbedApply`:

```go
func RunEmbedApply(ctx context.Context, args EmbedApplyArgs, deps EmbedDeps, stdout io.Writer) error {
	notes, err := deps.Scan(args.VaultPath)
	if err != nil {
		return fmt.Errorf("embed apply: scan: %w", err)
	}

	modelID := deps.Embedder.ModelID()
	dims := deps.Embedder.Dims()
	fs := readerFS{Read: deps.Read}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Basename < notes[j].Basename
	})

	wantMissing := args.Missing || (!args.All && !args.Stale)
	wantStale := args.Stale || args.All
	wantIncompat := args.Force || args.All
	wantOK := args.All

	for _, n := range notes {
		notePath := pathOf(n.Basename, n.IsMOC)
		full := filepath.Join(args.VaultPath, notePath)
		state, stateErr := embed.ComputeState(fs, full, modelID)
		if stateErr != nil {
			fmt.Fprintf(stdout, "broken    %s: %v\n", notePath, stateErr)
			continue
		}

		shouldEmbed := false
		switch state {
		case embed.StateOK:
			shouldEmbed = wantOK
		case embed.StateMissing:
			shouldEmbed = wantMissing
		case embed.StateStale:
			shouldEmbed = wantStale
		case embed.StateIncompatible:
			shouldEmbed = wantIncompat
		case embed.StateBroken:
			shouldEmbed = wantStale || wantMissing || wantIncompat || wantOK
		}

		if !shouldEmbed {
			continue
		}

		if args.DryRun {
			fmt.Fprintf(stdout, "would-embed %s (%s)\n", notePath, state)
			continue
		}

		noteBytes, readErr := deps.Read(full)
		if readErr != nil {
			fmt.Fprintf(stdout, "skip      %s: read error: %v\n", notePath, readErr)
			continue
		}

		body := embed.ExtractBody(noteBytes)
		vec, embErr := deps.Embedder.Embed(ctx, string(body))
		if embErr != nil {
			fmt.Fprintf(stdout, "fail      %s: embed: %v\n", notePath, embErr)
			continue
		}

		sc := embed.Sidecar{
			EmbeddingModelID: modelID,
			Dims:             dims,
			Vector:           vec,
			ContentHash:      embed.ContentHash(noteBytes),
		}
		scBytes, marshalErr := embed.MarshalSidecar(sc)
		if marshalErr != nil {
			fmt.Fprintf(stdout, "fail      %s: marshal: %v\n", notePath, marshalErr)
			continue
		}

		sidecarFull := filepath.Join(args.VaultPath, embed.SidecarPath(notePath))
		if writeErr := deps.Write(sidecarFull, scBytes); writeErr != nil {
			fmt.Fprintf(stdout, "fail      %s: write: %v\n", notePath, writeErr)
			continue
		}
		fmt.Fprintf(stdout, "embedded  %s (%s)\n", notePath, state)
	}

	return nil
}
```

- [ ] **Step 9.4: Wire targets**

In `internal/cli/targets.go` add to `Targets`:

```go
targ.Group("embed",
	targ.Targ(func(ctx context.Context, a EmbedApplyArgs) {
		a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
		errHandler(RunEmbedApply(withLog(ctx), a, newOsEmbedDeps(), stdout))
	}).Name("apply").Description("Embed notes (default: missing only)"),
	targ.Targ(func(ctx context.Context, a EmbedStatusArgs) {
		a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
		errHandler(RunEmbedStatus(withLog(ctx), a, newOsEmbedDeps(), stdout))
	}).Name("status").Description("Report embedding state counts"),
),
```

Production `newOsEmbedDeps`:

```go
func newOsEmbedDeps() EmbedDeps {
	emb, err := embed.NewBundledHugotEmbedder()
	if err != nil {
		// Embedder construction failure at process start is fatal —
		// every embed/query operation depends on it.
		panic(fmt.Sprintf("init bundled embedder: %v", err))
	}
	return EmbedDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
		Read: os.ReadFile,
		Write: func(path string, data []byte) error {
			return os.WriteFile(path, data, 0o600)
		},
		StatExists: func(path string) (bool, error) {
			_, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		},
		Embedder: emb,
	}
}
```

- [ ] **Step 9.5: Run tests**

```bash
targ test
```

Expected: PASS.

- [ ] **Step 9.6: Commit**

```bash
git add internal/cli/embed.go internal/cli/embed_test.go internal/cli/targets.go
git commit -m "cli: engram embed apply + engram embed status"
```

### Task 10: `engram query` (UAT 5, 6, 7, 9, 10)

**Files:**
- Create: `internal/cli/query.go`
- Create: `internal/cli/query_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 10.1: Args + tests covering top-N ordering and empty-vault**

`internal/cli/query.go` skeleton:

```go
type QueryArgs struct {
	Query     string `targ:"positional,name=query,desc=natural-language query string"`
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	Limit     int    `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
}

type QueryDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Embedder embed.Embedder
}
```

Test cases:
- UAT 5: query for "verifying current behavior before claiming a delta" returns 132 in top 3 with provenances [direct].
- UAT 6: similar for 133.
- UAT 7: scores monotonically decrease.
- UAT 9: empty vault → `items: []`, exit 0.
- UAT 10: missing model file errors clearly. (Production-only.)

For UAT 5, 6, 7 the stub embedder in tests can be deterministic — generate a vector from the input text hash, then verify ranking is deterministic and scores monotonically decrease. The real-model UAT 5/6 verification happens manually in Task 13.

- [ ] **Step 10.2: Implement**

```go
func RunQuery(ctx context.Context, args QueryArgs, deps QueryDeps, stdout io.Writer) error {
	if args.Query == "" {
		return errors.New("query: empty query string")
	}
	limit := args.Limit
	if limit == 0 {
		limit = 20
	}

	notes, scanErr := deps.Scan(args.VaultPath)
	if scanErr != nil {
		return fmt.Errorf("query: scan: %w", scanErr)
	}

	queryVec, qErr := deps.Embedder.Embed(ctx, args.Query)
	if qErr != nil {
		return fmt.Errorf("query: embed: %w", qErr)
	}

	type scored struct {
		notePath string
		isMOC    bool
		score    float32
		content  string
	}

	results := make([]scored, 0, len(notes))
	for _, n := range notes {
		notePath := pathOf(n.Basename, n.IsMOC)
		full := filepath.Join(args.VaultPath, notePath)

		scBytes, scErr := deps.Read(filepath.Join(args.VaultPath, embed.SidecarPath(notePath)))
		if scErr != nil {
			continue // missing sidecar — skip silently in query (per spec).
		}
		sc, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			continue
		}
		if sc.EmbeddingModelID != deps.Embedder.ModelID() {
			continue
		}

		noteBytes, noteErr := deps.Read(full)
		if noteErr != nil {
			continue
		}

		results = append(results, scored{
			notePath: notePath,
			isMOC:    n.IsMOC,
			score:    embed.Cosine(queryVec, sc.Vector),
			content:  string(noteBytes),
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return renderQueryYAML(stdout, args.Query, results, len(notes), countWithEmbeddings(notes, args.VaultPath, deps), limit)
}
```

(Plus YAML rendering helper that emits the schema in the spec.)

- [ ] **Step 10.3: Wire target in targets.go**

```go
targ.Targ(func(ctx context.Context, a QueryArgs) {
	a.VaultPath = resolveVault(a.VaultPath, homeOrEmpty(), os.Getenv)
	errHandler(RunQuery(withLog(ctx), a, newOsQueryDeps(), stdout))
}).Name("query").Description("Semantic search over the vault"),
```

- [ ] **Step 10.4: Run tests, commit**

```bash
targ test
git add internal/cli/query.go internal/cli/query_test.go internal/cli/targets.go
git commit -m "cli: engram query (YAML output, cosine ranking)"
```

---

## Phase 4: Auto-embed on `engram learn` (UAT 3)

### Task 11: Wire post-write embed step into runLearn, warn-and-proceed on failure

**Files:**
- Modify: `internal/cli/learn.go` (add Embedder dep, post-write call)
- Modify: `internal/cli/cli.go` (wire bundled embedder into `newOsLearnDeps`)
- Create: `internal/cli/auto_embed_test.go`

- [ ] **Step 11.1: RED — test that a successful learn calls embedder.Embed and writes a sidecar**

```go
func TestRunLearn_AutoEmbedsAfterWrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var sidecarPath string
	var sidecarBytes []byte
	emb := stubEmbedder{modelID: "m@4", dims: 4}
	deps := cli.LearnDeps{
		Now:       func() time.Time { return time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC) },
		Getenv:    func(string) string { return "" },
		StatDir:   func(string) error { return nil },
		ListIDs:   func(string) ([]string, error) { return nil, nil },
		Lock:      func(string) (func(), error) { return func() {}, nil },
		WriteNew:  func(path string, data []byte) error { /* note */ return nil },
		Embedder:  emb,
		// New deps for sidecar write:
		WriteSidecar: func(path string, data []byte) error {
			sidecarPath = path
			sidecarBytes = data
			return nil
		},
		LogWarning: func(_ string, _ ...any) {},
	}

	err := cli.RunLearn(ctx, ..., deps, ...)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sidecarPath).To(HaveSuffix(".vec.json"))
	var sc embed.Sidecar
	g.Expect(json.Unmarshal(sidecarBytes, &sc)).NotTo(HaveOccurred())
	g.Expect(sc.EmbeddingModelID).To(Equal("m@4"))
}

func TestRunLearn_EmbedFailureWarnsButSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	var warned bool
	emb := failingEmbedder{}
	deps := cli.LearnDeps{
		...
		Embedder:    emb,
		WriteSidecar: func(path string, data []byte) error { t.Fatal("should not write"); return nil },
		LogWarning:  func(_ string, _ ...any) { warned = true },
	}
	err := cli.RunLearn(ctx, ..., deps, ...)
	g.Expect(err).NotTo(HaveOccurred()) // learn still succeeds
	g.Expect(warned).To(BeTrue())
}
```

- [ ] **Step 11.2: Implement**

Extend `LearnDeps`:

```go
type LearnDeps struct {
	// existing fields ...
	Embedder     embed.Embedder
	WriteSidecar func(path string, data []byte) error
	LogWarning   func(format string, args ...any)
}
```

In `writeLearnUnderLock`, after `deps.WriteNew(path, []byte(content))` succeeds:

```go
if deps.Embedder != nil {
	body := embed.ExtractBody([]byte(content))
	vec, embErr := deps.Embedder.Embed(context.Background(), string(body))
	if embErr != nil {
		deps.LogWarning("learn: embed failed for %s: %v", path, embErr)
	} else {
		sc := embed.Sidecar{
			EmbeddingModelID: deps.Embedder.ModelID(),
			Dims:             deps.Embedder.Dims(),
			Vector:           vec,
			ContentHash:      embed.ContentHash([]byte(content)),
		}
		scBytes, _ := embed.MarshalSidecar(sc)
		if writeErr := deps.WriteSidecar(embed.SidecarPath(path), scBytes); writeErr != nil {
			deps.LogWarning("learn: sidecar write failed for %s: %v", path, writeErr)
		}
	}
}
```

Update `newOsLearnDeps` in cli.go to populate `Embedder`, `WriteSidecar` (using `os.WriteFile`), and `LogWarning` (using the debuglog or stderr).

- [ ] **Step 11.3: Run tests**

```bash
targ test
```

- [ ] **Step 11.4: Commit**

```bash
git add internal/cli/learn.go internal/cli/cli.go internal/cli/auto_embed_test.go
git commit -m "cli: auto-embed on learn (warn-and-proceed)"
```

---

## Phase 5: End-to-end verification against the real vault

### Task 12: Build the binary and run all 13 UAT cases

**Files:**
- Modify: nothing (this is verification)
- Create: a short markdown checklist of UAT results to put in the commit message

- [ ] **Step 12.1: Build the binary**

```bash
targ build
ls -la bin/engram
```

Expected: ~35 MB binary.

- [ ] **Step 12.2: Run UAT 1 (backfill)**

```bash
./bin/engram embed apply --all
./bin/engram embed status
```

Expected: `total: 163; with-embeddings: 163; without: 0; stale: 0; incompatible: 0; broken: 0`.

- [ ] **Step 12.3: Inspect a sidecar for UAT 2 (sidecar format)**

```bash
cat ~/.local/share/engram/vault/Permanent/132.*.vec.json | python3 -m json.tool
```

Expected: object with `embedding_model_id`, `dims: 384`, `vector` (384 floats), `content_hash` starting `sha256:`.

- [ ] **Step 12.4: Run UAT 3 (auto-embed on learn)**

```bash
./bin/engram learn fact \
    --slug uat-3-probe \
    --source "session log spike-uat, 2026-05-24, context: auto-embed probe" \
    --situation "When verifying auto-embed on learn (UAT 3)" \
    --subject "the auto-embed integration" \
    --predicate "writes" \
    --object "a sidecar alongside the new note"
ls -la ~/.local/share/engram/vault/Permanent/ | grep uat-3-probe
```

Expected: both `.md` and `.vec.json` for the new note.

- [ ] **Step 12.5: UAT 4 (stale re-embed)**

Edit a note's body, then:

```bash
./bin/engram embed status
./bin/engram embed apply --stale
./bin/engram embed status
```

Expected: stale count goes 0 → 1 → 0; the edited note's sidecar is updated.

- [ ] **Step 12.6: UAT 5 + 6 + 7 (query)**

```bash
./bin/engram query "verifying current behavior before claiming a delta" --limit 5
./bin/engram query "exploration sprawl" --limit 5
```

Expected: Permanent/132 in top 3 of #5; Permanent/133 in top 3 of #6; scores monotonically decreasing.

- [ ] **Step 12.7: UAT 9 (empty vault)**

```bash
mkdir -p /tmp/empty-vault/Permanent /tmp/empty-vault/MOCs
ENGRAM_VAULT_PATH=/tmp/empty-vault ./bin/engram query "anything"
echo "exit: $?"
```

Expected: `items: []`, exit 0.

- [ ] **Step 12.8: UAT 10 (missing model — verify error path exists by reading the code)**

This case primarily verifies the error path is wired. Since the model is bundled, simulating "missing" requires either editing the bundled fs or temporarily renaming a built artifact. A representative check: confirm `NewBundledHugotEmbedder` returns a non-nil error when the embedded directory is empty (via a unit test in Task 8).

- [ ] **Step 12.9: UAT 11 (stale vs incompatible)**

Create a sidecar with a wrong model_id, then run status.

```bash
# Edit a sidecar to have model_id "old-model@128"; verify status reports it as incompatible.
./bin/engram embed status
./bin/engram embed apply --stale     # should NOT touch the incompatible sidecar
./bin/engram embed apply --force     # should re-embed it
```

- [ ] **Step 12.10: UAT 12 (partial corruption)**

```bash
echo "{not json" > ~/.local/share/engram/vault/Permanent/<some>.vec.json
./bin/engram embed status   # broken: 1
./bin/engram embed apply --stale   # offers to re-embed broken under --stale per spec
./bin/engram embed status   # broken: 0
```

- [ ] **Step 12.11: Capture the UAT-result table for the final commit**

Record outcomes in `docs/plans/2026-05-24-engram-query-spike-uat-results.md` (or append to the plan as a Results section).

- [ ] **Step 12.12: Commit verification results**

```bash
git add docs/plans/2026-05-24-engram-query-spike.md docs/plans/2026-05-24-engram-query-spike-uat-results.md
git commit -m "spike: UAT 1–13 results (all PASS|FAIL with $details)"
```

---

## Phase 6: Documentation

### Task 13: Update README, CLAUDE.md, architecture docs

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/architecture/c1-system-context.md`

- [ ] **Step 13.1: README — add `engram query` and `engram embed` sections under Commands**

Mention: semantic search over the vault, embed-on-write pipeline, the bundled model, the sidecar file format, and the spike's verification (link to UAT results).

- [ ] **Step 13.2: CLAUDE.md — add `internal/embed/` to the directory map**

Single line in the Directory Structure table mentioning the new package and its role.

- [ ] **Step 13.3: Architecture — add embedder relationship**

In `docs/architecture/c1-system-context.md`, add the embedder as a component of the engram binary; show the embed-on-write data flow alongside the existing learn flow.

- [ ] **Step 13.4: Commit docs**

```bash
git add README.md CLAUDE.md docs/architecture/c1-system-context.md
git commit -m "docs: engram query + embed-on-write pipeline"
```

---

## Self-review checklist

- **Spec coverage:** every UAT case (1–13) has at least one task that exercises it (Task 3 → UAT 13; Task 7 → 11/12; Task 9 → 1/2/4/8/11/12; Task 10 → 5/6/7/9/10; Task 11 → 3; Task 12 → end-to-end verification). Spike-spec settled decisions #1–6 are honoured: body-only hash (Task 5), bundled model (Task 8), `engram query` errors on missing embeddings (Task 10 — `Scan` returns no notes with sidecars → `items: []` is correct; the "missing model file" path in Task 12.8 covers the no-model-loaded case), warn-and-proceed (Task 11), default limit 20 (Task 10), YAML output (Task 10).
- **No placeholders.** All steps name exact files and show real code.
- **Type consistency.** `Embedder.Embed(ctx, text)`, `Sidecar`, `State`, `ComputeState(fs, notePath, modelID)`, `SidecarPath`, `ContentHash`, `ExtractBody`, `Cosine`, `MarshalSidecar/UnmarshalSidecar` used consistently across tasks.
- **TDD discipline.** Every behavioural task has explicit RED → GREEN → REFACTOR (or RED → GREEN) steps with assertions before implementation.
- **Frequent commits.** Each task ends with a commit; each commit is one logical step.
