# Lazy Compositional L2 Synthesis — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **Phase 3 edits a SKILL.md and is GATED on superpowers:writing-skills (RED→GREEN→pressure) — no exceptions.**

**Goal:** Defer L2 fact/feedback creation from eager (every convention at `/learn`) to lazy (crystallized on demand at `/recall`, only when matching L1/L2 evidence has no covering L2), and prove the cost/quality trade against the eager baseline.

**Architecture:** Four dependency-linked chunks, each with a gate. (1) **Dual-vector sidecars** — every `.vec.json` carries a situation vector *and* a body vector (prerequisite; touches the whole vault). (2) **Binary `--synthesize-l2` mode** — `engram query` returns the matched L1+L2 neighborhood, clusters it with per-note matched-vector coordinates, and emits raw `nearest_l2 {path, cosine}` per cluster (no band decision). (3) **`/recall` three-band writes** — the skill consumes `nearest_l2.cosine` and blocks: ≥0.95 no-op · 0.80–0.95 update · <0.80 create, with recency-bias on divergence. (4) **Eager-vs-lazy experiment** — a new `l2.lazy` regime in the cumulative-accumulation harness, with arm-B recall persisting forward across the app chain.

**Tech Stack:** Go (pure, no CGO; MiniLM-L6-v2@384 embeddings via GoMLX `simplego`), `targ` build system, imptest + rapid + gomega tests, Python eval harness (`dev/eval/cumulative/`), superpowers:writing-skills for the SKILL.md edit.

**Source spec:** `docs/superpowers/specs/2026-06-09-lazy-l2-synthesis-design.md` (amended 2026-06-10: persist-forward arm B; `ContentHash` covers situation+body; schema-version check precedes vector-length validation).

---

## Build discipline (read before starting)

- **All build/test/check go through `targ`** — NEVER `go test`/`go vet`/`go build` directly. Tests: `targ test`. Lint+coverage: `targ check-full` (gets ALL errors at once; `targ check` stops early). Build: `targ build`. Treat `targ` as a black box.
- `targ test` runs the **whole suite**. During a struct-shape migration (Phase 1) the `cli` package will not compile until every call site is updated — so "RED" for a new behavior may surface as a *compile error referencing an undefined symbol* until you implement it, and the suite only goes fully green at the **Phase 1 checkpoint (Task 1.13)**. This is expected and called out where it happens.
- **nilaway + gomega:** after `g.Expect(err).NotTo(HaveOccurred())`, add `if err != nil { return }` before touching values. Use `MatchError(...)` not `err.Error()`. Nil-guard pointers before field access.
- **Tests:** blackbox `package *_test`; `t.Parallel()` on every test and subtest with no shared mutable state; named constants not magic numbers; wrapped errors `fmt.Errorf("ctx: %w", err)`; sentinel errors as package vars; lines < 120 chars.
- **Commit** after each task with the `AI-Used: [claude]` trailer. Work on branch `lazy-l2-synthesis` (already checked out). ff-only merges; rebase on main before merge.

---

## Shared contracts (locked vocabulary — every phase references these exact names)

**`internal/embed` package:**

```go
// embedder.go
const SidecarSchemaVersion = 1               // bump invalidates all sidecars (forces --force re-embed)

var ErrSchemaVersion = errors.New("sidecar schema version unsupported")  // joins ErrDimsMismatch, ErrSidecarMalformed

type Sidecar struct {                        // NEW shape (replaces single Vector)
	SchemaVersion    int       `json:"schema_version"`
	EmbeddingModelID string    `json:"embedding_model_id"`
	Dims             int       `json:"dims"`
	SituationVector  []float32 `json:"situation_vector"`
	BodyVector       []float32 `json:"body_vector"`
	ContentHash      string    `json:"content_hash"`
}

func BuildSidecar(ctx context.Context, e Embedder, raw []byte) (Sidecar, error)  // embeds both, sets schema/hash

// hash.go
func SituationText(raw []byte) []byte        // the `situation:` frontmatter field, for ALL note types ("" if absent)
func BodyText(raw []byte) []byte             // == ExtractBody(raw)
func ContentHash(raw []byte) string          // sha256(SituationText ‖ 0x00 ‖ BodyText)
// embed.Text() is REMOVED (its 4 callers move to BuildSidecar)
```

**`internal/cli/query.go`:**

```go
func bestVector(queryVec []float32, sc embed.Sidecar) (score float32, coord []float32)  // max(sit,body); coord = winner

type tierIndex struct {                      // replaces l3Index; reused for L2
	paths []string
	sit   [][]float32
	body  [][]float32
}
func gatherTierIndex(hits []compatibleSidecar, vault string, read func(string) ([]byte, error), tier string) tierIndex
func nearestInTierIndex(centroid []float32, idx tierIndex) (path string, cosine float32, found bool)  // max(sit,body)

// scoredCandidate and subgraphMember gain/keep a winning-coordinate vector (see Task 1.9)

// Phase 2 additions:
type queryNearestL2 struct { Path string `yaml:"path"`; Cosine float32 `yaml:"cosine"` }
// QueryArgs.SynthesizeL2 bool  (--synthesize-l2)
// queryCluster.NearestL2 *queryNearestL2 `yaml:"nearest_l2,omitempty"`
// aggregatedSummary.l2 tierIndex
func runSynthesizeL2Query(...) error                  // mirrors runSynthesisQuery, L1+L2-constrained
func nearestL2ForTier(centroid []float32, idx tierIndex, tiers []string) *queryNearestL2
func filterHitsToTiers(hits []compatibleSidecar, vault string, read func(string) ([]byte, error), tiers []string) []compatibleSidecar
```

**`internal/cli/check.go`:** `CheckDeps` gains `ReadSidecar func(path string) ([]byte, error)`; new `checkSidecars(...)` invariant (code `S1`).

**Eval harness (`dev/eval/cumulative/`):** new regime key `"l2.lazy"` (`{"write":"L1","read_mode":"synthesize_l2","read_tiers":[]}`); `build_prompt` `synthesize_l2` branch; `run_learn` persist-forward for `l2.lazy`; `run_build` post-op vault metrics; `aggregate.py` `WRITE_TIER`/`write_of` aliases.

---

## File structure map

| File | Responsibility | Phase |
|------|----------------|-------|
| `internal/embed/embedder.go` | `Sidecar` struct, `Embedder`, schema const, `ErrSchemaVersion`, `BuildSidecar` | 1 |
| `internal/embed/hash.go` | `SituationText`/`BodyText`/`ContentHash`; remove `Text` | 1 |
| `internal/embed/sidecar.go` | `UnmarshalSidecar` (schema-first), `MarshalSidecar` | 1 |
| `internal/embed/state.go` | `ComputeState` routes `ErrSchemaVersion`→`StateIncompatible` | 1 |
| `internal/cli/learn.go` | `autoEmbedNote` → `BuildSidecar` | 1 |
| `internal/cli/embed.go` | `applyOne` → `BuildSidecar` | 1 |
| `internal/cli/resituate.go` | `writeResituatedSidecar` → `BuildSidecar` | 1 |
| `internal/cli/migrate_episodes.go` | `migrateEpisodeSidecar` → `BuildSidecar` | 1 |
| `internal/cli/query.go` | dual-vector scoring/coordinate; `tierIndex`; `--synthesize-l2` mode + `nearest_l2` | 1, 2 |
| `internal/cli/check.go` | `checkSidecars` invariant + `ReadSidecar` dep | 1 |
| `internal/cli/*_test.go` | fixtures switch to dual-vector sidecars | 1, 2 |
| `skills/recall/SKILL.md` | three-band blocking L2 writes | 3 |
| `skills/recall/tests/baseline-three-band-writes.md` + results | writing-skills TDD evidence | 3 |
| `dev/eval/cumulative/harness.py` | `l2.lazy` regime, `synthesize_l2` recall prompt, persist-forward learn, vault metrics | 4 |
| `dev/eval/cumulative/aggregate.py` | `l2.lazy` aliases; arm A/B comparison | 4 |

---

# Phase 1 — Dual-vector sidecars (prerequisite)

**Gate (Task 1.13):** a re-embedded vault passes `engram check` (now with the `S1` sidecar invariant) and `engram query` retrieval is at least as good as the old single-vector ranking. The repo is not fully green until 1.13.

### Task 1.1: `Sidecar` struct + schema version + `ErrSchemaVersion` + codec

**Files:**
- Modify: `internal/embed/embedder.go:39-64` (errors, struct, add const + `ErrSchemaVersion`)
- Modify: `internal/embed/sidecar.go:35-53` (`UnmarshalSidecar`)
- Test: `internal/embed/sidecar_test.go`

- [ ] **Step 1: Write the failing tests** (append to `internal/embed/sidecar_test.go`)

```go
func TestMarshalUnmarshal_DualVector_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	original := embed.Sidecar{
		SchemaVersion:    embed.SidecarSchemaVersion,
		EmbeddingModelID: "minilm-l6-v2@384",
		Dims:             3,
		SituationVector:  []float32{0.1, 0.2, 0.3},
		BodyVector:       []float32{0.4, 0.5, 0.6},
		ContentHash:      "sha256:deadbeef",
	}

	out, parseErr := embed.UnmarshalSidecar(embed.MarshalSidecar(original))
	g.Expect(parseErr).NotTo(HaveOccurred())
	if parseErr != nil {
		return
	}
	g.Expect(out).To(Equal(original))
}

func TestUnmarshalSidecar_OldSingleVector_IsSchemaError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// An old-format sidecar: no schema_version, single "vector" key.
	old := []byte(`{"embedding_model_id":"minilm-l6-v2@384","dims":3,"vector":[0.1,0.2,0.3],"content_hash":"sha256:x"}`)

	_, parseErr := embed.UnmarshalSidecar(old)
	g.Expect(parseErr).To(MatchError(embed.ErrSchemaVersion))
}

func TestUnmarshalSidecar_DimsMismatch_OnEitherVector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bad := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@3", Dims: 3,
		SituationVector: []float32{0.1, 0.2, 0.3}, BodyVector: []float32{0.4, 0.5}, // body short
		ContentHash: "sha256:x",
	}

	_, parseErr := embed.UnmarshalSidecar(embed.MarshalSidecar(bad))
	g.Expect(parseErr).To(MatchError(embed.ErrDimsMismatch))
}
```

- [ ] **Step 2: Run to verify RED.** `targ test` — fails to compile (`SchemaVersion`/`SituationVector`/`BodyVector`/`ErrSchemaVersion` undefined).

- [ ] **Step 3: Implement.** In `internal/embed/embedder.go`, add to the `var (...)` block (after `ErrSidecarMalformed`):

```go
	ErrSchemaVersion = errors.New("sidecar schema version unsupported")
```

Add a const near the top of the file:

```go
// SidecarSchemaVersion is the on-disk sidecar format version. Bumping it
// invalidates all existing sidecars (they classify Incompatible) so an
// `engram embed apply --force` re-embeds the corpus. Version 1 introduced
// the dual situation/body vectors.
const SidecarSchemaVersion = 1
```

Replace the `Sidecar` struct (keep the `//nolint:tagliatelle` comment and the doc comment, update the doc to say "two vectors"):

```go
type Sidecar struct {
	SchemaVersion    int       `json:"schema_version"`
	EmbeddingModelID string    `json:"embedding_model_id"`
	Dims             int       `json:"dims"`
	SituationVector  []float32 `json:"situation_vector"`
	BodyVector       []float32 `json:"body_vector"`
	ContentHash      string    `json:"content_hash"`
}
```

In `internal/embed/sidecar.go`, replace `UnmarshalSidecar`'s body validation (schema check FIRST, then both vectors):

```go
func UnmarshalSidecar(data []byte) (Sidecar, error) {
	var sidecar Sidecar
	err := json.Unmarshal(data, &sidecar)
	if err != nil {
		return Sidecar{}, fmt.Errorf("%w: %w", ErrSidecarMalformed, err)
	}
	if sidecar.SchemaVersion != SidecarSchemaVersion {
		return Sidecar{}, fmt.Errorf("%w: got=%d want=%d", ErrSchemaVersion, sidecar.SchemaVersion, SidecarSchemaVersion)
	}
	if len(sidecar.SituationVector) != sidecar.Dims || len(sidecar.BodyVector) != sidecar.Dims {
		return Sidecar{}, fmt.Errorf("%w: dims=%d situation=%d body=%d",
			ErrDimsMismatch, sidecar.Dims, len(sidecar.SituationVector), len(sidecar.BodyVector))
	}
	return sidecar, nil
}
```

- [ ] **Step 4:** `targ test` — the three new tests pass (the broader suite still won't compile until later tasks; confirm these specific tests + the `embed` package build). Commit: `git commit -am "feat(embed): dual-vector sidecar schema + schema_version (v1)"`

### Task 1.2: `SituationText` / `BodyText` / `ContentHash(sit+body)`; remove `Text`

**Files:**
- Modify: `internal/embed/hash.go:14-74`
- Test: `internal/embed/hash_test.go`

- [ ] **Step 1: Write the failing tests** (replace the existing `embed.Text` tests in `hash_test.go`):

```go
func TestSituationText_ExtractsFieldForAnyType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	fact := []byte("---\ntype: fact\nsituation: when wiring a Go CLI\n---\n\nbody here\n")
	g.Expect(string(embed.SituationText(fact))).To(Equal("when wiring a Go CLI"))

	noFM := []byte("just a body, no frontmatter\n")
	g.Expect(embed.SituationText(noFM)).To(BeEmpty())
}

func TestBodyText_StripsFrontmatter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := []byte("---\ntype: fact\nsituation: x\n---\n\nthe body\n")
	g.Expect(string(embed.BodyText(raw))).To(Equal("the body\n"))
}

func TestContentHash_ChangesWhenEitherSourceChanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	base := []byte("---\ntype: fact\nsituation: A\n---\n\nbody B\n")
	sitChanged := []byte("---\ntype: fact\nsituation: A2\n---\n\nbody B\n")
	bodyChanged := []byte("---\ntype: fact\nsituation: A\n---\n\nbody B2\n")

	g.Expect(embed.ContentHash(base)).NotTo(Equal(embed.ContentHash(sitChanged)))
	g.Expect(embed.ContentHash(base)).NotTo(Equal(embed.ContentHash(bodyChanged)))
}
```

- [ ] **Step 2: Run to verify RED.** `targ test` — undefined `embed.SituationText`/`embed.BodyText`; old `Text` test gone.

- [ ] **Step 3: Implement.** In `internal/embed/hash.go`, replace `func Text(...)` with `SituationText`, add `BodyText`, and rewrite `ContentHash`:

```go
// SituationText returns the `situation:` frontmatter field for any note
// type ("" when absent or unparseable). It is the situation-vector source.
func SituationText(raw []byte) []byte {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(raw, delim) {
		return nil
	}
	rest := raw[len(delim):]
	frontmatter, _, ok := bytes.Cut(rest, delim)
	if !ok {
		return nil
	}
	situation := extractFrontmatterField(frontmatter, "situation")
	if situation == "" {
		return nil
	}
	return []byte(situation)
}

// BodyText returns the note body (frontmatter stripped). It is the
// body-vector source for every note type.
func BodyText(raw []byte) []byte {
	return ExtractBody(raw)
}
```

Replace `ContentHash` to cover both sources:

```go
func ContentHash(raw []byte) string {
	hasher := sha256.New()
	hasher.Write(SituationText(raw))
	hasher.Write([]byte{0})
	hasher.Write(BodyText(raw))
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
}
```

- [ ] **Step 4:** `targ test` — the three tests pass (suite still mid-migration). Commit: `git commit -am "feat(embed): split SituationText/BodyText; ContentHash covers both"`

### Task 1.3: `ComputeState` routes `ErrSchemaVersion` → `StateIncompatible`

**Files:**
- Modify: `internal/embed/state.go:25-54`
- Test: `internal/embed/state_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestComputeState_OldSchemaSidecar_IsIncompatible(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	note := []byte("---\ntype: fact\nsituation: x\n---\n\nbody\n")
	oldSidecar := []byte(`{"embedding_model_id":"minilm-l6-v2@384","dims":3,"vector":[0.1,0.2,0.3],"content_hash":"sha256:x"}`)
	fsys := fakeFS{"n.md": note, "n.vec.json": oldSidecar} // adapt to this file's existing fake-FS helper

	g.Expect(embed.ComputeState(fsys, "n.md", "minilm-l6-v2@384")).To(Equal(embed.StateIncompatible))
}
```

(Use whatever fake-FS constructor `state_test.go` already uses — match the existing tests' setup verbatim; `SidecarPath("n.md")` is `"n.vec.json"`.)

- [ ] **Step 2: Run to verify RED.** `targ test` — old-schema sidecar currently returns `StateBroken`.

- [ ] **Step 3: Implement.** In `internal/embed/state.go`, change the parse-error branch:

```go
	sidecar, parseErr := UnmarshalSidecar(scBytes)
	if parseErr != nil {
		if errors.Is(parseErr, ErrSchemaVersion) {
			return StateIncompatible
		}
		return StateBroken
	}
```

(`state.go` already imports `errors`.)

- [ ] **Step 4:** `targ test` — passes. Commit: `git commit -am "feat(embed): old-schema sidecars classify Incompatible, not Broken"`

### Task 1.4: `embed.BuildSidecar` helper (dual embed, situation fallback)

**Files:**
- Modify: `internal/embed/embedder.go` (add `BuildSidecar`; add `context`/`fmt` imports as needed)
- Test: `internal/embed/build_test.go` (new)

- [ ] **Step 1: Write the failing test** (new file `internal/embed/build_test.go`):

```go
package embed_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// seqEmbedder returns a distinct vector per call so we can tell situation
// from body. Call 1 -> {1,0,0}; call 2 -> {0,1,0}.
type seqEmbedder struct{ n int }

func (e *seqEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	e.n++
	if e.n == 1 {
		return []float32{1, 0, 0}, nil
	}
	return []float32{0, 1, 0}, nil
}
func (e *seqEmbedder) ModelID() string { return "m@3" }
func (e *seqEmbedder) Dims() int       { return 3 }

func TestBuildSidecar_EmbedsBothAndStamps(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := []byte("---\ntype: fact\nsituation: when X\n---\n\nbody Y\n")
	sc, err := embed.BuildSidecar(context.Background(), &seqEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(sc.SchemaVersion).To(Equal(embed.SidecarSchemaVersion))
	g.Expect(sc.SituationVector).To(Equal([]float32{1, 0, 0}))
	g.Expect(sc.BodyVector).To(Equal([]float32{0, 1, 0}))
	g.Expect(sc.ContentHash).To(Equal(embed.ContentHash(raw)))
}

func TestBuildSidecar_NoSituation_FallsBackToBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	raw := []byte("no frontmatter, body only\n") // SituationText == ""
	sc, err := embed.BuildSidecar(context.Background(), &seqEmbedder{}, raw)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	// Both embed calls receive BodyText, but seqEmbedder still returns
	// distinct vectors; the point is that situation embedding used a
	// non-empty input. Assert situation vector is the FIRST call's output.
	g.Expect(sc.SituationVector).To(Equal([]float32{1, 0, 0}))
}
```

- [ ] **Step 2: Run to verify RED.** `targ test` — `embed.BuildSidecar` undefined.

- [ ] **Step 3: Implement.** Add to `internal/embed/embedder.go` (add `"context"` and `"fmt"` to imports):

```go
// BuildSidecar embeds a note's situation and body and returns a fully
// stamped dual-vector sidecar. When the note has no situation field, the
// body text stands in for the situation embedding so every note still
// carries a meaningful situation vector. Either embed failure is returned
// to the caller, which applies its own warn-or-fail policy.
func BuildSidecar(ctx context.Context, e Embedder, raw []byte) (Sidecar, error) {
	situationInput := SituationText(raw)
	if len(situationInput) == 0 {
		situationInput = BodyText(raw)
	}

	situationVector, err := e.Embed(ctx, string(situationInput))
	if err != nil {
		return Sidecar{}, fmt.Errorf("embed: situation vector: %w", err)
	}

	bodyVector, err := e.Embed(ctx, string(BodyText(raw)))
	if err != nil {
		return Sidecar{}, fmt.Errorf("embed: body vector: %w", err)
	}

	return Sidecar{
		SchemaVersion:    SidecarSchemaVersion,
		EmbeddingModelID: e.ModelID(),
		Dims:             e.Dims(),
		SituationVector:  situationVector,
		BodyVector:       bodyVector,
		ContentHash:      ContentHash(raw),
	}, nil
}
```

- [ ] **Step 4:** `targ test` — passes. Commit: `git commit -am "feat(embed): BuildSidecar dual-embed helper with situation fallback"`

### Task 1.5: rewire `autoEmbedNote` (learn path)

**Files:** Modify `internal/cli/learn.go:374-403`; Test `internal/cli/auto_embed_test.go:45-77`

- [ ] **Step 1: Update the test** (`auto_embed_test.go`) — assert both vectors + schema version. Replace the `parsed.Vector` assertion (~line 71-75) with:

```go
	g.Expect(parsed.SchemaVersion).To(Equal(embed.SidecarSchemaVersion))
	g.Expect(parsed.SituationVector).To(HaveLen(4))
	g.Expect(parsed.BodyVector).To(HaveLen(4))
	g.Expect(parsed.ContentHash).To(HavePrefix("sha256:"))
```

(The `successEmbedder` fake returns the same vector each call, so both fields equal — fine for a structural assertion. If the fake panics on a second call, update it to be idempotent.)

- [ ] **Step 2: Run to verify RED.** `targ test` — `autoEmbedNote` still builds the old struct (`Vector` undefined → compile error).

- [ ] **Step 3: Implement.** Replace the body of `autoEmbedNote` between the nil-guard and the sidecar write:

```go
func autoEmbedNote(ctx context.Context, deps LearnDeps, notePath, content string) {
	if deps.Embedder == nil || deps.WriteSidecar == nil {
		return
	}

	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("learn: embed failed for %s: %v", notePath, embErr)
		}
		return
	}

	writeErr := deps.WriteSidecar(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("learn: sidecar write failed for %s: %v", notePath, writeErr)
	}
}
```

- [ ] **Step 4:** `targ test` — the auto-embed test passes. Commit: `git commit -am "feat(cli): autoEmbedNote writes dual-vector sidecar"`

### Task 1.6: rewire `applyOne` (`engram embed apply`)

**Files:** Modify `internal/cli/embed.go:192-237`; Test `internal/cli/embed_test.go:41-112`

- [ ] **Step 1: Update the test** — replace the `sidecar.Vector` assertion (~line 62-67) with `SituationVector`/`BodyVector`/`SchemaVersion` checks (as in Task 1.5).
- [ ] **Step 2: RED.** `targ test`.
- [ ] **Step 3: Implement.** In `applyOne`, replace the `embed.Text(...)` + `embed.Sidecar{Vector:...}` block with:

```go
	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, noteBytes)
	if embErr != nil {
		// preserve applyOne's existing per-note error handling/return shape here
		return embErr // adapt to the surrounding function's actual signature/accumulator
	}
	scBytes := embed.MarshalSidecar(sidecar)
	sidecarFull := filepath.Join(vault, embed.SidecarPath(notePath))
	writeErr := deps.Write(sidecarFull, scBytes)
```

(Read `applyOne`'s exact surrounding error-handling and match it; `modelID`/`dims` locals become unused — remove them.)

- [ ] **Step 4:** `targ test` passes. Commit: `git commit -am "feat(cli): embed apply writes dual-vector sidecar"`

### Task 1.7: rewire `writeResituatedSidecar`

**Files:** Modify `internal/cli/resituate.go:305-324`; Test `internal/cli/resituate_test.go:454-496`

- [ ] **Step 1: Update the test** — sidecar assertions to dual-vector; `readSidecarHash` helper still works (`ContentHash` field unchanged in name).
- [ ] **Step 2: RED.**
- [ ] **Step 3: Implement** (failures still surface, not warn-and-ignore):

```go
func writeResituatedSidecar(ctx context.Context, deps ResituateDeps, notePath, content string) error {
	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		return fmt.Errorf("resituate: embedding %s: %w", notePath, embErr)
	}
	writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("resituate: writing sidecar for %s: %w", notePath, writeErr)
	}
	return nil
}
```

- [ ] **Step 4:** `targ test` passes. Commit: `git commit -am "feat(cli): resituate writes dual-vector sidecar"`

### Task 1.8: rewire `migrateEpisodeSidecar`

**Files:** Modify `internal/cli/migrate_episodes.go:162-185`; Test exists in the migrate-episodes test file.

- [ ] **Step 1: Update the test** for dual-vector shape (keep the nil-Embedder guard test — nil embedder ⇒ no sidecar written, unchanged).
- [ ] **Step 2: RED.**
- [ ] **Step 3: Implement** (preserve the `deps.Embedder == nil` early return):

```go
func migrateEpisodeSidecar(ctx context.Context, deps MigrateEpisodesDeps, notePath, content string) error {
	if deps.Embedder == nil {
		return nil
	}
	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		return fmt.Errorf("migrate-episodes: embedding %s: %w", notePath, embErr)
	}
	writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("migrate-episodes: writing sidecar for %s: %w", notePath, writeErr)
	}
	return nil
}
```

- [ ] **Step 4:** `targ test` passes. Commit: `git commit -am "feat(cli): migrate-episodes writes dual-vector sidecar"`

### Task 1.9: query.go — dual-vector scoring + winning-vector coordinate + `tierIndex`

This is the retrieval refactor. `max(situation, body)` becomes the score; the **winning** vector becomes the clustering coordinate; the L3 nearest-index carries both vectors and gates by `max`.

**Files:** Modify `internal/cli/query.go` (sites: 198-203 `l3Index`, 456-491 `buildSubgraphMembers`, 497-513 `buildUnionSubgraph`, 741-767 `gatherL3Index`, 1231-1244 `nearestL3For`, 1359-1393 `rankCandidates` + `scoredCandidate`).

- [ ] **Step 1: Write the failing test** (`internal/cli/query_test.go`) — distinct situation/body vectors, query matches via the situation axis:

```go
func TestQuery_ScoresByMaxOfSituationAndBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	// Note whose BODY vector is orthogonal to the query but whose SITUATION
	// vector matches it — must still surface.
	plantDualVector(t, memFS, vault, "Permanent/1.fact.md",
		"---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nbody\n",
		/*sit*/ []float32{1, 0, 0, 0}, /*body*/ []float32{0, 0, 0, 1})

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, Limit: 20}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.Items).NotTo(BeEmpty(), "situation-axis match must surface even when body is orthogonal")
}
```

Add the `plantDualVector` helper near `plantWithFixedVector` in the test support file:

```go
func plantDualVector(t *testing.T, fs *inMemoryFS, vault, rel, content string, sit, body []float32) {
	t.Helper()
	writeFile(t, fs, filepath.Join(vault, rel), content) // match existing helper's write call
	sc := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@4", Dims: len(sit),
		SituationVector: sit, BodyVector: body, ContentHash: embed.ContentHash([]byte(content)),
	}
	scRel := rel[:len(rel)-len(".md")] + ".vec.json"
	writeFile(t, fs, filepath.Join(vault, scRel), string(embed.MarshalSidecar(sc)))
}
```

- [ ] **Step 2: RED.** `targ test` — `embed.Sidecar{...Vector...}` references throughout query.go fail to compile.

- [ ] **Step 3: Implement.** (a) Add the `bestVector` helper:

```go
// bestVector scores a sidecar against the query by the stronger of its two
// axes and returns that score with the WINNING vector — the coordinate the
// note is positioned by for clustering (per the lazy-L2 design: a note is
// clustered by the vector that matched it).
func bestVector(queryVec []float32, sc embed.Sidecar) (float32, []float32) {
	situationScore := embed.Cosine(queryVec, sc.SituationVector)
	bodyScore := embed.Cosine(queryVec, sc.BodyVector)
	if situationScore >= bodyScore {
		return situationScore, sc.SituationVector
	}
	return bodyScore, sc.BodyVector
}
```

(b) Add `coord []float32` to the `scoredCandidate` struct, and set both score and coord in `rankCandidates`:

```go
		score, coord := bestVector(queryVec, hit.sidecar)
		candidates = append(candidates, scoredCandidate{
			notePath: notePath,
			basename: hit.note.Basename,
			score:    score,
			coord:    coord,
			content:  stripWikilinks(string(noteBytes)),
		})
```

(`unionDirectHits` already keeps whole `scoredCandidate`s by max score — `coord` rides along automatically.)

(c) In `buildSubgraphMembers` (line ~474-475) use `bestVector`:

```go
		score, coord := bestVector(queryVec, hit.sidecar)
		member := subgraphMember{
			basename: name,
			notePath: notePath,
			vector:   coord,
			score:    score,
		}
```

(d) In `buildUnionSubgraph` (line ~506) take the coordinate from the scored hit instead of re-reading `sidecar.Vector`:

```go
		members = append(members, subgraphMember{
			basename: hit.basename,
			notePath: hit.notePath,
			vector:   hit.coord,
			score:    hit.score,
			content:  hit.content,
		})
```

(e) Replace `l3Index` with the general `tierIndex` and add the gather/nearest helpers:

```go
// tierIndex holds the vault-wide set of one tier's note paths and BOTH
// sidecar vectors for per-cluster nearest-tier lookup by max(situation,body).
type tierIndex struct {
	paths []string
	sit   [][]float32
	body  [][]float32
}

func gatherTierIndex(hits []compatibleSidecar, vault string, read func(string) ([]byte, error), tier string) tierIndex {
	idx := tierIndex{}
	for _, hit := range hits {
		notePath := pathOf(hit.note.Basename, hit.note.IsMOC)
		body, readErr := read(filepath.Join(vault, notePath))
		if readErr != nil {
			continue
		}
		item := resolvedItem{content: stripWikilinks(string(body))}
		if !itemMatchesTier(item, []string{tier}) {
			continue
		}
		idx.paths = append(idx.paths, notePath)
		idx.sit = append(idx.sit, hit.sidecar.SituationVector)
		idx.body = append(idx.body, hit.sidecar.BodyVector)
	}
	return idx
}

// nearestInTierIndex returns the index note nearest the centroid by the
// stronger of its two axes (the "either axis" gate). found is false for an
// empty index.
func nearestInTierIndex(centroid []float32, idx tierIndex) (string, float32, bool) {
	best, bestSim := -1, float32(-1)
	for i := range idx.paths {
		sim := embed.Cosine(centroid, idx.sit[i])
		if b := embed.Cosine(centroid, idx.body[i]); b > sim {
			sim = b
		}
		if sim > bestSim {
			bestSim = sim
			best = i
		}
	}
	if best < 0 {
		return "", 0, false
	}
	return idx.paths[best], bestSim, true
}
```

Rewrite `gatherL3Index` as a thin wrapper and `nearestL3For` to use `nearestInTierIndex`:

```go
func gatherL3Index(hits []compatibleSidecar, vault string, read func(string) ([]byte, error)) tierIndex {
	return gatherTierIndex(hits, vault, read, tierL3)
}

func nearestL3For(centroid []float32, l3Notes tierIndex) *queryNearestL3 {
	path, cosine, found := nearestInTierIndex(centroid, l3Notes)
	if !found {
		return nil
	}
	return &queryNearestL3{Path: path, Cosine: cosine}
}
```

Change `aggregatedSummary.l3` and any `l3Index` typed locals/params to `tierIndex`. `cluster.BestMatch` is no longer used by the L3 path (it may still be used elsewhere — leave it).

- [ ] **Step 4:** `targ test` — new test passes; this should make the `cli` package compile again *except* for remaining test fixtures (Task 1.12). Commit: `git commit -am "feat(cli): query scores by max(situation,body); winning vector is the cluster coordinate"`

### Task 1.10: `loadCompatibleSidecars` counts schema-version mismatches

**Files:** Modify `internal/cli/query.go:960-963`; Test `internal/cli/query_test.go`

- [ ] **Step 1: Write the failing test** — a vault of only old-schema sidecars returns `errQueryNoEmbeddings` (not a silent empty), proving they were counted/surfaced:

```go
func TestQuery_AllOldSchemaSidecars_SurfacesGuidance(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()
	writeFile(t, memFS, filepath.Join(vault, "Permanent/1.fact.md"), "---\ntype: fact\nsituation: x\n---\n\nb\n")
	writeFile(t, memFS, filepath.Join(vault, "Permanent/1.fact.vec.json"),
		`{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:x"}`)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}

	var out bytes.Buffer
	err := cli.RunQuery(context.Background(), cli.QueryArgs{Phrases: []string{"x"}, VaultPath: vault}, deps, &out)
	g.Expect(err).To(MatchError(cli.ErrQueryNoEmbeddings)) // export the sentinel via export_test.go if not already
}
```

(If `errQueryNoEmbeddings` is unexported and untestable from `cli_test`, add `var ErrQueryNoEmbeddings = errQueryNoEmbeddings` to `export_test.go`.)

- [ ] **Step 2: RED.** Today old-schema sidecars are silently dropped → `hits==0` but the test can't distinguish; once counted, the `len(notes)>0 && len(hits)==0` guard fires `errQueryNoEmbeddings`. (Actually the guard already fires; the *value* of this task is the WARN path — assert via a captured `LogWarning` instead if the guard already passes. Adjust the test to capture `deps.LogWarning` calls and assert a "schema" advisory was emitted.)

- [ ] **Step 3: Implement.** In `loadCompatibleSidecars`, replace the silent `continue`:

```go
		sidecar, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			if errors.Is(parseErr, embed.ErrSchemaVersion) {
				mismatchedCount++
				mismatchedIDs["(old sidecar schema; run `engram embed apply --force`)"] = struct{}{}
			}
			continue
		}
```

- [ ] **Step 4:** `targ test` passes. Commit: `git commit -am "feat(cli): query surfaces old-schema sidecars instead of dropping them silently"`

### Task 1.11: `RunCheck` gains the `S1` sidecar invariant

The spec's "re-embed passes `engram check`" gate is **vacuous today** — `RunCheck` never reads sidecars. Add an invariant that FAILs when any note's sidecar is schema-incompatible or malformed.

**Files:** Modify `internal/cli/check.go`; Test `internal/cli/check_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestRunCheck_FailsOnOldSchemaSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Build a vault: one note with an OLD-schema sidecar.
	// (Use the existing check_test fixtures/fake-FS for Scan + ReadNote + ReadSidecar.)
	deps := cli.CheckDeps{
		Scan:        func(string) ([]vaultgraph.Note, error) { return []vaultgraph.Note{{Basename: "1.fact"}}, nil },
		ReadNote:    func(string) ([]byte, error) { return []byte("---\ntype: fact\nsituation: x\n---\n\nb\n"), nil },
		ReadSidecar: func(string) ([]byte, error) {
			return []byte(`{"embedding_model_id":"m@4","dims":4,"vector":[1,0,0,0],"content_hash":"sha256:x"}`), nil
		},
	}

	var out bytes.Buffer
	err := cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)
	g.Expect(err).To(MatchError(cli.ErrCheckFailed)) // export errCheckFailed via export_test.go
	g.Expect(out.String()).To(ContainSubstring("FAIL  S1"))
}

func TestRunCheck_PassesOnCurrentSidecar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	good := embed.MarshalSidecar(embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@4", Dims: 4,
		SituationVector: []float32{1, 0, 0, 0}, BodyVector: []float32{1, 0, 0, 0}, ContentHash: "sha256:x",
	})
	deps := cli.CheckDeps{
		Scan:        func(string) ([]vaultgraph.Note, error) { return []vaultgraph.Note{{Basename: "1.fact"}}, nil },
		ReadNote:    func(string) ([]byte, error) { return []byte("---\ntype: fact\nsituation: x\n---\n\nb\n"), nil },
		ReadSidecar: func(string) ([]byte, error) { return good, nil },
	}
	var out bytes.Buffer
	g.Expect(cli.RunCheck(context.Background(), cli.CheckArgs{VaultPath: "v"}, deps, &out)).NotTo(HaveOccurred())
	g.Expect(out.String()).To(ContainSubstring("PASS  S1"))
}
```

- [ ] **Step 2: RED.** `RunCheck` has no `ReadSidecar` and no `S1` check.

- [ ] **Step 3: Implement.** Add `ReadSidecar` to `CheckDeps`:

```go
type CheckDeps struct {
	Scan        func(vault string) ([]vaultgraph.Note, error)
	ReadNote    func(path string) ([]byte, error)
	ReadSidecar func(path string) ([]byte, error)
}
```

Wire it in `RunCheck` after the situation check:

```go
	if deps.ReadSidecar != nil {
		failed = checkSidecars(notes, deps.ReadSidecar, args.VaultPath, stdout) || failed
	}
```

Add the invariant (MOC notes have no sidecar — skip; a missing sidecar is `WARN`, not `FAIL`; an old/malformed sidecar is `FAIL`):

```go
// checkSidecars verifies S1: every situation-bearing note's sidecar parses
// under the current schema. An old-schema or malformed sidecar FAILs (a
// re-embed is required); a missing sidecar WARNs (embed status covers it).
// Returns true if the FAIL-class invariant is violated.
func checkSidecars(
	notes []vaultgraph.Note,
	readSidecar func(path string) ([]byte, error),
	vault string,
	stdout io.Writer,
) bool {
	stale := make([]string, 0)
	missing := 0

	for _, note := range notes {
		if note.IsMOC {
			continue
		}
		notePath := filepath.Join(permanentDir, note.Basename+".md")
		scBytes, err := readSidecar(filepath.Join(vault, embed.SidecarPath(notePath)))
		if err != nil {
			missing++
			continue
		}
		if _, parseErr := embed.UnmarshalSidecar(scBytes); parseErr != nil {
			stale = append(stale, note.Basename)
		}
	}

	if missing > 0 {
		_, _ = fmt.Fprintf(stdout, "WARN  S1 sidecar-schema: %d note(s) missing a sidecar (run `engram embed apply`)\n", missing)
	}
	if len(stale) > 0 {
		_, _ = fmt.Fprintf(stdout, "FAIL  S1 sidecar-schema: %d sidecar(s) on an old/invalid schema (run `engram embed apply --force`)\n", len(stale))
		printNoteExamples(stdout, stale)
		return true
	}

	_, _ = fmt.Fprintln(stdout, "PASS  S1 sidecar-schema: every sidecar parses under the current schema")
	return false
}
```

Add `"github.com/toejough/engram/internal/embed"` to `check.go` imports. Wire `ReadSidecar` in `newOsCheckDeps` (`ReadSidecar: fsys.ReadFile`). Add `var ErrCheckFailed = errCheckFailed` to `export_test.go` if the sentinel isn't already exported.

- [ ] **Step 4:** `targ test` passes. Commit: `git commit -am "feat(cli): engram check S1 sidecar-schema invariant"`

### Task 1.12: update remaining inline-`Sidecar` test fixtures

Grep finds every remaining single-`Vector` literal that won't compile. **Find them all first, fix in one pass** (don't whack-a-mole):

- [ ] **Step 1:** `grep -rn "Vector:" internal/cli/*_test.go internal/embed/*_test.go` and `grep -rn '"vector"' internal/cli/*_test.go`. Expected sites (from recon): `query_test.go:458,687,916`, `query_pipeline_test.go:596` (the `plantWithFixedVector` helper), `cli_test.go:274-286` (inline anon struct with `json:"vector"`), `os_adapters_test.go:32-80`, `query_integration_test.go:187-197`, `learn_test.go:89-94`, `resituate_test.go:467-495`.
- [ ] **Step 2:** Update `plantWithFixedVector` to write a dual-vector sidecar (`SituationVector` = `BodyVector` = the given vector, `SchemaVersion` set) — this fixes most callers at once. Update each remaining inline `embed.Sidecar{...}` and the `cli_test.go` anon struct (`json:"vector"` → `json:"situation_vector"` + `json:"body_vector"` + `json:"schema_version"`).
- [ ] **Step 3:** `targ test` — **full suite compiles and is green** for the first time since Task 1.1.
- [ ] **Step 4:** `targ check-full` — fix any lint (unused `modelID`/`dims` locals, line length). Commit: `git commit -am "test(cli): migrate sidecar fixtures to dual-vector shape"`

### Task 1.13: migration + retrieval-no-regression gate (Phase 1 checkpoint)

- [ ] **Step 1:** `targ build`.
- [ ] **Step 2:** On a **copy** of a real vault (never the live one), run `engram embed apply --force` and confirm every note re-embeds (no errors). `engram embed status` shows 0 incompatible/stale.
- [ ] **Step 3:** `engram check` on the re-embedded copy → all PASS including `S1`. On a *pre-migration* copy (old sidecars) → `S1` FAILs (proves the gate is real, not vacuous).
- [ ] **Step 4:** Retrieval spot-check: run 3–5 representative `engram query --phrase "..."` calls against the re-embedded vault and against a single-vector baseline (git-stash the change, re-embed, query, compare). Confirm the dual-vector `max(situation,body)` ranking surfaces a superset (no dropped expected hits). Record the comparison in the commit message.
- [ ] **Step 5:** Commit: `git commit -am "chore: Phase 1 gate — dual-vector re-embed passes check + no retrieval regression"`. **Phase 1 complete.**

---

# Phase 2 — Binary `--synthesize-l2` query mode

**Gate (Task 2.6):** Go tests green; `nearest_l2` verified on a seeded vault; raw cosine emitted with no band decision.

### Task 2.1: `--synthesize-l2` flag + RunQuery dispatch

**Files:** Modify `internal/cli/query.go:23-30` (`QueryArgs`), `:75` (dispatch); Test `internal/cli/query_synthesis_test.go` (new cases) + `internal/cli/targets_test.go` (flag parses)

- [ ] **Step 1: Write the failing test** (flag parses + mutual-exclusion with `--synthesis`):

```go
func TestQuery_SynthesizeL2_FlagAndSynthesisAreMutuallyExclusive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	deps := newQueryDeps(newInMemoryFS())
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: []float32{1, 0, 0, 0}}
	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"x"}, VaultPath: t.TempDir(), Synthesis: true, SynthesizeL2: true},
		deps, &out)
	g.Expect(err).To(MatchError(cli.ErrQueryModeConflict))
}
```

- [ ] **Step 2: RED.** `SynthesizeL2` undefined.

- [ ] **Step 3: Implement.** Add the flag to `QueryArgs`:

```go
	SynthesizeL2 bool `targ:"flag,name=synthesize-l2,desc=union matched L1+L2 notes, cluster once, emit nearest_l2 per cluster for lazy L2 synthesis"` //nolint:lll
```

Add the sentinel `var errQueryModeConflict = errors.New("query: --synthesis and --synthesize-l2 are mutually exclusive")` and dispatch (before the `args.Synthesis` branch at line 75):

```go
	if args.Synthesis && args.SynthesizeL2 {
		return errQueryModeConflict
	}
	if args.SynthesizeL2 {
		return runSynthesizeL2Query(ctx, args, notes, hits, limit, deps, stdout)
	}
	if args.Synthesis {
		return runSynthesisQuery(ctx, args, notes, hits, limit, deps, stdout)
	}
```

Export `ErrQueryModeConflict` via `export_test.go`.

- [ ] **Step 4:** `targ test` (the new test fails only because `runSynthesizeL2Query` is undefined — implement in 2.5; for now stub it `return nil` to compile, OR sequence 2.5 before running). Commit after 2.5. *(Order note: implement 2.2–2.5 then run.)*

### Task 2.2: `queryNearestL2` + `queryCluster.NearestL2` + test struct

**Files:** Modify `internal/cli/query.go:228-235,263-267`; `internal/cli/query_pipeline_test.go:142-183` (`queryParsed`)

- [ ] **Step 1:** Add the wire type and field:

```go
type queryNearestL2 struct {
	Path   string  `yaml:"path"`
	Cosine float32 `yaml:"cosine"`
}
```

In `queryCluster`, add (after `NearestL3`):

```go
	NearestL2 *queryNearestL2 `yaml:"nearest_l2,omitempty"`
```

In the test `queryParsed` cluster struct, add the mirror field:

```go
	NearestL2 *struct {
		Path   string  `yaml:"path"`
		Cosine float32 `yaml:"cosine"`
	} `yaml:"nearest_l2"`
```

- [ ] **Step 2:** No standalone test — exercised by 2.6. Continue.

### Task 2.3: `aggregatedSummary.l2` + `renderClusters` gains `l2` + `nearestL2ForTier`

**Files:** Modify `internal/cli/query.go:150-163` (`aggregatedSummary`), `:1405-1437` (`renderClusters`), and `renderQueryPayload` (caller)

- [ ] **Step 1:** Add `l2 tierIndex` to `aggregatedSummary`. Change `renderClusters` signature and body:

```go
func renderClusters(phraseClusters []phrasedCluster, l3Notes, l2Notes tierIndex, tiers []string) []queryCluster {
	// ... unchanged loop ...
			out = append(out, queryCluster{
				ID:         clusterID,
				Phrase:     pc.phrase,
				Size:       len(members),
				Silhouette: pc.report.silhouettesByID[clusterID],
				Members:    members,
				NearestL3:  nearestL3ForTier(centroid, l3Notes, tiers),
				NearestL2:  nearestL2ForTier(centroid, l2Notes, tiers),
			})
	// ...
}
```

Add `nearestL2ForTier` (mirror `nearestL3ForTier` — suppress when an explicit tier set omits L2; for `--synthesize-l2` the tier set is empty/L1+L2 so it never suppresses):

```go
func nearestL2ForTier(centroid []float32, l2Notes tierIndex, tiers []string) *queryNearestL2 {
	if len(tiers) > 0 && !slices.Contains(tiers, tierL2) {
		return nil
	}
	path, cosine, found := nearestInTierIndex(centroid, l2Notes)
	if !found {
		return nil
	}
	return &queryNearestL2{Path: path, Cosine: cosine}
}
```

Update `renderQueryPayload` to pass `merged.l2` to `renderClusters` (and pass an empty `tierIndex{}` for `l2` on the non-synthesize paths so existing modes emit no `nearest_l2`).

- [ ] **Step 2:** Compiles after 2.5. Continue.

### Task 2.4: `filterHitsToTiers` (pre-clustering L1+L2 constraint)

**Files:** Modify `internal/cli/query.go` (new helper near `gatherTierIndex`); Test `internal/cli/query_synthesis_test.go`

The `--tier` flag filters *emitted items*; `--synthesize-l2` must constrain *which notes enter the cluster set* to L1+L2 (L3 must not be clustered).

- [ ] **Step 1: Write the failing test** — an L3 note in the vault must not appear in any `--synthesize-l2` cluster:

```go
func TestQuery_SynthesizeL2_ExcludesL3FromClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	vault := t.TempDir()
	memFS := newInMemoryFS()
	v := []float32{1, 0, 0, 0}
	plantDualVector(t, memFS, vault, "Permanent/1.ep.md", "---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", v, v)
	plantDualVector(t, memFS, vault, "Permanent/2.fact.md", "---\ntype: fact\ntier: L2\nsituation: alpha\n---\n\nb\n", v, v)
	plantDualVector(t, memFS, vault, "Permanent/3.adr.md", "---\ntype: fact\ntier: L3\nsituation: alpha\n---\n\nb\n", v, v)

	deps := newQueryDeps(memFS)
	deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: v}
	var out bytes.Buffer
	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"alpha"}, VaultPath: vault, SynthesizeL2: true}, deps, &out)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	var parsed queryParsed
	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())
	for _, c := range parsed.Clusters {
		for _, m := range c.Members {
			g.Expect(m.Path).NotTo(ContainSubstring("3.adr"), "L3 must not be clustered in synthesize-l2")
		}
	}
}
```

- [ ] **Step 2: RED.** No L1+L2 constraint yet.

- [ ] **Step 3: Implement.**

```go
// filterHitsToTiers keeps only the hits whose note is in the given tier
// set, by reading each note's frontmatter tier. Used by --synthesize-l2 to
// constrain the CLUSTERED set to L1+L2 (distinct from --tier, which filters
// emitted items post-clustering).
func filterHitsToTiers(hits []compatibleSidecar, vault string, read func(string) ([]byte, error), tiers []string) []compatibleSidecar {
	kept := make([]compatibleSidecar, 0, len(hits))
	for _, hit := range hits {
		body, readErr := read(filepath.Join(vault, pathOf(hit.note.Basename, hit.note.IsMOC)))
		if readErr != nil {
			continue
		}
		if itemMatchesTier(resolvedItem{content: stripWikilinks(string(body))}, tiers) {
			kept = append(kept, hit)
		}
	}
	return kept
}
```

- [ ] **Step 4:** Passes after 2.5.

### Task 2.5: `runSynthesizeL2Query`

**Files:** Modify `internal/cli/query.go` (new function, mirror `runSynthesisQuery:1580-1615`)

- [ ] **Step 1:** Implement (filter hits to L1+L2 *before* union, gather L2 index, render):

```go
func runSynthesizeL2Query(
	ctx context.Context,
	args QueryArgs,
	notes []vaultgraph.Note,
	hits []compatibleSidecar,
	limit int,
	deps QueryDeps,
	stdout io.Writer,
) error {
	l1l2Hits := filterHitsToTiers(hits, args.VaultPath, deps.Read, []string{tierL1, tierL2})

	union, err := unionDirectHits(ctx, args.Phrases, l1l2Hits, args.VaultPath, limit, deps)
	if err != nil {
		return err
	}

	subgraph := buildUnionSubgraph(union, l1l2Hits)
	report := clusterUnionForSynthesis(subgraph, strings.Join(args.Phrases, "\n"))

	resolved := mergeProvenances(union, expandedSubgraph{}, clusterReport{}, hubReport{})
	resolved = applyProjectFilter(resolved, args.Project)

	merged := aggregatedSummary{
		phrases:        args.Phrases,
		resolvedItems:  resolved,
		phraseClusters: []phrasedCluster{{phrase: synthesisClusterPhrase, report: report, subgraph: subgraph}},
		l3:             tierIndex{}, // L3 not emitted in this mode
		l2:             gatherTierIndex(hits, args.VaultPath, deps.Read, tierL2),
		outgoing:       outgoingByBasename(notes),
		tiers:          nil,
		totalNotes:     len(notes),
		withEmbeddings: len(hits),
		limit:          limit,
		subgraphSize:   len(subgraph.members),
	}

	return renderQueryPayload(stdout, merged)
}
```

Notes: the L2 index is gathered from the **full** `hits` (every L2 in the vault is a candidate nearest, not just the matched ones); the *clustered* set is the matched L1+L2 (`l1l2Hits`). `nearestL2ForTier` is called with `tiers: nil` → never suppressed.

- [ ] **Step 2:** `targ test` — Tasks 2.1/2.4 tests now compile and pass.

### Task 2.6: `nearest_l2` correctness + raw-cosine + property test (Phase 2 checkpoint)

**Files:** Test `internal/cli/query_synthesis_test.go`

- [ ] **Step 1: Write the tests:**

```go
func TestQuery_SynthesizeL2_NearestL2PresentWhenL2Exists(t *testing.T) { /* one L2 in vault → cluster.NearestL2 != nil */ }

func TestQuery_SynthesizeL2_NoL2_NearestL2Nil(t *testing.T) { /* only L1 notes → NearestL2 == nil */ }

func TestQuery_SynthesizeL2_NearDuplicateL2_CosineAtLeast095(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	// Cluster of L1+L2 all sharing a vector identical (or ~) to an existing L2 note
	// → nearest_l2.cosine >= 0.95 (raw value, no band applied by the binary).
	// ... plant identical vectors; assert parsed.Clusters[0].NearestL2.Cosine >= 0.95
}

func TestQuery_SynthesizeL2_EmitsRawCosineNoBand(t *testing.T) {
	// A cluster whose centroid is FAR from any L2 (cosine ~0.3) still emits
	// nearest_l2 with that raw low cosine (binary applies no <0.80 cutoff).
}
```

Add a rapid property test: for any vault containing a near-duplicate L2 of the cluster centroid, `nearest_l2.cosine >= 0.95`.

- [ ] **Step 2: RED → Step 3: (already implemented) → Step 4:** `targ test` + `targ check-full` green. Commit: `git commit -am "feat(cli): engram query --synthesize-l2 emits nearest_l2 per cluster"`. **Phase 2 complete.**

---

# Phase 3 — `/recall` three-band blocking L2 writes

**GATE: superpowers:writing-skills is MANDATORY.** Baseline behavior test (RED) → edit SKILL.md (GREEN) → pressure test → `engram update`. No editing the SKILL.md before capturing RED. The Iron Law applies to skill edits.

### Task 3.1: Baseline RED behavior test

**Files:** Create `skills/recall/tests/baseline-three-band-writes.md`, `skills/recall/tests/baseline-three-band-RED-results.md`

- [ ] **Step 1:** Write the baseline scenario file following the existing format (`skills/recall/tests/baseline-multi-query.md`). Scenario: give a subagent a synthetic `engram query --synthesize-l2` YAML payload with **three clusters**, one in each band — cluster A `nearest_l2.cosine = 0.97`, cluster B `0.86`, cluster C `0.42` — each with ≥3 members (mixed L1+L2) and member notes on disk with `created` frontmatter dates (one cluster has diverging members with different dates, to probe recency-bias). Ask the subagent to read the CURRENT `skills/recall/SKILL.md` and follow it.

  **GREEN behaviors to specify (the target after Task 3.2):**
  1. Cluster A (≥0.95) → **no** `engram learn` call.
  2. Cluster B (0.80–0.95) → `engram learn fact|feedback` with `--target <luhmann-id parsed from nearest_l2.path>` (update), **no `--tier`**.
  3. Cluster C (<0.80) → `engram learn fact|feedback --position top` (create), `--relation` to each cluster member, **no `--tier`**.
  4. Writes are **blocking**: the agent waits for the writes and then states it will *apply* the new L2s to the current task (not "dispatched fire-and-forget").
  5. The synthesis prompt/reasoning **prefers the more-recently-created member** where members diverge.

  **RED expectation:** the current skill (Step 3a) dispatches *fire-and-forget* synthesis subagents, applies no cosine bands (it has none — the 0.9 cut lives only in learn §6b), and treats recall as a non-writer it doesn't wait on.

- [ ] **Step 2:** Dispatch a subagent against the CURRENT skill (use the Agent tool / `subagent` per environment). Capture verbatim output to `baseline-three-band-RED-results.md`. Confirm it does NOT do the three-band blocking behavior (documents the gap).

- [ ] **Step 3:** Commit: `git commit -am "test(recall): baseline RED for three-band L2 writes"`

### Task 3.2: Edit `skills/recall/SKILL.md` — three-band blocking writes

**Files:** Modify `skills/recall/SKILL.md` (Step 3 ~78-130; Step 3a ~132-174; Step 4b line ~222; "not for" line ~272; red-flag rows ~240-244)

- [ ] **Step 1:** Make these edits (the RED test now defines GREEN):

  **(a) Step 3 header + query call** — change `engram query --tier L2` to `engram query --synthesize-l2`:
  > Issue a single `engram query --synthesize-l2` call passing each Step 1 phrase as `--phrase`. The binary unions the matched **L1+L2** neighborhood, clusters once with matched-vector coordinates, and returns each cluster's `nearest_l2 {path, cosine}` — the raw cosine of the cluster centroid to the closest existing L2. **The binary applies no decision; the bands below are yours.**

  **(b) Replace Step 3a's dispatch gate with the three-band rule.** Keep the parent-reads-only-the-rep / subagent-reads-members split, but make it a **ternary on `nearest_l2.cosine`** and **blocking**:
  > For each cluster (size ≥ 3), read `nearest_l2.cosine`:
  > - **≥ 0.95 → no-op.** An existing L2 already represents this cluster. Skip.
  > - **0.80 ≤ cosine < 0.95 → update.** Dispatch a synthesis subagent to fold the cluster's members into the nearest L2: `engram learn fact|feedback --target <luhmann-id from nearest_l2.path> --position continuation` (**no `--tier`** — absence = L2). Prefer the more-recently-created member's content where members diverge (read `created` from each member's frontmatter during the member-read step).
  > - **< 0.80 → create.** Dispatch a synthesis subagent to write a new L2 (Fact **and/or** Feedback per §4 of `/learn`) synthesizing the cluster, `--relation`-linked to every member (L1s and L2s), `--position top`, **no `--tier`**.
  >
  > **These writes are BLOCKING.** Unlike the L3 synthesis subagents (fire-and-forget, for *future* recalls), the recalling agent must **use** the freshly-minted L2s for its *current* task. Dispatch the per-cluster subagents, **wait for them to finish**, then read the resulting notes and apply them. The thresholds (0.95 / 0.80) are defaults; the harness may override them by naming different values in the recall instruction.

  **(c) Line ~222 (4b "No synthesis-write output"):** change to acknowledge blocking — the agent *does* apply the new L2s and may state which conventions it crystallized and is now applying (still no wikilinks, still no raw YAML).

  **(d) Line ~272 ("not for"):** widen the carve-out — recall is now a deliberate sometimes-writer; the L2 synthesis blocks and its output is used in-task.

  **(e) Red-flag rows ~240-244:** keep "dispatch, don't inline-synthesize" and "parent reads only the rep"; **add** a row: "You fired-and-forgot the L2 writes instead of waiting → the L2 synthesis is blocking; wait and apply the results." **Remove/replace** any row asserting recall never writes.

- [ ] **Step 2:** No code test; the GREEN check is Task 3.3.

### Task 3.3: GREEN behavior test

- [ ] **Step 1:** Re-dispatch the Task 3.1 subagent against the EDITED skill. Capture to `skills/recall/tests/baseline-three-band-GREEN-results.md`.
- [ ] **Step 2:** Verify all 5 GREEN behaviors. If any fails, refine the SKILL.md wording (close the loophole) and re-run — do not proceed until green.
- [ ] **Step 3:** Commit: `git commit -am "feat(recall): three-band blocking L2 synthesis writes (GREEN)"`

### Task 3.4: Pressure test

- [ ] **Step 1:** Create `skills/recall/tests/pressure-three-band.md` — adversarial cases: (a) a cluster with `nearest_l2.cosine` exactly 0.95 and exactly 0.80 (boundary: ≥0.95 no-op, ≥0.80 update); (b) time pressure ("the user is waiting") to test that blocking is honored; (c) a cluster with members diverging across `created` dates (recency-bias must pick the newer); (d) the temptation to add `--tier L3` (must not).
- [ ] **Step 2:** Run; capture results; close any loopholes found.
- [ ] **Step 3:** Commit: `git commit -am "test(recall): pressure tests for three-band writes"`

### Task 3.5: sync to live harness (Phase 3 checkpoint)

- [ ] **Step 1:** `engram update` (clears + re-copies `skills/` to `~/.claude/skills/` and `~/.config/opencode/skills/`). Confirm the edited `recall/SKILL.md` landed in `~/.claude/skills/recall/SKILL.md`.
- [ ] **Step 2:** Commit any remaining test artifacts. **Phase 3 complete.**

---

# Phase 4 — Eager-vs-lazy experiment (`l2.lazy` regime)

**GATE (Task 4.6):** zero-cost stub validation passes before any paid run. **The paid run (Task 4.7) is user-authorized only** — do not launch the matrix without explicit go-ahead (it spends real tokens).

### Task 4.1: add the `l2.lazy` regime + aggregate aliases

**Files:** Modify `dev/eval/cumulative/harness.py:47-55` (`REGIMES`), `dev/eval/cumulative/aggregate.py:98-99` (`WRITE_TIER`), `:407-408` (`write_of`)

- [ ] **Step 1: Write the failing check** — a stub matrix run must include `l2.lazy` ops and aggregate without `KeyError`. Add a small assertion to the stub-validation script (or run `python3 matrix.py --models sonnet --trials 1 --stub good` and grep the op list for `l2.lazy`).

- [ ] **Step 2: RED.** `l2.lazy` not in `REGIMES`.

- [ ] **Step 3: Implement.** Add to `REGIMES` (after `l2.l2`):

```python
    "l2.lazy":   {"write": "L1",  "read_mode": "synthesize_l2", "read_tiers": []},
```

Add `l2.lazy` to `aggregate.py`'s `WRITE_TIER` dict (→ `"L1"`) and `write_of` (→ `"L1"`), wherever the other regimes are aliased. (Arm A is the existing `l2.l1l2`; both arms end with L1+L2 readable — A writes L2 at learn, B crystallizes at recall.)

- [ ] **Step 4:** `python3 matrix.py --models sonnet --trials 1 --stub good` lists `l2.lazy` ops and `aggregate.py` runs clean. Commit: `git commit -am "feat(eval): add l2.lazy regime (lazy arm B)"`

### Task 4.2: `build_prompt` synthesize_l2 branch (recall + blocking write instruction)

**Files:** Modify `dev/eval/cumulative/harness.py:111-144` (`build_prompt`)

- [ ] **Step 1: Write the failing test** — `build_prompt("links", ..., "synthesize_l2", [])` must contain `engram query --synthesize-l2` and a blocking three-band write instruction (assert via a small unit check or a stub-run transcript grep).

- [ ] **Step 2: RED.** The `else: tier` branch would emit an invalid `engram query  --phrase ...` (empty `--tier`).

- [ ] **Step 3: Implement.** Add a branch before the `else: # tier`:

```python
    elif read_mode == "synthesize_l2":
        recall = (
            "\nBefore writing any code, consult your memory. Run exactly this, read every surfaced "
            "note, and APPLY every convention and decision it surfaces:\n"
            f"  engram query --synthesize-l2 {phrases}\n"
            "This is LAZY L2 synthesis. Each cluster in the payload carries `nearest_l2: {path, cosine}` "
            "— the closest existing L2 to that cluster. For each cluster (size >= 3), apply the three "
            "bands and WAIT for any writes before continuing (the new L2s are for THIS build):\n"
            "  - cosine >= 0.95 -> do nothing (an L2 already covers it).\n"
            "  - 0.80 <= cosine < 0.95 -> `engram learn fact|feedback` updating the nearest L2 "
            "(--target <luhmann-id from nearest_l2.path> --position continuation; NO --tier).\n"
            "  - cosine < 0.80 -> `engram learn fact|feedback` creating a new L2 synthesizing the "
            "cluster (--position top, --relation to each member; NO --tier).\n"
            "Prefer the more-recently-created member where members diverge. Then APPLY the surfaced "
            "and freshly-written L2 conventions to your build.\n"
        )
```

- [ ] **Step 4:** Stub run shows the new prompt branch. Commit: `git commit -am "feat(eval): build_prompt synthesize_l2 recall + blocking three-band writes"`

### Task 4.3: `run_learn` persist-forward for `l2.lazy`

**Files:** Modify `dev/eval/cumulative/harness.py:758-768` (the learn-stage seeding)

This is the **amortization fix**: for `l2.lazy`, seed the learn stage from the post-recall **build vault** (which holds `vault_in` + crystallized L2s), so they carry into `vault_out`.

- [ ] **Step 1: Write the failing test** — after an `l2.lazy` learn whose build vault contains a crystallized L2, `vault_out` must contain that L2. (Stub: pre-seed `workdir + ".buildvault"` with an L2 note + sidecar, run `run_learn` with `regime="l2.lazy"`, assert the L2 is in `vault_out`.)

- [ ] **Step 2: RED.** `run_learn` seeds from `args.vault_in` (line 764), discarding the build vault's L2s.

- [ ] **Step 3: Implement.** Replace the seed-source selection (lines 762-767) to prefer the build vault for the lazy regime:

```python
    learn_vault = args.vault_out + ".staging"
    shutil.rmtree(learn_vault, ignore_errors=True)

    # Lazy arm: recall crystallized L2s into the build vault during the build
    # session. Seed the learn stage from THAT (it already contains vault_in +
    # the new L2s) so crystallized L2s persist forward across the app chain.
    build_vault = args.workdir + ".buildvault"
    lazy_seed = args.regime == "l2.lazy" and os.path.isdir(build_vault)
    seed_src = build_vault if lazy_seed else args.vault_in

    if seed_src != "none" and os.path.isdir(seed_src):
        shutil.copytree(seed_src, learn_vault)
    else:
        os.makedirs(os.path.join(learn_vault, "Permanent"), exist_ok=True)
```

(`prune_to_ceiling(learn_vault, "L1")` still runs — but it drops notes *above* the L1 ceiling, which would delete the crystallized L2s! **Guard it:** for `l2.lazy`, the crystallized L2s are the experiment's *output* and must survive. Change the prune call for this regime to a no-op or a ceiling of L2.) Add near line 796:

```python
    prune_tier = "L2" if args.regime == "l2.lazy" else args.write_tier
    pruned = prune_to_ceiling(learn_vault, prune_tier)
```

- [ ] **Step 4:** Test passes (crystallized L2 in `vault_out`). Commit: `git commit -am "feat(eval): l2.lazy persists crystallized L2s forward (vault_out = build-vault ∪ L1)"`

### Task 4.4: `run_build` post-op vault metrics (#L2, composition)

**Files:** Modify `dev/eval/cumulative/harness.py` (`run_build`, after the build session)

- [ ] **Step 1: Write the failing test** — a build whose build vault gained 2 L2 notes (one linking to another L2) reports `l2_generated: 2`, `l2_composed: 1` in the build JSON.

- [ ] **Step 2: RED.** `run_build` scores Go source only; no vault inspection.

- [ ] **Step 3: Implement.** After the build session, inspect `workdir + ".buildvault"`: count L2 notes whose `created` date is the build date (newly crystallized vs. seeded — or diff against the seed `vault_in` note set), and count how many of those L2s have a `Related to:` wikilink pointing at another L2. Add to the build result dict:

```python
    bv = args.workdir + ".buildvault"
    seeded = set(os.path.basename(p) for p in glob_notes(args.vault_in)) if args.vault_in != "none" else set()
    new_l2 = [p for p in glob_notes(bv) if note_tier(p) == "L2" and os.path.basename(p) not in seeded]
    l2_composed = sum(1 for p in new_l2 if _links_to_l2(p, bv))  # helper: any Related-to target is an L2
    out["l2_generated"] = len(new_l2)
    out["l2_composed"] = l2_composed
    out["vault_notes_total"] = len(glob_notes(bv))
```

(Add a small `_links_to_l2` helper reading the note's wikilink targets and checking each target note's tier. Only meaningful for `l2.lazy`; arm A reports 0 here since it does not write during build — that's the correct contrast.)

- [ ] **Step 4:** Test passes. Commit: `git commit -am "feat(eval): run_build reports #L2-generated, composition, vault growth"`

### Task 4.5: stub support for `l2.lazy`

**Files:** Modify `dev/eval/cumulative/harness.py` (`_stub_build` / `_deterministic_learn` / stub recall)

- [ ] **Step 1:** For `--stub`, the `l2.lazy` build must exercise a fake synthesize-l2 recall that writes ≥1 L2 into the build vault (so persist-forward and the metrics path are validated without an LLM). The deterministic learn for `l2.lazy` writes L1 only; the persist-forward (Task 4.3) then carries the stub-crystallized L2 into `vault_out`.
- [ ] **Step 2:** Run `python3 matrix.py --models sonnet --trials 1 --stub good` — `l2.lazy` cells produce a build vault with a stub L2, `vault_out` contains it, metrics populate. Commit: `git commit -am "test(eval): stub path exercises l2.lazy crystallize + persist-forward"`

### Task 4.6: zero-cost stub validation (Phase 4 gate)

- [ ] **Step 1:** `python3 matrix.py --models sonnet --trials 1 --stub good` end-to-end. Verify: all `l2.lazy` ops run; `op_done` resume logic treats them correctly; `aggregate.py` produces the A-vs-B comparison table without `KeyError`; persist-forward carries the stub L2 across app2→app3; cost audit returns zeroes (stub).
- [ ] **Step 2:** Commit: `git commit -am "chore(eval): l2.lazy stub validation green (Phase 4 gate)"`

### Task 4.7: focused A/B run + aggregate (USER-AUTHORIZED, paid)

- [ ] **Step 1:** **Confirm with the user before launching** — this spends real tokens. Scope: `--models sonnet --trials 1,2,3,4,5`, regimes restricted to arm A (`l2.l1l2`) and arm B (`l2.lazy`), the full notes→links→feeds chain.
- [ ] **Step 2:** Launch with the standard resilience/resume wrapper (interruptions are normal — just resume; the matrix is resumable via `op_done`).
- [ ] **Step 3:** `python3 aggregate.py` over the run dir → A-vs-B table on: net chain cost, learn cost/tokens, #L2 generated, say-once convention restatements, completion, vault growth, L2 composition. `python3 compare.py <this-run> <eager-baseline>` for the differential.
- [ ] **Step 4:** Write `dev/eval/cumulative/results-lazy-l2.md` recording the result and the recommendation. Commit. **Phase 4 complete.**

---

## Self-review (run against the spec)

**Spec coverage:** §3.1 dual-vector sidecars → Phase 1 (Tasks 1.1–1.13, incl. the `S1` check gate and the no-regression gate). §3.2 binary `--synthesize-l2` + `nearest_l2`, no band decision → Phase 2 (2.1–2.6). §3.3 `/recall` three-band blocking writes, recency-bias, no `--tier`, thresholds-as-defaults → Phase 3 (writing-skills TDD, 3.1–3.5). §3.4 experiment: distinct `l2.lazy` regime, persist-forward, new metrics, recall-cost-folds-into-build (noted, not instrumented) → Phase 4 (4.1–4.7). §2 mechanism (match by max, winning-vector coordinate, gate by max, three bands, recency-bias) → Tasks 1.9 + 2.3/2.5 + 3.2. Amendments (ContentHash sit+body; schema-check-before-dims) → Tasks 1.1/1.2.

**Type consistency:** `Sidecar` shape, `BuildSidecar`, `SituationText`/`BodyText`, `bestVector`, `tierIndex`/`gatherTierIndex`/`nearestInTierIndex`, `queryNearestL2`, `runSynthesizeL2Query`, `nearestL2ForTier`, `filterHitsToTiers`, `checkSidecars`/`ReadSidecar`, `l2.lazy` regime — all defined once in Shared Contracts and referenced by those exact names throughout.

**Known follow-ups (out of scope, file as issues if pursued):** isolated per-turn recall-cost instrumentation; L3 lazy synthesis (could later consume lazy-L2); retrofitting non-eval recall callers; threshold *sweep* injection mechanism (v1 uses skill defaults + prompt override).
